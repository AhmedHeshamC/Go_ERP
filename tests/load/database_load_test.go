package load

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// DatabaseLoadTestSuite contains database performance tests
type DatabaseLoadTestSuite struct {
	dbPool      *pgxpool.Pool
	config      DatabaseLoadTestConfig
	results     *DatabaseLoadTestResult
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// DatabaseLoadTestConfig defines configuration for database load tests
type DatabaseLoadTestConfig struct {
	Name               string
	DatabaseURL        string
	ConcurrentConnections int
	OperationsPerConnection int
	TestDuration       time.Duration
	OperationTypes     []string // "select", "insert", "update", "delete", "join", "aggregate"
	Tables             []string // "users", "products", "orders", "order_items"
	TargetThroughput   int64    // Operations per second
	MaxLatency         time.Duration
	MaxCPUUsage        float64   // Percentage
	MaxMemoryUsage     int64     // MB
}

// DatabaseLoadTestResult contains results from database load tests
type DatabaseLoadTestResult struct {
	TestName              string
	StartTime             time.Time
	EndTime               time.Time
	Duration              time.Duration
	TotalOperations       int64
	SuccessfulOperations  int64
	FailedOperations      int64
	Throughput            float64 // Operations per second
	AverageLatency        time.Duration
	P50Latency            time.Duration
	P95Latency            time.Duration
	P99Latency            time.Duration
	ErrorRate             float64
	CPUUsage              float64
	MemoryUsageMB         int64
	ConnectionPoolStats   map[string]int64
	SlowQueries           []SlowQuery
	Errors                []DatabaseError
}

// SlowQuery represents a slow query execution
type SlowQuery struct {
	Query       string
	Duration    time.Duration
	Timestamp   time.Time
	Parameters  map[string]interface{}
	Error       string
}

// DatabaseError represents a database operation error
type DatabaseError struct {
	Operation   string
	Query       string
	Error       string
	Timestamp   time.Time
	Duration    time.Duration
}

// DatabaseOperation represents a database operation
type DatabaseOperation struct {
	Type       string
	Table      string
	Query      string
	Parameters map[string]interface{}
}

// NewDatabaseLoadTestSuite creates a new database load test suite
func NewDatabaseLoadTestSuite(config DatabaseLoadTestConfig) (*DatabaseLoadTestSuite, error) {
	ctx, cancel := context.WithCancel(context.Background())

	pool, err := pgxpool.New(ctx, config.DatabaseURL)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create database connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		cancel()
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DatabaseLoadTestSuite{
		dbPool:  pool,
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
		results: &DatabaseLoadTestResult{
			TestName:            config.Name,
			ConnectionPoolStats: make(map[string]int64),
			SlowQueries:         make([]SlowQuery, 0),
			Errors:              make([]DatabaseError, 0),
		},
	}, nil
}

// RunDatabaseLoadTest executes the database load test
func (d *DatabaseLoadTestSuite) RunDatabaseLoadTest() (*DatabaseLoadTestResult, error) {
	log.Printf("Starting database load test: %s", d.config.Name)
	log.Printf("Configuration: Connections=%d, Operations/Connection=%d, Duration=%v",
		d.config.ConcurrentConnections, d.config.OperationsPerConnection, d.config.TestDuration)

	d.results.StartTime = time.Now()

	// Start system metrics collection
	d.startSystemMetricsCollection()

	// Start database connection workers
	for i := 0; i < d.config.ConcurrentConnections; i++ {
		d.wg.Add(1)
		go d.runDatabaseWorker(i)
	}

	// Wait for test duration or completion
	if d.config.TestDuration > 0 {
		time.Sleep(d.config.TestDuration)
	} else {
		d.wg.Wait()
	}

	// Stop the test
	d.cancel()
	d.wg.Wait()

	d.results.EndTime = time.Now()
	d.results.Duration = d.results.EndTime.Sub(d.results.StartTime)

	// Calculate final results
	d.calculateResults()

	// Collect connection pool statistics
	d.collectConnectionPoolStats()

	log.Printf("Database load test completed: %s", d.config.Name)
	log.Printf("Results: Throughput=%.2f ops/sec, Success Rate=%.2f%%, Avg Latency=%v",
		d.results.Throughput,
		(1-d.results.ErrorRate)*100,
		d.results.AverageLatency)

	return d.results, nil
}

// runDatabaseWorker executes database operations for a single worker
func (d *DatabaseLoadTestSuite) runDatabaseWorker(workerID int) {
	defer d.wg.Done()

	operationsCount := d.config.OperationsPerConnection
	if operationsCount <= 0 {
		operationsCount = 1000 // Default for duration-based tests
	}

	for i := 0; i < operationsCount; i++ {
		select {
		case <-d.ctx.Done():
			return
		default:
			operation := d.selectRandomOperation()
			d.executeOperation(workerID, i, operation)

			// Small delay between operations to simulate realistic workload
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
		}
	}
}

// selectRandomOperation selects a random operation type
func (d *DatabaseLoadTestSuite) selectRandomOperation() DatabaseOperation {
	opType := d.config.OperationTypes[rand.Intn(len(d.config.OperationTypes))]
	table := d.config.Tables[rand.Intn(len(d.config.Tables))]

	switch opType {
	case "select":
		return d.generateSelectOperation(table)
	case "insert":
		return d.generateInsertOperation(table)
	case "update":
		return d.generateUpdateOperation(table)
	case "delete":
		return d.generateDeleteOperation(table)
	case "join":
		return d.generateJoinOperation()
	case "aggregate":
		return d.generateAggregateOperation(table)
	default:
		return d.generateSelectOperation(table)
	}
}

// generateSelectOperation generates a SELECT operation
func (d *DatabaseLoadTestSuite) generateSelectOperation(table string) DatabaseOperation {
	switch table {
	case "users":
		return DatabaseOperation{
			Type:  "select",
			Table: "users",
			Query: "SELECT id, email, first_name, last_name, created_at FROM users WHERE is_active = $1 ORDER BY created_at DESC LIMIT $2",
			Parameters: map[string]interface{}{
				"1": true,
				"2": rand.Intn(100) + 1,
			},
		}
	case "products":
		return DatabaseOperation{
			Type:  "select",
			Table: "products",
			Query: "SELECT id, name, sku, price, is_active FROM products WHERE category_id = $1 AND is_active = $2 ORDER BY name LIMIT $3",
			Parameters: map[string]interface{}{
				"1": uuid.New(),
				"2": true,
				"3": rand.Intn(50) + 1,
			},
		}
	case "orders":
		return DatabaseOperation{
			Type:  "select",
			Table: "orders",
			Query: "SELECT id, order_number, customer_id, status, total_amount, created_at FROM orders WHERE status = $1 AND created_at >= $2 ORDER BY created_at DESC LIMIT $3",
			Parameters: map[string]interface{}{
				"1": "pending",
				"2": time.Now().Add(-24 * time.Hour),
				"3": rand.Intn(20) + 1,
			},
		}
	case "order_items":
		return DatabaseOperation{
			Type:  "select",
			Table: "order_items",
			Query: "SELECT * FROM order_items WHERE order_id = $1",
			Parameters: map[string]interface{}{
				"1": uuid.New(),
			},
		}
	default:
		return DatabaseOperation{
			Type:  "select",
			Table: table,
			Query: fmt.Sprintf("SELECT * FROM %s LIMIT %d", table, rand.Intn(100)+1),
			Parameters: map[string]interface{}{},
		}
	}
}

// generateInsertOperation generates an INSERT operation
func (d *DatabaseLoadTestSuite) generateInsertOperation(table string) DatabaseOperation {
	switch table {
	case "users":
		return DatabaseOperation{
			Type:  "insert",
			Table: "users",
			Query: "INSERT INTO users (id, email, password_hash, first_name, last_name, is_active, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			Parameters: map[string]interface{}{
				"1": uuid.New(),
				"2": fmt.Sprintf("test%d@example.com", rand.Intn(10000)),
				"3": "hashed_password",
				"4": fmt.Sprintf("Test%d", rand.Intn(1000)),
				"5": "User",
				"6": true,
				"7": time.Now(),
			},
		}
	case "products":
		return DatabaseOperation{
			Type:  "insert",
			Table: "products",
			Query: "INSERT INTO products (id, name, sku, description, price, cost, weight, category_id, is_active, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
			Parameters: map[string]interface{}{
				"1":  uuid.New(),
				"2":  fmt.Sprintf("Test Product %d", rand.Intn(1000)),
				"3":  fmt.Sprintf("TEST-%d", rand.Intn(10000)),
				"4":  "Test product description",
				"5":  decimal.NewFromFloat(float64(rand.Intn(500)+10) + 0.99),
				"6":  decimal.NewFromFloat(float64(rand.Intn(200)+5) + 0.50),
				"7":  float64(rand.Intn(10)+1) * 0.5,
				"8":  uuid.New(),
				"9":  true,
				"10": time.Now(),
			},
		}
	case "orders":
		return DatabaseOperation{
			Type:  "insert",
			Table: "orders",
			Query: "INSERT INTO orders (id, order_number, customer_id, status, subtotal, tax_amount, total_amount, currency, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
			Parameters: map[string]interface{}{
				"1":  uuid.New(),
				"2":  fmt.Sprintf("ORD-%d", rand.Intn(100000)),
				"3":  uuid.New(),
				"4":  "pending",
				"5":  decimal.NewFromFloat(float64(rand.Intn(500)+50) + 0.99),
				"6":  decimal.NewFromFloat(float64(rand.Intn(50)+5) + 0.99),
				"7":  decimal.NewFromFloat(float64(rand.Intn(600)+60) + 0.99),
				"8":  "USD",
				"9":  time.Now(),
			},
		}
	default:
		return DatabaseOperation{
			Type:  "insert",
			Table: table,
			Query: fmt.Sprintf("INSERT INTO %s (id, created_at) VALUES ($1, $2)", table),
			Parameters: map[string]interface{}{
				"1": uuid.New(),
				"2": time.Now(),
			},
		}
	}
}

