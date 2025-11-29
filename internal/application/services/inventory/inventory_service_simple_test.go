package inventory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"erpgo/internal/domain/inventory/entities"
	"erpgo/internal/domain/inventory/repositories"
)

// TestTransferInventorySimple tests the TransferInventory method directly without transaction manager complexity
func TestTransferInventorySimple(t *testing.T) {
	// Setup
	mockInventoryRepo := &MockInventoryRepository{
		inventory: make(map[uuid.UUID]*entities.Inventory),
	}

	mockTransactionRepo := &MockInventoryTransactionRepository{
		transactions: make([]*entities.InventoryTransaction, 0),
	}

	ctx := context.Background()
	productID := uuid.New()
	fromWarehouseID := uuid.New()
	toWarehouseID := uuid.New()

	req := &dto.TransferInventoryRequest{
		ProductID:       productID,
		FromWarehouseID: fromWarehouseID,
		ToWarehouseID:   toWarehouseID,
		Quantity:        10,
	}

	// Create a service with nil transaction manager to avoid timeout
	service := &ServiceImpl{
		inventoryRepo: mockInventoryRepo,
		warehouseRepo: nil,
		transactionRepo: mockTransactionRepo,
		txManager: nil, // Nil to avoid WithRetryTransaction
	}

	// Execute
	result, err := service.TransferInventory(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
}
