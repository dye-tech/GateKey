// GateKey Mesh Hub
// This is a standalone hub server that runs OpenVPN and connects mesh gateways.
// It communicates with the GateKey control plane for configuration and route management.
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
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/firewall"
)

var (
	configPath       string
	logger           *zap.Logger
	currentConfigVer string
	firewallMgr      *firewall.Manager
)

const configVersionFile = "/etc/gatekey-hub/.config_version"

func main() {
	rootCmd := &cobra.Command{
		Use:   "gatekey-hub",
		Short: "GateKey Mesh Hub Server",
		Long: `GateKey Mesh Hub Server runs OpenVPN and accepts connections from:
- Mesh Gateways (remote sites connecting back to hub)
- VPN Clients (users connecting to access mesh resources)

The hub communicates with the GateKey control plane to:
- Receive configuration updates
- Get route information from connected gateways
- Report connection status and health`,
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "/etc/gatekey-hub/config.yaml", "config file path")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the mesh hub server",
		RunE:  runHub,
	}

	provisionCmd := &cobra.Command{
		Use:   "provision",
		Short: "Provision certificates from control plane",
		RunE:  provisionHub,
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show hub status",
		RunE:  showStatus,
	}

	rootCmd.AddCommand(runCmd, provisionCmd, statusCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// HubConfig holds hub configuration
type HubConfig struct {
	Name              string        `mapstructure:"name"`
	ControlPlaneURL   string        `mapstructure:"control_plane_url"`
	APIToken          string        `mapstructure:"api_token"`
	VPNPort           int           `mapstructure:"vpn_port"`
	VPNProtocol       string        `mapstructure:"vpn_protocol"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	LogLevel          string        `mapstructure:"log_level"`
}

// ProvisionResponse from control plane
type ProvisionResponse struct {
	CACert         string `json:"cacert"`
	ServerCert     string `json:"servercert"`
	ServerKey      string `json:"serverkey"`
	DHParams       string `json:"dhparams"`
	TLSAuthEnabled bool   `json:"tlsauthenabled"`
	TLSAuthKey     string `json:"tlsauthkey"`
	VPNPort        int    `json:"vpnport"`
	VPNProtocol    string `json:"vpnprotocol"`
	VPNSubnet      string `json:"vpnsubnet"`
	CryptoProfile  string `json:"cryptoprofile"`
	ConfigVersion  string `json:"configversion"`
}

// HeartbeatResponse from control plane
type HeartbeatResponse struct {
	OK               bool   `json:"ok"`
	NeedsReprovision bool   `json:"needsReprovision"`
	ConfigVersion    string `json:"configVersion"`
}

func loadConfig() (*HubConfig, error) {
	v := viper.New()
	v.SetConfigFile(configPath)

	v.SetDefault("vpn_port", 1194)
	v.SetDefault("vpn_protocol", "udp")
	v.SetDefault("heartbeat_interval", "30s")
	v.SetDefault("log_level", "info")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	v.SetEnvPrefix("GATEKEY_HUB")
	v.AutomaticEnv()

	var cfg HubConfig
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

func loadConfigVersion() string {
	data, err := os.ReadFile(configVersionFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveConfigVersion(version string) error {
	return os.WriteFile(configVersionFile, []byte(version), 0600)
}

func runHub(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	logger, err = initLogger(cfg.LogLevel)
	if err != nil {
		return err
	}
	defer logger.Sync()

	logger.Info("Starting GateKey Mesh Hub",
		zap.String("name", cfg.Name),
		zap.String("control_plane", cfg.ControlPlaneURL),
		zap.Int("vpn_port", cfg.VPNPort),
	)

	// Load persisted config version
	currentConfigVer = loadConfigVersion()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initial provision if no config exists
	if currentConfigVer == "" {
		logger.Info("No configuration found, running initial provision...")
		if err := doProvision(ctx, cfg); err != nil {
			logger.Error("Initial provision failed", zap.Error(err))
			return fmt.Errorf("initial provision failed: %w", err)
		}
	}

	// Initialize firewall manager for zero-trust enforcement
	nftBackend, err := firewall.NewNFTablesBackend(firewall.NFTablesConfig{
		TableName: "gatekey",
		ChainName: "forward",
	})
	if err != nil {
		logger.Warn("Failed to create nftables backend, firewall rules will not be enforced", zap.Error(err))
	} else {
		firewallMgr = firewall.NewManager(nftBackend)
		if err := firewallMgr.Initialize(ctx); err != nil {
			logger.Warn("Failed to initialize firewall manager", zap.Error(err))
			firewallMgr = nil
		} else {
			logger.Info("Firewall manager initialized")
		}
	}

	// Start OpenVPN if not running
	if !isOpenVPNRunning() {
		logger.Info("Starting OpenVPN...")
		if err := startOpenVPN(); err != nil {
			logger.Warn("Failed to start OpenVPN", zap.Error(err))
		}
	}

	// Start heartbeat loop
	go heartbeatLoop(ctx, cfg)

	// Start gateway monitoring loop
	go gatewayMonitorLoop(ctx, cfg)

	// Start firewall enforcement loop (zero-trust)
	go firewallEnforcementLoop(ctx, cfg)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down mesh hub")

	// Cleanup firewall rules
	if firewallMgr != nil {
		if err := firewallMgr.Cleanup(context.Background()); err != nil {
			logger.Warn("Failed to cleanup firewall rules", zap.Error(err))
		}
	}

	return nil
}

func heartbeatLoop(ctx context.Context, cfg *HubConfig) {
	ticker := time.NewTicker(cfg.HeartbeatInterval)
	defer ticker.Stop()

	// Send initial heartbeat
	sendHeartbeat(ctx, cfg)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resp, err := sendHeartbeat(ctx, cfg)
			if err != nil {
				logger.Warn("Heartbeat failed", zap.Error(err))
				continue
			}

			if resp.NeedsReprovision {
				logger.Info("Control plane signaled reprovision needed",
					zap.String("current_version", currentConfigVer),
					zap.String("server_version", resp.ConfigVersion))

				if err := doProvision(ctx, cfg); err != nil {
					logger.Error("Reprovision failed", zap.Error(err))
				} else {
					currentConfigVer = resp.ConfigVersion
					if err := saveConfigVersion(currentConfigVer); err != nil {
						logger.Warn("Failed to save config version", zap.Error(err))
					}
					logger.Info("Reprovision completed", zap.String("config_version", currentConfigVer))

					// Restart OpenVPN to pick up new config
					if err := restartOpenVPN(); err != nil {
						logger.Error("Failed to restart OpenVPN", zap.Error(err))
					}
				}
			}
		}
	}
}

func sendHeartbeat(ctx context.Context, cfg *HubConfig) (*HeartbeatResponse, error) {
	reqBody := struct {
		Token             string `json:"token"`
		Status            string `json:"status"`
		ConnectedGateways int    `json:"connectedGateways"`
		ConnectedClients  int    `json:"connectedClients"`
		ConfigVersion     string `json:"configVersion"`
	}{
		Token:             cfg.APIToken,
		Status:            "online",
		ConnectedGateways: getConnectedGatewayCount(),
		ConnectedClients:  getConnectedClientCount(),
		ConfigVersion:     currentConfigVer,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/mesh-hub/heartbeat"
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("control plane returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func provisionHub(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	logger, err = initLogger(cfg.LogLevel)
	if err != nil {
		return err
	}
	defer logger.Sync()

	ctx := context.Background()
	return doProvision(ctx, cfg)
}

func doProvision(ctx context.Context, cfg *HubConfig) error {
	logger.Info("Provisioning hub from control plane...")

	reqBody := struct {
		Token string `json:"token"`
	}{
		Token: cfg.APIToken,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/mesh-hub/provision"
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("control plane returned %d: %s", resp.StatusCode, string(respBody))
	}

	var provResp ProvisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&provResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Create OpenVPN directories
	openvpnDir := "/etc/openvpn/server"
	if err := os.MkdirAll(openvpnDir, 0755); err != nil {
		return fmt.Errorf("failed to create openvpn directory: %w", err)
	}

	// Write certificates
	if err := os.WriteFile(openvpnDir+"/ca.crt", []byte(provResp.CACert), 0644); err != nil {
		return fmt.Errorf("failed to write CA cert: %w", err)
	}
	if err := os.WriteFile(openvpnDir+"/server.crt", []byte(provResp.ServerCert), 0644); err != nil {
		return fmt.Errorf("failed to write server cert: %w", err)
	}
	if err := os.WriteFile(openvpnDir+"/server.key", []byte(provResp.ServerKey), 0600); err != nil {
		return fmt.Errorf("failed to write server key: %w", err)
	}

	// Generate DH params if needed
	dhPath := openvpnDir + "/dh.pem"
	if _, err := os.Stat(dhPath); os.IsNotExist(err) {
		logger.Info("Generating DH parameters (this may take a while)...")
		cmd := exec.Command("openssl", "dhparam", "-out", dhPath, "2048")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to generate DH params: %w", err)
		}
	}

	// Write TLS-Auth key if enabled
	if provResp.TLSAuthEnabled && provResp.TLSAuthKey != "" {
		if err := os.WriteFile(openvpnDir+"/ta.key", []byte(provResp.TLSAuthKey), 0600); err != nil {
			return fmt.Errorf("failed to write TLS-Auth key: %w", err)
		}
	}

	// Generate OpenVPN server config
	serverConfig := generateServerConfig(provResp)
	if err := os.WriteFile(openvpnDir+"/hub.conf", []byte(serverConfig), 0644); err != nil {
		return fmt.Errorf("failed to write server config: %w", err)
	}

	// Save config version
	currentConfigVer = provResp.ConfigVersion
	if err := saveConfigVersion(currentConfigVer); err != nil {
		logger.Warn("Failed to save config version", zap.Error(err))
	}

	logger.Info("Hub provisioned successfully",
		zap.String("config_version", currentConfigVer),
		zap.Int("vpn_port", provResp.VPNPort),
		zap.String("vpn_protocol", provResp.VPNProtocol),
	)

	return nil
}

func generateServerConfig(prov ProvisionResponse) string {
	var sb strings.Builder

	sb.WriteString("# GateKey Mesh Hub OpenVPN Server Configuration\n")
	sb.WriteString("# Auto-generated - do not edit manually\n\n")

	sb.WriteString(fmt.Sprintf("port %d\n", prov.VPNPort))
	sb.WriteString(fmt.Sprintf("proto %s\n", prov.VPNProtocol))
	sb.WriteString("dev tun\n\n")

	sb.WriteString("# Certificate files\n")
	sb.WriteString("ca /etc/openvpn/server/ca.crt\n")
	sb.WriteString("cert /etc/openvpn/server/server.crt\n")
	sb.WriteString("key /etc/openvpn/server/server.key\n")
	sb.WriteString("dh /etc/openvpn/server/dh.pem\n\n")

	if prov.TLSAuthEnabled {
		sb.WriteString("# TLS-Auth for additional security\n")
		sb.WriteString("tls-auth /etc/openvpn/server/ta.key 0\n\n")
	}

	// VPN subnet
	subnet := prov.VPNSubnet
	if subnet == "" {
		subnet = "172.30.0.0/16"
	}
	// Parse subnet to get network and mask
	parts := strings.Split(subnet, "/")
	if len(parts) == 2 {
		network := parts[0]
		// Simple mask calculation for /16, /24, etc.
		mask := "255.255.0.0"
		if parts[1] == "24" {
			mask = "255.255.255.0"
		}
		sb.WriteString(fmt.Sprintf("server %s %s\n\n", network, mask))
	}

	sb.WriteString("# Client configuration directory for spoke routes\n")
	sb.WriteString("client-config-dir /etc/openvpn/server/ccd\n\n")

	sb.WriteString("# Enable routing between clients (hub-and-spoke)\n")
	sb.WriteString("client-to-client\n\n")

	sb.WriteString("# Keep connections alive\n")
	sb.WriteString("keepalive 10 120\n\n")

	// Crypto profile
	switch prov.CryptoProfile {
	case "fips":
		sb.WriteString("# FIPS 140-3 compliant crypto\n")
		sb.WriteString("cipher AES-256-GCM\n")
		sb.WriteString("data-ciphers AES-256-GCM:AES-128-GCM\n")
		sb.WriteString("auth SHA384\n")
		sb.WriteString("tls-cipher TLS-ECDHE-ECDSA-WITH-AES-256-GCM-SHA384:TLS-ECDHE-RSA-WITH-AES-256-GCM-SHA384\n")
	case "compatible":
		sb.WriteString("# Maximum compatibility crypto\n")
		sb.WriteString("cipher AES-256-GCM\n")
		sb.WriteString("data-ciphers AES-256-GCM:AES-128-GCM:AES-256-CBC:AES-128-CBC\n")
		sb.WriteString("auth SHA256\n")
	default: // modern
		sb.WriteString("# Modern secure crypto\n")
		sb.WriteString("cipher AES-256-GCM\n")
		sb.WriteString("data-ciphers AES-256-GCM:CHACHA20-POLY1305\n")
		sb.WriteString("auth SHA256\n")
	}
	sb.WriteString("\n")

	sb.WriteString("# Logging\n")
	sb.WriteString("status /var/log/openvpn/hub-status.log\n")
	sb.WriteString("log-append /var/log/openvpn/hub.log\n")
	sb.WriteString("verb 1\n\n")

	sb.WriteString("# Persist settings across restarts\n")
	sb.WriteString("persist-key\n")
	sb.WriteString("persist-tun\n\n")

	sb.WriteString("# Hook scripts for gateway/client management\n")
	sb.WriteString("script-security 2\n")
	sb.WriteString("client-connect \"/usr/local/bin/gatekey-hub-hook connect\"\n")
	sb.WriteString("client-disconnect \"/usr/local/bin/gatekey-hub-hook disconnect\"\n")

	return sb.String()
}

func gatewayMonitorLoop(ctx context.Context, cfg *HubConfig) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// CRITICAL: Create CCD files immediately on startup, before any spokes connect
	// This prevents spokes from getting wrong IPs from the dynamic pool
	logger.Info("Creating initial CCD files for spokes...")
	updateGatewayRoutes(ctx, cfg)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Fetch routes from control plane and update CCD files
			updateGatewayRoutes(ctx, cfg)
		}
	}
}

func updateGatewayRoutes(ctx context.Context, cfg *HubConfig) {
	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/mesh-hub/spokes?token=" + cfg.APIToken
	resp, err := http.Get(url)
	if err != nil {
		logger.Warn("Failed to fetch spokes", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("Failed to fetch spokes", zap.Int("status", resp.StatusCode))
		return
	}

	var result struct {
		Spokes []struct {
			ID            string   `json:"id"`
			Name          string   `json:"name"`
			LocalNetworks []string `json:"localNetworks"`
			TunnelIP      string   `json:"tunnelIp"`
		} `json:"spokes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Warn("Failed to decode spokes", zap.Error(err))
		return
	}

	// Use the OpenVPN server's CCD directory
	ccdDir := "/etc/openvpn/server/ccd"
	_ = os.MkdirAll(ccdDir, 0755)

	needsRestart := false

	// Update CCD files and kernel routes for each spoke
	for _, spoke := range result.Spokes {
		if spoke.TunnelIP == "" {
			logger.Debug("Skipping spoke without tunnel IP", zap.String("spoke", spoke.Name))
			continue
		}

		// CCD file content: iroute directives for this spoke's networks
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# Spoke: %s\n", spoke.Name))
		sb.WriteString(fmt.Sprintf("ifconfig-push %s 255.255.0.0\n", spoke.TunnelIP))
		for _, network := range spoke.LocalNetworks {
			netIP, mask := cidrToNetmask(network)
			if netIP != "" && mask != "" {
				sb.WriteString(fmt.Sprintf("iroute %s %s\n", netIP, mask))
			}
		}

		newContent := sb.String()

		// Write CCD file (use spoke certificate CN as filename)
		ccdFile := fmt.Sprintf("%s/mesh-gateway-%s", ccdDir, spoke.Name)

		// Check if CCD file content changed
		existingContent, readErr := os.ReadFile(ccdFile)
		if readErr != nil || string(existingContent) != newContent {
			if err := os.WriteFile(ccdFile, []byte(newContent), 0644); err != nil {
				logger.Warn("Failed to write CCD file", zap.String("spoke", spoke.Name), zap.Error(err))
			} else {
				logger.Info("Updated CCD file", zap.String("spoke", spoke.Name), zap.String("file", ccdFile))
				// If file existed and content was different, flag for restart
				if readErr == nil && string(existingContent) != newContent {
					needsRestart = true
				}
			}
		}

		// Add kernel routes for each spoke network via the spoke's tunnel IP
		for _, network := range spoke.LocalNetworks {
			addKernelRoute(network, spoke.TunnelIP)
		}
	}

	// If CCD files changed, restart OpenVPN so clients reconnect with correct IPs
	if needsRestart {
		logger.Info("CCD files changed, restarting OpenVPN to apply new configurations...")
		if err := restartOpenVPN(); err != nil {
			logger.Warn("Failed to restart OpenVPN", zap.Error(err))
		}
	}
}

