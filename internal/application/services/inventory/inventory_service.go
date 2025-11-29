package inventory

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"

	"erpgo/internal/domain/inventory/entities"
	"erpgo/internal/domain/inventory/repositories"
	"erpgo/internal/interfaces/http/dto"
	"erpgo/pkg/database"
)

// InventoryAvailabilityResponse represents inventory availability information
type InventoryAvailabilityResponse struct {
	Available    bool   `json:"available"`
	AvailableQty int    `json:"available_qty"`
	RequestedQty int    `json:"requested_qty"`
	Reason       string `json:"reason,omitempty"`
}

// InventoryBulkOperationResponse represents the result of bulk inventory operations
type InventoryBulkOperationResponse struct {
	SuccessCount int                                 `json:"success_count"`
	TotalCount   int                                 `json:"total_count"`
	Errors       []string                            `json:"errors,omitempty"`
	Results      []*dto.InventoryTransactionResponse `json:"results"`
}

// Service defines the business logic interface for inventory management
type Service interface {
	// Basic inventory operations
	AdjustInventory(ctx *gin.Context, req *dto.AdjustInventoryRequest) (*dto.InventoryTransactionResponse, error)
	ReserveInventory(ctx *gin.Context, req *dto.ReserveInventoryRequest) (*dto.InventoryResponse, error)
	ReleaseInventory(ctx *gin.Context, req *dto.ReleaseInventoryRequest) (*dto.InventoryResponse, error)
	TransferInventory(ctx *gin.Context, req *dto.TransferInventoryRequest) (*dto.InventoryTransactionResponse, error)

	// Inventory queries
	ListInventory(ctx *gin.Context, req *dto.ListInventoryRequest) (*dto.InventoryListResponse, error)
	GetInventoryByProductAndWarehouse(ctx *gin.Context, productID, warehouseID uuid.UUID) (*dto.InventoryResponse, error)
	GetInventoryStats(ctx *gin.Context) (*dto.InventoryStatsResponse, error)
	GetLowStockItems(ctx *gin.Context, warehouseID *uuid.UUID) (*dto.InventoryListResponse, error)

	// Availability and history
	CheckInventoryAvailability(ctx *gin.Context, productID, warehouseID uuid.UUID, quantity int) (*dto.AvailabilityResponse, error)
	GetInventoryHistory(ctx *gin.Context, productID, warehouseID uuid.UUID, limit int) ([]*dto.InventoryTransactionResponse, error)

	// Bulk operations
	BulkInventoryAdjustment(ctx *gin.Context, req *dto.BulkInventoryAdjustmentRequest) (*dto.BulkInventoryOperationResponse, error)
}

// ServiceImpl implements the inventory service interface
type ServiceImpl struct {
	inventoryRepo   repositories.InventoryRepository
	warehouseRepo   repositories.WarehouseRepository
	transactionRepo repositories.InventoryTransactionRepository
	txManager       database.TransactionManagerInterface
	logger          *zerolog.Logger
}

// NewService creates a new inventory service instance
func NewService(
	inventoryRepo repositories.InventoryRepository,
	warehouseRepo repositories.WarehouseRepository,
	transactionRepo repositories.InventoryTransactionRepository,
	txManager database.TransactionManagerInterface,
	logger *zerolog.Logger,
) Service {
	return &ServiceImpl{
		inventoryRepo:   inventoryRepo,
		warehouseRepo:   warehouseRepo,
		transactionRepo: transactionRepo,
		txManager:       txManager,
		logger:          logger,
	}
}

