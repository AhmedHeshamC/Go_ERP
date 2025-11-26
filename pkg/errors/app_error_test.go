package errors

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestNewAppError(t *testing.T) {
	err := NewAppError(ErrCodeNotFound, "resource not found", http.StatusNotFound)
	
	if err.Code != ErrCodeNotFound {
		t.Errorf("Expected code %s, got %s", ErrCodeNotFound, err.Code)
	}
	
	if err.Message != "resource not found" {
		t.Errorf("Expected message 'resource not found', got '%s'", err.Message)
	}
	
	if err.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, err.StatusCode)
	}
}

func TestAppErrorWithContext(t *testing.T) {
	err := NewAppError(ErrCodeValidation, "validation failed", http.StatusBadRequest)
	err.WithContext("field", "email")
	err.WithContext("value", "invalid@")
	
	if err.Details["field"] != "email" {
		t.Errorf("Expected field context 'email', got '%v'", err.Details["field"])
	}
	
	if err.Details["value"] != "invalid@" {
		t.Errorf("Expected value context 'invalid@', got '%v'", err.Details["value"])
	}
}

func TestAppErrorWithCorrelationID(t *testing.T) {
	err := NewAppError(ErrCodeInternal, "internal error", http.StatusInternalServerError)
	correlationID := "test-correlation-123"
	err.WithCorrelationID(correlationID)
	
	if err.CorrelationID != correlationID {
		t.Errorf("Expected correlation ID '%s', got '%s'", correlationID, err.CorrelationID)
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := WrapError(originalErr, ErrCodeInternal, "wrapped error", http.StatusInternalServerError)
	
	if wrappedErr.Err != originalErr {
		t.Error("Expected wrapped error to contain original error")
	}
	
	if !errors.Is(wrappedErr, originalErr) {
		t.Error("Expected errors.Is to work with wrapped error")
	}
}

func TestNotFoundError(t *testing.T) {
	err := NewNotFoundError("user not found")
	
	if err.Code != ErrCodeNotFound {
		t.Errorf("Expected code %s, got %s", ErrCodeNotFound, err.Code)
	}
	
	if err.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, err.StatusCode)
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("validation failed")
	err.AddFieldError("email", "invalid format")
	err.AddFieldError("email", "required field")
	err.AddFieldError("password", "too short")
	
	if len(err.Fields["email"]) != 2 {
		t.Errorf("Expected 2 errors for email field, got %d", len(err.Fields["email"]))
	}
	
	if len(err.Fields["password"]) != 1 {
		t.Errorf("Expected 1 error for password field, got %d", len(err.Fields["password"]))
	}
}

func TestConflictError(t *testing.T) {
	err := NewConflictError("resource already exists")
	
	if err.Code != ErrCodeConflict {
		t.Errorf("Expected code %s, got %s", ErrCodeConflict, err.Code)
	}
	
	if err.StatusCode != http.StatusConflict {
		t.Errorf("Expected status code %d, got %d", http.StatusConflict, err.StatusCode)
	}
}

func TestUnauthorizedError(t *testing.T) {
	err := NewUnauthorizedError("invalid credentials")
	
	if err.Code != ErrCodeUnauthorized {
		t.Errorf("Expected code %s, got %s", ErrCodeUnauthorized, err.Code)
	}
	
	if err.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, err.StatusCode)
	}
}

func TestRateLimitError(t *testing.T) {
	retryAfter := 15 * time.Minute
	err := NewRateLimitError("rate limit exceeded", retryAfter)
	
	if err.Code != ErrCodeRateLimit {
		t.Errorf("Expected code %s, got %s", ErrCodeRateLimit, err.Code)
	}
	
	if err.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status code %d, got %d", http.StatusTooManyRequests, err.StatusCode)
	}
	
	if err.RetryAfter != retryAfter {
		t.Errorf("Expected retry after %v, got %v", retryAfter, err.RetryAfter)
	}
}

func TestGetStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"NotFoundError", NewNotFoundError("not found"), http.StatusNotFound},
		{"ValidationError", NewValidationError("validation failed"), http.StatusBadRequest},
		{"ConflictError", NewConflictError("conflict"), http.StatusConflict},
		{"UnauthorizedError", NewUnauthorizedError("unauthorized"), http.StatusUnauthorized},
		{"InternalError", NewInternalError("internal error"), http.StatusInternalServerError},
		{"GenericError", errors.New("generic error"), http.StatusInternalServerError},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := GetStatusCode(tt.err)
			if status != tt.wantStatus {
				t.Errorf("GetStatusCode() = %d, want %d", status, tt.wantStatus)
			}
		})
	}
}

func TestGetErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode ErrorCode
	}{
		{"NotFoundError", NewNotFoundError("not found"), ErrCodeNotFound},
		{"ValidationError", NewValidationError("validation failed"), ErrCodeValidation},
		{"ConflictError", NewConflictError("conflict"), ErrCodeConflict},
		{"UnauthorizedError", NewUnauthorizedError("unauthorized"), ErrCodeUnauthorized},
		{"InternalError", NewInternalError("internal error"), ErrCodeInternal},
		{"GenericError", errors.New("generic error"), ErrCodeInternal},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := GetErrorCode(tt.err)
			if code != tt.wantCode {
				t.Errorf("GetErrorCode() = %s, want %s", code, tt.wantCode)
			}
		})
	}
}

func TestAppErrorError(t *testing.T) {
	// Test error without wrapped error
	err1 := NewAppError(ErrCodeNotFound, "resource not found", http.StatusNotFound)
	expected1 := "NOT_FOUND: resource not found"
	if err1.Error() != expected1 {
		t.Errorf("Expected error string '%s', got '%s'", expected1, err1.Error())
	}
	
	// Test error with wrapped error
	originalErr := errors.New("database connection failed")
	err2 := WrapError(originalErr, ErrCodeInternal, "internal error", http.StatusInternalServerError)
	if !errors.Is(err2, originalErr) {
		t.Error("Expected wrapped error to be detectable with errors.Is")
	}
}
