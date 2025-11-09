package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"erpgo/internal/application/services/customer"
	"erpgo/internal/domain/customers/entities"
	customerrepo "erpgo/internal/domain/customers/repositories"
	"erpgo/internal/infrastructure/database"
	"erpgo/internal/infrastructure/logger"
)

// CustomerIntegrationTestSuite tests the customer API endpoints
type CustomerIntegrationTestSuite struct {
	suite.Suite
	db           *database.Database
	customerRepo customerrepo.CustomerRepository
	customerSvc  customer.Service
	router       *gin.Engine
	testCustomers []*entities.Customer
}

// SetupSuite sets up the test suite
func (suite *CustomerIntegrationTestSuite) SetupSuite() {
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
	suite.customerRepo = customerrepo.NewPostgresCustomerRepository(suite.db.GetDB(), dbConfig, logger.NewNopLogger())

	// Initialize service
	suite.customerSvc = customer.NewCustomerService(suite.customerRepo, logger.NewNopLogger())

	// Set up Gin router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.setupRoutes()
}

// TearDownSuite cleans up after the test suite
func (suite *CustomerIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.cleanupTestData()
		suite.db.Close()
	}
}

// SetupTest runs before each test
func (suite *CustomerIntegrationTestSuite) SetupTest() {
	suite.cleanupTestData()
	suite.createTestCustomers()
}

// TearDownTest runs after each test
func (suite *CustomerIntegrationTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

// setupRoutes sets up the API routes for testing
func (suite *CustomerIntegrationTestSuite) setupRoutes() {
	api := suite.router.Group("/api/v1")
	{
		api.GET("/customers", suite.listCustomers)
		api.POST("/customers", suite.createCustomer)
		api.GET("/customers/:id", suite.getCustomer)
		api.PUT("/customers/:id", suite.updateCustomer)
		api.DELETE("/customers/:id", suite.deleteCustomer)
		api.GET("/customers/search", suite.searchCustomers)
	}
}

// cleanupTestData removes all test data
func (suite *CustomerIntegrationTestSuite) cleanupTestData() {
	if suite.db != nil {
		suite.db.GetDB().Exec("DELETE FROM customers")
	}
}

// createTestCustomers creates sample customers for testing
func (suite *CustomerIntegrationTestSuite) createTestCustomers() {
	ctx := context.Background()

	customers := []*entities.Customer{
		{
			Name:        "John Doe",
			Email:       "john.doe@example.com",
			Phone:       "+1234567890",
			Address:     "123 Main St, City, State 12345",
			CustomerType: entities.IndividualCustomer,
			Status:      entities.ActiveCustomer,
		},
		{
			Name:        "Acme Corporation",
			Email:       "contact@acme.com",
			Phone:       "+0987654321",
			Address:     "456 Business Ave, Commercial City, State 67890",
			CustomerType: entities.BusinessCustomer,
			Status:      entities.ActiveCustomer,
			TaxID:       "123-45-6789",
		},
	}

	for _, customer := range customers {
		err := suite.customerRepo.Create(ctx, customer)
		if err == nil {
			suite.testCustomers = append(suite.testCustomers, customer)
		}
	}
}

// API Handlers

func (suite *CustomerIntegrationTestSuite) listCustomers(c *gin.Context) {
	ctx := c.Request.Context()

	customers, err := suite.customerRepo.List(ctx, customerrepo.CustomerFilter{
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"customers": customers})
}

func (suite *CustomerIntegrationTestSuite) createCustomer(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Name         string                 `json:"name" binding:"required"`
		Email        string                 `json:"email" binding:"required,email"`
		Phone        string                 `json:"phone"`
		Address      string                 `json:"address"`
		CustomerType entities.CustomerType  `json:"customer_type"`
		TaxID        string                 `json:"tax_id"`
		Metadata     map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer := &entities.Customer{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Address:      req.Address,
		CustomerType: req.CustomerType,
		Status:       entities.ActiveCustomer,
		TaxID:        req.TaxID,
		Metadata:     req.Metadata,
	}

	if err := suite.customerRepo.Create(ctx, customer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"customer": customer})
}

func (suite *CustomerIntegrationTestSuite) getCustomer(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	customer, err := suite.customerRepo.GetByID(ctx, id)
	if err != nil {
		if err == customerrepo.ErrCustomerNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"customer": customer})
}

func (suite *CustomerIntegrationTestSuite) updateCustomer(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	var req struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Phone   string `json:"phone"`
		Address string `json:"address"`
		Status  string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing customer
	customer, err := suite.customerRepo.GetByID(ctx, id)
	if err != nil {
		if err == customerrepo.ErrCustomerNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if req.Name != "" {
		customer.Name = req.Name
	}
	if req.Email != "" {
		customer.Email = req.Email
	}
	if req.Phone != "" {
		customer.Phone = req.Phone
	}
	if req.Address != "" {
		customer.Address = req.Address
	}
	if req.Status != "" {
		customer.Status = entities.CustomerStatus(req.Status)
	}

	if err := suite.customerRepo.Update(ctx, customer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"customer": customer})
}

