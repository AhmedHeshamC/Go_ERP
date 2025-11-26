package database

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

// ReadReplicaConfig holds configuration for read replicas
type ReadReplicaConfig struct {
	PrimaryURL    string        `json:"primary_url"`
	ReplicaURLs   []string      `json:"replica_urls"`
	ReadTimeout   time.Duration `json:"read_timeout"`
	WriteTimeout  time.Duration `json:"write_timeout"`
	MaxRetries    int           `json:"max_retries"`
	RetryDelay    time.Duration `json:"retry_delay"`
	HealthCheck   time.Duration `json:"health_check"`
}

// ReplicaDB manages read replicas for database operations
type ReplicaDB struct {
	primary   *Database
	replicas  []*Database
	config    *ReadReplicaConfig
	logger    *zerolog.Logger
	healthy   map[int]bool // Track health of each replica
}

// NewReplicaDB creates a new ReplicaDB with primary and read replicas
func NewReplicaDB(primaryConfig Config, replicaConfigs []Config, replicaConfig *ReadReplicaConfig, logger *zerolog.Logger) (*ReplicaDB, error) {
	if logger == nil {
		l := zerolog.Nop()
		logger = &l
	}

	// Create primary database connection
	primary, err := NewWithLogger(primaryConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create primary database: %w", err)
	}

	// Create replica connections
	replicas := make([]*Database, 0, len(replicaConfigs))
	for i, config := range replicaConfigs {
		replica, err := NewWithLogger(config, logger)
		if err != nil {
			logger.Warn().
				Int("replica_index", i).
				Err(err).
				Msg("Failed to create replica, skipping")
			continue
		}
		replicas = append(replicas, replica)
	}

	if len(replicas) == 0 {
		logger.Warn().Msg("No read replicas available, using primary only")
	}

	// Initialize replica manager
	replicaDB := &ReplicaDB{
		primary:  primary,
		replicas: replicas,
		config:   replicaConfig,
		logger:   logger,
		healthy:  make(map[int]bool),
	}

	// Mark all replicas as healthy initially
	for i := range replicas {
		replicaDB.healthy[i] = true
	}

	// Start health checking
	go replicaDB.healthCheckLoop()

	logger.Info().
		Int("replica_count", len(replicas)).
		Msg("ReplicaDB initialized successfully")

	return replicaDB, nil
}

// GetPrimary returns the primary database connection
func (rdb *ReplicaDB) GetPrimary() *Database {
	return rdb.primary
}

// GetReplica returns a healthy replica for read operations
func (rdb *ReplicaDB) GetReplica() *Database {
	if len(rdb.replicas) == 0 {
		return rdb.primary
	}

	// Find healthy replicas
	healthyReplicas := make([]*Database, 0)
	healthyIndices := make([]int, 0)

	for i, replica := range rdb.replicas {
		if rdb.healthy[i] {
			healthyReplicas = append(healthyReplicas, replica)
			healthyIndices = append(healthyIndices, i)
		}
	}

	if len(healthyReplicas) == 0 {
		rdb.logger.Warn().Msg("No healthy replicas available, using primary for read")
		return rdb.primary
	}

	// Use round-robin or random selection for load balancing
	// G404: Use crypto/rand instead of math/rand for security
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(healthyIndices))))
	if err != nil {
		// Fallback to first healthy replica if random fails
		selectedIndex := healthyIndices[0]
		rdb.logger.Warn().Err(err).Msg("Failed to generate random index, using first replica")
		return rdb.replicas[selectedIndex]
	}
	selectedIndex := healthyIndices[n.Int64()]

	rdb.logger.Debug().
		Int("selected_replica", selectedIndex).
		Msg("Using replica for read operation")

	return rdb.replicas[selectedIndex]
}

// Exec executes a write operation on the primary database
func (rdb *ReplicaDB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return rdb.executeWithRetry(ctx, rdb.primary, query, args...)
}

// Query executes a read operation on a replica
func (rdb *ReplicaDB) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	db := rdb.GetReplica()
	return db.Query(ctx, query, args...)
}

// QueryRow executes a read operation on a replica returning a single row
func (rdb *ReplicaDB) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	db := rdb.GetReplica()
	return db.QueryRow(ctx, query, args...)
}

// Begin begins a transaction on the primary database
func (rdb *ReplicaDB) Begin(ctx context.Context) (pgx.Tx, error) {
	return rdb.primary.Begin(ctx)
}

// BeginTx begins a transaction with options on the primary database
func (rdb *ReplicaDB) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return rdb.primary.BeginTx(ctx, txOptions)
}

// Close closes all database connections
func (rdb *ReplicaDB) Close() {
	rdb.primary.Close()
	for _, replica := range rdb.replicas {
		replica.Close()
	}
}

