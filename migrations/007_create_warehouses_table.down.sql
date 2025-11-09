-- Drop warehouse_zones table
DROP TABLE IF EXISTS warehouse_zones;

-- Drop warehouse_contacts table
DROP TABLE IF EXISTS warehouse_contacts;

-- Drop warehouses_extended table
DROP TABLE IF EXISTS warehouses_extended;

-- Drop warehouse_type enum
DROP TYPE IF EXISTS warehouse_type;

-- Drop warehouses table
DROP TABLE IF EXISTS warehouses;

-- Drop the update function if it exists
DROP FUNCTION IF EXISTS update_warehouses_updated_at();