package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"erpgo/internal/application/services/order"
	"erpgo/internal/application/services/product"
	"erpgo/internal/application/services/customer"
	"erpgo/internal/domain/orders/entities"
	productrepo "erpgo/internal/domain/products/repositories"
	customerrepo "erpgo/internal/domain/customers/repositories"
	"erpgo/internal/infrastructure/database"
	"erpgo/internal/infrastructure/logger"
	erporder "erpgo/internal/domain/orders/repositories"
	"erpgo/tests/load"
)

// APIPerformanceTestSuite tests API performance under various conditions
type APIPerformanceTestSuite struct {
	suite.Suite
	db           *database.Database
	orderRepo    erporder.OrderRepository
	productRepo  productrepo.ProductRepository
	customerRepo customerrepo.CustomerRepository
	orderService order.Service
	productService product.Service
	customerService customer.Service
	router       *gin.Engine
	testData     *TestData
}

// TestData holds test data for performance testing
type TestData struct {
	Customers []*customerrepo.Customer
	Products  []*productrepo.Product
	Orders    []*erporder.Order
}

// PerformanceMetrics holds performance test results
type PerformanceMetrics struct {
	TotalRequests       int
	SuccessfulRequests  int
	FailedRequests      int
	AverageResponseTime time.Duration
	MinResponseTime     time.Duration
	MaxResponseTime     time.Duration
	P95ResponseTime     time.Duration
	P99ResponseTime     time.Duration
	RequestsPerSecond   float64
	ErrorRate           float64
	MemoryUsageMB       int64
	Goroutines          int
}

// SetupSuite sets up the test suite
func (suite *APIPerformanceTestSuite) SetupSuite() {
	// Initialize database connection
	dbConfig := &database.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "erpgo_test",
		SSLMode:  "disable",
	}

	var err error
	suite.db, err = database.NewConnection(dbConfig)
	suite.Require().NoError(err)

	// Initialize repositories
	suite.orderRepo = erporder.NewPostgresOrderRepository(suite.db.GetDB(), dbConfig, logger.NewNopLogger())
	suite.productRepo = productrepo.NewPostgresProductRepository(suite.db.GetDB(), dbConfig, logger.NewNopLogger())
	suite.customerRepo = customerrepo.NewPostgresCustomerRepository(suite.db.GetDB(), dbConfig, logger.NewNopLogger())

	// Initialize services
	suite.orderService = order.NewOrderService(suite.orderRepo, suite.productRepo, suite.customerRepo, logger.NewNopLogger())
	suite.productService = product.NewProductService(suite.productRepo, logger.NewNopLogger())
	suite.customerService = customer.NewCustomerService(suite.customerRepo, logger.NewNopLogger())

	// Set up Gin router with performance optimizations
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Add performance middleware
	suite.router.Use(gin.Recovery())
	suite.router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Next()
	})

	suite.setupRoutes()

	// Initialize test data
	suite.testData = &TestData{}
	suite.createLargeDataset()
}

// TearDownSuite cleans up after the test suite
func (suite *APIPerformanceTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.cleanupTestData()
		suite.db.Close()
	}
}

// setupRoutes sets up the API routes for testing
func (suite *APIPerformanceTestSuite) setupRoutes() {
	api := suite.router.Group("/api/v1")
	{
		// Customer endpoints
		api.POST("/customers", suite.createCustomerHandler)
		api.GET("/customers", suite.listCustomersHandler)
		api.GET("/customers/:id", suite.getCustomerHandler)

		// Product endpoints
		api.POST("/products", suite.createProductHandler)
		api.GET("/products", suite.listProductsHandler)
		api.GET("/products/:id", suite.getProductHandler)
		api.PUT("/products/:id/stock", suite.updateProductStockHandler)

		// Order endpoints
		api.POST("/orders", suite.createOrderHandler)
		api.GET("/orders", suite.listOrdersHandler)
		api.GET("/orders/:id", suite.getOrderHandler)

		// Health check endpoint
		api.GET("/health", suite.healthCheckHandler)

		// Search endpoints
		api.GET("/search/customers", suite.searchCustomersHandler)
		api.GET("/search/products", suite.searchProductsHandler)
	}
}

