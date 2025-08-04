package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/viva/rate-limiter/internal/models"
	"github.com/viva/rate-limiter/internal/repositories"
)

// UsageTrackingService defines the interface for usage tracking business logic
type UsageTrackingService interface {
	LogUsage(ctx context.Context, req *UsageLogRequest) error
	BatchLogUsage(ctx context.Context, requests []*UsageLogRequest) error
	GetUsageStats(ctx context.Context, apiKeyID uuid.UUID, period TimePeriod) (*UsageStatistics, error)
	GetUsageHistory(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) ([]*models.UsageLog, error)
	GetTopEndpoints(ctx context.Context, apiKeyID uuid.UUID, period TimePeriod, limit int) ([]*EndpointUsageStats, error)
	GetHourlyUsage(ctx context.Context, apiKeyID uuid.UUID, period TimePeriod) ([]*HourlyUsageStats, error)
	GetUsageTrends(ctx context.Context, apiKeyID uuid.UUID, period TimePeriod) (*UsageTrends, error)
	CleanupOldLogs(ctx context.Context, retentionDays int) (int64, error)
	ExportUsageData(ctx context.Context, req *ExportRequest) (*ExportResult, error)
}

// UsageLogRequest contains data for logging API usage
type UsageLogRequest struct {
	APIKeyID     uuid.UUID `json:"api_key_id"`
	Endpoint     string    `json:"endpoint"`
	Method       string    `json:"method"`
	StatusCode   int       `json:"status_code"`
	ResponseTime int       `json:"response_time"` // milliseconds
	RequestSize  int       `json:"request_size"`  // bytes
	ResponseSize int       `json:"response_size"` // bytes
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	Country      string    `json:"country"`
	Timestamp    time.Time `json:"timestamp"`
}

// TimePeriod represents different time periods for statistics
type TimePeriod string

const (
	TimePeriodHour  TimePeriod = "hour"
	TimePeriodDay   TimePeriod = "day"
	TimePeriodWeek  TimePeriod = "week"
	TimePeriodMonth TimePeriod = "month"
	TimePeriodYear  TimePeriod = "year"
)

// UsageStatistics contains aggregated usage statistics
type UsageStatistics struct {
	Period              TimePeriod `json:"period"`
	StartTime           time.Time  `json:"start_time"`
	EndTime             time.Time  `json:"end_time"`
	TotalRequests       int64      `json:"total_requests"`
	SuccessfulRequests  int64      `json:"successful_requests"`
	FailedRequests      int64      `json:"failed_requests"`
	RateLimitedRequests int64      `json:"rate_limited_requests"`
	AvgResponseTime     float64    `json:"avg_response_time"`
	TotalBandwidth      int64      `json:"total_bandwidth"`
	UniqueEndpoints     int64      `json:"unique_endpoints"`
	TopStatusCodes      map[int]int64 `json:"top_status_codes"`
	RequestsByCountry   map[string]int64 `json:"requests_by_country"`
	ErrorRate           float64    `json:"error_rate"`
	SuccessRate         float64    `json:"success_rate"`
}

// EndpointUsageStats contains usage statistics for an endpoint
type EndpointUsageStats struct {
	Endpoint        string  `json:"endpoint"`
	Method          string  `json:"method"`
	TotalRequests   int64   `json:"total_requests"`
	AvgResponseTime float64 `json:"avg_response_time"`
	ErrorRate       float64 `json:"error_rate"`
	TotalBandwidth  int64   `json:"total_bandwidth"`
}

// HourlyUsageStats contains hourly usage statistics
type HourlyUsageStats struct {
	Hour            time.Time `json:"hour"`
	TotalRequests   int64     `json:"total_requests"`
	AvgResponseTime float64   `json:"avg_response_time"`
	ErrorCount      int64     `json:"error_count"`
	Bandwidth       int64     `json:"bandwidth"`
}

// UsageTrends contains trend analysis data
type UsageTrends struct {
	Period           TimePeriod        `json:"period"`
	GrowthRate       float64           `json:"growth_rate"` // percentage
	PeakUsageHour    int               `json:"peak_usage_hour"`
	AverageDaily     float64           `json:"average_daily"`
	WeekdayPattern   map[string]int64  `json:"weekday_pattern"`
	MonthlyGrowth    []MonthlyGrowth   `json:"monthly_growth"`
	TopGrowingEndpoints []EndpointGrowth `json:"top_growing_endpoints"`
}

