package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"erpgo/internal/domain/products/entities"
	"erpgo/internal/domain/products/repositories"
	"erpgo/pkg/database"
)

// PostgresProductVariantRepository implements ProductVariantRepository for PostgreSQL
type PostgresProductVariantRepository struct {
	db *database.Database
}

// NewPostgresProductVariantRepository creates a new PostgreSQL product variant repository
func NewPostgresProductVariantRepository(db *database.Database) *PostgresProductVariantRepository {
	return &PostgresProductVariantRepository{
		db: db,
	}
}

// Create creates a new product variant
func (r *PostgresProductVariantRepository) Create(ctx context.Context, variant *entities.ProductVariant) error {
	query := `
		INSERT INTO product_variants (
			id, product_id, sku, name, price, cost, weight, dimensions,
			length, width, height, volume, barcode, image_url, track_inventory,
			stock_quantity, min_stock_level, max_stock_level, allow_backorder,
			requires_shipping, taxable, tax_rate, is_active, is_digital,
			download_url, max_downloads, expiry_days, sort_order, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29
		)
	`

	_, err := r.db.Exec(ctx, query,
		variant.ID,
		variant.ProductID,
		variant.SKU,
		variant.Name,
		variant.Price,
		variant.Cost,
		variant.Weight,
		variant.Dimensions,
		variant.Length,
		variant.Width,
		variant.Height,
		variant.Volume,
		variant.Barcode,
		variant.ImageURL,
		variant.TrackInventory,
		variant.StockQuantity,
		variant.MinStockLevel,
		variant.MaxStockLevel,
		variant.AllowBackorder,
		variant.RequiresShipping,
		variant.Taxable,
		variant.TaxRate,
		variant.IsActive,
		variant.IsDigital,
		variant.DownloadURL,
		variant.MaxDownloads,
		variant.ExpiryDays,
		variant.SortOrder,
		variant.CreatedAt,
		variant.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create product variant: %w", err)
	}

	return nil
}

// GetByID retrieves a product variant by ID
func (r *PostgresProductVariantRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.ProductVariant, error) {
	query := `
		SELECT id, product_id, sku, name, price, cost, weight, dimensions,
		       length, width, height, volume, barcode, image_url, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_digital,
		       download_url, max_downloads, expiry_days, sort_order, created_at, updated_at
		FROM product_variants
		WHERE id = $1
	`

	variant := &entities.ProductVariant{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&variant.ID,
		&variant.ProductID,
		&variant.SKU,
		&variant.Name,
		&variant.Price,
		&variant.Cost,
		&variant.Weight,
		&variant.Dimensions,
		&variant.Length,
		&variant.Width,
		&variant.Height,
		&variant.Volume,
		&variant.Barcode,
		&variant.ImageURL,
		&variant.TrackInventory,
		&variant.StockQuantity,
		&variant.MinStockLevel,
		&variant.MaxStockLevel,
		&variant.AllowBackorder,
		&variant.RequiresShipping,
		&variant.Taxable,
		&variant.TaxRate,
		&variant.IsActive,
		&variant.IsDigital,
		&variant.DownloadURL,
		&variant.MaxDownloads,
		&variant.ExpiryDays,
		&variant.SortOrder,
		&variant.CreatedAt,
		&variant.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product variant with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get product variant by id: %w", err)
	}

	return variant, nil
}

