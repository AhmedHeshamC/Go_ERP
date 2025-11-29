package repositories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/domain/orders/repositories"
	"erpgo/tests/integration/testutil"
)

// OrderRepositoryTestSuite contains all tests for the order repository
type OrderRepositoryTestSuite struct {
	suite.Suite
	repo        repositories.OrderRepository
	cleanupFunc func()
}

// SetupSuite sets up the test suite
func (suite *OrderRepositoryTestSuite) SetupSuite() {
	// This would typically set up a test database
	// For now, we'll skip the actual database setup
	suite.T().Skip("Skipping database tests - requires test database setup")
}

// TearDownSuite tears down the test suite
func (suite *OrderRepositoryTestSuite) TearDownSuite() {
	if suite.cleanupFunc != nil {
		suite.cleanupFunc()
	}
}

// TestCreateOrder tests creating a new order
func (suite *OrderRepositoryTestSuite) TestCreateOrder() {
	ctx := testutil.CreateTestContext()
	order := testutil.CreateTestOrder(suite.T())

	err := suite.repo.Create(ctx, order)
	require.NoError(suite.T(), err)

	// Verify the order was created
	retrieved, err := suite.repo.GetByID(ctx, order.ID)
	require.NoError(suite.T(), err)
	testutil.AssertOrdersEqual(suite.T(), order, retrieved)
}

// TestGetByID tests retrieving an order by ID
func (suite *OrderRepositoryTestSuite) TestGetByID() {
	ctx := testutil.CreateTestContext()
	order := testutil.CreateTestOrder(suite.T())

	// Create the order first
	err := suite.repo.Create(ctx, order)
	require.NoError(suite.T(), err)

	// Retrieve the order
	retrieved, err := suite.repo.GetByID(ctx, order.ID)
	require.NoError(suite.T(), err)
	testutil.AssertOrdersEqual(suite.T(), order, retrieved)
}

// TestGetByNonExistentID tests retrieving a non-existent order
func (suite *OrderRepositoryTestSuite) TestGetByNonExistentID() {
	ctx := testutil.CreateTestContext()
	nonExistentID := uuid.New()

	_, err := suite.repo.GetByID(ctx, nonExistentID)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not found")
}

// TestGetByOrderNumber tests retrieving an order by order number
func (suite *OrderRepositoryTestSuite) TestGetByOrderNumber() {
	ctx := testutil.CreateTestContext()
	order := testutil.CreateTestOrder(suite.T())

	// Create the order first
	err := suite.repo.Create(ctx, order)
	require.NoError(suite.T(), err)

	// Retrieve the order by number
	retrieved, err := suite.repo.GetByOrderNumber(ctx, order.OrderNumber)
	require.NoError(suite.T(), err)
	testutil.AssertOrdersEqual(suite.T(), order, retrieved)
}

// TestUpdateOrder tests updating an order
func (suite *OrderRepositoryTestSuite) TestUpdateOrder() {
	ctx := testutil.CreateTestContext()
	order := testutil.CreateTestOrder(suite.T())

	// Create the order first
	err := suite.repo.Create(ctx, order)
	require.NoError(suite.T(), err)

	// Update the order
	order.Status = entities.OrderStatusConfirmed
	notes := "Updated notes"
	order.Notes = &notes
	order.UpdatedAt = time.Now()

	err = suite.repo.Update(ctx, order)
	require.NoError(suite.T(), err)

	// Verify the update
	retrieved, err := suite.repo.GetByID(ctx, order.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), entities.OrderStatusConfirmed, retrieved.Status)
	assert.Equal(suite.T(), "Updated notes", *retrieved.Notes)
}

// TestDeleteOrder tests deleting an order
func (suite *OrderRepositoryTestSuite) TestDeleteOrder() {
	ctx := testutil.CreateTestContext()
	order := testutil.CreateTestOrder(suite.T())

	// Create the order first
	err := suite.repo.Create(ctx, order)
	require.NoError(suite.T(), err)

	// Delete the order
	err = suite.repo.Delete(ctx, order.ID)
	require.NoError(suite.T(), err)

	// Verify the order was deleted
	_, err = suite.repo.GetByID(ctx, order.ID)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not found")
}

