package repositories

import (
	"context"
	"time"

	"erpgo/internal/domain/inventory/entities"
	"github.com/google/uuid"
)

// InventoryRepository defines the interface for inventory data operations
type InventoryRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, inventory *entities.Inventory) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Inventory, error)
	GetByProductAndWarehouse(ctx context.Context, productID, warehouseID uuid.UUID) (*entities.Inventory, error)
	Update(ctx context.Context, inventory *entities.Inventory) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Stock operations
	UpdateStock(ctx context.Context, productID, warehouseID uuid.UUID, quantity int) error
	AdjustStock(ctx context.Context, productID, warehouseID uuid.UUID, adjustment int) error
	ReserveStock(ctx context.Context, productID, warehouseID uuid.UUID, quantity int) error
	ReleaseStock(ctx context.Context, productID, warehouseID uuid.UUID, quantity int) error
	GetAvailableStock(ctx context.Context, productID, warehouseID uuid.UUID) (int, error)

	// Listing and filtering
	List(ctx context.Context, filter *InventoryFilter) ([]*entities.Inventory, error)
	GetByProduct(ctx context.Context, productID uuid.UUID) ([]*entities.Inventory, error)
	GetByWarehouse(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error)
	GetWarehouseInventory(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error)
	GetProductInventory(ctx context.Context, productID uuid.UUID) ([]*entities.Inventory, error)

	// Low stock and alerts
	GetLowStockItems(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error)
	GetLowStockItemsAll(ctx context.Context) ([]*entities.Inventory, error)
	GetOutOfStockItems(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error)
	GetOverstockItems(ctx context.Context, warehouseID uuid.UUID) ([]*entities.Inventory, error)

	// Search and counting
	Search(ctx context.Context, query string, limit int) ([]*entities.Inventory, error)
	Count(ctx context.Context, filter *InventoryFilter) (int, error)
	CountByWarehouse(ctx context.Context, warehouseID uuid.UUID) (int, error)
	CountByProduct(ctx context.Context, productID uuid.UUID) (int, error)

	// Existence checks
	ExistsByProductAndWarehouse(ctx context.Context, productID, warehouseID uuid.UUID) (bool, error)

	// Bulk operations
	BulkCreate(ctx context.Context, inventories []*entities.Inventory) error
	BulkUpdate(ctx context.Context, inventories []*entities.Inventory) error
	BulkDelete(ctx context.Context, inventoryIDs []uuid.UUID) error
	BulkAdjustStock(ctx context.Context, adjustments []StockAdjustment) error
	BulkReserveStock(ctx context.Context, reservations []StockReservation) error

	// Analytics and reporting
	GetInventoryValue(ctx context.Context, warehouseID *uuid.UUID) (float64, error)
	GetInventoryLevels(ctx context.Context, productID uuid.UUID) ([]*InventoryLevel, error)
	GetStockLevels(ctx context.Context, filter *InventoryFilter) ([]*StockLevel, error)
	GetInventoryTurnover(ctx context.Context, productID uuid.UUID, warehouseID *uuid.UUID, days int) (*InventoryTurnover, error)
	GetAgingInventory(ctx context.Context, warehouseID *uuid.UUID, days int) ([]*AgingInventoryItem, error)

	// Cycle count operations
	GetItemsForCycleCount(ctx context.Context, warehouseID uuid.UUID, limit int) ([]*entities.Inventory, error)
	UpdateCycleCount(ctx context.Context, inventoryID uuid.UUID, countedQuantity int, countedBy uuid.UUID) error
	GetLastCycleCountDate(ctx context.Context, inventoryID uuid.UUID) (*time.Time, error)

	// Stock reconciliation
	ReconcileStock(ctx context.Context, inventoryID uuid.UUID, systemQuantity, physicalQuantity int, reason string, reconciledBy uuid.UUID) error
	GetReconciliationHistory(ctx context.Context, inventoryID uuid.UUID, limit int) ([]*InventoryReconciliation, error)
}

