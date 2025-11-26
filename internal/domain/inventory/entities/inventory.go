package entities

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	apperrors "erpgo/pkg/errors"
)

// Inventory represents inventory levels for a product in a warehouse
type Inventory struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	ProductID        uuid.UUID  `json:"product_id" db:"product_id"`
	WarehouseID      uuid.UUID  `json:"warehouse_id" db:"warehouse_id"`
	QuantityOnHand   int        `json:"quantity_on_hand" db:"quantity_on_hand"`
	QuantityReserved int        `json:"quantity_reserved" db:"quantity_reserved"`
	ReorderLevel     int        `json:"reorder_level" db:"reorder_level"`
	MaxStock         *int       `json:"max_stock,omitempty" db:"max_stock"`
	MinStock         *int       `json:"min_stock,omitempty" db:"min_stock"`
	AverageCost      float64    `json:"average_cost" db:"average_cost"`
	LastCountDate    *time.Time `json:"last_count_date,omitempty" db:"last_count_date"`
	LastCountedBy    *uuid.UUID `json:"last_counted_by,omitempty" db:"last_counted_by"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	UpdatedBy        uuid.UUID  `json:"updated_by" db:"updated_by"`
}

// Validate validates the inventory entity
func (i *Inventory) Validate() error {
	var errs []error

	// Validate UUIDs
	if i.ID == uuid.Nil {
		errs = append(errs, errors.New("inventory ID cannot be empty"))
	}

	if i.ProductID == uuid.Nil {
		errs = append(errs, errors.New("product ID cannot be empty"))
	}

	if i.WarehouseID == uuid.Nil {
		errs = append(errs, errors.New("warehouse ID cannot be empty"))
	}

	if i.UpdatedBy == uuid.Nil {
		errs = append(errs, errors.New("updated by user ID cannot be empty"))
	}

	// Validate quantities
	if err := i.validateQuantities(); err != nil {
		errs = append(errs, fmt.Errorf("invalid quantities: %w", err))
	}

	// Validate stock levels
	if err := i.validateStockLevels(); err != nil {
		errs = append(errs, fmt.Errorf("invalid stock levels: %w", err))
	}

	// Validate average cost
	if err := i.validateAverageCost(); err != nil {
		errs = append(errs, fmt.Errorf("invalid average cost: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// validateQuantities validates the quantity fields
func (i *Inventory) validateQuantities() error {
	// Quantity on hand validation
	if i.QuantityOnHand < 0 {
		return errors.New("quantity on hand cannot be negative")
	}

	if i.QuantityOnHand > 999999999 {
		return errors.New("quantity on hand cannot exceed 999,999,999")
	}

	// Quantity reserved validation
	if i.QuantityReserved < 0 {
		return errors.New("quantity reserved cannot be negative")
	}

	if i.QuantityReserved > 999999999 {
		return errors.New("quantity reserved cannot exceed 999,999,999")
	}

	// Reserved quantity cannot exceed on-hand quantity
	if i.QuantityReserved > i.QuantityOnHand {
		return fmt.Errorf("reserved quantity (%d) cannot exceed on-hand quantity (%d)",
			i.QuantityReserved, i.QuantityOnHand)
	}

	return nil
}

// validateStockLevels validates the stock level settings
func (i *Inventory) validateStockLevels() error {
	// Reorder level validation
	if i.ReorderLevel < 0 {
		return errors.New("reorder level cannot be negative")
	}

	if i.ReorderLevel > 999999999 {
		return errors.New("reorder level cannot exceed 999,999,999")
	}

	// Min stock validation (optional)
	if i.MinStock != nil {
		if *i.MinStock < 0 {
			return errors.New("minimum stock cannot be negative")
		}
		if *i.MinStock > 999999999 {
			return errors.New("minimum stock cannot exceed 999,999,999")
		}
	}

	// Max stock validation (optional)
	if i.MaxStock != nil {
		if *i.MaxStock < 0 {
			return errors.New("maximum stock cannot be negative")
		}
		if *i.MaxStock > 999999999 {
			return errors.New("maximum stock cannot exceed 999,999,999")
		}
	}

	// Consistency checks between stock levels
	if i.MinStock != nil && i.MaxStock != nil {
		if *i.MinStock > *i.MaxStock {
			return errors.New("minimum stock cannot be greater than maximum stock")
		}
	}

	// Reorder level should be consistent with min/max levels
	if i.MinStock != nil && i.ReorderLevel < *i.MinStock {
		return errors.New("reorder level cannot be less than minimum stock")
	}

	if i.MaxStock != nil && i.ReorderLevel > *i.MaxStock {
		return errors.New("reorder level cannot be greater than maximum stock")
	}

	return nil
}

// validateAverageCost validates the average cost
func (i *Inventory) validateAverageCost() error {
	if i.AverageCost < 0 {
		return errors.New("average cost cannot be negative")
	}

	if i.AverageCost > 999999999.99 {
		return errors.New("average cost cannot exceed 999,999,999.99")
	}

	return nil
}

// Business Logic Methods

// GetAvailableQuantity returns the available quantity for sale
func (i *Inventory) GetAvailableQuantity() int {
	return i.QuantityOnHand - i.QuantityReserved
}

// GetTotalQuantity returns the total quantity on hand
func (i *Inventory) GetTotalQuantity() int {
	return i.QuantityOnHand
}

// GetReservedQuantity returns the reserved quantity
func (i *Inventory) GetReservedQuantity() int {
	return i.QuantityReserved
}

// IsAvailable returns true if the specified quantity is available
func (i *Inventory) IsAvailable(quantity int) error {
	if quantity <= 0 {
		return apperrors.NewBadRequestError("quantity must be positive")
	}

	available := i.GetAvailableQuantity()
	if available < quantity {
		return apperrors.NewInsufficientStockError(available, quantity)
	}

	return nil
}

// IsLowStock returns true if the inventory is at or below reorder level
func (i *Inventory) IsLowStock() bool {
	return i.QuantityOnHand <= i.ReorderLevel
}

// IsOverstock returns true if the inventory exceeds maximum stock level
func (i *Inventory) IsOverstock() bool {
	if i.MaxStock == nil {
		return false
	}
	return i.QuantityOnHand > *i.MaxStock
}

// IsUnderstock returns true if the inventory is below minimum stock level
func (i *Inventory) IsUnderstock() bool {
	if i.MinStock == nil {
		return false
	}
	return i.QuantityOnHand < *i.MinStock
}

// NeedsReorder returns true if the inventory needs to be reordered
func (i *Inventory) NeedsReorder() bool {
	return i.QuantityOnHand <= i.ReorderLevel
}

// CanFulfillOrder checks if the inventory can fulfill a given quantity
func (i *Inventory) CanFulfillOrder(quantity int) error {
	if quantity <= 0 {
		return apperrors.NewBadRequestError("order quantity must be positive")
	}

	available := i.GetAvailableQuantity()
	if available < quantity {
		return apperrors.NewInsufficientStockError(available, quantity)
	}

	return nil
}

// ReserveStock reserves a specified quantity of inventory
func (i *Inventory) ReserveStock(quantity int) error {
	if quantity <= 0 {
		return errors.New("reservation quantity must be positive")
	}

	if i.GetAvailableQuantity() < quantity {
		return fmt.Errorf("insufficient available stock: %d available, %d requested",
			i.GetAvailableQuantity(), quantity)
	}

	i.QuantityReserved += quantity
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// ReleaseStock releases a specified quantity of reserved inventory
func (i *Inventory) ReleaseStock(quantity int) error {
	if quantity <= 0 {
		return errors.New("release quantity must be positive")
	}

	if i.QuantityReserved < quantity {
		return fmt.Errorf("cannot release more than reserved: %d reserved, %d requested",
			i.QuantityReserved, quantity)
	}

	i.QuantityReserved -= quantity
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// AdjustStock adjusts the inventory quantity by a given amount
func (i *Inventory) AdjustStock(adjustment int) error {
	newQuantity := i.QuantityOnHand + adjustment

	if newQuantity < 0 {
		return errors.New("resulting quantity cannot be negative")
	}

	if newQuantity > 999999999 {
		return errors.New("resulting quantity cannot exceed 999,999,999")
	}

	// Check if adjustment would violate reserved quantity
	if i.QuantityReserved > newQuantity {
		return fmt.Errorf("adjustment would result in insufficient stock for reservations: "+
			"new quantity would be %d, but %d are reserved", newQuantity, i.QuantityReserved)
	}

	i.QuantityOnHand = newQuantity
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// SetStock sets the inventory to a specific quantity
func (i *Inventory) SetStock(newQuantity int) error {
	if newQuantity < 0 {
		return errors.New("quantity cannot be negative")
	}

	if newQuantity > 999999999 {
		return errors.New("quantity cannot exceed 999,999,999")
	}

	// Check if new quantity would violate reserved quantity
	if i.QuantityReserved > newQuantity {
		return fmt.Errorf("cannot set quantity below reserved amount: "+
			"new quantity would be %d, but %d are reserved", newQuantity, i.QuantityReserved)
	}

	i.QuantityOnHand = newQuantity
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// AddStock adds a specified quantity to inventory
func (i *Inventory) AddStock(quantity int) error {
	if quantity <= 0 {
		return errors.New("addition quantity must be positive")
	}

	newQuantity := i.QuantityOnHand + quantity
	if newQuantity > 999999999 {
		return errors.New("resulting quantity cannot exceed 999,999,999")
	}

	i.QuantityOnHand = newQuantity
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// RemoveStock removes a specified quantity from inventory
func (i *Inventory) RemoveStock(quantity int) error {
	if quantity <= 0 {
		return errors.New("removal quantity must be positive")
	}

	if i.GetAvailableQuantity() < quantity {
		return fmt.Errorf("insufficient available stock: %d available, %d requested",
			i.GetAvailableQuantity(), quantity)
	}

	// First, reduce reserved quantity if applicable
	reservedToReduce := min(quantity, i.QuantityReserved)
	i.QuantityReserved -= reservedToReduce
	remainingToRemove := quantity - reservedToReduce

	// Then reduce on-hand quantity
	i.QuantityOnHand -= remainingToRemove
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateReorderLevel updates the reorder level
func (i *Inventory) UpdateReorderLevel(reorderLevel int) error {
	if reorderLevel < 0 {
		return errors.New("reorder level cannot be negative")
	}

	if reorderLevel > 999999999 {
		return errors.New("reorder level cannot exceed 999,999,999")
	}

	// Check consistency with min/max levels
	if i.MinStock != nil && reorderLevel < *i.MinStock {
		return errors.New("reorder level cannot be less than minimum stock")
	}

	if i.MaxStock != nil && reorderLevel > *i.MaxStock {
		return errors.New("reorder level cannot be greater than maximum stock")
	}

	i.ReorderLevel = reorderLevel
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateMinStock updates the minimum stock level
func (i *Inventory) UpdateMinStock(minStock *int) error {
	if minStock != nil {
		if *minStock < 0 {
			return errors.New("minimum stock cannot be negative")
		}
		if *minStock > 999999999 {
			return errors.New("minimum stock cannot exceed 999,999,999")
		}

		// Check consistency with max stock
		if i.MaxStock != nil && *minStock > *i.MaxStock {
			return errors.New("minimum stock cannot be greater than maximum stock")
		}

		// Check consistency with reorder level
		if i.ReorderLevel < *minStock {
			return errors.New("reorder level cannot be less than minimum stock")
		}
	}

	i.MinStock = minStock
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateMaxStock updates the maximum stock level
func (i *Inventory) UpdateMaxStock(maxStock *int) error {
	if maxStock != nil {
		if *maxStock < 0 {
			return errors.New("maximum stock cannot be negative")
		}
		if *maxStock > 999999999 {
			return errors.New("maximum stock cannot exceed 999,999,999")
		}

		// Check consistency with min stock
		if i.MinStock != nil && *maxStock < *i.MinStock {
			return errors.New("maximum stock cannot be less than minimum stock")
		}

		// Check consistency with reorder level
		if i.ReorderLevel > *maxStock {
			return errors.New("reorder level cannot be greater than maximum stock")
		}
	}

	i.MaxStock = maxStock
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateAverageCost updates the average cost
func (i *Inventory) UpdateAverageCost(newCost float64) error {
	if newCost < 0 {
		return errors.New("average cost cannot be negative")
	}

	if newCost > 999999999.99 {
		return errors.New("average cost cannot exceed 999,999,999.99")
	}

	i.AverageCost = newCost
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateStockLevels updates all stock levels at once
func (i *Inventory) UpdateStockLevels(minStock, maxStock, reorderLevel *int) error {
	// Validate min stock
	if minStock != nil {
		if *minStock < 0 {
			return errors.New("minimum stock cannot be negative")
		}
		if *minStock > 999999999 {
			return errors.New("minimum stock cannot exceed 999,999,999")
		}
	}

	// Validate max stock
	if maxStock != nil {
		if *maxStock < 0 {
			return errors.New("maximum stock cannot be negative")
		}
		if *maxStock > 999999999 {
			return errors.New("maximum stock cannot exceed 999,999,999")
		}
	}

	// Validate reorder level
	if reorderLevel != nil {
		if *reorderLevel < 0 {
			return errors.New("reorder level cannot be negative")
		}
		if *reorderLevel > 999999999 {
			return errors.New("reorder level cannot exceed 999,999,999")
		}
	}

	// Check consistency between levels
	if minStock != nil && maxStock != nil && *minStock > *maxStock {
		return errors.New("minimum stock cannot be greater than maximum stock")
	}

	if minStock != nil && reorderLevel != nil && *reorderLevel < *minStock {
		return errors.New("reorder level cannot be less than minimum stock")
	}

	if maxStock != nil && reorderLevel != nil && *reorderLevel > *maxStock {
		return errors.New("reorder level cannot be greater than maximum stock")
	}

	// Apply updates
	if minStock != nil {
		i.MinStock = minStock
	}
	if maxStock != nil {
		i.MaxStock = maxStock
	}
	if reorderLevel != nil {
		i.ReorderLevel = *reorderLevel
	}

	i.UpdatedAt = time.Now().UTC()
	return nil
}

// RecordCycleCount records a cycle count
func (i *Inventory) RecordCycleCount(countedQuantity int, countedBy uuid.UUID) error {
	if countedQuantity < 0 {
		return errors.New("counted quantity cannot be negative")
	}

	if countedBy == uuid.Nil {
		return errors.New("counted by user ID cannot be empty")
	}

	if countedQuantity > 999999999 {
		return errors.New("counted quantity cannot exceed 999,999,999")
	}

	i.QuantityOnHand = countedQuantity
	i.LastCountDate = &time.Time{}
	*i.LastCountDate = time.Now().UTC()
	i.LastCountedBy = &countedBy
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// GetStockStatus returns a descriptive stock status
func (i *Inventory) GetStockStatus() string {
	if i.QuantityOnHand == 0 {
		return "OUT_OF_STOCK"
	}
	if i.IsUnderstock() {
		return "UNDERSTOCK"
	}
	if i.IsLowStock() {
		return "LOW_STOCK"
	}
	if i.IsOverstock() {
		return "OVERSTOCK"
	}
	return "NORMAL"
}

// GetDaysOfSupply calculates the days of supply based on average daily usage
func (i *Inventory) GetDaysOfSupply(averageDailyUsage float64) (float64, error) {
	if averageDailyUsage <= 0 {
		return 0, errors.New("average daily usage must be positive")
	}

	availableQuantity := float64(i.GetAvailableQuantity())
	daysOfSupply := availableQuantity / averageDailyUsage
	return daysOfSupply, nil
}

// GetReorderQuantity calculates the suggested reorder quantity
func (i *Inventory) GetReorderQuantity(economicOrderQuantity int) (int, error) {
	if economicOrderQuantity <= 0 {
		return 0, errors.New("economic order quantity must be positive")
	}

	// If no max stock is set, use economic order quantity
	if i.MaxStock == nil {
		return economicOrderQuantity, nil
	}

	// Calculate space available up to max stock
	availableSpace := *i.MaxStock - i.QuantityOnHand
	if availableSpace <= 0 {
		return 0, errors.New("warehouse at or above maximum stock level")
	}

	// Return the lesser of EOQ and available space
	if economicOrderQuantity <= availableSpace {
		return economicOrderQuantity, nil
	}

	return availableSpace, nil
}

// ToSafeInventory returns an inventory object without sensitive information
func (i *Inventory) ToSafeInventory() *Inventory {
	return &Inventory{
		ID:               i.ID,
		ProductID:        i.ProductID,
		WarehouseID:      i.WarehouseID,
		QuantityOnHand:   i.QuantityOnHand,
		QuantityReserved: i.QuantityReserved,
		ReorderLevel:     i.ReorderLevel,
		MaxStock:         i.MaxStock,
		MinStock:         i.MinStock,
		AverageCost:      i.AverageCost,
		LastCountDate:    i.LastCountDate,
		UpdatedAt:        i.UpdatedAt,
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}