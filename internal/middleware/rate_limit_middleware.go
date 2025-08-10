package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rdhawladar/viva-rate-limiter/internal/services"
)

// RateLimitMiddleware creates a middleware for rate limiting
func RateLimitMiddleware(
	apiKeyService services.APIKeyService,
	rateLimitService services.RateLimitService,
	usageService services.UsageTrackingService,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Skip rate limiting for health endpoints
		if strings.HasPrefix(c.Request.URL.Path, "/health") ||
			strings.HasPrefix(c.Request.URL.Path, "/ready") ||
			strings.HasPrefix(c.Request.URL.Path, "/live") {
			c.Next()
			return
		}

		// Extract API key from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Missing API key",
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Validate Bearer token format
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid authorization format",
				"message": "Authorization header must be in format 'Bearer API_KEY'",
			})
			c.Abort()
			return
		}

		apiKey := authHeader[7:]

		// Validate API key
		validatedKey, err := apiKeyService.ValidateAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid API key",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// Check rate limit
		rateLimitReq := &services.RateLimitRequest{
			APIKeyID:  validatedKey.ID,
			Endpoint:  c.Request.URL.Path,
			Method:    c.Request.Method,
			IPAddress: c.ClientIP(),
			UserAgent: c.GetHeader("User-Agent"),
			Country:   c.GetHeader("X-Country"), // Assuming you have geolocation middleware
			Timestamp: time.Now(),
		}

		rateLimitResult, err := rateLimitService.CheckRateLimit(c.Request.Context(), rateLimitReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Rate limit check failed",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", string(rune(rateLimitResult.Limit)))
		c.Header("X-RateLimit-Remaining", string(rune(rateLimitResult.Remaining)))
		c.Header("X-RateLimit-Reset", rateLimitResult.ResetTime.Format(time.RFC3339))

		// Check if rate limit exceeded
		if !rateLimitResult.Allowed {
			c.Header("Retry-After", string(rune(rateLimitResult.RetryAfter)))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":             "Rate limit exceeded",
				"message":           "Too many requests. Please try again later.",
				"limit":             rateLimitResult.Limit,
				"remaining":         rateLimitResult.Remaining,
				"reset_time":        rateLimitResult.ResetTime,
				"retry_after":       rateLimitResult.RetryAfter,
				"violation_recorded": rateLimitResult.ViolationRecorded,
			})
			c.Abort()
			return
		}

		// Store API key info in context for downstream use
		c.Set("api_key", validatedKey)
		c.Set("api_key_id", validatedKey.ID)

		// Continue to next handler
		c.Next()

		// Log usage after request completion
		responseTime := int(time.Since(startTime).Milliseconds())
		statusCode := c.Writer.Status()

		// Log usage asynchronously
		go func() {
			usageReq := &services.UsageLogRequest{
				APIKeyID:     validatedKey.ID,
				Endpoint:     c.Request.URL.Path,
				Method:       c.Request.Method,
				StatusCode:   statusCode,
				ResponseTime: responseTime,
				RequestSize:  int(c.Request.ContentLength),
				ResponseSize: c.Writer.Size(),
				IPAddress:    c.ClientIP(),
				UserAgent:    c.GetHeader("User-Agent"),
				Country:      c.GetHeader("X-Country"),
				Timestamp:    startTime,
			}

			if err := usageService.LogUsage(c.Request.Context(), usageReq); err != nil {
				// Log error but don't fail the request
				// In production, you'd want proper logging here
				// logger.Error("Failed to log usage", zap.Error(err))
			}
		}()
	}
}