// MonthlyGrowth represents monthly growth data
type MonthlyGrowth struct {
	Month    string  `json:"month"`
	Requests int64   `json:"requests"`
	Growth   float64 `json:"growth_rate"`
}

// EndpointGrowth represents endpoint growth data
type EndpointGrowth struct {
	Endpoint   string  `json:"endpoint"`
	Method     string  `json:"method"`
	Growth     float64 `json:"growth_rate"`
	Current    int64   `json:"current_requests"`
	Previous   int64   `json:"previous_requests"`
}

// ExportRequest contains parameters for exporting usage data
type ExportRequest struct {
	APIKeyID  uuid.UUID `json:"api_key_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Format    string    `json:"format"` // csv, json, xlsx
	Filters   map[string]interface{} `json:"filters"`
}

// ExportResult contains the result of a data export
type ExportResult struct {
	FileName    string    `json:"file_name"`
	Format      string    `json:"format"`
	RecordCount int64     `json:"record_count"`
	FileSize    int64     `json:"file_size"`
	GeneratedAt time.Time `json:"generated_at"`
	DownloadURL string    `json:"download_url"`
}

// usageTrackingService implements UsageTrackingService interface
type usageTrackingService struct {
	usageRepo   repositories.UsageLogRepository
	apiKeyRepo  repositories.APIKeyRepository
	cacheService CacheService
}

// NewUsageTrackingService creates a new usage tracking service
func NewUsageTrackingService(
	usageRepo repositories.UsageLogRepository,
	apiKeyRepo repositories.APIKeyRepository,
	cacheService CacheService,
) UsageTrackingService {
	return &usageTrackingService{
		usageRepo:   usageRepo,
		apiKeyRepo:  apiKeyRepo,
		cacheService: cacheService,
	}
}

// LogUsage logs a single API usage record
func (s *usageTrackingService) LogUsage(ctx context.Context, req *UsageLogRequest) error {
	// Validate API key exists
	if _, err := s.apiKeyRepo.GetByID(ctx, req.APIKeyID); err != nil {
		return fmt.Errorf("invalid api key: %w", err)
	}

	// Create usage log entry
	usageLog := &models.UsageLog{
		APIKeyID:     req.APIKeyID,
		Endpoint:     req.Endpoint,
		Method:       req.Method,
		StatusCode:   req.StatusCode,
		ResponseTime: req.ResponseTime,
		RequestSize:  int64(req.RequestSize),
		ResponseSize: int64(req.ResponseSize),
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		Country:      req.Country,
		Timestamp:    req.Timestamp,
	}

	if usageLog.Timestamp.IsZero() {
		usageLog.Timestamp = time.Now()
	}

	// Create in repository
	if err := s.usageRepo.Create(ctx, usageLog); err != nil {
		return fmt.Errorf("failed to create usage log: %w", err)
	}

	// Update API key usage counter asynchronously
	go func() {
		s.apiKeyRepo.UpdateLastUsed(context.Background(), req.APIKeyID)
		s.apiKeyRepo.IncrementUsage(context.Background(), req.APIKeyID, 1)
	}()

	return nil
}

// BatchLogUsage logs multiple API usage records efficiently
func (s *usageTrackingService) BatchLogUsage(ctx context.Context, requests []*UsageLogRequest) error {
	if len(requests) == 0 {
		return nil
	}

	// Convert requests to usage logs
	usageLogs := make([]*models.UsageLog, len(requests))
	apiKeyUpdates := make(map[uuid.UUID]int64)

	for i, req := range requests {
		// Validate API key exists (cache this check)
		cacheKey := fmt.Sprintf("api_key_exists:%s", req.APIKeyID.String())
		exists, err := s.cacheService.Exists(ctx, cacheKey)
		if err != nil || !exists {
			if _, err := s.apiKeyRepo.GetByID(ctx, req.APIKeyID); err != nil {
				return fmt.Errorf("invalid api key %s: %w", req.APIKeyID, err)
			}
			// Cache the existence for 5 minutes
			s.cacheService.Set(ctx, cacheKey, "1", 5*time.Minute)
		}

		usageLogs[i] = &models.UsageLog{
			APIKeyID:     req.APIKeyID,
			Endpoint:     req.Endpoint,
			Method:       req.Method,
			StatusCode:   req.StatusCode,
			ResponseTime: req.ResponseTime,
			RequestSize:  int64(req.RequestSize),
			ResponseSize: int64(req.ResponseSize),
			IPAddress:    req.IPAddress,
			UserAgent:    req.UserAgent,
			Country:      req.Country,
			Timestamp:    req.Timestamp,
		}

		if usageLogs[i].Timestamp.IsZero() {
			usageLogs[i].Timestamp = time.Now()
		}

		// Track API key usage counts
		apiKeyUpdates[req.APIKeyID]++
	}

	// Batch create usage logs
	if err := s.usageRepo.BatchCreate(ctx, usageLogs); err != nil {
		return fmt.Errorf("failed to batch create usage logs: %w", err)
	}

	// Update API key usage counters asynchronously
	go func() {
		for apiKeyID, count := range apiKeyUpdates {
			s.apiKeyRepo.UpdateLastUsed(context.Background(), apiKeyID)
			s.apiKeyRepo.IncrementUsage(context.Background(), apiKeyID, count)
		}
	}()

	return nil
}

// GetUsageStats retrieves aggregated usage statistics for a time period
func (s *usageTrackingService) GetUsageStats(ctx context.Context, apiKeyID uuid.UUID, period TimePeriod) (*UsageStatistics, error) {
	startTime, endTime := s.getTimePeriodBounds(period)

	// Get basic usage stats
	stats, err := s.usageRepo.GetUsageStats(ctx, apiKeyID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	// Get additional statistics
	logs, err := s.usageRepo.GetUsageByAPIKey(ctx, apiKeyID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage logs: %w", err)
	}

	// Analyze logs for additional metrics
	statusCodes := make(map[int]int64)
	countries := make(map[string]int64)
	endpointsSet := make(map[string]bool)

	for _, log := range logs {
		statusCodes[log.StatusCode]++
		if log.Country != "" {
			countries[log.Country]++
		}
		endpointsSet[log.Endpoint+":"+log.Method] = true
	}

	// Calculate rates
	var errorRate, successRate float64
	if stats.TotalRequests > 0 {
		errorRate = (float64(stats.FailedRequests) / float64(stats.TotalRequests)) * 100
		successRate = (float64(stats.SuccessfulRequests) / float64(stats.TotalRequests)) * 100
	}

	return &UsageStatistics{
		Period:              period,
		StartTime:           startTime,
		EndTime:             endTime,
		TotalRequests:       stats.TotalRequests,
		SuccessfulRequests:  stats.SuccessfulRequests,
		FailedRequests:      stats.FailedRequests,
		RateLimitedRequests: stats.RateLimitedRequests,
		AvgResponseTime:     stats.AvgResponseTime,
		TotalBandwidth:      stats.TotalBandwidth,
		UniqueEndpoints:     int64(len(endpointsSet)),
		TopStatusCodes:      statusCodes,
		RequestsByCountry:   countries,
		ErrorRate:           errorRate,
		SuccessRate:         successRate,
	}, nil
}

// GetUsageHistory retrieves historical usage logs
func (s *usageTrackingService) GetUsageHistory(ctx context.Context, apiKeyID uuid.UUID, startTime, endTime time.Time) ([]*models.UsageLog, error) {
	return s.usageRepo.GetUsageByAPIKey(ctx, apiKeyID, startTime, endTime)
}

// GetTopEndpoints retrieves top endpoints by usage
func (s *usageTrackingService) GetTopEndpoints(ctx context.Context, apiKeyID uuid.UUID, period TimePeriod, limit int) ([]*EndpointUsageStats, error) {
	startTime, endTime := s.getTimePeriodBounds(period)

	// Get endpoint statistics
	endpointStats, err := s.usageRepo.GetEndpointStats(ctx, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint stats: %w", err)
	}

	// Convert to our format and limit results
	results := make([]*EndpointUsageStats, 0, min(limit, len(endpointStats)))
	for i, stat := range endpointStats {
		if i >= limit {
			break
		}
		results = append(results, &EndpointUsageStats{
			Endpoint:        stat.Endpoint,
			Method:          stat.Method,
			TotalRequests:   stat.TotalRequests,
			AvgResponseTime: stat.AvgResponseTime,
			ErrorRate:       stat.ErrorRate,
			TotalBandwidth:  0, // Would need additional query to get bandwidth per endpoint
		})
	}

	return results, nil
}

// GetHourlyUsage retrieves hourly usage statistics
func (s *usageTrackingService) GetHourlyUsage(ctx context.Context, apiKeyID uuid.UUID, period TimePeriod) ([]*HourlyUsageStats, error) {
	startTime, endTime := s.getTimePeriodBounds(period)

	// Get hourly usage data
	hourlyData, err := s.usageRepo.GetHourlyUsage(ctx, apiKeyID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get hourly usage: %w", err)
	}

	// Convert to our format
	results := make([]*HourlyUsageStats, len(hourlyData))
	for i, data := range hourlyData {
		results[i] = &HourlyUsageStats{
			Hour:            data.Hour,
			TotalRequests:   data.TotalRequests,
			AvgResponseTime: data.AvgResponseTime,
			ErrorCount:      0, // Would need additional query
			Bandwidth:       0, // Would need additional query
		}
	}

	return results, nil
}

// GetUsageTrends retrieves usage trend analysis
func (s *usageTrackingService) GetUsageTrends(ctx context.Context, apiKeyID uuid.UUID, period TimePeriod) (*UsageTrends, error) {
	// This is a simplified implementation
	// In production, you'd want more sophisticated trend analysis
	
	currentStart, currentEnd := s.getTimePeriodBounds(period)
	var previousStart, previousEnd time.Time

	// Calculate previous period
	duration := currentEnd.Sub(currentStart)
	previousEnd = currentStart
	previousStart = previousEnd.Add(-duration)

	// Get current period stats
	currentStats, err := s.usageRepo.GetUsageStats(ctx, apiKeyID, currentStart, currentEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get current stats: %w", err)
	}

	// Get previous period stats
	previousStats, err := s.usageRepo.GetUsageStats(ctx, apiKeyID, previousStart, previousEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous stats: %w", err)
	}

	// Calculate growth rate
	var growthRate float64
	if previousStats.TotalRequests > 0 {
		growthRate = ((float64(currentStats.TotalRequests) - float64(previousStats.TotalRequests)) / float64(previousStats.TotalRequests)) * 100
	}

	// Get hourly usage to find peak hour
	hourlyData, err := s.GetHourlyUsage(ctx, apiKeyID, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get hourly usage: %w", err)
	}

	// Find peak usage hour
	var peakHour int
	var maxRequests int64
	for _, hourData := range hourlyData {
		if hourData.TotalRequests > maxRequests {
			maxRequests = hourData.TotalRequests
			peakHour = hourData.Hour.Hour()
		}
	}

	return &UsageTrends{
		Period:           period,
		GrowthRate:       growthRate,
		PeakUsageHour:    peakHour,
		AverageDaily:     float64(currentStats.TotalRequests) / float64(currentEnd.Sub(currentStart).Hours()/24),
		WeekdayPattern:   make(map[string]int64), // Would need additional implementation
		MonthlyGrowth:    []MonthlyGrowth{},      // Would need additional implementation
		TopGrowingEndpoints: []EndpointGrowth{},  // Would need additional implementation
	}, nil
}

// CleanupOldLogs removes old usage logs based on retention policy
func (s *usageTrackingService) CleanupOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	return s.usageRepo.DeleteOldLogs(ctx, retentionDays)
}

// ExportUsageData exports usage data in the requested format
func (s *usageTrackingService) ExportUsageData(ctx context.Context, req *ExportRequest) (*ExportResult, error) {
	// This is a placeholder implementation
	// In production, you'd implement actual file generation and storage
	
	// Get usage logs for the period
	logs, err := s.usageRepo.GetUsageByAPIKey(ctx, req.APIKeyID, req.StartTime, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage logs for export: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("usage_export_%s_%s.%s", 
		req.APIKeyID.String()[:8], 
		time.Now().Format("20060102_150405"), 
		req.Format)

	return &ExportResult{
		FileName:    filename,
		Format:      req.Format,
		RecordCount: int64(len(logs)),
		FileSize:    0, // Would calculate based on actual file
		GeneratedAt: time.Now(),
		DownloadURL: fmt.Sprintf("/api/exports/%s", filename),
	}, nil
}

// getTimePeriodBounds calculates start and end times for a given period
func (s *usageTrackingService) getTimePeriodBounds(period TimePeriod) (time.Time, time.Time) {
	now := time.Now()
	
	switch period {
	case TimePeriodHour:
		start := now.Add(-1 * time.Hour)
		return start, now
	case TimePeriodDay:
		start := now.AddDate(0, 0, -1)
		return start, now
	case TimePeriodWeek:
		start := now.AddDate(0, 0, -7)
		return start, now
	case TimePeriodMonth:
		start := now.AddDate(0, -1, 0)
		return start, now
	case TimePeriodYear:
		start := now.AddDate(-1, 0, 0)
		return start, now
	default:
		// Default to day
		start := now.AddDate(0, 0, -1)
		return start, now
	}
}