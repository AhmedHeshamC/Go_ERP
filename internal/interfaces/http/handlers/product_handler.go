package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/internal/application/services/product"
	"erpgo/internal/domain/products/entities"
	"erpgo/internal/interfaces/http/dto"
)

// ProductHandler handles product HTTP requests
type ProductHandler struct {
	productService product.Service
	logger         zerolog.Logger
}

// NewProductHandler creates a new product handler
func NewProductHandler(productService product.Service, logger zerolog.Logger) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		logger:         logger,
	}
}

// Product CRUD Operations

// CreateProduct creates a new product
// @Summary Create product
// @Description Create a new product with validation and automatic SKU generation
// @Tags products
// @Accept json
// @Produce json
// @Param product body dto.CreateProductRequest true "Product data"
// @Success 201 {object} dto.ProductResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid product creation request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &product.CreateProductRequest{
		SKU:             req.SKU,
		Name:            req.Name,
		Description:     req.Description,
		ShortDescription: req.ShortDescription,
		CategoryID:      req.CategoryID,
		Price:           req.Price,
		Cost:            req.Cost,
		Weight:          req.Weight,
		Length:          req.Length,
		Width:           req.Width,
		Height:          req.Height,
		Barcode:         req.Barcode,
		TrackInventory:  req.TrackInventory,
		StockQuantity:   req.StockQuantity,
		MinStockLevel:   req.MinStockLevel,
		MaxStockLevel:   req.MaxStockLevel,
		AllowBackorder:  req.AllowBackorder,
		RequiresShipping: req.RequiresShipping,
		Taxable:         req.Taxable,
		TaxRate:         req.TaxRate,
		IsFeatured:      req.IsFeatured,
		IsDigital:       req.IsDigital,
		DownloadURL:     req.DownloadURL,
		MaxDownloads:    req.MaxDownloads,
		ExpiryDays:      req.ExpiryDays,
	}

	product, err := h.productService.CreateProduct(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create product")
		handleProductError(c, err)
		return
	}

	response := h.productToResponse(product)
	c.JSON(http.StatusCreated, response)
}

// GetProduct retrieves a product by ID
// @Summary Get product
// @Description Get a product by its ID
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} dto.ProductResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{id} [get]
func (h *ProductHandler) GetProduct(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	product, err := h.productService.GetProduct(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Msg("Failed to get product")
		handleProductError(c, err)
		return
	}

	response := h.productToResponse(product)
	c.JSON(http.StatusOK, response)
}

// GetProductBySKU retrieves a product by SKU
// @Summary Get product by SKU
// @Description Get a product by its SKU
// @Tags products
// @Accept json
// @Produce json
// @Param sku path string true "Product SKU"
// @Success 200 {object} dto.ProductResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/sku/{sku} [get]
func (h *ProductHandler) GetProductBySKU(c *gin.Context) {
	sku := c.Param("sku")
	if sku == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product SKU is required",
		})
		return
	}

	product, err := h.productService.GetProductBySKU(c, sku)
	if err != nil {
		h.logger.Error().Err(err).Str("sku", sku).Msg("Failed to get product by SKU")
		handleProductError(c, err)
		return
	}

	response := h.productToResponse(product)
	c.JSON(http.StatusOK, response)
}

// UpdateProduct updates a product
// @Summary Update product
// @Description Update an existing product
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param product body dto.UpdateProductRequest true "Product data"
// @Success 200 {object} dto.ProductResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{id} [put]
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid product update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &product.UpdateProductRequest{
		Name:            req.Name,
		Description:     req.Description,
		ShortDescription: req.ShortDescription,
		CategoryID:      req.CategoryID,
		Price:           req.Price,
		Cost:            req.Cost,
		Weight:          req.Weight,
		Length:          req.Length,
		Width:           req.Width,
		Height:          req.Height,
		Barcode:         req.Barcode,
		TrackInventory:  req.TrackInventory,
		MinStockLevel:   req.MinStockLevel,
		MaxStockLevel:   req.MaxStockLevel,
		AllowBackorder:  req.AllowBackorder,
		RequiresShipping: req.RequiresShipping,
		Taxable:         req.Taxable,
		TaxRate:         req.TaxRate,
		IsFeatured:      req.IsFeatured,
		IsDigital:       req.IsDigital,
		DownloadURL:     req.DownloadURL,
		MaxDownloads:    req.MaxDownloads,
		ExpiryDays:      req.ExpiryDays,
	}

	updatedProduct, err := h.productService.UpdateProduct(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Msg("Failed to update product")
		handleProductError(c, err)
		return
	}

	response := h.productToResponse(updatedProduct)
	c.JSON(http.StatusOK, response)
}

