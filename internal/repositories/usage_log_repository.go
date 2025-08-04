package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/viva/rate-limiter/internal/models"
)

// UsageLogRepository defines the interface for usage log data access
type UsageLogRepository interface {
	Create(ctx context.Context, log *models.UsageLog) error
	BatchCreate(ctx context.Context, logs []*models.UsageLog) error
	GetByID(ctx context.Context, id uint64) (*models.UsageLog, error)
	List(ctx context.Context, filter *UsageLogFilter, pagination *PaginationParams) (*PaginatedResult, error)
	GetUsageByAPIKey(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) ([]*models.UsageLog, error)
	GetUsageStats(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) (*UsageStats, error)
	GetEndpointStats(ctx context.Context, startTime, endTime time.Time) ([]*EndpointStats, error)
	GetTopAPIKeys(ctx context.Context, startTime, endTime time.Time, limit int) ([]*APIKeyUsage, error)
	DeleteOldLogs(ctx context.Context, retentionDays int) (int64, error)
	GetHourlyUsage(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) ([]*HourlyUsage, error)
}

// UsageLogFilter contains filter parameters for usage log queries
type UsageLogFilter struct {
	APIKeyID    *uuid.UUID `json:"api_key_id"`
	Endpoint    string     `json:"endpoint"`
	Method      string     `json:"method"`
	StatusCode  *int       `json:"status_code"`
	IPAddress   string     `json:"ip_address"`
	Country     string     `json:"country"`
	StartTime   *time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	MinResponse *int       `json:"min_response_time"`
	MaxResponse *int       `json:"max_response_time"`
}

// UsageStats contains aggregated usage statistics
type UsageStats struct {
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests     int64   `json:"failed_requests"`
	RateLimitedRequests int64  `json:"rate_limited_requests"`
	AvgResponseTime    float64 `json:"avg_response_time"`
	TotalBandwidth     int64   `json:"total_bandwidth"`
}

// EndpointStats contains statistics for an endpoint
type EndpointStats struct {
	Endpoint       string  `json:"endpoint"`
	Method         string  `json:"method"`
	TotalRequests  int64   `json:"total_requests"`
	AvgResponseTime float64 `json:"avg_response_time"`
	ErrorRate      float64 `json:"error_rate"`
}

// APIKeyUsage contains usage information for an API key
type APIKeyUsage struct {
	APIKeyID      uuid.UUID `json:"api_key_id"`
	TotalRequests int64     `json:"total_requests"`
	TotalBandwidth int64    `json:"total_bandwidth"`
}

// HourlyUsage contains hourly usage data
type HourlyUsage struct {
	Hour          time.Time `json:"hour"`
	TotalRequests int64     `json:"total_requests"`
	AvgResponseTime float64 `json:"avg_response_time"`
}

// usageLogRepository implements UsageLogRepository interface
type usageLogRepository struct {
	*baseRepository
}

// NewUsageLogRepository creates a new usage log repository
func NewUsageLogRepository(db *gorm.DB) UsageLogRepository {
	return &usageLogRepository{
		baseRepository: NewBaseRepository(db),
	}
}

// Create creates a new usage log entry
func (r *usageLogRepository) Create(ctx context.Context, log *models.UsageLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("failed to create usage log: %w", err)
	}
	return nil
}

// BatchCreate creates multiple usage log entries
func (r *usageLogRepository) BatchCreate(ctx context.Context, logs []*models.UsageLog) error {
	if len(logs) == 0 {
		return nil
	}

	// Create in batches of 1000 for better performance
	batchSize := 1000
	for i := 0; i < len(logs); i += batchSize {
		end := i + batchSize
		if end > len(logs) {
			end = len(logs)
		}
		
		if err := r.db.WithContext(ctx).CreateInBatches(logs[i:end], batchSize).Error; err != nil {
			return fmt.Errorf("failed to batch create usage logs: %w", err)
		}
	}
	return nil
}

// GetByID retrieves a usage log by ID
func (r *usageLogRepository) GetByID(ctx context.Context, id uint64) (*models.UsageLog, error) {
	var log models.UsageLog
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&log).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("usage log not found")
		}
		return nil, fmt.Errorf("failed to get usage log: %w", err)
	}
	return &log, nil
}

