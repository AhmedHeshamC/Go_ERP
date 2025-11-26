package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/domain/orders/repositories"
	"erpgo/tests/integration/testutil"
)

// OrderItemRepositoryTestSuite contains all tests for the order item repository
type OrderItemRepositoryTestSuite struct {
	suite.Suite
	db          *testutil.TestDatabase
	repo        repositories.OrderItemRepository
	cleanupFunc func()
}

// SetupSuite sets up the test suite
func (suite *OrderItemRepositoryTestSuite) SetupSuite() {
	// This would typically set up a test database
	suite.T().Skip("Skipping database tests - requires test database setup")
}

// TearDownSuite tears down the test suite
func (suite *OrderItemRepositoryTestSuite) TearDownSuite() {
	if suite.cleanupFunc != nil {
		suite.cleanupFunc()
	}
}

// TestCreateOrderItem tests creating a new order item
func (suite *OrderItemRepositoryTestSuite) TestCreateOrderItem() {
	ctx := testutil.CreateTestContext()
	orderID := uuid.New()
	item := testutil.CreateTestOrderItem(suite.T(), orderID)

	err := suite.repo.Create(ctx, item)
	require.NoError(suite.T(), err)

	// Verify the order item was created
	retrieved, err := suite.repo.GetByID(ctx, item.ID)
	require.NoError(suite.T(), err)
	testutil.AssertOrderItemsEqual(suite.T(), item, retrieved)
}

// TestGetByOrderID tests retrieving order items by order ID
func (suite *OrderItemRepositoryTestSuite) TestGetByOrderID() {
	ctx := testutil.CreateTestContext()
	orderID := uuid.New()

	// Create test order items
	items := make([]*entities.OrderItem, 3)
	for i := 0; i < 3; i++ {
		items[i] = testutil.CreateTestOrderItem(suite.T(), orderID)
		items[i].ProductSKU = "TEST-SKU-" + string(rune('A'+i))
		err := suite.repo.Create(ctx, items[i])
		require.NoError(suite.T(), err)
	}

	// Retrieve items for the order
	retrievedItems, err := suite.repo.GetByOrderID(ctx, orderID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, len(retrievedItems))

	// Verify all returned items belong to the order
	for _, item := range retrievedItems {
		assert.Equal(suite.T(), orderID, item.OrderID)
	}
}

// TestGetByProductID tests retrieving order items by product ID
func (suite *OrderItemRepositoryTestSuite) TestGetByProductID() {
	ctx := testutil.CreateTestContext()
	productID := uuid.New()

	// Create test order items for the product
	for i := 0; i < 3; i++ {
		item := testutil.CreateTestOrderItem(suite.T(), uuid.New())
		item.ProductID = productID
		item.ProductSKU = "TEST-SKU-" + string(rune('A'+i))
		err := suite.repo.Create(ctx, item)
		require.NoError(suite.T(), err)
	}

	// Retrieve items for the product
	retrievedItems, err := suite.repo.GetByProductID(ctx, productID)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(retrievedItems), 3)

	// Verify all returned items belong to the product
	for _, item := range retrievedItems {
		assert.Equal(suite.T(), productID, item.ProductID)
	}
}

// TestUpdateItemStatus tests updating order item status
func (suite *OrderItemRepositoryTestSuite) TestUpdateItemStatus() {
	ctx := testutil.CreateTestContext()
	orderID := uuid.New()
	item := testutil.CreateTestOrderItem(suite.T(), orderID)

	// Create the order item first
	err := suite.repo.Create(ctx, item)
	require.NoError(suite.T(), err)

	// Update the item status
	err = suite.repo.UpdateItemStatus(ctx, item.ID, "SHIPPED")
	require.NoError(suite.T(), err)

	// Verify the status update
	retrieved, err := suite.repo.GetByID(ctx, item.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "SHIPPED", retrieved.Status)
}

