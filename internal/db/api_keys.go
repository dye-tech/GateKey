package db

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrAPIKeyNotFound = errors.New("api key not found")
	ErrAPIKeyRevoked  = errors.New("api key has been revoked")
	ErrAPIKeyExpired  = errors.New("api key has expired")
)

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID                 string     `json:"id"`
	UserID             string     `json:"user_id"`
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	KeyHash            string     `json:"-"` // Never expose the hash
	KeyPrefix          string     `json:"key_prefix"`
	Scopes             []string   `json:"scopes"`
	IsAdminProvisioned bool       `json:"is_admin_provisioned"`
	ProvisionedBy      *string    `json:"provisioned_by,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	LastUsedIP         *string    `json:"last_used_ip,omitempty"`
	IsRevoked          bool       `json:"is_revoked"`
	RevokedAt          *time.Time `json:"revoked_at,omitempty"`
	RevokedBy          *string    `json:"revoked_by,omitempty"`
	RevocationReason   *string    `json:"revocation_reason,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// APIKeyStore handles API key persistence
type APIKeyStore struct {
	db *DB
}

// NewAPIKeyStore creates a new API key store
func NewAPIKeyStore(db *DB) *APIKeyStore {
	return &APIKeyStore{db: db}
}

// GenerateAPIKey generates a new API key with hash and prefix
// Returns: rawKey (to give to user once), keyHash (for storage), keyPrefix (for display)
func GenerateAPIKey() (rawKey, keyHash, keyPrefix string, err error) {
	// Generate 32 bytes of random data
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", err
	}

	// Create raw key with prefix for easy identification
	rawKey = "gk_" + base64.URLEncoding.EncodeToString(bytes)

	// Hash for storage (SHA-256)
	hash := sha256.Sum256([]byte(rawKey))
	keyHash = hex.EncodeToString(hash[:])

	// Prefix for display/identification (first 12 chars)
	keyPrefix = rawKey[:12]

	return rawKey, keyHash, keyPrefix, nil
}

// HashAPIKey hashes an API key for lookup
func HashAPIKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}

// Create creates a new API key
func (s *APIKeyStore) Create(ctx context.Context, key *APIKey) error {
	scopesJSON, err := json.Marshal(key.Scopes)
	if err != nil {
		return err
	}

	_, err = s.db.Pool.Exec(ctx, `
		INSERT INTO api_keys (
			user_id, name, description, key_hash, key_prefix, scopes,
			is_admin_provisioned, provisioned_by, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, key.UserID, key.Name, key.Description, key.KeyHash, key.KeyPrefix,
		scopesJSON, key.IsAdminProvisioned, key.ProvisionedBy, key.ExpiresAt)

	return err
}

// GetByID retrieves an API key by ID
func (s *APIKeyStore) GetByID(ctx context.Context, id string) (*APIKey, error) {
	return s.scanAPIKey(s.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, name, description, key_hash, key_prefix, scopes,
			is_admin_provisioned, provisioned_by, expires_at, last_used_at, last_used_ip::text,
			is_revoked, revoked_at, revoked_by, revocation_reason, created_at, updated_at
		FROM api_keys WHERE id = $1
	`, id))
}

// GetByKeyHash retrieves an API key by its hash
func (s *APIKeyStore) GetByKeyHash(ctx context.Context, keyHash string) (*APIKey, error) {
	return s.scanAPIKey(s.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, name, description, key_hash, key_prefix, scopes,
			is_admin_provisioned, provisioned_by, expires_at, last_used_at, last_used_ip::text,
			is_revoked, revoked_at, revoked_by, revocation_reason, created_at, updated_at
		FROM api_keys WHERE key_hash = $1
	`, keyHash))
}

// ValidateKey validates an API key and returns the key and associated user if valid
func (s *APIKeyStore) ValidateKey(ctx context.Context, keyHash string) (*APIKey, *SSOUser, error) {
	key, err := s.GetByKeyHash(ctx, keyHash)
	if err != nil {
		if err == ErrAPIKeyNotFound {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	// Check if revoked
	if key.IsRevoked {
		return nil, nil, ErrAPIKeyRevoked
	}

	// Check if expired
	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, nil, ErrAPIKeyExpired
	}

	// Get the user
	var user SSOUser
	var groupsJSON []byte
	err = s.db.Pool.QueryRow(ctx, `
		SELECT id, external_id, provider, email, name, groups, is_admin, is_active, last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`, key.UserID).Scan(
		&user.ID, &user.ExternalID, &user.Provider, &user.Email, &user.Name,
		&groupsJSON, &user.IsAdmin, &user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil, ErrUserNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	if err := json.Unmarshal(groupsJSON, &user.Groups); err != nil {
		user.Groups = []string{}
	}

	if !user.IsActive {
		return nil, nil, ErrUserNotFound
	}

	return key, &user, nil
}

// ListByUser lists all API keys for a user
func (s *APIKeyStore) ListByUser(ctx context.Context, userID string) ([]*APIKey, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, name, description, key_hash, key_prefix, scopes,
			is_admin_provisioned, provisioned_by, expires_at, last_used_at, last_used_ip::text,
			is_revoked, revoked_at, revoked_by, revocation_reason, created_at, updated_at
		FROM api_keys WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanAPIKeys(rows)
}

// ListAll lists all API keys (admin function)
func (s *APIKeyStore) ListAll(ctx context.Context) ([]*APIKey, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, name, description, key_hash, key_prefix, scopes,
			is_admin_provisioned, provisioned_by, expires_at, last_used_at, last_used_ip::text,
			is_revoked, revoked_at, revoked_by, revocation_reason, created_at, updated_at
		FROM api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanAPIKeys(rows)
}

// UpdateLastUsed updates the last used timestamp and IP
func (s *APIKeyStore) UpdateLastUsed(ctx context.Context, id, ip string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE api_keys SET last_used_at = NOW(), last_used_ip = $2
		WHERE id = $1
	`, id, ip)
	return err
}

