#!/bin/bash

# Basic Security Scan Script for ERPGo
# This script runs basic security checks that don't require additional tools

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

# Run go vet
run_go_vet() {
    log "Running go vet..."

    cd "$PROJECT_ROOT"
    local report_file="${REPORTS_DIR}/go-vet-report-${TIMESTAMP}.txt"

    if go vet ./... > "$report_file" 2>&1; then
        success "go vet completed with no critical issues"
    else
        warn "go vet found issues"
        log "Report saved to: $report_file"

        # Count issues
        local issues=$(grep -c "vet:" "$report_file" 2>/dev/null || echo "0")
        log "Total issues found: $issues"
    fi
}

# Run go fmt check
run_go_fmt() {
    log "Running go fmt check..."

    cd "$PROJECT_ROOT"
    local report_file="${REPORTS_DIR}/go-fmt-report-${TIMESTAMP}.txt"

    # Find unformatted files
    local unformatted=$(gofmt -l . 2>/dev/null || true)

    if [ -z "$unformatted" ]; then
        success "All Go files are properly formatted"
    else
        warn "Some Go files need formatting"
        echo "$unformatted" > "$report_file"
        log "Unformatted files saved to: $report_file"
        echo "Files that need formatting:"
        echo "$unformatted"
    fi
}

# Run go mod tidy check
run_go_mod_check() {
    log "Running go mod check..."

    cd "$PROJECT_ROOT"
    local report_file="${REPORTS_DIR}/go-mod-report-${TIMESTAMP}.txt"

    # Check if go mod tidy changes anything
    local original_sum=$(cat go.sum 2>/dev/null || echo "")
    go mod tidy

    if [ "$original_sum" = "$(cat go.sum 2>/dev/null || echo "")" ]; then
        success "go.mod and go.sum are up to date"
    else
        warn "go.mod or go.sum needed updates"
        log "Run 'go mod tidy' to update dependencies"
        echo "Dependencies were updated by go mod tidy" > "$report_file"
    fi
}

# Check for hardcoded secrets
run_secrets_check() {
    log "Running basic secrets check..."

    cd "$PROJECT_ROOT"
    local report_file="${REPORTS_DIR}/secrets-check-${TIMESTAMP}.txt"

    # Basic patterns for secrets
    local patterns=(
        "password.*=.*[\"'].*[\"']"
        "secret.*=.*[\"'].*[\"']"
        "key.*=.*[\"'].*[\"']"
        "token.*=.*[\"'].*[\"']"
        "api_key.*=.*[\"'].*[\"']"
        "AWS_ACCESS_KEY_ID"
        "AWS_SECRET_ACCESS_KEY"
        "DATABASE_URL.*password"
    )

    local found=false
    for pattern in "${patterns[@]}"; do
        if grep -r -i -n "$pattern" --include="*.go" --include="*.yml" --include="*.yaml" --include="*.json" --exclude-dir=vendor . >> "$report_file" 2>/dev/null; then
            found=true
        fi
    done

    if [ "$found" = false ]; then
        success "No obvious hardcoded secrets found"
    else
        warn "Potential hardcoded secrets detected"
        log "Report saved to: $report_file"
        log "Please review the findings and use environment variables instead"
    fi
}

# Check for insecure dependencies
run_dependency_check() {
    log "Running basic dependency check..."

    cd "$PROJECT_ROOT"
    local report_file="${REPORTS_DIR}/dependency-check-${TIMESTAMP}.txt"

    # List all dependencies
    echo "# Dependency List" > "$report_file"
    go list -m all >> "$report_file" 2>&1

    # Check for known problematic modules
    local problematic=(
        "github.com/golang/protobuf"  # Check for old versions
        "golang.org/x/crypto"         # Check for old versions
        "github.com/gin-gonic"         # Check for versions with known vulnerabilities
    )

    echo "" >> "$report_file"
    echo "# Known problematic modules (if any):" >> "$report_file"

    for module in "${problematic[@]}"; do
        if grep "$module" "$report_file" > /dev/null; then
            echo "$module found - check for latest version" >> "$report_file"
        fi
    done

    success "Dependency list generated"
    log "Report saved to: $report_file"
}

