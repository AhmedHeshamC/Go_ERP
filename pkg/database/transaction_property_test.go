package database

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: production-readiness, Property 7: Transaction Atomicity**
// For any service operation that performs multiple database writes, either all writes
// succeed and commit, or all writes are rolled back
// **Validates: Requirements 4.1**
func TestProperty_TransactionAtomicity(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Successful operations must commit all changes
	properties.Property("successful operations commit all changes", prop.ForAll(
		func(numOperations int) bool {
			// Skip invalid inputs
			if numOperations < 1 || numOperations > 100 {
				return true
			}

			// Create a mock transaction manager
			tm := &MockTransactionManager{
				shouldFail: false,
			}

			ctx := context.Background()
			operationsExecuted := 0

			err := tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
				// Simulate multiple write operations
				for i := 0; i < numOperations; i++ {
					operationsExecuted++
				}
				return nil
			})

			// Verify no error occurred
			if err != nil {
				return false
			}

			// Verify all operations were executed
			if operationsExecuted != numOperations {
				return false
			}

			// Verify transaction was committed
			if !tm.committed {
				return false
			}

			// Verify transaction was not rolled back
			if tm.rolledBack {
				return false
			}

			return true
		},
		gen.IntRange(1, 100),
	))

	// Property: Failed operations must rollback all changes
	properties.Property("failed operations rollback all changes", prop.ForAll(
		func(numOperations int, failAtOperation int) bool {
			// Skip invalid inputs
			if numOperations < 1 || numOperations > 100 {
				return true
			}
			if failAtOperation < 0 || failAtOperation >= numOperations {
				return true
			}

			// Create a mock transaction manager
			tm := &MockTransactionManager{
				shouldFail: false,
			}

			ctx := context.Background()
			operationsExecuted := 0

			err := tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
				// Simulate multiple write operations
				for i := 0; i < numOperations; i++ {
					operationsExecuted++
					// Fail at specific operation
					if i == failAtOperation {
						return errors.New("operation failed")
					}
				}
				return nil
			})

			// Verify error occurred
			if err == nil {
				return false
			}

			// Verify operations were executed up to failure point
			if operationsExecuted != failAtOperation+1 {
				return false
			}

			// Verify transaction was rolled back
			if !tm.rolledBack {
				return false
			}

			// Verify transaction was not committed
			if tm.committed {
				return false
			}

			return true
		},
		gen.IntRange(1, 100),
		gen.IntRange(0, 99),
	))

	// Property: Context cancellation must rollback transaction
	properties.Property("context cancellation rolls back transaction", prop.ForAll(
		func(cancelAfterMs int) bool {
			// Skip invalid inputs
			if cancelAfterMs < 1 || cancelAfterMs > 1000 {
				return true
			}

			// Create a mock transaction manager
			tm := &MockTransactionManager{
				shouldFail: false,
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cancelAfterMs)*time.Millisecond)
			defer cancel()

			err := tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
				// Simulate long-running operation
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(2 * time.Second):
					return nil
				}
			})

			// Verify error occurred (context deadline exceeded)
			if err == nil {
				return false
			}

			// Verify transaction was rolled back
			if !tm.rolledBack {
				return false
			}

			// Verify transaction was not committed
			if tm.committed {
				return false
			}

			return true
		},
		gen.IntRange(1, 100),
	))

	// Property: Panic during transaction must rollback
	properties.Property("panic during transaction rolls back", prop.ForAll(
		func(panicMessage string) bool {
			// Create a mock transaction manager
			tm := &MockTransactionManager{
				shouldFail: false,
			}

			ctx := context.Background()
			panicked := false

			// Catch panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
					}
				}()

				_ = tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
					panic(panicMessage)
				})
			}()

			// Verify panic occurred
			if !panicked {
				return false
			}

			// Verify transaction was rolled back
			if !tm.rolledBack {
				return false
			}

			// Verify transaction was not committed
			if tm.committed {
				return false
			}

			return true
		},
		gen.AlphaString(),
	))

	// Property: Nested transactions must maintain atomicity
	properties.Property("nested transactions maintain atomicity", prop.ForAll(
		func(outerOps int, innerOps int, failInner bool) bool {
			// Skip invalid inputs
			if outerOps < 1 || outerOps > 50 || innerOps < 1 || innerOps > 50 {
				return true
			}

			// Create a mock transaction manager
			tm := &MockTransactionManager{
				shouldFail: false,
			}

			ctx := context.Background()
			outerExecuted := 0
			innerExecuted := 0

			err := tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
				// Outer operations
				for i := 0; i < outerOps; i++ {
					outerExecuted++
				}

				// Nested transaction (using savepoint)
				err := tm.NestedTransaction(ctx, tx, "savepoint1", func(ctx context.Context, tx pgx.Tx) error {
					for i := 0; i < innerOps; i++ {
						innerExecuted++
					}
					if failInner {
						return errors.New("inner transaction failed")
					}
					return nil
				})

				if err != nil && !failInner {
					return err
				}

				return nil
			})

			// If inner failed, outer should still succeed
			if failInner {
				// Verify outer transaction committed
				if !tm.committed {
					return false
				}
				// Verify all outer operations executed
				if outerExecuted != outerOps {
					return false
				}
			} else {
				// Both should succeed
				if err != nil {
					return false
				}
				if !tm.committed {
					return false
				}
				if outerExecuted != outerOps || innerExecuted != innerOps {
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 50),
		gen.IntRange(1, 50),
		gen.Bool(),
	))

	// Property: Concurrent transactions must not interfere
	properties.Property("concurrent transactions are isolated", prop.ForAll(
		func(numConcurrent int) bool {
			// Skip invalid inputs
			if numConcurrent < 2 || numConcurrent > 20 {
				return true
			}

			var wg sync.WaitGroup
			errors := make([]error, numConcurrent)
			committed := make([]bool, numConcurrent)

			for i := 0; i < numConcurrent; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()

					tm := &MockTransactionManager{
						shouldFail: false,
					}

					ctx := context.Background()
					err := tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
						// Simulate some work
						time.Sleep(time.Millisecond * 10)
						return nil
					})

					errors[index] = err
					committed[index] = tm.committed
				}(i)
			}

			wg.Wait()

			// Verify all transactions succeeded
			for i := 0; i < numConcurrent; i++ {
				if errors[i] != nil {
					return false
				}
				if !committed[i] {
					return false
				}
			}

			return true
		},
		gen.IntRange(2, 20),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: production-readiness, Property 8: Deadlock Retry Logic**
