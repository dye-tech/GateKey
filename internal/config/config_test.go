package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_Validate_RequiresDBURL(t *testing.T) {
	cfg := &Config{}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for missing database URL")
	}

	if err.Error() != "database.url is required" {
		t.Errorf("Expected error about database.url, got: %s", err.Error())
	}
}

func TestConfig_Validate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			URL: "postgres://localhost/gatekey",
		},
		PKI: PKIConfig{
			KeyAlgorithm: "ecdsa256",
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}
}

func TestConfig_Validate_KeyAlgorithms(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
		wantErr   bool
	}{
		{"rsa2048 is valid", "rsa2048", false},
		{"rsa4096 is valid", "rsa4096", false},
		{"ecdsa256 is valid", "ecdsa256", false},
		{"ecdsa384 is valid", "ecdsa384", false},
		{"invalid algorithm", "rsa1024", true},
		{"empty algorithm", "", true},
		{"unknown algorithm", "ed25519", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Database: DatabaseConfig{
					URL: "postgres://localhost/gatekey",
				},
				PKI: PKIConfig{
					KeyAlgorithm: tt.algorithm,
				},
			}

			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Expected error for algorithm %s, got nil", tt.algorithm)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error for algorithm %s, got: %v", tt.algorithm, err)
			}
		})
	}
}

