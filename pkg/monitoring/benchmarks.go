package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// BenchmarkSuite collects performance benchmarks and comparisons
type BenchmarkSuite struct {
	// Performance benchmark metrics
	benchmarkDuration *prometheus.HistogramVec
	benchmarkThroughput *prometheus.GaugeVec
	benchmarkLatency *prometheus.HistogramVec
	benchmarkErrorRate *prometheus.GaugeVec

	// Resource utilization metrics
	cpuUsage *prometheus.GaugeVec
	memoryUsage *prometheus.GaugeVec
	goroutineCount *prometheus.GaugeVec
	gcPauseDuration *prometheus.HistogramVec

	// Database performance benchmarks
	dbQueryLatency *prometheus.HistogramVec
	dbThroughput *prometheus.GaugeVec
	dbConnectionPool *prometheus.GaugeVec

	// Cache performance benchmarks
	cacheHitRatio *prometheus.GaugeVec
	cacheLatency *prometheus.HistogramVec
	cacheThroughput *prometheus.GaugeVec

	// API endpoint benchmarks
	apiRequestLatency *prometheus.HistogramVec
	apiRequestThroughput *prometheus.GaugeVec
	apiErrorRate *prometheus.GaugeVec

	mu sync.RWMutex
	activeBenchmarks map[string]*BenchmarkResult
}

// BenchmarkResult represents a single benchmark result
type BenchmarkResult struct {
	Name         string                 `json:"name"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Duration     time.Duration          `json:"duration"`
	Operations   int64                  `json:"operations"`
	Throughput   float64                `json:"throughput"`
	AvgLatency   time.Duration          `json:"avg_latency"`
	P95Latency   time.Duration          `json:"p95_latency"`
	P99Latency   time.Duration          `json:"p99_latency"`
	ErrorCount   int64                  `json:"error_count"`
	SuccessRate  float64                `json:"success_rate"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// BenchmarkConfig holds configuration for benchmark runs
type BenchmarkConfig struct {
	Name           string        `json:"name"`
	Duration       time.Duration `json:"duration"`
	Operations     int64         `json:"operations"`
	Concurrency    int           `json:"concurrency"`
	WarmupDuration time.Duration `json:"warmup_duration"`
	Timeout        time.Duration `json:"timeout"`
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite() *BenchmarkSuite {
	return &BenchmarkSuite{
		benchmarkDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "erpgo_benchmark_duration_seconds",
				Help:    "Duration of benchmark runs",
				Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800},
			},
			[]string{"benchmark_name", "category"},
		),
		benchmarkThroughput: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_benchmark_throughput_ops_per_second",
				Help: "Throughput of benchmark runs",
			},
			[]string{"benchmark_name", "category"},
		),
		benchmarkLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "erpgo_benchmark_latency_seconds",
				Help:    "Latency measurements for benchmarks",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"benchmark_name", "percentile"},
		),
		benchmarkErrorRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_benchmark_error_rate",
				Help: "Error rate for benchmark runs",
			},
			[]string{"benchmark_name"},
		),
		cpuUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_benchmark_cpu_usage_percent",
				Help: "CPU usage during benchmarks",
			},
			[]string{"benchmark_name"},
		),
		memoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_benchmark_memory_bytes",
				Help: "Memory usage during benchmarks",
			},
			[]string{"benchmark_name", "type"},
		),
		goroutineCount: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_benchmark_goroutines",
				Help: "Goroutine count during benchmarks",
			},
			[]string{"benchmark_name"},
		),
		gcPauseDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "erpgo_benchmark_gc_pause_seconds",
				Help:    "GC pause duration during benchmarks",
				Buckets: []float64{0.000001, 0.00001, 0.0001, 0.001, 0.01, 0.1},
			},
			[]string{"benchmark_name"},
		),
		dbQueryLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "erpgo_benchmark_db_query_latency_seconds",
				Help:    "Database query latency during benchmarks",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
			[]string{"benchmark_name", "operation", "table"},
		),
		dbThroughput: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_benchmark_db_throughput_ops_per_second",
				Help: "Database throughput during benchmarks",
			},
			[]string{"benchmark_name", "operation"},
		),
		dbConnectionPool: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_benchmark_db_connections",
				Help: "Database connection pool usage during benchmarks",
			},
			[]string{"benchmark_name", "state"},
		),
		cacheHitRatio: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_benchmark_cache_hit_ratio",
				Help: "Cache hit ratio during benchmarks",
			},
			[]string{"benchmark_name", "cache_type"},
		),
		cacheLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "erpgo_benchmark_cache_latency_seconds",
				Help:    "Cache operation latency during benchmarks",
				Buckets: []float64{0.000001, 0.00001, 0.0001, 0.001, 0.01, 0.1},
			},
			[]string{"benchmark_name", "operation", "cache_type"},
		),
		cacheThroughput: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_benchmark_cache_throughput_ops_per_second",
				Help: "Cache throughput during benchmarks",
			},
			[]string{"benchmark_name", "operation", "cache_type"},
		),
		apiRequestLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "erpgo_benchmark_api_latency_seconds",
				Help:    "API request latency during benchmarks",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"benchmark_name", "endpoint", "method"},
		),
		apiRequestThroughput: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_benchmark_api_throughput_requests_per_second",
				Help: "API request throughput during benchmarks",
			},
			[]string{"benchmark_name", "endpoint", "method"},
		),
		apiErrorRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "erpgo_benchmark_api_error_rate",
				Help: "API error rate during benchmarks",
			},
			[]string{"benchmark_name", "endpoint", "method"},
		),
		activeBenchmarks: make(map[string]*BenchmarkResult),
	}
}