func (suite *CustomerIntegrationTestSuite) deleteCustomer(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	if err := suite.customerRepo.Delete(ctx, id); err != nil {
		if err == customerrepo.ErrCustomerNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (suite *CustomerIntegrationTestSuite) searchCustomers(c *gin.Context) {
	ctx := c.Request.Context()

	query := c.Query("q")
	customerType := c.Query("type")
	status := c.Query("status")

	filter := customerrepo.CustomerFilter{
		Query:        query,
		CustomerType: entities.CustomerType(customerType),
		Status:       entities.CustomerStatus(status),
		Limit:        100,
		Offset:       0,
	}

	customers, err := suite.customerRepo.Search(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"customers": customers})
}

// Test Cases

func (suite *CustomerIntegrationTestSuite) TestListCustomers() {
	req, _ := http.NewRequest("GET", "/api/v1/customers", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	customers, ok := response["customers"].([]interface{})
	suite.True(ok)
	suite.Greater(len(customers), 0)
}

func (suite *CustomerIntegrationTestSuite) TestCreateCustomer() {
	payload := map[string]interface{}{
		"name":          "Jane Smith",
		"email":         "jane.smith@example.com",
		"phone":         "+15551234567",
		"address":       "789 Oak St, Town, State 11111",
		"customer_type": "individual",
		"metadata": map[string]interface{}{
			"source": "web",
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/customers", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	customer, ok := response["customer"].(map[string]interface{})
	suite.True(ok)
	suite.Equal("Jane Smith", customer["name"])
	suite.Equal("jane.smith@example.com", customer["email"])
}

func (suite *CustomerIntegrationTestSuite) TestGetCustomer() {
	if len(suite.testCustomers) == 0 {
		suite.T().Skip("No test customers available")
		return
	}

	customer := suite.testCustomers[0]
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/customers/%s", customer.ID.String()), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	retrievedCustomer, ok := response["customer"].(map[string]interface{})
	suite.True(ok)
	suite.Equal(customer.Name, retrievedCustomer["name"])
	suite.Equal(customer.Email, retrievedCustomer["email"])
}

func (suite *CustomerIntegrationTestSuite) TestUpdateCustomer() {
	if len(suite.testCustomers) == 0 {
		suite.T().Skip("No test customers available")
		return
	}

	customer := suite.testCustomers[0]
	payload := map[string]interface{}{
		"name":    "Updated Name",
		"phone":   "+19998887777",
		"address": "Updated Address",
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/customers/%s", customer.ID.String()), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	updatedCustomer, ok := response["customer"].(map[string]interface{})
	suite.True(ok)
	suite.Equal("Updated Name", updatedCustomer["name"])
	suite.Equal("+19998887777", updatedCustomer["phone"])
}

func (suite *CustomerIntegrationTestSuite) TestDeleteCustomer() {
	if len(suite.testCustomers) == 0 {
		suite.T().Skip("No test customers available")
		return
	}

	customer := suite.testCustomers[0]
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/customers/%s", customer.ID.String()), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNoContent, w.Code)
}

func (suite *CustomerIntegrationTestSuite) TestSearchCustomers() {
	req, _ := http.NewRequest("GET", "/api/v1/customers/search?q=John&type=individual", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	customers, ok := response["customers"].([]interface{})
	suite.True(ok)
}

func (suite *CustomerIntegrationTestSuite) TestCreateCustomerValidation() {
	// Test missing required fields
	payload := map[string]interface{}{
		"email": "test@example.com",
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/customers", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	// Test invalid email
	payload = map[string]interface{}{
		"name":  "Test Customer",
		"email": "invalid-email",
	}

	jsonData, _ = json.Marshal(payload)
	req, _ = http.NewRequest("POST", "/api/v1/customers", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *CustomerIntegrationTestSuite) TestGetNonExistentCustomer() {
	nonExistentID := uuid.New()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/customers/%s", nonExistentID.String()), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNotFound, w.Code)
}

// Run the test suite
func TestCustomerIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(CustomerIntegrationTestSuite))
}

// Benchmark tests
func BenchmarkListCustomers(b *testing.B) {
	suite := &CustomerIntegrationTestSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()
	suite.SetupTest()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/customers", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
	}
}

func BenchmarkCreateCustomer(b *testing.B) {
	suite := &CustomerIntegrationTestSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	payload := map[string]interface{}{
		"name":          "Benchmark Customer",
		"email":         "benchmark@example.com",
		"customer_type": "individual",
	}

	for i := 0; i < b.N; i++ {
		jsonData, _ := json.Marshal(payload)
		jsonData.(*[]byte)[0] = byte(i) // Make email unique
		req, _ := http.NewRequest("POST", "/api/v1/customers", bytes.NewBuffer(*jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
	}
}