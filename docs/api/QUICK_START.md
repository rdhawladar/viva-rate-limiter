# Quick Start Guide

Get up and running with the Viva Rate Limiter API in 5 minutes.

## 1. Start the API Server

```bash
# Clone the repository
git clone https://github.com/viva/rate-limiter
cd rate-limiter

# Start dependencies
docker-compose up -d postgres redis

# Run the API server
make run-api
# or
go run cmd/api/main.go
```

The API will be available at `http://localhost:8090`

## 2. Check Health

Verify the API is running:

```bash
curl http://localhost:8090/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "services": {
    "database": "healthy",
    "redis": "healthy"
  }
}
```

## 3. Use Test API Keys

For quick testing, use these pre-seeded API keys:

| Key | Tier | Rate Limit |
|-----|------|------------|
| `test-key-free-123` | Free | 100/hour |
| `test-key-standard-456` | Standard | 1,000/hour |
| `test-key-pro-789` | Pro | 10,000/hour |

## 4. Check Rate Limits

Test rate limiting with a simple request:

```bash
curl -X POST http://localhost:8090/api/public/v1/rate-limit/validate \
  -H "Content-Type: application/json" \
  -d '{
    "api_key": "test-key-free-123",
    "requests": 1
  }'
```

Expected response:
```json
{
  "allowed": true,
  "limit": 100,
  "remaining": 99,
  "reset_time": "2024-01-15T11:00:00Z",
  "reset_in_seconds": 3540
}
```

## 5. Create Your Own API Key

Create a new API key for your application:

```bash
curl -X POST http://localhost:8090/api/v1/api-keys \
  -H "X-API-Key: test-key-pro-789" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Application",
    "tier": "standard",
    "description": "API key for my awesome app"
  }'
```

Response:
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "key": "viva_live_sk_a1b2c3d4e5f6...",
    "name": "My Application",
    "tier": "standard",
    "status": "active",
    "rate_limit": 1000,
    "rate_limit_window": "1h",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

⚠️ **Important**: Save the `key` value - it's only shown once!

## 6. Monitor Usage

Check your API key usage:

```bash
curl -H "X-API-Key: your-new-api-key" \
  http://localhost:8090/api/v1/rate-limit/info
```

Response:
```json
{
  "data": {
    "limit": 1000,
    "remaining": 999,
    "used": 1,
    "window": "1h",
    "reset_time": "2024-01-15T11:00:00Z"
  }
}
```

## Integration Examples

### Go Application

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

const RateLimiterAPI = "http://localhost:8090"

func checkRateLimit(apiKey string) (bool, error) {
    payload := map[string]interface{}{
        "api_key": apiKey,
        "requests": 1,
    }
    
    jsonData, _ := json.Marshal(payload)
    
    resp, err := http.Post(
        RateLimiterAPI+"/api/public/v1/rate-limit/validate",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == 429 {
        return false, nil // Rate limited
    }
    
    return resp.StatusCode == 200, nil
}

func main() {
    allowed, err := checkRateLimit("test-key-free-123")
    if err != nil {
        panic(err)
    }
    
    if allowed {
        fmt.Println("Request allowed!")
    } else {
        fmt.Println("Rate limited!")
    }
}
```

### Python Flask Middleware

```python
import requests
from flask import Flask, request, jsonify
from functools import wraps

app = Flask(__name__)

def rate_limit_required(f):
    @wraps(f)
    def decorated_function(*args, **kwargs):
        api_key = request.headers.get('X-API-Key')
        if not api_key:
            return jsonify({'error': 'API key required'}), 401
        
        # Check rate limit
        response = requests.post(
            'http://localhost:8090/api/public/v1/rate-limit/validate',
            json={'api_key': api_key, 'requests': 1}
        )
        
        if response.status_code == 429:
            return jsonify({'error': 'Rate limit exceeded'}), 429
        
        return f(*args, **kwargs)
    return decorated_function

@app.route('/api/data')
@rate_limit_required
def get_data():
    return jsonify({'message': 'Here is your data!'})
```

### Node.js Express Middleware

```javascript
const express = require('express');
const axios = require('axios');

const app = express();

const rateLimitMiddleware = async (req, res, next) => {
    const apiKey = req.headers['x-api-key'];
    
    if (!apiKey) {
        return res.status(401).json({ error: 'API key required' });
    }
    
    try {
        const response = await axios.post(
            'http://localhost:8090/api/public/v1/rate-limit/validate',
            { api_key: apiKey, requests: 1 }
        );
        
        if (response.data.allowed) {
            next();
        } else {
            res.status(429).json({ error: 'Rate limit exceeded' });
        }
    } catch (error) {
        if (error.response && error.response.status === 429) {
            res.status(429).json({ error: 'Rate limit exceeded' });
        } else {
            res.status(500).json({ error: 'Rate limit check failed' });
        }
    }
};

app.use('/api', rateLimitMiddleware);

app.get('/api/data', (req, res) => {
    res.json({ message: 'Here is your data!' });
});
```

## Testing Rate Limits

### Stress Test a Key

Quickly hit the rate limit:

```bash
# Hit the free tier limit (100 requests/hour)
for i in {1..105}; do
  echo "Request $i:"
  curl -s -X POST http://localhost:8090/api/public/v1/rate-limit/validate \
    -H "Content-Type: application/json" \
    -d '{"api_key": "test-key-free-123", "requests": 1}' \
    | jq '.allowed, .remaining'
  
  if [ $i -eq 101 ]; then
    echo "Should start getting rate limited..."
  fi
done
```

### Reset for Testing

Reset a key's rate limit (admin only):

```bash
curl -X POST http://localhost:8090/api/v1/rate-limit/reset \
  -H "X-API-Key: test-key-pro-789" \
  -H "Content-Type: application/json" \
  -d '{"api_key": "test-key-free-123"}'
```

## Performance Testing

Run the included performance tests:

```bash
# Install k6
brew install k6  # macOS
# or sudo apt-get install k6  # Linux

# Run basic performance test
cd k6
k6 run rate_limiter_basic.js

# Run all tests
./run_performance_tests.sh
```

## Monitoring

### View Metrics

Get Prometheus metrics:

```bash
curl http://localhost:8090/metrics
```

### Check System Health

```bash
curl http://localhost:8090/health
```

## Common Issues

### API Server Not Starting

```bash
# Check if ports are available
lsof -i :8090

# Check dependencies
docker-compose ps

# View logs
docker-compose logs postgres redis
```

### Rate Limits Not Working

```bash
# Check Redis connection
redis-cli ping

# Check API key exists
curl -H "X-API-Key: test-key-pro-789" \
  "http://localhost:8090/api/v1/api-keys/by-key/test-key-free-123"
```

### Database Issues

```bash
# Reset database
docker-compose down
docker-compose up -d

# Check database connection
psql -h localhost -U postgres -d ratelimiter -c "SELECT 1;"
```

## Next Steps

1. **Read the [Full API Reference](API_REFERENCE.md)** for complete documentation
2. **Set up monitoring** with Prometheus and Grafana
3. **Configure production settings** in `configs/prod.yaml`
4. **Set up CI/CD** with the included GitHub Actions workflow
5. **Deploy to production** using Docker Compose or Kubernetes

## Getting Help

- 📖 [Full API Documentation](API_REFERENCE.md)
- 🐛 [Report Issues](https://github.com/viva/rate-limiter/issues)
- 💬 [Discussions](https://github.com/viva/rate-limiter/discussions)
- 📧 Email: support@viva-rate-limiter.com

Happy rate limiting! 🚀