// GateKey Gateway Agent
// This agent runs alongside OpenVPN and handles hook callbacks and firewall management.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gatekey-project/gatekey/internal/firewall"
	"github.com/gatekey-project/gatekey/internal/openvpn"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	configPath       string
	logger           *zap.Logger
	firewallMgr      *firewall.Manager
	connectedUsers   map[string]ConnectedClient // VPN IP -> client info
	currentConfigVer string                     // Current config version from control plane
)

const configVersionFile = "/etc/gatekey/.config_version"

// loadConfigVersion loads the persisted config version from disk
func loadConfigVersion() string {
	data, err := os.ReadFile(configVersionFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// saveConfigVersion persists the config version to disk
func saveConfigVersion(version string) error {
	return os.WriteFile(configVersionFile, []byte(version), 0644)
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "gatekey-gateway",
		Short: "GateKey Gateway Agent",
		Long: `GateKey Gateway Agent runs alongside OpenVPN and provides:
- Hook script handling for authentication and authorization
- Per-identity firewall rule management
- Connection state reporting to the control plane`,
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "/etc/gatekey/gateway.yaml", "config file path")

	// Run command - starts the gateway agent daemon
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the gateway agent",
		RunE:  runAgent,
	}

	// Hook command - handles OpenVPN hook callbacks
	hookCmd := &cobra.Command{
		Use:   "hook",
		Short: "Handle an OpenVPN hook callback",
		RunE:  handleHook,
	}
	hookCmd.Flags().String("type", "", "Hook type (auth-user-pass-verify, tls-verify, client-connect, client-disconnect)")

	rootCmd.AddCommand(runCmd, hookCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// GatewayConfig holds gateway agent configuration.
type GatewayConfig struct {
	ControlPlaneURL   string        `mapstructure:"control_plane_url"`
	Token             string        `mapstructure:"token"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	RuleRefreshInterval time.Duration `mapstructure:"rule_refresh_interval"`
	LogLevel          string        `mapstructure:"log_level"`
}

// ConnectedClient holds info about a connected VPN client.
type ConnectedClient struct {
	UserID     string
	UserEmail  string
	UserGroups []string
	VPNIP      string
	ConnectedAt time.Time
}

// ClientRulesResponse is the response from the control plane.
type ClientRulesResponse struct {
	UserID   string `json:"user_id"`
	ClientIP string `json:"client_ip"`
	Allowed  []AllowedDestination `json:"allowed"`
	Default  string `json:"default"`
}

// AllowedDestination represents an allowed destination.
type AllowedDestination struct {
	Type     string `json:"type"`
	Value    string `json:"value"`
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
}

func loadConfig() (*GatewayConfig, error) {
	v := viper.New()
	v.SetConfigFile(configPath)

	v.SetDefault("heartbeat_interval", "30s")
	v.SetDefault("rule_refresh_interval", "10s")
	v.SetDefault("log_level", "info")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	v.SetEnvPrefix("GATEX")
	v.AutomaticEnv()

	var cfg GatewayConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func initLogger(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	if level == "debug" {
		cfg = zap.NewDevelopmentConfig()
	}
	return cfg.Build()
}

func runAgent(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	logger, err = initLogger(cfg.LogLevel)
	if err != nil {
		return err
	}
	defer logger.Sync()

	logger.Info("Starting GateKey Gateway Agent",
		zap.String("control_plane", cfg.ControlPlaneURL),
	)

	// Initialize connected users map
	connectedUsers = make(map[string]ConnectedClient)

	// Initialize firewall manager
	nftBackend, err := firewall.NewNFTablesBackend(firewall.NFTablesConfig{
		TableName: "gatekey",
		ChainName: "forward",
	})
	if err != nil {
		logger.Warn("Failed to create nftables backend, firewall rules will not be enforced", zap.Error(err))
	} else {
		firewallMgr = firewall.NewManager(nftBackend)
		ctx := context.Background()
		if err := firewallMgr.Initialize(ctx); err != nil {
			logger.Warn("Failed to initialize firewall manager", zap.Error(err))
			firewallMgr = nil
		} else {
			logger.Info("Firewall manager initialized")
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start heartbeat
	go heartbeatLoop(ctx, cfg)

	// Start rule refresh loop
	go ruleRefreshLoop(ctx, cfg)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down gateway agent")

	// Cleanup firewall rules
	if firewallMgr != nil {
		if err := firewallMgr.Cleanup(context.Background()); err != nil {
			logger.Warn("Failed to cleanup firewall rules", zap.Error(err))
		}
	}

	return nil
}

func heartbeatLoop(ctx context.Context, cfg *GatewayConfig) {
	client := openvpn.NewHookClient(cfg.ControlPlaneURL, cfg.Token)
	ticker := time.NewTicker(cfg.HeartbeatInterval)
	defer ticker.Stop()

	// Load persisted config version from disk
	currentConfigVer = loadConfigVersion()
	if currentConfigVer != "" {
		logger.Info("Loaded config version from disk", zap.String("config_version", currentConfigVer))
	}

	// Get public IP on startup
	publicIP := getPublicIP()

	// Send initial heartbeat immediately
	resp, err := client.Heartbeat(publicIP, 0, isOpenVPNRunning(), currentConfigVer)
	if err != nil {
		logger.Warn("Initial heartbeat failed", zap.Error(err))
	} else {
		logger.Info("Initial heartbeat sent successfully",
			zap.String("config_version", resp.ConfigVersion))
		// If we have no config version, we need to reprovision to ensure our local files
		// match what the server expects. Don't just adopt the server's version blindly.
		if currentConfigVer == "" && resp.ConfigVersion != "" {
			logger.Info("No local config version - triggering initial provision",
				zap.String("server_version", resp.ConfigVersion))
			if err := handleReprovision(ctx, cfg, client); err != nil {
				logger.Error("Initial provision failed", zap.Error(err))
			} else {
				currentConfigVer = resp.ConfigVersion
				if err := saveConfigVersion(currentConfigVer); err != nil {
					logger.Warn("Failed to save config version", zap.Error(err))
				}
				logger.Info("Initial provision completed",
					zap.String("config_version", currentConfigVer))
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if OpenVPN is running
			openvpnRunning := isOpenVPNRunning()
			activeClients := getActiveClientCount()

			resp, err := client.Heartbeat(publicIP, activeClients, openvpnRunning, currentConfigVer)
			if err != nil {
				logger.Warn("Heartbeat failed", zap.Error(err))
				continue
			}

			// Check if we need to reprovision
			if resp.NeedsReprovision {
				logger.Info("Control plane signaled reprovision needed",
					zap.String("current_version", currentConfigVer),
					zap.String("server_version", resp.ConfigVersion))

				if err := handleReprovision(ctx, cfg, client); err != nil {
					logger.Error("Reprovision failed", zap.Error(err))
				} else {
					// Update our config version after successful reprovision
					currentConfigVer = resp.ConfigVersion
					if err := saveConfigVersion(currentConfigVer); err != nil {
						logger.Warn("Failed to save config version", zap.Error(err))
					}
					logger.Info("Reprovision completed successfully",
						zap.String("new_config_version", currentConfigVer))
				}
			}
		}
	}
}

// handleReprovision fetches new certificates and config, updates files, and restarts OpenVPN.
func handleReprovision(ctx context.Context, cfg *GatewayConfig, client *openvpn.HookClient) error {
	logger.Info("Starting reprovision...")

	// Fetch new certificates and config from control plane
	provResp, err := client.Provision()
	if err != nil {
		return fmt.Errorf("failed to provision: %w", err)
	}

	// Update certificate files
	openvpnDir := "/etc/openvpn/server"
	if err := os.WriteFile(openvpnDir+"/ca.crt", []byte(provResp.CACert), 0644); err != nil {
		return fmt.Errorf("failed to write CA cert: %w", err)
	}
	if err := os.WriteFile(openvpnDir+"/server.crt", []byte(provResp.ServerCert), 0644); err != nil {
		return fmt.Errorf("failed to write server cert: %w", err)
	}
	if err := os.WriteFile(openvpnDir+"/server.key", []byte(provResp.ServerKey), 0600); err != nil {
		return fmt.Errorf("failed to write server key: %w", err)
	}

	// Update TLS-Auth key if provided
	if provResp.TLSAuthEnabled && provResp.TLSAuthKey != "" {
		if err := os.WriteFile(openvpnDir+"/ta.key", []byte(provResp.TLSAuthKey), 0600); err != nil {
			return fmt.Errorf("failed to write TLS-Auth key: %w", err)
		}
	}

	logger.Info("Certificates updated, restarting OpenVPN...")

	// Restart OpenVPN to pick up new config
	if err := restartOpenVPN(); err != nil {
		return fmt.Errorf("failed to restart OpenVPN: %w", err)
	}

	return nil
}

// restartOpenVPN restarts the OpenVPN service.
func restartOpenVPN() error {
	// Try systemctl first (most common)
	cmd := exec.Command("systemctl", "restart", "openvpn-server@server")
	if err := cmd.Run(); err != nil {
		// Try alternative service name
		cmd = exec.Command("systemctl", "restart", "openvpn@server")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to restart OpenVPN service: %w", err)
		}
	}
	return nil
}

// getPublicIP attempts to determine the public IP address
func getPublicIP() string {
	// Try to get from environment first (set by cloud metadata)
	if ip := os.Getenv("PUBLIC_IP"); ip != "" {
		return ip
	}
	// Could also query a metadata service or external IP service
	return ""
}

// isOpenVPNRunning checks if OpenVPN process is running
func isOpenVPNRunning() bool {
	// Check if openvpn process exists by looking for pid file or process
	if _, err := os.Stat("/run/openvpn/server.pid"); err == nil {
		return true
	}
	if _, err := os.Stat("/var/run/openvpn/server.pid"); err == nil {
		return true
	}
	return false
}

// getActiveClientCount returns the number of active OpenVPN clients
func getActiveClientCount() int {
	// Could parse OpenVPN status file or management interface
	// For now return count of connected users
	return len(connectedUsers)
}

// ruleRefreshLoop periodically refreshes firewall rules for connected clients.
func ruleRefreshLoop(ctx context.Context, cfg *GatewayConfig) {
	ticker := time.NewTicker(cfg.RuleRefreshInterval)
	defer ticker.Stop()

	// Ensure clients directory exists
	os.MkdirAll(clientsDir, 0755)

	logger.Info("Started rule refresh loop", zap.Duration("interval", cfg.RuleRefreshInterval))

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Sync connected clients from files
			syncConnectedClients(cfg)

			// Refresh rules for all connected clients
			refreshAllClientRules(cfg)
		}
	}
}

// refreshAllClientRules refreshes firewall rules for all connected clients.
func refreshAllClientRules(cfg *GatewayConfig) {
	if firewallMgr == nil {
		return
	}

	for vpnIP, client := range connectedUsers {
		rules, err := fetchClientRules(cfg, client.UserID, client.UserEmail, client.UserGroups, vpnIP)
		if err != nil {
			logger.Warn("Failed to refresh rules for client",
				zap.String("vpn_ip", vpnIP),
				zap.Error(err))
			continue
		}

		// Convert and apply rules
		if err := applyFirewallRules(cfg, vpnIP, client.UserID, rules); err != nil {
			logger.Warn("Failed to apply refreshed rules",
				zap.String("vpn_ip", vpnIP),
				zap.Error(err))
		}
	}
}

// fetchClientRules fetches access rules for a client from the control plane.
func fetchClientRules(cfg *GatewayConfig, userID, userEmail string, userGroups []string, clientIP string) (*ClientRulesResponse, error) {
	reqBody := struct {
		Token      string   `json:"token"`
		UserID     string   `json:"user_id"`
		UserEmail  string   `json:"user_email"`
		UserGroups []string `json:"user_groups"`
		ClientIP   string   `json:"client_ip"`
	}{
		Token:      cfg.Token,
		UserID:     userID,
		UserEmail:  userEmail,
		UserGroups: userGroups,
		ClientIP:   clientIP,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/gateway/client-rules"
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("control plane returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result ClientRulesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// applyFirewallRules applies firewall rules for a client.
func applyFirewallRules(cfg *GatewayConfig, clientIP, userID string, rules *ClientRulesResponse) error {
	if firewallMgr == nil {
		return nil
	}

	// Convert allowed destinations to firewall rules
	var networks []net.IPNet
	var ports []firewall.PortRange

	for _, dest := range rules.Allowed {
		switch dest.Type {
		case "ip":
			// Single IP - convert to /32
			ip := net.ParseIP(dest.Value)
			if ip != nil {
				networks = append(networks, net.IPNet{
					IP:   ip,
					Mask: net.CIDRMask(32, 32),
				})
			}
		case "cidr":
			_, ipnet, err := net.ParseCIDR(dest.Value)
			if err == nil && ipnet != nil {
				networks = append(networks, *ipnet)
			}
		case "hostname", "hostname_wildcard":
			// Resolve hostname to IP
			ips, err := net.LookupIP(dest.Value)
			if err == nil {
				for _, ip := range ips {
					if ip4 := ip.To4(); ip4 != nil {
						networks = append(networks, net.IPNet{
							IP:   ip4,
							Mask: net.CIDRMask(32, 32),
						})
					}
				}
			}
		}

		// Parse port if specified
		if dest.Port != "" && dest.Port != "*" {
			protocol := firewall.ProtocolAny
			switch dest.Protocol {
			case "tcp":
				protocol = firewall.ProtocolTCP
			case "udp":
				protocol = firewall.ProtocolUDP
			}

			if strings.Contains(dest.Port, "-") {
				// Port range
				parts := strings.Split(dest.Port, "-")
				if len(parts) == 2 {
					start, _ := strconv.Atoi(parts[0])
					end, _ := strconv.Atoi(parts[1])
					ports = append(ports, firewall.PortRange{
						Protocol: protocol,
						Port:     start,
						PortEnd:  end,
					})
				}
			} else {
				port, _ := strconv.Atoi(dest.Port)
				if port > 0 {
					ports = append(ports, firewall.PortRange{
						Protocol: protocol,
						Port:     port,
					})
				}
			}
		}
	}

	// Parse client's VPN IP
	sourceIP := net.ParseIP(clientIP)
	if sourceIP == nil {
		return fmt.Errorf("invalid client IP: %s", clientIP)
	}

	// Parse user ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		uid = uuid.New() // Use a random UUID if parse fails
	}

	// Create a connection ID based on client IP
	connectionID := fmt.Sprintf("client-%s", strings.ReplaceAll(clientIP, ".", "-"))

	// Apply rules
	ctx := context.Background()
	return firewallMgr.ApplyRules(ctx, connectionID, uid, sourceIP, networks, ports)
}

// removeFirewallRules removes firewall rules for a disconnected client.
func removeFirewallRules(clientIP string) error {
	if firewallMgr == nil {
		return nil
	}

	connectionID := fmt.Sprintf("client-%s", strings.ReplaceAll(clientIP, ".", "-"))
	return firewallMgr.RemoveRules(context.Background(), connectionID)
}

func handleHook(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	hookType, _ := cmd.Flags().GetString("type")
	if hookType == "" {
		hookType = os.Getenv("script_type")
	}

	if hookType == "" {
		return fmt.Errorf("hook type not specified")
	}

	client := openvpn.NewHookClient(cfg.ControlPlaneURL, cfg.Token)
	req := openvpn.BuildHookRequest(openvpn.HookType(hookType))

	// Handle file-based credentials for auth-user-pass-verify
	if hookType == "auth-user-pass-verify" && len(args) > 0 {
		env, err := openvpn.ParseEnvFile(args[0])
		if err == nil {
			for k, v := range env {
				req.Env[k] = v
			}
			if username, ok := env["username"]; ok {
				req.Username = username
			}
		}
	}

	switch openvpn.HookType(hookType) {
	case openvpn.HookAuthUserPassVerify, openvpn.HookTLSVerify:
		resp, err := client.Verify(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Verification failed: %v\n", err)
			os.Exit(1)
		}
		if !resp.Allow {
			fmt.Fprintf(os.Stderr, "Access denied: %s\n", resp.Message)
			os.Exit(1)
		}
		fmt.Println("Access granted")
		os.Exit(0)

	case openvpn.HookClientConnect:
		resp, err := client.Connect(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Connect notification failed: %v\n", err)
			os.Exit(1)
		}

		// Write client config if provided
		if len(resp.ClientConfig) > 0 && len(args) > 0 {
			if err := openvpn.WriteClientConfig(args[0], resp.ClientConfig); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write client config: %v\n", err)
			}
		}

		// Write client info to file for daemon to pick up
		vpnIP := req.IFConfigRemote
		if vpnIP != "" {
			clientInfo := ConnectedClient{
				UserID:      req.CommonName, // Common name contains user ID from cert
				UserEmail:   req.Username,
				VPNIP:       vpnIP,
				ConnectedAt: time.Now(),
			}
			if err := writeClientFile(vpnIP, clientInfo); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to write client file: %v\n", err)
			}
		}
		os.Exit(0)

	case openvpn.HookClientDisconnect:
		if err := client.Disconnect(req); err != nil {
			fmt.Fprintf(os.Stderr, "Disconnect notification failed: %v\n", err)
			// Don't exit with error, client is already disconnecting
		}

		// Remove client info file so daemon removes firewall rules
		vpnIP := req.IFConfigRemote
		if vpnIP != "" {
			removeClientFile(vpnIP)
		}
		os.Exit(0)

	default:
		return fmt.Errorf("unknown hook type: %s", hookType)
	}

	return nil
}

// printJSON prints a value as JSON for debugging.
func printJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

const clientsDir = "/var/run/gatekey/clients"

// writeClientFile writes client info to a file for the daemon.
func writeClientFile(vpnIP string, client ConnectedClient) error {
	if err := os.MkdirAll(clientsDir, 0755); err != nil {
		return err
	}

	data, err := json.Marshal(client)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s/%s.json", clientsDir, strings.ReplaceAll(vpnIP, ".", "-"))
	return os.WriteFile(filename, data, 0644)
}

// removeClientFile removes the client info file.
func removeClientFile(vpnIP string) {
	filename := fmt.Sprintf("%s/%s.json", clientsDir, strings.ReplaceAll(vpnIP, ".", "-"))
	os.Remove(filename)
}

// loadConnectedClients loads all connected client files.
func loadConnectedClients() map[string]ConnectedClient {
	clients := make(map[string]ConnectedClient)

	entries, err := os.ReadDir(clientsDir)
	if err != nil {
		return clients
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			filepath := fmt.Sprintf("%s/%s", clientsDir, entry.Name())
			data, err := os.ReadFile(filepath)
			if err != nil {
				continue
			}

			var client ConnectedClient
			if err := json.Unmarshal(data, &client); err != nil {
				continue
			}

			clients[client.VPNIP] = client
		}
	}

	return clients
}

// syncConnectedClients syncs the connectedUsers map with client files.
func syncConnectedClients(cfg *GatewayConfig) {
	fileClients := loadConnectedClients()

	// Check for new connections
	for vpnIP, client := range fileClients {
		if _, exists := connectedUsers[vpnIP]; !exists {
			// New client connected
			connectedUsers[vpnIP] = client
			logger.Info("New client detected",
				zap.String("vpn_ip", vpnIP),
				zap.String("user_id", client.UserID))

			// Fetch and apply firewall rules
			rules, err := fetchClientRules(cfg, client.UserID, client.UserEmail, client.UserGroups, vpnIP)
			if err != nil {
				logger.Warn("Failed to fetch rules for new client",
					zap.String("vpn_ip", vpnIP),
					zap.Error(err))
				continue
			}

			if err := applyFirewallRules(cfg, vpnIP, client.UserID, rules); err != nil {
				logger.Warn("Failed to apply firewall rules",
					zap.String("vpn_ip", vpnIP),
					zap.Error(err))
			} else {
				logger.Info("Applied firewall rules for client",
					zap.String("vpn_ip", vpnIP),
					zap.Int("rule_count", len(rules.Allowed)))
			}
		}
	}

	// Check for disconnections
	for vpnIP := range connectedUsers {
		if _, exists := fileClients[vpnIP]; !exists {
			// Client disconnected
			logger.Info("Client disconnected",
				zap.String("vpn_ip", vpnIP))

			// Remove firewall rules
			if err := removeFirewallRules(vpnIP); err != nil {
				logger.Warn("Failed to remove firewall rules",
					zap.String("vpn_ip", vpnIP),
					zap.Error(err))
			}

			delete(connectedUsers, vpnIP)
		}
	}
}
