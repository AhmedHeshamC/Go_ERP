package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"erpgo/internal/domain/inventory/entities"
	"erpgo/internal/domain/inventory/repositories"
	"erpgo/pkg/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PostgresInventoryRepository implements InventoryRepository for PostgreSQL
type PostgresInventoryRepository struct {
	db     *database.Database
	logger interface{} // Can be zerolog.Logger or any logger
}

// NewPostgresInventoryRepository creates a new PostgreSQL inventory repository
func NewPostgresInventoryRepository(db *database.Database) *PostgresInventoryRepository {
	return &PostgresInventoryRepository{
		db: db,
	}
}

// Create creates a new inventory record
func (r *PostgresInventoryRepository) Create(ctx context.Context, inventory *entities.Inventory) error {
	query := `
		INSERT INTO inventory (id, product_id, warehouse_id, quantity_on_hand, quantity_reserved,
		                      reorder_level, max_stock, min_stock, average_cost, last_count_date,
		                      last_counted_by, updated_at, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (product_id, warehouse_id) DO UPDATE SET
			quantity_on_hand = EXCLUDED.quantity_on_hand,
			quantity_reserved = EXCLUDED.quantity_reserved,
			reorder_level = EXCLUDED.reorder_level,
			max_stock = EXCLUDED.max_stock,
			min_stock = EXCLUDED.min_stock,
			average_cost = EXCLUDED.average_cost,
			last_count_date = EXCLUDED.last_count_date,
			last_counted_by = EXCLUDED.last_counted_by,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by
	`

	_, err := r.db.Exec(ctx, query,
		inventory.ID,
		inventory.ProductID,
		inventory.WarehouseID,
		inventory.QuantityOnHand,
		inventory.QuantityReserved,
		inventory.ReorderLevel,
		inventory.MaxStock,
		inventory.MinStock,
		inventory.AverageCost,
		inventory.LastCountDate,
		inventory.LastCountedBy,
		inventory.UpdatedAt,
		inventory.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create inventory: %w", err)
	}

	return nil
}

