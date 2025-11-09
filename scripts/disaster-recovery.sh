#!/bin/bash

# ERPGo Disaster Recovery Script
# Provides comprehensive disaster recovery procedures with RTO/RPO validation

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_FILE="${PROJECT_ROOT}/.env.production"
DR_LOG="${PROJECT_ROOT}/logs/disaster_recovery.log"
DR_BACKUP_DIR="${PROJECT_ROOT}/backups/disaster_recovery"

# RTO/RPO Targets (in seconds)
RTO_TARGET=1800  # 30 minutes
RPO_TARGET=300   # 5 minutes

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Ensure log directory exists
mkdir -p "$(dirname "$DR_LOG")"
mkdir -p "$DR_BACKUP_DIR"

# Logging functions
log() {
    local message="[$(date '+%Y-%m-%d %H:%M:%S')] $1"
    echo -e "${BLUE}$message${NC}"
    echo "$message" >> "$DR_LOG"
}

error() {
    local message="[ERROR] $1"
    echo -e "${RED}$message${NC}" >&2
    echo "$message" >> "$DR_LOG"
}

warn() {
    local message="[WARN] $1"
    echo -e "${YELLOW}$message${NC}"
    echo "$message" >> "$DR_LOG"
}

success() {
    local message="[SUCCESS] $1"
    echo -e "${GREEN}$message${NC}"
    echo "$message" >> "$DR_LOG"
}

# Send critical notification
send_critical_notification() {
    local message="$1"

    log "CRITICAL: $message"

    # Send to all available notification channels
    if [[ -n "${SLACK_WEBHOOK_URL:-}" ]]; then
        curl -X POST "$SLACK_WEBHOOK_URL" \
            -H 'Content-type: application/json' \
            --data "{\"text\":\"🚨 CRITICAL: $message\",\"color\":\"danger\"}" \
            2>/dev/null || true
    fi

    if command -v mail &> /dev/null && [[ -n "${NOTIFICATION_EMAIL:-}" ]]; then
        echo "CRITICAL: $message" | mail -s "🚨 ERPGo CRITICAL INCIDENT" "$NOTIFICATION_EMAIL" 2>/dev/null || true
    fi

    # You can add PagerDuty, SMS, or other emergency notification methods here
}

# Load environment variables
load_config() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        error "Configuration file not found: $CONFIG_FILE"
        exit 1
    fi

    log "Loading configuration from: $CONFIG_FILE"
    set -a
    source <(grep -v '^#' "$CONFIG_FILE" | grep -v '^$' | grep '=')
    set +a
}

# Create disaster recovery backup
create_dr_backup() {
    local backup_type="${1:-full}"
    local timestamp=$(date '+%Y%m%d_%H%M%S')
    local backup_path="${DR_BACKUP_DIR}/dr_backup_${timestamp}"

    log "Creating disaster recovery backup: $backup_type"

    mkdir -p "$backup_path"

    local start_time=$(date +%s)

    # Database backup
    log "Creating database backup..."
    "${SCRIPT_DIR}/backup/database-backup.sh" backup full
    find "${POSTGRES_BACKUPS_PATH:-./backups/postgres}" -name "*backup_${timestamp}*" -exec cp {} "$backup_path/" \;

    # Application configuration backup
    log "Backing up application configuration..."
    cp "$CONFIG_FILE" "${backup_path}/env.production"
    cp -r "${PROJECT_ROOT}/configs" "${backup_path}/"

    # SSL certificates backup
    if [[ -d "${PROJECT_ROOT}/configs/nginx/ssl" ]]; then
        cp -r "${PROJECT_ROOT}/configs/nginx/ssl" "${backup_path}/"
    fi

    # Data directory backup
    if [[ -d "${UPLOADS_PATH:-./data/uploads}" ]]; then
        cp -r "${UPLOADS_PATH:-./data/uploads}" "${backup_path}/"
    fi

    # Container state backup
    log "Backing up container states..."
    docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" ps > "${backup_path}/containers_state.txt"
    docker images --format "table {{.Repository}}:{{.Tag}}\t{{.ID}}\t{{.CreatedAt}}" > "${backup_path}/images_list.txt"
    docker volume ls > "${backup_path}/volumes_list.txt"

    # System state backup
    log "Backing up system state..."
    df -h > "${backup_path}/disk_usage.txt"
    free -h > "${backup_path}/memory_usage.txt"
    docker system df > "${backup_path}/docker_usage.txt"

    # Create backup metadata
    cat > "${backup_path}/metadata.json" << EOF
{
    "backup_type": "$backup_type",
    "timestamp": "$timestamp",
    "created_at": "$(date -Iseconds)",
    "rto_target": $RTO_TARGET,
    "rpo_target": $RPO_TARGET,
    "version": "$(git rev-parse HEAD 2>/dev/null || echo 'unknown')",
    "environment": "${ENVIRONMENT:-production}",
    "backup_size": "$(du -sh "$backup_path" | cut -f1)"
}
EOF

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    success "Disaster recovery backup completed in ${duration}s"
    log "Backup location: $backup_path"

    echo "$backup_path"
}

