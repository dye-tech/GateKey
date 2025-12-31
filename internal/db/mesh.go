package db

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrMeshHubNotFound   = errors.New("mesh hub not found")
	ErrMeshHubExists     = errors.New("mesh hub already exists")
	ErrMeshSpokeNotFound = errors.New("mesh spoke not found")
	ErrMeshSpokeExists   = errors.New("mesh spoke already exists")
)

// MeshHub status constants
const (
	MeshHubStatusPending = "pending"
	MeshHubStatusOnline  = "online"
	MeshHubStatusOffline = "offline"
	MeshHubStatusError   = "error"
)

// MeshSpoke status constants
const (
	MeshSpokeStatusPending      = "pending"
	MeshSpokeStatusConnected    = "connected"
	MeshSpokeStatusDisconnected = "disconnected"
	MeshSpokeStatusError        = "error"
)

// MeshHub represents a standalone hub server that accepts mesh gateway connections
type MeshHub struct {
	ID          string
	Name        string
	Description string

	// Endpoint configuration
	PublicEndpoint string // hostname:port for gateways to connect to
	VPNPort        int
	VPNProtocol    string
	VPNSubnet      string // Mesh network subnet (e.g., 172.30.0.0/16)

	// Crypto configuration
	CryptoProfile  string
	TLSAuthEnabled bool
	TLSAuthKey     string

	// PKI - Hub's own CA for mesh
	CACert     string
	CAKey      string
	ServerCert string
	ServerKey  string
	DHParams   string

	// Control plane communication
	APIToken        string // Token for hub to authenticate with control plane
	ControlPlaneURL string // URL of the GateKey control plane

	// Status
	Status          string
	StatusMessage   string
	LastHeartbeat   *time.Time
	ConnectedSpokes int
	ConnectedClients int

	// Config versioning
	ConfigVersion string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// MeshSpoke represents a remote spoke that connects TO a hub
type MeshSpoke struct {
	ID          string
	HubID       string
	Name        string
	Description string

	// Networks behind this gateway
	LocalNetworks []string // Array of CIDRs

	// Assigned tunnel IP
	TunnelIP string

	// Client certificates for connecting to hub
	ClientCert string
	ClientKey  string

	// Authentication
	Token string

	// Status
	Status        string
	StatusMessage string
	LastSeen      *time.Time
	BytesSent     int64
	BytesReceived int64

	// Remote public IP when connected
	RemoteIP string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// MeshConnection represents a user connected to the mesh hub
type MeshConnection struct {
	ID               string
	HubID            string
	UserID           string
	ClientIP         string
	TunnelIP         string
	BytesSent        int64
	BytesReceived    int64
	ConnectedAt      time.Time
	DisconnectedAt   *time.Time
	DisconnectReason string
}

// MeshStore handles mesh hub and gateway persistence
type MeshStore struct {
	db *DB
}

// NewMeshStore creates a new mesh store
func NewMeshStore(db *DB) *MeshStore {
	return &MeshStore{db: db}
}

// GenerateMeshToken generates a secure random token for mesh authentication
func GenerateMeshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ==================== Hub Operations ====================

// CreateHub creates a new mesh hub
func (s *MeshStore) CreateHub(ctx context.Context, hub *MeshHub) error {
	// Default values
	if hub.VPNPort == 0 {
		hub.VPNPort = 1194
	}
	if hub.VPNProtocol == "" {
		hub.VPNProtocol = "udp"
	}
	if hub.VPNSubnet == "" {
		hub.VPNSubnet = "172.30.0.0/16"
	}
	if hub.CryptoProfile == "" {
		hub.CryptoProfile = CryptoProfileFIPS
	}
	if hub.Status == "" {
		hub.Status = MeshHubStatusPending
	}

	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO mesh_hubs (
			name, description, public_endpoint, vpn_port, vpn_protocol, vpn_subnet,
			crypto_profile, tls_auth_enabled, tls_auth_key,
			ca_cert, ca_key, server_cert, server_key, dh_params,
			api_token, control_plane_url, status, status_message
		) VALUES (
			$1, $2, $3, $4, $5, $6::cidr,
			$7, $8, $9,
			$10, $11, $12, $13, $14,
			$15, $16, $17, $18
		)
	`, hub.Name, hub.Description, hub.PublicEndpoint, hub.VPNPort, hub.VPNProtocol, hub.VPNSubnet,
		hub.CryptoProfile, hub.TLSAuthEnabled, hub.TLSAuthKey,
		hub.CACert, hub.CAKey, hub.ServerCert, hub.ServerKey, hub.DHParams,
		hub.APIToken, hub.ControlPlaneURL, hub.Status, hub.StatusMessage)

	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return ErrMeshHubExists
	}
	return err
}

// GetHub retrieves a mesh hub by ID
func (s *MeshStore) GetHub(ctx context.Context, id string) (*MeshHub, error) {
	var hub MeshHub
	var vpnSubnet *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, description,
			public_endpoint, vpn_port, vpn_protocol, vpn_subnet::text,
			crypto_profile, tls_auth_enabled, COALESCE(tls_auth_key, ''),
			COALESCE(ca_cert, ''), COALESCE(ca_key, ''), COALESCE(server_cert, ''), COALESCE(server_key, ''), COALESCE(dh_params, ''),
			api_token, control_plane_url,
			status, COALESCE(status_message, ''), last_heartbeat, connected_gateways, connected_clients,
			COALESCE(config_version, ''),
			created_at, updated_at
		FROM mesh_hubs WHERE id = $1
	`, id).Scan(
		&hub.ID, &hub.Name, &hub.Description,
		&hub.PublicEndpoint, &hub.VPNPort, &hub.VPNProtocol, &vpnSubnet,
		&hub.CryptoProfile, &hub.TLSAuthEnabled, &hub.TLSAuthKey,
		&hub.CACert, &hub.CAKey, &hub.ServerCert, &hub.ServerKey, &hub.DHParams,
		&hub.APIToken, &hub.ControlPlaneURL,
		&hub.Status, &hub.StatusMessage, &hub.LastHeartbeat, &hub.ConnectedSpokes, &hub.ConnectedClients,
		&hub.ConfigVersion,
		&hub.CreatedAt, &hub.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrMeshHubNotFound
	}
	if err != nil {
		return nil, err
	}
	if vpnSubnet != nil {
		hub.VPNSubnet = *vpnSubnet
	}
	return &hub, nil
}

// GetHubByToken retrieves a mesh hub by its API token
func (s *MeshStore) GetHubByToken(ctx context.Context, token string) (*MeshHub, error) {
	var hub MeshHub
	var vpnSubnet *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, description,
			public_endpoint, vpn_port, vpn_protocol, vpn_subnet::text,
			crypto_profile, tls_auth_enabled, COALESCE(tls_auth_key, ''),
			COALESCE(ca_cert, ''), COALESCE(ca_key, ''), COALESCE(server_cert, ''), COALESCE(server_key, ''), COALESCE(dh_params, ''),
			api_token, control_plane_url,
			status, COALESCE(status_message, ''), last_heartbeat, connected_gateways, connected_clients,
			COALESCE(config_version, ''),
			created_at, updated_at
		FROM mesh_hubs WHERE api_token = $1
	`, token).Scan(
		&hub.ID, &hub.Name, &hub.Description,
		&hub.PublicEndpoint, &hub.VPNPort, &hub.VPNProtocol, &vpnSubnet,
		&hub.CryptoProfile, &hub.TLSAuthEnabled, &hub.TLSAuthKey,
		&hub.CACert, &hub.CAKey, &hub.ServerCert, &hub.ServerKey, &hub.DHParams,
		&hub.APIToken, &hub.ControlPlaneURL,
		&hub.Status, &hub.StatusMessage, &hub.LastHeartbeat, &hub.ConnectedSpokes, &hub.ConnectedClients,
		&hub.ConfigVersion,
		&hub.CreatedAt, &hub.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrMeshHubNotFound
	}
	if err != nil {
		return nil, err
	}
	if vpnSubnet != nil {
		hub.VPNSubnet = *vpnSubnet
	}
	return &hub, nil
}

// ListHubs retrieves all mesh hubs
func (s *MeshStore) ListHubs(ctx context.Context) ([]*MeshHub, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, name, description,
			public_endpoint, vpn_port, vpn_protocol, vpn_subnet::text,
			crypto_profile, tls_auth_enabled,
			status, COALESCE(status_message, ''), last_heartbeat, connected_gateways, connected_clients,
			created_at, updated_at
		FROM mesh_hubs
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hubs []*MeshHub
	for rows.Next() {
		var hub MeshHub
		var vpnSubnet *string
		if err := rows.Scan(
			&hub.ID, &hub.Name, &hub.Description,
			&hub.PublicEndpoint, &hub.VPNPort, &hub.VPNProtocol, &vpnSubnet,
			&hub.CryptoProfile, &hub.TLSAuthEnabled,
			&hub.Status, &hub.StatusMessage, &hub.LastHeartbeat, &hub.ConnectedSpokes, &hub.ConnectedClients,
			&hub.CreatedAt, &hub.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if vpnSubnet != nil {
			hub.VPNSubnet = *vpnSubnet
		}
		hubs = append(hubs, &hub)
	}
	return hubs, rows.Err()
}

// UpdateHub updates a mesh hub
func (s *MeshStore) UpdateHub(ctx context.Context, hub *MeshHub) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_hubs SET
			name = $2, description = $3,
			public_endpoint = $4, vpn_port = $5, vpn_protocol = $6, vpn_subnet = $7::cidr,
			crypto_profile = $8, tls_auth_enabled = $9
		WHERE id = $1
	`, hub.ID, hub.Name, hub.Description,
		hub.PublicEndpoint, hub.VPNPort, hub.VPNProtocol, hub.VPNSubnet,
		hub.CryptoProfile, hub.TLSAuthEnabled)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrMeshHubExists
		}
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrMeshHubNotFound
	}
	return nil
}

