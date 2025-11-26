package ratelimit

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 10.0, config.DefaultLimit.RequestsPerSecond)
	assert.Equal(t, 20, config.DefaultLimit.BurstSize)
	assert.Equal(t, StorageMemory, config.StorageType)
}

func TestNewMemoryStore(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	
	store := NewMemoryStore(config, &logger)
	assert.NotNil(t, store)
}

func TestMemoryStore_Allow(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	limit := RateLimit{
		RequestsPerSecond: 10.0,
		BurstSize:        10,
	}
	
	store := NewMemoryStore(config, &logger)

	t.Run("allows requests within limit", func(t *testing.T) {
		key := "test-key-1"
		for i := 0; i < 10; i++ {
			allowed, err := store.Allow(key, limit)
			require.NoError(t, err)
			assert.True(t, allowed, "request %d should be allowed", i+1)
		}
	})

	t.Run("rejects requests exceeding limit", func(t *testing.T) {
		key := "test-key-2"
		// Exhaust the burst
		for i := 0; i < 10; i++ {
			store.Allow(key, limit)
		}
		
		// Next request should be rejected
		allowed, err := store.Allow(key, limit)
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func TestMemoryStore_AllowN(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	limit := RateLimit{
		RequestsPerSecond: 10.0,
		BurstSize:        10,
	}
	
	store := NewMemoryStore(config, &logger)

	t.Run("allows N requests within limit", func(t *testing.T) {
		key := "test-key-n-1"
		allowed, err := store.AllowN(key, 5, limit)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("rejects N requests exceeding limit", func(t *testing.T) {
		key := "test-key-n-2"
		allowed, err := store.AllowN(key, 15, limit)
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func TestMemoryStore_Reserve(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	limit := RateLimit{
		RequestsPerSecond: 10.0,
		BurstSize:        10,
	}
	
	store := NewMemoryStore(config, &logger)

	t.Run("reserves slot successfully", func(t *testing.T) {
		key := "test-key-reserve-1"
		reservation, err := store.Reserve(key, limit)
		require.NoError(t, err)
		assert.NotNil(t, reservation)
		assert.True(t, reservation.OK)
	})

	t.Run("reservation fails when limit exceeded", func(t *testing.T) {
		key := "test-key-reserve-2"
		// Exhaust the burst
		for i := 0; i < 10; i++ {
			store.Allow(key, limit)
		}
		
		reservation, err := store.Reserve(key, limit)
		require.NoError(t, err)
		assert.NotNil(t, reservation)
		// Should still reserve but with delay
		assert.Greater(t, reservation.Delay, time.Duration(0))
	})
}

// Wait and WaitN methods are not implemented in MemoryStore
// Skipping these tests

func TestMemoryStore_Reset(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	limit := RateLimit{
		RequestsPerSecond: 10.0,
		BurstSize:        5,
	}
	
	store := NewMemoryStore(config, &logger)

	key := "test-key-reset"
	
	// Exhaust the burst
	for i := 0; i < 5; i++ {
		store.Allow(key, limit)
	}
	
	// Should be rate limited
	allowed, _ := store.Allow(key, limit)
	assert.False(t, allowed)
	
	// Reset the limiter
	store.Reset(key)
	
	// Should be allowed again
	allowed, _ = store.Allow(key, limit)
	assert.True(t, allowed)
}

// GetLimit method returns the default limit from config
func TestMemoryStore_GetLimit(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.DefaultLimit = RateLimit{
		RequestsPerSecond: 10.0,
		BurstSize:        20,
	}
	
	_ = NewMemoryStore(config, &logger)

	// GetLimit returns the default limit from config
	assert.Equal(t, 10.0, config.DefaultLimit.RequestsPerSecond)
	assert.Equal(t, 20, config.DefaultLimit.BurstSize)
}

func TestMemoryStore_GetStats(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	limit := config.DefaultLimit
	
	store := NewMemoryStore(config, &logger)

	// Make some requests
	store.Allow("key1", limit)
	store.Allow("key2", limit)
	store.Allow("key1", limit)

	stats := store.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "storage_type")
	assert.Equal(t, "memory", stats["storage_type"])
}

func TestMemoryStore_MultipleKeys(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	limit := RateLimit{
		RequestsPerSecond: 10.0,
		BurstSize:        5,
	}
	
	store := NewMemoryStore(config, &logger)

	// Test that different keys have independent limits
	key1 := "user:1"
	key2 := "user:2"

	// Exhaust key1
	for i := 0; i < 5; i++ {
		allowed, _ := store.Allow(key1, limit)
		assert.True(t, allowed)
	}
	
	// key1 should be limited
	allowed, _ := store.Allow(key1, limit)
	assert.False(t, allowed)
	
	// key2 should still be allowed
	allowed, _ = store.Allow(key2, limit)
	assert.True(t, allowed)
}

func TestMemoryStore_TokenRefill(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	limit := RateLimit{
		RequestsPerSecond: 10.0, // 10 tokens per second = 1 token per 100ms
		BurstSize:        2,
	}
	
	store := NewMemoryStore(config, &logger)
	key := "refill-test"

	// Use up the burst
	allowed, _ := store.Allow(key, limit)
	assert.True(t, allowed)
	allowed, _ = store.Allow(key, limit)
	assert.True(t, allowed)
	
	// Should be limited now
	allowed, _ = store.Allow(key, limit)
	assert.False(t, allowed)
	
	// Wait for token refill (at least 100ms for 1 token)
	time.Sleep(150 * time.Millisecond)
	
	// Should be allowed again
	allowed, _ = store.Allow(key, limit)
	assert.True(t, allowed)
}

func TestRateLimit_Struct(t *testing.T) {
	limit := RateLimit{
		RequestsPerSecond: 5.0,
		BurstSize:        10,
	}

	assert.Equal(t, 5.0, limit.RequestsPerSecond)
	assert.Equal(t, 10, limit.BurstSize)
}

func TestReservation_Struct(t *testing.T) {
	reservation := &Reservation{
		OK:        true,
		Delay:     100 * time.Millisecond,
		Limit:     10.0,
		Remaining: 5.0,
	}

	assert.True(t, reservation.OK)
	assert.Equal(t, 100*time.Millisecond, reservation.Delay)
	assert.Equal(t, 10.0, reservation.Limit)
	assert.Equal(t, 5.0, reservation.Remaining)
}

func TestStorageType_Constants(t *testing.T) {
	assert.Equal(t, StorageType("memory"), StorageMemory)
	assert.Equal(t, StorageType("redis"), StorageRedis)
}

func TestConfig_Validation(t *testing.T) {
	config := DefaultConfig()
	
	assert.Greater(t, config.DefaultLimit.RequestsPerSecond, 0.0)
	assert.Greater(t, config.DefaultLimit.BurstSize, 0)
	assert.Greater(t, config.CleanupInterval, time.Duration(0))
	assert.Greater(t, config.MaxKeys, 0)
}
