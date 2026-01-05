// GateKey WireGuard Mesh Gateway (Spoke)
// This binary runs on remote gateway sites and connects TO the mesh hub using WireGuard.
// It advertises local networks to the hub and enables hub-and-spoke mesh connectivity.
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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/session"
	"github.com/gatekey-project/gatekey/internal/wireguard"
)

var (
	configPath       string
	logger           *zap.Logger
	currentConfigVer string
	provisionedName  string // Name from control plane provisioning
)

const (
	configVersionFile = "/etc/gatekey-wireguard-mesh/.config_version"
	gatewayNameFile   = "/etc/gatekey-wireguard-mesh/.gateway_name"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gatekey-wireguard-mesh-gateway",
		Short: "GateKey WireGuard Mesh Gateway (Spoke) Agent",
		Long: `GateKey WireGuard Mesh Gateway Agent runs on remote sites and connects TO the mesh hub.
It uses WireGuard to establish a connection back to the hub, allowing hub-and-spoke
mesh networking without requiring public IPs on gateway sites.

The gateway:
- Connects to the hub's WireGuard server as a peer
- Advertises local networks to the hub
- Enables routing between hub clients and local resources
- Uses persistent keepalive for NAT traversal`,
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "/etc/gatekey-wireguard-mesh/config.yaml", "config file path")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the WireGuard mesh gateway agent",
		RunE:  runGateway,
	}

	provisionCmd := &cobra.Command{
		Use:   "provision",
		Short: "Provision configuration from control plane",
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
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	LogLevel          string        `mapstructure:"log_level"`
	InterfaceName     string        `mapstructure:"interface_name"`
	SessionEnabled    bool          `mapstructure:"session_enabled"`
}

// ProvisionResponse from control plane
type ProvisionResponse struct {
	SpokeID        string   `json:"spoke_id"`
	SpokeName      string   `json:"spoke_name"`
	PrivateKey     string   `json:"private_key"`
	PublicKey      string   `json:"public_key"`
	PresharedKey   string   `json:"preshared_key,omitempty"`
	TunnelIP       string   `json:"tunnel_ip"`       // Spoke's assigned IP address
	HubEndpoint    string   `json:"hub_endpoint"`    // Hub's public endpoint (host:port)
	HubPublicKey   string   `json:"hub_public_key"`  // Hub's WireGuard public key
	HubAllowedIPs  []string `json:"hub_allowed_ips"` // Networks to route through hub
	LocalNetworks  []string `json:"local_networks"`  // Spoke's advertised local networks
	FullTunnel     bool     `json:"full_tunnel"`     // Route all traffic through hub
	DNSServers     []string `json:"dns_servers"`     // DNS servers to use
	ConfigVersion  string   `json:"config_version"`
	PersistentKA   int      `json:"persistent_keepalive"` // Keepalive interval (default 25s)
}

// Global state
var (
	spokeID       string
	spokeName     string
	privateKey    string
	publicKey     string
	presharedKey  string
	tunnelIP      string
	hubEndpoint   string
	hubPublicKey  string
	hubAllowedIPs []string
	localNetworks []string
	interfaceName string
	interfaceMgr  *wireguard.InterfaceManager
	persistentKA  int
)

