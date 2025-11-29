package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"erpgo/internal/domain/products/entities"
	"erpgo/internal/domain/products/repositories"
	"erpgo/pkg/database"
	"erpgo/pkg/validation"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// PostgresProductRepository implements ProductRepository for PostgreSQL
type PostgresProductRepository struct {
	db              *database.Database
	columnWhitelist *validation.SQLColumnWhitelist
}

// NewPostgresProductRepository creates a new PostgreSQL product repository
func NewPostgresProductRepository(db *database.Database) *PostgresProductRepository {
	return &PostgresProductRepository{
		db:              db,
		columnWhitelist: validation.NewProductColumnWhitelist(),
	}
}

// validateAndBuildOrderBy validates sort parameters and builds ORDER BY clause
func (r *PostgresProductRepository) validateAndBuildOrderBy(sortBy, sortOrder string, defaultSort string) (string, error) {
	// Set default sort column
	if sortBy == "" {
		sortBy = defaultSort
	} else {
		// Validate column name against whitelist
		if err := r.columnWhitelist.ValidateColumn(sortBy); err != nil {
			return "", fmt.Errorf("invalid sort column: %w", err)
		}
	}

	// Set default sort order
	if sortOrder == "" {
		sortOrder = "DESC"
	} else {
		sortOrder = strings.ToUpper(sortOrder)
		// Validate sort order
		if sortOrder != "ASC" && sortOrder != "DESC" {
			return "", fmt.Errorf("invalid sort order: must be ASC or DESC")
		}
	}

	return fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder), nil
}

// Create creates a new product
func (r *PostgresProductRepository) Create(ctx context.Context, product *entities.Product) error {
	query := `
		INSERT INTO products (
			id, sku, name, description, short_description, category_id, price, cost,
			weight, dimensions, length, width, height, volume, barcode, track_inventory,
			stock_quantity, min_stock_level, max_stock_level, allow_backorder,
			requires_shipping, taxable, tax_rate, is_active, is_featured, is_digital,
			download_url, max_downloads, expiry_days, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31
		)
	`

	_, err := r.db.Exec(ctx, query,
		product.ID,
		product.SKU,
		product.Name,
		product.Description,
		product.ShortDescription,
		product.CategoryID,
		product.Price,
		product.Cost,
		product.Weight,
		product.Dimensions,
		product.Length,
		product.Width,
		product.Height,
		product.Volume,
		product.Barcode,
		product.TrackInventory,
		product.StockQuantity,
		product.MinStockLevel,
		product.MaxStockLevel,
		product.AllowBackorder,
		product.RequiresShipping,
		product.Taxable,
		product.TaxRate,
		product.IsActive,
		product.IsFeatured,
		product.IsDigital,
		product.DownloadURL,
		product.MaxDownloads,
		product.ExpiryDays,
		product.CreatedAt,
		product.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	return nil
}

// GetByID retrieves a product by ID
func (r *PostgresProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Product, error) {
	query := `
		SELECT id, sku, name, description, short_description, category_id, price, cost,
		       weight, dimensions, length, width, height, volume, barcode, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_featured, is_digital,
		       download_url, max_downloads, expiry_days, created_at, updated_at
		FROM products
		WHERE id = $1
	`

	product := &entities.Product{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Description,
		&product.ShortDescription,
		&product.CategoryID,
		&product.Price,
		&product.Cost,
		&product.Weight,
		&product.Dimensions,
		&product.Length,
		&product.Width,
		&product.Height,
		&product.Volume,
		&product.Barcode,
		&product.TrackInventory,
		&product.StockQuantity,
		&product.MinStockLevel,
		&product.MaxStockLevel,
		&product.AllowBackorder,
		&product.RequiresShipping,
		&product.Taxable,
		&product.TaxRate,
		&product.IsActive,
		&product.IsFeatured,
		&product.IsDigital,
		&product.DownloadURL,
		&product.MaxDownloads,
		&product.ExpiryDays,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get product by id: %w", err)
	}

	return product, nil
}

// GetBySKU retrieves a product by SKU
func (r *PostgresProductRepository) GetBySKU(ctx context.Context, sku string) (*entities.Product, error) {
	query := `
		SELECT id, sku, name, description, short_description, category_id, price, cost,
		       weight, dimensions, length, width, height, volume, barcode, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_featured, is_digital,
		       download_url, max_downloads, expiry_days, created_at, updated_at
		FROM products
		WHERE sku = $1
	`

	product := &entities.Product{}
	err := r.db.QueryRow(ctx, query, sku).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Description,
		&product.ShortDescription,
		&product.CategoryID,
		&product.Price,
		&product.Cost,
		&product.Weight,
		&product.Dimensions,
		&product.Length,
		&product.Width,
		&product.Height,
		&product.Volume,
		&product.Barcode,
		&product.TrackInventory,
		&product.StockQuantity,
		&product.MinStockLevel,
		&product.MaxStockLevel,
		&product.AllowBackorder,
		&product.RequiresShipping,
		&product.Taxable,
		&product.TaxRate,
		&product.IsActive,
		&product.IsFeatured,
		&product.IsDigital,
		&product.DownloadURL,
		&product.MaxDownloads,
		&product.ExpiryDays,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product with sku %s not found", sku)
		}
		return nil, fmt.Errorf("failed to get product by sku: %w", err)
	}

	return product, nil
}

