# ERPGo API Error Codes

This document provides a comprehensive reference for all error codes returned by the ERPGo API.

## Error Response Format

All API errors return a JSON response with the following structure:

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "details": "Additional context or field-level errors"
}
```

For validation errors, the `details` field contains field-specific errors:

```json
{
  "error": "Validation failed",
  "code": "VALIDATION_ERROR",
  "details": {
    "email": ["Email is required", "Email must be valid"],
    "password": ["Password must be at least 8 characters"]
  }
}
```

## HTTP Status Codes

### Success Codes (2xx)

| Status Code | Description |
|------------|-------------|
| 200 OK | Request succeeded |
| 201 Created | Resource created successfully |
| 204 No Content | Request succeeded with no response body |

### Client Error Codes (4xx)

| Status Code | Error Code | Description | Resolution |
|------------|------------|-------------|------------|
| 400 Bad Request | `VALIDATION_ERROR` | Input validation failed | Check the `details` field for specific validation errors |
| 400 Bad Request | `INVALID_REQUEST` | Malformed request | Verify request format and required fields |
| 401 Unauthorized | `UNAUTHORIZED` | Missing or invalid authentication token | Include valid JWT token in Authorization header |
| 401 Unauthorized | `TOKEN_EXPIRED` | JWT token has expired | Refresh token using `/api/v1/auth/refresh` endpoint |
| 401 Unauthorized | `INVALID_CREDENTIALS` | Incorrect email or password | Verify credentials and try again |
| 403 Forbidden | `FORBIDDEN` | Insufficient permissions | Contact administrator for required permissions |
| 403 Forbidden | `ACCOUNT_LOCKED` | Account temporarily locked | Wait 15 minutes or contact support |
| 404 Not Found | `NOT_FOUND` | Requested resource not found | Verify resource ID and try again |
| 409 Conflict | `CONFLICT` | Resource already exists | Use different identifier or update existing resource |
| 409 Conflict | `DUPLICATE_EMAIL` | Email already registered | Use different email or login with existing account |
| 409 Conflict | `DUPLICATE_SKU` | Product SKU already exists | Use unique SKU for product |
| 422 Unprocessable Entity | `VALIDATION_ERROR` | One or more fields failed validation | Check `details` for field-specific errors |
| 429 Too Many Requests | `RATE_LIMIT_EXCEEDED` | Too many requests | Wait and retry with exponential backoff |

### Server Error Codes (5xx)

| Status Code | Error Code | Description | Resolution |
|------------|------------|-------------|------------|
| 500 Internal Server Error | `INTERNAL_ERROR` | Unexpected server error | Contact support with correlation ID |
| 503 Service Unavailable | `SERVICE_UNAVAILABLE` | Database or cache unavailable | Retry after a few seconds |

## Authentication Error Codes

### Login Errors

| Error Code | HTTP Status | Description | Resolution |
|-----------|-------------|-------------|------------|
| `INVALID_CREDENTIALS` | 401 | Email or password incorrect | Verify credentials |
| `ACCOUNT_LOCKED` | 403 | Account locked after failed attempts | Wait 15 minutes |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many login attempts | Wait 15 minutes |
| `VALIDATION_ERROR` | 400 | Missing or invalid fields | Check email and password format |

### Token Errors

| Error Code | HTTP Status | Description | Resolution |
|-----------|-------------|-------------|------------|
| `TOKEN_EXPIRED` | 401 | JWT token expired | Refresh token |
| `INVALID_TOKEN` | 401 | Token malformed or invalid | Login again |
| `TOKEN_BLACKLISTED` | 401 | Token has been revoked | Login again |
| `UNAUTHORIZED` | 401 | No token provided | Include Authorization header |

### Registration Errors

| Error Code | HTTP Status | Description | Resolution |
|-----------|-------------|-------------|------------|
| `DUPLICATE_EMAIL` | 409 | Email already registered | Use different email or login |
| `DUPLICATE_USERNAME` | 409 | Username already taken | Choose different username |
| `VALIDATION_ERROR` | 422 | Invalid registration data | Check field requirements |
| `WEAK_PASSWORD` | 422 | Password doesn't meet requirements | Use stronger password (min 8 chars) |

## Resource Error Codes

### Product Errors

| Error Code | HTTP Status | Description | Resolution |
|-----------|-------------|-------------|------------|
| `PRODUCT_NOT_FOUND` | 404 | Product does not exist | Verify product ID |
| `DUPLICATE_SKU` | 409 | SKU already exists | Use unique SKU |
| `INVALID_CATEGORY` | 400 | Category does not exist | Use valid category ID |
| `INVALID_PRICE` | 422 | Price must be positive | Provide valid price |
| `INSUFFICIENT_STOCK` | 400 | Not enough inventory | Reduce quantity or restock |

### Order Errors

| Error Code | HTTP Status | Description | Resolution |
|-----------|-------------|-------------|------------|
| `ORDER_NOT_FOUND` | 404 | Order does not exist | Verify order ID |
| `INVALID_ORDER_STATUS` | 400 | Cannot perform action in current status | Check order status |
| `INSUFFICIENT_INVENTORY` | 400 | Not enough stock for order | Reduce quantity or wait for restock |
| `INVALID_CUSTOMER` | 400 | Customer does not exist | Use valid customer ID |
| `INVALID_ADDRESS` | 400 | Shipping/billing address invalid | Provide complete address |
| `ORDER_ALREADY_CANCELLED` | 409 | Order already cancelled | Cannot modify cancelled order |
| `ORDER_ALREADY_SHIPPED` | 409 | Order already shipped | Cannot cancel shipped order |

### Inventory Errors

| Error Code | HTTP Status | Description | Resolution |
|-----------|-------------|-------------|------------|
| `INVENTORY_NOT_FOUND` | 404 | Inventory record not found | Verify product and warehouse |
| `INSUFFICIENT_STOCK` | 400 | Not enough inventory | Adjust quantity or restock |
| `INVALID_WAREHOUSE` | 400 | Warehouse does not exist | Use valid warehouse ID |
| `NEGATIVE_STOCK` | 400 | Stock cannot be negative | Adjust quantity |
| `INVALID_ADJUSTMENT` | 400 | Invalid inventory adjustment | Check adjustment reason and quantity |

### User Errors

| Error Code | HTTP Status | Description | Resolution |
|-----------|-------------|-------------|------------|
| `USER_NOT_FOUND` | 404 | User does not exist | Verify user ID |
| `USER_INACTIVE` | 403 | User account is inactive | Contact administrator |
| `INVALID_ROLE` | 400 | Role does not exist | Use valid role ID |
| `PERMISSION_DENIED` | 403 | User lacks required permission | Request permission from administrator |

## Rate Limiting

### Rate Limit Headers

All responses include rate limit information:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 99
X-RateLimit-Reset: 1640995200
```

