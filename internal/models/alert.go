package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeUsageThreshold     AlertType = "usage_threshold"
	AlertTypeRateLimitExceeded  AlertType = "rate_limit_exceeded"
	AlertTypeBillingOverage     AlertType = "billing_overage"
	AlertTypeSecurityAlert      AlertType = "security_alert"
	AlertTypeSystemHealth       AlertType = "system_health"
)

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive    AlertStatus = "active"
	AlertStatusResolved  AlertStatus = "resolved"
	AlertStatusSuppressed AlertStatus = "suppressed"
)

// Alert represents an alert in the system
type Alert struct {
	ID          uuid.UUID     `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	APIKeyID    *uuid.UUID    `json:"api_key_id,omitempty" gorm:"type:uuid;index"`
	Type        AlertType     `json:"type" gorm:"type:varchar(50);not null;index"`
	Severity    AlertSeverity `json:"severity" gorm:"type:varchar(20);not null;index"`
	Status      AlertStatus   `json:"status" gorm:"type:varchar(20);not null;default:'active';index"`
	
	// Alert details
	Title       string `json:"title" gorm:"not null;size:255"`
	Message     string `json:"message" gorm:"not null;size:1000"`
	Description string `json:"description" gorm:"size:2000"`
	
	// Threshold information
	Threshold    *float64 `json:"threshold,omitempty"`
	CurrentValue *float64 `json:"current_value,omitempty"`
	Unit         string   `json:"unit" gorm:"size:50"`
	
	// Additional data
	Metadata map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	Tags     []string               `json:"tags" gorm:"type:text[]"`
	
	// Timestamps
	TriggeredAt *time.Time `json:"triggered_at,omitempty"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	
	// Relations
	APIKey *APIKey `json:"-" gorm:"foreignKey:APIKeyID"`
}

// TableName returns the table name for Alert
func (Alert) TableName() string {
	return "alerts"
}

// BeforeCreate is called before creating an alert
func (a *Alert) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.TriggeredAt == nil {
		now := time.Now()
		a.TriggeredAt = &now
	}
	return nil
}

// IsActive returns true if the alert is active
func (a *Alert) IsActive() bool {
	return a.Status == AlertStatusActive
}

// IsResolved returns true if the alert is resolved
func (a *Alert) IsResolved() bool {
	return a.Status == AlertStatusResolved
}

// Resolve marks the alert as resolved
func (a *Alert) Resolve() {
	a.Status = AlertStatusResolved
	now := time.Now()
	a.ResolvedAt = &now
}

// Suppress marks the alert as suppressed
func (a *Alert) Suppress() {
	a.Status = AlertStatusSuppressed
}

// GetDurationActive returns how long the alert has been active
func (a *Alert) GetDurationActive() time.Duration {
	if a.TriggeredAt == nil {
		return 0
	}
	
	end := time.Now()
	if a.ResolvedAt != nil {
		end = *a.ResolvedAt
	}
	
	return end.Sub(*a.TriggeredAt)
}

// IsCritical returns true if the alert is critical
func (a *Alert) IsCritical() bool {
	return a.Severity == AlertSeverityCritical
}