package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/viva/rate-limiter/internal/cache"
	"github.com/viva/rate-limiter/internal/models"
)

// HealthController handles health check endpoints
type HealthController struct {
	redisClient *cache.RedisClient
}

// NewHealthController creates a new health controller
func NewHealthController(redisClient *cache.RedisClient) *HealthController {
	return &HealthController{
		redisClient: redisClient,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Services  map[string]ServiceHealth `json:"services"`
}

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Status  string        `json:"status"`
	Latency time.Duration `json:"latency,omitempty"`
	Error   string        `json:"error,omitempty"`
}

// Health performs a basic health check
// @Summary Health check
// @Description Get the health status of the API
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *HealthController) Health(c *gin.Context) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0", // This should come from config
		Services:  make(map[string]ServiceHealth),
	}

	// Check database
	dbStart := time.Now()
	if err := models.HealthCheck(); err != nil {
		response.Services["database"] = ServiceHealth{
			Status:  "unhealthy",
			Latency: time.Since(dbStart),
			Error:   err.Error(),
		}
		response.Status = "unhealthy"
	} else {
		response.Services["database"] = ServiceHealth{
			Status:  "healthy",
			Latency: time.Since(dbStart),
		}
	}

	// Check Redis
	if h.redisClient != nil {
		redisStart := time.Now()
		if err := h.redisClient.Health(c.Request.Context()); err != nil {
			response.Services["redis"] = ServiceHealth{
				Status:  "unhealthy",
				Latency: time.Since(redisStart),
				Error:   err.Error(),
			}
			response.Status = "unhealthy"
		} else {
			response.Services["redis"] = ServiceHealth{
				Status:  "healthy",
				Latency: time.Since(redisStart),
			}
		}
	}

	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// Ready performs a readiness check
// @Summary Readiness check
// @Description Check if the API is ready to serve requests
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /ready [get]
func (h *HealthController) Ready(c *gin.Context) {
	// For now, ready check is the same as health check
	// In production, you might have different criteria
	h.Health(c)
}

// Live performs a liveness check
// @Summary Liveness check
// @Description Check if the API is alive
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /live [get]
func (h *HealthController) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now(),
	})
}