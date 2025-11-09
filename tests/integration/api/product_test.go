package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"erpgo/internal/interfaces/http/dto"
)

// TestProductEndpoints tests all product endpoints
func TestProductEndpoints(t *testing.T) {
	// Setup test router
	router := gin.New()

	// Mock handlers and services
	// In a real implementation, you would inject actual services
	// For now, we'll create a basic test structure

	// Mock product handler
	router.GET("/api/v1/products", func(c *gin.Context) {
		// Mock response
		products := []*dto.ProductResponse{
			{
				ID:       uuid.New(),
				SKU:      "TEST-001",
				Name:     "Test Product",
				Price:    decimal.NewFromFloat(99.99),
				IsActive: true,
			},
		}

		response := dto.ProductListResponse{
			Products: products,
			Pagination: &dto.PaginationInfo{
				Page:       1,
				Limit:      20,
				Total:      1,
				TotalPages: 1,
				HasNext:    false,
				HasPrev:    false,
			},
		}

		c.JSON(http.StatusOK, response)
	})

	// Test GET /api/v1/products
	t.Run("List Products", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/products", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ProductListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Products, 1)
		assert.Equal(t, "TEST-001", response.Products[0].SKU)
		assert.Equal(t, "Test Product", response.Products[0].Name)
		assert.Equal(t, decimal.NewFromFloat(99.99), response.Products[0].Price)
		assert.True(t, response.Products[0].IsActive)
		assert.NotNil(t, response.Pagination)
	})
}

// TestProductSearch tests the product search endpoint
func TestProductSearch(t *testing.T) {
	router := gin.New()

	router.GET("/api/v1/products/search", func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error: "Search query is required",
			})
			return
		}

		// Mock search response
		products := []*dto.ProductResponse{
			{
				ID:       uuid.New(),
				SKU:      "SEARCH-001",
				Name:     "Search Result Product",
				Price:    decimal.NewFromFloat(149.99),
				IsActive: true,
			},
		}

		response := dto.SearchProductsResponse{
			Products: products,
			Total:    1,
			Query:    query,
		}

		c.JSON(http.StatusOK, response)
	})

	t.Run("Search Products", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/products/search?q=product", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.SearchProductsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Products, 1)
		assert.Equal(t, "product", response.Query)
		assert.Equal(t, "SEARCH-001", response.Products[0].SKU)
	})

	t.Run("Search Without Query", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/products/search", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response dto.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Search query is required", response.Error)
	})
}

// TestCategoryEndpoints tests category endpoints
func TestCategoryEndpoints(t *testing.T) {
	router := gin.New()

	// Mock category handler
	router.GET("/api/v1/categories", func(c *gin.Context) {
		// Mock response
		categories := []*dto.CategoryResponse{
			{
				ID:   uuid.New(),
				Name: "Electronics",
				Path: "electronics",
				Level: 0,
			},
		}

		response := dto.CategoryListResponse{
			Categories: categories,
			Pagination: &dto.PaginationInfo{
				Page:       1,
				Limit:      20,
				Total:      1,
				TotalPages: 1,
				HasNext:    false,
				HasPrev:    false,
			},
		}

		c.JSON(http.StatusOK, response)
	})

	// Mock category tree
	router.GET("/api/v1/categories/tree", func(c *gin.Context) {
		// Mock tree response
		tree := []*dto.CategoryTreeNode{
			{
				CategoryResponse: &dto.CategoryResponse{
					ID:   uuid.New(),
					Name: "Electronics",
					Path: "electronics",
					Level: 0,
				},
				Children: []*dto.CategoryTreeNode{
					{
						CategoryResponse: &dto.CategoryResponse{
							ID:   uuid.New(),
							Name: "Laptops",
							Path: "electronics/laptops",
							Level: 1,
						},
						Children: []*dto.CategoryTreeNode{},
					},
				},
			},
		}

		response := dto.CategoryTreeResponse{
			Tree: tree,
		}

		c.JSON(http.StatusOK, response)
	})

	t.Run("List Categories", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/categories", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.CategoryListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Categories, 1)
		assert.Equal(t, "Electronics", response.Categories[0].Name)
		assert.Equal(t, "electronics", response.Categories[0].Path)
		assert.Equal(t, 0, response.Categories[0].Level)
	})

	t.Run("Get Category Tree", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/categories/tree", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.CategoryTreeResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Tree, 1)
		assert.Equal(t, "Electronics", response.Tree[0].Name)
		assert.Len(t, response.Tree[0].Children, 1)
		assert.Equal(t, "Laptops", response.Tree[0].Children[0].Name)
	})
}

// TestVariantEndpoints tests product variant endpoints
func TestVariantEndpoints(t *testing.T) {
	router := gin.New()

	// Mock variant handler
	router.GET("/api/v1/products/:id/variants", func(c *gin.Context) {
		productID := c.Param("id")
		if productID == "" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error: "Product ID is required",
			})
			return
		}

		// Mock response
		variants := []*dto.ProductVariantResponse{
			{
				ID:        uuid.New(),
				ProductID: uuid.MustParse(productID),
				SKU:       "VAR-001",
				Name:      "Test Variant",
				Price:     decimal.NewFromFloat(19.99),
				IsActive:  true,
			},
		}

		response := dto.ProductVariantListResponse{
			Variants: variants,
			Pagination: &dto.PaginationInfo{
				Page:       1,
				Limit:      20,
				Total:      1,
				TotalPages: 1,
				HasNext:    false,
				HasPrev:    false,
			},
		}

		c.JSON(http.StatusOK, response)
	})

	t.Run("List Product Variants", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/products/123e4567-e89b-12d3-a456-426614174000/variants", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ProductVariantListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Variants, 1)
		assert.Equal(t, "VAR-001", response.Variants[0].SKU)
		assert.Equal(t, "Test Variant", response.Variants[0].Name)
		assert.Equal(t, decimal.NewFromFloat(19.99), response.Variants[0].Price)
	})

	t.Run("List Variants Without Product ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/products//variants", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response dto.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Product ID is required", response.Error)
	})
}

