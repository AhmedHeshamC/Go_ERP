# ERPGo API Documentation

## Overview

ERPGo is a comprehensive Enterprise Resource Planning system built with Go, providing powerful order management, inventory tracking, customer management, and business analytics capabilities.

## Table of Contents

1. [Authentication](#authentication)
2. [Base URL](#base-url)
3. [Common Headers](#common-headers)
4. [Error Handling](#error-handling)
5. [Rate Limiting](#rate-limiting)
6. [API Endpoints](#api-endpoints)
   - [Authentication](#authentication-endpoints)
   - [Customers](#customer-endpoints)
   - [Products](#product-endpoints)
   - [Orders](#order-endpoints)
   - [Analytics](#analytics-endpoints)
   - [Bulk Operations](#bulk-operations-endpoints)
   - [Export](#export-endpoints)

## Authentication

ERPGo uses JWT-based authentication for API access.

### Authentication Flow

1. Obtain a JWT token by logging in with your credentials
2. Include the token in the `Authorization` header of subsequent requests

### Request Example

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "your-password"
}
```

### Response Example

```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "roles": ["user"]
    },
    "expires_at": "2024-01-09T12:00:00Z"
  }
}
```

## Base URL

- **Production**: `https://api.erpgo.example.com`
- **Staging**: `https://api-staging.erpgo.example.com`
- **Development**: `http://localhost:8080`

## Common Headers

All API requests should include these headers:

```http
Content-Type: application/json
Authorization: Bearer <your-jwt-token>
```

## Error Handling

ERPGo uses standard HTTP status codes and follows REST conventions.

### Error Response Format

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": {
      "field": "email",
      "reason": "Invalid email format"
    },
    "timestamp": "2024-01-08T10:30:00Z",
    "request_id": "req_123456789"
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| VALIDATION_ERROR | 400 | Request validation failed |
| UNAUTHORIZED | 401 | Authentication required |
| FORBIDDEN | 403 | Insufficient permissions |
| NOT_FOUND | 404 | Resource not found |
| CONFLICT | 409 | Resource conflict |
| RATE_LIMIT_EXCEEDED | 429 | Too many requests |
| INTERNAL_ERROR | 500 | Server error |
| SERVICE_UNAVAILABLE | 503 | Service temporarily unavailable |

## Rate Limiting

API requests are rate-limited to ensure fair usage:

- **Standard Plan**: 1000 requests per hour
- **Premium Plan**: 5000 requests per hour
- **Enterprise Plan**: 25000 requests per hour

Rate limit headers are included in responses:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1641630000
```

## API Endpoints

### Authentication Endpoints

#### Login

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
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "first_name": "John",
      "last_name": "Doe",
      "roles": ["user"]
    },
    "expires_at": "2024-01-09T12:00:00Z"
  }
}
```

#### Logout

```http
POST /api/v1/auth/logout
```

**Headers:**
```http
Authorization: Bearer <token>
```

#### Refresh Token

```http
POST /api/v1/auth/refresh
```

**Headers:**
```http
Authorization: Bearer <token>
```

### Customer Endpoints

#### Create Customer

```http
POST /api/v1/customers
```

**Request Body:**
```json
{
  "first_name": "John",
  "last_name": "Doe",
  "email": "john.doe@example.com",
  "phone": "+1234567890",
  "type": "INDIVIDUAL",
  "addresses": [
    {
      "type": "billing",
      "street": "123 Main St",
      "city": "New York",
      "state": "NY",
      "postal_code": "10001",
      "country": "US"
    }
  ]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "first_name": "John",
    "last_name": "Doe",
    "email": "john.doe@example.com",
    "phone": "+1234567890",
    "type": "INDIVIDUAL",
    "status": "ACTIVE",
    "addresses": [...],
    "created_at": "2024-01-08T10:30:00Z",
    "updated_at": "2024-01-08T10:30:00Z"
  }
}
```

#### Get Customer

```http
GET /api/v1/customers/{id}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "first_name": "John",
    "last_name": "Doe",
    "email": "john.doe@example.com",
    "phone": "+1234567890",
    "type": "INDIVIDUAL",
    "status": "ACTIVE",
    "addresses": [...],
    "orders_count": 5,
    "total_spent": 1250.75,
    "created_at": "2024-01-08T10:30:00Z",
    "updated_at": "2024-01-08T10:30:00Z"
  }
}
```

#### List Customers

```http
GET /api/v1/customers?page=1&limit=20&status=ACTIVE
```

**Query Parameters:**
- `page` (int): Page number (default: 1)
- `limit` (int): Items per page (default: 20, max: 100)
- `status` (string): Filter by status (ACTIVE, INACTIVE, SUSPENDED)
- `search` (string): Search term for name/email
- `created_after` (string): Filter by creation date (ISO 8601)
- `created_before` (string): Filter by creation date (ISO 8601)

**Response:**
```json
{
  "success": true,
  "data": {
    "customers": [...],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 150,
      "total_pages": 8,
      "has_next": true,
      "has_prev": false
    }
  }
}
```

#### Update Customer

```http
PUT /api/v1/customers/{id}
```

**Request Body:**
```json
{
  "first_name": "John Updated",
  "last_name": "Doe",
  "phone": "+1234567890"
}
```

#### Delete Customer

```http
DELETE /api/v1/customers/{id}
```

### Product Endpoints

#### Create Product

```http
POST /api/v1/products
```

**Request Body:**
```json
{
  "name": "Premium Widget",
  "sku": "WIDGET-001",
  "description": "A high-quality widget",
  "price": 29.99,
  "category_id": "550e8400-e29b-41d4-a716-446655440000",
  "weight": 1.5,
  "dimensions": {
    "length": 10.0,
    "width": 8.0,
    "height": 5.0
  },
  "inventory": {
    "stock_level": 100,
    "reorder_point": 10,
    "reorder_quantity": 50
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "Premium Widget",
    "sku": "WIDGET-001",
    "description": "A high-quality widget",
    "price": 29.99,
    "category_id": "550e8400-e29b-41d4-a716-446655440000",
    "weight": 1.5,
    "dimensions": {...},
    "inventory": {...},
    "status": "ACTIVE",
    "created_at": "2024-01-08T10:30:00Z",
    "updated_at": "2024-01-08T10:30:00Z"
  }
}
```

#### Get Product

```http
GET /api/v1/products/{id}
```

#### List Products

```http
GET /api/v1/products?page=1&limit=20&category_id={id}
```

#### Update Product

```http
PUT /api/v1/products/{id}
```

#### Delete Product

```http
DELETE /api/v1/products/{id}
```

### Order Endpoints

#### Create Order

```http
POST /api/v1/orders
```

**Request Body:**
```json
{
  "customer_id": "550e8400-e29b-41d4-a716-446655440000",
  "currency": "USD",
  "shipping_method": "STANDARD",
  "priority": "NORMAL",
  "items": [
    {
      "product_id": "550e8400-e29b-41d4-a716-446655440001",
      "quantity": 2,
      "unit_price": 29.99
    }
  ],
  "shipping_address": {
    "street": "123 Main St",
    "city": "New York",
    "state": "NY",
    "postal_code": "10001",
    "country": "US"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440002",
    "order_number": "ORD-2024-001",
    "customer_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "DRAFT",
    "currency": "USD",
    "total_amount": 59.98,
    "items": [...],
    "shipping_address": {...},
    "created_at": "2024-01-08T10:30:00Z",
    "updated_at": "2024-01-08T10:30:00Z"
  }
}
```

#### Get Order

```http
GET /api/v1/orders/{id}
```

#### List Orders

```http
GET /api/v1/orders?page=1&limit=20&status=PENDING
```

#### Update Order Status

```http
PUT /api/v1/orders/{id}/status
```

**Request Body:**
```json
{
  "status": "CONFIRMED",
  "reason": "Customer payment received"
}
```

#### Cancel Order

```http
POST /api/v1/orders/{id}/cancel
```

**Request Body:**
```json
{
  "reason": "Customer requested cancellation"
}
```

### Analytics Endpoints

#### Get Order Metrics

```http
GET /api/v1/analytics/orders?start_date=2024-01-01&end_date=2024-01-31
```

**Response:**
```json
{
  "success": true,
  "data": {
    "total_orders": 150,
    "completed_orders": 120,
    "cancelled_orders": 10,
    "total_revenue": 15000.00,
    "average_order_value": 100.00,
    "conversion_rate": 3.5,
    "by_status": {
      "DRAFT": 5,
      "PENDING": 15,
      "CONFIRMED": 20,
      "PROCESSING": 30,
      "SHIPPED": 50,
      "DELIVERED": 120,
      "CANCELLED": 10
    },
    "by_date": [
      {
        "date": "2024-01-01",
        "orders": 10,
        "revenue": 1000.00
      }
    ]
  }
}
```

#### Get Revenue Metrics

```http
GET /api/v1/analytics/revenue?start_date=2024-01-01&end_date=2024-01-31&group_by=daily
```

#### Get Customer Analytics

```http
GET /api/v1/analytics/customers?start_date=2024-01-01&end_date=2024-01-31&limit=10
```

#### Get Product Analytics

```http
GET /api/v1/analytics/products?start_date=2024-01-01&end_date=2024-01-31&limit=10
```

#### Generate Sales Report

```http
POST /api/v1/analytics/reports/sales
```

**Request Body:**
```json
{
  "start_date": "2024-01-01",
  "end_date": "2024-01-31",
  "format": "PDF",
  "include_details": true
}
```

### Bulk Operations Endpoints

#### Bulk Status Change

```http
POST /api/v1/bulk/orders/status
```

**Request Body:**
```json
{
  "order_ids": ["id1", "id2", "id3"],
  "new_status": "PROCESSING",
  "reason": "Bulk processing"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "operation_id": "bulk-123",
    "total_orders": 3,
    "success_count": 3,
    "failed_count": 0,
    "failed_orders": [],
    "duration": "2.5s"
  }
}
```

#### Bulk Ship Orders

```http
POST /api/v1/bulk/orders/ship
```

**Request Body:**
```json
{
  "order_ids": ["id1", "id2", "id3"],
  "tracking_info": {
    "id1": {
      "tracking_number": "TRACK123",
      "carrier": "UPS"
    },
    "id2": {
      "tracking_number": "TRACK124",
      "carrier": "FedEx"
    }
  }
}
```

#### Bulk Import Orders

```http
POST /api/v1/bulk/orders/import
Content-Type: multipart/form-data

file: orders.csv
format: csv
```

### Export Endpoints

#### Export Orders to CSV

```http
POST /api/v1/export/orders/csv
```

**Request Body:**
```json
{
  "order_ids": ["id1", "id2", "id3"],
  "fields": ["id", "order_number", "customer_email", "status", "total_amount"],
  "options": {
    "include_items": true,
    "include_customer": true
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "export_id": "export-123",
    "format": "csv",
    "record_count": 3,
    "file_path": "/exports/orders_20240108.csv",
    "download_url": "https://api.erpgo.example.com/downloads/export-123",
    "expires_at": "2024-01-15T10:30:00Z"
  }
}
```

#### Export Orders to JSON

```http
POST /api/v1/export/orders/json
```

#### Export Orders to Excel

```http
POST /api/v1/export/orders/excel
```

## SDK Examples

### Go SDK

```go
import "github.com/erpgo/erpgo-sdk"

client := erpgo.NewClient("https://api.erpgo.example.com", "your-api-key")

// Create a customer
customer, err := client.Customers.Create(&erpgo.Customer{
    FirstName: "John",
    LastName:  "Doe",
    Email:     "john.doe@example.com",
})

// Create an order
order, err := client.Orders.Create(&erpgo.Order{
    CustomerID: customer.ID,
    Currency:   "USD",
    Items: []erpgo.OrderItem{
        {
            ProductID: "product-id",
            Quantity:  2,
        },
    },
})
```

### JavaScript SDK

```javascript
import { ERPGoClient } from '@erpgo/sdk';

const client = new ERPGoClient('https://api.erpgo.example.com', 'your-api-key');

// Create a customer
const customer = await client.customers.create({
    first_name: 'John',
    last_name: 'Doe',
    email: 'john.doe@example.com'
});

// Create an order
const order = await client.orders.create({
    customer_id: customer.id,
    currency: 'USD',
    items: [{
        product_id: 'product-id',
        quantity: 2
    }]
});
```

## Webhooks

ERPGo supports webhooks for real-time event notifications.

### Configure Webhook

```http
POST /api/v1/webhooks
```

**Request Body:**
```json
{
  "url": "https://your-app.com/webhooks/erpgo",
  "events": ["order.created", "order.updated", "customer.created"],
  "secret": "your-webhook-secret"
}
```

### Webhook Payload

```json
{
  "event": "order.created",
  "data": {
    "order": {...}
  },
  "timestamp": "2024-01-08T10:30:00Z",
  "signature": "sha256=..."
}
```

### Verify Webhook Signature

```go
import "crypto/hmac"
import "crypto/sha256"

func verifyWebhookSignature(payload, secret, signature string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(payload))
    expectedMAC := fmt.Sprintf("sha256=%x", mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expectedMAC))
}
```

## Testing

### Test Environment

Use the test environment for development and testing:

- **Base URL**: `https://api-test.erpgo.example.com`
- **Test API Key**: `test-api-key`

### Mock Data

ERPGo provides endpoints for generating mock data:

```http
POST /api/v1/test/data/customers
POST /api/v1/test/data/products
POST /api/v1/test/data/orders
```

## Support

For API support and questions:

- **Documentation**: https://docs.erpgo.example.com
- **API Status**: https://status.erpgo.example.com
- **Support Email**: api-support@erpgo.example.com
- **Developer Forum**: https://community.erpgo.example.com

## Changelog

### v1.0.0 (2024-01-08)
- Initial API release
- Full CRUD operations for customers, products, and orders
- Analytics and reporting endpoints
- Bulk operations support
- Export functionality
- Webhook support
- Comprehensive error handling
- Rate limiting
- Monitoring endpoints