# Validate backup integrity
validate_backup() {
    local backup_path="$1"

    if [[ ! -d "$backup_path" ]]; then
        error "Backup directory not found: $backup_path"
        return 1
    fi

    log "Validating backup integrity: $backup_path"

    local validation_errors=0

    # Check metadata file
    if [[ ! -f "${backup_path}/metadata.json" ]]; then
        error "Missing metadata.json"
        ((validation_errors++))
    fi

    # Check database backup
    if ! find "$backup_path" -name "*backup_*.sql*" | head -1 | xargs -I {} test -f "{}"; then
        error "Missing database backup file"
        ((validation_errors++))
    fi

    # Check configuration backup
    if [[ ! -f "${backup_path}/env.production" ]]; then
        error "Missing environment configuration"
        ((validation_errors++))
    fi

    # Check SSL certificates (if they should exist)
    if [[ ! -d "${backup_path}/ssl" ]] && [[ -d "${PROJECT_ROOT}/configs/nginx/ssl" ]]; then
        warn "SSL certificates not included in backup"
    fi

    # Verify backup file sizes
    local backup_size=$(du -sb "$backup_path" | cut -f1)
    if [[ $backup_size -lt 1048576 ]]; then  # Less than 1MB seems too small
        error "Backup size suspiciously small: $backup_size bytes"
        ((validation_errors++))
    fi

    if [[ $validation_errors -gt 0 ]]; then
        error "Backup validation failed with $validation_errors errors"
        return 1
    fi

    success "Backup validation passed"
    return 0
}

# Simulate disaster scenario
simulate_disaster() {
    local scenario="${1:-partial}"

    log "Simulating disaster scenario: $scenario"
    send_critical_notification "Disaster recovery simulation started: $scenario"

    case "$scenario" in
        "partial")
            # Stop application services only
            log "Stopping application services..."
            docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" stop api worker nginx
            ;;
        "database")
            # Stop database services
            log "Stopping database services..."
            docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" stop postgres-primary postgres-replica
            ;;
        "infrastructure")
            # Stop all infrastructure
            log "Stopping all services..."
            docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" down
            ;;
        "data_corruption")
            # Simulate data corruption (in test environment only)
            if [[ "${ENVIRONMENT:-}" == "test" ]]; then
                log "Simulating data corruption..."
                docker exec erpgo-postgres-primary psql -U "${POSTGRES_USER:-erpgo}" -d "${POSTGRES_DB:-erp}" -c "UPDATE users SET email = 'corrupted' WHERE id = (SELECT id FROM users LIMIT 1);"
            else
                error "Data corruption simulation only allowed in test environment"
                return 1
            fi
            ;;
        *)
            error "Unknown disaster scenario: $scenario"
            return 1
            ;;
    esac

    success "Disaster simulation completed: $scenario"
}

