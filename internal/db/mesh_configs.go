package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrMeshConfigNotFound = errMeshConfigNotFound{}
	ErrMeshConfigExpired  = errMeshConfigExpired{}
	ErrMeshConfigRevoked  = errMeshConfigRevoked{}
)

type errMeshConfigNotFound struct{}

func (e errMeshConfigNotFound) Error() string { return "mesh config not found" }

type errMeshConfigExpired struct{}

func (e errMeshConfigExpired) Error() string { return "mesh config expired" }

type errMeshConfigRevoked struct{}

func (e errMeshConfigRevoked) Error() string { return "mesh config revoked" }

// MeshGeneratedConfig represents a generated Mesh Hub VPN configuration
type MeshGeneratedConfig struct {
	ID            string
	UserID        string
	HubID         string
	HubName       string
	FileName      string
	ConfigData    []byte
	SerialNumber  string
	Fingerprint   string
	IsRevoked     bool
	RevokedAt     *time.Time
	RevokedReason string
	ExpiresAt     time.Time
	CreatedAt     time.Time
	DownloadedAt  *time.Time
}

// MeshConfigStore handles mesh generated config persistence
type MeshConfigStore struct {
	db *DB
}

// NewMeshConfigStore creates a new mesh config store
func NewMeshConfigStore(db *DB) *MeshConfigStore {
	return &MeshConfigStore{db: db}
}

// SaveConfig stores a generated mesh config
func (s *MeshConfigStore) SaveConfig(ctx context.Context, config *MeshGeneratedConfig) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO mesh_generated_configs (id, user_id, hub_id, hub_name, file_name, config_data, serial_number, fingerprint, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, config.ID, config.UserID, config.HubID, config.HubName, config.FileName, config.ConfigData, config.SerialNumber, config.Fingerprint, config.ExpiresAt)
	return err
}

// GetConfig retrieves a mesh config by ID
func (s *MeshConfigStore) GetConfig(ctx context.Context, id string) (*MeshGeneratedConfig, error) {
	var config MeshGeneratedConfig
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, hub_id, hub_name, file_name, config_data, serial_number, fingerprint,
		       is_revoked, revoked_at, COALESCE(revoked_reason, ''), expires_at, created_at, downloaded_at
		FROM mesh_generated_configs
		WHERE id = $1
	`, id).Scan(&config.ID, &config.UserID, &config.HubID, &config.HubName, &config.FileName, &config.ConfigData,
		&config.SerialNumber, &config.Fingerprint, &config.IsRevoked,
		&config.RevokedAt, &config.RevokedReason, &config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrMeshConfigNotFound
	}
	if err != nil {
		return nil, err
	}

	// Check if expired
	if time.Now().After(config.ExpiresAt) {
		return nil, ErrMeshConfigExpired
	}

	return &config, nil
}

// MarkDownloaded marks a mesh config as downloaded
func (s *MeshConfigStore) MarkDownloaded(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_generated_configs SET downloaded_at = NOW() WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrMeshConfigNotFound
	}
	return nil
}

// GetUserConfigs retrieves all mesh configs for a user (including revoked ones)
func (s *MeshConfigStore) GetUserConfigs(ctx context.Context, userID string) ([]*MeshGeneratedConfig, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, hub_id, hub_name, file_name, serial_number, fingerprint,
		       is_revoked, revoked_at, COALESCE(revoked_reason, ''), expires_at, created_at, downloaded_at
		FROM mesh_generated_configs
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*MeshGeneratedConfig
	for rows.Next() {
		var config MeshGeneratedConfig
		if err := rows.Scan(&config.ID, &config.UserID, &config.HubID, &config.HubName, &config.FileName,
			&config.SerialNumber, &config.Fingerprint, &config.IsRevoked,
			&config.RevokedAt, &config.RevokedReason, &config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt); err != nil {
			return nil, err
		}
		configs = append(configs, &config)
	}
	return configs, rows.Err()
}

// GetAllUserConfigs retrieves all mesh configs for a user regardless of expiration
func (s *MeshConfigStore) GetAllUserConfigs(ctx context.Context, userID string) ([]*MeshGeneratedConfig, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, hub_id, hub_name, file_name, serial_number, fingerprint,
		       is_revoked, revoked_at, COALESCE(revoked_reason, ''), expires_at, created_at, downloaded_at
		FROM mesh_generated_configs
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*MeshGeneratedConfig
	for rows.Next() {
		var config MeshGeneratedConfig
		if err := rows.Scan(&config.ID, &config.UserID, &config.HubID, &config.HubName, &config.FileName,
			&config.SerialNumber, &config.Fingerprint, &config.IsRevoked,
			&config.RevokedAt, &config.RevokedReason, &config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt); err != nil {
			return nil, err
		}
		configs = append(configs, &config)
	}
	return configs, rows.Err()
}

// RevokeConfig revokes a mesh config by ID
func (s *MeshConfigStore) RevokeConfig(ctx context.Context, id string, reason string) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_generated_configs
		SET is_revoked = TRUE, revoked_at = NOW(), revoked_reason = $2
		WHERE id = $1 AND is_revoked = FALSE
	`, id, reason)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrMeshConfigNotFound
	}
	return nil
}