// UpdateHubPKI updates the PKI certificates for a hub
func (s *MeshStore) UpdateHubPKI(ctx context.Context, hubID string, caCert, caKey, serverCert, serverKey, dhParams, tlsAuthKey string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_hubs SET
			ca_cert = $2, ca_key = $3, server_cert = $4, server_key = $5, dh_params = $6, tls_auth_key = $7,
			updated_at = NOW()
		WHERE id = $1
	`, hubID, caCert, caKey, serverCert, serverKey, dhParams, tlsAuthKey)
	return err
}

// UpdateHubStatus updates the status of a mesh hub (from heartbeat)
func (s *MeshStore) UpdateHubStatus(ctx context.Context, hubID string, status, statusMessage string, connectedGateways, connectedClients int) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_hubs SET
			status = $2, status_message = $3, last_heartbeat = NOW(),
			connected_gateways = $4, connected_clients = $5
		WHERE id = $1
	`, hubID, status, statusMessage, connectedGateways, connectedClients)
	return err
}

// DeleteHub deletes a mesh hub and all associated gateways
func (s *MeshStore) DeleteHub(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM mesh_hubs WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrMeshHubNotFound
	}
	return nil
}

// MarkInactiveHubs marks hubs as offline if they haven't sent a heartbeat recently
func (s *MeshStore) MarkInactiveHubs(ctx context.Context, threshold time.Duration) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_hubs SET status = 'offline'
		WHERE status = 'online' AND (last_heartbeat IS NULL OR last_heartbeat < NOW() - $1::interval)
	`, threshold.String())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// ==================== Gateway Operations ====================

// CreateMeshSpoke creates a new mesh gateway
func (s *MeshStore) CreateMeshSpoke(ctx context.Context, gw *MeshSpoke) error {
	if gw.Status == "" {
		gw.Status = MeshSpokeStatusPending
	}

	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO mesh_gateways (
			hub_id, name, description, local_networks,
			client_cert, client_key, token,
			status, status_message
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9
		)
	`, gw.HubID, gw.Name, gw.Description, gw.LocalNetworks,
		gw.ClientCert, gw.ClientKey, gw.Token,
		gw.Status, gw.StatusMessage)

	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return ErrMeshSpokeExists
	}
	return err
}

