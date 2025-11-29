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

// InventoryHandler handles inventory HTTP requests
type InventoryHandler struct {
	inventoryService InventoryService
	logger           zerolog.Logger
}

// InventoryService defines the interface for inventory service operations
type InventoryService interface {
	AdjustInventory(ctx *gin.Context, req *dto.AdjustInventoryRequest) (*dto.InventoryTransactionResponse, error)
	ReserveInventory(ctx *gin.Context, req *dto.ReserveInventoryRequest) (*dto.InventoryResponse, error)
	ReleaseInventory(ctx *gin.Context, req *dto.ReleaseInventoryRequest) (*dto.InventoryResponse, error)
	TransferInventory(ctx *gin.Context, req *dto.TransferInventoryRequest) (*dto.InventoryTransactionResponse, error)
	ListInventory(ctx *gin.Context, req *dto.ListInventoryRequest) (*dto.InventoryListResponse, error)
	GetInventoryByProductAndWarehouse(ctx *gin.Context, productID, warehouseID uuid.UUID) (*dto.InventoryResponse, error)
	GetInventoryStats(ctx *gin.Context) (*dto.InventoryStatsResponse, error)
	GetLowStockItems(ctx *gin.Context, warehouseID *uuid.UUID) (*dto.InventoryListResponse, error)
	BulkInventoryAdjustment(ctx *gin.Context, req *dto.BulkInventoryAdjustmentRequest) (*dto.BulkInventoryOperationResponse, error)
	CheckInventoryAvailability(ctx *gin.Context, productID, warehouseID uuid.UUID, quantity int) (*dto.AvailabilityResponse, error)
	GetInventoryHistory(ctx *gin.Context, productID, warehouseID uuid.UUID, limit int) ([]*dto.InventoryTransactionResponse, error)
}

// NewInventoryHandler creates a new inventory handler
func NewInventoryHandler(inventoryService InventoryService, logger zerolog.Logger) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
		logger:           logger,
	}
}

// Inventory Stock Management

// AdjustInventory adjusts inventory stock quantity
// @Summary Adjust inventory
// @Description Adjust the stock quantity for a product in a warehouse
// @Tags inventory
// @Accept json
// @Produce json
// @Param adjustment body dto.AdjustInventoryRequest true "Adjustment data"
// @Success 201 {object} dto.InventoryTransactionResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/adjust [post]
func (h *InventoryHandler) AdjustInventory(c *gin.Context) {
	var req dto.AdjustInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid inventory adjustment request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	transaction, err := h.inventoryService.AdjustInventory(c, &req)
	if err != nil {
		h.logger.Error().Err(err).
			Str("product_id", req.ProductID.String()).
			Str("warehouse_id", req.WarehouseID.String()).
			Int("adjustment", req.Adjustment).
			Msg("Failed to adjust inventory")
		handleInventoryError(c, err)
		return
	}

	c.JSON(http.StatusCreated, transaction)
}

// ReserveInventory reserves inventory stock
// @Summary Reserve inventory
// @Description Reserve a quantity of product in a warehouse
// @Tags inventory
// @Accept json
// @Produce json
// @Param reservation body dto.ReserveInventoryRequest true "Reservation data"
// @Success 201 {object} dto.InventoryResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/reserve [post]
func (h *InventoryHandler) ReserveInventory(c *gin.Context) {
	var req dto.ReserveInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid inventory reservation request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	inventory, err := h.inventoryService.ReserveInventory(c, &req)
	if err != nil {
		h.logger.Error().Err(err).
			Str("product_id", req.ProductID.String()).
			Str("warehouse_id", req.WarehouseID.String()).
			Int("quantity", req.Quantity).
			Msg("Failed to reserve inventory")
		handleInventoryError(c, err)
		return
	}

	c.JSON(http.StatusCreated, inventory)
}

// ReleaseInventory releases reserved inventory
// @Summary Release inventory
// @Description Release a quantity of reserved inventory
// @Tags inventory
// @Accept json
// @Produce json
// @Param release body dto.ReleaseInventoryRequest true "Release data"
// @Success 200 {object} dto.InventoryResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/release [post]
func (h *InventoryHandler) ReleaseInventory(c *gin.Context) {
	var req dto.ReleaseInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid inventory release request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	inventory, err := h.inventoryService.ReleaseInventory(c, &req)
	if err != nil {
		h.logger.Error().Err(err).
			Str("reservation_id", req.ReservationID.String()).
			Int("quantity", req.Quantity).
			Msg("Failed to release inventory")
		handleInventoryError(c, err)
		return
	}

	c.JSON(http.StatusOK, inventory)
}

