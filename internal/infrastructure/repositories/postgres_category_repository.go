package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"erpgo/internal/domain/products/entities"
	"erpgo/internal/domain/products/repositories"
	"erpgo/pkg/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PostgresCategoryRepository implements CategoryRepository for PostgreSQL
type PostgresCategoryRepository struct {
	db *database.Database
}

// NewPostgresCategoryRepository creates a new PostgreSQL category repository
func NewPostgresCategoryRepository(db *database.Database) *PostgresCategoryRepository {
	return &PostgresCategoryRepository{
		db: db,
	}
}

// Create creates a new category
func (r *PostgresCategoryRepository) Create(ctx context.Context, category *entities.ProductCategory) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert category
	query := `
		INSERT INTO product_categories (
			id, name, description, parent_id, level, path, image_url,
			sort_order, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = tx.Exec(ctx, query,
		category.ID,
		category.Name,
		category.Description,
		category.ParentID,
		category.Level,
		category.Path,
		category.ImageURL,
		category.SortOrder,
		category.IsActive,
		category.CreatedAt,
		category.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}

	// Insert category metadata
	metadataQuery := `
		INSERT INTO category_metadata (
			id, category_id, seo_title, seo_description, seo_keywords,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		ON CONFLICT (category_id) DO UPDATE SET
			seo_title = EXCLUDED.seo_title,
			seo_description = EXCLUDED.seo_description,
			seo_keywords = EXCLUDED.seo_keywords,
			updated_at = EXCLUDED.updated_at
	`

	_, err = tx.Exec(ctx, metadataQuery,
		uuid.New(),
		category.ID,
		category.SEOTitle,
		category.SEODescription,
		category.SEOKeywords,
		category.CreatedAt,
		category.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create category metadata: %w", err)
	}

	return tx.Commit(ctx)
}

// GetByID retrieves a category by ID
func (r *PostgresCategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.ProductCategory, error) {
	query := `
		SELECT c.id, c.name, c.description, c.parent_id, c.level, c.path, c.image_url,
		       c.sort_order, c.is_active, c.created_at, c.updated_at,
		       cm.seo_title, cm.seo_description, cm.seo_keywords
		FROM product_categories c
		LEFT JOIN category_metadata cm ON c.id = cm.category_id
		WHERE c.id = $1
	`

	category := &entities.ProductCategory{}
	var seoTitle, seoDescription, seoKeywords *string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&category.ID,
		&category.Name,
		&category.Description,
		&category.ParentID,
		&category.Level,
		&category.Path,
		&category.ImageURL,
		&category.SortOrder,
		&category.IsActive,
		&category.CreatedAt,
		&category.UpdatedAt,
		&seoTitle,
		&seoDescription,
		&seoKeywords,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("category with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get category by id: %w", err)
	}

	// Set SEO fields from metadata
	if seoTitle != nil {
		category.SEOTitle = *seoTitle
	}
	if seoDescription != nil {
		category.SEODescription = *seoDescription
	}
	if seoKeywords != nil {
		category.SEOKeywords = *seoKeywords
	}

	return category, nil
}

// GetByPath retrieves a category by path
func (r *PostgresCategoryRepository) GetByPath(ctx context.Context, path string) (*entities.ProductCategory, error) {
	query := `
		SELECT id, name, description, parent_id, level, path, image_url,
		       sort_order, is_active, created_at, updated_at
		FROM product_categories
		WHERE path = $1
	`

	category := &entities.ProductCategory{}
	err := r.db.QueryRow(ctx, query, path).Scan(
		&category.ID,
		&category.Name,
		&category.Description,
		&category.ParentID,
		&category.Level,
		&category.Path,
		&category.ImageURL,
		&category.SortOrder,
		&category.IsActive,
		&category.CreatedAt,
		&category.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("category with path %s not found", path)
		}
		return nil, fmt.Errorf("failed to get category by path: %w", err)
	}

	return category, nil
}

// Update updates a category
func (r *PostgresCategoryRepository) Update(ctx context.Context, category *entities.ProductCategory) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update category
	query := `
		UPDATE product_categories
		SET name = $2, description = $3, parent_id = $4, level = $5,
		    path = $6, image_url = $7, sort_order = $8, is_active = $9,
		    updated_at = $10
		WHERE id = $1
	`

	_, err = tx.Exec(ctx, query,
		category.ID,
		category.Name,
		category.Description,
		category.ParentID,
		category.Level,
		category.Path,
		category.ImageURL,
		category.SortOrder,
		category.IsActive,
		category.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	// Update category metadata
	metadataQuery := `
		UPDATE category_metadata
		SET seo_title = $2, seo_description = $3, seo_keywords = $4, updated_at = $5
		WHERE category_id = $1
	`

	_, err = tx.Exec(ctx, metadataQuery,
		category.ID,
		category.SEOTitle,
		category.SEODescription,
		category.SEOKeywords,
		category.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update category metadata: %w", err)
	}

	return tx.Commit(ctx)
}

// Delete deletes a category
func (r *PostgresCategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM product_categories WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	return nil
}

// List retrieves a list of categories
func (r *PostgresCategoryRepository) List(ctx context.Context, filter repositories.CategoryFilter) ([]*entities.ProductCategory, error) {
	// Build the base query
	baseQuery := `
		SELECT id, name, description, parent_id, level, path, image_url,
		       sort_order, is_active, created_at, updated_at
		FROM product_categories
		WHERE 1=1
	`

	// Build WHERE conditions
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add search filter
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex+1))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}

	// Add parent filter
	if filter.ParentID != nil {
		if *filter.ParentID == uuid.Nil {
			conditions = append(conditions, "parent_id IS NULL")
		} else {
			conditions = append(conditions, fmt.Sprintf("parent_id = $%d", argIndex))
			args = append(args, *filter.ParentID)
			argIndex++
		}
	}

	// Add active filter
	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	// Add level filter
	if filter.Level != nil {
		conditions = append(conditions, fmt.Sprintf("level = $%d", argIndex))
		args = append(args, *filter.Level)
		argIndex++
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
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	defer rows.Close()

	var categories []*entities.ProductCategory
	for rows.Next() {
		category := &entities.ProductCategory{}
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Description,
			&category.ParentID,
			&category.Level,
			&category.Path,
			&category.ImageURL,
			&category.SortOrder,
			&category.IsActive,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category row: %w", err)
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category rows: %w", err)
	}

	return categories, nil
}

// ListRoot retrieves root categories (categories with no parent)
func (r *PostgresCategoryRepository) ListRoot(ctx context.Context) ([]*entities.ProductCategory, error) {
	query := `
		SELECT id, name, description, parent_id, level, path, image_url,
		       sort_order, is_active, created_at, updated_at
		FROM product_categories
		WHERE parent_id IS NULL
		ORDER BY sort_order ASC, name ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list root categories: %w", err)
	}
	defer rows.Close()

	var categories []*entities.ProductCategory
	for rows.Next() {
		category := &entities.ProductCategory{}
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Description,
			&category.ParentID,
			&category.Level,
			&category.Path,
			&category.ImageURL,
			&category.SortOrder,
			&category.IsActive,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category row: %w", err)
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category rows: %w", err)
	}

	return categories, nil
}

