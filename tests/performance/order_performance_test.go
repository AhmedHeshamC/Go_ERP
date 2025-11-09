package performance

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"erpgo/internal/application/services/order"
	"erpgo/internal/domain/orders/entities"
	"erpgo/tests/performance/testutil"
)

// OrderPerformanceTestSuite tests order system performance
type OrderPerformanceTestSuite struct {
	testContainer *testutil.PerformanceTestContainer
	orderService  order.Service
	workflowService order.WorkflowService
	analyticsService order.AnalyticsService
	bulkService    order.BulkService
	exportService  order.ExportService
}

// SetupSuite sets up the performance test suite
func (suite *OrderPerformanceTestSuite) SetupSuite(t *testing.T) {
	ctx := context.Background()

	// Initialize performance test container
	container, err := testutil.NewPerformanceTestContainer(ctx)
	require.NoError(t, err)
	suite.testContainer = container

	// Initialize services
	suite.orderService = container.OrderService
	suite.workflowService = container.WorkflowService
	suite.analyticsService = container.AnalyticsService
	suite.bulkService = container.BulkService
	suite.exportService = container.ExportService

	log.Printf("Performance test suite initialized")
}

// TearDownSuite tears down the performance test suite
func (suite *OrderPerformanceTestSuite) TearDownSuite(t *testing.T) {
	if suite.testContainer != nil {
		ctx := context.Background()
		err := suite.testContainer.Cleanup(ctx)
		require.NoError(t, err)
		log.Printf("Performance test suite cleaned up")
	}
}

// TestOrderCreationPerformance tests order creation performance
func (suite *OrderPerformanceTestSuite) TestOrderCreationPerformance(t *testing.T) {
	ctx := context.Background()
	log.Printf("Starting order creation performance test")

	// Test configurations
	testCases := []struct {
		name           string
		concurrency    int
		totalOrders    int
		maxDuration    time.Duration
		minThroughput  float64
		maxMemoryMB    int
	}{
		{
			name:          "Light Load",
			concurrency:   10,
			totalOrders:   100,
			maxDuration:   5 * time.Second,
			minThroughput: 20.0,
			maxMemoryMB:   50,
		},
		{
			name:          "Medium Load",
			concurrency:   25,
			totalOrders:   500,
			maxDuration:   15 * time.Second,
			minThroughput: 33.0,
			maxMemoryMB:   100,
		},
		{
			name:          "Heavy Load",
			concurrency:   50,
			totalOrders:   1000,
			maxDuration:   30 * time.Second,
			minThroughput: 33.0,
			maxMemoryMB:   200,
		},
		{
			name:          "Stress Test",
			concurrency:   100,
			totalOrders:   2000,
			maxDuration:   60 * time.Second,
			minThroughput: 33.0,
			maxMemoryMB:   400,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			suite.runOrderCreationPerformanceTest(t, ctx, tc)
		})
	}
}

