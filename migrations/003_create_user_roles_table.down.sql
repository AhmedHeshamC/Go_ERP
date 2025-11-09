-- Migration: Drop user_roles table
-- Created: ERPGo System
-- Description: Drops the user_roles junction table

-- Drop indexes
DROP INDEX IF EXISTS idx_user_roles_user_id;
DROP INDEX IF EXISTS idx_user_roles_role_id;
DROP INDEX IF EXISTS idx_user_roles_assigned_at;

-- Drop table
DROP TABLE IF EXISTS user_roles;