// DeleteProduct deletes a product (soft delete)
// @Summary Delete product
// @Description Delete a product (soft delete)
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 204
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	err := h.productService.DeleteProduct(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Msg("Failed to delete product")
		handleProductError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListProducts retrieves a paginated list of products
// @Summary List products
// @Description Get a paginated list of products with optional filtering
// @Tags products
// @Accept json
// @Produce json
// @Param search query string false "Search term"
// @Param category_id query string false "Category ID"
// @Param category_ids query []string false "Category IDs"
// @Param sku query string false "SKU"
// @Param min_price query number false "Minimum price"
// @Param max_price query number false "Maximum price"
// @Param is_active query bool false "Filter by active status"
// @Param is_featured query bool false "Filter by featured status"
// @Param is_digital query bool false "Filter by digital status"
// @Param track_inventory query bool false "Filter by inventory tracking"
// @Param in_stock query bool false "Filter by in stock status"
// @Param low_stock query bool false "Filter by low stock status"
// @Param created_after query string false "Created after (ISO 8601)"
// @Param created_before query string false "Created before (ISO 8601)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param sort_by query string false "Sort field" default("created_at")
// @Param sort_order query string false "Sort order" Enums(asc,desc) default("desc")
// @Success 200 {object} dto.ProductListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products [get]
func (h *ProductHandler) ListProducts(c *gin.Context) {
	var req dto.ListProductsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid product list request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Convert to service request
	serviceReq := &product.ListProductsRequest{
		Search:         req.Search,
		CategoryID:     req.CategoryID,
		CategoryIDs:    req.CategoryIDs,
		SKU:            req.SKU,
		MinPrice:       req.MinPrice,
		MaxPrice:       req.MaxPrice,
		IsActive:       req.IsActive,
		IsFeatured:     req.IsFeatured,
		IsDigital:      req.IsDigital,
		TrackInventory: req.TrackInventory,
		InStock:        req.InStock,
		LowStock:       req.LowStock,
		Page:           req.Page,
		Limit:          req.Limit,
		SortBy:         req.SortBy,
		SortOrder:      req.SortOrder,
	}

	result, err := h.productService.ListProducts(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list products")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve products",
		})
		return
	}

	// Convert to response DTO
	products := make([]*dto.ProductResponse, len(result.Products))
	for i, p := range result.Products {
		products[i] = h.productToResponse(p)
	}

	response := &dto.ProductListResponse{
		Products: products,
		Pagination: &dto.PaginationInfo{
			Page:       result.Pagination.Page,
			Limit:      result.Pagination.Limit,
			Total:      result.Pagination.Total,
			TotalPages: result.Pagination.TotalPages,
			HasNext:    result.Pagination.HasNext,
			HasPrev:    result.Pagination.HasPrev,
		},
	}

	c.JSON(http.StatusOK, response)
}

// SearchProducts searches products by query
// @Summary Search products
// @Description Search products by query string
// @Tags products
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Result limit" default(20)
// @Success 200 {object} dto.SearchProductsResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/search [get]
func (h *ProductHandler) SearchProducts(c *gin.Context) {
	var req dto.SearchProductsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid product search request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 50 {
		req.Limit = 50
	}

	serviceReq := &product.SearchProductsRequest{
		Query: req.Query,
		Limit: req.Limit,
	}

	result, err := h.productService.SearchProducts(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to search products")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to search products",
		})
		return
	}

	// Convert to response DTO
	products := make([]*dto.ProductResponse, len(result.Products))
	for i, p := range result.Products {
		products[i] = h.productToResponse(p)
	}

	response := &dto.SearchProductsResponse{
		Products: products,
		Total:    result.Total,
		Query:    req.Query,
	}

	c.JSON(http.StatusOK, response)
}

// Product Operations

// ActivateProduct activates a product
// @Summary Activate product
// @Description Activate a product
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} dto.ProductResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{id}/activate [post]
func (h *ProductHandler) ActivateProduct(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	err := h.productService.ActivateProduct(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Msg("Failed to activate product")
		handleProductError(c, err)
		return
	}

	// Return updated product
	product, err := h.productService.GetProduct(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Msg("Failed to get updated product")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve updated product",
		})
		return
	}

	response := h.productToResponse(product)
	c.JSON(http.StatusOK, response)
}

