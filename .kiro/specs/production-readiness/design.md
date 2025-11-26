# Production Readiness Design - ERPGo System

## Overview

This design document outlines the technical approach to transform ERPGo from its current state into a production-ready enterprise application. The design focuses on security hardening, reliability improvements, performance optimization, and operational excellence while maintaining backward compatibility where possible.

### Design Principles

1. **Security First**: All security vulnerabilities must be addressed before other improvements
2. **Fail Safe**: System should fail safely and provide clear error messages
3. **Observable**: All critical operations must be traceable and measurable
4. **Backward Compatible**: Changes should not break existing functionality
5. **Incremental**: Changes can be deployed incrementally without big-bang releases

### Architecture Goals

- Zero hardcoded secrets in codebase
- Sub-500ms p99 latency for API requests
- 99.9% uptime SLA capability
- Horizontal scalability to 10,000+ concurrent users
- Complete audit trail for compliance
- Automated recovery from common failures

## Architecture

### Current Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────┐
│      Gin HTTP Server            │
│  ┌──────────────────────────┐   │
│  │   Middleware Stack       │   │
│  │  - Auth                  │   │
│  │  - CORS                  │   │
│  │  - Security Headers      │   │
│  └──────────────────────────┘   │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│     Application Services        │
│  - User Service                 │
│  - Auth Service                 │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│      Domain Layer               │
│  - Entities                     │
│  - Repository Interfaces        │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│    Infrastructure Layer         │
│  - PostgreSQL Repositories      │
│  - Redis Cache                  │
└─────────────────────────────────┘
```

### Target Production Architecture

```
┌──────────────┐
│ Load Balancer│
└──────┬───────┘
       │
       ▼
┌─────────────────────────────────────────────┐
│         Application Instances (N)           │
│  ┌────────────────────────────────────┐     │
│  │  Enhanced Middleware Stack         │     │
│  │  - Request ID                      │     │
│  │  - Distributed Tracing             │     │
│  │  - Auth + Rate Limiting            │     │
│  │  - Input Validation                │     │
│  │  - Security Headers                │     │
│  │  - Audit Logging                   │     │
│  └────────────────────────────────────┘     │
│  ┌────────────────────────────────────┐     │
│  │  Application Services              │     │
│  │  - Transaction Management          │     │
│  │  - Custom Error Types              │     │
│  │  - Cache Integration               │     │
│  └────────────────────────────────────┘     │
└────┬──────────────────┬─────────────────────┘
     │                  │
     ▼                  ▼
┌─────────────┐   ┌──────────────┐
│ PostgreSQL  │   │    Redis     │
│  - Indexed  │   │  - Sessions  │
│  - Pooled   │   │  - Cache     │
│  - Replicas │   │  - Rate Limit│
└─────────────┘   └──────────────┘
     │                  │
     ▼                  ▼
┌─────────────────────────────────┐
│    Observability Stack          │
│  - Prometheus (Metrics)         │
│  - Jaeger (Traces)              │
│  - Grafana (Dashboards)         │
│  - AlertManager (Alerts)        │
└─────────────────────────────────┘
```


## Components and Interfaces

### 1. Secret Management Component

**Purpose**: Centralized secret loading and validation

**Location**: `pkg/secrets/manager.go`

```go
type SecretManager interface {
    LoadSecret(key string, validator SecretValidator) (string, error)
    ValidateAll() error
    RotateSecret(key string, newValue string) error
}

type SecretValidator interface {
    Validate(value string) error
}

// Validators
type JWTSecretValidator struct {
    MinEntropyBits int // 256
}

type PepperValidator struct {
    ForbiddenValues []string
}
```

**Key Features**:
- Validates secrets on startup
- Supports multiple secret sources (env, vault, AWS Secrets Manager)
- Enables zero-downtime secret rotation
- Fails fast with clear error messages

### 2. Custom Error Types

**Purpose**: Domain-specific errors for better error handling

**Location**: `pkg/errors/errors.go`

```go
type ErrorCode string

