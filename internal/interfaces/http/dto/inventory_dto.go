package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Warehouse Request DTOs

// CreateWarehouseRequest represents a request to create a warehouse
type CreateWarehouseRequest struct {
	Name                  string     `json:"name" binding:"required,min=1,max=200"`
	Code                  string     `json:"code" binding:"required,min=2,max=20"`
	Address               string     `json:"address" binding:"required,min=1,max=500"`
	City                  string     `json:"city" binding:"required,min=1,max=100"`
	State                 string     `json:"state,omitempty" binding:"omitempty,max=100"`
	Country               string     `json:"country" binding:"required,min=1,max=100"`
	PostalCode            string     `json:"postal_code" binding:"required,min=1,max=20"`
	Phone                 string     `json:"phone,omitempty" binding:"omitempty,max=20"`
	Email                 string     `json:"email,omitempty" binding:"omitempty,email,max=255"`
	ManagerID             *uuid.UUID `json:"manager_id,omitempty"`
	Type                  string     `json:"type,omitempty" binding:"omitempty,oneof=RETAIL WHOLESALE DISTRIBUTION FULFILLMENT RETURN"`
	Capacity              *int       `json:"capacity,omitempty" binding:"omitempty,min=0"`
	SquareFootage         *int       `json:"square_footage,omitempty" binding:"omitempty,min=0"`
	DockCount             *int       `json:"dock_count,omitempty" binding:"omitempty,min=0,max=9999"`
	TemperatureControlled bool       `json:"temperature_controlled"`
	SecurityLevel         int        `json:"security_level" binding:"omitempty,min=0,max=10"`
	Description           string     `json:"description,omitempty" binding:"omitempty,max=2000"`
}

// UpdateWarehouseRequest represents a request to update a warehouse
type UpdateWarehouseRequest struct {
	Name                  *string    `json:"name,omitempty" binding:"omitempty,min=1,max=200"`
	Address               *string    `json:"address,omitempty" binding:"omitempty,min=1,max=500"`
	City                  *string    `json:"city,omitempty" binding:"omitempty,min=1,max=100"`
	State                 *string    `json:"state,omitempty" binding:"omitempty,max=100"`
	Country               *string    `json:"country,omitempty" binding:"omitempty,min=1,max=100"`
	PostalCode            *string    `json:"postal_code,omitempty" binding:"omitempty,min=1,max=20"`
	Phone                 *string    `json:"phone,omitempty" binding:"omitempty,max=20"`
	Email                 *string    `json:"email,omitempty" binding:"omitempty,email,max=255"`
	ManagerID             *uuid.UUID `json:"manager_id,omitempty"`
	Type                  *string    `json:"type,omitempty" binding:"omitempty,oneof=RETAIL WHOLESALE DISTRIBUTION FULFILLMENT RETURN"`
	Capacity              *int       `json:"capacity,omitempty" binding:"omitempty,min=0"`
	SquareFootage         *int       `json:"square_footage,omitempty" binding:"omitempty,min=0"`
	DockCount             *int       `json:"dock_count,omitempty" binding:"omitempty,min=0,max=9999"`
	TemperatureControlled *bool      `json:"temperature_controlled,omitempty"`
	SecurityLevel         *int       `json:"security_level,omitempty" binding:"omitempty,min=0,max=10"`
	Description           *string    `json:"description,omitempty" binding:"omitempty,max=2000"`
}

