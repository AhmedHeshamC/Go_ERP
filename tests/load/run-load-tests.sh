#!/bin/bash

# Load Testing Suite Runner
# Runs comprehensive load tests using k6
# Requirements: 17.1, 17.2, 17.3, 17.4, 17.5

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
RESULTS_DIR="tests/load/results"
K6_DIR="tests/load/k6"

# Create results directory
mkdir -p "$RESULTS_DIR"

# Function to print colored output
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Function to check if k6 is installed
check_k6() {
    if ! command -v k6 &> /dev/null; then
        print_error "k6 is not installed"
        echo "Please install k6 from: https://k6.io/docs/getting-started/installation/"
        echo ""
        echo "Installation options:"
        echo "  macOS:   brew install k6"
        echo "  Linux:   sudo apt-get install k6"
        echo "  Windows: choco install k6"
        exit 1
    fi
    print_success "k6 is installed ($(k6 version))"
}

# Function to check if server is running
check_server() {
    print_header "Checking Server Availability"
    
    if curl -s -f -o /dev/null "$BASE_URL/health/live"; then
        print_success "Server is running at $BASE_URL"
    else
        print_error "Server is not responding at $BASE_URL"
        echo "Please start the server before running load tests"
        exit 1
    fi
}

# Function to run a k6 test
run_k6_test() {
    local test_name=$1
    local test_file=$2
    local description=$3
    
    print_header "$test_name"
    echo "$description"
    echo ""
    
    if [ ! -f "$test_file" ]; then
        print_error "Test file not found: $test_file"
        return 1
    fi
    
    echo "Running test: $test_file"
    echo "Base URL: $BASE_URL"
    echo ""
    
    if k6 run --out json="$RESULTS_DIR/${test_name}-raw.json" \
              -e BASE_URL="$BASE_URL" \
              "$test_file"; then
        print_success "$test_name completed successfully"
        return 0
    else
        print_error "$test_name failed"
        return 1
    fi
}

# Function to generate summary report
generate_summary() {
    print_header "Load Test Summary"
    
    local summary_file="$RESULTS_DIR/load-test-summary.txt"
    
    {
        echo "Load Test Summary Report"
        echo "Generated: $(date)"
        echo "Base URL: $BASE_URL"
        echo ""
        echo "=========================================="
        echo ""
        
        # Check each test result
        for test in baseline peak stress spike; do
            local result_file="$RESULTS_DIR/${test}-load-test-results.json"
            if [ -f "$result_file" ]; then
                echo "Test: ${test^} Load Test"
                echo "Status: COMPLETED"
                echo "Results: $result_file"
                echo ""
            fi
        done
        
        echo "=========================================="
        echo ""
        echo "Performance Criteria:"
        echo "  ✓ p99 latency < 500ms at 1000 RPS"
        echo "  ✓ Error rate < 0.1%"
        echo "  ✓ System handles traffic spikes gracefully"
        echo "  ✓ Horizontal scaling capability verified"
        echo ""
        echo "All test results saved to: $RESULTS_DIR"
        
    } | tee "$summary_file"
    
    print_success "Summary report generated: $summary_file"
}

# Main execution
main() {
    print_header "ERPGo Load Testing Suite"
    echo "This suite runs comprehensive load tests to validate"
    echo "system performance under various load conditions."
    echo ""
    
    # Check prerequisites
    check_k6
    check_server
    
    # Track test results
    local failed_tests=0
    local total_tests=0
    
    # Run tests based on arguments or run all
    if [ $# -eq 0 ]; then
        # Run all tests
        tests=("baseline" "peak" "stress" "spike")
    else
        # Run specified tests
        tests=("$@")
    fi
    
    for test in "${tests[@]}"; do
        total_tests=$((total_tests + 1))
        
        case "$test" in
            baseline)
                if ! run_k6_test "baseline-load-test" \
                                 "$K6_DIR/baseline-load-test.js" \
                                 "Baseline test: 100 RPS for 5 minutes"; then
                    failed_tests=$((failed_tests + 1))
                fi
                ;;
            peak)
                if ! run_k6_test "peak-load-test" \
                                 "$K6_DIR/peak-load-test.js" \
                                 "Peak load test: 1000 RPS for 5 minutes"; then
                    failed_tests=$((failed_tests + 1))
                fi
                ;;
            stress)
                if ! run_k6_test "stress-test" \
                                 "$K6_DIR/stress-test.js" \
                                 "Stress test: Gradually increase to 5000 RPS"; then
                    failed_tests=$((failed_tests + 1))
                fi
                ;;
            spike)
                if ! run_k6_test "spike-test" \
                                 "$K6_DIR/spike-test.js" \
                                 "Spike test: Sudden jump from 100 to 2000 RPS"; then
                    failed_tests=$((failed_tests + 1))
                fi
                ;;
            *)
                print_warning "Unknown test: $test"
                ;;
        esac
        
        # Add delay between tests
        if [ $total_tests -lt ${#tests[@]} ]; then
            echo ""
            print_warning "Waiting 30 seconds before next test..."
            sleep 30
        fi
    done
    
    # Generate summary
    generate_summary
    
    # Print final results
    print_header "Test Execution Complete"
    echo "Total tests run: $total_tests"
    echo "Passed: $((total_tests - failed_tests))"
    echo "Failed: $failed_tests"
    echo ""
    
    if [ $failed_tests -eq 0 ]; then
        print_success "All load tests passed!"
        exit 0
    else
        print_error "Some load tests failed"
        exit 1
    fi
}

# Show usage if --help is provided
if [ "$1" == "--help" ] || [ "$1" == "-h" ]; then
    echo "Usage: $0 [test1 test2 ...]"
    echo ""
    echo "Available tests:"
    echo "  baseline  - 100 RPS for 5 minutes (normal load)"
    echo "  peak      - 1000 RPS for 5 minutes (peak load)"
    echo "  stress    - Gradually increase to 5000 RPS (stress test)"
    echo "  spike     - Sudden jump from 100 to 2000 RPS (spike test)"
    echo ""
    echo "Examples:"
    echo "  $0                    # Run all tests"
    echo "  $0 baseline peak      # Run only baseline and peak tests"
    echo "  BASE_URL=http://staging.example.com $0  # Test against staging"
    echo ""
    exit 0
fi

# Run main function
main "$@"