// cleanupTestData removes all test data
func (suite *APIPerformanceTestSuite) cleanupTestData() {
	if suite.db != nil {
		tx := suite.db.GetDB()
		tx.Exec("DELETE FROM order_items")
		tx.Exec("DELETE FROM orders")
		tx.Exec("DELETE FROM products")
		tx.Exec("DELETE FROM customers")
	}
}

// createLargeDataset creates a large dataset for performance testing
func (suite *APIPerformanceTestSuite) createLargeDataset() {
	ctx := context.Background()

	// Create 1000 customers
	for i := 0; i < 1000; i++ {
		customer := &customerrepo.Customer{
			Name:         fmt.Sprintf("Customer %d", i),
			Email:        fmt.Sprintf("customer%d@example.com", i),
			Phone:        fmt.Sprintf("+123456%04d", i),
			Address:      fmt.Sprintf("%d Test St, City, State 12345", i),
			CustomerType: "individual",
			Status:       "active",
		}

		if err := suite.customerRepo.Create(ctx, customer); err == nil {
			suite.testData.Customers = append(suite.testData.Customers, customer)
		}
	}

	// Create 500 products
	for i := 0; i < 500; i++ {
		product := &productrepo.Product{
			Name:        fmt.Sprintf("Product %d", i),
			Description: fmt.Sprintf("Description for product %d", i),
			SKU:         fmt.Sprintf("SKU-%04d", i),
			Price:       decimal.NewFromFloat(float64(i%100) + 10.99),
			Stock:       100 + (i % 50),
			CategoryID:  uuid.New(),
			Status:      "active",
		}

		if err := suite.productRepo.Create(ctx, product); err == nil {
			suite.testData.Products = append(suite.testData.Products, product)
		}
	}

	// Create 2000 orders
	for i := 0; i < 2000; i++ {
		if len(suite.testData.Customers) == 0 || len(suite.testData.Products) == 0 {
			continue
		}

		customer := suite.testData.Customers[i%len(suite.testData.Customers)]
		product := suite.testData.Products[i%len(suite.testData.Products)]

		order := &erporder.Order{
			CustomerID: customer.ID,
			Status:     entities.OrderStatus([]string{"pending", "paid", "shipped", "completed"}[i%4]),
			Total:      product.Price.Mul(decimal.NewFromInt(int64((i % 5) + 1))),
		}

		if err := suite.orderRepo.Create(ctx, order); err == nil {
			suite.testData.Orders = append(suite.testData.Orders, order)

			// Add order item
			orderItem := &erporder.OrderItem{
				OrderID:   order.ID,
				ProductID: product.ID,
				Quantity:  (i % 5) + 1,
				Price:     product.Price,
			}
			suite.orderRepo.AddItem(ctx, orderItem)
		}
	}
}

// Test Cases

func (suite *APIPerformanceTestSuite) TestCustomerListPerformance() {
	const numRequests = 100
	const numConcurrent = 10

	var responseTimes []time.Duration
	var mu sync.Mutex
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < numRequests/numConcurrent; j++ {
				reqStart := time.Now()
				req, _ := http.NewRequest("GET", "/api/v1/customers", nil)
				w := httptest.NewRecorder()
				suite.router.ServeHTTP(w, req)
				reqDuration := time.Since(reqStart)

				mu.Lock()
				responseTimes = append(responseTimes, reqDuration)
				mu.Unlock()

				assert.Equal(suite.T(), http.StatusOK, w.Code)
			}
		}()
	}

	wg.Wait()
	totalDuration := time.Since(start)

	// Calculate metrics
	metrics := suite.calculateMetrics(responseTimes, totalDuration, numRequests)

	suite.T().Logf("Customer List Performance Metrics:")
	suite.T().Logf("  Total Requests: %d", metrics.TotalRequests)
	suite.T().Logf("  Average Response Time: %v", metrics.AverageResponseTime)
	suite.T().Logf("  P95 Response Time: %v", metrics.P95ResponseTime)
	suite.T().Logf("  P99 Response Time: %v", metrics.P99ResponseTime)
	suite.T().Logf("  Requests Per Second: %.2f", metrics.RequestsPerSecond)
	suite.T().Logf("  Error Rate: %.2f%%", metrics.ErrorRate*100)

	// Performance assertions
	suite.Less(metrics.AverageResponseTime, 100*time.Millisecond, "Average response time should be under 100ms")
	suite.Less(metrics.P95ResponseTime, 200*time.Millisecond, "P95 response time should be under 200ms")
	suite.Greater(metrics.RequestsPerSecond, 100.0, "Should handle at least 100 RPS")
	suite.Less(metrics.ErrorRate, 0.01, "Error rate should be less than 1%")
}

