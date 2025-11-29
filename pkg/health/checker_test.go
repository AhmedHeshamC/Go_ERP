package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestProperty13_ReadinessCheckDatabaseVerification tests Property 13
// **Feature: production-readiness, Property 13: Readiness Check Database Verification**
// **Validates: Requirements 8.2**
// For any readiness check request, if the database is unreachable, the system must return 503 status
func TestProperty13_ReadinessCheckDatabaseVerification(t *testing.T) {
	tests := []struct {
		name           string
		dbHealthy      bool
		redisHealthy   bool
		expectedStatus HealthStatus
		expectedCode   int
	}{
		{
			name:           "database unhealthy returns unhealthy status",
			dbHealthy:      false,
			redisHealthy:   true,
			expectedStatus: StatusUnhealthy,
			expectedCode:   503,
		},
		{
			name:           "database healthy and redis unhealthy returns healthy (redis not critical)",
			dbHealthy:      true,
			redisHealthy:   false,
			expectedStatus: StatusHealthy,
			expectedCode:   200,
		},
		{
			name:           "both healthy returns healthy",
			dbHealthy:      true,
			redisHealthy:   true,
			expectedStatus: StatusHealthy,
			expectedCode:   200,
		},
		{
			name:           "both unhealthy returns unhealthy",
			dbHealthy:      false,
			redisHealthy:   false,
			expectedStatus: StatusUnhealthy,
			expectedCode:   503,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock health checks
			dbCheck := &mockHealthCheck{
				name:    "database",
				healthy: tt.dbHealthy,
			}
			redisCheck := &mockHealthCheck{
				name:    "redis",
				healthy: tt.redisHealthy,
			}

			// Create health checker
			checker := NewHealthChecker()
			checker.RegisterCheck("database", dbCheck)
			checker.RegisterCheck("redis", redisCheck)

			// Run readiness check
			ctx := context.Background()
			status := checker.CheckReadiness(ctx)

			// Verify status
			assert.Equal(t, tt.expectedStatus, status.Status)

			// Verify HTTP status code mapping
			httpCode := status.HTTPStatusCode()
			assert.Equal(t, tt.expectedCode, httpCode)
		})
	}
}

// TestLivenessProbe tests that liveness probe returns 200 if app is running
// Validates: Requirements 8.1
func TestLivenessProbe(t *testing.T) {
	checker := NewHealthChecker()

	ctx := context.Background()
	status := checker.CheckLiveness(ctx)

	assert.Equal(t, StatusHealthy, status.Status)
	assert.Equal(t, 200, status.HTTPStatusCode())
	assert.NotEmpty(t, status.Timestamp)
}

// TestReadinessProbeChecksDatabase tests that readiness probe verifies database connectivity
// Validates: Requirements 8.2
func TestReadinessProbeChecksDatabase(t *testing.T) {
	t.Run("database healthy", func(t *testing.T) {
		dbCheck := &mockHealthCheck{
			name:    "database",
			healthy: true,
		}

		checker := NewHealthChecker()
		checker.RegisterCheck("database", dbCheck)

		ctx := context.Background()
		status := checker.CheckReadiness(ctx)

		assert.Equal(t, StatusHealthy, status.Status)
		assert.True(t, dbCheck.called)
	})

	t.Run("database unhealthy", func(t *testing.T) {
		dbCheck := &mockHealthCheck{
			name:    "database",
			healthy: false,
		}

		checker := NewHealthChecker()
		checker.RegisterCheck("database", dbCheck)

		ctx := context.Background()
		status := checker.CheckReadiness(ctx)

		assert.Equal(t, StatusUnhealthy, status.Status)
		assert.True(t, dbCheck.called)
		assert.Contains(t, status.Checks, "database")
		assert.Equal(t, StatusUnhealthy, status.Checks["database"].Status)
	})
}

// TestReadinessProbeChecksRedis tests that readiness probe verifies Redis connectivity
// Validates: Requirements 8.2
func TestReadinessProbeChecksRedis(t *testing.T) {
	dbCheck := &mockHealthCheck{
		name:    "database",
		healthy: true,
	}
	redisCheck := &mockHealthCheck{
		name:    "redis",
		healthy: true,
	}

	checker := NewHealthChecker()
	checker.RegisterCheck("database", dbCheck)
	checker.RegisterCheck("redis", redisCheck)

	ctx := context.Background()
	status := checker.CheckReadiness(ctx)

	assert.Equal(t, StatusHealthy, status.Status)
	assert.True(t, dbCheck.called)
	assert.True(t, redisCheck.called)
}

