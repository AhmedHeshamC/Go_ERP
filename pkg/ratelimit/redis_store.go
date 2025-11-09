package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// RedisStore implements a Redis-based rate limiter store
type RedisStore struct {
	client *redis.Client
	config *Config
	logger *zerolog.Logger
	prefix string
}

// RedisBucket represents a token bucket stored in Redis
type RedisBucket struct {
	Tokens     float64   `json:"tokens"`
	Capacity   float64   `json:"capacity"`
	RefillRate float64   `json:"refill_rate"`
	LastUpdate int64     `json:"last_update"`
}

// NewRedisStore creates a new Redis store
func NewRedisStore(config *Config, logger *zerolog.Logger) (*RedisStore, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client: rdb,
		config: config,
		logger: logger,
		prefix: "ratelimit:",
	}, nil
}

// Allow checks if a request should be allowed
func (s *RedisStore) Allow(key string, limit RateLimit) (bool, error) {
	ctx := context.Background()
	redisKey := s.getRedisKey(key)

	// Use Lua script for atomic operations
	luaScript := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local refill_rate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])

		local bucket = redis.call('GET', key)
		local tokens, last_update

		if bucket then
			local data = cjson.decode(bucket)
			tokens = data.tokens
			last_update = data.last_update
		else
			tokens = capacity
			last_update = now
		end

		-- Refill tokens
		local elapsed = now - last_update
		tokens = math.min(capacity, tokens + elapsed * refill_rate)

		-- Check if allowed
		if tokens >= 1 then
			tokens = tokens - 1
			local new_bucket = cjson.encode({
				tokens = tokens,
				capacity = capacity,
				refill_rate = refill_rate,
				last_update = now
			})
			redis.call('SET', key, new_bucket, 'EX', 3600)
			return 1
		else
			local new_bucket = cjson.encode({
				tokens = tokens,
				capacity = capacity,
				refill_rate = refill_rate,
				last_update = now
			})
			redis.call('SET', key, new_bucket, 'EX', 3600)
			return 0
		end
	`

	result := s.client.Eval(ctx, luaScript, []string{redisKey},
		limit.BurstSize, limit.RequestsPerSecond, time.Now().Unix())

	if result.Err() != nil {
		return false, result.Err()
	}

	allowed := result.Val().(int64) == 1
	return allowed, nil
}

// AllowN checks if N requests should be allowed
func (s *RedisStore) AllowN(key string, n int, limit RateLimit) (bool, error) {
	if n <= 0 {
		return true, nil
	}

	ctx := context.Background()
	redisKey := s.getRedisKey(key)

	// Use Lua script for atomic operations
	luaScript := `
		local key = KEYS[1]
		local n = tonumber(ARGV[1])
		local capacity = tonumber(ARGV[2])
		local refill_rate = tonumber(ARGV[3])
		local now = tonumber(ARGV[4])

		local bucket = redis.call('GET', key)
		local tokens, last_update

		if bucket then
			local data = cjson.decode(bucket)
			tokens = data.tokens
			last_update = data.last_update
		else
			tokens = capacity
			last_update = now
		end

		-- Refill tokens
		local elapsed = now - last_update
		tokens = math.min(capacity, tokens + elapsed * refill_rate)

		-- Check if allowed
		if tokens >= n then
			tokens = tokens - n
			local new_bucket = cjson.encode({
				tokens = tokens,
				capacity = capacity,
				refill_rate = refill_rate,
				last_update = now
			})
			redis.call('SET', key, new_bucket, 'EX', 3600)
			return 1
		else
			local new_bucket = cjson.encode({
				tokens = tokens,
				capacity = capacity,
				refill_rate = refill_rate,
				last_update = now
			})
			redis.call('SET', key, new_bucket, 'EX', 3600)
			return 0
		end
	`

	result := s.client.Eval(ctx, luaScript, []string{redisKey},
		n, limit.BurstSize, limit.RequestsPerSecond, time.Now().Unix())

	if result.Err() != nil {
		return false, result.Err()
	}

	allowed := result.Val().(int64) == 1
	return allowed, nil
}

// Reserve reserves a request slot
func (s *RedisStore) Reserve(key string, limit RateLimit) (*Reservation, error) {
	ctx := context.Background()
	redisKey := s.getRedisKey(key)

	// Use Lua script for atomic operations
	luaScript := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local refill_rate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])

		local bucket = redis.call('GET', key)
		local tokens, last_update

		if bucket then
			local data = cjson.decode(bucket)
			tokens = data.tokens
			last_update = data.last_update
		else
			tokens = capacity
			last_update = now
		end

		-- Refill tokens
		local elapsed = now - last_update
		tokens = math.min(capacity, tokens + elapsed * refill_rate)

		-- Create reservation
		local ok = false
		local delay = 0

		if tokens >= 1 then
			ok = true
			tokens = tokens - 1
			delay = 0
		else
			ok = false
			delay = (1 - tokens) / refill_rate
		end

		-- Update bucket
		local new_bucket = cjson.encode({
			tokens = tokens,
			capacity = capacity,
			refill_rate = refill_rate,
			last_update = now
		})
		redis.call('SET', key, new_bucket, 'EX', 3600)

		-- Return result
		return cjson.encode({
			ok = ok,
			delay = delay,
			reservation = now,
			time_to_act = now + delay,
			limit = refill_rate,
			remaining = tokens
		})
	`

	result := s.client.Eval(ctx, luaScript, []string{redisKey},
		limit.BurstSize, limit.RequestsPerSecond, time.Now().Unix())

	if result.Err() != nil {
		return &Reservation{OK: false}, result.Err()
	}

	var reservation struct {
		OK          bool    `json:"ok"`
		Delay       float64 `json:"delay"`
		Reservation int64   `json:"reservation"`
		TimeToAct   int64   `json:"time_to_act"`
		Limit       float64 `json:"limit"`
		Remaining   float64 `json:"remaining"`
	}

	if err := json.Unmarshal([]byte(result.Val().(string)), &reservation); err != nil {
		return &Reservation{OK: false}, err
	}

	return &Reservation{
		OK:          reservation.OK,
		Delay:       time.Duration(reservation.Delay * float64(time.Second)),
		Reservation: time.Unix(reservation.Reservation, 0),
		TimeToAct:   time.Unix(reservation.TimeToAct, 0),
		Limit:       reservation.Limit,
		Remaining:   reservation.Remaining,
	}, nil
}

