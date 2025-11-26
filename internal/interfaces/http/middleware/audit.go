package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/pkg/audit"
	"erpgo/pkg/auth"
)

// AuditMiddleware creates middleware for auditing sensitive data access
func AuditMiddleware(auditLogger audit.AuditLogger, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process the request first
		c.Next()

		// Only audit successful requests to sensitive endpoints
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			// Check if this is a sensitive endpoint that should be audited
			if shouldAudit(c.Request.Method, c.FullPath()) {
				// Get user ID from context
				userID, exists := auth.GetCurrentUserID(c)
				if !exists {
					// Skip audit if user is not authenticated
					return
				}

				// Extract resource ID from path parameters
				resourceID := extractResourceID(c)

				// Create audit event
				auditEvent := audit.NewDataAccessEvent(
					userID,
					getResourceType(c.FullPath()),
					resourceID,
					c.ClientIP(),
				)

				// Log the audit event
				if err := auditLogger.LogEvent(c.Request.Context(), auditEvent); err != nil {
					logger.Error().Err(err).
						Str("user_id", userID.String()).
						Str("path", c.FullPath()).
						Msg("Failed to log audit event for data access")
				}
			}
		}
	}
}

// shouldAudit determines if a request should be audited based on method and path
func shouldAudit(method, path string) bool {
	// Audit GET requests to sensitive resources
	if method == "GET" {
		sensitivePatterns := []string{
			"/api/v1/users/:id",
			"/api/v1/users/:id/roles",
			"/api/v1/customers/:id",
			"/api/v1/orders/:id",
		}

		for _, pattern := range sensitivePatterns {
			if matchesPattern(path, pattern) {
				return true
			}
		}
	}

	// Audit all PUT, PATCH, DELETE requests to user data
	if method == "PUT" || method == "PATCH" || method == "DELETE" {
		modificationPatterns := []string{
			"/api/v1/users/:id",
			"/api/v1/customers/:id",
			"/api/v1/orders/:id",
		}

		for _, pattern := range modificationPatterns {
			if matchesPattern(path, pattern) {
				return true
			}
		}
	}

	return false
}

// matchesPattern checks if a path matches a pattern with :param placeholders
func matchesPattern(path, pattern string) bool {
	// Simple pattern matching - in production, use a proper router pattern matcher
	// For now, just check if the base path matches
	return path == pattern
}

// extractResourceID extracts the resource ID from path parameters
func extractResourceID(c *gin.Context) string {
	// Try common parameter names
	if id := c.Param("id"); id != "" {
		return id
	}
	if id := c.Param("userId"); id != "" {
		return id
	}
	if id := c.Param("customerId"); id != "" {
		return id
	}
	if id := c.Param("orderId"); id != "" {
		return id
	}
	return ""
}

// getResourceType determines the resource type from the path
func getResourceType(path string) string {
	if matchesPattern(path, "/api/v1/users/:id") {
		return "user"
	}
	if matchesPattern(path, "/api/v1/customers/:id") {
		return "customer"
	}
	if matchesPattern(path, "/api/v1/orders/:id") {
		return "order"
	}
	return "unknown"
}
