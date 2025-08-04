package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viva/rate-limiter/internal/models"
	"github.com/viva/rate-limiter/internal/repositories"
)

// APIKeyService defines the interface for API key business logic
type APIKeyService interface {
	CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*APIKeyResponse, error)
	GetAPIKey(ctx context.Context, id uuid.UUID) (*APIKeyResponse, error)
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKeyResponse, error)
	UpdateAPIKey(ctx context.Context, id uuid.UUID, req *UpdateAPIKeyRequest) (*APIKeyResponse, error)
	DeleteAPIKey(ctx context.Context, id uuid.UUID) error
	ListAPIKeys(ctx context.Context, filter *repositories.APIKeyFilter, pagination *repositories.PaginationParams) (*repositories.PaginatedResult, error)
	GenerateAPIKey(ctx context.Context) (string, string, error) // returns key, hash
	ValidateAPIKey(ctx context.Context, key string) (*models.APIKey, error)
	RotateAPIKey(ctx context.Context, id uuid.UUID) (*APIKeyResponse, error)
	GetAPIKeyStats(ctx context.Context, id uuid.UUID) (*APIKeyStats, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	IncrementUsage(ctx context.Context, id uuid.UUID, count int64) error
	GetActiveKeysByTier(ctx context.Context, tier models.APIKeyTier) ([]*models.APIKey, error)
	ExpireUnusedKeys(ctx context.Context, unusedDays int) (int, error)
}

// CreateAPIKeyRequest contains data for creating a new API key
type CreateAPIKeyRequest struct {
	Name         string            `json:"name" validate:"required,min=1,max=100"`
	Description  string            `json:"description" validate:"max=500"`
	Tier         models.APIKeyTier `json:"tier" validate:"required"`
	UserID       uuid.UUID         `json:"user_id" validate:"required"`
	TeamID       *uuid.UUID        `json:"team_id"`
	Tags         []string          `json:"tags"`
	ExpiresAt    *time.Time        `json:"expires_at"`
	RateLimit    int               `json:"rate_limit" validate:"min=1"`
	QuotaLimit   int64             `json:"quota_limit" validate:"min=1"`
}

// UpdateAPIKeyRequest contains data for updating an API key
type UpdateAPIKeyRequest struct {
	Name        *string           `json:"name" validate:"omitempty,min=1,max=100"`
	Description *string           `json:"description" validate:"omitempty,max=500"`
	Status      *models.APIKeyStatus `json:"status"`
	Tags        []string          `json:"tags"`
	ExpiresAt   *time.Time        `json:"expires_at"`
	RateLimit   *int              `json:"rate_limit" validate:"omitempty,min=1"`
	QuotaLimit  *int64            `json:"quota_limit" validate:"omitempty,min=1"`
}

