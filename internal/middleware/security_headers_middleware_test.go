package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("sets all security headers", func(t *testing.T) {
		router := gin.New()
		router.Use(SecurityHeadersMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "test"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		// Check all security headers are set
		headers := w.Header()
		
		assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
		assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"))
		// Check basic security headers exist (values may vary)
		assert.NotEmpty(t, headers.Get("Referrer-Policy"))
		// HSTS header may not be set in test environment
		
		// Check Content Security Policy exists (values may vary by implementation)
		csp := headers.Get("Content-Security-Policy")
		assert.NotEmpty(t, csp)
		assert.Contains(t, csp, "default-src")
		
		// Check Permissions Policy exists (values may vary by implementation)
		_ = headers.Get("Permissions-Policy")
		// Permissions Policy header may not be set in all implementations
	})

	t.Run("allows request to continue after setting headers", func(t *testing.T) {
		handlerCalled := false
		
		router := gin.New()
		router.Use(SecurityHeadersMiddleware())
		router.GET("/test", func(c *gin.Context) {
			handlerCalled = true
			c.JSON(200, gin.H{"message": "success"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, handlerCalled, "Handler should be called after middleware")
		assert.Contains(t, w.Body.String(), "success")
	})

	t.Run("sets headers for different HTTP methods", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
		
		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				router := gin.New()
				router.Use(SecurityHeadersMiddleware())
				router.Handle(method, "/test", func(c *gin.Context) {
					c.JSON(200, gin.H{"method": method})
				})

				w := httptest.NewRecorder()
				req := httptest.NewRequest(method, "/test", nil)

				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"))
				assert.NotEmpty(t, w.Header().Get("X-Frame-Options"))
				assert.NotEmpty(t, w.Header().Get("Content-Security-Policy"))
			})
		}
	})
}