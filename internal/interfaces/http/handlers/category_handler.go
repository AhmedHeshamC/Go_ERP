package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/internal/application/services/product"
	"erpgo/internal/domain/products/entities"
	"erpgo/internal/interfaces/http/dto"
)

// CategoryHandler handles category HTTP requests
type CategoryHandler struct {
	productService product.Service
	logger         zerolog.Logger
}

// NewCategoryHandler creates a new category handler
func NewCategoryHandler(productService product.Service, logger zerolog.Logger) *CategoryHandler {
	return &CategoryHandler{
		productService: productService,
		logger:         logger,
	}
}

// Category CRUD Operations

// CreateCategory creates a new category
// @Summary Create category
// @Description Create a new product category with hierarchy validation
// @Tags categories
// @Accept json
// @Produce json
// @Param category body dto.CreateCategoryRequest true "Category data"
// @Success 201 {object} dto.CategoryResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/categories [post]
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req dto.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid category creation request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &product.CreateCategoryRequest{
		Name:           req.Name,
		Description:    req.Description,
		ParentID:       nil,
		IsActive:       req.IsActive,
		SortOrder:      req.SortOrder,
		SEOTitle:       req.SEOTitle,
		SEODescription: req.SEODescription,
		SEOKeywords:    req.SEOKeywords,
	}

	// Set ParentID if provided
	if req.ParentID != "" {
		serviceReq.ParentID = &req.ParentID
	}

	category, err := h.productService.CreateCategory(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create category")
		handleCategoryError(c, err)
		return
	}

	response := h.categoryToResponse(category)
	c.JSON(http.StatusCreated, response)
}

// GetCategory retrieves a category by ID
// @Summary Get category
// @Description Get a category by its ID
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} dto.CategoryResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/categories/{id} [get]
func (h *CategoryHandler) GetCategory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Category ID is required",
		})
		return
	}

	category, err := h.productService.GetCategory(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("category_id", id).Msg("Failed to get category")
		handleCategoryError(c, err)
		return
	}

	response := h.categoryToResponse(category)
	c.JSON(http.StatusOK, response)
}

// GetCategoryByPath retrieves a category by path
// @Summary Get category by path
// @Description Get a category by its hierarchical path
// @Tags categories
// @Accept json
// @Produce json
// @Param path path string true "Category path"
// @Success 200 {object} dto.CategoryResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/categories/path/{path} [get]
func (h *CategoryHandler) GetCategoryByPath(c *gin.Context) {
	path := c.Param("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Category path is required",
		})
		return
	}

	category, err := h.productService.GetCategoryByPath(c, path)
	if err != nil {
		h.logger.Error().Err(err).Str("path", path).Msg("Failed to get category by path")
		handleCategoryError(c, err)
		return
	}

	response := h.categoryToResponse(category)
	c.JSON(http.StatusOK, response)
}

// UpdateCategory updates a category
// @Summary Update category
// @Description Update an existing category
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param category body dto.UpdateCategoryRequest true "Category data"
// @Success 200 {object} dto.CategoryResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Category ID is required",
		})
		return
	}

	var req dto.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid category update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &product.UpdateCategoryRequest{
		Name:           req.Name,
		Description:    req.Description,
		IsActive:       req.IsActive,
		SortOrder:      req.SortOrder,
		SEOTitle:       req.SEOTitle,
		SEODescription: req.SEODescription,
		SEOKeywords:    req.SEOKeywords,
	}

	updatedCategory, err := h.productService.UpdateCategory(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("category_id", id).Msg("Failed to update category")
		handleCategoryError(c, err)
		return
	}

	response := h.categoryToResponse(updatedCategory)
	c.JSON(http.StatusOK, response)
}

// DeleteCategory deletes a category
// @Summary Delete category
// @Description Delete a category (only if it has no products or children)
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Success 204
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Category ID is required",
		})
		return
	}

	err := h.productService.DeleteCategory(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("category_id", id).Msg("Failed to delete category")
		handleCategoryError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListCategories retrieves a paginated list of categories
// @Summary List categories
// @Description Get a paginated list of categories with optional filtering
// @Tags categories
// @Accept json
// @Produce json
// @Param search query string false "Search term"
// @Param parent_id query string false "Parent category ID"
// @Param is_active query bool false "Filter by active status"
// @Param level query int false "Filter by level"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param sort_by query string false "Sort field" default("sort_order")
// @Param sort_order query string false "Sort order" Enums(asc,desc) default("asc")
// @Success 200 {object} dto.CategoryListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/categories [get]
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	var req dto.ListCategoriesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid category list request")
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
	serviceReq := &product.ListCategoriesRequest{
		Search:    req.Search,
		ParentID:  nil,
		IsActive:  req.IsActive,
		Level:     req.Level,
		Page:      req.Page,
		Limit:     req.Limit,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	// Set ParentID if provided
	if req.ParentID != "" {
		serviceReq.ParentID = &req.ParentID
	}

	result, err := h.productService.ListCategories(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list categories")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve categories",
		})
		return
	}

	// Convert to response DTO
	categories := make([]*dto.CategoryResponse, len(result.Categories))
	for i, cat := range result.Categories {
		categories[i] = h.categoryToResponse(cat)
	}

	response := &dto.CategoryListResponse{
		Categories: categories,
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

// GetCategoryTree retrieves the complete category tree
// @Summary Get category tree
// @Description Get the complete hierarchical category tree
// @Tags categories
// @Accept json
// @Produce json
// @Success 200 {object} dto.CategoryTreeResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/categories/tree [get]
func (h *CategoryHandler) GetCategoryTree(c *gin.Context) {
	result, err := h.productService.GetCategoryTree(c)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get category tree")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve category tree",
		})
		return
	}

	// Convert to response DTO
	tree := make([]*dto.CategoryTreeNode, len(result.Tree))
	for i, node := range result.Tree {
		tree[i] = h.categoryNodeToResponse(node)
	}

	response := &dto.CategoryTreeResponse{
		Tree: tree,
	}

	c.JSON(http.StatusOK, response)
}

