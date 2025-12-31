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
	ErrGatewayNotFound = errors.New("gateway not found")
	ErrGatewayExists   = errors.New("gateway already exists")
)

// Gateway represents a registered VPN gateway
type Gateway struct {
	ID             string
	Name           string
	Hostname       string
	PublicIP       string
	VPNPort        int
	VPNProtocol    string
	CryptoProfile  string // "modern", "fips", or "compatible"
	VPNSubnet      string // VPN client subnet (e.g., "10.8.0.0/24")
	TLSAuthEnabled bool     // Enable TLS-Auth for additional security
	TLSAuthKey     string   // TLS-Auth static key (generated during provisioning)
	FullTunnelMode bool     // When true, route all traffic through VPN (push 0.0.0.0/0)
	PushDNS        bool     // When true, push DNS servers to VPN clients
	DNSServers     []string // DNS server IPs to push to clients
	ConfigVersion  string   // Hash of config settings - changes trigger gateway reprovision
	Token          string
	PublicKey      string
	IsActive       bool
	LastHeartbeat  *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Default VPN subnet if not specified
const DefaultVPNSubnet = "172.31.255.0/24"

// CryptoProfile constants
const (
	CryptoProfileModern     = "modern"     // Modern secure defaults (AES-256-GCM, CHACHA20-POLY1305)
	CryptoProfileFIPS       = "fips"       // FIPS 140-3 compliant (AES-256-GCM, AES-128-GCM)
	CryptoProfileCompatible = "compatible" // Maximum compatibility (AES-256-GCM, AES-128-GCM, AES-256-CBC, AES-128-CBC)
)

// GatewayStore handles gateway persistence
type GatewayStore struct {
	db *DB
}

// NewGatewayStore creates a new gateway store
func NewGatewayStore(db *DB) *GatewayStore {
	return &GatewayStore{db: db}
}

// GenerateToken generates a secure random token for gateway authentication
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CreateGateway registers a new gateway
func (s *GatewayStore) CreateGateway(ctx context.Context, gw *Gateway) error {
	// Default to modern crypto profile if not specified
	cryptoProfile := gw.CryptoProfile
	if cryptoProfile == "" {
		cryptoProfile = CryptoProfileModern
	}
	// Default VPN subnet
	vpnSubnet := gw.VPNSubnet
	if vpnSubnet == "" {
		vpnSubnet = DefaultVPNSubnet
	}
	// Use NULLIF to convert empty string to NULL for hostname and inet type
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO gateways (name, hostname, public_ip, vpn_port, vpn_protocol, crypto_profile, vpn_subnet, tls_auth_enabled, full_tunnel_mode, push_dns, dns_servers, token, public_key)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, '')::inet, $4, $5, $6, $7::cidr, $8, $9, $10, $11, $12, $13)
	`, gw.Name, gw.Hostname, gw.PublicIP, gw.VPNPort, gw.VPNProtocol, cryptoProfile, vpnSubnet, gw.TLSAuthEnabled, gw.FullTunnelMode, gw.PushDNS, gw.DNSServers, gw.Token, gw.PublicKey)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return ErrGatewayExists
	}
	return err
}

// GetGateway retrieves a gateway by ID
func (s *GatewayStore) GetGateway(ctx context.Context, id string) (*Gateway, error) {
	var gw Gateway
	var hostname, publicIP, vpnSubnet, tlsAuthKey *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, hostname, host(public_ip), vpn_port, vpn_protocol, crypto_profile, vpn_subnet::text, tls_auth_enabled, COALESCE(tls_auth_key, ''), full_tunnel_mode, push_dns, dns_servers, COALESCE(config_version, ''), token, public_key, is_active, last_heartbeat, created_at, updated_at
		FROM gateways WHERE id = $1
	`, id).Scan(&gw.ID, &gw.Name, &hostname, &publicIP, &gw.VPNPort, &gw.VPNProtocol, &gw.CryptoProfile, &vpnSubnet, &gw.TLSAuthEnabled, &gw.TLSAuthKey, &gw.FullTunnelMode, &gw.PushDNS, &gw.DNSServers, &gw.ConfigVersion, &gw.Token, &gw.PublicKey, &gw.IsActive, &gw.LastHeartbeat, &gw.CreatedAt, &gw.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrGatewayNotFound
	}
	if err != nil {
		return nil, err
	}
	if hostname != nil {
		gw.Hostname = *hostname
	}
	if publicIP != nil {
		gw.PublicIP = *publicIP
	}
	if vpnSubnet != nil {
		gw.VPNSubnet = *vpnSubnet
	} else {
		gw.VPNSubnet = DefaultVPNSubnet
	}
	_ = tlsAuthKey // unused, using COALESCE instead
	return &gw, nil
}