// APIKeyResponse contains API key data for responses
type APIKeyResponse struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Status       models.APIKeyStatus `json:"status"`
	Tier         models.APIKeyTier `json:"tier"`
	UserID       uuid.UUID         `json:"user_id"`
	TeamID       *uuid.UUID        `json:"team_id"`
	Tags         []string          `json:"tags"`
	RateLimit    int               `json:"rate_limit"`
	QuotaLimit   int64             `json:"quota_limit"`
	TotalUsage   int64             `json:"total_usage"`
	Key          *string           `json:"key,omitempty"` // Only included on creation
	LastUsedAt   *time.Time        `json:"last_used_at"`
	ExpiresAt    *time.Time        `json:"expires_at"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// APIKeyStats contains usage statistics for an API key
type APIKeyStats struct {
	TotalRequests     int64   `json:"total_requests"`
	RequestsToday     int64   `json:"requests_today"`
	RequestsThisMonth int64   `json:"requests_this_month"`
	QuotaUsedPercent  float64 `json:"quota_used_percent"`
	RateLimitHits     int64   `json:"rate_limit_hits"`
	LastActivity      *time.Time `json:"last_activity"`
	TopEndpoints      []EndpointUsage `json:"top_endpoints"`
}

// EndpointUsage contains usage data for an endpoint
type EndpointUsage struct {
	Endpoint     string `json:"endpoint"`
	Method       string `json:"method"`
	RequestCount int64  `json:"request_count"`
}

// apiKeyService implements APIKeyService interface
type apiKeyService struct {
	apiKeyRepo repositories.APIKeyRepository
	usageRepo  repositories.UsageLogRepository
}

// NewAPIKeyService creates a new API key service
func NewAPIKeyService(
	apiKeyRepo repositories.APIKeyRepository,
	usageRepo repositories.UsageLogRepository,
) APIKeyService {
	return &apiKeyService{
		apiKeyRepo: apiKeyRepo,
		usageRepo:  usageRepo,
	}
}

// CreateAPIKey creates a new API key
func (s *apiKeyService) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*APIKeyResponse, error) {
	// Generate API key and hash
	key, keyHash, err := s.GenerateAPIKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate api key: %w", err)
	}

	// Create API key model
	apiKey := &models.APIKey{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		KeyHash:     keyHash,
		Status:      models.APIKeyStatusActive,
		Tier:        req.Tier,
		UserID:      &req.UserID,
		TeamID:      req.TeamID,
		Tags:        req.Tags,
		RateLimit:   req.RateLimit,
		QuotaLimit:  req.QuotaLimit,
		TotalUsage:  0,
		ExpiresAt:   req.ExpiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create in repository
	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}

	// Convert to response
	response := s.modelToResponse(apiKey)
	response.Key = &key // Include the actual key only on creation

	return response, nil
}

// GetAPIKey retrieves an API key by ID
func (s *apiKeyService) GetAPIKey(ctx context.Context, id uuid.UUID) (*APIKeyResponse, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.modelToResponse(apiKey), nil
}

// GetAPIKeyByHash retrieves an API key by its hash
func (s *apiKeyService) GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKeyResponse, error) {
	apiKey, err := s.apiKeyRepo.GetByHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	return s.modelToResponse(apiKey), nil
}

// UpdateAPIKey updates an existing API key
func (s *apiKeyService) UpdateAPIKey(ctx context.Context, id uuid.UUID, req *UpdateAPIKeyRequest) (*APIKeyResponse, error) {
	// Get existing API key
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Name != nil {
		apiKey.Name = *req.Name
	}
	if req.Description != nil {
		apiKey.Description = *req.Description
	}
	if req.Status != nil {
		apiKey.Status = *req.Status
	}
	if req.Tags != nil {
		apiKey.Tags = req.Tags
	}
	if req.ExpiresAt != nil {
		apiKey.ExpiresAt = req.ExpiresAt
	}
	if req.RateLimit != nil {
		apiKey.RateLimit = *req.RateLimit
	}
	if req.QuotaLimit != nil {
		apiKey.QuotaLimit = *req.QuotaLimit
	}

	apiKey.UpdatedAt = time.Now()

	// Update in repository
	if err := s.apiKeyRepo.Update(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to update api key: %w", err)
	}

	return s.modelToResponse(apiKey), nil
}

// DeleteAPIKey soft deletes an API key
func (s *apiKeyService) DeleteAPIKey(ctx context.Context, id uuid.UUID) error {
	return s.apiKeyRepo.Delete(ctx, id)
}

// ListAPIKeys retrieves API keys with filtering and pagination
func (s *apiKeyService) ListAPIKeys(ctx context.Context, filter *repositories.APIKeyFilter, pagination *repositories.PaginationParams) (*repositories.PaginatedResult, error) {
	if pagination == nil {
		pagination = repositories.DefaultPagination()
	}

	result, err := s.apiKeyRepo.List(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	// Convert models to responses
	apiKeys := result.Data.([]models.APIKey)
	responses := make([]*APIKeyResponse, len(apiKeys))
	for i, apiKey := range apiKeys {
		responses[i] = s.modelToResponse(&apiKey)
	}

	// Update result data
	result.Data = responses
	return result, nil
}

// GenerateAPIKey generates a new API key and its hash
func (s *apiKeyService) GenerateAPIKey(ctx context.Context) (string, string, error) {
	// Generate random bytes
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Create the key with prefix
	key := fmt.Sprintf("viva_%s", hex.EncodeToString(keyBytes))

	// Create hash for storage
	hash := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(hash[:])

	return key, keyHash, nil
}

// ValidateAPIKey validates an API key and returns the associated model
func (s *apiKeyService) ValidateAPIKey(ctx context.Context, key string) (*models.APIKey, error) {
	// Check key format
	if !strings.HasPrefix(key, "viva_") {
		return nil, fmt.Errorf("invalid api key format")
	}

	// Hash the key
	hash := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(hash[:])

	// Get API key by hash
	apiKey, err := s.apiKeyRepo.GetByHash(ctx, keyHash)
	if err != nil {
		return nil, fmt.Errorf("invalid api key")
	}

	// Check if key is active
	if apiKey.Status != models.APIKeyStatusActive {
		return nil, fmt.Errorf("api key is not active")
	}

	// Check expiration
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, fmt.Errorf("api key has expired")
	}

	return apiKey, nil
}

// RotateAPIKey generates a new key for an existing API key
func (s *apiKeyService) RotateAPIKey(ctx context.Context, id uuid.UUID) (*APIKeyResponse, error) {
	// Get existing API key
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Generate new key and hash
	newKey, newKeyHash, err := s.GenerateAPIKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new api key: %w", err)
	}

	// Update the key hash
	apiKey.KeyHash = newKeyHash
	apiKey.UpdatedAt = time.Now()

	if err := s.apiKeyRepo.Update(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to update api key: %w", err)
	}

	// Convert to response
	response := s.modelToResponse(apiKey)
	response.Key = &newKey // Include the new key

	return response, nil
}

// GetAPIKeyStats retrieves statistics for an API key
func (s *apiKeyService) GetAPIKeyStats(ctx context.Context, id uuid.UUID) (*APIKeyStats, error) {
	// Get API key
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Get usage statistics
	totalStats, err := s.usageRepo.GetUsageStats(ctx, id, time.Time{}, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get total usage stats: %w", err)
	}

	todayStats, err := s.usageRepo.GetUsageStats(ctx, id, todayStart, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get today usage stats: %w", err)
	}

	monthStats, err := s.usageRepo.GetUsageStats(ctx, id, monthStart, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get month usage stats: %w", err)
	}

	// Calculate quota used percentage
	var quotaUsedPercent float64
	if apiKey.QuotaLimit > 0 {
		quotaUsedPercent = (float64(apiKey.TotalUsage) / float64(apiKey.QuotaLimit)) * 100
	}

	// Get top endpoints (last 30 days)
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	endpointStats, err := s.usageRepo.GetEndpointStats(ctx, thirtyDaysAgo, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint stats: %w", err)
	}

	topEndpoints := make([]EndpointUsage, 0, min(5, len(endpointStats)))
	for i, stat := range endpointStats {
		if i >= 5 {
			break
		}
		topEndpoints = append(topEndpoints, EndpointUsage{
			Endpoint:     stat.Endpoint,
			Method:       stat.Method,
			RequestCount: stat.TotalRequests,
		})
	}

	return &APIKeyStats{
		TotalRequests:     totalStats.TotalRequests,
		RequestsToday:     todayStats.TotalRequests,
		RequestsThisMonth: monthStats.TotalRequests,
		QuotaUsedPercent:  quotaUsedPercent,
		RateLimitHits:     totalStats.RateLimitedRequests,
		LastActivity:      apiKey.LastUsedAt,
		TopEndpoints:      topEndpoints,
	}, nil
}

// UpdateLastUsed updates the last used timestamp for an API key
func (s *apiKeyService) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	return s.apiKeyRepo.UpdateLastUsed(ctx, id)
}

// IncrementUsage increments the usage counter for an API key
func (s *apiKeyService) IncrementUsage(ctx context.Context, id uuid.UUID, count int64) error {
	return s.apiKeyRepo.IncrementUsage(ctx, id, count)
}

// GetActiveKeysByTier retrieves active API keys for a specific tier
func (s *apiKeyService) GetActiveKeysByTier(ctx context.Context, tier models.APIKeyTier) ([]*models.APIKey, error) {
	return s.apiKeyRepo.GetActiveByTier(ctx, tier)
}

// ExpireUnusedKeys marks unused API keys as expired
func (s *apiKeyService) ExpireUnusedKeys(ctx context.Context, unusedDays int) (int, error) {
	expiryTime := time.Now().AddDate(0, 0, -unusedDays)
	
	expiredKeys, err := s.apiKeyRepo.GetExpiredKeys(ctx, expiryTime)
	if err != nil {
		return 0, fmt.Errorf("failed to get expired keys: %w", err)
	}

	if len(expiredKeys) == 0 {
		return 0, nil
	}

	var ids []uuid.UUID
	for _, key := range expiredKeys {
		ids = append(ids, key.ID)
	}

	if err := s.apiKeyRepo.BatchUpdateStatus(ctx, ids, models.APIKeyStatusExpired); err != nil {
		return 0, fmt.Errorf("failed to update expired keys: %w", err)
	}

	return len(expiredKeys), nil
}

// modelToResponse converts a models.APIKey to APIKeyResponse
func (s *apiKeyService) modelToResponse(apiKey *models.APIKey) *APIKeyResponse {
	var userID uuid.UUID
	if apiKey.UserID != nil {
		userID = *apiKey.UserID
	}
	
	return &APIKeyResponse{
		ID:          apiKey.ID,
		Name:        apiKey.Name,
		Description: apiKey.Description,
		Status:      apiKey.Status,
		Tier:        apiKey.Tier,
		UserID:      userID,
		TeamID:      apiKey.TeamID,
		Tags:        apiKey.Tags,
		RateLimit:   apiKey.RateLimit,
		QuotaLimit:  apiKey.QuotaLimit,
		TotalUsage:  apiKey.TotalUsage,
		LastUsedAt:  apiKey.LastUsedAt,
		ExpiresAt:   apiKey.ExpiresAt,
		CreatedAt:   apiKey.CreatedAt,
		UpdatedAt:   apiKey.UpdatedAt,
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}