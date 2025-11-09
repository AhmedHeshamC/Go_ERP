-- Drop order_items table and related objects
DROP TRIGGER IF EXISTS trigger_update_order_totals_on_insert ON order_items;
DROP TRIGGER IF EXISTS trigger_update_order_totals_on_update ON order_items;
DROP TRIGGER IF EXISTS trigger_update_order_totals_on_delete ON order_items;
DROP TRIGGER IF EXISTS trigger_order_items_updated_at ON order_items;
DROP TABLE IF EXISTS order_items;

-- Drop the function
DROP FUNCTION IF EXISTS update_order_totals_on_item_change();