package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gatekey-project/gatekey/internal/config"
)

func TestGenerateSecureToken(t *testing.T) {
	token1, err := generateSecureToken(32)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if len(token1) == 0 {
		t.Error("Generated token is empty")
	}

	// Test uniqueness
	token2, err := generateSecureToken(32)
	if err != nil {
		t.Fatalf("Failed to generate second token: %v", err)
	}

	if token1 == token2 {
		t.Error("Two generated tokens should not be equal")
	}
}

func TestHashToken(t *testing.T) {
	token := "test-token-12345"

	hash1 := hashToken(token)
	hash2 := hashToken(token)

	if hash1 != hash2 {
		t.Error("Hash should be deterministic")
	}

	// Hash should be 64 characters (SHA256 = 32 bytes = 64 hex chars)
	if len(hash1) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}

	// Different tokens should have different hashes
	differentHash := hashToken("different-token")
	if hash1 == differentHash {
		t.Error("Different tokens should have different hashes")
	}
}

func TestGenerateState(t *testing.T) {
	state1, err := GenerateState()
	if err != nil {
		t.Fatalf("Failed to generate state: %v", err)
	}

	if len(state1) == 0 {
		t.Error("Generated state is empty")
	}

	state2, err := GenerateState()
	if err != nil {
		t.Fatalf("Failed to generate second state: %v", err)
	}

	if state1 == state2 {
		t.Error("Two generated states should not be equal")
	}
}

func TestNewManager(t *testing.T) {
	cfg := &config.AuthConfig{
		Session: config.SessionConfig{
			Validity:   12 * time.Hour,
			CookieName: "test_session",
			Secure:     true,
			HTTPOnly:   true,
			SameSite:   "lax",
		},
	}

	manager := NewManager(cfg, nil, nil)

	if manager == nil {
		t.Error("Manager should not be nil")
	}

	if manager.config != cfg {
		t.Error("Manager config should match")
	}
}

func TestListProviders(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)

	// Register a mock provider
	mockProvider := &mockAuthProvider{
		name:        "test",
		provType:    "oidc",
		displayName: "Test Provider",
	}
	manager.RegisterProvider(mockProvider)

	providers := manager.ListProviders()

	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	if providers[0].Name != "test" {
		t.Errorf("Expected provider name 'test', got '%s'", providers[0].Name)
	}

	if providers[0].Type != "oidc" {
		t.Errorf("Expected provider type 'oidc', got '%s'", providers[0].Type)
	}
}

func TestGetProvider(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)

	mockProvider := &mockAuthProvider{
		name:        "test",
		provType:    "oidc",
		displayName: "Test Provider",
	}
	manager.RegisterProvider(mockProvider)

	// Test getting existing provider
	provider, err := manager.GetProvider("oidc", "test")
	if err != nil {
		t.Errorf("Failed to get provider: %v", err)
	}
	if provider == nil {
		t.Error("Provider should not be nil")
	}

	// Test getting non-existing provider
	_, err = manager.GetProvider("oidc", "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existing provider")
	}
}

// mockAuthProvider is a mock implementation for testing.
type mockAuthProvider struct {
	name        string
	provType    string
	displayName string
}

func (m *mockAuthProvider) Name() string        { return m.name }
func (m *mockAuthProvider) Type() string        { return m.provType }
func (m *mockAuthProvider) DisplayName() string { return m.displayName }
func (m *mockAuthProvider) LoginURL(state string) (string, error) {
	return "https://example.com/login?state=" + state, nil
}
func (m *mockAuthProvider) HandleCallback(ctx context.Context, r *http.Request) (*UserInfo, error) {
	return &UserInfo{
		ExternalID: "123",
		Email:      "test@example.com",
		Name:       "Test User",
		Provider:   "mock",
	}, nil
}
