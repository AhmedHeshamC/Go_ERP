package tracing

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SpanKind represents the type of span
type SpanKind string

const (
	SpanKindServer   SpanKind = "server"
	SpanKindClient   SpanKind = "client"
	SpanKindProducer SpanKind = "producer"
	SpanKindConsumer SpanKind = "consumer"
	SpanKindInternal SpanKind = "internal"
)

// SpanStatus represents the status of a span
type SpanStatus struct {
	Code    SpanStatusCode `json:"code"`
	Message string          `json:"message,omitempty"`
	Error   error           `json:"error,omitempty"`
}

// SpanStatusCode represents span status codes
type SpanStatusCode int

const (
	SpanStatusOK    SpanStatusCode = 1
	SpanStatusError SpanStatusCode = 2
)

// Span represents a single span in a trace
type Span struct {
	// Core span information
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	ParentSpanID *string          `json:"parent_span_id,omitempty"`
	OperationName string         `json:"operation_name"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Duration    time.Duration     `json:"duration"`
	Kind        SpanKind          `json:"kind"`
	Status      SpanStatus        `json:"status"`

	// Attributes and events
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Events     []SpanEvent           `json:"events,omitempty"`
	Links      []SpanLink            `json:"links,omitempty"`

	// Resource and service information
	ServiceName    string            `json:"service_name"`
	Resource       map[string]string `json:"resource,omitempty"`

	// Sampling information
	Sampled    bool  `json:"sampled"`
	SampleRate float64 `json:"sample_rate,omitempty"`

	// Context
	Context context.Context `json:"-"`
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string                 `json:"name"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// SpanLink represents a link to another span
type SpanLink struct {
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// SpanContext contains the trace context for propagation
type SpanContext struct {
	TraceID      string  `json:"trace_id"`
	SpanID       string  `json:"span_id"`
	Sampled      bool    `json:"sampled"`
	IsRemote     bool    `json:"is_remote"`
	BaggageItems map[string]string `json:"baggage_items,omitempty"`
}

// Config holds the tracer configuration
type Config struct {
	// Service configuration
	ServiceName        string  `json:"service_name"`
	ServiceVersion     string  `json:"service_version"`
	Environment        string  `json:"environment"`

	// Sampling configuration
	SampleRate         float64 `json:"sample_rate"`
	MaxSpansPerTrace   int     `json:"max_spans_per_trace"`
	ForceSampling      bool    `json:"force_sampling"`

	// Performance configuration
	MaxAttributesPerSpan int    `json:"max_attributes_per_span"`
	MaxEventsPerSpan     int    `json:"max_events_per_span"`
	MaxLinksPerSpan      int    `json:"max_links_per_span"`
	FlushInterval        time.Duration `json:"flush_interval"`

	// Propagation configuration
	PropagationFormat   string `json:"propagation_format"` // "tracecontext", "b3", "jaeger", "w3c"
	EnableBaggage       bool   `json:"enable_baggage"`

	// Export configuration
	Exporter           Exporter `json:"-"`
	ExportTimeout      time.Duration `json:"export_timeout"`
	BatchSize         int           `json:"batch_size"`
	AsyncExport       bool          `json:"async_export"`

	// Logging configuration
	Logger             *zerolog.Logger `json:"-"`
	LogSpans           bool          `json:"log_spans"`
	LogSpanEvents      bool          `json:"log_span_events"`

	// Security configuration
	SanitizeAttributes  []string `json:"sanitize_attributes"`
	EnableSanitization  bool     `json:"enable_sanitization"`

	// Debug configuration
	DebugMode           bool `json:"debug_mode"`
	EnableSpanPrinting bool `json:"enable_span_printing"`
}

// DefaultConfig returns a default tracer configuration
func DefaultConfig() *Config {
	return &Config{
		ServiceName:        "erp-go-service",
		ServiceVersion:     "1.0.0",
		Environment:        "development",
		SampleRate:         1.0, // Sample all traces in development
		MaxSpansPerTrace:   1000,
		ForceSampling:      false,
		MaxAttributesPerSpan: 100,
		MaxEventsPerSpan:    100,
		MaxLinksPerSpan:     10,
		FlushInterval:      5 * time.Second,
		PropagationFormat:  "w3c",
		EnableBaggage:      true,
		ExportTimeout:      10 * time.Second,
		BatchSize:          100,
		AsyncExport:        true,
		LogSpans:           true,
		LogSpanEvents:      true,
		SanitizeAttributes: []string{
			"password", "token", "secret", "key", "auth",
			"credit_card", "ssn", "social_security",
		},
		EnableSanitization:  true,
		DebugMode:           true,
		EnableSpanPrinting:  true,
	}
}

// ProductionConfig returns a production-safe configuration
func ProductionConfig() *Config {
	config := DefaultConfig()
	config.SampleRate = 0.1 // Sample 10% of traces
	config.DebugMode = false
	config.EnableSpanPrinting = false
	config.LogSpans = false
	config.LogSpanEvents = false
	config.EnableSanitization = true
	return config
}

// Exporter defines the interface for trace exporters
type Exporter interface {
	ExportSpans(spans []*Span) error
	Shutdown(ctx context.Context) error
}

// Tracer represents the distributed tracer
type Tracer struct {
	config      *Config
	logger      *zerolog.Logger
	spans       map[string]*Span
	spanLinks   map[string][]*SpanLink
	spansMu     sync.RWMutex
	exporter    Exporter
	sampler     *sampler
	sanitizer   *sanitizer
	flushTicker *time.Ticker
	shutdownCh  chan struct{}
}

// sampler handles trace sampling
type sampler struct {
	rate       float64
	force      bool
}

// sanitizer handles attribute sanitization
type sanitizer struct {
	fields map[string]bool
}

// NewTracer creates a new tracer
func NewTracer(config *Config, logger *zerolog.Logger) (*Tracer, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if logger == nil {
		nopLogger := zerolog.Nop()
		logger = &nopLogger
	}

	tracer := &Tracer{
		config:    config,
		logger:    logger,
		spans:     make(map[string]*Span),
		spanLinks: make(map[string][]*SpanLink),
		exporter:  config.Exporter,
		sampler:   newSampler(config.SampleRate, config.ForceSampling),
		sanitizer: newSanitizer(config.SanitizeAttributes),
		shutdownCh: make(chan struct{}),
	}

	// Start background flush routine
	if config.AsyncExport {
		go tracer.backgroundFlush()
		tracer.flushTicker = time.NewTicker(config.FlushInterval)
	}

	return tracer, nil
}

// newSampler creates a new sampler
func newSampler(rate float64, force bool) *sampler {
	return &sampler{
		rate:  rate,
		force: force,
	}
}

// newSanitizer creates a new sanitizer
func newSanitizer(fields []string) *sanitizer {
	fieldMap := make(map[string]bool)
	for _, field := range fields {
		fieldMap[strings.ToLower(field)] = true
	}
	return &sanitizer{fields: fieldMap}
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(ctx context.Context, operationName string, kind SpanKind) (context.Context, *Span) {
	// Get or create trace context
	traceContext, parentSpanID := t.extractTraceContext(ctx)

	// Generate new span ID
	spanID := generateSpanID()

	// Determine sampling
	sampled := t.shouldSample(traceContext)

	// Create span
	span := &Span{
		TraceID:       traceContext.TraceID,
		SpanID:        spanID,
		ParentSpanID:  parentSpanID,
		OperationName: operationName,
		StartTime:     time.Now(),
		Kind:          kind,
		Status: SpanStatus{
			Code: SpanStatusOK,
		},
		Attributes:    make(map[string]interface{}),
		Events:        make([]SpanEvent, 0),
		Links:         make([]SpanLink, 0),
		ServiceName:   t.config.ServiceName,
		Resource:      t.buildResource(),
		Sampled:       sampled,
		SampleRate:    t.config.SampleRate,
		Context:       ctx,
	}

	// Add default attributes
	t.addDefaultAttributes(span)

	// Sanitize attributes if configured
	if t.config.EnableSanitization {
		t.sanitizeAttributes(span)
	}

	// Store span
	if sampled {
		t.spansMu.Lock()
		t.spans[span.SpanID] = span
		t.spansMu.Unlock()
	}

	// Create new context with span
	newTraceContext := &SpanContext{
		TraceID:      span.TraceID,
		SpanID:       span.SpanID,
		Sampled:      sampled,
		IsRemote:     false,
		BaggageItems: traceContext.BaggageItems,
	}

	newCtx := t.setSpanContext(ctx, newTraceContext)

	// Log span start
	if t.config.LogSpans && sampled {
		t.logger.Debug().
			Str("trace_id", span.TraceID).
			Str("span_id", span.SpanID).
			Str("parent_span_id", pointerToString(parentSpanID)).
			Str("operation", operationName).
			Str("kind", string(kind)).
			Msg("Span started")
	}

	return newCtx, span
}

// FinishSpan finishes a span
func (t *Tracer) FinishSpan(span *Span) {
	if span == nil {
		return
	}

	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)

	// Update status if not set
	if span.Status.Code == 0 {
		span.Status.Code = SpanStatusOK
	}

	// Export span if sampled
	if span.Sampled {
		if t.config.AsyncExport {
			// Span will be exported in background flush
		} else {
			t.exportSpan(span)
		}
	}

	// Remove from active spans
	t.spansMu.Lock()
	delete(t.spans, span.SpanID)
	t.spansMu.Unlock()

	// Log span completion
	if t.config.LogSpans && span.Sampled {
		t.logger.Debug().
			Str("trace_id", span.TraceID).
			Str("span_id", span.SpanID).
			Str("operation", span.OperationName).
			Dur("duration", span.Duration).
			Int("status_code", int(span.Status.Code)).
			Msg("Span finished")
	}
}

// AddEvent adds an event to a span
func (t *Tracer) AddEvent(span *Span, name string, attributes map[string]interface{}) {
	if span == nil || len(span.Events) >= t.config.MaxEventsPerSpan {
		return
	}

	event := SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attributes,
	}

	// Sanitize event attributes
	if t.config.EnableSanitization {
		if sanitizedData, ok := t.sanitizer.sanitizeData(attributes).(map[string]interface{}); ok {
			event.Attributes = sanitizedData
		}
	}

	span.Events = append(span.Events, event)

	// Log event
	if t.config.LogSpanEvents && span.Sampled {
		t.logger.Debug().
			Str("trace_id", span.TraceID).
			Str("span_id", span.SpanID).
			Str("event", name).
			Msg("Span event added")
	}
}

// SetAttribute sets an attribute on a span
func (t *Tracer) SetAttribute(span *Span, key string, value interface{}) {
	if span == nil || len(span.Attributes) >= t.config.MaxAttributesPerSpan {
		return
	}

	span.Attributes[key] = value

	// Sanitize attribute if configured
	if t.config.EnableSanitization {
		t.sanitizeAttributes(span)
	}
}

// SetStatus sets the status of a span
func (t *Tracer) SetStatus(span *Span, code SpanStatusCode, message string) {
	if span == nil {
		return
	}

	span.Status = SpanStatus{
		Code:    code,
		Message: message,
	}
}

// SetError sets an error on a span
func (t *Tracer) SetError(span *Span, err error) {
	if span == nil || err == nil {
		return
	}

	span.Status = SpanStatus{
		Code:    SpanStatusError,
		Message: err.Error(),
		Error:   err,
	}

	// Add error event
	t.AddEvent(span, "error", map[string]interface{}{
		"error.type":    fmt.Sprintf("%T", err),
		"error.message": err.Error(),
	})
}

// AddLink adds a link to another span
func (t *Tracer) AddLink(span *Span, traceID, spanID string, attributes map[string]interface{}) {
	if span == nil || len(span.Links) >= t.config.MaxLinksPerSpan {
		return
	}

	link := SpanLink{
		TraceID:    traceID,
		SpanID:     spanID,
		Attributes: attributes,
	}

	span.Links = append(span.Links, link)
}

// GetActiveSpan returns the active span from context
func (t *Tracer) GetActiveSpan(ctx context.Context) *Span {
	traceContext := t.getSpanContext(ctx)
	if traceContext == nil {
		return nil
	}

	t.spansMu.RLock()
	defer t.spansMu.RUnlock()

	return t.spans[traceContext.SpanID]
}

// GetTraceContext returns the trace context from context
func (t *Tracer) GetTraceContext(ctx context.Context) *SpanContext {
	return t.getSpanContext(ctx)
}

// Shutdown shuts down the tracer
func (t *Tracer) Shutdown(ctx context.Context) error {
	close(t.shutdownCh)

	if t.flushTicker != nil {
		t.flushTicker.Stop()
	}

	// Export remaining spans
	t.flushSpans()

	// Shutdown exporter
	if t.exporter != nil {
		return t.exporter.Shutdown(ctx)
	}

	return nil
}

// extractTraceContext extracts trace context from incoming request
func (t *Tracer) extractTraceContext(ctx context.Context) (*SpanContext, *string) {
	// Try to extract from existing span context
	if traceContext := t.getSpanContext(ctx); traceContext != nil {
		return traceContext, &traceContext.SpanID
	}

	// Try to extract from HTTP headers if this is an HTTP context
	if httpCtx, ok := ctx.Value(HTTPRequestContextKey).(HTTPRequestContext); ok {
		return t.extractTraceContextFromHeaders(httpCtx.Headers), nil
	}

	// Create new trace context
	return &SpanContext{
		TraceID: generateTraceID(),
		SpanID:  "",
		Sampled: t.shouldSample(nil),
	}, nil
}

// extractTraceContextFromHeaders extracts trace context from HTTP headers
func (t *Tracer) extractTraceContextFromHeaders(headers http.Header) *SpanContext {
	traceContext := &SpanContext{
		BaggageItems: make(map[string]string),
	}

	// Extract based on propagation format
	switch t.config.PropagationFormat {
	case "w3c", "tracecontext":
		t.extractW3CTraceContext(headers, traceContext)
	case "b3":
		t.extractB3TraceContext(headers, traceContext)
	default:
		// Default to W3C trace context
		t.extractW3CTraceContext(headers, traceContext)
	}

	// Extract baggage if enabled
	if t.config.EnableBaggage {
		t.extractBaggage(headers, traceContext)
	}

	return traceContext
}

// extractW3CTraceContext extracts W3C trace context from headers
func (t *Tracer) extractW3CTraceContext(headers http.Header, traceContext *SpanContext) {
	// Extract traceparent header
	if traceparent := headers.Get("traceparent"); traceparent != "" {
		parts := strings.Split(traceparent, "-")
		if len(parts) >= 4 {
			traceContext.TraceID = parts[1]
			traceContext.SpanID = parts[2]
			traceContext.Sampled = parts[3] == "01"
		}
	}

	// Extract tracestate header
	if tracestate := headers.Get("tracestate"); tracestate != "" {
		// Parse tracestate for additional information
		// This is a simplified implementation
	}
}

// extractB3TraceContext extracts B3 trace context from headers
func (t *Tracer) extractB3TraceContext(headers http.Header, traceContext *SpanContext) {
	// Extract B3 single header
	if b3 := headers.Get("b3"); b3 != "" {
		parts := strings.Split(b3, "-")
		if len(parts) >= 2 {
			traceContext.TraceID = parts[0]
			traceContext.SpanID = parts[1]
			if len(parts) >= 3 {
				traceContext.Sampled = parts[2] == "1"
			}
		}
		return
	}

	// Extract B3 multiple headers
	if traceID := headers.Get("x-b3-traceid"); traceID != "" {
		traceContext.TraceID = traceID
	}
	if spanID := headers.Get("x-b3-spanid"); spanID != "" {
		traceContext.SpanID = spanID
	}
	if sampled := headers.Get("x-b3-sampled"); sampled != "" {
		traceContext.Sampled = sampled == "1"
	}
}

// extractBaggage extracts baggage items from headers
func (t *Tracer) extractBaggage(headers http.Header, traceContext *SpanContext) {
	if baggage := headers.Get("baggage"); baggage != "" {
		items := strings.Split(baggage, ",")
		for _, item := range items {
			parts := strings.SplitN(item, "=", 2)
			if len(parts) == 2 {
				traceContext.BaggageItems[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
}

// shouldSample determines if a trace should be sampled
func (t *Tracer) shouldSample(traceContext *SpanContext) bool {
	if t.config.ForceSampling {
		return true
	}

	if traceContext != nil && traceContext.IsRemote {
		// Respect sampling decision from upstream
		return traceContext.Sampled
	}

	// Apply local sampling
	return rand.Float64() <= t.config.SampleRate
}

// addDefaultAttributes adds default attributes to a span
func (t *Tracer) addDefaultAttributes(span *Span) {
	// Service information
	span.Attributes["service.name"] = t.config.ServiceName
	span.Attributes["service.version"] = t.config.ServiceVersion
	span.Attributes["service.environment"] = t.config.Environment

	// Runtime information
	span.Attributes["runtime.name"] = "go"
	span.Attributes["runtime.version"] = runtime.Version()
	span.Attributes["runtime.os"] = runtime.GOOS
	span.Attributes["runtime.arch"] = runtime.GOARCH

	// Process information
	span.Attributes["process.pid"] = os.Getpid()
	if hostname, err := os.Hostname(); err == nil {
		span.Attributes["process.hostname"] = hostname
	}
}

// buildResource builds resource information
func (t *Tracer) buildResource() map[string]string {
	resource := map[string]string{
		"service.name":     t.config.ServiceName,
		"service.version":  t.config.ServiceVersion,
		"service.environment": t.config.Environment,
		"telemetry.sdk.name": "erp-go-tracing",
		"telemetry.sdk.version": "1.0.0",
		"telemetry.sdk.language": "go",
	}

	return resource
}

// sanitizeAttributes sanitizes span attributes
func (t *Tracer) sanitizeAttributes(span *Span) {
	if span.Attributes == nil {
		return
	}

	if sanitizedData, ok := t.sanitizer.sanitizeData(span.Attributes).(map[string]interface{}); ok {
		span.Attributes = sanitizedData
	}
}

// sanitizeData recursively sanitizes sensitive data
func (s *sanitizer) sanitizeData(data interface{}) interface{} {
	switch v := data.(type) {
	case string:
		return v
	case map[string]interface{}:
		for key, value := range v {
			if s.shouldSanitize(key) {
				v[key] = "[REDACTED]"
			} else {
				v[key] = s.sanitizeData(value)
			}
		}
		return v
	case []interface{}:
		for i, item := range v {
			v[i] = s.sanitizeData(item)
		}
		return v
	default:
		return v
	}
}

// shouldSanitize checks if a field should be sanitized
func (s *sanitizer) shouldSanitize(field string) bool {
	lowerField := strings.ToLower(field)
	for sensitiveField := range s.fields {
		if strings.Contains(lowerField, sensitiveField) {
			return true
		}
	}
	return false
}

// setSpanContext sets the span context in the Go context
func (t *Tracer) setSpanContext(ctx context.Context, spanContext *SpanContext) context.Context {
	return context.WithValue(ctx, SpanContextKey, spanContext)
}

// getSpanContext gets the span context from the Go context
func (t *Tracer) getSpanContext(ctx context.Context) *SpanContext {
	if spanContext := ctx.Value(SpanContextKey); spanContext != nil {
		if context, ok := spanContext.(*SpanContext); ok {
			return context
		}
	}
	return nil
}

// exportSpan exports a single span
func (t *Tracer) exportSpan(span *Span) {
	if t.exporter != nil {
		err := t.exporter.ExportSpans([]*Span{span})
		if err != nil {
			t.logger.Error().Err(err).
				Str("trace_id", span.TraceID).
				Str("span_id", span.SpanID).
				Msg("Failed to export span")
		}
	}
}

// flushSpans exports all active spans
func (t *Tracer) flushSpans() {
	if t.exporter == nil {
		return
	}

	t.spansMu.RLock()
	spans := make([]*Span, 0, len(t.spans))
	for _, span := range t.spans {
		spans = append(spans, span)
	}
	t.spansMu.RUnlock()

	if len(spans) > 0 {
		err := t.exporter.ExportSpans(spans)
		if err != nil {
			t.logger.Error().Err(err).Msg("Failed to flush spans")
		}
	}
}

// backgroundFlush periodically flushes spans
func (t *Tracer) backgroundFlush() {
	for {
		select {
		case <-t.flushTicker.C:
			t.flushSpans()
		case <-t.shutdownCh:
			return
		}
	}
}

// Context keys
type contextKey string

const (
	SpanContextKey        contextKey = "span_context"
	HTTPRequestContextKey contextKey = "http_request_context"
)

// HTTPRequestContext represents HTTP request context for tracing
type HTTPRequestContext struct {
	Headers http.Header
}

// GetStats returns tracer statistics
func (t *Tracer) GetStats() map[string]interface{} {
	t.spansMu.RLock()
	defer t.spansMu.RUnlock()

	stats := map[string]interface{}{
		"active_spans":    len(t.spans),
		"service_name":    t.config.ServiceName,
		"service_version": t.config.ServiceVersion,
		"environment":     t.config.Environment,
		"sample_rate":     t.config.SampleRate,
		"initialized":     true,
	}

	// Add span links count
	stats["span_links"] = len(t.spanLinks)

	// Add sampling statistics if available
	if t.sampler != nil {
		stats["sampling_enabled"] = true
	} else {
		stats["sampling_enabled"] = false
	}

	// Add sanitizer statistics if available
	if t.sanitizer != nil {
		stats["sanitization_enabled"] = t.config.EnableSanitization
	} else {
		stats["sanitization_enabled"] = false
	}

	return stats
}

// Helper functions

// generateTraceID generates a new trace ID
func generateTraceID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

// generateSpanID generates a new span ID
func generateSpanID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

// pointerToString safely converts a string pointer to string
func pointerToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Global tracer instance
var globalTracer *Tracer

// InitGlobalTracer initializes the global tracer
func InitGlobalTracer(config *Config, logger *zerolog.Logger) error {
	tracer, err := NewTracer(config, logger)
	if err != nil {
		return err
	}
	globalTracer = tracer
	return nil
}

// GetGlobalTracer returns the global tracer
func GetGlobalTracer() *Tracer {
	if globalTracer == nil {
		// Create with default config if not initialized
		nopLogger := zerolog.Nop()
		globalTracer, _ = NewTracer(DefaultConfig(), &nopLogger)
	}
	return globalTracer
}

// Global convenience functions (exported)
func GlobalStartSpan(ctx context.Context, operationName string, kind SpanKind) (context.Context, *Span) {
	return GetGlobalTracer().StartSpan(ctx, operationName, kind)
}

func GlobalSetAttribute(span *Span, key string, value interface{}) {
	GetGlobalTracer().SetAttribute(span, key, value)
}

func GlobalAddEvent(span *Span, name string, attributes map[string]interface{}) {
	GetGlobalTracer().AddEvent(span, name, attributes)
}

func GlobalSetError(span *Span, err error) {
	GetGlobalTracer().SetError(span, err)
}

func GlobalFinishSpan(span *Span) {
	GetGlobalTracer().FinishSpan(span)
}

func GlobalGetTraceContext(ctx context.Context) *SpanContext {
	return GetGlobalTracer().GetTraceContext(ctx)
}