// InventoryTransactionRepository defines the interface for inventory transaction data operations
type InventoryTransactionRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, transaction *entities.InventoryTransaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.InventoryTransaction, error)
	Update(ctx context.Context, transaction *entities.InventoryTransaction) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Transaction queries
	GetByProduct(ctx context.Context, productID uuid.UUID, filter *TransactionFilter) ([]*entities.InventoryTransaction, error)
	GetByWarehouse(ctx context.Context, warehouseID uuid.UUID, filter *TransactionFilter) ([]*entities.InventoryTransaction, error)
	GetByProductAndWarehouse(ctx context.Context, productID, warehouseID uuid.UUID, filter *TransactionFilter) ([]*entities.InventoryTransaction, error)
	GetByType(ctx context.Context, transactionType entities.TransactionType, filter *TransactionFilter) ([]*entities.InventoryTransaction, error)
	GetByReference(ctx context.Context, referenceType string, referenceID uuid.UUID) ([]*entities.InventoryTransaction, error)
	GetByBatch(ctx context.Context, batchNumber string) ([]*entities.InventoryTransaction, error)

	// Date range queries
	GetByDateRange(ctx context.Context, warehouseID *uuid.UUID, startDate, endDate time.Time, filter *TransactionFilter) ([]*entities.InventoryTransaction, error)
	GetRecentTransactions(ctx context.Context, warehouseID *uuid.UUID, hours int, limit int) ([]*entities.InventoryTransaction, error)

	// Search and filtering
	Search(ctx context.Context, query string, limit int) ([]*entities.InventoryTransaction, error)
	Count(ctx context.Context, filter *TransactionFilter) (int, error)

	// Approval operations
	GetPendingApproval(ctx context.Context, warehouseID *uuid.UUID) ([]*entities.InventoryTransaction, error)
	ApproveTransaction(ctx context.Context, transactionID uuid.UUID, approvedBy uuid.UUID) error
	RejectTransaction(ctx context.Context, transactionID uuid.UUID, rejectedBy uuid.UUID, reason string) error

	// Transfer operations
	GetTransferTransactions(ctx context.Context, fromWarehouseID, toWarehouseID uuid.UUID) ([]*entities.InventoryTransaction, error)
	GetPendingTransfers(ctx context.Context, warehouseID *uuid.UUID) ([]*entities.InventoryTransaction, error)

	// Analytics and reporting
	GetTransactionSummary(ctx context.Context, filter *TransactionFilter) (*TransactionSummary, error)
	GetTransactionHistory(ctx context.Context, productID uuid.UUID, warehouseID *uuid.UUID, limit int) ([]*entities.InventoryTransaction, error)
	GetCostOfGoodsSold(ctx context.Context, startDate, endDate time.Time) (float64, error)
	GetInventoryMovement(ctx context.Context, filter *MovementFilter) ([]*InventoryMovement, error)

	// Bulk operations
	BulkCreate(ctx context.Context, transactions []*entities.InventoryTransaction) error
	BulkApprove(ctx context.Context, transactionIDs []uuid.UUID, approvedBy uuid.UUID) error

	// Audit and compliance
	GetAuditTrail(ctx context.Context, filter *AuditFilter) ([]*entities.InventoryTransaction, error)
	GetComplianceReport(ctx context.Context, startDate, endDate time.Time) (*ComplianceReport, error)
}

