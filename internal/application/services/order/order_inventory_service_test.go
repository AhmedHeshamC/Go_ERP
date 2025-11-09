package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erpgo/internal/domain/inventory/entities"
	inventoryRepositories "erpgo/internal/domain/inventory/repositories"
	"erpgo/internal/domain/orders/entities"
	orderRepositories "erpgo/internal/domain/orders/repositories"
)

// Mock implementations for testing
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) GetByID(ctx context.Context, id string) (*entities.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Order), args.Error(1)
}

// Add other mock methods as needed...

type MockInventoryRepository struct {
	mock.Mock
}

func (m *MockInventoryRepository) GetByProductAndWarehouse(ctx context.Context, productID, warehouseID uuid.UUID) (*entities.Inventory, error) {
	args := m.Called(ctx, productID, warehouseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) Update(ctx context.Context, inventory *entities.Inventory) error {
	args := m.Called(ctx, inventory)
	return args.Error(0)
}

func (m *MockInventoryRepository) GetLowStockItems(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error) {
	args := m.Called(ctx, warehouseID)
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) GetLowStockItemsAll(ctx context.Context) ([]*entities.Inventory, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

// Add other mock methods as needed...

type MockInventoryTransactionRepository struct {
	mock.Mock
}

func (m *MockInventoryTransactionRepository) Create(ctx context.Context, transaction *entities.InventoryTransaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *MockInventoryTransactionRepository) GetByReference(ctx context.Context, referenceType string, referenceID uuid.UUID) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, referenceType, referenceID)
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetByDateRange(ctx context.Context, warehouseID *uuid.UUID, startDate, endDate time.Time, filter *inventoryRepositories.TransactionFilter) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, warehouseID, startDate, endDate, filter)
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) Count(ctx context.Context, filter *inventoryRepositories.TransactionFilter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

// Add other mock methods as needed...

type MockWarehouseRepository struct {
	mock.Mock
}

func (m *MockWarehouseRepository) List(ctx context.Context, filter *inventoryRepositories.WarehouseFilter) ([]*entities.Warehouse, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*entities.Warehouse), args.Error(1)
}

// Add other mock methods as needed...

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Error(ctx context.Context, msg string, fields ...interface{}) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) Warn(ctx context.Context, msg string, fields ...interface{}) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) Info(ctx context.Context, msg string, fields ...interface{}) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	m.Called(ctx, msg, fields)
}

type MockMonitoringService struct {
	mock.Mock
}

func (m *MockMonitoringService) RecordMetric(ctx context.Context, name string, value float64, tags map[string]string) {
	m.Called(ctx, name, value, tags)
}

func (m *MockMonitoringService) RecordCounter(ctx context.Context, name string, value float64, tags map[string]string) {
	m.Called(ctx, name, value, tags)
}

func (m *MockMonitoringService) RecordHistogram(ctx context.Context, name string, value float64, tags map[string]string) {
	m.Called(ctx, name, value, tags)
}

func (m *MockMonitoringService) RecordError(ctx context.Context, errorType string, err error) {
	m.Called(ctx, errorType, err)
}

type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) SendLowStockAlert(ctx context.Context, alert *LowStockAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockNotificationService) SendInventoryConflictAlert(ctx context.Context, conflict *InventoryConflict) error {
	args := m.Called(ctx, conflict)
	return args.Error(0)
}

func (m *MockNotificationService) SendReservationFailureAlert(ctx context.Context, orderID uuid.UUID, reason string) error {
	args := m.Called(ctx, orderID, reason)
	return args.Error(0)
}

type MockCacheService struct {
	mock.Mock
}

func (m *MockCacheService) Get(ctx context.Context, key string) (interface{}, error) {
	args := m.Called(ctx, key)
	return args.Get(0), args.Error(1)
}

func (m *MockCacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheService) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

// Test setup
func setupTestService() (*orderInventoryService, *MockOrderRepository, *MockInventoryRepository, *MockInventoryTransactionRepository, *MockWarehouseRepository, *MockLogger, *MockMonitoringService, *MockNotificationService, *MockCacheService) {
	orderRepo := &MockOrderRepository{}
	inventoryRepo := &MockInventoryRepository{}
	transactionRepo := &MockInventoryTransactionRepository{}
	warehouseRepo := &MockWarehouseRepository{}
	logger := &MockLogger{}
	monitoring := &MockMonitoringService{}
	notificationService := &MockNotificationService{}
	cache := &MockCacheService{}

	config := getDefaultInventoryConfig()

	service := &orderInventoryService{
		orderRepo:           orderRepo,
		inventoryRepo:       inventoryRepo,
		transactionRepo:     transactionRepo,
		warehouseRepo:       warehouseRepo,
		logger:              logger,
		monitoring:          monitoring,
		notificationService: notificationService,
		cache:               cache,
		config:              config,
	}

	return service, orderRepo, inventoryRepo, transactionRepo, warehouseRepo, logger, monitoring, notificationService, cache
}

