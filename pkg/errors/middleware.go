package errors

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// ErrorReportingMiddleware provides automatic error reporting for HTTP requests
type ErrorReportingMiddleware struct {
	reporter  *Reporter
	logger    *zerolog.Logger
	config    *MiddlewareConfig
}

// MiddlewareConfig holds the error reporting middleware configuration
type MiddlewareConfig struct {
	// Reporting settings
	ReportPanic        bool `json:"report_panic"`
	Report5xxErrors    bool `json:"report_5xx_errors"`
	Report4xxErrors    bool `json:"report_4xx_errors"`
	ReportValidationErrors bool `json:"report_validation_errors"`

	// Context extraction
	ExtractUserID       bool     `json:"extract_user_id"`
	ExtractRequestID    bool     `json:"extract_request_id"`
	ExtractHeaders      []string `json:"extract_headers"`

	// Request body settings
	IncludeRequestBody bool   `json:"include_request_body"`
	MaxRequestBodySize int64  `json:"max_request_body_size"`

	// Performance settings
	SlowRequestThreshold time.Duration `json:"slow_request_threshold"`
	ReportSlowRequests   bool          `json:"report_slow_requests"`

	// Filter settings
	IgnorePaths         []string `json:"ignore_paths"`
	IgnoreUserAgents     []string `json:"ignore_user_agents"`

	// Custom data extraction
	CustomDataExtractor func(*gin.Context) map[string]interface{} `json:"-"`
}

// DefaultMiddlewareConfig returns a default middleware configuration
func DefaultMiddlewareConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		ReportPanic:          true,
		Report5xxErrors:      true,
		Report4xxErrors:      false,
		ReportValidationErrors: false,
		ExtractUserID:        true,
		ExtractRequestID:     true,
		ExtractHeaders:       []string{},
		IncludeRequestBody:   false,
		MaxRequestBodySize:   1024 * 32, // 32KB
		SlowRequestThreshold: 5 * time.Second,
		ReportSlowRequests:   true,
		IgnorePaths:          []string{
			"/health",
			"/metrics",
			"/ready",
			"/live",
		},
		IgnoreUserAgents:     []string{},
		CustomDataExtractor:  nil,
	}
}

// NewErrorReportingMiddleware creates a new error reporting middleware
func NewErrorReportingMiddleware(reporter *Reporter, logger *zerolog.Logger, config *MiddlewareConfig) *ErrorReportingMiddleware {
	if config == nil {
		config = DefaultMiddlewareConfig()
	}

	if logger == nil {
		nopLogger := zerolog.Nop()
		logger = &nopLogger
	}

	return &ErrorReportingMiddleware{
		reporter: reporter,
		logger:   logger,
		config:   config,
	}
}

// Middleware returns the Gin middleware function
func (m *ErrorReportingMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

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

		// Set up panic recovery
		if m.config.ReportPanic {
			defer m.handlePanic(c)
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Check for slow requests
		if m.config.ReportSlowRequests && duration > m.config.SlowRequestThreshold {
			m.reportSlowRequest(c, duration)
		}

		// Check for errors in response
		statusCode := c.Writer.Status()
		m.reportHTTPError(c, statusCode, duration)
	}
}

// handlePanic handles panic recovery
func (m *ErrorReportingMiddleware) handlePanic(c *gin.Context) {
	if recovered := recover(); recovered != nil {
		// Capture stack trace
		stack := debug.Stack()

		// Create error context
		errorContext := m.createErrorContext(c)

		// Extract additional data
		customData := m.extractCustomData(c)

		// Report panic with custom data
		m.reporter.ReportPanic(c.Request.Context(), recovered, stack, errorContext)
		_ = customData // Use the variable to avoid unused variable warning

		// Add request ID to response headers if available
		if errorContext.RequestID != "" {
			c.Header("X-Request-ID", errorContext.RequestID)
		}

		// Log panic
		m.logger.Error().
			Interface("panic", recovered).
			Str("stack", string(stack)).
			Str("request_id", errorContext.RequestID).
			Str("correlation_id", errorContext.CorrelationID).
			Msg("Panic recovered in HTTP request")

		// Respond with error
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_server_error",
			"message": "An internal error occurred",
			"request_id": errorContext.RequestID,
		})
		c.Abort()
	}
}

