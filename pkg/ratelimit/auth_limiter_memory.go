package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// memoryAuthLimiter implements EnhancedRateLimiter using in-memory storage
type memoryAuthLimiter struct {
	config   *AuthLimiterConfig
	logger   interface{}
	attempts map[string]*loginAttempts
	lockouts map[string]*accountLockout
	mu       sync.RWMutex

	// Prometheus metrics
	loginAttemptsTotal   prometheus.Counter
	loginSuccessTotal    prometheus.Counter
	loginFailureTotal    prometheus.Counter
	accountLockoutsTotal prometheus.Counter
	rateLimitExceeded    prometheus.Counter
	activeLockedAccounts prometheus.Gauge
}

// loginAttempts tracks login attempts for an identifier
type loginAttempts struct {
	count       int
	windowStart time.Time
}

// accountLockout tracks account lockout information
type accountLockout struct {
	lockedAt   time.Time
	unlockTime time.Time
}

// Global metrics (shared across all instances to avoid duplicate registration)
var (
	globalLoginAttemptsTotal   prometheus.Counter
	globalLoginSuccessTotal    prometheus.Counter
	globalLoginFailureTotal    prometheus.Counter
	globalAccountLockoutsTotal prometheus.Counter
	globalRateLimitExceeded    prometheus.Counter
	globalActiveLockedAccounts prometheus.Gauge
	metricsOnce                sync.Once
)

// initMetrics initializes global metrics once
func initMetrics() {
	metricsOnce.Do(func() {
		globalLoginAttemptsTotal = promauto.NewCounter(prometheus.CounterOpts{
			Name: "auth_login_attempts_total",
			Help: "Total number of login attempts",
		})
		globalLoginSuccessTotal = promauto.NewCounter(prometheus.CounterOpts{
			Name: "auth_login_success_total",
			Help: "Total number of successful login attempts",
		})
		globalLoginFailureTotal = promauto.NewCounter(prometheus.CounterOpts{
			Name: "auth_login_failure_total",
			Help: "Total number of failed login attempts",
		})
		globalAccountLockoutsTotal = promauto.NewCounter(prometheus.CounterOpts{
			Name: "auth_account_lockouts_total",
			Help: "Total number of account lockouts",
		})
		globalRateLimitExceeded = promauto.NewCounter(prometheus.CounterOpts{
			Name: "auth_rate_limit_exceeded_total",
			Help: "Total number of times rate limit was exceeded",
		})
		globalActiveLockedAccounts = promauto.NewGauge(prometheus.GaugeOpts{
			Name: "auth_active_locked_accounts",
			Help: "Number of currently locked accounts",
		})
	})
}

// newMemoryAuthLimiter creates a new memory-based auth limiter
func newMemoryAuthLimiter(config *AuthLimiterConfig, logger interface{}) *memoryAuthLimiter {
	// Initialize global metrics
	initMetrics()

	limiter := &memoryAuthLimiter{
		config:               config,
		logger:               logger,
		attempts:             make(map[string]*loginAttempts),
		lockouts:             make(map[string]*accountLockout),
		loginAttemptsTotal:   globalLoginAttemptsTotal,
		loginSuccessTotal:    globalLoginSuccessTotal,
		loginFailureTotal:    globalLoginFailureTotal,
		accountLockoutsTotal: globalAccountLockoutsTotal,
		rateLimitExceeded:    globalRateLimitExceeded,
		activeLockedAccounts: globalActiveLockedAccounts,
	}

	// Start cleanup routine
	go limiter.cleanupRoutine()

	return limiter
}

