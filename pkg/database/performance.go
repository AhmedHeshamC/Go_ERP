package database

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
)

// PerformanceMetrics holds database performance metrics
type PerformanceMetrics struct {
	// Prometheus metrics
	queryDuration *prometheus.HistogramVec
	queryTotal    *prometheus.CounterVec
	queryErrors   *prometheus.CounterVec
	activeConns   prometheus.Gauge
	idleConns     prometheus.Gauge
	connAcquires  prometheus.Counter
	slowQueries   prometheus.Counter

	// Query caching
	cacheHits prometheus.Counter
	cacheMiss prometheus.Counter
	cacheSets prometheus.Counter

	logger *zerolog.Logger
}

// QueryCache interface for database query caching
type QueryCache interface {
	Get(ctx context.Context, key string) (interface{}, bool)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration)
	Delete(ctx context.Context, key string)
}

// PerformanceDB wraps Database with performance monitoring capabilities
type PerformanceDB struct {
	*Database
	metrics            *PerformanceMetrics
	queryCache         QueryCache
	slowQueryThreshold time.Duration

	// Query patterns for optimization
	queryPatterns map[string]*PerformanceQueryPattern
	patternMutex  sync.RWMutex
}

// PerformanceQueryPattern tracks query execution patterns for optimization
type PerformanceQueryPattern struct {
	Query      string
	Count      int64
	TotalTime  time.Duration
	MaxTime    time.Duration
	MinTime    time.Duration
	LastUsed   time.Time
	Parameters []string
}

// NewPerformanceDB creates a new PerformanceDB instance with monitoring
func NewPerformanceDB(db *Database, cache QueryCache, logger *zerolog.Logger) *PerformanceDB {
	if logger == nil {
		l := zerolog.Nop()
		logger = &l
	}

	metrics := &PerformanceMetrics{
		queryDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Database query execution duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"query_type", "table", "operation"}),
		queryTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "database_queries_total",
			Help: "Total number of database queries executed",
		}, []string{"query_type", "table", "operation", "status"}),
		queryErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "database_query_errors_total",
			Help: "Total number of database query errors",
		}, []string{"query_type", "table", "operation", "error_type"}),
		activeConns: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "database_active_connections",
			Help: "Number of active database connections",
		}),
		idleConns: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "database_idle_connections",
			Help: "Number of idle database connections",
		}),
		connAcquires: promauto.NewCounter(prometheus.CounterOpts{
			Name: "database_connection_acquires_total",
			Help: "Total number of connection pool acquisitions",
		}),
		slowQueries: promauto.NewCounter(prometheus.CounterOpts{
			Name: "database_slow_queries_total",
			Help: "Total number of slow queries exceeding threshold",
		}),
		cacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "database_cache_hits_total",
			Help: "Total number of query cache hits",
		}),
		cacheMiss: promauto.NewCounter(prometheus.CounterOpts{
			Name: "database_cache_misses_total",
			Help: "Total number of query cache misses",
		}),
		cacheSets: promauto.NewCounter(prometheus.CounterOpts{
			Name: "database_cache_sets_total",
			Help: "Total number of query cache sets",
		}),
		logger: logger,
	}

	return &PerformanceDB{
		Database:           db,
		metrics:            metrics,
		queryCache:         cache,
		slowQueryThreshold: 100 * time.Millisecond,
		queryPatterns:      make(map[string]*PerformanceQueryPattern),
	}
}

// Exec executes a query with performance monitoring
func (pdb *PerformanceDB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	start := time.Now()
	queryType, table, operation := pdb.parseQuery(query)

	// Check cache first for write operations (cache invalidation)
	cacheKey := pdb.generateCacheKey(query, args)
	if pdb.queryCache != nil && pdb.isCacheableQuery(query) {
		pdb.queryCache.Delete(ctx, cacheKey)
	}

	result, err := pdb.Database.Exec(ctx, query, args...)
	duration := time.Since(start)

	pdb.recordMetrics(queryType, table, operation, duration, err, false)

	return result, err
}