// TestListOrders tests listing orders with filters
func (suite *OrderRepositoryTestSuite) TestListOrders() {
	ctx := testutil.CreateTestContext()

	// Create test orders
	orders := make([]*entities.Order, 5)
	for i := 0; i < 5; i++ {
		orders[i] = testutil.CreateTestOrder(suite.T())
		if i%2 == 0 {
			orders[i].Status = entities.OrderStatusPending
		} else {
			orders[i].Status = entities.OrderStatusConfirmed
		}
		err := suite.repo.Create(ctx, orders[i])
		require.NoError(suite.T(), err)
	}

	// Test listing all orders
	filter := repositories.OrderFilter{
		Limit: 10,
	}
	allOrders, err := suite.repo.List(ctx, filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(allOrders), 5)

	// Test filtering by status
	filter.Status = []entities.OrderStatus{entities.OrderStatusPending}
	pendingOrders, err := suite.repo.List(ctx, filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(pendingOrders), 2)
}

// TestCountOrders tests counting orders with filters
func (suite *OrderRepositoryTestSuite) TestCountOrders() {
	ctx := testutil.CreateTestContext()

	// Create test orders
	for i := 0; i < 3; i++ {
		order := testutil.CreateTestOrder(suite.T())
		order.Status = entities.OrderStatusPending
		err := suite.repo.Create(ctx, order)
		require.NoError(suite.T(), err)
	}

	// Test counting all orders
	filter := repositories.OrderFilter{}
	count, err := suite.repo.Count(ctx, filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), count, 3)

	// Test counting by status
	filter.Status = []entities.OrderStatus{entities.OrderStatusPending}
	pendingCount, err := suite.repo.Count(ctx, filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), pendingCount, 3)
}

// TestGetByStatus tests retrieving orders by status
func (suite *OrderRepositoryTestSuite) TestGetByStatus() {
	ctx := testutil.CreateTestContext()

	// Create test orders
	for i := 0; i < 3; i++ {
		order := testutil.CreateTestOrder(suite.T())
		order.Status = entities.OrderStatusShipped
		err := suite.repo.Create(ctx, order)
		require.NoError(suite.T(), err)
	}

	// Retrieve shipped orders
	orders, err := suite.repo.GetByStatus(ctx, entities.OrderStatusShipped)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(orders), 3)

	// Verify all returned orders have the correct status
	for _, order := range orders {
		assert.Equal(suite.T(), entities.OrderStatusShipped, order.Status)
	}
}

// TestUpdateStatus tests updating order status
func (suite *OrderRepositoryTestSuite) TestUpdateStatus() {
	ctx := testutil.CreateTestContext()
	order := testutil.CreateTestOrder(suite.T())
	order.Status = entities.OrderStatusPending

	// Create the order first
	err := suite.repo.Create(ctx, order)
	require.NoError(suite.T(), err)

	// Update the status
	err = suite.repo.UpdateStatus(ctx, order.ID, entities.OrderStatusConfirmed, uuid.New())
	require.NoError(suite.T(), err)

	// Verify the status update
	retrieved, err := suite.repo.GetByID(ctx, order.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), entities.OrderStatusConfirmed, retrieved.Status)
	assert.Equal(suite.T(), entities.OrderStatusPending, *retrieved.PreviousStatus)
}

// TestGetByCustomerID tests retrieving orders by customer ID
func (suite *OrderRepositoryTestSuite) TestGetByCustomerID() {
	ctx := testutil.CreateTestContext()
	customerID := uuid.New()

	// Create test orders for the customer
	for i := 0; i < 3; i++ {
		order := testutil.CreateTestOrder(suite.T())
		order.CustomerID = customerID
		err := suite.repo.Create(ctx, order)
		require.NoError(suite.T(), err)
	}

	// Retrieve orders for the customer
	filter := repositories.OrderFilter{
		CustomerID: &customerID,
		Limit:      10,
	}
	orders, err := suite.repo.GetByCustomerID(ctx, customerID, filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(orders), 3)

	// Verify all returned orders belong to the customer
	for _, order := range orders {
		assert.Equal(suite.T(), customerID, order.CustomerID)
	}
}

