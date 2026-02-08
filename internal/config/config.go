// Package config provides application configuration management.
package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config holds the application configuration.
type Config struct {
	// General settings
	LogLevel    string `koanf:"log_level"`
	ColorScheme string `koanf:"color_scheme"`

	// Session defaults
	DefaultRegion          string        `koanf:"default_region"`
	DefaultSessionDuration time.Duration `koanf:"default_session_duration"`
	RefreshBeforeExpiry    time.Duration `koanf:"refresh_before_expiry"`
	ClearOnExit            bool          `koanf:"clear_on_exit"`

	// AWS configuration
	AWS AWSConfig `koanf:"aws"`

	// UI settings
	UI UIConfig `koanf:"ui"`

	// Storage settings
	Storage StorageConfig `koanf:"storage"`
}

// AWSConfig holds AWS-specific configuration.
type AWSConfig struct {
	// Path to AWS config file (default: ~/.aws/config)
	ConfigFile string `koanf:"config_file"`
	// Path to AWS credentials file (default: ~/.aws/credentials)
	CredentialsFile string `koanf:"credentials_file"`
	// Whether to use credential_process instead of writing credentials
	UseCredentialProcess bool `koanf:"use_credential_process"`
}

// UIConfig holds UI-specific configuration.
type UIConfig struct {
	// Whether to show timestamps in session list
	ShowTimestamps bool `koanf:"show_timestamps"`
	// Whether to show region in session list
	ShowRegion bool `koanf:"show_region"`
	// Compact mode (less spacing)
	CompactMode bool `koanf:"compact_mode"`
	// Theme name
	Theme string `koanf:"theme"`
}

// StorageConfig holds storage-specific configuration.
type StorageConfig struct {
	// Path to session data file
	SessionFile string `koanf:"session_file"`
	// Path to integration data file
	IntegrationFile string `koanf:"integration_file"`
	// Keyring service name
	KeyringService string `koanf:"keyring_service"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".accio")

	return &Config{
		LogLevel:               "info",
		ColorScheme:            "auto",
		DefaultRegion:          "us-east-1",
		DefaultSessionDuration: time.Hour,
		RefreshBeforeExpiry:    5 * time.Minute,
		ClearOnExit:            false,
		AWS: AWSConfig{
			ConfigFile:           filepath.Join(homeDir, ".aws", "config"),
			CredentialsFile:      filepath.Join(homeDir, ".aws", "credentials"),
			UseCredentialProcess: true,
		},
		UI: UIConfig{
			ShowTimestamps: true,
			ShowRegion:     true,
			CompactMode:    false,
			Theme:          "default",
		},
		Storage: StorageConfig{
			SessionFile:     filepath.Join(configDir, "sessions.yaml"),
			IntegrationFile: filepath.Join(configDir, "integrations.yaml"),
			KeyringService:  "accio",
		},
	}
}

// Manager handles configuration loading and saving.
type Manager struct {
	k          *koanf.Koanf
	configPath string
	config     *Config
}

// NewManager creates a new configuration manager.
func NewManager() *Manager {
	return &Manager{
		k:      koanf.New("."),
		config: DefaultConfig(),
	}
}

// Load loads configuration from file and environment.
func (m *Manager) Load() error {
	// Determine config path
	configPath := m.getConfigPath()
	m.configPath = configPath

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	// Load from file if it exists
	if _, err := os.Stat(configPath); err == nil {
		if err := m.k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
			return err
		}
	}

	// Load from environment variables (ACCIO_ prefix)
	if err := m.k.Load(env.Provider("ACCIO_", ".", func(s string) string {
		return s
	}), nil); err != nil {
		return err
	}

	// Unmarshal into config struct
	if err := m.k.Unmarshal("", m.config); err != nil {
		return err
	}

	return nil
}

// Save saves the current configuration to file.
func (m *Manager) Save() error {
	if m.configPath == "" {
		m.configPath = m.getConfigPath()
	}

	// Ensure config directory exists
	configDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	// Marshal config to YAML
	data, err := yaml.Parser().Marshal(m.configToMap())
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(m.configPath, data, 0600)
}

// Get returns the current configuration.
func (m *Manager) Get() *Config {
	return m.config
}

// Set updates the configuration.
func (m *Manager) Set(cfg *Config) {
	m.config = cfg
}

// GetConfigDir returns the configuration directory.
func (m *Manager) GetConfigDir() string {
	return filepath.Dir(m.getConfigPath())
}

// getConfigPath returns the path to the config file.
func (m *Manager) getConfigPath() string {
	// Check for environment variable override
	if path := os.Getenv("ACCIO_CONFIG"); path != "" {
		return path
	}

	// Check for XDG config directory
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "accio", "config.yaml")
	}

	// Default to ~/.accio/config.yaml
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".accio", "config.yaml")
}

// configToMap converts config to a map for YAML marshaling.
func (m *Manager) configToMap() map[string]interface{} {
	return map[string]interface{}{
		"log_level":                m.config.LogLevel,
		"color_scheme":             m.config.ColorScheme,
		"default_region":           m.config.DefaultRegion,
		"default_session_duration": m.config.DefaultSessionDuration.String(),
		"refresh_before_expiry":    m.config.RefreshBeforeExpiry.String(),
		"clear_on_exit":            m.config.ClearOnExit,
		"aws": map[string]interface{}{
			"config_file":            m.config.AWS.ConfigFile,
			"credentials_file":       m.config.AWS.CredentialsFile,
			"use_credential_process": m.config.AWS.UseCredentialProcess,
		},
		"ui": map[string]interface{}{
			"show_timestamps": m.config.UI.ShowTimestamps,
			"show_region":     m.config.UI.ShowRegion,
			"compact_mode":    m.config.UI.CompactMode,
			"theme":           m.config.UI.Theme,
		},
		"storage": map[string]interface{}{
			"session_file":     m.config.Storage.SessionFile,
			"integration_file": m.config.Storage.IntegrationFile,
			"keyring_service":  m.config.Storage.KeyringService,
		},
	}
}