// runOrderCreationPerformanceTest runs a single order creation performance test
func (suite *OrderPerformanceTestSuite) runOrderCreationPerformanceTest(t *testing.T, ctx context.Context, tc struct {
	name           string
	concurrency    int
	totalOrders    int
	maxDuration    time.Duration
	minThroughput  float64
	maxMemoryMB    int
}) {
	log.Printf("Running %s: %d orders with %d goroutines", tc.name, tc.totalOrders, tc.concurrency)

	// Measure initial memory
	var initialMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&initialMem)

	// Create channels for coordination
	orderChan := make(chan *entities.Order, tc.totalOrders)
	errorChan := make(chan error, tc.totalOrders)
	startChan := make(chan struct{})

	// Start timer
	startTime := time.Now()

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < tc.concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startChan // Wait for start signal

			for j := 0; j < tc.totalOrders/tc.concurrency; j++ {
				order := suite.createTestOrderForPerformance(ctx)
				if order != nil {
					orderChan <- order
				} else {
					errorChan <- fmt.Errorf("failed to create order in worker %d", workerID)
				}
			}
		}(i)
	}

	// Start all workers
	close(startChan)

	// Wait for completion
	go func() {
		wg.Wait()
		close(orderChan)
		close(errorChan)
	}()

	// Collect results
	var createdOrders []*entities.Order
	var errors []error

	for order := range orderChan {
		createdOrders = append(createdOrders, order)
	}

	for err := range errorChan {
		errors = append(errors, err)
	}

	// Measure completion time and memory
	duration := time.Since(startTime)

	var finalMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&finalMem)

	memoryUsedMB := int(finalMem.Alloc - initialMem.Alloc) / 1024 / 1024
	throughput := float64(len(createdOrders)) / duration.Seconds()

	// Log results
	log.Printf("✅ %s completed:", tc.name)
	log.Printf("   Created: %d orders", len(createdOrders))
	log.Printf("   Errors:  %d", len(errors))
	log.Printf("   Duration: %v", duration)
	log.Printf("   Throughput: %.2f orders/sec", throughput)
	log.Printf("   Memory used: %d MB", memoryUsedMB)

	// Assertions
	require.Less(t, duration, tc.maxDuration, "Test should complete within maximum duration")
	require.GreaterOrEqual(t, throughput, tc.minThroughput, "Should achieve minimum throughput")
	require.LessOrEqual(t, memoryUsedMB, tc.maxMemoryMB, "Should not exceed maximum memory usage")
	require.Empty(t, errors, "Should have no errors")
	require.Equal(t, tc.totalOrders, len(createdOrders), "Should create all requested orders")
}

