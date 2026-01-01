package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrConfigNotFound = errors.New("config not found")
	ErrConfigExpired  = errors.New("config expired")
	ErrConfigRevoked  = errors.New("config revoked")
)

// GeneratedConfig represents a generated VPN configuration
type GeneratedConfig struct {
	ID             string
	UserID         string
	GatewayID      string
	GatewayName    string
	FileName       string
	ConfigData     []byte
	SerialNumber   string
	Fingerprint    string
	CLICallbackURL string
	AuthToken      string     // Unique token for password authentication
	IsRevoked      bool       // Whether this config has been revoked
	RevokedAt      *time.Time // When the config was revoked
	RevokedReason  string     // Reason for revocation
	ExpiresAt      time.Time
	CreatedAt      time.Time
	DownloadedAt   *time.Time
}

// ConfigStore handles generated config persistence
type ConfigStore struct {
	db *DB
}

// NewConfigStore creates a new config store
func NewConfigStore(db *DB) *ConfigStore {
	return &ConfigStore{db: db}
}

// SaveConfig stores a generated config
func (s *ConfigStore) SaveConfig(ctx context.Context, config *GeneratedConfig) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO generated_configs (id, user_id, gateway_id, gateway_name, file_name, config_data, serial_number, fingerprint, cli_callback_url, auth_token, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, config.ID, config.UserID, config.GatewayID, config.GatewayName, config.FileName, config.ConfigData, config.SerialNumber, config.Fingerprint, config.CLICallbackURL, config.AuthToken, config.ExpiresAt)
	return err
}

// GetConfig retrieves a config by ID
func (s *ConfigStore) GetConfig(ctx context.Context, id string) (*GeneratedConfig, error) {
	var config GeneratedConfig
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, gateway_id, gateway_name, file_name, config_data, serial_number, fingerprint, cli_callback_url,
		       COALESCE(auth_token, ''), is_revoked, revoked_at, COALESCE(revoked_reason, ''), expires_at, created_at, downloaded_at
		FROM generated_configs
		WHERE id = $1
	`, id).Scan(&config.ID, &config.UserID, &config.GatewayID, &config.GatewayName, &config.FileName, &config.ConfigData,
		&config.SerialNumber, &config.Fingerprint, &config.CLICallbackURL, &config.AuthToken, &config.IsRevoked,
		&config.RevokedAt, &config.RevokedReason, &config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrConfigNotFound
	}
	if err != nil {
		return nil, err
	}

	// Check if expired
	if time.Now().After(config.ExpiresAt) {
		return nil, ErrConfigExpired
	}

	return &config, nil
}

// MarkDownloaded marks a config as downloaded
func (s *ConfigStore) MarkDownloaded(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE generated_configs SET downloaded_at = NOW() WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrConfigNotFound
	}
	return nil
}

// GetUserConfigs retrieves all configs for a user (including revoked ones)
func (s *ConfigStore) GetUserConfigs(ctx context.Context, userID string) ([]*GeneratedConfig, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, gateway_id, gateway_name, file_name, serial_number, fingerprint, cli_callback_url,
		       is_revoked, revoked_at, COALESCE(revoked_reason, ''), expires_at, created_at, downloaded_at
		FROM generated_configs
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*GeneratedConfig
	for rows.Next() {
		var config GeneratedConfig
		if err := rows.Scan(&config.ID, &config.UserID, &config.GatewayID, &config.GatewayName, &config.FileName,
			&config.SerialNumber, &config.Fingerprint, &config.CLICallbackURL, &config.IsRevoked,
			&config.RevokedAt, &config.RevokedReason, &config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt); err != nil {
			return nil, err
		}
		configs = append(configs, &config)
	}
	return configs, rows.Err()
}

// DeleteConfig deletes a config by ID
func (s *ConfigStore) DeleteConfig(ctx context.Context, id string) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM generated_configs WHERE id = $1`, id)
	return err
}

