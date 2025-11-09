package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/pkg/auth"
	"erpgo/pkg/cache"
)

// Logger creates a logging middleware with zerolog
func Logger(logger zerolog.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return ""
	})
}

// Recovery creates a recovery middleware with zerolog
func Recovery(logger zerolog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Error().Interface("panic", recovered).Str("path", c.Request.URL.Path).Msg("Request panic")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
			"code":  "INTERNAL_ERROR",
		})
	})
}

// CORS creates a CORS middleware
func CORS(origins, methods, headers []string) gin.HandlerFunc {
	return auth.CORSMiddleware(origins, methods, headers)
}

// RequestID creates a request ID middleware
func RequestID() gin.HandlerFunc {
	return auth.RequestIDMiddleware()
}

// RateLimit creates a rate limiting middleware (basic version for compatibility)
func RateLimit(rps, burst int, cache cache.Cache) gin.HandlerFunc {
	config := DefaultRateLimitConfig()
	config.RequestsPerSecond = rps
	config.Burst = burst
	return createRateLimitWithConfig(config, cache, zerolog.Nop())
}

// RateLimitWithConfig creates a rate limiting middleware with custom configuration
func RateLimitWithConfig(config RateLimitConfig, cache cache.Cache, logger zerolog.Logger) gin.HandlerFunc {
	return createRateLimitWithConfig(config, cache, logger)
}

// Auth creates an authentication middleware
func Auth(jwtService *auth.JWTService) gin.HandlerFunc {
	return auth.AuthMiddleware(jwtService)
}

// OptionalAuth creates an optional authentication middleware
func OptionalAuth(jwtService *auth.JWTService) gin.HandlerFunc {
	return auth.OptionalAuthMiddleware(jwtService)
}

// RequireRoles creates a role-based authorization middleware
func RequireRoles(roles ...string) gin.HandlerFunc {
	return auth.RequireRoles(roles...)
}

// RequireRole creates a single role authorization middleware
func RequireRole(role string) gin.HandlerFunc {
	return auth.RequireRole(role)
}

// RequireAllRoles creates a middleware that requires all specified roles
func RequireAllRoles(roles ...string) gin.HandlerFunc {
	return auth.RequireAllRoles(roles...)
}

// SecurityHeaders creates a security headers middleware
func SecurityHeaders() gin.HandlerFunc {
	return auth.SecurityHeadersMiddleware()
}

// AuditLoggingWithConfig creates an audit logging middleware with custom configuration
func AuditLoggingWithConfig(config AuditConfig, cache cache.Cache, logger zerolog.Logger) gin.HandlerFunc {
	return AuditLogging(config, cache, logger)
}

// Helper functions to avoid name conflicts
func createRateLimitWithConfig(config RateLimitConfig, cacheInterface cache.Cache, logger zerolog.Logger) gin.HandlerFunc {
	return RateLimitWithConfigFull(config, cacheInterface, logger)
}

