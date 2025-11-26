# Audit Logging System

The audit logging system provides comprehensive tracking of security-relevant events in the ERPGo application.

## Overview

The audit logging system captures and stores immutable records of:
- User authentication events (login, logout, failed attempts)
- Role and permission changes
- Sensitive data access
- Configuration changes
- Security events

## Database Schema

The audit logs are stored in the `audit_logs` table with the following structure:

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    user_id UUID REFERENCES users(id),
    resource_id VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);
```

The table is **append-only** - updates and deletes are prevented by database rules to ensure immutability.

## Usage

### Basic Setup

```go
import (
    "erpgo/pkg/audit"
    "github.com/jackc/pgx/v5/pgxpool"
)

// Create audit logger
db := // ... your database connection pool
auditLogger := audit.NewPostgresAuditLogger(db)
```

### Logging Events

#### Login Events

```go
// Successful login
event := audit.NewLoginEvent(userID, ipAddress, userAgent)
err := auditLogger.LogEvent(ctx, event)

// Failed login
event := audit.NewLoginFailedEvent(username, ipAddress, userAgent, "invalid_password")
err := auditLogger.LogEvent(ctx, event)
```

#### Logout Events

```go
event := audit.NewLogoutEvent(userID, ipAddress, userAgent)
err := auditLogger.LogEvent(ctx, event)
```

#### Role Assignment/Revocation

```go
// Assign role
event := audit.NewRoleAssignmentEvent(actorID, targetUserID, roleName, ipAddress)
err := auditLogger.LogEvent(ctx, event)

// Revoke role
event := audit.NewRoleRevocationEvent(actorID, targetUserID, roleName, ipAddress)
err := auditLogger.LogEvent(ctx, event)
```

#### Data Access

```go
event := audit.NewDataAccessEvent(userID, "customer", customerID, ipAddress)
err := auditLogger.LogEvent(ctx, event)
```

#### Custom Events

```go
event := &audit.AuditEvent{
    EventType:  audit.EventTypeSecurityEvent,
    UserID:     &userID,
    Action:     "custom_action",
    IPAddress:  ipAddress,
    Success:    true,
    Details: map[string]interface{}{
        "key": "value",
    },
}
err := auditLogger.LogEvent(ctx, event)
```

### Querying Audit Logs

```go
// Query by user ID
filter := audit.AuditFilter{
    UserID: &userID,
    Limit:  50,
}
events, err := auditLogger.Query(ctx, filter)

// Query by event type
eventType := audit.EventTypeLogin
filter := audit.AuditFilter{
    EventType: &eventType,
    Limit:     100,
}
events, err := auditLogger.Query(ctx, filter)

// Query by time range
startTime := time.Now().Add(-24 * time.Hour)
endTime := time.Now()
filter := audit.AuditFilter{
    StartTime: &startTime,
    EndTime:   &endTime,
    Limit:     100,
}
events, err := auditLogger.Query(ctx, filter)

// Count events
count, err := auditLogger.Count(ctx, filter)
```

## Integration with Handlers

### Authentication Handler

The authentication handler automatically logs:
- Successful logins
- Failed login attempts
- Logouts
- Password changes

```go
authHandler := handlers.NewAuthHandler(userService, logger)
authHandler.SetAuditLogger(auditLogger)
```

### Role Handler

The role handler automatically logs:
- Role assignments
- Role revocations
- Permission changes

```go
roleHandler := handlers.NewRoleHandler(roleRepo, logger)
roleHandler.SetAuditLogger(auditLogger)
```

### Audit Middleware

Use the audit middleware to automatically log sensitive data access:

```go
import "erpgo/internal/interfaces/http/middleware"

router.Use(middleware.AuditMiddleware(auditLogger, logger))
```

The middleware automatically audits:
- GET requests to individual user, customer, and order resources
- PUT, PATCH, DELETE requests to user data

## Event Types

The following event types are available:

- `EventTypeLogin` - Successful user login
- `EventTypeLogout` - User logout
- `EventTypeLoginFailed` - Failed login attempt
- `EventTypePermissionChange` - Permission added or removed from role
- `EventTypeRoleAssignment` - Role assigned to user
- `EventTypeRoleRevocation` - Role removed from user
- `EventTypeDataAccess` - Sensitive data accessed
- `EventTypeDataModification` - Data modified
- `EventTypeDataDeletion` - Data deleted
- `EventTypeConfigChange` - Configuration changed
- `EventTypeSecurityEvent` - Security-related event
- `EventTypeAccountLockout` - Account locked due to failed attempts
- `EventTypePasswordChange` - Password changed
- `EventTypeTokenRefresh` - Access token refreshed

## Testing

A mock audit logger is provided for testing:

```go
import "erpgo/pkg/audit"

mockLogger := audit.NewMockAuditLogger()

// Use in tests
err := mockLogger.LogEvent(ctx, event)

// Verify events
events, err := mockLogger.Query(ctx, audit.AuditFilter{})
assert.Len(t, events, 1)
```

## Compliance

The audit logging system is designed to meet compliance requirements:

- **Immutability**: Audit logs cannot be modified or deleted
- **Completeness**: All security-relevant events are logged
- **Traceability**: Each event includes user ID, timestamp, IP address, and action
- **Retention**: Logs are retained according to your retention policy
- **Searchability**: Logs can be queried and filtered efficiently

## Performance Considerations

- Audit logging is asynchronous where possible to minimize impact on request latency
- Indexes are created on commonly queried fields (user_id, event_type, timestamp)
- Failed audit logging does not cause request failures (logged as errors)
- Consider archiving old audit logs to maintain query performance

## Security

- Audit logs are stored in a separate table with restricted access
- Database rules prevent modification or deletion of audit logs
- Sensitive data should not be included in the `details` field
- IP addresses and user agents are logged for forensic analysis