// Update updates a product
func (r *PostgresProductRepository) Update(ctx context.Context, product *entities.Product) error {
	query := `
		UPDATE products
		SET name = $2, description = $3, short_description = $4, category_id = $5,
		    price = $6, cost = $7, weight = $8, dimensions = $9, length = $10,
		    width = $11, height = $12, volume = $13, barcode = $14, track_inventory = $15,
		    stock_quantity = $16, min_stock_level = $17, max_stock_level = $18,
		    allow_backorder = $19, requires_shipping = $20, taxable = $21,
		    tax_rate = $22, is_active = $23, is_featured = $24, is_digital = $25,
		    download_url = $26, max_downloads = $27, expiry_days = $28, updated_at = $29
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		product.ID,
		product.Name,
		product.Description,
		product.ShortDescription,
		product.CategoryID,
		product.Price,
		product.Cost,
		product.Weight,
		product.Dimensions,
		product.Length,
		product.Width,
		product.Height,
		product.Volume,
		product.Barcode,
		product.TrackInventory,
		product.StockQuantity,
		product.MinStockLevel,
		product.MaxStockLevel,
		product.AllowBackorder,
		product.RequiresShipping,
		product.Taxable,
		product.TaxRate,
		product.IsActive,
		product.IsFeatured,
		product.IsDigital,
		product.DownloadURL,
		product.MaxDownloads,
		product.ExpiryDays,
		product.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	return nil
}

// Delete deletes a product
func (r *PostgresProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM products WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	return nil
}

// List retrieves a list of products
func (r *PostgresProductRepository) List(ctx context.Context, filter repositories.ProductFilter) ([]*entities.Product, error) {
	// Build the base query
	baseQuery := `
		SELECT id, sku, name, description, short_description, category_id, price, cost,
		       weight, dimensions, length, width, height, volume, barcode, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_featured, is_digital,
		       download_url, max_downloads, expiry_days, created_at, updated_at
		FROM products
		WHERE 1=1
	`

	// Build WHERE conditions
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add search filter
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d OR sku ILIKE $%d)", argIndex, argIndex+1, argIndex+2))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		argIndex += 3
	}

	// Add category filter
	if filter.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("category_id = $%d", argIndex))
		args = append(args, *filter.CategoryID)
		argIndex++
	}

	// Add multiple categories filter
	if len(filter.CategoryIDs) > 0 {
		placeholders := make([]string, len(filter.CategoryIDs))
		for i, id := range filter.CategoryIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex+i)
			args = append(args, id)
		}
		conditions = append(conditions, fmt.Sprintf("category_id IN (%s)", strings.Join(placeholders, ", ")))
		argIndex += len(filter.CategoryIDs)
	}

	// Add SKU filter
	if filter.SKU != "" {
		conditions = append(conditions, fmt.Sprintf("sku ILIKE $%d", argIndex))
		args = append(args, "%"+filter.SKU+"%")
		argIndex++
	}

	// Add price filters
	if filter.MinPrice != nil {
		conditions = append(conditions, fmt.Sprintf("price >= $%d", argIndex))
		args = append(args, *filter.MinPrice)
		argIndex++
	}

	if filter.MaxPrice != nil {
		conditions = append(conditions, fmt.Sprintf("price <= $%d", argIndex))
		args = append(args, *filter.MaxPrice)
		argIndex++
	}

	// Add boolean filters
	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.IsFeatured != nil {
		conditions = append(conditions, fmt.Sprintf("is_featured = $%d", argIndex))
		args = append(args, *filter.IsFeatured)
		argIndex++
	}

	if filter.IsDigital != nil {
		conditions = append(conditions, fmt.Sprintf("is_digital = $%d", argIndex))
		args = append(args, *filter.IsDigital)
		argIndex++
	}

	if filter.TrackInventory != nil {
		conditions = append(conditions, fmt.Sprintf("track_inventory = $%d", argIndex))
		args = append(args, *filter.TrackInventory)
		argIndex++
	}

	// Add stock filters
	if filter.InStock != nil {
		if *filter.InStock {
			conditions = append(conditions, "(NOT track_inventory OR stock_quantity > 0 OR allow_backorder = true)")
		} else {
			conditions = append(conditions, "(track_inventory = true AND stock_quantity <= 0 AND allow_backorder = false)")
		}
	}

	if filter.LowStock != nil && *filter.LowStock {
		conditions = append(conditions, "track_inventory = true AND stock_quantity <= min_stock_level AND stock_quantity > 0")
	}

	// Add date filters
	if filter.CreatedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.CreatedAfter)
		argIndex++
	}

	if filter.CreatedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.CreatedBefore)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY with validation
	orderByClause, err := r.validateAndBuildOrderBy(filter.SortBy, filter.SortOrder, "created_at")
	if err != nil {
		return nil, err
	}
	baseQuery += orderByClause

	// Add LIMIT and OFFSET for pagination
	if filter.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++

		if filter.Page > 1 {
			offset := (filter.Page - 1) * filter.Limit
			baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, offset)
		}
	}

	// Execute query
	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	var products []*entities.Product
	for rows.Next() {
		product := &entities.Product{}
		err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.Description,
			&product.ShortDescription,
			&product.CategoryID,
			&product.Price,
			&product.Cost,
			&product.Weight,
			&product.Dimensions,
			&product.Length,
			&product.Width,
			&product.Height,
			&product.Volume,
			&product.Barcode,
			&product.TrackInventory,
			&product.StockQuantity,
			&product.MinStockLevel,
			&product.MaxStockLevel,
			&product.AllowBackorder,
			&product.RequiresShipping,
			&product.Taxable,
			&product.TaxRate,
			&product.IsActive,
			&product.IsFeatured,
			&product.IsDigital,
			&product.DownloadURL,
			&product.MaxDownloads,
			&product.ExpiryDays,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	return products, nil
}

// Search searches for products by query string
func (r *PostgresProductRepository) Search(ctx context.Context, query string, limit int) ([]*entities.Product, error) {
	searchQuery := `
		SELECT id, sku, name, description, short_description, category_id, price, cost,
		       weight, dimensions, length, width, height, volume, barcode, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_featured, is_digital,
		       download_url, max_downloads, expiry_days, created_at, updated_at
		FROM products
		WHERE (
			name ILIKE $1 OR
			description ILIKE $1 OR
			short_description ILIKE $1 OR
			sku ILIKE $1 OR
			barcode ILIKE $1
		) AND is_active = true
		ORDER BY
			CASE WHEN name ILIKE $1 THEN 1 ELSE 2 END,
			CASE WHEN sku ILIKE $1 THEN 1 ELSE 2 END,
			name
		LIMIT $2
	`

	searchPattern := "%" + query + "%"
	rows, err := r.db.Query(ctx, searchQuery, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search products: %w", err)
	}
	defer rows.Close()

	var products []*entities.Product
	for rows.Next() {
		product := &entities.Product{}
		err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.Description,
			&product.ShortDescription,
			&product.CategoryID,
			&product.Price,
			&product.Cost,
			&product.Weight,
			&product.Dimensions,
			&product.Length,
			&product.Width,
			&product.Height,
			&product.Volume,
			&product.Barcode,
			&product.TrackInventory,
			&product.StockQuantity,
			&product.MinStockLevel,
			&product.MaxStockLevel,
			&product.AllowBackorder,
			&product.RequiresShipping,
			&product.Taxable,
			&product.TaxRate,
			&product.IsActive,
			&product.IsFeatured,
			&product.IsDigital,
			&product.DownloadURL,
			&product.MaxDownloads,
			&product.ExpiryDays,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	return products, nil
}

// Count returns the count of products matching the filter
func (r *PostgresProductRepository) Count(ctx context.Context, filter repositories.ProductFilter) (int, error) {
	baseQuery := `SELECT COUNT(*) FROM products WHERE 1=1`

	// Build WHERE conditions (same as in List)
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d OR sku ILIKE $%d)", argIndex, argIndex+1, argIndex+2))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		argIndex += 3
	}

	if filter.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("category_id = $%d", argIndex))
		args = append(args, *filter.CategoryID)
		argIndex++
	}

	if len(filter.CategoryIDs) > 0 {
		placeholders := make([]string, len(filter.CategoryIDs))
		for i, id := range filter.CategoryIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex+i)
			args = append(args, id)
		}
		conditions = append(conditions, fmt.Sprintf("category_id IN (%s)", strings.Join(placeholders, ", ")))
		argIndex += len(filter.CategoryIDs)
	}

	if filter.SKU != "" {
		conditions = append(conditions, fmt.Sprintf("sku ILIKE $%d", argIndex))
		args = append(args, "%"+filter.SKU+"%")
		argIndex++
	}

	if filter.MinPrice != nil {
		conditions = append(conditions, fmt.Sprintf("price >= $%d", argIndex))
		args = append(args, *filter.MinPrice)
		argIndex++
	}

	if filter.MaxPrice != nil {
		conditions = append(conditions, fmt.Sprintf("price <= $%d", argIndex))
		args = append(args, *filter.MaxPrice)
		argIndex++
	}

	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.IsFeatured != nil {
		conditions = append(conditions, fmt.Sprintf("is_featured = $%d", argIndex))
		args = append(args, *filter.IsFeatured)
		argIndex++
	}

	if filter.IsDigital != nil {
		conditions = append(conditions, fmt.Sprintf("is_digital = $%d", argIndex))
		args = append(args, *filter.IsDigital)
		argIndex++
	}

	if filter.TrackInventory != nil {
		conditions = append(conditions, fmt.Sprintf("track_inventory = $%d", argIndex))
		args = append(args, *filter.TrackInventory)
		argIndex++
	}

	if filter.InStock != nil {
		if *filter.InStock {
			conditions = append(conditions, "(NOT track_inventory OR stock_quantity > 0 OR allow_backorder = true)")
		} else {
			conditions = append(conditions, "(track_inventory = true AND stock_quantity <= 0 AND allow_backorder = false)")
		}
	}

	if filter.LowStock != nil && *filter.LowStock {
		conditions = append(conditions, "track_inventory = true AND stock_quantity <= min_stock_level AND stock_quantity > 0")
	}

	if filter.CreatedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.CreatedAfter)
		argIndex++
	}

	if filter.CreatedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.CreatedBefore)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	var count int
	err := r.db.QueryRow(ctx, baseQuery, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count products: %w", err)
	}

	return count, nil
}

// GetByCategory retrieves products by category ID
func (r *PostgresProductRepository) GetByCategory(ctx context.Context, categoryID uuid.UUID) ([]*entities.Product, error) {
	query := `
		SELECT id, sku, name, description, short_description, category_id, price, cost,
		       weight, dimensions, length, width, height, volume, barcode, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_featured, is_digital,
		       download_url, max_downloads, expiry_days, created_at, updated_at
		FROM products
		WHERE category_id = $1 AND is_active = true
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get products by category: %w", err)
	}
	defer rows.Close()

	var products []*entities.Product
	for rows.Next() {
		product := &entities.Product{}
		err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.Description,
			&product.ShortDescription,
			&product.CategoryID,
			&product.Price,
			&product.Cost,
			&product.Weight,
			&product.Dimensions,
			&product.Length,
			&product.Width,
			&product.Height,
			&product.Volume,
			&product.Barcode,
			&product.TrackInventory,
			&product.StockQuantity,
			&product.MinStockLevel,
			&product.MaxStockLevel,
			&product.AllowBackorder,
			&product.RequiresShipping,
			&product.Taxable,
			&product.TaxRate,
			&product.IsActive,
			&product.IsFeatured,
			&product.IsDigital,
			&product.DownloadURL,
			&product.MaxDownloads,
			&product.ExpiryDays,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	return products, nil
}

