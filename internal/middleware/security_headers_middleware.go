package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Allow Swagger UI resources for /swagger paths
		if c.Request.URL.Path == "/swagger/" || c.Request.URL.Path == "/swagger/index" {
			c.Header("Content-Security-Policy", "default-src 'self' 'unsafe-inline' https://unpkg.com; script-src 'self' 'unsafe-inline' https://unpkg.com; style-src 'self' 'unsafe-inline' https://unpkg.com; img-src 'self' data: https:;")
		} else {
			c.Header("Content-Security-Policy", "default-src 'self'")
		}
		
		// Rate limiter specific headers
		c.Header("X-Rate-Limiter", "Viva/1.0")
		
		c.Next()
	}
}