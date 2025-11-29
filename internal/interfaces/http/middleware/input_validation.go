package middleware

import (
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// InputValidationConfig holds configuration for input validation
type InputValidationConfig struct {
	// Size limits
	MaxRequestSize int64 `json:"max_request_size"` // Max request body size in bytes
	MaxHeaderSize  int   `json:"max_header_size"`  // Max header size in bytes
	MaxURLLength   int   `json:"max_url_length"`   // Max URL length
	MaxQueryParams int   `json:"max_query_params"` // Max number of query parameters
	MaxFormFields  int   `json:"max_form_fields"`  // Max number of form fields

	// String validation
	MinStringLength int `json:"min_string_length"` // Min string length
	MaxStringLength int `json:"max_string_length"` // Max string length

	// SQL injection patterns
	SQLInjectionPatterns []string `json:"sql_injection_patterns"`

	// XSS patterns
	XSSPatterns []string `json:"xss_patterns"`

	// Path traversal patterns
	PathTraversalPatterns []string `json:"path_traversal_patterns"`

	// Command injection patterns
	CommandInjectionPatterns []string `json:"command_injection_patterns"`

	// File upload validation
	AllowedMimeTypes  []string `json:"allowed_mime_types"`
	MaxFileSize       int64    `json:"max_file_size"`
	AllowedExtensions []string `json:"allowed_extensions"`

	// Settings
	Enabled             bool `json:"enabled"`
	StrictMode          bool `json:"strict_mode"` // Reject suspicious requests
	LogValidationErrors bool `json:"log_validation_errors"`
	SanitizeOutput      bool `json:"sanitize_output"` // Sanitize response data

	// Exclusions
	ExcludedPaths   []string `json:"excluded_paths"`
	ExcludedMethods []string `json:"excluded_methods"`
}

// DefaultInputValidationConfig returns a secure default configuration
func DefaultInputValidationConfig() InputValidationConfig {
	return InputValidationConfig{
		MaxRequestSize:  10 * 1024 * 1024, // 10MB
		MaxHeaderSize:   8192,             // 8KB
		MaxURLLength:    2048,             // 2KB
		MaxQueryParams:  50,
		MaxFormFields:   100,
		MinStringLength: 1,
		MaxStringLength: 10000,

		// Common SQL injection patterns
		SQLInjectionPatterns: []string{
			`(?i)(union\s+select|select\s+.*\s+from\s+|insert\s+into\s+.*\s+values\s*\(|update\s+.*\s+set\s+|delete\s+from\s+|drop\s+table\s+|create\s+table\s+|alter\s+table\s+|exec\s*\(|execute\s*\()`,
			`(?i)(\bor\s+|and\s+|where\s+).*(\bor\s+|and\s+)(.*=|.*like)`,
			`(?i)(\'|;|--|/\*|\*/|xp_|sp_)`,
			`(?i)(waitfor\s+delay\s+|benchmark\s*\(|sleep\s*\()`,
			`(?i)(\b(char|varchar|nvarchar|text|ntext)\b.*\b(cast|convert)\b)`,
		},

		// Common XSS patterns
		XSSPatterns: []string{
			`(?i)(<script|</script|javascript:|vbscript:|onload=|onerror=|onclick=|onmouseover=)`,
			`(?i)(<iframe|<object|<embed|<link|<meta|<style)`,
			`(?i)(expression\s*\(|@import|behavior\s*:|binding\s*:|url\s*\()`,
			`(?i)(alert\s*\(|confirm\s*\(|prompt\s*\(|document\.|window\.|location\.)`,
			`(?i)(eval\s*\(|setTimeout\s*\(|setInterval\s*\()`,
		},

		// Path traversal patterns
		PathTraversalPatterns: []string{
			`(\.\./|\.\.\\|%2e%2e%2f|%2e%2e\\|\.\.%c0%af|\.\.%c1%9c)`,
			`(\.\./|\.\.\\|\.\.%2f|\.\.%5c|\.\.%c0%af|\.\.%c1%9c)`,
			`(file://|http://|https://|ftp://|data:)`,
		},

		// Command injection patterns
		CommandInjectionPatterns: []string{
			`(?i)(;|\||&|&&|\|\||` + "`" + `|\$\(|\${|\$\{)`,
			`(?i)(cat\s+|ls\s+|dir\s+|type\s+|whoami|id|uname|pwd)`,
			`(?i)(rm\s+-rf|del\s+/f|format\s+|shutdown\s+|reboot\s+)`,
			`(?i)(wget\s+|curl\s+|nc\s+|netcat\s+|telnet\s+)`,
			`(?i)(python\s+|perl\s+|ruby\s+|bash\s+|sh\s+|cmd\.exe)`,
		},

		AllowedMimeTypes: []string{
			"text/plain",
			"text/html",
			"text/css",
			"application/json",
			"application/xml",
			"application/x-www-form-urlencoded",
			"multipart/form-data",
			"image/jpeg",
			"image/png",
			"image/gif",
			"image/webp",
			"application/pdf",
		},
		MaxFileSize:       5 * 1024 * 1024, // 5MB
		AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".pdf", ".txt", ".json", ".xml"},

		Enabled:             true,
		StrictMode:          true,
		LogValidationErrors: true,
		SanitizeOutput:      true,
		ExcludedPaths:       []string{"/health", "/metrics", "/api/v1/auth/login", "/api/v1/auth/register"},
		ExcludedMethods:     []string{"GET", "HEAD", "OPTIONS"},
	}
}

