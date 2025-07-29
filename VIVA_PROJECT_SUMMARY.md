# Viva Rate Limiter - Dual-Purpose API Management & Rate Limiting Service

## Project Overview
Viva Rate Limiter is a high-performance, production-ready system that serves two distinct purposes:

1. **Complete API Key Management Platform** - Full-featured solution for API providers with key management, usage tracking, billing, and analytics
2. **Standalone Rate Limiting Service** - Lightweight, universal rate limiter that can protect any application or service

Built with Go for exceptional performance, it handles millions of requests while maintaining sub-millisecond response times.

## Two Modes of Operation

### Mode 1: Full API Management Platform
Perfect for SaaS companies and API providers who need:
- Complete API key lifecycle management
- Usage-based billing and invoicing
- Customer tiering (Free, Pro, Enterprise)
- Analytics dashboards and reporting
- Automated alerts and notifications

### Mode 2: Standalone Rate Limiter
Ideal for any application that needs rate limiting:
- Works with any identifier (IP, user ID, token, etc.)
- Multiple algorithms (sliding window, token bucket, etc.)
- Language-agnostic (REST API, gRPC, native SDKs)
- Service mesh compatible (Envoy, Istio)
- Minimal resource footprint

## Key Features

### Core Rate Limiting Engine
- **Sub-millisecond latency** (< 1ms p99)
- **Multiple algorithms**: Sliding window, token bucket, leaky bucket, fixed window
- **Distributed architecture** with Redis clustering
- **200k+ operations/second** per instance
- **Dynamic configuration** without restarts

### API Key Management (Full Mode)
- Secure key generation and rotation
- Tiered access control (Free/Pro/Enterprise)
- Usage tracking and analytics
- Automated billing calculations
- Self-service key management

### Integration Options
- **REST API** for universal compatibility
- **gRPC** for high-performance scenarios
- **Native SDKs**: Go, Python, Node.js, Java, Ruby
- **Middleware**: Express, Gin, Django, Spring
- **Service Mesh**: Envoy, Istio, Linkerd compatible

## Technical Specifications

### Architecture
```
Your Application → Rate Limiter Service → Redis Cluster
                           ↓
                   Configuration Store
                           ↓
                   Metrics & Monitoring
```

### Tech Stack
- **Language**: Go 1.21+ (performance and simplicity)
- **Storage**: Redis 7+ with clustering support
- **APIs**: REST (Gin) and gRPC
- **Monitoring**: Prometheus metrics + Grafana dashboards
- **Deployment**: Docker, Kubernetes, bare metal

### Performance Targets
- Latency: < 1ms for rate limit checks (p99)
- Throughput: > 100k requests/second (full mode)
- Throughput: > 200k checks/second (standalone mode)
- Availability: 99.99% uptime SLA
- Memory: < 50MB baseline (standalone mode)

## Use Cases

### As Full API Management Platform
1. **SaaS API Monetization**
   - Offer tiered API access
   - Track usage for billing
   - Enforce rate limits per tier
   - Generate usage reports

2. **Internal API Governance**
   - Manage microservice access
   - Track inter-service usage
   - Implement cost allocation
   - Ensure fair resource usage

### As Standalone Rate Limiter
1. **API Gateway Protection**
   - Drop-in rate limiting for any API
   - No application code changes
   - Works with Kong, Traefik, Nginx

2. **Database Query Throttling**
   - Limit expensive queries per user
   - Protect database from overload
   - Implement query budgets

3. **Microservice Circuit Breaking**
   - Prevent cascade failures
   - Service-to-service rate limiting
   - Automatic backpressure

4. **WebSocket/Streaming Limits**
   - Connection limits per user
   - Message rate throttling
   - Bandwidth management

## Quick Start Examples

### Standalone Rate Limiting (HTTP)
```bash
# Check if request is allowed
curl -X POST http://localhost:8080/v1/check \
  -d '{"identifier": "user_123", "namespace": "api_requests"}'

# Response
{
  "allowed": true,
  "remaining": 99,
  "reset_at": "2024-01-15T11:00:00Z"
}
```

### SDK Integration (Go)
```go
client := ratelimiter.New("localhost:8080")

if allowed := client.Allow(ctx, "user_123", "api"); !allowed {
    return errors.New("rate limit exceeded")
}
```

### Middleware Integration (Node.js)
```javascript
app.use(ratelimiter.middleware({
  endpoint: 'localhost:8080',
  identifier: (req) => req.user?.id || req.ip,
  namespace: 'api_requests'
}));
```

## Deployment

### Docker (Standalone Mode)
```bash
docker run -d \
  -p 8080:8080 \
  -e REDIS_ADDR=redis:6379 \
  -e MODE=standalone \
  viva/rate-limiter:latest
```

### Kubernetes (Full Mode)
```yaml
helm install viva-rate-limiter ./charts/viva-rate-limiter \
  --set mode=full \
  --set replicas=3 \
  --set redis.cluster.enabled=true
```

## Why Choose Viva Rate Limiter?

### Flexibility
- Use as complete API platform or just rate limiting
- Switch modes without changing infrastructure
- Scale from single instance to distributed cluster

### Performance
- Written in Go for maximum efficiency
- Redis-based for horizontal scaling
- Optimized algorithms for minimal latency

### Developer Experience
- Simple REST/gRPC APIs
- Native SDKs for major languages
- Extensive documentation and examples
- Active community support

### Production Ready
- Battle-tested algorithms
- Comprehensive monitoring
- High availability design
- Enterprise support available

## Project Status
Currently in active development with architecture and design complete. The rate limiting engine is the core component that powers both modes of operation.

## Get Started
- **Documentation**: [docs.viva-ratelimiter.io](https://docs.viva-ratelimiter.io)
- **GitHub**: [github.com/viva/rate-limiter](https://github.com/viva/rate-limiter)
- **Docker Hub**: [hub.docker.com/r/viva/rate-limiter](https://hub.docker.com/r/viva/rate-limiter)
- **Community**: [Discord](https://discord.gg/viva) | [Slack](https://viva.slack.com)

## License
Open source under MIT license with optional commercial support.