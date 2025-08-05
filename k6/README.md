# Performance Testing with k6

This directory contains comprehensive performance tests for the Viva Rate Limiter using [k6](https://k6.io/), a modern load testing tool.

## Quick Start

1. **Install k6**:
   ```bash
   # macOS
   brew install k6
   
   # Ubuntu/Debian
   sudo apt-get install k6
   
   # Windows
   choco install k6
   ```

2. **Start the API server**:
   ```bash
   # From project root
   make run-api
   # or
   go run cmd/api/main.go
   ```

3. **Run tests**:
   ```bash
   cd k6
   ./run_performance_tests.sh
   ```

## Test Suite Overview

### 1. Basic Rate Limiter Performance Test (`rate_limiter_basic.js`)
- **Purpose**: Tests core rate limiting functionality under increasing load
- **Load Profile**: 10 → 50 → 100 → 200 → 500 users over 5 minutes
- **Key Metrics**: 
  - p95 < 50ms, p99 < 100ms response times
  - <10% error rate
  - Rate limit validation accuracy
- **Best For**: General performance baseline and response time validation

### 2. API CRUD Operations Test (`api_crud_operations.js`)
- **Purpose**: Tests all API endpoints with realistic CRUD operations
- **Load Profile**: 5 → 20 → 40 users over 3.5 minutes
- **Operations**: Create keys (25%), List keys (40%), Get keys (20%), Update keys (10%), Usage tracking (30%)
- **Key Metrics**:
  - p95 < 200ms, p99 < 500ms (CRUD operations can be slower)
  - <5% error rate
- **Best For**: API endpoint performance and database operation testing

### 3. Redis Stress Test (`redis_stress_test.js`)
- **Purpose**: Tests Redis backend under high concurrent load with multiple keys
- **Load Profile**: 20 → 100 → 300 → 500 users over 5 minutes
- **Key Features**:
  - 100 different API keys for key diversity
  - Very strict timing requirements (p95 < 100ms, p99 < 200ms)
  - High Redis operation rate (>1000 ops/sec)
- **Best For**: Redis performance, connection pooling, and distributed rate limiting

### 4. Rate Limiting Accuracy Test (`rate_limiting_accuracy.js`)
- **Purpose**: Verifies that rate limits are enforced correctly under various scenarios
- **Test Type**: Single VU functional test (not load test)
- **Scenarios**:
  - Sequential requests within limit
  - Usage tracking accuracy
  - Bulk request validation
  - Rapid fire testing
  - Boundary condition testing
- **Key Metrics**:
  - >95% rate limit accuracy
  - <5 false positives, <2 false negatives
- **Best For**: Correctness validation and edge case testing

## Performance Targets

| Metric | Target | Test |
|--------|--------|------|
| Rate Limit Check | < 10ms (p99) | Basic, Redis Stress |
| API Response Time | < 50ms (p95) | Basic, CRUD |
| CRUD Operations | < 200ms (p95) | CRUD |
| Redis Operations | > 1000/sec | Redis Stress |
| Error Rate | < 5% | All Tests |
| Rate Limit Accuracy | > 95% | Accuracy |

## Test Runner Features

The `run_performance_tests.sh` script provides:

- **Interactive Menu**: Choose specific tests or run all
- **Health Check**: Verifies API is running before testing
- **Result Management**: Saves JSON results and summaries with timestamps
- **Sequential Mode**: Runs tests one after another with cool-down periods
- **Parallel Mode**: Runs multiple tests simultaneously (advanced)
- **Color Output**: Easy-to-read results with status colors

## Usage Examples

### Run Individual Tests

```bash
# Basic performance test only
./run_performance_tests.sh
# Select option 1

# Accuracy test only  
./run_performance_tests.sh
# Select option 4
```

### Run All Tests

```bash
# Sequential (recommended)
./run_performance_tests.sh
# Select option 5

# Parallel (advanced - high system stress)
./run_performance_tests.sh
# Select option 6
```

### Run Tests Directly with k6

```bash
# Run basic test
k6 run rate_limiter_basic.js

# Run with custom options
k6 run --vus 100 --duration 2m rate_limiter_basic.js

# Save results to file
k6 run --out json=results.json rate_limiter_basic.js
```

## Interpreting Results

### Key k6 Metrics

- **http_req_duration**: Request response times (p95, p99 percentiles)
- **http_req_failed**: Percentage of failed requests
- **http_reqs**: Total requests and requests per second
- **vus**: Virtual users (concurrent users)

### Custom Metrics

- **errors**: Custom error rate tracking
- **rate_limit_hits**: How often rate limits are triggered
- **redis_operations**: Redis operations per second
- **rate_limit_accuracy**: Percentage of correct rate limit decisions

### Good vs Bad Results

**✅ Good Results:**
```
http_req_duration..............: avg=15ms  p95=45ms  p99=85ms
http_req_failed................: 2.3%
rate_limit_accuracy............: 98.5%
redis_operations...............: 1500/sec
```

**❌ Bad Results:**
```
http_req_duration..............: avg=150ms p95=500ms p99=2s
http_req_failed................: 15%
rate_limit_accuracy............: 85%
redis_operations...............: 200/sec
```

## Troubleshooting

### Common Issues

1. **API Not Running**:
   ```
   ❌ API is not running at http://localhost:8090
   ```
   - Start the API: `make run-api` or `go run cmd/api/main.go`

2. **k6 Not Installed**:
   ```
   ❌ k6 is not installed
   ```
   - Install k6: `brew install k6` (macOS) or visit [k6.io](https://k6.io/)

3. **High Error Rates**:
   - Check API logs for errors
   - Verify database and Redis are running
   - Reduce test load (fewer VUs)

4. **Slow Response Times**:
   - Check system resources (CPU, memory)
   - Verify Redis performance
   - Check database connection pool settings

### Performance Tuning

If tests show poor performance:

1. **Database**: Optimize indexes, connection pooling
2. **Redis**: Check memory usage, connection pooling
3. **API**: Review middleware overhead, logging levels
4. **System**: CPU, memory, network bandwidth

## Advanced Usage

### Custom Test Scenarios

Create custom k6 tests by modifying existing files:

```javascript
export const options = {
  scenarios: {
    custom_load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 100 },
        { duration: '5m', target: 100 },
        { duration: '2m', target: 0 },
      ],
    },
  },
};
```

### Environment Variables

Configure tests with environment variables:

```bash
# Custom API URL
export BASE_URL=http://production-api.com
k6 run rate_limiter_basic.js

# Custom test duration
export TEST_DURATION=10m
k6 run rate_limiter_basic.js
```

### Results Analysis

Use k6's output options for detailed analysis:

```bash
# JSON output for analysis
k6 run --out json=results.json rate_limiter_basic.js

# InfluxDB output (with Grafana)
k6 run --out influxdb=http://localhost:8086/mydb rate_limiter_basic.js

# Multiple outputs
k6 run --out json=results.json --out influxdb=http://localhost:8086/mydb rate_limiter_basic.js
```

## Best Practices

1. **Start Small**: Begin with basic tests before stress testing
2. **Monitor Resources**: Watch CPU, memory, and database performance
3. **Cool Down**: Allow time between tests for system recovery
4. **Baseline**: Record initial performance for comparison
5. **Regular Testing**: Run tests after code changes
6. **Real Data**: Use realistic API keys and request patterns

## Integration with CI/CD

Example GitHub Actions workflow:

```yaml
name: Performance Tests
on: [push, pull_request]

jobs:
  performance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Start Services
        run: docker-compose up -d
      - name: Install k6
        run: |
          sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6
      - name: Run Performance Tests
        run: |
          cd k6
          k6 run rate_limiter_basic.js
```

For more information, visit the [k6 documentation](https://k6.io/docs/).