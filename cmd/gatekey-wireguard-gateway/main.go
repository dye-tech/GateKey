// GateKey WireGuard Gateway Agent
// This agent manages a WireGuard VPN gateway for GateKey.
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

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/gatekey-project/gatekey/internal/agent"
	"github.com/gatekey-project/gatekey/internal/firewall"
	"github.com/gatekey-project/gatekey/internal/session"
	"github.com/gatekey-project/gatekey/internal/wireguard"
)

var (
	configPath  string
	logger      *zap.Logger
	firewallMgr *firewall.Manager
	httpClient  = &http.Client{Timeout: 30 * time.Second}
)

// postWithGatewayToken sends a POST request with the X-Gateway-Token header
// to bypass CSRF protection for gateway-to-server communication
func postWithGatewayToken(url string, token string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-Token", token)
	return httpClient.Do(req)
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "gatekey-wireguard-gateway",
		Short: "GateKey WireGuard Gateway Agent",
		Long: `GateKey WireGuard Gateway Agent manages a WireGuard VPN gateway:
- Provisions WireGuard interface configuration from control plane
- Manages authorized peers based on issued configs
- Reports peer statistics to control plane
- Applies per-identity firewall rules`,
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "/etc/gatekey/wireguard-gateway.yaml", "config file path")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the WireGuard gateway agent",
		RunE:  runAgent,
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

