# Load Testing Quick Start Guide

## Prerequisites

### 1. Install k6

Choose your platform:

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

### 2. Verify Installation

```bash
k6 version
```

## Running Load Tests

### Option 1: Using Make (Recommended)

```bash
# Run all load tests
make load-test

# Run specific tests
make load-test-baseline    # 100 RPS for 5 minutes
make load-test-peak        # 1000 RPS for 5 minutes
make load-test-stress      # Gradually increase to 5000 RPS
make load-test-spike       # Sudden spike from 100 to 2000 RPS
```

### Option 2: Using the Test Runner Script

```bash
# Run all tests
./tests/load/run-load-tests.sh

# Run specific tests
./tests/load/run-load-tests.sh baseline peak

# Test against different environment
BASE_URL=http://staging.example.com ./tests/load/run-load-tests.sh
```

### Option 3: Using k6 Directly

```bash
# Run a specific test
k6 run -e BASE_URL=http://localhost:8080 tests/load/k6/baseline-load-test.js

# Run with custom options
k6 run --vus 100 --duration 5m tests/load/k6/peak-load-test.js
```

## Before Running Tests

### 1. Start the Application

```bash
# Using Docker Compose
docker-compose up -d

# Or start locally
go run cmd/api/main.go
```

### 2. Verify Application is Running

```bash
curl http://localhost:8080/health/live
```

Expected response: `{"status":"ok"}`

### 3. Validate Test Setup

```bash
./tests/load/validate-load-tests.sh
```

## Understanding Results

### Success Indicators

✅ **PASSED** - All thresholds met:
- Response times within limits
- Error rate below threshold
- Throughput meets target

### Key Metrics

- **P95**: 95% of requests faster than this
- **P99**: 99% of requests faster than this
- **Error Rate**: Percentage of failed requests
- **RPS**: Requests per second (throughput)

### Example Output

```
Peak Load Test Results (1000 RPS)
==========================================

Status: ✓ PASSED

HTTP Requests:
  Total: 300000
  Rate: 1000.00 req/s

Response Time:
  P95: 250.40ms ✓
  P99: 480.20ms ✓

Error Rate: 0.05% ✓
```

## Test Scenarios

### Baseline Test (100 RPS)
- **Purpose**: Establish baseline performance
- **Duration**: 5 minutes
- **Target**: P95 < 500ms, Error rate < 0.1%

### Peak Load Test (1000 RPS)
- **Purpose**: Validate production peak load
- **Duration**: 5 minutes
- **Target**: P95 < 500ms, P99 < 1000ms, Error rate < 0.1%

### Stress Test (up to 5000 RPS)
- **Purpose**: Find system breaking point
- **Duration**: 15 minutes
- **Target**: Graceful degradation, no crashes

### Spike Test (100 → 2000 RPS)
- **Purpose**: Test resilience to traffic spikes
- **Duration**: 7.5 minutes
- **Target**: Quick recovery, minimal errors

## Viewing Results

### Results Location

All results are saved to `tests/load/results/`:
- `baseline-load-test-results.json`
- `peak-load-test-results.json`
- `stress-test-results.json`
- `spike-test-results.json`
- `load-test-summary.txt`

### View Summary

```bash
cat tests/load/results/load-test-summary.txt
```

### Analyze Specific Test

```bash
# View full results
cat tests/load/results/peak-load-test-results.json

# Extract specific metrics
cat tests/load/results/peak-load-test-results.json | jq '.metrics.http_req_duration'
```

## Troubleshooting

### Connection Refused

**Problem**: `Error: dial tcp 127.0.0.1:8080: connect: connection refused`

**Solution**:
```bash
# Check if application is running
curl http://localhost:8080/health/live

# Start application if not running
docker-compose up -d
```

### High Error Rate

**Problem**: Error rate > 0.1%

**Solutions**:
1. Check application logs: `docker-compose logs -f api`
2. Verify database connectivity
3. Check resource utilization
4. Review rate limiting configuration

### High Response Times

**Problem**: P95 > 500ms

**Solutions**:
1. Check database query performance
2. Verify cache hit rate
3. Review connection pool sizing
4. Check for N+1 query problems

### k6 Not Found

**Problem**: `k6: command not found`

**Solution**: Install k6 (see Prerequisites section above)

## Monitoring During Tests

### Prometheus Metrics

Access Prometheus at http://localhost:9090

Key queries:
```promql
# Request rate
rate(http_requests_total[1m])

# P95 latency
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[1m]))

# Error rate
rate(http_requests_total{status=~"5.."}[1m]) / rate(http_requests_total[1m])
```

### Grafana Dashboards

Access Grafana at http://localhost:3000

Dashboards:
- Overview: Real-time system metrics
- Performance: Response times and throughput
- Database: Query performance and connections

## Next Steps

1. **Run Baseline Test**: Establish your performance baseline
   ```bash
   make load-test-baseline
   ```

2. **Run Peak Load Test**: Validate production readiness
   ```bash
   make load-test-peak
   ```

3. **Analyze Results**: Review metrics and identify bottlenecks

4. **Optimize**: Address any performance issues

5. **Re-test**: Verify improvements

6. **Automate**: Add to CI/CD pipeline

## Getting Help

- **Documentation**: See `tests/load/README.md` for detailed information
- **Implementation Details**: See `tests/load/LOAD_TEST_IMPLEMENTATION_SUMMARY.md`
- **k6 Documentation**: https://k6.io/docs/

## Common Commands Cheat Sheet

```bash
# Validate setup
./tests/load/validate-load-tests.sh

# Run all tests
make load-test

# Run specific test
make load-test-peak

# Test different environment
BASE_URL=http://staging.example.com make load-test-baseline

# View results
cat tests/load/results/load-test-summary.txt

# Check application health
curl http://localhost:8080/health/live

# View application logs
docker-compose logs -f api

# Monitor metrics
open http://localhost:3000  # Grafana
open http://localhost:9090  # Prometheus
```

## Success Criteria

Your system is production-ready when:

✅ Baseline test passes (100 RPS)  
✅ Peak load test passes (1000 RPS)  
✅ P95 latency < 500ms  
✅ P99 latency < 1000ms  
✅ Error rate < 0.1%  
✅ System handles traffic spikes gracefully  
✅ No memory leaks over extended tests  

Happy load testing! 🚀