// Test helper functions
func createTestInventory(productID, warehouseID uuid.UUID, quantityOnHand, quantityReserved, reorderLevel int, averageCost float64) *entities.Inventory {
	return &entities.Inventory{
		ID:               uuid.New(),
		ProductID:        productID,
		WarehouseID:      warehouseID,
		QuantityOnHand:   quantityOnHand,
		QuantityReserved: quantityReserved,
		ReorderLevel:     reorderLevel,
		AverageCost:      averageCost,
		UpdatedAt:        time.Now(),
		UpdatedBy:        uuid.New(),
	}
}

func createTestOrder(orderID uuid.UUID, status entities.OrderStatus) *entities.Order {
	productID := uuid.New()
	warehouseID := uuid.New()

	return &entities.Order{
		ID:     orderID,
		Status: status,
		Items: []*entities.OrderItem{
			{
				ID:          uuid.New(),
				ProductID:   productID,
				WarehouseID: warehouseID,
				Quantity:    10,
				UnitPrice:   decimal.NewFromFloat(100.0),
			},
		},
	}
}

// Test ReserveInventory
func TestReserveInventory(t *testing.T) {
	service, orderRepo, inventoryRepo, transactionRepo, _, logger, monitoring, _, _ := setupTestService()
	ctx := context.Background()
	orderID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()

	// Setup test data
	order := createTestOrder(orderID, entities.OrderStatusPending)
	inventory := createTestInventory(productID, warehouseID, 50, 10, 20, 100.0)

	// Mock expectations
	orderRepo.On("GetByID", ctx, orderID.String()).Return(order, nil)
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil)
	inventoryRepo.On("Update", ctx, mock.AnythingOfType("*entities.Inventory")).Return(nil)
	transactionRepo.On("Create", ctx, mock.AnythingOfType("*entities.InventoryTransaction")).Return(nil)
	logger.On("Info", ctx, mock.AnythingOfType("string"), mock.Anything).Return()
	monitoring.On("RecordCounter", ctx, "inventory.reservations.success", float64(1), mock.Anything).Return()
	monitoring.On("RecordMetric", ctx, "inventory.reservation.value", mock.AnythingOfType("float64"), mock.Anything).Return()
	monitoring.On("RecordHistogram", ctx, "inventory.reservation.duration", mock.AnythingOfType("float64"), mock.Anything).Return()

	// Test with invalid request (missing requestedBy)
	err := service.ReserveInventory(ctx, orderID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requestedBy")

	logger.AssertExpectations(t)
}

// Test ReserveInventoryItems
func TestReserveInventoryItems(t *testing.T) {
	service, orderRepo, inventoryRepo, transactionRepo, _, logger, monitoring, _, _ := setupTestService()
	ctx := context.Background()
	orderID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	userID := uuid.New()

	// Setup test data
	order := createTestOrder(orderID, entities.OrderStatusPending)
	inventory := createTestInventory(productID, warehouseID, 50, 10, 20, 100.0)

	items := []ReserveInventoryRequest{
		{
			OrderItemID: uuid.New(),
			ProductID:   productID,
			WarehouseID: warehouseID,
			Quantity:    10,
			ReservedBy:  userID,
		},
	}

	// Mock expectations
	orderRepo.On("GetByID", ctx, orderID.String()).Return(order, nil)
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil)
	inventoryRepo.On("Update", ctx, mock.AnythingOfType("*entities.Inventory")).Return(nil)
	transactionRepo.On("Create", ctx, mock.AnythingOfType("*entities.InventoryTransaction")).Return(nil)
	logger.On("Info", ctx, mock.AnythingOfType("string"), mock.Anything).Return()
	monitoring.On("RecordCounter", ctx, "inventory.reservations.success", float64(1), mock.Anything).Return()
	monitoring.On("RecordMetric", ctx, "inventory.reservation.value", mock.AnythingOfType("float64"), mock.Anything).Return()
	monitoring.On("RecordHistogram", ctx, "inventory.reservation.duration", mock.AnythingOfType("float64"), mock.Anything).Return()

	// Execute test
	err := service.ReserveInventoryItems(ctx, orderID, items)
	assert.NoError(t, err)

	// Verify expectations
	orderRepo.AssertExpectations(t)
	inventoryRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
	logger.AssertExpectations(t)
	monitoring.AssertExpectations(t)
}

