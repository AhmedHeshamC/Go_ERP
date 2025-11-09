package cors

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Config holds the CORS configuration
type Config struct {
	// Allowed origins for cross-origin requests
	// Can be specific domains, wildcards, or regex patterns
	AllowedOrigins []string `json:"allowed_origins"`

	// Allowed HTTP methods
	AllowedMethods []string `json:"allowed_methods"`

	// Allowed headers
	AllowedHeaders []string `json:"allowed_headers"`

	// Exposed headers that can be accessed by the client
	ExposedHeaders []string `json:"exposed_headers"`

	// Whether credentials (cookies, authorization headers) are allowed
	AllowCredentials bool `json:"allow_credentials"`

	// Maximum age for preflight requests cache
	MaxAge time.Duration `json:"max_age"`

	// Whether to allow private network access
	AllowPrivateNetworkAccess bool `json:"allow_private_network_access"`

	// Custom preflight continue handler
	PreflightContinue bool `json:"preflight_continue"`

	// Custom preflight status code
	PreflightStatusCode int `json:"preflight_status_code"`

	// Options passthrough (continue to next middleware if OPTIONS)
	OptionsPassthrough bool `json:"options_passthrough"`

	// Log CORS events
	LogCORS bool `json:"log_cors"`

	// Custom origin validator function
	OriginValidator func(string) bool `json:"-"`

	// Dynamic origin resolver
	OriginResolver func(*gin.Context) (string, error) `json:"-"`

	// Vary header handling
	VaryOrigin bool `json:"vary_origin"`
}

// DefaultConfig returns a default CORS configuration
func DefaultConfig() *Config {
	return &Config{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowedHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"X-Requested-With",
			"X-CSRF-Token",
		},
		ExposedHeaders: []string{
			"Content-Length",
			"Content-Range",
		},
		AllowCredentials:     false,
		MaxAge:              12 * time.Hour,
		PreflightContinue:    false,
		PreflightStatusCode:  http.StatusNoContent,
		OptionsPassthrough:   false,
		LogCORS:              true,
		VaryOrigin:           true,
	}
}

// DevelopmentConfig returns a development-friendly CORS configuration
func DevelopmentConfig() *Config {
	return &Config{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8080",
			"*",
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowedHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"X-Requested-With",
			"X-CSRF-Token",
			"X-Debug",
		},
		ExposedHeaders: []string{
			"Content-Length",
			"Content-Range",
			"X-Debug",
		},
		AllowCredentials:     true,
		MaxAge:              5 * time.Minute,
		PreflightContinue:    false,
		PreflightStatusCode:  http.StatusNoContent,
		OptionsPassthrough:   false,
		LogCORS:              true,
		VaryOrigin:           true,
	}
}

// ProductionConfig returns a production-safe CORS configuration
func ProductionConfig(allowedOrigins []string) *Config {
	return &Config{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"X-Requested-With",
		},
		ExposedHeaders: []string{
			"Content-Length",
		},
		AllowCredentials:     false,
		MaxAge:              2 * time.Hour,
		PreflightContinue:    false,
		PreflightStatusCode:  http.StatusNoContent,
		OptionsPassthrough:   false,
		LogCORS:              false, // Disable verbose logging in production
		VaryOrigin:           true,
	}
}

// Middleware provides CORS functionality
type Middleware struct {
	config      *Config
	logger      *zerolog.Logger
	originCache map[string]bool
}

// NewMiddleware creates a new CORS middleware
func NewMiddleware(config *Config, logger *zerolog.Logger) *Middleware {
	if config == nil {
		config = DefaultConfig()
	}

	if logger == nil {
		nopLogger := zerolog.Nop()
		logger = &nopLogger
	}

	return &Middleware{
		config:      config,
		logger:      logger,
		originCache: make(map[string]bool),
	}
}

// Middleware returns the Gin middleware function
func (m *Middleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		method := c.Request.Method

		// Log CORS request
		if m.config.LogCORS {
			m.logger.Debug().
				Str("origin", origin).
				Str("method", method).
				Str("path", c.Request.URL.Path).
				Msg("CORS request")
		}

		// Handle preflight requests
		if method == http.MethodOptions {
			m.handlePreflight(c)
			return
		}

		// Handle actual requests
		m.handleRequest(c, origin)

		// Continue to next middleware
		c.Next()
	}
}

