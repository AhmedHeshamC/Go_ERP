package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"erpgo/internal/domain/inventory/entities"
	"erpgo/internal/domain/inventory/repositories"
	"erpgo/pkg/database"
)

// PostgresWarehouseRepository implements WarehouseRepository for PostgreSQL
type PostgresWarehouseRepository struct {
	db     *database.Database
	logger interface{} // Can be zerolog.Logger or any logger
}

// NewPostgresWarehouseRepository creates a new PostgreSQL warehouse repository
func NewPostgresWarehouseRepository(db *database.Database) *PostgresWarehouseRepository {
	return &PostgresWarehouseRepository{
		db: db,
	}
}

// Create creates a new warehouse
func (r *PostgresWarehouseRepository) Create(ctx context.Context, warehouse *entities.Warehouse) error {
	query := `
		INSERT INTO warehouses (id, name, code, address, city, state, country, postal_code,
		                        phone, email, manager_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.Exec(ctx, query,
		warehouse.ID,
		warehouse.Name,
		warehouse.Code,
		warehouse.Address,
		warehouse.City,
		warehouse.State,
		warehouse.Country,
		warehouse.PostalCode,
		warehouse.Phone,
		warehouse.Email,
		warehouse.ManagerID,
		warehouse.IsActive,
		warehouse.CreatedAt,
		warehouse.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create warehouse: %w", err)
	}

	return nil
}

// GetByID retrieves a warehouse by ID
func (r *PostgresWarehouseRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Warehouse, error) {
	query := `
		SELECT id, name, code, address, city, state, country, postal_code,
		       phone, email, manager_id, is_active, created_at, updated_at
		FROM warehouses
		WHERE id = $1
	`

	warehouse := &entities.Warehouse{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&warehouse.ID,
		&warehouse.Name,
		&warehouse.Code,
		&warehouse.Address,
		&warehouse.City,
		&warehouse.State,
		&warehouse.Country,
		&warehouse.PostalCode,
		&warehouse.Phone,
		&warehouse.Email,
		&warehouse.ManagerID,
		&warehouse.IsActive,
		&warehouse.CreatedAt,
		&warehouse.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("warehouse with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to get warehouse: %w", err)
	}

	return warehouse, nil
}

// GetByCode retrieves a warehouse by code
func (r *PostgresWarehouseRepository) GetByCode(ctx context.Context, code string) (*entities.Warehouse, error) {
	query := `
		SELECT id, name, code, address, city, state, country, postal_code,
		       phone, email, manager_id, is_active, created_at, updated_at
		FROM warehouses
		WHERE code = $1
	`

	warehouse := &entities.Warehouse{}
	err := r.db.QueryRow(ctx, query, code).Scan(
		&warehouse.ID,
		&warehouse.Name,
		&warehouse.Code,
		&warehouse.Address,
		&warehouse.City,
		&warehouse.State,
		&warehouse.Country,
		&warehouse.PostalCode,
		&warehouse.Phone,
		&warehouse.Email,
		&warehouse.ManagerID,
		&warehouse.IsActive,
		&warehouse.CreatedAt,
		&warehouse.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("warehouse with code %s not found", code)
		}
		return nil, fmt.Errorf("failed to get warehouse: %w", err)
	}

	return warehouse, nil
}

// Update updates a warehouse
func (r *PostgresWarehouseRepository) Update(ctx context.Context, warehouse *entities.Warehouse) error {
	query := `
		UPDATE warehouses
		SET name = $2, code = $3, address = $4, city = $5, state = $6, country = $7,
		    postal_code = $8, phone = $9, email = $10, manager_id = $11, is_active = $12,
		    updated_at = $13
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		warehouse.ID,
		warehouse.Name,
		warehouse.Code,
		warehouse.Address,
		warehouse.City,
		warehouse.State,
		warehouse.Country,
		warehouse.PostalCode,
		warehouse.Phone,
		warehouse.Email,
		warehouse.ManagerID,
		warehouse.IsActive,
		warehouse.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update warehouse: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("warehouse with ID %s not found", warehouse.ID)
	}

	return nil
}

// Delete deletes a warehouse
func (r *PostgresWarehouseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM warehouses WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete warehouse: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("warehouse with ID %s not found", id)
	}

	return nil
}

