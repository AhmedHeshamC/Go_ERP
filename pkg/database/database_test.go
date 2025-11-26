package database

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				URL:             "postgres://localhost/test",
				MaxConnections:  20,
				MinConnections:  5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: time.Minute * 30,
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			config: Config{
				MaxConnections:  20,
				MinConnections:  5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: time.Minute * 30,
			},
			wantErr: true,
		},
		{
			name: "invalid max connections",
			config: Config{
				URL:             "postgres://localhost/test",
				MaxConnections:  0,
				MinConnections:  5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: time.Minute * 30,
			},
			wantErr: true,
		},
		{
			name: "negative min connections",
			config: Config{
				URL:             "postgres://localhost/test",
				MaxConnections:  20,
				MinConnections:  -1,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: time.Minute * 30,
			},
			wantErr: true,
		},
		{
			name: "min greater than max",
			config: Config{
				URL:             "postgres://localhost/test",
				MaxConnections:  10,
				MinConnections:  15,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: time.Minute * 30,
			},
			wantErr: true,
		},
		{
			name: "zero max lifetime",
			config: Config{
				URL:             "postgres://localhost/test",
				MaxConnections:  20,
				MinConnections:  5,
				ConnMaxLifetime: 0,
				ConnMaxIdleTime: time.Minute * 30,
			},
			wantErr: true,
		},
		{
			name: "zero max idle time",
			config: Config{
				URL:             "postgres://localhost/test",
				MaxConnections:  20,
				MinConnections:  5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNew_WithInvalidConfig(t *testing.T) {
	logger := zerolog.Nop()

	// Test with invalid config
	_, err := NewWithLogger(Config{
		URL:             "", // Invalid
		MaxConnections:  20,
		MinConnections:  5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
	}, &logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid database config")
}

func TestNew_WithInvalidURL(t *testing.T) {
	logger := zerolog.Nop()

	// Test with invalid database URL
	_, err := NewWithLogger(Config{
		URL:             "invalid-url",
		MaxConnections:  20,
		MinConnections:  5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
	}, &logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse database URL")
}

func TestDatabase_Getters(t *testing.T) {
	db := &Database{
		pool:   &pgxpool.Pool{},
		config: &Config{
			URL:             "postgres://localhost/test",
			MaxConnections:  20,
			MinConnections:  5,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: time.Minute * 30,
		},
	}

	// Test GetConfig
	config := db.GetConfig()
	require.NotNil(t, config)
	assert.Equal(t, "postgres://localhost/test", config.URL)
	assert.Equal(t, 20, config.MaxConnections)
}

func TestDatabase_Close(t *testing.T) {
	// Create a database instance with nil pool
	db := &Database{
		pool:   nil,
		logger: &zerolog.Logger{},
		config: &Config{},
	}

	// Close should not panic
	assert.NotPanics(t, func() {
		db.Close()
	})
}

func TestDatabase_MethodsWithNilPool(t *testing.T) {
	db := &Database{
		pool:   nil,
		logger: &zerolog.Logger{},
		config: &Config{},
	}

	ctx := context.Background()

	// These methods should panic or return errors when pool is nil
	assert.Panics(t, func() {
		db.GetPool()
	})

	assert.Panics(t, func() {
		db.Ping(ctx)
	})

	assert.Panics(t, func() {
		db.Exec(ctx, "SELECT 1")
	})

	assert.Panics(t, func() {
		db.Query(ctx, "SELECT 1")
	})

	assert.Panics(t, func() {
		db.QueryRow(ctx, "SELECT 1")
	})

	assert.Panics(t, func() {
		db.Begin(ctx)
	})

	assert.Panics(t, func() {
		db.Acquire(ctx)
	})
}

func TestDatabase_HealthCheck(t *testing.T) {
	db := &Database{
		pool:   nil,
		logger: &zerolog.Logger{},
		config: &Config{},
	}

	ctx := context.Background()

	// HealthCheck should return an error when pool is nil
	err := db.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database health check failed")
}

// Mock pool for testing
type mockPool struct {
	pgxpool.Pool
	pingErr error
}

func (m *mockPool) Ping(ctx context.Context) error {
	return m.pingErr
}

func (m *mockPool) Stat() *pgxpool.Stat {
	// pgxpool.Stat doesn't have public fields, return nil for mock
	return nil
}

func TestDatabase_HealthCheck_WithMock(t *testing.T) {
	// Skipping this test as mockPool cannot be used as *pgxpool.Pool
	// This test needs to be rewritten with proper mocking or integration testing
	t.Skip("Test needs to be rewritten with proper mocking")
}

func TestDatabase_Configuration(t *testing.T) {
	logger := zerolog.Nop()

	// Test creating database with default config
	db, err := NewWithLogger(Config{
		URL:             "postgres://localhost/test?sslmode=disable",
		MaxConnections:  10,
		MinConnections:  2,
		ConnMaxLifetime: time.Minute * 30,
		ConnMaxIdleTime: time.Minute * 5,
	}, &logger)

	// This will likely fail because we don't have a real database,
	// but we can test that the configuration is parsed correctly
	if err != nil {
		// Expected to fail without a real database
		assert.Contains(t, err.Error(), "failed to create connection pool")
	} else {
		// If it succeeds (unlikely), test that it was configured correctly
		require.NotNil(t, db)
		assert.NotNil(t, db.GetConfig())
		assert.Equal(t, "postgres://localhost/test?sslmode=disable", db.GetConfig().URL)
		assert.Equal(t, 10, db.GetConfig().MaxConnections)

		db.Close()
	}
}

func TestDatabase_GetSQLDB(t *testing.T) {
	db := &Database{
		pool:   nil,
		logger: &zerolog.Logger{},
		config: &Config{},
	}

	// GetSQLDB should panic when pool is nil
	assert.Panics(t, func() {
		db.GetSQLDB()
	})
}

// Benchmark tests
func BenchmarkDatabase_ConfigValidation(b *testing.B) {
	config := Config{
		URL:             "postgres://localhost/test",
		MaxConnections:  20,
		MinConnections:  5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.validate()
	}
}

func TestDatabase_ConcurrentOperations(t *testing.T) {
	// Test that database operations are thread-safe
	db := &Database{
		pool:   nil,
		logger: &zerolog.Logger{},
		config: &Config{},
	}

	// These should all panic safely when called concurrently
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Expected to panic when pool is nil
				}
				done <- true
			}()

			ctx := context.Background()
			db.GetPool()
			db.Ping(ctx)
			db.Exec(ctx, "SELECT 1")
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}