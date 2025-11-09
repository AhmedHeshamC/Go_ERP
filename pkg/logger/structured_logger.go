package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel represents the logging level
type LogLevel string

const (
	TraceLevel LogLevel = "trace"
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
	PanicLevel LogLevel = "panic"
)

// LogFormat represents the log output format
type LogFormat string

const (
	JSONFormat LogFormat = "json"
	ConsoleFormat LogFormat = "console"
)

// Config holds the structured logger configuration
type Config struct {
	// General settings
	Level      LogLevel `json:"level"`
	Format     LogFormat `json:"format"`
	Output     string   `json:"output"`
	MaxSize    int      `json:"max_size"`    // MB
	MaxBackups int      `json:"max_backups"`
	MaxAge     int      `json:"max_age"`     // days
	Compress   bool     `json:"compress"`

	// Structured logging settings
	EnableTimestamp     bool `json:"enable_timestamp"`
	EnableLevel         bool `json:"enable_level"`
	EnableCaller        bool `json:"enable_caller"`
	EnableStacktrace    bool `json:"enable_stacktrace"`
	EnableStackTraceOnError bool `json:"enable_stacktrace_on_error"`
	TimeFieldFormat     string `json:"time_field_format"`

	// Context settings
	EnableRequestID      bool `json:"enable_request_id"`
	EnableCorrelationID  bool `json:"enable_correlation_id"`
	EnableUserID         bool `json:"enable_user_id"`
	EnableTraceID        bool `json:"enable_trace_id"`
	EnableSpanID         bool `json:"enable_span_id"`

	// Sampling settings
	EnableSampling       bool    `json:"enable_sampling"`
	SampleLevel          LogLevel `json:"sample_level"`
	SampleRate           int      `json:"sample_rate"`  // Sample every N messages

	// Performance settings
	BufferSize           int      `json:"buffer_size"`
	FlushInterval        time.Duration `json:"flush_interval"`

	// Security settings
	SanitizeFields       []string `json:"sanitize_fields"`
	EnableFieldMasking   bool     `json:"enable_field_masking"`
	MaskingChar          string   `json:"masking_char"`

	// Development settings
	EnablePrettyPrint    bool `json:"enable_pretty_print"`
	EnableColors         bool `json:"enable_colors"`
}

// DefaultConfig returns a default structured logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:                InfoLevel,
		Format:               JSONFormat,
		Output:               "stdout",
		MaxSize:              100,    // 100MB
		MaxBackups:           10,
		MaxAge:               30,     // 30 days
		Compress:             true,
		EnableTimestamp:      true,
		EnableLevel:          true,
		EnableCaller:         false,
		EnableStacktrace:     false,
		EnableStackTraceOnError: true,
		TimeFieldFormat:      time.RFC3339,
		EnableRequestID:      true,
		EnableCorrelationID:  true,
		EnableUserID:         true,
		EnableTraceID:        true,
		EnableSpanID:         true,
		EnableSampling:       false,
		SampleLevel:          DebugLevel,
		SampleRate:           100,
		BufferSize:           1000,
		FlushInterval:        1 * time.Second,
		SanitizeFields: []string{
			"password", "token", "secret", "key", "auth",
			"credit_card", "ssn", "social_security",
		},
		EnableFieldMasking:   true,
		MaskingChar:          "*",
		EnablePrettyPrint:    false,
		EnableColors:         false,
	}
}

// DevelopmentConfig returns a development-friendly configuration
func DevelopmentConfig() *Config {
	config := DefaultConfig()
	config.Level = DebugLevel
	config.Format = ConsoleFormat
	config.EnableCaller = true
	config.EnableStacktrace = true
	config.EnablePrettyPrint = true
	config.EnableColors = true
	config.EnableSampling = false
	return config
}

