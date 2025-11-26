package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"erpgo/pkg/audit"
)

func TestAuditMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		path           string
		userID         uuid.UUID
		shouldAudit    bool
		expectedEvents int
	}{
		{
			name:           "GET user by ID should be audited",
			method:         "GET",
			path:           "/api/v1/users/:id",
			userID:         uuid.New(),
			shouldAudit:    true,
			expectedEvents: 1,
		},
		{
			name:           "GET users list should not be audited",
			method:         "GET",
			path:           "/api/v1/users",
			userID:         uuid.New(),
			shouldAudit:    false,
			expectedEvents: 0,
		},
		{
			name:           "PUT user should be audited",
			method:         "PUT",
			path:           "/api/v1/users/:id",
			userID:         uuid.New(),
			shouldAudit:    true,
			expectedEvents: 1,
		},
		{
			name:           "POST login should not be audited",
			method:         "POST",
			path:           "/api/v1/auth/login",
			userID:         uuid.New(),
			shouldAudit:    false,
			expectedEvents: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock audit logger
			mockLogger := audit.NewMockAuditLogger()
			logger := zerolog.Nop()

			// Create test router
			router := gin.New()
			router.Use(AuditMiddleware(mockLogger, logger))

			// Add test route
			router.Handle(tt.method, tt.path, func(c *gin.Context) {
				// Set user ID in context
				c.Set("user_id", tt.userID.String())
				c.Status(http.StatusOK)
			})

			// Create test request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			// Serve request
			router.ServeHTTP(w, req)

			// Check audit events
			events, err := mockLogger.Query(context.Background(), audit.AuditFilter{})
			assert.NoError(t, err)
			assert.Len(t, events, tt.expectedEvents)

			if tt.shouldAudit && len(events) > 0 {
				assert.Equal(t, audit.EventTypeDataAccess, events[0].EventType)
				assert.Equal(t, &tt.userID, events[0].UserID)
			}
		})
	}
}

func TestShouldAudit(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		expected bool
	}{
		{
			name:     "GET user by ID",
			method:   "GET",
			path:     "/api/v1/users/:id",
			expected: true,
		},
		{
			name:     "GET users list",
			method:   "GET",
			path:     "/api/v1/users",
			expected: false,
		},
		{
			name:     "PUT user",
			method:   "PUT",
			path:     "/api/v1/users/:id",
			expected: true,
		},
		{
			name:     "DELETE user",
			method:   "DELETE",
			path:     "/api/v1/users/:id",
			expected: true,
		},
		{
			name:     "POST login",
			method:   "POST",
			path:     "/api/v1/auth/login",
			expected: false,
		},
		{
			name:     "GET customer by ID",
			method:   "GET",
			path:     "/api/v1/customers/:id",
			expected: true,
		},
		{
			name:     "GET order by ID",
			method:   "GET",
			path:     "/api/v1/orders/:id",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldAudit(tt.method, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetResourceType(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "user path",
			path:     "/api/v1/users/:id",
			expected: "user",
		},
		{
			name:     "customer path",
			path:     "/api/v1/customers/:id",
			expected: "customer",
		},
		{
			name:     "order path",
			path:     "/api/v1/orders/:id",
			expected: "order",
		},
		{
			name:     "unknown path",
			path:     "/api/v1/unknown/:id",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getResourceType(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