// ListWarehousesRequest represents a request to list warehouses
type ListWarehousesRequest struct {
	Search     string `form:"search" binding:"omitempty,max=200"`
	Code       string `form:"code" binding:"omitempty,max=20"`
	City       string `form:"city" binding:"omitempty,max=100"`
	State      string `form:"state" binding:"omitempty,max=100"`
	Country    string `form:"country" binding:"omitempty,max=100"`
	Type       string `form:"type" binding:"omitempty,oneof=RETAIL WHOLESALE DISTRIBUTION FULFILLMENT RETURN"`
	IsActive   *bool  `form:"is_active"`
	HasManager *bool  `form:"has_manager"`
	ManagerID  string `form:"manager_id" binding:"omitempty,uuid"`
	Page       int    `form:"page,default=1" binding:"omitempty,min=1"`
	Limit      int    `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	SortBy     string `form:"sort_by,default=created_at" binding:"omitempty,oneof=name code city state country created_at updated_at"`
	SortOrder  string `form:"sort_order,default=desc" binding:"omitempty,oneof=asc desc"`
}

// Warehouse Response DTOs

// WarehouseResponse represents a warehouse response
type WarehouseResponse struct {
	ID                    uuid.UUID  `json:"id"`
	Name                  string     `json:"name"`
	Code                  string     `json:"code"`
	Address               string     `json:"address"`
	City                  string     `json:"city"`
	State                 string     `json:"state,omitempty"`
	Country               string     `json:"country"`
	PostalCode            string     `json:"postal_code"`
	Phone                 string     `json:"phone,omitempty"`
	Email                 string     `json:"email,omitempty"`
	ManagerID             *uuid.UUID `json:"manager_id,omitempty"`
	Type                  string     `json:"type,omitempty"`
	Capacity              *int       `json:"capacity,omitempty"`
	SquareFootage         *int       `json:"square_footage,omitempty"`
	DockCount             *int       `json:"dock_count,omitempty"`
	TemperatureControlled bool       `json:"temperature_controlled"`
	SecurityLevel         int        `json:"security_level"`
	Description           string     `json:"description,omitempty"`
	IsActive              bool       `json:"is_active"`
	UtilizationPercentage *float64   `json:"utilization_percentage,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// WarehouseListResponse represents a paginated warehouse list response
type WarehouseListResponse struct {
	Warehouses []*WarehouseResponse `json:"warehouses"`
	Pagination *PaginationInfo      `json:"pagination"`
}

// WarehouseStatsResponse represents warehouse statistics
type WarehouseStatsResponse struct {
	TotalWarehouses    int            `json:"total_warehouses"`
	ActiveWarehouses   int            `json:"active_warehouses"`
	InactiveWarehouses int            `json:"inactive_warehouses"`
	WarehousesByType   map[string]int `json:"warehouses_by_type"`
	AverageCapacity    *float64       `json:"average_capacity,omitempty"`
	TotalCapacity      *int           `json:"total_capacity,omitempty"`
	TotalUtilization   *int           `json:"total_utilization,omitempty"`
}

// Inventory Request DTOs

// AdjustInventoryRequest represents a request to adjust inventory
type AdjustInventoryRequest struct {
	ProductID     uuid.UUID  `json:"product_id" binding:"required"`
	WarehouseID   uuid.UUID  `json:"warehouse_id" binding:"required"`
	Adjustment    int        `json:"adjustment" binding:"required"`
	Reason        string     `json:"reason" binding:"required,min=1,max=500"`
	ReferenceID   *uuid.UUID `json:"reference_id,omitempty"`
	ReferenceType string     `json:"reference_type,omitempty" binding:"omitempty,oneof=order purchase return adjustment transfer"`
}

// ReserveInventoryRequest represents a request to reserve inventory
type ReserveInventoryRequest struct {
	ProductID     uuid.UUID  `json:"product_id" binding:"required"`
	WarehouseID   uuid.UUID  `json:"warehouse_id" binding:"required"`
	Quantity      int        `json:"quantity" binding:"required,min=1"`
	Reason        string     `json:"reason" binding:"required,min=1,max=500"`
	ReferenceID   *uuid.UUID `json:"reference_id,omitempty"`
	ReferenceType string     `json:"reference_type,omitempty" binding:"omitempty,oneof=order quote transfer"`
	Priority      int        `json:"priority" binding:"omitempty,min=1,max=10"`
}

// ReleaseInventoryRequest represents a request to release reserved inventory
type ReleaseInventoryRequest struct {
	ReservationID uuid.UUID `json:"reservation_id" binding:"required"`
	Quantity      int       `json:"quantity" binding:"required,min=1"`
	Reason        string    `json:"reason" binding:"required,min=1,max=500"`
}

// TransferInventoryRequest represents a request to transfer inventory between warehouses
type TransferInventoryRequest struct {
	ProductID       uuid.UUID  `json:"product_id" binding:"required"`
	FromWarehouseID uuid.UUID  `json:"from_warehouse_id" binding:"required"`
	ToWarehouseID   uuid.UUID  `json:"to_warehouse_id" binding:"required"`
	Quantity        int        `json:"quantity" binding:"required,min=1"`
	Reason          string     `json:"reason" binding:"required,min=1,max=500"`
	ReferenceID     *uuid.UUID `json:"reference_id,omitempty"`
	ReferenceType   string     `json:"reference_type,omitempty" binding:"omitempty,oneof=order transfer adjustment"`
}

// ListInventoryRequest represents a request to list inventory items
type ListInventoryRequest struct {
	ProductID       string `form:"product_id" binding:"omitempty,uuid"`
	WarehouseID     string `form:"warehouse_id" binding:"omitempty,uuid"`
	SKU             string `form:"sku" binding:"omitempty,max=100"`
	ProductName     string `form:"product_name" binding:"omitempty,max=200"`
	WarehouseCode   string `form:"warehouse_code" binding:"omitempty,max=20"`
	City            string `form:"city" binding:"omitempty,max=100"`
	State           string `form:"state" binding:"omitempty,max=100"`
	Country         string `form:"country" binding:"omitempty,max=100"`
	InStock         *bool  `form:"in_stock"`
	LowStock        *bool  `form:"low_stock"`
	OutOfStock      *bool  `form:"out_of_stock"`
	HasReservations *bool  `form:"has_reservations"`
	Page            int    `form:"page,default=1" binding:"omitempty,min=1"`
	Limit           int    `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	SortBy          string `form:"sort_by,default=created_at" binding:"omitempty,oneof=product_name warehouse_code quantity reserved_quantity available_quantity created_at updated_at"`
	SortOrder       string `form:"sort_order,default=desc" binding:"omitempty,oneof=asc desc"`
}

// Inventory Response DTOs

// InventoryResponse represents an inventory item response
type InventoryResponse struct {
	ID                uuid.UUID `json:"id"`
	ProductID         uuid.UUID `json:"product_id"`
	ProductSKU        string    `json:"product_sku"`
	ProductName       string    `json:"product_name"`
	WarehouseID       uuid.UUID `json:"warehouse_id"`
	WarehouseCode     string    `json:"warehouse_code"`
	WarehouseName     string    `json:"warehouse_name"`
	Quantity          int       `json:"quantity"`
	ReservedQuantity  int       `json:"reserved_quantity"`
	AvailableQuantity int       `json:"available_quantity"`
	MinStockLevel     int       `json:"min_stock_level"`
	MaxStockLevel     *int      `json:"max_stock_level,omitempty"`
	IsLowStock        bool      `json:"is_low_stock"`
	IsOutOfStock      bool      `json:"is_out_of_stock"`
	LastUpdated       time.Time `json:"last_updated"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// InventoryListResponse represents a paginated inventory list response
type InventoryListResponse struct {
	Inventory  []*InventoryResponse `json:"inventory"`
	Pagination *PaginationInfo      `json:"pagination"`
}

// InventoryStatsResponse represents inventory statistics
type InventoryStatsResponse struct {
	TotalProducts        int                  `json:"total_products"`
	TotalWarehouses      int                  `json:"total_warehouses"`
	TotalInventoryValue  decimal.Decimal      `json:"total_inventory_value"`
	TotalStockQuantity   int                  `json:"total_stock_quantity"`
	LowStockItems        int                  `json:"low_stock_items"`
	OutOfStockItems      int                  `json:"out_of_stock_items"`
	TotalReservations    int                  `json:"total_reservations"`
	TopWarehousesByStock []WarehouseStockInfo `json:"top_warehouses_by_stock"`
	TopProductsByValue   []ProductValueInfo   `json:"top_products_by_value"`
}

// WarehouseStockInfo represents warehouse stock information for statistics
type WarehouseStockInfo struct {
	WarehouseID   uuid.UUID       `json:"warehouse_id"`
	WarehouseName string          `json:"warehouse_name"`
	TotalStock    int             `json:"total_stock"`
	TotalValue    decimal.Decimal `json:"total_value"`
	ProductCount  int             `json:"product_count"`
}

// ProductValueInfo represents product value information for statistics
type ProductValueInfo struct {
	ProductID      uuid.UUID       `json:"product_id"`
	ProductSKU     string          `json:"product_sku"`
	ProductName    string          `json:"product_name"`
	TotalStock     int             `json:"total_stock"`
	TotalValue     decimal.Decimal `json:"total_value"`
	WarehouseCount int             `json:"warehouse_count"`
}

// Inventory Transaction Request DTOs

// ListInventoryTransactionsRequest represents a request to list inventory transactions
type ListInventoryTransactionsRequest struct {
	ProductID       string `form:"product_id" binding:"omitempty,uuid"`
	WarehouseID     string `form:"warehouse_id" binding:"omitempty,uuid"`
	TransactionType string `form:"transaction_type" binding:"omitempty,oneof=ADJUSTMENT RESERVATION RELEASE TRANSFER IN OUT"`
	ReferenceID     string `form:"reference_id" binding:"omitempty,uuid"`
	ReferenceType   string `form:"reference_type" binding:"omitempty,oneof=order purchase return adjustment transfer"`
	CreatedAfter    string `form:"created_after" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	CreatedBefore   string `form:"created_before" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	Page            int    `form:"page,default=1" binding:"omitempty,min=1"`
	Limit           int    `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	SortBy          string `form:"sort_by,default=created_at" binding:"omitempty,oneof=created_at transaction_type quantity reference_type"`
	SortOrder       string `form:"sort_order,default=desc" binding:"omitempty,oneof=asc desc"`
}

// ApproveTransactionRequest represents a request to approve a transaction
type ApproveTransactionRequest struct {
	ApprovedBy uuid.UUID `json:"approved_by" binding:"required"`
	Notes      string    `json:"notes" binding:"omitempty,max=500"`
}

// Inventory Transaction Response DTOs

// InventoryTransactionResponse represents an inventory transaction response
type InventoryTransactionResponse struct {
	ID               uuid.UUID  `json:"id"`
	ProductID        uuid.UUID  `json:"product_id"`
	ProductSKU       string     `json:"product_sku"`
	ProductName      string     `json:"product_name"`
	WarehouseID      uuid.UUID  `json:"warehouse_id"`
	WarehouseCode    string     `json:"warehouse_code"`
	WarehouseName    string     `json:"warehouse_name"`
	TransactionType  string     `json:"transaction_type"`
	Quantity         int        `json:"quantity"`
	PreviousQuantity int        `json:"previous_quantity"`
	NewQuantity      int        `json:"new_quantity"`
	Reason           string     `json:"reason"`
	ReferenceID      *uuid.UUID `json:"reference_id,omitempty"`
	ReferenceType    string     `json:"reference_type,omitempty"`
	ApprovedBy       *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	ApprovalNotes    string     `json:"approval_notes,omitempty"`
	CreatedBy        uuid.UUID  `json:"created_by"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// InventoryTransactionListResponse represents a paginated inventory transaction list response
type InventoryTransactionListResponse struct {
	Transactions []*InventoryTransactionResponse `json:"transactions"`
	Pagination   *PaginationInfo                 `json:"pagination"`
}

// Low Stock Alert DTOs

// LowStockAlertRequest represents a request to configure low stock alerts
type LowStockAlertRequest struct {
	ProductID   *uuid.UUID `json:"product_id,omitempty"`
	WarehouseID *uuid.UUID `json:"warehouse_id,omitempty"`
	Threshold   *int       `json:"threshold,omitempty" binding:"omitempty,min=0"`
	IsActive    *bool      `json:"is_active,omitempty"`
}

// LowStockAlertResponse represents a low stock alert response
type LowStockAlertResponse struct {
	ID            uuid.UUID  `json:"id"`
	ProductID     uuid.UUID  `json:"product_id"`
	ProductSKU    string     `json:"product_sku"`
	ProductName   string     `json:"product_name"`
	WarehouseID   uuid.UUID  `json:"warehouse_id"`
	WarehouseCode string     `json:"warehouse_code"`
	WarehouseName string     `json:"warehouse_name"`
	CurrentStock  int        `json:"current_stock"`
	MinStockLevel int        `json:"min_stock_level"`
	Threshold     int        `json:"threshold"`
	IsActive      bool       `json:"is_active"`
	LastAlertSent *time.Time `json:"last_alert_sent,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// LowStockAlertListResponse represents a paginated low stock alert list response
type LowStockAlertListResponse struct {
	Alerts     []*LowStockAlertResponse `json:"alerts"`
	Pagination *PaginationInfo          `json:"pagination"`
}

// Bulk Operations

// BulkInventoryAdjustmentRequest represents a request for bulk inventory adjustments
type BulkInventoryAdjustmentRequest struct {
	Adjustments []AdjustInventoryRequest `json:"adjustments" binding:"required,min=1,max=100"`
	DryRun      bool                     `json:"dry_run"`
}

// BulkInventoryOperationResponse represents a response for bulk inventory operations
type BulkInventoryOperationResponse struct {
	SuccessCount int                            `json:"success_count"`
	FailedCount  int                            `json:"failed_count"`
	TotalCount   int                            `json:"total_count"`
	Errors       []BulkInventoryOperationError  `json:"errors,omitempty"`
	Results      []BulkInventoryOperationResult `json:"results,omitempty"`
	Summary      *BulkOperationSummary          `json:"summary,omitempty"`
}

// BulkInventoryOperationError represents an error in a bulk inventory operation
type BulkInventoryOperationError struct {
	Index       int       `json:"index"`
	ProductID   uuid.UUID `json:"product_id"`
	WarehouseID uuid.UUID `json:"warehouse_id"`
	Error       string    `json:"error"`
	Details     string    `json:"details,omitempty"`
}

// BulkInventoryOperationResult represents a result in a bulk inventory operation
type BulkInventoryOperationResult struct {
	Index         int        `json:"index"`
	ProductID     uuid.UUID  `json:"product_id"`
	WarehouseID   uuid.UUID  `json:"warehouse_id"`
	Success       bool       `json:"success"`
	Adjustment    int        `json:"adjustment"`
	NewQuantity   int        `json:"new_quantity,omitempty"`
	TransactionID *uuid.UUID `json:"transaction_id,omitempty"`
	Error         string     `json:"error,omitempty"`
}

// BulkOperationSummary represents a summary of bulk operations
type BulkOperationSummary struct {
	TotalAdjustments    int             `json:"total_adjustments"`
	PositiveAdjustments int             `json:"positive_adjustments"`
	NegativeAdjustments int             `json:"negative_adjustments"`
	TotalValueChanged   decimal.Decimal `json:"total_value_changed"`
	AffectedProducts    int             `json:"affected_products"`
	AffectedWarehouses  int             `json:"affected_warehouses"`
}

// Transaction Stats DTOs

// TransactionStatsResponse represents transaction statistics
type TransactionStatsResponse struct {
	TotalTransactions    int                         `json:"total_transactions"`
	PendingApprovals     int                         `json:"pending_approvals"`
	ApprovedTransactions int                         `json:"approved_transactions"`
	RejectedTransactions int                         `json:"rejected_transactions"`
	TransactionsByType   map[string]int              `json:"transactions_by_type"`
	TransactionsByDay    []DailyTransactionStats     `json:"transactions_by_day"`
	TotalQuantityIn      int                         `json:"total_quantity_in"`
	TotalQuantityOut     int                         `json:"total_quantity_out"`
	NetQuantityChange    int                         `json:"net_quantity_change"`
	MostActiveProducts   []ProductTransactionStats   `json:"most_active_products"`
	MostActiveWarehouses []WarehouseTransactionStats `json:"most_active_warehouses"`
}

// DailyTransactionStats represents daily transaction statistics
type DailyTransactionStats struct {
	Date        string `json:"date"`
	Count       int    `json:"count"`
	QuantityIn  int    `json:"quantity_in"`
	QuantityOut int    `json:"quantity_out"`
}

// ProductTransactionStats represents product transaction statistics
type ProductTransactionStats struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductSKU  string    `json:"product_sku"`
	ProductName string    `json:"product_name"`
	Count       int       `json:"count"`
	Quantity    int       `json:"quantity"`
}

// WarehouseTransactionStats represents warehouse transaction statistics
type WarehouseTransactionStats struct {
	WarehouseID   uuid.UUID `json:"warehouse_id"`
	WarehouseName string    `json:"warehouse_name"`
	Count         int       `json:"count"`
	Quantity      int       `json:"quantity"`
}

// ListLowStockAlertsRequest represents a request to list low stock alerts
type ListLowStockAlertsRequest struct {
	ProductID   string `form:"product_id" binding:"omitempty,uuid"`
	WarehouseID string `form:"warehouse_id" binding:"omitempty,uuid"`
	IsActive    *bool  `form:"is_active"`
	Page        int    `form:"page,default=1" binding:"omitempty,min=1"`
	Limit       int    `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	SortBy      string `form:"sort_by,default=created_at" binding:"omitempty,oneof=created_at updated_at current_stock threshold"`
	SortOrder   string `form:"sort_order,default=desc" binding:"omitempty,oneof=asc desc"`
}

// Additional DTOs needed for inventory service operations
