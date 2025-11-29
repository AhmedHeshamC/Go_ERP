package load

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// APILoadTestSuite contains all API load tests
type APILoadTestSuite struct {
	baseURL    string
	authToken  string
	framework  *LoadTestFramework
	userTokens []string
	productIDs []string
	orderIDs   []string
}

// NewAPILoadTestSuite creates a new API load test suite
func NewAPILoadTestSuite(baseURL string) *APILoadTestSuite {
	return &APILoadTestSuite{
		baseURL: baseURL,
	}
}

// setupTestData sets up test data for load testing
func (suite *APILoadTestSuite) setupTestData(t *testing.T) {
	// Create admin user and get auth token
	suite.authToken = suite.createTestUser(t, "admin", "admin@example.com", "admin123")

	// Create regular users for concurrent testing
	for i := 0; i < 100; i++ {
		token := suite.createTestUser(t, fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@example.com", i), "password123")
		suite.userTokens = append(suite.userTokens, token)
	}

	// Create test products
	for i := 0; i < 50; i++ {
		productID := suite.createTestProduct(t, fmt.Sprintf("Product %d", i))
		suite.productIDs = append(suite.productIDs, productID)
	}

	// Create test orders
	for i := 0; i < 20; i++ {
		orderID := suite.createTestOrder(t)
		suite.orderIDs = append(suite.orderIDs, orderID)
	}
}

// createTestUser creates a test user and returns auth token
func (suite *APILoadTestSuite) createTestUser(t *testing.T, name, email, password string) string {
	// Register user
	registerReq := map[string]interface{}{
		"first_name": name,
		"last_name":  "User",
		"email":      email,
		"password":   password,
		"role":       "customer",
	}

	body, _ := json.Marshal(registerReq)
	resp, err := http.Post(suite.baseURL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Login and get token
	loginReq := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	body, _ = json.Marshal(loginReq)
	resp, err = http.Post(suite.baseURL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	var loginResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&loginResp)

	token, ok := loginResp["token"].(string)
	require.True(t, ok)
	return token
}

// createTestProduct creates a test product
func (suite *APILoadTestSuite) createTestProduct(t *testing.T, name string) string {
	productReq := map[string]interface{}{
		"name":        name,
		"sku":         fmt.Sprintf("SKU-%d", uuid.New().ID()),
		"description": "Test product for load testing",
		"price":       29.99,
		"cost":        15.50,
		"weight":      1.5,
		"category_id": uuid.New(),
		"is_active":   true,
	}

	body, _ := json.Marshal(productReq)
	req, _ := http.NewRequest("POST", suite.baseURL+"/api/v1/products", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.authToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var productResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&productResp)

	productID, ok := productResp["id"].(string)
	require.True(t, ok)
	return productID
}

// createTestOrder creates a test order
func (suite *APILoadTestSuite) createTestOrder(t *testing.T) string {
	orderReq := map[string]interface{}{
		"customer_id":     uuid.New(),
		"shipping_method": "standard",
		"currency":        "USD",
		"items": []map[string]interface{}{
			{
				"product_id": suite.productIDs[0],
				"quantity":   2,
				"unit_price": 29.99,
			},
		},
	}

	body, _ := json.Marshal(orderReq)
	req, _ := http.NewRequest("POST", suite.baseURL+"/api/v1/orders", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.authToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var orderResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orderResp)

	orderID, ok := orderResp["id"].(string)
	require.True(t, ok)
	return orderID
}

// TestProductAPIPerformance tests product API performance under load
func (suite *APILoadTestSuite) TestProductAPIPerformance(t *testing.T) {
	suite.setupTestData(t)

	testCases := []struct {
		name        string
		config      LoadTestConfig
		requestFunc func(user int, iteration int) (*http.Request, error)
	}{
		{
			name: "GetProductsList",
			config: LoadTestConfig{
				Name:               "Get Products List Load Test",
				BaseURL:            suite.baseURL,
				ConcurrentUsers:    50,
				RequestsPerUser:    20,
				TestDuration:       30 * time.Second,
				TimeoutPerRequest:  5 * time.Second,
				ThinkTime:          100 * time.Millisecond,
				TargetRPS:          1000,
				MaxErrorRate:       0.01, // 1%
				MaxResponseTime:    200 * time.Millisecond,
				ExpectedStatusCode: 200,
			},
			requestFunc: func(user int, iteration int) (*http.Request, error) {
				token := suite.userTokens[user%len(suite.userTokens)]
				req, err := http.NewRequest("GET", suite.baseURL+"/api/v1/products?limit=20&page=1", nil)
				if err != nil {
					return nil, err
				}
				req.Header.Set("Authorization", "Bearer "+token)
				return req, nil
			},
		},
		{
			name: "GetProductDetails",
			config: LoadTestConfig{
				Name:               "Get Product Details Load Test",
				BaseURL:            suite.baseURL,
				ConcurrentUsers:    30,
				RequestsPerUser:    15,
				TestDuration:       20 * time.Second,
				TimeoutPerRequest:  3 * time.Second,
				ThinkTime:          50 * time.Millisecond,
				TargetRPS:          800,
				MaxErrorRate:       0.01,
				MaxResponseTime:    150 * time.Millisecond,
				ExpectedStatusCode: 200,
			},
			requestFunc: func(user int, iteration int) (*http.Request, error) {
				token := suite.userTokens[user%len(suite.userTokens)]
				productID := suite.productIDs[iteration%len(suite.productIDs)]
				req, err := http.NewRequest("GET", suite.baseURL+"/api/v1/products/"+productID, nil)
				if err != nil {
					return nil, err
				}
				req.Header.Set("Authorization", "Bearer "+token)
				return req, nil
			},
		},
		{
			name: "SearchProducts",
			config: LoadTestConfig{
				Name:               "Search Products Load Test",
				BaseURL:            suite.baseURL,
				ConcurrentUsers:    25,
				RequestsPerUser:    10,
				TestDuration:       25 * time.Second,
				TimeoutPerRequest:  5 * time.Second,
				ThinkTime:          200 * time.Millisecond,
				TargetRPS:          600,
				MaxErrorRate:       0.02,
				MaxResponseTime:    300 * time.Millisecond,
				ExpectedStatusCode: 200,
			},
			requestFunc: func(user int, iteration int) (*http.Request, error) {
				token := suite.userTokens[user%len(suite.userTokens)]
				searchTerms := []string{"laptop", "phone", "tablet", "computer", "electronics"}
				term := searchTerms[iteration%len(searchTerms)]
				req, err := http.NewRequest("GET", suite.baseURL+"/api/v1/products/search?q="+url.QueryEscape(term), nil)
				if err != nil {
					return nil, err
				}
				req.Header.Set("Authorization", "Bearer "+token)
				return req, nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			framework := NewLoadTestFramework(&tc.config)
			result, err := framework.RunLoadTest(tc.requestFunc)
			require.NoError(t, err)

			err = framework.ValidateResults()
			require.NoError(t, err, "Load test validation failed: %v", err)

			// Print detailed results
			t.Logf("=== %s Results ===", tc.name)
			t.Logf("Duration: %v", result.Duration)
			t.Logf("Total Requests: %d", result.TotalRequests)
			t.Logf("Successful Requests: %d", result.SuccessfulRequests)
			t.Logf("Failed Requests: %d", result.FailedRequests)
			t.Logf("Requests Per Second: %.2f", result.RequestsPerSecond)
			t.Logf("Error Rate: %.2f%%", result.ErrorRate*100)
			t.Logf("Average Response Time: %v", result.AverageResponseTime)
			t.Logf("95th Percentile Response Time: %v", result.P95ResponseTime)
			t.Logf("Peak Memory Usage: %d MB", result.SystemMetrics.PeakMemoryMB)
			t.Logf("Peak Goroutine Count: %d", result.SystemMetrics.PeakGoroutines)
		})
	}
}

// TestOrderAPIPerformance tests order API performance under load
func (suite *APILoadTestSuite) TestOrderAPIPerformance(t *testing.T) {
	suite.setupTestData(t)

	testCases := []struct {
		name        string
		config      LoadTestConfig
		requestFunc func(user int, iteration int) (*http.Request, error)
	}{
		{
			name: "CreateOrder",
			config: LoadTestConfig{
				Name:               "Create Order Load Test",
				BaseURL:            suite.baseURL,
				ConcurrentUsers:    20,
				RequestsPerUser:    5,
				TestDuration:       30 * time.Second,
				TimeoutPerRequest:  10 * time.Second,
				ThinkTime:          500 * time.Millisecond,
				TargetRPS:          200,
				MaxErrorRate:       0.02,
				MaxResponseTime:    500 * time.Millisecond,
				ExpectedStatusCode: 201,
			},
			requestFunc: func(user int, iteration int) (*http.Request, error) {
				token := suite.userTokens[user%len(suite.userTokens)]

				orderReq := map[string]interface{}{
					"customer_id":     uuid.New().String(),
					"shipping_method": "standard",
					"currency":        "USD",
					"items": []map[string]interface{}{
						{
							"product_id": suite.productIDs[iteration%len(suite.productIDs)],
							"quantity":   (iteration % 5) + 1,
							"unit_price": 29.99,
						},
					},
				}

				body, _ := json.Marshal(orderReq)
				req, err := http.NewRequest("POST", suite.baseURL+"/api/v1/orders", bytes.NewBuffer(body))
				if err != nil {
					return nil, err
				}
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Content-Type", "application/json")
				return req, nil
			},
		},
		{
			name: "GetOrderList",
			config: LoadTestConfig{
				Name:               "Get Order List Load Test",
				BaseURL:            suite.baseURL,
				ConcurrentUsers:    30,
				RequestsPerUser:    10,
				TestDuration:       20 * time.Second,
				TimeoutPerRequest:  5 * time.Second,
				ThinkTime:          100 * time.Millisecond,
				TargetRPS:          800,
				MaxErrorRate:       0.01,
				MaxResponseTime:    200 * time.Millisecond,
				ExpectedStatusCode: 200,
			},
			requestFunc: func(user int, iteration int) (*http.Request, error) {
				token := suite.userTokens[user%len(suite.userTokens)]
				req, err := http.NewRequest("GET", suite.baseURL+"/api/v1/orders?limit=20&page=1", nil)
				if err != nil {
					return nil, err
				}
				req.Header.Set("Authorization", "Bearer "+token)
				return req, nil
			},
		},
		{
			name: "GetOrderDetails",
			config: LoadTestConfig{
				Name:               "Get Order Details Load Test",
				BaseURL:            suite.baseURL,
				ConcurrentUsers:    25,
				RequestsPerUser:    8,
				TestDuration:       15 * time.Second,
				TimeoutPerRequest:  3 * time.Second,
				ThinkTime:          50 * time.Millisecond,
				TargetRPS:          600,
				MaxErrorRate:       0.01,
				MaxResponseTime:    150 * time.Millisecond,
				ExpectedStatusCode: 200,
			},
			requestFunc: func(user int, iteration int) (*http.Request, error) {
				token := suite.userTokens[user%len(suite.userTokens)]
				orderID := suite.orderIDs[iteration%len(suite.orderIDs)]
				req, err := http.NewRequest("GET", suite.baseURL+"/api/v1/orders/"+orderID, nil)
				if err != nil {
					return nil, err
				}
				req.Header.Set("Authorization", "Bearer "+token)
				return req, nil
			},
		},
		{
			name: "UpdateOrderStatus",
			config: LoadTestConfig{
				Name:               "Update Order Status Load Test",
				BaseURL:            suite.baseURL,
				ConcurrentUsers:    15,
				RequestsPerUser:    6,
				TestDuration:       20 * time.Second,
				TimeoutPerRequest:  5 * time.Second,
				ThinkTime:          200 * time.Millisecond,
				TargetRPS:          400,
				MaxErrorRate:       0.02,
				MaxResponseTime:    300 * time.Millisecond,
				ExpectedStatusCode: 200,
			},
			requestFunc: func(user int, iteration int) (*http.Request, error) {
				token := suite.authToken // Only admin can update status

				statuses := []string{"confirmed", "processing", "shipped", "delivered"}
				status := statuses[iteration%len(statuses)]

				updateReq := map[string]interface{}{
					"status": status,
					"reason": "Load test status update",
				}

				body, _ := json.Marshal(updateReq)
				orderID := suite.orderIDs[iteration%len(suite.orderIDs)]
				req, err := http.NewRequest("PUT", suite.baseURL+"/api/v1/orders/"+orderID+"/status", bytes.NewBuffer(body))
				if err != nil {
					return nil, err
				}
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Content-Type", "application/json")
				return req, nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			framework := NewLoadTestFramework(&tc.config)
			result, err := framework.RunLoadTest(tc.requestFunc)
			require.NoError(t, err)

			err = framework.ValidateResults()
			require.NoError(t, err, "Load test validation failed: %v", err)

			// Print detailed results
			t.Logf("=== %s Results ===", tc.name)
			t.Logf("Duration: %v", result.Duration)
			t.Logf("Total Requests: %d", result.TotalRequests)
			t.Logf("Successful Requests: %d", result.SuccessfulRequests)
			t.Logf("Failed Requests: %d", result.FailedRequests)
			t.Logf("Requests Per Second: %.2f", result.RequestsPerSecond)
			t.Logf("Error Rate: %.2f%%", result.ErrorRate*100)
			t.Logf("Average Response Time: %v", result.AverageResponseTime)
			t.Logf("95th Percentile Response Time: %v", result.P95ResponseTime)
		})
	}
}

// TestAuthAPIPerformance tests authentication API performance under load
func (suite *APILoadTestSuite) TestAuthAPIPerformance(t *testing.T) {
	// Create test users for authentication testing
	var testUsers []map[string]string
	for i := 0; i < 50; i++ {
		email := fmt.Sprintf("authtest%d@example.com", i)
		token := suite.createTestUser(t, fmt.Sprintf("authtest%d", i), email, "password123")
		testUsers = append(testUsers, map[string]string{
			"email":    email,
			"password": "password123",
			"token":    token,
		})
	}

	testCases := []struct {
		name        string
		config      LoadTestConfig
		requestFunc func(user int, iteration int) (*http.Request, error)
	}{
		{
			name: "LoginLoadTest",
			config: LoadTestConfig{
				Name:               "Login Load Test",
				BaseURL:            suite.baseURL,
				ConcurrentUsers:    40,
				RequestsPerUser:    10,
				TestDuration:       30 * time.Second,
				TimeoutPerRequest:  5 * time.Second,
				ThinkTime:          100 * time.Millisecond,
				TargetRPS:          800,
				MaxErrorRate:       0.01,
				MaxResponseTime:    300 * time.Millisecond,
				ExpectedStatusCode: 200,
			},
			requestFunc: func(user int, iteration int) (*http.Request, error) {
				testUser := testUsers[user%len(testUsers)]

				loginReq := map[string]interface{}{
					"email":    testUser["email"],
					"password": testUser["password"],
				}

				body, _ := json.Marshal(loginReq)
				req, err := http.NewRequest("POST", suite.baseURL+"/api/v1/auth/login", bytes.NewBuffer(body))
				if err != nil {
					return nil, err
				}
				req.Header.Set("Content-Type", "application/json")
				return req, nil
			},
		},
		{
			name: "TokenRefreshLoadTest",
			config: LoadTestConfig{
				Name:               "Token Refresh Load Test",
				BaseURL:            suite.baseURL,
				ConcurrentUsers:    30,
				RequestsPerUser:    8,
				TestDuration:       20 * time.Second,
				TimeoutPerRequest:  3 * time.Second,
				ThinkTime:          200 * time.Millisecond,
				TargetRPS:          600,
				MaxErrorRate:       0.01,
				MaxResponseTime:    200 * time.Millisecond,
				ExpectedStatusCode: 200,
			},
			requestFunc: func(user int, iteration int) (*http.Request, error) {
				testUser := testUsers[user%len(testUsers)]

				refreshReq := map[string]interface{}{
					"token": testUser["token"],
				}

				body, _ := json.Marshal(refreshReq)
				req, err := http.NewRequest("POST", suite.baseURL+"/api/v1/auth/refresh", bytes.NewBuffer(body))
				if err != nil {
					return nil, err
				}
				req.Header.Set("Content-Type", "application/json")
				return req, nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			framework := NewLoadTestFramework(&tc.config)
			result, err := framework.RunLoadTest(tc.requestFunc)
			require.NoError(t, err)

			err = framework.ValidateResults()
			require.NoError(t, err, "Load test validation failed: %v", err)

			// Print detailed results
			t.Logf("=== %s Results ===", tc.name)
			t.Logf("Duration: %v", result.Duration)
			t.Logf("Total Requests: %d", result.TotalRequests)
			t.Logf("Successful Requests: %d", result.SuccessfulRequests)
			t.Logf("Failed Requests: %d", result.FailedRequests)
			t.Logf("Requests Per Second: %.2f", result.RequestsPerSecond)
			t.Logf("Error Rate: %.2f%%", result.ErrorRate*100)
			t.Logf("Average Response Time: %v", result.AverageResponseTime)
			t.Logf("95th Percentile Response Time: %v", result.P95ResponseTime)
		})
	}
}

// TestMixedWorkload tests mixed API workload to simulate real usage patterns
func (suite *APILoadTestSuite) TestMixedWorkload(t *testing.T) {
	suite.setupTestData(t)

	// Define workload distribution (60% reads, 30% searches, 10% writes)
	workloadDistribution := []struct {
		weight      int
		requestFunc func(user int, iteration int) (*http.Request, error)
	}{
		{
			weight: 60, // 60% reads
			requestFunc: func(user int, iteration int) (*http.Request, error) {
				token := suite.userTokens[user%len(suite.userTokens)]

				// Mix of product and order reads
				if iteration%2 == 0 {
					productID := suite.productIDs[iteration%len(suite.productIDs)]
					req, err := http.NewRequest("GET", suite.baseURL+"/api/v1/products/"+productID, nil)
					if err != nil {
						return nil, err
					}
					req.Header.Set("Authorization", "Bearer "+token)
					return req, nil
				} else {
					req, err := http.NewRequest("GET", suite.baseURL+"/api/v1/orders?limit=10&page=1", nil)
					if err != nil {
						return nil, err
					}
					req.Header.Set("Authorization", "Bearer "+token)
					return req, nil
				}
			},
		},
		{
			weight: 30, // 30% searches
			requestFunc: func(user int, iteration int) (*http.Request, error) {
				token := suite.userTokens[user%len(suite.userTokens)]
				searchTerms := []string{"product", "item", "order", "laptop", "phone"}
				term := searchTerms[iteration%len(searchTerms)]
				req, err := http.NewRequest("GET", suite.baseURL+"/api/v1/products/search?q="+url.QueryEscape(term), nil)
				if err != nil {
					return nil, err
				}
				req.Header.Set("Authorization", "Bearer "+token)
				return req, nil
			},
		},
		{
			weight: 10, // 10% writes
			requestFunc: func(user int, iteration int) (*http.Request, error) {
				token := suite.authToken // Only admin can create

				// Create new product
				productReq := map[string]interface{}{
					"name":        fmt.Sprintf("Load Test Product %d", iteration),
					"sku":         fmt.Sprintf("LT-%d", iteration),
					"description": "Product created during load test",
					"price":       decimal.NewFromFloat(19.99 + float64(iteration%100)),
					"cost":        decimal.NewFromFloat(10.00),
					"weight":      1.0,
					"category_id": suite.productIDs[0],
					"is_active":   true,
				}

				body, _ := json.Marshal(productReq)
				req, err := http.NewRequest("POST", suite.baseURL+"/api/v1/products", bytes.NewBuffer(body))
				if err != nil {
					return nil, err
				}
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Content-Type", "application/json")
				return req, nil
			},
		},
	}

	// Create weighted request selector
	totalWeight := 0
	for _, w := range workloadDistribution {
		totalWeight += w.weight
	}

	config := LoadTestConfig{
		Name:               "Mixed Workload Load Test",
		BaseURL:            suite.baseURL,
		ConcurrentUsers:    50,
		RequestsPerUser:    20,
		TestDuration:       60 * time.Second,
		TimeoutPerRequest:  10 * time.Second,
		ThinkTime:          50 * time.Millisecond,
		TargetRPS:          1000,
		MaxErrorRate:       0.02,
		MaxResponseTime:    300 * time.Millisecond,
		ExpectedStatusCode: 200,
	}

	requestFunc := func(user int, iteration int) (*http.Request, error) {
		// Select request type based on weighted distribution
		rand := (user*1000 + iteration) % totalWeight
		cumulative := 0

		for _, workload := range workloadDistribution {
			cumulative += workload.weight
			if rand < cumulative {
				return workload.requestFunc(user, iteration)
			}
		}

		// Fallback to first request type
		return workloadDistribution[0].requestFunc(user, iteration)
	}

	framework := NewLoadTestFramework(&config)
	result, err := framework.RunLoadTest(requestFunc)
	require.NoError(t, err)

	err = framework.ValidateResults()
	require.NoError(t, err, "Mixed workload test validation failed: %v", err)

	// Print detailed results
	t.Logf("=== Mixed Workload Test Results ===")
	t.Logf("Duration: %v", result.Duration)
	t.Logf("Total Requests: %d", result.TotalRequests)
	t.Logf("Successful Requests: %d", result.SuccessfulRequests)
	t.Logf("Failed Requests: %d", result.FailedRequests)
	t.Logf("Requests Per Second: %.2f", result.RequestsPerSecond)
	t.Logf("Error Rate: %.2f%%", result.ErrorRate*100)
	t.Logf("Average Response Time: %v", result.AverageResponseTime)
	t.Logf("95th Percentile Response Time: %v", result.P95ResponseTime)
	t.Logf("Peak Memory Usage: %d MB", result.SystemMetrics.PeakMemoryMB)
	t.Logf("Peak Goroutine Count: %d", result.SystemMetrics.PeakGoroutines)
}

// TestAPILoadTestSuite runs the complete API load test suite
func TestAPILoadTestSuite(t *testing.T) {
	baseURL := "http://localhost:8080"
	suite := NewAPILoadTestSuite(baseURL)

	t.Run("ProductAPI", suite.TestProductAPIPerformance)
	t.Run("OrderAPI", suite.TestOrderAPIPerformance)
	t.Run("AuthAPI", suite.TestAuthAPIPerformance)
	t.Run("MixedWorkload", suite.TestMixedWorkload)
}
