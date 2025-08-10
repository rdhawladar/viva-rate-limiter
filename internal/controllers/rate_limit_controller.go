package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rdhawladar/viva-rate-limiter/internal/models"
	"github.com/rdhawladar/viva-rate-limiter/internal/services"
)

// RateLimitController handles rate limiting endpoints
type RateLimitController struct {
	rateLimitService services.RateLimitService
	apiKeyService    services.APIKeyService
}

// NewRateLimitController creates a new rate limit controller
func NewRateLimitController(
	rateLimitService services.RateLimitService,
	apiKeyService services.APIKeyService,
) *RateLimitController {
	return &RateLimitController{
		rateLimitService: rateLimitService,
		apiKeyService:    apiKeyService,
	}
}

// CheckRateLimit checks if a request is within rate limits
// @Summary Check rate limit
// @Description Check if a request is within the rate limits for an API key
// @Tags rate-limit
// @Accept json
// @Produce json
// @Param request body RateLimitCheckRequest true "Rate limit check request"
// @Success 200 {object} services.RateLimitResult
// @Failure 400 {object} ErrorResponse
// @Failure 429 {object} RateLimitExceededResponse
// @Failure 500 {object} ErrorResponse
// @Router /rate-limit/check [post]
func (ctrl *RateLimitController) CheckRateLimit(c *gin.Context) {
	var req RateLimitCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &services.RateLimitRequest{
		APIKeyID:  req.APIKeyID,
		Endpoint:  req.Endpoint,
		Method:    req.Method,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		Country:   req.Country,
		Timestamp: time.Now(),
	}

	result, err := ctrl.rateLimitService.CheckRateLimit(c.Request.Context(), serviceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to check rate limit",
			Message: err.Error(),
		})
		return
	}

	// Set rate limit headers
	c.Header("X-RateLimit-Limit", string(rune(result.Limit)))
	c.Header("X-RateLimit-Remaining", string(rune(result.Remaining)))
	c.Header("X-RateLimit-Reset", result.ResetTime.Format(time.RFC3339))

	if !result.Allowed {
		c.Header("Retry-After", string(rune(result.RetryAfter)))
		c.JSON(http.StatusTooManyRequests, RateLimitExceededResponse{
			Error:             "Rate limit exceeded",
			Message:           "Too many requests. Please try again later.",
			Limit:             result.Limit,
			Remaining:         result.Remaining,
			ResetTime:         result.ResetTime,
			RetryAfter:        result.RetryAfter,
			ViolationRecorded: result.ViolationRecorded,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetRateLimitInfo retrieves rate limit information for an API key
// @Summary Get rate limit info
// @Description Get detailed rate limit information for an API key
// @Tags rate-limit
// @Accept json
// @Produce json
// @Param api_key_id path string true "API Key ID"
// @Success 200 {object} services.RateLimitInfo
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /rate-limit/{api_key_id}/info [get]
func (ctrl *RateLimitController) GetRateLimitInfo(c *gin.Context) {
	apiKeyIDStr := c.Param("api_key_id")
	apiKeyID, err := uuid.Parse(apiKeyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid API key ID",
			Message: err.Error(),
		})
		return
	}

	info, err := ctrl.rateLimitService.GetRateLimitInfo(c.Request.Context(), apiKeyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get rate limit info",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, info)
}

// ResetRateLimit resets the rate limit counter for an API key
// @Summary Reset rate limit
// @Description Reset the rate limit counter for an API key
// @Tags rate-limit
// @Accept json
// @Produce json
// @Param api_key_id path string true "API Key ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /rate-limit/{api_key_id}/reset [post]
func (ctrl *RateLimitController) ResetRateLimit(c *gin.Context) {
	apiKeyIDStr := c.Param("api_key_id")
	apiKeyID, err := uuid.Parse(apiKeyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid API key ID",
			Message: err.Error(),
		})
		return
	}

	if err := ctrl.rateLimitService.ResetRateLimit(c.Request.Context(), apiKeyID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to reset rate limit",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Rate limit reset successfully",
	})
}

// UpdateRateLimit updates the rate limit for an API key
// @Summary Update rate limit
// @Description Update the rate limit for an API key
// @Tags rate-limit
// @Accept json
// @Produce json
// @Param api_key_id path string true "API Key ID"
// @Param request body UpdateRateLimitRequest true "Update rate limit request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /rate-limit/{api_key_id} [put]
func (ctrl *RateLimitController) UpdateRateLimit(c *gin.Context) {
	apiKeyIDStr := c.Param("api_key_id")
	apiKeyID, err := uuid.Parse(apiKeyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid API key ID",
			Message: err.Error(),
		})
		return
	}

	var req UpdateRateLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	if err := ctrl.rateLimitService.UpdateRateLimit(c.Request.Context(), apiKeyID, req.NewLimit); err != nil {
		if err.Error() == "api key not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "API key not found",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update rate limit",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Rate limit updated successfully",
	})
}

