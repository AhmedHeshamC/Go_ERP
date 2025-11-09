-- Create product_variants table
CREATE TABLE IF NOT EXISTS product_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sku VARCHAR(100) NOT NULL,
    name VARCHAR(300) NOT NULL,
    price DECIMAL(12,2) NOT NULL CHECK (price > 0),
    cost DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (cost >= 0),
    weight DECIMAL(10,3) NOT NULL DEFAULT 0 CHECK (weight >= 0),
    dimensions VARCHAR(100),
    length DECIMAL(8,2) NOT NULL DEFAULT 0 CHECK (length >= 0),
    width DECIMAL(8,2) NOT NULL DEFAULT 0 CHECK (width >= 0),
    height DECIMAL(8,2) NOT NULL DEFAULT 0 CHECK (height >= 0),
    volume DECIMAL(10,3) NOT NULL DEFAULT 0 CHECK (volume >= 0),
    barcode VARCHAR(50),
    image_url VARCHAR(1000),
    track_inventory BOOLEAN NOT NULL DEFAULT true,
    stock_quantity INTEGER NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    min_stock_level INTEGER NOT NULL DEFAULT 0 CHECK (min_stock_level >= 0),
    max_stock_level INTEGER NOT NULL DEFAULT 0 CHECK (max_stock_level >= 0),
    allow_backorder BOOLEAN NOT NULL DEFAULT false,
    requires_shipping BOOLEAN NOT NULL DEFAULT true,
    taxable BOOLEAN NOT NULL DEFAULT true,
    tax_rate DECIMAL(5,2) NOT NULL DEFAULT 0 CHECK (tax_rate >= 0 AND tax_rate <= 100),
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_digital BOOLEAN NOT NULL DEFAULT false,
    download_url VARCHAR(1000),
    max_downloads INTEGER NOT NULL DEFAULT 0 CHECK (max_downloads >= 0),
    expiry_days INTEGER NOT NULL DEFAULT 0 CHECK (expiry_days >= 0),
    sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(product_id, sku)
);

-- Create indexes
CREATE INDEX idx_product_variants_product_id ON product_variants(product_id);
CREATE INDEX idx_product_variants_sku ON product_variants(sku);
CREATE INDEX idx_product_variants_name ON product_variants USING gin(to_tsvector('english', name));
CREATE INDEX idx_product_variants_price ON product_variants(price);
CREATE INDEX idx_product_variants_is_active ON product_variants(is_active);
CREATE INDEX idx_product_variants_sort_order ON product_variants(sort_order);
CREATE INDEX idx_product_variants_barcode ON product_variants(barcode) WHERE barcode IS NOT NULL;
CREATE INDEX idx_product_variants_stock_quantity ON product_variants(stock_quantity) WHERE track_inventory = true;

-- Composite indexes for common queries
CREATE INDEX idx_product_variants_product_active ON product_variants(product_id, is_active) WHERE is_active = true;
CREATE INDEX idx_product_variants_product_sort ON product_variants(product_id, sort_order);

-- Create trigger to automatically update updated_at timestamp
CREATE TRIGGER trigger_product_variants_updated_at
    BEFORE UPDATE ON product_variants
    FOR EACH ROW
    EXECUTE FUNCTION update_products_updated_at();

-- Add check constraints for business logic (similar to products table)
ALTER TABLE product_variants ADD CONSTRAINT check_variant_cost_not_higher_than_price
    CHECK (cost <= price OR cost = 0);

ALTER TABLE product_variants ADD CONSTRAINT check_variant_digital_product_consistency
    CHECK (
        (is_digital = true AND requires_shipping = false AND download_url IS NOT NULL) OR
        (is_digital = false)
    );

ALTER TABLE product_variants ADD CONSTRAINT check_variant_inventory_levels
    CHECK (
        (max_stock_level = 0 OR min_stock_level = 0 OR max_stock_level >= min_stock_level)
    );

ALTER TABLE product_variants ADD CONSTRAINT check_variant_tax_settings
    CHECK (
        (taxable = true) OR (taxable = false AND tax_rate = 0)
    );