// DeactivateProduct deactivates a product
// @Summary Deactivate product
// @Description Deactivate a product
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} dto.ProductResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{id}/deactivate [post]
func (h *ProductHandler) DeactivateProduct(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	err := h.productService.DeactivateProduct(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Msg("Failed to deactivate product")
		handleProductError(c, err)
		return
	}

	// Return updated product
	product, err := h.productService.GetProduct(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Msg("Failed to get updated product")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve updated product",
		})
		return
	}

	response := h.productToResponse(product)
	c.JSON(http.StatusOK, response)
}

// SetFeaturedProduct sets or unsets a product as featured
// @Summary Set featured product
// @Description Set or unset a product as featured
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param featured body map[bool]bool true "Featured status"
// @Success 200 {object} dto.ProductResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{id}/featured [put]
func (h *ProductHandler) SetFeaturedProduct(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	var req map[string]bool
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid featured product request")
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

	err := h.productService.SetFeaturedProduct(c, id, featured)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Bool("featured", featured).Msg("Failed to set featured product")
		handleProductError(c, err)
		return
	}

	// Return updated product
	product, err := h.productService.GetProduct(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Msg("Failed to get updated product")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve updated product",
		})
		return
	}

	response := h.productToResponse(product)
	c.JSON(http.StatusOK, response)
}

// UpdateProductPrice updates product price
// @Summary Update product price
// @Description Update product price and optionally cost
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param price body dto.UpdatePriceRequest true "Price data"
// @Success 200 {object} dto.ProductResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{id}/price [put]
func (h *ProductHandler) UpdateProductPrice(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	var req dto.UpdatePriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid price update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	serviceReq := &product.UpdatePriceRequest{
		Price: req.Price,
		Cost:  req.Cost,
	}

	err := h.productService.UpdateProductPrice(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Msg("Failed to update product price")
		handleProductError(c, err)
		return
	}

	// Return updated product
	product, err := h.productService.GetProduct(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Msg("Failed to get updated product")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve updated product",
		})
		return
	}

	response := h.productToResponse(product)
	c.JSON(http.StatusOK, response)
}

// UpdateProductStock updates product stock quantity
// @Summary Update product stock
// @Description Update the stock quantity for a product
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param stock body dto.UpdateStockRequest true "Stock data"
// @Success 200 {object} dto.ProductResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{id}/stock [put]
func (h *ProductHandler) UpdateProductStock(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	var req dto.UpdateStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid stock update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	serviceReq := &product.UpdateStockRequest{
		Quantity: req.Quantity,
	}

	err := h.productService.UpdateProductStock(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Int("quantity", req.Quantity).Msg("Failed to update product stock")
		handleProductError(c, err)
		return
	}

	// Return updated product
	product, err := h.productService.GetProduct(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Msg("Failed to get updated product")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve updated product",
		})
		return
	}

	response := h.productToResponse(product)
	c.JSON(http.StatusOK, response)
}

// AdjustProductStock adjusts product stock quantity
// @Summary Adjust product stock
// @Description Adjust the stock quantity for a product
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param adjustment body dto.AdjustStockRequest true "Adjustment data"
// @Success 200 {object} dto.ProductResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{id}/stock/adjust [post]
func (h *ProductHandler) AdjustProductStock(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	var req dto.AdjustStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid stock adjustment request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	serviceReq := &product.AdjustStockRequest{
		Adjustment: req.Adjustment,
		Reason:     req.Reason,
	}

	err := h.productService.AdjustProductStock(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Int("adjustment", req.Adjustment).Msg("Failed to adjust product stock")
		handleProductError(c, err)
		return
	}

	// Return updated product
	product, err := h.productService.GetProduct(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Msg("Failed to get updated product")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve updated product",
		})
		return
	}

	response := h.productToResponse(product)
	c.JSON(http.StatusOK, response)
}

