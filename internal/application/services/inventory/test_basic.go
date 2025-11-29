package inventory

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"erpgo/internal/domain/inventory/entities"
)

func TestBasicInventory(t *testing.T) {
	t.Run("can create inventory", func(t *testing.T) {
		inventory := &entities.Inventory{
			ID:              entities.NewInventoryID(),
			ProductID:       entities.NewProductID(),
			WarehouseID:     entities.NewWarehouseID(),
			Quantity:        100,
			MinStock:        10,
			MaxStock:        1000,
			AverageCost:     entities.NewDecimalFromFloat64(10.50),
			TotalValue:      entities.NewDecimalFromFloat64(1050.00),
			ReorderLevel:    25,
			ReorderQuantity:  50,
			LastStockCheck:  entities.Now(),
			CreatedAt:       entities.Now(),
			UpdatedAt:       entities.Now(),
		}

		err := inventory.Validate()
		if err != nil {
			t.Errorf("Inventory validation failed: %v", err)
		}

		// Test basic properties
		t.Logf("Created inventory: %+v", inventory)

		// Test that we can create a valid inventory
		assert.True(t, inventory.IsInStock())
		assert.False(t, inventory.IsOverstock())
		assert.False(t, inventory.IsLowStock())
		assert.True(t, inventory.NeedsReorder())

		// Test stock calculations
		assert.Equal(t, inventory.GetAvailableStock(), 100)
	})
}
