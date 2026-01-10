package wireguard

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// Peer represents an authorized WireGuard peer.
type Peer struct {
	PublicKey    string
	PresharedKey string   // Optional
	AllowedIPs   []string // Client's allowed IPs (typically just their assigned IP)
	UserID       string
	UserEmail    string
	ConfigID     string
	ExpiresAt    time.Time
}

// PeerManager manages WireGuard peers for a gateway.
type PeerManager struct {
	interfaceName string
	peers         map[string]*Peer // keyed by public key
	mu            sync.RWMutex
	executor      CommandExecutor
}

// NewPeerManager creates a new peer manager.
func NewPeerManager(interfaceName string) *PeerManager {
	return &PeerManager{
		interfaceName: interfaceName,
		peers:         make(map[string]*Peer),
		executor:      DefaultExecutor(),
	}
}

// NewPeerManagerWithExecutor creates a peer manager with a custom executor (for testing).
func NewPeerManagerWithExecutor(interfaceName string, executor CommandExecutor) *PeerManager {
	return &PeerManager{
		interfaceName: interfaceName,
		peers:         make(map[string]*Peer),
		executor:      executor,
	}
}

// AddPeer adds a peer to the WireGuard interface.
func (m *PeerManager) AddPeer(ctx context.Context, peer *Peer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := []string{"set", m.interfaceName, "peer", peer.PublicKey}

	// Add allowed IPs
	if len(peer.AllowedIPs) > 0 {
		args = append(args, "allowed-ips", strings.Join(peer.AllowedIPs, ","))
	}

	// Handle preshared key if present
	if peer.PresharedKey != "" {
		// Write PSK to temp file
		tmpFile, err := os.CreateTemp("", "wg-psk-*")
		if err != nil {
			return fmt.Errorf("failed to create temp file for PSK: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(peer.PresharedKey); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to write PSK: %w", err)
		}
		tmpFile.Close()

		args = append(args, "preshared-key", tmpFile.Name())
	}

	if output, err := m.executor.Output(ctx, "wg", args...); err != nil {
		return fmt.Errorf("wg set peer failed: %s: %w", string(output), err)
	}

	m.peers[peer.PublicKey] = peer
	return nil
}

// RemovePeer removes a peer from the WireGuard interface.
func (m *PeerManager) RemovePeer(ctx context.Context, publicKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if output, err := m.executor.Output(ctx, "wg", "set", m.interfaceName, "peer", publicKey, "remove"); err != nil {
		return fmt.Errorf("wg set peer remove failed: %s: %w", string(output), err)
	}

	delete(m.peers, publicKey)
	return nil
}

// SyncPeers synchronizes the local peer list with the authorized peers from control plane.
// It adds new peers, removes revoked/expired peers, and updates existing peers if needed.
func (m *PeerManager) SyncPeers(ctx context.Context, authorizedPeers []*Peer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Build a map of authorized peers for quick lookup
	authorizedMap := make(map[string]*Peer)
	for _, p := range authorizedPeers {
		authorizedMap[p.PublicKey] = p
	}

	// Remove peers that are no longer authorized
	for pubKey := range m.peers {
		if _, ok := authorizedMap[pubKey]; !ok {
			if output, err := m.executor.Output(ctx, "wg", "set", m.interfaceName, "peer", pubKey, "remove"); err != nil {
				// Log but continue - don't fail the entire sync for one peer
				fmt.Printf("warning: failed to remove peer %s: %s: %v\n", pubKey[:8], string(output), err)
			}
			delete(m.peers, pubKey)
		}
	}

	// Add or update authorized peers
	for _, peer := range authorizedPeers {
		// Skip if peer is expired
		if time.Now().After(peer.ExpiresAt) {
			continue
		}

		existing, exists := m.peers[peer.PublicKey]

		// Check if we need to update the peer
		needsUpdate := !exists || !equalStringSlices(existing.AllowedIPs, peer.AllowedIPs)

		if needsUpdate {
			args := []string{"set", m.interfaceName, "peer", peer.PublicKey}

			if len(peer.AllowedIPs) > 0 {
				args = append(args, "allowed-ips", strings.Join(peer.AllowedIPs, ","))
			}

			if peer.PresharedKey != "" {
				tmpFile, err := os.CreateTemp("", "wg-psk-*")
				if err != nil {
					fmt.Printf("warning: failed to create temp file for PSK: %v\n", err)
					continue
				}

				if _, err := tmpFile.WriteString(peer.PresharedKey); err != nil {
					tmpFile.Close()
					os.Remove(tmpFile.Name())
					fmt.Printf("warning: failed to write PSK: %v\n", err)
					continue
				}
				tmpFile.Close()

				args = append(args, "preshared-key", tmpFile.Name())
				defer os.Remove(tmpFile.Name())
			}

			if output, err := m.executor.Output(ctx, "wg", args...); err != nil {
				fmt.Printf("warning: failed to add/update peer %s: %s: %v\n", peer.PublicKey[:8], string(output), err)
				continue
			}

			m.peers[peer.PublicKey] = peer
		}
	}

	return nil
}