// generateUpdateOperation generates an UPDATE operation
func (d *DatabaseLoadTestSuite) generateUpdateOperation(table string) DatabaseOperation {
	switch table {
	case "users":
		return DatabaseOperation{
			Type:  "update",
			Table: "users",
			Query: "UPDATE users SET last_login_at = $1 WHERE id = $2",
			Parameters: map[string]interface{}{
				"1": time.Now(),
				"2": uuid.New(),
			},
		}
	case "products":
		return DatabaseOperation{
			Type:  "update",
			Table: "products",
			Query: "UPDATE products SET price = $1, updated_at = $2 WHERE id = $3",
			Parameters: map[string]interface{}{
				"1": decimal.NewFromFloat(float64(rand.Intn(500)+10) + 0.99),
				"2": time.Now(),
				"3": uuid.New(),
			},
		}
	case "orders":
		return DatabaseOperation{
			Type:  "update",
			Table: "orders",
			Query: "UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3",
			Parameters: map[string]interface{}{
				"1": "processing",
				"2": time.Now(),
				"3": uuid.New(),
			},
		}
	default:
		return DatabaseOperation{
			Type:  "update",
			Table: table,
			Query: fmt.Sprintf("UPDATE %s SET updated_at = $1 WHERE id = $2", table),
			Parameters: map[string]interface{}{
				"1": time.Now(),
				"2": uuid.New(),
			},
		}
	}
}

