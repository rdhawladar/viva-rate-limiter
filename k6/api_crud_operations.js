// API CRUD Operations Performance Test
// Tests all API endpoints with realistic CRUD operations
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const createKeyRate = new Rate('key_creations');
const deleteKeyRate = new Rate('key_deletions');

export const options = {
  stages: [
    { duration: '30s', target: 5 },   // Warm up
    { duration: '2m', target: 20 },   // Normal load
    { duration: '1m', target: 40 },   // High load
    { duration: '30s', target: 0 },   // Cool down
  ],
  thresholds: {
    http_req_duration: ['p(95)<200', 'p(99)<500'], // CRUD operations can be slower
    http_req_failed: ['rate<0.05'],                // Error rate under 5%
    errors: ['rate<0.03'],                         // Custom error rate under 3%
  },
};

const BASE_URL = 'http://localhost:8090';

// Admin API key for management operations
const ADMIN_API_KEY = 'test-key-pro-789';

export default function () {
  const testId = Math.floor(Math.random() * 100000);
  
  // Test 1: Create new API key (25% of operations)
  if (Math.random() < 0.25) {
    const createPayload = {
      name: `test-key-${testId}`,
      tier: ['free', 'standard', 'pro'][Math.floor(Math.random() * 3)],
      description: `Performance test key ${testId}`,
    };

    const createResponse = http.post(`${BASE_URL}/api/v1/api-keys`, 
      JSON.stringify(createPayload),
      {
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': ADMIN_API_KEY,
        },
        timeout: '30s',
      }
    );

    const createSuccess = check(createResponse, {
      'create key status is 201': (r) => r.status === 201,
      'create key response time < 500ms': (r) => r.timings.duration < 500,
      'create key has id': (r) => {
        try {
          const data = JSON.parse(r.body);
          return data.data && data.data.id;
        } catch (e) {
          return false;
        }
      },
    });

    createKeyRate.add(createSuccess ? 1 : 0);
    if (!createSuccess) errorRate.add(1);
  }

  // Test 2: List API keys (40% of operations)
  if (Math.random() < 0.4) {
    const listResponse = http.get(`${BASE_URL}/api/v1/api-keys?page=1&limit=10`, {
      headers: {
        'X-API-Key': ADMIN_API_KEY,
      },
      timeout: '15s',
    });

    const listSuccess = check(listResponse, {
      'list keys status is 200': (r) => r.status === 200,
      'list keys response time < 200ms': (r) => r.timings.duration < 200,
      'list keys has data array': (r) => {
        try {
          const data = JSON.parse(r.body);
          return Array.isArray(data.data);
        } catch (e) {
          return false;
        }
      },
    });

    if (!listSuccess) errorRate.add(1);
  }

  // Test 3: Get specific API key (20% of operations)
  if (Math.random() < 0.2) {
    // Use one of the test keys
    const testKeys = ['test-key-free-123', 'test-key-standard-456', 'test-key-pro-789'];
    const keyToGet = testKeys[Math.floor(Math.random() * testKeys.length)];
    
    const getResponse = http.get(`${BASE_URL}/api/v1/api-keys/by-key/${keyToGet}`, {
      headers: {
        'X-API-Key': ADMIN_API_KEY,
      },
      timeout: '10s',
    });

    const getSuccess = check(getResponse, {
      'get key status is 200': (r) => r.status === 200,
      'get key response time < 100ms': (r) => r.timings.duration < 100,
      'get key has data': (r) => {
        try {
          const data = JSON.parse(r.body);
          return data.data && data.data.key_hash;
        } catch (e) {
          return false;
        }
      },
    });

    if (!getSuccess) errorRate.add(1);
  }

  // Test 4: Update API key (10% of operations)
  if (Math.random() < 0.1) {
    const updatePayload = {
      name: `updated-key-${testId}`,
      description: `Updated description ${testId}`,
      status: 'active',
    };

    // Try to update a test key (this might fail if key doesn't exist, which is OK)
    const updateResponse = http.put(`${BASE_URL}/api/v1/api-keys/by-key/test-key-free-123`, 
      JSON.stringify(updatePayload),
      {
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': ADMIN_API_KEY,
        },
        timeout: '20s',
      }
    );

    check(updateResponse, {
      'update key status is 200 or 404': (r) => [200, 404].includes(r.status),
      'update key response time < 300ms': (r) => r.timings.duration < 300,
    });
  }

  // Test 5: Usage tracking operations (30% of operations)
  if (Math.random() < 0.3) {
    const usageResponse = http.get(`${BASE_URL}/api/v1/usage/test-key-free-123/current`, {
      headers: {
        'X-API-Key': ADMIN_API_KEY,
      },
      timeout: '10s',
    });

    const usageSuccess = check(usageResponse, {
      'usage tracking status is 200': (r) => r.status === 200,
      'usage tracking response time < 150ms': (r) => r.timings.duration < 150,
      'usage tracking has data': (r) => {
        try {
          const data = JSON.parse(r.body);
          return data.data && typeof data.data.requests_made === 'number';
        } catch (e) {
          return false;
        }
      },
    });

    if (!usageSuccess) errorRate.add(1);
  }

  // Test 6: Health check (5% of operations)
  if (Math.random() < 0.05) {
    const healthResponse = http.get(`${BASE_URL}/health`, {
      timeout: '5s',
    });

    check(healthResponse, {
      'health check status is 200': (r) => r.status === 200,
      'health check response time < 50ms': (r) => r.timings.duration < 50,
    });
  }

  // Realistic delay between operations (100-500ms)
  sleep(Math.random() * 0.4 + 0.1);
}

export function teardown() {
  console.log('CRUD operations performance test completed!');
  console.log('Key creation rate:', createKeyRate);
  console.log('Key deletion rate:', deleteKeyRate);
  console.log('Error rate:', errorRate);
}