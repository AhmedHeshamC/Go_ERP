#!/bin/bash

# Script to check if code coverage meets the 80% target for critical paths
# Usage: ./scripts/check-coverage-target.sh

set -e

echo "=========================================="
echo "Code Coverage Target Verification"
echo "Target: 80% for critical paths"
echo "=========================================="
echo ""

# Define critical packages
CRITICAL_PACKAGES=(
    "./internal/domain/inventory/entities"
    "./internal/domain/orders/entities"
    "./internal/domain/products/entities"
    "./pkg/auth"
    "./pkg/secrets"
    "./pkg/health"
    "./pkg/validation"
    "./pkg/errors"
    "./pkg/audit"
    "./pkg/ratelimit"
    "./pkg/shutdown"
    "./pkg/tracing"
)

# Run tests and collect coverage
echo "Running tests for critical packages..."
go test "${CRITICAL_PACKAGES[@]}" -coverprofile=critical_coverage.out -covermode=atomic > /dev/null 2>&1

# Parse coverage results
echo ""
echo "Coverage Results:"
echo "----------------------------------------"

TOTAL_COVERAGE=0
PACKAGE_COUNT=0
PASSING_COUNT=0
FAILING_COUNT=0

for pkg in "${CRITICAL_PACKAGES[@]}"; do
    # Get coverage for this package
    COVERAGE=$(go test "$pkg" -cover 2>&1 | grep -o 'coverage: [0-9.]*%' | grep -o '[0-9.]*' || echo "0")
    
    if [ -n "$COVERAGE" ] && [ "$COVERAGE" != "0" ]; then
        PACKAGE_COUNT=$((PACKAGE_COUNT + 1))
        TOTAL_COVERAGE=$(echo "$TOTAL_COVERAGE + $COVERAGE" | bc)
        
        # Check if meets target
        MEETS_TARGET=$(echo "$COVERAGE >= 80" | bc)
        if [ "$MEETS_TARGET" -eq 1 ]; then
            echo "✅ $pkg: ${COVERAGE}%"
            PASSING_COUNT=$((PASSING_COUNT + 1))
        else
            echo "❌ $pkg: ${COVERAGE}% (needs $(echo "80 - $COVERAGE" | bc)% more)"
            FAILING_COUNT=$((FAILING_COUNT + 1))
        fi
    fi
done

echo "----------------------------------------"

# Calculate average
if [ "$PACKAGE_COUNT" -gt 0 ]; then
    AVG_COVERAGE=$(echo "scale=1; $TOTAL_COVERAGE / $PACKAGE_COUNT" | bc)
    echo ""
    echo "Summary:"
    echo "  Total packages: $PACKAGE_COUNT"
    echo "  Passing (≥80%): $PASSING_COUNT"
    echo "  Failing (<80%): $FAILING_COUNT"
    echo "  Average coverage: ${AVG_COVERAGE}%"
    echo ""
    
    # Check if average meets target
    MEETS_AVG_TARGET=$(echo "$AVG_COVERAGE >= 80" | bc)
    if [ "$MEETS_AVG_TARGET" -eq 1 ]; then
        echo "✅ SUCCESS: Average coverage meets 80% target!"
        exit 0
    else
        GAP=$(echo "80 - $AVG_COVERAGE" | bc)
        echo "❌ FAILED: Average coverage is ${GAP}% below target"
        echo ""
        echo "Recommendations:"
        echo "1. Focus on packages with lowest coverage first"
        echo "2. Add unit tests for untested functions"
        echo "3. Add integration tests for critical flows"
        echo "4. Run: go tool cover -html=critical_coverage.out to see details"
        exit 1
    fi
else
    echo "❌ ERROR: No coverage data collected"
    exit 1
fi
