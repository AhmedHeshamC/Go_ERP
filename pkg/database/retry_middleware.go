package database

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

// RetryMiddleware provides middleware for database retry operations in HTTP handlers
type RetryMiddleware struct {
	retryManager *RetryManager
	logger       *zerolog.Logger
}

// NewRetryMiddleware creates a new retry middleware
func NewRetryMiddleware(config *RetryConfig, logger *zerolog.Logger) *RetryMiddleware {
	retryManager := NewRetryManager(config, logger)

	return &RetryMiddleware{
		retryManager: retryManager,
		logger:       logger,
	}
}

// WithRetry returns a Gin middleware that adds retry capabilities to handlers
func (rm *RetryMiddleware) WithRetry() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Store retry manager in context
		c.Set("retry_manager", rm.retryManager)
		c.Set("logger", rm.logger)

		c.Next()
	}
}

// ExecuteWithRetry executes a database operation within the context of an HTTP request
func (rm *RetryMiddleware) ExecuteWithRetry(c *gin.Context, operation func() error) error {
	retryManager, exists := c.Get("retry_manager")
	if !exists {
		return fmt.Errorf("retry manager not found in context")
	}

	manager := retryManager.(*RetryManager)

	// Create retry options with operation context
	options := WithRetryOptions(
		fmt.Sprintf("%s_%s", c.Request.Method, c.FullPath()),
		manager.config.MaxAttempts,
		manager.config.InitialDelay,
	).WithContext(c.Request.Context())

	// Add custom on retry callback for HTTP context
	options = options.WithOnRetry(func(attempt int, err error) {
		rm.logger.Warn().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("client_ip", c.ClientIP()).
			Int("attempt", attempt).
			Err(err).
			Msg("Retrying database operation in HTTP request")
	})

	retryResult := manager.ExecuteWithRetry(c.Request.Context(), operation, options)

	if !retryResult.Success {
		rm.logger.Error().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("client_ip", c.ClientIP()).
			Int("total_attempts", retryResult.Attempts).
			Dur("total_duration", retryResult.TotalDuration).
			Err(retryResult.FinalError).
			Msg("Database operation failed after retries in HTTP request")

		return retryResult.FinalError
	}

	return nil
}

// DatabaseRetryContext provides retry context for database operations
type DatabaseRetryContext struct {
	retryManager *RetryManager
	logger       *zerolog.Logger
}

// GetRetryContext extracts retry context from Gin context
func GetRetryContext(c *gin.Context) (*DatabaseRetryContext, error) {
	retryManager, exists := c.Get("retry_manager")
	if !exists {
		return nil, fmt.Errorf("retry manager not found in context")
	}

	logger, exists := c.Get("logger")
	if !exists {
		nopLogger := zerolog.Nop()
		logger = &nopLogger
	}

	return &DatabaseRetryContext{
		retryManager: retryManager.(*RetryManager),
		logger:       logger.(*zerolog.Logger),
	}, nil
}

// ExecWithRetry executes a database command with retry logic
func (drc *DatabaseRetryContext) ExecWithRetry(c *gin.Context, db *Database, query string, args ...interface{}) (pgconn.CommandTag, error) {
	var result pgconn.CommandTag

	operation := func() error {
		var err error
		result, err = db.Exec(c.Request.Context(), query, args...)
		return err
	}

	options := WithRetryOptions(
		fmt.Sprintf("exec_%s_%s", c.Request.Method, c.Request.URL.Path),
		drc.retryManager.config.MaxAttempts,
		drc.retryManager.config.InitialDelay,
	).WithContext(c.Request.Context())

	retryResult := drc.retryManager.ExecuteWithRetry(c.Request.Context(), operation, options)

	if !retryResult.Success {
		return pgconn.CommandTag{}, retryResult.FinalError
	}

	return result, nil
}

// QueryWithRetry executes a database query with retry logic
func (drc *DatabaseRetryContext) QueryWithRetry(c *gin.Context, db *Database, query string, args ...interface{}) (pgx.Rows, error) {
	var result pgx.Rows

	operation := func() error {
		var err error
		result, err = db.Query(c.Request.Context(), query, args...)
		return err
	}

	options := WithRetryOptions(
		fmt.Sprintf("query_%s_%s", c.Request.Method, c.Request.URL.Path),
		drc.retryManager.config.MaxAttempts,
		drc.retryManager.config.InitialDelay,
	).WithContext(c.Request.Context())

	retryResult := drc.retryManager.ExecuteWithRetry(c.Request.Context(), operation, options)

	if !retryResult.Success {
		return nil, retryResult.FinalError
	}

	return result, nil
}

