package repositories

import (
	"context"
	"fmt"

	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/domain/orders/repositories"
	"erpgo/pkg/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PostgresOrderItemRepository implements OrderItemRepository for PostgreSQL
type PostgresOrderItemRepository struct {
	db *database.Database
}

// NewPostgresOrderItemRepository creates a new PostgreSQL order item repository
func NewPostgresOrderItemRepository(db *database.Database) *PostgresOrderItemRepository {
	return &PostgresOrderItemRepository{
		db: db,
	}
}

// Create creates a new order item
func (r *PostgresOrderItemRepository) Create(ctx context.Context, item *entities.OrderItem) error {
	query := `
		INSERT INTO order_items (
			id, order_id, product_id, product_sku, product_name, quantity,
			unit_price, discount_amount, tax_rate, tax_amount, total_price,
			weight, dimensions, barcode, notes, status, quantity_shipped,
			quantity_returned
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18
		)
	`

	_, err := r.db.Exec(ctx, query,
		item.ID,
		item.OrderID,
		item.ProductID,
		item.ProductSKU,
		item.ProductName,
		item.Quantity,
		item.UnitPrice,
		item.DiscountAmount,
		item.TaxRate,
		item.TaxAmount,
		item.TotalPrice,
		item.Weight,
		item.Dimensions,
		item.Barcode,
		item.Notes,
		item.Status,
		item.QuantityShipped,
		item.QuantityReturned,
	)

	if err != nil {
		return fmt.Errorf("failed to create order item: %w", err)
	}

	return nil
}

// GetByID retrieves an order item by ID
func (r *PostgresOrderItemRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.OrderItem, error) {
	query := `
		SELECT
			id, order_id, product_id, product_sku, product_name, quantity,
			unit_price, discount_amount, tax_rate, tax_amount, total_price,
			weight, dimensions, barcode, notes, status, quantity_shipped,
			quantity_returned, created_at, updated_at
		FROM order_items
		WHERE id = $1
	`

	item := &entities.OrderItem{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.OrderID,
		&item.ProductID,
		&item.ProductSKU,
		&item.ProductName,
		&item.Quantity,
		&item.UnitPrice,
		&item.DiscountAmount,
		&item.TaxRate,
		&item.TaxAmount,
		&item.TotalPrice,
		&item.Weight,
		&item.Dimensions,
		&item.Barcode,
		&item.Notes,
		&item.Status,
		&item.QuantityShipped,
		&item.QuantityReturned,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("order item with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get order item by id: %w", err)
	}

	return item, nil
}

// Update updates an order item
func (r *PostgresOrderItemRepository) Update(ctx context.Context, item *entities.OrderItem) error {
	query := `
		UPDATE order_items SET
			product_id = $2, product_sku = $3, product_name = $4, quantity = $5,
			unit_price = $6, discount_amount = $7, tax_rate = $8, tax_amount = $9,
			total_price = $10, weight = $11, dimensions = $12, barcode = $13,
			notes = $14, status = $15, quantity_shipped = $16,
			quantity_returned = $17, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		item.ID,
		item.ProductID,
		item.ProductSKU,
		item.ProductName,
		item.Quantity,
		item.UnitPrice,
		item.DiscountAmount,
		item.TaxRate,
		item.TaxAmount,
		item.TotalPrice,
		item.Weight,
		item.Dimensions,
		item.Barcode,
		item.Notes,
		item.Status,
		item.QuantityShipped,
		item.QuantityReturned,
	)

	if err != nil {
		return fmt.Errorf("failed to update order item: %w", err)
	}

	return nil
}

// Delete deletes an order item
func (r *PostgresOrderItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM order_items WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete order item: %w", err)
	}

	return nil
}

// GetByOrderID retrieves order items for a specific order
func (r *PostgresOrderItemRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entities.OrderItem, error) {
	query := `
		SELECT
			id, order_id, product_id, product_sku, product_name, quantity,
			unit_price, discount_amount, tax_rate, tax_amount, total_price,
			weight, dimensions, barcode, notes, status, quantity_shipped,
			quantity_returned, created_at, updated_at
		FROM order_items
		WHERE order_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items by order id: %w", err)
	}
	defer rows.Close()

	var items []*entities.OrderItem
	for rows.Next() {
		item := &entities.OrderItem{}
		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.ProductSKU,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.DiscountAmount,
			&item.TaxRate,
			&item.TaxAmount,
			&item.TotalPrice,
			&item.Weight,
			&item.Dimensions,
			&item.Barcode,
			&item.Notes,
			&item.Status,
			&item.QuantityShipped,
			&item.QuantityReturned,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item row: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order item rows: %w", err)
	}

	return items, nil
}