// InventoryFilter defines filtering options for inventory queries
type InventoryFilter struct {
	IDs           []uuid.UUID `json:"ids,omitempty"`
	ProductIDs    []uuid.UUID `json:"product_ids,omitempty"`
	WarehouseIDs  []uuid.UUID `json:"warehouse_ids,omitempty"`
	SKU           string      `json:"sku,omitempty"`
	ProductName   string      `json:"product_name,omitempty"`
	WarehouseCode string      `json:"warehouse_code,omitempty"`

	// Stock level filters
	IsLowStock   *bool `json:"is_low_stock,omitempty"`
	IsOutOfStock *bool `json:"is_out_of_stock,omitempty"`
	IsOverstock  *bool `json:"is_overstock,omitempty"`
	MinQuantity  *int  `json:"min_quantity,omitempty"`
	MaxQuantity  *int  `json:"max_quantity,omitempty"`

	// Cost filters
	MinAverageCost *float64 `json:"min_average_cost,omitempty"`
	MaxAverageCost *float64 `json:"max_average_cost,omitempty"`

	// Date filters
	LastCountedAfter  *time.Time `json:"last_counted_after,omitempty"`
	LastCountedBefore *time.Time `json:"last_counted_before,omitempty"`
	UpdatedAfter      *time.Time `json:"updated_after,omitempty"`
	UpdatedBefore     *time.Time `json:"updated_before,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Sorting
	OrderBy string `json:"order_by,omitempty"`
	Order   string `json:"order,omitempty"`
}

// TransactionFilter defines filtering options for transaction queries
type TransactionFilter struct {
	IDs              []uuid.UUID                `json:"ids,omitempty"`
	ProductIDs       []uuid.UUID                `json:"product_ids,omitempty"`
	WarehouseIDs     []uuid.UUID                `json:"warehouse_ids,omitempty"`
	TransactionTypes []entities.TransactionType `json:"transaction_types,omitempty"`
	ReferenceType    string                     `json:"reference_type,omitempty"`
	ReferenceID      *uuid.UUID                 `json:"reference_id,omitempty"`
	CreatedBy        []uuid.UUID                `json:"created_by,omitempty"`
	ApprovedBy       []uuid.UUID                `json:"approved_by,omitempty"`
	BatchNumber      string                     `json:"batch_number,omitempty"`
	SerialNumber     string                     `json:"serial_number,omitempty"`

	// Status filters
	IsApproved *bool `json:"is_approved,omitempty"`
	IsPending  *bool `json:"is_pending,omitempty"`

	// Date filters
	DateFrom       *time.Time `json:"date_from,omitempty"`
	DateTo         *time.Time `json:"date_to,omitempty"`
	CreatedAfter   *time.Time `json:"created_after,omitempty"`
	CreatedBefore  *time.Time `json:"created_before,omitempty"`
	ApprovedAfter  *time.Time `json:"approved_after,omitempty"`
	ApprovedBefore *time.Time `json:"approved_before,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Sorting
	OrderBy string `json:"order_by,omitempty"`
	Order   string `json:"order,omitempty"`
}

// StockAdjustment represents a stock adjustment operation
type StockAdjustment struct {
	ProductID   uuid.UUID `json:"product_id"`
	WarehouseID uuid.UUID `json:"warehouse_id"`
	Adjustment  int       `json:"adjustment"`
	Reason      string    `json:"reason"`
	UpdatedBy   uuid.UUID `json:"updated_by"`
}

// StockReservation represents a stock reservation operation
type StockReservation struct {
	ProductID   uuid.UUID `json:"product_id"`
	WarehouseID uuid.UUID `json:"warehouse_id"`
	Quantity    int       `json:"quantity"`
	ReservedBy  uuid.UUID `json:"reserved_by"`
}

// InventoryLevel represents inventory level information for a product
type InventoryLevel struct {
	ProductID     uuid.UUID `json:"product_id"`
	ProductName   string    `json:"product_name"`
	SKU           string    `json:"sku"`
	WarehouseID   uuid.UUID `json:"warehouse_id"`
	WarehouseName string    `json:"warehouse_name"`
	Quantity      int       `json:"quantity"`
	Reserved      int       `json:"reserved"`
	Available     int       `json:"available"`
	ReorderLevel  int       `json:"reorder_level"`
	LastUpdated   time.Time `json:"last_updated"`
}