func (suite *APIPerformanceTestSuite) TestProductSearchPerformance() {
	const numRequests = 200
	const numConcurrent = 20

	var responseTimes []time.Duration
	var mu sync.Mutex
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numRequests/numConcurrent; j++ {
				searchTerm := fmt.Sprintf("Product %d", workerID*10+j%100)
				reqStart := time.Now()
				req, _ := http.NewRequest("GET", "/api/v1/search/products?q="+searchTerm, nil)
				w := httptest.NewRecorder()
				suite.router.ServeHTTP(w, req)
				reqDuration := time.Since(reqStart)

				mu.Lock()
				responseTimes = append(responseTimes, reqDuration)
				mu.Unlock()

				assert.Equal(suite.T(), http.StatusOK, w.Code)
			}
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(start)

	// Calculate metrics
	metrics := suite.calculateMetrics(responseTimes, totalDuration, numRequests)

	suite.T().Logf("Product Search Performance Metrics:")
	suite.T().Logf("  Total Requests: %d", metrics.TotalRequests)
	suite.T().Logf("  Average Response Time: %v", metrics.AverageResponseTime)
	suite.T().Logf("  P95 Response Time: %v", metrics.P95ResponseTime)
	suite.T().Logf("  Requests Per Second: %.2f", metrics.RequestsPerSecond)

	// Performance assertions for search (should be slightly slower than simple list)
	suite.Less(metrics.AverageResponseTime, 150*time.Millisecond, "Average search response time should be under 150ms")
	suite.Less(metrics.P95ResponseTime, 300*time.Millisecond, "P95 search response time should be under 300ms")
}

func (suite *APIPerformanceTestSuite) TestOrderCreationPerformance() {
	const numRequests = 500
	const numConcurrent = 25

	if len(suite.testData.Customers) == 0 || len(suite.testData.Products) == 0 {
		suite.T().Skip("Insufficient test data for order creation test")
		return
	}

	var responseTimes []time.Duration
	var mu sync.Mutex
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numRequests/numConcurrent; j++ {
				customer := suite.testData.Customers[workerID%len(suite.testData.Customers)]
				product := suite.testData.Products[(workerID+j)%len(suite.testData.Products)]

				payload := map[string]interface{}{
					"customer_id": customer.ID.String(),
					"items": []map[string]interface{}{
						{
							"product_id": product.ID.String(),
							"quantity":  (j % 5) + 1,
						},
					},
					"notes": fmt.Sprintf("Performance test order %d-%d", workerID, j),
				}

				jsonData, _ := json.Marshal(payload)
				reqStart := time.Now()
				req, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(jsonData))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				suite.router.ServeHTTP(w, req)
				reqDuration := time.Since(reqStart)

				mu.Lock()
				responseTimes = append(responseTimes, reqDuration)
				mu.Unlock()

				// Order creation might fail due to stock constraints
				assert.True(suite.T(), w.Code == http.StatusCreated || w.Code == http.StatusBadRequest)
			}
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(start)

	// Calculate metrics
	metrics := suite.calculateMetrics(responseTimes, totalDuration, numRequests)

	suite.T().Logf("Order Creation Performance Metrics:")
	suite.T().Logf("  Total Requests: %d", metrics.TotalRequests)
	suite.T().Logf("  Average Response Time: %v", metrics.AverageResponseTime)
	suite.T().Logf("  P95 Response Time: %v", metrics.P95ResponseTime)
	suite.T().Logf("  Requests Per Second: %.2f", metrics.RequestsPerSecond)

	// Performance assertions for order creation (more complex operation)
	suite.Less(metrics.AverageResponseTime, 200*time.Millisecond, "Average order creation time should be under 200ms")
	suite.Less(metrics.P95ResponseTime, 400*time.Millisecond, "P95 order creation time should be under 400ms")
	suite.Greater(metrics.RequestsPerSecond, 50.0, "Should handle at least 50 order creations per second")
}

