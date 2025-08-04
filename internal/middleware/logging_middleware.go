package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggingMiddleware creates a logging middleware with structured logging
func LoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Custom logging logic using zap
		fields := []zap.Field{
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("client_ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
		}

		if param.ErrorMessage != "" {
			fields = append(fields, zap.String("error", param.ErrorMessage))
		}

		// Log based on status code
		switch {
		case param.StatusCode >= 400 && param.StatusCode < 500:
			logger.Warn("Client error", fields...)
		case param.StatusCode >= 500:
			logger.Error("Server error", fields...)
		default:
			logger.Info("Request completed", fields...)
		}

		return ""
	})
}

// RequestResponseLoggingMiddleware logs detailed request and response information
func RequestResponseLoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Read and restore request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Capture response
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()

		// Log request and response details
		latency := time.Since(start)
		
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("request_size", len(requestBody)),
			zap.Int("response_size", blw.body.Len()),
		}

		// Add API key ID if available
		if apiKeyID, exists := c.Get("api_key_id"); exists {
			fields = append(fields, zap.Any("api_key_id", apiKeyID))
		}

		// Add request body for non-GET requests (be careful with sensitive data)
		if c.Request.Method != "GET" && len(requestBody) > 0 && len(requestBody) < 1024 {
			// Only log small request bodies and avoid sensitive endpoints
			if !isSensitiveEndpoint(c.Request.URL.Path) {
				fields = append(fields, zap.String("request_body", string(requestBody)))
			}
		}

		// Log based on status and latency
		switch {
		case c.Writer.Status() >= 500:
			logger.Error("Server error", fields...)
		case c.Writer.Status() >= 400:
			logger.Warn("Client error", fields...)
		case latency > time.Second:
			logger.Warn("Slow request", fields...)
		default:
			logger.Info("Request processed", fields...)
		}
	}
}

// bodyLogWriter captures response body for logging
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// isSensitiveEndpoint checks if an endpoint contains sensitive data
func isSensitiveEndpoint(path string) bool {
	sensitivePatterns := []string{
		"/api-keys",
		"/auth",
		"/login",
		"/password",
		"/token",
	}

	for _, pattern := range sensitivePatterns {
		if len(path) >= len(pattern) && path[:len(pattern)] == pattern {
			return true
		}
	}
	return false
}