// TransferInventory transfers inventory between warehouses
// @Summary Transfer inventory
// @Description Transfer inventory from one warehouse to another
// @Tags inventory
// @Accept json
// @Produce json
// @Param transfer body dto.TransferInventoryRequest true "Transfer data"
// @Success 201 {object} dto.InventoryTransactionResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/transfer [post]
func (h *InventoryHandler) TransferInventory(c *gin.Context) {
	var req dto.TransferInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid inventory transfer request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	transaction, err := h.inventoryService.TransferInventory(c, &req)
	if err != nil {
		h.logger.Error().Err(err).
			Str("product_id", req.ProductID.String()).
			Str("from_warehouse_id", req.FromWarehouseID.String()).
			Str("to_warehouse_id", req.ToWarehouseID.String()).
			Int("quantity", req.Quantity).
			Msg("Failed to transfer inventory")
		handleInventoryError(c, err)
		return
	}

	c.JSON(http.StatusCreated, transaction)
}

// Inventory Query Operations

// ListInventory retrieves a paginated list of inventory items
// @Summary List inventory
// @Description Get a paginated list of inventory items with optional filtering
// @Tags inventory
// @Accept json
// @Produce json
// @Param product_id query string false "Product ID"
// @Param warehouse_id query string false "Warehouse ID"
// @Param sku query string false "Product SKU"
// @Param product_name query string false "Product name"
// @Param warehouse_code query string false "Warehouse code"
// @Param city query string false "City"
// @Param state query string false "State"
// @Param country query string false "Country"
// @Param in_stock query bool false "Filter by in stock status"
// @Param low_stock query bool false "Filter by low stock status"
// @Param out_of_stock query bool false "Filter by out of stock status"
// @Param has_reservations query bool false "Filter by reservation status"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param sort_by query string false "Sort field" default("created_at")
// @Param sort_order query string false "Sort order" Enums(asc,desc) default("desc")
// @Success 200 {object} dto.InventoryListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory [get]
func (h *InventoryHandler) ListInventory(c *gin.Context) {
	var req dto.ListInventoryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid inventory list request")
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

	result, err := h.inventoryService.ListInventory(c, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list inventory")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve inventory",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetInventory retrieves inventory for a specific product and warehouse
// @Summary Get inventory
// @Description Get inventory for a specific product and warehouse combination
// @Tags inventory
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param warehouse_id path string true "Warehouse ID"
// @Success 200 {object} dto.InventoryResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/product/{product_id}/warehouse/{warehouse_id} [get]
func (h *InventoryHandler) GetInventory(c *gin.Context) {
	productIDStr := c.Param("product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid product ID format",
		})
		return
	}

	warehouseIDStr := c.Param("warehouse_id")
	warehouseID, err := uuid.Parse(warehouseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid warehouse ID format",
		})
		return
	}

	inventory, err := h.inventoryService.GetInventoryByProductAndWarehouse(c, productID, warehouseID)
	if err != nil {
		h.logger.Error().Err(err).
			Str("product_id", productIDStr).
			Str("warehouse_id", warehouseIDStr).
			Msg("Failed to get inventory")
		handleInventoryError(c, err)
		return
	}

	c.JSON(http.StatusOK, inventory)
}

// GetInventoryStats retrieves inventory statistics
// @Summary Get inventory statistics
// @Description Get inventory statistics and metrics
// @Tags inventory
// @Accept json
// @Produce json
// @Success 200 {object} dto.InventoryStatsResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/stats [get]
func (h *InventoryHandler) GetInventoryStats(c *gin.Context) {
	stats, err := h.inventoryService.GetInventoryStats(c)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get inventory statistics")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve inventory statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetLowStockItems retrieves items with low stock levels
// @Summary Get low stock items
// @Description Get items that are below their minimum stock level
// @Tags inventory
// @Accept json
// @Produce json
// @Param warehouse_id query string false "Warehouse ID filter"
// @Success 200 {object} dto.InventoryListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/low-stock [get]
func (h *InventoryHandler) GetLowStockItems(c *gin.Context) {
	warehouseIDStr := c.Query("warehouse_id")
	var warehouseID *uuid.UUID
	if warehouseIDStr != "" {
		id, err := uuid.Parse(warehouseIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error: "Invalid warehouse ID format",
			})
			return
		}
		warehouseID = &id
	}

	items, err := h.inventoryService.GetLowStockItems(c, warehouseID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get low stock items")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve low stock items",
		})
		return
	}

	c.JSON(http.StatusOK, items)
}

