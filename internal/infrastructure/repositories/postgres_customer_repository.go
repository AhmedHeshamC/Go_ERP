package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/domain/orders/repositories"
	"erpgo/pkg/database"
)

// PostgresCustomerRepository implements CustomerRepository for PostgreSQL
type PostgresCustomerRepository struct {
	db *database.Database
}

// NewPostgresCustomerRepository creates a new PostgreSQL customer repository
func NewPostgresCustomerRepository(db *database.Database) *PostgresCustomerRepository {
	return &PostgresCustomerRepository{
		db: db,
	}
}

// Create creates a new customer and returns the created customer
func (r *PostgresCustomerRepository) Create(ctx context.Context, customer *entities.Customer) (*entities.Customer, error) {
	query := `
		INSERT INTO customers (
			id, customer_code, company_id, type, first_name, last_name, email,
			phone, website, company_name, tax_id, industry, credit_limit,
			credit_used, terms, is_active, is_vat_exempt, preferred_currency,
			notes, source
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20
		)
	`

	_, err := r.db.Exec(ctx, query,
		customer.ID,
		customer.CustomerCode,
		customer.CompanyID,
		customer.Type,
		customer.FirstName,
		customer.LastName,
		customer.Email,
		customer.Phone,
		customer.Website,
		customer.CompanyName,
		customer.TaxID,
		customer.Industry,
		customer.CreditLimit,
		customer.CreditUsed,
		customer.Terms,
		customer.IsActive,
		customer.IsVATExempt,
		customer.PreferredCurrency,
		customer.Notes,
		customer.Source,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	return customer, nil
}

// GetByID retrieves a customer by ID
func (r *PostgresCustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Customer, error) {
	query := `
		SELECT
			id, customer_code, company_id, type, first_name, last_name, email,
			phone, website, company_name, tax_id, industry, credit_limit,
			credit_used, terms, is_active, is_vat_exempt, preferred_currency,
			notes, source, created_at, updated_at
		FROM customers
		WHERE id = $1
	`

	customer := &entities.Customer{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&customer.ID,
		&customer.CustomerCode,
		&customer.CompanyID,
		&customer.Type,
		&customer.FirstName,
		&customer.LastName,
		&customer.Email,
		&customer.Phone,
		&customer.Website,
		&customer.CompanyName,
		&customer.TaxID,
		&customer.Industry,
		&customer.CreditLimit,
		&customer.CreditUsed,
		&customer.Terms,
		&customer.IsActive,
		&customer.IsVATExempt,
		&customer.PreferredCurrency,
		&customer.Notes,
		&customer.Source,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get customer by id: %w", err)
	}

	return customer, nil
}

// GetByCustomerCode retrieves a customer by customer code
func (r *PostgresCustomerRepository) GetByCustomerCode(ctx context.Context, customerCode string) (*entities.Customer, error) {
	query := `
		SELECT
			id, customer_code, company_id, type, first_name, last_name, email,
			phone, website, company_name, tax_id, industry, credit_limit,
			credit_used, terms, is_active, is_vat_exempt, preferred_currency,
			notes, source, created_at, updated_at
		FROM customers
		WHERE customer_code = $1
	`

	customer := &entities.Customer{}
	err := r.db.QueryRow(ctx, query, customerCode).Scan(
		&customer.ID,
		&customer.CustomerCode,
		&customer.CompanyID,
		&customer.Type,
		&customer.FirstName,
		&customer.LastName,
		&customer.Email,
		&customer.Phone,
		&customer.Website,
		&customer.CompanyName,
		&customer.TaxID,
		&customer.Industry,
		&customer.CreditLimit,
		&customer.CreditUsed,
		&customer.Terms,
		&customer.IsActive,
		&customer.IsVATExempt,
		&customer.PreferredCurrency,
		&customer.Notes,
		&customer.Source,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer with code %s not found", customerCode)
		}
		return nil, fmt.Errorf("failed to get customer by code: %w", err)
	}

	return customer, nil
}

