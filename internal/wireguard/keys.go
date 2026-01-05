// Package wireguard provides WireGuard VPN configuration and management.
package wireguard

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// KeyPair represents a WireGuard public/private key pair.
type KeyPair struct {
	PrivateKey string
	PublicKey  string
}

// GenerateKeyPair generates a new WireGuard keypair using Curve25519.
// The private key is clamped per WireGuard specification.
func GenerateKeyPair() (*KeyPair, error) {
	// Generate 32 random bytes for private key
	privateKey := make([]byte, 32)
	if _, err := rand.Read(privateKey); err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Clamp the private key per WireGuard/Curve25519 specification
	// This ensures the key is valid for X25519
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	// Derive public key using Curve25519
	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("failed to derive public key: %w", err)
	}

	return &KeyPair{
		PrivateKey: base64.StdEncoding.EncodeToString(privateKey),
		PublicKey:  base64.StdEncoding.EncodeToString(publicKey),
	}, nil
}

// GeneratePresharedKey generates a random 256-bit preshared key.
// Preshared keys provide an additional layer of symmetric-key crypto
// to the existing public-key crypto, for post-quantum resistance.
func GeneratePresharedKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate preshared key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// ValidateKey checks if a key is a valid base64-encoded 32-byte key.
func ValidateKey(key string) error {
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %w", err)
	}
	if len(decoded) != 32 {
		return fmt.Errorf("key must be 32 bytes, got %d", len(decoded))
	}
	return nil
}

// DerivePublicKey derives a public key from a private key.
func DerivePublicKey(privateKeyBase64 string) (string, error) {
	privateKey, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privateKey) != 32 {
		return "", fmt.Errorf("private key must be 32 bytes, got %d", len(privateKey))
	}

	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return "", fmt.Errorf("failed to derive public key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(publicKey), nil
}