// StockLevel represents detailed stock level information
type StockLevel struct {
	InventoryLevel
	MinStock     *int    `json:"min_stock,omitempty"`
	MaxStock     *int    `json:"max_stock,omitempty"`
	AverageCost  float64 `json:"average_cost"`
	TotalValue   float64 `json:"total_value"`
	Status       string  `json:"status"`
	DaysOfSupply float64 `json:"days_of_supply"`
}

// InventoryTurnover represents inventory turnover information
type InventoryTurnover struct {
	ProductID       uuid.UUID `json:"product_id"`
	ProductName     string    `json:"product_name"`
	WarehouseID     uuid.UUID `json:"warehouse_id"`
	WarehouseName   string    `json:"warehouse_name"`
	Days            int       `json:"days"`
	BeginningStock  int       `json:"beginning_stock"`
	EndingStock     int       `json:"ending_stock"`
	CostOfGoodsSold float64   `json:"cost_of_goods_sold"`
	TurnoverRate    float64   `json:"turnover_rate"`
	DaysOfSupply    float64   `json:"days_of_supply"`
}

// AgingInventoryItem represents an aging inventory item
type AgingInventoryItem struct {
	InventoryID          uuid.UUID  `json:"inventory_id"`
	ProductID            uuid.UUID  `json:"product_id"`
	ProductName          string     `json:"product_name"`
	SKU                  string     `json:"sku"`
	WarehouseID          uuid.UUID  `json:"warehouse_id"`
	WarehouseName        string     `json:"warehouse_name"`
	Quantity             int        `json:"quantity"`
	AverageCost          float64    `json:"average_cost"`
	TotalValue           float64    `json:"total_value"`
	LastTransaction      time.Time  `json:"last_transaction"`
	DaysSinceTransaction int        `json:"days_since_transaction"`
	BatchNumber          string     `json:"batch_number,omitempty"`
	ExpiryDate           *time.Time `json:"expiry_date,omitempty"`
	DaysToExpiry         *int       `json:"days_to_expiry,omitempty"`
}

// InventoryReconciliation represents a stock reconciliation record
type InventoryReconciliation struct {
	InventoryID      uuid.UUID  `json:"inventory_id"`
	ProductID        uuid.UUID  `json:"product_id"`
	WarehouseID      uuid.UUID  `json:"warehouse_id"`
	SystemQuantity   int        `json:"system_quantity"`
	PhysicalQuantity int        `json:"physical_quantity"`
	Variance         int        `json:"variance"`
	VarianceValue    float64    `json:"variance_value"`
	Reason           string     `json:"reason"`
	ReconciledBy     uuid.UUID  `json:"reconciled_by"`
	ReconciledAt     time.Time  `json:"reconciled_at"`
	ApprovedBy       *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
}

// TransactionSummary represents a summary of inventory transactions
type TransactionSummary struct {
	TotalTransactions  int                                                  `json:"total_transactions"`
	TotalQuantityIn    int                                                  `json:"total_quantity_in"`
	TotalQuantityOut   int                                                  `json:"total_quantity_out"`
	TotalValueIn       float64                                              `json:"total_value_in"`
	TotalValueOut      float64                                              `json:"total_value_out"`
	TransactionsByType map[entities.TransactionType]*TransactionTypeSummary `json:"transactions_by_type"`
	TopProducts        []ProductTransactionSummary                          `json:"top_products"`
	TopWarehouses      []WarehouseTransactionSummary                        `json:"top_warehouses"`
	DateRange          DateRange                                            `json:"date_range"`
}

// TransactionTypeSummary represents summary for a specific transaction type
type TransactionTypeSummary struct {
	TransactionType entities.TransactionType `json:"transaction_type"`
	Count           int                      `json:"count"`
	TotalQuantity   int                      `json:"total_quantity"`
	TotalValue      float64                  `json:"total_value"`
}