// GatewayConfig holds WireGuard gateway agent configuration.
type GatewayConfig struct {
	Name              string        `mapstructure:"name"`
	ControlPlaneURL   string        `mapstructure:"control_plane_url"`
	Token             string        `mapstructure:"token"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	PeerSyncInterval  time.Duration `mapstructure:"peer_sync_interval"`
	StatsSyncInterval time.Duration `mapstructure:"stats_sync_interval"`
	LogLevel          string        `mapstructure:"log_level"`
	InterfaceName     string        `mapstructure:"interface_name"`
	AgentListenAddr   string        `mapstructure:"agent_listen_addr"`
	AgentEnabled      bool          `mapstructure:"agent_enabled"`
	SessionEnabled    bool          `mapstructure:"session_enabled"`
}

// ProvisionResponse is the response from the control plane provision endpoint.
type ProvisionResponse struct {
	GatewayID     string     `json:"gateway_id"`
	GatewayName   string     `json:"gateway_name"`
	PrivateKey    string     `json:"private_key"`
	PublicKey     string     `json:"public_key"`
	ListenPort    int        `json:"listen_port"`
	VPNSubnet     string     `json:"vpn_subnet"`
	InterfaceName string     `json:"interface_name"`
	Peers         []PeerInfo `json:"peers"`
}

// PeerInfo contains information about an authorized peer.
type PeerInfo struct {
	PublicKey    string   `json:"public_key"`
	PresharedKey string   `json:"preshared_key,omitempty"`
	AllowedIPs   []string `json:"allowed_ips"`
	UserID       string   `json:"user_id"`
	ConfigID     string   `json:"config_id"`
	ExpiresAt    string   `json:"expires_at"`
}

// SyncPeersResponse is the response from peer sync endpoint.
type SyncPeersResponse struct {
	Peers []PeerInfo `json:"peers"`
	Time  string     `json:"time"`
}

// ActivePeer tracks an active peer connection locally.
type ActivePeer struct {
	PublicKey  string
	UserID     string
	ConfigID   string
	AllowedIPs []string
}

// Global state
var (
	gatewayID     string
	gatewayName   string
	privateKey    string
	publicKey     string
	listenPort    int
	vpnSubnet     string
	interfaceName string
	activePeers   map[string]*ActivePeer // publicKey -> peer
	interfaceMgr  *wireguard.InterfaceManager
)

func loadConfig() (*GatewayConfig, error) {
	v := viper.New()
	v.SetConfigFile(configPath)

	v.SetDefault("heartbeat_interval", "30s")
	v.SetDefault("peer_sync_interval", "10s")
	v.SetDefault("stats_sync_interval", "5s")
	v.SetDefault("log_level", "info")
	v.SetDefault("interface_name", "wg0")
	v.SetDefault("agent_listen_addr", ":9444")
	v.SetDefault("agent_enabled", true)
	v.SetDefault("session_enabled", true)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	v.SetEnvPrefix("GATEKEY_WG")
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

	logger.Info("Starting GateKey WireGuard Gateway Agent",
		zap.String("control_plane", cfg.ControlPlaneURL))

	// Initialize active peers map
	activePeers = make(map[string]*ActivePeer)

	// Set interface name
	interfaceName = cfg.InterfaceName
	if interfaceName == "" {
		interfaceName = "wg0"
	}

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

	// Provision from control plane
	if err := provision(ctx, cfg); err != nil {
		return fmt.Errorf("failed to provision: %w", err)
	}

	// Setup WireGuard interface
	if err := setupInterface(ctx, cfg); err != nil {
		return fmt.Errorf("failed to setup WireGuard interface: %w", err)
	}

	// Start agent API server for remote tool execution
	var agentServer *agent.Server
	nodeName := cfg.Name
	if nodeName == "" {
		if h, err := os.Hostname(); err == nil {
			nodeName = h
		} else {
			nodeName = "wireguard-gateway"
		}
	}
	if cfg.AgentEnabled {
		agentServer = agent.NewServer(&agent.Config{
			ListenAddr: cfg.AgentListenAddr,
			APIToken:   cfg.Token,
			NodeType:   "wireguard-gateway",
			NodeName:   nodeName,
			Logger:     logger,
		})
		// Register trigger-sync handler for immediate peer sync from control plane
		agentServer.SetTriggerSyncHandler(func(ctx context.Context) error {
			logger.Info("Immediate peer sync triggered by control plane")
			syncPeers(ctx, cfg)
			return nil
		})
		go func() {
			logger.Info("Starting agent API server",
				zap.String("addr", cfg.AgentListenAddr))
			if err := agentServer.Start(); err != nil && err != http.ErrServerClosed {
				logger.Error("Agent API server failed", zap.Error(err))
			}
		}()
	}

	// Start remote session client
	var sessionClient *session.AgentClient
	if cfg.SessionEnabled {
		sessionClient = session.NewAgentClient(&session.AgentClientConfig{
			ControlPlaneURL: cfg.ControlPlaneURL,
			Token:           cfg.Token,
			NodeType:        "wireguard-gateway",
			NodeID:          nodeName,
			NodeName:        nodeName,
			Logger:          logger,
		})
		sessionClient.Start(ctx)
		logger.Info("Remote session client started")
	}

	// Start background loops
	go heartbeatLoop(ctx, cfg)
	go peerSyncLoop(ctx, cfg)
	go statsSyncLoop(ctx, cfg)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down WireGuard gateway agent")

	// Shutdown agent API server
	if agentServer != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := agentServer.Shutdown(shutdownCtx); err != nil {
			logger.Warn("Failed to shutdown agent server", zap.Error(err))
		}
	}

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

	// Cleanup firewall rules
	if firewallMgr != nil {
		if err := firewallMgr.Cleanup(context.Background()); err != nil {
			logger.Warn("Failed to cleanup firewall rules", zap.Error(err))
		}
	}

	return nil
}

// provision fetches configuration from the control plane.
func provision(ctx context.Context, cfg *GatewayConfig) error {
	logger.Info("Provisioning from control plane...")

	reqBody := struct {
		Token string `json:"token"`
	}{
		Token: cfg.Token,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/wireguard-gateway/provision"
	resp, err := postWithGatewayToken(url, cfg.Token, body)
	if err != nil {
		return fmt.Errorf("failed to send provision request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("provision failed: %d - %s", resp.StatusCode, string(respBody))
	}

	var provResp ProvisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&provResp); err != nil {
		return fmt.Errorf("failed to decode provision response: %w", err)
	}

	// Store configuration
	gatewayID = provResp.GatewayID
	gatewayName = provResp.GatewayName
	privateKey = provResp.PrivateKey
	publicKey = provResp.PublicKey
	listenPort = provResp.ListenPort
	vpnSubnet = provResp.VPNSubnet
	if provResp.InterfaceName != "" {
		interfaceName = provResp.InterfaceName
	}

	logger.Info("Provisioned successfully",
		zap.String("gateway_id", gatewayID),
		zap.String("gateway_name", gatewayName),
		zap.Int("listen_port", listenPort),
		zap.String("vpn_subnet", vpnSubnet),
		zap.Int("initial_peers", len(provResp.Peers)))

	// Store initial peers
	for _, peer := range provResp.Peers {
		activePeers[peer.PublicKey] = &ActivePeer{
			PublicKey:  peer.PublicKey,
			UserID:     peer.UserID,
			ConfigID:   peer.ConfigID,
			AllowedIPs: peer.AllowedIPs,
		}
	}

	return nil
}

// setupInterface creates and configures the WireGuard interface.
func setupInterface(ctx context.Context, cfg *GatewayConfig) error {
	logger.Info("Setting up WireGuard interface",
		zap.String("interface", interfaceName))

	// Calculate gateway address from VPN subnet (first usable IP)
	gatewayAddr, err := getGatewayAddress(vpnSubnet)
	if err != nil {
		return fmt.Errorf("failed to get gateway address: %w", err)
	}

	// Create interface manager
	interfaceMgr = wireguard.NewInterfaceManager(interfaceName, logger)

	// Setup the interface
	config := wireguard.InterfaceConfig{
		PrivateKey: privateKey,
		Address:    gatewayAddr,
		ListenPort: listenPort,
	}

	if err := interfaceMgr.Setup(ctx, config); err != nil {
		return fmt.Errorf("failed to setup interface: %w", err)
	}

	// Add initial peers
	for _, peer := range activePeers {
		peerConfig := wireguard.PeerConfig{
			PublicKey:  peer.PublicKey,
			AllowedIPs: peer.AllowedIPs,
		}
		if err := interfaceMgr.AddPeer(ctx, peerConfig); err != nil {
			logger.Warn("Failed to add initial peer",
				zap.String("public_key", peer.PublicKey[:8]),
				zap.Error(err))
		}
	}

	logger.Info("WireGuard interface setup complete",
		zap.String("address", gatewayAddr),
		zap.Int("port", listenPort),
		zap.Int("peer_count", len(activePeers)))

	return nil
}

// getGatewayAddress returns the gateway's IP address from the VPN subnet.
func getGatewayAddress(subnet string) (string, error) {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return "", fmt.Errorf("invalid subnet: %w", err)
	}

	// Get the first usable IP (network address + 1)
	ip := ipNet.IP.To4()
	if ip == nil {
		return "", fmt.Errorf("only IPv4 subnets are supported")
	}

	// Increment the last octet
	gatewayIP := net.IP(make([]byte, 4))
	copy(gatewayIP, ip)
	gatewayIP[3]++

	// Get the mask size for CIDR notation
	ones, _ := ipNet.Mask.Size()

	return fmt.Sprintf("%s/%d", gatewayIP.String(), ones), nil
}

// heartbeatLoop sends periodic heartbeats to the control plane.
func heartbeatLoop(ctx context.Context, cfg *GatewayConfig) {
	ticker := time.NewTicker(cfg.HeartbeatInterval)
	defer ticker.Stop()

	// Send initial heartbeat
	sendHeartbeat(cfg)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendHeartbeat(cfg)
		}
	}
}

func sendHeartbeat(cfg *GatewayConfig) {
	reqBody := struct {
		Token     string `json:"token"`
		PeerCount int    `json:"peer_count"`
		Version   string `json:"version"`
		Uptime    int64  `json:"uptime"`
	}{
		Token:     cfg.Token,
		PeerCount: len(activePeers),
		Version:   "1.0.0",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		logger.Debug("Failed to marshal heartbeat", zap.Error(err))
		return
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/wireguard-gateway/heartbeat"
	resp, err := postWithGatewayToken(url, cfg.Token, body)
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
	} else {
		logger.Debug("Heartbeat sent successfully", zap.Int("peer_count", len(activePeers)))
	}
}

// peerSyncLoop periodically syncs authorized peers from the control plane.
func peerSyncLoop(ctx context.Context, cfg *GatewayConfig) {
	ticker := time.NewTicker(cfg.PeerSyncInterval)
	defer ticker.Stop()

	// Perform initial sync immediately
	syncPeers(ctx, cfg)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncPeers(ctx, cfg)
		}
	}
}

func syncPeers(ctx context.Context, cfg *GatewayConfig) {
	reqBody := struct {
		Token string `json:"token"`
	}{
		Token: cfg.Token,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		logger.Debug("Failed to marshal sync request", zap.Error(err))
		return
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/wireguard-gateway/sync-peers"
	resp, err := postWithGatewayToken(url, cfg.Token, body)
	if err != nil {
		logger.Warn("Peer sync failed", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Warn("Peer sync returned error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)))
		return
	}

	var syncResp SyncPeersResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		logger.Warn("Failed to decode sync response", zap.Error(err))
		return
	}

	// Build map of authorized peers
	authorizedPeers := make(map[string]*PeerInfo)
	for i := range syncResp.Peers {
		authorizedPeers[syncResp.Peers[i].PublicKey] = &syncResp.Peers[i]
	}

	// Add new peers
	for pubKey, peerInfo := range authorizedPeers {
		if _, exists := activePeers[pubKey]; !exists {
			// New peer - add to WireGuard
			if interfaceMgr != nil {
				peerConfig := wireguard.PeerConfig{
					PublicKey:    peerInfo.PublicKey,
					PresharedKey: peerInfo.PresharedKey,
					AllowedIPs:   peerInfo.AllowedIPs,
				}
				if err := interfaceMgr.AddPeer(ctx, peerConfig); err != nil {
					logger.Warn("Failed to add peer",
						zap.String("public_key", pubKey[:8]),
						zap.Error(err))
					continue
				}
			}

			activePeers[pubKey] = &ActivePeer{
				PublicKey:  peerInfo.PublicKey,
				UserID:     peerInfo.UserID,
				ConfigID:   peerInfo.ConfigID,
				AllowedIPs: peerInfo.AllowedIPs,
			}

			logger.Info("Added new peer",
				zap.String("public_key", pubKey[:8]),
				zap.String("user_id", peerInfo.UserID))
		}
	}

	// Remove revoked/expired peers
	for pubKey, peer := range activePeers {
		if _, exists := authorizedPeers[pubKey]; !exists {
			// Peer no longer authorized - remove from WireGuard
			if interfaceMgr != nil {
				if err := interfaceMgr.RemovePeer(ctx, pubKey); err != nil {
					logger.Warn("Failed to remove peer",
						zap.String("public_key", pubKey[:8]),
						zap.Error(err))
				}
			}

			// Remove firewall rules
			if firewallMgr != nil {
				connectionID := fmt.Sprintf("wg-%s", strings.ReplaceAll(pubKey[:16], "/", "-"))
				if err := firewallMgr.RemoveRules(ctx, connectionID); err != nil {
					logger.Debug("Failed to remove firewall rules",
						zap.String("public_key", pubKey[:8]),
						zap.Error(err))
				}
			}

			delete(activePeers, pubKey)

			logger.Info("Removed peer",
				zap.String("public_key", pubKey[:8]),
				zap.String("user_id", peer.UserID))
		}
	}

	logger.Debug("Peer sync completed", zap.Int("authorized_peers", len(authorizedPeers)), zap.Int("active_peers", len(activePeers)))
}

// statsSyncLoop periodically reports peer statistics to the control plane.
func statsSyncLoop(ctx context.Context, cfg *GatewayConfig) {
	ticker := time.NewTicker(cfg.StatsSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncStats(ctx, cfg)
		}
	}
}

func syncStats(ctx context.Context, cfg *GatewayConfig) {
	// Get current peer stats from WireGuard
	peerStats, err := getWireGuardPeerStats()
	if err != nil {
		logger.Debug("Failed to get peer stats", zap.Error(err))
		return
	}

	if len(peerStats) == 0 {
		return
	}

	type peerStatEntry struct {
		PublicKey     string `json:"public_key"`
		Endpoint      string `json:"endpoint"`
		LastHandshake int64  `json:"last_handshake"`
		BytesSent     int64  `json:"bytes_sent"`
		BytesReceived int64  `json:"bytes_received"`
	}

	var peers []peerStatEntry
	for pubKey, stats := range peerStats {
		peers = append(peers, peerStatEntry{
			PublicKey:     pubKey,
			Endpoint:      stats.Endpoint,
			LastHandshake: stats.LastHandshake.Unix(),
			BytesSent:     stats.BytesSent,
			BytesReceived: stats.BytesReceived,
		})
	}

	reqBody := struct {
		Token string          `json:"token"`
		Peers []peerStatEntry `json:"peers"`
	}{
		Token: cfg.Token,
		Peers: peers,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		logger.Debug("Failed to marshal stats", zap.Error(err))
		return
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/wireguard-gateway/peer-stats"
	resp, err := postWithGatewayToken(url, cfg.Token, body)
	if err != nil {
		logger.Debug("Failed to send stats", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Debug("Stats sync returned error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)))
	}
}

// PeerStats contains statistics for a WireGuard peer.
type PeerStats struct {
	Endpoint      string
	LastHandshake time.Time
	BytesSent     int64
	BytesReceived int64
}

// getWireGuardPeerStats reads peer statistics from wg show.
func getWireGuardPeerStats() (map[string]*PeerStats, error) {
	stats := make(map[string]*PeerStats)

	// Run wg show to get peer stats
	cmd := exec.Command("wg", "show", interfaceName, "dump")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run wg show: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip interface line and empty lines
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 8 {
			continue
		}

		pubKey := fields[0]
		endpoint := fields[3]
		lastHandshake, _ := strconv.ParseInt(fields[5], 10, 64)
		rxBytes, _ := strconv.ParseInt(fields[6], 10, 64)
		txBytes, _ := strconv.ParseInt(fields[7], 10, 64)

		var handshakeTime time.Time
		if lastHandshake > 0 {
			handshakeTime = time.Unix(lastHandshake, 0)
		}

		stats[pubKey] = &PeerStats{
			Endpoint:      endpoint,
			LastHandshake: handshakeTime,
			BytesSent:     txBytes,
			BytesReceived: rxBytes,
		}
	}

	return stats, nil
}

// applyPeerFirewallRules applies firewall rules for a connected peer.
func applyPeerFirewallRules(ctx context.Context, peer *ActivePeer, clientIP string) error {
	if firewallMgr == nil {
		return nil
	}

	// Build networks from AllowedIPs
	var networks []net.IPNet
	for _, cidr := range peer.AllowedIPs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			// Try as single IP
			ip := net.ParseIP(strings.TrimSuffix(cidr, "/32"))
			if ip != nil {
				networks = append(networks, net.IPNet{
					IP:   ip,
					Mask: net.CIDRMask(32, 32),
				})
			}
			continue
		}
		networks = append(networks, *ipNet)
	}

	// Parse client IP
	sourceIP := net.ParseIP(clientIP)
	if sourceIP == nil {
		return fmt.Errorf("invalid client IP: %s", clientIP)
	}

	// Parse user ID
	uid, err := uuid.Parse(peer.UserID)
	if err != nil {
		uid = uuid.New()
	}

	connectionID := fmt.Sprintf("wg-%s", strings.ReplaceAll(peer.PublicKey[:16], "/", "-"))

	return firewallMgr.ApplyRules(ctx, connectionID, uid, sourceIP, networks, nil)
}

// provisionGateway handles the provision command to fetch config from control plane.
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
	if err := provision(ctx, cfg); err != nil {
		return fmt.Errorf("provision failed: %w", err)
	}

	fmt.Println("Gateway provisioned successfully")
	fmt.Printf("  Gateway ID:   %s\n", gatewayID)
	fmt.Printf("  Gateway Name: %s\n", gatewayName)
	fmt.Printf("  Listen Port:  %d\n", listenPort)
	fmt.Printf("  VPN Subnet:   %s\n", vpnSubnet)
	fmt.Printf("  Interface:    %s\n", interfaceName)
	fmt.Printf("  Initial Peers: %d\n", len(activePeers))

	return nil
}

// showStatus handles the status command to display gateway status.
func showStatus(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	fmt.Printf("GateKey WireGuard Gateway Status\n")
	fmt.Printf("================================\n")
	fmt.Printf("Name: %s\n", cfg.Name)
	fmt.Printf("Control Plane: %s\n", cfg.ControlPlaneURL)
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

	// Get peer count from wg show dump
	dumpOutput, err := exec.Command("wg", "show", ifName, "dump").Output()
	if err == nil {
		lines := strings.Split(string(dumpOutput), "\n")
		peerCount := 0
		activePeerCount := 0
		for i, line := range lines {
			if i == 0 || line == "" {
				continue
			}
			peerCount++
			// Check if peer has recent handshake
			fields := strings.Split(line, "\t")
			if len(fields) >= 6 {
				lastHandshake, _ := strconv.ParseInt(fields[5], 10, 64)
				if lastHandshake > 0 {
					handshakeTime := time.Unix(lastHandshake, 0)
					if time.Since(handshakeTime) < 3*time.Minute {
						activePeerCount++
					}
				}
			}
		}
		fmt.Printf("Total Peers: %d\n", peerCount)
		fmt.Printf("Active Peers (recent handshake): %d\n", activePeerCount)
	}

	fmt.Printf("\n%s", string(output))

	return nil
}
