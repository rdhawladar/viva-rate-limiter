# Viva Rate Limiter API Reference

## Table of Contents
- [Overview](#overview)
- [Authentication](#authentication)
- [Base URL](#base-url)
- [Error Responses](#error-responses)
- [Rate Limiting Headers](#rate-limiting-headers)
- [Endpoints](#endpoints)
  - [Health Check](#health-check)
  - [API Keys](#api-keys)
  - [Rate Limiting](#rate-limiting)
  - [Usage Tracking](#usage-tracking)
  - [Metrics](#metrics)

## Overview

The Viva Rate Limiter API provides comprehensive rate limiting and API key management functionality. All responses are in JSON format.

## Authentication

Most endpoints require authentication using an API key. Include your API key in the request header:

```
X-API-Key: your-api-key-here
```

### API Key Tiers

| Tier | Rate Limit | Features |
|------|------------|----------|
| Free | 100 requests/hour | Basic rate limiting |
| Standard | 1,000 requests/hour | Advanced features |
| Pro | 10,000 requests/hour | All features + priority support |

## Base URL

```
http://localhost:8090
```

For production deployments, replace with your actual domain.

## Error Responses

All errors follow a consistent format:

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded for this API key",
    "details": {
      "limit": 100,
      "window": "1h",
      "retry_after": 1234
    }
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_API_KEY` | 401 | API key is invalid or missing |
| `RATE_LIMIT_EXCEEDED` | 429 | Rate limit has been exceeded |
| `INVALID_REQUEST` | 400 | Request body is invalid |
| `NOT_FOUND` | 404 | Resource not found |
| `INTERNAL_ERROR` | 500 | Internal server error |

## Rate Limiting Headers

All API responses include rate limiting information:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1625097600
Retry-After: 3600
```

## Endpoints

### Health Check

#### Check API Health
```http
GET /health
```

Check if the API and its dependencies are healthy.

**Authentication**: Not required

**Response**
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

---

### API Keys

#### Create API Key
```http
POST /api/v1/api-keys
```

Create a new API key.

**Authentication**: Required (admin privileges)

**Request Body**
```json
{
  "name": "My Application",
  "tier": "standard",
  "description": "API key for production app",
  "metadata": {
    "app_id": "12345",
    "environment": "production"
  }
}
```

**Parameters**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Display name for the API key |
| tier | string | Yes | One of: `free`, `standard`, `pro` |
| description | string | No | Description of the key's purpose |
| metadata | object | No | Custom metadata |

**Response** (201 Created)
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
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z",
    "metadata": {
      "app_id": "12345",
      "environment": "production"
    }
  }
}
```

**Note**: The API key is only shown once during creation. Store it securely.

#### List API Keys
```http
GET /api/v1/api-keys
```

List all API keys for your account.

**Authentication**: Required

**Query Parameters**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | integer | 1 | Page number |
| limit | integer | 20 | Items per page (max: 100) |
| status | string | all | Filter by status: `active`, `suspended`, `revoked` |
| tier | string | all | Filter by tier: `free`, `standard`, `pro` |

**Response**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "My Application",
      "tier": "standard",
      "status": "active",
      "rate_limit": 1000,
      "rate_limit_window": "1h",
      "last_used_at": "2024-01-15T09:00:00Z",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45,
    "pages": 3
  }
}
```

#### Get API Key Details
```http
GET /api/v1/api-keys/{id}
```

Get detailed information about a specific API key.

**Authentication**: Required

**Response**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "My Application",
    "key_hash": "sha256:a1b2c3d4...",
    "tier": "standard",
    "status": "active",
    "rate_limit": 1000,
    "rate_limit_window": "1h",
    "last_used_at": "2024-01-15T09:00:00Z",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z",
    "metadata": {
      "app_id": "12345",
      "environment": "production"
    },
    "usage_stats": {
      "total_requests": 125000,
      "requests_this_month": 5000,
      "requests_today": 250
    }
  }
}
```

#### Get API Key by Key
```http
GET /api/v1/api-keys/by-key/{api_key}
```

Get details using the actual API key value (useful for debugging).

**Authentication**: Required (admin privileges)

#### Update API Key
```http
PUT /api/v1/api-keys/{id}
```

Update an existing API key.

**Authentication**: Required

**Request Body**
```json
{
  "name": "Updated Application Name",
  "description": "Updated description",
  "status": "suspended",
  "metadata": {
    "app_id": "12345",
    "environment": "staging"
  }
}
```

**Response**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Updated Application Name",
    "tier": "standard",
    "status": "suspended",
    "updated_at": "2024-01-15T11:00:00Z"
  }
}
```

#### Update API Key by Key
```http
PUT /api/v1/api-keys/by-key/{api_key}
```

Update using the actual API key value.

**Authentication**: Required (admin privileges)

#### Delete API Key
```http
DELETE /api/v1/api-keys/{id}
```

Permanently delete an API key.

**Authentication**: Required

**Response** (200 OK)
```json
{
  "message": "API key deleted successfully"
}
```

#### Delete API Key by Key
```http
DELETE /api/v1/api-keys/by-key/{api_key}
```

Delete using the actual API key value.

**Authentication**: Required (admin privileges)

---

### Rate Limiting

#### Validate Rate Limit (Public)
```http
POST /api/public/v1/rate-limit/validate
```

Check if requests are allowed for an API key. This is the primary endpoint for rate limiting.

**Authentication**: Not required (API key passed in body)

**Request Body**
```json
{
  "api_key": "viva_live_sk_a1b2c3d4e5f6...",
  "requests": 1
}
```

**Parameters**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| api_key | string | Yes | The API key to validate |
| requests | integer | No | Number of requests to validate (default: 1) |

**Response** (200 OK - Allowed)
```json
{
  "allowed": true,
  "limit": 1000,
  "remaining": 999,
  "reset_time": "2024-01-15T11:00:00Z",
  "reset_in_seconds": 1800
}
```

**Response** (429 Too Many Requests - Denied)
```json
{
  "allowed": false,
  "limit": 1000,
  "remaining": 0,
  "reset_time": "2024-01-15T11:00:00Z",
  "reset_in_seconds": 1800,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded"
  }
}
```

#### Get Rate Limit Info
```http
GET /api/v1/rate-limit/info
```

Get current rate limit status for the authenticated API key.

**Authentication**: Required

**Response**
```json
{
  "data": {
    "key": "user-api-key",
    "limit": 1000,
    "remaining": 750,
    "used": 250,
    "window": "1h",
    "window_start": "2024-01-15T10:00:00Z",
    "window_end": "2024-01-15T11:00:00Z",
    "reset_time": "2024-01-15T11:00:00Z",
    "retry_after": 0
  }
}
```

#### Reset Rate Limit
```http
POST /api/v1/rate-limit/reset
```

Reset rate limit counter for an API key (admin only).

**Authentication**: Required (admin privileges)

**Request Body**
```json
{
  "api_key": "viva_live_sk_a1b2c3d4e5f6..."
}
```

**Response**
```json
{
  "message": "Rate limit reset successfully",
  "data": {
    "key": "user-api-key",
    "limit": 1000,
    "remaining": 1000,
    "used": 0
  }
}
```

---

### Usage Tracking

#### Get Current Usage
```http
GET /api/v1/usage/{api_key}/current
```

Get current billing period usage for an API key.

**Authentication**: Required

**Response**
```json
{
  "data": {
    "api_key": "viva_live_sk_a1b2c3d4e5f6...",
    "billing_period": {
      "start": "2024-01-01T00:00:00Z",
      "end": "2024-01-31T23:59:59Z"
    },
    "requests_made": 45000,
    "requests_limit": 100000,
    "requests_remaining": 55000,
    "overage_requests": 0,
    "estimated_cost": 25.00,
    "currency": "USD"
  }
}
```

#### Get Usage History
```http
GET /api/v1/usage/{api_key}/history
```

Get historical usage data.

**Authentication**: Required

**Query Parameters**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| start_date | date | 30 days ago | Start date (YYYY-MM-DD) |
| end_date | date | today | End date (YYYY-MM-DD) |
| granularity | string | daily | One of: `hourly`, `daily`, `monthly` |

**Response**
```json
{
  "data": {
    "api_key": "viva_live_sk_a1b2c3d4e5f6...",
    "period": {
      "start": "2024-01-01T00:00:00Z",
      "end": "2024-01-15T23:59:59Z"
    },
    "granularity": "daily",
    "usage": [
      {
        "date": "2024-01-01",
        "requests": 3250,
        "successful": 3200,
        "rate_limited": 50,
        "errors": 0
      },
      {
        "date": "2024-01-02",
        "requests": 2800,
        "successful": 2750,
        "rate_limited": 45,
        "errors": 5
      }
    ],
    "summary": {
      "total_requests": 45000,
      "total_successful": 44500,
      "total_rate_limited": 450,
      "total_errors": 50,
      "average_daily": 3000
    }
  }
}
```

#### Get Usage Limits
```http
GET /api/v1/usage/{api_key}/limits
```

Get rate limit and usage quota information.

**Authentication**: Required

**Response**
```json
{
  "data": {
    "api_key": "viva_live_sk_a1b2c3d4e5f6...",
    "tier": "standard",
    "rate_limit": {
      "requests_per_window": 1000,
      "window_duration": "1h"
    },
    "monthly_quota": {
      "requests": 100000,
      "overage_allowed": true,
      "overage_rate": 0.001
    },
    "features": {
      "custom_limits": true,
      "analytics": true,
      "webhooks": false,
      "priority_support": false
    }
  }
}
```

---

### Metrics

#### Get Prometheus Metrics
```http
GET /metrics
```

Get metrics in Prometheus format for monitoring.

**Authentication**: Not required (if metrics are enabled)

**Response** (text/plain)
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",status="200"} 12345

# HELP rate_limit_checks_total Total number of rate limit checks
# TYPE rate_limit_checks_total counter
rate_limit_checks_total{result="allowed"} 10000
rate_limit_checks_total{result="denied"} 500

# HELP api_keys_active Number of active API keys
# TYPE api_keys_active gauge
api_keys_active{tier="free"} 100
api_keys_active{tier="standard"} 50
api_keys_active{tier="pro"} 10
```

---

## Code Examples

### cURL

#### Create an API Key
```bash
curl -X POST http://localhost:8090/api/v1/api-keys \
  -H "X-API-Key: admin-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production App",
    "tier": "standard",
    "description": "Main production API key"
  }'
```

#### Check Rate Limit
```bash
curl -X POST http://localhost:8090/api/public/v1/rate-limit/validate \
  -H "Content-Type: application/json" \
  -d '{
    "api_key": "viva_live_sk_a1b2c3d4e5f6...",
    "requests": 1
  }'
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

func checkRateLimit(apiKey string) error {
    payload := map[string]interface{}{
        "api_key": apiKey,
        "requests": 1,
    }
    
    jsonData, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    
    resp, err := http.Post(
        "http://localhost:8090/api/public/v1/rate-limit/validate",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == 429 {
        return fmt.Errorf("rate limit exceeded")
    }
    
    return nil
}
```

### Python

```python
import requests

def check_rate_limit(api_key: str) -> bool:
    response = requests.post(
        "http://localhost:8090/api/public/v1/rate-limit/validate",
        json={
            "api_key": api_key,
            "requests": 1
        }
    )
    
    if response.status_code == 200:
        data = response.json()
        return data["allowed"]
    elif response.status_code == 429:
        return False
    else:
        response.raise_for_status()
```

### Node.js

```javascript
const axios = require('axios');

async function checkRateLimit(apiKey) {
    try {
        const response = await axios.post(
            'http://localhost:8090/api/public/v1/rate-limit/validate',
            {
                api_key: apiKey,
                requests: 1
            }
        );
        return response.data.allowed;
    } catch (error) {
        if (error.response && error.response.status === 429) {
            return false;
        }
        throw error;
    }
}
```

## Webhooks (Coming Soon)

Webhooks for real-time notifications about:
- Rate limit violations
- Usage threshold alerts
- API key status changes

## SDK Support (Coming Soon)

Official SDKs planned for:
- Go
- Python
- Node.js
- Ruby
- Java

## Support

For API support:
- Email: api-support@viva-rate-limiter.com
- Documentation: https://docs.viva-rate-limiter.com
- Status Page: https://status.viva-rate-limiter.com