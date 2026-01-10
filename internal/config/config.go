// Package config handles configuration loading and validation for GateKey.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the GateKey server.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	PKI      PKIConfig      `mapstructure:"pki"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Gateway  GatewayConfig  `mapstructure:"gateway"`
	Policy   PolicyConfig   `mapstructure:"policy"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Audit    AuditConfig    `mapstructure:"audit"`
	Security SecurityConfig `mapstructure:"security"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Address        string   `mapstructure:"address"`
	TLSAddress     string   `mapstructure:"tls_address"`
	TLSEnabled     bool     `mapstructure:"tls_enabled"`
	TLSCert        string   `mapstructure:"tls_cert"`
	TLSKey         string   `mapstructure:"tls_key"`
	TrustedProxies []string `mapstructure:"trusted_proxies"`
	CORSOrigins    []string `mapstructure:"cors_origins"`
}

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	// TLS configuration
	SSLMode     string `mapstructure:"ssl_mode"`      // disable, require, verify-ca, verify-full
	SSLRootCert string `mapstructure:"ssl_root_cert"` // Path to CA certificate for server verification
	SSLCert     string `mapstructure:"ssl_cert"`      // Path to client certificate (for mTLS)
	SSLKey      string `mapstructure:"ssl_key"`       // Path to client private key (for mTLS)
}

// PKIConfig holds PKI/CA configuration.
type PKIConfig struct {
	CACert       string        `mapstructure:"ca_cert"`
	CAKey        string        `mapstructure:"ca_key"`
	CertValidity time.Duration `mapstructure:"cert_validity"`
	CAValidity   time.Duration `mapstructure:"ca_validity"`
	KeyAlgorithm string        `mapstructure:"key_algorithm"`
	Organization string        `mapstructure:"organization"`
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	Session SessionConfig `mapstructure:"session"`
	OIDC    OIDCConfig    `mapstructure:"oidc"`
	SAML    SAMLConfig    `mapstructure:"saml"`
}

// SessionConfig holds session management configuration.
type SessionConfig struct {
	Validity   time.Duration `mapstructure:"validity"`
	CookieName string        `mapstructure:"cookie_name"`
	Secure     bool          `mapstructure:"secure"`
	HTTPOnly   bool          `mapstructure:"http_only"`
	SameSite   string        `mapstructure:"same_site"`
}

// OIDCConfig holds OIDC provider configuration.
type OIDCConfig struct {
	Enabled   bool           `mapstructure:"enabled"`
	Providers []OIDCProvider `mapstructure:"providers"`
}

// OIDCProvider holds configuration for a single OIDC provider.
type OIDCProvider struct {
	Name         string            `mapstructure:"name"`
	DisplayName  string            `mapstructure:"display_name"`
	Issuer       string            `mapstructure:"issuer"`
	ClientID     string            `mapstructure:"client_id"`
	ClientSecret string            `mapstructure:"client_secret"`
	RedirectURL  string            `mapstructure:"redirect_url"`
	Scopes       []string          `mapstructure:"scopes"`
	Claims       map[string]string `mapstructure:"claims"`
}

// SAMLConfig holds SAML provider configuration.
type SAMLConfig struct {
	Enabled   bool           `mapstructure:"enabled"`
	Providers []SAMLProvider `mapstructure:"providers"`
}

// SAMLProvider holds configuration for a single SAML provider.
type SAMLProvider struct {
	Name           string            `mapstructure:"name"`
	DisplayName    string            `mapstructure:"display_name"`
	IDPMetadataURL string            `mapstructure:"idp_metadata_url"`
	EntityID       string            `mapstructure:"entity_id"`
	ACSURL         string            `mapstructure:"acs_url"`
	CertFile       string            `mapstructure:"cert_file"`
	KeyFile        string            `mapstructure:"key_file"`
	Attributes     map[string]string `mapstructure:"attributes"`
}

// GatewayConfig holds gateway communication configuration.
type GatewayConfig struct {
	APIURL            string        `mapstructure:"api_url"`
	Token             string        `mapstructure:"token"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
}

// PolicyConfig holds policy engine configuration.
type PolicyConfig struct {
	DefaultPolicy  string `mapstructure:"default_policy"`
	EvaluationMode string `mapstructure:"evaluation_mode"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// MetricsConfig holds metrics/monitoring configuration.
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
	Port    int    `mapstructure:"port"`
}

// AuditConfig holds audit logging configuration.
type AuditConfig struct {
	Enabled     bool     `mapstructure:"enabled"`
	Destination string   `mapstructure:"destination"`
	FilePath    string   `mapstructure:"file_path"`
	Events      []string `mapstructure:"events"`
}

