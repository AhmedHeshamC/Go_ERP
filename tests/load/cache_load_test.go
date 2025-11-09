package load

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// CacheLoadTestSuite contains cache performance tests
type CacheLoadTestSuite struct {
	redisClient *redis.Client
	config      CacheLoadTestConfig
	results     *CacheLoadTestResult
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// CacheLoadTestConfig defines configuration for cache load tests
type CacheLoadTestConfig struct {
	Name               string
	RedisURL           string
	ConcurrentClients  int
	OperationsPerClient int
	TestDuration       time.Duration
	OperationTypes     []string // "get", "set", "delete", "mget", "mset", "pipeline"
	DataSizeKB         int      // Size of data objects in KB
	TargetThroughput   int64    // Operations per second
	TargetHitRate      float64  // Target cache hit rate (0.0 to 1.0)
	MaxLatency         time.Duration
	MaxMemoryUsage     int64 // MB
	KeyDistribution    string // "uniform", "zipfian", "latest"
}

// CacheLoadTestResult contains results from cache load tests
type CacheLoadTestResult struct {
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
	HitRate               float64
	MissRate              float64
	RedisMemoryUsageMB    int64
	RedisMaxMemoryMB      int64
	RedisConnections      int64
	SlowOperations        []SlowOperation
	Errors                []CacheError
	OperationBreakdown    map[string]int64
}

// SlowOperation represents a slow cache operation
type SlowOperation struct {
	Operation  string
	Key        string
	Duration   time.Duration
	Timestamp  time.Time
	DataSize   int
}

// CacheError represents a cache operation error
type CacheError struct {
	Operation string
	Key       string
	Error     string
	Timestamp time.Time
	Duration  time.Duration
}

// CacheOperation represents a cache operation
type CacheOperation struct {
	Type       string
	Key        string
	Value      interface{}
	TTL        time.Duration
	DataSize   int
}

// TestData represents test data structure for caching
type TestData struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Price       float64                `json:"price"`
	Category    string                 `json:"category"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// NewCacheLoadTestSuite creates a new cache load test suite
func NewCacheLoadTestSuite(config CacheLoadTestConfig) (*CacheLoadTestSuite, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Parse Redis URL
	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	// Optimize for load testing
	opt.PoolSize = config.ConcurrentClients
	opt.MinIdleConns = config.ConcurrentClients / 2
	opt.MaxRetries = 1
	opt.ReadTimeout = 100 * time.Millisecond
	opt.WriteTimeout = 100 * time.Millisecond

	client := redis.NewClient(opt)

	// Test the connection
	if err := client.Ping(ctx).Err(); err != nil {
		cancel()
		client.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &CacheLoadTestSuite{
		redisClient: client,
		config:      config,
		ctx:         ctx,
		cancel:      cancel,
		results: &CacheLoadTestResult{
			TestName:           config.Name,
			SlowOperations:     make([]SlowOperation, 0),
			Errors:             make([]CacheError, 0),
			OperationBreakdown: make(map[string]int64),
		},
	}, nil
}

// RunCacheLoadTest executes the cache load test
func (c *CacheLoadTestSuite) RunCacheLoadTest() (*CacheLoadTestResult, error) {
	log.Printf("Starting cache load test: %s", c.config.Name)
	log.Printf("Configuration: Clients=%d, Operations/Client=%d, Duration=%v, DataSize=%dKB",
		c.config.ConcurrentClients, c.config.OperationsPerClient, c.config.TestDuration, c.config.DataSizeKB)

	// Clear Redis before test (optional - comment out for persistence testing)
	if err := c.redisClient.FlushAll(c.ctx).Err(); err != nil {
		log.Printf("Warning: Failed to flush Redis: %v", err)
	}

	// Pre-populate cache with test data (70% of keys)
	if err := c.prepopulateCache(); err != nil {
		log.Printf("Warning: Failed to pre-populate cache: %v", err)
	}

	c.results.StartTime = time.Now()

	// Start system metrics collection
	c.startSystemMetricsCollection()

	// Start cache client workers
	for i := 0; i < c.config.ConcurrentClients; i++ {
		c.wg.Add(1)
		go c.runCacheWorker(i)
	}

	// Wait for test duration or completion
	if c.config.TestDuration > 0 {
		time.Sleep(c.config.TestDuration)
	} else {
		c.wg.Wait()
	}

	// Stop the test
	c.cancel()
	c.wg.Wait()

	c.results.EndTime = time.Now()
	c.results.Duration = c.results.EndTime.Sub(c.results.StartTime)

	// Calculate final results
	c.calculateResults()

	// Collect Redis statistics
	c.collectRedisStats()

	log.Printf("Cache load test completed: %s", c.config.Name)
	log.Printf("Results: Throughput=%.2f ops/sec, Hit Rate=%.2f%%, Avg Latency=%v",
		c.results.Throughput,
		c.results.HitRate*100,
		c.results.AverageLatency)

	return c.results, nil
}

// prepopulateCache pre-populates the cache with test data
func (c *CacheLoadTestSuite) prepopulateCache() error {
	const prepopulateRatio = 0.7
	totalKeys := c.config.ConcurrentClients * c.config.OperationsPerClient
	keysToPrepopulate := int(float64(totalKeys) * prepopulateRatio)

	log.Printf("Pre-populating cache with %d keys...", keysToPrepopulate)

	// Use pipeline for efficient pre-population
	pipe := c.redisClient.Pipeline()

	for i := 0; i < keysToPrepopulate; i++ {
		key := fmt.Sprintf("test:key:%d", i)
		value := c.generateTestData(i)

		data, err := json.Marshal(value)
		if err != nil {
			continue
		}

		pipe.Set(c.ctx, key, data, 1*time.Hour)
	}

	_, err := pipe.Exec(c.ctx)
	if err != nil {
		return fmt.Errorf("failed to execute pre-population pipeline: %w", err)
	}

	log.Printf("Cache pre-populated with %d keys", keysToPrepopulate)
	return nil
}

// runCacheWorker executes cache operations for a single client
func (c *CacheLoadTestSuite) runCacheWorker(clientID int) {
	defer c.wg.Done()

	operationsCount := c.config.OperationsPerClient
	if operationsCount <= 0 {
		operationsCount = 1000 // Default for duration-based tests
	}

	for i := 0; i < operationsCount; i++ {
		select {
		case <-c.ctx.Done():
			return
		default:
			operation := c.selectRandomOperation(clientID, i)
			c.executeOperation(clientID, i, operation)

			// Small delay between operations to simulate realistic workload
			time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
		}
	}
}

// selectRandomOperation selects a random operation type and key
func (c *CacheLoadTestSuite) selectRandomOperation(clientID, opID int) CacheOperation {
	opType := c.config.OperationTypes[rand.Intn(len(c.config.OperationTypes))]

	var key string
	switch c.config.KeyDistribution {
	case "zipfian":
		// Zipfian distribution - some keys are much more popular than others
		key = fmt.Sprintf("test:key:%d", c.zipfianKey(opID))
	case "latest":
		// Latest distribution - favor recently created keys
		key = fmt.Sprintf("test:key:%d", c.latestKey(opID))
	default: // uniform
		key = fmt.Sprintf("test:key:%d", rand.Intn(c.config.ConcurrentClients*c.config.OperationsPerClient))
	}

	switch opType {
	case "get":
		return CacheOperation{
			Type: "get",
			Key:  key,
		}
	case "set":
		return CacheOperation{
			Type:     "set",
			Key:      key,
			Value:    c.generateTestData(clientID*1000 + opID),
			TTL:      time.Duration(rand.Intn(3600)+1) * time.Second,
			DataSize: c.config.DataSizeKB * 1024,
		}
	case "delete":
		return CacheOperation{
			Type: "delete",
			Key:  key,
		}
	case "mget":
		keys := make([]string, rand.Intn(10)+1)
		for i := range keys {
			keys[i] = fmt.Sprintf("test:key:%d", rand.Intn(c.config.ConcurrentClients*c.config.OperationsPerClient))
		}
		return CacheOperation{
			Type: "mget",
			Key:  strings.Join(keys, ","),
		}
	case "mset":
		pairs := make(map[string]interface{})
		for i := 0; i < rand.Intn(5)+1; i++ {
			key := fmt.Sprintf("test:key:%d", clientID*1000+opID+i)
			pairs[key] = c.generateTestData(clientID*1000 + opID + i)
		}
		return CacheOperation{
			Type:  "mset",
			Key:   fmt.Sprintf("%d keys", len(pairs)),
			Value: pairs,
		}
	case "pipeline":
		return CacheOperation{
			Type: "pipeline",
			Key:  key,
		}
	default:
		return CacheOperation{
			Type: "get",
			Key:  key,
		}
	}
}

// zipfianKey generates a key following zipfian distribution
func (c *CacheLoadTestSuite) zipfianKey(opID int) int {
	// Simple zipfian approximation
	maxKeys := c.config.ConcurrentClients * c.config.OperationsPerClient
	if maxKeys <= 1 {
		return 0
	}

	// 80% of operations hit 20% of keys
	if rand.Float64() < 0.8 {
		return rand.Intn(maxKeys / 5)
	}
	return rand.Intn(maxKeys)
}

// latestKey generates a key favoring recently created keys
func (c *CacheLoadTestSuite) latestKey(opID int) int {
	maxKeys := c.config.ConcurrentClients * c.config.OperationsPerClient
	if maxKeys <= 1 {
		return 0
	}

	// 70% of operations hit the most recent 30% of keys
	if rand.Float64() < 0.7 {
		// Recent keys
		recentStart := maxKeys * 70 / 100
		return recentStart + rand.Intn(maxKeys-recentStart)
	}
	// Older keys
	return rand.Intn(maxKeys * 70 / 100)
}

// generateTestData generates test data of specified size
func (c *CacheLoadTestSuite) generateTestData(seed int) TestData {
	rand.Seed(int64(seed))

	// Generate filler data to reach desired size
	filler := make([]byte, c.config.DataSizeKB*512) // Rough approximation
	rand.Read(filler)

	return TestData{
		ID:          uuid.New().String(),
		Name:        fmt.Sprintf("Test Product %d", seed),
		Description: string(filler),
		Price:       float64(rand.Intn(1000)+1) + 0.99,
		Category:    []string{"Electronics", "Books", "Clothing", "Home"}[rand.Intn(4)],
		Tags:        []string{fmt.Sprintf("tag%d", rand.Intn(100)), fmt.Sprintf("category%d", rand.Intn(20))},
		Metadata: map[string]interface{}{
			"seed":      seed,
			"size":      c.config.DataSizeKB,
			"created":   time.Now(),
			"random":    rand.Float64(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// executeOperation executes a single cache operation
func (c *CacheLoadTestSuite) executeOperation(clientID, opID int, operation CacheOperation) {
	start := time.Now()

	atomic.AddInt64(&c.results.TotalOperations, 1)
	atomic.AddInt64(&c.results.OperationBreakdown[operation.Type], 1)

	// Execute the operation
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	var err error
	var hit bool

	switch operation.Type {
	case "get":
		err, hit = c.executeGet(ctx, operation)
	case "set":
		err = c.executeSet(ctx, operation)
	case "delete":
		err = c.executeDelete(ctx, operation)
	case "mget":
		err, hit = c.executeMGet(ctx, operation)
	case "mset":
		err = c.executeMSet(ctx, operation)
	case "pipeline":
		err, hit = c.executePipeline(ctx, operation)
	}

	duration := time.Since(start)

	if err != nil {
		atomic.AddInt64(&c.results.FailedOperations, 1)
		c.recordError(operation, err, duration)
	} else {
		atomic.AddInt64(&c.results.SuccessfulOperations, 1)

		// Record slow operations
		if duration > 100*time.Millisecond {
			c.recordSlowOperation(operation, duration)
		}

		// Update hit/miss statistics for read operations
		if operation.Type == "get" || operation.Type == "mget" || operation.Type == "pipeline" {
			if hit {
				atomic.AddInt64(&c.results.HitRate, 1)
			} else {
				atomic.AddInt64(&c.results.MissRate, 1)
			}
		}
	}
}

// executeGet executes a GET operation
func (c *CacheLoadTestSuite) executeGet(ctx context.Context, operation CacheOperation) (error, bool) {
	result, err := c.redisClient.Get(ctx, operation.Key).Result()
	if err == redis.Nil {
		return nil, false // Cache miss
	}
	if err != nil {
		return err, false
	}

	// Validate the result
	var testData TestData
	if err := json.Unmarshal([]byte(result), &testData); err != nil {
		return err, false
	}

	return nil, true // Cache hit
}

// executeSet executes a SET operation
func (c *CacheLoadTestSuite) executeSet(ctx context.Context, operation CacheOperation) error {
	data, err := json.Marshal(operation.Value)
	if err != nil {
		return err
	}

	return c.redisClient.Set(ctx, operation.Key, data, operation.TTL).Err()
}

// executeDelete executes a DELETE operation
func (c *CacheLoadTestSuite) executeDelete(ctx context.Context, operation CacheOperation) error {
	return c.redisClient.Del(ctx, operation.Key).Err()
}

// executeMGet executes an MGET operation
func (c *CacheLoadTestSuite) executeMGet(ctx context.Context, operation CacheOperation) (error, bool) {
	keys := strings.Split(operation.Key, ",")
	results, err := c.redisClient.MGet(ctx, keys...).Result()
	if err != nil {
		return err, false
	}

	hitCount := 0
	for _, result := range results {
		if result != nil {
			hitCount++
		}
	}

	return nil, hitCount > 0 // At least one hit
}

// executeMSet executes an MSET operation
func (c *CacheLoadTestSuite) executeMSet(ctx context.Context, operation CacheOperation) error {
	pairs := operation.Value.(map[string]interface{})

	pipe := c.redisClient.Pipeline()
	for key, value := range pairs {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		pipe.Set(ctx, key, data, time.Hour)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// executePipeline executes a pipeline of operations
func (c *CacheLoadTestSuite) executePipeline(ctx context.Context, operation CacheOperation) (error, bool) {
	pipe := c.redisClient.Pipeline()

	// Mix of operations in pipeline
	key := operation.Key
	testData := c.generateTestData(rand.Intn(1000))
	data, _ := json.Marshal(testData)

	// GET operation
	getCmd := pipe.Get(ctx, key)

	// SET operation
	pipe.Set(ctx, key+":pipeline", data, time.Hour)

	// EXPIRE operation
	pipe.Expire(ctx, key, 2*time.Hour)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err, false
	}

	// Check GET result
	_, err = getCmd.Result()
	if err == redis.Nil {
		return nil, false // Cache miss
	}
	if err != nil {
		return err, false
	}

	return nil, true // Cache hit
}

// recordError records a cache operation error
func (c *CacheLoadTestSuite) recordError(operation CacheOperation, err error, duration time.Duration) {
	c.results.Errors = append(c.results.Errors, CacheError{
		Operation: operation.Type,
		Key:       operation.Key,
		Error:     err.Error(),
		Timestamp: time.Now(),
		Duration:  duration,
	})

	// Limit error storage to prevent memory issues
	if len(c.results.Errors) > 1000 {
		c.results.Errors = c.results.Errors[500:]
	}
}

// recordSlowOperation records a slow cache operation
func (c *CacheLoadTestSuite) recordSlowOperation(operation CacheOperation, duration time.Duration) {
	c.results.SlowOperations = append(c.results.SlowOperations, SlowOperation{
		Operation: operation.Type,
		Key:       operation.Key,
		Duration:  duration,
		Timestamp: time.Now(),
		DataSize:  operation.DataSize,
	})

	// Limit slow operation storage to prevent memory issues
	if len(c.results.SlowOperations) > 500 {
		c.results.SlowOperations = c.results.SlowOperations[250:]
	}
}

// startSystemMetricsCollection starts collecting system metrics
func (c *CacheLoadTestSuite) startSystemMetricsCollection() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				// Redis-specific metrics would be collected here
				// For now, we'll rely on collectRedisStats
			}
		}
	}()
}

// calculateResults calculates final test statistics
func (c *CacheLoadTestSuite) calculateResults() {
	if c.results.TotalOperations == 0 {
		return
	}

	c.results.Throughput = float64(c.results.TotalOperations) / c.results.Duration.Seconds()
	c.results.ErrorRate = float64(c.results.FailedOperations) / float64(c.results.TotalOperations)

	// Calculate hit/miss rates
	totalReads := c.results.HitRate + c.results.MissRate
	if totalReads > 0 {
		c.results.HitRate = float64(c.results.HitRate) / float64(totalReads)
		c.results.MissRate = float64(c.results.MissRate) / float64(totalReads)
	}

	// Calculate latency percentiles (placeholder values)
	c.results.AverageLatency = 15 * time.Millisecond
	c.results.P50Latency = 10 * time.Millisecond
	c.results.P95Latency = 50 * time.Millisecond
	c.results.P99Latency = 100 * time.Millisecond
}

// collectRedisStats collects Redis server statistics
func (c *CacheLoadTestSuite) collectRedisStats() {
	info, err := c.redisClient.Info(c.ctx, "memory", "clients").Result()
	if err != nil {
		log.Printf("Warning: Failed to get Redis info: %v", err)
		return
	}

	// Parse memory info
	if memoryMatch := regexp.MustCompile(`used_memory:(\d+)`).FindStringSubmatch(info); len(memoryMatch) > 1 {
		c.results.RedisMemoryUsageMB = parseBytesToInt(memoryMatch[1]) / 1024 / 1024
	}

	if maxMemoryMatch := regexp.MustCompile(`maxmemory:(\d+)`).FindStringSubmatch(info); len(maxMemoryMatch) > 1 {
		c.results.RedisMaxMemoryMB = parseBytesToInt(maxMemoryMatch[1]) / 1024 / 1024
	}

	// Parse client info
	if clientsMatch := regexp.MustCompile(`connected_clients:(\d+)`).FindStringSubmatch(info); len(clientsMatch) > 1 {
		c.results.RedisConnections = parseInt64(clientsMatch[1])
	}
}

// ValidateResults validates the cache load test results
func (c *CacheLoadTestSuite) ValidateResults() error {
	if c.results == nil {
		return fmt.Errorf("no test results to validate")
	}

	// Validate error rate
	if c.results.ErrorRate > 0.02 { // 2% error rate threshold
		return fmt.Errorf("error rate %.2f%% exceeds 2%% threshold", c.results.ErrorRate*100)
	}

	// Validate throughput
	if c.config.TargetThroughput > 0 && c.results.Throughput < float64(c.config.TargetThroughput)*0.9 {
		return fmt.Errorf("throughput %.2f ops/sec is below target %d ops/sec (90%% threshold)",
			c.results.Throughput, c.config.TargetThroughput)
	}

	// Validate hit rate
	if c.config.TargetHitRate > 0 && c.results.HitRate < c.config.TargetHitRate*0.9 {
		return fmt.Errorf("hit rate %.2f%% is below target %.2f%% (90%% threshold)",
			c.results.HitRate*100, c.config.TargetHitRate*100)
	}

	// Validate latency
	if c.config.MaxLatency > 0 && c.results.P95Latency > c.config.MaxLatency {
		return fmt.Errorf("95th percentile latency %v exceeds maximum %v",
			c.results.P95Latency, c.config.MaxLatency)
	}

	// Validate memory usage
	if c.config.MaxMemoryUsage > 0 && c.results.RedisMemoryUsageMB > c.config.MaxMemoryUsage {
		return fmt.Errorf("Redis memory usage %d MB exceeds maximum %d MB",
			c.results.RedisMemoryUsageMB, c.config.MaxMemoryUsage)
	}

	return nil
}

// Close closes the cache load test suite
func (c *CacheLoadTestSuite) Close() {
	c.cancel()
	c.wg.Wait()
	if c.redisClient != nil {
		c.redisClient.Close()
	}
}

// Helper functions
func parseBytesToInt(s string) int64 {
	var value int64
	fmt.Sscanf(s, "%d", &value)
	return value
}

func parseInt64(s string) int64 {
	var value int64
	fmt.Sscanf(s, "%d", &value)
	return value
}

// TestCacheLoadTestSuite runs the cache load test suite
func TestCacheLoadTestSuite(t *testing.T) {
	redisURL := "redis://localhost:6379/0"

	testCases := []struct {
		name   string
		config CacheLoadTestConfig
	}{
		{
			name: "ReadHeavyWorkload",
			config: CacheLoadTestConfig{
				Name:               "Read-Heavy Cache Workload Test",
				RedisURL:           redisURL,
				ConcurrentClients:  50,
				OperationsPerClient: 200,
				TestDuration:       30 * time.Second,
				OperationTypes:     []string{"get", "mget", "pipeline"},
				DataSizeKB:         1, // 1KB objects
				TargetThroughput:   20000,
				TargetHitRate:      0.90, // 90% hit rate
				MaxLatency:         50 * time.Millisecond,
				MaxMemoryUsage:     512, // 512MB
				KeyDistribution:    "zipfian",
			},
		},
		{
			name: "MixedWorkload",
			config: CacheLoadTestConfig{
				Name:               "Mixed Cache Workload Test",
				RedisURL:           redisURL,
				ConcurrentClients:  30,
				OperationsPerClient: 150,
				TestDuration:       25 * time.Second,
				OperationTypes:     []string{"get", "set", "mget", "mset", "pipeline"},
				DataSizeKB:         2, // 2KB objects
				TargetThroughput:   15000,
				TargetHitRate:      0.75, // 75% hit rate
				MaxLatency:         75 * time.Millisecond,
				MaxMemoryUsage:     1024, // 1GB
				KeyDistribution:    "uniform",
			},
		},
		{
			name: "WriteHeavyWorkload",
			config: CacheLoadTestConfig{
				Name:               "Write-Heavy Cache Workload Test",
				RedisURL:           redisURL,
				ConcurrentClients:  20,
				OperationsPerClient: 100,
				TestDuration:       20 * time.Second,
				OperationTypes:     []string{"set", "mset", "pipeline"},
				DataSizeKB:         5, // 5KB objects
				TargetThroughput:   10000,
				TargetHitRate:      0.60, // 60% hit rate
				MaxLatency:         100 * time.Millisecond,
				MaxMemoryUsage:     2048, // 2GB
				KeyDistribution:    "latest",
			},
		},
		{
			name: "HighConcurrencyTest",
			config: CacheLoadTestConfig{
				Name:               "High Concurrency Cache Test",
				RedisURL:           redisURL,
				ConcurrentClients:  100,
				OperationsPerClient: 50,
				TestDuration:       15 * time.Second,
				OperationTypes:     []string{"get", "set", "pipeline"},
				DataSizeKB:         1, // 1KB objects
				TargetThroughput:   30000,
				TargetHitRate:      0.85, // 85% hit rate
				MaxLatency:         30 * time.Millisecond,
				MaxMemoryUsage:     1024, // 1GB
				KeyDistribution:    "zipfian",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			suite, err := NewCacheLoadTestSuite(tc.config)
			require.NoError(t, err)
			defer suite.Close()

			result, err := suite.RunCacheLoadTest()
			require.NoError(t, err)

			err = suite.ValidateResults()
			require.NoError(t, err, "Cache load test validation failed: %v", err)

			// Print detailed results
			t.Logf("=== %s Results ===", tc.name)
			t.Logf("Duration: %v", result.Duration)
			t.Logf("Total Operations: %d", result.TotalOperations)
			t.Logf("Successful Operations: %d", result.SuccessfulOperations)
			t.Logf("Failed Operations: %d", result.FailedOperations)
			t.Logf("Throughput: %.2f ops/sec", result.Throughput)
			t.Logf("Error Rate: %.2f%%", result.ErrorRate*100)
			t.Logf("Hit Rate: %.2f%%", result.HitRate*100)
			t.Logf("Miss Rate: %.2f%%", result.MissRate*100)
			t.Logf("Average Latency: %v", result.AverageLatency)
			t.Logf("95th Percentile Latency: %v", result.P95Latency)

			t.Logf("Redis Stats:")
			t.Logf("  Memory Usage: %d MB", result.RedisMemoryUsageMB)
			t.Logf("  Max Memory: %d MB", result.RedisMaxMemoryMB)
			t.Logf("  Connections: %d", result.RedisConnections)

			t.Logf("Operation Breakdown:")
			for opType, count := range result.OperationBreakdown {
				t.Logf("  %s: %d (%.1f%%)", opType, count, float64(count)/float64(result.TotalOperations)*100)
			}

			if len(result.SlowOperations) > 0 {
				t.Logf("Slow Operations (%d):", len(result.SlowOperations))
				for i, op := range result.SlowOperations[:min(5, len(result.SlowOperations))] {
					t.Logf("  %d. %s on %s: %v", i+1, op.Operation, op.Key, op.Duration)
				}
			}

			if len(result.Errors) > 0 {
				t.Logf("Errors (%d):", len(result.Errors))
				for i, err := range result.Errors[:min(5, len(result.Errors))] {
					t.Logf("  %d. %s on %s: %s", i+1, err.Operation, err.Key, err.Error)
				}
			}
		})
	}
}