package entities

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// TestProduct_ComprehensiveCoverage tests all product methods for complete coverage
func TestProduct_ComprehensiveCoverage(t *testing.T) {
	t.Run("AdjustStock", func(t *testing.T) {
		product := &Product{
			ID:             uuid.New(),
			SKU:            "TEST-001",
			Name:           "Test Product",
			CategoryID:     uuid.New(),
			Price:          decimal.NewFromFloat(100.00),
			Cost:           decimal.NewFromFloat(60.00),
			TrackInventory: true,
			StockQuantity:  50,
			IsActive:       true,
		}

		// Positive adjustment
		err := product.AdjustStock(25)
		assert.NoError(t, err)
		assert.Equal(t, 75, product.StockQuantity)

		// Negative adjustment
		err = product.AdjustStock(-10)
		assert.NoError(t, err)
		assert.Equal(t, 65, product.StockQuantity)

		// Adjustment resulting in negative
		err = product.AdjustStock(-100)
		assert.Error(t, err)
	})

	t.Run("UpdateCost", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.NewFromFloat(100.00),
			Cost:       decimal.NewFromFloat(60.00),
			IsActive:   true,
		}

		// Valid update
		err := product.UpdateCost(decimal.NewFromFloat(70.00))
		assert.NoError(t, err)
		assert.True(t, product.Cost.Equal(decimal.NewFromFloat(70.00)))

		// Negative cost
		err = product.UpdateCost(decimal.NewFromFloat(-10.00))
		assert.Error(t, err)

		// Cost exceeding price
		err = product.UpdateCost(decimal.NewFromFloat(150.00))
		assert.Error(t, err)

		// Cost exceeding max
		err = product.UpdateCost(decimal.NewFromFloat(1000000.00))
		assert.Error(t, err)
	})

	t.Run("UpdateCategory", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.NewFromFloat(100.00),
			IsActive:   true,
		}

		newCategoryID := uuid.New()
		err := product.UpdateCategory(newCategoryID)
		assert.NoError(t, err)
		assert.Equal(t, newCategoryID, product.CategoryID)

		// Empty category ID
		err = product.UpdateCategory(uuid.Nil)
		assert.Error(t, err)
	})

	t.Run("UpdateDetails", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.NewFromFloat(100.00),
			IsActive:   true,
		}

		// Valid update
		err := product.UpdateDetails("New Name", "New Description", "New Short Description")
		assert.NoError(t, err)
		assert.Equal(t, "New Name", product.Name)
		assert.Equal(t, "New Description", product.Description)
		assert.Equal(t, "New Short Description", product.ShortDescription)

		// Empty name
		err = product.UpdateDetails("", "Description", "Short")
		assert.Error(t, err)

		// Name too long
		longName := string(make([]byte, 301))
		err = product.UpdateDetails(longName, "Description", "Short")
		assert.Error(t, err)

		// Description too long
		longDesc := string(make([]byte, 2001))
		err = product.UpdateDetails("Name", longDesc, "Short")
		assert.Error(t, err)

		// Short description too long
		longShort := string(make([]byte, 501))
		err = product.UpdateDetails("Name", "Description", longShort)
		assert.Error(t, err)
	})

	t.Run("ToSafeProduct", func(t *testing.T) {
		product := &Product{
			ID:               uuid.New(),
			SKU:              "TEST-001",
			Name:             "Test Product",
			CategoryID:       uuid.New(),
			Price:            decimal.NewFromFloat(100.00),
			Cost:             decimal.NewFromFloat(60.00),
			TrackInventory:   true,
			StockQuantity:    50,
			IsActive:         true,
		}

		safe := product.ToSafeProduct()
		assert.Equal(t, product.ID, safe.ID)
		assert.Equal(t, product.SKU, safe.SKU)
		assert.True(t, safe.Cost.IsZero()) // Cost should be removed
	})
}

