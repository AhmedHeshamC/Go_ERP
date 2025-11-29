package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

// TransactionManager manages database transactions with optimized boundaries
type TransactionManager struct {
	db             *Database
	logger         *zerolog.Logger
	defaultTimeout time.Duration
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(db *Database, logger *zerolog.Logger) *TransactionManager {
	if logger == nil {
		l := zerolog.Nop()
		logger = &l
	}

	return &TransactionManager{
		db:             db,
		logger:         logger,
		defaultTimeout: 30 * time.Second,
	}
}

// TransactionOptions holds configuration for transaction execution
type TransactionOptions struct {
	Timeout        time.Duration `json:"timeout"`
	IsolationLevel string        `json:"isolation_level"`
	ReadOnly       bool          `json:"read_only"`
	Deferrable     bool          `json:"deferrable"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryDelay     time.Duration `json:"retry_delay"`
}

// DefaultTransactionOptions returns sensible default transaction options
func DefaultTransactionOptions() TransactionOptions {
	return TransactionOptions{
		Timeout:        30 * time.Second,
		IsolationLevel: "READ COMMITTED",
		ReadOnly:       false,
		Deferrable:     false,
		RetryAttempts:  3,
		RetryDelay:     100 * time.Millisecond,
	}
}

// TransactionFunc represents a function that executes within a transaction
type TransactionFunc func(ctx context.Context, tx pgx.Tx) error

// Execute executes a function within a transaction with optimized boundaries
func (tm *TransactionManager) Execute(ctx context.Context, fn TransactionFunc) error {
	return tm.ExecuteWithOptions(ctx, DefaultTransactionOptions(), fn)
}

// ExecuteWithOptions executes a function within a transaction with custom options
func (tm *TransactionManager) ExecuteWithOptions(ctx context.Context, opts TransactionOptions, fn TransactionFunc) error {
	// Set timeout if not provided
	if opts.Timeout <= 0 {
		opts.Timeout = tm.defaultTimeout
	}

	// Create context with timeout
	txCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Begin transaction with appropriate options
	txOptions := tm.buildTxOptions(opts)
	tx, err := tm.db.BeginTx(txCtx, txOptions)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is handled properly
	defer func() {
		if p := recover(); p != nil {
			// Panic occurred, rollback transaction
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				tm.logger.Error().
					Err(rbErr).
					Msg("Failed to rollback transaction after panic")
			}
			panic(p) // Re-panic after cleanup
		}
	}()

	var lastErr error

	// Execute with retry logic
	for attempt := 0; attempt <= opts.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-txCtx.Done():
				return txCtx.Err()
			case <-time.After(opts.RetryDelay):
			}
		}

		// For retry attempts, we need a new transaction
		if attempt > 0 {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				tm.logger.Warn().
					Err(rbErr).
					Msg("Failed to rollback transaction for retry")
			}

			tx, err = tm.db.BeginTx(txCtx, txOptions)
			if err != nil {
				return fmt.Errorf("failed to begin retry transaction: %w", err)
			}
		}

		// Execute the transaction function
		err = fn(txCtx, tx)
		if err == nil {
			// Success - commit transaction
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return fmt.Errorf("failed to commit transaction: %w", commitErr)
			}

			tm.logger.Debug().
				Int("attempt", attempt+1).
				Msg("Transaction completed successfully")
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !tm.isRetryableError(err) || attempt == opts.RetryAttempts {
			break
		}

		tm.logger.Warn().
			Err(err).
			Int("attempt", attempt+1).
			Msg("Transaction failed, retrying")
	}

	// All attempts failed - rollback
	if rbErr := tx.Rollback(ctx); rbErr != nil {
		tm.logger.Error().
			Err(rbErr).
			Msg("Failed to rollback transaction after all attempts")
	}

	return fmt.Errorf("transaction failed after %d attempts: %w", opts.RetryAttempts+1, lastErr)
}

// buildTxOptions builds pgx.TxOptions from TransactionOptions
func (tm *TransactionManager) buildTxOptions(opts TransactionOptions) pgx.TxOptions {
	txOptions := pgx.TxOptions{}

	// Map isolation levels
	switch opts.IsolationLevel {
	case "READ UNCOMMITTED":
		txOptions.IsoLevel = pgx.ReadUncommitted
	case "READ COMMITTED":
		txOptions.IsoLevel = pgx.ReadCommitted
	case "REPEATABLE READ":
		txOptions.IsoLevel = pgx.RepeatableRead
	case "SERIALIZABLE":
		txOptions.IsoLevel = pgx.Serializable
	default:
		txOptions.IsoLevel = pgx.ReadCommitted
	}

	txOptions.AccessMode = pgx.ReadWrite
	if opts.ReadOnly {
		txOptions.AccessMode = pgx.ReadOnly
	}

	txOptions.DeferrableMode = pgx.NotDeferrable
	if opts.Deferrable {
		txOptions.DeferrableMode = pgx.Deferrable
	}

	return txOptions
}

// isRetryableError determines if a transaction error is retryable
func (tm *TransactionManager) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Common retryable errors
	retryableErrors := []string{
		"could not serialize access due to",
		"deadlock detected",
		"connection reset",
		"connection closed",
		"timeout",
		"network",
		"temporary failure",
	}

	for _, retryableErr := range retryableErrors {
		if contains(errStr, retryableErr) {
			return true
		}
	}

	return false
}

// ReadOnlyTransaction executes a read-only transaction
func (tm *TransactionManager) ReadOnlyTransaction(ctx context.Context, fn TransactionFunc) error {
	opts := DefaultTransactionOptions()
	opts.ReadOnly = true
	opts.IsolationLevel = "REPEATABLE READ" // Better for read-only consistency

	return tm.ExecuteWithOptions(ctx, opts, fn)
}

// WriteTransaction executes a write transaction with optimized settings
func (tm *TransactionManager) WriteTransaction(ctx context.Context, fn TransactionFunc) error {
	opts := DefaultTransactionOptions()
	opts.IsolationLevel = "READ COMMITTED" // Best for write performance
	opts.Timeout = 10 * time.Second        // Shorter timeout for writes

	return tm.ExecuteWithOptions(ctx, opts, fn)
}

// BatchTransaction executes a batch operation within a transaction
func (tm *TransactionManager) BatchTransaction(ctx context.Context, batchSize int, fn func(ctx context.Context, tx pgx.Tx, batchStart, batchEnd int) error) error {
	opts := DefaultTransactionOptions()
	opts.Timeout = 5 * time.Minute // Longer timeout for batch operations
	opts.IsolationLevel = "READ COMMITTED"

	return tm.ExecuteWithOptions(ctx, opts, func(ctx context.Context, tx pgx.Tx) error {
		// For simplicity, process one batch
		// In real implementation, you would loop through batches
		return fn(ctx, tx, 0, batchSize-1)
	})
}

// NestedTransaction simulates nested transactions using savepoints
func (tm *TransactionManager) NestedTransaction(ctx context.Context, tx pgx.Tx, savepointName string, fn TransactionFunc) error {
	// Create savepoint
	_, err := tx.Exec(ctx, fmt.Sprintf("SAVEPOINT %s", savepointName))
	if err != nil {
		return fmt.Errorf("failed to create savepoint %s: %w", savepointName, err)
	}

	// Execute nested operation
	err = fn(ctx, tx)
	if err != nil {
		// Rollback to savepoint
		_, rbErr := tx.Exec(ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", savepointName))
		if rbErr != nil {
			tm.logger.Error().
				Err(rbErr).
				Str("savepoint", savepointName).
				Msg("Failed to rollback to savepoint")
		}
		return err
	}

	// Release savepoint
	_, err = tx.Exec(ctx, fmt.Sprintf("RELEASE SAVEPOINT %s", savepointName))
	if err != nil {
		tm.logger.Warn().
			Err(err).
			Str("savepoint", savepointName).
			Msg("Failed to release savepoint")
	}

	return nil
}

// TransactionMetrics tracks transaction performance
type TransactionMetrics struct {
	TotalTransactions      int64         `json:"total_transactions"`
	SuccessfulTransactions int64         `json:"successful_transactions"`
	FailedTransactions     int64         `json:"failed_transactions"`
	RolledBackTransactions int64         `json:"rolled_back_transactions"`
	AvgDuration            time.Duration `json:"avg_duration"`
	MaxDuration            time.Duration `json:"max_duration"`
	MinDuration            time.Duration `json:"min_duration"`
	RetryCount             int64         `json:"retry_count"`
}

// MetricsTransactionManager wraps TransactionManager with metrics collection
type MetricsTransactionManager struct {
	*TransactionManager
	metrics TransactionMetrics
}

// NewMetricsTransactionManager creates a transaction manager with metrics
func NewMetricsTransactionManager(db *Database, logger *zerolog.Logger) *MetricsTransactionManager {
	return &MetricsTransactionManager{
		TransactionManager: NewTransactionManager(db, logger),
		metrics: TransactionMetrics{
			MinDuration: time.Hour, // Initialize with high value
		},
	}
}

// Execute executes a transaction with metrics collection
func (mtm *MetricsTransactionManager) Execute(ctx context.Context, fn TransactionFunc) error {
	start := time.Now()
	mtm.metrics.TotalTransactions++

	err := mtm.TransactionManager.Execute(ctx, fn)
	duration := time.Since(start)

	// Update metrics
	if err == nil {
		mtm.metrics.SuccessfulTransactions++
	} else {
		mtm.metrics.FailedTransactions++
	}

	// Update duration statistics
	if duration < mtm.metrics.MinDuration {
		mtm.metrics.MinDuration = duration
	}
	if duration > mtm.metrics.MaxDuration {
		mtm.metrics.MaxDuration = duration
	}

	// Calculate rolling average (simplified)
	if mtm.metrics.TotalTransactions > 0 {
		totalDuration := time.Duration(mtm.metrics.AvgDuration) * time.Duration(mtm.metrics.TotalTransactions-1)
		totalDuration += duration
		mtm.metrics.AvgDuration = totalDuration / time.Duration(mtm.metrics.TotalTransactions)
	}

	return err
}

// GetMetrics returns current transaction metrics
func (mtm *MetricsTransactionManager) GetMetrics() TransactionMetrics {
	return mtm.metrics
}

// ResetMetrics resets all transaction metrics
func (mtm *MetricsTransactionManager) ResetMetrics() {
	mtm.metrics = TransactionMetrics{
		MinDuration: time.Hour,
	}
}

// Utility function for string containment
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(hasPrefix(s, substr) || hasSuffix(s, substr) || indexOf(s, substr) >= 0))
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
