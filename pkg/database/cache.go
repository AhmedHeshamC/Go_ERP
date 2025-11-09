package database

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

// RedisCache implements QueryCache interface using Redis
type RedisCache struct {
	client     redis.Cmdable
	prefix     string
	defaultTTL time.Duration
	logger     *zerolog.Logger
}

// CacheEntry represents a cached database result
type CacheEntry struct {
	Data      interface{} `json:"data"`
	CreatedAt time.Time   `json:"created_at"`
	TTL       time.Duration `json:"ttl"`
}

// NewRedisCache creates a new Redis-based query cache
func NewRedisCache(client redis.Cmdable, prefix string, defaultTTL time.Duration, logger *zerolog.Logger) *RedisCache {
	if logger == nil {
		l := zerolog.Nop()
		logger = &l
	}

	return &RedisCache{
		client:     client,
		prefix:     prefix,
		defaultTTL: defaultTTL,
		logger:     logger,
	}
}

// Get retrieves a value from cache
func (rc *RedisCache) Get(ctx context.Context, key string) (interface{}, bool) {
	fullKey := rc.prefix + key

	start := time.Now()
	result, err := rc.client.Get(ctx, fullKey).Result()
	duration := time.Since(start)

	if err == redis.Nil {
		rc.logger.Debug().
			Str("key", key).
			Dur("duration", duration).
			Msg("Cache miss")
		return nil, false
	}

	if err != nil {
		rc.logger.Error().
			Err(err).
			Str("key", key).
			Dur("duration", duration).
			Msg("Cache get error")
		return nil, false
	}

	// Deserialize the cache entry
	var entry CacheEntry
	if err := json.Unmarshal([]byte(result), &entry); err != nil {
		rc.logger.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to deserialize cache entry")
		return nil, false
	}

	// Check if the entry has expired (double-check TTL)
	if time.Since(entry.CreatedAt) > entry.TTL {
		rc.client.Del(ctx, fullKey) // Clean up expired entry
		return nil, false
	}

	rc.logger.Debug().
		Str("key", key).
		Dur("duration", duration).
		Time("created_at", entry.CreatedAt).
		Msg("Cache hit")

	return entry.Data, true
}

// Set stores a value in cache with TTL
func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	if ttl <= 0 {
		ttl = rc.defaultTTL
	}

	entry := CacheEntry{
		Data:      value,
		CreatedAt: time.Now(),
		TTL:       ttl,
	}

	// Serialize the cache entry
	data, err := json.Marshal(entry)
	if err != nil {
		rc.logger.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to serialize cache entry")
		return
	}

	fullKey := rc.prefix + key

	start := time.Now()
	err = rc.client.Set(ctx, fullKey, data, ttl).Err()
	duration := time.Since(start)

	if err != nil {
		rc.logger.Error().
			Err(err).
			Str("key", key).
			Dur("duration", duration).
			Msg("Cache set error")
		return
	}

	rc.logger.Debug().
		Str("key", key).
		Dur("duration", duration).
		Dur("ttl", ttl).
		Msg("Cache set")
}

// Delete removes a value from cache
func (rc *RedisCache) Delete(ctx context.Context, key string) {
	fullKey := rc.prefix + key

	start := time.Now()
	err := rc.client.Del(ctx, fullKey).Err()
	duration := time.Since(start)

	if err != nil {
		rc.logger.Error().
			Err(err).
			Str("key", key).
			Dur("duration", duration).
			Msg("Cache delete error")
		return
	}

	rc.logger.Debug().
		Str("key", key).
		Dur("duration", duration).
		Msg("Cache delete")
}

// InvalidatePattern removes all keys matching a pattern
func (rc *RedisCache) InvalidatePattern(ctx context.Context, pattern string) error {
	fullPattern := rc.prefix + pattern

	start := time.Now()
	keys, err := rc.client.Keys(ctx, fullPattern).Result()
	duration := time.Since(start)

	if err != nil {
		rc.logger.Error().
			Err(err).
			Str("pattern", fullPattern).
			Dur("duration", duration).
			Msg("Cache pattern keys error")
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	err = rc.client.Del(ctx, keys...).Err()
	if err != nil {
		rc.logger.Error().
			Err(err).
			Str("pattern", fullPattern).
			Int("key_count", len(keys)).
			Dur("duration", duration).
			Msg("Cache pattern delete error")
		return err
	}

	rc.logger.Info().
		Str("pattern", fullPattern).
		Int("key_count", len(keys)).
		Dur("duration", duration).
		Msg("Cache pattern invalidated")

	return nil
}

// GetStats returns cache statistics
func (rc *RedisCache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := rc.client.Info(ctx, "memory", "keyspace", "stats").Result()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"redis_info": info,
		"prefix":     rc.prefix,
		"default_ttl": rc.defaultTTL.String(),
	}, nil
}

