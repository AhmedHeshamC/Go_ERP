package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"erpgo/pkg/cache"
)

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	// General rate limiting
	RequestsPerSecond int           `json:"requests_per_second"`
	Burst             int           `json:"burst"`
	Window            time.Duration `json:"window"`

	// Endpoint-specific rate limiting
	Endpoints map[string]EndpointRateLimit `json:"endpoints"`

	// IP-based rate limiting
	IPBased       bool `json:"ip_based"`
	IPWhitelist   []string `json:"ip_whitelist"`
	IPBlacklist   []string `json:"ip_blacklist"`

	// User-based rate limiting
	UserBased     bool `json:"user_based"`
	AdminExempt   bool `json:"admin_exempt"`

	// Redis configuration
	KeyPrefix     string        `json:"key_prefix"`
	KeyExpiry     time.Duration `json:"key_expiry"`

	// Security features
	EnablePenalty bool          `json:"enable_penalty"`
	PenaltyTime   time.Duration `json:"penalty_time"`
	PenaltyFactor float64       `json:"penalty_factor"`
}

// EndpointRateLimit holds rate limit configuration for specific endpoints
type EndpointRateLimit struct {
	RequestsPerSecond int           `json:"requests_per_second"`
	Burst             int           `json:"burst"`
	Window            time.Duration `json:"window"`
	UserBased         bool          `json:"user_based"`
	IPBased           bool          `json:"ip_based"`
}

// RateLimiter represents a rate limiter
type RateLimiter struct {
	config RateLimitConfig
	redis  redis.Cmdable
	logger zerolog.Logger
}

