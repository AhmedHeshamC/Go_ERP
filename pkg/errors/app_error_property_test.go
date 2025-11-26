package errors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: production-readiness, Property 5: Domain Error Type Consistency**
// For any error originating in the domain layer, the error must be wrapped in a
// domain-specific error type (AppError) with appropriate error code
// **Validates: Requirements 3.1**
func TestProperty_DomainErrorTypeConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: All domain errors must be wrapped in AppError types
	properties.Property("domain errors are wrapped in AppError", prop.ForAll(
		func(errorType string, message string) bool {
			var err error

			// Create different types of domain errors
			switch errorType {
			case "not_found":
				err = NewNotFoundError(message)
			case "validation":
				err = NewValidationError(message)
			case "conflict":
				err = NewConflictError(message)
			case "unauthorized":
				err = NewUnauthorizedError(message)
			case "forbidden":
				err = NewForbiddenError(message)
			case "rate_limit":
				err = NewRateLimitError(message, 0)
			case "internal":
				err = NewInternalError(message)
			case "bad_request":
				err = NewBadRequestError(message)
			case "timeout":
				err = NewTimeoutError(message)
			case "unavailable":
				err = NewUnavailableError(message)
			default:
				return true // Skip unknown types
			}

			// Verify the error has an AppError embedded
			// Extract the AppError from the specific error type
			var appErr *AppError
			switch e := err.(type) {
			case *NotFoundError:
				appErr = e.AppError
			case *ValidationError:
				appErr = e.AppError
			case *ConflictError:
				appErr = e.AppError
			case *UnauthorizedError:
				appErr = e.AppError
			case *ForbiddenError:
				appErr = e.AppError
			case *RateLimitError:
				appErr = e.AppError
			case *InternalError:
				appErr = e.AppError
			case *BadRequestError:
				appErr = e.AppError
			case *TimeoutError:
				appErr = e.AppError
			case *UnavailableError:
				appErr = e.AppError
			default:
				return false
			}

			if appErr == nil {
				return false
			}

			// Verify the error has the correct error code
			expectedCode := getExpectedErrorCode(errorType)
			if appErr.Code != expectedCode {
				return false
			}

			// Verify the error has the correct status code
			expectedStatus := getExpectedStatusCode(errorType)
			if appErr.StatusCode != expectedStatus {
				return false
			}

			// Verify the error message is set
			if appErr.Message == "" {
				return false
			}

			return true
		},
		genDomainErrorType(),
		genErrorMessage(),
	))

	// Property: Wrapped errors preserve the original error
	properties.Property("wrapped errors preserve original error", prop.ForAll(
		func(errorType string, originalMsg string) bool {
			originalErr := errors.New(originalMsg)
			var wrappedErr error

			// Wrap the error in different types
			switch errorType {
			case "not_found":
				wrappedErr = WrapNotFoundError(originalErr, "wrapped message")
			case "validation":
				wrappedErr = WrapValidationError(originalErr, "wrapped message")
			case "conflict":
				wrappedErr = WrapConflictError(originalErr, "wrapped message")
			case "unauthorized":
				wrappedErr = WrapUnauthorizedError(originalErr, "wrapped message")
			case "forbidden":
				wrappedErr = WrapForbiddenError(originalErr, "wrapped message")
			case "internal":
				wrappedErr = WrapInternalError(originalErr, "wrapped message")
			case "bad_request":
				wrappedErr = WrapBadRequestError(originalErr, "wrapped message")
			case "timeout":
				wrappedErr = WrapTimeoutError(originalErr, "wrapped message")
			case "unavailable":
				wrappedErr = WrapUnavailableError(originalErr, "wrapped message")
			default:
				return true // Skip unknown types
			}

			// Verify the original error is preserved
			if !errors.Is(wrappedErr, originalErr) {
				return false
			}

			// Verify we can unwrap to get the original error
			var appErr *AppError
			if errors.As(wrappedErr, &appErr) {
				if appErr.Err == nil {
					return false
				}
				if appErr.Err.Error() != originalMsg {
					return false
				}
			}

			return true
		},
		genDomainErrorType(),
		genErrorMessage(),
	))

	// Property: Error codes must match error types
	properties.Property("error codes match error types", prop.ForAll(
		func(errorType string) bool {
			var err error

			switch errorType {
			case "not_found":
				err = NewNotFoundError("test")
			case "validation":
				err = NewValidationError("test")
			case "conflict":
				err = NewConflictError("test")
			case "unauthorized":
				err = NewUnauthorizedError("test")
			case "forbidden":
				err = NewForbiddenError("test")
			case "rate_limit":
				err = NewRateLimitError("test", 0)
			case "internal":
				err = NewInternalError("test")
			case "bad_request":
				err = NewBadRequestError("test")
			case "timeout":
				err = NewTimeoutError("test")
			case "unavailable":
				err = NewUnavailableError("test")
			default:
				return true
			}

			// Get the error code using the helper function
			code := GetErrorCode(err)
			expectedCode := getExpectedErrorCode(errorType)

			return code == expectedCode
		},
		genDomainErrorType(),
	))

	// Property: Status codes must match error types
	properties.Property("status codes match error types", prop.ForAll(
		func(errorType string) bool {
			var err error

			switch errorType {
			case "not_found":
				err = NewNotFoundError("test")
			case "validation":
				err = NewValidationError("test")
			case "conflict":
				err = NewConflictError("test")
			case "unauthorized":
				err = NewUnauthorizedError("test")
			case "forbidden":
				err = NewForbiddenError("test")
			case "rate_limit":
				err = NewRateLimitError("test", 0)
			case "internal":
				err = NewInternalError("test")
			case "bad_request":
				err = NewBadRequestError("test")
			case "timeout":
				err = NewTimeoutError("test")
			case "unavailable":
				err = NewUnavailableError("test")
			default:
				return true
			}

			// Get the status code using the helper function
			statusCode := GetStatusCode(err)
			expectedStatus := getExpectedStatusCode(errorType)

			return statusCode == expectedStatus
		},
		genDomainErrorType(),
	))

	// Property: Context can be added to errors without losing type information
	properties.Property("context preserves error type", prop.ForAll(
		func(errorType string, key string, value string) bool {
			var err error

			switch errorType {
			case "not_found":
				err = NewNotFoundError("test")
			case "validation":
				err = NewValidationError("test")
			case "conflict":
				err = NewConflictError("test")
			case "unauthorized":
				err = NewUnauthorizedError("test")
			case "forbidden":
				err = NewForbiddenError("test")
			case "internal":
				err = NewInternalError("test")
			case "bad_request":
				err = NewBadRequestError("test")
			case "timeout":
				err = NewTimeoutError("test")
			case "unavailable":
				err = NewUnavailableError("test")
			default:
				return true
			}

			// Add context
			var appErr *AppError
			if errors.As(err, &appErr) {
				appErr.WithContext(key, value)

				// Verify context was added
				if appErr.Details == nil {
					return false
				}
				if appErr.Details[key] != value {
					return false
				}

				// Verify error code is still correct
				expectedCode := getExpectedErrorCode(errorType)
				if appErr.Code != expectedCode {
					return false
				}
			}

			return true
		},
		genDomainErrorType(),
		gen.AlphaString(),
		gen.AlphaString(),
	))

	// Property: ValidationError must support field-level errors
	properties.Property("validation errors support field errors", prop.ForAll(
		func(field string, message string) bool {
			err := NewValidationError("validation failed")
			err.AddFieldError(field, message)

			// Verify field error was added
			if err.Fields == nil {
				return false
			}
			fieldErrors, exists := err.Fields[field]
			if !exists {
				return false
			}
			if len(fieldErrors) == 0 {
				return false
			}
			if fieldErrors[0] != message {
				return false
			}

			return true
		},
		gen.AlphaString(),
		genErrorMessage(),
	))

	// Property: Entity-specific errors maintain consistency
	properties.Property("entity errors are consistent", prop.ForAll(
		func(entityType string, identifier string) bool {
			err := NewEntityNotFoundError(entityType, identifier)

			// Verify it's a NotFoundError
			var notFoundErr *NotFoundError
			if !errors.As(err, &notFoundErr) {
				return false
			}

			// Verify error code
			if notFoundErr.Code != ErrCodeNotFound {
				return false
			}

			// Verify status code
			if notFoundErr.StatusCode != http.StatusNotFound {
				return false
			}

			return true
		},
		gen.AlphaString(),
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: production-readiness, Property 6: Database Error Classification**
// For any database operation failure, the system must classify the error as one of:
// NotFound, ConstraintViolation, ConnectionError, or InternalError
// **Validates: Requirements 3.4**
func TestProperty_DatabaseErrorClassification(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: pgx.ErrNoRows must be classified as NotFoundError
	properties.Property("no rows error classified as NotFoundError", prop.ForAll(
		func(operation string) bool {
			err := ClassifyDatabaseError(pgx.ErrNoRows, operation)

			// Verify it's a NotFoundError
			var notFoundErr *NotFoundError
			if !errors.As(err, &notFoundErr) {
				return false
			}

			// Verify error code
			if notFoundErr.Code != ErrCodeNotFound {
				return false
			}

			return true
		},
		gen.AlphaString(),
	))

	// Property: Unique constraint violations must be classified as ConflictError
	properties.Property("unique violations classified as ConflictError", prop.ForAll(
		func(operation string) bool {
			// Create a unique constraint violation error
			pgErr := &pgconn.PgError{
				Code:    "23505", // unique_violation
				Message: "duplicate key value violates unique constraint",
			}

			err := ClassifyDatabaseError(pgErr, operation)

			// Verify it's a ConflictError
			var conflictErr *ConflictError
			if !errors.As(err, &conflictErr) {
				return false
			}

			// Verify error code
			if conflictErr.Code != ErrCodeConflict {
				return false
			}

			return true
		},
		gen.AlphaString(),
	))

	// Property: Foreign key violations must be classified as BadRequestError
	properties.Property("foreign key violations classified as BadRequestError", prop.ForAll(
		func(operation string) bool {
			// Create a foreign key violation error
			pgErr := &pgconn.PgError{
				Code:    "23503", // foreign_key_violation
				Message: "insert or update on table violates foreign key constraint",
			}

			err := ClassifyDatabaseError(pgErr, operation)

			// Verify it's a BadRequestError
			var badReqErr *BadRequestError
			if !errors.As(err, &badReqErr) {
				return false
			}

			// Verify error code
			if badReqErr.Code != ErrCodeBadRequest {
				return false
			}

			return true
		},
		gen.AlphaString(),
	))

	// Property: Check constraint violations must be classified as ValidationError
	properties.Property("check violations classified as ValidationError", prop.ForAll(
		func(operation string) bool {
			// Create a check constraint violation error
			pgErr := &pgconn.PgError{
				Code:    "23514", // check_violation
				Message: "new row violates check constraint",
			}

			err := ClassifyDatabaseError(pgErr, operation)

			// Verify it's a ValidationError
			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				return false
			}

			// Verify error code
			if validationErr.Code != ErrCodeValidation {
				return false
			}

			return true
		},
		gen.AlphaString(),
	))

	// Property: Not null violations must be classified as ValidationError
	properties.Property("not null violations classified as ValidationError", prop.ForAll(
		func(operation string) bool {
			// Create a not null violation error
			pgErr := &pgconn.PgError{
				Code:    "23502", // not_null_violation
				Message: "null value in column violates not-null constraint",
			}

			err := ClassifyDatabaseError(pgErr, operation)

			// Verify it's a ValidationError
			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				return false
			}

			// Verify error code
			if validationErr.Code != ErrCodeValidation {
				return false
			}

			return true
		},
		gen.AlphaString(),
	))

	// Property: Deadlock errors must be classified as ConflictError
	properties.Property("deadlock errors classified as ConflictError", prop.ForAll(
		func(operation string) bool {
			// Create a deadlock error
			pgErr := &pgconn.PgError{
				Code:    "40P01", // deadlock_detected
				Message: "deadlock detected",
			}

			err := ClassifyDatabaseError(pgErr, operation)

			// Verify it's a ConflictError
			var conflictErr *ConflictError
			if !errors.As(err, &conflictErr) {
				return false
			}

			// Verify error code
			if conflictErr.Code != ErrCodeConflict {
				return false
			}

			return true
		},
		gen.AlphaString(),
	))

	// Property: Serialization failures must be classified as ConflictError
	properties.Property("serialization failures classified as ConflictError", prop.ForAll(
		func(operation string) bool {
			// Create a serialization failure error
			pgErr := &pgconn.PgError{
				Code:    "40001", // serialization_failure
				Message: "could not serialize access",
			}

			err := ClassifyDatabaseError(pgErr, operation)

			// Verify it's a ConflictError
			var conflictErr *ConflictError
			if !errors.As(err, &conflictErr) {
				return false
			}

			// Verify error code
			if conflictErr.Code != ErrCodeConflict {
				return false
			}

			return true
		},
		gen.AlphaString(),
	))

	// Property: Connection errors must be classified as UnavailableError
	properties.Property("connection errors classified as UnavailableError", prop.ForAll(
		func(operation string, errorCode string) bool {
			// Create a connection error
			pgErr := &pgconn.PgError{
				Code:    errorCode,
				Message: "connection error",
			}

			err := ClassifyDatabaseError(pgErr, operation)

			// Verify it's an UnavailableError
			var unavailableErr *UnavailableError
			if !errors.As(err, &unavailableErr) {
				return false
			}

			// Verify error code
			if unavailableErr.Code != ErrCodeUnavailable {
				return false
			}

			return true
		},
		gen.AlphaString(),
		genConnectionErrorCode(),
	))

	// Property: Resource exhaustion errors must be classified as UnavailableError
	properties.Property("resource errors classified as UnavailableError", prop.ForAll(
		func(operation string, errorCode string) bool {
			// Create a resource exhaustion error
			pgErr := &pgconn.PgError{
				Code:    errorCode,
				Message: "insufficient resources",
			}

			err := ClassifyDatabaseError(pgErr, operation)

			// Verify it's an UnavailableError
			var unavailableErr *UnavailableError
			if !errors.As(err, &unavailableErr) {
				return false
			}

			// Verify error code
			if unavailableErr.Code != ErrCodeUnavailable {
				return false
			}

			return true
		},
		gen.AlphaString(),
		genResourceErrorCode(),
	))

	// Property: Unknown database errors must be classified as InternalError
	properties.Property("unknown errors classified as InternalError", prop.ForAll(
		func(operation string, errorCode string) bool {
			// Skip known error codes
			knownCodes := map[string]bool{
				"23505": true, "23503": true, "23514": true, "23502": true,
				"40P01": true, "40001": true,
				"08000": true, "08003": true, "08006": true,
				"53000": true, "53100": true, "53200": true, "53300": true, "53400": true,
			}
			if knownCodes[errorCode] {
				return true
			}

			// Create an unknown error
			pgErr := &pgconn.PgError{
				Code:    errorCode,
				Message: "unknown database error",
			}

			err := ClassifyDatabaseError(pgErr, operation)

			// Verify it's an InternalError
			var internalErr *InternalError
			if !errors.As(err, &internalErr) {
				return false
			}

			// Verify error code
			if internalErr.Code != ErrCodeInternal {
				return false
			}

			return true
		},
		gen.AlphaString(),
		genUnknownErrorCode(),
	))

	// Property: Retryable errors must be identified correctly
	properties.Property("retryable errors identified correctly", prop.ForAll(
		func(errorCode string) bool {
			pgErr := &pgconn.PgError{
				Code:    errorCode,
				Message: "test error",
			}

			isRetryable := IsRetryableError(pgErr)

			// Deadlock and serialization failures should be retryable
			if errorCode == "40P01" || errorCode == "40001" {
				return isRetryable
			}

			// Other errors should not be retryable (in this simple test)
			return !isRetryable
		},
		genDatabaseErrorCode(),
	))

	// Property: Nil errors must return nil
	properties.Property("nil errors return nil", prop.ForAll(
		func(operation string) bool {
			err := ClassifyDatabaseError(nil, operation)
			return err == nil
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper functions

func getExpectedErrorCode(errorType string) ErrorCode {
	switch errorType {
	case "not_found":
		return ErrCodeNotFound
	case "validation":
		return ErrCodeValidation
	case "conflict":
		return ErrCodeConflict
	case "unauthorized":
		return ErrCodeUnauthorized
	case "forbidden":
		return ErrCodeForbidden
	case "rate_limit":
		return ErrCodeRateLimit
	case "internal":
		return ErrCodeInternal
	case "bad_request":
		return ErrCodeBadRequest
	case "timeout":
		return ErrCodeTimeout
	case "unavailable":
		return ErrCodeUnavailable
	default:
		return ErrCodeInternal
	}
}

func getExpectedStatusCode(errorType string) int {
	switch errorType {
	case "not_found":
		return http.StatusNotFound
	case "validation":
		return http.StatusBadRequest
	case "conflict":
		return http.StatusConflict
	case "unauthorized":
		return http.StatusUnauthorized
	case "forbidden":
		return http.StatusForbidden
	case "rate_limit":
		return http.StatusTooManyRequests
	case "internal":
		return http.StatusInternalServerError
	case "bad_request":
		return http.StatusBadRequest
	case "timeout":
		return http.StatusRequestTimeout
	case "unavailable":
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Generators

func genDomainErrorType() gopter.Gen {
	return gen.OneConstOf(
		"not_found",
		"validation",
		"conflict",
		"unauthorized",
		"forbidden",
		"rate_limit",
		"internal",
		"bad_request",
		"timeout",
		"unavailable",
	)
}

func genErrorMessage() gopter.Gen {
	return gen.OneConstOf(
		"resource not found",
		"validation failed",
		"conflict detected",
		"unauthorized access",
		"forbidden operation",
		"rate limit exceeded",
		"internal server error",
		"bad request",
		"operation timeout",
		"service unavailable",
		"user not found",
		"invalid email format",
		"duplicate entry",
		"insufficient permissions",
		"access denied",
		"too many requests",
		"database error",
		"invalid input",
		"request timeout",
		"service down",
	)
}

func genConnectionErrorCode() gopter.Gen {
	return gen.OneConstOf(
		"08000", // connection_exception
		"08003", // connection_does_not_exist
		"08006", // connection_failure
	)
}

func genResourceErrorCode() gopter.Gen {
	return gen.OneConstOf(
		"53000", // insufficient_resources
		"53100", // disk_full
		"53200", // out_of_memory
		"53300", // too_many_connections
		"53400", // configuration_limit_exceeded
	)
}

func genUnknownErrorCode() gopter.Gen {
	return gen.OneConstOf(
		"99999", // unknown
		"12345", // unknown
		"XXXXX", // unknown
		"00000", // unknown
	)
}

func genDatabaseErrorCode() gopter.Gen {
	return gen.OneGenOf(
		gen.OneConstOf("23505", "23503", "23514", "23502"), // Constraint violations
		gen.OneConstOf("40P01", "40001"),                    // Deadlock/serialization
		genConnectionErrorCode(),
		genResourceErrorCode(),
		genUnknownErrorCode(),
	)
}
