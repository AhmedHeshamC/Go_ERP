-- Migration: Add production readiness indexes
-- Created: Production Readiness Phase 4
-- Description: Adds optimized indexes for production workload performance
-- Requirements: 6.1, 6.2

-- User table indexes
-- Optimized index for created_at with DESC order for recent user queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_created_at_desc 
ON users(created_at DESC);

-- Role table indexes
-- Case-insensitive index for role name lookups
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_roles_name_lower 
ON roles(LOWER(name));

-- User roles indexes
-- Optimized index for assigned_at with DESC order for recent assignments
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_roles_assigned_at_desc 
ON user_roles(assigned_at DESC);

-- Add index on assigned_by for audit queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_roles_assigned_by 
ON user_roles(assigned_by) 
WHERE assigned_by IS NOT NULL;

-- Foreign key indexes (if not already present)
-- These optimize JOIN operations and CASCADE deletes

-- Products table foreign keys
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_products_category_id 
ON products(category_id);

-- Product variants table foreign keys
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_product_variants_product_id 
ON product_variants(product_id);

-- Inventory table foreign keys
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_inventory_product_id 
ON inventory(product_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_inventory_warehouse_id 
ON inventory(warehouse_id);

-- Inventory transactions foreign keys
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_inventory_transactions_product_id 
ON inventory_transactions(product_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_inventory_transactions_warehouse_id 
ON inventory_transactions(warehouse_id);

-- Orders table foreign keys
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_customer_id 
ON orders(customer_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_billing_address_id 
ON orders(billing_address_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_shipping_address_id 
ON orders(shipping_address_id);

-- Order items foreign keys
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_order_items_order_id 
ON order_items(order_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_order_items_product_id 
ON order_items(product_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_order_items_variant_id 
ON order_items(variant_id);

-- Order addresses foreign keys
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_order_addresses_customer_id 
ON order_addresses(customer_id);

-- Email verifications foreign keys
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_email_verifications_user_id 
ON email_verifications(user_id);

-- Product categories foreign keys (self-referencing)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_product_categories_parent_id 
ON product_categories(parent_id);

-- Comments for documentation
COMMENT ON INDEX idx_users_created_at_desc IS 'Optimized index for recent user queries with DESC order';
COMMENT ON INDEX idx_roles_name_lower IS 'Case-insensitive index for role name lookups';
COMMENT ON INDEX idx_user_roles_assigned_at_desc IS 'Optimized index for recent role assignments with DESC order';
COMMENT ON INDEX idx_user_roles_assigned_by IS 'Index for audit queries on who assigned roles';
COMMENT ON INDEX idx_products_category_id IS 'Foreign key index for products-categories JOIN optimization';
COMMENT ON INDEX idx_product_variants_product_id IS 'Foreign key index for variants-products JOIN optimization';
COMMENT ON INDEX idx_inventory_product_id IS 'Foreign key index for inventory-products JOIN optimization';
COMMENT ON INDEX idx_inventory_warehouse_id IS 'Foreign key index for inventory-warehouses JOIN optimization';
COMMENT ON INDEX idx_inventory_transactions_product_id IS 'Foreign key index for transactions-products JOIN optimization';
COMMENT ON INDEX idx_inventory_transactions_warehouse_id IS 'Foreign key index for transactions-warehouses JOIN optimization';
COMMENT ON INDEX idx_orders_customer_id IS 'Foreign key index for orders-customers JOIN optimization';
COMMENT ON INDEX idx_orders_billing_address_id IS 'Foreign key index for orders-billing addresses JOIN optimization';
COMMENT ON INDEX idx_orders_shipping_address_id IS 'Foreign key index for orders-shipping addresses JOIN optimization';
COMMENT ON INDEX idx_order_items_order_id IS 'Foreign key index for order items-orders JOIN optimization';
COMMENT ON INDEX idx_order_items_product_id IS 'Foreign key index for order items-products JOIN optimization';
COMMENT ON INDEX idx_order_items_variant_id IS 'Foreign key index for order items-variants JOIN optimization';
COMMENT ON INDEX idx_order_addresses_customer_id IS 'Foreign key index for addresses-customers JOIN optimization';
COMMENT ON INDEX idx_email_verifications_user_id IS 'Foreign key index for email verifications-users JOIN optimization';
COMMENT ON INDEX idx_product_categories_parent_id IS 'Foreign key index for category hierarchy queries';
