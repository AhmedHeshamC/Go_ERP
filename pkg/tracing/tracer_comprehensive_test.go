package tracing

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// Mock exporter for testing
type mockExporter struct {
	exportedSpans []*Span
	exportError   error
	shutdownError error
}

func (m *mockExporter) ExportSpans(spans []*Span) error {
	if m.exportError != nil {
		return m.exportError
	}
	m.exportedSpans = append(m.exportedSpans, spans...)
	return nil
}

func (m *mockExporter) Shutdown(ctx context.Context) error {
	return m.shutdownError
}

func TestTracerCreation(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("NewTracerWithDefaultConfig", func(t *testing.T) {
		config := DefaultConfig()
		tracer, err := NewTracer(config, &logger)
		if err != nil {
			t.Fatalf("Failed to create tracer: %v", err)
		}
		if tracer == nil {
			t.Fatal("Expected tracer, got nil")
		}
		defer tracer.Shutdown(context.Background())
	})

	t.Run("NewTracerWithNilConfig", func(t *testing.T) {
		tracer, err := NewTracer(nil, &logger)
		if err != nil {
			t.Fatalf("Failed to create tracer with nil config: %v", err)
		}
		if tracer == nil {
			t.Fatal("Expected tracer, got nil")
		}
		defer tracer.Shutdown(context.Background())
	})

	t.Run("NewTracerWithNilLogger", func(t *testing.T) {
		config := DefaultConfig()
		tracer, err := NewTracer(config, nil)
		if err != nil {
			t.Fatalf("Failed to create tracer with nil logger: %v", err)
		}
		if tracer == nil {
			t.Fatal("Expected tracer, got nil")
		}
		defer tracer.Shutdown(context.Background())
	})

	t.Run("ProductionConfig", func(t *testing.T) {
		config := ProductionConfig()
		if config.SampleRate >= 1.0 {
			t.Error("Expected production sample rate < 1.0")
		}
		if config.DebugMode {
			t.Error("Expected debug mode to be false in production")
		}
	})
}

func TestSpanOperations(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.EnableSanitization = false // Disable sanitization for these tests
	tracer, _ := NewTracer(config, &logger)
	defer tracer.Shutdown(context.Background())

	t.Run("StartAndFinishSpan", func(t *testing.T) {
		ctx := context.Background()
		newCtx, span := tracer.StartSpan(ctx, "test-operation", SpanKindServer)

		if span == nil {
			t.Fatal("Expected span, got nil")
		}

		if span.TraceID == "" {
			t.Error("Expected trace ID to be set")
		}

		if span.SpanID == "" {
			t.Error("Expected span ID to be set")
		}

		if span.OperationName != "test-operation" {
			t.Errorf("Expected operation name 'test-operation', got %s", span.OperationName)
		}

		if newCtx == ctx {
			t.Error("Expected new context to be different from original")
		}

		tracer.FinishSpan(span)

		if span.EndTime.IsZero() {
			t.Error("Expected end time to be set")
		}

		if span.Duration == 0 {
			t.Error("Expected duration to be set")
		}
	})

	t.Run("SetAttribute", func(t *testing.T) {
		ctx := context.Background()
		_, span := tracer.StartSpan(ctx, "test-operation", SpanKindInternal)

		tracer.SetAttribute(span, "test.key", "test.value")

		if val, ok := span.Attributes["test.key"]; !ok || val != "test.value" {
			t.Errorf("Expected attribute 'test.key' = 'test.value', got %v", val)
		}

		tracer.FinishSpan(span)
	})

	t.Run("AddEvent", func(t *testing.T) {
		ctx := context.Background()
		_, span := tracer.StartSpan(ctx, "test-operation", SpanKindInternal)

		tracer.AddEvent(span, "test-event", map[string]interface{}{
			"event.key": "event.value",
		})

		if len(span.Events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(span.Events))
		}

		if span.Events[0].Name != "test-event" {
			t.Errorf("Expected event name 'test-event', got %s", span.Events[0].Name)
		}

		tracer.FinishSpan(span)
	})

	t.Run("SetStatus", func(t *testing.T) {
		ctx := context.Background()
		_, span := tracer.StartSpan(ctx, "test-operation", SpanKindInternal)

		tracer.SetStatus(span, SpanStatusError, "test error")

		if span.Status.Code != SpanStatusError {
			t.Errorf("Expected status code %d, got %d", SpanStatusError, span.Status.Code)
		}

		if span.Status.Message != "test error" {
			t.Errorf("Expected status message 'test error', got %s", span.Status.Message)
		}

		tracer.FinishSpan(span)
	})

	t.Run("SetError", func(t *testing.T) {
		ctx := context.Background()
		_, span := tracer.StartSpan(ctx, "test-operation", SpanKindInternal)

		testErr := errors.New("test error")
		tracer.SetError(span, testErr)

		if span.Status.Code != SpanStatusError {
			t.Errorf("Expected status code %d, got %d", SpanStatusError, span.Status.Code)
		}

		if span.Status.Error != testErr {
			t.Errorf("Expected error %v, got %v", testErr, span.Status.Error)
		}

		// Should have added an error event
		hasErrorEvent := false
		for _, event := range span.Events {
			if event.Name == "error" {
				hasErrorEvent = true
				break
			}
		}

		if !hasErrorEvent {
			t.Error("Expected error event to be added")
		}

		tracer.FinishSpan(span)
	})

	t.Run("AddLink", func(t *testing.T) {
		ctx := context.Background()
		_, span := tracer.StartSpan(ctx, "test-operation", SpanKindInternal)

		tracer.AddLink(span, "trace-id-123", "span-id-456", map[string]interface{}{
			"link.key": "link.value",
		})

		if len(span.Links) != 1 {
			t.Errorf("Expected 1 link, got %d", len(span.Links))
		}

		if span.Links[0].TraceID != "trace-id-123" {
			t.Errorf("Expected trace ID 'trace-id-123', got %s", span.Links[0].TraceID)
		}

		tracer.FinishSpan(span)
	})
}

