# Rate Limiter Package

A flexible and efficient rate limiting library for Go applications with support for multiple backends and algorithms.

## Features

- **Multiple Backends**: Redis (distributed) and in-memory (local)
- **Sliding Window Algorithm**: Accurate rate limiting with smooth request distribution
- **Flexible Configuration**: Per-key custom limits and global defaults
- **Production Ready**: Used in high-throughput applications
- **Zero Dependencies**: Core functionality doesn't require external dependencies
- **Thread Safe**: Concurrent access protection built-in
- **Callbacks**: Hooks for monitoring and metrics
- **Context Support**: Proper context handling and cancellation

## Quick Start

### Installation

```bash
go get github.com/viva/rate-limiter/pkg/ratelimit
```

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/viva/rate-limiter/pkg/ratelimit"
)

func main() {
    // Create a rate limiter with default settings (100 requests per hour)
    limiter := ratelimit.New(ratelimit.DefaultOptions())
    defer limiter.Close()
    
    ctx := context.Background()
    
    // Check if a request is allowed
    if limiter.Allow(ctx, "user123") {
        fmt.Println("Request allowed!")
    } else {
        fmt.Println("Rate limit exceeded!")
    }
    
    // Get detailed information
    info, err := limiter.Info(ctx, "user123")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Used: %d/%d, Remaining: %d\n", 
        info.Used, info.Limit, info.Remaining)
}
```

## Configuration

### Default Options

```go
opts := ratelimit.DefaultOptions()
// DefaultLimit: 100 requests
// DefaultWindow: 1 hour  
// Backend: In-memory
// KeyPrefix: "ratelimit:"
```

### Custom Configuration

```go
opts := ratelimit.Options{
    DefaultLimit:  1000,
    DefaultWindow: time.Hour,
    KeyPrefix:    "myapp:",
    OnLimitExceeded: func(key string, limit int, window time.Duration) {
        log.Printf("Rate limit exceeded for %s", key)
    },
    OnAllow: func(key string, remaining int, window time.Duration) {
        metrics.RecordAllowedRequest(key, remaining)
    },
}

limiter := ratelimit.New(opts)
```

## Backends

### In-Memory Backend (Default)

Perfect for single-instance applications and testing:

```go
// Uses in-memory backend by default
limiter := ratelimit.New(ratelimit.DefaultOptions())
```

### Redis Backend

For distributed applications and persistence:

```go
// Configure Redis
config := ratelimit.DefaultRedisConfig()
config.Addresses = []string{"localhost:6379"}
config.Password = "secret"

// Create Redis backend
backend, err := ratelimit.NewRedisBackend(config)
if err != nil {
    panic(err)
}

// Use with rate limiter
opts := ratelimit.DefaultOptions()
opts.Backend = backend

limiter := ratelimit.New(opts)
```

### Redis Cluster Support

```go
config := ratelimit.DefaultRedisConfig()
config.ClusterMode = true
config.Addresses = []string{
    "redis-node1:6379",
    "redis-node2:6379", 
    "redis-node3:6379",
}

backend, err := ratelimit.NewRedisBackend(config)
```

## Advanced Usage

### Per-Key Custom Limits

```go
limiter := ratelimit.New(ratelimit.DefaultOptions())

// Set custom limits for specific keys
limiter.SetLimit(ctx, "premium-user", 1000, time.Hour)
limiter.SetLimit(ctx, "free-user", 100, time.Hour)
limiter.SetLimit(ctx, "api-endpoint", 10000, time.Minute)
```

### Bulk Operations

```go
// Allow multiple requests at once
allowed := limiter.AllowN(ctx, "user123", 5)
```

### Rate Limit Information

```go
info, err := limiter.Info(ctx, "user123")
if err != nil {
    return err
}

