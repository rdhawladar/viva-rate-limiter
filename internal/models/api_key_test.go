package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAPIKey_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   APIKeyStatus
		expected bool
	}{
		{"active status", APIKeyStatusActive, true},
		{"suspended status", APIKeyStatusSuspended, false},
		{"revoked status", APIKeyStatusRevoked, false},
		{"expired status", APIKeyStatusExpired, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := &APIKey{
				Status: tt.status,
			}
			assert.Equal(t, tt.expected, apiKey.IsActive())
		})
	}
}

func TestAPIKey_CanMakeRequest(t *testing.T) {
	tests := []struct {
		name      string
		status    APIKeyStatus
		rateLimit int
		expected  bool
	}{
		{"active with positive rate limit", APIKeyStatusActive, 1000, true},
		{"active with zero rate limit", APIKeyStatusActive, 0, false},
		{"suspended with positive rate limit", APIKeyStatusSuspended, 1000, false},
		{"revoked with positive rate limit", APIKeyStatusRevoked, 1000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := &APIKey{
				Status:    tt.status,
				RateLimit: tt.rateLimit,
			}
			assert.Equal(t, tt.expected, apiKey.CanMakeRequest())
		})
	}
}

func TestAPIKey_GetRateLimitWindow(t *testing.T) {
	tests := []struct {
		name       string
		rateWindow int
		expected   time.Duration
	}{
		{"1 hour window", 3600, time.Hour},
		{"30 minute window", 1800, 30 * time.Minute},
		{"1 minute window", 60, time.Minute},
		{"zero window", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := &APIKey{
				RateWindow: tt.rateWindow,
			}
			assert.Equal(t, tt.expected, apiKey.GetRateLimitWindow())
		})
	}
}

func TestAPIKey_UpdateLastUsed(t *testing.T) {
	apiKey := &APIKey{}
	
	beforeUpdate := time.Now()
	apiKey.UpdateLastUsed()
	afterUpdate := time.Now()

	assert.NotNil(t, apiKey.LastUsedAt)
	assert.True(t, apiKey.LastUsedAt.After(beforeUpdate) || apiKey.LastUsedAt.Equal(beforeUpdate))
	assert.True(t, apiKey.LastUsedAt.Before(afterUpdate) || apiKey.LastUsedAt.Equal(afterUpdate))
}

func TestAPIKey_IncrementUsage(t *testing.T) {
	tests := []struct {
		name          string
		initialUsage  int64
		incrementBy   int64
		expectedUsage int64
	}{
		{"increment by 1", 10, 1, 11},
		{"increment by 100", 500, 100, 600},
		{"increment from zero", 0, 50, 50},
		{"increment by zero", 100, 0, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := &APIKey{
				TotalUsage: tt.initialUsage,
			}
			apiKey.IncrementUsage(tt.incrementBy)
			assert.Equal(t, tt.expectedUsage, apiKey.TotalUsage)
		})
	}
}

func TestAPIKey_BeforeCreate(t *testing.T) {
	t.Run("sets ID if not provided", func(t *testing.T) {
		apiKey := &APIKey{}
		
		err := apiKey.BeforeCreate(nil)
		
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, apiKey.ID)
	})

	t.Run("keeps existing ID if provided", func(t *testing.T) {
		existingID := uuid.New()
		apiKey := &APIKey{
			ID: existingID,
		}
		
		err := apiKey.BeforeCreate(nil)
		
		assert.NoError(t, err)
		assert.Equal(t, existingID, apiKey.ID)
	})
}

func TestAPIKey_TableName(t *testing.T) {
	apiKey := APIKey{}
	assert.Equal(t, "api_keys", apiKey.TableName())
}

func TestAPIKeyStatus_Constants(t *testing.T) {
	assert.Equal(t, APIKeyStatus("active"), APIKeyStatusActive)
	assert.Equal(t, APIKeyStatus("suspended"), APIKeyStatusSuspended)
	assert.Equal(t, APIKeyStatus("revoked"), APIKeyStatusRevoked)
	assert.Equal(t, APIKeyStatus("expired"), APIKeyStatusExpired)
}

func TestAPIKeyTier_Constants(t *testing.T) {
	assert.Equal(t, APIKeyTier("free"), APIKeyTierFree)
	assert.Equal(t, APIKeyTier("pro"), APIKeyTierPro)
	assert.Equal(t, APIKeyTier("enterprise"), APIKeyTierEnterprise)
	assert.Equal(t, APIKeyTier("all"), APIKeyTierAll)
}