// GetByCategories retrieves products by multiple category IDs
func (r *PostgresProductRepository) GetByCategories(ctx context.Context, categoryIDs []uuid.UUID) ([]*entities.Product, error) {
	if len(categoryIDs) == 0 {
		return []*entities.Product{}, nil
	}

	placeholders := make([]string, len(categoryIDs))
	args := make([]interface{}, len(categoryIDs))
	for i, id := range categoryIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, sku, name, description, short_description, category_id, price, cost,
		       weight, dimensions, length, width, height, volume, barcode, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_featured, is_digital,
		       download_url, max_downloads, expiry_days, created_at, updated_at
		FROM products
		WHERE category_id IN (%s) AND is_active = true
		ORDER BY name
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get products by categories: %w", err)
	}
	defer rows.Close()

	var products []*entities.Product
	for rows.Next() {
		product := &entities.Product{}
		err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.Description,
			&product.ShortDescription,
			&product.CategoryID,
			&product.Price,
			&product.Cost,
			&product.Weight,
			&product.Dimensions,
			&product.Length,
			&product.Width,
			&product.Height,
			&product.Volume,
			&product.Barcode,
			&product.TrackInventory,
			&product.StockQuantity,
			&product.MinStockLevel,
			&product.MaxStockLevel,
			&product.AllowBackorder,
			&product.RequiresShipping,
			&product.Taxable,
			&product.TaxRate,
			&product.IsActive,
			&product.IsFeatured,
			&product.IsDigital,
			&product.DownloadURL,
			&product.MaxDownloads,
			&product.ExpiryDays,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	return products, nil
}

