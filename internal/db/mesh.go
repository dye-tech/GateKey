package db

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
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

// Mesh gateway type constants
const (
	MeshGatewayTypeOpenVPN   = "openvpn"
	MeshGatewayTypeWireGuard = "wireguard"
)

// MeshHub represents a standalone hub server that accepts mesh gateway connections
type MeshHub struct {
	ID          string
	Name        string
	Description string

	// Gateway type (openvpn or wireguard)
	GatewayType string // "openvpn" or "wireguard"

	// Endpoint configuration
	PublicEndpoint string // hostname:port for gateways to connect to
	VPNPort        int
	VPNProtocol    string
	VPNSubnet      string // Mesh network subnet (e.g., 172.30.0.0/16)

	// Local networks directly accessible from the hub (pushed to all clients)
	LocalNetworks []string

	// Crypto configuration (OpenVPN)
	CryptoProfile  string
	TLSAuthEnabled bool
	TLSAuthKey     string

	// WireGuard configuration
	WGPrivateKey string // WireGuard private key (when gateway_type=wireguard)
	WGPublicKey  string // WireGuard public key (when gateway_type=wireguard)
	WGListenPort int    // WireGuard listen port (default 51820)

	// VPN client configuration
	FullTunnelMode bool     // Route all client traffic through hub
	PushDNS        bool     // Push DNS servers to clients
	DNSServers     []string // DNS server IPs to push

	// PKI - Hub's own CA for mesh (OpenVPN)
	CACert     string
	CAKey      string
	ServerCert string
	ServerKey  string
	DHParams   string

	// Control plane communication
	APIToken        string // Token for hub to authenticate with control plane
	ControlPlaneURL string // URL of the GateKey control plane

	// Status
	Status           string
	StatusMessage    string
	LastHeartbeat    *time.Time
	ConnectedSpokes  int
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

	// Gateway type (inherited from hub)
	GatewayType string // "openvpn" or "wireguard"

	// Networks behind this gateway
	LocalNetworks []string // Array of CIDRs

	// VPN settings
	FullTunnelMode bool     // Route all traffic through the mesh
	PushDNS        bool     // Push DNS servers to clients
	DNSServers     []string // DNS server IPs to push

	// Assigned tunnel IP
	TunnelIP string

	// Client certificates for connecting to hub (OpenVPN)
	ClientCert string
	ClientKey  string

	// WireGuard keys (when gateway_type=wireguard)
	WGPrivateKey   string // Spoke's private key
	WGPublicKey    string // Spoke's public key
	WGPresharedKey string // Optional PSK for extra security

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
	if hub.GatewayType == "" {
		hub.GatewayType = MeshGatewayTypeOpenVPN
	}
	if hub.VPNPort == 0 {
		if hub.GatewayType == MeshGatewayTypeWireGuard {
			hub.VPNPort = 51820
		} else {
			hub.VPNPort = 1194
		}
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
	if hub.WGListenPort == 0 {
		hub.WGListenPort = 51820
	}

	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO mesh_hubs (
			name, description, public_endpoint, vpn_port, vpn_protocol, vpn_subnet,
			gateway_type, crypto_profile, tls_auth_enabled, tls_auth_key,
			wg_private_key, wg_public_key, wg_listen_port,
			ca_cert, ca_key, server_cert, server_key, dh_params,
			api_token, control_plane_url, status, status_message
		) VALUES (
			$1, $2, $3, $4, $5, $6::cidr,
			$7, $8, $9, $10,
			$11, $12, $13,
			$14, $15, $16, $17, $18,
			$19, $20, $21, $22
		)
	`, hub.Name, hub.Description, hub.PublicEndpoint, hub.VPNPort, hub.VPNProtocol, hub.VPNSubnet,
		hub.GatewayType, hub.CryptoProfile, hub.TLSAuthEnabled, hub.TLSAuthKey,
		hub.WGPrivateKey, hub.WGPublicKey, hub.WGListenPort,
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
	var wgListenPort *int
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, description,
			COALESCE(gateway_type, 'openvpn'),
			public_endpoint, vpn_port, vpn_protocol, vpn_subnet::text,
			COALESCE(local_networks, '{}'),
			crypto_profile, tls_auth_enabled, COALESCE(tls_auth_key, ''),
			COALESCE(wg_private_key, ''), COALESCE(wg_public_key, ''), wg_listen_port,
			COALESCE(full_tunnel_mode, false), COALESCE(push_dns, false), COALESCE(dns_servers, '{}'),
			COALESCE(ca_cert, ''), COALESCE(ca_key, ''), COALESCE(server_cert, ''), COALESCE(server_key, ''), COALESCE(dh_params, ''),
			api_token, control_plane_url,
			status, COALESCE(status_message, ''), last_heartbeat, connected_gateways, connected_clients,
			COALESCE(config_version, ''),
			created_at, updated_at
		FROM mesh_hubs WHERE id = $1
	`, id).Scan(
		&hub.ID, &hub.Name, &hub.Description,
		&hub.GatewayType,
		&hub.PublicEndpoint, &hub.VPNPort, &hub.VPNProtocol, &vpnSubnet,
		&hub.LocalNetworks,
		&hub.CryptoProfile, &hub.TLSAuthEnabled, &hub.TLSAuthKey,
		&hub.WGPrivateKey, &hub.WGPublicKey, &wgListenPort,
		&hub.FullTunnelMode, &hub.PushDNS, &hub.DNSServers,
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
	if wgListenPort != nil {
		hub.WGListenPort = *wgListenPort
	} else {
		hub.WGListenPort = 51820
	}
	return &hub, nil
}

// GetHubByToken retrieves a mesh hub by its API token
func (s *MeshStore) GetHubByToken(ctx context.Context, token string) (*MeshHub, error) {
	var hub MeshHub
	var vpnSubnet *string
	var wgListenPort *int
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, description,
			COALESCE(gateway_type, 'openvpn'),
			public_endpoint, vpn_port, vpn_protocol, vpn_subnet::text,
			COALESCE(local_networks, '{}'),
			crypto_profile, tls_auth_enabled, COALESCE(tls_auth_key, ''),
			COALESCE(wg_private_key, ''), COALESCE(wg_public_key, ''), wg_listen_port,
			COALESCE(full_tunnel_mode, false), COALESCE(push_dns, false), COALESCE(dns_servers, '{}'),
			COALESCE(ca_cert, ''), COALESCE(ca_key, ''), COALESCE(server_cert, ''), COALESCE(server_key, ''), COALESCE(dh_params, ''),
			api_token, control_plane_url,
			status, COALESCE(status_message, ''), last_heartbeat, connected_gateways, connected_clients,
			COALESCE(config_version, ''),
			created_at, updated_at
		FROM mesh_hubs WHERE api_token = $1
	`, token).Scan(
		&hub.ID, &hub.Name, &hub.Description,
		&hub.GatewayType,
		&hub.PublicEndpoint, &hub.VPNPort, &hub.VPNProtocol, &vpnSubnet,
		&hub.LocalNetworks,
		&hub.CryptoProfile, &hub.TLSAuthEnabled, &hub.TLSAuthKey,
		&hub.WGPrivateKey, &hub.WGPublicKey, &wgListenPort,
		&hub.FullTunnelMode, &hub.PushDNS, &hub.DNSServers,
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
	if wgListenPort != nil {
		hub.WGListenPort = *wgListenPort
	} else {
		hub.WGListenPort = 51820
	}
	return &hub, nil
}

// GetHubByName retrieves a mesh hub by name
func (s *MeshStore) GetHubByName(name string) (*MeshHub, error) {
	ctx := context.Background()
	var hub MeshHub
	var vpnSubnet *string
	var wgListenPort *int
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, description,
			COALESCE(gateway_type, 'openvpn'),
			public_endpoint, vpn_port, vpn_protocol, vpn_subnet::text,
			COALESCE(local_networks, '{}'),
			crypto_profile, tls_auth_enabled, COALESCE(tls_auth_key, ''),
			COALESCE(wg_private_key, ''), COALESCE(wg_public_key, ''), wg_listen_port,
			COALESCE(full_tunnel_mode, false), COALESCE(push_dns, false), COALESCE(dns_servers, '{}'),
			COALESCE(ca_cert, ''), COALESCE(ca_key, ''), COALESCE(server_cert, ''), COALESCE(server_key, ''), COALESCE(dh_params, ''),
			api_token, control_plane_url,
			status, COALESCE(status_message, ''), last_heartbeat, connected_gateways, connected_clients,
			COALESCE(config_version, ''),
			created_at, updated_at
		FROM mesh_hubs WHERE name = $1
	`, name).Scan(
		&hub.ID, &hub.Name, &hub.Description,
		&hub.GatewayType,
		&hub.PublicEndpoint, &hub.VPNPort, &hub.VPNProtocol, &vpnSubnet,
		&hub.LocalNetworks,
		&hub.CryptoProfile, &hub.TLSAuthEnabled, &hub.TLSAuthKey,
		&hub.WGPrivateKey, &hub.WGPublicKey, &wgListenPort,
		&hub.FullTunnelMode, &hub.PushDNS, &hub.DNSServers,
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
	if wgListenPort != nil {
		hub.WGListenPort = *wgListenPort
	} else {
		hub.WGListenPort = 51820
	}
	return &hub, nil
}

// ListHubs retrieves all mesh hubs
func (s *MeshStore) ListHubs(ctx context.Context) ([]*MeshHub, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, name, description,
			COALESCE(gateway_type, 'openvpn'),
			public_endpoint, vpn_port, vpn_protocol, vpn_subnet::text,
			crypto_profile, tls_auth_enabled,
			COALESCE(wg_public_key, ''), wg_listen_port,
			COALESCE(full_tunnel_mode, false), COALESCE(push_dns, false), COALESCE(dns_servers, '{}'),
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
		var wgListenPort *int
		if err := rows.Scan(
			&hub.ID, &hub.Name, &hub.Description,
			&hub.GatewayType,
			&hub.PublicEndpoint, &hub.VPNPort, &hub.VPNProtocol, &vpnSubnet,
			&hub.CryptoProfile, &hub.TLSAuthEnabled,
			&hub.WGPublicKey, &wgListenPort,
			&hub.FullTunnelMode, &hub.PushDNS, &hub.DNSServers,
			&hub.Status, &hub.StatusMessage, &hub.LastHeartbeat, &hub.ConnectedSpokes, &hub.ConnectedClients,
			&hub.CreatedAt, &hub.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if vpnSubnet != nil {
			hub.VPNSubnet = *vpnSubnet
		}
		if wgListenPort != nil {
			hub.WGListenPort = *wgListenPort
		} else {
			hub.WGListenPort = 51820
		}
		hubs = append(hubs, &hub)
	}
	return hubs, rows.Err()
}

// UpdateHub updates a mesh hub
// Note: gateway_type cannot be changed after creation
func (s *MeshStore) UpdateHub(ctx context.Context, hub *MeshHub) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_hubs SET
			name = $2, description = $3,
			public_endpoint = $4, vpn_port = $5, vpn_protocol = $6, vpn_subnet = $7::cidr,
			crypto_profile = $8, tls_auth_enabled = $9, local_networks = $10,
			wg_listen_port = $11,
			full_tunnel_mode = $12, push_dns = $13, dns_servers = $14
		WHERE id = $1
	`, hub.ID, hub.Name, hub.Description,
		hub.PublicEndpoint, hub.VPNPort, hub.VPNProtocol, hub.VPNSubnet,
		hub.CryptoProfile, hub.TLSAuthEnabled, hub.LocalNetworks,
		hub.WGListenPort,
		hub.FullTunnelMode, hub.PushDNS, hub.DNSServers)

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

// UpdateHubPKI updates the PKI certificates for a hub (OpenVPN)
func (s *MeshStore) UpdateHubPKI(ctx context.Context, hubID string, caCert, caKey, serverCert, serverKey, dhParams, tlsAuthKey string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_hubs SET
			ca_cert = $2, ca_key = $3, server_cert = $4, server_key = $5, dh_params = $6, tls_auth_key = $7,
			updated_at = NOW()
		WHERE id = $1
	`, hubID, caCert, caKey, serverCert, serverKey, dhParams, tlsAuthKey)
	return err
}

