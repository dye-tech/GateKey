package crypto

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNewKeyEncryptor_ValidKey(t *testing.T) {
	// Generate a valid 32-byte key
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	base64Key := base64.StdEncoding.EncodeToString(key)

	encryptor, err := NewKeyEncryptor(base64Key)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if encryptor == nil {
		t.Fatal("Expected encryptor to be non-nil")
	}
}

func TestNewKeyEncryptor_EmptyKey(t *testing.T) {
	encryptor, err := NewKeyEncryptor("")
	if err != nil {
		t.Fatalf("Expected no error for empty key, got: %v", err)
	}
	if encryptor != nil {
		t.Fatal("Expected nil encryptor for empty key")
	}
}

func TestNewKeyEncryptor_InvalidBase64(t *testing.T) {
	_, err := NewKeyEncryptor("not-valid-base64!!!")
	if err == nil {
		t.Fatal("Expected error for invalid base64")
	}
}

func TestNewKeyEncryptor_WrongKeySize(t *testing.T) {
	// 16-byte key (too short)
	key := make([]byte, 16)
	base64Key := base64.StdEncoding.EncodeToString(key)

	_, err := NewKeyEncryptor(base64Key)
	if err != ErrInvalidKey {
		t.Fatalf("Expected ErrInvalidKey, got: %v", err)
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	base64Key := base64.StdEncoding.EncodeToString(key)

	encryptor, err := NewKeyEncryptor(base64Key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	testCases := []string{
		"Hello, World!",
		"-----BEGIN EC PRIVATE KEY-----\nMHQCAQEE...\n-----END EC PRIVATE KEY-----",
		"Short",
		strings.Repeat("Long text ", 1000),
	}

	for _, plaintext := range testCases {
		t.Run("", func(t *testing.T) {
			encrypted, err := encryptor.Encrypt(plaintext)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			// Verify prefix
			if !strings.HasPrefix(encrypted, EncryptedPrefix) {
				t.Errorf("Encrypted value should have prefix, got: %s", encrypted[:20])
			}

			// Verify it's different from plaintext
			if encrypted == plaintext {
				t.Error("Encrypted value should differ from plaintext")
			}

			// Decrypt
			decrypted, err := encryptor.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			if decrypted != plaintext {
				t.Errorf("Decrypted value doesn't match. Got %q, want %q", decrypted, plaintext)
			}
		})
	}
}

func TestEncrypt_EmptyValue(t *testing.T) {
	key := make([]byte, 32)
	base64Key := base64.StdEncoding.EncodeToString(key)

	encryptor, _ := NewKeyEncryptor(base64Key)

	_, err := encryptor.Encrypt("")
	if err != ErrEmptyValue {
		t.Fatalf("Expected ErrEmptyValue, got: %v", err)
	}
}

func TestDecrypt_UnencryptedValue(t *testing.T) {
	key := make([]byte, 32)
	base64Key := base64.StdEncoding.EncodeToString(key)

	encryptor, _ := NewKeyEncryptor(base64Key)

	// Unencrypted value should be returned as-is (backward compatibility)
	plaintext := "-----BEGIN EC PRIVATE KEY-----\nSome key data\n-----END EC PRIVATE KEY-----"
	result, err := encryptor.Decrypt(plaintext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if result != plaintext {
		t.Errorf("Expected plaintext to be returned as-is, got: %s", result)
	}
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)
	base64Key := base64.StdEncoding.EncodeToString(key)

	encryptor, _ := NewKeyEncryptor(base64Key)

	// Invalid encrypted value
	_, err := encryptor.Decrypt(EncryptedPrefix + "invalid-base64!!!")
	if err == nil {
		t.Fatal("Expected error for invalid ciphertext")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key1 {
		key1[i] = byte(i)
		key2[i] = byte(i + 1) // Different key
	}

	encryptor1, _ := NewKeyEncryptor(base64.StdEncoding.EncodeToString(key1))
	encryptor2, _ := NewKeyEncryptor(base64.StdEncoding.EncodeToString(key2))

	encrypted, _ := encryptor1.Encrypt("secret data")

	// Try to decrypt with wrong key
	_, err := encryptor2.Decrypt(encrypted)
	if err != ErrDecryptionFailed {
		t.Fatalf("Expected ErrDecryptionFailed, got: %v", err)
	}
}

func TestEncryptDecrypt_NilEncryptor(t *testing.T) {
	var encryptor *KeyEncryptor = nil

	// Encrypt with nil encryptor should return plaintext
	plaintext := "test data"
	result, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if result != plaintext {
		t.Errorf("Expected plaintext to be returned as-is")
	}

	// Decrypt with nil encryptor should return ciphertext
	result, err = encryptor.Decrypt(plaintext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if result != plaintext {
		t.Errorf("Expected ciphertext to be returned as-is")
	}
}

func TestIsEncrypted(t *testing.T) {
	if IsEncrypted("plain text") {
		t.Error("Plain text should not be detected as encrypted")
	}

	if !IsEncrypted(EncryptedPrefix + "base64data") {
		t.Error("Prefixed value should be detected as encrypted")
	}
}

func TestSecureWipe(t *testing.T) {
	data := []byte("sensitive data")
	SecureWipe(data)

	for i, b := range data {
		if b != 0 {
			t.Errorf("Byte at index %d should be 0, got %d", i, b)
		}
	}
}

func TestEncrypt_UniqueNonces(t *testing.T) {
	key := make([]byte, 32)
	base64Key := base64.StdEncoding.EncodeToString(key)
	encryptor, _ := NewKeyEncryptor(base64Key)

	plaintext := "same text"
	encrypted1, _ := encryptor.Encrypt(plaintext)
	encrypted2, _ := encryptor.Encrypt(plaintext)

	// Same plaintext should produce different ciphertexts (due to random nonces)
	if encrypted1 == encrypted2 {
		t.Error("Same plaintext should produce different ciphertexts")
	}
}
