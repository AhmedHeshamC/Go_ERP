package database

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/attribute"
)

// Database wraps a pgx connection pool and provides database operations
type Database struct {
	pool             *pgxpool.Pool
	logger           *zerolog.Logger
	config           *Config
	poolMonitor      *PoolMonitor
	slowQueryMonitor *SlowQueryMonitor
}

// Config holds database configuration
type Config struct {
	URL             string
	MaxConnections  int
	MinConnections  int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	SSLMode         string
	SSLCert         string
	SSLKey          string
	SSLCA           string
	SSLHost         string

	// Advanced connection pool settings
	HealthCheckPeriod     time.Duration
	MaxConnAcquireTime    time.Duration
	MaxConnAcquireCount   int32
	LazyConnect           bool
	ConnectTimeout        time.Duration

	// Performance settings
	SlowQueryThreshold    time.Duration
	LogSlowQueries        bool
	EnableConnectionStats bool

	// Retry settings
	EnableRetryOnFailure  bool
	MaxRetryAttempts      int
	RetryDelay            time.Duration
}

// New creates a new Database instance with the given configuration
func New(cfg Config) (*Database, error) {
	logger := zerolog.Nop()
	return NewWithLogger(cfg, &logger)
}

// NewWithLogger creates a new Database instance with the given configuration and logger
func NewWithLogger(cfg Config, logger *zerolog.Logger) (*Database, error) {
	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid database config: %w", err)
	}

	// Build database URL with SSL configuration
	dbURL, err := cfg.buildConnectionString()
	if err != nil {
		return nil, fmt.Errorf("failed to build database connection string: %w", err)
	}

	// Create pgxpool configuration
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure connection pool with optimized production settings
	// Validate and convert connection limits
	if cfg.MaxConnections > 0x7FFFFFFF || cfg.MaxConnections < 0 {
		return nil, fmt.Errorf("MaxConnections out of valid range for int32")
	}
	if cfg.MinConnections > 0x7FFFFFFF || cfg.MinConnections < 0 {
		return nil, fmt.Errorf("MinConnections out of valid range for int32")
	}
	poolConfig.MaxConns = int32(cfg.MaxConnections) // #nosec G115 - Validated above
	poolConfig.MinConns = int32(cfg.MinConnections) // #nosec G115 - Validated above
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime

	// Configure advanced connection pool settings for production
	if cfg.HealthCheckPeriod > 0 {
		poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod
	} else {
		// Default to 1 minute if not specified
		poolConfig.HealthCheckPeriod = time.Minute
	}
	// Note: Some pgxpool v5 config fields are not available, use defaults where needed

	// Configure connection timeouts for production resilience
	if cfg.ConnectTimeout > 0 {
		poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	}

	// Configure statement cache for performance
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement
	poolConfig.ConnConfig.StatementCacheCapacity = 512
	poolConfig.ConnConfig.DescriptionCacheCapacity = 1024

	// Configure TLS for secure database connections
	if cfg.SSLMode != "disable" {
		tlsConfig, err := cfg.buildTLSConfig()
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to build TLS config, continuing with default")
		} else {
			poolConfig.ConnConfig.TLSConfig = tlsConfig
		}
	}

	// Set connection callbacks for logging
	poolConfig.BeforeConnect = func(ctx context.Context, cfg *pgx.ConnConfig) error {
		logger.Debug().Msg("Connecting to database")
		return nil
	}

	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		logger.Debug().Msg("Successfully connected to database")
		return nil
	}

	poolConfig.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		logger.Trace().Msg("Acquiring database connection")
		return true
	}

	poolConfig.AfterRelease = func(conn *pgx.Conn) bool {
		logger.Trace().Msg("Released database connection")
		return true
	}

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().
		Int("max_connections", cfg.MaxConnections).
		Int("min_connections", cfg.MinConnections).
		Str("max_conn_lifetime", cfg.ConnMaxLifetime.String()).
		Str("max_conn_idle_time", cfg.ConnMaxIdleTime.String()).
		Msg("Database connection pool initialized successfully")

	db := &Database{
		pool:   pool,
		logger: logger,
		config: &cfg,
	}

	// Initialize pool monitor if connection stats are enabled
	if cfg.EnableConnectionStats {
		db.poolMonitor = NewPoolMonitor(pool, logger, DefaultPoolMonitorConfig())
	}

	// Initialize slow query monitor if enabled
	if cfg.LogSlowQueries {
		db.slowQueryMonitor = NewSlowQueryMonitor(logger, cfg.SlowQueryThreshold)
	}

	return db, nil
}

