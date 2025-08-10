package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/rdhawladar/viva-rate-limiter/internal/cache"
	"github.com/rdhawladar/viva-rate-limiter/internal/config"
	"github.com/rdhawladar/viva-rate-limiter/internal/controllers"
	"github.com/rdhawladar/viva-rate-limiter/internal/metrics"
	"github.com/rdhawladar/viva-rate-limiter/internal/middleware"
	"github.com/rdhawladar/viva-rate-limiter/internal/models"
	"github.com/rdhawladar/viva-rate-limiter/internal/repositories"
	"github.com/rdhawladar/viva-rate-limiter/internal/services"
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

	logger.Info("Starting Viva Rate Limiter API",
		zap.String("version", cfg.App.Version),
		zap.String("environment", cfg.App.Environment),
	)

	// Initialize database
	if err := models.InitDB(&cfg.Database); err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer models.CloseDB()

	logger.Info("Database connected successfully")

	// Run migrations if enabled
	if cfg.Database.AutoMigrate {
		logger.Info("Running database migrations...")
		if err := models.AutoMigrate(); err != nil {
			logger.Fatal("Failed to run migrations", zap.Error(err))
		}
		logger.Info("Database migrations completed")
	}

	// Initialize Prometheus metrics
	prometheusMetrics := metrics.NewPrometheusMetrics(cfg.Metrics.Namespace, cfg.Metrics.Subsystem)
	logger.Info("Prometheus metrics initialized")

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
	_ = repositories.NewBillingRecordRepository(db) // billingRepo - will be used later

	logger.Info("Repositories initialized")

	// Initialize services
	apiKeyService := services.NewAPIKeyService(apiKeyRepo, usageLogRepo)
	rateLimitService := services.NewRateLimitService(apiKeyRepo, violationRepo, usageLogRepo, cacheService)
	usageTrackingService := services.NewUsageTrackingService(usageLogRepo, apiKeyRepo, cacheService)
	alertService := services.NewAlertService(alertRepo, apiKeyRepo, usageLogRepo, violationRepo, nil) // nil for notification service

	logger.Info("Services initialized")

	// Initialize controllers
	healthController := controllers.NewHealthController(redisClient)
	apiKeyController := controllers.NewAPIKeyController(apiKeyService)
	rateLimitController := controllers.NewRateLimitController(rateLimitService, apiKeyService)
	swaggerController := controllers.NewSwaggerController()

	logger.Info("Controllers initialized")

	// Setup Gin router
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.MetricsMiddleware(prometheusMetrics))
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(gin.Recovery())

	// Health endpoints (no rate limiting)
	router.GET("/health", healthController.Health)
	router.GET("/ready", healthController.Ready)
	router.GET("/live", healthController.Live)

	// Swagger documentation endpoints (no rate limiting)
	router.GET("/swagger", swaggerController.ServeSwaggerRedirect)
	router.GET("/openapi.yaml", swaggerController.ServeOpenAPISpec)
	router.GET("/swagger/*any", swaggerController.ServeSwaggerUI())
	logger.Info("Swagger UI enabled at /swagger/")

	// Metrics endpoint
	if cfg.Metrics.Enabled {
		router.GET(cfg.Metrics.Path, gin.WrapH(promhttp.Handler()))
		logger.Info("Metrics endpoint enabled", zap.String("path", cfg.Metrics.Path))
	}

	// API routes with rate limiting
	v1 := router.Group("/api/v1")
	v1.Use(middleware.RateLimitMiddleware(apiKeyService, rateLimitService, usageTrackingService))
	{
		// API key management
		apiKeys := v1.Group("/api-keys")
		{
			apiKeys.POST("/", apiKeyController.CreateAPIKey)
			apiKeys.GET("/:id", apiKeyController.GetAPIKey)
			apiKeys.PUT("/:id", apiKeyController.UpdateAPIKey)
			apiKeys.DELETE("/:id", apiKeyController.DeleteAPIKey)
			apiKeys.GET("/", apiKeyController.ListAPIKeys)
			apiKeys.POST("/:id/rotate", apiKeyController.RotateAPIKey)
			apiKeys.GET("/:id/stats", apiKeyController.GetAPIKeyStats)
		}

		// Rate limiting
		rateLimit := v1.Group("/rate-limit")
		{
			rateLimit.POST("/check", rateLimitController.CheckRateLimit)
			rateLimit.GET("/:api_key_id/info", rateLimitController.GetRateLimitInfo)
			rateLimit.POST("/:api_key_id/reset", rateLimitController.ResetRateLimit)
			rateLimit.PUT("/:api_key_id", rateLimitController.UpdateRateLimit)
			rateLimit.GET("/:api_key_id/violations", rateLimitController.GetViolationHistory)
		}
	}

	// Public rate limit endpoints (no authentication required)
	public := router.Group("/api/public/v1")
	{
		public.POST("/rate-limit/validate", rateLimitController.ValidateAPIKey)
	}

	// Start HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting HTTP server",
			zap.Int("port", cfg.Server.Port),
			zap.String("environment", cfg.App.Environment),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	logger.Info("API server started successfully",
		zap.String("address", fmt.Sprintf("http://localhost:%d", cfg.Server.Port)),
	)

	// Start background alert processing
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := alertService.ProcessAlertRules(context.Background()); err != nil {
					logger.Error("Failed to process alert rules", zap.Error(err))
				}
			}
		}
	}()

	// Setup signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		logger.Error("Failed to close Redis connection", zap.Error(err))
	}

	// Close database connection
	models.CloseDB()

	logger.Info("Server shutdown complete")
}