// TestProductCategory_ComprehensiveCoverage tests all category methods
func TestProductCategory_ComprehensiveCoverage(t *testing.T) {
	t.Run("IsRootCategory", func(t *testing.T) {
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Electronics",
			Level:    0,
			Path:     "/electronics",
			IsActive: true,
		}

		assert.True(t, category.IsRootCategory())

		parentID := uuid.New()
		category.ParentID = &parentID
		assert.False(t, category.IsRootCategory())
	})

	t.Run("IsChildOf", func(t *testing.T) {
		parentID := uuid.New()
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Laptops",
			ParentID: &parentID,
			Level:    1,
			Path:     "/electronics/laptops",
			IsActive: true,
		}

		assert.True(t, category.IsChildOf(parentID))
		assert.False(t, category.IsChildOf(uuid.New()))
	})

	t.Run("GetDepth", func(t *testing.T) {
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Laptops",
			Level:    2,
			Path:     "/electronics/computers/laptops",
			IsActive: true,
		}

		assert.Equal(t, 2, category.GetDepth())
	})

	t.Run("CanHaveChildren", func(t *testing.T) {
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Electronics",
			Level:    0,
			Path:     "/electronics",
			IsActive: true,
		}

		assert.True(t, category.CanHaveChildren())

		category.Level = 5
		assert.False(t, category.CanHaveChildren())
	})

	t.Run("GetPathSegments", func(t *testing.T) {
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Laptops",
			Level:    2,
			Path:     "/electronics/computers/laptops",
			IsActive: true,
		}

		segments := category.GetPathSegments()
		assert.Equal(t, 3, len(segments))
		assert.Equal(t, "electronics", segments[0])
		assert.Equal(t, "computers", segments[1])
		assert.Equal(t, "laptops", segments[2])

		// Empty path
		category.Path = ""
		segments = category.GetPathSegments()
		assert.Equal(t, 0, len(segments))
	})

	t.Run("UpdateSortOrder", func(t *testing.T) {
		category := &ProductCategory{
			ID:        uuid.New(),
			Name:      "Electronics",
			Level:     0,
			Path:      "/electronics",
			SortOrder: 1,
			IsActive:  true,
		}

		err := category.UpdateSortOrder(5)
		assert.NoError(t, err)
		assert.Equal(t, 5, category.SortOrder)

		// Negative sort order
		err = category.UpdateSortOrder(-1)
		assert.Error(t, err)
	})

	t.Run("UpdateImage", func(t *testing.T) {
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Electronics",
			Level:    0,
			Path:     "/electronics",
			IsActive: true,
		}

		// Valid image URL
		err := category.UpdateImage("https://example.com/image.jpg")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com/image.jpg", category.ImageURL)

		// Invalid image URL
		err = category.UpdateImage("not-a-url")
		assert.Error(t, err)

		// Clear image URL
		err = category.UpdateImage("")
		assert.NoError(t, err)
		assert.Equal(t, "", category.ImageURL)
	})

	t.Run("MoveToParent", func(t *testing.T) {
		parentID := uuid.New()
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Laptops",
			ParentID: &parentID,
			Level:    1,
			Path:     "/electronics/laptops",
			IsActive: true,
		}

		// Move to self should fail
		err := category.MoveToParent(&category.ID, "")
		assert.Error(t, err)

		// Move to root (from having a parent)
		err = category.MoveToParent(nil, "")
		assert.Error(t, err) // This should error because parent changed

		// Create new category without parent
		category2 := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Laptops",
			Level:    0,
			Path:     "/laptops",
			IsActive: true,
		}

		// Move to root when already at root
		err = category2.MoveToParent(nil, "")
		assert.NoError(t, err)
		assert.Equal(t, 0, category2.Level)
	})

	t.Run("UpdateSEOFields", func(t *testing.T) {
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Electronics",
			Level:    0,
			Path:     "/electronics",
			IsActive: true,
		}

		// Valid update
		err := category.UpdateSEOFields("SEO Title", "SEO Description", "seo, keywords")
		assert.NoError(t, err)
		assert.Equal(t, "SEO Title", category.SEOTitle)
		assert.Equal(t, "SEO Description", category.SEODescription)
		assert.Equal(t, "seo, keywords", category.SEOKeywords)

		// SEO title too long
		longTitle := string(make([]byte, 201))
		err = category.UpdateSEOFields(longTitle, "", "")
		assert.Error(t, err)

		// SEO description too long
		longDesc := string(make([]byte, 301))
		err = category.UpdateSEOFields("", longDesc, "")
		assert.Error(t, err)

		// SEO keywords too long
		longKeywords := string(make([]byte, 501))
		err = category.UpdateSEOFields("", "", longKeywords)
		assert.Error(t, err)
	})

	t.Run("ToSafeCategory", func(t *testing.T) {
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Electronics",
			Level:    0,
			Path:     "/electronics",
			IsActive: true,
		}

		safe := category.ToSafeCategory()
		assert.Equal(t, category.ID, safe.ID)
		assert.Equal(t, category.Name, safe.Name)
	})
}

// TestProductVariant_ComprehensiveCoverage tests all variant methods
func TestProductVariant_ComprehensiveCoverage(t *testing.T) {
	t.Run("UpdateDetails", func(t *testing.T) {
		variant := &ProductVariant{
			ID:        uuid.New(),
			ProductID: uuid.New(),
			SKU:       "VAR-001",
			Name:      "Large Red",
			Price:     decimal.NewFromFloat(50.00),
			Cost:      decimal.NewFromFloat(30.00),
			IsActive:  true,
		}

		// Valid update
		err := variant.UpdateDetails("Large Blue")
		assert.NoError(t, err)
		assert.Equal(t, "Large Blue", variant.Name)

		// Empty name
		err = variant.UpdateDetails("")
		assert.Error(t, err)

		// Name too long
		longName := string(make([]byte, 301))
		err = variant.UpdateDetails(longName)
		assert.Error(t, err)
	})

	t.Run("ToSafeVariant", func(t *testing.T) {
		variant := &ProductVariant{
			ID:        uuid.New(),
			ProductID: uuid.New(),
			SKU:       "VAR-001",
			Name:      "Large Red",
			Price:     decimal.NewFromFloat(50.00),
			Cost:      decimal.NewFromFloat(30.00),
			IsActive:  true,
		}

		safe := variant.ToSafeVariant()
		assert.Equal(t, variant.ID, safe.ID)
		assert.Equal(t, variant.SKU, safe.SKU)
		assert.True(t, safe.Cost.IsZero()) // Cost should be removed
	})
}

