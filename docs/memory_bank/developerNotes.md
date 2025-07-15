# Developer Notes: Rate-Limited API Key Manager

## Development Setup

### Prerequisites
```bash
# Required software versions
go version 1.21+
docker version 20.10+
docker-compose version 2.0+
make version 4.0+
```

### Local Environment Setup
```bash
# Clone and setup project
git clone <repository-url>
cd rate-limiter

# Start dependencies
make dev-up

# Run database migrations
make migrate-up

# Install dependencies
go mod download

# Generate mocks
make generate

# Run tests
make test

# Start development server
make dev
```

### Docker Compose Services
```yaml
# Development services
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: ratelimiter_dev
      POSTGRES_USER: developer
      POSTGRES_PASSWORD: devpass
    ports: ["5432:5432"]
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./scripts/init.sql:/docker-entrypoint-initdb.d/init.sql

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data

  rabbitmq:
    image: rabbitmq:3.12-management-alpine
    environment:
      RABBITMQ_DEFAULT_USER: developer
      RABBITMQ_DEFAULT_PASS: devpass
    ports: 
      - "5672:5672"   # AMQP
      - "15672:15672" # Management UI
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq

  prometheus:
    image: prom/prometheus:latest
    ports: ["9090:9090"]
    volumes:
      - ./configs/prometheus.yml:/etc/prometheus/prometheus.yml
```

## Common Development Tasks

### Database Operations
```bash
# Create new migration
migrate create -ext sql -dir migrations -seq add_new_table

# Run migrations
make migrate-up

# Rollback migration
make migrate-down

# Reset database
make db-reset

# Seed test data
make db-seed
```

### Testing
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration

# Run load tests
make test-load

# Run specific test
go test -v ./internal/services -run TestAPIKeyService_Create
```

### Code Generation
```bash
# Generate mocks
make generate

# Generate swagger docs
make swagger

# Format code
make fmt

# Lint code
make lint
```

## Best Practices & Tips

### 1. Performance Optimization

#### Redis Operations
```go
//  DO: Use pipelining for batch operations
pipe := redisClient.Pipeline()
for _, key := range keys {
    pipe.Get(ctx, key)
}
results, err := pipe.Exec(ctx)

// L DON'T: Make individual calls in a loop
for _, key := range keys {
    value, err := redisClient.Get(ctx, key).Result()
    // This creates N round trips
}
```

#### Database Queries
```go
//  DO: Use prepared statements with GORM
result := db.Where("status = ? AND tier = ?", "active", "pro").Find(&keys)

//  DO: Use indexes for frequent queries
// CREATE INDEX idx_api_keys_status_tier ON api_keys(status, tier);

// L DON'T: Use raw SQL without prepared statements
db.Raw("SELECT * FROM api_keys WHERE status = '" + status + "'")
```

#### Connection Pooling
```go
//  DO: Configure connection pools properly
sqlDB, _ := db.DB()
sqlDB.SetMaxIdleConns(10)
sqlDB.SetMaxOpenConns(100)
sqlDB.SetConnMaxLifetime(time.Hour)

//  DO: Monitor connection pool metrics
prometheus.NewGaugeFunc(prometheus.GaugeOpts{
    Name: "db_connections_open",
}, func() float64 {
    stats := sqlDB.Stats()
    return float64(stats.OpenConnections)
})
```

### 2. Error Handling Patterns

#### Custom Error Types
```go
//  DO: Define domain-specific errors
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}

//  DO: Use error wrapping
if err := repo.Create(ctx, key); err != nil {
    return fmt.Errorf("failed to create API key: %w", err)
}
```

#### HTTP Error Responses
```go
//  DO: Consistent error response format
type ErrorResponse struct {
    Error struct {
        Code    string                 `json:"code"`
        Message string                 `json:"message"`
        Details map[string]interface{} `json:"details,omitempty"`
    } `json:"error"`
    RequestID string `json:"request_id"`
}

