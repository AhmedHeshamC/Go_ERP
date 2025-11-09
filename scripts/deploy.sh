#!/bin/bash

# ERPGo Production Deployment Script
# Features: Zero-downtime deployment, health checks, rollback capabilities, blue-green deployment

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_FILE="${PROJECT_ROOT}/.env.production"
DEPLOYMENT_LOG="${PROJECT_ROOT}/logs/deployment.log"
HEALTH_CHECK_TIMEOUT=300
ROLLBACK_TIMEOUT=600

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Ensure log directory exists
mkdir -p "$(dirname "$DEPLOYMENT_LOG")"

# Logging functions
log() {
    local message="[$(date '+%Y-%m-%d %H:%M:%S')] $1"
    echo -e "${BLUE}$message${NC}"
    echo "$message" >> "$DEPLOYMENT_LOG"
}

error() {
    local message="[ERROR] $1"
    echo -e "${RED}$message${NC}" >&2
    echo "$message" >> "$DEPLOYMENT_LOG"
}

warn() {
    local message="[WARN] $1"
    echo -e "${YELLOW}$message${NC}"
    echo "$message" >> "$DEPLOYMENT_LOG"
}

success() {
    local message="[SUCCESS] $1"
    echo -e "${GREEN}$message${NC}"
    echo "$message" >> "$DEPLOYMENT_LOG"
}

# Send notification
send_notification() {
    local level="$1"
    local message="$2"

    log "Notification: [$level] $message"

    # Example: Send to Slack webhook
    if [[ -n "${SLACK_WEBHOOK_URL:-}" ]]; then
        local color="good"
        case "$level" in
            "ERROR") color="danger" ;;
            "WARN") color="warning" ;;
        esac

        curl -X POST "$SLACK_WEBHOOK_URL" \
            -H 'Content-type: application/json' \
            --data "{\"text\":\"$message\",\"color\":\"$color\"}" \
            2>/dev/null || true
    fi

    # Example: Send email notification
    if command -v mail &> /dev/null && [[ -n "${NOTIFICATION_EMAIL:-}" ]]; then
        echo "$message" | mail -s "ERPGo Deployment [$level]" "$NOTIFICATION_EMAIL" 2>/dev/null || true
    fi
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

    # Validate required variables
    local required_vars=("ENVIRONMENT" "DOMAIN_NAME")
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            error "Required environment variable not set: $var"
            exit 1
        fi
    done
}

# Check prerequisites
check_prerequisites() {
    log "Checking deployment prerequisites..."

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

    # Check if we can connect to Docker
    if ! docker info &> /dev/null; then
        error "Cannot connect to Docker daemon"
        exit 1
    fi

    # Check if production docker-compose file exists
    local compose_file="${PROJECT_ROOT}/docker-compose.prod.yml"
    if [[ ! -f "$compose_file" ]]; then
        error "Production Docker Compose file not found: $compose_file"
        exit 1
    fi

    # Check if SSL certificates exist
    local ssl_cert="${PROJECT_ROOT}/configs/nginx/ssl/cert.pem"
    local ssl_key="${PROJECT_ROOT}/configs/nginx/ssl/key.pem"

    if [[ ! -f "$ssl_cert" ]] || [[ ! -f "$ssl_key" ]]; then
        warn "SSL certificates not found. Deployment will proceed but HTTPS may not work properly."
        warn "Expected files: $ssl_cert, $ssl_key"
    fi

    # Check disk space
    local available_space=$(df "$PROJECT_ROOT" | awk 'NR==2 {print $4}')
    local required_space=5242880  # 5GB in KB

    if [[ $available_space -lt $required_space ]]; then
        error "Insufficient disk space. Required: 5GB, Available: $((available_space/1024/1024))GB"
        exit 1
    fi

    success "Prerequisites check passed"
}