// TestUpdateShippedQuantity tests updating shipped quantity
func (suite *OrderItemRepositoryTestSuite) TestUpdateShippedQuantity() {
	ctx := testutil.CreateTestContext()
	orderID := uuid.New()
	item := testutil.CreateTestOrderItem(suite.T(), orderID)
	item.Quantity = 10

	// Create the order item first
	err := suite.repo.Create(ctx, item)
	require.NoError(suite.T(), err)

	// Update the shipped quantity
	err = suite.repo.UpdateShippedQuantity(ctx, item.ID, 5)
	require.NoError(suite.T(), err)

	// Verify the shipped quantity update
	retrieved, err := suite.repo.GetByID(ctx, item.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 5, retrieved.QuantityShipped)
}

// TestUpdateReturnedQuantity tests updating returned quantity
func (suite *OrderItemRepositoryTestSuite) TestUpdateReturnedQuantity() {
	ctx := testutil.CreateTestContext()
	orderID := uuid.New()
	item := testutil.CreateTestOrderItem(suite.T(), orderID)
	item.Quantity = 10

	// Create the order item first
	err := suite.repo.Create(ctx, item)
	require.NoError(suite.T(), err)

	// Update the returned quantity
	err = suite.repo.UpdateReturnedQuantity(ctx, item.ID, 2)
	require.NoError(suite.T(), err)

	// Verify the returned quantity update
	retrieved, err := suite.repo.GetByID(ctx, item.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, retrieved.QuantityReturned)
}

// TestBulkCreate tests bulk creating order items
func (suite *OrderItemRepositoryTestSuite) TestBulkCreate() {
	ctx := testutil.CreateTestContext()
	orderID := uuid.New()

	// Create test order items
	items := make([]*entities.OrderItem, 3)
	for i := 0; i < 3; i++ {
		items[i] = testutil.CreateTestOrderItem(suite.T(), orderID)
		items[i].ProductSKU = "BULK-SKU-" + string(rune('A'+i))
	}

	// Bulk create order items
	err := suite.repo.BulkCreate(ctx, items)
	require.NoError(suite.T(), err)

	// Verify the order items were created
	for _, item := range items {
		retrieved, err := suite.repo.GetByID(ctx, item.ID)
		require.NoError(suite.T(), err)
		testutil.AssertOrderItemsEqual(suite.T(), item, retrieved)
	}
}

// TestBulkUpdate tests bulk updating order items
func (suite *OrderItemRepositoryTestSuite) TestBulkUpdate() {
	ctx := testutil.CreateTestContext()
	orderID := uuid.New()

	// Create test order items
	items := make([]*entities.OrderItem, 3)
	for i := 0; i < 3; i++ {
		items[i] = testutil.CreateTestOrderItem(suite.T(), orderID)
		items[i].ProductSKU = "UPDATE-SKU-" + string(rune('A'+i))
		err := suite.repo.Create(ctx, items[i])
		require.NoError(suite.T(), err)
	}

	// Update the items
	for _, item := range items {
		item.Status = "PROCESSED"
		item.QuantityShipped = item.Quantity
	}

	// Bulk update order items
	err := suite.repo.BulkUpdate(ctx, items)
	require.NoError(suite.T(), err)

	// Verify the updates
	for _, item := range items {
		retrieved, err := suite.repo.GetByID(ctx, item.ID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "PROCESSED", retrieved.Status)
		assert.Equal(suite.T(), item.Quantity, retrieved.QuantityShipped)
	}
}

// TestDeleteByOrderID tests deleting order items by order ID
func (suite *OrderItemRepositoryTestSuite) TestDeleteByOrderID() {
	ctx := testutil.CreateTestContext()
	orderID := uuid.New()

	// Create test order items
	for i := 0; i < 3; i++ {
		item := testutil.CreateTestOrderItem(suite.T(), orderID)
		item.ProductSKU = "DELETE-SKU-" + string(rune('A'+i))
		err := suite.repo.Create(ctx, item)
		require.NoError(suite.T(), err)
	}

	// Delete all items for the order
	err := suite.repo.DeleteByOrderID(ctx, orderID)
	require.NoError(suite.T(), err)

	// Verify the items were deleted
	retrievedItems, err := suite.repo.GetByOrderID(ctx, orderID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, len(retrievedItems))
}