// TestVariantImage_Comprehensive tests variant image methods
func TestVariantImage_Comprehensive(t *testing.T) {
	image := &VariantImage{
		ID:        uuid.New(),
		VariantID: uuid.New(),
		ImageURL:  "https://example.com/image.jpg",
		IsMain:    false,
	}

	// Test that image is not main initially
	assert.False(t, image.IsMain)
}

// TestProduct_AdditionalCoverage adds more coverage for missing methods
func TestProduct_AdditionalCoverage(t *testing.T) {
	t.Run("Activate", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.NewFromFloat(100.00),
			Cost:       decimal.NewFromFloat(60.00),
			IsActive:   false,
		}

		product.Activate()
		assert.True(t, product.IsActive)
	})

	t.Run("Deactivate", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.NewFromFloat(100.00),
			Cost:       decimal.NewFromFloat(60.00),
			IsActive:   true,
		}

		product.Deactivate()
		assert.False(t, product.IsActive)
	})

	t.Run("UpdateStock", func(t *testing.T) {
		product := &Product{
			ID:             uuid.New(),
			SKU:            "TEST-001",
			Name:           "Test Product",
			CategoryID:     uuid.New(),
			Price:          decimal.NewFromFloat(100.00),
			Cost:           decimal.NewFromFloat(60.00),
			TrackInventory: true,
			StockQuantity:  50,
			IsActive:       true,
		}

		// Valid update
		err := product.UpdateStock(100)
		assert.NoError(t, err)
		assert.Equal(t, 100, product.StockQuantity)

		// Negative stock
		err = product.UpdateStock(-10)
		assert.Error(t, err)

		// Exceeding max
		err = product.UpdateStock(1000000)
		assert.Error(t, err)
	})

	t.Run("UpdatePrice", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.NewFromFloat(100.00),
			Cost:       decimal.NewFromFloat(60.00),
			IsActive:   true,
		}

		// Valid update
		err := product.UpdatePrice(decimal.NewFromFloat(120.00))
		assert.NoError(t, err)
		assert.True(t, product.Price.Equal(decimal.NewFromFloat(120.00)))

		// Zero or negative price
		err = product.UpdatePrice(decimal.Zero)
		assert.Error(t, err)

		err = product.UpdatePrice(decimal.NewFromFloat(-10.00))
		assert.Error(t, err)

		// Price exceeding max
		err = product.UpdatePrice(decimal.NewFromFloat(1000000.00))
		assert.Error(t, err)
	})

	t.Run("CalculateTax", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.NewFromFloat(100.00),
			Cost:       decimal.NewFromFloat(60.00),
			Taxable:    true,
			TaxRate:    decimal.NewFromFloat(8.00),
			IsActive:   true,
		}

		tax := product.CalculateTax()
		expected := decimal.NewFromFloat(8.00)
		assert.True(t, tax.Equal(expected))

		// Non-taxable product
		product.Taxable = false
		tax = product.CalculateTax()
		assert.True(t, tax.IsZero())
	})

	t.Run("CalculateTotalPrice", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.NewFromFloat(100.00),
			Cost:       decimal.NewFromFloat(60.00),
			Taxable:    true,
			TaxRate:    decimal.NewFromFloat(8.00),
			IsActive:   true,
		}

		total := product.CalculateTotalPrice()
		expected := decimal.NewFromFloat(108.00)
		assert.True(t, total.Equal(expected))
	})

	t.Run("validateDownloadURL", func(t *testing.T) {
		product := &Product{
			ID:          uuid.New(),
			SKU:         "TEST-001",
			Name:        "Test Product",
			CategoryID:  uuid.New(),
			Price:       decimal.NewFromFloat(100.00),
			Cost:        decimal.NewFromFloat(60.00),
			IsDigital:   true,
			DownloadURL: "not-a-url",
			IsActive:    true,
		}

		err := product.Validate()
		assert.Error(t, err)

		// Valid URL
		product.DownloadURL = "https://example.com/download"
		err = product.Validate()
		assert.NoError(t, err)
	})
}