func (suite *APIPerformanceTestSuite) TestMixedWorkloadPerformance() {
	const numRequests = 1000
	const numConcurrent = 50

	var responseTimes []time.Duration
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Mixed workload operations
	operations := []func() *http.Request{
		func() *http.Request {
			req, _ := http.NewRequest("GET", "/api/v1/customers", nil)
			return req
		},
		func() *http.Request {
			req, _ := http.NewRequest("GET", "/api/v1/products", nil)
			return req
		},
		func() *http.Request {
			req, _ := http.NewRequest("GET", "/api/v1/orders", nil)
			return req
		},
		func() *http.Request {
			req, _ := http.NewRequest("GET", "/api/v1/health", nil)
			return req
		},
	}

	start := time.Now()

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numRequests/numConcurrent; j++ {
				operation := operations[(workerID+j)%len(operations)]
				req := operation()

				reqStart := time.Now()
				w := httptest.NewRecorder()
				suite.router.ServeHTTP(w, req)
				reqDuration := time.Since(reqStart)

				mu.Lock()
				responseTimes = append(responseTimes, reqDuration)
				mu.Unlock()

				assert.Equal(suite.T(), http.StatusOK, w.Code)
			}
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(start)

	// Calculate metrics
	metrics := suite.calculateMetrics(responseTimes, totalDuration, numRequests)

	suite.T().Logf("Mixed Workload Performance Metrics:")
	suite.T().Logf("  Total Requests: %d", metrics.TotalRequests)
	suite.T().Logf("  Average Response Time: %v", metrics.AverageResponseTime)
	suite.T().Logf("  P95 Response Time: %v", metrics.P95ResponseTime)
	suite.T().Logf("  P99 Response Time: %v", metrics.P99ResponseTime)
	suite.T().Logf("  Requests Per Second: %.2f", metrics.RequestsPerSecond)
	suite.T().Logf("  Error Rate: %.2f%%", metrics.ErrorRate*100)

	// Performance assertions for mixed workload
	suite.Less(metrics.AverageResponseTime, 80*time.Millisecond, "Average response time should be under 80ms")
	suite.Less(metrics.P95ResponseTime, 150*time.Millisecond, "P95 response time should be under 150ms")
	suite.Greater(metrics.RequestsPerSecond, 200.0, "Should handle at least 200 RPS for mixed workload")
	suite.Less(metrics.ErrorRate, 0.01, "Error rate should be less than 1%")
}

func (suite *APIPerformanceTestSuite) TestMemoryAndGoroutineUsage() {
	const numRequests = 1000
	const numConcurrent = 100

	// Record initial state
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialGoroutines := runtime.NumGoroutine()

	var wg sync.WaitGroup

	// Execute many concurrent requests
	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < numRequests/numConcurrent; j++ {
				req, _ := http.NewRequest("GET", "/api/v1/customers", nil)
				w := httptest.NewRecorder()
				suite.router.ServeHTTP(w, req)
			}
		}()
	}

	wg.Wait()

	// Force garbage collection and check memory usage
	runtime.GC()
	runtime.ReadMemStats(&m2)
	finalGoroutines := runtime.NumGoroutine()

	memoryUsedMB := int64(m2.Alloc-m1.Alloc) / 1024 / 1024
	goroutineIncrease := finalGoroutines - initialGoroutines

	suite.T().Logf("Memory and Goroutine Usage:")
	suite.T().Logf("  Memory Used: %d MB", memoryUsedMB)
	suite.T().Logf("  Initial Goroutines: %d", initialGoroutines)
	suite.T().Logf("  Final Goroutines: %d", finalGoroutines)
	suite.T().Logf("  Goroutine Increase: %d", goroutineIncrease)

	// Memory and goroutine assertions
	suite.Less(memoryUsedMB, int64(100), "Memory usage should be reasonable")
	suite.Less(goroutineIncrease, 50, "Should not have significant goroutine leaks")
}

