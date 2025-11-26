#!/bin/bash

# Backup System Validation Script
# This script validates that the backup system is properly configured

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Counters
PASSED=0
FAILED=0
WARNINGS=0

# Logging functions
log() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[✓]${NC} $1"
    ((PASSED++))
}

error() {
    echo -e "${RED}[✗]${NC} $1"
    ((FAILED++))
}

warn() {
    echo -e "${YELLOW}[!]${NC} $1"
    ((WARNINGS++))
}

# Validation functions
check_script_exists() {
    local script="$1"
    local name="$2"
    
    if [[ -f "$script" ]]; then
        success "$name script exists"
    else
        error "$name script not found: $script"
    fi
}

check_script_executable() {
    local script="$1"
    local name="$2"
    
    if [[ -x "$script" ]]; then
        success "$name script is executable"
    else
        error "$name script is not executable: $script"
    fi
}

check_directory_exists() {
    local dir="$1"
    local name="$2"
    
    if [[ -d "$dir" ]]; then
        success "$name directory exists: $dir"
    else
        warn "$name directory does not exist: $dir (will be created on first backup)"
    fi
}

check_directory_writable() {
    local dir="$1"
    local name="$2"
    
    if [[ -w "$dir" ]]; then
        success "$name directory is writable"
    else
        error "$name directory is not writable: $dir"
    fi
}

check_docker_available() {
    if command -v docker &> /dev/null; then
        success "Docker is available"
    else
        error "Docker is not available"
    fi
}

check_docker_compose_available() {
    if command -v docker-compose &> /dev/null || docker compose version &> /dev/null 2>&1; then
        success "Docker Compose is available"
    else
        error "Docker Compose is not available"
    fi
}

check_database_container() {
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "postgres"; then
        success "Database container is running"
    else
        warn "Database container is not running"
    fi
}

check_database_connectivity() {
    if docker exec erpgo-postgres-primary pg_isready -U erpgo -d erp &> /dev/null 2>&1; then
        success "Database is accessible"
    else
        warn "Database is not accessible (container may not be running)"
    fi
}

check_env_variable() {
    local var="$1"
    local name="$2"
    local required="$3"
    
    if [[ -n "${!var:-}" ]]; then
        success "$name is set"
    else
        if [[ "$required" == "true" ]]; then
            error "$name is not set (required)"
        else
            warn "$name is not set (optional)"
        fi
    fi
}

check_cron_job() {
    if crontab -l 2>/dev/null | grep -q "automated-backup.sh"; then
        success "Cron job is configured"
    else
        warn "Cron job is not configured (run: ./scripts/backup/automated-backup.sh setup-cron)"
    fi
}

check_documentation() {
    local doc="$1"
    local name="$2"
    
    if [[ -f "$doc" ]]; then
        success "$name documentation exists"
    else
        error "$name documentation not found: $doc"
    fi
}

# Main validation
main() {
    log "Starting backup system validation..."
    echo ""
    
    # Check scripts
    log "Checking backup scripts..."
    check_script_exists "scripts/backup/database-backup.sh" "Database backup"
    check_script_exists "scripts/backup/automated-backup.sh" "Automated backup"
    check_script_exists "scripts/backup/disaster-recovery.sh" "Disaster recovery"
    
    check_script_executable "scripts/backup/database-backup.sh" "Database backup"
    check_script_executable "scripts/backup/automated-backup.sh" "Automated backup"
    check_script_executable "scripts/backup/disaster-recovery.sh" "Disaster recovery"
    echo ""
    
    # Check directories
    log "Checking backup directories..."
    BACKUP_DIR="${POSTGRES_BACKUPS_PATH:-./backups/postgres}"
    check_directory_exists "$BACKUP_DIR" "Backup"
    if [[ -d "$BACKUP_DIR" ]]; then
        check_directory_writable "$BACKUP_DIR" "Backup"
        check_directory_exists "$BACKUP_DIR/encrypted" "Encrypted backup"
        check_directory_exists "$BACKUP_DIR/logs" "Backup logs"
    fi
    echo ""
    
    # Check Docker
    log "Checking Docker environment..."
    check_docker_available
    check_docker_compose_available
    check_database_container
    check_database_connectivity
    echo ""
    
    # Check environment variables
    log "Checking environment variables..."
    check_env_variable "POSTGRES_DB" "POSTGRES_DB" "true"
    check_env_variable "POSTGRES_USER" "POSTGRES_USER" "true"
    check_env_variable "POSTGRES_PASSWORD" "POSTGRES_PASSWORD" "true"
    check_env_variable "POSTGRES_PRIMARY_HOST" "POSTGRES_PRIMARY_HOST" "true"
    check_env_variable "BACKUP_ENCRYPTION_KEY" "BACKUP_ENCRYPTION_KEY" "false"
    check_env_variable "BACKUP_RETENTION_DAILY_DAYS" "BACKUP_RETENTION_DAILY_DAYS" "false"
    check_env_variable "BACKUP_RETENTION_WEEKLY_WEEKS" "BACKUP_RETENTION_WEEKLY_WEEKS" "false"
    check_env_variable "BACKUP_RETENTION_MONTHLY_MONTHS" "BACKUP_RETENTION_MONTHLY_MONTHS" "false"
    echo ""
    
    # Check cron configuration
    log "Checking cron configuration..."
    check_cron_job
    echo ""
    
    # Check documentation
    log "Checking documentation..."
    check_documentation "docs/operations/DISASTER_RECOVERY_PROCEDURES.md" "Disaster Recovery Procedures"
    check_documentation "docs/operations/BACKUP_RUNBOOK.md" "Backup Runbook"
    check_documentation "docs/operations/RECOVERY_TEST_PLAN.md" "Recovery Test Plan"
    check_documentation "docs/operations/README.md" "Operations README"
    echo ""
    
    # Check backup configuration
    log "Checking backup configuration..."
    if [[ -f "scripts/backup/automated-backup.sh" ]]; then
        if grep -q "BACKUP_SCHEDULE.*0 \*/6 \* \* \*" scripts/backup/automated-backup.sh; then
            success "Backup schedule is set to every 6 hours"
        else
            warn "Backup schedule may not be set to every 6 hours"
        fi
        
        if grep -q "BACKUP_RETENTION_DAILY_DAYS" scripts/backup/automated-backup.sh; then
            success "Daily retention policy is configured"
        else
            error "Daily retention policy is not configured"
        fi
        
        if grep -q "BACKUP_RETENTION_WEEKLY_WEEKS" scripts/backup/automated-backup.sh; then
            success "Weekly retention policy is configured"
        else
            error "Weekly retention policy is not configured"
        fi
        
        if grep -q "BACKUP_RETENTION_MONTHLY_MONTHS" scripts/backup/automated-backup.sh; then
            success "Monthly retention policy is configured"
        else
            error "Monthly retention policy is not configured"
        fi
        
        if grep -q "BACKUP_MAX_RETRIES" scripts/backup/automated-backup.sh; then
            success "Retry logic is configured"
        else
            error "Retry logic is not configured"
        fi
    fi
    echo ""
    
    # Summary
    log "Validation Summary:"
    echo "  Passed:   $PASSED"
    echo "  Failed:   $FAILED"
    echo "  Warnings: $WARNINGS"
    echo ""
    
    if [[ $FAILED -eq 0 ]]; then
        success "Backup system validation passed!"
        if [[ $WARNINGS -gt 0 ]]; then
            warn "There are $WARNINGS warnings that should be addressed"
        fi
        exit 0
    else
        error "Backup system validation failed with $FAILED errors"
        exit 1
    fi
}

# Run validation
main