// GetMeshSpoke retrieves a mesh gateway by ID
func (s *MeshStore) GetMeshSpoke(ctx context.Context, id string) (*MeshSpoke, error) {
	var gw MeshSpoke
	var tunnelIP, remoteIP *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, hub_id, name, description, local_networks,
			host(tunnel_ip), COALESCE(client_cert, ''), COALESCE(client_key, ''), token,
			status, COALESCE(status_message, ''), last_seen, bytes_sent, bytes_received,
			host(remote_ip),
			created_at, updated_at
		FROM mesh_gateways WHERE id = $1
	`, id).Scan(
		&gw.ID, &gw.HubID, &gw.Name, &gw.Description, &gw.LocalNetworks,
		&tunnelIP, &gw.ClientCert, &gw.ClientKey, &gw.Token,
		&gw.Status, &gw.StatusMessage, &gw.LastSeen, &gw.BytesSent, &gw.BytesReceived,
		&remoteIP,
		&gw.CreatedAt, &gw.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrMeshSpokeNotFound
	}
	if err != nil {
		return nil, err
	}
	if tunnelIP != nil {
		gw.TunnelIP = *tunnelIP
	}
	if remoteIP != nil {
		gw.RemoteIP = *remoteIP
	}
	return &gw, nil
}

// GetMeshSpokeByToken retrieves a mesh gateway by its token
func (s *MeshStore) GetMeshSpokeByToken(ctx context.Context, token string) (*MeshSpoke, error) {
	var gw MeshSpoke
	var tunnelIP, remoteIP *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, hub_id, name, description, local_networks,
			host(tunnel_ip), COALESCE(client_cert, ''), COALESCE(client_key, ''), token,
			status, COALESCE(status_message, ''), last_seen, bytes_sent, bytes_received,
			host(remote_ip),
			created_at, updated_at
		FROM mesh_gateways WHERE token = $1
	`, token).Scan(
		&gw.ID, &gw.HubID, &gw.Name, &gw.Description, &gw.LocalNetworks,
		&tunnelIP, &gw.ClientCert, &gw.ClientKey, &gw.Token,
		&gw.Status, &gw.StatusMessage, &gw.LastSeen, &gw.BytesSent, &gw.BytesReceived,
		&remoteIP,
		&gw.CreatedAt, &gw.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrMeshSpokeNotFound
	}
	if err != nil {
		return nil, err
	}
	if tunnelIP != nil {
		gw.TunnelIP = *tunnelIP
	}
	if remoteIP != nil {
		gw.RemoteIP = *remoteIP
	}
	return &gw, nil
}

