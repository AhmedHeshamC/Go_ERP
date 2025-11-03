package entities

import (
	"time"

	"github.com/google/uuid"
)

// Product represents a product in the system
type Product struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	SKU         string     `json:"sku" db:"sku"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description,omitempty" db:"description"`
	CategoryID  *uuid.UUID `json:"category_id,omitempty" db:"category_id"`
	Price       float64    `json:"price" db:"price"`
	Cost        *float64   `json:"cost,omitempty" db:"cost"`
	Weight      *float64   `json:"weight,omitempty" db:"weight"`
	Dimensions  string     `json:"dimensions,omitempty" db:"dimensions"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// Category represents a product category
type Category struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description,omitempty" db:"description"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// Warehouse represents a storage location
type Warehouse struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Code        string    `json:"code" db:"code"`
	Address     string    `json:"address" db:"address"`
	City        string    `json:"city" db:"city"`
	State       string    `json:"state" db:"state"`
	Country     string    `json:"country" db:"country"`
	PostalCode  string    `json:"postal_code" db:"postal_code"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Inventory represents inventory levels for a product in a warehouse
type Inventory struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	ProductID         uuid.UUID  `json:"product_id" db:"product_id"`
	WarehouseID       uuid.UUID  `json:"warehouse_id" db:"warehouse_id"`
	QuantityAvailable int        `json:"quantity_available" db:"quantity_available"`
	QuantityReserved  int        `json:"quantity_reserved" db:"quantity_reserved"`
	ReorderLevel      int        `json:"reorder_level" db:"reorder_level"`
	MaxStock          *int       `json:"max_stock,omitempty" db:"max_stock"`
	LastUpdatedAt     time.Time  `json:"last_updated_at" db:"last_updated_at"`
	UpdatedBy         uuid.UUID  `json:"updated_by" db:"updated_by"`
}

// InventoryTransaction represents a movement of inventory
type InventoryTransaction struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	ProductID     uuid.UUID  `json:"product_id" db:"product_id"`
	WarehouseID   uuid.UUID  `json:"warehouse_id" db:"warehouse_id"`
	TransactionType string   `json:"transaction_type" db:"transaction_type"` // 'IN', 'OUT', 'ADJUST', 'TRANSFER'
	Quantity      int        `json:"quantity" db:"quantity"`
	ReferenceID   *uuid.UUID `json:"reference_id,omitempty" db:"reference_id"`
	Reason        string     `json:"reason,omitempty" db:"reason"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	CreatedBy     uuid.UUID  `json:"created_by" db:"created_by"`
}