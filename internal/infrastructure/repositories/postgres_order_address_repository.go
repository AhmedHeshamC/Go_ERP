package repositories

import (
	"context"
	"fmt"

	"erpgo/internal/domain/orders/entities"
	"erpgo/pkg/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PostgresOrderAddressRepository implements OrderAddressRepository for PostgreSQL
type PostgresOrderAddressRepository struct {
	db *database.Database
}

// NewPostgresOrderAddressRepository creates a new PostgreSQL order address repository
func NewPostgresOrderAddressRepository(db *database.Database) *PostgresOrderAddressRepository {
	return &PostgresOrderAddressRepository{
		db: db,
	}
}

// Create creates a new order address
func (r *PostgresOrderAddressRepository) Create(ctx context.Context, address *entities.OrderAddress) error {
	query := `
		INSERT INTO order_addresses (
			id, customer_id, order_id, type, first_name, last_name, company,
			address_line_1, address_line_2, city, state, postal_code, country,
			phone, email, instructions, is_default
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17
		)
	`

	_, err := r.db.Exec(ctx, query,
		address.ID,
		address.CustomerID,
		address.OrderID,
		address.Type,
		address.FirstName,
		address.LastName,
		address.Company,
		address.AddressLine1,
		address.AddressLine2,
		address.City,
		address.State,
		address.PostalCode,
		address.Country,
		address.Phone,
		address.Email,
		address.Instructions,
		address.IsDefault,
	)

	if err != nil {
		return fmt.Errorf("failed to create order address: %w", err)
	}

	return nil
}

// GetByID retrieves an order address by ID
func (r *PostgresOrderAddressRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.OrderAddress, error) {
	query := `
		SELECT
			id, customer_id, order_id, type, first_name, last_name, company,
			address_line_1, address_line_2, city, state, postal_code, country,
			phone, email, instructions, is_default, created_at, updated_at
		FROM order_addresses
		WHERE id = $1
	`

	address := &entities.OrderAddress{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&address.ID,
		&address.CustomerID,
		&address.OrderID,
		&address.Type,
		&address.FirstName,
		&address.LastName,
		&address.Company,
		&address.AddressLine1,
		&address.AddressLine2,
		&address.City,
		&address.State,
		&address.PostalCode,
		&address.Country,
		&address.Phone,
		&address.Email,
		&address.Instructions,
		&address.IsDefault,
		&address.CreatedAt,
		&address.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("order address with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get order address by id: %w", err)
	}

	return address, nil
}

// Update updates an order address
func (r *PostgresOrderAddressRepository) Update(ctx context.Context, address *entities.OrderAddress) error {
	query := `
		UPDATE order_addresses SET
			customer_id = $2, order_id = $3, type = $4, first_name = $5,
			last_name = $6, company = $7, address_line_1 = $8, address_line_2 = $9,
			city = $10, state = $11, postal_code = $12, country = $13,
			phone = $14, email = $15, instructions = $16, is_default = $17,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		address.ID,
		address.CustomerID,
		address.OrderID,
		address.Type,
		address.FirstName,
		address.LastName,
		address.Company,
		address.AddressLine1,
		address.AddressLine2,
		address.City,
		address.State,
		address.PostalCode,
		address.Country,
		address.Phone,
		address.Email,
		address.Instructions,
		address.IsDefault,
	)

	if err != nil {
		return fmt.Errorf("failed to update order address: %w", err)
	}

	return nil
}

// Delete deletes an order address
func (r *PostgresOrderAddressRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM order_addresses WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete order address: %w", err)
	}

	return nil
}

// GetByCustomerID retrieves addresses for a specific customer
func (r *PostgresOrderAddressRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*entities.OrderAddress, error) {
	query := `
		SELECT
			id, customer_id, order_id, type, first_name, last_name, company,
			address_line_1, address_line_2, city, state, postal_code, country,
			phone, email, instructions, is_default, created_at, updated_at
		FROM order_addresses
		WHERE customer_id = $1
		ORDER BY is_default DESC, created_at DESC
	`

	rows, err := r.db.Query(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order addresses by customer id: %w", err)
	}
	defer rows.Close()

	var addresses []*entities.OrderAddress
	for rows.Next() {
		address := &entities.OrderAddress{}
		err := rows.Scan(
			&address.ID,
			&address.CustomerID,
			&address.OrderID,
			&address.Type,
			&address.FirstName,
			&address.LastName,
			&address.Company,
			&address.AddressLine1,
			&address.AddressLine2,
			&address.City,
			&address.State,
			&address.PostalCode,
			&address.Country,
			&address.Phone,
			&address.Email,
			&address.Instructions,
			&address.IsDefault,
			&address.CreatedAt,
			&address.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order address row: %w", err)
		}
		addresses = append(addresses, address)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order address rows: %w", err)
	}

	return addresses, nil
}

// GetByOrderID retrieves addresses for a specific order
func (r *PostgresOrderAddressRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entities.OrderAddress, error) {
	query := `
		SELECT
			id, customer_id, order_id, type, first_name, last_name, company,
			address_line_1, address_line_2, city, state, postal_code, country,
			phone, email, instructions, is_default, created_at, updated_at
		FROM order_addresses
		WHERE order_id = $1
		ORDER BY type ASC
	`

	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order addresses by order id: %w", err)
	}
	defer rows.Close()

	var addresses []*entities.OrderAddress
	for rows.Next() {
		address := &entities.OrderAddress{}
		err := rows.Scan(
			&address.ID,
			&address.CustomerID,
			&address.OrderID,
			&address.Type,
			&address.FirstName,
			&address.LastName,
			&address.Company,
			&address.AddressLine1,
			&address.AddressLine2,
			&address.City,
			&address.State,
			&address.PostalCode,
			&address.Country,
			&address.Phone,
			&address.Email,
			&address.Instructions,
			&address.IsDefault,
			&address.CreatedAt,
			&address.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order address row: %w", err)
		}
		addresses = append(addresses, address)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order address rows: %w", err)
	}

	return addresses, nil
}

// GetByCustomerAndType retrieves addresses for a customer by type
func (r *PostgresOrderAddressRepository) GetByCustomerAndType(ctx context.Context, customerID uuid.UUID, addressType string) ([]*entities.OrderAddress, error) {
	query := `
		SELECT
			id, customer_id, order_id, type, first_name, last_name, company,
			address_line_1, address_line_2, city, state, postal_code, country,
			phone, email, instructions, is_default, created_at, updated_at
		FROM order_addresses
		WHERE customer_id = $1 AND type = $2
		ORDER BY is_default DESC, created_at DESC
	`

	rows, err := r.db.Query(ctx, query, customerID, addressType)
	if err != nil {
		return nil, fmt.Errorf("failed to get order addresses by customer and type: %w", err)
	}
	defer rows.Close()

	var addresses []*entities.OrderAddress
	for rows.Next() {
		address := &entities.OrderAddress{}
		err := rows.Scan(
			&address.ID,
			&address.CustomerID,
			&address.OrderID,
			&address.Type,
			&address.FirstName,
			&address.LastName,
			&address.Company,
			&address.AddressLine1,
			&address.AddressLine2,
			&address.City,
			&address.State,
			&address.PostalCode,
			&address.Country,
			&address.Phone,
			&address.Email,
			&address.Instructions,
			&address.IsDefault,
			&address.CreatedAt,
			&address.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order address row: %w", err)
		}
		addresses = append(addresses, address)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order address rows: %w", err)
	}

	return addresses, nil
}

// GetDefaultAddress retrieves the default address for a customer and type
func (r *PostgresOrderAddressRepository) GetDefaultAddress(ctx context.Context, customerID uuid.UUID, addressType string) (*entities.OrderAddress, error) {
	query := `
		SELECT
			id, customer_id, order_id, type, first_name, last_name, company,
			address_line_1, address_line_2, city, state, postal_code, country,
			phone, email, instructions, is_default, created_at, updated_at
		FROM order_addresses
		WHERE customer_id = $1 AND type = $2 AND is_default = true
		LIMIT 1
	`

	address := &entities.OrderAddress{}
	err := r.db.QueryRow(ctx, query, customerID, addressType).Scan(
		&address.ID,
		&address.CustomerID,
		&address.OrderID,
		&address.Type,
		&address.FirstName,
		&address.LastName,
		&address.Company,
		&address.AddressLine1,
		&address.AddressLine2,
		&address.City,
		&address.State,
		&address.PostalCode,
		&address.Country,
		&address.Phone,
		&address.Email,
		&address.Instructions,
		&address.IsDefault,
		&address.CreatedAt,
		&address.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("default address for customer %s and type %s not found", customerID, addressType)
		}
		return nil, fmt.Errorf("failed to get default address: %w", err)
	}

	return address, nil
}

// GetShippingAddresses retrieves shipping addresses for a customer
func (r *PostgresOrderAddressRepository) GetShippingAddresses(ctx context.Context, customerID uuid.UUID) ([]*entities.OrderAddress, error) {
	return r.GetByCustomerAndType(ctx, customerID, "SHIPPING")
}

// GetBillingAddresses retrieves billing addresses for a customer
func (r *PostgresOrderAddressRepository) GetBillingAddresses(ctx context.Context, customerID uuid.UUID) ([]*entities.OrderAddress, error) {
	return r.GetByCustomerAndType(ctx, customerID, "BILLING")
}

// DeleteByCustomerID deletes all addresses for a customer
func (r *PostgresOrderAddressRepository) DeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error {
	query := `DELETE FROM order_addresses WHERE customer_id = $1`

	_, err := r.db.Exec(ctx, query, customerID)
	if err != nil {
		return fmt.Errorf("failed to delete order addresses by customer id: %w", err)
	}

	return nil
}

// DeleteByOrderID deletes all addresses for an order
func (r *PostgresOrderAddressRepository) DeleteByOrderID(ctx context.Context, orderID uuid.UUID) error {
	query := `DELETE FROM order_addresses WHERE order_id = $1`

	_, err := r.db.Exec(ctx, query, orderID)
	if err != nil {
		return fmt.Errorf("failed to delete order addresses by order id: %w", err)
	}

	return nil
}
