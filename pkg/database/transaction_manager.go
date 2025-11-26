package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

// TransactionManagerInterface defines the interface for transaction management
type TransactionManagerInterface interface {
	// WithTransaction executes a function within a transaction
	WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error
	
	// WithRetryTransaction executes a function within a transaction with retry logic
	WithRetryTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error
	
	// WithTransactionOptions executes a function within a transaction with custom options
	WithTransactionOptions(ctx context.Context, opts TransactionConfig, fn func(tx pgx.Tx) error) error
}

// TransactionConfig holds configuration for transaction execution
type TransactionConfig struct {
	MaxRetries      int
	RetryDelay      time.Duration
	IsolationLevel  pgx.TxIsoLevel
	Timeout         time.Duration
}

// DefaultTransactionConfig returns default transaction configuration
func DefaultTransactionConfig() TransactionConfig {
	return TransactionConfig{
		MaxRetries:     3,
		RetryDelay:     100 * time.Millisecond,
		IsolationLevel: pgx.ReadCommitted,
		Timeout:        30 * time.Second,
	}
}

// TransactionManagerImpl implements TransactionManagerInterface
type TransactionManagerImpl struct {
	db     *Database
	logger *zerolog.Logger
}

// NewTransactionManagerImpl creates a new transaction manager implementation
func NewTransactionManagerImpl(db *Database, logger *zerolog.Logger) *TransactionManagerImpl {
	if logger == nil {
		l := zerolog.Nop()
		logger = &l
	}

	return &TransactionManagerImpl{
		db:     db,
		logger: logger,
	}
}

// WithTransaction executes a function within a transaction
func (tm *TransactionManagerImpl) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return tm.WithTransactionOptions(ctx, DefaultTransactionConfig(), fn)
}

// WithRetryTransaction executes a function within a transaction with retry logic
func (tm *TransactionManagerImpl) WithRetryTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	config := DefaultTransactionConfig()
	config.MaxRetries = 3
	return tm.WithTransactionOptions(ctx, config, fn)
}

// WithTransactionOptions executes a function within a transaction with custom options
func (tm *TransactionManagerImpl) WithTransactionOptions(ctx context.Context, opts TransactionConfig, fn func(tx pgx.Tx) error) error {
	// Apply timeout if specified
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	var lastErr error
	
	for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
		// Apply retry delay with exponential backoff
		if attempt > 0 {
			var delay time.Duration
			if attempt < 30 { // Prevent overflow: 2^30 is already > 1 billion
				delay = time.Duration(1<<uint(attempt-1)) * opts.RetryDelay // #nosec G115 - Protected by attempt < 30 check
			} else {
				delay = 5 * time.Second
			}
			// Cap at 5 seconds
			if delay > 5*time.Second {
				delay = 5 * time.Second
			}

			tm.logger.Debug().
				Int("attempt", attempt+1).
				Dur("delay", delay).
				Msg("Retrying transaction after delay")

			select {
			case <-ctx.Done():
				return fmt.Errorf("transaction cancelled during retry: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		// Begin transaction
		txOpts := pgx.TxOptions{
			IsoLevel: opts.IsolationLevel,
		}

		tx, err := tm.db.BeginTx(ctx, txOpts)
		if err != nil {
			lastErr = fmt.Errorf("failed to begin transaction: %w", err)
			
			// Check if error is retryable
			if !tm.isRetryableError(err) || attempt == opts.MaxRetries {
				return lastErr
			}
			continue
		}

		// Execute transaction function with panic recovery
		err = tm.executeWithPanicRecovery(ctx, tx, fn)

		// Check for context cancellation
		if ctx.Err() != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				tm.logger.Error().
					Err(rbErr).
					Msg("Failed to rollback transaction after context cancellation")
			}
			return fmt.Errorf("transaction cancelled: %w", ctx.Err())
		}

		// If successful, commit
		if err == nil {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				lastErr = fmt.Errorf("failed to commit transaction: %w", commitErr)
				
				// Check if commit error is retryable
				if !tm.isRetryableError(commitErr) || attempt == opts.MaxRetries {
					return lastErr
				}
				continue
			}

			tm.logger.Debug().
				Int("attempt", attempt+1).
				Msg("Transaction committed successfully")
			return nil
		}

		// Transaction function returned error
		lastErr = err

		// Rollback transaction
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			tm.logger.Error().
				Err(rbErr).
				Err(err).
				Msg("Failed to rollback transaction after error")
		}

		// Check if error is retryable
		if !tm.isRetryableError(err) || attempt == opts.MaxRetries {
			tm.logger.Debug().
				Err(err).
				Int("attempt", attempt+1).
				Bool("retryable", tm.isRetryableError(err)).
				Msg("Transaction failed, not retrying")
			return fmt.Errorf("transaction failed after %d attempts: %w", attempt+1, lastErr)
		}

		tm.logger.Warn().
			Err(err).
			Int("attempt", attempt+1).
			Int("max_retries", opts.MaxRetries).
			Msg("Transaction failed, will retry")
	}

	return fmt.Errorf("transaction failed after %d attempts: %w", opts.MaxRetries+1, lastErr)
}

// executeWithPanicRecovery executes the transaction function with panic recovery
func (tm *TransactionManagerImpl) executeWithPanicRecovery(ctx context.Context, tx pgx.Tx, fn func(tx pgx.Tx) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// Rollback on panic
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				tm.logger.Error().
					Err(rbErr).
					Interface("panic", r).
					Msg("Failed to rollback transaction after panic")
			}

			tm.logger.Error().
				Interface("panic", r).
				Msg("Panic occurred during transaction execution")

			// Convert panic to error
			err = fmt.Errorf("panic during transaction: %v", r)
		}
	}()

	return fn(tx)
}

// isRetryableError determines if an error is retryable
func (tm *TransactionManagerImpl) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for PostgreSQL error codes
	var pgErr *pgconn.PgError
	if ok := errors.As(err, &pgErr); ok {
		switch pgErr.Code {
		case "40P01": // deadlock_detected
			return true
		case "40001": // serialization_failure
			return true
		case "08000", "08003", "08006": // connection errors
			return true
		}
	}

	// Check error message for retryable patterns
	errStr := err.Error()
	retryablePatterns := []string{
		"deadlock",
		"could not serialize access",
		"serialization failure",
		"connection reset",
		"connection closed",
		"connection failure",
		"connection does not exist",
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}
