# ERPGo Makefile
# Provides convenient commands for development, testing, and deployment

.PHONY: help build clean test test-coverage lint dev docker-build docker-up docker-down install-tools generate format migrate-up migrate-down migrate-status

# Variables
APP_NAME := erpgo
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commit=$(COMMIT)"

# Docker variables
DOCKER_REGISTRY := ghcr.io
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(APP_NAME)
DOCKER_TAG := $(VERSION)

# Default target
.DEFAULT_GOAL := help

help: ## Display this help message
	@echo "$(APP_NAME) - Comprehensive ERP System"
	@echo "Version: $(VERSION)"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development
dev: ## Run the application in development mode with hot reload
	@echo "🚀 Starting development server with hot reload..."
	@which air > /dev/null || (echo "❌ 'air' not found. Run 'make install-tools' first." && exit 1)
	@air

run: ## Run the application without hot reload
	@echo "🚀 Starting application..."
	@go run $(LDFLAGS) cmd/api/main.go

run-worker: ## Run the background worker
	@echo "🚀 Starting background worker..."
	@go run $(LDFLAGS) cmd/worker/main.go

# Building
build: ## Build the application binaries
	@echo "🔨 Building application..."
	@mkdir -p bin
	@echo "Building API server..."
	@go build $(LDFLAGS) -o bin/$(APP_NAME)-api cmd/api/main.go
	@echo "Building migrator..."
	@go build $(LDFLAGS) -o bin/$(APP_NAME)-migrator cmd/migrator/main.go
	@echo "Building worker..."
	@go build $(LDFLAGS) -o bin/$(APP_NAME)-worker cmd/worker/main.go
	@echo "✅ Build completed successfully!"

build-all: ## Build binaries for all platforms
	@echo "🔨 Building for all platforms..."
	@mkdir -p bin
	@for os in linux windows darwin; do \
		for arch in amd64 arm64; do \
			if [ "$$os" = "windows" ] && [ "$$arch" = "arm64" ]; then continue; fi; \
			echo "Building $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o bin/$(APP_NAME)-$$os-$$arch cmd/api/main.go; \
		done; \
	done
	@echo "✅ Cross-platform build completed!"

clean: ## Clean build artifacts and temporary files
	@echo "🧹 Cleaning up..."
	@rm -rf bin/
	@rm -rf tmp/
	@rm -f coverage.out coverage.html
	@rm -f *.prof
	@echo "✅ Cleanup completed!"

# Testing
test: ## Run all tests
	@echo "🧪 Running tests..."
	@go test -v -race ./...

test-unit: ## Run unit tests only
	@echo "🧪 Running unit tests..."
	@go test -v -race -short ./...

test-integration: ## Run integration tests
	@echo "🧪 Running integration tests..."
	@go test -v -tags=integration ./tests/integration/...

test-e2e: ## Run end-to-end tests
	@echo "🧪 Running end-to-end tests..."
	@go test -v -tags=e2e ./tests/e2e/...

test-coverage: ## Run tests with coverage report
	@echo "🧪 Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out

benchmark: ## Run benchmarks
	@echo "⚡ Running benchmarks..."
	@go test -bench=. -benchmem ./...

race: ## Run tests with race detector
	@echo "🏁 Running tests with race detector..."
	@go test -race -short ./...

# Code Quality
lint: ## Run linter
	@echo "🔍 Running linter..."
	@which golangci-lint > /dev/null || (echo "❌ 'golangci-lint' not found. Run 'make install-tools' first." && exit 1)
	@golangci-lint run

format: ## Format code
	@echo "🎨 Formatting code..."
	@go fmt ./...
	@goimports -w .

fmt-check: ## Check if code is formatted
	@echo "🔍 Checking code formatting..."
	@test -z "$$(gofmt -l .)" || (echo "❌ Code is not formatted. Run 'make format' to fix." && exit 1)

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	@go vet ./...

security: ## Run security scanner
	@echo "🔒 Running security scan..."
	@which gosec > /dev/null || (echo "❌ 'gosec' not found. Run 'make install-tools' first." && exit 1)
	@gosec ./...

# Dependencies
deps: ## Download and verify dependencies
	@echo "📦 Downloading dependencies..."
	@go mod download
	@go mod verify

deps-update: ## Update dependencies
	@echo "📦 Updating dependencies..."
	@go get -u ./...
	@go mod tidy