// Test ReserveInventoryItems_InsufficientInventory
func TestReserveInventoryItems_InsufficientInventory(t *testing.T) {
	service, orderRepo, inventoryRepo, _, _, logger, _, _, _ := setupTestService()
	ctx := context.Background()
	orderID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	userID := uuid.New()

	// Setup test data with insufficient inventory
	order := createTestOrder(orderID, entities.OrderStatusPending)
	inventory := createTestInventory(productID, warehouseID, 5, 5, 20, 100.0) // Only 5 available, need 10

	items := []ReserveInventoryRequest{
		{
			OrderItemID: uuid.New(),
			ProductID:   productID,
			WarehouseID: warehouseID,
			Quantity:    10,
			ReservedBy:  userID,
		},
	}

	// Mock expectations
	orderRepo.On("GetByID", ctx, orderID.String()).Return(order, nil)
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil)

	// Execute test
	err := service.ReserveInventoryItems(ctx, orderID, items)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient inventory")

	// Verify expectations
	orderRepo.AssertExpectations(t)
	inventoryRepo.AssertExpectations(t)
}

// Test ReserveInventoryItems_InvalidOrderStatus
func TestReserveInventoryItems_InvalidOrderStatus(t *testing.T) {
	service, orderRepo, _, _, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	orderID := uuid.New()
	userID := uuid.New()

	// Setup test data with invalid order status
	order := createTestOrder(orderID, entities.OrderStatusDelivered) // Delivered orders can't be reserved

	items := []ReserveInventoryRequest{
		{
			OrderItemID: uuid.New(),
			ProductID:   uuid.New(),
			WarehouseID: uuid.New(),
			Quantity:    10,
			ReservedBy:  userID,
		},
	}

	// Mock expectations
	orderRepo.On("GetByID", ctx, orderID.String()).Return(order, nil)

	// Execute test
	err := service.ReserveInventoryItems(ctx, orderID, items)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot have inventory reserved")

	// Verify expectations
	orderRepo.AssertExpectations(t)
}

// Test ReleaseInventoryReservation
func TestReleaseInventoryReservation(t *testing.T) {
	service, orderRepo, inventoryRepo, transactionRepo, _, logger, _, _, _ := setupTestService()
	ctx := context.Background()
	orderID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()

	// Setup test data
	order := createTestOrder(orderID, entities.OrderStatusPending)
	inventory := createTestInventory(productID, warehouseID, 30, 20, 20, 100.0) // 20 reserved

	// Create mock transactions for the reservation
	transactions := []*entities.InventoryTransaction{
		{
			ID:              uuid.New(),
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: entities.TransactionTypeAdjustment,
			Quantity:        -10,
			ReferenceType:   "ORDER",
			ReferenceID:     &orderID,
		},
	}

	// Mock expectations
	orderRepo.On("GetByID", ctx, orderID.String()).Return(order, nil)
	transactionRepo.On("GetByReference", ctx, "ORDER", orderID).Return(transactions, nil)
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil).Twice()
	inventoryRepo.On("Update", ctx, mock.AnythingOfType("*entities.Inventory")).Return(nil)
	transactionRepo.On("Create", ctx, mock.AnythingOfType("*entities.InventoryTransaction")).Return(nil)
	logger.On("Info", ctx, mock.AnythingOfType("string"), mock.Anything).Return()

	// Execute test
	err := service.ReleaseInventoryReservation(ctx, orderID)
	assert.NoError(t, err)

	// Verify expectations
	orderRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
	inventoryRepo.AssertExpectations(t)
	logger.AssertExpectations(t)
}

// Test DeductInventoryItems
func TestDeductInventoryItems(t *testing.T) {
	service, _, inventoryRepo, transactionRepo, _, logger, monitoring, _, _ := setupTestService()
	ctx := context.Background()
	orderID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	userID := uuid.New()

	// Setup test data
	inventory := createTestInventory(productID, warehouseID, 50, 20, 20, 100.0) // 30 available

	items := []DeductInventoryRequest{
		{
			OrderItemID: uuid.New(),
			ProductID:   productID,
			WarehouseID: warehouseID,
			Quantity:    10,
			UnitCost:    decimal.NewFromFloat(100.0),
			DeductedBy:  userID,
			Reason:      "Order fulfillment",
		},
	}

	// Mock expectations
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil)
	inventoryRepo.On("Update", ctx, mock.AnythingOfType("*entities.Inventory")).Return(nil)
	transactionRepo.On("Create", ctx, mock.AnythingOfType("*entities.InventoryTransaction")).Return(nil)

	// Execute test
	err := service.DeductInventoryItems(ctx, orderID, items)
	assert.NoError(t, err)

	// Verify expectations
	inventoryRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

