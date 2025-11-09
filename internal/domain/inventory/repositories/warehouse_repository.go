package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"erpgo/internal/domain/inventory/entities"
)

// WarehouseRepository defines the interface for warehouse data operations
type WarehouseRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, warehouse *entities.Warehouse) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Warehouse, error)
	GetByCode(ctx context.Context, code string) (*entities.Warehouse, error)
	Update(ctx context.Context, warehouse *entities.Warehouse) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Listing and filtering
	List(ctx context.Context, filter *WarehouseFilter) ([]*entities.Warehouse, error)
	GetActive(ctx context.Context) ([]*entities.Warehouse, error)
	GetByManager(ctx context.Context, managerID uuid.UUID) ([]*entities.Warehouse, error)
	GetByLocation(ctx context.Context, city, state, country string) ([]*entities.Warehouse, error)

	// Search and filtering
	Search(ctx context.Context, query string, limit int) ([]*entities.Warehouse, error)
	Count(ctx context.Context, filter *WarehouseFilter) (int, error)

	// Existence checks
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)
	ExistsByCode(ctx context.Context, code string) (bool, error)

	// Bulk operations
	BulkUpdateStatus(ctx context.Context, warehouseIDs []uuid.UUID, isActive bool) error
	BulkAssignManager(ctx context.Context, warehouseIDs []uuid.UUID, managerID uuid.UUID) error

	// Analytics and reporting
	GetWarehouseStats(ctx context.Context, warehouseID uuid.UUID) (*WarehouseStats, error)
	GetAllWarehouseStats(ctx context.Context) ([]*WarehouseStats, error)
	GetCapacityUtilization(ctx context.Context, warehouseID uuid.UUID) (*CapacityUtilization, error)

	// Extended warehouse operations (for WarehouseExtended entities)
	CreateExtended(ctx context.Context, warehouse *entities.WarehouseExtended) error
	GetExtendedByID(ctx context.Context, id uuid.UUID) (*entities.WarehouseExtended, error)
	UpdateExtended(ctx context.Context, warehouse *entities.WarehouseExtended) error
	GetByType(ctx context.Context, warehouseType entities.WarehouseType) ([]*entities.WarehouseExtended, error)
}

// WarehouseFilter defines filtering options for warehouse queries
type WarehouseFilter struct {
	// Basic filters
	IDs       []uuid.UUID `json:"ids,omitempty"`
	Code      string      `json:"code,omitempty"`
	Name      string      `json:"name,omitempty"`
	IsActive  *bool       `json:"is_active,omitempty"`
	ManagerID *uuid.UUID  `json:"manager_id,omitempty"`

	// Location filters
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`
	Country string `json:"country,omitempty"`

	// Type filters (for extended warehouses)
	Type *entities.WarehouseType `json:"type,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Sorting
	OrderBy string `json:"order_by,omitempty"`
	Order   string `json:"order,omitempty"` // "asc" or "desc"

	// Date filters
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	UpdatedAfter  *time.Time `json:"updated_after,omitempty"`
	UpdatedBefore *time.Time `json:"updated_before,omitempty"`
}

// WarehouseStats contains statistics for a warehouse
type WarehouseStats struct {
	WarehouseID        uuid.UUID `json:"warehouse_id"`
	WarehouseName      string    `json:"warehouse_name"`
	WarehouseCode      string    `json:"warehouse_code"`
	TotalProducts      int       `json:"total_products"`
	TotalQuantity      int       `json:"total_quantity"`
	TotalValue         float64   `json:"total_value"`
	LowStockProducts   int       `json:"low_stock_products"`
	OutOfStockProducts int       `json:"out_of_stock_products"`
	LastUpdated        time.Time `json:"last_updated"`
}

// CapacityUtilization contains capacity utilization information
type CapacityUtilization struct {
	WarehouseID       uuid.UUID `json:"warehouse_id"`
	WarehouseName     string    `json:"warehouse_name"`
	WarehouseCode     string    `json:"warehouse_code"`
	Capacity          *int      `json:"capacity,omitempty"`
	CurrentStock      int       `json:"current_stock"`
	UtilizationPercent float64  `json:"utilization_percent"`
	AvailableSpace    *int      `json:"available_space,omitempty"`
	LastCalculated    time.Time `json:"last_calculated"`
}

// WarehouseSortUpdate defines a sort order update for warehouses
type WarehouseSortUpdate struct {
	WarehouseID uuid.UUID `json:"warehouse_id"`
	SortOrder   int       `json:"sort_order"`
}