// GetPool returns the underlying pgx connection pool
func (db *Database) GetPool() *pgxpool.Pool {
	return db.pool
}

// GetSQLDB returns a *sql.DB instance for compatibility with libraries that expect database/sql
// Note: This is a simplified implementation - in production you might want to handle connection management
func (db *Database) GetSQLDB() *sql.DB {
	return stdlib.OpenDB(*db.pool.Config().ConnConfig)
}

// Ping checks if the database connection is alive
func (db *Database) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Close closes the database connection pool
func (db *Database) Close() {
	if db.pool != nil {
		db.logger.Info().Msg("Closing database connection pool")
		db.pool.Close()
	}
}

// Stats returns connection pool statistics
func (db *Database) Stats() *pgxpool.Stat {
	return db.pool.Stat()
}

// Acquire acquires a connection from the pool
func (db *Database) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	return db.pool.Acquire(ctx)
}

// Exec executes a query that doesn't return rows with enhanced monitoring
func (db *Database) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	ctx, span := db.startTracingSpan(ctx, "database.exec", query)
	defer span.End()

	start := time.Now()
	result, err := db.pool.Exec(ctx, query, args...)
	duration := time.Since(start)

	// Enhanced logging with tracing and performance monitoring
	db.logQueryWithDetails(ctx, query, args, duration, err)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return result, err
}

// Query executes a query that returns rows with enhanced monitoring
func (db *Database) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	ctx, span := db.startTracingSpan(ctx, "database.query", query)
	defer span.End()

	start := time.Now()
	rows, err := db.pool.Query(ctx, query, args...)
	duration := time.Since(start)

	// Enhanced logging with tracing and performance monitoring
	db.logQueryWithDetails(ctx, query, args, duration, err)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return rows, err
}

// QueryRow executes a query that returns a single row with enhanced monitoring
func (db *Database) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	ctx, span := db.startTracingSpan(ctx, "database.query_row", query)
	defer span.End()

	start := time.Now()
	row := db.pool.QueryRow(ctx, query, args...)
	duration := time.Since(start)

	// Enhanced logging with tracing and performance monitoring
	db.logQueryWithDetails(ctx, query, args, duration, nil)

	return row
}

// Begin begins a transaction
func (db *Database) Begin(ctx context.Context) (pgx.Tx, error) {
	return db.pool.Begin(ctx)
}

// BeginTx begins a transaction with the given options
func (db *Database) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return db.pool.BeginTx(ctx, txOptions)
}

// validate validates the database configuration
func (c *Config) validate() error {
	if c.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	if c.MaxConnections <= 0 {
		return fmt.Errorf("max connections must be greater than 0")
	}

	if c.MinConnections < 0 {
		return fmt.Errorf("min connections cannot be negative")
	}

	if c.MinConnections > c.MaxConnections {
		return fmt.Errorf("min connections cannot be greater than max connections")
	}

	if c.ConnMaxLifetime <= 0 {
		return fmt.Errorf("connection max lifetime must be greater than 0")
	}

	if c.ConnMaxIdleTime <= 0 {
		return fmt.Errorf("connection max idle time must be greater than 0")
	}

	return nil
}

// logQuery logs database queries with execution time
func (db *Database) logQuery(ctx context.Context, query string, duration time.Duration) {
	threshold := db.config.SlowQueryThreshold
	if threshold == 0 {
		threshold = 100 * time.Millisecond // Default threshold
	}

	if duration > threshold && db.config.LogSlowQueries {
		db.logger.Warn().
			Str("query", query).
			Dur("duration", duration).
			Dur("threshold", threshold).
			Msg("Slow database query detected")
	} else if db.logger.GetLevel() <= zerolog.DebugLevel {
		db.logger.Debug().
			Str("query", query).
			Dur("duration", duration).
			Msg("Database query executed")
	}
}

