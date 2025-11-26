# Swagger UI Setup and Configuration

This document describes the Swagger UI setup and configuration for the ERPGo API.

## Overview

The ERPGo API now includes comprehensive OpenAPI 3.0 documentation with interactive Swagger UI for testing and exploring the API.

## What Was Implemented

### 1. Main Documentation File (`cmd/api/docs.go`)

Created a comprehensive documentation file with:
- **API Overview**: Detailed description of the ERP system and its capabilities
- **Authentication Guide**: Complete JWT authentication flow with examples
- **Rate Limiting Documentation**: Detailed rate limit policies and headers
- **Comprehensive Error Codes**: All error codes with descriptions and resolutions
- **Pagination Documentation**: How to use pagination across endpoints
- **Security Best Practices**: Guidelines for secure API usage
- **Tag Descriptions**: Organized endpoints by functional area

### 2. Error Codes Documentation (`docs/ERROR_CODES.md`)

Complete reference guide including:
- All HTTP status codes used by the API
- Detailed error code descriptions
- Resolution steps for each error
- Error response format examples
- Validation error examples
- Rate limiting details
- Best practices for error handling

### 3. API Examples (`docs/API_EXAMPLES.md`)

Practical examples for:
- Authentication (login, register, refresh token)
- User management operations
- Product CRUD operations
- Category management
- Order processing
- Inventory management
- Error handling patterns
- Pagination examples
- Token refresh implementation

### 4. Swagger UI Configuration

Enhanced Swagger UI with:
- **Multiple Access Points**:
  - Primary: `/api/docs`
  - Alternative: `/api/v1/docs`
  - Alternative: `/docs`
  - Alternative: `/swagger`

- **Configuration Features**:
  - Deep linking enabled
  - Persistent authorization (tokens saved across page refreshes)
  - List expansion for better navigation
  - Proper model depth for complex schemas

- **Authentication Support**:
  - Bearer token authentication configured
  - Authorization button in UI
  - Token persistence across requests

### 5. Documentation README (`docs/README.md`)

Central documentation hub with:
- Links to all documentation files
- Quick start guide
- API endpoint overview
- Development instructions
- Testing guidelines

## Accessing Swagger UI

### Local Development

1. Start the API server:
   ```bash
   go run cmd/api/main.go
   ```

2. Open Swagger UI in your browser:
   - http://localhost:8080/api/docs
   - http://localhost:8080/docs
   - http://localhost:8080/swagger/index.html

### Using Swagger UI

1. **Authenticate**:
   - Click the "Authorize" button (lock icon) at the top right
   - Enter your JWT token in the format: `Bearer YOUR_TOKEN`
   - Click "Authorize" and then "Close"
   - Your token will be included in all subsequent requests

2. **Test Endpoints**:
   - Browse endpoints by tag (Authentication, Users, Products, etc.)
   - Click on an endpoint to expand it
   - Click "Try it out" to enable the form
   - Fill in required parameters
   - Click "Execute" to make the request
   - View the response below

3. **View Schemas**:
   - Scroll to the bottom to see all data models
   - Click on a model to expand its properties
   - View validation rules and examples

## Regenerating Documentation

After making changes to API endpoints or documentation annotations:

```bash
# Install swag if not already installed
go install github.com/swaggo/swag/cmd/swag@latest

# Regenerate Swagger documentation
swag init -g cmd/api/docs.go -o docs --parseDependency --parseInternal
```

This will update:
- `docs/docs.go` - Generated Go code
- `docs/swagger.json` - JSON specification
- `docs/swagger.yaml` - YAML specification

## Documentation Annotations

The main documentation is in `cmd/api/docs.go` with annotations:

```go
// @title ERPGo API
// @version 1.0.0
// @description Comprehensive ERP system API

// @contact.name ERPGo Support
// @contact.email support@erpgo.example.com

// @license.name AGPL-3.0
// @license.url https://www.gnu.org/licenses/agpl-3.0.en.html

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
```