// Stats returns aggregated statistics from all databases
func (rdb *ReplicaDB) Stats() map[string]interface{} {
	stats := make(map[string]interface{})

	primaryStats := rdb.primary.Stats()
	stats["primary"] = map[string]interface{}{
		"acquired_conns": primaryStats.AcquiredConns(),
		"idle_conns":     primaryStats.IdleConns(),
		"total_conns":    primaryStats.TotalConns(),
		"max_conns":      primaryStats.MaxConns(),
	}

	replicaStats := make([]map[string]interface{}, len(rdb.replicas))
	for i, replica := range rdb.replicas {
		s := replica.Stats()
		replicaStats[i] = map[string]interface{}{
			"index":          i,
			"healthy":        rdb.healthy[i],
			"acquired_conns": s.AcquiredConns(),
			"idle_conns":     s.IdleConns(),
			"total_conns":    s.TotalConns(),
			"max_conns":      s.MaxConns(),
		}
	}
	stats["replicas"] = replicaStats

	return stats
}

// executeWithRetry executes a query with retry logic
func (rdb *ReplicaDB) executeWithRetry(ctx context.Context, db *Database, query string, args ...interface{}) (pgconn.CommandTag, error) {
	var lastErr error
	maxRetries := rdb.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Add delay for retry
			select {
			case <-ctx.Done():
				return pgconn.CommandTag{}, ctx.Err()
			case <-time.After(rdb.config.RetryDelay):
			}
		}

		result, err := db.Exec(ctx, query, args...)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !rdb.isRetryableError(err) {
			break
		}

		rdb.logger.Warn().
			Int("attempt", attempt+1).
			Err(err).
			Str("query", rdb.sanitizeQuery(query)).
			Msg("Database operation failed, retrying")
	}

	return pgconn.CommandTag{}, fmt.Errorf("operation failed after %d attempts: %w", maxRetries+1, lastErr)
}

// isRetryableError determines if an error is retryable
func (rdb *ReplicaDB) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Retry on connection errors
	retryableErrors := []string{
		"connection",
		"timeout",
		"network",
		"temporary",
		"deadlock",
		"connection refused",
		"connection reset",
		"broken pipe",
	}

	for _, retryableErr := range retryableErrors {
		if strings.Contains(errStr, retryableErr) {
			return true
		}
	}

	return false
}

// sanitizeQuery removes sensitive information from query for logging
func (rdb *ReplicaDB) sanitizeQuery(query string) string {
	// Simple sanitization - in production, use proper query parsing
	if len(query) > 200 {
		return query[:200] + "..."
	}
	return query
}

// healthCheckLoop performs periodic health checks on replicas
func (rdb *ReplicaDB) healthCheckLoop() {
	if rdb.config.HealthCheck <= 0 {
		rdb.config.HealthCheck = 30 * time.Second
	}

	ticker := time.NewTicker(rdb.config.HealthCheck)
	defer ticker.Stop()

	for range ticker.C {
		rdb.performHealthCheck()
	}
}

// performHealthCheck checks the health of all replicas
func (rdb *ReplicaDB) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i, replica := range rdb.replicas {
		err := replica.Ping(ctx)
		wasHealthy := rdb.healthy[i]
		rdb.healthy[i] = err == nil

		if wasHealthy && err != nil {
			rdb.logger.Warn().
				Int("replica_index", i).
				Err(err).
				Msg("Replica became unhealthy")
		} else if !wasHealthy && err == nil {
			rdb.logger.Info().
				Int("replica_index", i).
				Msg("Replica became healthy")
		}
	}
}

// QueryBuilder helps build read/write queries
type QueryBuilder struct {
	query string
	args  []interface{}
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{}
}

// Select adds a SELECT clause
func (qb *QueryBuilder) Select(columns string) *QueryBuilder {
	qb.query = "SELECT " + columns
	return qb
}

// From adds a FROM clause
func (qb *QueryBuilder) From(table string) *QueryBuilder {
	qb.query += " FROM " + table
	return qb
}

// Where adds a WHERE clause
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	if !strings.Contains(qb.query, "WHERE") {
		qb.query += " WHERE " + condition
	} else {
		qb.query += " AND " + condition
	}
	qb.args = append(qb.args, args...)
	return qb
}

// OrderBy adds an ORDER BY clause
func (qb *QueryBuilder) OrderBy(columns string) *QueryBuilder {
	qb.query += " ORDER BY " + columns
	return qb
}

// Limit adds a LIMIT clause
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.query += fmt.Sprintf(" LIMIT %d", limit)
	return qb
}

// Offset adds an OFFSET clause
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.query += fmt.Sprintf(" OFFSET %d", offset)
	return qb
}

// Build returns the final query and arguments
func (qb *QueryBuilder) Build() (string, []interface{}) {
	return qb.query, qb.args
}

// IsWriteQuery determines if a query is a write operation
func (qb *QueryBuilder) IsWriteQuery(query string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	writePrefixes := []string{"insert", "update", "delete", "create", "alter", "drop"}

	for _, prefix := range writePrefixes {
		if strings.HasPrefix(query, prefix) {
			return true
		}
	}

	return false
}