#!/bin/bash

# ERPGo Log Rotation and Cleanup Script
# This script manages log rotation for all services and cleans up old files

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_FILE="${PROJECT_ROOT}/.env.production"
LOGROTATE_CONFIG="${PROJECT_ROOT}/configs/logrotate/logrotate.conf"
LOG_DIR="${LOGS_PATH:-./data/logs}"

# Default rotation settings
DEFAULT_RETENTION_DAYS=30
DEFAULT_MAX_SIZE="100M"
DEFAULT_ROTATE=10

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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
    if [[ -f "$CONFIG_FILE" ]]; then
        log "Loading configuration from: $CONFIG_FILE"
        set -a
        source <(grep -v '^#' "$CONFIG_FILE" | grep -v '^$' | grep '=')
        set +a
    fi
}

# Create logrotate configuration
create_logrotate_config() {
    local config_dir="${PROJECT_ROOT}/configs/logrotate"
    mkdir -p "$config_dir"

    cat > "$LOGROTATE_CONFIG" << EOF
# ERPGo Log Rotation Configuration
# Generated automatically by log-rotation.sh

# Application logs
${LOG_DIR}/erpgo/*.log {
    daily
    missingok
    rotate ${DEFAULT_ROTATE}
    compress
    delaycompress
    notifempty
    create 644 erpgo erpgo
    maxsize ${DEFAULT_MAX_SIZE}
    size ${DEFAULT_MAX_SIZE}
    copytruncate
    sharedscripts
    postrotate
        # Send SIGHUP to application to reopen log files
        docker kill -s HUP erpgo-api 2>/dev/null || true
        docker kill -s HUP erpgo-worker 2>/dev/null || true
    endscript
}

# Nginx logs
${LOG_DIR}/nginx/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 nginx nginx
    maxsize 50M
    size 50M
    copytruncate
    sharedscripts
    postrotate
        # Reopen nginx logs
        docker exec erpgo-nginx nginx -s reopen 2>/dev/null || true
    endscript
}

# Database logs
${LOG_DIR}/postgres/*.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    create 644 postgres postgres
    maxsize 100M
    size 100M
    copytruncate
    sharedscripts
    postrotate
        # Reload PostgreSQL configuration
        docker exec erpgo-postgres-primary pg_reload_conf 2>/dev/null || true
    endscript
}

# Redis logs
${LOG_DIR}/redis/*.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    create 644 redis redis
    maxsize 50M
    size 50M
    copytruncate
}

# Monitoring logs
${LOG_DIR}/prometheus/*.log {
    weekly
    missingok
    rotate 4
    compress
    delaycompress
    notifempty
    create 644 prometheus prometheus
    maxsize 100M
    size 100M
}

${LOG_DIR}/grafana/*.log {
    weekly
    missingok
    rotate 4
    compress
    delaycompress
    notifempty
    create 644 grafana grafana
    maxsize 50M
    size 50M
}

${LOG_DIR}/alertmanager/*.log {
    weekly
    missingok
    rotate 4
    compress
    delaycompress
    notifempty
    create 644 alertmanager alertmanager
    maxsize 20M
    size 20M
}

# Loki logs
${LOG_DIR}/loki/*.log {
    weekly
    missingok
    rotate 4
    compress
    delaycompress
    notifempty
    create 644 loki loki
    maxsize 100M
    size 100M
}

# System and container logs
/var/log/docker/*.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    maxsize 100M
    size 100M
}

/var/log/containers/*.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    maxsize 100M
    size 100M
}

# Backup logs
${LOG_DIR}/backups/*.log {
    monthly
    missingok
    rotate 12
    compress
    delaycompress
    notifempty
    create 644 root root
    maxsize 50M
    size 50M
}

# Deployment logs
${LOG_DIR}/deployment/*.log {
    monthly
    missingok
    rotate 12
    compress
    delaycompress
    notifempty
    create 644 root root
    maxsize 20M
    size 20M
}

# Audit logs (keep longer for compliance)
${LOG_DIR}/audit/*.log {
    daily
    missingok
    rotate 90
    compress
    delaycompress
    notifempty
    create 644 root root
    maxsize 50M
    size 50M
    copytruncate
}

# Security logs
${LOG_DIR}/security/*.log {
    daily
    missingok
    rotate 90
    compress
    delaycompress
    notifempty
    create 644 root root
    maxsize 20M
    size 20M
    copytruncate
}
EOF

    success "Logrotate configuration created: $LOGROTATE_CONFIG"
}

# Setup logrotate cron job
setup_logrotate_cron() {
    log "Setting up logrotate cron job..."

    local cron_file="/etc/cron.d/erpgo-logrotate"
    local cron_entry="0 2 * * * root ${SCRIPT_DIR}/log-rotation.sh rotate >/dev/null 2>&1"

    # Create cron entry
    if [[ $EUID -eq 0 ]]; then
        echo "$cron_entry" > "$cron_file"
        chmod 644 "$cron_file"
        success "Logrotate cron job created: $cron_file"
    else
        warn "Running as non-root user. Please manually add to crontab:"
        warn "0 2 * * * ${SCRIPT_DIR}/log-rotation.sh rotate >/dev/null 2>&1"
    fi
}

# Rotate logs using logrotate
rotate_logs() {
    log "Starting log rotation..."

    # Check if logrotate is available
    if ! command -v logrotate &> /dev/null; then
        warn "logrotate not found, using manual rotation"
        manual_rotation
        return $?
    fi

    # Create config if it doesn't exist
    if [[ ! -f "$LOGROTATE_CONFIG" ]]; then
        create_logrotate_config
    fi

    # Test configuration
    if logrotate -d "$LOGROTATE_CONFIG" &>/dev/null; then
        log "Logrotate configuration is valid"
    else
        error "Invalid logrotate configuration"
        return 1
    fi

    # Run logrotate
    if logrotate -f "$LOGROTATE_CONFIG"; then
        success "Log rotation completed successfully"
    else
        error "Log rotation failed"
        return 1
    fi

    # Show disk space saved
    show_disk_usage
}

# Manual log rotation (fallback)
manual_rotation() {
    log "Performing manual log rotation..."

    local retention_days="${BACKUP_RETENTION_DAYS:-$DEFAULT_RETENTION_DAYS}"

    # Rotate application logs
    find "$LOG_DIR" -name "*.log" -type f -mtime +1 -exec gzip {} \;
    find "$LOG_DIR" -name "*.log.gz" -type f -mtime +$retention_days -delete

    # Rotate nginx logs
    find "${LOG_DIR}/nginx" -name "*.log" -type f -mtime +1 -exec gzip {} \;
    find "${LOG_DIR}/nginx" -name "*.log.gz" -type f -mtime +$retention_days -delete

    # Rotate database logs
    find "${LOG_DIR}/postgres" -name "*.log" -type f -mtime +7 -exec gzip {} \;
    find "${LOG_DIR}/postgres" -name "*.log.gz" -type f -mtime +30 -delete

    success "Manual log rotation completed"
}

# Clean up old files
cleanup_old_files() {
    log "Cleaning up old files..."

    local retention_days="${BACKUP_RETENTION_DAYS:-$DEFAULT_RETENTION_DAYS}"

    # Clean up old temporary files
    find /tmp -name "erpgo-*" -type f -mtime +1 -delete 2>/dev/null || true

    # Clean up old Docker container logs
    docker system prune -f --volumes 2>/dev/null || true

    # Clean up old backup files (beyond retention period)
    find "${PROJECT_ROOT}/backups" -name "*" -type f -mtime +$((retention_days * 2)) -delete 2>/dev/null || true

    # Clean up old build artifacts
    find "$PROJECT_ROOT" -name "*.tmp" -type f -mtime +7 -delete 2>/dev/null || true
    find "$PROJECT_ROOT" -name "*.bak" -type f -mtime +30 -delete 2>/dev/null || true

    success "Old files cleanup completed"
}

# Compress large log files
compress_large_logs() {
    log "Compressing large log files..."

    # Find and compress log files larger than 10MB
    find "$LOG_DIR" -name "*.log" -type f -size +10M -not -name "*.gz" -exec gzip {} \; 2>/dev/null || true

    # Find and compress files older than 1 day
    find "$LOG_DIR" -name "*.log" -type f -mtime +1 -not -name "*.gz" -exec gzip {} \; 2>/dev/null || true

    success "Large log files compression completed"
}

# Analyze log sizes
analyze_logs() {
    log "Analyzing log sizes..."

    echo ""
    echo "=== Log Directory Sizes ==="
    du -sh "${LOG_DIR}"/* 2>/dev/null | sort -hr | head -10

    echo ""
    echo "=== Largest Log Files ==="
    find "$LOG_DIR" -name "*.log*" -type f -exec du -h {} \; 2>/dev/null | sort -hr | head -10

    echo ""
    echo "=== Log File Counts ==="
    echo "Total log files: $(find "$LOG_DIR" -name "*.log*" -type f | wc -l)"
    echo "Compressed files: $(find "$LOG_DIR" -name "*.gz" -type f | wc -l)"
    echo "Uncompressed files: $(find "$LOG_DIR" -name "*.log" -type f | wc -l)"
}

# Show disk usage
show_disk_usage() {
    log "Checking disk usage..."

    local usage_output=$(df -h "$PROJECT_ROOT" | awk 'NR==2 {print $5}' | sed 's/%//')
    local available=$(df -h "$PROJECT_ROOT" | awk 'NR==2 {print $4}')

    echo "Disk usage: ${usage_output}%"
    echo "Available space: $available"

    if [[ $usage_output -gt 80 ]]; then
        warn "High disk usage detected (${usage_output}%)"
        warn "Consider increasing cleanup frequency or adding more storage"
    elif [[ $usage_output -gt 90 ]]; then
        error "Critical disk usage (${usage_output}%)"
        error "Immediate cleanup required"
        return 1
    else
        success "Disk usage is acceptable (${usage_output}%)"
    fi
}

# Create log directories
create_log_directories() {
    log "Creating log directories..."

    local directories=(
        "${LOG_DIR}/erpgo"
        "${LOG_DIR}/nginx"
        "${LOG_DIR}/postgres"
        "${LOG_DIR}/redis"
        "${LOG_DIR}/prometheus"
        "${LOG_DIR}/grafana"
        "${LOG_DIR}/alertmanager"
        "${LOG_DIR}/loki"
        "${LOG_DIR}/backups"
        "${LOG_DIR}/deployment"
        "${LOG_DIR}/audit"
        "${LOG_DIR}/security"
    )

    for dir in "${directories[@]}"; do
        mkdir -p "$dir"
        chmod 755 "$dir"
    done

    success "Log directories created"
}

# Generate log rotation report
generate_report() {
    local report_file="${LOG_DIR}/rotation_report.log"

    {
        echo "=== ERPGo Log Rotation Report ==="
        echo "Timestamp: $(date '+%Y-%m-%d %H:%M:%S')"
        echo ""
        echo "=== Disk Usage ==="
        df -h "$PROJECT_ROOT"
        echo ""
        echo "=== Log Directory Sizes ==="
        du -sh "${LOG_DIR}"/* 2>/dev/null || echo "No log directories found"
        echo ""
        echo "=== Recent Rotations ==="
        find "$LOG_DIR" -name "*.gz" -type f -mtime -7 | wc -l
        echo "files rotated in the last 7 days"
        echo ""
        echo "=== Log File Statistics ==="
        echo "Total files: $(find "$LOG_DIR" -name "*.log*" -type f | wc -l)"
        echo "Compressed: $(find "$LOG_DIR" -name "*.gz" -type f | wc -l)"
        echo "Uncompressed: $(find "$LOG_DIR" -name "*.log" -type f | wc -l)"
        echo ""
        echo "=== End of Report ==="
        echo ""
    } > "$report_file"

    success "Log rotation report generated: $report_file"
}

# Main execution function
main() {
    local action="${1:-rotate}"

    log "Starting ERPGo log rotation script..."
    log "Action: $action"

    # Load configuration
    load_config

    # Create log directories
    create_log_directories

    case "$action" in
        "rotate")
            rotate_logs
            ;;
        "compress")
            compress_large_logs
            ;;
        "cleanup")
            cleanup_old_files
            ;;
        "analyze")
            analyze_logs
            ;;
        "setup")
            create_logrotate_config
            setup_logrotate_cron
            ;;
        "report")
            generate_report
            ;;
        "all")
            create_logrotate_config
            rotate_logs
            compress_large_logs
            cleanup_old_files
            generate_report
            ;;
        *)
            error "Unknown action: $action"
            echo "Usage: $0 {rotate|compress|cleanup|analyze|setup|report|all}"
            exit 1
            ;;
    esac

    success "Log rotation script completed"
}

# Show usage
usage() {
    echo "ERPGo Log Rotation and Cleanup Script"
    echo ""
    echo "Usage:"
    echo "  $0 [ACTION]"
    echo ""
    echo "Actions:"
    echo "  rotate      Rotate logs using logrotate (default)"
    echo "  compress    Compress large log files"
    echo "  cleanup     Clean up old files"
    echo "  analyze     Analyze log sizes and disk usage"
    echo "  setup       Setup logrotate configuration and cron job"
    echo "  report      Generate rotation report"
    echo "  all         Run all maintenance tasks"
    echo ""
    echo "Examples:"
    echo "  $0 rotate           # Rotate logs"
    echo "  $0 compress         # Compress large logs"
    echo "  $0 all              # Run all tasks"
    echo ""
    echo "Environment variables:"
    echo "  LOGS_PATH          Log directory path (default: ./data/logs)"
    echo "  BACKUP_RETENTION_DAYS  Backup retention period (default: 30)"
    echo ""
}

# Command handling
case "${1:-help}" in
    "help"|"-h"|"--help")
        usage
        ;;
    *)
        main "$1"
        ;;
esac