package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// CSRFConfig holds configuration for CSRF protection
type CSRFConfig struct {
	// Token configuration
	TokenLength   int           `json:"token_length"`
	TokenName     string        `json:"token_name"`
	HeaderName    string        `json:"header_name"`
	FormFieldName string        `json:"form_field_name"`
	CookieName    string        `json:"cookie_name"`
	CookieDomain  string        `json:"cookie_domain"`
	CookiePath    string        `json:"cookie_path"`
	CookieMaxAge  time.Duration `json:"cookie_max_age"`
	SecureFlag    bool          `json:"secure_flag"`
	HTTPOnlyFlag  bool          `json:"http_only_flag"`
	SameSite      string        `json:"same_site"`

	// Exclusions
	ExcludedMethods []string          `json:"excluded_methods"`
	ExcludedPaths   []string          `json:"excluded_paths"`
	TrustedOrigins  []string          `json:"trusted_origins"`
	TrustedIPs      []string          `json:"trusted_ips"`

	// Security settings
	Enabled        bool `json:"enabled"`
	RequireDouble  bool `json:"require_double"` // Require double submit cookie pattern
	IgnoreGET      bool `json:"ignore_get"`     // Ignore CSRF for GET requests
}

// DefaultCSRFConfig returns a secure default configuration
func DefaultCSRFConfig() CSRFConfig {
	return CSRFConfig{
		TokenLength:     32,
		TokenName:       "csrf_token",
		HeaderName:      "X-CSRF-Token",
		FormFieldName:   "csrf_token",
		CookieName:      "_csrf",
		CookiePath:      "/",
		CookieMaxAge:    24 * time.Hour,
		SecureFlag:      true,  // Enable in production
		HTTPOnlyFlag:    false, // Allow JavaScript access
		SameSite:        "Strict",
		ExcludedMethods: []string{"GET", "HEAD", "OPTIONS", "TRACE"},
		ExcludedPaths:   []string{"/health", "/metrics", "/api/v1/auth/login"},
		TrustedOrigins:  []string{},
		TrustedIPs:      []string{},
		Enabled:         true,
		RequireDouble:   true,
		IgnoreGET:       true,
	}
}

// DevelopmentCSRFConfig returns a less restrictive configuration for development
func DevelopmentCSRFConfig() CSRFConfig {
	config := DefaultCSRFConfig()
	config.SecureFlag = false // Disable for HTTP in development
	config.SameSite = "Lax"
	config.TrustedOrigins = []string{"http://localhost:*", "http://127.0.0.1:*"}
	return config
}

// CSRF represents a CSRF protection middleware
type CSRF struct {
	config CSRFConfig
	logger zerolog.Logger
}

// NewCSRF creates a new CSRF protection middleware
func NewCSRF(config CSRFConfig, logger zerolog.Logger) *CSRF {
	return &CSRF{
		config: config,
		logger: logger,
	}
}

// Middleware returns the CSRF protection middleware
func (c *CSRF) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Skip CSRF if disabled
		if !c.config.Enabled {
			ctx.Next()
			return
		}

		// Skip for excluded methods
		if c.isExcludedMethod(ctx.Request.Method) {
			ctx.Next()
			return
		}

		// Skip for excluded paths
		if c.isExcludedPath(ctx.Request.URL.Path) {
			ctx.Next()
			return
		}

		// Skip for GET requests if configured
		if c.config.IgnoreGET && ctx.Request.Method == "GET" {
			ctx.Next()
			return
		}

		// Check for trusted origin or IP
		if c.isTrustedOrigin(ctx) || c.isTrustedIP(ctx.ClientIP()) {
			ctx.Next()
			return
		}

		// Process request based on method
		switch ctx.Request.Method {
		case "GET":
			c.handleGETRequest(ctx)
		default:
			c.handleStateChangingRequest(ctx)
		}
	}
}

// handleGETRequest handles GET requests by generating and storing CSRF token
func (c *CSRF) handleGETRequest(ctx *gin.Context) {
	// Generate new CSRF token
	token, err := c.generateToken()
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to generate CSRF token")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
			"code":  "CSRF_GENERATION_FAILED",
		})
		ctx.Abort()
		return
	}

	// Store token in context for template rendering
	ctx.Set(c.config.TokenName, token)

	// Set CSRF cookie
	c.setCSRFCookie(ctx, token)

	ctx.Next()
}

// handleStateChangingRequest handles state-changing requests by validating CSRF token
func (c *CSRF) handleStateChangingRequest(ctx *gin.Context) {
	// Get CSRF token from cookie
	cookieToken, err := ctx.Cookie(c.config.CookieName)
	if err != nil {
		c.logger.Warn().Str("client_ip", ctx.ClientIP()).Msg("CSRF cookie missing")
		c.rejectRequest(ctx, "CSRF cookie missing")
		return
	}

	// Get CSRF token from request
	requestToken := c.getRequestToken(ctx)
	if requestToken == "" {
		c.logger.Warn().Str("client_ip", ctx.ClientIP()).Msg("CSRF token missing from request")
		c.rejectRequest(ctx, "CSRF token missing from request")
		return
	}

	// Validate tokens
	if !c.validateTokens(cookieToken, requestToken) {
		c.logger.Warn().
			Str("client_ip", ctx.ClientIP()).
			Str("method", ctx.Request.Method).
			Str("path", ctx.Request.URL.Path).
			Msg("CSRF token validation failed")
		c.rejectRequest(ctx, "Invalid CSRF token")
		return
	}

	ctx.Next()
}

