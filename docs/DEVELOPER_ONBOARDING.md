# ERPGo Developer Onboarding Guide

## Welcome to ERPGo!

This guide will help you get started with the ERPGo codebase, understand the architecture, and start contributing effectively.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Development Environment Setup](#development-environment-setup)
3. [Project Structure](#project-structure)
4. [Architecture Overview](#architecture-overview)
5. [Key Concepts](#key-concepts)
6. [Development Workflow](#development-workflow)
7. [Running Tests](#running-tests)
8. [Debugging](#debugging)
9. [Code Style and Standards](#code-style-and-standards)
10. [Common Development Tasks](#common-development-tasks)
11. [Troubleshooting](#troubleshooting)
12. [Resources and References](#resources-and-references)

## Prerequisites

### Required Software

- **Go 1.21+**: The primary programming language
- **PostgreSQL 14+**: Primary database
- **Redis 6+**: Caching and session storage
- **Docker & Docker Compose**: For local development environment
- **Git**: Version control
- **Make**: Build automation (recommended)

### Development Tools

- **VS Code** or **GoLand**: Recommended IDEs
- **Postman** or **Insomnia**: API testing
- **pgAdmin** or **DBeaver**: Database management
- **Redis Desktop Manager**: Redis management (optional)

### Environment Variables

Create a `.env` file in the root directory with the following variables:

```env
# Server Configuration
SERVER_PORT=8080
ENVIRONMENT=development
LOG_LEVEL=debug
DEBUG_MODE=true

# Database Configuration
DATABASE_URL=postgres://erpgo:password@localhost:5432/erp?sslmode=disable
DB_HOST=localhost
DB_PORT=5432
DB_NAME=erp
DB_USER=erpgo
DB_PASSWORD=password
DB_SSL_MODE=disable

# Redis Configuration
REDIS_URL=redis://localhost:6379
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
JWT_EXPIRY=24h
REFRESH_EXPIRY=168h

# File Storage
UPLOAD_PATH=./uploads
MAX_FILE_SIZE=10MB

# CORS Configuration
CORS_ORIGINS=http://localhost:3000,http://localhost:8080

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=100
RATE_LIMIT_BURST=200

# API Documentation
API_DOCS_ENABLED=true
API_DOCS_PATH=/docs
```

## Development Environment Setup

### Option 1: Using Docker Compose (Recommended)

1. **Clone the repository**
   ```bash
   git clone https://github.com/erpgo/erpgo.git
   cd erpgo
   ```

2. **Start the development environment**
   ```bash
   docker-compose up -d
   ```

3. **Wait for services to be ready**
   ```bash
   docker-compose logs -f
   # Wait until you see "Database is ready to accept connections"
   ```

4. **Run database migrations**
   ```bash
   docker-compose exec erpgo-api go run cmd/migrate/main.go up
   ```

5. **Verify the setup**
   ```bash
   curl http://localhost:8080/health
   # Should return: {"status":"healthy","timestamp":"...","version":"..."}
   ```

### Option 2: Local Development Setup

1. **Install PostgreSQL**
   ```bash
   # macOS
   brew install postgresql
   brew services start postgresql

   # Ubuntu/Debian
   sudo apt-get install postgresql postgresql-contrib
   sudo systemctl start postgresql
   ```

2. **Install Redis**
   ```bash
   # macOS
   brew install redis
   brew services start redis

   # Ubuntu/Debian
   sudo apt-get install redis-server
   sudo systemctl start redis
   ```

3. **Create database**
   ```bash
   createdb erp
   createuser erpgo
   psql -c "ALTER USER erpgo WITH PASSWORD 'password';"
   psql -c "GRANT ALL PRIVILEGES ON DATABASE erp TO erpgo;"
   ```

4. **Install Go dependencies**
   ```bash
   go mod download
   go mod tidy
   ```

5. **Run database migrations**
   ```bash
   go run cmd/migrate/main.go up
   ```

6. **Start the application**
   ```bash
   go run cmd/api/main.go
   ```

## Project Structure

```
erpgo/
├── cmd/                        # Application entry points
│   ├── api/                   # Main API application
│   │   └── main.go
│   ├── migrate/               # Database migration tool
│   │   └── main.go
│   └── worker/                # Background job processor
│       └── main.go
├── internal/                   # Private application code
│   ├── application/           # Application services
│   │   ├── services/
│   │   │   ├── user/
│   │   │   ├── product/
│   │   │   ├── order/
│   │   │   └── inventory/
│   │   └── dto/               # Data transfer objects
│   ├── domain/                # Business entities and rules
│   │   ├── users/
│   │   │   ├── entities/
│   │   │   ├── repositories/
│   │   │   └── services/
│   │   ├── products/
│   │   ├── orders/
│   │   └── inventory/
│   ├── infrastructure/        # External dependencies
│   │   ├── database/
│   │   ├── cache/
│   │   ├── messaging/
│   │   └── storage/
│   └── interfaces/            # Interfaces and adapters
│       ├── http/
│       │   ├── handlers/
│       │   ├── middleware/
│       │   ├── routes/
│       │   └── dto/
│       └── grpc/
├── pkg/                       # Public library code
│   ├── auth/
│   ├── config/
│   ├── database/
│   ├── errors/
│   ├── logger/
│   └── utils/
├── migrations/                 # Database migration files
│   ├── 001_create_users_table.up.sql
│   ├── 001_create_users_table.down.sql
│   └── ...
├── configs/                    # Configuration files
│   ├── prometheus.yml
│   ├── grafana/
│   └── alertmanager.yml
├── docs/                      # Documentation
├── scripts/                   # Utility scripts
├── tests/                     # Test files
│   ├── unit/
│   ├── integration/
│   └── e2e/
├── docker-compose.yml         # Docker development environment
├── Dockerfile                 # Production Docker image
├── Makefile                   # Build automation
├── go.mod                     # Go module file
├── go.sum                     # Go module checksums
└── README.md                  # Project documentation
```

## Architecture Overview

ERPGo follows **Clean Architecture** principles with **Domain-Driven Design (DDD)**:

### Clean Architecture Layers

1. **Domain Layer** (`internal/domain/`)
   - Business entities and rules
   - Domain services
   - Repository interfaces
   - No external dependencies

2. **Application Layer** (`internal/application/`)
   - Use cases and business logic
   - Application services
   - DTOs for data transfer
   - Depends on domain layer

3. **Infrastructure Layer** (`internal/infrastructure/`)
   - External dependencies
   - Database implementations
   - Cache implementations
   - External API clients

4. **Interface Layer** (`internal/interfaces/`)
   - HTTP handlers and routes
   - Middleware
   - Request/response DTOs
   - External interfaces

### Key Design Patterns

- **Repository Pattern**: For data access abstraction
- **Service Layer Pattern**: For business logic encapsulation
- **Dependency Injection**: For loose coupling
- **Middleware Pattern**: For cross-cutting concerns
- **Factory Pattern**: For object creation

## Key Concepts

### Domain Entities

- **User**: System users with authentication and authorization
- **Product**: Items in the product catalog
- **Order**: Customer orders with items and fulfillment
- **Inventory**: Stock levels and warehouse management
- **Category**: Product categorization hierarchy

### Repositories

Repository interfaces are defined in the domain layer and implemented in the infrastructure layer:

```go
// internal/domain/users/repositories/user_repository.go
type Repository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id uuid.UUID) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, filter *Filter) ([]*User, error)
}

// internal/infrastructure/database/user_repository.go
type userRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) repositories.Repository {
    return &userRepository{db: db}
}
```

### Services

Application services orchestrate business operations:

```go
// internal/application/services/user/user_service.go
type Service struct {
    userRepo    repositories.Repository
    authSvc     auth.Service
    eventBus    events.Bus
    logger      logger.Logger
}

func (s *Service) CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error) {
    // Business logic implementation
}
```

### HTTP Handlers

HTTP handlers are thin and delegate to services:

```go
// internal/interfaces/http/handlers/user_handler.go
type Handler struct {
    userService user.Service
    logger      logger.Logger
}

func (h *Handler) CreateUser(c *gin.Context) {
    var req dto.CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, dto.ErrorResponse{
            Error:   "validation_error",
            Message: err.Error(),
        })
        return
    }

    user, err := h.userService.CreateUser(c.Request.Context(), &req)
    if err != nil {
        h.handleError(c, err)
        return
    }

    c.JSON(http.StatusCreated, user)
}
```

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/user-profile-enhancement
```

### 2. Make Your Changes

Follow the existing code patterns and conventions. Make sure to:

- Write tests for new functionality
- Update documentation if needed
- Follow the code style guidelines

### 3. Run Tests

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Run tests with coverage
make test-coverage
```

### 4. Lint and Format Code

```bash
# Format code
make format

# Run linter
make lint

# Run security scan
make security
```

### 5. Commit Your Changes

```bash
git add .
git commit -m "feat: add user profile enhancement

- Add profile picture upload
- Add bio field
- Update user validation
- Add corresponding tests

Closes #123"
```

### 6. Push and Create Pull Request

```bash
git push origin feature/user-profile-enhancement
```

Create a pull request following the template in `.github/PULL_REQUEST_TEMPLATE.md`.

## Running Tests

### Unit Tests

```bash
# Run all unit tests
go test ./tests/unit/...

# Run specific test file
go test ./tests/unit/domain/users/user_test.go

# Run with verbose output
go test -v ./tests/unit/...

# Run with coverage
go test -cover ./tests/unit/...
```

### Integration Tests

```bash
# Run integration tests (requires test database)
go test -tags=integration ./tests/integration/...

# Run specific integration test
go test -tags=integration ./tests/integration/api/user_test.go
```

### End-to-End Tests

```bash
# Run E2E tests (requires full environment)
go test ./tests/e2e/...

# Run with test environment
ENVIRONMENT=test go test ./tests/e2e/...
```

### Test Database Setup

Integration tests use a separate test database:

```bash
# Create test database
createdb erp_test

# Run migrations on test database
DATABASE_URL=postgres://erpgo:password@localhost:5432/erp_test go run cmd/migrate/main.go up
```

## Debugging

### Using Delve (Go Debugger)

1. **Install Delve**
   ```bash
   go install github.com/go-delve/delve/cmd/dlv@latest
   ```

2. **Debug the main application**
   ```bash
   dlv debug cmd/api/main.go
   ```

3. **Debug tests**
   ```bash
   dlv test ./tests/unit/...
   ```

### Common Debugging Commands

```bash
# Set breakpoint
(break main.go:10)

# Continue execution
(c)

# Step over
(n)

# Step into
(s)

# Print variable
(p variableName)

# List goroutines
(goroutines)

# Switch goroutine
(goroutine <id>)
```

### VS Code Debugging

Create `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch ERPGo API",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/api/main.go",
            "env": {
                "ENVIRONMENT": "development",
                "LOG_LEVEL": "debug"
            }
        },
        {
            "name": "Launch Tests",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${workspaceFolder}/tests/unit/..."
        }
    ]
}
```

## Code Style and Standards

### Go Formatting

Always use `gofmt` and `goimports`:

```bash
gofmt -s -w .
goimports -w .
```

### Naming Conventions

- **Package names**: lowercase, short, descriptive
- **Constants**: UPPER_SNAKE_CASE
- **Variables**: camelCase
- **Functions**: camelCase, exported if public
- **Types**: PascalCase, exported if public
- **Files**: lowercase with underscores

### Code Organization

```go
package user

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "github.com/rs/zerolog"

    "erpgo/internal/domain/users/entities"
    "erpgo/pkg/errors"
)

// Service handles user-related business logic
type Service struct {
    repo   Repository
    logger zerolog.Logger
}

// NewService creates a new user service
func NewService(repo Repository, logger zerolog.Logger) *Service {
    return &Service{
        repo:   repo,
        logger: logger,
    }
}

// CreateUser creates a new user with validation
func (s *Service) CreateUser(ctx context.Context, req *CreateUserRequest) (*entities.User, error) {
    if err := req.Validate(); err != nil {
        return nil, errors.ValidationError(err)
    }

    // Business logic here

    return user, nil
}
```

### Error Handling

Use structured error handling with custom error types:

```go
// pkg/errors/errors.go
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

// Usage
if req.Email == "" {
    return nil, &ValidationError{
        Field:   "email",
        Message: "email is required",
    }
}
```

### Logging

Use structured logging with zerolog:

```go
logger.Info().
    Str("user_id", user.ID.String()).
    Str("action", "user_created").
    Msg("User created successfully")

logger.Error().
    Err(err).
    Str("user_id", user.ID.String()).
    Msg("Failed to create user")
```

## Common Development Tasks

### Adding a New API Endpoint

1. **Define the route** in `internal/interfaces/http/routes/`
2. **Create the handler** in `internal/interfaces/http/handlers/`
3. **Create DTOs** in `internal/interfaces/http/dto/`
4. **Implement service logic** in `internal/application/services/`
5. **Add tests** for the handler and service
6. **Update OpenAPI documentation**

### Adding a New Entity

1. **Define entity** in `internal/domain/{entity}/entities/`
2. **Create repository interface** in `internal/domain/{entity}/repositories/`
3. **Implement repository** in `internal/infrastructure/database/`
4. **Create service** in `internal/application/services/{entity}/`
5. **Add migration** in `migrations/`
6. **Write tests** for all layers

### Adding Database Migration

1. **Create migration files**:
   ```bash
   # Up migration
   echo "CREATE TABLE new_table (...);" > migrations/015_create_new_table.up.sql

   # Down migration
   echo "DROP TABLE new_table;" > migrations/015_create_new_table.down.sql
   ```

2. **Run migration**:
   ```bash
   go run cmd/migrate/main.go up
   ```

3. **Generate entity models** if using ORM:
   ```bash
   go run cmd/generate/main.go
   ```

### Adding Tests

1. **Unit test example**:
   ```go
   func TestUserService_CreateUser(t *testing.T) {
       tests := []struct {
           name    string
           req     *dto.CreateUserRequest
           want    *entities.User
           wantErr bool
           errType error
       }{
           {
               name: "valid user creation",
               req: &dto.CreateUserRequest{
                   Email:    "test@example.com",
                   Username: "testuser",
                   Password: "password123",
               },
               want: &entities.User{
                   Email:    "test@example.com",
                   Username: "testuser",
               },
               wantErr: false,
           },
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // Setup test
               repo := &mocks.UserRepository{}
               service := NewService(repo, logger)

               // Mock repository calls
               repo.On("Create", mock.Anything, mock.AnythingOfType("*entities.User")).Return(nil)

               // Execute
               got, err := service.CreateUser(context.Background(), tt.req)

               // Assert
               if (err != nil) != tt.wantErr {
                   t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
                   return
               }

               if !reflect.DeepEqual(got, tt.want) {
                   t.Errorf("CreateUser() = %v, want %v", got, tt.want)
               }
           })
       }
   }
   ```

2. **Integration test example**:
   ```go
   func TestUserHandler_CreateUser_Integration(t *testing.T) {
       // Setup test database
       db := setupTestDB(t)
       defer cleanupTestDB(t, db)

       // Setup application
       app := setupTestApp(t, db)

       // Test request
       reqBody := `{
           "email": "test@example.com",
           "username": "testuser",
           "password": "password123",
           "first_name": "Test",
           "last_name": "User"
       }`

       req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(reqBody))
       req.Header.Set("Content-Type", "application/json")
       w := httptest.NewRecorder()

       // Execute
       app.ServeHTTP(w, req)

       // Assert
       assert.Equal(t, http.StatusCreated, w.Code)

       var response dto.UserResponse
       err := json.Unmarshal(w.Body.Bytes(), &response)
       assert.NoError(t, err)
       assert.Equal(t, "test@example.com", response.Email)
   }
   ```

## Troubleshooting

### Common Issues

1. **Database Connection Issues**
   ```bash
   # Check PostgreSQL status
   brew services list | grep postgresql
   sudo systemctl status postgresql

   # Test connection
   psql -h localhost -U erpgo -d erp
   ```

2. **Redis Connection Issues**
   ```bash
   # Check Redis status
   redis-cli ping

   # Check Redis configuration
   redis-cli config get "*"
   ```

3. **Port Conflicts**
   ```bash
   # Check what's using port 8080
   lsof -i :8080

   # Kill process
   kill -9 <PID>
   ```

4. **Go Module Issues**
   ```bash
   # Clean module cache
   go clean -modcache

   # Re-download dependencies
   go mod download
   go mod tidy
   ```

### Debug Tips

1. **Enable debug logging**:
   ```env
   LOG_LEVEL=debug
   DEBUG_MODE=true
   ```

2. **Use database logging**:
   ```sql
   -- Enable query logging in PostgreSQL
   ALTER SYSTEM SET log_statement = 'all';
   SELECT pg_reload_conf();
   ```

3. **Check application logs**:
   ```bash
   # Docker logs
   docker-compose logs -f erpgo-api

   # Local logs
   tail -f logs/app.log
   ```

### Performance Issues

1. **Profile the application**:
   ```bash
   # CPU profiling
   go tool pprof http://localhost:8080/debug/pprof/profile

   # Memory profiling
   go tool pprof http://localhost:8080/debug/pprof/heap
   ```

2. **Check database queries**:
   ```sql
   -- Slow queries
   SELECT query, mean_time, calls
   FROM pg_stat_statements
   ORDER BY mean_time DESC
   LIMIT 10;
   ```

## Resources and References

### Documentation

- [API Documentation](./API_DOCUMENTATION.md)
- [Deployment Guide](./DEPLOYMENT_GUIDE.md)
- [Architecture Overview](./ARCHITECTURE_OVERVIEW.md)
- [OpenAPI Specification](./openapi.yaml)

### Tools and Libraries

- [Gin Web Framework](https://gin-gonic.com/)
- [GORM ORM](https://gorm.io/)
- [Zerolog Logger](https://github.com/rs/zerolog)
- [Testify Testing Framework](https://github.com/stretchr/testify)
- [Mockery Mock Generation](https://github.com/vektra/mockery)

### Go Resources

- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go by Example](https://gobyexample.com/)

### Community

- [GitHub Repository](https://github.com/erpgo/erpgo)
- [Issues and Discussions](https://github.com/erpgo/erpgo/issues)
- [Slack Channel](https://erpgo.slack.com)

## Getting Help

If you're stuck or have questions:

1. Check existing [GitHub Issues](https://github.com/erpgo/erpgo/issues)
2. Search the [documentation](./)
3. Ask in the [Slack channel](https://erpgo.slack.com)
4. Create a new issue with the `question` label

Happy coding! 🚀