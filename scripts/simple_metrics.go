package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "viva",
			Subsystem: "ratelimiter",
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status_code"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "viva",
			Subsystem: "ratelimiter", 
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "viva",
			Subsystem: "ratelimiter",
			Name:      "http_requests_in_flight",
			Help:      "Number of HTTP requests currently being processed",
		},
	)
)

// Simple metrics middleware
func metricsMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		httpRequestsInFlight.Inc()
		
		c.Next()
		
		duration := time.Since(start).Seconds()
		httpRequestsInFlight.Dec()
		
		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		statusCode := strconv.Itoa(c.Writer.Status())
		
		httpRequestsTotal.WithLabelValues(method, path, statusCode).Inc()
		httpRequestDuration.WithLabelValues(method, path).Observe(duration)
	})
}

func main() {
	// Create Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	
	// Add metrics middleware
	router.Use(metricsMiddleware())
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
	fmt.Printf("🚀 Starting metrics test server on port %d\n", port)
	fmt.Printf("📊 Endpoints:\n")
	fmt.Printf("   Health:  http://localhost:%d/health\n", port)
	fmt.Printf("   Metrics: http://localhost:%d/metrics\n", port)
	fmt.Printf("   Test:    http://localhost:%d/test/fast\n", port)
	fmt.Printf("   Test:    http://localhost:%d/test/slow\n", port)
	fmt.Printf("   Test:    http://localhost:%d/test/error\n", port)
	fmt.Printf("\n💡 Try making some requests to generate metrics!\n")
	
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}