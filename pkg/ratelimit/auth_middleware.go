package ratelimit

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthRateLimitMiddleware creates a middleware for authentication rate limiting
func AuthRateLimitMiddleware(limiter EnhancedRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		ipAddress := c.ClientIP()

		// Check rate limit
		allowed, err := limiter.AllowLogin(c.Request.Context(), ipAddress)
		if err != nil || !allowed {
			// Handle rate limit error
			if rateLimitErr, ok := err.(*RateLimitError); ok {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":       "rate_limit_exceeded",
					"message":     rateLimitErr.Message,
					"retry_after": int(rateLimitErr.RetryAfter.Seconds()),
				})
				c.Abort()
				return
			}

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many login attempts. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AccountLockoutMiddleware creates a middleware to check for account lockout
func AccountLockoutMiddleware(limiter EnhancedRateLimiter, identifierFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get identifier (email or username)
		identifier := identifierFunc(c)
		if identifier == "" {
			// If we can't get the identifier, skip the check
			c.Next()
			return
		}

		// Check if account is locked
		isLocked, unlockTime, err := limiter.IsAccountLocked(c.Request.Context(), identifier)
		if err != nil {
			// Log error but don't block the request
			c.Next()
			return
		}

		if isLocked {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "account_locked",
				"message":     fmt.Sprintf("Account is locked until %s", unlockTime.Format(time.RFC3339)),
				"unlock_time": unlockTime.Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RecordFailedLoginMiddleware creates a middleware to record failed login attempts
// This should be used after authentication has failed
func RecordFailedLoginMiddleware(limiter EnhancedRateLimiter, identifierFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only record if the request failed with 401 Unauthorized
		if c.Writer.Status() == http.StatusUnauthorized {
			identifier := identifierFunc(c)
			if identifier != "" {
				if err := limiter.RecordFailedLogin(c.Request.Context(), identifier); err != nil {
					// Log error but don't fail the request
					// In production, you'd use a proper logger here
				}
			}
		}
	}
}
