// Redis Stress Test for Distributed Rate Limiting
// Tests Redis backend under high concurrent load with multiple keys
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const rateLimitHitRate = new Rate('rate_limit_hits');
const redisOperationRate = new Rate('redis_operations');

export const options = {
  stages: [
    { duration: '30s', target: 20 },   // Warm up
    { duration: '1m', target: 100 },   // Build up
    { duration: '2m', target: 300 },   // High concurrency
    { duration: '1m', target: 500 },   // Peak load
    { duration: '30s', target: 100 },  // Scale down
    { duration: '30s', target: 0 },    // Cool down
  ],
  thresholds: {
    http_req_duration: ['p(95)<100', 'p(99)<200'], // Very strict for Redis operations
    http_req_failed: ['rate<0.02'],                // Very low error rate expected
    errors: ['rate<0.01'],                         // Almost no custom errors
    redis_operations: ['rate>1000'],               // At least 1000 Redis ops/sec
  },
};

const BASE_URL = 'http://localhost:8090';

// Generate a large pool of API keys to stress Redis with different keys
const API_KEYS = [];
for (let i = 0; i < 100; i++) {
  API_KEYS.push(`stress-test-key-${i}`);
}

export function setup() {
  console.log('Setting up Redis stress test...');
  console.log(`Generated ${API_KEYS.length} test keys for Redis stress testing`);
  
  // Create some test keys in the system
  const adminKey = 'test-key-pro-789';
  const createdKeys = [];
  
  for (let i = 0; i < 10; i++) {
    const createPayload = {
      name: `stress-key-${i}`,
      tier: 'standard',
      description: `Redis stress test key ${i}`,
    };

    const createResponse = http.post(`${BASE_URL}/api/v1/api-keys`, 
      JSON.stringify(createPayload),
      {
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': adminKey,
        },
        timeout: '30s',
      }
    );

    if (createResponse.status === 201) {
      try {
        const data = JSON.parse(createResponse.body);
        if (data.data && data.data.key) {
          createdKeys.push(data.data.key);
        }
      } catch (e) {
        // Ignore parsing errors in setup
      }
    }
  }
  
  console.log(`Created ${createdKeys.length} additional test keys`);
  return { createdKeys };
}

export default function (data) {
  // Select random API key for high key diversity
  const apiKey = Math.random() < 0.8 
    ? API_KEYS[Math.floor(Math.random() * API_KEYS.length)]  // 80% use stress test keys
    : data.createdKeys[Math.floor(Math.random() * data.createdKeys.length)] || API_KEYS[0]; // 20% use created keys

  // Test 1: High-frequency rate limit checks (70% of operations)
  if (Math.random() < 0.7) {
    const validateResponse = http.post(`${BASE_URL}/api/public/v1/rate-limit/validate`, 
      JSON.stringify({
        api_key: apiKey,
        requests: Math.floor(Math.random() * 5) + 1, // 1-5 requests
      }), 
      {
        headers: {
          'Content-Type': 'application/json',
        },
        timeout: '5s',
      }
    );

    const validateSuccess = check(validateResponse, {
      'redis validate status is 200 or 429': (r) => [200, 429].includes(r.status),
      'redis validate response time < 20ms': (r) => r.timings.duration < 20,  // Very strict
      'redis validate response time < 5ms': (r) => r.timings.duration < 5,    // Target
    });

    redisOperationRate.add(1);
    
    if (validateResponse.status === 429) {
      rateLimitHitRate.add(1);
    }
    
    if (!validateSuccess) {
      errorRate.add(1);
    }
  }

  // Test 2: Parallel rate limit info requests (20% of operations)
  if (Math.random() < 0.2) {
    const infoResponse = http.get(`${BASE_URL}/api/v1/rate-limit/info`, {
      headers: {
        'X-API-Key': apiKey,
      },
      timeout: '3s',
    });

    const infoSuccess = check(infoResponse, {
      'redis info status is 200 or 404': (r) => [200, 404].includes(r.status),
      'redis info response time < 30ms': (r) => r.timings.duration < 30,
      'redis info response time < 10ms': (r) => r.timings.duration < 10,
    });

    redisOperationRate.add(1);
    
    if (!infoSuccess) {
      errorRate.add(1);
    }
  }

  // Test 3: Bulk validation requests (10% of operations)
  if (Math.random() < 0.1) {
    // Test multiple keys in quick succession to stress Redis connection pool
    const bulkKeys = [apiKey];
    for (let i = 0; i < 3; i++) {
      bulkKeys.push(API_KEYS[Math.floor(Math.random() * API_KEYS.length)]);
    }

    for (const key of bulkKeys) {
      const bulkResponse = http.post(`${BASE_URL}/api/public/v1/rate-limit/validate`, 
        JSON.stringify({
          api_key: key,
          requests: 1,
        }), 
        {
          headers: {
            'Content-Type': 'application/json',
          },
          timeout: '2s',
        }
      );

      check(bulkResponse, {
        'bulk redis operation successful': (r) => [200, 429].includes(r.status),
        'bulk redis operation fast': (r) => r.timings.duration < 15,
      });

      redisOperationRate.add(1);
    }
  }

  // Very minimal delay to maximize Redis stress
  sleep(Math.random() * 0.01 + 0.001); // 1-11ms delay
}

export function teardown(data) {
  console.log('Redis stress test completed!');
  console.log('Total Redis operations per second (target: >1000)');
  console.log('Rate limit hit rate (indicates Redis is tracking limits correctly)');
  console.log('Error rate should be minimal (<1%)');
  
  // Clean up created keys
  const adminKey = 'test-key-pro-789';
  
  if (data.createdKeys) {
    console.log(`Cleaning up ${data.createdKeys.length} test keys...`);
    
    for (const key of data.createdKeys) {
      http.del(`${BASE_URL}/api/v1/api-keys/by-key/${key}`, {
        headers: {
          'X-API-Key': adminKey,
        },
        timeout: '10s',
      });
    }
  }
}