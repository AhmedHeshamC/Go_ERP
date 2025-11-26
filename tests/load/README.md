# Load Testing Suite

Comprehensive load testing suite for the ERPGo system using k6. This suite validates system performance under various load conditions and ensures the system meets production readiness requirements.

## Requirements Validated

This load testing suite validates the following requirements from the production readiness specification:

- **Requirement 17.1**: System handles 1000 requests per second with p99 latency < 500ms
- **Requirement 17.2**: Error rate < 0.1% under load
- **Requirement 17.3**: System scales horizontally by adding instances
- **Requirement 17.4**: Performance regression detection
- **Requirement 17.5**: Capacity planning metrics (CPU, memory, database connections per request)

## Test Scenarios

### 1. Baseline Load Test
**File**: `k6/baseline-load-test.js`  
**Load**: 100 RPS for 5 minutes  
**Purpose**: Establishes baseline performance metrics under normal load conditions

**Success Criteria**:
- p95 response time < 500ms
- p99 response time < 1000ms
- Error rate < 0.1%

**Workload Distribution**:
- 60% Read operations (product/order listings)
- 30% Search operations
- 10% Write operations

### 2. Peak Load Test
**File**: `k6/peak-load-test.js`  
**Load**: 1000 RPS for 5 minutes  
**Purpose**: Validates system performance under peak production load

**Success Criteria**:
- p95 response time < 500ms
- p99 response time < 1000ms
- Error rate < 0.1%
- Sustained throughput > 900 RPS

**Workload Distribution**:
- 40% Product browsing
- 30% Product search
- 15% Order viewing
- 10% Authentication
- 5% Order creation

### 3. Stress Test
**File**: `k6/stress-test.js`  
**Load**: Gradually increase from 500 to 5000 RPS  
**Duration**: 15 minutes  
**Purpose**: Identifies system breaking point and behavior under extreme load

**Success Criteria**:
- p95 response time < 2000ms
- p99 response time < 5000ms
- Error rate < 5%
- Graceful degradation (no crashes)

**Load Stages**:
1. 2 min: Ramp to 500 VUs
2. 3 min: Ramp to 1000 VUs
3. 3 min: Ramp to 2000 VUs
4. 3 min: Ramp to 3000 VUs
5. 2 min: Spike to 5000 VUs
6. 2 min: Ramp down to 0

### 4. Spike Test
**File**: `k6/spike-test.js`  
**Load**: Sudden jump from 100 to 2000 RPS  
**Duration**: 7.5 minutes  
**Purpose**: Tests system resilience to sudden traffic spikes

**Success Criteria**:
- p95 response time < 1000ms
- p99 response time < 2000ms
- Error rate < 2%
- System recovers to normal performance within 2 minutes

**Load Stages**:
1. 1 min: Normal load (100 VUs)
2. 30 sec: Sudden spike to 2000 VUs
3. 3 min: Sustain spike
4. 1 min: Return to normal
5. 1 min: Observe recovery

## Prerequisites

### 1. Install k6

**macOS**:
```bash
brew install k6
```

**Linux (Debian/Ubuntu)**:
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Windows (Chocolatey)**:
```powershell
choco install k6
```

**Docker**:
```bash
docker pull grafana/k6:latest
```

### 2. Start the Application

Ensure the ERPGo application is running and accessible:

```bash
# Start with Docker Compose
docker-compose up -d

# Or start locally
go run cmd/api/main.go
```

Verify the application is running:
```bash
curl http://localhost:8080/health/live
```

## Running Load Tests

### Run All Tests

```bash
./tests/load/run-load-tests.sh
```

### Run Specific Tests

```bash
# Run only baseline test
./tests/load/run-load-tests.sh baseline

# Run baseline and peak tests
./tests/load/run-load-tests.sh baseline peak

# Run all tests
./tests/load/run-load-tests.sh baseline peak stress spike
```

### Run Against Different Environment

```bash
# Test against staging
BASE_URL=http://staging.example.com ./tests/load/run-load-tests.sh

# Test against production (use with caution!)
BASE_URL=https://api.example.com ./tests/load/run-load-tests.sh baseline
```

### Run Individual k6 Tests

```bash
# Run baseline test directly with k6
k6 run -e BASE_URL=http://localhost:8080 tests/load/k6/baseline-load-test.js

# Run with custom VUs and duration
k6 run --vus 100 --duration 5m tests/load/k6/baseline-load-test.js

# Run with output to InfluxDB
k6 run --out influxdb=http://localhost:8086/k6 tests/load/k6/peak-load-test.js
```

## Test Results

### Results Location

All test results are saved to `tests/load/results/`:
- `baseline-load-test-results.json` - Baseline test results
- `peak-load-test-results.json` - Peak load test results
- `stress-test-results.json` - Stress test results
- `spike-test-results.json` - Spike test results
- `load-test-summary.txt` - Overall summary report

### Interpreting Results

#### Response Time Metrics
- **Avg**: Average response time across all requests
- **P50**: 50th percentile (median) - half of requests are faster
- **P95**: 95th percentile - 95% of requests are faster
- **P99**: 99th percentile - 99% of requests are faster
- **Max**: Slowest request

#### Success Criteria
✓ **PASS**: All thresholds met  
✗ **FAIL**: One or more thresholds failed  
⚠ **WARNING**: Close to threshold limits

