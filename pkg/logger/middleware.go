package logger

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LoggingMiddleware provides request/response logging middleware
type LoggingMiddleware struct {
	logger         *Logger
	skipPaths      []string
	logRequestBody bool
	logResponseBody bool
	maxBodySize    int64
}

// Config holds the logging middleware configuration
type MiddlewareConfig struct {
	// Skip certain paths from logging
	SkipPaths []string `json:"skip_paths"`

	// Log request and response bodies
	LogRequestBody    bool `json:"log_request_body"`
	LogResponseBody   bool `json:"log_response_body"`
	MaxBodySize       int64 `json:"max_body_size"` // bytes

	// Additional context to log
	LogUserAgent      bool `json:"log_user_agent"`
	LogRemoteAddr     bool `json:"log_remote_addr"`
	LogRequestHeaders bool `json:"log_request_headers"`
	LogResponseHeaders bool `json:"log_response_headers"`

	// Performance settings
	SlowRequestThreshold time.Duration `json:"slow_request_threshold"`
	LogSlowRequests      bool          `json:"log_slow_requests"`

	// Security settings
	SanitizeHeaders []string `json:"sanitize_headers"`
}

// DefaultMiddlewareConfig returns a default middleware configuration
func DefaultMiddlewareConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/ready",
			"/live",
		},
		LogRequestBody:     false,
		LogResponseBody:    false,
		MaxBodySize:        1024 * 64, // 64KB
		LogUserAgent:       true,
		LogRemoteAddr:      true,
		LogRequestHeaders:  false,
		LogResponseHeaders: false,
		SlowRequestThreshold: 1 * time.Second,
		LogSlowRequests:     true,
		SanitizeHeaders: []string{
			"authorization",
			"cookie",
			"set-cookie",
		},
	}
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(logger *Logger, config *MiddlewareConfig) *LoggingMiddleware {
	if config == nil {
		config = DefaultMiddlewareConfig()
	}

	return &LoggingMiddleware{
		logger:         logger,
		skipPaths:      config.SkipPaths,
		logRequestBody: config.LogRequestBody,
		logResponseBody: config.LogResponseBody,
		maxBodySize:    config.MaxBodySize,
	}
}

// Middleware returns the Gin middleware function
func (m *LoggingMiddleware) Middleware(config *MiddlewareConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultMiddlewareConfig()
	}

	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		_ = c.Request.URL.RawQuery

		// Skip logging for specified paths
		if m.shouldSkipPath(path) {
			c.Next()
			return
		}

		// Generate request ID and correlation ID if not present
		requestID := GetRequestID(c.Request.Context())
		if requestID == "" {
			requestID = uuid.New().String()
			c.Request = c.Request.WithContext(WithStructuredRequestID(c.Request.Context(), requestID))
		}

		correlationID := GetCorrelationID(c.Request.Context())
		if correlationID == "" {
			correlationID = uuid.New().String()
			c.Request = c.Request.WithContext(WithStructuredCorrelationID(c.Request.Context(), correlationID))
		}

		// Add IDs to response headers
		c.Header("X-Request-ID", requestID)
		c.Header("X-Correlation-ID", correlationID)

		// Create logger with context
		logger := m.logger.WithContext(c.Request.Context())

		// Log request
		m.logRequest(logger, c, config)

		// Capture response writer
		responseWriter := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:          &bytes.Buffer{},
		}
		c.Writer = responseWriter

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Log response
		m.logResponse(logger, c, responseWriter, duration, config)

		// Log slow requests
		if config.LogSlowRequests && duration > config.SlowRequestThreshold {
			logger.Warn().
				Str("path", path).
				Dur("duration", duration).
				Dur("threshold", config.SlowRequestThreshold).
				Msg("Slow request detected")
		}
	})
}

