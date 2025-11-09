package monitoring

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/codes"
)

// OpenTelemetryConfig holds configuration for OpenTelemetry setup
type OpenTelemetryConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string

	// Jaeger configuration
	JaegerEndpoint string

	// Zipkin configuration
	ZipkinEndpoint string

	// Sampling configuration
	SampleRate float64

	// Additional resource attributes
	ResourceAttributes map[string]string
}

// OpenTelemetryTracer wraps OpenTelemetry tracing capabilities
type OpenTelemetryTracer struct {
	tracer trace.Tracer
	config OpenTelemetryConfig
}

// NewOpenTelemetryTracer creates a new OpenTelemetry tracer
func NewOpenTelemetryTracer(config OpenTelemetryConfig) (*OpenTelemetryTracer, error) {
	// Create tracer provider
	tp, err := createTraceProvider(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace provider: %w", err)
	}

	// Register as global tracer provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := otel.Tracer(config.ServiceName)

	return &OpenTelemetryTracer{
		tracer: tracer,
		config: config,
	}, nil
}

// createTraceProvider creates a configured trace provider
func createTraceProvider(config OpenTelemetryConfig) (*sdktrace.TracerProvider, error) {
	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", config.ServiceName),
			attribute.String("service.version", config.ServiceVersion),
			attribute.String("service.environment", config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Add additional resource attributes
	if len(config.ResourceAttributes) > 0 {
		var attrs []attribute.KeyValue
		for k, v := range config.ResourceAttributes {
			attrs = append(attrs, attribute.String(k, v))
		}
		res, err = resource.Merge(res, resource.NewWithAttributes("service", attrs...))
		if err != nil {
			return nil, fmt.Errorf("failed to merge resource attributes: %w", err)
		}
	}

	var exporter sdktrace.SpanExporter

	// Choose exporter based on configuration
	if config.JaegerEndpoint != "" {
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))
	} else if config.ZipkinEndpoint != "" {
		exporter, err = zipkin.New(config.ZipkinEndpoint)
	} else {
		// Default to stdout exporter for development
		exporter, err = stdouttrace.New()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create span exporter: %w", err)
	}

	// Create sampler
	sampler := sdktrace.AlwaysSample()
	if config.SampleRate < 1.0 {
		sampler = sdktrace.TraceIDRatioBased(config.SampleRate)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	return tp, nil
}

// StartSpan starts a new OpenTelemetry span
func (ot *OpenTelemetryTracer) StartSpan(ctx context.Context, operationName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ot.tracer.Start(ctx, operationName, opts...)
}

// TraceHTTP traces an HTTP request with OpenTelemetry
func (ot *OpenTelemetryTracer) TraceHTTP(ctx context.Context, method, url string, fn func(context.Context) error) error {
	ctx, span := ot.StartSpan(ctx, fmt.Sprintf("HTTP %s", method),
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.url", url),
			attribute.String("component", "http"),
		),
	)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error", err.Error()))
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// TraceDatabase traces a database operation with OpenTelemetry
func (ot *OpenTelemetryTracer) TraceDatabase(ctx context.Context, operation, table string, fn func(context.Context) error) error {
	ctx, span := ot.StartSpan(ctx, fmt.Sprintf("DB %s", operation),
		trace.WithAttributes(
			attribute.String("db.operation", operation),
			attribute.String("db.table", table),
			attribute.String("db.system", "postgresql"),
			attribute.String("component", "database"),
		),
	)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error", err.Error()))
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// TraceCache traces a cache operation with OpenTelemetry
func (ot *OpenTelemetryTracer) TraceCache(ctx context.Context, operation string, fn func(context.Context) error) error {
	ctx, span := ot.StartSpan(ctx, fmt.Sprintf("Cache %s", operation),
		trace.WithAttributes(
			attribute.String("cache.operation", operation),
			attribute.String("component", "cache"),
		),
	)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error", err.Error()))
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// TraceBusiness traces a business operation with OpenTelemetry
func (ot *OpenTelemetryTracer) TraceBusiness(ctx context.Context, operation string, userID string, fn func(context.Context) error) error {
	ctx, span := ot.StartSpan(ctx, fmt.Sprintf("Business %s", operation),
		trace.WithAttributes(
			attribute.String("business.operation", operation),
			attribute.String("user.id", userID),
			attribute.String("component", "business"),
		),
	)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error", err.Error()))
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// TraceFunction traces a generic function with OpenTelemetry
func (ot *OpenTelemetryTracer) TraceFunction(ctx context.Context, functionName string, fn func(context.Context) error) error {
	ctx, span := ot.StartSpan(ctx, functionName,
		trace.WithAttributes(
			attribute.String("function.name", functionName),
		),
	)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error", err.Error()))
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// AddSpanAttributes adds attributes to the current span
func (ot *OpenTelemetryTracer) AddSpanAttributes(ctx context.Context, attrs map[string]interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			span.SetAttributes(attribute.String(k, val))
		case int:
			span.SetAttributes(attribute.Int(k, val))
		case int64:
			span.SetAttributes(attribute.Int64(k, val))
		case float64:
			span.SetAttributes(attribute.Float64(k, val))
		case bool:
			span.SetAttributes(attribute.Bool(k, val))
		default:
			span.SetAttributes(attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}
}

// RecordSpanEvent records an event on the current span
func (ot *OpenTelemetryTracer) RecordSpanEvent(ctx context.Context, eventName string, attrs map[string]interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	var otelAttrs []attribute.KeyValue
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			otelAttrs = append(otelAttrs, attribute.String(k, val))
		case int:
			otelAttrs = append(otelAttrs, attribute.Int(k, val))
		case int64:
			otelAttrs = append(otelAttrs, attribute.Int64(k, val))
		case float64:
			otelAttrs = append(otelAttrs, attribute.Float64(k, val))
		case bool:
			otelAttrs = append(otelAttrs, attribute.Bool(k, val))
		default:
			otelAttrs = append(otelAttrs, attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}

	span.AddEvent(eventName, trace.WithAttributes(otelAttrs...))
}

// SetSpanStatus sets the status on the current span
func (ot *OpenTelemetryTracer) SetSpanStatus(ctx context.Context, code codes.Code, message string) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	span.SetStatus(code, message)
}

// GetTraceID returns the trace ID from the context
func (ot *OpenTelemetryTracer) GetTraceID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return ""
	}
	return spanCtx.TraceID().String()
}

// GetSpanID returns the span ID from the context
func (ot *OpenTelemetryTracer) GetSpanID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return ""
	}
	return spanCtx.SpanID().String()
}

