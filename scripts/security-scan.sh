#!/bin/bash

# ERPGo Security Scanning Script
# This script runs various security vulnerability scans on the codebase

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REPORTS_DIR="${PROJECT_ROOT}/security-reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Create reports directory
mkdir -p "$REPORTS_DIR"

# Logging function
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Check if a tool is installed
check_tool() {
    local tool=$1
    local url=${2:-"https://github.com"}

    if ! command -v "$tool" &> /dev/null; then
        error "$tool is not installed. Please install it from $url"
        return 1
    fi
    return 0
}

# Run gosec security scanner
run_gosec() {
    log "Running gosec security scanner..."

    if ! check_tool "gosec" "https://github.com/securecodewarrior/gosec"; then
        warn "Skipping gosec scan - tool not installed"
        return 0
    fi

    local report_file="${REPORTS_DIR}/gosec-report-${TIMESTAMP}.json"

    cd "$PROJECT_ROOT"

    # Run gosec with configuration
    if gosec -conf .gosec.yml -fmt json -out "$report_file" ./...; then
        local issues=$(jq '.Issues | length' "$report_file" 2>/dev/null || echo "0")
        if [ "$issues" -eq 0 ]; then
            success "gosec scan completed with no issues found"
        else
            warn "gosec scan completed with $issues issues found"
            log "Report saved to: $report_file"
        fi
    else
        error "gosec scan failed"
        return 1
    fi
}

# Run go vet
run_go_vet() {
    log "Running go vet..."

    cd "$PROJECT_ROOT"

    local report_file="${REPORTS_DIR}/go-vet-report-${TIMESTAMP}.txt"

    if go vet ./... > "$report_file" 2>&1; then
        success "go vet completed with no issues found"
    else
        warn "go vet found issues"
        log "Report saved to: $report_file"
    fi
}

# Run go mod audit
run_go_mod_audit() {
    log "Running go mod audit..."

    cd "$PROJECT_ROOT"

    local report_file="${REPORTS_DIR}/go-mod-audit-${TIMESTAMP}.txt"

    # Check for known vulnerabilities in dependencies
    if command -v "govulncheck" &> /dev/null; then
        if govulncheck ./... > "$report_file" 2>&1; then
            success "No vulnerabilities found in dependencies"
        else
            warn "Vulnerabilities found in dependencies"
            log "Report saved to: $report_file"
        fi
    else
        # Fallback: list dependencies for manual review
        go list -json -m all | grep -E '"Module"|"Version"' > "$report_file" 2>&1
        log "Dependency list saved to: $report_file"
        warn "govulncheck not installed, please install it for vulnerability scanning: go install golang.org/x/vuln/cmd/govulncheck@latest"
    fi
}

# Run staticcheck for advanced static analysis
run_staticcheck() {
    log "Running staticcheck..."

    if ! check_tool "staticcheck" "https://staticcheck.io"; then
        warn "Skipping staticcheck - tool not installed"
        return 0
    fi

    cd "$PROJECT_ROOT"

    local report_file="${REPORTS_DIR}/staticcheck-report-${TIMESTAMP}.txt"

    if staticcheck ./... > "$report_file" 2>&1; then
        success "staticcheck completed with no issues found"
    else
        warn "staticcheck found issues"
        log "Report saved to: $report_file"
    fi
}

# Run ineffassign for ineffective assignments
run_ineffassign() {
    log "Running ineffassign..."

    if ! check_tool "ineffassign" "https://github.com/gordonklaus/ineffassign"; then
        warn "Skipping ineffassign - tool not installed"
        return 0
    fi

    cd "$PROJECT_ROOT"

    local report_file="${REPORTS_DIR}/ineffassign-report-${TIMESTAMP}.txt"

    if ineffassign ./... > "$report_file" 2>&1; then
        success "ineffassign completed with no issues found"
    else
        warn "ineffassign found issues"
        log "Report saved to: $report_file"
    fi
}

# Run misspell for spelling mistakes
run_misspell() {
    log "Running misspell..."

    if ! check_tool "misspell" "https://github.com/client9/misspell"; then
        warn "Skipping misspell - tool not installed"
        return 0
    fi

    cd "$PROJECT_ROOT"

    local report_file="${REPORTS_DIR}/misspell-report-${TIMESTAMP}.txt"

    if misspell -error . > "$report_file" 2>&1; then
        success "misspell completed with no spelling mistakes found"
    else
        warn "misspell found spelling mistakes"
        log "Report saved to: $report_file"
    fi
}

# Run golangci-lint (comprehensive linter)
run_golangci_lint() {
    log "Running golangci-lint..."

    if ! check_tool "golangci-lint" "https://golangci-lint.run"; then
        warn "Skipping golangci-lint - tool not installed"
        return 0
    fi

    cd "$PROJECT_ROOT"

    local report_file="${REPORTS_DIR}/golangci-lint-report-${TIMESTAMP}.json"

    # Run with configuration file if it exists
    if [ -f ".golangci.yml" ]; then
        golangci-lint run --out-format json --output-file "$report_file"
    else
        golangci-lint run --enable-all --disable=godot,exhaustivestruct --out-format json --output-file "$report_file"
    fi

    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        success "golangci-lint completed with no issues found"
    else
        warn "golangci-lint found issues"
        log "Report saved to: $report_file"
    fi
}

