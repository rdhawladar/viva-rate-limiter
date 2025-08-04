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
	AlertTypeRateLimit          AlertType = "rate_limit"
	AlertTypeQuotaExceeded      AlertType = "quota_exceeded"
	AlertTypeAbnormalUsage      AlertType = "abnormal_usage"
	AlertTypeSystemError        AlertType = "system_error"
	AlertTypeSecurityIssue      AlertType = "security_issue"
)

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityInfo     AlertSeverity = "info"
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
	APIKeyID    uuid.UUID     `json:"api_key_id" gorm:"type:uuid;not null;index"`
	Type        AlertType     `json:"type" gorm:"type:varchar(50);not null;index"`
	Severity    AlertSeverity `json:"severity" gorm:"type:varchar(20);not null;index"`
	Message     string        `json:"message" gorm:"not null;size:1000"`
	
	// Additional data
	Metadata map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	
	// Resolution info
	Resolved   bool       `json:"resolved" gorm:"default:false;index"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy *string    `json:"resolved_by,omitempty" gorm:"size:255"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
	// Relations
	APIKey APIKey `json:"-" gorm:"foreignKey:APIKeyID"`
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
	return nil
}

// IsResolved returns true if the alert is resolved
func (a *Alert) IsResolved() bool {
	return a.Resolved
}

// Resolve marks the alert as resolved
func (a *Alert) Resolve(resolvedBy string) {
	a.Resolved = true
	now := time.Now()
	a.ResolvedAt = &now
	a.ResolvedBy = &resolvedBy
}

// GetDurationActive returns how long the alert has been active
func (a *Alert) GetDurationActive() time.Duration {
	end := time.Now()
	if a.ResolvedAt != nil {
		end = *a.ResolvedAt
	}
	
	return end.Sub(a.CreatedAt)
}

// IsCritical returns true if the alert is critical
func (a *Alert) IsCritical() bool {
	return a.Severity == AlertSeverityCritical
}