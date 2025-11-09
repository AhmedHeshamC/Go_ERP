#!/bin/bash

# ERPGo Configuration Validation Script
# This script validates environment configuration for production readiness

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Validation results
VALIDATION_ERRORS=0
VALIDATION_WARNINGS=0

# Logging functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    ((VALIDATION_ERRORS++))
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((VALIDATION_WARNINGS++))
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

info() {
    echo -e "[INFO] $1"
}

# Load configuration file
load_config() {
    local config_file="${1:-$PROJECT_ROOT/.env.production}"

    if [[ ! -f "$config_file" ]]; then
        error "Configuration file not found: $config_file"
        return 1
    fi

    log "Loading configuration from: $config_file"

    # Load environment variables
    set -a
    source <(grep -v '^#' "$config_file" | grep -v '^$' | grep '=')
    set +a

    success "Configuration loaded successfully"
}

# Validate environment type
validate_environment() {
    log "Validating environment configuration..."

    local env="${ENVIRONMENT:-development}"

    case "$env" in
        "production"|"prod")
            info "Environment: Production (strict validation enabled)"
            ENVIRONMENT_TYPE="production"
            ;;
        "staging"|"stage")
            info "Environment: Staging (moderate validation enabled)"
            ENVIRONMENT_TYPE="staging"
            ;;
        "development"|"dev")
            info "Environment: Development (lenient validation enabled)"
            ENVIRONMENT_TYPE="development"
            ;;
        *)
            error "Unknown environment: $env. Expected: production, staging, or development"
            return 1
            ;;
    esac
}

# Validate required variables
validate_required_variables() {
    log "Validating required configuration variables..."

    local required_vars=(
        "ENVIRONMENT"
        "SERVER_PORT"
        "DATABASE_URL"
        "JWT_SECRET"
    )

    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            error "Required variable not set: $var"
        else
            success "Required variable set: $var"
        fi
    done
}

