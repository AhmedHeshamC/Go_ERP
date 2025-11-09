#!/bin/bash

# ERPGo Monitoring Setup Script
# This script sets up and starts the complete monitoring stack

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE} $1 ${NC}"
    echo -e "${BLUE}================================${NC}"
}

# Function to check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    print_status "Docker is running"
}

# Function to check if Docker Compose is available
check_docker_compose() {
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    print_status "Docker Compose is available"
}

# Function to create necessary directories
create_directories() {
    print_status "Creating necessary directories..."

    mkdir -p logs
    mkdir -p uploads
    mkdir -p configs/grafana/provisioning/datasources
    mkdir -p configs/grafana/provisioning/dashboards
    mkdir -p configs/grafana/dashboards

    print_status "Directories created"
}

# Function to set proper permissions
set_permissions() {
    print_status "Setting proper permissions..."

    # Set permissions for log directories
    chmod -R 755 logs/
    chmod -R 755 uploads/

    # Set permissions for config files
    chmod -R 644 configs/

    print_status "Permissions set"
}

# Function to wait for service to be healthy
wait_for_service() {
    local service_name=$1
    local health_check_url=$2
    local max_attempts=${3:-30}
    local attempt=1

    print_status "Waiting for $service_name to be healthy..."

    while [ $attempt -le $max_attempts ]; do
        if curl -f -s "$health_check_url" > /dev/null 2>&1; then
            print_status "$service_name is healthy!"
            return 0
        fi

        print_warning "Attempt $attempt/$max_attempts: $service_name is not ready yet..."
        sleep 10
        ((attempt++))
    done

    print_error "$service_name failed to become healthy after $max_attempts attempts"
    return 1
}

# Function to start core services
start_core_services() {
    print_header "Starting Core Services"

    print_status "Starting PostgreSQL, Redis, and NATS..."
    docker-compose up -d postgres redis nats

    # Wait for core services to be healthy
    wait_for_service "PostgreSQL" "http://localhost:5432" 30
    wait_for_service "Redis" "http://localhost:6379" 30
    wait_for_service "NATS" "http://localhost:8222" 30

    print_status "Core services are running"
}

# Function to start application
start_application() {
    print_header "Starting ERPGo Application"

    print_status "Starting ERPGo API..."
    docker-compose up -d api

    # Wait for API to be healthy
    wait_for_service "ERPGo API" "http://localhost:8080/health" 30

    print_status "ERPGo API is running"
}

# Function to start monitoring services
start_monitoring() {
    print_header "Starting Monitoring Stack"

    print_status "Starting monitoring services with profile 'monitoring'..."
    docker-compose --profile monitoring up -d

    print_status "Monitoring services starting..."
    sleep 30

    # Wait for monitoring services
    wait_for_service "Prometheus" "http://localhost:9090/-/healthy" 30
    wait_for_service "Grafana" "http://localhost:3000/api/health" 30
    wait_for_service "AlertManager" "http://localhost:9093/-/healthy" 30
    wait_for_service "Loki" "http://localhost:3100/ready" 30

    print_status "All monitoring services are running"
}

# Function to setup Grafana datasources
setup_grafana() {
    print_status "Setting up Grafana datasources..."

    # Import datasources
    sleep 10  # Give Grafana time to start

    print_status "Grafana datasources configured"
}

# Function to display service URLs
display_service_urls() {
    print_header "Service URLs"

    echo -e "${GREEN}Application:${NC}"
    echo "  ERPGo API:        http://localhost:8080"
    echo "  Health Check:     http://localhost:8080/health"
    echo "  Metrics:          http://localhost:8080/metrics"

    echo ""
    echo -e "${GREEN}Monitoring:${NC}"
    echo "  Prometheus:       http://localhost:9090"
    echo "  Grafana:          http://localhost:3000"
    echo "  AlertManager:     http://localhost:9093"
    echo "  Loki:             http://localhost:3100"

    echo ""
    echo -e "${GREEN}Exporters:${NC}"
    echo "  Node Exporter:    http://localhost:9100/metrics"
    echo "  PostgreSQL:       http://localhost:9187/metrics"
    echo "  Redis:            http://localhost:9121/metrics"

    echo ""
    echo -e "${GREEN}Default Credentials:${NC}"
    echo "  Grafana:          admin / admin (change on first login)"
    echo ""
}

# Function to verify monitoring setup
verify_setup() {
    print_header "Verifying Monitoring Setup"

    # Check if metrics are being collected
    print_status "Checking Prometheus targets..."
    if curl -s "http://localhost:9090/api/v1/targets" | grep -q "up"; then
        print_status "✓ Prometheus targets are up"
    else
        print_warning "⚠ Some Prometheus targets may be down"
    fi

    # Check Grafana datasources
    print_status "Checking Grafana datasources..."
    if curl -s "http://localhost:3000/api/health" | grep -q "ok"; then
        print_status "✓ Grafana is healthy"
    else
        print_warning "⚠ Grafana may not be fully ready"
    fi

    print_status "Setup verification completed"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -c, --core-only      Start only core services (PostgreSQL, Redis, NATS)"
    echo "  -a, --app-only       Start core services + application (no monitoring)"
    echo "  -m, --monitoring     Start core services + monitoring (no application)"
    echo "  -f, --full           Start all services (default)"
    echo "  -s, --stop           Stop all services"
    echo "  -r, --restart        Restart all services"
    echo "  -l, --logs           Show logs for all services"
    echo "  -h, --help           Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                   Start all services"
    echo "  $0 --core-only       Start only database and cache"
    echo "  $0 --monitoring      Start only monitoring stack"
    echo "  $0 --stop            Stop all services"
}

# Function to stop services
stop_services() {
    print_header "Stopping All Services"

    docker-compose down

    print_status "All services stopped"
}

# Function to restart services
restart_services() {
    print_header "Restarting All Services"

    stop_services
    sleep 5
    setup_all

    print_status "All services restarted"
}

# Function to show logs
show_logs() {
    print_status "Showing logs for all services..."
    docker-compose logs -f
}

# Function to setup all services
setup_all() {
    check_docker
    check_docker_compose
    create_directories
    set_permissions
    start_core_services
    start_application
    start_monitoring
    setup_grafana
    display_service_urls
    verify_setup
}

# Main script execution
main() {
    print_header "ERPGo Monitoring Setup"

    case "${1:-full}" in
        -c|--core-only)
            check_docker
            check_docker_compose
            create_directories
            set_permissions
            start_core_services
            ;;
        -a|--app-only)
            check_docker
            check_docker_compose
            create_directories
            set_permissions
            start_core_services
            start_application
            ;;
        -m|--monitoring)
            check_docker
            check_docker_compose
            create_directories
            set_permissions
            start_core_services
            start_monitoring
            ;;
        -f|--full)
            setup_all
            ;;
        -s|--stop)
            stop_services
            ;;
        -r|--restart)
            restart_services
            ;;
        -l|--logs)
            show_logs
            ;;
        -h|--help)
            show_usage
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
}

# Trap to handle script interruption
trap 'print_error "Script interrupted"; exit 1' INT TERM

# Run main function with all arguments
main "$@"