// UpdateHubWireGuardKeys updates the WireGuard keys for a hub
func (s *MeshStore) UpdateHubWireGuardKeys(ctx context.Context, hubID string, privateKey, publicKey string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_hubs SET
			wg_private_key = $2, wg_public_key = $3,
			updated_at = NOW()
		WHERE id = $1
	`, hubID, privateKey, publicKey)
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

// RotateHubToken generates a new API token for a mesh hub and returns it
// The old token is immediately invalidated
func (s *MeshStore) RotateHubToken(ctx context.Context, id string) (string, error) {
	// Generate new token
	newToken, err := GenerateMeshToken()
	if err != nil {
		return "", err
	}

	result, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_hubs
		SET api_token = $2, updated_at = NOW()
		WHERE id = $1
	`, id, newToken)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() == 0 {
		return "", ErrMeshHubNotFound
	}
	return newToken, nil
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
	if gw.GatewayType == "" {
		gw.GatewayType = MeshGatewayTypeOpenVPN
	}

	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO mesh_gateways (
			hub_id, name, description, local_networks,
			gateway_type,
			client_cert, client_key,
			wg_private_key, wg_public_key, wg_preshared_key,
			token,
			status, status_message
		) VALUES (
			$1, $2, $3, $4,
			$5,
			$6, $7,
			$8, $9, $10,
			$11,
			$12, $13
		)
	`, gw.HubID, gw.Name, gw.Description, gw.LocalNetworks,
		gw.GatewayType,
		gw.ClientCert, gw.ClientKey,
		gw.WGPrivateKey, gw.WGPublicKey, gw.WGPresharedKey,
		gw.Token,
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
			COALESCE(gateway_type, 'openvpn'),
			COALESCE(full_tunnel_mode, false), COALESCE(push_dns, false), COALESCE(dns_servers, '{}'),
			host(tunnel_ip), COALESCE(client_cert, ''), COALESCE(client_key, ''),
			COALESCE(wg_private_key, ''), COALESCE(wg_public_key, ''), COALESCE(wg_preshared_key, ''),
			token,
			status, COALESCE(status_message, ''), last_seen, bytes_sent, bytes_received,
			host(remote_ip),
			created_at, updated_at
		FROM mesh_gateways WHERE id = $1
	`, id).Scan(
		&gw.ID, &gw.HubID, &gw.Name, &gw.Description, &gw.LocalNetworks,
		&gw.GatewayType,
		&gw.FullTunnelMode, &gw.PushDNS, &gw.DNSServers,
		&tunnelIP, &gw.ClientCert, &gw.ClientKey,
		&gw.WGPrivateKey, &gw.WGPublicKey, &gw.WGPresharedKey,
		&gw.Token,
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
			COALESCE(gateway_type, 'openvpn'),
			COALESCE(full_tunnel_mode, false), COALESCE(push_dns, false), COALESCE(dns_servers, '{}'),
			host(tunnel_ip), COALESCE(client_cert, ''), COALESCE(client_key, ''),
			COALESCE(wg_private_key, ''), COALESCE(wg_public_key, ''), COALESCE(wg_preshared_key, ''),
			token,
			status, COALESCE(status_message, ''), last_seen, bytes_sent, bytes_received,
			host(remote_ip),
			created_at, updated_at
		FROM mesh_gateways WHERE token = $1
	`, token).Scan(
		&gw.ID, &gw.HubID, &gw.Name, &gw.Description, &gw.LocalNetworks,
		&gw.GatewayType,
		&gw.FullTunnelMode, &gw.PushDNS, &gw.DNSServers,
		&tunnelIP, &gw.ClientCert, &gw.ClientKey,
		&gw.WGPrivateKey, &gw.WGPublicKey, &gw.WGPresharedKey,
		&gw.Token,
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

// GetMeshSpokeByName retrieves a mesh spoke by name
func (s *MeshStore) GetMeshSpokeByName(name string) (*MeshSpoke, error) {
	ctx := context.Background()
	var gw MeshSpoke
	var tunnelIP, remoteIP *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, hub_id, name, description, local_networks,
			COALESCE(gateway_type, 'openvpn'),
			COALESCE(full_tunnel_mode, false), COALESCE(push_dns, false), COALESCE(dns_servers, '{}'),
			host(tunnel_ip), COALESCE(client_cert, ''), COALESCE(client_key, ''),
			COALESCE(wg_private_key, ''), COALESCE(wg_public_key, ''), COALESCE(wg_preshared_key, ''),
			token,
			status, COALESCE(status_message, ''), last_seen, bytes_sent, bytes_received,
			host(remote_ip),
			created_at, updated_at
		FROM mesh_gateways WHERE name = $1
	`, name).Scan(
		&gw.ID, &gw.HubID, &gw.Name, &gw.Description, &gw.LocalNetworks,
		&gw.GatewayType,
		&gw.FullTunnelMode, &gw.PushDNS, &gw.DNSServers,
		&tunnelIP, &gw.ClientCert, &gw.ClientKey,
		&gw.WGPrivateKey, &gw.WGPublicKey, &gw.WGPresharedKey,
		&gw.Token,
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
			COALESCE(gateway_type, 'openvpn'),
			COALESCE(full_tunnel_mode, false), COALESCE(push_dns, false), COALESCE(dns_servers, '{}'),
			host(tunnel_ip), COALESCE(wg_public_key, ''), COALESCE(wg_preshared_key, ''),
			status, COALESCE(status_message, ''), last_seen,
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
			&gw.GatewayType,
			&gw.FullTunnelMode, &gw.PushDNS, &gw.DNSServers,
			&tunnelIP, &gw.WGPublicKey, &gw.WGPresharedKey,
			&gw.Status, &gw.StatusMessage, &gw.LastSeen,
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
			name = $2, description = $3, local_networks = $4,
			full_tunnel_mode = $5, push_dns = $6, dns_servers = $7
		WHERE id = $1
	`, gw.ID, gw.Name, gw.Description, gw.LocalNetworks,
		gw.FullTunnelMode, gw.PushDNS, gw.DNSServers)

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

// UpdateMeshSpokePKI updates the client certificates for a mesh gateway (OpenVPN)
func (s *MeshStore) UpdateMeshSpokePKI(ctx context.Context, gwID, clientCert, clientKey, tunnelIP string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_gateways SET
			client_cert = $2, client_key = $3, tunnel_ip = NULLIF($4, '')::inet,
			updated_at = NOW()
		WHERE id = $1
	`, gwID, clientCert, clientKey, tunnelIP)
	return err
}

