package timeout

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Config holds the timeout middleware configuration
type Config struct {
	// Default timeout for all requests
	DefaultTimeout time.Duration `json:"default_timeout"`

	// Timeouts for specific HTTP methods
	MethodTimeouts map[string]time.Duration `json:"method_timeouts"`

	// Timeouts for specific URL patterns
	PathTimeouts map[string]time.Duration `json:"path_timeouts"`

	// Custom message for timeout responses
	TimeoutMessage string `json:"timeout_message"`

	// Custom status code for timeout responses
	TimeoutStatusCode int `json:"timeout_status_code"`

	// Whether to log timeout events
	LogTimeouts bool `json:"log_timeouts"`

	// Whether to include request details in timeout response
	IncludeRequestDetails bool `json:"include_request_details"`

	// Response headers to add on timeout
	ResponseHeaders map[string]string `json:"response_headers"`
}

// DefaultConfig returns a default timeout configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultTimeout:     30 * time.Second,
		MethodTimeouts: map[string]time.Duration{
			http.MethodGet:     10 * time.Second,
			http.MethodPost:    30 * time.Second,
			http.MethodPut:     30 * time.Second,
			http.MethodPatch:   30 * time.Second,
			http.MethodDelete:  30 * time.Second,
		},
		PathTimeouts: make(map[string]time.Duration),
		TimeoutMessage:       "Request timeout",
		TimeoutStatusCode:    http.StatusRequestTimeout,
		LogTimeouts:          true,
		IncludeRequestDetails: false,
		ResponseHeaders: map[string]string{
			"X-Timeout": "true",
		},
	}
}

// Middleware provides request timeout functionality
type Middleware struct {
	config *Config
	logger *zerolog.Logger
}

// NewMiddleware creates a new timeout middleware
func NewMiddleware(config *Config, logger *zerolog.Logger) *Middleware {
	if config == nil {
		config = DefaultConfig()
	}

	if logger == nil {
		nopLogger := zerolog.Nop()
		logger = &nopLogger
	}

	return &Middleware{
		config: config,
		logger: logger,
	}
}

// Middleware returns the Gin middleware function
func (m *Middleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Determine timeout for this request
		timeout := m.getTimeoutForRequest(c)

		// Create context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Replace request context with timeout context
		c.Request = c.Request.WithContext(ctx)

		// Channel to signal completion
		finished := make(chan struct{})

		// Goroutine to continue processing the request
		go func() {
			defer func() {
				if r := recover(); r != nil {
					m.logger.Error().
						Interface("panic", r).
						Str("method", c.Request.Method).
						Str("path", c.Request.URL.Path).
						Msg("Request panicked")

					c.JSON(http.StatusInternalServerError, gin.H{
						"error":   "internal_server_error",
						"message": "Internal server error occurred",
					})
					c.Abort()
				}
				close(finished)
			}()

			c.Next()
		}()

		// Wait for request to finish or timeout
		select {
		case <-finished:
			// Request completed normally
			return
		case <-ctx.Done():
			// Request timed out
			m.handleTimeout(c, ctx)
			return
		}
	}
}

// getTimeoutForRequest determines the appropriate timeout for a request
func (m *Middleware) getTimeoutForRequest(c *gin.Context) time.Duration {
	// Check for path-specific timeout first
	for pattern, timeout := range m.config.PathTimeouts {
		if m.matchPath(c.Request.URL.Path, pattern) {
			return timeout
		}
	}

	// Check for method-specific timeout
	if timeout, exists := m.config.MethodTimeouts[c.Request.Method]; exists {
		return timeout
	}

	// Use default timeout
	return m.config.DefaultTimeout
}

// matchPath checks if a request path matches a pattern
func (m *Middleware) matchPath(path, pattern string) bool {
	// Simple exact match
	if path == pattern {
		return true
	}

	// Prefix match for patterns ending with *
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(path) >= len(prefix) && path[:len(prefix)] == prefix
	}

	return false
}

// handleTimeout handles request timeout
func (m *Middleware) handleTimeout(c *gin.Context, ctx context.Context) {
	// Log the timeout
	if m.config.LogTimeouts {
		m.logger.Warn().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Dur("timeout", m.getTimeoutForRequest(c)).
			Str("user_agent", c.Request.UserAgent()).
			Str("client_ip", c.ClientIP()).
			Msg("Request timed out")
	}

	// Add response headers
	for key, value := range m.config.ResponseHeaders {
		c.Header(key, value)
	}

	// Prepare response
	response := gin.H{
		"error":       "request_timeout",
		"message":     m.config.TimeoutMessage,
		"timeout_sec": int(m.getTimeoutForRequest(c).Seconds()),
	}

	// Include request details if configured
	if m.config.IncludeRequestDetails {
		response["request"] = gin.H{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"query":  c.Request.URL.RawQuery,
		}
	}

	// Send timeout response
	c.JSON(m.config.TimeoutStatusCode, response)
	c.Abort()
}

// WithDefaultTimeout sets the default timeout
func (m *Middleware) WithDefaultTimeout(timeout time.Duration) *Middleware {
	m.config.DefaultTimeout = timeout
	return m
}

// WithMethodTimeout sets timeout for a specific HTTP method
func (m *Middleware) WithMethodTimeout(method string, timeout time.Duration) *Middleware {
	if m.config.MethodTimeouts == nil {
		m.config.MethodTimeouts = make(map[string]time.Duration)
	}
	m.config.MethodTimeouts[method] = timeout
	return m
}