// GetFeatured retrieves featured products
func (r *PostgresProductRepository) GetFeatured(ctx context.Context, limit int) ([]*entities.Product, error) {
	query := `
		SELECT id, sku, name, description, short_description, category_id, price, cost,
		       weight, dimensions, length, width, height, volume, barcode, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_featured, is_digital,
		       download_url, max_downloads, expiry_days, created_at, updated_at
		FROM products
		WHERE is_featured = true AND is_active = true
		ORDER BY name
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get featured products: %w", err)
	}
	defer rows.Close()

	var products []*entities.Product
	for rows.Next() {
		product := &entities.Product{}
		err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.Description,
			&product.ShortDescription,
			&product.CategoryID,
			&product.Price,
			&product.Cost,
			&product.Weight,
			&product.Dimensions,
			&product.Length,
			&product.Width,
			&product.Height,
			&product.Volume,
			&product.Barcode,
			&product.TrackInventory,
			&product.StockQuantity,
			&product.MinStockLevel,
			&product.MaxStockLevel,
			&product.AllowBackorder,
			&product.RequiresShipping,
			&product.Taxable,
			&product.TaxRate,
			&product.IsActive,
			&product.IsFeatured,
			&product.IsDigital,
			&product.DownloadURL,
			&product.MaxDownloads,
			&product.ExpiryDays,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	return products, nil
}

// GetActive retrieves active products
func (r *PostgresProductRepository) GetActive(ctx context.Context, limit int) ([]*entities.Product, error) {
	query := `
		SELECT id, sku, name, description, short_description, category_id, price, cost,
		       weight, dimensions, length, width, height, volume, barcode, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_featured, is_digital,
		       download_url, max_downloads, expiry_days, created_at, updated_at
		FROM products
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get active products: %w", err)
	}
	defer rows.Close()

	var products []*entities.Product
	for rows.Next() {
		product := &entities.Product{}
		err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.Description,
			&product.ShortDescription,
			&product.CategoryID,
			&product.Price,
			&product.Cost,
			&product.Weight,
			&product.Dimensions,
			&product.Length,
			&product.Width,
			&product.Height,
			&product.Volume,
			&product.Barcode,
			&product.TrackInventory,
			&product.StockQuantity,
			&product.MinStockLevel,
			&product.MaxStockLevel,
			&product.AllowBackorder,
			&product.RequiresShipping,
			&product.Taxable,
			&product.TaxRate,
			&product.IsActive,
			&product.IsFeatured,
			&product.IsDigital,
			&product.DownloadURL,
			&product.MaxDownloads,
			&product.ExpiryDays,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	return products, nil
}

