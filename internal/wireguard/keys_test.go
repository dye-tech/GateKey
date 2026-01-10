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

func TestValidateKey_TooLong(t *testing.T) {
	// Test with a key that's too long (more than 32 bytes)
	longKey := make([]byte, 64)
	for i := range longKey {
		longKey[i] = byte(i)
	}
	longKeyBase64 := base64.StdEncoding.EncodeToString(longKey)

	err := ValidateKey(longKeyBase64)
	if err == nil {
		t.Error("ValidateKey should fail for key longer than 32 bytes")
	}
}

func TestValidateKey_ExactLength(t *testing.T) {
	// Test with exactly 32 bytes
	exactKey := make([]byte, 32)
	for i := range exactKey {
		exactKey[i] = byte(i)
	}
	exactKeyBase64 := base64.StdEncoding.EncodeToString(exactKey)

	err := ValidateKey(exactKeyBase64)
	if err != nil {
		t.Errorf("ValidateKey should pass for exactly 32 bytes: %v", err)
	}
}

func TestDerivePublicKey_Consistency(t *testing.T) {
	// Derive public key multiple times from same private key
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	var derivedKeys []string
	for i := 0; i < 5; i++ {
		derived, err := DerivePublicKey(keyPair.PrivateKey)
		if err != nil {
			t.Fatalf("DerivePublicKey failed: %v", err)
		}
		derivedKeys = append(derivedKeys, derived)
	}

	// All derived keys should be identical
	for i := 1; i < len(derivedKeys); i++ {
		if derivedKeys[i] != derivedKeys[0] {
			t.Errorf("Derived key %d doesn't match: %s vs %s", i, derivedKeys[i], derivedKeys[0])
		}
	}

	// And should match the original public key
	if derivedKeys[0] != keyPair.PublicKey {
		t.Errorf("Derived key doesn't match original: %s vs %s", derivedKeys[0], keyPair.PublicKey)
	}
}

func TestKeyPair_Struct(t *testing.T) {
	kp := KeyPair{
		PrivateKey: "test-private-key",
		PublicKey:  "test-public-key",
	}

	if kp.PrivateKey != "test-private-key" {
		t.Errorf("Expected PrivateKey 'test-private-key', got %q", kp.PrivateKey)
	}
	if kp.PublicKey != "test-public-key" {
		t.Errorf("Expected PublicKey 'test-public-key', got %q", kp.PublicKey)
	}
}

func TestGenerateKeyPair_KeysAreBase64(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Verify both keys are valid base64
	_, err = base64.StdEncoding.DecodeString(keyPair.PrivateKey)
	if err != nil {
		t.Errorf("Private key is not valid base64: %v", err)
	}

	_, err = base64.StdEncoding.DecodeString(keyPair.PublicKey)
	if err != nil {
		t.Errorf("Public key is not valid base64: %v", err)
	}

	// Verify key lengths (base64 encoded 32 bytes = 44 characters)
	if len(keyPair.PrivateKey) != 44 {
		t.Errorf("Expected private key length 44, got %d", len(keyPair.PrivateKey))
	}
	if len(keyPair.PublicKey) != 44 {
		t.Errorf("Expected public key length 44, got %d", len(keyPair.PublicKey))
	}
}

func TestGeneratePresharedKey_IsBase64(t *testing.T) {
	psk, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey failed: %v", err)
	}

	// Verify PSK is valid base64
	decoded, err := base64.StdEncoding.DecodeString(psk)
	if err != nil {
		t.Errorf("PSK is not valid base64: %v", err)
	}

	if len(decoded) != 32 {
		t.Errorf("Expected PSK decoded length 32, got %d", len(decoded))
	}

	// Verify PSK base64 length
	if len(psk) != 44 {
		t.Errorf("Expected PSK length 44, got %d", len(psk))
	}
}

func TestValidateKey_ValidPSK(t *testing.T) {
	// Generate a PSK and validate it
	psk, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey failed: %v", err)
	}

	err = ValidateKey(psk)
	if err != nil {
		t.Errorf("ValidateKey should pass for valid PSK: %v", err)
	}
}

func TestDerivePublicKey_WrongLengthButValidBase64(t *testing.T) {
	// Test with valid base64 but wrong length (16 bytes instead of 32)
	shortKey := make([]byte, 16)
	for i := range shortKey {
		shortKey[i] = byte(i)
	}
	shortKeyBase64 := base64.StdEncoding.EncodeToString(shortKey)

	_, err := DerivePublicKey(shortKeyBase64)
	if err == nil {
		t.Error("DerivePublicKey should fail for 16-byte key")
	}

	// Test with 64 bytes
	longKey := make([]byte, 64)
	for i := range longKey {
		longKey[i] = byte(i)
	}
	longKeyBase64 := base64.StdEncoding.EncodeToString(longKey)

	_, err = DerivePublicKey(longKeyBase64)
	if err == nil {
		t.Error("DerivePublicKey should fail for 64-byte key")
	}
}