# Restore from disaster recovery backup
restore_from_backup() {
    local backup_path="$1"
    local restore_type="${2:-full}"

    if [[ ! -d "$backup_path" ]]; then
        error "Backup directory not found: $backup_path"
        exit 1
    fi

    log "Starting disaster recovery from: $backup_path"
    log "Restore type: $restore_type"

    local start_time=$(date +%s)
    send_critical_notification "Disaster recovery started from backup: $(basename "$backup_path")"

    # Validate backup before restore
    if ! validate_backup "$backup_path"; then
        error "Backup validation failed. Aborting restore."
        exit 1
    fi

    case "$restore_type" in
        "full")
            # Full system restore
            log "Performing full system restore..."

            # Stop all services
            docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" down

            # Restore database
            log "Restoring database..."
            local db_backup=$(find "$backup_path" -name "*backup_*.sql*" | head -1)
            if [[ -f "$db_backup" ]]; then
                "${SCRIPT_DIR}/backup/database-backup.sh" restore "$db_backup"
            fi

            # Restore configuration
            log "Restoring configuration..."
            if [[ -f "${backup_path}/env.production" ]]; then
                cp "${backup_path}/env.production" "$CONFIG_FILE"
            fi

            if [[ -d "${backup_path}/configs" ]]; then
                cp -r "${backup_path}/configs"/* "${PROJECT_ROOT}/configs/"
            fi

            # Restore data files
            if [[ -d "${backup_path}/uploads" ]]; then
                cp -r "${backup_path}/uploads"/* "${UPLOADS_PATH:-./data/uploads}/"
            fi

            # Start all services
            log "Starting services..."
            docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" up -d

            ;;
        "database")
            # Database only restore
            log "Performing database-only restore..."

            local db_backup=$(find "$backup_path" -name "*backup_*.sql*" | head -1)
            if [[ -f "$db_backup" ]]; then
                "${SCRIPT_DIR}/backup/database-backup.sh" restore "$db_backup"
            else
                error "No database backup found in: $backup_path"
                exit 1
            fi

            ;;
        "configuration")
            # Configuration only restore
            log "Performing configuration-only restore..."

            if [[ -f "${backup_path}/env.production" ]]; then
                cp "${backup_path}/env.production" "$CONFIG_FILE"
            fi

            if [[ -d "${backup_path}/configs" ]]; then
                cp -r "${backup_path}/configs"/* "${PROJECT_ROOT}/configs/"
            fi

            # Restart services with new configuration
            docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" restart

            ;;
        *)
            error "Unknown restore type: $restore_type"
            exit 1
            ;;
    esac

    # Wait for services to start
    log "Waiting for services to start..."
    sleep 60

    # Validate restore
    if validate_restore; then
        local end_time=$(date +%s)
        local rto=$((end_time - start_time))

        success "Disaster recovery completed in ${rto}s (RTO target: ${RTO_TARGET}s)"

        if [[ $rto -le $RTO_TARGET ]]; then
            success "RTO target met! (${rto}s <= ${RTO_TARGET}s)"
        else
            warn "RTO target exceeded! (${rto}s > ${RTO_TARGET}s)"
        fi

        send_critical_notification "Disaster recovery completed successfully in ${rto}s"

        # Generate recovery report
        generate_recovery_report "$backup_path" "$restore_type" "$rto"

    else
        error "Disaster recovery validation failed"
        send_critical_notification "Disaster recovery validation failed"
        exit 1
    fi
}

# Validate restore
validate_restore() {
    log "Validating disaster recovery..."

    local validation_errors=0

    # Check database connectivity
    log "Checking database connectivity..."
    if ! docker exec erpgo-postgres-primary pg_isready -U "${POSTGRES_USER:-erpgo}" -d "${POSTGRES_DB:-erp}" &>/dev/null; then
        error "Database connectivity failed"
        ((validation_errors++))
    fi

    # Check Redis connectivity
    log "Checking Redis connectivity..."
    if ! docker exec erpgo-redis-master redis-cli ping &>/dev/null; then
        error "Redis connectivity failed"
        ((validation_errors++))
    fi

    # Check API health
    log "Checking API health..."
    local api_health=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/health" 2>/dev/null || echo "000")
    if [[ "$api_health" != "200" ]]; then
        error "API health check failed (HTTP $api_health)"
        ((validation_errors++))
    fi

    # Check Nginx health
    log "Checking Nginx health..."
    local nginx_health=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost/health" 2>/dev/null || echo "000")
    if [[ "$nginx_health" != "200" ]]; then
        error "Nginx health check failed (HTTP $nginx_health)"
        ((validation_errors++))
    fi

    # Basic data validation
    log "Performing basic data validation..."
    local user_count=$(docker exec erpgo-postgres-primary psql -U "${POSTGRES_USER:-erpgo}" -d "${POSTGRES_DB:-erp}" -t -c "SELECT COUNT(*) FROM users;" 2>/dev/null | xargs || echo "0")
    if [[ $user_count -lt 1 ]]; then
        error "Data validation failed: No users found in database"
        ((validation_errors++))
    fi

    if [[ $validation_errors -gt 0 ]]; then
        error "Restore validation failed with $validation_errors errors"
        return 1
    fi

    success "Restore validation passed"
    return 0
}

# Generate recovery report
generate_recovery_report() {
    local backup_path="$1"
    local restore_type="$2"
    local rto="$3"
    local report_file="${PROJECT_ROOT}/logs/recovery_report_$(date '+%Y%m%d_%H%M%S').log"

    {
        echo "=== ERPGo Disaster Recovery Report ==="
        echo "Timestamp: $(date '+%Y-%m-%d %H:%M:%S')"
        echo ""
        echo "Recovery Details:"
        echo "- Backup: $(basename "$backup_path")"
        echo "- Restore Type: $restore_type"
        echo "- RTO Achieved: ${rto}s"
        echo "- RTO Target: ${RTO_TARGET}s"
        echo "- RTO Met: $([ $rto -le $RTO_TARGET ] && echo 'YES' || echo 'NO')"
        echo ""
        echo "System Status:"
        echo "- Database: $(docker exec erpgo-postgres-primary pg_isready -U "${POSTGRES_USER:-erpgo}" -d "${POSTGRES_DB:-erp}" &>/dev/null && echo 'UP' || echo 'DOWN')"
        echo "- Redis: $(docker exec erpgo-redis-master redis-cli ping &>/dev/null && echo 'UP' || echo 'DOWN')"
        echo "- API: $(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/health" 2>/dev/null || echo "000")"
        echo "- Nginx: $(curl -s -o /dev/null -w "%{http_code}" "http://localhost/health" 2>/dev/null || echo "000")"
        echo ""
        echo "Container Status:"
        docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" ps
        echo ""
        echo "System Resources:"
        df -h
        echo ""
        free -h
        echo ""
        echo "=== End of Report ==="
    } > "$report_file"

    success "Recovery report generated: $report_file"
}

# Test RTO/RPO compliance
test_rto_rpo() {
    log "Testing RTO/RPO compliance..."

    # Create test backup
    local backup_path
    backup_path=$(create_dr_backup test)

    # Calculate RPO (time since last backup)
    local backup_timestamp=$(stat -c %Y "$backup_path" 2>/dev/null || stat -f %m "$backup_path")
    local current_time=$(date +%s)
    local rpo=$((current_time - backup_timestamp))

    log "RPO measured: ${rpo}s (target: ${RPO_TARGET}s)"

    # Simulate disaster
    simulate_disaster partial

    # Measure RTO
    local restore_start=$(date +%s)
    restore_from_backup "$backup_path" full
    local restore_end=$(date +%s)
    local measured_rto=$((restore_end - restore_start))

    # Generate compliance report
    local compliance_report="${PROJECT_ROOT}/logs/rto_rpo_compliance_$(date '+%Y%m%d_%H%M%S').log"
    {
        echo "=== RTO/RPO Compliance Report ==="
        echo "Timestamp: $(date '+%Y-%m-%d %H:%M:%S')"
        echo ""
        echo "RPO (Recovery Point Objective):"
        echo "- Measured: ${rpo}s"
        echo "- Target: ${RPO_TARGET}s"
        echo "- Compliant: $([ $rpo -le $RPO_TARGET ] && echo 'YES' || echo 'NO')"
        echo ""
        echo "RTO (Recovery Time Objective):"
        echo "- Measured: ${measured_rto}s"
        echo "- Target: ${RTO_TARGET}s"
        echo "- Compliant: $([ $measured_rto -le $RTO_TARGET ] && echo 'YES' || echo 'NO')"
        echo ""
        echo "Overall Compliance: $([ $rpo -le $RPO_TARGET ] && [ $measured_rto -le $RTO_TARGET ] && echo 'COMPLIANT' || echo 'NON-COMPLIANT')"
        echo ""
        echo "=== End of Report ==="
    } > "$compliance_report"

    success "RTO/RPO test completed. Report: $compliance_report"
}

# List available DR backups
list_dr_backups() {
    log "Available disaster recovery backups:"
    echo ""

    if [[ -d "$DR_BACKUP_DIR" ]]; then
        for backup in "$DR_BACKUP_DIR"/dr_backup_*; do
            if [[ -d "$backup" ]]; then
                local backup_name=$(basename "$backup")
                local backup_size=$(du -sh "$backup" | cut -f1)
                local backup_date=$(stat -c %y "$backup" 2>/dev/null | cut -d' ' -f1,2 | cut -d'.' -f1)

                echo "Backup: $backup_name"
                echo "  Size: $backup_size"
                echo "  Date: $backup_date"

                if [[ -f "${backup}/metadata.json" ]]; then
                    local backup_type=$(jq -r '.backup_type' "${backup}/metadata.json" 2>/dev/null || echo 'unknown')
                    local version=$(jq -r '.version' "${backup}/metadata.json" 2>/dev/null || echo 'unknown')
                    echo "  Type: $backup_type"
                    echo "  Version: $version"
                fi
                echo ""
            fi
        done
    else
        warn "No disaster recovery backups found"
    fi
}

# Clean up old DR backups
cleanup_old_dr_backups() {
    local retention_days="${1:-30}"

    log "Cleaning up DR backups older than $retention_days days..."

    if [[ -d "$DR_BACKUP_DIR" ]]; then
        find "$DR_BACKUP_DIR" -name "dr_backup_*" -type d -mtime +$retention_days -exec rm -rf {} \;
        success "Old DR backups cleaned up"
    else
        warn "DR backup directory not found: $DR_BACKUP_DIR"
    fi
}

# Main execution function
main() {
    local action="${1:-help}"

    log "Starting ERPGo disaster recovery script..."
    log "Action: $action"

    # Load configuration
    load_config

    case "$action" in
        "backup")
            create_dr_backup "${2:-full}"
            ;;
        "restore")
            restore_from_backup "$2" "${3:-full}"
            ;;
        "simulate")
            simulate_disaster "${2:-partial}"
            ;;
        "test")
            test_rto_rpo
            ;;
        "validate")
            validate_backup "$2"
            ;;
        "list")
            list_dr_backups
            ;;
        "cleanup")
            cleanup_old_dr_backups "${2:-30}"
            ;;
        *)
            error "Unknown action: $action"
            echo "Usage: $0 {backup|restore|simulate|test|validate|list|cleanup}"
            exit 1
            ;;
    esac

    success "Disaster recovery script completed"
}

# Show usage
usage() {
    echo "ERPGo Disaster Recovery Script"
    echo ""
    echo "Usage:"
    echo "  $0 [ACTION] [OPTIONS]"
    echo ""
    echo "Actions:"
    echo "  backup [type]                 Create DR backup (default: full)"
    echo "  restore <backup_path> [type]  Restore from backup (default: full)"
    echo "  simulate [scenario]           Simulate disaster (default: partial)"
    echo "  test                          Test RTO/RPO compliance"
    echo "  validate <backup_path>        Validate backup integrity"
    echo "  list                          List available DR backups"
    echo "  cleanup [days]                Clean up old backups (default: 30)"
    echo ""
    echo "Disaster Scenarios:"
    echo "  partial                       Stop application services"
    echo "  database                      Stop database services"
    echo "  infrastructure                Stop all services"
    echo "  data_corruption               Simulate data corruption (test only)"
    echo ""
    echo "Restore Types:"
    echo "  full                          Full system restore"
    echo "  database                      Database only restore"
    echo "  configuration                 Configuration only restore"
    echo ""
    echo "Examples:"
    echo "  $0 backup full                # Create full DR backup"
    echo "  $0 restore /path/to/backup    # Full restore from backup"
    echo "  $0 simulate database          # Simulate database disaster"
    echo "  $0 test                       # Test RTO/RPO compliance"
    echo ""
}

# Command handling
case "${1:-help}" in
    "help"|"-h"|"--help")
        usage
        ;;
    *)
        main "$@"
        ;;
esac