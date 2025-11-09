-- Migration: Drop email_verifications table
-- Created: ERPGo System
-- Description: Drops the email_verifications table and related objects

-- Drop indexes
DROP INDEX IF EXISTS idx_email_verifications_user_id;
DROP INDEX IF EXISTS idx_email_verifications_email;
DROP INDEX IF EXISTS idx_email_verifications_token;
DROP INDEX IF EXISTS idx_email_verifications_token_type;
DROP INDEX IF EXISTS idx_email_verifications_expires_at;
DROP INDEX IF EXISTS idx_email_verifications_is_used;
DROP INDEX IF EXISTS idx_email_verifications_active_tokens;

-- Drop trigger
DROP TRIGGER IF EXISTS update_email_verifications_updated_at ON email_verifications;

-- Drop constraints
ALTER TABLE email_verifications DROP CONSTRAINT IF EXISTS check_token_type;
ALTER TABLE email_verifications DROP CONSTRAINT IF EXISTS check_expires_at_future;

-- Drop table
DROP TABLE IF EXISTS email_verifications;