func TestSpanContext(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	tracer, _ := NewTracer(config, &logger)
	defer tracer.Shutdown(context.Background())

	t.Run("GetActiveSpan", func(t *testing.T) {
		ctx := context.Background()
		newCtx, span := tracer.StartSpan(ctx, "test-operation", SpanKindServer)

		activeSpan := tracer.GetActiveSpan(newCtx)
		if activeSpan == nil {
			t.Error("Expected active span, got nil")
		}

		if activeSpan.SpanID != span.SpanID {
			t.Errorf("Expected span ID %s, got %s", span.SpanID, activeSpan.SpanID)
		}

		tracer.FinishSpan(span)
	})

	t.Run("GetTraceContext", func(t *testing.T) {
		ctx := context.Background()
		newCtx, span := tracer.StartSpan(ctx, "test-operation", SpanKindServer)

		traceContext := tracer.GetTraceContext(newCtx)
		if traceContext == nil {
			t.Fatal("Expected trace context, got nil")
		}

		if traceContext.TraceID != span.TraceID {
			t.Errorf("Expected trace ID %s, got %s", span.TraceID, traceContext.TraceID)
		}

		tracer.FinishSpan(span)
	})
}

func TestTracerWithExporter(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.AsyncExport = false

	mockExp := &mockExporter{}
	config.Exporter = mockExp

	tracer, _ := NewTracer(config, &logger)
	defer tracer.Shutdown(context.Background())

	t.Run("ExportSpan", func(t *testing.T) {
		ctx := context.Background()
		_, span := tracer.StartSpan(ctx, "test-operation", SpanKindServer)
		tracer.FinishSpan(span)

		// Give some time for export
		time.Sleep(10 * time.Millisecond)

		if len(mockExp.exportedSpans) == 0 {
			t.Error("Expected spans to be exported")
		}
	})

	t.Run("ExportError", func(t *testing.T) {
		mockExp.exportError = errors.New("export error")
		mockExp.exportedSpans = nil

		ctx := context.Background()
		_, span := tracer.StartSpan(ctx, "test-operation", SpanKindServer)
		tracer.FinishSpan(span)

		// Error should be logged but not returned
		time.Sleep(10 * time.Millisecond)
	})
}

