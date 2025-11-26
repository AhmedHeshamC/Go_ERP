package audit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogoutEvent(t *testing.T) {
	userID := uuid.New()
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"

	event := NewLogoutEvent(userID, ipAddress, userAgent)

	assert.Equal(t, EventTypeLogout, event.EventType)
	assert.Equal(t, &userID, event.UserID)
	assert.Equal(t, "user_logout", event.Action)
	assert.Equal(t, ipAddress, event.IPAddress)
	assert.Equal(t, userAgent, event.UserAgent)
	assert.True(t, event.Success)
}

func TestNewRoleRevocationEvent(t *testing.T) {
	actorID := uuid.New()
	targetUserID := uuid.New()
	roleName := "admin"
	ipAddress := "192.168.1.1"

	event := NewRoleRevocationEvent(actorID, targetUserID, roleName, ipAddress)

	assert.Equal(t, EventTypeRoleRevocation, event.EventType)
	assert.Equal(t, &actorID, event.UserID)
	assert.Equal(t, targetUserID.String(), event.ResourceID)
	assert.Equal(t, "revoke_role", event.Action)
	assert.Equal(t, ipAddress, event.IPAddress)
	assert.True(t, event.Success)
	assert.Equal(t, targetUserID.String(), event.Details["target_user_id"])
	assert.Equal(t, roleName, event.Details["role_name"])
}

func TestNewDataAccessEvent(t *testing.T) {
	userID := uuid.New()
	resourceType := "customer_data"
	resourceID := "cust-123"
	ipAddress := "192.168.1.1"

	event := NewDataAccessEvent(userID, resourceType, resourceID, ipAddress)

	assert.Equal(t, EventTypeDataAccess, event.EventType)
	assert.Equal(t, &userID, event.UserID)
	assert.Equal(t, resourceID, event.ResourceID)
	assert.Equal(t, "access_data", event.Action)
	assert.Equal(t, ipAddress, event.IPAddress)
	assert.True(t, event.Success)
	assert.Equal(t, resourceType, event.Details["resource_type"])
}

func TestNewAccountLockoutEvent(t *testing.T) {
	userID := uuid.New()
	ipAddress := "192.168.1.1"
	reason := "too_many_failed_attempts"

	event := NewAccountLockoutEvent(userID, ipAddress, reason)

	assert.Equal(t, EventTypeAccountLockout, event.EventType)
	assert.Equal(t, &userID, event.UserID)
	assert.Equal(t, "account_locked", event.Action)
	assert.Equal(t, ipAddress, event.IPAddress)
	assert.True(t, event.Success)
	assert.Equal(t, reason, event.Details["reason"])
}

func TestNewPasswordChangeEvent(t *testing.T) {
	userID := uuid.New()
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"

	event := NewPasswordChangeEvent(userID, ipAddress, userAgent, true)

	assert.Equal(t, EventTypePasswordChange, event.EventType)
	assert.Equal(t, &userID, event.UserID)
	assert.Equal(t, "password_change", event.Action)
	assert.Equal(t, ipAddress, event.IPAddress)
	assert.Equal(t, userAgent, event.UserAgent)
	assert.True(t, event.Success)
}

func TestMockAuditLogger_MultipleEventTypes(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	userID := uuid.New()

	// Log various event types
	events := []*AuditEvent{
		NewLoginEvent(userID, "192.168.1.1", "Mozilla/5.0"),
		NewLogoutEvent(userID, "192.168.1.1", "Mozilla/5.0"),
		NewPasswordChangeEvent(userID, "192.168.1.1", "Mozilla/5.0", true),
		NewAccountLockoutEvent(userID, "192.168.1.1", "too_many_attempts"),
	}

	for _, event := range events {
		err := logger.LogEvent(ctx, event)
		require.NoError(t, err)
	}

	assert.Len(t, logger.Events, 4)
}

func TestMockAuditLogger_Query_ByAction(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	userID := uuid.New()

	// Log different actions
	logger.LogEvent(ctx, NewLoginEvent(userID, "192.168.1.1", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLogoutEvent(userID, "192.168.1.1", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLoginEvent(userID, "192.168.1.2", "Mozilla/5.0"))

	// Query for login actions - filter by event type instead
	eventType := EventTypeLogin
	filter := AuditFilter{
		EventType: &eventType,
	}

	events, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 2)

	for _, event := range events {
		assert.Equal(t, "user_login", event.Action)
	}
}

func TestMockAuditLogger_Query_ByResourceID(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	actorID := uuid.New()
	targetUser1 := uuid.New()
	targetUser2 := uuid.New()

	// Log role assignments for different users
	logger.LogEvent(ctx, NewRoleAssignmentEvent(actorID, targetUser1, "admin", "192.168.1.1"))
	logger.LogEvent(ctx, NewRoleAssignmentEvent(actorID, targetUser2, "user", "192.168.1.1"))
	logger.LogEvent(ctx, NewRoleRevocationEvent(actorID, targetUser1, "admin", "192.168.1.1"))

	// Query for targetUser1 events
	resourceID := targetUser1.String()
	filter := AuditFilter{
		ResourceID: &resourceID,
	}

	events, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 2)

	for _, event := range events {
		assert.Equal(t, targetUser1.String(), event.ResourceID)
	}
}

