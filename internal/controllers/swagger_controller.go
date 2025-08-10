package controllers

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type SwaggerController struct{}

func NewSwaggerController() *SwaggerController {
	return &SwaggerController{}
}

// ServeSwaggerUI serves the Swagger UI interface
func (sc *SwaggerController) ServeSwaggerUI() gin.HandlerFunc {
	return ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/openapi.yaml"))
}

// ServeOpenAPISpec serves the OpenAPI specification
func (sc *SwaggerController) ServeOpenAPISpec(c *gin.Context) {
	// Read the OpenAPI spec file
	specPath := filepath.Join("docs", "api", "openapi.yaml")
	data, err := os.ReadFile(specPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "OpenAPI specification not found",
			"message": "The API documentation is not available",
		})
		return
	}

	c.Header("Content-Type", "application/yaml")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Data(http.StatusOK, "application/yaml", data)
}

// ServeSwaggerRedirect redirects /swagger to /swagger/
func (sc *SwaggerController) ServeSwaggerRedirect(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/swagger/")
}