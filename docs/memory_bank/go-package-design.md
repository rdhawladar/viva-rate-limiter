# Go Package Design for Viva Rate Limiter

## Package Structure

The standalone rate limiter will be published as a Go package that can be imported directly into any Go application, providing both in-memory and distributed Redis-based rate limiting.

### Package Organization
```
github.com/viva/ratelimiter
├── ratelimiter.go          # Main package interface
├── algorithms/             # Rate limiting algorithms
│   ├── sliding_window.go
│   ├── token_bucket.go
│   ├── leaky_bucket.go
│   └── fixed_window.go
├── store/                  # Storage backends
│   ├── memory.go          # In-memory store
│   ├── redis.go           # Redis store
│   └── interface.go       # Store interface
├── middleware/            # HTTP middleware
│   ├── gin.go
│   ├── echo.go
│   ├── chi.go
│   └── http.go           # Standard net/http
└── examples/
    ├── basic/
    ├── middleware/
    └── distributed/
```

## Public API Design

### Core Interface
```go
package ratelimiter

import (
    "context"
    "time"
)

// RateLimiter is the main interface for rate limiting
type RateLimiter interface {
    // Allow checks if a request is allowed
    Allow(ctx context.Context, key string) (bool, error)
    
    // AllowN checks if n requests are allowed
    AllowN(ctx context.Context, key string, n int) (bool, error)
    
    // Check returns detailed rate limit information without consuming
    Check(ctx context.Context, key string) (*Status, error)
    
    // Reset clears the rate limit for a key
    Reset(ctx context.Context, key string) error
}

// Status contains rate limit status information
type Status struct {
    Allowed   bool
    Limit     int
    Remaining int
    ResetAt   time.Time
    RetryAfter time.Duration
}

// Config holds rate limiter configuration
type Config struct {
    // Rate limit configuration
    Rate     int           // Requests per window
    Window   time.Duration // Time window
    Burst    int           // Burst capacity (for token bucket)
    
    // Algorithm selection
    Algorithm Algorithm
    
    // Storage backend
    Store    Store
    
    // Options
    KeyPrefix string // Prefix for all keys
    OnLimit   func(key string, status *Status) // Callback on rate limit
}

// Algorithm types
type Algorithm int

const (
    SlidingWindow Algorithm = iota
    TokenBucket
    LeakyBucket
    FixedWindow
)
```

### Constructor Functions
```go
// New creates a rate limiter with custom config
func New(config Config) (RateLimiter, error)

// NewMemory creates an in-memory rate limiter
func NewMemory(rate int, window time.Duration) RateLimiter

// NewRedis creates a Redis-backed rate limiter
func NewRedis(redisURL string, rate int, window time.Duration) (RateLimiter, error)

// NewCluster creates a Redis cluster-backed rate limiter
func NewCluster(addresses []string, rate int, window time.Duration) (RateLimiter, error)
```

## Usage Examples

### Basic In-Memory Usage
```go
import "github.com/viva/ratelimiter"

// Create limiter: 100 requests per minute
limiter := ratelimiter.NewMemory(100, time.Minute)

// Check if request is allowed
if allowed, _ := limiter.Allow(ctx, "user:123"); !allowed {
    // Rate limit exceeded
    return errors.New("too many requests")
}
```

### Redis-Backed Usage
```go
// Create distributed limiter
limiter, err := ratelimiter.NewRedis("redis://localhost:6379", 1000, time.Hour)
if err != nil {
    log.Fatal(err)
}

// Use with detailed status
status, err := limiter.Check(ctx, "api:user:123")
if err != nil {
    return err
}

if !status.Allowed {
    w.Header().Set("Retry-After", fmt.Sprint(status.RetryAfter.Seconds()))
    w.Header().Set("X-RateLimit-Remaining", "0")
    w.WriteHeader(429)
    return
}
```

### Advanced Configuration
```go
limiter, err := ratelimiter.New(ratelimiter.Config{
    Rate:      1000,
    Window:    time.Hour,
    Burst:     50, // Allow burst of 50
    Algorithm: ratelimiter.TokenBucket,
    Store: ratelimiter.NewRedisStore(ratelimiter.RedisConfig{
        Addresses: []string{"redis1:6379", "redis2:6379"},
        Cluster:   true,
        PoolSize:  100,
    }),
    KeyPrefix: "myapp:",
    OnLimit: func(key string, status *ratelimiter.Status) {
        log.Printf("Rate limit hit for %s, retry after %v", key, status.RetryAfter)
    },
})
```

### HTTP Middleware

