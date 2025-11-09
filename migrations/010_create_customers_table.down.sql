-- Drop customers table and related objects
DROP TRIGGER IF EXISTS trigger_customers_updated_at ON customers;
DROP TABLE IF EXISTS customers;

-- Drop companies table and related objects
DROP TRIGGER IF EXISTS trigger_companies_updated_at ON companies;
DROP TABLE IF EXISTS companies;

-- Drop the trigger function (if it's not used elsewhere)
DROP FUNCTION IF EXISTS update_companies_updated_at();