const (
    ErrCodeNotFound      ErrorCode = "NOT_FOUND"
    ErrCodeValidation    ErrorCode = "VALIDATION_ERROR"
    ErrCodeConflict      ErrorCode = "CONFLICT"
    ErrCodeUnauthorized  ErrorCode = "UNAUTHORIZED"
    ErrCodeForbidden     ErrorCode = "FORBIDDEN"
    ErrCodeInternal      ErrorCode = "INTERNAL_ERROR"
    ErrCodeRateLimit     ErrorCode = "RATE_LIMIT_EXCEEDED"
)

type AppError struct {
    Code       ErrorCode
    Message    string
    Details    map[string]interface{}
    Err        error
    StatusCode int
}

func (e *AppError) Error() string
func (e *AppError) Unwrap() error
func (e *AppError) WithContext(key string, value interface{}) *AppError
```


### 3. Transaction Manager

**Purpose**: Unified transaction management with retry logic

**Location**: `pkg/database/transaction.go`

```go
type TransactionManager interface {
    WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error
    WithRetryTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error
}

type TransactionConfig struct {
    MaxRetries      int
    RetryDelay      time.Duration
    IsolationLevel  pgx.TxIsoLevel
}
```

**Features**:
- Automatic rollback on error
- Deadlock detection and retry
- Configurable isolation levels
- Context cancellation support

### 4. Input Validator

**Purpose**: Centralized input validation with schema support

**Location**: `pkg/validation/validator.go`

```go
type Validator interface {
    Validate(input interface{}) error
    ValidateStruct(input interface{}) *ValidationResult
}

type ValidationResult struct {
    Valid  bool
    Errors map[string][]string
}

type SQLColumnWhitelist struct {
    AllowedColumns map[string]bool
}

func (w *SQLColumnWhitelist) ValidateColumn(column string) error
```

**Features**:
- Struct tag validation
- SQL column whitelisting
- Custom validation rules
- Field-level error messages


### 5. Enhanced Rate Limiter

**Purpose**: Multi-tier rate limiting with account lockout

**Location**: `pkg/ratelimit/enhanced_limiter.go`

```go
type EnhancedRateLimiter interface {
    AllowLogin(ctx context.Context, identifier string) (bool, error)
    RecordFailedLogin(ctx context.Context, identifier string) error
    IsAccountLocked(ctx context.Context, identifier string) (bool, time.Time, error)
    UnlockAccount(ctx context.Context, identifier string) error
}

type LoginRateLimitConfig struct {
    MaxAttempts     int           // 5
    WindowDuration  time.Duration // 15 minutes
    LockoutDuration time.Duration // 15 minutes
}
```

**Features**:
- Per-IP and per-account rate limiting
- Automatic account lockout
- Configurable thresholds
- Redis-backed for distributed systems

### 6. Cache Manager

**Purpose**: Intelligent caching with invalidation

**Location**: `pkg/cache/manager.go`

```go
type CacheManager interface {
    Get(ctx context.Context, key string, dest interface{}) error
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, keys ...string) error
    InvalidatePattern(ctx context.Context, pattern string) error
}

type CacheStrategy struct {
    TTL              time.Duration
    InvalidateOnWrite bool
    FallbackToSource  bool
}
```

**Caching Strategy**:
- User permissions: 5 minutes TTL
- User roles: 10 minutes TTL
- Invalidate on role/permission changes
- Fallback to database if Redis unavailable


### 7. Health Check System

**Purpose**: Comprehensive health and readiness checks

**Location**: `pkg/health/checker.go`

```go
type HealthChecker interface {
    CheckLiveness(ctx context.Context) *HealthStatus
    CheckReadiness(ctx context.Context) *HealthStatus
    RegisterCheck(name string, check HealthCheck) error
}

type HealthCheck interface {
    Name() string
    Check(ctx context.Context) error
    Timeout() time.Duration
}

type HealthStatus struct {
    Status    string                 // "healthy", "degraded", "unhealthy"
    Checks    map[string]CheckResult
    Timestamp time.Time
}
```

**Built-in Checks**:
- Database connectivity
- Redis connectivity
- Disk space
- Memory usage
- Goroutine count

### 8. Graceful Shutdown Manager

**Purpose**: Coordinate graceful shutdown

**Location**: `pkg/shutdown/manager.go`

```go
type ShutdownManager interface {
    RegisterHook(name string, hook ShutdownHook) error
    Shutdown(ctx context.Context) error
    NotifyShutdown() <-chan struct{}
}

