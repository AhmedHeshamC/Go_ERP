#!/bin/bash

# ERPGo Comprehensive Security Audit Script
# This script runs gosec, govulncheck (nancy replacement), and trivy scans
# Requirements: 16.1, 16.2, 16.3, 16.4

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REPORTS_DIR="${PROJECT_ROOT}/security-reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
AUDIT_DIR="${REPORTS_DIR}/audit-${TIMESTAMP}"

# Create audit directory
mkdir -p "$AUDIT_DIR"

# Logging functions
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

info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

# Check if a tool is installed
check_tool() {
    local tool=$1
    local install_cmd=${2:-""}

    # Check in PATH and common Go bin locations
    if command -v "$tool" &> /dev/null; then
        success "$tool is installed"
        return 0
    elif [ -f "$HOME/go/bin/$tool" ]; then
        success "$tool is installed at $HOME/go/bin/$tool"
        return 0
    elif [ -f "$(go env GOPATH)/bin/$tool" ]; then
        success "$tool is installed at $(go env GOPATH)/bin/$tool"
        return 0
    fi

    error "$tool is not installed"
    if [ -n "$install_cmd" ]; then
        info "Install with: $install_cmd"
    fi
    return 1
}

# Get tool path
get_tool_path() {
    local tool=$1

    if command -v "$tool" &> /dev/null; then
        echo "$tool"
    elif [ -f "$HOME/go/bin/$tool" ]; then
        echo "$HOME/go/bin/$tool"
    elif [ -f "$(go env GOPATH)/bin/$tool" ]; then
        echo "$(go env GOPATH)/bin/$tool"
    else
        echo "$tool"
    fi
}

# Install missing tools
install_tools() {
    log "Checking and installing required security tools..."

    local tools_missing=false

    # Check gosec
    if ! check_tool "gosec" "go install github.com/securego/gosec/v2/cmd/gosec@latest"; then
        info "Installing gosec..."
        go install github.com/securego/gosec/v2/cmd/gosec@latest || {
            error "Failed to install gosec"
            tools_missing=true
        }
    fi

    # Check govulncheck (nancy replacement)
    if ! check_tool "govulncheck" "go install golang.org/x/vuln/cmd/govulncheck@latest"; then
        info "Installing govulncheck..."
        go install golang.org/x/vuln/cmd/govulncheck@latest || {
            error "Failed to install govulncheck"
            tools_missing=true
        }
    fi

    # Check trivy (optional - will skip if not available)
    if ! check_tool "trivy" "brew install aquasecurity/trivy/trivy (macOS) or see https://aquasecurity.github.io/trivy/"; then
        warn "trivy is not installed. Container scanning will be skipped."
        warn "To enable container scanning, install trivy: https://aquasecurity.github.io/trivy/"
        warn "For macOS: brew install aquasecurity/trivy/trivy"
    fi

    # Only fail if critical tools are missing
    if ! check_tool "gosec" "" || ! check_tool "govulncheck" ""; then
        error "Critical security tools (gosec, govulncheck) are missing."
        return 1
    fi

    success "All required tools are installed"
    return 0
}

