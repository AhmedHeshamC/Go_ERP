# Order Service Implementation

This package provides a comprehensive order management service for the ERPGo system. It implements the full order lifecycle including creation, validation, payment processing, fulfillment, and analytics.

## Architecture

The order service follows clean architecture principles with clear separation of concerns:

- **Service Layer**: Business logic and orchestration
- **Repository Layer**: Data access abstraction
- **Entity Layer**: Domain models and business rules
- **Integration Layer**: External service adapters

## Key Features

### Order Management
- ✅ Order creation with inventory validation and reservation
- ✅ Order status management with state machine integration
- ✅ Order modification and cancellation
- ✅ Order cloning and archiving
- ✅ Bulk operations for batch processing

### Payment Processing
- ✅ Payment processing and validation
- ✅ Refund processing (full and partial)
- ✅ Multiple payment methods support
- ✅ Transaction tracking and reconciliation

### Fulfillment Workflow
- ✅ Order processing and shipping
- ✅ Partial shipment support
- ✅ Delivery tracking
- ✅ Return and refund processing

### Inventory Integration
- ✅ Real-time inventory availability checking
- ✅ Inventory reservation on order creation
- ✅ Inventory consumption on fulfillment
- ✅ Backorder handling

### Pricing and Taxation
- ✅ Dynamic pricing calculation
- ✅ Tax calculation with multiple rates
- ✅ Discount and coupon support
- ✅ Shipping cost calculation

### Analytics and Reporting
- ✅ Order statistics and metrics
- ✅ Revenue analysis by period
- ✅ Customer analytics
- ✅ Product sales reporting
- ✅ Top customers and products

## Service Interface

The service provides a comprehensive interface for order management:

```go
type Service interface {
    // Order management
    CreateOrder(ctx context.Context, req *CreateOrderRequest) (*entities.Order, error)
    GetOrder(ctx context.Context, id string) (*entities.Order, error)
    ListOrders(ctx context.Context, req *ListOrdersRequest) (*ListOrdersResponse, error)
    UpdateOrder(ctx context.Context, id string, req *UpdateOrderRequest) (*entities.Order, error)
    DeleteOrder(ctx context.Context, id string) error

    // Order status management
    UpdateOrderStatus(ctx context.Context, id string, req *UpdateOrderStatusRequest) (*entities.Order, error)
    CancelOrder(ctx context.Context, id string, req *CancelOrderRequest) (*entities.Order, error)
    ApproveOrder(ctx context.Context, id string, approvedBy string) (*entities.Order, error)

    // Fulfillment
    ShipOrder(ctx context.Context, id string, req *ShipOrderRequest) (*entities.Order, error)
    DeliverOrder(ctx context.Context, id string, req *DeliverOrderRequest) (*entities.Order, error)

    // Payment processing
    ProcessPayment(ctx context.Context, id string, req *ProcessPaymentRequest) (*entities.Order, error)
    RefundOrder(ctx context.Context, id string, req *RefundOrderRequest) (*entities.Order, error)

    // Analytics
    GetOrderAnalytics(ctx context.Context, req *GetOrderAnalyticsRequest) (*OrderAnalyticsResponse, error)
    GetOrderStats(ctx context.Context, req *GetOrderStatsRequest) (*repositories.OrderStats, error)

    // ... and many more methods
}
```

## Usage Examples

### Basic Order Creation

```go
factory := NewServiceFactory(db)
container := NewDependencyContainer(
    productService,
    inventoryService,
    userService,
    notificationService,
    paymentService,
    taxCalculator,
    shippingCalculator,
)

orderService := factory.CreateServiceFromContainer(container)

req := &CreateOrderRequest{
    CustomerID:        "customer-uuid",
    Type:              entities.OrderTypeSales,
    ShippingMethod:    entities.ShippingMethodStandard,
    ShippingAddressID: "shipping-address-uuid",
    BillingAddressID:  "billing-address-uuid",
    Currency:          "USD",
    Items: []CreateOrderItemRequest{
        {
            ProductID: "product-uuid",
            Quantity:  2,
            UnitPrice: decimal.NewFromFloat(50.00),
        },
    },
    CreatedBy: "user-uuid",
}

order, err := orderService.CreateOrder(ctx, req)
if err != nil {
    log.Printf("Failed to create order: %v", err)
    return
}

log.Printf("Created order: %s", order.OrderNumber)
```

### Order Status Management

```go
// Approve an order
order, err := orderService.ApproveOrder(ctx, orderID, "approver-uuid")
if err != nil {
    log.Printf("Failed to approve order: %v", err)
    return
}

// Ship an order
shipReq := &ShipOrderRequest{
    TrackingNumber: "1Z999AA10123456784",
    Carrier:        "UPS",
    Notify:         true,
    ShippedBy:      "shipper-uuid",
    Items: []ShipItemRequest{
        {
            ItemID:   "item-uuid",
            Quantity: 2,
        },
    },
}

order, err = orderService.ShipOrder(ctx, orderID, shipReq)
if err != nil {
    log.Printf("Failed to ship order: %v", err)
    return
}
```

### Payment Processing

```go
// Process a payment
paymentReq := &ProcessPaymentRequest{
    Amount:        decimal.NewFromFloat(125.00),
    PaymentMethod: "credit_card",
    TransactionID: "txn_123456789",
    PaymentBy:     "user-uuid",
}

order, err := orderService.ProcessPayment(ctx, orderID, paymentReq)
if err != nil {
    log.Printf("Failed to process payment: %v", err)
    return
}

log.Printf("Payment processed. Order status: %s", order.PaymentStatus)
```

