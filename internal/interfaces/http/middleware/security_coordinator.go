package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/pkg/auth"
	"erpgo/pkg/cache"
	"erpgo/pkg/security"
)

// SecurityCoordinator coordinates all security middleware
type SecurityCoordinator struct {
	config                     SecurityCoordinatorConfig
	logger                     zerolog.Logger
	cache                      cache.Cache
	apiKeyService              *auth.APIKeyService
	securityMonitor            *security.SecurityMonitor
	rateLimiter                *RateLimiter
	inputValidator             *InputValidator
	csrfProtection            *CSRF
	auditLogger               *Auditor
}

// SecurityCoordinatorConfig holds configuration for security coordinator
type SecurityCoordinatorConfig struct {
	// General settings
	Enabled                bool   `json:"enabled"`
	Environment           string  `json:"environment"` // "development", "staging", "production"

	// Component enablement
	EnableSecurityHeaders  bool `json:"enable_security_headers"`
	EnableRateLimiting     bool `json:"enable_rate_limiting"`
	EnableCSRFProtection  bool `json:"enable_csrf_protection"`
	EnableInputValidation bool `json:"enable_input_validation"`
	EnableAuditLogging    bool `json:"enable_audit_logging"`
	EnableAPICredentials  bool `json:"enable_api_credentials"`
	EnableSecurityMonitor bool `json:"enable_security_monitor"`

	// Security headers configuration
	SecurityHeaders SecurityConfig `json:"security_headers"`

	// Rate limiting configuration
	RateLimiting RateLimitConfig `json:"rate_limiting"`

	// CSRF configuration
	CSRF CSRFConfig `json:"csrf"`

	// Input validation configuration
	InputValidation InputValidationConfig `json:"input_validation"`

	// Audit configuration
	Audit AuditConfig `json:"audit"`

	// API key configuration
	APIKey auth.APIKeyConfig `json:"api_key"`

	// Security monitoring configuration
	SecurityMonitoring security.SecurityConfig `json:"security_monitoring"`
}

// DefaultSecurityCoordinatorConfig returns a secure default configuration
func DefaultSecurityCoordinatorConfig(environment string) SecurityCoordinatorConfig {
	return SecurityCoordinatorConfig{
		Enabled:                true,
		Environment:           environment,
		EnableSecurityHeaders:  true,
		EnableRateLimiting:     true,
		EnableCSRFProtection:  true,
		EnableInputValidation: true,
		EnableAuditLogging:    true,
		EnableAPICredentials:  true,
		EnableSecurityMonitor: true,

		SecurityHeaders:       SecurityConfig{},
		RateLimiting:         DefaultRateLimitConfig(),
		CSRF:                 DefaultCSRFConfig(),
		InputValidation:      DefaultInputValidationConfig(),
		Audit:                DefaultAuditConfig(),
		APIKey:               auth.DefaultAPIKeyConfig(),
		SecurityMonitoring:   security.DefaultSecurityConfig(),
	}
}

// NewSecurityCoordinator creates a new security coordinator
func NewSecurityCoordinator(
	config SecurityCoordinatorConfig,
	cache cache.Cache,
	logger zerolog.Logger,
) (*SecurityCoordinator, error) {
	coordinator := &SecurityCoordinator{
		config: config,
		cache:  cache,
		logger: logger,
	}

	// Initialize components if enabled
	var err error

	if config.EnableAPICredentials {
		// Create in-memory repository for API keys (replace with database repo in production)
		apiKeyRepo := auth.NewInMemoryAPIKeyRepository()
		coordinator.apiKeyService = auth.NewAPIKeyService(apiKeyRepo, cache, logger, config.APIKey)
	}

	if config.EnableSecurityMonitor {
		coordinator.securityMonitor = security.NewSecurityMonitor(config.SecurityMonitoring, cache, logger)
	}

	if config.EnableRateLimiting {
		// TODO: Fix rate limiter initialization
		// coordinator.rateLimiter = NewRateLimiter(config.RateLimiting, coordinator.cache.(*cache.RedisCache).GetClient(), logger)
	}

	if config.EnableInputValidation {
		coordinator.inputValidator, err = NewInputValidator(config.InputValidation, logger)
		if err != nil {
			return nil, err
		}
	}

	if config.EnableCSRFProtection {
		coordinator.csrfProtection = NewCSRF(config.CSRF, logger)
	}

	if config.EnableAuditLogging {
		coordinator.auditLogger = NewAuditor(config.Audit, cache, logger)
	}

	return coordinator, nil
}

