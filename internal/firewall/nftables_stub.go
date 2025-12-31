//go:build !linux

// Package firewall provides nftables backend stub for non-Linux platforms.
package firewall

import (
	"context"
	"errors"
	"net"
)

var errNotSupported = errors.New("nftables is only supported on Linux")

// NFTablesBackend is a stub for non-Linux platforms.
type NFTablesBackend struct{}

// NFTablesConfig holds nftables configuration.
type NFTablesConfig struct {
	TableName string
	ChainName string
}

// NewNFTablesBackend returns an error on non-Linux platforms.
func NewNFTablesBackend(cfg NFTablesConfig) (*NFTablesBackend, error) {
	return nil, errNotSupported
}

// Initialize returns an error on non-Linux platforms.
func (b *NFTablesBackend) Initialize(ctx context.Context) error {
	return errNotSupported
}

// AddDefaultDropRule returns an error on non-Linux platforms.
func (b *NFTablesBackend) AddDefaultDropRule(ctx context.Context, sourceIP net.IP) error {
	return errNotSupported
}

// FlushAllRules returns an error on non-Linux platforms.
func (b *NFTablesBackend) FlushAllRules(ctx context.Context) error {
	return errNotSupported
}

// AddRules returns an error on non-Linux platforms.
func (b *NFTablesBackend) AddRules(ctx context.Context, rules []Rule) error {
	return errNotSupported
}

// RemoveRules returns an error on non-Linux platforms.
func (b *NFTablesBackend) RemoveRules(ctx context.Context, connectionID string) error {
	return errNotSupported
}

// ListRules returns an error on non-Linux platforms.
func (b *NFTablesBackend) ListRules(ctx context.Context) ([]Rule, error) {
	return nil, errNotSupported
}

// Cleanup returns an error on non-Linux platforms.
func (b *NFTablesBackend) Cleanup(ctx context.Context) error {
	return errNotSupported
}

// Close is a no-op on non-Linux platforms.
func (b *NFTablesBackend) Close() error {
	return nil
}