// GetBySKU retrieves a product variant by SKU
func (r *PostgresProductVariantRepository) GetBySKU(ctx context.Context, sku string) (*entities.ProductVariant, error) {
	query := `
		SELECT id, product_id, sku, name, price, cost, weight, dimensions,
		       length, width, height, volume, barcode, image_url, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_digital,
		       download_url, max_downloads, expiry_days, sort_order, created_at, updated_at
		FROM product_variants
		WHERE sku = $1
	`

	variant := &entities.ProductVariant{}
	err := r.db.QueryRow(ctx, query, sku).Scan(
		&variant.ID,
		&variant.ProductID,
		&variant.SKU,
		&variant.Name,
		&variant.Price,
		&variant.Cost,
		&variant.Weight,
		&variant.Dimensions,
		&variant.Length,
		&variant.Width,
		&variant.Height,
		&variant.Volume,
		&variant.Barcode,
		&variant.ImageURL,
		&variant.TrackInventory,
		&variant.StockQuantity,
		&variant.MinStockLevel,
		&variant.MaxStockLevel,
		&variant.AllowBackorder,
		&variant.RequiresShipping,
		&variant.Taxable,
		&variant.TaxRate,
		&variant.IsActive,
		&variant.IsDigital,
		&variant.DownloadURL,
		&variant.MaxDownloads,
		&variant.ExpiryDays,
		&variant.SortOrder,
		&variant.CreatedAt,
		&variant.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product variant with sku %s not found", sku)
		}
		return nil, fmt.Errorf("failed to get product variant by sku: %w", err)
	}

	return variant, nil
}

// GetByProductID retrieves all variants for a product
func (r *PostgresProductVariantRepository) GetByProductID(ctx context.Context, productID uuid.UUID) ([]*entities.ProductVariant, error) {
	query := `
		SELECT id, product_id, sku, name, price, cost, weight, dimensions,
		       length, width, height, volume, barcode, image_url, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_digital,
		       download_url, max_downloads, expiry_days, sort_order, created_at, updated_at
		FROM product_variants
		WHERE product_id = $1
		ORDER BY sort_order ASC, name ASC
	`

	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variants by product id: %w", err)
	}
	defer rows.Close()

	var variants []*entities.ProductVariant
	for rows.Next() {
		variant := &entities.ProductVariant{}
		err := rows.Scan(
			&variant.ID,
			&variant.ProductID,
			&variant.SKU,
			&variant.Name,
			&variant.Price,
			&variant.Cost,
			&variant.Weight,
			&variant.Dimensions,
			&variant.Length,
			&variant.Width,
			&variant.Height,
			&variant.Volume,
			&variant.Barcode,
			&variant.ImageURL,
			&variant.TrackInventory,
			&variant.StockQuantity,
			&variant.MinStockLevel,
			&variant.MaxStockLevel,
			&variant.AllowBackorder,
			&variant.RequiresShipping,
			&variant.Taxable,
			&variant.TaxRate,
			&variant.IsActive,
			&variant.IsDigital,
			&variant.DownloadURL,
			&variant.MaxDownloads,
			&variant.ExpiryDays,
			&variant.SortOrder,
			&variant.CreatedAt,
			&variant.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan variant row: %w", err)
		}
		variants = append(variants, variant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating variant rows: %w", err)
	}

	return variants, nil
}