// Shutdown gracefully shuts down the OpenTelemetry tracer
func (ot *OpenTelemetryTracer) Shutdown(ctx context.Context) error {
	tp := otel.GetTracerProvider()
	if sdkProvider, ok := tp.(*sdktrace.TracerProvider); ok {
		return sdkProvider.Shutdown(ctx)
	}
	return nil
}

// Global OpenTelemetry tracer instance
var GlobalOpenTelemetryTracer *OpenTelemetryTracer

// InitializeOpenTelemetry initializes the global OpenTelemetry tracer
func InitializeOpenTelemetry(config OpenTelemetryConfig) error {
	tracer, err := NewOpenTelemetryTracer(config)
	if err != nil {
		return fmt.Errorf("failed to initialize OpenTelemetry tracer: %w", err)
	}

	GlobalOpenTelemetryTracer = tracer
	return nil
}

// GetOpenTelemetryTracer returns the global OpenTelemetry tracer
func GetOpenTelemetryTracer() *OpenTelemetryTracer {
	return GlobalOpenTelemetryTracer
}

// HybridTracer combines both custom tracing and OpenTelemetry
type HybridTracer struct {
	customTracer        *Tracer
	openTelemetryTracer *OpenTelemetryTracer
}

// NewHybridTracer creates a new hybrid tracer
func NewHybridTracer(customTracer *Tracer, otTracer *OpenTelemetryTracer) *HybridTracer {
	return &HybridTracer{
		customTracer:        customTracer,
		openTelemetryTracer: otTracer,
	}
}

// TraceHTTP traces an HTTP request with both tracers
func (ht *HybridTracer) TraceHTTP(ctx context.Context, method, url string, fn func(context.Context) error) error {
	// Start custom span
	_, customSpan := ht.customTracer.StartSpan(ctx, "http.request",
		WithResource("http"),
		WithTags(map[string]interface{}{
			"http.method": method,
			"http.url":    url,
			"component":   "http",
		}),
	)
	defer ht.customTracer.FinishSpan(customSpan)

	// Start OpenTelemetry span
	_, otelSpan := ht.openTelemetryTracer.StartSpan(ctx, fmt.Sprintf("HTTP %s", method),
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.url", url),
			attribute.String("component", "http"),
		),
	)
	defer otelSpan.End()

	// Execute function
	err := fn(ctx)
	if err != nil {
		ht.customTracer.SetError(customSpan, err)
		otelSpan.RecordError(err)
		otelSpan.SetStatus(codes.Error, err.Error())
	}

	return err
}

// TraceDatabase traces a database operation with both tracers
func (ht *HybridTracer) TraceDatabase(ctx context.Context, operation, table string, fn func(context.Context) error) error {
	// Start custom span
	_, customSpan := ht.customTracer.StartSpan(ctx, "db."+operation,
		WithResource("database"),
		WithTags(map[string]interface{}{
			"db.operation": operation,
			"db.table":     table,
			"component":    "database",
		}),
	)
	defer ht.customTracer.FinishSpan(customSpan)

	// Start OpenTelemetry span
	_, otelSpan := ht.openTelemetryTracer.StartSpan(ctx, fmt.Sprintf("DB %s", operation),
		trace.WithAttributes(
			attribute.String("db.operation", operation),
			attribute.String("db.table", table),
			attribute.String("db.system", "postgresql"),
			attribute.String("component", "database"),
		),
	)
	defer otelSpan.End()

	// Execute function
	err := fn(ctx)
	if err != nil {
		ht.customTracer.SetError(customSpan, err)
		otelSpan.RecordError(err)
		otelSpan.SetStatus(codes.Error, err.Error())
	}

	return err
}

// Global hybrid tracer instance
var GlobalHybridTracer *HybridTracer

// InitializeHybridTracer initializes the global hybrid tracer
func InitializeHybridTracer() {
	if GlobalTracer != nil && GlobalOpenTelemetryTracer != nil {
		GlobalHybridTracer = NewHybridTracer(GlobalTracer, GlobalOpenTelemetryTracer)
	}
}

// GetHybridTracer returns the global hybrid tracer
func GetHybridTracer() *HybridTracer {
	return GlobalHybridTracer
}