// reportHTTPError reports HTTP errors
func (m *ErrorReportingMiddleware) reportHTTPError(c *gin.Context, statusCode int, duration time.Duration) {
	// Determine if error should be reported
	shouldReport := false
	var errorType ErrorType
	var message string

	switch {
	case statusCode >= 500:
		shouldReport = m.config.Report5xxErrors
		errorType = ErrorTypeSystem
		message = fmt.Sprintf("HTTP %d: %s", statusCode, http.StatusText(statusCode))
	case statusCode >= 400:
		shouldReport = m.config.Report4xxErrors
		if statusCode == 400 {
			shouldReport = shouldReport || m.config.ReportValidationErrors
			errorType = ErrorTypeValidation
		} else if statusCode == 401 {
			errorType = ErrorTypeAuthentication
		} else if statusCode == 403 {
			errorType = ErrorTypeAuthorization
		} else if statusCode == 429 {
			errorType = ErrorTypeRateLimit
		} else {
			errorType = ErrorTypeBusiness
		}
		message = fmt.Sprintf("HTTP %d: %s", statusCode, http.StatusText(statusCode))
	}

	if !shouldReport {
		return
	}

	// Create error context
	errorContext := m.createErrorContext(c)

	// Extract additional data
	customData := m.extractCustomData(c)
	customData["status_code"] = statusCode
	customData["duration_ms"] = duration.Milliseconds()

	// Create error
	err := fmt.Errorf("HTTP %d error: %s", statusCode, http.StatusText(statusCode))

	// Determine severity
	severity := m.mapStatusCodeToSeverity(statusCode)

	// Report error
	m.reporter.Report(c.Request.Context(), err, severity, errorType, message, errorContext, customData)
}

// reportSlowRequest reports slow requests
func (m *ErrorReportingMiddleware) reportSlowRequest(c *gin.Context, duration time.Duration) {
	errorContext := m.createErrorContext(c)

	customData := m.extractCustomData(c)
	customData["duration_ms"] = duration.Milliseconds()
	customData["threshold_ms"] = m.config.SlowRequestThreshold.Milliseconds()

	message := fmt.Sprintf("Slow request: %s %s took %v", c.Request.Method, c.Request.URL.Path, duration)

	err := fmt.Errorf("slow request: %v", duration)

	m.reporter.Report(c.Request.Context(), err, SeverityWarning, ErrorTypePerformance, message, errorContext, customData)
}

// createErrorContext creates error context from Gin context
func (m *ErrorReportingMiddleware) createErrorContext(c *gin.Context) *Context {
	errorContext := &Context{
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Tags:      make(map[string]string),
		Extra:     make(map[string]interface{}),
	}

	// Extract request ID
	if m.config.ExtractRequestID {
		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			errorContext.RequestID = requestID
		} else {
			errorContext.RequestID = c.GetString("request_id")
		}
	}

	// Extract correlation ID
	if correlationID := c.GetHeader("X-Correlation-ID"); correlationID != "" {
		errorContext.CorrelationID = correlationID
	} else {
		errorContext.CorrelationID = c.GetString("correlation_id")
	}

	// Extract user ID
	if m.config.ExtractUserID {
		if userID := c.GetString("user_id"); userID != "" {
			errorContext.UserID = userID
		} else if userID, exists := c.Get("user_id"); exists {
			if id, ok := userID.(string); ok {
				errorContext.UserID = id
			}
		}
	}

	// Extract trace/span IDs
	if traceID := c.GetHeader("X-Trace-ID"); traceID != "" {
		errorContext.TraceID = traceID
	}
	if spanID := c.GetHeader("X-Span-ID"); spanID != "" {
		errorContext.SpanID = spanID
	}

	// Extract additional headers
	for _, header := range m.config.ExtractHeaders {
		if value := c.GetHeader(header); value != "" {
			errorContext.Extra[header] = value
		}
	}

	// Add request information
	errorContext.Extra["method"] = c.Request.Method
	errorContext.Extra["path"] = c.Request.URL.Path
	errorContext.Extra["query"] = c.Request.URL.RawQuery
	errorContext.Extra["proto"] = c.Request.Proto
	errorContext.Extra["content_length"] = c.Request.ContentLength

	// Add response information
	errorContext.Extra["response_status"] = c.Writer.Status()
	errorContext.Extra["response_size"] = c.Writer.Size()

	return errorContext
}

