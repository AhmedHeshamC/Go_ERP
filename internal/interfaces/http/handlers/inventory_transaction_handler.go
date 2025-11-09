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

// InventoryTransactionHandler handles inventory transaction HTTP requests
type InventoryTransactionHandler struct {
	transactionService InventoryTransactionService
	logger             zerolog.Logger
}

// InventoryTransactionService defines the interface for inventory transaction service operations
type InventoryTransactionService interface {
	GetTransaction(ctx *gin.Context, id uuid.UUID) (*dto.InventoryTransactionResponse, error)
	ListTransactions(ctx *gin.Context, req *dto.ListInventoryTransactionsRequest) (*dto.InventoryTransactionListResponse, error)
	ApproveTransaction(ctx *gin.Context, id uuid.UUID, req *dto.ApproveTransactionRequest) (*dto.InventoryTransactionResponse, error)
	RejectTransaction(ctx *gin.Context, id uuid.UUID, reason string) (*dto.InventoryTransactionResponse, error)
	GetTransactionStats(ctx *gin.Context, warehouseID *uuid.UUID, productID *uuid.UUID) (*dto.TransactionStatsResponse, error)
	GetPendingApprovals(ctx *gin.Context, warehouseID *uuid.UUID) (*dto.InventoryTransactionListResponse, error)
	CreateLowStockAlert(ctx *gin.Context, req *dto.LowStockAlertRequest) (*dto.LowStockAlertResponse, error)
	ListLowStockAlerts(ctx *gin.Context, req *dto.ListLowStockAlertsRequest) (*dto.LowStockAlertListResponse, error)
	UpdateLowStockAlert(ctx *gin.Context, id uuid.UUID, req *dto.LowStockAlertRequest) (*dto.LowStockAlertResponse, error)
	DeleteLowStockAlert(ctx *gin.Context, id uuid.UUID) error
	GetLowStockAlertsByWarehouse(ctx *gin.Context, warehouseID uuid.UUID) ([]*dto.LowStockAlertResponse, error)
}

// NewInventoryTransactionHandler creates a new inventory transaction handler
func NewInventoryTransactionHandler(transactionService InventoryTransactionService, logger zerolog.Logger) *InventoryTransactionHandler {
	return &InventoryTransactionHandler{
		transactionService: transactionService,
		logger:             logger,
	}
}

// Inventory Transaction CRUD Operations

