package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventType represents the type of audit event
type EventType string

const (
	EventTypeLogin              EventType = "LOGIN"
	EventTypeLogout             EventType = "LOGOUT"
	EventTypeLoginFailed        EventType = "LOGIN_FAILED"
	EventTypePermissionChange   EventType = "PERMISSION_CHANGE"
	EventTypeRoleAssignment     EventType = "ROLE_ASSIGNMENT"
	EventTypeRoleRevocation     EventType = "ROLE_REVOCATION"
	EventTypeDataAccess         EventType = "DATA_ACCESS"
	EventTypeDataModification   EventType = "DATA_MODIFICATION"
	EventTypeDataDeletion       EventType = "DATA_DELETION"
	EventTypeConfigChange       EventType = "CONFIG_CHANGE"
	EventTypeSecurityEvent      EventType = "SECURITY_EVENT"
	EventTypeAccountLockout     EventType = "ACCOUNT_LOCKOUT"
	EventTypePasswordChange     EventType = "PASSWORD_CHANGE"
	EventTypeTokenRefresh       EventType = "TOKEN_REFRESH"
)

// AuditEvent represents a single audit log entry
type AuditEvent struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	EventType  EventType              `json:"event_type"`
	UserID     *uuid.UUID             `json:"user_id,omitempty"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Action     string                 `json:"action"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	Success    bool                   `json:"success"`
	Details    map[string]interface{} `json:"details,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// AuditFilter represents filtering criteria for querying audit logs
type AuditFilter struct {
	UserID     *uuid.UUID
	EventType  *EventType
	ResourceID *string
	StartTime  *time.Time
	EndTime    *time.Time
	Success    *bool
	Limit      int
	Offset     int
}

// AuditLogger defines the interface for audit logging operations
type AuditLogger interface {
	// LogEvent logs a single audit event
	LogEvent(ctx context.Context, event *AuditEvent) error
	
	// Query retrieves audit logs based on filter criteria
	Query(ctx context.Context, filter AuditFilter) ([]*AuditEvent, error)
	
	// Count returns the total number of audit logs matching the filter
	Count(ctx context.Context, filter AuditFilter) (int64, error)
}

// PostgresAuditLogger implements AuditLogger using PostgreSQL
type PostgresAuditLogger struct {
	db *pgxpool.Pool
}

// NewPostgresAuditLogger creates a new PostgreSQL-backed audit logger
func NewPostgresAuditLogger(db *pgxpool.Pool) *PostgresAuditLogger {
	return &PostgresAuditLogger{
		db: db,
	}
}

// LogEvent logs a single audit event to the database
func (l *PostgresAuditLogger) LogEvent(ctx context.Context, event *AuditEvent) error {
	if event == nil {
		return fmt.Errorf("audit event cannot be nil")
	}

	// Set defaults if not provided
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	// Validate required fields
	if event.EventType == "" {
		return fmt.Errorf("event_type is required")
	}
	if event.Action == "" {
		return fmt.Errorf("action is required")
	}

	// Convert details to JSONB
	var detailsJSON []byte
	var err error
	if event.Details != nil {
		detailsJSON, err = json.Marshal(event.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal details: %w", err)
		}
	}

	query := `
		INSERT INTO audit_logs (
			id, timestamp, event_type, user_id, resource_id, 
			action, ip_address, user_agent, success, details, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = l.db.Exec(ctx, query,
		event.ID,
		event.Timestamp,
		string(event.EventType),
		event.UserID,
		event.ResourceID,
		event.Action,
		event.IPAddress,
		event.UserAgent,
		event.Success,
		detailsJSON,
		event.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	return nil
}

// Query retrieves audit logs based on filter criteria
func (l *PostgresAuditLogger) Query(ctx context.Context, filter AuditFilter) ([]*AuditEvent, error) {
	query := `
		SELECT 
			id, timestamp, event_type, user_id, resource_id,
			action, ip_address, user_agent, success, details, created_at
		FROM audit_logs
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Build WHERE clause based on filter
	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argPos)
		args = append(args, filter.UserID)
		argPos++
	}

	if filter.EventType != nil {
		query += fmt.Sprintf(" AND event_type = $%d", argPos)
		args = append(args, string(*filter.EventType))
		argPos++
	}

	if filter.ResourceID != nil {
		query += fmt.Sprintf(" AND resource_id = $%d", argPos)
		args = append(args, *filter.ResourceID)
		argPos++
	}

	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argPos)
		args = append(args, filter.StartTime)
		argPos++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argPos)
		args = append(args, filter.EndTime)
		argPos++
	}

	if filter.Success != nil {
		query += fmt.Sprintf(" AND success = $%d", argPos)
		args = append(args, *filter.Success)
		argPos++
	}

	// Order by timestamp descending (most recent first)
	query += " ORDER BY timestamp DESC"

	// Add pagination
	limit := filter.Limit
	if limit <= 0 {
		limit = 100 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Max limit
	}
	query += fmt.Sprintf(" LIMIT $%d", argPos)
	args = append(args, limit)
	argPos++

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
		argPos++
	}

	rows, err := l.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	events := []*AuditEvent{}
	for rows.Next() {
		event := &AuditEvent{}
		var detailsJSON []byte
		var eventTypeStr string

		err := rows.Scan(
			&event.ID,
			&event.Timestamp,
			&eventTypeStr,
			&event.UserID,
			&event.ResourceID,
			&event.Action,
			&event.IPAddress,
			&event.UserAgent,
			&event.Success,
			&detailsJSON,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log row: %w", err)
		}

		event.EventType = EventType(eventTypeStr)

		// Unmarshal details if present
		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &event.Details); err != nil {
				return nil, fmt.Errorf("failed to unmarshal details: %w", err)
			}
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit log rows: %w", err)
	}

	return events, nil
}

// Count returns the total number of audit logs matching the filter
func (l *PostgresAuditLogger) Count(ctx context.Context, filter AuditFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	// Build WHERE clause based on filter (same as Query)
	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argPos)
		args = append(args, filter.UserID)
		argPos++
	}

	if filter.EventType != nil {
		query += fmt.Sprintf(" AND event_type = $%d", argPos)
		args = append(args, string(*filter.EventType))
		argPos++
	}

	if filter.ResourceID != nil {
		query += fmt.Sprintf(" AND resource_id = $%d", argPos)
		args = append(args, *filter.ResourceID)
		argPos++
	}

	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argPos)
		args = append(args, filter.StartTime)
		argPos++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argPos)
		args = append(args, filter.EndTime)
		argPos++
	}

	if filter.Success != nil {
		query += fmt.Sprintf(" AND success = $%d", argPos)
		args = append(args, *filter.Success)
		argPos++
	}

	var count int64
	err := l.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	return count, nil
}

// Helper functions for creating audit events

// NewLoginEvent creates an audit event for successful login
func NewLoginEvent(userID uuid.UUID, ipAddress, userAgent string) *AuditEvent {
	return &AuditEvent{
		EventType: EventTypeLogin,
		UserID:    &userID,
		Action:    "user_login",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   true,
	}
}

// NewLoginFailedEvent creates an audit event for failed login
func NewLoginFailedEvent(username, ipAddress, userAgent, reason string) *AuditEvent {
	return &AuditEvent{
		EventType: EventTypeLoginFailed,
		Action:    "user_login_failed",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   false,
		Details: map[string]interface{}{
			"username": username,
			"reason":   reason,
		},
	}
}

// NewLogoutEvent creates an audit event for logout
func NewLogoutEvent(userID uuid.UUID, ipAddress, userAgent string) *AuditEvent {
	return &AuditEvent{
		EventType: EventTypeLogout,
		UserID:    &userID,
		Action:    "user_logout",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   true,
	}
}

// NewRoleAssignmentEvent creates an audit event for role assignment
func NewRoleAssignmentEvent(actorID, targetUserID uuid.UUID, roleName, ipAddress string) *AuditEvent {
	return &AuditEvent{
		EventType:  EventTypeRoleAssignment,
		UserID:     &actorID,
		ResourceID: targetUserID.String(),
		Action:     "assign_role",
		IPAddress:  ipAddress,
		Success:    true,
		Details: map[string]interface{}{
			"target_user_id": targetUserID.String(),
			"role_name":      roleName,
		},
	}
}

// NewRoleRevocationEvent creates an audit event for role revocation
func NewRoleRevocationEvent(actorID, targetUserID uuid.UUID, roleName, ipAddress string) *AuditEvent {
	return &AuditEvent{
		EventType:  EventTypeRoleRevocation,
		UserID:     &actorID,
		ResourceID: targetUserID.String(),
		Action:     "revoke_role",
		IPAddress:  ipAddress,
		Success:    true,
		Details: map[string]interface{}{
			"target_user_id": targetUserID.String(),
			"role_name":      roleName,
		},
	}
}

// NewDataAccessEvent creates an audit event for sensitive data access
func NewDataAccessEvent(userID uuid.UUID, resourceType, resourceID, ipAddress string) *AuditEvent {
	return &AuditEvent{
		EventType:  EventTypeDataAccess,
		UserID:     &userID,
		ResourceID: resourceID,
		Action:     "access_data",
		IPAddress:  ipAddress,
		Success:    true,
		Details: map[string]interface{}{
			"resource_type": resourceType,
		},
	}
}

// NewAccountLockoutEvent creates an audit event for account lockout
func NewAccountLockoutEvent(userID uuid.UUID, ipAddress, reason string) *AuditEvent {
	return &AuditEvent{
		EventType: EventTypeAccountLockout,
		UserID:    &userID,
		Action:    "account_locked",
		IPAddress: ipAddress,
		Success:   true,
		Details: map[string]interface{}{
			"reason": reason,
		},
	}
}

// NewPasswordChangeEvent creates an audit event for password change
func NewPasswordChangeEvent(userID uuid.UUID, ipAddress, userAgent string, success bool) *AuditEvent {
	return &AuditEvent{
		EventType: EventTypePasswordChange,
		UserID:    &userID,
		Action:    "password_change",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   success,
	}
}


// MockAuditLogger is a mock implementation for testing (exported for use in other packages)
type MockAuditLogger struct {
	Events []*AuditEvent
}

// NewMockAuditLogger creates a new mock audit logger
func NewMockAuditLogger() *MockAuditLogger {
	return &MockAuditLogger{
		Events: make([]*AuditEvent, 0),
	}
}

func (m *MockAuditLogger) LogEvent(ctx context.Context, event *AuditEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	m.Events = append(m.Events, event)
	return nil
}

func (m *MockAuditLogger) Query(ctx context.Context, filter AuditFilter) ([]*AuditEvent, error) {
	result := make([]*AuditEvent, 0)
	
	for _, event := range m.Events {
		// Apply filters
		if filter.UserID != nil && (event.UserID == nil || *event.UserID != *filter.UserID) {
			continue
		}
		if filter.EventType != nil && event.EventType != *filter.EventType {
			continue
		}
		if filter.ResourceID != nil && event.ResourceID != *filter.ResourceID {
			continue
		}
		if filter.StartTime != nil && event.Timestamp.Before(*filter.StartTime) {
			continue
		}
		if filter.EndTime != nil && event.Timestamp.After(*filter.EndTime) {
			continue
		}
		if filter.Success != nil && event.Success != *filter.Success {
			continue
		}
		
		result = append(result, event)
	}
	
	// Apply limit and offset
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	
	start := filter.Offset
	if start > len(result) {
		return []*AuditEvent{}, nil
	}
	
	end := start + limit
	if end > len(result) {
		end = len(result)
	}
	
	return result[start:end], nil
}

func (m *MockAuditLogger) Count(ctx context.Context, filter AuditFilter) (int64, error) {
	events, err := m.Query(ctx, filter)
	if err != nil {
		return 0, err
	}
	return int64(len(events)), nil
}
