package errors

// Domain-specific error constructors for common validation scenarios

// NewDomainValidationError creates a validation error for domain entities
func NewDomainValidationError(entityType string, fieldErrors map[string][]string) *ValidationError {
	err := NewValidationError(entityType + " validation failed")
	err.Fields = fieldErrors
	return err
}

// NewFieldValidationError creates a validation error for a single field
func NewFieldValidationError(entityType, field, message string) *ValidationError {
	err := NewValidationError(entityType + " validation failed")
	err.AddFieldError(field, message)
	return err
}

// NewEntityNotFoundError creates a not found error for domain entities
func NewEntityNotFoundError(entityType string, identifier interface{}) *NotFoundError {
	return NewNotFoundError(entityType + " not found")
}

// NewInsufficientStockError creates a conflict error for insufficient stock
func NewInsufficientStockError(available, requested int) *ConflictError {
	err := NewConflictError("insufficient stock")
	err.WithContext("available", available)
	err.WithContext("requested", requested)
	return err
}

// NewInvalidTransitionError creates a bad request error for invalid state transitions
func NewInvalidTransitionError(from, to string) *BadRequestError {
	err := NewBadRequestError("invalid state transition")
	err.WithContext("from", from)
	err.WithContext("to", to)
	return err
}
