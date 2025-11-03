#!/bin/bash

# ERPGo Setup Script
# This script sets up the development environment for ERPGo

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21 or higher."
        exit 1
    fi

    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_status "Go version: $GO_VERSION"

    # Check if Go version is 1.21 or higher
    if ! go version | grep -E "go1\.2[1-9]|go1\.[3-9][0-9]|go[2-9]\." > /dev/null; then
        print_error "Go version 1.21 or higher is required."
        exit 1
    fi
}

# Check if Docker is installed
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_warning "Docker is not installed. Some features may not work."
        return 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        print_warning "Docker Compose is not installed. Some features may not work."
        return 1
    fi

    print_status "Docker version: $(docker --version)"
    print_status "Docker Compose version: $(docker-compose --version)"
    return 0
}

# Setup environment file
setup_env() {
    if [ ! -f .env ]; then
        print_status "Creating .env file from template..."
        cp .env.example .env
        print_status "Created .env file. Please review and update the values."
    else
        print_status ".env file already exists."
    fi
}

# Install Go dependencies
install_deps() {
    print_status "Installing Go dependencies..."
    go mod download
    go mod verify
    print_status "Dependencies installed successfully."
}

# Install development tools
install_tools() {
    print_status "Installing development tools..."

    # List of tools to install
    tools=(
        "github.com/cosmtrek/air@latest"
        "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
        "github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"
        "github.com/golang/mock/mockgen@latest"
        "golang.org/x/tools/cmd/goimports@latest"
        "github.com/swaggo/swag/cmd/swag@latest"
        "github.com/psampaz/go-mod-outdated@latest"
        "github.com/joho/godotenv/cmd/godotenv@latest"
        "github.com/rakyll/hey@latest"
        "github.com/sonatypecommunity/nancy@latest"
    )

    for tool in "${tools[@]}"; do
        tool_name=$(basename $(echo $tool | cut -d'@' -f1))
        if ! command -v $tool_name &> /dev/null; then
            print_status "Installing $tool_name..."
            go install $tool
        else
            print_status "$tool_name is already installed."
        fi
    done

    print_status "Development tools installed successfully."
}

# Setup Git hooks
setup_git_hooks() {
    print_status "Setting up Git hooks..."

    # Create pre-commit hook
    cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
# Pre-commit hook for ERPGo

echo "Running pre-commit checks..."

# Run go fmt
echo "Checking code formatting..."
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
    echo "The following files are not formatted:"
    echo "$unformatted"
    echo "Please run 'make format' to fix formatting issues."
    exit 1
fi

# Run go vet
echo "Running go vet..."
go vet ./...
if [ $? -ne 0 ]; then
    echo "go vet failed. Please fix the issues before committing."
    exit 1
fi

# Run golangci-lint
echo "Running linter..."
golangci-lint run
if [ $? -ne 0 ]; then
    echo "Linting failed. Please fix the issues before committing."
    exit 1
fi

# Run tests
echo "Running tests..."
go test -short ./...
if [ $? -ne 0 ]; then
    echo "Tests failed. Please fix the issues before committing."
    exit 1
fi

echo "Pre-commit checks passed!"
EOF

    chmod +x .git/hooks/pre-commit
    print_status "Git hooks installed successfully."
}

# Start services
start_services() {
    if command -v docker-compose &> /dev/null; then
        print_status "Starting services with Docker Compose..."
        docker-compose up -d postgres redis

        # Wait for services to be ready
        print_status "Waiting for services to be ready..."
        sleep 10

        # Run migrations
        print_status "Running database migrations..."
        if [ -f cmd/migrator/main.go ]; then
            go run cmd/migrator/main.go up
        else
            print_warning "Migrator not found. Please run migrations manually."
        fi

        print_status "Services started successfully."
    else
        print_warning "Docker Compose not available. Please start services manually."
    fi
}

# Main setup function
main() {
    print_status "Starting ERPGo setup..."

    # Check prerequisites
    check_go
    check_docker

    # Setup environment
    setup_env

    # Install dependencies
    install_deps
    install_tools

    # Setup Git hooks
    setup_git_hooks

    # Start services if Docker is available
    if command -v docker-compose &> /dev/null; then
        read -p "Do you want to start the development services? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            start_services
        fi
    fi

    print_status "Setup completed successfully!"
    echo
    print_status "Next steps:"
    echo "1. Review and update .env file"
    echo "2. Run 'make dev' to start the development server"
    echo "3. Visit http://localhost:8080/health to check if the API is running"
    echo "4. Visit http://localhost:8080/docs to view API documentation"
    echo
    print_status "For more information, see the README.md file."
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "ERPGo Setup Script"
        echo
        echo "Usage: $0 [OPTION]"
        echo
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --no-docker    Skip Docker-related setup"
        echo "  --tools-only   Only install development tools"
        echo
        exit 0
        ;;
    --no-docker)
        print_warning "Skipping Docker setup."
        check_go
        setup_env
        install_deps
        install_tools
        setup_git_hooks
        print_status "Setup completed (without Docker)!"
        ;;
    --tools-only)
        print_status "Only installing development tools..."
        install_tools
        print_status "Tools installed successfully!"
        ;;
    *)
        main
        ;;
esac