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

// APIKeyRepository defines the interface for API key data access
type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *models.APIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error)
	GetByHash(ctx context.Context, keyHash string) (*models.APIKey, error)
	Update(ctx context.Context, apiKey *models.APIKey) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *APIKeyFilter, pagination *PaginationParams) (*PaginatedResult, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	IncrementUsage(ctx context.Context, id uuid.UUID, count int64) error
	GetActiveByTier(ctx context.Context, tier models.APIKeyTier) ([]*models.APIKey, error)
	CountByStatus(ctx context.Context, status models.APIKeyStatus) (int64, error)
	GetExpiredKeys(ctx context.Context, expiryTime time.Time) ([]*models.APIKey, error)
	BatchUpdateStatus(ctx context.Context, ids []uuid.UUID, status models.APIKeyStatus) error
}

// APIKeyFilter contains filter parameters for API key queries
type APIKeyFilter struct {
	Status   *models.APIKeyStatus `json:"status"`
	Tier     *models.APIKeyTier   `json:"tier"`
	UserID   *uuid.UUID           `json:"user_id"`
	TeamID   *uuid.UUID           `json:"team_id"`
	Search   string               `json:"search"`
	Tags     []string             `json:"tags"`
	FromDate *time.Time           `json:"from_date"`
	ToDate   *time.Time           `json:"to_date"`
}

// apiKeyRepository implements APIKeyRepository interface
type apiKeyRepository struct {
	*baseRepository
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{
		baseRepository: NewBaseRepository(db),
	}
}

// Create creates a new API key
func (r *apiKeyRepository) Create(ctx context.Context, apiKey *models.APIKey) error {
	if err := r.db.WithContext(ctx).Create(apiKey).Error; err != nil {
		return fmt.Errorf("failed to create api key: %w", err)
	}
	return nil
}

// GetByID retrieves an API key by ID
func (r *apiKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("api key not found")
		}
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}
	return &apiKey, nil
}

// GetByHash retrieves an API key by its hash
func (r *apiKeyRepository) GetByHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := r.db.WithContext(ctx).Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("api key not found")
		}
		return nil, fmt.Errorf("failed to get api key by hash: %w", err)
	}
	return &apiKey, nil
}

// Update updates an API key
func (r *apiKeyRepository) Update(ctx context.Context, apiKey *models.APIKey) error {
	result := r.db.WithContext(ctx).Model(apiKey).Updates(apiKey)
	if result.Error != nil {
		return fmt.Errorf("failed to update api key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

// Delete soft deletes an API key
func (r *apiKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.APIKey{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete api key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

// List retrieves API keys with filtering and pagination
func (r *apiKeyRepository) List(ctx context.Context, filter *APIKeyFilter, pagination *PaginationParams) (*PaginatedResult, error) {
	query := r.db.WithContext(ctx).Model(&models.APIKey{})

	// Apply filters
	if filter != nil {
		if filter.Status != nil {
			query = query.Where("status = ?", *filter.Status)
		}
		if filter.Tier != nil {
			query = query.Where("tier = ?", *filter.Tier)
		}
		if filter.UserID != nil {
			query = query.Where("user_id = ?", *filter.UserID)
		}
		if filter.TeamID != nil {
			query = query.Where("team_id = ?", *filter.TeamID)
		}
		if filter.Search != "" {
			query = query.Where("name ILIKE ? OR description ILIKE ?", 
				"%"+filter.Search+"%", "%"+filter.Search+"%")
		}
		if len(filter.Tags) > 0 {
			query = query.Where("tags && ?", filter.Tags)
		}
		if filter.FromDate != nil {
			query = query.Where("created_at >= ?", *filter.FromDate)
		}
		if filter.ToDate != nil {
			query = query.Where("created_at <= ?", *filter.ToDate)
		}
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count api keys: %w", err)
	}

	// Apply pagination
	var apiKeys []models.APIKey
	if err := query.
		Order(pagination.GetOrderBy()).
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to list api keys: %w", err)
	}

	return NewPaginatedResult(apiKeys, total, pagination), nil
}

// UpdateLastUsed updates the last used timestamp for an API key
func (r *apiKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("id = ?", id).
		Update("last_used_at", now)
	
	if result.Error != nil {
		return fmt.Errorf("failed to update last used: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

// IncrementUsage increments the total usage counter for an API key
func (r *apiKeyRepository) IncrementUsage(ctx context.Context, id uuid.UUID, count int64) error {
	result := r.db.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("id = ?", id).
		UpdateColumn("total_usage", gorm.Expr("total_usage + ?", count))
	
	if result.Error != nil {
		return fmt.Errorf("failed to increment usage: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

// GetActiveByTier retrieves all active API keys for a specific tier
func (r *apiKeyRepository) GetActiveByTier(ctx context.Context, tier models.APIKeyTier) ([]*models.APIKey, error) {
	var apiKeys []*models.APIKey
	if err := r.db.WithContext(ctx).
		Where("tier = ? AND status = ?", tier, models.APIKeyStatusActive).
		Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get active keys by tier: %w", err)
	}
	return apiKeys, nil
}

// CountByStatus counts API keys by status
func (r *apiKeyRepository) CountByStatus(ctx context.Context, status models.APIKeyStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("status = ?", status).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count by status: %w", err)
	}
	return count, nil
}

// GetExpiredKeys retrieves API keys that haven't been used since the expiry time
func (r *apiKeyRepository) GetExpiredKeys(ctx context.Context, expiryTime time.Time) ([]*models.APIKey, error) {
	var apiKeys []*models.APIKey
	if err := r.db.WithContext(ctx).
		Where("last_used_at < ? OR (last_used_at IS NULL AND created_at < ?)", 
			expiryTime, expiryTime).
		Where("status = ?", models.APIKeyStatusActive).
		Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get expired keys: %w", err)
	}
	return apiKeys, nil
}

// BatchUpdateStatus updates the status of multiple API keys
func (r *apiKeyRepository) BatchUpdateStatus(ctx context.Context, ids []uuid.UUID, status models.APIKeyStatus) error {
	if len(ids) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("id IN ?", ids).
		Update("status", status)
	
	if result.Error != nil {
		return fmt.Errorf("failed to batch update status: %w", result.Error)
	}
	return nil
}