// QueryRowWithRetry executes a database query that returns a single row with retry logic
func (drc *DatabaseRetryContext) QueryRowWithRetry(c *gin.Context, db *Database, query string, args ...interface{}) pgx.Row {
	var result pgx.Row

	operation := func() error {
		result = db.QueryRow(c.Request.Context(), query, args...)
		return nil // We can't test QueryRow without scanning
	}

	options := WithRetryOptions(
		fmt.Sprintf("query_row_%s_%s", c.Request.Method, c.Request.URL.Path),
		drc.retryManager.config.MaxAttempts,
		drc.retryManager.config.InitialDelay,
	).WithContext(c.Request.Context())

	retryResult := drc.retryManager.ExecuteWithRetry(c.Request.Context(), operation, options)

	if !retryResult.Success {
		return &errorRow{err: retryResult.FinalError}
	}

	return result
}

// WithTransactionRetry executes a function within a transaction with retry logic
func (drc *DatabaseRetryContext) WithTransactionRetry(c *gin.Context, db *Database, fn func(pgx.Tx) error) error {
	operation := func() error {
		return db.WithTransaction(c.Request.Context(), fn)
	}

	options := WithRetryOptions(
		fmt.Sprintf("transaction_%s_%s", c.Request.Method, c.Request.URL.Path),
		drc.retryManager.config.MaxAttempts,
		drc.retryManager.config.InitialDelay,
	).WithContext(c.Request.Context())

	retryResult := drc.retryManager.ExecuteWithRetry(c.Request.Context(), operation, options)

	if !retryResult.Success {
		return retryResult.FinalError
	}

	return nil
}

// Global retry middleware instance
var GlobalRetryMiddleware *RetryMiddleware

// InitializeRetryMiddleware initializes the global retry middleware
func InitializeRetryMiddleware(config *RetryConfig, logger *zerolog.Logger) {
	GlobalRetryMiddleware = NewRetryMiddleware(config, logger)
}

// GetRetryMiddleware returns the global retry middleware
func GetRetryMiddleware() *RetryMiddleware {
	if GlobalRetryMiddleware == nil {
		// Create with default config if not initialized
		nopLogger := zerolog.Nop()
		GlobalRetryMiddleware = NewRetryMiddleware(DefaultRetryConfig(), &nopLogger)
	}
	return GlobalRetryMiddleware
}

// Helper functions for common retry patterns

// RetryableQuery executes a query with standard retry configuration
func RetryableQuery(c *gin.Context, db *Database, query string, args ...interface{}) (pgx.Rows, error) {
	drc, err := GetRetryContext(c)
	if err != nil {
		return nil, err
	}
	return drc.QueryWithRetry(c, db, query, args...)
}

// RetryableExec executes a command with standard retry configuration
func RetryableExec(c *gin.Context, db *Database, query string, args ...interface{}) (pgconn.CommandTag, error) {
	drc, err := GetRetryContext(c)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	return drc.ExecWithRetry(c, db, query, args...)
}

// RetryableQueryRow executes a single-row query with standard retry configuration
func RetryableQueryRow(c *gin.Context, db *Database, query string, args ...interface{}) pgx.Row {
	drc, err := GetRetryContext(c)
	if err != nil {
		return &errorRow{err: err}
	}
	return drc.QueryRowWithRetry(c, db, query, args...)
}

// RetryableTransaction executes a transaction with standard retry configuration
func RetryableTransaction(c *gin.Context, db *Database, fn func(pgx.Tx) error) error {
	drc, err := GetRetryContext(c)
	if err != nil {
		return err
	}
	return drc.WithTransactionRetry(c, db, fn)
}

// RetryConfigBuilder helps build retry configurations for different scenarios
type RetryConfigBuilder struct {
	config *RetryConfig
}

// NewRetryConfigBuilder creates a new retry configuration builder
func NewRetryConfigBuilder() *RetryConfigBuilder {
	return &RetryConfigBuilder{
		config: DefaultRetryConfig(),
	}
}

// WithMaxAttempts sets the maximum number of retry attempts
func (b *RetryConfigBuilder) WithMaxAttempts(attempts int) *RetryConfigBuilder {
	b.config.MaxAttempts = attempts
	return b
}

// WithInitialDelay sets the initial delay between retries
func (b *RetryConfigBuilder) WithInitialDelay(delay time.Duration) *RetryConfigBuilder {
	b.config.InitialDelay = delay
	return b
}

