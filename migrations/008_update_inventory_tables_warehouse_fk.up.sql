-- Update existing inventory tables to add foreign key constraints to warehouses table

-- Add foreign key constraint to product_inventory table
ALTER TABLE product_inventory
ADD CONSTRAINT fk_product_inventory_warehouse_id
FOREIGN KEY (warehouse_id) REFERENCES warehouses(id) ON DELETE RESTRICT;

-- Add foreign key constraint to inventory_transactions table
ALTER TABLE inventory_transactions
ADD CONSTRAINT fk_inventory_transactions_warehouse_id
FOREIGN KEY (warehouse_id) REFERENCES warehouses(id) ON DELETE RESTRICT;

-- Create additional indexes for improved query performance
CREATE INDEX IF NOT EXISTS idx_product_inventory_warehouse_product
ON product_inventory(warehouse_id, product_id);

CREATE INDEX IF NOT EXISTS idx_inventory_transactions_warehouse_product
ON inventory_transactions(warehouse_id, product_id);

CREATE INDEX IF NOT EXISTS idx_inventory_transactions_warehouse_type
ON inventory_transactions(warehouse_id, transaction_type);

-- Create composite index for low stock alerts
CREATE INDEX IF NOT EXISTS idx_product_inventory_low_stock
ON product_inventory(warehouse_id, quantity_available, reorder_level)
WHERE quantity_available <= reorder_level;

-- Create index for warehouse capacity management
CREATE INDEX IF NOT EXISTS idx_product_inventory_warehouse_capacity
ON product_inventory(warehouse_id, quantity_available);

-- Add comments for foreign key constraints
COMMENT ON CONSTRAINT fk_product_inventory_warehouse_id ON product_inventory
IS 'Foreign key reference to warehouses table';

COMMENT ON CONSTRAINT fk_inventory_transactions_warehouse_id ON inventory_transactions
IS 'Foreign key reference to warehouses table';

-- Create a view for warehouse inventory summary
CREATE OR REPLACE VIEW warehouse_inventory_summary AS
SELECT
    w.id as warehouse_id,
    w.name as warehouse_name,
    w.code as warehouse_code,
    COUNT(pi.id) as total_products,
    COALESCE(SUM(pi.quantity_available), 0) as total_quantity_available,
    COALESCE(SUM(pi.quantity_reserved), 0) as total_quantity_reserved,
    COALESCE(SUM(pi.quantity_available + pi.quantity_reserved), 0) as total_quantity_on_hand,
    COUNT(CASE WHEN pi.quantity_available <= pi.reorder_level THEN 1 END) as low_stock_products,
    COUNT(CASE WHEN pi.max_stock IS NOT NULL AND pi.quantity_available > pi.max_stock THEN 1 END) as overstock_products,
    w.is_active,
    w.created_at as warehouse_created_at
FROM warehouses w
LEFT JOIN product_inventory pi ON w.id = pi.warehouse_id
GROUP BY w.id, w.name, w.code, w.is_active, w.created_at;

-- Create indexes for the view
-- Note: Views don't have indexes, but the underlying tables are already indexed

-- Add comments for the view
COMMENT ON VIEW warehouse_inventory_summary IS 'Summary view of warehouse inventory levels and status';

-- Create a view for recent inventory transactions per warehouse
CREATE OR REPLACE VIEW warehouse_recent_transactions AS
SELECT DISTINCT ON (it.warehouse_id)
    it.warehouse_id,
    w.name as warehouse_name,
    w.code as warehouse_code,
    it.id as last_transaction_id,
    it.transaction_type,
    it.quantity,
    it.reason,
    it.created_at as last_transaction_date,
    it.created_by as last_transaction_user
FROM inventory_transactions it
JOIN warehouses w ON it.warehouse_id = w.id
ORDER BY it.warehouse_id, it.created_at DESC;

-- Add comments for the recent transactions view
COMMENT ON VIEW warehouse_recent_transactions IS 'Most recent transaction for each warehouse';

-- Create a materialized view for warehouse performance metrics
CREATE MATERIALIZED VIEW IF NOT EXISTS warehouse_performance_metrics AS
SELECT
    w.id as warehouse_id,
    w.name as warehouse_name,
    w.code as warehouse_code,
    COUNT(DISTINCT pi.product_id) as unique_products_count,
    COALESCE(AVG(pi.quantity_available), 0) as avg_stock_level,
    COALESCE(SUM(pi.quantity_available), 0) as total_stock,
    COUNT(CASE WHEN pi.quantity_available <= pi.reorder_level THEN 1 END) as low_stock_count,
    COUNT(CASE WHEN pi.max_stock IS NOT NULL AND pi.quantity_available > pi.max_stock THEN 1 END) as overstock_count,
    COUNT(CASE WHEN it.id IS NOT NULL THEN 1 END) as transaction_count_30d,
    w.is_active,
    LAST(pi.updated_at) as last_inventory_update
FROM warehouses w
LEFT JOIN product_inventory pi ON w.id = pi.warehouse_id
LEFT JOIN inventory_transactions it ON w.id = it.warehouse_id
    AND it.created_at >= NOW() - INTERVAL '30 days'
GROUP BY w.id, w.name, w.code, w.is_active;

-- Create index for the materialized view
CREATE INDEX IF NOT EXISTS idx_warehouse_performance_metrics_warehouse_id
ON warehouse_performance_metrics(warehouse_id);

-- Add comments for the performance metrics view
COMMENT ON MATERIALIZED VIEW warehouse_performance_metrics IS 'Warehouse performance metrics and KPIs';

-- Create a function to refresh the materialized view
CREATE OR REPLACE FUNCTION refresh_warehouse_performance_metrics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY warehouse_performance_metrics;
END;
$$ LANGUAGE plpgsql;

-- Add comment for the refresh function
COMMENT ON FUNCTION refresh_warehouse_performance_metrics()
IS 'Function to refresh the warehouse performance metrics materialized view';

-- Create a trigger function to track warehouse changes for audit purposes
CREATE OR REPLACE FUNCTION log_warehouse_inventory_changes()
RETURNS TRIGGER AS $$
BEGIN
    -- Log significant inventory changes
    IF TG_OP = 'UPDATE' THEN
        -- Check if quantity changed significantly (more than 10% or more than 100 units)
        IF ABS(NEW.quantity_available - OLD.quantity_available) > 100 OR
           (OLD.quantity_available > 0 AND
            ABS(NEW.quantity_available - OLD.quantity_available)::decimal / OLD.quantity_available > 0.1) THEN

            INSERT INTO inventory_transactions (
                product_id,
                warehouse_id,
                transaction_type,
                quantity,
                reason,
                created_at,
                created_by
            ) VALUES (
                NEW.product_id,
                NEW.warehouse_id,
                'ADJUSTMENT',
                NEW.quantity_available - OLD.quantity_available,
                'Automatic audit logging',
                NOW(),
                COALESCE(NEW.updated_by, OLD.updated_by)
            );
        END IF;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Create trigger for automatic change logging
-- Note: This trigger should be created carefully as it could generate many transactions
-- Consider enabling it only for specific warehouses or during specific periods
-- DROP TRIGGER IF EXISTS trigger_product_inventory_audit_log ON product_inventory;
-- CREATE TRIGGER trigger_product_inventory_audit_log
--     AFTER UPDATE ON product_inventory
--     FOR EACH ROW
--     EXECUTE FUNCTION log_warehouse_inventory_changes();

-- Add comment for the audit trigger function
COMMENT ON FUNCTION log_warehouse_inventory_changes()
IS 'Function to log significant inventory changes automatically';