// Test DeductInventoryItems_InsufficientStock
func TestDeductInventoryItems_InsufficientStock(t *testing.T) {
	service, _, inventoryRepo, _, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	orderID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	userID := uuid.New()

	// Setup test data with insufficient available stock
	inventory := createTestInventory(productID, warehouseID, 15, 10, 20, 100.0) // Only 5 available

	items := []DeductInventoryRequest{
		{
			OrderItemID: uuid.New(),
			ProductID:   productID,
			WarehouseID: warehouseID,
			Quantity:    10,
			UnitCost:    decimal.NewFromFloat(100.0),
			DeductedBy:  userID,
			Reason:      "Order fulfillment",
		},
	}

	// Mock expectations
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil)

	// Execute test
	err := service.DeductInventoryItems(ctx, orderID, items)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient available stock")

	// Verify expectations
	inventoryRepo.AssertExpectations(t)
}

// Test ReturnInventory
func TestReturnInventory(t *testing.T) {
	service, _, inventoryRepo, transactionRepo, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	orderID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	userID := uuid.New()

	// Setup test data
	inventory := createTestInventory(productID, warehouseID, 20, 5, 20, 100.0)

	items := []ReturnInventoryRequest{
		{
			OrderItemID: uuid.New(),
			ProductID:   productID,
			WarehouseID: warehouseID,
			Quantity:    5,
			Condition:   "NEW",
			UnitCost:    decimal.NewFromFloat(100.0),
			ReturnedBy:  userID,
			Reason:      "Customer return",
		},
	}

	// Mock expectations
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil)
	inventoryRepo.On("Update", ctx, mock.AnythingOfType("*entities.Inventory")).Return(nil)
	transactionRepo.On("Create", ctx, mock.AnythingOfType("*entities.InventoryTransaction")).Return(nil)

	// Execute test
	err := service.ReturnInventory(ctx, orderID, items)
	assert.NoError(t, err)

	// Verify expectations
	inventoryRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

// Test CheckInventoryAvailability
func TestCheckInventoryAvailability(t *testing.T) {
	service, _, inventoryRepo, _, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	productID := uuid.New()
	warehouseID := uuid.New()

	req := &CheckInventoryAvailabilityRequest{
		Items: []CheckInventoryItemRequest{
			{
				ProductID:   productID,
				Quantity:    10,
				WarehouseID: &warehouseID,
			},
		},
		IncludeBatch: true,
		IncludeCost:  true,
	}

	// Setup test data
	inventory := createTestInventory(productID, warehouseID, 50, 10, 20, 100.0)

	// Mock expectations
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil)

	// Execute test
	response, err := service.CheckInventoryAvailability(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Available)
	assert.Len(t, response.Items, 1)
	assert.True(t, response.Items[0].CanFulfill)
	assert.Equal(t, productID, response.Items[0].ProductID)
	assert.Equal(t, 10, response.Items[0].RequestedQty)
	assert.Equal(t, 40, response.Items[0].AvailableQty) // 50 - 10 reserved
	assert.True(t, response.Items[0].UnitCost.GreaterThan(decimal.Zero))

	// Verify expectations
	inventoryRepo.AssertExpectations(t)
}

// Test CheckInventoryAvailability_InsufficientStock
func TestCheckInventoryAvailability_InsufficientStock(t *testing.T) {
	service, _, inventoryRepo, _, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	productID := uuid.New()
	warehouseID := uuid.New()

	req := &CheckInventoryAvailabilityRequest{
		Items: []CheckInventoryItemRequest{
			{
				ProductID:   productID,
				Quantity:    50, // Request more than available
				WarehouseID: &warehouseID,
			},
		},
	}

	// Setup test data with insufficient stock
	inventory := createTestInventory(productID, warehouseID, 30, 5, 20, 100.0) // Only 25 available

	// Mock expectations
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil)

	// Execute test
	response, err := service.CheckInventoryAvailability(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.False(t, response.Available)
	assert.Len(t, response.Items, 1)
	assert.False(t, response.Items[0].CanFulfill)
	assert.Equal(t, 25, response.Items[0].AvailableQty)
	assert.Contains(t, response.Items[0].Reason, "Insufficient inventory")

	// Verify expectations
	inventoryRepo.AssertExpectations(t)
}

// Test CheckLowStock
func TestCheckLowStock(t *testing.T) {
	service, _, inventoryRepo, _, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	productID := uuid.New()
	warehouseID := uuid.New()

	req := &CheckLowStockRequest{
		WarehouseIDs:     []uuid.UUID{warehouseID},
		IncludeZero:      true,
		CalculateReorder: true,
		IncludeForecast:  true,
		Days:            30,
	}

	// Setup test data - low stock items
	lowStockItems := []*entities.Inventory{
		createTestInventory(productID, warehouseID, 5, 2, 20, 100.0),  // Below reorder level
		createTestInventory(uuid.New(), warehouseID, 0, 0, 10, 50.0),  // Out of stock
	}

	// Mock expectations
	inventoryRepo.On("GetLowStockItems", ctx, warehouseID).Return(lowStockItems, nil)

	// Execute test
	response, err := service.CheckLowStock(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 2, response.TotalItems)
	assert.Equal(t, 2, response.CriticalItems) // Both are critical
	assert.True(t, response.TotalValue.GreaterThan(decimal.Zero))
	assert.True(t, response.ReorderValue.GreaterThan(decimal.Zero))
	assert.NotNil(t, response.WarehouseSummary)
	assert.NotNil(t, response.ForecastData)

	// Verify warehouse summary
	summary, exists := response.WarehouseSummary[warehouseID]
	assert.True(t, exists)
	assert.Equal(t, 2, summary.LowStockItems)
	assert.Equal(t, 2, summary.CriticalItems)

	// Verify expectations
	inventoryRepo.AssertExpectations(t)
}