// shouldSkipPath checks if a path should be skipped from logging
func (m *LoggingMiddleware) shouldSkipPath(path string) bool {
	for _, skipPath := range m.skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// logRequest logs the incoming request
func (m *LoggingMiddleware) logRequest(logger *Logger, c *gin.Context, config *MiddlewareConfig) {
	fields := map[string]interface{}{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
		"query":  c.Request.URL.RawQuery,
		"proto":  c.Request.Proto,
	}

	if config.LogRemoteAddr {
		fields["remote_addr"] = c.ClientIP()
	}

	if config.LogUserAgent {
		fields["user_agent"] = c.Request.UserAgent()
	}

	// Log content length
	if c.Request.ContentLength > 0 {
		fields["content_length"] = c.Request.ContentLength
	}

	// Log request headers
	if config.LogRequestHeaders {
		headers := make(map[string]string)
		for key, values := range c.Request.Header {
			if !m.shouldSanitizeHeader(key, config.SanitizeHeaders) {
				headers[key] = strings.Join(values, ", ")
			} else {
				headers[key] = "[REDACTED]"
			}
		}
		fields["headers"] = headers
	}

	// Log request body
	if config.LogRequestBody && c.Request.ContentLength > 0 && c.Request.ContentLength <= m.maxBodySize {
		body := m.readRequestBody(c)
		if body != "" {
			fields["body"] = body
		}
	}

	logger.WithFields(fields).Info().Msg("Incoming request")
}

// logResponse logs the outgoing response
func (m *LoggingMiddleware) logResponse(logger *Logger, c *gin.Context, rw *responseBodyWriter, duration time.Duration, config *MiddlewareConfig) {
	fields := map[string]interface{}{
		"status_code": rw.Status(),
		"duration":    duration,
		"size":        rw.body.Len(),
	}

	// Log response headers
	if config.LogResponseHeaders {
		headers := make(map[string]string)
		for key, values := range rw.Header() {
			if !m.shouldSanitizeHeader(key, config.SanitizeHeaders) {
				headers[key] = strings.Join(values, ", ")
			} else {
				headers[key] = "[REDACTED]"
			}
		}
		fields["response_headers"] = headers
	}

	// Log response body
	if config.LogResponseBody && rw.body.Len() > 0 && int64(rw.body.Len()) <= m.maxBodySize {
		fields["response_body"] = rw.body.String()
	}

	// Add user ID if available
	if userID := GetUserID(c.Request.Context()); userID != "" {
		fields["user_id"] = userID
	}

	// Determine log level based on status code
	statusCode := rw.Status()
	switch {
	case statusCode >= 500:
		logger.WithFields(fields).Error().Msg("Request completed with server error")
	case statusCode >= 400:
		logger.WithFields(fields).Warn().Msg("Request completed with client error")
	case statusCode >= 300:
		logger.WithFields(fields).Info().Msg("Request completed with redirect")
	default:
		logger.WithFields(fields).Info().Msg("Request completed successfully")
	}
}

// readRequestBody reads the request body
func (m *LoggingMiddleware) readRequestBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	// Read body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return "[BODY_READ_ERROR]"
	}

	// Restore body for subsequent reads
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return string(bodyBytes)
}

// shouldSanitizeHeader checks if a header should be sanitized
func (m *LoggingMiddleware) shouldSanitizeHeader(header string, sanitizeHeaders []string) bool {
	lowerHeader := strings.ToLower(header)
	for _, sensitiveHeader := range sanitizeHeaders {
		if lowerHeader == strings.ToLower(sensitiveHeader) {
			return true
		}
	}
	return false
}

// responseBodyWriter captures the response body
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// Helper functions for Gin context

// GetLoggerFromContext gets logger from Gin context
func GetLoggerFromContext(c *gin.Context) *Logger {
	logger := c.Value("logger")
	if l, ok := logger.(*Logger); ok {
		return l
	}
	return GetGlobalLogger()
}

// SetLoggerInContext sets logger in Gin context
func SetLoggerInContext(c *gin.Context, logger *Logger) {
	c.Set("logger", logger)
}

// LogError logs an error with request context
func LogError(c *gin.Context, err error, message string) {
	logger := GetLoggerFromContext(c).WithError(err)
	if message != "" {
		logger = logger.WithField("message", message)
	}
	logger.Error().Msg("Request error")
}

// LogInfo logs an info message with request context
func LogInfo(c *gin.Context, message string, fields map[string]interface{}) {
	logger := GetLoggerFromContext(c).WithFields(fields)
	logger.Info().Msg(message)
}

