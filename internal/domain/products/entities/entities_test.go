package entities

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductCategory_Validate(t *testing.T) {
	tests := []struct {
		name        string
		category    *ProductCategory
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid category",
			category: &ProductCategory{
				ID:          uuid.New(),
				Name:        "Electronics",
				Description: "Electronic devices and accessories",
				Level:       0,
				Path:        "/electronics",
				ImageURL:    "https://example.com/image.jpg",
				SortOrder:   1,
				IsActive:    true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			expectError: false,
		},
		{
			name: "missing ID",
			category: &ProductCategory{
				Name:      "Electronics",
				Level:     0,
				Path:      "/electronics",
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectError: true,
			errorMsg:    "category ID cannot be empty",
		},
		{
			name: "empty name",
			category: &ProductCategory{
				ID:        uuid.New(),
				Name:      "",
				Level:     0,
				Path:      "/electronics",
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectError: true,
			errorMsg:    "category name cannot be empty",
		},
		{
			name: "name too long",
			category: &ProductCategory{
				ID:        uuid.New(),
				Name:      string(make([]byte, 201)), // 201 characters
				Level:     0,
				Path:      "/electronics",
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectError: true,
			errorMsg:    "category name cannot exceed 200 characters",
		},
		{
			name: "invalid name characters",
			category: &ProductCategory{
				ID:        uuid.New(),
				Name:      "Electronics@#$",
				Level:     0,
				Path:      "/electronics",
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectError: true,
			errorMsg:    "category name can only contain letters, numbers, spaces, hyphens, underscores, and forward slashes",
		},
		{
			name: "negative level",
			category: &ProductCategory{
				ID:        uuid.New(),
				Name:      "Electronics",
				Level:     -1,
				Path:      "/electronics",
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectError: true,
			errorMsg:    "category level cannot be negative",
		},
		{
			name: "invalid path format",
			category: &ProductCategory{
				ID:        uuid.New(),
				Name:      "Electronics",
				Level:     0,
				Path:      "electronics", // missing leading slash
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectError: true,
			errorMsg:    "path must start with forward slash",
		},
		{
			name: "invalid image URL",
			category: &ProductCategory{
				ID:        uuid.New(),
				Name:      "Electronics",
				Level:     0,
				Path:      "/electronics",
				ImageURL:  "not-a-url",
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectError: true,
			errorMsg:    "invalid image URL format",
		},
		{
			name: "negative sort order",
			category: &ProductCategory{
				ID:        uuid.New(),
				Name:      "Electronics",
				Level:     0,
				Path:      "/electronics",
				SortOrder: -1,
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectError: true,
			errorMsg:    "sort order cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.category.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProductCategory_BuildPath(t *testing.T) {
	tests := []struct {
		name          string
		category      *ProductCategory
		parentPath    string
		expectedPath  string
		expectedLevel int
	}{
		{
			name: "root category",
			category: &ProductCategory{
				Name: "Electronics",
			},
			parentPath:    "",
			expectedPath:  "/electronics",
			expectedLevel: 0,
		},
		{
			name: "child category",
			category: &ProductCategory{
				Name:     "Computers",
				ParentID: &uuid.UUID{}, // Non-nil parent ID
			},
			parentPath:    "/electronics",
			expectedPath:  "/electronics/computers",
			expectedLevel: 1,
		},
		{
			name: "grandchild category",
			category: &ProductCategory{
				Name:     "Laptops",
				ParentID: &uuid.UUID{}, // Non-nil parent ID
			},
			parentPath:    "/electronics/computers",
			expectedPath:  "/electronics/computers/laptops",
			expectedLevel: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.category.BuildPath(tt.parentPath)

			assert.Equal(t, tt.expectedPath, tt.category.Path)
			assert.Equal(t, tt.expectedLevel, tt.category.Level)
		})
	}
}

func TestProduct_Validate(t *testing.T) {
	price := decimal.NewFromFloat(99.99)
	cost := decimal.NewFromFloat(50.00)
	taxRate := decimal.NewFromFloat(8.25)

	tests := []struct {
		name        string
		product     *Product
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid physical product",
			product: &Product{
				ID:               uuid.New(),
				SKU:              "TEST-001",
				Name:             "Test Product",
				Description:      "This is a test product",
				ShortDescription: "Test product short description",
				CategoryID:       uuid.New(),
				Price:            price,
				Cost:             cost,
				Weight:           1.5,
				Dimensions:       "10 x 5 x 3",
				TrackInventory:   true,
				StockQuantity:    100,
				MinStockLevel:    10,
				RequiresShipping: true,
				Taxable:          true,
				TaxRate:          taxRate,
				IsActive:         true,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
			expectError: false,
		},
		{
			name: "valid digital product",
			product: &Product{
				ID:               uuid.New(),
				SKU:              "DIGITAL-001",
				Name:             "Digital Product",
				Description:      "This is a digital product",
				CategoryID:       uuid.New(),
				Price:            price,
				Cost:             decimal.Zero,
				TrackInventory:   false,
				RequiresShipping: false,
				Taxable:          false,
				IsDigital:        true,
				DownloadURL:      "https://example.com/download/file.pdf",
				MaxDownloads:     5,
				ExpiryDays:       30,
				IsActive:         true,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
			expectError: false,
		},
		{
			name: "missing ID",
			product: &Product{
				SKU:        "TEST-001",
				Name:       "Test Product",
				CategoryID: uuid.New(),
				Price:      price,
				IsActive:   true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			expectError: true,
			errorMsg:    "product ID cannot be empty",
		},
		{
			name: "empty SKU",
			product: &Product{
				ID:         uuid.New(),
				SKU:        "",
				Name:       "Test Product",
				CategoryID: uuid.New(),
				Price:      price,
				IsActive:   true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			expectError: true,
			errorMsg:    "SKU cannot be empty",
		},
		{
			name: "empty name",
			product: &Product{
				ID:         uuid.New(),
				SKU:        "TEST-001",
				Name:       "",
				CategoryID: uuid.New(),
				Price:      price,
				IsActive:   true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			expectError: true,
			errorMsg:    "product name cannot be empty",
		},
		{
			name: "zero price",
			product: &Product{
				ID:         uuid.New(),
				SKU:        "TEST-001",
				Name:       "Test Product",
				CategoryID: uuid.New(),
				Price:      decimal.Zero,
				IsActive:   true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			expectError: true,
			errorMsg:    "price must be greater than 0",
		},
		{
			name: "cost higher than price",
			product: &Product{
				ID:         uuid.New(),
				SKU:        "TEST-001",
				Name:       "Test Product",
				CategoryID: uuid.New(),
				Price:      decimal.NewFromFloat(50.00),
				Cost:       decimal.NewFromFloat(75.00),
				IsActive:   true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			expectError: true,
			errorMsg:    "cost cannot be higher than price",
		},
		{
			name: "negative weight",
			product: &Product{
				ID:         uuid.New(),
				SKU:        "TEST-001",
				Name:       "Test Product",
				CategoryID: uuid.New(),
				Price:      price,
				Weight:     -1.0,
				IsActive:   true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			expectError: true,
			errorMsg:    "weight cannot be negative",
		},
		{
			name: "invalid dimensions format",
			product: &Product{
				ID:         uuid.New(),
				SKU:        "TEST-001",
				Name:       "Test Product",
				CategoryID: uuid.New(),
				Price:      price,
				Dimensions: "invalid dimensions",
				IsActive:   true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			expectError: true,
			errorMsg:    "dimensions must be in format 'L x W x H' or 'LxWxH'",
		},
		{
			name: "negative stock quantity",
			product: &Product{
				ID:             uuid.New(),
				SKU:            "TEST-001",
				Name:           "Test Product",
				CategoryID:     uuid.New(),
				Price:          price,
				TrackInventory: true,
				StockQuantity:  -10,
				IsActive:       true,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
			expectError: true,
			errorMsg:    "stock quantity cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.product.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProduct_BusinessLogic(t *testing.T) {
	product := &Product{
		ID:             uuid.New(),
		SKU:            "TEST-001",
		Name:           "Test Product",
		CategoryID:     uuid.New(),
		Price:          decimal.NewFromFloat(100.00),
		Cost:           decimal.NewFromFloat(60.00),
		TrackInventory: true,
		StockQuantity:  50,
		MinStockLevel:  10,
		AllowBackorder: false,
		Taxable:        true,
		TaxRate:        decimal.NewFromFloat(8.25),
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	t.Run("IsInStock with positive quantity", func(t *testing.T) {
		assert.True(t, product.IsInStock())
	})

	t.Run("IsInStock with zero quantity and no backorder", func(t *testing.T) {
		product.StockQuantity = 0
		assert.False(t, product.IsInStock())
	})

	t.Run("IsInStock with zero quantity and backorder allowed", func(t *testing.T) {
		product.AllowBackorder = true
		assert.True(t, product.IsInStock())
	})

	t.Run("IsLowStock", func(t *testing.T) {
		product.StockQuantity = 5
		product.MinStockLevel = 10
		assert.True(t, product.IsLowStock())

		product.StockQuantity = 15
		assert.False(t, product.IsLowStock())
	})

	t.Run("CanFulfillOrder", func(t *testing.T) {
		product.StockQuantity = 50
		product.AllowBackorder = false

		// Can fulfill
		err := product.CanFulfillOrder(10)
		assert.NoError(t, err)

		// Cannot fulfill - insufficient stock
		err = product.CanFulfillOrder(60)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient stock")

		// Cannot fulfill - inactive product
		product.IsActive = false
		err = product.CanFulfillOrder(10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "product is not active")
	})

	t.Run("CalculateProfit", func(t *testing.T) {
		product.IsActive = true
		profit := product.CalculateProfit()
		expectedProfit := decimal.NewFromFloat(40.00) // 100 - 60
		assert.True(t, profit.Equal(expectedProfit))
	})

	t.Run("CalculateProfitMargin", func(t *testing.T) {
		margin := product.CalculateProfitMargin()
		expectedMargin := decimal.NewFromFloat(40.00) // 40% margin
		assert.True(t, margin.Equal(expectedMargin))
	})

	t.Run("CalculateTax", func(t *testing.T) {
		tax := product.CalculateTax()
		expectedTax := decimal.NewFromFloat(8.25) // 100 * 8.25%
		assert.True(t, tax.Equal(expectedTax))
	})

	t.Run("CalculateTotalPrice", func(t *testing.T) {
		totalPrice := product.CalculateTotalPrice()
		expectedTotal := decimal.NewFromFloat(108.25) // 100 + 8.25
		assert.True(t, totalPrice.Equal(expectedTotal))
	})

	t.Run("UpdateStock", func(t *testing.T) {
		err := product.UpdateStock(100)
		assert.NoError(t, err)
		assert.Equal(t, 100, product.StockQuantity)

		// Test negative quantity
		err = product.UpdateStock(-10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stock quantity cannot be negative")
	})

	t.Run("UpdatePrice", func(t *testing.T) {
		newPrice := decimal.NewFromFloat(120.00)
		err := product.UpdatePrice(newPrice)
		assert.NoError(t, err)
		assert.True(t, product.Price.Equal(newPrice))

		// Test zero price
		err = product.UpdatePrice(decimal.Zero)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "price must be greater than 0")
	})

	t.Run("Activate and Deactivate", func(t *testing.T) {
		product.Deactivate()
		assert.False(t, product.IsActiveProduct())

		product.Activate()
		assert.True(t, product.IsActiveProduct())
	})

	t.Run("SetFeatured", func(t *testing.T) {
		product.SetFeatured(true)
		assert.True(t, product.IsFeatured)

		product.SetFeatured(false)
		assert.False(t, product.IsFeatured)
	})
}

func TestProductVariant_Validate(t *testing.T) {
	price := decimal.NewFromFloat(99.99)
	cost := decimal.NewFromFloat(50.00)

	tests := []struct {
		name        string
		variant     *ProductVariant
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid variant",
			variant: &ProductVariant{
				ID:             uuid.New(),
				ProductID:      uuid.New(),
				SKU:            "VARIANT-001",
				Name:           "Large Red Shirt",
				Price:          price,
				Cost:           cost,
				Weight:         0.5,
				TrackInventory: true,
				StockQuantity:  50,
				IsActive:       true,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
			expectError: false,
		},
		{
			name: "missing product ID",
			variant: &ProductVariant{
				ID:        uuid.New(),
				SKU:       "VARIANT-001",
				Name:      "Large Red Shirt",
				Price:     price,
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectError: true,
			errorMsg:    "product ID cannot be empty",
		},
		{
			name: "empty SKU",
			variant: &ProductVariant{
				ID:        uuid.New(),
				ProductID: uuid.New(),
				Name:      "Large Red Shirt",
				Price:     price,
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectError: true,
			errorMsg:    "variant SKU cannot be empty",
		},
		{
			name: "digital variant without download URL",
			variant: &ProductVariant{
				ID:        uuid.New(),
				ProductID: uuid.New(),
				SKU:       "DIGITAL-VARIANT-001",
				Name:      "Digital Variant",
				Price:     price,
				IsDigital: true,
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectError: true,
			errorMsg:    "digital variants must have a download URL",
		},
		{
			name: "digital variant with shipping requirement",
			variant: &ProductVariant{
				ID:               uuid.New(),
				ProductID:        uuid.New(),
				SKU:              "DIGITAL-VARIANT-001",
				Name:             "Digital Variant",
				Price:            price,
				IsDigital:        true,
				DownloadURL:      "https://example.com/download",
				RequiresShipping: true,
				IsActive:         true,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
			expectError: true,
			errorMsg:    "digital variants cannot require shipping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.variant.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProductVariant_BusinessLogic(t *testing.T) {
	variant := &ProductVariant{
		ID:             uuid.New(),
		ProductID:      uuid.New(),
		SKU:            "VARIANT-001",
		Name:           "Large Red Shirt",
		Price:          decimal.NewFromFloat(50.00),
		Cost:           decimal.NewFromFloat(30.00),
		TrackInventory: true,
		StockQuantity:  25,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	t.Run("CalculateProfit", func(t *testing.T) {
		profit := variant.CalculateProfit()
		expectedProfit := decimal.NewFromFloat(20.00) // 50 - 30
		assert.True(t, profit.Equal(expectedProfit))
	})

	t.Run("CalculateProfitMargin", func(t *testing.T) {
		margin := variant.CalculateProfitMargin()
		expectedMargin := decimal.NewFromFloat(40.00) // 40% margin
		assert.True(t, margin.Equal(expectedMargin))
	})

	t.Run("UpdateSortOrder", func(t *testing.T) {
		err := variant.UpdateSortOrder(5)
		assert.NoError(t, err)
		assert.Equal(t, 5, variant.SortOrder)

		// Test negative sort order
		err = variant.UpdateSortOrder(-1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "variant sort order cannot be negative")
	})
}

func TestVariantAttribute_Validate(t *testing.T) {
	tests := []struct {
		name        string
		attribute   *VariantAttribute
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid attribute",
			attribute: &VariantAttribute{
				ID:        uuid.New(),
				VariantID: uuid.New(),
				Name:      "Color",
				Value:     "Red",
				Type:      "color",
				SortOrder: 1,
			},
			expectError: false,
		},
		{
			name: "empty name",
			attribute: &VariantAttribute{
				ID:        uuid.New(),
				VariantID: uuid.New(),
				Name:      "",
				Value:     "Red",
				Type:      "color",
				SortOrder: 1,
			},
			expectError: true,
			errorMsg:    "attribute name cannot be empty",
		},
		{
			name: "empty value",
			attribute: &VariantAttribute{
				ID:        uuid.New(),
				VariantID: uuid.New(),
				Name:      "Color",
				Value:     "",
				Type:      "color",
				SortOrder: 1,
			},
			expectError: true,
			errorMsg:    "attribute value cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.attribute.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVariantImage_Validate(t *testing.T) {
	tests := []struct {
		name        string
		image       *VariantImage
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid image",
			image: &VariantImage{
				ID:        uuid.New(),
				VariantID: uuid.New(),
				ImageURL:  "https://example.com/image.jpg",
				AltText:   "Product image",
				SortOrder: 1,
				IsMain:    false,
			},
			expectError: false,
		},
		{
			name: "empty image URL",
			image: &VariantImage{
				ID:        uuid.New(),
				VariantID: uuid.New(),
				ImageURL:  "",
				AltText:   "Product image",
				SortOrder: 1,
				IsMain:    false,
			},
			expectError: true,
			errorMsg:    "image URL cannot be empty",
		},
		{
			name: "invalid image URL format",
			image: &VariantImage{
				ID:        uuid.New(),
				VariantID: uuid.New(),
				ImageURL:  "not-a-url",
				AltText:   "Product image",
				SortOrder: 1,
				IsMain:    false,
			},
			expectError: true,
			errorMsg:    "invalid image URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.image.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVariantImage_SetAsMain(t *testing.T) {
	image := &VariantImage{
		ID:        uuid.New(),
		VariantID: uuid.New(),
		ImageURL:  "https://example.com/image.jpg",
		IsMain:    false,
	}

	image.SetAsMain()
	assert.True(t, image.IsMain)
}

func TestCategoryMetadata_Validate(t *testing.T) {
	tests := []struct {
		name        string
		metadata    *CategoryMetadata
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid metadata",
			metadata: &CategoryMetadata{
				CategoryID:     uuid.New(),
				SeoTitle:       "Best Electronics",
				SeoDescription: "Find the best electronics online",
				SeoKeywords:    "electronics, gadgets, technology",
			},
			expectError: false,
		},
		{
			name: "missing category ID",
			metadata: &CategoryMetadata{
				SeoTitle:       "Best Electronics",
				SeoDescription: "Find the best electronics online",
				SeoKeywords:    "electronics, gadgets, technology",
			},
			expectError: true,
			errorMsg:    "category ID cannot be empty",
		},
		{
			name: "SEO title too long",
			metadata: &CategoryMetadata{
				CategoryID: uuid.New(),
				SeoTitle:   string(make([]byte, 201)), // 201 characters
			},
			expectError: true,
			errorMsg:    "SEO title cannot exceed 200 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Benchmark tests
func BenchmarkProduct_Validate(b *testing.B) {
	product := &Product{
		ID:               uuid.New(),
		SKU:              "BENCHMARK-001",
		Name:             "Benchmark Product",
		Description:      "This is a benchmark test product",
		CategoryID:       uuid.New(),
		Price:            decimal.NewFromFloat(99.99),
		Cost:             decimal.NewFromFloat(50.00),
		Weight:           1.0,
		TrackInventory:   true,
		StockQuantity:    100,
		RequiresShipping: true,
		Taxable:          true,
		TaxRate:          decimal.NewFromFloat(8.25),
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = product.Validate()
	}
}

func BenchmarkProductVariant_Validate(b *testing.B) {
	variant := &ProductVariant{
		ID:             uuid.New(),
		ProductID:      uuid.New(),
		SKU:            "VARIANT-BENCHMARK-001",
		Name:           "Benchmark Variant",
		Price:          decimal.NewFromFloat(99.99),
		Cost:           decimal.NewFromFloat(50.00),
		TrackInventory: true,
		StockQuantity:  50,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = variant.Validate()
	}
}

func BenchmarkProduct_CalculateTotalPrice(b *testing.B) {
	product := &Product{
		Price:   decimal.NewFromFloat(100.00),
		Taxable: true,
		TaxRate: decimal.NewFromFloat(8.25),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = product.CalculateTotalPrice()
	}
}

// Integration test to test entity interactions
func TestProduct_WithVariants_Integration(t *testing.T) {
	// Create a parent product
	product := &Product{
		ID:               uuid.New(),
		SKU:              "PARENT-001",
		Name:             "Parent Product",
		Description:      "This is a parent product with variants",
		CategoryID:       uuid.New(),
		Price:            decimal.NewFromFloat(100.00),
		Cost:             decimal.NewFromFloat(60.00),
		TrackInventory:   false, // Track inventory at variant level
		RequiresShipping: true,
		Taxable:          true,
		TaxRate:          decimal.NewFromFloat(8.25),
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Validate parent product
	require.NoError(t, product.Validate())

	// Create variants
	attributes := []VariantAttribute{
		{
			ID:        uuid.New(),
			VariantID: uuid.New(), // Will be set below
			Name:      "Size",
			Value:     "Large",
			Type:      "size",
			SortOrder: 1,
		},
		{
			ID:        uuid.New(),
			VariantID: uuid.New(), // Will be set below
			Name:      "Color",
			Value:     "Red",
			Type:      "color",
			SortOrder: 2,
		},
	}

	images := []VariantImage{
		{
			ID:        uuid.New(),
			VariantID: uuid.New(), // Will be set below
			ImageURL:  "https://example.com/variant-image.jpg",
			AltText:   "Product variant image",
			SortOrder: 1,
			IsMain:    true,
		},
	}

	// Test attribute validation
	for _, attr := range attributes {
		require.NoError(t, attr.Validate())
	}

	// Test image validation
	for _, img := range images {
		require.NoError(t, img.Validate())
	}

	// Create a variant
	variant := &ProductVariant{
		ID:             uuid.New(),
		ProductID:      product.ID,
		SKU:            "VARIANT-LARGE-RED",
		Name:           "Large Red Variant",
		Price:          decimal.NewFromFloat(110.00),
		Cost:           decimal.NewFromFloat(65.00),
		Weight:         1.2,
		TrackInventory: true,
		StockQuantity:  25,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Validate variant
	require.NoError(t, variant.Validate())

	// Set relationships
	for i := range attributes {
		attributes[i].VariantID = variant.ID
	}

	for i := range images {
		images[i].VariantID = variant.ID
	}

	// Test variant business logic
	assert.True(t, variant.IsInStock())
	assert.False(t, variant.IsLowStock())

	profit := variant.CalculateProfit()
	expectedProfit := decimal.NewFromFloat(45.00) // 110 - 65
	assert.True(t, profit.Equal(expectedProfit))

	// Test ability to fulfill order
	err := variant.CanFulfillOrder(10)
	assert.NoError(t, err)

	err = variant.CanFulfillOrder(30)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient variant stock")
}