// GetByID retrieves inventory by ID
func (r *PostgresInventoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Inventory, error) {
	query := `
		SELECT id, product_id, warehouse_id, quantity_on_hand, quantity_reserved,
		       reorder_level, max_stock, min_stock, average_cost, last_count_date,
		       last_counted_by, updated_at, updated_by
		FROM inventory
		WHERE id = $1
	`

	inventory := &entities.Inventory{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&inventory.ID,
		&inventory.ProductID,
		&inventory.WarehouseID,
		&inventory.QuantityOnHand,
		&inventory.QuantityReserved,
		&inventory.ReorderLevel,
		&inventory.MaxStock,
		&inventory.MinStock,
		&inventory.AverageCost,
		&inventory.LastCountDate,
		&inventory.LastCountedBy,
		&inventory.UpdatedAt,
		&inventory.UpdatedBy,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("inventory with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	return inventory, nil
}

// GetByProductAndWarehouse retrieves inventory by product and warehouse
func (r *PostgresInventoryRepository) GetByProductAndWarehouse(ctx context.Context, productID, warehouseID uuid.UUID) (*entities.Inventory, error) {
	query := `
		SELECT id, product_id, warehouse_id, quantity_on_hand, quantity_reserved,
		       reorder_level, max_stock, min_stock, average_cost, last_count_date,
		       last_counted_by, updated_at, updated_by
		FROM inventory
		WHERE product_id = $1 AND warehouse_id = $2
	`

	inventory := &entities.Inventory{}
	err := r.db.QueryRow(ctx, query, productID, warehouseID).Scan(
		&inventory.ID,
		&inventory.ProductID,
		&inventory.WarehouseID,
		&inventory.QuantityOnHand,
		&inventory.QuantityReserved,
		&inventory.ReorderLevel,
		&inventory.MaxStock,
		&inventory.MinStock,
		&inventory.AverageCost,
		&inventory.LastCountDate,
		&inventory.LastCountedBy,
		&inventory.UpdatedAt,
		&inventory.UpdatedBy,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("inventory for product %s in warehouse %s not found", productID, warehouseID)
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	return inventory, nil
}

// Update updates an inventory record
func (r *PostgresInventoryRepository) Update(ctx context.Context, inventory *entities.Inventory) error {
	query := `
		UPDATE inventory
		SET quantity_on_hand = $2, quantity_reserved = $3, reorder_level = $4,
		    max_stock = $5, min_stock = $6, average_cost = $7, last_count_date = $8,
		    last_counted_by = $9, updated_at = $10, updated_by = $11
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		inventory.ID,
		inventory.QuantityOnHand,
		inventory.QuantityReserved,
		inventory.ReorderLevel,
		inventory.MaxStock,
		inventory.MinStock,
		inventory.AverageCost,
		inventory.LastCountDate,
		inventory.LastCountedBy,
		inventory.UpdatedAt,
		inventory.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("inventory with ID %s not found", inventory.ID)
	}

	return nil
}

// Delete deletes an inventory record
func (r *PostgresInventoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM inventory WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete inventory: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("inventory with ID %s not found", id)
	}

	return nil
}

// UpdateStock updates the stock quantity for a product in a warehouse
func (r *PostgresInventoryRepository) UpdateStock(ctx context.Context, productID, warehouseID uuid.UUID, quantity int) error {
	query := `
		UPDATE inventory
		SET quantity_on_hand = $3, updated_at = NOW()
		WHERE product_id = $1 AND warehouse_id = $2
	`

	result, err := r.db.Exec(ctx, query, productID, warehouseID, quantity)
	if err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("inventory for product %s in warehouse %s not found", productID, warehouseID)
	}

	return nil
}

// AdjustStock adjusts the stock quantity by a given amount
func (r *PostgresInventoryRepository) AdjustStock(ctx context.Context, productID, warehouseID uuid.UUID, adjustment int) error {
	query := `
		UPDATE inventory
		SET quantity_on_hand = quantity_on_hand + $3, updated_at = NOW()
		WHERE product_id = $1 AND warehouse_id = $2
	`

	result, err := r.db.Exec(ctx, query, productID, warehouseID, adjustment)
	if err != nil {
		return fmt.Errorf("failed to adjust stock: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("inventory for product %s in warehouse %s not found", productID, warehouseID)
	}

	return nil
}

// ReserveStock reserves a specified quantity of inventory
func (r *PostgresInventoryRepository) ReserveStock(ctx context.Context, productID, warehouseID uuid.UUID, quantity int) error {
	query := `
		UPDATE inventory
		SET quantity_reserved = quantity_reserved + $3, updated_at = NOW()
		WHERE product_id = $1 AND warehouse_id = $2
		  AND quantity_on_hand - quantity_reserved >= $3
	`

	result, err := r.db.Exec(ctx, query, productID, warehouseID, quantity)
	if err != nil {
		return fmt.Errorf("failed to reserve stock: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("insufficient available stock for product %s in warehouse %s", productID, warehouseID)
	}

	return nil
}

// ReleaseStock releases a specified quantity of reserved inventory
func (r *PostgresInventoryRepository) ReleaseStock(ctx context.Context, productID, warehouseID uuid.UUID, quantity int) error {
	query := `
		UPDATE inventory
		SET quantity_reserved = quantity_reserved - $3, updated_at = NOW()
		WHERE product_id = $1 AND warehouse_id = $2
		  AND quantity_reserved >= $3
	`

	result, err := r.db.Exec(ctx, query, productID, warehouseID, quantity)
	if err != nil {
		return fmt.Errorf("failed to release stock: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("insufficient reserved stock for product %s in warehouse %s", productID, warehouseID)
	}

	return nil
}

// GetAvailableStock gets the available stock quantity for a product in a warehouse
func (r *PostgresInventoryRepository) GetAvailableStock(ctx context.Context, productID, warehouseID uuid.UUID) (int, error) {
	query := `
		SELECT quantity_on_hand - quantity_reserved
		FROM inventory
		WHERE product_id = $1 AND warehouse_id = $2
	`

	var availableStock int
	err := r.db.QueryRow(ctx, query, productID, warehouseID).Scan(&availableStock)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("inventory for product %s in warehouse %s not found", productID, warehouseID)
		}
		return 0, fmt.Errorf("failed to get available stock: %w", err)
	}

	return availableStock, nil
}

// List retrieves inventory records with filtering
func (r *PostgresInventoryRepository) List(ctx context.Context, filter *repositories.InventoryFilter) ([]*entities.Inventory, error) {
	query := `
		SELECT i.id, i.product_id, i.warehouse_id, i.quantity_on_hand, i.quantity_reserved,
		       i.reorder_level, i.max_stock, i.min_stock, i.average_cost, i.last_count_date,
		       i.last_counted_by, i.updated_at, i.updated_by
		FROM inventory i
		JOIN products p ON i.product_id = p.id
		JOIN warehouses w ON i.warehouse_id = w.id
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	// Add filters
	if filter != nil {
		if len(filter.IDs) > 0 {
			placeholders := make([]string, len(filter.IDs))
			for i, id := range filter.IDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, id)
				argIndex++
			}
			query += fmt.Sprintf(" AND i.id IN (%s)", strings.Join(placeholders, ","))
		}

		if len(filter.ProductIDs) > 0 {
			placeholders := make([]string, len(filter.ProductIDs))
			for i, id := range filter.ProductIDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, id)
				argIndex++
			}
			query += fmt.Sprintf(" AND i.product_id IN (%s)", strings.Join(placeholders, ","))
		}

		if len(filter.WarehouseIDs) > 0 {
			placeholders := make([]string, len(filter.WarehouseIDs))
			for i, id := range filter.WarehouseIDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, id)
				argIndex++
			}
			query += fmt.Sprintf(" AND i.warehouse_id IN (%s)", strings.Join(placeholders, ","))
		}

		if filter.SKU != "" {
			query += fmt.Sprintf(" AND p.sku ILIKE $%d", argIndex)
			args = append(args, "%"+filter.SKU+"%")
			argIndex++
		}

		if filter.ProductName != "" {
			query += fmt.Sprintf(" AND p.name ILIKE $%d", argIndex)
			args = append(args, "%"+filter.ProductName+"%")
			argIndex++
		}

		if filter.WarehouseCode != "" {
			query += fmt.Sprintf(" AND w.code ILIKE $%d", argIndex)
			args = append(args, "%"+filter.WarehouseCode+"%")
			argIndex++
		}

		if filter.IsLowStock != nil {
			if *filter.IsLowStock {
				query += fmt.Sprintf(" AND i.quantity_on_hand <= i.reorder_level")
			} else {
				query += fmt.Sprintf(" AND i.quantity_on_hand > i.reorder_level")
			}
		}

		if filter.IsOutOfStock != nil {
			if *filter.IsOutOfStock {
				query += fmt.Sprintf(" AND i.quantity_on_hand = 0")
			} else {
				query += fmt.Sprintf(" AND i.quantity_on_hand > 0")
			}
		}

		if filter.IsOverstock != nil {
			if *filter.IsOverstock {
				query += fmt.Sprintf(" AND i.quantity_on_hand > i.max_stock")
			} else {
				query += fmt.Sprintf(" AND (i.max_stock IS NULL OR i.quantity_on_hand <= i.max_stock)")
			}
		}

		if filter.MinQuantity != nil {
			query += fmt.Sprintf(" AND i.quantity_on_hand >= $%d", argIndex)
			args = append(args, *filter.MinQuantity)
			argIndex++
		}

		if filter.MaxQuantity != nil {
			query += fmt.Sprintf(" AND i.quantity_on_hand <= $%d", argIndex)
			args = append(args, *filter.MaxQuantity)
			argIndex++
		}

		if filter.MinAverageCost != nil {
			query += fmt.Sprintf(" AND i.average_cost >= $%d", argIndex)
			args = append(args, *filter.MinAverageCost)
			argIndex++
		}

		if filter.MaxAverageCost != nil {
			query += fmt.Sprintf(" AND i.average_cost <= $%d", argIndex)
			args = append(args, *filter.MaxAverageCost)
			argIndex++
		}

		if filter.LastCountedAfter != nil {
			query += fmt.Sprintf(" AND i.last_count_date >= $%d", argIndex)
			args = append(args, *filter.LastCountedAfter)
			argIndex++
		}

		if filter.LastCountedBefore != nil {
			query += fmt.Sprintf(" AND i.last_count_date <= $%d", argIndex)
			args = append(args, *filter.LastCountedBefore)
			argIndex++
		}

		if filter.UpdatedAfter != nil {
			query += fmt.Sprintf(" AND i.updated_at >= $%d", argIndex)
			args = append(args, *filter.UpdatedAfter)
			argIndex++
		}

		if filter.UpdatedBefore != nil {
			query += fmt.Sprintf(" AND i.updated_at <= $%d", argIndex)
			args = append(args, *filter.UpdatedBefore)
			argIndex++
		}

		// Add ordering
		orderBy := "p.name"
		if filter.OrderBy != "" {
			orderBy = filter.OrderBy
		}

		order := "ASC"
		if filter.Order != "" {
			order = strings.ToUpper(filter.Order)
		}

		query += fmt.Sprintf(" ORDER BY %s %s", orderBy, order)

		// Add pagination
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argIndex)
			args = append(args, filter.Limit)
			argIndex++

			if filter.Offset > 0 {
				query += fmt.Sprintf(" OFFSET $%d", argIndex)
				args = append(args, filter.Offset)
			}
		}
	} else {
		query += " ORDER BY p.name ASC, w.name ASC"
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory: %w", err)
	}
	defer rows.Close()

	var inventories []*entities.Inventory
	for rows.Next() {
		inventory := &entities.Inventory{}
		err := rows.Scan(
			&inventory.ID,
			&inventory.ProductID,
			&inventory.WarehouseID,
			&inventory.QuantityOnHand,
			&inventory.QuantityReserved,
			&inventory.ReorderLevel,
			&inventory.MaxStock,
			&inventory.MinStock,
			&inventory.AverageCost,
			&inventory.LastCountDate,
			&inventory.LastCountedBy,
			&inventory.UpdatedAt,
			&inventory.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory row: %w", err)
		}
		inventories = append(inventories, inventory)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory rows: %w", err)
	}

	return inventories, nil
}