func TestSanitization(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.EnableSanitization = true
	config.SanitizeAttributes = []string{"password", "secret", "token"}

	tracer, _ := NewTracer(config, &logger)
	defer tracer.Shutdown(context.Background())

	t.Run("SanitizeAttributes", func(t *testing.T) {
		ctx := context.Background()
		_, span := tracer.StartSpan(ctx, "test-operation", SpanKindInternal)

		tracer.SetAttribute(span, "password", "secret123")
		tracer.SetAttribute(span, "username", "testuser")

		if span.Attributes["password"] != "[REDACTED]" {
			t.Errorf("Expected password to be redacted, got %v", span.Attributes["password"])
		}

		if span.Attributes["username"] != "testuser" {
			t.Errorf("Expected username to be preserved, got %v", span.Attributes["username"])
		}

		tracer.FinishSpan(span)
	})
}

func TestHTTPHeaderExtraction(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.PropagationFormat = "w3c"

	tracer, _ := NewTracer(config, &logger)
	defer tracer.Shutdown(context.Background())

	t.Run("ExtractW3CTraceContext", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")

		traceContext := tracer.extractTraceContextFromHeaders(headers)

		if traceContext.TraceID != "0af7651916cd43dd8448eb211c80319c" {
			t.Errorf("Expected trace ID to be extracted, got %s", traceContext.TraceID)
		}

		if traceContext.SpanID != "b7ad6b7169203331" {
			t.Errorf("Expected span ID to be extracted, got %s", traceContext.SpanID)
		}

		if !traceContext.Sampled {
			t.Error("Expected sampled to be true")
		}
	})

	t.Run("ExtractB3TraceContext", func(t *testing.T) {
		config.PropagationFormat = "b3"
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(context.Background())

		headers := http.Header{}
		headers.Set("b3", "0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-1")

		traceContext := tracer.extractTraceContextFromHeaders(headers)

		if traceContext.TraceID != "0af7651916cd43dd8448eb211c80319c" {
			t.Errorf("Expected trace ID to be extracted, got %s", traceContext.TraceID)
		}
	})

	t.Run("ExtractB3MultipleHeaders", func(t *testing.T) {
		config.PropagationFormat = "b3"
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(context.Background())

		headers := http.Header{}
		headers.Set("x-b3-traceid", "0af7651916cd43dd8448eb211c80319c")
		headers.Set("x-b3-spanid", "b7ad6b7169203331")
		headers.Set("x-b3-sampled", "1")

		traceContext := tracer.extractTraceContextFromHeaders(headers)

		if traceContext.TraceID != "0af7651916cd43dd8448eb211c80319c" {
			t.Errorf("Expected trace ID to be extracted, got %s", traceContext.TraceID)
		}

		if traceContext.SpanID != "b7ad6b7169203331" {
			t.Errorf("Expected span ID to be extracted, got %s", traceContext.SpanID)
		}

		if !traceContext.Sampled {
			t.Error("Expected sampled to be true")
		}
	})

	t.Run("ExtractBaggage", func(t *testing.T) {
		config.EnableBaggage = true
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(context.Background())

		headers := http.Header{}
		headers.Set("baggage", "key1=value1, key2=value2")

		traceContext := tracer.extractTraceContextFromHeaders(headers)

		if traceContext.BaggageItems["key1"] != "value1" {
			t.Errorf("Expected baggage key1=value1, got %v", traceContext.BaggageItems["key1"])
		}

		if traceContext.BaggageItems["key2"] != "value2" {
			t.Errorf("Expected baggage key2=value2, got %v", traceContext.BaggageItems["key2"])
		}
	})
}

func TestGetStats(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	tracer, _ := NewTracer(config, &logger)
	defer tracer.Shutdown(context.Background())

	stats := tracer.GetStats()

	if stats == nil {
		t.Fatal("Expected stats, got nil")
	}

	if stats["service_name"] != config.ServiceName {
		t.Errorf("Expected service name %s, got %v", config.ServiceName, stats["service_name"])
	}

	if stats["initialized"] != true {
		t.Error("Expected initialized to be true")
	}
}

