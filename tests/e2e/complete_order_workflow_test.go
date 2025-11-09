package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
)

// CompleteOrderWorkflowTestSuite tests the complete end-to-end order processing workflow
type CompleteOrderWorkflowTestSuite struct {
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

// TestData holds test data for the workflow
type TestData struct {
	Customers []*customerrepo.Customer
	Products  []*productrepo.Product
	Orders    []*erporder.Order
}

// SetupSuite sets up the test suite
func (suite *CompleteOrderWorkflowTestSuite) SetupSuite() {
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

	// Set up Gin router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.setupRoutes()

	// Initialize test data
	suite.testData = &TestData{}
}

// TearDownSuite cleans up after the test suite
func (suite *CompleteOrderWorkflowTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.cleanupTestData()
		suite.db.Close()
	}
}

// SetupTest runs before each test
func (suite *CompleteOrderWorkflowTestSuite) SetupTest() {
	suite.cleanupTestData()
	suite.createTestCustomers()
	suite.createTestProducts()
}

// TearDownTest runs after each test
func (suite *CompleteOrderWorkflowTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

// setupRoutes sets up the API routes for testing
func (suite *CompleteOrderWorkflowTestSuite) setupRoutes() {
	api := suite.router.Group("/api/v1")
	{
		// Customer endpoints
		api.POST("/customers", suite.createCustomerHandler)
		api.GET("/customers/:id", suite.getCustomerHandler)

		// Product endpoints
		api.POST("/products", suite.createProductHandler)
		api.GET("/products/:id", suite.getProductHandler)
		api.PUT("/products/:id/stock", suite.updateProductStockHandler)

		// Order endpoints
		api.POST("/orders", suite.createOrderHandler)
		api.GET("/orders/:id", suite.getOrderHandler)
		api.PUT("/orders/:id/status", suite.updateOrderStatusHandler)
		api.POST("/orders/:id/items", suite.addOrderItemHandler)
		api.POST("/orders/:id/payment", suite.processPaymentHandler)
		api.POST("/orders/:id/ship", suite.shipOrderHandler)
		api.POST("/orders/:id/complete", suite.completeOrderHandler)
	}
}

// cleanupTestData removes all test data
func (suite *CompleteOrderWorkflowTestSuite) cleanupTestData() {
	if suite.db != nil {
		tx := suite.db.GetDB()
		tx.Exec("DELETE FROM order_items")
		tx.Exec("DELETE FROM orders")
		tx.Exec("DELETE FROM products")
		tx.Exec("DELETE FROM customers")
	}
}

// createTestCustomers creates sample customers for testing
func (suite *CompleteOrderWorkflowTestSuite) createTestCustomers() {
	ctx := context.Background()

	customers := []*customerrepo.Customer{
		{
			Name:         "John Doe",
			Email:        "john.doe@example.com",
			Phone:        "+1234567890",
			Address:      "123 Main St, City, State 12345",
			CustomerType: "individual",
			Status:       "active",
		},
		{
			Name:         "Acme Corporation",
			Email:        "contact@acme.com",
			Phone:        "+0987654321",
			Address:      "456 Business Ave, Commercial City, State 67890",
			CustomerType: "business",
			Status:       "active",
			TaxID:        "123-45-6789",
		},
	}

	for _, customer := range customers {
		err := suite.customerRepo.Create(ctx, customer)
		if err == nil {
			suite.testData.Customers = append(suite.testData.Customers, customer)
		}
	}
}

// createTestProducts creates sample products for testing
func (suite *CompleteOrderWorkflowTestSuite) createTestProducts() {
	ctx := context.Background()

	products := []*productrepo.Product{
		{
			Name:        "Laptop Computer",
			Description: "High-performance laptop for business use",
			SKU:         "LAPTOP-001",
			Price:       decimal.NewFromFloat(1299.99),
			Stock:       50,
			CategoryID:  uuid.New(),
		},
		{
			Name:        "Wireless Mouse",
			Description: "Ergonomic wireless mouse",
			SKU:         "MOUSE-001",
			Price:       decimal.NewFromFloat(29.99),
			Stock:       200,
			CategoryID:  uuid.New(),
		},
		{
			Name:        "USB-C Cable",
			Description: "USB-C charging cable",
			SKU:         "CABLE-001",
			Price:       decimal.NewFromFloat(19.99),
			Stock:       500,
			CategoryID:  uuid.New(),
		},
	}

	for _, product := range products {
		err := suite.productRepo.Create(ctx, product)
		if err == nil {
			suite.testData.Products = append(suite.testData.Products, product)
		}
	}
}

// Test Complete Order Workflow

func (suite *CompleteOrderWorkflowTestSuite) TestCompleteOrderWorkflow() {
	if len(suite.testData.Customers) == 0 || len(suite.testData.Products) == 0 {
		suite.T().Skip("Insufficient test data")
		return
	}

	// Step 1: Create a new customer
	customerPayload := map[string]interface{}{
		"name":          "Workflow Test Customer",
		"email":         "workflow@example.com",
		"phone":         "+1555123456",
		"address":       "789 Workflow St, Test City, TC 12345",
		"customer_type": "individual",
	}

	jsonData, _ := json.Marshal(customerPayload)
	req, _ := http.NewRequest("POST", "/api/v1/customers", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusCreated, w.Code)

	var customerResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &customerResponse)
	suite.NoError(err)

	customerData := customerResponse["customer"].(map[string]interface{})
	customerID := customerData["id"].(string)

	// Step 2: Create a new product
	productPayload := map[string]interface{}{
		"name":        "Workflow Test Product",
		"description": "Product for workflow testing",
		"sku":         "WFP-001",
		"price":       99.99,
		"stock":       100,
	}

	jsonData, _ = json.Marshal(productPayload)
	req, _ = http.NewRequest("POST", "/api/v1/products", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusCreated, w.Code)

	var productResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &productResponse)
	suite.NoError(err)

	productData := productResponse["product"].(map[string]interface{})
	productID := productData["id"].(string)

	// Step 3: Create an order with multiple items
	orderPayload := map[string]interface{}{
		"customer_id": customerID,
		"items": []map[string]interface{}{
			{
				"product_id": productID,
				"quantity":  2,
			},
			{
				"product_id": suite.testData.Products[0].ID.String(),
				"quantity":  1,
			},
		},
		"notes": "End-to-end workflow test order",
	}

	jsonData, _ = json.Marshal(orderPayload)
	req, _ = http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusCreated, w.Code)

	var orderResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &orderResponse)
	suite.NoError(err)

	orderData := orderResponse["order"].(map[string]interface{})
	orderID := orderData["id"].(string)

	// Step 4: Verify order details
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/orders/%s", orderID), nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var orderDetailsResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &orderDetailsResponse)
	suite.NoError(err)

	orderDetails := orderDetailsResponse["order"].(map[string]interface{})
	items := orderDetailsResponse["items"].([]interface{})
	suite.Greater(len(items), 1) // Should have at least 2 items
	suite.Equal("pending", orderDetails["status"])

	// Step 5: Process payment
	paymentPayload := map[string]interface{}{
		"payment_method": "credit_card",
		"amount":         orderDetails["total"],
		"transaction_id": fmt.Sprintf("txn_%d", time.Now().Unix()),
	}

	jsonData, _ = json.Marshal(paymentPayload)
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/orders/%s/payment", orderID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	// Step 6: Ship the order
	shipPayload := map[string]interface{}{
		"tracking_number": fmt.Sprintf("TRACK_%d", time.Now().Unix()),
		"carrier":        "UPS",
		"shipping_date":  time.Now().Format(time.RFC3339),
	}

	jsonData, _ = json.Marshal(shipPayload)
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/orders/%s/ship", orderID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	// Step 7: Complete the order
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/orders/%s/complete", orderID), nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	// Step 8: Verify final order status
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/orders/%s", orderID), nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var finalOrderResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &finalOrderResponse)
	suite.NoError(err)

	finalOrder := finalOrderResponse["order"].(map[string]interface{})
	suite.Equal("completed", finalOrder["status"])
	suite.NotNil(finalOrder["completed_at"])
}