// List retrieves warehouses with filtering
func (r *PostgresWarehouseRepository) List(ctx context.Context, filter *repositories.WarehouseFilter) ([]*entities.Warehouse, error) {
	query := `
		SELECT id, name, code, address, city, state, country, postal_code,
		       phone, email, manager_id, is_active, created_at, updated_at
		FROM warehouses
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
			query += fmt.Sprintf(" AND id IN (%s)", strings.Join(placeholders, ","))
		}

		if filter.Code != "" {
			query += fmt.Sprintf(" AND code ILIKE $%d", argIndex)
			args = append(args, "%"+filter.Code+"%")
			argIndex++
		}

		if filter.Name != "" {
			query += fmt.Sprintf(" AND name ILIKE $%d", argIndex)
			args = append(args, "%"+filter.Name+"%")
			argIndex++
		}

		if filter.IsActive != nil {
			query += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, *filter.IsActive)
			argIndex++
		}

		if filter.ManagerID != nil {
			query += fmt.Sprintf(" AND manager_id = $%d", argIndex)
			args = append(args, *filter.ManagerID)
			argIndex++
		}

		if filter.City != "" {
			query += fmt.Sprintf(" AND city ILIKE $%d", argIndex)
			args = append(args, "%"+filter.City+"%")
			argIndex++
		}

		if filter.State != "" {
			query += fmt.Sprintf(" AND state ILIKE $%d", argIndex)
			args = append(args, "%"+filter.State+"%")
			argIndex++
		}

		if filter.Country != "" {
			query += fmt.Sprintf(" AND country ILIKE $%d", argIndex)
			args = append(args, "%"+filter.Country+"%")
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

		if filter.UpdatedAfter != nil {
			query += fmt.Sprintf(" AND updated_at >= $%d", argIndex)
			args = append(args, *filter.UpdatedAfter)
			argIndex++
		}

		if filter.UpdatedBefore != nil {
			query += fmt.Sprintf(" AND updated_at <= $%d", argIndex)
			args = append(args, *filter.UpdatedBefore)
			argIndex++
		}

		// Add ordering
		orderBy := "name"
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
		query += " ORDER BY name ASC"
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query warehouses: %w", err)
	}
	defer rows.Close()

	var warehouses []*entities.Warehouse
	for rows.Next() {
		warehouse := &entities.Warehouse{}
		err := rows.Scan(
			&warehouse.ID,
			&warehouse.Name,
			&warehouse.Code,
			&warehouse.Address,
			&warehouse.City,
			&warehouse.State,
			&warehouse.Country,
			&warehouse.PostalCode,
			&warehouse.Phone,
			&warehouse.Email,
			&warehouse.ManagerID,
			&warehouse.IsActive,
			&warehouse.CreatedAt,
			&warehouse.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan warehouse row: %w", err)
		}
		warehouses = append(warehouses, warehouse)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating warehouse rows: %w", err)
	}

	return warehouses, nil
}

// GetActive retrieves all active warehouses
func (r *PostgresWarehouseRepository) GetActive(ctx context.Context) ([]*entities.Warehouse, error) {
	query := `
		SELECT id, name, code, address, city, state, country, postal_code,
		       phone, email, manager_id, is_active, created_at, updated_at
		FROM warehouses
		WHERE is_active = true
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active warehouses: %w", err)
	}
	defer rows.Close()

	var warehouses []*entities.Warehouse
	for rows.Next() {
		warehouse := &entities.Warehouse{}
		err := rows.Scan(
			&warehouse.ID,
			&warehouse.Name,
			&warehouse.Code,
			&warehouse.Address,
			&warehouse.City,
			&warehouse.State,
			&warehouse.Country,
			&warehouse.PostalCode,
			&warehouse.Phone,
			&warehouse.Email,
			&warehouse.ManagerID,
			&warehouse.IsActive,
			&warehouse.CreatedAt,
			&warehouse.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan warehouse row: %w", err)
		}
		warehouses = append(warehouses, warehouse)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating warehouse rows: %w", err)
	}

	return warehouses, nil
}

