package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Save original env vars
	originalVIVAEnv := os.Getenv("VIVA_ENV")
	originalGOEnv := os.Getenv("GO_ENV")
	
	defer func() {
		os.Setenv("VIVA_ENV", originalVIVAEnv)
		os.Setenv("GO_ENV", originalGOEnv)
		viper.Reset()
	}()

	t.Run("loads dev config by default", func(t *testing.T) {
		viper.Reset()
		os.Setenv("VIVA_ENV", "")
		os.Setenv("GO_ENV", "")
		
		cfg, err := Load()
		
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "development", cfg.App.Environment)
		assert.Equal(t, "viva-rate-limiter", cfg.App.Name)
		assert.Equal(t, 8080, cfg.Server.Port)
		// Database host and port depend on config file contents
		assert.NotEmpty(t, cfg.Database.Host)
		assert.Greater(t, cfg.Database.Port, 0)
	})

	t.Run("uses VIVA_ENV environment variable", func(t *testing.T) {
		viper.Reset()
		os.Setenv("VIVA_ENV", "dev")
		
		cfg, err := Load()
		
		require.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("overrides config with environment variables", func(t *testing.T) {
		viper.Reset()
		os.Setenv("VIVA_ENV", "dev")
		os.Setenv("VIVA_SERVER_PORT", "9090")
		os.Setenv("VIVA_DATABASE_HOST", "custom-host")
		
		cfg, err := Load()
		
		require.NoError(t, err)
		assert.Equal(t, 9090, cfg.Server.Port)
		assert.Equal(t, "custom-host", cfg.Database.Host)
		
		// Cleanup
		os.Unsetenv("VIVA_SERVER_PORT")
		os.Unsetenv("VIVA_DATABASE_HOST")
	})
}

func TestDatabaseConfig_GetDSN(t *testing.T) {
	dbConfig := &DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		Name:     "testdb",
		SSLMode:  "disable",
		Timezone: "UTC",
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable TimeZone=UTC"
	actual := dbConfig.GetDSN()

	assert.Equal(t, expected, actual)
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		environment string
		expected    bool
	}{
		{"production", true},
		{"Production", true},
		{"PRODUCTION", true},
		{"development", false},
		{"dev", false},
		{"staging", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.environment, func(t *testing.T) {
			cfg := &Config{
				App: AppConfig{
					Environment: tt.environment,
				},
			}
			assert.Equal(t, tt.expected, cfg.IsProduction())
		})
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		environment string
		expected    bool
	}{
		{"development", true},
		{"Development", true},
		{"DEVELOPMENT", true},
		{"production", false},
		{"prod", false},
		{"staging", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.environment, func(t *testing.T) {
			cfg := &Config{
				App: AppConfig{
					Environment: tt.environment,
				},
			}
			assert.Equal(t, tt.expected, cfg.IsDevelopment())
		})
	}
}

func TestServerConfig_GetServerAddress(t *testing.T) {
	serverConfig := &ServerConfig{
		Host: "0.0.0.0",
		Port: 8080,
	}

	expected := "0.0.0.0:8080"
	actual := serverConfig.GetServerAddress()

	assert.Equal(t, expected, actual)
}

func TestWorkerConfig_GetWorkerAddress(t *testing.T) {
	workerConfig := &WorkerConfig{
		Port: 8081,
	}

	expected := ":8081"
	actual := workerConfig.GetWorkerAddress()

	assert.Equal(t, expected, actual)
}

func TestValidateConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test-app",
			},
			Server: ServerConfig{
				Port: 8080,
			},
			Database: DatabaseConfig{
				Host: "localhost",
			},
			Redis: RedisConfig{
				Addresses: []string{"localhost:6379"},
			},
			RateLimit: RateLimitConfig{
				KeyPrefix: "test:",
			},
		}

		err := validateConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("missing app name", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "",
			},
		}

		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "app.name is required")
	})

	t.Run("invalid server port", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test-app",
			},
			Server: ServerConfig{
				Port: 0,
			},
		}

		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server.port must be between 1 and 65535")
	})

	t.Run("missing database host", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test-app",
			},
			Server: ServerConfig{
				Port: 8080,
			},
			Database: DatabaseConfig{
				Host: "",
			},
		}

		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database.host is required")
	})

	t.Run("missing redis addresses", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test-app",
			},
			Server: ServerConfig{
				Port: 8080,
			},
			Database: DatabaseConfig{
				Host: "localhost",
			},
			Redis: RedisConfig{
				Addresses: []string{},
			},
		}

		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis.addresses is required")
	})

	t.Run("missing rate limit key prefix", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test-app",
			},
			Server: ServerConfig{
				Port: 8080,
			},
			Database: DatabaseConfig{
				Host: "localhost",
			},
			Redis: RedisConfig{
				Addresses: []string{"localhost:6379"},
			},
			RateLimit: RateLimitConfig{
				KeyPrefix: "",
			},
		}

		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate_limiter.key_prefix is required")
	})
}