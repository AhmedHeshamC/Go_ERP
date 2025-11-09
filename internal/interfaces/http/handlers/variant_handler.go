package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/internal/application/services/product"
	"erpgo/internal/domain/products/entities"
	"erpgo/internal/interfaces/http/dto"
)

// VariantHandler handles product variant HTTP requests
type VariantHandler struct {
	productService product.Service
	logger         zerolog.Logger
}

// NewVariantHandler creates a new variant handler
func NewVariantHandler(productService product.Service, logger zerolog.Logger) *VariantHandler {
	return &VariantHandler{
		productService: productService,
		logger:         logger,
	}
}

// Variant CRUD Operations

// CreateProductVariant creates a new product variant
// @Summary Create product variant
// @Description Create a new product variant with validation and SKU generation
// @Tags variants
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param variant body dto.CreateProductVariantRequest true "Variant data"
// @Success 201 {object} dto.ProductVariantResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{product_id}/variants [post]
func (h *VariantHandler) CreateProductVariant(c *gin.Context) {
	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	var req dto.CreateProductVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid variant creation request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Set product ID from path
	req.ProductID = productID

	// Convert to service request
	serviceReq := &product.CreateProductVariantRequest{
		ProductID:      req.ProductID,
		SKU:            req.SKU,
		Name:           req.Name,
		Price:          req.Price,
		Cost:           req.Cost,
		Weight:         req.Weight,
		Barcode:        req.Barcode,
		TrackInventory: req.TrackInventory,
		StockQuantity:  req.StockQuantity,
		MinStockLevel:  req.MinStockLevel,
		MaxStockLevel:  req.MaxStockLevel,
		AllowBackorder: req.AllowBackorder,
		IsActive:       req.IsActive,
	}

	// Convert attributes
	if len(req.Attributes) > 0 {
		serviceReq.Attributes = make([]product.VariantAttributeRequest, len(req.Attributes))
		for i, attr := range req.Attributes {
			serviceReq.Attributes[i] = product.VariantAttributeRequest{
				Name:  attr.Name,
				Value: attr.Value,
			}
		}
	}

	// Convert images
	if len(req.Images) > 0 {
		serviceReq.Images = make([]product.VariantImageRequest, len(req.Images))
		for i, img := range req.Images {
			serviceReq.Images[i] = product.VariantImageRequest{
				URL:       img.URL,
				Alt:       img.Alt,
				SortOrder: img.SortOrder,
				IsMain:    img.IsMain,
			}
		}
	}

	variant, err := h.productService.CreateProductVariant(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", productID).Msg("Failed to create product variant")
		handleVariantError(c, err)
		return
	}

	response := h.variantToResponse(variant)
	c.JSON(http.StatusCreated, response)
}

// GetProductVariant retrieves a product variant by ID
// @Summary Get product variant
// @Description Get a product variant by its ID
// @Tags variants
// @Accept json
// @Produce json
// @Param id path string true "Variant ID"
// @Success 200 {object} dto.ProductVariantResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/variants/{id} [get]
func (h *VariantHandler) GetProductVariant(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Variant ID is required",
		})
		return
	}

	variant, err := h.productService.GetProductVariant(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("variant_id", id).Msg("Failed to get product variant")
		handleVariantError(c, err)
		return
	}

	response := h.variantToResponse(variant)
	c.JSON(http.StatusOK, response)
}

// GetProductVariantBySKU retrieves a product variant by SKU
// @Summary Get product variant by SKU
// @Description Get a product variant by its SKU
// @Tags variants
// @Accept json
// @Produce json
// @Param sku path string true "Variant SKU"
// @Success 200 {object} dto.ProductVariantResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/variants/sku/{sku} [get]
func (h *VariantHandler) GetProductVariantBySKU(c *gin.Context) {
	sku := c.Param("sku")
	if sku == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Variant SKU is required",
		})
		return
	}

	variant, err := h.productService.GetProductVariantBySKU(c, sku)
	if err != nil {
		h.logger.Error().Err(err).Str("sku", sku).Msg("Failed to get product variant by SKU")
		handleVariantError(c, err)
		return
	}

	response := h.variantToResponse(variant)
	c.JSON(http.StatusOK, response)
}

// UpdateProductVariant updates a product variant
// @Summary Update product variant
// @Description Update an existing product variant
// @Tags variants
// @Accept json
// @Produce json
// @Param id path string true "Variant ID"
// @Param variant body dto.UpdateProductVariantRequest true "Variant data"
// @Success 200 {object} dto.ProductVariantResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/variants/{id} [put]
func (h *VariantHandler) UpdateProductVariant(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Variant ID is required",
		})
		return
	}

	var req dto.UpdateProductVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid variant update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &product.UpdateProductVariantRequest{
		Name:           req.Name,
		Price:          req.Price,
		Cost:           req.Cost,
		Weight:         req.Weight,
		Barcode:        req.Barcode,
		TrackInventory: req.TrackInventory,
		MinStockLevel:  req.MinStockLevel,
		MaxStockLevel:  req.MaxStockLevel,
		AllowBackorder: req.AllowBackorder,
		IsActive:       req.IsActive,
	}

	// Convert attributes
	if len(req.Attributes) > 0 {
		serviceReq.Attributes = make([]product.VariantAttributeRequest, len(req.Attributes))
		for i, attr := range req.Attributes {
			serviceReq.Attributes[i] = product.VariantAttributeRequest{
				Name:  attr.Name,
				Value: attr.Value,
			}
		}
	}

	updatedVariant, err := h.productService.UpdateProductVariant(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("variant_id", id).Msg("Failed to update product variant")
		handleVariantError(c, err)
		return
	}

	response := h.variantToResponse(updatedVariant)
	c.JSON(http.StatusOK, response)
}

