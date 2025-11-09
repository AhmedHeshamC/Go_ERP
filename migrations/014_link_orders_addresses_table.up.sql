-- Add foreign key constraint from order_addresses to orders table
ALTER TABLE order_addresses
ADD CONSTRAINT fk_order_addresses_order_id
FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE;

-- Add indexes for improved query performance
CREATE INDEX IF NOT EXISTS idx_order_addresses_customer_id_active ON order_addresses(customer_id, is_active) WHERE customer_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_order_addresses_order_id_active ON order_addresses(order_id, is_active) WHERE order_id IS NOT NULL;

-- Create a view for customer addresses (addresses not linked to specific orders)
CREATE OR REPLACE VIEW customer_addresses AS
SELECT
    oa.*,
    c.customer_code,
    CASE
        WHEN oa.company IS NOT NULL AND oa.company != '' THEN oa.company
        WHEN c.company_name IS NOT NULL AND c.company_name != '' THEN c.company_name
        ELSE c.first_name || ' ' || c.last_name
    END as display_name
FROM order_addresses oa
INNER JOIN customers c ON oa.customer_id = c.id
WHERE oa.order_id IS NULL AND oa.is_active = true;

-- Create a view for order addresses (addresses linked to specific orders)
CREATE OR REPLACE VIEW order_shipping_addresses AS
SELECT
    oa.*,
    o.order_number,
    o.customer_id,
    c.customer_code,
    CASE
        WHEN oa.company IS NOT NULL AND oa.company != '' THEN oa.company
        WHEN c.company_name IS NOT NULL AND c.company_name != '' THEN c.company_name
        ELSE c.first_name || ' ' || c.last_name
    END as display_name
FROM order_addresses oa
INNER JOIN orders o ON oa.order_id = o.id
INNER JOIN customers c ON o.customer_id = c.id
WHERE oa.type IN ('SHIPPING', 'BOTH') AND oa.is_active = true;

-- Create a view for order billing addresses
CREATE OR REPLACE VIEW order_billing_addresses AS
SELECT
    oa.*,
    o.order_number,
    o.customer_id,
    c.customer_code,
    CASE
        WHEN oa.company IS NOT NULL AND oa.company != '' THEN oa.company
        WHEN c.company_name IS NOT NULL AND c.company_name != '' THEN c.company_name
        ELSE c.first_name || ' ' || c.last_name
    END as display_name
FROM order_addresses oa
INNER JOIN orders o ON oa.order_id = o.id
INNER JOIN customers c ON o.customer_id = c.id
WHERE oa.type IN ('BILLING', 'BOTH') AND oa.is_active = true;

-- Add product foreign key constraint to order_items (linking to existing products table)
ALTER TABLE order_items
ADD CONSTRAINT fk_order_items_product_id
FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT;

-- Create a comprehensive order summary view
CREATE OR REPLACE VIEW order_summary AS
SELECT
    o.*,
    c.customer_code,
    CASE
        WHEN c.company_name IS NOT NULL AND c.company_name != '' THEN c.company_name
        ELSE c.first_name || ' ' || c.last_name
    END as customer_name,
    c.email as customer_email,
    c.phone as customer_phone,
    c.type as customer_type,
    sa.first_name || ' ' || sa.last_name as shipping_contact_name,
    sa.company as shipping_company,
    sa.address_line_1 || ', ' || COALESCE(sa.address_line_2 || ', ', '') || sa.city || ', ' || sa.state || ' ' || sa.postal_code as shipping_address,
    sa.country as shipping_country,
    ba.first_name || ' ' || ba.last_name as billing_contact_name,
    ba.company as billing_company,
    ba.address_line_1 || ', ' || COALESCE(ba.address_line_2 || ', ', '') || ba.city || ', ' || ba.state || ' ' || ba.postal_code as billing_address,
    ba.country as billing_country,
    COALESCE(oi.item_count, 0) as item_count,
    CASE
        WHEN o.paid_amount >= o.total_amount THEN 'PAID'
        WHEN o.paid_amount > 0 THEN 'PARTIALLY_PAID'
        ELSE 'UNPAID'
    END as payment_status_display,
    CASE
        WHEN o.status IN ('DELIVERED', 'CANCELLED', 'REFUNDED') THEN 'COMPLETED'
        WHEN o.status = 'SHIPPED' THEN 'IN_TRANSIT'
        ELSE 'PENDING'
    END as order_stage
FROM orders o
INNER JOIN customers c ON o.customer_id = c.id
INNER JOIN order_addresses sa ON o.shipping_address_id = sa.id
INNER JOIN order_addresses ba ON o.billing_address_id = ba.id
LEFT JOIN (
    SELECT
        order_id,
        COUNT(*) as item_count,
        SUM(quantity) as total_quantity,
        SUM(total_price) as items_total
    FROM order_items
    GROUP BY order_id
) oi ON oi.order_id = o.id;

