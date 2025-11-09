package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityConfig holds configuration for security headers
type SecurityConfig struct {
	// Content Security Policy
	CSPEnabled       bool     `json:"csp_enabled"`
	CSPDefaultSrc    string   `json:"csp_default_src"`
	CSPScriptSrc     string   `json:"csp_script_src"`
	CSPStyleSrc      string   `json:"csp_style_src"`
	CSPImgSrc        string   `json:"csp_img_src"`
	CSPFontSrc       string   `json:"csp_font_src"`
	CSPConnectSrc    string   `json:"csp_connect_src"`
	CSPMediaSrc      string   `json:"csp_media_src"`
	CSPObjectSrc     string   `json:"csp_object_src"`
	CSPChildSrc      string   `json:"csp_child_src"`
	CSPFrameSrc      string   `json:"csp_frame_src"`
	CSPWorkerSrc     string   `json:"csp_worker_src"`
	CSPManifestSrc   string   `json:"csp_manifest_src"`
	CSPUpgradeInsecureRequests bool `json:"csp_upgrade_insecure_requests"`

	// HSTS Configuration
	HSTSEnabled         bool   `json:"hsts_enabled"`
	HSTSMaxAge          int    `json:"hsts_max_age"`
	HSTSIncludeSubDomains bool  `json:"hsts_include_subdomains"`
	HSTSPreload         bool   `json:"hsts_preload"`

	// Other Security Headers
	XFrameOptions         string `json:"x_frame_options"`     // DENY, SAMEORIGIN, ALLOW-FROM
	XContentTypeOptions   string `json:"x_content_type_options"` // nosniff
	XSSProtection         string `json:"xss_protection"`        // 1; mode=block
	ReferrerPolicy        string `json:"referrer_policy"`
	PermissionsPolicy     string `json:"permissions_policy"`
	StrictTransportSecurity string `json:"strict_transport_security"`

	// Feature Policy / Permissions Policy
	FeaturePolicyEnabled  bool     `json:"feature_policy_enabled"`
	AllowedFeatures       []string `json:"allowed_features"`

	// Custom Headers
	CustomHeaders map[string]string `json:"custom_headers"`

	// Environment-specific settings
	IsProduction bool `json:"is_production"`
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig(isProduction bool) SecurityConfig {
	config := SecurityConfig{
		CSPEnabled: true,
		CSPDefaultSrc: "'self'",
		CSPScriptSrc: "'self' 'unsafe-inline' 'unsafe-eval'",
		CSPStyleSrc: "'self' 'unsafe-inline'",
		CSPImgSrc: "'self' data: https:",
		CSPFontSrc: "'self' data:",
		CSPConnectSrc: "'self'",
		CSPMediaSrc: "'self'",
		CSPObjectSrc: "'none'",
		CSPChildSrc: "'self'",
		CSPFrameSrc: "'none'",
		CSPWorkerSrc: "'self'",
		CSPManifestSrc: "'self'",
		CSPUpgradeInsecureRequests: isProduction,

		// HSTS - enabled in production
		HSTSEnabled: isProduction,
		HSTSMaxAge: 31536000, // 1 year
		HSTSIncludeSubDomains: true,
		HSTSPreload: true,

		// Standard security headers
		XFrameOptions: "DENY",
		XContentTypeOptions: "nosniff",
		XSSProtection: "1; mode=block",
		ReferrerPolicy: "strict-origin-when-cross-origin",

		// Permissions Policy
		FeaturePolicyEnabled: true,
		AllowedFeatures: []string{
			"geolocation=()",
			"microphone=()",
			"camera=()",
			"payment=()",
			"usb=()",
			"magnetometer=()",
			"gyroscope=()",
			"accelerometer=()",
			"ambient-light-sensor=()",
			"autoplay=(self)",
			"encrypted-media=(self)",
			"fullscreen=(self)",
			"picture-in-picture=(self)",
		},

		CustomHeaders: make(map[string]string),
		IsProduction: isProduction,
	}

	// Build CSP and Permissions Policy strings
	config.CSPDefaultSrc = config.buildCSP()
	config.PermissionsPolicy = strings.Join(config.AllowedFeatures, ", ")

	return config
}

// DevelopmentSecurityConfig returns a less restrictive configuration for development
func DevelopmentSecurityConfig() SecurityConfig {
	config := DefaultSecurityConfig(false)

	// More permissive CSP for development
	config.CSPScriptSrc = "'self' 'unsafe-inline' 'unsafe-eval' localhost:* 127.0.0.1:* ws://localhost:* wss://localhost:*"
	config.CSPStyleSrc = "'self' 'unsafe-inline' localhost:* 127.0.0.1:*"
	config.CSPConnectSrc = "'self' localhost:* 127.0.0.1:* ws://localhost:* wss://localhost:*"

	// Disable HSTS in development
	config.HSTSEnabled = false

	return config
}

// APISecurityConfig returns configuration optimized for APIs
func APISecurityConfig(isProduction bool) SecurityConfig {
	config := DefaultSecurityConfig(isProduction)

	// More restrictive CSP for APIs (no script/style sources needed)
	config.CSPScriptSrc = "'none'"
	config.CSPStyleSrc = "'none'"
	config.CSPImgSrc = "'none'"
	config.CSPFontSrc = "'none'"
	config.CSPMediaSrc = "'none'"
	config.CSPObjectSrc = "'none'"
	config.CSPChildSrc = "'none'"
	config.CSPFrameSrc = "'none'"
	config.CSPWorkerSrc = "'none'"
	config.CSPManifestSrc = "'none'"

	// API-specific headers
	config.CustomHeaders["X-API-Version"] = "v1"
	config.CustomHeaders["X-Content-Type-Options"] = "nosniff"

	return config
}

// buildCSP builds the Content-Security-Policy header value
func (c SecurityConfig) buildCSP() string {
	var directives []string

	if c.CSPDefaultSrc != "" {
		directives = append(directives, "default-src "+c.CSPDefaultSrc)
	}
	if c.CSPScriptSrc != "" {
		directives = append(directives, "script-src "+c.CSPScriptSrc)
	}
	if c.CSPStyleSrc != "" {
		directives = append(directives, "style-src "+c.CSPStyleSrc)
	}
	if c.CSPImgSrc != "" {
		directives = append(directives, "img-src "+c.CSPImgSrc)
	}
	if c.CSPFontSrc != "" {
		directives = append(directives, "font-src "+c.CSPFontSrc)
	}
	if c.CSPConnectSrc != "" {
		directives = append(directives, "connect-src "+c.CSPConnectSrc)
	}
	if c.CSPMediaSrc != "" {
		directives = append(directives, "media-src "+c.CSPMediaSrc)
	}
	if c.CSPObjectSrc != "" {
		directives = append(directives, "object-src "+c.CSPObjectSrc)
	}
	if c.CSPChildSrc != "" {
		directives = append(directives, "child-src "+c.CSPChildSrc)
	}
	if c.CSPFrameSrc != "" {
		directives = append(directives, "frame-src "+c.CSPFrameSrc)
	}
	if c.CSPWorkerSrc != "" {
		directives = append(directives, "worker-src "+c.CSPWorkerSrc)
	}
	if c.CSPManifestSrc != "" {
		directives = append(directives, "manifest-src "+c.CSPManifestSrc)
	}
	if c.CSPUpgradeInsecureRequests {
		directives = append(directives, "upgrade-insecure-requests")
	}

	return strings.Join(directives, "; ")
}

// buildHSTS builds the Strict-Transport-Security header value
func (c SecurityConfig) buildHSTS() string {
	if !c.HSTSEnabled {
		return ""
	}

	var parts []string
	parts = append(parts, "max-age="+string(rune(c.HSTSMaxAge)))

	if c.HSTSIncludeSubDomains {
		parts = append(parts, "includeSubDomains")
	}

	if c.HSTSPreload {
		parts = append(parts, "preload")
	}

	return strings.Join(parts, "; ")
}

// EnhancedSecurityHeaders creates a middleware with comprehensive security headers
func EnhancedSecurityHeaders(config SecurityConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content Security Policy
		if config.CSPEnabled {
			csp := config.buildCSP()
			if csp != "" {
				c.Header("Content-Security-Policy", csp)
			}
		}

		// HTTP Strict Transport Security (HSTS)
		if config.HSTSEnabled {
			hsts := config.buildHSTS()
			if hsts != "" {
				c.Header("Strict-Transport-Security", hsts)
			}
		}

		// X-Frame-Options
		if config.XFrameOptions != "" {
			c.Header("X-Frame-Options", config.XFrameOptions)
		}

		// X-Content-Type-Options
		if config.XContentTypeOptions != "" {
			c.Header("X-Content-Type-Options", config.XContentTypeOptions)
		}

		// X-XSS-Protection
		if config.XSSProtection != "" {
			c.Header("X-XSS-Protection", config.XSSProtection)
		}

		// Referrer Policy
		if config.ReferrerPolicy != "" {
			c.Header("Referrer-Policy", config.ReferrerPolicy)
		}

		// Permissions Policy
		if config.FeaturePolicyEnabled && config.PermissionsPolicy != "" {
			c.Header("Permissions-Policy", config.PermissionsPolicy)
		}

		// Custom headers
		for key, value := range config.CustomHeaders {
			c.Header(key, value)
		}

		// Additional security headers
		c.Header("X-DNS-Prefetch-Control", "off")
		c.Header("X-Download-Options", "noopen")
		c.Header("X-Permitted-Cross-Domain-Policies", "none")
		c.Header("Cross-Origin-Embedder-Policy", "require-corp")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")

		// Remove server information
		c.Header("Server", "")

		// Cache control for sensitive endpoints
		if isSensitiveEndpoint(c.Request.URL.Path) {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
			c.Header("Surrogate-Control", "no-store")
		}

		c.Next()
	}
}

