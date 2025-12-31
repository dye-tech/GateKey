//go:build linux

// Package firewall provides nftables backend implementation.
package firewall

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
)

// NFTablesBackend implements the firewall backend using nftables.
type NFTablesBackend struct {
	conn      *nftables.Conn
	table     *nftables.Table
	chain     *nftables.Chain
	tableName string
	chainName string
	rules     map[string][]*nftables.Rule // connectionID -> nftables rules
	mu        sync.Mutex
}

// NFTablesConfig holds nftables configuration.
type NFTablesConfig struct {
	TableName string
	ChainName string
}

// NewNFTablesBackend creates a new nftables backend.
func NewNFTablesBackend(cfg NFTablesConfig) (*NFTablesBackend, error) {
	conn, err := nftables.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create nftables connection: %w", err)
	}

	if cfg.TableName == "" {
		cfg.TableName = "gatekey"
	}
	if cfg.ChainName == "" {
		cfg.ChainName = "forward"
	}

	return &NFTablesBackend{
		conn:      conn,
		tableName: cfg.TableName,
		chainName: cfg.ChainName,
		rules:     make(map[string][]*nftables.Rule),
	}, nil
}

// Initialize sets up the nftables table and chain.
func (b *NFTablesBackend) Initialize(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Create table
	b.table = &nftables.Table{
		Family: nftables.TableFamilyIPv4,
		Name:   b.tableName,
	}
	b.conn.AddTable(b.table)

	// Create chain with default accept policy
	// We'll add explicit drop rules for VPN traffic at the end
	b.chain = &nftables.Chain{
		Name:     b.chainName,
		Table:    b.table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookForward,
		Priority: nftables.ChainPriorityFilter,
	}
	b.conn.AddChain(b.chain)

	// Flush and apply
	if err := b.conn.Flush(); err != nil {
		return fmt.Errorf("failed to initialize nftables: %w", err)
	}

	return nil
}

// AddDefaultDropRule adds a rule to drop all traffic from a VPN client IP
// This should be called after adding allow rules to create a whitelist
// The rule is tracked with connectionID so it can be removed later
func (b *NFTablesBackend) AddDefaultDropRule(ctx context.Context, sourceIP net.IP) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Create a drop rule for all traffic from this VPN client
	rule := &nftables.Rule{
		Table: b.table,
		Chain: b.chain,
		Exprs: []expr.Any{
			// Match source IP (VPN client)
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       12,
				Len:          4,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     sourceIP.To4(),
			},
			// Drop the packet
			&expr.Verdict{
				Kind: expr.VerdictDrop,
			},
		},
	}
	b.conn.AddRule(rule)

	if err := b.conn.Flush(); err != nil {
		return fmt.Errorf("failed to add default drop rule: %w", err)
	}

	// Track this rule with a special key so it gets removed on cleanup
	dropKey := fmt.Sprintf("drop-%s", sourceIP.String())
	b.rules[dropKey] = []*nftables.Rule{rule}

	return nil
}

// FlushAllRules removes all rules from the gatekey chain
// This is a nuclear option used before re-adding rules for a client
func (b *NFTablesBackend) FlushAllRules(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Get all rules in the chain and delete them
	rules, err := b.conn.GetRules(b.table, b.chain)
	if err != nil {
		return fmt.Errorf("failed to get rules: %w", err)
	}

	for _, rule := range rules {
		b.conn.DelRule(rule)
	}

	if err := b.conn.Flush(); err != nil {
		return fmt.Errorf("failed to flush rules: %w", err)
	}

	// Clear our tracking
	b.rules = make(map[string][]*nftables.Rule)
	return nil
}