// ListMeshSpokesByHub retrieves all mesh gateways for a specific hub
func (s *MeshStore) ListMeshSpokesByHub(ctx context.Context, hubID string) ([]*MeshSpoke, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, hub_id, name, description, local_networks,
			host(tunnel_ip), status, COALESCE(status_message, ''), last_seen,
			bytes_sent, bytes_received, host(remote_ip),
			created_at, updated_at
		FROM mesh_gateways
		WHERE hub_id = $1
		ORDER BY name
	`, hubID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gateways []*MeshSpoke
	for rows.Next() {
		var gw MeshSpoke
		var tunnelIP, remoteIP *string
		if err := rows.Scan(
			&gw.ID, &gw.HubID, &gw.Name, &gw.Description, &gw.LocalNetworks,
			&tunnelIP, &gw.Status, &gw.StatusMessage, &gw.LastSeen,
			&gw.BytesSent, &gw.BytesReceived, &remoteIP,
			&gw.CreatedAt, &gw.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if tunnelIP != nil {
			gw.TunnelIP = *tunnelIP
		}
		if remoteIP != nil {
			gw.RemoteIP = *remoteIP
		}
		gateways = append(gateways, &gw)
	}
	return gateways, rows.Err()
}

// UpdateMeshSpoke updates a mesh gateway
func (s *MeshStore) UpdateMeshSpoke(ctx context.Context, gw *MeshSpoke) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_gateways SET
			name = $2, description = $3, local_networks = $4
		WHERE id = $1
	`, gw.ID, gw.Name, gw.Description, gw.LocalNetworks)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrMeshSpokeExists
		}
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrMeshSpokeNotFound
	}
	return nil
}