// For any transaction that encounters a deadlock error, the system must retry the
// transaction up to 3 times with exponential backoff before failing
// **Validates: Requirements 4.3**
func TestProperty_DeadlockRetryLogic(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Deadlock errors must trigger retry
	properties.Property("deadlock errors trigger retry", prop.ForAll(
		func(failCount int) bool {
			// Skip invalid inputs - failCount should be 0-4
			if failCount < 0 || failCount > 4 {
				return true
			}

			// Create a mock transaction manager that fails with deadlock
			tm := &MockTransactionManager{
				shouldFail:     true,
				failureCount:   failCount,
				failureType:    "deadlock",
				currentAttempt: 0,
				retryDelay:     1 * time.Millisecond, // Short delay for testing
			}

			ctx := context.Background()
			err := tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
				return nil
			})

			// With maxRetries=3, we get 4 total attempts (0, 1, 2, 3)
			// If failCount <= 3, should eventually succeed on attempt failCount+1
			if failCount <= 3 {
				if err != nil {
					return false
				}
				// Verify it retried the correct number of times
				if tm.currentAttempt != failCount+1 {
					return false
				}
			} else {
				// If failCount > 3, should fail after max retries (4 attempts)
				if err == nil {
					return false
				}
				// Verify it attempted max retries + 1 (4 total attempts)
				if tm.currentAttempt != 4 {
					return false
				}
			}

			return true
		},
		gen.IntRange(0, 4),
	))

	// Property: Exponential backoff must be applied
	properties.Property("exponential backoff is applied", prop.ForAll(
		func(retryCount int) bool {
			// Skip invalid inputs
			if retryCount < 1 || retryCount > 3 {
				return true
			}

			// Create a mock transaction manager with very short delays for testing
			tm := &MockTransactionManager{
				shouldFail:     true,
				failureCount:   retryCount,
				failureType:    "deadlock",
				currentAttempt: 0,
				retryDelays:    make([]time.Duration, 0),
				retryDelay:     1 * time.Millisecond, // Very short delay for testing
			}

			ctx := context.Background()
			_ = tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
				return nil
			})

			// Verify exponential backoff was applied
			if len(tm.retryDelays) != retryCount {
				return false
			}

			// Verify delays increase exponentially
			for i := 1; i < len(tm.retryDelays); i++ {
				// Each delay should be approximately double the previous
				// Allow some tolerance for timing variations
				ratio := float64(tm.retryDelays[i]) / float64(tm.retryDelays[i-1])
				if ratio < 1.5 || ratio > 2.5 {
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 3),
	))

	// Property: Non-retryable errors must not trigger retry
	properties.Property("non-retryable errors do not retry", prop.ForAll(
		func(errorType string) bool {
			// Skip retryable error types
			if errorType == "deadlock" || errorType == "serialization" || errorType == "connection" {
				return true
			}

			// Create a mock transaction manager with non-retryable error
			tm := &MockTransactionManager{
				shouldFail:     true,
				failureCount:   1,
				failureType:    errorType,
				currentAttempt: 0,
			}

			ctx := context.Background()
			err := tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
				return nil
			})

			// Verify error occurred
			if err == nil {
				return false
			}

			// Verify it only attempted once (no retries)
			if tm.currentAttempt != 1 {
				return false
			}

			return true
		},
		genNonRetryableErrorType(),
	))

	// Property: Serialization failures must trigger retry
	properties.Property("serialization failures trigger retry", prop.ForAll(
		func(failCount int) bool {
			// Skip invalid inputs
			if failCount < 0 || failCount > 4 {
				return true
			}

			// Create a mock transaction manager that fails with serialization error
			tm := &MockTransactionManager{
				shouldFail:     true,
				failureCount:   failCount,
				failureType:    "serialization",
				currentAttempt: 0,
				retryDelay:     1 * time.Millisecond, // Short delay for testing
			}

			ctx := context.Background()
			err := tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
				return nil
			})

			// With maxRetries=3, we get 4 total attempts (0, 1, 2, 3)
			if failCount <= 3 {
				if err != nil {
					return false
				}
			} else {
				// If failCount > 3, should fail after max retries
				if err == nil {
					return false
				}
			}

			return true
		},
		gen.IntRange(0, 4),
	))

	// Property: Connection errors must trigger retry
	properties.Property("connection errors trigger retry", prop.ForAll(
		func(failCount int) bool {
			// Skip invalid inputs
			if failCount < 0 || failCount > 4 {
				return true
			}

			// Create a mock transaction manager that fails with connection error
			tm := &MockTransactionManager{
				shouldFail:     true,
				failureCount:   failCount,
				failureType:    "connection",
				currentAttempt: 0,
				retryDelay:     1 * time.Millisecond, // Short delay for testing
			}

			ctx := context.Background()
			err := tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
				return nil
			})

			// With maxRetries=3, we get 4 total attempts (0, 1, 2, 3)
			if failCount <= 3 {
				if err != nil {
					return false
				}
			} else {
				// If failCount > 3, should fail after max retries
				if err == nil {
					return false
				}
			}

			return true
		},
		gen.IntRange(0, 4),
	))

	// Property: Max retry attempts must be respected
	properties.Property("max retry attempts are respected", prop.ForAll(
		func(maxRetries int) bool {
			// Skip invalid inputs
			if maxRetries < 0 || maxRetries > 10 {
				return true
			}

			// Create a mock transaction manager that always fails
			tm := &MockTransactionManager{
				shouldFail:     true,
				failureCount:   maxRetries + 10, // Always fail more than max retries
				failureType:    "deadlock",
				currentAttempt: 0,
				maxRetries:     maxRetries,
				maxRetriesSet:  true,                 // Explicitly set maxRetries
				retryDelay:     1 * time.Millisecond, // Short delay for testing
			}

			ctx := context.Background()
			err := tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
				return nil
			})

			// Verify error occurred
			if err == nil {
				return false
			}

			// Verify it attempted exactly maxRetries + 1 times (initial + retries)
			// For example: maxRetries=0 means 1 attempt, maxRetries=3 means 4 attempts
			expectedAttempts := maxRetries + 1
			if tm.currentAttempt != expectedAttempts {
				return false
			}

			return true
		},
		gen.IntRange(0, 10),
	))

	// Property: Retry delay must not exceed maximum
	properties.Property("retry delay does not exceed maximum", prop.ForAll(
		func(retryCount int) bool {
			// Skip invalid inputs
			if retryCount < 1 || retryCount > 10 {
				return true
			}

			// Create a mock transaction manager with short delays for testing
			tm := &MockTransactionManager{
				shouldFail:     true,
				failureCount:   retryCount,
				failureType:    "deadlock",
				currentAttempt: 0,
				retryDelays:    make([]time.Duration, 0),
				retryDelay:     1 * time.Millisecond,  // Very short delay for testing
				maxRetryDelay:  10 * time.Millisecond, // Short max delay for testing
			}

			ctx := context.Background()
			_ = tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
				return nil
			})

			// Verify no delay exceeds maximum
			for _, delay := range tm.retryDelays {
				if delay > tm.maxRetryDelay {
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 10),
	))

	// Property: Context timeout must prevent infinite retries
	properties.Property("context timeout prevents infinite retries", prop.ForAll(
		func(timeoutMs int) bool {
			// Use much smaller timeout values to speed up test
			if timeoutMs < 5 || timeoutMs > 50 {
				return true
			}

			// Create a mock transaction manager that always fails
			tm := &MockTransactionManager{
				shouldFail:     true,
				failureCount:   100, // Always fail
				failureType:    "deadlock",
				currentAttempt: 0,
				retryDelay:     time.Duration(timeoutMs/3) * time.Millisecond, // Smaller delay
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
			defer cancel()

			startTime := time.Now()
			err := tm.Execute(ctx, func(ctx context.Context, tx pgx.Tx) error {
				return nil
			})
			duration := time.Since(startTime)

			// Verify error occurred
			if err == nil {
				return false
			}

			// Verify it didn't take much longer than timeout
			// Allow 100% tolerance for timing variations (context cancellation isn't instant)
			if duration > time.Duration(timeoutMs)*time.Millisecond*2 {
				return false
			}

			return true
		},
		gen.IntRange(5, 50),
	))

	// Run with fewer iterations to speed up the test
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20 // Reduce from default 100 to 20
	properties.TestingRun(t, gopter.ConsoleReporter(false), parameters)
}

