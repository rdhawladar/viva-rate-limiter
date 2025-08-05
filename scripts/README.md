# Viva Rate Limiter - Testing Guide

## Quick Start

### 1. Start the Services

```bash
# Start database and Redis
make docker-up

# Start API server (in terminal 1)
VIVA_SERVER_PORT=8090 make run-api

# Start worker (in terminal 2) 
make run-worker
```

### 2. Load Test Data

```bash
# Connect to PostgreSQL and run seed script
psql -h localhost -p 5433 -U viva_user -d viva_ratelimiter -f scripts/seed_data.sql
```

When prompted for password, use: `viva_password`

### 3. Test the API

```bash
# Test API health
go run scripts/test_client.go health

# Test with seed data API keys
go run scripts/test_client.go validate "test-key-free-123"
go run scripts/test_client.go validate "test-key-standard-456" 
go run scripts/test_client.go validate "test-key-pro-789"
```

## Test Data Reference

The seed script creates these test API keys:

| API Key | Tier | Rate Limit | Status | Hash (for X-API-Key header) |
|---------|------|------------|--------|------------------------------|
| `test-key-free-123` | free | 1000/hour | active | `9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08` |
| `test-key-standard-456` | standard | 10000/hour | active | `60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752e` |
| `test-key-pro-789` | pro | 100000/hour | active | `ef2d127de37b942baad06145e54b0c619a1f22327b2ebbcfbec78f5564afe39d` |
| `test-key-inactive-000` | basic | 5000/hour | inactive | `b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9` |

## API Testing Examples

### Health Check
```bash
curl http://localhost:8090/health
```

### Validate API Key (Public Endpoint)
```bash
curl -X POST http://localhost:8090/api/public/v1/rate-limit/validate \
  -H "Content-Type: application/json" \
  -d '{"api_key": "test-key-free-123"}'
```

### List API Keys (Requires Auth)
```bash
curl http://localhost:8090/api/v1/api-keys \
  -H "X-API-Key: 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
```

### Create New API Key
```bash
curl -X POST http://localhost:8090/api/v1/api-keys \
  -H "X-API-Key: 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My New Key",
    "description": "Created via API",
    "tier": "standard",
    "rate_limit": 5000
  }'
```

### Check Rate Limit
```bash
curl -X POST http://localhost:8090/api/v1/rate-limit/check \
  -H "X-API-Key: 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08" \
  -H "Content-Type: application/json" \
  -d '{
    "api_key_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "endpoint": "/test",
    "method": "GET"
  }'
```

### Get API Key Stats
```bash
curl http://localhost:8090/api/v1/api-keys/a1b2c3d4-e5f6-7890-abcd-ef1234567890/stats \
  -H "X-API-Key: 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
```

## Using the Test Client

### Check API Health
```bash
go run scripts/test_client.go health
```

### Validate API Keys
```bash
go run scripts/test_client.go validate "test-key-free-123"
go run scripts/test_client.go validate "test-key-standard-456"
go run scripts/test_client.go validate "invalid-key"
```

### List All API Keys
```bash
# Using the free tier key hash for authentication
go run scripts/test_client.go list "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
```

### Create New API Key
```bash
go run scripts/test_client.go create "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08" "My Test Key"
```

### Check Rate Limit
```bash
go run scripts/test_client.go rate-check "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
```

## Load Testing

### Simple Rate Limit Test
```bash
# Test rate limiting by making multiple requests quickly
for i in {1..15}; do
  echo "Request $i:"
  curl -X POST http://localhost:8090/api/public/v1/rate-limit/validate \
    -H "Content-Type: application/json" \
    -d '{"api_key": "test-key-free-123"}' 
  echo ""
  sleep 0.1
done
```

### Batch API Key Creation
```bash
# Create multiple API keys
AUTH_KEY="9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

for i in {1..5}; do
  curl -X POST http://localhost:8090/api/v1/api-keys \
    -H "X-API-Key: $AUTH_KEY" \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"Batch Key $i\",
      \"description\": \"Created in batch test\",
      \"tier\": \"basic\",
      \"rate_limit\": 2000
    }"
  echo ""
done
```

## Monitoring

### Check Service Health
```bash
# API health
curl http://localhost:8090/health | jq .

# Readiness probe  
curl http://localhost:8090/ready | jq .

# Liveness probe
curl http://localhost:8090/live | jq .
```

### View Logs
```bash
# API server logs
tail -f api.log

# Database logs
docker logs viva-postgres-dev

# Redis logs  
docker logs viva-redis-dev
```

### Database Queries
```bash
# Connect to database
psql -h localhost -p 5433 -U viva_user -d viva_ratelimiter

# Check data
SELECT name, tier, rate_limit, status FROM api_keys;
SELECT COUNT(*) FROM usage_logs;
SELECT COUNT(*) FROM rate_limit_violations;
```

## Troubleshooting

### Common Issues

1. **Port conflicts**: API runs on 8090, not 8080
2. **Database connection**: Ensure `make docker-up` ran successfully
3. **API key format**: Use SHA256 hash, not raw key for X-API-Key header
4. **Authentication**: Most `/api/v1/*` endpoints require X-API-Key header

### Reset Environment
```bash
# Stop services
make docker-down
pkill -f "go run cmd/api/main.go"

# Clean and restart
make docker-up
VIVA_SERVER_PORT=8090 make run-api
```

## Next Steps

1. **Add more test scenarios** for edge cases
2. **Set up integration tests** with the test client
3. **Create Postman collection** for easier API testing
4. **Add performance benchmarks** with tools like `ab` or `wrk`
5. **Implement monitoring dashboards** with Grafana