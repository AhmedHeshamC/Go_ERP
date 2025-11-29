package ratelimit

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// MemoryStore implements an in-memory rate limiter store
type MemoryStore struct {
	config  *Config
	logger  *zerolog.Logger
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
	stats   *StoreStats
}

// StoreStats tracks store statistics
type StoreStats struct {
	ActiveBuckets int
	TotalBuckets  int
	PeakKeys      int
	LastCleanup   time.Time
}

// NewMemoryStore creates a new memory store
func NewMemoryStore(config *Config, logger *zerolog.Logger) *MemoryStore {
	return &MemoryStore{
		config:  config,
		logger:  logger,
		buckets: make(map[string]*TokenBucket),
		stats:   &StoreStats{},
	}
}

// Allow checks if a request should be allowed
func (s *MemoryStore) Allow(key string, limit RateLimit) (bool, error) {
	bucket := s.getBucket(key, limit)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.LastUpdate).Seconds()
	bucket.Tokens += elapsed * limit.RequestsPerSecond
	if bucket.Tokens > bucket.Capacity {
		bucket.Tokens = bucket.Capacity
	}
	bucket.LastUpdate = now

	if bucket.Tokens >= 1 {
		bucket.Tokens--
		return true, nil
	}

	return false, nil
}

// AllowN checks if N requests should be allowed
func (s *MemoryStore) AllowN(key string, n int, limit RateLimit) (bool, error) {
	if n <= 0 {
		return true, nil
	}

	bucket := s.getBucket(key, limit)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.LastUpdate).Seconds()
	bucket.Tokens += elapsed * limit.RequestsPerSecond
	if bucket.Tokens > bucket.Capacity {
		bucket.Tokens = bucket.Capacity
	}
	bucket.LastUpdate = now

	if bucket.Tokens >= float64(n) {
		bucket.Tokens -= float64(n)
		return true, nil
	}

	return false, nil
}

// Reserve reserves a request slot
func (s *MemoryStore) Reserve(key string, limit RateLimit) (*Reservation, error) {
	bucket := s.getBucket(key, limit)

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Refill tokens
	elapsed := now.Sub(bucket.LastUpdate).Seconds()
	bucket.Tokens += elapsed * limit.RequestsPerSecond
	if bucket.Tokens > bucket.Capacity {
		bucket.Tokens = bucket.Capacity
	}
	bucket.LastUpdate = now

	if bucket.Tokens >= 1 {
		bucket.Tokens--
		return &Reservation{
			OK:        true,
			Delay:     0,
			TimeToAct: now,
			Limit:     limit.RequestsPerSecond,
			Remaining: bucket.Tokens,
		}, nil
	}

	// Calculate delay for next token
	delay := time.Duration((1.0 - bucket.Tokens) / limit.RequestsPerSecond * float64(time.Second))
	return &Reservation{
		OK:          false,
		Delay:       delay,
		Reservation: now,
		TimeToAct:   now.Add(delay),
		Limit:       limit.RequestsPerSecond,
		Remaining:   bucket.Tokens,
	}, nil
}

// Reset resets the rate limiter for a key
func (s *MemoryStore) Reset(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.buckets, key)
	s.stats.ActiveBuckets = len(s.buckets)

	return nil
}

// Get returns the token bucket for a key
func (s *MemoryStore) Get(key string) (*TokenBucket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bucket, exists := s.buckets[key]
	if !exists {
		return nil, fmt.Errorf("bucket not found for key: %s", key)
	}

	return bucket, nil
}

// Set sets the token bucket for a key
func (s *MemoryStore) Set(key string, bucket *TokenBucket) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buckets[key] = bucket

	if len(s.buckets) > s.stats.TotalBuckets {
		s.stats.TotalBuckets = len(s.buckets)
	}
	s.stats.ActiveBuckets = len(s.buckets)

	if len(s.buckets) > s.stats.PeakKeys {
		s.stats.PeakKeys = len(s.buckets)
	}

	return nil
}

// Delete deletes the token bucket for a key
func (s *MemoryStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.buckets, key)
	s.stats.ActiveBuckets = len(s.buckets)

	return nil
}

// Cleanup removes expired buckets
func (s *MemoryStore) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-s.config.CleanupInterval)
	removed := 0

	for key, bucket := range s.buckets {
		if bucket.LastUpdate.Before(cutoff) {
			delete(s.buckets, key)
			removed++
		}
	}

	s.stats.ActiveBuckets = len(s.buckets)
	s.stats.LastCleanup = now

	if removed > 0 && s.logger.Debug().Enabled() {
		s.logger.Debug().
			Int("removed", removed).
			Int("remaining", len(s.buckets)).
			Msg("Cleaned up expired rate limiter buckets")
	}

	return nil
}

// getBucket returns or creates a token bucket for a key
func (s *MemoryStore) getBucket(key string, limit RateLimit) *TokenBucket {
	bucket, exists := s.buckets[key]
	if !exists {
		bucket = &TokenBucket{
			Tokens:     float64(limit.BurstSize),
			Capacity:   float64(limit.BurstSize),
			RefillRate: limit.RequestsPerSecond,
			LastUpdate: time.Now(),
		}
		s.buckets[key] = bucket

		if len(s.buckets) > s.stats.TotalBuckets {
			s.stats.TotalBuckets = len(s.buckets)
		}
		s.stats.ActiveBuckets = len(s.buckets)
	}

	return bucket
}

// GetStats returns store statistics
func (s *MemoryStore) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"active_buckets": s.stats.ActiveBuckets,
		"total_buckets":  s.stats.TotalBuckets,
		"peak_keys":      s.stats.PeakKeys,
		"last_cleanup":   s.stats.LastCleanup,
		"storage_type":   string(s.config.StorageType),
	}
}
