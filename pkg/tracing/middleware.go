package tracing

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// TracingMiddleware provides automatic request tracing
type TracingMiddleware struct {
	tracer *Tracer
	logger *zerolog.Logger
	config *MiddlewareConfig
}

// MiddlewareConfig holds the tracing middleware configuration
type MiddlewareConfig struct {
	// Basic settings
	OperationName string `json:"operation_name"`
	ComponentName string `json:"component_name"`
	ServiceName   string `json:"service_name"`

	// Span settings
	CreateClientSpans   bool  `json:"create_client_spans"`
	IncludeRequestBody  bool  `json:"include_request_body"`
	IncludeResponseBody bool  `json:"include_response_body"`
	MaxRequestBodySize  int64 `json:"max_request_body_size"`
	MaxResponseBodySize int64 `json:"max_response_body_size"`

	// Attribute settings
	IncludeUserAttributes bool                   `json:"include_user_attributes"`
	IncludeHeaders        bool                   `json:"include_headers"`
	HeaderBlacklist       []string               `json:"header_blacklist"`
	CustomAttributes      map[string]interface{} `json:"custom_attributes"`

	// Event settings
	RecordRequestEvents  bool `json:"record_request_events"`
	RecordResponseEvents bool `json:"record_response_events"`
	RecordErrorEvents    bool `json:"record_error_events"`

	// Sampling settings
	IgnorePaths        []string `json:"ignore_paths"`
	IgnoreUserAgents   []string `json:"ignore_user_agents"`
	SampleRateOverride *float64 `json:"sample_rate_override"`

	// Error handling
	IgnoreErrors     bool  `json:"ignore_errors"`
	ErrorStatusCodes []int `json:"error_status_codes"`

	// Performance settings
	SlowRequestThreshold time.Duration `json:"slow_request_threshold"`
	RecordSlowRequests   bool          `json:"record_slow_requests"`
}

// DefaultMiddlewareConfig returns a default middleware configuration
func DefaultMiddlewareConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		OperationName:         "HTTP Request",
		ComponentName:         "gin",
		ServiceName:           "erp-go-api",
		CreateClientSpans:     false,
		IncludeRequestBody:    false,
		IncludeResponseBody:   false,
		MaxRequestBodySize:    1024 * 32, // 32KB
		MaxResponseBodySize:   1024 * 64, // 64KB
		IncludeUserAttributes: true,
		IncludeHeaders:        false,
		HeaderBlacklist: []string{
			"authorization", "cookie", "set-cookie",
			"x-api-key", "x-auth-token",
		},
		CustomAttributes:     make(map[string]interface{}),
		RecordRequestEvents:  true,
		RecordResponseEvents: true,
		RecordErrorEvents:    true,
		IgnorePaths: []string{
			"/health", "/metrics", "/ready", "/live",
		},
		IgnoreUserAgents: []string{
			"HealthChecker", "kube-probe", "UptimeRobot",
		},
		SampleRateOverride:   nil,
		IgnoreErrors:         false,
		ErrorStatusCodes:     []int{404}, // Ignore 404s by default
		SlowRequestThreshold: 1 * time.Second,
		RecordSlowRequests:   true,
	}
}

// NewTracingMiddleware creates a new tracing middleware
func NewTracingMiddleware(tracer *Tracer, logger *zerolog.Logger, config *MiddlewareConfig) *TracingMiddleware {
	if config == nil {
		config = DefaultMiddlewareConfig()
	}

	if logger == nil {
		nopLogger := zerolog.Nop()
		logger = &nopLogger
	}

	return &TracingMiddleware{
		tracer: tracer,
		logger: logger,
		config: config,
	}
}

