-- Create products table
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(300) NOT NULL,
    description TEXT,
    short_description VARCHAR(500),
    category_id UUID NOT NULL REFERENCES product_categories(id) ON DELETE RESTRICT,
    price DECIMAL(12,2) NOT NULL CHECK (price > 0),
    cost DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (cost >= 0),
    weight DECIMAL(10,3) NOT NULL DEFAULT 0 CHECK (weight >= 0),
    dimensions VARCHAR(100),
    length DECIMAL(8,2) NOT NULL DEFAULT 0 CHECK (length >= 0),
    width DECIMAL(8,2) NOT NULL DEFAULT 0 CHECK (width >= 0),
    height DECIMAL(8,2) NOT NULL DEFAULT 0 CHECK (height >= 0),
    volume DECIMAL(10,3) NOT NULL DEFAULT 0 CHECK (volume >= 0),
    barcode VARCHAR(50),
    track_inventory BOOLEAN NOT NULL DEFAULT true,
    stock_quantity INTEGER NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    min_stock_level INTEGER NOT NULL DEFAULT 0 CHECK (min_stock_level >= 0),
    max_stock_level INTEGER NOT NULL DEFAULT 0 CHECK (max_stock_level >= 0),
    allow_backorder BOOLEAN NOT NULL DEFAULT false,
    requires_shipping BOOLEAN NOT NULL DEFAULT true,
    taxable BOOLEAN NOT NULL DEFAULT true,
    tax_rate DECIMAL(5,2) NOT NULL DEFAULT 0 CHECK (tax_rate >= 0 AND tax_rate <= 100),
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_featured BOOLEAN NOT NULL DEFAULT false,
    is_digital BOOLEAN NOT NULL DEFAULT false,
    download_url VARCHAR(1000),
    max_downloads INTEGER NOT NULL DEFAULT 0 CHECK (max_downloads >= 0),
    expiry_days INTEGER NOT NULL DEFAULT 0 CHECK (expiry_days >= 0),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_products_sku ON products(sku);
CREATE INDEX idx_products_name ON products USING gin(to_tsvector('english', name));
CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_price ON products(price);
CREATE INDEX idx_products_is_active ON products(is_active);
CREATE INDEX idx_products_is_featured ON products(is_featured);
CREATE INDEX idx_products_is_digital ON products(is_digital);
CREATE INDEX idx_products_barcode ON products(barcode) WHERE barcode IS NOT NULL;
CREATE INDEX idx_products_stock_quantity ON products(stock_quantity) WHERE track_inventory = true;
CREATE INDEX idx_products_created_at ON products(created_at);

-- Composite indexes for common queries
CREATE INDEX idx_products_category_active ON products(category_id, is_active) WHERE is_active = true;
CREATE INDEX idx_products_active_featured ON products(is_active, is_featured) WHERE is_active = true AND is_featured = true;
CREATE INDEX idx_products_category_featured ON products(category_id, is_featured) WHERE is_featured = true;

-- Create trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_products_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_products_updated_at();

-- Add check constraints for business logic
ALTER TABLE products ADD CONSTRAINT check_cost_not_higher_than_price
    CHECK (cost <= price OR cost = 0);

ALTER TABLE products ADD CONSTRAINT check_digital_product_consistency
    CHECK (
        (is_digital = true AND requires_shipping = false AND download_url IS NOT NULL) OR
        (is_digital = false)
    );

ALTER TABLE products ADD CONSTRAINT check_inventory_levels
    CHECK (
        (max_stock_level = 0 OR min_stock_level = 0 OR max_stock_level >= min_stock_level)
    );

ALTER TABLE products ADD CONSTRAINT check_tax_settings
    CHECK (
        (taxable = true) OR (taxable = false AND tax_rate = 0)
    );

ALTER TABLE products ADD CONSTRAINT check_digital_settings
    CHECK (
        (is_digital = true AND max_downloads >= 0 AND expiry_days >= 0) OR
        (is_digital = false AND max_downloads = 0 AND expiry_days = 0)
    );