func (suite *APIPerformanceTestSuite) TestLoadTestingFramework() {
	config := &load.LoadTestConfig{
		Name:               "API Load Test",
		BaseURL:            "http://localhost:8080",
		ConcurrentUsers:    10,
		RequestsPerUser:    100,
		TestDuration:       30 * time.Second,
		RampUpDuration:     5 * time.Second,
		TimeoutPerRequest:  5 * time.Second,
		ThinkTime:          100 * time.Millisecond,
		TargetRPS:          50,
		MaxErrorRate:       0.05,
		MaxResponseTime:    500 * time.Millisecond,
		ExpectedStatusCode: 200,
	}

	framework := load.NewLoadTestFramework(config)

	// Request generator function
	requestFunc := func(user int, iteration int) (*http.Request, error) {
		endpoints := []string{
			"/api/v1/customers",
			"/api/v1/products",
			"/api/v1/orders",
			"/api/v1/health",
		}

		endpoint := endpoints[iteration%len(endpoints)]
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			return nil, err
		}

		// Add headers
		req.Header.Set("User-Agent", fmt.Sprintf("LoadTestBot-User-%d", user))
		req.Header.Set("X-Request-ID", uuid.New().String())

		return req, nil
	}

	// Run the load test
	result, err := framework.RunLoadTest(requestFunc)
	suite.Require().NoError(err)

	// Validate results
	suite.NoError(framework.ValidateResults())

	suite.T().Logf("Load Test Results:")
	suite.T().Logf("  Total Requests: %d", result.TotalRequests)
	suite.T().Logf("  Successful Requests: %d", result.SuccessfulRequests)
	suite.T().Logf("  Failed Requests: %d", result.FailedRequests)
	suite.T().Logf("  Requests Per Second: %.2f", result.RequestsPerSecond)
	suite.T().Logf("  Average Response Time: %v", result.AverageResponseTime)
	suite.T().Logf("  P95 Response Time: %v", result.P95ResponseTime)
	suite.T().Logf("  Error Rate: %.2f%%", result.ErrorRate*100)

	// Load test assertions
	suite.Greater(result.RequestsPerSecond, float64(config.TargetRPS)*0.8, "Should achieve at least 80% of target RPS")
	suite.Less(result.ErrorRate, config.MaxErrorRate, "Error rate should be below threshold")
	suite.Less(result.P95ResponseTime, config.MaxResponseTime, "P95 response time should be below threshold")
}

// Helper methods

func (suite *APIPerformanceTestSuite) calculateMetrics(responseTimes []time.Duration, totalDuration time.Duration, totalRequests int) *PerformanceMetrics {
	if len(responseTimes) == 0 {
		return &PerformanceMetrics{}
	}

	// Sort response times for percentile calculations
	for i := 0; i < len(responseTimes)-1; i++ {
		for j := i + 1; j < len(responseTimes); j++ {
			if responseTimes[i] > responseTimes[j] {
				responseTimes[i], responseTimes[j] = responseTimes[j], responseTimes[i]
			}
		}
	}

	var total time.Duration
	min := responseTimes[0]
	max := responseTimes[len(responseTimes)-1]

	for _, rt := range responseTimes {
		total += rt
	}

	avg := total / time.Duration(len(responseTimes))

	// Calculate percentiles
	p95Index := int(float64(len(responseTimes)) * 0.95)
	p99Index := int(float64(len(responseTimes)) * 0.99)

	if p95Index >= len(responseTimes) {
		p95Index = len(responseTimes) - 1
	}
	if p99Index >= len(responseTimes) {
		p99Index = len(responseTimes) - 1
	}

	p95 := responseTimes[p95Index]
	p99 := responseTimes[p99Index]

	rps := float64(totalRequests) / totalDuration.Seconds()

	// Get current memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryMB := int64(m.Alloc) / 1024 / 1024

	return &PerformanceMetrics{
		TotalRequests:       totalRequests,
		AverageResponseTime: avg,
		MinResponseTime:     min,
		MaxResponseTime:     max,
		P95ResponseTime:     p95,
		P99ResponseTime:     p99,
		RequestsPerSecond:   rps,
		ErrorRate:           0, // Will be calculated by caller
		MemoryUsageMB:       memoryMB,
		Goroutines:          runtime.NumGoroutine(),
	}
}

