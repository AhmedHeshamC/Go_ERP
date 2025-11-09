package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"erpgo/pkg/cache"
)

// AuditEvent represents an audit log entry
type AuditEvent struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	Level        string                 `json:"level"`
	Event        string                 `json:"event"`
	Category     string                 `json:"category"`
	UserID       string                 `json:"user_id,omitempty"`
	Username     string                 `json:"username,omitempty"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	StatusCode   int                    `json:"status_code"`
	Duration     time.Duration          `json:"duration"`
	RequestSize  int64                  `json:"request_size,omitempty"`
	ResponseSize int64                  `json:"response_size,omitempty"`
	Success      bool                   `json:"success"`
	Message      string                 `json:"message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	RequestID    string                 `json:"request_id"`
	SessionID    string                 `json:"session_id,omitempty"`
}

// AuditConfig holds configuration for audit logging
type AuditConfig struct {
	// Logging configuration
	Enabled          bool     `json:"enabled"`
	Level            string   `json:"level"` // DEBUG, INFO, WARN, ERROR
	ExcludeEndpoints []string `json:"exclude_endpoints"`
	IncludeHeaders   []string `json:"include_headers"`
	SensitiveHeaders []string `json:"sensitive_headers"` // Headers to redact

	// Storage configuration
	StoreInRedis      bool          `json:"store_in_redis"`
	StoreInFile       bool          `json:"store_in_file"`
	RetentionPeriod   time.Duration `json:"retention_period"`
	MaxLogSize        int64         `json:"max_log_size"` // bytes

	// Security events to log
	LogAuthEvents      bool `json:"log_auth_events"`
	LogFailedRequests  bool `json:"log_failed_requests"`
	LogSensitiveOps    bool `json:"log_sensitive_ops"`
	LogDataAccess      bool `json:"log_data_access"`
	LogAdminActions    bool `json:"log_admin_actions"`
	LogSecurityEvents  bool `json:"log_security_events"`

	// Performance monitoring
	LogSlowQueries     bool          `json:"log_slow_queries"`
	SlowQueryThreshold time.Duration `json:"slow_query_threshold"`

	// Key prefixes for storage
	RedisKeyPrefix string `json:"redis_key_prefix"`
	LogFilePath    string `json:"log_file_path"`
}

// DefaultAuditConfig returns a secure default configuration
func DefaultAuditConfig() AuditConfig {
	return AuditConfig{
		Enabled:     true,
		Level:       "INFO",
		ExcludeEndpoints: []string{
			"/health",
			"/metrics",
			"/favicon.ico",
			"/robots.txt",
		},
		IncludeHeaders: []string{
			"X-Request-ID",
			"X-Forwarded-For",
			"X-Real-IP",
			"User-Agent",
			"Referer",
			"Origin",
		},
		SensitiveHeaders: []string{
			"Authorization",
			"Cookie",
			"X-API-Key",
			"X-Auth-Token",
			"Set-Cookie",
		},
		StoreInRedis:      true,
		StoreInFile:       true,
		RetentionPeriod:   30 * 24 * time.Hour, // 30 days
		MaxLogSize:        100 * 1024 * 1024,  // 100MB
		LogAuthEvents:     true,
		LogFailedRequests: true,
		LogSensitiveOps:   true,
		LogDataAccess:     true,
		LogAdminActions:   true,
		LogSecurityEvents: true,
		LogSlowQueries:    true,
		SlowQueryThreshold: 1 * time.Second,
		RedisKeyPrefix:    "audit:",
		LogFilePath:       "./logs/audit.log",
	}
}

// AuditFileWriter handles file-based audit logging with rotation
type AuditFileWriter struct {
	config      AuditConfig
	file        *os.File
	mu          sync.Mutex
	lastWrite   time.Time
	currentSize int64
	logger      zerolog.Logger
}

// NewAuditFileWriter creates a new audit file writer
func NewAuditFileWriter(config AuditConfig, logger zerolog.Logger) (*AuditFileWriter, error) {
	// Ensure directory exists
	dir := filepath.Dir(config.LogFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	// Open the audit file
	file, err := os.OpenFile(config.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	// Get current file size
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat audit log file: %w", err)
	}

	return &AuditFileWriter{
		config:      config,
		file:        file,
		lastWrite:   time.Now(),
		currentSize: stat.Size(),
		logger:      logger,
	}, nil
}

// Write writes an audit event to the file
func (w *AuditFileWriter) Write(event AuditEvent) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if file rotation is needed
	if err := w.rotateIfNeeded(); err != nil {
		return err
	}

	// Serialize the event
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	// Write to file
	data = append(data, '\n') // Add newline for each log entry
	n, err := w.file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write audit event: %w", err)
	}

	w.currentSize += int64(n)
	w.lastWrite = time.Now()

	// Sync to disk for audit trail integrity
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync audit file: %w", err)
	}

	return nil
}