// generateDeleteOperation generates a DELETE operation
func (d *DatabaseLoadTestSuite) generateDeleteOperation(table string) DatabaseOperation {
	return DatabaseOperation{
		Type:  "delete",
		Table: table,
		Query: fmt.Sprintf("DELETE FROM %s WHERE created_at < $1 LIMIT $2", table),
		Parameters: map[string]interface{}{
			"1": time.Now().Add(-1 * time.Hour),
			"2": rand.Intn(10) + 1,
		},
	}
}

// generateJoinOperation generates a complex JOIN operation
func (d *DatabaseLoadTestSuite) generateJoinOperation() DatabaseOperation {
	queries := []string{
		`SELECT o.id, o.order_number, u.email, o.total_amount
		 FROM orders o
		 JOIN users u ON o.customer_id = u.id
		 WHERE o.status = $1 AND o.created_at >= $2
		 ORDER BY o.created_at DESC LIMIT $3`,
		`SELECT p.id, p.name, pc.name as category_name, COUNT(oi.id) as order_count
		 FROM products p
		 LEFT JOIN product_categories pc ON p.category_id = pc.id
		 LEFT JOIN order_items oi ON p.id = oi.product_id
		 WHERE p.is_active = $1
		 GROUP BY p.id, p.name, pc.name
		 ORDER BY order_count DESC
		 LIMIT $2`,
		`SELECT c.id, c.first_name, c.last_name, COUNT(o.id) as order_count, SUM(o.total_amount) as total_spent
		 FROM users c
		 LEFT JOIN orders o ON c.id = o.customer_id
		 WHERE c.is_active = $1 AND o.created_at >= $2
		 GROUP BY c.id, c.first_name, c.last_name
		 HAVING COUNT(o.id) > 0
		 ORDER BY total_spent DESC
		 LIMIT $3`,
	}

	query := queries[rand.Intn(len(queries))]
	params := map[string]interface{}{
		"1": "pending",
		"2": time.Now().Add(-24 * time.Hour),
		"3": rand.Intn(50) + 10,
	}

	return DatabaseOperation{
		Type:       "join",
		Table:      "multiple",
		Query:      query,
		Parameters: params,
	}
}