deps-check: ## Check for outdated dependencies
	@echo "📦 Checking for outdated dependencies..."
	@which go-mod-outdated > /dev/null || (echo "❌ 'go-mod-outdated' not found. Run 'make install-tools' first." && exit 1)
	@go list -u -m -json all | go-mod-outdated -update -direct

# Generation
generate: ## Run code generation
	@echo "🏗️ Running code generation..."
	@go generate ./...

mocks: ## Generate mocks
	@echo "🎭 Generating mocks..."
	@which mockgen > /dev/null || (echo "❌ 'mockgen' not found. Run 'make install-tools' first." && exit 1)
	@go generate ./...

docs: ## Generate documentation
	@echo "📚 Generating documentation..."
	@which swag > /dev/null || (echo "❌ 'swag' not found. Run 'make install-tools' first." && exit 1)
	@swag init -g cmd/api/main.go -o docs/

# Database
migrate-up: ## Run database migrations up
	@echo "⬆️ Running database migrations..."
	@go run cmd/migrator/main.go up

migrate-down: ## Run database migrations down
	@echo "⬇️ Rolling back database migrations..."
	@go run cmd/migrator/main.go down

migrate-create: ## Create new migration (usage: make migrate-create name=migration_name)
	@if [ -z "$(name)" ]; then echo "❌ Migration name required. Usage: make migrate-create name=migration_name"; exit 1; fi
	@echo "📝 Creating migration: $(name)"
	@go run cmd/migrator/main.go create $(name)

migrate-status: ## Show migration status
	@echo "📊 Migration status:"
	@go run cmd/migrator/main.go status

db-seed: ## Seed database with test data
	@echo "🌱 Seeding database..."
	@go run cmd/seeder/main.go

# Docker
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest
	@echo "✅ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-push: ## Push Docker image to registry
	@echo "🐳 Pushing Docker image..."
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker push $(DOCKER_IMAGE):latest

docker-run: ## Run Docker container
	@echo "🐳 Running Docker container..."
	@docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE):$(DOCKER_TAG)

# Docker Compose
docker-up: ## Start services with Docker Compose
	@echo "🐳 Starting services..."
	@docker-compose up -d

docker-down: ## Stop services with Docker Compose
	@echo "🐳 Stopping services..."
	@docker-compose down

docker-logs: ## Show Docker Compose logs
	@echo "📋 Showing logs..."
	@docker-compose logs -f

docker-dev: ## Start development environment
	@echo "🐳 Starting development environment..."
	@docker-compose -f docker-compose.dev.yml up

docker-test: ## Start test environment
	@echo "🐳 Starting test environment..."
	@docker-compose -f docker-compose.yml --profile test up -d

# Tool Installation
install-tools: ## Install development tools
	@echo "🔧 Installing development tools..."
	@echo "Installing air..."
	@go install github.com/cosmtrek/air@latest
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing gosec..."
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "Installing mockgen..."
	@go install github.com/golang/mock/mockgen@latest
	@echo "Installing goimports..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing swag..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Installing go-mod-outdated..."
	@go install github.com/psampaz/go-mod-outdated@latest
	@echo "Installing godotenv..."
	@go install github.com/joho/godotenv/cmd/godotenv@latest
	@echo "✅ Tools installed successfully!"

install-deps: ## Install project dependencies
	@echo "📦 Installing dependencies..."
	@go mod download
	@go mod verify

