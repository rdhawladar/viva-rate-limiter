package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Simple test for UsageLog structure
func TestUsageLog_BasicFields(t *testing.T) {
	t.Run("creates usage log with basic fields", func(t *testing.T) {
		log := &UsageLog{
			StatusCode:   200,
			RequestSize:  1024,
			ResponseSize: 2048,
		}
		
		assert.Equal(t, 200, log.StatusCode)
		assert.Equal(t, int64(1024), log.RequestSize)
		assert.Equal(t, int64(2048), log.ResponseSize)
	})
}

// Test TableName methods
func TestUsageLog_TableName(t *testing.T) {
	log := UsageLog{}
	assert.Equal(t, "usage_logs", log.TableName())
}

func TestAlert_TableName(t *testing.T) {
	alert := Alert{}
	assert.Equal(t, "alerts", alert.TableName())
}

func TestRateLimitViolation_TableName(t *testing.T) {
	violation := RateLimitViolation{}
	assert.Equal(t, "rate_limit_violations", violation.TableName())
}

func TestBillingRecord_TableName(t *testing.T) {
	record := BillingRecord{}
	assert.Equal(t, "billing_records", record.TableName())
}