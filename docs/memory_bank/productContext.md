# Product Context: Rate-Limited API Key Manager

## User Personas

### 1. API Developer (External)
**Profile**: Third-party developer integrating with our API
**Needs**:
- Quick API key generation
- Clear rate limit visibility
- Usage tracking dashboard
- Comprehensive documentation

**User Journey**:
1. Signs up for API access
2. Creates API key with appropriate tier
3. Integrates key into application
4. Monitors usage and limits
5. Upgrades tier when needed

### 2. Platform Administrator
**Profile**: Internal team managing the API platform
**Needs**:
- Bulk key management
- Usage analytics across all keys
- Billing report generation
- System health monitoring

**User Journey**:
1. Reviews daily usage reports
2. Identifies suspicious activity
3. Suspends/revokes problematic keys
4. Generates billing reports
5. Adjusts rate limits as needed

### 3. DevOps Engineer
**Profile**: Team member responsible for system reliability
**Needs**:
- Real-time system metrics
- Alert configuration
- Performance monitoring
- Capacity planning data

**User Journey**:
1. Monitors system dashboards
2. Responds to alerts
3. Scales infrastructure
4. Optimizes performance
5. Plans capacity upgrades

## Feature Specifications

### API Key Management

#### Create API Key
**Endpoint**: `POST /api/v1/keys/create`
**Request**:
```json
{
  "name": "Production API Key",
  "tier": "pro",
  "metadata": {
    "environment": "production",
    "team": "mobile"
  }
}
```
**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "key": "sk_live_4eC39HqLyjWDarjtT1zdp7dc",
  "name": "Production API Key",
  "tier": "pro",
  "rate_limit": 10000,
  "rate_window": 3600,
  "created_at": "2024-01-15T10:30:00Z"
}
```

#### List API Keys
**Endpoint**: `GET /api/v1/keys/list`
**Query Parameters**:
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20)
- `status`: Filter by status (active, suspended, revoked)
- `tier`: Filter by tier

#### Get Key Details
**Endpoint**: `GET /api/v1/keys/:keyId`
**Response**: Complete key information including current usage

#### Update Rate Limits
**Endpoint**: `PUT /api/v1/keys/:keyId`
**Request**:
```json
{
  "rate_limit": 50000,
  "rate_window": 3600,
  "tier": "enterprise"
}
```

#### Rotate Key
**Endpoint**: `POST /api/v1/keys/:keyId/rotate`
**Response**: New API key while maintaining same ID and settings

#### Revoke Key
**Endpoint**: `DELETE /api/v1/keys/:keyId`
**Effect**: Immediately invalidates the key

### Usage Tracking

#### Current Usage
**Endpoint**: `GET /api/v1/usage/:keyId/current`
**Response**:
```json
{
  "key_id": "550e8400-e29b-41d4-a716-446655440000",
  "current_window": {
    "start": "2024-01-15T10:00:00Z",
    "end": "2024-01-15T11:00:00Z",
    "requests": 4532,
    "limit": 10000,
    "remaining": 5468,
    "reset_at": "2024-01-15T11:00:00Z"
  }
}
```

#### Historical Usage
**Endpoint**: `GET /api/v1/usage/:keyId/history`
**Query Parameters**:
- `start_date`: Beginning of period
- `end_date`: End of period
- `granularity`: hour, day, week, month

**Response**:
```json
{
  "key_id": "550e8400-e29b-41d4-a716-446655440000",
  "period": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-01-15T00:00:00Z"
  },
  "usage": [
    {
      "timestamp": "2024-01-01T00:00:00Z",
      "requests": 8542,
      "successful": 8400,
      "rate_limited": 142,
      "errors": 0
    }
  ],
  "summary": {
    "total_requests": 128430,
    "total_successful": 127000,
    "total_rate_limited": 1430,
    "average_daily": 8562
  }
}
```

### Rate Limit Validation

#### Validate Key
**Endpoint**: `POST /api/v1/validate`
**Request**:
```json
{
  "api_key": "sk_live_4eC39HqLyjWDarjtT1zdp7dc",
  "endpoint": "/api/v1/users"
}
```
**Response**:
```json
{
  "valid": true,
  "rate_limit": {
    "allowed": true,
    "limit": 10000,
    "remaining": 5468,
    "reset_at": "2024-01-15T11:00:00Z"
  },
  "key_info": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "tier": "pro",
    "status": "active"
  }
}
```

## User Workflows

### 1. New Developer Onboarding
```mermaid
graph LR
    A[Sign Up] --> B[Choose Tier]
    B --> C[Create API Key]
    C --> D[Download Credentials]
    D --> E[Test Integration]
    E --> F[Monitor Usage]