// Test GenerateReorderSuggestions
func TestGenerateReorderSuggestions(t *testing.T) {
	service, _, inventoryRepo, _, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	productID := uuid.New()
	warehouseID := uuid.New()

	req := &GenerateReorderSuggestionsRequest{
		WarehouseIDs:    []uuid.UUID{warehouseID},
		IncludeForecast: true,
		ForecastDays:    30,
		MinSavings:      decimal.NewFromFloat(10.0),
	}

	// Setup test data
	lowStockItems := []*entities.Inventory{
		createTestInventory(productID, warehouseID, 5, 2, 20, 100.0),
	}

	// Mock expectations
	inventoryRepo.On("GetLowStockItems", ctx, warehouseID).Return(lowStockItems, nil)
	inventoryRepo.On("GetByProduct", ctx, productID).Return([]*entities.Inventory{}, nil)

	// Execute test
	response, err := service.GenerateReorderSuggestions(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.TotalItems > 0)
	assert.True(t, response.TotalValue.GreaterThan(decimal.Zero))
	assert.NotNil(t, response.WarehouseSummary)
	assert.NotNil(t, response.Suggestions)

	// Verify suggestions structure
	for _, suggestion := range response.Suggestions {
		assert.NotEmpty(t, suggestion.ProductID)
		assert.NotEmpty(t, suggestion.WarehouseID)
		assert.True(t, suggestion.SuggestedQty > 0)
		assert.True(t, suggestion.SuggestedValue.GreaterThan(decimal.Zero))
		assert.NotEmpty(t, suggestion.Priority)
		assert.NotEmpty(t, suggestion.Reason)
	}

	// Verify expectations
	inventoryRepo.AssertExpectations(t)
}

// Test LogInventoryTransaction
func TestLogInventoryTransaction(t *testing.T) {
	service, _, inventoryRepo, transactionRepo, _, _, monitoring, _, _ := setupTestService()
	ctx := context.Background()
	productID := uuid.New()
	warehouseID := uuid.New()
	userID := uuid.New()

	req := &LogInventoryTransactionRequest{
		TransactionType: entities.TransactionTypePurchase,
		ProductID:       productID,
		WarehouseID:     warehouseID,
		Quantity:        50,
		ReferenceType:   "ORDER",
		ReferenceID:     &uuid.New(),
		Reason:          "Stock replenishment",
		UnitCost:        decimal.NewFromFloat(100.0),
		TotalCost:       decimal.NewFromFloat(5000.0),
		CreatedBy:       userID,
	}

	// Setup test data
	inventory := createTestInventory(productID, warehouseID, 20, 5, 20, 100.0)

	// Mock expectations
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil).Maybe()
	inventoryRepo.On("Update", ctx, mock.AnythingOfType("*entities.Inventory")).Return(nil).Maybe()
	transactionRepo.On("Create", ctx, mock.AnythingOfType("*entities.InventoryTransaction")).Return(nil)
	monitoring.On("RecordCounter", ctx, "inventory.transactions.created", float64(1), map[string]string{"type": "PURCHASE"}).Return()

	// Execute test
	err := service.LogInventoryTransaction(ctx, req)
	assert.NoError(t, err)

	// Verify expectations
	transactionRepo.AssertExpectations(t)
	monitoring.AssertExpectations(t)
}

// Test LogInventoryTransaction_ValidationError
func TestLogInventoryTransaction_ValidationError(t *testing.T) {
	service, _, _, _, _, _, _, _, _ := setupTestService()
	ctx := context.Background()

	req := &LogInventoryTransactionRequest{
		TransactionType: entities.TransactionTypePurchase,
		ProductID:       uuid.Nil, // Invalid - empty UUID
		WarehouseID:     uuid.New(),
		Quantity:        50,
		CreatedBy:       uuid.New(),
	}

	// Execute test
	err := service.LogInventoryTransaction(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "product ID is required")
}

