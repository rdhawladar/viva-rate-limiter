# Viva Rate Limiter

A production-ready Go-based API key management system with sophisticated rate limiting capabilities, built for high-performance and scalability.

## Use as a Go Package

You can use the rate limiting functionality in your own Go projects:

```bash
go get github.com/rdhawladar/viva-rate-limiter/pkg/ratelimit
```

```go
import "github.com/rdhawladar/viva-rate-limiter/pkg/ratelimit"

// Create a rate limiter
limiter := ratelimit.NewSlidingWindow(redis.Client, 1000, time.Hour)

// Check rate limit
allowed, remaining := limiter.Allow(ctx, "user-key")
```

## Features

- **API Key Management**: Secure creation, rotation, and revocation of API keys
- **Advanced Rate Limiting**: Sliding window algorithm with Redis-backed distributed rate limiting
- **Multi-Tier Support**: Flexible tier-based rate limits (Free, Basic, Pro, Enterprise)
- **Real-time Usage Tracking**: Monitor API usage with detailed metrics and analytics
- **Automated Billing**: Usage-based billing with overage handling
- **High Availability**: PostgreSQL with read replicas and Redis sharding
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

### Install dependencies
```bash
go mod download
go mod tidy
```

### Set up environment variables
```bash
cp .env.example .env
# Edit .env with your configuration
```

### Run with Docker Compose
```bash
docker-compose up -d
```

### Run database migrations
```bash
migrate -path migrations -database "postgresql://user:pass@localhost/ratelimiter?sslmode=disable" up
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

### Create an API Key
```bash
curl -X POST http://localhost:8080/api/v1/keys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "name": "Production API Key",
    "tier": "pro",
    "rate_limit": 10000
  }'
```

### Check Rate Limit
```bash
curl -X POST http://localhost:8080/api/v1/rate-limit/check \
  -H "X-API-Key: YOUR_API_KEY"
```

### Get Usage Statistics
```bash
curl -X GET http://localhost:8080/api/v1/keys/{key_id}/usage \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

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
├── pkg/                  # Public packages
│   ├── ratelimit/       # Rate limiting algorithms
│   └── errors/         # Custom error types
├── migrations/          # Database migrations
├── configs/            # Environment configs
├── docker/             # Docker configurations
└── docs/              # Documentation
```

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
- [Architecture Overview](docs/memory_bank/architecture.md)
- [API Reference](docs/api/)
- [Development Guide](docs/memory_bank/developerNotes.md)
- [Deployment Guide](docs/deployment.md)

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