// GetByProductID retrieves order items for a specific product
func (r *PostgresOrderItemRepository) GetByProductID(ctx context.Context, productID uuid.UUID) ([]*entities.OrderItem, error) {
	query := `
		SELECT
			id, order_id, product_id, product_sku, product_name, quantity,
			unit_price, discount_amount, tax_rate, tax_amount, total_price,
			weight, dimensions, barcode, notes, status, quantity_shipped,
			quantity_returned, created_at, updated_at
		FROM order_items
		WHERE product_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items by product id: %w", err)
	}
	defer rows.Close()

	var items []*entities.OrderItem
	for rows.Next() {
		item := &entities.OrderItem{}
		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.ProductSKU,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.DiscountAmount,
			&item.TaxRate,
			&item.TaxAmount,
			&item.TotalPrice,
			&item.Weight,
			&item.Dimensions,
			&item.Barcode,
			&item.Notes,
			&item.Status,
			&item.QuantityShipped,
			&item.QuantityReturned,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item row: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order item rows: %w", err)
	}

	return items, nil
}

// GetByOrderAndProduct retrieves a specific order item
func (r *PostgresOrderItemRepository) GetByOrderAndProduct(ctx context.Context, orderID, productID uuid.UUID) (*entities.OrderItem, error) {
	query := `
		SELECT
			id, order_id, product_id, product_sku, product_name, quantity,
			unit_price, discount_amount, tax_rate, tax_amount, total_price,
			weight, dimensions, barcode, notes, status, quantity_shipped,
			quantity_returned, created_at, updated_at
		FROM order_items
		WHERE order_id = $1 AND product_id = $2
		LIMIT 1
	`

	item := &entities.OrderItem{}
	err := r.db.QueryRow(ctx, query, orderID, productID).Scan(
		&item.ID,
		&item.OrderID,
		&item.ProductID,
		&item.ProductSKU,
		&item.ProductName,
		&item.Quantity,
		&item.UnitPrice,
		&item.DiscountAmount,
		&item.TaxRate,
		&item.TaxAmount,
		&item.TotalPrice,
		&item.Weight,
		&item.Dimensions,
		&item.Barcode,
		&item.Notes,
		&item.Status,
		&item.QuantityShipped,
		&item.QuantityReturned,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("order item for order %s and product %s not found", orderID, productID)
		}
		return nil, fmt.Errorf("failed to get order item by order and product: %w", err)
	}

	return item, nil
}

// UpdateItemStatus updates the status of an order item
func (r *PostgresOrderItemRepository) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status string) error {
	query := `
		UPDATE order_items
		SET status = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, itemID, status)
	if err != nil {
		return fmt.Errorf("failed to update order item status: %w", err)
	}

	return nil
}

