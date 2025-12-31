package db

import (
	"context"
	"strconv"
	"time"
)

// Setting represents a system setting
type Setting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SettingsStore handles system settings persistence
type SettingsStore struct {
	db *DB
}

// NewSettingsStore creates a new settings store
func NewSettingsStore(db *DB) *SettingsStore {
	return &SettingsStore{db: db}
}

// GetAll returns all system settings
func (s *SettingsStore) GetAll(ctx context.Context) ([]*Setting, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT key, value, COALESCE(description, ''), updated_at
		FROM system_settings
		ORDER BY key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []*Setting
	for rows.Next() {
		var setting Setting
		if err := rows.Scan(&setting.Key, &setting.Value, &setting.Description, &setting.UpdatedAt); err != nil {
			return nil, err
		}
		settings = append(settings, &setting)
	}
	return settings, rows.Err()
}

// Get returns a single setting by key
func (s *SettingsStore) Get(ctx context.Context, key string) (*Setting, error) {
	var setting Setting
	err := s.db.Pool.QueryRow(ctx, `
		SELECT key, value, COALESCE(description, ''), updated_at
		FROM system_settings WHERE key = $1
	`, key).Scan(&setting.Key, &setting.Value, &setting.Description, &setting.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

// Set updates a setting value
func (s *SettingsStore) Set(ctx context.Context, key, value string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO system_settings (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()
	`, key, value)
	return err
}

// GetInt returns a setting as an integer
func (s *SettingsStore) GetInt(ctx context.Context, key string, defaultVal int) int {
	setting, err := s.Get(ctx, key)
	if err != nil {
		return defaultVal
	}
	val, err := strconv.Atoi(setting.Value)
	if err != nil {
		return defaultVal
	}
	return val
}

// GetBool returns a setting as a boolean
func (s *SettingsStore) GetBool(ctx context.Context, key string, defaultVal bool) bool {
	setting, err := s.Get(ctx, key)
	if err != nil {
		return defaultVal
	}
	val, err := strconv.ParseBool(setting.Value)
	if err != nil {
		return defaultVal
	}
	return val
}

// Common setting keys
const (
	SettingSessionDurationHours  = "session_duration_hours"
	SettingSecureCookies         = "secure_cookies"
	SettingVPNCertValidityHours  = "vpn_cert_validity_hours"
	SettingRequireFIPS           = "require_fips"
	SettingAllowedCryptoProfiles = "allowed_crypto_profiles" // Comma-separated: modern,fips,compatible
	SettingMinTLSVersion         = "min_tls_version"         // 1.0, 1.1, 1.2, 1.3
	SettingAllowedCiphers        = "allowed_ciphers"         // Comma-separated cipher list
)

// Default crypto profiles (all enabled by default)
const DefaultAllowedCryptoProfiles = "modern,fips,compatible"

// ValidCryptoProfiles contains all valid profile names
var ValidCryptoProfiles = []string{"modern", "fips", "compatible"}