// RunBenchmark executes a benchmark with the given configuration and function
func (bs *BenchmarkSuite) RunBenchmark(ctx context.Context, config BenchmarkConfig, fn func(context.Context) error) (*BenchmarkResult, error) {
	// Validate configuration
	if config.Duration <= 0 && config.Operations <= 0 {
		return nil, fmt.Errorf("either duration or operations must be specified")
	}

	if config.Concurrency <= 0 {
		config.Concurrency = 1
	}

	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Minute
	}

	// Initialize benchmark result
	result := &BenchmarkResult{
		Name:      config.Name,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Store active benchmark
	bs.mu.Lock()
	bs.activeBenchmarks[config.Name] = result
	bs.mu.Unlock()

	// Cleanup function
	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		// Calculate throughput and success rate
		if result.Duration > 0 {
			result.Throughput = float64(result.Operations) / result.Duration.Seconds()
		}
		if result.Operations > 0 {
			result.SuccessRate = float64(result.Operations-result.ErrorCount) / float64(result.Operations) * 100
		}

		// Record metrics
		bs.recordBenchmarkMetrics(result)

		// Remove from active benchmarks
		bs.mu.Lock()
		delete(bs.activeBenchmarks, config.Name)
		bs.mu.Unlock()
	}()

	// Warmup phase if specified
	if config.WarmupDuration > 0 {
		warmupCtx, cancel := context.WithTimeout(ctx, config.WarmupDuration)
		defer cancel()

		if err := fn(warmupCtx); err != nil && warmupCtx.Err() == nil {
			// Log warmup error but don't fail the benchmark
			fmt.Printf("Benchmark warmup failed: %v\n", err)
		}
	}

	// Start resource monitoring
	resourceCtx, cancelResourceMonitoring := context.WithCancel(ctx)
	defer cancelResourceMonitoring()

	go bs.monitorResources(resourceCtx, config.Name)

	// Determine execution mode
	if config.Duration > 0 {
		// Time-based benchmark
		return bs.runTimeBasedBenchmark(ctx, config, result, fn)
	} else {
		// Operations-based benchmark
		return bs.runOperationsBasedBenchmark(ctx, config, result, fn)
	}
}

// runTimeBasedBenchmark runs a benchmark for a specified duration
func (bs *BenchmarkSuite) runTimeBasedBenchmark(ctx context.Context, config BenchmarkConfig, result *BenchmarkResult, fn func(context.Context) error) (*BenchmarkResult, error) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// Create duration context
	durationCtx, durationCancel := context.WithTimeout(timeoutCtx, config.Duration)
	defer durationCancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalOps int64
	var totalErrors int64
	latencies := make([]time.Duration, 0)

	// Start workers
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for {
				select {
				case <-durationCtx.Done():
					return
				default:
					start := time.Now()
					err := fn(durationCtx)
					latency := time.Since(start)

					mu.Lock()
					totalOps++
					if err != nil && durationCtx.Err() == nil {
						totalErrors++
					}
					latencies = append(latencies, latency)
					mu.Unlock()
				}
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()

	// Update result
	mu.Lock()
	result.Operations = totalOps
	result.ErrorCount = totalErrors
	if len(latencies) > 0 {
		result.AvgLatency = bs.calculateAverage(latencies)
		result.P95Latency = bs.calculatePercentile(latencies, 0.95)
		result.P99Latency = bs.calculatePercentile(latencies, 0.99)
	}
	mu.Unlock()

	return result, nil
}