// TestGetProductOrderHistory tests retrieving product order history
func (suite *OrderItemRepositoryTestSuite) TestGetProductOrderHistory() {
	ctx := testutil.CreateTestContext()
	productID := uuid.New()

	// Create test order items for the product
	for i := 0; i < 5; i++ {
		item := testutil.CreateTestOrderItem(suite.T(), uuid.New())
		item.ProductID = productID
		item.ProductSKU = "HISTORY-SKU-" + string(rune('A'+i))
		item.TotalPrice = decimal.NewFromFloat(float64(i+1) * 10.0)
		err := suite.repo.Create(ctx, item)
		require.NoError(suite.T(), err)
	}

	// Retrieve product order history
	history, err := suite.repo.GetProductOrderHistory(ctx, productID, 10)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(history), 5)

	// Verify all returned items belong to the product
	for _, item := range history {
		assert.Equal(suite.T(), productID, item.ProductID)
	}
}

// TestGetItemsByStatus tests retrieving order items by status
func (suite *OrderItemRepositoryTestSuite) TestGetItemsByStatus() {
	ctx := testutil.CreateTestContext()

	// Create test order items with different statuses
	statuses := []string{"ORDERED", "SHIPPED", "DELIVERED"}
	for _, status := range statuses {
		for i := 0; i < 2; i++ {
			item := testutil.CreateTestOrderItem(suite.T(), uuid.New())
			item.Status = status
			item.ProductSKU = "STATUS-SKU-" + status + "-" + string(rune('A'+i))
			err := suite.repo.Create(ctx, item)
			require.NoError(suite.T(), err)
		}
	}

	// Retrieve items with SHIPPED status
	shippedItems, err := suite.repo.GetItemsByStatus(ctx, "SHIPPED")
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(shippedItems), 2)

	// Verify all returned items have the correct status
	for _, item := range shippedItems {
		assert.Equal(suite.T(), "SHIPPED", item.Status)
	}
}

// TestGetByOrderAndProduct tests retrieving a specific order item
func (suite *OrderItemRepositoryTestSuite) TestGetByOrderAndProduct() {
	ctx := testutil.CreateTestContext()
	orderID := uuid.New()
	productID := uuid.New()

	// Create test order item
	item := testutil.CreateTestOrderItem(suite.T(), orderID)
	item.ProductID = productID
	err := suite.repo.Create(ctx, item)
	require.NoError(suite.T(), err)

	// Retrieve the specific order item
	retrieved, err := suite.repo.GetByOrderAndProduct(ctx, orderID, productID)
	require.NoError(suite.T(), err)
	testutil.AssertOrderItemsEqual(suite.T(), item, retrieved)
}

// TestLowStockItems tests retrieving products with low stock
func (suite *OrderItemRepositoryTestSuite) TestLowStockItems() {
	ctx := testutil.CreateTestContext()

	// This test would require inventory data to be set up
	// For now, we'll test the method exists and doesn't error
	items, err := suite.repo.GetLowStockItems(ctx, 10)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), items)
}

// Run the test suite
func TestOrderItemRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(OrderItemRepositoryTestSuite))
}

// BenchmarkCreateOrderItem benchmarks order item creation
func BenchmarkCreateOrderItem(b *testing.B) {
	// This would require a test database setup
	b.Skip("Skipping benchmark - requires test database setup")
}

// BenchmarkGetOrderItems benchmarks order item retrieval
func BenchmarkGetOrderItems(b *testing.B) {
	// This would require a test database setup
	b.Skip("Skipping benchmark - requires test database setup")
}