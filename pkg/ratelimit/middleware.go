package ratelimit

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Middleware provides rate limiting middleware for Gin
type Middleware struct {
	limiter     RateLimiter
	logger      *zerolog.Logger
	keyFunc     KeyFunc
	skipper     Skipper
	errorHandler ErrorHandler
}

// KeyFunc generates a rate limiting key from a request
type KeyFunc func(*gin.Context) string

// Skipper determines if a request should be skipped from rate limiting
type Skipper func(*gin.Context) bool

// ErrorHandler handles rate limit exceeded errors
type ErrorHandler func(*gin.Context, error)

// KeyFunc constants for common rate limiting strategies
var (
	KeyFuncIP = KeyFunc(func(c *gin.Context) string {
		return c.ClientIP()
	})

	KeyFuncUser = KeyFunc(func(c *gin.Context) string {
		userID, exists := c.Get("user_id")
		if exists {
			return fmt.Sprintf("user:%v", userID)
		}
		return KeyFuncIP(c)
	})

	KeyFuncIPUser = KeyFunc(func(c *gin.Context) string {
		userID, exists := c.Get("user_id")
		if exists {
			return fmt.Sprintf("%s:user:%v", c.ClientIP(), userID)
		}
		return c.ClientIP()
	})

	KeyFuncEndpoint = KeyFunc(func(c *gin.Context) string {
		return fmt.Sprintf("%s:%s", c.Request.Method, c.FullPath())
	})

	KeyFuncRoute = KeyFunc(func(c *gin.Context) string {
		return fmt.Sprintf("%s:%s", c.Request.Method, c.Request.URL.Path)
	})
)

// DefaultErrorHandler is the default error handler for rate limit exceeded
func DefaultErrorHandler(c *gin.Context, err error) {
	c.JSON(http.StatusTooManyRequests, gin.H{
		"error":   "rate limit exceeded",
		"message": "Too many requests. Please try again later.",
		"code":    "RATE_LIMIT_EXCEEDED",
	})
}

// NewMiddleware creates a new rate limiting middleware
func NewMiddleware(limiter RateLimiter, logger *zerolog.Logger) *Middleware {
	return &Middleware{
		limiter:      limiter,
		logger:       logger,
		keyFunc:      KeyFuncIP,
		skipper:      func(c *gin.Context) bool { return false },
		errorHandler: DefaultErrorHandler,
	}
}

// WithKeyFunc sets the key function for rate limiting
func (m *Middleware) WithKeyFunc(keyFunc KeyFunc) *Middleware {
	m.keyFunc = keyFunc
	return m
}

// WithSkipper sets the skipper function for rate limiting
func (m *Middleware) WithSkipper(skipper Skipper) *Middleware {
	m.skipper = skipper
	return m
}

// WithErrorHandler sets the error handler for rate limit exceeded
func (m *Middleware) WithErrorHandler(errorHandler ErrorHandler) *Middleware {
	m.errorHandler = errorHandler
	return m
}

// Middleware returns the Gin middleware function
func (m *Middleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
	// Skip rate limiting if configured
		if m.skipper(c) {
			c.Next()
			return
		}

		// Generate rate limiting key
		key := m.keyFunc(c)

	// Apply rate limiting
	allowed := m.limiter.Allow(key)
		if !allowed {
			m.logger.Warn().
				Str("key", key).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Str("ip", c.ClientIP()).
				Msg("Rate limit exceeded")

			m.errorHandler(c, fmt.Errorf("rate limit exceeded for key: %s", key))
			c.Abort()
			return
		}

		// Add rate limit headers
		m.addRateLimitHeaders(c, key)

		c.Next()
	}
}

// addRateLimitHeaders adds rate limit headers to the response
func (m *Middleware) addRateLimitHeaders(c *gin.Context, key string) {
	limit := m.limiter.GetLimit(key)

	// Add standard rate limit headers
	c.Header("X-RateLimit-Limit", fmt.Sprintf("%.0f", limit.RequestsPerSecond))
	c.Header("X-RateLimit-Remaining", fmt.Sprintf("%.0f", limit.RequestsPerSecond-1))
	c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))

	// Add custom headers
	c.Header("X-RateLimit-Burst", fmt.Sprintf("%d", limit.BurstSize))
	c.Header("X-RateLimit-Policy", "fixed-window")
}

// LimitConfig represents rate limiting configuration for specific routes
type LimitConfig struct {
	Path     string    `json:"path"`
	Method   string    `json:"method"`
	Limit    RateLimit `json:"limit"`
	KeyFunc  KeyFunc   `json:"-"`
	Skipper  Skipper   `json:"-"`
}