// runOperationsBasedBenchmark runs a benchmark for a specified number of operations
func (bs *BenchmarkSuite) runOperationsBasedBenchmark(ctx context.Context, config BenchmarkConfig, result *BenchmarkResult, fn func(context.Context) error) (*BenchmarkResult, error) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalOps int64
	var totalErrors int64
	latencies := make([]time.Duration, 0)

	// Calculate operations per worker
	opsPerWorker := config.Operations / int64(config.Concurrency)
	if opsPerWorker < 1 {
		opsPerWorker = 1
	}

	// Start workers
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			var workerOps int64
			for workerOps < opsPerWorker {
				select {
				case <-timeoutCtx.Done():
					return
				default:
					start := time.Now()
					err := fn(timeoutCtx)
					latency := time.Since(start)

					mu.Lock()
					totalOps++
					workerOps++
					if err != nil && timeoutCtx.Err() == nil {
						totalErrors++
					}
					latencies = append(latencies, latency)
					mu.Unlock()
				}
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()

	// Update result
	mu.Lock()
	result.Operations = totalOps
	result.ErrorCount = totalErrors
	if len(latencies) > 0 {
		result.AvgLatency = bs.calculateAverage(latencies)
		result.P95Latency = bs.calculatePercentile(latencies, 0.95)
		result.P99Latency = bs.calculatePercentile(latencies, 0.99)
	}
	mu.Unlock()

	return result, nil
}

// monitorResources monitors system resources during benchmark execution
func (bs *BenchmarkSuite) monitorResources(ctx context.Context, benchmarkName string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Collect system metrics
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Record Prometheus metrics
			bs.memoryUsage.WithLabelValues(benchmarkName, "alloc").Set(float64(m.Alloc))
			bs.memoryUsage.WithLabelValues(benchmarkName, "total_alloc").Set(float64(m.TotalAlloc))
			bs.memoryUsage.WithLabelValues(benchmarkName, "sys").Set(float64(m.Sys))
			bs.memoryUsage.WithLabelValues(benchmarkName, "heap_alloc").Set(float64(m.HeapAlloc))
			bs.memoryUsage.WithLabelValues(benchmarkName, "heap_sys").Set(float64(m.HeapSys))

			bs.goroutineCount.WithLabelValues(benchmarkName).Set(float64(runtime.NumGoroutine()))

			if m.NumGC > 0 {
				bs.gcPauseDuration.WithLabelValues(benchmarkName).Observe(float64(m.PauseTotalNs) / 1e9) // Convert nanoseconds to seconds
			}
		}
	}
}

// recordBenchmarkMetrics records benchmark metrics to Prometheus
func (bs *BenchmarkSuite) recordBenchmarkMetrics(result *BenchmarkResult) {
	// Record basic benchmark metrics
	bs.benchmarkDuration.WithLabelValues(result.Name, "duration").Observe(result.Duration.Seconds())
	bs.benchmarkThroughput.WithLabelValues(result.Name, "throughput").Set(result.Throughput)
	bs.benchmarkLatency.WithLabelValues(result.Name, "avg").Observe(result.AvgLatency.Seconds())
	bs.benchmarkLatency.WithLabelValues(result.Name, "p95").Observe(result.P95Latency.Seconds())
	bs.benchmarkLatency.WithLabelValues(result.Name, "p99").Observe(result.P99Latency.Seconds())

	errorRate := 0.0
	if result.Operations > 0 {
		errorRate = float64(result.ErrorCount) / float64(result.Operations) * 100
	}
	bs.benchmarkErrorRate.WithLabelValues(result.Name).Set(errorRate)
}

// calculateAverage calculates the average of a slice of durations
func (bs *BenchmarkSuite) calculateAverage(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}

	return total / time.Duration(len(durations))
}

// calculatePercentile calculates the percentile of a slice of durations
func (bs *BenchmarkSuite) calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Create a copy and sort it
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	// Simple bubble sort for small datasets
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate percentile index
	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// GetActiveBenchmarks returns currently running benchmarks
func (bs *BenchmarkSuite) GetActiveBenchmarks() map[string]*BenchmarkResult {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*BenchmarkResult)
	for name, benchmark := range bs.activeBenchmarks {
		result[name] = benchmark
	}

	return result
}

// StopBenchmark stops a running benchmark by name
func (bs *BenchmarkSuite) StopBenchmark(benchmarkName string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if benchmark, exists := bs.activeBenchmarks[benchmarkName]; exists {
		// Set end time to stop the benchmark
		benchmark.EndTime = time.Now()
		return nil
	}

	return fmt.Errorf("benchmark '%s' not found", benchmarkName)
}

// Global benchmark suite instance
var GlobalBenchmarkSuite = NewBenchmarkSuite()

// Convenience functions for global benchmark suite

// RunBenchmark runs a benchmark using the global suite
func RunBenchmark(ctx context.Context, config BenchmarkConfig, fn func(context.Context) error) (*BenchmarkResult, error) {
	return GlobalBenchmarkSuite.RunBenchmark(ctx, config, fn)
}

// GetActiveBenchmarks returns active benchmarks from the global suite
func GetActiveBenchmarks() map[string]*BenchmarkResult {
	return GlobalBenchmarkSuite.GetActiveBenchmarks()
}

// StopBenchmark stops a benchmark in the global suite
func StopBenchmark(benchmarkName string) error {
	return GlobalBenchmarkSuite.StopBenchmark(benchmarkName)
}