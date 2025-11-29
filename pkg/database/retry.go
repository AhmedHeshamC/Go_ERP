package database

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

// RetryConfig defines configuration for database retry logic
type RetryConfig struct {
	// Basic retry settings
	MaxAttempts  int           `json:"max_attempts"`
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
	Multiplier   float64       `json:"multiplier"`
	Jitter       bool          `json:"jitter"`
	JitterFactor float64       `json:"jitter_factor"`

	// Backoff strategy
	BackoffStrategy BackoffStrategy `json:"backoff_strategy"`

	// Retry conditions
	RetryOnTimeout        bool `json:"retry_on_timeout"`
	RetryOnConnectionLoss bool `json:"retry_on_connection_loss"`
	RetryOnDeadlock       bool `json:"retry_on_deadlock"`
	RetryOnQueryCancel    bool `json:"retry_on_query_cancel"`

	// Logging settings
	LogRetries         bool `json:"log_retries"`
	LogRetryAttempts   bool `json:"log_retry_attempts"`
	LogSuccessfulRetry bool `json:"log_successful_retry"`

	// Advanced settings
	EnableCircuitBreaker    bool          `json:"enable_circuit_breaker"`
	CircuitBreakerThreshold int           `json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   time.Duration `json:"circuit_breaker_timeout"`
}

// BackoffStrategy defines the backoff strategy for retries
type BackoffStrategy string

const (
	BackoffStrategyFixed       BackoffStrategy = "fixed"
	BackoffStrategyLinear      BackoffStrategy = "linear"
	BackoffStrategyExponential BackoffStrategy = "exponential"
)

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:             3,
		InitialDelay:            100 * time.Millisecond,
		MaxDelay:                5 * time.Second,
		Multiplier:              2.0,
		Jitter:                  true,
		JitterFactor:            0.1,
		BackoffStrategy:         BackoffStrategyExponential,
		RetryOnTimeout:          true,
		RetryOnConnectionLoss:   true,
		RetryOnDeadlock:         true,
		RetryOnQueryCancel:      false,
		LogRetries:              true,
		LogRetryAttempts:        true,
		LogSuccessfulRetry:      true,
		EnableCircuitBreaker:    false,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   30 * time.Second,
	}
}

// RetryableError determines if an error is retryable
type RetryableError interface {
	IsRetryable() bool
	ShouldRetry(attempt int) bool
	GetRetryDelay() time.Duration
}

// RetryOptions defines options for a specific retry operation
type RetryOptions struct {
	// Override default config
	MaxAttempts  *int
	InitialDelay *time.Duration
	MaxDelay     *time.Duration
	Multiplier   *float64
	Jitter       *bool

	// Operation-specific settings
	Context       context.Context
	OperationName string
	OnRetry       func(attempt int, err error)
	ShouldRetry   func(attempt int, err error) bool
}

// RetryResult contains information about the retry operation
type RetryResult struct {
	Success       bool           `json:"success"`
	Attempts      int            `json:"attempts"`
	TotalDuration time.Duration  `json:"total_duration"`
	FinalError    error          `json:"final_error,omitempty"`
	RetryHistory  []RetryAttempt `json:"retry_history"`
}

// RetryAttempt contains information about a single retry attempt
type RetryAttempt struct {
	AttemptNumber int           `json:"attempt_number"`
	Delay         time.Duration `json:"delay"`
	Error         error         `json:"error,omitempty"`
	Duration      time.Duration `json:"duration"`
	Timestamp     time.Time     `json:"timestamp"`
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name        string
	config      *RetryConfig
	logger      *zerolog.Logger
	state       CircuitBreakerState
	failures    int
	lastFailure time.Time
	nextAttempt time.Time
}

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config *RetryConfig, logger *zerolog.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		name:   name,
		config: config,
		logger: logger,
		state:  CircuitBreakerClosed,
	}
}

// CanExecute checks if the operation can be executed
func (cb *CircuitBreaker) CanExecute() bool {
	now := time.Now()

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		if now.After(cb.nextAttempt) {
			cb.state = CircuitBreakerHalfOpen
			cb.logger.Info().Str("circuit_breaker", cb.name).Msg("Circuit breaker transitioning to half-open")
			return true
		}
		return false
	case CircuitBreakerHalfOpen:
		return true
	default:
		return false
	}
}

// OnSuccess records a successful operation
func (cb *CircuitBreaker) OnSuccess() {
	if cb.state == CircuitBreakerHalfOpen {
		cb.state = CircuitBreakerClosed
		cb.failures = 0
		cb.logger.Info().Str("circuit_breaker", cb.name).Msg("Circuit breaker closed after successful operation")
	}
}

// OnFailure records a failed operation
func (cb *CircuitBreaker) OnFailure() {
	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.config.CircuitBreakerThreshold {
		cb.state = CircuitBreakerOpen
		cb.nextAttempt = time.Now().Add(cb.config.CircuitBreakerTimeout)
		cb.logger.Warn().
			Str("circuit_breaker", cb.name).
			Int("failures", cb.failures).
			Time("next_attempt", cb.nextAttempt).
			Msg("Circuit breaker opened")
	}
}

// RetryManager manages database retry operations
type RetryManager struct {
	config         *RetryConfig
	logger         *zerolog.Logger
	circuitBreaker *CircuitBreaker
	randSource     *rand.Rand
}

// NewRetryManager creates a new retry manager
func NewRetryManager(config *RetryConfig, logger *zerolog.Logger) *RetryManager {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var circuitBreaker *CircuitBreaker
	if config.EnableCircuitBreaker {
		circuitBreaker = NewCircuitBreaker("database", config, logger)
	}

	return &RetryManager{
		config:         config,
		logger:         logger,
		circuitBreaker: circuitBreaker,
		randSource:     rand.New(rand.NewSource(time.Now().UnixNano())), // #nosec G404 - Used for timing jitter, not security
	}
}

// ExecuteWithRetry executes a database operation with retry logic
func (rm *RetryManager) ExecuteWithRetry(ctx context.Context, operation func() error, options *RetryOptions) *RetryResult {
	if options == nil {
		options = &RetryOptions{}
	}

	// Merge options with default config
	maxAttempts := rm.config.MaxAttempts
	if options.MaxAttempts != nil {
		maxAttempts = *options.MaxAttempts
	}

	initialDelay := rm.config.InitialDelay
	if options.InitialDelay != nil {
		initialDelay = *options.InitialDelay
	}

	maxDelay := rm.config.MaxDelay
	if options.MaxDelay != nil {
		maxDelay = *options.MaxDelay
	}

	multiplier := rm.config.Multiplier
	if options.Multiplier != nil {
		multiplier = *options.Multiplier
	}

	jitter := rm.config.Jitter
	if options.Jitter != nil {
		jitter = *options.Jitter
	}

	// Check circuit breaker
	if rm.circuitBreaker != nil && !rm.circuitBreaker.CanExecute() {
		return &RetryResult{
			Success:    false,
			Attempts:   0,
			FinalError: fmt.Errorf("circuit breaker is open"),
		}
	}

	startTime := time.Now()
	result := &RetryResult{
		RetryHistory: make([]RetryAttempt, 0),
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptStart := time.Now()

		// Execute the operation
		err := operation()
		attemptDuration := time.Since(attemptStart)

		if err == nil {
			// Success
			result.Success = true
			result.Attempts = attempt
			result.TotalDuration = time.Since(startTime)

			// Record successful attempt
			result.RetryHistory = append(result.RetryHistory, RetryAttempt{
				AttemptNumber: attempt,
				Duration:      attemptDuration,
				Timestamp:     attemptStart,
			})

			// Update circuit breaker
			if rm.circuitBreaker != nil {
				rm.circuitBreaker.OnSuccess()
			}

			// Log successful retry if this wasn't the first attempt
			if attempt > 1 && rm.config.LogSuccessfulRetry {
				rm.logger.Info().
					Str("operation", options.OperationName).
					Int("attempt", attempt).
					Dur("total_duration", result.TotalDuration).
					Msg("Operation succeeded after retries")
			}

			return result
		}

		// Record failed attempt
		lastErr = err
		result.RetryHistory = append(result.RetryHistory, RetryAttempt{
			AttemptNumber: attempt,
			Error:         err,
			Duration:      attemptDuration,
			Timestamp:     attemptStart,
		})

		// Check if we should retry
		if attempt == maxAttempts || !rm.shouldRetryError(err, attempt, options) {
			break
		}

		// Calculate delay for next attempt
		delay := rm.calculateDelay(attempt-1, initialDelay, maxDelay, multiplier, jitter)

		// Log retry attempt
		if rm.config.LogRetryAttempts {
			rm.logger.Warn().
				Str("operation", options.OperationName).
				Int("attempt", attempt).
				Int("max_attempts", maxAttempts).
				Dur("delay", delay).
				Err(err).
				Msg("Retrying database operation")
		}

		// Call on retry callback if provided
		if options.OnRetry != nil {
			options.OnRetry(attempt, err)
		}

		// Wait before retry
		if ctx != nil {
			select {
			case <-ctx.Done():
				result.FinalError = ctx.Err()
				result.TotalDuration = time.Since(startTime)
				result.Attempts = attempt
				return result
			case <-time.After(delay):
				// Continue with retry
			}
		} else {
			time.Sleep(delay)
		}
	}

	// All attempts failed
	result.Success = false
	result.Attempts = maxAttempts
	result.TotalDuration = time.Since(startTime)
	result.FinalError = lastErr

	// Update circuit breaker
	if rm.circuitBreaker != nil {
		rm.circuitBreaker.OnFailure()
	}

	// Log final failure
	if rm.config.LogRetries {
		rm.logger.Error().
			Str("operation", options.OperationName).
			Int("attempts", maxAttempts).
			Dur("total_duration", result.TotalDuration).
			Err(lastErr).
			Msg("Operation failed after all retry attempts")
	}

	return result
}

// shouldRetryError determines if an error should be retried
func (rm *RetryManager) shouldRetryError(err error, attempt int, options *RetryOptions) bool {
	// Check custom should retry function first
	if options.ShouldRetry != nil {
		return options.ShouldRetry(attempt, err)
	}

	// Check for specific error types
	switch e := err.(type) {
	case *pgconn.PgError:
		return rm.shouldRetryPgError(e)
	case interface{ Timeout() bool }:
		if e.Timeout() {
			return rm.config.RetryOnTimeout
		}
	case interface{ Canceled() bool }:
		if e.Canceled() {
			return rm.config.RetryOnQueryCancel
		}
	default:
		// For other errors, check if they contain retryable keywords
		errStr := err.Error()
		retryablePatterns := []string{
			"connection reset",
			"connection refused",
			"broken pipe",
			"connection timed out",
			"temporary failure",
			"try again",
			"deadlock",
		}

		for _, pattern := range retryablePatterns {
			if containsString(errStr, pattern) {
				return true
			}
		}
	}

	return false
}

// shouldRetryPgError determines if a PostgreSQL error should be retried
func (rm *RetryManager) shouldRetryPgError(err *pgconn.PgError) bool {
	// Check for specific PostgreSQL error codes that are retryable
	retryableCodes := map[string]bool{
		"40001": true, // serialization_failure (deadlock)
		"40P01": true, // deadlock_detected
		"53000": true, // insufficient_resources
		"53100": true, // disk_full
		"53200": true, // out_of_memory
		"53300": true, // too_many_connections
		"53400": true, // configuration_limit_exceeded
		"57P03": true, // cannot_connect_now
		"58P01": true, // system_error (some system errors are retryable)
		"58P02": true, // io_error
	}

	if retryableCodes[err.Code] {
		// Special handling for deadlocks
		if err.Code == "40001" || err.Code == "40P01" {
			return rm.config.RetryOnDeadlock
		}
		return true
	}

	// Check for connection-related errors
	if err.Severity == "FATAL" && containsString(err.Message, "connection") {
		return rm.config.RetryOnConnectionLoss
	}

	return false
}

// calculateDelay calculates the delay for the next retry attempt
func (rm *RetryManager) calculateDelay(attempt int, initialDelay, maxDelay time.Duration, multiplier float64, jitter bool) time.Duration {
	var delay time.Duration

	switch rm.config.BackoffStrategy {
	case BackoffStrategyFixed:
		delay = initialDelay
	case BackoffStrategyLinear:
		delay = time.Duration(float64(initialDelay) * float64(attempt))
	case BackoffStrategyExponential:
		delay = time.Duration(float64(initialDelay) * math.Pow(multiplier, float64(attempt)))
	default:
		delay = initialDelay
	}

	// Apply maximum delay limit
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter if enabled
	if jitter {
		jitterAmount := time.Duration(float64(delay) * rm.config.JitterFactor)
		if jitterAmount > 0 {
			// Use the random source to generate jitter
			jitterValue := rm.randSource.Float64()
			delay += time.Duration(float64(jitterAmount) * (2*jitterValue - 1))
		}
	}

	// Ensure delay is non-negative
	if delay < 0 {
		delay = 0
	}

	return delay
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetCircuitBreakerStatus returns the current circuit breaker status
func (rm *RetryManager) GetCircuitBreakerStatus() map[string]interface{} {
	if rm.circuitBreaker == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	state := "unknown"
	switch rm.circuitBreaker.state {
	case CircuitBreakerClosed:
		state = "closed"
	case CircuitBreakerOpen:
		state = "open"
	case CircuitBreakerHalfOpen:
		state = "half-open"
	}

	return map[string]interface{}{
		"enabled":      true,
		"state":        state,
		"failures":     rm.circuitBreaker.failures,
		"last_failure": rm.circuitBreaker.lastFailure,
		"next_attempt": rm.circuitBreaker.nextAttempt,
		"threshold":    rm.config.CircuitBreakerThreshold,
		"timeout":      rm.config.CircuitBreakerTimeout,
	}
}

// GetRetryStats returns retry statistics
func (rm *RetryManager) GetRetryStats() map[string]interface{} {
	return map[string]interface{}{
		"max_attempts":     rm.config.MaxAttempts,
		"initial_delay":    rm.config.InitialDelay.String(),
		"max_delay":        rm.config.MaxDelay.String(),
		"multiplier":       rm.config.Multiplier,
		"jitter":           rm.config.Jitter,
		"backoff_strategy": rm.config.BackoffStrategy,
		"circuit_breaker":  rm.GetCircuitBreakerStatus(),
	}
}

// WithRetryOptions creates retry options with the given parameters
func WithRetryOptions(operationName string, maxAttempts int, initialDelay time.Duration) *RetryOptions {
	return &RetryOptions{
		OperationName: operationName,
		MaxAttempts:   &maxAttempts,
		InitialDelay:  &initialDelay,
	}
}

// WithContext adds context to retry options
func (options *RetryOptions) WithContext(ctx context.Context) *RetryOptions {
	options.Context = ctx
	return options
}

// WithOnRetry adds a retry callback to retry options
func (options *RetryOptions) WithOnRetry(onRetry func(attempt int, err error)) *RetryOptions {
	options.OnRetry = onRetry
	return options
}

// WithShouldRetry adds a custom should retry function to retry options
func (options *RetryOptions) WithShouldRetry(shouldRetry func(attempt int, err error) bool) *RetryOptions {
	options.ShouldRetry = shouldRetry
	return options
}
