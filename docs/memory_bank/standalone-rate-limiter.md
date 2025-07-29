# Standalone Rate Limiter Service

## Overview
The Viva Rate Limiter can function as both:
1. **Complete API Key Management System** - Full-featured API key lifecycle management with billing
2. **Standalone Rate Limiter Service** - Lightweight, high-performance rate limiting for any application

## Standalone Rate Limiter Features

### Core Capabilities
- **Universal Rate Limiting**: Works with any identifier (API keys, IPs, user IDs, tokens)
- **Multiple Algorithms**: Sliding window, token bucket, leaky bucket, fixed window
- **Distributed Architecture**: Redis-based for horizontal scaling
- **Sub-millisecond Performance**: Optimized for minimal latency
- **Flexible Integration**: REST API, gRPC, or native Go SDK

### Key Differentiators
- No API key management overhead when used standalone
- Lightweight deployment option (single binary)
- Protocol-agnostic (HTTP, gRPC, WebSocket, TCP)
- Multi-tenant support with namespace isolation
- Real-time configuration updates without restarts

## Architecture for Standalone Mode

### Simplified Components
```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Your Service   │────▶│  Rate Limiter    │────▶│  Redis Cluster  │
│  (Any Language) │     │  (HTTP/gRPC)     │     │  (Sharded)      │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                │
                                ▼
                        ┌──────────────────┐
                        │  Configuration   │
                        │  (Dynamic)       │
                        └──────────────────┘
```

### Standalone API Endpoints

#### 1. Check Rate Limit
```
POST /v1/check
{
  "identifier": "user_123",           // Any unique identifier
  "namespace": "api_requests",        // Logical grouping
  "operation": "read",               // Optional operation type
  "cost": 1                          // Optional cost (default: 1)
}

Response:
{
  "allowed": true,
  "limit": 1000,
  "remaining": 567,
  "reset_at": "2024-01-15T11:00:00Z",
  "retry_after": null
}
```

#### 2. Check and Decrement (Atomic)
```
POST /v1/consume
{
  "identifier": "user_123",
  "namespace": "api_requests",
  "cost": 10
}
```

#### 3. Configure Limits
```
PUT /v1/limits
{
  "namespace": "api_requests",
  "rules": [
    {
      "identifier_pattern": "user_*",
      "limit": 1000,
      "window": "1h",
      "algorithm": "sliding_window"
    },
    {
      "identifier_pattern": "ip_*",
      "limit": 100,
      "window": "1m",
      "algorithm": "token_bucket"
    }
  ]
}
```

#### 4. Get Usage Stats
```
GET /v1/stats/{namespace}/{identifier}
```

### gRPC Service Definition
```protobuf
service RateLimiter {
  rpc Check(CheckRequest) returns (CheckResponse);
  rpc Consume(ConsumeRequest) returns (ConsumeResponse);
  rpc Configure(ConfigureRequest) returns (ConfigureResponse);
  rpc GetStats(StatsRequest) returns (StatsResponse);
  rpc StreamStats(StreamStatsRequest) returns (stream StatsUpdate);
}

message CheckRequest {
  string identifier = 1;
  string namespace = 2;
  string operation = 3;
  int32 cost = 4;
}

message CheckResponse {
  bool allowed = 1;
  int32 limit = 2;
  int32 remaining = 3;
  int64 reset_at = 4;
  int32 retry_after = 5;
}
```

## Integration Patterns

### 1. HTTP Middleware (Go)
```go
func RateLimitMiddleware(limiter *ratelimit.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        identifier := c.GetHeader("X-User-ID")
        if identifier == "" {
            identifier = c.ClientIP()
        }
        
        resp, err := limiter.Check(c.Request.Context(), &ratelimit.CheckRequest{
            Identifier: identifier,
            Namespace:  "http_requests",
            Operation:  c.Request.Method,
        })
        
        if err != nil || !resp.Allowed {
            c.Header("X-RateLimit-Limit", fmt.Sprint(resp.Limit))
            c.Header("X-RateLimit-Remaining", fmt.Sprint(resp.Remaining))
            c.Header("Retry-After", fmt.Sprint(resp.RetryAfter))
            c.AbortWithStatus(429)
            return
        }
        
        c.Next()
    }
}
```

### 2. Service Mesh Integration (Envoy)
```yaml
http_filters:
- name: envoy.filters.http.ratelimit
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit
    domain: production
    rate_limit_service:
      grpc_service:
        envoy_grpc:
          cluster_name: rate_limit_cluster
```

### 3. SDK Examples

#### Go SDK
```go
import "github.com/viva/ratelimiter-go"

client := ratelimiter.New(ratelimiter.Config{
    Endpoint: "localhost:8080",
    Timeout:  100 * time.Millisecond,
})

allowed, err := client.Allow(ctx, "user_123", "api_requests")
if !allowed {
    return errors.New("rate limit exceeded")
}
```