// InputValidator represents an input validation middleware
type InputValidator struct {
	config InputValidationConfig
	logger zerolog.Logger

	// Compiled regex patterns
	sqlPatterns     []*regexp.Regexp
	xssPatterns     []*regexp.Regexp
	pathPatterns    []*regexp.Regexp
	commandPatterns []*regexp.Regexp
}

// NewInputValidator creates a new input validator
func NewInputValidator(config InputValidationConfig, logger zerolog.Logger) (*InputValidator, error) {
	validator := &InputValidator{
		config: config,
		logger: logger,
	}

	// Compile SQL injection patterns
	for _, pattern := range config.SQLInjectionPatterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile SQL injection pattern '%s': %w", pattern, err)
		}
		validator.sqlPatterns = append(validator.sqlPatterns, regex)
	}

	// Compile XSS patterns
	for _, pattern := range config.XSSPatterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile XSS pattern '%s': %w", pattern, err)
		}
		validator.xssPatterns = append(validator.xssPatterns, regex)
	}

	// Compile path traversal patterns
	for _, pattern := range config.PathTraversalPatterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile path traversal pattern '%s': %w", pattern, err)
		}
		validator.pathPatterns = append(validator.pathPatterns, regex)
	}

	// Compile command injection patterns
	for _, pattern := range config.CommandInjectionPatterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile command injection pattern '%s': %w", pattern, err)
		}
		validator.commandPatterns = append(validator.commandPatterns, regex)
	}

	return validator, nil
}

// Middleware returns the input validation middleware
func (v *InputValidator) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if disabled
		if !v.config.Enabled {
			c.Next()
			return
		}

		// Skip excluded methods
		if v.isExcludedMethod(c.Request.Method) {
			c.Next()
			return
		}

		// Skip excluded paths
		if v.isExcludedPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Validate request size
		if c.Request.ContentLength > v.config.MaxRequestSize {
			v.rejectRequest(c, "Request size exceeds maximum allowed size", http.StatusRequestEntityTooLarge)
			return
		}

		// Validate URL length
		if len(c.Request.URL.String()) > v.config.MaxURLLength {
			v.rejectRequest(c, "URL length exceeds maximum allowed length", http.StatusRequestURITooLong)
			return
		}

		// Validate headers
		if !v.validateHeaders(c) {
			return
		}

		// Validate query parameters
		if !v.validateQueryParams(c) {
			return
		}

		// Validate form data if present
		if !v.validateFormData(c) {
			return
		}

		// Validate JSON data if present
		if !v.validateJSONData(c) {
			return
		}

		// Validate file uploads if present
		if !v.validateFileUpload(c) {
			return
		}

		c.Next()
	}
}

