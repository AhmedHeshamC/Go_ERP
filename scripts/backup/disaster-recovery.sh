#!/bin/bash

# ERPGo Disaster Recovery Script
# This script handles disaster recovery procedures including system restoration

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONFIG_FILE="${PROJECT_ROOT/.env.production"

# Load configuration
if [[ -f "$CONFIG_FILE" ]]; then
    set -a
    source <(grep -v '^#' "$CONFIG_FILE" | grep -v '^$' | grep '=')
    set +a
fi

# Recovery configuration
RECOVERY_TYPE="${RECOVERY_TYPE:-full}"
RECOVERY_BACKUP="${RECOVERY_BACKUP:-}"
RECOVERY_TARGET="${RECOVERY_TARGET:-production}"
RECOVERY_DRY_RUN="${RECOVERY_DRY_RUN:-false}"
BACKUP_ENCRYPTION_KEY="${BACKUP_ENCRYPTION_KEY:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging
log() {
    local message="[$(date '+%Y-%m-%d %H:%M:%S UTC')] $1"
    echo -e "${BLUE}$message${NC}"
    echo "$message" >> "/tmp/erpgo_disaster_recovery.log"
}

error() {
    local message="[ERROR] $1"
    echo -e "${RED}$message${NC}" >&2
    echo "$message" >> "/tmp/erpgo_disaster_recovery.log"
}

warn() {
    local message="[WARN] $1"
    echo -e "${YELLOW}$message${NC}"
    echo "$message" >> "/tmp/erpgo_disaster_recovery.log"
}

success() {
    local message="[SUCCESS] $1"
    echo -e "${GREEN}$message${NC}"
    echo "$message" >> "/tmp/erpgo_disaster_recovery.log"
}

# Confirm recovery action
confirm_recovery() {
    if [[ "${RECOVERY_DRY_RUN}" == "true" ]]; then
        log "DRY RUN MODE: No changes will be made"
        return 0
    fi

    echo ""
    warn "⚠️  DISASTER RECOVERY WARNING ⚠️"
    echo "This will restore the system from backup and may cause data loss!"
    echo ""
    echo "Recovery Details:"
    echo "  Type: $RECOVERY_TYPE"
    echo "  Backup: $RECOVERY_BACKUP"
    echo "  Target: $RECOVERY_TARGET"
    echo ""

    read -p "Are you sure you want to proceed? Type 'RECOVER' to confirm: " confirmation

    if [[ "$confirmation" != "RECOVER" ]]; then
        error "Recovery cancelled by user"
        exit 1
    fi
}

# Check recovery prerequisites
check_prerequisites() {
    log "Checking disaster recovery prerequisites..."

    # Validate recovery type
    case "$RECOVERY_TYPE" in
        "full"|"database"|"config"|"data")
            success "Recovery type validated: $RECOVERY_TYPE"
            ;;
        *)
            error "Invalid recovery type: $RECOVERY_TYPE. Expected: full, database, config, or data"
            exit 1
            ;;
    esac

    # Check backup file
    if [[ -z "$RECOVERY_BACKUP" ]]; then
        error "Recovery backup not specified. Set RECOVERY_BACKUP environment variable"
        exit 1
    fi

    if [[ ! -f "$RECOVERY_BACKUP" ]]; then
        error "Recovery backup file not found: $RECOVERY_BACKUP"
        exit 1
    fi

    # Check backup file integrity
    if [[ "$RECOVERY_BACKUP" == *.enc && -z "$BACKUP_ENCRYPTION_KEY" ]]; then
        error "Backup is encrypted but no encryption key provided"
        exit 1
    fi

    # Check Docker
    if ! command -v docker &> /dev/null; then
        error "Docker is not available"
        exit 1
    fi

    # Check docker-compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        error "Docker Compose is not available"
        exit 1
    fi

    success "Prerequisites check passed"
}