// DeleteProductVariant deletes a product variant
// @Summary Delete product variant
// @Description Delete a product variant
// @Tags variants
// @Accept json
// @Produce json
// @Param id path string true "Variant ID"
// @Success 204
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/variants/{id} [delete]
func (h *VariantHandler) DeleteProductVariant(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Variant ID is required",
		})
		return
	}

	err := h.productService.DeleteProductVariant(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("variant_id", id).Msg("Failed to delete product variant")
		handleVariantError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListProductVariants retrieves a paginated list of product variants
// @Summary List product variants
// @Description Get a paginated list of product variants for a specific product
// @Tags variants
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param search query string false "Search term"
// @Param min_price query number false "Minimum price"
// @Param max_price query number false "Maximum price"
// @Param is_active query bool false "Filter by active status"
// @Param in_stock query bool false "Filter by in stock status"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param sort_by query string false "Sort field" default("created_at")
// @Param sort_order query string false "Sort order" Enums(asc,desc) default("desc")
// @Success 200 {object} dto.ProductVariantListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{product_id}/variants [get]
func (h *VariantHandler) ListProductVariants(c *gin.Context) {
	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	var req dto.ListProductVariantsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid variant list request")
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
	serviceReq := &product.ListProductVariantsRequest{
		Search:    req.Search,
		MinPrice:  req.MinPrice,
		MaxPrice:  req.MaxPrice,
		IsActive:  req.IsActive,
		Page:      req.Page,
		Limit:     req.Limit,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	result, err := h.productService.ListProductVariants(c, productID, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", productID).Msg("Failed to list product variants")
		handleVariantError(c, err)
		return
	}

	// Convert to response DTO
	variants := make([]*dto.ProductVariantResponse, len(result.Variants))
	for i, v := range result.Variants {
		variants[i] = h.variantToResponse(v)
	}

	response := &dto.ProductVariantListResponse{
		Variants: variants,
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

// Variant Inventory Operations

// UpdateVariantStock updates variant stock quantity
// @Summary Update variant stock
// @Description Update the stock quantity for a product variant
// @Tags variants
// @Accept json
// @Produce json
// @Param id path string true "Variant ID"
// @Param stock body dto.UpdateStockRequest true "Stock data"
// @Success 200 {object} dto.ProductVariantResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/variants/{id}/stock [put]
func (h *VariantHandler) UpdateVariantStock(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Variant ID is required",
		})
		return
	}

	var req dto.UpdateStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid variant stock update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// This would require extending the service to support variant stock operations
	// For now, we'll return an error indicating this is not yet implemented
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Variant stock operations are not yet implemented",
	})
}

// AdjustVariantStock adjusts variant stock quantity
// @Summary Adjust variant stock
// @Description Adjust the stock quantity for a product variant
// @Tags variants
// @Accept json
// @Produce json
// @Param id path string true "Variant ID"
// @Param adjustment body dto.AdjustStockRequest true "Adjustment data"
// @Success 200 {object} dto.ProductVariantResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/variants/{id}/stock/adjust [post]
func (h *VariantHandler) AdjustVariantStock(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Variant ID is required",
		})
		return
	}

	var req dto.AdjustStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid variant stock adjustment request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// This would require extending the service to support variant stock operations
	// For now, we'll return an error indicating this is not yet implemented
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Variant stock operations are not yet implemented",
	})
}

// Helper Methods

// variantToResponse converts a variant entity to a response DTO
func (h *VariantHandler) variantToResponse(v *entities.ProductVariant) *dto.ProductVariantResponse {
	return &dto.ProductVariantResponse{
		ID:             v.ID,
		ProductID:      v.ProductID,
		SKU:            v.SKU,
		Name:           v.Name,
		Price:          v.Price,
		Cost:           &v.Cost, // Expose cost in API (can be hidden based on requirements)
		Weight:         v.Weight,
		Barcode:        v.Barcode,
		TrackInventory: v.TrackInventory,
		StockQuantity:  v.StockQuantity,
		MinStockLevel:  v.MinStockLevel,
		MaxStockLevel:  v.MaxStockLevel,
		AllowBackorder: v.AllowBackorder,
		IsActive:       v.IsActive,
		CreatedAt:      v.CreatedAt,
		UpdatedAt:      v.UpdatedAt,
		// Note: Attributes and Images would need additional service methods to fetch
	}
}

// handleVariantError handles variant service errors
func handleVariantError(c *gin.Context, err error) {
	switch {
	case err.Error() == "product not found":
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Product not found",
			Details: err.Error(),
		})
	case err.Error() == "product variant not found":
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Product variant not found",
			Details: err.Error(),
		})
	case err.Error() == "product variant already exists":
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Product variant already exists",
			Details: err.Error(),
		})
	case err.Error() == "invalid product ID":
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid product ID",
			Details: err.Error(),
		})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Details: err.Error(),
		})
	}
}