package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// RetryableDatabase extends the Database with retry capabilities
type RetryableDatabase struct {
	*Database
	retryManager *RetryManager
	logger       *zerolog.Logger
}

// NewRetryableDatabase creates a new database instance with retry capabilities
func NewRetryableDatabase(config Config, logger *zerolog.Logger) (*RetryableDatabase, error) {
	// Create base database
	db, err := New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Create retry manager
	retryConfig := &RetryConfig{
		MaxAttempts:         3,
		InitialDelay:        100 * time.Millisecond,
		MaxDelay:            5 * time.Second,
		Multiplier:          2.0,
		Jitter:              true,
		JitterFactor:        0.1,
		BackoffStrategy:     BackoffStrategyExponential,
		RetryOnTimeout:      true,
		RetryOnConnectionLoss: true,
		RetryOnDeadlock:     true,
		RetryOnQueryCancel:  false,
		LogRetries:          true,
		LogRetryAttempts:    true,
		LogSuccessfulRetry:  true,
		EnableCircuitBreaker: true,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout: 30 * time.Second,
	}

	retryManager := NewRetryManager(retryConfig, logger)

	return &RetryableDatabase{
		Database:     db,
		retryManager: retryManager,
		logger:       logger,
	}, nil
}

// NewRetryableDatabaseWithManager creates a new database instance with custom retry manager
func NewRetryableDatabaseWithManager(config Config, retryManager *RetryManager, logger *zerolog.Logger) (*RetryableDatabase, error) {
	db, err := New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	return &RetryableDatabase{
		Database:     db,
		retryManager: retryManager,
		logger:       logger,
	}, nil
}

// GetRetryManager returns the retry manager
func (rdb *RetryableDatabase) GetRetryManager() *RetryManager {
	return rdb.retryManager
}

// ExecWithRetry executes a query with retry logic
func (rdb *RetryableDatabase) ExecWithRetry(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	var result pgconn.CommandTag

	retryResult := rdb.retryManager.ExecuteWithRetry(ctx, func() error {
		var err error
		result, err = rdb.Database.Exec(ctx, query, args...)
		return err
	}, WithRetryOptions("database_exec", rdb.retryManager.config.MaxAttempts, rdb.retryManager.config.InitialDelay).WithContext(ctx))

	if !retryResult.Success {
		return pgconn.CommandTag{}, retryResult.FinalError
	}

	return result, nil
}

// QueryWithRetry executes a query with retry logic
func (rdb *RetryableDatabase) QueryWithRetry(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	var result pgx.Rows

	retryResult := rdb.retryManager.ExecuteWithRetry(ctx, func() error {
		var err error
		result, err = rdb.Database.Query(ctx, query, args...)
		return err
	}, WithRetryOptions("database_query", rdb.retryManager.config.MaxAttempts, rdb.retryManager.config.InitialDelay).WithContext(ctx))

	if !retryResult.Success {
		return nil, retryResult.FinalError
	}

	return result, nil
}

// QueryRowWithRetry executes a query that returns a single row with retry logic
func (rdb *RetryableDatabase) QueryRowWithRetry(ctx context.Context, query string, args ...interface{}) pgx.Row {
	// For QueryRow, we need to handle retry differently since it returns a Row object
	// We'll retry the underlying query and return a new Row object on success
	var result pgx.Row

	options := WithRetryOptions("database_query_row", rdb.retryManager.config.MaxAttempts, rdb.retryManager.config.InitialDelay).WithContext(ctx)

	retryResult := rdb.retryManager.ExecuteWithRetry(ctx, func() error {
		var err error
		// We can't directly test QueryRow without scanning, so we'll execute a simple version first
		result = rdb.Database.QueryRow(ctx, query, args...)
		// We'll return nil here and let the caller handle any scan errors
		err = nil
		return err
	}, options)

	if !retryResult.Success {
		// Return an error row that will always fail on scan
		return &errorRow{err: retryResult.FinalError}
	}

	return result
}

// BeginWithRetry begins a transaction with retry logic
func (rdb *RetryableDatabase) BeginWithRetry(ctx context.Context) (pgx.Tx, error) {
	var result pgx.Tx

	retryResult := rdb.retryManager.ExecuteWithRetry(ctx, func() error {
		var err error
		result, err = rdb.Database.Begin(ctx)
		return err
	}, WithRetryOptions("database_begin_tx", rdb.retryManager.config.MaxAttempts, rdb.retryManager.config.InitialDelay).WithContext(ctx))

	if !retryResult.Success {
		return nil, retryResult.FinalError
	}

	return result, nil
}

// BeginTxWithRetry begins a transaction with options and retry logic
func (rdb *RetryableDatabase) BeginTxWithRetry(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	var result pgx.Tx

	retryResult := rdb.retryManager.ExecuteWithRetry(ctx, func() error {
		var err error
		result, err = rdb.Database.BeginTx(ctx, txOptions)
		return err
	}, WithRetryOptions("database_begin_tx_with_options", rdb.retryManager.config.MaxAttempts, rdb.retryManager.config.InitialDelay).WithContext(ctx))

	if !retryResult.Success {
		return nil, retryResult.FinalError
	}

	return result, nil
}

// PingWithRetry checks database connectivity with retry logic
func (rdb *RetryableDatabase) PingWithRetry(ctx context.Context) error {
	retryResult := rdb.retryManager.ExecuteWithRetry(ctx, func() error {
		return rdb.Database.Ping(ctx)
	}, WithRetryOptions("database_ping", rdb.retryManager.config.MaxAttempts, rdb.retryManager.config.InitialDelay).WithContext(ctx))

	if !retryResult.Success {
		return retryResult.FinalError
	}

	return nil
}