```

### 2. Rate Limit Hit Handling
```mermaid
graph TD
    A[API Request] --> B{Check Rate Limit}
    B -->|Within Limit| C[Process Request]
    B -->|Exceeded| D[Return 429 Error]
    D --> E[Check Retry-After Header]
    E --> F[Wait and Retry]
    C --> G[Update Usage Counter]
```

### 3. Tier Upgrade Process
```mermaid
graph LR
    A[Monitor Usage] --> B{Approaching Limit?}
    B -->|Yes| C[Request Upgrade]
    C --> D[Admin Approval]
    D --> E[Update Limits]
    E --> F[Notify User]
    B -->|No| A
```

## API Response Standards

### Success Response
```json
{
  "success": true,
  "data": { },
  "metadata": {
    "request_id": "req_1234567890",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "API rate limit exceeded for key",
    "details": {
      "limit": 10000,
      "window": "1h",
      "retry_after": 1800
    }
  },
  "metadata": {
    "request_id": "req_1234567890",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

## Rate Limit Headers

All API responses include:
- `X-RateLimit-Limit`: Request limit per window
- `X-RateLimit-Remaining`: Requests remaining
- `X-RateLimit-Reset`: Window reset timestamp
- `X-RateLimit-Window`: Window duration
- `Retry-After`: Seconds until next request (when limited)

## Tier Specifications

### Free Tier
- 1,000 requests/hour
- Basic usage tracking
- Email support
- Single API key

### Pro Tier
- 10,000 requests/hour
- Advanced analytics
- Priority support
- Up to 5 API keys
- Custom endpoints

### Enterprise Tier
- 100,000+ requests/hour
- Custom rate limits
- Dedicated support
- Unlimited API keys
- SLA guarantees
- Custom billing

## Integration Examples

### JavaScript/Node.js
```javascript
const axios = require('axios');

const client = axios.create({
  baseURL: 'https://api.example.com',
  headers: {
    'Authorization': 'Bearer sk_live_4eC39HqLyjWDarjtT1zdp7dc'
  }
});

// Check rate limit before request
async function makeRequest() {
  try {
    const response = await client.get('/api/v1/data');
    console.log(`Remaining: ${response.headers['x-ratelimit-remaining']}`);
    return response.data;
  } catch (error) {
    if (error.response?.status === 429) {
      const retryAfter = error.response.headers['retry-after'];
      console.log(`Rate limited. Retry after ${retryAfter} seconds`);
    }
  }
}
```

### Python
```python
import requests
from datetime import datetime, timedelta

class APIClient:
    def __init__(self, api_key):
        self.api_key = api_key
        self.base_url = "https://api.example.com"
        
    def make_request(self, endpoint):
        headers = {"Authorization": f"Bearer {self.api_key}"}
        response = requests.get(f"{self.base_url}{endpoint}", headers=headers)
        
        if response.status_code == 429:
            retry_after = int(response.headers.get('Retry-After', 60))
            print(f"Rate limited. Waiting {retry_after} seconds...")
            time.sleep(retry_after)
            return self.make_request(endpoint)
            
        return response.json()
```

## Monitoring Dashboard Views

### Key Metrics Display
- Current request rate (requests/sec)
- Active API keys count
- Rate limit violations/hour
- Average response time
- Error rate percentage

### Usage Analytics
- Top API keys by usage
- Usage trends over time
- Endpoint popularity
- Geographic distribution
- Peak usage hours

### Billing Dashboard
- Current billing period usage
- Projected monthly cost
- Overage warnings
- Historical billing data
- Cost breakdown by key