// TestProductCategory_AdditionalCoverage adds more coverage for missing methods
func TestProductCategory_AdditionalCoverage(t *testing.T) {
	t.Run("Activate", func(t *testing.T) {
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Electronics",
			Level:    0,
			Path:     "/electronics",
			IsActive: false,
		}

		category.Activate()
		assert.True(t, category.IsActive)
	})

	t.Run("Deactivate", func(t *testing.T) {
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Electronics",
			Level:    0,
			Path:     "/electronics",
			IsActive: true,
		}

		category.Deactivate()
		assert.False(t, category.IsActive)
	})

	t.Run("IsActiveCategory", func(t *testing.T) {
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Electronics",
			Level:    0,
			Path:     "/electronics",
			IsActive: true,
		}

		assert.True(t, category.IsActiveCategory())

		category.IsActive = false
		assert.False(t, category.IsActiveCategory())
	})
}

// TestProductVariant_AdditionalCoverage adds more coverage for missing methods
func TestProductVariant_AdditionalCoverage(t *testing.T) {
	t.Run("Activate", func(t *testing.T) {
		variant := &ProductVariant{
			ID:        uuid.New(),
			ProductID: uuid.New(),
			SKU:       "VAR-001",
			Name:      "Large Red",
			Price:     decimal.NewFromFloat(50.00),
			Cost:      decimal.NewFromFloat(30.00),
			IsActive:  false,
		}

		variant.Activate()
		assert.True(t, variant.IsActive)
	})

	t.Run("Deactivate", func(t *testing.T) {
		variant := &ProductVariant{
			ID:        uuid.New(),
			ProductID: uuid.New(),
			SKU:       "VAR-001",
			Name:      "Large Red",
			Price:     decimal.NewFromFloat(50.00),
			Cost:      decimal.NewFromFloat(30.00),
			IsActive:  true,
		}

		variant.Deactivate()
		assert.False(t, variant.IsActive)
	})

	t.Run("IsActiveVariant", func(t *testing.T) {
		variant := &ProductVariant{
			ID:        uuid.New(),
			ProductID: uuid.New(),
			SKU:       "VAR-001",
			Name:      "Large Red",
			Price:     decimal.NewFromFloat(50.00),
			Cost:      decimal.NewFromFloat(30.00),
			IsActive:  true,
		}

		assert.True(t, variant.IsActiveVariant())

		variant.IsActive = false
		assert.False(t, variant.IsActiveVariant())
	})

	t.Run("UpdatePrice", func(t *testing.T) {
		variant := &ProductVariant{
			ID:        uuid.New(),
			ProductID: uuid.New(),
			SKU:       "VAR-001",
			Name:      "Large Red",
			Price:     decimal.NewFromFloat(50.00),
			Cost:      decimal.NewFromFloat(30.00),
			IsActive:  true,
		}

		// Valid update
		err := variant.UpdatePrice(decimal.NewFromFloat(60.00))
		assert.NoError(t, err)
		assert.True(t, variant.Price.Equal(decimal.NewFromFloat(60.00)))

		// Zero or negative price
		err = variant.UpdatePrice(decimal.Zero)
		assert.Error(t, err)

		// Price exceeding max
		err = variant.UpdatePrice(decimal.NewFromFloat(1000000.00))
		assert.Error(t, err)
	})

	t.Run("UpdateCost", func(t *testing.T) {
		variant := &ProductVariant{
			ID:        uuid.New(),
			ProductID: uuid.New(),
			SKU:       "VAR-001",
			Name:      "Large Red",
			Price:     decimal.NewFromFloat(50.00),
			Cost:      decimal.NewFromFloat(30.00),
			IsActive:  true,
		}

		// Valid update
		err := variant.UpdateCost(decimal.NewFromFloat(35.00))
		assert.NoError(t, err)
		assert.True(t, variant.Cost.Equal(decimal.NewFromFloat(35.00)))

		// Negative cost
		err = variant.UpdateCost(decimal.NewFromFloat(-10.00))
		assert.Error(t, err)

		// Cost exceeding price
		err = variant.UpdateCost(decimal.NewFromFloat(60.00))
		assert.Error(t, err)
	})

	t.Run("UpdateStock", func(t *testing.T) {
		variant := &ProductVariant{
			ID:             uuid.New(),
			ProductID:      uuid.New(),
			SKU:            "VAR-001",
			Name:           "Large Red",
			Price:          decimal.NewFromFloat(50.00),
			Cost:           decimal.NewFromFloat(30.00),
			TrackInventory: true,
			StockQuantity:  50,
			IsActive:       true,
		}

		// Valid update
		err := variant.UpdateStock(100)
		assert.NoError(t, err)
		assert.Equal(t, 100, variant.StockQuantity)

		// Negative stock
		err = variant.UpdateStock(-10)
		assert.Error(t, err)
	})

	t.Run("AdjustStock", func(t *testing.T) {
		variant := &ProductVariant{
			ID:             uuid.New(),
			ProductID:      uuid.New(),
			SKU:            "VAR-001",
			Name:           "Large Red",
			Price:          decimal.NewFromFloat(50.00),
			Cost:           decimal.NewFromFloat(30.00),
			TrackInventory: true,
			StockQuantity:  50,
			IsActive:       true,
		}

		// Positive adjustment
		err := variant.AdjustStock(25)
		assert.NoError(t, err)
		assert.Equal(t, 75, variant.StockQuantity)

		// Negative adjustment
		err = variant.AdjustStock(-10)
		assert.NoError(t, err)
		assert.Equal(t, 65, variant.StockQuantity)

		// Adjustment resulting in negative
		err = variant.AdjustStock(-100)
		assert.Error(t, err)
	})

	t.Run("UpdateImage", func(t *testing.T) {
		variant := &ProductVariant{
			ID:        uuid.New(),
			ProductID: uuid.New(),
			SKU:       "VAR-001",
			Name:      "Large Red",
			Price:     decimal.NewFromFloat(50.00),
			Cost:      decimal.NewFromFloat(30.00),
			IsActive:  true,
		}

		// Valid image URL
		err := variant.UpdateImage("https://example.com/image.jpg")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com/image.jpg", variant.ImageURL)

		// Invalid image URL
		err = variant.UpdateImage("not-a-url")
		assert.Error(t, err)

		// Clear image URL
		err = variant.UpdateImage("")
		assert.NoError(t, err)
		assert.Equal(t, "", variant.ImageURL)
	})
}


