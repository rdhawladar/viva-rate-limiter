package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/rdhawladar/viva-rate-limiter/internal/services"
)

// Task type constants
const (
	TaskTypeProcessUsageLogs   = "usage:process_logs"
	TaskTypeCheckRateLimit     = "ratelimit:check"
	TaskTypeGenerateBilling    = "billing:generate"
	TaskTypeProcessAlerts      = "alerts:process"
	TaskTypeCleanupExpiredData = "cleanup:expired_data"
	TaskTypeSyncCacheWithDB    = "cache:sync"
)

// TaskHandlers contains all task handler implementations
type TaskHandlers struct {
	apiKeyService        *services.APIKeyService
	rateLimitService     *services.RateLimitService
	usageTrackingService *services.UsageTrackingService
	alertService         *services.AlertService
	billingService       *services.BillingService
	logger               *zap.Logger
}

// NewTaskHandlers creates a new instance of TaskHandlers
func NewTaskHandlers(
	apiKeyService *services.APIKeyService,
	rateLimitService *services.RateLimitService,
	usageTrackingService *services.UsageTrackingService,
	alertService *services.AlertService,
	billingService *services.BillingService,
	logger *zap.Logger,
) *TaskHandlers {
	return &TaskHandlers{
		apiKeyService:        apiKeyService,
		rateLimitService:     rateLimitService,
		usageTrackingService: usageTrackingService,
		alertService:         alertService,
		billingService:       billingService,
		logger:               logger,
	}
}

// ProcessUsageLogs aggregates and processes usage logs
func (h *TaskHandlers) ProcessUsageLogs(ctx context.Context, t *asynq.Task) error {
	h.logger.Info("Processing usage logs",
		zap.String("task_id", t.ResultWriter().TaskID()),
	)

	startTime := time.Now()

	// Get all active API keys
	apiKeys, err := h.apiKeyService.ListAPIKeys(ctx, 1000, 0)
	if err != nil {
		return fmt.Errorf("failed to list API keys: %w", err)
	}

	processed := 0
	for _, apiKey := range apiKeys {
		// Aggregate usage for each API key
		usage, err := h.usageTrackingService.GetCurrentUsage(ctx, apiKey.ID.String())
		if err != nil {
			h.logger.Error("Failed to get usage for API key",
				zap.String("api_key_id", apiKey.ID.String()),
				zap.Error(err),
			)
			continue
		}

		// Check if usage exceeds limits
		if usage > int64(apiKey.RateLimit) {
			h.logger.Warn("API key exceeded rate limit",
				zap.String("api_key_id", apiKey.ID.String()),
				zap.Int64("usage", usage),
				zap.Int("limit", apiKey.RateLimit),
			)
		}

		processed++
	}

	h.logger.Info("Usage log processing completed",
		zap.Int("processed", processed),
		zap.Duration("duration", time.Since(startTime)),
	)

	return nil
}

// CheckRateLimit performs rate limit checks
func (h *TaskHandlers) CheckRateLimit(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		APIKeyID string `json:"api_key_id"`
	}

	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	h.logger.Info("Checking rate limit",
		zap.String("api_key_id", payload.APIKeyID),
	)

	allowed, remaining, resetAt, err := h.rateLimitService.CheckRateLimit(ctx, payload.APIKeyID)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}

	h.logger.Info("Rate limit check completed",
		zap.String("api_key_id", payload.APIKeyID),
		zap.Bool("allowed", allowed),
		zap.Int("remaining", remaining),
		zap.Time("reset_at", resetAt),
	)

	return nil
}

// GenerateBilling generates billing records
func (h *TaskHandlers) GenerateBilling(ctx context.Context, t *asynq.Task) error {
	h.logger.Info("Generating billing records",
		zap.String("task_id", t.ResultWriter().TaskID()),
	)

	startTime := time.Now()

	// Get billing period (previous month)
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)

	// Generate billing for all API keys
	records, err := h.billingService.GenerateBillingRecords(ctx, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("failed to generate billing records: %w", err)
	}

	h.logger.Info("Billing generation completed",
		zap.Int("records", len(records)),
		zap.Duration("duration", time.Since(startTime)),
		zap.Time("period_start", periodStart),
		zap.Time("period_end", periodEnd),
	)

	return nil
}

