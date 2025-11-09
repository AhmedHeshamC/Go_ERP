-- Drop views
DROP VIEW IF EXISTS order_items_detail;
DROP VIEW IF EXISTS order_summary;
DROP VIEW IF EXISTS order_billing_addresses;
DROP VIEW IF EXISTS order_shipping_addresses;
DROP VIEW IF EXISTS customer_addresses;

-- Drop functions
DROP FUNCTION IF EXISTS generate_next_order_number();
DROP FUNCTION IF EXISTS validate_order_number(VARCHAR);
DROP FUNCTION IF EXISTS get_customer_default_addresses(UUID);

-- Drop foreign key constraints
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS fk_order_items_product_id;
ALTER TABLE order_addresses DROP CONSTRAINT IF EXISTS fk_order_addresses_order_id;

-- Drop indexes
DROP INDEX IF EXISTS idx_order_addresses_order_id_active;
DROP INDEX IF EXISTS idx_order_addresses_customer_id_active;