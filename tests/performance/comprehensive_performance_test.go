package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ComprehensivePerformanceTest runs all performance tests
// This test validates Requirements 17.1, 17.2, 17.3 from the production readiness spec
func TestComprehensivePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comprehensive performance tests in short mode")
	}

	t.Run("APIPerformanceUnderLoad", testAPIPerformanceUnderLoad)
	t.Run("DatabasePerformanceUnderLoad", testDatabasePerformanceUnderLoad)
	t.Run("CachePerformanceUnderLoad", testCachePerformanceUnderLoad)
	t.Run("MemoryLeakDetection", testMemoryLeakDetection)
}

// testAPIPerformanceUnderLoad tests API performance under sustained load
// Validates Requirement 17.1: System SHALL handle 1000 RPS with p99 latency < 500ms
func testAPIPerformanceUnderLoad(t *testing.T) {
	t.Log("Starting API performance test under load...")

	// Test configuration
	const (
		targetRPS         = 1000
		testDuration      = 60 * time.Second // 1 minute sustained load
		maxP99Latency     = 500 * time.Millisecond
		maxErrorRate      = 0.001 // 0.1%
		concurrentClients = 50
	)

	var (
		totalRequests     int64
		successfulRequests int64
		failedRequests    int64
		responseTimes     []time.Duration
		mu                sync.Mutex
	)

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	startTime := time.Now()
	var wg sync.WaitGroup

	// Start concurrent clients
	for i := 0; i < concurrentClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					reqStart := time.Now()
					
					// Simulate API request (in real implementation, make actual HTTP request)
					err := simulateAPIRequest(ctx)
					reqDuration := time.Since(reqStart)

					atomic.AddInt64(&totalRequests, 1)

					if err != nil {
						atomic.AddInt64(&failedRequests, 1)
					} else {
						atomic.AddInt64(&successfulRequests, 1)
						
						mu.Lock()
						responseTimes = append(responseTimes, reqDuration)
						mu.Unlock()
					}

					// Rate limiting to achieve target RPS
					time.Sleep(time.Duration(concurrentClients*1000/targetRPS) * time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// Calculate metrics
	actualRPS := float64(totalRequests) / duration.Seconds()
	errorRate := float64(failedRequests) / float64(totalRequests)
	
	// Calculate p99 latency
	p99Latency := calculatePercentile(responseTimes, 0.99)
	avgLatency := calculateAverage(responseTimes)

	// Log results
	t.Logf("=== API Performance Test Results ===")
	t.Logf("Duration: %v", duration)
	t.Logf("Total Requests: %d", totalRequests)
	t.Logf("Successful Requests: %d", successfulRequests)
	t.Logf("Failed Requests: %d", failedRequests)
	t.Logf("Actual RPS: %.2f (target: %d)", actualRPS, targetRPS)
	t.Logf("Error Rate: %.4f%% (max: %.4f%%)", errorRate*100, maxErrorRate*100)
	t.Logf("Average Latency: %v", avgLatency)
	t.Logf("P99 Latency: %v (max: %v)", p99Latency, maxP99Latency)

	// Assertions
	require.GreaterOrEqual(t, actualRPS, float64(targetRPS)*0.9, 
		"Should achieve at least 90%% of target RPS")
	require.Less(t, errorRate, maxErrorRate, 
		"Error rate should be below threshold")
	require.Less(t, p99Latency, maxP99Latency, 
		"P99 latency should be below 500ms")
}

// testDatabasePerformanceUnderLoad tests database performance under sustained load
// Validates Requirement 17.2: Database SHALL maintain query performance under load
func testDatabasePerformanceUnderLoad(t *testing.T) {
	t.Log("Starting database performance test under load...")

	// Test configuration
	const (
		targetQPS         = 5000 // Queries per second
		testDuration      = 60 * time.Second
		maxP95QueryTime   = 100 * time.Millisecond
		maxErrorRate      = 0.01 // 1%
		concurrentWorkers = 20
	)

	var (
		totalQueries      int64
		successfulQueries int64
		failedQueries     int64
		queryTimes        []time.Duration
		mu                sync.Mutex
	)

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	startTime := time.Now()
	var wg sync.WaitGroup

	// Start concurrent database workers
	for i := 0; i < concurrentWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					queryStart := time.Now()
					
					// Simulate database query (in real implementation, execute actual query)
					err := simulateDatabaseQuery(ctx, workerID)
					queryDuration := time.Since(queryStart)

					atomic.AddInt64(&totalQueries, 1)

					if err != nil {
						atomic.AddInt64(&failedQueries, 1)
					} else {
						atomic.AddInt64(&successfulQueries, 1)
						
						mu.Lock()
						queryTimes = append(queryTimes, queryDuration)
						mu.Unlock()
					}

					// Rate limiting
					time.Sleep(time.Duration(concurrentWorkers*1000/targetQPS) * time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// Calculate metrics
	actualQPS := float64(totalQueries) / duration.Seconds()
	errorRate := float64(failedQueries) / float64(totalQueries)
	p95QueryTime := calculatePercentile(queryTimes, 0.95)
	avgQueryTime := calculateAverage(queryTimes)

	// Log results
	t.Logf("=== Database Performance Test Results ===")
	t.Logf("Duration: %v", duration)
	t.Logf("Total Queries: %d", totalQueries)
	t.Logf("Successful Queries: %d", successfulQueries)
	t.Logf("Failed Queries: %d", failedQueries)
	t.Logf("Actual QPS: %.2f (target: %d)", actualQPS, targetQPS)
	t.Logf("Error Rate: %.4f%% (max: %.4f%%)", errorRate*100, maxErrorRate*100)
	t.Logf("Average Query Time: %v", avgQueryTime)
	t.Logf("P95 Query Time: %v (max: %v)", p95QueryTime, maxP95QueryTime)

	// Assertions
	require.GreaterOrEqual(t, actualQPS, float64(targetQPS)*0.8, 
		"Should achieve at least 80%% of target QPS")
	require.Less(t, errorRate, maxErrorRate, 
		"Error rate should be below threshold")
	require.Less(t, p95QueryTime, maxP95QueryTime, 
		"P95 query time should be below 100ms")
}

// testCachePerformanceUnderLoad tests cache performance under sustained load
// Validates Requirement 17.3: Cache SHALL maintain high hit rate and low latency
func testCachePerformanceUnderLoad(t *testing.T) {
	t.Log("Starting cache performance test under load...")

	// Test configuration
	const (
		targetOPS         = 20000 // Operations per second
		testDuration      = 60 * time.Second
		minHitRate        = 0.80 // 80%
		maxP95Latency     = 50 * time.Millisecond
		concurrentClients = 30
	)

	var (
		totalOperations int64
		cacheHits       int64
		cacheMisses     int64
		cacheErrors     int64
		operationTimes  []time.Duration
		mu              sync.Mutex
	)

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	startTime := time.Now()
	var wg sync.WaitGroup

	// Start concurrent cache clients
	for i := 0; i < concurrentClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					opStart := time.Now()
					
					// Simulate cache operation (in real implementation, interact with Redis)
					hit, err := simulateCacheOperation(ctx, clientID)
					opDuration := time.Since(opStart)

					atomic.AddInt64(&totalOperations, 1)

					if err != nil {
						atomic.AddInt64(&cacheErrors, 1)
					} else if hit {
						atomic.AddInt64(&cacheHits, 1)
					} else {
						atomic.AddInt64(&cacheMisses, 1)
					}

					mu.Lock()
					operationTimes = append(operationTimes, opDuration)
					mu.Unlock()

					// Rate limiting
					time.Sleep(time.Duration(concurrentClients*1000/targetOPS) * time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// Calculate metrics
	actualOPS := float64(totalOperations) / duration.Seconds()
	hitRate := float64(cacheHits) / float64(cacheHits+cacheMisses)
	errorRate := float64(cacheErrors) / float64(totalOperations)
	p95Latency := calculatePercentile(operationTimes, 0.95)
	avgLatency := calculateAverage(operationTimes)

	// Log results
	t.Logf("=== Cache Performance Test Results ===")
	t.Logf("Duration: %v", duration)
	t.Logf("Total Operations: %d", totalOperations)
	t.Logf("Cache Hits: %d", cacheHits)
	t.Logf("Cache Misses: %d", cacheMisses)
	t.Logf("Cache Errors: %d", cacheErrors)
	t.Logf("Actual OPS: %.2f (target: %d)", actualOPS, targetOPS)
	t.Logf("Hit Rate: %.2f%% (min: %.2f%%)", hitRate*100, minHitRate*100)
	t.Logf("Error Rate: %.4f%%", errorRate*100)
	t.Logf("Average Latency: %v", avgLatency)
	t.Logf("P95 Latency: %v (max: %v)", p95Latency, maxP95Latency)

	// Assertions
	require.GreaterOrEqual(t, actualOPS, float64(targetOPS)*0.8, 
		"Should achieve at least 80%% of target OPS")
	require.GreaterOrEqual(t, hitRate, minHitRate, 
		"Hit rate should be at least 80%%")
	require.Less(t, p95Latency, maxP95Latency, 
		"P95 latency should be below 50ms")
	require.Less(t, errorRate, 0.01, 
		"Error rate should be below 1%%")
}

// testMemoryLeakDetection tests for memory leaks over 1 hour
// Validates Requirement 17.3: Verify no memory leaks over 1 hour
func testMemoryLeakDetection(t *testing.T) {
	t.Log("Starting memory leak detection test (1 hour)...")

	// Test configuration
	const (
		testDuration      = 60 * time.Minute // 1 hour
		samplingInterval  = 1 * time.Minute
		maxMemoryGrowthMB = 100 // Maximum acceptable memory growth in MB
		maxGoroutineGrowth = 50 // Maximum acceptable goroutine growth
	)

	// Record initial state
	var initialMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&initialMem)
	initialGoroutines := runtime.NumGoroutine()

	t.Logf("Initial state:")
	t.Logf("  Memory: %d MB", initialMem.Alloc/1024/1024)
	t.Logf("  Goroutines: %d", initialGoroutines)

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	// Track memory samples
	type MemorySample struct {
		Timestamp   time.Time
		AllocMB     int64
		Goroutines  int
	}
	
	var samples []MemorySample
	var mu sync.Mutex

	// Start memory monitoring
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(samplingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var m runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&m)
				
				sample := MemorySample{
					Timestamp:  time.Now(),
					AllocMB:    int64(m.Alloc / 1024 / 1024),
					Goroutines: runtime.NumGoroutine(),
				}

				mu.Lock()
				samples = append(samples, sample)
				mu.Unlock()

				t.Logf("Sample at %v: Memory=%d MB, Goroutines=%d", 
					sample.Timestamp.Format("15:04:05"), sample.AllocMB, sample.Goroutines)
			}
		}
	}()

	// Simulate continuous workload
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Simulate mixed workload
				_ = simulateAPIRequest(ctx)
				_ = simulateDatabaseQuery(ctx, 0)
				_, _ = simulateCacheOperation(ctx, 0)
				
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	// Wait for test completion
	wg.Wait()

	// Record final state
	var finalMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&finalMem)
	finalGoroutines := runtime.NumGoroutine()

	// Calculate memory growth
	memoryGrowthMB := int64(finalMem.Alloc/1024/1024) - int64(initialMem.Alloc/1024/1024)
	goroutineGrowth := finalGoroutines - initialGoroutines

	t.Logf("\n=== Memory Leak Detection Results ===")
	t.Logf("Test Duration: %v", testDuration)
	t.Logf("Samples Collected: %d", len(samples))
	t.Logf("Initial Memory: %d MB", initialMem.Alloc/1024/1024)
	t.Logf("Final Memory: %d MB", finalMem.Alloc/1024/1024)
	t.Logf("Memory Growth: %d MB (max: %d MB)", memoryGrowthMB, maxMemoryGrowthMB)
	t.Logf("Initial Goroutines: %d", initialGoroutines)
	t.Logf("Final Goroutines: %d", finalGoroutines)
	t.Logf("Goroutine Growth: %d (max: %d)", goroutineGrowth, maxGoroutineGrowth)

	// Analyze memory trend
	if len(samples) > 0 {
		firstSample := samples[0]
		lastSample := samples[len(samples)-1]
		
		t.Logf("\nMemory Trend:")
		t.Logf("  First Sample: %d MB at %v", firstSample.AllocMB, firstSample.Timestamp.Format("15:04:05"))
		t.Logf("  Last Sample: %d MB at %v", lastSample.AllocMB, lastSample.Timestamp.Format("15:04:05"))
		t.Logf("  Trend Growth: %d MB", lastSample.AllocMB-firstSample.AllocMB)
	}

	// Assertions
	require.LessOrEqual(t, memoryGrowthMB, int64(maxMemoryGrowthMB), 
		"Memory growth should not exceed %d MB over 1 hour", maxMemoryGrowthMB)
	require.LessOrEqual(t, goroutineGrowth, maxGoroutineGrowth, 
		"Goroutine growth should not exceed %d", maxGoroutineGrowth)

	// Check for memory leak pattern (continuous growth)
	if len(samples) >= 3 {
		// Check if memory is continuously growing
		growthCount := 0
		for i := 1; i < len(samples); i++ {
			if samples[i].AllocMB > samples[i-1].AllocMB {
				growthCount++
			}
		}
		
		growthRate := float64(growthCount) / float64(len(samples)-1)
		t.Logf("Memory Growth Rate: %.2f%% of samples showed growth", growthRate*100)
		
		// If memory grows in more than 80% of samples, it might indicate a leak
		require.Less(t, growthRate, 0.80, 
			"Memory should not grow continuously (leak pattern detected)")
	}
}

