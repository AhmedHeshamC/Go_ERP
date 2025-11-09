-- Drop tables in reverse order due to foreign key constraints
DROP TABLE IF EXISTS inventory_transactions;
DROP TABLE IF EXISTS product_inventory;
DROP TABLE IF EXISTS products;

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_product_inventory_updated_at ON product_inventory;
DROP TRIGGER IF EXISTS trigger_products_updated_at ON products;

-- Drop indexes
DROP INDEX IF EXISTS idx_inventory_transactions_reference_id;
DROP INDEX IF EXISTS idx_inventory_transactions_created_at;
DROP INDEX IF EXISTS idx_inventory_transactions_type;
DROP INDEX IF EXISTS idx_inventory_transactions_warehouse_id;
DROP INDEX IF EXISTS idx_inventory_transactions_product_id;

DROP INDEX IF EXISTS idx_product_inventory_reorder_level;
DROP INDEX IF EXISTS idx_product_inventory_quantity_available;
DROP INDEX IF EXISTS idx_product_inventory_warehouse_id;
DROP INDEX IF EXISTS idx_product_inventory_product_id;

DROP INDEX IF EXISTS idx_products_category_featured;
DROP INDEX IF EXISTS idx_products_active_featured;
DROP INDEX IF EXISTS idx_products_category_active;
DROP INDEX IF EXISTS idx_products_created_at;
DROP INDEX IF EXISTS idx_products_stock_quantity;
DROP INDEX IF EXISTS idx_products_barcode;
DROP INDEX IF EXISTS idx_products_is_digital;
DROP INDEX IF EXISTS idx_products_is_featured;
DROP INDEX IF EXISTS idx_products_is_active;
DROP INDEX IF EXISTS idx_products_price;
DROP INDEX IF EXISTS idx_products_category_id;
DROP INDEX IF EXISTS idx_products_name;
DROP INDEX IF EXISTS idx_products_sku;

-- Drop the trigger function
DROP FUNCTION IF EXISTS update_products_updated_at();