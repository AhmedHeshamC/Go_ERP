-- Migration: Rollback production readiness indexes
-- Description: Removes indexes added for production readiness

-- Drop user table indexes
DROP INDEX CONCURRENTLY IF EXISTS idx_users_created_at_desc;

-- Drop role table indexes
DROP INDEX CONCURRENTLY IF EXISTS idx_roles_name_lower;

-- Drop user roles indexes
DROP INDEX CONCURRENTLY IF EXISTS idx_user_roles_assigned_at_desc;
DROP INDEX CONCURRENTLY IF EXISTS idx_user_roles_assigned_by;

-- Drop foreign key indexes
DROP INDEX CONCURRENTLY IF EXISTS idx_products_category_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_product_variants_product_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_inventory_product_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_inventory_warehouse_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_inventory_transactions_product_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_inventory_transactions_warehouse_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_orders_customer_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_orders_billing_address_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_orders_shipping_address_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_order_items_order_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_order_items_product_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_order_items_variant_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_order_addresses_customer_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_email_verifications_user_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_product_categories_parent_id;
