package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
)

// PoolMonitor monitors database connection pool metrics
type PoolMonitor struct {
	pool   *pgxpool.Pool
	logger *zerolog.Logger
	config *PoolMonitorConfig

	// Prometheus metrics
	acquireCount            prometheus.Counter
	acquireDuration         prometheus.Histogram
	acquiredConns           prometheus.Gauge
	canceledAcquireCount    prometheus.Counter
	constructingConns       prometheus.Gauge
	emptyAcquireCount       prometheus.Counter
	idleConns               prometheus.Gauge
	maxConns                prometheus.Gauge
	totalConns              prometheus.Gauge
	newConnsCount           prometheus.Counter
	maxLifetimeDestroyCount prometheus.Counter
	maxIdleDestroyCount     prometheus.Counter
	poolUtilization         prometheus.Gauge
}

// PoolMonitorConfig holds configuration for pool monitoring
type PoolMonitorConfig struct {
	// Interval for collecting pool statistics
	CollectionInterval time.Duration

	// Threshold for logging warnings (percentage of max connections)
	WarningThreshold float64

	// Threshold for logging critical alerts (percentage of max connections)
	CriticalThreshold float64

	// Enable detailed logging
	EnableDetailedLogging bool
}

// DefaultPoolMonitorConfig returns default configuration for pool monitoring
func DefaultPoolMonitorConfig() *PoolMonitorConfig {
	return &PoolMonitorConfig{
		CollectionInterval:    30 * time.Second,
		WarningThreshold:      0.80, // 80%
		CriticalThreshold:     0.95, // 95%
		EnableDetailedLogging: false,
	}
}

// NewPoolMonitor creates a new connection pool monitor
func NewPoolMonitor(pool *pgxpool.Pool, logger *zerolog.Logger, config *PoolMonitorConfig) *PoolMonitor {
	if config == nil {
		config = DefaultPoolMonitorConfig()
	}

	monitor := &PoolMonitor{
		pool:   pool,
		logger: logger,
		config: config,
	}

	// Initialize Prometheus metrics
	monitor.initMetrics()

	return monitor
}

// initMetrics initializes Prometheus metrics for connection pool monitoring
func (m *PoolMonitor) initMetrics() {
	m.acquireCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "db_pool_acquire_count_total",
		Help: "Total number of successful connection acquisitions from the pool",
	})

	m.acquireDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "db_pool_acquire_duration_seconds",
		Help:    "Duration of connection acquisition from the pool",
		Buckets: prometheus.DefBuckets,
	})

	m.acquiredConns = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_acquired_connections",
		Help: "Number of currently acquired connections in the pool",
	})

	m.canceledAcquireCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "db_pool_canceled_acquire_count_total",
		Help: "Total number of canceled connection acquisitions",
	})

	m.constructingConns = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_constructing_connections",
		Help: "Number of connections currently being constructed",
	})

	m.emptyAcquireCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "db_pool_empty_acquire_count_total",
		Help: "Total number of times a connection was acquired without waiting",
	})

	m.idleConns = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_idle_connections",
		Help: "Number of currently idle connections in the pool",
	})

	m.maxConns = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_max_connections",
		Help: "Maximum number of connections allowed in the pool",
	})

	m.totalConns = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_total_connections",
		Help: "Total number of connections in the pool (idle + acquired + constructing)",
	})

	m.newConnsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "db_pool_new_connections_total",
		Help: "Total number of new connections created",
	})

	m.maxLifetimeDestroyCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "db_pool_max_lifetime_destroy_count_total",
		Help: "Total number of connections destroyed due to max lifetime",
	})

	m.maxIdleDestroyCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "db_pool_max_idle_destroy_count_total",
		Help: "Total number of connections destroyed due to max idle time",
	})

	m.poolUtilization = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_utilization_percent",
		Help: "Connection pool utilization as a percentage of max connections",
	})
}

// Start begins monitoring the connection pool
func (m *PoolMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.config.CollectionInterval)
	defer ticker.Stop()

	m.logger.Info().
		Dur("interval", m.config.CollectionInterval).
		Float64("warning_threshold", m.config.WarningThreshold).
		Float64("critical_threshold", m.config.CriticalThreshold).
		Msg("Starting connection pool monitoring")

	for {
		select {
		case <-ctx.Done():
			m.logger.Info().Msg("Stopping connection pool monitoring")
			return
		case <-ticker.C:
			m.collectMetrics()
		}
	}
}