// cidrToNetmask converts CIDR notation to network IP and netmask
func cidrToNetmask(cidr string) (string, string) {
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return "", ""
	}
	netIP := parts[0]
	prefix := parts[1]

	// Convert prefix to netmask
	maskMap := map[string]string{
		"8":  "255.0.0.0",
		"9":  "255.128.0.0",
		"10": "255.192.0.0",
		"11": "255.224.0.0",
		"12": "255.240.0.0",
		"13": "255.248.0.0",
		"14": "255.252.0.0",
		"15": "255.254.0.0",
		"16": "255.255.0.0",
		"17": "255.255.128.0",
		"18": "255.255.192.0",
		"19": "255.255.224.0",
		"20": "255.255.240.0",
		"21": "255.255.248.0",
		"22": "255.255.252.0",
		"23": "255.255.254.0",
		"24": "255.255.255.0",
		"25": "255.255.255.128",
		"26": "255.255.255.192",
		"27": "255.255.255.224",
		"28": "255.255.255.240",
		"29": "255.255.255.248",
		"30": "255.255.255.252",
		"31": "255.255.255.254",
		"32": "255.255.255.255",
	}

	mask, ok := maskMap[prefix]
	if !ok {
		return "", ""
	}
	return netIP, mask
}

// addKernelRoute adds a route in the kernel routing table
func addKernelRoute(network, gateway string) {
	// Check if route already exists
	checkCmd := exec.Command("ip", "route", "show", network)
	output, _ := checkCmd.Output()
	if len(output) > 0 && strings.Contains(string(output), gateway) {
		// Route already exists with correct gateway
		return
	}

	// Add the route (replace if exists with different gateway)
	cmd := exec.Command("ip", "route", "replace", network, "via", gateway)
	if err := cmd.Run(); err != nil {
		logger.Warn("Failed to add kernel route",
			zap.String("network", network),
			zap.String("gateway", gateway),
			zap.Error(err))
	} else {
		logger.Info("Added kernel route",
			zap.String("network", network),
			zap.String("gateway", gateway))
	}
}