// GetLowStock retrieves products with low stock
func (r *PostgresProductRepository) GetLowStock(ctx context.Context, threshold int) ([]*entities.Product, error) {
	query := `
		SELECT id, sku, name, description, short_description, category_id, price, cost,
		       weight, dimensions, length, width, height, volume, barcode, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_featured, is_digital,
		       download_url, max_downloads, expiry_days, created_at, updated_at
		FROM products
		WHERE track_inventory = true
		  AND stock_quantity <= COALESCE(min_stock_level, $1)
		  AND stock_quantity > 0
		  AND is_active = true
		ORDER BY stock_quantity ASC
	`

	rows, err := r.db.Query(ctx, query, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get low stock products: %w", err)
	}
	defer rows.Close()

	var products []*entities.Product
	for rows.Next() {
		product := &entities.Product{}
		err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.Description,
			&product.ShortDescription,
			&product.CategoryID,
			&product.Price,
			&product.Cost,
			&product.Weight,
			&product.Dimensions,
			&product.Length,
			&product.Width,
			&product.Height,
			&product.Volume,
			&product.Barcode,
			&product.TrackInventory,
			&product.StockQuantity,
			&product.MinStockLevel,
			&product.MaxStockLevel,
			&product.AllowBackorder,
			&product.RequiresShipping,
			&product.Taxable,
			&product.TaxRate,
			&product.IsActive,
			&product.IsFeatured,
			&product.IsDigital,
			&product.DownloadURL,
			&product.MaxDownloads,
			&product.ExpiryDays,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	return products, nil
}

// ExistsBySKU checks if a product exists by SKU
func (r *PostgresProductRepository) ExistsBySKU(ctx context.Context, sku string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM products WHERE sku = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, sku).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if product exists by sku: %w", err)
	}

	return exists, nil
}