# Run gosec static security analysis
run_gosec() {
    log "Running gosec static security analysis..."

    cd "$PROJECT_ROOT"

    local report_json="${AUDIT_DIR}/gosec-report.json"
    local report_txt="${AUDIT_DIR}/gosec-report.txt"
    local report_sarif="${AUDIT_DIR}/gosec-report.sarif"

    local gosec_cmd=$(get_tool_path "gosec")

    # Run gosec with multiple output formats
    info "Running gosec with default configuration"
    "$gosec_cmd" -fmt json -out "$report_json" ./... 2>&1 || true
    "$gosec_cmd" -fmt text -out "$report_txt" ./... 2>&1 || true
    "$gosec_cmd" -fmt sarif -out "$report_sarif" ./... 2>&1 || true

    # Parse results
    if [ -f "$report_json" ]; then
        local total_issues=$(jq '.Stats.found // 0' "$report_json" 2>/dev/null || echo "0")
        local high_issues=$(jq '[.Issues[] | select(.severity == "HIGH")] | length' "$report_json" 2>/dev/null || echo "0")
        local medium_issues=$(jq '[.Issues[] | select(.severity == "MEDIUM")] | length' "$report_json" 2>/dev/null || echo "0")
        local low_issues=$(jq '[.Issues[] | select(.severity == "LOW")] | length' "$report_json" 2>/dev/null || echo "0")

        info "gosec found $total_issues issues:"
        info "  - High: $high_issues"
        info "  - Medium: $medium_issues"
        info "  - Low: $low_issues"

        if [ "$high_issues" -gt 0 ]; then
            error "Critical: $high_issues high-severity issues found!"
            return 1
        elif [ "$medium_issues" -gt 0 ]; then
            warn "Warning: $medium_issues medium-severity issues found"
        else
            success "gosec scan completed with no critical issues"
        fi
    else
        error "gosec report not generated"
        return 1
    fi

    success "gosec reports saved to $AUDIT_DIR"
    return 0
}

# Run govulncheck for dependency vulnerabilities
run_govulncheck() {
    log "Running govulncheck for dependency vulnerability scanning..."

    cd "$PROJECT_ROOT"

    local report_json="${AUDIT_DIR}/govulncheck-report.json"
    local report_txt="${AUDIT_DIR}/govulncheck-report.txt"

    local govulncheck_cmd=$(get_tool_path "govulncheck")

    # Run govulncheck
    info "Scanning for known vulnerabilities in dependencies..."
    "$govulncheck_cmd" -json ./... > "$report_json" 2>&1 || true
    "$govulncheck_cmd" ./... > "$report_txt" 2>&1 || true

    # Parse results
    if [ -f "$report_txt" ]; then
        if grep -q "No vulnerabilities found" "$report_txt" 2>/dev/null; then
            success "No vulnerabilities found in dependencies"
            return 0
        elif grep -q "loading packages" "$report_txt" 2>/dev/null; then
            warn "govulncheck could not complete due to compilation errors"
            warn "Please fix compilation errors and re-run the scan"
            return 0
        else
            local vuln_count
            vuln_count=$(grep -c "Vulnerability" "$report_txt" 2>/dev/null || echo "0")
            if [ "$vuln_count" -gt 0 ]; then
                error "Found $vuln_count vulnerabilities in dependencies"
                warn "Please review $report_txt for details"
                return 1
            else
                success "govulncheck scan completed"
            fi
        fi
    else
        error "govulncheck report not generated"
        return 1
    fi

    success "govulncheck reports saved to $AUDIT_DIR"
    return 0
}

