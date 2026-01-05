package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrWGPeerNotFound = errors.New("wireguard peer not found")
)

// WireGuardPeer represents an active WireGuard peer connection
type WireGuardPeer struct {
	ID            string
	GatewayID     string
	ConfigID      string
	UserID        string
	UserType      string // "sso" or "local"
	PublicKey     string
	AssignedIP    string
	AllowedIPs    []string
	Endpoint      string
	LastHandshake *time.Time
	BytesSent     int64
	BytesReceived int64
	ConnectedAt   time.Time
	DisconnectedAt *time.Time
}

// WireGuardPeerStore handles WireGuard peer persistence
type WireGuardPeerStore struct {
	db *DB
}

// NewWireGuardPeerStore creates a new WireGuard peer store
func NewWireGuardPeerStore(db *DB) *WireGuardPeerStore {
	return &WireGuardPeerStore{db: db}
}

// Upsert creates or updates a peer connection
func (s *WireGuardPeerStore) Upsert(ctx context.Context, peer *WireGuardPeer) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO wireguard_peers (id, gateway_id, config_id, user_id, user_type, public_key, assigned_ip, allowed_ips, endpoint, last_handshake, bytes_sent, bytes_received, connected_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::cidr, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (gateway_id, public_key) DO UPDATE SET
			endpoint = EXCLUDED.endpoint,
			last_handshake = EXCLUDED.last_handshake,
			bytes_sent = EXCLUDED.bytes_sent,
			bytes_received = EXCLUDED.bytes_received,
			allowed_ips = EXCLUDED.allowed_ips,
			disconnected_at = NULL
	`, peer.ID, peer.GatewayID, peer.ConfigID, peer.UserID, peer.UserType, peer.PublicKey,
		peer.AssignedIP, peer.AllowedIPs, peer.Endpoint, peer.LastHandshake, peer.BytesSent, peer.BytesReceived, peer.ConnectedAt)
	return err
}

// UpdateStats updates peer statistics
func (s *WireGuardPeerStore) UpdateStats(ctx context.Context, gatewayID, publicKey string, endpoint string, lastHandshake *time.Time, bytesSent, bytesReceived int64) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE wireguard_peers
		SET endpoint = $3, last_handshake = $4, bytes_sent = $5, bytes_received = $6
		WHERE gateway_id = $1 AND public_key = $2 AND disconnected_at IS NULL
	`, gatewayID, publicKey, endpoint, lastHandshake, bytesSent, bytesReceived)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrWGPeerNotFound
	}
	return nil
}

// Disconnect marks a peer as disconnected
func (s *WireGuardPeerStore) Disconnect(ctx context.Context, gatewayID, publicKey string) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE wireguard_peers
		SET disconnected_at = NOW()
		WHERE gateway_id = $1 AND public_key = $2 AND disconnected_at IS NULL
	`, gatewayID, publicKey)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrWGPeerNotFound
	}
	return nil
}

// GetByPublicKey retrieves a peer by public key
func (s *WireGuardPeerStore) GetByPublicKey(ctx context.Context, gatewayID, publicKey string) (*WireGuardPeer, error) {
	var peer WireGuardPeer
	var endpoint *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, gateway_id, config_id, user_id, user_type, public_key, assigned_ip::text, allowed_ips,
		       endpoint, last_handshake, bytes_sent, bytes_received, connected_at, disconnected_at
		FROM wireguard_peers
		WHERE gateway_id = $1 AND public_key = $2
	`, gatewayID, publicKey).Scan(&peer.ID, &peer.GatewayID, &peer.ConfigID, &peer.UserID, &peer.UserType,
		&peer.PublicKey, &peer.AssignedIP, &peer.AllowedIPs, &endpoint, &peer.LastHandshake,
		&peer.BytesSent, &peer.BytesReceived, &peer.ConnectedAt, &peer.DisconnectedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrWGPeerNotFound
	}
	if err != nil {
		return nil, err
	}
	if endpoint != nil {
		peer.Endpoint = *endpoint
	}
	return &peer, nil
}