// ProductTransactionSummary represents transaction summary for a product
type ProductTransactionSummary struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	SKU         string    `json:"sku"`
	Count       int       `json:"count"`
	Quantity    int       `json:"quantity"`
	Value       float64   `json:"value"`
}

// WarehouseTransactionSummary represents transaction summary for a warehouse
type WarehouseTransactionSummary struct {
	WarehouseID   uuid.UUID `json:"warehouse_id"`
	WarehouseName string    `json:"warehouse_name"`
	WarehouseCode string    `json:"warehouse_code"`
	Count         int       `json:"count"`
	Quantity      int       `json:"quantity"`
	Value         float64   `json:"value"`
}

// MovementFilter defines filtering options for inventory movement analysis
type MovementFilter struct {
	ProductIDs   []uuid.UUID `json:"product_ids,omitempty"`
	WarehouseIDs []uuid.UUID `json:"warehouse_ids,omitempty"`
	StartDate    time.Time   `json:"start_date"`
	EndDate      time.Time   `json:"end_date"`
	GroupBy      string      `json:"group_by"` // "product", "warehouse", "type", "day"
}

// InventoryMovement represents inventory movement data
type InventoryMovement struct {
	GroupByValue string    `json:"group_by_value"`
	Date         time.Time `json:"date"`
	QuantityIn   int       `json:"quantity_in"`
	QuantityOut  int       `json:"quantity_out"`
	NetMovement  int       `json:"net_movement"`
	ValueIn      float64   `json:"value_in"`
	ValueOut     float64   `json:"value_out"`
	NetValue     float64   `json:"net_value"`
}

// AuditFilter defines filtering options for audit trail queries
type AuditFilter struct {
	UserIDs          []uuid.UUID                `json:"user_ids,omitempty"`
	ProductIDs       []uuid.UUID                `json:"product_ids,omitempty"`
	WarehouseIDs     []uuid.UUID                `json:"warehouse_ids,omitempty"`
	TransactionTypes []entities.TransactionType `json:"transaction_types,omitempty"`
	StartDate        time.Time                  `json:"start_date"`
	EndDate          time.Time                  `json:"end_date"`
	IncludeApproved  *bool                      `json:"include_approved,omitempty"`
	IncludePending   *bool                      `json:"include_pending,omitempty"`
}

// ComplianceReport represents a compliance report
type ComplianceReport struct {
	Period                 DateRange                        `json:"period"`
	TotalTransactions      int                              `json:"total_transactions"`
	ApprovedTransactions   int                              `json:"approved_transactions"`
	PendingTransactions    int                              `json:"pending_transactions"`
	HighValueTransactions  int                              `json:"high_value_transactions"`
	TransactionsByType     map[entities.TransactionType]int `json:"transactions_by_type"`
	UsersWithActivity      []UserActivitySummary            `json:"users_with_activity"`
	WarehousesWithActivity []WarehouseActivitySummary       `json:"warehouses_with_activity"`
	ComplianceScore        float64                          `json:"compliance_score"`
	Recommendations        []string                         `json:"recommendations"`
}

// UserActivitySummary represents activity summary for a user
type UserActivitySummary struct {
	UserID           uuid.UUID `json:"user_id"`
	UserName         string    `json:"user_name"`
	TransactionCount int       `json:"transaction_count"`
	TotalValue       float64   `json:"total_value"`
	LastActivity     time.Time `json:"last_activity"`
}

// WarehouseActivitySummary represents activity summary for a warehouse
type WarehouseActivitySummary struct {
	WarehouseID      uuid.UUID `json:"warehouse_id"`
	WarehouseName    string    `json:"warehouse_name"`
	TransactionCount int       `json:"transaction_count"`
	TotalValue       float64   `json:"total_value"`
	LastActivity     time.Time `json:"last_activity"`
}

// DateRange represents a date range
type DateRange struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}
