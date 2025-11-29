package security

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"erpgo/internal/interfaces/http/middleware"
	"erpgo/pkg/auth"
	"erpgo/pkg/cache"
	"erpgo/pkg/security"
)

// TestSecurityCoordinatorIntegration tests the complete security middleware integration
func TestSecurityCoordinatorIntegration(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test logger
	logger := getLogger(t)

	// Create test cache
	cache := getTestCache(t)

	// Create security coordinator
	securityConfig := middleware.DefaultSecurityCoordinatorConfig("development")
	coordinator, err := middleware.NewSecurityCoordinator(securityConfig, cache, logger)
	require.NoError(t, err)
	coordinator.Start()
	defer coordinator.Stop()

	// Create test router
	router := gin.New()
	router.Use(coordinator.Middleware())

	// Add test endpoints
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	router.GET("/api/v1/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "protected"})
	})

	t.Run("Security Headers Present", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Check security headers
		assert.Contains(t, w.Header().Get("X-Content-Type-Options"), "nosniff")
		assert.Contains(t, w.Header().Get("X-Frame-Options"), "DENY")
		assert.Contains(t, w.Header().Get("X-XSS-Protection"), "1; mode=block")
	})

	t.Run("Input Validation", func(t *testing.T) {
		// Test SQL injection attempt
		req, _ := http.NewRequest("GET", "/test?input=' OR 1=1 --", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// In development mode with strict mode disabled, should pass
		// In production with strict mode enabled, would be blocked
		if securityConfig.InputValidation.StrictMode {
			assert.Equal(t, http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CSRF Protection", func(t *testing.T) {
		// First GET request to set CSRF token
		req1, _ := http.NewRequest("GET", "/test", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Extract CSRF token from cookie
		cookies := w1.Result().Cookies()
		var csrfToken string
		for _, cookie := range cookies {
			if cookie.Name == "_csrf" {
				csrfToken = cookie.Value
				break
			}
		}

		// POST request without CSRF token
		req2, _ := http.NewRequest("POST", "/test", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		// Should be rejected in production mode
		if securityConfig.CSRF.Enabled && securityConfig.Environment == "production" {
			assert.Equal(t, http.StatusForbidden, w2.Code)
		}

		// POST request with CSRF token
		req3, _ := http.NewRequest("POST", "/test", nil)
		req3.Header.Set("X-CSRF-Token", csrfToken)
		w3 := httptest.NewRecorder()
		router.ServeHTTP(w3, req3)

		assert.Equal(t, http.StatusOK, w3.Code)
	})
}

// TestAPIKeyManagement tests the API key management system
func TestAPIKeyManagement(t *testing.T) {
	logger := getLogger(t)
	cache := getTestCache(t)

	// Create API key service
	apiKeyRepo := auth.NewInMemoryAPIKeyRepository()
	apiKeyService := auth.NewAPIKeyService(apiKeyRepo, cache, logger, auth.DefaultAPIKeyConfig())

	userID := uuid.New()
	createdBy := uuid.New()

	t.Run("Create API Key", func(t *testing.T) {
		req := &auth.CreateAPIKeyRequest{
			Name:        "Test API Key",
			Description: "A test API key",
			UserID:      userID,
			Roles:       []string{"read"},
			Permissions: []string{"api.read"},
			CreatedBy:   createdBy,
		}

		resp, err := apiKeyService.CreateAPIKey(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.PlainText)
		assert.Equal(t, req.Name, resp.APIKey.Name)
		assert.Equal(t, req.UserID, resp.APIKey.UserID)
	})

	t.Run("Validate API Key", func(t *testing.T) {
		// First create an API key
		req := &auth.CreateAPIKeyRequest{
			Name:        "Validation Test Key",
			Description: "A key for validation testing",
			UserID:      userID,
			Roles:       []string{"read"},
			CreatedBy:   createdBy,
		}

		createResp, err := apiKeyService.CreateAPIKey(context.Background(), req)
		require.NoError(t, err)

		// Validate the API key
		validateReq := &auth.ValidateAPIKeyRequest{
			APIKey: createResp.PlainText,
		}

		validateResp, err := apiKeyService.ValidateAPIKey(context.Background(), validateReq)
		require.NoError(t, err)
		assert.True(t, validateResp.Valid)
		assert.NotNil(t, validateResp.APIKey)
		assert.Equal(t, createResp.APIKey.ID, validateResp.APIKey.ID)
	})

	t.Run("Invalid API Key", func(t *testing.T) {
		validateReq := &auth.ValidateAPIKeyRequest{
			APIKey: "invalid-key-123",
		}

		validateResp, err := apiKeyService.ValidateAPIKey(context.Background(), validateReq)
		require.NoError(t, err)
		assert.False(t, validateResp.Valid)
		assert.Nil(t, validateResp.APIKey)
		assert.Equal(t, "Invalid API key", validateResp.Message)
	})

	t.Run("Delete API Key", func(t *testing.T) {
		// First create an API key
		req := &auth.CreateAPIKeyRequest{
			Name:        "Delete Test Key",
			Description: "A key for deletion testing",
			UserID:      userID,
			CreatedBy:   createdBy,
		}

		createResp, err := apiKeyService.CreateAPIKey(context.Background(), req)
		require.NoError(t, err)

		// Delete the API key
		err = apiKeyService.DeleteAPIKey(context.Background(), createResp.APIKey.ID, userID)
		require.NoError(t, err)

		// Try to validate the deleted key
		validateReq := &auth.ValidateAPIKeyRequest{
			APIKey: createResp.PlainText,
		}

		validateResp, err := apiKeyService.ValidateAPIKey(context.Background(), validateReq)
		require.NoError(t, err)
		assert.False(t, validateResp.Valid)
	})
}

// TestSecurityMonitoring tests the security monitoring system
func TestSecurityMonitoring(t *testing.T) {
	logger := getLogger(t)
	cache := getTestCache(t)

	// Create security monitor
	monitor := security.NewSecurityMonitor(security.DefaultSecurityConfig(), cache, logger)
	monitor.Start()
	defer monitor.Stop()

	t.Run("Log Authentication Failed", func(t *testing.T) {
		monitor.AuthenticationFailed("192.168.1.100", "TestAgent", "user123", "invalid password")

		// Wait for event processing
		time.Sleep(100 * time.Millisecond)

		stats := monitor.GetSecurityStats()
		assert.Contains(t, stats, "failed_auth_count")
	})

	t.Run("Log Unauthorized Access", func(t *testing.T) {
		monitor.UnauthorizedAccess("192.168.1.101", "TestAgent", "", "/api/v1/admin", "GET")

		// Wait for event processing
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Log Suspicious Activity", func(t *testing.T) {
		metadata := map[string]interface{}{
			"pattern": "SQL injection attempt",
			"payload": "' OR 1=1 --",
		}
		monitor.SuspiciousActivity("192.168.1.102", "sqlmap/1.0", "", "SQL injection detected", metadata)

		// Wait for event processing
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Log Injection Attempt", func(t *testing.T) {
		monitor.InjectionAttempt("192.168.1.103", "TestAgent", "", "SQL", "' UNION SELECT * FROM users --")

		// Wait for event processing
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Log Rate Limit Exceeded", func(t *testing.T) {
		monitor.RateLimitExceeded("192.168.1.104", "TestAgent", "", "/api/v1/users")

		// Wait for event processing
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Log Privilege Escalation", func(t *testing.T) {
		monitor.PrivilegeEscalation("192.168.1.105", "TestAgent", "user123", "admin")

		// Wait for event processing
		time.Sleep(100 * time.Millisecond)
	})
}

// TestRateLimiting tests the rate limiting functionality
func TestRateLimiting(t *testing.T) {
	logger := getLogger(t)

	// Create rate limiter
	config := middleware.DefaultRateLimitConfig()
	config.RequestsPerSecond = 2 // Very low limit for testing
	config.Burst = 3

	limiter := middleware.NewRateLimiter(config, nil, logger)

	// Create test router
	router := gin.New()
	router.Use(limiter.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	t.Run("Normal Requests", func(t *testing.T) {
		// Make requests within limit
		for i := 0; i < 2; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("Rate Limit Exceeded", func(t *testing.T) {
		// Make requests that exceed limit
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if i >= 2 {
				assert.Equal(t, http.StatusTooManyRequests, w.Code)
			}
		}
	})

	t.Run("Rate Limit Headers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Contains(t, w.Header().Get("X-RateLimit-Limit"), "2")
		assert.Contains(t, w.Header().Get("X-RateLimit-Remaining"), "1")
	})
}

// TestInputValidation tests the input validation functionality
func TestInputValidation(t *testing.T) {
	logger := getLogger(t)

	// Create input validator
	config := middleware.DefaultInputValidationConfig()
	config.MaxStringLength = 100 // Lower limit for testing
	config.StrictMode = true

	validator, err := middleware.NewInputValidator(config, logger)
	require.NoError(t, err)

	// Create test router
	router := gin.New()
	router.Use(validator.Middleware())
	router.POST("/test", func(c *gin.Context) {
		var data map[string]interface{}
		c.ShouldBindJSON(&data)
		c.JSON(http.StatusOK, data)
	})

	t.Run("Valid Input", func(t *testing.T) {
		data := map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
		}
		jsonData, _ := json.Marshal(data)

		req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("SQL Injection Attempt", func(t *testing.T) {
		data := map[string]interface{}{
			"query": "SELECT * FROM users WHERE id = 1 OR 1=1 --",
		}
		jsonData, _ := json.Marshal(data)

		req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("XSS Attempt", func(t *testing.T) {
		data := map[string]interface{}{
			"comment": "<script>alert('XSS')</script>",
		}
		jsonData, _ := json.Marshal(data)

		req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Oversized Input", func(t *testing.T) {
		longString := make([]byte, 200) // Exceeds MaxStringLength of 100
		for i := range longString {
			longString[i] = 'a'
		}

		data := map[string]interface{}{
			"data": string(longString),
		}
		jsonData, _ := json.Marshal(data)

		req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestCSRFProtection tests the CSRF protection functionality
func TestCSRFProtection(t *testing.T) {
	logger := getLogger(t)

	// Create CSRF protection
	config := middleware.DefaultCSRFConfig()
	config.Enabled = true

	csrf := middleware.NewCSRF(config, logger)

	// Create test router
	router := gin.New()
	router.Use(csrf.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	t.Run("GET Request Sets CSRF Token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Check if CSRF cookie is set
		cookies := w.Result().Cookies()
		var csrfCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "_csrf" {
				csrfCookie = cookie
				break
			}
		}
		assert.NotNil(t, csrfCookie)
		assert.NotEmpty(t, csrfCookie.Value)
	})

	t.Run("POST Without CSRF Token", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("POST With CSRF Token", func(t *testing.T) {
		// First, make a GET request to get the CSRF token
		req1, _ := http.NewRequest("GET", "/test", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Extract CSRF token
		var csrfToken string
		for _, cookie := range w1.Result().Cookies() {
			if cookie.Name == "_csrf" {
				csrfToken = cookie.Value
				break
			}
		}

		// Make POST request with CSRF token
		req2, _ := http.NewRequest("POST", "/test", nil)
		req2.Header.Set("X-CSRF-Token", csrfToken)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)
	})
}

// TestAuditLogging tests the audit logging functionality
func TestAuditLogging(t *testing.T) {
	logger := getLogger(t)
	cache := getTestCache(t)

	// Create audit logger
	config := middleware.DefaultAuditConfig()
	config.Enabled = true

	auditor := middleware.NewAuditor(config, cache, logger)

	// Create test router
	router := gin.New()
	router.Use(auditor.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"token": "test-token"})
	})
	router.POST("/api/v1/auth/login-fail", func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
	})

	t.Run("Successful Request Logged", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// TODO: Check audit log entries
		// In a real test, you would verify that audit events were logged
	})

	t.Run("Authentication Event Logged", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(`{"email":"test@example.com","password":"password"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// TODO: Check that authentication event was logged
	})

	t.Run("Failed Authentication Logged", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/login-fail", bytes.NewBufferString(`{"email":"test@example.com","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// TODO: Check that authentication failure was logged
	})
}

// Helper functions

func getLogger(t *testing.T) zerolog.Logger {
	return zerolog.New(zerolog.NewConsoleWriter())
}

func getTestCache(t *testing.T) cache.Cache {
	return cache.NewMockCache()
}