// ExistsByID checks if a product exists by ID
func (r *PostgresProductRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if product exists by id: %w", err)
	}

	return exists, nil
}

// UpdateStock updates the stock quantity for a product
func (r *PostgresProductRepository) UpdateStock(ctx context.Context, productID uuid.UUID, quantity int) error {
	query := `UPDATE products SET stock_quantity = $2, updated_at = $3 WHERE id = $1`

	_, err := r.db.Exec(ctx, query, productID, quantity, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update product stock: %w", err)
	}

	return nil
}

// AdjustStock adjusts the stock quantity for a product
func (r *PostgresProductRepository) AdjustStock(ctx context.Context, productID uuid.UUID, adjustment int) error {
	query := `
		UPDATE products
		SET stock_quantity = stock_quantity + $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, productID, adjustment, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to adjust product stock: %w", err)
	}

	return nil
}

// GetPrice gets the price for a product
func (r *PostgresProductRepository) GetPrice(ctx context.Context, productID uuid.UUID) (decimal.Decimal, error) {
	query := `SELECT price FROM products WHERE id = $1`

	var price decimal.Decimal
	err := r.db.QueryRow(ctx, query, productID).Scan(&price)
	if err != nil {
		if err == pgx.ErrNoRows {
			return decimal.Zero, fmt.Errorf("product with id %s not found", productID)
		}
		return decimal.Zero, fmt.Errorf("failed to get product price: %w", err)
	}

	return price, nil
}

// UpdatePrice updates the price for a product
func (r *PostgresProductRepository) UpdatePrice(ctx context.Context, productID uuid.UUID, price decimal.Decimal) error {
	query := `UPDATE products SET price = $2, updated_at = $3 WHERE id = $1`

	_, err := r.db.Exec(ctx, query, productID, price, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update product price: %w", err)
	}

	return nil
}

// BulkUpdateStatus updates the status for multiple products
func (r *PostgresProductRepository) BulkUpdateStatus(ctx context.Context, productIDs []uuid.UUID, isActive bool) error {
	if len(productIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(productIDs))
	args := make([]interface{}, len(productIDs))
	for i, id := range productIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2) // Start from $2 since $1 is the status
		args[i] = id
	}

	query := fmt.Sprintf(`
		UPDATE products
		SET is_active = $1, updated_at = $2
		WHERE id IN (%s)
	`, strings.Join(placeholders, ", "))

	// Add the status and timestamp to the beginning of args
	args = append([]interface{}{isActive, time.Now().UTC()}, args...)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk update product status: %w", err)
	}

	return nil
}

// GetProductStats retrieves product statistics
func (r *PostgresProductRepository) GetProductStats(ctx context.Context, filter repositories.ProductFilter) (*repositories.ProductStats, error) {
	// Build WHERE conditions for stats
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Apply category filters if present
	if filter.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("category_id = $%d", argIndex))
		args = append(args, *filter.CategoryID)
		argIndex++
	}

	if len(filter.CategoryIDs) > 0 {
		placeholders := make([]string, len(filter.CategoryIDs))
		for i, id := range filter.CategoryIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex+i)
			args = append(args, id)
		}
		conditions = append(conditions, fmt.Sprintf("category_id IN (%s)", strings.Join(placeholders, ", ")))
		argIndex += len(filter.CategoryIDs)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_products,
			COUNT(CASE WHEN is_active = true THEN 1 END) as active_products,
			COUNT(CASE WHEN is_active = false THEN 1 END) as inactive_products,
			COUNT(CASE WHEN is_featured = true THEN 1 END) as featured_products,
			COUNT(CASE WHEN track_inventory = true AND stock_quantity <= COALESCE(min_stock_level, 0) AND stock_quantity > 0 THEN 1 END) as low_stock_products,
			COUNT(CASE WHEN track_inventory = true AND stock_quantity <= 0 AND allow_backorder = false THEN 1 END) as out_of_stock_products,
			COUNT(CASE WHEN is_digital = true THEN 1 END) as digital_products,
			COUNT(CASE WHEN is_digital = false THEN 1 END) as physical_products,
			COALESCE(AVG(price), 0) as average_price,
			COALESCE(MIN(price), 0) as min_price,
			COALESCE(MAX(price), 0) as max_price,
			COALESCE(SUM(stock_quantity * cost), 0) as total_stock_value
		FROM products
		%s
	`, whereClause)

	stats := &repositories.ProductStats{}
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&stats.TotalProducts,
		&stats.ActiveProducts,
		&stats.InactiveProducts,
		&stats.FeaturedProducts,
		&stats.LowStockProducts,
		&stats.OutOfStockProducts,
		&stats.DigitalProducts,
		&stats.PhysicalProducts,
		&stats.AveragePrice,
		&stats.MinPrice,
		&stats.MaxPrice,
		&stats.TotalStockValue,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get product stats: %w", err)
	}

	return stats, nil
}
