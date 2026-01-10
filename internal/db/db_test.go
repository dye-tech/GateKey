package db

import (
	"testing"
)

func TestTLSConfig_Fields(t *testing.T) {
	cfg := TLSConfig{
		SSLMode:     "verify-full",
		SSLRootCert: "/path/to/ca.crt",
		SSLCert:     "/path/to/client.crt",
		SSLKey:      "/path/to/client.key",
	}

	if cfg.SSLMode != "verify-full" {
		t.Errorf("Expected SSLMode 'verify-full', got %q", cfg.SSLMode)
	}
	if cfg.SSLRootCert != "/path/to/ca.crt" {
		t.Errorf("Expected SSLRootCert '/path/to/ca.crt', got %q", cfg.SSLRootCert)
	}
	if cfg.SSLCert != "/path/to/client.crt" {
		t.Errorf("Expected SSLCert '/path/to/client.crt', got %q", cfg.SSLCert)
	}
	if cfg.SSLKey != "/path/to/client.key" {
		t.Errorf("Expected SSLKey '/path/to/client.key', got %q", cfg.SSLKey)
	}
}

func TestBuildTLSConfig_Require(t *testing.T) {
	cfg := &TLSConfig{
		SSLMode: "require",
	}

	tlsConfig, err := buildTLSConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to build TLS config: %v", err)
	}

	// In 'require' mode, InsecureSkipVerify should be true (encrypt but don't verify)
	if !tlsConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true for 'require' mode")
	}

	// MinVersion should be TLS 1.2
	if tlsConfig.MinVersion != 0x0303 { // TLS 1.2
		t.Errorf("Expected MinVersion TLS 1.2, got %x", tlsConfig.MinVersion)
	}
}

func TestBuildTLSConfig_VerifyCA_NoRootCert(t *testing.T) {
	cfg := &TLSConfig{
		SSLMode: "verify-ca",
		// No SSLRootCert provided
	}

	tlsConfig, err := buildTLSConfig(cfg)
	if err != nil {
		t.Fatalf("buildTLSConfig should not error with missing root cert (file read would fail later): %v", err)
	}

	// InsecureSkipVerify should be false for verify modes
	if tlsConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be false for 'verify-ca' mode")
	}
}

func TestBuildTLSConfig_VerifyFull(t *testing.T) {
	cfg := &TLSConfig{
		SSLMode: "verify-full",
	}

	tlsConfig, err := buildTLSConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to build TLS config: %v", err)
	}

	// In 'verify-full' mode, InsecureSkipVerify should be false
	if tlsConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be false for 'verify-full' mode")
	}
}

func TestBuildTLSConfig_InvalidRootCert(t *testing.T) {
	cfg := &TLSConfig{
		SSLMode:     "verify-ca",
		SSLRootCert: "/nonexistent/path/to/ca.crt",
	}

	_, err := buildTLSConfig(cfg)
	if err == nil {
		t.Error("Expected error for nonexistent CA certificate file")
	}
}

func TestBuildTLSConfig_InvalidClientCert(t *testing.T) {
	cfg := &TLSConfig{
		SSLMode: "require",
		SSLCert: "/nonexistent/client.crt",
		SSLKey:  "/nonexistent/client.key",
	}

	_, err := buildTLSConfig(cfg)
	if err == nil {
		t.Error("Expected error for nonexistent client certificate files")
	}
}
