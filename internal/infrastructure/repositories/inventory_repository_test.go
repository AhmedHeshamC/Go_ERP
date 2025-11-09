package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"erpgo/internal/domain/inventory/entities"
)

func TestPostgresWarehouseRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresWarehouseRepository(db)

	ctx := context.Background()
	warehouse := &entities.Warehouse{
		ID:         uuid.New(),
		Name:       "Test Warehouse",
		Code:       "TEST001",
		Address:    "123 Test St",
		City:       "Test City",
		State:      "Test State",
		Country:    "Test Country",
		PostalCode: "12345",
		Phone:      "+1234567890",
		Email:      "test@example.com",
		IsActive:   true,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	err := repo.Create(ctx, warehouse)
	assert.NoError(t, err)

	// Verify creation
	retrieved, err := repo.GetByID(ctx, warehouse.ID)
	assert.NoError(t, err)
	assert.Equal(t, warehouse.Name, retrieved.Name)
	assert.Equal(t, warehouse.Code, retrieved.Code)
}

func TestPostgresWarehouseRepository_GetByCode(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresWarehouseRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")

	retrieved, err := repo.GetByCode(ctx, "TEST001")
	assert.NoError(t, err)
	assert.Equal(t, warehouse.ID, retrieved.ID)
	assert.Equal(t, warehouse.Name, retrieved.Name)
}

func TestPostgresWarehouseRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresWarehouseRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")

	// Update warehouse
	warehouse.Name = "Updated Warehouse"
	warehouse.UpdatedAt = time.Now().UTC()
	err := repo.Update(ctx, warehouse)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetByID(ctx, warehouse.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Warehouse", retrieved.Name)
}

func TestPostgresWarehouseRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresWarehouseRepository(db)

	ctx := context.Background()
	warehouse1 := createTestWarehouse(t, db, "Warehouse A", "WH001")
	warehouse2 := createTestWarehouse(t, db, "Warehouse B", "WH002")

	// Test list all
	warehouses, err := repo.List(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, warehouses, 2)

	// Test filter by name
	filter := &WarehouseFilter{
		Name: "Warehouse A",
	}
	warehouses, err = repo.List(ctx, filter)
	assert.NoError(t, err)
	assert.Len(t, warehouses, 1)
	assert.Equal(t, warehouse1.ID, warehouses[0].ID)

	// Test filter by active status
	activeFilter := &WarehouseFilter{
		IsActive: &[]bool{true}[0],
	}
	warehouses, err = repo.List(ctx, activeFilter)
	assert.NoError(t, err)
	assert.Len(t, warehouses, 2)
}

func TestPostgresWarehouseRepository_Search(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresWarehouseRepository(db)

	ctx := context.Background()
	createTestWarehouse(t, db, "Test Warehouse", "TEST001")
	createTestWarehouse(t, db, "Distribution Center", "DIST001")

	// Search by name
	warehouses, err := repo.Search(ctx, "Test", 10)
	assert.NoError(t, err)
	assert.Len(t, warehouses, 1)

	// Search by code
	warehouses, err = repo.Search(ctx, "DIST", 10)
	assert.NoError(t, err)
	assert.Len(t, warehouses, 1)
}

func TestPostgresWarehouseRepository_GetWarehouseStats(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresWarehouseRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")

	stats, err := repo.GetWarehouseStats(ctx, warehouse.ID)
	assert.NoError(t, err)
	assert.Equal(t, warehouse.ID, stats.WarehouseID)
	assert.Equal(t, warehouse.Name, stats.WarehouseName)
	assert.Equal(t, warehouse.Code, stats.WarehouseCode)
}

func TestPostgresInventoryRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresInventoryRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")
	product := createTestProduct(t, db, "Test Product", "TEST001")

	inventory := &entities.Inventory{
		ID:               uuid.New(),
		ProductID:        product.ID,
		WarehouseID:      warehouse.ID,
		QuantityOnHand:   100,
		QuantityReserved: 10,
		ReorderLevel:     20,
		AverageCost:      50.0,
		UpdatedAt:        time.Now().UTC(),
		UpdatedBy:        uuid.New(),
	}

	err := repo.Create(ctx, inventory)
	assert.NoError(t, err)

	// Verify creation
	retrieved, err := repo.GetByProductAndWarehouse(ctx, product.ID, warehouse.ID)
	assert.NoError(t, err)
	assert.Equal(t, inventory.QuantityOnHand, retrieved.QuantityOnHand)
	assert.Equal(t, inventory.QuantityReserved, retrieved.QuantityReserved)
}

