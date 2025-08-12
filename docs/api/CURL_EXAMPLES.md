# Viva Rate Limiter API - cURL Examples

Base URL: `http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com`

## Health & Status Endpoints

### Health Check
```bash
curl -X GET "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/health"
```

### Readiness Check
```bash
curl -X GET "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/ready"
```

### Liveness Check
```bash
curl -X GET "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/live"
```

### Prometheus Metrics
```bash
curl -X GET "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/metrics"
```

## Documentation

### Swagger UI
```bash
# Open in browser
open "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/swagger/"
```

### OpenAPI Spec
```bash
curl -X GET "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/openapi.yaml"
```

## API Key Management

### Create API Key
```bash
curl -X POST "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/v1/api-keys/" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test API Key",
    "description": "API key for testing purposes",
    "tier": "free",
    "rate_limit": {
      "requests": 1000,
      "window": "1h",
      "burst": 10
    },
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

### Get API Key Details
```bash
# Replace {api_key_id} with actual ID from create response
curl -X GET "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/v1/api-keys/{api_key_id}"
```

### List API Keys
```bash
curl -X GET "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/v1/api-keys/?limit=10&offset=0"
```

### Update API Key
```bash
curl -X PUT "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/v1/api-keys/{api_key_id}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Test API Key",
    "description": "Updated description for testing",
    "tier": "pro",
    "rate_limit": {
      "requests": 5000,
      "window": "1h",
      "burst": 50
    }
  }'
```

### Rotate API Key
```bash
curl -X POST "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/v1/api-keys/{api_key_id}/rotate" \
  -H "Content-Type: application/json" \
  -d '{}'
```

### Get API Key Statistics
```bash
curl -X GET "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/v1/api-keys/{api_key_id}/stats"
```

### Delete API Key
```bash
curl -X DELETE "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/v1/api-keys/{api_key_id}"
```

## Rate Limiting Operations

### Check Rate Limit
```bash
curl -X POST "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/v1/rate-limit/check" \
  -H "Content-Type: application/json" \
  -d '{
    "api_key": "your_api_key_here",
    "identifier": "user_123",
    "cost": 1
  }'
```

### Get Rate Limit Info
```bash
curl -X GET "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/v1/rate-limit/{api_key_id}/info"
```

### Update Rate Limit Configuration
```bash
curl -X PUT "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/v1/rate-limit/{api_key_id}" \
  -H "Content-Type: application/json" \
  -d '{
    "requests": 2000,
    "window": "1h",
    "burst": 20
  }'
```

### Reset Rate Limit Counters
```bash
curl -X POST "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/v1/rate-limit/{api_key_id}/reset" \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "user_123"
  }'
```

### Get Violation History
```bash
curl -X GET "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/v1/rate-limit/{api_key_id}/violations?limit=10&offset=0"
```

## Public API (Client Integration)

### Validate API Key & Check Rate Limit
```bash
# This is the main endpoint your clients would use
curl -X POST "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/public/v1/rate-limit/validate" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key_here" \
  -d '{
    "identifier": "user_123",
    "cost": 1
  }'
```

## Quick Testing Workflow

1. **Check API Health:**
```bash
curl "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/health"
```

2. **Create an API Key:**
```bash
API_RESPONSE=$(curl -s -X POST "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/v1/api-keys/" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Quick Test Key",
    "tier": "free",
    "rate_limit": {
      "requests": 100,
      "window": "1h",
      "burst": 5
    }
  }')

echo "API Key created: $API_RESPONSE"
```

3. **Extract API Key and ID:**
```bash
API_KEY=$(echo $API_RESPONSE | jq -r '.key')
API_KEY_ID=$(echo $API_RESPONSE | jq -r '.id')
echo "API Key: $API_KEY"
echo "API Key ID: $API_KEY_ID"
```

4. **Test Rate Limiting:**
```bash
curl -X POST "http://aa1e4012f5f7a40958069e83128ba92a-1088395492.ap-southeast-1.elb.amazonaws.com/api/public/v1/rate-limit/validate" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "identifier": "test_user",
    "cost": 1
  }'
```

## Response Examples

### Successful Health Check
```json
{
  "status": "healthy",
  "timestamp": "2025-08-12T04:13:57.197790939Z",
  "version": "1.0.0",
  "services": {
    "database": {"status": "healthy", "latency": 201543},
    "redis": {"status": "healthy", "latency": 155772}
  }
}
```

### API Key Creation Response
```json
{
  "id": "01234567-89ab-cdef-0123-456789abcdef",
  "key": "viva_dev_abcdef123456...",
  "name": "Test API Key",
  "tier": "free",
  "created_at": "2025-08-12T04:15:00Z",
  "rate_limit": {
    "requests": 1000,
    "window": "1h",
    "burst": 10
  }
}
```

### Rate Limit Check Response
```json
{
  "allowed": true,
  "remaining": 999,
  "reset_time": "2025-08-12T05:15:00Z",
  "retry_after": null
}
```

## Notes

- Replace `{api_key_id}` with actual API key IDs from responses
- Replace `your_api_key_here` with actual API keys
- All timestamps are in ISO 8601 format
- Rate limit windows use Go duration format (e.g., "1h", "30m", "60s")
- The dev environment uses HTTP (not HTTPS) for testing purposes