-- Drop tables in reverse order due to foreign key constraints
DROP TABLE IF EXISTS variant_inventory_transactions;
DROP TABLE IF EXISTS variant_inventory;
DROP TABLE IF EXISTS variant_images;
DROP TABLE IF EXISTS variant_attributes;
DROP TABLE IF EXISTS product_variants;

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_ensure_variant_consistency ON products;
DROP TRIGGER IF EXISTS trigger_variant_inventory_updated_at ON variant_inventory;
DROP TRIGGER IF EXISTS trigger_product_variants_updated_at ON product_variants;

-- Drop indexes
DROP INDEX IF EXISTS idx_variant_inventory_transactions_reference_id;
DROP INDEX IF EXISTS idx_variant_inventory_transactions_created_at;
DROP INDEX IF EXISTS idx_variant_inventory_transactions_type;
DROP INDEX IF EXISTS idx_variant_inventory_transactions_warehouse_id;
DROP INDEX IF EXISTS idx_variant_inventory_transactions_variant_id;

DROP INDEX IF EXISTS idx_variant_inventory_reorder_level;
DROP INDEX IF EXISTS idx_variant_inventory_quantity_available;
DROP INDEX IF EXISTS idx_variant_inventory_warehouse_id;
DROP INDEX IF EXISTS idx_variant_inventory_variant_id;

DROP INDEX IF EXISTS idx_variant_images_unique_main;
DROP INDEX IF EXISTS idx_variant_images_is_main;
DROP INDEX IF EXISTS idx_variant_images_sort_order;
DROP INDEX IF EXISTS idx_variant_images_variant_id;

DROP INDEX IF EXISTS idx_variant_attributes_unique_main;
DROP INDEX IF EXISTS idx_variant_attributes_sort_order;
DROP INDEX IF EXISTS idx_variant_attributes_type;
DROP INDEX IF EXISTS idx_variant_attributes_name;
DROP INDEX IF EXISTS idx_variant_attributes_variant_id;

DROP INDEX IF EXISTS idx_product_variants_product_sort;
DROP INDEX IF EXISTS idx_product_variants_product_active;
DROP INDEX IF EXISTS idx_product_variants_stock_quantity;
DROP INDEX IF EXISTS idx_product_variants_barcode;
DROP INDEX IF EXISTS idx_product_variants_sort_order;
DROP INDEX IF EXISTS idx_product_variants_is_active;
DROP INDEX IF EXISTS idx_product_variants_price;
DROP INDEX IF EXISTS idx_product_variants_name;
DROP INDEX IF EXISTS idx_product_variants_sku;
DROP INDEX IF EXISTS idx_product_variants_product_id;

-- Drop the trigger function
DROP FUNCTION IF EXISTS ensure_variant_consistency();