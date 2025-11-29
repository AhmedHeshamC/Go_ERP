// OrderFilter defines filter criteria for order queries
type OrderFilter struct {
	// Customer filter
	CustomerID *uuid.UUID

	// Status filter
	Status *entities.OrderStatus

	// Date range filters
	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	// Pagination
	Limit  *int
	Offset *int

	// Sorting
	OrderBy *string
	Order   *string
}
```
<tool_call>edit_file
<arg_key>display_description</arg_key>
<arg_value>Create missing OrderItemFilter and OrderAddressFilter types</arg_value>
<arg_key>path</arg_key>
<arg_value>/Users/m/Desktop/Go_ERP/internal/domain/orders/repositories/order_filter.go</arg_value>
<arg_key>mode</arg_key>
<arg_value>edit</arg_value>
</tool_call>

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
	AddressType *entities.AddressType

	// Pagination
	Limit  *int
	Offset *int
}

