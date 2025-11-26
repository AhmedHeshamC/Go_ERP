# ERPGo API Examples

This document provides practical examples for common API operations.

## Table of Contents

- [Authentication](#authentication)
- [User Management](#user-management)
- [Product Management](#product-management)
- [Order Management](#order-management)
- [Inventory Management](#inventory-management)

## Authentication

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "remember": true
  }'
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2024-01-09T12:00:00Z",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "username": "user",
    "first_name": "John",
    "last_name": "Doe",
    "roles": ["user"],
    "is_active": true,
    "is_verified": true
  }
}
```

### Register

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "username": "newuser",
    "password": "password123",
    "first_name": "Jane",
    "last_name": "Smith",
    "phone": "+1234567890"
  }'
```

### Refresh Token

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

### Using Authentication Token

Include the token in the Authorization header for all authenticated requests:

```bash
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## User Management

### List Users

```bash
curl -X GET "http://localhost:8080/api/v1/users?page=1&limit=20&is_active=true" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Response:
```json
{
  "users": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "username": "user",
      "first_name": "John",
      "last_name": "Doe",
      "roles": ["user", "admin"],
      "is_active": true,
      "is_verified": true,
      "created_at": "2024-01-08T10:30:00Z"
    }
  ],
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

### Get User by ID

```bash
curl -X GET http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Update User

```bash
curl -X PUT http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "John",
    "last_name": "Doe Updated",
    "phone": "+1234567890"
  }'
```

## Product Management

### Create Product

```bash
curl -X POST http://localhost:8080/api/v1/products \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "sku": "PROD-001",
    "name": "Premium Widget",
    "description": "A high-quality widget for all your needs",
    "category_id": "550e8400-e29b-41d4-a716-446655440000",
    "price": 29.99,
    "cost": 15.50,
    "weight": 1.5,
    "track_inventory": true,
    "stock_quantity": 100,
    "min_stock_level": 10
  }'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "sku": "PROD-001",
  "name": "Premium Widget",
  "description": "A high-quality widget for all your needs",
  "category_id": "550e8400-e29b-41d4-a716-446655440000",
  "category": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Widgets",
    "path": "Electronics/Widgets"
  },
  "price": 29.99,
  "cost": 15.50,
  "weight": 1.5,
  "track_inventory": true,
  "stock_quantity": 100,
  "min_stock_level": 10,
  "is_active": true,
  "is_featured": false,
  "created_at": "2024-01-08T10:30:00Z",
  "updated_at": "2024-01-08T10:30:00Z"
}
```

### List Products with Filtering

```bash
curl -X GET "http://localhost:8080/api/v1/products?page=1&limit=20&category_id=550e8400-e29b-41d4-a716-446655440000&is_active=true&search=widget" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Search Products

```bash
curl -X GET "http://localhost:8080/api/v1/products/search?q=widget&min_price=10&max_price=50&in_stock=true" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Update Product

```bash
curl -X PUT http://localhost:8080/api/v1/products/550e8400-e29b-41d4-a716-446655440001 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Premium Widget Pro",
    "price": 34.99,
    "is_featured": true
  }'
```

### Delete Product

```bash
curl -X DELETE http://localhost:8080/api/v1/products/550e8400-e29b-41d4-a716-446655440001 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Category Management

### Create Category

```bash
curl -X POST http://localhost:8080/api/v1/categories \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Smartphones",
    "description": "Mobile phones and accessories",
    "parent_id": "550e8400-e29b-41d4-a716-446655440000",
    "sort_order": 1
  }'
```

### Get Category Tree

```bash
curl -X GET http://localhost:8080/api/v1/categories/tree \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Response:
```json
{
  "tree": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Electronics",
      "path": "Electronics",
      "level": 1,
      "children": [
        {
          "id": "550e8400-e29b-41d4-a716-446655440001",
          "name": "Smartphones",
          "path": "Electronics/Smartphones",
          "level": 2,
          "children": []
        }
      ]
    }
  ]
}
```

## Order Management

### Create Order

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "SALES",
    "currency": "USD",
    "shipping_method": "STANDARD",
    "billing_address_id": "550e8400-e29b-41d4-a716-446655440001",
    "shipping_address_id": "550e8400-e29b-41d4-a716-446655440002",
    "items": [
      {
        "product_id": "550e8400-e29b-41d4-a716-446655440003",
        "quantity": 2,
        "unit_price": 29.99
      },
      {
        "product_id": "550e8400-e29b-41d4-a716-446655440004",
        "quantity": 1,
        "unit_price": 49.99
      }
    ],
    "notes": "Please deliver before 5 PM"
  }'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440005",
  "order_number": "ORD-2024-00001",
  "customer_id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_name": "John Doe",
  "type": "SALES",
  "status": "PENDING",
  "currency": "USD",
  "subtotal": 109.97,
  "tax_amount": 10.00,
  "shipping_amount": 5.00,
  "total_amount": 124.97,
  "items": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440006",
      "product_id": "550e8400-e29b-41d4-a716-446655440003",
      "product_name": "Premium Widget",
      "quantity": 2,
      "unit_price": 29.99,
      "total_price": 59.98
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440007",
      "product_id": "550e8400-e29b-41d4-a716-446655440004",
      "product_name": "Deluxe Gadget",
      "quantity": 1,
      "unit_price": 49.99,
      "total_price": 49.99
    }
  ],
  "created_at": "2024-01-08T10:30:00Z"
}
```

### List Orders

```bash
curl -X GET "http://localhost:8080/api/v1/orders?page=1&limit=20&status=PENDING&customer_id=550e8400-e29b-41d4-a716-446655440000" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Update Order Status