func TestPostgresInventoryRepository_AdjustStock(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresInventoryRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")
	product := createTestProduct(t, db, "Test Product", "TEST001")
	inventory := createTestInventory(t, db, product.ID, warehouse.ID, 100, 10, 20)

	// Adjust stock up
	err := repo.AdjustStock(ctx, product.ID, warehouse.ID, 50)
	assert.NoError(t, err)

	// Verify adjustment
	retrieved, err := repo.GetByProductAndWarehouse(ctx, product.ID, warehouse.ID)
	assert.NoError(t, err)
	assert.Equal(t, 150, retrieved.QuantityOnHand)

	// Adjust stock down
	err = repo.AdjustStock(ctx, product.ID, warehouse.ID, -30)
	assert.NoError(t, err)

	retrieved, err = repo.GetByProductAndWarehouse(ctx, product.ID, warehouse.ID)
	assert.NoError(t, err)
	assert.Equal(t, 120, retrieved.QuantityOnHand)
}

func TestPostgresInventoryRepository_ReserveStock(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresInventoryRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")
	product := createTestProduct(t, db, "Test Product", "TEST001")
	inventory := createTestInventory(t, db, product.ID, warehouse.ID, 100, 10, 20)

	// Reserve stock
	err := repo.ReserveStock(ctx, product.ID, warehouse.ID, 30)
	assert.NoError(t, err)

	// Verify reservation
	retrieved, err := repo.GetByProductAndWarehouse(ctx, product.ID, warehouse.ID)
	assert.NoError(t, err)
	assert.Equal(t, 40, retrieved.QuantityReserved) // 10 + 30

	availableStock, err := repo.GetAvailableStock(ctx, product.ID, warehouse.ID)
	assert.NoError(t, err)
	assert.Equal(t, 60, availableStock) // 100 - 40

	// Release stock
	err = repo.ReleaseStock(ctx, product.ID, warehouse.ID, 20)
	assert.NoError(t, err)

	retrieved, err = repo.GetByProductAndWarehouse(ctx, product.ID, warehouse.ID)
	assert.NoError(t, err)
	assert.Equal(t, 20, retrieved.QuantityReserved) // 40 - 20
}

func TestPostgresInventoryRepository_GetLowStockItems(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresInventoryRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")
	product1 := createTestProduct(t, db, "Product 1", "P001")
	product2 := createTestProduct(t, db, "Product 2", "P002")

	// Create inventory with low stock
	createTestInventory(t, db, product1.ID, warehouse.ID, 15, 5, 20) // Below reorder level
	// Create inventory with normal stock
	createTestInventory(t, db, product2.ID, warehouse.ID, 100, 10, 20) // Above reorder level

	lowStockItems, err := repo.GetLowStockItems(ctx, warehouse.ID)
	assert.NoError(t, err)
	assert.Len(t, lowStockItems, 1)
	assert.Equal(t, product1.ID, lowStockItems[0].ProductID)
}

func TestPostgresInventoryRepository_BulkOperations(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresInventoryRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")
	product1 := createTestProduct(t, db, "Product 1", "P001")
	product2 := createTestProduct(t, db, "Product 2", "P002")

	// Test bulk create
	inventories := []*entities.Inventory{
		{
			ID:               uuid.New(),
			ProductID:        product1.ID,
			WarehouseID:      warehouse.ID,
			QuantityOnHand:   100,
			QuantityReserved: 0,
			ReorderLevel:     20,
			AverageCost:      50.0,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        uuid.New(),
		},
		{
			ID:               uuid.New(),
			ProductID:        product2.ID,
			WarehouseID:      warehouse.ID,
			QuantityOnHand:   200,
			QuantityReserved: 0,
			ReorderLevel:     50,
			AverageCost:      75.0,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        uuid.New(),
		},
	}

	err := repo.BulkCreate(ctx, inventories)
	assert.NoError(t, err)

	// Verify creation
	for _, inv := range inventories {
		retrieved, err := repo.GetByProductAndWarehouse(ctx, inv.ProductID, inv.WarehouseID)
		assert.NoError(t, err)
		assert.Equal(t, inv.QuantityOnHand, retrieved.QuantityOnHand)
	}

	// Test bulk adjust
	adjustments := []StockAdjustment{
		{
			ProductID:   product1.ID,
			WarehouseID: warehouse.ID,
			Adjustment:  -10,
			Reason:      "Test adjustment",
			UpdatedBy:   uuid.New(),
		},
		{
			ProductID:   product2.ID,
			WarehouseID: warehouse.ID,
			Adjustment:  25,
			Reason:      "Test adjustment",
			UpdatedBy:   uuid.New(),
		},
	}

	err = repo.BulkAdjustStock(ctx, adjustments)
	assert.NoError(t, err)

	// Verify adjustments
	retrieved1, _ := repo.GetByProductAndWarehouse(ctx, product1.ID, warehouse.ID)
	assert.Equal(t, 90, retrieved1.QuantityOnHand)

	retrieved2, _ := repo.GetByProductAndWarehouse(ctx, product2.ID, warehouse.ID)
	assert.Equal(t, 225, retrieved2.QuantityOnHand)
}

func TestPostgresInventoryTransactionRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresInventoryTransactionRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")
	product := createTestProduct(t, db, "Test Product", "TEST001")

	transaction := &entities.InventoryTransaction{
		ID:              uuid.New(),
		ProductID:       product.ID,
		WarehouseID:     warehouse.ID,
		TransactionType: entities.TransactionTypePurchase,
		Quantity:        100,
		UnitCost:        50.0,
		TotalCost:       5000.0,
		BatchNumber:     "BATCH001",
		CreatedAt:       time.Now().UTC(),
		CreatedBy:       uuid.New(),
	}

	err := repo.Create(ctx, transaction)
	assert.NoError(t, err)

	// Verify creation
	retrieved, err := repo.GetByID(ctx, transaction.ID)
	assert.NoError(t, err)
	assert.Equal(t, transaction.TransactionType, retrieved.TransactionType)
	assert.Equal(t, transaction.Quantity, retrieved.Quantity)
	assert.Equal(t, transaction.BatchNumber, retrieved.BatchNumber)
}

func TestPostgresInventoryTransactionRepository_GetByProduct(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresInventoryTransactionRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")
	product := createTestProduct(t, db, "Test Product", "TEST001")

	// Create multiple transactions
	transaction1 := createTestTransaction(t, db, product.ID, warehouse.ID, entities.TransactionTypePurchase, 100)
	transaction2 := createTestTransaction(t, db, product.ID, warehouse.ID, entities.TransactionTypeSale, -50)

	filter := &TransactionFilter{
		ProductIDs: []uuid.UUID{product.ID},
	}

	transactions, err := repo.GetByProduct(ctx, product.ID, filter)
	assert.NoError(t, err)
	assert.Len(t, transactions, 2)

	// Verify transactions are ordered by created_at DESC (most recent first)
	assert.True(t, transactions[0].CreatedAt.After(transactions[1].CreatedAt) ||
		transactions[0].CreatedAt.Equal(transactions[1].CreatedAt))
}

func TestPostgresInventoryTransactionRepository_GetByType(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresInventoryTransactionRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")
	product1 := createTestProduct(t, db, "Product 1", "P001")
	product2 := createTestProduct(t, db, "Product 2", "P002")

	// Create transactions of different types
	createTestTransaction(t, db, product1.ID, warehouse.ID, entities.TransactionTypePurchase, 100)
	createTestTransaction(t, db, product2.ID, warehouse.ID, entities.TransactionTypePurchase, 200)
	createTestTransaction(t, db, product1.ID, warehouse.ID, entities.TransactionTypeSale, -50)

	// Get only purchase transactions
	transactions, err := repo.GetByType(ctx, entities.TransactionTypePurchase, nil)
	assert.NoError(t, err)
	assert.Len(t, transactions, 2)

	for _, tx := range transactions {
		assert.Equal(t, entities.TransactionTypePurchase, tx.TransactionType)
		assert.Greater(t, tx.Quantity, 0) // Purchase transactions should have positive quantities
	}
}

func TestPostgresInventoryTransactionRepository_ApproveTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresInventoryTransactionRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")
	product := createTestProduct(t, db, "Test Product", "TEST001")
	transaction := createTestTransaction(t, db, product.ID, warehouse.ID, entities.TransactionTypePurchase, 100)

	// Approve transaction
	approverID := uuid.New()
	err := repo.ApproveTransaction(ctx, transaction.ID, approverID)
	assert.NoError(t, err)

	// Verify approval
	retrieved, err := repo.GetByID(ctx, transaction.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved.ApprovedAt)
	assert.NotNil(t, retrieved.ApprovedBy)
	assert.Equal(t, approverID, *retrieved.ApprovedBy)
}

func TestPostgresInventoryTransactionRepository_GetTransactionSummary(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresInventoryTransactionRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")
	product := createTestProduct(t, db, "Test Product", "TEST001")

	// Create transactions
	createTestTransaction(t, db, product.ID, warehouse.ID, entities.TransactionTypePurchase, 100)
	createTestTransaction(t, db, product.ID, warehouse.ID, entities.TransactionTypePurchase, 50)
	createTestTransaction(t, db, product.ID, warehouse.ID, entities.TransactionTypeSale, -75)

	startDate := time.Now().Add(-24 * time.Hour)
	endDate := time.Now().Add(24 * time.Hour)
	filter := &TransactionFilter{
		DateFrom: &startDate,
		DateTo:   &endDate,
	}

	summary, err := repo.GetTransactionSummary(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 3, summary.TotalTransactions)
	assert.Equal(t, 150, summary.TotalQuantityIn) // 100 + 50
	assert.Equal(t, 75, summary.TotalQuantityOut)  // 75
	assert.Contains(t, summary.TransactionsByType, entities.TransactionTypePurchase)
	assert.Contains(t, summary.TransactionsByType, entities.TransactionTypeSale)
}

