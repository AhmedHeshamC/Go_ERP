-- Drop order_addresses table and related objects
DROP TRIGGER IF EXISTS trigger_order_addresses_updated_at ON order_addresses;
DROP TABLE IF EXISTS order_addresses;