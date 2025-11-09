-- Drop the metadata table first (due to foreign key constraint)
DROP TABLE IF EXISTS category_metadata;

-- Drop the trigger function
DROP TRIGGER IF EXISTS trigger_category_metadata_updated_at ON category_metadata;
DROP TRIGGER IF EXISTS trigger_product_categories_updated_at ON product_categories;

-- Drop the indexes
DROP INDEX IF EXISTS idx_category_metadata_category_id;
DROP INDEX IF EXISTS idx_product_categories_unique_path;
DROP INDEX IF EXISTS idx_product_categories_name_parent_id;
DROP INDEX IF EXISTS idx_product_categories_sort_order;
DROP INDEX IF EXISTS idx_product_categories_is_active;
DROP INDEX IF EXISTS idx_product_categories_level;
DROP INDEX IF EXISTS idx_product_categories_path;
DROP INDEX IF EXISTS idx_product_categories_parent_id;

-- Drop the product categories table
DROP TABLE IF EXISTS product_categories;

-- Drop the trigger function (shared function)
DROP FUNCTION IF EXISTS update_product_categories_updated_at();