// List retrieves usage logs with filtering and pagination
func (r *usageLogRepository) List(ctx context.Context, filter *UsageLogFilter, pagination *PaginationParams) (*PaginatedResult, error) {
	query := r.db.WithContext(ctx).Model(&models.UsageLog{})

	// Apply filters
	if filter != nil {
		if filter.APIKeyID != nil {
			query = query.Where("api_key_id = ?", *filter.APIKeyID)
		}
		if filter.Endpoint != "" {
			query = query.Where("endpoint = ?", filter.Endpoint)
		}
		if filter.Method != "" {
			query = query.Where("method = ?", filter.Method)
		}
		if filter.StatusCode != nil {
			query = query.Where("status_code = ?", *filter.StatusCode)
		}
		if filter.IPAddress != "" {
			query = query.Where("ip_address = ?", filter.IPAddress)
		}
		if filter.Country != "" {
			query = query.Where("country = ?", filter.Country)
		}
		if filter.StartTime != nil {
			query = query.Where("timestamp >= ?", *filter.StartTime)
		}
		if filter.EndTime != nil {
			query = query.Where("timestamp <= ?", *filter.EndTime)
		}
		if filter.MinResponse != nil {
			query = query.Where("response_time >= ?", *filter.MinResponse)
		}
		if filter.MaxResponse != nil {
			query = query.Where("response_time <= ?", *filter.MaxResponse)
		}
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count usage logs: %w", err)
	}

	// Apply pagination
	var logs []models.UsageLog
	if err := query.
		Order(pagination.GetOrderBy()).
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to list usage logs: %w", err)
	}

	return NewPaginatedResult(logs, total, pagination), nil
}

// GetUsageByAPIKey retrieves usage logs for a specific API key within a time range
func (r *usageLogRepository) GetUsageByAPIKey(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) ([]*models.UsageLog, error) {
	var logs []*models.UsageLog
	if err := r.db.WithContext(ctx).
		Where("api_key_id = ? AND timestamp >= ? AND timestamp <= ?", 
			apiKeyID, startTime, endTime).
		Order("timestamp DESC").
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to get usage by api key: %w", err)
	}
	return logs, nil
}

// GetUsageStats retrieves aggregated usage statistics for an API key
func (r *usageLogRepository) GetUsageStats(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) (*UsageStats, error) {
	var stats UsageStats

	// Use raw SQL for better performance with aggregations
	query := `
		SELECT 
			COUNT(*) as total_requests,
			COUNT(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 END) as successful_requests,
			COUNT(CASE WHEN status_code >= 400 THEN 1 END) as failed_requests,
			COUNT(CASE WHEN status_code = 429 THEN 1 END) as rate_limited_requests,
			COALESCE(AVG(response_time), 0) as avg_response_time,
			COALESCE(SUM(request_size + response_size), 0) as total_bandwidth
		FROM usage_logs
		WHERE api_key_id = ? AND timestamp >= ? AND timestamp <= ?
	`

	if err := r.db.WithContext(ctx).Raw(query, apiKeyID, startTime, endTime).Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	return &stats, nil
}

// GetEndpointStats retrieves statistics grouped by endpoint
func (r *usageLogRepository) GetEndpointStats(ctx context.Context, startTime, endTime time.Time) ([]*EndpointStats, error) {
	var stats []*EndpointStats

	query := `
		SELECT 
			endpoint,
			method,
			COUNT(*) as total_requests,
			COALESCE(AVG(response_time), 0) as avg_response_time,
			COALESCE(CAST(COUNT(CASE WHEN status_code >= 400 THEN 1 END) AS FLOAT) / COUNT(*) * 100, 0) as error_rate
		FROM usage_logs
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY endpoint, method
		ORDER BY total_requests DESC
	`

	if err := r.db.WithContext(ctx).Raw(query, startTime, endTime).Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("failed to get endpoint stats: %w", err)
	}

	return stats, nil
}

// GetTopAPIKeys retrieves the top API keys by usage
func (r *usageLogRepository) GetTopAPIKeys(ctx context.Context, startTime, endTime time.Time, limit int) ([]*APIKeyUsage, error) {
	var usage []*APIKeyUsage

	query := `
		SELECT 
			api_key_id,
			COUNT(*) as total_requests,
			COALESCE(SUM(request_size + response_size), 0) as total_bandwidth
		FROM usage_logs
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY api_key_id
		ORDER BY total_requests DESC
		LIMIT ?
	`

	if err := r.db.WithContext(ctx).Raw(query, startTime, endTime, limit).Scan(&usage).Error; err != nil {
		return nil, fmt.Errorf("failed to get top api keys: %w", err)
	}

	return usage, nil
}

// DeleteOldLogs deletes usage logs older than the specified retention days
func (r *usageLogRepository) DeleteOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	result := r.db.WithContext(ctx).
		Where("timestamp < ?", cutoffDate).
		Delete(&models.UsageLog{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old logs: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// GetHourlyUsage retrieves hourly usage statistics for an API key
func (r *usageLogRepository) GetHourlyUsage(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) ([]*HourlyUsage, error) {
	var usage []*HourlyUsage

	query := `
		SELECT 
			DATE_TRUNC('hour', timestamp) as hour,
			COUNT(*) as total_requests,
			COALESCE(AVG(response_time), 0) as avg_response_time
		FROM usage_logs
		WHERE api_key_id = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY hour
		ORDER BY hour ASC
	`

	if err := r.db.WithContext(ctx).Raw(query, apiKeyID, startTime, endTime).Scan(&usage).Error; err != nil {
		return nil, fmt.Errorf("failed to get hourly usage: %w", err)
	}

	return usage, nil
}