// Middleware returns the Gin middleware function
func (m *TracingMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if path should be ignored
		if m.shouldIgnorePath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Check if user agent should be ignored
		if m.shouldIgnoreUserAgent(c.Request.UserAgent()) {
			c.Next()
			return
		}

		// Start span
		operationName := m.getOperationName(c)
		_, span := m.tracer.StartSpan(c.Request.Context(), operationName, SpanKindServer)

		// Add span to Gin context
		c.Set("span", span)

		// Set up response writer to capture response
		responseWriter := &responseWriter{
			ResponseWriter: c.Writer,
			body:           make([]byte, 0),
		}
		c.Writer = responseWriter

		// Add initial attributes
		m.addRequestAttributes(span, c)

		// Record request start event
		if m.config.RecordRequestEvents {
			m.tracer.AddEvent(span, "request.started", map[string]interface{}{
				"http.method":      c.Request.Method,
				"http.url":         c.Request.URL.String(),
				"http.user_agent":  c.Request.UserAgent(),
				"http.remote_addr": c.ClientIP(),
			})
		}

		// Process request
		c.Next()

		// Add response attributes
		m.addResponseAttributes(span, c, responseWriter)

		// Record response end event
		if m.config.RecordResponseEvents {
			m.tracer.AddEvent(span, "request.completed", map[string]interface{}{
				"http.status_code":   c.Writer.Status(),
				"http.response_size": len(responseWriter.body),
			})
		}

		// Check for errors
		if len(c.Errors) > 0 {
			m.handleErrors(span, c)
		} else {
			// Check for error status codes
			if m.isErrorStatusCode(c.Writer.Status()) {
				m.handleStatusCodeError(span, c.Writer.Status())
			}
		}

		// Check for slow requests
		if m.config.RecordSlowRequests && span.Duration > m.config.SlowRequestThreshold {
			m.tracer.AddEvent(span, "request.slow", map[string]interface{}{
				"duration_ms":  span.Duration.Milliseconds(),
				"threshold_ms": m.config.SlowRequestThreshold.Milliseconds(),
			})
		}

		// Finish span
		m.tracer.FinishSpan(span)

		// Add trace headers to response
		m.addTraceHeaders(c, span)
	}
}

// getOperationName gets the operation name for the span
func (m *TracingMiddleware) getOperationName(c *gin.Context) string {
	if m.config.OperationName != "" {
		return fmt.Sprintf("%s %s", m.config.OperationName, c.Request.Method)
	}
	return fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
}

// addRequestAttributes adds request attributes to the span
func (m *TracingMiddleware) addRequestAttributes(span *Span, c *gin.Context) {
	// HTTP attributes
	m.tracer.SetAttribute(span, "http.method", c.Request.Method)
	m.tracer.SetAttribute(span, "http.url", c.Request.URL.String())
	m.tracer.SetAttribute(span, "http.scheme", c.Request.URL.Scheme)
	m.tracer.SetAttribute(span, "http.host", c.Request.Host)
	m.tracer.SetAttribute(span, "http.target", c.Request.URL.Path+c.Request.URL.RawQuery)
	m.tracer.SetAttribute(span, "http.flavor", c.Request.Proto)
	m.tracer.SetAttribute(span, "http.remote_addr", c.ClientIP())
	m.tracer.SetAttribute(span, "http.user_agent", c.Request.UserAgent())
	m.tracer.SetAttribute(span, "http.request_content_length", c.Request.ContentLength)

	// Component attributes
	m.tracer.SetAttribute(span, "component", m.config.ComponentName)
	m.tracer.SetAttribute(span, "service.name", m.config.ServiceName)

	// Custom attributes
	for key, value := range m.config.CustomAttributes {
		m.tracer.SetAttribute(span, key, value)
	}

	// Headers
	if m.config.IncludeHeaders {
		headers := make(map[string]interface{})
		for key, values := range c.Request.Header {
			if !m.shouldBlacklistHeader(key) {
				headers[key] = values
			}
		}
		m.tracer.SetAttribute(span, "http.request_headers", headers)
	}

	// User attributes
	if m.config.IncludeUserAttributes {
		if userID := c.GetString("user_id"); userID != "" {
			m.tracer.SetAttribute(span, "user.id", userID)
		}
		if username := c.GetString("username"); username != "" {
			m.tracer.SetAttribute(span, "user.username", username)
		}
	}

	// Request body (if configured and not too large)
	if m.config.IncludeRequestBody && c.Request.ContentLength > 0 &&
		c.Request.ContentLength <= m.config.MaxRequestBodySize {
		// Note: In a real implementation, you'd need to capture the body before
		// it's consumed by other middleware. This is a simplified version.
		m.tracer.SetAttribute(span, "http.request_body_size", c.Request.ContentLength)
	}
}

