-- Enhance inventory transaction tracking with additional features

-- Create transaction_types enum for better type safety
CREATE TYPE transaction_type AS ENUM (
    'PURCHASE',
    'SALE',
    'ADJUSTMENT',
    'TRANSFER_IN',
    'TRANSFER_OUT',
    'RETURN',
    'DAMAGE',
    'THEFT',
    'EXPIRY',
    'PRODUCTION',
    'CONSUMPTION',
    'COUNT'
);

-- Create transaction_status enum for workflow management
CREATE TYPE transaction_status AS ENUM (
    'PENDING',
    'APPROVED',
    'REJECTED',
    'COMPLETED',
    'CANCELLED'
);

-- Add new columns to inventory_transactions table
ALTER TABLE inventory_transactions
ADD COLUMN IF NOT EXISTS transaction_status transaction_type NOT NULL DEFAULT 'ADJUSTMENT';

ALTER TABLE inventory_transactions
ADD COLUMN IF NOT EXISTS status transaction_status NOT NULL DEFAULT 'COMPLETED';

ALTER TABLE inventory_transactions
ADD COLUMN IF NOT EXISTS reference_type VARCHAR(50);

ALTER TABLE inventory_transactions
ADD COLUMN IF NOT EXISTS unit_cost DECIMAL(12,2) CHECK (unit_cost >= 0);

ALTER TABLE inventory_transactions
ADD COLUMN IF NOT EXISTS total_cost DECIMAL(12,2) CHECK (total_cost >= 0);

ALTER TABLE inventory_transactions
ADD COLUMN IF NOT EXISTS batch_number VARCHAR(100);

ALTER TABLE inventory_transactions
ADD COLUMN IF NOT EXISTS expiry_date DATE;

ALTER TABLE inventory_transactions
ADD COLUMN IF NOT EXISTS serial_number VARCHAR(100);

ALTER TABLE inventory_transactions
ADD COLUMN IF NOT EXISTS from_warehouse_id UUID REFERENCES warehouses(id) ON DELETE SET NULL;

ALTER TABLE inventory_transactions
ADD COLUMN IF NOT EXISTS to_warehouse_id UUID REFERENCES warehouses(id) ON DELETE SET NULL;

ALTER TABLE inventory_transactions
ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE inventory_transactions
ADD COLUMN IF NOT EXISTS approved_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- Add check constraints for enhanced validation
ALTER TABLE inventory_transactions
ADD CONSTRAINT check_transaction_type_quantity
CHECK (
    (transaction_type IN ('PURCHASE', 'TRANSFER_IN', 'RETURN', 'PRODUCTION', 'COUNT') AND quantity > 0) OR
    (transaction_type IN ('SALE', 'TRANSFER_OUT', 'DAMAGE', 'THEFT', 'EXPIRY', 'CONSUMPTION') AND quantity < 0) OR
    (transaction_type = 'ADJUSTMENT') -- Adjustments can be positive or negative
);

ALTER TABLE inventory_transactions
ADD CONSTRAINT check_transfer_warehouses
CHECK (
    (transaction_type IN ('TRANSFER_IN', 'TRANSFER_OUT') AND
     from_warehouse_id IS NOT NULL AND
     to_warehouse_id IS NOT NULL AND
     from_warehouse_id != to_warehouse_id) OR
    (transaction_type NOT IN ('TRANSFER_IN', 'TRANSFER_OUT') AND
     from_warehouse_id IS NULL AND
     to_warehouse_id IS NULL)
);

ALTER TABLE inventory_transactions
ADD CONSTRAINT check_approval_consistency
CHECK (
    (status IN ('APPROVED', 'COMPLETED') AND approved_at IS NOT NULL AND approved_by IS NOT NULL) OR
    (status NOT IN ('APPROVED', 'COMPLETED'))
);

