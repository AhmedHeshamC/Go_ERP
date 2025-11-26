package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestLivenessEndpoint tests the /health/live endpoint
// Validates: Requirements 8.1
func TestLivenessEndpoint(t *testing.T) {
	checker := NewHealthChecker()
	handler := NewHandler(checker)

	router := gin.New()
	router.GET("/health/live", handler.LivenessHandler)

	req, _ := http.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.NotNil(t, response["timestamp"])
	assert.Equal(t, "Service is alive", response["message"])
}

// TestReadinessEndpointHealthy tests the /health/ready endpoint when all dependencies are healthy
// Validates: Requirements 8.2
func TestReadinessEndpointHealthy(t *testing.T) {
	checker := NewHealthChecker()
	
	// Register healthy database check
	dbCheck := &mockHealthCheck{
		name:    "database",
		healthy: true,
	}
	checker.RegisterCheck("database", dbCheck)

	handler := NewHandler(checker)
	router := gin.New()
	router.GET("/health/ready", handler.ReadinessHandler)

	req, _ := http.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "Service is ready", response["message"])
	assert.NotNil(t, response["checks"])
}

// TestReadinessEndpointUnhealthy tests the /health/ready endpoint when database is unhealthy
// Validates: Requirements 8.2, 8.3
func TestReadinessEndpointUnhealthy(t *testing.T) {
	checker := NewHealthChecker()
	
	// Register unhealthy database check
	dbCheck := &mockHealthCheck{
		name:    "database",
		healthy: false,
		err:     errors.New("connection refused"),
	}
	checker.RegisterCheck("database", dbCheck)

	handler := NewHandler(checker)
	router := gin.New()
	router.GET("/health/ready", handler.ReadinessHandler)

	req, _ := http.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 503 when unhealthy
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "unhealthy", response["status"])
	assert.Equal(t, "Service is not ready", response["message"])
	
	// Verify detailed status is included
	checks, ok := response["checks"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, checks, "database")
	
	dbCheckResult, ok := checks["database"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "unhealthy", dbCheckResult["status"])
	assert.NotEmpty(t, dbCheckResult["message"])
}

// TestReadinessEndpointWithDetails tests that detailed status is returned for each dependency
// Validates: Requirements 8.3
func TestReadinessEndpointWithDetails(t *testing.T) {
	checker := NewHealthChecker()
	
	// Register multiple checks
	dbCheck := &mockHealthCheck{
		name:    "database",
		healthy: true,
	}
	redisCheck := &mockHealthCheck{
		name:    "redis",
		healthy: false,
		err:     errors.New("redis connection timeout"),
	}
	
	checker.RegisterCheck("database", dbCheck)
	checker.RegisterCheck("redis", redisCheck)

	handler := NewHandler(checker)
	router := gin.New()
	router.GET("/health/ready", handler.ReadinessHandler)

	req, _ := http.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify both checks are present
	checks, ok := response["checks"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, checks, "database")
	assert.Contains(t, checks, "redis")
	
	// Verify database check details
	dbCheckResult, ok := checks["database"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "healthy", dbCheckResult["status"])
	assert.NotNil(t, dbCheckResult["duration"])
	assert.NotNil(t, dbCheckResult["timestamp"])
	
	// Verify redis check details (including error details)
	redisCheckResult, ok := checks["redis"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "unhealthy", redisCheckResult["status"])
	assert.NotEmpty(t, redisCheckResult["message"])
	assert.NotNil(t, redisCheckResult["details"])
}

// TestReadinessEndpointDuringShutdown tests that readiness returns 503 during shutdown
// Validates: Requirements 8.4
func TestReadinessEndpointDuringShutdown(t *testing.T) {
	checker := NewHealthChecker()
	
	// Register healthy database check
	dbCheck := &mockHealthCheck{
		name:    "database",
		healthy: true,
	}
	checker.RegisterCheck("database", dbCheck)

	handler := NewHandler(checker)
	router := gin.New()
	router.GET("/health/ready", handler.ReadinessHandler)

	// First request - should be healthy
	req1, _ := http.NewRequest("GET", "/health/ready", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Set shutting down
	checker.SetShuttingDown(true)

	// Second request - should be unhealthy
	req2, _ := http.NewRequest("GET", "/health/ready", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusServiceUnavailable, w2.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w2.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "unhealthy", response["status"])
}

// TestRegisterRoutes tests that routes are registered correctly
func TestRegisterRoutes(t *testing.T) {
	checker := NewHealthChecker()
	handler := NewHandler(checker)

	router := gin.New()
	handler.RegisterRoutes(router)

	// Test /health/live
	req1, _ := http.NewRequest("GET", "/health/live", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Test /health/ready
	req2, _ := http.NewRequest("GET", "/health/ready", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

// TestLivenessNotAffectedByDatabaseFailure tests that liveness is independent of database status
// Validates: Requirements 8.1
func TestLivenessNotAffectedByDatabaseFailure(t *testing.T) {
	checker := NewHealthChecker()
	
	// Register unhealthy database check
	dbCheck := &mockHealthCheck{
		name:    "database",
		healthy: false,
		err:     errors.New("database down"),
	}
	checker.RegisterCheck("database", dbCheck)

	handler := NewHandler(checker)
	router := gin.New()
	router.GET("/health/live", handler.LivenessHandler)

	req, _ := http.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Liveness should still return 200 even if database is down
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

// Mock health check implementation for testing
type mockHealthCheckHandler struct {
	name    string
	healthy bool
	err     error
	delay   time.Duration
	called  bool
}

func (m *mockHealthCheckHandler) Name() string {
	return m.name
}

func (m *mockHealthCheckHandler) Check(ctx context.Context) error {
	m.called = true
	
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	if !m.healthy {
		if m.err != nil {
			return m.err
		}
		return errors.New("health check failed")
	}
	
	return nil
}

func (m *mockHealthCheckHandler) Timeout() time.Duration {
	return 1 * time.Second
}