// addResponseAttributes adds response attributes to the span
func (m *TracingMiddleware) addResponseAttributes(span *Span, c *gin.Context, rw *responseWriter) {
	// HTTP response attributes
	m.tracer.SetAttribute(span, "http.status_code", c.Writer.Status())
	m.tracer.SetAttribute(span, "http.response_content_length", int64(len(rw.body)))

	// Response body (if configured and not too large)
	if m.config.IncludeResponseBody && len(rw.body) > 0 &&
		int64(len(rw.body)) <= m.config.MaxResponseBodySize {
		// Note: Be careful with response body content as it may contain sensitive data
		m.tracer.SetAttribute(span, "http.response_body_size", len(rw.body))
	}
}

// addTraceHeaders adds trace headers to the HTTP response
func (m *TracingMiddleware) addTraceHeaders(c *gin.Context, span *Span) {
	// Add W3C traceparent header
	traceparent := fmt.Sprintf("00-%s-%s-%s-01",
		span.TraceID, span.SpanID, span.SpanID)
	c.Header("traceparent", traceparent)

	// Add tracestate header (simplified)
	c.Header("tracestate", "rojo=1")

	// Add custom trace headers
	c.Header("x-trace-id", span.TraceID)
	c.Header("x-span-id", span.SpanID)
}

// handleErrors handles Gin errors and adds them to the span
func (m *TracingMiddleware) handleErrors(span *Span, c *gin.Context) {
	if m.config.IgnoreErrors {
		return
	}

	for _, ginError := range c.Errors {
		m.tracer.SetError(span, ginError)
		m.tracer.AddEvent(span, "error", map[string]interface{}{
			"error.type":    "gin_error",
			"error.message": ginError.Error(),
		})

		if m.config.RecordErrorEvents {
			m.logger.Warn().
				Str("trace_id", span.TraceID).
				Str("span_id", span.SpanID).
				Err(ginError).
				Msg("Request error")
		}
	}
}

// handleStatusCodeError handles HTTP status code errors
func (m *TracingMiddleware) handleStatusCodeError(span *Span, statusCode int) {
	for _, ignoredCode := range m.config.ErrorStatusCodes {
		if statusCode == ignoredCode {
			return
		}
	}

	err := fmt.Errorf("HTTP %d error: %s", statusCode, http.StatusText(statusCode))
	m.tracer.SetError(span, err)
	m.tracer.AddEvent(span, "http.error", map[string]interface{}{
		"http.status_code": statusCode,
		"http.status_text": http.StatusText(statusCode),
	})

	if m.config.RecordErrorEvents {
		m.logger.Warn().
			Str("trace_id", span.TraceID).
			Str("span_id", span.SpanID).
			Int("status_code", statusCode).
			Msg("HTTP error")
	}
}

// isErrorStatusCode checks if a status code represents an error
func (m *TracingMiddleware) isErrorStatusCode(statusCode int) bool {
	return statusCode >= 400
}

// shouldIgnorePath checks if a path should be ignored
func (m *TracingMiddleware) shouldIgnorePath(path string) bool {
	for _, ignorePath := range m.config.IgnorePaths {
		if strings.HasPrefix(path, ignorePath) {
			return true
		}
	}
	return false
}

// shouldIgnoreUserAgent checks if a user agent should be ignored
func (m *TracingMiddleware) shouldIgnoreUserAgent(userAgent string) bool {
	for _, ignoreUA := range m.config.IgnoreUserAgents {
		if strings.Contains(strings.ToLower(userAgent), strings.ToLower(ignoreUA)) {
			return true
		}
	}
	return false
}

// shouldBlacklistHeader checks if a header should be blacklisted
func (m *TracingMiddleware) shouldBlacklistHeader(header string) bool {
	lowerHeader := strings.ToLower(header)
	for _, blacklist := range m.config.HeaderBlacklist {
		if lowerHeader == strings.ToLower(blacklist) {
			return true
		}
	}
	return false
}

// responseWriter captures response data
type responseWriter struct {
	gin.ResponseWriter
	body []byte
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.body = append(rw.body, b...)
	return n, err
}

// GetStats returns middleware statistics
func (m *TracingMiddleware) GetStats() map[string]interface{} {
	if m.tracer != nil {
		return m.tracer.GetStats()
	}
	return map[string]interface{}{
		"tracer_initialized": false,
	}
}

// Convenience functions for creating middleware with default configurations