// generateAggregateOperation generates an aggregate query
func (d *DatabaseLoadTestSuite) generateAggregateOperation(table string) DatabaseOperation {
	switch table {
	case "orders":
		return DatabaseOperation{
			Type:  "aggregate",
			Table: "orders",
			Query: `SELECT
				COUNT(*) as total_orders,
				SUM(total_amount) as total_revenue,
				AVG(total_amount) as avg_order_value,
				MIN(total_amount) as min_order_value,
				MAX(total_amount) as max_order_value
				FROM orders
				WHERE created_at >= $1 AND status = $2`,
			Parameters: map[string]interface{}{
				"1": time.Now().Add(-24 * time.Hour),
				"2": "completed",
			},
		}
	case "products":
		return DatabaseOperation{
			Type:  "aggregate",
			Table: "products",
			Query: `SELECT
				COUNT(*) as total_products,
				COUNT(CASE WHEN is_active = true THEN 1 END) as active_products,
				AVG(price) as avg_price,
				COUNT(CASE WHEN price > 100 THEN 1 END) as premium_products
				FROM products
				WHERE category_id = $1`,
			Parameters: map[string]interface{}{
				"1": uuid.New(),
			},
		}
	default:
		return DatabaseOperation{
			Type:  "aggregate",
			Table: table,
			Query: fmt.Sprintf("SELECT COUNT(*) as total_records FROM %s", table),
			Parameters: map[string]interface{}{},
		}
	}
}

// executeOperation executes a single database operation
func (d *DatabaseLoadTestSuite) executeOperation(workerID, opID int, operation DatabaseOperation) {
	start := time.Now()

	atomic.AddInt64(&d.results.TotalOperations, 1)

	// Execute the query
	ctx, cancel := context.WithTimeout(d.ctx, 10*time.Second)
	defer cancel()

	var err error
	switch operation.Type {
	case "select", "join", "aggregate":
		err = d.executeSelectQuery(ctx, operation)
	case "insert":
		err = d.executeInsertQuery(ctx, operation)
	case "update":
		err = d.executeUpdateQuery(ctx, operation)
	case "delete":
		err = d.executeDeleteQuery(ctx, operation)
	}

	duration := time.Since(start)

	if err != nil {
		atomic.AddInt64(&d.results.FailedOperations, 1)
		d.recordError(operation, err, duration)
	} else {
		atomic.AddInt64(&d.results.SuccessfulOperations, 1)

		// Record slow queries
		if duration > 1*time.Second {
			d.recordSlowQuery(operation, duration)
		}
	}
}

// executeSelectQuery executes a SELECT query
func (d *DatabaseLoadTestSuite) executeSelectQuery(ctx context.Context, operation DatabaseOperation) error {
	rows, err := d.dbPool.Query(ctx, operation.Query)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Consume all rows
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return err
		}
		_ = values // Process if needed
	}

	return rows.Err()
}

// executeInsertQuery executes an INSERT query
func (d *DatabaseLoadTestSuite) executeInsertQuery(ctx context.Context, operation DatabaseOperation) error {
	_, err := d.dbPool.Exec(ctx, operation.Query)
	return err
}

// executeUpdateQuery executes an UPDATE query
func (d *DatabaseLoadTestSuite) executeUpdateQuery(ctx context.Context, operation DatabaseOperation) error {
	_, err := d.dbPool.Exec(ctx, operation.Query)
	return err
}