// Update updates a product variant
func (r *PostgresProductVariantRepository) Update(ctx context.Context, variant *entities.ProductVariant) error {
	query := `
		UPDATE product_variants
		SET name = $2, price = $3, cost = $4, weight = $5, dimensions = $6,
		    length = $7, width = $8, height = $9, volume = $10, barcode = $11,
		    image_url = $12, track_inventory = $13, stock_quantity = $14,
		    min_stock_level = $15, max_stock_level = $16, allow_backorder = $17,
		    requires_shipping = $18, taxable = $19, tax_rate = $20,
		    is_active = $21, is_digital = $22, download_url = $23,
		    max_downloads = $24, expiry_days = $25, sort_order = $26, updated_at = $27
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		variant.ID,
		variant.Name,
		variant.Price,
		variant.Cost,
		variant.Weight,
		variant.Dimensions,
		variant.Length,
		variant.Width,
		variant.Height,
		variant.Volume,
		variant.Barcode,
		variant.ImageURL,
		variant.TrackInventory,
		variant.StockQuantity,
		variant.MinStockLevel,
		variant.MaxStockLevel,
		variant.AllowBackorder,
		variant.RequiresShipping,
		variant.Taxable,
		variant.TaxRate,
		variant.IsActive,
		variant.IsDigital,
		variant.DownloadURL,
		variant.MaxDownloads,
		variant.ExpiryDays,
		variant.SortOrder,
		variant.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update product variant: %w", err)
	}

	return nil
}

// Delete deletes a product variant
func (r *PostgresProductVariantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM product_variants WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete product variant: %w", err)
	}

	return nil
}

// List retrieves a list of product variants
func (r *PostgresProductVariantRepository) List(ctx context.Context, filter repositories.ProductVariantFilter) ([]*entities.ProductVariant, error) {
	// Build the base query
	baseQuery := `
		SELECT id, product_id, sku, name, price, cost, weight, dimensions,
		       length, width, height, volume, barcode, image_url, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_digital,
		       download_url, max_downloads, expiry_days, sort_order, created_at, updated_at
		FROM product_variants
		WHERE 1=1
	`

	// Build WHERE conditions
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add product filter
	if filter.ProductID != nil {
		conditions = append(conditions, fmt.Sprintf("product_id = $%d", argIndex))
		args = append(args, *filter.ProductID)
		argIndex++
	}

	// Add search filter
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR sku ILIKE $%d)", argIndex, argIndex+1))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
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

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY
	sortBy := "sort_order, name"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}

	sortOrder := "ASC"
	if filter.SortOrder != "" {
		sortOrder = strings.ToUpper(filter.SortOrder)
	}
	baseQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

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
		return nil, fmt.Errorf("failed to list product variants: %w", err)
	}
	defer rows.Close()

	var variants []*entities.ProductVariant
	for rows.Next() {
		variant := &entities.ProductVariant{}
		err := rows.Scan(
			&variant.ID,
			&variant.ProductID,
			&variant.SKU,
			&variant.Name,
			&variant.Price,
			&variant.Cost,
			&variant.Weight,
			&variant.Dimensions,
			&variant.Length,
			&variant.Width,
			&variant.Height,
			&variant.Volume,
			&variant.Barcode,
			&variant.ImageURL,
			&variant.TrackInventory,
			&variant.StockQuantity,
			&variant.MinStockLevel,
			&variant.MaxStockLevel,
			&variant.AllowBackorder,
			&variant.RequiresShipping,
			&variant.Taxable,
			&variant.TaxRate,
			&variant.IsActive,
			&variant.IsDigital,
			&variant.DownloadURL,
			&variant.MaxDownloads,
			&variant.ExpiryDays,
			&variant.SortOrder,
			&variant.CreatedAt,
			&variant.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan variant row: %w", err)
		}
		variants = append(variants, variant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating variant rows: %w", err)
	}

	return variants, nil
}

// Count returns the count of variants matching the filter
func (r *PostgresProductVariantRepository) Count(ctx context.Context, filter repositories.ProductVariantFilter) (int, error) {
	baseQuery := `SELECT COUNT(*) FROM product_variants WHERE 1=1`

	// Build WHERE conditions (same as in List)
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.ProductID != nil {
		conditions = append(conditions, fmt.Sprintf("product_id = $%d", argIndex))
		args = append(args, *filter.ProductID)
		argIndex++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR sku ILIKE $%d)", argIndex, argIndex+1))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
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

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	var count int
	err := r.db.QueryRow(ctx, baseQuery, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count product variants: %w", err)
	}

	return count, nil
}

// ExistsBySKU checks if a variant exists by SKU
func (r *PostgresProductVariantRepository) ExistsBySKU(ctx context.Context, sku string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM product_variants WHERE sku = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, sku).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if variant exists by sku: %w", err)
	}

	return exists, nil
}

// UpdateStock updates the stock quantity for a variant
func (r *PostgresProductVariantRepository) UpdateStock(ctx context.Context, variantID uuid.UUID, quantity int) error {
	query := `UPDATE product_variants SET stock_quantity = $2, updated_at = $3 WHERE id = $1`

	_, err := r.db.Exec(ctx, query, variantID, quantity, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update variant stock: %w", err)
	}

	return nil
}

// AdjustStock adjusts the stock quantity for a variant
func (r *PostgresProductVariantRepository) AdjustStock(ctx context.Context, variantID uuid.UUID, adjustment int) error {
	query := `
		UPDATE product_variants
		SET stock_quantity = stock_quantity + $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, variantID, adjustment, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to adjust variant stock: %w", err)
	}

	return nil
}