// GetByCode retrieves a customer by code (alias for GetByCustomerCode)
func (r *PostgresCustomerRepository) GetByCode(ctx context.Context, code string) (*entities.Customer, error) {
	return r.GetByCustomerCode(ctx, code)
}

// GetByEmail retrieves a customer by email
func (r *PostgresCustomerRepository) GetByEmail(ctx context.Context, email string) (*entities.Customer, error) {
	query := `
		SELECT
			id, customer_code, company_id, type, first_name, last_name, email,
			phone, website, company_name, tax_id, industry, credit_limit,
			credit_used, terms, is_active, is_vat_exempt, preferred_currency,
			notes, source, created_at, updated_at
		FROM customers
		WHERE email = $1
	`

	customer := &entities.Customer{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&customer.ID,
		&customer.CustomerCode,
		&customer.CompanyID,
		&customer.Type,
		&customer.FirstName,
		&customer.LastName,
		&customer.Email,
		&customer.Phone,
		&customer.Website,
		&customer.CompanyName,
		&customer.TaxID,
		&customer.Industry,
		&customer.CreditLimit,
		&customer.CreditUsed,
		&customer.Terms,
		&customer.IsActive,
		&customer.IsVATExempt,
		&customer.PreferredCurrency,
		&customer.Notes,
		&customer.Source,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer with email %s not found", email)
		}
		return nil, fmt.Errorf("failed to get customer by email: %w", err)
	}

	return customer, nil
}

// GetStats retrieves customer statistics
func (r *PostgresCustomerRepository) GetStats(ctx context.Context, startDate, endDate *time.Time) (*repositories.CustomerStats, error) {
	var start, end time.Time
	if startDate != nil {
		start = *startDate
	} else {
		// Default to 30 days ago
		start = time.Now().AddDate(0, 0, -30)
	}

	if endDate != nil {
		end = *endDate
	} else {
		// Default to now
		end = time.Now()
	}

	filter := repositories.CustomerStatsFilter{
		StartDate: start,
		EndDate:   end,
	}

	return r.GetCustomerStats(ctx, filter)
}