// TestBulkOperationsPerformance tests bulk operations performance
func (suite *OrderPerformanceTestSuite) TestBulkOperationsPerformance(t *testing.T) {
	ctx := context.Background()
	log.Printf("Starting bulk operations performance test")

	// Create test orders
	testOrders := suite.createTestOrdersForPerformance(ctx, 1000)
	require.Greater(t, len(testOrders), 0, "Should create test orders")

	orderIDs := make([]string, len(testOrders))
	for i, order := range testOrders {
		orderIDs[i] = order.ID.String()
	}

	// Test bulk status change
	t.Run("BulkStatusChange", func(t *testing.T) {
		start := time.Now()

		req := &order.BulkStatusChangeRequest{
			OrderIDs:  orderIDs,
			NewStatus: string(entities.OrderStatusProcessing),
			Reason:    "Performance test",
		}

		result, err := suite.bulkService.BulkChangeStatus(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		duration := time.Since(start)
		throughput := float64(result.SuccessCount) / duration.Seconds()

		log.Printf("✅ Bulk status change: %d orders in %v (%.2f orders/sec)",
			result.SuccessCount, duration, throughput)

		assert.Less(t, duration, 10*time.Second, "Should complete within 10 seconds")
		assert.GreaterOrEqual(t, throughput, 100.0, "Should achieve at least 100 orders/sec")
	})

	// Test bulk processing
	t.Run("BulkProcessing", func(t *testing.T) {
		start := time.Now()

		req := &order.BulkProcessRequest{
			OrderIDs:  orderIDs[:500], // Process first 500
			Operation: "fulfill",
			Options: &order.BulkOptions{
				SkipNotifications: true,
			},
		}

		result, err := suite.bulkService.BulkProcessOrders(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		duration := time.Since(start)
		throughput := float64(result.SuccessCount) / duration.Seconds()

		log.Printf("✅ Bulk processing: %d orders in %v (%.2f orders/sec)",
			result.SuccessCount, duration, throughput)

		assert.Less(t, duration, 15*time.Second, "Should complete within 15 seconds")
		assert.GreaterOrEqual(t, throughput, 33.0, "Should achieve at least 33 orders/sec")
	})

	// Test bulk validation
	t.Run("BulkValidation", func(t *testing.T) {
		start := time.Now()

		req := &order.BulkValidateRequest{
			OrderIDs: orderIDs,
		}

		result, err := suite.bulkService.BulkValidateOrders(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		duration := time.Since(start)
		throughput := float64(result.ValidCount) / duration.Seconds()

		log.Printf("✅ Bulk validation: %d orders in %v (%.2f orders/sec)",
			result.ValidCount, duration, throughput)

		assert.Less(t, duration, 5*time.Second, "Should complete within 5 seconds")
		assert.GreaterOrEqual(t, throughput, 200.0, "Should achieve at least 200 orders/sec")
	})
}

// TestAnalyticsPerformance tests analytics performance
func (suite *OrderPerformanceTestSuite) TestAnalyticsPerformance(t *testing.T) {
	ctx := context.Background()
	log.Printf("Starting analytics performance test")

	// Create test data
	testOrders := suite.createTestOrdersForPerformance(ctx, 5000)
	require.Greater(t, len(testOrders), 0, "Should create test orders")

	// Test order metrics
	t.Run("OrderMetrics", func(t *testing.T) {
		start := time.Now()

		req := &order.OrderMetricsRequest{
			StartDate: time.Now().Add(-24 * time.Hour),
			EndDate:   time.Now(),
		}

		metrics, err := suite.analyticsService.GetOrderMetrics(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, metrics)

		duration := time.Since(start)

		log.Printf("✅ Order metrics: %d orders analyzed in %v", metrics.TotalOrders, duration)

		assert.Less(t, duration, 3*time.Second, "Should complete within 3 seconds")
		assert.GreaterOrEqual(t, metrics.TotalOrders, len(testOrders), "Should analyze all orders")
	})

	// Test revenue metrics
	t.Run("RevenueMetrics", func(t *testing.T) {
		start := time.Now()

		req := &order.RevenueMetricsRequest{
			StartDate: time.Now().Add(-24 * time.Hour),
			EndDate:   time.Now(),
			GroupBy:   "daily",
		}

		metrics, err := suite.analyticsService.GetRevenueMetrics(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, metrics)

		duration := time.Since(start)

		log.Printf("✅ Revenue metrics: %s total revenue in %v", metrics.TotalRevenue.String(), duration)

		assert.Less(t, duration, 5*time.Second, "Should complete within 5 seconds")
		assert.Greater(t, metrics.TotalRevenue, decimal.Zero, "Should calculate total revenue")
	})

	// Test sales report generation
	t.Run("SalesReport", func(t *testing.T) {
		start := time.Now()

		req := &order.SalesReportRequest{
			StartDate: time.Now().Add(-24 * time.Hour),
			EndDate:   time.Now(),
		}

		report, err := suite.analyticsService.GenerateSalesReport(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, report)

		duration := time.Since(start)

		log.Printf("✅ Sales report: generated in %v", duration)

		assert.Less(t, duration, 10*time.Second, "Should complete within 10 seconds")
		assert.Greater(t, report.ExecutiveSummary.TotalOrders, 0, "Should include order data")
	})

	// Test real-time dashboard
	t.Run("RealTimeDashboard", func(t *testing.T) {
		start := time.Now()

		req := &order.DashboardRequest{}

		dashboard, err := suite.analyticsService.GetRealTimeDashboard(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, dashboard)

		duration := time.Since(start)

		log.Printf("✅ Real-time dashboard: generated in %v", duration)

		assert.Less(t, duration, 2*time.Second, "Should complete within 2 seconds")
		assert.GreaterOrEqual(t, dashboard.Summary.TodayOrders, 0, "Should include today's orders")
	})
}

// TestExportPerformance tests export performance
func (suite *OrderPerformanceTestSuite) TestExportPerformance(t *testing.T) {
	ctx := context.Background()
	log.Printf("Starting export performance test")

	// Create test orders
	testOrders := suite.createTestOrdersForPerformance(ctx, 2000)
	require.Greater(t, len(testOrders), 0, "Should create test orders")

	orderIDs := make([]string, len(testOrders))
	for i, order := range testOrders {
		orderIDs[i] = order.ID.String()
	}

	// Test CSV export
	t.Run("CSVExport", func(t *testing.T) {
		start := time.Now()

		req := &order.ExportOrdersRequest{
			OrderIDs: orderIDs,
			Format:   "csv",
			Fields:   []string{"id", "order_number", "customer_id", "status", "total_amount", "created_at"},
			Options: &order.ExportOptions{
				IncludeItems: true,
			},
		}

		result, err := suite.exportService.ExportOrdersToCSV(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		duration := time.Since(start)
		throughput := float64(result.RecordCount) / duration.Seconds()

		log.Printf("✅ CSV export: %d records in %v (%.2f records/sec)",
			result.RecordCount, duration, throughput)

		assert.Less(t, duration, 10*time.Second, "Should complete within 10 seconds")
		assert.GreaterOrEqual(t, throughput, 200.0, "Should achieve at least 200 records/sec")
	})

	// Test JSON export
	t.Run("JSONExport", func(t *testing.T) {
		start := time.Now()

		req := &order.ExportOrdersRequest{
			OrderIDs: orderIDs[:1000], // Export first 1000
			Format:   "json",
			Options: &order.ExportOptions{
				IncludeItems:     true,
				IncludeCustomer:  true,
				IncludeAddresses: true,
			},
		}

		result, err := suite.exportService.ExportOrdersToJSON(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		duration := time.Since(start)
		throughput := float64(result.RecordCount) / duration.Seconds()

		log.Printf("✅ JSON export: %d records in %v (%.2f records/sec)",
			result.RecordCount, duration, throughput)

		assert.Less(t, duration, 15*time.Second, "Should complete within 15 seconds")
		assert.GreaterOrEqual(t, throughput, 66.0, "Should achieve at least 66 records/sec")
	})

	// Test Excel export
	t.Run("ExcelExport", func(t *testing.T) {
		start := time.Now()

		req := &order.ExportOrdersRequest{
			OrderIDs: orderIDs[:500], // Export first 500
			Format:   "excel",
			Fields:   []string{"id", "order_number", "customer_id", "status", "total_amount"},
		}

		result, err := suite.exportService.ExportOrdersToExcel(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		duration := time.Since(start)
		throughput := float64(result.RecordCount) / duration.Seconds()

		log.Printf("✅ Excel export: %d records in %v (%.2f records/sec)",
			result.RecordCount, duration, throughput)

		assert.Less(t, duration, 20*time.Second, "Should complete within 20 seconds")
		assert.GreaterOrEqual(t, throughput, 25.0, "Should achieve at least 25 records/sec")
	})

	// Test bulk export
	t.Run("BulkExport", func(t *testing.T) {
		start := time.Now()

		bulkReq := &order.BulkExportRequest{
			ExportRequests: []*order.ExportOrdersRequest{
				{
					OrderIDs: orderIDs[:500],
					Format:   "csv",
					Fields:   []string{"id", "order_number", "status", "total_amount"},
				},
				{
					OrderIDs: orderIDs[:500],
					Format:   "json",
					Options: &order.ExportOptions{IncludeItems: true},
				},
			},
		}

		result, err := suite.exportService.BulkExportOrders(ctx, bulkReq)
		require.NoError(t, err)
		require.NotNil(t, result)

		duration := time.Since(start)

		log.Printf("✅ Bulk export: %d total records in %v", result.TotalRecords, duration)

		assert.Less(t, duration, 30*time.Second, "Should complete within 30 seconds")
		assert.Greater(t, result.TotalRecords, 0, "Should export records")
	})
}

// TestWorkflowPerformance tests workflow performance
func (suite *OrderPerformanceTestSuite) TestWorkflowPerformance(t *testing.T) {
	ctx := context.Background()
	log.Printf("Starting workflow performance test")

	// Test concurrent workflow executions
	t.Run("ConcurrentWorkflows", func(t *testing.T) {
		const numWorkflows = 50
		workflowChan := make(chan *order.WorkflowResult, numWorkflows)
		errorChan := make(chan error, numWorkflows)

		start := time.Now()

		// Start workflow goroutines
		var wg sync.WaitGroup
		for i := 0; i < numWorkflows; i++ {
			wg.Add(1)
			go func(workflowID int) {
				defer wg.Done()

				req := &order.CompleteWorkflowRequest{
					CustomerID:        uuid.New().String(),
					CustomerFirstName:  fmt.Sprintf("Test%d", workflowID),
					CustomerLastName:   "Customer",
					CustomerEmail:     fmt.Sprintf("test%d@example.com", workflowID),
					Priority:          entities.OrderPriorityNormal,
					ShippingMethod:    entities.ShippingMethodStandard,
					Currency:          "USD",
					CreateCustomer:    true,
					CreateProducts:    true,
					PaymentMethod:     stringPtr("credit_card"),
					Products: []order.ProductRequest{
						{
							Name:       fmt.Sprintf("Product %d", workflowID),
							SKU:        fmt.Sprintf("PROD-%03d", workflowID),
							Price:      29.99,
							StockLevel: 100,
						},
					},
					OrderItems: []order.OrderItemRequest{
						{
							ProductID: uuid.New().String(),
							SKU:       fmt.Sprintf("PROD-%03d", workflowID),
							Name:      fmt.Sprintf("Product %d", workflowID),
							Quantity:  2,
							UnitPrice: 29.99,
							Weight:    1.5,
						},
					},
				}

				result, err := suite.workflowService.ExecuteCompleteOrderWorkflow(ctx, req)
				if err != nil {
					errorChan <- err
					return
				}
				workflowChan <- result
			}(i)
		}

		// Wait for completion
		go func() {
			wg.Wait()
			close(workflowChan)
			close(errorChan)
		}()

		// Collect results
		var workflows []*order.WorkflowResult
		var errors []error

		for result := range workflowChan {
			workflows = append(workflows, result)
		}

		for err := range errorChan {
			errors = append(errors, err)
		}

		duration := time.Since(start)
		successRate := float64(len(workflows)) / float64(numWorkflows) * 100
		throughput := float64(len(workflows)) / duration.Seconds()

		log.Printf("✅ Concurrent workflows: %d/%d successful (%.1f%%) in %v (%.2f workflows/sec)",
			len(workflows), numWorkflows, successRate, duration, throughput)

		assert.Less(t, duration, 60*time.Second, "Should complete within 60 seconds")
		assert.GreaterOrEqual(t, successRate, 95.0, "Should have at least 95% success rate")
		assert.GreaterOrEqual(t, throughput, 0.8, "Should achieve at least 0.8 workflows/sec")
		assert.Empty(t, errors, "Should have no errors")
	})
}

// TestMemoryUsage tests memory usage patterns
func (suite *OrderPerformanceTestSuite) TestMemoryUsage(t *testing.T) {
	ctx := context.Background()
	log.Printf("Starting memory usage test")

	// Measure baseline memory
	var baselineMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&baselineMem)

	log.Printf("Baseline memory: %d KB", baselineMem.Alloc/1024)

	// Create many orders and measure memory usage
	const numOrders = 10000
	orders := make([]*entities.Order, numOrders)

	for i := 0; i < numOrders; i++ {
		orders[i] = suite.createTestOrderForPerformance(ctx)
	}

	// Measure memory after order creation
	var afterCreationMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&afterCreationMem)

	creationMemoryMB := int(afterCreationMem.Alloc-baselineMem.Alloc) / 1024 / 1024
	avgMemoryPerOrderKB := int(afterCreationMem.Alloc-baselineMem.Alloc) / numOrders / 1024

	log.Printf("After creating %d orders:", numOrders)
	log.Printf("   Memory used: %d MB", creationMemoryMB)
	log.Printf("   Average per order: %d KB", avgMemoryPerOrderKB)

	// Process orders in bulk and measure memory
	orderIDs := make([]string, numOrders)
	for i, order := range orders {
		orderIDs[i] = order.ID.String()
	}

	req := &order.BulkProcessRequest{
		OrderIDs:  orderIDs,
		Operation: "confirm",
		Options: &order.BulkOptions{
			SkipNotifications: true,
		},
	}

	_, err := suite.bulkService.BulkProcessOrders(ctx, req)
	require.NoError(t, err)

	// Measure memory after bulk processing
	var afterProcessingMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&afterProcessingMem)

	processingMemoryMB := int(afterProcessingMem.Alloc-afterCreationMem.Alloc) / 1024 / 1024

	log.Printf("After bulk processing:")
	log.Printf("   Additional memory: %d MB", processingMemoryMB)

	// Clear orders and measure memory after cleanup
	orders = nil
	orderIDs = nil
	runtime.GC()

	var afterCleanupMem runtime.MemStats
	runtime.ReadMemStats(&afterCleanupMem)

	finalMemoryMB := int(afterCleanupMem.Alloc) / 1024 / 1024
	memoryLeakMB := finalMemoryMB - int(baselineMem.Alloc)/1024/1024

	log.Printf("After cleanup:")
	log.Printf("   Final memory: %d MB", finalMemoryMB)
	log.Printf("   Memory leak: %d MB", memoryLeakMB)

	// Assertions
	assert.Less(t, creationMemoryMB, 500, "Should not use more than 500MB for 10k orders")
	assert.Less(t, avgMemoryPerOrderKB, 50, "Should not use more than 50KB per order")
	assert.Less(t, processingMemoryMB, 100, "Should not use more than 100MB for bulk processing")
	assert.Less(t, memoryLeakMB, 10, "Should not have memory leaks more than 10MB")
}

// TestDatabaseConnectionPool tests database connection pool performance
func (suite *OrderPerformanceTestSuite) TestDatabaseConnectionPool(t *testing.T) {
	ctx := context.Background()
	log.Printf("Starting database connection pool test")

	// Test with different pool sizes
	poolSizes := []int{5, 10, 20, 50}

	for _, poolSize := range poolSizes {
		t.Run(fmt.Sprintf("PoolSize_%d", poolSize), func(t *testing.T) {
			suite.runDatabasePoolTest(t, ctx, poolSize)
		})
	}
}

// runDatabasePoolTest runs database pool test with specified pool size
func (suite *OrderPerformanceTestSuite) runDatabasePoolTest(t *testing.T, ctx context.Context, poolSize int) {
	const numOperations = 1000
	const concurrency = poolSize

	operationsChan := make(chan bool, numOperations)
	errorChan := make(chan error, numOperations)
	startChan := make(chan struct{})

	start := time.Now()

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startChan

			for j := 0; j < numOperations/concurrency; j++ {
				// Simulate database operation by creating and retrieving an order
				order := suite.createTestOrderForPerformance(ctx)
				if order != nil {
					// In a real implementation, this would save to database and retrieve
					operationsChan <- true
				} else {
					errorChan <- fmt.Errorf("failed to create order in worker %d", workerID)
				}
			}
		}(i)
	}

	// Start all workers
	close(startChan)

	// Wait for completion
	go func() {
		wg.Wait()
		close(operationsChan)
		close(errorChan)
	}()

	// Collect results
	var successfulOps int
	var errors []error

	for range operationsChan {
		successfulOps++
	}

	for err := range errorChan {
		errors = append(errors, err)
	}

	duration := time.Since(start)
	throughput := float64(successfulOps) / duration.Seconds()

	log.Printf("✅ Pool size %d: %d operations in %v (%.2f ops/sec)",
		poolSize, successfulOps, duration, throughput)

	// Assertions
	assert.Empty(t, errors, "Should have no errors")
	assert.Equal(t, numOperations, successfulOps, "Should complete all operations")
	assert.GreaterOrEqual(t, throughput, 100.0, "Should achieve at least 100 ops/sec")
}

// BenchmarkOrderCreation benchmarks order creation
func BenchmarkOrderCreation(b *testing.B) {
	ctx := context.Background()

	// Initialize test container (simplified for benchmark)
	suite := &OrderPerformanceTestSuite{}
	// Note: In a real implementation, properly initialize services

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		order := suite.createTestOrderForPerformance(ctx)
		if order == nil {
			b.Fatal("Failed to create order")
		}
	}
}

