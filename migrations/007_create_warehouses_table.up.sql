-- Create warehouses table
CREATE TABLE IF NOT EXISTS warehouses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    code VARCHAR(20) NOT NULL UNIQUE,
    address TEXT NOT NULL,
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100),
    country VARCHAR(100) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    phone VARCHAR(20),
    email VARCHAR(255),
    manager_id UUID REFERENCES users(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for warehouses table
CREATE INDEX idx_warehouses_code ON warehouses(code);
CREATE INDEX idx_warehouses_name ON warehouses USING gin(to_tsvector('english', name));
CREATE INDEX idx_warehouses_city ON warehouses(city);
CREATE INDEX idx_warehouses_state ON warehouses(state) WHERE state IS NOT NULL;
CREATE INDEX idx_warehouses_country ON warehouses(country);
CREATE INDEX idx_warehouses_postal_code ON warehouses(postal_code);
CREATE INDEX idx_warehouses_is_active ON warehouses(is_active);
CREATE INDEX idx_warehouses_manager_id ON warehouses(manager_id) WHERE manager_id IS NOT NULL;
CREATE INDEX idx_warehouses_created_at ON warehouses(created_at);

-- Composite indexes for common queries
CREATE INDEX idx_warehouses_active_manager ON warehouses(is_active, manager_id) WHERE is_active = true AND manager_id IS NOT NULL;
CREATE INDEX idx_warehouses_location_active ON warehouses(country, city, is_active) WHERE is_active = true;

-- Create trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_warehouses_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_warehouses_updated_at
    BEFORE UPDATE ON warehouses
    FOR EACH ROW
    EXECUTE FUNCTION update_warehouses_updated_at();

-- Add check constraints for business logic
ALTER TABLE warehouses ADD CONSTRAINT check_warehouse_code_format
    CHECK (code ~ '^[A-Z0-9\-_]+$');

ALTER TABLE warehouses ADD CONSTRAINT check_warehouse_phone_format
    CHECK (phone IS NULL OR phone ~ '^\+?[\d\s\-\(\)]{7,20}$');

ALTER TABLE warehouses ADD CONSTRAINT check_warehouse_email_format
    CHECK (email IS NULL OR email ~ '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$');

-- Add comments for documentation
COMMENT ON TABLE warehouses IS 'Warehouses and storage locations for inventory management';
COMMENT ON COLUMN warehouses.id IS 'Unique identifier for the warehouse';
COMMENT ON COLUMN warehouses.name IS 'Warehouse display name';
COMMENT ON COLUMN warehouses.code IS 'Unique warehouse code (uppercase alphanumeric)';
COMMENT ON COLUMN warehouses.address IS 'Street address of the warehouse';
COMMENT ON COLUMN warehouses.city IS 'City where the warehouse is located';
COMMENT ON COLUMN warehouses.state IS 'State or province where the warehouse is located';
COMMENT ON COLUMN warehouses.country IS 'Country where the warehouse is located';
COMMENT ON COLUMN warehouses.postal_code IS 'Postal code or ZIP code';
COMMENT ON COLUMN warehouses.phone IS 'Contact phone number for the warehouse';
COMMENT ON COLUMN warehouses.email IS 'Contact email for the warehouse';
COMMENT ON COLUMN warehouses.manager_id IS 'Reference to the warehouse manager (user)';
COMMENT ON COLUMN warehouses.is_active IS 'Whether the warehouse is currently active';
COMMENT ON COLUMN warehouses.created_at IS 'Timestamp when the warehouse was created';
COMMENT ON COLUMN warehouses.updated_at IS 'Timestamp when the warehouse was last updated';

-- Create warehouse_types enum for extended warehouse functionality
CREATE TYPE warehouse_type AS ENUM (
    'RETAIL',
    'WHOLESALE',
    'DISTRIBUTION',
    'FULFILLMENT',
    'RETURN'
);

-- Create warehouses_extended table for additional warehouse metadata
CREATE TABLE IF NOT EXISTS warehouses_extended (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    type warehouse_type NOT NULL DEFAULT 'RETAIL',
    capacity INTEGER CHECK (capacity >= 0),
    square_footage INTEGER CHECK (square_footage >= 0),
    dock_count INTEGER CHECK (dock_count >= 0),
    temperature_controlled BOOLEAN NOT NULL DEFAULT false,
    security_level INTEGER NOT NULL DEFAULT 0 CHECK (security_level >= 0 AND security_level <= 10),
    description TEXT,
    operating_hours JSONB, -- Store operating hours as JSON
    special_instructions TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(warehouse_id)
);

-- Create indexes for warehouses_extended table
CREATE INDEX idx_warehouses_extended_warehouse_id ON warehouses_extended(warehouse_id);
CREATE INDEX idx_warehouses_extended_type ON warehouses_extended(type);
CREATE INDEX idx_warehouses_extended_capacity ON warehouses_extended(capacity) WHERE capacity IS NOT NULL;
CREATE INDEX idx_warehouses_extended_security_level ON warehouses_extended(security_level);
CREATE INDEX idx_warehouses_extended_temperature_controlled ON warehouses_extended(temperature_controlled);

-- Create trigger for warehouses_extended table
CREATE TRIGGER trigger_warehouses_extended_updated_at
    BEFORE UPDATE ON warehouses_extended
    FOR EACH ROW
    EXECUTE FUNCTION update_warehouses_updated_at();

-- Add comments for warehouses_extended table
COMMENT ON TABLE warehouses_extended IS 'Extended warehouse information and metadata';
COMMENT ON COLUMN warehouses_extended.id IS 'Unique identifier for the extended warehouse record';
COMMENT ON COLUMN warehouses_extended.warehouse_id IS 'Reference to the base warehouse';
COMMENT ON COLUMN warehouses_extended.type IS 'Type of warehouse (RETAIL, WHOLESALE, etc.)';
COMMENT ON COLUMN warehouses_extended.capacity IS 'Maximum storage capacity (units)';
COMMENT ON COLUMN warehouses_extended.square_footage IS 'Total square footage of the warehouse';
COMMENT ON COLUMN warehouses_extended.dock_count IS 'Number of loading docks';
COMMENT ON COLUMN warehouses_extended.temperature_controlled IS 'Whether the warehouse has temperature control';
COMMENT ON COLUMN warehouses_extended.security_level IS 'Security level (0-10)';
COMMENT ON COLUMN warehouses_extended.description IS 'Detailed description of the warehouse';
COMMENT ON COLUMN warehouses_extended.operating_hours IS 'Operating hours stored as JSON';
COMMENT ON COLUMN warehouses_extended.special_instructions IS 'Special handling instructions';
COMMENT ON COLUMN warehouses_extended.created_at IS 'Timestamp when the extended record was created';
COMMENT ON COLUMN warehouses_extended.updated_at IS 'Timestamp when the extended record was last updated';

-- Create warehouse_contacts table for multiple contacts per warehouse
CREATE TABLE IF NOT EXISTS warehouse_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    title VARCHAR(100),
    phone VARCHAR(20),
    email VARCHAR(255),
    is_primary BOOLEAN NOT NULL DEFAULT false,
    contact_type VARCHAR(50) NOT NULL DEFAULT 'GENERAL', -- GENERAL, SHIPPING, RECEIVING, SECURITY, etc.
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for warehouse_contacts table
CREATE INDEX idx_warehouse_contacts_warehouse_id ON warehouse_contacts(warehouse_id);
CREATE INDEX idx_warehouse_contacts_is_primary ON warehouse_contacts(is_primary);
CREATE INDEX idx_warehouse_contacts_contact_type ON warehouse_contacts(contact_type);
CREATE INDEX idx_warehouse_contacts_email ON warehouse_contacts(email) WHERE email IS NOT NULL;

-- Create trigger for warehouse_contacts table
CREATE TRIGGER trigger_warehouse_contacts_updated_at
    BEFORE UPDATE ON warehouse_contacts
    FOR EACH ROW
    EXECUTE FUNCTION update_warehouses_updated_at();

-- Add check constraint to ensure only one primary contact per warehouse
CREATE UNIQUE INDEX idx_warehouse_contacts_unique_primary
ON warehouse_contacts(warehouse_id)
WHERE is_primary = true;

-- Add check constraints for contact information
ALTER TABLE warehouse_contacts ADD CONSTRAINT check_warehouse_contact_phone_format
    CHECK (phone IS NULL OR phone ~ '^\+?[\d\s\-\(\)]{7,20}$');

ALTER TABLE warehouse_contacts ADD CONSTRAINT check_warehouse_contact_email_format
    CHECK (email IS NULL OR email ~ '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$');

-- Add comments for warehouse_contacts table
COMMENT ON TABLE warehouse_contacts IS 'Contact information for warehouse personnel';
COMMENT ON COLUMN warehouse_contacts.id IS 'Unique identifier for the contact';
COMMENT ON COLUMN warehouse_contacts.warehouse_id IS 'Reference to the warehouse';
COMMENT ON COLUMN warehouse_contacts.name IS 'Contact person name';
COMMENT ON COLUMN warehouse_contacts.title IS 'Contact person job title';
COMMENT ON COLUMN warehouse_contacts.phone IS 'Contact phone number';
COMMENT ON COLUMN warehouse_contacts.email IS 'Contact email address';
COMMENT ON COLUMN warehouse_contacts.is_primary IS 'Whether this is the primary contact for the warehouse';
COMMENT ON COLUMN warehouse_contacts.contact_type IS 'Type of contact (GENERAL, SHIPPING, etc.)';
COMMENT ON COLUMN warehouse_contacts.created_at IS 'Timestamp when the contact was created';
COMMENT ON COLUMN warehouse_contacts.updated_at IS 'Timestamp when the contact was last updated';

-- Create warehouse_zones table for zone management within warehouses
CREATE TABLE IF NOT EXISTS warehouse_zones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(20) NOT NULL,
    zone_type VARCHAR(50) NOT NULL DEFAULT 'STORAGE', -- STORAGE, PICKING, RECEIVING, SHIPPING, etc.
    description TEXT,
    capacity INTEGER CHECK (capacity >= 0),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(warehouse_id, code)
);