// GetCustomerOrders retrieves customer orders
func (r *PostgresCustomerRepository) GetCustomerOrders(ctx context.Context, customerID string, limit int) ([]*entities.Order, error) {
	customerUUID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, fmt.Errorf("invalid customer ID: %w", err)
	}

	query := `
		SELECT
			id, order_number, customer_id, status, previous_status, priority,
			type, payment_status, shipping_method, subtotal, tax_amount,
			shipping_amount, discount_amount, total_amount, paid_amount,
			refunded_amount, currency, order_date, required_date,
			shipping_date, delivery_date, cancelled_date, shipping_address_id,
			billing_address_id, notes, internal_notes, customer_notes,
			tracking_number, carrier, created_by, approved_by, shipped_by,
			created_at, updated_at, approved_at, shipped_at
		FROM orders
		WHERE customer_id = $1
		ORDER BY order_date DESC
		LIMIT $2
	`

	if limit <= 0 {
		limit = 100 // Default limit
	}

	rows, err := r.db.Query(ctx, query, customerUUID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer orders: %w", err)
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

// TransferOrders transfers orders from one customer to another
func (r *PostgresCustomerRepository) TransferOrders(ctx context.Context, fromCustomerID, toCustomerID string) error {
	fromUUID, err := uuid.Parse(fromCustomerID)
	if err != nil {
		return fmt.Errorf("invalid from customer ID: %w", err)
	}

	toUUID, err := uuid.Parse(toCustomerID)
	if err != nil {
		return fmt.Errorf("invalid to customer ID: %w", err)
	}

	query := `UPDATE orders SET customer_id = $1 WHERE customer_id = $2`

	_, err = r.db.Exec(ctx, query, toUUID, fromUUID)
	if err != nil {
		return fmt.Errorf("failed to transfer orders: %w", err)
	}

	return nil
}

// Update updates a customer and returns the updated customer
func (r *PostgresCustomerRepository) Update(ctx context.Context, customer *entities.Customer) (*entities.Customer, error) {
	query := `
		UPDATE customers SET
			company_id = $2, type = $3, first_name = $4, last_name = $5,
			email = $6, phone = $7, website = $8, company_name = $9,
			tax_id = $10, industry = $11, credit_limit = $12, credit_used = $13,
			terms = $14, is_active = $15, is_vat_exempt = $16,
			preferred_currency = $17, notes = $18, source = $19,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		customer.ID,
		customer.CompanyID,
		customer.Type,
		customer.FirstName,
		customer.LastName,
		customer.Email,
		customer.Phone,
		customer.Website,
		customer.CompanyName,
		customer.TaxID,
		customer.Industry,
		customer.CreditLimit,
		customer.CreditUsed,
		customer.Terms,
		customer.IsActive,
		customer.IsVATExempt,
		customer.PreferredCurrency,
		customer.Notes,
		customer.Source,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	return customer, nil
}

// Delete deletes a customer
func (r *PostgresCustomerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM customers WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	return nil
}

// List retrieves a list of customers with filtering
func (r *PostgresCustomerRepository) List(ctx context.Context, filter repositories.CustomerFilter) ([]*entities.Customer, error) {
	baseQuery, args, err := r.buildCustomerQuery(filter, false)
	if err != nil {
		return nil, fmt.Errorf("failed to build customer query: %w", err)
	}

	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list customers: %w", err)
	}
	defer rows.Close()

	var customers []*entities.Customer
	for rows.Next() {
		customer := &entities.Customer{}
		err := rows.Scan(
			&customer.ID,
			&customer.CustomerCode,
			&customer.CompanyID,
			&customer.Type,
			&customer.FirstName,
			&customer.LastName,
			&customer.Email,
			&customer.Phone,
			&customer.Website,
			&customer.CompanyName,
			&customer.TaxID,
			&customer.Industry,
			&customer.CreditLimit,
			&customer.CreditUsed,
			&customer.Terms,
			&customer.IsActive,
			&customer.IsVATExempt,
			&customer.PreferredCurrency,
			&customer.Notes,
			&customer.Source,
			&customer.CreatedAt,
			&customer.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		customers = append(customers, customer)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating customer rows: %w", err)
	}

	return customers, nil
}

// Count returns the count of customers matching the filter
func (r *PostgresCustomerRepository) Count(ctx context.Context, filter repositories.CustomerFilter) (int, error) {
	baseQuery, args, err := r.buildCustomerQuery(filter, true)
	if err != nil {
		return 0, fmt.Errorf("failed to build customer count query: %w", err)
	}

	var count int
	err = r.db.QueryRow(ctx, baseQuery, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count customers: %w", err)
	}

	return count, nil
}

// buildCustomerQuery builds the SQL query for customers based on filter
func (r *PostgresCustomerRepository) buildCustomerQuery(filter repositories.CustomerFilter, isCount bool) (string, []interface{}, error) {
	var selectClause string
	if isCount {
		selectClause = "SELECT COUNT(*)"
	} else {
		selectClause = `
			SELECT
				id, customer_code, company_id, type, first_name, last_name, email,
				phone, website, company_name, tax_id, industry, credit_limit,
				credit_used, terms, is_active, is_vat_exempt, preferred_currency,
				notes, source, created_at, updated_at
		`
	}

	baseQuery := selectClause + " FROM customers WHERE 1=1"

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Search filter
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(customer_code ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d OR COALESCE(company_name, '') ILIKE $%d)",
			argIndex, argIndex+1, argIndex+2, argIndex+3, argIndex+4))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern)
		argIndex += 5
	}

	// Type filter
	if filter.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, filter.Type)
		argIndex++
	}

	// Company filter
	if filter.CompanyID != nil {
		conditions = append(conditions, fmt.Sprintf("company_id = $%d", argIndex))
		args = append(args, *filter.CompanyID)
		argIndex++
	}

	// Active status filter
	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	// Industry filter
	if filter.Industry != nil {
		conditions = append(conditions, fmt.Sprintf("industry = $%d", argIndex))
		args = append(args, *filter.Industry)
		argIndex++
	}

	// Source filter
	if filter.Source != "" {
		conditions = append(conditions, fmt.Sprintf("source = $%d", argIndex))
		args = append(args, filter.Source)
		argIndex++
	}

	// Credit limit filters
	if filter.HasCreditLimit != nil {
		if *filter.HasCreditLimit {
			conditions = append(conditions, fmt.Sprintf("credit_limit > $%d", argIndex))
			args = append(args, decimal.Zero)
			argIndex++
		} else {
			conditions = append(conditions, fmt.Sprintf("credit_limit = $%d", argIndex))
			args = append(args, decimal.Zero)
			argIndex++
		}
	}

	if filter.MinCreditLimit != nil {
		conditions = append(conditions, fmt.Sprintf("credit_limit >= $%d", argIndex))
		args = append(args, *filter.MinCreditLimit)
		argIndex++
	}

	if filter.MaxCreditLimit != nil {
		conditions = append(conditions, fmt.Sprintf("credit_limit <= $%d", argIndex))
		args = append(args, *filter.MaxCreditLimit)
		argIndex++
	}

	// Date filters
	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY for non-count queries
	if !isCount {
		sortBy := "created_at"
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

// Search performs advanced search on customers
func (r *PostgresCustomerRepository) Search(ctx context.Context, query string, filter repositories.CustomerFilter) ([]*entities.Customer, error) {
	filter.Search = query
	return r.List(ctx, filter)
}

// ExistsByEmail checks if a customer exists by email
func (r *PostgresCustomerRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM customers WHERE email = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if customer exists by email: %w", err)
	}

	return exists, nil
}

// ExistsByCustomerCode checks if a customer exists by customer code
func (r *PostgresCustomerRepository) ExistsByCustomerCode(ctx context.Context, customerCode string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM customers WHERE customer_code = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, customerCode).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if customer exists by code: %w", err)
	}

	return exists, nil
}

// GetActiveCustomers retrieves active customers
func (r *PostgresCustomerRepository) GetActiveCustomers(ctx context.Context) ([]*entities.Customer, error) {
	filter := repositories.CustomerFilter{
		IsActive: func() *bool { b := true; return &b }(),
		Limit:    1000,
	}
	return r.List(ctx, filter)
}

// GetInactiveCustomers retrieves inactive customers
func (r *PostgresCustomerRepository) GetInactiveCustomers(ctx context.Context) ([]*entities.Customer, error) {
	filter := repositories.CustomerFilter{
		IsActive: func() *bool { b := false; return &b }(),
		Limit:    1000,
	}
	return r.List(ctx, filter)
}

// GetCustomersByType retrieves customers by type
func (r *PostgresCustomerRepository) GetCustomersByType(ctx context.Context, customerType string) ([]*entities.Customer, error) {
	filter := repositories.CustomerFilter{
		Type:  customerType,
		Limit: 1000,
	}
	return r.List(ctx, filter)
}

// GetCustomersWithCreditLimit retrieves customers with credit limits
func (r *PostgresCustomerRepository) GetCustomersWithCreditLimit(ctx context.Context) ([]*entities.Customer, error) {
	query := `
		SELECT
			id, customer_code, company_id, type, first_name, last_name, email,
			phone, website, company_name, tax_id, industry, credit_limit,
			credit_used, terms, is_active, is_vat_exempt, preferred_currency,
			notes, source, created_at, updated_at
		FROM customers
		WHERE credit_limit > 0
		ORDER BY credit_limit DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get customers with credit limit: %w", err)
	}
	defer rows.Close()

	var customers []*entities.Customer
	for rows.Next() {
		customer := &entities.Customer{}
		err := rows.Scan(
			&customer.ID,
			&customer.CustomerCode,
			&customer.CompanyID,
			&customer.Type,
			&customer.FirstName,
			&customer.LastName,
			&customer.Email,
			&customer.Phone,
			&customer.Website,
			&customer.CompanyName,
			&customer.TaxID,
			&customer.Industry,
			&customer.CreditLimit,
			&customer.CreditUsed,
			&customer.Terms,
			&customer.IsActive,
			&customer.IsVATExempt,
			&customer.PreferredCurrency,
			&customer.Notes,
			&customer.Source,
			&customer.CreatedAt,
			&customer.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer credit limit row: %w", err)
		}
		customers = append(customers, customer)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating customer credit limit rows: %w", err)
	}

	return customers, nil
}

