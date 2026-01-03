package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Token  string `mapstructure:"token"`
	HomeID string `mapstructure:"home_id"`
	Format string `mapstructure:"format"`
}

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".tibber", "config.yaml")
}

// Load reads configuration from environment and config file
// Priority: env vars > config file > defaults
func Load(configPath string) (*Config, error) {
	cfg := &Config{
		Format: "pretty", // default: beautiful CLI output
	}

	// Check environment variable first (highest priority)
	if token := os.Getenv("TIBBER_TOKEN"); token != "" {
		cfg.Token = token
	}

	if homeID := os.Getenv("TIBBER_HOME_ID"); homeID != "" {
		cfg.HomeID = homeID
	}

	// Try to load config file
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	if configPath != "" {
		viper.SetConfigFile(configPath)
		viper.SetConfigType("yaml")

		if err := viper.ReadInConfig(); err == nil {
			// Only override if not set by env var
			if cfg.Token == "" {
				cfg.Token = viper.GetString("token")
			}
			if cfg.HomeID == "" {
				cfg.HomeID = viper.GetString("home_id")
			}
			if format := viper.GetString("format"); format != "" {
				cfg.Format = format
			}
		}
		// Ignore file not found - config file is optional
	}

	return cfg, nil
}

// Validate checks if required configuration is present
func (c *Config) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("no API token found. Set TIBBER_TOKEN environment variable or create config at %s", DefaultConfigPath())
	}
	return nil
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configDir := filepath.Join(home, ".tibber")
	return os.MkdirAll(configDir, 0700)
}
