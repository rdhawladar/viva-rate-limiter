package models

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/viva/rate-limiter/internal/config"
)

// DB holds the database connection
var DB *gorm.DB

// InitDB initializes the database connection
func InitDB(cfg *config.DatabaseConfig) error {
	var err error
	
	// Configure GORM logger
	logLevel := logger.Silent
	switch cfg.LogLevel {
	case "debug":
		logLevel = logger.Info
	case "info":
		logLevel = logger.Warn
	case "warn":
		logLevel = logger.Error
	case "error":
		logLevel = logger.Error
	}

	gormConfig := &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logLevel,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// Connect to database
	DB, err = gorm.Open(postgres.Open(cfg.GetDSN()), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Auto migrate if enabled
	if cfg.AutoMigrate {
		if err := AutoMigrate(); err != nil {
			return fmt.Errorf("failed to auto migrate: %w", err)
		}
	}

	return nil
}

// AutoMigrate runs database migrations
func AutoMigrate() error {
	err := DB.AutoMigrate(
		&APIKey{},
		&UsageLog{},
		&Alert{},
		&RateLimitViolation{},
		&BillingRecord{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	// Create indexes for better performance
	if err := createIndexes(); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// createIndexes creates additional database indexes
func createIndexes() error {
	// Create indexes one by one to handle any missing columns gracefully
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_usage_logs_api_key_timestamp ON usage_logs (api_key_id, timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_rate_violations_processed ON rate_limit_violations (processed_at) WHERE processed_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_billing_records_period ON billing_records (api_key_id, period_start, period_end)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_severity_status ON alerts (severity, created_at DESC)`,
	}
	
	for _, idx := range indexes {
		// Try to create each index, but don't fail if one fails
		if err := DB.Exec(idx).Error; err != nil {
			// Log the error but continue
			fmt.Printf("Warning: Could not create index: %v\n", err)
		}
	}

	return nil
}

// CloseDB closes the database connection
func CloseDB() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	return sqlDB.Close()
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// Transaction executes a function within a database transaction
func Transaction(fn func(*gorm.DB) error) error {
	return DB.Transaction(fn)
}

// HealthCheck checks if the database is healthy
func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}