// GetByProduct retrieves inventory records for a specific product
func (r *PostgresInventoryRepository) GetByProduct(ctx context.Context, productID uuid.UUID) ([]*entities.Inventory, error) {
	query := `
		SELECT id, product_id, warehouse_id, quantity_on_hand, quantity_reserved,
		       reorder_level, max_stock, min_stock, average_cost, last_count_date,
		       last_counted_by, updated_at, updated_by
		FROM inventory
		WHERE product_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory by product: %w", err)
	}
	defer rows.Close()

	var inventories []*entities.Inventory
	for rows.Next() {
		inventory := &entities.Inventory{}
		err := rows.Scan(
			&inventory.ID,
			&inventory.ProductID,
			&inventory.WarehouseID,
			&inventory.QuantityOnHand,
			&inventory.QuantityReserved,
			&inventory.ReorderLevel,
			&inventory.MaxStock,
			&inventory.MinStock,
			&inventory.AverageCost,
			&inventory.LastCountDate,
			&inventory.LastCountedBy,
			&inventory.UpdatedAt,
			&inventory.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory row: %w", err)
		}
		inventories = append(inventories, inventory)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory rows: %w", err)
	}

	return inventories, nil
}

// GetByWarehouse retrieves inventory records for a specific warehouse
func (r *PostgresInventoryRepository) GetByWarehouse(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error) {
	query := `
		SELECT i.id, i.product_id, i.warehouse_id, i.quantity_on_hand, i.quantity_reserved,
		       i.reorder_level, i.max_stock, i.min_stock, i.average_cost, i.last_count_date,
		       i.last_counted_by, i.updated_at, i.updated_by
		FROM inventory i
		JOIN products p ON i.product_id = p.id
		WHERE i.warehouse_id = $1
		ORDER BY p.name ASC
	`

	rows, err := r.db.Query(ctx, query, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory by warehouse: %w", err)
	}
	defer rows.Close()

	var inventories []*entities.Inventory
	for rows.Next() {
		inventory := &entities.Inventory{}
		err := rows.Scan(
			&inventory.ID,
			&inventory.ProductID,
			&inventory.WarehouseID,
			&inventory.QuantityOnHand,
			&inventory.QuantityReserved,
			&inventory.ReorderLevel,
			&inventory.MaxStock,
			&inventory.MinStock,
			&inventory.AverageCost,
			&inventory.LastCountDate,
			&inventory.LastCountedBy,
			&inventory.UpdatedAt,
			&inventory.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory row: %w", err)
		}
		inventories = append(inventories, inventory)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory rows: %w", err)
	}

	return inventories, nil
}

// GetWarehouseInventory is an alias for GetByWarehouse
func (r *PostgresInventoryRepository) GetWarehouseInventory(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error) {
	return r.GetByWarehouse(ctx, warehouseID)
}

// GetProductInventory is an alias for GetByProduct
func (r *PostgresInventoryRepository) GetProductInventory(ctx context.Context, productID uuid.UUID) ([]*entities.Inventory, error) {
	return r.GetByProduct(ctx, productID)
}

// GetLowStockItems retrieves low stock items for a warehouse
func (r *PostgresInventoryRepository) GetLowStockItems(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error) {
	query := `
		SELECT i.id, i.product_id, i.warehouse_id, i.quantity_on_hand, i.quantity_reserved,
		       i.reorder_level, i.max_stock, i.min_stock, i.average_cost, i.last_count_date,
		       i.last_counted_by, i.updated_at, i.updated_by
		FROM inventory i
		JOIN products p ON i.product_id = p.id
		WHERE i.warehouse_id = $1 AND i.quantity_on_hand <= i.reorder_level AND i.quantity_on_hand > 0
		ORDER BY (i.quantity_on_hand - i.reorder_level) ASC
	`

	rows, err := r.db.Query(ctx, query, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query low stock items: %w", err)
	}
	defer rows.Close()

	var inventories []*entities.Inventory
	for rows.Next() {
		inventory := &entities.Inventory{}
		err := rows.Scan(
			&inventory.ID,
			&inventory.ProductID,
			&inventory.WarehouseID,
			&inventory.QuantityOnHand,
			&inventory.QuantityReserved,
			&inventory.ReorderLevel,
			&inventory.MaxStock,
			&inventory.MinStock,
			&inventory.AverageCost,
			&inventory.LastCountDate,
			&inventory.LastCountedBy,
			&inventory.UpdatedAt,
			&inventory.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory row: %w", err)
		}
		inventories = append(inventories, inventory)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory rows: %w", err)
	}

	return inventories, nil
}

// GetLowStockItemsAll retrieves all low stock items across all warehouses
func (r *PostgresInventoryRepository) GetLowStockItemsAll(ctx context.Context) ([]*entities.Inventory, error) {
	query := `
		SELECT i.id, i.product_id, i.warehouse_id, i.quantity_on_hand, i.quantity_reserved,
		       i.reorder_level, i.max_stock, i.min_stock, i.average_cost, i.last_count_date,
		       i.last_counted_by, i.updated_at, i.updated_by
		FROM inventory i
		JOIN products p ON i.product_id = p.id
		JOIN warehouses w ON i.warehouse_id = w.id
		WHERE i.quantity_on_hand <= i.reorder_level AND i.quantity_on_hand > 0
		ORDER BY w.name ASC, (i.quantity_on_hand - i.reorder_level) ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all low stock items: %w", err)
	}
	defer rows.Close()

	var inventories []*entities.Inventory
	for rows.Next() {
		inventory := &entities.Inventory{}
		err := rows.Scan(
			&inventory.ID,
			&inventory.ProductID,
			&inventory.WarehouseID,
			&inventory.QuantityOnHand,
			&inventory.QuantityReserved,
			&inventory.ReorderLevel,
			&inventory.MaxStock,
			&inventory.MinStock,
			&inventory.AverageCost,
			&inventory.LastCountDate,
			&inventory.LastCountedBy,
			&inventory.UpdatedAt,
			&inventory.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory row: %w", err)
		}
		inventories = append(inventories, inventory)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory rows: %w", err)
	}

	return inventories, nil
}