-- Create indexes for new columns
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_transaction_type ON inventory_transactions(transaction_type);
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_status ON inventory_transactions(status);
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_batch_number ON inventory_transactions(batch_number) WHERE batch_number IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_serial_number ON inventory_transactions(serial_number) WHERE serial_number IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_expiry_date ON inventory_transactions(expiry_date) WHERE expiry_date IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_from_warehouse_id ON inventory_transactions(from_warehouse_id) WHERE from_warehouse_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_to_warehouse_id ON inventory_transactions(to_warehouse_id) WHERE to_warehouse_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_approved_by ON inventory_transactions(approved_by) WHERE approved_by IS NOT NULL;

-- Create composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_warehouse_type_status
ON inventory_transactions(warehouse_id, transaction_type, status);

CREATE INDEX IF NOT EXISTS idx_inventory_transactions_product_type_date
ON inventory_transactions(product_id, transaction_type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_inventory_transactions_batch_expiry
ON inventory_transactions(batch_number, expiry_date)
WHERE batch_number IS NOT NULL AND expiry_date IS NOT NULL;

-- Create inventory_adjustments table for tracking adjustment reasons and approvals
CREATE TABLE IF NOT EXISTS inventory_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES inventory_transactions(id) ON DELETE CASCADE,
    adjustment_reason VARCHAR(100) NOT NULL,
    adjustment_category VARCHAR(50) NOT NULL, -- DAMAGE, THEFT, EXPIRY, COUNT_DISCREPANCY, SYSTEM_ERROR, etc.
    notes TEXT,
    supporting_documents JSONB, -- Store document references or metadata
    requested_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMP WITH TIME ZONE,
    rejection_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for inventory_adjustments table
CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_transaction_id ON inventory_adjustments(transaction_id);
CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_category ON inventory_adjustments(adjustment_category);
CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_requested_by ON inventory_adjustments(requested_by);
CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_approved_by ON inventory_adjustments(approved_by) WHERE approved_by IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_created_at ON inventory_adjustments(created_at);

-- Create inventory_batches table for batch tracking
CREATE TABLE IF NOT EXISTS inventory_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    batch_number VARCHAR(100) NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    quantity_available INTEGER NOT NULL DEFAULT 0 CHECK (quantity_available >= 0),
    quantity_reserved INTEGER NOT NULL DEFAULT 0 CHECK (quantity_reserved >= 0),
    manufacture_date DATE,
    expiry_date DATE,
    unit_cost DECIMAL(12,2) CHECK (unit_cost >= 0),
    supplier_id UUID, -- Reference to supplier if available
    notes TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(product_id, warehouse_id, batch_number)
);

