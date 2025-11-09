-- Create health check logs table for monitoring database health over time
CREATE TABLE IF NOT EXISTS health_check_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,
    response_time_ms INTEGER NOT NULL,
    message TEXT,
    error_details TEXT,
    check_details JSONB,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_health_check_logs_service_name ON health_check_logs(service_name);
CREATE INDEX IF NOT EXISTS idx_health_check_logs_status ON health_check_logs(status);
CREATE INDEX IF NOT EXISTS idx_health_check_logs_timestamp ON health_check_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_health_check_logs_service_timestamp ON health_check_logs(service_name, timestamp);

-- Create a table for health check metrics aggregation
CREATE TABLE IF NOT EXISTS health_check_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name VARCHAR(100) NOT NULL,
    date DATE NOT NULL,
    total_checks INTEGER NOT NULL DEFAULT 0,
    successful_checks INTEGER NOT NULL DEFAULT 0,
    failed_checks INTEGER NOT NULL DEFAULT 0,
    degraded_checks INTEGER NOT NULL DEFAULT 0,
    avg_response_time_ms DECIMAL(10,2),
    max_response_time_ms INTEGER,
    min_response_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(service_name, date)
);

-- Create indexes for metrics table
CREATE INDEX IF NOT EXISTS idx_health_check_metrics_service_date ON health_check_metrics(service_name, date);

-- Create a function to update metrics
CREATE OR REPLACE FUNCTION update_health_check_metrics()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO health_check_metrics (
        service_name,
        date,
        total_checks,
        successful_checks,
        failed_checks,
        degraded_checks,
        avg_response_time_ms,
        max_response_time_ms,
        min_response_time_ms
    )
    VALUES (
        NEW.service_name,
        DATE(NEW.timestamp),
        1,
        CASE WHEN NEW.status = 'healthy' THEN 1 ELSE 0 END,
        CASE WHEN NEW.status = 'unhealthy' THEN 1 ELSE 0 END,
        CASE WHEN NEW.status = 'degraded' THEN 1 ELSE 0 END,
        NEW.response_time_ms,
        NEW.response_time_ms,
        NEW.response_time_ms
    )
    ON CONFLICT (service_name, date)
    DO UPDATE SET
        total_checks = health_check_metrics.total_checks + 1,
        successful_checks = health_check_metrics.successful_checks +
            CASE WHEN NEW.status = 'healthy' THEN 1 ELSE 0 END,
        failed_checks = health_check_metrics.failed_checks +
            CASE WHEN NEW.status = 'unhealthy' THEN 1 ELSE 0 END,
        degraded_checks = health_check_metrics.degraded_checks +
            CASE WHEN NEW.status = 'degraded' THEN 1 ELSE 0 END,
        avg_response_time_ms = (
            (health_check_metrics.avg_response_time_ms * health_check_metrics.total_checks) + NEW.response_time_ms
        ) / (health_check_metrics.total_checks + 1),
        max_response_time_ms = GREATEST(health_check_metrics.max_response_time_ms, NEW.response_time_ms),
        min_response_time_ms = LEAST(health_check_metrics.min_response_time_ms, NEW.response_time_ms),
        updated_at = NOW();

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update metrics
CREATE TRIGGER trigger_update_health_check_metrics
    AFTER INSERT ON health_check_logs
    FOR EACH ROW
    EXECUTE FUNCTION update_health_check_metrics();

-- Create a view for recent health check status
CREATE OR REPLACE VIEW recent_health_status AS
SELECT DISTINCT ON (service_name)
    service_name,
    status,
    response_time_ms,
    message,
    timestamp
FROM health_check_logs
ORDER BY service_name, timestamp DESC;

-- Create a view for daily health summaries
CREATE OR REPLACE VIEW daily_health_summary AS
SELECT
    service_name,
    date,
    total_checks,
    successful_checks,
    failed_checks,
    degraded_checks,
    ROUND((successful_checks::DECIMAL / NULLIF(total_checks, 0)) * 100, 2) as success_rate,
    avg_response_time_ms,
    max_response_time_ms,
    min_response_time_ms
FROM health_check_metrics
ORDER BY service_name, date DESC;