// GetByManager retrieves warehouses managed by a specific user
func (r *PostgresWarehouseRepository) GetByManager(ctx context.Context, managerID uuid.UUID) ([]*entities.Warehouse, error) {
	query := `
		SELECT id, name, code, address, city, state, country, postal_code,
		       phone, email, manager_id, is_active, created_at, updated_at
		FROM warehouses
		WHERE manager_id = $1
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query, managerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query warehouses by manager: %w", err)
	}
	defer rows.Close()

	var warehouses []*entities.Warehouse
	for rows.Next() {
		warehouse := &entities.Warehouse{}
		err := rows.Scan(
			&warehouse.ID,
			&warehouse.Name,
			&warehouse.Code,
			&warehouse.Address,
			&warehouse.City,
			&warehouse.State,
			&warehouse.Country,
			&warehouse.PostalCode,
			&warehouse.Phone,
			&warehouse.Email,
			&warehouse.ManagerID,
			&warehouse.IsActive,
			&warehouse.CreatedAt,
			&warehouse.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan warehouse row: %w", err)
		}
		warehouses = append(warehouses, warehouse)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating warehouse rows: %w", err)
	}

	return warehouses, nil
}

// GetByLocation retrieves warehouses by location
func (r *PostgresWarehouseRepository) GetByLocation(ctx context.Context, city, state, country string) ([]*entities.Warehouse, error) {
	query := `
		SELECT id, name, code, address, city, state, country, postal_code,
		       phone, email, manager_id, is_active, created_at, updated_at
		FROM warehouses
		WHERE ($1 = '' OR city ILIKE $1)
		  AND ($2 = '' OR state ILIKE $2)
		  AND ($3 = '' OR country ILIKE $3)
		ORDER BY name ASC
	`

	args := []interface{}{}
	if city != "" {
		args = append(args, "%"+city+"%")
	} else {
		args = append(args, "")
	}
	if state != "" {
		args = append(args, "%"+state+"%")
	} else {
		args = append(args, "")
	}
	if country != "" {
		args = append(args, "%"+country+"%")
	} else {
		args = append(args, "")
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query warehouses by location: %w", err)
	}
	defer rows.Close()

	var warehouses []*entities.Warehouse
	for rows.Next() {
		warehouse := &entities.Warehouse{}
		err := rows.Scan(
			&warehouse.ID,
			&warehouse.Name,
			&warehouse.Code,
			&warehouse.Address,
			&warehouse.City,
			&warehouse.State,
			&warehouse.Country,
			&warehouse.PostalCode,
			&warehouse.Phone,
			&warehouse.Email,
			&warehouse.ManagerID,
			&warehouse.IsActive,
			&warehouse.CreatedAt,
			&warehouse.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan warehouse row: %w", err)
		}
		warehouses = append(warehouses, warehouse)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating warehouse rows: %w", err)
	}

	return warehouses, nil
}

// Search searches warehouses by name or code
func (r *PostgresWarehouseRepository) Search(ctx context.Context, query string, limit int) ([]*entities.Warehouse, error) {
	sqlQuery := `
		SELECT id, name, code, address, city, state, country, postal_code,
		       phone, email, manager_id, is_active, created_at, updated_at
		FROM warehouses
		WHERE name ILIKE $1 OR code ILIKE $1 OR city ILIKE $1
		ORDER BY name ASC
	`

	args := []interface{}{"%" + query + "%"}

	if limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", 2)
		args = append(args, limit)
	}

	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search warehouses: %w", err)
	}
	defer rows.Close()

	var warehouses []*entities.Warehouse
	for rows.Next() {
		warehouse := &entities.Warehouse{}
		err := rows.Scan(
			&warehouse.ID,
			&warehouse.Name,
			&warehouse.Code,
			&warehouse.Address,
			&warehouse.City,
			&warehouse.State,
			&warehouse.Country,
			&warehouse.PostalCode,
			&warehouse.Phone,
			&warehouse.Email,
			&warehouse.ManagerID,
			&warehouse.IsActive,
			&warehouse.CreatedAt,
			&warehouse.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan warehouse row: %w", err)
		}
		warehouses = append(warehouses, warehouse)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating warehouse rows: %w", err)
	}

	return warehouses, nil
}

// Count counts warehouses with filtering
func (r *PostgresWarehouseRepository) Count(ctx context.Context, filter *repositories.WarehouseFilter) (int, error) {
	query := `SELECT COUNT(*) FROM warehouses WHERE 1=1`

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
			query += fmt.Sprintf(" AND id IN (%s)", strings.Join(placeholders, ","))
		}

		if filter.Code != "" {
			query += fmt.Sprintf(" AND code ILIKE $%d", argIndex)
			args = append(args, "%"+filter.Code+"%")
			argIndex++
		}

		if filter.Name != "" {
			query += fmt.Sprintf(" AND name ILIKE $%d", argIndex)
			args = append(args, "%"+filter.Name+"%")
			argIndex++
		}

		if filter.IsActive != nil {
			query += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, *filter.IsActive)
			argIndex++
		}

		if filter.ManagerID != nil {
			query += fmt.Sprintf(" AND manager_id = $%d", argIndex)
			args = append(args, *filter.ManagerID)
			argIndex++
		}

		if filter.City != "" {
			query += fmt.Sprintf(" AND city ILIKE $%d", argIndex)
			args = append(args, "%"+filter.City+"%")
			argIndex++
		}

		if filter.State != "" {
			query += fmt.Sprintf(" AND state ILIKE $%d", argIndex)
			args = append(args, "%"+filter.State+"%")
			argIndex++
		}

		if filter.Country != "" {
			query += fmt.Sprintf(" AND country ILIKE $%d", argIndex)
			args = append(args, "%"+filter.Country+"%")
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

		if filter.UpdatedAfter != nil {
			query += fmt.Sprintf(" AND updated_at >= $%d", argIndex)
			args = append(args, *filter.UpdatedAfter)
			argIndex++
		}

		if filter.UpdatedBefore != nil {
			query += fmt.Sprintf(" AND updated_at <= $%d", argIndex)
			args = append(args, *filter.UpdatedBefore)
			argIndex++
		}
	}

	var count int
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count warehouses: %w", err)
	}

	return count, nil
}

// ExistsByID checks if a warehouse exists by ID
func (r *PostgresWarehouseRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM warehouses WHERE id = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check warehouse existence: %w", err)
	}

	return exists, nil
}

// ExistsByCode checks if a warehouse exists by code
func (r *PostgresWarehouseRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM warehouses WHERE code = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, code).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check warehouse code existence: %w", err)
	}

	return exists, nil
}

// BulkUpdateStatus updates the status of multiple warehouses
func (r *PostgresWarehouseRepository) BulkUpdateStatus(ctx context.Context, warehouseIDs []uuid.UUID, isActive bool) error {
	if len(warehouseIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(warehouseIDs))
	args := make([]interface{}, len(warehouseIDs)+1)
	args[0] = isActive

	for i, id := range warehouseIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		UPDATE warehouses
		SET is_active = $1, updated_at = NOW()
		WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk update warehouse status: %w", err)
	}

	return nil
}

// BulkAssignManager assigns a manager to multiple warehouses
func (r *PostgresWarehouseRepository) BulkAssignManager(ctx context.Context, warehouseIDs []uuid.UUID, managerID uuid.UUID) error {
	if len(warehouseIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(warehouseIDs))
	args := make([]interface{}, len(warehouseIDs)+1)
	args[0] = managerID

	for i, id := range warehouseIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		UPDATE warehouses
		SET manager_id = $1, updated_at = NOW()
		WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk assign warehouse manager: %w", err)
	}

	return nil
}

// GetWarehouseStats retrieves statistics for a specific warehouse
func (r *PostgresWarehouseRepository) GetWarehouseStats(ctx context.Context, warehouseID uuid.UUID) (*repositories.WarehouseStats, error) {
	query := `
		SELECT
			w.id,
			w.name,
			w.code,
			COALESCE(COUNT(DISTINCT i.product_id), 0) as total_products,
			COALESCE(SUM(i.quantity_on_hand), 0) as total_quantity,
			COALESCE(SUM(i.quantity_on_hand * i.average_cost), 0) as total_value,
			COALESCE(COUNT(CASE WHEN i.quantity_on_hand <= i.reorder_level THEN 1 END), 0) as low_stock_products,
			COALESCE(COUNT(CASE WHEN i.quantity_on_hand = 0 THEN 1 END), 0) as out_of_stock_products,
			GREATEST(w.updated_at, COALESCE(MAX(i.updated_at), w.updated_at)) as last_updated
		FROM warehouses w
		LEFT JOIN inventory i ON w.id = i.warehouse_id
		WHERE w.id = $1
		GROUP BY w.id, w.name, w.code, w.updated_at
	`

	stats := &repositories.WarehouseStats{}
	err := r.db.QueryRow(ctx, query, warehouseID).Scan(
		&stats.WarehouseID,
		&stats.WarehouseName,
		&stats.WarehouseCode,
		&stats.TotalProducts,
		&stats.TotalQuantity,
		&stats.TotalValue,
		&stats.LowStockProducts,
		&stats.OutOfStockProducts,
		&stats.LastUpdated,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("warehouse with ID %s not found", warehouseID)
		}
		return nil, fmt.Errorf("failed to get warehouse stats: %w", err)
	}

	return stats, nil
}

// GetAllWarehouseStats retrieves statistics for all warehouses
func (r *PostgresWarehouseRepository) GetAllWarehouseStats(ctx context.Context) ([]*repositories.WarehouseStats, error) {
	query := `
		SELECT
			w.id,
			w.name,
			w.code,
			COALESCE(COUNT(DISTINCT i.product_id), 0) as total_products,
			COALESCE(SUM(i.quantity_on_hand), 0) as total_quantity,
			COALESCE(SUM(i.quantity_on_hand * i.average_cost), 0) as total_value,
			COALESCE(COUNT(CASE WHEN i.quantity_on_hand <= i.reorder_level THEN 1 END), 0) as low_stock_products,
			COALESCE(COUNT(CASE WHEN i.quantity_on_hand = 0 THEN 1 END), 0) as out_of_stock_products,
			GREATEST(w.updated_at, COALESCE(MAX(i.updated_at), w.updated_at)) as last_updated
		FROM warehouses w
		LEFT JOIN inventory i ON w.id = i.warehouse_id
		GROUP BY w.id, w.name, w.code, w.updated_at
		ORDER BY w.name ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all warehouse stats: %w", err)
	}
	defer rows.Close()

	var allStats []*repositories.WarehouseStats
	for rows.Next() {
		stats := &repositories.WarehouseStats{}
		err := rows.Scan(
			&stats.WarehouseID,
			&stats.WarehouseName,
			&stats.WarehouseCode,
			&stats.TotalProducts,
			&stats.TotalQuantity,
			&stats.TotalValue,
			&stats.LowStockProducts,
			&stats.OutOfStockProducts,
			&stats.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan warehouse stats row: %w", err)
		}
		allStats = append(allStats, stats)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating warehouse stats rows: %w", err)
	}

	return allStats, nil
}

