-- Drop views created in the up migration
DROP VIEW IF EXISTS inventory_transfer_status;
DROP VIEW IF EXISTS inventory_batch_tracking;
DROP VIEW IF EXISTS warehouse_transaction_summary;

-- Drop tables created in the up migration
DROP TABLE IF EXISTS inventory_transfer_items;
DROP TABLE IF EXISTS inventory_transfers;
DROP TABLE IF EXISTS inventory_batches;
DROP TABLE IF EXISTS inventory_adjustments;

-- Drop indexes created in the up migration
DROP INDEX IF EXISTS idx_inventory_transactions_approved_by;
DROP INDEX IF EXISTS idx_inventory_transactions_to_warehouse_id;
DROP INDEX IF EXISTS idx_inventory_transactions_from_warehouse_id;
DROP INDEX IF EXISTS idx_inventory_transactions_expiry_date;
DROP INDEX IF EXISTS idx_inventory_transactions_serial_number;
DROP INDEX IF EXISTS idx_inventory_transactions_batch_number;
DROP INDEX IF EXISTS idx_inventory_transactions_status;
DROP INDEX IF EXISTS idx_inventory_transactions_transaction_type;
DROP INDEX IF EXISTS idx_inventory_transfers_initiated_by;
DROP INDEX IF EXISTS idx_inventory_transfers_created_at;
DROP INDEX IF EXISTS idx_inventory_transfers_status;
DROP INDEX IF EXISTS idx_inventory_transfers_to_warehouse_id;
DROP INDEX IF EXISTS idx_inventory_transfers_from_warehouse_id;
DROP INDEX IF EXISTS idx_inventory_transfers_transfer_number;
DROP INDEX IF EXISTS idx_inventory_transfer_items_batch_number;
DROP INDEX IF EXISTS idx_inventory_transfer_items_product_id;
DROP INDEX IF EXISTS idx_inventory_transfer_items_transfer_id;
DROP INDEX IF EXISTS idx_inventory_batches_quantity_available;
DROP INDEX IF EXISTS idx_inventory_batches_is_active;
DROP INDEX IF EXISTS idx_inventory_batches_expiry_date;
DROP INDEX IF EXISTS idx_inventory_batches_batch_number;
DROP INDEX IF EXISTS idx_inventory_batches_warehouse_id;
DROP INDEX IF EXISTS idx_inventory_batches_product_id;
DROP INDEX IF EXISTS idx_inventory_adjustments_approved_by;
DROP INDEX IF EXISTS idx_inventory_adjustments_requested_by;
DROP INDEX IF EXISTS idx_inventory_adjustments_category;
DROP INDEX IF EXISTS idx_inventory_adjustments_transaction_id;

-- Drop composite indexes
DROP INDEX IF EXISTS idx_inventory_transactions_product_type_date;
DROP INDEX IF EXISTS idx_inventory_transactions_warehouse_type_status;
DROP INDEX IF EXISTS idx_inventory_transactions_batch_expiry;

-- Drop check constraints
ALTER TABLE inventory_transfer_items DROP CONSTRAINT IF EXISTS check_transfer_item_quantities;
ALTER TABLE inventory_batches DROP CONSTRAINT IF EXISTS check_batch_quantities;
ALTER TABLE inventory_transactions DROP CONSTRAINT IF EXISTS check_approval_consistency;
ALTER TABLE inventory_transactions DROP CONSTRAINT IF EXISTS check_transfer_warehouses;
ALTER TABLE inventory_transactions DROP CONSTRAINT IF EXISTS check_transaction_type_quantity;

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_inventory_transfer_items_updated_at ON inventory_transfer_items;
DROP TRIGGER IF EXISTS trigger_inventory_transfers_updated_at ON inventory_transfers;
DROP TRIGGER IF EXISTS trigger_inventory_batches_updated_at ON inventory_batches;

-- Drop columns added to inventory_transactions table
ALTER TABLE inventory_transactions
DROP COLUMN IF EXISTS approved_by,
DROP COLUMN IF EXISTS approved_at,
DROP COLUMN IF EXISTS to_warehouse_id,
DROP COLUMN IF EXISTS from_warehouse_id,
DROP COLUMN IF EXISTS serial_number,
DROP COLUMN IF EXISTS expiry_date,
DROP COLUMN IF EXISTS batch_number,
DROP COLUMN IF EXISTS total_cost,
DROP COLUMN IF EXISTS unit_cost,
DROP COLUMN IF EXISTS reference_type,
DROP COLUMN IF EXISTS status,
DROP COLUMN IF EXISTS transaction_type;

-- Drop enums
DROP TYPE IF EXISTS transaction_status;
DROP TYPE IF EXISTS transaction_type;