// GetOutOfStockItems retrieves out of stock items for a warehouse
func (r *PostgresInventoryRepository) GetOutOfStockItems(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error) {
	query := `
		SELECT i.id, i.product_id, i.warehouse_id, i.quantity_on_hand, i.quantity_reserved,
		       i.reorder_level, i.max_stock, i.min_stock, i.average_cost, i.last_count_date,
		       i.last_counted_by, i.updated_at, i.updated_by
		FROM inventory i
		JOIN products p ON i.product_id = p.id
		WHERE i.warehouse_id = $1 AND i.quantity_on_hand = 0
		ORDER BY p.name ASC
	`

	rows, err := r.db.Query(ctx, query, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query out of stock items: %w", err)
	}
	defer rows.Close()

	var inventories []*entities.Inventory
	for rows.Next() {
		inventory := &entities.Inventory{}
		err := rows.Scan(
			&inventory.ID,
			&inventory.ProductID,
			&inventory.WarehouseID,
			&inventory.QuantityOnHand,
			&inventory.QuantityReserved,
			&inventory.ReorderLevel,
			&inventory.MaxStock,
			&inventory.MinStock,
			&inventory.AverageCost,
			&inventory.LastCountDate,
			&inventory.LastCountedBy,
			&inventory.UpdatedAt,
			&inventory.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory row: %w", err)
		}
		inventories = append(inventories, inventory)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory rows: %w", err)
	}

	return inventories, nil
}

// GetOverstockItems retrieves overstock items for a warehouse
func (r *PostgresInventoryRepository) GetOverstockItems(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error) {
	query := `
		SELECT i.id, i.product_id, i.warehouse_id, i.quantity_on_hand, i.quantity_reserved,
		       i.reorder_level, i.max_stock, i.min_stock, i.average_cost, i.last_count_date,
		       i.last_counted_by, i.updated_at, i.updated_by
		FROM inventory i
		JOIN products p ON i.product_id = p.id
		WHERE i.warehouse_id = $1 AND i.max_stock IS NOT NULL AND i.quantity_on_hand > i.max_stock
		ORDER BY (i.quantity_on_hand - i.max_stock) DESC
	`

	rows, err := r.db.Query(ctx, query, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query overstock items: %w", err)
	}
	defer rows.Close()

	var inventories []*entities.Inventory
	for rows.Next() {
		inventory := &entities.Inventory{}
		err := rows.Scan(
			&inventory.ID,
			&inventory.ProductID,
			&inventory.WarehouseID,
			&inventory.QuantityOnHand,
			&inventory.QuantityReserved,
			&inventory.ReorderLevel,
			&inventory.MaxStock,
			&inventory.MinStock,
			&inventory.AverageCost,
			&inventory.LastCountDate,
			&inventory.LastCountedBy,
			&inventory.UpdatedAt,
			&inventory.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory row: %w", err)
		}
		inventories = append(inventories, inventory)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory rows: %w", err)
	}

	return inventories, nil
}

// Search searches inventory records
func (r *PostgresInventoryRepository) Search(ctx context.Context, query string, limit int) ([]*entities.Inventory, error) {
	sqlQuery := `
		SELECT i.id, i.product_id, i.warehouse_id, i.quantity_on_hand, i.quantity_reserved,
		       i.reorder_level, i.max_stock, i.min_stock, i.average_cost, i.last_count_date,
		       i.last_counted_by, i.updated_at, i.updated_by
		FROM inventory i
		JOIN products p ON i.product_id = p.id
		JOIN warehouses w ON i.warehouse_id = w.id
		WHERE p.name ILIKE $1 OR p.sku ILIKE $1 OR w.name ILIKE $1 OR w.code ILIKE $1
		ORDER BY p.name ASC, w.name ASC
	`

	args := []interface{}{"%" + query + "%"}

	if limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", 2)
		args = append(args, limit)
	}

	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search inventory: %w", err)
	}
	defer rows.Close()

	var inventories []*entities.Inventory
	for rows.Next() {
		inventory := &entities.Inventory{}
		err := rows.Scan(
			&inventory.ID,
			&inventory.ProductID,
			&inventory.WarehouseID,
			&inventory.QuantityOnHand,
			&inventory.QuantityReserved,
			&inventory.ReorderLevel,
			&inventory.MaxStock,
			&inventory.MinStock,
			&inventory.AverageCost,
			&inventory.LastCountDate,
			&inventory.LastCountedBy,
			&inventory.UpdatedAt,
			&inventory.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory row: %w", err)
		}
		inventories = append(inventories, inventory)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory rows: %w", err)
	}

	return inventories, nil
}

