package db

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
)

// SSOUser represents a user synced from an identity provider (OIDC/SAML)
type SSOUser struct {
	ID          string     `json:"id"`
	ExternalID  string     `json:"external_id"`
	Provider    string     `json:"provider"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	Groups      []string   `json:"groups"`
	IsAdmin     bool       `json:"is_admin"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// LocalUser represents a local admin user
type LocalUser struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	Email        string     `json:"email"`
	IsAdmin      bool       `json:"is_admin"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// AdminSession represents an admin session
type AdminSession struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// UserStore handles local user persistence
type UserStore struct {
	db *DB
	// Argon2 parameters
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

// NewUserStore creates a new user store
func NewUserStore(db *DB) *UserStore {
	return &UserStore{
		db:      db,
		time:    1,
		memory:  64 * 1024, // 64MB
		threads: 4,
		keyLen:  32,
	}
}

// HasUsers returns true if any local users exist
func (s *UserStore) HasUsers(ctx context.Context) (bool, error) {
	var count int
	err := s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM local_users`).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateUser creates a new local user
func (s *UserStore) CreateUser(ctx context.Context, username, password, email string, isAdmin bool) error {
	hash, err := s.hashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = s.db.Pool.Exec(ctx, `
		INSERT INTO local_users (username, password_hash, email, is_admin)
		VALUES ($1, $2, $3, $4)
	`, username, hash, email, isAdmin)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return ErrUserExists
	}
	return err
}

// GetUser returns a user by username
func (s *UserStore) GetUser(ctx context.Context, username string) (*LocalUser, error) {
	var u LocalUser
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, username, password_hash, email, is_admin, last_login_at, created_at
		FROM local_users WHERE username = $1
	`, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Email, &u.IsAdmin, &u.LastLoginAt, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByID retrieves a local user by ID
func (s *UserStore) GetUserByID(ctx context.Context, id string) (*LocalUser, error) {
	var user LocalUser
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, username, password_hash, email, is_admin, last_login_at, created_at
		FROM local_users WHERE id = $1
	`, id).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Email, &user.IsAdmin, &user.LastLoginAt, &user.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// DeleteLocalUser deletes a local user by ID
func (s *UserStore) DeleteLocalUser(ctx context.Context, id string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM local_users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// Authenticate validates username and password
func (s *UserStore) Authenticate(ctx context.Context, username, password string) (*LocalUser, error) {
	user, err := s.GetUser(ctx, username)
	if err == ErrUserNotFound {
		// Still do password comparison to prevent timing attacks
		_, _ = s.hashPassword(password) // Ignore result, just for timing
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if !s.verifyPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Update last login (best effort, don't fail auth if this fails)
	_, _ = s.db.Pool.Exec(ctx, `UPDATE local_users SET last_login_at = NOW() WHERE id = $1`, user.ID)

	return user, nil
}

// UpdatePassword updates a user's password
func (s *UserStore) UpdatePassword(ctx context.Context, username, newPassword string) error {
	hash, err := s.hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	result, err := s.db.Pool.Exec(ctx, `
		UPDATE local_users SET password_hash = $2 WHERE username = $1
	`, username, hash)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// InitDefaultAdmin creates a default admin user if no users exist
func (s *UserStore) InitDefaultAdmin(ctx context.Context) (string, bool, error) {
	hasUsers, err := s.HasUsers(ctx)
	if err != nil {
		return "", false, err
	}
	if hasUsers {
		return "", false, nil
	}

	// Generate a random password
	passwordBytes := make([]byte, 16)
	rand.Read(passwordBytes)
	password := base64.RawURLEncoding.EncodeToString(passwordBytes)

	err = s.CreateUser(ctx, "admin", password, "admin@localhost", true)
	if err != nil {
		return "", false, err
	}

	return password, true, nil
}

// Session operations

// CreateSession creates a new admin session
func (s *UserStore) CreateSession(ctx context.Context, userID, token string, expiresAt time.Time, ipAddress, userAgent string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO admin_sessions (user_id, token, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3::inet, $4, $5)
	`, userID, token, ipAddress, userAgent, expiresAt)
	return err
}

// GetSession retrieves a session by token
func (s *UserStore) GetSession(ctx context.Context, token string) (*AdminSession, *LocalUser, error) {
	var session AdminSession
	var user LocalUser

	err := s.db.Pool.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.token, s.expires_at, s.created_at,
		       u.id, u.username, u.email, u.is_admin, u.last_login_at, u.created_at
		FROM admin_sessions s
		JOIN local_users u ON s.user_id = u.id
		WHERE s.token = $1
	`, token).Scan(
		&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt,
		&user.ID, &user.Username, &user.Email, &user.IsAdmin, &user.LastLoginAt, &user.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		// Clean up expired session (best effort)
		_ = s.DeleteSession(ctx, token)
		return nil, nil, ErrSessionExpired
	}

	return &session, &user, nil
}

// DeleteSession removes a session
func (s *UserStore) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM admin_sessions WHERE token = $1`, token)
	return err
}

// CleanupExpiredSessions removes all expired sessions
func (s *UserStore) CleanupExpiredSessions(ctx context.Context) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM admin_sessions WHERE expires_at < NOW()`)
	return err
}

// Password hashing helpers

func (s *UserStore) hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, s.time, s.memory, s.threads, s.keyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, s.memory, s.time, s.threads, b64Salt, b64Hash), nil
}

