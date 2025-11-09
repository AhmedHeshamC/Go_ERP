-- Migration: Create email_verifications table
-- Created: ERPGo System
-- Description: Creates the email_verifications table for email verification tokens

-- Create email_verifications table
CREATE TABLE IF NOT EXISTS email_verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    token_type VARCHAR(50) NOT NULL, -- 'verification', 'password_reset', 'email_change'
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_used BOOLEAN DEFAULT FALSE,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for email_verifications table
CREATE INDEX IF NOT EXISTS idx_email_verifications_user_id ON email_verifications(user_id);
CREATE INDEX IF NOT EXISTS idx_email_verifications_email ON email_verifications(email);
CREATE INDEX IF NOT EXISTS idx_email_verifications_token ON email_verifications(token);
CREATE INDEX IF NOT EXISTS idx_email_verifications_token_type ON email_verifications(token_type);
CREATE INDEX IF NOT EXISTS idx_email_verifications_expires_at ON email_verifications(expires_at);
CREATE INDEX IF NOT EXISTS idx_email_verifications_is_used ON email_verifications(is_used);

-- Create composite index for active tokens lookup
CREATE INDEX IF NOT EXISTS idx_email_verifications_active_tokens
ON email_verifications(user_id, token_type, is_used, expires_at)
WHERE is_used = FALSE AND expires_at > CURRENT_TIMESTAMP;

-- Create trigger to automatically update updated_at timestamp
CREATE TRIGGER update_email_verifications_updated_at
    BEFORE UPDATE ON email_verifications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add constraint to ensure token_type is valid
ALTER TABLE email_verifications
ADD CONSTRAINT check_token_type
CHECK (token_type IN ('verification', 'password_reset', 'email_change'));

-- Add constraint to ensure expires_at is in the future
ALTER TABLE email_verifications
ADD CONSTRAINT check_expires_at_future
CHECK (expires_at > created_at);

-- Add comment for the table
COMMENT ON TABLE email_verifications IS 'Stores email verification tokens for account verification, password resets, and email changes';

-- Add comments for columns
COMMENT ON COLUMN email_verifications.id IS 'Primary key identifier';
COMMENT ON COLUMN email_verifications.user_id IS 'Foreign key to users table';
COMMENT ON COLUMN email_verifications.email IS 'Email address where verification was sent';
COMMENT ON COLUMN email_verifications.token IS 'Unique verification token';
COMMENT ON COLUMN email_verifications.token_type IS 'Type of verification token';
COMMENT ON COLUMN email_verifications.expires_at IS 'Token expiration timestamp';
COMMENT ON COLUMN email_verifications.is_used IS 'Whether the token has been used';
COMMENT ON COLUMN email_verifications.used_at IS 'Timestamp when token was used';
COMMENT ON COLUMN email_verifications.created_at IS 'Record creation timestamp';
COMMENT ON COLUMN email_verifications.updated_at IS 'Record last update timestamp';