// GetGatewayByName retrieves a gateway by name
func (s *GatewayStore) GetGatewayByName(ctx context.Context, name string) (*Gateway, error) {
	var gw Gateway
	var hostname, publicIP, vpnSubnet *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, hostname, host(public_ip), vpn_port, vpn_protocol, crypto_profile, vpn_subnet::text, tls_auth_enabled, COALESCE(tls_auth_key, ''), full_tunnel_mode, push_dns, dns_servers, COALESCE(config_version, ''), token, public_key, is_active, last_heartbeat, created_at, updated_at
		FROM gateways WHERE name = $1
	`, name).Scan(&gw.ID, &gw.Name, &hostname, &publicIP, &gw.VPNPort, &gw.VPNProtocol, &gw.CryptoProfile, &vpnSubnet, &gw.TLSAuthEnabled, &gw.TLSAuthKey, &gw.FullTunnelMode, &gw.PushDNS, &gw.DNSServers, &gw.ConfigVersion, &gw.Token, &gw.PublicKey, &gw.IsActive, &gw.LastHeartbeat, &gw.CreatedAt, &gw.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrGatewayNotFound
	}
	if err != nil {
		return nil, err
	}
	if hostname != nil {
		gw.Hostname = *hostname
	}
	if publicIP != nil {
		gw.PublicIP = *publicIP
	}
	if vpnSubnet != nil {
		gw.VPNSubnet = *vpnSubnet
	} else {
		gw.VPNSubnet = DefaultVPNSubnet
	}
	return &gw, nil
}

// GetGatewayByToken retrieves a gateway by its authentication token
func (s *GatewayStore) GetGatewayByToken(ctx context.Context, token string) (*Gateway, error) {
	var gw Gateway
	var hostname, publicIP, vpnSubnet *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, hostname, host(public_ip), vpn_port, vpn_protocol, crypto_profile, vpn_subnet::text, tls_auth_enabled, COALESCE(tls_auth_key, ''), full_tunnel_mode, push_dns, dns_servers, COALESCE(config_version, ''), token, public_key, is_active, last_heartbeat, created_at, updated_at
		FROM gateways WHERE token = $1
	`, token).Scan(&gw.ID, &gw.Name, &hostname, &publicIP, &gw.VPNPort, &gw.VPNProtocol, &gw.CryptoProfile, &vpnSubnet, &gw.TLSAuthEnabled, &gw.TLSAuthKey, &gw.FullTunnelMode, &gw.PushDNS, &gw.DNSServers, &gw.ConfigVersion, &gw.Token, &gw.PublicKey, &gw.IsActive, &gw.LastHeartbeat, &gw.CreatedAt, &gw.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrGatewayNotFound
	}
	if err != nil {
		return nil, err
	}
	if hostname != nil {
		gw.Hostname = *hostname
	}
	if publicIP != nil {
		gw.PublicIP = *publicIP
	}
	if vpnSubnet != nil {
		gw.VPNSubnet = *vpnSubnet
	} else {
		gw.VPNSubnet = DefaultVPNSubnet
	}
	return &gw, nil
}