// UpdateMeshSpokeWireGuardKeys updates the WireGuard keys for a spoke
func (s *MeshStore) UpdateMeshSpokeWireGuardKeys(ctx context.Context, gwID, privateKey, publicKey, presharedKey, tunnelIP string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_gateways SET
			wg_private_key = $2, wg_public_key = $3, wg_preshared_key = $4,
			tunnel_ip = NULLIF($5, '')::inet,
			updated_at = NOW()
		WHERE id = $1
	`, gwID, privateKey, publicKey, presharedKey, tunnelIP)
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

// RotateSpokeToken generates a new token for a mesh spoke and returns it
// The old token is immediately invalidated
func (s *MeshStore) RotateSpokeToken(ctx context.Context, id string) (string, error) {
	// Generate new token
	newToken, err := GenerateMeshToken()
	if err != nil {
		return "", err
	}

	result, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_gateways
		SET token = $2, updated_at = NOW()
		WHERE id = $1
	`, id, newToken)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() == 0 {
		return "", ErrMeshSpokeNotFound
	}
	return newToken, nil
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
			COALESCE(h.gateway_type, 'openvpn'),
			h.public_endpoint, h.vpn_port, h.vpn_protocol, h.vpn_subnet::text,
			h.crypto_profile, h.tls_auth_enabled,
			COALESCE(h.wg_public_key, ''), h.wg_listen_port,
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
		var wgListenPort *int
		if err := rows.Scan(
			&hub.ID, &hub.Name, &hub.Description,
			&hub.GatewayType,
			&hub.PublicEndpoint, &hub.VPNPort, &hub.VPNProtocol, &vpnSubnet,
			&hub.CryptoProfile, &hub.TLSAuthEnabled,
			&hub.WGPublicKey, &wgListenPort,
			&hub.Status, &hub.StatusMessage, &hub.LastHeartbeat, &hub.ConnectedSpokes, &hub.ConnectedClients,
			&hub.CreatedAt, &hub.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if vpnSubnet != nil {
			hub.VPNSubnet = *vpnSubnet
		}
		if wgListenPort != nil {
			hub.WGListenPort = *wgListenPort
		} else {
			hub.WGListenPort = 51820
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
// Also includes hub's local_networks (networks directly reachable from the hub)
func (s *MeshStore) GetUserMeshRoutes(ctx context.Context, hubID, userID string, groups []string) ([]string, error) {
	// Get routes from spokes the user has access to
	rows, err := s.db.Pool.Query(ctx, `
		SELECT DISTINCT network FROM (
			-- Spoke networks the user has access to
			SELECT unnest(g.local_networks) as network
			FROM mesh_gateways g
			WHERE g.hub_id = $1 AND g.status = 'connected'
			AND (
				EXISTS (SELECT 1 FROM mesh_gateway_users WHERE gateway_id = g.id AND user_id = $2)
				OR EXISTS (SELECT 1 FROM mesh_gateway_groups WHERE gateway_id = g.id AND group_name = ANY($3))
			)
			UNION
			-- Hub's local networks (accessible to all users with hub access)
			SELECT unnest(COALESCE(local_networks, '{}')) as network
			FROM mesh_hubs
			WHERE id = $1
		) combined
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

// ==================== Hub Network Assignment ====================

// AssignNetworkToHub assigns a network to a mesh hub
func (s *MeshStore) AssignNetworkToHub(ctx context.Context, hubID, networkID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO mesh_hub_networks (hub_id, network_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, hubID, networkID)
	return err
}

// RemoveNetworkFromHub removes a network from a mesh hub
func (s *MeshStore) RemoveNetworkFromHub(ctx context.Context, hubID, networkID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM mesh_hub_networks WHERE hub_id = $1 AND network_id = $2
	`, hubID, networkID)
	return err
}

// GetHubNetworks returns all networks assigned to a mesh hub
func (s *MeshStore) GetHubNetworks(ctx context.Context, hubID string) ([]*Network, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT n.id, n.name, n.description, n.cidr::text, n.is_active, n.created_at, n.updated_at
		FROM networks n
		JOIN mesh_hub_networks hn ON n.id = hn.network_id
		WHERE hn.hub_id = $1
		ORDER BY n.name
	`, hubID)
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

// GetUserMeshAccessRules returns access rules a user has through hub networks
// Zero-trust: Only routes defined in access rules the user is assigned to are allowed
// Returns both CIDR and IP rules (IP rules are converted to /32 CIDR)
func (s *MeshStore) GetUserMeshAccessRules(ctx context.Context, hubID, userID string, groups []string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT DISTINCT
			CASE
				WHEN ar.rule_type = 'ip' THEN ar.value || '/32'
				ELSE ar.value
			END as route
		FROM access_rules ar
		JOIN mesh_hub_networks hn ON ar.network_id = hn.network_id
		WHERE hn.hub_id = $1
		AND ar.rule_type IN ('cidr', 'ip')
		AND ar.is_active = true
		AND (
			-- User has direct access to the rule
			EXISTS (SELECT 1 FROM user_access_rules uar WHERE uar.access_rule_id = ar.id AND uar.user_id = $2)
			-- Or user's group has access to the rule
			OR EXISTS (SELECT 1 FROM group_access_rules gar WHERE gar.access_rule_id = ar.id AND gar.group_name = ANY($3))
		)
		ORDER BY route
	`, hubID, userID, groups)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []string
	for rows.Next() {
		var route string
		if err := rows.Scan(&route); err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	return routes, rows.Err()
}

// GetUserMeshAccessRulesByEmail returns access rules for a user by email through hub networks
// Used by mesh hub firewall enforcement to look up rules for connected clients
func (s *MeshStore) GetUserMeshAccessRulesByEmail(ctx context.Context, hubID, email string) ([]string, error) {
	// First, get the user's ID and groups from the email
	var userID string
	var groups []string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id::text, COALESCE(groups, '{}'::text[])
		FROM users
		WHERE email = $1
	`, email).Scan(&userID, &groups)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return s.GetUserMeshAccessRules(ctx, hubID, userID, groups)
}

// MeshAccessRule represents an access rule for firewall enforcement
type MeshAccessRule struct {
	Type     string `json:"type"`     // ip, cidr, hostname, hostname_wildcard
	Value    string `json:"value"`    // The rule value
	Port     string `json:"port"`     // Port or port range (optional, * for all)
	Protocol string `json:"protocol"` // tcp, udp, * for all
}

// GetUserMeshAccessRulesDetailed returns detailed access rules for firewall enforcement
// Includes all rule types: ip, cidr, hostname, hostname_wildcard
func (s *MeshStore) GetUserMeshAccessRulesDetailed(ctx context.Context, hubID, userID string, groups []string) ([]MeshAccessRule, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT DISTINCT ar.rule_type, ar.value, COALESCE(ar.port, '*'), COALESCE(ar.protocol, '*')
		FROM access_rules ar
		JOIN mesh_hub_networks hn ON ar.network_id = hn.network_id
		WHERE hn.hub_id = $1
		AND ar.is_active = true
		AND (
			-- User has direct access to the rule
			EXISTS (SELECT 1 FROM user_access_rules uar WHERE uar.access_rule_id = ar.id AND uar.user_id = $2)
			-- Or user's group has access to the rule
			OR EXISTS (SELECT 1 FROM group_access_rules gar WHERE gar.access_rule_id = ar.id AND gar.group_name = ANY($3))
		)
		ORDER BY ar.rule_type, ar.value
	`, hubID, userID, groups)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []MeshAccessRule
	for rows.Next() {
		var rule MeshAccessRule
		if err := rows.Scan(&rule.Type, &rule.Value, &rule.Port, &rule.Protocol); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

// GetUserMeshAccessRulesDetailedByEmail returns detailed access rules for a user by email
func (s *MeshStore) GetUserMeshAccessRulesDetailedByEmail(ctx context.Context, hubID, email string) ([]MeshAccessRule, error) {
	var userID string
	var groups []string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id::text, COALESCE(groups, '{}'::text[])
		FROM users
		WHERE email = $1
	`, email).Scan(&userID, &groups)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return s.GetUserMeshAccessRulesDetailed(ctx, hubID, userID, groups)
}

// ==================== WireGuard Mesh Support Methods ====================

// SetHubWireGuardKeys sets the WireGuard keys for a hub (alias for UpdateHubWireGuardKeys)
func (s *MeshStore) SetHubWireGuardKeys(ctx context.Context, hubID, privateKey, publicKey string) error {
	return s.UpdateHubWireGuardKeys(ctx, hubID, privateKey, publicKey)
}

// UpdateHubHeartbeat updates the hub's last heartbeat timestamp and public IP
func (s *MeshStore) UpdateHubHeartbeat(ctx context.Context, hubID, publicIP string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_hubs SET
			last_heartbeat = NOW(),
			status = 'online'
		WHERE id = $1
	`, hubID)
	return err
}

// UpdateHubConnectedCounts updates the connected spoke and client counts for a hub
func (s *MeshStore) UpdateHubConnectedCounts(ctx context.Context, hubID string, spokeCount, clientCount int) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_hubs SET
			connected_gateways = $2,
			connected_clients = $3
		WHERE id = $1
	`, hubID, spokeCount, clientCount)
	return err
}

// GetMeshSpokeByWGPublicKey finds a spoke by its WireGuard public key within a hub
func (s *MeshStore) GetMeshSpokeByWGPublicKey(ctx context.Context, hubID, publicKey string) (*MeshSpoke, error) {
	var gw MeshSpoke
	var remoteIP *string
	var tunnelIP *string
	var lastSeen *time.Time
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, hub_id, name, COALESCE(description, ''),
			gateway_type,
			api_token, COALESCE(local_networks, '{}'),
			COALESCE(wg_private_key, ''), COALESCE(wg_public_key, ''), COALESCE(wg_preshared_key, ''),
			status, COALESCE(status_message, ''), last_seen, remote_ip::text, tunnel_ip,
			bytes_sent, bytes_received,
			COALESCE(full_tunnel_mode, false), COALESCE(push_dns, false), COALESCE(dns_servers, '{}'),
			created_at, updated_at
		FROM mesh_gateways
		WHERE hub_id = $1 AND wg_public_key = $2
	`, hubID, publicKey).Scan(
		&gw.ID, &gw.HubID, &gw.Name, &gw.Description,
		&gw.GatewayType,
		&gw.Token, &gw.LocalNetworks,
		&gw.WGPrivateKey, &gw.WGPublicKey, &gw.WGPresharedKey,
		&gw.Status, &gw.StatusMessage, &lastSeen, &remoteIP, &tunnelIP,
		&gw.BytesSent, &gw.BytesReceived,
		&gw.FullTunnelMode, &gw.PushDNS, &gw.DNSServers,
		&gw.CreatedAt, &gw.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, ErrMeshSpokeNotFound
		}
		return nil, err
	}
	gw.LastSeen = lastSeen
	if remoteIP != nil {
		gw.RemoteIP = *remoteIP
	}
	if tunnelIP != nil {
		gw.TunnelIP = *tunnelIP
	}
	return &gw, nil
}