// TestProduct_ValidationCoverage adds more validation coverage
func TestProduct_ValidationCoverage(t *testing.T) {
	t.Run("validateBarcode_AllCases", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.NewFromFloat(100.00),
			Cost:       decimal.NewFromFloat(60.00),
			IsActive:   true,
		}

		// Valid barcode
		product.Barcode = "123456789012"
		err := product.Validate()
		assert.NoError(t, err)

		// Barcode too long
		product.Barcode = string(make([]byte, 51))
		err = product.Validate()
		assert.Error(t, err)

		// Invalid barcode characters
		product.Barcode = "BARCODE@#$"
		err = product.Validate()
		assert.Error(t, err)

		// Empty barcode (should be valid)
		product.Barcode = ""
		err = product.Validate()
		assert.NoError(t, err)
	})

	t.Run("validateInventory_AllCases", func(t *testing.T) {
		product := &Product{
			ID:            uuid.New(),
			SKU:           "TEST-001",
			Name:          "Test Product",
			CategoryID:    uuid.New(),
			Price:         decimal.NewFromFloat(100.00),
			Cost:          decimal.NewFromFloat(60.00),
			StockQuantity: 50,
			MinStockLevel: 10,
			MaxStockLevel: 100,
			IsActive:      true,
		}

		// Valid inventory
		err := product.Validate()
		assert.NoError(t, err)

		// Stock quantity exceeding max
		product.StockQuantity = 1000000
		err = product.Validate()
		assert.Error(t, err)
		product.StockQuantity = 50

		// Min stock exceeding max
		product.MinStockLevel = 1000000
		err = product.Validate()
		assert.Error(t, err)
		product.MinStockLevel = 10

		// Max stock exceeding max
		product.MaxStockLevel = 1000000
		err = product.Validate()
		assert.Error(t, err)
		product.MaxStockLevel = 100

		// Max < Min
		product.MaxStockLevel = 5
		err = product.Validate()
		assert.Error(t, err)
		product.MaxStockLevel = 100
	})

	t.Run("validateTaxSettings_AllCases", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.NewFromFloat(100.00),
			Cost:       decimal.NewFromFloat(60.00),
			Taxable:    true,
			TaxRate:    decimal.NewFromFloat(8.00),
			IsActive:   true,
		}

		// Valid tax settings
		err := product.Validate()
		assert.NoError(t, err)

		// Tax rate exceeding 100%
		product.TaxRate = decimal.NewFromFloat(150.00)
		err = product.Validate()
		assert.Error(t, err)

		// Non-taxable with non-zero tax rate
		product.Taxable = false
		product.TaxRate = decimal.NewFromFloat(8.00)
		err = product.Validate()
		assert.Error(t, err)
	})

	t.Run("validateDigitalSettings_AllCases", func(t *testing.T) {
		product := &Product{
			ID:              uuid.New(),
			SKU:             "TEST-001",
			Name:            "Test Product",
			CategoryID:      uuid.New(),
			Price:           decimal.NewFromFloat(100.00),
			Cost:            decimal.NewFromFloat(60.00),
			IsDigital:       true,
			DownloadURL:     "https://example.com/download",
			MaxDownloads:    10,
			ExpiryDays:      30,
			RequiresShipping: false,
			IsActive:        true,
		}

		// Valid digital product
		err := product.Validate()
		assert.NoError(t, err)

		// Digital product requiring shipping
		product.RequiresShipping = true
		err = product.Validate()
		assert.Error(t, err)
		product.RequiresShipping = false

		// Digital product without download URL
		product.DownloadURL = ""
		err = product.Validate()
		assert.Error(t, err)
		product.DownloadURL = "https://example.com/download"

		// Max downloads exceeding limit
		product.MaxDownloads = 10000
		err = product.Validate()
		assert.Error(t, err)
		product.MaxDownloads = 10

		// Expiry days exceeding limit
		product.ExpiryDays = 4000
		err = product.Validate()
		assert.Error(t, err)
		product.ExpiryDays = 30

		// Physical product with digital settings
		product.IsDigital = false
		product.DownloadURL = "https://example.com/download"
		err = product.Validate()
		assert.Error(t, err)

		product.DownloadURL = ""
		product.MaxDownloads = 10
		err = product.Validate()
		assert.Error(t, err)

		product.MaxDownloads = 0
		product.ExpiryDays = 30
		err = product.Validate()
		assert.Error(t, err)
	})

	t.Run("validatePricing_AllCases", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.NewFromFloat(100.00),
			Cost:       decimal.NewFromFloat(60.00),
			IsActive:   true,
		}

		// Valid pricing
		err := product.Validate()
		assert.NoError(t, err)

		// Price exceeding max
		product.Price = decimal.NewFromFloat(1000000.00)
		err = product.Validate()
		assert.Error(t, err)
		product.Price = decimal.NewFromFloat(100.00)

		// Cost exceeding max
		product.Cost = decimal.NewFromFloat(1000000.00)
		err = product.Validate()
		assert.Error(t, err)
		product.Cost = decimal.NewFromFloat(60.00)

		// Cost > Price
		product.Cost = decimal.NewFromFloat(150.00)
		err = product.Validate()
		assert.Error(t, err)
	})

	t.Run("validatePhysicalProperties_AllCases", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.NewFromFloat(100.00),
			Cost:       decimal.NewFromFloat(60.00),
			Weight:     5.5,
			Length:     10.0,
			Width:      8.0,
			Height:     6.0,
			Volume:     480.0,
			Dimensions: "10 x 8 x 6",
			IsActive:   true,
		}

		// Valid physical properties
		err := product.Validate()
		assert.NoError(t, err)

		// Weight exceeding max
		product.Weight = 1000000.00
		err = product.Validate()
		assert.Error(t, err)
		product.Weight = 5.5

		// Length exceeding max
		product.Length = 100000.0
		err = product.Validate()
		assert.Error(t, err)
		product.Length = 10.0

		// Width exceeding max
		product.Width = 100000.0
		err = product.Validate()
		assert.Error(t, err)
		product.Width = 8.0

		// Height exceeding max
		product.Height = 100000.0
		err = product.Validate()
		assert.Error(t, err)
		product.Height = 6.0

		// Invalid dimensions format
		product.Dimensions = "invalid-format"
		err = product.Validate()
		assert.Error(t, err)
	})

	t.Run("IsInStock_AllCases", func(t *testing.T) {
		product := &Product{
			ID:             uuid.New(),
			SKU:            "TEST-001",
			Name:           "Test Product",
			CategoryID:     uuid.New(),
			Price:          decimal.NewFromFloat(100.00),
			Cost:           decimal.NewFromFloat(60.00),
			TrackInventory: true,
			StockQuantity:  10,
			AllowBackorder: false,
			IsActive:       true,
		}

		// In stock
		assert.True(t, product.IsInStock())

		// Out of stock, no backorder
		product.StockQuantity = 0
		assert.False(t, product.IsInStock())

		// Out of stock, with backorder
		product.AllowBackorder = true
		assert.True(t, product.IsInStock())

		// Not tracking inventory
		product.TrackInventory = false
		product.AllowBackorder = false
		product.StockQuantity = 0
		assert.True(t, product.IsInStock())
	})

	t.Run("IsLowStock_AllCases", func(t *testing.T) {
		product := &Product{
			ID:             uuid.New(),
			SKU:            "TEST-001",
			Name:           "Test Product",
			CategoryID:     uuid.New(),
			Price:          decimal.NewFromFloat(100.00),
			Cost:           decimal.NewFromFloat(60.00),
			TrackInventory: true,
			StockQuantity:  5,
			MinStockLevel:  10,
			IsActive:       true,
		}

		// Low stock
		assert.True(t, product.IsLowStock())

		// Not low stock
		product.StockQuantity = 15
		assert.False(t, product.IsLowStock())

		// Not tracking inventory
		product.TrackInventory = false
		product.StockQuantity = 5
		assert.False(t, product.IsLowStock())

		// Min stock level not set
		product.TrackInventory = true
		product.MinStockLevel = 0
		assert.False(t, product.IsLowStock())
	})

	t.Run("CalculateProfitMargin_ZeroPrice", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.Zero,
			Cost:       decimal.NewFromFloat(60.00),
			IsActive:   true,
		}

		margin := product.CalculateProfitMargin()
		assert.True(t, margin.IsZero())
	})
}


