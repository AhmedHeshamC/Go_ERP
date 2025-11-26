package errors

import (
	"fmt"
	"net/http"
	"time"
)

// ErrorCode represents a unique error code for categorizing errors
type ErrorCode string

const (
	ErrCodeNotFound      ErrorCode = "NOT_FOUND"
	ErrCodeValidation    ErrorCode = "VALIDATION_ERROR"
	ErrCodeConflict      ErrorCode = "CONFLICT"
	ErrCodeUnauthorized  ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden     ErrorCode = "FORBIDDEN"
	ErrCodeInternal      ErrorCode = "INTERNAL_ERROR"
	ErrCodeRateLimit     ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrCodeBadRequest    ErrorCode = "BAD_REQUEST"
	ErrCodeTimeout       ErrorCode = "TIMEOUT"
	ErrCodeUnavailable   ErrorCode = "SERVICE_UNAVAILABLE"
)

// AppError represents an application error with rich context
type AppError struct {
	Code          ErrorCode              `json:"code"`
	Message       string                 `json:"message"`
	Details       map[string]interface{} `json:"details,omitempty"`
	Err           error                  `json:"-"`
	StatusCode    int                    `json:"-"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error for errors.Is and errors.As
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithContext adds context information to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithCorrelationID adds a correlation ID to the error
func (e *AppError) WithCorrelationID(correlationID string) *AppError {
	e.CorrelationID = correlationID
	return e
}

// NewAppError creates a new AppError
func NewAppError(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Timestamp:  time.Now(),
	}
}

// WrapError wraps an existing error with AppError context
func WrapError(err error, code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Err:        err,
		StatusCode: statusCode,
		Timestamp:  time.Now(),
	}
}

// NotFoundError creates a new not found error
type NotFoundError struct {
	*AppError
}

// NewNotFoundError creates a new NotFoundError
func NewNotFoundError(message string) *NotFoundError {
	return &NotFoundError{
		AppError: NewAppError(ErrCodeNotFound, message, http.StatusNotFound),
	}
}

// WrapNotFoundError wraps an error as NotFoundError
func WrapNotFoundError(err error, message string) *NotFoundError {
	return &NotFoundError{
		AppError: WrapError(err, ErrCodeNotFound, message, http.StatusNotFound),
	}
}

// ValidationError creates a validation error with field-level details
type ValidationError struct {
	*AppError
	Fields map[string][]string `json:"fields,omitempty"`
}

// NewValidationError creates a new ValidationError
func NewValidationError(message string) *ValidationError {
	return &ValidationError{
		AppError: NewAppError(ErrCodeValidation, message, http.StatusBadRequest),
		Fields:   make(map[string][]string),
	}
}

// WrapValidationError wraps an error as ValidationError
func WrapValidationError(err error, message string) *ValidationError {
	return &ValidationError{
		AppError: WrapError(err, ErrCodeValidation, message, http.StatusBadRequest),
		Fields:   make(map[string][]string),
	}
}

// AddFieldError adds a field-specific error message
func (e *ValidationError) AddFieldError(field string, message string) *ValidationError {
	if e.Fields == nil {
		e.Fields = make(map[string][]string)
	}
	e.Fields[field] = append(e.Fields[field], message)
	return e
}

// ConflictError represents a conflict error (e.g., duplicate resource)
type ConflictError struct {
	*AppError
}

// NewConflictError creates a new ConflictError
func NewConflictError(message string) *ConflictError {
	return &ConflictError{
		AppError: NewAppError(ErrCodeConflict, message, http.StatusConflict),
	}
}

// WrapConflictError wraps an error as ConflictError
func WrapConflictError(err error, message string) *ConflictError {
	return &ConflictError{
		AppError: WrapError(err, ErrCodeConflict, message, http.StatusConflict),
	}
}

// UnauthorizedError represents an authentication error
type UnauthorizedError struct {
	*AppError
}

// NewUnauthorizedError creates a new UnauthorizedError
func NewUnauthorizedError(message string) *UnauthorizedError {
	return &UnauthorizedError{
		AppError: NewAppError(ErrCodeUnauthorized, message, http.StatusUnauthorized),
	}
}

// WrapUnauthorizedError wraps an error as UnauthorizedError
func WrapUnauthorizedError(err error, message string) *UnauthorizedError {
	return &UnauthorizedError{
		AppError: WrapError(err, ErrCodeUnauthorized, message, http.StatusUnauthorized),
	}
}

// ForbiddenError represents an authorization error
type ForbiddenError struct {
	*AppError
}

// NewForbiddenError creates a new ForbiddenError
func NewForbiddenError(message string) *ForbiddenError {
	return &ForbiddenError{
		AppError: NewAppError(ErrCodeForbidden, message, http.StatusForbidden),
	}
}

// WrapForbiddenError wraps an error as ForbiddenError
func WrapForbiddenError(err error, message string) *ForbiddenError {
	return &ForbiddenError{
		AppError: WrapError(err, ErrCodeForbidden, message, http.StatusForbidden),
	}
}

// RateLimitError represents a rate limit exceeded error
type RateLimitError struct {
	*AppError
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

// NewRateLimitError creates a new RateLimitError
func NewRateLimitError(message string, retryAfter time.Duration) *RateLimitError {
	return &RateLimitError{
		AppError:   NewAppError(ErrCodeRateLimit, message, http.StatusTooManyRequests),
		RetryAfter: retryAfter,
	}
}

// WrapRateLimitError wraps an error as RateLimitError
func WrapRateLimitError(err error, message string, retryAfter time.Duration) *RateLimitError {
	return &RateLimitError{
		AppError:   WrapError(err, ErrCodeRateLimit, message, http.StatusTooManyRequests),
		RetryAfter: retryAfter,
	}
}

// InternalError represents an internal server error
type InternalError struct {
	*AppError
}

// NewInternalError creates a new InternalError
func NewInternalError(message string) *InternalError {
	return &InternalError{
		AppError: NewAppError(ErrCodeInternal, message, http.StatusInternalServerError),
	}
}

// WrapInternalError wraps an error as InternalError
func WrapInternalError(err error, message string) *InternalError {
	return &InternalError{
		AppError: WrapError(err, ErrCodeInternal, message, http.StatusInternalServerError),
	}
}

// BadRequestError represents a bad request error
type BadRequestError struct {
	*AppError
}

// NewBadRequestError creates a new BadRequestError
func NewBadRequestError(message string) *BadRequestError {
	return &BadRequestError{
		AppError: NewAppError(ErrCodeBadRequest, message, http.StatusBadRequest),
	}
}

// WrapBadRequestError wraps an error as BadRequestError
func WrapBadRequestError(err error, message string) *BadRequestError {
	return &BadRequestError{
		AppError: WrapError(err, ErrCodeBadRequest, message, http.StatusBadRequest),
	}
}

// TimeoutError represents a timeout error
type TimeoutError struct {
	*AppError
}

// NewTimeoutError creates a new TimeoutError
func NewTimeoutError(message string) *TimeoutError {
	return &TimeoutError{
		AppError: NewAppError(ErrCodeTimeout, message, http.StatusRequestTimeout),
	}
}

// WrapTimeoutError wraps an error as TimeoutError
func WrapTimeoutError(err error, message string) *TimeoutError {
	return &TimeoutError{
		AppError: WrapError(err, ErrCodeTimeout, message, http.StatusRequestTimeout),
	}
}

// UnavailableError represents a service unavailable error
type UnavailableError struct {
	*AppError
}

// NewUnavailableError creates a new UnavailableError
func NewUnavailableError(message string) *UnavailableError {
	return &UnavailableError{
		AppError: NewAppError(ErrCodeUnavailable, message, http.StatusServiceUnavailable),
	}
}

// WrapUnavailableError wraps an error as UnavailableError
func WrapUnavailableError(err error, message string) *UnavailableError {
	return &UnavailableError{
		AppError: WrapError(err, ErrCodeUnavailable, message, http.StatusServiceUnavailable),
	}
}

// GetStatusCode extracts the HTTP status code from an error
func GetStatusCode(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.StatusCode
	}
	
	// Check for specific error types
	switch err.(type) {
	case *NotFoundError:
		return http.StatusNotFound
	case *ValidationError:
		return http.StatusBadRequest
	case *ConflictError:
		return http.StatusConflict
	case *UnauthorizedError:
		return http.StatusUnauthorized
	case *ForbiddenError:
		return http.StatusForbidden
	case *RateLimitError:
		return http.StatusTooManyRequests
	case *BadRequestError:
		return http.StatusBadRequest
	case *TimeoutError:
		return http.StatusRequestTimeout
	case *UnavailableError:
		return http.StatusServiceUnavailable
	case *InternalError:
		return http.StatusInternalServerError
	}
	
	return http.StatusInternalServerError
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) ErrorCode {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	
	// Check for specific error types
	switch err.(type) {
	case *NotFoundError:
		return ErrCodeNotFound
	case *ValidationError:
		return ErrCodeValidation
	case *ConflictError:
		return ErrCodeConflict
	case *UnauthorizedError:
		return ErrCodeUnauthorized
	case *ForbiddenError:
		return ErrCodeForbidden
	case *RateLimitError:
		return ErrCodeRateLimit
	case *BadRequestError:
		return ErrCodeBadRequest
	case *TimeoutError:
		return ErrCodeTimeout
	case *UnavailableError:
		return ErrCodeUnavailable
	case *InternalError:
		return ErrCodeInternal
	}
	
	return ErrCodeInternal
}
