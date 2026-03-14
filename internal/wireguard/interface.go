package wireguard

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

// InterfaceConfig holds WireGuard interface configuration.
type InterfaceConfig struct {
	PrivateKey string // Base64-encoded private key
	ListenPort int    // UDP listen port
	Address    string // CIDR notation (e.g., "10.0.0.1/24")
}

// InterfaceStats contains statistics for a WireGuard interface.
type InterfaceStats struct {
	PublicKey  string
	ListenPort int
	Peers      []InterfacePeerStats
}

// InterfacePeerStats contains statistics for a single peer.
type InterfacePeerStats struct {
	PublicKey           string
	Endpoint            string
	AllowedIPs          []string
	LatestHandshake     int64 // Unix timestamp
	TransferRx          int64 // Bytes received
	TransferTx          int64 // Bytes transmitted
	PersistentKeepalive int
}

// InterfaceManager manages a WireGuard network interface.
type InterfaceManager struct {
	name     string
	config   InterfaceConfig
	logger   *zap.Logger
	executor CommandExecutor
}

// NewInterfaceManager creates a new interface manager.
func NewInterfaceManager(name string, logger *zap.Logger) *InterfaceManager {
	return &InterfaceManager{
		name:     name,
		logger:   logger,
		executor: DefaultExecutor(),
	}
}

// NewInterfaceManagerWithExecutor creates an interface manager with a custom executor (for testing).
func NewInterfaceManagerWithExecutor(name string, logger *zap.Logger, executor CommandExecutor) *InterfaceManager {
	return &InterfaceManager{
		name:     name,
		logger:   logger,
		executor: executor,
	}
}

// Setup creates and configures the WireGuard interface.
func (m *InterfaceManager) Setup(ctx context.Context, config InterfaceConfig) error {
	m.config = config
	// 1. Create the interface
	if err := m.createInterface(ctx); err != nil {
		return fmt.Errorf("failed to create interface: %w", err)
	}

	// 2. Set the private key
	if err := m.setPrivateKey(ctx); err != nil {
		// Cleanup on failure
		_ = m.Teardown(ctx)
		return fmt.Errorf("failed to set private key: %w", err)
	}

	// 3. Set listen port
	if err := m.setListenPort(ctx); err != nil {
		_ = m.Teardown(ctx)
		return fmt.Errorf("failed to set listen port: %w", err)
	}

	// 4. Set address
	if err := m.setAddress(ctx); err != nil {
		_ = m.Teardown(ctx)
		return fmt.Errorf("failed to set address: %w", err)
	}

	// 5. Bring up the interface
	if err := m.bringUp(ctx); err != nil {
		_ = m.Teardown(ctx)
		return fmt.Errorf("failed to bring up interface: %w", err)
	}

	// 6. Configure networking for routing/forwarding
	if err := m.configureRouting(ctx); err != nil {
		if m.logger != nil {
			m.logger.Warn("Failed to configure routing (may require manual setup)", zap.Error(err))
		}
	}

	return nil
}

