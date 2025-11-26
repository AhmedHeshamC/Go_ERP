package user

import (
	"context"

	"github.com/jackc/pgx/v5"

	"erpgo/pkg/database"
)

// MockTransactionManager is a mock implementation of TransactionManagerInterface for testing
type MockTransactionManager struct {
	WithTransactionFunc        func(ctx context.Context, fn func(tx pgx.Tx) error) error
	WithRetryTransactionFunc   func(ctx context.Context, fn func(tx pgx.Tx) error) error
	WithTransactionOptionsFunc func(ctx context.Context, opts database.TransactionConfig, fn func(tx pgx.Tx) error) error
}

// WithTransaction executes a function within a transaction (mock)
func (m *MockTransactionManager) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	if m.WithTransactionFunc != nil {
		return m.WithTransactionFunc(ctx, fn)
	}
	// Default behavior: just execute the function without a real transaction
	return fn(nil)
}

// WithRetryTransaction executes a function within a transaction with retry logic (mock)
func (m *MockTransactionManager) WithRetryTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	if m.WithRetryTransactionFunc != nil {
		return m.WithRetryTransactionFunc(ctx, fn)
	}
	// Default behavior: just execute the function without a real transaction
	return fn(nil)
}

// WithTransactionOptions executes a function within a transaction with custom options (mock)
func (m *MockTransactionManager) WithTransactionOptions(ctx context.Context, opts database.TransactionConfig, fn func(tx pgx.Tx) error) error {
	if m.WithTransactionOptionsFunc != nil {
		return m.WithTransactionOptionsFunc(ctx, opts, fn)
	}
	// Default behavior: just execute the function without a real transaction
	return fn(nil)
}
