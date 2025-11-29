package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/domain/orders/repositories"
	"erpgo/pkg/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// PostgresOrderRepository implements OrderRepository for PostgreSQL
type PostgresOrderRepository struct {
	db *database.Database
}

// NewPostgresOrderRepository creates a new PostgreSQL order repository
func NewPostgresOrderRepository(db *database.Database) *PostgresOrderRepository {
	return &PostgresOrderRepository{
		db: db,
	}
}

// Create creates a new order
func (r *PostgresOrderRepository) Create(ctx context.Context, order *entities.Order) error {
	query := `
		INSERT INTO orders (
			id, order_number, customer_id, status, previous_status, priority, type,
			payment_status, shipping_method, subtotal, tax_amount, shipping_amount,
			discount_amount, total_amount, paid_amount, refunded_amount, currency,
			order_date, required_date, shipping_date, delivery_date, cancelled_date,
			shipping_address_id, billing_address_id, notes, internal_notes,
			customer_notes, tracking_number, carrier, created_by, approved_by,
			shipped_by, approved_at, shipped_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28,
			$29, $30, $31, $32, $33, $34
		)
	`

	_, err := r.db.Exec(ctx, query,
		order.ID,
		order.OrderNumber,
		order.CustomerID,
		order.Status,
		order.PreviousStatus,
		order.Priority,
		order.Type,
		order.PaymentStatus,
		order.ShippingMethod,
		order.Subtotal,
		order.TaxAmount,
		order.ShippingAmount,
		order.DiscountAmount,
		order.TotalAmount,
		order.PaidAmount,
		order.RefundedAmount,
		order.Currency,
		order.OrderDate,
		order.RequiredDate,
		order.ShippingDate,
		order.DeliveryDate,
		order.CancelledDate,
		order.ShippingAddressID,
		order.BillingAddressID,
		order.Notes,
		order.InternalNotes,
		order.CustomerNotes,
		order.TrackingNumber,
		order.Carrier,
		order.CreatedBy,
		order.ApprovedBy,
		order.ShippedBy,
		order.ApprovedAt,
		order.ShippedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

// GetByID retrieves an order by ID
func (r *PostgresOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Order, error) {
	query := `
		SELECT
			id, order_number, customer_id, status, previous_status, priority, type,
			payment_status, shipping_method, subtotal, tax_amount, shipping_amount,
			discount_amount, total_amount, paid_amount, refunded_amount, currency,
			order_date, required_date, shipping_date, delivery_date, cancelled_date,
			shipping_address_id, billing_address_id, notes, internal_notes,
			customer_notes, tracking_number, carrier, created_by, approved_by,
			shipped_by, created_at, updated_at, approved_at, shipped_at
		FROM orders
		WHERE id = $1
	`

	order := &entities.Order{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.OrderNumber,
		&order.CustomerID,
		&order.Status,
		&order.PreviousStatus,
		&order.Priority,
		&order.Type,
		&order.PaymentStatus,
		&order.ShippingMethod,
		&order.Subtotal,
		&order.TaxAmount,
		&order.ShippingAmount,
		&order.DiscountAmount,
		&order.TotalAmount,
		&order.PaidAmount,
		&order.RefundedAmount,
		&order.Currency,
		&order.OrderDate,
		&order.RequiredDate,
		&order.ShippingDate,
		&order.DeliveryDate,
		&order.CancelledDate,
		&order.ShippingAddressID,
		&order.BillingAddressID,
		&order.Notes,
		&order.InternalNotes,
		&order.CustomerNotes,
		&order.TrackingNumber,
		&order.Carrier,
		&order.CreatedBy,
		&order.ApprovedBy,
		&order.ShippedBy,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.ApprovedAt,
		&order.ShippedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("order with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get order by id: %w", err)
	}

	return order, nil
}

// GetByOrderNumber retrieves an order by order number
func (r *PostgresOrderRepository) GetByOrderNumber(ctx context.Context, orderNumber string) (*entities.Order, error) {
	query := `
		SELECT
			id, order_number, customer_id, status, previous_status, priority, type,
			payment_status, shipping_method, subtotal, tax_amount, shipping_amount,
			discount_amount, total_amount, paid_amount, refunded_amount, currency,
			order_date, required_date, shipping_date, delivery_date, cancelled_date,
			shipping_address_id, billing_address_id, notes, internal_notes,
			customer_notes, tracking_number, carrier, created_by, approved_by,
			shipped_by, created_at, updated_at, approved_at, shipped_at
		FROM orders
		WHERE order_number = $1
	`

	order := &entities.Order{}
	err := r.db.QueryRow(ctx, query, orderNumber).Scan(
		&order.ID,
		&order.OrderNumber,
		&order.CustomerID,
		&order.Status,
		&order.PreviousStatus,
		&order.Priority,
		&order.Type,
		&order.PaymentStatus,
		&order.ShippingMethod,
		&order.Subtotal,
		&order.TaxAmount,
		&order.ShippingAmount,
		&order.DiscountAmount,
		&order.TotalAmount,
		&order.PaidAmount,
		&order.RefundedAmount,
		&order.Currency,
		&order.OrderDate,
		&order.RequiredDate,
		&order.ShippingDate,
		&order.DeliveryDate,
		&order.CancelledDate,
		&order.ShippingAddressID,
		&order.BillingAddressID,
		&order.Notes,
		&order.InternalNotes,
		&order.CustomerNotes,
		&order.TrackingNumber,
		&order.Carrier,
		&order.CreatedBy,
		&order.ApprovedBy,
		&order.ShippedBy,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.ApprovedAt,
		&order.ShippedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("order with number %s not found", orderNumber)
		}
		return nil, fmt.Errorf("failed to get order by number: %w", err)
	}

	return order, nil
}

// Update updates an order
func (r *PostgresOrderRepository) Update(ctx context.Context, order *entities.Order) error {
	query := `
		UPDATE orders SET
			customer_id = $2, status = $3, previous_status = $4, priority = $5, type = $6,
			payment_status = $7, shipping_method = $8, subtotal = $9, tax_amount = $10,
			shipping_amount = $11, discount_amount = $12, total_amount = $13,
			paid_amount = $14, refunded_amount = $15, currency = $16, required_date = $17,
			shipping_date = $18, delivery_date = $19, cancelled_date = $20,
			shipping_address_id = $21, billing_address_id = $22, notes = $23,
			internal_notes = $24, customer_notes = $25, tracking_number = $26,
			carrier = $27, approved_by = $28, shipped_by = $29, approved_at = $30,
			shipped_at = $31, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		order.ID,
		order.CustomerID,
		order.Status,
		order.PreviousStatus,
		order.Priority,
		order.Type,
		order.PaymentStatus,
		order.ShippingMethod,
		order.Subtotal,
		order.TaxAmount,
		order.ShippingAmount,
		order.DiscountAmount,
		order.TotalAmount,
		order.PaidAmount,
		order.RefundedAmount,
		order.Currency,
		order.RequiredDate,
		order.ShippingDate,
		order.DeliveryDate,
		order.CancelledDate,
		order.ShippingAddressID,
		order.BillingAddressID,
		order.Notes,
		order.InternalNotes,
		order.CustomerNotes,
		order.TrackingNumber,
		order.Carrier,
		order.ApprovedBy,
		order.ShippedBy,
		order.ApprovedAt,
		order.ShippedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	return nil
}

// Delete deletes an order
func (r *PostgresOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM orders WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	return nil
}

// List retrieves a list of orders with filtering
func (r *PostgresOrderRepository) List(ctx context.Context, filter repositories.OrderFilter) ([]*entities.Order, error) {
	baseQuery, args, err := r.buildOrderQuery(filter, false)
	if err != nil {
		return nil, fmt.Errorf("failed to build order query: %w", err)
	}

	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	var orders []*entities.Order
	for rows.Next() {
		order := &entities.Order{}
		err := rows.Scan(
			&order.ID,
			&order.OrderNumber,
			&order.CustomerID,
			&order.Status,
			&order.PreviousStatus,
			&order.Priority,
			&order.Type,
			&order.PaymentStatus,
			&order.ShippingMethod,
			&order.Subtotal,
			&order.TaxAmount,
			&order.ShippingAmount,
			&order.DiscountAmount,
			&order.TotalAmount,
			&order.PaidAmount,
			&order.RefundedAmount,
			&order.Currency,
			&order.OrderDate,
			&order.RequiredDate,
			&order.ShippingDate,
			&order.DeliveryDate,
			&order.CancelledDate,
			&order.ShippingAddressID,
			&order.BillingAddressID,
			&order.Notes,
			&order.InternalNotes,
			&order.CustomerNotes,
			&order.TrackingNumber,
			&order.Carrier,
			&order.CreatedBy,
			&order.ApprovedBy,
			&order.ShippedBy,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.ApprovedAt,
			&order.ShippedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order row: %w", err)
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order rows: %w", err)
	}

	return orders, nil
}

// Count returns the count of orders matching the filter
func (r *PostgresOrderRepository) Count(ctx context.Context, filter repositories.OrderFilter) (int, error) {
	baseQuery, args, err := r.buildOrderQuery(filter, true)
	if err != nil {
		return 0, fmt.Errorf("failed to build order count query: %w", err)
	}

	var count int
	err = r.db.QueryRow(ctx, baseQuery, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count orders: %w", err)
	}

	return count, nil
}

// buildOrderQuery builds the SQL query for orders based on filter
func (r *PostgresOrderRepository) buildOrderQuery(filter repositories.OrderFilter, isCount bool) (string, []interface{}, error) {
	var selectClause string
	if isCount {
		selectClause = "SELECT COUNT(*)"
	} else {
		selectClause = `
			SELECT
				id, order_number, customer_id, status, previous_status, priority, type,
				payment_status, shipping_method, subtotal, tax_amount, shipping_amount,
				discount_amount, total_amount, paid_amount, refunded_amount, currency,
				order_date, required_date, shipping_date, delivery_date, cancelled_date,
				shipping_address_id, billing_address_id, notes, internal_notes,
				customer_notes, tracking_number, carrier, created_by, approved_by,
				shipped_by, created_at, updated_at, approved_at, shipped_at
		`
	}

	baseQuery := selectClause + " FROM orders WHERE 1=1"

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Search filter
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(order_number ILIKE $%d OR EXISTS (SELECT 1 FROM order_items oi WHERE oi.order_id = orders.id AND (oi.product_sku ILIKE $%d OR oi.product_name ILIKE $%d)))",
			argIndex, argIndex+1, argIndex+2))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		argIndex += 3
	}

	// Status filter
	if len(filter.Status) > 0 {
		placeholders := make([]string, len(filter.Status))
		for i, status := range filter.Status {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, status)
			argIndex++
		}
		conditions = append(conditions, "status IN ("+strings.Join(placeholders, ", ")+")")
	}

	// Payment status filter
	if len(filter.PaymentStatus) > 0 {
		placeholders := make([]string, len(filter.PaymentStatus))
		for i, status := range filter.PaymentStatus {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, status)
			argIndex++
		}
		conditions = append(conditions, "payment_status IN ("+strings.Join(placeholders, ", ")+")")
	}

	// Priority filter
	if len(filter.Priority) > 0 {
		placeholders := make([]string, len(filter.Priority))
		for i, priority := range filter.Priority {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, priority)
			argIndex++
		}
		conditions = append(conditions, "priority IN ("+strings.Join(placeholders, ", ")+")")
	}

	// Type filter
	if len(filter.Type) > 0 {
		placeholders := make([]string, len(filter.Type))
		for i, orderType := range filter.Type {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, orderType)
			argIndex++
		}
		conditions = append(conditions, "type IN ("+strings.Join(placeholders, ", ")+")")
	}

	// Shipping method filter
	if len(filter.ShippingMethod) > 0 {
		placeholders := make([]string, len(filter.ShippingMethod))
		for i, method := range filter.ShippingMethod {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, method)
			argIndex++
		}
		conditions = append(conditions, "shipping_method IN ("+strings.Join(placeholders, ", ")+")")
	}

	// Customer filter
	if filter.CustomerID != nil {
		conditions = append(conditions, fmt.Sprintf("customer_id = $%d", argIndex))
		args = append(args, *filter.CustomerID)
		argIndex++
	}

	// Date filters
	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("order_date >= $%d", argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("order_date <= $%d", argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	// Financial filters
	if filter.MinTotalAmount != nil {
		conditions = append(conditions, fmt.Sprintf("total_amount >= $%d", argIndex))
		args = append(args, *filter.MinTotalAmount)
		argIndex++
	}

	if filter.MaxTotalAmount != nil {
		conditions = append(conditions, fmt.Sprintf("total_amount <= $%d", argIndex))
		args = append(args, *filter.MaxTotalAmount)
		argIndex++
	}

	// Currency filter
	if filter.Currency != "" {
		conditions = append(conditions, fmt.Sprintf("currency = $%d", argIndex))
		args = append(args, filter.Currency)
		argIndex++
	}

	// Created by filter
	if filter.CreatedBy != nil {
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argIndex))
		args = append(args, *filter.CreatedBy)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY for non-count queries
	if !isCount {
		sortBy := "order_date"
		if filter.SortBy != "" {
			sortBy = filter.SortBy
		}

		sortOrder := "DESC"
		if filter.SortOrder != "" {
			sortOrder = strings.ToUpper(filter.SortOrder)
		}
		baseQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

		// Add LIMIT and OFFSET for pagination
		if filter.Limit > 0 {
			baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
			args = append(args, filter.Limit)
			argIndex++

			if filter.Offset > 0 {
				baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
				args = append(args, filter.Offset)
			} else if filter.Page > 1 {
				offset := (filter.Page - 1) * filter.Limit
				baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
				args = append(args, offset)
			}
		}
	}

	return baseQuery, args, nil
}

// GetByStatus retrieves orders by status
func (r *PostgresOrderRepository) GetByStatus(ctx context.Context, status entities.OrderStatus) ([]*entities.Order, error) {
	filter := repositories.OrderFilter{
		Status: []entities.OrderStatus{status},
		Limit:  1000, // Reasonable default
	}
	return r.List(ctx, filter)
}

// GetByStatusAndDateRange retrieves orders by status within a date range
func (r *PostgresOrderRepository) GetByStatusAndDateRange(ctx context.Context, status entities.OrderStatus, startDate, endDate time.Time) ([]*entities.Order, error) {
	filter := repositories.OrderFilter{
		Status:    []entities.OrderStatus{status},
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     1000,
	}
	return r.List(ctx, filter)
}

// UpdateStatus updates the status of an order
func (r *PostgresOrderRepository) UpdateStatus(ctx context.Context, orderID uuid.UUID, newStatus entities.OrderStatus, updatedBy uuid.UUID) error {
	// First get the current status to update previous_status
	query := `UPDATE orders SET status = $2, previous_status = status, updated_at = CURRENT_TIMESTAMP WHERE id = $1`

	_, err := r.db.Exec(ctx, query, orderID, newStatus)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

// GetByCustomerID retrieves orders for a specific customer
func (r *PostgresOrderRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID, filter repositories.OrderFilter) ([]*entities.Order, error) {
	filter.CustomerID = &customerID
	return r.List(ctx, filter)
}

// GetCustomerOrderHistory retrieves customer's order history
func (r *PostgresOrderRepository) GetCustomerOrderHistory(ctx context.Context, customerID uuid.UUID, limit int) ([]*entities.Order, error) {
	filter := repositories.OrderFilter{
		CustomerID: &customerID,
		Limit:      limit,
		SortBy:     "order_date",
		SortOrder:  "DESC",
	}
	return r.List(ctx, filter)
}

// GetByDateRange retrieves orders within a date range
func (r *PostgresOrderRepository) GetByDateRange(ctx context.Context, startDate, endDate time.Time, filter repositories.OrderFilter) ([]*entities.Order, error) {
	filter.StartDate = &startDate
	filter.EndDate = &endDate
	return r.List(ctx, filter)
}

// GetOverdueOrders retrieves overdue orders
func (r *PostgresOrderRepository) GetOverdueOrders(ctx context.Context) ([]*entities.Order, error) {
	query := `
		SELECT
			id, order_number, customer_id, status, previous_status, priority, type,
			payment_status, shipping_method, subtotal, tax_amount, shipping_amount,
			discount_amount, total_amount, paid_amount, refunded_amount, currency,
			order_date, required_date, shipping_date, delivery_date, cancelled_date,
			shipping_address_id, billing_address_id, notes, internal_notes,
			customer_notes, tracking_number, carrier, created_by, approved_by,
			shipped_by, created_at, updated_at, approved_at, shipped_at
		FROM orders
		WHERE payment_status != 'PAID'
		AND required_date < CURRENT_DATE
		AND status NOT IN ('CANCELLED', 'DELIVERED', 'REFUNDED')
		ORDER BY required_date ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue orders: %w", err)
	}
	defer rows.Close()

	var orders []*entities.Order
	for rows.Next() {
		order := &entities.Order{}
		err := rows.Scan(
			&order.ID,
			&order.OrderNumber,
			&order.CustomerID,
			&order.Status,
			&order.PreviousStatus,
			&order.Priority,
			&order.Type,
			&order.PaymentStatus,
			&order.ShippingMethod,
			&order.Subtotal,
			&order.TaxAmount,
			&order.ShippingAmount,
			&order.DiscountAmount,
			&order.TotalAmount,
			&order.PaidAmount,
			&order.RefundedAmount,
			&order.Currency,
			&order.OrderDate,
			&order.RequiredDate,
			&order.ShippingDate,
			&order.DeliveryDate,
			&order.CancelledDate,
			&order.ShippingAddressID,
			&order.BillingAddressID,
			&order.Notes,
			&order.InternalNotes,
			&order.CustomerNotes,
			&order.TrackingNumber,
			&order.Carrier,
			&order.CreatedBy,
			&order.ApprovedBy,
			&order.ShippedBy,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.ApprovedAt,
			&order.ShippedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan overdue order row: %w", err)
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating overdue order rows: %w", err)
	}

	return orders, nil
}

// GetPendingOrders retrieves pending orders
func (r *PostgresOrderRepository) GetPendingOrders(ctx context.Context) ([]*entities.Order, error) {
	filter := repositories.OrderFilter{
		Status: []entities.OrderStatus{entities.OrderStatusPending},
		Limit:  1000,
	}
	return r.List(ctx, filter)
}

// GetUnpaidOrders retrieves unpaid orders
func (r *PostgresOrderRepository) GetUnpaidOrders(ctx context.Context) ([]*entities.Order, error) {
	query := `
		SELECT
			id, order_number, customer_id, status, previous_status, priority, type,
			payment_status, shipping_method, subtotal, tax_amount, shipping_amount,
			discount_amount, total_amount, paid_amount, refunded_amount, currency,
			order_date, required_date, shipping_date, delivery_date, cancelled_date,
			shipping_address_id, billing_address_id, notes, internal_notes,
			customer_notes, tracking_number, carrier, created_by, approved_by,
			shipped_by, created_at, updated_at, approved_at, shipped_at
		FROM orders
		WHERE payment_status IN ('PENDING', 'PARTIALLY_PAID', 'OVERDUE')
		AND status NOT IN ('CANCELLED', 'REFUNDED')
		ORDER BY order_date ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get unpaid orders: %w", err)
	}
	defer rows.Close()

	var orders []*entities.Order
	for rows.Next() {
		order := &entities.Order{}
		err := rows.Scan(
			&order.ID,
			&order.OrderNumber,
			&order.CustomerID,
			&order.Status,
			&order.PreviousStatus,
			&order.Priority,
			&order.Type,
			&order.PaymentStatus,
			&order.ShippingMethod,
			&order.Subtotal,
			&order.TaxAmount,
			&order.ShippingAmount,
			&order.DiscountAmount,
			&order.TotalAmount,
			&order.PaidAmount,
			&order.RefundedAmount,
			&order.Currency,
			&order.OrderDate,
			&order.RequiredDate,
			&order.ShippingDate,
			&order.DeliveryDate,
			&order.CancelledDate,
			&order.ShippingAddressID,
			&order.BillingAddressID,
			&order.Notes,
			&order.InternalNotes,
			&order.CustomerNotes,
			&order.TrackingNumber,
			&order.Carrier,
			&order.CreatedBy,
			&order.ApprovedBy,
			&order.ShippedBy,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.ApprovedAt,
			&order.ShippedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan unpaid order row: %w", err)
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating unpaid order rows: %w", err)
	}

	return orders, nil
}

// GetOrdersByPaymentStatus retrieves orders by payment status
func (r *PostgresOrderRepository) GetOrdersByPaymentStatus(ctx context.Context, paymentStatus entities.PaymentStatus) ([]*entities.Order, error) {
	filter := repositories.OrderFilter{
		PaymentStatus: []entities.PaymentStatus{paymentStatus},
		Limit:         1000,
	}
	return r.List(ctx, filter)
}

// UpdatePaymentStatus updates the payment status of an order
func (r *PostgresOrderRepository) UpdatePaymentStatus(ctx context.Context, orderID uuid.UUID, paymentStatus entities.PaymentStatus, paidAmount decimal.Decimal) error {
	query := `
		UPDATE orders
		SET payment_status = $2, paid_amount = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, orderID, paymentStatus, paidAmount)
	if err != nil {
		return fmt.Errorf("failed to update order payment status: %w", err)
	}

	return nil
}

