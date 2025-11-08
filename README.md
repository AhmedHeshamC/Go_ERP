# ERPGo API

A comprehensive Enterprise Resource Planning (ERP) backend API built with Go and PostgreSQL, providing user management, inventory management, and role-based access control.

## 📄 License

This project is licensed under the **GNU Affero General Public License v3.0 (AGPL-3.0)**.

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 12 or higher
- Redis 6 or higher (optional, for caching)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/erpgo/erpgo.git
   cd erpgo
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

4. **Run database migrations**
   ```bash
   # Migration files are located in the migrations/ directory
   # Run them against your PostgreSQL database
   ```

5. **Start the server**
   ```bash
   go run cmd/api/main.go
   ```

The API will be available at `http://localhost:8080`

## 📚 API Documentation

### Base URL
- **Production**: `https://your-domain.com`
- **Development**: `http://localhost:8080`

### API Version
Current version: `v1`

### Content-Type
All API requests must include the `Content-Type: application/json` header.

### Authentication
The API uses JWT (JSON Web Tokens) for authentication. Include the token in the `Authorization` header:

```http
Authorization: Bearer <your-jwt-token>
```

## 🔐 Authentication

### Login
```http
POST /api/v1/auth/login
```

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "your-password"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "username": "username",
      "first_name": "John",
      "last_name": "Doe",
      "is_active": true,
      "is_verified": true,
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    },
    "access_token": "jwt-access-token",
    "refresh_token": "jwt-refresh-token",
    "expires_in": 86400
  }
}
```

### Refresh Token
```http
POST /api/v1/auth/refresh
```

**Request Body:**
```json
{
  "refresh_token": "your-refresh-token"
}
```

### Logout
```http
POST /api/v1/auth/logout
```

**Headers:**
```
Authorization: Bearer <access-token>
```

## 👥 User Management

### Create User
```http
POST /api/v1/users
```

**Request Body:**
```json
{
  "email": "newuser@example.com",
  "username": "newuser",
  "password": "password123",
  "first_name": "John",
  "last_name": "Doe",
  "phone": "+1234567890"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "email": "newuser@example.com",
    "username": "newuser",
    "first_name": "John",
    "last_name": "Doe",
    "phone": "+1234567890",
    "is_active": true,
    "is_verified": false,
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z"
  }
}
```

### Get User
```http
GET /api/v1/users/{id}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "username": "username",
    "first_name": "John",
    "last_name": "Doe",
    "phone": "+1234567890",
    "is_active": true,
    "is_verified": true,
    "last_login_at": "2023-01-01T12:00:00Z",
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z"
  }
}
```

### Update User
```http
PUT /api/v1/users/{id}
```

**Request Body:**
```json
{
  "first_name": "Updated",
  "last_name": "Name",
  "phone": "+9876543210",
  "is_active": true
}
```

### List Users
```http
GET /api/v1/users?page=1&limit=10&search=john&is_active=true&sort_by=created_at&sort_order=desc
```

**Query Parameters:**
- `page` (int): Page number (default: 1)
- `limit` (int): Items per page (default: 10, max: 100)
- `search` (string): Search by username, email, or name
- `is_active` (boolean): Filter by active status
- `is_verified` (boolean): Filter by verification status
- `role_id` (string): Filter by role ID
- `sort_by` (string): Sort field (e.g., created_at, username, email)
- `sort_order` (string): Sort order (asc, desc)

**Response:**
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "uuid",
        "email": "user@example.com",
        "username": "username",
        "first_name": "John",
        "last_name": "Doe",
        "is_active": true,
        "created_at": "2023-01-01T00:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 25,
      "total_pages": 3,
      "has_next": true,
      "has_prev": false
    }
  }
}
```

### Delete User
```http
DELETE /api/v1/users/{id}
```

**Response:** `204 No Content`

## 🔐 Role Management

### Create Role
```http
POST /api/v1/roles
```

**Request Body:**
```json
{
  "name": "Manager",
  "description": "Store manager with full access",
  "permissions": [
    "users:read",
    "users:write",
    "products:read",
    "products:write",
    "inventory:read",
    "inventory:write"
  ]
}
```

### List Roles
```http
GET /api/v1/roles?page=1&limit=10&search=manager
```

## 📦 Product Management

### Create Product
```http
POST /api/v1/products
```

**Request Body:**
```json
{
  "sku": "PROD-001",
  "name": "Product Name",
  "description": "Product description",
  "category_id": "uuid",
  "price": 99.99,
  "cost": 50.00,
  "weight": 1.5,
  "dimensions": "10x5x3"
}
```

### Get Product
```http
GET /api/v1/products/{id}
```

