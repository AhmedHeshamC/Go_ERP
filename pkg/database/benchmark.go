package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// BenchmarkConfig holds configuration for performance benchmarks
type BenchmarkConfig struct {
	// Test duration
	Duration time.Duration `json:"duration"`

	// Concurrency settings
	ConcurrentReaders  int `json:"concurrent_readers"`
	ConcurrentWriters  int `json:"concurrent_writers"`
	ConcurrentMixed    int `json:"concurrent_mixed"`

	// Query patterns
	SelectQueries      []QueryPattern `json:"select_queries"`
	InsertQueries      []QueryPattern `json:"insert_queries"`
	UpdateQueries      []QueryPattern `json:"update_queries"`
	DeleteQueries      []QueryPattern `json:"delete_queries"`

	// Data size settings
	InitialRecords     int `json:"initial_records"`
	BatchSize          int `json:"batch_size"`

	// Performance targets
	TargetResponseTime time.Duration `json:"target_response_time"`
	TargetThroughput   int64         `json:"target_throughput"`
}

// QueryPattern represents a test query pattern
type QueryPattern struct {
	Name        string `json:"name"`
	Query       string `json:"query"`
	Args        []interface{} `json:"args"`
	Weight      int    `json:"weight"` // Weight for selection in random tests
}

// BenchmarkResult holds the results of a performance benchmark
type BenchmarkResult struct {
	TestName          string        `json:"test_name"`
	Duration          time.Duration `json:"duration"`
	TotalQueries      int64         `json:"total_queries"`
	SuccessfulQueries int64         `json:"successful_queries"`
	FailedQueries     int64         `json:"failed_queries"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	MinResponseTime   time.Duration `json:"min_response_time"`
	MaxResponseTime   time.Duration `json:"max_response_time"`
	P95ResponseTime   time.Duration `json:"p95_response_time"`
	P99ResponseTime   time.Duration `json:"p99_response_time"`
	Throughput        float64       `json:"throughput_qps"`
	Errors            []string      `json:"errors"`
	CPUUsage          float64       `json:"cpu_usage"`
	MemoryUsage       int64         `json:"memory_usage"`
	ConnectionStats   map[string]interface{} `json:"connection_stats"`
}

// DatabaseBenchmark performs database performance testing
type DatabaseBenchmark struct {
	db     *Database
	config *BenchmarkConfig
	logger *zerolog.Logger
}

// NewDatabaseBenchmark creates a new database benchmark instance
func NewDatabaseBenchmark(db *Database, config *BenchmarkConfig, logger *zerolog.Logger) *DatabaseBenchmark {
	if logger == nil {
		l := zerolog.Nop()
		logger = &l
	}

	// Set defaults if not provided
	if config.Duration <= 0 {
		config.Duration = 5 * time.Minute
	}
	if config.ConcurrentReaders <= 0 {
		config.ConcurrentReaders = 10
	}
	if config.ConcurrentWriters <= 0 {
		config.ConcurrentWriters = 5
	}
	if config.ConcurrentMixed <= 0 {
		config.ConcurrentMixed = 15
	}
	if config.TargetResponseTime <= 0 {
		config.TargetResponseTime = 100 * time.Millisecond
	}
	if config.TargetThroughput <= 0 {
		config.TargetThroughput = 1000
	}

	return &DatabaseBenchmark{
		db:     db,
		config: config,
		logger: logger,
	}
}

// RunAllBenchmarks executes all benchmark tests
func (dbb *DatabaseBenchmark) RunAllBenchmarks(ctx context.Context) map[string]*BenchmarkResult {
	results := make(map[string]*BenchmarkResult)

	dbb.logger.Info().
		Dur("duration", dbb.config.Duration).
		Int("readers", dbb.config.ConcurrentReaders).
		Int("writers", dbb.config.ConcurrentWriters).
		Msg("Starting database benchmarks")

	// Warm up database
	dbb.warmUp(ctx)

	// Run different benchmark types
	if len(dbb.config.SelectQueries) > 0 {
		results["read_benchmark"] = dbb.runReadBenchmark(ctx)
	}

	if len(dbb.config.InsertQueries) > 0 {
		results["write_benchmark"] = dbb.runWriteBenchmark(ctx)
	}

	if len(dbb.config.UpdateQueries) > 0 {
		results["update_benchmark"] = dbb.runUpdateBenchmark(ctx)
	}

	results["mixed_benchmark"] = dbb.runMixedBenchmark(ctx)
	results["connection_stress"] = dbb.runConnectionStressBenchmark(ctx)
	results["high_frequency_small"] = dbb.runHighFrequencySmallBenchmark(ctx)

	return results
}

// warmUp performs warm-up queries to prepare the database
func (dbb *DatabaseBenchmark) warmUp(ctx context.Context) {
	dbb.logger.Info().Msg("Warming up database...")

	warmQueries := []string{
		"SELECT COUNT(*) FROM users",
		"SELECT COUNT(*) FROM products",
		"SELECT COUNT(*) FROM orders",
		"SELECT 1", // Simple connection test
	}

	for _, query := range warmQueries {
		start := time.Now()
		_, err := dbb.db.Exec(ctx, query)
		if err != nil {
			dbb.logger.Warn().
				Err(err).
				Str("query", query).
				Msg("Warm-up query failed")
		} else {
			dbb.logger.Debug().
				Str("query", query).
				Dur("duration", time.Since(start)).
				Msg("Warm-up query completed")
		}
	}

	time.Sleep(1 * time.Second) // Allow for connection stabilization
}

// runReadBenchmark performs read-only performance testing
func (dbb *DatabaseBenchmark) runReadBenchmark(ctx context.Context) *BenchmarkResult {
	result := &BenchmarkResult{
		TestName:        "read_benchmark",
		MinResponseTime: time.Hour, // Initialize with high value
		Errors:          make([]string, 0),
		ConnectionStats: make(map[string]interface{}),
	}

	var wg sync.WaitGroup
	var mutex sync.Mutex
	responseTimes := make([]time.Duration, 0)

	startTime := time.Now()
	endTime := startTime.Add(dbb.config.Duration)

	// Start concurrent readers
	for i := 0; i < dbb.config.ConcurrentReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			dbb.runReaderWorker(ctx, &wg, &mutex, result, &responseTimes, endTime, readerID)
		}(i)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	// Calculate statistics
	dbb.calculateStatistics(result, responseTimes)

	return result
}

// runReaderWorker executes read queries for a single worker
func (dbb *DatabaseBenchmark) runReaderWorker(ctx context.Context, wg *sync.WaitGroup, mutex *sync.Mutex, result *BenchmarkResult, responseTimes *[]time.Duration, endTime time.Time, readerID int) {
	for time.Now().Before(endTime) {
		// Select random query pattern
		pattern := dbb.selectRandomQuery(dbb.config.SelectQueries)

		queryStart := time.Now()
		_, err := dbb.db.Query(ctx, pattern.Query, pattern.Args...)
		responseTime := time.Since(queryStart)

		mutex.Lock()
		*responseTimes = append(*responseTimes, responseTime)
		result.TotalQueries++

		if err != nil {
			result.FailedQueries++
			result.Errors = append(result.Errors, fmt.Sprintf("Reader %d: %v", readerID, err))
		} else {
			result.SuccessfulQueries++
			if responseTime < result.MinResponseTime {
				result.MinResponseTime = responseTime
			}
			if responseTime > result.MaxResponseTime {
				result.MaxResponseTime = responseTime
			}
		}
		mutex.Unlock()

		// Small delay to prevent overwhelming the database
		time.Sleep(10 * time.Millisecond)
	}
}

// runWriteBenchmark performs write-only performance testing
func (dbb *DatabaseBenchmark) runWriteBenchmark(ctx context.Context) *BenchmarkResult {
	result := &BenchmarkResult{
		TestName:        "write_benchmark",
		MinResponseTime: time.Hour,
		Errors:          make([]string, 0),
		ConnectionStats: make(map[string]interface{}),
	}

	var wg sync.WaitGroup
	var mutex sync.Mutex
	responseTimes := make([]time.Duration, 0)

	startTime := time.Now()
	endTime := startTime.Add(dbb.config.Duration)

	// Start concurrent writers
	for i := 0; i < dbb.config.ConcurrentWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			dbb.runWriterWorker(ctx, &wg, &mutex, result, &responseTimes, endTime, writerID)
		}(i)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	// Calculate statistics
	dbb.calculateStatistics(result, responseTimes)

	return result
}

// runWriterWorker executes write queries for a single worker
func (dbb *DatabaseBenchmark) runWriterWorker(ctx context.Context, wg *sync.WaitGroup, mutex *sync.Mutex, result *BenchmarkResult, responseTimes *[]time.Duration, endTime time.Time, writerID int) {
	for time.Now().Before(endTime) {
		// Create test data
		pattern := dbb.generateInsertPattern()

		queryStart := time.Now()
		_, err := dbb.db.Exec(ctx, pattern.Query, pattern.Args...)
		responseTime := time.Since(queryStart)

		mutex.Lock()
		*responseTimes = append(*responseTimes, responseTime)
		result.TotalQueries++

		if err != nil {
			result.FailedQueries++
			result.Errors = append(result.Errors, fmt.Sprintf("Writer %d: %v", writerID, err))
		} else {
			result.SuccessfulQueries++
			if responseTime < result.MinResponseTime {
				result.MinResponseTime = responseTime
			}
			if responseTime > result.MaxResponseTime {
				result.MaxResponseTime = responseTime
			}
		}
		mutex.Unlock()

		// Delay for write operations
		time.Sleep(50 * time.Millisecond)
	}
}

// runUpdateBenchmark performs update performance testing
func (dbb *DatabaseBenchmark) runUpdateBenchmark(ctx context.Context) *BenchmarkResult {
	result := &BenchmarkResult{
		TestName:        "update_benchmark",
		MinResponseTime: time.Hour,
		Errors:          make([]string, 0),
		ConnectionStats: make(map[string]interface{}),
	}

	var wg sync.WaitGroup
	var mutex sync.Mutex
	responseTimes := make([]time.Duration, 0)

	startTime := time.Now()
	endTime := startTime.Add(dbb.config.Duration)

	// Start concurrent updaters
	for i := 0; i < dbb.config.ConcurrentWriters; i++ {
		wg.Add(1)
		go func(updaterID int) {
			defer wg.Done()
			dbb.runUpdaterWorker(ctx, &wg, &mutex, result, &responseTimes, endTime, updaterID)
		}(i)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	// Calculate statistics
	dbb.calculateStatistics(result, responseTimes)

	return result
}

// runUpdaterWorker executes update queries for a single worker
func (dbb *DatabaseBenchmark) runUpdaterWorker(ctx context.Context, wg *sync.WaitGroup, mutex *sync.Mutex, result *BenchmarkResult, responseTimes *[]time.Duration, endTime time.Time, updaterID int) {
	for time.Now().Before(endTime) {
		// Create update query
		pattern := dbb.generateUpdatePattern()

		queryStart := time.Now()
		_, err := dbb.db.Exec(ctx, pattern.Query, pattern.Args...)
		responseTime := time.Since(queryStart)

		mutex.Lock()
		*responseTimes = append(*responseTimes, responseTime)
		result.TotalQueries++

		if err != nil {
			result.FailedQueries++
			result.Errors = append(result.Errors, fmt.Sprintf("Updater %d: %v", updaterID, err))
		} else {
			result.SuccessfulQueries++
			if responseTime < result.MinResponseTime {
				result.MinResponseTime = responseTime
			}
			if responseTime > result.MaxResponseTime {
				result.MaxResponseTime = responseTime
			}
		}
		mutex.Unlock()

		time.Sleep(30 * time.Millisecond)
	}
}

// runMixedBenchmark performs mixed read/write performance testing
func (dbb *DatabaseBenchmark) runMixedBenchmark(ctx context.Context) *BenchmarkResult {
	result := &BenchmarkResult{
		TestName:        "mixed_benchmark",
		MinResponseTime: time.Hour,
		Errors:          make([]string, 0),
		ConnectionStats: make(map[string]interface{}),
	}

	var wg sync.WaitGroup
	var mutex sync.Mutex
	responseTimes := make([]time.Duration, 0)

	startTime := time.Now()
	endTime := startTime.Add(dbb.config.Duration)

	// Start concurrent mixed workers
	for i := 0; i < dbb.config.ConcurrentMixed; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			dbb.runMixedWorker(ctx, &wg, &mutex, result, &responseTimes, endTime, workerID)
		}(i)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	// Calculate statistics
	dbb.calculateStatistics(result, responseTimes)

	return result
}

// runMixedWorker executes mixed queries for a single worker
func (dbb *DatabaseBenchmark) runMixedWorker(ctx context.Context, wg *sync.WaitGroup, mutex *sync.Mutex, result *BenchmarkResult, responseTimes *[]time.Duration, endTime time.Time, workerID int) {
	for time.Now().Before(endTime) {
		var pattern QueryPattern

		// 70% reads, 20% inserts, 10% updates
		rand := uuid.New().ID() % 10
		if rand < 7 {
			pattern = dbb.selectRandomQuery(dbb.config.SelectQueries)
		} else if rand < 9 {
			pattern = dbb.generateInsertPattern()
		} else {
			pattern = dbb.generateUpdatePattern()
		}

		queryStart := time.Now()
		_, err := dbb.db.Query(ctx, pattern.Query, pattern.Args...)
		responseTime := time.Since(queryStart)

		mutex.Lock()
		*responseTimes = append(*responseTimes, responseTime)
		result.TotalQueries++

		if err != nil {
			result.FailedQueries++
			result.Errors = append(result.Errors, fmt.Sprintf("Mixed worker %d: %v", workerID, err))
		} else {
			result.SuccessfulQueries++
			if responseTime < result.MinResponseTime {
				result.MinResponseTime = responseTime
			}
			if responseTime > result.MaxResponseTime {
				result.MaxResponseTime = responseTime
			}
		}
		mutex.Unlock()

		time.Sleep(20 * time.Millisecond)
	}
}

// runConnectionStressBenchmark tests connection pool performance
func (dbb *DatabaseBenchmark) runConnectionStressBenchmark(ctx context.Context) *BenchmarkResult {
	result := &BenchmarkResult{
		TestName:        "connection_stress",
		MinResponseTime: time.Hour,
		Errors:          make([]string, 0),
		ConnectionStats: make(map[string]interface{}),
	}

	var wg sync.WaitGroup
	var mutex sync.Mutex
	responseTimes := make([]time.Duration, 0)

	startTime := time.Now()
	endTime := startTime.Add(dbb.config.Duration / 2) // Shorter test for connection stress

	// High concurrency for connection stress
	concurrency := dbb.config.ConcurrentReaders * 3
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			dbb.runConnectionStressWorker(ctx, &wg, &mutex, result, &responseTimes, endTime, workerID)
		}(i)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	// Calculate statistics
	dbb.calculateStatistics(result, responseTimes)

	return result
}

// runConnectionStressWorker creates many short-lived connections
func (dbb *DatabaseBenchmark) runConnectionStressWorker(ctx context.Context, wg *sync.WaitGroup, mutex *sync.Mutex, result *BenchmarkResult, responseTimes *[]time.Duration, endTime time.Time, workerID int) {
	for time.Now().Before(endTime) {
		// Acquire and release connections rapidly
		queryStart := time.Now()
		conn, err := dbb.db.Acquire(ctx)
		if err != nil {
			mutex.Lock()
			result.TotalQueries++
			result.FailedQueries++
			result.Errors = append(result.Errors, fmt.Sprintf("Connection stress %d: %v", workerID, err))
			mutex.Unlock()
			continue
		}

		// Simple query
		_, err = conn.Query(ctx, "SELECT 1")
		responseTime := time.Since(queryStart)
		conn.Release()

		mutex.Lock()
		*responseTimes = append(*responseTimes, responseTime)
		result.TotalQueries++

		if err != nil {
			result.FailedQueries++
			result.Errors = append(result.Errors, fmt.Sprintf("Connection stress %d: %v", workerID, err))
		} else {
			result.SuccessfulQueries++
			if responseTime < result.MinResponseTime {
				result.MinResponseTime = responseTime
			}
			if responseTime > result.MaxResponseTime {
				result.MaxResponseTime = responseTime
			}
		}
		mutex.Unlock()

		time.Sleep(5 * time.Millisecond)
	}
}

// runHighFrequencySmallBenchmark tests many small, fast queries
func (dbb *DatabaseBenchmark) runHighFrequencySmallBenchmark(ctx context.Context) *BenchmarkResult {
	result := &BenchmarkResult{
		TestName:        "high_frequency_small",
		MinResponseTime: time.Hour,
		Errors:          make([]string, 0),
		ConnectionStats: make(map[string]interface{}),
	}

	var wg sync.WaitGroup
	var mutex sync.Mutex
	responseTimes := make([]time.Duration, 0)

	startTime := time.Now()
	endTime := startTime.Add(dbb.config.Duration / 2)

	// High frequency workers
	concurrency := dbb.config.ConcurrentReaders * 2
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			dbb.runHighFrequencyWorker(ctx, &wg, &mutex, result, &responseTimes, endTime, workerID)
		}(i)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	// Calculate statistics
	dbb.calculateStatistics(result, responseTimes)

	return result
}

// runHighFrequencyWorker executes many small queries
func (dbb *DatabaseBenchmark) runHighFrequencyWorker(ctx context.Context, wg *sync.WaitGroup, mutex *sync.Mutex, result *BenchmarkResult, responseTimes *[]time.Duration, endTime time.Time, workerID int) {
	query := "SELECT COUNT(*) FROM users WHERE is_active = true LIMIT 1"

	for time.Now().Before(endTime) {
		queryStart := time.Now()
		_, err := dbb.db.Query(ctx, query)
		responseTime := time.Since(queryStart)

		mutex.Lock()
		*responseTimes = append(*responseTimes, responseTime)
		result.TotalQueries++

		if err != nil {
			result.FailedQueries++
			result.Errors = append(result.Errors, fmt.Sprintf("High freq worker %d: %v", workerID, err))
		} else {
			result.SuccessfulQueries++
			if responseTime < result.MinResponseTime {
				result.MinResponseTime = responseTime
			}
			if responseTime > result.MaxResponseTime {
				result.MaxResponseTime = responseTime
			}
		}
		mutex.Unlock()

		// Minimal delay for high frequency
		time.Sleep(1 * time.Millisecond)
	}
}

// Helper methods

func (dbb *DatabaseBenchmark) selectRandomQuery(patterns []QueryPattern) QueryPattern {
	if len(patterns) == 0 {
		return QueryPattern{Query: "SELECT 1"}
	}

	// Simple random selection based on weights
	totalWeight := 0
	for _, pattern := range patterns {
		totalWeight += pattern.Weight
	}

	if totalWeight == 0 {
		return patterns[0]
	}

	rand := int(uuid.New().ID()) % int(totalWeight)
	currentWeight := 0

	for _, pattern := range patterns {
		currentWeight += pattern.Weight
		if rand < currentWeight {
			return pattern
		}
	}

	return patterns[0]
}

func (dbb *DatabaseBenchmark) generateInsertPattern() QueryPattern {
	return QueryPattern{
		Query: "INSERT INTO users (id, email, username, password_hash, first_name, last_name, is_active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		Args: []interface{}{
			uuid.New(),
			fmt.Sprintf("test%d@example.com", uuid.New().ID()),
			fmt.Sprintf("user%d", uuid.New().ID()),
			"hashed_password",
			"Test",
			"User",
			true,
			time.Now(),
			time.Now(),
		},
	}
}

func (dbb *DatabaseBenchmark) generateUpdatePattern() QueryPattern {
	return QueryPattern{
		Query: "UPDATE users SET last_login_at = $1 WHERE is_active = true ORDER BY created_at DESC LIMIT 1",
		Args:  []interface{}{time.Now()},
	}
}

func (dbb *DatabaseBenchmark) calculateStatistics(result *BenchmarkResult, responseTimes []time.Duration) {
	if len(responseTimes) == 0 {
		return
	}

	// Calculate average response time
	var totalTime time.Duration
	for _, rt := range responseTimes {
		totalTime += rt
	}
	result.AvgResponseTime = totalTime / time.Duration(len(responseTimes))

	// Calculate percentiles
	sorted := make([]time.Duration, len(responseTimes))
	copy(sorted, responseTimes)

	// Simple bubble sort for percentiles (small datasets)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	p95Index := int(float64(len(sorted)) * 0.95)
	if p95Index >= len(sorted) {
		p95Index = len(sorted) - 1
	}
	result.P95ResponseTime = sorted[p95Index]

	p99Index := int(float64(len(sorted)) * 0.99)
	if p99Index >= len(sorted) {
		p99Index = len(sorted) - 1
	}
	result.P99ResponseTime = sorted[p99Index]

	// Calculate throughput
	if result.Duration.Seconds() > 0 {
		result.Throughput = float64(result.SuccessfulQueries) / result.Duration.Seconds()
	}

	// Collect connection statistics
	stats := dbb.db.Stats()
	result.ConnectionStats["acquired_conns"] = stats.AcquiredConns()
	result.ConnectionStats["idle_conns"] = stats.IdleConns()
	result.ConnectionStats["total_conns"] = stats.TotalConns()
	result.ConnectionStats["max_conns"] = stats.MaxConns()
}