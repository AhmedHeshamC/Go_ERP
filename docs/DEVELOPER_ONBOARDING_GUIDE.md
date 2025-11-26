# ERPGo Developer Onboarding Guide

Welcome to the ERPGo development team! This guide will help you get up to speed with our codebase, development practices, and team workflows.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Development Environment Setup](#development-environment-setup)
3. [Codebase Overview](#codebase-overview)
4. [Development Workflow](#development-workflow)
5. [Testing Strategy](#testing-strategy)
6. [Code Review Process](#code-review-process)
7. [Deployment Process](#deployment-process)
8. [Troubleshooting](#troubleshooting)
9. [Resources](#resources)

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.21+**: [Installation Guide](https://golang.org/doc/install)
- **Docker & Docker Compose**: [Installation Guide](https://docs.docker.com/get-docker/)
- **PostgreSQL Client**: `brew install postgresql` (macOS) or `apt-get install postgresql-client` (Linux)
- **Redis Client**: `brew install redis` (macOS) or `apt-get install redis-tools` (Linux)
- **Git**: [Installation Guide](https://git-scm.com/downloads)
- **Make**: Usually pre-installed on Unix systems

### First Day Checklist

- [ ] Clone the repository
- [ ] Set up development environment
- [ ] Run the application locally
- [ ] Run all tests successfully
- [ ] Review architecture documentation
- [ ] Join team communication channels (Slack, etc.)
- [ ] Schedule 1:1 with team lead
- [ ] Review open issues and current sprint

## Development Environment Setup

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/erpgo.git
cd erpgo
```

### 2. Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install development tools
make install-tools

# This installs:
# - golangci-lint (linting)
# - gosec (security scanning)
# - mockery (mock generation)
# - swag (API documentation)
```

### 3. Set Up Environment Variables

```bash
# Copy example environment file
cp .env.example .env

# Edit .env with your local settings
# Key variables to set:
# - DB_PASSWORD
# - JWT_SECRET (generate with: openssl rand -base64 32)
# - REFRESH_SECRET (generate with: openssl rand -base64 32)
# - PASSWORD_PEPPER (generate with: openssl rand -base64 32)
```

### 4. Start Local Services

```bash
# Start PostgreSQL and Redis using Docker Compose
docker-compose up -d postgres redis

# Verify services are running
docker-compose ps

# Check logs if needed
docker-compose logs postgres
docker-compose logs redis
```

### 5. Run Database Migrations

```bash
# Run all migrations
make migrate-up

# Verify migrations
make migrate-status

# Seed development data (optional)
make seed-dev
```

### 6. Start the Application

```bash
# Run the application
make run

# Or with hot reload (recommended for development)
make dev

# The API will be available at http://localhost:8080
```

### 7. Verify Installation

```bash
# Check health endpoint
curl http://localhost:8080/health/live

# Expected response: {"status":"healthy"}

# Check readiness endpoint
curl http://localhost:8080/health/ready

# Expected response: {"status":"ready","checks":{...}}

# Run tests
make test

# All tests should pass
```

## Codebase Overview

### Project Structure

```
erpgo/
├── cmd/                    # Application entry points
│   ├── api/               # Main API server
│   ├── generate-jwt/      # JWT generation utility
│   └── export-jwt/        # JWT export utility
├── internal/              # Private application code
│   ├── application/       # Application services (business logic)
│   │   └── services/      # Service implementations
│   ├── domain/            # Domain models and interfaces
│   │   ├── users/         # User domain
│   │   ├── products/      # Product domain
│   │   ├── orders/        # Order domain
│   │   └── inventory/     # Inventory domain
│   ├── infrastructure/    # Infrastructure implementations
│   │   ├── repositories/  # Database repositories
│   │   ├── storage/       # File storage
│   │   └── monitoring/    # Monitoring infrastructure
│   └── interfaces/        # External interfaces
│       └── http/          # HTTP handlers and routes
├── pkg/                   # Public packages (reusable)
│   ├── auth/              # Authentication utilities
│   ├── cache/             # Caching utilities
│   ├── database/          # Database utilities
│   ├── errors/            # Error handling
│   ├── logger/            # Logging utilities
│   ├── monitoring/        # Monitoring utilities
│   ├── ratelimit/         # Rate limiting
│   ├── secrets/           # Secret management
│   └── validation/        # Input validation
├── migrations/            # Database migrations
├── tests/                 # Test files
│   ├── unit/              # Unit tests
│   ├── integration/       # Integration tests
│   ├── e2e/               # End-to-end tests
│   └── load/              # Load tests
├── docs/                  # Documentation
├── scripts/               # Utility scripts
├── configs/               # Configuration files
└── Makefile              # Build and development commands
```

### Architecture Layers

**1. Domain Layer** (`internal/domain/`)
- Core business entities and logic
- Domain-specific errors
- Repository interfaces
- No dependencies on other layers

**2. Application Layer** (`internal/application/`)
- Use cases and business workflows
- Service implementations
- Transaction management
- Depends on domain layer

**3. Infrastructure Layer** (`internal/infrastructure/`)
- Database implementations
- External service integrations
- File storage implementations
- Depends on domain layer

**4. Interface Layer** (`internal/interfaces/`)
- HTTP handlers
- Request/response DTOs
- Middleware
- Depends on application layer

### Key Design Patterns

**Repository Pattern**
```go
// Domain defines interface
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id uuid.UUID) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id uuid.UUID) error
}

// Infrastructure implements interface
type PostgresUserRepository struct {
    db *sql.DB
}
```

**Service Pattern**
```go
// Application service coordinates business logic
type UserService struct {
    userRepo UserRepository
    roleRepo RoleRepository
    cache    CacheManager
    txManager TransactionManager
}

func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
    // Validation, business logic, transaction management
}
```

**Dependency Injection**
```go
// Wire up dependencies in main.go
func main() {
    db := database.Connect(config.DatabaseURL)
    cache := cache.NewRedisCache(config.RedisURL)
    
    userRepo := repositories.NewPostgresUserRepository(db)
    userService := services.NewUserService(userRepo, cache)
    userHandler := handlers.NewUserHandler(userService)
    
    router.SetupRoutes(userHandler)
}
```

## Development Workflow

### 1. Pick a Task

```bash
# Check current sprint board
# Assign yourself a task
# Move task to "In Progress"
```

### 2. Create a Feature Branch

```bash
# Update main branch
git checkout main
git pull origin main

# Create feature branch
git checkout -b feature/TICKET-123-short-description

# Branch naming conventions:
# - feature/TICKET-123-description (new features)
# - fix/TICKET-123-description (bug fixes)
# - refactor/TICKET-123-description (refactoring)
# - docs/TICKET-123-description (documentation)
```

### 3. Make Changes

```bash
# Write code following our style guide
# Add tests for new functionality
# Update documentation if needed

# Run tests frequently
make test

# Run linter
make lint

# Format code
make format
```

### 4. Commit Changes

```bash
# Stage changes
git add .

# Commit with descriptive message
git commit -m "feat(users): add email verification

- Implement email verification service
- Add verification token generation
- Add email sending integration
- Add tests for verification flow

Closes TICKET-123"

# Commit message format:
# <type>(<scope>): <subject>
#
# <body>
#
# <footer>
#
# Types: feat, fix, docs, style, refactor, test, chore
```

### 5. Push and Create Pull Request

```bash
# Push branch
git push origin feature/TICKET-123-short-description

# Create pull request on GitHub
# - Fill out PR template
# - Link to ticket
# - Add reviewers
# - Add labels
```

### 6. Address Review Comments

```bash
# Make requested changes
# Commit changes
git commit -m "address review comments"

# Push updates
git push origin feature/TICKET-123-short-description
```

### 7. Merge

```bash
# Once approved, squash and merge
# Delete feature branch
git branch -d feature/TICKET-123-short-description
```

## Testing Strategy

### Unit Tests

**Location**: Next to the code being tested (e.g., `user_service_test.go`)

**Purpose**: Test individual functions and methods in isolation

**Example**:
```go
func TestUserService_CreateUser(t *testing.T) {
    // Arrange
    mockRepo := new(MockUserRepository)
    service := NewUserService(mockRepo, nil)
    
    user := &User{Email: "test@example.com"}
    mockRepo.On("Create", mock.Anything, user).Return(nil)
    
    // Act
    err := service.CreateUser(context.Background(), user)
    
    // Assert
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

**Run unit tests**:
```bash
make test-unit
# or
go test ./... -short
```

### Integration Tests

**Location**: `tests/integration/`

**Purpose**: Test interactions between components with real dependencies

**Example**:
```go
func TestUserAPI_CreateUser_Integration(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    // Create test server
    server := setupTestServer(db)
    
    // Make request
    resp := makeRequest(server, "POST", "/api/v1/users", userPayload)
    
    // Assert
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
```

**Run integration tests**:
```bash
make test-integration
```

### Property-Based Tests

**Location**: `*_property_test.go` files

**Purpose**: Test properties that should hold for all inputs

**Example**:
```go
func TestProperty_JWTSecretMinimumLength(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("secrets under 32 bytes are rejected", 
        prop.ForAll(
            func(secret string) bool {
                validator := NewJWTSecretValidator(256)
                err := validator.Validate(secret)
                if len(secret) < 32 {
                    return err != nil
                }
                return err == nil
            },
            gen.AnyString(),
        ),
    )
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

**Run property tests**:
```bash
make test-property
```

### End-to-End Tests

**Location**: `tests/e2e/`

**Purpose**: Test complete user workflows

**Run e2e tests**:
```bash
make test-e2e
```

### Test Coverage

```bash
# Generate coverage report
make coverage

# View coverage in browser
make coverage-html

# Target: 80% coverage for critical paths
```

## Code Review Process

### As an Author

1. **Before requesting review**:
   - [ ] All tests pass
   - [ ] Linter passes
   - [ ] Code is self-documented
   - [ ] PR description is complete
   - [ ] Screenshots/videos for UI changes

2. **Requesting review**:
   - Add 2+ reviewers
   - Add relevant labels
   - Link to ticket
   - Respond to comments promptly

3. **After approval**:
   - Squash and merge
   - Delete branch
   - Move ticket to "Done"

### As a Reviewer

1. **Review checklist**:
   - [ ] Code follows style guide
   - [ ] Tests are comprehensive
   - [ ] No security vulnerabilities
   - [ ] Performance considerations
   - [ ] Documentation updated
   - [ ] Error handling is proper

2. **Providing feedback**:
   - Be constructive and specific
   - Explain the "why" behind suggestions
   - Distinguish between blocking and non-blocking comments
   - Approve when satisfied

## Deployment Process

### Development Environment

```bash
# Automatic deployment on merge to main
# Deployed to: https://dev.erpgo.com
```

### Staging Environment

```bash
# Manual deployment via GitHub Actions
# Deployed to: https://staging.erpgo.com

# To deploy:
# 1. Go to Actions tab
# 2. Select "Deploy to Staging"
# 3. Run workflow
```

### Production Environment

```bash
# Scheduled deployments (Tuesdays and Thursdays)
# Deployed to: https://api.erpgo.com

# Emergency hotfix process:
# 1. Create hotfix branch from main
# 2. Make fix and test
# 3. Get expedited review
# 4. Deploy via "Emergency Deploy" workflow
```

See [Deployment Runbook](./operations/RUNBOOKS.md#deployment-procedures) for detailed procedures.

## Troubleshooting

### Common Issues

#### Database Connection Errors

```bash
# Check if PostgreSQL is running
docker-compose ps postgres

# Check logs
docker-compose logs postgres

# Restart PostgreSQL
docker-compose restart postgres

# Verify connection
psql -h localhost -U postgres -d erpgo
```

#### Redis Connection Errors

```bash
# Check if Redis is running
docker-compose ps redis

# Test connection
redis-cli ping

# Restart Redis
docker-compose restart redis
```

#### Tests Failing

```bash
# Clean test cache
go clean -testcache

# Run specific test
go test -v ./path/to/package -run TestName

# Run with race detector
go test -race ./...

# Check for database state issues
make test-db-reset
```

#### Build Errors

```bash
# Clean build cache
go clean -cache

# Update dependencies
go mod tidy
go mod download

# Rebuild
make build
```

### Getting Help

1. **Check documentation**: Start with docs in `docs/` directory
2. **Search existing issues**: Check GitHub issues
3. **Ask in Slack**: #engineering channel
4. **Pair programming**: Schedule time with a teammate
5. **Team lead**: Reach out for guidance

## Resources

### Documentation

- [Architecture Overview](./ARCHITECTURE_OVERVIEW.md)
- [API Documentation](./API_DOCUMENTATION.md)
- [Database Schema](./DATABASE_SCHEMA.md)
- [Security Best Practices](./SECURITY_BEST_PRACTICES.md)
- [Monitoring Guide](./MONITORING.md)
- [Operational Runbooks](./operations/RUNBOOKS.md)

### External Resources

- [Go Documentation](https://golang.org/doc/)
- [Gin Framework](https://gin-gonic.com/docs/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Redis Documentation](https://redis.io/documentation)
- [Docker Documentation](https://docs.docker.com/)

### Team Contacts

- **Engineering Manager**: [Name] - [email]
- **Tech Lead**: [Name] - [email]
- **DevOps Lead**: [Name] - [email]
- **On-call Rotation**: Check PagerDuty

### Communication Channels

- **Slack**:
  - #engineering (general engineering discussion)
  - #engineering-alerts (automated alerts)
  - #engineering-deploys (deployment notifications)
  - #engineering-incidents (incident response)

- **Meetings**:
  - Daily Standup: 9:30 AM (15 minutes)
  - Sprint Planning: Every 2 weeks (2 hours)
  - Sprint Retro: Every 2 weeks (1 hour)
  - Tech Talks: Fridays 4 PM (optional)

### Development Tools

- **IDE**: VS Code or GoLand recommended
- **VS Code Extensions**:
  - Go (official)
  - GitLens
  - Docker
  - PostgreSQL
  - REST Client

- **Useful Commands**:
```bash
# See all available make commands
make help

# Run development server with hot reload
make dev

# Run all checks (tests, lint, security)
make check

# Generate mocks
make generate-mocks

# Update API documentation
make swagger

# Run database migrations
make migrate-up

# Rollback last migration
make migrate-down

# Create new migration
make migrate-create NAME=add_users_table
```

## Next Steps

Now that you're set up, here are some suggested next steps:

1. **Week 1**: 
   - Complete environment setup
   - Read architecture documentation
   - Fix a "good first issue"
   - Attend team meetings

2. **Week 2**:
   - Take on a small feature
   - Pair with a teammate
   - Review others' PRs
   - Learn deployment process

3. **Week 3-4**:
   - Work on medium-sized features
   - Participate in on-call rotation (shadowing)
   - Contribute to documentation
   - Share learnings in tech talk

4. **Month 2+**:
   - Own larger features
   - Mentor new team members
   - Contribute to architecture decisions
   - Lead technical initiatives

Welcome to the team! We're excited to have you here. Don't hesitate to ask questions – we're all here to help each other succeed.
