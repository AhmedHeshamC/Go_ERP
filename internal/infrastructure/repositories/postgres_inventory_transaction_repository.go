package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"erpgo/internal/domain/inventory/entities"
	"erpgo/internal/domain/inventory/repositories"
	"erpgo/pkg/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PostgresInventoryTransactionRepository implements InventoryTransactionRepository for PostgreSQL
type PostgresInventoryTransactionRepository struct {
	db     *database.Database
	logger interface{} // Can be zerolog.Logger or any logger
}

// NewPostgresInventoryTransactionRepository creates a new PostgreSQL inventory transaction repository
func NewPostgresInventoryTransactionRepository(db *database.Database) *PostgresInventoryTransactionRepository {
	return &PostgresInventoryTransactionRepository{
		db: db,
	}
}

// Create creates a new inventory transaction
func (r *PostgresInventoryTransactionRepository) Create(ctx context.Context, transaction *entities.InventoryTransaction) error {
	query := `
		INSERT INTO inventory_transactions (id, product_id, warehouse_id, transaction_type, quantity,
		                                   reference_type, reference_id, reason, unit_cost, total_cost,
		                                   batch_number, expiry_date, serial_number, from_warehouse_id,
		                                   to_warehouse_id, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err := r.db.Exec(ctx, query,
		transaction.ID,
		transaction.ProductID,
		transaction.WarehouseID,
		transaction.TransactionType,
		transaction.Quantity,
		transaction.ReferenceType,
		transaction.ReferenceID,
		transaction.Reason,
		transaction.UnitCost,
		transaction.TotalCost,
		transaction.BatchNumber,
		transaction.ExpiryDate,
		transaction.SerialNumber,
		transaction.FromWarehouseID,
		transaction.ToWarehouseID,
		transaction.CreatedAt,
		transaction.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create inventory transaction: %w", err)
	}

	return nil
}

// GetByID retrieves an inventory transaction by ID
func (r *PostgresInventoryTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.InventoryTransaction, error) {
	query := `
		SELECT id, product_id, warehouse_id, transaction_type, quantity,
		       reference_type, reference_id, reason, unit_cost, total_cost,
		       batch_number, expiry_date, serial_number, from_warehouse_id,
		       to_warehouse_id, created_at, created_by, approved_at, approved_by
		FROM inventory_transactions
		WHERE id = $1
	`

	transaction := &entities.InventoryTransaction{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&transaction.ID,
		&transaction.ProductID,
		&transaction.WarehouseID,
		&transaction.TransactionType,
		&transaction.Quantity,
		&transaction.ReferenceType,
		&transaction.ReferenceID,
		&transaction.Reason,
		&transaction.UnitCost,
		&transaction.TotalCost,
		&transaction.BatchNumber,
		&transaction.ExpiryDate,
		&transaction.SerialNumber,
		&transaction.FromWarehouseID,
		&transaction.ToWarehouseID,
		&transaction.CreatedAt,
		&transaction.CreatedBy,
		&transaction.ApprovedAt,
		&transaction.ApprovedBy,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("inventory transaction with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to get inventory transaction: %w", err)
	}

	return transaction, nil
}

// Update updates an inventory transaction
func (r *PostgresInventoryTransactionRepository) Update(ctx context.Context, transaction *entities.InventoryTransaction) error {
	query := `
		UPDATE inventory_transactions
		SET product_id = $2, warehouse_id = $3, transaction_type = $4, quantity = $5,
		    reference_type = $6, reference_id = $7, reason = $8, unit_cost = $9,
		    total_cost = $10, batch_number = $11, expiry_date = $12, serial_number = $13,
		    from_warehouse_id = $14, to_warehouse_id = $15, approved_at = $16, approved_by = $17
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		transaction.ID,
		transaction.ProductID,
		transaction.WarehouseID,
		transaction.TransactionType,
		transaction.Quantity,
		transaction.ReferenceType,
		transaction.ReferenceID,
		transaction.Reason,
		transaction.UnitCost,
		transaction.TotalCost,
		transaction.BatchNumber,
		transaction.ExpiryDate,
		transaction.SerialNumber,
		transaction.FromWarehouseID,
		transaction.ToWarehouseID,
		transaction.ApprovedAt,
		transaction.ApprovedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to update inventory transaction: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("inventory transaction with ID %s not found", transaction.ID)
	}

	return nil
}

// Delete deletes an inventory transaction
func (r *PostgresInventoryTransactionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM inventory_transactions WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete inventory transaction: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("inventory transaction with ID %s not found", id)
	}

	return nil
}

