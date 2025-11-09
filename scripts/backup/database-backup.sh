#!/bin/bash

# ERPGo Database Backup Script
# This script creates automated backups of the PostgreSQL database with encryption and retention policies

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_FILE="${PROJECT_ROOT}/.env.production"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
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

# Load environment variables
load_config() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        error "Configuration file not found: $CONFIG_FILE"
        exit 1
    fi

    # Load environment variables, ignoring comments and empty lines
    set -a
    source <(grep -v '^#' "$CONFIG_FILE" | grep -v '^$' | grep '=')
    set +a

    # Validate required variables
    local required_vars=("POSTGRES_DB" "POSTGRES_USER" "POSTGRES_PASSWORD" "POSTGRES_PRIMARY_HOST")
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            error "Required environment variable not set: $var"
            exit 1
        fi
    done
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."

    # Check if docker is available
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed or not in PATH"
        exit 1
    fi

    # Check if docker-compose is available
    if ! command -v docker-compose &> /dev/null; then
        error "Docker Compose is not installed or not in PATH"
        exit 1
    fi

    # Check if openssl is available for encryption
    if ! command -v openssl &> /dev/null; then
        error "OpenSSL is not installed or not in PATH"
        exit 1
    fi

    # Check if required directories exist
    local backup_dir="${POSTGRES_BACKUPS_PATH:-./backups/postgres}"
    mkdir -p "$backup_dir"
    mkdir -p "${backup_dir}/encrypted"
    mkdir -p "${backup_dir}/logs"

    success "Prerequisites check passed"
}

# Get database connection details
get_db_connection() {
    local db_host="${POSTGRES_PRIMARY_HOST:-postgres-primary}"
    local db_port="${POSTGRES_PORT:-5432}"
    local db_name="${POSTGRES_DB}"
    local db_user="${POSTGRES_USER}"
    local db_password="${POSTGRES_PASSWORD}"

    echo "postgresql://${db_user}:${db_password}@${db_host}:${db_port}/${db_name}?sslmode=require"
}

# Create backup
create_backup() {
    local backup_type="${1:-full}"
    local timestamp=$(date '+%Y%m%d_%H%M%S')
    local backup_filename="${backup_type}_backup_${timestamp}.sql"
    local backup_path="${POSTGRES_BACKUPS_PATH:-./backups/postgres}/${backup_filename}"
    local log_file="${POSTGRES_BACKUPS_PATH:-./backups/postgres}/logs/backup_${timestamp}.log"

    log "Starting ${backup_type} backup..."

    # Create backup using docker exec
    local container_name="erpgo-postgres-primary"
    local db_connection=$(get_db_connection)

    case "$backup_type" in
        "full")
            log "Creating full database backup..."
            docker exec "$container_name" pg_dump \
                --no-owner \
                --no-privileges \
                --verbose \
                --format=custom \
                --compress=9 \
                --file="/tmp/${backup_filename}" \
                "$POSTGRES_DB" > "$log_file" 2>&1
            ;;
        "schema")
            log "Creating schema-only backup..."
            docker exec "$container_name" pg_dump \
                --no-owner \
                --no-privileges \
                --schema-only \
                --verbose \
                --format=custom \
                --file="/tmp/${backup_filename}" \
                "$POSTGRES_DB" > "$log_file" 2>&1
            ;;
        "data")
            log "Creating data-only backup..."
            docker exec "$container_name" pg_dump \
                --no-owner \
                --no-privileges \
                --data-only \
                --verbose \
                --format=custom \
                --compress=9 \
                --file="/tmp/${backup_filename}" \
                "$POSTGRES_DB" > "$log_file" 2>&1
            ;;
        *)
            error "Invalid backup type: $backup_type. Use 'full', 'schema', or 'data'"
            exit 1
            ;;
    esac

    # Check if backup was created successfully
    local exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        error "Backup creation failed with exit code: $exit_code"
        error "Check log file: $log_file"
        exit 1
    fi

    # Copy backup from container to host
    docker cp "${container_name}:/tmp/${backup_filename}" "$backup_path"

    # Clean up temporary backup file in container
    docker exec "$container_name" rm -f "/tmp/${backup_filename}"

    # Verify backup file
    if [[ ! -f "$backup_path" ]]; then
        error "Backup file not found: $backup_path"
        exit 1
    fi

    local backup_size=$(du -h "$backup_path" | cut -f1)
    success "Backup created successfully: $backup_filename (Size: $backup_size)"

    echo "$backup_path"
}

