package inventory

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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
	return args.Get(0).(*entities.Inventory), args.Error(1)
}

// BulkAdjustStock mocks the BulkAdjustStock method
func (m *MockInventoryRepository) BulkAdjustStock(ctx context.Context, adjustments []repositories.StockAdjustment) error {
	args := m.Called(ctx, adjustments)
	return args.Error(0)
}

// GetByProductAndWarehouse mocks the GetByProductAndWarehouse method
func (m *MockInventoryRepository) GetByProductAndWarehouse(ctx context.Context, productID, warehouseID uuid.UUID) (*entities.Inventory, error) {
	args := m.Called(ctx, productID, warehouseID)
	return args.Get(0).(*entities.Inventory), args.Error(1)
}

// GetLowStockItems mocks the GetLowStockItems method
func (m *MockInventoryRepository) GetLowStockItems(ctx context.Context, warehouseID *uuid.UUID) ([]*entities.Inventory, error) {
	args := m.Called(ctx, warehouseID)
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

// GetLowStockItemsAll mocks the GetLowStockItemsAll method
func (m *MockInventoryRepository) GetLowStockItemsAll(ctx context.Context) ([]*entities.Inventory, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entities.Inventory), args.Error(1)
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

func (m *MockInventoryRepository) BulkAdjustStock(ctx context.Context, adjustments []repositories.StockAdjustment) error {
	args := m.Called(ctx, adjustments)
	return args.Error(0)
}

func (m *MockInventoryRepository) BulkCreate(ctx context.Context, inventories []*entities.Inventory) error {
	args := m.Called(ctx, inventories)
	return args.Error(0)
}

func (m *MockInventoryRepository) BulkDelete(ctx context.Context, inventoryIDs []uuid.UUID) error {
	args := m.Called(ctx, inventoryIDs)
	return args.Error(0)
}

func (m *MockInventoryRepository) BulkReserveStock(ctx context.Context, reservations []repositories.StockReservation) error {
	args := m.Called(ctx, reservations)
	return args.Error(0)
}

func (m *MockInventoryRepository) BulkUpdate(ctx context.Context, inventories []*entities.Inventory) error {
	args := m.Called(ctx, inventories)
	return args.Error(0)
}

func (m *MockInventoryRepository) CountByProduct(ctx context.Context, productID uuid.UUID) (int, error) {
	args := m.Called(ctx, productID)
	return args.Int(0), args.Error(1)
}

func (m *MockInventoryRepository) CountByWarehouse(ctx context.Context, warehouseID uuid.UUID) (int, error) {
	args := m.Called(ctx, warehouseID)
	return args.Int(0), args.Error(1)
}

func (m *MockInventoryRepository) ExistsByProductAndWarehouse(ctx context.Context, productID, warehouseID uuid.UUID) (bool, error) {
	args := m.Called(ctx, productID, warehouseID)
	return args.Bool(0), args.Error(1)
}

func (m *MockInventoryRepository) GetAgingInventory(ctx context.Context, warehouseID *uuid.UUID, days int) ([]*repositories.AgingInventoryItem, error) {
	args := m.Called(ctx, warehouseID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repositories.AgingInventoryItem), args.Error(1)
}

func (m *MockInventoryRepository) GetByProduct(ctx context.Context, productID uuid.UUID) ([]*entities.Inventory, error) {
	args := m.Called(ctx, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) GetByWarehouse(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error) {
	args := m.Called(ctx, warehouseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) GetInventoryLevels(ctx context.Context, productID uuid.UUID) ([]*repositories.InventoryLevel, error) {
	args := m.Called(ctx, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repositories.InventoryLevel), args.Error(1)
}

func (m *MockInventoryRepository) GetInventoryTurnover(ctx context.Context, productID uuid.UUID, warehouseID *uuid.UUID, days int) (*repositories.InventoryTurnover, error) {
	args := m.Called(ctx, productID, warehouseID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.InventoryTurnover), args.Error(1)
}

func (m *MockInventoryRepository) GetItemsForCycleCount(ctx context.Context, warehouseID uuid.UUID, limit int) ([]*entities.Inventory, error) {
	args := m.Called(ctx, warehouseID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) GetLastCycleCountDate(ctx context.Context, inventoryID uuid.UUID) (*time.Time, error) {
	args := m.Called(ctx, inventoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

func (m *MockInventoryRepository) GetOutOfStockItems(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error) {
	args := m.Called(ctx, warehouseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) GetOverstockItems(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error) {
	args := m.Called(ctx, warehouseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) GetProductInventory(ctx context.Context, productID uuid.UUID) ([]*entities.Inventory, error) {
	args := m.Called(ctx, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) GetWarehouseInventory(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error) {
	args := m.Called(ctx, warehouseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) ReconcileStock(ctx context.Context, inventoryID uuid.UUID, systemQuantity, physicalQuantity int, reason string, reconciledBy uuid.UUID) error {
	args := m.Called(ctx, inventoryID, systemQuantity, physicalQuantity, reason, reconciledBy)
	return args.Error(0)
}

func (m *MockInventoryRepository) GetReconciliationHistory(ctx context.Context, inventoryID uuid.UUID, limit int) ([]*repositories.InventoryReconciliation, error) {
	args := m.Called(ctx, inventoryID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repositories.InventoryReconciliation), args.Error(1)
}

func (m *MockInventoryRepository) Search(ctx context.Context, query string, limit int) ([]*entities.Inventory, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) UpdateCycleCount(ctx context.Context, inventoryID uuid.UUID, countedQuantity int, countedBy uuid.UUID) error {
	args := m.Called(ctx, inventoryID, countedQuantity, countedBy)
	return args.Error(0)
}

func (m *MockInventoryRepository) UpdateStock(ctx context.Context, productID, warehouseID uuid.UUID, quantity int) error {
	args := m.Called(ctx, productID, warehouseID, quantity)
	return args.Error(0)
}

func (m *MockInventoryRepository) GetStockLevels(ctx context.Context, filter *repositories.InventoryFilter) ([]*repositories.StockLevel, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repositories.StockLevel), args.Error(1)
}

func (m *MockInventoryRepository) GetInventoryValue(ctx context.Context, warehouseID *uuid.UUID) (float64, error) {
	args := m.Called(ctx, warehouseID)
	return args.Get(0).(float64), args.Error(1)
}

// MockWarehouseRepository for testing
type MockWarehouseRepository struct {
	mock.Mock
}

// BulkAssignManager mocks the BulkAssignManager method
func (m *MockWarehouseRepository) BulkAssignManager(ctx context.Context, assignments []repositories.WarehouseManagerAssignment) error {
	args := m.Called(ctx, assignments)
	return args.Error(0)
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

func (m *MockWarehouseRepository) BulkAssignManager(ctx context.Context, warehouseIDs []uuid.UUID, managerID uuid.UUID) error {
	args := m.Called(ctx, warehouseIDs, managerID)
	return args.Error(0)
}

func (m *MockWarehouseRepository) BulkUpdateStatus(ctx context.Context, warehouseIDs []uuid.UUID, isActive bool) error {
	args := m.Called(ctx, warehouseIDs, isActive)
	return args.Error(0)
}

func (m *MockWarehouseRepository) CreateExtended(ctx context.Context, warehouse *entities.WarehouseExtended) error {
	args := m.Called(ctx, warehouse)
	return args.Error(0)
}

func (m *MockWarehouseRepository) GetExtendedByID(ctx context.Context, id uuid.UUID) (*entities.WarehouseExtended, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.WarehouseExtended), args.Error(1)
}

func (m *MockWarehouseRepository) UpdateExtended(ctx context.Context, warehouse *entities.WarehouseExtended) error {
	args := m.Called(ctx, warehouse)
	return args.Error(0)
}

func (m *MockWarehouseRepository) GetByType(ctx context.Context, warehouseType entities.WarehouseType) ([]*entities.WarehouseExtended, error) {
	args := m.Called(ctx, warehouseType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.WarehouseExtended), args.Error(1)
}

func (m *MockWarehouseRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	args := m.Called(ctx, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockWarehouseRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockWarehouseRepository) GetActive(ctx context.Context) ([]*entities.Warehouse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) GetAllWarehouseStats(ctx context.Context) ([]*repositories.WarehouseStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repositories.WarehouseStats), args.Error(1)
}

func (m *MockWarehouseRepository) GetByCode(ctx context.Context, code string) (*entities.Warehouse, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) GetByLocation(ctx context.Context, city, state, country string) ([]*entities.Warehouse, error) {
	args := m.Called(ctx, city, state, country)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) GetByManager(ctx context.Context, managerID uuid.UUID) ([]*entities.Warehouse, error) {
	args := m.Called(ctx, managerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) GetCapacityUtilization(ctx context.Context, warehouseID uuid.UUID) (*repositories.CapacityUtilization, error) {
	args := m.Called(ctx, warehouseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.CapacityUtilization), args.Error(1)
}

func (m *MockWarehouseRepository) GetWarehouseStats(ctx context.Context, warehouseID uuid.UUID) (*repositories.WarehouseStats, error) {
	args := m.Called(ctx, warehouseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.WarehouseStats), args.Error(1)
}

func (m *MockWarehouseRepository) Search(ctx context.Context, query string, limit int) ([]*entities.Warehouse, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Warehouse), args.Error(1)
}

// MockInventoryTransactionRepository for testing
type MockInventoryTransactionFilter struct {
	mock.Mock
}

// Filter mocks the Filter method
func (m *MockInventoryTransactionRepository) Filter(ctx context.Context, filter repositories.InventoryTransactionFilter) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) Create(ctx context.Context, transaction *entities.InventoryTransaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

// ApproveTransaction mocks the ApproveTransaction method
func (m *MockInventoryTransactionRepository) ApproveTransaction(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInventoryTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.InventoryTransaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) List(ctx context.Context, filter *repositories.TransactionFilter) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) Count(ctx context.Context, filter *repositories.TransactionFilter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

func (m *MockInventoryTransactionRepository) ApproveTransaction(ctx context.Context, transactionID uuid.UUID, approvedBy uuid.UUID) error {
	args := m.Called(ctx, transactionID, approvedBy)
	return args.Error(0)
}

func (m *MockInventoryTransactionRepository) BulkApprove(ctx context.Context, transactionIDs []uuid.UUID, approvedBy uuid.UUID) error {
	args := m.Called(ctx, transactionIDs, approvedBy)
	return args.Error(0)
}

func (m *MockInventoryTransactionRepository) BulkCreate(ctx context.Context, transactions []*entities.InventoryTransaction) error {
	args := m.Called(ctx, transactions)
	return args.Error(0)
}

func (m *MockInventoryTransactionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInventoryTransactionRepository) GetAuditTrail(ctx context.Context, filter *repositories.AuditFilter) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetByBatch(ctx context.Context, batchNumber string) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, batchNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetByDateRange(ctx context.Context, warehouseID *uuid.UUID, startDate, endDate time.Time, filter *repositories.TransactionFilter) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, warehouseID, startDate, endDate, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetByReference(ctx context.Context, referenceType string, referenceID uuid.UUID) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, referenceType, referenceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetByType(ctx context.Context, transactionType entities.TransactionType, filter *repositories.TransactionFilter) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, transactionType, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetByWarehouse(ctx context.Context, warehouseID uuid.UUID, filter *repositories.TransactionFilter) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, warehouseID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetByProduct(ctx context.Context, productID uuid.UUID, filter *repositories.TransactionFilter) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, productID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetComplianceReport(ctx context.Context, startDate, endDate time.Time) (*repositories.ComplianceReport, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.ComplianceReport), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetCostOfGoodsSold(ctx context.Context, startDate, endDate time.Time) (float64, error) {
	args := m.Called(ctx, startDate, endDate)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetInventoryMovement(ctx context.Context, filter *repositories.MovementFilter) ([]*repositories.InventoryMovement, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repositories.InventoryMovement), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetPendingApproval(ctx context.Context, warehouseID *uuid.UUID) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, warehouseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) RejectTransaction(ctx context.Context, transactionID uuid.UUID, rejectedBy uuid.UUID, reason string) error {
	args := m.Called(ctx, transactionID, rejectedBy, reason)
	return args.Error(0)
}

func (m *MockInventoryTransactionRepository) GetTransactionSummary(ctx context.Context, filter *repositories.TransactionFilter) (*repositories.TransactionSummary, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.TransactionSummary), args.Error(1)
}

func (m *MockInventoryTransactionRepository) Update(ctx context.Context, transaction *entities.InventoryTransaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *MockInventoryTransactionRepository) GetRecentTransactions(ctx context.Context, warehouseID *uuid.UUID, hours int, limit int) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, warehouseID, hours, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetTransferTransactions(ctx context.Context, fromWarehouseID, toWarehouseID uuid.UUID) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, fromWarehouseID, toWarehouseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetPendingTransfers(ctx context.Context, warehouseID *uuid.UUID) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, warehouseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetByProductAndWarehouse(ctx context.Context, productID, warehouseID uuid.UUID, filter *repositories.TransactionFilter) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, productID, warehouseID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) GetTransactionHistory(ctx context.Context, productID uuid.UUID, warehouseID *uuid.UUID, limit int) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, productID, warehouseID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) Search(ctx context.Context, query string, limit int) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

// MockTransactionManager for testing
type MockTransactionManager struct {
	mock.Mock
}

// WithTransactionOptions mocks the WithTransactionOptions method
func (m *MockTransactionManager) WithTransactionOptions(ctx context.Context, opts database.TransactionConfig, fn func(tx pgx.Tx) error) error {
	args := m.Called(ctx, opts, fn)
	if args.Get(0) != nil {
		return args.Get(0).(func(context.Context, database.TransactionConfig, func(tx pgx.Tx) error) error)(ctx, opts, fn)
	}
	// Default: execute function without transaction
	return fn(nil)
}

// Filter mocks the Filter method
func (m *MockTransactionFilter) Filter(ctx context.Context, filter repositories.InventoryTransactionFilter) ([]*entities.InventoryTransaction, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*entities.InventoryTransaction), args.Error(1)
}

// WithTransactionOptions mocks the WithTransactionOptions method
func (m *MockTransactionManager) WithTransactionOptions(ctx context.Context, opts database.TransactionConfig, fn func(tx pgx.Tx) error) error {
	args := m.Called(ctx, opts, fn)
	if args.Get(0) != nil {
		return args.Get(0).(func(context.Context, database.TransactionConfig, func(tx pgx.Tx) error) error)(ctx, opts, fn)
	}
	// Default: execute function without transaction
	return fn(nil)
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
		// Execute the provided callback function within the mock transaction
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
			// Create transactions directly without using a nil transaction
			outboundTx := &entities.InventoryTransaction{
				ID:              uuid.New(),
				ProductID:       productID,
				WarehouseID:     fromWarehouseID,
				TransactionType: entities.TransactionTypeTransferOut,
				Quantity:        -req.Quantity,
				Reason:          fmt.Sprintf("Transfer to warehouse %s", toWarehouseID.String()),
				CreatedAt:       time.Now().UTC(),
			}

			inboundTx := &entities.InventoryTransaction{
				ID:              uuid.New(),
				ProductID:       productID,
				WarehouseID:     toWarehouseID,
				TransactionType: entities.TransactionTypeTransferIn,
				Quantity:        req.Quantity,
				Reason:          fmt.Sprintf("Transfer from warehouse %s", fromWarehouseID.String()),
				CreatedAt:       time.Now().UTC(),
			}

			// Create transactions directly (no actual transaction)
			if err := mockTransactionRepo.Create(ctx, outboundTx); err != nil {
				return fmt.Errorf("failed to create outbound transaction: %w", err)
			}

			if err := mockTransactionRepo.Create(ctx, inboundTx); err != nil {
				return fmt.Errorf("failed to create inbound transaction: %w", err)
			}

			// Adjust inventory directly
			if err := mockInventoryRepo.AdjustStock(ctx, productID, fromWarehouseID, -req.Quantity); err != nil {
				return fmt.Errorf("failed to adjust source inventory: %w", err)
			}

			if err := mockInventoryRepo.AdjustStock(ctx, productID, toWarehouseID, req.Quantity); err != nil {
				return fmt.Errorf("failed to adjust destination inventory: %w", err)
			}

			return nil
		})

	// Create a gin context with a background context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = c.Request.WithContext(ctx)

	// Execute
	result, err := service.TransferInventory(c, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, transactionCalled, "WithRetryTransaction should have been called")

	mockInventoryRepo.AssertExpectations(t)
	mockTransactionRepo.AssertExpectations(t)
	mockTxManager.AssertExpectations(t)
}