// TestProduct_EdgeCases adds more edge case coverage
func TestProduct_EdgeCases(t *testing.T) {
	t.Run("CanFulfillOrder_EdgeCases", func(t *testing.T) {
		product := &Product{
			ID:             uuid.New(),
			SKU:            "TEST-001",
			Name:           "Test Product",
			CategoryID:     uuid.New(),
			Price:          decimal.NewFromFloat(100.00),
			Cost:           decimal.NewFromFloat(60.00),
			TrackInventory: true,
			StockQuantity:  10,
			AllowBackorder: false,
			IsActive:       false, // Inactive
		}

		// Inactive product
		err := product.CanFulfillOrder(5)
		assert.Error(t, err)

		// Active product, sufficient stock
		product.IsActive = true
		err = product.CanFulfillOrder(5)
		assert.NoError(t, err)

		// Insufficient stock, no backorder
		err = product.CanFulfillOrder(15)
		assert.Error(t, err)

		// Insufficient stock, with backorder
		product.AllowBackorder = true
		err = product.CanFulfillOrder(15)
		assert.NoError(t, err)

		// Not tracking inventory
		product.TrackInventory = false
		product.AllowBackorder = false
		product.StockQuantity = 0
		err = product.CanFulfillOrder(100)
		assert.NoError(t, err)
	})

	t.Run("validateSKU_EdgeCases", func(t *testing.T) {
		product := &Product{
			ID:         uuid.New(),
			SKU:        "TEST-001",
			Name:       "Test Product",
			CategoryID: uuid.New(),
			Price:      decimal.NewFromFloat(100.00),
			Cost:       decimal.NewFromFloat(60.00),
			IsActive:   true,
		}

		// Valid SKU
		err := product.Validate()
		assert.NoError(t, err)

		// SKU too long
		product.SKU = string(make([]byte, 101))
		err = product.Validate()
		assert.Error(t, err)

		// Invalid SKU characters
		product.SKU = "TEST@#$"
		err = product.Validate()
		assert.Error(t, err)
	})

	t.Run("validateDownloadURL_EdgeCases", func(t *testing.T) {
		product := &Product{
			ID:          uuid.New(),
			SKU:         "TEST-001",
			Name:        "Test Product",
			CategoryID:  uuid.New(),
			Price:       decimal.NewFromFloat(100.00),
			Cost:        decimal.NewFromFloat(60.00),
			IsDigital:   true,
			DownloadURL: "https://example.com/download",
			IsActive:    true,
		}

		// Valid URL
		err := product.Validate()
		assert.NoError(t, err)

		// URL too long
		product.DownloadURL = "https://example.com/" + string(make([]byte, 1000))
		err = product.Validate()
		assert.Error(t, err)
	})
}