# Encrypt backup
encrypt_backup() {
    local backup_path="$1"
    local encryption_key="${BACKUP_ENCRYPTION_KEY:-}"

    if [[ -z "$encryption_key" ]]; then
        warn "No encryption key provided, skipping encryption"
        return 0
    fi

    if [[ ! -f "$backup_path" ]]; then
        error "Backup file not found: $backup_path"
        exit 1
    fi

    log "Encrypting backup..."

    local encrypted_path="${POSTGRES_BACKUPS_PATH:-./backups/postgres}/encrypted/$(basename "$backup_path").enc"

    # Encrypt using AES-256-CBC
    openssl enc -aes-256-cbc -salt -in "$backup_path" -out "$encrypted_path" -pass pass:"$encryption_key"

    # Verify encryption
    if [[ ! -f "$encrypted_path" ]]; then
        error "Encryption failed for: $backup_path"
        exit 1
    fi

    # Remove unencrypted backup
    rm "$backup_path"

    local encrypted_size=$(du -h "$encrypted_path" | cut -f1)
    success "Backup encrypted successfully: $(basename "$encrypted_path") (Size: $encrypted_size)"

    echo "$encrypted_path"
}

# Verify backup integrity
verify_backup() {
    local backup_path="$1"
    local encryption_key="${BACKUP_ENCRYPTION_KEY:-}"

    log "Verifying backup integrity..."

    # Determine if backup is encrypted
    if [[ "$backup_path" == *.enc ]]; then
        # Create temporary decrypted file
        local temp_decrypted="/tmp/verify_$(basename "$backup_path").sql"

        if [[ -z "$encryption_key" ]]; then
            error "Cannot verify encrypted backup without encryption key"
            exit 1
        fi

        # Decrypt to temporary file
        openssl enc -aes-256-cbc -d -in "$backup_path" -out "$temp_decrypted" -pass pass:"$encryption_key"

        # Verify using pg_restore
        local container_name="erpgo-postgres-primary"
        docker cp "$temp_decrypted" "${container_name}:/tmp/verify_backup.sql"

        # Test restore (just validation, don't actually restore)
        local validation_result=$(docker exec "$container_name" pg_restore \
            --list \
            --format=custom \
            "/tmp/verify_backup.sql" 2>&1)

        local exit_code=$?

        # Clean up
        docker exec "$container_name" rm -f "/tmp/verify_backup.sql"
        rm -f "$temp_decrypted"

        if [[ $exit_code -ne 0 ]]; then
            error "Backup verification failed: $validation_result"
            exit 1
        fi

    else
        # Unencrypted backup - verify directly
        local container_name="erpgo-postgres-primary"
        docker cp "$backup_path" "${container_name}:/tmp/verify_backup.sql"

        local validation_result=$(docker exec "$container_name" pg_restore \
            --list \
            --format=custom \
            "/tmp/verify_backup.sql" 2>&1)

        local exit_code=$?

        # Clean up
        docker exec "$container_name" rm -f "/tmp/verify_backup.sql"

        if [[ $exit_code -ne 0 ]]; then
            error "Backup verification failed: $validation_result"
            exit 1
        fi
    fi

    success "Backup integrity verified"
}

# Clean up old backups
cleanup_old_backups() {
    local retention_days="${BACKUP_RETENTION_DAYS:-30}"
    local backup_dir="${POSTGRES_BACKUPS_PATH:-./backups/postgres}"

    log "Cleaning up backups older than $retention_days days..."

    # Clean up old unencrypted backups
    find "$backup_dir" -name "*.sql" -type f -mtime +$retention_days -delete

    # Clean up old encrypted backups
    find "$backup_dir/encrypted" -name "*.enc" -type f -mtime +$retention_days -delete

    # Clean up old log files
    find "$backup_dir/logs" -name "*.log" -type f -mtime +$retention_days -delete

    success "Old backups cleaned up"
}

# Generate backup report
generate_backup_report() {
    local backup_path="$1"
    local backup_type="${2:-full}"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local backup_file=$(basename "$backup_path")
    local backup_size=$(du -h "$backup_path" | cut -f1)

    local report_file="${POSTGRES_BACKUPS_PATH:-./backups/postgres}/logs/backup_report.log"

    {
        echo "=== Backup Report ==="
        echo "Timestamp: $timestamp"
        echo "Type: $backup_type"
        echo "File: $backup_file"
        echo "Size: $backup_size"
        echo "Database: ${POSTGRES_DB}"
        echo "Host: ${POSTGRES_PRIMARY_HOST}"
        echo "Status: Success"
        echo "====================="
        echo ""
    } >> "$report_file"

    success "Backup report generated: $report_file"
}

# Send notification
send_notification() {
    local status="$1"
    local message="$2"

    # Placeholder for notification logic
    # You can integrate with email, Slack, PagerDuty, etc.
    log "Notification: [$status] $message"

    # Example: Send email using curl to an API
    # curl -X POST "https://api.yourdomain.com/notifications" \
    #     -H "Content-Type: application/json" \
    #     -d "{\"level\":\"$status\",\"message\":\"$message\",\"service\":\"erpgo-backup\"}"
}