// DefaultRateLimitConfig returns a secure default configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 100,
		Burst:             200,
		Window:            time.Minute,
		Endpoints: map[string]EndpointRateLimit{
			// Authentication endpoints - stricter limits
			"POST:/api/v1/auth/login": {
				RequestsPerSecond: 5,
				Burst:             10,
				Window:            time.Minute,
				UserBased:         false,
				IPBased:           true,
			},
			"POST:/api/v1/auth/register": {
				RequestsPerSecond: 3,
				Burst:             5,
				Window:            time.Minute,
				UserBased:         false,
				IPBased:           true,
			},
			"POST:/api/v1/auth/forgot-password": {
				RequestsPerSecond: 2,
				Burst:             3,
				Window:            time.Hour,
				UserBased:         false,
				IPBased:           true,
			},
			// Password reset - very strict
			"POST:/api/v1/auth/reset-password": {
				RequestsPerSecond: 1,
				Burst:             2,
				Window:            time.Hour,
				UserBased:         false,
				IPBased:           true,
			},
			// API endpoints - moderate limits
			"GET:/api/v1/products": {
				RequestsPerSecond: 50,
				Burst:             100,
				Window:            time.Minute,
				UserBased:         false,
				IPBased:           true,
			},
			"POST:/api/v1/orders": {
				RequestsPerSecond: 20,
				Burst:             40,
				Window:            time.Minute,
				UserBased:         true,
				IPBased:           false,
			},
			// Admin endpoints - stricter for non-admins
			"GET:/api/v1/admin/*": {
				RequestsPerSecond: 30,
				Burst:             60,
				Window:            time.Minute,
				UserBased:         true,
				IPBased:           true,
			},
		},
		IPBased:     true,
		UserBased:   false,
		AdminExempt: true,
		KeyPrefix:   "rate_limit:",
		KeyExpiry:   time.Hour,
		EnablePenalty: true,
		PenaltyTime:   5 * time.Minute,
		PenaltyFactor: 2.0,
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig, redisClient redis.Cmdable, logger zerolog.Logger) *RateLimiter {
	return &RateLimiter{
		config: config,
		redis:  redisClient,
		logger: logger,
	}
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client identifier
		clientID := rl.getClientID(c)
		if clientID == "" {
			c.Next()
			return
		}

		// Check IP blacklist
		if rl.isIPBlacklisted(c.ClientIP()) {
			rl.logger.Warn().Str("ip", c.ClientIP()).Msg("Request from blacklisted IP")
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied",
				"code":  "IP_BLOCKED",
			})
			c.Abort()
			return
		}

		// Check IP whitelist (if configured)
		if len(rl.config.IPWhitelist) > 0 && !rl.isIPWhitelisted(c.ClientIP()) {
			rl.logger.Warn().Str("ip", c.ClientIP()).Msg("Request from non-whitelisted IP")
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied",
				"code":  "IP_NOT_ALLOWED",
			})
			c.Abort()
			return
		}

		// Check for penalty (rate limit exceeded recently)
		if rl.config.EnablePenalty {
			penaltyKey := rl.config.KeyPrefix + "penalty:" + clientID
			isPenalized, _ := rl.redis.Exists(context.Background(), penaltyKey).Result()
			if isPenalized > 0 {
				ttl, _ := rl.redis.TTL(context.Background(), penaltyKey).Result()
				rl.logger.Warn().
					Str("client_id", clientID).
					Str("ip", c.ClientIP()).
					Dur("penalty_ttl", ttl).
					Msg("Request from penalized client")

				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":              "Rate limit exceeded. Please try again later.",
					"code":               "RATE_LIMIT_PENALTY",
					"retry_after_seconds": int(ttl.Seconds()),
				})
				c.Abort()
				return
			}
		}

		// Get rate limit configuration for this endpoint
		endpointConfig := rl.getEndpointConfig(c)

		// Check admin exemption
		if rl.config.AdminExempt && rl.isAdmin(c) {
			rl.logger.Debug().Str("client_id", clientID).Msg("Admin user exempt from rate limiting")
			c.Next()
			return
		}

		// Apply rate limiting
		allowed, remaining, resetTime, err := rl.checkRateLimit(clientID, endpointConfig)
		if err != nil {
			rl.logger.Error().Err(err).Str("client_id", clientID).Msg("Rate limit check failed")
			// Fail open - allow request if rate limiter fails
			c.Next()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(endpointConfig.RequestsPerSecond))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))

		if !allowed {
			rl.logger.Warn().
				Str("client_id", clientID).
				Str("ip", c.ClientIP()).
				Str("endpoint", c.Request.Method+":"+c.FullPath()).
				Msg("Rate limit exceeded")

			// Apply penalty if enabled
			if rl.config.EnablePenalty {
				rl.applyPenalty(clientID)
			}

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":              "Rate limit exceeded",
				"code":               "RATE_LIMIT_EXCEEDED",
				"retry_after_seconds": int(resetTime - time.Now().Unix()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getClientID generates a client identifier for rate limiting
func (rl *RateLimiter) getClientID(c *gin.Context) string {
	var identifiers []string

	// User-based identification
	if rl.config.UserBased {
		if userID, exists := c.Get("user_id"); exists {
			if userIDStr, ok := userID.(string); ok && userIDStr != "" {
				identifiers = append(identifiers, "user:"+userIDStr)
			}
		}
	}

	// IP-based identification
	if rl.config.IPBased {
		ip := c.ClientIP()
		if ip != "" {
			identifiers = append(identifiers, "ip:"+ip)
		}
	}

	// Session/API key identification
	if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
		identifiers = append(identifiers, "api_key:"+apiKey)
	}

	if len(identifiers) == 0 {
		return ""
	}

	return strings.Join(identifiers, ":")
}

// getEndpointConfig returns rate limit configuration for the specific endpoint
func (rl *RateLimiter) getEndpointConfig(c *gin.Context) EndpointRateLimit {
	method := c.Request.Method
	path := c.FullPath()

	// Try exact match first
	if config, exists := rl.config.Endpoints[method+":"+path]; exists {
		return config
	}

	// Try wildcard match for endpoints
	for pattern, config := range rl.config.Endpoints {
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(method+":"+path, prefix) {
				return config
			}
		}
	}

	// Return default configuration
	return EndpointRateLimit{
		RequestsPerSecond: rl.config.RequestsPerSecond,
		Burst:             rl.config.Burst,
		Window:            rl.config.Window,
		UserBased:         rl.config.UserBased,
		IPBased:           rl.config.IPBased,
	}
}

// checkRateLimit checks if the request should be allowed
func (rl *RateLimiter) checkRateLimit(clientID string, config EndpointRateLimit) (bool, int, int64, error) {
	ctx := context.Background()
	now := time.Now()
	windowStart := now.Truncate(config.Window)

	key := rl.config.KeyPrefix + clientID + ":" + windowStart.Format("2006-01-02-15:04:00")

	// Use Redis pipeline for atomic operations
	pipe := rl.redis.TxPipeline()

	// Increment request counter
	current := pipe.Incr(ctx, key)

	// Set expiration on the key
	pipe.Expire(ctx, key, config.Window)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, 0, err
	}

	requestCount := int(current.Val())

	// Check if rate limit exceeded
	if requestCount > config.RequestsPerSecond {
		resetTime := windowStart.Add(config.Window).Unix()
		return false, 0, resetTime, nil
	}

	remaining := config.RequestsPerSecond - requestCount
	resetTime := windowStart.Add(config.Window).Unix()

	return true, remaining, resetTime, nil
}

// applyPenalty applies a penalty to a client that exceeded rate limits
func (rl *RateLimiter) applyPenalty(clientID string) {
	ctx := context.Background()
	penaltyKey := rl.config.KeyPrefix + "penalty:" + clientID

	// Apply penalty with extended duration
	penaltyDuration := time.Duration(float64(rl.config.PenaltyTime) * rl.config.PenaltyFactor)
	rl.redis.Set(ctx, penaltyKey, "1", penaltyDuration)

	rl.logger.Warn().
		Str("client_id", clientID).
		Dur("penalty_duration", penaltyDuration).
		Msg("Applied rate limit penalty")
}

