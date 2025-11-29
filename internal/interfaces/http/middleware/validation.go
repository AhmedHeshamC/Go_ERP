package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// ValidationConfig holds configuration for input validation
type ValidationConfig struct {
	// Max request body size in bytes
	MaxRequestBodySize int64 `json:"max_request_body_size"`

	// Allowed content types
	AllowedContentTypes []string `json:"allowed_content_types"`

	// Enable XSS protection
	EnableXSSProtection bool `json:"enable_xss_protection"`

	// Enable SQL injection protection
	EnableSQLInjectionProtection bool `json:"enable_sql_injection_protection"`

	// Maximum field lengths
	MaxFieldLengths map[string]int `json:"max_field_lengths"`

	// Required fields for specific endpoints
	RequiredFields map[string][]string `json:"required_fields"`

	// Allowed patterns for specific fields
	FieldPatterns map[string]*regexp.Regexp `json:"field_patterns"`

	// Blacklisted patterns for all fields
	BlacklistedPatterns []*regexp.Regexp `json:"blacklisted_patterns"`

	// Pagination limits
	MaxPaginationLimit     int `json:"max_pagination_limit"`
	DefaultPaginationLimit int `json:"default_pagination_limit"`
}

// ValidationError represents a field-level validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationResult represents the result of validation with field-level errors
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// DefaultValidationConfig returns a secure default configuration
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MaxRequestBodySize:           10 * 1024 * 1024, // 10MB
		AllowedContentTypes:          []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		EnableXSSProtection:          true,
		EnableSQLInjectionProtection: true,
		MaxPaginationLimit:           1000,
		DefaultPaginationLimit:       50,
		MaxFieldLengths: map[string]int{
			"name":     100,
			"email":    255,
			"username": 50,
			"password": 128,
			"phone":    20,
			"address":  500,
			"company":  100,
			"title":    200,
			"content":  10000,
		},
		RequiredFields: map[string][]string{
			"POST:/api/v1/users/register": {"email", "password", "username"},
			"POST:/api/v1/users/login":    {"email", "password"},
			"POST:/api/v1/products":       {"name", "price"},
			"POST:/api/v1/orders":         {"customer_id", "items"},
		},
		FieldPatterns: map[string]*regexp.Regexp{
			"email":    regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
			"username": regexp.MustCompile(`^[a-zA-Z0-9_-]{3,50}$`),
			"phone":    regexp.MustCompile(`^\+?[\d\s\-\(\)]{10,20}$`),
			"uuid":     regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
		},
		BlacklistedPatterns: []*regexp.Regexp{
			// XSS patterns - enhanced
			regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
			regexp.MustCompile(`(?i)javascript:[^"\s]*`),
			regexp.MustCompile(`(?i)on\w+\s*=[^"\s]*`),
			regexp.MustCompile(`(?i)<iframe[^>]*>`),
			regexp.MustCompile(`(?i)<object[^>]*>`),
			regexp.MustCompile(`(?i)<embed[^>]*>`),
			regexp.MustCompile(`(?i)<applet[^>]*>`),
			regexp.MustCompile(`(?i)<meta[^>]*>`),
			regexp.MustCompile(`(?i)<link[^>]*>`),
			regexp.MustCompile(`(?i)<style[^>]*>`),
			regexp.MustCompile(`(?i)vbscript:[^"\s]*`),
			regexp.MustCompile(`(?i)onload\s*=[^"\s]*`),
			regexp.MustCompile(`(?i)onerror\s*=[^"\s]*`),
			regexp.MustCompile(`(?i)onclick\s*=[^"\s]*`),
			regexp.MustCompile(`(?i)onmouseover\s*=[^"\s]*`),
			regexp.MustCompile(`(?i)onfocus\s*=[^"\s]*`),
			regexp.MustCompile(`(?i)onblur\s*=[^"\s]*`),
			regexp.MustCompile(`(?i)onchange\s*=[^"\s]*`),
			regexp.MustCompile(`(?i)onsubmit\s*=[^"\s]*`),
			// Expression binding attacks
			regexp.MustCompile(`(?i)expression\s*\(`),
			regexp.MustCompile(`(?i)@import`),
			regexp.MustCompile(`(?i)-moz-binding`),
			// Data URLs
			regexp.MustCompile(`(?i)data:text/html`),
			regexp.MustCompile(`(?i)data:application`),

			// SQL injection patterns - enhanced
			regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute|truncate|merge)\s+`),
			regexp.MustCompile(`(?i)(or|and)\s+\d+\s*=\s*\d+`),
			regexp.MustCompile(`(?i)(or|and)\s+['"][^'"]+['"]\s*=\s*['"][^'"]+['"]`),
			regexp.MustCompile(`(?i)(\-\-|\/\*|\*\/|;|\||&&|@@|@@variable)`),
			regexp.MustCompile(`(?i)(benchmark|sleep|waitfor|delay|pg_sleep|dbms_pipe)\s*\(`),
			regexp.MustCompile(`(?i)(information_schema|sysobjects|syscolumns|mysql\.user|pg_user)`),
			regexp.MustCompile(`(?i)(load_file|into\s+outfile|dumpfile)\s*\(`),
			regexp.MustCompile(`(?i)(concat|char|ascii|substring|length)\s*\(`),
			regexp.MustCompile(`(?i)(ascii|char|varchar|nvarchar)\s*\(`),
			regexp.MustCompile(`(?i)(cast|convert)\s*\(`),
			regexp.MustCompile(`(?i)(waitfor\s+time|waitfor\s+delay)\s*`),
			// Blind SQL injection
			regexp.MustCompile(`(?i)'\s+or\s+['"]?\d+['"]?\s*=\s*['"]?\d+['"]?`),
			regexp.MustCompile(`(?i)'\s+and\s+['"]?\d+['"]?\s*=\s*['"]?\d+['"]?`),
			regexp.MustCompile(`(?i)'\s+or\s+['"]?[^'"]+['"]?\s*=\s*['"]?[^'"]+['"]?`),
			regexp.MustCompile(`(?i)'\s+and\s+['"]?[^'"]+['"]?\s*=\s*['"]?[^'"]+['"]?`),

			// Command injection patterns - enhanced
			regexp.MustCompile(`(?i)(;|\||&|&&|\$\(|\x60|>|>>|<)[^a-zA-Z0-9\s]`),
			regexp.MustCompile(`(?i)(rm|del|format|shutdown|reboot|halt|poweroff)\s`),
			regexp.MustCompile(`(?i)(wget|curl|nc|netcat|telnet|ssh|ftp|tftp)\s+[^\s]`),
			regexp.MustCompile(`(?i)(cat|type|more|less|head|tail)\s+[^\s]`),
			regexp.MustCompile(`(?i)(ls|dir|find|locate|which|whereis)\s`),
			regexp.MustCompile(`(?i)(ps|top|kill|killall)\s`),
			regexp.MustCompile(`(?i)(ifconfig|ipconfig|netstat|arp)\s`),
			regexp.MustCompile(`(?i)(uname|whoami|id|pwd)\s`),
			regexp.MustCompile(`(?i)(/bin/|/usr/bin/|/usr/local/bin/|/sbin/|/usr/sbin/)`),

			// Path traversal patterns - enhanced
			regexp.MustCompile(`(?i)\.\.[\/\\]`),
			regexp.MustCompile(`(?i)%2e%2e[\/\\]`),
			regexp.MustCompile(`(?i)%252e%252e[\/\\]`),
			regexp.MustCompile(`(?i)\.\.%c0%af`),
			regexp.MustCompile(`(?i)\.\.%c1%9c`),
			regexp.MustCompile(`(?i)\.\.%c1%af`),
			regexp.MustCompile(`(?i)file://`),
			regexp.MustCompile(`(?i)http://[^/]*`),
			regexp.MustCompile(`(?i)https://[^/]*`),
			regexp.MustCompile(`(?i)ftp://[^/]*`),

			// Log injection patterns
			regexp.MustCompile(`[\r\n]`), // Prevent log injection via newlines
			regexp.MustCompile(`(?i)(error|exception|fatal|critical)\s*:`),

			// LDAP injection patterns
			regexp.MustCompile(`(?i)(\*|\(|\)|\\|\00)`),
			regexp.MustCompile(`(?i)(\&|\|)\([^)]*\)`),

			// NoSQL injection patterns
			regexp.MustCompile(`(?i)\$where`),
			regexp.MustCompile(`(?i)\$ne\s*:`),
			regexp.MustCompile(`(?i)\$gt\s*:`),
			regexp.MustCompile(`(?i)\$lt\s*:`),
			regexp.MustCompile(`(?i)\$in\s*:`),
			regexp.MustCompile(`(?i)\$nin\s*:`),
			regexp.MustCompile(`(?i)\$regex\s*:`),
		},
	}
}

// InputValidationMiddleware creates a middleware for input validation
func InputValidationMiddleware(config ValidationConfig, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate content type
		if !validateContentType(c, config.AllowedContentTypes) {
			logger.Warn().Str("content_type", c.GetHeader("Content-Type")).Msg("Unsupported content type")
			c.JSON(http.StatusUnsupportedMediaType, gin.H{
				"error": "Unsupported content type",
				"code":  "UNSUPPORTED_MEDIA_TYPE",
			})
			c.Abort()
			return
		}

		// Validate request body size
		if c.Request.ContentLength > config.MaxRequestBodySize {
			logger.Warn().Int64("size", c.Request.ContentLength).Int64("max_size", config.MaxRequestBodySize).Msg("Request body too large")
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body too large",
				"code":  "REQUEST_TOO_LARGE",
			})
			c.Abort()
			return
		}

		// Read and validate request body if it exists
		if c.Request.Body != nil && c.Request.ContentLength > 0 {
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to read request body")
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Failed to read request body",
					"code":  "READ_BODY_ERROR",
				})
				c.Abort()
				return
			}

			// Restore the body for subsequent handlers
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

			// Validate JSON body
			if isJSONContentType(c.GetHeader("Content-Type")) {
				if err := validateJSONBody(c, body, config, logger); err != nil {
					logger.Warn().Err(err).Msg("Invalid request body")
					c.JSON(http.StatusBadRequest, gin.H{
						"error": err.Error(),
						"code":  "VALIDATION_ERROR",
					})
					c.Abort()
					return
				}
			}
		}

		// Validate query parameters
		if err := validateQueryParams(c, config, logger); err != nil {
			logger.Warn().Err(err).Msg("Invalid query parameters")
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   err.Error(),
				"code":    "INVALID_QUERY_PARAMS",
				"details": map[string]string{"message": err.Error()},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// validateContentType checks if the content type is allowed
func validateContentType(c *gin.Context, allowedTypes []string) bool {
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		return true // Allow empty content type for GET requests
	}

	// Extract the base content type (ignore charset, boundary, etc.)
	baseType := strings.Split(contentType, ";")[0]
	baseType = strings.TrimSpace(baseType)

	for _, allowedType := range allowedTypes {
		if strings.EqualFold(baseType, allowedType) {
			return true
		}
	}

	return false
}