func showStatus(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	fmt.Printf("GateKey Mesh Hub Status\n")
	fmt.Printf("=======================\n")
	fmt.Printf("Name: %s\n", cfg.Name)
	fmt.Printf("Control Plane: %s\n", cfg.ControlPlaneURL)
	fmt.Printf("VPN Port: %d/%s\n", cfg.VPNPort, cfg.VPNProtocol)
	fmt.Printf("Config Version: %s\n", loadConfigVersion())
	fmt.Printf("OpenVPN Running: %v\n", isOpenVPNRunning())
	fmt.Printf("Connected Gateways: %d\n", getConnectedGatewayCount())
	fmt.Printf("Connected Clients: %d\n", getConnectedClientCount())

	return nil
}

func isOpenVPNRunning() bool {
	if _, err := os.Stat("/run/openvpn/server.pid"); err == nil {
		return true
	}
	if _, err := os.Stat("/var/run/openvpn/server.pid"); err == nil {
		return true
	}
	return false
}

func startOpenVPN() error {
	cmd := exec.Command("systemctl", "start", "openvpn-server@hub")
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("systemctl", "start", "openvpn@hub")
		return cmd.Run()
	}
	return nil
}

func restartOpenVPN() error {
	cmd := exec.Command("systemctl", "restart", "openvpn-server@hub")
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("systemctl", "restart", "openvpn@hub")
		return cmd.Run()
	}
	return nil
}

