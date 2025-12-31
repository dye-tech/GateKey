// Package firewall provides per-identity firewall rule management.
package firewall

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/google/uuid"
)

// Backend represents a firewall backend.
type Backend interface {
	// Initialize sets up the firewall backend.
	Initialize(ctx context.Context) error

	// AddRules adds firewall rules for a connection.
	AddRules(ctx context.Context, rules []Rule) error

	// AddDefaultDropRule adds a drop rule for all traffic from a source IP
	AddDefaultDropRule(ctx context.Context, sourceIP net.IP) error

	// RemoveRules removes firewall rules for a connection.
	RemoveRules(ctx context.Context, connectionID string) error

	// FlushAllRules removes all rules from the firewall
	FlushAllRules(ctx context.Context) error

	// ListRules lists all rules managed by gatekey.
	ListRules(ctx context.Context) ([]Rule, error)

	// Cleanup removes all gatekey-managed rules.
	Cleanup(ctx context.Context) error

	// Close cleans up resources.
	Close() error
}

// Rule represents a firewall rule.
type Rule struct {
	ID           string   `json:"id"`
	ConnectionID string   `json:"connection_id"`
	UserID       uuid.UUID `json:"user_id"`
	SourceIP     net.IP   `json:"source_ip"`
	Action       Action   `json:"action"`
	Protocol     Protocol `json:"protocol"`
	DestNetwork  net.IPNet `json:"dest_network"`
	DestPort     int      `json:"dest_port,omitempty"`
	DestPortEnd  int      `json:"dest_port_end,omitempty"` // For port ranges
	Comment      string   `json:"comment,omitempty"`
}

// Action represents the firewall action.
type Action string

const (
	ActionAccept Action = "accept"
	ActionDrop   Action = "drop"
	ActionReject Action = "reject"
)

// Protocol represents the network protocol.
type Protocol string

const (
	ProtocolTCP  Protocol = "tcp"
	ProtocolUDP  Protocol = "udp"
	ProtocolICMP Protocol = "icmp"
	ProtocolAny  Protocol = "any"
)

// Manager manages firewall rules for connections.
type Manager struct {
	backend Backend
	mu      sync.RWMutex
	rules   map[string][]Rule // connectionID -> rules
}

// NewManager creates a new firewall manager.
func NewManager(backend Backend) *Manager {
	return &Manager{
		backend: backend,
		rules:   make(map[string][]Rule),
	}
}

// Initialize initializes the firewall manager.
func (m *Manager) Initialize(ctx context.Context) error {
	return m.backend.Initialize(ctx)
}

// ApplyRules applies firewall rules for a connection.
func (m *Manager) ApplyRules(ctx context.Context, connectionID string, userID uuid.UUID, sourceIP net.IP, networks []net.IPNet, ports []PortRange) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Flush all existing rules to start fresh
	// This ensures no stale rules remain and DROP is always at the end
	if err := m.backend.FlushAllRules(ctx); err != nil {
		// Log but continue - rules might not exist yet
	}
	m.rules = make(map[string][]Rule)

	// Build new rules
	var rules []Rule

	// Add default drop rule for this source (will be added last)
	for _, network := range networks {
		// Allow rules for each network
		for _, pr := range ports {
			rule := Rule{
				ID:           fmt.Sprintf("%s-%d", connectionID, len(rules)),
				ConnectionID: connectionID,
				UserID:       userID,
				SourceIP:     sourceIP,
				Action:       ActionAccept,
				Protocol:     pr.Protocol,
				DestNetwork:  network,
				DestPort:     pr.Port,
				DestPortEnd:  pr.PortEnd,
				Comment:      fmt.Sprintf("gatekey: user=%s", userID),
			}
			rules = append(rules, rule)
		}

		// If no specific ports, allow all ports to this network
		if len(ports) == 0 {
			rule := Rule{
				ID:           fmt.Sprintf("%s-%d", connectionID, len(rules)),
				ConnectionID: connectionID,
				UserID:       userID,
				SourceIP:     sourceIP,
				Action:       ActionAccept,
				Protocol:     ProtocolAny,
				DestNetwork:  network,
				Comment:      fmt.Sprintf("gatekey: user=%s", userID),
			}
			rules = append(rules, rule)
		}
	}

	if len(rules) == 0 {
		return nil // No rules to apply
	}

	// Apply rules
	if err := m.backend.AddRules(ctx, rules); err != nil {
		return fmt.Errorf("failed to add rules: %w", err)
	}

	// Add default drop rule for all traffic from this VPN client
	// This creates a whitelist - only explicitly allowed destinations are reachable
	if err := m.backend.AddDefaultDropRule(ctx, sourceIP); err != nil {
		return fmt.Errorf("failed to add default drop rule: %w", err)
	}

	m.rules[connectionID] = rules
	return nil
}

// RemoveRules removes firewall rules for a connection.
func (m *Manager) RemoveRules(ctx context.Context, connectionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.rules[connectionID]; !exists {
		return nil // No rules to remove
	}

	if err := m.backend.RemoveRules(ctx, connectionID); err != nil {
		return fmt.Errorf("failed to remove rules: %w", err)
	}

	delete(m.rules, connectionID)
	return nil
}

// ListRules lists all managed rules.
func (m *Manager) ListRules(ctx context.Context) ([]Rule, error) {
	return m.backend.ListRules(ctx)
}

// Cleanup removes all gatekey-managed rules.
func (m *Manager) Cleanup(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.backend.Cleanup(ctx); err != nil {
		return fmt.Errorf("failed to cleanup rules: %w", err)
	}

	m.rules = make(map[string][]Rule)
	return nil
}

// Close closes the firewall manager.
func (m *Manager) Close() error {
	return m.backend.Close()
}

// PortRange represents a port or port range.
type PortRange struct {
	Protocol Protocol
	Port     int
	PortEnd  int // 0 if single port
}

// ConnectionRules returns the rules for a specific connection.
func (m *Manager) ConnectionRules(connectionID string) []Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rules[connectionID]
}

// UserConnections returns all connection IDs for a user.
func (m *Manager) UserConnections(userID uuid.UUID) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var connections []string
	for connID, rules := range m.rules {
		if len(rules) > 0 && rules[0].UserID == userID {
			connections = append(connections, connID)
		}
	}
	return connections
}