// TestProductCategory_EdgeCases adds more edge case coverage for categories
func TestProductCategory_EdgeCases(t *testing.T) {
	t.Run("validateImageURL_EdgeCases", func(t *testing.T) {
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Electronics",
			Level:    0,
			Path:     "/electronics",
			ImageURL: "https://example.com/image.jpg",
			IsActive: true,
		}

		// Valid image URL
		err := category.Validate()
		assert.NoError(t, err)

		// Image URL too long
		category.ImageURL = "https://example.com/" + string(make([]byte, 1000))
		err = category.Validate()
		assert.Error(t, err)

		// Invalid image URL format
		category.ImageURL = "not-a-url"
		err = category.Validate()
		assert.Error(t, err)

		// Empty image URL (should be valid)
		category.ImageURL = ""
		err = category.Validate()
		assert.NoError(t, err)
	})

	t.Run("BuildPath", func(t *testing.T) {
		// Root category
		category := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Laptops",
			Level:    0,
			IsActive: true,
		}
		category.BuildPath("")
		assert.Equal(t, "/laptops", category.Path)
		assert.Equal(t, 0, category.Level)

		// Child category
		parentID := uuid.New()
		category2 := &ProductCategory{
			ID:       uuid.New(),
			Name:     "Gaming Laptops",
			ParentID: &parentID,
			IsActive: true,
		}
		category2.BuildPath("/electronics")
		assert.Equal(t, "/electronics/gaming-laptops", category2.Path)
		assert.Equal(t, 1, category2.Level)
	})
}