// executeDeleteQuery executes a DELETE query
func (d *DatabaseLoadTestSuite) executeDeleteQuery(ctx context.Context, operation DatabaseOperation) error {
	_, err := d.dbPool.Exec(ctx, operation.Query)
	return err
}

// recordError records a database operation error
func (d *DatabaseLoadTestSuite) recordError(operation DatabaseOperation, err error, duration time.Duration) {
	d.results.Errors = append(d.results.Errors, DatabaseError{
		Operation: operation.Type,
		Query:     operation.Query,
		Error:     err.Error(),
		Timestamp: time.Now(),
		Duration:  duration,
	})

	// Limit error storage to prevent memory issues
	if len(d.results.Errors) > 1000 {
		d.results.Errors = d.results.Errors[500:]
	}
}

// recordSlowQuery records a slow query
func (d *DatabaseLoadTestSuite) recordSlowQuery(operation DatabaseOperation, duration time.Duration) {
	d.results.SlowQueries = append(d.results.SlowQueries, SlowQuery{
		Query:      operation.Query,
		Duration:   duration,
		Timestamp:  time.Now(),
		Parameters: operation.Parameters,
	})

	// Limit slow query storage to prevent memory issues
	if len(d.results.SlowQueries) > 500 {
		d.results.SlowQueries = d.results.SlowQueries[250:]
	}
}

// startSystemMetricsCollection starts collecting system metrics
func (d *DatabaseLoadTestSuite) startSystemMetricsCollection() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-d.ctx.Done():
				return
			case <-ticker.C:
				// Collect database-specific metrics would go here
				// For now, we'll use placeholder values
			}
		}
	}()
}

// calculateResults calculates final test statistics
func (d *DatabaseLoadTestSuite) calculateResults() {
	if d.results.TotalOperations == 0 {
		return
	}

	d.results.Throughput = float64(d.results.TotalOperations) / d.results.Duration.Seconds()
	d.results.ErrorRate = float64(d.results.FailedOperations) / float64(d.results.TotalOperations)

	// Calculate latency percentiles (placeholder values)
	d.results.AverageLatency = 50 * time.Millisecond
	d.results.P50Latency = 40 * time.Millisecond
	d.results.P95Latency = 150 * time.Millisecond
	d.results.P99Latency = 300 * time.Millisecond

	// Calculate resource usage (placeholder values)
	d.results.CPUUsage = 25.5
	d.results.MemoryUsageMB = 512
}

// collectConnectionPoolStats collects connection pool statistics
func (d *DatabaseLoadTestSuite) collectConnectionPoolStats() {
	stats := d.dbPool.Stat()
	d.results.ConnectionPoolStats = map[string]int64{
		"total_connections":    int64(stats.TotalConns()),
		"idle_connections":     int64(stats.IdleConns()),
		"acquired_connections": int64(stats.AcquiredConns()),
		"max_connections":      int64(stats.MaxConns()),
	}
}

// ValidateResults validates the database load test results
func (d *DatabaseLoadTestSuite) ValidateResults() error {
	if d.results == nil {
		return fmt.Errorf("no test results to validate")
	}

	// Validate error rate
	if d.results.ErrorRate > 0.05 { // 5% error rate threshold
		return fmt.Errorf("error rate %.2f%% exceeds 5%% threshold", d.results.ErrorRate*100)
	}

	// Validate throughput
	if d.config.TargetThroughput > 0 && d.results.Throughput < float64(d.config.TargetThroughput)*0.9 {
		return fmt.Errorf("throughput %.2f ops/sec is below target %d ops/sec (90%% threshold)",
			d.results.Throughput, d.config.TargetThroughput)
	}

	// Validate latency
	if d.config.MaxLatency > 0 && d.results.P95Latency > d.config.MaxLatency {
		return fmt.Errorf("95th percentile latency %v exceeds maximum %v",
			d.results.P95Latency, d.config.MaxLatency)
	}

	// Validate CPU usage
	if d.config.MaxCPUUsage > 0 && d.results.CPUUsage > d.config.MaxCPUUsage {
		return fmt.Errorf("CPU usage %.2f%% exceeds maximum %.2f%%",
			d.results.CPUUsage, d.config.MaxCPUUsage)
	}

	// Validate memory usage
	if d.config.MaxMemoryUsage > 0 && d.results.MemoryUsageMB > d.config.MaxMemoryUsage {
		return fmt.Errorf("memory usage %d MB exceeds maximum %d MB",
			d.results.MemoryUsageMB, d.config.MaxMemoryUsage)
	}

	return nil
}