// rotateIfNeeded checks if file rotation is needed and performs it
func (w *AuditFileWriter) rotateIfNeeded() error {
	// Check file size limit
	if w.config.MaxLogSize > 0 && w.currentSize >= w.config.MaxLogSize {
		return w.rotateFile()
	}

	// Check time-based rotation (rotate daily)
	if time.Since(w.lastWrite) > 24*time.Hour {
		// Check if we should rotate based on date
		now := time.Now()
		if now.Day() != w.lastWrite.Day() || now.Month() != w.lastWrite.Month() || now.Year() != w.lastWrite.Year() {
			return w.rotateFile()
		}
	}

	return nil
}

// rotateFile rotates the current audit file
func (w *AuditFileWriter) rotateFile() error {
	// Close current file
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("failed to close current audit file: %w", err)
	}

	// Generate timestamp for old file
	timestamp := w.lastWrite.Format("20060102-150405")
	oldPath := fmt.Sprintf("%s.%s", w.config.LogFilePath, timestamp)

	// Rename current file
	if err := os.Rename(w.config.LogFilePath, oldPath); err != nil {
		return fmt.Errorf("failed to rotate audit file: %w", err)
	}

	// Open new file
	file, err := os.OpenFile(w.config.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new audit file: %w", err)
	}

	w.file = file
	w.currentSize = 0

	w.logger.Info().Str("old_file", oldPath).Str("new_file", w.config.LogFilePath).Msg("Audit log rotated")
	return nil
}

// Close closes the audit file writer
func (w *AuditFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Cleanup removes old audit files based on retention period
func (w *AuditFileWriter) Cleanup() error {
	if w.config.RetentionPeriod <= 0 {
		return nil // No cleanup configured
	}

	cutoffTime := time.Now().Add(-w.config.RetentionPeriod)
	dir := filepath.Dir(w.config.LogFilePath)
	baseName := filepath.Base(w.config.LogFilePath)

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and current file
		if info.IsDir() || path == w.config.LogFilePath {
			return nil
		}

		// Check if it's a rotated audit file
		if strings.HasPrefix(info.Name(), baseName+".") {
			if info.ModTime().Before(cutoffTime) {
				if removeErr := os.Remove(path); removeErr != nil {
					w.logger.Warn().Err(removeErr).Str("file", path).Msg("Failed to remove old audit file")
				} else {
					w.logger.Info().Str("file", path).Msg("Removed old audit file")
				}
			}
		}

		return nil
	})
}

// Auditor represents an audit logging system
type Auditor struct {
	config       AuditConfig
	cache        cache.Cache
	logger       zerolog.Logger
	fileWriter   *AuditFileWriter
	mu           sync.RWMutex
}

// NewAuditor creates a new audit logging system
func NewAuditor(config AuditConfig, cache cache.Cache, logger zerolog.Logger) *Auditor {
	auditor := &Auditor{
		config: config,
		cache:  cache,
		logger: logger,
	}

	// Initialize file writer if file logging is enabled
	if config.StoreInFile && config.LogFilePath != "" {
		fileWriter, err := NewAuditFileWriter(config, logger)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to initialize audit file writer")
			config.StoreInFile = false
		} else {
			auditor.fileWriter = fileWriter
		}
	}

	return auditor
}