// GetActiveByProductID retrieves active variants for a product
func (r *PostgresProductVariantRepository) GetActiveByProductID(ctx context.Context, productID uuid.UUID) ([]*entities.ProductVariant, error) {
	query := `
		SELECT id, product_id, sku, name, price, cost, weight, dimensions,
		       length, width, height, volume, barcode, image_url, track_inventory,
		       stock_quantity, min_stock_level, max_stock_level, allow_backorder,
		       requires_shipping, taxable, tax_rate, is_active, is_digital,
		       download_url, max_downloads, expiry_days, sort_order, created_at, updated_at
		FROM product_variants
		WHERE product_id = $1 AND is_active = true
		ORDER BY sort_order ASC, name ASC
	`

	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active variants by product id: %w", err)
	}
	defer rows.Close()

	var variants []*entities.ProductVariant
	for rows.Next() {
		variant := &entities.ProductVariant{}
		err := rows.Scan(
			&variant.ID,
			&variant.ProductID,
			&variant.SKU,
			&variant.Name,
			&variant.Price,
			&variant.Cost,
			&variant.Weight,
			&variant.Dimensions,
			&variant.Length,
			&variant.Width,
			&variant.Height,
			&variant.Volume,
			&variant.Barcode,
			&variant.ImageURL,
			&variant.TrackInventory,
			&variant.StockQuantity,
			&variant.MinStockLevel,
			&variant.MaxStockLevel,
			&variant.AllowBackorder,
			&variant.RequiresShipping,
			&variant.Taxable,
			&variant.TaxRate,
			&variant.IsActive,
			&variant.IsDigital,
			&variant.DownloadURL,
			&variant.MaxDownloads,
			&variant.ExpiryDays,
			&variant.SortOrder,
			&variant.CreatedAt,
			&variant.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan variant row: %w", err)
		}
		variants = append(variants, variant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating variant rows: %w", err)
	}

	return variants, nil
}