// AllowLogin checks if a login attempt should be allowed
func (l *memoryAuthLimiter) AllowLogin(ctx context.Context, identifier string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Record login attempt
	l.loginAttemptsTotal.Inc()

	// Check if account is locked
	if lockout, exists := l.lockouts[identifier]; exists {
		if time.Now().Before(lockout.unlockTime) {
			l.rateLimitExceeded.Inc()
			return false, &RateLimitError{
				Message:    fmt.Sprintf("account locked until %s", lockout.unlockTime.Format(time.RFC3339)),
				RetryAfter: time.Until(lockout.unlockTime),
			}
		}
		// Lockout expired, remove it
		delete(l.lockouts, identifier)
		l.activeLockedAccounts.Dec()
	}

	// Get or create attempts record
	attempts, exists := l.attempts[identifier]
	if !exists {
		attempts = &loginAttempts{
			count:       0,
			windowStart: time.Now(),
		}
		l.attempts[identifier] = attempts
	}

	// Check if window has expired
	if time.Since(attempts.windowStart) > l.config.LoginWindow {
		// Reset the window
		attempts.count = 0
		attempts.windowStart = time.Now()
	}

	// Check if limit exceeded
	if attempts.count >= l.config.MaxLoginAttempts {
		l.rateLimitExceeded.Inc()
		return false, &RateLimitError{
			Message:    fmt.Sprintf("rate limit exceeded: %d attempts in %s", attempts.count, l.config.LoginWindow),
			RetryAfter: l.config.LoginWindow - time.Since(attempts.windowStart),
		}
	}

	// Increment attempt count
	attempts.count++

	return true, nil
}

// RecordFailedLogin records a failed login attempt
func (l *memoryAuthLimiter) RecordFailedLogin(ctx context.Context, identifier string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Record failed login
	l.loginFailureTotal.Inc()

	// Get or create attempts record
	attempts, exists := l.attempts[identifier]
	if !exists {
		attempts = &loginAttempts{
			count:       0,
			windowStart: time.Now(),
		}
		l.attempts[identifier] = attempts
	}

	// Check if window has expired
	if time.Since(attempts.windowStart) > l.config.LoginWindow {
		// Reset the window
		attempts.count = 0
		attempts.windowStart = time.Now()
	}

	// Increment failure count
	attempts.count++

	// Check if we should lock the account
	if attempts.count >= l.config.MaxLoginAttempts {
		now := time.Now()
		unlockTime := now.Add(l.config.LockoutDuration)
		l.lockouts[identifier] = &accountLockout{
			lockedAt:   now,
			unlockTime: unlockTime,
		}

		// Record account lockout
		l.accountLockoutsTotal.Inc()
		l.activeLockedAccounts.Inc()

		// Send notification if enabled
		if l.config.EnableNotifications && l.config.EmailService != nil {
			go func() {
				ctx := context.Background()
				if err := l.config.EmailService.SendAccountLockoutNotification(ctx, identifier, unlockTime); err != nil {
					// Log error but don't fail the operation
					// In a real implementation, we'd use the logger here
				}
			}()
		}
	}

	return nil
}

// RecordSuccessfulLogin records a successful login attempt
func (l *memoryAuthLimiter) RecordSuccessfulLogin(ctx context.Context, identifier string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Record successful login
	l.loginSuccessTotal.Inc()

	// Clear attempts for this identifier
	delete(l.attempts, identifier)

	return nil
}

// IsAccountLocked checks if an account is locked
func (l *memoryAuthLimiter) IsAccountLocked(ctx context.Context, identifier string) (bool, time.Time, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	lockout, exists := l.lockouts[identifier]
	if !exists {
		return false, time.Time{}, nil
	}

	// Check if lockout has expired
	if time.Now().After(lockout.unlockTime) {
		return false, time.Time{}, nil
	}

	return true, lockout.unlockTime, nil
}

// UnlockAccount manually unlocks an account
func (l *memoryAuthLimiter) UnlockAccount(ctx context.Context, identifier string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.lockouts[identifier]; exists {
		l.activeLockedAccounts.Dec()
	}

	delete(l.lockouts, identifier)
	delete(l.attempts, identifier)

	return nil
}

// cleanupRoutine periodically cleans up expired entries
func (l *memoryAuthLimiter) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanup()
	}
}

// cleanup removes expired entries
func (l *memoryAuthLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// Clean up expired lockouts
	for identifier, lockout := range l.lockouts {
		if now.After(lockout.unlockTime) {
			delete(l.lockouts, identifier)
			l.activeLockedAccounts.Dec()
		}
	}

	// Clean up old attempt records
	for identifier, attempts := range l.attempts {
		if time.Since(attempts.windowStart) > l.config.LoginWindow*2 {
			delete(l.attempts, identifier)
		}
	}

	// Update active locked accounts gauge to ensure accuracy
	l.activeLockedAccounts.Set(float64(len(l.lockouts)))
}

// RateLimitError represents a rate limit error
type RateLimitError struct {
	Message    string
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return e.Message
}