// Count counts inventory records with filtering
func (r *PostgresInventoryRepository) Count(ctx context.Context, filter *repositories.InventoryFilter) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM inventory i
		JOIN products p ON i.product_id = p.id
		JOIN warehouses w ON i.warehouse_id = w.id
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	// Add filters (same logic as List method)
	if filter != nil {
		if len(filter.IDs) > 0 {
			placeholders := make([]string, len(filter.IDs))
			for i, id := range filter.IDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, id)
				argIndex++
			}
			query += fmt.Sprintf(" AND i.id IN (%s)", strings.Join(placeholders, ","))
		}

		if len(filter.ProductIDs) > 0 {
			placeholders := make([]string, len(filter.ProductIDs))
			for i, id := range filter.ProductIDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, id)
				argIndex++
			}
			query += fmt.Sprintf(" AND i.product_id IN (%s)", strings.Join(placeholders, ","))
		}

		if len(filter.WarehouseIDs) > 0 {
			placeholders := make([]string, len(filter.WarehouseIDs))
			for i, id := range filter.WarehouseIDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, id)
				argIndex++
			}
			query += fmt.Sprintf(" AND i.warehouse_id IN (%s)", strings.Join(placeholders, ","))
		}

		if filter.SKU != "" {
			query += fmt.Sprintf(" AND p.sku ILIKE $%d", argIndex)
			args = append(args, "%"+filter.SKU+"%")
			argIndex++
		}

		if filter.ProductName != "" {
			query += fmt.Sprintf(" AND p.name ILIKE $%d", argIndex)
			args = append(args, "%"+filter.ProductName+"%")
			argIndex++
		}

		if filter.WarehouseCode != "" {
			query += fmt.Sprintf(" AND w.code ILIKE $%d", argIndex)
			args = append(args, "%"+filter.WarehouseCode+"%")
			argIndex++
		}

		if filter.IsLowStock != nil {
			if *filter.IsLowStock {
				query += fmt.Sprintf(" AND i.quantity_on_hand <= i.reorder_level")
			} else {
				query += fmt.Sprintf(" AND i.quantity_on_hand > i.reorder_level")
			}
		}

		if filter.IsOutOfStock != nil {
			if *filter.IsOutOfStock {
				query += fmt.Sprintf(" AND i.quantity_on_hand = 0")
			} else {
				query += fmt.Sprintf(" AND i.quantity_on_hand > 0")
			}
		}

		if filter.IsOverstock != nil {
			if *filter.IsOverstock {
				query += fmt.Sprintf(" AND i.quantity_on_hand > i.max_stock")
			} else {
				query += fmt.Sprintf(" AND (i.max_stock IS NULL OR i.quantity_on_hand <= i.max_stock)")
			}
		}

		if filter.MinQuantity != nil {
			query += fmt.Sprintf(" AND i.quantity_on_hand >= $%d", argIndex)
			args = append(args, *filter.MinQuantity)
			argIndex++
		}

		if filter.MaxQuantity != nil {
			query += fmt.Sprintf(" AND i.quantity_on_hand <= $%d", argIndex)
			args = append(args, *filter.MaxQuantity)
			argIndex++
		}

		if filter.MinAverageCost != nil {
			query += fmt.Sprintf(" AND i.average_cost >= $%d", argIndex)
			args = append(args, *filter.MinAverageCost)
			argIndex++
		}

		if filter.MaxAverageCost != nil {
			query += fmt.Sprintf(" AND i.average_cost <= $%d", argIndex)
			args = append(args, *filter.MaxAverageCost)
			argIndex++
		}

		if filter.LastCountedAfter != nil {
			query += fmt.Sprintf(" AND i.last_count_date >= $%d", argIndex)
			args = append(args, *filter.LastCountedAfter)
			argIndex++
		}

		if filter.LastCountedBefore != nil {
			query += fmt.Sprintf(" AND i.last_count_date <= $%d", argIndex)
			args = append(args, *filter.LastCountedBefore)
			argIndex++
		}

		if filter.UpdatedAfter != nil {
			query += fmt.Sprintf(" AND i.updated_at >= $%d", argIndex)
			args = append(args, *filter.UpdatedAfter)
			argIndex++
		}

		if filter.UpdatedBefore != nil {
			query += fmt.Sprintf(" AND i.updated_at <= $%d", argIndex)
			args = append(args, *filter.UpdatedBefore)
			argIndex++
		}
	}

	var count int
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count inventory: %w", err)
	}

	return count, nil
}

// CountByWarehouse counts inventory records for a specific warehouse
func (r *PostgresInventoryRepository) CountByWarehouse(ctx context.Context, warehouseID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM inventory WHERE warehouse_id = $1`

	var count int
	err := r.db.QueryRow(ctx, query, warehouseID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count inventory by warehouse: %w", err)
	}

	return count, nil
}

// CountByProduct counts inventory records for a specific product
func (r *PostgresInventoryRepository) CountByProduct(ctx context.Context, productID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM inventory WHERE product_id = $1`

	var count int
	err := r.db.QueryRow(ctx, query, productID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count inventory by product: %w", err)
	}

	return count, nil
}