// UpdateMeshSpokeStats updates the statistics for a mesh spoke (WireGuard-specific)
func (s *MeshStore) UpdateMeshSpokeStats(ctx context.Context, spokeID, status, endpoint string, bytesSent, bytesReceived int64, lastHandshake *time.Time) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_gateways SET
			status = $2,
			remote_ip = NULLIF($3, '')::inet,
			bytes_sent = $4,
			bytes_received = $5,
			last_seen = COALESCE($6, last_seen)
		WHERE id = $1
	`, spokeID, status, endpoint, bytesSent, bytesReceived, lastHandshake)
	return err
}

// SetSpokeWireGuardKeys sets the WireGuard keys for a spoke
func (s *MeshStore) SetSpokeWireGuardKeys(ctx context.Context, spokeID, privateKey, publicKey, presharedKey string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_gateways SET
			wg_private_key = $2,
			wg_public_key = $3,
			wg_preshared_key = $4
		WHERE id = $1
	`, spokeID, privateKey, publicKey, presharedKey)
	return err
}

// UpdateMeshSpokeTunnelIP updates the tunnel IP for a spoke
func (s *MeshStore) UpdateMeshSpokeTunnelIP(ctx context.Context, spokeID, tunnelIP string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_gateways SET
			tunnel_ip = $2
		WHERE id = $1
	`, spokeID, tunnelIP)
	return err
}

// UpdateMeshSpokeHeartbeat updates the spoke's last seen timestamp and public IP
func (s *MeshStore) UpdateMeshSpokeHeartbeat(ctx context.Context, spokeID, publicIP string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_gateways SET
			last_seen = NOW(),
			status = 'online',
			remote_ip = NULLIF($2, '')::inet
		WHERE id = $1
	`, spokeID, publicIP)
	return err
}

