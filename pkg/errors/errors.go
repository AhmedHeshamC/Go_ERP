package errors

import (
	"errors"
	"net/http"
)

// Common application errors
var (
	ErrNotFound          = errors.New("resource not found")
	ErrInvalidInput      = errors.New("invalid input")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrConflict          = errors.New("conflict")
	ErrInternalServer    = errors.New("internal server error")
	ErrInsufficientStock = errors.New("insufficient stock")
)

// Error represents an application error with HTTP status code
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	return e.Message
}

// NewError creates a new application error
func NewError(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithDetails creates a new application error with details
func NewErrorWithDetails(code int, message, details string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// HTTPStatus returns the appropriate HTTP status code for common errors
func HTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// Error type checking helper functions
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsConflictError(err error) bool {
	return errors.Is(err, ErrConflict)
}

func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

func IsUnauthorizedError(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

func IsForbiddenError(err error) bool {
	return errors.Is(err, ErrForbidden)
}

func IsInternalServerError(err error) bool {
	return errors.Is(err, ErrInternalServer)
}

func IsInsufficientStockError(err error) bool {
	return errors.Is(err, ErrInsufficientStock)
}
