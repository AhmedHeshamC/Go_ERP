package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"erpgo/pkg/monitoring"
)

func TestMonitoringService(t *testing.T) {
	// Setup test logger
	logger := zerolog.New(zerolog.NewTestWriter(t))

	// Create monitoring service
	config := monitoring.MonitoringConfig{
		Version:              "test-version",
		BuildTime:            "2023-01-01T00:00:00Z",
		Commit:               "test-commit",
		LogLevel:             "info",
		Environment:          "test",
		ServiceName:          "erpgo-test",
		EnableMetrics:        true,
		EnableTracing:        true,
		EnableErrorTracking:  true,
		TracingSampleRate:    1.0,
		MetricsPath:          "/metrics",
		HealthPath:           "/health",
	}

	ms := monitoring.NewMonitoringService(config, logger)
	require.NotNil(t, ms)

	// Test metrics collection
	t.Run("MetricsCollection", func(t *testing.T) {
		collector := ms.GetMetricsCollector()
		require.NotNil(t, collector)

		// Record some metrics
		collector.RecordHTTPRequest("GET", "/test", "200", "test-agent", 100*time.Millisecond, 1024, 2048)
		collector.RecordOrderCreated("pending", "credit", "premium", 100.0)
		collector.RecordUserRegistration("premium", "web")
		collector.RecordRevenue("USD", "order", 100.0)

		// Test metrics handler
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		handler := collector.GetMetricsHandler()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "erpgo_http_requests_total")
		assert.Contains(t, w.Body.String(), "erpgo_orders_created_total")
		assert.Contains(t, w.Body.String(), "erpgo_revenue_total")
	})

	// Test structured logging
	t.Run("StructuredLogging", func(t *testing.T) {
		contextLogger := ms.GetContextLogger()
		require.NotNil(t, contextLogger)

		ctx := context.Background()
		ctx = monitoring.WithCorrelationID(ctx, "test-correlation-id")
		ctx = monitoring.WithTraceID(ctx, "test-trace-id")
		ctx = monitoring.WithRequestID(ctx, "test-request-id")
		ctx = monitoring.WithUserID(ctx, "test-user-id")

		logCtx := monitoring.LogContext{
			CorrelationID: "test-correlation-id",
			TraceID:       "test-trace-id",
			SpanID:        "test-span-id",
			UserID:        "test-user-id",
			RequestID:     "test-request-id",
			Component:     "test-component",
		}

		logger := contextLogger.WithContext(ctx, logCtx)
		require.NotNil(t, logger)

		// Test different log levels
		logger.Info("Test info message", map[string]interface{}{
			"test_key": "test_value",
		})

		logger.Error("Test error message", fmt.Errorf("test error"), map[string]interface{}{
			"error_code": 500,
		})

		logger.Audit("test_action", "test_user", "test_resource", map[string]interface{}{
			"ip_address": "127.0.0.1",
		})

		logger.Performance("test_operation", 100*time.Millisecond, map[string]interface{}{
			"records_processed": 100,
		})
	})

	// Test distributed tracing
	t.Run("DistributedTracing", func(t *testing.T) {
		tracer := ms.GetTracer()
		require.NotNil(t, tracer)

		ctx := context.Background()

		// Test tracing a function
		err := monitoring.TraceFunction(ctx, "test_function", func(ctx context.Context) error {
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			return nil
		})

		assert.NoError(t, err)

		// Test tracing HTTP request
		err = monitoring.TraceHTTP(ctx, "GET", "/api/test", func(ctx context.Context) error {
			// Simulate HTTP request processing
			time.Sleep(5 * time.Millisecond)
			return nil
		})

		assert.NoError(t, err)

		// Test tracing database operation
		err = monitoring.TraceDatabase(ctx, "SELECT", "users", func(ctx context.Context) error {
			// Simulate database query
			time.Sleep(2 * time.Millisecond)
			return nil
		})

		assert.NoError(t, err)
	})

	// Test health checks
	t.Run("HealthChecks", func(t *testing.T) {
		healthChecker := ms.GetHealthChecker()
		require.NotNil(t, healthChecker)

		// Setup test health checks
		healthChecker.SetupDefaultChecks(
			func(ctx context.Context) monitoring.HealthCheckResult {
				return monitoring.HealthCheckResult{
					Status:    monitoring.HealthStatusHealthy,
					Message:   "Database is healthy",
					Timestamp: time.Now(),
				}
			},
			func(ctx context.Context) monitoring.HealthCheckResult {
				return monitoring.HealthCheckResult{
					Status:    monitoring.HealthStatusHealthy,
					Message:   "Cache is healthy",
					Timestamp: time.Now(),
				}
			},
		)

		// Run all checks
		report := healthChecker.GetHealthReport(context.Background())
		assert.Equal(t, monitoring.HealthStatusHealthy, report.Status)
		assert.Contains(t, report.Checks, "database")
		assert.Contains(t, report.Checks, "cache")
		assert.Contains(t, report.Checks, "memory")
		assert.Contains(t, report.Checks, "goroutines")
	})

	// Test error tracking
	t.Run("ErrorTracking", func(t *testing.T) {
		errorTracker := ms.GetErrorTracker()
		require.NotNil(t, errorTracker)

		ctx := context.Background()
		ctx = monitoring.WithUserID(ctx, "test-user-id")
		ctx = monitoring.WithRequestID(ctx, "test-request-id")

		// Track different types of errors
		errorTracker.TrackError(ctx, fmt.Errorf("test database error"), "database", monitoring.ErrorSeverityError, monitoring.ErrorCategoryDatabase, map[string]interface{}{
			"query": "SELECT * FROM users",
		})

		errorTracker.TrackError(ctx, fmt.Errorf("test network error"), "http", monitoring.ErrorSeverityWarning, monitoring.ErrorCategoryNetwork, map[string]interface{}{
			"endpoint": "/api/test",
		})

		errorTracker.TrackError(ctx, fmt.Errorf("test validation error"), "validation", monitoring.ErrorSeverityInfo, monitoring.ErrorCategoryValidation, map[string]interface{}{
			"field": "email",
		})

		// Test panic tracking
		defer func() {
			if r := recover(); r != nil {
				errorTracker.TrackPanic(ctx, r, "test_component")
			}
		}()

		// Get error statistics
		stats := errorTracker.GetErrorStats()
		assert.Contains(t, stats, "total_errors")
		assert.Contains(t, stats, "severity_breakdown")
		assert.Contains(t, stats, "category_breakdown")

		// Get recent errors
		errors := errorTracker.GetErrors(10, "", "")
		assert.Len(t, errors, 3) // We tracked 3 errors above
	})

	// Test HTTP middleware integration
	t.Run("HTTPMiddleware", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		// Add monitoring middleware
		router.Use(ms.Middleware())
		router.Use(ms.RecoverMiddleware())

		// Add monitoring routes
		ms.SetupRoutes(router, config)

		// Test health endpoint
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Test detailed health endpoint
		req = httptest.NewRequest("GET", "/health/detailed", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Test metrics endpoint
		req = httptest.NewRequest("GET", "/metrics", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Test debug endpoints
		req = httptest.NewRequest("GET", "/debug/vars", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		req = httptest.NewRequest("GET", "/debug/stats", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		req = httptest.NewRequest("GET", "/debug/errors", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		req = httptest.NewRequest("GET", "/monitoring", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "<!DOCTYPE html>")
	})

	// Test shutdown
	t.Run("Shutdown", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := ms.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestMonitoringMetricsIntegration(t *testing.T) {
	// Create a test registry
	registry := prometheus.NewRegistry()

	// Create metrics collector
	collector := monitoring.NewMetricsCollector()
	registry.MustRegister(collector.httpRequestsTotal)
	registry.MustRegister(collector.httpRequestDuration)
	registry.MustRegister(collector.ordersCreated)
	registry.MustRegister(collector.revenueTotal)

	// Record test metrics
	collector.RecordHTTPRequest("GET", "/api/users", "200", "test-agent", 50*time.Millisecond, 1024, 2048)
	collector.RecordOrderCreated("pending", "credit", "premium", 250.50)
	collector.RecordRevenue("USD", "order", 250.50)

	// Test metric collection
	metricFamilies, err := registry.Gather()
	assert.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Verify specific metrics exist
	var foundHTTPRequests, foundOrderCreated, foundRevenue bool
	for _, family := range metricFamilies {
		switch *family.Name {
		case "erpgo_http_requests_total":
			foundHTTPRequests = true
		case "erpgo_orders_created_total":
			foundOrderCreated = true
		case "erpgo_revenue_total":
			foundRevenue = true
		}
	}

	assert.True(t, foundHTTPRequests, "HTTP requests metric should be present")
	assert.True(t, foundOrderCreated, "Orders created metric should be present")
	assert.True(t, foundRevenue, "Revenue metric should be present")
}

func TestMonitoringTracingIntegration(t *testing.T) {
	tracer := monitoring.NewTracer("test-service", monitoring.NewProbabilitySampler(1.0))
	require.NotNil(t, tracer)

	ctx := context.Background()

	// Test span creation and finishing
	ctx, span := tracer.StartSpan(ctx, "test_operation",
		monitoring.WithResource("test-resource"),
		monitoring.WithTags(map[string]interface{}{
			"test_key": "test_value",
		}),
	)
	require.NotNil(t, span)
	require.NotEmpty(t, span.TraceID)
	require.NotEmpty(t, span.SpanID)

	// Add logs to span
	tracer.LogEvent(span, "info", "Test log message", map[string]interface{}{
		"log_field": "log_value",
	})

	// Set tags
	tracer.SetTag(span, "additional_tag", "additional_value")

	// Finish span
	tracer.FinishSpan(span, monitoring.WithStatus(0, "Success"))

	// Verify span attributes
	assert.Equal(t, "test_operation", span.OperationName)
	assert.Equal(t, "test-resource", span.Resource)
	assert.Equal(t, "Success", span.Status.Message)
	assert.NotZero(t, span.Duration)
	assert.NotEmpty(t, span.Logs)
}

func TestMonitoringErrorTrackingIntegration(t *testing.T) {
	errorTracker := monitoring.NewErrorTracker(100, 1*time.Hour)
	require.NotNil(t, errorTracker)

	ctx := context.Background()
	ctx = monitoring.WithUserID(ctx, "test-user")
	ctx = monitoring.WithRequestID(ctx, "test-request")

	// Track different error types
	testErrors := []struct {
		err       error
		component string
		severity  monitoring.ErrorSeverity
		category  monitoring.ErrorCategory
	}{
		{
			err:       fmt.Errorf("database connection failed"),
			component: "database",
			severity:  monitoring.ErrorSeverityCritical,
			category:  monitoring.ErrorCategoryDatabase,
		},
		{
			err:       fmt.Errorf("invalid input"),
			component: "validation",
			severity:  monitoring.ErrorSeverityWarning,
			category:  monitoring.ErrorCategoryValidation,
		},
		{
			err:       fmt.Errorf("network timeout"),
			component: "http",
			severity:  monitoring.ErrorSeverityError,
			category:  monitoring.ErrorCategoryNetwork,
		},
	}

	for _, testErr := range testErrors {
		errorTracker.TrackError(ctx, testErr.err, testErr.component, testErr.severity, testErr.category, map[string]interface{}{
			"test_context": "test_value",
		})
	}

	// Test error aggregation
	aggregations := errorTracker.GetAggregations()
	assert.NotEmpty(t, aggregations)

	// Test error statistics
	stats := errorTracker.GetErrorStats()
	assert.Contains(t, stats, "total_errors")
	assert.Contains(t, stats, "severity_breakdown")
	assert.Contains(t, stats, "category_breakdown")

	// Test error filtering by severity
	warningErrors := errorTracker.GetErrors(10, monitoring.ErrorSeverityWarning, "")
	assert.NotEmpty(t, warningErrors)

	// Test error filtering by category
	dbErrors := errorTracker.GetErrors(10, "", monitoring.ErrorCategoryDatabase)
	assert.NotEmpty(t, dbErrors)

	// Test panic tracking
	defer func() {
		if r := recover(); r != nil {
			errorTracker.TrackPanic(ctx, r, "test_component")
		}
	}()

	// Simulate a panic in a controlled way
	func() {
		defer func() {
			if r := recover(); r != nil {
				errorTracker.TrackPanic(ctx, r, "test_component")
			}
		}()
		panic("test panic")
	}()
}