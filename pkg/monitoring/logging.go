package monitoring

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ContextLogger provides enhanced logging with correlation and tracing
type ContextLogger struct {
	logger *zerolog.Logger
}

// LogContext holds correlation and tracing information
type LogContext struct {
	CorrelationID string
	TraceID       string
	SpanID        string
	UserID        string
	RequestID     string
	SessionID     string
	Component     string
	Version       string
}

// NewContextLogger creates a new context logger
func NewContextLogger(logger *zerolog.Logger) *ContextLogger {
	return &ContextLogger{
		logger: logger,
	}
}

// WithContext adds context fields to the logger
func (cl *ContextLogger) WithContext(ctx context.Context, logCtx LogContext) zerolog.Logger {
	event := cl.logger.With()

	// Add correlation and tracing fields
	if logCtx.CorrelationID != "" {
		event = event.Str("correlation_id", logCtx.CorrelationID)
	}
	if logCtx.TraceID != "" {
		event = event.Str("trace_id", logCtx.TraceID)
	}
	if logCtx.SpanID != "" {
		event = event.Str("span_id", logCtx.SpanID)
	}

	// Add user and session fields
	if logCtx.UserID != "" {
		event = event.Str("user_id", logCtx.UserID)
	}
	if logCtx.SessionID != "" {
		event = event.Str("session_id", logCtx.SessionID)
	}

	// Add request fields
	if logCtx.RequestID != "" {
		event = event.Str("request_id", logCtx.RequestID)
	}

	// Add component fields
	if logCtx.Component != "" {
		event = event.Str("component", logCtx.Component)
	}
	if logCtx.Version != "" {
		event = event.Str("version", logCtx.Version)
	}

	// Add system fields
	event = event.Str("hostname", getLogHostname())
	event = event.Str("pid", getLogPID())

	// Add timestamp
	event = event.Time("timestamp", time.Now().UTC())

	return event.Logger()
}

// WithRequestContext creates a logger with request context
func (cl *ContextLogger) WithRequestContext(ctx context.Context, userID, requestID, component string) zerolog.Logger {
	logCtx := LogContext{
		CorrelationID: getCorrelationIDFromContext(ctx),
		TraceID:       getTraceIDFromContext(ctx),
		SpanID:        getSpanIDFromContext(ctx),
		UserID:        userID,
		RequestID:     requestID,
		Component:     component,
	}

	return cl.WithContext(ctx, logCtx)
}

// WithComponent creates a logger with component context
func (cl *ContextLogger) WithComponent(component string) zerolog.Logger {
	return cl.logger.With().Str("component", component).Logger()
}

// WithError creates a logger with error context
func (cl *ContextLogger) WithError(err error) zerolog.Logger {
	return cl.logger.With().Err(err).Logger()
}

// Structured logging methods with enhanced context

// Info logs an info message with structured context
func (cl *ContextLogger) Info(message string, fields map[string]interface{}) {
	cl.logWithLevel(zerolog.InfoLevel, message, fields)
}

// Debug logs a debug message with structured context
func (cl *ContextLogger) Debug(message string, fields map[string]interface{}) {
	cl.logWithLevel(zerolog.DebugLevel, message, fields)
}

// Warn logs a warning message with structured context
func (cl *ContextLogger) Warn(message string, fields map[string]interface{}) {
	cl.logWithLevel(zerolog.WarnLevel, message, fields)
}

// Error logs an error message with structured context
func (cl *ContextLogger) Error(message string, err error, fields map[string]interface{}) {
	mergedFields := make(map[string]interface{})
	for k, v := range fields {
		mergedFields[k] = v
	}
	if err != nil {
		mergedFields["error"] = err.Error()
		mergedFields["error_type"] = getErrorType(err)
	}

	cl.logWithLevel(zerolog.ErrorLevel, message, mergedFields)
}

// Fatal logs a fatal message with structured context
func (cl *ContextLogger) Fatal(message string, err error, fields map[string]interface{}) {
	mergedFields := make(map[string]interface{})
	for k, v := range fields {
		mergedFields[k] = v
	}
	if err != nil {
		mergedFields["error"] = err.Error()
		mergedFields["error_type"] = getErrorType(err)
	}

	cl.logWithLevel(zerolog.FatalLevel, message, mergedFields)
}

// Audit logs an audit event with enhanced security context
func (cl *ContextLogger) Audit(action string, userID string, resource string, fields map[string]interface{}) {
	auditFields := make(map[string]interface{})
	for k, v := range fields {
		auditFields[k] = v
	}

	auditFields["action"] = action
	auditFields["user_id"] = userID
	auditFields["resource"] = resource
	auditFields["audit_event"] = true
	auditFields["timestamp"] = time.Now().UTC()

	// Add security context
	auditFields["security_level"] = getSecurityLevel(action)

	cl.logWithLevel(zerolog.InfoLevel, "AUDIT: "+action, auditFields)
}

