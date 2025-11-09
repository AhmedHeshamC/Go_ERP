#!/bin/bash

# ERPGo Comprehensive Load Testing Script
# This script runs all load tests and generates comprehensive reports

set -euo pipefail

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
DATABASE_URL="${DATABASE_URL:-postgres://erpgo_user:erpgo_password@localhost:5432/erpgo_db?sslmode=disable}"
REDIS_URL="${REDIS_URL:-redis://localhost:6379/0}"
OUTPUT_DIR="${OUTPUT_DIR:-./load-test-results}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_DIR="$OUTPUT_DIR/report_$TIMESTAMP"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create output directory
create_output_dir() {
    log_info "Creating output directory: $REPORT_DIR"
    mkdir -p "$REPORT_DIR"
    mkdir -p "$REPORT_DIR/logs"
    mkdir -p "$REPORT_DIR/metrics"
    mkdir -p "$REPORT_DIR/reports"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi

    # Check if required services are running
    if ! curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        log_error "ERPGo API is not accessible at $BASE_URL"
        exit 1
    fi

    # Check PostgreSQL connection
    if ! PGPASSWORD=erpgo_password psql -h localhost -U erpgo_user -d erpgo_db -c "SELECT 1;" > /dev/null 2>&1; then
        log_error "Cannot connect to PostgreSQL database"
        exit 1
    fi

    # Check Redis connection
    if ! redis-cli -u "$REDIS_URL" ping > /dev/null 2>&1; then
        log_error "Cannot connect to Redis"
        exit 1
    fi

    log_success "All prerequisites satisfied"
}

# Warm up the system
warm_up_system() {
    log_info "Warming up the system..."

    # Make some initial requests to warm up caches and connections
    for i in {1..10}; do
        curl -s "$BASE_URL/api/v1/products" > /dev/null 2>&1 &
        curl -s "$BASE_URL/api/v1/health" > /dev/null 2>&1 &
    done

    wait
    sleep 5

    log_success "System warm-up completed"
}

# Run API load tests
run_api_load_tests() {
    log_info "Running API load tests..."

    cd "$(dirname "$0")/.."

    # Run API load tests
    go test -tags=load -v ./tests/load/api_load_test.go \
        -base-url="$BASE_URL" \
        -timeout=10m \
        2>&1 | tee "$REPORT_DIR/logs/api_load_tests.log"

    # Check if tests passed
    if [ $? -eq 0 ]; then
        log_success "API load tests completed successfully"
    else
        log_error "API load tests failed"
        return 1
    fi
}

# Run database load tests
run_database_load_tests() {
    log_info "Running database load tests..."

    cd "$(dirname "$0")/.."

    # Run database load tests
    go test -tags=load -v ./tests/load/database_load_test.go \
        -database-url="$DATABASE_URL" \
        -timeout=15m \
        2>&1 | tee "$REPORT_DIR/logs/database_load_tests.log"

    if [ $? -eq 0 ]; then
        log_success "Database load tests completed successfully"
    else
        log_error "Database load tests failed"
        return 1
    fi
}

# Run cache load tests
run_cache_load_tests() {
    log_info "Running cache load tests..."

    cd "$(dirname "$0")/.."

    # Run cache load tests
    go test -tags=load -v ./tests/load/cache_load_test.go \
        -redis-url="$REDIS_URL" \
        -timeout=10m \
        2>&1 | tee "$REPORT_DIR/logs/cache_load_tests.log"

    if [ $? -eq 0 ]; then
        log_success "Cache load tests completed successfully"
    else
        log_error "Cache load tests failed"
        return 1
    fi
}

# Run authentication stress tests
run_auth_stress_tests() {
    log_info "Running authentication stress tests..."

    cd "$(dirname "$0")/.."

    # Run auth stress tests
    go test -tags=load -v ./tests/load/auth_stress_test.go \
        -base-url="$BASE_URL" \
        -timeout=15m \
        2>&1 | tee "$REPORT_DIR/logs/auth_stress_tests.log"

    if [ $? -eq 0 ]; then
        log_success "Authentication stress tests completed successfully"
    else
        log_error "Authentication stress tests failed"
        return 1
    fi
}