// Query executes a query with performance monitoring and caching
func (pdb *PerformanceDB) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	start := time.Now()
	queryType, table, operation := pdb.parseQuery(query)

	// Check cache for read operations
	if pdb.queryCache != nil && pdb.isCacheableQuery(query) {
		cacheKey := pdb.generateCacheKey(query, args)
		if cached, found := pdb.queryCache.Get(ctx, cacheKey); found {
			pdb.metrics.cacheHits.Inc()
			pdb.recordMetrics(queryType, table, operation, time.Since(start), nil, true)
			return cached.(pgx.Rows), nil
		}
		pdb.metrics.cacheMiss.Inc()
	}

	result, err := pdb.Database.Query(ctx, query, args...)
	duration := time.Since(start)

	// Cache successful read results
	if pdb.queryCache != nil && err == nil && pdb.isCacheableQuery(query) {
		cacheKey := pdb.generateCacheKey(query, args)
		pdb.queryCache.Set(ctx, cacheKey, result, 5*time.Minute) // 5-minute TTL
		pdb.metrics.cacheSets.Inc()
	}

	pdb.recordMetrics(queryType, table, operation, duration, err, false)

	return result, err
}

// QueryRow executes a query returning a single row with monitoring
func (pdb *PerformanceDB) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	start := time.Now()
	queryType, table, operation := pdb.parseQuery(query)

	// Check cache for read operations
	if pdb.queryCache != nil && pdb.isCacheableQuery(query) {
		cacheKey := pdb.generateCacheKey(query, args)
		if cached, found := pdb.queryCache.Get(ctx, cacheKey); found {
			pdb.metrics.cacheHits.Inc()
			pdb.recordMetrics(queryType, table, operation, time.Since(start), nil, true)
			return cached.(pgx.Row)
		}
		pdb.metrics.cacheMiss.Inc()
	}

	result := pdb.Database.QueryRow(ctx, query, args...)
	duration := time.Since(start)

	pdb.recordMetrics(queryType, table, operation, duration, nil, false)

	return result
}

// Begin begins a transaction with performance monitoring
func (pdb *PerformanceDB) Begin(ctx context.Context) (pgx.Tx, error) {
	start := time.Now()

	tx, err := pdb.Database.Begin(ctx)

	duration := time.Since(start)
	pdb.recordMetrics("transaction", "system", "begin", duration, err, false)

	return tx, err
}

// BeginTx begins a transaction with options and monitoring
func (pdb *PerformanceDB) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	start := time.Now()

	tx, err := pdb.Database.BeginTx(ctx, txOptions)

	duration := time.Since(start)
	pdb.recordMetrics("transaction", "system", "begin", duration, err, false)

	return tx, err
}

// Acquire acquires a connection with monitoring
func (pdb *PerformanceDB) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	start := time.Now()

	conn, err := pdb.Database.Acquire(ctx)

	if err == nil {
		pdb.metrics.connAcquires.Inc()
	}

	duration := time.Since(start)
	pdb.recordMetrics("connection", "pool", "acquire", duration, err, false)

	return conn, err
}

// UpdateMetrics updates connection pool metrics
func (pdb *PerformanceDB) UpdateMetrics() {
	stats := pdb.Database.Stats()
	pdb.metrics.activeConns.Set(float64(stats.AcquiredConns()))
	pdb.metrics.idleConns.Set(float64(stats.IdleConns()))
}

// GetQueryPatterns returns current query execution patterns
func (pdb *PerformanceDB) GetQueryPatterns() map[string]*PerformanceQueryPattern {
	pdb.patternMutex.RLock()
	defer pdb.patternMutex.RUnlock()

	patterns := make(map[string]*PerformanceQueryPattern)
	for k, v := range pdb.queryPatterns {
		patterns[k] = v
	}
	return patterns
}

// recordMetrics records performance metrics for a query
func (pdb *PerformanceDB) recordMetrics(queryType, table, operation string, duration time.Duration, err error, fromCache bool) {
	// Record query duration
	pdb.metrics.queryDuration.WithLabelValues(queryType, table, operation).Observe(duration.Seconds())

	// Record query count
	status := "success"
	if err != nil {
		status = "error"
		pdb.metrics.queryErrors.WithLabelValues(queryType, table, operation, pdb.getErrorType(err)).Inc()
	}

	if !fromCache {
		pdb.metrics.queryTotal.WithLabelValues(queryType, table, operation, status).Inc()
	}

	// Record slow queries
	if duration > pdb.slowQueryThreshold && !fromCache {
		pdb.metrics.slowQueries.Inc()
		pdb.logger.Warn().
			Str("query_type", queryType).
			Str("table", table).
			Str("operation", operation).
			Dur("duration", duration).
			Msg("Slow query detected")
	}

	// Update query patterns
	pdb.updateQueryPattern(queryType, table, operation, duration)
}

