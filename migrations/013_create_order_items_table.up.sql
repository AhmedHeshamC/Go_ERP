-- Create order_items table
CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL, -- Will reference products table
    product_sku VARCHAR(100) NOT NULL,
    product_name VARCHAR(300) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0 AND quantity <= 9999),
    unit_price DECIMAL(12,2) NOT NULL CHECK (unit_price >= 0 AND unit_price <= 999999.99),
    discount_amount DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0 AND discount_amount <= unit_price),
    tax_rate DECIMAL(5,2) NOT NULL DEFAULT 0 CHECK (tax_rate >= 0 AND tax_rate <= 100),
    tax_amount DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (tax_amount >= 0),
    total_price DECIMAL(12,2) NOT NULL CHECK (total_price >= 0),

    -- Additional fields
    weight DECIMAL(10,3) NOT NULL DEFAULT 0 CHECK (weight >= 0 AND weight <= 999999.99),
    dimensions VARCHAR(100),
    barcode VARCHAR(50),
    notes TEXT,

    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'ORDERED' CHECK (status IN ('ORDERED', 'SHIPPED', 'DELIVERED', 'CANCELLED', 'RETURNED', 'PARTIALLY_SHIPPED')),
    quantity_shipped INTEGER NOT NULL DEFAULT 0 CHECK (quantity_shipped >= 0 AND quantity_shipped <= quantity),
    quantity_returned INTEGER NOT NULL DEFAULT 0 CHECK (quantity_returned >= 0 AND quantity_returned <= quantity_shipped),

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for order_items table
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
CREATE INDEX idx_order_items_product_sku ON order_items(product_sku);
CREATE INDEX idx_order_items_product_name ON order_items(product_name);
CREATE INDEX idx_order_items_quantity ON order_items(quantity);
CREATE INDEX idx_order_items_unit_price ON order_items(unit_price);
CREATE INDEX idx_order_items_total_price ON order_items(total_price);
CREATE INDEX idx_order_items_weight ON order_items(weight);
CREATE INDEX idx_order_items_barcode ON order_items(barcode) WHERE barcode IS NOT NULL;
CREATE INDEX idx_order_items_status ON order_items(status);
CREATE INDEX idx_order_items_quantity_shipped ON order_items(quantity_shipped);
CREATE INDEX idx_order_items_quantity_returned ON order_items(quantity_returned);
CREATE INDEX idx_order_items_created_at ON order_items(created_at);
CREATE INDEX idx_order_items_updated_at ON order_items(updated_at);

-- Composite indexes for common queries
CREATE INDEX idx_order_items_order_status ON order_items(order_id, status);
CREATE INDEX idx_order_items_order_shipped ON order_items(order_id, quantity_shipped) WHERE quantity_shipped < quantity;
CREATE INDEX idx_order_items_product_status ON order_items(product_id, status);
CREATE INDEX idx_order_items_order_product ON order_items(order_id, product_id);

-- Full-text search indexes
CREATE INDEX idx_order_items_product_name_fts ON order_items USING gin(to_tsvector('english', product_name));

-- Create trigger for order_items table
CREATE TRIGGER trigger_order_items_updated_at
    BEFORE UPDATE ON order_items
    FOR EACH ROW
    EXECUTE FUNCTION update_companies_updated_at();

-- Add check constraints for business logic
ALTER TABLE order_items ADD CONSTRAINT check_item_total_calculation
    CHECK (total_price = (unit_price * quantity) - discount_amount + tax_amount);

ALTER TABLE order_items ADD CONSTRAINT check_item_shipped_vs_quantity
    CHECK (quantity_shipped <= quantity);

ALTER TABLE order_items ADD CONSTRAINT check_item_returned_vs_shipped
    CHECK (quantity_returned <= quantity_shipped);

-- Add comments for order_items table
COMMENT ON TABLE order_items IS 'Individual items within an order';
COMMENT ON COLUMN order_items.id IS 'Unique identifier for the order item';
COMMENT ON COLUMN order_items.order_id IS 'Reference to the parent order';
COMMENT ON COLUMN order_items.product_id IS 'Reference to the product';
COMMENT ON COLUMN order_items.product_sku IS 'Product SKU at time of order';
COMMENT ON COLUMN order_items.product_name IS 'Product name at time of order';
COMMENT ON COLUMN order_items.quantity IS 'Quantity ordered';
COMMENT ON COLUMN order_items.unit_price IS 'Unit price at time of order';
COMMENT ON COLUMN order_items.discount_amount IS 'Discount amount for this item';
COMMENT ON COLUMN order_items.tax_rate IS 'Tax rate applied to this item';
COMMENT ON COLUMN order_items.tax_amount IS 'Tax amount for this item';
COMMENT ON COLUMN order_items.total_price IS 'Total price for this item';
COMMENT ON COLUMN order_items.weight IS 'Weight per unit';
COMMENT ON COLUMN order_items.dimensions IS 'Product dimensions';
COMMENT ON COLUMN order_items.barcode IS 'Product barcode';
COMMENT ON COLUMN order_items.notes IS 'Item-specific notes';
COMMENT ON COLUMN order_items.status IS 'Current status of this item';
COMMENT ON COLUMN order_items.quantity_shipped IS 'Quantity shipped so far';
COMMENT ON COLUMN order_items.quantity_returned IS 'Quantity returned so far';
COMMENT ON COLUMN order_items.created_at IS 'Timestamp when the item was created';
COMMENT ON COLUMN order_items.updated_at IS 'Timestamp when the item was last updated';

-- Create a function to update order totals when items change
CREATE OR REPLACE FUNCTION update_order_totals_on_item_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Update order subtotal, tax, and total amounts based on items
    UPDATE orders
    SET
        subtotal = (
            SELECT COALESCE(SUM(unit_price * quantity - discount_amount), 0)
            FROM order_items
            WHERE order_id = NEW.order_id
        ),
        tax_amount = (
            SELECT COALESCE(SUM(tax_amount), 0)
            FROM order_items
            WHERE order_id = NEW.order_id
        ),
        updated_at = NOW()
    WHERE id = NEW.order_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers to update order totals
CREATE TRIGGER trigger_update_order_totals_on_insert
    AFTER INSERT ON order_items
    FOR EACH ROW
    EXECUTE FUNCTION update_order_totals_on_item_change();

CREATE TRIGGER trigger_update_order_totals_on_update
    AFTER UPDATE ON order_items
    FOR EACH ROW
    EXECUTE FUNCTION update_order_totals_on_item_change();

CREATE TRIGGER trigger_update_order_totals_on_delete
    AFTER DELETE ON order_items
    FOR EACH ROW
    EXECUTE FUNCTION update_order_totals_on_item_change();