// CleanupExpiredConfigs removes expired configs
func (s *ConfigStore) CleanupExpiredConfigs(ctx context.Context) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM generated_configs WHERE expires_at < NOW()`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// GetConfigBySerial retrieves a config by certificate serial number
func (s *ConfigStore) GetConfigBySerial(ctx context.Context, serial string) (*GeneratedConfig, error) {
	var config GeneratedConfig
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, gateway_id, gateway_name, file_name, config_data, serial_number, fingerprint, cli_callback_url,
		       COALESCE(auth_token, ''), is_revoked, revoked_at, COALESCE(revoked_reason, ''), expires_at, created_at, downloaded_at
		FROM generated_configs
		WHERE serial_number = $1
	`, serial).Scan(&config.ID, &config.UserID, &config.GatewayID, &config.GatewayName, &config.FileName, &config.ConfigData,
		&config.SerialNumber, &config.Fingerprint, &config.CLICallbackURL, &config.AuthToken, &config.IsRevoked,
		&config.RevokedAt, &config.RevokedReason, &config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrConfigNotFound
	}
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetConfigByAuthToken retrieves a config by auth token for gateway verification
// Returns nil if not found, expired, or revoked
func (s *ConfigStore) GetConfigByAuthToken(ctx context.Context, authToken string) (*GeneratedConfig, error) {
	var config GeneratedConfig
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, gateway_id, gateway_name, file_name, serial_number, fingerprint,
		       COALESCE(auth_token, ''), is_revoked, revoked_at, COALESCE(revoked_reason, ''), expires_at, created_at
		FROM generated_configs
		WHERE auth_token = $1
	`, authToken).Scan(&config.ID, &config.UserID, &config.GatewayID, &config.GatewayName, &config.FileName,
		&config.SerialNumber, &config.Fingerprint, &config.AuthToken, &config.IsRevoked,
		&config.RevokedAt, &config.RevokedReason, &config.ExpiresAt, &config.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrConfigNotFound
	}
	if err != nil {
		return nil, err
	}

	// Check if revoked
	if config.IsRevoked {
		return nil, ErrConfigRevoked
	}

	// Check if expired
	if time.Now().After(config.ExpiresAt) {
		return nil, ErrConfigExpired
	}

	return &config, nil
}

// RevokeConfig revokes a config by ID
func (s *ConfigStore) RevokeConfig(ctx context.Context, id string, reason string) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE generated_configs
		SET is_revoked = TRUE, revoked_at = NOW(), revoked_reason = $2
		WHERE id = $1 AND is_revoked = FALSE
	`, id, reason)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrConfigNotFound
	}
	return nil
}

// RevokeUserConfigs revokes all configs for a user
func (s *ConfigStore) RevokeUserConfigs(ctx context.Context, userID string, reason string) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE generated_configs
		SET is_revoked = TRUE, revoked_at = NOW(), revoked_reason = $2
		WHERE user_id = $1 AND is_revoked = FALSE AND expires_at > NOW()
	`, userID, reason)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// ValidateAuthToken checks if an auth token is valid (not revoked, not expired)
// Returns the user email and config ID if valid
func (s *ConfigStore) ValidateAuthToken(ctx context.Context, authToken string) (userID string, configID string, err error) {
	err = s.db.Pool.QueryRow(ctx, `
		SELECT user_id, id
		FROM generated_configs
		WHERE auth_token = $1 AND is_revoked = FALSE AND expires_at > NOW()
	`, authToken).Scan(&userID, &configID)
	if err == pgx.ErrNoRows {
		return "", "", ErrConfigNotFound
	}
	return userID, configID, err
}

// DeleteExpiredConfigs deletes configs that expired more than the specified duration ago.
// Returns the number of configs deleted.
func (s *ConfigStore) DeleteExpiredConfigs(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.db.Pool.Exec(ctx, `
		DELETE FROM generated_configs
		WHERE expires_at < $1
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// ConfigWithUser extends GeneratedConfig with user information
type ConfigWithUser struct {
	GeneratedConfig
	UserEmail string
	UserName  string
}

// GetAllConfigs retrieves all configs with user info (for admin)
func (s *ConfigStore) GetAllConfigs(ctx context.Context, limit, offset int) ([]*ConfigWithUser, int, error) {
	// Get total count
	var total int
	err := s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM generated_configs`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Pool.Query(ctx, `
		SELECT gc.id, gc.user_id, gc.gateway_id, gc.gateway_name, gc.file_name, gc.serial_number, gc.fingerprint,
		       gc.is_revoked, gc.revoked_at, COALESCE(gc.revoked_reason, ''), gc.expires_at, gc.created_at, gc.downloaded_at,
		       COALESCE(u.email, lu.email, gc.user_id) as user_email,
		       COALESCE(u.name, lu.username, '') as user_name
		FROM generated_configs gc
		LEFT JOIN users u ON gc.user_id = u.id::text
		LEFT JOIN local_users lu ON gc.user_id = lu.id::text
		ORDER BY gc.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var configs []*ConfigWithUser
	for rows.Next() {
		var config ConfigWithUser
		if err := rows.Scan(&config.ID, &config.UserID, &config.GatewayID, &config.GatewayName, &config.FileName,
			&config.SerialNumber, &config.Fingerprint, &config.IsRevoked,
			&config.RevokedAt, &config.RevokedReason, &config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt,
			&config.UserEmail, &config.UserName); err != nil {
			return nil, 0, err
		}
		configs = append(configs, &config)
	}
	return configs, total, rows.Err()
}
