-- Create product_categories table
CREATE TABLE IF NOT EXISTS product_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    parent_id UUID REFERENCES product_categories(id) ON DELETE SET NULL,
    level INTEGER NOT NULL DEFAULT 0 CHECK (level >= 0),
    path VARCHAR(500) NOT NULL,
    image_url VARCHAR(500),
    sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_product_categories_parent_id ON product_categories(parent_id);
CREATE INDEX idx_product_categories_path ON product_categories(path);
CREATE INDEX idx_product_categories_level ON product_categories(level);
CREATE INDEX idx_product_categories_is_active ON product_categories(is_active);
CREATE INDEX idx_product_categories_sort_order ON product_categories(sort_order);

-- Create unique constraint on name per parent (to avoid duplicate names in the same category)
CREATE UNIQUE INDEX idx_product_categories_name_parent_id ON product_categories(name, parent_id) WHERE is_active = true;

-- Create unique constraint on path (to avoid duplicate paths)
CREATE UNIQUE INDEX idx_product_categories_unique_path ON product_categories(path);

-- Create trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_product_categories_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_product_categories_updated_at
    BEFORE UPDATE ON product_categories
    FOR EACH ROW
    EXECUTE FUNCTION update_product_categories_updated_at();

-- Add comments for documentation
COMMENT ON TABLE product_categories IS 'Product categories with hierarchical structure';
COMMENT ON COLUMN product_categories.id IS 'Unique identifier for the product category';
COMMENT ON COLUMN product_categories.name IS 'Display name of the category';
COMMENT ON COLUMN product_categories.description IS 'Detailed description of the category';
COMMENT ON COLUMN product_categories.parent_id IS 'Parent category ID for hierarchical structure';
COMMENT ON COLUMN product_categories.level IS 'Depth level in the category hierarchy (0 = root)';
COMMENT ON COLUMN product_categories.path IS 'URL-friendly path for the category (e.g., /electronics/computers)';
COMMENT ON COLUMN product_categories.image_url IS 'URL to the category image';
COMMENT ON COLUMN product_categories.sort_order IS 'Display order for categories at the same level';
COMMENT ON COLUMN product_categories.is_active IS 'Whether the category is currently active';
COMMENT ON COLUMN product_categories.created_at IS 'Timestamp when the category was created';
COMMENT ON COLUMN product_categories.updated_at IS 'Timestamp when the category was last updated';

-- Create category metadata table
CREATE TABLE IF NOT EXISTS category_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL REFERENCES product_categories(id) ON DELETE CASCADE,
    seo_title VARCHAR(200),
    seo_description VARCHAR(300),
    seo_keywords VARCHAR(500),
    meta_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for metadata table
CREATE UNIQUE INDEX idx_category_metadata_category_id ON category_metadata(category_id);

-- Create trigger for metadata table
CREATE TRIGGER trigger_category_metadata_updated_at
    BEFORE UPDATE ON category_metadata
    FOR EACH ROW
    EXECUTE FUNCTION update_product_categories_updated_at();

-- Add comments for metadata table
COMMENT ON TABLE category_metadata IS 'SEO and metadata information for product categories';
COMMENT ON COLUMN category_metadata.category_id IS 'Reference to the product category';
COMMENT ON COLUMN category_metadata.seo_title IS 'SEO title for the category page';
COMMENT ON COLUMN category_metadata.seo_description IS 'SEO description for search engines';
COMMENT ON COLUMN category_metadata.seo_keywords IS 'SEO keywords for search engines';
COMMENT ON COLUMN category_metadata.meta_data IS 'Additional metadata as JSONB';

-- Insert default root category if it doesn't exist
INSERT INTO product_categories (id, name, description, level, path, is_active)
SELECT gen_random_uuid(), 'Root Category', 'Root category for all products', 0, '/root', true
WHERE NOT EXISTS (SELECT 1 FROM product_categories WHERE path = '/root');