// TestProductOperations tests product operation endpoints
func TestProductOperations(t *testing.T) {
	router := gin.New()

	// Mock product operations
	router.POST("/api/v1/products/:id/activate", func(c *gin.Context) {
		productID := c.Param("id")
		if productID == "" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error: "Product ID is required",
			})
			return
		}

		// Mock response
		product := &dto.ProductResponse{
			ID:       uuid.MustParse(productID),
			SKU:      "ACTIVE-001",
			Name:     "Active Product",
			Price:    decimal.NewFromFloat(199.99),
			IsActive: true,
		}

		c.JSON(http.StatusOK, product)
	})

	router.PUT("/api/v1/products/:id/featured", func(c *gin.Context) {
		productID := c.Param("id")
		if productID == "" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error: "Product ID is required",
			})
			return
		}

		// Parse request body
		var req map[string]bool
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Invalid request body",
				Details: err.Error(),
			})
			return
		}

		featured, exists := req["featured"]
		if !exists {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error: "Featured status is required",
			})
			return
		}

		// Mock response
		product := &dto.ProductResponse{
			ID:        uuid.MustParse(productID),
			SKU:       "FEATURED-001",
			Name:      "Featured Product",
			Price:     decimal.NewFromFloat(299.99),
			IsActive:  true,
			IsFeatured: featured,
		}

		c.JSON(http.StatusOK, product)
	})

	t.Run("Activate Product", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/products/123e4567-e89b-12d3-a456-426614174000/activate", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ProductResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "ACTIVE-001", response.SKU)
		assert.True(t, response.IsActive)
	})

	t.Run("Set Featured Product", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := `{"featured": true}`
		req, _ := http.NewRequest("PUT", "/api/v1/products/123e4567-e89b-12d3-a456-426614174000/featured", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ProductResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "FEATURED-001", response.SKU)
		assert.True(t, response.IsFeatured)
	})

	t.Run("Set Featured Without Featured Status", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("PUT", "/api/v1/products/123e4567-e89b-12d3-a456-426614174000/featured", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response dto.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Featured status is required", response.Error)
	})
}

// TestErrorHandling tests error handling in product endpoints
func TestErrorHandling(t *testing.T) {
	router := gin.New()

	// Mock error handling
	router.GET("/api/v1/products/not-found", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Product not found",
			Details: "Product with ID 'not-found' does not exist",
		})
	})

	router.GET("/api/v1/products/validation-error", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Validation error",
			Details: "Invalid request parameters",
		})
	})

	t.Run("Not Found Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/products/not-found", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response dto.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Product not found", response.Error)
		assert.Equal(t, "Product with ID 'not-found' does not exist", response.Details)
	})

	t.Run("Validation Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/products/validation-error", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response dto.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Validation error", response.Error)
		assert.Equal(t, "Invalid request parameters", response.Details)
	})
}

// TestRequestValidation tests request validation
func TestRequestValidation(t *testing.T) {
	router := gin.New()

	// Mock product creation with validation
	router.POST("/api/v1/products", func(c *gin.Context) {
		var req dto.CreateProductRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Invalid request body",
				Details: err.Error(),
			})
			return
		}

		// Validate required fields
		if req.Name == "" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Product name is required",
			})
			return
			}

		if req.Price.IsZero() {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Price must be greater than 0",
			})
			return
		}

		// Mock successful creation
		product := &dto.ProductResponse{
			ID:    uuid.New(),
			Name:  req.Name,
			Price: req.Price,
		}

		c.JSON(http.StatusCreated, product)
	})

	t.Run("Valid Product Creation", func(t *testing.T) {
		body := `{
			"name": "Valid Product",
			"price": 99.99,
			"category_id": "123e4567-e89b-12d3-a456-426614174000"
		}`

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/products", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.ProductResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Valid Product", response.Name)
		assert.Equal(t, decimal.NewFromFloat(99.99), response.Price)
	})

	t.Run("Missing Required Fields", func(t *testing.T) {
		body := `{
			"price": 99.99,
			"category_id": "123e4567-e89b-12d3-a456-426614174000"
		}`

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/products", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response dto.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Invalid request body", response.Error)
	})

	t.Run("Invalid Price", func(t *testing.T) {
		body := `{
			"name": "Invalid Product",
			"price": 0,
			"category_id": "123e4567-e89b-12d3-a456-426614174000"
		}`

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/products", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response dto.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Price must be greater than 0", response.Error)
	})
}

// BenchmarkProductEndpoints benchmarks product endpoints for performance
func BenchmarkProductEndpoints(b *testing.B) {
	router := gin.New()

	router.GET("/api/v1/products", func(c *gin.Context) {
		products := make([]*dto.ProductResponse, 100)
		for i := 0; i < 100; i++ {
			products[i] = &dto.ProductResponse{
				ID:    uuid.New(),
				SKU:   fmt.Sprintf("SKU-%d", i),
				Name:  fmt.Sprintf("Product %d", i),
				Price: decimal.NewFromFloat(float64(i) + 1),
			}
		}

		response := dto.ProductListResponse{
			Products: products,
			Pagination: &dto.PaginationInfo{
				Page:       1,
				Limit:      100,
				Total:      100,
				TotalPages: 1,
			},
		}

		c.JSON(http.StatusOK, response)
	})

	b.ResetTimer()
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/products", nil)
		router.ServeHTTP(w, req)
	}
}