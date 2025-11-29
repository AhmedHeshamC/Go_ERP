package repositories

import (
	"github.com/google/uuid"
)

// OrderItemFilter defines filter criteria for order item queries

// OrderItemFilter defines filter criteria for order item queries
type OrderItemFilter struct {
	// Order filter
	OrderID *uuid.UUID

	// Product filter
	ProductID *uuid.UUID

	// Pagination
	Limit  *int
	Offset *int
}

// OrderAddressFilter defines filter criteria for order address queries
type OrderAddressFilter struct {
	// Order filter
	OrderID *uuid.UUID

	// Address type filter
	AddressType *string

	// Pagination
	Limit  *int
	Offset *int
}