-- Create indexes for inventory_batches table
CREATE INDEX IF NOT EXISTS idx_inventory_batches_product_id ON inventory_batches(product_id);
CREATE INDEX IF NOT EXISTS idx_inventory_batches_warehouse_id ON inventory_batches(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_inventory_batches_batch_number ON inventory_batches(batch_number);
CREATE INDEX IF NOT EXISTS idx_inventory_batches_expiry_date ON inventory_batches(expiry_date) WHERE expiry_date IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_inventory_batches_is_active ON inventory_batches(is_active);
CREATE INDEX IF NOT EXISTS idx_inventory_batches_quantity_available ON inventory_batches(quantity_available) WHERE quantity_available > 0;

-- Create trigger for inventory_batches table
CREATE TRIGGER trigger_inventory_batches_updated_at
    BEFORE UPDATE ON inventory_batches
    FOR EACH ROW
    EXECUTE FUNCTION update_products_updated_at(); -- Reuse existing function

-- Add check constraint for inventory_batches
ALTER TABLE inventory_batches
ADD CONSTRAINT check_batch_quantities
CHECK (quantity_available <= quantity AND quantity_reserved <= quantity_available);

-- Create inventory_transfers table for detailed transfer tracking
CREATE TABLE IF NOT EXISTS inventory_transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transfer_number VARCHAR(50) NOT NULL UNIQUE,
    from_warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    to_warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING', -- PENDING, IN_TRANSIT, RECEIVED, CANCELLED
    initiated_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMP WITH TIME ZONE,
    shipped_at TIMESTAMP WITH TIME ZONE,
    received_at TIMESTAMP WITH TIME ZONE,
    received_by UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    expected_delivery_date DATE,
    actual_delivery_date DATE,
    tracking_number VARCHAR(100),
    carrier VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for inventory_transfers table
CREATE INDEX IF NOT EXISTS idx_inventory_transfers_transfer_number ON inventory_transfers(transfer_number);
CREATE INDEX IF NOT EXISTS idx_inventory_transfers_from_warehouse_id ON inventory_transfers(from_warehouse_id);
CREATE INDEX IF NOT EXISTS idx_inventory_transfers_to_warehouse_id ON inventory_transfers(to_warehouse_id);
CREATE INDEX IF NOT EXISTS idx_inventory_transfers_status ON inventory_transfers(status);
CREATE INDEX IF NOT EXISTS idx_inventory_transfers_initiated_by ON inventory_transfers(initiated_by);
CREATE INDEX IF NOT EXISTS idx_inventory_transfers_created_at ON inventory_transfers(created_at);

-- Create trigger for inventory_transfers table
CREATE TRIGGER trigger_inventory_transfers_updated_at
    BEFORE UPDATE ON inventory_transfers
    FOR EACH ROW
    EXECUTE FUNCTION update_products_updated_at(); -- Reuse existing function

-- Create inventory_transfer_items table for transfer line items
CREATE TABLE IF NOT EXISTS inventory_transfer_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transfer_id UUID NOT NULL REFERENCES inventory_transfers(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity_requested INTEGER NOT NULL CHECK (quantity_requested > 0),
    quantity_shipped INTEGER NOT NULL DEFAULT 0 CHECK (quantity_shipped >= 0),
    quantity_received INTEGER NOT NULL DEFAULT 0 CHECK (quantity_received >= 0),
    batch_number VARCHAR(100),
    unit_cost DECIMAL(12,2) CHECK (unit_cost >= 0),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for inventory_transfer_items table
CREATE INDEX IF NOT EXISTS idx_inventory_transfer_items_transfer_id ON inventory_transfer_items(transfer_id);
CREATE INDEX IF NOT EXISTS idx_inventory_transfer_items_product_id ON inventory_transfer_items(product_id);
CREATE INDEX IF NOT EXISTS idx_inventory_transfer_items_batch_number ON inventory_transfer_items(batch_number) WHERE batch_number IS NOT NULL;

-- Create trigger for inventory_transfer_items table
CREATE TRIGGER trigger_inventory_transfer_items_updated_at
    BEFORE UPDATE ON inventory_transfer_items
    FOR EACH ROW
    EXECUTE FUNCTION update_products_updated_at(); -- Reuse existing function

-- Add check constraints for inventory_transfer_items
ALTER TABLE inventory_transfer_items
ADD CONSTRAINT check_transfer_item_quantities
CHECK (quantity_shipped <= quantity_requested AND quantity_received <= quantity_shipped);

-- Create enhanced views for better reporting

-- View for warehouse transaction summary
CREATE OR REPLACE VIEW warehouse_transaction_summary AS
SELECT
    w.id as warehouse_id,
    w.name as warehouse_name,
    w.code as warehouse_code,
    it.transaction_type,
    it.status,
    COUNT(*) as transaction_count,
    SUM(ABS(it.quantity)) as total_quantity,
    COALESCE(SUM(it.total_cost), 0) as total_value,
    MIN(it.created_at) as earliest_transaction,
    MAX(it.created_at) as latest_transaction
FROM warehouses w
JOIN inventory_transactions it ON w.id = it.warehouse_id
WHERE w.is_active = true
GROUP BY w.id, w.name, w.code, it.transaction_type, it.status;

-- View for batch tracking
CREATE OR REPLACE VIEW inventory_batch_tracking AS
SELECT
    ib.id,
    ib.product_id,
    p.name as product_name,
    p.sku as product_sku,
    ib.warehouse_id,
    w.name as warehouse_name,
    w.code as warehouse_code,
    ib.batch_number,
    ib.quantity,
    ib.quantity_available,
    ib.quantity_reserved,
    ib.manufacture_date,
    ib.expiry_date,
    ib.unit_cost,
    CASE
        WHEN ib.expiry_date < CURRENT_DATE THEN 'EXPIRED'
        WHEN ib.expiry_date <= CURRENT_DATE + INTERVAL '30 days' THEN 'EXPIRING_SOON'
        ELSE 'ACTIVE'
    END as batch_status,
    ib.is_active,
    ib.created_at,
    ib.updated_at
FROM inventory_batches ib
JOIN products p ON ib.product_id = p.id
JOIN warehouses w ON ib.warehouse_id = w.id
WHERE ib.is_active = true;

-- View for transfer status
CREATE OR REPLACE VIEW inventory_transfer_status AS
SELECT
    it.id,
    it.transfer_number,
    it.from_warehouse_id,
    w_from.name as from_warehouse_name,
    w_from.code as from_warehouse_code,
    it.to_warehouse_id,
    w_to.name as to_warehouse_name,
    w_to.code as to_warehouse_code,
    it.status,
    COUNT(iti.id) as total_items,
    SUM(iti.quantity_requested) as total_quantity_requested,
    SUM(iti.quantity_shipped) as total_quantity_shipped,
    SUM(iti.quantity_received) as total_quantity_received,
    it.initiated_by,
    u_initiator.username as initiator_username,
    it.approved_by,
    u_approver.username as approver_username,
    it.approved_at,
    it.shipped_at,
    it.received_at,
    it.expected_delivery_date,
    it.actual_delivery_date,
    it.tracking_number,
    it.carrier,
    it.created_at,
    it.updated_at
FROM inventory_transfers it
JOIN warehouses w_from ON it.from_warehouse_id = w_from.id
JOIN warehouses w_to ON it.to_warehouse_id = w_to.id
JOIN users u_initiator ON it.initiated_by = u_initiator.id
LEFT JOIN users u_approver ON it.approved_by = u_approver.id
LEFT JOIN inventory_transfer_items iti ON it.id = iti.transfer_id
GROUP BY it.id, w_from.id, w_to.id, u_initiator.id, u_approver.id;

-- Add comments for new tables and columns
COMMENT ON TYPE transaction_type IS 'Enumeration of inventory transaction types';
COMMENT ON TYPE transaction_status IS 'Enumeration of transaction workflow statuses';
COMMENT ON COLUMN inventory_transactions.transaction_type IS 'Type of inventory transaction using enum for type safety';
COMMENT ON COLUMN inventory_transactions.status IS 'Workflow status of the transaction';
COMMENT ON COLUMN inventory_transactions.reference_type IS 'Type of reference document (PO, SO, etc.)';
COMMENT ON COLUMN inventory_transactions.unit_cost IS 'Unit cost for the transaction items';
COMMENT ON COLUMN inventory_transactions.total_cost IS 'Total cost for the transaction';
COMMENT ON COLUMN inventory_transactions.batch_number IS 'Batch number for traceability';
COMMENT ON COLUMN inventory_transactions.expiry_date IS 'Expiry date for batched items';
COMMENT ON COLUMN inventory_transactions.serial_number IS 'Serial number for serialized items';
COMMENT ON COLUMN inventory_transactions.from_warehouse_id IS 'Source warehouse for transfers';
COMMENT ON COLUMN inventory_transactions.to_warehouse_id IS 'Destination warehouse for transfers';
COMMENT ON COLUMN inventory_transactions.approved_at IS 'Timestamp when transaction was approved';
COMMENT ON COLUMN inventory_transactions.approved_by IS 'User who approved the transaction';

COMMENT ON TABLE inventory_adjustments IS 'Detailed tracking of inventory adjustments with approval workflow';
COMMENT ON TABLE inventory_batches IS 'Batch tracking for inventory items';
COMMENT ON TABLE inventory_transfers IS 'Warehouse transfer management';
COMMENT ON TABLE inventory_transfer_items IS 'Individual items within a warehouse transfer';

COMMENT ON VIEW warehouse_transaction_summary IS 'Summary of transactions by warehouse and type';
COMMENT ON VIEW inventory_batch_tracking IS 'Comprehensive batch tracking information';
COMMENT ON VIEW inventory_transfer_status IS 'Current status of warehouse transfers';