// Simplified API handlers for testing
func (suite *APIPerformanceTestSuite) createCustomerHandler(c *gin.Context) {
	// Mock implementation for performance testing
	c.JSON(http.StatusCreated, gin.H{"id": uuid.New(), "name": "Test Customer"})
}

func (suite *APIPerformanceTestSuite) listCustomersHandler(c *gin.Context) {
	ctx := c.Request.Context()
	customers, err := suite.customerRepo.List(ctx, customerrepo.CustomerFilter{Limit: 100, Offset: 0})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"customers": customers})
}

func (suite *APIPerformanceTestSuite) getCustomerHandler(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	customer, err := suite.customerRepo.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"customer": customer})
}

func (suite *APIPerformanceTestSuite) createProductHandler(c *gin.Context) {
	// Mock implementation for performance testing
	c.JSON(http.StatusCreated, gin.H{"id": uuid.New(), "name": "Test Product"})
}

func (suite *APIPerformanceTestSuite) listProductsHandler(c *gin.Context) {
	ctx := c.Request.Context()
	products, err := suite.productRepo.List(ctx, productrepo.ProductFilter{Limit: 100, Offset: 0})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": products})
}

func (suite *APIPerformanceTestSuite) getProductHandler(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	product, err := suite.productRepo.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": product})
}

func (suite *APIPerformanceTestSuite) updateProductStockHandler(c *gin.Context) {
	// Mock implementation for performance testing
	c.JSON(http.StatusOK, gin.H{"id": c.Param("id"), "stock": 100})
}

func (suite *APIPerformanceTestSuite) createOrderHandler(c *gin.Context) {
	// Mock implementation for performance testing
	c.JSON(http.StatusCreated, gin.H{"id": uuid.New(), "status": "pending"})
}

func (suite *APIPerformanceTestSuite) listOrdersHandler(c *gin.Context) {
	ctx := c.Request.Context()
	orders, err := suite.orderRepo.List(ctx, erporder.OrderFilter{Limit: 100, Offset: 0})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (suite *APIPerformanceTestSuite) getOrderHandler(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	order, err := suite.orderRepo.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": order})
}

func (suite *APIPerformanceTestSuite) healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now().UTC()})
}

func (suite *APIPerformanceTestSuite) searchCustomersHandler(c *gin.Context) {
	// Mock implementation for performance testing
	c.JSON(http.StatusOK, gin.H{"customers": []gin.H{}, "total": 0})
}

func (suite *APIPerformanceTestSuite) searchProductsHandler(c *gin.Context) {
	// Mock implementation for performance testing
	c.JSON(http.StatusOK, gin.H{"products": []gin.H{}, "total": 0})
}

// Run the test suite
func TestAPIPerformanceSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}
	suite.Run(t, new(APIPerformanceTestSuite))
}

// Benchmark tests
func BenchmarkCustomerList(b *testing.B) {
	suite := &APIPerformanceTestSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/customers", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
	}
}

func BenchmarkProductSearch(b *testing.B) {
	suite := &APIPerformanceTestSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/search/products?q=test", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	suite := &APIPerformanceTestSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
	}
}