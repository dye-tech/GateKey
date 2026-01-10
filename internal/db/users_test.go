package db

import (
	"strings"
	"testing"
)

func TestNewUserStore(t *testing.T) {
	store := NewUserStore(nil)

	if store == nil {
		t.Fatal("Expected non-nil store")
	}
	// Verify Argon2 parameters are set
	if store.time != 1 {
		t.Errorf("Expected time=1, got %d", store.time)
	}
	if store.memory != 64*1024 {
		t.Errorf("Expected memory=64*1024, got %d", store.memory)
	}
	if store.threads != 4 {
		t.Errorf("Expected threads=4, got %d", store.threads)
	}
	if store.keyLen != 32 {
		t.Errorf("Expected keyLen=32, got %d", store.keyLen)
	}
}

func TestUserStore_HashPassword(t *testing.T) {
	store := NewUserStore(nil)

	hash, err := store.hashPassword("testpassword123")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Verify hash format (Argon2id)
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("Expected hash to start with '$argon2id$', got %q", hash[:20])
	}

	// Hash should have 6 parts separated by $
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Errorf("Expected 6 parts in hash, got %d", len(parts))
	}

	// Verify version part
	if !strings.HasPrefix(parts[2], "v=") {
		t.Errorf("Expected version in hash, got %q", parts[2])
	}

	// Verify parameters part
	if !strings.Contains(parts[3], "m=") || !strings.Contains(parts[3], "t=") || !strings.Contains(parts[3], "p=") {
		t.Errorf("Expected m, t, p parameters in hash, got %q", parts[3])
	}
}

func TestUserStore_HashPassword_Uniqueness(t *testing.T) {
	store := NewUserStore(nil)

	// Same password should produce different hashes (due to random salt)
	hash1, err := store.hashPassword("samepassword")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	hash2, err := store.hashPassword("samepassword")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash1 == hash2 {
		t.Error("Expected different hashes for same password (random salt)")
	}
}

func TestUserStore_VerifyPassword_Valid(t *testing.T) {
	store := NewUserStore(nil)
	password := "correctpassword123"

	hash, err := store.hashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !store.verifyPassword(password, hash) {
		t.Error("Expected password verification to succeed")
	}
}

func TestUserStore_VerifyPassword_Invalid(t *testing.T) {
	store := NewUserStore(nil)

	hash, err := store.hashPassword("correctpassword")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if store.verifyPassword("wrongpassword", hash) {
		t.Error("Expected password verification to fail with wrong password")
	}
}

func TestUserStore_VerifyPassword_EmptyPassword(t *testing.T) {
	store := NewUserStore(nil)

	hash, err := store.hashPassword("actualpassword")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if store.verifyPassword("", hash) {
		t.Error("Expected verification to fail with empty password")
	}
}

func TestUserStore_VerifyPassword_MalformedHash(t *testing.T) {
	store := NewUserStore(nil)

	// Test various malformed hashes
	malformedHashes := []string{
		"",
		"notahash",
		"$argon2id$",
		"$argon2id$v=19$",
		"$argon2id$v=19$m=65536$",
		"$argon2id$v=19$m=65536,t=1,p=4$",
		"invalid$parts$count",
	}

	for _, hash := range malformedHashes {
		if store.verifyPassword("password", hash) {
			t.Errorf("Expected verification to fail for malformed hash: %q", hash)
		}
	}
}

func TestUserStore_VerifyPassword_CorruptedSalt(t *testing.T) {
	store := NewUserStore(nil)

	// Hash with invalid base64 salt
	hash := "$argon2id$v=19$m=65536,t=1,p=4$!!!invalid!!!$validhash"
	if store.verifyPassword("password", hash) {
		t.Error("Expected verification to fail with corrupted salt")
	}
}

func TestUserStore_VerifyPassword_CorruptedHash(t *testing.T) {
	store := NewUserStore(nil)

	// Hash with invalid base64 hash
	hash := "$argon2id$v=19$m=65536,t=1,p=4$dmFsaWRzYWx0$!!!invalid!!!"
	if store.verifyPassword("password", hash) {
		t.Error("Expected verification to fail with corrupted hash")
	}
}

func TestUserErrors(t *testing.T) {
	// Verify error messages are meaningful
	if ErrUserNotFound.Error() == "" {
		t.Error("ErrUserNotFound should have an error message")
	}
	if ErrUserExists.Error() == "" {
		t.Error("ErrUserExists should have an error message")
	}
	if ErrInvalidCredentials.Error() == "" {
		t.Error("ErrInvalidCredentials should have an error message")
	}
	if ErrSessionNotFound.Error() == "" {
		t.Error("ErrSessionNotFound should have an error message")
	}
	if ErrSessionExpired.Error() == "" {
		t.Error("ErrSessionExpired should have an error message")
	}
}

func TestSSOUser_Fields(t *testing.T) {
	user := SSOUser{
		ID:         "uuid-123",
		ExternalID: "ext-456",
		Provider:   "google",
		Email:      "user@example.com",
		Name:       "Test User",
		Groups:     []string{"admin", "developers"},
		IsAdmin:    true,
		IsActive:   true,
	}

	if user.Email != "user@example.com" {
		t.Errorf("Expected email 'user@example.com', got %q", user.Email)
	}
	if len(user.Groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(user.Groups))
	}
	if !user.IsAdmin {
		t.Error("Expected IsAdmin to be true")
	}
}

func TestLocalUser_Fields(t *testing.T) {
	user := LocalUser{
		ID:           "uuid-123",
		Username:     "admin",
		PasswordHash: "hashed",
		Email:        "admin@localhost",
		IsAdmin:      true,
	}

	if user.Username != "admin" {
		t.Errorf("Expected username 'admin', got %q", user.Username)
	}
	if !user.IsAdmin {
		t.Error("Expected IsAdmin to be true")
	}
}

func TestAdminSession_Fields(t *testing.T) {
	session := AdminSession{
		ID:     "session-uuid",
		UserID: "user-uuid",
		Token:  "session-token",
	}

	if session.ID != "session-uuid" {
		t.Errorf("Expected ID 'session-uuid', got %q", session.ID)
	}
	if session.UserID != "user-uuid" {
		t.Errorf("Expected UserID 'user-uuid', got %q", session.UserID)
	}
}

func TestUserStore_HashPassword_SpecialCharacters(t *testing.T) {
	store := NewUserStore(nil)

	specialPasswords := []string{
		"password with spaces",
		"pässwörd",
		"密码测试",
		"password\nwith\nnewlines",
		"password\twith\ttabs",
		"p@$$w0rd!#$%^&*()",
		strings.Repeat("a", 1000), // very long password
	}

	for _, password := range specialPasswords {
		hash, err := store.hashPassword(password)
		if err != nil {
			t.Errorf("Failed to hash password with special chars: %v", err)
			continue
		}

		if !store.verifyPassword(password, hash) {
			t.Errorf("Failed to verify password with special chars: %q", password[:min(20, len(password))])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
