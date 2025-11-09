package middleware

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/pkg/config"
)

// CORSMiddleware creates a CORS middleware with proper origin validation
func CORSMiddleware(corsConfig config.CORSConfig, logger zerolog.Logger) gin.HandlerFunc {
	// Precompile regex patterns for better performance
	var allowedPatterns []*regexp.Regexp
	var blockedPatterns []*regexp.Regexp

	for _, origin := range corsConfig.Origins {
		if strings.Contains(origin, "*") {
			// Convert wildcard patterns to regex
			pattern := strings.ReplaceAll(origin, ".", "\\.")
			pattern = strings.ReplaceAll(pattern, "*", ".*")
			regex, err := regexp.Compile("^" + pattern + "$")
			if err != nil {
				logger.Warn().Str("origin", origin).Err(err).Msg("Failed to compile CORS origin pattern")
				continue
			}
			allowedPatterns = append(allowedPatterns, regex)
		}
	}

	// Blocked domains (common attack vectors)
	blockedDomains := []string{
		"malicious.com",
		"evil.com",
		"attack.com",
		"phishing.com",
	}

	for _, domain := range blockedDomains {
		pattern := ".*" + strings.ReplaceAll(domain, ".", "\\.") + ".*"
		regex, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		blockedPatterns = append(blockedPatterns, regex)
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Handle requests without Origin header (same-origin, mobile apps, etc.)
		if origin == "" {
			// Add security headers for non-CORS requests
			c.Header("X-Content-Type-Options", "nosniff")
			c.Header("X-Frame-Options", "DENY")
			c.Next()
			return
		}

		// Check if origin is blocked
		for _, pattern := range blockedPatterns {
			if pattern.MatchString(origin) {
				logger.Warn().Str("origin", origin).Msg("Blocked CORS request from suspicious origin")
				c.JSON(http.StatusForbidden, gin.H{
					"error": "Origin not allowed",
					"code":  "ORIGIN_BLOCKED",
				})
				c.Abort()
				return
			}
		}

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range corsConfig.Origins {
			if allowedOrigin == "*" {
				// Wildcard - allow all origins (only in development)
				if !corsConfig.IsProduction {
					allowed = true
					break
				} else {
					// In production, don't allow wildcard
					logger.Warn().Str("origin", origin).Msg("Wildcard CORS origin not allowed in production")
					continue
				}
			}

			if allowedOrigin == origin {
				allowed = true
				break
			}
		}

		// Check regex patterns if no exact match found
		if !allowed {
			for _, pattern := range allowedPatterns {
				if pattern.MatchString(origin) {
					allowed = true
					break
				}
			}
		}

		// Environment-based validation for production
		if allowed && corsConfig.IsProduction && corsConfig.EnvironmentWhitelist {
			if !isOriginValidForProduction(origin) {
				logger.Warn().Str("origin", origin).Msg("Origin failed production validation")
				allowed = false
			}
		}

		if allowed {
			// Set CORS headers
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", strings.Join(corsConfig.Methods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(corsConfig.Headers, ", "))
			c.Header("Access-Control-Max-Age", strconv.Itoa(corsConfig.MaxAge))
			c.Header("Vary", "Origin")

			if corsConfig.Credentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}

			// Additional security headers
			c.Header("Access-Control-Expose-Headers", "X-Request-ID,X-RateLimit-Limit,X-RateLimit-Remaining")

			logger.Debug().Str("origin", origin).Msg("CORS request allowed")
		} else {
			logger.Warn().Str("origin", origin).Msg("CORS request denied - origin not allowed")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if allowed {
				c.Status(http.StatusNoContent)
			} else {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "Origin not allowed",
					"code":  "ORIGIN_NOT_ALLOWED",
				})
			}
			c.Abort()
			return
		}

		if !allowed && corsConfig.IsProduction {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Origin not allowed",
				"code":  "ORIGIN_NOT_ALLOWED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// isOriginValidForProduction performs additional validation for production origins
func isOriginValidForProduction(origin string) bool {
	// In production, require HTTPS
	if !strings.HasPrefix(origin, "https://") {
		// Allow localhost for development/testing, even in production config
		if !strings.Contains(origin, "localhost") && !strings.Contains(origin, "127.0.0.1") {
			return false
		}
	}

	// Check for valid domain structure
	if !strings.Contains(origin, ".") && !strings.Contains(origin, "localhost") {
		return false
	}

	// Additional checks can be added here:
	// - Domain reputation check
	// - HSTS preloaded list
	// - Certificate pinning validation
	// - Rate limiting per origin

	return true
}

// CORSForAPI creates a CORS middleware specifically configured for APIs
func CORSForAPI(corsConfig config.CORSConfig, logger zerolog.Logger) gin.HandlerFunc {
	// More restrictive configuration for APIs
	apiConfig := corsConfig
	apiConfig.Headers = append(apiConfig.Headers, "X-API-Key", "X-Request-ID")

	if !apiConfig.IsProduction {
		// In development, allow more methods for testing
		apiConfig.Methods = append(apiConfig.Methods, "PATCH", "HEAD")
	}

	return CORSMiddleware(apiConfig, logger)
}

// CORSForWebApp creates a CORS middleware specifically configured for web applications
func CORSForWebApp(corsConfig config.CORSConfig, logger zerolog.Logger) gin.HandlerFunc {
	// Configuration optimized for web applications
	webConfig := corsConfig

	// Web apps typically need more headers
	webConfig.Headers = append(webConfig.Headers,
		"X-Requested-With",
		"Accept-Language",
		"Accept-Encoding",
		"Cache-Control",
		"Pragma",
	)

	// Allow preflight caching
	webConfig.MaxAge = 86400 // 24 hours

	return CORSMiddleware(webConfig, logger)
}

// CORSWithDynamicOrigins creates a CORS middleware that can dynamically validate origins
func CORSWithDynamicOrigins(originValidator func(string) bool, corsConfig config.CORSConfig, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin == "" {
			c.Next()
			return
		}

		// Use dynamic validator
		allowed := originValidator(origin)

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", strings.Join(corsConfig.Methods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(corsConfig.Headers, ", "))
			c.Header("Access-Control-Max-Age", strconv.Itoa(corsConfig.MaxAge))

			if corsConfig.Credentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}

			logger.Debug().Str("origin", origin).Msg("Dynamic CORS request allowed")
		} else {
			logger.Warn().Str("origin", origin).Msg("Dynamic CORS request denied")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if allowed {
				c.Status(http.StatusNoContent)
			} else {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "Origin not allowed",
					"code":  "ORIGIN_NOT_ALLOWED",
				})
			}
			c.Abort()
			return
		}

		if !allowed && corsConfig.IsProduction {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Origin not allowed",
				"code":  "ORIGIN_NOT_ALLOWED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PreflightCache creates a middleware to cache preflight requests
func PreflightCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			// Set cache headers for preflight requests
			c.Header("Access-Control-Max-Age", "86400") // 24 hours
			c.Header("Cache-Control", "public, max-age=86400")
		}
		c.Next()
	}
}