// parseQuery extracts query metadata for monitoring
func (pdb *PerformanceDB) parseQuery(query string) (queryType, table, operation string) {
	query = strings.TrimSpace(strings.ToLower(query))

	// Determine query type
	if strings.HasPrefix(query, "select") {
		queryType = "select"
	} else if strings.HasPrefix(query, "insert") {
		queryType = "insert"
	} else if strings.HasPrefix(query, "update") {
		queryType = "update"
	} else if strings.HasPrefix(query, "delete") {
		queryType = "delete"
	} else if strings.HasPrefix(query, "create") {
		queryType = "ddl"
	} else if strings.HasPrefix(query, "alter") {
		queryType = "ddl"
	} else if strings.HasPrefix(query, "drop") {
		queryType = "ddl"
	} else {
		queryType = "other"
	}

	// Extract table name using regex
	re := regexp.MustCompile(`(?:from|join|into|update)\s+([a-z_][a-z0-9_]*)`)
	matches := re.FindStringSubmatch(query)
	if len(matches) > 1 {
		table = matches[1]
	}

	// Determine operation based on query type and context
	if queryType == "select" {
		if strings.Contains(query, "count(") {
			operation = "count"
		} else if strings.Contains(query, "sum(") {
			operation = "aggregate"
		} else if strings.Contains(query, "group by") {
			operation = "group"
		} else if strings.Contains(query, "order by") {
			operation = "sort"
		} else {
			operation = "fetch"
		}
	} else {
		operation = queryType
	}

	return queryType, table, operation
}

// isCacheableQuery determines if a query can be cached
func (pdb *PerformanceDB) isCacheableQuery(query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))

	// Only cache SELECT queries
	if !strings.HasPrefix(query, "select") {
		return false
	}

	// Don't cache queries with certain patterns
	uncacheablePatterns := []string{
		"now()",
		"current_timestamp",
		"random()",
		"transaction",
		"lock",
	}

	for _, pattern := range uncacheablePatterns {
		if strings.Contains(query, pattern) {
			return false
		}
	}

	return true
}

// generateCacheKey creates a cache key for a query
func (pdb *PerformanceDB) generateCacheKey(query string, args []interface{}) string {
	// Simple cache key generation - in production, use proper hashing
	key := fmt.Sprintf("%s:%v", query, args)
	return fmt.Sprintf("db_query_%x", len(key)) // Simple hash
}

// getErrorType categorizes database errors
func (pdb *PerformanceDB) getErrorType(err error) string {
	if err == nil {
		return "none"
	}

	errStr := strings.ToLower(err.Error())

	if strings.Contains(errStr, "timeout") {
		return "timeout"
	} else if strings.Contains(errStr, "connection") {
		return "connection"
	} else if strings.Contains(errStr, "constraint") {
		return "constraint"
	} else if strings.Contains(errStr, "duplicate") {
		return "duplicate"
	} else if strings.Contains(errStr, "not found") {
		return "not_found"
	} else {
		return "unknown"
	}
}

// updateQueryPattern updates query execution statistics
func (pdb *PerformanceDB) updateQueryPattern(queryType, table, operation string, duration time.Duration) {
	patternKey := fmt.Sprintf("%s:%s:%s", queryType, table, operation)

	pdb.patternMutex.Lock()
	defer pdb.patternMutex.Unlock()

	pattern, exists := pdb.queryPatterns[patternKey]
	if !exists {
		pattern = &PerformanceQueryPattern{
			Query:     patternKey,
			MinTime:   duration,
			MaxTime:   duration,
			TotalTime: duration,
			Count:     1,
			LastUsed:  time.Now(),
		}
		pdb.queryPatterns[patternKey] = pattern
	}

	pattern.Count++
	pattern.TotalTime += duration
	pattern.LastUsed = time.Now()

	if duration > pattern.MaxTime {
		pattern.MaxTime = duration
	}
	if duration < pattern.MinTime {
		pattern.MinTime = duration
	}
}