# Validate security configuration
validate_security() {
    log "Validating security configuration..."

    # JWT Secret validation
    if [[ "${JWT_SECRET:-}" == "dev_secret_change_me_in_production_please" ]] || \
       [[ "${JWT_SECRET:-}" == "your-super-secret-jwt-key" ]] || \
       [[ -z "${JWT_SECRET:-}" ]]; then
        error "JWT_SECRET is using a default value or is empty. Please set a secure secret."
    else
        if [[ ${#JWT_SECRET} -lt 32 ]]; then
            error "JWT_SECRET must be at least 32 characters long. Current length: ${#JWT_SECRET}"
        else
            success "JWT_SECRET is properly configured"
        fi
    fi

    # Bcrypt cost validation
    local bcrypt_cost="${BCRYPT_COST:-12}"
    if [[ "$ENVIRONMENT_TYPE" == "production" && $bcrypt_cost -lt 12 ]]; then
        error "BCRYPT_COST should be at least 12 for production. Current: $bcrypt_cost"
    elif [[ $bcrypt_cost -lt 10 ]]; then
        warn "BCRYPT_COST is low (recommended: 12+ for production). Current: $bcrypt_cost"
    else
        success "BCRYPT_COST is acceptable: $bcrypt_cost"
    fi

    # Database SSL validation
    if [[ "$ENVIRONMENT_TYPE" == "production" ]]; then
        local ssl_mode="${DATABASE_SSL_MODE:-disable}"
        if [[ "$ssl_mode" != "require" && "$ssl_mode" != "verify-ca" && "$ssl_mode" != "verify-full" ]]; then
            error "DATABASE_SSL_MODE should be 'require', 'verify-ca', or 'verify-full' for production. Current: $ssl_mode"
        else
            success "DATABASE_SSL_MODE is properly configured for production: $ssl_mode"
        fi
    fi

    # Rate limiting validation
    if [[ "$ENVIRONMENT_TYPE" == "production" && "${RATE_LIMIT_ENABLED:-true}" != "true" ]]; then
        error "RATE_LIMIT_ENABLED should be true for production"
    else
        success "RATE_LIMIT_ENABLED is properly configured"
    fi

    # Debug mode validation
    if [[ "$ENVIRONMENT_TYPE" == "production" && "${DEBUG_MODE:-false}" != "false" ]]; then
        error "DEBUG_MODE should be false for production"
    else
        success "DEBUG_MODE is properly configured"
    fi

    # API docs validation
    if [[ "$ENVIRONMENT_TYPE" == "production" && "${API_DOCS_ENABLED:-true}" != "false" ]]; then
        warn "API_DOCS_ENABLED should be false for production security"
    fi
}

# Validate database configuration
validate_database() {
    log "Validating database configuration..."

    # Database URL validation
    local db_url="${DATABASE_URL:-}"
    if [[ -z "$db_url" ]]; then
        error "DATABASE_URL is not set"
        return 1
    fi

    if [[ ! "$db_url" =~ ^postgres:// ]]; then
        error "DATABASE_URL should be a PostgreSQL connection string"
    else
        success "DATABASE_URL format is valid"
    fi

    # Connection pool validation
    local max_conn="${MAX_CONNECTIONS:-20}"
    local min_conn="${MIN_CONNECTIONS:-5}"

    if [[ $max_conn -le $min_conn ]]; then
        error "MAX_CONNECTIONS ($max_conn) must be greater than MIN_CONNECTIONS ($min_conn)"
    else
        success "Database connection pool configuration is valid"
    fi

    # Production-specific database validation
    if [[ "$ENVIRONMENT_TYPE" == "production" ]]; then
        # Check for read replica configuration
        if [[ -z "${DATABASE_READ_URL:-}" ]]; then
            warn "DATABASE_READ_URL not configured. Consider adding a read replica for production scalability"
        else
            success "DATABASE_READ_URL is configured"
        fi
    fi
}

# Validate Redis configuration
validate_redis() {
    log "Validating Redis configuration..."

    local redis_url="${REDIS_URL:-}"
    if [[ -z "$redis_url" ]]; then
        error "REDIS_URL is not set"
        return 1
    fi

    if [[ ! "$redis_url" =~ ^redis:// ]]; then
        error "REDIS_URL should start with redis://"
    else
        success "REDIS_URL format is valid"
    fi

    # Redis pool size validation
    local redis_pool="${REDIS_POOL_SIZE:-10}"
    if [[ $redis_pool -lt 5 ]]; then
        warn "REDIS_POOL_SIZE is low (recommended: 10+ for production). Current: $redis_pool"
    else
        success "REDIS_POOL_SIZE is acceptable: $redis_pool"
    fi

    # Production-specific Redis validation
    if [[ "$ENVIRONMENT_TYPE" == "production" ]]; then
        # Check for Redis password
        if [[ -z "${REDIS_PASSWORD:-}" ]]; then
            warn "REDIS_PASSWORD not configured. Redis should be password-protected in production"
        else
            success "REDIS_PASSWORD is configured"
        fi

        # Check for Redis replica configuration
        if [[ -n "${REDIS_REPLICA_URL:-}" ]]; then
            success "REDIS_REPLICA_URL is configured for high availability"
        fi
    fi
}

# Validate file storage configuration
validate_storage() {
    log "Validating file storage configuration..."

    local storage_type="${STORAGE_TYPE:-local}"

    case "$storage_type" in
        "local")
            if [[ "$ENVIRONMENT_TYPE" == "production" ]]; then
                error "Local storage should not be used in production. Use S3 or similar cloud storage"
            else
                success "Local storage configured (appropriate for $ENVIRONMENT_TYPE)"
            fi
            ;;
        "s3")
            local s3_vars=("S3_BUCKET" "S3_REGION" "S3_ACCESS_KEY" "S3_SECRET_KEY")
            for var in "${s3_vars[@]}"; do
                if [[ -z "${!var:-}" ]]; then
                    error "S3 configuration incomplete: $var is not set"
                fi
            done

            if [[ ${#s3_vars[@]} -eq $(printf '%s\n' "${s3_vars[@]}" | xargs -I {} sh -c 'test -n "${'{}':-}" && echo {}' | wc -l) ]]; then
                success "S3 storage is properly configured"
            fi
            ;;
        *)
            error "Unknown STORAGE_TYPE: $storage_type. Expected: local or s3"
            ;;
    esac
}

# Validate email configuration
validate_email() {
    log "Validating email configuration..."

    if [[ "$ENVIRONMENT_TYPE" == "production" ]]; then
        local email_vars=("SMTP_HOST" "SMTP_PORT" "SMTP_USERNAME" "SMTP_PASSWORD" "EMAIL_FROM")
        for var in "${email_vars[@]}"; do
            if [[ -z "${!var:-}" ]]; then
                error "Email configuration incomplete: $var is not set"
            fi
        done

        if [[ ${#email_vars[@]} -eq $(printf '%s\n' "${email_vars[@]}" | xargs -I {} sh -c 'test -n "${'{}':-}" && echo {}' | wc -l) ]]; then
            success "Email configuration is complete for production"
        fi
    else
        info "Email validation skipped for $ENVIRONMENT_TYPE environment"
    fi
}

# Validate monitoring configuration
validate_monitoring() {
    log "Validating monitoring configuration..."

    local metrics_enabled="${METRICS_ENABLED:-true}"
    local tracing_enabled="${TRACING_ENABLED:-false}"

    if [[ "$metrics_enabled" != "true" ]]; then
        warn "METRICS_ENABLED is disabled. Monitoring is recommended for production"
    else
        success "Metrics are enabled"
    fi

    if [[ "$ENVIRONMENT_TYPE" == "production" && "$tracing_enabled" != "true" ]]; then
        warn "TRACING_ENABLED is disabled. Distributed tracing is recommended for production"
    else
        success "Tracing configuration: $tracing_enabled"
    fi
}

# Validate CORS configuration
validate_cors() {
    log "Validating CORS configuration..."

    if [[ "$ENVIRONMENT_TYPE" == "production" ]]; then
        local cors_origins="${PRODUCTION_CORS_ORIGINS:-${CORS_ORIGINS:-}}"

        if [[ -z "$cors_origins" ]]; then
            error "CORS origins not configured for production"
        elif [[ "$cors_origins" == *"localhost"* ]]; then
            error "CORS origins contain localhost, which should not be in production"
        else
            success "CORS origins are properly configured for production"
        fi
    fi
}

# Validate performance settings
validate_performance() {
    log "Validating performance configuration..."

    # Server timeout validation
    local read_timeout="${SERVER_READ_TIMEOUT:-30s}"
    local write_timeout="${SERVER_WRITE_TIMEOUT:-30s}"

    if [[ "$ENVIRONMENT_TYPE" == "production" ]]; then
        # Check for reasonable timeout values
        if [[ "$read_timeout" == "30s" || "$write_timeout" == "30s" ]]; then
            info "Server timeouts are at default values. Consider tuning for your specific needs"
        fi

        # Check worker count
        local worker_count="${WORKER_COUNT:-5}"
        if [[ $worker_count -lt 5 ]]; then
            warn "WORKER_COUNT is low for production (recommended: 10+). Current: $worker_count"
        fi
    fi

    success "Performance settings reviewed"
}

# Generate validation report
generate_report() {
    local report_file="$PROJECT_ROOT/config-validation-report-$(date '+%Y%m%d_%H%M%S').txt"

    {
        echo "=== ERPGo Configuration Validation Report ==="
        echo "Generated: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
        echo "Environment: $ENVIRONMENT_TYPE"
        echo ""
        echo "Validation Results:"
        echo "  Errors: $VALIDATION_ERRORS"
        echo "  Warnings: $VALIDATION_WARNINGS"
        echo ""
        echo "Configuration File: ${CONFIG_FILE:-.env.production}"
        echo ""

        if [[ $VALIDATION_ERRORS -eq 0 ]]; then
            echo "Status: PASSED"
            echo "Configuration is ready for deployment"
        else
            echo "Status: FAILED"
            echo "Configuration has critical issues that must be resolved"
        fi

        if [[ $VALIDATION_WARNINGS -gt 0 ]]; then
            echo ""
            echo "Note: There are $VALIDATION_WARNINGS warnings that should be reviewed"
        fi

        echo ""
        echo "=== End of Report ==="
    } > "$report_file"

    echo "Validation report generated: $report_file"
    echo "$report_file"
}

# Main validation function
main() {
    local config_file="${1:-}"

    log "Starting ERPGo configuration validation..."

    # Load configuration
    if [[ -n "$config_file" ]]; then
        load_config "$config_file"
    else
        # Try to find the appropriate config file based on environment
        if [[ -f "$PROJECT_ROOT/.env.production" ]]; then
            load_config "$PROJECT_ROOT/.env.production"
        elif [[ -f "$PROJECT_ROOT/.env.staging" ]]; then
            load_config "$PROJECT_ROOT/.env.staging"
        elif [[ -f "$PROJECT_ROOT/.env" ]]; then
            load_config "$PROJECT_ROOT/.env"
        else
            error "No configuration file found. Expected .env.production, .env.staging, or .env"
            exit 1
        fi
    fi

    # Run validations
    validate_environment
    validate_required_variables
    validate_security
    validate_database
    validate_redis
    validate_storage
    validate_email
    validate_monitoring
    validate_cors
    validate_performance

    # Generate report
    local report_file
    report_file=$(generate_report)

    # Print summary
    echo ""
    log "Validation Summary:"
    info "Errors: $VALIDATION_ERRORS"
    info "Warnings: $VALIDATION_WARNINGS"

    if [[ $VALIDATION_ERRORS -eq 0 ]]; then
        success "Configuration validation PASSED"
        success "Configuration is ready for deployment"
        exit 0
    else
        error "Configuration validation FAILED"
        error "Please resolve the critical errors before deployment"
        exit 1
    fi
}

# Show usage
usage() {
    echo "ERPGo Configuration Validation Script"
    echo ""
    echo "Usage:"
    echo "  $0 [CONFIG_FILE]"
    echo ""
    echo "Arguments:"
    echo "  CONFIG_FILE    Path to environment configuration file"
    echo "                 (default: auto-detects .env.production, .env.staging, or .env)"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Auto-detect config file"
    echo "  $0 .env.production                   # Validate production config"
    echo "  $0 configs/environments/.env.prod    # Validate custom config"
}

# Parse command line arguments
case "${1:-}" in
    "help"|"-h"|"--help")
        usage
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac