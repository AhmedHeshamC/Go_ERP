#!/bin/bash

# ERPGo Automated Backup Script
# This script handles scheduled backups with automatic rotation and monitoring

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONFIG_FILE="${PROJECT_ROOT}/.env.production"

# Load configuration
if [[ -f "$CONFIG_FILE" ]]; then
    set -a
    source <(grep -v '^#' "$CONFIG_FILE" | grep -v '^$' | grep '=')
    set +a
fi

# Backup configuration
BACKUP_TYPE="${BACKUP_TYPE:-full}"
BACKUP_SCHEDULE="${BACKUP_SCHEDULE:-0 2 * * *}"  # Daily at 2 AM UTC
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
BACKUP_ENCRYPTION_KEY="${BACKUP_ENCRYPTION_KEY:-}"
BACKUP_STORAGE_TYPE="${BACKUP_STORAGE_TYPE:-local}"
BACKUP_S3_BUCKET="${BACKUP_S3_BUCKET:-}"
BACKUP_S3_REGION="${BACKUP_S3_REGION:-}"

# Directories
BACKUP_DIR="${POSTGRES_BACKUPS_PATH:-$PROJECT_ROOT/backups/postgres}"
LOG_DIR="$BACKUP_DIR/logs"
TEMP_DIR="/tmp/erpgo_backup"

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
    echo "$message" >> "$LOG_DIR/automated-backup.log"
}

error() {
    local message="[ERROR] $1"
    echo -e "${RED}$message${NC}" >&2
    echo "$message" >> "$LOG_DIR/automated-backup.log"
}

warn() {
    local message="[WARN] $1"
    echo -e "${YELLOW}$message${NC}"
    echo "$message" >> "$LOG_DIR/automated-backup.log"
}

success() {
    local message="[SUCCESS] $1"
    echo -e "${GREEN}$message${NC}"
    echo "$message" >> "$LOG_DIR/automated-backup.log"
}

# Ensure directories exist
ensure_directories() {
    mkdir -p "$BACKUP_DIR" "$BACKUP_DIR/encrypted" "$LOG_DIR" "$TEMP_DIR"
}

# Check prerequisites
check_prerequisites() {
    log "Checking backup prerequisites..."

    # Check if database is accessible
    if ! docker exec erpgo-postgres-primary pg_isready -U "${POSTGRES_USER:-erpgo}" -d "${POSTGRES_DB:-erp}" &>/dev/null; then
        error "Database is not accessible"
        return 1
    fi

    # Check disk space
    local available_space=$(df "$BACKUP_DIR" | awk 'NR==2 {print $4}')
    local required_space=1073741824  # 1GB in KB

    if [[ $available_space -lt $required_space ]]; then
        error "Insufficient disk space for backup. Required: 1GB, Available: $((available_space/1024/1024))MB"
        return 1
    fi

    # Check S3 configuration if using S3 storage
    if [[ "$BACKUP_STORAGE_TYPE" == "s3" ]]; then
        if ! command -v aws &> /dev/null; then
            error "AWS CLI not found but S3 storage is configured"
            return 1
        fi

        if [[ -z "$BACKUP_S3_BUCKET" ]]; then
            error "S3 bucket not configured"
            return 1
        fi

        # Test S3 connectivity
        if ! aws s3 ls "s3://$BACKUP_S3_BUCKET" &>/dev/null; then
            error "Cannot access S3 bucket: $BACKUP_S3_BUCKET"
            return 1
        fi
    fi

    success "Prerequisites check passed"
}

# Create backup
create_backup() {
    local backup_type="${1:-$BACKUP_TYPE}"
    local timestamp=$(date '+%Y%m%d_%H%M%S')
    local backup_filename="automated_${backup_type}_backup_${timestamp}.sql"
    local backup_path="$TEMP_DIR/$backup_filename"

    log "Starting automated $backup_type backup..."

    # Create backup using the main backup script
    if "$SCRIPT_DIR/database-backup.sh" backup "$backup_type" > "$LOG_DIR/backup_${timestamp}.log" 2>&1; then
        success "Backup created successfully"

        # Move backup to temp directory for processing
        local created_backup=$(find "$BACKUP_DIR" -name "*${backup_type}_backup_${timestamp}*.sql" -type f | head -1)
        if [[ -n "$created_backup" ]]; then
            mv "$created_backup" "$backup_path"
            echo "$backup_path"
        else
            error "Backup file not found after creation"
            return 1
        fi
    else
        error "Backup creation failed. Check log: $LOG_DIR/backup_${timestamp}.log"
        return 1
    fi
}

# Compress backup
compress_backup() {
    local backup_path="$1"

    log "Compressing backup..."

    local compressed_path="${backup_path}.gz"

    if gzip -c "$backup_path" > "$compressed_path"; then
        # Remove uncompressed backup
        rm "$backup_path"

        local original_size=$(stat -c%s "$backup_path" 2>/dev/null || echo 0)
        local compressed_size=$(stat -c%s "$compressed_path")
        local compression_ratio=$(( (original_size - compressed_size) * 100 / original_size ))

        success "Backup compressed successfully (saved ${compression_ratio}% space)"
        echo "$compressed_path"
    else
        error "Backup compression failed"
        return 1
    fi
}

# Encrypt backup
encrypt_backup() {
    local backup_path="$1"

    if [[ -z "$BACKUP_ENCRYPTION_KEY" ]]; then
        warn "No encryption key provided, skipping encryption"
        echo "$backup_path"
        return 0
    fi

    log "Encrypting backup..."

    local encrypted_path="${backup_path}.enc"

    # Encrypt using AES-256-CBC
    if openssl enc -aes-256-cbc -salt -in "$backup_path" -out "$encrypted_path" -pass pass:"$BACKUP_ENCRYPTION_KEY"; then
        # Remove unencrypted backup
        rm "$backup_path"
        success "Backup encrypted successfully"
        echo "$encrypted_path"
    else
        error "Backup encryption failed"
        return 1
    fi
}

# Verify backup integrity
verify_backup() {
    local backup_path="$1"

    log "Verifying backup integrity..."

    # Determine backup type
    if [[ "$backup_path" == *.enc ]]; then
        # Encrypted backup verification
        local temp_decrypted="$TEMP_DIR/verify_$(basename "$backup_path").sql"

        # Decrypt to temporary file
        if openssl enc -aes-256-cbc -d -in "$backup_path" -out "$temp_decrypted" -pass pass:"$BACKUP_ENCRYPTION_KEY"; then
            # Verify using pg_restore
            if docker exec erpgo-postgres-primary pg_restore --list --format=custom "$temp_decrypted" &>/dev/null; then
                success "Backup integrity verified (encrypted)"
            else
                error "Backup integrity check failed (encrypted)"
                rm -f "$temp_decrypted"
                return 1
            fi
            rm -f "$temp_decrypted"
        else
            error "Failed to decrypt backup for verification"
            return 1
        fi
    else
        # Unencrypted backup verification
        if docker exec erpgo-postgres-primary pg_restore --list --format=custom "$backup_path" &>/dev/null; then
            success "Backup integrity verified (unencrypted)"
        else
            error "Backup integrity check failed (unencrypted)"
            return 1
        fi
    fi
}

# Store backup
store_backup() {
    local backup_path="$1"

    log "Storing backup..."

    local backup_filename=$(basename "$backup_path")
    local final_path="$BACKUP_DIR/$backup_filename"

    # Move to final location
    mv "$backup_path" "$final_path"

    # Store to S3 if configured
    if [[ "$BACKUP_STORAGE_TYPE" == "s3" ]]; then
        local s3_path="s3://$BACKUP_S3_BUCKET/backups/$(date +%Y/%m/%d)/$backup_filename"

        log "Uploading backup to S3: $s3_path"

        if aws s3 cp "$final_path" "$s3_path" --storage-class GLACIER; then
            success "Backup uploaded to S3"
        else
            warn "Failed to upload backup to S3"
        fi
    fi

    success "Backup stored successfully: $final_path"
    echo "$final_path"
}

