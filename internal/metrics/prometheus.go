package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusMetrics contains all Prometheus metrics for the application
type PrometheusMetrics struct {
	// HTTP metrics
	HTTPRequestsTotal     *prometheus.CounterVec
	HTTPRequestDuration   *prometheus.HistogramVec
	HTTPRequestsInFlight  prometheus.Gauge

	// Rate limiting metrics
	RateLimitChecksTotal    *prometheus.CounterVec
	RateLimitViolationsTotal *prometheus.CounterVec
	RateLimitResetTotal     prometheus.Counter

	// API Key metrics
	APIKeysTotal           *prometheus.GaugeVec
	APIKeyRequestsTotal    *prometheus.CounterVec
	APIKeyUsageBytes       *prometheus.CounterVec

	// Cache metrics
	CacheHitsTotal         *prometheus.CounterVec
	CacheMissesTotal       *prometheus.CounterVec
	CacheOperationsTotal   *prometheus.CounterVec

	// Database metrics
	DatabaseConnectionsActive prometheus.Gauge
	DatabaseQueriesTotal      *prometheus.CounterVec
	DatabaseQueryDuration     *prometheus.HistogramVec

	// Worker metrics
	WorkerTasksTotal         *prometheus.CounterVec
	WorkerTaskDuration       *prometheus.HistogramVec
	WorkerQueueSize          *prometheus.GaugeVec

	// Business metrics
	BillingRecordsTotal      *prometheus.CounterVec
	AlertsTriggeredTotal     *prometheus.CounterVec
	UsageLogsProcessedTotal  prometheus.Counter
}

// NewPrometheusMetrics creates and registers all Prometheus metrics
func NewPrometheusMetrics(namespace, subsystem string) *PrometheusMetrics {
	return &PrometheusMetrics{
		// HTTP metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status_code"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		HTTPRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_requests_in_flight",
				Help:      "Number of HTTP requests currently being processed",
			},
		),

		// Rate limiting metrics
		RateLimitChecksTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "rate_limit_checks_total",
				Help:      "Total number of rate limit checks",
			},
			[]string{"api_key_id", "tier", "result"},
		),
		RateLimitViolationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "rate_limit_violations_total",
				Help:      "Total number of rate limit violations",
			},
			[]string{"api_key_id", "tier", "endpoint"},
		),
		RateLimitResetTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "rate_limit_resets_total",
				Help:      "Total number of rate limit resets",
			},
		),

		// API Key metrics
		APIKeysTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "api_keys_total",
				Help:      "Total number of API keys by tier and status",
			},
			[]string{"tier", "status"},
		),
		APIKeyRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "api_key_requests_total",
				Help:      "Total number of requests per API key",
			},
			[]string{"api_key_id", "tier", "endpoint", "method"},
		),
		APIKeyUsageBytes: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "api_key_usage_bytes_total",
				Help:      "Total bytes transferred per API key",
			},
			[]string{"api_key_id", "tier", "direction"},
		),

		// Cache metrics
		CacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache_type", "key_pattern"},
		),
		CacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache_type", "key_pattern"},
		),
		CacheOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cache_operations_total",
				Help:      "Total number of cache operations",
			},
			[]string{"operation", "result"},
		),

		// Database metrics
		DatabaseConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "database_connections_active",
				Help:      "Number of active database connections",
			},
		),
		DatabaseQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "database_queries_total",
				Help:      "Total number of database queries",
			},
			[]string{"table", "operation", "result"},
		),
		DatabaseQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "database_query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"table", "operation"},
		),

		// Worker metrics
		WorkerTasksTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "worker_tasks_total",
				Help:      "Total number of worker tasks processed",
			},
			[]string{"task_type", "queue", "result"},
		),
		WorkerTaskDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "worker_task_duration_seconds",
				Help:      "Worker task duration in seconds",
				Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60, 300},
			},
			[]string{"task_type", "queue"},
		),
		WorkerQueueSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "worker_queue_size",
				Help:      "Current size of worker queues",
			},
			[]string{"queue"},
		),

		// Business metrics
		BillingRecordsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "billing_records_total",
				Help:      "Total number of billing records",
			},
			[]string{"status", "tier"},
		),
		AlertsTriggeredTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "alerts_triggered_total",
				Help:      "Total number of alerts triggered",
			},
			[]string{"type", "severity", "api_key_id"},
		),
		UsageLogsProcessedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "usage_logs_processed_total",
				Help:      "Total number of usage logs processed",
			},
		),
	}
}