// ==================== WireGuard Mesh Client Peers ====================

// WGMeshPeer represents a WireGuard client peer connected to a mesh hub
type WGMeshPeer struct {
	ID           string
	HubID        string
	ConfigID     string
	UserID       string
	Email        string
	PublicKey    string
	PresharedKey string
	AssignedIP   string
	AllowedIPs   []string
	ExpiresAt    time.Time
	CreatedAt    time.Time
	LastSeen     *time.Time
	IsRevoked    bool
	RevokedAt    *time.Time
}

// AddWGMeshPeer adds a WireGuard mesh peer (mobile/desktop client)
func (s *MeshStore) AddWGMeshPeer(ctx context.Context, peer *WGMeshPeer) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO wg_mesh_peers (id, hub_id, config_id, user_id, email, public_key, preshared_key, assigned_ip, allowed_ips, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::inet, $9, $10)
		ON CONFLICT (hub_id, public_key) DO UPDATE SET
			config_id = EXCLUDED.config_id,
			user_id = EXCLUDED.user_id,
			email = EXCLUDED.email,
			preshared_key = EXCLUDED.preshared_key,
			assigned_ip = EXCLUDED.assigned_ip,
			allowed_ips = EXCLUDED.allowed_ips,
			expires_at = EXCLUDED.expires_at,
			is_revoked = FALSE,
			revoked_at = NULL
	`, peer.ID, peer.HubID, peer.ConfigID, peer.UserID, peer.Email, peer.PublicKey, peer.PresharedKey, peer.AssignedIP, peer.AllowedIPs, peer.ExpiresAt)
	return err
}

// GetWGMeshPeer retrieves a WireGuard mesh peer by ID
func (s *MeshStore) GetWGMeshPeer(ctx context.Context, id string) (*WGMeshPeer, error) {
	var peer WGMeshPeer
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, hub_id, COALESCE(config_id::text, ''), user_id, COALESCE(email, ''),
		       public_key, COALESCE(preshared_key, ''), host(assigned_ip), allowed_ips,
		       expires_at, created_at, last_seen, is_revoked, revoked_at
		FROM wg_mesh_peers
		WHERE id = $1
	`, id).Scan(&peer.ID, &peer.HubID, &peer.ConfigID, &peer.UserID, &peer.Email,
		&peer.PublicKey, &peer.PresharedKey, &peer.AssignedIP, &peer.AllowedIPs,
		&peer.ExpiresAt, &peer.CreatedAt, &peer.LastSeen, &peer.IsRevoked, &peer.RevokedAt)
	if err == pgx.ErrNoRows {
		return nil, errors.New("wg mesh peer not found")
	}
	return &peer, err
}

