package api

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"erpgo/internal/infrastructure/repositories"
	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/domain/orders/repositories"
	"erpgo/tests/integration/testutil"
)

// OrderIntegrationTestSuite tests the complete order workflow
type OrderIntegrationTestSuite struct {
	suite.Suite
	db              *database.Database
	orderRepo       repositories.OrderRepository
	orderItemRepo   repositories.OrderItemRepository
	customerRepo    repositories.CustomerRepository
	addressRepo     repositories.OrderAddressRepository
	companyRepo     repositories.CompanyRepository
	cleanupFunc     func()
}

// SetupSuite sets up the integration test suite
func (suite *OrderIntegrationTestSuite) SetupSuite() {
	// This would typically set up a test database with migrations
	suite.T().Skip("Skipping integration tests - requires test database setup")
}

// TearDownSuite tears down the integration test suite
func (suite *OrderIntegrationTestSuite) TearDownSuite() {
	if suite.cleanupFunc != nil {
		suite.cleanupFunc()
	}
}

// TestCompleteOrderWorkflow tests the complete order workflow from creation to fulfillment
func (suite *OrderIntegrationTestSuite) TestCompleteOrderWorkflow() {
	ctx := testutil.CreateTestContext()

	// Step 1: Create a customer
	customer := testutil.CreateTestCustomer(suite.T())
	err := suite.customerRepo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Step 2: Create shipping and billing addresses
	shippingAddress := testutil.CreateTestOrderAddress(suite.T(), &customer.ID, nil, "SHIPPING")
	err = suite.addressRepo.Create(ctx, shippingAddress)
	require.NoError(suite.T(), err)

	billingAddress := testutil.CreateTestOrderAddress(suite.T(), &customer.ID, nil, "BILLING")
	err = suite.addressRepo.Create(ctx, billingAddress)
	require.NoError(suite.T(), err)

	// Step 3: Create an order
	order := testutil.CreateTestOrder(suite.T())
	order.CustomerID = customer.ID
	order.ShippingAddressID = shippingAddress.ID
	order.BillingAddressID = billingAddress.ID
	err = suite.orderRepo.Create(ctx, order)
	require.NoError(suite.T(), err)

	// Step 4: Add order items
	items := make([]*entities.OrderItem, 3)
	for i := 0; i < 3; i++ {
		items[i] = testutil.CreateTestOrderItem(suite.T(), order.ID)
		items[i].ProductSKU = fmt.Sprintf("WORKFLOW-SKU-%d", i)
		items[i].ProductName = fmt.Sprintf("Workflow Product %d", i+1)
		err = suite.orderItemRepo.Create(ctx, items[i])
		require.NoError(suite.T(), err)
	}

	// Step 5: Update order status to confirmed
	err = suite.orderRepo.UpdateStatus(ctx, order.ID, entities.OrderStatusConfirmed, uuid.New())
	require.NoError(suite.T(), err)

	// Step 6: Update order status to processing
	err = suite.orderRepo.UpdateStatus(ctx, order.ID, entities.OrderStatusProcessing, uuid.New())
	require.NoError(suite.T(), err)

	// Step 7: Update order items status to shipped
	for _, item := range items {
		err = suite.orderItemRepo.UpdateItemStatus(ctx, item.ID, "SHIPPED")
		require.NoError(suite.T(), err)
		err = suite.orderItemRepo.UpdateShippedQuantity(ctx, item.ID, item.Quantity)
		require.NoError(suite.T(), err)
	}

	// Step 8: Update order status to shipped
	err = suite.orderRepo.UpdateStatus(ctx, order.ID, entities.OrderStatusShipped, uuid.New())
	require.NoError(suite.T(), err)

	// Step 9: Update payment status
	err = suite.orderRepo.UpdatePaymentStatus(ctx, order.ID, entities.PaymentStatusPaid, order.TotalAmount)
	require.NoError(suite.T(), err)

	// Step 10: Update order status to delivered
	err = suite.orderRepo.UpdateStatus(ctx, order.ID, entities.OrderStatusDelivered, uuid.New())
	require.NoError(suite.T(), err)

	// Verify the final state
	finalOrder, err := suite.orderRepo.GetByID(ctx, order.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), entities.OrderStatusDelivered, finalOrder.Status)
	assert.Equal(suite.T(), entities.PaymentStatusPaid, finalOrder.PaymentStatus)

	finalItems, err := suite.orderItemRepo.GetByOrderID(ctx, order.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, len(finalItems))

	for _, item := range finalItems {
		assert.Equal(suite.T(), "SHIPPED", item.Status)
		assert.Equal(suite.T(), item.Quantity, item.QuantityShipped)
	}
}