# Main backup function
main() {
    local backup_type="${1:-full}"

    log "Starting ERPGo database backup process..."
    log "Backup type: $backup_type"

    # Load configuration
    load_config

    # Check prerequisites
    check_prerequisites

    # Create backup
    local backup_path
    backup_path=$(create_backup "$backup_type")

    # Encrypt backup if key is provided
    if [[ -n "${BACKUP_ENCRYPTION_KEY:-}" ]]; then
        backup_path=$(encrypt_backup "$backup_path")
    fi

    # Verify backup integrity
    verify_backup "$backup_path"

    # Clean up old backups
    cleanup_old_backups

    # Generate report
    generate_backup_report "$backup_path" "$backup_type"

    # Send success notification
    send_notification "INFO" "Database backup completed successfully: $(basename "$backup_path")"

    success "Backup process completed successfully!"
    log "Backup file: $backup_path"
}

# Recovery function
restore_backup() {
    local backup_file="$1"
    local target_db="${2:-${POSTGRES_DB}}"

    if [[ -z "$backup_file" ]]; then
        error "Backup file path is required"
        exit 1
    fi

    if [[ ! -f "$backup_file" ]]; then
        error "Backup file not found: $backup_file"
        exit 1
    fi

    log "Starting database restore from: $backup_file"
    log "Target database: $target_db"

    # Load configuration
    load_config

    # Check if backup is encrypted
    if [[ "$backup_file" == *.enc ]]; then
        if [[ -z "${BACKUP_ENCRYPTION_KEY:-}" ]]; then
            error "Cannot restore encrypted backup without encryption key"
            exit 1
        fi

        log "Decrypting backup file..."
        local temp_decrypted="/tmp/restore_$(basename "$backup_file").sql"

        # Decrypt backup
        openssl enc -aes-256-cbc -d -in "$backup_file" -out "$temp_decrypted" -pass pass:"${BACKUP_ENCRYPTION_KEY}"

        # Copy decrypted file to container
        local container_name="erpgo-postgres-primary"
        docker cp "$temp_decrypted" "${container_name}:/tmp/restore_backup.sql"

        # Restore from decrypted file
        log "Restoring database..."
        docker exec "$container_name" pg_restore \
            --verbose \
            --clean \
            --if-exists \
            --no-owner \
            --no-privileges \
            --dbname="$target_db" \
            "/tmp/restore_backup.sql"

        # Clean up
        docker exec "$container_name" rm -f "/tmp/restore_backup.sql"
        rm -f "$temp_decrypted"

    else
        # Unencrypted backup
        local container_name="erpgo-postgres-primary"
        docker cp "$backup_file" "${container_name}:/tmp/restore_backup.sql"

        # Restore database
        log "Restoring database..."
        docker exec "$container_name" pg_restore \
            --verbose \
            --clean \
            --if-exists \
            --no-owner \
            --no-privileges \
            --dbname="$target_db" \
            "/tmp/restore_backup.sql"

        # Clean up
        docker exec "$container_name" rm -f "/tmp/restore_backup.sql"
    fi

    success "Database restore completed successfully!"
    send_notification "INFO" "Database restore completed successfully from: $(basename "$backup_file")"
}

# List available backups
list_backups() {
    local backup_dir="${POSTGRES_BACKUPS_PATH:-./backups/postgres}"

    log "Available backups:"
    echo ""

    # List unencrypted backups
    if [[ -d "$backup_dir" ]] && [[ "$(ls -A "$backup_dir" 2>/dev/null)" ]]; then
        echo "=== Unencrypted Backups ==="
        ls -lah "$backup_dir"/*.sql 2>/dev/null | while read -r line; do
            echo "$line"
        done
        echo ""
    fi

    # List encrypted backups
    if [[ -d "$backup_dir/encrypted" ]] && [[ "$(ls -A "$backup_dir/encrypted" 2>/dev/null)" ]]; then
        echo "=== Encrypted Backups ==="
        ls -lah "$backup_dir/encrypted"/*.enc 2>/dev/null | while read -r line; do
            echo "$line"
        done
        echo ""
    fi
}

# Show usage
usage() {
    echo "ERPGo Database Backup Script"
    echo ""
    echo "Usage:"
    echo "  $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  backup [full|schema|data]    Create database backup (default: full)"
    echo "  restore <backup_file> [db]   Restore database from backup"
    echo "  list                         List available backups"
    echo "  cleanup                      Clean up old backups"
    echo "  help                         Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 backup full                # Create full backup"
    echo "  $0 backup schema              # Create schema-only backup"
    echo "  $0 restore backup_20231201.sql # Restore from backup"
    echo "  $0 list                       # List all backups"
    echo ""
    echo "Environment variables:"
    echo "  POSTGRES_DB                   Database name"
    echo "  POSTGRES_USER                 Database user"
    echo "  POSTGRES_PASSWORD             Database password"
    echo "  POSTGRES_PRIMARY_HOST         Database host"
    echo "  BACKUP_ENCRYPTION_KEY         Encryption key (optional)"
    echo "  BACKUP_RETENTION_DAYS         Backup retention period (default: 30)"
    echo ""
}

# Command handling
case "${1:-backup}" in
    "backup")
        main "${2:-full}"
        ;;
    "restore")
        restore_backup "$2" "$3"
        ;;
    "list")
        load_config
        list_backups
        ;;
    "cleanup")
        load_config
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