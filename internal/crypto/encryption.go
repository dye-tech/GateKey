// Package crypto provides cryptographic utilities for GateKey.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
)

const (
	// EncryptedPrefix is prepended to encrypted values to identify them.
	EncryptedPrefix = "enc:v1:"

	// KeySize is the required size for AES-256 keys (32 bytes).
	KeySize = 32

	// NonceSize is the size for AES-GCM nonces (12 bytes).
	NonceSize = 12
)

var (
	ErrInvalidKey       = errors.New("encryption key must be 32 bytes (256 bits)")
	ErrDecryptionFailed = errors.New("decryption failed: invalid ciphertext or key")
	ErrNotEncrypted     = errors.New("value is not encrypted")
	ErrEmptyValue       = errors.New("cannot encrypt empty value")
)

// KeyEncryptor handles encryption and decryption of sensitive data.
type KeyEncryptor struct {
	key []byte
}

// NewKeyEncryptor creates a new encryptor with the given base64-encoded key.
// The key must decode to exactly 32 bytes (256 bits) for AES-256.
func NewKeyEncryptor(base64Key string) (*KeyEncryptor, error) {
	if base64Key == "" {
		return nil, nil // No encryption configured
	}

	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 key: %w", err)
	}

	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	return &KeyEncryptor{key: key}, nil
}

// Encrypt encrypts the given plaintext using AES-256-GCM.
// Returns the encrypted value with the "enc:v1:" prefix.
func (e *KeyEncryptor) Encrypt(plaintext string) (string, error) {
	if e == nil || e.key == nil {
		return plaintext, nil // No encryption configured, return as-is
	}

	if plaintext == "" {
		return "", ErrEmptyValue
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode and prefix
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return EncryptedPrefix + encoded, nil
}

// Decrypt decrypts the given ciphertext that was encrypted with Encrypt.
// If the value doesn't have the encryption prefix, it's returned as-is
// (for backward compatibility with unencrypted data).
func (e *KeyEncryptor) Decrypt(ciphertext string) (string, error) {
	if e == nil || e.key == nil {
		return ciphertext, nil // No encryption configured, return as-is
	}

	// Check if this is an encrypted value
	if !strings.HasPrefix(ciphertext, EncryptedPrefix) {
		// Not encrypted, return as-is (backward compatibility)
		return ciphertext, nil
	}

	// Remove prefix and decode
	encoded := strings.TrimPrefix(ciphertext, EncryptedPrefix)
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	if len(data) < NonceSize {
		return "", ErrDecryptionFailed
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce and ciphertext
	nonce := data[:NonceSize]
	encryptedData := data[NonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// IsEncrypted checks if a value is encrypted (has the encryption prefix).
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, EncryptedPrefix)
}

// SecureWipe overwrites a byte slice with zeros to prevent memory recovery.
// Uses volatile write pattern to prevent compiler optimization from removing the wipe.
func SecureWipe(data []byte) {
	for i := range data {
		data[i] = 0
	}
	// Use runtime.KeepAlive to ensure the wipe isn't optimized away
	// The compiler cannot prove the data is unused after this call
	runtime.KeepAlive(&data)
}
