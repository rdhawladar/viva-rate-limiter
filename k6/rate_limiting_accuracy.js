// Rate Limiting Accuracy Test
// Verifies that rate limits are enforced correctly under various scenarios
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const rateLimitAccuracy = new Rate('rate_limit_accuracy');
const falsePositives = new Counter('false_positives');
const falseNegatives = new Counter('false_negatives');

export const options = {
  scenarios: {
    accuracy_test: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 1,
      maxDuration: '10m',
    },
  },
  thresholds: {
    rate_limit_accuracy: ['rate>0.95'],  // 95% accuracy required
    false_positives: ['count<5'],        // Less than 5 false positives
    false_negatives: ['count<2'],        // Less than 2 false negatives  
    http_req_failed: ['rate<0.01'],      // Very low failure rate
  },
};

const BASE_URL = 'http://localhost:8090';
const ADMIN_API_KEY = 'test-key-pro-789';

export function setup() {
  console.log('Setting up rate limiting accuracy test...');
  
  // Create a test key with known limits for accuracy testing
  const createPayload = {
    name: 'accuracy-test-key',
    tier: 'standard', // Should have 1000 requests per hour
    description: 'Key for testing rate limiting accuracy',
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

  let testKey = null;
  if (createResponse.status === 201) {
    try {
      const data = JSON.parse(createResponse.body);
      testKey = data.data.key;
    } catch (e) {
      console.error('Failed to parse created key response');
    }
  }

  if (!testKey) {
    console.error('Failed to create test key, using fallback');
    testKey = 'test-key-standard-456'; // Fallback to existing key
  }

  console.log(`Using test key: ${testKey}`);
  return { testKey };
}

export default function (data) {
  const testKey = data.testKey;
  
  console.log('Starting rate limiting accuracy test...');
  
  // Test 1: Verify initial state (should be allowed)
  console.log('Test 1: Verifying initial state...');
  let infoResponse = http.get(`${BASE_URL}/api/v1/rate-limit/info`, {
    headers: { 'X-API-Key': testKey },
    timeout: '10s',
  });
  
  let currentInfo = null;
  if (infoResponse.status === 200) {
    try {
      const data = JSON.parse(infoResponse.body);
      currentInfo = data.data;
      console.log(`Initial state - Used: ${currentInfo.used}, Limit: ${currentInfo.limit}, Remaining: ${currentInfo.remaining}`);
    } catch (e) {
      console.error('Failed to parse info response');
      return;
    }
  } else {
    console.error(`Failed to get initial info: ${infoResponse.status}`);
    return;
  }

  // Reset the key to start fresh
  const resetResponse = http.post(`${BASE_URL}/api/v1/rate-limit/reset`, 
    JSON.stringify({ api_key: testKey }),
    {
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': ADMIN_API_KEY,
      },
      timeout: '10s',
    }
  );

  if (resetResponse.status !== 200) {
    console.error('Failed to reset test key');
    return;
  }

  console.log('Reset test key successfully');
  sleep(1); // Wait for reset to propagate

  // Test 2: Sequential requests within limit
  console.log('Test 2: Testing sequential requests within limit...');
  const sequentialTests = 20; // Well within the standard tier limit
  let allowedCount = 0;
  let deniedCount = 0;

  for (let i = 0; i < sequentialTests; i++) {
    const validateResponse = http.post(`${BASE_URL}/api/public/v1/rate-limit/validate`, 
      JSON.stringify({
        api_key: testKey,
        requests: 1,
      }), 
      {
        headers: { 'Content-Type': 'application/json' },
        timeout: '10s',
      }
    );

    if (validateResponse.status === 200) {
      allowedCount++;
    } else if (validateResponse.status === 429) {
      deniedCount++;
      console.error(`Unexpected denial at request ${i + 1}/${sequentialTests}`);
      falseNegatives.add(1);
    }

    // Small delay to simulate realistic usage
    sleep(0.1);
  }

  console.log(`Sequential test: ${allowedCount} allowed, ${deniedCount} denied out of ${sequentialTests}`);
  
  const sequentialAccuracy = allowedCount === sequentialTests;
  rateLimitAccuracy.add(sequentialAccuracy ? 1 : 0);

  // Test 3: Verify usage tracking accuracy
  console.log('Test 3: Verifying usage tracking accuracy...');
  infoResponse = http.get(`${BASE_URL}/api/v1/rate-limit/info`, {
    headers: { 'X-API-Key': testKey },
    timeout: '10s',
  });

  if (infoResponse.status === 200) {
    try {
      const data = JSON.parse(infoResponse.body);
      const updatedInfo = data.data;
      console.log(`After sequential test - Used: ${updatedInfo.used}, Expected: ${sequentialTests}, Remaining: ${updatedInfo.remaining}`);
      
      const usageAccuracy = Math.abs(updatedInfo.used - sequentialTests) <= 1; // Allow 1 request tolerance
      rateLimitAccuracy.add(usageAccuracy ? 1 : 0);
      
      if (!usageAccuracy) {
        console.error(`Usage tracking inaccuracy: expected ~${sequentialTests}, got ${updatedInfo.used}`);
      }
    } catch (e) {
      console.error('Failed to parse updated info response');
      errorRate.add(1);
    }
  }

  // Test 4: Bulk request validation
  console.log('Test 4: Testing bulk request validation...');
  const bulkSize = 10;
  const bulkResponse = http.post(`${BASE_URL}/api/public/v1/rate-limit/validate`, 
    JSON.stringify({
      api_key: testKey,
      requests: bulkSize,
    }), 
    {
      headers: { 'Content-Type': 'application/json' },
      timeout: '10s',
    }
  );

  const bulkSuccess = check(bulkResponse, {
    'bulk request status is 200': (r) => r.status === 200,
    'bulk request has proper response': (r) => {
      try {
        const data = JSON.parse(r.body);
        return data.allowed !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  if (bulkSuccess) {
    rateLimitAccuracy.add(1);
  } else {
    errorRate.add(1);
  }

  // Test 5: Rapid fire test (stress the sliding window)
  console.log('Test 5: Rapid fire test...');
  const rapidTests = 50;
  let rapidAllowed = 0;
  let rapidDenied = 0;

  for (let i = 0; i < rapidTests; i++) {
    const rapidResponse = http.post(`${BASE_URL}/api/public/v1/rate-limit/validate`, 
      JSON.stringify({
        api_key: testKey,
        requests: 1,
      }), 
      {
        headers: { 'Content-Type': 'application/json' },
        timeout: '5s',
      }
    );

    if (rapidResponse.status === 200) {
      rapidAllowed++;
    } else if (rapidResponse.status === 429) {
      rapidDenied++;
    }

    // No delay - true rapid fire
  }

  console.log(`Rapid fire test: ${rapidAllowed} allowed, ${rapidDenied} denied out of ${rapidTests}`);

  // Get final usage info
  console.log('Test 6: Final usage verification...');
  const finalInfoResponse = http.get(`${BASE_URL}/api/v1/rate-limit/info`, {
    headers: { 'X-API-Key': testKey },
    timeout: '10s',
  });

  if (finalInfoResponse.status === 200) {
    try {
      const data = JSON.parse(finalInfoResponse.body);
      const finalInfo = data.data;
      console.log(`Final state - Used: ${finalInfo.used}, Limit: ${finalInfo.limit}, Remaining: ${finalInfo.remaining}`);
      
      const totalExpected = sequentialTests + bulkSize + rapidAllowed;
      const finalAccuracy = Math.abs(finalInfo.used - totalExpected) <= 2; // Allow 2 request tolerance
      rateLimitAccuracy.add(finalAccuracy ? 1 : 0);
      
      if (!finalAccuracy) {
        console.error(`Final usage tracking inaccuracy: expected ~${totalExpected}, got ${finalInfo.used}`);
      }
    } catch (e) {
      console.error('Failed to parse final info response');
      errorRate.add(1);
    }
  }

  // Test 7: Rate limit enforcement at boundary
  console.log('Test 7: Testing rate limit enforcement at boundary...');
  
  // Try to reset and then exceed the limit quickly
  http.post(`${BASE_URL}/api/v1/rate-limit/reset`, 
    JSON.stringify({ api_key: testKey }),
    {
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': ADMIN_API_KEY,
      },
      timeout: '10s',
    }
  );

  sleep(1); // Wait for reset

  // For standard tier, try to use most of the limit
  const boundaryTest = 900; // Close to 1000 limit for standard tier
  const boundaryResponse = http.post(`${BASE_URL}/api/public/v1/rate-limit/validate`, 
    JSON.stringify({
      api_key: testKey,
      requests: boundaryTest,
    }), 
    {
      headers: { 'Content-Type': 'application/json' },
      timeout: '30s',
    }
  );

  if (boundaryResponse.status === 200) {
    // Now try to exceed the limit
    const exceedResponse = http.post(`${BASE_URL}/api/public/v1/rate-limit/validate`, 
      JSON.stringify({
        api_key: testKey,
        requests: 200, // This should be denied
      }), 
      {
        headers: { 'Content-Type': 'application/json' },
        timeout: '10s',
      }
    );

    const boundaryAccuracy = exceedResponse.status === 429;
    rateLimitAccuracy.add(boundaryAccuracy ? 1 : 0);
    
    if (!boundaryAccuracy) {
      console.error(`Boundary test failed: expected 429, got ${exceedResponse.status}`);
      falsePositives.add(1);
    } else {
      console.log('Boundary test passed: rate limit correctly enforced');
    }
  }

  console.log('Rate limiting accuracy test completed!');
}

export function teardown(data) {
  console.log('Cleaning up accuracy test...');
  
  // Delete the test key if it was created
  if (data.testKey && data.testKey !== 'test-key-standard-456') {
    const deleteResponse = http.del(`${BASE_URL}/api/v1/api-keys/by-key/${data.testKey}`, {
      headers: {
        'X-API-Key': ADMIN_API_KEY,
      },
      timeout: '10s',
    });
    
    if (deleteResponse.status === 200) {
      console.log('Test key cleaned up successfully');
    }
  }

  console.log('Accuracy test summary:');
  console.log('- Rate limit accuracy should be >95%');
  console.log('- False positives should be <5');  
  console.log('- False negatives should be <2');
  console.log('Check the detailed metrics for results.');
}