#### Python SDK
```python
from viva_ratelimiter import RateLimiter

limiter = RateLimiter(endpoint="localhost:8080")

if not limiter.allow("user_123", "api_requests"):
    raise Exception("Rate limit exceeded")
```

#### Node.js SDK
```javascript
const RateLimiter = require('@viva/ratelimiter');

const limiter = new RateLimiter({
  endpoint: 'localhost:8080'
});

const allowed = await limiter.check('user_123', 'api_requests');
if (!allowed) {
  throw new Error('Rate limit exceeded');
}
```

## Deployment Options

### 1. Lightweight Container
```dockerfile
FROM scratch
COPY rate-limiter /
EXPOSE 8080 9090
ENTRYPOINT ["/rate-limiter", "--mode=standalone"]
```

### 2. Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rate-limiter
spec:
  replicas: 3
  selector:
    matchLabels:
      app: rate-limiter
  template:
    metadata:
      labels:
        app: rate-limiter
    spec:
      containers:
      - name: rate-limiter
        image: viva/rate-limiter:latest
        args: ["--mode=standalone"]
        ports:
        - containerPort: 8080  # HTTP/gRPC
        - containerPort: 9090  # Metrics
        env:
        - name: REDIS_CLUSTER
          value: "redis-cluster:6379"
```

### 3. Docker Compose
```yaml
version: '3.8'
services:
  rate-limiter:
    image: viva/rate-limiter:latest
    command: ["--mode=standalone"]
    ports:
      - "8080:8080"
    environment:
      - REDIS_ADDR=redis:6379
      - LOG_LEVEL=info
    depends_on:
      - redis
  
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

## Configuration

### Standalone Mode Config
```yaml
mode: standalone  # or "full" for complete API key management

rate_limiter:
  algorithms:
    - sliding_window
    - token_bucket
    - leaky_bucket
    - fixed_window
  
  default_limits:
    - namespace: "default"
      limit: 1000
      window: "1h"
      algorithm: "sliding_window"
  
  redis:
    mode: "cluster"  # or "single"
    addresses:
      - "redis-1:6379"
      - "redis-2:6379"
      - "redis-3:6379"
    
  performance:
    connection_pool_size: 100
    pipeline_size: 50
    async_workers: 10

api:
  http_port: 8080
  grpc_port: 8081
  metrics_port: 9090
  
monitoring:
  prometheus:
    enabled: true
    namespace: "viva_ratelimiter"
```

## Performance Benchmarks

### Standalone Mode Performance
- **Latency**: < 0.5ms p99 (local Redis)
- **Throughput**: > 200k checks/second (single instance)
- **Memory**: < 50MB baseline
- **CPU**: < 10% at 100k ops/sec

### Comparison with Popular Solutions

| Feature | Viva Rate Limiter | Redis Cell | Nginx | HAProxy |
|---------|------------------|------------|--------|----------|
| Distributed | ✓ | ✓ | ✗ | ✗ |
| Multiple Algorithms | ✓ | ✗ | ✗ | ✗ |
| Dynamic Config | ✓ | ✗ | ✗ | ✗ |
| gRPC Support | ✓ | ✗ | ✗ | ✗ |
| Sub-ms Latency | ✓ | ✓ | ✓ | ✓ |
| Multi-tenant | ✓ | ✗ | ✗ | ✗ |

## Migration Guide

### From nginx rate limiting
```nginx
# Before (nginx)
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
limit_req zone=api burst=20;

# After (Viva)
curl -X PUT http://rate-limiter/v1/limits -d '{
  "namespace": "api",
  "rules": [{
    "identifier_pattern": "ip_*",
    "limit": 10,
    "window": "1s",
    "burst": 20
  }]
}'
```

### From application-level rate limiting
```go
// Before (in-app)
if rateLimiter.IsAllowed(userID) {
    // process request
}

// After (Viva SDK)
if client.Allow(ctx, userID, "api") {
    // process request
}
```

## Use Cases

1. **API Gateway Rate Limiting**
   - Front your APIs with rate limiting
   - No code changes required
   - Works with Kong, Traefik, Envoy

2. **Microservice Protection**
   - Service-to-service rate limiting
   - Prevent cascade failures
   - Circuit breaker patterns

3. **Database Query Throttling**
   - Limit expensive queries
   - Protect database resources
   - Per-user query budgets

4. **WebSocket Connection Limiting**
   - Limit connections per user
   - Message rate limiting
   - Bandwidth throttling

5. **Background Job Rate Limiting**
   - Control job processing rates
   - Prevent resource exhaustion
   - Fair scheduling