// SetTLSAuthKey updates the TLS-Auth key for a gateway (used during provisioning)
func (s *GatewayStore) SetTLSAuthKey(ctx context.Context, gatewayID, tlsAuthKey string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE gateways SET tls_auth_key = $2, updated_at = NOW() WHERE id = $1
	`, gatewayID, tlsAuthKey)
	return err
}

// ListGateways retrieves all gateways
func (s *GatewayStore) ListGateways(ctx context.Context) ([]*Gateway, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, name, hostname, host(public_ip), vpn_port, vpn_protocol, crypto_profile, vpn_subnet::text, tls_auth_enabled, full_tunnel_mode, push_dns, dns_servers, is_active, last_heartbeat, created_at, updated_at
		FROM gateways
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gateways []*Gateway
	for rows.Next() {
		var gw Gateway
		var hostname, publicIP, vpnSubnet *string
		if err := rows.Scan(&gw.ID, &gw.Name, &hostname, &publicIP, &gw.VPNPort, &gw.VPNProtocol, &gw.CryptoProfile, &vpnSubnet, &gw.TLSAuthEnabled, &gw.FullTunnelMode, &gw.PushDNS, &gw.DNSServers, &gw.IsActive, &gw.LastHeartbeat, &gw.CreatedAt, &gw.UpdatedAt); err != nil {
			return nil, err
		}
		if hostname != nil {
			gw.Hostname = *hostname
		}
		if publicIP != nil {
			gw.PublicIP = *publicIP
		}
		if vpnSubnet != nil {
			gw.VPNSubnet = *vpnSubnet
		} else {
			gw.VPNSubnet = DefaultVPNSubnet
		}
		gateways = append(gateways, &gw)
	}
	return gateways, rows.Err()
}

// ListActiveGateways retrieves all active gateways
func (s *GatewayStore) ListActiveGateways(ctx context.Context) ([]*Gateway, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, name, hostname, host(public_ip), vpn_port, vpn_protocol, crypto_profile, vpn_subnet::text, tls_auth_enabled, full_tunnel_mode, push_dns, dns_servers, is_active, last_heartbeat, created_at, updated_at
		FROM gateways
		WHERE is_active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gateways []*Gateway
	for rows.Next() {
		var gw Gateway
		var hostname, publicIP, vpnSubnet *string
		if err := rows.Scan(&gw.ID, &gw.Name, &hostname, &publicIP, &gw.VPNPort, &gw.VPNProtocol, &gw.CryptoProfile, &vpnSubnet, &gw.TLSAuthEnabled, &gw.FullTunnelMode, &gw.PushDNS, &gw.DNSServers, &gw.IsActive, &gw.LastHeartbeat, &gw.CreatedAt, &gw.UpdatedAt); err != nil {
			return nil, err
		}
		if hostname != nil {
			gw.Hostname = *hostname
		}
		if publicIP != nil {
			gw.PublicIP = *publicIP
		}
		if vpnSubnet != nil {
			gw.VPNSubnet = *vpnSubnet
		} else {
			gw.VPNSubnet = DefaultVPNSubnet
		}
		gateways = append(gateways, &gw)
	}
	return gateways, rows.Err()
}

// UpdateHeartbeat updates the gateway's last heartbeat time and sets it active
func (s *GatewayStore) UpdateHeartbeat(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE gateways SET last_heartbeat = NOW(), is_active = true WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrGatewayNotFound
	}
	return nil
}

// UpdateGatewayStatus updates the gateway's public IP and active status
func (s *GatewayStore) UpdateGatewayStatus(ctx context.Context, id, publicIP string) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE gateways SET public_ip = NULLIF($2, '')::inet, last_heartbeat = NOW(), is_active = true WHERE id = $1
	`, id, publicIP)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrGatewayNotFound
	}
	return nil
}

// DeleteGateway removes a gateway
func (s *GatewayStore) DeleteGateway(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM gateways WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrGatewayNotFound
	}
	return nil
}

// MarkInactiveGateways marks gateways as inactive if they haven't sent a heartbeat recently
func (s *GatewayStore) MarkInactiveGateways(ctx context.Context, threshold time.Duration) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE gateways SET is_active = false
		WHERE is_active = true AND (last_heartbeat IS NULL OR last_heartbeat < NOW() - $1::interval)
	`, threshold.String())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// UpdateGateway updates a gateway's properties
func (s *GatewayStore) UpdateGateway(ctx context.Context, gw *Gateway) error {
	// Default to modern crypto profile if not specified
	cryptoProfile := gw.CryptoProfile
	if cryptoProfile == "" {
		cryptoProfile = CryptoProfileModern
	}
	// Default VPN subnet
	vpnSubnet := gw.VPNSubnet
	if vpnSubnet == "" {
		vpnSubnet = DefaultVPNSubnet
	}
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE gateways
		SET name = $2, hostname = NULLIF($3, ''), public_ip = NULLIF($4, '')::inet,
		    vpn_port = $5, vpn_protocol = $6, crypto_profile = $7, vpn_subnet = $8::cidr, tls_auth_enabled = $9, full_tunnel_mode = $10, push_dns = $11, dns_servers = $12, updated_at = NOW()
		WHERE id = $1
	`, gw.ID, gw.Name, gw.Hostname, gw.PublicIP, gw.VPNPort, gw.VPNProtocol, cryptoProfile, vpnSubnet, gw.TLSAuthEnabled, gw.FullTunnelMode, gw.PushDNS, gw.DNSServers)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrGatewayExists
		}
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrGatewayNotFound
	}
	return nil
}

// AssignUserToGateway assigns a user to a gateway
func (s *GatewayStore) AssignUserToGateway(ctx context.Context, userID, gatewayID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO user_gateways (user_id, gateway_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, gatewayID)
	return err
}

// RemoveUserFromGateway removes a user from a gateway
func (s *GatewayStore) RemoveUserFromGateway(ctx context.Context, userID, gatewayID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM user_gateways WHERE user_id = $1 AND gateway_id = $2
	`, userID, gatewayID)
	return err
}

// AssignGroupToGateway assigns a group to a gateway
func (s *GatewayStore) AssignGroupToGateway(ctx context.Context, groupName, gatewayID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO group_gateways (group_name, gateway_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, groupName, gatewayID)
	return err
}

// RemoveGroupFromGateway removes a group from a gateway
func (s *GatewayStore) RemoveGroupFromGateway(ctx context.Context, groupName, gatewayID string) error {
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM group_gateways WHERE group_name = $1 AND gateway_id = $2
	`, groupName, gatewayID)
	return err
}