// Tracing creates a tracing middleware with default configuration
func Tracing(tracer *Tracer, logger *zerolog.Logger) gin.HandlerFunc {
	middleware := NewTracingMiddleware(tracer, logger, DefaultMiddlewareConfig())
	return middleware.Middleware()
}

// DevelopmentTracing creates a tracing middleware for development
func DevelopmentTracing(tracer *Tracer, logger *zerolog.Logger) gin.HandlerFunc {
	config := &MiddlewareConfig{
		OperationName:         "HTTP Request",
		ComponentName:         "gin",
		ServiceName:           "erp-go-api",
		CreateClientSpans:     true,
		IncludeRequestBody:    true,
		IncludeResponseBody:   true,
		MaxRequestBodySize:    1024 * 64,  // 64KB
		MaxResponseBodySize:   1024 * 128, // 128KB
		IncludeUserAttributes: true,
		IncludeHeaders:        true,
		HeaderBlacklist:       []string{"authorization", "cookie"},
		CustomAttributes: map[string]interface{}{
			"environment": "development",
		},
		RecordRequestEvents:  true,
		RecordResponseEvents: true,
		RecordErrorEvents:    true,
		IgnorePaths:          []string{"/health"},
		IgnoreUserAgents:     []string{},
		SampleRateOverride:   nil,
		IgnoreErrors:         false,
		ErrorStatusCodes:     []int{},
		SlowRequestThreshold: 500 * time.Millisecond,
		RecordSlowRequests:   true,
	}

	middleware := NewTracingMiddleware(tracer, logger, config)
	return middleware.Middleware()
}

// ProductionTracing creates a tracing middleware for production
func ProductionTracing(tracer *Tracer, logger *zerolog.Logger) gin.HandlerFunc {
	config := &MiddlewareConfig{
		OperationName:         "HTTP Request",
		ComponentName:         "gin",
		ServiceName:           "erp-go-api",
		CreateClientSpans:     false,
		IncludeRequestBody:    false,
		IncludeResponseBody:   false,
		MaxRequestBodySize:    0,
		MaxResponseBodySize:   0,
		IncludeUserAttributes: false, // Don't include user data in production
		IncludeHeaders:        false,
		HeaderBlacklist: []string{
			"authorization", "cookie", "set-cookie",
			"x-api-key", "x-auth-token", "x-session-id",
		},
		CustomAttributes: map[string]interface{}{
			"environment": "production",
		},
		RecordRequestEvents:  false, // Usually handled by APM
		RecordResponseEvents: false,
		RecordErrorEvents:    false,
		IgnorePaths: []string{
			"/health", "/metrics", "/ready", "/live",
			"/ping",
		},
		IgnoreUserAgents: []string{
			"HealthChecker", "kube-probe", "UptimeRobot",
			"GoogleHC", "AWS HealthChecker",
		},
		SampleRateOverride:   nil,
		IgnoreErrors:         false,
		ErrorStatusCodes:     []int{404}, // Ignore 404s
		SlowRequestThreshold: 5 * time.Second,
		RecordSlowRequests:   false, // Usually handled by APM
	}

	middleware := NewTracingMiddleware(tracer, logger, config)
	return middleware.Middleware()
}

// Helper functions for Gin context

// GetSpan gets the current span from Gin context
func GetSpan(c *gin.Context) *Span {
	if span, exists := c.Get("span"); exists {
		if s, ok := span.(*Span); ok {
			return s
		}
	}
	return nil
}

// SetAttribute sets an attribute on the current span
func SetAttribute(c *gin.Context, key string, value interface{}) {
	if span := GetSpan(c); span != nil && globalTracer != nil {
		globalTracer.SetAttribute(span, key, value)
	}
}

// AddEvent adds an event to the current span
func AddEvent(c *gin.Context, name string, attributes map[string]interface{}) {
	if span := GetSpan(c); span != nil && globalTracer != nil {
		globalTracer.AddEvent(span, name, attributes)
	}
}

// SetError sets an error on the current span
func SetError(c *gin.Context, err error) {
	if span := GetSpan(c); span != nil && globalTracer != nil {
		globalTracer.SetError(span, err)
	}
}

// StartClientSpan starts a client span from within a server span
func StartClientSpan(c *gin.Context, operationName string) (context.Context, *Span) {
	if globalTracer != nil {
		return globalTracer.StartSpan(c.Request.Context(), operationName, SpanKindClient)
	}
	return c.Request.Context(), nil
}