// TestGetByDateRange tests retrieving orders within a date range
func (suite *OrderRepositoryTestSuite) TestGetByDateRange() {
	ctx := testutil.CreateTestContext()
	now := time.Now()
	startDate := now.AddDate(0, 0, -7)
	endDate := now.AddDate(0, 0, 7)

	// Create test orders within the date range
	for i := 0; i < 3; i++ {
		order := testutil.CreateTestOrder(suite.T())
		order.OrderDate = now.AddDate(0, 0, -i)
		err := suite.repo.Create(ctx, order)
		require.NoError(suite.T(), err)
	}

	// Retrieve orders within the date range
	filter := repositories.OrderFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     10,
	}
	orders, err := suite.repo.GetByDateRange(ctx, startDate, endDate, filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(orders), 3)
}

// TestGetUnpaidOrders tests retrieving unpaid orders
func (suite *OrderRepositoryTestSuite) TestGetUnpaidOrders() {
	ctx := testutil.CreateTestContext()

	// Create test orders with different payment statuses
	paymentStatuses := []entities.PaymentStatus{
		entities.PaymentStatusPending,
		entities.PaymentStatusPartiallyPaid,
		entities.PaymentStatusPaid,
	}

	for _, status := range paymentStatuses {
		order := testutil.CreateTestOrder(suite.T())
		order.PaymentStatus = status
		err := suite.repo.Create(ctx, order)
		require.NoError(suite.T(), err)
	}

	// Retrieve unpaid orders
	orders, err := suite.repo.GetUnpaidOrders(ctx)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(orders), 2)

	// Verify all returned orders are unpaid
	for _, order := range orders {
		assert.NotEqual(suite.T(), entities.PaymentStatusPaid, order.PaymentStatus)
	}
}

// TestBulkUpdateStatus tests bulk updating order statuses
func (suite *OrderRepositoryTestSuite) TestBulkUpdateStatus() {
	ctx := testutil.CreateTestContext()

	// Create test orders
	orderIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		order := testutil.CreateTestOrder(suite.T())
		order.Status = entities.OrderStatusPending
		err := suite.repo.Create(ctx, order)
		require.NoError(suite.T(), err)
		orderIDs[i] = order.ID
	}

	// Bulk update statuses
	err := suite.repo.BulkUpdateStatus(ctx, orderIDs, entities.OrderStatusConfirmed, uuid.New())
	require.NoError(suite.T(), err)

	// Verify the updates
	for _, orderID := range orderIDs {
		retrieved, err := suite.repo.GetByID(ctx, orderID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), entities.OrderStatusConfirmed, retrieved.Status)
	}
}

// TestBulkCreate tests bulk creating orders
func (suite *OrderRepositoryTestSuite) TestBulkCreate() {
	ctx := testutil.CreateTestContext()

	// Create test orders
	orders := make([]*entities.Order, 3)
	for i := 0; i < 3; i++ {
		orders[i] = testutil.CreateTestOrder(suite.T())
	}

	// Bulk create orders
	err := suite.repo.BulkCreate(ctx, orders)
	require.NoError(suite.T(), err)

	// Verify the orders were created
	for _, order := range orders {
		retrieved, err := suite.repo.GetByID(ctx, order.ID)
		require.NoError(suite.T(), err)
		testutil.AssertOrdersEqual(suite.T(), order, retrieved)
	}
}

// TestGetOrderStats tests retrieving order statistics
func (suite *OrderRepositoryTestSuite) TestGetOrderStats() {
	ctx := testutil.CreateTestContext()
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	// Create test orders
	for i := 0; i < 5; i++ {
		order := testutil.CreateTestOrder(suite.T())
		order.OrderDate = time.Now().AddDate(0, 0, -i)
		order.Status = entities.OrderStatusConfirmed
		err := suite.repo.Create(ctx, order)
		require.NoError(suite.T(), err)
	}

	// Retrieve order statistics
	filter := repositories.OrderStatsFilter{
		StartDate: startDate,
		EndDate:   endDate,
	}
	stats, err := suite.repo.GetOrderStats(ctx, filter)
	require.NoError(suite.T(), err)
	assert.Greater(suite.T(), stats.TotalOrders, int64(0))
	assert.GreaterOrEqual(suite.T(), stats.TotalRevenue, decimal.Zero)
	assert.GreaterOrEqual(suite.T(), stats.AverageOrderValue, decimal.Zero)
	assert.NotNil(suite.T(), stats.StatusCounts)
	assert.NotNil(suite.T(), stats.PaymentStatusCounts)
}