// UpdateMeshSpokePKI updates the client certificates for a mesh gateway
func (s *MeshStore) UpdateMeshSpokePKI(ctx context.Context, gwID, clientCert, clientKey, tunnelIP string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_gateways SET
			client_cert = $2, client_key = $3, tunnel_ip = NULLIF($4, '')::inet,
			updated_at = NOW()
		WHERE id = $1
	`, gwID, clientCert, clientKey, tunnelIP)
	return err
}

// UpdateMeshSpokeStatus updates the status of a mesh gateway
func (s *MeshStore) UpdateMeshSpokeStatus(ctx context.Context, gwID, status, statusMessage, remoteIP string, bytesSent, bytesReceived int64) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_gateways SET
			status = $2, status_message = $3, last_seen = NOW(),
			remote_ip = NULLIF($4, '')::inet, bytes_sent = $5, bytes_received = $6
		WHERE id = $1
	`, gwID, status, statusMessage, remoteIP, bytesSent, bytesReceived)
	return err
}

// DeleteMeshSpoke deletes a mesh gateway
func (s *MeshStore) DeleteMeshSpoke(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM mesh_gateways WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrMeshSpokeNotFound
	}
	return nil
}

// MarkInactiveGateways marks mesh gateways as disconnected if they haven't reported recently
func (s *MeshStore) MarkInactiveMeshSpokes(ctx context.Context, threshold time.Duration) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_gateways SET status = 'disconnected'
		WHERE status = 'connected' AND (last_seen IS NULL OR last_seen < NOW() - $1::interval)
	`, threshold.String())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// ==================== Access Control Operations ====================

// AssignUserToHub assigns a user to a mesh hub
func (s *MeshStore) AssignUserToHub(ctx context.Context, hubID, userID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO mesh_hub_users (hub_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, hubID, userID)
	return err
}

// RemoveUserFromHub removes a user from a mesh hub
func (s *MeshStore) RemoveUserFromHub(ctx context.Context, hubID, userID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM mesh_hub_users WHERE hub_id = $1 AND user_id = $2
	`, hubID, userID)
	return err
}

// AssignGroupToHub assigns a group to a mesh hub
func (s *MeshStore) AssignGroupToHub(ctx context.Context, hubID, groupName string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO mesh_hub_groups (hub_id, group_name)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, hubID, groupName)
	return err
}

// RemoveGroupFromHub removes a group from a mesh hub
func (s *MeshStore) RemoveGroupFromHub(ctx context.Context, hubID, groupName string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM mesh_hub_groups WHERE hub_id = $1 AND group_name = $2
	`, hubID, groupName)
	return err
}

// GetHubUsers returns all users assigned to a hub
func (s *MeshStore) GetHubUsers(ctx context.Context, hubID string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT user_id FROM mesh_hub_users WHERE hub_id = $1 ORDER BY user_id
	`, hubID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		users = append(users, userID)
	}
	return users, rows.Err()
}

// GetHubGroups returns all groups assigned to a hub
func (s *MeshStore) GetHubGroups(ctx context.Context, hubID string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT group_name FROM mesh_hub_groups WHERE hub_id = $1 ORDER BY group_name
	`, hubID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var groupName string
		if err := rows.Scan(&groupName); err != nil {
			return nil, err
		}
		groups = append(groups, groupName)
	}
	return groups, rows.Err()
}

// UserHasHubAccess checks if a user has access to a mesh hub
func (s *MeshStore) UserHasHubAccess(ctx context.Context, userID, hubID string, groups []string) (bool, error) {
	var hasAccess bool
	err := s.db.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM mesh_hub_users WHERE user_id = $1 AND hub_id = $2
			UNION
			SELECT 1 FROM mesh_hub_groups WHERE hub_id = $2 AND group_name = ANY($3)
		)
	`, userID, hubID, groups).Scan(&hasAccess)
	return hasAccess, err
}