// GetCapacityUtilization retrieves capacity utilization for a warehouse
func (r *PostgresWarehouseRepository) GetCapacityUtilization(ctx context.Context, warehouseID uuid.UUID) (*repositories.CapacityUtilization, error) {
	query := `
		SELECT
			w.id,
			w.name,
			w.code,
			we.capacity,
			COALESCE(SUM(i.quantity_on_hand), 0) as current_stock,
			NOW() as last_calculated
		FROM warehouses w
		LEFT JOIN warehouse_extended we ON w.id = we.id
		LEFT JOIN inventory i ON w.id = i.warehouse_id
		WHERE w.id = $1
		GROUP BY w.id, w.name, w.code, we.capacity
	`

	utilization := &repositories.CapacityUtilization{}
	var capacity sql.NullInt32

	err := r.db.QueryRow(ctx, query, warehouseID).Scan(
		&utilization.WarehouseID,
		&utilization.WarehouseName,
		&utilization.WarehouseCode,
		&capacity,
		&utilization.CurrentStock,
		&utilization.LastCalculated,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("warehouse with ID %s not found", warehouseID)
		}
		return nil, fmt.Errorf("failed to get capacity utilization: %w", err)
	}

	if capacity.Valid {
		utilization.Capacity = new(int)
		*utilization.Capacity = int(capacity.Int32)

		if *utilization.Capacity > 0 {
			utilization.UtilizationPercent = float64(utilization.CurrentStock) / float64(*utilization.Capacity) * 100
			availableSpace := *utilization.Capacity - utilization.CurrentStock
			utilization.AvailableSpace = &availableSpace
		}
	}

	return utilization, nil
}