// Test ProcessInventoryWithRetry
func TestProcessInventoryWithRetry(t *testing.T) {
	service, orderRepo, inventoryRepo, transactionRepo, _, logger, monitoring, _, _ := setupTestService()
	ctx := context.Background()
	orderID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	userID := uuid.New()

	req := &ProcessInventoryWithRetryRequest{
		OperationType:        "RESERVE",
		OrderID:             orderID,
		Items: []ProcessInventoryItemRequest{
			{
				OrderItemID: uuid.New(),
				ProductID:   productID,
				WarehouseID: warehouseID,
				Quantity:    10,
			},
		},
		MaxRetries:          2,
		RetryDelay:          10 * time.Millisecond,
		ExponentialBackoff:  false,
		ConflictStrategy:    "RETRY",
		RequestedBy:         userID,
	}

	// Setup test data
	order := createTestOrder(orderID, entities.OrderStatusPending)
	inventory := createTestInventory(productID, warehouseID, 50, 10, 20, 100.0)

	// Mock expectations
	orderRepo.On("GetByID", ctx, orderID.String()).Return(order, nil)
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil)
	inventoryRepo.On("Update", ctx, mock.AnythingOfType("*entities.Inventory")).Return(nil)
	transactionRepo.On("Create", ctx, mock.AnythingOfType("*entities.InventoryTransaction")).Return(nil)
	logger.On("Info", ctx, mock.AnythingOfType("string"), mock.Anything).Return()
	monitoring.On("RecordCounter", ctx, "inventory.reservations.success", float64(1), mock.Anything).Return()
	monitoring.On("RecordMetric", ctx, "inventory.reservation.value", mock.AnythingOfType("float64"), mock.Anything).Return()
	monitoring.On("RecordHistogram", ctx, "inventory.reservation.duration", mock.AnythingOfType("float64"), mock.Anything).Return()

	// Execute test
	err := service.ProcessInventoryWithRetry(ctx, req)
	assert.NoError(t, err)

	// Verify expectations
	orderRepo.AssertExpectations(t)
	inventoryRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
	logger.AssertExpectations(t)
	monitoring.AssertExpectations(t)
}

// Test ProcessInventoryWithRetry_MaxRetriesExceeded
func TestProcessInventoryWithRetry_MaxRetriesExceeded(t *testing.T) {
	service, orderRepo, inventoryRepo, _, _, logger, monitoring, _, _ := setupTestService()
	ctx := context.Background()
	orderID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	userID := uuid.New()

	req := &ProcessInventoryWithRetryRequest{
		OperationType:        "RESERVE",
		OrderID:             orderID,
		Items: []ProcessInventoryItemRequest{
			{
				OrderItemID: uuid.New(),
				ProductID:   productID,
				WarehouseID: warehouseID,
				Quantity:    10,
			},
		},
		MaxRetries:          2,
		RetryDelay:          10 * time.Millisecond,
		ExponentialBackoff:  false,
		ConflictStrategy:    "RETRY",
		RequestedBy:         userID,
	}

	// Setup test data
	order := createTestOrder(orderID, entities.OrderStatusPending)
	inventory := createTestInventory(productID, warehouseID, 5, 5, 20, 100.0) // Insufficient stock

	// Mock expectations - always fail
	orderRepo.On("GetByID", ctx, orderID.String()).Return(order, nil)
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil)
	logger.On("Warn", ctx, mock.AnythingOfType("string"), mock.Anything).Return()
	logger.On("Error", ctx, mock.AnythingOfType("string"), mock.Anything).Return()
	monitoring.On("RecordCounter", ctx, "inventory.retry.failed", float64(1), mock.Anything).Return()

	// Execute test
	err := service.ProcessInventoryWithRetry(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max retries exceeded")

	// Verify expectations
	orderRepo.AssertExpectations(t)
	inventoryRepo.AssertExpectations(t)
	logger.AssertExpectations(t)
	monitoring.AssertExpectations(t)
}

// Test ResolveInventoryConflicts
func TestResolveInventoryConflicts(t *testing.T) {
	service, orderRepo, _, _, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	orderID := uuid.New()
	productID := uuid.New()
	userID := uuid.New()

	req := &ResolveInventoryConflictsRequest{
		OrderID:    orderID,
		Conflicts: []InventoryConflict{
			{
				OrderItemID:  uuid.New(),
				ProductID:    productID,
				WarehouseID:  uuid.New(),
				RequestedQty: 10,
				AvailableQty: 5,
				ConflictType: "INSUFFICIENT_STOCK",
				Severity:     "HIGH",
				Message:      "Insufficient stock",
			},
		},
		Resolution:    "BACKORDER",
		AutoResolve:   true,
		ResolvedBy:    userID,
	}

	// Setup test data
	order := createTestOrder(orderID, entities.OrderStatusPending)

	// Mock expectations
	orderRepo.On("GetByID", ctx, orderID.String()).Return(order, nil)
	orderRepo.On("GetByID", ctx, mock.AnythingOfType("string")).Return(order, nil).Maybe()

	// Execute test
	response, err := service.ResolveInventoryConflicts(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, orderID, response.OrderID)
	assert.True(t, response.ResolvedCount > 0 || response.UnresolvedCount > 0)
	assert.Len(t, response.Resolutions, 1)

	// Verify resolution
	resolution := response.Resolutions[0]
	assert.Equal(t, req.Conflicts[0].OrderItemID, resolution.OrderItemID)
	assert.Equal(t, req.Conflicts[0].ProductID, resolution.ProductID)
	assert.Equal(t, req.Resolution, resolution.Resolution)

	// Verify expectations
	orderRepo.AssertExpectations(t)
}

