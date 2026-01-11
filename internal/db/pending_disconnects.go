package db

import (
	"context"
	"time"
)

// PendingDisconnect represents a request to disconnect a specific user's VPN session.
// This targets individual users only - NOT groups. Gateway agents poll for these
// and execute the disconnect while keeping the gateway running for other users.
type PendingDisconnect struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	UserEmail    string     `json:"user_email"`
	GatewayID    *string    `json:"gateway_id,omitempty"`    // nil = all gateways
	NodeType     string     `json:"node_type"`               // "gateway" or "hub"
	ConnectionID *string    `json:"connection_id,omitempty"` // specific connection
	VPNAddress   *string    `json:"vpn_address,omitempty"`   // for identification
	Reason       string     `json:"reason"`
	RequestedBy  string     `json:"requested_by"`
	RequestedAt  time.Time  `json:"requested_at"`
	ExecutedAt   *time.Time `json:"executed_at,omitempty"`
	ExecutedBy   *string    `json:"executed_by_gateway,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// DisconnectRequest is used to create a new pending disconnect.
type DisconnectRequest struct {
	UserID       string  `json:"user_id"`
	UserEmail    string  `json:"user_email"`
	GatewayID    *string `json:"gateway_id,omitempty"`
	NodeType     string  `json:"node_type"`
	ConnectionID *string `json:"connection_id,omitempty"`
	VPNAddress   *string `json:"vpn_address,omitempty"`
	Reason       string  `json:"reason"`
	RequestedBy  string  `json:"requested_by"`
}

// PendingDisconnectStore handles pending disconnect persistence.
type PendingDisconnectStore struct {
	db *DB
}

// NewPendingDisconnectStore creates a new pending disconnect store.
func NewPendingDisconnectStore(db *DB) *PendingDisconnectStore {
	return &PendingDisconnectStore{db: db}
}

// CreateDisconnectRequest creates a new pending disconnect request for a specific user.
func (s *PendingDisconnectStore) CreateDisconnectRequest(ctx context.Context, req *DisconnectRequest) (*PendingDisconnect, error) {
	var pd PendingDisconnect
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO pending_disconnects (user_id, user_email, gateway_id, node_type, connection_id, vpn_address, reason, requested_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, user_email, gateway_id, node_type, connection_id, vpn_address, reason, requested_by, requested_at, expires_at, created_at
	`, req.UserID, req.UserEmail, req.GatewayID, req.NodeType, req.ConnectionID, req.VPNAddress, req.Reason, req.RequestedBy).Scan(
		&pd.ID, &pd.UserID, &pd.UserEmail, &pd.GatewayID, &pd.NodeType, &pd.ConnectionID,
		&pd.VPNAddress, &pd.Reason, &pd.RequestedBy, &pd.RequestedAt, &pd.ExpiresAt, &pd.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &pd, nil
}

// GetPendingDisconnectsForGateway retrieves pending disconnects for a specific gateway.
// This is called by gateway agents to check for users they need to disconnect.
func (s *PendingDisconnectStore) GetPendingDisconnectsForGateway(ctx context.Context, gatewayID string, nodeType string) ([]*PendingDisconnect, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, user_email, gateway_id, node_type, connection_id, vpn_address,
		       reason, requested_by, requested_at, expires_at, created_at
		FROM pending_disconnects
		WHERE (gateway_id = $1 OR gateway_id IS NULL)
		  AND node_type = $2
		  AND executed_at IS NULL
		  AND expires_at > NOW()
		ORDER BY requested_at ASC
	`, gatewayID, nodeType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var disconnects []*PendingDisconnect
	for rows.Next() {
		var pd PendingDisconnect
		if err := rows.Scan(&pd.ID, &pd.UserID, &pd.UserEmail, &pd.GatewayID, &pd.NodeType,
			&pd.ConnectionID, &pd.VPNAddress, &pd.Reason, &pd.RequestedBy, &pd.RequestedAt,
			&pd.ExpiresAt, &pd.CreatedAt); err != nil {
			return nil, err
		}
		disconnects = append(disconnects, &pd)
	}
	return disconnects, rows.Err()
}

// GetPendingDisconnectsForUser retrieves all pending disconnects for a specific user.
func (s *PendingDisconnectStore) GetPendingDisconnectsForUser(ctx context.Context, userID string) ([]*PendingDisconnect, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, user_email, gateway_id, node_type, connection_id, vpn_address,
		       reason, requested_by, requested_at, executed_at, executed_by_gateway, expires_at, created_at
		FROM pending_disconnects
		WHERE user_id = $1
		ORDER BY requested_at DESC
		LIMIT 100
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var disconnects []*PendingDisconnect
	for rows.Next() {
		var pd PendingDisconnect
		if err := rows.Scan(&pd.ID, &pd.UserID, &pd.UserEmail, &pd.GatewayID, &pd.NodeType,
			&pd.ConnectionID, &pd.VPNAddress, &pd.Reason, &pd.RequestedBy, &pd.RequestedAt,
			&pd.ExecutedAt, &pd.ExecutedBy, &pd.ExpiresAt, &pd.CreatedAt); err != nil {
			return nil, err
		}
		disconnects = append(disconnects, &pd)
	}
	return disconnects, rows.Err()
}

// MarkDisconnectExecuted marks a pending disconnect as executed by a gateway.
func (s *PendingDisconnectStore) MarkDisconnectExecuted(ctx context.Context, disconnectID string, gatewayID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE pending_disconnects
		SET executed_at = NOW(), executed_by_gateway = $2
		WHERE id = $1
	`, disconnectID, gatewayID)
	return err
}