# Check file permissions
run_permissions_check() {
    log "Running file permissions check..."

    cd "$PROJECT_ROOT"
    local report_file="${REPORTS_DIR}/permissions-check-${TIMESTAMP}.txt"

    echo "# File Permissions Report" > "$report_file"

    # Check for executable files that shouldn't be executable
    find . -name "*.go" -executable -ls >> "$report_file" 2>/dev/null || true

    # Check for configuration files with too open permissions
    find . -name "*.env*" -type f -exec ls -la {} \; >> "$report_file" 2>/dev/null || true
    find . -name "*.key" -type f -exec ls -la {} \; >> "$report_file" 2>/dev/null || true
    find . -name "*.pem" -type f -exec ls -la {} \; >> "$report_file" 2>/dev/null || true

    # Check for scripts without proper shebang
    find . -name "*.sh" -type f ! -perm -u+x -exec ls -la {} \; >> "$report_file" 2>/dev/null || true

    success "File permissions check completed"
    log "Report saved to: $report_file"
}

# Check Docker configuration
run_docker_check() {
    log "Running Docker configuration check..."

    cd "$PROJECT_ROOT"
    local report_file="${REPORTS_DIR}/docker-check-${TIMESTAMP}.txt"

    echo "# Docker Configuration Report" > "$report_file"

    if [ -f "Dockerfile" ]; then
        echo "## Dockerfile Analysis" >> "$report_file"

        # Check for FROM statements with tags
        grep -n "^FROM" Dockerfile >> "$report_file" 2>/dev/null || true

        # Check for running as root
        if grep -i "USER" Dockerfile > /dev/null; then
            echo "✓ Non-root user specified" >> "$report_file"
        else
            echo "⚠ No USER directive found (may run as root)" >> "$report_file"
        fi

        # Check for exposed ports
        grep -n "EXPOSE" Dockerfile >> "$report_file" 2>/dev/null || true
    else
        echo "No Dockerfile found" >> "$report_file"
    fi

    if [ -f "docker-compose.yml" ]; then
        echo "" >> "$report_file"
        echo "## Docker Compose Analysis" >> "$report_file"

        # Check for default passwords
        if grep -i "password.*:" docker-compose.yml | grep -v "\${" > /dev/null; then
            echo "⚠ Default passwords may be hardcoded in docker-compose.yml" >> "$report_file"
        fi

        # Check for exposed ports
        grep -n "ports:" docker-compose.yml >> "$report_file" 2>/dev/null || true
    fi

    success "Docker configuration check completed"
    log "Report saved to: $report_file"
}

# Generate summary
generate_summary() {
    log "Generating security scan summary..."

    local summary_file="${REPORTS_DIR}/security-summary-${TIMESTAMP}.md"

    cat > "$summary_file" << EOF
# ERPGo Basic Security Scan Summary

**Date:** $(date)
**Scan ID:** ${TIMESTAMP}

## Executive Summary

This report contains the results of basic security scans performed on the ERPGo codebase.

## Scan Results

### Tools Run

EOF

    # Add results for each tool
    for tool in go-vet go-fmt go-mod secrets-check dependency-check permissions-check docker-check; do
        local report_pattern="${REPORTS_DIR}/${tool}-report-*.txt"

        if ls $report_pattern 1> /dev/null 2>&1; then
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

1. Review all findings and prioritize based on severity
2. Fix any compilation errors or warnings
3. Update dependencies to latest secure versions
4. Ensure all secrets are properly externalized
5. Implement automated scanning in CI/CD pipeline

### Next Steps

1. Address any critical security findings immediately
2. Implement proper secret management
3. Set up regular security scanning
4. Review and update security practices

---

*This report was generated automatically by the ERPGo basic security scanning script.*
EOF

    success "Security scan summary generated: $summary_file"
}

# Main execution
main() {
    log "Starting ERPGo basic security scanning..."
    log "Project root: $PROJECT_ROOT"
    log "Reports directory: $REPORTS_DIR"

    # Change to project directory
    cd "$PROJECT_ROOT"

    # Run all security scans
    run_go_vet
    run_go_fmt
    run_go_mod_check
    run_secrets_check
    run_dependency_check
    run_permissions_check
    run_docker_check

    # Generate summary
    generate_summary

    success "Basic security scanning completed!"
    log "All reports are available in: $REPORTS_DIR"

    return 0
}

# Run main function
main "$@"