# Run performance benchmarks
run_performance_benchmarks() {
    log_info "Running performance benchmarks..."

    cd "$(dirname "$0")/.."

    # Run performance benchmarks
    go test -tags=load -v ./tests/load/performance_benchmark_test.go \
        -base-url="$BASE_URL" \
        -timeout=20m \
        2>&1 | tee "$REPORT_DIR/logs/performance_benchmarks.log"

    if [ $? -eq 0 ]; then
        log_success "Performance benchmarks completed successfully"
    else
        log_error "Performance benchmarks failed"
        return 1
    fi
}

# Collect system metrics during tests
collect_system_metrics() {
    log_info "Starting system metrics collection..."

    # Start system monitoring in background
    (
        while true; do
            echo "$(date): $(top -bn1 | grep -E "(Cpu|Mem|Load)" | head -3)" >> "$REPORT_DIR/metrics/system_metrics.log"
            echo "$(date): $(df -h /)" >> "$REPORT_DIR/metrics/disk_usage.log"
            echo "$(date): $(netstat -i)" >> "$REPORT_DIR/metrics/network_stats.log"
            sleep 10
        done
    ) &

    SYSTEM_METRICS_PID=$!
    echo $SYSTEM_METRICS_PID > "$REPORT_DIR/system_metrics.pid"

    # Database metrics
    (
        while true; do
            echo "$(date): $(PGPASSWORD=erpgo_password psql -h localhost -U erpgo_user -d erpgo_db -c "SELECT count(*) as connections FROM pg_stat_activity;")" >> "$REPORT_DIR/metrics/db_connections.log"
            echo "$(date): $(PGPASSWORD=erpgo_password psql -h localhost -U erpgo_user -d erpgo_db -c "SELECT schemaname,tablename,n_tup_ins,n_tup_upd,n_tup_del FROM pg_stat_user_tables;" )" >> "$REPORT_DIR/metrics/db_stats.log"
            sleep 30
        done
    ) &

    DB_METRICS_PID=$!
    echo $DB_METRICS_PID >> "$REPORT_DIR/system_metrics.pid"

    # Redis metrics
    (
        while true; do
            echo "$(date): $(redis-cli -u "$REDIS_URL" info memory)" >> "$REPORT_DIR/metrics/redis_memory.log"
            echo "$(date): $(redis-cli -u "$REDIS_URL" info stats)" >> "$REPORT_DIR/metrics/redis_stats.log"
            sleep 30
        done
    ) &

    REDIS_METRICS_PID=$!
    echo $REDIS_METRICS_PID >> "$REPORT_DIR/system_metrics.pid"

    log_success "System metrics collection started"
}

# Stop system metrics collection
stop_system_metrics() {
    log_info "Stopping system metrics collection..."

    if [ -f "$REPORT_DIR/system_metrics.pid" ]; then
        while read -r pid; do
            if kill -0 "$pid" 2>/dev/null; then
                kill "$pid"
                log_info "Stopped metrics collection process $pid"
            fi
        done < "$REPORT_DIR/system_metrics.pid"

        rm -f "$REPORT_DIR/system_metrics.pid"
    fi

    log_success "System metrics collection stopped"
}

