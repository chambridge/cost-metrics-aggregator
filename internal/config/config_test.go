package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Reset viper to ensure a clean state for each test
	viper.Reset()

	// Helper function to clear environment variables
	clearEnv := func() {
		os.Unsetenv("SERVER_ADDRESS")
		os.Unsetenv("DATABASE_URL")
	}

	t.Run("DefaultValues", func(t *testing.T) {
		// Arrange
		clearEnv()

		// Act
		cfg, err := LoadConfig()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, ":8080", cfg.ServerAddress, "ServerAddress should be default value")
		assert.Equal(t, "postgres://user:pass@localhost:5432/dbname", cfg.DatabaseURL, "DatabaseURL should be default value")
	})

	t.Run("EnvironmentVariableOverride", func(t *testing.T) {
		// Arrange
		clearEnv()
		err := os.Setenv("SERVER_ADDRESS", ":9090")
		require.NoError(t, err)
		err = os.Setenv("DATABASE_URL", "postgres://test:test@db:5432/testdb")
		require.NoError(t, err)

		// Act
		cfg, err := LoadConfig()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, ":9090", cfg.ServerAddress, "ServerAddress should be overridden by environment variable")
		assert.Equal(t, "postgres://test:test@db:5432/testdb", cfg.DatabaseURL, "DatabaseURL should be overridden by environment variable")
	})

	t.Run("InvalidConfigFormat", func(t *testing.T) {
		// Arrange
		clearEnv()
		// Simulate invalid viper configuration by setting a malformed struct tag
		viper.Set("server_address", map[string]interface{}{"invalid": "data"}) // Cause unmarshal failure

		// Act
		cfg, err := LoadConfig()

		// Assert
		assert.Error(t, err, "LoadConfig should return an error for invalid configuration")
		assert.Nil(t, cfg, "Config should be nil on error")
	})
}