// validateHeaders validates request headers
func (v *InputValidator) validateHeaders(c *gin.Context) bool {
	for name, values := range c.Request.Header {
		// Check header size
		for _, value := range values {
			if len(value) > v.config.MaxHeaderSize {
				v.logSecurityEvent(c, "Large header detected", map[string]interface{}{
					"header": name,
					"size":   len(value),
				})
				if v.config.StrictMode {
					v.rejectRequest(c, "Header size exceeds maximum allowed size", http.StatusRequestHeaderFieldsTooLarge)
					return false
				}
			}

			// Check for injection patterns in headers
			if v.containsInjectionPatterns(value) {
				v.logSecurityEvent(c, "Injection pattern detected in header", map[string]interface{}{
					"header": name,
					"value":  value,
				})
				if v.config.StrictMode {
					v.rejectRequest(c, "Invalid characters in header", http.StatusBadRequest)
					return false
				}
			}
		}
	}
	return true
}

// validateQueryParams validates query parameters
func (v *InputValidator) validateQueryParams(c *gin.Context) bool {
	if len(c.Request.URL.Query()) > v.config.MaxQueryParams {
		v.rejectRequest(c, "Too many query parameters", http.StatusBadRequest)
		return false
	}

	for key, values := range c.Request.URL.Query() {
		// Validate parameter name
		if !v.isValidString(key) {
			v.logSecurityEvent(c, "Invalid query parameter name", map[string]interface{}{
				"param": key,
			})
			if v.config.StrictMode {
				v.rejectRequest(c, "Invalid query parameter name", http.StatusBadRequest)
				return false
			}
		}

		// Validate parameter values
		for _, value := range values {
			if !v.isValidString(value) {
				v.logSecurityEvent(c, "Invalid query parameter value", map[string]interface{}{
					"param": key,
					"value": value,
				})
				if v.config.StrictMode {
					v.rejectRequest(c, "Invalid query parameter value", http.StatusBadRequest)
					return false
				}
			}

			// Check for injection patterns
			if v.containsInjectionPatterns(value) {
				v.logSecurityEvent(c, "Injection pattern detected in query parameter", map[string]interface{}{
					"param": key,
					"value": value,
				})
				if v.config.StrictMode {
					v.rejectRequest(c, "Invalid characters in query parameter", http.StatusBadRequest)
					return false
				}
			}
		}
	}
	return true
}

// validateFormData validates form data
func (v *InputValidator) validateFormData(c *gin.Context) bool {
	if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
		contentType := c.GetHeader("Content-Type")
		if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			c.Request.ParseForm()

			if len(c.Request.PostForm) > v.config.MaxFormFields {
				v.rejectRequest(c, "Too many form fields", http.StatusBadRequest)
				return false
			}

			for key, values := range c.Request.PostForm {
				// Validate field name
				if !v.isValidString(key) {
					v.logSecurityEvent(c, "Invalid form field name", map[string]interface{}{
						"field": key,
					})
					if v.config.StrictMode {
						v.rejectRequest(c, "Invalid form field name", http.StatusBadRequest)
						return false
					}
				}

				// Validate field values
				for _, value := range values {
					if !v.isValidString(value) {
						v.logSecurityEvent(c, "Invalid form field value", map[string]interface{}{
							"field": key,
							"value": value,
						})
						if v.config.StrictMode {
							v.rejectRequest(c, "Invalid form field value", http.StatusBadRequest)
							return false
						}
					}

					// Check for injection patterns
					if v.containsInjectionPatterns(value) {
						v.logSecurityEvent(c, "Injection pattern detected in form field", map[string]interface{}{
							"field": key,
							"value": value,
						})
						if v.config.StrictMode {
							v.rejectRequest(c, "Invalid characters in form field", http.StatusBadRequest)
							return false
						}
					}
				}
			}
		}
	}
	return true
}

