package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test struct initialization and basic functionality
func TestCreateAPIKeyRequest_Validation(t *testing.T) {
	t.Run("creates valid request", func(t *testing.T) {
		request := &CreateAPIKeyRequest{
			Name: "Test API Key",
		}
		
		assert.NotNil(t, request)
		assert.Equal(t, "Test API Key", request.Name)
	})
}

func TestUpdateAPIKeyRequest_Validation(t *testing.T) {
	t.Run("creates valid update request", func(t *testing.T) {
		name := "Updated Name"
		request := &UpdateAPIKeyRequest{
			Name: &name,
		}
		
		assert.NotNil(t, request)
		assert.NotNil(t, request.Name)
		assert.Equal(t, "Updated Name", *request.Name)
	})
}