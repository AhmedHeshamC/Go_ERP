#!/bin/bash

# ERPGo Monitoring Verification Script
# This script verifies that all monitoring components are working correctly

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global variables
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
SKIPPED_CHECKS=0

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE} $1 ${NC}"
    echo -e "${BLUE}================================${NC}"
}

# Function to run a check
run_check() {
    local check_name=$1
    local check_command=$2
    local expected_status=${3:-0}

    ((TOTAL_CHECKS++))

    echo -n "Testing $check_name... "

    if eval "$check_command" >/dev/null 2>&1; then
        echo -e "${GREEN}✓ PASS${NC}"
        ((PASSED_CHECKS++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        ((FAILED_CHECKS++))
        return 1
    fi
}

# Function to run a check with timeout
run_check_with_timeout() {
    local check_name=$1
    local check_command=$2
    local timeout=${3:-30}

    ((TOTAL_CHECKS++))

    echo -n "Testing $check_name (timeout: ${timeout}s)... "

    if timeout "$timeout" bash -c "$check_command" >/dev/null 2>&1; then
        echo -e "${GREEN}✓ PASS${NC}"
        ((PASSED_CHECKS++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        ((FAILED_CHECKS++))
        return 1
    fi
}

# Function to check if service is running
check_service_running() {
    local service_name=$1
    local container_name=$2

    run_check "$service_name container running" "docker ps --format '{{.Names}}' | grep -q '^${container_name}$'"
}

# Function to check HTTP endpoint
check_http_endpoint() {
    local service_name=$1
    local url=$2
    local expected_code=${3:-200}

    run_check_with_timeout "$service_name HTTP endpoint" "curl -f -s -o /dev/null -w '%{http_code}' '$url' | grep -q '$expected_code'"
}

# Function to check Prometheus targets
check_prometheus_targets() {
    print_status "Checking Prometheus targets..."

    # Get all targets from Prometheus
    local targets=$(curl -s "http://localhost:9090/api/v1/targets" | jq -r '.data.activeTargets[] | select(.scrapeUrl | contains("localhost:9090") | not) | .labels.job' 2>/dev/null || echo "")

    if [[ -z "$targets" ]]; then
        print_warning "Could not fetch Prometheus targets"
        return 1
    fi

    while IFS= read -r target; do
        if [[ -n "$target" ]]; then
            run_check "Prometheus target: $target" "curl -s 'http://localhost:9090/api/v1/query?query=up{job=\"$target\"}' | jq -r '.data.result[0].value[1]' | grep -q '1'"
        fi
    done <<< "$targets"
}

# Function to check metrics availability
check_metrics_availability() {
    print_status "Checking metrics availability..."

    # Check basic application metrics
    local metrics=(
        "erpgo_http_requests_total"
        "erpgo_http_request_duration_seconds_bucket"
        "erpgo_database_connections"
        "erpgo_cache_hit_rate"
        "erpgo_system_memory_bytes"
        "erpgo_goroutines"
    )

    for metric in "${metrics[@]}"; do
        run_check "Metric available: $metric" "curl -s 'http://localhost:8080/metrics' | grep -q '$metric'"
    done
}

# Function to check Grafana datasources
check_grafana_datasources() {
    print_status "Checking Grafana datasources..."

    # Check if Grafana is accessible
    run_check_with_timeout "Grafana accessibility" "curl -f -s 'http://localhost:3000/api/health'" 60

    # Check datasources
    local datasources=("Prometheus" "Loki")

    for ds in "${datasources[@]}"; do
        run_check "Grafana datasource: $ds" "curl -s -u admin:admin 'http://localhost:3000/api/datasources/name/$ds' | jq -r .name | grep -q '$ds'"
    done
}

# Function to check dashboards availability
check_grafana_dashboards() {
    print_status "Checking Grafana dashboards..."

    local dashboards=(
        "erpgo-system-performance"
        "erpgo-database-performance"
        "erpgo-business-metrics"
        "erpgo-security-metrics"
    )

    for dashboard in "${dashboards[@]}"; do
        run_check "Grafana dashboard: $dashboard" "curl -s -u admin:admin 'http://localhost:3000/api/dashboards/uid/$dashboard' | jq -r .dashboard.uid | grep -q '$dashboard'"
    done
}

# Function to check Loki log ingestion
check_loki_ingestion() {
    print_status "Checking Loki log ingestion..."

    # Check if Loki is ready
    run_check_with_timeout "Loki readiness" "curl -f -s 'http://localhost:3100/ready'" 60

    # Check if we can query logs
    local query='{job="erpgo-logs"}'
    run_check "Loki query capability" "curl -s -G 'http://localhost:3100/loki/api/v1/query' --data-urlencode 'query=$query' | jq -r .status | grep -q 'success'"
}

# Function to check AlertManager configuration
check_alertmanager() {
    print_status "Checking AlertManager..."

    # Check if AlertManager is healthy
    run_check_with_timeout "AlertManager health" "curl -f -s 'http://localhost:9093/-/healthy'" 60

    # Check alert rules
    run_check "AlertManager configuration" "curl -s 'http://localhost:9093/api/v1/status' | jq -r .data.config | grep -q 'alertmanager.yml'"
}

# Function to test alert generation
test_alert_generation() {
    print_status "Testing alert generation..."

    # This would require simulating conditions that trigger alerts
    # For now, just check if alert rules are loaded in Prometheus

    run_check "Prometheus alert rules loaded" "curl -s 'http://localhost:9090/api/v1/rules' | jq -r '.data.groups[].rules[] | select(.type=="alerting") | .name' | wc -l | grep -q '[1-9]'"
}

# Function to perform load testing
perform_load_test() {
    print_status "Performing basic load test..."

    # Generate some traffic to test monitoring
    for i in {1..10}; do
        curl -s "http://localhost:8080/health" >/dev/null 2>&1 &
    done

    wait

    # Check if metrics are being generated
    run_check "Load test metrics generated" "curl -s 'http://localhost:9090/api/v1/query?query=rate(erpgo_http_requests_total[1m])' | jq -r '.data.result[0].value[1]' | grep -v '^null$'"
}

# Function to check exporters
check_exporters() {
    print_status "Checking exporters..."

    local exporters=(
        "Node Exporter:9100"
        "PostgreSQL Exporter:9187"
        "Redis Exporter:9121"
    )

    for exporter in "${exporters[@]}"; do
        local name=$(echo "$exporter" | cut -d: -f1)
        local port=$(echo "$exporter" | cut -d: -f2)

        check_http_endpoint "$name" "http://localhost:$port/metrics"
        check_service_running "$name" "erpgo-${name,,}"
    done
}

# Function to generate report
generate_report() {
    print_header "Monitoring Verification Report"

    echo "Total Checks:     $TOTAL_CHECKS"
    echo "Passed:           $PASSED_CHECKS"
    echo "Failed:           $FAILED_CHECKS"
    echo "Skipped:          $SKIPPED_CHECKS"
    echo ""

    local success_rate=$((PASSED_CHECKS * 100 / TOTAL_CHECKS))
    echo "Success Rate:     ${success_rate}%"
    echo ""

    if [[ $FAILED_CHECKS -eq 0 ]]; then
        echo -e "${GREEN}✓ All monitoring components are working correctly!${NC}"
    else
        echo -e "${YELLOW}⚠ Some monitoring components may need attention.${NC}"
        echo ""
        echo "Failed checks indicate potential issues with:"
        echo "- Service connectivity"
        echo "- Configuration errors"
        echo "- Resource constraints"
        echo ""
        echo "Please review the failed checks and address any issues."
    fi

    echo ""
    echo "For troubleshooting, refer to:"
    echo "- docs/MONITORING.md"
    echo "- docs/monitoring-runbooks.md"
}

# Function to show help
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -q, --quick         Run only basic checks (skip load testing)"
    echo "  -v, --verbose       Show detailed output"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "This script verifies that all monitoring components are working correctly."
    echo "It tests connectivity, metrics collection, dashboards, and alerting."
}

# Main verification function
run_verification() {
    local quick_mode=${1:-false}

    print_header "ERPGo Monitoring Verification"

    echo "Starting comprehensive monitoring verification..."
    echo ""

    # Basic service connectivity checks
    print_header "Service Connectivity"
    check_service_running "ERPGo API" "erpgo-api"
    check_service_running "PostgreSQL" "erpgo-postgres"
    check_service_running "Redis" "erpgo-redis"
    check_service_running "NATS" "erpgo-nats"
    check_service_running "Prometheus" "erpgo-prometheus"
    check_service_running "Grafana" "erpgo-grafana"
    check_service_running "AlertManager" "erpgo-alertmanager"
    check_service_running "Loki" "erpgo-loki"
    check_service_running "Promtail" "erpgo-promtail"

    echo ""

    # HTTP endpoint checks
    print_header "HTTP Endpoints"
    check_http_endpoint "ERPGo Health" "http://localhost:8080/health"
    check_http_endpoint "ERPGo Metrics" "http://localhost:8080/metrics"
    check_http_endpoint "Prometheus" "http://localhost:9090/-/healthy"
    check_http_endpoint "Grafana" "http://localhost:3000/api/health"
    check_http_endpoint "AlertManager" "http://localhost:9093/-/healthy"
    check_http_endpoint "Loki" "http://localhost:3100/ready"

    echo ""

    # Monitoring-specific checks
    print_header "Monitoring Components"
    check_metrics_availability
    check_prometheus_targets
    check_grafana_datasources
    check_grafana_dashboards
    check_loki_ingestion
    check_alertmanager
    check_exporters

    echo ""

    # Alert generation test
    if [[ "$quick_mode" == "false" ]]; then
        print_header "Alert Testing"
        test_alert_generation

        echo ""

        # Load testing
        print_header "Load Testing"
        perform_load_test
    fi

    echo ""

    # Generate final report
    generate_report

    # Exit with appropriate code
    if [[ $FAILED_CHECKS -eq 0 ]]; then
        exit 0
    else
        exit 1
    fi
}

# Main script execution
main() {
    local quick_mode=false
    local verbose=false

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -q|--quick)
                quick_mode=true
                shift
                ;;
            -v|--verbose)
                verbose=true
                set -x
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # Check prerequisites
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed or not in PATH"
        exit 1
    fi

    if ! command -v curl &> /dev/null; then
        print_error "curl is not installed or not in PATH"
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        print_error "jq is not installed or not in PATH"
        exit 1
    fi

    # Run verification
    run_verification "$quick_mode"
}

# Trap to handle script interruption
trap 'print_error "Script interrupted"; exit 1' INT TERM

# Run main function with all arguments
main "$@"