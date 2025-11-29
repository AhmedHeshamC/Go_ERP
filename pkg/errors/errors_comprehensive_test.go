package errors

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestError_Error(t *testing.T) {
	err := &Error{
		Code:    http.StatusNotFound,
		Message: "resource not found",
		Details: "user with ID 123 not found",
	}

	assert.Equal(t, "resource not found", err.Error())
}

func TestNewError(t *testing.T) {
	err := NewError(http.StatusBadRequest, "invalid input")

	assert.NotNil(t, err)
	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "invalid input", err.Message)
	assert.Empty(t, err.Details)
}

func TestNewErrorWithDetails(t *testing.T) {
	err := NewErrorWithDetails(http.StatusNotFound, "resource not found", "user with ID 123 not found")

	assert.NotNil(t, err)
	assert.Equal(t, http.StatusNotFound, err.Code)
	assert.Equal(t, "resource not found", err.Message)
	assert.Equal(t, "user with ID 123 not found", err.Details)
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "NotFound",
			err:      ErrNotFound,
			expected: http.StatusNotFound,
		},
		{
			name:     "InvalidInput",
			err:      ErrInvalidInput,
			expected: http.StatusBadRequest,
		},
		{
			name:     "Unauthorized",
			err:      ErrUnauthorized,
			expected: http.StatusUnauthorized,
		},
		{
			name:     "Forbidden",
			err:      ErrForbidden,
			expected: http.StatusForbidden,
		},
		{
			name:     "Conflict",
			err:      ErrConflict,
			expected: http.StatusConflict,
		},
		{
			name:     "InternalServer",
			err:      ErrInternalServer,
			expected: http.StatusInternalServerError,
		},
		{
			name:     "UnknownError",
			err:      errors.New("unknown error"),
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := HTTPStatus(tt.err)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	assert.True(t, IsNotFoundError(ErrNotFound))
	assert.False(t, IsNotFoundError(ErrInvalidInput))
	assert.False(t, IsNotFoundError(errors.New("other error")))
}

func TestIsConflictError(t *testing.T) {
	assert.True(t, IsConflictError(ErrConflict))
	assert.False(t, IsConflictError(ErrNotFound))
	assert.False(t, IsConflictError(errors.New("other error")))
}

func TestIsValidationError(t *testing.T) {
	assert.True(t, IsValidationError(ErrInvalidInput))
	assert.False(t, IsValidationError(ErrNotFound))
	assert.False(t, IsValidationError(errors.New("other error")))
}

func TestIsUnauthorizedError(t *testing.T) {
	assert.True(t, IsUnauthorizedError(ErrUnauthorized))
	assert.False(t, IsUnauthorizedError(ErrForbidden))
	assert.False(t, IsUnauthorizedError(errors.New("other error")))
}

func TestIsForbiddenError(t *testing.T) {
	assert.True(t, IsForbiddenError(ErrForbidden))
	assert.False(t, IsForbiddenError(ErrUnauthorized))
	assert.False(t, IsForbiddenError(errors.New("other error")))
}

func TestIsInternalServerError(t *testing.T) {
	assert.True(t, IsInternalServerError(ErrInternalServer))
	assert.False(t, IsInternalServerError(ErrNotFound))
	assert.False(t, IsInternalServerError(errors.New("other error")))
}

func TestIsInsufficientStockError(t *testing.T) {
	assert.True(t, IsInsufficientStockError(ErrInsufficientStock))
	assert.False(t, IsInsufficientStockError(ErrNotFound))
	assert.False(t, IsInsufficientStockError(errors.New("other error")))
}

// Test domain error constructors
func TestNewDomainValidationError(t *testing.T) {
	fieldErrors := map[string][]string{
		"name":  {"cannot be empty"},
		"price": {"must be positive"},
	}
	err := NewDomainValidationError("Product", fieldErrors)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Product")
	assert.Len(t, err.Fields, 2)
}

func TestNewFieldValidationError(t *testing.T) {
	err := NewFieldValidationError("User", "email", "invalid email format")

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "User")
	assert.Len(t, err.Fields, 1)
	assert.Contains(t, err.Fields["email"], "invalid email format")
}