type ShutdownHook interface {
    Name() string
    Shutdown(ctx context.Context) error
    Priority() int // Lower runs first
}
```

**Shutdown Sequence**:
1. Stop accepting new requests (HTTP server)
2. Wait for in-flight requests (30s timeout)
3. Close database connections
4. Close Redis connections
5. Flush metrics and logs
6. Exit


### 9. Audit Logger

**Purpose**: Immutable audit trail for compliance

**Location**: `pkg/audit/logger.go`

```go
type AuditLogger interface {
    LogEvent(ctx context.Context, event *AuditEvent) error
    Query(ctx context.Context, filter AuditFilter) ([]*AuditEvent, error)
}

type AuditEvent struct {
    ID          string
    Timestamp   time.Time
    EventType   string
    UserID      string
    ResourceID  string
    Action      string
    IPAddress   string
    UserAgent   string
    Success     bool
    Details     map[string]interface{}
}
```

**Audited Events**:
- User login/logout
- Permission changes
- Role assignments
- Sensitive data access
- Configuration changes
- Failed authentication attempts

### 10. Metrics Collector

**Purpose**: Comprehensive metrics collection

**Location**: `pkg/metrics/collector.go`

```go
type MetricsCollector interface {
    RecordRequest(method, path string, statusCode int, duration time.Duration)
    RecordDatabaseQuery(queryType string, duration time.Duration, err error)
    RecordCacheOperation(operation string, hit bool, duration time.Duration)
    RecordAuthAttempt(success bool, reason string)
}
```

**Key Metrics**:
- HTTP request rate, latency, error rate (by endpoint)
- Database query latency (by query type)
- Cache hit/miss rate
- Authentication success/failure rate
- Connection pool utilization
- Goroutine count
- Memory usage


## Data Models

### Secret Configuration

```go
type SecretConfig struct {
    JWTSecret       string `env:"JWT_SECRET" validate:"required,min=32"`
    RefreshSecret   string `env:"REFRESH_SECRET" validate:"required,min=32"`
    PasswordPepper  string `env:"PASSWORD_PEPPER" validate:"required,notdefault"`
    DatabaseURL     string `env:"DATABASE_URL" validate:"required"`
    RedisURL        string `env:"REDIS_URL" validate:"required"`
}
```

### Database Indexes

**New Indexes to Add**:

```sql
-- User table indexes
CREATE INDEX CONCURRENTLY idx_users_email_active ON users(email) WHERE is_active = true;
CREATE INDEX CONCURRENTLY idx_users_created_at_desc ON users(created_at DESC);

-- Role table indexes  
CREATE INDEX CONCURRENTLY idx_roles_name_lower ON roles(LOWER(name));

-- User roles indexes
CREATE INDEX CONCURRENTLY idx_user_roles_composite ON user_roles(user_id, role_id);
CREATE INDEX CONCURRENTLY idx_user_roles_assigned_at ON user_roles(assigned_at DESC);

-- Audit log indexes (new table)
CREATE INDEX CONCURRENTLY idx_audit_user_id_timestamp ON audit_logs(user_id, timestamp DESC);
CREATE INDEX CONCURRENTLY idx_audit_event_type_timestamp ON audit_logs(event_type, timestamp DESC);
CREATE INDEX CONCURRENTLY idx_audit_resource_id ON audit_logs(resource_id);
```

### Audit Log Table

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type VARCHAR(100) NOT NULL,
    user_id UUID REFERENCES users(id),
    resource_id VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Make it append-only (no updates/deletes)
CREATE RULE audit_logs_no_update AS ON UPDATE TO audit_logs DO INSTEAD NOTHING;
CREATE RULE audit_logs_no_delete AS ON DELETE TO audit_logs DO INSTEAD NOTHING;
```


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Secret Validation on Startup
*For any* system startup, all required secrets (JWT_SECRET, PASSWORD_PEPPER, DATABASE_URL) must be loaded from environment variables and validated before the system accepts requests
**Validates: Requirements 1.1, 1.2**

### Property 2: JWT Secret Entropy
*For any* JWT_SECRET value, if it has less than 256 bits of entropy (32 bytes), the system must reject it during startup
**Validates: Requirements 1.3**

