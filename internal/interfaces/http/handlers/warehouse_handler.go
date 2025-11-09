package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"erpgo/internal/interfaces/http/dto"
	"erpgo/pkg/errors"
)

// WarehouseHandler handles warehouse HTTP requests
type WarehouseHandler struct {
	warehouseService WarehouseService
	logger           zerolog.Logger
}

// WarehouseService defines the interface for warehouse service operations
type WarehouseService interface {
	CreateWarehouse(ctx *gin.Context, req *dto.CreateWarehouseRequest) (*dto.WarehouseResponse, error)
	GetWarehouse(ctx *gin.Context, id uuid.UUID) (*dto.WarehouseResponse, error)
	GetWarehouseByCode(ctx *gin.Context, code string) (*dto.WarehouseResponse, error)
	UpdateWarehouse(ctx *gin.Context, id uuid.UUID, req *dto.UpdateWarehouseRequest) (*dto.WarehouseResponse, error)
	DeleteWarehouse(ctx *gin.Context, id uuid.UUID) error
	ListWarehouses(ctx *gin.Context, req *dto.ListWarehousesRequest) (*dto.WarehouseListResponse, error)
	ActivateWarehouse(ctx *gin.Context, id uuid.UUID) (*dto.WarehouseResponse, error)
	DeactivateWarehouse(ctx *gin.Context, id uuid.UUID) (*dto.WarehouseResponse, error)
	AssignManager(ctx *gin.Context, id uuid.UUID, managerID uuid.UUID) (*dto.WarehouseResponse, error)
	RemoveManager(ctx *gin.Context, id uuid.UUID) (*dto.WarehouseResponse, error)
	GetWarehouseStats(ctx *gin.Context) (*dto.WarehouseStatsResponse, error)
}

// NewWarehouseHandler creates a new warehouse handler
func NewWarehouseHandler(warehouseService WarehouseService, logger zerolog.Logger) *WarehouseHandler {
	return &WarehouseHandler{
		warehouseService: warehouseService,
		logger:           logger,
	}
}

// Warehouse CRUD Operations

// CreateWarehouse creates a new warehouse
// @Summary Create warehouse
// @Description Create a new warehouse with validation
// @Tags warehouses
// @Accept json
// @Produce json
// @Param warehouse body dto.CreateWarehouseRequest true "Warehouse data"
// @Success 201 {object} dto.WarehouseResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/warehouses [post]
func (h *WarehouseHandler) CreateWarehouse(c *gin.Context) {
	var req dto.CreateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid warehouse creation request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	warehouse, err := h.warehouseService.CreateWarehouse(c, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create warehouse")
		handleWarehouseError(c, err)
		return
	}

	c.JSON(http.StatusCreated, warehouse)
}

// GetWarehouse retrieves a warehouse by ID
// @Summary Get warehouse
// @Description Get a warehouse by its ID
// @Tags warehouses
// @Accept json
// @Produce json
// @Param id path string true "Warehouse ID"
// @Success 200 {object} dto.WarehouseResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/warehouses/{id} [get]
func (h *WarehouseHandler) GetWarehouse(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid warehouse ID format",
		})
		return
	}

	warehouse, err := h.warehouseService.GetWarehouse(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("warehouse_id", idStr).Msg("Failed to get warehouse")
		handleWarehouseError(c, err)
		return
	}

	c.JSON(http.StatusOK, warehouse)
}

// GetWarehouseByCode retrieves a warehouse by code
// @Summary Get warehouse by code
// @Description Get a warehouse by its code
// @Tags warehouses
// @Accept json
// @Produce json
// @Param code path string true "Warehouse code"
// @Success 200 {object} dto.WarehouseResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/warehouses/code/{code} [get]
func (h *WarehouseHandler) GetWarehouseByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Warehouse code is required",
		})
		return
	}

	warehouse, err := h.warehouseService.GetWarehouseByCode(c, code)
	if err != nil {
		h.logger.Error().Err(err).Str("code", code).Msg("Failed to get warehouse by code")
		handleWarehouseError(c, err)
		return
	}

	c.JSON(http.StatusOK, warehouse)
}

// UpdateWarehouse updates a warehouse
// @Summary Update warehouse
// @Description Update an existing warehouse
// @Tags warehouses
// @Accept json
// @Produce json
// @Param id path string true "Warehouse ID"
// @Param warehouse body dto.UpdateWarehouseRequest true "Warehouse data"
// @Success 200 {object} dto.WarehouseResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/warehouses/{id} [put]
func (h *WarehouseHandler) UpdateWarehouse(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid warehouse ID format",
		})
		return
	}

	var req dto.UpdateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid warehouse update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	warehouse, err := h.warehouseService.UpdateWarehouse(c, id, &req)
	if err != nil {
		h.logger.Error().Err(err).Str("warehouse_id", idStr).Msg("Failed to update warehouse")
		handleWarehouseError(c, err)
		return
	}

	c.JSON(http.StatusOK, warehouse)
}

