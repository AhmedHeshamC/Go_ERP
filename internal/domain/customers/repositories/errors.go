package repositories

import "errors"

// Common errors for customer repository
var (
	ErrCustomerNotFound           = errors.New("customer not found")
	ErrCustomerAlreadyExists      = errors.New("customer already exists")
	ErrCustomerEmailAlreadyExists = errors.New("customer with this email already exists")
)