fmt.Printf("Limit: %d\n", info.Limit)
fmt.Printf("Used: %d\n", info.Used) 
fmt.Printf("Remaining: %d\n", info.Remaining)
fmt.Printf("Window: %v\n", info.Window)
fmt.Printf("Reset Time: %v\n", info.ResetTime)
fmt.Printf("Retry After: %v\n", info.RetryAfter)
```

### Reset Rate Limits

```go
// Reset rate limit for a specific key
err := limiter.Reset(ctx, "user123")
```

## Web Server Integration

### HTTP Middleware Example

```go
func rateLimitMiddleware(limiter ratelimit.Limiter, key string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow(r.Context(), key) {
                info, _ := limiter.Info(r.Context(), key)
                
                w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
                w.Header().Set("X-RateLimit-Remaining", "0")
                w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", info.ResetTime.Unix()))
                
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### User-Based Rate Limiting

```go
func getUserRateLimit(userID string) string {
    return "user:" + userID
}

func apiHandler(limiter ratelimit.Limiter) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID := getUserID(r) // Extract user ID from request
        key := getUserRateLimit(userID)
        
        if !limiter.Allow(r.Context(), key) {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        
        // Handle request...
    }
}
```

## Error Handling

```go
import "github.com/viva/rate-limiter/pkg/errors"

info, err := limiter.Info(ctx, "user123")
if err != nil {
    switch err {
    case errors.ErrBackendUnavailable:
        // Handle backend connectivity issues
        log.Printf("Rate limiter backend unavailable: %v", err)
        // Fail open or closed based on your requirements
        
    case errors.ErrInvalidKey:
        // Handle invalid key
        log.Printf("Invalid rate limit key: %v", err)
        
    default:
        // Handle other errors
        log.Printf("Rate limiter error: %v", err)
    }
}
```

## Monitoring and Metrics

### Callbacks for Observability

```go
opts := ratelimit.Options{
    OnLimitExceeded: func(key string, limit int, window time.Duration) {
        // Record metrics
        metrics.Counter("rate_limit_exceeded").
            WithTag("key", key).
            WithTag("limit", limit).
            Increment()
            
        // Log for debugging
        log.Printf("Rate limit exceeded: key=%s limit=%d window=%v", 
            key, limit, window)
    },
    OnAllow: func(key string, remaining int, window time.Duration) {
        // Record allowed requests
        metrics.Counter("rate_limit_allowed").
            WithTag("key", key).
            Increment()
            
        metrics.Gauge("rate_limit_remaining").
            WithTag("key", key).
            Set(float64(remaining))
    },
}
```

### Health Checks

```go
// For Redis backend
if redisBackend, ok := limiter.Backend.(*ratelimit.RedisBackend); ok {
    err := redisBackend.Ping(context.Background())
    if err != nil {
        log.Printf("Rate limiter backend unhealthy: %v", err)
    }
}
```

## Performance Considerations

### Redis Performance

- Uses Lua scripts for atomic operations
- Automatic cleanup of expired entries
- Connection pooling and retries built-in
- Supports Redis pipelining

### Memory Backend Performance

- Lock-free reads when possible
- Automatic cleanup of expired entries
- Configurable cleanup intervals
- Memory usage scales with active keys

### Best Practices

1. **Key Design**: Use hierarchical keys like `user:123`, `api:endpoint:v1`
2. **Cleanup**: Redis backend auto-cleans, memory backend has configurable cleanup
3. **Error Handling**: Decide whether to fail open or closed on backend errors
4. **Monitoring**: Use callbacks to track rate limiting behavior
5. **Resource Management**: Always call `Close()` to clean up resources

## Examples

See the [examples](examples/) directory for complete working examples:

- [Basic Usage](examples/basic/main.go) - Simple in-memory rate limiting
- [Redis Backend](examples/redis/main.go) - Redis-backed distributed rate limiting  
- [Web Server](examples/webserver/main.go) - HTTP API with rate limiting

## Testing

Run tests with:

```bash
go test ./...
```

For Redis tests, ensure Redis is running on localhost:6379:

```bash
# Start Redis with Docker
docker run -d -p 6379:6379 redis:7-alpine

# Run all tests including Redis
go test -tags=redis ./...
```

## License

MIT License - see [LICENSE](../../LICENSE) file for details.

## Contributing

Contributions are welcome! Please see our [Contributing Guide](../../CONTRIBUTING.md) for details.