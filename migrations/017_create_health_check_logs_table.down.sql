-- Drop views
DROP VIEW IF EXISTS daily_health_summary;
DROP VIEW IF EXISTS recent_health_status;

-- Drop trigger
DROP TRIGGER IF EXISTS trigger_update_health_check_metrics ON health_check_logs;

-- Drop function
DROP FUNCTION IF EXISTS update_health_check_metrics();

-- Drop tables
DROP TABLE IF EXISTS health_check_metrics;
DROP TABLE IF EXISTS health_check_logs;