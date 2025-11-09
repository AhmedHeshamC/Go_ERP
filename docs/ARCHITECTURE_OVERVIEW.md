# ERPGo Architecture Overview

## Executive Summary

ERPGo is a modern, cloud-native Enterprise Resource Planning system built with Go, designed for high performance, scalability, and maintainability. This document provides a comprehensive overview of the system's architecture, design principles, and technical implementation.

## Table of Contents

1. [Architectural Vision](#architectural-vision)
2. [System Architecture](#system-architecture)
3. [Design Principles](#design-principles)
4. [Domain-Driven Design](#domain-driven-design)
5. [Microservices Architecture](#microservices-architecture)
6. [Data Architecture](#data-architecture)
7. [Security Architecture](#security-architecture)
8. [Performance Architecture](#performance-architecture)
9. [Scalability Architecture](#scalability-architecture)
10. [Technology Stack](#technology-stack)
11. [API Design](#api-design)
12. [Integration Patterns](#integration-patterns)
13. [Deployment Architecture](#deployment-architecture)
14. [Monitoring and Observability](#monitoring-and-observability)

## Architectural Vision

### Core Principles

- **Simplicity**: Keep the architecture simple and understandable
- **Scalability**: Design for horizontal scaling from day one
- **Performance**: Optimize for high throughput and low latency
- **Reliability**: Build fault-tolerant systems with high availability
- **Security**: Implement security by design and default
- **Maintainability**: Write clean, testable, and maintainable code
- **Flexibility**: Design for extensibility and adaptability

### Business Goals

- Support 10,000+ concurrent users
- Process 1M+ orders per day
- Achieve 99.9% uptime
- Sub-second response times for 95% of requests
- Global deployment capability
- Multi-tenant architecture support

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Layer                             │
├─────────────────────────────────────────────────────────────────┤
│  Web App  │  Mobile App  │  API Clients  │  Third-party Integrations │
├─────────────────────────────────────────────────────────────────┤
│                        API Gateway                              │
├─────────────────────────────────────────────────────────────────┤
│                      Service Layer                              │
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│ │   User      │ │  Product    │ │  Inventory  │ │   Order     │ │
│ │   Service   │ │  Service    │ │   Service   │ │   Service   │ │
│ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ │
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│ │   Payment   │ │  Notification│ │  Analytics  │ │   Export    │ │
│ │   Service   │ │   Service   │ │   Service   │ │   Service   │ │
│ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                    Infrastructure Layer                         │
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│ │ PostgreSQL  │ │    Redis    │ │  File Store │ │ Message     │ │
│ │  Database   │ │    Cache    │ │  (S3/MinIO) │ │   Queue     │ │
│ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Component Interaction

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   API Gateway   │────│   Load Balancer │────│   Web Server    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                ┌───────────────┼───────────────┐
                │               │               │
        ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
        │  User Auth  │ │  Order API  │ │  Product API│
        └─────────────┘ └─────────────┘ └─────────────┘
                │               │               │
                └───────────────┼───────────────┘
                                │
                        ┌─────────────┐
                        │   Database   │
                        └─────────────┘
```

## Design Principles

### SOLID Principles

1. **Single Responsibility Principle (SRP)**
   - Each service has a single, well-defined responsibility
   - Database schemas are separated by domain
   - Clear separation of concerns between layers

2. **Open/Closed Principle (OCP)**
   - Interfaces are designed for extension without modification
   - Plugin architecture for notification channels
   - Configurable export formats

3. **Liskov Substitution Principle (LSP)**
   - Interface implementations are interchangeable
   - Consistent behavior across service implementations
   - Proper inheritance hierarchies

4. **Interface Segregation Principle (ISP)**
   - Focused, cohesive interfaces
   - Clients depend only on interfaces they use
   - Minimal interface surface area

5. **Dependency Inversion Principle (DIP)**
   - High-level modules don't depend on low-level modules
   - Both depend on abstractions
   - Dependency injection throughout the system

### Clean Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Presentation Layer                       │
│                   (HTTP Handlers, GraphQL, gRPC)                 │
├─────────────────────────────────────────────────────────────────┤
│                        Application Layer                        │
│                    (Use Cases, Business Logic)                  │
├─────────────────────────────────────────────────────────────────┤
│                          Domain Layer                           │
│                (Entities, Value Objects, Services)               │
├─────────────────────────────────────────────────────────────────┤
│                     Infrastructure Layer                        │
│           (Database, External APIs, File System, Cache)         │
└─────────────────────────────────────────────────────────────────┘
```

## Domain-Driven Design (DDD)

### Bounded Contexts

1. **User Management Context**
   - Customer registration and authentication
   - User roles and permissions
   - Profile management

2. **Product Management Context**
   - Product catalog management
   - Pricing and inventory
   - Categories and attributes

3. **Inventory Management Context**
   - Stock level tracking
   - Warehouse management
   - Stock movements and adjustments

4. **Order Management Context**
   - Order lifecycle management
   - Order processing workflows
   - Order fulfillment

5. **Payment Context**
   - Payment processing
   - Refund management
   - Payment methods

### Domain Models

#### Order Aggregate

```go
type Order struct {
    ID               uuid.UUID
    OrderNumber      string
    CustomerID       uuid.UUID
    Status           OrderStatus
    Priority         OrderPriority
    Type             OrderType
    Currency         string
    Items            []OrderItem
    ShippingAddress  *Address
    BillingAddress   *Address
    TotalAmount      decimal.Decimal
    TaxAmount        decimal.Decimal
    ShippingAmount   decimal.Decimal
    DiscountAmount   decimal.Decimal
    PaymentStatus    PaymentStatus
    PaymentMethod    string
    TrackingNumber   *string
    Notes            *string
    InternalNotes    *string
    CreatedAt        time.Time
    UpdatedAt        time.Time
    OrderDate        time.Time
    ShippingDate     *time.Time
    DeliveryDate     *time.Time
    CancelledDate    *time.Time
    CreatedBy        uuid.UUID
    UpdatedBy        uuid.UUID
}

type OrderItem struct {
    ID               uuid.UUID
    OrderID          uuid.UUID
    ProductID        uuid.UUID
    ProductSKU       string
    ProductName      string
    Quantity         int
    UnitPrice        decimal.Decimal
    TotalPrice       decimal.Decimal
    Weight           float64
    Dimensions       *Dimensions
    Status           string
    QuantityShipped  int
    QuantityReturned int
    Notes            *string
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

#### Customer Aggregate

```go
type Customer struct {
    ID           uuid.UUID
    FirstName    string
    LastName     string
    Email        string
    Phone        *string
    Company      *string
    Type         CustomerType
    Status       CustomerStatus
    Addresses    []Address
    Preferences  CustomerPreferences
    Metadata     map[string]interface{}
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type Address struct {
    ID        uuid.UUID
    Type      AddressType
    Street    string
    City      string
    State     string
    PostalCode string
    Country   string
    IsDefault bool
}
```

## Microservices Architecture

### Service Boundaries

```
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│   User Service  │  │ Product Service │  │ Order Service   │
│                 │  │                 │  │                 │
│ • Registration  │  │ • Catalog       │  │ • Order Mgmt    │
│ • Authentication│  │ • Pricing       │  │ • Processing    │
│ • Authorization │  │ • Inventory     │  │ • Fulfillment   │
│ • Profiles      │  │ • Categories    │  │ • Tracking      │
└─────────────────┘  └─────────────────┘  └─────────────────┘

┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│ Payment Service │  │ Notification   │  │ Analytics       │
│                 │  │ Service         │  │ Service         │
│ • Processing    │  │ • Email         │  │ • Metrics       │
│ • Refunds       │  │ • SMS           │  │ • Reports       │
│ • Methods       │  │ • Push          │  │ • Dashboard     │
│ • Transactions  │  │ • Webhooks      │  │ • Forecasts     │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

### Inter-Service Communication

#### Synchronous Communication

```go
// Direct service calls for real-time requirements
type OrderService interface {
    CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error)
    GetOrder(ctx context.Context, id uuid.UUID) (*Order, error)
    UpdateOrderStatus(ctx context.Context, id uuid.UUID, status OrderStatus) error
}

// Circuit breaker pattern for fault tolerance
type ServiceClient struct {
    client    *http.Client
    breaker   *circuit.Breaker
    endpoint  string
    retries   int
    timeout   time.Duration
}
```

#### Asynchronous Communication

```go
// Event-driven architecture
type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(ctx context.Context, eventType string, handler EventHandler) error
}

type OrderEvent struct {
    EventID   uuid.UUID `json:"event_id"`
    EventType string    `json:"event_type"`
    Aggregate string    `json:"aggregate"`
    AggregateID uuid.UUID `json:"aggregate_id"`
    Data      interface{} `json:"data"`
    Metadata  map[string]interface{} `json:"metadata"`
    Timestamp time.Time `json:"timestamp"`
}
```

## Data Architecture

### Database Design

#### Primary Database (PostgreSQL)

```sql
-- Core business entities
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    company VARCHAR(100),
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number VARCHAR(50) UNIQUE NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id),
    status VARCHAR(20) NOT NULL,
    priority VARCHAR(10) NOT NULL DEFAULT 'NORMAL',
    type VARCHAR(20) NOT NULL DEFAULT 'SALES',
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    total_amount DECIMAL(15,2) NOT NULL,
    tax_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    shipping_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    payment_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    payment_method VARCHAR(50),
    tracking_number VARCHAR(100),
    notes TEXT,
    internal_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    order_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    shipping_date TIMESTAMP WITH TIME ZONE,
    delivery_date TIMESTAMP WITH TIME ZONE,
    cancelled_date TIMESTAMP WITH TIME ZONE,
    created_by UUID,
    updated_by UUID
);

-- Indexes for performance
CREATE INDEX idx_orders_customer_id ON orders(customer_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at);
CREATE INDEX idx_orders_order_date ON orders(order_date);
CREATE INDEX idx_customers_email ON customers(email);
```

#### Cache Layer (Redis)

```go
type CacheService struct {
    client redis.Client
}

// Multi-level caching strategy
func (cs *CacheService) GetOrder(ctx context.Context, id uuid.UUID) (*Order, error) {
    // L1: Memory cache
    if order, found := cs.memoryCache.Get(id); found {
        return order.(*Order), nil
    }

    // L2: Redis cache
    data, err := cs.client.Get(ctx, fmt.Sprintf("order:%s", id)).Result()
    if err == nil {
        var order Order
        if err := json.Unmarshal([]byte(data), &order); err == nil {
            cs.memoryCache.Set(id, &order, time.Minute*5)
            return &order, nil
        }
    }

    // L3: Database
    order, err := cs.orderRepo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // Cache for future requests
    data, _ = json.Marshal(order)
    cs.client.Set(ctx, fmt.Sprintf("order:%s", id), data, time.Hour)
    cs.memoryCache.Set(id, order, time.Minute*5)

    return order, nil
}
```

### Data Consistency

#### Eventual Consistency Pattern

```go
type EventualConsistencyManager struct {
    eventBus    EventBus
    eventStore  EventStore
    sagas       map[string]Saga
}

type Saga interface {
    Start(ctx context.Context, event Event) error
    Handle(ctx context.Context, event Event) error
    Compensate(ctx context.Context, event Event) error
}

// Order processing saga
type OrderProcessingSaga struct {
    orderService     OrderService
    inventoryService InventoryService
    paymentService   PaymentService
    notificationService NotificationService
}

func (s *OrderProcessingSaga) Start(ctx context.Context, event Event) error {
    orderCreated := event.Data.(*OrderCreatedEvent)

    // Step 1: Reserve inventory
    if err := s.inventoryService.ReserveInventory(ctx, orderCreated.OrderID, orderCreated.Items); err != nil {
        return err
    }

    // Step 2: Process payment
    if err := s.paymentService.ProcessPayment(ctx, orderCreated.OrderID, orderCreated.Amount); err != nil {
        // Compensate: Release inventory
        s.inventoryService.ReleaseInventory(ctx, orderCreated.OrderID, orderCreated.Items)
        return err
    }

    // Step 3: Confirm order
    return s.orderService.ConfirmOrder(ctx, orderCreated.OrderID)
}
```

## Security Architecture

### Authentication & Authorization

#### JWT-Based Authentication

```go
type AuthService struct {
    jwtSecret     []byte
    tokenExpiry   time.Duration
    refreshTokenExpiry time.Duration
}

type Claims struct {
    UserID   uuid.UUID `json:"user_id"`
    Email    string    `json:"email"`
    Roles    []string  `json:"roles"`
    jwt.StandardClaims
}

func (as *AuthService) GenerateToken(user *User) (string, error) {
    claims := &Claims{
        UserID: user.ID,
        Email:  user.Email,
        Roles:  user.Roles,
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(as.tokenExpiry).Unix(),
            IssuedAt:  time.Now().Unix(),
            Subject:   user.ID.String(),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(as.jwtSecret)
}
```

#### Role-Based Access Control (RBAC)

```go
type AuthorizationService struct {
    roleRepo RoleRepository
    permRepo PermissionRepository
}

func (as *AuthorizationService) CanAccess(ctx context.Context, userID uuid.UUID, resource string, action string) bool {
    user, err := as.userRepo.GetByID(ctx, userID)
    if err != nil {
        return false
    }

    for _, role := range user.Roles {
        permissions, err := as.permRepo.GetByRole(ctx, role)
        if err != nil {
            continue
        }

        for _, permission := range permissions {
            if permission.Resource == resource && permission.Action == action {
                return true
            }
        }
    }

    return false
}
```

### Data Encryption

#### Encryption at Rest

```go
type EncryptionService struct {
    masterKey []byte
}

func (es *EncryptionService) Encrypt(data []byte) ([]byte, error) {
    block, err := aes.NewCipher(es.masterKey)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    return gcm.Seal(nonce, nonce, data, nil), nil
}

func (es *EncryptionService) Decrypt(data []byte) ([]byte, error) {
    block, err := aes.NewCipher(es.masterKey)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return nil, fmt.Errorf("ciphertext too short")
    }

    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    return gcm.Open(nil, nonce, ciphertext, nil)
}
```

## Performance Architecture

### Connection Pooling

```go
type DatabaseManager struct {
    primaryDB *sql.DB
    replicas  []*sql.DB
    pool      *ConnectionPool
}

type ConnectionPool struct {
    maxOpenConns    int
    maxIdleConns    int
    connMaxLifetime time.Duration
    connMaxIdleTime time.Duration
}

func (dm *DatabaseManager) Initialize() error {
    // Primary database configuration
    dm.primaryDB.SetMaxOpenConns(dm.pool.maxOpenConns)
    dm.primaryDB.SetMaxIdleConns(dm.pool.maxIdleConns)
    dm.primaryDB.SetConnMaxLifetime(dm.pool.connMaxLifetime)
    dm.primaryDB.SetConnMaxIdleTime(dm.pool.connMaxIdleTime)

    // Replica configuration for read queries
    for _, replica := range dm.replicas {
        replica.SetMaxOpenConns(dm.pool.maxOpenConns / 2)
        replica.SetMaxIdleConns(dm.pool.maxIdleConns / 2)
    }

    return nil
}
```

### Caching Strategy

```go
type CacheManager struct {
    l1Cache *sync.Map           // Memory cache
    l2Cache *redis.Client       // Redis cache
    l3Cache *memcached.Client   // Memcached cache
}

func (cm *CacheManager) Get(key string) (interface{}, bool) {
    // Try L1 cache first
    if value, ok := cm.l1Cache.Load(key); ok {
        return value, true
    }

    // Try L2 cache
    if value, err := cm.l2Cache.Get(key).Result(); err == nil {
        var result interface{}
        if err := json.Unmarshal([]byte(value), &result); err == nil {
            cm.l1Cache.Store(key, result)
            return result, true
        }
    }

    // Try L3 cache
    if value, err := cm.l3Cache.Get(key); err == nil {
        var result interface{}
        if err := json.Unmarshal(value, &result); err == nil {
            cm.l1Cache.Store(key, result)
            cm.l2Cache.Set(key, value, time.Hour)
            return result, true
        }
    }

    return nil, false
}
```

## Scalability Architecture

### Horizontal Scaling

```go
type LoadBalancer interface {
    AddInstance(instance *ServiceInstance) error
    RemoveInstance(instanceID string) error
    GetHealthyInstances() []*ServiceInstance
    RouteRequest(req *http.Request) (*ServiceInstance, error)
}

type ServiceInstance struct {
    ID       string
    Address  string
    Port     int
    Health   HealthStatus
    Weight   int
    Metadata map[string]string
}

type HealthCheckService struct {
    instances map[string]*ServiceInstance
    checker   HealthChecker
}

func (hcs *HealthCheckService) PerformHealthChecks() {
    for id, instance := range hcs.instances {
        go func(id string, instance *ServiceInstance) {
            healthy := hcs.checker.Check(instance.Address, instance.Port)
            instance.Health = NewHealthStatus(healthy)
        }(id, instance)
    }
}
```

### Auto-Scaling

```go
type AutoScaler struct {
    minInstances int
    maxInstances int
    targetCPU    float64
    targetMemory float64
    cooldown     time.Duration
    instances    []*ServiceInstance
    metrics      MetricsCollector
}

func (as *AutoScaler) EvaluateScaling() {
    metrics := as.metrics.GetSystemMetrics()

    shouldScaleUp := metrics.CPUUsage > as.targetCPU || metrics.MemoryUsage > as.targetMemory
    shouldScaleDown := metrics.CPUUsage < as.targetCPU*0.7 && metrics.MemoryUsage < as.targetMemory*0.7

    if shouldScaleUp && len(as.instances) < as.maxInstances {
        as.ScaleUp()
    } else if shouldScaleDown && len(as.instances) > as.minInstances {
        as.ScaleDown()
    }
}
```

## Technology Stack

### Backend Technologies

- **Go 1.21+**: Primary programming language
- **Gin**: HTTP web framework
- **GORM**: ORM for database operations
- **PostgreSQL**: Primary database
- **Redis**: Caching and session storage
- **NATS**: Message queue for events
- **Elasticsearch**: Search and analytics

### Infrastructure Technologies

- **Docker**: Containerization
- **Kubernetes**: Container orchestration
- **Helm**: Package management
- **Prometheus**: Metrics collection
- **Grafana**: Visualization and dashboards
- **Jaeger**: Distributed tracing
- **ELK Stack**: Logging and monitoring

### Development Tools

- **Swagger/OpenAPI**: API documentation
- **gRPC/Protobuf**: Inter-service communication
- **GraphQL**: API query language
- **Testify**: Testing framework
- **Mockery**: Mock generation
- **GolangCI-Lint**: Code linting

## API Design

### RESTful API Design

```go
// API structure following REST conventions
type API struct {
    router          *gin.Engine
    userService     UserService
    orderService    OrderService
    productService  ProductService
    authMiddleware  gin.HandlerFunc
    rateLimit       gin.HandlerFunc
}

func (api *API) SetupRoutes() {
    v1 := api.router.Group("/api/v1")

    // Authentication routes
    auth := v1.Group("/auth")
    {
        auth.POST("/login", api.authMiddleware, api.Login)
        auth.POST("/logout", api.authMiddleware, api.Logout)
        auth.POST("/refresh", api.authMiddleware, api.RefreshToken)
    }

    // Customer routes
    customers := v1.Group("/customers")
    customers.Use(api.authMiddleware)
    customers.Use(api.rateLimit)
    {
        customers.POST("", api.CreateCustomer)
        customers.GET("/:id", api.GetCustomer)
        customers.PUT("/:id", api.UpdateCustomer)
        customers.DELETE("/:id", api.DeleteCustomer)
        customers.GET("", api.ListCustomers)
    }

    // Order routes
    orders := v1.Group("/orders")
    orders.Use(api.authMiddleware)
    orders.Use(api.rateLimit)
    {
        orders.POST("", api.CreateOrder)
        orders.GET("/:id", api.GetOrder)
        orders.PUT("/:id", api.UpdateOrder)
        orders.DELETE("/:id", api.DeleteOrder)
        orders.GET("", api.ListOrders)
        orders.PUT("/:id/status", api.UpdateOrderStatus)
        orders.POST("/:id/cancel", api.CancelOrder)
    }
}
```

### GraphQL API Design

```go
type Query struct {
    Customer   *CustomerResolver   `json:"customer"`
    Order      *OrderResolver      `json:"order"`
    Product    *ProductResolver    `json:"product"`
    Orders     []*OrderResolver    `json:"orders"`
    Customers  []*CustomerResolver `json:"customers"`
}

type Mutation struct {
    CreateCustomer *CustomerResolver `json:"createCustomer"`
    UpdateCustomer *CustomerResolver `json:"updateCustomer"`
    DeleteCustomer bool              `json:"deleteCustomer"`
    CreateOrder    *OrderResolver    `json:"createOrder"`
    UpdateOrder    *OrderResolver    `json:"updateOrder"`
    CancelOrder    *OrderResolver    `json:"cancelOrder"`
}
```

## Integration Patterns

### API Gateway Pattern

```go
type APIGateway struct {
    routes         map[string]*Route
    middlewares    []gin.HandlerFunc
    loadBalancer   LoadBalancer
    circuitBreaker CircuitBreaker
}

type Route struct {
    Path        string
    Method      string
    Service     string
    Middlewares []gin.HandlerFunc
    Timeout     time.Duration
}

func (ag *APIGateway) ProxyRequest(c *gin.Context) {
    route := ag.findRoute(c.Request.URL.Path, c.Request.Method)
    if route == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Route not found"})
        return
    }

    // Apply circuit breaker
    if !ag.circuitBreaker.CanExecute(route.Service) {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
        return
    }

    // Load balance request
    instance := ag.loadBalancer.GetHealthyInstance(route.Service)
    if instance == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No healthy instances"})
        return
    }

    // Proxy request to service
    ag.proxyRequest(c, instance, route)
}
```

### Event-Driven Architecture

```go
type EventBus struct {
    publishers map[string]Publisher
    subscribers map[string][]Subscriber
    middlewares []EventMiddleware
}

type Event struct {
    ID          uuid.UUID              `json:"id"`
    Type        string                 `json:"type"`
    Source      string                 `json:"source"`
    Subject     string                 `json:"subject"`
    Data        interface{}            `json:"data"`
    Timestamp   time.Time              `json:"timestamp"`
    Metadata    map[string]interface{} `json:"metadata"`
}

func (eb *EventBus) Publish(ctx context.Context, event Event) error {
    // Apply middlewares
    for _, middleware := range eb.middlewares {
        if err := middleware(ctx, &event); err != nil {
            return err
        }
    }

    // Publish to subscribers
    if subscribers, exists := eb.subscribers[event.Type]; exists {
        for _, subscriber := range subscribers {
            go subscriber.Handle(ctx, event)
        }
    }

    return nil
}
```

## Deployment Architecture

### Container Strategy

```dockerfile
# Multi-stage build for optimal image size
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o erpgo cmd/api/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/erpgo .
COPY --from=builder /app/config ./config

EXPOSE 8080
CMD ["./erpgo"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: erpgo-api
  labels:
    app: erpgo-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: erpgo-api
  template:
    metadata:
      labels:
        app: erpgo-api
    spec:
      containers:
      - name: erpgo-api
        image: erpgo/api:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: "postgres-service"
        - name: REDIS_HOST
          value: "redis-service"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: erpgo-api-service
spec:
  selector:
    app: erpgo-api
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

## Monitoring and Observability

### Metrics Collection

```go
type MetricsCollector struct {
    requestCounter   prometheus.Counter
    requestDuration  prometheus.Histogram
    errorCounter     prometheus.Counter
    activeConnections prometheus.Gauge
}

func (mc *MetricsCollector) RecordRequest(method, path string, statusCode int, duration time.Duration) {
    mc.requestCounter.WithLabelValues(method, path, strconv.Itoa(statusCode)).Inc()
    mc.requestDuration.WithLabelValues(method, path).Observe(duration.Seconds())

    if statusCode >= 400 {
        mc.errorCounter.WithLabelValues(method, path, strconv.Itoa(statusCode)).Inc()
    }
}
```

### Distributed Tracing

```go
type TracingService struct {
    tracer opentracing.Tracer
}

func (ts *TracingService) StartSpan(operationName string, ctx context.Context) (opentracing.Span, context.Context) {
    return ts.tracer.StartSpanFromContext(ctx, operationName)
}

func (ts *TracingService) TraceMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        span, ctx := ts.StartSpan(c.Request.URL.Path, c.Request.Context())
        defer span.Finish()

        c.Request = c.Request.WithContext(ctx)
        c.Next()

        span.SetTag("http.method", c.Request.Method)
        span.SetTag("http.status_code", c.Writer.Status())
        span.SetTag("http.url", c.Request.URL.String())
    }
}
```

### Health Checks

```go
type HealthCheckService struct {
    checks map[string]HealthCheck
}

type HealthCheck interface {
    Name() string
    Check(ctx context.Context) HealthStatus
}

type HealthStatus struct {
    Status  string                 `json:"status"`
    Message string                 `json:"message"`
    Details map[string]interface{} `json:"details"`
    Timestamp time.Time            `json:"timestamp"`
}

func (hcs *HealthCheckService) RunChecks(ctx context.Context) map[string]HealthStatus {
    results := make(map[string]HealthStatus)

    for name, check := range hcs.checks {
        status := check.Check(ctx)
        results[name] = status
    }

    return results
}
```

## Conclusion

The ERPGo architecture is designed to be:

- **Scalable**: Can handle growing workloads through horizontal scaling
- **Resilient**: Built with fault tolerance and high availability
- **Maintainable**: Clean architecture with clear separation of concerns
- **Secure**: Comprehensive security measures at all layers
- **Performant**: Optimized for high throughput and low latency
- **Observable**: Complete monitoring and tracing capabilities

This architecture provides a solid foundation for enterprise-grade ERP operations while maintaining flexibility for future enhancements and adaptations.