// Reset resets the rate limiter for a key
func (s *RedisStore) Reset(key string) error {
	ctx := context.Background()
	redisKey := s.getRedisKey(key)

	return s.client.Del(ctx, redisKey).Err()
}

// Get returns the token bucket for a key
func (s *RedisStore) Get(key string) (*TokenBucket, error) {
	ctx := context.Background()
	redisKey := s.getRedisKey(key)

	result, err := s.client.Get(ctx, redisKey).Result()
	if err != nil {
		return nil, err
	}

	if result == "" {
		return nil, fmt.Errorf("bucket not found for key: %s", key)
	}

	var redisBucket RedisBucket
	if err := json.Unmarshal([]byte(result), &redisBucket); err != nil {
		return nil, err
	}

	return &TokenBucket{
		Tokens:     redisBucket.Tokens,
		Capacity:   redisBucket.Capacity,
		RefillRate: redisBucket.RefillRate,
		LastUpdate: time.Unix(redisBucket.LastUpdate, 0),
	}, nil
}

// Set sets the token bucket for a key
func (s *RedisStore) Set(key string, bucket *TokenBucket) error {
	ctx := context.Background()
	redisKey := s.getRedisKey(key)

	redisBucket := RedisBucket{
		Tokens:     bucket.Tokens,
		Capacity:   bucket.Capacity,
		RefillRate: bucket.RefillRate,
		LastUpdate: bucket.LastUpdate.Unix(),
	}

	data, err := json.Marshal(redisBucket)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, redisKey, data, 0).Err()
}

// Delete deletes the token bucket for a key
func (s *RedisStore) Delete(key string) error {
	ctx := context.Background()
	redisKey := s.getRedisKey(key)

	return s.client.Del(ctx, redisKey).Err()
}

// Cleanup removes expired buckets
func (s *RedisStore) Cleanup() error {
	ctx := context.Background()
	pattern := s.prefix + "*"

	iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()
	removed := 0
	now := time.Now()

	for iter.Next(ctx) {
		key := iter.Val()

		// Check if bucket is expired
		bucket, err := s.Get(key)
		if err != nil {
			// Try to delete corrupted bucket
			s.client.Del(ctx, key)
			removed++
			continue
		}

		if bucket.LastUpdate.Before(now.Add(-s.config.CleanupInterval)) {
			if err := s.Delete(key); err == nil {
				removed++
			}
		}
	}

	if removed > 0 && s.logger.Debug().Enabled() {
		s.logger.Debug().
			Int("removed", removed).
			Msg("Cleaned up expired Redis rate limiter buckets")
	}

	return nil
}

// getRedisKey returns the Redis key for a rate limiter key
func (s *RedisStore) getRedisKey(key string) string {
	return s.prefix + key
}

// GetStats returns store statistics
func (s *RedisStore) GetStats() map[string]interface{} {
	ctx := context.Background()

	// Count active keys
	pattern := s.prefix + "*"
	iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()
	activeKeys := 0

	for iter.Next(ctx) {
		activeKeys++
	}

	return map[string]interface{}{
		"active_keys": activeKeys,
		"storage_type": string(s.config.StorageType),
		"redis_addr":  s.config.RedisAddr,
		"redis_db":   s.config.RedisDB,
	}
}