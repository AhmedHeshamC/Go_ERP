package repositories

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"erpgo/internal/domain/products/entities"
	"erpgo/internal/domain/products/repositories"
	"erpgo/pkg/database"
)

// Test setup helper functions - product-specific helpers

func createTestCategory(t testing.TB) *entities.ProductCategory {
	t.Helper()

	return &entities.ProductCategory{
		ID:          uuid.New(),
		Name:        "Test Category",
		Description: "Test category description",
		ParentID:    nil,
		Level:       0,
		Path:        "/test-category",
		ImageURL:    "https://example.com/image.jpg",
		SortOrder:   1,
		IsActive:    true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

func createTestProductEntity(t testing.TB, categoryID uuid.UUID) *entities.Product {
	t.Helper()

	return &entities.Product{
		ID:                uuid.New(),
		SKU:               "TEST-001",
		Name:              "Test Product",
		Description:       "Test product description",
		ShortDescription:  "Test short description",
		CategoryID:        categoryID,
		Price:             decimal.NewFromFloat(99.99),
		Cost:              decimal.NewFromFloat(50.00),
		Weight:            1.5,
		Dimensions:        "10 x 5 x 3",
		Length:            10.0,
		Width:             5.0,
		Height:            3.0,
		Volume:            150.0,
		Barcode:           "1234567890123",
		TrackInventory:    true,
		StockQuantity:     100,
		MinStockLevel:     10,
		MaxStockLevel:     500,
		AllowBackorder:    false,
		RequiresShipping:  true,
		Taxable:           true,
		TaxRate:           decimal.NewFromFloat(10.0),
		IsActive:          true,
		IsFeatured:        false,
		IsDigital:         false,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
}

func createTestVariant(t testing.TB, productID uuid.UUID) *entities.ProductVariant {
	t.Helper()

	return &entities.ProductVariant{
		ID:                uuid.New(),
		ProductID:         productID,
		SKU:               "TEST-001-RED",
		Name:              "Test Product - Red",
		Price:             decimal.NewFromFloat(99.99),
		Cost:              decimal.NewFromFloat(50.00),
		Weight:            1.5,
		TrackInventory:    true,
		StockQuantity:     50,
		MinStockLevel:     5,
		MaxStockLevel:     200,
		AllowBackorder:    false,
		RequiresShipping:  true,
		Taxable:           true,
		TaxRate:           decimal.NewFromFloat(10.0),
		IsActive:          true,
		IsDigital:         false,
		SortOrder:         1,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
}

// Product Repository Tests
func TestPostgresProductRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(t)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	// Create test product
	product := createTestProductEntity(t, category.ID)

	err = repo.Create(ctx, product)
	assert.NoError(t, err)

	// Verify product was created
	retrieved, err := repo.GetByID(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, product.ID, retrieved.ID)
	assert.Equal(t, product.SKU, retrieved.SKU)
	assert.Equal(t, product.Name, retrieved.Name)
}

func TestPostgresProductRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(t)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	// Create test product
	product := createTestProductEntity(t, category.ID)
	err = repo.Create(ctx, product)
	require.NoError(t, err)

	// Get product by ID
	retrieved, err := repo.GetByID(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, product.ID, retrieved.ID)
	assert.Equal(t, product.SKU, retrieved.SKU)
	assert.Equal(t, product.Name, retrieved.Name)
}

func TestPostgresProductRepository_GetBySKU(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(t)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	// Create test product
	product := createTestProductEntity(t, category.ID)
	err = repo.Create(ctx, product)
	require.NoError(t, err)

	// Get product by SKU
	retrieved, err := repo.GetBySKU(ctx, product.SKU)
	require.NoError(t, err)
	assert.Equal(t, product.ID, retrieved.ID)
	assert.Equal(t, product.SKU, retrieved.SKU)
	assert.Equal(t, product.Name, retrieved.Name)
}

func TestPostgresProductRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(t)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	// Create test product
	product := createTestProductEntity(t, category.ID)
	err = repo.Create(ctx, product)
	require.NoError(t, err)

	// Update product
	product.Name = "Updated Product Name"
	product.Price = decimal.NewFromFloat(149.99)
	product.UpdatedAt = time.Now().UTC()

	err = repo.Update(ctx, product)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetByID(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Product Name", retrieved.Name)
	assert.True(t, product.Price.Equals(retrieved.Price))
}

func TestPostgresProductRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(t)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	// Create test product
	product := createTestProductEntity(t, category.ID)
	err = repo.Create(ctx, product)
	require.NoError(t, err)

	// Delete product
	err = repo.Delete(ctx, product.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = repo.GetByID(ctx, product.ID)
	assert.Error(t, err)
}

func TestPostgresProductRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(t)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	// Create test products
	products := make([]*entities.Product, 5)
	for i := 0; i < 5; i++ {
		product := createTestProductEntity(t, category.ID)
		product.SKU = fmt.Sprintf("TEST-%03d", i+1)
		product.Name = fmt.Sprintf("Test Product %d", i+1)
		product.Price = decimal.NewFromFloat(float64(i+1) * 10.0)
		products[i] = product
		err := repo.Create(ctx, product)
		require.NoError(t, err)
	}

	// Test list without filter
	filter := repositories.ProductFilter{
		Limit: 10,
	}
	result, err := repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, result, 5)

	// Test list with search filter
	filter.Search = "Product 1"
	result, err = repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, result, 1)

	// Test list with price filter
	filter = repositories.ProductFilter{
		MinPrice: decimalPtr(decimal.NewFromFloat(20.0)),
		MaxPrice: decimalPtr(decimal.NewFromFloat(40.0)),
	}
	result, err = repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, result, 3) // Products 2, 3, 4 (20, 30, 40)
}

