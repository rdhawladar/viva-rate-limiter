package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rdhawladar/viva-rate-limiter/internal/metrics"
)

// MetricsMiddleware creates a Gin middleware that records Prometheus metrics
func MetricsMiddleware(prometheusMetrics *metrics.PrometheusMetrics) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Record start time
		start := time.Now()
		
		// Increment in-flight requests
		prometheusMetrics.IncHTTPRequestsInFlight()
		
		// Process request
		c.Next()
		
		// Calculate duration
		duration := time.Since(start).Seconds()
		
		// Decrement in-flight requests
		prometheusMetrics.DecHTTPRequestsInFlight()
		
		// Get request details
		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		statusCode := strconv.Itoa(c.Writer.Status())
		
		// Record metrics
		prometheusMetrics.RecordHTTPRequest(method, path, statusCode, duration)
		
		// Record API key usage if present
		if apiKeyID, exists := c.Get("api_key_id"); exists {
			if tier, tierExists := c.Get("api_key_tier"); tierExists {
				prometheusMetrics.RecordAPIKeyRequest(
					apiKeyID.(string),
					tier.(string),
					path,
					method,
				)
				
				// Record request/response size
				if c.Request.ContentLength > 0 {
					prometheusMetrics.RecordAPIKeyUsage(
						apiKeyID.(string),
						tier.(string),
						"request",
						float64(c.Request.ContentLength),
					)
				}
				
				if c.Writer.Size() > 0 {
					prometheusMetrics.RecordAPIKeyUsage(
						apiKeyID.(string),
						tier.(string),
						"response",
						float64(c.Writer.Size()),
					)
				}
			}
		}
	})
}