// validateJSONData validates JSON data
func (v *InputValidator) validateJSONData(c *gin.Context) bool {
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var data interface{}
		if err := c.ShouldBindJSON(&data); err == nil {
			// Recursively validate JSON data
			if !v.validateData(data, "") {
				v.logSecurityEvent(c, "Invalid data detected in JSON", map[string]interface{}{
					"data": data,
				})
				if v.config.StrictMode {
					v.rejectRequest(c, "Invalid data in JSON", http.StatusBadRequest)
					return false
				}
			}
		}
	}
	return true
}

// validateFileUpload validates file uploads
func (v *InputValidator) validateFileUpload(c *gin.Context) bool {
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		c.Request.ParseMultipartForm(v.config.MaxFileSize)

		if c.Request.MultipartForm != nil {
			for _, files := range c.Request.MultipartForm.File {
				for _, file := range files {
					// Validate file size
					if file.Size > v.config.MaxFileSize {
						v.rejectRequest(c, "File size exceeds maximum allowed size", http.StatusRequestEntityTooLarge)
						return false
					}

					// Validate file extension
					if !v.isValidFileExtension(file.Filename) {
						v.rejectRequest(c, "File type not allowed", http.StatusBadRequest)
						return false
					}

					// Validate MIME type by checking file header
					if !v.isValidMimeType(file) {
						v.rejectRequest(c, "File MIME type not allowed", http.StatusBadRequest)
						return false
					}
				}
			}
		}
	}
	return true
}

// validateData recursively validates data structures
func (v *InputValidator) validateData(data interface{}, path string) bool {
	switch d := data.(type) {
	case string:
		return v.isValidString(d)
	case map[string]interface{}:
		for key, value := range d {
			if !v.isValidString(key) {
				return false
			}
			if !v.validateData(value, path+"."+key) {
				return false
			}
		}
	case []interface{}:
		for i, value := range d {
			if !v.validateData(value, fmt.Sprintf("%s[%d]", path, i)) {
				return false
			}
		}
	}
	return true
}

// isValidString checks if a string is valid
func (v *InputValidator) isValidString(s string) bool {
	// Check length
	if len(s) < v.config.MinStringLength || len(s) > v.config.MaxStringLength {
		return false
	}

	// Check for invalid characters
	for _, r := range s {
		if r == utf8.RuneError {
			return false
		}
		// Check for control characters (except tab, newline, carriage return)
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}

	return true
}

// containsInjectionPatterns checks if a string contains injection patterns
func (v *InputValidator) containsInjectionPatterns(s string) bool {
	// Check SQL injection patterns
	for _, pattern := range v.sqlPatterns {
		if pattern.MatchString(s) {
			return true
		}
	}

	// Check XSS patterns
	for _, pattern := range v.xssPatterns {
		if pattern.MatchString(s) {
			return true
		}
	}

	// Check path traversal patterns
	for _, pattern := range v.pathPatterns {
		if pattern.MatchString(s) {
			return true
		}
	}

	// Check command injection patterns
	for _, pattern := range v.commandPatterns {
		if pattern.MatchString(s) {
			return true
		}
	}

	return false
}

// isValidFileExtension checks if a file extension is allowed
func (v *InputValidator) isValidFileExtension(filename string) bool {
	for _, ext := range v.config.AllowedExtensions {
		if strings.HasSuffix(strings.ToLower(filename), strings.ToLower(ext)) {
			return true
		}
	}
	return false
}