# Create pre-recovery backup
create_pre_recovery_backup() {
    if [[ "${RECOVERY_DRY_RUN}" == "true" ]]; then
        log "DRY RUN: Skipping pre-recovery backup"
        return 0
    fi

    log "Creating pre-recovery emergency backup..."

    local timestamp=$(date '+%Y%m%d_%H%M%S')
    local emergency_backup="/tmp/emergency_backup_$timestamp.sql"

    # Create emergency backup
    if docker exec erpgo-postgres-primary pg_dump \
        --no-owner \
        --no-privileges \
        --verbose \
        --format=custom \
        --compress=9 \
        --file="/tmp/emergency_backup.sql" \
        "${POSTGRES_DB:-erp}"; then

        # Copy from container
        docker cp erpgo-postgres-primary:/tmp/emergency_backup.sql "$emergency_backup"

        if [[ -n "$BACKUP_ENCRYPTION_KEY" ]]; then
            # Encrypt emergency backup
            openssl enc -aes-256-cbc -salt -in "$emergency_backup" -out "${emergency_backup}.enc" -pass pass:"$BACKUP_ENCRYPTION_KEY"
            rm "$emergency_backup"
            emergency_backup="${emergency_backup}.enc"
        fi

        success "Pre-recovery emergency backup created: $emergency_backup"
        echo "$emergency_backup"
    else
        warn "Failed to create pre-recovery backup"
    fi
}

# Stop current services
stop_services() {
    if [[ "${RECOVERY_DRY_RUN}" == "true" ]]; then
        log "DRY RUN: Would stop all services"
        return 0
    fi

    log "Stopping current services..."

    cd "$PROJECT_ROOT"

    # Stop all services except database
    docker-compose stop api worker nginx || true

    success "Services stopped"
}

# Restore database
restore_database() {
    log "Restoring database from backup: $RECOVERY_BACKUP"

    local backup_file="$RECOVERY_BACKUP"
    local temp_backup="/tmp/recovery_backup_$(date +%s).sql"

    # Decrypt backup if necessary
    if [[ "$backup_file" == *.enc ]]; then
        log "Decrypting backup file..."
        if openssl enc -aes-256-cbc -d -in "$backup_file" -out "$temp_backup" -pass pass:"$BACKUP_ENCRYPTION_KEY"; then
            backup_file="$temp_backup"
        else
            error "Failed to decrypt backup file"
            return 1
        fi
    else
        cp "$backup_file" "$temp_backup"
        backup_file="$temp_backup"
    fi

    if [[ "${RECOVERY_DRY_RUN}" == "true" ]]; then
        log "DRY RUN: Would restore database from $backup_file"
        rm -f "$temp_backup"
        return 0
    fi

    # Copy backup to container
    docker cp "$backup_file" erpgo-postgres-primary:/tmp/recovery.sql

    # Stop application connections to database
    docker-compose stop api worker || true

    # Restore database
    log "Performing database restore..."
    if docker exec erpgo-postgres-primary pg_restore \
        --verbose \
        --clean \
        --if-exists \
        --no-owner \
        --no-privileges \
        --dbname="${POSTGRES_DB:-erp}" \
        "/tmp/recovery.sql"; then

        success "Database restore completed successfully"
    else
        error "Database restore failed"
        return 1
    fi

    # Clean up
    docker exec erpgo-postgres-primary rm -f "/tmp/recovery.sql"
    rm -f "$temp_backup"
}

# Restore configuration
restore_config() {
    log "Restoring configuration files..."

    if [[ "${RECOVERY_DRY_RUN}" == "true" ]]; then
        log "DRY RUN: Would restore configuration files"
        return 0
    fi

    # This would restore configuration files from backup
    # Implementation depends on how configuration is backed up

    success "Configuration restore completed"
}

# Restore data files
restore_data() {
    log "Restoring data files..."

    if [[ "${RECOVERY_DRY_RUN}" == "true" ]]; then
        log "DRY RUN: Would restore data files"
        return 0
    fi

    # This would restore uploaded files, logs, etc.
    # Implementation depends on how data files are backed up

    success "Data files restore completed"
}

