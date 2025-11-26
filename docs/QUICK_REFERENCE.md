# ERPGo API Quick Reference

## 🚀 Quick Start

### 1. Access Swagger UI
```
http://localhost:8080/api/docs
```

### 2. Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

### 3. Use Token
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/api/v1/users
```

## 📚 Documentation

| Resource | Location |
|----------|----------|
| Swagger UI | http://localhost:8080/api/docs |
| Main Docs | [docs/README.md](README.md) |
| Error Codes | [docs/ERROR_CODES.md](ERROR_CODES.md) |
| Examples | [docs/API_EXAMPLES.md](API_EXAMPLES.md) |
| Setup Guide | [docs/SWAGGER_SETUP.md](SWAGGER_SETUP.md) |

## 🔐 Authentication

### Login
```
POST /api/v1/auth/login
Body: {"email": "...", "password": "..."}
Returns: {"access_token": "...", "refresh_token": "..."}
```

### Use Token
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

### Refresh Token
```
POST /api/v1/auth/refresh
Body: {"refresh_token": "..."}
```

## 📋 Common Endpoints

### Users
```
GET    /api/v1/users           - List users
GET    /api/v1/users/{id}      - Get user
PUT    /api/v1/users/{id}      - Update user
DELETE /api/v1/users/{id}      - Delete user
```

### Products
```
POST   /api/v1/products        - Create product
GET    /api/v1/products        - List products
GET    /api/v1/products/{id}   - Get product
PUT    /api/v1/products/{id}   - Update product
DELETE /api/v1/products/{id}   - Delete product
GET    /api/v1/products/search - Search products
```

### Orders
```
POST   /api/v1/orders          - Create order
GET    /api/v1/orders          - List orders
GET    /api/v1/orders/{id}     - Get order
PUT    /api/v1/orders/{id}/status - Update status
POST   /api/v1/orders/{id}/cancel - Cancel order
```

### Inventory
```
POST   /api/v1/inventory/adjust       - Adjust inventory
POST   /api/v1/inventory/bulk-adjust  - Bulk adjustment
POST   /api/v1/inventory/check-availability - Check stock
GET    /api/v1/inventory              - List inventory
GET    /api/v1/inventory/stats        - Get statistics
```

## ⚠️ Error Codes

| Code | Status | Meaning |
|------|--------|---------|
| `VALIDATION_ERROR` | 400 | Invalid input |
| `UNAUTHORIZED` | 401 | Auth required |
| `TOKEN_EXPIRED` | 401 | Token expired |
| `FORBIDDEN` | 403 | No permission |
| `NOT_FOUND` | 404 | Not found |
| `CONFLICT` | 409 | Already exists |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |

## 🔄 Pagination

```
?page=1&limit=20&sort_by=created_at&sort_order=desc
```

Response includes:
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false
  }
}
```

## 🚦 Rate Limits

| Endpoint | Limit | Window |
|----------|-------|--------|
| General | 100 req | 1 min |
| Login | 5 attempts | 15 min |
| Register | 3 attempts | 1 hour |

Headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 99
X-RateLimit-Reset: 1640995200
```

## 🛠️ Development

### Regenerate Docs
```bash
swag init -g cmd/api/docs.go -o docs --parseDependency --parseInternal
```

### Verify Setup
```bash
./scripts/verify-swagger.sh
```

### Start Server
```bash
go run cmd/api/main.go
```

## 💡 Tips

1. **Use Swagger UI** for interactive testing
2. **Check error codes** in ERROR_CODES.md
3. **Copy examples** from API_EXAMPLES.md
4. **Implement retry logic** for rate limits
5. **Refresh tokens** before expiration
6. **Log correlation IDs** for debugging

## 🔗 Links

- Swagger UI: http://localhost:8080/api/docs
- Health Check: http://localhost:8080/health
- Metrics: http://localhost:8080/metrics

## 📞 Support

- Email: support@erpgo.example.com
- Docs: https://docs.erpgo.example.com