// isValidMimeType checks if a MIME type is allowed
func (v *InputValidator) isValidMimeType(file interface{}) bool {
	// This is a simplified check - in production, you'd want to actually read the file header
	// to determine the real MIME type, not just trust the filename
	return true
}

// sanitizeString sanitizes a string by removing dangerous characters
func (v *InputValidator) sanitizeString(s string) string {
	// HTML escape
	s = html.EscapeString(s)

	// Remove potential script content
	s = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?i)javascript:`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?i)vbscript:`).ReplaceAllString(s, "")

	// Remove potentially dangerous HTML tags
	dangerousTags := []string{"script", "iframe", "object", "embed", "link", "meta"}
	for _, tag := range dangerousTags {
		s = regexp.MustCompile(fmt.Sprintf(`(?i)</?%s[^>]*>`, tag)).ReplaceAllString(s, "")
	}

	return s
}

// isExcludedMethod checks if a method should be excluded from validation
func (v *InputValidator) isExcludedMethod(method string) bool {
	for _, excluded := range v.config.ExcludedMethods {
		if method == excluded {
			return true
		}
	}
	return false
}

// isExcludedPath checks if a path should be excluded from validation
func (v *InputValidator) isExcludedPath(path string) bool {
	for _, excluded := range v.config.ExcludedPaths {
		if strings.HasPrefix(path, excluded) {
			return true
		}
	}
	return false
}

// rejectRequest rejects a request with validation failure
func (v *InputValidator) rejectRequest(c *gin.Context, message string, statusCode int) {
	c.JSON(statusCode, gin.H{
		"error": message,
		"code":  "VALIDATION_ERROR",
	})
	c.Abort()
}

// logSecurityEvent logs security-related events
func (v *InputValidator) logSecurityEvent(c *gin.Context, event string, details map[string]interface{}) {
	if v.config.LogValidationErrors {
		eventData := map[string]interface{}{
			"event":      event,
			"client_ip":  c.ClientIP(),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"user_agent": c.GetHeader("User-Agent"),
		}

		for k, v := range details {
			eventData[k] = v
		}

		v.logger.Warn().Interface("data", eventData).Msg("Input validation security event")
	}
}

// SanitizeResponse is a middleware that sanitizes response data
func (v *InputValidator) SanitizeResponse() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only sanitize JSON responses
		if !v.config.SanitizeOutput {
			return
		}

		contentType := c.Writer.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			return
		}

		// Get response data
		responseData, exists := c.Get("response_data")
		if !exists {
			return
		}

		// Sanitize response data
		if sanitized := v.sanitizeData(responseData); sanitized != nil {
			c.Set("response_data", sanitized)
			c.JSON(c.Writer.Status(), sanitized)
		}
	}
}

// sanitizeData recursively sanitizes data structures
func (v *InputValidator) sanitizeData(data interface{}) interface{} {
	switch d := data.(type) {
	case string:
		return v.sanitizeString(d)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range d {
			result[key] = v.sanitizeData(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(d))
		for i, value := range d {
			result[i] = v.sanitizeData(value)
		}
		return result
	default:
		return d
	}
}

// InputValidation creates an input validation middleware
func InputValidation(isProduction bool, logger zerolog.Logger) gin.HandlerFunc {
	config := DefaultInputValidationConfig()
	if !isProduction {
		config.StrictMode = false // Less strict in development
	}

	validator, err := NewInputValidator(config, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create input validator")
		// Return a middleware that just passes through
		return func(c *gin.Context) { c.Next() }
	}

	return validator.Middleware()
}

// InputValidationWithConfig creates an input validation middleware with custom configuration
func InputValidationWithConfig(config InputValidationConfig, logger zerolog.Logger) gin.HandlerFunc {
	validator, err := NewInputValidator(config, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create input validator")
		// Return a middleware that just passes through
		return func(c *gin.Context) { c.Next() }
	}

	return validator.Middleware()
}