// handlePreflight handles CORS preflight requests
func (m *Middleware) handlePreflight(c *gin.Context) {
	origin := c.Request.Header.Get("Origin")
	requestMethod := c.Request.Header.Get("Access-Control-Request-Method")
	requestHeaders := c.Request.Header.Get("Access-Control-Request-Headers")

	// Check if origin is allowed
	if !m.isOriginAllowed(origin) {
		m.handleCORSError(c, "Origin not allowed", http.StatusForbidden)
		return
	}

	// Check if method is allowed
	if !m.isMethodAllowed(requestMethod) {
		m.handleCORSError(c, "Method not allowed", http.StatusForbidden)
		return
	}

	// Check if headers are allowed
	if requestHeaders != "" && !m.areHeadersAllowed(requestHeaders) {
		m.handleCORSError(c, "Headers not allowed", http.StatusForbidden)
		return
	}

	// Set preflight headers
	m.setPreflightHeaders(c, origin)

	// Handle preflight response
	if m.config.PreflightContinue {
		c.Next()
	} else if m.config.OptionsPassthrough {
		c.Next()
	} else {
		c.AbortWithStatus(m.config.PreflightStatusCode)
	}
}

// handleRequest handles actual CORS requests
func (m *Middleware) handleRequest(c *gin.Context, origin string) {
	if origin != "" && m.isOriginAllowed(origin) {
		m.setActualRequestHeaders(c, origin)
	}
}

// isOriginAllowed checks if an origin is allowed
func (m *Middleware) isOriginAllowed(origin string) bool {
	if origin == "" {
		return true // Same-origin requests don't need CORS
	}

	// Check cache first
	if cached, exists := m.originCache[origin]; exists {
		return cached
	}

	allowed := false

	// Check custom origin validator
	if m.config.OriginValidator != nil {
		allowed = m.config.OriginValidator(origin)
	} else {
		// Check against allowed origins
		for _, allowedOrigin := range m.config.AllowedOrigins {
			if m.matchOrigin(origin, allowedOrigin) {
				allowed = true
				break
			}
		}
	}

	// Cache the result
	m.originCache[origin] = allowed
	return allowed
}

// matchOrigin matches an origin against an allowed origin pattern
func (m *Middleware) matchOrigin(origin, allowedOrigin string) bool {
	// Exact match
	if origin == allowedOrigin {
		return true
	}

	// Wildcard match
	if allowedOrigin == "*" {
		return true
	}

	// Regex match
	if strings.HasPrefix(allowedOrigin, "regex:") {
		pattern := allowedOrigin[6:] // Remove "regex:" prefix
		matched, err := regexp.MatchString(pattern, origin)
		if err == nil {
			return matched
		}
	}

	// Prefix/suffix wildcard match
	if strings.Contains(allowedOrigin, "*") {
		// Convert to regex pattern
		pattern := regexp.QuoteMeta(allowedOrigin)
		pattern = strings.ReplaceAll(pattern, "\\*", ".*")
		pattern = "^" + pattern + "$"
		matched, err := regexp.MatchString(pattern, origin)
		if err == nil {
			return matched
		}
	}

	return false
}

// isMethodAllowed checks if a method is allowed
func (m *Middleware) isMethodAllowed(method string) bool {
	if method == "" {
		return true // No method requested
	}

	for _, allowedMethod := range m.config.AllowedMethods {
		if method == allowedMethod {
			return true
		}
	}
	return false
}

// areHeadersAllowed checks if headers are allowed
func (m *Middleware) areHeadersAllowed(requestHeaders string) bool {
	if requestHeaders == "" {
		return true // No headers requested
	}

	headers := strings.Split(requestHeaders, ",")
	for _, header := range headers {
		header = strings.TrimSpace(header)
		allowed := false

		for _, allowedHeader := range m.config.AllowedHeaders {
			if allowedHeader == "*" || allowedHeader == header {
				allowed = true
				break
			}
		}

		if !allowed {
			return false
		}
	}

	return true
}

// setPreflightHeaders sets headers for preflight requests
func (m *Middleware) setPreflightHeaders(c *gin.Context, origin string) {
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Access-Control-Allow-Methods", strings.Join(m.config.AllowedMethods, ", "))
	c.Header("Access-Control-Allow-Headers", strings.Join(m.config.AllowedHeaders, ", "))
	c.Header("Access-Control-Max-Age", fmt.Sprintf("%.0f", m.config.MaxAge.Seconds()))

	if m.config.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	if m.config.AllowPrivateNetworkAccess {
		c.Header("Access-Control-Allow-Private-Network", "true")
	}

	if len(m.config.ExposedHeaders) > 0 {
		c.Header("Access-Control-Expose-Headers", strings.Join(m.config.ExposedHeaders, ", "))
	}

	if m.config.VaryOrigin {
		c.Header("Vary", "Origin")
	}
}

// setActualRequestHeaders sets headers for actual requests
func (m *Middleware) setActualRequestHeaders(c *gin.Context, origin string) {
	c.Header("Access-Control-Allow-Origin", origin)

	if m.config.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	if len(m.config.ExposedHeaders) > 0 {
		c.Header("Access-Control-Expose-Headers", strings.Join(m.config.ExposedHeaders, ", "))
	}

	if m.config.VaryOrigin {
		c.Header("Vary", "Origin")
	}
}

