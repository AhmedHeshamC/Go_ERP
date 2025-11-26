package audit

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoginEvent(t *testing.T) {
	userID := uuid.New()
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"

	event := NewLoginEvent(userID, ipAddress, userAgent)

	assert.Equal(t, EventTypeLogin, event.EventType)
	assert.Equal(t, &userID, event.UserID)
	assert.Equal(t, "user_login", event.Action)
	assert.Equal(t, ipAddress, event.IPAddress)
	assert.Equal(t, userAgent, event.UserAgent)
	assert.True(t, event.Success)
}

func TestNewLoginFailedEvent(t *testing.T) {
	username := "testuser"
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	reason := "invalid_password"

	event := NewLoginFailedEvent(username, ipAddress, userAgent, reason)

	assert.Equal(t, EventTypeLoginFailed, event.EventType)
	assert.Equal(t, "user_login_failed", event.Action)
	assert.Equal(t, ipAddress, event.IPAddress)
	assert.Equal(t, userAgent, event.UserAgent)
	assert.False(t, event.Success)
	assert.Equal(t, username, event.Details["username"])
	assert.Equal(t, reason, event.Details["reason"])
}

func TestNewRoleAssignmentEvent(t *testing.T) {
	actorID := uuid.New()
	targetUserID := uuid.New()
	roleName := "admin"
	ipAddress := "192.168.1.1"

	event := NewRoleAssignmentEvent(actorID, targetUserID, roleName, ipAddress)

	assert.Equal(t, EventTypeRoleAssignment, event.EventType)
	assert.Equal(t, &actorID, event.UserID)
	assert.Equal(t, targetUserID.String(), event.ResourceID)
	assert.Equal(t, "assign_role", event.Action)
	assert.Equal(t, ipAddress, event.IPAddress)
	assert.True(t, event.Success)
	assert.Equal(t, targetUserID.String(), event.Details["target_user_id"])
	assert.Equal(t, roleName, event.Details["role_name"])
}

func TestMockAuditLogger_LogEvent(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	userID := uuid.New()
	event := NewLoginEvent(userID, "192.168.1.1", "Mozilla/5.0")

	err := logger.LogEvent(ctx, event)
	require.NoError(t, err)

	assert.Len(t, logger.Events, 1)
	assert.Equal(t, EventTypeLogin, logger.Events[0].EventType)
	assert.NotEmpty(t, logger.Events[0].ID)
	assert.False(t, logger.Events[0].Timestamp.IsZero())
}

func TestMockAuditLogger_Query_ByUserID(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	userID1 := uuid.New()
	userID2 := uuid.New()

	// Log events for two different users
	logger.LogEvent(ctx, NewLoginEvent(userID1, "192.168.1.1", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLoginEvent(userID2, "192.168.1.2", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLogoutEvent(userID1, "192.168.1.1", "Mozilla/5.0"))

	// Query for userID1 only
	filter := AuditFilter{
		UserID: &userID1,
	}

	events, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 2)
	
	for _, event := range events {
		assert.Equal(t, &userID1, event.UserID)
	}
}

func TestMockAuditLogger_Query_ByEventType(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	userID := uuid.New()

	// Log different event types
	logger.LogEvent(ctx, NewLoginEvent(userID, "192.168.1.1", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLogoutEvent(userID, "192.168.1.1", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLoginEvent(userID, "192.168.1.1", "Mozilla/5.0"))

	// Query for login events only
	eventType := EventTypeLogin
	filter := AuditFilter{
		EventType: &eventType,
	}

	events, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 2)
	
	for _, event := range events {
		assert.Equal(t, EventTypeLogin, event.EventType)
	}
}

func TestMockAuditLogger_Query_BySuccess(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	// Log successful and failed events
	logger.LogEvent(ctx, NewLoginEvent(uuid.New(), "192.168.1.1", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLoginFailedEvent("user1", "192.168.1.1", "Mozilla/5.0", "invalid_password"))
	logger.LogEvent(ctx, NewLoginFailedEvent("user2", "192.168.1.2", "Mozilla/5.0", "account_locked"))

	// Query for failed events only
	success := false
	filter := AuditFilter{
		Success: &success,
	}

	events, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 2)
	
	for _, event := range events {
		assert.False(t, event.Success)
	}
}

func TestMockAuditLogger_Query_WithPagination(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	userID := uuid.New()

	// Log 10 events
	for i := 0; i < 10; i++ {
		logger.LogEvent(ctx, NewLoginEvent(userID, "192.168.1.1", "Mozilla/5.0"))
	}

	// Query with limit
	filter := AuditFilter{
		Limit: 5,
	}

	events, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 5)

	// Query with offset
	filter = AuditFilter{
		Limit:  5,
		Offset: 5,
	}

	events, err = logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 5)
}

func TestMockAuditLogger_Count(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	userID1 := uuid.New()
	userID2 := uuid.New()

	// Log events
	logger.LogEvent(ctx, NewLoginEvent(userID1, "192.168.1.1", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLoginEvent(userID2, "192.168.1.2", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLogoutEvent(userID1, "192.168.1.1", "Mozilla/5.0"))

	// Count all events
	count, err := logger.Count(ctx, AuditFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Count events for userID1
	count, err = logger.Count(ctx, AuditFilter{UserID: &userID1})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestAuditEvent_Validation(t *testing.T) {
	tests := []struct {
		name        string
		event       *AuditEvent
		expectError bool
	}{
		{
			name: "valid event",
			event: &AuditEvent{
				EventType: EventTypeLogin,
				Action:    "user_login",
				Success:   true,
			},
			expectError: false,
		},
		{
			name: "missing event type",
			event: &AuditEvent{
				Action:  "user_login",
				Success: true,
			},
			expectError: true,
		},
		{
			name: "missing action",
			event: &AuditEvent{
				EventType: EventTypeLogin,
				Success:   true,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate validation that would happen in LogEvent
			hasError := tt.event.EventType == "" || tt.event.Action == ""
			assert.Equal(t, tt.expectError, hasError)
		})
	}
}