// GetChildren retrieves direct children of a category
func (r *PostgresCategoryRepository) GetChildren(ctx context.Context, parentID uuid.UUID) ([]*entities.ProductCategory, error) {
	query := `
		SELECT id, name, description, parent_id, level, path, image_url,
		       sort_order, is_active, created_at, updated_at
		FROM product_categories
		WHERE parent_id = $1
		ORDER BY sort_order ASC, name ASC
	`

	rows, err := r.db.Query(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category children: %w", err)
	}
	defer rows.Close()

	var categories []*entities.ProductCategory
	for rows.Next() {
		category := &entities.ProductCategory{}
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Description,
			&category.ParentID,
			&category.Level,
			&category.Path,
			&category.ImageURL,
			&category.SortOrder,
			&category.IsActive,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category row: %w", err)
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category rows: %w", err)
	}

	return categories, nil
}

// GetDescendants retrieves all descendants of a category using recursive CTE
func (r *PostgresCategoryRepository) GetDescendants(ctx context.Context, categoryID uuid.UUID) ([]*entities.ProductCategory, error) {
	query := `
		WITH RECURSIVE category_tree AS (
			SELECT id, name, description, parent_id, level, path, image_url,
			       sort_order, is_active, created_at, updated_at
			FROM product_categories
			WHERE parent_id = $1

			UNION ALL

			SELECT c.id, c.name, c.description, c.parent_id, c.level, c.path,
			       c.image_url, c.sort_order, c.is_active, c.created_at, c.updated_at
			FROM product_categories c
			INNER JOIN category_tree ct ON c.parent_id = ct.id
		)
		SELECT id, name, description, parent_id, level, path, image_url,
		       sort_order, is_active, created_at, updated_at
		FROM category_tree
		ORDER BY level ASC, sort_order ASC, name ASC
	`

	rows, err := r.db.Query(ctx, query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category descendants: %w", err)
	}
	defer rows.Close()

	var categories []*entities.ProductCategory
	for rows.Next() {
		category := &entities.ProductCategory{}
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Description,
			&category.ParentID,
			&category.Level,
			&category.Path,
			&category.ImageURL,
			&category.SortOrder,
			&category.IsActive,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category row: %w", err)
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category rows: %w", err)
	}

	return categories, nil
}