// Middleware returns the audit logging middleware
func (a *Auditor) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !a.config.Enabled {
			c.Next()
			return
		}

		// Check if endpoint should be excluded
		if a.shouldExcludeEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Read request body for logging (if needed)
		var requestBody []byte
		if c.Request.Body != nil && a.shouldLogRequestBody(c) {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Create response writer wrapper to capture response
		wrapper := &responseWriter{
			ResponseWriter: c.Writer,
			body:          &bytes.Buffer{},
		}
		c.Writer = wrapper

		// Process request
		c.Next()

		// Determine if this is a security event that needs special logging
		event := a.determineEvent(c, wrapper.Status(), time.Since(start))

		// Create audit event
		auditEvent := AuditEvent{
			ID:           uuid.New().String(),
			Timestamp:    time.Now().UTC(),
			Level:        a.determineLevel(c, wrapper.Status(), event),
			Event:        event,
			Category:     a.determineCategory(c, event),
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			StatusCode:   wrapper.Status(),
			Duration:     time.Since(start),
			RequestSize:  c.Request.ContentLength,
			ResponseSize: int64(wrapper.body.Len()),
			Success:      wrapper.Status() < 400,
			RequestID:    requestID,
			Metadata:     make(map[string]interface{}),
		}

		// Add user information if available
		if userID, exists := c.Get("user_id"); exists {
			auditEvent.UserID = userID.(string)
		}
		if username, exists := c.Get("username"); exists {
			auditEvent.Username = username.(string)
		}

		// Add request/response details for sensitive operations
		if a.shouldLogFullDetails(c, event) {
			auditEvent.Metadata["request_headers"] = a.extractHeaders(c.Request.Header)
			auditEvent.Metadata["response_headers"] = a.extractHeaders(wrapper.Header())
			if len(requestBody) > 0 {
				auditEvent.Metadata["request_body"] = a.sanitizeRequestBody(requestBody)
			}
		}

		// Add performance metrics
		if auditEvent.Duration > a.config.SlowQueryThreshold {
			auditEvent.Metadata["slow_request"] = true
		}

		// Add security-related metadata
		if a.isSecurityEvent(c, event) {
			a.addSecurityMetadata(c, &auditEvent)
		}

		// Log the event
		a.logEvent(auditEvent)

		// Store the event if configured
		if a.config.StoreInRedis || a.config.StoreInFile {
			a.storeEvent(auditEvent)
		}
	}
}

// shouldExcludeEndpoint checks if an endpoint should be excluded from audit logging
func (a *Auditor) shouldExcludeEndpoint(path string) bool {
	for _, excluded := range a.config.ExcludeEndpoints {
		if strings.HasPrefix(path, excluded) {
			return true
		}
	}
	return false
}

// shouldLogRequestBody checks if request body should be logged
func (a *Auditor) shouldLogRequestBody(c *gin.Context) bool {
	// Don't log file uploads
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		return false
	}

	// Log body for sensitive endpoints
	sensitiveEndpoints := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/users",
		"/api/v1/admin",
	}

	for _, endpoint := range sensitiveEndpoints {
		if strings.HasPrefix(c.Request.URL.Path, endpoint) {
			return true
		}
	}

	return false
}

// determineEvent determines the audit event type
func (a *Auditor) determineEvent(c *gin.Context, statusCode int, duration time.Duration) string {
	path := c.Request.URL.Path
	method := c.Request.Method

	// Authentication events
	if strings.Contains(path, "/auth/") {
		switch {
		case strings.Contains(path, "/login"):
			if statusCode == 200 {
				return "AUTH_LOGIN_SUCCESS"
			} else {
				return "AUTH_LOGIN_FAILED"
			}
		case strings.Contains(path, "/register"):
			if statusCode == 201 {
				return "AUTH_REGISTER_SUCCESS"
			} else {
				return "AUTH_REGISTER_FAILED"
			}
		case strings.Contains(path, "/logout"):
			return "AUTH_LOGOUT"
		case strings.Contains(path, "/forgot-password"):
			return "AUTH_PASSWORD_RESET_REQUEST"
		case strings.Contains(path, "/reset-password"):
			return "AUTH_PASSWORD_RESET"
		}
	}

	// User management events
	if strings.Contains(path, "/users/") {
		switch method {
		case "POST":
			return "USER_CREATED"
		case "PUT", "PATCH":
			return "USER_UPDATED"
		case "DELETE":
			return "USER_DELETED"
		}
	}

	// Data access events
	if strings.Contains(path, "/api/v1/") {
		switch method {
		case "GET":
			return "DATA_READ"
		case "POST":
			return "DATA_CREATED"
		case "PUT", "PATCH":
			return "DATA_UPDATED"
		case "DELETE":
			return "DATA_DELETED"
		}
	}

	// Admin events
	if strings.Contains(path, "/admin/") {
		return "ADMIN_ACTION"
	}

	// Security events
	if statusCode >= 400 {
		switch {
		case statusCode == 401:
			return "SECURITY_UNAUTHORIZED"
		case statusCode == 403:
			return "SECURITY_FORBIDDEN"
		case statusCode == 429:
			return "SECURITY_RATE_LIMITED"
		case statusCode >= 500:
			return "SECURITY_SERVER_ERROR"
		default:
			return "SECURITY_CLIENT_ERROR"
		}
	}

	// Performance events
	if duration > a.config.SlowQueryThreshold {
		return "PERFORMANCE_SLOW_REQUEST"
	}

	return "REQUEST_PROCESSED"
}

