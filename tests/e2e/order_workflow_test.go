//go:build e2e
// +build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"erpgo/tests/e2e/testutil"
)

type OrderWorkflowTestSuite struct {
	suite.Suite
	baseURL    string
	testUtil   *testutil.E2ETestUtil
	adminToken string
	userToken  string
}

func (suite *OrderWorkflowTestSuite) SetupSuite() {
	// Initialize test utilities
	suite.testUtil = testutil.NewE2ETestUtil(suite.T())
	suite.baseURL = "http://localhost:8080"

	// Wait for the API to be ready
	suite.testUtil.WaitForAPI(suite.baseURL, 30*time.Second)

	// Setup test data and authentication
	suite.setupAuthentication()
}

func (suite *OrderWorkflowTestSuite) TearDownSuite() {
	suite.testUtil.Cleanup()
}

func (suite *OrderWorkflowTestSuite) SetupTest() {
	// Clean up any test data before each test
	suite.testUtil.CleanupTestData()
}

func (suite *OrderWorkflowTestSuite) setupAuthentication() {
	// Create admin user and get token
	suite.adminToken = suite.testUtil.CreateAdminUser()

	// Create regular user and get token
	suite.userToken = suite.testUtil.CreateRegularUser()
}

func (suite *OrderWorkflowTestSuite) TestCompleteOrderWorkflow() {
	// This test covers the complete order workflow from creation to delivery
	// 1. Create customer
	// 2. Create products
	// 3. Create order
	// 4. Confirm order
	// 5. Process payment
	// 6. Ship order
	// 7. Deliver order
	// 8. Generate invoice

	customer := suite.createCustomer()
	products := suite.createProducts()
	order := suite.createOrder(customer.ID, products)
	suite.confirmOrder(order.ID)
	suite.processPayment(order.ID)
	suite.shipOrder(order.ID)
	suite.deliverOrder(order.ID)
	invoice := suite.generateInvoice(order.ID)

	// Verify final state
	suite.verifyOrderState(order.ID, "delivered")
	suite.verifyInvoiceState(invoice.ID, "paid")
}

func (suite *OrderWorkflowTestSuite) TestOrderCancellationWorkflow() {
	// Test order cancellation at different stages
	customer := suite.createCustomer()
	products := suite.createProducts()
	order := suite.createOrder(customer.ID, products)

	// Cancel pending order
	suite.cancelOrder(order.ID)
	suite.verifyOrderState(order.ID, "cancelled")

	// Verify that inventory is restored
	suite.verifyInventoryRestored(products[0].ID, 5) // Original quantity was 5
}

func (suite *OrderWorkflowTestSuite) TestOrderWithInsufficientInventory() {
	// Test order creation when inventory is insufficient
	customer := suite.createCustomer()
	product := suite.createProductWithInventory(1) // Only 1 item in stock

	// Try to create order with more items than available
	orderRequest := map[string]interface{}{
		"customer_id": customer.ID.String(),
		"items": []map[string]interface{}{
			{
				"product_id": product.ID.String(),
				"quantity":   5, // More than available
			},
		},
	}

	resp, err := suite.makeAuthenticatedRequest("POST", "/api/v1/orders", orderRequest, suite.userToken)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), response["error"], "insufficient inventory")
}

func (suite *OrderWorkflowTestSuite) TestCustomerOrderHistory() {
	// Test retrieving customer order history
	customer := suite.createCustomer()
	products := suite.createProducts()

	// Create multiple orders
	order1 := suite.createOrder(customer.ID, products[:1])
	order2 := suite.createOrder(customer.ID, products[1:])

	// Retrieve order history
	resp, err := suite.makeAuthenticatedRequest(
		fmt.Sprintf("GET", "/api/v1/customers/%s/orders", customer.ID.String()),
		nil,
		suite.userToken,
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), response["success"].(bool))
	data := response["data"].(map[string]interface{})
	orders := data["items"].([]interface{})
	assert.Len(suite.T(), orders, 2)
}