func loadConfig() (*GatewayConfig, error) {
	v := viper.New()
	v.SetConfigFile(configPath)

	v.SetDefault("heartbeat_interval", "30s")
	v.SetDefault("log_level", "info")
	v.SetDefault("interface_name", "wg0")
	v.SetDefault("session_enabled", true)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	v.SetEnvPrefix("GATEKEY_WG_MESH")
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
	dir := "/etc/gatekey-wireguard-mesh"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
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
	dir := "/etc/gatekey-wireguard-mesh"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
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

	logger.Info("Starting GateKey WireGuard Mesh Gateway",
		zap.String("name", cfg.Name),
		zap.String("control_plane", cfg.ControlPlaneURL),
	)

	// Set interface name
	interfaceName = cfg.InterfaceName
	if interfaceName == "" {
		interfaceName = "wg0"
	}

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

	// Setup WireGuard interface
	if err := setupInterface(ctx); err != nil {
		logger.Error("Failed to setup WireGuard interface", zap.Error(err))
		return fmt.Errorf("failed to setup interface: %w", err)
	}

	// Start remote session client (connects outbound to control plane)
	var sessionClient *session.AgentClient
	if cfg.SessionEnabled && effectiveName != "" {
		sessionClient = session.NewAgentClient(&session.AgentClientConfig{
			ControlPlaneURL: cfg.ControlPlaneURL,
			Token:           cfg.GatewayToken,
			NodeType:        "wireguard-spoke",
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

	logger.Info("Shutting down WireGuard mesh gateway")

	// Stop session client
	if sessionClient != nil {
		sessionClient.Stop()
	}

	// Cleanup WireGuard interface
	if interfaceMgr != nil {
		if err := interfaceMgr.Teardown(context.Background()); err != nil {
			logger.Warn("Failed to teardown WireGuard interface", zap.Error(err))
		}
	}

	return nil
}

// HeartbeatResponse from control plane
type HeartbeatResponse struct {
	OK               bool   `json:"ok"`
	ConfigVersion    string `json:"configVersion"`
	NeedsReprovision bool   `json:"needsReprovision"`
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

func sendHeartbeat(ctx context.Context, cfg *GatewayConfig) {
	status := "disconnected"
	if isWireGuardConnected() {
		status = "connected"
	}

	// Get traffic stats
	bytesSent, bytesReceived := getTrafficStats()

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
		BytesSent:     bytesSent,
		BytesReceived: bytesReceived,
		ConfigVersion: currentConfigVer,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		logger.Warn("Failed to marshal heartbeat", zap.Error(err))
		return
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/wg-mesh-spoke/heartbeat"
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

		// Reconfigure WireGuard interface with new settings
		logger.Info("Reconfiguring WireGuard interface with new configuration...")
		if interfaceMgr != nil {
			// Teardown existing interface
			if err := interfaceMgr.Teardown(ctx); err != nil {
				logger.Warn("Failed to teardown existing interface", zap.Error(err))
			}
		}

		// Setup new interface
		if err := setupInterface(ctx); err != nil {
			logger.Error("Failed to setup new interface", zap.Error(err))
		} else {
			logger.Info("WireGuard interface reconfigured successfully")
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
			// Check if WireGuard is connected via handshake
			if !isWireGuardConnected() {
				logger.Warn("WireGuard not connected to hub, checking...")
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

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/wg-mesh-spoke/provision"
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

	// Store configuration in global state
	spokeID = provResp.SpokeID
	spokeName = provResp.SpokeName
	privateKey = provResp.PrivateKey
	publicKey = provResp.PublicKey
	presharedKey = provResp.PresharedKey
	tunnelIP = provResp.TunnelIP
	hubEndpoint = provResp.HubEndpoint
	hubPublicKey = provResp.HubPublicKey
	hubAllowedIPs = provResp.HubAllowedIPs
	localNetworks = provResp.LocalNetworks
	persistentKA = provResp.PersistentKA
	if persistentKA == 0 {
		persistentKA = 25 // Default keepalive for NAT traversal
	}

	// Save config version
	currentConfigVer = provResp.ConfigVersion
	if currentConfigVer == "" {
		currentConfigVer = provResp.SpokeID
	}
	if err := saveConfigVersion(currentConfigVer); err != nil {
		logger.Warn("Failed to save config version", zap.Error(err))
	}

	// Save gateway name for session authentication
	if provResp.SpokeName != "" {
		provisionedName = provResp.SpokeName
		if err := saveGatewayName(provResp.SpokeName); err != nil {
			logger.Warn("Failed to save gateway name", zap.Error(err))
		}
		logger.Info("Gateway name saved from provisioning", zap.String("name", provResp.SpokeName))
	}

	logger.Info("Gateway provisioned successfully",
		zap.String("spoke_id", spokeID),
		zap.String("spoke_name", spokeName),
		zap.String("tunnel_ip", tunnelIP),
		zap.String("hub_endpoint", hubEndpoint),
		zap.Strings("local_networks", localNetworks),
		zap.String("config_version", currentConfigVer),
	)

	return nil
}

func setupInterface(ctx context.Context) error {
	logger.Info("Setting up WireGuard interface",
		zap.String("interface", interfaceName),
		zap.String("tunnel_ip", tunnelIP),
		zap.String("hub_endpoint", hubEndpoint))

	// Create interface manager
	interfaceMgr = wireguard.NewInterfaceManager(interfaceName, logger)

	// Parse tunnel IP to add proper CIDR notation
	tunnelAddr := tunnelIP
	if !strings.Contains(tunnelAddr, "/") {
		tunnelAddr = tunnelAddr + "/32" // Use /32 for point-to-point
	}

	// Setup the interface
	config := wireguard.InterfaceConfig{
		PrivateKey: privateKey,
		Address:    tunnelAddr,
		ListenPort: 0, // Random port for client mode
	}

	if err := interfaceMgr.Setup(ctx, config); err != nil {
		return fmt.Errorf("failed to setup interface: %w", err)
	}

	// Add hub as peer
	peerConfig := wireguard.PeerConfig{
		PublicKey:           hubPublicKey,
		PresharedKey:        presharedKey,
		Endpoint:            hubEndpoint,
		AllowedIPs:          hubAllowedIPs,
		PersistentKeepalive: persistentKA,
	}

	if err := interfaceMgr.AddPeer(ctx, peerConfig); err != nil {
		return fmt.Errorf("failed to add hub peer: %w", err)
	}

	// Setup routes for hub networks
	if err := setupRoutes(ctx); err != nil {
		logger.Warn("Failed to setup some routes", zap.Error(err))
	}

	logger.Info("WireGuard interface setup complete",
		zap.String("address", tunnelAddr),
		zap.String("hub", hubEndpoint),
		zap.Int("keepalive", persistentKA))

	return nil
}

func setupRoutes(ctx context.Context) error {
	// Add routes for hub's allowed IPs through the WireGuard interface
	for _, network := range hubAllowedIPs {
		// Skip default route (handled by WireGuard AllowedIPs)
		if network == "0.0.0.0/0" {
			continue
		}

		// Add route via WireGuard interface
		cmd := exec.CommandContext(ctx, "ip", "route", "add", network, "dev", interfaceName)
		if err := cmd.Run(); err != nil {
			// Check if route already exists
			if !strings.Contains(err.Error(), "File exists") {
				logger.Debug("Failed to add route",
					zap.String("network", network),
					zap.Error(err))
			}
		}
	}

	return nil
}

func showStatus(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	fmt.Printf("GateKey WireGuard Mesh Gateway Status\n")
	fmt.Printf("======================================\n")
	fmt.Printf("Name: %s\n", cfg.Name)
	fmt.Printf("Control Plane: %s\n", cfg.ControlPlaneURL)
	fmt.Printf("Config Version: %s\n", loadConfigVersion())
	fmt.Printf("Interface: %s\n", cfg.InterfaceName)

	ifName := cfg.InterfaceName
	if ifName == "" {
		ifName = "wg0"
	}

	// Check WireGuard interface
	output, err := exec.Command("wg", "show", ifName).Output()
	if err != nil {
		fmt.Println("WireGuard Status: NOT RUNNING")
		return nil
	}

	fmt.Printf("WireGuard Status: RUNNING\n")
	fmt.Printf("Connected to Hub: %v\n", isWireGuardConnected())
	fmt.Printf("\n%s", string(output))

	return nil
}

// isWireGuardConnected checks if the hub peer has had a recent handshake
func isWireGuardConnected() bool {
	if hubPublicKey == "" {
		return false
	}

	// Run wg show to get peer info
	cmd := exec.Command("wg", "show", interfaceName, "dump")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip interface line and empty lines
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 6 {
			continue
		}

		pubKey := fields[0]
		if pubKey == hubPublicKey {
			// Check last handshake time
			lastHandshake, _ := strconv.ParseInt(fields[5], 10, 64)
			if lastHandshake > 0 {
				// Consider connected if handshake was within last 3 minutes
				handshakeTime := time.Unix(lastHandshake, 0)
				return time.Since(handshakeTime) < 3*time.Minute
			}
		}
	}

	return false
}

// getTrafficStats returns bytes sent and received for the hub peer
func getTrafficStats() (int64, int64) {
	if hubPublicKey == "" {
		return 0, 0
	}

	cmd := exec.Command("wg", "show", interfaceName, "dump")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 8 {
			continue
		}

		pubKey := fields[0]
		if pubKey == hubPublicKey {
			rxBytes, _ := strconv.ParseInt(fields[6], 10, 64)
			txBytes, _ := strconv.ParseInt(fields[7], 10, 64)
			return txBytes, rxBytes
		}
	}

	return 0, 0
}

func getPublicIP() string {
	if ip := os.Getenv("PUBLIC_IP"); ip != "" {
		return ip
	}

	// Try to get public IP from interface
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() || ip.IsPrivate() {
				continue
			}

			return ip.String()
		}
	}

	return ""
}
