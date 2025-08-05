package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/viva/rate-limiter/internal/metrics"
	"github.com/viva/rate-limiter/internal/middleware"
)

func main() {
	// Initialize Prometheus metrics
	prometheusMetrics := metrics.NewPrometheusMetrics("viva", "ratelimiter")
	
	// Create Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	
	// Add metrics middleware
	router.Use(middleware.MetricsMiddleware(prometheusMetrics))
	router.Use(gin.Recovery())
	
	// Health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"timestamp": time.Now(),
			"metrics":   "enabled",
		})
	})
	
	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	
	// Test endpoints to generate metrics
	router.GET("/test/fast", func(c *gin.Context) {
		time.Sleep(10 * time.Millisecond)
		c.JSON(200, gin.H{"message": "fast response"})
	})
	
	router.GET("/test/slow", func(c *gin.Context) {
		time.Sleep(500 * time.Millisecond)
		c.JSON(200, gin.H{"message": "slow response"})
	})
	
	router.GET("/test/error", func(c *gin.Context) {
		c.JSON(500, gin.H{"error": "simulated error"})
	})
	
	port := 8091
	fmt.Printf("Starting metrics test server on port %d\n", port)
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  Health:  http://localhost:%d/health\n", port)
	fmt.Printf("  Metrics: http://localhost:%d/metrics\n", port)
	fmt.Printf("  Test:    http://localhost:%d/test/fast\n", port)
	fmt.Printf("  Test:    http://localhost:%d/test/slow\n", port)
	fmt.Printf("  Test:    http://localhost:%d/test/error\n", port)
	
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}