-- Create an order items detailed view with product information
CREATE OR REPLACE VIEW order_items_detail AS
SELECT
    oi.*,
    o.order_number,
    o.customer_id,
    c.customer_code,
    p.name as current_product_name,
    p.sku as current_product_sku,
    p.is_active as product_is_active,
    p.track_inventory as product_track_inventory,
    p.stock_quantity as product_stock_quantity,
    CASE
        WHEN p.track_inventory AND p.stock_quantity >= oi.quantity THEN 'IN_STOCK'
        WHEN p.track_inventory AND p.stock_quantity > 0 THEN 'LOW_STOCK'
        WHEN p.track_inventory THEN 'OUT_OF_STOCK'
        ELSE 'NOT_TRACKED'
    END as inventory_status,
    CASE
        WHEN oi.quantity_shipped = oi.quantity THEN 'FULLY_SHIPPED'
        WHEN oi.quantity_shipped > 0 THEN 'PARTIALLY_SHIPPED'
        ELSE 'NOT_SHIPPED'
    END as shipping_status,
    CASE
        WHEN oi.quantity_returned = oi.quantity_shipped THEN 'FULLY_RETURNED'
        WHEN oi.quantity_returned > 0 THEN 'PARTIALLY_RETURNED'
        ELSE 'NOT_RETURNED'
    END as return_status
FROM order_items oi
INNER JOIN orders o ON oi.order_id = o.id
INNER JOIN customers c ON o.customer_id = c.id
LEFT JOIN products p ON oi.product_id = p.id;

-- Add comments for views
COMMENT ON VIEW customer_addresses IS 'Customer addresses that can be reused for orders';
COMMENT ON VIEW order_shipping_addresses IS 'Shipping addresses for specific orders';
COMMENT ON VIEW order_billing_addresses IS 'Billing addresses for specific orders';
COMMENT ON VIEW order_summary IS 'Comprehensive order summary with customer and address information';
COMMENT ON VIEW order_items_detail IS 'Detailed order items with current product information';

-- Create functions for common address operations
CREATE OR REPLACE FUNCTION get_customer_default_addresses(p_customer_id UUID)
RETURNS TABLE (
    shipping_address_id UUID,
    billing_address_id UUID,
    shipping_full_address TEXT,
    billing_full_address TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        sa.id as shipping_address_id,
        ba.id as billing_address_id,
        sa.first_name || ' ' || sa.last_name || COALESCE(', ' || sa.company, '') || E'\n' ||
        sa.address_line_1 || COALESCE(E'\n' || sa.address_line_2, '') || E'\n' ||
        sa.city || ', ' || sa.state || ' ' || sa.postal_code || E'\n' ||
        sa.country as shipping_full_address,
        ba.first_name || ' ' || ba.last_name || COALESCE(', ' || ba.company, '') || E'\n' ||
        ba.address_line_1 || COALESCE(E'\n' || ba.address_line_2, '') || E'\n' ||
        ba.city || ', ' || ba.state || ' ' || ba.postal_code || E'\n' ||
        ba.country as billing_full_address
    FROM customers c
    LEFT JOIN LATERAL (
        SELECT id, first_name, last_name, company, address_line_1, address_line_2, city, state, postal_code, country
        FROM order_addresses
        WHERE customer_id = p_customer_id
        AND type IN ('SHIPPING', 'BOTH')
        AND is_default = true
        AND is_active = true
        LIMIT 1
    ) sa ON true
    LEFT JOIN LATERAL (
        SELECT id, first_name, last_name, company, address_line_1, address_line_2, city, state, postal_code, country
        FROM order_addresses
        WHERE customer_id = p_customer_id
        AND type IN ('BILLING', 'BOTH')
        AND is_default = true
        AND is_active = true
        LIMIT 1
    ) ba ON true
    WHERE c.id = p_customer_id;
END;
$$ LANGUAGE plpgsql;

-- Create a function to validate order number format
CREATE OR REPLACE FUNCTION validate_order_number(p_order_number VARCHAR)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN p_order_number ~ '^\d{4}-\d{6}$';
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Create a function to generate next order number
CREATE OR REPLACE FUNCTION generate_next_order_number()
RETURNS VARCHAR AS $$
DECLARE
    v_year INTEGER := EXTRACT(YEAR FROM NOW());
    v_sequence INTEGER;
    v_order_number VARCHAR;
BEGIN
    -- Get the last sequence number for this year
    SELECT COALESCE(MAX(CAST(SPLIT_PART(order_number, '-', 2) AS INTEGER)), 0) + 1
    INTO v_sequence
    FROM orders
    WHERE order_number LIKE v_year || '-%';

    -- Format the order number
    v_order_number := v_year || '-' || LPAD(v_sequence::TEXT, 6, '0');

    RETURN v_order_number;
END;
$$ LANGUAGE plpgsql;