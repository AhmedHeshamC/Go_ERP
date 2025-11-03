# ERPGo - Comprehensive ERP System

A modern, scalable Enterprise Resource Planning (ERP) system built with Go, following clean architecture principles and best practices.

## 🚀 Features

- **User Management**: Authentication, authorization, and role-based access control (RBAC)
- **Inventory Management**: Product catalog, stock tracking, warehouse management
- **Order Management**: Complete order lifecycle from quote to delivery
- **Finance & Accounting**: Invoicing, expenses, financial reporting
- **Procurement**: Vendor management, purchase orders, supply chain
- **HR Management**: Employee records, payroll, leave management
- **Reporting & Analytics**: Business intelligence and custom reports
- **Real-time Notifications**: Multi-channel notification system

## 🏗️ Architecture

This project follows **Clean Architecture** principles with clear separation of concerns:

```
├── cmd/                    # Application entry points
│   ├── api/               # HTTP API server
│   ├── migrator/          # Database migration tool
│   └── worker/            # Background worker
├── internal/
│   ├── domain/            # Business entities and rules
│   │   ├── users/         # User domain
│   │   ├── products/      # Product domain
│   │   ├── orders/        # Order domain
│   │   ├── inventory/     # Inventory domain
│   │   ├── finance/       # Finance domain
│   │   ├── procurement/   # Procurement domain
│   │   ├── hr/           # HR domain
│   │   └── reports/      # Reports domain
│   ├── application/       # Use cases and application logic
│   │   ├── services/      # Application services
│   │   └── usecases/      # Business use cases
│   ├── infrastructure/    # External concerns
│   │   ├── database/      # Database implementations
│   │   ├── cache/         # Cache implementations
│   │   ├── messaging/     # Message queue implementations
│   │   └── storage/       # File storage implementations
│   └── interfaces/        # Controllers and interfaces
│       ├── http/          # HTTP handlers
│       ├── grpc/          # gRPC handlers
│       └── events/        # Event handlers
├── pkg/                   # Shared libraries
│   ├── auth/             # Authentication utilities
│   ├── cache/            # Cache utilities
│   ├── config/           # Configuration management
│   ├── database/         # Database utilities
│   ├── logger/           # Logging utilities
│   ├── middleware/       # HTTP middleware
│   ├── validator/        # Validation utilities
│   └── utils/            # General utilities
├── configs/              # Configuration files
├── migrations/           # Database migrations
├── tests/               # Test files
│   ├── unit/            # Unit tests
│   ├── integration/     # Integration tests
│   └── e2e/             # End-to-end tests
└── docs/                # Documentation
```

## 🛠️ Technology Stack

- **Backend**: Go 1.21+ with Gin framework
- **Database**: PostgreSQL 15+ with connection pooling
- **Cache**: Redis 7+ with clustering support
- **Message Queue**: NATS for event-driven architecture
- **Authentication**: JWT with refresh tokens
- **Containerization**: Docker + Docker Compose
- **CI/CD**: GitHub Actions
- **Monitoring**: Prometheus + Grafana
- **Logging**: Structured logging with ELK stack

## 📋 Requirements

- Go 1.21 or higher
- Docker and Docker Compose
- PostgreSQL 15+
- Redis 7+
- Make (optional, for using Makefile commands)

## 🚀 Quick Start

### Using Docker Compose (Recommended)

1. **Clone the repository**
   ```bash
   git clone https://github.com/your-org/erpgo.git
   cd erpgo
   ```

2. **Start all services**
   ```bash
   docker-compose up -d
   ```

3. **Run database migrations**
   ```bash
   docker-compose --profile migration up migrator
   ```

4. **Access the API**
   ```
   API: http://localhost:8080
   Health Check: http://localhost:8080/health
   API Documentation: http://localhost:8080/docs
   ```

### Local Development

1. **Install dependencies**
   ```bash
   go mod download
   ```

2. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Start services with Docker Compose**
   ```bash
   docker-compose -f docker-compose.dev.yml up -d postgres redis
   ```

4. **Run database migrations**
   ```bash
   go run cmd/migrator/main.go up
   ```

5. **Start the API server**
   ```bash
   go run cmd/api/main.go
   ```

## 🛠️ Development

### Using Makefile Commands

```bash
# Install development tools
make install-tools

# Run linter
make lint

# Run tests
make test

# Run tests with coverage
make test-coverage

# Build the application
make build

# Run with hot reload
make dev

# Generate code (mocks, etc.)
make generate

# Clean up
make clean
```

### Running Tests

```bash
# Run all tests
make test

# Run unit tests only
go test -v ./...

# Run integration tests
go test -v -tags=integration ./tests/integration/...

# Run end-to-end tests
go test -v -tags=e2e ./tests/e2e/...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Database Migrations

```bash
# Run migrations
go run cmd/migrator/main.go up