### Update Product
```http
PUT /api/v1/products/{id}
```

### List Products
```http
GET /api/v1/products?page=1&limit=10&search=product&category_id=uuid&is_active=true
```

## 📊 Inventory Management

### Get Inventory Levels
```http
GET /api/v1/inventory?warehouse_id=uuid&product_id=uuid
```

### Update Inventory
```http
POST /api/v1/inventory/adjust
```

**Request Body:**
```json
{
  "product_id": "uuid",
  "warehouse_id": "uuid",
  "quantity": 10,
  "transaction_type": "IN",
  "reason": "Stock replenishment"
}
```

## 🏭 Product Categories

### Create Category
```http
POST /api/v1/categories
```

**Request Body:**
```json
{
  "name": "Electronics",
  "description": "Electronic devices and accessories",
  "parent_id": null
}
```

### List Categories
```http
GET /api/v1/categories?parent_id=null&is_active=true
```

## 📍 Warehouses

### Create Warehouse
```http
POST /api/v1/warehouses
```

**Request Body:**
```json
{
  "name": "Main Warehouse",
  "code": "WH001",
  "address": "123 Storage St",
  "city": "New York",
  "state": "NY",
  "country": "USA",
  "postal_code": "10001"
}
```

## 📋 Error Responses

All API endpoints return errors in a consistent format:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request data",
    "details": [
      {
        "field": "email",
        "message": "Invalid email format"
      }
    ]
  }
}
```

### Common HTTP Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `204 No Content`: Request successful, no content returned
- `400 Bad Request`: Invalid request data
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource already exists
- `422 Unprocessable Entity`: Validation failed
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error

## 🔧 Rate Limiting

The API implements rate limiting to protect against abuse:
- **Default**: 100 requests per second
- **Burst**: 200 requests
- **Window**: 1 hour

Rate limit headers are included in responses:
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 99
X-RateLimit-Reset: 1640995200
```

## 🔒 Security Features

- JWT-based authentication
- Password hashing with bcrypt
- Rate limiting
- CORS protection
- SQL injection prevention
- Input validation
- Secure headers

## 📊 Monitoring & Metrics

### Health Check
```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2023-01-01T12:00:00Z",
  "version": "v1.0.0"
}
```

### Metrics (if enabled)
```http
GET /metrics
```

Prometheus-compatible metrics endpoint.

## 🛠️ Development

### Environment Variables

Create a `.env` file with the following variables:

```env
# Server
SERVER_PORT=8080
ENVIRONMENT=development
LOG_LEVEL=info

# Database
DATABASE_URL=postgres://user:password@localhost/erp?sslmode=disable

# Redis (optional)
REDIS_URL=redis://localhost:6379

# JWT
JWT_SECRET=your-super-secret-jwt-key
JWT_EXPIRY=24h
REFRESH_EXPIRY=168h

# CORS
CORS_ORIGINS=http://localhost:3000,http://localhost:8080

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=100
RATE_LIMIT_BURST=200

# API Documentation
API_DOCS_ENABLED=true
API_DOCS_PATH=/docs
```

### Running Tests

```bash
# Run unit tests
go test ./tests/unit/...

# Run integration tests
go test -tags=integration ./tests/integration/...

# Run E2E tests
go test ./tests/e2e/...

# Run all tests
go test ./...
```

### API Documentation

When `API_DOCS_ENABLED=true`, Swagger UI is available at:
- Development: `http://localhost:8080/docs`
- Production: `https://your-domain.com/docs`

## 🏗️ Architecture

The ERPGo API follows a clean architecture pattern:

```
├── cmd/api/              # Application entry point
├── internal/
│   ├── domain/          # Business entities and rules
│   │   ├── users/
│   │   ├── products/
│   │   └── inventory/
│   ├── application/     # Application services
│   ├── infrastructure/  # External dependencies
│   └── interfaces/      # HTTP handlers and middleware
├── pkg/                 # Shared packages
├── migrations/          # Database migrations
└── tests/              # Test files
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## 📞 Support

For support and questions:
- Create an issue on GitHub
- Check the API documentation at `/docs`
- Review the error messages carefully

## 🚀 Deployment

### Docker

```bash
# Build the image
docker build -t erpgo-api .

# Run the container
docker run -p 8080:8080 --env-file .env erpgo-api
```

### Production Considerations

- Use HTTPS in production
- Set strong JWT secrets
- Configure proper CORS origins
- Enable rate limiting
- Set up monitoring and logging
- Use environment-specific configurations
- Regularly update dependencies

---

**Note:** This API is under active development. Features and endpoints may change. Always check the latest documentation for the most up-to-date information.