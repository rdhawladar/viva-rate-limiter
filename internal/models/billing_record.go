package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BillingPeriodStatus represents the status of a billing period
type BillingPeriodStatus string

const (
	BillingPeriodStatusActive    BillingPeriodStatus = "active"
	BillingPeriodStatusCompleted BillingPeriodStatus = "completed"
	BillingPeriodStatusProcessing BillingPeriodStatus = "processing"
)

// BillingRecord represents a billing record for an API key
type BillingRecord struct {
	ID          uuid.UUID           `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	APIKeyID    uuid.UUID           `json:"api_key_id" gorm:"type:uuid;not null;index"`
	
	// Billing period
	PeriodStart time.Time           `json:"period_start" gorm:"not null;index"`
	PeriodEnd   time.Time           `json:"period_end" gorm:"not null;index"`
	Status      BillingPeriodStatus `json:"status" gorm:"type:varchar(20);not null;default:'active'"`
	
	// Usage metrics
	TotalRequests    int64 `json:"total_requests" gorm:"not null;default:0"`
	SuccessRequests  int64 `json:"success_requests" gorm:"not null;default:0"`
	ErrorRequests    int64 `json:"error_requests" gorm:"not null;default:0"`
	OverageRequests  int64 `json:"overage_requests" gorm:"not null;default:0"`
	
	// Rate limiting metrics
	RateLimitHits    int64 `json:"rate_limit_hits" gorm:"not null;default:0"`
	
	// Bandwidth metrics
	TotalBandwidth   int64 `json:"total_bandwidth" gorm:"not null;default:0"` // in bytes
	
	// Cost calculation
	BaseAmount      float64 `json:"base_amount" gorm:"type:decimal(10,4);default:0"`
	OverageAmount   float64 `json:"overage_amount" gorm:"type:decimal(10,4);default:0"`
	TotalAmount     float64 `json:"total_amount" gorm:"type:decimal(10,4);default:0"`
	Currency        string  `json:"currency" gorm:"size:3;default:'USD'"`
	
	// Tier information
	TierAtStart string `json:"tier_at_start" gorm:"size:20"`
	TierAtEnd   string `json:"tier_at_end" gorm:"size:20"`
	
	// Additional data
	Metadata map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	
	// Audit fields
	CalculatedAt *time.Time `json:"calculated_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	
	// Relations
	APIKey APIKey `json:"-" gorm:"foreignKey:APIKeyID"`
}

// TableName returns the table name for BillingRecord
func (BillingRecord) TableName() string {
	return "billing_records"
}

// BeforeCreate is called before creating a billing record
func (b *BillingRecord) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// IsActive returns true if the billing period is active
func (b *BillingRecord) IsActive() bool {
	return b.Status == BillingPeriodStatusActive
}

// IsCompleted returns true if the billing period is completed
func (b *BillingRecord) IsCompleted() bool {
	return b.Status == BillingPeriodStatusCompleted
}

// MarkCompleted marks the billing record as completed
func (b *BillingRecord) MarkCompleted() {
	b.Status = BillingPeriodStatusCompleted
	now := time.Now()
	b.CalculatedAt = &now
}

// GetSuccessRate returns the success rate as a percentage
func (b *BillingRecord) GetSuccessRate() float64 {
	if b.TotalRequests == 0 {
		return 0
	}
	return float64(b.SuccessRequests) / float64(b.TotalRequests) * 100
}

// GetErrorRate returns the error rate as a percentage
func (b *BillingRecord) GetErrorRate() float64 {
	if b.TotalRequests == 0 {
		return 0
	}
	return float64(b.ErrorRequests) / float64(b.TotalRequests) * 100
}

// GetOveragePercentage returns the overage as a percentage of total requests
func (b *BillingRecord) GetOveragePercentage() float64 {
	if b.TotalRequests == 0 {
		return 0
	}
	return float64(b.OverageRequests) / float64(b.TotalRequests) * 100
}

// GetAverageBandwidthPerRequest returns average bandwidth per request in bytes
func (b *BillingRecord) GetAverageBandwidthPerRequest() float64 {
	if b.TotalRequests == 0 {
		return 0
	}
	return float64(b.TotalBandwidth) / float64(b.TotalRequests)
}

// GetPeriodDuration returns the duration of the billing period
func (b *BillingRecord) GetPeriodDuration() time.Duration {
	return b.PeriodEnd.Sub(b.PeriodStart)
}

// GetDaysInPeriod returns the number of days in the billing period
func (b *BillingRecord) GetDaysInPeriod() int {
	return int(b.GetPeriodDuration().Hours() / 24)
}

// CalculateTotalAmount calculates and updates the total amount
func (b *BillingRecord) CalculateTotalAmount() {
	b.TotalAmount = b.BaseAmount + b.OverageAmount
}