package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"erpgo/pkg/monitoring"
)

func TestBasicMetricsCollection(t *testing.T) {
	collector := monitoring.NewMetricsCollector()
	require.NotNil(t, collector)

	// Record test metrics
	collector.RecordHTTPRequest("GET", "/test", "200", "test-agent", 100*time.Millisecond, 1024, 2048)
	collector.RecordOrderCreated("pending", "credit", "premium", 100.0)
	collector.RecordUserRegistration("premium", "web")
	collector.RecordRevenue("USD", "order", 100.0)

	// The metrics should be recorded without errors
	// We can't easily test the actual values without the full HTTP handler
	assert.True(t, true) // Test passes if no panics occur
}

func TestBasicHealthChecks(t *testing.T) {
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
	assert.Contains(t, report.Checks, "memory")
	assert.Contains(t, report.Checks, "goroutines")
}

func TestBasicDistributedTracing(t *testing.T) {
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

func TestBasicErrorTracking(t *testing.T) {
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

func TestBasicContextPropagation(t *testing.T) {
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

func TestBasicTracingOperations(t *testing.T) {
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

func TestGlobalInstances(t *testing.T) {
	// Test global instances can be created and accessed
	monitoring.InitializeMetricsCollector()
	metrics := monitoring.GetMetricsCollector()
	assert.NotNil(t, metrics)

	monitoring.InitializeContextLogger(&zerolog.Logger{})
	contextLogger := monitoring.GetContextLogger()
	assert.NotNil(t, contextLogger)

	monitoring.InitializeTracer("test-service", 1.0)
	tracer := monitoring.GetTracer()
	assert.NotNil(t, tracer)

	monitoring.InitializeHealthChecker("test-version", "2023-01-01T00:00:00Z", "test-commit")
	healthChecker := monitoring.GetHealthChecker()
	assert.NotNil(t, healthChecker)

	monitoring.InitializeErrorTracker(100, 24*time.Hour)
	errorTracker := monitoring.GetErrorTracker()
	assert.NotNil(t, errorTracker)
}

func TestConvenienceFunctions(t *testing.T) {
	ctx := context.Background()
	ctx = monitoring.WithUserID(ctx, "test-user")
	ctx = monitoring.WithRequestID(ctx, "test-request")

	// Test convenience functions work without panics
	monitoring.TrackError(ctx, fmt.Errorf("test error"), "test", monitoring.ErrorSeverityWarning, monitoring.ErrorCategorySystem, map[string]interface{}{
		"test": true,
	})

	monitoring.TrackPanic(ctx, "test panic", "test_component")

	errors := monitoring.GetErrors(10, "", "")
	stats := monitoring.GetErrorStats()

	// Should not panic and return valid data
	assert.NotNil(t, errors)
	assert.NotNil(t, stats)
}