// GetAncestors retrieves all ancestors of a category using recursive CTE
func (r *PostgresCategoryRepository) GetAncestors(ctx context.Context, categoryID uuid.UUID) ([]*entities.ProductCategory, error) {
	query := `
		WITH RECURSIVE category_tree AS (
			SELECT id, name, description, parent_id, level, path, image_url,
			       sort_order, is_active, created_at, updated_at
			FROM product_categories
			WHERE id = $1

			UNION ALL

			SELECT c.id, c.name, c.description, c.parent_id, c.level, c.path,
			       c.image_url, c.sort_order, c.is_active, c.created_at, c.updated_at
			FROM product_categories c
			INNER JOIN category_tree ct ON c.id = ct.parent_id
		)
		SELECT id, name, description, parent_id, level, path, image_url,
		       sort_order, is_active, created_at, updated_at
		FROM category_tree
		WHERE id != $1
		ORDER BY level DESC
	`

	rows, err := r.db.Query(ctx, query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category ancestors: %w", err)
	}
	defer rows.Close()

	var categories []*entities.ProductCategory
	for rows.Next() {
		category := &entities.ProductCategory{}
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Description,
			&category.ParentID,
			&category.Level,
			&category.Path,
			&category.ImageURL,
			&category.SortOrder,
			&category.IsActive,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category row: %w", err)
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category rows: %w", err)
	}

	return categories, nil
}

// GetPath retrieves the full path (ancestors + self) of a category
func (r *PostgresCategoryRepository) GetPath(ctx context.Context, categoryID uuid.UUID) ([]*entities.ProductCategory, error) {
	query := `
		WITH RECURSIVE category_tree AS (
			SELECT id, name, description, parent_id, level, path, image_url,
			       sort_order, is_active, created_at, updated_at
			FROM product_categories
			WHERE id = $1

			UNION ALL

			SELECT c.id, c.name, c.description, c.parent_id, c.level, c.path,
			       c.image_url, c.sort_order, c.is_active, c.created_at, c.updated_at
			FROM product_categories c
			INNER JOIN category_tree ct ON c.id = ct.parent_id
		)
		SELECT id, name, description, parent_id, level, path, image_url,
		       sort_order, is_active, created_at, updated_at
		FROM category_tree
		ORDER BY level ASC
	`

	rows, err := r.db.Query(ctx, query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category path: %w", err)
	}
	defer rows.Close()

	var categories []*entities.ProductCategory
	for rows.Next() {
		category := &entities.ProductCategory{}
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Description,
			&category.ParentID,
			&category.Level,
			&category.Path,
			&category.ImageURL,
			&category.SortOrder,
			&category.IsActive,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category row: %w", err)
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category rows: %w", err)
	}

	return categories, nil
}