// configureRouting enables IP forwarding and disables reverse path filtering
// on the WireGuard interface so VPN traffic can be routed to the LAN.
func (m *InterfaceManager) configureRouting(ctx context.Context) error {
	// Enable IP forwarding
	if err := m.executor.Run(ctx, "sysctl", "-w", "net.ipv4.ip_forward=1"); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}

	// Disable reverse path filtering on the WireGuard interface
	// rp_filter=1 (strict) drops packets where the source address would not be
	// routed back via the same interface, which breaks VPN forwarding
	rpFilterKey := fmt.Sprintf("net.ipv4.conf.%s.rp_filter=0", m.name)
	if err := m.executor.Run(ctx, "sysctl", "-w", rpFilterKey); err != nil {
		return fmt.Errorf("failed to disable rp_filter: %w", err)
	}

	// Add NAT masquerade for VPN traffic exiting to the LAN
	// This ensures reply packets from LAN hosts are routed back through the hub
	if m.config.Address != "" {
		// Extract subnet from address (e.g., "172.30.0.1/16" -> "172.30.0.0/16")
		parts := strings.SplitN(m.config.Address, "/", 2)
		if len(parts) == 2 {
			ip := net.ParseIP(parts[0])
			if ip != nil {
				_, ipNet, err := net.ParseCIDR(m.config.Address)
				if err == nil {
					subnet := ipNet.String()
					// Check if masquerade rule already exists before adding
					checkErr := m.executor.Run(ctx, "iptables", "-t", "nat", "-C", "POSTROUTING",
						"-s", subnet, "-o", "eth0", "-j", "MASQUERADE")
					if checkErr != nil {
						// Rule doesn't exist, add it
						if err := m.executor.Run(ctx, "iptables", "-t", "nat", "-A", "POSTROUTING",
							"-s", subnet, "!", "-o", m.name, "-j", "MASQUERADE"); err != nil {
							return fmt.Errorf("failed to add masquerade rule: %w", err)
						}
					}
				}
			}
		}
	}

	if m.logger != nil {
		m.logger.Info("Routing configured for VPN forwarding",
			zap.String("interface", m.name))
	}
	return nil
}

// Teardown removes the WireGuard interface.
func (m *InterfaceManager) Teardown(ctx context.Context) error {
	if err := m.executor.Run(ctx, "ip", "link", "delete", m.name); err != nil {
		// Don't return error if interface doesn't exist
		if !strings.Contains(err.Error(), "Cannot find device") {
			return fmt.Errorf("failed to delete interface: %w", err)
		}
	}
	return nil
}