// BulkCreate creates multiple variants in a single transaction
func (r *PostgresProductVariantRepository) BulkCreate(ctx context.Context, variants []*entities.ProductVariant) error {
	if len(variants) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO product_variants (
			id, product_id, sku, name, price, cost, weight, dimensions,
			length, width, height, volume, barcode, image_url, track_inventory,
			stock_quantity, min_stock_level, max_stock_level, allow_backorder,
			requires_shipping, taxable, tax_rate, is_active, is_digital,
			download_url, max_downloads, expiry_days, sort_order, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29
		)
	`

	for _, variant := range variants {
		_, err = tx.Exec(ctx, query,
			variant.ID,
			variant.ProductID,
			variant.SKU,
			variant.Name,
			variant.Price,
			variant.Cost,
			variant.Weight,
			variant.Dimensions,
			variant.Length,
			variant.Width,
			variant.Height,
			variant.Volume,
			variant.Barcode,
			variant.ImageURL,
			variant.TrackInventory,
			variant.StockQuantity,
			variant.MinStockLevel,
			variant.MaxStockLevel,
			variant.AllowBackorder,
			variant.RequiresShipping,
			variant.Taxable,
			variant.TaxRate,
			variant.IsActive,
			variant.IsDigital,
			variant.DownloadURL,
			variant.MaxDownloads,
			variant.ExpiryDays,
			variant.SortOrder,
			variant.CreatedAt,
			variant.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to create variant: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// BulkDelete deletes multiple variants in a single transaction
func (r *PostgresProductVariantRepository) BulkDelete(ctx context.Context, variantIDs []uuid.UUID) error {
	if len(variantIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(variantIDs))
	args := make([]interface{}, len(variantIDs))
	for i, id := range variantIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`DELETE FROM product_variants WHERE id IN (%s)`, strings.Join(placeholders, ", "))

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk delete variants: %w", err)
	}

	return nil
}

// PostgresVariantAttributeRepository implements VariantAttributeRepository for PostgreSQL
type PostgresVariantAttributeRepository struct {
	db *database.Database
}

// NewPostgresVariantAttributeRepository creates a new PostgreSQL variant attribute repository
func NewPostgresVariantAttributeRepository(db *database.Database) *PostgresVariantAttributeRepository {
	return &PostgresVariantAttributeRepository{
		db: db,
	}
}

// Create creates a new variant attribute
func (r *PostgresVariantAttributeRepository) Create(ctx context.Context, attribute *entities.VariantAttribute) error {
	query := `
		INSERT INTO variant_attributes (id, variant_id, name, value, type, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(ctx, query,
		attribute.ID,
		attribute.VariantID,
		attribute.Name,
		attribute.Value,
		attribute.Type,
		attribute.SortOrder,
	)

	if err != nil {
		return fmt.Errorf("failed to create variant attribute: %w", err)
	}

	return nil
}

// GetByVariantID retrieves attributes for a variant
func (r *PostgresVariantAttributeRepository) GetByVariantID(ctx context.Context, variantID uuid.UUID) ([]*entities.VariantAttribute, error) {
	query := `
		SELECT id, variant_id, name, value, type, sort_order
		FROM variant_attributes
		WHERE variant_id = $1
		ORDER BY sort_order ASC, name ASC
	`

	rows, err := r.db.Query(ctx, query, variantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attributes by variant id: %w", err)
	}
	defer rows.Close()

	var attributes []*entities.VariantAttribute
	for rows.Next() {
		attribute := &entities.VariantAttribute{}
		err := rows.Scan(
			&attribute.ID,
			&attribute.VariantID,
			&attribute.Name,
			&attribute.Value,
			&attribute.Type,
			&attribute.SortOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attribute row: %w", err)
		}
		attributes = append(attributes, attribute)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attribute rows: %w", err)
	}

	return attributes, nil
}

// Update updates a variant attribute
func (r *PostgresVariantAttributeRepository) Update(ctx context.Context, attribute *entities.VariantAttribute) error {
	query := `
		UPDATE variant_attributes
		SET name = $2, value = $3, type = $4, sort_order = $5
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		attribute.ID,
		attribute.Name,
		attribute.Value,
		attribute.Type,
		attribute.SortOrder,
	)

	if err != nil {
		return fmt.Errorf("failed to update variant attribute: %w", err)
	}

	return nil
}

// Delete deletes a variant attribute
func (r *PostgresVariantAttributeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM variant_attributes WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete variant attribute: %w", err)
	}

	return nil
}

// DeleteByVariantID deletes all attributes for a variant
func (r *PostgresVariantAttributeRepository) DeleteByVariantID(ctx context.Context, variantID uuid.UUID) error {
	query := `DELETE FROM variant_attributes WHERE variant_id = $1`

	_, err := r.db.Exec(ctx, query, variantID)
	if err != nil {
		return fmt.Errorf("failed to delete variant attributes: %w", err)
	}

	return nil
}

// BulkCreate creates multiple attributes in a single transaction
func (r *PostgresVariantAttributeRepository) BulkCreate(ctx context.Context, attributes []*entities.VariantAttribute) error {
	if len(attributes) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO variant_attributes (id, variant_id, name, value, type, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, attribute := range attributes {
		_, err = tx.Exec(ctx, query,
			attribute.ID,
			attribute.VariantID,
			attribute.Name,
			attribute.Value,
			attribute.Type,
			attribute.SortOrder,
		)
		if err != nil {
			return fmt.Errorf("failed to create attribute: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// PostgresVariantImageRepository implements VariantImageRepository for PostgreSQL
type PostgresVariantImageRepository struct {
	db *database.Database
}

// NewPostgresVariantImageRepository creates a new PostgreSQL variant image repository
func NewPostgresVariantImageRepository(db *database.Database) *PostgresVariantImageRepository {
	return &PostgresVariantImageRepository{
		db: db,
	}
}

// Create creates a new variant image
func (r *PostgresVariantImageRepository) Create(ctx context.Context, image *entities.VariantImage) error {
	query := `
		INSERT INTO variant_images (id, variant_id, image_url, alt_text, sort_order, is_main)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(ctx, query,
		image.ID,
		image.VariantID,
		image.ImageURL,
		image.AltText,
		image.SortOrder,
		image.IsMain,
	)

	if err != nil {
		return fmt.Errorf("failed to create variant image: %w", err)
	}

	return nil
}

// GetByVariantID retrieves images for a variant
func (r *PostgresVariantImageRepository) GetByVariantID(ctx context.Context, variantID uuid.UUID) ([]*entities.VariantImage, error) {
	query := `
		SELECT id, variant_id, image_url, alt_text, sort_order, is_main
		FROM variant_images
		WHERE variant_id = $1
		ORDER BY is_main DESC, sort_order ASC
	`

	rows, err := r.db.Query(ctx, query, variantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get images by variant id: %w", err)
	}
	defer rows.Close()

	var images []*entities.VariantImage
	for rows.Next() {
		image := &entities.VariantImage{}
		err := rows.Scan(
			&image.ID,
			&image.VariantID,
			&image.ImageURL,
			&image.AltText,
			&image.SortOrder,
			&image.IsMain,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan image row: %w", err)
		}
		images = append(images, image)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating image rows: %w", err)
	}

	return images, nil
}

// GetMainImage retrieves the main image for a variant
func (r *PostgresVariantImageRepository) GetMainImage(ctx context.Context, variantID uuid.UUID) (*entities.VariantImage, error) {
	query := `
		SELECT id, variant_id, image_url, alt_text, sort_order, is_main
		FROM variant_images
		WHERE variant_id = $1 AND is_main = true
		LIMIT 1
	`

	image := &entities.VariantImage{}
	err := r.db.QueryRow(ctx, query, variantID).Scan(
		&image.ID,
		&image.VariantID,
		&image.ImageURL,
		&image.AltText,
		&image.SortOrder,
		&image.IsMain,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("main image for variant %s not found", variantID)
		}
		return nil, fmt.Errorf("failed to get main image: %w", err)
	}

	return image, nil
}