// isJSONContentType checks if the content type is JSON
func isJSONContentType(contentType string) bool {
	if contentType == "" {
		return false
	}

	baseType := strings.Split(contentType, ";")[0]
	baseType = strings.TrimSpace(baseType)

	return strings.EqualFold(baseType, "application/json")
}

// validateJSONBody validates JSON request body
func validateJSONBody(c *gin.Context, body []byte, config ValidationConfig, logger zerolog.Logger) error {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// Get the endpoint key for validation
	endpointKey := c.Request.Method + ":" + c.Request.URL.Path

	// Check required fields
	if requiredFields, exists := config.RequiredFields[endpointKey]; exists {
		for _, field := range requiredFields {
			if value, found := data[field]; !found || value == nil || value == "" {
				return fmt.Errorf("required field '%s' is missing or empty", field)
			}
		}
	}

	// Validate each field
	for field, value := range data {
		if err := validateField(field, value, config, logger); err != nil {
			return err
		}
	}

	return nil
}

// validateField validates a single field
func validateField(field string, value interface{}, config ValidationConfig, logger zerolog.Logger) error {
	// Convert value to string for validation
	strValue := fmt.Sprintf("%v", value)

	// Check maximum field length
	if maxLen, exists := config.MaxFieldLengths[field]; exists {
		if utf8.RuneCountInString(strValue) > maxLen {
			return fmt.Errorf("field '%s' exceeds maximum length of %d characters", field, maxLen)
		}
	}

	// Check field patterns
	if pattern, exists := config.FieldPatterns[field]; exists {
		if !pattern.MatchString(strValue) {
			return fmt.Errorf("field '%s' does not match required pattern", field)
		}
	}

	// XSS protection
	if config.EnableXSSProtection {
		for _, pattern := range config.BlacklistedPatterns {
			if pattern.MatchString(strValue) {
				logger.Warn().Str("field", field).Str("pattern", pattern.String()).Msg("XSS or injection attempt detected")
				return fmt.Errorf("field '%s' contains potentially dangerous content", field)
			}
		}
	}

	// SQL injection protection (additional checks)
	if config.EnableSQLInjectionProtection {
		if containsSQLInjectionPatterns(strValue) {
			logger.Warn().Str("field", field).Msg("SQL injection attempt detected")
			return fmt.Errorf("field '%s' contains potentially dangerous SQL patterns", field)
		}
	}

	return nil
}

