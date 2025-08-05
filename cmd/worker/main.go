package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/viva/rate-limiter/internal/cache"
	"github.com/viva/rate-limiter/internal/config"
	"github.com/viva/rate-limiter/internal/models"
	"github.com/viva/rate-limiter/internal/queue"
	"github.com/viva/rate-limiter/internal/repositories"
	"github.com/viva/rate-limiter/internal/services"
)

func main() {
	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	logger.Info("Starting Viva Rate Limiter Worker",
		zap.String("version", cfg.App.Version),
		zap.String("environment", cfg.App.Environment),
	)

	// Initialize database
	if err := models.InitDB(&cfg.Database); err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer models.CloseDB()

	logger.Info("Database connected successfully")

	// Initialize Redis
	logger.Info("Connecting to Redis...")
	redisClient, err := cache.NewRedisClient(&cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	logger.Info("Redis connected successfully")

	// Initialize cache service
	cacheService := cache.NewCacheService(redisClient)

	// Get database connection
	db := models.GetDB()

	// Initialize repositories
	apiKeyRepo := repositories.NewAPIKeyRepository(db)
	usageLogRepo := repositories.NewUsageLogRepository(db)
	alertRepo := repositories.NewAlertRepository(db)
	violationRepo := repositories.NewRateLimitViolationRepository(db)
	billingRepo := repositories.NewBillingRecordRepository(db)

	logger.Info("Repositories initialized")

	// Initialize services
	apiKeyService := services.NewAPIKeyService(apiKeyRepo, usageLogRepo)
	rateLimitService := services.NewRateLimitService(apiKeyRepo, violationRepo, usageLogRepo, cacheService)
	usageTrackingService := services.NewUsageTrackingService(usageLogRepo, apiKeyRepo, cacheService)
	alertService := services.NewAlertService(alertRepo, apiKeyRepo, usageLogRepo, violationRepo, nil)
	billingService := services.NewBillingService(billingRepo, apiKeyRepo, usageLogRepo)

	logger.Info("Services initialized")

	// Create Asynq client for task enqueueing
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     cfg.Asynq.RedisAddr,
		Password: cfg.Asynq.RedisPassword,
		DB:       cfg.Asynq.RedisDB,
	})
	defer asynqClient.Close()

	// Create task handlers
	taskHandlers := queue.NewTaskHandlers(
		apiKeyService,
		rateLimitService,
		usageTrackingService,
		alertService,
		billingService,
		logger,
	)

	// Create Asynq server
	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.Asynq.RedisAddr,
			Password: cfg.Asynq.RedisPassword,
			DB:       cfg.Asynq.RedisDB,
		},
		asynq.Config{
			Concurrency: cfg.Asynq.Concurrency,
			Queues:      cfg.Asynq.Queues,
			StrictPriority: cfg.Asynq.StrictPriority,
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				return time.Duration(n) * time.Minute
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				logger.Error("Task processing failed",
					zap.String("type", task.Type()),
					zap.Error(err),
				)
			}),
			HealthCheckFunc: func(err error) {
				if err != nil {
					logger.Error("Worker health check failed", zap.Error(err))
				}
			},
			HealthCheckInterval: time.Duration(15) * time.Second,
		},
	)

	// Create mux for task routing
	mux := asynq.NewServeMux()
	
	// Register task handlers
	mux.HandleFunc(queue.TaskTypeProcessUsageLogs, taskHandlers.ProcessUsageLogs)
	mux.HandleFunc(queue.TaskTypeCheckRateLimit, taskHandlers.CheckRateLimit)
	mux.HandleFunc(queue.TaskTypeGenerateBilling, taskHandlers.GenerateBilling)
	mux.HandleFunc(queue.TaskTypeProcessAlerts, taskHandlers.ProcessAlerts)
	mux.HandleFunc(queue.TaskTypeCleanupExpiredData, taskHandlers.CleanupExpiredData)
	mux.HandleFunc(queue.TaskTypeSyncCacheWithDB, taskHandlers.SyncCacheWithDB)

	logger.Info("Task handlers registered")

	// Start periodic tasks scheduler
	go startPeriodicTasks(asynqClient, logger)

	// Run the server in a goroutine
	go func() {
		logger.Info("Starting Asynq worker server",
			zap.Int("concurrency", cfg.Asynq.Concurrency),
			zap.Any("queues", cfg.Asynq.Queues),
		)

		if err := srv.Run(mux); err != nil {
			logger.Fatal("Failed to run worker server", zap.Error(err))
		}
	}()

	logger.Info("Worker server started successfully",
		zap.String("redis", cfg.Asynq.RedisAddr),
		zap.Int("concurrency", cfg.Asynq.Concurrency),
	)

	// Setup signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down worker...")

	// Graceful shutdown
	srv.Shutdown()

	// Close connections
	if err := redisClient.Close(); err != nil {
		logger.Error("Failed to close Redis connection", zap.Error(err))
	}

	models.CloseDB()

	logger.Info("Worker shutdown complete")
}

// startPeriodicTasks schedules periodic background tasks
func startPeriodicTasks(client *asynq.Client, logger *zap.Logger) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Schedule usage log aggregation
			task := asynq.NewTask(queue.TaskTypeProcessUsageLogs, nil)
			if _, err := client.Enqueue(task, asynq.Queue("default")); err != nil {
				logger.Error("Failed to enqueue usage log processing", zap.Error(err))
			}

			// Schedule alert processing every 5 minutes
			if time.Now().Minute()%5 == 0 {
				alertTask := asynq.NewTask(queue.TaskTypeProcessAlerts, nil)
				if _, err := client.Enqueue(alertTask, asynq.Queue("critical")); err != nil {
					logger.Error("Failed to enqueue alert processing", zap.Error(err))
				}
			}

			// Schedule billing generation at midnight
			if time.Now().Hour() == 0 && time.Now().Minute() == 0 {
				billingTask := asynq.NewTask(queue.TaskTypeGenerateBilling, nil)
				if _, err := client.Enqueue(billingTask, asynq.Queue("low")); err != nil {
					logger.Error("Failed to enqueue billing generation", zap.Error(err))
				}
			}

			// Schedule cleanup daily at 3 AM
			if time.Now().Hour() == 3 && time.Now().Minute() == 0 {
				cleanupTask := asynq.NewTask(queue.TaskTypeCleanupExpiredData, nil)
				if _, err := client.Enqueue(cleanupTask, asynq.Queue("low")); err != nil {
					logger.Error("Failed to enqueue cleanup task", zap.Error(err))
				}
			}

			// Sync cache with DB every 10 minutes
			if time.Now().Minute()%10 == 0 {
				syncTask := asynq.NewTask(queue.TaskTypeSyncCacheWithDB, nil)
				if _, err := client.Enqueue(syncTask, asynq.Queue("default")); err != nil {
					logger.Error("Failed to enqueue cache sync", zap.Error(err))
				}
			}
		}
	}
}