// extractCustomData extracts custom data from the request
func (m *ErrorReportingMiddleware) extractCustomData(c *gin.Context) map[string]interface{} {
	customData := make(map[string]interface{})

	// Include request body if configured
	if m.config.IncludeRequestBody && c.Request.ContentLength > 0 &&
	   c.Request.ContentLength <= m.config.MaxRequestBodySize {

		// Note: In a real implementation, you'd need to capture the body before
		// it's consumed by other middleware. This is a simplified version.
		customData["has_request_body"] = true
		customData["request_body_size"] = c.Request.ContentLength
	}

	// Call custom data extractor if provided
	if m.config.CustomDataExtractor != nil {
		extracted := m.config.CustomDataExtractor(c)
		for key, value := range extracted {
			customData[key] = value
		}
	}

	// Add Gin-specific information
	if errors := c.Errors; len(errors) > 0 {
		customData["gin_errors"] = errors.String()
	}

	if keys := c.Keys; len(keys) > 0 {
		// Only include safe keys (exclude sensitive data)
		safeKeys := make(map[string]interface{})
		for key, value := range keys {
			if !m.isSensitiveKey(key) {
				safeKeys[key] = value
			}
		}
		if len(safeKeys) > 0 {
			customData["gin_keys"] = safeKeys
		}
	}

	return customData
}

// isSensitiveKey checks if a key might contain sensitive information
func (m *ErrorReportingMiddleware) isSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)
	sensitivePatterns := []string{
		"password", "token", "secret", "key", "auth",
		"credential", "session", "cookie", "header",
		"authorization", "bearer", "basic",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerKey, pattern) {
			return true
		}
	}
	return false
}

// shouldIgnorePath checks if a path should be ignored
func (m *ErrorReportingMiddleware) shouldIgnorePath(path string) bool {
	for _, ignorePath := range m.config.IgnorePaths {
		if strings.HasPrefix(path, ignorePath) {
			return true
		}
	}
	return false
}

// shouldIgnoreUserAgent checks if a user agent should be ignored
func (m *ErrorReportingMiddleware) shouldIgnoreUserAgent(userAgent string) bool {
	for _, ignoreUA := range m.config.IgnoreUserAgents {
		if strings.Contains(strings.ToLower(userAgent), strings.ToLower(ignoreUA)) {
			return true
		}
	}
	return false
}

// mapStatusCodeToSeverity maps HTTP status codes to severity levels
func (m *ErrorReportingMiddleware) mapStatusCodeToSeverity(statusCode int) Severity {
	switch {
	case statusCode >= 500:
		return SeverityError
	case statusCode >= 400:
		return SeverityWarning
	default:
		return SeverityInfo
	}
}

// GetStats returns middleware statistics
func (m *ErrorReportingMiddleware) GetStats() map[string]interface{} {
	if m.reporter != nil {
		return m.reporter.GetStats()
	}
	return map[string]interface{}{
		"reporter_initialized": false,
	}
}

// Convenience functions for creating middleware with default configurations

// ErrorReporting creates an error reporting middleware with default configuration
func ErrorReporting(reporter *Reporter, logger *zerolog.Logger) gin.HandlerFunc {
	middleware := NewErrorReportingMiddleware(reporter, logger, DefaultMiddlewareConfig())
	return middleware.Middleware()
}

