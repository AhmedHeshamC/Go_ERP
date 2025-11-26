package main

// @title ERPGo API
// @version 1.0.0
// @description A comprehensive Enterprise Resource Planning (ERP) system built with Go, providing user management, inventory management, order management, and business analytics capabilities.
// @description
// @description ## Overview
// @description ERPGo is a modern, cloud-native ERP system designed for high performance, scalability, and maintainability. It provides:
// @description - User authentication and authorization with JWT tokens
// @description - Product catalog management with categories and variants
// @description - Inventory tracking and warehouse management
// @description - Order processing and fulfillment workflows
// @description - Customer management and address handling
// @description - Comprehensive analytics and reporting
// @description
// @description ## Authentication
// @description The API uses JWT (JSON Web Tokens) for authentication. To authenticate:
// @description 1. Call the `/api/v1/auth/login` endpoint with your credentials
// @description 2. Include the returned token in the `Authorization` header for subsequent requests:
// @description    ```
// @description    Authorization: Bearer <your-jwt-token>
// @description    ```
// @description
// @description ## Rate Limiting
// @description API requests are rate-limited to ensure fair usage:
// @description - **Default**: 100 requests per minute per IP
// @description - **Authentication endpoints**: 5 attempts per 15 minutes per IP
// @description - **Account lockout**: After 5 failed login attempts, accounts are locked for 15 minutes
// @description
// @description Rate limit headers are included in responses:
// @description ```
// @description X-RateLimit-Limit: 100
// @description X-RateLimit-Remaining: 99
// @description X-RateLimit-Reset: 1640995200
// @description ```
// @description
// @description ## Error Codes
// @description The API uses standard HTTP status codes and returns detailed error information:
// @description
// @description ### Success Codes
// @description - **200 OK**: Request succeeded
// @description - **201 Created**: Resource created successfully
// @description - **204 No Content**: Request succeeded with no response body
// @description
// @description ### Client Error Codes
// @description - **400 Bad Request**: Invalid request format or parameters
// @description   - Error Code: `VALIDATION_ERROR` - Input validation failed
// @description   - Error Code: `INVALID_REQUEST` - Malformed request
// @description - **401 Unauthorized**: Authentication required or failed
// @description   - Error Code: `UNAUTHORIZED` - Missing or invalid authentication token
// @description   - Error Code: `TOKEN_EXPIRED` - JWT token has expired
// @description   - Error Code: `INVALID_CREDENTIALS` - Incorrect email or password
// @description - **403 Forbidden**: Authenticated but not authorized
// @description   - Error Code: `FORBIDDEN` - Insufficient permissions
// @description   - Error Code: `ACCOUNT_LOCKED` - Account temporarily locked due to failed login attempts
// @description - **404 Not Found**: Resource does not exist
// @description   - Error Code: `NOT_FOUND` - Requested resource not found
// @description - **409 Conflict**: Resource conflict
// @description   - Error Code: `CONFLICT` - Resource already exists
// @description   - Error Code: `DUPLICATE_EMAIL` - Email already registered
// @description   - Error Code: `DUPLICATE_SKU` - Product SKU already exists
// @description - **422 Unprocessable Entity**: Validation errors
// @description   - Error Code: `VALIDATION_ERROR` - One or more fields failed validation
// @description - **429 Too Many Requests**: Rate limit exceeded
// @description   - Error Code: `RATE_LIMIT_EXCEEDED` - Too many requests, try again later
// @description
// @description ### Server Error Codes
// @description - **500 Internal Server Error**: Unexpected server error
// @description   - Error Code: `INTERNAL_ERROR` - An unexpected error occurred
// @description - **503 Service Unavailable**: Service temporarily unavailable
// @description   - Error Code: `SERVICE_UNAVAILABLE` - Database or cache unavailable
// @description
// @description ### Error Response Format
// @description All errors return a JSON response with the following structure:
// @description ```json
// @description {
// @description   "error": "Human-readable error message",
// @description   "code": "ERROR_CODE",
// @description   "details": "Additional context or field-level errors"
// @description }
// @description ```
// @description
// @description For validation errors, the `details` field contains field-specific errors:
// @description ```json
// @description {
// @description   "error": "Validation failed",
// @description   "code": "VALIDATION_ERROR",
// @description   "details": {
// @description     "email": ["Email is required", "Email must be valid"],
// @description     "password": ["Password must be at least 8 characters"]
// @description   }
// @description }
// @description ```
// @description
// @description ## Pagination
// @description List endpoints support pagination with the following query parameters:
// @description - `page`: Page number (default: 1, min: 1)
// @description - `limit`: Items per page (default: 20, min: 1, max: 100)
// @description - `sort_by`: Field to sort by (varies by endpoint)
// @description - `sort_order`: Sort direction (`asc` or `desc`, default: `desc`)
// @description
// @description Paginated responses include metadata:
// @description ```json
// @description {
// @description   "data": [...],
// @description   "pagination": {
// @description     "page": 1,
// @description     "limit": 20,
// @description     "total": 150,
// @description     "total_pages": 8,
// @description     "has_next": true,
// @description     "has_prev": false
// @description   }
// @description }
// @description ```
// @description
// @description ## Filtering and Search
// @description Many list endpoints support filtering and search:
// @description - `search`: Full-text search across relevant fields
// @description - Resource-specific filters (e.g., `is_active`, `category_id`, `status`)
// @description
// @description ## Security Best Practices
// @description - Always use HTTPS in production
// @description - Store JWT tokens securely (httpOnly cookies recommended)
// @description - Implement token refresh before expiration
// @description - Never log or expose sensitive data (passwords, tokens)
// @description - Validate all input on the client side before sending
// @description - Handle rate limit errors gracefully with exponential backoff
// @description
// @description ## Support
// @description For API support, contact: support@erpgo.example.com
// @description Documentation: https://docs.erpgo.example.com

// @contact.name ERPGo Support
// @contact.url https://docs.erpgo.example.com
// @contact.email support@erpgo.example.com

// @license.name AGPL-3.0
// @license.url https://www.gnu.org/licenses/agpl-3.0.en.html

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token. Example: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

// @schemes http https
// @produce json
// @consumes json

// @tag.name Authentication
// @tag.description User authentication and authorization endpoints. Includes login, registration, token refresh, and password reset.

// @tag.name Users
// @tag.description User management operations. Requires authentication and appropriate permissions.

// @tag.name Products
// @tag.description Product catalog management including creation, updates, and search functionality.

// @tag.name Categories
// @tag.description Product category management with hierarchical structure support.

// @tag.name Variants
// @tag.description Product variant management for products with multiple options (size, color, etc.).

// @tag.name Inventory
// @tag.description Inventory tracking and stock management across multiple warehouses.

// @tag.name Warehouses
// @tag.description Warehouse management including location and capacity tracking.

// @tag.name Orders
// @tag.description Order processing, fulfillment, and management workflows.

// @tag.name Customers
// @tag.description Customer management including contact information and order history.

// @tag.name Images
// @tag.description Image upload and management for products and other resources.

// @tag.name Health
// @tag.description System health checks and monitoring endpoints.