# Clean up old backups
cleanup_old_backups() {
    log "Cleaning up backups older than $BACKUP_RETENTION_DAYS days..."

    local deleted_count=0

    # Clean local backups
    while IFS= read -r -d '' backup_file; do
        log "Deleting old backup: $(basename "$backup_file")"
        rm -f "$backup_file"
        ((deleted_count++))
    done < <(find "$BACKUP_DIR" -name "automated_*backup_*.sql*" -type f -mtime +$BACKUP_RETENTION_DAYS -print0)

    # Clean S3 backups if configured
    if [[ "$BACKUP_STORAGE_TYPE" == "s3" ]]; then
        local cutoff_date=$(date -d "$BACKUP_RETENTION_DAYS days ago" +%Y-%m-%d)

        while IFS= read -r s3_file; do
            log "Deleting old S3 backup: $s3_file"
            aws s3 rm "$s3_file" || warn "Failed to delete S3 file: $s3_file"
            ((deleted_count++))
        done < <(aws s3 ls "s3://$BACKUP_S3_BUCKET/backups/" --recursive | \
                  awk '$1 < "'$cutoff_date'" {print $4}' | \
                  sed 's/^/s3:\/\/'$BACKUP_S3_BUCKET'\//')
    fi

    success "Cleaned up $deleted_count old backups"
}

# Generate backup report
generate_report() {
    local backup_path="$1"
    local backup_type="$2"
    local start_time="$3"
    local end_time=$(date -u '+%Y-%m-%d %H:%M:%S UTC')

    local backup_size=$(du -h "$backup_path" | cut -f1)
    local duration=$(( $(date -d "$end_time" +%s) - $(date -d "$start_time" +%s) ))

    local report_file="$LOG_DIR/backup_report_$(date '+%Y%m%d_%H%M%S').txt"

    {
        echo "=== Automated Backup Report ==="
        echo "Start Time: $start_time"
        echo "End Time: $end_time"
        echo "Duration: ${duration}s"
        echo "Backup Type: $backup_type"
        echo "Backup File: $(basename "$backup_path")"
        echo "Backup Size: $backup_size"
        echo "Storage Type: $BACKUP_STORAGE_TYPE"
        echo "Environment: ${ENVIRONMENT:-unknown}"
        echo ""
        echo "=== Configuration ==="
        echo "Retention Period: $BACKUP_RETENTION_DAYS days"
        echo "Encryption: $([ -n "$BACKUP_ENCRYPTION_KEY" ] && echo "Enabled" || echo "Disabled")"
        echo "S3 Storage: $BACKUP_STORAGE_TYPE"
        if [[ "$BACKUP_STORAGE_TYPE" == "s3" ]]; then
            echo "S3 Bucket: $BACKUP_S3_BUCKET"
            echo "S3 Region: $BACKUP_S3_REGION"
        fi
        echo ""
        echo "=== Status ==="
        echo "Status: Success"
        echo "Backup verified and stored"
        echo ""
        echo "=== End of Report ==="
    } > "$report_file"

    success "Backup report generated: $report_file"
    echo "$report_file"
}