// ProductionConfig returns a production-safe configuration
func ProductionConfig() *Config {
	config := DefaultConfig()
	config.Level = InfoLevel
	config.Format = JSONFormat
	config.EnableCaller = false
	config.EnableStacktrace = false
	config.EnableStackTraceOnError = true
	config.EnableSampling = true
	config.SampleLevel = DebugLevel
	config.SampleRate = 10
	config.EnablePrettyPrint = false
	config.EnableColors = false
	return config
}

// Context keys for logger context
type contextKey string

const (
	RequestIDKey      contextKey = "request_id"
	CorrelationIDKey  contextKey = "correlation_id"
	UserIDKey         contextKey = "user_id"
	TraceIDKey        contextKey = "trace_id"
	SpanIDKey         contextKey = "span_id"
	LoggerKey         contextKey = "logger"
)

// Logger wraps zerolog.Logger with additional functionality
type Logger struct {
	zerolog.Logger
	config   *Config
	mu       sync.RWMutex
	sampler  *sampler
	masker   *fieldMasker
}

// sampler implements log sampling
type sampler struct {
	mu       sync.Mutex
	counter  int
	level    zerolog.Level
	rate     int
}

// fieldMasker implements field sanitization
type fieldMasker struct {
	fields  map[string]bool
	char    string
}

// NewLogger creates a new structured logger
func NewLogger(config *Config) (*Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create output writer
	output, err := createOutput(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create output: %w", err)
	}

	// Create zerolog logger
	zLogger := zerolog.New(output)

	// Configure zerolog logger
	if config.EnableTimestamp {
		zLogger = zLogger.With().Timestamp().Logger()
	}

	if config.EnableCaller {
		zLogger = zLogger.With().Caller().Logger()
	}

	if config.EnableStacktrace {
		zLogger = zLogger.With().Stack().Logger()
	}

	// Set log level
	level := parseLevel(config.Level)
	zLogger = zLogger.Level(level)

	// Create logger with additional functionality
	logger := &Logger{
		Logger:  zLogger,
		config:  config,
		sampler: newSampler(parseLevel(config.SampleLevel), config.SampleRate),
		masker:  newFieldMasker(config.SanitizeFields, config.MaskingChar),
	}

	// Enable sampling if configured
	if config.EnableSampling {
		logger.Logger = logger.Logger.Hook(logger.sampler)
	}

	return logger, nil
}

// createOutput creates the output writer for the logger
func createOutput(config *Config) (io.Writer, error) {
	switch config.Output {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "discard":
		return io.Discard, nil
	default:
		// Assume file path
		if err := os.MkdirAll(filepath.Dir(config.Output), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		if config.MaxSize > 0 {
			// Use lumberjack for log rotation
			return &lumberjack.Logger{
				Filename:   config.Output,
				MaxSize:    config.MaxSize,
				MaxBackups: config.MaxBackups,
				MaxAge:     config.MaxAge,
				Compress:   config.Compress,
			}, nil
		}

		return os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	}
}

// parseLevel converts string log level to zerolog level
func parseLevel(level LogLevel) zerolog.Level {
	switch level {
	case TraceLevel:
		return zerolog.TraceLevel
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	case FatalLevel:
		return zerolog.FatalLevel
	case PanicLevel:
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// newSampler creates a new sampler
func newSampler(level zerolog.Level, rate int) *sampler {
	return &sampler{
		level: level,
		rate:  rate,
	}
}

// newFieldMasker creates a new field masker
func newFieldMasker(fields []string, char string) *fieldMasker {
	fieldMap := make(map[string]bool)
	for _, field := range fields {
		fieldMap[strings.ToLower(field)] = true
	}
	return &fieldMasker{
		fields: fieldMap,
		char:   char,
	}
}

// Hook implements zerolog.Hook for sampling
func (s *sampler) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if level >= s.level {
		s.mu.Lock()
		s.counter++
		shouldSample := s.counter%s.rate == 0
		s.mu.Unlock()

		if !shouldSample {
			e.Discard()
		}
	}
}

// WithContext creates a logger with context fields
func (l *Logger) WithContext(ctx context.Context) *Logger {
	event := l.With()

	// Add request ID
	if l.config.EnableRequestID {
		if requestID := ctx.Value(RequestIDKey); requestID != nil {
			if id, ok := requestID.(string); ok {
				event = event.Str("request_id", id)
			}
		}
	}

	// Add correlation ID
	if l.config.EnableCorrelationID {
		if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
			if id, ok := correlationID.(string); ok {
				event = event.Str("correlation_id", id)
			}
		}
	}

	// Add user ID
	if l.config.EnableUserID {
		if userID := ctx.Value(UserIDKey); userID != nil {
			if id, ok := userID.(string); ok {
				event = event.Str("user_id", id)
			}
		}
	}

	// Add trace ID
	if l.config.EnableTraceID {
		if traceID := ctx.Value(TraceIDKey); traceID != nil {
			if id, ok := traceID.(string); ok {
				event = event.Str("trace_id", id)
			}
		}
	}

	// Add span ID
	if l.config.EnableSpanID {
		if spanID := ctx.Value(SpanIDKey); spanID != nil {
			if id, ok := spanID.(string); ok {
				event = event.Str("span_id", id)
			}
		}
	}

	return &Logger{
		Logger: event.Logger(),
		config: l.config,
	}
}

// WithFields creates a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	event := l.With()
	for key, value := range fields {
		// Sanitize sensitive fields
		if l.config.EnableFieldMasking && l.masker.shouldMask(key) {
			value = l.masker.mask(value)
		}
		event = event.Interface(key, value)
	}
	return &Logger{
		Logger: event.Logger(),
		config: l.config,
	}
}