// TestExistsByOrderNumber tests checking if an order exists by order number
func (suite *OrderRepositoryTestSuite) TestExistsByOrderNumber() {
	ctx := testutil.CreateTestContext()
	order := testutil.CreateTestOrder(suite.T())

	// Create the order first
	err := suite.repo.Create(ctx, order)
	require.NoError(suite.T(), err)

	// Test existing order
	exists, err := suite.repo.ExistsByOrderNumber(ctx, order.OrderNumber)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// Test non-existing order
	exists, err = suite.repo.ExistsByOrderNumber(ctx, "NON-EXISTENT-ORDER")
	require.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

// TestGenerateUniqueOrderNumber tests generating unique order numbers
func (suite *OrderRepositoryTestSuite) TestGenerateUniqueOrderNumber() {
	ctx := testutil.CreateTestContext()

	// Generate multiple order numbers
	orderNumbers := make(map[string]bool)
	for i := 0; i < 10; i++ {
		orderNumber, err := suite.repo.GenerateUniqueOrderNumber(ctx)
		require.NoError(suite.T(), err)
		assert.NotEmpty(suite.T(), orderNumber)
		assert.False(suite.T(), orderNumbers[orderNumber], "Order number should be unique")
		orderNumbers[orderNumber] = true
	}
}

// TestSearchOrders tests searching orders
func (suite *OrderRepositoryTestSuite) TestSearchOrders() {
	ctx := testutil.CreateTestContext()

	// Create test orders with specific data
	order1 := testutil.CreateTestOrder(suite.T())
	customerNotes := "Special delivery instructions"
	order1.CustomerNotes = &customerNotes
	err := suite.repo.Create(ctx, order1)
	require.NoError(suite.T(), err)

	order2 := testutil.CreateTestOrder(suite.T())
	notes := "Urgent priority order"
	order2.Notes = &notes
	err = suite.repo.Create(ctx, order2)
	require.NoError(suite.T(), err)

	// Search for orders with specific terms
	filter := repositories.OrderFilter{
		Limit: 10,
	}

	// Search by customer notes
	orders, err := suite.repo.SearchOrders(ctx, "Special", filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(orders), 1)

	// Search by notes
	orders, err = suite.repo.SearchOrders(ctx, "Urgent", filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(orders), 1)
}

// TestGetOrdersWithItems tests retrieving orders with their items
func (suite *OrderRepositoryTestSuite) TestGetOrdersWithItems() {
	ctx := testutil.CreateTestContext()

	// Create test orders
	orderIDs := make([]uuid.UUID, 2)
	for i := 0; i < 2; i++ {
		order := testutil.CreateTestOrder(suite.T())
		err := suite.repo.Create(ctx, order)
		require.NoError(suite.T(), err)
		orderIDs[i] = order.ID
	}

	// Retrieve orders with items
	orders, err := suite.repo.GetOrdersWithItems(ctx, orderIDs)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, len(orders))

	// Verify the orders are returned (items would be loaded separately)
	for i, order := range orders {
		assert.Equal(suite.T(), orderIDs[i], order.ID)
	}
}

// Run the test suite
func TestOrderRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(OrderRepositoryTestSuite))
}

// BenchmarkCreateOrder benchmarks order creation
func BenchmarkCreateOrder(b *testing.B) {
	// This would require a test database setup
	b.Skip("Skipping benchmark - requires test database setup")
}

// BenchmarkGetOrder benchmarks order retrieval
func BenchmarkGetOrder(b *testing.B) {
	// This would require a test database setup
	b.Skip("Skipping benchmark - requires test database setup")
}

// BenchmarkListOrders benchmarks order listing
func BenchmarkListOrders(b *testing.B) {
	// This would require a test database setup
	b.Skip("Skipping benchmark - requires test database setup")
}