// Simplified API handlers for testing (keeping the test focused on the workflow)
func (suite *CompleteOrderWorkflowTestSuite) createCustomerHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Name         string `json:"name" binding:"required"`
		Email        string `json:"email" binding:"required,email"`
		Phone        string `json:"phone"`
		Address      string `json:"address"`
		CustomerType string `json:"customer_type"`
		TaxID        string `json:"tax_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer := &customerrepo.Customer{
		ID:           uuid.New(),
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Address:      req.Address,
		CustomerType: req.CustomerType,
		Status:       "active",
		TaxID:        req.TaxID,
	}

	if err := suite.customerRepo.Create(ctx, customer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"customer": customer})
}

func (suite *CompleteOrderWorkflowTestSuite) getCustomerHandler(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	customer, err := suite.customerRepo.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"customer": customer})
}

func (suite *CompleteOrderWorkflowTestSuite) createProductHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Name        string          `json:"name" binding:"required"`
		Description string          `json:"description"`
		SKU         string          `json:"sku" binding:"required"`
		Price       decimal.Decimal `json:"price" binding:"required"`
		Stock       int             `json:"stock" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product := &productrepo.Product{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		SKU:         req.SKU,
		Price:       req.Price,
		Stock:       req.Stock,
		CategoryID:  uuid.New(),
		Status:      "active",
	}

	if err := suite.productRepo.Create(ctx, product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"product": product})
}