Individual endpoint documentation is in handler files using Swagger annotations.

## Features Documented

### Authentication & Authorization
- JWT token-based authentication
- Token refresh mechanism
- Password reset flow
- Role-based access control

### Error Handling
- Standardized error responses
- Detailed error codes
- Field-level validation errors
- Correlation IDs for debugging

### Rate Limiting
- Per-IP rate limiting
- Authentication endpoint limits
- Account lockout policies
- Rate limit headers

### Pagination
- Consistent pagination across endpoints
- Configurable page size
- Sorting and filtering
- Pagination metadata

### Security
- HTTPS recommendations
- Token storage best practices
- Input validation
- Security headers

## Configuration Options

The Swagger UI is configured in `cmd/api/main.go`:

```go
swaggerConfig := ginSwagger.Config{
    URL:                      "/swagger/doc.json",
    DocExpansion:             "list",
    DeepLinking:              true,
    DefaultModelsExpandDepth: 1,
    PersistAuthorization:     true,
}
```

### Configuration Parameters

- **URL**: Path to the OpenAPI specification
- **DocExpansion**: How endpoints are displayed (`list`, `full`, `none`)
- **DeepLinking**: Enable direct links to specific endpoints
- **DefaultModelsExpandDepth**: How deep to expand model schemas
- **PersistAuthorization**: Save auth tokens across page refreshes

## Testing with Swagger UI

### Example Workflow

1. **Login**:
   - Navigate to `POST /api/v1/auth/login`
   - Click "Try it out"
   - Enter credentials:
     ```json
     {
       "email": "user@example.com",
       "password": "password123"
     }
     ```
   - Click "Execute"
   - Copy the `access_token` from the response

2. **Authorize**:
   - Click "Authorize" button at top
   - Enter: `Bearer YOUR_ACCESS_TOKEN`
   - Click "Authorize" and "Close"

3. **Test Protected Endpoint**:
   - Navigate to `GET /api/v1/users`
   - Click "Try it out"
   - Set pagination parameters if desired
   - Click "Execute"
   - View the response

## Troubleshooting

### Swagger UI Not Loading

1. Verify the server is running:
   ```bash
   curl http://localhost:8080/health
   ```

2. Check that Swagger files exist:
   ```bash
   ls -la docs/swagger.*
   ```

3. Regenerate documentation:
   ```bash
   swag init -g cmd/api/docs.go -o docs --parseDependency --parseInternal
   ```

### Authentication Not Working

1. Ensure token format is correct: `Bearer YOUR_TOKEN`
2. Check token hasn't expired (default: 15 minutes)
3. Verify token is valid by testing with curl:
   ```bash
   curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/v1/users
   ```

### Endpoints Not Showing

1. Ensure handler has Swagger annotations
2. Regenerate documentation
3. Restart the server
4. Clear browser cache

## Production Considerations

### Security

1. **Disable Swagger UI in Production** (optional):
   ```go
   if !cfg.IsProduction() {
       router.GET("/api/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
   }
   ```

2. **Protect with Authentication**:
   ```go
   docs := router.Group("/api/docs")
   docs.Use(authMiddleware)
   docs.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
   ```

3. **Use HTTPS Only**:
   - Update `@schemes` to only include `https`
   - Configure proper TLS certificates

### Performance

1. **Cache Swagger Files**:
   - Serve static files with caching headers
   - Use CDN for Swagger UI assets

2. **Minimize Documentation Size**:
   - Remove unnecessary examples
   - Compress JSON/YAML files

## Additional Resources

- [Swagger Documentation](https://swagger.io/docs/)
- [OpenAPI Specification](https://spec.openapis.org/oas/v3.0.0)
- [Swaggo Documentation](https://github.com/swaggo/swag)
- [Gin Swagger](https://github.com/swaggo/gin-swagger)

## Support

For questions or issues with the API documentation:
- Email: support@erpgo.example.com
- Documentation: https://docs.erpgo.example.com
- GitHub Issues: https://github.com/erpgo/erpgo/issues
