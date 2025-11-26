package database

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
)

// SlowQueryMonitor monitors and logs slow database queries
type SlowQueryMonitor struct {
	logger    *zerolog.Logger
	threshold time.Duration
	mu        sync.RWMutex
	queries   []SlowQueryRecord

	// Prometheus metrics
	slowQueryCount    prometheus.Counter
	slowQueryDuration prometheus.Histogram
}

// SlowQueryRecord represents a slow query record
type SlowQueryRecord struct {
	Query     string        `json:"query"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
	QueryType string        `json:"query_type"`
	Args      int           `json:"args"`
}

// NewSlowQueryMonitor creates a new slow query monitor
func NewSlowQueryMonitor(logger *zerolog.Logger, threshold time.Duration) *SlowQueryMonitor {
	if threshold == 0 {
		threshold = 100 * time.Millisecond // Default to 100ms
	}

	monitor := &SlowQueryMonitor{
		logger:    logger,
		threshold: threshold,
		queries:   make([]SlowQueryRecord, 0, 100), // Keep last 100 slow queries
	}

	// Initialize Prometheus metrics
	monitor.slowQueryCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "db_slow_queries_total",
		Help: "Total number of slow database queries (>100ms)",
	})

	monitor.slowQueryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "db_slow_query_duration_seconds",
		Help:    "Duration of slow database queries",
		Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
	})

	return monitor
}

// RecordQuery records a query execution
func (m *SlowQueryMonitor) RecordQuery(ctx context.Context, query string, queryType string, args int, duration time.Duration) {
	if duration > m.threshold {
		m.slowQueryCount.Inc()
		m.slowQueryDuration.Observe(duration.Seconds())

		record := SlowQueryRecord{
			Query:     query,
			Duration:  duration,
			Timestamp: time.Now(),
			QueryType: queryType,
			Args:      args,
		}

		m.mu.Lock()
		// Keep only last 100 slow queries
		if len(m.queries) >= 100 {
			m.queries = m.queries[1:]
		}
		m.queries = append(m.queries, record)
		m.mu.Unlock()

		m.logger.Warn().
			Str("query", query).
			Str("query_type", queryType).
			Dur("duration", duration).
			Dur("threshold", m.threshold).
			Int("args", args).
			Msg("Slow database query detected")
	}
}

// GetSlowQueries returns recent slow queries
func (m *SlowQueryMonitor) GetSlowQueries() []SlowQueryRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]SlowQueryRecord, len(m.queries))
	copy(result, m.queries)
	return result
}

// ClearSlowQueries clears the slow query history
func (m *SlowQueryMonitor) ClearSlowQueries() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queries = make([]SlowQueryRecord, 0, 100)
}

// GetThreshold returns the current slow query threshold
func (m *SlowQueryMonitor) GetThreshold() time.Duration {
	return m.threshold
}

// SetThreshold updates the slow query threshold
func (m *SlowQueryMonitor) SetThreshold(threshold time.Duration) {
	m.threshold = threshold
	m.logger.Info().
		Dur("threshold", threshold).
		Msg("Slow query threshold updated")
}
