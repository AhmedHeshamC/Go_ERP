-- Drop views and functions created in the up migration
DROP MATERIALIZED VIEW IF EXISTS warehouse_performance_metrics;
DROP VIEW IF EXISTS warehouse_recent_transactions;
DROP VIEW IF EXISTS warehouse_inventory_summary;
DROP FUNCTION IF EXISTS refresh_warehouse_performance_metrics();
DROP FUNCTION IF EXISTS log_warehouse_inventory_changes();

-- Drop indexes created in the up migration
DROP INDEX IF EXISTS idx_product_inventory_warehouse_capacity;
DROP INDEX IF EXISTS idx_product_inventory_low_stock;
DROP INDEX IF EXISTS idx_inventory_transactions_warehouse_type;
DROP INDEX IF EXISTS idx_inventory_transactions_warehouse_product;
DROP INDEX IF EXISTS idx_product_inventory_warehouse_product;
DROP INDEX IF EXISTS idx_warehouse_performance_metrics_warehouse_id;

-- Drop foreign key constraints (commented out as they might be needed by other constraints)
-- ALTER TABLE inventory_transactions DROP CONSTRAINT IF EXISTS fk_inventory_transactions_warehouse_id;
-- ALTER TABLE product_inventory DROP CONSTRAINT IF EXISTS fk_product_inventory_warehouse_id;