// Bulk Operations

// BulkInventoryAdjustment performs bulk inventory adjustments
// @Summary Bulk inventory adjustment
// @Description Perform multiple inventory adjustments in a single operation
// @Tags inventory
// @Accept json
// @Produce json
// @Param adjustment body dto.BulkInventoryAdjustmentRequest true "Bulk adjustment data"
// @Success 201 {object} dto.BulkInventoryOperationResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/bulk-adjust [post]
func (h *InventoryHandler) BulkInventoryAdjustment(c *gin.Context) {
	var req dto.BulkInventoryAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid bulk inventory adjustment request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	result, err := h.inventoryService.BulkInventoryAdjustment(c, &req)
	if err != nil {
		h.logger.Error().Err(err).Int("adjustment_count", len(req.Adjustments)).Msg("Failed to perform bulk inventory adjustment")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to perform bulk inventory adjustment",
		})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// Utility Operations

// CheckInventoryAvailability checks if a product can fulfill a quantity in a warehouse
// @Summary Check inventory availability
// @Description Check if a product can fulfill a requested quantity in a warehouse
// @Tags inventory
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param warehouse_id path string true "Warehouse ID"
// @Param quantity query int true "Quantity to check"
// @Success 200 {object} dto.AvailabilityResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/product/{product_id}/warehouse/{warehouse_id}/check-availability [get]
func (h *InventoryHandler) CheckInventoryAvailability(c *gin.Context) {
	productIDStr := c.Param("product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid product ID format",
		})
		return
	}

	warehouseIDStr := c.Param("warehouse_id")
	warehouseID, err := uuid.Parse(warehouseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid warehouse ID format",
		})
		return
	}

	quantityStr := c.Query("quantity")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity <= 0 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Valid quantity parameter is required",
		})
		return
	}

	availability, err := h.inventoryService.CheckInventoryAvailability(c, productID, warehouseID, quantity)
	if err != nil {
		h.logger.Error().Err(err).
			Str("product_id", productIDStr).
			Str("warehouse_id", warehouseIDStr).
			Int("quantity", quantity).
			Msg("Failed to check inventory availability")
		handleInventoryError(c, err)
		return
	}

	c.JSON(http.StatusOK, availability)
}

// GetInventoryHistory retrieves inventory transaction history
// @Summary Get inventory history
// @Description Get transaction history for a product in a warehouse
// @Tags inventory
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param warehouse_id path string true "Warehouse ID"
// @Param limit query int false "Result limit" default(50)
// @Success 200 {array} dto.InventoryTransactionResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/product/{product_id}/warehouse/{warehouse_id}/history [get]
func (h *InventoryHandler) GetInventoryHistory(c *gin.Context) {
	productIDStr := c.Param("product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid product ID format",
		})
		return
	}

	warehouseIDStr := c.Param("warehouse_id")
	warehouseID, err := uuid.Parse(warehouseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid warehouse ID format",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	history, err := h.inventoryService.GetInventoryHistory(c, productID, warehouseID, limit)
	if err != nil {
		h.logger.Error().Err(err).
			Str("product_id", productIDStr).
			Str("warehouse_id", warehouseIDStr).
			Int("limit", limit).
			Msg("Failed to get inventory history")
		handleInventoryError(c, err)
		return
	}

	c.JSON(http.StatusOK, history)
}

// handleInventoryError handles inventory service errors
func handleInventoryError(c *gin.Context, err error) {
	switch {
	case errors.IsNotFoundError(err):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Inventory not found",
			Details: err.Error(),
		})
	case errors.IsConflictError(err):
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Inventory conflict",
			Details: err.Error(),
		})
	case errors.IsValidationError(err):
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Validation error",
			Details: err.Error(),
		})
	case errors.IsInsufficientStockError(err):
		c.Status(http.StatusConflict)
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Insufficient stock",
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