### Property 3: Input Validation Consistency
*For any* API endpoint and any invalid input, the system must reject the input with a structured validation error before processing
**Validates: Requirements 2.1**

### Property 4: SQL Column Whitelisting
*For any* dynamic SQL query with ORDER BY clause, if the column name is not in the whitelist, the system must reject the query
**Validates: Requirements 2.2**

### Property 5: Domain Error Type Consistency
*For any* error originating in the domain layer, the error must be wrapped in a domain-specific error type (AppError) with appropriate error code
**Validates: Requirements 3.1**

### Property 6: Database Error Classification
*For any* database operation failure, the system must classify the error as one of: NotFound, ConstraintViolation, ConnectionError, or InternalError
**Validates: Requirements 3.4**

### Property 7: Transaction Atomicity
*For any* service operation that performs multiple database writes, either all writes succeed and commit, or all writes are rolled back
**Validates: Requirements 4.1**

### Property 8: Deadlock Retry Logic
*For any* transaction that encounters a deadlock error, the system must retry the transaction up to 3 times with exponential backoff before failing
**Validates: Requirements 4.3**


### Property 9: Login Rate Limiting
*For any* IP address, if more than 5 login attempts occur within 15 minutes, subsequent attempts must be rejected with rate limit error
**Validates: Requirements 5.1**

### Property 10: Account Lockout After Failed Logins
*For any* user account, if 5 consecutive login failures occur, the account must be locked for 15 minutes regardless of correct password
**Validates: Requirements 5.2**

### Property 11: Permission Cache Consistency
*For any* user permission check, if the result is cached, subsequent checks within 5 minutes must return the cached result
**Validates: Requirements 7.1**

### Property 12: Cache Invalidation on Role Change
*For any* user whose roles are modified, the permission cache for that user must be invalidated immediately
**Validates: Requirements 7.3**

### Property 13: Readiness Check Database Verification
*For any* readiness check request, if the database is unreachable, the system must return 503 status
**Validates: Requirements 8.2**

### Property 14: Graceful Shutdown Request Completion
*For any* in-flight request when shutdown is initiated, the request must be allowed to complete (up to 30 second timeout) before connections are closed
**Validates: Requirements 9.2**

### Property 15: Trace ID Uniqueness
*For any* two concurrent requests, the trace IDs generated must be unique
**Validates: Requirements 12.1**

### Property 16: Audit Log Immutability
*For any* audit log entry, once written, it cannot be modified or deleted through the application
**Validates: Requirements 18.4**

### Property 17: Migration Transaction Rollback
*For any* database migration that fails, all changes made by that migration must be rolled back
**Validates: Requirements 19.2**


## Error Handling

### Error Hierarchy

```
AppError (base)
├── NotFoundError (404)
├── ValidationError (400)
│   └── Fields map[string][]string
├── ConflictError (409)
├── UnauthorizedError (401)
├── ForbiddenError (403)
├── RateLimitError (429)
│   └── RetryAfter time.Duration
└── InternalError (500)
```

### Error Handling Strategy

**Layer-Specific Handling**:

1. **Domain Layer**: Returns domain errors (ErrUserNotFound, ErrInvalidEmail)
2. **Application Layer**: Wraps domain errors with context, manages transactions
3. **Infrastructure Layer**: Converts database/cache errors to domain errors
4. **Interface Layer**: Converts domain errors to HTTP responses

**Error Context**:
- Every error includes correlation ID from request context
- Errors include user ID when available (for audit)
- Errors include operation name and parameters (sanitized)
- Stack traces captured for internal errors

**Error Logging**:
- 4xx errors: INFO level (expected errors)
- 5xx errors: ERROR level with full context
- Security errors: WARN level with IP and user agent
- All errors include structured fields for querying

### Retry Strategy

**Retryable Operations**:
- Database deadlocks: 3 retries, exponential backoff (100ms, 200ms, 400ms)
- Redis connection errors: 2 retries, linear backoff (50ms, 100ms)
- External API calls: 3 retries with circuit breaker

**Non-Retryable Operations**:
- Validation errors
- Authentication failures
- Not found errors
- Conflict errors


## Testing Strategy

### Unit Testing

**Coverage Target**: 80% for critical paths