-- Add comments for documentation
COMMENT ON TABLE products IS 'Main products table containing all product information';
COMMENT ON COLUMN products.id IS 'Unique identifier for the product';
COMMENT ON COLUMN products.sku IS 'Stock Keeping Unit - unique product identifier';
COMMENT ON COLUMN products.name IS 'Product display name';
COMMENT ON COLUMN products.description IS 'Detailed product description';
COMMENT ON COLUMN products.short_description IS 'Brief product description for listings';
COMMENT ON COLUMN products.category_id IS 'Reference to product category';
COMMENT ON COLUMN products.price IS 'Selling price of the product';
COMMENT ON COLUMN products.cost IS 'Cost price of the product';
COMMENT ON COLUMN products.weight IS 'Product weight in kg';
COMMENT ON COLUMN products.dimensions IS 'Human-readable dimensions (e.g., "10 x 5 x 3")';
COMMENT ON COLUMN products.length IS 'Product length in cm';
COMMENT ON COLUMN products.width IS 'Product width in cm';
COMMENT ON COLUMN products.height IS 'Product height in cm';
COMMENT ON COLUMN products.volume IS 'Product volume in cubic cm';
COMMENT ON COLUMN products.barcode IS 'Product barcode for inventory management';
COMMENT ON COLUMN products.track_inventory IS 'Whether to track inventory for this product';
COMMENT ON COLUMN products.stock_quantity IS 'Current stock quantity';
COMMENT ON COLUMN products.min_stock_level IS 'Minimum stock level before reorder';
COMMENT ON COLUMN products.max_stock_level IS 'Maximum stock level to hold';
COMMENT ON COLUMN products.allow_backorder IS 'Whether backorders are allowed';
COMMENT ON COLUMN products.requires_shipping IS 'Whether product requires shipping';
COMMENT ON COLUMN products.taxable IS 'Whether product is taxable';
COMMENT ON COLUMN products.tax_rate IS 'Tax rate percentage';
COMMENT ON COLUMN products.is_active IS 'Whether product is currently active/sellable';
COMMENT ON COLUMN products.is_featured IS 'Whether product is featured';
COMMENT ON COLUMN products.is_digital IS 'Whether product is digital';
COMMENT ON COLUMN products.download_url IS 'Download URL for digital products';
COMMENT ON COLUMN products.max_downloads IS 'Maximum download attempts for digital products';
COMMENT ON COLUMN products.expiry_days IS 'Download expiry in days for digital products';
COMMENT ON COLUMN products.created_at IS 'Timestamp when the product was created';
COMMENT ON COLUMN products.updated_at IS 'Timestamp when the product was last updated';

-- Create product inventory table for multi-warehouse support
CREATE TABLE IF NOT EXISTS product_inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    warehouse_id UUID NOT NULL,
    quantity_available INTEGER NOT NULL DEFAULT 0 CHECK (quantity_available >= 0),
    quantity_reserved INTEGER NOT NULL DEFAULT 0 CHECK (quantity_reserved >= 0),
    reorder_level INTEGER NOT NULL DEFAULT 0 CHECK (reorder_level >= 0),
    max_stock INTEGER CHECK (max_stock >= 0),
    last_updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_by UUID,
    UNIQUE(product_id, warehouse_id)
);

-- Create indexes for inventory table
CREATE INDEX idx_product_inventory_product_id ON product_inventory(product_id);
CREATE INDEX idx_product_inventory_warehouse_id ON product_inventory(warehouse_id);
CREATE INDEX idx_product_inventory_quantity_available ON product_inventory(quantity_available);
CREATE INDEX idx_product_inventory_reorder_level ON product_inventory(reorder_level) WHERE quantity_available <= reorder_level;

-- Create trigger for inventory table
CREATE TRIGGER trigger_product_inventory_updated_at
    BEFORE UPDATE ON product_inventory
    FOR EACH ROW
    EXECUTE FUNCTION update_products_updated_at();

-- Add comments for inventory table
COMMENT ON TABLE product_inventory IS 'Product inventory levels by warehouse';
COMMENT ON COLUMN product_inventory.product_id IS 'Reference to the product';
COMMENT ON COLUMN product_inventory.warehouse_id IS 'Reference to the warehouse';
COMMENT ON COLUMN product_inventory.quantity_available IS 'Available quantity for sale';
COMMENT ON COLUMN product_inventory.quantity_reserved IS 'Quantity reserved for orders';
COMMENT ON COLUMN product_inventory.reorder_level IS 'Reorder threshold';
COMMENT ON COLUMN product_inventory.max_stock IS 'Maximum stock to maintain';
COMMENT ON COLUMN product_inventory.last_updated_at IS 'Last update timestamp';
COMMENT ON COLUMN product_inventory.updated_by IS 'User who last updated the inventory';

-- Create inventory transactions table for tracking stock movements
CREATE TABLE IF NOT EXISTS inventory_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    warehouse_id UUID NOT NULL,
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('IN', 'OUT', 'ADJUST', 'TRANSFER')),
    quantity INTEGER NOT NULL,
    reference_id UUID,
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by UUID
);

-- Create indexes for transactions table
CREATE INDEX idx_inventory_transactions_product_id ON inventory_transactions(product_id);
CREATE INDEX idx_inventory_transactions_warehouse_id ON inventory_transactions(warehouse_id);
CREATE INDEX idx_inventory_transactions_type ON inventory_transactions(transaction_type);
CREATE INDEX idx_inventory_transactions_created_at ON inventory_transactions(created_at);
CREATE INDEX idx_inventory_transactions_reference_id ON inventory_transactions(reference_id) WHERE reference_id IS NOT NULL;

-- Add comments for transactions table
COMMENT ON TABLE inventory_transactions IS 'Inventory movement transactions';
COMMENT ON COLUMN inventory_transactions.product_id IS 'Reference to the product';
COMMENT ON COLUMN inventory_transactions.warehouse_id IS 'Reference to the warehouse';
COMMENT ON COLUMN inventory_transactions.transaction_type IS 'Type of transaction: IN, OUT, ADJUST, TRANSFER';
COMMENT ON COLUMN inventory_transactions.quantity IS 'Quantity moved (positive for IN, negative for OUT)';
COMMENT ON COLUMN inventory_transactions.reference_id IS 'Reference to related order or transaction';
COMMENT ON COLUMN inventory_transactions.reason IS 'Reason for the inventory movement';
COMMENT ON COLUMN inventory_transactions.created_at IS 'Transaction timestamp';
COMMENT ON COLUMN inventory_transactions.created_by IS 'User who created the transaction';