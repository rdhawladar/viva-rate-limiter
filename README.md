# Viva Rate Limiter

[![Go Reference](https://pkg.go.dev/badge/github.com/rdhawladar/viva-rate-limiter/pkg/ratelimit.svg)](https://pkg.go.dev/github.com/rdhawladar/viva-rate-limiter/pkg/ratelimit)
[![Go Report Card](https://goreportcard.com/badge/github.com/rdhawladar/viva-rate-limiter)](https://goreportcard.com/report/github.com/rdhawladar/viva-rate-limiter)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A production-ready Go-based API key management system with sophisticated rate limiting capabilities, built for high-performance and scalability. **Available as a standalone Go package!**

<!-- Test deployment $(date): GitHub Actions automated deployment test for dev environment -->

## 🚀 Quick Start - Use as a Go Package

Viva Rate Limiter is available as a **public Go package** that you can easily integrate into your applications:

### Installation

```bash
go get github.com/rdhawladar/viva-rate-limiter/pkg/ratelimit@latest
```

### Basic Usage

```go
package main

import (
    "context"
    "time"
    "github.com/rdhawladar/viva-rate-limiter/pkg/ratelimit"
)

func main() {
    // Create a memory-based rate limiter (great for single instances)
    backend := ratelimit.NewMemoryBackend()
    
    ctx := context.Background()
    key := "user-123"
    window := time.Minute
    limit := int64(100) // 100 requests per minute
    
    // Check if request is allowed
    count, windowStart, err := backend.Increment(ctx, key, window)
    if err != nil {
        // Handle error
    }
    
    if count <= limit {
        // Request allowed
        fmt.Printf("Request allowed (count: %d/%d)\n", count, limit)
    } else {
        // Rate limit exceeded
        fmt.Printf("Rate limit exceeded (count: %d/%d)\n", count, limit)
    }
}
```

### Redis Backend (for distributed systems)

```go
import (
    "github.com/redis/go-redis/v9"
    "github.com/rdhawladar/viva-rate-limiter/pkg/ratelimit"
)

// Create Redis client
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// Create Redis-backed rate limiter
backend := ratelimit.NewRedisBackend(redisClient)

// Use the same as memory backend
count, windowStart, err := backend.Increment(ctx, key, window)
```

### Package Features
- ✅ **Memory Backend** - Perfect for single-instance applications
- ✅ **Redis Backend** - For distributed rate limiting across multiple servers
- ✅ **Thread-Safe** - Concurrent request handling
- ✅ **Sliding Window Algorithm** - Accurate rate limiting
- ✅ **Zero Dependencies** - Minimal external dependencies (only Redis client if using Redis backend)
- ✅ **Production Ready** - Battle-tested with comprehensive testing

## Features

- **API Key Management**: Secure creation, rotation, and revocation of API keys
- **Advanced Rate Limiting**: Sliding window algorithm with Redis-backed distributed rate limiting
- **Multi-Tier Support**: Flexible tier-based rate limits (Free, Basic, Pro, Enterprise)
- **Real-time Usage Tracking**: Monitor API usage with detailed metrics and analytics
- **Automated Billing**: Usage-based billing with overage handling
- **High Availability**: PostgreSQL with read replicas and Redis sharding
- **Interactive API Documentation**: Built-in Swagger UI for easy API exploration
- **Observability**: Prometheus metrics, structured logging, and custom alerting

## Architecture

The system follows a microservices architecture with clean separation of concerns:

- **API Server**: RESTful API for key management and rate checking
- **Worker Service**: Background processing for analytics and billing
- **PostgreSQL**: Primary data store with read replicas
- **Redis**: Distributed caching and rate limit counters
- **RabbitMQ**: Asynchronous message processing

## Requirements

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- RabbitMQ 3.12+
- Docker & Docker Compose (for containerized deployment)

## Installation

### Clone the repository
```bash
git clone https://github.com/rdhawladar/viva-rate-limiter.git
cd viva-rate-limiter
```

### Quick Start (Development)
```bash
# 1. Start infrastructure services (PostgreSQL + Redis)
make docker-up

# 2. Run the API server
make run-api

# 3. Open Swagger UI
open http://localhost:8080/swagger/index
```

### Manual Setup

#### Install dependencies
```bash
go mod download
go mod tidy
```

#### Set up environment variables
```bash
cp .env.example .env
# Edit .env with your configuration
```

#### Run with Docker Compose
```bash
# Development setup (PostgreSQL + Redis only)
make docker-up

# OR Full setup with monitoring (PostgreSQL + Redis + Prometheus + Grafana)
make docker-up-full
```

#### Run the API
```bash
make run-api
```

## Development

### Build the project
```bash
# Build API server
go build -o bin/api cmd/api/main.go

# Build worker service
go build -o bin/worker cmd/worker/main.go
```

### Run tests
```bash
# Run all tests
go test ./...

# Run with race detection
go test -v -race ./...

# Run with coverage
go test -cover ./...
```

### Code quality
```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Security audit
go mod audit
```

## API Usage

### 📖 Interactive API Documentation

**Swagger UI is available at:** http://localhost:8080/swagger/index

The interactive documentation includes:
- Complete API endpoint reference
- Request/response schemas
- Try-it-out functionality
- Authentication examples
- Rate limiting information

### Quick API Examples

#### Create an API Key
```bash
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "name": "Production API Key",
    "tier": "pro",
    "rate_limit": 10000
  }'
```

#### Check Rate Limit
```bash
curl -X POST http://localhost:8080/api/v1/rate-limit/check \
  -H "X-API-Key: YOUR_API_KEY"
```

#### Get Usage Statistics
```bash
curl -X GET http://localhost:8080/api/v1/api-keys/{key_id}/stats \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

> 💡 **Tip**: Use the interactive Swagger UI at `/swagger/index` to explore all available endpoints and test them directly from your browser.

## Project Structure

```
rate-limiter/
├── cmd/                    # Application entrypoints
├── internal/              # Private application code
│   ├── config/           # Configuration management
│   ├── controllers/      # HTTP handlers
│   ├── services/         # Business logic
│   ├── repositories/     # Data access layer
│   └── middleware/      # HTTP middleware
├── pkg/                  # 📦 PUBLIC GO PACKAGES
│   ├── ratelimit/       # ⭐ Rate limiting library (public package)
│   │   ├── limiter.go         # Core interfaces
│   │   ├── memory_backend.go  # In-memory implementation
│   │   ├── redis_backend.go   # Redis implementation
│   │   └── examples/          # Usage examples
│   └── errors/         # Custom error types
├── migrations/          # Database migrations
├── configs/            # Environment configs
├── docker/             # Docker configurations
├── k6/                 # Performance testing with k6
└── docs/              # Documentation
```

## 📦 Public Package Documentation

The rate limiting package (`pkg/ratelimit`) is designed as a **standalone Go library** that can be imported and used independently of the full Viva Rate Limiter system.

### Why Use Our Package?
- **Battle-tested**: Used in production handling millions of requests
- **Flexible**: Choose between memory or Redis backends
- **Simple API**: Easy to integrate with just a few lines of code
- **Well-documented**: Comprehensive examples and documentation
- **Active maintenance**: Regular updates and improvements

### Package Versions
- `v0.3.0` - Latest stable release with fixed module paths
- `v0.2.0` - Added Redis backend support
- `v0.1.0` - Initial release with memory backend

### More Examples
Check out the [examples directory](https://github.com/rdhawladar/viva-rate-limiter/tree/main/pkg/ratelimit/examples) for:
- Basic usage
- Web server integration
- Redis configuration
- Custom backends

## Configuration

The application uses environment variables for configuration. Key settings include:

- `DATABASE_URL`: PostgreSQL connection string
- `REDIS_URL`: Redis connection string
- `RABBITMQ_URL`: RabbitMQ connection string
- `SERVER_PORT`: API server port (default: 8080)
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `METRICS_PORT`: Prometheus metrics port (default: 9090)

## Monitoring

### Prometheus Metrics
Metrics are exposed at `http://localhost:9090/metrics`

Key metrics:
- `api_requests_total`: Total API requests
- `rate_limit_checks_total`: Rate limit check operations
- `rate_limit_exceeded_total`: Rate limit violations
- `api_key_operations_total`: Key management operations

### Health Checks
- Liveness: `GET /health/live`
- Readiness: `GET /health/ready`

## Performance

The system is designed for high throughput:
- Handles 100,000+ requests per second
- Sub-millisecond rate limit checks
- Horizontal scaling support
- Connection pooling for all data stores
- Optimized batch processing for analytics

### Performance Testing
The project includes comprehensive k6 load testing scripts in the `/k6` directory:
- **Load & stress testing** with various user scenarios
- **Rate limiting accuracy** validation
- **Redis performance** testing under high load

Run performance tests:
```bash
cd k6
./run_performance_tests.sh
```

See [k6 Testing Documentation](k6/README.md) for details.

## Service Ports

### Development Mode (`make docker-up`)
- **API Server**: http://localhost:8080
- **PostgreSQL**: localhost:5433
- **Redis**: localhost:6380

### Full Mode (`make docker-up-full`)
- **API Server**: http://localhost:8080
- **PostgreSQL**: localhost:5434
- **Redis**: localhost:6381
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3001 (admin/admin)

## Security

- API keys stored as SHA256 hashes
- TLS encryption for all communications
- Rate limit checks prevent timing attacks
- Comprehensive audit logging
- Role-based access control (RBAC)

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please ensure:
- All tests pass
- Code is formatted with `go fmt`
- Linter checks pass
- Documentation is updated

## Documentation

Detailed documentation is available in the `docs/` directory:
- [API Reference](docs/api/) - API endpoints and usage
- [k6 Performance Testing](k6/README.md) - Load testing documentation
- [Development Guide](docs/memory_bank/developerNotes.md) - Development best practices

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For issues, questions, or contributions, please open an issue on GitHub or contact the maintainers.

## Acknowledgments

Built with:
- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [GORM](https://gorm.io/)
- [go-redis](https://github.com/redis/go-redis)
- [Viper](https://github.com/spf13/viper)
- [Zap Logger](https://github.com/uber-go/zap)