// GetViolationHistory retrieves rate limit violation history
// @Summary Get violation history
// @Description Get rate limit violation history for an API key
// @Tags rate-limit
// @Accept json
// @Produce json
// @Param api_key_id path string true "API Key ID"
// @Param hours query int false "Hours to look back" default(24)
// @Success 200 {object} ViolationHistoryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /rate-limit/{api_key_id}/violations [get]
func (ctrl *RateLimitController) GetViolationHistory(c *gin.Context) {
	apiKeyIDStr := c.Param("api_key_id")
	apiKeyID, err := uuid.Parse(apiKeyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid API key ID",
			Message: err.Error(),
		})
		return
	}

	hours := 24 // default
	if hoursStr := c.Query("hours"); hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 { // max 1 week
			hours = h
		}
	}

	violations, err := ctrl.rateLimitService.GetViolationHistory(c.Request.Context(), apiKeyID, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get violation history",
			Message: err.Error(),
		})
		return
	}

	response := ViolationHistoryResponse{
		APIKeyID:   apiKeyID,
		Hours:      hours,
		Violations: violations,
		Count:      len(violations),
	}

	c.JSON(http.StatusOK, response)
}

// ValidateAPIKey validates an API key using the Authorization header
// @Summary Validate API key
// @Description Validate an API key from Authorization header
// @Tags rate-limit
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer API_KEY"
// @Success 200 {object} services.APIKeyResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /rate-limit/validate [post]
func (ctrl *RateLimitController) ValidateAPIKey(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Missing Authorization header",
			Message: "Authorization header is required",
		})
		return
	}

	// Extract API key from Bearer token
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid Authorization header format",
			Message: "Authorization header must be in format 'Bearer API_KEY'",
		})
		return
	}

	apiKey := authHeader[7:]

	// Validate the API key
	validatedKey, err := ctrl.apiKeyService.ValidateAPIKey(c.Request.Context(), apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Invalid API key",
			Message: err.Error(),
		})
		return
	}

	// Convert to response format (without the actual key)
	var userID uuid.UUID
	if validatedKey.UserID != nil {
		userID = *validatedKey.UserID
	}
	
	response := &services.APIKeyResponse{
		ID:          validatedKey.ID,
		Name:        validatedKey.Name,
		Description: validatedKey.Description,
		Status:      validatedKey.Status,
		Tier:        validatedKey.Tier,
		UserID:      userID,
		TeamID:      validatedKey.TeamID,
		Tags:        validatedKey.Tags,
		RateLimit:   validatedKey.RateLimit,
		QuotaLimit:  validatedKey.QuotaLimit,
		TotalUsage:  validatedKey.TotalUsage,
		LastUsedAt:  validatedKey.LastUsedAt,
		ExpiresAt:   validatedKey.ExpiresAt,
		CreatedAt:   validatedKey.CreatedAt,
		UpdatedAt:   validatedKey.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// Request/Response types

// RateLimitCheckRequest represents a rate limit check request
type RateLimitCheckRequest struct {
	APIKeyID  uuid.UUID `json:"api_key_id" binding:"required"`
	Endpoint  string    `json:"endpoint" binding:"required"`
	Method    string    `json:"method" binding:"required"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Country   string    `json:"country"`
}

// RateLimitExceededResponse represents a rate limit exceeded response
type RateLimitExceededResponse struct {
	Error             string    `json:"error"`
	Message           string    `json:"message"`
	Limit             int       `json:"limit"`
	Remaining         int       `json:"remaining"`
	ResetTime         time.Time `json:"reset_time"`
	RetryAfter        int       `json:"retry_after"`
	ViolationRecorded bool      `json:"violation_recorded"`
}

// UpdateRateLimitRequest represents an update rate limit request
type UpdateRateLimitRequest struct {
	NewLimit int `json:"new_limit" binding:"required,min=1"`
}

// ViolationHistoryResponse represents a violation history response
type ViolationHistoryResponse struct {
	APIKeyID   uuid.UUID                    `json:"api_key_id"`
	Hours      int                          `json:"hours"`
	Count      int                          `json:"count"`
	Violations []*models.RateLimitViolation `json:"violations"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
}