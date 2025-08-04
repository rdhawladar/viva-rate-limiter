package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/viva/rate-limiter/internal/models"
	"github.com/viva/rate-limiter/internal/repositories"
)

// RateLimitService defines the interface for rate limiting business logic
type RateLimitService interface {
	CheckRateLimit(ctx context.Context, req *RateLimitRequest) (*RateLimitResult, error)
	GetRateLimitInfo(ctx context.Context, apiKeyID uuid.UUID) (*RateLimitInfo, error)
	ResetRateLimit(ctx context.Context, apiKeyID uuid.UUID) error
	UpdateRateLimit(ctx context.Context, apiKeyID uuid.UUID, newLimit int) error
	GetViolationHistory(ctx context.Context, apiKeyID uuid.UUID, hours int) ([]*models.RateLimitViolation, error)
	RecordViolation(ctx context.Context, req *RateLimitRequest, attemptedRequests int) error
	GetCurrentWindowUsage(ctx context.Context, apiKeyID uuid.UUID) (int64, error)
}

// RateLimitRequest contains data for rate limit checks
type RateLimitRequest struct {
	APIKeyID  uuid.UUID `json:"api_key_id"`
	Endpoint  string    `json:"endpoint"`
	Method    string    `json:"method"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Country   string    `json:"country"`
	Timestamp time.Time `json:"timestamp"`
}

// RateLimitResult contains the result of a rate limit check
type RateLimitResult struct {
	Allowed           bool      `json:"allowed"`
	Limit             int       `json:"limit"`
	Remaining         int       `json:"remaining"`
	ResetTime         time.Time `json:"reset_time"`
	WindowStart       time.Time `json:"window_start"`
	WindowEnd         time.Time `json:"window_end"`
	RetryAfter        int       `json:"retry_after"` // seconds
	ViolationRecorded bool      `json:"violation_recorded"`
}

// RateLimitInfo contains detailed rate limit information
type RateLimitInfo struct {
	APIKeyID        uuid.UUID           `json:"api_key_id"`
	CurrentLimit    int                 `json:"current_limit"`
	WindowSizeMin   int                 `json:"window_size_minutes"`
	CurrentUsage    int64               `json:"current_usage"`
	WindowStart     time.Time           `json:"window_start"`
	WindowEnd       time.Time           `json:"window_end"`
	RecentViolations int64              `json:"recent_violations"`
	Tier            models.APIKeyTier   `json:"tier"`
	Status          models.APIKeyStatus `json:"status"`
}

// rateLimitService implements RateLimitService interface
type rateLimitService struct {
	apiKeyRepo    repositories.APIKeyRepository
	violationRepo repositories.RateLimitViolationRepository
	usageRepo     repositories.UsageLogRepository
	cacheService  CacheService // Redis-based cache service
	windowSize    time.Duration
}

// NewRateLimitService creates a new rate limit service
func NewRateLimitService(
	apiKeyRepo repositories.APIKeyRepository,
	violationRepo repositories.RateLimitViolationRepository,
	usageRepo repositories.UsageLogRepository,
	cacheService CacheService,
) RateLimitService {
	return &rateLimitService{
		apiKeyRepo:    apiKeyRepo,
		violationRepo: violationRepo,
		usageRepo:     usageRepo,
		cacheService:  cacheService,
		windowSize:    time.Hour, // 1 hour sliding window
	}
}

// CheckRateLimit performs a rate limit check using sliding window algorithm
func (s *rateLimitService) CheckRateLimit(ctx context.Context, req *RateLimitRequest) (*RateLimitResult, error) {
	// Get API key information
	apiKey, err := s.apiKeyRepo.GetByID(ctx, req.APIKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}

	// Check if API key is active
	if apiKey.Status != models.APIKeyStatusActive {
		return &RateLimitResult{
			Allowed:           false,
			Limit:             apiKey.RateLimit,
			Remaining:         0,
			ResetTime:         time.Now().Add(s.windowSize),
			WindowStart:       time.Now(),
			WindowEnd:         time.Now().Add(s.windowSize),
			RetryAfter:        int(s.windowSize.Seconds()),
			ViolationRecorded: false,
		}, nil
	}

	// Calculate window boundaries
	now := req.Timestamp
	if now.IsZero() {
		now = time.Now()
	}

	windowStart := now.Truncate(s.windowSize)
	windowEnd := windowStart.Add(s.windowSize)

	// Get current usage in the window using cache first, then fallback to database
	cacheKey := fmt.Sprintf("rate_limit:%s:%d", req.APIKeyID.String(), windowStart.Unix())
	currentUsage, err := s.cacheService.GetCounter(ctx, cacheKey)
	if err != nil {
		// Fallback to database query
		logs, err := s.usageRepo.GetUsageByAPIKey(ctx, req.APIKeyID, windowStart, windowEnd)
		if err != nil {
			return nil, fmt.Errorf("failed to get usage logs: %w", err)
		}
		currentUsage = int64(len(logs))

		// Update cache
		s.cacheService.SetCounter(ctx, cacheKey, currentUsage, s.windowSize)
	}

	// Check if limit is exceeded
	allowed := currentUsage < int64(apiKey.RateLimit)
	remaining := int64(apiKey.RateLimit) - currentUsage
	if remaining < 0 {
		remaining = 0
	}

	result := &RateLimitResult{
		Allowed:           allowed,
		Limit:             apiKey.RateLimit,
		Remaining:         int(remaining),
		ResetTime:         windowEnd,
		WindowStart:       windowStart,
		WindowEnd:         windowEnd,
		RetryAfter:        int(windowEnd.Sub(now).Seconds()),
		ViolationRecorded: false,
	}

	// If not allowed, record violation
	if !allowed {
		if err := s.RecordViolation(ctx, req, 1); err != nil {
			// Log error but don't fail the rate limit check
			fmt.Printf("Failed to record violation: %v\n", err)
		} else {
			result.ViolationRecorded = true
		}
	} else {
		// Increment usage counter in cache
		s.cacheService.IncrementCounter(ctx, cacheKey, 1, s.windowSize)
	}

	return result, nil
}

// GetRateLimitInfo retrieves detailed rate limit information
func (s *rateLimitService) GetRateLimitInfo(ctx context.Context, apiKeyID uuid.UUID) (*RateLimitInfo, error) {
	// Get API key
	apiKey, err := s.apiKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	windowStart := now.Truncate(s.windowSize)
	windowEnd := windowStart.Add(s.windowSize)

	// Get current usage
	currentUsage, err := s.GetCurrentWindowUsage(ctx, apiKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage: %w", err)
	}

	// Get recent violations (last 24 hours)
	recentViolations, err := s.violationRepo.CountRecentViolations(ctx, apiKeyID, 24*60) // 24 hours in minutes
	if err != nil {
		return nil, fmt.Errorf("failed to count recent violations: %w", err)
	}

	return &RateLimitInfo{
		APIKeyID:         apiKeyID,
		CurrentLimit:     apiKey.RateLimit,
		WindowSizeMin:    int(s.windowSize.Minutes()),
		CurrentUsage:     currentUsage,
		WindowStart:      windowStart,
		WindowEnd:        windowEnd,
		RecentViolations: recentViolations,
		Tier:             apiKey.Tier,
		Status:           apiKey.Status,
	}, nil
}

// ResetRateLimit resets the rate limit counter for an API key
func (s *rateLimitService) ResetRateLimit(ctx context.Context, apiKeyID uuid.UUID) error {
	now := time.Now()
	windowStart := now.Truncate(s.windowSize)
	cacheKey := fmt.Sprintf("rate_limit:%s:%d", apiKeyID.String(), windowStart.Unix())

	return s.cacheService.DeleteKey(ctx, cacheKey)
}

// UpdateRateLimit updates the rate limit for an API key
func (s *rateLimitService) UpdateRateLimit(ctx context.Context, apiKeyID uuid.UUID, newLimit int) error {
	// Get API key
	apiKey, err := s.apiKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return err
	}

	// Update rate limit
	apiKey.RateLimit = newLimit
	apiKey.UpdatedAt = time.Now()

	return s.apiKeyRepo.Update(ctx, apiKey)
}

// GetViolationHistory retrieves violation history for an API key
func (s *rateLimitService) GetViolationHistory(ctx context.Context, apiKeyID uuid.UUID, hours int) ([]*models.RateLimitViolation, error) {
	startTime := time.Now().Add(time.Duration(-hours) * time.Hour)
	endTime := time.Now()

	return s.violationRepo.GetByAPIKey(ctx, apiKeyID, startTime, endTime)
}

// RecordViolation records a rate limit violation
func (s *rateLimitService) RecordViolation(ctx context.Context, req *RateLimitRequest, attemptedRequests int) error {
	violation := &models.RateLimitViolation{
		APIKeyID:       req.APIKeyID,
		Endpoint:       req.Endpoint,
		Method:         req.Method,
		ClientIP:       req.IPAddress,
		UserAgent:      req.UserAgent,
		Country:        req.Country,
		ViolationCount: attemptedRequests,
		Timestamp:      req.Timestamp,
	}

	if violation.Timestamp.IsZero() {
		violation.Timestamp = time.Now()
	}

	return s.violationRepo.Create(ctx, violation)
}

// GetCurrentWindowUsage gets the current usage in the rate limit window
func (s *rateLimitService) GetCurrentWindowUsage(ctx context.Context, apiKeyID uuid.UUID) (int64, error) {
	now := time.Now()
	windowStart := now.Truncate(s.windowSize)
	windowEnd := windowStart.Add(s.windowSize)

	// Try cache first
	cacheKey := fmt.Sprintf("rate_limit:%s:%d", apiKeyID.String(), windowStart.Unix())
	currentUsage, err := s.cacheService.GetCounter(ctx, cacheKey)
	if err == nil {
		return currentUsage, nil
	}

	// Fallback to database
	logs, err := s.usageRepo.GetUsageByAPIKey(ctx, apiKeyID, windowStart, windowEnd)
	if err != nil {
		return 0, fmt.Errorf("failed to get usage logs: %w", err)
	}

	currentUsage = int64(len(logs))

	// Update cache for future requests
	s.cacheService.SetCounter(ctx, cacheKey, currentUsage, s.windowSize)

	return currentUsage, nil
}

// CacheService defines the interface for cache operations
type CacheService interface {
	GetCounter(ctx context.Context, key string) (int64, error)
	SetCounter(ctx context.Context, key string, value int64, expiration time.Duration) error
	IncrementCounter(ctx context.Context, key string, delta int64, expiration time.Duration) (int64, error)
	DeleteKey(ctx context.Context, key string) error
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}