# Generate comprehensive report
generate_report() {
    log_info "Generating comprehensive load test report..."

    cat > "$REPORT_DIR/reports/load_test_report.md" << EOF
# ERPGo Load Testing Report

**Generated:** $(date)
**Test Duration:** $(cat "$REPORT_DIR/logs/api_load_tests.log" | grep -E "Duration:" | tail -1 || echo "N/A")
**Base URL:** $BASE_URL

## Executive Summary

This report contains the results of comprehensive load testing conducted on the ERPGo system. The tests evaluated API performance, database performance, cache performance, authentication stress testing, and overall system benchmarks.

## Test Results Overview

### API Load Tests
$(cat "$REPORT_DIR/logs/api_load_tests.log" | grep -A 20 "===.*Results===" || echo "Results not available")

### Database Load Tests
$(cat "$REPORT_DIR/logs/database_load_tests.log" | grep -A 15 "===.*Results===" || echo "Results not available")

### Cache Load Tests
$(cat "$REPORT_DIR/logs/cache_load_tests.log" | grep -A 15 "===.*Results===" || echo "Results not available")

### Authentication Stress Tests
$(cat "$REPORT_DIR/logs/auth_stress_tests.log" | grep -A 20 "===.*Results===" || echo "Results not available")

### Performance Benchmarks
$(cat "$REPORT_DIR/logs/performance_benchmarks.log" | grep -A 30 "=== Performance Benchmark Results ===" || echo "Results not available")

## System Metrics

- **Peak CPU Usage:** $(grep -E "Cpu.*us" "$REPORT_DIR/metrics/system_metrics.log" | awk '{print $2}' | sort -n | tail -1 || echo "N/A")%
- **Peak Memory Usage:** $(grep -E "Mem.*used" "$REPORT_DIR/metrics/system_metrics.log" | tail -1 || echo "N/A")
- **Peak Database Connections:** $(cat "$REPORT_DIR/metrics/db_connections.log" | awk '{print $3}' | sort -n | tail -1 || echo "N/A")
- **Peak Redis Memory:** $(grep "used_memory:" "$REPORT_DIR/metrics/redis_memory.log" | tail -1 | cut -d: -f2 || echo "N/A")

## Performance Grades

$(cat "$REPORT_DIR/logs/performance_benchmarks.log" | grep "Overall Performance Grade:" || echo "Grade not available")

## Recommendations

$(cat "$REPORT_DIR/logs/performance_benchmarks.log" | grep -A 10 "=== Recommendations ===" || echo "No recommendations available")

## Detailed Logs

- [API Load Tests](api_load_tests.log)
- [Database Load Tests](database_load_tests.log)
- [Cache Load Tests](cache_load_tests.log)
- [Authentication Stress Tests](auth_stress_tests.log)
- [Performance Benchmarks](performance_benchmarks.log)
- [System Metrics](../metrics/)

## Test Configuration

- **Base URL:** $BASE_URL
- **Database URL:** $DATABASE_URL
- **Redis URL:** $REDIS_URL
- **Test Date:** $(date)
- **Test Environment:** Local Development

---

*This report was automatically generated by the ERPGo load testing script.*
EOF

    log_success "Comprehensive report generated: $REPORT_DIR/reports/load_test_report.md"
}

