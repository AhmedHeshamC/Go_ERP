# Health Check Package

This package provides comprehensive health check functionality for the ERPGo system, implementing Requirements 8.1-8.5 from the production readiness specification.

## Features

- **Liveness Probe**: Simple check that returns 200 if the application is running
- **Readiness Probe**: Checks database and Redis connectivity before marking service as ready
- **Timeout Protection**: All health checks complete within 1 second maximum
- **Detailed Status**: Returns 503 with detailed information when dependencies are unhealthy
- **Shutdown Awareness**: Returns 503 from readiness probe during graceful shutdown

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "time"
    
    "erpgo/pkg/health"
    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/redis/go-redis/v9"
)

func main() {
    // Create health checker
    checker := health.NewHealthChecker()
    
    // Register database health check
    db := pgxpool.Pool{} // Your database connection
    dbCheck := &DatabaseHealthCheck{pool: db}
    checker.RegisterCheck("database", dbCheck)
    
    // Register Redis health check
    redisClient := redis.Client{} // Your Redis client
    redisCheck := &RedisHealthCheck{client: redisClient}
    checker.RegisterCheck("redis", redisCheck)
    
    // Create HTTP handler
    handler := health.NewHandler(checker)
    
    // Register routes
    router := gin.Default()
    handler.RegisterRoutes(router)
    
    // Start server
    router.Run(":8080")
}
```

### Implementing Custom Health Checks

```go
// DatabaseHealthCheck implements the HealthCheck interface
type DatabaseHealthCheck struct {
    pool *pgxpool.Pool
}

func (d *DatabaseHealthCheck) Name() string {
    return "database"
}

func (d *DatabaseHealthCheck) Check(ctx context.Context) error {
    // Ping the database
    return d.pool.Ping(ctx)
}

func (d *DatabaseHealthCheck) Timeout() time.Duration {
    return 1 * time.Second
}

// RedisHealthCheck implements the HealthCheck interface
type RedisHealthCheck struct {
    client *redis.Client
}

func (r *RedisHealthCheck) Name() string {
    return "redis"
}

func (r *RedisHealthCheck) Check(ctx context.Context) error {
    // Ping Redis
    return r.client.Ping(ctx).Err()
}

func (r *RedisHealthCheck) Timeout() time.Duration {
    return 500 * time.Millisecond
}
```

### Graceful Shutdown Integration

```go
func main() {
    checker := health.NewHealthChecker()
    // ... register checks ...
    
    // Setup graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-quit
        log.Println("Shutting down server...")
        
        // Mark as shutting down - readiness will return 503
        checker.SetShuttingDown(true)
        
        // Give time for load balancer to detect unhealthy status
        time.Sleep(5 * time.Second)
        
        // Shutdown server
        // ...
    }()
    
    // Start server
    // ...
}
```

## Endpoints

### GET /health/live

Liveness probe endpoint. Returns 200 if the application is running.

**Response (200 OK):**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "message": "Service is alive"
}
```

### GET /health/ready

Readiness probe endpoint. Returns 200 if ready to serve traffic, 503 if not ready.

**Response (200 OK):**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "message": "Service is ready",
  "checks": {
    "database": {
      "status": "healthy",
      "message": "Health check passed",
      "duration": "15ms",
      "timestamp": "2024-01-15T10:30:00Z"
    },
    "redis": {
      "status": "healthy",
      "message": "Health check passed",
      "duration": "5ms",
      "timestamp": "2024-01-15T10:30:00Z"
    }
  }
}
```

**Response (503 Service Unavailable):**
```json
{
  "status": "unhealthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "message": "Service is not ready",
  "checks": {
    "database": {
      "status": "unhealthy",
      "message": "Health check failed: connection refused",
      "duration": "1s",
      "timestamp": "2024-01-15T10:30:00Z",
      "details": {
        "error": "connection refused",
        "check_name": "database",
        "duration": "1s"
      }
    }
  }
}
```

## Kubernetes Integration

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: erpgo
spec:
  containers:
  - name: erpgo
    image: erpgo:latest
    ports:
    - containerPort: 8080
    livenessProbe:
      httpGet:
        path: /health/live
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 10
      timeoutSeconds: 1
      failureThreshold: 3
    readinessProbe:
      httpGet:
        path: /health/ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
      timeoutSeconds: 1
      failureThreshold: 3
```

## Testing

Run the tests:

```bash
go test -v ./pkg/health/...
```

Run the property-based test:

```bash
go test -v ./pkg/health/... -run TestProperty13
```

## Requirements Validation

This implementation validates the following requirements:

- **8.1**: Liveness probe returns 200 if app is running
- **8.2**: Readiness probe verifies database connectivity
- **8.3**: Returns 503 with details when dependencies are unhealthy
- **8.4**: Returns 503 from readiness during shutdown
- **8.5**: Health checks complete within 1 second

## Property 13: Readiness Check Database Verification

**Property**: For any readiness check request, if the database is unreachable, the system must return 503 status.

This property is tested in `TestProperty13_ReadinessCheckDatabaseVerification` which verifies:
- Database unhealthy → 503 status
- Database healthy, Redis unhealthy → 200 status (Redis not critical)
- Both healthy → 200 status
- Both unhealthy → 503 status
