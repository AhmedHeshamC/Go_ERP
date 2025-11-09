package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"erpgo/pkg/monitoring"
)

func TestMetricsCollection(t *testing.T) {
	collector := monitoring.NewMetricsCollector()
	require.NotNil(t, collector)

	// Record test metrics
	collector.RecordHTTPRequest("GET", "/test", "200", "test-agent", 100*time.Millisecond, 1024, 2048)
	collector.RecordOrderCreated("pending", "credit", "premium", 100.0)
	collector.RecordUserRegistration("premium", "web")
	collector.RecordRevenue("USD", "order", 100.0)

	// Test metrics handler
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler := promhttp.Handler()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "http_requests_total")
}

func TestHealthChecks(t *testing.T) {
	healthChecker := monitoring.NewHealthChecker("test-version", "2023-01-01T00:00:00Z", "test-commit")
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
}

func TestStructuredLogging(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	contextLogger := monitoring.NewContextLogger(&logger)
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

	log := contextLogger.WithContext(ctx, logCtx)
	require.NotNil(t, log)

	// Test different log levels
	log.Info("Test info message", map[string]interface{}{
		"test_key": "test_value",
	})

	log.Error("Test error message", fmt.Errorf("test error"), map[string]interface{}{
		"error_code": 500,
	})

	log.Audit("test_action", "test_user", "test_resource", map[string]interface{}{
		"ip_address": "127.0.0.1",
	})

	log.Performance("test_operation", 100*time.Millisecond, map[string]interface{}{
		"records_processed": 100,
	})
}

func TestDistributedTracing(t *testing.T) {
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

func TestErrorTracking(t *testing.T) {
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
}

func TestTracingOperations(t *testing.T) {
	tracer := monitoring.NewTracer("test-service", monitoring.NewProbabilitySampler(1.0))
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

	// Test tracing business operation
	err = monitoring.TraceBusiness(ctx, "order_creation", "user123", func(ctx context.Context) error {
		// Simulate business operation
		time.Sleep(3 * time.Millisecond)
		return nil
	})

	assert.NoError(t, err)
}

func TestMonitoringIntegration(t *testing.T) {
	// Setup test logger
	logger := zerolog.New(zerolog.NewTestWriter(t))

	// Create monitoring service config
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

	// Create monitoring service
	ms := monitoring.NewMonitoringService(config, logger)
	require.NotNil(t, ms)

	// Test components
	assert.NotNil(t, ms.GetMetricsCollector())
	assert.NotNil(t, ms.GetContextLogger())
	assert.NotNil(t, ms.GetTracer())
	assert.NotNil(t, ms.GetHealthChecker())
	assert.NotNil(t, ms.GetErrorTracker())

	// Test health checker setup
	healthChecker := ms.GetHealthChecker()
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

	// Test health report
	report := healthChecker.GetHealthReport(context.Background())
	assert.Equal(t, monitoring.HealthStatusHealthy, report.Status)

	// Test metrics collection
	collector := ms.GetMetricsCollector()
	collector.RecordHTTPRequest("GET", "/test", "200", "test-agent", 100*time.Millisecond, 1024, 2048)
	collector.RecordOrderCreated("pending", "credit", "premium", 100.0)

	// Test context logging
	contextLogger := ms.GetContextLogger()
	ctx := context.Background()
	ctx = monitoring.WithCorrelationID(ctx, "test-correlation-id")
	ctx = monitoring.WithTraceID(ctx, "test-trace-id")

	logCtx := monitoring.LogContext{
		CorrelationID: "test-correlation-id",
		TraceID:       "test-trace-id",
		Component:     "test-component",
	}

	log := contextLogger.WithContext(ctx, logCtx)
	log.Info("Test integration message", map[string]interface{}{
		"integration": true,
	})

	// Test error tracking
	errorTracker := ms.GetErrorTracker()
	errorTracker.TrackError(ctx, fmt.Errorf("integration test error"), "test", monitoring.ErrorSeverityWarning, monitoring.ErrorCategorySystem, map[string]interface{}{
		"test": true,
	})

	// Test shutdown
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ms.Shutdown(ctxShutdown)
	assert.NoError(t, err)
}

func TestMonitoringHTTPMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup test logger
	logger := zerolog.New(zerolog.NewTestWriter(t))

	// Create monitoring service
	config := monitoring.MonitoringConfig{
		Version:              "test-version",
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

	// Setup router with monitoring
	router := gin.New()
	router.Use(ms.Middleware())
	router.Use(ms.RecoverMiddleware())
	ms.SetupRoutes(router, config)

	// Test basic request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code) // 404 because no route matches /test

	// Test health endpoint
	req = httptest.NewRequest("GET", "/health", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test metrics endpoint
	req = httptest.NewRequest("GET", "/metrics", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "http_requests_total")
}

func TestContextPropagation(t *testing.T) {
	// Test correlation ID propagation
	ctx := context.Background()
	ctx = monitoring.WithCorrelationID(ctx, "test-correlation-123")
	ctx = monitoring.WithTraceID(ctx, "test-trace-456")
	ctx = monitoring.WithSpanID(ctx, "test-span-789")
	ctx = monitoring.WithRequestID(ctx, "test-request-012")
	ctx = monitoring.WithUserID(ctx, "test-user-345")
	ctx = monitoring.WithSessionID(ctx, "test-session-678")

	// Verify context values
	correlationID := monitoring.GetCorrelationIDFromContext(ctx)
	assert.Equal(t, "test-correlation-123", correlationID)

	traceID := monitoring.GetTraceIDFromContext(ctx)
	assert.Equal(t, "test-trace-456", traceID)

	spanID := monitoring.GetSpanIDFromContext(ctx)
	assert.Equal(t, "test-span-789", spanID)

	// Test ID generation
	newCorrelationID := monitoring.GenerateCorrelationID()
	assert.NotEmpty(t, newCorrelationID)
	assert.NotEqual(t, "test-correlation-123", newCorrelationID)

	newTraceID := monitoring.GenerateTraceID()
	assert.NotEmpty(t, newTraceID)
	assert.NotEqual(t, "test-trace-456", newTraceID)

	newSpanID := monitoring.GenerateSpanID()
	assert.NotEmpty(t, newSpanID)
	assert.NotEqual(t, "test-span-789", newSpanID)
}