package repositories

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"erpgo/internal/domain/products/entities"
	"erpgo/internal/domain/products/repositories"
)

// Benchmark suite for repository performance testing
// These benchmarks help identify performance bottlenecks and ensure the repositories meet performance requirements

func BenchmarkProductRepository_Create(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(b)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	if err != nil {
		b.Fatalf("Failed to create test category: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		product := createTestProductEntity(b, category.ID)
		product.SKU = fmt.Sprintf("BENCH-%08d", i)
		product.Name = fmt.Sprintf("Benchmark Product %d", i)

		err := repo.Create(ctx, product)
		if err != nil {
			b.Fatalf("Failed to create product: %v", err)
		}
	}
}

func BenchmarkProductRepository_GetByID(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(b)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	if err != nil {
		b.Fatalf("Failed to create test category: %v", err)
	}

	// Create test products
	productIDs := make([]uuid.UUID, 100)
	for i := 0; i < 100; i++ {
		product := createTestProductEntity(b, category.ID)
		product.SKU = fmt.Sprintf("BENCH-%08d", i)
		product.Name = fmt.Sprintf("Benchmark Product %d", i)
		err := repo.Create(ctx, product)
		if err != nil {
			b.Fatalf("Failed to create product: %v", err)
		}
		productIDs[i] = product.ID
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		productID := productIDs[i%len(productIDs)]
		_, err := repo.GetByID(ctx, productID)
		if err != nil {
			b.Fatalf("Failed to get product: %v", err)
		}
	}
}

func BenchmarkProductRepository_List(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(b)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	if err != nil {
		b.Fatalf("Failed to create test category: %v", err)
	}

	// Create test products
	for i := 0; i < 1000; i++ {
		product := createTestProductEntity(b, category.ID)
		product.SKU = fmt.Sprintf("BENCH-%08d", i)
		product.Name = fmt.Sprintf("Benchmark Product %d", i)
		product.Price = decimal.NewFromFloat(float64(i%100) + 1.0)
		product.IsActive = i%10 != 0 // 90% active
		err := repo.Create(ctx, product)
		if err != nil {
			b.Fatalf("Failed to create product: %v", err)
		}
	}

	filter := repositories.ProductFilter{
		Limit:     50,
		Page:      1,
		SortBy:    "name",
		SortOrder: "ASC",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		filter.Page = (i % 20) + 1 // Cycle through pages
		_, err := repo.List(ctx, filter)
		if err != nil {
			b.Fatalf("Failed to list products: %v", err)
		}
	}
}

func BenchmarkProductRepository_Search(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(b)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	if err != nil {
		b.Fatalf("Failed to create test category: %v", err)
	}

	// Create test products with searchable names
	keywords := []string{"Laptop", "Computer", "Phone", "Tablet", "Monitor", "Keyboard", "Mouse"}
	for i := 0; i < 1000; i++ {
		product := createTestProductEntity(b, category.ID)
		product.SKU = fmt.Sprintf("BENCH-%08d", i)
		keyword := keywords[i%len(keywords)]
		product.Name = fmt.Sprintf("%s Model %d", keyword, i)
		product.Description = fmt.Sprintf("High quality %s with advanced features", keyword)
		product.IsActive = true
		err := repo.Create(ctx, product)
		if err != nil {
			b.Fatalf("Failed to create product: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		keyword := keywords[i%len(keywords)]
		query := fmt.Sprintf("%s Model", keyword)
		_, err := repo.Search(ctx, query, 20)
		if err != nil {
			b.Fatalf("Failed to search products: %v", err)
		}
	}
}

func BenchmarkProductRepository_Count(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(b)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	if err != nil {
		b.Fatalf("Failed to create test category: %v", err)
	}

	// Create test products
	for i := 0; i < 1000; i++ {
		product := createTestProductEntity(b, category.ID)
		product.SKU = fmt.Sprintf("BENCH-%08d", i)
		product.Name = fmt.Sprintf("Benchmark Product %d", i)
		product.IsActive = i%10 != 0 // 90% active
		err := repo.Create(ctx, product)
		if err != nil {
			b.Fatalf("Failed to create product: %v", err)
		}
	}

	filter := repositories.ProductFilter{
		IsActive: boolPtr(true),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := repo.Count(ctx, filter)
		if err != nil {
			b.Fatalf("Failed to count products: %v", err)
		}
	}
}

func BenchmarkCategoryRepository_GetDescendants(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create a deep category tree
	var parentID *uuid.UUID
	categoryIDs := make([]uuid.UUID, 0)

	for level := 0; level < 10; level++ {
		for i := 0; i < 5; i++ {
			category := createTestCategory(b)
			category.Name = fmt.Sprintf("Level %d Category %d", level, i)
			category.Path = fmt.Sprintf("/level-%d/category-%d", level, i)
			category.Level = level
			category.ParentID = parentID
			category.SortOrder = i

			err := repo.Create(ctx, category)
			if err != nil {
				b.Fatalf("Failed to create category: %v", err)
			}

			if level == 0 && i == 0 {
				categoryIDs = append(categoryIDs, category.ID)
				parentID = &category.ID
			}
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		categoryID := categoryIDs[i%len(categoryIDs)]
		_, err := repo.GetDescendants(ctx, categoryID)
		if err != nil {
			b.Fatalf("Failed to get descendants: %v", err)
		}
	}
}

func BenchmarkCategoryRepository_GetPath(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create a deep category tree path
	var parentID *uuid.UUID
	leafCategoryIDs := make([]uuid.UUID, 0)

	for level := 0; level < 8; level++ {
		for i := 0; i < 3; i++ {
			category := createTestCategory(b)
			category.Name = fmt.Sprintf("Level %d Category %d", level, i)
			category.Path = fmt.Sprintf("/level-%d/category-%d", level, i)
			category.Level = level
			category.ParentID = parentID
			category.SortOrder = i

			err := repo.Create(ctx, category)
			if err != nil {
				b.Fatalf("Failed to create category: %v", err)
			}

			if level == 7 {
				leafCategoryIDs = append(leafCategoryIDs, category.ID)
			}

			if i == 0 {
				parentID = &category.ID
			}
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		categoryID := leafCategoryIDs[i%len(leafCategoryIDs)]
		_, err := repo.GetPath(ctx, categoryID)
		if err != nil {
			b.Fatalf("Failed to get path: %v", err)
		}
	}
}

func BenchmarkProductVariantRepository_Create(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := NewPostgresProductVariantRepository(db)
	ctx := context.Background()

	// Create test product first
	category := createTestCategory(b)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	if err != nil {
		b.Fatalf("Failed to create test category: %v", err)
	}

	product := createTestProductEntity(b, category.ID)
	productRepo := NewPostgresProductRepository(db)
	err = productRepo.Create(ctx, product)
	if err != nil {
		b.Fatalf("Failed to create test product: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		variant := createTestVariant(b, product.ID)
		variant.SKU = fmt.Sprintf("BENCH-VAR-%08d", i)
		variant.Name = fmt.Sprintf("Benchmark Variant %d", i)

		err := repo.Create(ctx, variant)
		if err != nil {
			b.Fatalf("Failed to create variant: %v", err)
		}
	}
}

func BenchmarkProductVariantRepository_GetByProductID(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := NewPostgresProductVariantRepository(db)
	ctx := context.Background()

	// Create test product first
	category := createTestCategory(b)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	if err != nil {
		b.Fatalf("Failed to create test category: %v", err)
	}

	// Create multiple test products
	productIDs := make([]uuid.UUID, 10)
	for p := 0; p < 10; p++ {
		product := createTestProductEntity(b, category.ID)
		product.SKU = fmt.Sprintf("BENCH-PROD-%08d", p)
		product.Name = fmt.Sprintf("Benchmark Product %d", p)
		productRepo := NewPostgresProductRepository(db)
		err = productRepo.Create(ctx, product)
		if err != nil {
			b.Fatalf("Failed to create test product: %v", err)
		}
		productIDs[p] = product.ID

		// Create variants for each product
		for v := 0; v < 20; v++ {
			variant := createTestVariant(b, product.ID)
			variant.SKU = fmt.Sprintf("BENCH-VAR-%08d", p*20+v)
			variant.Name = fmt.Sprintf("Variant %d", v)
			variant.SortOrder = v
			err := repo.Create(ctx, variant)
			if err != nil {
				b.Fatalf("Failed to create variant: %v", err)
			}
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		productID := productIDs[i%len(productIDs)]
		_, err := repo.GetByProductID(ctx, productID)
		if err != nil {
			b.Fatalf("Failed to get variants by product ID: %v", err)
		}
	}
}

func BenchmarkProductVariantRepository_List(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := NewPostgresProductVariantRepository(db)
	ctx := context.Background()

	// Create test product first
	category := createTestCategory(b)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	if err != nil {
		b.Fatalf("Failed to create test category: %v", err)
	}

	// Create test products
	for p := 0; p < 10; p++ {
		product := createTestProductEntity(b, category.ID)
		product.SKU = fmt.Sprintf("BENCH-PROD-%08d", p)
		product.Name = fmt.Sprintf("Benchmark Product %d", p)
		productRepo := NewPostgresProductRepository(db)
		err = productRepo.Create(ctx, product)
		if err != nil {
			b.Fatalf("Failed to create test product: %v", err)
		}

		// Create variants for each product
		for v := 0; v < 50; v++ {
			variant := createTestVariant(b, product.ID)
			variant.SKU = fmt.Sprintf("BENCH-VAR-%08d", p*50+v)
			variant.Name = fmt.Sprintf("Variant %d", v)
			variant.Price = decimal.NewFromFloat(float64(v%20) + 10.0)
			variant.IsActive = v%5 != 0 // 80% active
			variant.SortOrder = v
			err := repo.Create(ctx, variant)
			if err != nil {
				b.Fatalf("Failed to create variant: %v", err)
			}
		}
	}

	filter := repositories.ProductVariantFilter{
		Limit:     100,
		Page:      1,
		SortBy:    "sort_order",
		SortOrder: "ASC",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		filter.Page = (i % 5) + 1 // Cycle through pages
		_, err := repo.List(ctx, filter)
		if err != nil {
			b.Fatalf("Failed to list variants: %v", err)
		}
	}
}

// Benchmark for concurrent operations
func BenchmarkProductRepository_ConcurrentReads(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(b)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	if err != nil {
		b.Fatalf("Failed to create test category: %v", err)
	}

	// Create test products
	productIDs := make([]uuid.UUID, 100)
	for i := 0; i < 100; i++ {
		product := createTestProductEntity(b, category.ID)
		product.SKU = fmt.Sprintf("BENCH-%08d", i)
		product.Name = fmt.Sprintf("Benchmark Product %d", i)
		err := repo.Create(ctx, product)
		if err != nil {
			b.Fatalf("Failed to create product: %v", err)
		}
		productIDs[i] = product.ID
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for _, productID := range productIDs {
				_, err := repo.GetByID(ctx, productID)
				if err != nil {
					b.Fatalf("Failed to get product: %v", err)
				}
			}
		}
	})
}

// Benchmark for bulk operations
func BenchmarkProductVariantRepository_BulkCreate(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := NewPostgresProductVariantRepository(db)
	ctx := context.Background()

	// Create test product first
	category := createTestCategory(b)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	if err != nil {
		b.Fatalf("Failed to create test category: %v", err)
	}

	product := createTestProductEntity(b, category.ID)
	productRepo := NewPostgresProductRepository(db)
	err = productRepo.Create(ctx, product)
	if err != nil {
		b.Fatalf("Failed to create test product: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		variants := make([]*entities.ProductVariant, 50)
		for j := 0; j < 50; j++ {
			variants[j] = createTestVariant(b, product.ID)
			variants[j].SKU = fmt.Sprintf("BULK-BENCH-%08d-%03d", i, j)
			variants[j].Name = fmt.Sprintf("Bulk Variant %d-%d", i, j)
			variants[j].SortOrder = j
		}

		err := repo.BulkCreate(ctx, variants)
		if err != nil {
			b.Fatalf("Failed to bulk create variants: %v", err)
		}
	}
}

// Memory and allocation benchmarks
func BenchmarkProductRepository_List_Memory(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := NewPostgresProductRepository(db)
	ctx := context.Background()

	// Create test category first
	category := createTestCategory(b)
	categoryRepo := NewPostgresCategoryRepository(db)
	err := categoryRepo.Create(ctx, category)
	if err != nil {
		b.Fatalf("Failed to create test category: %v", err)
	}

	// Create test products
	for i := 0; i < 100; i++ {
		product := createTestProductEntity(b, category.ID)
		product.SKU = fmt.Sprintf("BENCH-%08d", i)
		product.Name = fmt.Sprintf("Benchmark Product %d", i)
		err := repo.Create(ctx, product)
		if err != nil {
			b.Fatalf("Failed to create product: %v", err)
		}
	}

	filter := repositories.ProductFilter{
		Limit:     50,
		SortBy:    "name",
		SortOrder: "ASC",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		products, err := repo.List(ctx, filter)
		if err != nil {
			b.Fatalf("Failed to list products: %v", err)
		}
		// Use the products to prevent optimization
		_ = len(products)
	}
}