func TestPostgresInventoryTransactionRepository_GetAuditTrail(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)
	repo := NewPostgresInventoryTransactionRepository(db)

	ctx := context.Background()
	warehouse := createTestWarehouse(t, db, "Test Warehouse", "TEST001")
	product := createTestProduct(t, db, "Test Product", "TEST001")
	userID := uuid.New()

	// Create transactions
	transaction1 := createTestTransactionWithUser(t, db, product.ID, warehouse.ID, entities.TransactionTypePurchase, 100, userID)
	transaction2 := createTestTransactionWithUser(t, db, product.ID, warehouse.ID, entities.TransactionTypeSale, -50, userID)

	startDate := time.Now().Add(-24 * time.Hour)
	endDate := time.Now().Add(24 * time.Hour)
	filter := &AuditFilter{
		UserIDs:         []uuid.UUID{userID},
		StartDate:       startDate,
		EndDate:         endDate,
		IncludeApproved: &[]bool{true}[0],
		IncludePending:  &[]bool{true}[0],
	}

	auditTrail, err := repo.GetAuditTrail(ctx, filter)
	assert.NoError(t, err)
	assert.Len(t, auditTrail, 2)

	// Verify transactions are ordered by created_at DESC
	assert.True(t, auditTrail[0].CreatedAt.After(auditTrail[1].CreatedAt) ||
		auditTrail[0].CreatedAt.Equal(auditTrail[1].CreatedAt))

	// Verify all transactions belong to the specified user
	for _, tx := range auditTrail {
		assert.Equal(t, userID, tx.CreatedBy)
	}
}

// Helper functions

func setupTestDB(t *testing.T) *database.Database {
	// This would set up a test database connection
	// For now, return a mock implementation
	t.Helper()
	// TODO: Implement test database setup
	return nil
}

func cleanupTestDB(t *testing.T) {
	// This would clean up the test database
	t.Helper()
	// TODO: Implement test database cleanup
}

func createTestWarehouse(t *testing.T, db *database.Database, name, code string) *entities.Warehouse {
	t.Helper()
	warehouse := &entities.Warehouse{
		ID:        uuid.New(),
		Name:      name,
		Code:      code,
		Address:   "123 Test St",
		City:      "Test City",
		State:     "Test State",
		Country:   "Test Country",
		PostalCode: "12345",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// TODO: Insert into database
	return warehouse
}

func createTestProduct(t *testing.T, db *database.Database, name, sku string) *Product {
	t.Helper()
	product := &Product{
		ID:        uuid.New(),
		Name:      name,
		SKU:       sku,
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// TODO: Insert into database
	return product
}

func createTestInventory(t *testing.T, db *database.Database, productID, warehouseID uuid.UUID, quantityOnHand, quantityReserved, reorderLevel int) *entities.Inventory {
	t.Helper()
	inventory := &entities.Inventory{
		ID:               uuid.New(),
		ProductID:        productID,
		WarehouseID:      warehouseID,
		QuantityOnHand:   quantityOnHand,
		QuantityReserved: quantityReserved,
		ReorderLevel:     reorderLevel,
		AverageCost:      50.0,
		UpdatedAt:        time.Now().UTC(),
		UpdatedBy:        uuid.New(),
	}

	// TODO: Insert into database
	return inventory
}

func createTestTransaction(t *testing.T, db *database.Database, productID, warehouseID uuid.UUID, transactionType entities.TransactionType, quantity int) *entities.InventoryTransaction {
	t.Helper()
	return createTestTransactionWithUser(t, db, productID, warehouseID, transactionType, quantity, uuid.New())
}

func createTestTransactionWithUser(t *testing.T, db *database.Database, productID, warehouseID uuid.UUID, transactionType entities.TransactionType, quantity int, userID uuid.UUID) *entities.InventoryTransaction {
	t.Helper()
	transaction := &entities.InventoryTransaction{
		ID:              uuid.New(),
		ProductID:       productID,
		WarehouseID:     warehouseID,
		TransactionType: transactionType,
		Quantity:        quantity,
		UnitCost:        50.0,
		TotalCost:       float64(quantity) * 50.0,
		BatchNumber:     "BATCH001",
		CreatedAt:       time.Now().UTC(),
		CreatedBy:       userID,
	}

	// TODO: Insert into database
	return transaction
}

// Mock Product type for testing
type Product struct {
	ID        uuid.UUID
	Name      string
	SKU       string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}