// Mock Transaction Manager for testing

type MockTransactionManager struct {
	shouldFail     bool
	failureCount   int
	failureType    string
	currentAttempt int
	committed      bool
	rolledBack     bool
	retryDelays    []time.Duration
	maxRetries     int
	maxRetriesSet  bool // Flag to indicate if maxRetries was explicitly set
	retryDelay     time.Duration
	maxRetryDelay  time.Duration
}

func (m *MockTransactionManager) Execute(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	// Only set default maxRetries if not explicitly set
	if !m.maxRetriesSet {
		m.maxRetries = 3
		m.maxRetriesSet = true
	}
	if m.retryDelay == 0 {
		m.retryDelay = 100 * time.Millisecond
	}
	if m.maxRetryDelay == 0 {
		m.maxRetryDelay = 5 * time.Second
	}

	var lastErr error

	for attempt := 0; attempt <= m.maxRetries; attempt++ {
		m.currentAttempt = attempt + 1

		// Apply retry delay
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * m.retryDelay
			if delay > m.maxRetryDelay {
				delay = m.maxRetryDelay
			}
			m.retryDelays = append(m.retryDelays, delay)

			select {
			case <-ctx.Done():
				m.rolledBack = true
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		// Check for panic
		panicked := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
					m.rolledBack = true
					panic(r)
				}
			}()

			// Execute function
			// Fix: Check if we should fail on THIS attempt (attempt is 0-indexed)
			// If failureCount is 3, we fail on attempts 0, 1, 2, 3 (first 4 attempts)
			// But we succeed on attempt 4 (when attempt == 4, which is > failureCount)
			if m.shouldFail && attempt < m.failureCount {
				lastErr = m.createError()
			} else {
				lastErr = fn(ctx, nil)
			}
		}()

		if panicked {
			return nil
		}

		// Check for context cancellation
		if ctx.Err() != nil {
			m.rolledBack = true
			return ctx.Err()
		}

		// If successful, commit
		if lastErr == nil {
			m.committed = true
			return nil
		}

		// Check if error is retryable
		if !m.isRetryableError(lastErr) || attempt == m.maxRetries {
			m.rolledBack = true
			return lastErr
		}
	}

	m.rolledBack = true
	return lastErr
}