// SecurityConfig holds security-related configuration.
type SecurityConfig struct {
	// ProxySSRFProtection blocks proxy requests to private/internal IP addresses.
	// Set to true for strict SSRF protection, but this may block legitimate
	// internal proxying for VPN use cases. Default is false for backward compatibility.
	ProxySSRFProtection bool `mapstructure:"proxy_ssrf_protection"`
	// ProxyDNSRebindingProtection validates IPs at connection time to prevent
	// DNS rebinding attacks. This is always enabled by default and should remain so.
	ProxyDNSRebindingProtection bool `mapstructure:"proxy_dns_rebinding_protection"`
	// ProxyTLSVerify enables TLS certificate verification for proxy targets.
	// When enabled, proxy requests will verify server certificates against the
	// system CA bundle or the custom CA bundle specified in ProxyCABundle.
	ProxyTLSVerify bool `mapstructure:"proxy_tls_verify"`
	// ProxyCABundle is the path to a custom CA bundle file (PEM format) for
	// verifying proxy target certificates. If empty, the system CA bundle is used.
	ProxyCABundle string `mapstructure:"proxy_ca_bundle"`
	// ProxyTLSMinVersion is the minimum TLS version for proxy connections.
	// Valid values: "1.2", "1.3". Default is "1.2".
	ProxyTLSMinVersion string `mapstructure:"proxy_tls_min_version"`
	// CAKeyEncryptionKey is the 32-byte (256-bit) key used to encrypt CA private keys
	// at rest in the database. Should be set via environment variable for security.
	// If empty, CA keys are stored unencrypted (for backward compatibility).
	// Generate with: openssl rand -base64 32
	CAKeyEncryptionKey string `mapstructure:"ca_key_encryption_key"`
}

// Load reads configuration from the specified file and environment variables.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read config file if specified
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Read from environment variables
	v.SetEnvPrefix("GATEX")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Expand environment variables in string values
	for _, key := range v.AllKeys() {
		val := v.GetString(key)
		if strings.HasPrefix(val, "${") && strings.HasSuffix(val, "}") {
			envVar := strings.TrimSuffix(strings.TrimPrefix(val, "${"), "}")
			v.Set(key, os.Getenv(envVar))
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.address", ":8080")
	v.SetDefault("server.tls_address", ":8443")
	v.SetDefault("server.tls_enabled", false)

	// Database defaults
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "5m")
	v.SetDefault("database.ssl_mode", "require") // Default to requiring TLS

	// PKI defaults
	v.SetDefault("pki.cert_validity", "24h")
	v.SetDefault("pki.ca_validity", "87600h") // 10 years
	v.SetDefault("pki.key_algorithm", "ecdsa256")
	v.SetDefault("pki.organization", "GateKey")

	// Session defaults
	v.SetDefault("auth.session.validity", "12h")
	v.SetDefault("auth.session.cookie_name", "gatekey_session")
	v.SetDefault("auth.session.secure", true)
	v.SetDefault("auth.session.http_only", true)
	v.SetDefault("auth.session.same_site", "lax")

	// Gateway defaults
	v.SetDefault("gateway.heartbeat_interval", "30s")

	// Policy defaults
	v.SetDefault("policy.default_policy", "deny-all")
	v.SetDefault("policy.evaluation_mode", "strict")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")

	// Metrics defaults
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
	v.SetDefault("metrics.port", 0)

	// Audit defaults
	v.SetDefault("audit.enabled", true)
	v.SetDefault("audit.destination", "database")

	// Security defaults
	v.SetDefault("security.proxy_ssrf_protection", false)            // Off by default for backward compat with VPN use cases
	v.SetDefault("security.proxy_dns_rebinding_protection", true)    // Always on by default for safety
	v.SetDefault("security.proxy_tls_verify", false)                 // Off by default for backward compat (internal self-signed certs)
	v.SetDefault("security.proxy_ca_bundle", "")                     // Empty means use system CA bundle
	v.SetDefault("security.proxy_tls_min_version", "1.2")            // TLS 1.2 minimum
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("database.url is required")
	}

	// Validate database SSL mode
	validSSLModes := map[string]bool{
		"disable":     true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}
	if c.Database.SSLMode != "" && !validSSLModes[c.Database.SSLMode] {
		return fmt.Errorf("invalid database.ssl_mode: %s (must be disable, require, verify-ca, or verify-full)", c.Database.SSLMode)
	}

	// Validate SSL certificate paths if verify modes are used
	if c.Database.SSLMode == "verify-ca" || c.Database.SSLMode == "verify-full" {
		if c.Database.SSLRootCert == "" {
			return fmt.Errorf("database.ssl_root_cert is required when ssl_mode is %s", c.Database.SSLMode)
		}
	}

	if c.Auth.OIDC.Enabled && len(c.Auth.OIDC.Providers) == 0 {
		return fmt.Errorf("at least one OIDC provider must be configured when OIDC is enabled")
	}

	if c.Auth.SAML.Enabled && len(c.Auth.SAML.Providers) == 0 {
		return fmt.Errorf("at least one SAML provider must be configured when SAML is enabled")
	}

	validKeyAlgorithms := map[string]bool{
		"rsa2048":  true,
		"rsa4096":  true,
		"ecdsa256": true,
		"ecdsa384": true,
	}
	if !validKeyAlgorithms[c.PKI.KeyAlgorithm] {
		return fmt.Errorf("invalid key algorithm: %s", c.PKI.KeyAlgorithm)
	}

	return nil
}