func TestGlobalTracer(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.EnableSanitization = false // Disable sanitization for these tests

	t.Run("InitGlobalTracer", func(t *testing.T) {
		err := InitGlobalTracer(config, &logger)
		if err != nil {
			t.Fatalf("Failed to init global tracer: %v", err)
		}
	})

	t.Run("GetGlobalTracer", func(t *testing.T) {
		tracer := GetGlobalTracer()
		if tracer == nil {
			t.Fatal("Expected global tracer, got nil")
		}
	})

	t.Run("GlobalStartSpan", func(t *testing.T) {
		ctx := context.Background()
		newCtx, span := GlobalStartSpan(ctx, "global-test", SpanKindServer)

		if span == nil {
			t.Fatal("Expected span, got nil")
		}

		if newCtx == ctx {
			t.Error("Expected new context")
		}

		GlobalFinishSpan(span)
	})

	t.Run("GlobalSetAttribute", func(t *testing.T) {
		ctx := context.Background()
		_, span := GlobalStartSpan(ctx, "global-test", SpanKindInternal)

		GlobalSetAttribute(span, "test.key", "test.value")

		if val, ok := span.Attributes["test.key"]; !ok || val != "test.value" {
			t.Errorf("Expected attribute 'test.key' = 'test.value', got %v", val)
		}

		GlobalFinishSpan(span)
	})

	t.Run("GlobalAddEvent", func(t *testing.T) {
		ctx := context.Background()
		_, span := GlobalStartSpan(ctx, "global-test", SpanKindInternal)

		GlobalAddEvent(span, "test-event", map[string]interface{}{
			"event.key": "event.value",
		})

		if len(span.Events) == 0 {
			t.Error("Expected event to be added")
		}

		GlobalFinishSpan(span)
	})

	t.Run("GlobalSetError", func(t *testing.T) {
		ctx := context.Background()
		_, span := GlobalStartSpan(ctx, "global-test", SpanKindInternal)

		testErr := errors.New("global test error")
		GlobalSetError(span, testErr)

		if span.Status.Code != SpanStatusError {
			t.Error("Expected error status")
		}

		GlobalFinishSpan(span)
	})

	t.Run("GlobalGetTraceContext", func(t *testing.T) {
		ctx := context.Background()
		newCtx, span := GlobalStartSpan(ctx, "global-test", SpanKindServer)

		traceContext := GlobalGetTraceContext(newCtx)

		if traceContext == nil {
			t.Fatal("Expected trace context, got nil")
		}

		if traceContext.TraceID != span.TraceID {
			t.Error("Expected matching trace IDs")
		}

		GlobalFinishSpan(span)
	})
}

func TestNilSpanHandling(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	tracer, _ := NewTracer(config, &logger)
	defer tracer.Shutdown(context.Background())

	t.Run("FinishNilSpan", func(t *testing.T) {
		// Should not panic
		tracer.FinishSpan(nil)
	})

	t.Run("SetAttributeNilSpan", func(t *testing.T) {
		// Should not panic
		tracer.SetAttribute(nil, "key", "value")
	})

	t.Run("AddEventNilSpan", func(t *testing.T) {
		// Should not panic
		tracer.AddEvent(nil, "event", nil)
	})

	t.Run("SetStatusNilSpan", func(t *testing.T) {
		// Should not panic
		tracer.SetStatus(nil, SpanStatusOK, "message")
	})

	t.Run("SetErrorNilSpan", func(t *testing.T) {
		// Should not panic
		tracer.SetError(nil, errors.New("test"))
	})

	t.Run("AddLinkNilSpan", func(t *testing.T) {
		// Should not panic
		tracer.AddLink(nil, "trace", "span", nil)
	})
}

func TestSpanLimits(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.MaxAttributesPerSpan = 2
	config.MaxEventsPerSpan = 2
	config.MaxLinksPerSpan = 2

	tracer, _ := NewTracer(config, &logger)
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	_, span := tracer.StartSpan(ctx, "test-operation", SpanKindInternal)

	// Try to add more attributes than limit
	for i := 0; i < 5; i++ {
		tracer.SetAttribute(span, "key", "value")
	}

	// Try to add more events than limit
	for i := 0; i < 5; i++ {
		tracer.AddEvent(span, "event", nil)
	}

	// Try to add more links than limit
	for i := 0; i < 5; i++ {
		tracer.AddLink(span, "trace", "span", nil)
	}

	tracer.FinishSpan(span)
}