func (suite *OrderWorkflowTestSuite) createCustomer() *Customer {
	customerRequest := map[string]interface{}{
		"customer_code":   fmt.Sprintf("CUST-%d", time.Now().Unix()),
		"name":            "Test Customer",
		"email":           fmt.Sprintf("customer-%d@example.com", time.Now().Unix()),
		"phone":           "+1234567890",
		"billing_address": "123 Test St, Test City, TC 12345",
		"credit_limit":    10000.00,
	}

	resp, err := suite.makeAuthenticatedRequest("POST", "/api/v1/customers", customerRequest, suite.adminToken)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	data := response["data"].(map[string]interface{})
	return &Customer{
		ID:           parseUUID(data["id"].(string)),
		Name:         data["name"].(string),
		Email:        data["email"].(string),
		CustomerCode: data["customer_code"].(string),
	}
}

func (suite *OrderWorkflowTestSuite) createProducts() []*Product {
	products := make([]*Product, 2)

	for i := 0; i < 2; i++ {
		product := suite.createProductWithInventory(10 + i*5)
		products[i] = product
	}

	return products
}

func (suite *OrderWorkflowTestSuite) createProductWithInventory(quantity int) *Product {
	productRequest := map[string]interface{}{
		"sku":         fmt.Sprintf("PROD-%d", time.Now().UnixNano()),
		"name":        fmt.Sprintf("Test Product %d", time.Now().Unix()),
		"description": "Test product description",
		"price":       99.99,
		"cost":        50.00,
		"is_active":   true,
	}

	resp, err := suite.makeAuthenticatedRequest("POST", "/api/v1/products", productRequest, suite.adminToken)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	data := response["data"].(map[string]interface{})
	productID := parseUUID(data["id"].(string))

	// Set up inventory for the product
	inventoryRequest := map[string]interface{}{
		"product_id":         productID.String(),
		"warehouse_id":       suite.testUtil.GetDefaultWarehouseID().String(),
		"quantity_available": quantity,
		"reorder_level":      5,
	}

	resp, err = suite.makeAuthenticatedRequest("POST", "/api/v1/inventory/set", inventoryRequest, suite.adminToken)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	return &Product{
		ID:    productID,
		Name:  data["name"].(string),
		SKU:   data["sku"].(string),
		Price: data["price"].(float64),
	}
}

func (suite *OrderWorkflowTestSuite) createOrder(customerID uuid.UUID, products []*Product) *Order {
	orderItems := make([]map[string]interface{}, len(products))
	for i, product := range products {
		orderItems[i] = map[string]interface{}{
			"product_id": product.ID.String(),
			"quantity":   2,
		}
	}

	orderRequest := map[string]interface{}{
		"customer_id": customerID.String(),
		"items":       orderItems,
	}

	resp, err := suite.makeAuthenticatedRequest("POST", "/api/v1/orders", orderRequest, suite.userToken)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	data := response["data"].(map[string]interface{})
	return &Order{
		ID:          parseUUID(data["id"].(string)),
		CustomerID:  customerID,
		Status:      data["status"].(string),
		TotalAmount: data["total_amount"].(float64),
	}
}