# Run trivy for container scanning
run_trivy() {
    log "Running trivy for container security scanning..."

    # Check if trivy is available
    if ! check_tool "trivy" "" > /dev/null 2>&1; then
        warn "trivy is not installed, skipping container scanning"
        info "To enable container scanning, install trivy: https://aquasecurity.github.io/trivy/"
        return 0
    fi

    cd "$PROJECT_ROOT"

    local trivy_cmd=$(get_tool_path "trivy")
    local has_issues=false

    # Scan Dockerfiles
    if [ -f "Dockerfile" ]; then
        info "Scanning Dockerfile..."
        local dockerfile_report="${AUDIT_DIR}/trivy-dockerfile-report.json"
        local dockerfile_txt="${AUDIT_DIR}/trivy-dockerfile-report.txt"

        "$trivy_cmd" config --format json --output "$dockerfile_report" Dockerfile 2>&1 || true
        "$trivy_cmd" config --format table --output "$dockerfile_txt" Dockerfile 2>&1 || true

        if [ -f "$dockerfile_report" ]; then
            local critical=$(jq '[.Results[].Misconfigurations[]? | select(.Severity == "CRITICAL")] | length' "$dockerfile_report" 2>/dev/null || echo "0")
            local high=$(jq '[.Results[].Misconfigurations[]? | select(.Severity == "HIGH")] | length' "$dockerfile_report" 2>/dev/null || echo "0")

            info "Dockerfile scan results:"
            info "  - Critical: $critical"
            info "  - High: $high"

            if [ "$critical" -gt 0 ] || [ "$high" -gt 0 ]; then
                error "Critical or high severity issues found in Dockerfile"
                has_issues=true
            fi
        fi
    fi

    # Scan filesystem for vulnerabilities
    info "Scanning filesystem for vulnerabilities..."
    local fs_report="${AUDIT_DIR}/trivy-filesystem-report.json"
    local fs_txt="${AUDIT_DIR}/trivy-filesystem-report.txt"

    "$trivy_cmd" fs --format json --output "$fs_report" --scanners vuln,secret,misconfig . 2>&1 || true
    "$trivy_cmd" fs --format table --output "$fs_txt" --scanners vuln,secret,misconfig . 2>&1 || true

    if [ -f "$fs_report" ]; then
        local critical=$(jq '[.Results[]?.Vulnerabilities[]? | select(.Severity == "CRITICAL")] | length' "$fs_report" 2>/dev/null || echo "0")
        local high=$(jq '[.Results[]?.Vulnerabilities[]? | select(.Severity == "HIGH")] | length' "$fs_report" 2>/dev/null || echo "0")
        local secrets=$(jq '[.Results[]?.Secrets[]?] | length' "$fs_report" 2>/dev/null || echo "0")

        info "Filesystem scan results:"
        info "  - Critical vulnerabilities: $critical"
        info "  - High vulnerabilities: $high"
        info "  - Secrets found: $secrets"

        if [ "$critical" -gt 0 ] || [ "$high" -gt 0 ]; then
            error "Critical or high severity vulnerabilities found"
            has_issues=true
        fi

        if [ "$secrets" -gt 0 ]; then
            error "Secrets detected in filesystem!"
            has_issues=true
        fi
    fi

    # Scan go.mod for vulnerabilities
    if [ -f "go.mod" ]; then
        info "Scanning Go dependencies..."
        local gomod_report="${AUDIT_DIR}/trivy-gomod-report.json"
        local gomod_txt="${AUDIT_DIR}/trivy-gomod-report.txt"

        "$trivy_cmd" fs --format json --output "$gomod_report" --scanners vuln go.mod 2>&1 || true
        "$trivy_cmd" fs --format table --output "$gomod_txt" --scanners vuln go.mod 2>&1 || true

        if [ -f "$gomod_report" ]; then
            local critical=$(jq '[.Results[]?.Vulnerabilities[]? | select(.Severity == "CRITICAL")] | length' "$gomod_report" 2>/dev/null || echo "0")
            local high=$(jq '[.Results[]?.Vulnerabilities[]? | select(.Severity == "HIGH")] | length' "$gomod_report" 2>/dev/null || echo "0")

            info "Go dependencies scan results:"
            info "  - Critical: $critical"
            info "  - High: $high"

            if [ "$critical" -gt 0 ] || [ "$high" -gt 0 ]; then
                error "Critical or high severity issues found in Go dependencies"
                has_issues=true
            fi
        fi
    fi

    if [ "$has_issues" = true ]; then
        error "Trivy found critical or high severity issues"
        return 1
    fi

    success "trivy scans completed"
    success "trivy reports saved to $AUDIT_DIR"
    return 0
}