// isSensitiveEndpoint checks if the endpoint is sensitive and should not be cached
func isSensitiveEndpoint(path string) bool {
	sensitivePaths := []string{
		"/api/v1/auth",
		"/api/v1/users",
		"/api/v1/admin",
		"/api/v1/secure",
		"/api/v1/payment",
		"/api/v1/personal",
		"/api/v1/sensitive",
	}

	for _, sensitive := range sensitivePaths {
		if strings.HasPrefix(path, sensitive) {
			return true
		}
	}

	return false
}

// SecurityHeadersWithEnvironment creates security headers based on environment
func SecurityHeadersWithEnvironment(isProduction bool) gin.HandlerFunc {
	var config SecurityConfig

	if isProduction {
		config = APISecurityConfig(true)
	} else {
		config = DevelopmentSecurityConfig()
	}

	return EnhancedSecurityHeaders(config)
}

// CORSSecurityHeaders creates CORS middleware with security considerations
func CORSSecurityHeaders(allowedOrigins, allowedMethods, allowedHeaders []string, isProduction bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// In production, be more restrictive with origins
		if isProduction {
			// Validate origin more strictly in production
			if !isOriginAllowed(origin, allowedOrigins) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "Origin not allowed",
					"code":  "ORIGIN_NOT_ALLOWED",
				})
				c.Abort()
				return
			}
		}

		// Check if origin is in allowed list
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

			// Add security headers for CORS
			c.Header("Access-Control-Max-Age", "86400") // 24 hours
			c.Header("Vary", "Origin")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isOriginAllowed performs additional validation on origins
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return true // Allow same-origin requests
	}

	// Check for exact match
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}

	// Additional validation can be added here
	// - Check for HTTPS in production
	// - Validate domain patterns
	// - Check against blocklist

	return false
}