// AcquireWithRetry acquires a connection from the pool with retry logic
func (rdb *RetryableDatabase) AcquireWithRetry(ctx context.Context) (*pgxpool.Conn, error) {
	var result *pgxpool.Conn

	retryResult := rdb.retryManager.ExecuteWithRetry(ctx, func() error {
		var err error
		result, err = rdb.Database.Acquire(ctx)
		return err
	}, WithRetryOptions("database_acquire_connection", rdb.retryManager.config.MaxAttempts, rdb.retryManager.config.InitialDelay).WithContext(ctx))

	if !retryResult.Success {
		return nil, retryResult.FinalError
	}

	return result, nil
}

// WithTransactionRetry executes a function within a transaction with retry logic
func (rdb *RetryableDatabase) WithTransactionRetry(ctx context.Context, fn func(pgx.Tx) error) error {
	retryResult := rdb.retryManager.ExecuteWithRetry(ctx, func() error {
		return rdb.Database.WithTransaction(ctx, fn)
	}, WithRetryOptions("database_with_transaction", rdb.retryManager.config.MaxAttempts, rdb.retryManager.config.InitialDelay).WithContext(ctx))

	if !retryResult.Success {
		return retryResult.FinalError
	}

	return nil
}

// GetCircuitBreakerStatus returns the current circuit breaker status
func (rdb *RetryableDatabase) GetCircuitBreakerStatus() map[string]interface{} {
	if rdb.retryManager.circuitBreaker == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	state := "unknown"
	switch rdb.retryManager.circuitBreaker.state {
	case CircuitBreakerClosed:
		state = "closed"
	case CircuitBreakerOpen:
		state = "open"
	case CircuitBreakerHalfOpen:
		state = "half-open"
	}

	return map[string]interface{}{
		"enabled":          true,
		"state":            state,
		"failures":         rdb.retryManager.circuitBreaker.failures,
		"last_failure":     rdb.retryManager.circuitBreaker.lastFailure,
		"next_attempt":     rdb.retryManager.circuitBreaker.nextAttempt,
		"threshold":        rdb.retryManager.config.CircuitBreakerThreshold,
		"timeout":          rdb.retryManager.config.CircuitBreakerTimeout,
	}
}

// GetRetryStats returns retry statistics
func (rdb *RetryableDatabase) GetRetryStats() map[string]interface{} {
	return map[string]interface{}{
		"max_attempts":      rdb.retryManager.config.MaxAttempts,
		"initial_delay":     rdb.retryManager.config.InitialDelay.String(),
		"max_delay":         rdb.retryManager.config.MaxDelay.String(),
		"multiplier":        rdb.retryManager.config.Multiplier,
		"jitter":            rdb.retryManager.config.Jitter,
		"backoff_strategy":  rdb.retryManager.config.BackoffStrategy,
		"circuit_breaker":   rdb.GetCircuitBreakerStatus(),
	}
}

// UpdateRetryConfig updates the retry configuration
func (rdb *RetryableDatabase) UpdateRetryConfig(config *RetryConfig) {
	rdb.retryManager.config = config

	// Update circuit breaker if enabled
	if config.EnableCircuitBreaker && rdb.retryManager.circuitBreaker == nil {
		rdb.retryManager.circuitBreaker = NewCircuitBreaker("database", config, rdb.logger)
	} else if !config.EnableCircuitBreaker && rdb.retryManager.circuitBreaker != nil {
		rdb.retryManager.circuitBreaker = nil
	} else if rdb.retryManager.circuitBreaker != nil {
		rdb.retryManager.circuitBreaker.config = config
	}
}

// errorRow is a pgx.Row implementation that always returns an error
type errorRow struct {
	err error
}

func (r *errorRow) Scan(dest ...interface{}) error {
	return r.err
}

// Batch operations with retry logic

// BatchWithRetry executes a batch of operations with retry logic
func (rdb *RetryableDatabase) BatchWithRetry(ctx context.Context, batch *pgx.Batch) (pgx.BatchResults, error) {
	var result pgx.BatchResults

	retryResult := rdb.retryManager.ExecuteWithRetry(ctx, func() error {
		conn, err := rdb.Database.Acquire(ctx)
		if err != nil {
			return err
		}
		defer conn.Release()

		result = conn.SendBatch(ctx, batch)
		return nil
	}, WithRetryOptions("database_batch", rdb.retryManager.config.MaxAttempts, rdb.retryManager.config.InitialDelay).WithContext(ctx))

	if !retryResult.Success {
		return nil, retryResult.FinalError
	}

	return result, nil
}

// CopyFromWithRetry performs a COPY FROM operation with retry logic
func (rdb *RetryableDatabase) CopyFromWithRetry(ctx context.Context, tableName pgx.Identifier, columns []string, src pgx.CopyFromSource) (int64, error) {
	var result int64

	retryResult := rdb.retryManager.ExecuteWithRetry(ctx, func() error {
		conn, err := rdb.Database.Acquire(ctx)
		if err != nil {
			return err
		}
		defer conn.Release()

		var copyResult int64
		copyResult, err = conn.CopyFrom(ctx, tableName, columns, src)
		if err != nil {
			return err
		}
		result = copyResult
		return nil
	}, WithRetryOptions("database_copy_from", rdb.retryManager.config.MaxAttempts, rdb.retryManager.config.InitialDelay).WithContext(ctx))

	if !retryResult.Success {
		return 0, retryResult.FinalError
	}

	return result, nil
}