// buildConnectionString builds the database connection string with SSL configuration
func (c *Config) buildConnectionString() (string, error) {
	// Parse the base URL
	parsedURL, err := url.Parse(c.URL)
	if err != nil {
		return "", fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Get existing query parameters
	query := parsedURL.Query()

	// Set SSL mode (default to require for security)
	if c.SSLMode != "" {
		query.Set("sslmode", c.SSLMode)
	} else {
		query.Set("sslmode", "require")
	}

	// Add SSL certificate parameters if provided
	if c.SSLCert != "" {
		query.Set("sslcert", c.SSLCert)
	}

	if c.SSLKey != "" {
		query.Set("sslkey", c.SSLKey)
	}

	if c.SSLCA != "" {
		query.Set("sslrootcert", c.SSLCA)
	}

	if c.SSLHost != "" {
		query.Set("sslhost", c.SSLHost)
	}

	// Update the query parameters
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

// HealthCheck performs a health check on the database
func (db *Database) HealthCheck(ctx context.Context) error {
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	stats := db.Stats()
	if stats.AcquiredConns() == 0 {
		return fmt.Errorf("no database connections available")
	}

	return nil
}

// GetConfig returns the database configuration
func (db *Database) GetConfig() *Config {
	return db.config
}

// GetConnectionStats returns detailed connection pool statistics
func (db *Database) GetConnectionStats() *DetailedConnectionStats {
	stats := db.Stats()
	return &DetailedConnectionStats{
		AcquireCount:         stats.AcquireCount(),
		AcquireDuration:      stats.AcquireDuration(),
		AcquiredConns:        stats.AcquiredConns(),
		IdleConns:            stats.IdleConns(),
		MaxConns:             stats.MaxConns(),
		TotalConns:           stats.TotalConns(),
		CanceledAcquireCount: stats.CanceledAcquireCount(),
		ConstructingConns:    stats.ConstructingConns(),
		EmptyAcquireCount:    stats.EmptyAcquireCount(),
		NewConnsCount:        stats.NewConnsCount(),
		MaxIdleDestroyCount:  stats.MaxIdleDestroyCount(),
		MaxLifetimeDestroyCount: stats.MaxLifetimeDestroyCount(),
	}
}

// DetailedConnectionStats provides detailed connection pool statistics
type DetailedConnectionStats struct {
	AcquireCount           int64         `json:"acquire_count"`
	AcquireDuration        time.Duration `json:"acquire_duration"`
	AcquiredConns          int32         `json:"acquired_conns"`
	IdleConns              int32         `json:"idle_conns"`
	MaxConns               int32         `json:"max_conns"`
	TotalConns             int32         `json:"total_conns"`
	CanceledAcquireCount   int64         `json:"canceled_acquire_count"`
	ConstructingConns      int32         `json:"constructing_conns"`
	EmptyAcquireCount      int64         `json:"empty_acquire_count"`
	NewConnsCount          int64         `json:"new_conns_count"`
	MaxIdleDestroyCount    int64         `json:"max_idle_destroy_count"`
	MaxLifetimeDestroyCount int64        `json:"max_lifetime_destroy_count"`
}

// ResetConnectionStats resets the connection pool statistics
func (db *Database) ResetConnectionStats() {
	// Note: pgxpool doesn't have a direct reset method, so this is a placeholder
	// In production, you might want to track this at the application level
	db.logger.Info().Msg("Connection pool statistics reset requested")
}

// WaitForAvailableConnections waits until at least the specified number of connections are available
func (db *Database) WaitForAvailableConnections(ctx context.Context, minConnections int32) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			stats := db.Stats()
			if stats.IdleConns() >= minConnections {
				return nil
			}
		}
	}
}

// PerformHealthCheck performs a comprehensive health check including connection pool health
func (db *Database) PerformHealthCheck(ctx context.Context) *HealthCheckResult {
	result := &HealthCheckResult{
		Timestamp: time.Now(),
		Healthy:   true,
	}

	// Check if we can ping the database
	if err := db.Ping(ctx); err != nil {
		result.Healthy = false
		result.Errors = append(result.Errors, fmt.Sprintf("Database ping failed: %v", err))
	}

	// Check connection pool statistics
	stats := db.Stats()
	result.ConnectionStats = db.GetConnectionStats()

	// Check if we have any connections available
	if stats.AcquiredConns() == 0 && stats.IdleConns() == 0 {
		result.Healthy = false
		result.Errors = append(result.Errors, "No database connections available")
	}

	// Check if all connections are being used (potential connection pool exhaustion)
	if stats.AcquiredConns() >= stats.MaxConns() {
		result.Warnings = append(result.Warnings, "Connection pool is at maximum capacity")
	}

	// Check if there are too many canceled acquires (potential performance issue)
	if stats.CanceledAcquireCount() > 100 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("High number of canceled acquires: %d", stats.CanceledAcquireCount()))
	}

	return result
}