# Generate JSON summary for CI/CD integration
generate_json_summary() {
    log_info "Generating JSON summary for CI/CD integration..."

    # Extract key metrics and create JSON summary
    cat > "$REPORT_DIR/load_test_summary.json" << EOF
{
    "test_run": {
        "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
        "base_url": "$BASE_URL",
        "duration_seconds": "$(cat "$REPORT_DIR/logs/api_load_tests.log" | grep -oE "Duration: [0-9.]+[a-z]+" | head -1 || echo "0")",
        "overall_grade": "$(cat "$REPORT_DIR/logs/performance_benchmarks.log" | grep "Overall Performance Grade:" | cut -d: -f2 | xargs || echo "N/A")"
    },
    "api_metrics": {
        "total_requests": "$(cat "$REPORT_DIR/logs/api_load_tests.log" | grep "Total Requests:" | head -1 | awk '{print $3}' || echo "0")",
        "requests_per_second": "$(cat "$REPORT_DIR/logs/api_load_tests.log" | grep "Requests Per Second:" | head -1 | awk '{print $4}' || echo "0")",
        "error_rate_percent": "$(cat "$REPORT_DIR/logs/api_load_tests.log" | grep "Error Rate:" | head -1 | awk '{print $3}' | sed 's/%//' || echo "0")",
        "p95_response_time_ms": "$(cat "$REPORT_DIR/logs/api_load_tests.log" | grep "95th Percentile Response Time:" | head -1 | awk '{print $5}' | sed 's/ms//' || echo "0")"
    },
    "database_metrics": {
        "total_queries": "$(cat "$REPORT_DIR/logs/database_load_tests.log" | grep "Total Operations:" | head -1 | awk '{print $3}' || echo "0")",
        "queries_per_second": "$(cat "$REPORT_DIR/logs/database_load_tests.log" | grep "Throughput:" | head -1 | awk '{print $2}' || echo "0")",
        "p95_latency_ms": "$(cat "$REPORT_DIR/logs/database_load_tests.log" | grep "95th Percentile Latency:" | head -1 | awk '{print $4}' | sed 's/ms//' || echo "0")",
        "error_rate_percent": "$(cat "$REPORT_DIR/logs/database_load_tests.log" | grep "Error Rate:" | head -1 | awk '{print $3}' | sed 's/%//' || echo "0")"
    },
    "cache_metrics": {
        "total_operations": "$(cat "$REPORT_DIR/logs/cache_load_tests.log" | grep "Total Operations:" | head -1 | awk '{print $3}' || echo "0")",
        "hit_rate_percent": "$(cat "$REPORT_DIR/logs/cache_load_tests.log" | grep "Hit Rate:" | head -1 | awk '{print $3}' | sed 's/%//' || echo "0")",
        "operations_per_second": "$(cat "$REPORT_DIR/logs/cache_load_tests.log" | grep "Throughput:" | head -1 | awk '{print $2}' || echo "0")",
        "p95_latency_ms": "$(cat "$REPORT_DIR/logs/cache_load_tests.log" | grep "95th Percentile Latency:" | head -1 | awk '{print $4}' | sed 's/ms//' || echo "0")"
    },
    "auth_metrics": {
        "login_attempts": "$(cat "$REPORT_DIR/logs/auth_stress_tests.log" | grep "Login Attempts:" | head -1 | awk '{print $3}' || echo "0")",
        "successful_logins": "$(cat "$REPORT_DIR/logs/auth_stress_tests.log" | grep "Successful Logins:" | head -1 | awk '{print $3}' || echo "0")",
        "brute_force_attempts": "$(cat "$REPORT_DIR/logs/auth_stress_tests.log" | grep "Brute Force Attempts:" | head -1 | awk '{print $4}' || echo "0")",
        "permission_checks": "$(cat "$REPORT_DIR/logs/auth_stress_tests.log" | grep "Permission Checks:" | head -1 | awk '{print $3}' || echo "0")"
    },
    "system_metrics": {
        "peak_cpu_percent": "$(grep -E "Cpu.*us" "$REPORT_DIR/metrics/system_metrics.log" | awk '{print $2}' | sort -n | tail -1 || echo "0")",
        "peak_memory_usage": "$(grep -E "Mem.*used" "$REPORT_DIR/metrics/system_metrics.log" | tail -1 | awk '{print $3}' || echo "0")",
        "peak_db_connections": "$(cat "$REPORT_DIR/metrics/db_connections.log" | awk '{print $3}' | sort -n | tail -1 || echo "0")"
    },
    "test_results": {
        "api_load_tests": "$(grep -q "PASS" "$REPORT_DIR/logs/api_load_tests.log" && echo "PASS" || echo "FAIL")",
        "database_load_tests": "$(grep -q "PASS" "$REPORT_DIR/logs/database_load_tests.log" && echo "PASS" || echo "FAIL")",
        "cache_load_tests": "$(grep -q "PASS" "$REPORT_DIR/logs/cache_load_tests.log" && echo "PASS" || echo "FAIL")",
        "auth_stress_tests": "$(grep -q "PASS" "$REPORT_DIR/logs/auth_stress_tests.log" && echo "PASS" || echo "FAIL")",
        "performance_benchmarks": "$(grep -q "PASS" "$REPORT_DIR/logs/performance_benchmarks.log" && echo "PASS" || echo "FAIL")"
    }
}
EOF

    log_success "JSON summary generated: $REPORT_DIR/load_test_summary.json"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up..."
    stop_system_metrics
}