// ExistsByProductAndWarehouse checks if inventory exists for a product in a warehouse
func (r *PostgresInventoryRepository) ExistsByProductAndWarehouse(ctx context.Context, productID, warehouseID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM inventory WHERE product_id = $1 AND warehouse_id = $2)`

	var exists bool
	err := r.db.QueryRow(ctx, query, productID, warehouseID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check inventory existence: %w", err)
	}

	return exists, nil
}

// BulkCreate creates multiple inventory records
func (r *PostgresInventoryRepository) BulkCreate(ctx context.Context, inventories []*entities.Inventory) error {
	if len(inventories) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO inventory (id, product_id, warehouse_id, quantity_on_hand, quantity_reserved,
		                      reorder_level, max_stock, min_stock, average_cost, last_count_date,
		                      last_counted_by, updated_at, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (product_id, warehouse_id) DO UPDATE SET
			quantity_on_hand = EXCLUDED.quantity_on_hand,
			quantity_reserved = EXCLUDED.quantity_reserved,
			reorder_level = EXCLUDED.reorder_level,
			max_stock = EXCLUDED.max_stock,
			min_stock = EXCLUDED.min_stock,
			average_cost = EXCLUDED.average_cost,
			last_count_date = EXCLUDED.last_count_date,
			last_counted_by = EXCLUDED.last_counted_by,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by
	`

	for _, inventory := range inventories {
		_, err = tx.Exec(ctx, query,
			inventory.ID,
			inventory.ProductID,
			inventory.WarehouseID,
			inventory.QuantityOnHand,
			inventory.QuantityReserved,
			inventory.ReorderLevel,
			inventory.MaxStock,
			inventory.MinStock,
			inventory.AverageCost,
			inventory.LastCountDate,
			inventory.LastCountedBy,
			inventory.UpdatedAt,
			inventory.UpdatedBy,
		)
		if err != nil {
			return fmt.Errorf("failed to create inventory: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BulkUpdate updates multiple inventory records
func (r *PostgresInventoryRepository) BulkUpdate(ctx context.Context, inventories []*entities.Inventory) error {
	if len(inventories) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE inventory
		SET quantity_on_hand = $2, quantity_reserved = $3, reorder_level = $4,
		    max_stock = $5, min_stock = $6, average_cost = $7, last_count_date = $8,
		    last_counted_by = $9, updated_at = $10, updated_by = $11
		WHERE id = $1
	`

	for _, inventory := range inventories {
		result, err := tx.Exec(ctx, query,
			inventory.ID,
			inventory.QuantityOnHand,
			inventory.QuantityReserved,
			inventory.ReorderLevel,
			inventory.MaxStock,
			inventory.MinStock,
			inventory.AverageCost,
			inventory.LastCountDate,
			inventory.LastCountedBy,
			inventory.UpdatedAt,
			inventory.UpdatedBy,
		)
		if err != nil {
			return fmt.Errorf("failed to update inventory: %w", err)
		}

		rowsAffected := result.RowsAffected()
		if rowsAffected == 0 {
			return fmt.Errorf("inventory with ID %s not found", inventory.ID)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BulkDelete deletes multiple inventory records
func (r *PostgresInventoryRepository) BulkDelete(ctx context.Context, inventoryIDs []uuid.UUID) error {
	if len(inventoryIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(inventoryIDs))
	args := make([]interface{}, len(inventoryIDs))
	for i, id := range inventoryIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM inventory WHERE id IN (%s)", strings.Join(placeholders, ","))

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk delete inventory: %w", err)
	}

	return nil
}

// BulkAdjustStock performs multiple stock adjustments
func (r *PostgresInventoryRepository) BulkAdjustStock(ctx context.Context, adjustments []repositories.StockAdjustment) error {
	if len(adjustments) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE inventory
		SET quantity_on_hand = quantity_on_hand + $3, updated_at = NOW(), updated_by = $4
		WHERE product_id = $1 AND warehouse_id = $2
	`

	for _, adjustment := range adjustments {
		result, err := tx.Exec(ctx, query,
			adjustment.ProductID,
			adjustment.WarehouseID,
			adjustment.Adjustment,
			adjustment.UpdatedBy,
		)
		if err != nil {
			return fmt.Errorf("failed to adjust stock: %w", err)
		}

		rowsAffected := result.RowsAffected()
		if rowsAffected == 0 {
			return fmt.Errorf("inventory for product %s in warehouse %s not found",
				adjustment.ProductID, adjustment.WarehouseID)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BulkReserveStock performs multiple stock reservations
func (r *PostgresInventoryRepository) BulkReserveStock(ctx context.Context, reservations []repositories.StockReservation) error {
	if len(reservations) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE inventory
		SET quantity_reserved = quantity_reserved + $3, updated_at = NOW()
		WHERE product_id = $1 AND warehouse_id = $2
		  AND quantity_on_hand - quantity_reserved >= $3
	`

	for _, reservation := range reservations {
		result, err := tx.Exec(ctx, query,
			reservation.ProductID,
			reservation.WarehouseID,
			reservation.Quantity,
		)
		if err != nil {
			return fmt.Errorf("failed to reserve stock: %w", err)
		}

		rowsAffected := result.RowsAffected()
		if rowsAffected == 0 {
			return fmt.Errorf("insufficient available stock for product %s in warehouse %s",
				reservation.ProductID, reservation.WarehouseID)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetInventoryValue calculates the total inventory value
func (r *PostgresInventoryRepository) GetInventoryValue(ctx context.Context, warehouseID *uuid.UUID) (float64, error) {
	query := `
		SELECT COALESCE(SUM(quantity_on_hand * average_cost), 0)
		FROM inventory
	`

	args := []interface{}{}
	if warehouseID != nil {
		query += " WHERE warehouse_id = $1"
		args = append(args, *warehouseID)
	}

	var totalValue float64
	err := r.db.QueryRow(ctx, query, args...).Scan(&totalValue)
	if err != nil {
		return 0, fmt.Errorf("failed to get inventory value: %w", err)
	}

	return totalValue, nil
}

// GetInventoryLevels retrieves inventory levels for a product
func (r *PostgresInventoryRepository) GetInventoryLevels(ctx context.Context, productID uuid.UUID) ([]*repositories.InventoryLevel, error) {
	query := `
		SELECT i.product_id, p.name, p.sku, i.warehouse_id, w.name, w.code,
		       i.quantity_on_hand, i.quantity_reserved, i.quantity_on_hand - i.quantity_reserved,
		       i.reorder_level, i.updated_at
		FROM inventory i
		JOIN products p ON i.product_id = p.id
		JOIN warehouses w ON i.warehouse_id = w.id
		WHERE i.product_id = $1
		ORDER BY w.name ASC
	`

	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory levels: %w", err)
	}
	defer rows.Close()

	var levels []*repositories.InventoryLevel
	for rows.Next() {
		level := &repositories.InventoryLevel{}
		err := rows.Scan(
			&level.ProductID,
			&level.ProductName,
			&level.SKU,
			&level.WarehouseID,
			&level.WarehouseName,
			&level.Quantity,
			&level.Reserved,
			&level.Available,
			&level.ReorderLevel,
			&level.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory level row: %w", err)
		}
		levels = append(levels, level)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory level rows: %w", err)
	}

	return levels, nil
}

// GetStockLevels retrieves detailed stock levels
func (r *PostgresInventoryRepository) GetStockLevels(ctx context.Context, filter *repositories.InventoryFilter) ([]*repositories.StockLevel, error) {
	query := `
		SELECT i.product_id, p.name, p.sku, i.warehouse_id, w.name, w.code,
		       i.quantity_on_hand, i.quantity_reserved, i.quantity_on_hand - i.quantity_reserved,
		       i.reorder_level, i.min_stock, i.max_stock, i.average_cost,
		       i.quantity_on_hand * i.average_cost, i.updated_at
		FROM inventory i
		JOIN products p ON i.product_id = p.id
		JOIN warehouses w ON i.warehouse_id = w.id
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	// Add filters (similar to List method)
	if filter != nil {
		// Add filters here similar to List method...
		// For brevity, implementing basic filtering only
		if len(filter.ProductIDs) > 0 {
			placeholders := make([]string, len(filter.ProductIDs))
			for i, id := range filter.ProductIDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, id)
				argIndex++
			}
			query += fmt.Sprintf(" AND i.product_id IN (%s)", strings.Join(placeholders, ","))
		}

		if len(filter.WarehouseIDs) > 0 {
			placeholders := make([]string, len(filter.WarehouseIDs))
			for i, id := range filter.WarehouseIDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, id)
				argIndex++
			}
			query += fmt.Sprintf(" AND i.warehouse_id IN (%s)", strings.Join(placeholders, ","))
		}
	}

	query += " ORDER BY p.name ASC, w.name ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query stock levels: %w", err)
	}
	defer rows.Close()

	var levels []*repositories.StockLevel
	for rows.Next() {
		level := &repositories.StockLevel{}
		err := rows.Scan(
			&level.ProductID,
			&level.ProductName,
			&level.SKU,
			&level.WarehouseID,
			&level.WarehouseName,
			&level.Quantity,
			&level.Reserved,
			&level.Available,
			&level.ReorderLevel,
			&level.MinStock,
			&level.MaxStock,
			&level.AverageCost,
			&level.TotalValue,
			&level.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stock level row: %w", err)
		}

		// Calculate status
		if level.Quantity == 0 {
			level.Status = "OUT_OF_STOCK"
		} else if level.MinStock != nil && level.Quantity < *level.MinStock {
			level.Status = "UNDERSTOCK"
		} else if level.Quantity <= level.ReorderLevel {
			level.Status = "LOW_STOCK"
		} else if level.MaxStock != nil && level.Quantity > *level.MaxStock {
			level.Status = "OVERSTOCK"
		} else {
			level.Status = "NORMAL"
		}

		levels = append(levels, level)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating stock level rows: %w", err)
	}

	return levels, nil
}