// createInterface creates the WireGuard interface.
func (m *InterfaceManager) createInterface(ctx context.Context) error {
	// Check if interface already exists
	if m.executor.Run(ctx, "ip", "link", "show", m.name) == nil {
		// Interface exists, delete it first
		if err := m.Teardown(ctx); err != nil {
			return err
		}
	}

	// Try kernel module first
	output, err := m.executor.Output(ctx, "ip", "link", "add", m.name, "type", "wireguard")
	if err == nil {
		return nil
	}

	// If kernel module not available, try wireguard-go
	outputStr := string(output)
	if strings.Contains(outputStr, "Operation not supported") || strings.Contains(outputStr, "not supported") {
		if m.logger != nil {
			m.logger.Info("Kernel WireGuard not available, trying wireguard-go userspace implementation")
		}

		// Check if wireguard-go is available
		wgGoPath, err := m.executor.LookPath("wireguard-go")
		if err != nil {
			return fmt.Errorf("kernel WireGuard not supported and wireguard-go not found in PATH: %w", err)
		}

		// Start wireguard-go in background mode (default behavior - it forks)
		if wgOutput, err := m.executor.Output(ctx, wgGoPath, m.name); err != nil {
			return fmt.Errorf("wireguard-go failed: %s: %w", string(wgOutput), err)
		}

		// Wait for the interface to be created
		for i := 0; i < 30; i++ {
			if m.executor.Run(ctx, "ip", "link", "show", m.name) == nil {
				if m.logger != nil {
					m.logger.Info("WireGuard interface created via wireguard-go", zap.String("interface", m.name))
				}
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
		return fmt.Errorf("wireguard-go started but interface %s not created", m.name)
	}

	return fmt.Errorf("ip link add failed: %s: %w", outputStr, err)
}

// setPrivateKey sets the WireGuard private key.
func (m *InterfaceManager) setPrivateKey(ctx context.Context) error {
	// Write private key to temp file (wg command reads from file)
	tmpFile, err := os.CreateTemp("", "wg-key-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(m.config.PrivateKey); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}
	tmpFile.Close()

	if output, err := m.executor.Output(ctx, "wg", "set", m.name, "private-key", tmpFile.Name()); err != nil {
		return fmt.Errorf("wg set private-key failed: %s: %w", string(output), err)
	}
	return nil
}

// setListenPort sets the WireGuard listen port.
func (m *InterfaceManager) setListenPort(ctx context.Context) error {
	if output, err := m.executor.Output(ctx, "wg", "set", m.name, "listen-port", fmt.Sprintf("%d", m.config.ListenPort)); err != nil {
		return fmt.Errorf("wg set listen-port failed: %s: %w", string(output), err)
	}
	return nil
}

// setAddress assigns an IP address to the interface.
func (m *InterfaceManager) setAddress(ctx context.Context) error {
	if output, err := m.executor.Output(ctx, "ip", "address", "add", m.config.Address, "dev", m.name); err != nil {
		// Ignore "RTNETLINK answers: File exists" error (address already set)
		if !strings.Contains(string(output), "File exists") {
			return fmt.Errorf("ip address add failed: %s: %w", string(output), err)
		}
	}
	return nil
}

// bringUp brings the interface up.
func (m *InterfaceManager) bringUp(ctx context.Context) error {
	if output, err := m.executor.Output(ctx, "ip", "link", "set", m.name, "up"); err != nil {
		return fmt.Errorf("ip link set up failed: %s: %w", string(output), err)
	}
	return nil
}

// GetStats retrieves interface and peer statistics.
func (m *InterfaceManager) GetStats(ctx context.Context) (*InterfaceStats, error) {
	output, err := m.executor.Output(ctx, "wg", "show", m.name, "dump")
	if err != nil {
		return nil, fmt.Errorf("wg show failed: %w", err)
	}

	return parseWgShowDump(string(output))
}

// parseWgShowDump parses the output of "wg show <interface> dump".
// Format: each line is tab-separated
// Line 1: private-key, public-key, listen-port, fwmark
// Subsequent lines (peers): public-key, preshared-key, endpoint, allowed-ips, latest-handshake, transfer-rx, transfer-tx, persistent-keepalive
func parseWgShowDump(output string) (*InterfaceStats, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty output")
	}

	stats := &InterfaceStats{}

	// Parse interface line
	ifaceParts := strings.Split(lines[0], "\t")
	if len(ifaceParts) >= 3 {
		stats.PublicKey = ifaceParts[1]
		fmt.Sscanf(ifaceParts[2], "%d", &stats.ListenPort)
	}

	// Parse peer lines
	for _, line := range lines[1:] {
		parts := strings.Split(line, "\t")
		if len(parts) < 8 {
			continue
		}

		peer := InterfacePeerStats{
			PublicKey: parts[0],
			Endpoint:  parts[2],
		}

		if parts[3] != "(none)" {
			peer.AllowedIPs = strings.Split(parts[3], ",")
		}

		fmt.Sscanf(parts[4], "%d", &peer.LatestHandshake)
		fmt.Sscanf(parts[5], "%d", &peer.TransferRx)
		fmt.Sscanf(parts[6], "%d", &peer.TransferTx)
		fmt.Sscanf(parts[7], "%d", &peer.PersistentKeepalive)

		stats.Peers = append(stats.Peers, peer)
	}

	return stats, nil
}

// IsInterfaceUp checks if the interface exists and is up.
func (m *InterfaceManager) IsInterfaceUp(ctx context.Context) bool {
	return m.executor.Run(ctx, "ip", "link", "show", m.name, "up") == nil
}

// AddPeer adds a peer to the WireGuard interface.
func (m *InterfaceManager) AddPeer(ctx context.Context, peer PeerConfig) error {
	args := []string{"set", m.name, "peer", peer.PublicKey}

	// Add preshared key if provided
	var tmpPSKFile *os.File
	if peer.PresharedKey != "" {
		// Write PSK to temp file
		var err error
		tmpPSKFile, err = os.CreateTemp("", "wg-psk-*")
		if err != nil {
			return fmt.Errorf("failed to create temp file for PSK: %w", err)
		}
		defer os.Remove(tmpPSKFile.Name())
		if _, err := tmpPSKFile.WriteString(peer.PresharedKey); err != nil {
			tmpPSKFile.Close()
			return fmt.Errorf("failed to write PSK: %w", err)
		}
		tmpPSKFile.Close()
		args = append(args, "preshared-key", tmpPSKFile.Name())
	}

	// Add endpoint if provided (for client-mode connections)
	if peer.Endpoint != "" {
		args = append(args, "endpoint", peer.Endpoint)
	}

	// Add allowed IPs
	if len(peer.AllowedIPs) > 0 {
		args = append(args, "allowed-ips", strings.Join(peer.AllowedIPs, ","))
	}

	// Add persistent keepalive if provided (for NAT traversal)
	if peer.PersistentKeepalive > 0 {
		args = append(args, "persistent-keepalive", fmt.Sprintf("%d", peer.PersistentKeepalive))
	}

	if output, err := m.executor.Output(ctx, "wg", args...); err != nil {
		return fmt.Errorf("wg set peer failed: %s: %w", string(output), err)
	}

	// Add routes for AllowedIPs that are networks (not just single hosts)
	// This is necessary because WireGuard's wg tool doesn't add routes automatically
	for _, allowedIP := range peer.AllowedIPs {
		if err := m.addRouteForAllowedIP(ctx, allowedIP); err != nil {
			if m.logger != nil {
				m.logger.Warn("Failed to add route for allowed IP",
					zap.String("allowed_ip", allowedIP),
					zap.Error(err))
			}
			// Don't fail the peer addition if route add fails
		}
	}

	if m.logger != nil {
		m.logger.Info("Added WireGuard peer",
			zap.String("interface", m.name),
			zap.String("public_key", peer.PublicKey),
			zap.Strings("allowed_ips", peer.AllowedIPs),
			zap.String("endpoint", peer.Endpoint),
			zap.Int("persistent_keepalive", peer.PersistentKeepalive))
	}

	return nil
}

// addRouteForAllowedIP adds a route for an AllowedIP if it's a network range
func (m *InterfaceManager) addRouteForAllowedIP(ctx context.Context, allowedIP string) error {
	// Skip single host addresses (/32 for IPv4, /128 for IPv6) that are in the VPN subnet
	// These are typically peer tunnel IPs and don't need explicit routes
	if strings.HasSuffix(allowedIP, "/32") || strings.HasSuffix(allowedIP, "/128") {
		return nil
	}

	// Try to add the route - ignore "File exists" errors (route already exists)
	output, err := m.executor.Output(ctx, "ip", "route", "add", allowedIP, "dev", m.name)
	if err != nil {
		outputStr := string(output)
		// Ignore "RTNETLINK answers: File exists" - route already present
		if strings.Contains(outputStr, "File exists") {
			return nil
		}
		return fmt.Errorf("ip route add failed: %s: %w", outputStr, err)
	}

	if m.logger != nil {
		m.logger.Debug("Added route for peer network",
			zap.String("network", allowedIP),
			zap.String("interface", m.name))
	}

	return nil
}

// RemovePeer removes a peer from the WireGuard interface.
func (m *InterfaceManager) RemovePeer(ctx context.Context, publicKey string) error {
	if output, err := m.executor.Output(ctx, "wg", "set", m.name, "peer", publicKey, "remove"); err != nil {
		return fmt.Errorf("wg set peer remove failed: %s: %w", string(output), err)
	}

	if m.logger != nil {
		m.logger.Info("Removed WireGuard peer",
			zap.String("interface", m.name),
			zap.String("public_key", publicKey))
	}

	return nil
}

// GetName returns the interface name.
func (m *InterfaceManager) GetName() string {
	return m.name
}

// GetConfig returns the current interface configuration.
func (m *InterfaceManager) GetConfig() InterfaceConfig {
	return m.config
}
