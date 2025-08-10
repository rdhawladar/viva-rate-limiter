package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rdhawladar/viva-rate-limiter/internal/repositories"
	"github.com/rdhawladar/viva-rate-limiter/internal/services"
)

// APIKeyController handles API key related endpoints
type APIKeyController struct {
	apiKeyService services.APIKeyService
}

// NewAPIKeyController creates a new API key controller
func NewAPIKeyController(apiKeyService services.APIKeyService) *APIKeyController {
	return &APIKeyController{
		apiKeyService: apiKeyService,
	}
}

// CreateAPIKey creates a new API key
// @Summary Create API key
// @Description Create a new API key
// @Tags api-keys
// @Accept json
// @Produce json
// @Param request body services.CreateAPIKeyRequest true "Create API key request"
// @Success 201 {object} services.APIKeyResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api-keys [post]
func (ctrl *APIKeyController) CreateAPIKey(c *gin.Context) {
	var req services.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	apiKey, err := ctrl.apiKeyService.CreateAPIKey(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create API key",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, apiKey)
}

// GetAPIKey retrieves an API key by ID
// @Summary Get API key
// @Description Get an API key by ID
// @Tags api-keys
// @Accept json
// @Produce json
// @Param id path string true "API Key ID"
// @Success 200 {object} services.APIKeyResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api-keys/{id} [get]
func (ctrl *APIKeyController) GetAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid API key ID",
			Message: err.Error(),
		})
		return
	}

	apiKey, err := ctrl.apiKeyService.GetAPIKey(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "API key not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

// UpdateAPIKey updates an existing API key
// @Summary Update API key
// @Description Update an existing API key
// @Tags api-keys
// @Accept json
// @Produce json
// @Param id path string true "API Key ID"
// @Param request body services.UpdateAPIKeyRequest true "Update API key request"
// @Success 200 {object} services.APIKeyResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api-keys/{id} [put]
func (ctrl *APIKeyController) UpdateAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid API key ID",
			Message: err.Error(),
		})
		return
	}

	var req services.UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	apiKey, err := ctrl.apiKeyService.UpdateAPIKey(c.Request.Context(), id, &req)
	if err != nil {
		if err.Error() == "api key not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "API key not found",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update API key",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

// DeleteAPIKey deletes an API key
// @Summary Delete API key
// @Description Delete an API key
// @Tags api-keys
// @Accept json
// @Produce json
// @Param id path string true "API Key ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api-keys/{id} [delete]
func (ctrl *APIKeyController) DeleteAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid API key ID",
			Message: err.Error(),
		})
		return
	}

	if err := ctrl.apiKeyService.DeleteAPIKey(c.Request.Context(), id); err != nil {
		if err.Error() == "api key not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "API key not found",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete API key",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListAPIKeys lists API keys with filtering and pagination
// @Summary List API keys
// @Description List API keys with optional filtering and pagination
// @Tags api-keys
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param status query string false "Filter by status"
// @Param tier query string false "Filter by tier"
// @Param user_id query string false "Filter by user ID"
// @Success 200 {object} repositories.PaginatedResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api-keys [get]
func (ctrl *APIKeyController) ListAPIKeys(c *gin.Context) {
	// Parse pagination parameters
	page := 1
	pageSize := 20

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	pagination := &repositories.PaginationParams{
		Page:     page,
		PageSize: pageSize,
		OrderBy:  c.DefaultQuery("order_by", "created_at"),
		Order:    c.DefaultQuery("order", "desc"),
	}

	// Parse filter parameters
	filter := &repositories.APIKeyFilter{}

	if status := c.Query("status"); status != "" {
		// Parse status enum
		// This would need proper enum parsing based on your models
		filter.Search = status
	}

	if tier := c.Query("tier"); tier != "" {
		// Parse tier enum
		filter.Search = tier
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			filter.UserID = &userID
		}
	}

	if search := c.Query("search"); search != "" {
		filter.Search = search
	}

	result, err := ctrl.apiKeyService.ListAPIKeys(c.Request.Context(), filter, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list API keys",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RotateAPIKey rotates an API key (generates a new key)
// @Summary Rotate API key
// @Description Generate a new key for an existing API key
// @Tags api-keys
// @Accept json
// @Produce json
// @Param id path string true "API Key ID"
// @Success 200 {object} services.APIKeyResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api-keys/{id}/rotate [post]
func (ctrl *APIKeyController) RotateAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid API key ID",
			Message: err.Error(),
		})
		return
	}

	apiKey, err := ctrl.apiKeyService.RotateAPIKey(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "api key not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "API key not found",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to rotate API key",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

// GetAPIKeyStats retrieves statistics for an API key
// @Summary Get API key statistics
// @Description Get usage statistics for an API key
// @Tags api-keys
// @Accept json
// @Produce json
// @Param id path string true "API Key ID"
// @Success 200 {object} services.APIKeyStats
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api-keys/{id}/stats [get]
func (ctrl *APIKeyController) GetAPIKeyStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid API key ID",
			Message: err.Error(),
		})
		return
	}

	stats, err := ctrl.apiKeyService.GetAPIKeyStats(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "api key not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "API key not found",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get API key stats",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}