// GatewayUser represents a user assigned to a gateway
type GatewayUser struct {
	UserID    string
	Email     string
	Name      string
	CreatedAt time.Time
}

// GetGatewayUsers returns all users assigned to a gateway
func (s *GatewayStore) GetGatewayUsers(ctx context.Context, gatewayID string) ([]GatewayUser, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT ug.user_id, COALESCE(u.email, ''), COALESCE(u.name, ''), ug.created_at
		FROM user_gateways ug
		LEFT JOIN users u ON ug.user_id = u.id::text
		WHERE ug.gateway_id = $1
		ORDER BY u.email
	`, gatewayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []GatewayUser
	for rows.Next() {
		var u GatewayUser
		if err := rows.Scan(&u.UserID, &u.Email, &u.Name, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GatewayGroup represents a group assigned to a gateway
type GatewayGroup struct {
	GroupName string
	CreatedAt time.Time
}

// GetGatewayGroups returns all groups assigned to a gateway
func (s *GatewayStore) GetGatewayGroups(ctx context.Context, gatewayID string) ([]GatewayGroup, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT group_name, created_at
		FROM group_gateways
		WHERE gateway_id = $1
		ORDER BY group_name
	`, gatewayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []GatewayGroup
	for rows.Next() {
		var g GatewayGroup
		if err := rows.Scan(&g.GroupName, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// ListUserGateways returns gateways accessible by a user (via direct assignment or group membership)
func (s *GatewayStore) ListUserGateways(ctx context.Context, userID string, groups []string) ([]*Gateway, error) {
	// Query gateways that the user can access via direct assignment or group membership
	rows, err := s.db.Pool.Query(ctx, `
		SELECT DISTINCT g.id, g.name, g.hostname, host(g.public_ip), g.vpn_port, g.vpn_protocol,
		       g.crypto_profile, g.is_active, g.last_heartbeat, g.created_at, g.updated_at
		FROM gateways g
		WHERE g.id IN (
			SELECT gateway_id FROM user_gateways WHERE user_id = $1
			UNION
			SELECT gateway_id FROM group_gateways WHERE group_name = ANY($2)
		)
		ORDER BY g.name
	`, userID, groups)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gateways []*Gateway
	for rows.Next() {
		var gw Gateway
		var hostname, publicIP *string
		if err := rows.Scan(&gw.ID, &gw.Name, &hostname, &publicIP, &gw.VPNPort, &gw.VPNProtocol, &gw.CryptoProfile, &gw.IsActive, &gw.LastHeartbeat, &gw.CreatedAt, &gw.UpdatedAt); err != nil {
			return nil, err
		}
		if hostname != nil {
			gw.Hostname = *hostname
		}
		if publicIP != nil {
			gw.PublicIP = *publicIP
		}
		gateways = append(gateways, &gw)
	}
	return gateways, rows.Err()
}

// UserHasGatewayAccess checks if a user has access to a specific gateway
func (s *GatewayStore) UserHasGatewayAccess(ctx context.Context, userID, gatewayID string, groups []string) (bool, error) {
	var hasAccess bool
	err := s.db.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_gateways WHERE user_id = $1 AND gateway_id = $2
			UNION
			SELECT 1 FROM group_gateways WHERE gateway_id = $2 AND group_name = ANY($3)
		)
	`, userID, gatewayID, groups).Scan(&hasAccess)
	return hasAccess, err
}

// GetGatewaysForUser returns all gateways directly assigned to a user (not via groups)
func (s *GatewayStore) GetGatewaysForUser(ctx context.Context, userID string) ([]*Gateway, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT g.id, g.name, g.hostname, host(g.public_ip), g.vpn_port, g.vpn_protocol,
		       g.crypto_profile, g.is_active, g.last_heartbeat, g.created_at, g.updated_at
		FROM gateways g
		INNER JOIN user_gateways ug ON g.id = ug.gateway_id
		WHERE ug.user_id = $1
		ORDER BY g.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gateways []*Gateway
	for rows.Next() {
		var gw Gateway
		var hostname, publicIP *string
		if err := rows.Scan(&gw.ID, &gw.Name, &hostname, &publicIP, &gw.VPNPort, &gw.VPNProtocol, &gw.CryptoProfile, &gw.IsActive, &gw.LastHeartbeat, &gw.CreatedAt, &gw.UpdatedAt); err != nil {
			return nil, err
		}
		if hostname != nil {
			gw.Hostname = *hostname
		}
		if publicIP != nil {
			gw.PublicIP = *publicIP
		}
		gateways = append(gateways, &gw)
	}
	return gateways, rows.Err()
}
