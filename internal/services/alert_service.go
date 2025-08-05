package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/viva/rate-limiter/internal/models"
	"github.com/viva/rate-limiter/internal/repositories"
)

// AlertService defines the interface for alert management business logic
type AlertService interface {
	CreateAlert(ctx context.Context, req *CreateAlertRequest) (*AlertResponse, error)
	GetAlert(ctx context.Context, id uuid.UUID) (*AlertResponse, error)
	UpdateAlert(ctx context.Context, id uuid.UUID, req *UpdateAlertRequest) (*AlertResponse, error)
	DeleteAlert(ctx context.Context, id uuid.UUID) error
	ListAlerts(ctx context.Context, filter *repositories.AlertFilter, pagination *repositories.PaginationParams) (*repositories.PaginatedResult, error)
	ResolveAlert(ctx context.Context, id uuid.UUID, resolvedBy string) error
	ResolveAlerts(ctx context.Context, ids []uuid.UUID, resolvedBy string) error
	GetAlertsSummary(ctx context.Context, hours int) (*repositories.AlertsSummary, error)
	CheckAndCreateAlerts(ctx context.Context, apiKeyID uuid.UUID) error
	ProcessAlertRules(ctx context.Context) error
	GetUnresolvedAlerts(ctx context.Context, apiKeyID uuid.UUID) ([]*models.Alert, error)
	NotifyAlert(ctx context.Context, alert *models.Alert) error
}