// UpdateShippedQuantity updates the shipped quantity of an order item
func (r *PostgresOrderItemRepository) UpdateShippedQuantity(ctx context.Context, itemID uuid.UUID, quantity int) error {
	query := `
		UPDATE order_items
		SET quantity_shipped = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, itemID, quantity)
	if err != nil {
		return fmt.Errorf("failed to update shipped quantity: %w", err)
	}

	return nil
}

// UpdateReturnedQuantity updates the returned quantity of an order item
func (r *PostgresOrderItemRepository) UpdateReturnedQuantity(ctx context.Context, itemID uuid.UUID, quantity int) error {
	query := `
		UPDATE order_items
		SET quantity_returned = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, itemID, quantity)
	if err != nil {
		return fmt.Errorf("failed to update returned quantity: %w", err)
	}

	return nil
}

// BulkCreate creates multiple order items
func (r *PostgresOrderItemRepository) BulkCreate(ctx context.Context, items []*entities.OrderItem) error {
	if len(items) == 0 {
		return nil
	}

	// For bulk operations, we'll use a transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO order_items (
			id, order_id, product_id, product_sku, product_name, quantity,
			unit_price, discount_amount, tax_rate, tax_amount, total_price,
			weight, dimensions, barcode, notes, status, quantity_shipped,
			quantity_returned
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18
		)
	`

	for _, item := range items {
		_, err := tx.Exec(ctx, query,
			item.ID,
			item.OrderID,
			item.ProductID,
			item.ProductSKU,
			item.ProductName,
			item.Quantity,
			item.UnitPrice,
			item.DiscountAmount,
			item.TaxRate,
			item.TaxAmount,
			item.TotalPrice,
			item.Weight,
			item.Dimensions,
			item.Barcode,
			item.Notes,
			item.Status,
			item.QuantityShipped,
			item.QuantityReturned,
		)
		if err != nil {
			return fmt.Errorf("failed to bulk create order item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BulkUpdate updates multiple order items
func (r *PostgresOrderItemRepository) BulkUpdate(ctx context.Context, items []*entities.OrderItem) error {
	if len(items) == 0 {
		return nil
	}

	// For bulk operations, we'll use a transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE order_items SET
			product_id = $2, product_sku = $3, product_name = $4, quantity = $5,
			unit_price = $6, discount_amount = $7, tax_rate = $8, tax_amount = $9,
			total_price = $10, weight = $11, dimensions = $12, barcode = $13,
			notes = $14, status = $15, quantity_shipped = $16,
			quantity_returned = $17, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	for _, item := range items {
		_, err := tx.Exec(ctx, query,
			item.ID,
			item.ProductID,
			item.ProductSKU,
			item.ProductName,
			item.Quantity,
			item.UnitPrice,
			item.DiscountAmount,
			item.TaxRate,
			item.TaxAmount,
			item.TotalPrice,
			item.Weight,
			item.Dimensions,
			item.Barcode,
			item.Notes,
			item.Status,
			item.QuantityShipped,
			item.QuantityReturned,
		)
		if err != nil {
			return fmt.Errorf("failed to bulk update order item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteByOrderID deletes all order items for an order
func (r *PostgresOrderItemRepository) DeleteByOrderID(ctx context.Context, orderID uuid.UUID) error {
	query := `DELETE FROM order_items WHERE order_id = $1`

	_, err := r.db.Exec(ctx, query, orderID)
	if err != nil {
		return fmt.Errorf("failed to delete order items by order id: %w", err)
	}

	return nil
}

// GetProductOrderHistory retrieves product order history
func (r *PostgresOrderItemRepository) GetProductOrderHistory(ctx context.Context, productID uuid.UUID, limit int) ([]*entities.OrderItem, error) {
	query := `
		SELECT
			oi.id, oi.order_id, oi.product_id, oi.product_sku, oi.product_name,
			oi.quantity, oi.unit_price, oi.discount_amount, oi.tax_rate,
			oi.tax_amount, oi.total_price, oi.weight, oi.dimensions, oi.barcode,
			oi.notes, oi.status, oi.quantity_shipped, oi.quantity_returned,
			oi.created_at, oi.updated_at
		FROM order_items oi
		INNER JOIN orders o ON oi.order_id = o.id
		WHERE oi.product_id = $1
		ORDER BY o.order_date DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, productID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get product order history: %w", err)
	}
	defer rows.Close()

	var items []*entities.OrderItem
	for rows.Next() {
		item := &entities.OrderItem{}
		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.ProductSKU,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.DiscountAmount,
			&item.TaxRate,
			&item.TaxAmount,
			&item.TotalPrice,
			&item.Weight,
			&item.Dimensions,
			&item.Barcode,
			&item.Notes,
			&item.Status,
			&item.QuantityShipped,
			&item.QuantityReturned,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product order history row: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product order history rows: %w", err)
	}

	return items, nil
}

