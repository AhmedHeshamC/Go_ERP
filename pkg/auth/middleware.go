package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"erpgo/internal/domain/users/repositories"
	"erpgo/pkg/config"
	"erpgo/pkg/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ContextKey represents the type for context keys
type ContextKey string

const (
	// UserContextKey is the key used to store user information in the request context
	UserContextKey ContextKey = "user"
	// UserIDContextKey is the key used to store user ID in the request context
	UserIDContextKey ContextKey = "user_id"
	// UserRolesContextKey is the key used to store user roles in the request context
	UserRolesContextKey ContextKey = "user_roles"
)

// UserInfo represents user information stored in the request context
type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	Username string    `json:"username"`
	Roles    []string  `json:"roles"`
}

// AuthMiddleware creates an authentication middleware
func AuthMiddleware(jwtService *JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
				"code":  "MISSING_AUTH_HEADER",
			})
			c.Abort()
			return
		}

		// Extract token from Bearer header
		token, err := ExtractTokenFromBearer(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
				"code":  "INVALID_AUTH_FORMAT",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwtService.ValidateAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
				"code":  "INVALID_TOKEN",
			})
			c.Abort()
			return
		}

		// Store user information in context
		userInfo := &UserInfo{
			ID:       claims.UserID,
			Email:    claims.Email,
			Username: claims.Username,
			Roles:    claims.Roles,
		}

		c.Set(string(UserContextKey), userInfo)
		c.Set(string(UserIDContextKey), claims.UserID.String())
		c.Set(string(UserRolesContextKey), claims.Roles)

		c.Next()
	}
}

// OptionalAuthMiddleware creates an optional authentication middleware
// It doesn't abort the request if authentication fails, but still sets user info if token is valid
func OptionalAuthMiddleware(jwtService *JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		token, err := ExtractTokenFromBearer(authHeader)
		if err != nil {
			c.Next()
			return
		}

		claims, err := jwtService.ValidateAccessToken(token)
		if err != nil {
			c.Next()
			return
		}

		userInfo := &UserInfo{
			ID:       claims.UserID,
			Email:    claims.Email,
			Username: claims.Username,
			Roles:    claims.Roles,
		}

		c.Set(string(UserContextKey), userInfo)
		c.Set(string(UserIDContextKey), claims.UserID.String())
		c.Set(string(UserRolesContextKey), claims.Roles)

		c.Next()
	}
}

// RequireRoles creates a middleware that requires specific roles
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get(string(UserRolesContextKey))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
				"code":  "NOT_AUTHENTICATED",
			})
			c.Abort()
			return
		}

		userRoleSlice, ok := userRoles.([]string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid user roles format",
				"code":  "INTERNAL_ERROR",
			})
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range roles {
			for _, userRole := range userRoleSlice {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
				"code":  "INSUFFICIENT_PERMISSIONS",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole creates a middleware that requires a specific role
func RequireRole(role string) gin.HandlerFunc {
	return RequireRoles(role)
}

// RequireAllRoles creates a middleware that requires all specified roles
func RequireAllRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get(string(UserRolesContextKey))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
				"code":  "NOT_AUTHENTICATED",
			})
			c.Abort()
			return
		}

		userRoleSlice, ok := userRoles.([]string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid user roles format",
				"code":  "INTERNAL_ERROR",
			})
			c.Abort()
			return
		}

		// Check if user has all required roles
		userRoleMap := make(map[string]bool)
		for _, userRole := range userRoleSlice {
			userRoleMap[userRole] = true
		}

		for _, requiredRole := range roles {
			if !userRoleMap[requiredRole] {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "Insufficient permissions",
					"code":  "INSUFFICIENT_PERMISSIONS",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// RequirePermission creates a middleware that requires a specific permission
// Uses database-backed permission checking
func RequirePermission(roleRepo repositories.RoleRepository, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetCurrentUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
				"code":  "NOT_AUTHENTICATED",
			})
			c.Abort()
			return
		}

		// Check if user has permission using the repository
		hasPermission, err := roleRepo.UserHasPermission(c.Request.Context(), userID, permission)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check permissions",
				"code":  "PERMISSION_CHECK_FAILED",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
				"code":  "INSUFFICIENT_PERMISSIONS",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCurrentUser retrieves the current user from the request context
func GetCurrentUser(c *gin.Context) (*UserInfo, bool) {
	user, exists := c.Get(string(UserContextKey))
	if !exists {
		return nil, false
	}

	userInfo, ok := user.(*UserInfo)
	return userInfo, ok
}

