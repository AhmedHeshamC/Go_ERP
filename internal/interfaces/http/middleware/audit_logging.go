package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/pkg/audit"
	"erpgo/pkg/cache"
)

// AuditConfig holds configuration for audit logging middleware
type AuditConfig struct {
	Enabled           bool          `json:"enabled"`
	LogAllRequests    bool          `json:"log_all_requests"`
	LogSensitiveData  bool          `json:"log_sensitive_data"`
	ExcludePaths      []string      `json:"exclude_paths"`
	CacheTTL          time.Duration `json:"cache_ttl"`
}

// DefaultAuditConfig returns default audit configuration
func DefaultAuditConfig() AuditConfig {
	return AuditConfig{
		Enabled:          true,
		LogAllRequests:   false,
		LogSensitiveData: false,
		ExcludePaths: []string{
			"/health",
			"/metrics",
			"/api/v1/health",
		},
		CacheTTL: 5 * time.Minute,
	}
}

// Auditor wraps the audit logger for middleware use
type Auditor struct {
	config      AuditConfig
	auditLogger audit.AuditLogger
	cache       cache.Cache
	logger      zerolog.Logger
}

// NewAuditor creates a new auditor for middleware
func NewAuditor(config AuditConfig, cache cache.Cache, logger zerolog.Logger) *Auditor {
	return &Auditor{
		config: config,
		cache:  cache,
		logger: logger,
	}
}

// SetAuditLogger sets the audit logger (useful for lazy initialization)
func (a *Auditor) SetAuditLogger(auditLogger audit.AuditLogger) {
	a.auditLogger = auditLogger
}

// Middleware returns the audit logging middleware
func (a *Auditor) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !a.config.Enabled {
			c.Next()
			return
		}

		// Skip excluded paths
		for _, path := range a.config.ExcludePaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		// Process request
		c.Next()

		// Log audit event if audit logger is available
		if a.auditLogger != nil {
			a.logAuditEvent(c)
		}
	}
}

// logAuditEvent logs an audit event for the request
func (a *Auditor) logAuditEvent(c *gin.Context) {
	// Only audit if configured or if it's a sensitive endpoint
	if !a.config.LogAllRequests && !a.isSensitiveEndpoint(c) {
		return
	}

	// Get user ID from context if available
	var userID *string
	if uid, exists := c.Get("user_id"); exists {
		if uidStr, ok := uid.(string); ok {
			userID = &uidStr
		}
	}

	// Create audit event based on request
	event := &audit.AuditEvent{
		EventType:  a.determineEventType(c),
		Action:     c.Request.Method + " " + c.Request.URL.Path,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
		Success:    c.Writer.Status() >= 200 && c.Writer.Status() < 400,
		ResourceID: c.Param("id"),
	}

	// Add user ID if available
	if userID != nil {
		// Note: This is a simplified version. In production, parse UUID properly
		event.Details = map[string]interface{}{
			"user_id": *userID,
		}
	}

	// Log the event
	if err := a.auditLogger.LogEvent(c.Request.Context(), event); err != nil {
		a.logger.Error().Err(err).
			Str("path", c.Request.URL.Path).
			Msg("Failed to log audit event")
	}
}

// isSensitiveEndpoint checks if the endpoint is sensitive
func (a *Auditor) isSensitiveEndpoint(c *gin.Context) bool {
	path := c.Request.URL.Path
	method := c.Request.Method

	// Check for sensitive patterns
	sensitivePatterns := []string{
		"/api/v1/users",
		"/api/v1/auth",
		"/api/v1/roles",
		"/api/v1/permissions",
	}

	for _, pattern := range sensitivePatterns {
		if len(path) >= len(pattern) && path[:len(pattern)] == pattern {
			return true
		}
	}

	// All write operations are considered sensitive
	return method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE"
}

// determineEventType determines the audit event type based on the request
func (a *Auditor) determineEventType(c *gin.Context) audit.EventType {
	path := c.Request.URL.Path
	method := c.Request.Method

	// Check for specific event types
	if path == "/api/v1/auth/login" {
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			return audit.EventTypeLogin
		}
		return audit.EventTypeLoginFailed
	}

	if path == "/api/v1/auth/logout" {
		return audit.EventTypeLogout
	}

	// Check for data operations
	switch method {
	case "GET":
		return audit.EventTypeDataAccess
	case "POST", "PUT", "PATCH":
		return audit.EventTypeDataModification
	case "DELETE":
		return audit.EventTypeDataDeletion
	default:
		return audit.EventTypeSecurityEvent
	}
}

// AuditLogging creates an audit logging middleware
func AuditLogging(config AuditConfig, cache cache.Cache, logger zerolog.Logger) gin.HandlerFunc {
	auditor := NewAuditor(config, cache, logger)
	return auditor.Middleware()
}