### Analytics

```go
// Get order analytics for the last month
analyticsReq := &GetOrderAnalyticsRequest{
    StartDate: time.Now().AddDate(0, -1, 0),
    EndDate:   time.Now(),
    GroupBy:   "week",
}

response, err := orderService.GetOrderAnalytics(ctx, analyticsReq)
if err != nil {
    log.Printf("Failed to get analytics: %v", err)
    return
}

log.Printf("Total orders: %d", response.OrderStats.TotalOrders)
log.Printf("Total revenue: %s", response.OrderStats.TotalRevenue)
```

## Configuration

The service supports flexible configuration through the `ServiceConfig` struct:

```go
config := &ServiceConfig{
    DefaultCurrency:            "USD",
    EnableTaxCalculation:      true,
    EnableShippingCalculation: true,
    EnableNotifications:       true,
    EnablePaymentProcessing:   true,
}

service := factory.CreateConfiguredService(config, container)
```

## Integration Adapters

The service uses integration adapters to connect with external services:

- **ProductAdapter**: Integrates with product catalog service
- **InventoryAdapter**: Integrates with inventory management system
- **UserAdapter**: Integrates with user management service
- **NotificationAdapter**: Integrates with notification service
- **PaymentAdapter**: Integrates with payment processing service
- **TaxCalculatorAdapter**: Integrates with tax calculation service
- **ShippingCalculatorAdapter**: Integrates with shipping calculation service

## Error Handling

The service provides comprehensive error handling with specific error types:

- `ErrOrderNotFound`: Order not found
- `ErrInvalidOrderStatus`: Invalid order status
- `ErrInvalidStatusTransition`: Invalid status transition
- `ErrInsufficientInventory`: Insufficient inventory
- `ErrCustomerNotFound`: Customer not found
- `ErrPaymentFailed`: Payment processing failed
- `ErrRefundFailed`: Refund processing failed

## Testing

The service includes comprehensive test coverage:

```go
func TestServiceImpl_CreateOrder(t *testing.T) {
    // Setup mocks
    mockOrderRepo := &MockOrderRepository{}
    mockProductService := &MockProductService{}
    mockInventoryService := &MockInventoryService{}

    service := &ServiceImpl{
        orderRepo:        mockOrderRepo,
        productService:   mockProductService,
        inventoryService: mockInventoryService,
    }

    // Test order creation
    req := &CreateOrderRequest{...}
    order, err := service.CreateOrder(ctx, req)

    require.NoError(t, err)
    assert.NotNil(t, order)
    assert.Equal(t, entities.OrderStatusDraft, order.Status)
}
```

## Performance Considerations

### Database Optimizations
- Uses efficient pagination for large result sets
- Implements bulk operations for batch processing
- Includes proper indexing strategies
- Uses connection pooling

### Caching Strategy
- Customer data caching
- Product information caching
- Tax and shipping rate caching
- Analytics result caching

### Inventory Management
- Real-time inventory checking
- Optimistic locking for inventory updates
- Batch inventory operations
- Asynchronous inventory updates

## Security

### Input Validation
- Comprehensive request validation
- SQL injection prevention
- Cross-site scripting (XSS) protection
- Input sanitization

### Access Control
- Role-based access control (RBAC)
- Permission-based operations
- Audit logging
- Secure token handling

### Data Protection
- Sensitive data encryption
- PII protection
- GDPR compliance
- Secure payment processing

## Monitoring and Logging

### Metrics
- Order creation rate
- Payment success rate
- Fulfillment time
- Error rates by category

### Logging
- Structured logging with context
- Operation tracing
- Error tracking
- Performance monitoring

### Health Checks
- Database connectivity
- External service availability
- System resource monitoring
- Queue health (if using async processing)

## Deployment

### Environment Variables
```
ORDER_SERVICE_DEFAULT_CURRENCY=USD
ORDER_SERVICE_ENABLE_TAX_CALCULATION=true
ORDER_SERVICE_ENABLE_SHIPPING_CALCULATION=true
ORDER_SERVICE_ENABLE_NOTIFICATIONS=true
ORDER_SERVICE_ENABLE_PAYMENT_PROCESSING=true
```

### Database Requirements
- PostgreSQL 12+ with JSON support
- Proper indexing on order tables
- Optimized queries for analytics
- Connection pooling configuration

### External Dependencies
- Product Service (HTTP/gRPC)
- Inventory Service (HTTP/gRPC)
- User Service (HTTP/gRPC)
- Payment Gateway (HTTP API)
- Notification Service (HTTP/gRPC)
- Tax Calculation Service (HTTP/gRPC)
- Shipping Calculation Service (HTTP/gRPC)

## Future Enhancements

### Planned Features
- Real-time order tracking
- Advanced analytics dashboard
- Machine learning for fraud detection
- Automated order routing
- Multi-warehouse support
- International shipping
- Subscription billing
- Advanced discount rules

### Performance Improvements
- Read replicas for analytics queries
- Event-driven architecture
- CQRS pattern implementation
- Advanced caching strategies
- GraphQL API support

## Contributing

When contributing to the order service:

1. Follow SOLID principles
2. Write comprehensive tests
3. Update documentation
4. Ensure backward compatibility
5. Add error handling for new features
6. Include performance benchmarks
7. Update integration adapters if needed

## License

This implementation is part of the ERPGo project and follows the same licensing terms.