// GetHubsForUser returns all mesh hubs that a user has access to
func (s *MeshStore) GetHubsForUser(ctx context.Context, userID string, groups []string) ([]*MeshHub, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT DISTINCT h.id, h.name, h.description,
			h.public_endpoint, h.vpn_port, h.vpn_protocol, h.vpn_subnet::text,
			h.crypto_profile, h.tls_auth_enabled,
			h.status, COALESCE(h.status_message, ''), h.last_heartbeat, h.connected_gateways, h.connected_clients,
			h.created_at, h.updated_at
		FROM mesh_hubs h
		WHERE EXISTS (
			SELECT 1 FROM mesh_hub_users WHERE user_id = $1 AND hub_id = h.id
			UNION
			SELECT 1 FROM mesh_hub_groups WHERE hub_id = h.id AND group_name = ANY($2)
		)
		ORDER BY h.name
	`, userID, groups)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hubs []*MeshHub
	for rows.Next() {
		var hub MeshHub
		var vpnSubnet *string
		if err := rows.Scan(
			&hub.ID, &hub.Name, &hub.Description,
			&hub.PublicEndpoint, &hub.VPNPort, &hub.VPNProtocol, &vpnSubnet,
			&hub.CryptoProfile, &hub.TLSAuthEnabled,
			&hub.Status, &hub.StatusMessage, &hub.LastHeartbeat, &hub.ConnectedSpokes, &hub.ConnectedClients,
			&hub.CreatedAt, &hub.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if vpnSubnet != nil {
			hub.VPNSubnet = *vpnSubnet
		}
		hubs = append(hubs, &hub)
	}
	return hubs, rows.Err()
}

// ==================== Route Aggregation ====================

// GetAllMeshRoutes returns all routes from all connected mesh gateways for a hub
func (s *MeshStore) GetAllMeshRoutes(ctx context.Context, hubID string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT DISTINCT unnest(local_networks) as network
		FROM mesh_gateways
		WHERE hub_id = $1 AND status = 'connected'
		ORDER BY network
	`, hubID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []string
	for rows.Next() {
		var network string
		if err := rows.Scan(&network); err != nil {
			return nil, err
		}
		routes = append(routes, network)
	}
	return routes, rows.Err()
}

// GetUserMeshRoutes returns routes a user can access based on their gateway assignments
func (s *MeshStore) GetUserMeshRoutes(ctx context.Context, hubID, userID string, groups []string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT DISTINCT unnest(g.local_networks) as network
		FROM mesh_gateways g
		WHERE g.hub_id = $1 AND g.status = 'connected'
		AND (
			EXISTS (SELECT 1 FROM mesh_gateway_users WHERE gateway_id = g.id AND user_id = $2)
			OR EXISTS (SELECT 1 FROM mesh_gateway_groups WHERE gateway_id = g.id AND group_name = ANY($3))
		)
		ORDER BY network
	`, hubID, userID, groups)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []string
	for rows.Next() {
		var network string
		if err := rows.Scan(&network); err != nil {
			return nil, err
		}
		routes = append(routes, network)
	}
	return routes, rows.Err()
}

// ==================== Spoke User/Group Access ====================

// AddUserToSpoke assigns a user to a mesh spoke
func (s *MeshStore) AddUserToSpoke(ctx context.Context, spokeID, userID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO mesh_gateway_users (gateway_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, spokeID, userID)
	return err
}

// RemoveUserFromSpoke removes a user from a mesh spoke
func (s *MeshStore) RemoveUserFromSpoke(ctx context.Context, spokeID, userID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM mesh_gateway_users WHERE gateway_id = $1 AND user_id = $2
	`, spokeID, userID)
	return err
}

// GetSpokeUsers returns all users assigned to a spoke
func (s *MeshStore) GetSpokeUsers(ctx context.Context, spokeID string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT user_id FROM mesh_gateway_users WHERE gateway_id = $1 ORDER BY user_id
	`, spokeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		users = append(users, userID)
	}
	return users, rows.Err()
}

// AddGroupToSpoke assigns a group to a mesh spoke
func (s *MeshStore) AddGroupToSpoke(ctx context.Context, spokeID, groupName string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO mesh_gateway_groups (gateway_id, group_name)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, spokeID, groupName)
	return err
}

// RemoveGroupFromSpoke removes a group from a mesh spoke
func (s *MeshStore) RemoveGroupFromSpoke(ctx context.Context, spokeID, groupName string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM mesh_gateway_groups WHERE gateway_id = $1 AND group_name = $2
	`, spokeID, groupName)
	return err
}

// GetSpokeGroups returns all groups assigned to a spoke
func (s *MeshStore) GetSpokeGroups(ctx context.Context, spokeID string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT group_name FROM mesh_gateway_groups WHERE gateway_id = $1 ORDER BY group_name
	`, spokeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var groupName string
		if err := rows.Scan(&groupName); err != nil {
			return nil, err
		}
		groups = append(groups, groupName)
	}
	return groups, rows.Err()
}