# Set up trap for cleanup
trap cleanup EXIT INT TERM

# Main execution
main() {
    log_info "Starting ERPGo comprehensive load testing..."
    log_info "Report will be saved to: $REPORT_DIR"

    # Create output directory
    create_output_dir

    # Check prerequisites
    check_prerequisites

    # Warm up system
    warm_up_system

    # Start system metrics collection
    collect_system_metrics

    # Run all tests
    local test_results=()

    if run_api_load_tests; then
        test_results+=("API_LOAD_TESTS:PASS")
    else
        test_results+=("API_LOAD_TESTS:FAIL")
    fi

    if run_database_load_tests; then
        test_results+=("DATABASE_LOAD_TESTS:PASS")
    else
        test_results+=("DATABASE_LOAD_TESTS:FAIL")
    fi

    if run_cache_load_tests; then
        test_results+=("CACHE_LOAD_TESTS:PASS")
    else
        test_results+=("CACHE_LOAD_TESTS:FAIL")
    fi

    if run_auth_stress_tests; then
        test_results+=("AUTH_STRESS_TESTS:PASS")
    else
        test_results+=("AUTH_STRESS_TESTS:FAIL")
    fi

    if run_performance_benchmarks; then
        test_results+=("PERFORMANCE_BENCHMARKS:PASS")
    else
        test_results+=("PERFORMANCE_BENCHMARKS:FAIL")
    fi

    # Generate reports
    generate_report
    generate_json_summary

    # Print final summary
    echo
    log_info "=== LOAD TESTING SUMMARY ==="
    log_info "Report Directory: $REPORT_DIR"
    log_info "Markdown Report: $REPORT_DIR/reports/load_test_report.md"
    log_info "JSON Summary: $REPORT_DIR/load_test_summary.json"

    echo
    log_info "=== TEST RESULTS ==="
    for result in "${test_results[@]}"; do
        IFS=':' read -r test_name test_status <<< "$result"
        if [ "$test_status" = "PASS" ]; then
            log_success "$test_name: PASSED"
        else
            log_error "$test_name: FAILED"
        fi
    done

    # Check overall success
    local failed_tests=0
    for result in "${test_results[@]}"; do
        if [[ "$result" == *":FAIL" ]]; then
            ((failed_tests++))
        fi
    done

    echo
    if [ $failed_tests -eq 0 ]; then
        log_success "All load tests PASSED! 🎉"
        exit 0
    else
        log_error "$failed_tests test(s) FAILED!"
        exit 1
    fi
}

# Parse command line arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo
        echo "Options:"
        echo "  --help, -h              Show this help message"
        echo "  --base-url URL         Set base URL for API tests (default: http://localhost:8080)"
        echo "  --database-url URL      Set database URL (default: postgres://erpgo_user:erpgo_password@localhost:5432/erpgo_db?sslmode=disable)"
        echo "  --redis-url URL         Set Redis URL (default: redis://localhost:6379/0)"
        echo "  --output-dir DIR        Set output directory (default: ./load-test-results)"
        echo
        echo "Environment Variables:"
        echo "  BASE_URL               API base URL"
        echo "  DATABASE_URL           PostgreSQL connection URL"
        echo "  REDIS_URL              Redis connection URL"
        echo "  OUTPUT_DIR             Output directory for reports"
        echo
        exit 0
        ;;
    --base-url)
        BASE_URL="$2"
        shift 2
        ;;
    --database-url)
        DATABASE_URL="$2"
        shift 2
        ;;
    --redis-url)
        REDIS_URL="$2"
        shift 2
        ;;
    --output-dir)
        OUTPUT_DIR="$2"
        shift 2
        ;;
esac

# Run main function
main "$@"