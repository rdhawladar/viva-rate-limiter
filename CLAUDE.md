# CLAUDE.md - Essential Context for Claude Code

## Project Overview
This is a **Rate-Limited API Key Manager** - a production-ready Go system for managing API keys with sophisticated rate limiting. The project is currently in the planning phase with comprehensive architecture documentation.

## Critical Information
- **Language**: Go 1.21+
- **Status**: Architecture & Documentation Phase (No code implementation yet)
- **Memory Bank**: Complete documentation in `./docs/memory_bank/`
- **Project Name**: Viva Rate Limiter (formerly Curi)

## Essential Commands

### Development Environment Setup
```bash
# Install Go dependencies (when go.mod exists)
go mod download
go mod tidy

# Run tests (when implemented)
go test ./...
go test -v -race ./...

# Build the project (when main.go exists)
go build -o bin/api cmd/api/main.go
go build -o bin/worker cmd/worker/main.go

# Run with Docker Compose
docker-compose up -d
docker-compose logs -f

# Database migrations (when implemented)
migrate -path migrations -database "postgresql://user:pass@localhost/ratelimiter?sslmode=disable" up
```

### Code Quality Commands
```bash
# Format code
go fmt ./...
gofmt -w .

# Lint code (install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
golangci-lint run

# Generate mocks (install: go install github.com/vektra/mockery/v2@latest)
mockery --all --output mocks/

# Check for vulnerabilities
go mod audit
```

## Architecture Quick Reference

### Project Structure
```
rate-limiter/
├── cmd/                    # Application entrypoints
│   ├── api/               # HTTP API server
│   └── worker/            # Background workers
├── internal/              # Private application code
│   ├── config/           # Configuration management
│   ├── controllers/      # HTTP handlers
│   ├── services/         # Business logic
│   ├── repositories/     # Data access layer
│   ├── models/          # Domain models
│   ├── middleware/      # HTTP middleware
│   ├── queue/          # Message queue handlers
│   ├── cache/          # Redis operations
│   └── metrics/        # Prometheus metrics
├── pkg/                  # Public packages
│   ├── ratelimit/       # Rate limiting algorithms
│   ├── utils/          # Shared utilities
│   └── errors/         # Custom error types
├── migrations/          # Database migrations
├── configs/            # Environment configs
├── docker/             # Docker configurations
├── tests/              # Test suites
└── docs/              # Documentation
    └── memory_bank/   # Project knowledge base
```

### Key Technologies
- **Web Framework**: Gin (HTTP routing)
- **ORM**: GORM v2 (database access)
- **Database**: PostgreSQL 15 (primary + read replicas)
- **Cache**: Redis 7 with sharding
- **Queue**: RabbitMQ 3.12
- **Monitoring**: Prometheus + Grafana
- **Config**: Viper
- **Logging**: Zap (structured)
- **Testing**: Testify + Mockery

### Core Features
1. **API Key Management**: CRUD operations with secure storage
2. **Rate Limiting**: Sliding window algorithm with Redis
3. **Usage Tracking**: Real-time metrics and historical data
4. **Billing**: Automated usage calculations and overage handling
5. **Monitoring**: Prometheus metrics and custom alerts

### Critical Patterns
1. **Repository Pattern**: Data access abstraction
2. **Service Layer**: Business logic separation
3. **Middleware Chain**: Request processing pipeline
4. **Event-Driven**: RabbitMQ for async operations
5. **Caching Strategy**: Multi-level with Redis

## Development Workflow

### When Starting a New Session
1. Review `docs/memory_bank/activeContext.md` for current status
2. Check `docs/memory_bank/projectbrief.md` for project goals
3. Reference `docs/memory_bank/techContext.md` for technical details

### When Implementing Features
1. Follow patterns in `docs/memory_bank/systemPatterns.md`
2. Update `docs/memory_bank/activeContext.md` with progress
3. Use consistent error handling and logging patterns
4. Write tests alongside implementation

### Before Committing
1. Run `go fmt ./...` for formatting
2. Run `go test ./...` for tests
3. Run `golangci-lint run` for linting
4. Update relevant memory bank files

## Important Conventions

### Error Handling
```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create api key: %w", err)
}

// Use custom error types from pkg/errors
return errors.NewValidationError("invalid rate limit")
```

### Context Usage
```go
// Always pass context for cancellation
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
```

### Logging
```go
// Use structured logging with Zap
logger.Info("api key created",
    zap.String("key_id", key.ID),
    zap.String("tier", key.Tier),
)
```

## Performance Considerations
- Redis operations use pipelining for efficiency
- Database queries use prepared statements
- Connection pools are configured for high throughput
- Batch processing for analytics and billing

## Security Notes
- API keys stored as SHA256 hashes
- All communications use TLS
- Rate limit checks prevent timing attacks
- Audit logging for all operations

## Next Steps
1. Set up Go module and dependencies
2. Implement database models and migrations
3. Create core API endpoints
4. Implement rate limiting with Redis
5. Add monitoring and metrics

## Memory Bank Reference
For detailed information, consult the memory bank:
- Project overview: `docs/memory_bank/projectbrief.md`
- User flows: `docs/memory_bank/productContext.md`
- Technical details: `docs/memory_bank/techContext.md`
- Coding patterns: `docs/memory_bank/systemPatterns.md`
- Current work: `docs/memory_bank/activeContext.md`
- Progress tracking: `docs/memory_bank/progress.md`
- Best practices: `docs/memory_bank/developerNotes.md`