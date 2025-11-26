-- Create audit_logs table with immutability rules
-- This table stores all security-relevant events for compliance and auditing

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type VARCHAR(100) NOT NULL,
    user_id UUID REFERENCES users(id),
    resource_id VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient querying
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_user_id_timestamp 
    ON audit_logs(user_id, timestamp DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_event_type_timestamp 
    ON audit_logs(event_type, timestamp DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_resource_id 
    ON audit_logs(resource_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_timestamp 
    ON audit_logs(timestamp DESC);

-- Make the table append-only (no updates/deletes allowed)
-- This ensures audit log immutability for compliance
CREATE RULE audit_logs_no_update AS 
    ON UPDATE TO audit_logs 
    DO INSTEAD NOTHING;

CREATE RULE audit_logs_no_delete AS 
    ON DELETE TO audit_logs 
    DO INSTEAD NOTHING;

-- Add comment to document the table purpose
COMMENT ON TABLE audit_logs IS 'Immutable audit log for security-relevant events. No updates or deletes allowed.';