// GetPeers returns a copy of the current peer list.
func (m *PeerManager) GetPeers() []*Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peers := make([]*Peer, 0, len(m.peers))
	for _, p := range m.peers {
		// Return a copy to avoid race conditions
		peerCopy := *p
		peers = append(peers, &peerCopy)
	}
	return peers
}

// GetPeer returns a specific peer by public key.
func (m *PeerManager) GetPeer(publicKey string) (*Peer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peer, ok := m.peers[publicKey]
	if !ok {
		return nil, false
	}

	// Return a copy
	peerCopy := *peer
	return &peerCopy, true
}

// PeerCount returns the number of active peers.
func (m *PeerManager) PeerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.peers)
}

// equalStringSlices compares two string slices for equality.
func equalStringSlices(a, b []string) bool {
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

// IPAllocator manages IP address allocation within a subnet.
type IPAllocator struct {
	network   *net.IPNet
	allocated map[string]bool
	mu        sync.Mutex
}

// NewIPAllocator creates a new IP allocator for the given subnet.
func NewIPAllocator(cidr string) (*IPAllocator, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	return &IPAllocator{
		network:   network,
		allocated: make(map[string]bool),
	}, nil
}

// Allocate allocates the next available IP in the subnet.
// The first IP (network address) and last IP (broadcast for IPv4) are skipped.
// The second IP (.1) is typically reserved for the server.
func (a *IPAllocator) Allocate() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	ip := make(net.IP, len(a.network.IP))
	copy(ip, a.network.IP)

	// Start from .2 (skip network address and server address)
	for i := 2; ; i++ {
		// Calculate the IP
		candidateIP := incrementIP(a.network.IP, i)

		// Check if still in network
		if !a.network.Contains(candidateIP) {
			return "", fmt.Errorf("no available IPs in subnet")
		}

		ipStr := candidateIP.String()
		if !a.allocated[ipStr] {
			a.allocated[ipStr] = true
			// Return with /32 for client
			return ipStr + "/32", nil
		}
	}
}

// Release releases an allocated IP back to the pool.
func (a *IPAllocator) Release(ip string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Strip CIDR suffix if present
	if idx := strings.Index(ip, "/"); idx != -1 {
		ip = ip[:idx]
	}

	delete(a.allocated, ip)
}

// MarkAllocated marks an IP as allocated (used when loading existing configs).
func (a *IPAllocator) MarkAllocated(ip string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Strip CIDR suffix if present
	if idx := strings.Index(ip, "/"); idx != -1 {
		ip = ip[:idx]
	}

	a.allocated[ip] = true
}

// incrementIP increments an IP address by n.
func incrementIP(ip net.IP, n int) net.IP {
	result := make(net.IP, len(ip))
	copy(result, ip)

	for i := len(result) - 1; i >= 0 && n > 0; i-- {
		sum := int(result[i]) + n
		result[i] = byte(sum % 256)
		n = sum / 256
	}

	return result
}