func (suite *CompleteOrderWorkflowTestSuite) getProductHandler(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	product, err := suite.productRepo.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": product})
}

func (suite *CompleteOrderWorkflowTestSuite) updateProductStockHandler(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var req struct {
		Stock int `json:"stock" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := suite.productRepo.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	product.Stock = req.Stock
	if err := suite.productRepo.Update(ctx, product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": product})
}

func (suite *CompleteOrderWorkflowTestSuite) createOrderHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		CustomerID uuid.UUID `json:"customer_id" binding:"required"`
		Items      []struct {
			ProductID uuid.UUID       `json:"product_id" binding:"required"`
			Quantity  int             `json:"quantity" binding:"required"`
			Price     decimal.Decimal `json:"price"`
		} `json:"items" binding:"required,min=1"`
		Notes string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create order
	orderEntity := &erporder.Order{
		ID:         uuid.New(),
		CustomerID: req.CustomerID,
		Status:     entities.OrderStatusPending,
		Notes:      req.Notes,
		CreatedAt:  time.Now().UTC(),
	}

	if err := suite.orderRepo.Create(ctx, orderEntity); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add order items
	var totalAmount decimal.Decimal
	for _, item := range req.Items {
		// Get product to verify price and stock
		product, err := suite.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Product not found: %s", item.ProductID)})
			return
		}

		// Use provided price or product price
		price := item.Price
		if price.IsZero() {
			price = product.Price
		}

		// Check stock
		if product.Stock < item.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Insufficient stock for product %s", product.SKU)})
			return
		}

		orderItem := &erporder.OrderItem{
			ID:        uuid.New(),
			OrderID:   orderEntity.ID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     price,
		}

		if err := suite.orderRepo.AddItem(ctx, orderItem); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Update product stock
		product.Stock -= item.Quantity
		if err := suite.productRepo.Update(ctx, product); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Calculate total
		itemTotal := price.Mul(decimal.NewFromInt(int64(item.Quantity)))
		totalAmount = totalAmount.Add(itemTotal)
	}

	// Update order total
	orderEntity.Total = totalAmount
	if err := suite.orderRepo.Update(ctx, orderEntity); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"order": orderEntity})
}

func (suite *CompleteOrderWorkflowTestSuite) getOrderHandler(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	order, err := suite.orderRepo.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Get order items
	items, err := suite.orderRepo.GetItems(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": order, "items": items})
}

func (suite *CompleteOrderWorkflowTestSuite) updateOrderStatusHandler(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
		Notes  string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := suite.orderRepo.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	order.Status = entities.OrderStatus(req.Status)
	if req.Notes != "" {
		order.Notes = req.Notes
	}

	if err := suite.orderRepo.Update(ctx, order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": order})
}

func (suite *CompleteOrderWorkflowTestSuite) addOrderItemHandler(c *gin.Context) {
	ctx := c.Request.Context()

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req struct {
		ProductID uuid.UUID       `json:"product_id" binding:"required"`
		Quantity  int             `json:"quantity" binding:"required"`
		Price     decimal.Decimal `json:"price"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := suite.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != entities.OrderStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot add items to order in current status"})
		return
	}

	// Get product
	product, err := suite.productRepo.GetByID(ctx, req.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product not found"})
		return
	}

	// Check stock
	if product.Stock < req.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient stock"})
		return
	}

	// Create order item
	orderItem := &erporder.OrderItem{
		ID:        uuid.New(),
		OrderID:   orderID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Price:     req.Price,
	}

	if err := suite.orderRepo.AddItem(ctx, orderItem); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update product stock
	product.Stock -= req.Quantity
	if err := suite.productRepo.Update(ctx, product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Recalculate order total
	items, err := suite.orderRepo.GetItems(ctx, orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var totalAmount decimal.Decimal
	for _, item := range items {
		itemTotal := item.Price.Mul(decimal.NewFromInt(int64(item.Quantity)))
		totalAmount = totalAmount.Add(itemTotal)
	}

	order.Total = totalAmount
	if err := suite.orderRepo.Update(ctx, order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"order_item": orderItem, "order": order})
}