// containsSQLInjectionPatterns checks for SQL injection patterns
func containsSQLInjectionPatterns(input string) bool {
	input = strings.ToLower(input)

	// Common SQL injection patterns
	patterns := []string{
		"union select",
		"union all select",
		"select * from",
		"insert into",
		"update set",
		"delete from",
		"drop table",
		"create table",
		"alter table",
		"exec (",
		"execute(",
		"sp_executesql",
		"xp_cmdshell",
		"' or '1'='1",
		"' or 1=1",
		"' or 1=1 --",
		"' or 1=1#",
		"' or 1=1/*",
		") or (1=1",
		"' or '1'='1' --",
		"' or '1'='1'/*",
		"' or '1'='1'#",
		"admin'--",
		"admin'/*",
		"' or 'x'='x",
		"' or 1=1--",
		"' or 1=1#",
		"' or 1=1/*",
		") or '1'='1--",
		") or ('1'='1--",
	}

	for _, pattern := range patterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}

	return false
}

// validateQueryParams validates query parameters
func validateQueryParams(c *gin.Context, config ValidationConfig, logger zerolog.Logger) error {
	for key, values := range c.Request.URL.Query() {
		for _, value := range values {
			if err := validateField(key, value, config, logger); err != nil {
				return fmt.Errorf("query parameter '%s': %w", key, err)
			}
		}
	}

	// Validate pagination parameters
	if err := validatePaginationParams(c, config); err != nil {
		return err
	}

	return nil
}