// NewMultiLimiterMiddleware creates a middleware that applies different limits based on configuration
func NewMultiLimiterMiddleware(limiter RateLimiter, configs []LimitConfig, logger *zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Find matching configuration
		var matchingConfig *LimitConfig
		for _, config := range configs {
			if matchesConfig(c, config) {
				matchingConfig = &config
				break
			}
		}

		// Use default config if no match found
		if matchingConfig == nil {
			matchingConfig = &LimitConfig{
				Limit: limiter.GetLimit("default"),
			}
		}

		// Apply skipper if configured
		if matchingConfig.Skipper != nil && matchingConfig.Skipper(c) {
			c.Next()
			return
		}

		// Generate key
		keyFunc := matchingConfig.KeyFunc
		if keyFunc == nil {
			keyFunc = KeyFuncIP
		}
		key := keyFunc(c)

		// Check rate limit
		allowed := limiter.Allow(key)
		if !allowed {
			logger.Warn().
				Str("key", key).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Float64("limit", matchingConfig.Limit.RequestsPerSecond).
				Msg("Rate limit exceeded for specific route")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": "Too many requests. Please try again later.",
				"code":    "RATE_LIMIT_EXCEEDED",
				"limit":   matchingConfig.Limit.RequestsPerSecond,
			})
			c.Abort()
			return
		}

		// Add headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%.0f", matchingConfig.Limit.RequestsPerSecond))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%.0f", matchingConfig.Limit.RequestsPerSecond-1))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))

		c.Next()
	}
}

// matchesConfig checks if a request matches a limit configuration
func matchesConfig(c *gin.Context, config LimitConfig) bool {
	// Check method
	if config.Method != "" && config.Method != c.Request.Method {
		return false
	}

	// Check path
	if config.Path != "" {
		// Exact match
		if config.Path == c.Request.URL.Path {
			return true
			}
		// Prefix match
		if strings.HasSuffix(config.Path, "*") {
			prefix := strings.TrimSuffix(config.Path, "*")
			if strings.HasPrefix(c.Request.URL.Path, prefix) {
				return true
			}
		}
	}

	return false
}

// Global rate limiter instance
var GlobalLimiter RateLimiter

// InitializeGlobalLimiter initializes the global rate limiter
func InitializeGlobalLimiter(config *Config, logger *zerolog.Logger) error {
	var err error
	GlobalLimiter, err = New(config, logger)
	return err
}

// GetGlobalLimiter returns the global rate limiter
func GetGlobalLimiter() RateLimiter {
	if GlobalLimiter == nil {
		// Create with default config if not initialized
		nopLogger := zerolog.Nop()
		GlobalLimiter, _ = New(DefaultConfig(), &nopLogger)
	}
	return GlobalLimiter
}

// RateLimitMiddleware is a convenient middleware that uses the global rate limiter
func RateLimitMiddleware(limit RateLimit) gin.HandlerFunc {
	limiter := GetGlobalLimiter()

	return func(c *gin.Context) {
		key := KeyFuncIP(c)
		allowed := limiter.Allow(key)

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": "Too many requests. Please try again later.",
				"code":    "RATE_LIMIT_EXCEEDED",
				"limit":   limit.RequestsPerSecond,
			})
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%.0f", limit.RequestsPerSecond))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%.0f", limit.RequestsPerSecond-1))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))
		c.Header("X-RateLimit-Burst", fmt.Sprintf("%d", limit.BurstSize))

		c.Next()
	}
}

// RateLimitByIP limits requests by IP address
func RateLimitByIP(requestsPerSecond float64, burstSize int) gin.HandlerFunc {
	return RateLimitMiddleware(RateLimit{
		RequestsPerSecond: requestsPerSecond,
		BurstSize:        burstSize,
	})
}

// RateLimitByUser limits requests by user ID
func RateLimitByUser(requestsPerSecond float64, burstSize int) gin.HandlerFunc {
	limiter := GetGlobalLimiter()

	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			// Fallback to IP-based limiting if user is not authenticated
			key := KeyFuncIP(c)
			allowed := limiter.Allow(key)

			if !allowed {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":   "rate limit exceeded",
					"message": "Too many requests. Please try again later.",
					"code":    "RATE_LIMIT_EXCEEDED",
				})
				c.Abort()
				return
			}

			c.Next()
			return
		}

		key := fmt.Sprintf("user:%v", userID)
		allowed := limiter.Allow(key)

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": "Too many requests. Please try again later.",
				"code":    "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByEndpoint limits requests by endpoint
func RateLimitByEndpoint(requestsPerSecond float64, burstSize int) gin.HandlerFunc {
	limiter := GetGlobalLimiter()

	return func(c *gin.Context) {
		key := fmt.Sprintf("%s:%s", c.Request.Method, c.FullPath())
		allowed := limiter.Allow(key)

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": "Too many requests for this endpoint. Please try again later.",
				"code":    "RATE_LIMIT_EXCEEDED",
				"endpoint": c.FullPath(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}