ALTER TABLE product_variants ADD CONSTRAINT check_variant_digital_settings
    CHECK (
        (is_digital = true AND max_downloads >= 0 AND expiry_days >= 0) OR
        (is_digital = false AND max_downloads = 0 AND expiry_days = 0)
    );

-- Add comments for documentation
COMMENT ON TABLE product_variants IS 'Product variants with different sizes, colors, etc.';
COMMENT ON COLUMN product_variants.id IS 'Unique identifier for the variant';
COMMENT ON COLUMN product_variants.product_id IS 'Reference to parent product';
COMMENT ON COLUMN product_variants.sku IS 'Stock Keeping Unit for the variant';
COMMENT ON COLUMN product_variants.name IS 'Variant display name (e.g., "Large Red Shirt")';
COMMENT ON COLUMN product_variants.price IS 'Variant selling price';
COMMENT ON COLUMN product_variants.cost IS 'Variant cost price';
COMMENT ON COLUMN product_variants.weight IS 'Variant weight in kg';
COMMENT ON COLUMN product_variants.image_url IS 'Main image URL for the variant';
COMMENT ON COLUMN product_variants.sort_order IS 'Display order among variants';

-- Create variant_attributes table
CREATE TABLE IF NOT EXISTS variant_attributes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    value VARCHAR(200) NOT NULL,
    type VARCHAR(50) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0)
);

-- Create indexes for variant attributes
CREATE INDEX idx_variant_attributes_variant_id ON variant_attributes(variant_id);
CREATE INDEX idx_variant_attributes_name ON variant_attributes(name);
CREATE INDEX idx_variant_attributes_type ON variant_attributes(type);
CREATE INDEX idx_variant_attributes_sort_order ON variant_attributes(sort_order);

-- Unique constraint to prevent duplicate attributes
CREATE UNIQUE INDEX idx_variant_attributes_variant_name_type ON variant_attributes(variant_id, name, type);

-- Add comments for variant attributes
COMMENT ON TABLE variant_attributes IS 'Attributes for product variants (color, size, material, etc.)';
COMMENT ON COLUMN variant_attributes.variant_id IS 'Reference to the product variant';
COMMENT ON COLUMN variant_attributes.name IS 'Attribute name (e.g., "Color", "Size")';
COMMENT ON COLUMN variant_attributes.value IS 'Attribute value (e.g., "Red", "Large")';
COMMENT ON COLUMN variant_attributes.type IS 'Attribute type for grouping (e.g., "color", "size")';
COMMENT ON COLUMN variant_attributes.sort_order IS 'Display order for attributes';

-- Create variant_images table
CREATE TABLE IF NOT EXISTS variant_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    image_url VARCHAR(1000) NOT NULL,
    alt_text VARCHAR(200),
    sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    is_main BOOLEAN NOT NULL DEFAULT false
);

-- Create indexes for variant images
CREATE INDEX idx_variant_images_variant_id ON variant_images(variant_id);
CREATE INDEX idx_variant_images_sort_order ON variant_images(sort_order);
CREATE INDEX idx_variant_images_is_main ON variant_images(is_main) WHERE is_main = true;

-- Ensure only one main image per variant
CREATE UNIQUE INDEX idx_variant_images_unique_main ON variant_images(variant_id) WHERE is_main = true;

-- Add comments for variant images
COMMENT ON TABLE variant_images IS 'Images for product variants';
COMMENT ON COLUMN variant_images.variant_id IS 'Reference to the product variant';
COMMENT ON COLUMN variant_images.image_url IS 'Image URL';
COMMENT ON COLUMN variant_images.alt_text IS 'Alt text for accessibility';
COMMENT ON COLUMN variant_images.sort_order IS 'Display order for images';
COMMENT ON COLUMN variant_images.is_main IS 'Whether this is the main image for the variant';

