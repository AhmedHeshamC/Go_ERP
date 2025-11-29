package errors

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ClassifyDatabaseError classifies a database error into an appropriate AppError
func ClassifyDatabaseError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Check for no rows error
	if errors.Is(err, pgx.ErrNoRows) {
		return NewNotFoundError("resource not found")
	}

	// Check for PostgreSQL errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return classifyPostgresError(pgErr, operation)
	}

	// Check for connection errors
	if isConnectionError(err) {
		return WrapUnavailableError(err, "database connection error")
	}

	// Default to internal error
	return WrapInternalError(err, "database operation failed")
}

// classifyPostgresError classifies PostgreSQL-specific errors
func classifyPostgresError(pgErr *pgconn.PgError, operation string) error {
	switch pgErr.Code {
	// Unique violation
	case "23505":
		return WrapConflictError(pgErr, "resource already exists")

	// Foreign key violation
	case "23503":
		return WrapBadRequestError(pgErr, "referenced resource does not exist")

	// Check constraint violation
	case "23514":
		return WrapValidationError(pgErr, "data validation failed")

	// Not null violation
	case "23502":
		return WrapValidationError(pgErr, "required field is missing")

	// Deadlock detected
	case "40P01":
		return WrapConflictError(pgErr, "operation conflicted with another transaction")

	// Serialization failure
	case "40001":
		return WrapConflictError(pgErr, "transaction serialization failed")

	// Connection errors
	case "08000", "08003", "08006":
		return WrapUnavailableError(pgErr, "database connection error")

	// Insufficient resources
	case "53000", "53100", "53200", "53300", "53400":
		return WrapUnavailableError(pgErr, "database resources exhausted")

	default:
		return WrapInternalError(pgErr, "database error: "+pgErr.Message)
	}
}

// isConnectionError checks if the error is a connection-related error
func isConnectionError(err error) bool {
	errMsg := err.Error()
	connectionKeywords := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"no such host",
		"network is unreachable",
		"broken pipe",
	}

	for _, keyword := range connectionKeywords {
		if strings.Contains(strings.ToLower(errMsg), keyword) {
			return true
		}
	}

	return false
}

// IsRetryableError checks if a database error is retryable
func IsRetryableError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Deadlock and serialization failures are retryable
		return pgErr.Code == "40P01" || pgErr.Code == "40001"
	}

	// Connection errors might be retryable
	return isConnectionError(err)
}