// LogWarn logs a warning message with request context
func LogWarn(c *gin.Context, message string, fields map[string]interface{}) {
	logger := GetLoggerFromContext(c).WithFields(fields)
	logger.Warn().Msg(message)
}

// LogDebug logs a debug message with request context
func LogDebug(c *gin.Context, message string, fields map[string]interface{}) {
	logger := GetLoggerFromContext(c).WithFields(fields)
	logger.Debug().Msg(message)
}

// LogSecurityEvent logs a security event with request context
func LogSecurityEvent(c *gin.Context, event string, data map[string]interface{}) {
	logger := GetLoggerFromContext(c)
	userID := GetUserID(c.Request.Context())
	remoteAddr := c.ClientIP()

	logger.LogSecurityEvent(event, userID, remoteAddr, data)
}

// LogBusinessEvent logs a business event with request context
func LogBusinessEvent(c *gin.Context, event string, data map[string]interface{}) {
	logger := GetLoggerFromContext(c)
	userID := GetUserID(c.Request.Context())

	logger.LogBusinessEvent(event, userID, data)
}

// Convenience middleware function

// RequestLogging creates a request logging middleware
func RequestLogging(logger *Logger, config *MiddlewareConfig) gin.HandlerFunc {
	middleware := NewLoggingMiddleware(logger, config)
	return middleware.Middleware(config)
}

// DefaultRequestLogging creates a request logging middleware with default configuration
func DefaultRequestLogging(logger *Logger) gin.HandlerFunc {
	return RequestLogging(logger, DefaultMiddlewareConfig())
}

// DevelopmentRequestLogging creates a request logging middleware for development
func DevelopmentRequestLogging(logger *Logger) gin.HandlerFunc {
	config := &MiddlewareConfig{
		SkipPaths:           []string{"/health", "/metrics"},
		LogRequestBody:       true,
		LogResponseBody:      true,
		MaxBodySize:          1024 * 64, // 64KB
		LogUserAgent:         true,
		LogRemoteAddr:        true,
		LogRequestHeaders:    true,
		LogResponseHeaders:   true,
		SlowRequestThreshold: 100 * time.Millisecond,
		LogSlowRequests:      true,
		SanitizeHeaders:      []string{"authorization", "cookie"},
	}

	return RequestLogging(logger, config)
}

// ProductionRequestLogging creates a request logging middleware for production
func ProductionRequestLogging(logger *Logger) gin.HandlerFunc {
	config := &MiddlewareConfig{
		SkipPaths:           []string{"/health", "/metrics", "/ready", "/live"},
		LogRequestBody:       false,
		LogResponseBody:      false,
		MaxBodySize:          1024 * 32, // 32KB
		LogUserAgent:         false,
		LogRemoteAddr:        true,
		LogRequestHeaders:    false,
		LogResponseHeaders:   false,
		SlowRequestThreshold: 2 * time.Second,
		LogSlowRequests:      true,
		SanitizeHeaders:      []string{
			"authorization", "cookie", "set-cookie",
			"x-api-key", "x-auth-token",
		},
	}

	return RequestLogging(logger, config)
}
// CorrelationPropagationMiddleware creates middleware for correlation ID propagation
func CorrelationPropagationMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Extract trace context from headers
		headers := make(map[string]string)
		for key, values := range c.Request.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}

		traceCtx, err := ExtractTraceContextFromHeaders(headers)
		if err != nil {
			// Generate new trace context if extraction fails
			traceCtx = &DistributedTraceContext{
				TraceID: uuid.New().String(),
				SpanID:  generateSpanID(),
				Sampled: true,
			}
		}

		// Add service name to trace context
		ctx = context.WithValue(ctx, contextKey("service_name"), serviceName)

		// Update context with trace information
		ctx = ToContext(ctx, traceCtx)

		// Create child span for this service
		ctx = CreateChildSpan(ctx, c.Request.URL.Path)

		// Inject trace context back into response headers
		injectedHeaders := InjectTraceContextToHeaders(traceCtx)
		for key, value := range injectedHeaders {
			c.Header(key, value)
		}

		// Update request context
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
