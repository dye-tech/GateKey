// GateKey Mesh Gateway
// This binary runs on remote gateway sites and connects TO the mesh hub using OpenVPN.
// It advertises local networks to the hub and enables hub-and-spoke mesh connectivity.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gatekey-project/gatekey/internal/session"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	configPath       string
	logger           *zap.Logger
	currentConfigVer string
	provisionedName  string // Name from control plane provisioning
)

const (
	configVersionFile = "/etc/gatekey-mesh/.config_version"
	gatewayNameFile   = "/etc/gatekey-mesh/.gateway_name"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gatekey-mesh-gateway",
		Short: "GateKey Mesh Gateway Agent",
		Long: `GateKey Mesh Gateway Agent runs on remote sites and connects TO the mesh hub.
It uses OpenVPN in CLIENT mode to establish a connection back to the hub,
allowing hub-and-spoke mesh networking without requiring public IPs on gateway sites.

The gateway:
- Connects to the hub's OpenVPN server
- Advertises local networks (iroute)
- Enables routing between hub clients and local resources`,
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "/etc/gatekey-mesh/config.yaml", "config file path")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the mesh gateway agent",
		RunE:  runGateway,
	}

	provisionCmd := &cobra.Command{
		Use:   "provision",
		Short: "Provision certificates from control plane",
		RunE:  provisionGateway,
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show gateway status",
		RunE:  showStatus,
	}

	rootCmd.AddCommand(runCmd, provisionCmd, statusCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// GatewayConfig holds gateway configuration
type GatewayConfig struct {
	Name              string        `mapstructure:"name"`
	ControlPlaneURL   string        `mapstructure:"control_plane_url"`
	GatewayToken      string        `mapstructure:"gateway_token"`
	HubEndpoint       string        `mapstructure:"hub_endpoint"`
	LocalNetworks     []string      `mapstructure:"local_networks"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	LogLevel          string        `mapstructure:"log_level"`
	SessionEnabled    bool          `mapstructure:"session_enabled"`
}

// ProvisionResponse from control plane
type ProvisionResponse struct {
	GatewayID      string   `json:"gatewayId"`
	GatewayName    string   `json:"gatewayName"` // Name for session authentication
	HubEndpoint    string   `json:"hubEndpoint"`
	HubVPNPort     int      `json:"hubVpnPort"`
	HubVPNProtocol string   `json:"hubVpnProtocol"`
	CACert         string   `json:"caCert"`
	ClientCert     string   `json:"clientCert"`
	ClientKey      string   `json:"clientKey"`
	TunnelIP       string   `json:"tunnelIp"`
	LocalNetworks  []string `json:"localNetworks"`
	TLSAuthEnabled bool     `json:"tlsAuthEnabled"`
	TLSAuthKey     string   `json:"tlsAuthKey"`
	CryptoProfile  string   `json:"cryptoProfile"`
	ConfigVersion  string   `json:"configVersion"`
}

func loadConfig() (*GatewayConfig, error) {
	v := viper.New()
	v.SetConfigFile(configPath)

	v.SetDefault("heartbeat_interval", "30s")
	v.SetDefault("log_level", "info")
	v.SetDefault("session_enabled", true)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	v.SetEnvPrefix("GATEKEY_MESH")
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

func loadGatewayName() string {
	data, err := os.ReadFile(gatewayNameFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveGatewayName(name string) error {
	return os.WriteFile(gatewayNameFile, []byte(name), 0600)
}

func runGateway(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	logger, err = initLogger(cfg.LogLevel)
	if err != nil {
		return err
	}
	defer logger.Sync()

	logger.Info("Starting GateKey Mesh Gateway",
		zap.String("name", cfg.Name),
		zap.String("control_plane", cfg.ControlPlaneURL),
		zap.Strings("local_networks", cfg.LocalNetworks),
	)

	// Load persisted config version and gateway name
	currentConfigVer = loadConfigVersion()
	provisionedName = loadGatewayName()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initial provision if no config exists
	if currentConfigVer == "" {
		logger.Info("No configuration found, running initial provision...")
		if err := doProvision(ctx, cfg); err != nil {
			logger.Error("Initial provision failed", zap.Error(err))
			return fmt.Errorf("initial provision failed: %w", err)
		}
		// Reload provisioned name after provisioning
		provisionedName = loadGatewayName()
	}

	// Determine effective name: prefer config, fallback to provisioned name
	effectiveName := cfg.Name
	if effectiveName == "" {
		effectiveName = provisionedName
	}
	if effectiveName != "" {
		logger.Info("Using gateway name", zap.String("name", effectiveName))
	}

	// Start OpenVPN client if not running
	if !isOpenVPNRunning() {
		logger.Info("Starting OpenVPN client...")
		if err := startOpenVPN(); err != nil {
			logger.Warn("Failed to start OpenVPN", zap.Error(err))
		}
	}

	// Start remote session client (connects outbound to control plane)
	var sessionClient *session.AgentClient
	if cfg.SessionEnabled && effectiveName != "" {
		sessionClient = session.NewAgentClient(&session.AgentClientConfig{
			ControlPlaneURL: cfg.ControlPlaneURL,
			Token:           cfg.GatewayToken,
			NodeType:        "spoke",
			NodeID:          effectiveName,
			NodeName:        effectiveName,
			Logger:          logger,
		})
		sessionClient.Start(ctx)
		logger.Info("Remote session client started")
	} else if effectiveName == "" {
		logger.Warn("Remote sessions disabled: gateway name not available (set 'name' in config or run provision)")
	}

	// Start heartbeat loop
	go heartbeatLoop(ctx, cfg)

	// Start connection monitor loop
	go connectionMonitorLoop(ctx)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down mesh gateway")

	// Stop session client
	if sessionClient != nil {
		sessionClient.Stop()
	}

	return nil
}

func heartbeatLoop(ctx context.Context, cfg *GatewayConfig) {
	ticker := time.NewTicker(cfg.HeartbeatInterval)
	defer ticker.Stop()

	// Send initial heartbeat
	sendHeartbeat(ctx, cfg)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendHeartbeat(ctx, cfg)
		}
	}
}

// HeartbeatResponse from control plane
type HeartbeatResponse struct {
	OK               bool   `json:"ok"`
	ConfigVersion    string `json:"configVersion"`
	NeedsReprovision bool   `json:"needsReprovision"`
	TLSAuthEnabled   bool   `json:"tlsAuthEnabled"`
}

func sendHeartbeat(ctx context.Context, cfg *GatewayConfig) {
	status := "disconnected"
	if isOpenVPNConnected() {
		status = "connected"
	}

	reqBody := struct {
		Token         string `json:"token"`
		Status        string `json:"status"`
		RemoteIP      string `json:"remoteIp"`
		BytesSent     int64  `json:"bytesSent"`
		BytesReceived int64  `json:"bytesReceived"`
		ConfigVersion string `json:"configVersion"`
	}{
		Token:         cfg.GatewayToken,
		Status:        status,
		RemoteIP:      getPublicIP(),
		BytesSent:     getBytesSent(),
		BytesReceived: getBytesReceived(),
		ConfigVersion: currentConfigVer,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		logger.Warn("Failed to marshal heartbeat", zap.Error(err))
		return
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/mesh-gateway/heartbeat"
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		logger.Warn("Heartbeat failed", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Warn("Heartbeat returned error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)))
		return
	}

	// Parse heartbeat response
	var hbResp HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&hbResp); err != nil {
		logger.Warn("Failed to decode heartbeat response", zap.Error(err))
		return
	}

	// Check if we need to reprovision (config changed on control plane)
	if hbResp.NeedsReprovision {
		logger.Info("Config version mismatch detected, reprovisioning...",
			zap.String("local_version", currentConfigVer),
			zap.String("hub_version", hbResp.ConfigVersion))

		// Reprovision from control plane
		if err := doProvision(ctx, cfg); err != nil {
			logger.Error("Failed to reprovision", zap.Error(err))
			return
		}

		// Update local config version
		currentConfigVer = hbResp.ConfigVersion
		if err := saveConfigVersion(currentConfigVer); err != nil {
			logger.Warn("Failed to save config version", zap.Error(err))
		}

		// Restart OpenVPN to apply new configuration
		logger.Info("Restarting OpenVPN with new configuration...")
		if err := restartOpenVPN(); err != nil {
			logger.Error("Failed to restart OpenVPN", zap.Error(err))
		} else {
			logger.Info("OpenVPN restarted successfully")
		}
	}
}

func connectionMonitorLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if OpenVPN is connected, restart if needed
			if !isOpenVPNConnected() && isOpenVPNRunning() {
				logger.Warn("OpenVPN running but not connected, checking...")
			}
		}
	}
}

func provisionGateway(cmd *cobra.Command, args []string) error {
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

func doProvision(ctx context.Context, cfg *GatewayConfig) error {
	logger.Info("Provisioning gateway from control plane...")

	reqBody := struct {
		Token string `json:"token"`
	}{
		Token: cfg.GatewayToken,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/mesh-gateway/provision"
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
	openvpnDir := "/etc/openvpn/client"
	if err := os.MkdirAll(openvpnDir, 0755); err != nil {
		return fmt.Errorf("failed to create openvpn directory: %w", err)
	}

	// Write certificates
	if err := os.WriteFile(openvpnDir+"/ca.crt", []byte(provResp.CACert), 0644); err != nil {
		return fmt.Errorf("failed to write CA cert: %w", err)
	}
	if err := os.WriteFile(openvpnDir+"/client.crt", []byte(provResp.ClientCert), 0644); err != nil {
		return fmt.Errorf("failed to write client cert: %w", err)
	}
	if err := os.WriteFile(openvpnDir+"/client.key", []byte(provResp.ClientKey), 0600); err != nil {
		return fmt.Errorf("failed to write client key: %w", err)
	}

	// Write TLS-Auth key if enabled
	if provResp.TLSAuthEnabled && provResp.TLSAuthKey != "" {
		if err := os.WriteFile(openvpnDir+"/ta.key", []byte(provResp.TLSAuthKey), 0600); err != nil {
			return fmt.Errorf("failed to write TLS-Auth key: %w", err)
		}
	}

	// Update hub endpoint from provision response
	hubEndpoint := provResp.HubEndpoint
	if hubEndpoint == "" {
		hubEndpoint = cfg.HubEndpoint
	}

	// Generate OpenVPN client config
	clientConfig := generateClientConfig(provResp, hubEndpoint)
	if err := os.WriteFile(openvpnDir+"/mesh-hub.conf", []byte(clientConfig), 0644); err != nil {
		return fmt.Errorf("failed to write client config: %w", err)
	}

	// Save config version
	currentConfigVer = provResp.ConfigVersion
	if currentConfigVer == "" {
		// Fallback to gateway ID if config version not provided
		currentConfigVer = provResp.GatewayID
	}
	if err := saveConfigVersion(currentConfigVer); err != nil {
		logger.Warn("Failed to save config version", zap.Error(err))
	}

	// Save gateway name for session authentication
	if provResp.GatewayName != "" {
		provisionedName = provResp.GatewayName
		if err := saveGatewayName(provResp.GatewayName); err != nil {
			logger.Warn("Failed to save gateway name", zap.Error(err))
		}
		logger.Info("Gateway name saved from provisioning", zap.String("name", provResp.GatewayName))
	}

	logger.Info("Gateway provisioned successfully",
		zap.String("name", provResp.GatewayName),
		zap.String("hub_endpoint", hubEndpoint),
		zap.String("tunnel_ip", provResp.TunnelIP),
		zap.String("config_version", currentConfigVer),
	)

	return nil
}

func generateClientConfig(prov ProvisionResponse, hubEndpoint string) string {
	var sb strings.Builder

	sb.WriteString("# GateKey Mesh Gateway OpenVPN Client Configuration\n")
	sb.WriteString("# Auto-generated - do not edit manually\n\n")

	sb.WriteString("client\n")
	sb.WriteString("dev tun\n")
	sb.WriteString(fmt.Sprintf("proto %s\n", prov.HubVPNProtocol))
	sb.WriteString("\n")

	// Parse hub endpoint (could be host:port or just host)
	host := hubEndpoint
	port := prov.HubVPNPort
	if strings.Contains(hubEndpoint, ":") {
		parts := strings.Split(hubEndpoint, ":")
		host = parts[0]
		if len(parts) > 1 {
			fmt.Sscanf(parts[1], "%d", &port)
		}
	}
	sb.WriteString(fmt.Sprintf("remote %s %d\n\n", host, port))

	sb.WriteString("# Certificate files\n")
	sb.WriteString("ca /etc/openvpn/client/ca.crt\n")
	sb.WriteString("cert /etc/openvpn/client/client.crt\n")
	sb.WriteString("key /etc/openvpn/client/client.key\n\n")

	if prov.TLSAuthEnabled {
		sb.WriteString("# TLS-Auth for additional security\n")
		sb.WriteString("tls-auth /etc/openvpn/client/ta.key 1\n\n")
	}

	// Crypto profile
	// data-ciphers is only supported in OpenVPN 2.5+, use ncp-ciphers for 2.4
	useDataCiphers := isOpenVPN25OrNewer()

	switch prov.CryptoProfile {
	case "fips":
		sb.WriteString("# FIPS 140-3 compliant crypto\n")
		sb.WriteString("cipher AES-256-GCM\n")
		if useDataCiphers {
			sb.WriteString("data-ciphers AES-256-GCM:AES-128-GCM\n")
		} else {
			sb.WriteString("ncp-ciphers AES-256-GCM:AES-128-GCM\n")
		}
		sb.WriteString("auth SHA384\n")
		sb.WriteString("tls-cipher TLS-ECDHE-ECDSA-WITH-AES-256-GCM-SHA384:TLS-ECDHE-RSA-WITH-AES-256-GCM-SHA384\n")
	case "compatible":
		sb.WriteString("# Maximum compatibility crypto\n")
		sb.WriteString("cipher AES-256-GCM\n")
		if useDataCiphers {
			sb.WriteString("data-ciphers AES-256-GCM:AES-128-GCM:AES-256-CBC:AES-128-CBC\n")
		} else {
			sb.WriteString("ncp-ciphers AES-256-GCM:AES-128-GCM:AES-256-CBC:AES-128-CBC\n")
		}
		sb.WriteString("auth SHA256\n")
	default: // modern
		sb.WriteString("# Modern secure crypto\n")
		sb.WriteString("cipher AES-256-GCM\n")
		if useDataCiphers {
			sb.WriteString("data-ciphers AES-256-GCM:CHACHA20-POLY1305\n")
		} else {
			sb.WriteString("ncp-ciphers AES-256-GCM:CHACHA20-POLY1305\n")
		}
		sb.WriteString("auth SHA256\n")
	}
	sb.WriteString("\n")

	sb.WriteString("# Keep connection alive\n")
	sb.WriteString("keepalive 10 60\n\n")

	sb.WriteString("# Persist settings across restarts\n")
	sb.WriteString("persist-key\n")
	sb.WriteString("persist-tun\n\n")

	sb.WriteString("# Reconnect forever\n")
	sb.WriteString("resolv-retry infinite\n\n")

	sb.WriteString("# Don't require user input\n")
	sb.WriteString("nobind\n\n")

	sb.WriteString("# Logging\n")
	sb.WriteString("status /var/log/openvpn/mesh-status.log\n")
	sb.WriteString("log-append /var/log/openvpn/mesh-gateway.log\n")
	sb.WriteString("verb 1\n")

	return sb.String()
}

func showStatus(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	fmt.Printf("GateKey Mesh Gateway Status\n")
	fmt.Printf("============================\n")
	fmt.Printf("Name: %s\n", cfg.Name)
	fmt.Printf("Control Plane: %s\n", cfg.ControlPlaneURL)
	fmt.Printf("Hub Endpoint: %s\n", cfg.HubEndpoint)
	fmt.Printf("Local Networks: %v\n", cfg.LocalNetworks)
	fmt.Printf("Config Version: %s\n", loadConfigVersion())
	fmt.Printf("OpenVPN Running: %v\n", isOpenVPNRunning())
	fmt.Printf("OpenVPN Connected: %v\n", isOpenVPNConnected())

	return nil
}

func isOpenVPNRunning() bool {
	cmd := exec.Command("pgrep", "-f", "openvpn.*mesh-hub")
	return cmd.Run() == nil
}

func isOpenVPNConnected() bool {
	// Check for tun interface with the expected IP
	cmd := exec.Command("ip", "addr", "show", "tun0")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "inet ")
}

func startOpenVPN() error {
	cmd := exec.Command("systemctl", "start", "openvpn-client@mesh-hub")
	if err := cmd.Run(); err != nil {
		// Try direct openvpn start
		cmd = exec.Command("openvpn", "--daemon", "--config", "/etc/openvpn/client/mesh-hub.conf")
		return cmd.Run()
	}
	return nil
}

func restartOpenVPN() error {
	// Try systemctl restart first
	cmd := exec.Command("systemctl", "restart", "openvpn-client@mesh-hub")
	if err := cmd.Run(); err != nil {
		// Fall back to killing and restarting manually
		stopCmd := exec.Command("pkill", "-f", "openvpn.*mesh-hub")
		stopCmd.Run() // Ignore error, process might not exist

		// Wait a moment for process to die
		time.Sleep(time.Second)

		// Start again
		startCmd := exec.Command("openvpn", "--daemon", "--config", "/etc/openvpn/client/mesh-hub.conf")
		return startCmd.Run()
	}
	return nil
}

func getPublicIP() string {
	if ip := os.Getenv("PUBLIC_IP"); ip != "" {
		return ip
	}
	// Could query external IP service
	return ""
}

func getBytesSent() int64 {
	// Parse from OpenVPN status or interface stats
	return 0
}

func getBytesReceived() int64 {
	// Parse from OpenVPN status or interface stats
	return 0
}

// isOpenVPN25OrNewer checks if the installed OpenVPN is version 2.5 or newer
// OpenVPN 2.5+ uses data-ciphers, 2.4 uses ncp-ciphers
func isOpenVPN25OrNewer() bool {
	cmd := exec.Command("openvpn", "--version")
	output, err := cmd.Output()
	if err != nil {
		// Default to older syntax if we can't determine version
		return false
	}

	// Parse version from output like "OpenVPN 2.5.1 ..." or "OpenVPN 2.4.7 ..."
	versionStr := string(output)
	if strings.Contains(versionStr, "OpenVPN 2.5") ||
		strings.Contains(versionStr, "OpenVPN 2.6") ||
		strings.Contains(versionStr, "OpenVPN 2.7") ||
		strings.Contains(versionStr, "OpenVPN 3.") {
		return true
	}

	return false
}
