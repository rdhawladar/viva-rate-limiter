package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UsageLog represents a usage log entry
type UsageLog struct {
	ID           uint64                 `json:"id" gorm:"primaryKey;autoIncrement"`
	APIKeyID     uuid.UUID              `json:"api_key_id" gorm:"type:uuid;not null;index"`
	Endpoint     string                 `json:"endpoint" gorm:"not null;size:255;index"`
	Method       string                 `json:"method" gorm:"not null;size:10"`
	StatusCode   int                    `json:"status_code" gorm:"not null;index"`
	ResponseTime int                    `json:"response_time"` // in milliseconds
	UserAgent    string                 `json:"user_agent" gorm:"size:500"`
	IPAddress    string                 `json:"ip_address" gorm:"size:45;index"` // IPv6 max length
	Country      string                 `json:"country" gorm:"size:2"`           // ISO country code
	Region       string                 `json:"region" gorm:"size:100"`
	
	// Request details
	RequestSize  int64                  `json:"request_size"`  // in bytes
	ResponseSize int64                  `json:"response_size"` // in bytes
	
	// Additional metadata
	Metadata     map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	
	// Timestamp
	Timestamp time.Time `json:"timestamp" gorm:"not null;index"`
	
	// Relations
	APIKey APIKey `json:"-" gorm:"foreignKey:APIKeyID"`
}

// TableName returns the table name for UsageLog
func (UsageLog) TableName() string {
	return "usage_logs"
}

// BeforeCreate is called before creating a usage log
func (u *UsageLog) BeforeCreate(tx *gorm.DB) error {
	if u.Timestamp.IsZero() {
		u.Timestamp = time.Now()
	}
	return nil
}

// IsSuccessful returns true if the request was successful
func (u *UsageLog) IsSuccessful() bool {
	return u.StatusCode >= 200 && u.StatusCode < 300
}

// IsClientError returns true if the request resulted in a client error
func (u *UsageLog) IsClientError() bool {
	return u.StatusCode >= 400 && u.StatusCode < 500
}

// IsServerError returns true if the request resulted in a server error
func (u *UsageLog) IsServerError() bool {
	return u.StatusCode >= 500
}

// GetResponseTimeSeconds returns response time in seconds
func (u *UsageLog) GetResponseTimeSeconds() float64 {
	return float64(u.ResponseTime) / 1000.0
}