// WithField creates a logger with a single field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	// Sanitize sensitive fields
	if l.config.EnableFieldMasking && l.masker.shouldMask(key) {
		value = l.masker.mask(value)
	}
	return &Logger{
		Logger: l.With().Interface(key, value).Logger(),
		config: l.config,
	}
}

// WithError creates a logger with an error field and stack trace if configured
func (l *Logger) WithError(err error) *Logger {
	event := l.With().Err(err)

	// Add stack trace for errors if configured
	if l.config.EnableStackTraceOnError {
		if pc, file, line, ok := runtime.Caller(1); ok {
			event = event.Str("stack", fmt.Sprintf("%s:%d %s",
				filepath.Base(file), line, runtime.FuncForPC(pc).Name()))
		}
	}

	return &Logger{
		Logger: event.Logger(),
		config: l.config,
	}
}

// HTTP logging methods
func (l *Logger) LogHTTPRequest(method, path, remoteAddr, userAgent string, duration time.Duration, statusCode int) {
	l.Info().
		Str("method", method).
		Str("path", path).
		Str("remote_addr", remoteAddr).
		Str("user_agent", userAgent).
		Dur("duration", duration).
		Int("status_code", statusCode).
		Msg("HTTP request")
}

func (l *Logger) LogHTTPResponse(method, path string, statusCode int, bodySize int64, duration time.Duration) {
	l.Info().
		Str("method", method).
		Str("path", path).
		Int("status_code", statusCode).
		Int64("body_size", bodySize).
		Dur("duration", duration).
		Msg("HTTP response")
}

// Database logging methods
func (l *Logger) LogDBQuery(query string, args []interface{}, duration time.Duration, rowsAffected int64) {
	maskedQuery := l.masker.maskSensitiveData(query)
	if queryStr, ok := maskedQuery.(string); ok {
		l.Debug().
			Str("query", queryStr).
			Interface("args", l.masker.maskSensitiveData(args)).
			Dur("duration", duration).
			Int64("rows_affected", rowsAffected).
			Msg("Database query")
	}
}

func (l *Logger) LogDBError(operation string, err error, query string, args []interface{}) {
	maskedQuery := l.masker.maskSensitiveData(query)
	if queryStr, ok := maskedQuery.(string); ok {
		l.Error().
			Err(err).
			Str("operation", operation).
			Str("query", queryStr).
			Interface("args", l.masker.maskSensitiveData(args)).
			Msg("Database error")
	}
}