// BulkUpdateStatus updates status for multiple orders
func (r *PostgresOrderRepository) BulkUpdateStatus(ctx context.Context, orderIDs []uuid.UUID, newStatus entities.OrderStatus, updatedBy uuid.UUID) error {
	if len(orderIDs) == 0 {
		return nil
	}

	// Create placeholders for IN clause
	placeholders := make([]string, len(orderIDs))
	args := make([]interface{}, len(orderIDs))
	for i, id := range orderIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2) // Start from $2 because $1 is newStatus
		args[i] = id
	}

	query := fmt.Sprintf(`
		UPDATE orders
		SET status = $1, previous_status = status, updated_at = CURRENT_TIMESTAMP
		WHERE id IN (%s)
	`, strings.Join(placeholders, ", "))

	args = append([]interface{}{newStatus}, args...)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk update order status: %w", err)
	}

	return nil
}

// BulkCreate creates multiple orders
func (r *PostgresOrderRepository) BulkCreate(ctx context.Context, orders []*entities.Order) error {
	if len(orders) == 0 {
		return nil
	}

	// For bulk operations, we'll use a transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO orders (
			id, order_number, customer_id, status, previous_status, priority, type,
			payment_status, shipping_method, subtotal, tax_amount, shipping_amount,
			discount_amount, total_amount, paid_amount, refunded_amount, currency,
			order_date, required_date, shipping_date, delivery_date, cancelled_date,
			shipping_address_id, billing_address_id, notes, internal_notes,
			customer_notes, tracking_number, carrier, created_by, approved_by,
			shipped_by, approved_at, shipped_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28,
			$29, $30, $31, $32, $33, $34
		)
	`

	for _, order := range orders {
		_, err := tx.Exec(ctx, query,
			order.ID,
			order.OrderNumber,
			order.CustomerID,
			order.Status,
			order.PreviousStatus,
			order.Priority,
			order.Type,
			order.PaymentStatus,
			order.ShippingMethod,
			order.Subtotal,
			order.TaxAmount,
			order.ShippingAmount,
			order.DiscountAmount,
			order.TotalAmount,
			order.PaidAmount,
			order.RefundedAmount,
			order.Currency,
			order.OrderDate,
			order.RequiredDate,
			order.ShippingDate,
			order.DeliveryDate,
			order.CancelledDate,
			order.ShippingAddressID,
			order.BillingAddressID,
			order.Notes,
			order.InternalNotes,
			order.CustomerNotes,
			order.TrackingNumber,
			order.Carrier,
			order.CreatedBy,
			order.ApprovedBy,
			order.ShippedBy,
			order.ApprovedAt,
			order.ShippedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to bulk create order: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetOrderStats retrieves order statistics
func (r *PostgresOrderRepository) GetOrderStats(ctx context.Context, filter repositories.OrderStatsFilter) (*repositories.OrderStats, error) {
	query := `
		SELECT
			COUNT(*) as total_orders,
			COALESCE(SUM(total_amount), 0) as total_revenue,
			COALESCE(AVG(total_amount), 0) as average_order_value
		FROM orders
		WHERE order_date >= $1 AND order_date <= $2
	`

	args := []interface{}{filter.StartDate, filter.EndDate}
	argIndex := 3

	if len(filter.Status) > 0 {
		placeholders := make([]string, len(filter.Status))
		for i, status := range filter.Status {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, status)
			argIndex++
		}
		query += " AND status IN (" + strings.Join(placeholders, ", ") + ")"
	}

	if filter.CustomerID != nil {
		query += fmt.Sprintf(" AND customer_id = $%d", argIndex)
		args = append(args, *filter.CustomerID)
		argIndex++
	}

	stats := &repositories.OrderStats{}
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&stats.TotalOrders,
		&stats.TotalRevenue,
		&stats.AverageOrderValue,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get order stats: %w", err)
	}

	// Get status counts
	statusQuery := `
		SELECT status, COUNT(*)
		FROM orders
		WHERE order_date >= $1 AND order_date <= $2
	`

	args = []interface{}{filter.StartDate, filter.EndDate}
	argIndex = 3

	if len(filter.Status) > 0 {
		placeholders := make([]string, len(filter.Status))
		for i, status := range filter.Status {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, status)
			argIndex++
		}
		statusQuery += " AND status IN (" + strings.Join(placeholders, ", ") + ")"
	}

	if filter.CustomerID != nil {
		statusQuery += fmt.Sprintf(" AND customer_id = $%d", argIndex)
		args = append(args, *filter.CustomerID)
	}

	statusQuery += " GROUP BY status"

	rows, err := r.db.Query(ctx, statusQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}
	defer rows.Close()

	stats.StatusCounts = make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		err := rows.Scan(&status, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan status count: %w", err)
		}
		stats.StatusCounts[status] = count
	}

	// Get payment status counts
	paymentQuery := `
		SELECT payment_status, COUNT(*)
		FROM orders
		WHERE order_date >= $1 AND order_date <= $2
	`

	args = []interface{}{filter.StartDate, filter.EndDate}
	argIndex = 3

	if len(filter.Status) > 0 {
		placeholders := make([]string, len(filter.Status))
		for i, status := range filter.Status {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, status)
			argIndex++
		}
		paymentQuery += " AND status IN (" + strings.Join(placeholders, ", ") + ")"
	}

	if filter.CustomerID != nil {
		paymentQuery += fmt.Sprintf(" AND customer_id = $%d", argIndex)
		args = append(args, *filter.CustomerID)
	}

	paymentQuery += " GROUP BY payment_status"

	rows, err = r.db.Query(ctx, paymentQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment status counts: %w", err)
	}
	defer rows.Close()

	stats.PaymentStatusCounts = make(map[string]int64)
	for rows.Next() {
		var paymentStatus string
		var count int64
		err := rows.Scan(&paymentStatus, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment status count: %w", err)
		}
		stats.PaymentStatusCounts[paymentStatus] = count
	}

	return stats, nil
}

// GetRevenueByPeriod retrieves revenue data grouped by time period
func (r *PostgresOrderRepository) GetRevenueByPeriod(ctx context.Context, startDate, endDate time.Time, groupBy string) ([]*repositories.RevenueByPeriod, error) {
	analyticsRepo := NewPostgresOrderAnalyticsRepository(r.db)
	return analyticsRepo.GetRevenueByPeriod(ctx, startDate, endDate, groupBy)
}

// GetTopCustomers retrieves top customers by revenue
func (r *PostgresOrderRepository) GetTopCustomers(ctx context.Context, startDate, endDate time.Time, limit int) ([]*repositories.CustomerOrderStats, error) {
	analyticsRepo := NewPostgresOrderAnalyticsRepository(r.db)
	return analyticsRepo.GetTopCustomers(ctx, startDate, endDate, limit)
}

// GetSalesByProduct retrieves top-selling products
func (r *PostgresOrderRepository) GetSalesByProduct(ctx context.Context, startDate, endDate time.Time, limit int) ([]*repositories.ProductSalesStats, error) {
	analyticsRepo := NewPostgresOrderAnalyticsRepository(r.db)
	return analyticsRepo.GetSalesByProduct(ctx, startDate, endDate, limit)
}

// SearchOrders performs advanced search on orders
func (r *PostgresOrderRepository) SearchOrders(ctx context.Context, query string, filter repositories.OrderFilter) ([]*entities.Order, error) {
	filter.Search = query
	return r.List(ctx, filter)
}

// GetOrdersWithItems retrieves orders with their items
func (r *PostgresOrderRepository) GetOrdersWithItems(ctx context.Context, orderIDs []uuid.UUID) ([]*entities.Order, error) {
	if len(orderIDs) == 0 {
		return []*entities.Order{}, nil
	}

	// Create placeholders for IN clause
	placeholders := make([]string, len(orderIDs))
	args := make([]interface{}, len(orderIDs))
	for i, id := range orderIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT
			id, order_number, customer_id, status, previous_status, priority, type,
			payment_status, shipping_method, subtotal, tax_amount, shipping_amount,
			discount_amount, total_amount, paid_amount, refunded_amount, currency,
			order_date, required_date, shipping_date, delivery_date, cancelled_date,
			shipping_address_id, billing_address_id, notes, internal_notes,
			customer_notes, tracking_number, carrier, created_by, approved_by,
			shipped_by, created_at, updated_at, approved_at, shipped_at
		FROM orders
		WHERE id IN (%s)
		ORDER BY order_date DESC
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders with items: %w", err)
	}
	defer rows.Close()

	var orders []*entities.Order
	for rows.Next() {
		order := &entities.Order{}
		err := rows.Scan(
			&order.ID,
			&order.OrderNumber,
			&order.CustomerID,
			&order.Status,
			&order.PreviousStatus,
			&order.Priority,
			&order.Type,
			&order.PaymentStatus,
			&order.ShippingMethod,
			&order.Subtotal,
			&order.TaxAmount,
			&order.ShippingAmount,
			&order.DiscountAmount,
			&order.TotalAmount,
			&order.PaidAmount,
			&order.RefundedAmount,
			&order.Currency,
			&order.OrderDate,
			&order.RequiredDate,
			&order.ShippingDate,
			&order.DeliveryDate,
			&order.CancelledDate,
			&order.ShippingAddressID,
			&order.BillingAddressID,
			&order.Notes,
			&order.InternalNotes,
			&order.CustomerNotes,
			&order.TrackingNumber,
			&order.Carrier,
			&order.CreatedBy,
			&order.ApprovedBy,
			&order.ShippedBy,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.ApprovedAt,
			&order.ShippedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order row: %w", err)
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order rows: %w", err)
	}

	return orders, nil
}

// ExistsByOrderNumber checks if an order exists by order number
func (r *PostgresOrderRepository) ExistsByOrderNumber(ctx context.Context, orderNumber string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM orders WHERE order_number = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, orderNumber).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if order exists by number: %w", err)
	}

	return exists, nil
}

// GenerateUniqueOrderNumber generates a unique order number
func (r *PostgresOrderRepository) GenerateUniqueOrderNumber(ctx context.Context) (string, error) {
	// Generate order number with format: ORD-YYYYMMDD-XXXXX
	dateStr := time.Now().Format("20060102")

	for i := 0; i < 10; i++ { // Try 10 times to generate unique number
		randomSuffix := fmt.Sprintf("%05d", time.Now().UnixNano()%100000)
		orderNumber := fmt.Sprintf("ORD-%s-%s", dateStr, randomSuffix)

		exists, err := r.ExistsByOrderNumber(ctx, orderNumber)
		if err != nil {
			return "", fmt.Errorf("failed to check order number uniqueness: %w", err)
		}

		if !exists {
			return orderNumber, nil
		}

		// Wait a bit before trying again
		time.Sleep(time.Millisecond * 10)
	}

	return "", fmt.Errorf("failed to generate unique order number after 10 attempts")
}