// generateToken generates a new CSRF token
func (c *CSRF) generateToken() (string, error) {
	bytes := make([]byte, c.config.TokenLength)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// setCSRFCookie sets the CSRF cookie
func (c *CSRF) setCSRFCookie(ctx *gin.Context, token string) {
	ctx.SetCookie(
		c.config.CookieName,
		token,
		int(c.config.CookieMaxAge.Seconds()),
		c.config.CookiePath,
		c.config.CookieDomain,
		c.config.SecureFlag,
		c.config.HTTPOnlyFlag,
	)

	// Set SameSite attribute
	if gin.Mode() == gin.ReleaseMode {
		// In production mode, set proper SameSite attribute
		switch strings.ToLower(c.config.SameSite) {
		case "strict":
			ctx.SetSameSite(http.SameSiteStrictMode)
		case "lax":
			ctx.SetSameSite(http.SameSiteLaxMode)
		case "none":
			ctx.SetSameSite(http.SameSiteNoneMode)
		}
	}
}

// getRequestToken extracts CSRF token from request
func (c *CSRF) getRequestToken(ctx *gin.Context) string {
	// Try header first
	token := ctx.GetHeader(c.config.HeaderName)
	if token != "" {
		return token
	}

	// Try form field
	token = ctx.PostForm(c.config.FormFieldName)
	if token != "" {
		return token
	}

	// Try query parameter (less secure, but for compatibility)
	token = ctx.Query(c.config.FormFieldName)
	return token
}

// validateTokens validates that the cookie and request tokens match
func (c *CSRF) validateTokens(cookieToken, requestToken string) bool {
	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(cookieToken), []byte(requestToken)) == 1
}

// isExcludedMethod checks if a method should be excluded from CSRF protection
func (c *CSRF) isExcludedMethod(method string) bool {
	for _, excluded := range c.config.ExcludedMethods {
		if method == excluded {
			return true
		}
	}
	return false
}

// isExcludedPath checks if a path should be excluded from CSRF protection
func (c *CSRF) isExcludedPath(path string) bool {
	for _, excluded := range c.config.ExcludedPaths {
		if strings.HasPrefix(path, excluded) {
			return true
		}
	}
	return false
}

// isTrustedOrigin checks if the request comes from a trusted origin
func (c *CSRF) isTrustedOrigin(ctx *gin.Context) bool {
	origin := ctx.GetHeader("Origin")
	referer := ctx.GetHeader("Referer")

	// Check against trusted origins
	for _, trusted := range c.config.TrustedOrigins {
		// Support wildcards
		if strings.HasSuffix(trusted, "*") {
			prefix := strings.TrimSuffix(trusted, "*")
			if strings.HasPrefix(origin, prefix) || strings.HasPrefix(referer, prefix) {
				return true
			}
		} else {
			if origin == trusted || referer == trusted {
				return true
			}
		}
	}

	return false
}

// isTrustedIP checks if the request comes from a trusted IP
func (c *CSRF) isTrustedIP(clientIP string) bool {
	for _, trusted := range c.config.TrustedIPs {
		if clientIP == trusted {
			return true
		}
	}
	return false
}

// rejectRequest rejects a request with CSRF validation failure
func (c *CSRF) rejectRequest(ctx *gin.Context, message string) {
	ctx.JSON(http.StatusForbidden, gin.H{
		"error": message,
		"code":  "CSRF_VALIDATION_FAILED",
	})
	ctx.Abort()
}

// GetCSRFToken is a helper function to get the CSRF token from the context
func GetCSRFToken(ctx *gin.Context, tokenName string) (string, bool) {
	token, exists := ctx.Get(tokenName)
	if !exists {
		return "", false
	}

	tokenStr, ok := token.(string)
	return tokenStr, ok
}

// CSRFProtection creates a CSRF protection middleware with default configuration
func CSRFProtection(isProduction bool, logger zerolog.Logger) gin.HandlerFunc {
	var config CSRFConfig
	if isProduction {
		config = DefaultCSRFConfig()
	} else {
		config = DevelopmentCSRFConfig()
	}

	csrf := NewCSRF(config, logger)
	return csrf.Middleware()
}

// CSRFProtectionWithConfig creates a CSRF protection middleware with custom configuration
func CSRFProtectionWithConfig(config CSRFConfig, logger zerolog.Logger) gin.HandlerFunc {
	csrf := NewCSRF(config, logger)
	return csrf.Middleware()
}

// APIKeyCSRFProtection creates CSRF protection that's bypassed for API key authentication
func APIKeyCSRFProtection(isProduction bool, logger zerolog.Logger) gin.HandlerFunc {
	config := DefaultCSRFConfig()
	if !isProduction {
		config = DevelopmentCSRFConfig()
	}

	// Add API key paths to exclusions
	config.ExcludedPaths = append(config.ExcludedPaths, "/api/v1/auth/verify-api-key")

	csrf := NewCSRF(config, logger)

	return func(ctx *gin.Context) {
		// Skip CSRF if using API key authentication
		if apiKey := ctx.GetHeader("X-API-Key"); apiKey != "" {
			ctx.Next()
			return
		}

		csrf.Middleware()(ctx)
	}
}