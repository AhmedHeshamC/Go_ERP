-- Migration: Add missing performance indexes
-- Created: Database Performance Optimization
-- Description: Adds missing indexes for optimal query performance (<100ms response times)

-- Add missing composite index for users table (email, is_active)
-- This optimizes user authentication queries that filter by both email and active status
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email_active
ON users(email, is_active)
WHERE is_active = true;

-- Add missing index for users last_login_at
-- This optimizes queries that fetch recently active users for reporting
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_last_login
ON users(last_login_at)
WHERE last_login_at IS NOT NULL;

-- Add missing partial index for products active only
-- This optimizes product listing queries that only show active products
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_products_active_partial
ON products(category_id, created_at)
WHERE is_active = true;

-- Add missing composite index for orders by date and status (correct order)
-- This optimizes order reporting queries by date range and status
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_date_status_correct
ON orders(order_date, status);

-- Add composite index for customer orders by date for reporting
-- This optimizes customer order history queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_customer_date
ON orders(customer_id, order_date DESC);

-- Add index for low stock products across all warehouses
-- This optimizes inventory reorder alerts
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_products_low_stock_all
ON products(min_stock_level, stock_quantity)
WHERE track_inventory = true AND stock_quantity <= min_stock_level;

-- Add composite index for order payment analytics
-- This optimizes financial reporting queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_payment_analytics
ON orders(payment_status, order_date, total_amount);

-- Add index for customer type and active status filtering
-- This optimizes customer listing queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_customers_type_active
ON customers(type, is_active)
WHERE is_active = true;

-- Add index for product categories with active status
-- This optimizes category-based product searches
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_product_categories_active
ON product_categories(parent_id, is_active)
WHERE is_active = true;

-- Add index for warehouse location-based queries
-- This optimizes location-based warehouse searches
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_warehouses_location_active
ON warehouses(country, city, is_active)
WHERE is_active = true;

-- Add composite index for inventory transactions analysis
-- This optimizes inventory movement reporting
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_inventory_transactions_analytics
ON inventory_transactions(transaction_type, created_at, product_id);

-- Create function to monitor index usage
CREATE OR REPLACE FUNCTION monitor_index_usage()
RETURNS TABLE(
    schemaname name,
    tablename name,
    indexname name,
    idx_scan bigint,
    idx_tup_read bigint,
    idx_tup_fetch bigint
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        pg_stat_user_indexes.schemaname,
        pg_stat_user_indexes.relname::name as tablename,
        pg_stat_user_indexes.indexrelname::name as indexname,
        pg_stat_user_indexes.idx_scan,
        pg_stat_user_indexes.idx_tup_read,
        pg_stat_user_indexes.idx_tup_fetch
    FROM pg_stat_user_indexes
    WHERE pg_stat_user_indexes.schemaname = 'public'
    ORDER BY pg_stat_user_indexes.idx_scan DESC;
END;
$$ LANGUAGE plpgsql;

-- Create view for slow query monitoring
CREATE OR REPLACE VIEW slow_queries AS
SELECT
    query,
    calls,
    total_time,
    mean_time,
    rows,
    100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0) AS hit_percent
FROM pg_stat_statements
WHERE mean_time > 100 -- queries taking more than 100ms
ORDER BY mean_time DESC;

-- Create function to analyze table sizes for optimization
CREATE OR REPLACE FUNCTION analyze_table_sizes()
RETURNS TABLE(
    tablename name,
    total_size text,
    index_size text,
    table_size text,
    row_count bigint
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        pg_class.relname::name,
        pg_size_pretty(pg_total_relation_size(pg_class.oid)) as total_size,
        pg_size_pretty(pg_indexes_size(pg_class.oid)) as index_size,
        pg_size_pretty(pg_relation_size(pg_class.oid)) as table_size,
        pg_class.reltuples::bigint as row_count
    FROM pg_class
    LEFT JOIN pg_namespace ON (pg_namespace.oid = pg_class.relnamespace)
    WHERE pg_namespace.nspname = 'public' AND pg_class.relkind = 'r'
    ORDER BY pg_total_relation_size(pg_class.oid) DESC;
END;
$$ LANGUAGE plpgsql;

-- Grant execute permissions to monitoring functions
GRANT EXECUTE ON FUNCTION monitor_index_usage() TO PUBLIC;
GRANT EXECUTE ON FUNCTION analyze_table_sizes() TO PUBLIC;
GRANT SELECT ON slow_queries TO PUBLIC;

-- Create procedure to update table statistics
CREATE OR REPLACE PROCEDURE update_table_statistics()
LANGUAGE plpgsql
AS $$
DECLARE
    table_record RECORD;
BEGIN
    FOR table_record IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
    LOOP
        EXECUTE 'ANALYZE ' || quote_ident(table_record.tablename);
    END LOOP;
END;
$$;

-- Comments for documentation
COMMENT ON INDEX idx_users_email_active IS 'Composite index for user authentication queries (email + active status)';
COMMENT ON INDEX idx_users_last_login IS 'Index for recently active users queries';
COMMENT ON INDEX idx_products_active_partial IS 'Partial index for active products only queries';
COMMENT ON INDEX idx_orders_date_status_correct IS 'Index for order reporting by date and status';
COMMENT ON INDEX idx_orders_customer_date IS 'Index for customer order history';
COMMENT ON INDEX idx_products_low_stock_all IS 'Index for low stock alerts across all warehouses';
COMMENT ON INDEX idx_orders_payment_analytics IS 'Index for financial reporting and payment analytics';
COMMENT ON INDEX idx_customers_type_active IS 'Index for customer listing with type filtering';
COMMENT ON INDEX idx_product_categories_active IS 'Index for active category hierarchy queries';
COMMENT ON INDEX idx_warehouses_location_active IS 'Index for location-based warehouse searches';
COMMENT ON INDEX idx_inventory_transactions_analytics IS 'Index for inventory movement reporting and analytics';

COMMENT ON FUNCTION monitor_index_usage() IS 'Monitor index usage statistics for performance optimization';
COMMENT ON VIEW slow_queries IS 'View to monitor queries taking longer than 100ms';
COMMENT ON FUNCTION analyze_table_sizes() IS 'Analyze table sizes for storage optimization';
COMMENT ON PROCEDURE update_table_statistics() IS 'Update table statistics for query optimizer';