```bash
curl -X PUT http://localhost:8080/api/v1/orders/550e8400-e29b-41d4-a716-446655440005/status \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "PROCESSING",
    "notes": "Order is being prepared"
  }'
```

### Cancel Order

```bash
curl -X POST http://localhost:8080/api/v1/orders/550e8400-e29b-41d4-a716-446655440005/cancel \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Customer requested cancellation",
    "refund_payment": true,
    "restock_items": true,
    "notify_customer": true
  }'
```

## Inventory Management

### Check Inventory Availability

```bash
curl -X POST http://localhost:8080/api/v1/inventory/check-availability \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "550e8400-e29b-41d4-a716-446655440003",
    "quantity": 5
  }'
```

Response:
```json
{
  "product_id": "550e8400-e29b-41d4-a716-446655440003",
  "requested_qty": 5,
  "available": true,
  "can_fulfill": true,
  "backorder_allowed": false
}
```

### Adjust Inventory

```bash
curl -X POST http://localhost:8080/api/v1/inventory/adjust \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "550e8400-e29b-41d4-a716-446655440003",
    "warehouse_id": "550e8400-e29b-41d4-a716-446655440008",
    "adjustment": 50,
    "reason": "Received new shipment",
    "reference_type": "purchase",
    "reference_id": "PO-2024-00001"
  }'
```

### Bulk Inventory Adjustment

```bash
curl -X POST http://localhost:8080/api/v1/inventory/bulk-adjust \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "adjustments": [
      {
        "product_id": "550e8400-e29b-41d4-a716-446655440003",
        "warehouse_id": "550e8400-e29b-41d4-a716-446655440008",
        "adjustment": 50,
        "reason": "Received shipment"
      },
      {
        "product_id": "550e8400-e29b-41d4-a716-446655440004",
        "warehouse_id": "550e8400-e29b-41d4-a716-446655440008",
        "adjustment": 25,
        "reason": "Received shipment"
      }
    ],
    "dry_run": false
  }'
```

### Get Inventory Statistics

```bash
curl -X GET http://localhost:8080/api/v1/inventory/stats \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Response:
```json
{
  "total_products": 150,
  "total_warehouses": 3,
  "total_stock_quantity": 5000,
  "total_inventory_value": 125000.00,
  "low_stock_items": 12,
  "out_of_stock_items": 3,
  "total_reservations": 250,
  "top_products_by_value": [
    {
      "product_id": "550e8400-e29b-41d4-a716-446655440003",
      "product_name": "Premium Widget",
      "total_value": 15000.00,
      "quantity": 500
    }
  ],
  "top_warehouses_by_stock": [
    {
      "warehouse_id": "550e8400-e29b-41d4-a716-446655440008",
      "warehouse_name": "Main Warehouse",
      "total_quantity": 3000
    }
  ]
}
```

## Error Handling Examples

### Handling Validation Errors

```javascript
try {
  const response = await fetch('http://localhost:8080/api/v1/products', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      name: 'Test Product',
      // Missing required fields
    })
  });

  if (!response.ok) {
    const error = await response.json();
    
    if (error.code === 'VALIDATION_ERROR') {
      // Display field-specific errors
      Object.entries(error.details).forEach(([field, errors]) => {
        console.error(`${field}: ${errors.join(', ')}`);
      });
    }
  }
} catch (error) {
  console.error('Network error:', error);
}
```

### Handling Rate Limits

```javascript
async function makeRequestWithRetry(url, options, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await fetch(url, options);
      
      if (response.status === 429) {
        const retryAfter = response.headers.get('Retry-After') || Math.pow(2, i);
        console.log(`Rate limited. Retrying after ${retryAfter} seconds...`);
        await new Promise(resolve => setTimeout(resolve, retryAfter * 1000));
        continue;
      }
      
      return response;
    } catch (error) {
      if (i === maxRetries - 1) throw error;
    }
  }
}
```

### Handling Token Expiration

```javascript
async function makeAuthenticatedRequest(url, options) {
  let token = localStorage.getItem('access_token');
  
  const response = await fetch(url, {
    ...options,
    headers: {
      ...options.headers,
      'Authorization': `Bearer ${token}`
    }
  });
  
  if (response.status === 401) {
    const error = await response.json();
    
    if (error.code === 'TOKEN_EXPIRED') {
      // Refresh token
      const refreshToken = localStorage.getItem('refresh_token');
      const refreshResponse = await fetch('http://localhost:8080/api/v1/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken })
      });
      
      if (refreshResponse.ok) {
        const data = await refreshResponse.json();
        localStorage.setItem('access_token', data.access_token);
        
        // Retry original request
        return makeAuthenticatedRequest(url, options);
      }
    }
  }
  
  return response;
}
```

## Pagination Example

```javascript
async function fetchAllProducts() {
  const allProducts = [];
  let page = 1;
  let hasMore = true;
  
  while (hasMore) {
    const response = await fetch(
      `http://localhost:8080/api/v1/products?page=${page}&limit=100`,
      {
        headers: { 'Authorization': `Bearer ${token}` }
      }
    );
    
    const data = await response.json();
    allProducts.push(...data.products);
    
    hasMore = data.pagination.has_next;
    page++;
  }
  
  return allProducts;
}
```

## Testing with Swagger UI

1. Navigate to http://localhost:8080/api/docs
2. Click "Authorize" button at the top
3. Enter your JWT token in the format: `Bearer YOUR_TOKEN`
4. Click "Authorize" and then "Close"
5. All subsequent requests will include the authentication token
6. Try out endpoints directly from the UI

## Additional Resources

- [Error Codes Reference](ERROR_CODES.md)
- [API Documentation](API_DOCUMENTATION.md)
- [Authentication Guide](AUTHENTICATION.md)
- [Swagger UI](http://localhost:8080/api/docs)