#### Gin Middleware
```go
import (
    "github.com/viva/ratelimiter"
    "github.com/viva/ratelimiter/middleware"
)

limiter := ratelimiter.NewMemory(100, time.Minute)

// Per-IP rate limiting
router.Use(middleware.Gin(limiter, middleware.Config{
    KeyFunc: func(c *gin.Context) string {
        return c.ClientIP()
    },
    ErrorHandler: func(c *gin.Context, status *ratelimiter.Status) {
        c.JSON(429, gin.H{
            "error": "rate limit exceeded",
            "retry_after": status.RetryAfter.Seconds(),
        })
    },
}))

// Per-user rate limiting
router.Use(middleware.Gin(limiter, middleware.Config{
    KeyFunc: func(c *gin.Context) string {
        return c.GetString("user_id")
    },
    Skip: func(c *gin.Context) bool {
        return c.GetString("user_id") == "" // Skip if no user
    },
}))
```

#### Standard HTTP Middleware
```go
import (
    "github.com/viva/ratelimiter"
    "github.com/viva/ratelimiter/middleware"
)

limiter := ratelimiter.NewMemory(60, time.Minute)

// Wrap any http.Handler
handler := middleware.HTTP(limiter, middleware.Config{
    KeyFunc: func(r *http.Request) string {
        return r.Header.Get("X-API-Key")
    },
})(yourHandler)

http.ListenAndServe(":8080", handler)
```

### Custom Key Strategies
```go
// Hierarchical rate limiting
type HierarchicalLimiter struct {
    global  ratelimiter.RateLimiter // 10k/hour global
    perUser ratelimiter.RateLimiter // 1k/hour per user
    perIP   ratelimiter.RateLimiter // 100/hour per IP
}

func (h *HierarchicalLimiter) Allow(userID, ip string) bool {
    // Check all limits
    if allowed, _ := h.global.Allow(ctx, "global"); !allowed {
        return false
    }
    if allowed, _ := h.perUser.Allow(ctx, "user:"+userID); !allowed {
        return false
    }
    if allowed, _ := h.perIP.Allow(ctx, "ip:"+ip); !allowed {
        return false
    }
    return true
}
```

### Testing Support
```go
import (
    "github.com/viva/ratelimiter"
    "github.com/viva/ratelimiter/testutil"
)

func TestMyHandler(t *testing.T) {
    // Create test limiter with time control
    limiter := testutil.NewTestLimiter(10, time.Minute)
    
    // Fast-forward time in tests
    limiter.Advance(time.Minute)
    
    // Assert rate limits
    testutil.AssertAllowed(t, limiter, "key", 10)
    testutil.AssertBlocked(t, limiter, "key", 1)
}
```

## Package Features

### 1. Zero Dependencies for Core
The core package has zero external dependencies, using only Go standard library. Redis support is optional.

### 2. Thread-Safe
All operations are thread-safe and can be used concurrently.

### 3. Context Support
All methods accept context for cancellation and timeouts.

### 4. Metrics Integration
```go
// Prometheus metrics
limiter.EnableMetrics(ratelimiter.MetricsConfig{
    Namespace: "myapp",
    Subsystem: "ratelimiter",
})

// Custom metrics
limiter.OnAllow(func(key string) {
    myMetrics.IncrementAllowed(key)
})
limiter.OnLimit(func(key string) {
    myMetrics.IncrementLimited(key)
})
```

### 5. Flexible Storage
```go
// Implement custom storage
type CustomStore struct{}

func (s *CustomStore) Get(ctx context.Context, key string) (int64, error)
func (s *CustomStore) Increment(ctx context.Context, key string, window time.Time) (int64, error)
func (s *CustomStore) Reset(ctx context.Context, key string) error
```

## Installation

```bash
go get github.com/viva/ratelimiter
```

### Version Support
- Go 1.18+ (uses generics for type-safe options)
- Redis 6.2+ (for Redis backend)

## Performance Characteristics

### In-Memory Performance
- **Operations**: 10M+ ops/second
- **Memory**: O(n) where n = number of active keys
- **Latency**: < 100ns per operation

### Redis Performance
- **Operations**: 100k+ ops/second (single Redis)
- **Latency**: < 1ms per operation
- **Scalability**: Linear with Redis cluster size

## Package Distribution

### Go Modules
```go
module github.com/viva/ratelimiter

go 1.21

require (
    github.com/redis/go-redis/v9 v9.0.0
)
```

### Semantic Versioning
- v1.x.x - Stable API, backward compatible
- v2.x.x - Major changes (if needed)

### Tags and Releases
```bash
git tag v1.0.0
git push origin v1.0.0
```

### Documentation
- pkg.go.dev auto-generated docs
- Comprehensive examples
- Benchmarks included