// AddRules adds firewall rules for a connection.
func (b *NFTablesBackend) AddRules(ctx context.Context, rules []Rule) error {
	if len(rules) == 0 {
		return nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	connectionID := rules[0].ConnectionID
	var nftRules []*nftables.Rule

	for _, rule := range rules {
		nftRule := b.buildRule(rule)
		if nftRule != nil {
			b.conn.AddRule(nftRule)
			nftRules = append(nftRules, nftRule)
		}
	}

	if err := b.conn.Flush(); err != nil {
		return fmt.Errorf("failed to add nftables rules: %w", err)
	}

	b.rules[connectionID] = nftRules
	return nil
}

// RemoveRules removes firewall rules for a connection.
func (b *NFTablesBackend) RemoveRules(ctx context.Context, connectionID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	rules, exists := b.rules[connectionID]
	if !exists {
		return nil
	}

	for _, rule := range rules {
		if err := b.conn.DelRule(rule); err != nil {
			// Log but continue
			continue
		}
	}

	if err := b.conn.Flush(); err != nil {
		return fmt.Errorf("failed to remove nftables rules: %w", err)
	}

	delete(b.rules, connectionID)
	return nil
}

// ListRules lists all rules managed by gatekey.
func (b *NFTablesBackend) ListRules(ctx context.Context) ([]Rule, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var allRules []Rule
	for _, rules := range b.rules {
		for range rules {
			// Note: Converting back from nftables rules to our Rule struct
			// is complex and typically not needed. This is a placeholder.
			allRules = append(allRules, Rule{})
		}
	}
	return allRules, nil
}

// Cleanup removes all gatekey-managed rules.
func (b *NFTablesBackend) Cleanup(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Delete all tracked rules
	for connectionID := range b.rules {
		for _, rule := range b.rules[connectionID] {
			b.conn.DelRule(rule)
		}
	}

	if err := b.conn.Flush(); err != nil {
		return fmt.Errorf("failed to cleanup nftables rules: %w", err)
	}

	b.rules = make(map[string][]*nftables.Rule)
	return nil
}

// Close closes the nftables connection.
func (b *NFTablesBackend) Close() error {
	return nil // nftables.Conn doesn't need explicit close
}

// buildRule converts our Rule to an nftables Rule.
func (b *NFTablesBackend) buildRule(rule Rule) *nftables.Rule {
	var exprs []expr.Any

	// Match source IP
	if rule.SourceIP != nil {
		exprs = append(exprs,
			// Load source IP
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       12, // Source IP offset in IPv4 header
				Len:          4,
			},
			// Compare with rule source IP
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     rule.SourceIP.To4(),
			},
		)
	}

	// Match destination network
	if rule.DestNetwork.IP != nil {
		ones, _ := rule.DestNetwork.Mask.Size()
		exprs = append(exprs,
			// Load destination IP
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       16, // Destination IP offset in IPv4 header
				Len:          4,
			},
			// Apply network mask and compare
			&expr.Bitwise{
				SourceRegister: 1,
				DestRegister:   1,
				Len:            4,
				Mask:           net.CIDRMask(ones, 32),
				Xor:            []byte{0, 0, 0, 0},
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     rule.DestNetwork.IP.Mask(rule.DestNetwork.Mask),
			},
		)
	}

	// Match protocol
	if rule.Protocol != ProtocolAny && rule.Protocol != "" {
		var proto byte
		switch rule.Protocol {
		case ProtocolTCP:
			proto = 6
		case ProtocolUDP:
			proto = 17
		case ProtocolICMP:
			proto = 1
		case ProtocolAny:
			// ProtocolAny is already filtered above, but included for exhaustiveness
		}

		if proto > 0 {
			exprs = append(exprs,
				// Load protocol
				&expr.Payload{
					DestRegister: 1,
					Base:         expr.PayloadBaseNetworkHeader,
					Offset:       9, // Protocol offset in IPv4 header
					Len:          1,
				},
				&expr.Cmp{
					Op:       expr.CmpOpEq,
					Register: 1,
					Data:     []byte{proto},
				},
			)
		}
	}

	// Match destination port
	if rule.DestPort > 0 && (rule.Protocol == ProtocolTCP || rule.Protocol == ProtocolUDP) {
		exprs = append(exprs,
			// Load destination port
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2, // Destination port offset
				Len:          2,
			},
		)

		if rule.DestPortEnd > 0 && rule.DestPortEnd != rule.DestPort {
			// Port range
			exprs = append(exprs,
				&expr.Cmp{
					Op:       expr.CmpOpGte,
					Register: 1,
					Data:     []byte{byte(rule.DestPort >> 8), byte(rule.DestPort)},
				},
				&expr.Cmp{
					Op:       expr.CmpOpLte,
					Register: 1,
					Data:     []byte{byte(rule.DestPortEnd >> 8), byte(rule.DestPortEnd)},
				},
			)
		} else {
			// Single port
			exprs = append(exprs,
				&expr.Cmp{
					Op:       expr.CmpOpEq,
					Register: 1,
					Data:     []byte{byte(rule.DestPort >> 8), byte(rule.DestPort)},
				},
			)
		}
	}

	// Add verdict
	var verdict *expr.Verdict
	switch rule.Action {
	case ActionAccept:
		verdict = &expr.Verdict{Kind: expr.VerdictAccept}
	case ActionDrop:
		verdict = &expr.Verdict{Kind: expr.VerdictDrop}
	case ActionReject:
		// For reject, we use drop (reject requires additional setup)
		verdict = &expr.Verdict{Kind: expr.VerdictDrop}
	default:
		verdict = &expr.Verdict{Kind: expr.VerdictAccept}
	}
	exprs = append(exprs, verdict)

	return &nftables.Rule{
		Table: b.table,
		Chain: b.chain,
		Exprs: exprs,
	}
}
