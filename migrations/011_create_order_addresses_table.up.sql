-- Create order addresses table
CREATE TABLE IF NOT EXISTS order_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
    order_id UUID, -- Will be linked to orders table later
    type VARCHAR(20) NOT NULL CHECK (type IN ('SHIPPING', 'BILLING', 'BOTH')),

    -- Address fields
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    company VARCHAR(200),
    address_line_1 VARCHAR(255) NOT NULL,
    address_line_2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    country VARCHAR(100) NOT NULL,

    -- Additional fields
    phone VARCHAR(50),
    email VARCHAR(255),
    instructions TEXT,

    -- Metadata
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_validated BOOLEAN NOT NULL DEFAULT false,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Ensure address is linked to either customer or order
    CONSTRAINT check_address_link CHECK (customer_id IS NOT NULL OR order_id IS NOT NULL)
);

-- Create indexes for order_addresses table
CREATE INDEX idx_order_addresses_customer_id ON order_addresses(customer_id);
CREATE INDEX idx_order_addresses_order_id ON order_addresses(order_id) WHERE order_id IS NOT NULL;
CREATE INDEX idx_order_addresses_type ON order_addresses(type);
CREATE INDEX idx_order_addresses_first_name ON order_addresses(first_name);
CREATE INDEX idx_order_addresses_last_name ON order_addresses(last_name);
CREATE INDEX idx_order_addresses_city ON order_addresses(city);
CREATE INDEX idx_order_addresses_state ON order_addresses(state);
CREATE INDEX idx_order_addresses_country ON order_addresses(country);
CREATE INDEX idx_order_addresses_postal_code ON order_addresses(postal_code);
CREATE INDEX idx_order_addresses_is_default ON order_addresses(is_default);
CREATE INDEX idx_order_addresses_is_active ON order_addresses(is_active);
CREATE INDEX idx_order_addresses_is_validated ON order_addresses(is_validated);
CREATE INDEX idx_order_addresses_created_at ON order_addresses(created_at);

-- Composite indexes for common queries
CREATE INDEX idx_order_addresses_customer_type ON order_addresses(customer_id, type) WHERE customer_id IS NOT NULL;
CREATE INDEX idx_order_addresses_customer_default ON order_addresses(customer_id, is_default) WHERE customer_id IS NOT NULL AND is_default = true;
CREATE INDEX idx_order_addresses_order_type ON order_addresses(order_id, type) WHERE order_id IS NOT NULL;

-- Unique constraints to prevent duplicate default addresses per customer/type
CREATE UNIQUE INDEX idx_order_addresses_customer_shipping_default ON order_addresses(customer_id)
WHERE customer_id IS NOT NULL AND type IN ('SHIPPING', 'BOTH') AND is_default = true;

CREATE UNIQUE INDEX idx_order_addresses_customer_billing_default ON order_addresses(customer_id)
WHERE customer_id IS NOT NULL AND type IN ('BILLING', 'BOTH') AND is_default = true;

-- Create trigger for order_addresses table
CREATE TRIGGER trigger_order_addresses_updated_at
    BEFORE UPDATE ON order_addresses
    FOR EACH ROW
    EXECUTE FUNCTION update_companies_updated_at();

-- Add comments for order_addresses table
COMMENT ON TABLE order_addresses IS 'Shipping and billing addresses for customers and orders';
COMMENT ON COLUMN order_addresses.id IS 'Unique identifier for the address';
COMMENT ON COLUMN order_addresses.customer_id IS 'Reference to customer (for saved addresses)';
COMMENT ON COLUMN order_addresses.order_id IS 'Reference to order (for order-specific addresses)';
COMMENT ON COLUMN order_addresses.type IS 'Address type: SHIPPING, BILLING, BOTH';
COMMENT ON COLUMN order_addresses.first_name IS 'Recipient first name';
COMMENT ON COLUMN order_addresses.last_name IS 'Recipient last name';
COMMENT ON COLUMN order_addresses.company IS 'Company name (optional)';
COMMENT ON COLUMN order_addresses.address_line_1 IS 'Street address line 1';
COMMENT ON COLUMN order_addresses.address_line_2 IS 'Street address line 2 (optional)';
COMMENT ON COLUMN order_addresses.city IS 'City name';
COMMENT ON COLUMN order_addresses.state IS 'State or province';
COMMENT ON COLUMN order_addresses.postal_code IS 'Postal or ZIP code';
COMMENT ON COLUMN order_addresses.country IS 'Country name';
COMMENT ON COLUMN order_addresses.phone IS 'Contact phone number';
COMMENT ON COLUMN order_addresses.email IS 'Contact email address';
COMMENT ON COLUMN order_addresses.instructions IS 'Delivery instructions';
COMMENT ON COLUMN order_addresses.is_default IS 'Whether this is the default address for the type';
COMMENT ON COLUMN order_addresses.is_active IS 'Whether the address is active';
COMMENT ON COLUMN order_addresses.is_validated IS 'Whether the address has been validated';
COMMENT ON COLUMN order_addresses.created_at IS 'Timestamp when the address was created';
COMMENT ON COLUMN order_addresses.updated_at IS 'Timestamp when the address was last updated';