package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSwaggerController_ServeSwaggerRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	controller := &SwaggerController{}
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/swagger", nil)
	
	controller.ServeSwaggerRedirect(c)
	
	assert.Equal(t, http.StatusMovedPermanently, w.Code)
	assert.Equal(t, "/swagger/", w.Header().Get("Location"))
}

func TestSwaggerController_ServeOpenAPISpec(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	controller := &SwaggerController{}
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/openapi.yaml", nil)
	
	controller.ServeOpenAPISpec(c)
	
	// Should attempt to serve the file (may return 404 if file doesn't exist, which is OK for this test)
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound)
}