// Performance logging methods
func (l *Logger) LogPerformance(operation string, duration time.Duration, metadata map[string]interface{}) {
	event := l.Info().
		Str("operation", operation).
		Dur("duration", duration)

	for key, value := range metadata {
		event = event.Interface(key, value)
	}

	event.Msg("Performance metric")
}

// Security logging methods
func (l *Logger) LogSecurityEvent(event, userID, remoteAddr string, metadata map[string]interface{}) {
	logEvent := l.Warn().
		Str("security_event", event).
		Str("user_id", userID).
		Str("remote_addr", remoteAddr)

	for key, value := range metadata {
		logEvent = logEvent.Interface(key, value)
	}

	logEvent.Msg("Security event")
}

func (l *Logger) LogAuthAttempt(userID, remoteAddr, userAgent string, success bool) {
	level := l.Info()
	if !success {
		level = l.Warn()
	}

	level.Str("user_id", userID).
		Str("remote_addr", remoteAddr).
		Str("user_agent", userAgent).
		Bool("success", success).
		Msg("Authentication attempt")
}

// Business logic logging methods
func (l *Logger) LogBusinessEvent(event string, userID string, data map[string]interface{}) {
	eventLog := l.Info().
		Str("business_event", event).
		Str("user_id", userID)

	for key, value := range data {
		eventLog = eventLog.Interface(key, value)
	}

	eventLog.Msg("Business event")
}

// System logging methods
func (l *Logger) LogSystemEvent(event string, data map[string]interface{}) {
	eventLog := l.Info().
		Str("system_event", event)

	for key, value := range data {
		eventLog = eventLog.Interface(key, value)
	}

	eventLog.Msg("System event")
}

// shouldMask checks if a field should be masked
func (fm *fieldMasker) shouldMask(field string) bool {
	lowerField := strings.ToLower(field)
	for sensitiveField := range fm.fields {
		if strings.Contains(lowerField, sensitiveField) {
			return true
		}
	}
	return false
}

// mask masks a sensitive value
func (fm *fieldMasker) mask(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		if len(v) <= 4 {
			return strings.Repeat(fm.char, len(v))
		}
		return v[:2] + strings.Repeat(fm.char, len(v)-2)
	case []byte:
		if len(v) <= 4 {
			return string(strings.Repeat(fm.char, len(v)))
		}
		return string(v[:2]) + strings.Repeat(fm.char, len(v)-2)
	default:
		return strings.Repeat(fm.char, 8)
	}
}

// maskSensitiveData masks sensitive data in complex structures
func (fm *fieldMasker) maskSensitiveData(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	switch v := data.(type) {
	case string:
		if fm.shouldMask("data") {
			return fm.mask(v)
		}
		return v
	case []interface{}:
		for i, item := range v {
			v[i] = fm.maskSensitiveData(item)
		}
		return v
	case map[string]interface{}:
		for key, value := range v {
			if fm.shouldMask(key) {
				v[key] = fm.mask(value)
			} else {
				v[key] = fm.maskSensitiveData(value)
			}
		}
		return v
	default:
		return data
	}
}

// GetStats returns logger statistics
func (l *Logger) GetStats() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return map[string]interface{}{
		"level":                  l.config.Level,
		"format":                 l.config.Format,
		"output":                 l.config.Output,
		"enable_timestamp":       l.config.EnableTimestamp,
		"enable_caller":          l.config.EnableCaller,
		"enable_stacktrace":      l.config.EnableStacktrace,
		"enable_sampling":        l.config.EnableSampling,
		"sample_rate":            l.config.SampleRate,
		"enable_field_masking":   l.config.EnableFieldMasking,
		"sanitized_fields_count": len(l.config.SanitizeFields),
	}
}

// Context helper functions

