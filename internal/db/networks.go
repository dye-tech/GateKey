package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrNetworkNotFound = errors.New("network not found")
	ErrNetworkExists   = errors.New("network already exists")
)

// Network represents a CIDR network block
type Network struct {
	ID          string
	Name        string
	Description string
	CIDR        string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NetworkStore handles network persistence
type NetworkStore struct {
	db *DB
}

// NewNetworkStore creates a new network store
func NewNetworkStore(db *DB) *NetworkStore {
	return &NetworkStore{db: db}
}

// CreateNetwork creates a new network
func (s *NetworkStore) CreateNetwork(ctx context.Context, network *Network) error {
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO networks (name, description, cidr, is_active)
		VALUES ($1, $2, $3::cidr, $4)
		RETURNING id, created_at, updated_at
	`, network.Name, network.Description, network.CIDR, network.IsActive).Scan(
		&network.ID, &network.CreatedAt, &network.UpdatedAt,
	)
	if err != nil && err.Error() == "ERROR: duplicate key value violates unique constraint \"networks_name_key\" (SQLSTATE 23505)" {
		return ErrNetworkExists
	}
	return err
}

// GetNetwork retrieves a network by ID
func (s *NetworkStore) GetNetwork(ctx context.Context, id string) (*Network, error) {
	var network Network
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, description, cidr::text, is_active, created_at, updated_at
		FROM networks WHERE id = $1
	`, id).Scan(&network.ID, &network.Name, &network.Description, &network.CIDR,
		&network.IsActive, &network.CreatedAt, &network.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNetworkNotFound
	}
	return &network, err
}

// ListNetworks retrieves all networks
func (s *NetworkStore) ListNetworks(ctx context.Context) ([]*Network, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, name, description, cidr::text, is_active, created_at, updated_at
		FROM networks ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var networks []*Network
	for rows.Next() {
		var n Network
		if err := rows.Scan(&n.ID, &n.Name, &n.Description, &n.CIDR,
			&n.IsActive, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		networks = append(networks, &n)
	}
	return networks, rows.Err()
}

// UpdateNetwork updates a network
func (s *NetworkStore) UpdateNetwork(ctx context.Context, network *Network) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE networks SET name = $2, description = $3, cidr = $4::cidr, is_active = $5
		WHERE id = $1
	`, network.ID, network.Name, network.Description, network.CIDR, network.IsActive)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNetworkNotFound
	}
	return nil
}

// DeleteNetwork deletes a network
func (s *NetworkStore) DeleteNetwork(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM networks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNetworkNotFound
	}
	return nil
}

// AssignGatewayToNetwork assigns a gateway to a network
func (s *NetworkStore) AssignGatewayToNetwork(ctx context.Context, gatewayID, networkID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO gateway_networks (gateway_id, network_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, gatewayID, networkID)
	return err
}

// RemoveGatewayFromNetwork removes a gateway from a network
func (s *NetworkStore) RemoveGatewayFromNetwork(ctx context.Context, gatewayID, networkID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM gateway_networks WHERE gateway_id = $1 AND network_id = $2
	`, gatewayID, networkID)
	return err
}

// GetGatewayNetworks gets all networks assigned to a gateway
func (s *NetworkStore) GetGatewayNetworks(ctx context.Context, gatewayID string) ([]*Network, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT n.id, n.name, n.description, n.cidr::text, n.is_active, n.created_at, n.updated_at
		FROM networks n
		JOIN gateway_networks gn ON n.id = gn.network_id
		WHERE gn.gateway_id = $1
		ORDER BY n.name
	`, gatewayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var networks []*Network
	for rows.Next() {
		var n Network
		if err := rows.Scan(&n.ID, &n.Name, &n.Description, &n.CIDR,
			&n.IsActive, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		networks = append(networks, &n)
	}
	return networks, rows.Err()
}

// GetNetworkGateways gets all gateways assigned to a network
func (s *NetworkStore) GetNetworkGateways(ctx context.Context, networkID string) ([]*Gateway, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT g.id, g.name, g.hostname, host(g.public_ip), g.vpn_port, g.vpn_protocol,
		       g.is_active, g.last_heartbeat, g.created_at, g.updated_at
		FROM gateways g
		JOIN gateway_networks gn ON g.id = gn.gateway_id
		WHERE gn.network_id = $1
		ORDER BY g.name
	`, networkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gateways []*Gateway
	for rows.Next() {
		var g Gateway
		var hostname, publicIP *string
		if err := rows.Scan(&g.ID, &g.Name, &hostname, &publicIP, &g.VPNPort, &g.VPNProtocol,
			&g.IsActive, &g.LastHeartbeat, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		if hostname != nil {
			g.Hostname = *hostname
		}
		if publicIP != nil {
			g.PublicIP = *publicIP
		}
		gateways = append(gateways, &g)
	}
	return gateways, rows.Err()
}