// RevokeUserConfigs revokes all mesh configs for a user
func (s *MeshConfigStore) RevokeUserConfigs(ctx context.Context, userID string, reason string) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE mesh_generated_configs
		SET is_revoked = TRUE, revoked_at = NOW(), revoked_reason = $2
		WHERE user_id = $1 AND is_revoked = FALSE AND expires_at > NOW()
	`, userID, reason)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// DeleteConfig deletes a mesh config by ID
func (s *MeshConfigStore) DeleteConfig(ctx context.Context, id string) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM mesh_generated_configs WHERE id = $1`, id)
	return err
}

// DeleteExpiredConfigs deletes mesh configs that expired more than the specified duration ago.
// Returns the number of configs deleted.
func (s *MeshConfigStore) DeleteExpiredConfigs(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.db.Pool.Exec(ctx, `
		DELETE FROM mesh_generated_configs
		WHERE expires_at < $1
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// GetConfigBySerial retrieves a mesh config by certificate serial number
func (s *MeshConfigStore) GetConfigBySerial(ctx context.Context, serial string) (*MeshGeneratedConfig, error) {
	var config MeshGeneratedConfig
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, hub_id, hub_name, file_name, config_data, serial_number, fingerprint,
		       is_revoked, revoked_at, COALESCE(revoked_reason, ''), expires_at, created_at, downloaded_at
		FROM mesh_generated_configs
		WHERE serial_number = $1
	`, serial).Scan(&config.ID, &config.UserID, &config.HubID, &config.HubName, &config.FileName, &config.ConfigData,
		&config.SerialNumber, &config.Fingerprint, &config.IsRevoked,
		&config.RevokedAt, &config.RevokedReason, &config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrMeshConfigNotFound
	}
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// CleanupExpiredConfigs removes expired mesh configs
func (s *MeshConfigStore) CleanupExpiredConfigs(ctx context.Context) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM mesh_generated_configs WHERE expires_at < NOW()`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// MeshConfigWithUser extends MeshGeneratedConfig with user information
type MeshConfigWithUser struct {
	MeshGeneratedConfig
	UserEmail string
	UserName  string
}

// GetAllConfigs retrieves all mesh configs with user info (for admin listing)
func (s *MeshConfigStore) GetAllConfigs(ctx context.Context, limit, offset int) ([]*MeshConfigWithUser, int64, error) {
	// Get total count
	var total int64
	err := s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM mesh_generated_configs`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Pool.Query(ctx, `
		SELECT mc.id, mc.user_id, mc.hub_id, mc.hub_name, mc.file_name, mc.serial_number, mc.fingerprint,
		       mc.is_revoked, mc.revoked_at, COALESCE(mc.revoked_reason, ''), mc.expires_at, mc.created_at, mc.downloaded_at,
		       COALESCE(u.email, lu.email, mc.user_id) as user_email,
		       COALESCE(u.name, lu.username, '') as user_name
		FROM mesh_generated_configs mc
		LEFT JOIN users u ON mc.user_id = u.id::text
		LEFT JOIN local_users lu ON mc.user_id = lu.id::text
		ORDER BY mc.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var configs []*MeshConfigWithUser
	for rows.Next() {
		var config MeshConfigWithUser
		if err := rows.Scan(&config.ID, &config.UserID, &config.HubID, &config.HubName, &config.FileName,
			&config.SerialNumber, &config.Fingerprint, &config.IsRevoked,
			&config.RevokedAt, &config.RevokedReason, &config.ExpiresAt, &config.CreatedAt, &config.DownloadedAt,
			&config.UserEmail, &config.UserName); err != nil {
			return nil, 0, err
		}
		configs = append(configs, &config)
	}
	return configs, total, rows.Err()
}