func handleError(c *gin.Context, err error) {
    var apiErr *APIError
    if errors.As(err, &apiErr) {
        c.JSON(apiErr.StatusCode, ErrorResponse{
            Error: struct {
                Code    string                 `json:"code"`
                Message string                 `json:"message"`
                Details map[string]interface{} `json:"details,omitempty"`
            }{
                Code:    apiErr.Code,
                Message: apiErr.Message,
                Details: apiErr.Details,
            },
            RequestID: getRequestID(c),
        })
        return
    }
    
    // Default to internal server error
    c.JSON(500, ErrorResponse{
        Error: struct {
            Code    string                 `json:"code"`
            Message string                 `json:"message"`
            Details map[string]interface{} `json:"details,omitempty"`
        }{
            Code:    "INTERNAL_ERROR",
            Message: "An internal error occurred",
        },
        RequestID: getRequestID(c),
    })
}
```

### 3. Logging Best Practices

#### Structured Logging
```go
//  DO: Use structured logging with context
logger := zap.L().With(
    zap.String("api_key_id", keyID.String()),
    zap.String("request_id", requestID),
    zap.String("operation", "create_key"),
)

logger.Info("creating API key",
    zap.String("tier", req.Tier),
    zap.Int("rate_limit", req.RateLimit),
)