// WithStructuredRequestID adds request ID to context
func WithStructuredRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		requestID = uuid.New().String()
	}
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithStructuredCorrelationID adds correlation ID to context
func WithStructuredCorrelationID(ctx context.Context, correlationID string) context.Context {
	if correlationID == "" {
		correlationID = uuid.New().String()
	}
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// WithStructuredUserID adds user ID to context
func WithStructuredUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithStructuredTraceID adds trace ID to context
func WithStructuredTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithStructuredSpanID adds span ID to context
func WithStructuredSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// GetRequestID gets request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// GetCorrelationID gets correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		if id, ok := correlationID.(string); ok {
			return id
		}
	}
	return ""
}

// GetUserID gets user ID from context
func GetUserID(ctx context.Context) string {
	if userID := ctx.Value(UserIDKey); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// Enhanced correlation ID management with distributed tracing support

// DistributedTraceContext holds distributed tracing context
type DistributedTraceContext struct {
	TraceID      string `json:"trace_id"`
	SpanID       string `json:"span_id"`
	ParentSpanID string `json:"parent_span_id,omitempty"`
	TraceFlags   uint8  `json:"trace_flags"`
	Sampled      bool   `json:"sampled"`
}

// FromContext extracts trace context from the standard Go context
func FromContext(ctx context.Context) *DistributedTraceContext {
	traceCtx := &DistributedTraceContext{}

	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		if id, ok := traceID.(string); ok {
			traceCtx.TraceID = id
		}
	}

	if spanID := ctx.Value(SpanIDKey); spanID != nil {
		if id, ok := spanID.(string); ok {
			traceCtx.SpanID = id
		}
	}

	return traceCtx
}

// ToContext adds trace context to the standard Go context
func ToContext(ctx context.Context, traceCtx *DistributedTraceContext) context.Context {
	ctx = WithStructuredTraceID(ctx, traceCtx.TraceID)
	ctx = WithStructuredSpanID(ctx, traceCtx.SpanID)
	return ctx
}

// ExtractTraceContextFromHeaders extracts W3C trace context from HTTP headers
func ExtractTraceContextFromHeaders(headers map[string]string) (*DistributedTraceContext, error) {
	traceCtx := &DistributedTraceContext{}

	// Extract from traceparent header (W3C Trace Context)
	if traceparent := headers["traceparent"]; traceparent != "" {
		parts := strings.Split(traceparent, "-")
		if len(parts) >= 4 {
			traceCtx.TraceID = parts[1]
			traceCtx.SpanID = parts[2]
			traceCtx.TraceFlags = parseHexToUint8(parts[3])
			traceCtx.Sampled = (traceCtx.TraceFlags & 0x01) != 0
		}
	}

	// Extract from tracestate header
	if tracestate := headers["tracestate"]; tracestate != "" {
		// Parse additional trace state information
		// This would contain vendor-specific trace data
	}

	// Fallback to x-trace-id header for legacy systems
	if traceCtx.TraceID == "" {
		if traceID := headers["x-trace-id"]; traceID != "" {
			traceCtx.TraceID = traceID
		}
	}

	// Generate new IDs if not present
	if traceCtx.TraceID == "" {
		traceCtx.TraceID = uuid.New().String()
	}
	if traceCtx.SpanID == "" {
		traceCtx.SpanID = generateSpanID()
	}

	return traceCtx, nil
}

// InjectTraceContextToHeaders injects trace context into HTTP headers
func InjectTraceContextToHeaders(traceCtx *DistributedTraceContext) map[string]string {
	headers := make(map[string]string)

	// W3C Trace Context format
	traceparent := fmt.Sprintf("00-%s-%s-%02x",
		traceCtx.TraceID,
		traceCtx.SpanID,
		traceCtx.TraceFlags)
	headers["traceparent"] = traceparent

	// Legacy headers
	headers["x-trace-id"] = traceCtx.TraceID
	headers["x-span-id"] = traceCtx.SpanID

	// Correlation ID (for systems that don't support full tracing)
	correlationID := traceCtx.TraceID
	if traceCtx.SpanID != "" {
		correlationID += ":" + traceCtx.SpanID
	}
	headers["x-correlation-id"] = correlationID

	return headers
}