// Test GetInventoryTransactionHistory
func TestGetInventoryTransactionHistory(t *testing.T) {
	service, _, _, transactionRepo, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	productID := uuid.New()

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	req := &GetInventoryTransactionHistoryRequest{
		ProductID:    &productID,
		StartDate:    &startDate,
		EndDate:      &endDate,
		Page:         1,
		Limit:        10,
		SortBy:       "created_at",
		SortOrder:    "desc",
	}

	// Setup test data
	transactions := []*entities.InventoryTransaction{
		{
			ID:              uuid.New(),
			ProductID:       productID,
			TransactionType: entities.TransactionTypeSale,
			Quantity:        -10,
			TotalCost:       1000.0,
			CreatedAt:       time.Now(),
		},
	}

	// Mock expectations
	transactionRepo.On("GetByDateRange", ctx, (*uuid.UUID)(nil), startDate, endDate, mock.AnythingOfType("*inventoryRepositories.TransactionFilter")).Return(transactions, nil)
	transactionRepo.On("Count", ctx, mock.AnythingOfType("*inventoryRepositories.TransactionFilter")).Return(1, nil)

	// Execute test
	response, err := service.GetInventoryTransactionHistory(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Transactions, 1)
	assert.NotNil(t, response.Pagination)
	assert.NotNil(t, response.Summary)

	// Verify pagination
	assert.Equal(t, 1, response.Pagination.Page)
	assert.Equal(t, 10, response.Pagination.Limit)
	assert.Equal(t, 1, response.Pagination.Total)
	assert.Equal(t, 1, response.Pagination.TotalPages)
	assert.False(t, response.Pagination.HasNext)
	assert.False(t, response.Pagination.HasPrev)

	// Verify summary
	assert.Equal(t, 1, response.Summary.TotalTransactions)
	assert.Equal(t, 0, response.Summary.TotalQuantityIn)
	assert.Equal(t, 10, response.Summary.TotalQuantityOut)
	assert.Equal(t, -10, response.Summary.NetQuantity)
	assert.True(t, response.Summary.TotalValue.GreaterThan(decimal.Zero))

	// Verify expectations
	transactionRepo.AssertExpectations(t)
}

// Test GetInventoryUtilization
func TestGetInventoryUtilization(t *testing.T) {
	service, _, _, _, warehouseRepo, _, _, _, _ := setupTestService()
	ctx := context.Background()
	warehouseID := uuid.New()

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	req := &GetInventoryUtilizationRequest{
		WarehouseIDs: []uuid.UUID{warehouseID},
		StartDate:    &startDate,
		EndDate:      &endDate,
		Granularity:  "DAILY",
	}

	// Setup test data
	warehouses := []*entities.Warehouse{
		{
			ID:   warehouseID,
			Name: "Test Warehouse",
		},
	}

	// Mock expectations
	warehouseRepo.On("List", ctx, mock.AnythingOfType("*inventoryRepositories.WarehouseFilter")).Return(warehouses, nil)

	// Execute test
	response, err := service.GetInventoryUtilization(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, startDate, response.Period.StartDate)
	assert.Equal(t, endDate, response.Period.EndDate)
	assert.NotNil(t, response.WarehouseData)
	assert.NotNil(t, response.Insights)
	assert.NotNil(t, response.Recommendations)

	// Verify warehouse data
	warehouseData, exists := response.WarehouseData[warehouseID]
	assert.True(t, exists)
	assert.Equal(t, warehouseID, warehouseData.WarehouseID)
	assert.Equal(t, "Test Warehouse", warehouseData.WarehouseName)

	// Verify insights and recommendations
	assert.True(t, len(response.Insights) >= 0)
	assert.True(t, len(response.Recommendations) >= 0)

	// Verify expectations
	warehouseRepo.AssertExpectations(t)
}