// GetTransaction retrieves an inventory transaction by ID
// @Summary Get inventory transaction
// @Description Get an inventory transaction by its ID
// @Tags inventory-transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Success 200 {object} dto.InventoryTransactionResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/transactions/{id} [get]
func (h *InventoryTransactionHandler) GetTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid transaction ID format",
		})
		return
	}

	transaction, err := h.transactionService.GetTransaction(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("transaction_id", idStr).Msg("Failed to get inventory transaction")
		handleTransactionError(c, err)
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// ListTransactions retrieves a paginated list of inventory transactions
// @Summary List inventory transactions
// @Description Get a paginated list of inventory transactions with optional filtering
// @Tags inventory-transactions
// @Accept json
// @Produce json
// @Param product_id query string false "Product ID"
// @Param warehouse_id query string false "Warehouse ID"
// @Param transaction_type query string false "Transaction type"
// @Param reference_id query string false "Reference ID"
// @Param reference_type query string false "Reference type"
// @Param created_after query string false "Created after (ISO 8601)"
// @Param created_before query string false "Created before (ISO 8601)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param sort_by query string false "Sort field" default("created_at")
// @Param sort_order query string false "Sort order" Enums(asc,desc) default("desc")
// @Success 200 {object} dto.InventoryTransactionListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/transactions [get]
func (h *InventoryTransactionHandler) ListTransactions(c *gin.Context) {
	var req dto.ListInventoryTransactionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid transaction list request")
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

	result, err := h.transactionService.ListTransactions(c, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list inventory transactions")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve inventory transactions",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Transaction Management Operations

// ApproveTransaction approves an inventory transaction
// @Summary Approve inventory transaction
// @Description Approve a pending inventory transaction
// @Tags inventory-transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Param approval body dto.ApproveTransactionRequest true "Approval data"
// @Success 200 {object} dto.InventoryTransactionResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/transactions/{id}/approve [post]
func (h *InventoryTransactionHandler) ApproveTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid transaction ID format",
		})
		return
	}

	var req dto.ApproveTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid transaction approval request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	transaction, err := h.transactionService.ApproveTransaction(c, id, &req)
	if err != nil {
		h.logger.Error().Err(err).Str("transaction_id", idStr).Msg("Failed to approve inventory transaction")
		handleTransactionError(c, err)
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// RejectTransaction rejects an inventory transaction
// @Summary Reject inventory transaction
// @Description Reject a pending inventory transaction
// @Tags inventory-transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Param rejection body map[string]string true "Rejection reason"
// @Success 200 {object} dto.InventoryTransactionResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/transactions/{id}/reject [post]
func (h *InventoryTransactionHandler) RejectTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid transaction ID format",
		})
		return
	}

	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid transaction rejection request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	reason, exists := req["reason"]
	if !exists || reason == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Rejection reason is required",
		})
		return
	}

	transaction, err := h.transactionService.RejectTransaction(c, id, reason)
	if err != nil {
		h.logger.Error().Err(err).Str("transaction_id", idStr).Msg("Failed to reject inventory transaction")
		handleTransactionError(c, err)
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// GetTransactionStats retrieves transaction statistics
// @Summary Get transaction statistics
// @Description Get inventory transaction statistics and metrics
// @Tags inventory-transactions
// @Accept json
// @Produce json
// @Param warehouse_id query string false "Warehouse ID filter"
// @Param product_id query string false "Product ID filter"
// @Success 200 {object} dto.TransactionStatsResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/transactions/stats [get]
func (h *InventoryTransactionHandler) GetTransactionStats(c *gin.Context) {
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

	productIDStr := c.Query("product_id")
	var productID *uuid.UUID
	if productIDStr != "" {
		id, err := uuid.Parse(productIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error: "Invalid product ID format",
			})
			return
		}
		productID = &id
	}

	stats, err := h.transactionService.GetTransactionStats(c, warehouseID, productID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get transaction statistics")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve transaction statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetPendingApprovals retrieves transactions pending approval
// @Summary Get pending approvals
// @Description Get inventory transactions that are pending approval
// @Tags inventory-transactions
// @Accept json
// @Produce json
// @Param warehouse_id query string false "Warehouse ID filter"
// @Success 200 {object} dto.InventoryTransactionListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/transactions/pending [get]
func (h *InventoryTransactionHandler) GetPendingApprovals(c *gin.Context) {
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

	transactions, err := h.transactionService.GetPendingApprovals(c, warehouseID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get pending approvals")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve pending approvals",
		})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

// Low Stock Alert Operations

// CreateLowStockAlert creates a new low stock alert configuration
// @Summary Create low stock alert
// @Description Create a new low stock alert configuration
// @Tags low-stock-alerts
// @Accept json
// @Produce json
// @Param alert body dto.LowStockAlertRequest true "Alert configuration"
// @Success 201 {object} dto.LowStockAlertResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/alerts/low-stock [post]
func (h *InventoryTransactionHandler) CreateLowStockAlert(c *gin.Context) {
	var req dto.LowStockAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid low stock alert creation request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	alert, err := h.transactionService.CreateLowStockAlert(c, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create low stock alert")
		handleTransactionError(c, err)
		return
	}

	c.JSON(http.StatusCreated, alert)
}

// ListLowStockAlerts retrieves a paginated list of low stock alerts
// @Summary List low stock alerts
// @Description Get a paginated list of low stock alerts
// @Tags low-stock-alerts
// @Accept json
// @Produce json
// @Param product_id query string false "Product ID filter"
// @Param warehouse_id query string false "Warehouse ID filter"
// @Param is_active query bool false "Filter by active status"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param sort_by query string false "Sort field" default("created_at")
// @Param sort_order query string false "Sort order" Enums(asc,desc) default("desc")
// @Success 200 {object} dto.LowStockAlertListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/alerts/low-stock [get]
func (h *InventoryTransactionHandler) ListLowStockAlerts(c *gin.Context) {
	var req dto.ListLowStockAlertsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid low stock alert list request")
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

	result, err := h.transactionService.ListLowStockAlerts(c, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list low stock alerts")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve low stock alerts",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateLowStockAlert updates a low stock alert configuration
// @Summary Update low stock alert
// @Description Update an existing low stock alert configuration
// @Tags low-stock-alerts
// @Accept json
// @Produce json
// @Param id path string true "Alert ID"
// @Param alert body dto.LowStockAlertRequest true "Alert configuration"
// @Success 200 {object} dto.LowStockAlertResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/alerts/low-stock/{id} [put]
func (h *InventoryTransactionHandler) UpdateLowStockAlert(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid alert ID format",
		})
		return
	}

	var req dto.LowStockAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid low stock alert update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	alert, err := h.transactionService.UpdateLowStockAlert(c, id, &req)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", idStr).Msg("Failed to update low stock alert")
		handleTransactionError(c, err)
		return
	}

	c.JSON(http.StatusOK, alert)
}

