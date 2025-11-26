package ratelimit

import (
	"context"
	"time"
)

// EnhancedRateLimiter provides authentication-specific rate limiting with account lockout
type EnhancedRateLimiter interface {
	// AllowLogin checks if a login attempt should be allowed for the given identifier (IP or username)
	AllowLogin(ctx context.Context, identifier string) (bool, error)

	// RecordFailedLogin records a failed login attempt for the given identifier
	RecordFailedLogin(ctx context.Context, identifier string) error

	// IsAccountLocked checks if an account is locked and returns the unlock time
	IsAccountLocked(ctx context.Context, identifier string) (bool, time.Time, error)

	// UnlockAccount manually unlocks an account
	UnlockAccount(ctx context.Context, identifier string) error
}

// AuthLimiterConfig holds configuration for the enhanced rate limiter
type AuthLimiterConfig struct {
	// MaxLoginAttempts is the maximum number of login attempts allowed within the window
	MaxLoginAttempts int

	// LoginWindow is the time window for counting login attempts
	LoginWindow time.Duration

	// LockoutDuration is how long an account remains locked after exceeding max attempts
	LockoutDuration time.Duration

	// StorageType determines the storage backend (memory or redis)
	StorageType StorageType

	// Redis configuration (only used if StorageType is StorageRedis)
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// EnableNotifications determines if email notifications should be sent on lockout
	EnableNotifications bool

	// EmailService is the service used to send notification emails (optional)
	EmailService EmailNotifier
}

// EmailNotifier defines the interface for sending email notifications
type EmailNotifier interface {
	SendAccountLockoutNotification(ctx context.Context, identifier string, unlockTime time.Time) error
}

// NewEnhancedRateLimiter creates a new enhanced rate limiter for authentication
func NewEnhancedRateLimiter(config *AuthLimiterConfig, logger interface{}) (EnhancedRateLimiter, error) {
	if config == nil {
		return nil, &ValidationError{Message: "config cannot be nil"}
	}

	if config.MaxLoginAttempts <= 0 {
		return nil, &ValidationError{Message: "MaxLoginAttempts must be positive"}
	}

	if config.LoginWindow <= 0 {
		return nil, &ValidationError{Message: "LoginWindow must be positive"}
	}

	if config.LockoutDuration <= 0 {
		return nil, &ValidationError{Message: "LockoutDuration must be positive"}
	}

	switch config.StorageType {
	case StorageMemory:
		return newMemoryAuthLimiter(config, logger), nil
	case StorageRedis:
		return newRedisAuthLimiter(config, logger)
	default:
		return nil, &ValidationError{Message: "unsupported storage type"}
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