// validatePaginationParams validates pagination query parameters
func validatePaginationParams(c *gin.Context, config ValidationConfig) error {
	// Validate limit parameter
	if limitStr := c.Query("limit"); limitStr != "" {
		var limit int
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			return fmt.Errorf("invalid limit parameter: must be a positive integer")
		}
		if limit < 1 {
			return fmt.Errorf("invalid limit parameter: must be at least 1")
		}
		if limit > config.MaxPaginationLimit {
			return fmt.Errorf("invalid limit parameter: must not exceed %d", config.MaxPaginationLimit)
		}
	}

	// Validate page parameter
	if pageStr := c.Query("page"); pageStr != "" {
		var page int
		if _, err := fmt.Sscanf(pageStr, "%d", &page); err != nil {
			return fmt.Errorf("invalid page parameter: must be a positive integer")
		}
		if page < 1 {
			return fmt.Errorf("invalid page parameter: must be at least 1")
		}
	}

	// Validate offset parameter (alternative to page)
	if offsetStr := c.Query("offset"); offsetStr != "" {
		var offset int
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
			return fmt.Errorf("invalid offset parameter: must be a non-negative integer")
		}
		if offset < 0 {
			return fmt.Errorf("invalid offset parameter: must be at least 0")
		}
	}

	return nil
}