func getConnectedGatewayCount() int {
	// Parse OpenVPN status file for connected gateways
	// Gateways have CN starting with "mesh-gateway-"
	return countConnections("mesh-gateway-")
}

func getConnectedClientCount() int {
	// Parse OpenVPN status file for connected clients (non-gateway connections)
	total := countConnections("")
	gateways := countConnections("mesh-gateway-")
	return total - gateways
}

func countConnections(prefix string) int {
	statusFile := "/var/log/openvpn/hub-status.log"
	data, err := os.ReadFile(statusFile)
	if err != nil {
		return 0
	}

	count := 0
	lines := strings.Split(string(data), "\n")
	inClientList := false

	for _, line := range lines {
		if strings.HasPrefix(line, "ROUTING TABLE") {
			inClientList = false
		}
		if strings.HasPrefix(line, "Common Name,") {
			inClientList = true
			continue
		}
		if inClientList && line != "" {
			parts := strings.Split(line, ",")
			if len(parts) >= 1 {
				cn := parts[0]
				if prefix == "" || strings.HasPrefix(cn, prefix) {
					count++
				}
			}
		}
	}

	return count
}

// ==================== Firewall Enforcement ====================

// ConnectedClient represents a connected VPN client
type ConnectedClient struct {
	CN       string // Common Name (email)
	TunnelIP string // VPN tunnel IP
	RealIP   string // Real client IP
}