func TestPostgresProductRepository_Count(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(t)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	// Create test products
	for i := 0; i < 5; i++ {
		product := createTestProductEntity(t, category.ID)
		product.SKU = fmt.Sprintf("TEST-%03d", i+1)
		product.Name = fmt.Sprintf("Test Product %d", i+1)
		product.IsActive = i < 3 // First 3 are active
		err := repo.Create(ctx, product)
		require.NoError(t, err)
	}

	// Test count without filter
	filter := repositories.ProductFilter{}
	count, err := repo.Count(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	// Test count with active filter
	filter.IsActive = boolPtr(true)
	count, err = repo.Count(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestPostgresProductRepository_Search(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(t)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	// Create test products
	products := []struct {
		sku        string
		name       string
		isActive   bool
	}{
		{"SEARCH-001", "Laptop Computer", true},
		{"SEARCH-002", "Desktop Computer", true},
		{"SEARCH-003", "Computer Mouse", false},
		{"SEARCH-004", "Computer Keyboard", true},
	}

	for _, p := range products {
		product := createTestProductEntity(t, category.ID)
		product.SKU = p.sku
		product.Name = p.name
		product.IsActive = p.isActive
		err := repo.Create(ctx, product)
		require.NoError(t, err)
	}

	// Search for "computer"
	results, err := repo.Search(ctx, "computer", 10)
	require.NoError(t, err)
	assert.Len(t, results, 3) // Should only return active products

	// Verify results
	for _, result := range results {
		assert.Contains(t, strings.ToLower(result.Name), "computer")
		assert.True(t, result.IsActive)
	}
}

func TestPostgresProductRepository_GetLowStock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(t)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	// Create test products with different stock levels
	testCases := []struct {
		stockQuantity int
		minStockLevel int
		shouldAppear  bool
	}{
		{50, 10, false},  // Normal stock
		{5, 10, true},   // Low stock
		{15, 20, true},  // Low stock
		{0, 10, false}, // Out of stock (not low stock)
	}

	for i, tc := range testCases {
		product := createTestProductEntity(t, category.ID)
		product.SKU = fmt.Sprintf("STOCK-%03d", i+1)
		product.Name = fmt.Sprintf("Stock Test %d", i+1)
		product.StockQuantity = tc.stockQuantity
		product.MinStockLevel = tc.minStockLevel
		product.TrackInventory = true
		err := repo.Create(ctx, product)
		require.NoError(t, err)
	}

	// Get low stock products
	results, err := repo.GetLowStock(ctx, 100)
	require.NoError(t, err)
	assert.Len(t, results, 2) // Should return 2 products with low stock

	// Verify results
	for _, result := range results {
		assert.True(t, result.TrackInventory)
		assert.True(t, result.StockQuantity > 0)
		assert.True(t, result.StockQuantity <= result.MinStockLevel)
	}
}

func TestPostgresProductRepository_ExistsBySKU(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(t)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	// Create test product
	product := createTestProductEntity(t, category.ID)
	err = repo.Create(ctx, product)
	require.NoError(t, err)

	// Test existing SKU
	exists, err := repo.ExistsBySKU(ctx, product.SKU)
	require.NoError(t, err)
	assert.True(t, exists)

	// Test non-existing SKU
	exists, err = repo.ExistsBySKU(ctx, "NON-EXISTENT")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPostgresProductRepository_UpdateStock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(t)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	// Create test product
	product := createTestProductEntity(t, category.ID)
	err = repo.Create(ctx, product)
	require.NoError(t, err)

	// Update stock
	newQuantity := 250
	err = repo.UpdateStock(ctx, product.ID, newQuantity)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetByID(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, newQuantity, retrieved.StockQuantity)
}

func TestPostgresProductRepository_GetProductStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(t)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	// Create test products with different configurations
	products := []struct {
		isActive      bool
		isFeatured    bool
		isDigital     bool
		trackInventory bool
		stockQuantity int
		minStockLevel int
		price         float64
		cost          float64
	}{
		{true, true, false, true, 100, 10, 99.99, 50.00},
		{true, false, true, false, 0, 0, 29.99, 10.00},
		{true, false, false, true, 5, 10, 49.99, 25.00}, // Low stock
		{false, false, false, true, 0, 10, 199.99, 100.00}, // Inactive, out of stock
		{true, true, false, true, 0, 10, 79.99, 40.00}, // Featured, out of stock
	}

	for _, p := range products {
		product := createTestProductEntity(t, category.ID)
		product.SKU = uuid.New().String()[:8]
		product.Name = "Stats Test Product"
		product.IsActive = p.isActive
		product.IsFeatured = p.isFeatured
		product.IsDigital = p.isDigital
		product.TrackInventory = p.trackInventory
		product.StockQuantity = p.stockQuantity
		product.MinStockLevel = p.minStockLevel
		product.Price = decimal.NewFromFloat(p.price)
		product.Cost = decimal.NewFromFloat(p.cost)
		err := repo.Create(ctx, product)
		require.NoError(t, err)
	}

	// Get stats
	stats, err := repo.GetProductStats(ctx, repositories.ProductFilter{})
	require.NoError(t, err)

	// Verify stats
	assert.Equal(t, 5, stats.TotalProducts)
	assert.Equal(t, 4, stats.ActiveProducts)
	assert.Equal(t, 1, stats.InactiveProducts)
	assert.Equal(t, 2, stats.FeaturedProducts)
	assert.Equal(t, 1, stats.LowStockProducts)
	assert.Equal(t, 1, stats.OutOfStockProducts)
	assert.Equal(t, 1, stats.DigitalProducts)
	assert.Equal(t, 4, stats.PhysicalProducts)
}

// Helper functions
func decimalPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}

func boolPtr(b bool) *bool {
	return &b
}