// Performance logs a performance event with metrics
func (cl *ContextLogger) Performance(operation string, duration time.Duration, fields map[string]interface{}) {
	perfFields := make(map[string]interface{})
	for k, v := range fields {
		perfFields[k] = v
	}

	perfFields["operation"] = operation
	perfFields["duration_ms"] = duration.Milliseconds()
	perfFields["performance_event"] = true

	cl.logWithLevel(zerolog.InfoLevel, "PERF: "+operation, perfFields)
}

// Business logs a business event with context
func (cl *ContextLogger) Business(event string, fields map[string]interface{}) {
	bizFields := make(map[string]interface{})
	for k, v := range fields {
		bizFields[k] = v
	}

	bizFields["business_event"] = event
	bizFields["timestamp"] = time.Now().UTC()

	cl.logWithLevel(zerolog.InfoLevel, "BIZ: "+event, bizFields)
}

// Security logs a security event
func (cl *ContextLogger) Security(event string, severity string, fields map[string]interface{}) {
	secFields := make(map[string]interface{})
	for k, v := range fields {
		secFields[k] = v
	}

	secFields["security_event"] = event
	secFields["severity"] = severity
	secFields["timestamp"] = time.Now().UTC()

	level := zerolog.InfoLevel
	if severity == "high" || severity == "critical" {
		level = zerolog.WarnLevel
	}

	cl.logWithLevel(level, "SEC: "+event, secFields)
}

// logWithLevel logs a message at the specified level
func (cl *ContextLogger) logWithLevel(level zerolog.Level, message string, fields map[string]interface{}) {
	event := cl.logger.WithLevel(level)

	// Add all fields
	for k, v := range fields {
		event = event.Interface(k, v)
	}

	// Add caller information
	if cl.logger.GetLevel() <= zerolog.DebugLevel {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			event = event.Str("caller", file+":"+string(rune(line)))
		}
	}

	event.Msg(message)
}

// Context key types for correlation and tracing
type contextKey string

const (
	CorrelationIDKey contextKey = "correlation_id"
	TraceIDKey       contextKey = "trace_id"
	SpanIDKey        contextKey = "span_id"
	UserIDKey        contextKey = "user_id"
	RequestIDKey     contextKey = "request_id"
	SessionIDKey     contextKey = "session_id"
)

// WithCorrelationID adds correlation ID to context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// WithTraceID adds trace ID to context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithSpanID adds span ID to context
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithSessionID adds session ID to context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, SessionIDKey, sessionID)
}

// Get correlation ID from context
func getCorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return id
	}
	return ""
}

// Get trace ID from context
func getTraceIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(TraceIDKey).(string); ok {
		return id
	}
	return ""
}

// Get span ID from context
func getSpanIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(SpanIDKey).(string); ok {
		return id
	}
	return ""
}

// Public functions for context access
func GetCorrelationIDFromContext(ctx context.Context) string {
	return getCorrelationIDFromContext(ctx)
}

func GetTraceIDFromContext(ctx context.Context) string {
	return getTraceIDFromContext(ctx)
}

func GetSpanIDFromContext(ctx context.Context) string {
	return getSpanIDFromContext(ctx)
}

func GetRequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

func GetUserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(UserIDKey).(string); ok {
		return id
	}
	return ""
}

func GetSessionIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(SessionIDKey).(string); ok {
		return id
	}
	return ""
}

// Helper functions

// getHostname returns the hostname
func getLogHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// getPID returns the process ID
func getLogPID() string {
	return fmt.Sprintf("%d", os.Getpid())
}

// getErrorType returns the error type
func getErrorType(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// getSecurityLevel returns security level for an action
func getSecurityLevel(action string) string {
	// Define security levels for different actions
	secureActions := map[string]string{
		"login":           "authentication",
		"logout":          "authentication",
		"password_change": "high",
		"user_create":     "high",
		"user_delete":     "critical",
		"order_create":    "medium",
		"payment_process": "high",
		"admin_access":    "high",
	}

	if level, ok := secureActions[action]; ok {
		return level
	}
	return "low"
}

// GenerateCorrelationID generates a new correlation ID
func GenerateCorrelationID() string {
	return uuid.New().String()
}

// GenerateTraceID generates a new trace ID
func GenerateTraceID() string {
	return uuid.New().String()
}

// GenerateSpanID generates a new span ID
func GenerateSpanID() string {
	return uuid.New().String()
}

// Global context logger instance
var GlobalContextLogger *ContextLogger

// InitializeContextLogger initializes the global context logger
func InitializeContextLogger(logger *zerolog.Logger) {
	GlobalContextLogger = NewContextLogger(logger)
}

// GetContextLogger returns the global context logger
func GetContextLogger() *ContextLogger {
	if GlobalContextLogger == nil {
		// Fallback to basic logger
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		GlobalContextLogger = NewContextLogger(&logger)
	}
	return GlobalContextLogger
}