// AccessRule represents an access rule from the control plane
type AccessRule struct {
	Type     string `json:"type"`     // ip, cidr, hostname, hostname_wildcard
	Value    string `json:"value"`    // Rule value
	Port     string `json:"port"`     // Port or * for all
	Protocol string `json:"protocol"` // tcp, udp, * for all
}

// clientFirewallState tracks firewall rules for a client
type clientFirewallState struct {
	rules    []AccessRule
	rulesSet bool
}

var (
	clientFirewallStates = make(map[string]*clientFirewallState) // CN -> state
	clientFirewallMutex  = &sync.Mutex{}
)

// firewallEnforcementLoop periodically syncs firewall rules for connected clients
func firewallEnforcementLoop(ctx context.Context, cfg *HubConfig) {
	if firewallMgr == nil {
		logger.Warn("Firewall manager not initialized, skipping enforcement loop")
		return
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Run initial sync
	syncFirewallRules(ctx, cfg)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncFirewallRules(ctx, cfg)
		}
	}
}

// getConnectedClients parses OpenVPN status to get connected clients
func getConnectedClients() []ConnectedClient {
	statusFile := "/var/log/openvpn/hub-status.log"
	data, err := os.ReadFile(statusFile)
	if err != nil {
		return nil
	}

	var clients []ConnectedClient
	lines := strings.Split(string(data), "\n")

	// Parse client list (Common Name, Real Address, Virtual Address, ...)
	inClientList := false
	for _, line := range lines {
		if strings.HasPrefix(line, "ROUTING TABLE") {
			inClientList = false
		}
		if strings.HasPrefix(line, "Virtual Address,") || strings.HasPrefix(line, "Common Name,") {
			inClientList = true
			continue
		}
		if inClientList && line != "" {
			parts := strings.Split(line, ",")
			if len(parts) >= 3 {
				cn := parts[0]
				// Skip mesh gateways - they don't need client firewall rules
				if strings.HasPrefix(cn, "mesh-gateway-") {
					continue
				}
				realIP := parts[1]
				if idx := strings.Index(realIP, ":"); idx > 0 {
					realIP = realIP[:idx]
				}
				clients = append(clients, ConnectedClient{
					CN:     cn,
					RealIP: realIP,
				})
			}
		}
	}

	// Get virtual IPs from routing table
	inRoutingTable := false
	for _, line := range lines {
		if strings.HasPrefix(line, "ROUTING TABLE") {
			inRoutingTable = true
			continue
		}
		if strings.HasPrefix(line, "GLOBAL STATS") {
			break
		}
		if inRoutingTable && line != "" && !strings.HasPrefix(line, "Virtual Address,") {
			parts := strings.Split(line, ",")
			if len(parts) >= 2 {
				virtualIP := parts[0]
				cn := parts[1]
				// Match with client and update tunnel IP
				for i := range clients {
					if clients[i].CN == cn {
						clients[i].TunnelIP = virtualIP
						break
					}
				}
			}
		}
	}

	return clients
}

