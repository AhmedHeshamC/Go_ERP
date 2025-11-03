-- Migration: Rollback users table
-- Description: Drops the users table and related objects

-- Drop trigger
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_verified;
DROP INDEX IF EXISTS idx_users_active;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_email;

-- Drop table
DROP TABLE IF EXISTS users;