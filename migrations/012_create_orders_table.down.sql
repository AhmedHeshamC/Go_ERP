-- Drop orders table and related objects
DROP TRIGGER IF EXISTS trigger_orders_updated_at ON orders;
DROP TABLE IF EXISTS orders;