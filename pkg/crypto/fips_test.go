package crypto

import (
	"bytes"
	"testing"
)

func TestHash(t *testing.T) {
	data := []byte("test data for hashing")

	tests := []struct {
		alg       HashAlgorithm
		expectLen int
	}{
		{SHA256, 32},
		{SHA384, 48},
		{SHA512, 64},
	}

	for _, test := range tests {
		t.Run(string(test.alg), func(t *testing.T) {
			hash, err := Hash(test.alg, data)
			if err != nil {
				t.Fatalf("Hash failed: %v", err)
			}

			if len(hash) != test.expectLen {
				t.Errorf("Expected hash length %d, got %d", test.expectLen, len(hash))
			}

			// Hash should be deterministic
			hash2, _ := Hash(test.alg, data)
			if !bytes.Equal(hash, hash2) {
				t.Error("Hash should be deterministic")
			}

			// Different data should produce different hash
			hash3, _ := Hash(test.alg, []byte("different data"))
			if bytes.Equal(hash, hash3) {
				t.Error("Different data should produce different hash")
			}
		})
	}
}

func TestHashHex(t *testing.T) {
	data := []byte("test data")

	hex, err := HashHex(SHA256, data)
	if err != nil {
		t.Fatalf("HashHex failed: %v", err)
	}

	// SHA256 = 32 bytes = 64 hex characters
	if len(hex) != 64 {
		t.Errorf("Expected hex length 64, got %d", len(hex))
	}

	// Verify only contains hex characters
	for _, c := range hex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Invalid hex character: %c", c)
		}
	}
}

func TestSecureRandomBytes(t *testing.T) {
	lengths := []int{16, 32, 64, 128}

	for _, length := range lengths {
		t.Run(string(rune(length)), func(t *testing.T) {
			bytes1, err := SecureRandomBytes(length)
			if err != nil {
				t.Fatalf("SecureRandomBytes failed: %v", err)
			}

			if len(bytes1) != length {
				t.Errorf("Expected length %d, got %d", length, len(bytes1))
			}

			// Should be unique
			bytes2, _ := SecureRandomBytes(length)
			if bytes.Equal(bytes1, bytes2) {
				t.Error("Two random byte arrays should not be equal")
			}
		})
	}
}

func TestSecureRandomHex(t *testing.T) {
	hex, err := SecureRandomHex(16)
	if err != nil {
		t.Fatalf("SecureRandomHex failed: %v", err)
	}

	// 16 bytes = 32 hex characters
	if len(hex) != 32 {
		t.Errorf("Expected hex length 32, got %d", len(hex))
	}

	// Should be unique
	hex2, _ := SecureRandomHex(16)
	if hex == hex2 {
		t.Error("Two random hex strings should not be equal")
	}
}

func TestConstantTimeCompare(t *testing.T) {
	a := []byte("test data 12345")
	b := []byte("test data 12345")
	c := []byte("different data!")
	d := []byte("short")

	if !ConstantTimeCompare(a, b) {
		t.Error("Identical slices should return true")
	}

	if ConstantTimeCompare(a, c) {
		t.Error("Different slices should return false")
	}

	if ConstantTimeCompare(a, d) {
		t.Error("Different length slices should return false")
	}
}

func TestValidateKeyAlgorithm(t *testing.T) {
	validAlgorithms := []KeyAlgorithm{RSA2048, RSA3072, RSA4096, ECDSA256, ECDSA384}

	for _, alg := range validAlgorithms {
		if err := ValidateKeyAlgorithm(alg); err != nil {
			t.Errorf("Algorithm %s should be valid: %v", alg, err)
		}
	}

	invalidAlgorithms := []KeyAlgorithm{"rsa1024", "dsa", "invalid"}

	for _, alg := range invalidAlgorithms {
		if err := ValidateKeyAlgorithm(alg); err == nil {
			t.Errorf("Algorithm %s should be invalid", alg)
		}
	}
}

func TestMinimumKeySize(t *testing.T) {
	tests := []struct {
		alg      KeyAlgorithm
		expected int
	}{
		{RSA2048, 2048},
		{RSA3072, 3072},
		{RSA4096, 4096},
		{ECDSA256, 256},
		{ECDSA384, 384},
		{"invalid", 0},
	}

	for _, test := range tests {
		result := MinimumKeySize(test.alg)
		if result != test.expected {
			t.Errorf("MinimumKeySize(%s) = %d, expected %d", test.alg, result, test.expected)
		}
	}
}

func TestGetFIPSStatus(t *testing.T) {
	status := GetFIPSStatus()

	// Basic sanity checks
	if status.Provider == "" {
		t.Error("Provider should not be empty")
	}

	if len(status.Algorithms) == 0 {
		t.Error("Should have at least one algorithm listed")
	}

	// Check that common algorithms are listed
	hasAES := false
	hasSHA := false
	for _, alg := range status.Algorithms {
		if alg == "AES-256-GCM" {
			hasAES = true
		}
		if alg == "SHA-256" {
			hasSHA = true
		}
	}

	if !hasAES {
		t.Error("AES-256-GCM should be in the algorithm list")
	}

	if !hasSHA {
		t.Error("SHA-256 should be in the algorithm list")
	}
}

func BenchmarkHash(b *testing.B) {
	data := make([]byte, 1024)

	b.Run("SHA256", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Hash(SHA256, data)
		}
	})

	b.Run("SHA384", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Hash(SHA384, data)
		}
	})

	b.Run("SHA512", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Hash(SHA512, data)
		}
	})
}

func BenchmarkSecureRandomBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SecureRandomBytes(32)
	}
}