// Count returns the count of categories matching the filter
func (r *PostgresCategoryRepository) Count(ctx context.Context, filter repositories.CategoryFilter) (int, error) {
	baseQuery := `SELECT COUNT(*) FROM product_categories WHERE 1=1`

	// Build WHERE conditions (same as in List)
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex+1))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}

	if filter.ParentID != nil {
		if *filter.ParentID == uuid.Nil {
			conditions = append(conditions, "parent_id IS NULL")
		} else {
			conditions = append(conditions, fmt.Sprintf("parent_id = $%d", argIndex))
			args = append(args, *filter.ParentID)
			argIndex++
		}
	}

	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.Level != nil {
		conditions = append(conditions, fmt.Sprintf("level = $%d", argIndex))
		args = append(args, *filter.Level)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	var count int
	err := r.db.QueryRow(ctx, baseQuery, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count categories: %w", err)
	}

	return count, nil
}

// CountProducts counts products in a category and its subcategories
func (r *PostgresCategoryRepository) CountProducts(ctx context.Context, categoryID uuid.UUID) (int, error) {
	query := `
		WITH RECURSIVE category_tree AS (
			SELECT id
			FROM product_categories
			WHERE id = $1

			UNION ALL

			SELECT c.id
			FROM product_categories c
			INNER JOIN category_tree ct ON c.parent_id = ct.id
		)
		SELECT COUNT(*)
		FROM products p
		INNER JOIN category_tree ct ON p.category_id = ct.id
		WHERE p.is_active = true
	`

	var count int
	err := r.db.QueryRow(ctx, query, categoryID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count products in category: %w", err)
	}

	return count, nil
}

// ExistsByPath checks if a category exists by path
func (r *PostgresCategoryRepository) ExistsByPath(ctx context.Context, path string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM product_categories WHERE path = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, path).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if category exists by path: %w", err)
	}

	return exists, nil
}