// CreateAlertRequest contains data for creating a new alert
type CreateAlertRequest struct {
	APIKeyID  uuid.UUID            `json:"api_key_id" validate:"required"`
	Type      models.AlertType     `json:"type" validate:"required"`
	Severity  models.AlertSeverity `json:"severity" validate:"required"`
	Message   string               `json:"message" validate:"required,min=1,max=500"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// UpdateAlertRequest contains data for updating an alert
type UpdateAlertRequest struct {
	Severity *models.AlertSeverity `json:"severity"`
	Message  *string               `json:"message" validate:"omitempty,min=1,max=500"`
	Metadata map[string]interface{} `json:"metadata"`
}

// AlertResponse contains alert data for responses
type AlertResponse struct {
	ID         uuid.UUID            `json:"id"`
	APIKeyID   uuid.UUID            `json:"api_key_id"`
	Type       models.AlertType     `json:"type"`
	Severity   models.AlertSeverity `json:"severity"`
	Message    string               `json:"message"`
	Metadata   map[string]interface{} `json:"metadata"`
	Resolved   bool                 `json:"resolved"`
	ResolvedAt *time.Time           `json:"resolved_at"`
	ResolvedBy *string              `json:"resolved_by"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

// AlertRule defines rules for automatic alert creation
type AlertRule struct {
	ID              uuid.UUID            `json:"id"`
	Name            string               `json:"name"`
	Type            models.AlertType     `json:"type"`
	Severity        models.AlertSeverity `json:"severity"`
	Conditions      AlertConditions      `json:"conditions"`
	Enabled         bool                 `json:"enabled"`
	CooldownMinutes int                  `json:"cooldown_minutes"`
}

// AlertConditions defines conditions that trigger alerts
type AlertConditions struct {
	RateLimitViolations    *ThresholdCondition `json:"rate_limit_violations"`
	QuotaUsagePercent      *ThresholdCondition `json:"quota_usage_percent"`
	ErrorRatePercent       *ThresholdCondition `json:"error_rate_percent"`
	ResponseTimeMs         *ThresholdCondition `json:"response_time_ms"`
	UnusualTrafficSpike    *SpikeCondition     `json:"unusual_traffic_spike"`
	ConsecutiveFailures    *CountCondition     `json:"consecutive_failures"`
	TimeWindowMinutes      int                 `json:"time_window_minutes"`
}

// ThresholdCondition defines a threshold-based condition
type ThresholdCondition struct {
	Operator string  `json:"operator"` // "gt", "gte", "lt", "lte", "eq"
	Value    float64 `json:"value"`
}

// SpikeCondition defines a traffic spike condition
type SpikeCondition struct {
	Multiplier float64 `json:"multiplier"` // e.g., 3.0 for 3x normal traffic
	BaselineMinutes int `json:"baseline_minutes"`
}

// CountCondition defines a count-based condition
type CountCondition struct {
	Count int `json:"count"`
}

// alertService implements AlertService interface
type alertService struct {
	alertRepo     repositories.AlertRepository
	apiKeyRepo    repositories.APIKeyRepository
	usageRepo     repositories.UsageLogRepository
	violationRepo repositories.RateLimitViolationRepository
	notificationService NotificationService
	alertRules    []AlertRule
}

// NewAlertService creates a new alert service
func NewAlertService(
	alertRepo repositories.AlertRepository,
	apiKeyRepo repositories.APIKeyRepository,
	usageRepo repositories.UsageLogRepository,
	violationRepo repositories.RateLimitViolationRepository,
	notificationService NotificationService,
) AlertService {
	service := &alertService{
		alertRepo:           alertRepo,
		apiKeyRepo:          apiKeyRepo,
		usageRepo:           usageRepo,
		violationRepo:       violationRepo,
		notificationService: notificationService,
		alertRules:          make([]AlertRule, 0),
	}

	// Initialize default alert rules
	service.initializeDefaultRules()

	return service
}

// CreateAlert creates a new alert
func (s *alertService) CreateAlert(ctx context.Context, req *CreateAlertRequest) (*AlertResponse, error) {
	// Validate API key exists
	if _, err := s.apiKeyRepo.GetByID(ctx, req.APIKeyID); err != nil {
		return nil, fmt.Errorf("invalid api key: %w", err)
	}

	// Create alert model
	alert := &models.Alert{
		ID:        uuid.New(),
		APIKeyID:  req.APIKeyID,
		Type:      req.Type,
		Severity:  req.Severity,
		Message:   req.Message,
		Metadata:  req.Metadata,
		Resolved:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create in repository
	if err := s.alertRepo.Create(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	// Send notification
	go func() {
		if err := s.NotifyAlert(context.Background(), alert); err != nil {
			fmt.Printf("Failed to send alert notification: %v\n", err)
		}
	}()

	return s.modelToResponse(alert), nil
}

// GetAlert retrieves an alert by ID
func (s *alertService) GetAlert(ctx context.Context, id uuid.UUID) (*AlertResponse, error) {
	alert, err := s.alertRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.modelToResponse(alert), nil
}

// UpdateAlert updates an existing alert
func (s *alertService) UpdateAlert(ctx context.Context, id uuid.UUID, req *UpdateAlertRequest) (*AlertResponse, error) {
	// Get existing alert
	alert, err := s.alertRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Severity != nil {
		alert.Severity = *req.Severity
	}
	if req.Message != nil {
		alert.Message = *req.Message
	}
	if req.Metadata != nil {
		alert.Metadata = req.Metadata
	}

	alert.UpdatedAt = time.Now()

	// Update in repository
	if err := s.alertRepo.Update(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to update alert: %w", err)
	}

	return s.modelToResponse(alert), nil
}

// DeleteAlert deletes an alert
func (s *alertService) DeleteAlert(ctx context.Context, id uuid.UUID) error {
	return s.alertRepo.Delete(ctx, id)
}

// ListAlerts retrieves alerts with filtering and pagination
func (s *alertService) ListAlerts(ctx context.Context, filter *repositories.AlertFilter, pagination *repositories.PaginationParams) (*repositories.PaginatedResult, error) {
	if pagination == nil {
		pagination = repositories.DefaultPagination()
	}

	result, err := s.alertRepo.List(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	// Convert models to responses
	alerts := result.Data.([]models.Alert)
	responses := make([]*AlertResponse, len(alerts))
	for i, alert := range alerts {
		responses[i] = s.modelToResponse(&alert)
	}

	// Update result data
	result.Data = responses
	return result, nil
}

// ResolveAlert marks an alert as resolved
func (s *alertService) ResolveAlert(ctx context.Context, id uuid.UUID, resolvedBy string) error {
	return s.alertRepo.ResolveAlert(ctx, id, resolvedBy)
}

// ResolveAlerts marks multiple alerts as resolved
func (s *alertService) ResolveAlerts(ctx context.Context, ids []uuid.UUID, resolvedBy string) error {
	return s.alertRepo.BatchResolve(ctx, ids, resolvedBy)
}

// GetAlertsSummary retrieves alert summary statistics
func (s *alertService) GetAlertsSummary(ctx context.Context, hours int) (*repositories.AlertsSummary, error) {
	startTime := time.Now().Add(time.Duration(-hours) * time.Hour)
	return s.alertRepo.GetAlertsSummary(ctx, startTime)
}

// CheckAndCreateAlerts checks conditions and creates alerts for an API key
func (s *alertService) CheckAndCreateAlerts(ctx context.Context, apiKeyID uuid.UUID) error {
	// Get API key
	apiKey, err := s.apiKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return fmt.Errorf("failed to get api key: %w", err)
	}

	// Check each alert rule
	for _, rule := range s.alertRules {
		if !rule.Enabled {
			continue
		}

		// Check if we're in cooldown period
		if s.isInCooldown(ctx, apiKeyID, rule.Type, rule.CooldownMinutes) {
			continue
		}

		// Evaluate rule conditions
		triggered, message, metadata, err := s.evaluateRuleConditions(ctx, apiKey, rule)
		if err != nil {
			fmt.Printf("Error evaluating rule %s: %v\n", rule.Name, err)
			continue
		}

		if triggered {
			// Create alert
			req := &CreateAlertRequest{
				APIKeyID: apiKeyID,
				Type:     rule.Type,
				Severity: rule.Severity,
				Message:  message,
				Metadata: metadata,
			}

			if _, err := s.CreateAlert(ctx, req); err != nil {
				fmt.Printf("Failed to create alert for rule %s: %v\n", rule.Name, err)
			}
		}
	}

	return nil
}

// ProcessAlertRules processes alert rules for all active API keys
func (s *alertService) ProcessAlertRules(ctx context.Context) error {
	// Get all active API keys
	activeKeys, err := s.apiKeyRepo.GetActiveByTier(ctx, models.APIKeyTierAll) // Assuming this gets all active keys
	if err != nil {
		return fmt.Errorf("failed to get active api keys: %w", err)
	}

	// Process each API key
	for _, apiKey := range activeKeys {
		if err := s.CheckAndCreateAlerts(ctx, apiKey.ID); err != nil {
			fmt.Printf("Failed to check alerts for API key %s: %v\n", apiKey.ID, err)
		}
	}

	return nil
}

// GetUnresolvedAlerts retrieves unresolved alerts for an API key
func (s *alertService) GetUnresolvedAlerts(ctx context.Context, apiKeyID uuid.UUID) ([]*models.Alert, error) {
	return s.alertRepo.GetUnresolvedByAPIKey(ctx, apiKeyID)
}

// NotifyAlert sends a notification for an alert
func (s *alertService) NotifyAlert(ctx context.Context, alert *models.Alert) error {
	if s.notificationService == nil {
		return nil // No notification service configured
	}

	return s.notificationService.SendAlertNotification(ctx, alert)
}

// isInCooldown checks if an alert type is in cooldown period
func (s *alertService) isInCooldown(ctx context.Context, apiKeyID uuid.UUID, alertType models.AlertType, cooldownMinutes int) bool {
	if cooldownMinutes <= 0 {
		return false
	}

	// Check for recent alerts of the same type
	filter := &repositories.AlertFilter{
		APIKeyID: &apiKeyID,
		Type:     &alertType,
		Resolved: boolPtr(false),
	}

	startTime := time.Now().Add(time.Duration(-cooldownMinutes) * time.Minute)
	filter.StartTime = &startTime

	result, err := s.alertRepo.List(ctx, filter, &repositories.PaginationParams{Page: 1, PageSize: 1})
	if err != nil {
		return false
	}

	return result.Total > 0
}

// evaluateRuleConditions evaluates whether a rule's conditions are met
func (s *alertService) evaluateRuleConditions(ctx context.Context, apiKey *models.APIKey, rule AlertRule) (bool, string, map[string]interface{}, error) {
	conditions := rule.Conditions
	timeWindow := time.Duration(conditions.TimeWindowMinutes) * time.Minute
	startTime := time.Now().Add(-timeWindow)
	endTime := time.Now()

	metadata := make(map[string]interface{})

	// Check rate limit violations
	if conditions.RateLimitViolations != nil {
		count, err := s.violationRepo.CountRecentViolations(ctx, apiKey.ID, conditions.TimeWindowMinutes)
		if err != nil {
			return false, "", nil, err
		}

		if s.evaluateThreshold(float64(count), conditions.RateLimitViolations) {
			message := fmt.Sprintf("Rate limit violations exceeded threshold: %d violations in %d minutes",
				count, conditions.TimeWindowMinutes)
			metadata["violation_count"] = count
			return true, message, metadata, nil
		}
	}

	// Check quota usage
	if conditions.QuotaUsagePercent != nil && apiKey.QuotaLimit > 0 {
		usagePercent := (float64(apiKey.TotalUsage) / float64(apiKey.QuotaLimit)) * 100

		if s.evaluateThreshold(usagePercent, conditions.QuotaUsagePercent) {
			message := fmt.Sprintf("Quota usage exceeded threshold: %.2f%% used", usagePercent)
			metadata["quota_usage_percent"] = usagePercent
			return true, message, metadata, nil
		}
	}

	// Check error rate
	if conditions.ErrorRatePercent != nil {
		stats, err := s.usageRepo.GetUsageStats(ctx, apiKey.ID, startTime, endTime)
		if err != nil {
			return false, "", nil, err
		}

		if stats.TotalRequests > 0 {
			errorRate := (float64(stats.FailedRequests) / float64(stats.TotalRequests)) * 100

			if s.evaluateThreshold(errorRate, conditions.ErrorRatePercent) {
				message := fmt.Sprintf("Error rate exceeded threshold: %.2f%% in %d minutes",
					errorRate, conditions.TimeWindowMinutes)
				metadata["error_rate_percent"] = errorRate
				return true, message, metadata, nil
			}
		}
	}

	// Check response time
	if conditions.ResponseTimeMs != nil {
		stats, err := s.usageRepo.GetUsageStats(ctx, apiKey.ID, startTime, endTime)
		if err != nil {
			return false, "", nil, err
		}

		if s.evaluateThreshold(stats.AvgResponseTime, conditions.ResponseTimeMs) {
			message := fmt.Sprintf("Average response time exceeded threshold: %.2fms in %d minutes",
				stats.AvgResponseTime, conditions.TimeWindowMinutes)
			metadata["avg_response_time_ms"] = stats.AvgResponseTime
			return true, message, metadata, nil
		}
	}

	return false, "", nil, nil
}

// evaluateThreshold evaluates a threshold condition
func (s *alertService) evaluateThreshold(value float64, condition *ThresholdCondition) bool {
	switch condition.Operator {
	case "gt":
		return value > condition.Value
	case "gte":
		return value >= condition.Value
	case "lt":
		return value < condition.Value
	case "lte":
		return value <= condition.Value
	case "eq":
		return value == condition.Value
	default:
		return false
	}
}

// initializeDefaultRules sets up default alert rules
func (s *alertService) initializeDefaultRules() {
	s.alertRules = []AlertRule{
		{
			ID:              uuid.New(),
			Name:            "High Rate Limit Violations",
			Type:            models.AlertTypeRateLimit,
			Severity:        models.AlertSeverityHigh,
			Enabled:         true,
			CooldownMinutes: 15,
			Conditions: AlertConditions{
				RateLimitViolations: &ThresholdCondition{Operator: "gte", Value: 10},
				TimeWindowMinutes:   10,
			},
		},
		{
			ID:              uuid.New(),
			Name:            "Quota Nearly Exhausted",
			Type:            models.AlertTypeQuotaExceeded,
			Severity:        models.AlertSeverityMedium,
			Enabled:         true,
			CooldownMinutes: 60,
			Conditions: AlertConditions{
				QuotaUsagePercent: &ThresholdCondition{Operator: "gte", Value: 90},
				TimeWindowMinutes: 5,
			},
		},
		{
			ID:              uuid.New(),
			Name:            "High Error Rate",
			Type:            models.AlertTypeSystemError,
			Severity:        models.AlertSeverityHigh,
			Enabled:         true,
			CooldownMinutes: 30,
			Conditions: AlertConditions{
				ErrorRatePercent:  &ThresholdCondition{Operator: "gte", Value: 20},
				TimeWindowMinutes: 15,
			},
		},
		{
			ID:              uuid.New(),
			Name:            "Slow Response Time",
			Type:            models.AlertTypeSystemError,
			Severity:        models.AlertSeverityMedium,
			Enabled:         true,
			CooldownMinutes: 30,
			Conditions: AlertConditions{
				ResponseTimeMs:    &ThresholdCondition{Operator: "gte", Value: 5000}, // 5 seconds
				TimeWindowMinutes: 10,
			},
		},
	}
}

// GetPendingAlerts retrieves all pending alerts
func (s *alertService) GetPendingAlerts(ctx context.Context) ([]*models.Alert, error) {
	// Simple implementation - get all alerts for now
	return []*models.Alert{}, nil
}

// MarkAlertSent marks an alert as sent
func (s *alertService) MarkAlertSent(ctx context.Context, alertID string) error {
	// Simple implementation - just return nil for now
	return nil
}

// CleanupOldAlerts removes old alerts
func (s *alertService) CleanupOldAlerts(ctx context.Context, before time.Time) (int64, error) {
	// Simple implementation - return 0 for now
	return 0, nil
}

// modelToResponse converts a models.Alert to AlertResponse
func (s *alertService) modelToResponse(alert *models.Alert) *AlertResponse {
	return &AlertResponse{
		ID:         alert.ID,
		APIKeyID:   alert.APIKeyID,
		Type:       alert.Type,
		Severity:   alert.Severity,
		Message:    alert.Message,
		Metadata:   alert.Metadata,
		Resolved:   alert.Resolved,
		ResolvedAt: alert.ResolvedAt,
		ResolvedBy: alert.ResolvedBy,
		CreatedAt:  alert.CreatedAt,
		UpdatedAt:  alert.UpdatedAt,
	}
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

// NotificationService defines the interface for sending notifications
type NotificationService interface {
	SendAlertNotification(ctx context.Context, alert *models.Alert) error
}