-- Create indexes for warehouse_zones table
CREATE INDEX idx_warehouse_zones_warehouse_id ON warehouse_zones(warehouse_id);
CREATE INDEX idx_warehouse_zones_code ON warehouse_zones(code);
CREATE INDEX idx_warehouse_zones_type ON warehouse_zones(zone_type);
CREATE INDEX idx_warehouse_zones_is_active ON warehouse_zones(is_active);

-- Create trigger for warehouse_zones table
CREATE TRIGGER trigger_warehouse_zones_updated_at
    BEFORE UPDATE ON warehouse_zones
    FOR EACH ROW
    EXECUTE FUNCTION update_warehouses_updated_at();

-- Add comments for warehouse_zones table
COMMENT ON TABLE warehouse_zones IS 'Zones or areas within warehouses for organization';
COMMENT ON COLUMN warehouse_zones.id IS 'Unique identifier for the zone';
COMMENT ON COLUMN warehouse_zones.warehouse_id IS 'Reference to the warehouse';
COMMENT ON COLUMN warehouse_zones.name IS 'Zone display name';
COMMENT ON COLUMN warehouse_zones.code IS 'Zone code (unique within warehouse)';
COMMENT ON COLUMN warehouse_zones.zone_type IS 'Type of zone (STORAGE, PICKING, etc.)';
COMMENT ON COLUMN warehouse_zones.description IS 'Description of the zone';
COMMENT ON COLUMN warehouse_zones.capacity IS 'Maximum capacity for the zone';
COMMENT ON COLUMN warehouse_zones.is_active IS 'Whether the zone is currently active';
COMMENT ON COLUMN warehouse_zones.created_at IS 'Timestamp when the zone was created';
COMMENT ON COLUMN warehouse_zones.updated_at IS 'Timestamp when the zone was last updated';

-- Insert default warehouse types and provide sample data
-- This is optional and can be removed if not needed for initial setup
DO $$
BEGIN
    -- Insert a default warehouse if no warehouses exist
    IF NOT EXISTS (SELECT 1 FROM warehouses LIMIT 1) THEN
        INSERT INTO warehouses (name, code, address, city, state, country, postal_code, is_active)
        VALUES (
            'Main Warehouse',
            'WH001',
            '1000 Warehouse Boulevard',
            'Atlanta',
            'GA',
            'USA',
            '30301',
            true
        ) RETURNING id INTO @default_warehouse_id;

        -- Insert extended information for the default warehouse
        INSERT INTO warehouses_extended (warehouse_id, type, capacity, square_footage, dock_count, temperature_controlled, security_level)
        VALUES (
            @default_warehouse_id,
            'DISTRIBUTION',
            10000,
            50000,
            20,
            true,
            5
        );
    END IF;
END $$;