// Start starts the security coordinator and its components
func (sc *SecurityCoordinator) Start() {
	if sc.config.EnableSecurityMonitor && sc.securityMonitor != nil {
		sc.securityMonitor.Start()
		sc.logger.Info().Msg("Security monitor started")
	}
}

// Stop stops the security coordinator and its components
func (sc *SecurityCoordinator) Stop() {
	if sc.config.EnableSecurityMonitor && sc.securityMonitor != nil {
		sc.securityMonitor.Stop()
		sc.logger.Info().Msg("Security monitor stopped")
	}
}

// Middleware returns the combined security middleware
func (sc *SecurityCoordinator) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if security is disabled
		if !sc.config.Enabled {
			c.Next()
			return
		}

		// Set security headers first
		if sc.config.EnableSecurityHeaders {
			sc.applySecurityHeaders(c)
		}

		// Apply input validation early
		if sc.config.EnableInputValidation && sc.inputValidator != nil {
			if !sc.applyInputValidation(c) {
				return
			}
		}

		// Check for API key authentication
		var apiKeyValidated bool
		if sc.config.EnableAPICredentials && sc.apiKeyService != nil {
			apiKeyValidated = sc.validateAPIKey(c)
		}

		// Apply rate limiting
		if sc.config.EnableRateLimiting && sc.rateLimiter != nil {
			if !sc.applyRateLimiting(c) {
				return
			}
		}

		// Apply CSRF protection (skip for API key requests)
		if sc.config.EnableCSRFProtection && sc.csrfProtection != nil && !apiKeyValidated {
			if !sc.applyCSRFProtection(c) {
				return
			}
		}

		// Apply audit logging
		if sc.config.EnableAuditLogging && sc.auditLogger != nil {
			sc.applyAuditLogging(c)
		}

		// Log security events if enabled
		if sc.config.EnableSecurityMonitor && sc.securityMonitor != nil {
			sc.logSecurityEvents(c)
		}

		c.Next()
	}
}

// applySecurityHeaders applies security headers
func (sc *SecurityCoordinator) applySecurityHeaders(c *gin.Context) {
	isProduction := sc.config.Environment == "production"
	securityConfig := APISecurityConfig(isProduction)
	EnhancedSecurityHeaders(securityConfig)(c)
}

// applyInputValidation applies input validation
func (sc *SecurityCoordinator) applyInputValidation(c *gin.Context) bool {
	// Use the input validator middleware
	sc.inputValidator.Middleware()(c)
	return !c.IsAborted()
}

// validateAPIKey validates API key if present
func (sc *SecurityCoordinator) validateAPIKey(c *gin.Context) bool {
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		return false
	}

	req := &auth.ValidateAPIKeyRequest{APIKey: apiKey}
	resp, err := sc.apiKeyService.ValidateAPIKey(c.Request.Context(), req)
	if err != nil || !resp.Valid {
		if sc.config.EnableSecurityMonitor && sc.securityMonitor != nil {
			sc.securityMonitor.UnauthorizedAccess(
				c.ClientIP(),
				c.GetHeader("User-Agent"),
				"",
				c.Request.URL.Path,
				c.Request.Method,
			)
		}
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid API key",
			"code":  "INVALID_API_KEY",
		})
		c.Abort()
		return false
	}

	// Set user information from API key
	if resp.APIKey != nil {
		c.Set("user_id", resp.APIKey.UserID.String())
		c.Set("user_roles", resp.APIKey.Roles)
		c.Set("api_key_id", resp.APIKey.ID.String())
		c.Set("authenticated_via", "api_key")
	}

	return true
}

// applyRateLimiting applies rate limiting
func (sc *SecurityCoordinator) applyRateLimiting(c *gin.Context) bool {
	sc.rateLimiter.Middleware()(c)
	return !c.IsAborted()
}

// applyCSRFProtection applies CSRF protection
func (sc *SecurityCoordinator) applyCSRFProtection(c *gin.Context) bool {
	sc.csrfProtection.Middleware()(c)
	return !c.IsAborted()
}

// applyAuditLogging applies audit logging
func (sc *SecurityCoordinator) applyAuditLogging(c *gin.Context) {
	sc.auditLogger.Middleware()(c)
}