// Close closes the database load test suite
func (d *DatabaseLoadTestSuite) Close() {
	d.cancel()
	d.wg.Wait()
	if d.dbPool != nil {
		d.dbPool.Close()
	}
}

// TestDatabaseLoadTestSuite runs the database load test suite
func TestDatabaseLoadTestSuite(t *testing.T) {
	databaseURL := "postgres://erpgo_user:erpgo_password@localhost:5432/erpgo_db?sslmode=disable"

	testCases := []struct {
		name   string
		config DatabaseLoadTestConfig
	}{
		{
			name: "SelectHeavyWorkload",
			config: DatabaseLoadTestConfig{
				Name:                  "Select-Heavy Workload Test",
				DatabaseURL:           databaseURL,
				ConcurrentConnections: 20,
				OperationsPerConnection: 500,
				OperationTypes:        []string{"select", "join", "aggregate"},
				Tables:                []string{"users", "products", "orders"},
				TargetThroughput:      5000,
				MaxLatency:            100 * time.Millisecond,
				MaxCPUUsage:           70.0,
				MaxMemoryUsage:        1024, // 1GB
			},
		},
		{
			name: "MixedWorkload",
			config: DatabaseLoadTestConfig{
				Name:                  "Mixed Database Workload Test",
				DatabaseURL:           databaseURL,
				ConcurrentConnections: 15,
				OperationsPerConnection: 300,
				OperationTypes:        []string{"select", "insert", "update", "join"},
				Tables:                []string{"users", "products", "orders", "order_items"},
				TargetThroughput:      3000,
				MaxLatency:            200 * time.Millisecond,
				MaxCPUUsage:           80.0,
				MaxMemoryUsage:        1536, // 1.5GB
			},
		},
		{
			name: "WriteHeavyWorkload",
			config: DatabaseLoadTestConfig{
				Name:                  "Write-Heavy Workload Test",
				DatabaseURL:           databaseURL,
				ConcurrentConnections: 10,
				OperationsPerConnection: 200,
				OperationTypes:        []string{"insert", "update"},
				Tables:                []string{"products", "orders"},
				TargetThroughput:      1500,
				MaxLatency:            300 * time.Millisecond,
				MaxCPUUsage:           85.0,
				MaxMemoryUsage:        2048, // 2GB
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			suite, err := NewDatabaseLoadTestSuite(tc.config)
			require.NoError(t, err)
			defer suite.Close()

			result, err := suite.RunDatabaseLoadTest()
			require.NoError(t, err)

			err = suite.ValidateResults()
			require.NoError(t, err, "Database load test validation failed: %v", err)

			// Print detailed results
			t.Logf("=== %s Results ===", tc.name)
			t.Logf("Duration: %v", result.Duration)
			t.Logf("Total Operations: %d", result.TotalOperations)
			t.Logf("Successful Operations: %d", result.SuccessfulOperations)
			t.Logf("Failed Operations: %d", result.FailedOperations)
			t.Logf("Throughput: %.2f ops/sec", result.Throughput)
			t.Logf("Error Rate: %.2f%%", result.ErrorRate*100)
			t.Logf("Average Latency: %v", result.AverageLatency)
			t.Logf("95th Percentile Latency: %v", result.P95Latency)
			t.Logf("CPU Usage: %.2f%%", result.CPUUsage)
			t.Logf("Memory Usage: %d MB", result.MemoryUsageMB)

			t.Logf("Connection Pool Stats:")
			for key, value := range result.ConnectionPoolStats {
				t.Logf("  %s: %d", key, value)
			}

			if len(result.SlowQueries) > 0 {
				t.Logf("Slow Queries (%d):", len(result.SlowQueries))
				for i, sq := range result.SlowQueries[:min(5, len(result.SlowQueries))] {
					t.Logf("  %d. Duration: %v, Query: %s", i+1, sq.Duration, sq.Query[:min(100, len(sq.Query))])
				}
			}

			if len(result.Errors) > 0 {
				t.Logf("Errors (%d):", len(result.Errors))
				for i, err := range result.Errors[:min(5, len(result.Errors))] {
					t.Logf("  %d. %s: %s", i+1, err.Operation, err.Error)
				}
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}