// DeleteWarehouse deletes a warehouse (soft delete)
// @Summary Delete warehouse
// @Description Delete a warehouse (soft delete)
// @Tags warehouses
// @Accept json
// @Produce json
// @Param id path string true "Warehouse ID"
// @Success 204
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/warehouses/{id} [delete]
func (h *WarehouseHandler) DeleteWarehouse(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid warehouse ID format",
		})
		return
	}

	err = h.warehouseService.DeleteWarehouse(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("warehouse_id", idStr).Msg("Failed to delete warehouse")
		handleWarehouseError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListWarehouses retrieves a paginated list of warehouses
// @Summary List warehouses
// @Description Get a paginated list of warehouses with optional filtering
// @Tags warehouses
// @Accept json
// @Produce json
// @Param search query string false "Search term"
// @Param code query string false "Warehouse code"
// @Param city query string false "City"
// @Param state query string false "State"
// @Param country query string false "Country"
// @Param type query string false "Warehouse type"
// @Param is_active query bool false "Filter by active status"
// @Param has_manager query bool false "Filter by manager assignment"
// @Param manager_id query string false "Manager ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param sort_by query string false "Sort field" default("created_at")
// @Param sort_order query string false "Sort order" Enums(asc,desc) default("desc")
// @Success 200 {object} dto.WarehouseListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/warehouses [get]
func (h *WarehouseHandler) ListWarehouses(c *gin.Context) {
	var req dto.ListWarehousesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid warehouse list request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Set defaults and validate
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	result, err := h.warehouseService.ListWarehouses(c, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list warehouses")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve warehouses",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Warehouse Operations

// ActivateWarehouse activates a warehouse
// @Summary Activate warehouse
// @Description Activate a warehouse
// @Tags warehouses
// @Accept json
// @Produce json
// @Param id path string true "Warehouse ID"
// @Success 200 {object} dto.WarehouseResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/warehouses/{id}/activate [post]
func (h *WarehouseHandler) ActivateWarehouse(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid warehouse ID format",
		})
		return
	}

	warehouse, err := h.warehouseService.ActivateWarehouse(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("warehouse_id", idStr).Msg("Failed to activate warehouse")
		handleWarehouseError(c, err)
		return
	}

	c.JSON(http.StatusOK, warehouse)
}

// DeactivateWarehouse deactivates a warehouse
// @Summary Deactivate warehouse
// @Description Deactivate a warehouse
// @Tags warehouses
// @Accept json
// @Produce json
// @Param id path string true "Warehouse ID"
// @Success 200 {object} dto.WarehouseResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/warehouses/{id}/deactivate [post]
func (h *WarehouseHandler) DeactivateWarehouse(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid warehouse ID format",
		})
		return
	}

	warehouse, err := h.warehouseService.DeactivateWarehouse(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("warehouse_id", idStr).Msg("Failed to deactivate warehouse")
		handleWarehouseError(c, err)
		return
	}

	c.JSON(http.StatusOK, warehouse)
}

// AssignManager assigns a manager to a warehouse
// @Summary Assign warehouse manager
// @Description Assign a manager to a warehouse
// @Tags warehouses
// @Accept json
// @Produce json
// @Param id path string true "Warehouse ID"
// @Param manager body map[string]string true "Manager ID"
// @Success 200 {object} dto.WarehouseResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/warehouses/{id}/manager [put]
func (h *WarehouseHandler) AssignManager(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid warehouse ID format",
		})
		return
	}

	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid manager assignment request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	managerIDStr, exists := req["manager_id"]
	if !exists {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Manager ID is required",
		})
		return
	}

	managerID, err := uuid.Parse(managerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid manager ID format",
		})
		return
	}

	warehouse, err := h.warehouseService.AssignManager(c, id, managerID)
	if err != nil {
		h.logger.Error().Err(err).Str("warehouse_id", idStr).Str("manager_id", managerIDStr).Msg("Failed to assign manager")
		handleWarehouseError(c, err)
		return
	}

	c.JSON(http.StatusOK, warehouse)
}

// RemoveManager removes the assigned manager from a warehouse
// @Summary Remove warehouse manager
// @Description Remove the assigned manager from a warehouse
// @Tags warehouses
// @Accept json
// @Produce json
// @Param id path string true "Warehouse ID"
// @Success 200 {object} dto.WarehouseResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/warehouses/{id}/manager [delete]
func (h *WarehouseHandler) RemoveManager(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid warehouse ID format",
		})
		return
	}

	warehouse, err := h.warehouseService.RemoveManager(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("warehouse_id", idStr).Msg("Failed to remove manager")
		handleWarehouseError(c, err)
		return
	}

	c.JSON(http.StatusOK, warehouse)
}

// GetWarehouseStats retrieves warehouse statistics
// @Summary Get warehouse statistics
// @Description Get warehouse statistics and metrics
// @Tags warehouses
// @Accept json
// @Produce json
// @Success 200 {object} dto.WarehouseStatsResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/warehouses/stats [get]
func (h *WarehouseHandler) GetWarehouseStats(c *gin.Context) {
	stats, err := h.warehouseService.GetWarehouseStats(c)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get warehouse statistics")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve warehouse statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Utility Methods

// SearchWarehouses searches warehouses by query
// @Summary Search warehouses
// @Description Search warehouses by query string
// @Tags warehouses
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Result limit" default(20)
// @Success 200 {object} dto.WarehouseListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/warehouses/search [get]
func (h *WarehouseHandler) SearchWarehouses(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Search query is required",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	req := &dto.ListWarehousesRequest{
		Search:    query,
		Page:      1,
		Limit:     limit,
		SortBy:    "name",
		SortOrder: "asc",
	}

	result, err := h.warehouseService.ListWarehouses(c, req)
	if err != nil {
		h.logger.Error().Err(err).Str("query", query).Msg("Failed to search warehouses")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to search warehouses",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// handleWarehouseError handles warehouse service errors
func handleWarehouseError(c *gin.Context, err error) {
	switch {
	case errors.IsNotFoundError(err):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Warehouse not found",
			Details: err.Error(),
		})
	case errors.IsConflictError(err):
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Warehouse already exists",
			Details: err.Error(),
		})
	case errors.IsValidationError(err):
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Validation error",
			Details: err.Error(),
		})
	case errors.IsUnauthorizedError(err):
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Details: err.Error(),
		})
	case errors.IsForbiddenError(err):
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Error:   "Forbidden",
			Details: err.Error(),
		})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Details: err.Error(),
		})
	}
}