// GetWGMeshPeerByPublicKey retrieves a WireGuard mesh peer by public key
func (s *MeshStore) GetWGMeshPeerByPublicKey(ctx context.Context, hubID, publicKey string) (*WGMeshPeer, error) {
	var peer WGMeshPeer
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, hub_id, COALESCE(config_id::text, ''), user_id, COALESCE(email, ''),
		       public_key, COALESCE(preshared_key, ''), host(assigned_ip), allowed_ips,
		       expires_at, created_at, last_seen, is_revoked, revoked_at
		FROM wg_mesh_peers
		WHERE hub_id = $1 AND public_key = $2
	`, hubID, publicKey).Scan(&peer.ID, &peer.HubID, &peer.ConfigID, &peer.UserID, &peer.Email,
		&peer.PublicKey, &peer.PresharedKey, &peer.AssignedIP, &peer.AllowedIPs,
		&peer.ExpiresAt, &peer.CreatedAt, &peer.LastSeen, &peer.IsRevoked, &peer.RevokedAt)
	if err == pgx.ErrNoRows {
		return nil, errors.New("wg mesh peer not found")
	}
	return &peer, err
}

// ListActiveWGMeshPeers returns all active (non-expired, non-revoked) WireGuard mesh peers for a hub
func (s *MeshStore) ListActiveWGMeshPeers(ctx context.Context, hubID string) ([]*WGMeshPeer, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, hub_id, COALESCE(config_id::text, ''), user_id, COALESCE(email, ''),
		       public_key, COALESCE(preshared_key, ''), host(assigned_ip), allowed_ips,
		       expires_at, created_at, last_seen, is_revoked, revoked_at
		FROM wg_mesh_peers
		WHERE hub_id = $1 AND is_revoked = FALSE AND expires_at > NOW()
		ORDER BY created_at DESC
	`, hubID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []*WGMeshPeer
	for rows.Next() {
		var peer WGMeshPeer
		if err := rows.Scan(&peer.ID, &peer.HubID, &peer.ConfigID, &peer.UserID, &peer.Email,
			&peer.PublicKey, &peer.PresharedKey, &peer.AssignedIP, &peer.AllowedIPs,
			&peer.ExpiresAt, &peer.CreatedAt, &peer.LastSeen, &peer.IsRevoked, &peer.RevokedAt); err != nil {
			return nil, err
		}
		peers = append(peers, &peer)
	}
	return peers, rows.Err()
}