### Rate Limit Policies

| Endpoint Pattern | Limit | Window | Scope |
|-----------------|-------|--------|-------|
| `/api/v1/auth/login` | 5 attempts | 15 minutes | Per IP |
| `/api/v1/auth/register` | 3 attempts | 1 hour | Per IP |
| `/api/v1/*` (general) | 100 requests | 1 minute | Per IP |
| `/api/v1/*` (authenticated) | 1000 requests | 1 hour | Per user |

### Rate Limit Error Response

```json
{
  "error": "Rate limit exceeded",
  "code": "RATE_LIMIT_EXCEEDED",
  "details": {
    "retry_after": 900,
    "limit": 5,
    "window": "15 minutes"
  }
}
```

## Validation Errors

### Common Validation Rules

| Field Type | Validation Rules |
|-----------|------------------|
| Email | Valid email format, max 255 characters |
| Password | Min 8 characters, at least one letter and number |
| UUID | Valid UUID v4 format |
| Phone | Valid phone number format |
| URL | Valid URL format |
| Price | Positive decimal, max 2 decimal places |
| Quantity | Positive integer |
| Date | ISO 8601 format (YYYY-MM-DD) |
| DateTime | ISO 8601 format with timezone |

### Validation Error Example

```json
{
  "error": "Validation failed",
  "code": "VALIDATION_ERROR",
  "details": {
    "email": [
      "Email is required",
      "Email must be a valid email address"
    ],
    "password": [
      "Password must be at least 8 characters",
      "Password must contain at least one letter and one number"
    ],
    "price": [
      "Price must be a positive number"
    ]
  }
}
```

## Best Practices

### Error Handling

1. **Always check HTTP status code first**
   ```javascript
   if (response.status >= 400) {
     const error = await response.json();
     console.error(`Error ${error.code}: ${error.error}`);
   }
   ```

2. **Handle specific error codes**
   ```javascript
   switch (error.code) {
     case 'TOKEN_EXPIRED':
       await refreshToken();
       break;
     case 'RATE_LIMIT_EXCEEDED':
       await sleep(error.details.retry_after * 1000);
       break;
     case 'VALIDATION_ERROR':
       displayFieldErrors(error.details);
       break;
   }
   ```

3. **Implement exponential backoff for rate limits**
   ```javascript
   async function retryWithBackoff(fn, maxRetries = 3) {
     for (let i = 0; i < maxRetries; i++) {
       try {
         return await fn();
       } catch (error) {
         if (error.code === 'RATE_LIMIT_EXCEEDED' && i < maxRetries - 1) {
           await sleep(Math.pow(2, i) * 1000);
         } else {
           throw error;
         }
       }
     }
   }
   ```

4. **Log correlation IDs for debugging**
   ```javascript
   if (error.code === 'INTERNAL_ERROR') {
     console.error(`Server error. Correlation ID: ${response.headers.get('X-Correlation-ID')}`);
   }
   ```

### Security Considerations

1. **Never expose sensitive information in error messages**
   - Error messages should be user-friendly but not reveal system internals
   - Use correlation IDs for debugging instead of stack traces

2. **Handle authentication errors gracefully**
   - Redirect to login on 401 errors
   - Clear stored tokens on authentication failures
   - Implement automatic token refresh

3. **Respect rate limits**
   - Implement client-side rate limiting
   - Use exponential backoff for retries
   - Cache responses when appropriate

## Support

If you encounter an error not documented here or need assistance:

- **Email**: support@erpgo.example.com
- **Documentation**: https://docs.erpgo.example.com
- **Status Page**: https://status.erpgo.example.com

When reporting errors, include:
- HTTP status code
- Error code
- Correlation ID (from `X-Correlation-ID` header)
- Request details (endpoint, method, timestamp)
- Steps to reproduce