func TestNewInsufficientStockError(t *testing.T) {
	err := NewInsufficientStockError(5, 10)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "insufficient stock")
	assert.Equal(t, 5, err.Details["available"])
	assert.Equal(t, 10, err.Details["requested"])
}

func TestNewInvalidTransitionError(t *testing.T) {
	err := NewInvalidTransitionError("PENDING", "CANCELLED")

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid state transition")
	assert.Equal(t, "PENDING", err.Details["from"])
	assert.Equal(t, "CANCELLED", err.Details["to"])
}

// Test AppError methods
func TestAppError_ErrorMethod(t *testing.T) {
	err := NewBadRequestError("invalid input")
	assert.Contains(t, err.Error(), "invalid input")

	err2 := NewNotFoundError("user not found")
	assert.Contains(t, err2.Error(), "user not found")
}

func TestValidationError_AddFieldError(t *testing.T) {
	err := NewValidationError("validation failed")
	err.AddFieldError("email", "invalid format")
	err.AddFieldError("password", "too short")

	assert.Len(t, err.Fields, 2)
	assert.Contains(t, err.Fields["email"], "invalid format")
	assert.Contains(t, err.Fields["password"], "too short")

	// Test nil Fields map
	err2 := &ValidationError{AppError: &AppError{Message: "test"}}
	err2.AddFieldError("field", "error")
	assert.Len(t, err2.Fields, 1)
}

func TestWrapRateLimitError(t *testing.T) {
	baseErr := errors.New("too many requests")
	err := WrapRateLimitError(baseErr, "rate limit exceeded", 60*time.Second)

	assert.NotNil(t, err)
	assert.Equal(t, 429, err.StatusCode)
	assert.Contains(t, err.Message, "rate limit")
	assert.Equal(t, 60*time.Second, err.RetryAfter)
}

// Test database error classification
func TestClassifyDatabaseError(t *testing.T) {
	t.Run("ConnectionError", func(t *testing.T) {
		err := errors.New("connection refused")
		result := ClassifyDatabaseError(err, "connect")
		assert.NotNil(t, result)
		// Should be wrapped as UnavailableError
		if unavailErr, ok := result.(*UnavailableError); ok {
			assert.Equal(t, 503, unavailErr.StatusCode)
		}
	})

	t.Run("UnknownError", func(t *testing.T) {
		err := errors.New("unknown database error")
		result := ClassifyDatabaseError(err, "query")
		assert.NotNil(t, result)
		// Should be wrapped as InternalError
		if internalErr, ok := result.(*InternalError); ok {
			assert.Equal(t, 500, internalErr.StatusCode)
		}
	})

	t.Run("NilError", func(t *testing.T) {
		result := ClassifyDatabaseError(nil, "query")
		assert.Nil(t, result)
	})
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ConnectionRefused",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "ConnectionReset",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "ConnectionTimeout",
			err:      errors.New("connection timeout"),
			expected: true,
		},
		{
			name:     "NoSuchHost",
			err:      errors.New("no such host"),
			expected: true,
		},
		{
			name:     "OtherError",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isConnectionError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ConnectionError",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "ConnectionReset",
			err:      errors.New("connection reset"),
			expected: true,
		},
		{
			name:     "UniqueViolation",
			err:      errors.New("duplicate key value violates unique constraint"),
			expected: false,
		},
		{
			name:     "OtherError",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test Reporter functionality - Additional coverage for edge cases
func TestReporter_DisabledReporting(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	config.Enabled = false
	config.AsyncReporting = false

	reporter, err := NewReporter(config, &logger)
	assert.NoError(t, err)

	ctx := context.Background()
	testErr := errors.New("test error")

	reporter.Report(ctx, testErr, SeverityError, ErrorTypeSystem, "test", nil, nil)

	// Should not be buffered when disabled
	reporter.bufferMu.RLock()
	assert.Equal(t, 0, len(reporter.buffer))
	reporter.bufferMu.RUnlock()
}
