package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Span represents a distributed tracing span
type Span struct {
	TraceID      string                 `json:"trace_id"`
	SpanID       string                 `json:"span_id"`
	ParentSpanID string                 `json:"parent_span_id,omitempty"`
	OperationName string                `json:"operation_name"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Duration     time.Duration          `json:"duration"`
	Tags         map[string]interface{} `json:"tags,omitempty"`
	Logs         []SpanLog              `json:"logs,omitempty"`
	Status       SpanStatus             `json:"status"`
	ServiceName  string                 `json:"service_name"`
	Resource     string                 `json:"resource,omitempty"`
}

// SpanLog represents a log entry within a span
type SpanLog struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// SpanStatus represents the status of a span
type SpanStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

// Tracer manages distributed tracing
type Tracer struct {
	serviceName string
	spans       map[string]*Span
	spansMutex  sync.RWMutex
	sampler     Sampler
	exporters   []SpanExporter
}

// Sampler determines whether to sample a trace
type Sampler interface {
	ShouldSample(traceID string) bool
}

// SpanExporter exports spans to external systems
type SpanExporter interface {
	Export(ctx context.Context, spans []*Span) error
	Shutdown(ctx context.Context) error
}

// ProbabilitySampler samples traces based on probability
type ProbabilitySampler struct {
	probability float64
}

// NewProbabilitySampler creates a new probability sampler
func NewProbabilitySampler(probability float64) *ProbabilitySampler {
	return &ProbabilitySampler{probability: probability}
}

// ShouldSample determines if a trace should be sampled
func (ps *ProbabilitySampler) ShouldSample(traceID string) bool {
	// Simple hash-based sampling for deterministic behavior
	hash := 0
	for _, char := range traceID {
		hash = hash*31 + int(char)
	}
	return float64(hash%10000)/10000 < ps.probability
}

// ConsoleSpanExporter exports spans to console
type ConsoleSpanExporter struct{}

// NewConsoleSpanExporter creates a new console span exporter
func NewConsoleSpanExporter() *ConsoleSpanExporter {
	return &ConsoleSpanExporter{}
}

// Export exports spans to console
func (cse *ConsoleSpanExporter) Export(ctx context.Context, spans []*Span) error {
	for _, span := range spans {
		fmt.Printf("[TRACE] %s [%s] %s - %s (%.2fms)\n",
			span.TraceID,
			span.ServiceName,
			span.OperationName,
			span.Status.Message,
			float64(span.Duration.Nanoseconds())/1000000)
	}
	return nil
}

// Shutdown closes the console exporter
func (cse *ConsoleSpanExporter) Shutdown(ctx context.Context) error {
	return nil
}

// NewTracer creates a new tracer
func NewTracer(serviceName string, sampler Sampler) *Tracer {
	return &Tracer{
		serviceName: serviceName,
		spans:       make(map[string]*Span),
		sampler:     sampler,
		exporters:   []SpanExporter{NewConsoleSpanExporter()},
	}
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(ctx context.Context, operationName string, opts ...SpanOption) (context.Context, *Span) {
	span := &Span{
		SpanID:        GenerateSpanID(),
		OperationName: operationName,
		StartTime:     time.Now().UTC(),
		Tags:          make(map[string]interface{}),
		Logs:          make([]SpanLog, 0),
		Status:        SpanStatus{Code: 0},
		ServiceName:   t.serviceName,
	}

	// Extract trace context from parent context
	parentTraceID := getTraceIDFromContext(ctx)
	parentSpanID := getSpanIDFromContext(ctx)

	if parentTraceID != "" {
		span.TraceID = parentTraceID
		span.ParentSpanID = parentSpanID
	} else {
		span.TraceID = GenerateTraceID()
	}

	// Apply span options
	for _, opt := range opts {
		opt(span)
	}

	// Store span
	t.spansMutex.Lock()
	t.spans[span.SpanID] = span
	t.spansMutex.Unlock()

	// Add span to context
	ctx = WithTraceID(ctx, span.TraceID)
	ctx = WithSpanID(ctx, span.SpanID)
	ctx = WithCorrelationID(ctx, span.TraceID) // Use trace ID as correlation ID

	return ctx, span
}

// FinishSpan finishes a span
func (t *Tracer) FinishSpan(span *Span, opts ...SpanFinishOption) {
	span.EndTime = time.Now().UTC()
	span.Duration = span.EndTime.Sub(span.StartTime)

	// Apply finish options
	for _, opt := range opts {
		opt(span)
	}

	// Export span asynchronously
	go func() {
		ctx := context.Background()
		for _, exporter := range t.exporters {
			if err := exporter.Export(ctx, []*Span{span}); err != nil {
				// Log error but don't fail the trace
				fmt.Printf("Failed to export span: %v\n", err)
			}
		}
	}()

	// Remove from active spans
	t.spansMutex.Lock()
	delete(t.spans, span.SpanID)
	t.spansMutex.Unlock()
}

// SetTag sets a tag on a span
func (t *Tracer) SetTag(span *Span, key string, value interface{}) {
	if span.Tags == nil {
		span.Tags = make(map[string]interface{})
	}
	span.Tags[key] = value
}

// SetError sets error on a span
func (t *Tracer) SetError(span *Span, err error) {
	if err != nil {
		span.Status.Code = 1
		span.Status.Message = err.Error()
		t.SetTag(span, "error", true)
		t.SetTag(span, "error.message", err.Error())
		t.SetTag(span, "error.type", fmt.Sprintf("%T", err))
	}
}

// LogEvent logs an event to a span
func (t *Tracer) LogEvent(span *Span, level, message string, fields map[string]interface{}) {
	logEntry := SpanLog{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}
	span.Logs = append(span.Logs, logEntry)
}

// AddExporter adds a span exporter
func (t *Tracer) AddExporter(exporter SpanExporter) {
	t.exporters = append(t.exporters, exporter)
}

// GetActiveSpans returns the number of active spans
func (t *Tracer) GetActiveSpans() int {
	t.spansMutex.RLock()
	defer t.spansMutex.RUnlock()
	return len(t.spans)
}

// Shutdown shuts down the tracer
func (t *Tracer) Shutdown(ctx context.Context) error {
	var errors []error

	for _, exporter := range t.exporters {
		if err := exporter.Shutdown(ctx); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}

// SpanOption configures a span
type SpanOption func(*Span)

// WithResource sets the resource for a span
func WithResource(resource string) SpanOption {
	return func(s *Span) {
		s.Resource = resource
	}
}

// WithTags adds tags to a span
func WithTags(tags map[string]interface{}) SpanOption {
	return func(s *Span) {
		for k, v := range tags {
			s.Tags[k] = v
		}
	}
}

// WithParent sets the parent span ID
func WithParent(parentSpanID string) SpanOption {
	return func(s *Span) {
		s.ParentSpanID = parentSpanID
	}
}

// SpanFinishOption configures span finishing
type SpanFinishOption func(*Span)

// WithError finishes a span with an error
func WithError(err error) SpanFinishOption {
	return func(s *Span) {
		if err != nil {
			s.Status.Code = 1
			s.Status.Message = err.Error()
			s.Tags["error"] = true
			s.Tags["error.message"] = err.Error()
		}
	}
}

// WithStatus finishes a span with a specific status
func WithStatus(code int, message string) SpanFinishOption {
	return func(s *Span) {
		s.Status.Code = code
		s.Status.Message = message
	}
}

// TraceFunction traces a function execution
func (t *Tracer) TraceFunction(ctx context.Context, functionName string, fn func(context.Context) error) error {
	ctx, span := t.StartSpan(ctx, functionName)
	defer t.FinishSpan(span)

	err := fn(ctx)
	if err != nil {
		t.SetError(span, err)
	}

	return err
}

// TraceHTTP traces an HTTP request
func (t *Tracer) TraceHTTP(ctx context.Context, method, url string, fn func(context.Context) error) error {
	ctx, span := t.StartSpan(ctx, "http.request",
		WithResource("http"),
		WithTags(map[string]interface{}{
			"http.method": method,
			"http.url":    url,
			"component":   "http",
		}),
	)
	defer t.FinishSpan(span)

	err := fn(ctx)
	if err != nil {
		t.SetError(span, err)
	}

	return err
}

// TraceDatabase traces a database operation
func (t *Tracer) TraceDatabase(ctx context.Context, operation, table string, fn func(context.Context) error) error {
	ctx, span := t.StartSpan(ctx, "db."+operation,
		WithResource("database"),
		WithTags(map[string]interface{}{
			"db.operation": operation,
			"db.table":     table,
			"component":    "database",
		}),
	)
	defer t.FinishSpan(span)

	err := fn(ctx)
	if err != nil {
		t.SetError(span, err)
	}

	return err
}

// TraceCache traces a cache operation
func (t *Tracer) TraceCache(ctx context.Context, operation string, fn func(context.Context) error) error {
	ctx, span := t.StartSpan(ctx, "cache."+operation,
		WithResource("cache"),
		WithTags(map[string]interface{}{
			"cache.operation": operation,
			"component":       "cache",
		}),
	)
	defer t.FinishSpan(span)

	err := fn(ctx)
	if err != nil {
		t.SetError(span, err)
	}

	return err
}

// TraceBusiness traces a business operation
func (t *Tracer) TraceBusiness(ctx context.Context, operation string, userID string, fn func(context.Context) error) error {
	ctx, span := t.StartSpan(ctx, "business."+operation,
		WithResource("business"),
		WithTags(map[string]interface{}{
			"business.operation": operation,
			"user.id":            userID,
			"component":          "business",
		}),
	)
	defer t.FinishSpan(span)

	err := fn(ctx)
	if err != nil {
		t.SetError(span, err)
	}

	return err
}

// Global tracer instance
var GlobalTracer *Tracer

// InitializeTracer initializes the global tracer
func InitializeTracer(serviceName string, sampleRate float64) {
	sampler := NewProbabilitySampler(sampleRate)
	GlobalTracer = NewTracer(serviceName, sampler)
}

// GetTracer returns the global tracer
func GetTracer() *Tracer {
	if GlobalTracer == nil {
		GlobalTracer = NewTracer("unknown", NewProbabilitySampler(1.0))
	}
	return GlobalTracer
}

// Convenience functions for global tracer

// StartSpan starts a new span using the global tracer
func StartSpan(ctx context.Context, operationName string, opts ...SpanOption) (context.Context, *Span) {
	return GetTracer().StartSpan(ctx, operationName, opts...)
}

// FinishSpan finishes a span using the global tracer
func FinishSpan(span *Span, opts ...SpanFinishOption) {
	GetTracer().FinishSpan(span, opts...)
}

// TraceFunction traces a function using the global tracer
func TraceFunction(ctx context.Context, functionName string, fn func(context.Context) error) error {
	return GetTracer().TraceFunction(ctx, functionName, fn)
}

// TraceHTTP traces an HTTP request using the global tracer
func TraceHTTP(ctx context.Context, method, url string, fn func(context.Context) error) error {
	return GetTracer().TraceHTTP(ctx, method, url, fn)
}

// TraceDatabase traces a database operation using the global tracer
func TraceDatabase(ctx context.Context, operation, table string, fn func(context.Context) error) error {
	return GetTracer().TraceDatabase(ctx, operation, table, fn)
}

// TraceCache traces a cache operation using the global tracer
func TraceCache(ctx context.Context, operation string, fn func(context.Context) error) error {
	return GetTracer().TraceCache(ctx, operation, fn)
}

// TraceBusiness traces a business operation using the global tracer
func TraceBusiness(ctx context.Context, operation string, userID string, fn func(context.Context) error) error {
	return GetTracer().TraceBusiness(ctx, operation, userID, fn)
}