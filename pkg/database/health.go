package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
)

// HealthStatus represents the health status of the database
type HealthStatus struct {
	Status     string                 `json:"status"`
	Message    string                 `json:"message,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Checks     map[string]CheckResult `json:"checks,omitempty"`
}

// CheckResult represents the result of a health check
type CheckResult struct {
	Status  string        `json:"status"`
	Message string        `json:"message,omitempty"`
	Latency time.Duration `json:"latency"`
}

// HealthChecker performs health checks on the database
type HealthChecker struct {
	db     *Database
	logger *zerolog.Logger
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *Database, logger *zerolog.Logger) *HealthChecker {
	return &HealthChecker{
		db:     db,
		logger: logger,
	}
}

// Check performs a comprehensive health check
func (hc *HealthChecker) Check(ctx context.Context) HealthStatus {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	status := HealthStatus{
		Timestamp: time.Now().UTC(),
		Checks:    make(map[string]CheckResult),
		Details:   make(map[string]interface{}),
	}

	// Perform basic connection check
	connCheck := hc.checkConnection(ctx)
	status.Checks["connection"] = connCheck

	// Perform query check
	queryCheck := hc.checkQuery(ctx)
	status.Checks["query"] = queryCheck

	// Get pool statistics
	stats := hc.db.Stats()
	status.Details["pool_stats"] = map[string]interface{}{
		"max_conns":         stats.MaxConns(),
		"total_conns":       stats.TotalConns(),
		"idle_conns":        stats.IdleConns(),
		"acquired_conns":    stats.AcquiredConns(),
		"constructing_conns": stats.ConstructingConns(),
	}

	// Determine overall status
	allHealthy := true
	for _, check := range status.Checks {
		if check.Status != "healthy" {
			allHealthy = false
			break
		}
	}

	if allHealthy {
		status.Status = "healthy"
		status.Message = "All database health checks passed"
	} else {
		status.Status = "unhealthy"
		status.Message = "One or more database health checks failed"
	}

	hc.logger.Info().
		Str("status", status.Status).
		Str("message", status.Message).
		Msg("Database health check completed")

	return status
}

// checkConnection performs a basic connection check
func (hc *HealthChecker) checkConnection(ctx context.Context) CheckResult {
	start := time.Now()
	err := hc.db.Ping(ctx)
	latency := time.Since(start)

	if err != nil {
		hc.logger.Error().Err(err).Dur("latency", latency).Msg("Database connection health check failed")
		return CheckResult{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Connection failed: %v", err),
			Latency: latency,
		}
	}

	return CheckResult{
		Status:  "healthy",
		Message: "Connection successful",
		Latency: latency,
	}
}

// checkQuery performs a simple query test
func (hc *HealthChecker) checkQuery(ctx context.Context) CheckResult {
	start := time.Now()

	var version string
	err := hc.db.QueryRow(ctx, "SELECT version()").Scan(&version)
	latency := time.Since(start)

	if err != nil {
		hc.logger.Error().Err(err).Dur("latency", latency).Msg("Database query health check failed")
		return CheckResult{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Query failed: %v", err),
			Latency: latency,
		}
	}

	return CheckResult{
		Status:  "healthy",
		Message: "Query successful",
		Latency: latency,
	}
}

// Metrics holds database metrics
type Metrics struct {
	// Connection metrics
	ConnectionsActive   prometheus.Gauge
	ConnectionsIdle     prometheus.Gauge
	ConnectionsMax      prometheus.Gauge
	ConnectionsTotal    prometheus.Counter

	// Query metrics
	QueryDuration       prometheus.Histogram
	QueryErrors         prometheus.Counter
	QueryTotal          prometheus.Counter

	// Transaction metrics
	TransactionDuration prometheus.Histogram
	TransactionErrors   prometheus.Counter
	TransactionTotal    prometheus.Counter

	// Pool metrics
	PoolAcquireDuration prometheus.Histogram
	PoolAcquireErrors   prometheus.Counter
}

// NewMetrics creates and registers database metrics
func NewMetrics(namespace string) *Metrics {
	return &Metrics{
		ConnectionsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "connections_active",
			Help:      "Number of active database connections",
		}),
		ConnectionsIdle: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "connections_idle",
			Help:      "Number of idle database connections",
		}),
		ConnectionsMax: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "connections_max",
			Help:      "Maximum number of database connections",
		}),
		ConnectionsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "connections_total",
			Help:      "Total number of database connections created",
		}),
		QueryDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "query_duration_seconds",
			Help:      "Duration of database queries",
			Buckets:   prometheus.DefBuckets,
		}),
		QueryErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "query_errors_total",
			Help:      "Total number of database query errors",
		}),
		QueryTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "queries_total",
			Help:      "Total number of database queries executed",
		}),
		TransactionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "transaction_duration_seconds",
			Help:      "Duration of database transactions",
			Buckets:   prometheus.DefBuckets,
		}),
		TransactionErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "transaction_errors_total",
			Help:      "Total number of database transaction errors",
		}),
		TransactionTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "transactions_total",
			Help:      "Total number of database transactions",
		}),
		PoolAcquireDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "pool_acquire_duration_seconds",
			Help:      "Duration of acquiring connections from pool",
			Buckets:   prometheus.DefBuckets,
		}),
		PoolAcquireErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "pool_acquire_errors_total",
			Help:      "Total number of connection pool acquire errors",
		}),
	}
}

// UpdateMetrics updates the metrics with current database statistics
func (m *Metrics) UpdateMetrics(db *Database) {
	stats := db.Stats()

	m.ConnectionsActive.Set(float64(stats.TotalConns() - stats.IdleConns()))
	m.ConnectionsIdle.Set(float64(stats.IdleConns()))
	m.ConnectionsMax.Set(float64(stats.MaxConns()))
}

// MonitoredDatabase wraps a Database with metrics collection
type MonitoredDatabase struct {
	*Database
	metrics *Metrics
	logger  *zerolog.Logger
}

// NewMonitoredDatabase creates a new monitored database
func NewMonitoredDatabase(cfg Config, metrics *Metrics, logger *zerolog.Logger) (*MonitoredDatabase, error) {
	db, err := NewWithLogger(cfg, logger)
	if err != nil {
		return nil, err
	}

	return &MonitoredDatabase{
		Database: db,
		metrics:  metrics,
		logger:   logger,
	}, nil
}

// Exec executes a query with metrics collection
func (md *MonitoredDatabase) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	start := time.Now()
	md.metrics.QueryTotal.Inc()

	result, err := md.Database.Exec(ctx, query, args...)

	duration := time.Since(start)
	md.metrics.QueryDuration.Observe(duration.Seconds())

	if err != nil {
		md.metrics.QueryErrors.Inc()
		md.logger.Error().
			Err(err).
			Str("query", query).
			Dur("duration", duration).
			Msg("Database query failed")
	} else {
		md.logger.Debug().
			Str("query", query).
			Dur("duration", duration).
			Msg("Database query executed successfully")
	}

	return result, err
}

// Query executes a query that returns rows with metrics collection
func (md *MonitoredDatabase) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	start := time.Now()
	md.metrics.QueryTotal.Inc()

	result, err := md.Database.Query(ctx, query, args...)

	duration := time.Since(start)
	md.metrics.QueryDuration.Observe(duration.Seconds())

	if err != nil {
		md.metrics.QueryErrors.Inc()
		md.logger.Error().
			Err(err).
			Str("query", query).
			Dur("duration", duration).
			Msg("Database query failed")
	}

	return result, err
}

// QueryRow executes a query that returns a single row with metrics collection
func (md *MonitoredDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	start := time.Now()
	md.metrics.QueryTotal.Inc()

	result := md.Database.QueryRow(ctx, query, args...)

	// Note: We can't measure the actual query duration here since the row
	// hasn't been scanned yet. This is a limitation of the pgx API.
	duration := time.Since(start)
	md.logger.Debug().
		Str("query", query).
		Dur("duration", duration).
		Msg("Database query started")

	return result
}

// Begin begins a transaction with metrics collection
func (md *MonitoredDatabase) Begin(ctx context.Context) (pgx.Tx, error) {
	start := time.Now()
	md.metrics.TransactionTotal.Inc()

	tx, err := md.Database.Begin(ctx)

	duration := time.Since(start)
	md.metrics.TransactionDuration.Observe(duration.Seconds())

	if err != nil {
		md.metrics.TransactionErrors.Inc()
		md.logger.Error().
			Err(err).
			Dur("duration", duration).
			Msg("Database transaction begin failed")
	}

	return &MonitoredTx{Tx: tx, metrics: md.metrics, logger: md.logger}, err
}

// BeginTx begins a transaction with the given options and metrics collection
func (md *MonitoredDatabase) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	start := time.Now()
	md.metrics.TransactionTotal.Inc()

	tx, err := md.Database.BeginTx(ctx, txOptions)

	duration := time.Since(start)
	md.metrics.TransactionDuration.Observe(duration.Seconds())

	if err != nil {
		md.metrics.TransactionErrors.Inc()
		md.logger.Error().
			Err(err).
			Dur("duration", duration).
			Msg("Database transaction begin failed")
	}

	return &MonitoredTx{Tx: tx, metrics: md.metrics, logger: md.logger}, err
}

// Acquire acquires a connection from the pool with metrics collection
func (md *MonitoredDatabase) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	start := time.Now()

	conn, err := md.Database.Acquire(ctx)

	duration := time.Since(start)
	md.metrics.PoolAcquireDuration.Observe(duration.Seconds())

	if err != nil {
		md.metrics.PoolAcquireErrors.Inc()
		md.logger.Error().
			Err(err).
			Dur("duration", duration).
			Msg("Connection pool acquire failed")
	}

	return conn, err
}

// MonitoredTx wraps a pgx.Tx with metrics collection
type MonitoredTx struct {
	pgx.Tx
	metrics *Metrics
	logger  *zerolog.Logger
}

// Commit commits the transaction with metrics collection
func (mt *MonitoredTx) Commit(ctx context.Context) error {
	start := time.Now()

	err := mt.Tx.Commit(ctx)

	duration := time.Since(start)
	mt.metrics.TransactionDuration.Observe(duration.Seconds())

	if err != nil {
		mt.metrics.TransactionErrors.Inc()
		mt.logger.Error().
			Err(err).
			Dur("duration", duration).
			Msg("Database transaction commit failed")
	} else {
		mt.logger.Debug().
			Dur("duration", duration).
			Msg("Database transaction committed successfully")
	}

	return err
}

// Rollback rolls back the transaction with metrics collection
func (mt *MonitoredTx) Rollback(ctx context.Context) error {
	start := time.Now()

	err := mt.Tx.Rollback(ctx)

	duration := time.Since(start)
	mt.metrics.TransactionDuration.Observe(duration.Seconds())

	if err != nil {
		mt.metrics.TransactionErrors.Inc()
		mt.logger.Error().
			Err(err).
			Dur("duration", duration).
			Msg("Database transaction rollback failed")
	} else {
		mt.logger.Debug().
			Dur("duration", duration).
			Msg("Database transaction rolled back successfully")
	}

	return err
}