# Create new migration
go run cmd/migrator/main.go create migration_name

# Rollback migrations
go run cmd/migrator/main.go down

# Get migration status
go run cmd/migrator/main.go status
```

## 📊 Monitoring

### Health Checks

- **API Health**: `GET /health`
- **Database Health**: `GET /health/db`
- **Redis Health**: `GET /health/redis`
- **Dependencies Health**: `GET /health/deps`

### Metrics

- **Prometheus Metrics**: `GET /metrics`
- **Grafana Dashboard**: Available when monitoring profile is enabled

### Logs

The application uses structured logging with multiple levels:

```bash
# View logs with Docker Compose
docker-compose logs -f api

# View logs locally
tail -f logs/app.log
```

## 🔧 Configuration

The application can be configured using environment variables. See `.env.example` for all available options.

### Key Configuration Options

```bash
# Server
SERVER_PORT=8080
ENVIRONMENT=development
LOG_LEVEL=debug

# Database
DATABASE_URL=postgres://user:password@localhost:5432/erp?sslmode=disable
MAX_CONNECTIONS=20

# Redis
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=

# JWT
JWT_SECRET=your-super-secret-jwt-key
JWT_EXPIRY=24h

# CORS
CORS_ORIGINS=http://localhost:3000,http://localhost:8080
```

## 🧪 Testing Strategy

This project follows the **Test Pyramid** approach:

- **70% Unit Tests**: Test individual functions and methods in isolation
- **20% Integration Tests**: Test interactions between components
- **10% End-to-End Tests**: Test complete user workflows

### Test Structure

```
tests/
├── unit/                   # Unit tests
│   ├── domain/
│   ├── application/
│   └── infrastructure/
├── integration/           # Integration tests
│   ├── api/
│   ├── database/
│   └── cache/
└── e2e/                  # End-to-end tests
    ├── workflows/
    └── scenarios/
```

## 🚀 Deployment

### Docker Deployment

1. **Build the image**
   ```bash
   docker build -t erpgo:latest .
   ```

2. **Run with Docker Compose**
   ```bash
   docker-compose -f docker-compose.yml up -d
   ```

### Kubernetes Deployment

```bash
# Apply configurations
kubectl apply -f k8s/

# Check deployment status
kubectl get pods -n erpgo
```

## 📝 API Documentation

### Authentication

All API endpoints (except authentication endpoints) require a valid JWT token.

```bash
# Login
POST /auth/login
{
  "email": "user@example.com",
  "password": "password"
}

# Get profile
GET /auth/profile
Authorization: Bearer <token>
```

### Example API Usage

```bash
# Create user
POST /api/v1/users
Authorization: Bearer <token>
{
  "email": "newuser@example.com",
  "username": "newuser",
  "password": "password",
  "first_name": "John",
  "last_name": "Doe"
}

# List products
GET /api/v1/products?page=1&limit=20
Authorization: Bearer <token>

# Create order
POST /api/v1/orders
Authorization: Bearer <token>
{
  "customer_id": "uuid",
  "items": [
    {
      "product_id": "uuid",
      "quantity": 2
    }
  ]
}
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and idiomatic code
- Write tests for new functionality
- Ensure all tests pass before submitting PR
- Follow the existing code style
- Update documentation as needed

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- 📧 Email: support@yourdomain.com
- 💬 Discord: [Join our Discord](https://discord.gg/your-invite)
- 📖 Documentation: [docs.erpgo.com](https://docs.erpgo.com)
- 🐛 Issues: [GitHub Issues](https://github.com/your-org/erpgo/issues)

## 🗺️ Roadmap

### Phase 1: Foundation (Weeks 1-2)
- [x] Project setup and structure
- [ ] Database design and migrations
- [ ] Authentication and authorization

### Phase 2: Core Services (Weeks 3-6)
- [ ] User management service
- [ ] Product management service
- [ ] Inventory management service

### Phase 3: Business Operations (Weeks 7-10)
- [ ] Order management service
- [ ] Customer management service
- [ ] Invoice and payment service

### Phase 4: Advanced Features (Weeks 11-14)
- [ ] Reporting and analytics
- [ ] Notification service
- [ ] API gateway and rate limiting

### Phase 5: Performance and Optimization (Weeks 15-16)
- [ ] Performance optimization
- [ ] Monitoring and alerting
- [ ] Security hardening

## 🙏 Acknowledgments

- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) by Robert C. Martin
- [Gin](https://gin-gonic.com/) Web Framework
- [PostgreSQL](https://www.postgresql.org/) Database
- [Redis](https://redis.io/) Caching
- All our contributors and users!