**Critical Paths**:
- Authentication and authorization
- Password hashing and validation
- JWT token generation and validation
- Input validation
- Error handling and wrapping
- Transaction management

**Testing Approach**:
- Table-driven tests for validation logic
- Mock repositories for service layer tests
- Test all error paths explicitly
- Use testify/assert for assertions

### Property-Based Testing

**Library**: Use `gopter` or `rapid` for property-based testing in Go

**Properties to Test**:
1. Secret validation (Property 1, 2)
2. Input validation (Property 3, 4)
3. Error type consistency (Property 5, 6)
4. Transaction atomicity (Property 7)
5. Rate limiting (Property 9, 10)
6. Cache behavior (Property 11, 12)
7. Trace ID uniqueness (Property 15)

**Configuration**: Run each property test with minimum 100 iterations

**Example Property Test**:
```go
// Property: JWT secrets with less than 32 bytes must be rejected
func TestProperty_JWTSecretMinimumLength(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("secrets under 32 bytes are rejected", 
        prop.ForAll(
            func(secret string) bool {
                validator := NewJWTSecretValidator(256)
                err := validator.Validate(secret)
                if len(secret) < 32 {
                    return err != nil // Should fail
                }
                return err == nil // Should pass
            },
            gen.AnyString(),
        ),
    )
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```


### Integration Testing

**Test Database**: Use dedicated test database with transactions for isolation

**Test Scenarios**:
- Full authentication flow (register, login, refresh token)
- Role assignment and permission checking
- Multi-step operations with transaction rollback
- Rate limiting across multiple requests
- Cache invalidation on data changes
- Graceful shutdown with in-flight requests

**Test Utilities**:
- `SetupTestDatabase()`: Creates isolated test database
- `CreateTestUser()`: Helper for creating test users
- `WithTestTransaction()`: Wraps tests in transaction that rolls back

### End-to-End Testing

**Test Environment**: Docker Compose with all dependencies

**Test Scenarios**:
- Complete user registration and login flow
- Permission-based access control
- Rate limiting and account lockout
- Health check endpoints
- Metrics collection
- Graceful shutdown

### Load Testing

**Tool**: Use `k6` or `hey` for load testing

**Scenarios**:
1. **Baseline**: 100 RPS for 5 minutes
2. **Peak Load**: 1000 RPS for 5 minutes
3. **Stress Test**: Gradually increase to 5000 RPS
4. **Spike Test**: Sudden jump from 100 to 2000 RPS

**Success Criteria**:
- p99 latency < 500ms at 1000 RPS
- Error rate < 0.1%
- No memory leaks over 1 hour test
- Graceful degradation under overload

### Security Testing

**Automated Scans**:
- `gosec` for static security analysis
- `nancy` for dependency vulnerability scanning
- `trivy` for container scanning

**Manual Testing**:
- SQL injection attempts
- JWT token manipulation
- Rate limit bypass attempts
- CSRF token validation
- XSS prevention


## Deployment Strategy

### Phase 1: Security Hardening (Week 1-2)

**Goal**: Eliminate all critical security vulnerabilities

**Changes**:
1. Implement SecretManager with validation
2. Remove hardcoded pepper constant
3. Add JWT secret entropy validation
4. Implement SQL column whitelisting
5. Add input validation middleware

**Deployment**: Can be deployed independently, backward compatible

**Validation**:
- All secrets loaded from environment
- System fails fast with invalid secrets
- SQL injection tests pass
- Input validation tests pass

### Phase 2: Reliability Improvements (Week 3-4)

**Goal**: Improve error handling and transaction management

**Changes**:
1. Implement custom error types
2. Add transaction manager with retry logic
3. Enhance error logging with context
4. Add graceful shutdown
5. Implement health checks

**Deployment**: Requires brief downtime for error type migration

**Validation**:
- All errors properly typed
- Transactions rollback on failure
- Graceful shutdown works
- Health checks return correct status

### Phase 3: Performance Optimization (Week 5-6)

**Goal**: Optimize database queries and add caching

**Changes**:
1. Add database indexes
2. Implement cache manager
3. Add permission caching
4. Optimize N+1 queries
5. Add connection pool monitoring

**Deployment**: Zero-downtime, indexes created concurrently

