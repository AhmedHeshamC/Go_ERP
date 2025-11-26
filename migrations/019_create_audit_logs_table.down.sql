-- Drop audit_logs table and related rules

-- Drop the immutability rules first
DROP RULE IF EXISTS audit_logs_no_delete ON audit_logs;
DROP RULE IF EXISTS audit_logs_no_update ON audit_logs;

-- Drop indexes
DROP INDEX IF EXISTS idx_audit_timestamp;
DROP INDEX IF EXISTS idx_audit_resource_id;
DROP INDEX IF EXISTS idx_audit_event_type_timestamp;
DROP INDEX IF EXISTS idx_audit_user_id_timestamp;

-- Drop the table
DROP TABLE IF EXISTS audit_logs;