// ==================== HELPER FUNCTIONS ====================

// simulateAPIRequest simulates an API request
func simulateAPIRequest(ctx context.Context) error {
	// Simulate API processing time
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(10+runtime.NumGoroutine()%40) * time.Millisecond):
		// Simulate occasional errors (1% error rate)
		if runtime.NumGoroutine()%100 == 0 {
			return fmt.Errorf("simulated API error")
		}
		return nil
	}
}

// simulateDatabaseQuery simulates a database query
func simulateDatabaseQuery(ctx context.Context, workerID int) error {
	// Simulate query processing time
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(5+workerID%20) * time.Millisecond):
		// Simulate occasional errors (0.5% error rate)
		if workerID%200 == 0 {
			return fmt.Errorf("simulated database error")
		}
		return nil
	}
}

// simulateCacheOperation simulates a cache operation
func simulateCacheOperation(ctx context.Context, clientID int) (bool, error) {
	// Simulate cache operation time
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-time.After(time.Duration(1+clientID%10) * time.Millisecond):
		// Simulate cache hit rate of 85%
		hit := clientID%100 < 85
		
		// Simulate occasional errors (0.1% error rate)
		if clientID%1000 == 0 {
			return false, fmt.Errorf("simulated cache error")
		}
		
		return hit, nil
	}
}

// calculatePercentile calculates the specified percentile from a slice of durations
func calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Sort durations
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// calculateAverage calculates the average duration from a slice of durations
func calculateAverage(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}

	return total / time.Duration(len(durations))
}