// determineCategory determines the audit event category
func (a *Auditor) determineCategory(c *gin.Context, event string) string {
	if strings.HasPrefix(event, "AUTH_") {
		return "AUTHENTICATION"
	}
	if strings.HasPrefix(event, "USER_") {
		return "USER_MANAGEMENT"
	}
	if strings.HasPrefix(event, "DATA_") {
		return "DATA_ACCESS"
	}
	if strings.HasPrefix(event, "ADMIN_") {
		return "ADMINISTRATION"
	}
	if strings.HasPrefix(event, "SECURITY_") {
		return "SECURITY"
	}
	if strings.HasPrefix(event, "PERFORMANCE_") {
		return "PERFORMANCE"
	}
	return "GENERAL"
}

// determineLevel determines the log level based on the request
func (a *Auditor) determineLevel(c *gin.Context, statusCode int, event string) string {
	switch {
	case statusCode >= 500:
		return "ERROR"
	case statusCode >= 400:
		return "WARN"
	case strings.HasPrefix(event, "SECURITY_"):
		return "WARN"
	case strings.HasPrefix(event, "ADMIN_"):
		return "INFO"
	case strings.HasPrefix(event, "AUTH_"):
		return "INFO"
	default:
		return "INFO"
	}
}

// shouldLogFullDetails checks if full request/response details should be logged
func (a *Auditor) shouldLogFullDetails(c *gin.Context, event string) bool {
	if !a.config.LogSensitiveOps {
		return false
	}

	sensitiveEvents := []string{
		"AUTH_LOGIN",
		"AUTH_REGISTER",
		"USER_CREATED",
		"USER_UPDATED",
		"ADMIN_ACTION",
		"SECURITY_UNAUTHORIZED",
		"SECURITY_FORBIDDEN",
	}

	for _, sensitive := range sensitiveEvents {
		if strings.HasPrefix(event, sensitive) {
			return true
		}
	}

	return false
}

// extractHeaders extracts relevant headers for logging
func (a *Auditor) extractHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)

	for _, header := range a.config.IncludeHeaders {
		if values := headers[header]; len(values) > 0 {
			if a.isSensitiveHeader(header) {
				result[header] = "[REDACTED]"
			} else {
				result[header] = strings.Join(values, ", ")
			}
		}
	}

	return result
}

// isSensitiveHeader checks if a header contains sensitive information
func (a *Auditor) isSensitiveHeader(header string) bool {
	for _, sensitive := range a.config.SensitiveHeaders {
		if strings.EqualFold(header, sensitive) {
			return true
		}
	}
	return false
}

// sanitizeRequestBody sanitizes request body for logging
func (a *Auditor) sanitizeRequestBody(body []byte) interface{} {
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return string(body) // Return as string if not valid JSON
	}

	// Recursively sanitize sensitive fields
	return a.sanitizeData(data)
}

// sanitizeData recursively sanitizes sensitive data
func (a *Auditor) sanitizeData(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			if a.isSensitiveField(key) {
				result[key] = "[REDACTED]"
			} else {
				result[key] = a.sanitizeData(value)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = a.sanitizeData(item)
		}
		return result
	default:
		return v
	}
}

// isSensitiveField checks if a field contains sensitive information
func (a *Auditor) isSensitiveField(field string) bool {
	sensitiveFields := []string{
		"password", "passwd", "secret", "token", "key", "auth",
		"credit_card", "ssn", "social_security", "bank_account",
		"api_key", "private_key", "certificate", "credentials",
	}

	lowerField := strings.ToLower(field)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(lowerField, sensitive) {
			return true
		}
	}

	return false
}

// isSecurityEvent checks if this is a security-related event
func (a *Auditor) isSecurityEvent(c *gin.Context, event string) bool {
	return strings.HasPrefix(event, "SECURITY_") ||
		strings.HasPrefix(event, "AUTH_") ||
		c.Request.URL.Path == "/api/v1/auth/login" ||
		c.Request.URL.Path == "/api/v1/auth/register"
}

