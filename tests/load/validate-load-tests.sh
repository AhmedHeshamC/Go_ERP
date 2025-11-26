#!/bin/bash

# Load Test Validation Script
# Validates that all load test components are properly configured

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "Load Test Suite Validation"
echo "=========================================="
echo ""

# Track validation status
VALIDATION_PASSED=true

# Function to check file exists
check_file() {
    local file=$1
    local description=$2
    
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} $description: $file"
        return 0
    else
        echo -e "${RED}✗${NC} $description: $file (NOT FOUND)"
        VALIDATION_PASSED=false
        return 1
    fi
}

# Function to check directory exists
check_dir() {
    local dir=$1
    local description=$2
    
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✓${NC} $description: $dir"
        return 0
    else
        echo -e "${RED}✗${NC} $description: $dir (NOT FOUND)"
        VALIDATION_PASSED=false
        return 1
    fi
}

# Function to check command exists
check_command() {
    local cmd=$1
    local description=$2
    
    if command -v "$cmd" &> /dev/null; then
        echo -e "${GREEN}✓${NC} $description: $cmd ($(command -v $cmd))"
        return 0
    else
        echo -e "${YELLOW}⚠${NC} $description: $cmd (NOT INSTALLED)"
        return 1
    fi
}

echo "Checking k6 Test Scripts..."
echo "----------------------------"
check_file "tests/load/k6/baseline-load-test.js" "Baseline Load Test"
check_file "tests/load/k6/peak-load-test.js" "Peak Load Test"
check_file "tests/load/k6/stress-test.js" "Stress Test"
check_file "tests/load/k6/spike-test.js" "Spike Test"
echo ""

echo "Checking Infrastructure Files..."
echo "--------------------------------"
check_file "tests/load/run-load-tests.sh" "Test Runner Script"
check_file "tests/load/README.md" "Documentation"
check_file "tests/load/LOAD_TEST_IMPLEMENTATION_SUMMARY.md" "Implementation Summary"
check_dir "tests/load/results" "Results Directory"
echo ""

echo "Checking Script Permissions..."
echo "------------------------------"
if [ -x "tests/load/run-load-tests.sh" ]; then
    echo -e "${GREEN}✓${NC} run-load-tests.sh is executable"
else
    echo -e "${RED}✗${NC} run-load-tests.sh is not executable"
    echo "  Run: chmod +x tests/load/run-load-tests.sh"
    VALIDATION_PASSED=false
fi
echo ""

echo "Checking Dependencies..."
echo "------------------------"
check_command "k6" "k6 Load Testing Tool"
echo ""

echo "Validating k6 Scripts..."
echo "------------------------"
for script in tests/load/k6/*.js; do
    if k6 inspect "$script" &> /dev/null; then
        echo -e "${GREEN}✓${NC} $(basename $script) is valid"
    else
        echo -e "${RED}✗${NC} $(basename $script) has syntax errors"
        VALIDATION_PASSED=false
    fi
done
echo ""

echo "Checking Makefile Targets..."
echo "----------------------------"
if grep -q "load-test:" Makefile; then
    echo -e "${GREEN}✓${NC} Makefile has load-test target"
else
    echo -e "${RED}✗${NC} Makefile missing load-test target"
    VALIDATION_PASSED=false
fi

if grep -q "load-test-baseline:" Makefile; then
    echo -e "${GREEN}✓${NC} Makefile has load-test-baseline target"
else
    echo -e "${RED}✗${NC} Makefile missing load-test-baseline target"
    VALIDATION_PASSED=false
fi

if grep -q "load-test-peak:" Makefile; then
    echo -e "${GREEN}✓${NC} Makefile has load-test-peak target"
else
    echo -e "${RED}✗${NC} Makefile missing load-test-peak target"
    VALIDATION_PASSED=false
fi

if grep -q "load-test-stress:" Makefile; then
    echo -e "${GREEN}✓${NC} Makefile has load-test-stress target"
else
    echo -e "${RED}✗${NC} Makefile missing load-test-stress target"
    VALIDATION_PASSED=false
fi

if grep -q "load-test-spike:" Makefile; then
    echo -e "${GREEN}✓${NC} Makefile has load-test-spike target"
else
    echo -e "${RED}✗${NC} Makefile missing load-test-spike target"
    VALIDATION_PASSED=false
fi
echo ""

echo "=========================================="
if [ "$VALIDATION_PASSED" = true ]; then
    echo -e "${GREEN}✓ All validations passed!${NC}"
    echo ""
    echo "Load testing suite is ready to use."
    echo ""
    echo "Quick start:"
    echo "  1. Ensure application is running: docker-compose up -d"
    echo "  2. Run all tests: make load-test"
    echo "  3. Or run specific test: make load-test-peak"
    echo ""
    exit 0
else
    echo -e "${RED}✗ Some validations failed${NC}"
    echo ""
    echo "Please fix the issues above before running load tests."
    echo ""
    exit 1
fi
