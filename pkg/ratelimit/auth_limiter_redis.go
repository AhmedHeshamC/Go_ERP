package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisAuthLimiter implements EnhancedRateLimiter using Redis storage
type redisAuthLimiter struct {
	config *AuthLimiterConfig
	logger interface{}
	client *redis.Client
	prefix string
}

// newRedisAuthLimiter creates a new Redis-based auth limiter
func newRedisAuthLimiter(config *AuthLimiterConfig, logger interface{}) (*redisAuthLimiter, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &redisAuthLimiter{
		config: config,
		logger: logger,
		client: client,
		prefix: "auth_limiter:",
	}, nil
}

// AllowLogin checks if a login attempt should be allowed
func (l *redisAuthLimiter) AllowLogin(ctx context.Context, identifier string) (bool, error) {
	lockoutKey := l.getLockoutKey(identifier)
	attemptsKey := l.getAttemptsKey(identifier)

	// Check if account is locked
	lockoutTime, err := l.client.Get(ctx, lockoutKey).Result()
	if err == nil {
		unlockTime, err := time.Parse(time.RFC3339, lockoutTime)
		if err == nil && time.Now().Before(unlockTime) {
			return false, &RateLimitError{
				Message:    fmt.Sprintf("account locked until %s", unlockTime.Format(time.RFC3339)),
				RetryAfter: time.Until(unlockTime),
			}
		}
		// Lockout expired, delete it
		l.client.Del(ctx, lockoutKey)
	}

	// Use Lua script for atomic increment and check
	luaScript := `
		local key = KEYS[1]
		local max_attempts = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		
		local current = redis.call('GET', key)
		if not current then
			redis.call('SET', key, 1, 'EX', window)
			return 1
		end
		
		local count = tonumber(current)
		if count >= max_attempts then
			return 0
		end
		
		redis.call('INCR', key)
		return 1
	`

	result := l.client.Eval(ctx, luaScript, []string{attemptsKey},
		l.config.MaxLoginAttempts, int(l.config.LoginWindow.Seconds()))

	if result.Err() != nil {
		return false, result.Err()
	}

	allowed := result.Val().(int64) == 1
	if !allowed {
		// Get current count for error message
		count, _ := l.client.Get(ctx, attemptsKey).Int()
		return false, &RateLimitError{
			Message:    fmt.Sprintf("rate limit exceeded: %d attempts in %s", count, l.config.LoginWindow),
			RetryAfter: l.config.LoginWindow,
		}
	}

	return true, nil
}

// RecordFailedLogin records a failed login attempt
func (l *redisAuthLimiter) RecordFailedLogin(ctx context.Context, identifier string) error {
	attemptsKey := l.getAttemptsKey(identifier)
	lockoutKey := l.getLockoutKey(identifier)

	// Use Lua script for atomic increment and lockout check
	luaScript := `
		local attempts_key = KEYS[1]
		local lockout_key = KEYS[2]
		local max_attempts = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local lockout_duration = tonumber(ARGV[3])
		local unlock_time = ARGV[4]
		
		local current = redis.call('GET', attempts_key)
		local count = 1
		
		if current then
			count = tonumber(current) + 1
			redis.call('INCR', attempts_key)
		else
			redis.call('SET', attempts_key, 1, 'EX', window)
		end
		
		if count >= max_attempts then
			redis.call('SET', lockout_key, unlock_time, 'EX', lockout_duration)
			return 1
		end
		
		return 0
	`

	unlockTime := time.Now().Add(l.config.LockoutDuration)
	result := l.client.Eval(ctx, luaScript,
		[]string{attemptsKey, lockoutKey},
		l.config.MaxLoginAttempts,
		int(l.config.LoginWindow.Seconds()),
		int(l.config.LockoutDuration.Seconds()),
		unlockTime.Format(time.RFC3339))

	if result.Err() != nil {
		return result.Err()
	}

	// Check if account was locked
	locked := result.Val().(int64) == 1
	if locked && l.config.EnableNotifications && l.config.EmailService != nil {
		go func() {
			ctx := context.Background()
			if err := l.config.EmailService.SendAccountLockoutNotification(ctx, identifier, unlockTime); err != nil {
				// Log error but don't fail the operation
			}
		}()
	}

	return nil
}

// IsAccountLocked checks if an account is locked
func (l *redisAuthLimiter) IsAccountLocked(ctx context.Context, identifier string) (bool, time.Time, error) {
	lockoutKey := l.getLockoutKey(identifier)

	lockoutTime, err := l.client.Get(ctx, lockoutKey).Result()
	if err == redis.Nil {
		return false, time.Time{}, nil
	}
	if err != nil {
		return false, time.Time{}, err
	}

	unlockTime, err := time.Parse(time.RFC3339, lockoutTime)
	if err != nil {
		return false, time.Time{}, err
	}

	// Check if lockout has expired
	if time.Now().After(unlockTime) {
		l.client.Del(ctx, lockoutKey)
		return false, time.Time{}, nil
	}

	return true, unlockTime, nil
}

// UnlockAccount manually unlocks an account
func (l *redisAuthLimiter) UnlockAccount(ctx context.Context, identifier string) error {
	lockoutKey := l.getLockoutKey(identifier)
	attemptsKey := l.getAttemptsKey(identifier)

	pipe := l.client.Pipeline()
	pipe.Del(ctx, lockoutKey)
	pipe.Del(ctx, attemptsKey)
	_, err := pipe.Exec(ctx)

	return err
}

// getLockoutKey returns the Redis key for account lockout
func (l *redisAuthLimiter) getLockoutKey(identifier string) string {
	return l.prefix + "lockout:" + identifier
}

// getAttemptsKey returns the Redis key for login attempts
func (l *redisAuthLimiter) getAttemptsKey(identifier string) string {
	return l.prefix + "attempts:" + identifier
}

// Helper function to convert string to int
func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