func (suite *OrderWorkflowTestSuite) confirmOrder(orderID uuid.UUID) {
	resp, err := suite.makeAuthenticatedRequest(
		fmt.Sprintf("POST", "/api/v1/orders/%s/confirm", orderID.String()),
		nil,
		suite.userToken,
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

func (suite *OrderWorkflowTestSuite) processPayment(orderID uuid.UUID) {
	paymentRequest := map[string]interface{}{
		"method":      "credit_card",
		"amount":      100.00, // This should match the order total
		"card_number": "4111111111111111",
		"expiry":      "12/25",
		"cvv":         "123",
	}

	resp, err := suite.makeAuthenticatedRequest(
		fmt.Sprintf("POST", "/api/v1/orders/%s/pay", orderID.String()),
		paymentRequest,
		suite.userToken,
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

func (suite *OrderWorkflowTestSuite) shipOrder(orderID uuid.UUID) {
	shippingRequest := map[string]interface{}{
		"tracking_number": "TRACK123456",
		"carrier":         "UPS",
		"shipped_at":      time.Now().Format(time.RFC3339),
	}

	resp, err := suite.makeAuthenticatedRequest(
		fmt.Sprintf("POST", "/api/v1/orders/%s/ship", orderID.String()),
		shippingRequest,
		suite.userToken,
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

func (suite *OrderWorkflowTestSuite) deliverOrder(orderID uuid.UUID) {
	deliveryRequest := map[string]interface{}{
		"delivered_at": time.Now().Format(time.RFC3339),
		"notes":        "Delivered to front desk",
	}

	resp, err := suite.makeAuthenticatedRequest(
		fmt.Sprintf("POST", "/api/v1/orders/%s/deliver", orderID.String()),
		deliveryRequest,
		suite.userToken,
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

func (suite *OrderWorkflowTestSuite) generateInvoice(orderID uuid.UUID) *Invoice {
	resp, err := suite.makeAuthenticatedRequest(
		fmt.Sprintf("POST", "/api/v1/orders/%s/invoice", orderID.String()),
		nil,
		suite.userToken,
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	data := response["data"].(map[string]interface{})
	return &Invoice{
		ID:            parseUUID(data["id"].(string)),
		OrderID:       &orderID,
		InvoiceNumber: data["invoice_number"].(string),
		TotalAmount:   data["total_amount"].(float64),
		Status:        data["status"].(string),
	}
}

func (suite *OrderWorkflowTestSuite) cancelOrder(orderID uuid.UUID) {
	resp, err := suite.makeAuthenticatedRequest(
		fmt.Sprintf("DELETE", "/api/v1/orders/%s", orderID.String()),
		nil,
		suite.userToken,
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

func (suite *OrderWorkflowTestSuite) verifyOrderState(orderID uuid.UUID, expectedStatus string) {
	resp, err := suite.makeAuthenticatedRequest(
		fmt.Sprintf("GET", "/api/v1/orders/%s", orderID.String()),
		nil,
		suite.userToken,
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	data := response["data"].(map[string]interface{})
	assert.Equal(suite.T(), expectedStatus, data["status"].(string))
}

func (suite *OrderWorkflowTestSuite) verifyInvoiceState(invoiceID uuid.UUID, expectedStatus string) {
	resp, err := suite.makeAuthenticatedRequest(
		fmt.Sprintf("GET", "/api/v1/invoices/%s", invoiceID.String()),
		nil,
		suite.userToken,
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	data := response["data"].(map[string]interface{})
	assert.Equal(suite.T(), expectedStatus, data["status"].(string))
}

func (suite *OrderWorkflowTestSuite) verifyInventoryRestored(productID uuid.UUID, expectedQuantity int) {
	resp, err := suite.makeAuthenticatedRequest(
		fmt.Sprintf("GET", "/api/v1/products/%s/inventory", productID.String()),
		nil,
		suite.adminToken,
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	data := response["data"].(map[string]interface{})
	assert.Equal(suite.T(), expectedQuantity, int(data["quantity_available"].(float64)))
}

func (suite *OrderWorkflowTestSuite) makeAuthenticatedRequest(method, endpoint string, body interface{}, token string) (*http.Response, error) {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, suite.baseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return http.DefaultClient.Do(req)
}

// Helper types
type Customer struct {
	ID           uuid.UUID
	Name         string
	Email        string
	CustomerCode string
}

type Product struct {
	ID    uuid.UUID
	Name  string
	SKU   string
	Price float64
}

type Order struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	Status      string
	TotalAmount float64
}

type Invoice struct {
	ID            uuid.UUID
	OrderID       *uuid.UUID
	InvoiceNumber string
	TotalAmount   float64
	Status        string
}

func parseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err) // This should not happen in tests
	}
	return id
}

// Test runner
func TestOrderWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(OrderWorkflowTestSuite))
}