// ProcessAlerts processes alert rules and sends notifications
func (h *TaskHandlers) ProcessAlerts(ctx context.Context, t *asynq.Task) error {
	h.logger.Info("Processing alerts",
		zap.String("task_id", t.ResultWriter().TaskID()),
	)

	startTime := time.Now()

	// Process all alert rules
	if err := h.alertService.ProcessAlertRules(ctx); err != nil {
		return fmt.Errorf("failed to process alert rules: %w", err)
	}

	// Check for pending alerts to send
	pendingAlerts, err := h.alertService.GetPendingAlerts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending alerts: %w", err)
	}

	sent := 0
	for _, alert := range pendingAlerts {
		// Send notification (implement notification service)
		h.logger.Info("Alert triggered",
			zap.String("alert_id", alert.ID.String()),
			zap.String("type", string(alert.Type)),
			zap.String("severity", string(alert.Severity)),
		)

		// Mark as sent
		if err := h.alertService.MarkAlertSent(ctx, alert.ID.String()); err != nil {
			h.logger.Error("Failed to mark alert as sent",
				zap.String("alert_id", alert.ID.String()),
				zap.Error(err),
			)
			continue
		}

		sent++
	}

	h.logger.Info("Alert processing completed",
		zap.Int("alerts_sent", sent),
		zap.Duration("duration", time.Since(startTime)),
	)

	return nil
}

// CleanupExpiredData removes old data from the database
func (h *TaskHandlers) CleanupExpiredData(ctx context.Context, t *asynq.Task) error {
	h.logger.Info("Cleaning up expired data",
		zap.String("task_id", t.ResultWriter().TaskID()),
	)

	startTime := time.Now()

	// Define retention periods
	usageLogRetention := 90 * 24 * time.Hour    // 90 days
	violationRetention := 180 * 24 * time.Hour  // 180 days
	alertRetention := 365 * 24 * time.Hour      // 1 year

	// Cleanup old usage logs
	usageLogsCleaned, err := h.usageTrackingService.CleanupOldLogs(ctx, time.Now().Add(-usageLogRetention))
	if err != nil {
		h.logger.Error("Failed to cleanup usage logs", zap.Error(err))
	}

	// Cleanup old violations
	violationsCleaned, err := h.rateLimitService.CleanupOldViolations(ctx, time.Now().Add(-violationRetention))
	if err != nil {
		h.logger.Error("Failed to cleanup violations", zap.Error(err))
	}

	// Cleanup old alerts
	alertsCleaned, err := h.alertService.CleanupOldAlerts(ctx, time.Now().Add(-alertRetention))
	if err != nil {
		h.logger.Error("Failed to cleanup alerts", zap.Error(err))
	}

	h.logger.Info("Cleanup completed",
		zap.Int64("usage_logs_cleaned", usageLogsCleaned),
		zap.Int64("violations_cleaned", violationsCleaned),
		zap.Int64("alerts_cleaned", alertsCleaned),
		zap.Duration("duration", time.Since(startTime)),
	)

	return nil
}

// SyncCacheWithDB synchronizes cache with database
func (h *TaskHandlers) SyncCacheWithDB(ctx context.Context, t *asynq.Task) error {
	h.logger.Info("Syncing cache with database",
		zap.String("task_id", t.ResultWriter().TaskID()),
	)

	startTime := time.Now()

	// Get all active API keys
	apiKeys, err := h.apiKeyService.ListAPIKeys(ctx, 1000, 0)
	if err != nil {
		return fmt.Errorf("failed to list API keys: %w", err)
	}

	synced := 0
	for _, apiKey := range apiKeys {
		// Sync rate limit counters
		if err := h.rateLimitService.SyncCounterWithDB(ctx, apiKey.ID.String()); err != nil {
			h.logger.Error("Failed to sync counter for API key",
				zap.String("api_key_id", apiKey.ID.String()),
				zap.Error(err),
			)
			continue
		}

		// Sync usage counters
		if err := h.usageTrackingService.SyncUsageWithDB(ctx, apiKey.ID.String()); err != nil {
			h.logger.Error("Failed to sync usage for API key",
				zap.String("api_key_id", apiKey.ID.String()),
				zap.Error(err),
			)
			continue
		}

		synced++
	}

	h.logger.Info("Cache sync completed",
		zap.Int("synced", synced),
		zap.Duration("duration", time.Since(startTime)),
	)

	return nil
}

// CreateTask creates a new task for enqueueing
func CreateTask(taskType string, payload interface{}) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return asynq.NewTask(taskType, data), nil
}