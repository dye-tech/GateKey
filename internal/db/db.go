// Package db provides database connectivity and store implementations.
package db

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a PostgreSQL connection pool
type DB struct {
	Pool *pgxpool.Pool
}

// TLSConfig holds database TLS configuration options
type TLSConfig struct {
	SSLMode     string // disable, require, verify-ca, verify-full
	SSLRootCert string // Path to CA certificate
	SSLCert     string // Path to client certificate (for mTLS)
	SSLKey      string // Path to client private key (for mTLS)
}

// New creates a new database connection pool
func New(ctx context.Context, connString string) (*DB, error) {
	return NewWithTLS(ctx, connString, nil)
}

// NewWithTLS creates a new database connection pool with TLS configuration
func NewWithTLS(ctx context.Context, connString string, tlsCfg *TLSConfig) (*DB, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 5 * time.Minute
	config.MaxConnIdleTime = 1 * time.Minute

	// Apply TLS configuration if provided
	if tlsCfg != nil && tlsCfg.SSLMode != "" && tlsCfg.SSLMode != "disable" {
		tlsConfig, err := buildTLSConfig(tlsCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
		config.ConnConfig.TLSConfig = tlsConfig
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// buildTLSConfig creates a crypto/tls.Config from TLSConfig options
func buildTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Set InsecureSkipVerify based on SSL mode
	switch cfg.SSLMode {
	case "require":
		// Encrypt connection but don't verify server certificate
		tlsConfig.InsecureSkipVerify = true
	case "verify-ca", "verify-full":
		// Verify server certificate
		tlsConfig.InsecureSkipVerify = false

		// Load CA certificate
		if cfg.SSLRootCert != "" {
			caCert, err := os.ReadFile(cfg.SSLRootCert)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate: %w", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}
			tlsConfig.RootCAs = caCertPool
		}

		// For verify-full, also verify the server hostname
		if cfg.SSLMode == "verify-full" {
			// ServerName will be set by pgx from the host in connection string
			// We just need to ensure InsecureSkipVerify is false
		}
	}

	// Load client certificate for mTLS if provided
	if cfg.SSLCert != "" && cfg.SSLKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.SSLCert, cfg.SSLKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.Pool.Close()
}

// Ping checks the database connection
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}