// handleCORSError handles CORS errors
func (m *Middleware) handleCORSError(c *gin.Context, message string, statusCode int) {
	if m.config.LogCORS {
		m.logger.Warn().
			Str("origin", c.Request.Header.Get("Origin")).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("error", message).
			Msg("CORS error")
	}

	c.JSON(statusCode, gin.H{
		"error":   "cors_error",
		"message": message,
	})
	c.Abort()
}

// Builder methods for fluent configuration

// WithAllowedOrigins sets the allowed origins
func (m *Middleware) WithAllowedOrigins(origins []string) *Middleware {
	m.config.AllowedOrigins = origins
	return m
}

// WithAllowedMethods sets the allowed methods
func (m *Middleware) WithAllowedMethods(methods []string) *Middleware {
	m.config.AllowedMethods = methods
	return m
}

// WithAllowedHeaders sets the allowed headers
func (m *Middleware) WithAllowedHeaders(headers []string) *Middleware {
	m.config.AllowedHeaders = headers
	return m
}

// WithExposedHeaders sets the exposed headers
func (m *Middleware) WithExposedHeaders(headers []string) *Middleware {
	m.config.ExposedHeaders = headers
	return m
}

// WithAllowCredentials sets whether credentials are allowed
func (m *Middleware) WithAllowCredentials(allow bool) *Middleware {
	m.config.AllowCredentials = allow
	return m
}

// WithMaxAge sets the max age for preflight cache
func (m *Middleware) WithMaxAge(age time.Duration) *Middleware {
	m.config.MaxAge = age
	return m
}

// WithOriginValidator sets a custom origin validator
func (m *Middleware) WithOriginValidator(validator func(string) bool) *Middleware {
	m.config.OriginValidator = validator
	return m
}

// WithLogCORS sets whether to log CORS events
func (m *Middleware) WithLogCORS(log bool) *Middleware {
	m.config.LogCORS = log
	return m
}

// GetStats returns middleware statistics
func (m *Middleware) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"allowed_origins":      len(m.config.AllowedOrigins),
		"allowed_methods":      len(m.config.AllowedMethods),
		"allowed_headers":      len(m.config.AllowedHeaders),
		"exposed_headers":      len(m.config.ExposedHeaders),
		"allow_credentials":    m.config.AllowCredentials,
		"max_age":              m.config.MaxAge.String(),
		"origin_cache_size":    len(m.originCache),
		"log_cors":             m.config.LogCORS,
		"vary_origin":          m.config.VaryOrigin,
	}
}

// Predefined middleware functions for common use cases

// Default returns middleware with default CORS configuration
func Default(logger *zerolog.Logger) gin.HandlerFunc {
	return NewMiddleware(DefaultConfig(), logger).Middleware()
}

// Permissive returns a very permissive CORS configuration (use with caution)
func Permissive(logger *zerolog.Logger) gin.HandlerFunc {
	config := &Config{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
			http.MethodTrace,
			http.MethodConnect,
		},
		AllowedHeaders: []string{"*"},
		AllowCredentials: true,
		MaxAge:          24 * time.Hour,
		LogCORS:         true,
		VaryOrigin:      false,
	}

	return NewMiddleware(config, logger).Middleware()
}

// Restrictive returns a restrictive CORS configuration for production
func Restrictive(allowedOrigins []string, logger *zerolog.Logger) gin.HandlerFunc {
	config := &Config{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
		},
		AllowedHeaders: []string{
			"Content-Type",
			"Authorization",
		},
		AllowCredentials: false,
		MaxAge:           1 * time.Hour,
		LogCORS:          false,
		VaryOrigin:       true,
	}

	return NewMiddleware(config, logger).Middleware()
}

// Development returns middleware optimized for development environments
func Development(logger *zerolog.Logger) gin.HandlerFunc {
	return NewMiddleware(DevelopmentConfig(), logger).Middleware()
}

// Production returns middleware optimized for production environments
func Production(allowedOrigins []string, logger *zerolog.Logger) gin.HandlerFunc {
	return NewMiddleware(ProductionConfig(allowedOrigins), logger).Middleware()
}

// Custom returns middleware with custom configuration
func Custom(config *Config, logger *zerolog.Logger) gin.HandlerFunc {
	return NewMiddleware(config, logger).Middleware()
}

// Utility functions for common origin patterns

// LocalhostOrigins returns common localhost origins for development
func LocalhostOrigins() []string {
	return []string{
		"http://localhost:3000",
		"http://localhost:3001",
		"http://localhost:8080",
		"http://localhost:8000",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:8080",
		"http://127.0.0.1:8000",
	}
}

// ProductionOrigins creates a list of production origins
func ProductionOrigins(domains []string, ports []int) []string {
	var origins []string
	for _, domain := range domains {
		for _, port := range ports {
			origins = append(origins, fmt.Sprintf("https://%s:%d", domain, port))
		}
		if len(ports) == 0 {
			origins = append(origins, fmt.Sprintf("https://%s", domain))
		}
	}
	return origins
}