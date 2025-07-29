package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RateLimitViolation represents a rate limit violation event
type RateLimitViolation struct {
	ID           uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	EventID      uuid.UUID `json:"event_id" gorm:"type:uuid;uniqueIndex;not null"`
	APIKeyID     uuid.UUID `json:"api_key_id" gorm:"type:uuid;not null;index"`
	
	// Request details
	Endpoint   string `json:"endpoint" gorm:"not null;size:255;index"`
	Method     string `json:"method" gorm:"not null;size:10"`
	ClientIP   string `json:"client_ip" gorm:"size:45;index"`
	UserAgent  string `json:"user_agent" gorm:"size:500"`
	
	// Rate limit details
	LimitValue     int    `json:"limit_value" gorm:"not null"`
	WindowSeconds  int    `json:"window_seconds" gorm:"not null"`
	CurrentCount   int    `json:"current_count" gorm:"not null"`
	TierType       string `json:"tier_type" gorm:"size:20"`
	
	// Violation context
	IsRepeated     bool   `json:"is_repeated" gorm:"default:false;index"`
	ViolationCount int    `json:"violation_count" gorm:"default:1"`
	
	// Geographic information
	Country string `json:"country" gorm:"size:2"`
	Region  string `json:"region" gorm:"size:100"`
	
	// Processing status
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	
	// Timestamps
	Timestamp time.Time `json:"timestamp" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at"`
	
	// Relations
	APIKey APIKey `json:"-" gorm:"foreignKey:APIKeyID"`
}

// TableName returns the table name for RateLimitViolation
func (RateLimitViolation) TableName() string {
	return "rate_limit_violations"
}

// BeforeCreate is called before creating a rate limit violation
func (r *RateLimitViolation) BeforeCreate(tx *gorm.DB) error {
	if r.EventID == uuid.Nil {
		r.EventID = uuid.New()
	}
	if r.Timestamp.IsZero() {
		r.Timestamp = time.Now()
	}
	return nil
}

// MarkProcessed marks the violation as processed
func (r *RateLimitViolation) MarkProcessed() {
	now := time.Now()
	r.ProcessedAt = &now
}

// IsProcessed returns true if the violation has been processed
func (r *RateLimitViolation) IsProcessed() bool {
	return r.ProcessedAt != nil
}

// GetExcessRequests returns how many requests exceeded the limit
func (r *RateLimitViolation) GetExcessRequests() int {
	return r.CurrentCount - r.LimitValue
}

// GetViolationSeverity returns the severity based on how much the limit was exceeded
func (r *RateLimitViolation) GetViolationSeverity() string {
	excess := r.GetExcessRequests()
	ratio := float64(excess) / float64(r.LimitValue)
	
	switch {
	case ratio >= 5.0:
		return "critical"
	case ratio >= 2.0:
		return "high"
	case ratio >= 0.5:
		return "medium"
	default:
		return "low"
	}
}

// GetWindowDuration returns the rate limit window as duration
func (r *RateLimitViolation) GetWindowDuration() time.Duration {
	return time.Duration(r.WindowSeconds) * time.Second
}