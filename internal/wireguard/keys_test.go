package wireguard

import (
	"encoding/base64"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// WireGuard keys should be 32 bytes (256 bits), base64 encoded
	privateBytes, err := base64.StdEncoding.DecodeString(keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to decode private key: %v", err)
	}
	if len(privateBytes) != 32 {
		t.Errorf("Expected private key to be 32 bytes, got %d", len(privateBytes))
	}

	publicBytes, err := base64.StdEncoding.DecodeString(keyPair.PublicKey)
	if err != nil {
		t.Fatalf("Failed to decode public key: %v", err)
	}
	if len(publicBytes) != 32 {
		t.Errorf("Expected public key to be 32 bytes, got %d", len(publicBytes))
	}

	// Private and public keys should be different
	if keyPair.PrivateKey == keyPair.PublicKey {
		t.Error("Private and public keys should be different")
	}
}

func TestGenerateKeyPairUniqueness(t *testing.T) {
	// Generate two key pairs and verify each has unique keys
	kp1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	kp2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	if kp1.PublicKey == kp2.PublicKey {
		t.Error("Two generated key pairs should have different public keys")
	}
	if kp1.PrivateKey == kp2.PrivateKey {
		t.Error("Two generated key pairs should have different private keys")
	}
}

func TestDerivePublicKey(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Derive public key from private key
	derivedPublicKey, err := DerivePublicKey(keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("DerivePublicKey failed: %v", err)
	}

	// Derived public key should match the originally generated public key
	if derivedPublicKey != keyPair.PublicKey {
		t.Errorf("Derived public key doesn't match: expected %s, got %s", keyPair.PublicKey, derivedPublicKey)
	}
}

func TestDerivePublicKeyInvalidInput(t *testing.T) {
	// Test with invalid base64
	_, err := DerivePublicKey("not-valid-base64!!!")
	if err == nil {
		t.Error("DerivePublicKey should fail with invalid base64")
	}

	// Test with wrong length key (too short)
	shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err = DerivePublicKey(shortKey)
	if err == nil {
		t.Error("DerivePublicKey should fail with wrong length key")
	}
}

func TestGeneratePresharedKey(t *testing.T) {
	psk, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey failed: %v", err)
	}

	// PSK should be 32 bytes (256 bits), base64 encoded
	pskBytes, err := base64.StdEncoding.DecodeString(psk)
	if err != nil {
		t.Fatalf("Failed to decode PSK: %v", err)
	}
	if len(pskBytes) != 32 {
		t.Errorf("Expected PSK to be 32 bytes, got %d", len(pskBytes))
	}
}

func TestGeneratePresharedKeyUniqueness(t *testing.T) {
	psk1, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey failed: %v", err)
	}

	psk2, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey failed: %v", err)
	}

	if psk1 == psk2 {
		t.Error("Two generated PSKs should be different")
	}
}

func TestValidateKey(t *testing.T) {
	// Generate a valid key
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Valid key should pass
	if err := ValidateKey(keyPair.PublicKey); err != nil {
		t.Errorf("Valid key should pass validation: %v", err)
	}

	// Invalid keys should fail
	invalidKeys := []struct {
		key     string
		wantErr bool
	}{
		{"", true},                    // empty
		{"short", true},               // too short
		{"not-valid-base64!!!", true}, // invalid base64
	}

	for _, tc := range invalidKeys {
		err := ValidateKey(tc.key)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateKey(%q) error = %v, wantErr = %v", tc.key, err, tc.wantErr)
		}
	}
}

func TestPrivateKeyIsClamped(t *testing.T) {
	// Generate multiple keys and verify they are properly clamped
	for i := 0; i < 10; i++ {
		keyPair, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair failed: %v", err)
		}

		privateBytes, err := base64.StdEncoding.DecodeString(keyPair.PrivateKey)
		if err != nil {
			t.Fatalf("Failed to decode private key: %v", err)
		}

		// Check clamping: first byte & 248, last byte & 127 | 64
		if privateBytes[0]&7 != 0 {
			t.Error("Private key not properly clamped (first byte)")
		}
		if privateBytes[31]&128 != 0 {
			t.Error("Private key not properly clamped (last byte high bit)")
		}
		if privateBytes[31]&64 == 0 {
			t.Error("Private key not properly clamped (last byte bit 6)")
		}
	}
}
