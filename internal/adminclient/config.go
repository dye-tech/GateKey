// Package adminclient provides the GateKey admin client functionality.
package adminclient

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the admin client configuration.
type Config struct {
	ServerURL string `yaml:"server_url"`
	APIKey    string `yaml:"api_key,omitempty"`

	// Runtime paths (not saved to config)
	configPath string `yaml:"-"`
	dataDir    string `yaml:"-"`
}

// DefaultConfig returns a configuration with default values.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".gatekey-admin")

	return &Config{
		dataDir: dataDir,
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
		configPath = filepath.Join(homeDir, ".gatekey-admin", "config.yaml")
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
		configPath = filepath.Join(homeDir, ".gatekey-admin", "config.yaml")
	}

	cfg.configPath = configPath
	cfg.dataDir = filepath.Dir(configPath)

	// Create directory
	if err := os.MkdirAll(cfg.dataDir, 0700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
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
	case "api_key":
		c.APIKey = value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	if err := c.Save(); err != nil {
		return err
	}

	if key == "api_key" {
		fmt.Printf("Set %s = %s...%s\n", key, value[:12], value[len(value)-4:])
	} else {
		fmt.Printf("Set %s = %s\n", key, value)
	}
	return nil
}

// Print displays the current configuration.
func (c *Config) Print() error {
	fmt.Println("GateKey Admin Client Configuration")
	fmt.Println("===================================")
	fmt.Printf("Config file: %s\n", c.configPath)
	fmt.Printf("Server URL:  %s\n", c.ServerURL)
	if c.APIKey != "" {
		fmt.Printf("API Key:     %s...%s\n", c.APIKey[:12], c.APIKey[len(c.APIKey)-4:])
	} else {
		fmt.Println("API Key:     (not set)")
	}
	return nil
}

// DataDir returns the data directory path.
func (c *Config) DataDir() string {
	return c.dataDir
}

// TokenPath returns the path to the session token file.
func (c *Config) TokenPath() string {
	return filepath.Join(c.dataDir, "token")
}

// ConfigPath returns the path to the config file.
func (c *Config) ConfigPath() string {
	return c.configPath
}

// HasAPIKey returns true if an API key is configured.
func (c *Config) HasAPIKey() bool {
	return c.APIKey != ""
}

// ClearAPIKey removes the API key from config.
func (c *Config) ClearAPIKey() error {
	c.APIKey = ""
	return c.Save()
}
