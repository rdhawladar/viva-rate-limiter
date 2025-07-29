# Viva Rate Limiter Makefile

.PHONY: help build run test clean docker-up docker-down deps lint fmt vet

# Default target
help:
	@echo "Available commands:"
	@echo "  build       - Build the application binaries"
	@echo "  run-api     - Run the API server"
	@echo "  run-worker  - Run the background worker"
	@echo "  test        - Run tests"
	@echo "  test-race   - Run tests with race detection"
	@echo "  clean       - Clean build artifacts"
	@echo "  docker-up   - Start Docker services"
	@echo "  docker-down - Stop Docker services"
	@echo "  deps        - Download and verify dependencies"
	@echo "  lint        - Run linter"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  migrate-up  - Run database migrations up"
	@echo "  migrate-down- Run database migrations down"

# Build targets
build:
	@echo "Building Viva Rate Limiter..."
	@mkdir -p bin
	go build -o bin/viva-api cmd/api/main.go
	go build -o bin/viva-worker cmd/worker/main.go

# Run targets
run-api:
	@echo "Starting API server..."
	go run cmd/api/main.go

run-worker:
	@echo "Starting background worker..."
	go run cmd/worker/main.go

# Test targets
test:
	go test ./...

test-race:
	go test -race ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Development targets
deps:
	go mod download
	go mod tidy
	go mod verify

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

fmt:
	go fmt ./...
	gofmt -w .

vet:
	go vet ./...

# Docker targets
docker-up:
	@echo "Starting Docker services..."
	docker-compose -f docker-compose.dev.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Services started. Check health with: docker-compose -f docker-compose.dev.yml ps"

docker-down:
	@echo "Stopping Docker services..."
	docker-compose -f docker-compose.dev.yml down

docker-logs:
	docker-compose -f docker-compose.dev.yml logs -f

docker-up-full:
	@echo "Starting full Docker services (with monitoring)..."
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 15
	@echo "Services started. Grafana: http://localhost:3001 (admin/admin)"
	@echo "PostgreSQL: localhost:5434, Redis: localhost:6381"

run-api-full:
	@echo "Starting API server with full stack config..."
	VIVA_ENV=full go run cmd/api/main.go

run-worker-full:
	@echo "Starting background worker with full stack config..."
	VIVA_ENV=full go run cmd/worker/main.go

docker-down-full:
	@echo "Stopping full Docker services..."  
	docker-compose down

# Database targets
migrate-up:
	@echo "Running database migrations up..."
	@if [ -f bin/viva-api ]; then \
		./bin/viva-api migrate up; \
	else \
		go run cmd/api/main.go migrate up; \
	fi

migrate-down:
	@echo "Running database migrations down..."
	@if [ -f bin/viva-api ]; then \
		./bin/viva-api migrate down; \
	else \
		go run cmd/api/main.go migrate down; \
	fi

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Usage: make migrate-create name=migration_name"; \
		exit 1; \
	fi
	@echo "Creating migration: $(name)"
	@timestamp=$$(date +%Y%m%d%H%M%S); \
	touch migrations/$${timestamp}_$(name).up.sql; \
	touch migrations/$${timestamp}_$(name).down.sql; \
	echo "Created migrations/$${timestamp}_$(name).up.sql"; \
	echo "Created migrations/$${timestamp}_$(name).down.sql"

# Clean targets
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -cache
	go clean -modcache

# Development workflow
dev-setup: deps docker-up
	@echo "Development environment setup complete!"
	@echo "Run 'make run-api' to start the API server"

dev-reset: docker-down clean docker-up

# Quality checks
check: fmt vet lint test

# Production build
build-prod:
	@echo "Building for production..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o bin/viva-api cmd/api/main.go
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o bin/viva-worker cmd/worker/main.go