func TestConfig_Validate_OIDCEnabled(t *testing.T) {
	// OIDC enabled without providers should fail
	cfg := &Config{
		Database: DatabaseConfig{
			URL: "postgres://localhost/gatekey",
		},
		PKI: PKIConfig{
			KeyAlgorithm: "ecdsa256",
		},
		Auth: AuthConfig{
			OIDC: OIDCConfig{
				Enabled:   true,
				Providers: []OIDCProvider{},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error when OIDC enabled without providers")
	}

	// OIDC enabled with providers should pass
	cfg.Auth.OIDC.Providers = []OIDCProvider{
		{
			Name:     "google",
			Issuer:   "https://accounts.google.com",
			ClientID: "client-id",
		},
	}

	err = cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error with OIDC provider configured, got: %v", err)
	}
}

func TestConfig_Validate_SAMLEnabled(t *testing.T) {
	// SAML enabled without providers should fail
	cfg := &Config{
		Database: DatabaseConfig{
			URL: "postgres://localhost/gatekey",
		},
		PKI: PKIConfig{
			KeyAlgorithm: "ecdsa256",
		},
		Auth: AuthConfig{
			SAML: SAMLConfig{
				Enabled:   true,
				Providers: []SAMLProvider{},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error when SAML enabled without providers")
	}

	// SAML enabled with providers should pass
	cfg.Auth.SAML.Providers = []SAMLProvider{
		{
			Name:           "okta",
			IDPMetadataURL: "https://example.okta.com/metadata",
			EntityID:       "https://gatekey.example.com",
		},
	}

	err = cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error with SAML provider configured, got: %v", err)
	}
}

func TestConfig_Validate_OIDCDisabled(t *testing.T) {
	// OIDC disabled should not require providers
	cfg := &Config{
		Database: DatabaseConfig{
			URL: "postgres://localhost/gatekey",
		},
		PKI: PKIConfig{
			KeyAlgorithm: "ecdsa256",
		},
		Auth: AuthConfig{
			OIDC: OIDCConfig{
				Enabled:   false,
				Providers: []OIDCProvider{},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error when OIDC disabled, got: %v", err)
	}
}

func TestConfig_Validate_SAMLDisabled(t *testing.T) {
	// SAML disabled should not require providers
	cfg := &Config{
		Database: DatabaseConfig{
			URL: "postgres://localhost/gatekey",
		},
		PKI: PKIConfig{
			KeyAlgorithm: "ecdsa256",
		},
		Auth: AuthConfig{
			SAML: SAMLConfig{
				Enabled:   false,
				Providers: []SAMLProvider{},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error when SAML disabled, got: %v", err)
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent config file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Create a temp file with invalid YAML
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(tmpFile, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	_, err = Load(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid YAML file")
	}
}

func TestLoad_ValidConfigFile(t *testing.T) {
	// Create a valid config file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `
database:
  url: postgres://localhost/gatekey
pki:
  key_algorithm: ecdsa256
`
	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Expected no error loading valid config, got: %v", err)
	}

	if cfg.Database.URL != "postgres://localhost/gatekey" {
		t.Errorf("Expected database URL 'postgres://localhost/gatekey', got '%s'", cfg.Database.URL)
	}

	if cfg.PKI.KeyAlgorithm != "ecdsa256" {
		t.Errorf("Expected key algorithm 'ecdsa256', got '%s'", cfg.PKI.KeyAlgorithm)
	}
}

func TestLoad_AppliesDefaults(t *testing.T) {
	// Create a minimal config file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "minimal.yaml")

	configContent := `
database:
  url: postgres://localhost/gatekey
pki:
  key_algorithm: ecdsa256
`
	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Expected no error loading config, got: %v", err)
	}

	// Verify defaults are applied
	if cfg.Server.Address != ":8080" {
		t.Errorf("Expected default server address ':8080', got '%s'", cfg.Server.Address)
	}

	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("Expected default max_open_conns 25, got %d", cfg.Database.MaxOpenConns)
	}

	if cfg.Logging.Level != "info" {
		t.Errorf("Expected default logging level 'info', got '%s'", cfg.Logging.Level)
	}

	if cfg.Logging.Format != "json" {
		t.Errorf("Expected default logging format 'json', got '%s'", cfg.Logging.Format)
	}

	if cfg.Auth.Session.CookieName != "gatekey_session" {
		t.Errorf("Expected default cookie name 'gatekey_session', got '%s'", cfg.Auth.Session.CookieName)
	}

	if !cfg.Metrics.Enabled {
		t.Error("Expected metrics enabled by default")
	}

	if !cfg.Audit.Enabled {
		t.Error("Expected audit enabled by default")
	}
}

func TestLoad_EmptyPath(t *testing.T) {
	// Loading with empty path should apply defaults but fail validation (no DB URL)
	_, err := Load("")
	if err == nil {
		t.Error("Expected error loading with empty path (missing DB URL)")
	}
}

func TestLoad_EnvironmentVariableExpansion(t *testing.T) {
	// Create a config file with env var references
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "envconfig.yaml")

	configContent := `
database:
  url: ${TEST_DB_URL}
pki:
  key_algorithm: ecdsa256
`
	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Set environment variable
	os.Setenv("TEST_DB_URL", "postgres://test:test@localhost/testdb")
	defer os.Unsetenv("TEST_DB_URL")

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Expected no error loading config with env var, got: %v", err)
	}

	if cfg.Database.URL != "postgres://test:test@localhost/testdb" {
		t.Errorf("Expected env var expanded DB URL, got '%s'", cfg.Database.URL)
	}
}

func TestLoad_ConfigValidationFails(t *testing.T) {
	// Create a config file that fails validation
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid_config.yaml")

	configContent := `
database:
  url: postgres://localhost/gatekey
pki:
  key_algorithm: invalid_algorithm
`
	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	_, err = Load(tmpFile)
	if err == nil {
		t.Error("Expected validation error for invalid key algorithm")
	}
}

func TestOIDCProvider_Fields(t *testing.T) {
	provider := OIDCProvider{
		Name:         "google",
		DisplayName:  "Google",
		Issuer:       "https://accounts.google.com",
		ClientID:     "client-id-12345",
		ClientSecret: "secret",
		RedirectURL:  "https://gatekey.example.com/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Claims: map[string]string{
			"email": "email",
			"name":  "name",
		},
	}

	if provider.Name != "google" {
		t.Errorf("Expected name 'google', got '%s'", provider.Name)
	}

	if len(provider.Scopes) != 3 {
		t.Errorf("Expected 3 scopes, got %d", len(provider.Scopes))
	}

	if provider.Claims["email"] != "email" {
		t.Errorf("Expected email claim mapping, got '%s'", provider.Claims["email"])
	}
}

func TestSAMLProvider_Fields(t *testing.T) {
	provider := SAMLProvider{
		Name:           "okta",
		DisplayName:    "Okta SSO",
		IDPMetadataURL: "https://example.okta.com/metadata.xml",
		EntityID:       "https://gatekey.example.com",
		ACSURL:         "https://gatekey.example.com/saml/acs",
		CertFile:       "/path/to/cert.pem",
		KeyFile:        "/path/to/key.pem",
		Attributes: map[string]string{
			"email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
		},
	}

	if provider.Name != "okta" {
		t.Errorf("Expected name 'okta', got '%s'", provider.Name)
	}

	if provider.CertFile != "/path/to/cert.pem" {
		t.Errorf("Expected cert file path, got '%s'", provider.CertFile)
	}
}

func TestDatabaseConfig_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "db_defaults.yaml")

	configContent := `
database:
  url: postgres://localhost/gatekey
pki:
  key_algorithm: ecdsa256
`
	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("Expected max_open_conns default 25, got %d", cfg.Database.MaxOpenConns)
	}

	if cfg.Database.MaxIdleConns != 5 {
		t.Errorf("Expected max_idle_conns default 5, got %d", cfg.Database.MaxIdleConns)
	}
}

func TestPolicyConfig_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "policy_defaults.yaml")

	configContent := `
database:
  url: postgres://localhost/gatekey
pki:
  key_algorithm: ecdsa256
`
	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Policy.DefaultPolicy != "deny-all" {
		t.Errorf("Expected default policy 'deny-all', got '%s'", cfg.Policy.DefaultPolicy)
	}

	if cfg.Policy.EvaluationMode != "strict" {
		t.Errorf("Expected evaluation mode 'strict', got '%s'", cfg.Policy.EvaluationMode)
	}
}

func TestServerConfig_TLS(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tls_config.yaml")

	configContent := `
database:
  url: postgres://localhost/gatekey
pki:
  key_algorithm: ecdsa256
server:
  tls_enabled: true
  tls_cert: /path/to/cert.pem
  tls_key: /path/to/key.pem
  trusted_proxies:
    - 10.0.0.0/8
    - 192.168.0.0/16
`
	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if !cfg.Server.TLSEnabled {
		t.Error("Expected TLS enabled")
	}

	if cfg.Server.TLSCert != "/path/to/cert.pem" {
		t.Errorf("Expected TLS cert path, got '%s'", cfg.Server.TLSCert)
	}

	if len(cfg.Server.TrustedProxies) != 2 {
		t.Errorf("Expected 2 trusted proxies, got %d", len(cfg.Server.TrustedProxies))
	}
}

func TestConfig_Validate_SSLModes(t *testing.T) {
	tests := []struct {
		name        string
		sslMode     string
		sslRootCert string
		wantErr     bool
	}{
		{"disable is valid", "disable", "", false},
		{"require is valid", "require", "", false},
		{"verify-ca is valid with root cert", "verify-ca", "/path/to/ca.crt", false},
		{"verify-ca without root cert fails", "verify-ca", "", true},
		{"verify-full is valid with root cert", "verify-full", "/path/to/ca.crt", false},
		{"verify-full without root cert fails", "verify-full", "", true},
		{"invalid ssl mode", "invalid-mode", "", true},
		{"empty ssl mode is valid (defaults)", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Database: DatabaseConfig{
					URL:         "postgres://localhost/gatekey",
					SSLMode:     tt.sslMode,
					SSLRootCert: tt.sslRootCert,
				},
				PKI: PKIConfig{
					KeyAlgorithm: "ecdsa256",
				},
			}

			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Expected error for SSL mode %s, got nil", tt.sslMode)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error for SSL mode %s, got: %v", tt.sslMode, err)
			}
		})
	}
}

func TestDatabaseConfig_SSLFields(t *testing.T) {
	cfg := DatabaseConfig{
		URL:         "postgres://localhost/gatekey",
		SSLMode:     "verify-full",
		SSLRootCert: "/path/to/ca.crt",
		SSLCert:     "/path/to/client.crt",
		SSLKey:      "/path/to/client.key",
	}

	if cfg.SSLMode != "verify-full" {
		t.Errorf("Expected SSLMode 'verify-full', got '%s'", cfg.SSLMode)
	}
	if cfg.SSLRootCert != "/path/to/ca.crt" {
		t.Errorf("Expected SSLRootCert '/path/to/ca.crt', got '%s'", cfg.SSLRootCert)
	}
	if cfg.SSLCert != "/path/to/client.crt" {
		t.Errorf("Expected SSLCert '/path/to/client.crt', got '%s'", cfg.SSLCert)
	}
	if cfg.SSLKey != "/path/to/client.key" {
		t.Errorf("Expected SSLKey '/path/to/client.key', got '%s'", cfg.SSLKey)
	}
}

func TestLoad_DatabaseSSLDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "ssl_defaults.yaml")

	configContent := `
database:
  url: postgres://localhost/gatekey
pki:
  key_algorithm: ecdsa256
`
	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Default ssl_mode should be 'require'
	if cfg.Database.SSLMode != "require" {
		t.Errorf("Expected default ssl_mode 'require', got '%s'", cfg.Database.SSLMode)
	}
}

func TestLoad_SecurityDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "security_defaults.yaml")

	configContent := `
database:
  url: postgres://localhost/gatekey
pki:
  key_algorithm: ecdsa256
`
	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// SSRF protection should be enabled by default
	if !cfg.Security.ProxySSRFProtection {
		t.Error("Expected ProxySSRFProtection to be true by default")
	}

	// DNS rebinding protection should be enabled by default
	if !cfg.Security.ProxyDNSRebindingProtection {
		t.Error("Expected ProxyDNSRebindingProtection to be true by default")
	}

	// TLS verification should be enabled by default
	if !cfg.Security.ProxyTLSVerify {
		t.Error("Expected ProxyTLSVerify to be true by default")
	}

	// Security headers should be enabled by default
	if !cfg.Security.SecurityHeadersEnabled {
		t.Error("Expected SecurityHeadersEnabled to be true by default")
	}

	// HSTS should be enabled by default
	if !cfg.Security.HSTSEnabled {
		t.Error("Expected HSTSEnabled to be true by default")
	}

	// HSTS max-age should be 1 year by default
	if cfg.Security.HSTSMaxAge != 31536000 {
		t.Errorf("Expected HSTSMaxAge to be 31536000, got %d", cfg.Security.HSTSMaxAge)
	}

	// Frame ancestors should be dynamic by default (for multi-tenant)
	if cfg.Security.FrameAncestors != "dynamic" {
		t.Errorf("Expected FrameAncestors to be 'dynamic', got '%s'", cfg.Security.FrameAncestors)
	}

	// TLS min version should be 1.2 by default
	if cfg.Security.ProxyTLSMinVersion != "1.2" {
		t.Errorf("Expected ProxyTLSMinVersion to be '1.2', got '%s'", cfg.Security.ProxyTLSMinVersion)
	}
}

func TestLoad_SecurityCanBeDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "security_disabled.yaml")

	configContent := `
database:
  url: postgres://localhost/gatekey
pki:
  key_algorithm: ecdsa256
security:
  proxy_ssrf_protection: false
  security_headers_enabled: false
  hsts_enabled: false
`
	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify security can be disabled via config
	if cfg.Security.ProxySSRFProtection {
		t.Error("Expected ProxySSRFProtection to be false when explicitly disabled")
	}

	if cfg.Security.SecurityHeadersEnabled {
		t.Error("Expected SecurityHeadersEnabled to be false when explicitly disabled")
	}

	if cfg.Security.HSTSEnabled {
		t.Error("Expected HSTSEnabled to be false when explicitly disabled")
	}
}