func syncFirewallRules(ctx context.Context, cfg *HubConfig) {
	if firewallMgr == nil {
		return
	}

	clients := getConnectedClients()
	if len(clients) == 0 {
		return
	}

	// Build list of client emails
	var emails []string
	for _, c := range clients {
		if c.CN != "" {
			emails = append(emails, c.CN)
		}
	}

	// Fetch rules from control plane
	clientRules := fetchClientRules(ctx, cfg, emails)
	if clientRules == nil {
		return
	}

	clientFirewallMutex.Lock()
	defer clientFirewallMutex.Unlock()

	// Track active clients
	activeClients := make(map[string]bool)
	for _, c := range clients {
		activeClients[c.CN] = true
	}

	// Remove firewall rules for disconnected clients
	for cn := range clientFirewallStates {
		if !activeClients[cn] {
			removeClientFirewallRules(ctx, cn)
			delete(clientFirewallStates, cn)
		}
	}

	// Update firewall rules for connected clients
	for _, client := range clients {
		if client.TunnelIP == "" {
			continue
		}

		rules, ok := clientRules[client.CN]
		if !ok {
			// No rules = no access (zero trust)
			rules = []AccessRule{}
		}

		state, exists := clientFirewallStates[client.CN]
		if !exists {
			state = &clientFirewallState{}
			clientFirewallStates[client.CN] = state
		}

		// Check if rules changed
		rulesChanged := !state.rulesSet || !rulesEqual(state.rules, rules)
		if rulesChanged {
			applyClientFirewallRules(ctx, client, rules)
			state.rules = rules
			state.rulesSet = true
		}
	}
}