// GetInventoryTurnover calculates inventory turnover for a product
func (r *PostgresInventoryRepository) GetInventoryTurnover(ctx context.Context, productID uuid.UUID, warehouseID *uuid.UUID, days int) (*repositories.InventoryTurnover, error) {
	// This is a simplified implementation
	// In a real system, you'd want to query from inventory_transactions table
	// and calculate based on actual sales data

	query := `
		SELECT
			i.product_id,
			p.name,
			COALESCE(i.warehouse_id, $2::uuid) as warehouse_id,
			COALESCE(w.name, 'All Warehouses') as warehouse_name,
			i.quantity_on_hand as ending_stock,
			i.average_cost
		FROM inventory i
		JOIN products p ON i.product_id = p.id
		LEFT JOIN warehouses w ON i.warehouse_id = w.id
		WHERE i.product_id = $1
	`

	args := []interface{}{productID}
	if warehouseID != nil {
		query += " AND i.warehouse_id = $2"
		args = append(args, *warehouseID)
	} else {
		query += " GROUP BY i.product_id, p.name, i.average_cost"
	}

	turnover := &repositories.InventoryTurnover{
		ProductID:       productID,
		Days:            days,
		BeginningStock:  0, // Would need historical data
		EndingStock:     0,
		CostOfGoodsSold: 0, // Would need transaction data
		TurnoverRate:    0,
		DaysOfSupply:    0,
	}

	var warehouseName string
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&turnover.ProductID,
		&turnover.ProductName,
		&turnover.WarehouseID,
		&warehouseName,
		&turnover.EndingStock,
		&turnover.CostOfGoodsSold,
	)

	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to calculate inventory turnover: %w", err)
	}

	if warehouseName != "" {
		turnover.WarehouseName = warehouseName
	}

	// Simplified calculations - in reality would use transaction data
	if turnover.EndingStock > 0 && turnover.CostOfGoodsSold > 0 {
		turnover.TurnoverRate = turnover.CostOfGoodsSold / float64(turnover.EndingStock)
		if turnover.TurnoverRate > 0 {
			turnover.DaysOfSupply = float64(days) / turnover.TurnoverRate
		}
	}

	return turnover, nil
}

