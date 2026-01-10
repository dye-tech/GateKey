package db

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	rawKey, keyHash, keyPrefix, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Check raw key format
	if !strings.HasPrefix(rawKey, "gk_") {
		t.Errorf("Raw key should start with 'gk_', got '%s'", rawKey[:10])
	}

	// Check key length is reasonable (gk_ prefix + base64 encoded 32 bytes)
	if len(rawKey) < 40 {
		t.Errorf("Raw key too short: %d chars", len(rawKey))
	}

	// Check hash length (SHA-256 = 64 hex chars)
	if len(keyHash) != 64 {
		t.Errorf("Key hash should be 64 chars, got %d", len(keyHash))
	}

	// Check prefix length (first 12 chars)
	if len(keyPrefix) != 12 {
		t.Errorf("Key prefix should be 12 chars, got %d", len(keyPrefix))
	}

	// Verify prefix matches raw key start
	if !strings.HasPrefix(rawKey, keyPrefix) {
		t.Error("Key prefix should match start of raw key")
	}
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	keys := make(map[string]bool)

	// Generate 100 keys and ensure all are unique
	for i := 0; i < 100; i++ {
		rawKey, keyHash, _, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("Failed to generate API key: %v", err)
		}

		if keys[rawKey] {
			t.Errorf("Duplicate raw key generated: %s", rawKey[:20])
		}
		keys[rawKey] = true

		if keys[keyHash] {
			t.Errorf("Duplicate key hash generated: %s", keyHash[:20])
		}
		keys[keyHash] = true
	}
}

func TestHashAPIKey(t *testing.T) {
	rawKey := "gk_test_api_key_12345"

	hash1 := HashAPIKey(rawKey)
	hash2 := HashAPIKey(rawKey)

	// Hash should be deterministic
	if hash1 != hash2 {
		t.Error("Hash should be deterministic")
	}

	// Hash should be 64 chars (SHA-256)
	if len(hash1) != 64 {
		t.Errorf("Hash should be 64 chars, got %d", len(hash1))
	}

	// Different keys should have different hashes
	differentHash := HashAPIKey("gk_different_key_67890")
	if hash1 == differentHash {
		t.Error("Different keys should have different hashes")
	}
}

func TestHashAPIKey_EmptyKey(t *testing.T) {
	hash := HashAPIKey("")

	// Empty key should still produce a valid hash
	if len(hash) != 64 {
		t.Errorf("Hash of empty key should be 64 chars, got %d", len(hash))
	}
}

func TestAPIKeyStore_New(t *testing.T) {
	// Test that NewAPIKeyStore doesn't panic with nil db
	store := NewAPIKeyStore(nil)

	if store == nil {
		t.Error("Store should not be nil")
	}
}

func TestAPIKeyErrors(t *testing.T) {
	// Ensure error messages are meaningful
	if ErrAPIKeyNotFound.Error() == "" {
		t.Error("ErrAPIKeyNotFound should have an error message")
	}

	if ErrAPIKeyRevoked.Error() == "" {
		t.Error("ErrAPIKeyRevoked should have an error message")
	}

	if ErrAPIKeyExpired.Error() == "" {
		t.Error("ErrAPIKeyExpired should have an error message")
	}
}

func TestAPIKeyPrefixFormat(t *testing.T) {
	// Generate several keys and verify prefix format consistency
	for i := 0; i < 10; i++ {
		rawKey, _, keyPrefix, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("Failed to generate API key: %v", err)
		}

		// Prefix should start with gk_
		if !strings.HasPrefix(keyPrefix, "gk_") {
			t.Errorf("Prefix should start with 'gk_', got '%s'", keyPrefix)
		}

		// Prefix should be exactly 12 chars
		if len(keyPrefix) != 12 {
			t.Errorf("Prefix should be 12 chars, got %d", len(keyPrefix))
		}

		// Raw key should be URL-safe (no + or /)
		remainder := strings.TrimPrefix(rawKey, "gk_")
		if strings.Contains(remainder, "+") || strings.Contains(remainder, "/") {
			t.Error("Raw key should use URL-safe base64 encoding")
		}
	}
}
