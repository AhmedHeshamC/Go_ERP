package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"erpgo/pkg/monitoring"
)

// HealthCheckLog represents a health check log entry
type HealthCheckLog struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	ServiceName   string                 `json:"service_name" db:"service_name"`
	Status        monitoring.HealthStatus `json:"status" db:"status"`
	ResponseTime  time.Duration          `json:"response_time" db:"response_time_ms"`
	Message       *string                `json:"message,omitempty" db:"message"`
	ErrorDetails  *string                `json:"error_details,omitempty" db:"error_details"`
	CheckDetails  map[string]interface{} `json:"check_details,omitempty" db:"check_details"`
	Timestamp     time.Time              `json:"timestamp" db:"timestamp"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// HealthCheckMetric represents aggregated health check metrics
type HealthCheckMetric struct {
	ID               uuid.UUID `json:"id" db:"id"`
	ServiceName      string    `json:"service_name" db:"service_name"`
	Date             time.Time `json:"date" db:"date"`
	TotalChecks      int       `json:"total_checks" db:"total_checks"`
	SuccessfulChecks int       `json:"successful_checks" db:"successful_checks"`
	FailedChecks     int       `json:"failed_checks" db:"failed_checks"`
	DegradedChecks   int       `json:"degraded_checks" db:"degraded_checks"`
	AvgResponseTime  float64   `json:"avg_response_time_ms" db:"avg_response_time_ms"`
	MaxResponseTime  int       `json:"max_response_time_ms" db:"max_response_time_ms"`
	MinResponseTime  int       `json:"min_response_time_ms" db:"min_response_time_ms"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// Repository defines the interface for health check data operations
type Repository interface {
	// Health check logs
	LogHealthCheck(ctx context.Context, log *HealthCheckLog) error
	GetRecentHealthChecks(ctx context.Context, serviceName string, limit int) ([]*HealthCheckLog, error)
	GetHealthChecksByTimeRange(ctx context.Context, serviceName string, startTime, endTime time.Time) ([]*HealthCheckLog, error)

	// Health check metrics
	GetDailyMetrics(ctx context.Context, serviceName string, days int) ([]*HealthCheckMetric, error)
	GetMetricsByDateRange(ctx context.Context, serviceName string, startDate, endDate time.Time) ([]*HealthCheckMetric, error)
	GetCurrentStatus(ctx context.Context) (map[string]*HealthCheckLog, error)

	// Cleanup
	OldHealthCheckLogs(ctx context.Context, olderThan time.Time) (int64, error)
}

// PostgreSQL implementation of health check repository
type postgresRepository struct {
	db *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL health check repository
func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

// LogHealthCheck logs a health check result to the database
func (r *postgresRepository) LogHealthCheck(ctx context.Context, log *HealthCheckLog) error {
	query := `
		INSERT INTO health_check_logs (
			id, service_name, status, response_time_ms, message,
			error_details, check_details, timestamp, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	var checkDetailsJSON []byte
	var err error
	if log.CheckDetails != nil {
		checkDetailsJSON, err = json.Marshal(log.CheckDetails)
		if err != nil {
			return fmt.Errorf("failed to marshal check details: %w", err)
		}
	}

	_, err = r.db.Exec(ctx, query,
		log.ID,
		log.ServiceName,
		string(log.Status),
		log.ResponseTime.Milliseconds(),
		log.Message,
		log.ErrorDetails,
		checkDetailsJSON,
		log.Timestamp,
		log.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to log health check: %w", err)
	}

	return nil
}

// GetRecentHealthChecks retrieves recent health check logs for a service
func (r *postgresRepository) GetRecentHealthChecks(ctx context.Context, serviceName string, limit int) ([]*HealthCheckLog, error) {
	query := `
		SELECT id, service_name, status, response_time_ms, message,
			   error_details, check_details, timestamp, created_at
		FROM health_check_logs
		WHERE service_name = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, serviceName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent health checks: %w", err)
	}
	defer rows.Close()

	var logs []*HealthCheckLog
	for rows.Next() {
		log := &HealthCheckLog{}
		var responseTimeMs int64
		var checkDetailsJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.ServiceName,
			&log.Status,
			&responseTimeMs,
			&log.Message,
			&log.ErrorDetails,
			&checkDetailsJSON,
			&log.Timestamp,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan health check log: %w", err)
		}

		log.ResponseTime = time.Duration(responseTimeMs) * time.Millisecond

		if len(checkDetailsJSON) > 0 {
			if err := json.Unmarshal(checkDetailsJSON, &log.CheckDetails); err != nil {
				return nil, fmt.Errorf("failed to unmarshal check details: %w", err)
			}
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating health check logs: %w", err)
	}

	return logs, nil
}