// Update updates a variant image
func (r *PostgresVariantImageRepository) Update(ctx context.Context, image *entities.VariantImage) error {
	query := `
		UPDATE variant_images
		SET image_url = $2, alt_text = $3, sort_order = $4, is_main = $5
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		image.ID,
		image.ImageURL,
		image.AltText,
		image.SortOrder,
		image.IsMain,
	)

	if err != nil {
		return fmt.Errorf("failed to update variant image: %w", err)
	}

	return nil
}

// Delete deletes a variant image
func (r *PostgresVariantImageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM variant_images WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete variant image: %w", err)
	}

	return nil
}

// DeleteByVariantID deletes all images for a variant
func (r *PostgresVariantImageRepository) DeleteByVariantID(ctx context.Context, variantID uuid.UUID) error {
	query := `DELETE FROM variant_images WHERE variant_id = $1`

	_, err := r.db.Exec(ctx, query, variantID)
	if err != nil {
		return fmt.Errorf("failed to delete variant images: %w", err)
	}

	return nil
}

// SetMainImage sets an image as the main image for a variant
func (r *PostgresVariantImageRepository) SetMainImage(ctx context.Context, variantID, imageID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Unset all existing main images
	unsetQuery := `UPDATE variant_images SET is_main = false WHERE variant_id = $1`
	_, err = tx.Exec(ctx, unsetQuery, variantID)
	if err != nil {
		return fmt.Errorf("failed to unset main images: %w", err)
	}

	// Set the new main image
	setQuery := `UPDATE variant_images SET is_main = true WHERE id = $1 AND variant_id = $2`
	_, err = tx.Exec(ctx, setQuery, imageID, variantID)
	if err != nil {
		return fmt.Errorf("failed to set main image: %w", err)
	}

	return tx.Commit(ctx)
}

// BulkCreate creates multiple images in a single transaction
func (r *PostgresVariantImageRepository) BulkCreate(ctx context.Context, images []*entities.VariantImage) error {
	if len(images) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO variant_images (id, variant_id, image_url, alt_text, sort_order, is_main)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, image := range images {
		_, err = tx.Exec(ctx, query,
			image.ID,
			image.VariantID,
			image.ImageURL,
			image.AltText,
			image.SortOrder,
			image.IsMain,
		)
		if err != nil {
			return fmt.Errorf("failed to create image: %w", err)
		}
	}

	return tx.Commit(ctx)
}