// Revoke revokes an API key
func (s *APIKeyStore) Revoke(ctx context.Context, id, revokedBy, reason string) error {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE api_keys
		SET is_revoked = TRUE, revoked_at = NOW(), revoked_by = $2, revocation_reason = $3
		WHERE id = $1 AND is_revoked = FALSE
	`, id, revokedBy, reason)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

// RevokeAllForUser revokes all API keys for a user
func (s *APIKeyStore) RevokeAllForUser(ctx context.Context, userID, revokedBy, reason string) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE api_keys
		SET is_revoked = TRUE, revoked_at = NOW(), revoked_by = $2, revocation_reason = $3
		WHERE user_id = $1 AND is_revoked = FALSE
	`, userID, revokedBy, reason)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// Delete permanently deletes an API key
func (s *APIKeyStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM api_keys WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

// DeleteExpiredKeys deletes all expired keys older than 30 days
func (s *APIKeyStore) DeleteExpiredKeys(ctx context.Context) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `
		DELETE FROM api_keys
		WHERE expires_at < NOW() - INTERVAL '30 days'
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// DeleteRevokedKeys deletes revoked keys older than 24 hours
func (s *APIKeyStore) DeleteRevokedKeys(ctx context.Context) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `
		DELETE FROM api_keys
		WHERE is_revoked = TRUE AND revoked_at < NOW() - INTERVAL '24 hours'
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// DeleteAllForUser permanently deletes all API keys for a user
func (s *APIKeyStore) DeleteAllForUser(ctx context.Context, userID string) (int64, error) {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM api_keys WHERE user_id = $1`, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// scanAPIKey scans a single API key row
func (s *APIKeyStore) scanAPIKey(row pgx.Row) (*APIKey, error) {
	var key APIKey
	var scopesJSON []byte
	var lastUsedIP *string

	err := row.Scan(
		&key.ID, &key.UserID, &key.Name, &key.Description, &key.KeyHash, &key.KeyPrefix,
		&scopesJSON, &key.IsAdminProvisioned, &key.ProvisionedBy, &key.ExpiresAt,
		&key.LastUsedAt, &lastUsedIP, &key.IsRevoked, &key.RevokedAt, &key.RevokedBy,
		&key.RevocationReason, &key.CreatedAt, &key.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(scopesJSON, &key.Scopes); err != nil {
		key.Scopes = []string{}
	}

	key.LastUsedIP = lastUsedIP

	return &key, nil
}

// scanAPIKeys scans multiple API key rows
func (s *APIKeyStore) scanAPIKeys(rows pgx.Rows) ([]*APIKey, error) {
	var keys []*APIKey
	for rows.Next() {
		var key APIKey
		var scopesJSON []byte
		var lastUsedIP *string

		err := rows.Scan(
			&key.ID, &key.UserID, &key.Name, &key.Description, &key.KeyHash, &key.KeyPrefix,
			&scopesJSON, &key.IsAdminProvisioned, &key.ProvisionedBy, &key.ExpiresAt,
			&key.LastUsedAt, &lastUsedIP, &key.IsRevoked, &key.RevokedAt, &key.RevokedBy,
			&key.RevocationReason, &key.CreatedAt, &key.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(scopesJSON, &key.Scopes); err != nil {
			key.Scopes = []string{}
		}

		key.LastUsedIP = lastUsedIP
		keys = append(keys, &key)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}