// CreateChildSpan creates a child span context from a parent context
func CreateChildSpan(parentCtx context.Context, operationName string) context.Context {
	parentTraceCtx := FromContext(parentCtx)

	childTraceCtx := &DistributedTraceContext{
		TraceID:      parentTraceCtx.TraceID,
		SpanID:       generateSpanID(),
		ParentSpanID: parentTraceCtx.SpanID,
		TraceFlags:   parentTraceCtx.TraceFlags,
		Sampled:      parentTraceCtx.Sampled,
	}

	childCtx := ToContext(parentCtx, childTraceCtx)

	// Add operation name to context for logging
	return context.WithValue(childCtx, contextKey("operation_name"), operationName)
}

// propagateAcrossServices simulates trace propagation across microservices
func propagateAcrossServices(ctx context.Context, targetService string) context.Context {
	traceCtx := FromContext(ctx)

	// Create a new span for the target service
	serviceSpanCtx := &DistributedTraceContext{
		TraceID:      traceCtx.TraceID,
		SpanID:       generateSpanID(),
		ParentSpanID: traceCtx.SpanID,
		TraceFlags:   traceCtx.TraceFlags,
		Sampled:      traceCtx.Sampled,
	}

	// Add service information
	serviceCtx := ToContext(ctx, serviceSpanCtx)
	serviceCtx = context.WithValue(serviceCtx, contextKey("target_service"), targetService)

	return serviceCtx
}

// parseHexToUint8 converts a hex string to uint8
func parseHexToUint8(hexStr string) uint8 {
	var result uint8
	fmt.Sscanf(hexStr, "%02x", &result)
	return result
}

// generateSpanID generates a random span ID
func generateSpanID() string {
	// Generate a 16-character hex span ID (64 bits)
	newUUID := uuid.New()
	uuidBytes := newUUID[:]
	if len(uuidBytes) >= 8 {
		return fmt.Sprintf("%016x", uuidBytes[:8])
	}
	// Fallback to using string representation
	return fmt.Sprintf("%016x", newUUID.String()[:16])
}

// CrossServiceContext creates context for calling another service
func CrossServiceContext(ctx context.Context, targetService string) context.Context {
	return propagateAcrossServices(ctx, targetService)
}

// FromStructuredContext gets logger from context or creates a new one
func FromStructuredContext(ctx context.Context) *Logger {
	if logger := ctx.Value(LoggerKey); logger != nil {
		if l, ok := logger.(*Logger); ok {
			return l
		}
	}

	// Return a default logger if none found in context
	return &Logger{
		Logger: log.Logger,
		config: DefaultConfig(),
	}
}

// WithStructuredLogger adds logger to context
func WithStructuredLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

// Global logger instance
var globalLogger *Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(config *Config) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// GetGlobalLogger returns the global logger
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		globalLogger, _ = NewLogger(DefaultConfig())
	}
	return globalLogger
}

// Global logger convenience functions
func StructuredTrace() *zerolog.Event {
	return globalLogger.Trace()
}

func StructuredDebug() *zerolog.Event {
	return globalLogger.Debug()
}

func StructuredInfo() *zerolog.Event {
	return globalLogger.Info()
}

func StructuredWarn() *zerolog.Event {
	return globalLogger.Warn()
}

func StructuredError() *zerolog.Event {
	return globalLogger.Error()
}

func StructuredFatal() *zerolog.Event {
	return globalLogger.Fatal()
}

func StructuredPanic() *zerolog.Event {
	return globalLogger.Panic()
}

func StructuredWithContext(ctx context.Context) *Logger {
	return globalLogger.WithContext(ctx)
}

func StructuredWithFields(fields map[string]interface{}) *Logger {
	return globalLogger.WithFields(fields)
}

func StructuredWithField(key string, value interface{}) *Logger {
	return globalLogger.WithField(key, value)
}

func StructuredWithError(err error) *Logger {
	return globalLogger.WithError(err)
}