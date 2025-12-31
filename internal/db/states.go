package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

// OAuthState represents an OAuth state for OIDC/SAML flows
type OAuthState struct {
	State          string
	Provider       string
	ProviderType   string // "oidc" or "saml"
	Nonce          string
	RelayState     string
	CLICallbackURL string // For CLI login flow
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// StateStore handles OAuth state persistence
type StateStore struct {
	db *DB
}

// NewStateStore creates a new state store
func NewStateStore(db *DB) *StateStore {
	return &StateStore{db: db}
}

// SaveState stores an OAuth state
func (s *StateStore) SaveState(ctx context.Context, state *OAuthState) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO oauth_states (state, provider, provider_type, nonce, relay_state, cli_callback_url, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, state.State, state.Provider, state.ProviderType, state.Nonce, state.RelayState, state.CLICallbackURL, state.ExpiresAt)
	return err
}

// GetState retrieves and deletes an OAuth state (one-time use)
func (s *StateStore) GetState(ctx context.Context, state string) (*OAuthState, error) {
	var st OAuthState
	var cliCallbackURL *string
	err := s.db.Pool.QueryRow(ctx, `
		DELETE FROM oauth_states
		WHERE state = $1
		RETURNING state, provider, provider_type, nonce, relay_state, cli_callback_url, expires_at, created_at
	`, state).Scan(&st.State, &st.Provider, &st.ProviderType, &st.Nonce, &st.RelayState, &cliCallbackURL, &st.ExpiresAt, &st.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	if cliCallbackURL != nil {
		st.CLICallbackURL = *cliCallbackURL
	}

	// Check if expired
	if time.Now().After(st.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	return &st, nil
}

// SaveCLICallback stores a CLI callback URL for a state
func (s *StateStore) SaveCLICallback(ctx context.Context, state, callbackURL string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO oauth_states (state, provider, provider_type, cli_callback_url, expires_at)
		VALUES ($1, 'cli', 'cli', $2, $3)
	`, state, callbackURL, time.Now().Add(10*time.Minute))
	return err
}

// GetCLICallback retrieves and deletes a CLI callback URL
func (s *StateStore) GetCLICallback(ctx context.Context, state string) (string, error) {
	var callbackURL string
	err := s.db.Pool.QueryRow(ctx, `
		DELETE FROM oauth_states
		WHERE state = $1 AND provider_type = 'cli'
		RETURNING cli_callback_url
	`, state).Scan(&callbackURL)
	if err == pgx.ErrNoRows {
		return "", ErrSessionNotFound
	}
	if err != nil {
		return "", err
	}
	return callbackURL, nil
}

// CleanupExpiredStates removes expired states
func (s *StateStore) CleanupExpiredStates(ctx context.Context) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM oauth_states WHERE expires_at < NOW()`)
	return err
}

// SSOSession represents an SSO user session
type SSOSession struct {
	Token     string
	UserID    string
	Username  string
	Email     string
	Name      string
	Groups    []string
	Provider  string
	IsAdmin   bool
	ExpiresAt time.Time
	CreatedAt time.Time
}

// SaveSSOSession stores an SSO session
func (s *StateStore) SaveSSOSession(ctx context.Context, session *SSOSession) error {
	groupsJSON, _ := json.Marshal(session.Groups)
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO sso_sessions (token, user_id, username, email, name, groups, provider, is_admin, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (token) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			username = EXCLUDED.username,
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			groups = EXCLUDED.groups,
			provider = EXCLUDED.provider,
			is_admin = EXCLUDED.is_admin,
			expires_at = EXCLUDED.expires_at
	`, session.Token, session.UserID, session.Username, session.Email, session.Name, groupsJSON, session.Provider, session.IsAdmin, session.ExpiresAt)
	return err
}

// GetSSOSession retrieves an SSO session by token
func (s *StateStore) GetSSOSession(ctx context.Context, token string) (*SSOSession, error) {
	var session SSOSession
	var groupsJSON []byte
	err := s.db.Pool.QueryRow(ctx, `
		SELECT token, user_id, username, email, name, groups, provider, is_admin, expires_at, created_at
		FROM sso_sessions
		WHERE token = $1
	`, token).Scan(&session.Token, &session.UserID, &session.Username, &session.Email, &session.Name, &groupsJSON, &session.Provider, &session.IsAdmin, &session.ExpiresAt, &session.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		_ = s.DeleteSSOSession(ctx, token) // Best effort cleanup
		return nil, ErrSessionExpired
	}

	json.Unmarshal(groupsJSON, &session.Groups)
	return &session, nil
}

// DeleteSSOSession removes an SSO session
func (s *StateStore) DeleteSSOSession(ctx context.Context, token string) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM sso_sessions WHERE token = $1`, token)
	return err
}

// CleanupExpiredSSOSessions removes expired SSO sessions
func (s *StateStore) CleanupExpiredSSOSessions(ctx context.Context) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM sso_sessions WHERE expires_at < NOW()`)
	return err
}