// GetItemsByStatus retrieves order items by status
func (r *PostgresOrderItemRepository) GetItemsByStatus(ctx context.Context, status string) ([]*entities.OrderItem, error) {
	query := `
		SELECT
			id, order_id, product_id, product_sku, product_name, quantity,
			unit_price, discount_amount, tax_rate, tax_amount, total_price,
			weight, dimensions, barcode, notes, status, quantity_shipped,
			quantity_returned, created_at, updated_at
		FROM order_items
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT 1000
	`

	rows, err := r.db.Query(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items by status: %w", err)
	}
	defer rows.Close()

	var items []*entities.OrderItem
	for rows.Next() {
		item := &entities.OrderItem{}
		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.ProductSKU,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.DiscountAmount,
			&item.TaxRate,
			&item.TaxAmount,
			&item.TotalPrice,
			&item.Weight,
			&item.Dimensions,
			&item.Barcode,
			&item.Notes,
			&item.Status,
			&item.QuantityShipped,
			&item.QuantityReturned,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item status row: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order item status rows: %w", err)
	}

	return items, nil
}

// GetLowStockItems retrieves products with low stock
func (r *PostgresOrderItemRepository) GetLowStockItems(ctx context.Context, threshold int) ([]*repositories.ProductLowStock, error) {
	query := `
		WITH product_inventory AS (
			SELECT
				p.id as product_id,
				p.sku as product_sku,
				p.name as product_name,
				COALESCE(SUM(i.quantity_available), 0) as current_stock,
				p.reorder_level
			FROM products p
			LEFT JOIN inventory i ON p.id = i.product_id
			GROUP BY p.id, p.sku, p.name, p.reorder_level
		),
		pending_orders AS (
			SELECT
				oi.product_id,
				SUM(oi.quantity - COALESCE(oi.quantity_shipped, 0)) as pending_quantity
			FROM order_items oi
			INNER JOIN orders o ON oi.order_id = o.id
			WHERE oi.status IN ('ORDERED', 'PROCESSING')
			AND o.status NOT IN ('CANCELLED', 'DELIVERED', 'REFUNDED')
			GROUP BY oi.product_id
		)
		SELECT
			pi.product_id,
			pi.product_sku,
			pi.product_name,
			pi.current_stock,
			pi.reorder_level,
			COALESCE(po.pending_quantity, 0) as pending_orders
		FROM product_inventory pi
		LEFT JOIN pending_orders po ON pi.product_id = po.product_id
		WHERE pi.current_stock <= pi.reorder_level
		OR pi.current_stock - COALESCE(po.pending_quantity, 0) <= $1
		ORDER BY (pi.current_stock - COALESCE(po.pending_quantity, 0)) ASC
	`

	rows, err := r.db.Query(ctx, query, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get low stock items: %w", err)
	}
	defer rows.Close()

	var items []*repositories.ProductLowStock
	for rows.Next() {
		item := &repositories.ProductLowStock{}
		err := rows.Scan(
			&item.ProductID,
			&item.ProductSKU,
			&item.ProductName,
			&item.CurrentStock,
			&item.ReorderLevel,
			&item.PendingOrders,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan low stock item row: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating low stock item rows: %w", err)
	}

	return items, nil
}