// UpdateCreditUsed updates the credit used amount for a customer
func (r *PostgresCustomerRepository) UpdateCreditUsed(ctx context.Context, customerID uuid.UUID, amount decimal.Decimal) error {
	query := `
		UPDATE customers
		SET credit_used = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, customerID, amount)
	if err != nil {
		return fmt.Errorf("failed to update customer credit used: %w", err)
	}

	return nil
}

// GetCustomersWithOverdueCredit retrieves customers with overdue credit
func (r *PostgresCustomerRepository) GetCustomersWithOverdueCredit(ctx context.Context) ([]*entities.Customer, error) {
	query := `
		SELECT DISTINCT
			c.id, c.customer_code, c.company_id, c.type, c.first_name, c.last_name,
			c.email, c.phone, c.website, c.company_name, c.tax_id, c.industry,
			c.credit_limit, c.credit_used, c.terms, c.is_active, c.is_vat_exempt,
			c.preferred_currency, c.notes, c.source, c.created_at, c.updated_at
		FROM customers c
		INNER JOIN orders o ON c.id = o.customer_id
		WHERE c.credit_used > c.credit_limit
		OR (o.payment_status IN ('PENDING', 'PARTIALLY_PAID', 'OVERDUE')
		AND o.status NOT IN ('CANCELLED', 'DELIVERED', 'REFUNDED')
		AND o.required_date < CURRENT_DATE)
		ORDER BY (c.credit_used - c.credit_limit) DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get customers with overdue credit: %w", err)
	}
	defer rows.Close()

	var customers []*entities.Customer
	for rows.Next() {
		customer := &entities.Customer{}
		err := rows.Scan(
			&customer.ID,
			&customer.CustomerCode,
			&customer.CompanyID,
			&customer.Type,
			&customer.FirstName,
			&customer.LastName,
			&customer.Email,
			&customer.Phone,
			&customer.Website,
			&customer.CompanyName,
			&customer.TaxID,
			&customer.Industry,
			&customer.CreditLimit,
			&customer.CreditUsed,
			&customer.Terms,
			&customer.IsActive,
			&customer.IsVATExempt,
			&customer.PreferredCurrency,
			&customer.Notes,
			&customer.Source,
			&customer.CreatedAt,
			&customer.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan overdue credit customer row: %w", err)
		}
		customers = append(customers, customer)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating overdue credit customer rows: %w", err)
	}

	return customers, nil
}