// DeleteLowStockAlert deletes a low stock alert configuration
// @Summary Delete low stock alert
// @Description Delete a low stock alert configuration
// @Tags low-stock-alerts
// @Accept json
// @Produce json
// @Param id path string true "Alert ID"
// @Success 204
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/alerts/low-stock/{id} [delete]
func (h *InventoryTransactionHandler) DeleteLowStockAlert(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid alert ID format",
		})
		return
	}

	err = h.transactionService.DeleteLowStockAlert(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", idStr).Msg("Failed to delete low stock alert")
		handleTransactionError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetLowStockAlertsByWarehouse retrieves low stock alerts for a specific warehouse
// @Summary Get warehouse low stock alerts
// @Description Get all low stock alerts for a specific warehouse
// @Tags low-stock-alerts
// @Accept json
// @Produce json
// @Param warehouse_id path string true "Warehouse ID"
// @Success 200 {array} dto.LowStockAlertResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/alerts/low-stock/warehouse/{warehouse_id} [get]
func (h *InventoryTransactionHandler) GetLowStockAlertsByWarehouse(c *gin.Context) {
	warehouseIDStr := c.Param("warehouse_id")
	warehouseID, err := uuid.Parse(warehouseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid warehouse ID format",
		})
		return
	}

	alerts, err := h.transactionService.GetLowStockAlertsByWarehouse(c, warehouseID)
	if err != nil {
		h.logger.Error().Err(err).Str("warehouse_id", warehouseIDStr).Msg("Failed to get low stock alerts by warehouse")
		handleTransactionError(c, err)
		return
	}

	c.JSON(http.StatusOK, alerts)
}

// Utility Operations

// SearchTransactions searches inventory transactions by query
// @Summary Search inventory transactions
// @Description Search inventory transactions by query string
// @Tags inventory-transactions
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Result limit" default(20)
// @Success 200 {object} dto.InventoryTransactionListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/inventory/transactions/search [get]
func (h *InventoryTransactionHandler) SearchTransactions(c *gin.Context) {
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

	// Create a list request with search parameters
	req := &dto.ListInventoryTransactionsRequest{
		Page:      1,
		Limit:     limit,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	// Note: The actual search implementation would be in the service layer
	// This is a placeholder that would need proper search logic
	result, err := h.transactionService.ListTransactions(c, req)
	if err != nil {
		h.logger.Error().Err(err).Str("query", query).Msg("Failed to search inventory transactions")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to search inventory transactions",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// handleTransactionError handles inventory transaction service errors
func handleTransactionError(c *gin.Context, err error) {
	switch {
	case errors.IsNotFoundError(err):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Transaction not found",
			Details: err.Error(),
		})
	case errors.IsConflictError(err):
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Transaction conflict",
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