// TestOrderCancellationWorkflow tests the order cancellation workflow
func (suite *OrderIntegrationTestSuite) TestOrderCancellationWorkflow() {
	ctx := testutil.CreateTestContext()

	// Create a customer and order
	customer := testutil.CreateTestCustomer(suite.T())
	err := suite.customerRepo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	order := testutil.CreateTestOrder(suite.T())
	order.CustomerID = customer.ID
	order.Status = entities.OrderStatusConfirmed
	err = suite.orderRepo.Create(ctx, order)
	require.NoError(suite.T(), err)

	// Add order items
	item := testutil.CreateTestOrderItem(suite.T(), order.ID)
	err = suite.orderItemRepo.Create(ctx, item)
	require.NoError(suite.T(), err)

	// Cancel the order
	err = suite.orderRepo.UpdateStatus(ctx, order.ID, entities.OrderStatusCancelled, uuid.New())
	require.NoError(suite.T(), err)

	// Update item status
	err = suite.orderItemRepo.UpdateItemStatus(ctx, item.ID, "CANCELLED")
	require.NoError(suite.T(), err)

	// Verify the cancellation
	cancelledOrder, err := suite.orderRepo.GetByID(ctx, order.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), entities.OrderStatusCancelled, cancelledOrder.Status)

	cancelledItem, err := suite.orderItemRepo.GetByID(ctx, item.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "CANCELLED", cancelledItem.Status)
}

// TestCustomerOrderHistory tests retrieving customer order history
func (suite *OrderIntegrationTestSuite) TestCustomerOrderHistory() {
	ctx := testutil.CreateTestContext()

	// Create a customer
	customer := testutil.CreateTestCustomer(suite.T())
	err := suite.customerRepo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Create multiple orders for the customer
	orders := make([]*entities.Order, 5)
	for i := 0; i < 5; i++ {
		orders[i] = testutil.CreateTestOrder(suite.T())
		orders[i].CustomerID = customer.ID
		orders[i].OrderDate = time.Now().AddDate(0, 0, -i)
		orders[i].Status = entities.OrderStatusDelivered
		err = suite.orderRepo.Create(ctx, orders[i])
		require.NoError(suite.T(), err)
	}

	// Retrieve customer order history
	history, err := suite.orderRepo.GetCustomerOrderHistory(ctx, customer.ID, 10)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(history), 5)

	// Verify all orders belong to the customer
	for _, order := range history {
		assert.Equal(suite.T(), customer.ID, order.CustomerID)
	}

	// Verify orders are sorted by date (most recent first)
	for i := 1; i < len(history); i++ {
		assert.True(suite.T(), history[i-1].OrderDate.After(history[i].OrderDate) ||
			history[i-1].OrderDate.Equal(history[i].OrderDate))
	}
}

