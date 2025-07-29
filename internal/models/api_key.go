package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// APIKeyStatus represents the status of an API key
type APIKeyStatus string

const (
	APIKeyStatusActive    APIKeyStatus = "active"
	APIKeyStatusSuspended APIKeyStatus = "suspended"
	APIKeyStatusRevoked   APIKeyStatus = "revoked"
)

// APIKeyTier represents the tier of an API key
type APIKeyTier string

const (
	APIKeyTierFree       APIKeyTier = "free"
	APIKeyTierPro        APIKeyTier = "pro"
	APIKeyTierEnterprise APIKeyTier = "enterprise"
)

// APIKey represents an API key in the system
type APIKey struct {
	ID          uuid.UUID    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	KeyHash     string       `json:"-" gorm:"uniqueIndex;not null;size:64"` // SHA256 hash of the actual key
	Name        string       `json:"name" gorm:"not null;size:255"`
	Description string       `json:"description" gorm:"size:500"`
	Tier        APIKeyTier   `json:"tier" gorm:"type:varchar(20);not null;default:'free'"`
	RateLimit   int          `json:"rate_limit" gorm:"not null;default:1000"`
	RateWindow  int          `json:"rate_window" gorm:"not null;default:3600"` // in seconds
	Status      APIKeyStatus `json:"status" gorm:"type:varchar(20);not null;default:'active'"`
	
	// Metadata
	Metadata   map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	Tags       []string               `json:"tags" gorm:"type:text[]"`
	
	// Ownership
	UserID     *uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;index"`
	TeamID     *uuid.UUID `json:"team_id,omitempty" gorm:"type:uuid;index"`
	
	// Usage tracking
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	TotalUsage int64      `json:"total_usage" gorm:"default:0"`
	
	// Audit fields
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relations
	UsageLogs []UsageLog `json:"-" gorm:"foreignKey:APIKeyID"`
	Alerts    []Alert    `json:"-" gorm:"foreignKey:APIKeyID"`
}

// TableName returns the table name for APIKey
func (APIKey) TableName() string {
	return "api_keys"
}

// BeforeCreate is called before creating an API key
func (a *APIKey) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// IsActive returns true if the API key is active
func (a *APIKey) IsActive() bool {
	return a.Status == APIKeyStatusActive
}

// CanMakeRequest returns true if the API key can make requests
func (a *APIKey) CanMakeRequest() bool {
	return a.IsActive() && a.RateLimit > 0
}

// GetRateLimitWindow returns the rate limit window as duration
func (a *APIKey) GetRateLimitWindow() time.Duration {
	return time.Duration(a.RateWindow) * time.Second
}

// UpdateLastUsed updates the last used timestamp
func (a *APIKey) UpdateLastUsed() {
	now := time.Now()
	a.LastUsedAt = &now
}

// IncrementUsage increments the total usage counter
func (a *APIKey) IncrementUsage(count int64) {
	a.TotalUsage += count
}