package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "auto", cfg.ColorScheme)
	assert.Equal(t, "us-east-1", cfg.DefaultRegion)
	assert.Equal(t, time.Hour, cfg.DefaultSessionDuration)
	assert.Equal(t, 5*time.Minute, cfg.RefreshBeforeExpiry)
	assert.False(t, cfg.ClearOnExit)

	// AWS config
	assert.True(t, cfg.AWS.UseCredentialProcess)

	// UI config
	assert.True(t, cfg.UI.ShowTimestamps)
	assert.True(t, cfg.UI.ShowRegion)
	assert.False(t, cfg.UI.CompactMode)
	assert.Equal(t, "default", cfg.UI.Theme)

	// Storage config
	assert.Equal(t, "accio", cfg.Storage.KeyringService)
	assert.Contains(t, cfg.Storage.SessionFile, ".accio")
	assert.Contains(t, cfg.Storage.IntegrationFile, ".accio")
}

func TestNewManager(t *testing.T) {
	m := NewManager()

	assert.NotNil(t, m)
	assert.NotNil(t, m.config)
	assert.NotNil(t, m.k)
}

func TestManager_Get(t *testing.T) {
	m := NewManager()
	cfg := m.Get()

	assert.NotNil(t, cfg)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestManager_Set(t *testing.T) {
	m := NewManager()
	newCfg := &Config{
		LogLevel: "debug",
	}

	m.Set(newCfg)

	assert.Equal(t, "debug", m.Get().LogLevel)
}

func TestManager_LoadWithEnvOverride(t *testing.T) {
	// Create a temp directory for the test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Set environment variable to point to our test config
	os.Setenv("ACCIO_CONFIG", configPath)
	defer os.Unsetenv("ACCIO_CONFIG")

	m := NewManager()
	err := m.Load()

	// Should not error even if file doesn't exist
	assert.NoError(t, err)
}

func TestManager_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "accio", "config.yaml")

	os.Setenv("ACCIO_CONFIG", configPath)
	defer os.Unsetenv("ACCIO_CONFIG")

	// Create and save config
	m := NewManager()
	m.config.DefaultRegion = "eu-west-1"
	m.config.LogLevel = "debug"

	err := m.Save()
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Load config in new manager
	m2 := NewManager()
	err = m2.Load()
	require.NoError(t, err)

	// Check values were persisted
	assert.Equal(t, "eu-west-1", m2.Get().DefaultRegion)
	assert.Equal(t, "debug", m2.Get().LogLevel)
}

func TestManager_GetConfigDir(t *testing.T) {
	m := NewManager()
	dir := m.GetConfigDir()

	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, ".accio")
}

func TestManager_getConfigPath(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		contains string
	}{
		{
			name:     "default path",
			contains: ".accio",
		},
		{
			name:     "custom path via env",
			envVar:   "ACCIO_CONFIG",
			envValue: "/custom/path/config.yaml",
			contains: "/custom/path/config.yaml",
		},
		{
			name:     "XDG config home",
			envVar:   "XDG_CONFIG_HOME",
			envValue: "/home/user/.config",
			contains: "accio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment
			os.Unsetenv("ACCIO_CONFIG")
			os.Unsetenv("XDG_CONFIG_HOME")

			if tt.envVar != "" {
				os.Setenv(tt.envVar, tt.envValue)
				defer os.Unsetenv(tt.envVar)
			}

			m := NewManager()
			path := m.getConfigPath()

			assert.Contains(t, path, tt.contains)
		})
	}
}
