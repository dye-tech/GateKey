package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrWGConfigNotFound = errors.New("wireguard config not found")
	ErrWGConfigExpired  = errors.New("wireguard config expired")
	ErrWGConfigRevoked  = errors.New("wireguard config revoked")
)

// WireGuardConfig represents a generated WireGuard VPN configuration
type WireGuardConfig struct {
	ID               string
	UserID           string
	GatewayID        string
	GatewayName      string
	FileName         string
	ConfigData       []byte
	ClientPrivateKey string
	ClientPublicKey  string
	AssignedIP       string
	PresharedKey     string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	DownloadedAt     *time.Time
	RevokedAt        *time.Time
	RevocationReason string
}

// WireGuardConfigStore handles WireGuard config persistence
type WireGuardConfigStore struct {
	db *DB
}

// NewWireGuardConfigStore creates a new WireGuard config store
func NewWireGuardConfigStore(db *DB) *WireGuardConfigStore {
	return &WireGuardConfigStore{db: db}
}

// Create stores a new WireGuard config
func (s *WireGuardConfigStore) Create(ctx context.Context, config *WireGuardConfig) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO wireguard_configs (id, user_id, gateway_id, gateway_name, file_name, config_data,
		                               client_private_key, client_public_key, assigned_ip, preshared_key, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::cidr, $10, $11)
	`, config.ID, config.UserID, config.GatewayID, config.GatewayName, config.FileName, config.ConfigData,
		config.ClientPrivateKey, config.ClientPublicKey, config.AssignedIP, config.PresharedKey, config.ExpiresAt)
	return err
}

// GetByID retrieves a WireGuard config by ID
func (s *WireGuardConfigStore) GetByID(ctx context.Context, id string) (*WireGuardConfig, error) {
	var config WireGuardConfig
	var presharedKey *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, gateway_id, gateway_name, file_name, config_data,
		       client_private_key, client_public_key, assigned_ip::text, preshared_key,
		       expires_at, created_at, downloaded_at, revoked_at, COALESCE(revocation_reason, '')
		FROM wireguard_configs
		WHERE id = $1
	`, id).Scan(&config.ID, &config.UserID, &config.GatewayID, &config.GatewayName, &config.FileName, &config.ConfigData,
		&config.ClientPrivateKey, &config.ClientPublicKey, &config.AssignedIP, &presharedKey,
		&config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt, &config.RevokedAt, &config.RevocationReason)
	if err == pgx.ErrNoRows {
		return nil, ErrWGConfigNotFound
	}
	if err != nil {
		return nil, err
	}
	if presharedKey != nil {
		config.PresharedKey = *presharedKey
	}

	// Check if revoked
	if config.RevokedAt != nil {
		return nil, ErrWGConfigRevoked
	}

	// Check if expired
	if time.Now().After(config.ExpiresAt) {
		return nil, ErrWGConfigExpired
	}

	return &config, nil
}

// GetByPublicKey retrieves a config by client public key for a specific gateway
func (s *WireGuardConfigStore) GetByPublicKey(ctx context.Context, gatewayID, publicKey string) (*WireGuardConfig, error) {
	var config WireGuardConfig
	var presharedKey *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, gateway_id, gateway_name, file_name,
		       client_private_key, client_public_key, assigned_ip::text, preshared_key,
		       expires_at, created_at, downloaded_at, revoked_at, COALESCE(revocation_reason, '')
		FROM wireguard_configs
		WHERE gateway_id = $1 AND client_public_key = $2
	`, gatewayID, publicKey).Scan(&config.ID, &config.UserID, &config.GatewayID, &config.GatewayName, &config.FileName,
		&config.ClientPrivateKey, &config.ClientPublicKey, &config.AssignedIP, &presharedKey,
		&config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt, &config.RevokedAt, &config.RevocationReason)
	if err == pgx.ErrNoRows {
		return nil, ErrWGConfigNotFound
	}
	if err != nil {
		return nil, err
	}
	if presharedKey != nil {
		config.PresharedKey = *presharedKey
	}
	return &config, nil
}

// GetActiveByGateway returns all active (non-revoked, non-expired) configs for a gateway
func (s *WireGuardConfigStore) GetActiveByGateway(ctx context.Context, gatewayID string) ([]*WireGuardConfig, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, gateway_id, gateway_name, file_name,
		       client_private_key, client_public_key, assigned_ip::text, preshared_key,
		       expires_at, created_at, downloaded_at
		FROM wireguard_configs
		WHERE gateway_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
	`, gatewayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*WireGuardConfig
	for rows.Next() {
		var config WireGuardConfig
		var presharedKey *string
		if err := rows.Scan(&config.ID, &config.UserID, &config.GatewayID, &config.GatewayName, &config.FileName,
			&config.ClientPrivateKey, &config.ClientPublicKey, &config.AssignedIP, &presharedKey,
			&config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt); err != nil {
			return nil, err
		}
		if presharedKey != nil {
			config.PresharedKey = *presharedKey
		}
		configs = append(configs, &config)
	}
	return configs, rows.Err()
}

