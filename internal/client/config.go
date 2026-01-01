// Package client provides the GateKey VPN client functionality.
package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the client configuration.
type Config struct {
	ServerURL     string `yaml:"server_url"`
	OpenVPNBinary string `yaml:"openvpn_binary"`
	ConfigDir     string `yaml:"config_dir"`
	LogLevel      string `yaml:"log_level"`
	APIKey        string `yaml:"api_key,omitempty"`

	// Runtime paths (not saved to config)
	configPath string `yaml:"-"`
	dataDir    string `yaml:"-"`
}

// DefaultConfig returns a configuration with default values.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".gatekey")

	return &Config{
		OpenVPNBinary: "openvpn",
		ConfigDir:     filepath.Join(dataDir, "configs"),
		LogLevel:      "info",
		dataDir:       dataDir,
	}
}

// LoadConfig loads configuration from file or creates default.
func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// Determine config path
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".gatekey", "config.yaml")
	}

	cfg.configPath = configPath
	cfg.dataDir = filepath.Dir(configPath)

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(cfg.dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Load config if it exists
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config doesn't exist, use defaults
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Ensure ConfigDir is absolute
	if cfg.ConfigDir != "" && !filepath.IsAbs(cfg.ConfigDir) {
		cfg.ConfigDir = filepath.Join(cfg.dataDir, cfg.ConfigDir)
	}

	return cfg, nil
}

// InitConfig initializes a new configuration file.
func InitConfig(configPath string, serverURL string) error {
	cfg := DefaultConfig()
	cfg.ServerURL = serverURL

	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".gatekey", "config.yaml")
	}

	cfg.configPath = configPath
	cfg.dataDir = filepath.Dir(configPath)

	// Create directories
	if err := os.MkdirAll(cfg.dataDir, 0700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	if err := os.MkdirAll(cfg.ConfigDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Printf("Configuration initialized at %s\n", configPath)
	fmt.Printf("Server URL: %s\n", serverURL)
	return nil
}

// Save writes the configuration to disk.
func (c *Config) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(c.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Set sets a configuration value by key.
func (c *Config) Set(key, value string) error {
	key = strings.ToLower(key)

	switch key {
	case "server_url", "server":
		c.ServerURL = value
	case "openvpn_binary", "openvpn":
		c.OpenVPNBinary = value
	case "config_dir":
		c.ConfigDir = value
	case "log_level":
		c.LogLevel = value
	case "api_key":
		c.APIKey = value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	if err := c.Save(); err != nil {
		return err
	}

	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}

// Print displays the current configuration.
func (c *Config) Print() error {
	fmt.Println("GateKey Client Configuration")
	fmt.Println("==========================")
	fmt.Printf("Config file:    %s\n", c.configPath)
	fmt.Printf("Server URL:     %s\n", c.ServerURL)
	fmt.Printf("OpenVPN binary: %s\n", c.OpenVPNBinary)
	fmt.Printf("Config dir:     %s\n", c.ConfigDir)
	fmt.Printf("Log level:      %s\n", c.LogLevel)
	return nil
}

// DataDir returns the data directory path.
func (c *Config) DataDir() string {
	return c.dataDir
}

// TokenPath returns the path to the token file.
func (c *Config) TokenPath() string {
	return filepath.Join(c.dataDir, "token")
}

// PidPath returns the path to the PID file.
func (c *Config) PidPath() string {
	return filepath.Join(c.dataDir, "openvpn.pid")
}

// LogPath returns the path to the OpenVPN log file.
func (c *Config) LogPath() string {
	return filepath.Join(c.dataDir, "openvpn.log")
}

// CurrentConfigPath returns the path to the currently active config.
func (c *Config) CurrentConfigPath() string {
	return filepath.Join(c.dataDir, "current.ovpn")
}

// StateFilePath returns the path to the state file.
func (c *Config) StateFilePath() string {
	return filepath.Join(c.dataDir, "state.json")
}

// GatewayPidPath returns the path to the PID file for a specific gateway.
func (c *Config) GatewayPidPath(gatewayName string) string {
	return filepath.Join(c.dataDir, fmt.Sprintf("openvpn-%s.pid", gatewayName))
}

// GatewayLogPath returns the path to the log file for a specific gateway.
func (c *Config) GatewayLogPath(gatewayName string) string {
	return filepath.Join(c.dataDir, fmt.Sprintf("openvpn-%s.log", gatewayName))
}

// GatewayConfigPath returns the path to the OpenVPN config for a specific gateway.
func (c *Config) GatewayConfigPath(gatewayName string) string {
	return filepath.Join(c.dataDir, fmt.Sprintf("%s.ovpn", gatewayName))
}