// GetActiveByGateway returns all active (connected) peers for a gateway
func (s *WireGuardPeerStore) GetActiveByGateway(ctx context.Context, gatewayID string) ([]*WireGuardPeer, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, gateway_id, config_id, user_id, user_type, public_key, assigned_ip::text, allowed_ips,
		       endpoint, last_handshake, bytes_sent, bytes_received, connected_at
		FROM wireguard_peers
		WHERE gateway_id = $1 AND disconnected_at IS NULL
		ORDER BY connected_at DESC
	`, gatewayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []*WireGuardPeer
	for rows.Next() {
		var peer WireGuardPeer
		var endpoint *string
		if err := rows.Scan(&peer.ID, &peer.GatewayID, &peer.ConfigID, &peer.UserID, &peer.UserType,
			&peer.PublicKey, &peer.AssignedIP, &peer.AllowedIPs, &endpoint, &peer.LastHandshake,
			&peer.BytesSent, &peer.BytesReceived, &peer.ConnectedAt); err != nil {
			return nil, err
		}
		if endpoint != nil {
			peer.Endpoint = *endpoint
		}
		peers = append(peers, &peer)
	}
	return peers, rows.Err()
}

// CountActiveByGateway returns the number of active peers for a gateway
func (s *WireGuardPeerStore) CountActiveByGateway(ctx context.Context, gatewayID string) (int, error) {
	var count int
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM wireguard_peers
		WHERE gateway_id = $1 AND disconnected_at IS NULL
	`, gatewayID).Scan(&count)
	return count, err
}

// DeleteByGateway deletes all peers for a gateway
func (s *WireGuardPeerStore) DeleteByGateway(ctx context.Context, gatewayID string) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `
		DELETE FROM wireguard_peers WHERE gateway_id = $1
	`, gatewayID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// DeleteByConfig deletes all peers for a specific config
func (s *WireGuardPeerStore) DeleteByConfig(ctx context.Context, configID string) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `
		DELETE FROM wireguard_peers WHERE config_id = $1
	`, configID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// WireGuardPeerWithUser extends WireGuardPeer with user information
type WireGuardPeerWithUser struct {
	WireGuardPeer
	UserEmail   string
	UserName    string
	GatewayName string
}

// GetAllActivePeers returns all active peers across all gateways (for admin/topology view)
func (s *WireGuardPeerStore) GetAllActivePeers(ctx context.Context) ([]*WireGuardPeerWithUser, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT wp.id, wp.gateway_id, wp.config_id, wp.user_id, wp.user_type, wp.public_key, wp.assigned_ip::text,
		       wp.allowed_ips, wp.endpoint, wp.last_handshake, wp.bytes_sent, wp.bytes_received, wp.connected_at,
		       COALESCE(u.email, lu.email, wp.user_id) as user_email,
		       COALESCE(u.name, lu.username, '') as user_name,
		       g.name as gateway_name
		FROM wireguard_peers wp
		LEFT JOIN users u ON wp.user_id = u.id::text
		LEFT JOIN local_users lu ON wp.user_id = lu.id::text
		LEFT JOIN gateways g ON wp.gateway_id = g.id
		WHERE wp.disconnected_at IS NULL
		ORDER BY wp.connected_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []*WireGuardPeerWithUser
	for rows.Next() {
		var peer WireGuardPeerWithUser
		var endpoint *string
		if err := rows.Scan(&peer.ID, &peer.GatewayID, &peer.ConfigID, &peer.UserID, &peer.UserType,
			&peer.PublicKey, &peer.AssignedIP, &peer.AllowedIPs, &endpoint, &peer.LastHandshake,
			&peer.BytesSent, &peer.BytesReceived, &peer.ConnectedAt,
			&peer.UserEmail, &peer.UserName, &peer.GatewayName); err != nil {
			return nil, err
		}
		if endpoint != nil {
			peer.Endpoint = *endpoint
		}
		peers = append(peers, &peer)
	}
	return peers, rows.Err()
}

// CleanupStale removes peer entries that haven't had a handshake in the specified duration
func (s *WireGuardPeerStore) CleanupStale(ctx context.Context, staleThreshold time.Duration) (int64, error) {
	cutoff := time.Now().Add(-staleThreshold)
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE wireguard_peers
		SET disconnected_at = NOW()
		WHERE disconnected_at IS NULL AND (last_handshake IS NULL OR last_handshake < $1)
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