// GetUserConfigs retrieves all configs for a user
func (s *WireGuardConfigStore) GetUserConfigs(ctx context.Context, userID string) ([]*WireGuardConfig, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, gateway_id, gateway_name, file_name,
		       client_public_key, assigned_ip::text,
		       expires_at, created_at, downloaded_at, revoked_at, COALESCE(revocation_reason, '')
		FROM wireguard_configs
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*WireGuardConfig
	for rows.Next() {
		var config WireGuardConfig
		if err := rows.Scan(&config.ID, &config.UserID, &config.GatewayID, &config.GatewayName, &config.FileName,
			&config.ClientPublicKey, &config.AssignedIP,
			&config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt, &config.RevokedAt, &config.RevocationReason); err != nil {
			return nil, err
		}
		configs = append(configs, &config)
	}
	return configs, rows.Err()
}

// MarkDownloaded marks a config as downloaded
func (s *WireGuardConfigStore) MarkDownloaded(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE wireguard_configs SET downloaded_at = NOW() WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrWGConfigNotFound
	}
	return nil
}

// Revoke revokes a WireGuard config
func (s *WireGuardConfigStore) Revoke(ctx context.Context, id, reason string) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE wireguard_configs
		SET revoked_at = NOW(), revocation_reason = $2
		WHERE id = $1 AND revoked_at IS NULL
	`, id, reason)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrWGConfigNotFound
	}
	return nil
}

// RevokeByGateway revokes all configs for a gateway
func (s *WireGuardConfigStore) RevokeByGateway(ctx context.Context, gatewayID, reason string) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE wireguard_configs
		SET revoked_at = NOW(), revocation_reason = $2
		WHERE gateway_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, gatewayID, reason)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// RevokeByUser revokes all configs for a user
func (s *WireGuardConfigStore) RevokeByUser(ctx context.Context, userID, reason string) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE wireguard_configs
		SET revoked_at = NOW(), revocation_reason = $2
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, userID, reason)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// DeleteExpired deletes configs that expired more than the specified duration ago
func (s *WireGuardConfigStore) DeleteExpired(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.db.Pool.Exec(ctx, `
		DELETE FROM wireguard_configs WHERE expires_at < $1
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// WireGuardConfigWithUser extends WireGuardConfig with user information
type WireGuardConfigWithUser struct {
	WireGuardConfig
	UserEmail string
	UserName  string
}

// ListAll retrieves all configs with user info (for admin)
func (s *WireGuardConfigStore) ListAll(ctx context.Context, limit, offset int) ([]*WireGuardConfigWithUser, int, error) {
	// Get total count
	var total int
	err := s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM wireguard_configs`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Pool.Query(ctx, `
		SELECT wc.id, wc.user_id, wc.gateway_id, wc.gateway_name, wc.file_name,
		       wc.client_public_key, wc.assigned_ip::text,
		       wc.expires_at, wc.created_at, wc.downloaded_at, wc.revoked_at, COALESCE(wc.revocation_reason, ''),
		       COALESCE(u.email, lu.email, wc.user_id) as user_email,
		       COALESCE(u.name, lu.username, '') as user_name
		FROM wireguard_configs wc
		LEFT JOIN users u ON wc.user_id = u.id::text
		LEFT JOIN local_users lu ON wc.user_id = lu.id::text
		ORDER BY wc.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var configs []*WireGuardConfigWithUser
	for rows.Next() {
		var config WireGuardConfigWithUser
		if err := rows.Scan(&config.ID, &config.UserID, &config.GatewayID, &config.GatewayName, &config.FileName,
			&config.ClientPublicKey, &config.AssignedIP,
			&config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt, &config.RevokedAt, &config.RevocationReason,
			&config.UserEmail, &config.UserName); err != nil {
			return nil, 0, err
		}
		configs = append(configs, &config)
	}
	return configs, total, rows.Err()
}

// GetAllocatedIPs returns all allocated IPs for a gateway (for IP allocation)
func (s *WireGuardConfigStore) GetAllocatedIPs(ctx context.Context, gatewayID string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT assigned_ip::text
		FROM wireguard_configs
		WHERE gateway_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, gatewayID)
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