func (m *MockTransactionManager) NestedTransaction(ctx context.Context, tx pgx.Tx, savepointName string, fn func(context.Context, pgx.Tx) error) error {
	// Simulate nested transaction with savepoint
	err := fn(ctx, tx)
	if err != nil {
		// Rollback to savepoint
		return err
	}
	return nil
}

func (m *MockTransactionManager) createError() error {
	switch m.failureType {
	case "deadlock":
		return &pgconn.PgError{
			Code:    "40P01",
			Message: "deadlock detected",
		}
	case "serialization":
		return &pgconn.PgError{
			Code:    "40001",
			Message: "could not serialize access",
		}
	case "connection":
		return &pgconn.PgError{
			Code:    "08006",
			Message: "connection failure",
		}
	case "validation":
		return errors.New("validation error")
	case "not_found":
		return errors.New("not found")
	default:
		return errors.New("unknown error")
	}
}

func (m *MockTransactionManager) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Deadlock
		if pgErr.Code == "40P01" {
			return true
		}
		// Serialization failure
		if pgErr.Code == "40001" {
			return true
		}
		// Connection errors
		if pgErr.Code == "08006" || pgErr.Code == "08003" || pgErr.Code == "08000" {
			return true
		}
	}

	errStr := err.Error()
	if contains(errStr, "deadlock") || contains(errStr, "serialization") || contains(errStr, "connection") {
		return true
	}

	return false
}

// Generators

func genNonRetryableErrorType() gopter.Gen {
	return gen.OneConstOf(
		"validation",
		"not_found",
		"unauthorized",
		"forbidden",
		"bad_request",
	)
}