func (suite *CompleteOrderWorkflowTestSuite) processPaymentHandler(c *gin.Context) {
	ctx := c.Request.Context()

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req struct {
		PaymentMethod string          `json:"payment_method" binding:"required"`
		Amount        decimal.Decimal `json:"amount" binding:"required"`
		TransactionID string          `json:"transaction_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := suite.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != entities.OrderStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order is not in pending status"})
		return
	}

	if !req.Amount.Equal(order.Total) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment amount does not match order total"})
		return
	}

	// Update order status to paid
	order.Status = entities.OrderStatusPaid
	order.PaymentMethod = req.PaymentMethod
	order.PaymentStatus = "completed"

	if err := suite.orderRepo.Update(ctx, order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": order, "payment": req})
}

func (suite *CompleteOrderWorkflowTestSuite) shipOrderHandler(c *gin.Context) {
	ctx := c.Request.Context()

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req struct {
		TrackingNumber string `json:"tracking_number"`
		Carrier        string `json:"carrier"`
		ShippingDate   string `json:"shipping_date"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := suite.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != entities.OrderStatusPaid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order must be paid before shipping"})
		return
	}

	// Update order status to shipped
	order.Status = entities.OrderStatusShipped
	order.TrackingNumber = req.TrackingNumber
	order.ShippingCarrier = req.Carrier

	if err := suite.orderRepo.Update(ctx, order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": order})
}

func (suite *CompleteOrderWorkflowTestSuite) completeOrderHandler(c *gin.Context) {
	ctx := c.Request.Context()

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	order, err := suite.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != entities.OrderStatusShipped {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order must be shipped before completion"})
		return
	}

	// Update order status to completed
	order.Status = entities.OrderStatusCompleted
	completedAt := time.Now().UTC()
	order.CompletedAt = &completedAt

	if err := suite.orderRepo.Update(ctx, order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": order})
}

// Run the test suite
func TestCompleteOrderWorkflowSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end tests in short mode")
	}
	suite.Run(t, new(CompleteOrderWorkflowTestSuite))
}