// GetAgingInventory retrieves aging inventory items
func (r *PostgresInventoryRepository) GetAgingInventory(ctx context.Context, warehouseID *uuid.UUID, days int) ([]*repositories.AgingInventoryItem, error) {
	query := `
		SELECT
			i.id, i.product_id, p.name, p.sku,
			COALESCE(i.warehouse_id, $2::uuid) as warehouse_id,
			COALESCE(w.name, 'All Warehouses') as warehouse_name,
			i.quantity_on_hand, i.average_cost, i.quantity_on_hand * i.average_cost,
			COALESCE(i.updated_at, NOW()) as last_transaction
		FROM inventory i
		JOIN products p ON i.product_id = p.id
		LEFT JOIN warehouses w ON i.warehouse_id = w.id
		WHERE i.quantity_on_hand > 0
	`

	args := []interface{}{}
	if warehouseID != nil {
		query += " AND i.warehouse_id = $1"
		args = append(args, *warehouseID)
		query = strings.Replace(query, "$2::uuid", "$1::uuid", 1)
	} else {
		query += " GROUP BY i.id, i.product_id, p.name, p.sku, i.warehouse_id, w.name, i.quantity_on_hand, i.average_cost, i.updated_at"
		args = append(args, nil)
	}

	query += " ORDER BY i.updated_at ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query aging inventory: %w", err)
	}
	defer rows.Close()

	var items []*repositories.AgingInventoryItem
	now := time.Now()

	for rows.Next() {
		item := &repositories.AgingInventoryItem{}
		err := rows.Scan(
			&item.InventoryID,
			&item.ProductID,
			&item.ProductName,
			&item.SKU,
			&item.WarehouseID,
			&item.WarehouseName,
			&item.Quantity,
			&item.AverageCost,
			&item.TotalValue,
			&item.LastTransaction,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan aging inventory row: %w", err)
		}

		// Calculate days since last transaction
		item.DaysSinceTransaction = int(now.Sub(item.LastTransaction).Hours() / 24)

		// Filter by days threshold
		if item.DaysSinceTransaction >= days {
			items = append(items, item)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating aging inventory rows: %w", err)
	}

	return items, nil
}

// GetItemsForCycleCount retrieves items due for cycle counting
func (r *PostgresInventoryRepository) GetItemsForCycleCount(ctx context.Context, warehouseID uuid.UUID, limit int) ([]*entities.Inventory, error) {
	query := `
		SELECT i.id, i.product_id, i.warehouse_id, i.quantity_on_hand, i.quantity_reserved,
		       i.reorder_level, i.max_stock, i.min_stock, i.average_cost, i.last_count_date,
		       i.last_counted_by, i.updated_at, i.updated_by
		FROM inventory i
		JOIN products p ON i.product_id = p.id
		WHERE i.warehouse_id = $1
		ORDER BY
			CASE WHEN i.last_count_date IS NULL THEN 0 ELSE 1 END,
			i.last_count_date ASC,
			(i.quantity_on_hand - i.reorder_level) ASC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", 2)
	}

	rows, err := r.db.Query(ctx, query, warehouseID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query cycle count items: %w", err)
	}
	defer rows.Close()

	var inventories []*entities.Inventory
	for rows.Next() {
		inventory := &entities.Inventory{}
		err := rows.Scan(
			&inventory.ID,
			&inventory.ProductID,
			&inventory.WarehouseID,
			&inventory.QuantityOnHand,
			&inventory.QuantityReserved,
			&inventory.ReorderLevel,
			&inventory.MaxStock,
			&inventory.MinStock,
			&inventory.AverageCost,
			&inventory.LastCountDate,
			&inventory.LastCountedBy,
			&inventory.UpdatedAt,
			&inventory.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory row: %w", err)
		}
		inventories = append(inventories, inventory)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory rows: %w", err)
	}

	return inventories, nil
}

// UpdateCycleCount updates a cycle count
func (r *PostgresInventoryRepository) UpdateCycleCount(ctx context.Context, inventoryID uuid.UUID, countedQuantity int, countedBy uuid.UUID) error {
	query := `
		UPDATE inventory
		SET quantity_on_hand = $2, last_count_date = NOW(), last_counted_by = $3, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, inventoryID, countedQuantity, countedBy)
	if err != nil {
		return fmt.Errorf("failed to update cycle count: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("inventory with ID %s not found", inventoryID)
	}

	return nil
}

// GetLastCycleCountDate retrieves the last cycle count date for inventory
func (r *PostgresInventoryRepository) GetLastCycleCountDate(ctx context.Context, inventoryID uuid.UUID) (*time.Time, error) {
	query := `SELECT last_count_date FROM inventory WHERE id = $1`

	var lastCountDate sql.NullTime
	err := r.db.QueryRow(ctx, query, inventoryID).Scan(&lastCountDate)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("inventory with ID %s not found", inventoryID)
		}
		return nil, fmt.Errorf("failed to get last cycle count date: %w", err)
	}

	if lastCountDate.Valid {
		return &lastCountDate.Time, nil
	}

	return nil, nil
}

// ReconcileStock performs stock reconciliation
func (r *PostgresInventoryRepository) ReconcileStock(ctx context.Context, inventoryID uuid.UUID, systemQuantity, physicalQuantity int, reason string, reconciledBy uuid.UUID) error {
	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update inventory with physical count
	query := `
		UPDATE inventory
		SET quantity_on_hand = $2, updated_at = NOW(), updated_by = $3
		WHERE id = $1
	`

	result, err := tx.Exec(ctx, query, inventoryID, physicalQuantity, reconciledBy)
	if err != nil {
		return fmt.Errorf("failed to update inventory during reconciliation: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("inventory with ID %s not found", inventoryID)
	}

	// Create a reconciliation transaction record (would need inventory_reconciliation table)
	// TODO: Implement when inventory_reconciliation table is created

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit reconciliation transaction: %w", err)
	}

	return nil
}

// GetReconciliationHistory retrieves reconciliation history
func (r *PostgresInventoryRepository) GetReconciliationHistory(ctx context.Context, inventoryID uuid.UUID, limit int) ([]*repositories.InventoryReconciliation, error) {
	// TODO: Implement when inventory_reconciliation table is created
	return nil, fmt.Errorf("reconciliation history not yet implemented")
}