# Create deployment backup
create_deployment_backup() {
    log "Creating deployment backup..."

    local backup_dir="${PROJECT_ROOT}/backups/deployment"
    local timestamp=$(date '+%Y%m%d_%H%M%S')
    local deployment_backup="${backup_dir}/deployment_${timestamp}"

    mkdir -p "$backup_dir"

    # Backup current running containers state
    docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" ps > "${deployment_backup}_containers.txt"

    # Backup current images
    docker images --format "table {{.Repository}}:{{.Tag}}\t{{.ID}}\t{{.CreatedAt}}\t{{.Size}}" > "${deployment_backup}_images.txt"

    # Backup configuration
    cp "$CONFIG_FILE" "${deployment_backup}_env.txt"

    # Backup SSL certificates
    if [[ -d "${PROJECT_ROOT}/configs/nginx/ssl" ]]; then
        cp -r "${PROJECT_ROOT}/configs/nginx/ssl" "${deployment_backup}_ssl"
    fi

    log "Deployment backup created: $deployment_backup"
    echo "$deployment_backup"
}

# Build application image
build_image() {
    local version_tag="${1:-latest}"

    log "Building ERPGo application image: $version_tag"

    cd "$PROJECT_ROOT"

    # Build with no cache to ensure fresh build
    docker build \
        --no-cache \
        --tag "erpgo:$version_tag" \
        --tag "erpgo:latest" \
        --build-arg BUILD_DATE="$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
        --build-arg VCS_REF="$(git rev-parse HEAD 2>/dev/null || echo 'unknown')" \
        --build-arg VERSION="$version_tag" \
        .

    # Verify image was built
    if ! docker images | grep -q "erpgo.*$version_tag"; then
        error "Failed to build application image"
        exit 1
    fi

    success "Application image built successfully: erpgo:$version_tag"
}

# Run health checks
run_health_checks() {
    local service="$1"
    local timeout="${2:-$HEALTH_CHECK_TIMEOUT}"

    log "Running health checks for $service (timeout: ${timeout}s)..."

    local elapsed=0
    local interval=10

    while [[ $elapsed -lt $timeout ]]; do
        # Check if service is healthy
        local health_status
        case "$service" in
            "api")
                # Check API health endpoint
                health_status=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/health" 2>/dev/null || echo "000")
                if [[ "$health_status" == "200" ]]; then
                    success "API health check passed"
                    return 0
                fi
                ;;
            "database")
                # Check database connectivity
                if docker exec erpgo-postgres-primary pg_isready -U "${POSTGRES_USER:-erpgo}" -d "${POSTGRES_DB:-erp}" &>/dev/null; then
                    success "Database health check passed"
                    return 0
                fi
                ;;
            "redis")
                # Check Redis connectivity
                if docker exec erpgo-redis-master redis-cli ping &>/dev/null; then
                    success "Redis health check passed"
                    return 0
                fi
                ;;
            "nginx")
                # Check Nginx health
                health_status=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost/health" 2>/dev/null || echo "000")
                if [[ "$health_status" == "200" ]]; then
                    success "Nginx health check passed"
                    return 0
                fi
                ;;
        esac

        elapsed=$((elapsed + interval))
        if [[ $elapsed -lt $timeout ]]; then
            sleep $interval
            echo -n "."
        fi
    done

    echo ""
    error "Health check failed for $service after ${timeout}s"
    return 1
}

# Deploy with rolling update
deploy_rolling_update() {
    local version_tag="${1:-latest}"

    log "Starting rolling update deployment..."

    cd "$PROJECT_ROOT"

    # Update docker-compose.yml to use new image
    sed -i.bak "s|image: erpgo:.*|image: erpgo:$version_tag|g" docker-compose.prod.yml

    # Deploy with rolling update
    docker-compose -f docker-compose.prod.yml up -d --no-deps api worker

    # Wait for rolling update to complete
    log "Waiting for rolling update to complete..."
    sleep 30

    # Run health checks
    if ! run_health_checks "api"; then
        error "API health check failed after rolling update"
        return 1
    fi

    success "Rolling update completed successfully"
}