// GetByCompanyID retrieves customers for a specific company
func (r *PostgresCustomerRepository) GetByCompanyID(ctx context.Context, companyID uuid.UUID) ([]*entities.Customer, error) {
	filter := repositories.CustomerFilter{
		CompanyID: &companyID,
		Limit:     1000,
	}
	return r.List(ctx, filter)
}

// GetBusinessCustomers retrieves business customers
func (r *PostgresCustomerRepository) GetBusinessCustomers(ctx context.Context) ([]*entities.Customer, error) {
	query := `
		SELECT
			id, customer_code, company_id, type, first_name, last_name, email,
			phone, website, company_name, tax_id, industry, credit_limit,
			credit_used, terms, is_active, is_vat_exempt, preferred_currency,
			notes, source, created_at, updated_at
		FROM customers
		WHERE type IN ('BUSINESS', 'GOVERNMENT', 'NON_PROFIT')
		ORDER BY company_name ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get business customers: %w", err)
	}
	defer rows.Close()

	var customers []*entities.Customer
	for rows.Next() {
		customer := &entities.Customer{}
		err := rows.Scan(
			&customer.ID,
			&customer.CustomerCode,
			&customer.CompanyID,
			&customer.Type,
			&customer.FirstName,
			&customer.LastName,
			&customer.Email,
			&customer.Phone,
			&customer.Website,
			&customer.CompanyName,
			&customer.TaxID,
			&customer.Industry,
			&customer.CreditLimit,
			&customer.CreditUsed,
			&customer.Terms,
			&customer.IsActive,
			&customer.IsVATExempt,
			&customer.PreferredCurrency,
			&customer.Notes,
			&customer.Source,
			&customer.CreatedAt,
			&customer.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan business customer row: %w", err)
		}
		customers = append(customers, customer)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating business customer rows: %w", err)
	}

	return customers, nil
}

// GetCustomerStats retrieves customer statistics
func (r *PostgresCustomerRepository) GetCustomerStats(ctx context.Context, filter repositories.CustomerStatsFilter) (*repositories.CustomerStats, error) {
	query := `
		SELECT
			COUNT(*) as total_customers,
			COUNT(CASE WHEN is_active = true THEN 1 END) as active_customers,
			COUNT(CASE WHEN created_at >= $1 AND created_at <= $2 THEN 1 END) as new_customers
		FROM customers
		WHERE created_at <= $2
	`

	args := []interface{}{filter.StartDate, filter.EndDate}

	// Add additional filters
	if filter.Type != "" {
		query += " AND type = $3"
		args = append(args, filter.Type)
	}

	if filter.IsActive != nil {
		if len(args) == 2 {
			query += " AND is_active = $3"
		} else {
			query += " AND is_active = $4"
		}
		args = append(args, *filter.IsActive)
	}

	stats := &repositories.CustomerStats{}
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&stats.TotalCustomers,
		&stats.ActiveCustomers,
		&stats.NewCustomers,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer stats: %w", err)
	}

	// Get customers by type
	typeQuery := `
		SELECT type, COUNT(*)
		FROM customers
		WHERE created_at <= $2
	`

	args = []interface{}{filter.StartDate, filter.EndDate}
	argIndex := 3

	if filter.Type != "" {
		typeQuery += " AND type = $" + fmt.Sprintf("%d", argIndex)
		args = append(args, filter.Type)
		argIndex++
	}

	if filter.IsActive != nil {
		typeQuery += " AND is_active = $" + fmt.Sprintf("%d", argIndex)
		args = append(args, *filter.IsActive)
		argIndex++
	}

	typeQuery += " GROUP BY type"

	rows, err := r.db.Query(ctx, typeQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer type stats: %w", err)
	}
	defer rows.Close()

	stats.CustomersByType = make(map[string]int64)
	for rows.Next() {
		var customerType string
		var count int64
		err := rows.Scan(&customerType, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer type stat: %w", err)
		}
		stats.CustomersByType[customerType] = count
	}

	// Get customers by source
	sourceQuery := `
		SELECT source, COUNT(*)
		FROM customers
		WHERE created_at <= $2
	`

	args = []interface{}{filter.StartDate, filter.EndDate}
	argIndex = 3

	if filter.Type != "" {
		sourceQuery += " AND type = $" + fmt.Sprintf("%d", argIndex)
		args = append(args, filter.Type)
		argIndex++
	}

	if filter.IsActive != nil {
		sourceQuery += " AND is_active = $" + fmt.Sprintf("%d", argIndex)
		args = append(args, *filter.IsActive)
		argIndex++
	}

	sourceQuery += " GROUP BY source"

	rows, err = r.db.Query(ctx, sourceQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer source stats: %w", err)
	}
	defer rows.Close()

	stats.CustomersBySource = make(map[string]int64)
	for rows.Next() {
		var source string
		var count int64
		err := rows.Scan(&source, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer source stat: %w", err)
		}
		stats.CustomersBySource[source] = count
	}

	return stats, nil
}

// GetCustomerOrdersSummary retrieves a customer's order summary
func (r *PostgresCustomerRepository) GetCustomerOrdersSummary(ctx context.Context, customerID uuid.UUID) (*repositories.CustomerOrdersSummary, error) {
	query := `
		SELECT
			COUNT(*) as total_orders,
			COALESCE(SUM(total_amount), 0) as total_revenue,
			COALESCE(AVG(total_amount), 0) as average_order_value,
			MAX(order_date) as last_order_date,
			MIN(order_date) as first_order_date
		FROM orders
		WHERE customer_id = $1
	`

	summary := &repositories.CustomerOrdersSummary{
		CustomerID: customerID,
		StatusCounts: make(map[string]int64),
	}

	err := r.db.QueryRow(ctx, query, customerID).Scan(
		&summary.TotalOrders,
		&summary.TotalRevenue,
		&summary.AverageOrderValue,
		&summary.LastOrderDate,
		&summary.FirstOrderDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer orders summary: %w", err)
	}

	// Get status counts
	statusQuery := `
		SELECT status, COUNT(*)
		FROM orders
		WHERE customer_id = $1
		GROUP BY status
	`

	rows, err := r.db.Query(ctx, statusQuery, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer order status counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		err := rows.Scan(&status, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer order status count: %w", err)
		}
		summary.StatusCounts[status] = count
	}

	return summary, nil
}

// GetTopCustomersByRevenue retrieves top customers by revenue
func (r *PostgresCustomerRepository) GetTopCustomersByRevenue(ctx context.Context, startDate, endDate time.Time, limit int) ([]*repositories.CustomerRevenueStats, error) {
	query := `
		SELECT
			c.id,
			c.first_name,
			c.last_name,
			c.email,
			c.company_name,
			COALESCE(SUM(o.total_amount), 0) as total_revenue,
			COUNT(o.id) as order_count,
			COALESCE(AVG(o.total_amount), 0) as average_order_value
		FROM customers c
		INNER JOIN orders o ON c.id = o.customer_id
		WHERE o.order_date >= $1 AND o.order_date <= $2
		AND o.status NOT IN ('CANCELLED', 'REFUNDED')
		GROUP BY c.id, c.first_name, c.last_name, c.email, c.company_name
		ORDER BY total_revenue DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top customers by revenue: %w", err)
	}
	defer rows.Close()

	var customers []*repositories.CustomerRevenueStats
	for rows.Next() {
		customer := &repositories.CustomerRevenueStats{}
		err := rows.Scan(
			&customer.CustomerID,
			&customer.CustomerName,
			&customer.CustomerEmail,
			&customer.CustomerEmail,
			&customer.CompanyName,
			&customer.TotalRevenue,
			&customer.OrderCount,
			&customer.AverageOrderValue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top customer revenue row: %w", err)
		}
		customers = append(customers, customer)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating top customer revenue rows: %w", err)
	}

	return customers, nil
}