// GetCurrentUserID retrieves the current user ID from the request context
func GetCurrentUserID(c *gin.Context) (uuid.UUID, bool) {
	userIDStr, exists := c.Get(string(UserIDContextKey))
	if !exists {
		return uuid.Nil, false
	}

	userIDStrValue, ok := userIDStr.(string)
	if !ok {
		return uuid.Nil, false
	}

	userID, err := uuid.Parse(userIDStrValue)
	if err != nil {
		return uuid.Nil, false
	}

	return userID, true
}

// GetCurrentUserRoles retrieves the current user roles from the request context
func GetCurrentUserRoles(c *gin.Context) ([]string, bool) {
	roles, exists := c.Get(string(UserRolesContextKey))
	if !exists {
		return nil, false
	}

	roleSlice, ok := roles.([]string)
	return roleSlice, ok
}

// HasRole checks if the current user has a specific role
func HasRole(c *gin.Context, role string) bool {
	userRoles, exists := GetCurrentUserRoles(c)
	if !exists {
		return false
	}

	for _, userRole := range userRoles {
		if userRole == role {
			return true
		}
	}

	return false
}

// HasAnyRole checks if the current user has any of the specified roles
func HasAnyRole(c *gin.Context, roles ...string) bool {
	userRoles, exists := GetCurrentUserRoles(c)
	if !exists {
		return false
	}

	for _, requiredRole := range roles {
		for _, userRole := range userRoles {
			if userRole == requiredRole {
				return true
			}
		}
	}

	return false
}

// HasAllRoles checks if the current user has all of the specified roles
func HasAllRoles(c *gin.Context, roles ...string) bool {
	userRoles, exists := GetCurrentUserRoles(c)
	if !exists {
		return false
	}

	userRoleMap := make(map[string]bool)
	for _, userRole := range userRoles {
		userRoleMap[userRole] = true
	}

	for _, requiredRole := range roles {
		if !userRoleMap[requiredRole] {
			return false
		}
	}

	return true
}

// RequireAnyPermission creates a middleware that requires any of the specified permissions
func RequireAnyPermission(roleRepo repositories.RoleRepository, permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetCurrentUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
				"code":  "NOT_AUTHENTICATED",
			})
			c.Abort()
			return
		}

		// Check if user has any of the required permissions
		hasPermission, err := roleRepo.UserHasAnyPermission(c.Request.Context(), userID, permissions...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check permissions",
				"code":  "PERMISSION_CHECK_FAILED",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
				"code":  "INSUFFICIENT_PERMISSIONS",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAllPermissions creates a middleware that requires all of the specified permissions
func RequireAllPermissions(roleRepo repositories.RoleRepository, permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetCurrentUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
				"code":  "NOT_AUTHENTICATED",
			})
			c.Abort()
			return
		}

		// Check if user has all required permissions
		hasPermissions, err := roleRepo.UserHasAllPermissions(c.Request.Context(), userID, permissions...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check permissions",
				"code":  "PERMISSION_CHECK_FAILED",
			})
			c.Abort()
			return
		}

		if !hasPermissions {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
				"code":  "INSUFFICIENT_PERMISSIONS",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermissionForRole creates a middleware that requires a specific permission for a specific role
func RequirePermissionForRole(roleRepo repositories.RoleRepository, role, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetCurrentUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
				"code":  "NOT_AUTHENTICATED",
			})
			c.Abort()
			return
		}

		// First check if user has the required role
		userRoles, err := roleRepo.GetUserRoles(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get user roles",
				"code":  "ROLE_CHECK_FAILED",
			})
			c.Abort()
			return
		}

		hasRole := false
		for _, userRole := range userRoles {
			if userRole.Name == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Required role not found",
				"code":  "REQUIRED_ROLE_MISSING",
			})
			c.Abort()
			return
		}

		// Then check if the role has the required permission
		roleEntity, err := roleRepo.GetRoleByName(c.Request.Context(), role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get role details",
				"code":  "ROLE_FETCH_FAILED",
			})
			c.Abort()
			return
		}

		if !roleEntity.HasPermission(permission) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Role lacks required permission",
				"code":  "ROLE_PERMISSION_MISSING",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserPermissions retrieves all permissions for the current user