// RecordHTTPRequest records HTTP request metrics
func (m *PrometheusMetrics) RecordHTTPRequest(method, path, statusCode string, duration float64) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, statusCode).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
}

// IncHTTPRequestsInFlight increments in-flight requests
func (m *PrometheusMetrics) IncHTTPRequestsInFlight() {
	m.HTTPRequestsInFlight.Inc()
}

// DecHTTPRequestsInFlight decrements in-flight requests
func (m *PrometheusMetrics) DecHTTPRequestsInFlight() {
	m.HTTPRequestsInFlight.Dec()
}

// RecordRateLimitCheck records rate limit check metrics
func (m *PrometheusMetrics) RecordRateLimitCheck(apiKeyID, tier, result string) {
	m.RateLimitChecksTotal.WithLabelValues(apiKeyID, tier, result).Inc()
}

// RecordRateLimitViolation records rate limit violation metrics
func (m *PrometheusMetrics) RecordRateLimitViolation(apiKeyID, tier, endpoint string) {
	m.RateLimitViolationsTotal.WithLabelValues(apiKeyID, tier, endpoint).Inc()
}

// RecordRateLimitReset records rate limit reset
func (m *PrometheusMetrics) RecordRateLimitReset() {
	m.RateLimitResetTotal.Inc()
}

// UpdateAPIKeyCount updates API key count metrics
func (m *PrometheusMetrics) UpdateAPIKeyCount(tier, status string, count float64) {
	m.APIKeysTotal.WithLabelValues(tier, status).Set(count)
}

// RecordAPIKeyRequest records API key request metrics
func (m *PrometheusMetrics) RecordAPIKeyRequest(apiKeyID, tier, endpoint, method string) {
	m.APIKeyRequestsTotal.WithLabelValues(apiKeyID, tier, endpoint, method).Inc()
}

// RecordAPIKeyUsage records API key usage bytes
func (m *PrometheusMetrics) RecordAPIKeyUsage(apiKeyID, tier, direction string, bytes float64) {
	m.APIKeyUsageBytes.WithLabelValues(apiKeyID, tier, direction).Add(bytes)
}

// RecordCacheHit records cache hit
func (m *PrometheusMetrics) RecordCacheHit(cacheType, keyPattern string) {
	m.CacheHitsTotal.WithLabelValues(cacheType, keyPattern).Inc()
}

// RecordCacheMiss records cache miss
func (m *PrometheusMetrics) RecordCacheMiss(cacheType, keyPattern string) {
	m.CacheMissesTotal.WithLabelValues(cacheType, keyPattern).Inc()
}

// RecordCacheOperation records cache operation
func (m *PrometheusMetrics) RecordCacheOperation(operation, result string) {
	m.CacheOperationsTotal.WithLabelValues(operation, result).Inc()
}

// UpdateDatabaseConnections updates database connection count
func (m *PrometheusMetrics) UpdateDatabaseConnections(count float64) {
	m.DatabaseConnectionsActive.Set(count)
}

// RecordDatabaseQuery records database query metrics
func (m *PrometheusMetrics) RecordDatabaseQuery(table, operation, result string, duration float64) {
	m.DatabaseQueriesTotal.WithLabelValues(table, operation, result).Inc()
	m.DatabaseQueryDuration.WithLabelValues(table, operation).Observe(duration)
}

// RecordWorkerTask records worker task metrics
func (m *PrometheusMetrics) RecordWorkerTask(taskType, queue, result string, duration float64) {
	m.WorkerTasksTotal.WithLabelValues(taskType, queue, result).Inc()
	m.WorkerTaskDuration.WithLabelValues(taskType, queue).Observe(duration)
}

// UpdateWorkerQueueSize updates worker queue size
func (m *PrometheusMetrics) UpdateWorkerQueueSize(queue string, size float64) {
	m.WorkerQueueSize.WithLabelValues(queue).Set(size)
}

// RecordBillingRecord records billing record metrics
func (m *PrometheusMetrics) RecordBillingRecord(status, tier string) {
	m.BillingRecordsTotal.WithLabelValues(status, tier).Inc()
}

// RecordAlert records alert metrics
func (m *PrometheusMetrics) RecordAlert(alertType, severity, apiKeyID string) {
	m.AlertsTriggeredTotal.WithLabelValues(alertType, severity, apiKeyID).Inc()
}

// RecordUsageLogProcessed records usage log processing
func (m *PrometheusMetrics) RecordUsageLogProcessed() {
	m.UsageLogsProcessedTotal.Inc()
}