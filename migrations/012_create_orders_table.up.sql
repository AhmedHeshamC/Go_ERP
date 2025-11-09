-- Create orders table
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number VARCHAR(50) NOT NULL UNIQUE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PENDING', 'CONFIRMED', 'PROCESSING', 'SHIPPED', 'DELIVERED', 'CANCELLED', 'REFUNDED', 'RETURNED', 'ON_HOLD', 'PARTIALLY_SHIPPED')),
    previous_status VARCHAR(20) CHECK (previous_status IN ('DRAFT', 'PENDING', 'CONFIRMED', 'PROCESSING', 'SHIPPED', 'DELIVERED', 'CANCELLED', 'REFUNDED', 'RETURNED', 'ON_HOLD', 'PARTIALLY_SHIPPED')),
    priority VARCHAR(10) NOT NULL DEFAULT 'NORMAL' CHECK (priority IN ('LOW', 'NORMAL', 'HIGH', 'URGENT', 'CRITICAL')),
    type VARCHAR(15) NOT NULL DEFAULT 'SALES' CHECK (type IN ('SALES', 'PURCHASE', 'RETURN', 'EXCHANGE', 'TRANSFER', 'ADJUSTMENT')),
    payment_status VARCHAR(15) NOT NULL DEFAULT 'PENDING' CHECK (payment_status IN ('PENDING', 'PAID', 'PARTIALLY_PAID', 'OVERDUE', 'REFUNDED', 'FAILED')),
    shipping_method VARCHAR(20) NOT NULL DEFAULT 'STANDARD' CHECK (shipping_method IN ('STANDARD', 'EXPRESS', 'OVERNIGHT', 'INTERNATIONAL', 'PICKUP', 'DIGITAL')),

    -- Financial fields
    subtotal DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (subtotal >= 0),
    tax_amount DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (tax_amount >= 0),
    shipping_amount DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (shipping_amount >= 0),
    discount_amount DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    total_amount DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    paid_amount DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (paid_amount >= 0),
    refunded_amount DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (refunded_amount >= 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'USD' CHECK (currency ~ '^[A-Z]{3}$'),

    -- Date fields
    order_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    required_date TIMESTAMP WITH TIME ZONE,
    shipping_date TIMESTAMP WITH TIME ZONE,
    delivery_date TIMESTAMP WITH TIME ZONE,
    cancelled_date TIMESTAMP WITH TIME ZONE,

    -- Address references
    shipping_address_id UUID NOT NULL REFERENCES order_addresses(id) ON DELETE RESTRICT,
    billing_address_id UUID NOT NULL REFERENCES order_addresses(id) ON DELETE RESTRICT,

    -- Metadata
    notes TEXT,
    internal_notes TEXT,
    customer_notes TEXT,
    tracking_number VARCHAR(100),
    carrier VARCHAR(50),

    -- System fields
    created_by UUID NOT NULL,
    approved_by UUID,
    shipped_by UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMP WITH TIME ZONE,
    shipped_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for orders table
CREATE INDEX idx_orders_order_number ON orders(order_number);
CREATE INDEX idx_orders_customer_id ON orders(customer_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_previous_status ON orders(previous_status) WHERE previous_status IS NOT NULL;
CREATE INDEX idx_orders_priority ON orders(priority);
CREATE INDEX idx_orders_type ON orders(type);
CREATE INDEX idx_orders_payment_status ON orders(payment_status);
CREATE INDEX idx_orders_shipping_method ON orders(shipping_method);
CREATE INDEX idx_orders_currency ON orders(currency);
CREATE INDEX idx_orders_order_date ON orders(order_date);
CREATE INDEX idx_orders_required_date ON orders(required_date) WHERE required_date IS NOT NULL;
CREATE INDEX idx_orders_shipping_date ON orders(shipping_date) WHERE shipping_date IS NOT NULL;
CREATE INDEX idx_orders_delivery_date ON orders(delivery_date) WHERE delivery_date IS NOT NULL;
CREATE INDEX idx_orders_cancelled_date ON orders(cancelled_date) WHERE cancelled_date IS NOT NULL;
CREATE INDEX idx_orders_shipping_address_id ON orders(shipping_address_id);
CREATE INDEX idx_orders_billing_address_id ON orders(billing_address_id);
CREATE INDEX idx_orders_tracking_number ON orders(tracking_number) WHERE tracking_number IS NOT NULL;
CREATE INDEX idx_orders_carrier ON orders(carrier) WHERE carrier IS NOT NULL;
CREATE INDEX idx_orders_created_by ON orders(created_by);
CREATE INDEX idx_orders_approved_by ON orders(approved_by) WHERE approved_by IS NOT NULL;
CREATE INDEX idx_orders_shipped_by ON orders(shipped_by) WHERE shipped_by IS NOT NULL;
CREATE INDEX idx_orders_created_at ON orders(created_at);
CREATE INDEX idx_orders_updated_at ON orders(updated_at);
CREATE INDEX idx_orders_approved_at ON orders(approved_at) WHERE approved_at IS NOT NULL;
CREATE INDEX idx_orders_shipped_at ON orders(shipped_at) WHERE shipped_at IS NOT NULL;

-- Composite indexes for common queries
CREATE INDEX idx_orders_customer_status ON orders(customer_id, status);
CREATE INDEX idx_orders_status_date ON orders(status, order_date);
CREATE INDEX idx_orders_payment_status_date ON orders(payment_status, order_date);
CREATE INDEX idx_orders_customer_priority ON orders(customer_id, priority) WHERE priority IN ('HIGH', 'URGENT', 'CRITICAL');
CREATE INDEX idx_orders_shipping_tracking ON orders(shipping_method, tracking_number) WHERE tracking_number IS NOT NULL;

-- Financial indexes
CREATE INDEX idx_orders_total_amount ON orders(total_amount);
CREATE INDEX idx_orders_subtotal ON orders(subtotal);
CREATE INDEX idx_orders_currency_total ON orders(currency, total_amount);

-- Create trigger for orders table
CREATE TRIGGER trigger_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_companies_updated_at();

-- Add check constraints for business logic
ALTER TABLE orders ADD CONSTRAINT check_total_calculation
    CHECK (total_amount = subtotal + tax_amount + shipping_amount - discount_amount);

ALTER TABLE orders ADD CONSTRAINT check_paid_vs_total
    CHECK (paid_amount <= total_amount);

ALTER TABLE orders ADD CONSTRAINT check_refunded_vs_paid
    CHECK (refunded_amount <= paid_amount);

ALTER TABLE orders ADD CONSTRAINT check_order_date_not_future
    CHECK (order_date <= NOW());

ALTER TABLE orders ADD CONSTRAINT check_required_date_after_order
    CHECK (required_date IS NULL OR required_date >= order_date);

ALTER TABLE orders ADD CONSTRAINT check_shipping_date_after_order
    CHECK (shipping_date IS NULL OR shipping_date >= order_date);

ALTER TABLE orders ADD CONSTRAINT check_delivery_date_after_shipping
    CHECK (delivery_date IS NULL OR shipping_date IS NULL OR delivery_date >= shipping_date);

ALTER TABLE orders ADD CONSTRAINT check_cancelled_date_after_order
    CHECK (cancelled_date IS NULL OR cancelled_date >= order_date);

-- Add comments for orders table
COMMENT ON TABLE orders IS 'Customer orders and sales transactions';
COMMENT ON COLUMN orders.id IS 'Unique identifier for the order';
COMMENT ON COLUMN orders.order_number IS 'Unique order number in format YYYY-NNNNNN';
COMMENT ON COLUMN orders.customer_id IS 'Reference to the customer';
COMMENT ON COLUMN orders.status IS 'Current order status';
COMMENT ON COLUMN orders.previous_status IS 'Previous order status before last change';
COMMENT ON COLUMN orders.priority IS 'Order priority level';
COMMENT ON COLUMN orders.type IS 'Order type: SALES, PURCHASE, RETURN, etc.';
COMMENT ON COLUMN orders.payment_status IS 'Payment status';
COMMENT ON COLUMN orders.shipping_method IS 'Shipping method';
COMMENT ON COLUMN orders.subtotal IS 'Subtotal of all items';
COMMENT ON COLUMN orders.tax_amount IS 'Total tax amount';
COMMENT ON COLUMN orders.shipping_amount IS 'Shipping cost';
COMMENT ON COLUMN orders.discount_amount IS 'Total discount amount';
COMMENT ON COLUMN orders.total_amount IS 'Total order amount';
COMMENT ON COLUMN orders.paid_amount IS 'Amount paid so far';
COMMENT ON COLUMN orders.refunded_amount IS 'Amount refunded';
COMMENT ON COLUMN orders.currency IS 'Currency code (ISO 4217)';
COMMENT ON COLUMN orders.order_date IS 'Date when order was placed';
COMMENT ON COLUMN orders.required_date IS 'Required delivery date';
COMMENT ON COLUMN orders.shipping_date IS 'Date when order was shipped';
COMMENT ON COLUMN orders.delivery_date IS 'Date when order was delivered';
COMMENT ON COLUMN orders.cancelled_date IS 'Date when order was cancelled';
COMMENT ON COLUMN orders.shipping_address_id IS 'Reference to shipping address';
COMMENT ON COLUMN orders.billing_address_id IS 'Reference to billing address';
COMMENT ON COLUMN orders.notes IS 'Order notes';
COMMENT ON COLUMN orders.internal_notes IS 'Internal notes (not visible to customer)';
COMMENT ON COLUMN orders.customer_notes IS 'Notes provided by customer';
COMMENT ON COLUMN orders.tracking_number IS 'Shipment tracking number';
COMMENT ON COLUMN orders.carrier IS 'Shipping carrier name';
COMMENT ON COLUMN orders.created_by IS 'User who created the order';
COMMENT ON COLUMN orders.approved_by IS 'User who approved the order';
COMMENT ON COLUMN orders.shipped_by IS 'User who shipped the order';
COMMENT ON COLUMN orders.created_at IS 'Timestamp when the order was created';
COMMENT ON COLUMN orders.updated_at IS 'Timestamp when the order was last updated';
COMMENT ON COLUMN orders.approved_at IS 'Timestamp when the order was approved';
COMMENT ON COLUMN orders.shipped_at IS 'Timestamp when the order was shipped';