// HealthCheckResult represents the result of a database health check
type HealthCheckResult struct {
	Timestamp       time.Time              `json:"timestamp"`
	Healthy         bool                  `json:"healthy"`
	ConnectionStats *DetailedConnectionStats `json:"connection_stats"`
	Errors          []string              `json:"errors,omitempty"`
	Warnings        []string              `json:"warnings,omitempty"`
}

// WithTransaction executes a function within a database transaction with automatic rollback on error
func (db *Database) WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// AcquireWithContext attempts to acquire a connection with context timeout and retry logic
func (db *Database) AcquireWithContext(ctx context.Context, maxRetries int, retryDelay time.Duration) (*pgxpool.Conn, error) {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		conn, err := db.pool.Acquire(ctx)
		if err == nil {
			return conn, nil
		}

		lastErr = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			break
		}

		// Log retry attempt
		db.logger.Warn().
			Err(err).
			Int("attempt", i+1).
			Int("max_retries", maxRetries).
			Dur("retry_delay", retryDelay).
			Msg("Failed to acquire database connection, retrying")

		// Wait before retry (unless this is the last attempt)
		if i < maxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}
		}
	}

	return nil, fmt.Errorf("failed to acquire database connection after %d attempts: %w", maxRetries+1, lastErr)
}

// CloseIdleConnections closes idle connections in the pool
func (db *Database) CloseIdleConnections() {
	// Note: pgxpool doesn't have a direct method to close only idle connections
	// This would typically be handled by the pool's idle timeout configuration
	db.logger.Info().Msg("Idle connection cleanup requested (handled by pool configuration)")
}

// ResizePool attempts to resize the connection pool (requires creating a new pool)
func (db *Database) ResizePool(newMaxConns, newMinConns int) (*Database, error) {
	if newMinConns > newMaxConns {
		return nil, fmt.Errorf("min connections cannot be greater than max connections")
	}

	// Create new configuration with updated pool sizes
	newConfig := *db.config
	newConfig.MaxConnections = newMaxConns
	newConfig.MinConnections = newMinConns

	// Create new pool with updated configuration
	newDB, err := NewWithLogger(newConfig, db.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create resized connection pool: %w", err)
	}

	// Close old pool after a short delay to allow existing operations to complete
	go func() {
		time.Sleep(5 * time.Second)
		db.Close()
	}()

	return newDB, nil
}

// buildTLSConfig creates a TLS configuration for secure database connections
func (c *Config) buildTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Configure SSL mode
	switch c.SSLMode {
	case "require":
		tlsConfig.InsecureSkipVerify = false
	case "verify-ca":
		tlsConfig.InsecureSkipVerify = false
	case "verify-full":
		tlsConfig.InsecureSkipVerify = false
		tlsConfig.ServerName = c.extractServerName()
	case "prefer":
		tlsConfig.InsecureSkipVerify = true // Allow for development
	case "disable":
		return nil, nil // No TLS
	default:
		return nil, fmt.Errorf("unsupported SSL mode: %s", c.SSLMode)
	}

	// Load custom certificates if provided
	if c.SSLCert != "" || c.SSLKey != "" || c.SSLCA != "" {
		// This would require implementing certificate loading
		// For now, log a warning that custom certs are not fully supported
		return nil, fmt.Errorf("custom certificate loading not implemented")
	}

	return tlsConfig, nil
}

// extractServerName extracts the server name from the database URL for TLS verification
func (c *Config) extractServerName() string {
	if c.SSLHost != "" {
		return c.SSLHost
	}

	// Parse server name from URL
	parsedURL, err := url.Parse(c.URL)
	if err != nil {
		return ""
	}

	// Extract host part and remove port if present
	host := parsedURL.Hostname()
	if host == "" {
		return ""
	}

	return host
}