// DevelopmentErrorReporting creates an error reporting middleware for development
func DevelopmentErrorReporting(reporter *Reporter, logger *zerolog.Logger) gin.HandlerFunc {
	config := &MiddlewareConfig{
		ReportPanic:            true,
		Report5xxErrors:        true,
		Report4xxErrors:        true,
		ReportValidationErrors: true,
		ExtractUserID:          true,
		ExtractRequestID:       true,
		IncludeRequestBody:     true,
		MaxRequestBodySize:      1024 * 64, // 64KB
		SlowRequestThreshold:   1 * time.Second,
		ReportSlowRequests:     true,
		IgnorePaths:            []string{"/health"},
		IgnoreUserAgents:       []string{},
	}

	middleware := NewErrorReportingMiddleware(reporter, logger, config)
	return middleware.Middleware()
}

// ProductionErrorReporting creates an error reporting middleware for production
func ProductionErrorReporting(reporter *Reporter, logger *zerolog.Logger) gin.HandlerFunc {
	config := &MiddlewareConfig{
		ReportPanic:            true,
		Report5xxErrors:        true,
		Report4xxErrors:        false,
		ReportValidationErrors: false,
		ExtractUserID:          false, // Don't extract user data in production
		ExtractRequestID:       true,
		IncludeRequestBody:     false,
		SlowRequestThreshold:   5 * time.Second,
		ReportSlowRequests:     false, // Usually handled by APM
		IgnorePaths:            []string{"/health", "/metrics", "/ready", "/live"},
		IgnoreUserAgents:       []string{
			"HealthChecker", "kube-probe", "UptimeRobot",
		},
	}

	middleware := NewErrorReportingMiddleware(reporter, logger, config)
	return middleware.Middleware()
}

// Helper functions for Gin context

// ReportError reports an error from Gin context
func ReportError(c *gin.Context, err error, errorType ErrorType, message string) {
	if globalReporter != nil {
		errorContext := &Context{
			RequestID:     c.GetHeader("X-Request-ID"),
			CorrelationID: c.GetHeader("X-Correlation-ID"),
			IPAddress:     c.ClientIP(),
			UserAgent:     c.GetHeader("User-Agent"),
		}

		// Extract user ID if available
		if userID := c.GetString("user_id"); userID != "" {
			errorContext.UserID = userID
		}

		globalReporter.Report(c.Request.Context(), err, SeverityError, errorType, message, errorContext, nil)
	}
}

// ReportValidationError reports a validation error from Gin context
func ReportValidationError(c *gin.Context, err error, message string) {
	if globalReporter != nil {
		errorContext := &Context{
			RequestID:     c.GetHeader("X-Request-ID"),
			CorrelationID: c.GetHeader("X-Correlation-ID"),
			IPAddress:     c.ClientIP(),
			UserAgent:     c.GetHeader("User-Agent"),
		}

		if userID := c.GetString("user_id"); userID != "" {
			errorContext.UserID = userID
		}

		globalReporter.Report(c.Request.Context(), err, SeverityWarning, ErrorTypeValidation, message, errorContext, nil)
	}
}

// ReportSecurityError reports a security-related error from Gin context
func ReportSecurityError(c *gin.Context, err error, message string) {
	if globalReporter != nil {
		errorContext := &Context{
			RequestID:     c.GetHeader("X-Request-ID"),
			CorrelationID: c.GetHeader("X-Correlation-ID"),
			IPAddress:     c.ClientIP(),
			UserAgent:     c.GetHeader("User-Agent"),
		}

		if userID := c.GetString("user_id"); userID != "" {
			errorContext.UserID = userID
		}

		// Add security-specific data
		customData := map[string]interface{}{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"query":  c.Request.URL.RawQuery,
		}

		globalReporter.Report(c.Request.Context(), err, SeverityError, ErrorTypeSecurity, message, errorContext, customData)
	}
}