// GetByProduct retrieves inventory transactions for a product
func (r *PostgresInventoryTransactionRepository) GetByProduct(ctx context.Context, productID uuid.UUID, filter *repositories.TransactionFilter) ([]*entities.InventoryTransaction, error) {
	query := `
		SELECT id, product_id, warehouse_id, transaction_type, quantity,
		       reference_type, reference_id, reason, unit_cost, total_cost,
		       batch_number, expiry_date, serial_number, from_warehouse_id,
		       to_warehouse_id, created_at, created_by, approved_at, approved_by
		FROM inventory_transactions
		WHERE product_id = $1
	`

	args := []interface{}{productID}
	argIndex := 2

	// Add filters
	query, args, argIndex = r.addTransactionFilters(query, args, argIndex, filter)

	// Add ordering
	if filter != nil && filter.OrderBy != "" {
		order := "ASC"
		if filter.Order != "" {
			order = strings.ToUpper(filter.Order)
		}
		query += fmt.Sprintf(" ORDER BY %s %s", filter.OrderBy, order)
	} else {
		query += " ORDER BY created_at DESC"
	}

	// Add pagination
	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory transactions by product: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// GetByWarehouse retrieves inventory transactions for a warehouse
func (r *PostgresInventoryTransactionRepository) GetByWarehouse(ctx context.Context, warehouseID uuid.UUID, filter *repositories.TransactionFilter) ([]*entities.InventoryTransaction, error) {
	query := `
		SELECT id, product_id, warehouse_id, transaction_type, quantity,
		       reference_type, reference_id, reason, unit_cost, total_cost,
		       batch_number, expiry_date, serial_number, from_warehouse_id,
		       to_warehouse_id, created_at, created_by, approved_at, approved_by
		FROM inventory_transactions
		WHERE warehouse_id = $1
	`

	args := []interface{}{warehouseID}
	argIndex := 2

	// Add filters
	query, args, argIndex = r.addTransactionFilters(query, args, argIndex, filter)

	// Add ordering
	if filter != nil && filter.OrderBy != "" {
		order := "ASC"
		if filter.Order != "" {
			order = strings.ToUpper(filter.Order)
		}
		query += fmt.Sprintf(" ORDER BY %s %s", filter.OrderBy, order)
	} else {
		query += " ORDER BY created_at DESC"
	}

	// Add pagination
	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory transactions by warehouse: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// GetByProductAndWarehouse retrieves inventory transactions for a product in a warehouse
func (r *PostgresInventoryTransactionRepository) GetByProductAndWarehouse(ctx context.Context, productID, warehouseID uuid.UUID, filter *repositories.TransactionFilter) ([]*entities.InventoryTransaction, error) {
	query := `
		SELECT id, product_id, warehouse_id, transaction_type, quantity,
		       reference_type, reference_id, reason, unit_cost, total_cost,
		       batch_number, expiry_date, serial_number, from_warehouse_id,
		       to_warehouse_id, created_at, created_by, approved_at, approved_by
		FROM inventory_transactions
		WHERE product_id = $1 AND warehouse_id = $2
	`

	args := []interface{}{productID, warehouseID}
	argIndex := 3

	// Add filters
	query, args, argIndex = r.addTransactionFilters(query, args, argIndex, filter)

	// Add ordering
	if filter != nil && filter.OrderBy != "" {
		order := "ASC"
		if filter.Order != "" {
			order = strings.ToUpper(filter.Order)
		}
		query += fmt.Sprintf(" ORDER BY %s %s", filter.OrderBy, order)
	} else {
		query += " ORDER BY created_at DESC"
	}

	// Add pagination
	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory transactions by product and warehouse: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// GetByType retrieves inventory transactions by type
func (r *PostgresInventoryTransactionRepository) GetByType(ctx context.Context, transactionType entities.TransactionType, filter *repositories.TransactionFilter) ([]*entities.InventoryTransaction, error) {
	query := `
		SELECT id, product_id, warehouse_id, transaction_type, quantity,
		       reference_type, reference_id, reason, unit_cost, total_cost,
		       batch_number, expiry_date, serial_number, from_warehouse_id,
		       to_warehouse_id, created_at, created_by, approved_at, approved_by
		FROM inventory_transactions
		WHERE transaction_type = $1
	`

	args := []interface{}{transactionType}
	argIndex := 2

	// Add filters
	query, args, argIndex = r.addTransactionFilters(query, args, argIndex, filter)

	// Add ordering
	if filter != nil && filter.OrderBy != "" {
		order := "ASC"
		if filter.Order != "" {
			order = strings.ToUpper(filter.Order)
		}
		query += fmt.Sprintf(" ORDER BY %s %s", filter.OrderBy, order)
	} else {
		query += " ORDER BY created_at DESC"
	}

	// Add pagination
	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory transactions by type: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// GetByReference retrieves inventory transactions by reference
func (r *PostgresInventoryTransactionRepository) GetByReference(ctx context.Context, referenceType string, referenceID uuid.UUID) ([]*entities.InventoryTransaction, error) {
	query := `
		SELECT id, product_id, warehouse_id, transaction_type, quantity,
		       reference_type, reference_id, reason, unit_cost, total_cost,
		       batch_number, expiry_date, serial_number, from_warehouse_id,
		       to_warehouse_id, created_at, created_by, approved_at, approved_by
		FROM inventory_transactions
		WHERE reference_type = $1 AND reference_id = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, referenceType, referenceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory transactions by reference: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// GetByBatch retrieves inventory transactions by batch number
func (r *PostgresInventoryTransactionRepository) GetByBatch(ctx context.Context, batchNumber string) ([]*entities.InventoryTransaction, error) {
	query := `
		SELECT id, product_id, warehouse_id, transaction_type, quantity,
		       reference_type, reference_id, reason, unit_cost, total_cost,
		       batch_number, expiry_date, serial_number, from_warehouse_id,
		       to_warehouse_id, created_at, created_by, approved_at, approved_by
		FROM inventory_transactions
		WHERE batch_number = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, batchNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory transactions by batch: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// GetByDateRange retrieves inventory transactions within a date range
func (r *PostgresInventoryTransactionRepository) GetByDateRange(ctx context.Context, warehouseID *uuid.UUID, startDate, endDate time.Time, filter *repositories.TransactionFilter) ([]*entities.InventoryTransaction, error) {
	query := `
		SELECT id, product_id, warehouse_id, transaction_type, quantity,
		       reference_type, reference_id, reason, unit_cost, total_cost,
		       batch_number, expiry_date, serial_number, from_warehouse_id,
		       to_warehouse_id, created_at, created_by, approved_at, approved_by
		FROM inventory_transactions
		WHERE created_at BETWEEN $1 AND $2
	`

	args := []interface{}{startDate, endDate}
	argIndex := 3

	if warehouseID != nil {
		query += fmt.Sprintf(" AND warehouse_id = $%d", argIndex)
		args = append(args, *warehouseID)
		argIndex++
	}

	// Add additional filters
	query, args, argIndex = r.addTransactionFilters(query, args, argIndex, filter)

	// Add ordering
	query += " ORDER BY created_at DESC"

	// Add pagination
	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory transactions by date range: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// GetRecentTransactions retrieves recent inventory transactions
func (r *PostgresInventoryTransactionRepository) GetRecentTransactions(ctx context.Context, warehouseID *uuid.UUID, hours int, limit int) ([]*entities.InventoryTransaction, error) {
	query := `
		SELECT id, product_id, warehouse_id, transaction_type, quantity,
		       reference_type, reference_id, reason, unit_cost, total_cost,
		       batch_number, expiry_date, serial_number, from_warehouse_id,
		       to_warehouse_id, created_at, created_by, approved_at, approved_by
		FROM inventory_transactions
		WHERE created_at >= NOW() - INTERVAL '%d hours'
	`

	args := []interface{}{hours}
	argIndex := 1

	if warehouseID != nil {
		query += fmt.Sprintf(" AND warehouse_id = $%d", argIndex)
		args = append(args, *warehouseID)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent inventory transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// Search searches inventory transactions
func (r *PostgresInventoryTransactionRepository) Search(ctx context.Context, query string, limit int) ([]*entities.InventoryTransaction, error) {
	sqlQuery := `
		SELECT it.id, it.product_id, it.warehouse_id, it.transaction_type, it.quantity,
		       it.reference_type, it.reference_id, it.reason, it.unit_cost, it.total_cost,
		       it.batch_number, it.expiry_date, it.serial_number, it.from_warehouse_id,
		       it.to_warehouse_id, it.created_at, it.created_by, it.approved_at, it.approved_by
		FROM inventory_transactions it
		JOIN products p ON it.product_id = p.id
		JOIN warehouses w ON it.warehouse_id = w.id
		WHERE p.name ILIKE $1 OR p.sku ILIKE $1 OR w.name ILIKE $1 OR w.code ILIKE $1
		   OR it.reason ILIKE $1 OR it.batch_number ILIKE $1 OR it.serial_number ILIKE $1
		ORDER BY it.created_at DESC
	`

	args := []interface{}{"%" + query + "%"}

	if limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", 2)
		args = append(args, limit)
	}

	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search inventory transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// Count counts inventory transactions with filtering
func (r *PostgresInventoryTransactionRepository) Count(ctx context.Context, filter *repositories.TransactionFilter) (int, error) {
	query := `SELECT COUNT(*) FROM inventory_transactions WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	// Add filters
	query, args, argIndex = r.addTransactionFilters(query, args, argIndex, filter)

	var count int
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count inventory transactions: %w", err)
	}

	return count, nil
}

// GetPendingApproval retrieves transactions pending approval
func (r *PostgresInventoryTransactionRepository) GetPendingApproval(ctx context.Context, warehouseID *uuid.UUID) ([]*entities.InventoryTransaction, error) {
	query := `
		SELECT id, product_id, warehouse_id, transaction_type, quantity,
		       reference_type, reference_id, reason, unit_cost, total_cost,
		       batch_number, expiry_date, serial_number, from_warehouse_id,
		       to_warehouse_id, created_at, created_by, approved_at, approved_by
		FROM inventory_transactions
		WHERE approved_at IS NULL
	`

	args := []interface{}{}
	argIndex := 1

	if warehouseID != nil {
		query += fmt.Sprintf(" AND warehouse_id = $%d", argIndex)
		args = append(args, *warehouseID)
		argIndex++
	}

	query += " ORDER BY created_at ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending approval transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// ApproveTransaction approves a transaction
func (r *PostgresInventoryTransactionRepository) ApproveTransaction(ctx context.Context, transactionID uuid.UUID, approvedBy uuid.UUID) error {
	query := `
		UPDATE inventory_transactions
		SET approved_at = NOW(), approved_by = $2
		WHERE id = $1 AND approved_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, transactionID, approvedBy)
	if err != nil {
		return fmt.Errorf("failed to approve transaction: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("transaction with ID %s not found or already approved", transactionID)
	}

	return nil
}

// RejectTransaction rejects a transaction
func (r *PostgresInventoryTransactionRepository) RejectTransaction(ctx context.Context, transactionID uuid.UUID, rejectedBy uuid.UUID, reason string) error {
	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update transaction with rejection info
	updateQuery := `
		UPDATE inventory_transactions
		SET approved_at = NOW(), approved_by = $2, reason = $3
		WHERE id = $1 AND approved_at IS NULL
	`

	result, err := tx.Exec(ctx, updateQuery, transactionID, rejectedBy, reason)
	if err != nil {
		return fmt.Errorf("failed to reject transaction: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("transaction with ID %s not found or already processed", transactionID)
	}

	// Create a rejection record (would need transaction_rejections table)
	// TODO: Implement when transaction_rejections table is created

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit rejection transaction: %w", err)
	}

	return nil
}

// GetTransferTransactions retrieves transfer transactions
func (r *PostgresInventoryTransactionRepository) GetTransferTransactions(ctx context.Context, fromWarehouseID, toWarehouseID uuid.UUID) ([]*entities.InventoryTransaction, error) {
	query := `
		SELECT id, product_id, warehouse_id, transaction_type, quantity,
		       reference_type, reference_id, reason, unit_cost, total_cost,
		       batch_number, expiry_date, serial_number, from_warehouse_id,
		       to_warehouse_id, created_at, created_by, approved_at, approved_by
		FROM inventory_transactions
		WHERE (transaction_type = 'TRANSFER_OUT' AND warehouse_id = $1 AND to_warehouse_id = $2)
		   OR (transaction_type = 'TRANSFER_IN' AND warehouse_id = $2 AND from_warehouse_id = $1)
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, fromWarehouseID, toWarehouseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transfer transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// GetPendingTransfers retrieves pending transfer transactions
func (r *PostgresInventoryTransactionRepository) GetPendingTransfers(ctx context.Context, warehouseID *uuid.UUID) ([]*entities.InventoryTransaction, error) {
	query := `
		SELECT id, product_id, warehouse_id, transaction_type, quantity,
		       reference_type, reference_id, reason, unit_cost, total_cost,
		       batch_number, expiry_date, serial_number, from_warehouse_id,
		       to_warehouse_id, created_at, created_by, approved_at, approved_by
		FROM inventory_transactions
		WHERE transaction_type IN ('TRANSFER_OUT', 'TRANSFER_IN')
		  AND approved_at IS NULL
	`

	args := []interface{}{}
	argIndex := 1

	if warehouseID != nil {
		query += fmt.Sprintf(" AND (warehouse_id = $%d OR from_warehouse_id = $%d OR to_warehouse_id = $%d)", argIndex, argIndex, argIndex)
		args = append(args, *warehouseID)
		argIndex++
	}

	query += " ORDER BY created_at ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending transfers: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// GetTransactionSummary retrieves transaction summary
func (r *PostgresInventoryTransactionRepository) GetTransactionSummary(ctx context.Context, filter *repositories.TransactionFilter) (*repositories.TransactionSummary, error) {
	// Base query for overall statistics
	baseQuery := `
		SELECT
			COUNT(*) as total_transactions,
			COALESCE(SUM(CASE WHEN quantity > 0 THEN quantity ELSE 0 END), 0) as total_quantity_in,
			COALESCE(SUM(CASE WHEN quantity < 0 THEN ABS(quantity) ELSE 0 END), 0) as total_quantity_out,
			COALESCE(SUM(CASE WHEN quantity > 0 THEN total_cost ELSE 0 END), 0) as total_value_in,
			COALESCE(SUM(CASE WHEN quantity < 0 THEN total_cost ELSE 0 END), 0) as total_value_out
		FROM inventory_transactions
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	// Add filters
	baseQuery, args, argIndex = r.addTransactionFilters(baseQuery, args, argIndex, filter)

	summary := &repositories.TransactionSummary{
		TransactionsByType: make(map[entities.TransactionType]*repositories.TransactionTypeSummary),
		TopProducts:        []repositories.ProductTransactionSummary{},
		TopWarehouses:      []repositories.WarehouseTransactionSummary{},
	}

	// Execute base query
	err := r.db.QueryRow(ctx, baseQuery, args...).Scan(
		&summary.TotalTransactions,
		&summary.TotalQuantityIn,
		&summary.TotalQuantityOut,
		&summary.TotalValueIn,
		&summary.TotalValueOut,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction summary: %w", err)
	}

	// Get transactions by type
	typeQuery := `
		SELECT transaction_type, COUNT(*), SUM(ABS(quantity)), COALESCE(SUM(total_cost), 0)
		FROM inventory_transactions
		WHERE 1=1
	`

	typeArgs := []interface{}{}
	typeArgIndex := 1
	typeQuery, typeArgs, typeArgIndex = r.addTransactionFilters(typeQuery, typeArgs, typeArgIndex, filter)
	typeQuery += " GROUP BY transaction_type"

	typeRows, err := r.db.Query(ctx, typeQuery, typeArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction summary by type: %w", err)
	}
	defer typeRows.Close()

	for typeRows.Next() {
		var transactionType entities.TransactionType
		var typeSummary repositories.TransactionTypeSummary
		err := typeRows.Scan(&transactionType, &typeSummary.Count, &typeSummary.TotalQuantity, &typeSummary.TotalValue)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction type summary: %w", err)
		}
		typeSummary.TransactionType = transactionType
		summary.TransactionsByType[transactionType] = &typeSummary
	}

	// Set date range if specified
	if filter != nil && filter.DateFrom != nil && filter.DateTo != nil {
		summary.DateRange = repositories.DateRange{
			StartDate: *filter.DateFrom,
			EndDate:   *filter.DateTo,
		}
	} else {
		// Default to last 30 days
		now := time.Now()
		summary.DateRange = repositories.DateRange{
			StartDate: now.AddDate(0, 0, -30),
			EndDate:   now,
		}
	}

	return summary, nil
}

// GetTransactionHistory retrieves transaction history for a product
func (r *PostgresInventoryTransactionRepository) GetTransactionHistory(ctx context.Context, productID uuid.UUID, warehouseID *uuid.UUID, limit int) ([]*entities.InventoryTransaction, error) {
	query := `
		SELECT id, product_id, warehouse_id, transaction_type, quantity,
		       reference_type, reference_id, reason, unit_cost, total_cost,
		       batch_number, expiry_date, serial_number, from_warehouse_id,
		       to_warehouse_id, created_at, created_by, approved_at, approved_by
		FROM inventory_transactions
		WHERE product_id = $1
	`

	args := []interface{}{productID}
	argIndex := 2

	if warehouseID != nil {
		query += fmt.Sprintf(" AND warehouse_id = $%d", argIndex)
		args = append(args, *warehouseID)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction history: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// GetCostOfGoodsSold calculates cost of goods sold for a period
func (r *PostgresInventoryTransactionRepository) GetCostOfGoodsSold(ctx context.Context, startDate, endDate time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(total_cost), 0)
		FROM inventory_transactions
		WHERE transaction_type IN ('SALE', 'CONSUMPTION')
		  AND created_at BETWEEN $1 AND $2
		  AND approved_at IS NOT NULL
	`

	var cogs float64
	err := r.db.QueryRow(ctx, query, startDate, endDate).Scan(&cogs)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate cost of goods sold: %w", err)
	}

	return cogs, nil
}

// GetInventoryMovement retrieves inventory movement data
func (r *PostgresInventoryTransactionRepository) GetInventoryMovement(ctx context.Context, filter *repositories.MovementFilter) ([]*repositories.InventoryMovement, error) {
	// This is a complex query that would need to be implemented based on specific requirements
	// For now, returning a simplified implementation
	query := `
		SELECT
			CASE
				WHEN $1 = 'product' THEN p.name::text
				WHEN $1 = 'warehouse' THEN w.name::text
				WHEN $1 = 'type' THEN it.transaction_type::text
				ELSE DATE_TRUNC('day', it.created_at)::text
			END as group_by_value,
			DATE_TRUNC('day', it.created_at) as date,
			COALESCE(SUM(CASE WHEN it.quantity > 0 THEN it.quantity ELSE 0 END), 0) as quantity_in,
			COALESCE(SUM(CASE WHEN it.quantity < 0 THEN ABS(it.quantity) ELSE 0 END), 0) as quantity_out,
			COALESCE(SUM(it.quantity), 0) as net_movement,
			COALESCE(SUM(CASE WHEN it.quantity > 0 THEN it.total_cost ELSE 0 END), 0) as value_in,
			COALESCE(SUM(CASE WHEN it.quantity < 0 THEN it.total_cost ELSE 0 END), 0) as value_out,
			COALESCE(SUM(it.total_cost), 0) as net_value
		FROM inventory_transactions it
		JOIN products p ON it.product_id = p.id
		JOIN warehouses w ON it.warehouse_id = w.id
		WHERE it.created_at BETWEEN $2 AND $3
	`

	args := []interface{}{filter.GroupBy, filter.StartDate, filter.EndDate}
	argIndex := 4

	// Add additional filters
	if len(filter.ProductIDs) > 0 {
		placeholders := make([]string, len(filter.ProductIDs))
		for i, id := range filter.ProductIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		query += fmt.Sprintf(" AND it.product_id IN (%s)", strings.Join(placeholders, ","))
	}

	if len(filter.WarehouseIDs) > 0 {
		placeholders := make([]string, len(filter.WarehouseIDs))
		for i, id := range filter.WarehouseIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		query += fmt.Sprintf(" AND it.warehouse_id IN (%s)", strings.Join(placeholders, ","))
	}

	query += " GROUP BY group_by_value, date ORDER BY date, group_by_value"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory movement: %w", err)
	}
	defer rows.Close()

	var movements []*repositories.InventoryMovement
	for rows.Next() {
		movement := &repositories.InventoryMovement{}
		err := rows.Scan(
			&movement.GroupByValue,
			&movement.Date,
			&movement.QuantityIn,
			&movement.QuantityOut,
			&movement.NetMovement,
			&movement.ValueIn,
			&movement.ValueOut,
			&movement.NetValue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory movement row: %w", err)
		}
		movements = append(movements, movement)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory movement rows: %w", err)
	}

	return movements, nil
}

// BulkCreate creates multiple inventory transactions
func (r *PostgresInventoryTransactionRepository) BulkCreate(ctx context.Context, transactions []*entities.InventoryTransaction) error {
	if len(transactions) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO inventory_transactions (id, product_id, warehouse_id, transaction_type, quantity,
		                                   reference_type, reference_id, reason, unit_cost, total_cost,
		                                   batch_number, expiry_date, serial_number, from_warehouse_id,
		                                   to_warehouse_id, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	for _, transaction := range transactions {
		_, err = tx.Exec(ctx, query,
			transaction.ID,
			transaction.ProductID,
			transaction.WarehouseID,
			transaction.TransactionType,
			transaction.Quantity,
			transaction.ReferenceType,
			transaction.ReferenceID,
			transaction.Reason,
			transaction.UnitCost,
			transaction.TotalCost,
			transaction.BatchNumber,
			transaction.ExpiryDate,
			transaction.SerialNumber,
			transaction.FromWarehouseID,
			transaction.ToWarehouseID,
			transaction.CreatedAt,
			transaction.CreatedBy,
		)
		if err != nil {
			return fmt.Errorf("failed to create inventory transaction: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BulkApprove approves multiple transactions
func (r *PostgresInventoryTransactionRepository) BulkApprove(ctx context.Context, transactionIDs []uuid.UUID, approvedBy uuid.UUID) error {
	if len(transactionIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(transactionIDs))
	args := make([]interface{}, len(transactionIDs)+1)
	args[0] = approvedBy

	for i, id := range transactionIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		UPDATE inventory_transactions
		SET approved_at = NOW(), approved_by = $1
		WHERE id IN (%s) AND approved_at IS NULL
	`, strings.Join(placeholders, ","))

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk approve transactions: %w", err)
	}

	return nil
}

// GetAuditTrail retrieves audit trail
func (r *PostgresInventoryTransactionRepository) GetAuditTrail(ctx context.Context, filter *repositories.AuditFilter) ([]*entities.InventoryTransaction, error) {
	query := `
		SELECT it.id, it.product_id, it.warehouse_id, it.transaction_type, it.quantity,
		       it.reference_type, it.reference_id, it.reason, it.unit_cost, it.total_cost,
		       it.batch_number, it.expiry_date, it.serial_number, it.from_warehouse_id,
		       it.to_warehouse_id, it.created_at, it.created_by, it.approved_at, it.approved_by
		FROM inventory_transactions it
		WHERE it.created_at BETWEEN $1 AND $2
	`

	args := []interface{}{filter.StartDate, filter.EndDate}
	argIndex := 3

	// Add user filters
	if len(filter.UserIDs) > 0 {
		placeholders := make([]string, len(filter.UserIDs))
		for i, id := range filter.UserIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		query += fmt.Sprintf(" AND it.created_by IN (%s)", strings.Join(placeholders, ","))
	}

	// Add product filters
	if len(filter.ProductIDs) > 0 {
		placeholders := make([]string, len(filter.ProductIDs))
		for i, id := range filter.ProductIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		query += fmt.Sprintf(" AND it.product_id IN (%s)", strings.Join(placeholders, ","))
	}

	// Add warehouse filters
	if len(filter.WarehouseIDs) > 0 {
		placeholders := make([]string, len(filter.WarehouseIDs))
		for i, id := range filter.WarehouseIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		query += fmt.Sprintf(" AND it.warehouse_id IN (%s)", strings.Join(placeholders, ","))
	}

	// Add transaction type filters
	if len(filter.TransactionTypes) > 0 {
		placeholders := make([]string, len(filter.TransactionTypes))
		for i, ttype := range filter.TransactionTypes {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, ttype)
			argIndex++
		}
		query += fmt.Sprintf(" AND it.transaction_type IN (%s)", strings.Join(placeholders, ","))
	}

	// Add status filters
	if filter.IncludeApproved != nil && filter.IncludePending != nil {
		if *filter.IncludeApproved && *filter.IncludePending {
			// Include both - no additional filter needed
		} else if *filter.IncludeApproved {
			query += " AND it.approved_at IS NOT NULL"
		} else if *filter.IncludePending {
			query += " AND it.approved_at IS NULL"
		}
	}

	query += " ORDER BY it.created_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit trail: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.InventoryTransaction
	for rows.Next() {
		transaction := &entities.InventoryTransaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.ProductID,
			&transaction.WarehouseID,
			&transaction.TransactionType,
			&transaction.Quantity,
			&transaction.ReferenceType,
			&transaction.ReferenceID,
			&transaction.Reason,
			&transaction.UnitCost,
			&transaction.TotalCost,
			&transaction.BatchNumber,
			&transaction.ExpiryDate,
			&transaction.SerialNumber,
			&transaction.FromWarehouseID,
			&transaction.ToWarehouseID,
			&transaction.CreatedAt,
			&transaction.CreatedBy,
			&transaction.ApprovedAt,
			&transaction.ApprovedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	return transactions, nil
}

// GetComplianceReport generates a compliance report
func (r *PostgresInventoryTransactionRepository) GetComplianceReport(ctx context.Context, startDate, endDate time.Time) (*repositories.ComplianceReport, error) {
	report := &repositories.ComplianceReport{
		Period: repositories.DateRange{
			StartDate: startDate,
			EndDate:   endDate,
		},
		TransactionsByType: make(map[entities.TransactionType]int),
	}

	// Get overall statistics
	statsQuery := `
		SELECT
			COUNT(*) as total_transactions,
			COUNT(CASE WHEN approved_at IS NOT NULL THEN 1 END) as approved_transactions,
			COUNT(CASE WHEN approved_at IS NULL THEN 1 END) as pending_transactions,
			COUNT(CASE WHEN total_cost > 10000 THEN 1 END) as high_value_transactions
		FROM inventory_transactions
		WHERE created_at BETWEEN $1 AND $2
	`

	err := r.db.QueryRow(ctx, statsQuery, startDate, endDate).Scan(
		&report.TotalTransactions,
		&report.ApprovedTransactions,
		&report.PendingTransactions,
		&report.HighValueTransactions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance statistics: %w", err)
	}

	// Get transactions by type
	typeQuery := `
		SELECT transaction_type, COUNT(*)
		FROM inventory_transactions
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY transaction_type
	`

	typeRows, err := r.db.Query(ctx, typeQuery, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance data by type: %w", err)
	}
	defer typeRows.Close()

	for typeRows.Next() {
		var transactionType entities.TransactionType
		var count int
		err := typeRows.Scan(&transactionType, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan compliance type data: %w", err)
		}
		report.TransactionsByType[transactionType] = count
	}

	// Calculate compliance score
	if report.TotalTransactions > 0 {
		report.ComplianceScore = float64(report.ApprovedTransactions) / float64(report.TotalTransactions) * 100
	}

	// Add recommendations based on compliance score
	if report.ComplianceScore < 80 {
		report.Recommendations = append(report.Recommendations, "Consider implementing automatic approval workflows")
	}
	if report.PendingTransactions > 100 {
		report.Recommendations = append(report.Recommendations, "High number of pending transactions - review approval process")
	}
	if report.HighValueTransactions > 50 {
		report.Recommendations = append(report.Recommendations, "Multiple high-value transactions - additional review recommended")
	}

	return report, nil
}

// Helper method to add transaction filters to queries
func (r *PostgresInventoryTransactionRepository) addTransactionFilters(query string, args []interface{}, argIndex int, filter *repositories.TransactionFilter) (string, []interface{}, int) {
	if filter == nil {
		return query, args, argIndex
	}

	if len(filter.IDs) > 0 {
		placeholders := make([]string, len(filter.IDs))
		for i, id := range filter.IDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		query += fmt.Sprintf(" AND id IN (%s)", strings.Join(placeholders, ","))
	}

	if len(filter.ProductIDs) > 0 {
		placeholders := make([]string, len(filter.ProductIDs))
		for i, id := range filter.ProductIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		query += fmt.Sprintf(" AND product_id IN (%s)", strings.Join(placeholders, ","))
	}

	if len(filter.WarehouseIDs) > 0 {
		placeholders := make([]string, len(filter.WarehouseIDs))
		for i, id := range filter.WarehouseIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		query += fmt.Sprintf(" AND warehouse_id IN (%s)", strings.Join(placeholders, ","))
	}

	if len(filter.TransactionTypes) > 0 {
		placeholders := make([]string, len(filter.TransactionTypes))
		for i, ttype := range filter.TransactionTypes {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, ttype)
			argIndex++
		}
		query += fmt.Sprintf(" AND transaction_type IN (%s)", strings.Join(placeholders, ","))
	}

	if filter.ReferenceType != "" {
		query += fmt.Sprintf(" AND reference_type = $%d", argIndex)
		args = append(args, filter.ReferenceType)
		argIndex++
	}

	if filter.ReferenceID != nil {
		query += fmt.Sprintf(" AND reference_id = $%d", argIndex)
		args = append(args, *filter.ReferenceID)
		argIndex++
	}

	if len(filter.CreatedBy) > 0 {
		placeholders := make([]string, len(filter.CreatedBy))
		for i, id := range filter.CreatedBy {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		query += fmt.Sprintf(" AND created_by IN (%s)", strings.Join(placeholders, ","))
	}

	if len(filter.ApprovedBy) > 0 {
		placeholders := make([]string, len(filter.ApprovedBy))
		for i, id := range filter.ApprovedBy {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		query += fmt.Sprintf(" AND approved_by IN (%s)", strings.Join(placeholders, ","))
	}

	if filter.BatchNumber != "" {
		query += fmt.Sprintf(" AND batch_number ILIKE $%d", argIndex)
		args = append(args, "%"+filter.BatchNumber+"%")
		argIndex++
	}

	if filter.SerialNumber != "" {
		query += fmt.Sprintf(" AND serial_number ILIKE $%d", argIndex)
		args = append(args, "%"+filter.SerialNumber+"%")
		argIndex++
	}

	if filter.IsApproved != nil {
		if *filter.IsApproved {
			query += fmt.Sprintf(" AND approved_at IS NOT NULL")
		} else {
			query += fmt.Sprintf(" AND approved_at IS NULL")
		}
	}

	if filter.IsPending != nil {
		if *filter.IsPending {
			query += fmt.Sprintf(" AND approved_at IS NULL")
		} else {
			query += fmt.Sprintf(" AND approved_at IS NOT NULL")
		}
	}

	if filter.DateFrom != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filter.DateFrom)
		argIndex++
	}

	if filter.DateTo != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filter.DateTo)
		argIndex++
	}

	if filter.CreatedAfter != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filter.CreatedAfter)
		argIndex++
	}

	if filter.CreatedBefore != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filter.CreatedBefore)
		argIndex++
	}

	if filter.ApprovedAfter != nil {
		query += fmt.Sprintf(" AND approved_at >= $%d", argIndex)
		args = append(args, *filter.ApprovedAfter)
		argIndex++
	}

	if filter.ApprovedBefore != nil {
		query += fmt.Sprintf(" AND approved_at <= $%d", argIndex)
		args = append(args, *filter.ApprovedBefore)
		argIndex++
	}

	return query, args, argIndex
}