# Deploy with blue-green strategy
deploy_blue_green() {
    local version_tag="${1:-latest}"
    local green_port=8081

    log "Starting blue-green deployment..."

    cd "$PROJECT_ROOT"

    # Save current state (blue environment)
    local blue_compose="${PROJECT_ROOT}/docker-compose.blue.yml"
    local green_compose="${PROJECT_ROOT}/docker-compose.green.yml"

    # Create blue environment (current)
    cp docker-compose.prod.yml "$blue_compose"

    # Create green environment (new version)
    cp docker-compose.prod.yml "$green_compose"

    # Update green environment to use new image and different ports
    sed -i.bak "s|image: erpgo:.*|image: erpgo:$version_tag|g" "$green_compose"
    sed -i.bak "s|\"8080:8080\"|\"${green_port}:8080\"|g" "$green_compose"

    # Start green environment
    log "Starting green environment..."
    docker-compose -f "$green_compose" up -d

    # Wait for green environment to be ready
    sleep 60

    # Run health checks on green environment
    log "Testing green environment..."
    if run_health_checks "api" "$((HEALTH_CHECK_TIMEOUT / 2))"; then
        # Update nginx to route traffic to green environment
        log "Switching traffic to green environment..."

        # Update nginx configuration to point to green port
        sed -i.bak "s|server api:8080|server 127.0.0.1:${green_port}|g" configs/nginx/nginx.conf

        # Reload nginx
        docker exec erpgo-nginx nginx -s reload

        # Wait for traffic switch
        sleep 30

        # Final health check
        if run_health_checks "nginx"; then
            success "Blue-green deployment completed successfully"

            # Update production compose file
            cp "$green_compose" docker-compose.prod.yml

            # Clean up blue environment
            docker-compose -f "$blue_compose" down

            # Restore nginx configuration
            sed -i.bak 's|server 127.0.0.1:8081|server api:8080|g' configs/nginx/nginx.conf
            docker exec erpgo-nginx nginx -s reload

            # Clean up temporary files
            rm -f "$blue_compose" "$green_compose"
            rm -f configs/nginx/nginx.conf.bak

            return 0
        else
            error "Final health check failed after switching traffic"
        fi
    else
        error "Green environment health checks failed"
    fi

    # Rollback to blue environment
    warn "Rolling back to blue environment..."
    docker-compose -f "$green_compose" down
    rm -f "$blue_compose" "$green_compose"
    rm -f configs/nginx/nginx.conf.bak

    return 1
}

