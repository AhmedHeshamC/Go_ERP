package entities

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestInventory_ComprehensiveCoverage tests all inventory methods for complete coverage
func TestInventory_ComprehensiveCoverage(t *testing.T) {
	inventoryID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	updatedBy := uuid.New()

	t.Run("GetTotalQuantity", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 20,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		total := inventory.GetTotalQuantity()
		if total != 100 {
			t.Errorf("Expected total quantity 100, got %d", total)
		}
	})

	t.Run("GetReservedQuantity", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 20,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		reserved := inventory.GetReservedQuantity()
		if reserved != 20 {
			t.Errorf("Expected reserved quantity 20, got %d", reserved)
		}
	})

	t.Run("IsOverstock", func(t *testing.T) {
		maxStock := 50
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   60,
			QuantityReserved: 0,
			ReorderLevel:     10,
			MaxStock:         &maxStock,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		if !inventory.IsOverstock() {
			t.Error("Expected inventory to be overstock")
		}

		inventory.QuantityOnHand = 40
		if inventory.IsOverstock() {
			t.Error("Expected inventory to not be overstock")
		}

		// Test with no max stock set
		inventory.MaxStock = nil
		if inventory.IsOverstock() {
			t.Error("Expected inventory with no max stock to not be overstock")
		}
	})

	t.Run("NeedsReorder", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   25,
			QuantityReserved: 0,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		if !inventory.NeedsReorder() {
			t.Error("Expected inventory to need reorder")
		}

		inventory.QuantityOnHand = 35
		if inventory.NeedsReorder() {
			t.Error("Expected inventory to not need reorder")
		}
	})

	t.Run("CanFulfillOrder", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 20,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Can fulfill
		err := inventory.CanFulfillOrder(50)
		if err != nil {
			t.Errorf("Expected to fulfill order, got error: %v", err)
		}

		// Cannot fulfill - insufficient stock
		err = inventory.CanFulfillOrder(100)
		if err == nil {
			t.Error("Expected error for insufficient stock")
		}

		// Invalid quantity
		err = inventory.CanFulfillOrder(0)
		if err == nil {
			t.Error("Expected error for zero quantity")
		}
	})

	t.Run("AdjustStock", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 20,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Positive adjustment
		err := inventory.AdjustStock(50)
		if err != nil {
			t.Errorf("Expected successful adjustment, got error: %v", err)
		}
		if inventory.QuantityOnHand != 150 {
			t.Errorf("Expected quantity 150, got %d", inventory.QuantityOnHand)
		}

		// Negative adjustment
		err = inventory.AdjustStock(-30)
		if err != nil {
			t.Errorf("Expected successful adjustment, got error: %v", err)
		}
		if inventory.QuantityOnHand != 120 {
			t.Errorf("Expected quantity 120, got %d", inventory.QuantityOnHand)
		}

		// Adjustment resulting in negative quantity
		err = inventory.AdjustStock(-150)
		if err == nil {
			t.Error("Expected error for negative resulting quantity")
		}

		// Adjustment exceeding max
		err = inventory.AdjustStock(999999999)
		if err == nil {
			t.Error("Expected error for exceeding max quantity")
		}

		// Adjustment violating reserved quantity
		inventory.QuantityOnHand = 50
		inventory.QuantityReserved = 40
		err = inventory.AdjustStock(-30)
		if err == nil {
			t.Error("Expected error for violating reserved quantity")
		}
	})

	t.Run("SetStock", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 20,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Valid set
		err := inventory.SetStock(150)
		if err != nil {
			t.Errorf("Expected successful set, got error: %v", err)
		}
		if inventory.QuantityOnHand != 150 {
			t.Errorf("Expected quantity 150, got %d", inventory.QuantityOnHand)
		}

		// Negative quantity
		err = inventory.SetStock(-10)
		if err == nil {
			t.Error("Expected error for negative quantity")
		}

		// Exceeding max
		err = inventory.SetStock(1000000000)
		if err == nil {
			t.Error("Expected error for exceeding max")
		}

		// Below reserved quantity
		inventory.QuantityReserved = 50
		err = inventory.SetStock(40)
		if err == nil {
			t.Error("Expected error for setting below reserved quantity")
		}
	})

	t.Run("UpdateReorderLevel", func(t *testing.T) {
		minStock := 10
		maxStock := 100
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   50,
			QuantityReserved: 0,
			ReorderLevel:     20,
			MinStock:         &minStock,
			MaxStock:         &maxStock,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Valid update
		err := inventory.UpdateReorderLevel(30)
		if err != nil {
			t.Errorf("Expected successful update, got error: %v", err)
		}
		if inventory.ReorderLevel != 30 {
			t.Errorf("Expected reorder level 30, got %d", inventory.ReorderLevel)
		}

		// Negative reorder level
		err = inventory.UpdateReorderLevel(-5)
		if err == nil {
			t.Error("Expected error for negative reorder level")
		}

		// Exceeding max
		err = inventory.UpdateReorderLevel(1000000000)
		if err == nil {
			t.Error("Expected error for exceeding max")
		}

		// Less than min stock
		err = inventory.UpdateReorderLevel(5)
		if err == nil {
			t.Error("Expected error for reorder level less than min stock")
		}

		// Greater than max stock
		err = inventory.UpdateReorderLevel(150)
		if err == nil {
			t.Error("Expected error for reorder level greater than max stock")
		}
	})

	t.Run("UpdateMinStock", func(t *testing.T) {
		maxStock := 100
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   50,
			QuantityReserved: 0,
			ReorderLevel:     20,
			MaxStock:         &maxStock,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Valid update
		minStock := 15
		err := inventory.UpdateMinStock(&minStock)
		if err != nil {
			t.Errorf("Expected successful update, got error: %v", err)
		}
		if *inventory.MinStock != 15 {
			t.Errorf("Expected min stock 15, got %d", *inventory.MinStock)
		}

		// Set to nil
		err = inventory.UpdateMinStock(nil)
		if err != nil {
			t.Errorf("Expected successful update to nil, got error: %v", err)
		}

		// Negative min stock
		negativeMin := -5
		err = inventory.UpdateMinStock(&negativeMin)
		if err == nil {
			t.Error("Expected error for negative min stock")
		}

		// Exceeding max
		exceedingMin := 1000000000
		err = inventory.UpdateMinStock(&exceedingMin)
		if err == nil {
			t.Error("Expected error for exceeding max")
		}

		// Greater than max stock
		greaterThanMax := 150
		err = inventory.UpdateMinStock(&greaterThanMax)
		if err == nil {
			t.Error("Expected error for min stock greater than max stock")
		}

		// Reorder level less than min stock
		inventory.ReorderLevel = 10
		tooHighMin := 20
		err = inventory.UpdateMinStock(&tooHighMin)
		if err == nil {
			t.Error("Expected error for reorder level less than min stock")
		}
	})

	t.Run("UpdateMaxStock", func(t *testing.T) {
		minStock := 10
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   50,
			QuantityReserved: 0,
			ReorderLevel:     20,
			MinStock:         &minStock,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Valid update
		maxStock := 100
		err := inventory.UpdateMaxStock(&maxStock)
		if err != nil {
			t.Errorf("Expected successful update, got error: %v", err)
		}
		if *inventory.MaxStock != 100 {
			t.Errorf("Expected max stock 100, got %d", *inventory.MaxStock)
		}

		// Set to nil
		err = inventory.UpdateMaxStock(nil)
		if err != nil {
			t.Errorf("Expected successful update to nil, got error: %v", err)
		}

		// Negative max stock
		negativeMax := -5
		err = inventory.UpdateMaxStock(&negativeMax)
		if err == nil {
			t.Error("Expected error for negative max stock")
		}

		// Exceeding max
		exceedingMax := 1000000000
		err = inventory.UpdateMaxStock(&exceedingMax)
		if err == nil {
			t.Error("Expected error for exceeding max")
		}

		// Less than min stock
		lessThanMin := 5
		err = inventory.UpdateMaxStock(&lessThanMin)
		if err == nil {
			t.Error("Expected error for max stock less than min stock")
		}

		// Reorder level greater than max stock
		inventory.ReorderLevel = 30
		tooLowMax := 25
		err = inventory.UpdateMaxStock(&tooLowMax)
		if err == nil {
			t.Error("Expected error for reorder level greater than max stock")
		}
	})

	t.Run("UpdateAverageCost", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 0,
			ReorderLevel:     20,
			AverageCost:      25.50,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Valid update
		err := inventory.UpdateAverageCost(30.00)
		if err != nil {
			t.Errorf("Expected successful update, got error: %v", err)
		}
		if inventory.AverageCost != 30.00 {
			t.Errorf("Expected average cost 30.00, got %.2f", inventory.AverageCost)
		}

		// Negative cost
		err = inventory.UpdateAverageCost(-5.00)
		if err == nil {
			t.Error("Expected error for negative cost")
		}

		// Exceeding max
		err = inventory.UpdateAverageCost(1000000000.00)
		if err == nil {
			t.Error("Expected error for exceeding max cost")
		}
	})

	t.Run("UpdateStockLevels", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 0,
			ReorderLevel:     20,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Valid update
		minStock := 10
		maxStock := 200
		reorderLevel := 30
		err := inventory.UpdateStockLevels(&minStock, &maxStock, &reorderLevel)
		if err != nil {
			t.Errorf("Expected successful update, got error: %v", err)
		}
		if *inventory.MinStock != 10 || *inventory.MaxStock != 200 || inventory.ReorderLevel != 30 {
			t.Error("Stock levels not updated correctly")
		}

		// Invalid: min > max
		invalidMin := 250
		err = inventory.UpdateStockLevels(&invalidMin, &maxStock, nil)
		if err == nil {
			t.Error("Expected error for min > max")
		}

		// Invalid: reorder < min
		invalidReorder := 5
		err = inventory.UpdateStockLevels(&minStock, &maxStock, &invalidReorder)
		if err == nil {
			t.Error("Expected error for reorder < min")
		}

		// Invalid: reorder > max
		invalidReorder = 250
		err = inventory.UpdateStockLevels(&minStock, &maxStock, &invalidReorder)
		if err == nil {
			t.Error("Expected error for reorder > max")
		}
	})

	t.Run("RecordCycleCount", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 0,
			ReorderLevel:     20,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		countedBy := uuid.New()

		// Valid cycle count
		err := inventory.RecordCycleCount(95, countedBy)
		if err != nil {
			t.Errorf("Expected successful cycle count, got error: %v", err)
		}
		if inventory.QuantityOnHand != 95 {
			t.Errorf("Expected quantity 95, got %d", inventory.QuantityOnHand)
		}
		if inventory.LastCountDate == nil {
			t.Error("Expected last count date to be set")
		}
		if inventory.LastCountedBy == nil || *inventory.LastCountedBy != countedBy {
			t.Error("Expected last counted by to be set correctly")
		}

		// Negative count
		err = inventory.RecordCycleCount(-5, countedBy)
		if err == nil {
			t.Error("Expected error for negative count")
		}

		// Empty counted by
		err = inventory.RecordCycleCount(100, uuid.Nil)
		if err == nil {
			t.Error("Expected error for empty counted by")
		}

		// Exceeding max
		err = inventory.RecordCycleCount(1000000000, countedBy)
		if err == nil {
			t.Error("Expected error for exceeding max")
		}
	})

	t.Run("GetDaysOfSupply", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 20,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Valid calculation
		days, err := inventory.GetDaysOfSupply(10.0)
		if err != nil {
			t.Errorf("Expected successful calculation, got error: %v", err)
		}
		if days != 8.0 { // 80 available / 10 per day
			t.Errorf("Expected 8 days of supply, got %.2f", days)
		}

		// Zero or negative usage
		_, err = inventory.GetDaysOfSupply(0)
		if err == nil {
			t.Error("Expected error for zero usage")
		}

		_, err = inventory.GetDaysOfSupply(-5)
		if err == nil {
			t.Error("Expected error for negative usage")
		}
	})

	t.Run("GetReorderQuantity", func(t *testing.T) {
		maxStock := 200
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   50,
			QuantityReserved: 0,
			ReorderLevel:     30,
			MaxStock:         &maxStock,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// EOQ fits within available space
		qty, err := inventory.GetReorderQuantity(100)
		if err != nil {
			t.Errorf("Expected successful calculation, got error: %v", err)
		}
		if qty != 100 {
			t.Errorf("Expected reorder quantity 100, got %d", qty)
		}

		// EOQ exceeds available space
		qty, err = inventory.GetReorderQuantity(200)
		if err != nil {
			t.Errorf("Expected successful calculation, got error: %v", err)
		}
		if qty != 150 { // 200 max - 50 on hand
			t.Errorf("Expected reorder quantity 150, got %d", qty)
		}

		// No max stock set
		inventory.MaxStock = nil
		qty, err = inventory.GetReorderQuantity(100)
		if err != nil {
			t.Errorf("Expected successful calculation, got error: %v", err)
		}
		if qty != 100 {
			t.Errorf("Expected reorder quantity 100, got %d", qty)
		}

		// At or above max stock
		inventory.MaxStock = &maxStock
		inventory.QuantityOnHand = 200
		_, err = inventory.GetReorderQuantity(50)
		if err == nil {
			t.Error("Expected error for warehouse at max stock")
		}

		// Invalid EOQ
		_, err = inventory.GetReorderQuantity(0)
		if err == nil {
			t.Error("Expected error for zero EOQ")
		}
	})

	t.Run("ToSafeInventory", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 20,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		safe := inventory.ToSafeInventory()
		if safe.ID != inventory.ID {
			t.Error("Safe inventory ID mismatch")
		}
		if safe.UpdatedBy != uuid.Nil {
			t.Error("Expected UpdatedBy to be removed in safe inventory")
		}
	})

	t.Run("GetStockStatus", func(t *testing.T) {
		minStock := 20
		maxStock := 200
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   0,
			QuantityReserved: 0,
			ReorderLevel:     30,
			MinStock:         &minStock,
			MaxStock:         &maxStock,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Out of stock
		status := inventory.GetStockStatus()
		if status != "OUT_OF_STOCK" {
			t.Errorf("Expected OUT_OF_STOCK, got %s", status)
		}

		// Understock
		inventory.QuantityOnHand = 15
		status = inventory.GetStockStatus()
		if status != "UNDERSTOCK" {
			t.Errorf("Expected UNDERSTOCK, got %s", status)
		}

		// Low stock
		inventory.QuantityOnHand = 25
		status = inventory.GetStockStatus()
		if status != "LOW_STOCK" {
			t.Errorf("Expected LOW_STOCK, got %s", status)
		}

		// Overstock
		inventory.QuantityOnHand = 250
		status = inventory.GetStockStatus()
		if status != "OVERSTOCK" {
			t.Errorf("Expected OVERSTOCK, got %s", status)
		}

		// Normal
		inventory.QuantityOnHand = 100
		status = inventory.GetStockStatus()
		if status != "NORMAL" {
			t.Errorf("Expected NORMAL, got %s", status)
		}
	})

	t.Run("IsUnderstock", func(t *testing.T) {
		minStock := 20
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   15,
			QuantityReserved: 0,
			ReorderLevel:     30,
			MinStock:         &minStock,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		if !inventory.IsUnderstock() {
			t.Error("Expected inventory to be understock")
		}

		inventory.QuantityOnHand = 25
		if inventory.IsUnderstock() {
			t.Error("Expected inventory to not be understock")
		}

		// No min stock set
		inventory.MinStock = nil
		if inventory.IsUnderstock() {
			t.Error("Expected inventory with no min stock to not be understock")
		}
	})

	t.Run("AddStock", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 20,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Valid addition
		err := inventory.AddStock(50)
		if err != nil {
			t.Errorf("Expected successful addition, got error: %v", err)
		}
		if inventory.QuantityOnHand != 150 {
			t.Errorf("Expected quantity 150, got %d", inventory.QuantityOnHand)
		}

		// Zero or negative quantity
		err = inventory.AddStock(0)
		if err == nil {
			t.Error("Expected error for zero quantity")
		}

		err = inventory.AddStock(-10)
		if err == nil {
			t.Error("Expected error for negative quantity")
		}

		// Exceeding max
		err = inventory.AddStock(999999999)
		if err == nil {
			t.Error("Expected error for exceeding max")
		}
	})

	t.Run("RemoveStock", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 20,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Valid removal - removes from available (on-hand - reserved)
		// Available = 100 - 20 = 80
		// Removing 30 should: reduce reserved by 20, then on-hand by 10
		err := inventory.RemoveStock(30)
		if err != nil {
			t.Errorf("Expected successful removal, got error: %v", err)
		}
		// Reserved reduced by min(30, 20) = 20, so reserved = 0
		// Remaining to remove = 30 - 20 = 10, so on-hand = 100 - 10 = 90
		if inventory.QuantityReserved != 0 {
			t.Errorf("Expected reserved 0, got %d", inventory.QuantityReserved)
		}
		if inventory.QuantityOnHand != 90 {
			t.Errorf("Expected on-hand 90, got %d", inventory.QuantityOnHand)
		}

		// Zero or negative quantity
		err = inventory.RemoveStock(0)
		if err == nil {
			t.Error("Expected error for zero quantity")
		}

		err = inventory.RemoveStock(-10)
		if err == nil {
			t.Error("Expected error for negative quantity")
		}

		// Insufficient available stock
		// Available = 90 - 0 = 90
		err = inventory.RemoveStock(100)
		if err == nil {
			t.Error("Expected error for insufficient stock")
		}

		// Test removal with reserved quantity
		inventory.QuantityOnHand = 100
		inventory.QuantityReserved = 30
		// Available = 100 - 30 = 70
		err = inventory.RemoveStock(70)
		if err != nil {
			t.Errorf("Expected successful removal, got error: %v", err)
		}
		// Should reduce reserved first by min(70, 30) = 30, so reserved = 0
		// Remaining to remove = 70 - 30 = 40, so on-hand = 100 - 40 = 60
		if inventory.QuantityReserved != 0 {
			t.Errorf("Expected reserved=0, got reserved=%d", inventory.QuantityReserved)
		}
		if inventory.QuantityOnHand != 60 {
			t.Errorf("Expected on-hand=60, got on-hand=%d", inventory.QuantityOnHand)
		}
	})

	t.Run("ReserveStock", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 20,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Valid reservation
		err := inventory.ReserveStock(30)
		if err != nil {
			t.Errorf("Expected successful reservation, got error: %v", err)
		}
		if inventory.QuantityReserved != 50 {
			t.Errorf("Expected reserved 50, got %d", inventory.QuantityReserved)
		}

		// Zero or negative quantity
		err = inventory.ReserveStock(0)
		if err == nil {
			t.Error("Expected error for zero quantity")
		}

		err = inventory.ReserveStock(-10)
		if err == nil {
			t.Error("Expected error for negative quantity")
		}

		// Insufficient available stock
		err = inventory.ReserveStock(100)
		if err == nil {
			t.Error("Expected error for insufficient available stock")
		}
	})

	t.Run("ReleaseStock", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 50,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Valid release
		err := inventory.ReleaseStock(20)
		if err != nil {
			t.Errorf("Expected successful release, got error: %v", err)
		}
		if inventory.QuantityReserved != 30 {
			t.Errorf("Expected reserved 30, got %d", inventory.QuantityReserved)
		}

		// Zero or negative quantity
		err = inventory.ReleaseStock(0)
		if err == nil {
			t.Error("Expected error for zero quantity")
		}

		err = inventory.ReleaseStock(-10)
		if err == nil {
			t.Error("Expected error for negative quantity")
		}

		// Release more than reserved
		err = inventory.ReleaseStock(100)
		if err == nil {
			t.Error("Expected error for releasing more than reserved")
		}
	})

	t.Run("IsAvailable", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 20,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Available quantity
		err := inventory.IsAvailable(50)
		if err != nil {
			t.Errorf("Expected quantity to be available, got error: %v", err)
		}

		// Not available
		err = inventory.IsAvailable(100)
		if err == nil {
			t.Error("Expected error for insufficient stock")
		}

		// Zero or negative quantity
		err = inventory.IsAvailable(0)
		if err == nil {
			t.Error("Expected error for zero quantity")
		}

		err = inventory.IsAvailable(-10)
		if err == nil {
			t.Error("Expected error for negative quantity")
		}
	})
}