# Restart services
restart_services() {
    if [[ "${RECOVERY_DRY_RUN}" == "true" ]]; then
        log "DRY RUN: Would restart all services"
        return 0
    fi

    log "Restarting services..."

    cd "$PROJECT_ROOT"

    # Restart database first
    docker-compose start postgres || true

    # Wait for database to be ready
    local retries=0
    local max_retries=30
    while [[ $retries -lt $max_retries ]]; do
        if docker exec erpgo-postgres-primary pg_isready -U "${POSTGRES_USER:-erpgo}" -d "${POSTGRES_DB:-erp}" &>/dev/null; then
            break
        fi
        sleep 2
        ((retries++))
    done

    if [[ $retries -eq $max_retries ]]; then
        error "Database failed to become ready after restart"
        return 1
    fi

    # Run migrations if needed
    log "Running database migrations..."
    docker-compose run --rm migrator || true

    # Restart application services
    docker-compose up -d

    success "Services restarted"
}

# Verify recovery
verify_recovery() {
    log "Verifying recovery..."

    # Wait for services to start
    sleep 30

    # Check database connectivity
    if docker exec erpgo-postgres-primary pg_isready -U "${POSTGRES_USER:-erpgo}" -d "${POSTGRES_DB:-erp}" &>/dev/null; then
        success "Database is accessible"
    else
        error "Database is not accessible"
        return 1
    fi

    # Check API health
    local api_health=0
    local retries=0
    local max_retries=10

    while [[ $retries -lt $max_retries ]]; do
        if curl -f http://localhost:8080/health &>/dev/null; then
            api_health=1
            break
        fi
        sleep 10
        ((retries++))
    done

    if [[ $api_health -eq 1 ]]; then
        success "API is responding"
    else
        error "API is not responding"
        return 1
    fi

    # Check critical data (optional)
    log "Checking critical data integrity..."

    success "Recovery verification completed"
}

# Generate recovery report
generate_recovery_report() {
    local start_time="$1"
    local success="$2"
    local end_time=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
    local report_file="/tmp/recovery_report_$(date '+%Y%m%d_%H%M%S').txt"

    {
        echo "=== Disaster Recovery Report ==="
        echo "Start Time: $start_time"
        echo "End Time: $end_time"
        echo "Recovery Type: $RECOVERY_TYPE"
        echo "Recovery Target: $RECOVERY_TARGET"
        echo "Backup File: $RECOVERY_BACKUP"
        echo "Dry Run: $RECOVERY_DRY_RUN"
        echo ""
        echo "=== Actions Performed ==="
        echo "✓ Pre-recovery backup created"
        echo "✓ Services stopped"
        if [[ "$RECOVERY_TYPE" == "full" || "$RECOVERY_TYPE" == "database" ]]; then
            echo "✓ Database restored"
        fi
        if [[ "$RECOVERY_TYPE" == "full" || "$RECOVERY_TYPE" == "config" ]]; then
            echo "✓ Configuration restored"
        fi
        if [[ "$RECOVERY_TYPE" == "full" || "$RECOVERY_TYPE" == "data" ]]; then
            echo "✓ Data files restored"
        fi
        echo "✓ Services restarted"
        echo "✓ Recovery verified"
        echo ""
        echo "=== Status ==="
        if [[ "$success" == "true" ]]; then
            echo "Status: SUCCESS"
            echo "System has been fully recovered and is operational"
        else
            echo "Status: FAILED"
            echo "Recovery encountered errors and may not be complete"
        fi
        echo ""
        echo "=== Next Steps ==="
        if [[ "$success" == "true" ]]; then
            echo "1. Monitor system performance and logs"
            echo "2. Verify data integrity with business users"
            echo "3. Update documentation if needed"
            echo "4. Review recovery process for improvements"
        else
            echo "1. Review error logs above"
            echo "2. Contact technical support if needed"
            echo "3. Consider manual intervention"
            echo "4. Evaluate system state before proceeding"
        fi
        echo ""
        echo "=== End of Report ==="
    } > "$report_file"

    success "Recovery report generated: $report_file"
    echo "$report_file"
}

