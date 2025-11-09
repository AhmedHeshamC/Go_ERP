-- Rollback migration: Remove performance indexes
-- Description: Removes indexes created in performance optimization migration

-- Drop performance indexes
DROP INDEX CONCURRENTLY IF EXISTS idx_users_email_active;
DROP INDEX CONCURRENTLY IF EXISTS idx_users_last_login;
DROP INDEX CONCURRENTLY IF EXISTS idx_products_active_partial;
DROP INDEX CONCURRENTLY IF EXISTS idx_orders_date_status_correct;
DROP INDEX CONCURRENTLY IF EXISTS idx_orders_customer_date;
DROP INDEX CONCURRENTLY IF EXISTS idx_products_low_stock_all;
DROP INDEX CONCURRENTLY IF EXISTS idx_orders_payment_analytics;
DROP INDEX CONCURRENTLY IF EXISTS idx_customers_type_active;
DROP INDEX CONCURRENTLY IF EXISTS idx_product_categories_active;
DROP INDEX CONCURRENTLY IF EXISTS idx_warehouses_location_active;
DROP INDEX CONCURRENTLY IF EXISTS idx_inventory_transactions_analytics;

-- Drop monitoring functions and views
DROP FUNCTION IF EXISTS monitor_index_usage();
DROP VIEW IF EXISTS slow_queries;
DROP FUNCTION IF EXISTS analyze_table_sizes();
DROP PROCEDURE IF EXISTS update_table_statistics();