// Extended warehouse operations (for WarehouseExtended entities)
// These would require additional tables and can be implemented later
func (r *PostgresWarehouseRepository) CreateExtended(ctx context.Context, warehouse *entities.WarehouseExtended) error {
	// First create the base warehouse
	baseWarehouse := &warehouse.Warehouse
	if err := r.Create(ctx, baseWarehouse); err != nil {
		return fmt.Errorf("failed to create base warehouse: %w", err)
	}

	// Then create the extended record (would need warehouse_extended table)
	// TODO: Implement when warehouse_extended table is created
	return nil
}

func (r *PostgresWarehouseRepository) GetExtendedByID(ctx context.Context, id uuid.UUID) (*entities.WarehouseExtended, error) {
	// TODO: Implement when warehouse_extended table is created
	return nil, fmt.Errorf("extended warehouse operations not yet implemented")
}

func (r *PostgresWarehouseRepository) UpdateExtended(ctx context.Context, warehouse *entities.WarehouseExtended) error {
	// First update the base warehouse
	baseWarehouse := &warehouse.Warehouse
	if err := r.Update(ctx, baseWarehouse); err != nil {
		return fmt.Errorf("failed to update base warehouse: %w", err)
	}

	// Then update the extended record (would need warehouse_extended table)
	// TODO: Implement when warehouse_extended table is created
	return nil
}

func (r *PostgresWarehouseRepository) GetByType(ctx context.Context, warehouseType entities.WarehouseType) ([]*entities.WarehouseExtended, error) {
	// TODO: Implement when warehouse_extended table is created
	return nil, fmt.Errorf("extended warehouse operations not yet implemented")
}