// GetCategoryDescendants retrieves all descendant categories
// @Summary Get category descendants
// @Description Get all descendant categories for a given category
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {array} dto.CategoryResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/categories/{id}/descendants [get]
func (h *CategoryHandler) GetCategoryDescendants(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Category ID is required",
		})
		return
	}

	descendants, err := h.productService.GetCategoryDescendants(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("category_id", id).Msg("Failed to get category descendants")
		handleCategoryError(c, err)
		return
	}

	// Convert to response DTO
	response := make([]*dto.CategoryResponse, len(descendants))
	for i, cat := range descendants {
		response[i] = h.categoryToResponse(cat)
	}

	c.JSON(http.StatusOK, response)
}

// GetCategoryAncestors retrieves all ancestor categories
// @Summary Get category ancestors
// @Description Get all ancestor categories for a given category
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {array} dto.CategoryResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/categories/{id}/ancestors [get]
func (h *CategoryHandler) GetCategoryAncestors(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Category ID is required",
		})
		return
	}

	ancestors, err := h.productService.GetCategoryAncestors(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("category_id", id).Msg("Failed to get category ancestors")
		handleCategoryError(c, err)
		return
	}

	// Convert to response DTO
	response := make([]*dto.CategoryResponse, len(ancestors))
	for i, cat := range ancestors {
		response[i] = h.categoryToResponse(cat)
	}

	c.JSON(http.StatusOK, response)
}

// MoveCategory moves a category to a new parent
// @Summary Move category
// @Description Move a category to a new parent category
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param move body dto.MoveCategoryRequest true "Move request"
// @Success 200 {object} dto.CategoryResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/categories/{id}/move [post]
func (h *CategoryHandler) MoveCategory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Category ID is required",
		})
		return
	}

	var req dto.MoveCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid category move request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	err := h.productService.MoveCategory(c, id, req.ParentID)
	if err != nil {
		h.logger.Error().Err(err).Str("category_id", id).Msg("Failed to move category")
		handleCategoryError(c, err)
		return
	}

	// Return the updated category
	category, err := h.productService.GetCategory(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("category_id", id).Msg("Failed to get updated category")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve updated category",
		})
		return
	}

	response := h.categoryToResponse(category)
	c.JSON(http.StatusOK, response)
}

// Admin Category Operations (placeholder implementations)

// BulkCategoryOperation performs bulk operations on categories
func (h *CategoryHandler) BulkCategoryOperation(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Bulk category operations are not yet implemented",
	})
}

// ImportCategories imports categories from data
func (h *CategoryHandler) ImportCategories(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Category import is not yet implemented",
	})
}

// ExportCategories exports categories to various formats
func (h *CategoryHandler) ExportCategories(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Category export is not yet implemented",
	})
}

// ReorderCategories reorders categories
func (h *CategoryHandler) ReorderCategories(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Category reordering is not yet implemented",
	})
}

// Helper Methods

// categoryToResponse converts a category entity to a response DTO
func (h *CategoryHandler) categoryToResponse(cat *entities.ProductCategory) *dto.CategoryResponse {
	return &dto.CategoryResponse{
		ID:            cat.ID,
		Name:          cat.Name,
		Description:   cat.Description,
		ParentID:      cat.ParentID,
		Path:          cat.Path,
		Level:         cat.Level,
		IsActive:      cat.IsActive,
		SortOrder:     cat.SortOrder,
		SEOTitle:      cat.SEOTitle,
		SEODescription: cat.SEODescription,
		SEOKeywords:   cat.SEOKeywords,
		CreatedAt:     cat.CreatedAt,
		UpdatedAt:     cat.UpdatedAt,
	}
}

// categoryNodeToResponse converts a category tree node to a response DTO
func (h *CategoryHandler) categoryNodeToResponse(node *product.CategoryNode) *dto.CategoryTreeNode {
	categoryResponse := h.categoryToResponse(node.ProductCategory)

	children := make([]*dto.CategoryTreeNode, len(node.Children))
	for i, child := range node.Children {
		children[i] = h.categoryNodeToResponse(child)
	}

	return &dto.CategoryTreeNode{
		CategoryResponse: categoryResponse,
		Children:         children,
	}
}

// handleCategoryError handles category service errors
func handleCategoryError(c *gin.Context, err error) {
	switch {
	case err.Error() == "category not found":
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Category not found",
			Details: err.Error(),
		})
	case err.Error() == "category already exists":
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Category already exists",
			Details: err.Error(),
		})
	case err.Error() == "invalid parent ID":
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid parent ID",
			Details: err.Error(),
		})
	case err.Error() == "category has products":
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Cannot delete category with products",
			Details: err.Error(),
		})
	case err.Error() == "category has children":
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Cannot delete category with child categories",
			Details: err.Error(),
		})
	case err.Error() == "invalid category hierarchy":
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid category hierarchy",
			Details: err.Error(),
		})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Details: err.Error(),
		})
	}
}