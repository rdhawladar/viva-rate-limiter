package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	App        AppConfig        `mapstructure:"app"`
	Server     ServerConfig     `mapstructure:"server"`
	Worker     WorkerConfig     `mapstructure:"worker"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	RateLimit  RateLimitConfig  `mapstructure:"rate_limiter"`
	Asynq      AsynqConfig      `mapstructure:"asynq"`
	Metrics    MetricsConfig    `mapstructure:"metrics"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Security   SecurityConfig   `mapstructure:"security"`
	Alerts     AlertsConfig     `mapstructure:"alerts"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
	Debug       bool   `mapstructure:"debug"`
}

type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type WorkerConfig struct {
	Port         int            `mapstructure:"port"`
	Concurrency  int            `mapstructure:"concurrency"`
	Queues       map[string]int `mapstructure:"queues"`
	StrictPriority bool         `mapstructure:"strict_priority"`
}

type DatabaseConfig struct {
	Host              string        `mapstructure:"host"`
	Port              int           `mapstructure:"port"`
	User              string        `mapstructure:"user"`
	Password          string        `mapstructure:"password"`
	Name              string        `mapstructure:"name"`
	SSLMode           string        `mapstructure:"sslmode"`
	Timezone          string        `mapstructure:"timezone"`
	MaxOpenConns      int           `mapstructure:"max_open_conns"`
	MaxIdleConns      int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime   time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime   time.Duration `mapstructure:"conn_max_idle_time"`
	AutoMigrate       bool          `mapstructure:"auto_migrate"`
	LogLevel          string        `mapstructure:"log_level"`
}

type RedisConfig struct {
	Mode           string        `mapstructure:"mode"`
	Addresses      []string      `mapstructure:"addresses"`
	Password       string        `mapstructure:"password"`
	DB             int           `mapstructure:"db"`
	PoolSize       int           `mapstructure:"pool_size"`
	MinIdleConns   int           `mapstructure:"min_idle_conns"`
	MaxRetries     int           `mapstructure:"max_retries"`
	RetryDelay     time.Duration `mapstructure:"retry_delay"`
	DialTimeout    time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	PoolTimeout    time.Duration `mapstructure:"pool_timeout"`
	IdleTimeout    time.Duration `mapstructure:"idle_timeout"`
}

type RateLimitConfig struct {
	DefaultAlgorithm string                    `mapstructure:"default_algorithm"`
	DefaultLimits    map[string]RateLimitTier  `mapstructure:"default_limits"`
	KeyPrefix        string                    `mapstructure:"key_prefix"`
	CleanupInterval  time.Duration             `mapstructure:"cleanup_interval"`
}

type RateLimitTier struct {
	Requests int           `mapstructure:"requests"`
	Window   time.Duration `mapstructure:"window"`
	Burst    int           `mapstructure:"burst"`
}

type AsynqConfig struct {
	RedisAddr           string            `mapstructure:"redis_addr"`
	RedisPassword       string            `mapstructure:"redis_password"`
	RedisDB             int               `mapstructure:"redis_db"`
	Concurrency         int               `mapstructure:"concurrency"`
	Queues              map[string]int    `mapstructure:"queues"`
	StrictPriority      bool              `mapstructure:"strict_priority"`
	RetentionPeriod     time.Duration     `mapstructure:"retention_period"`
	HealthCheckInterval time.Duration     `mapstructure:"healthcheck_interval"`
}

type MetricsConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Path      string `mapstructure:"path"`
	Port      int    `mapstructure:"port"`
	Namespace string `mapstructure:"namespace"`
	Subsystem string `mapstructure:"subsystem"`
}

type LoggingConfig struct {
	Level       string `mapstructure:"level"`
	Format      string `mapstructure:"format"`
	Output      string `mapstructure:"output"`
	FilePath    string `mapstructure:"file_path"`
	MaxSize     int    `mapstructure:"max_size"`
	MaxBackups  int    `mapstructure:"max_backups"`
	MaxAge      int    `mapstructure:"max_age"`
	Compress    bool   `mapstructure:"compress"`
}

type SecurityConfig struct {
	APIKeyLength int        `mapstructure:"api_key_length"`
	HashCost     int        `mapstructure:"hash_cost"`
	JWTSecret    string     `mapstructure:"jwt_secret"`
	JWTExpiry    time.Duration `mapstructure:"jwt_expiry"`
	CORS         CORSConfig `mapstructure:"cors"`
}

type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
}

type AlertsConfig struct {
	Enabled       bool              `mapstructure:"enabled"`
	Thresholds    AlertThresholds   `mapstructure:"thresholds"`
	Notifications NotificationConfig `mapstructure:"notifications"`
}

type AlertThresholds struct {
	UsageWarning         int `mapstructure:"usage_warning"`
	UsageCritical        int `mapstructure:"usage_critical"`
	RateLimitViolations  int `mapstructure:"rate_limit_violations"`
}

type NotificationConfig struct {
	WebhookURL   string `mapstructure:"webhook_url"`
	EmailEnabled bool   `mapstructure:"email_enabled"`
	SlackEnabled bool   `mapstructure:"slack_enabled"`
}

type MonitoringConfig struct {
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
	MetricsInterval     time.Duration `mapstructure:"metrics_interval"`
	TraceSampleRate     float64       `mapstructure:"trace_sample_rate"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Set default configuration file
	viper.SetConfigName("dev")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("../configs")
	viper.AddConfigPath("../../configs")

	// Check for environment-specific config
	env := os.Getenv("VIVA_ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}
	if env == "" {
		env = "dev"
	}

	// Load environment-specific config
	viper.SetConfigName(env)

	// Enable environment variable binding
	viper.AutomaticEnv()
	viper.SetEnvPrefix("VIVA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal config
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate config
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	if cfg.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}

	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}

	if cfg.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}

	if len(cfg.Redis.Addresses) == 0 {
		return fmt.Errorf("redis.addresses is required")
	}

	if cfg.RateLimit.KeyPrefix == "" {
		return fmt.Errorf("rate_limiter.key_prefix is required")
	}

	return nil
}

// GetDSN returns the database connection string
func (d *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode, d.Timezone)
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.App.Environment) == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.App.Environment) == "development"
}

// GetServerAddress returns the full server address
func (s *ServerConfig) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// GetWorkerAddress returns the full worker address
func (w *WorkerConfig) GetWorkerAddress() string {
	return fmt.Sprintf(":%d", w.Port)
}