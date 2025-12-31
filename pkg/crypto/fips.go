// Package crypto provides FIPS-compliant cryptographic helpers for GateKey.
package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
)

// FIPS-approved algorithms for GateKey:
// - Key Agreement: ECDH (P-256, P-384)
// - Digital Signatures: ECDSA (P-256, P-384), RSA (2048+)
// - Hashing: SHA-256, SHA-384, SHA-512
// - Symmetric Encryption: AES-128-GCM, AES-256-GCM
// - Key Derivation: HKDF, PBKDF2

// HashAlgorithm represents a FIPS-approved hash algorithm.
type HashAlgorithm string

const (
	SHA256 HashAlgorithm = "sha256"
	SHA384 HashAlgorithm = "sha384"
	SHA512 HashAlgorithm = "sha512"
)

// NewHash creates a new hash function for the specified algorithm.
func NewHash(alg HashAlgorithm) (hash.Hash, error) {
	switch alg {
	case SHA256:
		return sha256.New(), nil
	case SHA384:
		return sha512.New384(), nil
	case SHA512:
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", alg)
	}
}

// Hash computes the hash of data using the specified algorithm.
func Hash(alg HashAlgorithm, data []byte) ([]byte, error) {
	h, err := NewHash(alg)
	if err != nil {
		return nil, err
	}
	h.Write(data)
	return h.Sum(nil), nil
}

// HashHex computes the hash and returns it as a hex string.
func HashHex(alg HashAlgorithm, data []byte) (string, error) {
	h, err := Hash(alg, data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h), nil
}

// SecureRandomBytes generates cryptographically secure random bytes.
func SecureRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}

// SecureRandomHex generates cryptographically secure random bytes as hex.
func SecureRandomHex(n int) (string, error) {
	b, err := SecureRandomBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ConstantTimeCompare compares two byte slices in constant time.
// This prevents timing attacks.
func ConstantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// KeyAlgorithm represents a FIPS-approved key algorithm.
type KeyAlgorithm string

const (
	RSA2048   KeyAlgorithm = "rsa2048"
	RSA3072   KeyAlgorithm = "rsa3072"
	RSA4096   KeyAlgorithm = "rsa4096"
	ECDSA256  KeyAlgorithm = "ecdsa256"
	ECDSA384  KeyAlgorithm = "ecdsa384"
)

// ValidateKeyAlgorithm checks if a key algorithm is FIPS-approved.
func ValidateKeyAlgorithm(alg KeyAlgorithm) error {
	switch alg {
	case RSA2048, RSA3072, RSA4096, ECDSA256, ECDSA384:
		return nil
	default:
		return fmt.Errorf("key algorithm %s is not FIPS-approved", alg)
	}
}

// MinimumKeySize returns the minimum FIPS-approved key size for an algorithm.
func MinimumKeySize(alg KeyAlgorithm) int {
	switch alg {
	case RSA2048:
		return 2048
	case RSA3072:
		return 3072
	case RSA4096:
		return 4096
	case ECDSA256:
		return 256
	case ECDSA384:
		return 384
	default:
		return 0
	}
}

// FIPSCompliance contains information about FIPS compliance status.
type FIPSCompliance struct {
	Enabled    bool   `json:"enabled"`
	Provider   string `json:"provider"`
	Version    string `json:"version"`
	Algorithms []string `json:"algorithms"`
}

// GetFIPSStatus returns the current FIPS compliance status.
// In production, this would check if FIPS mode is enabled in the OS/OpenSSL.
func GetFIPSStatus() FIPSCompliance {
	// TODO: Implement actual FIPS mode detection
	// This would typically check:
	// - /proc/sys/crypto/fips_enabled on Linux
	// - OpenSSL FIPS provider status
	// - Go crypto/tls FIPS mode (when using boringcrypto)

	return FIPSCompliance{
		Enabled:  false, // Would be true if FIPS mode detected
		Provider: "go-crypto",
		Version:  "1.23",
		Algorithms: []string{
			"RSA-2048", "RSA-3072", "RSA-4096",
			"ECDSA-P256", "ECDSA-P384",
			"SHA-256", "SHA-384", "SHA-512",
			"AES-128-GCM", "AES-256-GCM",
		},
	}
}
