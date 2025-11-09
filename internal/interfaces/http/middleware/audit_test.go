package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"erpgo/pkg/cache"
)

func TestDefaultAuditConfig(t *testing.T) {
	config := DefaultAuditConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, "INFO", config.Level)
	assert.Contains(t, config.ExcludeEndpoints, "/health")
	assert.Contains(t, config.IncludeHeaders, "X-Request-ID")
	assert.Contains(t, config.SensitiveHeaders, "Authorization")
	assert.True(t, config.StoreInRedis)
	assert.True(t, config.LogAuthEvents)
	assert.True(t, config.LogSecurityEvents)
	assert.Equal(t, "audit:", config.RedisKeyPrefix)
}

func TestNewAuditor(t *testing.T) {
	config := DefaultAuditConfig()
	mockCache := cache.NewMockCache()
	logger := zerolog.Nop()

	auditor := NewAuditor(config, mockCache, logger)

	assert.NotNil(t, auditor)
	assert.Equal(t, config, auditor.config)
	assert.Equal(t, mockCache, auditor.cache)
}

func TestAuditorShouldExcludeEndpoint(t *testing.T) {
	config := DefaultAuditConfig()
	mockCache := cache.NewMockCache()
	logger := zerolog.Nop()
	auditor := NewAuditor(config, mockCache, logger)

	tests := []struct {
		path     string
		expected bool
	}{
		{
			path:     "/health",
			expected: true,
		},
		{
			path:     "/metrics",
			expected: true,
		},
		{
			path:     "/api/v1/users",
			expected: false,
		},
		{
			path:     "/favicon.ico",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := auditor.shouldExcludeEndpoint(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuditorShouldLogRequestBody(t *testing.T) {
	config := DefaultAuditConfig()
	mockCache := cache.NewMockCache()
	logger := zerolog.Nop()
	auditor := NewAuditor(config, mockCache, logger)

	tests := []struct {
		name       string
		setupFunc  func(*gin.Context)
		expected   bool
	}{
		{
			name: "Auth login endpoint",
			setupFunc: func(c *gin.Context) {
				c.Request.URL.Path = "/api/v1/auth/login"
			},
			expected: true,
		},
		{
			name: "File upload",
			setupFunc: func(c *gin.Context) {
				c.Request.URL.Path = "/api/v1/upload"
				c.Request.Header.Set("Content-Type", "multipart/form-data")
			},
			expected: false,
		},
		{
			name: "Regular endpoint",
			setupFunc: func(c *gin.Context) {
				c.Request.URL.Path = "/api/v1/products"
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/test", nil)

			tt.setupFunc(c)

			result := auditor.shouldLogRequestBody(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuditorDetermineEvent(t *testing.T) {
	config := DefaultAuditConfig()
	mockCache := cache.NewMockCache()
	logger := zerolog.Nop()
	auditor := NewAuditor(config, mockCache, logger)

	tests := []struct {
		name         string
		setupFunc    func(*gin.Context)
		statusCode   int
		duration     time.Duration
		expected     string
	}{
		{
			name: "Successful login",
			setupFunc: func(c *gin.Context) {
				c.Request.URL.Path = "/api/v1/auth/login"
				c.Request.Method = "POST"
			},
			statusCode: 200,
			duration:   100 * time.Millisecond,
			expected:   "AUTH_LOGIN_SUCCESS",
		},
		{
			name: "Failed login",
			setupFunc: func(c *gin.Context) {
				c.Request.URL.Path = "/api/v1/auth/login"
				c.Request.Method = "POST"
			},
			statusCode: 401,
			duration:   100 * time.Millisecond,
			expected:   "AUTH_LOGIN_FAILED",
		},
		{
			name: "User creation",
			setupFunc: func(c *gin.Context) {
				c.Request.URL.Path = "/api/v1/users"
				c.Request.Method = "POST"
			},
			statusCode: 201,
			duration:   100 * time.Millisecond,
			expected:   "USER_CREATED",
		},
		{
			name: "Data read",
			setupFunc: func(c *gin.Context) {
				c.Request.URL.Path = "/api/v1/products"
				c.Request.Method = "GET"
			},
			statusCode: 200,
			duration:   100 * time.Millisecond,
			expected:   "DATA_READ",
		},
		{
			name: "Unauthorized access",
			setupFunc: func(c *gin.Context) {
				c.Request.URL.Path = "/api/v1/admin"
				c.Request.Method = "GET"
			},
			statusCode: 401,
			duration:   100 * time.Millisecond,
			expected:   "SECURITY_UNAUTHORIZED",
		},
		{
			name: "Slow request",
			setupFunc: func(c *gin.Context) {
				c.Request.URL.Path = "/api/v1/products"
				c.Request.Method = "GET"
			},
			statusCode: 200,
			duration:   2 * time.Second, // Assuming slow query threshold is 1s
			expected:   "PERFORMANCE_SLOW_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(tt.setupFunc(nil), nil)

			tt.setupFunc(c)

			event := auditor.determineEvent(c, tt.statusCode, tt.duration)
			assert.Equal(t, tt.expected, event)
		})
	}
}

func TestAuditorDetermineCategory(t *testing.T) {
	config := DefaultAuditConfig()
	mockCache := cache.NewMockCache()
	logger := zerolog.Nop()
	auditor := NewAuditor(config, mockCache, logger)

	tests := []struct {
		event    string
		expected string
	}{
		{
			event:    "AUTH_LOGIN_SUCCESS",
			expected: "AUTHENTICATION",
		},
		{
			event:    "USER_CREATED",
			expected: "USER_MANAGEMENT",
		},
		{
			event:    "DATA_READ",
			expected: "DATA_ACCESS",
		},
		{
			event:    "ADMIN_ACTION",
			expected: "ADMINISTRATION",
		},
		{
			event:    "SECURITY_UNAUTHORIZED",
			expected: "SECURITY",
		},
		{
			event:    "PERFORMANCE_SLOW_REQUEST",
			expected: "PERFORMANCE",
		},
		{
			event:    "REQUEST_PROCESSED",
			expected: "GENERAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			category := auditor.determineCategory(nil, tt.event)
			assert.Equal(t, tt.expected, category)
		})
	}
}

func TestAuditorDetermineLevel(t *testing.T) {
	config := DefaultAuditConfig()
	mockCache := cache.NewMockCache()
	logger := zerolog.Nop()
	auditor := NewAuditor(config, mockCache, logger)

	tests := []struct {
		statusCode int
		event      string
		expected   string
	}{
		{
			statusCode: 200,
			event:      "REQUEST_PROCESSED",
			expected:   "INFO",
		},
		{
			statusCode: 201,
			event:      "USER_CREATED",
			expected:   "INFO",
		},
		{
			statusCode: 404,
			event:      "REQUEST_PROCESSED",
			expected:   "WARN",
		},
		{
			statusCode: 401,
			event:      "SECURITY_UNAUTHORIZED",
			expected:   "WARN",
		},
		{
			statusCode: 500,
			event:      "REQUEST_PROCESSED",
			expected:   "ERROR",
		},
		{
			statusCode: 200,
			event:      "ADMIN_ACTION",
			expected:   "INFO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			level := auditor.determineLevel(nil, tt.statusCode, tt.event)
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestAuditorShouldLogFullDetails(t *testing.T) {
	config := DefaultAuditConfig()
	config.LogSensitiveOps = true
	mockCache := cache.NewMockCache()
	logger := zerolog.Nop()
	auditor := NewAuditor(config, mockCache, logger)

	tests := []struct {
		event    string
		expected bool
	}{
		{
			event:    "AUTH_LOGIN_SUCCESS",
			expected: true,
		},
		{
			event:    "USER_CREATED",
			expected: true,
		},
		{
			event:    "ADMIN_ACTION",
			expected: true,
		},
		{
			event:    "SECURITY_UNAUTHORIZED",
			expected: true,
		},
		{
			event:    "DATA_READ",
			expected: false,
		},
		{
			event:    "REQUEST_PROCESSED",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			result := auditor.shouldLogFullDetails(nil, tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuditorIsSensitiveHeader(t *testing.T) {
	config := DefaultAuditConfig()
	mockCache := cache.NewMockCache()
	logger := zerolog.Nop()
	auditor := NewAuditor(config, mockCache, logger)

	tests := []struct {
		header   string
		expected bool
	}{
		{
			header:   "Authorization",
			expected: true,
		},
		{
			header:   "X-API-Key",
			expected: true,
		},
		{
			header:   "Cookie",
			expected: true,
		},
		{
			header:   "Content-Type",
			expected: false,
		},
		{
			header:   "User-Agent",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			result := auditor.isSensitiveHeader(tt.header)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuditorIsSensitiveField(t *testing.T) {
	config := DefaultAuditConfig()
	mockCache := cache.NewMockCache()
	logger := zerolog.Nop()
	auditor := NewAuditor(config, mockCache, logger)

	tests := []struct {
		field    string
		expected bool
	}{
		{
			field:    "password",
			expected: true,
		},
		{
			field:    "api_key",
			expected: true,
		},
		{
			field:    "credit_card_number",
			expected: true,
		},
		{
			field:    "name",
			expected: false,
		},
		{
			field:    "email",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			result := auditor.isSensitiveField(tt.field)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuditorSanitizeData(t *testing.T) {
	config := DefaultAuditConfig()
	mockCache := cache.NewMockCache()
	logger := zerolog.Nop()
	auditor := NewAuditor(config, mockCache, logger)

	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name: "Sanitize map with sensitive fields",
			input: map[string]interface{}{
				"username": "testuser",
				"password": "secret123",
				"email":    "test@example.com",
			},
			expected: map[string]interface{}{
				"username": "testuser",
				"password": "[REDACTED]",
				"email":    "test@example.com",
			},
		},
		{
			name: "Sanitize array",
			input: []interface{}{
				map[string]interface{}{
					"name":     "test",
					"password": "secret",
				},
				"regular_string",
			},
			expected: []interface{}{
				map[string]interface{}{
					"name":     "test",
					"password": "[REDACTED]",
				},
				"regular_string",
			},
		},
		{
			name:     "Leave primitive types unchanged",
			input:    "regular string",
			expected: "regular string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auditor.sanitizeData(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuditorIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := DefaultAuditConfig()
	config.StoreInRedis = false // Disable Redis for testing
	mockCache := cache.NewMockCache()

	// Create a logger that captures log output
	var logOutput strings.Builder
	logger := zerolog.New(&zerolog.ConsoleWriter{Out: &logOutput})

	auditor := NewAuditor(config, mockCache, logger)

	// Create test router with audit middleware
	router := gin.New()
	router.Use(auditor.Middleware())
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		// Simulate authentication
		c.JSON(200, gin.H{"token": "test_token"})
	})

	// Make a request
	requestBody := `{"username": "testuser", "password": "secret123"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, 200, w.Code)

	// Check that audit log was created
	assert.NotEmpty(t, logOutput.String())
	assert.Contains(t, logOutput.String(), "Audit event")
	assert.Contains(t, logOutput.String(), "AUTH_LOGIN_SUCCESS")
}

func TestAuditLogging(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := DefaultAuditConfig()
	config.StoreInRedis = false
	mockCache := cache.NewMockCache()
	logger := zerolog.Nop()

	// Create middleware
	middleware := AuditLogging(config, mockCache, logger)

	assert.NotNil(t, middleware)

	// Test that middleware can be applied to router
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestSecurityAuditLogging(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockCache := cache.NewMockCache()
	logger := zerolog.Nop()

	// Create security audit middleware
	middleware := SecurityAuditLogging(mockCache, logger)

	assert.NotNil(t, middleware)

	// Test that middleware can be applied to router
	router := gin.New()
	router.Use(middleware)
	router.GET("/api/v1/users", func(c *gin.Context) {
		c.JSON(200, gin.H{"users": []string{}})
	})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestResponseWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create response writer wrapper
	w := httptest.NewRecorder()
	wrapper := &responseWriter{
		ResponseWriter: gin.CreateTestContext(w).Writer,
		body:          &bytes.Buffer{},
	}

	// Write data
	data := []byte("test response")
	n, err := wrapper.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, "test response", wrapper.body.String())
}