// AdjustInventory adjusts inventory stock quantity
func (s *ServiceImpl) AdjustInventory(c *gin.Context, req *dto.AdjustInventoryRequest) (*dto.InventoryTransactionResponse, error) {
	ctx := c.Request.Context()

	// Validate request
	if err := s.validateAdjustInventoryRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create adjustment transaction
	transaction := &entities.InventoryTransaction{
		ID:              uuid.New(),
		ProductID:       req.ProductID,
		WarehouseID:     req.WarehouseID,
		TransactionType: entities.TransactionTypeAdjustment,
		Quantity:        req.Adjustment,
		Reason:          req.Reason,
		CreatedAt:       time.Now().UTC(),
	}

	// Execute transaction creation and stock adjustment within a database transaction
	err := s.txManager.WithRetryTransaction(ctx, func(tx pgx.Tx) error {
		// Save transaction
		if err := s.transactionRepo.Create(ctx, transaction); err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		// Update inventory stock
		if err := s.inventoryRepo.AdjustStock(ctx, req.ProductID, req.WarehouseID, req.Adjustment); err != nil {
			return fmt.Errorf("failed to adjust stock: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Return response
	return &dto.InventoryTransactionResponse{
		ID:              transaction.ID,
		ProductID:       transaction.ProductID,
		WarehouseID:     transaction.WarehouseID,
		TransactionType: string(transaction.TransactionType),
		Quantity:        transaction.Quantity,
		Reason:          transaction.Reason,
		CreatedBy:       uuid.Nil, // No CreatedBy field in transaction entity
		CreatedAt:       transaction.CreatedAt,
		UpdatedAt:       transaction.CreatedAt,
	}, nil
}

// ReserveInventory reserves inventory stock for orders
func (s *ServiceImpl) ReserveInventory(c *gin.Context, req *dto.ReserveInventoryRequest) (*dto.InventoryResponse, error) {
	ctx := c.Request.Context()
	// Validate request
	if err := s.validateReserveInventoryRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check availability
	available, err := s.inventoryRepo.GetAvailableStock(ctx, req.ProductID, req.WarehouseID)
	if err != nil {
		return nil, fmt.Errorf("failed to check available stock: %w", err)
	}

	if available < req.Quantity {
		return nil, fmt.Errorf("insufficient stock: available %d, requested %d", available, req.Quantity)
	}

	// Reserve stock
	if err := s.inventoryRepo.ReserveStock(ctx, req.ProductID, req.WarehouseID, req.Quantity); err != nil {
		return nil, fmt.Errorf("failed to reserve stock: %w", err)
	}

	// Get updated inventory
	inventory, err := s.inventoryRepo.GetByProductAndWarehouse(ctx, req.ProductID, req.WarehouseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated inventory: %w", err)
	}

	return s.inventoryToDTO(inventory), nil
}

// ReleaseInventory releases reserved inventory stock
func (s *ServiceImpl) ReleaseInventory(c *gin.Context, req *dto.ReleaseInventoryRequest) (*dto.InventoryResponse, error) {

	ctx := c.Request.Context()
	// Validate request
	if err := s.validateReleaseInventoryRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// TODO: This method needs reservation tracking system to work properly
	// For now, return an error as the current repository doesn't support release by reservation ID
	return nil, fmt.Errorf("release inventory by reservation ID not yet implemented - requires reservation tracking")
}

// TransferInventory transfers inventory between warehouses
func (s *ServiceImpl) TransferInventory(c *gin.Context, req *dto.TransferInventoryRequest) (*dto.InventoryTransactionResponse, error) {

	ctx := c.Request.Context()
	// Validate request
	if err := s.validateTransferInventoryRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check availability at source warehouse
	available, err := s.inventoryRepo.GetAvailableStock(ctx, req.ProductID, req.FromWarehouseID)
	if err != nil {
		return nil, fmt.Errorf("failed to check available stock: %w", err)
	}

	if available < req.Quantity {
		return nil, fmt.Errorf("insufficient stock at source warehouse: available %d, requested %d", available, req.Quantity)
	}

	// Create outbound transaction
	outboundTransaction := &entities.InventoryTransaction{
		ID:              uuid.New(),
		ProductID:       req.ProductID,
		WarehouseID:     req.FromWarehouseID,
		TransactionType: entities.TransactionTypeTransferOut,
		Quantity:        -req.Quantity, // Negative for outbound
		Reason:          fmt.Sprintf("Transfer to warehouse %s", req.ToWarehouseID.String()),
		CreatedAt:       time.Now().UTC(),
	}

	// Create inbound transaction
	inboundTransaction := &entities.InventoryTransaction{
		ID:              uuid.New(),
		ProductID:       req.ProductID,
		WarehouseID:     req.ToWarehouseID,
		TransactionType: entities.TransactionTypeTransferIn,
		Quantity:        req.Quantity, // Positive for inbound
		Reason:          fmt.Sprintf("Transfer from warehouse %s", req.FromWarehouseID.String()),
		CreatedAt:       time.Now().UTC(),
	}

	// Execute all operations within a transaction
	var response *dto.InventoryTransactionResponse
	err = s.txManager.WithRetryTransaction(ctx, func(tx pgx.Tx) error {
		// Save outbound transaction
		if err := s.transactionRepo.Create(ctx, outboundTransaction); err != nil {
			return fmt.Errorf("failed to create outbound transaction: %w", err)
		}

		// Save inbound transaction
		if err := s.transactionRepo.Create(ctx, inboundTransaction); err != nil {
			return fmt.Errorf("failed to create inbound transaction: %w", err)
		}

		// Update source inventory (remove stock)
		if err := s.inventoryRepo.AdjustStock(ctx, req.ProductID, req.FromWarehouseID, -req.Quantity); err != nil {
			return fmt.Errorf("failed to adjust source inventory: %w", err)
		}

		// Update destination inventory (add stock)
		if err := s.inventoryRepo.AdjustStock(ctx, req.ProductID, req.ToWarehouseID, req.Quantity); err != nil {
			return fmt.Errorf("failed to adjust destination inventory: %w", err)
		}

		// Prepare response (using outbound transaction as primary)
		response = &dto.InventoryTransactionResponse{
			ID:              outboundTransaction.ID,
			ProductID:       outboundTransaction.ProductID,
			WarehouseID:     outboundTransaction.WarehouseID,
			TransactionType: string(outboundTransaction.TransactionType),
			Quantity:        outboundTransaction.Quantity,
			Reason:          outboundTransaction.Reason,
			CreatedBy:       uuid.Nil, // No CreatedBy field in transaction entity
			CreatedAt:       outboundTransaction.CreatedAt,
			UpdatedAt:       outboundTransaction.CreatedAt,
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("transfer inventory transaction failed: %w", err)
	}

	return response, nil
}

// ListInventory lists inventory with filtering
func (s *ServiceImpl) ListInventory(c *gin.Context, req *dto.ListInventoryRequest) (*dto.InventoryListResponse, error) {

	ctx := c.Request.Context()
	// Build filter
	filter := &repositories.InventoryFilter{
		Limit:   req.Limit,
		Offset:  (req.Page - 1) * req.Limit,
		OrderBy: req.SortBy,
		Order:   req.SortOrder,
	}

	// Parse UUID filters if provided
	if req.WarehouseID != "" {
		if warehouseID, err := uuid.Parse(req.WarehouseID); err == nil {
			filter.WarehouseIDs = []uuid.UUID{warehouseID}
		}
	}

	if req.ProductID != "" {
		if productID, err := uuid.Parse(req.ProductID); err == nil {
			filter.ProductIDs = []uuid.UUID{productID}
		}
	}

	// Add other filters
	if req.SKU != "" {
		filter.SKU = req.SKU
	}
	if req.ProductName != "" {
		filter.ProductName = req.ProductName
	}
	if req.WarehouseCode != "" {
		filter.WarehouseCode = req.WarehouseCode
	}

	// Get inventory list
	inventories, err := s.inventoryRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory: %w", err)
	}

	// Get total count
	total, err := s.inventoryRepo.Count(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count inventory: %w", err)
	}

	// Convert to DTOs
	inventoryDTOs := make([]*dto.InventoryResponse, len(inventories))
	for i, inv := range inventories {
		inventoryDTOs[i] = s.inventoryToDTO(inv)
	}

	totalPages := (total + req.Limit - 1) / req.Limit
	return &dto.InventoryListResponse{
		Inventory: inventoryDTOs,
		Pagination: &dto.PaginationInfo{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    req.Page < totalPages,
			HasPrev:    req.Page > 1,
		},
	}, nil
}

// GetInventoryByProductAndWarehouse gets inventory for a specific product and warehouse
func (s *ServiceImpl) GetInventoryByProductAndWarehouse(c *gin.Context, productID, warehouseID uuid.UUID) (*dto.InventoryResponse, error) {
	ctx := c.Request.Context()

	inventory, err := s.inventoryRepo.GetByProductAndWarehouse(ctx, productID, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	return s.inventoryToDTO(inventory), nil
}

// GetInventoryStats gets inventory statistics
func (s *ServiceImpl) GetInventoryStats(c *gin.Context) (*dto.InventoryStatsResponse, error) {

	ctx := c.Request.Context()
	// This is a simplified implementation
	// In a real system, you might want to cache these statistics or use more efficient queries

	// Get all inventory
	allInventory, err := s.inventoryRepo.List(ctx, &repositories.InventoryFilter{
		Limit: 10000, // Adjust based on your needs
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get all inventory: %w", err)
	}

	// Calculate statistics
	stats := &dto.InventoryStatsResponse{
		TotalProducts:       len(allInventory),
		TotalWarehouses:     0, // Would need to be calculated separately
		LowStockItems:       0,
		OutOfStockItems:     0,
		TotalInventoryValue: decimal.Zero,
		TotalStockQuantity:  0,
	}

	warehouseSet := make(map[uuid.UUID]bool)

	for _, inv := range allInventory {
		// Count unique warehouses
		warehouseSet[inv.WarehouseID] = true

		// Count low stock and out of stock items
		if inv.IsLowStock() {
			stats.LowStockItems++
		}
		if inv.GetStockStatus() == "OUT_OF_STOCK" {
			stats.OutOfStockItems++
		}

		// Calculate total value and stock levels
		availableStock := inv.QuantityOnHand - inv.QuantityReserved
		itemValue := decimal.NewFromFloat(inv.AverageCost).Mul(decimal.NewFromInt(int64(availableStock)))
		stats.TotalInventoryValue = stats.TotalInventoryValue.Add(itemValue)
		stats.TotalStockQuantity += availableStock
	}

	stats.TotalWarehouses = len(warehouseSet)

	return stats, nil
}

// GetLowStockItems gets items with low stock
func (s *ServiceImpl) GetLowStockItems(c *gin.Context, warehouseID *uuid.UUID) (*dto.InventoryListResponse, error) {
	ctx := c.Request.Context()

	var inventories []*entities.Inventory
	var err error

	if warehouseID != nil {
		inventories, err = s.inventoryRepo.GetLowStockItems(ctx, *warehouseID)
	} else {
		inventories, err = s.inventoryRepo.GetLowStockItemsAll(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get low stock items: %w", err)
	}

	// Convert to DTOs
	inventoryDTOs := make([]*dto.InventoryResponse, len(inventories))
	for i, inv := range inventories {
		inventoryDTOs[i] = s.inventoryToDTO(inv)
	}

	return &dto.InventoryListResponse{
		Inventory: inventoryDTOs,
		Pagination: &dto.PaginationInfo{
			Page:       1,
			Limit:      len(inventories),
			Total:      len(inventories),
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		},
	}, nil
}

// CheckInventoryAvailability checks if inventory is available
func (s *ServiceImpl) CheckInventoryAvailability(c *gin.Context, productID, warehouseID uuid.UUID, quantity int) (*dto.AvailabilityResponse, error) {
	ctx := c.Request.Context()

	available, err := s.inventoryRepo.GetAvailableStock(ctx, productID, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("failed to check available stock: %w", err)
	}

	isAvailable := available >= quantity
	var reason string
	if !isAvailable {
		reason = fmt.Sprintf("Insufficient stock: %d available, %d requested", available, quantity)
	}

	return &dto.AvailabilityResponse{
		ProductID:        productID,
		RequestedQty:     quantity,
		Available:        isAvailable,
		Reason:           reason,
		CanFulfill:       isAvailable,
		BackorderAllowed: false,
	}, nil
}

// GetInventoryHistory gets inventory transaction history
func (s *ServiceImpl) GetInventoryHistory(c *gin.Context, productID, warehouseID uuid.UUID, limit int) ([]*dto.InventoryTransactionResponse, error) {
	// This would require implementing a method in the transaction repository
	// For now, return empty slice
	return []*dto.InventoryTransactionResponse{}, nil
}

// BulkInventoryAdjustment performs bulk inventory adjustments
func (s *ServiceImpl) BulkInventoryAdjustment(c *gin.Context, req *dto.BulkInventoryAdjustmentRequest) (*dto.BulkInventoryOperationResponse, error) {
	if len(req.Adjustments) == 0 {
		return nil, fmt.Errorf("no adjustments provided")
	}

	responses := make([]*dto.InventoryTransactionResponse, len(req.Adjustments))
	errors := make([]string, 0)

	for i, adj := range req.Adjustments {
		response, err := s.AdjustInventory(c, &adj)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Adjustment %d failed: %v", i+1, err))
			continue
		}

		responses[i] = response
	}

	return &dto.BulkInventoryOperationResponse{
		SuccessCount: len(req.Adjustments) - len(errors),
		FailedCount:  len(errors),
		TotalCount:   len(req.Adjustments),
		Results:      nil, // TODO: Convert responses to proper results
	}, nil
}

// Helper methods

func (s *ServiceImpl) validateAdjustInventoryRequest(ctx context.Context, req *dto.AdjustInventoryRequest) error {
	if req.ProductID == uuid.Nil {
		return fmt.Errorf("product ID is required")
	}
	if req.WarehouseID == uuid.Nil {
		return fmt.Errorf("warehouse ID is required")
	}
	if req.Adjustment == 0 {
		return fmt.Errorf("adjustment cannot be zero")
	}
	if req.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

func (s *ServiceImpl) validateReserveInventoryRequest(ctx context.Context, req *dto.ReserveInventoryRequest) error {
	if req.ProductID == uuid.Nil {
		return fmt.Errorf("product ID is required")
	}
	if req.WarehouseID == uuid.Nil {
		return fmt.Errorf("warehouse ID is required")
	}
	if req.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	return nil
}

func (s *ServiceImpl) validateReleaseInventoryRequest(ctx context.Context, req *dto.ReleaseInventoryRequest) error {
	if req.ReservationID == uuid.Nil {
		return fmt.Errorf("reservation ID is required")
	}
	if req.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if req.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

func (s *ServiceImpl) validateTransferInventoryRequest(ctx context.Context, req *dto.TransferInventoryRequest) error {
	if req.ProductID == uuid.Nil {
		return fmt.Errorf("product ID is required")
	}
	if req.FromWarehouseID == uuid.Nil {
		return fmt.Errorf("source warehouse ID is required")
	}
	if req.ToWarehouseID == uuid.Nil {
		return fmt.Errorf("destination warehouse ID is required")
	}
	if req.FromWarehouseID == req.ToWarehouseID {
		return fmt.Errorf("source and destination warehouses cannot be the same")
	}
	if req.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	return nil
}

func (s *ServiceImpl) inventoryToDTO(inventory *entities.Inventory) *dto.InventoryResponse {
	availableStock := inventory.QuantityOnHand - inventory.QuantityReserved

	dto := &dto.InventoryResponse{
		ID:                inventory.ID,
		ProductID:         inventory.ProductID,
		ProductSKU:        "", // Not in entity, would need join
		ProductName:       "", // Not in entity, would need join
		WarehouseID:       inventory.WarehouseID,
		WarehouseCode:     "", // Not in entity, would need join
		WarehouseName:     "", // Not in entity, would need join
		Quantity:          inventory.QuantityOnHand,
		ReservedQuantity:  inventory.QuantityReserved,
		AvailableQuantity: availableStock,
		MinStockLevel:     0, // Not in entity, would need to calculate
		MaxStockLevel:     inventory.MaxStock,
		IsLowStock:        inventory.IsLowStock(),
		IsOutOfStock:      inventory.GetStockStatus() == "OUT_OF_STOCK",
		LastUpdated:       inventory.UpdatedAt,
		CreatedAt:         time.Time{}, // Not in entity
		UpdatedAt:         inventory.UpdatedAt,
	}

	return dto
}