// collectMetrics collects and updates pool statistics
func (m *PoolMonitor) collectMetrics() {
	stats := m.pool.Stat()

	// Update Prometheus metrics
	m.acquireCount.Add(float64(stats.AcquireCount()))
	m.acquiredConns.Set(float64(stats.AcquiredConns()))
	m.canceledAcquireCount.Add(float64(stats.CanceledAcquireCount()))
	m.constructingConns.Set(float64(stats.ConstructingConns()))
	m.emptyAcquireCount.Add(float64(stats.EmptyAcquireCount()))
	m.idleConns.Set(float64(stats.IdleConns()))
	m.maxConns.Set(float64(stats.MaxConns()))
	m.totalConns.Set(float64(stats.TotalConns()))
	m.newConnsCount.Add(float64(stats.NewConnsCount()))
	m.maxLifetimeDestroyCount.Add(float64(stats.MaxLifetimeDestroyCount()))
	m.maxIdleDestroyCount.Add(float64(stats.MaxIdleDestroyCount()))

	// Calculate pool utilization
	maxConns := float64(stats.MaxConns())
	totalConns := float64(stats.TotalConns())
	utilization := 0.0
	if maxConns > 0 {
		utilization = (totalConns / maxConns) * 100
	}
	m.poolUtilization.Set(utilization)

	// Log warnings based on thresholds
	utilizationRatio := totalConns / maxConns
	if utilizationRatio >= m.config.CriticalThreshold {
		m.logger.Error().
			Int32("total_conns", stats.TotalConns()).
			Int32("max_conns", stats.MaxConns()).
			Int32("acquired_conns", stats.AcquiredConns()).
			Int32("idle_conns", stats.IdleConns()).
			Int32("constructing_conns", stats.ConstructingConns()).
			Float64("utilization_percent", utilization).
			Msg("CRITICAL: Connection pool near exhaustion")
	} else if utilizationRatio >= m.config.WarningThreshold {
		m.logger.Warn().
			Int32("total_conns", stats.TotalConns()).
			Int32("max_conns", stats.MaxConns()).
			Int32("acquired_conns", stats.AcquiredConns()).
			Int32("idle_conns", stats.IdleConns()).
			Float64("utilization_percent", utilization).
			Msg("WARNING: Connection pool utilization high")
	}

	// Detailed logging if enabled
	if m.config.EnableDetailedLogging {
		m.logger.Debug().
			Int64("acquire_count", stats.AcquireCount()).
			Int32("acquired_conns", stats.AcquiredConns()).
			Int64("canceled_acquire_count", stats.CanceledAcquireCount()).
			Int32("constructing_conns", stats.ConstructingConns()).
			Int64("empty_acquire_count", stats.EmptyAcquireCount()).
			Int32("idle_conns", stats.IdleConns()).
			Int32("max_conns", stats.MaxConns()).
			Int32("total_conns", stats.TotalConns()).
			Int64("new_conns_count", stats.NewConnsCount()).
			Int64("max_lifetime_destroy_count", stats.MaxLifetimeDestroyCount()).
			Int64("max_idle_destroy_count", stats.MaxIdleDestroyCount()).
			Float64("utilization_percent", utilization).
			Msg("Connection pool statistics")
	}
}

// GetCurrentStats returns current pool statistics
func (m *PoolMonitor) GetCurrentStats() *PoolStats {
	stats := m.pool.Stat()
	maxConns := float64(stats.MaxConns())
	totalConns := float64(stats.TotalConns())
	utilization := 0.0
	if maxConns > 0 {
		utilization = (totalConns / maxConns) * 100
	}

	return &PoolStats{
		AcquireCount:            stats.AcquireCount(),
		AcquiredConns:           stats.AcquiredConns(),
		CanceledAcquireCount:    stats.CanceledAcquireCount(),
		ConstructingConns:       stats.ConstructingConns(),
		EmptyAcquireCount:       stats.EmptyAcquireCount(),
		IdleConns:               stats.IdleConns(),
		MaxConns:                stats.MaxConns(),
		TotalConns:              stats.TotalConns(),
		NewConnsCount:           stats.NewConnsCount(),
		MaxLifetimeDestroyCount: stats.MaxLifetimeDestroyCount(),
		MaxIdleDestroyCount:     stats.MaxIdleDestroyCount(),
		UtilizationPercent:      utilization,
	}
}

// PoolStats represents connection pool statistics
type PoolStats struct {
	AcquireCount            int64   `json:"acquire_count"`
	AcquiredConns           int32   `json:"acquired_conns"`
	CanceledAcquireCount    int64   `json:"canceled_acquire_count"`
	ConstructingConns       int32   `json:"constructing_conns"`
	EmptyAcquireCount       int64   `json:"empty_acquire_count"`
	IdleConns               int32   `json:"idle_conns"`
	MaxConns                int32   `json:"max_conns"`
	TotalConns              int32   `json:"total_conns"`
	NewConnsCount           int64   `json:"new_conns_count"`
	MaxLifetimeDestroyCount int64   `json:"max_lifetime_destroy_count"`
	MaxIdleDestroyCount     int64   `json:"max_idle_destroy_count"`
	UtilizationPercent      float64 `json:"utilization_percent"`
}

// IsHealthy returns true if the pool is healthy based on utilization
func (s *PoolStats) IsHealthy(warningThreshold, criticalThreshold float64) (bool, string) {
	utilizationRatio := float64(s.TotalConns) / float64(s.MaxConns)

	if utilizationRatio >= criticalThreshold {
		return false, "critical: pool near exhaustion"
	}

	if utilizationRatio >= warningThreshold {
		return true, "warning: high utilization"
	}

	return true, "healthy"
}