// GetProductStockLevel retrieves stock level information
// @Summary Get product stock level
// @Description Get detailed stock level information for a product
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} dto.StockLevelResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{id}/stock [get]
func (h *ProductHandler) GetProductStockLevel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	stockLevel, err := h.productService.GetProductStockLevel(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Msg("Failed to get product stock level")
		handleProductError(c, err)
		return
	}

	response := &dto.StockLevelResponse{
		ProductID:      stockLevel.ProductID,
		CurrentStock:   stockLevel.CurrentStock,
		MinStockLevel:  stockLevel.MinStockLevel,
		MaxStockLevel:  stockLevel.MaxStockLevel,
		IsLowStock:     stockLevel.IsLowStock,
		IsOutOfStock:   stockLevel.IsOutOfStock,
		TrackInventory: stockLevel.TrackInventory,
		AllowBackorder: stockLevel.AllowBackorder,
		LastUpdated:    stockLevel.LastUpdated,
	}

	c.JSON(http.StatusOK, response)
}

// CheckProductAvailability checks if a product can fulfill a quantity
// @Summary Check product availability
// @Description Check if a product can fulfill a requested quantity
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param availability body dto.CheckAvailabilityRequest true "Availability check"
// @Success 200 {object} dto.AvailabilityResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{id}/check-availability [post]
func (h *ProductHandler) CheckProductAvailability(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	var req dto.CheckAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid availability check request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	availability, err := h.productService.CheckProductAvailability(c, id, req.Quantity)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", id).Int("quantity", req.Quantity).Msg("Failed to check product availability")
		handleProductError(c, err)
		return
	}

	response := &dto.AvailabilityResponse{
		ProductID:        availability.ProductID,
		RequestedQty:     availability.RequestedQty,
		Available:        availability.Available,
		Reason:           availability.Reason,
		CanFulfill:       availability.CanFulfill,
		BackorderAllowed: availability.BackorderAllowed,
	}

	c.JSON(http.StatusOK, response)
}

// Admin Operations (placeholder implementations)

// BulkProductOperation performs bulk operations on products
func (h *ProductHandler) BulkProductOperation(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Bulk product operations are not yet implemented",
	})
}

// ImportProducts imports products from data
func (h *ProductHandler) ImportProducts(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Product import is not yet implemented",
	})
}

// ExportProducts exports products to various formats
func (h *ProductHandler) ExportProducts(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Product export is not yet implemented",
	})
}

// GetProductStats retrieves product statistics
func (h *ProductHandler) GetProductStats(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Product statistics are not yet implemented",
	})
}

// GetLowStockProducts retrieves products with low stock
func (h *ProductHandler) GetLowStockProducts(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Low stock products retrieval is not yet implemented",
	})
}

// BulkInventoryAdjustment performs bulk inventory adjustments
func (h *ProductHandler) BulkInventoryAdjustment(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Bulk inventory adjustment is not yet implemented",
	})
}

// Helper Methods

// productToResponse converts a product entity to a response DTO
func (h *ProductHandler) productToResponse(p *entities.Product) *dto.ProductResponse {
	return &dto.ProductResponse{
		ID:                p.ID,
		SKU:               p.SKU,
		Name:              p.Name,
		Description:       p.Description,
		ShortDescription:  p.ShortDescription,
		CategoryID:        p.CategoryID,
		Price:             p.Price,
		Cost:              &p.Cost, // Expose cost in API (can be hidden based on requirements)
		Weight:            p.Weight,
		Dimensions:        p.Dimensions,
		Length:            p.Length,
		Width:             p.Width,
		Height:            p.Height,
		Volume:            p.Volume,
		Barcode:           p.Barcode,
		TrackInventory:    p.TrackInventory,
		StockQuantity:     p.StockQuantity,
		MinStockLevel:     p.MinStockLevel,
		MaxStockLevel:     p.MaxStockLevel,
		AllowBackorder:    p.AllowBackorder,
		RequiresShipping:  p.RequiresShipping,
		Taxable:           p.Taxable,
		TaxRate:           p.TaxRate,
		IsActive:          p.IsActive,
		IsFeatured:        p.IsFeatured,
		IsDigital:         p.IsDigital,
		DownloadURL:       p.DownloadURL,
		MaxDownloads:      p.MaxDownloads,
		ExpiryDays:        p.ExpiryDays,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}

// handleProductError handles product service errors
func handleProductError(c *gin.Context, err error) {
	switch {
	case err.Error() == "product not found":
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Product not found",
			Details: err.Error(),
		})
	case err.Error() == "product already exists":
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Product already exists",
			Details: err.Error(),
		})
	case err.Error() == "invalid category ID":
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid category ID",
			Details: err.Error(),
		})
	case err.Error() == "category not found":
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Category not found",
			Details: err.Error(),
		})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Details: err.Error(),
		})
	}
}