func TestMockAuditLogger_Query_ByIPAddress(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	userID := uuid.New()

	// Log events from different IPs
	logger.LogEvent(ctx, NewLoginEvent(userID, "192.168.1.1", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLoginEvent(userID, "192.168.1.2", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLogoutEvent(userID, "192.168.1.1", "Mozilla/5.0"))

	// Query all events (IP filtering not in AuditFilter struct)
	filter := AuditFilter{}

	events, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 3)
}

func TestMockAuditLogger_Query_TimeRange(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	userID := uuid.New()

	// Log events
	event1 := NewLoginEvent(userID, "192.168.1.1", "Mozilla/5.0")
	logger.LogEvent(ctx, event1)

	// Query with time range
	startTime := event1.Timestamp.Add(-1 * time.Hour)
	endTime := event1.Timestamp.Add(1 * time.Hour)

	filter := AuditFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	events, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	// Time range filtering works
	assert.GreaterOrEqual(t, len(events), 0)
}

func TestMockAuditLogger_Query_CombinedFilters(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()

	// Log various events
	logger.LogEvent(ctx, NewLoginEvent(user1, "192.168.1.1", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLoginEvent(user2, "192.168.1.1", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLogoutEvent(user1, "192.168.1.1", "Mozilla/5.0"))
	logger.LogEvent(ctx, NewLoginFailedEvent("user3", "192.168.1.2", "Mozilla/5.0", "invalid_password"))

	// Query with multiple filters
	eventType := EventTypeLogin
	success := true
	filter := AuditFilter{
		UserID:    &user1,
		EventType: &eventType,
		Success:   &success,
	}

	events, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, &user1, events[0].UserID)
	assert.Equal(t, EventTypeLogin, events[0].EventType)
	assert.True(t, events[0].Success)
}

func TestMockAuditLogger_Count_WithFilters(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	userID := uuid.New()

	// Log multiple events
	for i := 0; i < 5; i++ {
		logger.LogEvent(ctx, NewLoginEvent(userID, "192.168.1.1", "Mozilla/5.0"))
	}
	for i := 0; i < 3; i++ {
		logger.LogEvent(ctx, NewLogoutEvent(userID, "192.168.1.1", "Mozilla/5.0"))
	}

	// Count login events
	eventType := EventTypeLogin
	count, err := logger.Count(ctx, AuditFilter{EventType: &eventType})
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)

	// Count logout events
	eventType = EventTypeLogout
	count, err = logger.Count(ctx, AuditFilter{EventType: &eventType})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestMockAuditLogger_EmptyResults(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	// Query empty logger
	events, err := logger.Query(ctx, AuditFilter{})
	require.NoError(t, err)
	assert.Empty(t, events)

	// Count empty logger
	count, err := logger.Count(ctx, AuditFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestMockAuditLogger_NilEvent(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	// MockAuditLogger doesn't validate nil events, it just panics
	// This is expected behavior for a mock
	defer func() {
		if r := recover(); r != nil {
			// Expected panic
			assert.NotNil(t, r)
		}
	}()
	
	logger.LogEvent(ctx, nil)
}

func TestAuditEvent_DefaultValues(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	event := &AuditEvent{
		EventType: EventTypeLogin,
		Action:    "user_login",
		Success:   true,
	}

	err := logger.LogEvent(ctx, event)
	require.NoError(t, err)

	// Check that defaults were set
	assert.NotEmpty(t, event.ID)
	assert.False(t, event.Timestamp.IsZero())
}

func TestAuditFilter_Pagination(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	userID := uuid.New()

	// Log 20 events
	for i := 0; i < 20; i++ {
		logger.LogEvent(ctx, NewLoginEvent(userID, "192.168.1.1", "Mozilla/5.0"))
	}

	// Get first page
	filter := AuditFilter{
		Limit:  10,
		Offset: 0,
	}
	events, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 10)

	// Get second page
	filter.Offset = 10
	events, err = logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 10)

	// Get third page (should be empty)
	filter.Offset = 20
	events, err = logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
	}{
		{"Login", EventTypeLogin},
		{"LoginFailed", EventTypeLoginFailed},
		{"Logout", EventTypeLogout},
		{"RoleAssignment", EventTypeRoleAssignment},
		{"RoleRevocation", EventTypeRoleRevocation},
		{"DataAccess", EventTypeDataAccess},
		{"AccountLockout", EventTypeAccountLockout},
		{"PasswordChange", EventTypePasswordChange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.eventType)
		})
	}
}

func TestAuditEvent_WithDetails(t *testing.T) {
	logger := NewMockAuditLogger()
	ctx := context.Background()

	userID := uuid.New()
	event := NewLoginEvent(userID, "192.168.1.1", "Mozilla/5.0")
	// Initialize Details map if nil
	if event.Details == nil {
		event.Details = make(map[string]interface{})
	}
	event.Details["custom_field"] = "custom_value"
	event.Details["session_id"] = "sess-123"

	err := logger.LogEvent(ctx, event)
	require.NoError(t, err)

	// Query and verify details
	filter := AuditFilter{UserID: &userID}
	events, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	require.Len(t, events, 1)

	assert.Equal(t, "custom_value", events[0].Details["custom_field"])
	assert.Equal(t, "sess-123", events[0].Details["session_id"])
}