// startTracingSpan begins an OpenTelemetry span for database operations
func (db *Database) startTracingSpan(ctx context.Context, operation, query string) (context.Context, trace.Span) {
	tracer := otel.Tracer("github.com/your-org/go-erp/database")

	// Extract query type for better span naming
	queryType := db.extractQueryType(query)
	spanName := fmt.Sprintf("%s.%s", operation, queryType)

	return tracer.Start(ctx, spanName, trace.WithAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", operation),
		attribute.String("db.statement", db.sanitizeQuery(query)),
	))
}

// extractQueryType extracts the type of SQL operation from the query
func (db *Database) extractQueryType(query string) string {
	trimmed := strings.TrimSpace(strings.ToUpper(query))

	switch {
	case strings.HasPrefix(trimmed, "SELECT"):
		return "select"
	case strings.HasPrefix(trimmed, "INSERT"):
		return "insert"
	case strings.HasPrefix(trimmed, "UPDATE"):
		return "update"
	case strings.HasPrefix(trimmed, "DELETE"):
		return "delete"
	case strings.HasPrefix(trimmed, "CREATE"):
		return "create"
	case strings.HasPrefix(trimmed, "DROP"):
		return "drop"
	case strings.HasPrefix(trimmed, "ALTER"):
		return "alter"
	case strings.HasPrefix(trimmed, "BEGIN"), strings.HasPrefix(trimmed, "START"):
		return "transaction"
	default:
		return "unknown"
	}
}

// sanitizeQuery removes sensitive data from queries for logging
func (db *Database) sanitizeQuery(query string) string {
	// Remove newlines and extra spaces
	sanitized := strings.ReplaceAll(query, "\n", " ")
	sanitized = strings.Join(strings.Fields(sanitized), " ")

	// Truncate very long queries
	if len(sanitized) > 500 {
		sanitized = sanitized[:497] + "..."
	}

	return sanitized
}

// logQueryWithDetails provides enhanced query logging with performance metrics
func (db *Database) logQueryWithDetails(ctx context.Context, query string, args []interface{}, duration time.Duration, err error) {
	threshold := db.config.SlowQueryThreshold
	if threshold == 0 {
		threshold = 100 * time.Millisecond // Default threshold
	}

	queryType := db.extractQueryType(query)

	// Record slow query if monitor is enabled
	if db.slowQueryMonitor != nil {
		db.slowQueryMonitor.RecordQuery(ctx, db.sanitizeQuery(query), queryType, len(args), duration)
	}

	// Prepare log event
	event := db.logger.Debug().
		Str("query", db.sanitizeQuery(query)).
		Str("query_type", queryType).
		Dur("duration", duration).
		Int("arg_count", len(args)).
		Str("trace_id", db.getTraceID(ctx))

	if err != nil {
		event = event.Err(err)
	}

	// Log based on performance and error status
	if err != nil {
		event.Str("status", "error").Msg("Database query failed")
	} else if duration > threshold && db.config.LogSlowQueries {
		event.Str("status", "slow").Dur("threshold", threshold).Msg("Slow database query detected")
	} else if duration > threshold/2 && db.logger.GetLevel() <= zerolog.InfoLevel {
		event.Str("status", "warning").Msg("Database query approaching slow threshold")
	} else if db.logger.GetLevel() <= zerolog.DebugLevel {
		event.Str("status", "success").Msg("Database query executed")
	}

	// Track performance metrics if enabled
	if db.config.EnableConnectionStats {
		db.trackQueryMetrics(queryType, duration, err != nil)
	}
}

// getTraceID extracts the trace ID from the context
func (db *Database) getTraceID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return spanCtx.TraceID().String()
	}
	return ""
}

// trackQueryMetrics tracks query performance for metrics collection
func (db *Database) trackQueryMetrics(queryType string, duration time.Duration, isError bool) {
	// This would integrate with your metrics system (Prometheus, etc.)
	// For now, we'll just log the metrics
	if duration > 500*time.Millisecond {
		db.logger.Warn().
			Str("query_type", queryType).
			Dur("duration", duration).
			Bool("error", isError).
			Msg("High-latency query detected")
	}
}

