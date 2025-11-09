# ERPGo Developer Guide

## Table of Contents
1. [Introduction](#introduction)
2. [Project Structure](#project-structure)
3. [Getting Started](#getting-started)
4. [API Documentation](#api-documentation)
5. [Database Setup](#database-setup)
6. [Authentication & Authorization](#authentication--authorization)
7. [Testing](#testing)
8. [Development Workflow](#development-workflow)
9. [Code Standards](#code-standards)
10. [Troubleshooting](#troubleshooting)

## Introduction

ERPGo is a comprehensive Enterprise Resource Planning system built with Go, designed to provide robust business management capabilities. This guide will help developers get started with contributing to and extending the ERPGo platform.

### Key Features
- **User Management**: Complete user authentication and role-based access control
- **Product Management**: Product catalog with categories and variants
- **Inventory Management**: Stock tracking, transfers, and adjustments
- **Order Management**: Order processing and fulfillment
- **Customer Management**: Customer data and relationship management
- **Security**: JWT-based authentication, rate limiting, and security middleware

### Technology Stack
- **Backend**: Go 1.24+, Gin Framework
- **Database**: PostgreSQL
- **Cache**: Redis
- **Authentication**: JWT tokens
- **Documentation**: Swagger/OpenAPI
- **Monitoring**: Prometheus metrics
- **Containerization**: Docker & Docker Compose

## Project Structure

```
Go_ERP/
├── cmd/api/                 # Application entry point
├── internal/                # Private application code
│   ├── application/         # Application services
│   │   └── services/        # Business logic services
│   ├── domain/             # Domain models and entities
│   │   ├── users/          # User domain
│   │   ├── products/       # Product domain
│   │   ├── inventory/      # Inventory domain
│   │   └── orders/         # Order domain
│   ├── infrastructure/     # External interfaces
│   │   └── repositories/   # Data access layer
│   └── interfaces/         # Interface adapters
│       └── http/          # HTTP handlers and routes
├── pkg/                    # Reusable packages
│   ├── auth/              # Authentication utilities
│   ├── cache/             # Cache implementation
│   ├── config/            # Configuration management
│   ├── database/          # Database connection
│   ├── email/             # Email services
│   └── logger/            # Logging utilities
├── docs/                   # Documentation
├── migrations/            # Database migrations
├── tests/                 # Test files
└── scripts/              # Utility scripts
```

## Getting Started

### Prerequisites

- Go 1.24 or higher
- PostgreSQL 13 or higher
- Redis 6 or higher
- Docker (optional, for containerized setup)
- Git

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd Go_ERP
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Set up the database**
   ```bash
   # Create database
   createdb erpgo_dev

   # Run migrations
   migrate -path migrations -database "postgres://user:password@localhost/erpgo_dev?sslmode=disable" up
   ```

5. **Start Redis server**
   ```bash
   redis-server
   ```

6. **Install Swag for API documentation**
   ```bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ```

7. **Generate Swagger documentation**
   ```bash
   swag init -g cmd/api/main.go --output docs
   ```

8. **Run the application**
   ```bash
   go run cmd/api/main.go
   ```

The application will start on `http://localhost:8080` by default.

### Docker Setup

For a quick development setup using Docker:

```bash
# Start all services
docker-compose up -d

# Run migrations
docker-compose exec api migrate -path /app/migrations -database $DATABASE_URL up

# Generate Swagger docs
docker-compose exec api swag init -g cmd/api/main.go --output /app/docs
```

## API Documentation

### Interactive Documentation

Once the application is running, you can access the interactive API documentation at:

- **Swagger UI**: `http://localhost:8080/docs/index.html`
- **OpenAPI JSON**: `http://localhost:8080/docs/swagger.json`
- **OpenAPI YAML**: `http://localhost:8080/docs/swagger.yaml`

### API Endpoints Overview

#### Authentication
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/refresh` - Refresh JWT token
- `POST /api/v1/auth/logout` - User logout

#### Users
- `GET /api/v1/users` - List users (paginated)
- `GET /api/v1/users/{id}` - Get user by ID
- `PUT /api/v1/users/{id}` - Update user
- `DELETE /api/v1/users/{id}` - Delete user (soft delete)
- `GET /api/v1/profile` - Get current user profile
- `PUT /api/v1/profile` - Update current user profile

#### Products
- `POST /api/v1/products` - Create product
- `GET /api/v1/products` - List products
- `GET /api/v1/products/{id}` - Get product by ID
- `PUT /api/v1/products/{id}` - Update product
- `DELETE /api/v1/products/{id}` - Delete product

#### Categories
- `POST /api/v1/categories` - Create category
- `GET /api/v1/categories` - List categories
- `PUT /api/v1/categories/{id}` - Update category
- `DELETE /api/v1/categories/{id}` - Delete category

#### Inventory
- `POST /api/v1/inventory/adjust` - Adjust inventory
- `POST /api/v1/inventory/reserve` - Reserve inventory
- `POST /api/v1/inventory/release` - Release inventory
- `POST /api/v1/inventory/transfer` - Transfer inventory
- `GET /api/v1/inventory` - List inventory items

#### Warehouses
- `POST /api/v1/warehouses` - Create warehouse
- `GET /api/v1/warehouses` - List warehouses
- `PUT /api/v1/warehouses/{id}` - Update warehouse

### Authentication

The API uses JWT (JSON Web Tokens) for authentication. Include the token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

## Database Setup

### Database Migrations

Migrations are located in the `migrations/` directory. To run migrations:

```bash
# Run all migrations
migrate -path migrations -database "postgres://user:password@localhost/erpgo_dev?sslmode=disable" up

# Rollback migration
migrate -path migrations -database "postgres://user:password@localhost/erpgo_dev?sslmode=disable" down

# Create new migration
migrate create -ext sql -dir migrations -seq create_new_table
```

### Database Schema

The database includes the following main tables:
- `users` - User accounts and profiles
- `roles` - User roles and permissions
- `user_roles` - User-role relationships
- `products` - Product catalog
- `product_categories` - Product categories
- `product_variants` - Product variants
- `warehouses` - Warehouse locations
- `inventory` - Stock levels
- `inventory_transactions` - Inventory movement history
- `orders` - Customer orders
- `order_items` - Order line items
- `customers` - Customer information

## Authentication & Authorization

### JWT Token Structure

Access tokens contain the following claims:
- `sub`: User ID
- `email`: User email
- `roles`: Array of user roles
- `exp`: Token expiration time
- `iat`: Token issued at time

### Role-Based Access Control (RBAC)

The system supports role-based access control with the following default roles:
- `admin`: Full system access
- `manager`: Business operations access
- `operator`: Day-to-day operations
- `viewer`: Read-only access

### Authentication Flow

1. **Login**: User provides credentials to `/api/v1/auth/login`
2. **Token Validation**: JWT tokens are validated on protected routes
3. **Token Refresh**: Use `/api/v1/auth/refresh` to get new tokens
4. **Logout**: Token is invalidated via `/api/v1/auth/logout`

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test package
go test ./tests/unit/domain/users/

# Run integration tests
go test ./tests/integration/

# Run end-to-end tests
go test ./tests/e2e/
```

### Test Structure

```
tests/
├── unit/           # Unit tests for individual components
├── integration/    # Integration tests for API endpoints
├── e2e/           # End-to-end tests for complete workflows
├── performance/   # Performance and load tests
└── security/      # Security-related tests
```

### Test Database

Tests use a separate database to avoid affecting development data:

```bash
# Create test database
createdb erpgo_test

# Run migrations on test database
DATABASE_URL="postgres://user:password@localhost/erpgo_test?sslmode=disable" migrate up
```

## Development Workflow

### Code Organization

Follow the Clean Architecture principles:
- **Domain Layer**: Business entities and rules
- **Application Layer**: Use cases and application services
- **Infrastructure Layer**: External concerns (database, APIs)
- **Interface Layer**: Controllers and presenters

### Adding New Features

1. **Define Domain Model**
   - Create entity in `internal/domain/{feature}/entities/`
   - Define repository interface

2. **Implement Infrastructure**
   - Create repository implementation in `internal/infrastructure/repositories/`
   - Add database migrations if needed

3. **Create Application Service**
   - Implement business logic in `internal/application/services/{feature}/`
   - Define service interface

4. **Add HTTP Handler**
   - Create handler in `internal/interfaces/http/handlers/`
   - Add routes in `internal/interfaces/http/routes/`

5. **Add Tests**
   - Unit tests for service layer
   - Integration tests for API endpoints
   - Update documentation

### Git Workflow

1. Create feature branch from `develop`
2. Make changes with atomic commits
3. Add tests for new functionality
4. Update documentation
5. Create pull request with:
   - Clear description of changes
   - Test results
   - Updated documentation

## Code Standards

### Go Conventions

Follow the official Go conventions and best practices:
- Use `gofmt` for code formatting
- Use `golint` for style checking
- Follow the naming conventions from Effective Go
- Keep functions small and focused
- Use interfaces for dependency injection

### Error Handling

- Use explicit error handling
- Wrap errors with context using `fmt.Errorf`
- Define custom error types for domain-specific errors
- Log errors appropriately

### API Standards

- Use RESTful principles
- Consistent response formats
- Proper HTTP status codes
- Input validation and sanitization
- Rate limiting for all endpoints

### Security Standards

- Input validation on all endpoints
- SQL injection prevention
- XSS protection
- Authentication and authorization
- Secure headers
- Rate limiting

## Troubleshooting

### Common Issues

#### Database Connection Issues
```bash
# Check PostgreSQL is running
pg_isready -h localhost -p 5432

# Check connection string
psql "postgres://user:password@localhost/erpgo_dev?sslmode=disable"
```

#### Redis Connection Issues
```bash
# Check Redis is running
redis-cli ping

# Check Redis logs
tail -f /usr/local/var/log/redis.log
```

#### Compilation Errors
```bash
# Clean Go cache
go clean -modcache

# Re-download dependencies
go mod download

# Update dependencies
go mod tidy
```

#### Migration Issues
```bash
# Check migration status
migrate -path migrations -database $DATABASE_URL version

# Force migration version
migrate -path migrations -database $DATABASE_URL force 1
```

### Debug Mode

Enable debug mode by setting:
```bash
export DEBUG=true
export LOG_LEVEL=debug
```

### Performance Issues

For performance investigation:
1. Check database query performance
2. Monitor Redis usage
3. Review application logs
4. Use profiling tools:
   ```bash
   go tool pprof http://localhost:8080/debug/pprof/profile
   ```

### Getting Help

- Check the application logs
- Review the test cases for examples
- Consult the API documentation
- Create an issue with detailed information

---

This guide is continuously updated. For the most recent information, check the project documentation and README files.