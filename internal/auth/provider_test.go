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

func TestMockProviderLoginURL(t *testing.T) {
	provider := &mockAuthProvider{
		name:        "test",
		provType:    "oidc",
		displayName: "Test Provider",
	}

	state := "random-state-123"
	loginURL, err := provider.LoginURL(state)
	if err != nil {
		t.Fatalf("Failed to get login URL: %v", err)
	}

	expected := "https://example.com/login?state=" + state
	if loginURL != expected {
		t.Errorf("Expected %q, got %q", expected, loginURL)
	}
}

func TestMockProviderHandleCallback(t *testing.T) {
	provider := &mockAuthProvider{
		name:        "test",
		provType:    "oidc",
		displayName: "Test Provider",
	}

	userInfo, err := provider.HandleCallback(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to handle callback: %v", err)
	}

	if userInfo == nil {
		t.Fatal("UserInfo should not be nil")
	}

	if userInfo.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %q", userInfo.Email)
	}

	if userInfo.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got %q", userInfo.Name)
	}
}

func TestRegisterMultipleProviders(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)

	providers := []*mockAuthProvider{
		{name: "google", provType: "oidc", displayName: "Google"},
		{name: "azure", provType: "oidc", displayName: "Azure AD"},
		{name: "okta", provType: "saml", displayName: "Okta"},
	}

	for _, p := range providers {
		manager.RegisterProvider(p)
	}

	list := manager.ListProviders()
	if len(list) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(list))
	}

	// Verify we can get each provider
	for _, p := range providers {
		got, err := manager.GetProvider(p.provType, p.name)
		if err != nil {
			t.Errorf("Failed to get provider %s:%s: %v", p.provType, p.name, err)
		}
		if got.Name() != p.name {
			t.Errorf("Expected name %q, got %q", p.name, got.Name())
		}
	}
}

func TestGetProvider_NotFound(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)

	tests := []struct {
		name  string
		pType string
		pName string
	}{
		{"non-existent oidc", "oidc", "nonexistent"},
		{"non-existent saml", "saml", "unknown"},
		{"wrong type", "ldap", "test"},
		{"empty name", "oidc", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.GetProvider(tt.pType, tt.pName)
			if err == nil {
				t.Error("Expected error for non-existent provider")
			}
		})
	}
}

func TestGenerateSecureToken_DifferentLengths(t *testing.T) {
	lengths := []int{8, 16, 32, 64}

	for _, length := range lengths {
		t.Run("length_"+string(rune('0'+length)), func(t *testing.T) {
			token, err := generateSecureToken(length)
			if err != nil {
				t.Fatalf("Failed to generate token of length %d: %v", length, err)
			}

			// Base64 encoding increases size: output = ceil(input * 4/3)
			// URL-safe base64 doesn't use padding by default in Go
			if len(token) == 0 {
				t.Errorf("Token for length %d should not be empty", length)
			}
		})
	}
}

func TestHashToken_EmptyInput(t *testing.T) {
	hash := hashToken("")

	// Even empty string should produce valid hash
	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}
}

func TestHashToken_Consistency(t *testing.T) {
	testCases := []string{
		"simple",
		"with spaces",
		"special!@#$%^&*()",
		"unicode-日本語",
		"very-long-token-" + string(make([]byte, 1000)),
	}

	for _, input := range testCases {
		hash1 := hashToken(input)
		hash2 := hashToken(input)

		if hash1 != hash2 {
			t.Errorf("Hash should be deterministic for input %q", input[:min(20, len(input))])
		}
	}
}

func TestUserInfo_Fields(t *testing.T) {
	ui := &UserInfo{
		ExternalID: "ext-123",
		Email:      "user@example.com",
		Name:       "Test User",
		Groups:     []string{"admin", "users"},
		Provider:   "oidc:google",
		Attributes: map[string]interface{}{
			"department":  "engineering",
			"employee_id": 12345,
		},
	}

	if ui.ExternalID != "ext-123" {
		t.Errorf("Expected ExternalID 'ext-123', got %q", ui.ExternalID)
	}

	if len(ui.Groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(ui.Groups))
	}

	if ui.Attributes["department"] != "engineering" {
		t.Errorf("Expected department 'engineering', got %v", ui.Attributes["department"])
	}
}

func TestProviderInfo_Fields(t *testing.T) {
	info := ProviderInfo{
		Type:        "oidc",
		Name:        "google",
		DisplayName: "Google Workspace",
	}

	if info.Type != "oidc" {
		t.Errorf("Expected Type 'oidc', got %q", info.Type)
	}

	if info.Name != "google" {
		t.Errorf("Expected Name 'google', got %q", info.Name)
	}

	if info.DisplayName != "Google Workspace" {
		t.Errorf("Expected DisplayName 'Google Workspace', got %q", info.DisplayName)
	}
}

func TestListProviders_Empty(t *testing.T) {
	cfg := &config.AuthConfig{}
	manager := NewManager(cfg, nil, nil)

	providers := manager.ListProviders()
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers for empty manager, got %d", len(providers))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