// ExistsByName checks if a category exists by name within a parent
func (r *PostgresCategoryRepository) ExistsByName(ctx context.Context, name string, parentID *uuid.UUID) (bool, error) {
	var query string
	var args []interface{}

	if parentID == nil {
		query = `SELECT EXISTS(SELECT 1 FROM product_categories WHERE name = $1 AND parent_id IS NULL)`
		args = []interface{}{name}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM product_categories WHERE name = $1 AND parent_id = $2)`
		args = []interface{}{name, *parentID}
	}

	var exists bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if category exists by name: %w", err)
	}

	return exists, nil
}

// RebuildPaths rebuilds all category paths
func (r *PostgresCategoryRepository) RebuildPaths(ctx context.Context) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// First, reset all paths
	resetQuery := `UPDATE product_categories SET path = NULL, level = 0`
	_, err = tx.Exec(ctx, resetQuery)
	if err != nil {
		return fmt.Errorf("failed to reset category paths: %w", err)
	}

	// Build paths for root categories
	rootQuery := `
		UPDATE product_categories
		SET path = '/' || LOWER(REGEXP_REPLACE(name, '[^a-zA-Z0-9\s]', '', 'g')),
		    level = 0
		WHERE parent_id IS NULL
	`
	_, err = tx.Exec(ctx, rootQuery)
	if err != nil {
		return fmt.Errorf("failed to build root category paths: %w", err)
	}

	// Recursively build paths for child categories
	childQuery := `
		WITH RECURSIVE category_tree AS (
			SELECT id, parent_id, name, path, level
			FROM product_categories
			WHERE parent_id IS NULL

			UNION ALL

			SELECT c.id, c.parent_id, c.name, c.path, c.level
			FROM product_categories c
			INNER JOIN category_tree ct ON c.parent_id = ct.id
		)
		UPDATE product_categories pc
		SET path = (
			SELECT CASE
				WHEN ct.parent_id IS NULL THEN '/' || LOWER(REGEXP_REPLACE(ct.name, '[^a-zA-Z0-9\s]', '', 'g'))
				ELSE (
					SELECT parent_pc.path
					FROM product_categories parent_pc
					WHERE parent_pc.id = ct.parent_id
				) || '/' || LOWER(REGEXP_REPLACE(ct.name, '[^a-zA-Z0-9\s]', '', 'g'))
			END
			FROM category_tree ct
			WHERE ct.id = pc.id
		),
		level = (
			SELECT COUNT(*) - 1
			FROM category_tree ct_path
			WHERE ct_path.id IN (
				WITH RECURSIVE path_tree AS (
					SELECT id, parent_id
					FROM product_categories
					WHERE id = pc.id

					UNION ALL

					SELECT c.id, c.parent_id
					FROM product_categories c
					INNER JOIN path_tree pt ON c.id = pt.parent_id
				)
				SELECT id FROM path_tree
			)
		)
	`

	_, err = tx.Exec(ctx, childQuery)
	if err != nil {
		return fmt.Errorf("failed to build child category paths: %w", err)
	}

	// Update timestamps
	updateTimestampsQuery := `
		UPDATE product_categories
		SET updated_at = $1
		WHERE updated_at < $1 OR updated_at IS NULL
	`
	_, err = tx.Exec(ctx, updateTimestampsQuery, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update timestamps: %w", err)
	}

	return tx.Commit(ctx)
}

// RebuildCategoryPaths rebuilds paths for a specific category and its children
func (r *PostgresCategoryRepository) RebuildCategoryPaths(ctx context.Context, categoryID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		WITH RECURSIVE category_tree AS (
			SELECT id, parent_id, name, path
			FROM product_categories
			WHERE id = $1

			UNION ALL

			SELECT c.id, c.parent_id, c.name, c.path
			FROM product_categories c
			INNER JOIN category_tree ct ON c.parent_id = ct.id
		)
		UPDATE product_categories pc
		SET path = (
			SELECT CASE
				WHEN ct.parent_id IS NULL THEN '/' || LOWER(REGEXP_REPLACE(ct.name, '[^a-zA-Z0-9\s]', '', 'g'))
				ELSE (
					SELECT COALESCE(parent_pc.path, '') || '/' || LOWER(REGEXP_REPLACE(ct.name, '[^a-zA-Z0-9\s]', '', 'g'))
					FROM product_categories parent_pc
					WHERE parent_pc.id = ct.parent_id
				)
			END
			FROM category_tree ct
			WHERE ct.id = pc.id
		),
		level = (
			SELECT COUNT(*) - 1
			FROM category_tree ct_level
			WHERE ct_level.id IN (
				WITH RECURSIVE path_tree AS (
					SELECT id, parent_id
					FROM product_categories
					WHERE id = pc.id

					UNION ALL

					SELECT c.id, c.parent_id
					FROM product_categories c
					INNER JOIN path_tree pt ON c.id = pt.parent_id
				)
				SELECT id FROM path_tree
			)
		),
		updated_at = $2
	`

	_, err = tx.Exec(ctx, query, categoryID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to rebuild category paths: %w", err)
	}

	return tx.Commit(ctx)
}

// BulkUpdateSortOrder updates sort order for multiple categories
func (r *PostgresCategoryRepository) BulkUpdateSortOrder(ctx context.Context, updates []repositories.CategorySortUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, update := range updates {
		query := `
			UPDATE product_categories
			SET sort_order = $2, updated_at = $3
			WHERE id = $1
		`
		_, err = tx.Exec(ctx, query, update.CategoryID, update.SortOrder, time.Now().UTC())
		if err != nil {
			return fmt.Errorf("failed to update category sort order: %w", err)
		}
	}

	return tx.Commit(ctx)
}
