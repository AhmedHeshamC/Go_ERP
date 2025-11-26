package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"erpgo/internal/domain/inventory/entities"
	"erpgo/pkg/database"
)

// Shared test helper functions for repository tests

// setupTestDB creates a test database connection
func setupTestDB(t testing.TB) *database.Database {
	t.Helper()

	config := database.Config{
		URL:             "postgres://postgres:password@localhost:5432/test_erpgo?sslmode=disable",
		MaxConnections:  10,
		MinConnections:  1,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
		SSLMode:         "disable",
	}

	db, err := database.New(config)
	if err != nil {
		t.Skip("Skipping test - database not available:", err)
		return nil
	}

	return db
}

// cleanupTestDB cleans up the test database
func cleanupTestDB(t *testing.T) {
	t.Helper()
	// TODO: Implement test database cleanup
}

// Product is a simplified product type for testing
type Product struct {
	ID        uuid.UUID
	Name      string
	SKU       string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// createTestProduct creates a test product
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

	// TODO: Insert into database when needed
	return product
}

// createTestWarehouse creates a test warehouse
func createTestWarehouse(t *testing.T, db *database.Database, name, code string) *entities.Warehouse {
	t.Helper()
	warehouse := &entities.Warehouse{
		ID:         uuid.New(),
		Name:       name,
		Code:       code,
		Address:    "123 Test St",
		City:       "Test City",
		State:      "Test State",
		Country:    "Test Country",
		PostalCode: "12345",
		IsActive:   true,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	// TODO: Insert into database when needed
	return warehouse
}

// createTestInventory creates a test inventory record
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

	// TODO: Insert into database when needed
	return inventory
}

// createTestTransaction creates a test inventory transaction
func createTestTransaction(t *testing.T, db *database.Database, productID, warehouseID uuid.UUID, transactionType entities.TransactionType, quantity int) *entities.InventoryTransaction {
	t.Helper()
	return createTestTransactionWithUser(t, db, productID, warehouseID, transactionType, quantity, uuid.New())
}

// createTestTransactionWithUser creates a test inventory transaction with a specific user
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

	// TODO: Insert into database when needed
	return transaction
}

// setupTestSchema sets up the test database schema
func setupTestSchema(ctx context.Context, db *database.Database) error {
	// Drop tables if they exist
	queries := []string{
		`DROP TABLE IF EXISTS variant_images CASCADE`,
		`DROP TABLE IF EXISTS product_variants CASCADE`,
		`DROP TABLE IF EXISTS products CASCADE`,
		`DROP TABLE IF EXISTS product_categories CASCADE`,
	}

	for _, query := range queries {
		_, err := db.Exec(ctx, query)
		if err != nil {
			return err
		}
	}

	return nil
}