// TestProductVariant_EdgeCases adds more edge case coverage for variants
func TestProductVariant_EdgeCases(t *testing.T) {
	t.Run("CanFulfillOrder_EdgeCases", func(t *testing.T) {
		variant := &ProductVariant{
			ID:             uuid.New(),
			ProductID:      uuid.New(),
			SKU:            "VAR-001",
			Name:           "Large Red",
			Price:          decimal.NewFromFloat(50.00),
			Cost:           decimal.NewFromFloat(30.00),
			TrackInventory: true,
			StockQuantity:  10,
			AllowBackorder: false,
			IsActive:       false, // Inactive
		}

		// Inactive variant
		err := variant.CanFulfillOrder(5)
		assert.Error(t, err)

		// Active variant, sufficient stock
		variant.IsActive = true
		err = variant.CanFulfillOrder(5)
		assert.NoError(t, err)

		// Insufficient stock, no backorder
		err = variant.CanFulfillOrder(15)
		assert.Error(t, err)

		// Insufficient stock, with backorder
		variant.AllowBackorder = true
		err = variant.CanFulfillOrder(15)
		assert.NoError(t, err)
	})

	t.Run("IsInStock_EdgeCases", func(t *testing.T) {
		variant := &ProductVariant{
			ID:             uuid.New(),
			ProductID:      uuid.New(),
			SKU:            "VAR-001",
			Name:           "Large Red",
			Price:          decimal.NewFromFloat(50.00),
			Cost:           decimal.NewFromFloat(30.00),
			TrackInventory: true,
			StockQuantity:  10,
			AllowBackorder: false,
			IsActive:       true,
		}

		// In stock
		assert.True(t, variant.IsInStock())

		// Out of stock, no backorder
		variant.StockQuantity = 0
		assert.False(t, variant.IsInStock())

		// Out of stock, with backorder
		variant.AllowBackorder = true
		assert.True(t, variant.IsInStock())

		// Not tracking inventory
		variant.TrackInventory = false
		variant.AllowBackorder = false
		variant.StockQuantity = 0
		assert.True(t, variant.IsInStock())
	})

	t.Run("IsLowStock_EdgeCases", func(t *testing.T) {
		variant := &ProductVariant{
			ID:             uuid.New(),
			ProductID:      uuid.New(),
			SKU:            "VAR-001",
			Name:           "Large Red",
			Price:          decimal.NewFromFloat(50.00),
			Cost:           decimal.NewFromFloat(30.00),
			TrackInventory: true,
			StockQuantity:  5,
			MinStockLevel:  10,
			IsActive:       true,
		}

		// Low stock
		assert.True(t, variant.IsLowStock())

		// Not low stock
		variant.StockQuantity = 15
		assert.False(t, variant.IsLowStock())

		// Not tracking inventory
		variant.TrackInventory = false
		variant.StockQuantity = 5
		assert.False(t, variant.IsLowStock())

		// Min stock level not set
		variant.TrackInventory = true
		variant.MinStockLevel = 0
		assert.False(t, variant.IsLowStock())
	})

	t.Run("CalculateProfit", func(t *testing.T) {
		variant := &ProductVariant{
			ID:        uuid.New(),
			ProductID: uuid.New(),
			SKU:       "VAR-001",
			Name:      "Large Red",
			Price:     decimal.NewFromFloat(50.00),
			Cost:      decimal.NewFromFloat(30.00),
			IsActive:  true,
		}

		profit := variant.CalculateProfit()
		expected := decimal.NewFromFloat(20.00)
		assert.True(t, profit.Equal(expected))
	})

	t.Run("CalculateProfitMargin", func(t *testing.T) {
		variant := &ProductVariant{
			ID:        uuid.New(),
			ProductID: uuid.New(),
			SKU:       "VAR-001",
			Name:      "Large Red",
			Price:     decimal.NewFromFloat(50.00),
			Cost:      decimal.NewFromFloat(30.00),
			IsActive:  true,
		}

		margin := variant.CalculateProfitMargin()
		expected := decimal.NewFromFloat(40.00) // (50-30)/50 * 100 = 40%
		assert.True(t, margin.Equal(expected))

		// Zero price
		variant.Price = decimal.Zero
		margin = variant.CalculateProfitMargin()
		assert.True(t, margin.IsZero())
	})

	t.Run("CalculateTax", func(t *testing.T) {
		variant := &ProductVariant{
			ID:        uuid.New(),
			ProductID: uuid.New(),
			SKU:       "VAR-001",
			Name:      "Large Red",
			Price:     decimal.NewFromFloat(50.00),
			Cost:      decimal.NewFromFloat(30.00),
			Taxable:   true,
			TaxRate:   decimal.NewFromFloat(8.00),
			IsActive:  true,
		}

		tax := variant.CalculateTax()
		expected := decimal.NewFromFloat(4.00)
		assert.True(t, tax.Equal(expected))

		// Non-taxable
		variant.Taxable = false
		tax = variant.CalculateTax()
		assert.True(t, tax.IsZero())
	})

	t.Run("CalculateTotalPrice", func(t *testing.T) {
		variant := &ProductVariant{
			ID:        uuid.New(),
			ProductID: uuid.New(),
			SKU:       "VAR-001",
			Name:      "Large Red",
			Price:     decimal.NewFromFloat(50.00),
			Cost:      decimal.NewFromFloat(30.00),
			Taxable:   true,
			TaxRate:   decimal.NewFromFloat(8.00),
			IsActive:  true,
		}

		total := variant.CalculateTotalPrice()
		expected := decimal.NewFromFloat(54.00) // 50 + 4 (8% tax)
		assert.True(t, total.Equal(expected))
	})

	t.Run("SetAsMain", func(t *testing.T) {
		image := &VariantImage{
			ID:        uuid.New(),
			VariantID: uuid.New(),
			ImageURL:  "https://example.com/image.jpg",
			IsMain:    false,
		}

		image.SetAsMain()
		assert.True(t, image.IsMain)
	})
}