// MarkConnectionDisconnectRequested updates the connection record to show disconnect was requested.
func (s *PendingDisconnectStore) MarkConnectionDisconnectRequested(ctx context.Context, connectionID string, requestedBy string, isGateway bool) error {
	table := "gateway_connections"
	if !isGateway {
		table = "mesh_connections"
	}
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE `+table+`
		SET disconnect_requested_at = NOW(), disconnect_requested_by = $2
		WHERE id = $1
	`, connectionID, requestedBy)
	return err
}

// DisconnectUserConnection immediately marks a connection as disconnected by admin.
// This updates the connection record so it shows as disconnected in the UI.
func (s *PendingDisconnectStore) DisconnectUserConnection(ctx context.Context, connectionID string, reason string, isGateway bool) error {
	table := "gateway_connections"
	if !isGateway {
		table = "mesh_connections"
	}
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE `+table+`
		SET disconnected_at = NOW(), disconnect_reason = $2
		WHERE id = $1 AND disconnected_at IS NULL
	`, connectionID, reason)
	return err
}

// DisconnectAllUserConnections disconnects all active connections for a specific user.
// This is for immediate revocation - targets individual user only.
func (s *PendingDisconnectStore) DisconnectAllUserConnections(ctx context.Context, userID string, reason string) (int64, error) {
	// Disconnect from gateways
	result1, err := s.db.Pool.Exec(ctx, `
		UPDATE gateway_connections
		SET disconnected_at = NOW(), disconnect_reason = $2
		WHERE user_id = $1 AND disconnected_at IS NULL
	`, userID, reason)
	if err != nil {
		return 0, err
	}
	count1 := result1.RowsAffected()

	// Disconnect from mesh hubs
	result2, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_connections
		SET disconnected_at = NOW(), disconnect_reason = $2
		WHERE user_id = $1 AND disconnected_at IS NULL
	`, userID, reason)
	if err != nil {
		return count1, err
	}
	count2 := result2.RowsAffected()

	return count1 + count2, nil
}

// CleanupExpiredDisconnects removes expired disconnect requests.
func (s *PendingDisconnectStore) CleanupExpiredDisconnects(ctx context.Context) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `
		DELETE FROM pending_disconnects
		WHERE expires_at < NOW() OR (executed_at IS NOT NULL AND executed_at < NOW() - INTERVAL '1 hour')
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// GetActiveConnectionsForUser returns active connections for a specific user.
func (s *PendingDisconnectStore) GetActiveConnectionsForUser(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT
			gc.id, gc.gateway_id, gc.client_ip, gc.vpn_ipv4, gc.connected_at,
			g.name as gateway_name, 'gateway' as node_type
		FROM gateway_connections gc
		JOIN gateways g ON gc.gateway_id = g.id
		WHERE gc.user_id = $1 AND gc.disconnected_at IS NULL
		UNION ALL
		SELECT
			mc.id, mc.hub_id as gateway_id, mc.client_ip, mc.vpn_ipv4, mc.connected_at,
			mh.name as gateway_name, 'hub' as node_type
		FROM mesh_connections mc
		JOIN mesh_hubs mh ON mc.hub_id = mh.id
		WHERE mc.user_id = $1 AND mc.disconnected_at IS NULL
		ORDER BY connected_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []map[string]interface{}
	for rows.Next() {
		var id, gatewayID, clientIP, vpnIP, gatewayName, nodeType string
		var connectedAt time.Time
		if err := rows.Scan(&id, &gatewayID, &clientIP, &vpnIP, &connectedAt, &gatewayName, &nodeType); err != nil {
			return nil, err
		}
		connections = append(connections, map[string]interface{}{
			"id":           id,
			"gateway_id":   gatewayID,
			"gateway_name": gatewayName,
			"client_ip":    clientIP,
			"vpn_address":  vpnIP,
			"connected_at": connectedAt,
			"node_type":    nodeType,
		})
	}
	return connections, rows.Err()
}

// CancelPendingDisconnect cancels a pending disconnect request (if not yet executed).
func (s *PendingDisconnectStore) CancelPendingDisconnect(ctx context.Context, disconnectID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM pending_disconnects
		WHERE id = $1 AND executed_at IS NULL
	`, disconnectID)
	return err
}