**Validation**:
- Query performance improved
- Cache hit rate > 70%
- No N+1 queries detected
- Connection pool healthy


### Phase 4: Observability (Week 7-8)

**Goal**: Add comprehensive monitoring and tracing

**Changes**:
1. Enhance Prometheus metrics
2. Implement distributed tracing
3. Add audit logging
4. Create Grafana dashboards
5. Configure alerts

**Deployment**: Zero-downtime, additive changes

**Validation**:
- All metrics exposed
- Traces visible in Jaeger
- Audit logs captured
- Dashboards functional
- Alerts firing correctly

### Phase 5: Final Validation (Week 9-10)

**Goal**: Load testing and security audit

**Changes**:
1. Run load tests
2. Perform security audit
3. Update documentation
4. Create runbooks
5. Train operations team

**Deployment**: Documentation and operational changes only

**Validation**:
- Load tests pass
- Security audit clean
- Documentation complete
- Team trained

### Rollback Strategy

**Each Phase**:
- Feature flags for new functionality
- Database migrations are reversible
- Configuration changes documented
- Rollback procedures tested

**Rollback Triggers**:
- Error rate > 5%
- p99 latency > 2 seconds
- Critical security issue discovered
- Data corruption detected


## Monitoring and Alerting

### Key Metrics

**Application Metrics**:
```
# Request metrics
http_requests_total{method, path, status}
http_request_duration_seconds{method, path}
http_requests_in_flight

# Authentication metrics
auth_attempts_total{result}
auth_account_lockouts_total
auth_token_validations_total{result}

# Database metrics
db_queries_total{query_type}
db_query_duration_seconds{query_type}
db_connection_pool_size
db_connection_pool_in_use
db_connection_pool_wait_duration_seconds

# Cache metrics
cache_operations_total{operation, result}
cache_hit_rate
cache_evictions_total

# Business metrics
users_active_total
users_created_total
api_keys_active_total
```

### Alert Rules

**Critical Alerts** (Page immediately):
```yaml
- alert: HighErrorRate
  expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
  for: 2m
  annotations:
    summary: "Error rate above 5%"
    runbook: "https://runbooks.example.com/high-error-rate"

- alert: DatabaseDown
  expr: up{job="postgres"} == 0
  for: 1m
  annotations:
    summary: "Database is down"
    runbook: "https://runbooks.example.com/database-down"

- alert: HighLatency
  expr: histogram_quantile(0.99, http_request_duration_seconds) > 1.0
  for: 5m
  annotations:
    summary: "p99 latency above 1 second"
    runbook: "https://runbooks.example.com/high-latency"
```

