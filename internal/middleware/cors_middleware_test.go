package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("sets CORS headers for regular request", func(t *testing.T) {
		router := gin.New()
		router.Use(CORSMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "test"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Origin, Content-Type, Accept, Authorization, X-Requested-With", w.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("handles OPTIONS preflight request", func(t *testing.T) {
		router := gin.New()
		router.Use(CORSMiddleware())
		router.POST("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "test"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Origin, Content-Type, Accept, Authorization, X-Requested-With", w.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("returns 204 for OPTIONS without proceeding to handler", func(t *testing.T) {
		handlerCalled := false
		
		router := gin.New()
		router.Use(CORSMiddleware())
		router.OPTIONS("/test", func(c *gin.Context) {
			handlerCalled = true
			c.JSON(200, gin.H{"message": "should not be called"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("OPTIONS", "/test", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.False(t, handlerCalled, "Handler should not be called for OPTIONS request")
	})
}