// GetHealthChecksByTimeRange retrieves health check logs within a time range
func (r *postgresRepository) GetHealthChecksByTimeRange(ctx context.Context, serviceName string, startTime, endTime time.Time) ([]*HealthCheckLog, error) {
	query := `
		SELECT id, service_name, status, response_time_ms, message,
			   error_details, check_details, timestamp, created_at
		FROM health_check_logs
		WHERE service_name = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp DESC
	`

	rows, err := r.db.Query(ctx, query, serviceName, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query health checks by time range: %w", err)
	}
	defer rows.Close()

	var logs []*HealthCheckLog
	for rows.Next() {
		log := &HealthCheckLog{}
		var responseTimeMs int64
		var checkDetailsJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.ServiceName,
			&log.Status,
			&responseTimeMs,
			&log.Message,
			&log.ErrorDetails,
			&checkDetailsJSON,
			&log.Timestamp,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan health check log: %w", err)
		}

		log.ResponseTime = time.Duration(responseTimeMs) * time.Millisecond

		if len(checkDetailsJSON) > 0 {
			if err := json.Unmarshal(checkDetailsJSON, &log.CheckDetails); err != nil {
				return nil, fmt.Errorf("failed to unmarshal check details: %w", err)
			}
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating health check logs: %w", err)
	}

	return logs, nil
}

// GetDailyMetrics retrieves daily health check metrics for a service
func (r *postgresRepository) GetDailyMetrics(ctx context.Context, serviceName string, days int) ([]*HealthCheckMetric, error) {
	query := `
		SELECT id, service_name, date, total_checks, successful_checks,
			   failed_checks, degraded_checks, avg_response_time_ms,
			   max_response_time_ms, min_response_time_ms, created_at, updated_at
		FROM health_check_metrics
		WHERE service_name = $1
		ORDER BY date DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, serviceName, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*HealthCheckMetric
	for rows.Next() {
		metric := &HealthCheckMetric{}
		err := rows.Scan(
			&metric.ID,
			&metric.ServiceName,
			&metric.Date,
			&metric.TotalChecks,
			&metric.SuccessfulChecks,
			&metric.FailedChecks,
			&metric.DegradedChecks,
			&metric.AvgResponseTime,
			&metric.MaxResponseTime,
			&metric.MinResponseTime,
			&metric.CreatedAt,
			&metric.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan health check metric: %w", err)
		}

		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating health check metrics: %w", err)
	}

	return metrics, nil
}

// GetMetricsByDateRange retrieves health check metrics within a date range
func (r *postgresRepository) GetMetricsByDateRange(ctx context.Context, serviceName string, startDate, endDate time.Time) ([]*HealthCheckMetric, error) {
	query := `
		SELECT id, service_name, date, total_checks, successful_checks,
			   failed_checks, degraded_checks, avg_response_time_ms,
			   max_response_time_ms, min_response_time_ms, created_at, updated_at
		FROM health_check_metrics
		WHERE service_name = $1 AND date BETWEEN $2 AND $3
		ORDER BY date DESC
	`

	rows, err := r.db.Query(ctx, query, serviceName, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics by date range: %w", err)
	}
	defer rows.Close()

	var metrics []*HealthCheckMetric
	for rows.Next() {
		metric := &HealthCheckMetric{}
		err := rows.Scan(
			&metric.ID,
			&metric.ServiceName,
			&metric.Date,
			&metric.TotalChecks,
			&metric.SuccessfulChecks,
			&metric.FailedChecks,
			&metric.DegradedChecks,
			&metric.AvgResponseTime,
			&metric.MaxResponseTime,
			&metric.MinResponseTime,
			&metric.CreatedAt,
			&metric.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan health check metric: %w", err)
		}

		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating health check metrics: %w", err)
	}

	return metrics, nil
}

// GetCurrentStatus retrieves the current health status of all services
func (r *postgresRepository) GetCurrentStatus(ctx context.Context) (map[string]*HealthCheckLog, error) {
	query := `
		SELECT id, service_name, status, response_time_ms, message,
			   error_details, check_details, timestamp, created_at
		FROM recent_health_status
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query current health status: %w", err)
	}
	defer rows.Close()

	status := make(map[string]*HealthCheckLog)
	for rows.Next() {
		log := &HealthCheckLog{}
		var responseTimeMs int64
		var checkDetailsJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.ServiceName,
			&log.Status,
			&responseTimeMs,
			&log.Message,
			&log.ErrorDetails,
			&checkDetailsJSON,
			&log.Timestamp,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan current health status: %w", err)
		}

		log.ResponseTime = time.Duration(responseTimeMs) * time.Millisecond

		if len(checkDetailsJSON) > 0 {
			if err := json.Unmarshal(checkDetailsJSON, &log.CheckDetails); err != nil {
				return nil, fmt.Errorf("failed to unmarshal check details: %w", err)
			}
		}

		status[log.ServiceName] = log
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating current health status: %w", err)
	}

	return status, nil
}

// OldHealthCheckLogs removes old health check logs and returns the count of deleted records
func (r *postgresRepository) OldHealthCheckLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM health_check_logs WHERE timestamp < $1`

	result, err := r.db.Exec(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old health check logs: %w", err)
	}

	return result.RowsAffected(), nil
}