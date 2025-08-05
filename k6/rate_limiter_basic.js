// Basic Rate Limiter Performance Test
// Tests the core rate limiting functionality with increasing load
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '30s', target: 10 },   // Warm up: 10 users
    { duration: '1m', target: 50 },    // Ramp up: 50 users
    { duration: '2m', target: 100 },   // Normal load: 100 users
    { duration: '1m', target: 200 },   // High load: 200 users
    { duration: '30s', target: 500 },  // Spike: 500 users
    { duration: '30s', target: 0 },    // Cool down
  ],
  thresholds: {
    http_req_duration: ['p(95)<50', 'p(99)<100'], // 95% under 50ms, 99% under 100ms
    http_req_failed: ['rate<0.1'],                // Error rate under 10%
    errors: ['rate<0.05'],                        // Custom error rate under 5%
  },
};

const BASE_URL = 'http://localhost:8090';

// Test configuration
const API_KEYS = [
  'test-key-free-123',      // Free tier
  'test-key-standard-456',  // Standard tier  
  'test-key-pro-789',       // Pro tier
];

export default function () {
  // Select random API key for this iteration
  const apiKey = API_KEYS[Math.floor(Math.random() * API_KEYS.length)];
  
  // Test 1: Rate limit validation (most critical path)
  const validateResponse = http.post(`${BASE_URL}/api/public/v1/rate-limit/validate`, 
    JSON.stringify({
      api_key: apiKey,
      requests: 1
    }), 
    {
      headers: {
        'Content-Type': 'application/json',
      },
      timeout: '10s',
    }
  );

  const validateSuccess = check(validateResponse, {
    'rate limit validate status is 200 or 429': (r) => [200, 429].includes(r.status),
    'rate limit validate response time < 50ms': (r) => r.timings.duration < 50,
    'rate limit validate response time < 10ms': (r) => r.timings.duration < 10,
    'validate has proper headers': (r) => r.headers['Content-Type'] && r.headers['Content-Type'].includes('application/json'),
  });

  if (!validateSuccess) {
    errorRate.add(1);
  }

  // Test 2: API key info lookup (secondary path)
  if (Math.random() < 0.3) { // 30% of requests also test info endpoint
    const infoResponse = http.get(`${BASE_URL}/api/v1/rate-limit/info`, {
      headers: {
        'X-API-Key': apiKey,
      },
      timeout: '5s',
    });

    const infoSuccess = check(infoResponse, {
      'info endpoint status is 200': (r) => r.status === 200,
      'info response time < 100ms': (r) => r.timings.duration < 100,
      'info has rate limit data': (r) => {
        try {
          const data = JSON.parse(r.body);
          return data.limit && data.used !== undefined && data.remaining !== undefined;
        } catch (e) {
          return false;
        }
      },
    });

    if (!infoSuccess) {
      errorRate.add(1);
    }
  }

  // Realistic delay between requests (10-100ms)
  sleep(Math.random() * 0.09 + 0.01);
}

export function teardown() {
  console.log('Performance test completed!');
  console.log('Check the summary for detailed metrics.');
}