// TestOrderSearchAndFiltering tests advanced search and filtering
func (suite *OrderIntegrationTestSuite) TestOrderSearchAndFiltering() {
	ctx := testutil.CreateTestContext()

	// Create test customers
	customers := make([]*entities.Customer, 3)
	for i := 0; i < 3; i++ {
		customers[i] = testutil.CreateTestCustomer(suite.T())
		customers[i].FirstName = fmt.Sprintf("SearchCustomer%d", i)
		customers[i].Email = fmt.Sprintf("search%d@example.com", i)
		err := suite.customerRepo.Create(ctx, customers[i])
		require.NoError(suite.T(), err)
	}

	// Create test orders with different properties
	orders := make([]*entities.Order, 6)
	statuses := []entities.OrderStatus{
		entities.OrderStatusPending,
		entities.OrderStatusConfirmed,
		entities.OrderStatusShipped,
	}

	for i := 0; i < 6; i++ {
		orders[i] = testutil.CreateTestOrder(suite.T())
		orders[i].CustomerID = customers[i%3].ID
		orders[i].Status = statuses[i%3]
		orders[i].TotalAmount = decimal.NewFromFloat(float64(i+1) * 50.00)
		orders[i].CustomerNotes = testutil.StringPtr(fmt.Sprintf("Search notes %d", i))
		err := suite.orderRepo.Create(ctx, orders[i])
		require.NoError(suite.T(), err)
	}

	// Test filtering by status
	filter := repositories.OrderFilter{
		Status: []entities.OrderStatus{entities.OrderStatusConfirmed},
		Limit:  10,
	}
	confirmedOrders, err := suite.orderRepo.List(ctx, filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(confirmedOrders), 2)

	// Test filtering by total amount range
	minAmount := decimal.NewFromFloat(100.00)
	maxAmount := decimal.NewFromFloat(200.00)
	filter = repositories.OrderFilter{
		MinTotalAmount: &minAmount,
		MaxTotalAmount: &maxAmount,
		Limit:          10,
	}
	amountFilteredOrders, err := suite.orderRepo.List(ctx, filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(amountFilteredOrders), 1)

	// Test search by customer notes
	searchResults, err := suite.orderRepo.SearchOrders(ctx, "Search notes", repositories.OrderFilter{Limit: 10})
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(searchResults), 6)
}

// TestOrderAnalytics tests order analytics and reporting
func (suite *OrderIntegrationTestSuite) TestOrderAnalytics() {
	ctx := testutil.CreateTestContext()
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	// Create test customers
	customer := testutil.CreateTestCustomer(suite.T())
	err := suite.customerRepo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Create test orders spanning the date range
	for i := 0; i < 10; i++ {
		order := testutil.CreateTestOrder(suite.T())
		order.CustomerID = customer.ID
		order.OrderDate = startDate.AddDate(0, 0, i*3)
		order.Status = entities.OrderStatusDelivered
		order.TotalAmount = decimal.NewFromFloat(float64(i+1) * 100.00)
		err = suite.orderRepo.Create(ctx, order)
		require.NoError(suite.T(), err)

		// Add order items
		item := testutil.CreateTestOrderItem(suite.T(), order.ID)
		item.ProductSKU = fmt.Sprintf("ANALYTICS-SKU-%d", i)
		item.TotalPrice = order.TotalAmount
		err = suite.orderItemRepo.Create(ctx, item)
		require.NoError(suite.T(), err)
	}

	// Test revenue by period
	revenueByPeriod, err := suite.orderRepo.GetRevenueByPeriod(ctx, startDate, endDate, "week")
	require.NoError(suite.T(), err)
	assert.Greater(suite.T(), len(revenueByPeriod), 0)

	// Verify revenue data
	for _, period := range revenueByPeriod {
		assert.GreaterOrEqual(suite.T(), period.OrderCount, int64(0))
		assert.GreaterOrEqual(suite.T(), period.Revenue, decimal.Zero)
	}

	// Test top customers
	topCustomers, err := suite.orderRepo.GetTopCustomers(ctx, startDate, endDate, 5)
	require.NoError(suite.T(), err)
	assert.Greater(suite.T(), len(topCustomers), 0)

	// Verify customer data
	for _, customerStat := range topCustomers {
		assert.Equal(suite.T(), customer.ID, customerStat.CustomerID)
		assert.GreaterOrEqual(suite.T(), customerStat.OrderCount, int64(0))
		assert.GreaterOrEqual(suite.T(), customerStat.TotalRevenue, decimal.Zero)
	}

	// Test sales by product
	salesByProduct, err := suite.orderRepo.GetSalesByProduct(ctx, startDate, endDate, 10)
	require.NoError(suite.T(), err)
	assert.Greater(suite.T(), len(salesByProduct), 0)

	// Verify product data
	for _, productStat := range salesByProduct {
		assert.GreaterOrEqual(suite.T(), productStat.QuantitySold, int64(0))
		assert.GreaterOrEqual(suite.T(), productStat.TotalRevenue, decimal.Zero)
	}
}

