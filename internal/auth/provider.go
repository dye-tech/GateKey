// Package auth provides authentication services for GateKey.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/gatekey-project/gatekey/internal/config"
	"github.com/gatekey-project/gatekey/internal/models"
)

// UserInfo represents authenticated user information from an IdP.
type UserInfo struct {
	ExternalID string
	Email      string
	Name       string
	Groups     []string
	Attributes map[string]interface{}
	Provider   string
}

// Provider defines the interface for authentication providers.
type Provider interface {
	// Name returns the provider name.
	Name() string

	// Type returns the provider type (oidc or saml).
	Type() string

	// DisplayName returns the display name for UI.
	DisplayName() string

	// LoginURL returns the URL to redirect users for login.
	LoginURL(state string) (string, error)

	// HandleCallback processes the authentication callback and returns user info.
	HandleCallback(ctx context.Context, r *http.Request) (*UserInfo, error)
}

// Manager manages authentication providers and sessions.
type Manager struct {
	config      *config.AuthConfig
	providers   map[string]Provider
	userRepo    *models.UserRepository
	sessionRepo *models.SessionRepository
}

// NewManager creates a new authentication manager.
func NewManager(cfg *config.AuthConfig, userRepo *models.UserRepository, sessionRepo *models.SessionRepository) *Manager {
	return &Manager{
		config:      cfg,
		providers:   make(map[string]Provider),
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

// RegisterProvider registers an authentication provider.
func (m *Manager) RegisterProvider(p Provider) {
	key := fmt.Sprintf("%s:%s", p.Type(), p.Name())
	m.providers[key] = p
}

// GetProvider returns a provider by type and name.
func (m *Manager) GetProvider(providerType, name string) (Provider, error) {
	key := fmt.Sprintf("%s:%s", providerType, name)
	p, ok := m.providers[key]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", key)
	}
	return p, nil
}

// ListProviders returns all registered providers with their login URLs.
func (m *Manager) ListProviders() []ProviderInfo {
	var providers []ProviderInfo
	for _, p := range m.providers {
		providers = append(providers, ProviderInfo{
			Type:        p.Type(),
			Name:        p.Name(),
			DisplayName: p.DisplayName(),
		})
	}
	return providers
}

// ProviderInfo contains provider metadata for the UI.
type ProviderInfo struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// CreateSession creates a new session for a user.
func (m *Manager) CreateSession(ctx context.Context, userInfo *UserInfo, ipAddress, userAgent string) (*models.Session, string, error) {
	// Upsert user
	user := &models.User{
		ExternalID: userInfo.ExternalID,
		Provider:   userInfo.Provider,
		Email:      userInfo.Email,
		Name:       userInfo.Name,
		Groups:     userInfo.Groups,
		IsActive:   true,
	}

	now := time.Now()
	user.LastLoginAt = &now

	if err := m.userRepo.Upsert(ctx, user); err != nil {
		return nil, "", fmt.Errorf("failed to upsert user: %w", err)
	}

	// Generate session token
	rawToken, err := generateSecureToken(32)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate session token: %w", err)
	}

	// Hash token for storage
	tokenHash := hashToken(rawToken)

	session := &models.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     tokenHash,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		ExpiresAt: time.Now().Add(m.config.Session.Validity),
	}

	if err := m.sessionRepo.Create(ctx, session); err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}

	return session, rawToken, nil
}

// ValidateSession validates a session token and returns the session and user.
func (m *Manager) ValidateSession(ctx context.Context, rawToken string) (*models.Session, *models.User, error) {
	tokenHash := hashToken(rawToken)

	session, err := m.sessionRepo.GetByToken(ctx, tokenHash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return nil, nil, nil // Session not found or expired
	}

	user, err := m.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil || !user.IsActive {
		return nil, nil, nil // User not found or inactive
	}

	return session, user, nil
}

// RevokeSession revokes a session.
func (m *Manager) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	return m.sessionRepo.Revoke(ctx, sessionID)
}

// RevokeAllUserSessions revokes all sessions for a user.
func (m *Manager) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	return m.sessionRepo.RevokeAllForUser(ctx, userID)
}

// GenerateState generates a secure random state for OAuth/SAML flows.
func GenerateState() (string, error) {
	return generateSecureToken(16)
}

// generateSecureToken generates a cryptographically secure random token.
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// hashToken creates a SHA-256 hash of a token for storage.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