// logSecurityEvents logs security events to the security monitor
func (sc *SecurityCoordinator) logSecurityEvents(c *gin.Context) {
	// Log authentication events
	if strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth/") {
		statusCode := c.Writer.Status()
		if statusCode >= 400 {
			var eventType string
			switch {
			case statusCode == 401:
				eventType = string(security.EventTypeAuthentication)
			case statusCode == 403:
				eventType = string(security.EventTypeAuthorization)
			}

			sc.securityMonitor.LogSecurityEvent(&security.SecurityEvent{
				Type:        security.SecurityEventType(eventType),
				Category:    security.CategoryThreat,
				Title:       "Authentication/Authorization Failed",
				Description: fmt.Sprintf("Auth failure for %s %s", c.Request.Method, c.Request.URL.Path),
				Source:      "auth_middleware",
				ClientIP:    c.ClientIP(),
				UserAgent:   c.GetHeader("User-Agent"),
				Path:        c.Request.URL.Path,
				Method:      c.Request.Method,
				StatusCode:  statusCode,
				Level:       security.SecurityLevelWarning,
				Severity:    security.SeverityMedium,
			})
		}
	}

	// Log rate limit violations
	if c.Writer.Status() == http.StatusTooManyRequests {
		sc.securityMonitor.RateLimitExceeded(
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			sc.getUserID(c),
			c.Request.URL.Path,
		)
	}

	// Log suspicious user agents
	userAgent := c.GetHeader("User-Agent")
	if sc.isSuspiciousUserAgent(userAgent) {
		sc.securityMonitor.SuspiciousActivity(
			c.ClientIP(),
			userAgent,
			sc.getUserID(c),
			"Suspicious user agent detected",
			map[string]interface{}{
				"user_agent": userAgent,
			},
		)
	}
}

// isSuspiciousUserAgent checks if a user agent is suspicious
func (sc *SecurityCoordinator) isSuspiciousUserAgent(userAgent string) bool {
	suspiciousPatterns := []string{
		"sqlmap", "nikto", "nmap", "masscan", "zap", "burp",
		"python-requests", "curl", "wget", "scanner", "bot",
	}

	lowerUserAgent := strings.ToLower(userAgent)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerUserAgent, pattern) {
			return true
		}
	}

	return false
}

// getUserID extracts user ID from context
func (sc *SecurityCoordinator) getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok {
			return userIDStr
		}
	}
	return ""
}

// GetAPIKeyService returns the API key service
func (sc *SecurityCoordinator) GetAPIKeyService() *auth.APIKeyService {
	return sc.apiKeyService
}

// GetSecurityMonitor returns the security monitor
func (sc *SecurityCoordinator) GetSecurityMonitor() *security.SecurityMonitor {
	return sc.securityMonitor
}

// GetSecurityStats returns security statistics
func (sc *SecurityCoordinator) GetSecurityStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if sc.config.EnableSecurityMonitor && sc.securityMonitor != nil {
		stats["security_monitor"] = sc.securityMonitor.GetSecurityStats()
	}

	if sc.config.EnableRateLimiting && sc.rateLimiter != nil {
		// TODO: Add rate limiting stats
		stats["rate_limiting"] = map[string]interface{}{
			"enabled": true,
		}
	}

	stats["enabled_components"] = map[string]bool{
		"security_headers":   sc.config.EnableSecurityHeaders,
		"rate_limiting":      sc.config.EnableRateLimiting,
		"csrf_protection":    sc.config.EnableCSRFProtection,
		"input_validation":  sc.config.EnableInputValidation,
		"audit_logging":      sc.config.EnableAuditLogging,
		"api_credentials":    sc.config.EnableAPICredentials,
		"security_monitor":   sc.config.EnableSecurityMonitor,
	}

	return stats
}

// SecurityMiddleware creates a comprehensive security middleware
func SecurityMiddleware(environment string, cache cache.Cache, logger zerolog.Logger) (gin.HandlerFunc, *SecurityCoordinator, error) {
	config := DefaultSecurityCoordinatorConfig(environment)

	// Adjust configuration based on environment
	if environment == "development" {
		config.CSRF = DevelopmentCSRFConfig()
		config.InputValidation.StrictMode = false
		config.RateLimiting.EnablePenalty = false
	}

	coordinator, err := NewSecurityCoordinator(config, cache, logger)
	if err != nil {
		return nil, nil, err
	}

	// Start the coordinator
	coordinator.Start()

	// Return middleware that can be used in routes
	middleware := func(c *gin.Context) {
		coordinator.Middleware()(c)
	}

	return middleware, coordinator, nil
}

// SecurityMiddlewareWithConfig creates a security middleware with custom configuration
func SecurityMiddlewareWithConfig(config SecurityCoordinatorConfig, cache cache.Cache, logger zerolog.Logger) (gin.HandlerFunc, *SecurityCoordinator, error) {
	coordinator, err := NewSecurityCoordinator(config, cache, logger)
	if err != nil {
		return nil, nil, err
	}

	// Start the coordinator
	coordinator.Start()

	// Return middleware that can be used in routes
	middleware := func(c *gin.Context) {
		coordinator.Middleware()(c)
	}

	return middleware, coordinator, nil
}