// addSecurityMetadata adds security-related metadata
func (a *Auditor) addSecurityMetadata(c *gin.Context, event *AuditEvent) {
	event.Metadata["security_event"] = true

	// Add IP geolocation if available (placeholder)
	// In production, you might use a geolocation service
	event.Metadata["ip_country"] = "UNKNOWN"

	// Add device fingerprinting (placeholder)
	event.Metadata["device_fingerprint"] = "UNKNOWN"

	// Add threat intelligence (placeholder)
	event.Metadata["threat_score"] = 0
}

// logEvent logs the audit event using zerolog
func (a *Auditor) logEvent(event AuditEvent) {
	logEvent := a.logger.WithLevel(a.getLogLevel(event.Level))

	logEvent.
		Str("audit_id", event.ID).
		Str("event", event.Event).
		Str("category", event.Category).
		Str("ip_address", event.IPAddress).
		Str("method", event.Method).
		Str("path", event.Path).
		Int("status_code", event.StatusCode).
		Dur("duration", event.Duration).
		Bool("success", event.Success).
		Str("request_id", event.RequestID)

	if event.UserID != "" {
		logEvent = logEvent.Str("user_id", event.UserID)
	}
	if event.Username != "" {
		logEvent = logEvent.Str("username", event.Username)
	}
	if event.UserAgent != "" {
		logEvent = logEvent.Str("user_agent", event.UserAgent)
	}
	if event.Message != "" {
		logEvent = logEvent.Str("message", event.Message)
	}

	// Add metadata fields
	for key, value := range event.Metadata {
		logEvent = logEvent.Interface(key, value)
	}

	logEvent.Msg("Audit event")
}

// getLogLevel converts string level to zerolog level
func (a *Auditor) getLogLevel(level string) zerolog.Level {
	switch level {
	case "DEBUG":
		return zerolog.DebugLevel
	case "INFO":
		return zerolog.InfoLevel
	case "WARN":
		return zerolog.WarnLevel
	case "ERROR":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// storeEvent stores the audit event in configured storage
func (a *Auditor) storeEvent(event AuditEvent) {
	// Store in Redis
	if a.config.StoreInRedis && a.cache != nil {
		key := a.config.RedisKeyPrefix + event.ID
		a.cache.SetJSON(context.Background(), key, event, a.config.RetentionPeriod)
	}

	// Store in file
	if a.config.StoreInFile && a.fileWriter != nil {
		if err := a.fileWriter.Write(event); err != nil {
			a.logger.Error().Err(err).Str("event_id", event.ID).Msg("Failed to write audit event to file")
		}
	}
}

// GetAuditEvents retrieves audit events from storage
func (a *Auditor) GetAuditEvents(ctx context.Context, limit int, offset int) ([]AuditEvent, error) {
	var events []AuditEvent

	// In a real implementation, you would query a database or use Redis SCAN
	// For now, this is a placeholder that returns an empty slice
	// The file-based audit trail provides the primary storage and retrieval

	a.logger.Debug().
		Int("limit", limit).
		Int("offset", offset).
		Msg("GetAuditEvents called - placeholder implementation")

	return events, nil
}

// CleanupOldAuditEvents removes audit events older than the retention period
func (a *Auditor) CleanupOldAuditEvents() {
	// Cleanup Redis-based events (if any were stored)
	// Note: Without a Keys method in the cache interface, we rely on
	// Redis TTL and file-based cleanup for audit trail management

	a.logger.Debug().Msg("CleanupOldAuditEvents called - using file-based cleanup")

	// Cleanup files
	if a.fileWriter != nil {
		if err := a.fileWriter.Cleanup(); err != nil {
			a.logger.Error().Err(err).Msg("Failed to cleanup old audit files")
		}
	}
}

// Close closes the auditor and cleans up resources
func (a *Auditor) Close() error {
	if a.fileWriter != nil {
		return a.fileWriter.Close()
	}
	return nil
}

// responseWriter is a wrapper to capture response data
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// AuditLogging creates an audit logging middleware
func AuditLogging(config AuditConfig, cache cache.Cache, logger zerolog.Logger) gin.HandlerFunc {
	auditor := NewAuditor(config, cache, logger)
	return auditor.Middleware()
}

// SecurityAuditLogging creates audit logging specifically for security events
func SecurityAuditLogging(cache cache.Cache, logger zerolog.Logger) gin.HandlerFunc {
	config := DefaultAuditConfig()
	config.LogSecurityEvents = true
	config.LogAuthEvents = true
	config.LogFailedRequests = true

	return AuditLogging(config, cache, logger)
}