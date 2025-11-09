-- Migration: Drop roles table
-- Created: ERPGo System
-- Description: Drops the roles table

-- Drop trigger
DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;

-- Drop indexes
DROP INDEX IF EXISTS idx_roles_name;
DROP INDEX IF EXISTS idx_roles_created_at;

-- Drop table
DROP TABLE IF EXISTS roles;