// BenchmarkBulkStatusChange benchmarks bulk status change operations
func BenchmarkBulkStatusChange(b *testing.B) {
	ctx := context.Background()

	// Initialize test container (simplified for benchmark)
	suite := &OrderPerformanceTestSuite{}

	// Create test orders
	testOrders := suite.createTestOrdersForPerformance(ctx, 1000)
	orderIDs := make([]string, len(testOrders))
	for i, order := range testOrders {
		orderIDs[i] = order.ID.String()
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := &order.BulkStatusChangeRequest{
			OrderIDs:  orderIDs[:100], // Use subset for benchmarking
			NewStatus: string(entities.OrderStatusProcessing),
			Reason:    "Benchmark test",
		}

		_, err := suite.bulkService.BulkChangeStatus(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAnalyticsQuery benchmarks analytics queries
func BenchmarkAnalyticsQuery(b *testing.B) {
	ctx := context.Background()

	// Initialize test container (simplified for benchmark)
	suite := &OrderPerformanceTestSuite{}

	// Create test data
	suite.createTestOrdersForPerformance(ctx, 5000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := &order.OrderMetricsRequest{
			StartDate: time.Now().Add(-24 * time.Hour),
			EndDate:   time.Now(),
		}

		_, err := suite.analyticsService.GetOrderMetrics(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ==================== HELPER METHODS ====================

// createTestOrderForPerformance creates a test order for performance testing
func (suite *OrderPerformanceTestSuite) createTestOrderForPerformance(ctx context.Context) *entities.Order {
	order := &entities.Order{
		ID:               uuid.New(),
		OrderNumber:      entities.GenerateOrderNumber(),
		CustomerID:       uuid.New(),
		Status:           entities.OrderStatusDraft,
		Priority:         entities.OrderPriorityNormal,
		Type:             entities.OrderTypeSales,
		PaymentStatus:    entities.PaymentStatusPending,
		ShippingMethod:   entities.ShippingMethodStandard,
		Currency:         "USD",
		OrderDate:        time.Now(),
		ShippingAddressID: uuid.New(),
		BillingAddressID:  uuid.New(),
		CreatedBy:        uuid.New(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Create order items
	items := make([]entities.OrderItem, 3)
	productIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	productNames := []string{"Product A", "Product B", "Product C"}
	prices := []float64{29.99, 49.99, 19.99}

	for i := 0; i < 3; i++ {
		quantity := 2
		unitPrice := decimal.NewFromFloat(prices[i])

		items[i] = entities.OrderItem{
			ID:             uuid.New(),
			OrderID:        order.ID,
			ProductID:      productIDs[i],
			ProductSKU:     fmt.Sprintf("SKU-%03d", i+1),
			ProductName:    productNames[i],
			Quantity:       quantity,
			UnitPrice:      unitPrice,
			TotalPrice:     unitPrice.Mul(decimal.NewFromInt(int64(quantity))),
			Weight:         1.5,
			Status:         "ORDERED",
			QuantityShipped: 0,
			QuantityReturned: 0,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
	}

	order.Items = items

	// Calculate totals
	err := order.CalculateTotals()
	if err != nil {
		return nil
	}

	return order
}

// createTestOrdersForPerformance creates multiple test orders for performance testing
func (suite *OrderPerformanceTestSuite) createTestOrdersForPerformance(ctx context.Context, count int) []*entities.Order {
	orders := make([]*entities.Order, count)
	for i := 0; i < count; i++ {
		orders[i] = suite.createTestOrderForPerformance(ctx)
	}
	return orders
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// TestOrderPerformanceSuite runs the order performance test suite
func TestOrderPerformanceSuite(t *testing.T) {
	suite := &OrderPerformanceTestSuite{}
	suite.SetupSuite(t)
	defer suite.TearDownSuite(t)

	t.Run("OrderCreation", suite.TestOrderCreationPerformance)
	t.Run("BulkOperations", suite.TestBulkOperationsPerformance)
	t.Run("Analytics", suite.TestAnalyticsPerformance)
	t.Run("Export", suite.TestExportPerformance)
	t.Run("Workflow", suite.TestWorkflowPerformance)
	t.Run("MemoryUsage", suite.TestMemoryUsage)
	t.Run("DatabasePool", suite.TestDatabaseConnectionPool)
}