#### Example Output
```
Peak Load Test Results (1000 RPS)
==========================================

Status: ✓ PASSED

Duration: 300.00s

HTTP Requests:
  Total: 300000
  Rate: 1000.00 req/s

Response Time:
  Avg: 120.50ms
  Min: 10.20ms
  Max: 980.30ms
  P50: 80.10ms
  P95: 250.40ms ✓
  P99: 480.20ms ✓

Error Rate: 0.05% ✓

Thresholds:
  ✓ http_req_duration{p(95)}<500
  ✓ http_req_duration{p(99)}<1000
  ✓ http_req_failed<0.001
  ✓ http_reqs>900
```

## Performance Targets

### Production Readiness Targets

| Metric | Target | Measured At |
|--------|--------|-------------|
| Throughput | 1000 RPS | Peak load |
| P95 Latency | < 500ms | Peak load |
| P99 Latency | < 1000ms | Peak load |
| Error Rate | < 0.1% | All tests |
| Availability | > 99.9% | All tests |

### Resource Utilization Targets

| Resource | Target | Limit |
|----------|--------|-------|
| CPU | < 70% | < 90% |
| Memory | < 75% | < 85% |
| DB Connections | < 80% | < 90% |
| Cache Hit Rate | > 85% | > 70% |

## Monitoring During Load Tests

### Prometheus Metrics

Monitor these metrics during load tests:

```promql
# Request rate
rate(http_requests_total[1m])

# Response time percentiles
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[1m]))
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[1m]))

# Error rate
rate(http_requests_total{status=~"5.."}[1m]) / rate(http_requests_total[1m])

# Database connection pool
db_connection_pool_in_use / db_connection_pool_size

# Cache hit rate
rate(cache_hits_total[1m]) / rate(cache_operations_total[1m])
```

### Grafana Dashboards

View real-time metrics during load tests:
- **Overview Dashboard**: http://localhost:3000/d/erpgo-overview
- **Performance Dashboard**: http://localhost:3000/d/erpgo-performance
- **Database Dashboard**: http://localhost:3000/d/erpgo-database

## Troubleshooting

### Common Issues

#### 1. Connection Refused
```
Error: dial tcp 127.0.0.1:8080: connect: connection refused
```
**Solution**: Ensure the application is running on the specified BASE_URL

#### 2. High Error Rate
```
Error Rate: 15.3% ✗
```
**Solutions**:
- Check application logs for errors
- Verify database connectivity
- Check resource utilization (CPU, memory)
- Review rate limiting configuration

#### 3. High Response Times
```
P95: 2500.00ms ✗
```
**Solutions**:
- Check database query performance
- Verify cache hit rate
- Review connection pool sizing
- Check for N+1 query problems

#### 4. Test Timeout
```
Error: test execution timed out
```
**Solutions**:
- Increase test timeout in k6 options
- Reduce concurrent users
- Check system resources

### Debug Mode

Run tests with verbose output:

```bash
# Enable k6 debug logging
k6 run --verbose tests/load/k6/baseline-load-test.js

# Enable HTTP debug logging
k6 run --http-debug tests/load/k6/baseline-load-test.js

# Save detailed logs
k6 run --log-output=file=test.log tests/load/k6/baseline-load-test.js
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Load Tests

on:
  schedule:
    - cron: '0 2 * * *'  # Run daily at 2 AM
  workflow_dispatch:

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install k6
        run: |
          sudo gpg -k
          sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6
      
      - name: Start application
        run: docker-compose up -d
      
      - name: Wait for application
        run: |
          timeout 60 bash -c 'until curl -f http://localhost:8080/health/live; do sleep 2; done'
      
      - name: Run load tests
        run: ./tests/load/run-load-tests.sh baseline peak
      
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: load-test-results
          path: tests/load/results/
```

## Best Practices

### 1. Test Environment
- Use dedicated test environment
- Match production configuration
- Isolate from other tests
- Clean state between tests

### 2. Test Data
- Use realistic test data
- Avoid hardcoded IDs
- Clean up after tests
- Use separate test database

### 3. Monitoring
- Monitor system resources
- Track database performance
- Watch for memory leaks
- Check error logs

### 4. Analysis
- Compare against baselines
- Track trends over time
- Identify bottlenecks
- Document findings

### 5. Continuous Testing
- Run regularly (daily/weekly)
- Test before releases
- Monitor production metrics
- Update baselines as needed

## Advanced Usage

### Custom Scenarios

Create custom test scenarios by modifying the k6 scripts:

```javascript
export const options = {
  scenarios: {
    custom_scenario: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 100 },
        { duration: '5m', target: 100 },
        { duration: '2m', target: 0 },
      ],
      gracefulRampDown: '30s',
    },
  },
};
```

### Cloud Execution

Run tests from k6 Cloud:

```bash
# Login to k6 Cloud
k6 login cloud

# Run test in cloud
k6 cloud tests/load/k6/peak-load-test.js
```

### Distributed Testing

Run tests from multiple locations:

```bash
# Run from multiple k6 instances
k6 run --out cloud tests/load/k6/peak-load-test.js
```

## Support

For issues or questions:
1. Check application logs: `docker-compose logs -f api`
2. Review Grafana dashboards
3. Check Prometheus metrics
4. Consult the troubleshooting guide above

## References

- [k6 Documentation](https://k6.io/docs/)
- [k6 Best Practices](https://k6.io/docs/testing-guides/test-types/)
- [Production Readiness Requirements](../../.kiro/specs/production-readiness/requirements.md)
- [System Design Document](../../.kiro/specs/production-readiness/design.md)