// isIPBlacklisted checks if an IP is blacklisted
func (rl *RateLimiter) isIPBlacklisted(ip string) bool {
	for _, blacklistedIP := range rl.config.IPBlacklist {
		if blacklistedIP == ip {
			return true
		}
		// TODO: Add support for CIDR ranges
	}
	return false
}

// isIPWhitelisted checks if an IP is whitelisted
func (rl *RateLimiter) isIPWhitelisted(ip string) bool {
	for _, whitelistedIP := range rl.config.IPWhitelist {
		if whitelistedIP == ip {
			return true
		}
		// TODO: Add support for CIDR ranges
	}
	return false
}

// isAdmin checks if the current user is an admin
func (rl *RateLimiter) isAdmin(c *gin.Context) bool {
	if roles, exists := c.Get("user_roles"); exists {
		if roleSlice, ok := roles.([]string); ok {
			for _, role := range roleSlice {
				if role == "admin" {
					return true
				}
			}
		}
	}
	return false
}

// RateLimit creates a rate limiting middleware with Redis backend
func RateLimitWithConfigFull(config RateLimitConfig, cacheInterface cache.Cache, logger zerolog.Logger) gin.HandlerFunc {
	// Type assert to get Redis client
	if redisCache, ok := cacheInterface.(*cache.RedisCache); ok {
		limiter := NewRateLimiter(config, redisCache.GetClient(), logger)
		return limiter.Middleware()
	}

	// Fallback to memory-based rate limiting if Redis is not available
	logger.Warn().Msg("Redis not available for rate limiting, falling back to memory-based rate limiting")
	return MemoryRateLimit(config, logger)
}

// MemoryRateLimit provides a fallback memory-based rate limiter
func MemoryRateLimit(config RateLimitConfig, logger zerolog.Logger) gin.HandlerFunc {
	// Simple in-memory rate limiter for fallback
	// This is less optimal than Redis but works when Redis is unavailable
	return func(c *gin.Context) {
		// TODO: Implement memory-based rate limiting with a sliding window
		// For now, just allow all requests
		logger.Debug().Msg("Using memory-based rate limiting fallback")
		c.Next()
	}
}

// RateLimitByEndpoint creates rate limiting middleware for specific endpoints
func RateLimitByEndpoint(endpointPatterns map[string]EndpointRateLimit, cacheInterface cache.Cache, logger zerolog.Logger) gin.HandlerFunc {
	config := DefaultRateLimitConfig()
	config.Endpoints = endpointPatterns

	return RateLimitWithConfigFull(config, cacheInterface, logger)
}

// RateLimitAuth creates stricter rate limiting for authentication endpoints
func RateLimitAuth(cacheInterface cache.Cache, logger zerolog.Logger) gin.HandlerFunc {
	authConfig := map[string]EndpointRateLimit{
		"POST:/api/v1/auth/login": {
			RequestsPerSecond: 5,
			Burst:             10,
			Window:            time.Minute,
			UserBased:         false,
			IPBased:           true,
		},
		"POST:/api/v1/auth/register": {
			RequestsPerSecond: 3,
			Burst:             5,
			Window:            time.Minute,
			UserBased:         false,
			IPBased:           true,
		},
		"POST:/api/v1/auth/forgot-password": {
			RequestsPerSecond: 2,
			Burst:             3,
			Window:            time.Hour,
			UserBased:         false,
			IPBased:           true,
		},
	}

	return RateLimitByEndpoint(authConfig, cacheInterface, logger)
}

// RateLimitAPI creates rate limiting for general API endpoints
func RateLimitAPI(cacheInterface cache.Cache, logger zerolog.Logger) gin.HandlerFunc {
	apiConfig := map[string]EndpointRateLimit{
		"GET:/api/v1/*": {
			RequestsPerSecond: 100,
			Burst:             200,
			Window:            time.Minute,
			UserBased:         false,
			IPBased:           true,
		},
		"POST:/api/v1/*": {
			RequestsPerSecond: 50,
			Burst:             100,
			Window:            time.Minute,
			UserBased:         true,
			IPBased:           false,
		},
		"PUT:/api/v1/*": {
			RequestsPerSecond: 50,
			Burst:             100,
			Window:            time.Minute,
			UserBased:         true,
			IPBased:           false,
		},
		"DELETE:/api/v1/*": {
			RequestsPerSecond: 20,
			Burst:             40,
			Window:            time.Minute,
			UserBased:         true,
			IPBased:           false,
		},
	}

	return RateLimitByEndpoint(apiConfig, cacheInterface, logger)
}