func (s *UserStore) verifyPassword(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	var version int
	var memory, time uint32
	var threads uint8

	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false
	}

	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(expectedHash)))

	return subtle.ConstantTimeCompare(hash, expectedHash) == 1
}

// SSO User operations

// ListSSOUsers returns all SSO users
func (s *UserStore) ListSSOUsers(ctx context.Context) ([]*SSOUser, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, external_id, provider, email, name, groups, is_admin, is_active, last_login_at, created_at, updated_at
		FROM users
		ORDER BY email
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*SSOUser
	for rows.Next() {
		var u SSOUser
		var groupsJSON []byte
		if err := rows.Scan(&u.ID, &u.ExternalID, &u.Provider, &u.Email, &u.Name,
			&groupsJSON, &u.IsAdmin, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		if len(groupsJSON) > 0 {
			json.Unmarshal(groupsJSON, &u.Groups)
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

// GetSSOUser returns an SSO user by ID
func (s *UserStore) GetSSOUser(ctx context.Context, id string) (*SSOUser, error) {
	var u SSOUser
	var groupsJSON []byte
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, external_id, provider, email, name, groups, is_admin, is_active, last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.ExternalID, &u.Provider, &u.Email, &u.Name,
		&groupsJSON, &u.IsAdmin, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(groupsJSON) > 0 {
		json.Unmarshal(groupsJSON, &u.Groups)
	}
	return &u, nil
}

// GetSSOUserByEmail returns an SSO user by email
func (s *UserStore) GetSSOUserByEmail(ctx context.Context, email string) (*SSOUser, error) {
	var u SSOUser
	var groupsJSON []byte
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, external_id, provider, email, name, groups, is_admin, is_active, last_login_at, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.ExternalID, &u.Provider, &u.Email, &u.Name,
		&groupsJSON, &u.IsAdmin, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(groupsJSON) > 0 {
		json.Unmarshal(groupsJSON, &u.Groups)
	}
	return &u, nil
}

// GetLocalUserByEmail retrieves a local user by email
func (s *UserStore) GetLocalUserByEmail(ctx context.Context, email string) (*LocalUser, error) {
	var u LocalUser
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, username, email, is_admin, last_login_at, created_at
		FROM local_users WHERE email = $1
	`, email).Scan(&u.ID, &u.Username, &u.Email, &u.IsAdmin, &u.LastLoginAt, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetLocalUserByUsername retrieves a local user by username
func (s *UserStore) GetLocalUserByUsername(ctx context.Context, username string) (*LocalUser, error) {
	var u LocalUser
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, username, email, is_admin, last_login_at, created_at
		FROM local_users WHERE username = $1
	`, username).Scan(&u.ID, &u.Username, &u.Email, &u.IsAdmin, &u.LastLoginAt, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ListLocalUsers returns all local admin users
func (s *UserStore) ListLocalUsers(ctx context.Context) ([]*LocalUser, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, username, email, is_admin, last_login_at, created_at
		FROM local_users
		ORDER BY username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*LocalUser
	for rows.Next() {
		var u LocalUser
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.IsAdmin, &u.LastLoginAt, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

// ListAllGroups returns all unique group names from SSO users and group_access_rules
func (s *UserStore) ListAllGroups(ctx context.Context) ([]string, error) {
	// Get unique groups from both SSO user groups and group_access_rules
	rows, err := s.db.Pool.Query(ctx, `
		SELECT DISTINCT group_name FROM (
			SELECT jsonb_array_elements_text(groups) as group_name FROM users
			UNION
			SELECT group_name FROM group_access_rules
			UNION
			SELECT group_name FROM group_gateways
		) all_groups
		WHERE group_name IS NOT NULL AND group_name != ''
		ORDER BY group_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// GetGroupMembers returns all SSO users that belong to a specific group
func (s *UserStore) GetGroupMembers(ctx context.Context, groupName string) ([]*SSOUser, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, external_id, provider, email, name, groups, is_admin, is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE groups ? $1
		ORDER BY email
	`, groupName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*SSOUser
	for rows.Next() {
		var u SSOUser
		var groupsJSON []byte
		if err := rows.Scan(&u.ID, &u.ExternalID, &u.Provider, &u.Email, &u.Name,
			&groupsJSON, &u.IsAdmin, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		if len(groupsJSON) > 0 {
			json.Unmarshal(groupsJSON, &u.Groups)
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

// UpsertSSOUser creates or updates an SSO user in the database
// This is called during SSO login to persist user information
func (s *UserStore) UpsertSSOUser(ctx context.Context, externalID, provider, email, name string, groups []string, isAdmin bool) (*SSOUser, error) {
	groupsJSON, err := json.Marshal(groups)
	if err != nil {
		return nil, err
	}

	var u SSOUser
	var groupsOut []byte
	err = s.db.Pool.QueryRow(ctx, `
		INSERT INTO users (external_id, provider, email, name, groups, is_admin, is_active, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, true, NOW())
		ON CONFLICT (provider, external_id) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			groups = EXCLUDED.groups,
			is_admin = COALESCE(users.is_admin, EXCLUDED.is_admin),
			last_login_at = NOW(),
			updated_at = NOW()
		RETURNING id, external_id, provider, email, name, groups, is_admin, is_active, last_login_at, created_at, updated_at
	`, externalID, provider, email, name, groupsJSON, isAdmin).Scan(
		&u.ID, &u.ExternalID, &u.Provider, &u.Email, &u.Name,
		&groupsOut, &u.IsAdmin, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(groupsOut) > 0 {
		json.Unmarshal(groupsOut, &u.Groups)
	}
	return &u, nil
}