// GetHubAllocatedIPs returns all IPs currently allocated to peers (spokes + mobile clients) for a hub
func (s *MeshStore) GetHubAllocatedIPs(ctx context.Context, hubID string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT ip FROM (
			-- IPs allocated to spokes
			SELECT host(tunnel_ip) as ip FROM mesh_gateways
			WHERE hub_id = $1 AND tunnel_ip IS NOT NULL
			UNION
			-- IPs allocated to WireGuard mobile clients
			SELECT host(assigned_ip) as ip FROM wg_mesh_peers
			WHERE hub_id = $1 AND assigned_ip IS NOT NULL AND is_revoked = FALSE AND expires_at > NOW()
		) combined
		WHERE ip IS NOT NULL
	`, hubID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, err
		}
		ips = append(ips, ip)
	}
	return ips, rows.Err()
}

// UpdateWGMeshPeerLastSeen updates the last seen timestamp for a peer
func (s *MeshStore) UpdateWGMeshPeerLastSeen(ctx context.Context, hubID, publicKey string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE wg_mesh_peers SET last_seen = NOW()
		WHERE hub_id = $1 AND public_key = $2
	`, hubID, publicKey)
	return err
}

// RevokeWGMeshPeer revokes a WireGuard mesh peer
func (s *MeshStore) RevokeWGMeshPeer(ctx context.Context, id string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE wg_mesh_peers SET is_revoked = TRUE, revoked_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

// CleanupExpiredWGMeshPeers removes expired WireGuard mesh peers
func (s *MeshStore) CleanupExpiredWGMeshPeers(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.db.Pool.Exec(ctx, `
		DELETE FROM wg_mesh_peers WHERE expires_at < $1
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