func fetchClientRules(ctx context.Context, cfg *HubConfig, emails []string) map[string][]AccessRule {
	reqBody := struct {
		Token   string   `json:"token"`
		Clients []string `json:"clients"`
	}{
		Token:   cfg.APIToken,
		Clients: emails,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		logger.Warn("Failed to marshal request", zap.Error(err))
		return nil
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/mesh-hub/all-client-rules"
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		logger.Warn("Failed to fetch client rules", zap.Error(err))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("Control plane returned error", zap.Int("status", resp.StatusCode))
		return nil
	}

	var result struct {
		ClientRules map[string][]AccessRule `json:"clientRules"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Warn("Failed to decode response", zap.Error(err))
		return nil
	}

	return result.ClientRules
}

func rulesEqual(a, b []AccessRule) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func applyClientFirewallRules(ctx context.Context, client ConnectedClient, rules []AccessRule) {
	if firewallMgr == nil {
		return
	}

	// Remove existing rules for this client first
	connectionID := fmt.Sprintf("mesh-client-%s", strings.ReplaceAll(client.TunnelIP, ".", "-"))
	_ = firewallMgr.RemoveRules(ctx, connectionID)

	logger.Info("Applying firewall rules",
		zap.String("client", client.CN),
		zap.String("tunnelIP", client.TunnelIP),
		zap.Int("ruleCount", len(rules)))

	// Parse source IP
	sourceIP := net.ParseIP(client.TunnelIP)
	if sourceIP == nil {
		logger.Warn("Invalid tunnel IP", zap.String("tunnelIP", client.TunnelIP))
		return
	}

	// Convert access rules to firewall networks and ports
	var networks []net.IPNet
	var ports []firewall.PortRange

	for _, rule := range rules {
		switch rule.Type {
		case "ip":
			ip := net.ParseIP(rule.Value)
			if ip != nil {
				networks = append(networks, net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)})
			}
		case "cidr":
			_, ipnet, err := net.ParseCIDR(rule.Value)
			if err == nil && ipnet != nil {
				networks = append(networks, *ipnet)
			}
		case "hostname", "hostname_wildcard":
			// Resolve hostname to IP
			ips, err := net.LookupIP(rule.Value)
			if err != nil || len(ips) == 0 {
				logger.Debug("Failed to resolve hostname",
					zap.String("hostname", rule.Value),
					zap.Error(err))
				continue
			}
			// Add all IPv4 addresses
			for _, ip := range ips {
				if ip4 := ip.To4(); ip4 != nil {
					networks = append(networks, net.IPNet{IP: ip4, Mask: net.CIDRMask(32, 32)})
				}
			}
		}

		// Parse port if specified
		if rule.Port != "" && rule.Port != "*" {
			protocol := firewall.ProtocolAny
			switch rule.Protocol {
			case "tcp":
				protocol = firewall.ProtocolTCP
			case "udp":
				protocol = firewall.ProtocolUDP
			}

			if strings.Contains(rule.Port, "-") {
				// Port range
				parts := strings.Split(rule.Port, "-")
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
				port, _ := strconv.Atoi(rule.Port)
				if port > 0 {
					ports = append(ports, firewall.PortRange{
						Protocol: protocol,
						Port:     port,
					})
				}
			}
		}
	}

	// Generate a UUID from client email for firewall tracking
	// Use namespace UUID to generate deterministic UUID from email
	userUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(client.CN))

	// Apply rules using firewall manager
	if err := firewallMgr.ApplyRules(ctx, connectionID, userUUID, sourceIP, networks, ports); err != nil {
		logger.Warn("Failed to apply firewall rules",
			zap.String("client", client.CN),
			zap.Error(err))
	}
}

func removeClientFirewallRules(ctx context.Context, cn string) {
	if firewallMgr == nil {
		return
	}

	// Find client's tunnel IP from connected clients
	clients := getConnectedClients()
	var tunnelIP string
	for _, c := range clients {
		if c.CN == cn {
			tunnelIP = c.TunnelIP
			break
		}
	}
	if tunnelIP == "" {
		return
	}

	connectionID := fmt.Sprintf("mesh-client-%s", strings.ReplaceAll(tunnelIP, ".", "-"))
	if err := firewallMgr.RemoveRules(ctx, connectionID); err != nil {
		logger.Debug("Error removing firewall rules", zap.String("cn", cn), zap.Error(err))
	}
}
