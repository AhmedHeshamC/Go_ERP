package inventory

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"erpgo/internal/domain/inventory/entities"
	"erpgo/internal/domain/inventory/repositories"
	"erpgo/internal/interfaces/http/dto"
	"erpgo/pkg/database"
)

// MockInventoryRepository for testing
type MockInventoryRepository struct {
	mock.Mock
}

func (m *MockInventoryRepository) Create(ctx context.Context, inventory *entities.Inventory) error {
	args := m.Called(ctx, inventory)
	return args.Error(0)
}

func (m *MockInventoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Inventory, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Inventory), args.Error(1)
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

func (m *MockInventoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInventoryRepository) List(ctx context.Context, filter *repositories.InventoryFilter) ([]*entities.Inventory, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) Count(ctx context.Context, filter *repositories.InventoryFilter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

func (m *MockInventoryRepository) AdjustStock(ctx context.Context, productID, warehouseID uuid.UUID, adjustment int) error {
	args := m.Called(ctx, productID, warehouseID, adjustment)
	return args.Error(0)
}

func (m *MockInventoryRepository) ReserveStock(ctx context.Context, productID, warehouseID uuid.UUID, quantity int) error {
	args := m.Called(ctx, productID, warehouseID, quantity)
	return args.Error(0)
}

func (m *MockInventoryRepository) ReleaseStock(ctx context.Context, productID, warehouseID uuid.UUID, quantity int) error {
	args := m.Called(ctx, productID, warehouseID, quantity)
	return args.Error(0)
}

func (m *MockInventoryRepository) GetAvailableStock(ctx context.Context, productID, warehouseID uuid.UUID) (int, error) {
	args := m.Called(ctx, productID, warehouseID)
	return args.Int(0), args.Error(1)
}

func (m *MockInventoryRepository) GetLowStockItems(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error) {
	args := m.Called(ctx, warehouseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) GetLowStockItemsAll(ctx context.Context) ([]*entities.Inventory, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

// MockWarehouseRepository for testing
type MockWarehouseRepository struct {
	mock.Mock
}

func (m *MockWarehouseRepository) Create(ctx context.Context, warehouse *entities.Warehouse) error {
	args := m.Called(ctx, warehouse)
	return args.Error(0)
}

func (m *MockWarehouseRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Warehouse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) Update(ctx context.Context, warehouse *entities.Warehouse) error {
	args := m.Called(ctx, warehouse)
	return args.Error(0)
}

func (m *MockWarehouseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWarehouseRepository) List(ctx context.Context, filter *repositories.WarehouseFilter) ([]*entities.Warehouse, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) Count(ctx context.Context, filter *repositories.WarehouseFilter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

// MockInventoryTransactionRepository for testing
type MockInventoryTransactionRepository struct {
	mock.Mock
}

func (m *MockInventoryTransactionRepository) Create(ctx context.Context, transaction *entities.InventoryTransaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *MockInventoryTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.InventoryTransaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) List(ctx context.Context, filter *repositories.InventoryTransactionFilter) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) Count(ctx context.Context, filter *repositories.InventoryTransactionFilter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

// MockTransactionManager for testing
type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) != nil {
		return args.Get(0).(func(context.Context, func(tx pgx.Tx) error) error)(ctx, fn)
	}
	// Default: execute function without transaction
	return fn(nil)
}

func (m *MockTransactionManager) WithRetryTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) != nil {
		return args.Get(0).(func(context.Context, func(tx pgx.Tx) error) error)(ctx, fn)
	}
	// Default: execute function without transaction
	return fn(nil)
}

func (m *MockTransactionManager) WithTransactionOptions(ctx context.Context, opts database.TransactionConfig, fn func(tx pgx.Tx) error) error {
	args := m.Called(ctx, opts, fn)
	if args.Get(0) != nil {
		return args.Get(0).(func(context.Context, database.TransactionConfig, func(tx pgx.Tx) error) error)(ctx, opts, fn)
	}
	// Default: execute function without transaction
	return fn(nil)
}

// TestTransferInventoryUsesTransaction verifies that TransferInventory uses transactions
func TestTransferInventoryUsesTransaction(t *testing.T) {
	// Setup
	mockInventoryRepo := new(MockInventoryRepository)
	mockWarehouseRepo := new(MockWarehouseRepository)
	mockTransactionRepo := new(MockInventoryTransactionRepository)
	mockTxManager := new(MockTransactionManager)
	logger := zerolog.Nop()

	service := NewService(mockInventoryRepo, mockWarehouseRepo, mockTransactionRepo, mockTxManager, &logger)

	ctx := context.Background()
	productID := uuid.New()
	fromWarehouseID := uuid.New()
	toWarehouseID := uuid.New()

	req := &dto.TransferInventoryRequest{
		ProductID:       productID,
		FromWarehouseID: fromWarehouseID,
		ToWarehouseID:   toWarehouseID,
		Quantity:        10,
	}

	// Mock expectations
	mockInventoryRepo.On("GetAvailableStock", ctx, productID, fromWarehouseID).Return(20, nil)
	
	// Verify that WithRetryTransaction is called
	transactionCalled := false
	mockTxManager.On("WithRetryTransaction", ctx, mock.AnythingOfType("func(pgx.Tx) error")).
		Return(func(ctx context.Context, fn func(tx pgx.Tx) error) error {
			transactionCalled = true
			// Execute the transaction function
			mockTransactionRepo.On("Create", ctx, mock.AnythingOfType("*entities.InventoryTransaction")).Return(nil).Times(2)
			mockInventoryRepo.On("AdjustStock", ctx, productID, fromWarehouseID, -10).Return(nil)
			mockInventoryRepo.On("AdjustStock", ctx, productID, toWarehouseID, 10).Return(nil)
			return fn(nil)
		})

	// Execute
	result, err := service.TransferInventory(nil, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, transactionCalled, "WithRetryTransaction should have been called")
	
	mockInventoryRepo.AssertExpectations(t)
	mockTransactionRepo.AssertExpectations(t)
	mockTxManager.AssertExpectations(t)
}