// ValidateStructWithDetails validates a struct and returns detailed field-level errors
func ValidateStructWithDetails(data map[string]interface{}, config ValidationConfig, logger zerolog.Logger) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	// Get the endpoint key for validation
	endpointKey := "" // This would need to be passed in or extracted from context

	// Check required fields
	if requiredFields, exists := config.RequiredFields[endpointKey]; exists {
		for _, field := range requiredFields {
			if value, found := data[field]; !found || value == nil || value == "" {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   field,
					Message: fmt.Sprintf("field '%s' is required", field),
					Code:    "REQUIRED_FIELD_MISSING",
				})
			}
		}
	}

	// Validate each field
	for field, value := range data {
		if err := validateField(field, value, config, logger); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   field,
				Message: err.Error(),
				Code:    "VALIDATION_ERROR",
			})
		}
	}

	return result
}

// SanitizeInput sanitizes input by removing potentially dangerous characters
func SanitizeInput(input string) string {
	// Remove null bytes and other dangerous control characters
	input = strings.ReplaceAll(input, "\x00", "")
	input = strings.ReplaceAll(input, "\uFFFD", "") // Replacement character

	// Remove control characters except newlines, tabs, and carriage returns
	input = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`).ReplaceAllString(input, "")

	// Remove Unicode control characters
	input = regexp.MustCompile(`[\p{C}&&[^\r\n\t]]`).ReplaceAllString(input, "")

	// Normalize whitespace
	input = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(input), " ")

	// Remove potential XSS event handlers
	input = regexp.MustCompile(`(?i)on\w+\s*=`).ReplaceAllString(input, "")

	// Remove JavaScript pseudo-protocols
	input = regexp.MustCompile(`(?i)javascript:`).ReplaceAllString(input, "")

	// Remove VBScript pseudo-protocols
	input = regexp.MustCompile(`(?i)vbscript:`).ReplaceAllString(input, "")

	// Remove potential XML external entity references
	input = regexp.MustCompile(`<!ENTITY[^>]*>`).ReplaceAllString(input, "")

	return input
}

// OWASP Security Headers Middleware
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent content type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Force HTTPS (recommended for production)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none';")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions policy
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// CSRFProtectionMiddleware provides basic CSRF protection
func CSRFProtectionMiddleware(exemptPaths []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF protection for exempted paths
		for _, path := range exemptPaths {
			if strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}

		// Only apply to state-changing methods
		if c.Request.Method != "POST" && c.Request.Method != "PUT" &&
			c.Request.Method != "DELETE" && c.Request.Method != "PATCH" {
			c.Next()
			return
		}

		// Check for CSRF token
		token := c.GetHeader("X-CSRF-Token")
		if token == "" {
			token = c.PostForm("_csrf_token")
		}

		// For API endpoints, also check Authorization header as fallback
		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				// Skip CSRF check for authenticated API requests with JWT tokens
				c.Next()
				return
			}
		}

		if token == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "CSRF token required",
				"code":  "CSRF_TOKEN_REQUIRED",
			})
			c.Abort()
			return
		}

		// In a real implementation, you would validate the token against
		// the session or stored value
		c.Next()
	}
}

// ValidateAndSanitize validates and sanitizes input data
func ValidateAndSanitize(data map[string]interface{}, config ValidationConfig, logger zerolog.Logger) (map[string]interface{}, error) {
	sanitized := make(map[string]interface{})

	for field, value := range data {
		strValue := fmt.Sprintf("%v", value)

		// Validate field
		if err := validateField(field, value, config, logger); err != nil {
			return nil, err
		}

		// Sanitize field
		sanitizedValue := SanitizeInput(strValue)

		// Try to maintain original type if possible
		switch v := value.(type) {
		case string:
			sanitized[field] = sanitizedValue
		case int, int64, float64:
			sanitized[field] = v // Keep numeric values as-is
		case bool:
			sanitized[field] = v // Keep boolean values as-is
		default:
			sanitized[field] = sanitizedValue
		}
	}

	return sanitized, nil
}
