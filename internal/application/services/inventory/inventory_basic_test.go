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
		minStock := 10
		maxStock := 1000
		lastCountDate := time.Now()
		updatedBy := uuid.New()

		inventory := &entities.Inventory{
			ID:               uuid.New(),
			ProductID:        uuid.New(),
			WarehouseID:      uuid.New(),
			QuantityOnHand:   100,
			QuantityReserved: 0,
			ReorderLevel:     25,
			MinStock:         &minStock,
			MaxStock:         &maxStock,
			AverageCost:      10.50,
			LastCountDate:    &lastCountDate,
			UpdatedBy:        updatedBy,
			UpdatedAt:        time.Now().UTC(),
		}

		err := inventory.Validate()
		if err != nil {
			t.Errorf("Inventory validation failed: %v", err)
		}

		// Test basic properties
		t.Logf("Created inventory: %+v", inventory)

		// Test that we can create a valid inventory
		assert.True(t, inventory.QuantityOnHand > 0)
		assert.False(t, inventory.QuantityOnHand > maxStock)
		assert.False(t, inventory.QuantityOnHand < inventory.ReorderLevel)
		assert.True(t, inventory.QuantityOnHand >= inventory.ReorderLevel)

		// Test stock calculations
		assert.Equal(t, inventory.QuantityOnHand, 100)
	})
}