**Warning Alerts** (Notify, don't page):
```yaml
- alert: ConnectionPoolNearExhaustion
  expr: db_connection_pool_in_use / db_connection_pool_size > 0.8
  for: 5m
  annotations:
    summary: "Connection pool 80% utilized"

- alert: LowCacheHitRate
  expr: cache_hit_rate < 0.7
  for: 10m
  annotations:
    summary: "Cache hit rate below 70%"

- alert: HighAuthFailureRate
  expr: rate(auth_attempts_total{result="failure"}[5m]) > 0.1
  for: 5m
  annotations:
    summary: "High authentication failure rate"
```


### Dashboards

**Overview Dashboard**:
- Request rate, error rate, latency (RED metrics)
- Active users
- System health status
- Recent alerts

**Database Dashboard**:
- Query latency by type
- Connection pool utilization
- Slow queries (>100ms)
- Transaction rollback rate

**Authentication Dashboard**:
- Login attempts (success/failure)
- Account lockouts
- Token validation rate
- Rate limit violations

**Cache Dashboard**:
- Hit/miss rate
- Eviction rate
- Memory usage
- Operation latency

## Security Considerations

### Secret Management

**Production Secrets**:
- Store in HashiCorp Vault or AWS Secrets Manager
- Rotate every 90 days
- Use different secrets per environment
- Never log secret values

**Secret Rotation Process**:
1. Generate new secret
2. Add to secret manager with version tag
3. Update application to accept both old and new
4. Deploy application
5. Remove old secret after 24 hours

### Authentication Security

**JWT Tokens**:
- Short-lived access tokens (15 minutes)
- Longer refresh tokens (7 days)
- Token blacklisting via Redis
- Secure token storage (httpOnly cookies)

**Password Security**:
- bcrypt with cost 12
- Unique pepper per environment
- Password strength requirements enforced
- Common password detection

**Rate Limiting**:
- Per-IP: 100 requests/minute
- Per-user: 1000 requests/hour
- Login endpoint: 5 attempts/15 minutes
- Distributed rate limiting via Redis


### Input Validation

**Validation Strategy**:
- Validate at API boundary (middleware)
- Use struct tags for common validations
- Custom validators for business rules
- Whitelist approach for SQL columns

**SQL Injection Prevention**:
- Always use parameterized queries
- Whitelist allowed column names for ORDER BY
- Validate and sanitize all user inputs
- Use ORM/query builder where appropriate

### Audit and Compliance

**Audit Requirements**:
- Log all authentication events
- Log all permission changes
- Log all sensitive data access
- Retain logs for 1 year minimum

**Compliance**:
- GDPR: User data export and deletion
- SOC 2: Audit logging and access controls
- HIPAA: Encryption at rest and in transit (if applicable)

## Performance Optimization

### Database Optimization

**Connection Pooling**:
- Max connections: 100
- Min connections: 10
- Connection lifetime: 1 hour
- Idle timeout: 30 minutes

**Query Optimization**:
- Add indexes on foreign keys
- Use EXPLAIN ANALYZE for slow queries
- Implement query result caching
- Use connection pooling

**N+1 Query Prevention**:
- Use JOIN queries for related data
- Implement eager loading
- Monitor query count per request

### Caching Strategy

**Cache Layers**:
1. **Application Cache**: In-memory for hot data (5 second TTL)
2. **Redis Cache**: Distributed cache (5-10 minute TTL)
3. **Database**: Source of truth

**Cache Keys**:
```
user:permissions:{user_id}     # TTL: 5 minutes
user:roles:{user_id}           # TTL: 10 minutes
role:permissions:{role_id}     # TTL: 10 minutes
```

**Cache Invalidation**:
- Invalidate on write operations
- Use cache tags for bulk invalidation
- Implement cache warming for critical data


### Resource Management

**Memory Management**:
- Set GOMEMLIMIT to 80% of container memory
- Monitor goroutine count
- Use connection pooling
- Implement request timeouts

**CPU Management**:
- Use GOMAXPROCS appropriately
- Profile CPU usage under load
- Optimize hot paths
- Use worker pools for background tasks

**Goroutine Management**:
- Always use context for cancellation
- Set timeouts on all operations
- Monitor goroutine leaks
- Use errgroup for concurrent operations

## Operational Procedures

### Deployment Checklist

**Pre-Deployment**:
- [ ] All tests passing (unit, integration, e2e)
- [ ] Security scan clean
- [ ] Load tests passed
- [ ] Database migrations tested
- [ ] Rollback plan documented
- [ ] Monitoring dashboards ready
- [ ] Alerts configured
- [ ] Runbooks updated

**Deployment**:
- [ ] Deploy to staging first
- [ ] Run smoke tests
- [ ] Monitor metrics for 30 minutes
- [ ] Deploy to production (blue-green or canary)
- [ ] Run smoke tests in production
- [ ] Monitor for 1 hour

**Post-Deployment**:
- [ ] Verify all health checks passing
- [ ] Check error rates and latency
- [ ] Verify audit logs working
- [ ] Update deployment log
- [ ] Notify team of successful deployment

### Incident Response

**Severity Levels**:
- **P0 (Critical)**: System down, data loss, security breach
- **P1 (High)**: Major feature broken, high error rate
- **P2 (Medium)**: Minor feature broken, degraded performance
- **P3 (Low)**: Cosmetic issues, minor bugs

**Response Times**:
- P0: Immediate response, page on-call
- P1: 15 minute response
- P2: 1 hour response
- P3: Next business day

**Incident Process**:
1. Acknowledge alert
2. Assess severity
3. Notify stakeholders
4. Investigate and diagnose
5. Implement fix or rollback
6. Verify resolution
7. Post-mortem (for P0/P1)


### Backup and Recovery

**Backup Strategy**:
- Automated backups every 6 hours
- Retain daily backups for 7 days
- Retain weekly backups for 4 weeks
- Retain monthly backups for 1 year
- Store backups in separate region/zone

**Backup Verification**:
- Automated integrity checks
- Monthly restore tests
- Document restore procedures
- Track backup success rate

**Recovery Procedures**:
- RTO (Recovery Time Objective): 4 hours
- RPO (Recovery Point Objective): 1 hour
- Documented step-by-step procedures
- Tested quarterly

### Disaster Recovery

**Scenarios**:
1. **Database Failure**: Failover to replica (5 minute RTO)
2. **Region Failure**: Failover to backup region (4 hour RTO)
3. **Data Corruption**: Restore from backup (4 hour RTO)
4. **Security Breach**: Isolate, investigate, restore (varies)

**DR Testing**:
- Quarterly DR drills
- Document lessons learned
- Update procedures based on findings

## Migration Path

### From Current State to Production

**Step 1: Security Audit** (Week 1)
- Review all code for hardcoded secrets
- Identify all security vulnerabilities
- Create remediation plan
- Prioritize by severity

**Step 2: Implement Secret Management** (Week 1-2)
- Create SecretManager component
- Add secret validation
- Update configuration loading
- Test with invalid secrets
- Deploy to staging

**Step 3: Fix SQL Injection Risks** (Week 2)
- Implement column whitelisting
- Audit all dynamic queries
- Add input validation
- Test with malicious inputs
- Deploy to staging


**Step 4: Implement Error Handling** (Week 3)
- Create custom error types
- Update all error returns
- Add error context
- Test error scenarios
- Deploy to staging

**Step 5: Add Transaction Management** (Week 3-4)
- Implement TransactionManager
- Update services to use transactions
- Add retry logic
- Test rollback scenarios
- Deploy to staging

**Step 6: Database Optimization** (Week 5)
- Create indexes (CONCURRENTLY)
- Optimize N+1 queries
- Add query monitoring
- Load test
- Deploy to production (zero-downtime)

**Step 7: Implement Caching** (Week 5-6)
- Create CacheManager
- Add permission caching
- Implement invalidation
- Monitor cache hit rate
- Deploy to production

**Step 8: Add Observability** (Week 7)
- Enhance metrics
- Implement tracing
- Create dashboards
- Configure alerts
- Deploy to production

**Step 9: Implement Audit Logging** (Week 7-8)
- Create audit_logs table
- Implement AuditLogger
- Add audit events
- Test immutability
- Deploy to production

**Step 10: Final Validation** (Week 9-10)
- Run load tests
- Perform security audit
- Update documentation
- Train team
- Production launch

## Success Metrics

### Technical Metrics

**Performance**:
- p50 latency < 100ms
- p99 latency < 500ms
- p99.9 latency < 2s
- Throughput: 1000 RPS sustained

**Reliability**:
- Uptime: 99.9% (43 minutes downtime/month)
- Error rate: < 0.1%
- MTTR (Mean Time To Recovery): < 1 hour
- MTBF (Mean Time Between Failures): > 720 hours

**Security**:
- Zero critical vulnerabilities
- Zero hardcoded secrets
- 100% authentication events audited
- All secrets rotated within 90 days

**Quality**:
- Code coverage: > 80% for critical paths
- All property tests passing
- Zero known data corruption issues
- All migrations reversible

### Operational Metrics

**Monitoring**:
- 100% of critical metrics collected
- All alerts tested and functional
- Mean time to detect (MTTD): < 5 minutes
- Dashboard coverage: 100% of key metrics

**Documentation**:
- API documentation: 100% coverage
- Runbooks: All critical scenarios covered
- Architecture diagrams: Up to date
- Deployment procedures: Documented and tested

**Team Readiness**:
- On-call rotation established
- Team trained on runbooks
- Incident response tested
- Escalation paths defined

## Conclusion

This design provides a comprehensive roadmap to transform ERPGo into a production-ready enterprise application. The phased approach allows for incremental improvements while maintaining system stability. Each phase builds upon the previous one, with clear validation criteria and rollback procedures.

The focus on security, reliability, performance, and observability ensures the system can handle production workloads while providing the visibility needed to maintain and improve it over time.

**Next Steps**: Review this design document and proceed to create the detailed implementation task list.