# Git hooks
install-hooks: ## Install git hooks
	@echo "🪝 Installing git hooks..."
	@cp scripts/hooks/* .git/hooks/
	@chmod +x .git/hooks/*
	@echo "✅ Git hooks installed!"

# Quality Gates
check-all: ## Run all quality checks
	@echo "🔍 Running all quality checks..."
	@make fmt-check
	@make lint
	@make vet
	@make security
	@make test
	@echo "✅ All quality checks passed!"

ci: ## Run CI pipeline locally
	@echo "🚀 Running CI pipeline..."
	@make fmt-check
	@make lint
	@make security
	@make test-coverage
	@make test-integration
	@echo "✅ CI pipeline completed successfully!"

# Release
release: ## Create a new release
	@if [ -z "$(version)" ]; then echo "❌ Version required. Usage: make release version=vX.Y.Z"; exit 1; fi
	@echo "🚀 Creating release v$(version)..."
	@git tag -a v$(version) -m "Release v$(version)"
	@git push origin v$(version)
	@echo "✅ Release v$(version) created!"

# Utility
info: ## Show project information
	@echo "📋 Project Information:"
	@echo "  Name: $(APP_NAME)"
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Commit: $(COMMIT)"
	@echo "  Go Version: $(shell go version)"
	@echo "  Docker Image: $(DOCKER_IMAGE):$(DOCKER_TAG)"

version: ## Show current version
	@echo "$(VERSION)"

# Quick development commands
quick-start: ## Quick start for development
	@echo "🚀 Quick start setup..."
	@make install-tools
	@make install-deps
	@make docker-up
	@sleep 5
	@make migrate-up
	@echo "✅ Quick start completed! API is running at http://localhost:8080"

quick-test: ## Quick test for development
	@echo "🧪 Quick test..."
	@make fmt-check
	@make test-unit
	@echo "✅ Quick test completed!"

# Clean everything including Docker
clean-all: ## Clean everything including Docker containers and images
	@echo "🧹 Cleaning everything..."
	@make clean
	@docker-compose down -v --remove-orphans
	@docker system prune -f
	@echo "✅ Deep cleanup completed!"

# Environment setup
setup-dev: ## Set up development environment
	@echo "🔧 Setting up development environment..."
	@make install-tools
	@make install-deps
	@make install-hooks
	@if [ ! -f .env ]; then cp .env.example .env; echo "📝 Created .env file from template"; fi
	@echo "✅ Development environment setup completed!"

setup-env: ## Set up environment file
	@if [ ! -f .env ]; then cp .env.example .env; echo "📝 Created .env file from template"; else echo "📝 .env file already exists"; fi

# Load Testing
load-test: ## Run comprehensive load tests with k6
	@echo "⚡ Running comprehensive load tests..."
	@which k6 > /dev/null || (echo "❌ 'k6' not found. Install from: https://k6.io/docs/getting-started/installation/" && exit 1)
	@./tests/load/run-load-tests.sh

load-test-baseline: ## Run baseline load test (100 RPS)
	@echo "⚡ Running baseline load test..."
	@which k6 > /dev/null || (echo "❌ 'k6' not found. Install from: https://k6.io/docs/getting-started/installation/" && exit 1)
	@./tests/load/run-load-tests.sh baseline

load-test-peak: ## Run peak load test (1000 RPS)
	@echo "⚡ Running peak load test..."
	@which k6 > /dev/null || (echo "❌ 'k6' not found. Install from: https://k6.io/docs/getting-started/installation/" && exit 1)
	@./tests/load/run-load-tests.sh peak

load-test-stress: ## Run stress test (up to 5000 RPS)
	@echo "⚡ Running stress test..."
	@which k6 > /dev/null || (echo "❌ 'k6' not found. Install from: https://k6.io/docs/getting-started/installation/" && exit 1)
	@./tests/load/run-load-tests.sh stress

load-test-spike: ## Run spike test (sudden traffic spike)
	@echo "⚡ Running spike test..."
	@which k6 > /dev/null || (echo "❌ 'k6' not found. Install from: https://k6.io/docs/getting-started/installation/" && exit 1)
	@./tests/load/run-load-tests.sh spike

# Performance testing
perf-test: ## Run quick performance test with hey
	@echo "⚡ Running quick performance test..."
	@which hey > /dev/null || (echo "❌ 'hey' not found. Install with: go install github.com/rakyll/hey@latest" && exit 1)
	@echo "Running load test against http://localhost:8080..."
	@hey -n 1000 -c 10 -m GET http://localhost:8080/health

# Profiling
profile-cpu: ## Generate CPU profile
	@echo "📊 Generating CPU profile..."
	@go tool pprof http://localhost:8080/debug/pprof/profile

profile-memory: ## Generate memory profile
	@echo "📊 Generating memory profile..."
	@curl -o mem.prof http://localhost:8080/debug/pprof/heap
	@go tool pprof mem.prof

# Security audit
security-audit: ## Run comprehensive security audit
	@echo "🔒 Running security audit..."
	@make security
	@which nancy > /dev/null || (echo "❌ 'nancy' not found. Install with: go install github.com/sonatypecommunity/nancy@latest" && exit 1)
	@go list -m -json all | nancy sleuth
	@echo "✅ Security audit completed!"