// WithMaxDelay sets the maximum delay between retries
func (b *RetryConfigBuilder) WithMaxDelay(delay time.Duration) *RetryConfigBuilder {
	b.config.MaxDelay = delay
	return b
}

// WithMultiplier sets the backoff multiplier
func (b *RetryConfigBuilder) WithMultiplier(multiplier float64) *RetryConfigBuilder {
	b.config.Multiplier = multiplier
	return b
}

// WithJitter enables or disables jitter
func (b *RetryConfigBuilder) WithJitter(enabled bool, factor float64) *RetryConfigBuilder {
	b.config.Jitter = enabled
	b.config.JitterFactor = factor
	return b
}

// WithBackoffStrategy sets the backoff strategy
func (b *RetryConfigBuilder) WithBackoffStrategy(strategy BackoffStrategy) *RetryConfigBuilder {
	b.config.BackoffStrategy = strategy
	return b
}

// WithRetryConditions sets retry conditions
func (b *RetryConfigBuilder) WithRetryConditions(timeout, connectionLoss, deadlock, queryCancel bool) *RetryConfigBuilder {
	b.config.RetryOnTimeout = timeout
	b.config.RetryOnConnectionLoss = connectionLoss
	b.config.RetryOnDeadlock = deadlock
	b.config.RetryOnQueryCancel = queryCancel
	return b
}

// WithLogging configures logging settings
func (b *RetryConfigBuilder) WithLogging(retries, attempts, success bool) *RetryConfigBuilder {
	b.config.LogRetries = retries
	b.config.LogRetryAttempts = attempts
	b.config.LogSuccessfulRetry = success
	return b
}

// WithCircuitBreaker enables or disables circuit breaker
func (b *RetryConfigBuilder) WithCircuitBreaker(enabled bool, threshold int, timeout time.Duration) *RetryConfigBuilder {
	b.config.EnableCircuitBreaker = enabled
	b.config.CircuitBreakerThreshold = threshold
	b.config.CircuitBreakerTimeout = timeout
	return b
}

// Build builds the final retry configuration
func (b *RetryConfigBuilder) Build() *RetryConfig {
	return b.config
}

// Predefined retry configurations for different scenarios

// DefaultConfigForReadOperations returns a retry config optimized for read operations
func DefaultConfigForReadOperations() *RetryConfig {
	return NewRetryConfigBuilder().
		WithMaxAttempts(3).
		WithInitialDelay(50 * time.Millisecond).
		WithMaxDelay(2 * time.Second).
		WithMultiplier(1.5).
		WithJitter(true, 0.1).
		WithBackoffStrategy(BackoffStrategyExponential).
		WithRetryConditions(true, true, true, false).
		WithLogging(true, true, true).
		WithCircuitBreaker(true, 3, 15*time.Second).
		Build()
}

// DefaultConfigForWriteOperations returns a retry config optimized for write operations
func DefaultConfigForWriteOperations() *RetryConfig {
	return NewRetryConfigBuilder().
		WithMaxAttempts(2).
		WithInitialDelay(100 * time.Millisecond).
		WithMaxDelay(3 * time.Second).
		WithMultiplier(2.0).
		WithJitter(true, 0.2).
		WithBackoffStrategy(BackoffStrategyExponential).
		WithRetryConditions(true, true, false, false).
		WithLogging(true, true, true).
		WithCircuitBreaker(true, 5, 30*time.Second).
		Build()
}

// DefaultConfigForTransactions returns a retry config optimized for transactions
func DefaultConfigForTransactions() *RetryConfig {
	return NewRetryConfigBuilder().
		WithMaxAttempts(2).
		WithInitialDelay(200 * time.Millisecond).
		WithMaxDelay(5 * time.Second).
		WithMultiplier(2.0).
		WithJitter(true, 0.15).
		WithBackoffStrategy(BackoffStrategyExponential).
		WithRetryConditions(true, true, true, false).
		WithLogging(true, true, true).
		WithCircuitBreaker(true, 4, 45*time.Second).
		Build()
}

// DefaultConfigForBatchOperations returns a retry config optimized for batch operations
func DefaultConfigForBatchOperations() *RetryConfig {
	return NewRetryConfigBuilder().
		WithMaxAttempts(1). // Usually don't retry batch operations
		WithInitialDelay(500 * time.Millisecond).
		WithMaxDelay(10 * time.Second).
		WithMultiplier(1.0).
		WithJitter(false, 0).
		WithBackoffStrategy(BackoffStrategyFixed).
		WithRetryConditions(false, true, false, false).
		WithLogging(true, false, true).
		WithCircuitBreaker(false, 0, 0).
		Build()
}