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

// RateLimitViolationRepository defines the interface for rate limit violation data access
type RateLimitViolationRepository interface {
	Create(ctx context.Context, violation *models.RateLimitViolation) error
	BatchCreate(ctx context.Context, violations []*models.RateLimitViolation) error
	GetByID(ctx context.Context, id uint64) (*models.RateLimitViolation, error)
	List(ctx context.Context, filter *ViolationFilter, pagination *PaginationParams) (*PaginatedResult, error)
	GetByAPIKey(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) ([]*models.RateLimitViolation, error)
	GetViolationStats(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) (*ViolationStats, error)
	GetTopViolators(ctx context.Context, startTime, endTime time.Time, limit int) ([]*ViolatorInfo, error)
	GetViolationsByEndpoint(ctx context.Context, startTime, endTime time.Time) ([]*EndpointViolations, error)
	CountRecentViolations(ctx context.Context, apiKeyID uuid.UUID, minutes int) (int64, error)
	DeleteOldViolations(ctx context.Context, retentionDays int) (int64, error)
	GetHourlyViolations(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) ([]*HourlyViolations, error)
}

// ViolationFilter contains filter parameters for violation queries
type ViolationFilter struct {
	APIKeyID  *uuid.UUID `json:"api_key_id"`
	Endpoint  string     `json:"endpoint"`
	Method    string     `json:"method"`
	IPAddress string     `json:"ip_address"`
	Country   string     `json:"country"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	MinCount  *int       `json:"min_attempted_requests"`
}

// ViolationStats contains aggregated violation statistics
type ViolationStats struct {
	TotalViolations      int64   `json:"total_violations"`
	UniqueEndpoints      int64   `json:"unique_endpoints"`
	TotalAttemptedRequests int64 `json:"total_attempted_requests"`
	AverageAttempts      float64 `json:"average_attempts_per_violation"`
	PeakHour            string  `json:"peak_hour"`
	PeakHourViolations  int64   `json:"peak_hour_violations"`
}

// ViolatorInfo contains information about top violators
type ViolatorInfo struct {
	APIKeyID        uuid.UUID `json:"api_key_id"`
	TotalViolations int64     `json:"total_violations"`
	TotalAttempts   int64     `json:"total_attempts"`
	UniqueEndpoints int64     `json:"unique_endpoints"`
}

// EndpointViolations contains violation data by endpoint
type EndpointViolations struct {
	Endpoint        string  `json:"endpoint"`
	Method          string  `json:"method"`
	TotalViolations int64   `json:"total_violations"`
	UniqueAPIKeys   int64   `json:"unique_api_keys"`
	AvgAttempts     float64 `json:"avg_attempts"`
}

// HourlyViolations contains hourly violation data
type HourlyViolations struct {
	Hour            time.Time `json:"hour"`
	TotalViolations int64     `json:"total_violations"`
	UniqueAPIKeys   int64     `json:"unique_api_keys"`
}

// rateLimitViolationRepository implements RateLimitViolationRepository interface
type rateLimitViolationRepository struct {
	*baseRepository
}

// NewRateLimitViolationRepository creates a new rate limit violation repository
func NewRateLimitViolationRepository(db *gorm.DB) RateLimitViolationRepository {
	return &rateLimitViolationRepository{
		baseRepository: NewBaseRepository(db),
	}
}

// Create creates a new rate limit violation record
func (r *rateLimitViolationRepository) Create(ctx context.Context, violation *models.RateLimitViolation) error {
	if err := r.db.WithContext(ctx).Create(violation).Error; err != nil {
		return fmt.Errorf("failed to create rate limit violation: %w", err)
	}
	return nil
}

// BatchCreate creates multiple violation records
func (r *rateLimitViolationRepository) BatchCreate(ctx context.Context, violations []*models.RateLimitViolation) error {
	if len(violations) == 0 {
		return nil
	}

	// Create in batches of 1000 for better performance
	batchSize := 1000
	for i := 0; i < len(violations); i += batchSize {
		end := i + batchSize
		if end > len(violations) {
			end = len(violations)
		}

		if err := r.db.WithContext(ctx).CreateInBatches(violations[i:end], batchSize).Error; err != nil {
			return fmt.Errorf("failed to batch create violations: %w", err)
		}
	}
	return nil
}

// GetByID retrieves a violation by ID
func (r *rateLimitViolationRepository) GetByID(ctx context.Context, id uint64) (*models.RateLimitViolation, error) {
	var violation models.RateLimitViolation
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&violation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("violation not found")
		}
		return nil, fmt.Errorf("failed to get violation: %w", err)
	}
	return &violation, nil
}

// List retrieves violations with filtering and pagination
func (r *rateLimitViolationRepository) List(ctx context.Context, filter *ViolationFilter, pagination *PaginationParams) (*PaginatedResult, error) {
	query := r.db.WithContext(ctx).Model(&models.RateLimitViolation{})

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
		if filter.MinCount != nil {
			query = query.Where("attempted_requests >= ?", *filter.MinCount)
		}
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count violations: %w", err)
	}

	// Apply pagination
	var violations []models.RateLimitViolation
	if err := query.
		Order(pagination.GetOrderBy()).
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Find(&violations).Error; err != nil {
		return nil, fmt.Errorf("failed to list violations: %w", err)
	}

	return NewPaginatedResult(violations, total, pagination), nil
}

// GetByAPIKey retrieves violations for a specific API key within a time range
func (r *rateLimitViolationRepository) GetByAPIKey(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) ([]*models.RateLimitViolation, error) {
	var violations []*models.RateLimitViolation
	if err := r.db.WithContext(ctx).
		Where("api_key_id = ? AND timestamp >= ? AND timestamp <= ?",
			apiKeyID, startTime, endTime).
		Order("timestamp DESC").
		Find(&violations).Error; err != nil {
		return nil, fmt.Errorf("failed to get violations by api key: %w", err)
	}
	return violations, nil
}

// GetViolationStats retrieves aggregated violation statistics
func (r *rateLimitViolationRepository) GetViolationStats(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) (*ViolationStats, error) {
	stats := &ViolationStats{}

	// Basic statistics
	baseQuery := r.db.WithContext(ctx).
		Model(&models.RateLimitViolation{}).
		Where("api_key_id = ? AND timestamp >= ? AND timestamp <= ?", apiKeyID, startTime, endTime)

	// Total violations
	if err := baseQuery.Count(&stats.TotalViolations).Error; err != nil {
		return nil, fmt.Errorf("failed to count violations: %w", err)
	}

	// Unique endpoints
	if err := baseQuery.
		Select("COUNT(DISTINCT endpoint || method)").
		Scan(&stats.UniqueEndpoints).Error; err != nil {
		return nil, fmt.Errorf("failed to count unique endpoints: %w", err)
	}

	// Total attempted requests and average
	var result struct {
		TotalAttempts int64
		AvgAttempts   float64
	}
	if err := baseQuery.
		Select("COALESCE(SUM(attempted_requests), 0) as total_attempts, COALESCE(AVG(attempted_requests), 0) as avg_attempts").
		Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to get attempt statistics: %w", err)
	}
	stats.TotalAttemptedRequests = result.TotalAttempts
	stats.AverageAttempts = result.AvgAttempts

	// Peak hour analysis
	type peakResult struct {
		Hour       time.Time
		Violations int64
	}
	var peak peakResult
	query := `
		SELECT 
			DATE_TRUNC('hour', timestamp) as hour,
			COUNT(*) as violations
		FROM rate_limit_violations
		WHERE api_key_id = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY hour
		ORDER BY violations DESC
		LIMIT 1
	`
	if err := r.db.WithContext(ctx).Raw(query, apiKeyID, startTime, endTime).Scan(&peak).Error; err != nil {
		return nil, fmt.Errorf("failed to get peak hour: %w", err)
	}
	if peak.Violations > 0 {
		stats.PeakHour = peak.Hour.Format("2006-01-02 15:00")
		stats.PeakHourViolations = peak.Violations
	}

	return stats, nil
}

// GetTopViolators retrieves the API keys with most violations
func (r *rateLimitViolationRepository) GetTopViolators(ctx context.Context, startTime, endTime time.Time, limit int) ([]*ViolatorInfo, error) {
	var violators []*ViolatorInfo

	query := `
		SELECT 
			api_key_id,
			COUNT(*) as total_violations,
			COALESCE(SUM(attempted_requests), 0) as total_attempts,
			COUNT(DISTINCT endpoint || method) as unique_endpoints
		FROM rate_limit_violations
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY api_key_id
		ORDER BY total_violations DESC
		LIMIT ?
	`

	if err := r.db.WithContext(ctx).Raw(query, startTime, endTime, limit).Scan(&violators).Error; err != nil {
		return nil, fmt.Errorf("failed to get top violators: %w", err)
	}

	return violators, nil
}

// GetViolationsByEndpoint retrieves violation statistics grouped by endpoint
func (r *rateLimitViolationRepository) GetViolationsByEndpoint(ctx context.Context, startTime, endTime time.Time) ([]*EndpointViolations, error) {
	var violations []*EndpointViolations

	query := `
		SELECT 
			endpoint,
			method,
			COUNT(*) as total_violations,
			COUNT(DISTINCT api_key_id) as unique_api_keys,
			COALESCE(AVG(attempted_requests), 0) as avg_attempts
		FROM rate_limit_violations
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY endpoint, method
		ORDER BY total_violations DESC
	`

	if err := r.db.WithContext(ctx).Raw(query, startTime, endTime).Scan(&violations).Error; err != nil {
		return nil, fmt.Errorf("failed to get violations by endpoint: %w", err)
	}

	return violations, nil
}

// CountRecentViolations counts violations in the last N minutes
func (r *rateLimitViolationRepository) CountRecentViolations(ctx context.Context, apiKeyID uuid.UUID, minutes int) (int64, error) {
	var count int64
	cutoff := time.Now().Add(time.Duration(-minutes) * time.Minute)

	if err := r.db.WithContext(ctx).
		Model(&models.RateLimitViolation{}).
		Where("api_key_id = ? AND timestamp >= ?", apiKeyID, cutoff).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count recent violations: %w", err)
	}

	return count, nil
}

// DeleteOldViolations deletes violations older than the specified retention days
func (r *rateLimitViolationRepository) DeleteOldViolations(ctx context.Context, retentionDays int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	result := r.db.WithContext(ctx).
		Where("timestamp < ?", cutoffDate).
		Delete(&models.RateLimitViolation{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old violations: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// GetHourlyViolations retrieves hourly violation statistics
func (r *rateLimitViolationRepository) GetHourlyViolations(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) ([]*HourlyViolations, error) {
	var violations []*HourlyViolations

	query := `
		SELECT 
			DATE_TRUNC('hour', timestamp) as hour,
			COUNT(*) as total_violations,
			COUNT(DISTINCT api_key_id) as unique_api_keys
		FROM rate_limit_violations
		WHERE timestamp >= ? AND timestamp <= ?
	`

	args := []interface{}{startTime, endTime}
	if apiKeyID != uuid.Nil {
		query += " AND api_key_id = ?"
		args = append(args, apiKeyID)
	}

	query += " GROUP BY hour ORDER BY hour ASC"

	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&violations).Error; err != nil {
		return nil, fmt.Errorf("failed to get hourly violations: %w", err)
	}

	return violations, nil
}