func GetUserPermissions(c *gin.Context, roleRepo repositories.RoleRepository) ([]string, error) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		return nil, fmt.Errorf("user not authenticated")
	}

	return roleRepo.GetUserPermissions(c.Request.Context(), userID)
}

// UserHasPermission checks if the current user has a specific permission
func UserHasPermission(c *gin.Context, roleRepo repositories.RoleRepository, permission string) (bool, error) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		return false, fmt.Errorf("user not authenticated")
	}

	return roleRepo.UserHasPermission(c.Request.Context(), userID, permission)
}

// UserHasAnyPermission checks if the current user has any of the specified permissions
func UserHasAnyPermission(c *gin.Context, roleRepo repositories.RoleRepository, permissions ...string) (bool, error) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		return false, fmt.Errorf("user not authenticated")
	}

	return roleRepo.UserHasAnyPermission(c.Request.Context(), userID, permissions...)
}

// UserHasAllPermissions checks if the current user has all of the specified permissions
func UserHasAllPermissions(c *gin.Context, roleRepo repositories.RoleRepository, permissions ...string) (bool, error) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		return false, fmt.Errorf("user not authenticated")
	}

	return roleRepo.UserHasAllPermissions(c.Request.Context(), userID, permissions...)
}

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Content Security Policy
		if gin.Mode() == gin.ReleaseMode {
			c.Header("Content-Security-Policy", "default-src 'self'")
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// CORSMiddleware creates a CORS middleware
func CORSMiddleware(allowedOrigins, allowedMethods, allowedHeaders []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware implements Redis-based rate limiting with configurable limits
func RateLimitMiddleware(cfg *config.Config, logger *zerolog.Logger) gin.HandlerFunc {
	// Create rate limiter configuration
	rlConfig := ratelimit.DefaultConfig()

	// Override with configuration from config package
	if cfg != nil && cfg.RateLimit != nil {
		rlConfig.DefaultLimit = ratelimit.RateLimit{
			RequestsPerSecond: cfg.RateLimit.RequestsPerSecond,
			BurstSize:         cfg.RateLimit.BurstSize,
		}
		rlConfig.StorageType = ratelimit.StorageType(cfg.RateLimit.StorageType)
		rlConfig.RedisAddr = cfg.GetRedisAddress()
		if cfg.Redis != nil {
			rlConfig.RedisPassword = cfg.Redis.Password
			rlConfig.RedisDB = cfg.Redis.DB
		}
		rlConfig.LogRequests = cfg.RateLimit.LogRequests
		rlConfig.LogRejections = cfg.RateLimit.LogRejections
	}

	// Create rate limiter instance
	limiter, err := ratelimit.New(rlConfig, logger)
	if err != nil {
		// If rate limiter fails to initialize, log error but don't block requests
		logger.Error().Err(err).Msg("Failed to initialize rate limiter, continuing without rate limiting")
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		// Generate rate limit key based on IP and optionally user ID
		key := c.ClientIP()

		// If user is authenticated, include user ID in rate limit key
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(string); ok {
				key = fmt.Sprintf("user:%s", uid)
			}
		}

		// Check rate limit
		if !limiter.Allow(key) {
			logger.Warn().
				Str("client_ip", c.ClientIP()).
				Str("key", key).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Msg("Rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"code":        "RATE_LIMIT_EXCEEDED",
				"retry_after": "60s", // Suggest retry after 1 minute
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitMiddlewareWithLimits creates a rate limit middleware with custom limits
func RateLimitMiddlewareWithLimits(requestsPerSecond float64, burstSize int, storageType string, redisAddr string, logger *zerolog.Logger) gin.HandlerFunc {
	rlConfig := &ratelimit.Config{
		DefaultLimit: ratelimit.RateLimit{
			RequestsPerSecond: requestsPerSecond,
			BurstSize:         burstSize,
		},
		StorageType:     ratelimit.StorageType(storageType),
		RedisAddr:       redisAddr,
		LogRequests:     true,
		LogRejections:   true,
		CleanupInterval: 5 * time.Minute,
	}

	limiter, err := ratelimit.New(rlConfig, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize custom rate limiter")
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		key := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(string); ok {
				key = fmt.Sprintf("user:%s", uid)
			}
		}

		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// LoggingMiddleware logs requests and responses
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format("02/Jan/2006:15:04:05 -0700"),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}

// RecoveryMiddleware recovers from panics and returns a 500 error
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
			"code":  "INTERNAL_ERROR",
		})
	})
}