-- Create variant inventory table for multi-warehouse support
CREATE TABLE IF NOT EXISTS variant_inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    warehouse_id UUID NOT NULL,
    quantity_available INTEGER NOT NULL DEFAULT 0 CHECK (quantity_available >= 0),
    quantity_reserved INTEGER NOT NULL DEFAULT 0 CHECK (quantity_reserved >= 0),
    reorder_level INTEGER NOT NULL DEFAULT 0 CHECK (reorder_level >= 0),
    max_stock INTEGER CHECK (max_stock >= 0),
    last_updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_by UUID,
    UNIQUE(variant_id, warehouse_id)
);

-- Create indexes for variant inventory
CREATE INDEX idx_variant_inventory_variant_id ON variant_inventory(variant_id);
CREATE INDEX idx_variant_inventory_warehouse_id ON variant_inventory(warehouse_id);
CREATE INDEX idx_variant_inventory_quantity_available ON variant_inventory(quantity_available);
CREATE INDEX idx_variant_inventory_reorder_level ON variant_inventory(reorder_level) WHERE quantity_available <= reorder_level;

-- Create trigger for variant inventory
CREATE TRIGGER trigger_variant_inventory_updated_at
    BEFORE UPDATE ON variant_inventory
    FOR EACH ROW
    EXECUTE FUNCTION update_products_updated_at();

-- Add comments for variant inventory
COMMENT ON TABLE variant_inventory IS 'Variant inventory levels by warehouse';
COMMENT ON COLUMN variant_inventory.variant_id IS 'Reference to the product variant';
COMMENT ON COLUMN variant_inventory.warehouse_id IS 'Reference to the warehouse';
COMMENT ON COLUMN variant_inventory.quantity_available IS 'Available quantity for sale';
COMMENT ON COLUMN variant_inventory.quantity_reserved IS 'Quantity reserved for orders';

-- Create variant inventory transactions table
CREATE TABLE IF NOT EXISTS variant_inventory_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    warehouse_id UUID NOT NULL,
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('IN', 'OUT', 'ADJUST', 'TRANSFER')),
    quantity INTEGER NOT NULL,
    reference_id UUID,
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by UUID
);

-- Create indexes for variant inventory transactions
CREATE INDEX idx_variant_inventory_transactions_variant_id ON variant_inventory_transactions(variant_id);
CREATE INDEX idx_variant_inventory_transactions_warehouse_id ON variant_inventory_transactions(warehouse_id);
CREATE INDEX idx_variant_inventory_transactions_type ON variant_inventory_transactions(transaction_type);
CREATE INDEX idx_variant_inventory_transactions_created_at ON variant_inventory_transactions(created_at);
CREATE INDEX idx_variant_inventory_transactions_reference_id ON variant_inventory_transactions(reference_id) WHERE reference_id IS NOT NULL;

-- Add comments for variant inventory transactions
COMMENT ON TABLE variant_inventory_transactions IS 'Inventory movement transactions for variants';
COMMENT ON COLUMN variant_inventory_transactions.variant_id IS 'Reference to the product variant';
COMMENT ON COLUMN variant_inventory_transactions.transaction_type IS 'Type of transaction: IN, OUT, ADJUST, TRANSFER';
COMMENT ON COLUMN variant_inventory_transactions.quantity IS 'Quantity moved';
COMMENT ON COLUMN variant_inventory_transactions.reference_id IS 'Reference to related order or transaction';

-- Create function to validate that at least one variant exists when product has variants
CREATE OR REPLACE FUNCTION ensure_variant_consistency()
RETURNS TRIGGER AS $$
BEGIN
    -- If this is a product update and product is being set to active
    -- and it has variants, ensure at least one variant is active
    IF TG_TABLE_NAME = 'products' AND NEW.is_active = true THEN
        IF EXISTS (
            SELECT 1 FROM product_variants
            WHERE product_id = NEW.id
            LIMIT 1
        ) AND NOT EXISTS (
            SELECT 1 FROM product_variants
            WHERE product_id = NEW.id AND is_active = true
            LIMIT 1
        ) THEN
            RAISE EXCEPTION 'Cannot activate product with no active variants';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for product-variant consistency
CREATE TRIGGER trigger_ensure_variant_consistency
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION ensure_variant_consistency();