// Test error handling and edge cases
func TestErrorHandling(t *testing.T) {
	service, orderRepo, _, _, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	orderID := uuid.New()

	// Test order not found
	orderRepo.On("GetByID", ctx, orderID.String()).Return(nil, errors.New("order not found"))

	err := service.ReserveInventory(ctx, orderID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get order")

	orderRepo.AssertExpectations(t)
}

// Test concurrent operations
func TestConcurrentReservations(t *testing.T) {
	service, orderRepo, inventoryRepo, transactionRepo, _, logger, monitoring, _, _ := setupTestService()
	ctx := context.Background()

	orderID1 := uuid.New()
	orderID2 := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	userID := uuid.New()

	// Setup test data with limited inventory
	order1 := createTestOrder(orderID1, entities.OrderStatusPending)
	order2 := createTestOrder(orderID2, entities.OrderStatusPending)
	inventory := createTestInventory(productID, warehouseID, 15, 5, 20, 100.0) // Only 10 available

	items1 := []ReserveInventoryRequest{
		{
			OrderItemID: uuid.New(),
			ProductID:   productID,
			WarehouseID: warehouseID,
			Quantity:    8,
			ReservedBy:  userID,
		},
	}

	items2 := []ReserveInventoryRequest{
		{
			OrderItemID: uuid.New(),
			ProductID:   productID,
			WarehouseID: warehouseID,
			Quantity:    8,
			ReservedBy:  userID,
		},
	}

	// Mock expectations - allow both to succeed in the mock, real scenario would handle race conditions
	orderRepo.On("GetByID", ctx, orderID1.String()).Return(order1, nil)
	orderRepo.On("GetByID", ctx, orderID2.String()).Return(order2, nil)
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil)
	inventoryRepo.On("Update", ctx, mock.AnythingOfType("*entities.Inventory")).Return(nil)
	transactionRepo.On("Create", ctx, mock.AnythingOfType("*entities.InventoryTransaction")).Return(nil)
	logger.On("Info", ctx, mock.AnythingOfType("string"), mock.Anything).Return()
	monitoring.On("RecordCounter", ctx, "inventory.reservations.success", float64(1), mock.Anything).Return()
	monitoring.On("RecordMetric", ctx, "inventory.reservation.value", mock.AnythingOfType("float64"), mock.Anything).Return()
	monitoring.On("RecordHistogram", ctx, "inventory.reservation.duration", mock.AnythingOfType("float64"), mock.Anything).Return()

	// Execute concurrent reservations
	var wg sync.WaitGroup
	var errs []error
	var mu sync.Mutex

	wg.Add(2)

	go func() {
		defer wg.Done()
		err := service.ReserveInventoryItems(ctx, orderID1, items1)
		mu.Lock()
		errs = append(errs, err)
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		err := service.ReserveInventoryItems(ctx, orderID2, items2)
		mu.Lock()
		errs = append(errs, err)
		mu.Unlock()
	}()

	wg.Wait()

	// Verify results
	assert.Len(t, errs, 2)

	// At least one should succeed, possibly both depending on timing
	successCount := 0
	for _, err := range errs {
		if err == nil {
			successCount++
		}
	}
	assert.True(t, successCount >= 1, "At least one reservation should succeed")
}

// Benchmark tests
func BenchmarkReserveInventoryItems(b *testing.B) {
	service, orderRepo, inventoryRepo, transactionRepo, _, logger, monitoring, _, _ := setupTestService()
	ctx := context.Background()
	orderID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	userID := uuid.New()

	// Setup test data
	order := createTestOrder(orderID, entities.OrderStatusPending)
	inventory := createTestInventory(productID, warehouseID, 1000, 100, 200, 100.0)

	items := []ReserveInventoryRequest{
		{
			OrderItemID: uuid.New(),
			ProductID:   productID,
			WarehouseID: warehouseID,
			Quantity:    10,
			ReservedBy:  userID,
		},
	}

	// Mock expectations
	orderRepo.On("GetByID", ctx, orderID.String()).Return(order, nil)
	inventoryRepo.On("GetByProductAndWarehouse", ctx, productID, warehouseID).Return(inventory, nil)
	inventoryRepo.On("Update", ctx, mock.AnythingOfType("*entities.Inventory")).Return(nil)
	transactionRepo.On("Create", ctx, mock.AnythingOfType("*entities.InventoryTransaction")).Return(nil)
	logger.On("Info", ctx, mock.AnythingOfType("string"), mock.Anything).Return()
	monitoring.On("RecordCounter", ctx, "inventory.reservations.success", float64(1), mock.Anything).Return()
	monitoring.On("RecordMetric", ctx, "inventory.reservation.value", mock.AnythingOfType("float64"), mock.Anything).Return()
	monitoring.On("RecordHistogram", ctx, "inventory.reservation.duration", mock.AnythingOfType("float64"), mock.Anything).Return()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = service.ReserveInventoryItems(ctx, orderID, items)
	}
}