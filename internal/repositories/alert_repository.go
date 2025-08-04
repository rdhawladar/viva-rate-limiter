package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/viva/rate-limiter/internal/models"
)

// AlertRepository defines the interface for alert data access
type AlertRepository interface {
	Create(ctx context.Context, alert *models.Alert) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Alert, error)
	Update(ctx context.Context, alert *models.Alert) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *AlertFilter, pagination *PaginationParams) (*PaginatedResult, error)
	GetUnresolvedByAPIKey(ctx context.Context, apiKeyID uuid.UUID) ([]*models.Alert, error)
	GetByType(ctx context.Context, alertType models.AlertType, resolved bool) ([]*models.Alert, error)
	ResolveAlert(ctx context.Context, id uuid.UUID, resolvedBy string) error
	BatchResolve(ctx context.Context, ids []uuid.UUID, resolvedBy string) error
	CountBySeverity(ctx context.Context, severity models.AlertSeverity, resolved bool) (int64, error)
	GetRecentAlerts(ctx context.Context, limit int) ([]*models.Alert, error)
	GetAlertsSummary(ctx context.Context, startTime time.Time) (*AlertsSummary, error)
}

// AlertFilter contains filter parameters for alert queries
type AlertFilter struct {
	APIKeyID    *uuid.UUID            `json:"api_key_id"`
	Type        *models.AlertType     `json:"type"`
	Severity    *models.AlertSeverity `json:"severity"`
	Resolved    *bool                 `json:"resolved"`
	StartTime   *time.Time            `json:"start_time"`
	EndTime     *time.Time            `json:"end_time"`
	ResolvedBy  string                `json:"resolved_by"`
	Search      string                `json:"search"`
}

// AlertsSummary contains summarized alert statistics
type AlertsSummary struct {
	TotalAlerts       int64                       `json:"total_alerts"`
	UnresolvedAlerts  int64                       `json:"unresolved_alerts"`
	BySeverity        map[string]int64            `json:"by_severity"`
	ByType            map[string]int64            `json:"by_type"`
	AverageResolution float64                     `json:"average_resolution_hours"`
}

// alertRepository implements AlertRepository interface
type alertRepository struct {
	*baseRepository
}

// NewAlertRepository creates a new alert repository
func NewAlertRepository(db *gorm.DB) AlertRepository {
	return &alertRepository{
		baseRepository: NewBaseRepository(db),
	}
}

// Create creates a new alert
func (r *alertRepository) Create(ctx context.Context, alert *models.Alert) error {
	if err := r.db.WithContext(ctx).Create(alert).Error; err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}
	return nil
}

// GetByID retrieves an alert by ID
func (r *alertRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Alert, error) {
	var alert models.Alert
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&alert).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("alert not found")
		}
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}
	return &alert, nil
}

// Update updates an alert
func (r *alertRepository) Update(ctx context.Context, alert *models.Alert) error {
	result := r.db.WithContext(ctx).Model(alert).Updates(alert)
	if result.Error != nil {
		return fmt.Errorf("failed to update alert: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("alert not found")
	}
	return nil
}

// Delete deletes an alert
func (r *alertRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Alert{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete alert: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("alert not found")
	}
	return nil
}

// List retrieves alerts with filtering and pagination
func (r *alertRepository) List(ctx context.Context, filter *AlertFilter, pagination *PaginationParams) (*PaginatedResult, error) {
	query := r.db.WithContext(ctx).Model(&models.Alert{})

	// Apply filters
	if filter != nil {
		if filter.APIKeyID != nil {
			query = query.Where("api_key_id = ?", *filter.APIKeyID)
		}
		if filter.Type != nil {
			query = query.Where("type = ?", *filter.Type)
		}
		if filter.Severity != nil {
			query = query.Where("severity = ?", *filter.Severity)
		}
		if filter.Resolved != nil {
			query = query.Where("resolved = ?", *filter.Resolved)
		}
		if filter.StartTime != nil {
			query = query.Where("created_at >= ?", *filter.StartTime)
		}
		if filter.EndTime != nil {
			query = query.Where("created_at <= ?", *filter.EndTime)
		}
		if filter.ResolvedBy != "" {
			query = query.Where("resolved_by = ?", filter.ResolvedBy)
		}
		if filter.Search != "" {
			query = query.Where("message ILIKE ?", "%"+filter.Search+"%")
		}
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count alerts: %w", err)
	}

	// Apply pagination
	var alerts []models.Alert
	if err := query.
		Order(pagination.GetOrderBy()).
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Find(&alerts).Error; err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}

	return NewPaginatedResult(alerts, total, pagination), nil
}