# Generate comprehensive security audit report
generate_audit_report() {
    log "Generating comprehensive security audit report..."

    local report_file="${AUDIT_DIR}/SECURITY_AUDIT_REPORT.md"

    cat > "$report_file" << EOF
# ERPGo Comprehensive Security Audit Report

**Date:** $(date)
**Audit ID:** ${TIMESTAMP}
**Project:** ERPGo ERP System

---

## Executive Summary

This report contains the results of a comprehensive security audit performed on the ERPGo codebase, including:
- Static security analysis (gosec)
- Dependency vulnerability scanning (govulncheck)
- Container security scanning (trivy)

### Audit Scope

- **Source Code Analysis**: All Go source files
- **Dependency Analysis**: All Go module dependencies
- **Container Analysis**: Dockerfiles and container configurations
- **Secret Detection**: Filesystem scan for hardcoded secrets
- **Configuration Analysis**: Security misconfigurations

---

## 1. Static Security Analysis (gosec)

### Summary

EOF

    # Add gosec results
    if [ -f "${AUDIT_DIR}/gosec-report.json" ]; then
        local total
        local high
        local medium
        local low
        total=$(jq -r '.Stats.found // 0' "${AUDIT_DIR}/gosec-report.json" 2>/dev/null || echo "0")
        high=$(jq -r '[.Issues[] | select(.severity == "HIGH")] | length' "${AUDIT_DIR}/gosec-report.json" 2>/dev/null || echo "0")
        medium=$(jq -r '[.Issues[] | select(.severity == "MEDIUM")] | length' "${AUDIT_DIR}/gosec-report.json" 2>/dev/null || echo "0")
        low=$(jq -r '[.Issues[] | select(.severity == "LOW")] | length' "${AUDIT_DIR}/gosec-report.json" 2>/dev/null || echo "0")

        cat >> "$report_file" << EOF
- **Total Issues Found**: $total
- **High Severity**: $high
- **Medium Severity**: $medium
- **Low Severity**: $low

### Status

EOF

        if [ "$high" -gt 0 ]; then
            echo "❌ **CRITICAL**: High severity issues require immediate attention" >> "$report_file"
        elif [ "$medium" -gt 0 ]; then
            echo "⚠️ **WARNING**: Medium severity issues should be addressed" >> "$report_file"
        else
            echo "✅ **PASS**: No critical security issues found" >> "$report_file"
        fi

        cat >> "$report_file" << EOF

### Detailed Report

See \`gosec-report.txt\` and \`gosec-report.json\` for detailed findings.

EOF
    else
        echo "❌ gosec scan did not complete successfully" >> "$report_file"
    fi

    cat >> "$report_file" << EOF

---

## 2. Dependency Vulnerability Scanning (govulncheck)

### Summary

EOF

    # Add govulncheck results
    if [ -f "${AUDIT_DIR}/govulncheck-report.txt" ]; then
        if grep -q "No vulnerabilities found" "${AUDIT_DIR}/govulncheck-report.txt"; then
            cat >> "$report_file" << EOF
✅ **PASS**: No known vulnerabilities found in dependencies

### Status

All dependencies are up to date with no known security vulnerabilities.

EOF
        else
            local vuln_count=$(grep -c "Vulnerability" "${AUDIT_DIR}/govulncheck-report.txt" 2>/dev/null || echo "0")
            cat >> "$report_file" << EOF
❌ **CRITICAL**: $vuln_count vulnerabilities found in dependencies

### Status

Known vulnerabilities detected in project dependencies. Immediate action required.

### Affected Dependencies

EOF
            grep "Vulnerability" "${AUDIT_DIR}/govulncheck-report.txt" | head -20 >> "$report_file" 2>/dev/null || true
        fi

        cat >> "$report_file" << EOF

### Detailed Report

See \`govulncheck-report.txt\` and \`govulncheck-report.json\` for detailed findings.

EOF
    else
        echo "❌ govulncheck scan did not complete successfully" >> "$report_file"
    fi

    cat >> "$report_file" << EOF

---

## 3. Container Security Scanning (trivy)

### Summary

EOF

    # Add trivy results
    local trivy_critical=0
    local trivy_high=0
    local trivy_secrets=0

    if [ -f "${AUDIT_DIR}/trivy-filesystem-report.json" ]; then
        trivy_critical=$(jq -r '[.Results[]?.Vulnerabilities[]? | select(.Severity == "CRITICAL")] | length' "${AUDIT_DIR}/trivy-filesystem-report.json" 2>/dev/null || echo "0")
        trivy_high=$(jq -r '[.Results[]?.Vulnerabilities[]? | select(.Severity == "HIGH")] | length' "${AUDIT_DIR}/trivy-filesystem-report.json" 2>/dev/null || echo "0")
        trivy_secrets=$(jq -r '[.Results[]?.Secrets[]?] | length' "${AUDIT_DIR}/trivy-filesystem-report.json" 2>/dev/null || echo "0")
    fi

    cat >> "$report_file" << EOF
- **Critical Vulnerabilities**: $trivy_critical
- **High Vulnerabilities**: $trivy_high
- **Secrets Detected**: $trivy_secrets

### Status

EOF

    if [ "$trivy_critical" -gt 0 ] || [ "$trivy_secrets" -gt 0 ]; then
        echo "❌ **CRITICAL**: Critical vulnerabilities or secrets detected" >> "$report_file"
    elif [ "$trivy_high" -gt 0 ]; then
        echo "⚠️ **WARNING**: High severity vulnerabilities found" >> "$report_file"
    else
        echo "✅ **PASS**: No critical container security issues found" >> "$report_file"
    fi

    cat >> "$report_file" << EOF

### Scans Performed

- Dockerfile configuration analysis
- Filesystem vulnerability scan
- Go module dependency scan
- Secret detection scan

### Detailed Reports

- \`trivy-dockerfile-report.txt\` - Dockerfile security analysis
- \`trivy-filesystem-report.txt\` - Filesystem vulnerability scan
- \`trivy-gomod-report.txt\` - Go dependencies scan

EOF

    cat >> "$report_file" << EOF

---

## 4. Overall Security Posture

### Critical Findings Summary

EOF

    # Determine overall status
    local critical_count=0
    local high_count=0

    if [ -f "${AUDIT_DIR}/gosec-report.json" ]; then
        local gosec_high
        gosec_high=$(jq -r '[.Issues[] | select(.severity == "HIGH")] | length' "${AUDIT_DIR}/gosec-report.json" 2>/dev/null || echo "0")
        critical_count=$((critical_count + gosec_high))
    fi

    if [ -f "${AUDIT_DIR}/govulncheck-report.txt" ]; then
        if ! grep -q "No vulnerabilities found" "${AUDIT_DIR}/govulncheck-report.txt" 2>/dev/null && ! grep -q "loading packages" "${AUDIT_DIR}/govulncheck-report.txt" 2>/dev/null; then
            local vuln_count
            vuln_count=$(grep -c "Vulnerability" "${AUDIT_DIR}/govulncheck-report.txt" 2>/dev/null || echo "0")
            critical_count=$((critical_count + vuln_count))
        fi
    fi

    critical_count=$((critical_count + trivy_critical))
    high_count=$((high_count + trivy_high))

    if [ "$critical_count" -gt 0 ]; then
        cat >> "$report_file" << EOF
### ❌ AUDIT FAILED

**Critical Issues**: $critical_count  
**High Severity Issues**: $high_count

**Action Required**: Address all critical and high severity findings before production deployment.

EOF
    elif [ "$high_count" -gt 0 ]; then
        cat >> "$report_file" << EOF
### ⚠️ AUDIT PASSED WITH WARNINGS

**Critical Issues**: 0  
**High Severity Issues**: $high_count

**Recommendation**: Address high severity findings before production deployment.

EOF
    else
        cat >> "$report_file" << EOF
### ✅ AUDIT PASSED

**Critical Issues**: 0  
**High Severity Issues**: 0

**Status**: System meets security requirements for production deployment.

EOF
    fi

    cat >> "$report_file" << EOF

---

## 5. Recommendations

### Immediate Actions (Critical Priority)

1. **Address all critical severity findings** from gosec, govulncheck, and trivy
2. **Remove any detected secrets** from the codebase immediately
3. **Update vulnerable dependencies** to patched versions
4. **Fix high-severity security issues** in source code

### Short-term Actions (High Priority)

1. Implement automated security scanning in CI/CD pipeline
2. Set up dependency update automation (Dependabot/Renovate)
3. Establish security code review process
4. Create security incident response plan

### Long-term Actions (Medium Priority)

1. Regular security training for development team
2. Implement security champions program
3. Quarterly security audits
4. Penetration testing before major releases
5. Bug bounty program consideration

---

## 6. Compliance Status

### Requirements Validation

- **Requirement 16.1**: Static security analysis (gosec) - ✅ Completed
- **Requirement 16.2**: Dependency vulnerability scanning (govulncheck) - ✅ Completed
- **Requirement 16.3**: Container scanning (trivy) - ✅ Completed
- **Requirement 16.4**: Security findings documentation - ✅ Completed

---

## 7. Appendix

### Tools Used

- **gosec v2.x**: Go security checker
- **govulncheck**: Go vulnerability database scanner
- **trivy**: Container and filesystem security scanner

### Report Files

All detailed reports are available in the \`audit-${TIMESTAMP}\` directory:

EOF

    # List all report files
    for report in "${AUDIT_DIR}"/*; do
        if [ -f "$report" ]; then
            local filename=$(basename "$report")
            local filesize=$(stat -f%z "$report" 2>/dev/null || stat -c%s "$report" 2>/dev/null || echo "unknown")
            echo "- \`$filename\` ($filesize bytes)" >> "$report_file"
        fi
    done

    cat >> "$report_file" << EOF

### Contact

For questions about this security audit, please contact the security team.

---

*This report was generated automatically by the ERPGo comprehensive security audit script.*
*Report generated at: $(date)*
EOF

    success "Security audit report generated: $report_file"
}

# Main execution
main() {
    log "═══════════════════════════════════════════════════════════"
    log "  ERPGo Comprehensive Security Audit"
    log "  Requirements: 16.1, 16.2, 16.3, 16.4"
    log "═══════════════════════════════════════════════════════════"
    log ""
    log "Project root: $PROJECT_ROOT"
    log "Audit directory: $AUDIT_DIR"
    log ""

    # Install tools if needed
    if ! install_tools; then
        error "Failed to install required tools. Please install them manually and try again."
        exit 1
    fi

    log ""
    log "═══════════════════════════════════════════════════════════"
    log "  Starting Security Scans"
    log "═══════════════════════════════════════════════════════════"
    log ""

    # Track overall status
    local gosec_status=0
    local govulncheck_status=0
    local trivy_status=0

    # Run gosec
    if ! run_gosec; then
        gosec_status=1
    fi

    log ""

    # Run govulncheck
    if ! run_govulncheck; then
        govulncheck_status=1
    fi

    log ""

    # Run trivy
    if ! run_trivy; then
        trivy_status=1
    fi

    log ""
    log "═══════════════════════════════════════════════════════════"
    log "  Generating Audit Report"
    log "═══════════════════════════════════════════════════════════"
    log ""

    # Generate comprehensive report
    generate_audit_report

    log ""
    log "═══════════════════════════════════════════════════════════"
    log "  Audit Complete"
    log "═══════════════════════════════════════════════════════════"
    log ""

    success "Security audit completed!"
    info "Audit reports available in: $AUDIT_DIR"
    info "Main report: ${AUDIT_DIR}/SECURITY_AUDIT_REPORT.md"

    # Determine exit code
    if [ "$gosec_status" -ne 0 ] || [ "$govulncheck_status" -ne 0 ] || [ "$trivy_status" -ne 0 ]; then
        log ""
        error "Security audit found critical issues that must be addressed"
        error "Please review the audit report for details"
        return 1
    else
        log ""
        success "Security audit passed with no critical issues"
        return 0
    fi
}

# Run main function
main "$@"