// GetNewCustomersByPeriod retrieves new customers grouped by time period
func (r *PostgresCustomerRepository) GetNewCustomersByPeriod(ctx context.Context, startDate, endDate time.Time, groupBy string) ([]*repositories.NewCustomersByPeriod, error) {
	var dateFormat string
	switch groupBy {
	case "day":
		dateFormat = "YYYY-MM-DD"
	case "week":
		dateFormat = "YYYY-\"WW\""
	case "month":
		dateFormat = "YYYY-MM"
	case "year":
		dateFormat = "YYYY"
	default:
		dateFormat = "YYYY-MM"
	}

	query := fmt.Sprintf(`
		WITH period_customers AS (
			SELECT
				TO_CHAR(created_at, '%s') as period,
				COUNT(*) as new_customers
			FROM customers
			WHERE created_at >= $1 AND created_at <= $2
			GROUP BY TO_CHAR(created_at, '%s')
		),
		cumulative_customers AS (
			SELECT
				TO_CHAR(created_at, '%s') as period,
				COUNT(*) as cumulative_count
			FROM customers
			WHERE created_at <= $2
			GROUP BY TO_CHAR(created_at, '%s')
		)
		SELECT
			pc.period,
			pc.new_customers,
			COALESCE(cc.cumulative_count, 0) as total_customers
		FROM period_customers pc
		LEFT JOIN cumulative_customers cc ON pc.period = cc.period
		ORDER BY pc.period
	`, dateFormat, dateFormat, dateFormat, dateFormat)

	rows, err := r.db.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get new customers by period: %w", err)
	}
	defer rows.Close()

	var results []*repositories.NewCustomersByPeriod
	for rows.Next() {
		result := &repositories.NewCustomersByPeriod{}
		err := rows.Scan(
			&result.Period,
			&result.NewCustomers,
			&result.TotalCustomers,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan new customers by period row: %w", err)
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating new customers by period rows: %w", err)
	}

	return results, nil
}