if err != nil {
    logger.Error("failed to create API key",
        zap.Error(err),
        zap.Duration("duration", time.Since(start)),
    )
}
```

#### Log Levels
```go
//  DO: Use appropriate log levels
logger.Debug("cache hit", zap.String("key", cacheKey))    // Development debugging
logger.Info("API key created", zap.String("id", keyID))   // Important events
logger.Warn("rate limit approaching", zap.Int("usage", 90)) // Warning conditions
logger.Error("database connection failed", zap.Error(err)) // Error conditions
```

### 4. Testing Strategies

#### Unit Test Patterns
```go
//  DO: Use table-driven tests
func TestValidateAPIKey(t *testing.T) {
    tests := []struct {
        name    string
        key     string
        want    bool
        wantErr bool
    }{
        {
            name: "valid key format",
            key:  "sk_live_4eC39HqLyjWDarjtT1zdp7dc",
            want: true,
        },
        {
            name:    "invalid prefix",
            key:     "invalid_prefix_xyz",
            want:    false,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ValidateAPIKey(tt.key)
            
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

#### Mock Usage
```go
//  DO: Use dependency injection for testing
type APIKeyService struct {
    repo   APIKeyRepository
    cache  Cache
    logger *zap.Logger
}

func TestAPIKeyService_Create(t *testing.T) {
    mockRepo := new(mocks.APIKeyRepository)
    mockCache := new(mocks.Cache)
    
    service := &APIKeyService{
        repo:   mockRepo,
        cache:  mockCache,
        logger: zap.NewNop(),
    }
    
    mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.APIKey")).Return(nil)
    mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(nil)
    
    result, err := service.CreateAPIKey(context.Background(), CreateRequest{
        Name: "Test Key",
        Tier: "pro",
    })
    
    assert.NoError(t, err)
    assert.NotEmpty(t, result.Key)
    mockRepo.AssertExpectations(t)
}
```

## Common Pitfalls & Solutions

### 1. Redis Memory Issues
**Problem**: Redis memory usage grows unbounded
```go
// L WRONG: No expiration on keys
redisClient.Set(ctx, key, value, 0)

//  CORRECT: Always set appropriate TTL
redisClient.Set(ctx, key, value, time.Hour)

//  CORRECT: Set expiration in pipeline operations
pipe := redisClient.Pipeline()
pipe.ZAdd(ctx, key, member)
pipe.Expire(ctx, key, window)
pipe.Exec(ctx)
```

### 2. Database Connection Leaks
**Problem**: Running out of database connections
```go
// L WRONG: Not configuring connection pool
db, err := gorm.Open(postgres.Open(dsn))

//  CORRECT: Configure connection pool limits
db, err := gorm.Open(postgres.Open(dsn))
sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(25)
sqlDB.SetMaxIdleConns(5)
```

### 3. Context Cancellation
**Problem**: Long-running operations don't respect context cancellation
```go
// L WRONG: Ignoring context
func (s *Service) ProcessBatch(ctx context.Context, items []Item) error {
    for _, item := range items {
        // Long operation without checking context
        s.processItem(item)
    }
    return nil
}

//  CORRECT: Check context cancellation
func (s *Service) ProcessBatch(ctx context.Context, items []Item) error {
    for _, item := range items {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            if err := s.processItem(ctx, item); err != nil {
                return err
            }
        }
    }
    return nil
}
```

### 4. Race Conditions in Rate Limiting
**Problem**: Concurrent requests can bypass rate limits
```go
// L WRONG: Check-then-act race condition
count := getCurrentCount(key)
if count < limit {
    incrementCount(key) // Race condition here
    return true
}
return false

//  CORRECT: Atomic operations with Lua script
luaScript := `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Remove old entries
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

-- Get current count
local current = redis.call('ZCARD', key)

if current < limit then
    -- Add current request
    redis.call('ZADD', key, now, now)
    redis.call('EXPIRE', key, window)
    return 1
else
    return 0
end
`
```

## Monitoring & Debugging

### Key Metrics to Monitor
```go
// Request metrics
var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
        },
        []string{"method", "endpoint", "status"},
    )
    
    rateLimitChecks = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rate_limit_checks_total",
        },
        []string{"result"}, // "allowed" or "denied"
    )
    
    activeAPIKeys = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "active_api_keys_total",
    })
)
```

### Debug Endpoints
```go
// Debug handlers (only in development)
if config.Environment == "development" {
    debug := router.Group("/debug")
    debug.GET("/health", healthCheck)
    debug.GET("/metrics", gin.WrapH(promhttp.Handler()))
    debug.GET("/vars", gin.WrapH(expvar.Handler()))
    
    // Redis debugging
    debug.GET("/redis/keys/:pattern", func(c *gin.Context) {
        pattern := c.Param("pattern")
        keys, err := redisClient.Keys(c, pattern).Result()
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, gin.H{"keys": keys})
    })
}
```

### Useful Redis Commands for Debugging
```bash
# Check rate limit keys
redis-cli --scan --pattern "key:rate:*" | head -10

# Inspect rate limit data
redis-cli ZRANGE key:rate:api_key_xxx:1234567890 0 -1 WITHSCORES

# Check key metadata
redis-cli HGETALL key:meta:api_key_xxx

# Monitor Redis operations in real-time
redis-cli MONITOR

# Check memory usage
redis-cli INFO memory
```

### Database Query Debugging
```sql
-- Check slow queries
SELECT query, mean_time, calls, total_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;

-- Check table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check index usage
SELECT 
    indexrelname,
    idx_tup_read,
    idx_tup_fetch,
    idx_scan
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```

## Performance Tuning Tips

### 1. Reduce Memory Allocations
```go
//  Use sync.Pool for frequently allocated objects
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 0, 1024)
    },
}

func processRequest() {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf[:0])
    
    // Use buffer
}
```

### 2. Optimize JSON Marshaling
```go
//  Use json.Encoder for streaming
func writeJSONResponse(w http.ResponseWriter, data interface{}) error {
    w.Header().Set("Content-Type", "application/json")
    return json.NewEncoder(w).Encode(data)
}

//  Pre-allocate slices when size is known
keys := make([]*APIKey, 0, expectedCount)
```

### 3. Batch Database Operations
```go
//  Batch insert usage logs
func (r *Repository) BatchInsertUsageLogs(ctx context.Context, logs []UsageLog) error {
    if len(logs) == 0 {
        return nil
    }
    
    // Use GORM's CreateInBatches for efficient batch inserts
    return r.db.WithContext(ctx).CreateInBatches(logs, 1000).Error
}
```

## Security Checklist

### API Key Security
- [ ] API keys are never logged in plain text
- [ ] Keys are hashed before database storage
- [ ] Key rotation is supported
- [ ] Keys have appropriate entropy (32+ bytes)

### Request Security
- [ ] All requests are authenticated
- [ ] Rate limiting prevents abuse
- [ ] Request size limits are enforced
- [ ] CORS headers are properly configured

### Database Security
- [ ] Database connections use TLS
- [ ] Prepared statements prevent SQL injection
- [ ] Database user has minimal required permissions
- [ ] Sensitive data is encrypted at rest

### Operational Security
- [ ] Secrets are not in source code
- [ ] Environment variables for configuration
- [ ] Regular security updates
- [ ] Security scanning in CI/CD