// GetUnresolvedByAPIKey retrieves unresolved alerts for a specific API key
func (r *alertRepository) GetUnresolvedByAPIKey(ctx context.Context, apiKeyID uuid.UUID) ([]*models.Alert, error) {
	var alerts []*models.Alert
	if err := r.db.WithContext(ctx).
		Where("api_key_id = ? AND resolved = ?", apiKeyID, false).
		Order("created_at DESC").
		Find(&alerts).Error; err != nil {
		return nil, fmt.Errorf("failed to get unresolved alerts: %w", err)
	}
	return alerts, nil
}

// GetByType retrieves alerts by type and resolution status
func (r *alertRepository) GetByType(ctx context.Context, alertType models.AlertType, resolved bool) ([]*models.Alert, error) {
	var alerts []*models.Alert
	if err := r.db.WithContext(ctx).
		Where("type = ? AND resolved = ?", alertType, resolved).
		Order("created_at DESC").
		Find(&alerts).Error; err != nil {
		return nil, fmt.Errorf("failed to get alerts by type: %w", err)
	}
	return alerts, nil
}

// ResolveAlert marks an alert as resolved
func (r *alertRepository) ResolveAlert(ctx context.Context, id uuid.UUID, resolvedBy string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"resolved":     true,
			"resolved_at":  now,
			"resolved_by":  resolvedBy,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to resolve alert: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("alert not found")
	}
	return nil
}

// BatchResolve resolves multiple alerts
func (r *alertRepository) BatchResolve(ctx context.Context, ids []uuid.UUID, resolvedBy string) error {
	if len(ids) == 0 {
		return nil
	}

	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"resolved":     true,
			"resolved_at":  now,
			"resolved_by":  resolvedBy,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to batch resolve alerts: %w", result.Error)
	}
	return nil
}

// CountBySeverity counts alerts by severity and resolution status
func (r *alertRepository) CountBySeverity(ctx context.Context, severity models.AlertSeverity, resolved bool) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Where("severity = ? AND resolved = ?", severity, resolved).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count by severity: %w", err)
	}
	return count, nil
}

// GetRecentAlerts retrieves the most recent alerts
func (r *alertRepository) GetRecentAlerts(ctx context.Context, limit int) ([]*models.Alert, error) {
	var alerts []*models.Alert
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&alerts).Error; err != nil {
		return nil, fmt.Errorf("failed to get recent alerts: %w", err)
	}
	return alerts, nil
}

// GetAlertsSummary retrieves summarized alert statistics
func (r *alertRepository) GetAlertsSummary(ctx context.Context, startTime time.Time) (*AlertsSummary, error) {
	summary := &AlertsSummary{
		BySeverity: make(map[string]int64),
		ByType:     make(map[string]int64),
	}

	// Get total and unresolved counts
	if err := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Where("created_at >= ?", startTime).
		Count(&summary.TotalAlerts).Error; err != nil {
		return nil, fmt.Errorf("failed to count total alerts: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Where("created_at >= ? AND resolved = ?", startTime, false).
		Count(&summary.UnresolvedAlerts).Error; err != nil {
		return nil, fmt.Errorf("failed to count unresolved alerts: %w", err)
	}

	// Count by severity
	severities := []models.AlertSeverity{
		models.AlertSeverityCritical,
		models.AlertSeverityHigh,
		models.AlertSeverityMedium,
		models.AlertSeverityLow,
		models.AlertSeverityInfo,
	}

	for _, severity := range severities {
		var count int64
		if err := r.db.WithContext(ctx).
			Model(&models.Alert{}).
			Where("created_at >= ? AND severity = ?", startTime, severity).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("failed to count by severity %s: %w", severity, err)
		}
		summary.BySeverity[string(severity)] = count
	}

	// Count by type
	types := []models.AlertType{
		models.AlertTypeRateLimit,
		models.AlertTypeQuotaExceeded,
		models.AlertTypeAbnormalUsage,
		models.AlertTypeSystemError,
		models.AlertTypeSecurityIssue,
	}

	for _, alertType := range types {
		var count int64
		if err := r.db.WithContext(ctx).
			Model(&models.Alert{}).
			Where("created_at >= ? AND type = ?", startTime, alertType).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("failed to count by type %s: %w", alertType, err)
		}
		summary.ByType[string(alertType)] = count
	}

	// Calculate average resolution time
	var avgResolution float64
	query := `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600), 0) as avg_hours
		FROM alerts
		WHERE created_at >= ? AND resolved = true AND resolved_at IS NOT NULL
	`
	if err := r.db.WithContext(ctx).Raw(query, startTime).Scan(&avgResolution).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate average resolution time: %w", err)
	}
	summary.AverageResolution = avgResolution

	return summary, nil
}