// WithPathTimeout sets timeout for a specific path pattern
func (m *Middleware) WithPathTimeout(pattern string, timeout time.Duration) *Middleware {
	if m.config.PathTimeouts == nil {
		m.config.PathTimeouts = make(map[string]time.Duration)
	}
	m.config.PathTimeouts[pattern] = timeout
	return m
}

// WithTimeoutMessage sets the timeout response message
func (m *Middleware) WithTimeoutMessage(message string) *Middleware {
	m.config.TimeoutMessage = message
	return m
}

// WithTimeoutStatusCode sets the timeout response status code
func (m *Middleware) WithTimeoutStatusCode(code int) *Middleware {
	m.config.TimeoutStatusCode = code
	return m
}

// WithLogTimeouts enables or disables timeout logging
func (m *Middleware) WithLogTimeouts(log bool) *Middleware {
	m.config.LogTimeouts = log
	return m
}

// WithResponseHeader adds a custom response header for timeouts
func (m *Middleware) WithResponseHeader(key, value string) *Middleware {
	if m.config.ResponseHeaders == nil {
		m.config.ResponseHeaders = make(map[string]string)
	}
	m.config.ResponseHeaders[key] = value
	return m
}

// GetStats returns timeout middleware statistics
func (m *Middleware) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"default_timeout":     m.config.DefaultTimeout.String(),
		"method_timeouts":     m.config.MethodTimeouts,
		"path_timeouts":       len(m.config.PathTimeouts),
		"timeout_status_code": m.config.TimeoutStatusCode,
		"log_timeouts":        m.config.LogTimeouts,
	}
}

// Common timeout configurations
var (
	// FastAPITimeout for fast API endpoints
	FastAPITimeout = 5 * time.Second

	// StandardAPITimeout for standard API endpoints
	StandardAPITimeout = 30 * time.Second

	// SlowAPITimeout for slow API endpoints (file uploads, complex queries)
	SlowAPITimeout = 5 * time.Minute

	// WebTimeout for web page requests
	WebTimeout = 10 * time.Second
)

// Predefined middleware functions for common use cases

// FastAPI returns middleware optimized for fast API endpoints
func FastAPI(logger *zerolog.Logger) gin.HandlerFunc {
	config := &Config{
		DefaultTimeout:     FastAPITimeout,
		MethodTimeouts:     map[string]time.Duration{},
		PathTimeouts:       make(map[string]time.Duration),
		TimeoutMessage:     "API request timeout",
		TimeoutStatusCode:  http.StatusRequestTimeout,
		LogTimeouts:        true,
		IncludeRequestDetails: false,
		ResponseHeaders: map[string]string{
			"X-Timeout-Type": "fast-api",
		},
	}

	return NewMiddleware(config, logger).Middleware()
}

// StandardAPI returns middleware for standard API endpoints
func StandardAPI(logger *zerolog.Logger) gin.HandlerFunc {
	config := &Config{
		DefaultTimeout:     StandardAPITimeout,
		MethodTimeouts: map[string]time.Duration{
			http.MethodGet:     10 * time.Second,
			http.MethodPost:    30 * time.Second,
			http.MethodPut:     30 * time.Second,
			http.MethodPatch:   30 * time.Second,
			http.MethodDelete:  30 * time.Second,
		},
		PathTimeouts:       make(map[string]time.Duration),
		TimeoutMessage:     "Request timeout",
		TimeoutStatusCode:  http.StatusRequestTimeout,
		LogTimeouts:        true,
		IncludeRequestDetails: false,
		ResponseHeaders: map[string]string{
			"X-Timeout-Type": "standard-api",
		},
	}

	return NewMiddleware(config, logger).Middleware()
}

// SlowAPI returns middleware for slow API endpoints
func SlowAPI(logger *zerolog.Logger) gin.HandlerFunc {
	config := &Config{
		DefaultTimeout:     SlowAPITimeout,
		MethodTimeouts: map[string]time.Duration{
			http.MethodPost:    10 * time.Minute,  // File uploads
			http.MethodPut:     10 * time.Minute,  // File updates
			http.MethodPatch:   5 * time.Minute,   // Complex updates
		},
		PathTimeouts:       make(map[string]time.Duration),
		TimeoutMessage:     "Request timeout. This may be due to a large file upload or complex processing.",
		TimeoutStatusCode:  http.StatusRequestTimeout,
		LogTimeouts:        true,
		IncludeRequestDetails: true,
		ResponseHeaders: map[string]string{
			"X-Timeout-Type": "slow-api",
		},
	}

	return NewMiddleware(config, logger).Middleware()
}

// Custom returns middleware with custom timeout configuration
func Custom(timeout time.Duration, logger *zerolog.Logger) gin.HandlerFunc {
	config := &Config{
		DefaultTimeout:     timeout,
		MethodTimeouts:     make(map[string]time.Duration),
		PathTimeouts:       make(map[string]time.Duration),
		TimeoutMessage:     fmt.Sprintf("Request timeout after %v", timeout),
		TimeoutStatusCode:  http.StatusRequestTimeout,
		LogTimeouts:        true,
		IncludeRequestDetails: false,
		ResponseHeaders: map[string]string{
			"X-Timeout-Type": "custom",
		},
	}

	return NewMiddleware(config, logger).Middleware()
}