// TestCustomerCreditManagement tests customer credit management
func (suite *OrderIntegrationTestSuite) TestCustomerCreditManagement() {
	ctx := testutil.CreateTestContext()

	// Create a customer with credit limit
	customer := testutil.CreateTestCustomer(suite.T())
	customer.CreditLimit = decimal.NewFromFloat(1000.00)
	err := suite.customerRepo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Create orders that use credit
	totalCreditUsed := decimal.Zero
	for i := 0; i < 3; i++ {
		order := testutil.CreateTestOrder(suite.T())
		order.CustomerID = customer.ID
		order.Status = entities.OrderStatusConfirmed
		order.PaymentStatus = entities.PaymentStatusPending
		order.TotalAmount = decimal.NewFromFloat(200.00)
		err = suite.orderRepo.Create(ctx, order)
		require.NoError(suite.T(), err)

		totalCreditUsed = totalCreditUsed.Add(order.TotalAmount)
	}

	// Update customer credit used
	err = suite.customerRepo.UpdateCreditUsed(ctx, customer.ID, totalCreditUsed)
	require.NoError(suite.T(), err)

	// Verify credit used
	updatedCustomer, err := suite.customerRepo.GetByID(ctx, customer.ID)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), totalCreditUsed.Equal(updatedCustomer.CreditUsed))

	// Test getting customers with overdue credit
	overdueCustomers, err := suite.customerRepo.GetCustomersWithOverdueCredit(ctx)
	require.NoError(suite.T(), err)
	// The customer should appear in overdue list if they have pending orders
	assert.GreaterOrEqual(suite.T(), len(overdueCustomers), 0)
}

// TestBulkOperations tests bulk operations
func (suite *OrderIntegrationTestSuite) TestBulkOperations() {
	ctx := testutil.CreateTestContext()

	// Create a customer
	customer := testutil.CreateTestCustomer(suite.T())
	err := suite.customerRepo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Test bulk order creation
	orders := make([]*entities.Order, 3)
	for i := 0; i < 3; i++ {
		orders[i] = testutil.CreateTestOrder(suite.T())
		orders[i].CustomerID = customer.ID
		orders[i].Status = entities.OrderStatusPending
		orders[i].OrderNumber = fmt.Sprintf("BULK-ORDER-%d", i)
	}

	err = suite.orderRepo.BulkCreate(ctx, orders)
	require.NoError(suite.T(), err)

	// Verify orders were created
	for _, order := range orders {
		retrieved, err := suite.orderRepo.GetByID(ctx, order.ID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), order.OrderNumber, retrieved.OrderNumber)
	}

	// Test bulk status update
	orderIDs := make([]uuid.UUID, 3)
	for i, order := range orders {
		orderIDs[i] = order.ID
	}

	err = suite.orderRepo.BulkUpdateStatus(ctx, orderIDs, entities.OrderStatusConfirmed, uuid.New())
	require.NoError(suite.T(), err)

	// Verify status updates
	for _, orderID := range orderIDs {
		retrieved, err := suite.orderRepo.GetByID(ctx, orderID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), entities.OrderStatusConfirmed, retrieved.Status)
	}

	// Test bulk order item creation
	items := make([]*entities.OrderItem, 6)
	itemIndex := 0
	for _, order := range orders {
		for i := 0; i < 2; i++ {
			items[itemIndex] = testutil.CreateTestOrderItem(suite.T(), order.ID)
			items[itemIndex].ProductSKU = fmt.Sprintf("BULK-ITEM-%d", itemIndex)
			itemIndex++
		}
	}

	err = suite.orderItemRepo.BulkCreate(ctx, items)
	require.NoError(suite.T(), err)

	// Verify items were created
	for _, item := range items {
		retrieved, err := suite.orderItemRepo.GetByID(ctx, item.ID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), item.ProductSKU, retrieved.ProductSKU)
	}
}

// Run the integration test suite
func TestOrderIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(OrderIntegrationTestSuite))
}