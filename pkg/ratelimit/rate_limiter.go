package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	// Allow checks if a request should be allowed
	Allow(key string) bool
	// AllowN checks if N requests should be allowed
	AllowN(key string, n int) bool
	// Reserve reserves a request slot
	Reserve(key string) *Reservation
	// Wait blocks until a request is allowed or context is cancelled
	Wait(ctx context.Context, key string) error
	// WaitN blocks until N requests are allowed or context is cancelled
	WaitN(ctx context.Context, key string, n int) error
	// Reset resets the rate limiter for a key
	Reset(key string)
	// GetLimit returns the current limit for a key
	GetLimit(key string) RateLimit
	// GetStats returns rate limiter statistics
	GetStats() map[string]interface{}
}

// RateLimit represents rate limiting configuration
type RateLimit struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	BurstSize         int     `json:"burst_size"`
}

// Reservation represents a rate limit reservation
type Reservation struct {
	OK          bool          `json:"ok"`
	Delay       time.Duration `json:"delay"`
	Reservation time.Time     `json:"reservation"`
	TimeToAct   time.Time     `json:"time_to_act"`
	Limit       float64       `json:"limit"`
	Remaining   float64       `json:"remaining"`
}

// Config holds rate limiter configuration
type Config struct {
	// Global settings
	DefaultLimit    RateLimit     `json:"default_limit"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	MaxKeys         int           `json:"max_keys"`

	// Storage settings
	StorageType   StorageType `json:"storage_type"`
	RedisAddr     string      `json:"redis_addr,omitempty"`
	RedisPassword string      `json:"redis_password,omitempty"`
	RedisDB       int         `json:"redis_db,omitempty"`

	// Logging settings
	LogRequests     bool `json:"log_requests"`
	LogRejections   bool `json:"log_rejections"`
	LogReservations bool `json:"log_reservations"`

	// Advanced settings
	EnableKeyHashing    bool `json:"enable_key_hashing"`
	EnableSlidingWindow bool `json:"enable_sliding_window"`
}

// StorageType defines the storage backend for rate limiting
type StorageType string

const (
	StorageMemory StorageType = "memory"
	StorageRedis  StorageType = "redis"
)

// DefaultConfig returns a default rate limiter configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultLimit: RateLimit{
			RequestsPerSecond: 10.0,
			BurstSize:         20,
		},
		CleanupInterval:     5 * time.Minute,
		MaxKeys:             10000,
		StorageType:         StorageMemory,
		LogRequests:         true,
		LogRejections:       true,
		LogReservations:     false,
		EnableKeyHashing:    false,
		EnableSlidingWindow: false,
	}
}

// Limiter implements the main rate limiter
type Limiter struct {
	config *Config
	logger *zerolog.Logger
	store  Store
	stats  *Stats
}

// Store defines the interface for rate limiter storage backends
type Store interface {
	Allow(key string, limit RateLimit) (bool, error)
	AllowN(key string, n int, limit RateLimit) (bool, error)
	Reserve(key string, limit RateLimit) (*Reservation, error)
	Reset(key string) error
	Get(key string) (*TokenBucket, error)
	Set(key string, bucket *TokenBucket) error
	Delete(key string) error
	Cleanup() error
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	Tokens     float64   `json:"tokens"`
	Capacity   float64   `json:"capacity"`
	RefillRate float64   `json:"refill_rate"`
	LastUpdate time.Time `json:"last_update"`
}

// Stats tracks rate limiter statistics
type Stats struct {
	mu               sync.RWMutex
	TotalRequests    int64     `json:"total_requests"`
	AllowedRequests  int64     `json:"allowed_requests"`
	RejectedRequests int64     `json:"rejected_requests"`
	Reservations     int64     `json:"reservations"`
	ActiveKeys       int       `json:"active_keys"`
	PeakKeys         int       `json:"peak_keys"`
	LastCleanup      time.Time `json:"last_cleanup"`
}

// New creates a new rate limiter
func New(config *Config, logger *zerolog.Logger) (*Limiter, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if logger == nil {
		nopLogger := zerolog.Nop()
		logger = &nopLogger
	}

	var store Store
	var err error

	switch config.StorageType {
	case StorageMemory:
		store = NewMemoryStore(config, logger)
	case StorageRedis:
		store, err = NewRedisStore(config, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create redis store: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.StorageType)
	}

	limiter := &Limiter{
		config: config,
		logger: logger,
		store:  store,
		stats: &Stats{
			LastCleanup: time.Now(),
		},
	}

	// Start cleanup routine
	go limiter.cleanupRoutine()

	return limiter, nil
}

// Allow checks if a request should be allowed
func (l *Limiter) Allow(key string) bool {
	allowed, err := l.store.Allow(key, l.config.DefaultLimit)
	if err != nil {
		l.logger.Error().Err(err).Str("key", key).Msg("Rate limiter error")
		return false
	}

	l.updateStats(allowed, false)

	if l.config.LogRequests {
		l.logger.Debug().
			Str("key", key).
			Bool("allowed", allowed).
			Msg("Rate limit check")
	}

	return allowed
}

// AllowN checks if N requests should be allowed
func (l *Limiter) AllowN(key string, n int) bool {
	allowed, err := l.store.AllowN(key, n, l.config.DefaultLimit)
	if err != nil {
		l.logger.Error().Err(err).Str("key", key).Int("n", n).Msg("Rate limiter error")
		return false
	}

	l.updateStats(allowed, false)

	if l.config.LogRequests {
		l.logger.Debug().
			Str("key", key).
			Int("n", n).
			Bool("allowed", allowed).
			Msg("Rate limit check")
	}

	return allowed
}

// Reserve reserves a request slot
func (l *Limiter) Reserve(key string) *Reservation {
	reservation, err := l.store.Reserve(key, l.config.DefaultLimit)
	if err != nil {
		l.logger.Error().Err(err).Str("key", key).Msg("Rate limiter reservation error")
		return &Reservation{OK: false}
	}

	if l.config.LogReservations {
		l.logger.Debug().
			Str("key", key).
			Bool("ok", reservation.OK).
			Dur("delay", reservation.Delay).
			Msg("Rate limit reservation")
	}

	l.updateStats(reservation.OK, true)

	return reservation
}

// Wait blocks until a request is allowed or context is cancelled
func (l *Limiter) Wait(ctx context.Context, key string) error {
	for {
		if l.Allow(key) {
			return nil
		}

		reservation := l.Reserve(key)
		if !reservation.OK {
			return fmt.Errorf("rate limit exceeded")
		}

		if reservation.Delay == 0 {
			return nil
		}

		timer := time.NewTimer(reservation.Delay)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Try again
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// WaitN blocks until N requests are allowed or context is cancelled
func (l *Limiter) WaitN(ctx context.Context, key string, n int) error {
	for n > 0 {
		if l.Allow(key) {
			n--
			continue
		}

		reservation := l.Reserve(key)
		if !reservation.OK {
			return fmt.Errorf("rate limit exceeded")
		}

		if reservation.Delay == 0 {
			n--
			continue
		}

		timer := time.NewTimer(reservation.Delay)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Try again
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// Reset resets the rate limiter for a key
func (l *Limiter) Reset(key string) {
	err := l.store.Reset(key)
	if err != nil {
		l.logger.Error().Err(err).Str("key", key).Msg("Failed to reset rate limiter")
	}

	if l.config.LogRequests {
		l.logger.Info().Str("key", key).Msg("Rate limiter reset")
	}
}

// GetLimit returns the current limit for a key
func (l *Limiter) GetLimit(key string) RateLimit {
	return l.config.DefaultLimit
}

// GetStats returns rate limiter statistics
func (l *Limiter) GetStats() map[string]interface{} {
	l.stats.mu.RLock()
	defer l.stats.mu.RUnlock()

	return map[string]interface{}{
		"total_requests":    l.stats.TotalRequests,
		"allowed_requests":  l.stats.AllowedRequests,
		"rejected_requests": l.stats.RejectedRequests,
		"reservations":      l.stats.Reservations,
		"active_keys":       l.stats.ActiveKeys,
		"peak_keys":         l.stats.PeakKeys,
		"last_cleanup":      l.stats.LastCleanup,
		"rejection_rate":    float64(l.stats.RejectedRequests) / float64(l.stats.TotalRequests) * 100,
		"allowance_rate":    float64(l.stats.AllowedRequests) / float64(l.stats.TotalRequests) * 100,
	}
}

// cleanupRoutine periodically cleans up expired keys
func (l *Limiter) cleanupRoutine() {
	ticker := time.NewTicker(l.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := l.store.Cleanup(); err != nil {
			l.logger.Error().Err(err).Msg("Rate limiter cleanup failed")
		}

		l.stats.mu.Lock()
		l.stats.LastCleanup = time.Now()
		l.stats.mu.Unlock()
	}
}

// updateStats updates rate limiter statistics
func (l *Limiter) updateStats(allowed bool, isReservation bool) {
	l.stats.mu.Lock()
	defer l.stats.mu.Unlock()

	l.stats.TotalRequests++

	if isReservation {
		l.stats.Reservations++
	} else if allowed {
		l.stats.AllowedRequests++
	} else {
		l.stats.RejectedRequests++
		if l.config.LogRejections {
			l.logger.Warn().
				Int64("total", l.stats.TotalRequests).
				Int64("rejected", l.stats.RejectedRequests).
				Float64("rejection_rate", float64(l.stats.RejectedRequests)/float64(l.stats.TotalRequests)*100).
				Msg("High rejection rate detected")
		}
	}
}

// hashKey optionally hashes a key for privacy or storage optimization
func (l *Limiter) hashKey(key string) string {
	if !l.config.EnableKeyHashing {
		return key
	}
	// Simple hash implementation - in production, use a proper hash function
	return fmt.Sprintf("%x", len(key)+int(key[0])*31)
}
