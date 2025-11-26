# ERPGo API Documentation

Welcome to the ERPGo API documentation. This directory contains comprehensive documentation for the ERPGo REST API.

## Documentation Files

- **[API Documentation](API_DOCUMENTATION.md)** - Complete API reference with all endpoints
- **[Error Codes](ERROR_CODES.md)** - Comprehensive error code reference
- **[API Examples](API_EXAMPLES.md)** - Practical examples for common operations
- **[Authentication Guide](AUTHENTICATION.md)** - Authentication and authorization details
- **[OpenAPI Specification](openapi.yaml)** - Machine-readable API specification

## Interactive Documentation

### Swagger UI

The API includes interactive Swagger UI documentation that allows you to:
- Browse all available endpoints
- View request/response schemas
- Test endpoints directly from your browser
- Authenticate and make real API calls

**Access Swagger UI:**
- Primary: http://localhost:8080/api/docs
- Alternative: http://localhost:8080/docs
- Alternative: http://localhost:8080/swagger/index.html

### Using Swagger UI

1. **Start the API server:**
   ```bash
   go run cmd/api/main.go
   ```

2. **Navigate to Swagger UI:**
   Open http://localhost:8080/api/docs in your browser

3. **Authenticate:**
   - Click the "Authorize" button at the top right
   - Enter your JWT token in the format: `Bearer YOUR_TOKEN`
   - Click "Authorize" and then "Close"

4. **Test Endpoints:**
   - Expand any endpoint
   - Click "Try it out"
   - Fill in the required parameters
   - Click "Execute"
   - View the response

## Quick Start

### 1. Authentication

First, obtain an access token by logging in:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2024-01-09T12:00:00Z",
  "user": { ... }
}
```

### 2. Make Authenticated Requests

Include the access token in the Authorization header:

```bash
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### 3. Handle Token Expiration

When your token expires, refresh it:

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "YOUR_REFRESH_TOKEN"
  }'
```

## API Features

### Authentication & Authorization
- JWT-based authentication
- Role-based access control (RBAC)
- Token refresh mechanism
- Password reset functionality

### Rate Limiting
- Per-IP rate limiting: 100 requests/minute
- Authentication endpoints: 5 attempts/15 minutes
- Account lockout after 5 failed login attempts

### Error Handling
- Standardized error responses
- Detailed error codes
- Field-level validation errors
- Correlation IDs for debugging

### Pagination
- Consistent pagination across list endpoints
- Configurable page size (max 100 items)
- Sorting and filtering support

## API Endpoints

### Authentication
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - User logout
- `POST /api/v1/auth/forgot-password` - Request password reset
- `POST /api/v1/auth/reset-password` - Reset password

### Users
- `GET /api/v1/users` - List users
- `GET /api/v1/users/{id}` - Get user by ID
- `PUT /api/v1/users/{id}` - Update user
- `DELETE /api/v1/users/{id}` - Delete user

### Products
- `POST /api/v1/products` - Create product
- `GET /api/v1/products` - List products
- `GET /api/v1/products/{id}` - Get product by ID
- `PUT /api/v1/products/{id}` - Update product
- `DELETE /api/v1/products/{id}` - Delete product
- `GET /api/v1/products/search` - Search products

### Categories
- `POST /api/v1/categories` - Create category
- `GET /api/v1/categories` - List categories
- `GET /api/v1/categories/tree` - Get category tree
- `GET /api/v1/categories/{id}` - Get category by ID
- `PUT /api/v1/categories/{id}` - Update category
- `DELETE /api/v1/categories/{id}` - Delete category

### Orders
- `POST /api/v1/orders` - Create order
- `GET /api/v1/orders` - List orders
- `GET /api/v1/orders/{id}` - Get order by ID
- `PUT /api/v1/orders/{id}/status` - Update order status
- `POST /api/v1/orders/{id}/cancel` - Cancel order

### Inventory
- `POST /api/v1/inventory/adjust` - Adjust inventory
- `POST /api/v1/inventory/bulk-adjust` - Bulk inventory adjustment
- `POST /api/v1/inventory/check-availability` - Check availability
- `GET /api/v1/inventory` - List inventory
- `GET /api/v1/inventory/stats` - Get inventory statistics

### Health & Monitoring
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

## Error Codes

The API uses standard HTTP status codes and returns detailed error information:

### Common Error Codes
- `VALIDATION_ERROR` (400) - Input validation failed
- `UNAUTHORIZED` (401) - Authentication required or failed
- `FORBIDDEN` (403) - Insufficient permissions
- `NOT_FOUND` (404) - Resource not found
- `CONFLICT` (409) - Resource conflict
- `RATE_LIMIT_EXCEEDED` (429) - Too many requests
- `INTERNAL_ERROR` (500) - Server error

See [ERROR_CODES.md](ERROR_CODES.md) for complete reference.

## Best Practices

### Security
1. Always use HTTPS in production
2. Store JWT tokens securely (httpOnly cookies recommended)
3. Implement token refresh before expiration
4. Never log or expose sensitive data
5. Validate all input on the client side

### Performance
1. Use pagination for large datasets
2. Implement caching where appropriate
3. Use bulk operations when available
4. Monitor rate limits and implement backoff

### Error Handling
1. Always check HTTP status codes
2. Handle specific error codes appropriately
3. Implement exponential backoff for rate limits
4. Log correlation IDs for debugging

## Development

### Regenerating Documentation

After making changes to API endpoints or documentation annotations:

```bash
# Install swag if not already installed
go install github.com/swaggo/swag/cmd/swag@latest

# Regenerate Swagger documentation
swag init -g cmd/api/docs.go -o docs --parseDependency --parseInternal
```

### Testing Endpoints

Use the Swagger UI for interactive testing, or use curl/Postman for automated testing.

Example test script:
```bash
#!/bin/bash

# Login and get token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}' \
  | jq -r '.access_token')

# Test authenticated endpoint
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN"
```

## Support

For API support and questions:

- **Email**: support@erpgo.example.com
- **Documentation**: https://docs.erpgo.example.com
- **Issues**: https://github.com/erpgo/erpgo/issues

## Version History

### v1.0.0 (Current)
- Initial API release
- Complete CRUD operations for all resources
- JWT authentication
- Rate limiting
- Comprehensive error handling
- Interactive Swagger documentation

## License

AGPL-3.0 - See LICENSE file for details