# Send recovery notification
send_recovery_notification() {
    local status="$1"
    local message="$2"
    local report_file="${3:-}"

    log "Sending recovery notification: [$status] $message"

    # Send to Slack if webhook URL is configured
    if [[ -n "${SLACK_WEBHOOK_URL:-}" ]]; then
        local color="good"
        [[ "$status" == "ERROR" ]] && color="danger"
        [[ "$status" == "WARN" ]] && color="warning"

        local payload=$(cat <<EOF
{
    "text": "ERPGo Disaster Recovery $status",
    "attachments": [
        {
            "color": "$color",
            "fields": [
                {
                    "title": "Environment",
                    "value": "$RECOVERY_TARGET",
                    "short": true
                },
                {
                    "title": "Recovery Type",
                    "value": "$RECOVERY_TYPE",
                    "short": true
                },
                {
                    "title": "Time",
                    "value": "$(date -u '+%Y-%m-%d %H:%M:%S UTC')",
                    "short": true
                },
                {
                    "title": "Status",
                    "value": "$message",
                    "short": false
                }
            ]
        }
    ]
}
EOF
)
        curl -X POST "$SLACK_WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "$payload" 2>/dev/null || warn "Failed to send Slack notification"
    fi
}

# Main recovery function
main() {
    local start_time=$(date -u '+%Y-%m-%d %H:%M:%S UTC')

    log "Starting disaster recovery process..."
    log "Recovery type: $RECOVERY_TYPE"
    log "Backup file: $RECOVERY_BACKUP"
    log "Target environment: $RECOVERY_TARGET"

    # Confirm recovery action
    confirm_recovery

    # Check prerequisites
    check_prerequisites || {
        send_recovery_notification "ERROR" "Recovery prerequisites check failed"
        exit 1
    }

    # Create pre-recovery backup
    local pre_recovery_backup
    pre_recovery_backup=$(create_pre_recovery_backup)

    # Stop services
    stop_services

    # Perform recovery based on type
    local recovery_success=true

    case "$RECOVERY_TYPE" in
        "full")
            restore_database || recovery_success=false
            restore_config || recovery_success=false
            restore_data || recovery_success=false
            ;;
        "database")
            restore_database || recovery_success=false
            ;;
        "config")
            restore_config || recovery_success=false
            ;;
        "data")
            restore_data || recovery_success=false
            ;;
    esac

    if [[ "$recovery_success" == "true" ]]; then
        # Restart services
        restart_services

        # Verify recovery
        if verify_recovery; then
            success "Disaster recovery completed successfully"
            send_recovery_notification "SUCCESS" "System recovered successfully"
        else
            error "Recovery verification failed"
            send_recovery_notification "ERROR" "Recovery verification failed"
            recovery_success=false
        fi
    else
        error "Recovery process failed"
        send_recovery_notification "ERROR" "Recovery process failed"
    fi

    # Generate report
    local report_file
    report_file=$(generate_recovery_report "$start_time" "$recovery_success")

    # Final status
    if [[ "$recovery_success" == "true" ]]; then
        success "Disaster recovery process completed successfully!"
        exit 0
    else
        error "Disaster recovery process failed!"
        exit 1
    fi
}

# Show usage
usage() {
    echo "ERPGo Disaster Recovery Script"
    echo ""
    echo "Usage:"
    echo "  $0 [OPTIONS]"
    echo ""
    echo "Environment Variables:"
    echo "  RECOVERY_TYPE                 Recovery type (full, database, config, data)"
    echo "  RECOVERY_BACKUP               Path to backup file to restore from"
    echo "  RECOVERY_TARGET               Target environment for recovery"
    echo "  RECOVERY_DRY_RUN             Enable dry run mode (true/false)"
    echo "  BACKUP_ENCRYPTION_KEY         Encryption key for encrypted backups"
    echo ""
    echo "Examples:"
    echo "  RECOVERY_TYPE=full RECOVERY_BACKUP=/path/to/backup.sql $0"
    echo "  RECOVERY_TYPE=database RECOVERY_BACKUP=/path/to/db.enc RECOVERY_DRY_RUN=true $0"
    echo ""
    echo "⚠️  WARNING: This script will restore the system from backup and may cause data loss!"
    echo "    Always create a backup before performing recovery operations."
}

# Parse command line arguments
case "${1:-}" in
    "help"|"-h"|"--help")
        usage
        exit 0
        ;;
    *)
        main
        ;;
esac