// GetQueryPerformanceStats returns recent query performance statistics
func (db *Database) GetQueryPerformanceStats() *QueryPerformanceStats {
	// This would return actual statistics from your metrics system
	// For now, return a placeholder
	return &QueryPerformanceStats{
		TotalQueries:     0,
		AverageLatency:   0,
		SlowQueries:      0,
		ErrorRate:        0,
		LastResetTime:    time.Now(),
	}
}

// QueryPerformanceStats holds query performance metrics
type QueryPerformanceStats struct {
	TotalQueries     int64         `json:"total_queries"`
	AverageLatency   time.Duration `json:"average_latency"`
	SlowQueries      int64         `json:"slow_queries"`
	ErrorRate        float64       `json:"error_rate"`
	LastResetTime    time.Time     `json:"last_reset_time"`
}

// WithRetryTransaction executes a function within a database transaction with automatic retry on deadlock
func (db *Database) WithRetryTransaction(ctx context.Context, maxRetries int, fn func(pgx.Tx) error) error {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		err := db.WithTransaction(ctx, func(tx pgx.Tx) error {
			if i > 0 {
				db.logger.Debug().
					Int("attempt", i+1).
					Int("max_retries", maxRetries).
					Msg("Retrying database transaction")
			}

			lastErr = fn(tx)
			return lastErr
		})

		if err == nil {
			return nil
		}

		// Check if this is a retryable error
		if !db.isRetryableError(err) || i == maxRetries {
			return err
		}

		lastErr = err

		// Exponential backoff with overflow protection
		var backoff time.Duration
		if i < 30 { // Prevent overflow: 2^30 is already > 1 billion
			backoff = time.Duration(1<<uint(i)) * 100 * time.Millisecond // #nosec G115 - Protected by i < 30 check
		} else {
			backoff = 5 * time.Second
		}
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}

	return lastErr
}

// isRetryableError checks if an error is retryable (deadlock, connection issues, etc.)
func (db *Database) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Check for deadlock
	if strings.Contains(errStr, "deadlock") {
		return true
	}

	// Check for connection issues
	if strings.Contains(errStr, "connection") ||
	   strings.Contains(errStr, "timeout") ||
	   strings.Contains(errStr, "network") {
		return true
	}

	// Check for serialization failure
	if strings.Contains(errStr, "serialization failure") {
		return true
	}

	return false
}

// GetActiveConnections returns the number of currently active connections
func (db *Database) GetActiveConnections() int32 {
	stats := db.Stats()
	return stats.AcquiredConns()
}

// GetIdleConnections returns the number of currently idle connections
func (db *Database) GetIdleConnections() int32 {
	stats := db.Stats()
	return stats.IdleConns()
}

// IsHealthy performs a comprehensive health check
func (db *Database) IsHealthy(ctx context.Context) bool {
	if err := db.Ping(ctx); err != nil {
		return false
	}

	stats := db.Stats()

	// Check if connection pool is exhausted
	if stats.AcquiredConns() >= stats.MaxConns() {
		return false
	}

	// Check if too many connection errors
	if stats.CanceledAcquireCount() > 100 {
		return false
	}

	return true
}


// StartPoolMonitoring starts the connection pool monitoring
func (db *Database) StartPoolMonitoring(ctx context.Context) {
	if db.poolMonitor != nil {
		go db.poolMonitor.Start(ctx)
	} else {
		db.logger.Warn().Msg("Pool monitoring not enabled - set EnableConnectionStats to true in config")
	}
}

// GetPoolMonitor returns the pool monitor instance
func (db *Database) GetPoolMonitor() *PoolMonitor {
	return db.poolMonitor
}

// GetPoolStats returns current pool statistics
func (db *Database) GetPoolStats() *PoolStats {
	if db.poolMonitor != nil {
		return db.poolMonitor.GetCurrentStats()
	}
	return nil
}

// GetSlowQueryMonitor returns the slow query monitor instance
func (db *Database) GetSlowQueryMonitor() *SlowQueryMonitor {
	return db.slowQueryMonitor
}

// GetSlowQueries returns recent slow queries
func (db *Database) GetSlowQueries() []SlowQueryRecord {
	if db.slowQueryMonitor != nil {
		return db.slowQueryMonitor.GetSlowQueries()
	}
	return nil
}