# Send notification
send_notification() {
    local status="$1"
    local message="$2"
    local report_file="${3:-}"

    log "Sending notification: [$status] $message"

    # Send to Slack if webhook URL is configured
    if [[ -n "${SLACK_WEBHOOK_URL:-}" ]]; then
        local color="good"
        [[ "$status" == "ERROR" ]] && color="danger"
        [[ "$status" == "WARN" ]] && color="warning"

        local payload=$(cat <<EOF
{
    "text": "ERPGo Backup $status",
    "attachments": [
        {
            "color": "$color",
            "fields": [
                {
                    "title": "Environment",
                    "value": "${ENVIRONMENT:-unknown}",
                    "short": true
                },
                {
                    "title": "Backup Type",
                    "value": "$BACKUP_TYPE",
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

    # Send email if configured
    if command -v mail &> /dev/null && [[ -n "${NOTIFICATION_EMAIL:-}" ]]; then
        local email_subject="ERPGo Backup $status - ${ENVIRONMENT:-unknown}"
        local email_body="ERPGo Backup $status

Environment: ${ENVIRONMENT:-unknown}
Backup Type: $BACKUP_TYPE
Time: $(date -u '+%Y-%m-%d %H:%M:%S UTC')
Message: $message"

        if [[ -n "$report_file" ]]; then
            email_body="$email_body

Report attached: $(basename "$report_file")"
        fi

        echo "$email_body" | mail -s "$email_subject" "$NOTIFICATION_EMAIL" 2>/dev/null || warn "Failed to send email notification"
    fi
}

# Setup automatic backup scheduling
setup_cron() {
    log "Setting up automated backup scheduling..."

    local cron_entry="$BACKUP_SCHEDULE $SCRIPT_DIR/automated-backup.sh >> $LOG_DIR/cron.log 2>&1"

    # Add to crontab
    (crontab -l 2>/dev/null | grep -v "$SCRIPT_DIR/automated-backup.sh"; echo "$cron_entry") | crontab -

    success "Automated backup scheduled: $BACKUP_SCHEDULE"
}

# Main backup function
main() {
    local start_time=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
    local backup_type="${1:-$BACKUP_TYPE}"

    log "Starting automated backup process..."
    log "Backup type: $backup_type"

    # Ensure directories exist
    ensure_directories

    # Check prerequisites
    check_prerequisites || {
        send_notification "ERROR" "Backup prerequisites check failed"
        exit 1
    }

    # Create backup
    local backup_path
    backup_path=$(create_backup "$backup_type") || {
        send_notification "ERROR" "Backup creation failed"
        exit 1
    }

    # Compress backup
    backup_path=$(compress_backup "$backup_path") || {
        send_notification "ERROR" "Backup compression failed"
        exit 1
    }

    # Encrypt backup if key is provided
    if [[ -n "$BACKUP_ENCRYPTION_KEY" ]]; then
        backup_path=$(encrypt_backup "$backup_path") || {
            send_notification "ERROR" "Backup encryption failed"
            exit 1
        }
    fi

    # Verify backup integrity
    verify_backup "$backup_path" || {
        send_notification "ERROR" "Backup verification failed"
        exit 1
    }

    # Store backup
    backup_path=$(store_backup "$backup_path") || {
        send_notification "ERROR" "Backup storage failed"
        exit 1
    }

    # Clean up old backups
    cleanup_old_backups

    # Generate report
    local report_file
    report_file=$(generate_report "$backup_path" "$backup_type" "$start_time")

    # Send success notification
    send_notification "SUCCESS" "Automated backup completed successfully" "$report_file"

    success "Automated backup completed successfully!"
    log "Backup file: $backup_path"
}

# Show usage
usage() {
    echo "ERPGo Automated Backup Script"
    echo ""
    echo "Usage:"
    echo "  $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  backup [full|schema|data]    Run backup (default: full)"
    echo "  setup-cron                    Setup automatic backup scheduling"
    echo "  cleanup                       Clean up old backups"
    echo "  help                          Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  BACKUP_TYPE                   Backup type (full, schema, data)"
    echo "  BACKUP_SCHEDULE               Cron schedule for backups"
    echo "  BACKUP_RETENTION_DAYS         Backup retention period"
    echo "  BACKUP_ENCRYPTION_KEY         Encryption key for backups"
    echo "  BACKUP_STORAGE_TYPE           Storage type (local, s3)"
    echo "  BACKUP_S3_BUCKET              S3 bucket name"
    echo "  BACKUP_S3_REGION              S3 bucket region"
    echo ""
    echo "Examples:"
    echo "  $0 backup full                # Run full backup"
    echo "  $0 setup-cron                 # Setup automatic scheduling"
    echo "  $0 cleanup                    # Clean old backups"
}

# Command handling
case "${1:-backup}" in
    "backup")
        main "${2:-full}"
        ;;
    "setup-cron")
        setup_cron
        ;;
    "cleanup")
        ensure_directories
        cleanup_old_backups
        ;;
    "help"|"-h"|"--help")
        usage
        ;;
    *)
        error "Unknown command: $1"
        usage
        exit 1
        ;;
esac