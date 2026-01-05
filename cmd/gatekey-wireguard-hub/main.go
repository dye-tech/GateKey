// GateKey WireGuard Mesh Hub
// This is a standalone hub server that runs WireGuard and connects mesh spokes.
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
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gatekey-wireguard-hub",
		Short: "GateKey WireGuard Mesh Hub Server",
		Long: `GateKey WireGuard Mesh Hub Server runs WireGuard and accepts connections from:
- Mesh Spokes (remote sites connecting back to hub)
- VPN Clients (users connecting to access mesh resources)

The hub communicates with the GateKey control plane to:
- Receive configuration updates
- Get route information from connected spokes
- Report connection status and health`,
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "/etc/gatekey-wireguard-hub/config.yaml", "config file path")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the WireGuard mesh hub server",
		RunE:  runHub,
	}

	provisionCmd := &cobra.Command{
		Use:   "provision",
		Short: "Provision configuration from control plane",
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

// HubConfig holds WireGuard mesh hub configuration
type HubConfig struct {
	Name              string        `mapstructure:"name"`
	ControlPlaneURL   string        `mapstructure:"control_plane_url"`
	APIToken          string        `mapstructure:"api_token"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	PeerSyncInterval  time.Duration `mapstructure:"peer_sync_interval"`
	StatsSyncInterval time.Duration `mapstructure:"stats_sync_interval"`
	LogLevel          string        `mapstructure:"log_level"`
	InterfaceName     string        `mapstructure:"interface_name"`
	AgentListenAddr   string        `mapstructure:"agent_listen_addr"`
	AgentEnabled      bool          `mapstructure:"agent_enabled"`
	SessionEnabled    bool          `mapstructure:"session_enabled"`
}

// ProvisionResponse from control plane
type ProvisionResponse struct {
	HubID         string     `json:"hub_id"`
	HubName       string     `json:"hub_name"`
	PrivateKey    string     `json:"private_key"`
	PublicKey     string     `json:"public_key"`
	ListenPort    int        `json:"listen_port"`
	VPNSubnet     string     `json:"vpn_subnet"`
	InterfaceName string     `json:"interface_name"`
	Peers         []PeerInfo `json:"peers"`
	FullTunnel    bool       `json:"full_tunnel"`
	PushDNS       bool       `json:"push_dns"`
	DNSServers    []string   `json:"dns_servers"`
}

// PeerInfo contains information about an authorized spoke peer
type PeerInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	PublicKey    string   `json:"public_key"`
	PresharedKey string   `json:"preshared_key,omitempty"`
	AllowedIPs   []string `json:"allowed_ips"`
	TunnelIP     string   `json:"tunnel_ip,omitempty"`
}

// SyncPeersResponse is the response from peer sync endpoint
type SyncPeersResponse struct {
	Peers []PeerInfo `json:"peers"`
	Time  string     `json:"time"`
}

// ActivePeer tracks an active spoke connection locally
type ActivePeer struct {
	ID         string
	Name       string
	PublicKey  string
	AllowedIPs []string
	TunnelIP   string
}

// Global state
var (
	hubID         string
	hubName       string
	privateKey    string
	publicKey     string
	listenPort    int
	vpnSubnet     string
	interfaceName string
	activePeers   map[string]*ActivePeer // publicKey -> peer
	interfaceMgr  *wireguard.InterfaceManager
)

func loadConfig() (*HubConfig, error) {
	v := viper.New()
	v.SetConfigFile(configPath)

	v.SetDefault("heartbeat_interval", "30s")
	v.SetDefault("peer_sync_interval", "10s")
	v.SetDefault("stats_sync_interval", "5s")
	v.SetDefault("log_level", "info")
	v.SetDefault("interface_name", "wg0")
	v.SetDefault("agent_listen_addr", ":9445")
	v.SetDefault("agent_enabled", true)
	v.SetDefault("session_enabled", true)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	v.SetEnvPrefix("GATEKEY_WG_HUB")
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

	logger.Info("Starting GateKey WireGuard Mesh Hub",
		zap.String("name", cfg.Name),
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
			nodeName = "wireguard-hub"
		}
	}
	if cfg.AgentEnabled {
		agentServer = agent.NewServer(&agent.Config{
			ListenAddr: cfg.AgentListenAddr,
			APIToken:   cfg.APIToken,
			NodeType:   "wireguard-hub",
			NodeName:   nodeName,
			Logger:     logger,
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
			Token:           cfg.APIToken,
			NodeType:        "wireguard-hub",
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

	logger.Info("Shutting down WireGuard mesh hub")

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

func showStatus(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	fmt.Printf("WireGuard Mesh Hub: %s\n", cfg.Name)
	fmt.Printf("Control Plane: %s\n", cfg.ControlPlaneURL)
	fmt.Printf("Interface: %s\n", cfg.InterfaceName)

	// Check WireGuard interface
	output, err := exec.Command("wg", "show", cfg.InterfaceName).Output()
	if err != nil {
		fmt.Println("WireGuard interface: NOT RUNNING")
		return nil
	}

	fmt.Printf("WireGuard interface: RUNNING\n%s", string(output))
	return nil
}

// provision fetches configuration from the control plane
func provision(ctx context.Context, cfg *HubConfig) error {
	logger.Info("Provisioning from control plane...")

	reqBody := struct {
		Token string `json:"token"`
	}{
		Token: cfg.APIToken,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/wg-mesh-hub/provision"
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
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
	hubID = provResp.HubID
	hubName = provResp.HubName
	privateKey = provResp.PrivateKey
	publicKey = provResp.PublicKey
	listenPort = provResp.ListenPort
	vpnSubnet = provResp.VPNSubnet
	if provResp.InterfaceName != "" {
		interfaceName = provResp.InterfaceName
	}

	logger.Info("Provisioned successfully",
		zap.String("hub_id", hubID),
		zap.String("hub_name", hubName),
		zap.Int("listen_port", listenPort),
		zap.String("vpn_subnet", vpnSubnet),
		zap.Int("initial_peers", len(provResp.Peers)))

	// Store initial peers (spokes)
	for _, peer := range provResp.Peers {
		activePeers[peer.PublicKey] = &ActivePeer{
			ID:         peer.ID,
			Name:       peer.Name,
			PublicKey:  peer.PublicKey,
			AllowedIPs: peer.AllowedIPs,
			TunnelIP:   peer.TunnelIP,
		}
	}

	return nil
}

// setupInterface creates and configures the WireGuard interface
func setupInterface(ctx context.Context, cfg *HubConfig) error {
	logger.Info("Setting up WireGuard interface",
		zap.String("interface", interfaceName))

	// Calculate hub address from VPN subnet (first usable IP)
	hubAddr, err := getHubAddress(vpnSubnet)
	if err != nil {
		return fmt.Errorf("failed to get hub address: %w", err)
	}

	// Create interface manager
	interfaceMgr = wireguard.NewInterfaceManager(interfaceName, logger)

	// Setup the interface
	config := wireguard.InterfaceConfig{
		PrivateKey: privateKey,
		Address:    hubAddr,
		ListenPort: listenPort,
	}

	if err := interfaceMgr.Setup(ctx, config); err != nil {
		return fmt.Errorf("failed to setup interface: %w", err)
	}

	// Add initial peers (spokes)
	for _, peer := range activePeers {
		peerConfig := wireguard.PeerConfig{
			PublicKey:  peer.PublicKey,
			AllowedIPs: peer.AllowedIPs,
		}
		if err := interfaceMgr.AddPeer(ctx, peerConfig); err != nil {
			logger.Warn("Failed to add initial peer",
				zap.String("name", peer.Name),
				zap.String("public_key", peer.PublicKey[:8]),
				zap.Error(err))
		}
	}

	logger.Info("WireGuard interface setup complete",
		zap.String("address", hubAddr),
		zap.Int("port", listenPort),
		zap.Int("peer_count", len(activePeers)))

	return nil
}

// getHubAddress returns the hub's IP address from the VPN subnet
func getHubAddress(subnet string) (string, error) {
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
	hubIP := net.IP(make([]byte, 4))
	copy(hubIP, ip)
	hubIP[3]++

	// Get the mask size for CIDR notation
	ones, _ := ipNet.Mask.Size()

	return fmt.Sprintf("%s/%d", hubIP.String(), ones), nil
}

// heartbeatLoop sends periodic heartbeats to the control plane
func heartbeatLoop(ctx context.Context, cfg *HubConfig) {
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

func sendHeartbeat(cfg *HubConfig) {
	reqBody := struct {
		Token            string `json:"token"`
		PeerCount        int    `json:"peer_count"`
		ConnectedClients int    `json:"connected_clients"`
		Version          string `json:"version"`
		Uptime           int64  `json:"uptime"`
	}{
		Token:     cfg.APIToken,
		PeerCount: len(activePeers),
		Version:   "1.0.0",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		logger.Debug("Failed to marshal heartbeat", zap.Error(err))
		return
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/wg-mesh-hub/heartbeat"
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
	}
}

// peerSyncLoop periodically syncs authorized spokes from the control plane
func peerSyncLoop(ctx context.Context, cfg *HubConfig) {
	ticker := time.NewTicker(cfg.PeerSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncPeers(ctx, cfg)
		}
	}
}

func syncPeers(ctx context.Context, cfg *HubConfig) {
	reqBody := struct {
		Token string `json:"token"`
	}{
		Token: cfg.APIToken,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		logger.Debug("Failed to marshal sync request", zap.Error(err))
		return
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/wg-mesh-hub/sync-peers"
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
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
						zap.String("name", peerInfo.Name),
						zap.String("public_key", pubKey[:8]),
						zap.Error(err))
					continue
				}
			}

			activePeers[pubKey] = &ActivePeer{
				ID:         peerInfo.ID,
				Name:       peerInfo.Name,
				PublicKey:  peerInfo.PublicKey,
				AllowedIPs: peerInfo.AllowedIPs,
				TunnelIP:   peerInfo.TunnelIP,
			}

			logger.Info("Added new spoke peer",
				zap.String("name", peerInfo.Name),
				zap.String("public_key", pubKey[:8]))
		}
	}

	// Remove deleted peers
	for pubKey, peer := range activePeers {
		if _, exists := authorizedPeers[pubKey]; !exists {
			// Peer no longer authorized - remove from WireGuard
			if interfaceMgr != nil {
				if err := interfaceMgr.RemovePeer(ctx, pubKey); err != nil {
					logger.Warn("Failed to remove peer",
						zap.String("name", peer.Name),
						zap.String("public_key", pubKey[:8]),
						zap.Error(err))
				}
			}

			// Remove firewall rules
			if firewallMgr != nil {
				connectionID := fmt.Sprintf("wg-spoke-%s", strings.ReplaceAll(pubKey[:16], "/", "-"))
				if err := firewallMgr.RemoveRules(ctx, connectionID); err != nil {
					logger.Debug("Failed to remove firewall rules",
						zap.String("name", peer.Name),
						zap.Error(err))
				}
			}

			delete(activePeers, pubKey)

			logger.Info("Removed spoke peer",
				zap.String("name", peer.Name),
				zap.String("public_key", pubKey[:8]))
		}
	}
}

// statsSyncLoop periodically reports peer statistics to the control plane
func statsSyncLoop(ctx context.Context, cfg *HubConfig) {
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

func syncStats(ctx context.Context, cfg *HubConfig) {
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
		Token: cfg.APIToken,
		Peers: peers,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		logger.Debug("Failed to marshal stats", zap.Error(err))
		return
	}

	url := strings.TrimSuffix(cfg.ControlPlaneURL, "/") + "/api/v1/wg-mesh-hub/peer-stats"
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
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

// PeerStats contains statistics for a WireGuard peer
type PeerStats struct {
	Endpoint      string
	LastHandshake time.Time
	BytesSent     int64
	BytesReceived int64
}

// getWireGuardPeerStats reads peer statistics from wg show
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

// applyPeerFirewallRules applies firewall rules for a connected spoke
func applyPeerFirewallRules(ctx context.Context, peer *ActivePeer) error {
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

	// Parse source IP from tunnel IP
	sourceIP := net.ParseIP(strings.TrimSuffix(peer.TunnelIP, "/32"))
	if sourceIP == nil {
		return fmt.Errorf("invalid tunnel IP: %s", peer.TunnelIP)
	}

	// Generate a unique ID for the connection
	uid := uuid.New()
	connectionID := fmt.Sprintf("wg-spoke-%s", strings.ReplaceAll(peer.PublicKey[:16], "/", "-"))

	return firewallMgr.ApplyRules(ctx, connectionID, uid, sourceIP, networks, nil)
}

// provisionHub handles the provision command to fetch config from control plane.
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

	// Initialize active peers map
	activePeers = make(map[string]*ActivePeer)

	ctx := context.Background()
	if err := provision(ctx, cfg); err != nil {
		return fmt.Errorf("provision failed: %w", err)
	}

	fmt.Println("Hub provisioned successfully")
	fmt.Printf("  Hub ID:        %s\n", hubID)
	fmt.Printf("  Hub Name:      %s\n", hubName)
	fmt.Printf("  Listen Port:   %d\n", listenPort)
	fmt.Printf("  VPN Subnet:    %s\n", vpnSubnet)
	fmt.Printf("  Interface:     %s\n", interfaceName)
	fmt.Printf("  Initial Spokes: %d\n", len(activePeers))

	return nil
}