# Check for secrets in code
run_secrets_scan() {
    log "Running secrets scan..."

    if ! check_tool "trufflehog" "https://github.com/trufflesecurity/trufflehog"; then
        warn "Skipping secrets scan - tool not installed"
        return 0
    fi

    cd "$PROJECT_ROOT"

    local report_file="${REPORTS_DIR}/trufflehog-report-${TIMESTAMP}.json"

    # Exclude common non-secret directories
    if trufflehog filesystem --directory . --output-file "$report_file" \
        --exclude "vendor/*,test*,*.md,*.txt,*.json,*.yml,*.yaml,node_modules,.git/*" \
        --json; then
        success "trufflehog scan completed"
    else
        warn "trufflehog scan found potential secrets"
        log "Report saved to: $report_file"
    fi
}

# Check Docker security
run_docker_security() {
    log "Running Docker security scan..."

    if [ ! -f "Dockerfile" ]; then
        log "No Dockerfile found, skipping Docker security scan"
        return 0
    fi

    if ! check_tool "hadolint" "https://github.com/hadolint/hadolint"; then
        warn "Skipping Docker security scan - hadolint not installed"
        return 0
    fi

    cd "$PROJECT_ROOT"

    local report_file="${REPORTS_DIR}/hadolint-report-${TIMESTAMP}.txt"

    if hadolint Dockerfile > "$report_file" 2>&1; then
        success "hadolint completed with no issues found"
    else
        warn "hadolint found issues in Dockerfile"
        log "Report saved to: $report_file"
    fi
}

# Generate summary report
generate_summary() {
    log "Generating security scan summary..."

    local summary_file="${REPORTS_DIR}/security-summary-${TIMESTAMP}.md"

    cat > "$summary_file" << EOF
# ERPGo Security Scan Summary

**Date:** $(date)
**Scan ID:** ${TIMESTAMP}

## Executive Summary

This report contains the results of various security scans performed on the ERPGo codebase.

## Scan Results

### Tools Run

EOF

    # Add results for each tool
    for tool in gosec go-vet go-mod-audit staticcheck ineffassign misspell golangci-lint trufflehog hadolint; do
        local report_pattern="${REPORTS_DIR}/${tool}-report-*.txt"
        local report_json_pattern="${REPORTS_DIR}/${tool}-report-*.json"

        if ls $report_pattern 1> /dev/null 2>&1 || ls $report_json_pattern 1> /dev/null 2>&1; then
            echo "- ✅ $tool: Completed" >> "$summary_file"
        else
            echo "- ⚠️ $tool: Skipped or failed" >> "$summary_file"
        fi
    done

    cat >> "$summary_file" << EOF

### Detailed Reports

The following detailed reports are available:

EOF

    # List all report files
    for report in "${REPORTS_DIR}"/*report-${TIMESTAMP}*; do
        if [ -f "$report" ]; then
            local filename=$(basename "$report")
            local filesize=$(stat -f%z "$report" 2>/dev/null || stat -c%s "$report" 2>/dev/null || echo "unknown")
            echo "- [$filename](./$filename) ($filesize bytes)" >> "$summary_file"
        fi
    done

    cat >> "$summary_file" << EOF

### Recommendations

1. Review all high-severity findings immediately
2. Address medium-severity findings in the next sprint
3. Implement automated scanning in CI/CD pipeline
4. Regular dependency updates and vulnerability scanning
5. Security code reviews for sensitive changes

### Next Steps

1. Prioritize fixes based on severity and exploitability
2. Update dependencies to latest secure versions
3. Implement security testing in development workflow
4. Schedule regular security scans

---

*This report was generated automatically by the ERPGo security scanning script.*
EOF

    success "Security scan summary generated: $summary_file"
}

# Cleanup old reports (keep last 10)
cleanup_old_reports() {
    log "Cleaning up old security reports..."

    cd "$REPORTS_DIR"

    # Keep only the 10 most recent reports for each tool
    for tool in gosec go-vet go-mod-audit staticcheck ineffassign misspell golangci-lint trufflehog hadolint; do
        ls -t ${tool}-report-* 2>/dev/null | tail -n +11 | xargs -r rm -f
    done

    # Keep only the 10 most recent summaries
    ls -t security-summary-* 2>/dev/null | tail -n +11 | xargs -r rm -f

    success "Old reports cleaned up"
}

# Main execution
main() {
    log "Starting ERPGo security scanning..."
    log "Project root: $PROJECT_ROOT"
    log "Reports directory: $REPORTS_DIR"

    # Change to project directory
    cd "$PROJECT_ROOT"

    # Run all security scans
    run_go_vet
    run_go_mod_audit
    run_gosec
    run_staticcheck
    run_ineffassign
    run_misspell
    run_golangci_lint
    run_secrets_scan
    run_docker_security

    # Generate summary
    generate_summary

    # Cleanup old reports
    cleanup_old_reports

    success "Security scanning completed!"
    log "All reports are available in: $REPORTS_DIR"

    # Exit with error if any critical issues were found
    # You can customize this logic based on your requirements
    return 0
}

# Run main function
main "$@"