// InMemoryCache implements QueryCache interface using in-memory storage
type InMemoryCache struct {
	cache map[string]*CacheEntry
	mutex sync.RWMutex
	ttl   time.Duration
}

// NewInMemoryCache creates a new in-memory query cache
func NewInMemoryCache(ttl time.Duration) *InMemoryCache {
	cache := &InMemoryCache{
		cache: make(map[string]*CacheEntry),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a value from in-memory cache
func (imc *InMemoryCache) Get(ctx context.Context, key string) (interface{}, bool) {
	imc.mutex.RLock()
	defer imc.mutex.RUnlock()

	entry, exists := imc.cache[key]
	if !exists {
		return nil, false
	}

	// Check if entry has expired
	if time.Since(entry.CreatedAt) > entry.TTL {
		imc.mutex.RUnlock()
		imc.mutex.Lock()
		delete(imc.cache, key)
		imc.mutex.Unlock()
		imc.mutex.RLock()
		return nil, false
	}

	return entry.Data, true
}

// Set stores a value in in-memory cache
func (imc *InMemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	if ttl <= 0 {
		ttl = imc.ttl
	}

	imc.mutex.Lock()
	defer imc.mutex.Unlock()

	imc.cache[key] = &CacheEntry{
		Data:      value,
		CreatedAt: time.Now(),
		TTL:       ttl,
	}
}

// Delete removes a value from in-memory cache
func (imc *InMemoryCache) Delete(ctx context.Context, key string) {
	imc.mutex.Lock()
	defer imc.mutex.Unlock()

	delete(imc.cache, key)
}

// cleanup removes expired entries from cache
func (imc *InMemoryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		imc.mutex.Lock()

		for key, entry := range imc.cache {
			if now.Sub(entry.CreatedAt) > entry.TTL {
				delete(imc.cache, key)
			}
		}

		imc.mutex.Unlock()
	}
}

// CachedRows wraps pgx.Rows for caching
type CachedRows struct {
	data     []map[string]interface{}
	position int
}

// NewCachedRows creates a new CachedRows instance
func NewCachedRows(data []map[string]interface{}) *CachedRows {
	return &CachedRows{
		data:     data,
		position: -1,
	}
}

// Next advances to the next row
func (cr *CachedRows) Next() bool {
	cr.position++
	return cr.position < len(cr.data)
}

// Values returns the values for the current row
func (cr *CachedRows) Values() ([]interface{}, error) {
	if cr.position < 0 || cr.position >= len(cr.data) {
		return nil, fmt.Errorf("no current row")
	}

	row := cr.data[cr.position]
	values := make([]interface{}, 0, len(row))
	for _, value := range row {
		values = append(values, value)
	}

	return values, nil
}

// Scan copies the column values into the provided values
func (cr *CachedRows) Scan(dest ...interface{}) error {
	if cr.position < 0 || cr.position >= len(cr.data) {
		return fmt.Errorf("no current row")
	}

	row := cr.data[cr.position]
	if len(row) != len(dest) {
		return fmt.Errorf("column count mismatch")
	}

	// Convert map values to slice to maintain order
	values := make([]interface{}, len(row))
	index := 0
	for _, value := range row {
		values[index] = value
		index++
	}

	for i, value := range values {
		switch d := dest[i].(type) {
		case *interface{}:
			*d = value
		case *string:
			if s, ok := value.(string); ok {
				*d = s
			}
		case *int:
			if i, ok := value.(int); ok {
				*d = i
			}
		case *int64:
			if i, ok := value.(int64); ok {
				*d = i
			}
		case *float64:
			if f, ok := value.(float64); ok {
				*d = f
			}
		case *bool:
			if b, ok := value.(bool); ok {
				*d = b
			}
		case *time.Time:
			if t, ok := value.(time.Time); ok {
				*d = t
			}
		case *[]byte:
			if b, ok := value.([]byte); ok {
				*d = b
			}
		default:
			// For other types, try to assign directly
			// This is a simplified implementation
			reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(value))
		}
	}

	return nil
}

// Close closes the rows
func (cr *CachedRows) Close() {
	// No-op for cached rows
}

// Err returns any error that occurred while scanning
func (cr *CachedRows) Err() error {
	return nil
}

// CommandTag returns the command tag from the query
func (cr *CachedRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

// FieldDescriptions returns the field descriptions for the query
func (cr *CachedRows) FieldDescriptions() []pgconn.FieldDescription {
	// This is a simplified implementation
	return nil
}

// Values returns the raw values for the current row
func (cr *CachedRows) RawValues() [][]byte {
	// This is a simplified implementation
	return nil
}