// BulkUpdate updates multiple customers
func (r *PostgresCustomerRepository) BulkUpdate(ctx context.Context, customers []*entities.Customer) error {
	if len(customers) == 0 {
		return nil
	}

	// For bulk operations, we'll use a transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE customers SET
			company_id = $2, type = $3, first_name = $4, last_name = $5,
			email = $6, phone = $7, website = $8, company_name = $9,
			tax_id = $10, industry = $11, credit_limit = $12, credit_used = $13,
			terms = $14, is_active = $15, is_vat_exempt = $16,
			preferred_currency = $17, notes = $18, source = $19,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	for _, customer := range customers {
		_, err := tx.Exec(ctx, query,
			customer.ID,
			customer.CompanyID,
			customer.Type,
			customer.FirstName,
			customer.LastName,
			customer.Email,
			customer.Phone,
			customer.Website,
			customer.CompanyName,
			customer.TaxID,
			customer.Industry,
			customer.CreditLimit,
			customer.CreditUsed,
			customer.Terms,
			customer.IsActive,
			customer.IsVATExempt,
			customer.PreferredCurrency,
			customer.Notes,
			customer.Source,
		)
		if err != nil {
			return fmt.Errorf("failed to bulk update customer: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BulkCreate creates multiple customers
func (r *PostgresCustomerRepository) BulkCreate(ctx context.Context, customers []*entities.Customer) error {
	if len(customers) == 0 {
		return nil
	}

	// For bulk operations, we'll use a transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO customers (
			id, customer_code, company_id, type, first_name, last_name, email,
			phone, website, company_name, tax_id, industry, credit_limit,
			credit_used, terms, is_active, is_vat_exempt, preferred_currency,
			notes, source
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20
		)
	`

	for _, customer := range customers {
		_, err := tx.Exec(ctx, query,
			customer.ID,
			customer.CustomerCode,
			customer.CompanyID,
			customer.Type,
			customer.FirstName,
			customer.LastName,
			customer.Email,
			customer.Phone,
			customer.Website,
			customer.CompanyName,
			customer.TaxID,
			customer.Industry,
			customer.CreditLimit,
			customer.CreditUsed,
			customer.Terms,
			customer.IsActive,
			customer.IsVATExempt,
			customer.PreferredCurrency,
			customer.Notes,
			customer.Source,
		)
		if err != nil {
			return fmt.Errorf("failed to bulk create customer: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}