// TestHealthCheckTimeout tests that health checks complete within 1 second
// Validates: Requirements 8.5
func TestHealthCheckTimeout(t *testing.T) {
	slowCheck := &mockHealthCheck{
		name:    "slow",
		healthy: true,
		delay:   2 * time.Second, // Intentionally slow
	}

	checker := NewHealthChecker()
	checker.RegisterCheck("slow", slowCheck)

	ctx := context.Background()
	start := time.Now()
	status := checker.CheckReadiness(ctx)
	duration := time.Since(start)

	// Should timeout within 1 second
	assert.Less(t, duration, 1500*time.Millisecond, "Health check should timeout within 1 second")

	// Check should be marked as unhealthy due to timeout
	assert.Contains(t, status.Checks, "slow")
	checkResult := status.Checks["slow"]
	assert.Equal(t, StatusUnhealthy, checkResult.Status)
	assert.Contains(t, checkResult.Message, "timed out")
}

// TestHealthCheckReturns503WithDetails tests that unhealthy dependencies return 503 with details
// Validates: Requirements 8.3
func TestHealthCheckReturns503WithDetails(t *testing.T) {
	dbCheck := &mockHealthCheck{
		name:    "database",
		healthy: false,
		err:     errors.New("connection refused"),
	}

	checker := NewHealthChecker()
	checker.RegisterCheck("database", dbCheck)

	ctx := context.Background()
	status := checker.CheckReadiness(ctx)

	assert.Equal(t, StatusUnhealthy, status.Status)
	assert.Equal(t, 503, status.HTTPStatusCode())

	// Verify details are included
	assert.Contains(t, status.Checks, "database")
	dbResult := status.Checks["database"]
	assert.Equal(t, StatusUnhealthy, dbResult.Status)
	assert.NotEmpty(t, dbResult.Message)
	assert.NotNil(t, dbResult.Details)
}

// TestShutdownUpdatesReadiness tests that readiness returns 503 during shutdown
// Validates: Requirements 8.4
func TestShutdownUpdatesReadiness(t *testing.T) {
	dbCheck := &mockHealthCheck{
		name:    "database",
		healthy: true,
	}

	checker := NewHealthChecker()
	checker.RegisterCheck("database", dbCheck)

	ctx := context.Background()

	// Before shutdown - should be healthy
	status := checker.CheckReadiness(ctx)
	assert.Equal(t, StatusHealthy, status.Status)

	// Initiate shutdown
	checker.SetShuttingDown(true)

	// After shutdown - should be unhealthy
	status = checker.CheckReadiness(ctx)
	assert.Equal(t, StatusUnhealthy, status.Status)
	assert.Equal(t, 503, status.HTTPStatusCode())
}

// TestLivenessNotAffectedByDependencies tests that liveness is independent of dependencies
// Validates: Requirements 8.1
func TestLivenessNotAffectedByDependencies(t *testing.T) {
	dbCheck := &mockHealthCheck{
		name:    "database",
		healthy: false,
	}

	checker := NewHealthChecker()
	checker.RegisterCheck("database", dbCheck)

	ctx := context.Background()

	// Liveness should still be healthy even if database is down
	status := checker.CheckLiveness(ctx)
	assert.Equal(t, StatusHealthy, status.Status)
	assert.Equal(t, 200, status.HTTPStatusCode())
}

// TestMultipleDependencyFailures tests handling of multiple dependency failures
func TestMultipleDependencyFailures(t *testing.T) {
	dbCheck := &mockHealthCheck{
		name:    "database",
		healthy: false,
		err:     errors.New("db connection failed"),
	}
	redisCheck := &mockHealthCheck{
		name:    "redis",
		healthy: false,
		err:     errors.New("redis connection failed"),
	}

	checker := NewHealthChecker()
	checker.RegisterCheck("database", dbCheck)
	checker.RegisterCheck("redis", redisCheck)

	ctx := context.Background()
	status := checker.CheckReadiness(ctx)

	assert.Equal(t, StatusUnhealthy, status.Status)
	assert.Len(t, status.Checks, 2)

	// Both checks should be present with details
	assert.Contains(t, status.Checks, "database")
	assert.Contains(t, status.Checks, "redis")
	assert.Equal(t, StatusUnhealthy, status.Checks["database"].Status)
	assert.Equal(t, StatusUnhealthy, status.Checks["redis"].Status)
}

// TestHealthCheckWithContext tests that context cancellation is respected
func TestHealthCheckWithContext(t *testing.T) {
	slowCheck := &mockHealthCheck{
		name:    "slow",
		healthy: true,
		delay:   5 * time.Second,
	}

	checker := NewHealthChecker()
	checker.RegisterCheck("slow", slowCheck)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	status := checker.CheckReadiness(ctx)
	duration := time.Since(start)

	// Should respect context timeout
	assert.Less(t, duration, 1*time.Second)
	assert.Contains(t, status.Checks, "slow")
}

// Mock health check for testing
type mockHealthCheck struct {
	name    string
	healthy bool
	err     error
	delay   time.Duration
	called  bool
}

func (m *mockHealthCheck) Name() string {
	return m.name
}

func (m *mockHealthCheck) Check(ctx context.Context) error {
	m.called = true

	// Simulate delay if specified
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

func (m *mockHealthCheck) Timeout() time.Duration {
	return 1 * time.Second
}