# Rollback deployment
rollback_deployment() {
    local backup_path="$1"

    log "Starting deployment rollback from: $backup_path"

    if [[ ! -d "$backup_path" ]]; then
        error "Backup directory not found: $backup_path"
        exit 1
    fi

    cd "$PROJECT_ROOT"

    # Stop current services
    log "Stopping current services..."
    docker-compose -f docker-compose.prod.yml down

    # Restore configuration
    if [[ -f "${backup_path}_env.txt" ]]; then
        cp "${backup_path}_env.txt" "$CONFIG_FILE"
        log "Configuration restored"
    fi

    # Restore SSL certificates
    if [[ -d "${backup_path}_ssl" ]]; then
        cp -r "${backup_path}_ssl"/* configs/nginx/ssl/
        log "SSL certificates restored"
    fi

    # Start services with restored configuration
    log "Starting services with restored configuration..."
    docker-compose -f docker-compose.prod.yml up -d

    # Wait for services to start
    sleep 60

    # Run health checks
    local services=("database" "redis" "api" "nginx")
    for service in "${services[@]}"; do
        if ! run_health_checks "$service" "$ROLLBACK_TIMEOUT"; then
            error "Rollback failed: $service health check failed"
            send_notification "ERROR" "Deployment rollback failed: $service health check failed"
            exit 1
        fi
    done

    success "Deployment rollback completed successfully"
    send_notification "INFO" "Deployment rollback completed successfully"
}

# Verify deployment
verify_deployment() {
    log "Verifying deployment..."

    local checks_passed=0
    local total_checks=4

    # Check API health
    if run_health_checks "api"; then
        ((checks_passed++))
    fi

    # Check database connectivity
    if run_health_checks "database"; then
        ((checks_passed++))
    fi

    # Check Redis connectivity
    if run_health_checks "redis"; then
        ((checks_passed++))
    fi

    # Check Nginx health
    if run_health_checks "nginx"; then
        ((checks_passed++))
    fi

    if [[ $checks_passed -eq $total_checks ]]; then
        success "Deployment verification passed ($checks_passed/$total_checks checks)"
        return 0
    else
        error "Deployment verification failed ($checks_passed/$total_checks checks passed)"
        return 1
    fi
}

# Main deployment function
deploy() {
    local deployment_type="${1:-rolling}"
    local version_tag="${2:-latest}"

    log "Starting ERPGo deployment..."
    log "Deployment type: $deployment_type"
    log "Version tag: $version_tag"

    send_notification "INFO" "Starting ERPGo deployment: $deployment_type ($version_tag)"

    # Load configuration
    load_config

    # Check prerequisites
    check_prerequisites

    # Create deployment backup
    local deployment_backup
    deployment_backup=$(create_deployment_backup)

    # Build application image
    build_image "$version_tag"

    # Deploy based on strategy
    local deployment_success=false
    case "$deployment_type" in
        "rolling")
            if deploy_rolling_update "$version_tag"; then
                deployment_success=true
            fi
            ;;
        "blue-green")
            if deploy_blue_green "$version_tag"; then
                deployment_success=true
            fi
            ;;
        *)
            error "Unknown deployment type: $deployment_type. Use 'rolling' or 'blue-green'"
            exit 1
            ;;
    esac

    if [[ "$deployment_success" == "true" ]]; then
        # Verify deployment
        if verify_deployment; then
            success "Deployment completed successfully!"
            send_notification "INFO" "ERPGo deployment completed successfully: $version_tag"
            return 0
        else
            error "Deployment verification failed"
            send_notification "ERROR" "ERPGo deployment verification failed"
            return 1
        fi
    else
        error "Deployment failed"
        send_notification "ERROR" "ERPGo deployment failed"
        return 1
    fi
}

# Cleanup old images and containers
cleanup() {
    log "Cleaning up old Docker resources..."

    # Remove unused images
    docker image prune -f

    # Remove unused containers
    docker container prune -f

    # Remove unused volumes (be careful with this)
    # docker volume prune -f

    success "Cleanup completed"
}

# Show usage
usage() {
    echo "ERPGo Production Deployment Script"
    echo ""
    echo "Usage:"
    echo "  $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  deploy [rolling|blue-green] [tag]  Deploy application (default: rolling latest)"
    echo "  rollback <backup_path>               Rollback to previous deployment"
    echo "  health [service]                     Run health checks"
    echo "  build [tag]                          Build application image"
    echo "  cleanup                              Clean up Docker resources"
    echo "  help                                 Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 deploy rolling v1.2.3             # Rolling update deployment"
    echo "  $0 deploy blue-green latest          # Blue-green deployment"
    echo "  $0 rollback ./backups/deployment_*   # Rollback to backup"
    echo "  $0 health api                        # Check API health"
    echo ""
    echo "Environment variables:"
    echo "  SLACK_WEBHOOK_URL                    Slack webhook for notifications"
    echo "  NOTIFICATION_EMAIL                   Email for notifications"
    echo "  HEALTH_CHECK_TIMEOUT                 Health check timeout (default: 300s)"
    echo "  ROLLBACK_TIMEOUT                     Rollback timeout (default: 600s)"
    echo ""
}

# Command handling
case "${1:-help}" in
    "deploy")
        deploy "${2:-rolling}" "${3:-latest}"
        ;;
    "rollback")
        rollback_deployment "$2"
        ;;
    "health")
        load_config
        run_health_checks "${2:-api}"
        ;;
    "build")
        load_config
        build_image "${2:-latest}"
        ;;
    "cleanup")
        cleanup
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