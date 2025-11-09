# ERPGo Authentication Guide

## Overview

ERPGo uses JSON Web Tokens (JWT) for authentication and implements a comprehensive role-based access control (RBAC) system. This guide covers authentication flows, token management, and security best practices.

## Table of Contents
1. [Authentication Flow](#authentication-flow)
2. [JWT Token Structure](#jwt-token-structure)
3. [API Authentication](#api-authentication)
4. [Role-Based Access Control](#role-based-access-control)
5. [Security Best Practices](#security-best-practices)
6. [Token Management](#token-management)
7. [Error Handling](#error-handling)

## Authentication Flow

### User Registration

1. **Request**:
   ```http
   POST /api/v1/auth/register
   Content-Type: application/json

   {
     "email": "user@example.com",
     "username": "johndoe",
     "password": "SecurePassword123!",
     "first_name": "John",
     "last_name": "Doe",
     "phone": "+1234567890"
   }
   ```

2. **Response**:
   ```http
   HTTP/1.1 201 Created
   Content-Type: application/json

   {
     "message": "User registered successfully",
     "user": {
       "id": "550e8400-e29b-41d4-a716-446655440000",
       "email": "user@example.com",
       "username": "johndoe",
       "first_name": "John",
       "last_name": "Doe",
       "is_verified": false,
       "created_at": "2024-01-01T00:00:00Z"
     }
   }
   ```

### User Login

1. **Request**:
   ```http
   POST /api/v1/auth/login
   Content-Type: application/json

   {
     "email": "user@example.com",
     "password": "SecurePassword123!",
     "remember": true
   }
   ```

2. **Response**:
   ```http
   HTTP/1.1 200 OK
   Content-Type: application/json

   {
     "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
     "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
     "expires_at": "2024-01-01T08:00:00Z",
     "user": {
       "id": "550e8400-e29b-41d4-a716-446655440000",
       "email": "user@example.com",
       "username": "johndoe",
       "roles": ["user"],
       "is_active": true,
       "is_verified": true
     }
   }
   ```

### Token Refresh

1. **Request**:
   ```http
   POST /api/v1/auth/refresh
   Content-Type: application/json

   {
     "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
   }
   ```

2. **Response**:
   ```http
   HTTP/1.1 200 OK
   Content-Type: application/json

   {
     "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
     "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
     "expires_at": "2024-01-01T08:00:00Z"
   }
   ```

## JWT Token Structure

### Access Token

Access tokens are short-lived JWTs (typically 15-30 minutes) with the following claims:

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "username": "johndoe",
  "roles": ["user", "manager"],
  "iat": 1640995200,
  "exp": 1640998800,
  "iss": "erpgo-api",
  "aud": "erpgo-client"
}
```

**Token Claims:**
- `sub`: Subject (User ID)
- `email`: User email address
- `username`: User's username
- `roles`: Array of user roles
- `iat`: Issued at timestamp
- `exp`: Expiration timestamp
- `iss`: Issuer (API identifier)
- `aud`: Audience (client identifier)

### Refresh Token

Refresh tokens are long-lived JWTs (typically 7-30 days) used to obtain new access tokens:

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "type": "refresh",
  "iat": 1640995200,
  "exp": 1641600000,
  "iss": "erpgo-api"
}
```

## API Authentication

### Including Tokens in Requests

Include the JWT access token in the Authorization header:

```http
GET /api/v1/users/me
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

### Authentication Middleware

All protected routes use authentication middleware that:

1. Validates JWT signature
2. Checks token expiration
3. Verifies issuer and audience
4. Extracts user information into request context
5. Enforces rate limiting per user

### Example Protected Route

```http
GET /api/v1/profile
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

Response:
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "username": "johndoe",
  "first_name": "John",
  "last_name": "Doe",
  "roles": ["user"],
  "is_active": true,
  "last_login": "2024-01-01T12:00:00Z"
}
```

## Role-Based Access Control

### Default Roles

#### Admin
- Full system access
- User management
- System configuration
- All CRUD operations

#### Manager
- Business operations
- Order management
- Inventory management
- Customer management

#### Operator
- Day-to-day operations
- Order processing
- Inventory adjustments
- Customer service

#### Viewer
- Read-only access
- Dashboard viewing
- Report generation

### Permission Matrix

| Feature | Admin | Manager | Operator | Viewer |
|---------|-------|---------|----------|--------|
| Users (CRUD) | ✅ | ❌ | ❌ | ❌ |
| Products (CRUD) | ✅ | ✅ | ⚠️* | 📖 |
| Inventory (Adjust) | ✅ | ✅ | ✅ | ❌ |
| Inventory (View) | ✅ | ✅ | ✅ | ✅ |
| Orders (Process) | ✅ | ✅ | ✅ | 📖 |
| Orders (View) | ✅ | ✅ | ✅ | ✅ |
| Customers (CRUD) | ✅ | ✅ | ⚠️* | 📖 |
| Reports | ✅ | ✅ | ⚠️* | ✅ |

*⚠️ Limited access (own records only)
*📖 Read-only access

### Role Assignment

Only administrators can assign roles:

```http
POST /api/v1/users/{user_id}/roles
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "role_id": "role_uuid_here"
}
```

### Authorization Middleware

Protected routes use authorization middleware that checks:

1. User authentication status
2. Required roles for the endpoint
3. Resource ownership (when applicable)
4. Rate limits based on user role

### Example Role-Protected Endpoint

```http
POST /api/v1/products
Authorization: Bearer <admin-or-manager-token>
Content-Type: application/json

{
  "name": "New Product",
  "sku": "PROD-001",
  "price": 99.99,
  "category_id": "category_uuid"
}
```

## Security Best Practices

### Password Requirements

- Minimum 8 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one number
- At least one special character
- Cannot be a common password
- Cannot contain email or username

### Rate Limiting

Authentication endpoints have strict rate limiting:

- Login: 5 attempts per 15 minutes per IP
- Registration: 3 attempts per hour per IP
- Password reset: 3 attempts per hour per email

### Token Security

1. **Access Tokens**: 15-minute expiration
2. **Refresh Tokens**: 7-day expiration
3. **Token Storage**: Store tokens securely (httpOnly cookies or secure storage)
4. **Token Rotation**: Refresh tokens are rotated on each use
5. **Token Revocation**: Tokens are revoked on logout

### Session Management

- Active sessions are tracked in Redis
- Maximum 5 concurrent sessions per user
- Old sessions are invalidated when limit exceeded
- Users can view and revoke active sessions

### Password Reset Flow

1. **Request Password Reset**:
   ```http
   POST /api/v1/auth/forgot-password
   Content-Type: application/json

   {
     "email": "user@example.com"
   }
   ```

2. **Reset Password**:
   ```http
   POST /api/v1/auth/reset-password
   Content-Type: application/json

   {
     "token": "reset_token_here",
     "new_password": "NewSecurePassword123!"
   }
   ```

## Token Management

### Token Storage Recommendations

**Web Applications:**
- Store tokens in httpOnly, secure cookies
- Use SameSite=Strict or SameSite=Lax
- Implement CSRF protection

**Mobile Applications:**
- Use platform-specific secure storage
- Keychain (iOS) or Keystore (Android)
- Implement biometric authentication for token access

**Single Page Applications:**
- Store tokens in memory
- Use refresh tokens stored securely
- Implement silent token refresh

### Token Refresh Implementation

**Automatic Token Refresh:**
```javascript
// Example: Axios interceptor for automatic token refresh
axios.interceptors.response.use(
  response => response,
  async error => {
    if (error.response?.status === 401) {
      try {
        const refreshToken = localStorage.getItem('refresh_token');
        const response = await axios.post('/api/v1/auth/refresh', {
          refresh_token: refreshToken
        });

        localStorage.setItem('access_token', response.data.access_token);
        localStorage.setItem('refresh_token', response.data.refresh_token);

        // Retry original request
        error.config.headers.Authorization = `Bearer ${response.data.access_token}`;
        return axios.request(error.config);
      } catch (refreshError) {
        // Refresh failed, redirect to login
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);
```

### Session Management

**View Active Sessions:**
```http
GET /api/v1/auth/sessions
Authorization: Bearer <access_token>
```

**Revoke Session:**
```http
DELETE /api/v1/auth/sessions/{session_id}
Authorization: Bearer <access_token>
```

**Revoke All Sessions (Logout Everywhere):**
```http
POST /api/v1/auth/logout-all
Authorization: Bearer <access_token>
```

## Error Handling

### Authentication Errors

**Invalid Credentials (401):**
```json
{
  "error": "Invalid credentials",
  "details": "Email or password is incorrect"
}
```

**Token Expired (401):**
```json
{
  "error": "Token expired",
  "details": "Access token has expired, please refresh"
}
```

**Invalid Token (401):**
```json
{
  "error": "Invalid token",
  "details": "Token is malformed or signature is invalid"
}
```

**Insufficient Permissions (403):**
```json
{
  "error": "Insufficient permissions",
  "details": "You don't have permission to access this resource"
}
```

### Authorization Errors

**Role Required (403):**
```json
{
  "error": "Role required",
  "details": "This endpoint requires admin or manager role"
}
```

**Account Inactive (403):**
```json
{
  "error": "Account inactive",
  "details": "Your account has been deactivated, please contact support"
}
```

**Email Not Verified (403):**
```json
{
  "error": "Email not verified",
  "details": "Please verify your email address before proceeding"
}
```

### Rate Limiting Errors

**Too Many Requests (429):**
```json
{
  "error": "Too many requests",
  "details": "Rate limit exceeded, please try again later",
  "retry_after": 900
}
```

## Development Testing

### Test Authentication

Use the test endpoint to verify authentication setup:

```http
POST /api/v1/auth/test
Authorization: Bearer <test-token>
Content-Type: application/json

{
  "message": "Test authentication"
}
```

### Environment Variables for Testing

```bash
# JWT Configuration
JWT_SECRET=your-super-secret-key
JWT_ACCESS_TOKEN_DURATION=15m
JWT_REFRESH_TOKEN_DURATION=168h

# Rate Limiting
RATE_LIMIT_LOGIN=5
RATE_LIMIT_REGISTER=3
RATE_LIMIT_PASSWORD_RESET=3

# Security
BCRYPT_COST=12
SESSION_MAX_DEVICES=5
```

### Development Tools

- **JWT Debugger**: Use https://jwt.io to inspect tokens
- **Postman Collections**: Import API collection with authentication setup
- **Swagger UI**: Test authentication endpoints at `/docs`
- **Health Check**: Monitor authentication service health at `/health`

---

For implementation details and code examples, refer to the [Developer Guide](DEVELOPER_GUIDE.md) and the API documentation at `/docs`.