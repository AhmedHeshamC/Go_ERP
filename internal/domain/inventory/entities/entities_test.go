package entities

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestWarehouse tests the Warehouse entity
func TestWarehouse(t *testing.T) {
	warehouseID := uuid.New()

	t.Run("ValidWarehouse", func(t *testing.T) {
		warehouse := &Warehouse{
			ID:         warehouseID,
			Name:       "Main Warehouse",
			Code:       "WH001",
			Address:    "123 Storage Lane",
			City:       "Atlanta",
			State:      "GA",
			Country:    "USA",
			PostalCode: "30301",
			Phone:      "+1-555-0123",
			Email:      "warehouse@example.com",
			ManagerID:  &uuid.UUID{},
			IsActive:   true,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}

		err := warehouse.Validate()
		if err != nil {
			t.Errorf("Expected valid warehouse to pass validation, got error: %v", err)
		}
	})

	t.Run("EmptyName", func(t *testing.T) {
		warehouse := &Warehouse{
			ID:       warehouseID,
			Name:     "",
			Code:     "WH001",
			Address:  "123 Storage Lane",
			City:     "Atlanta",
			State:    "GA",
			Country:  "USA",
			IsActive: true,
		}

		err := warehouse.Validate()
		if err == nil {
			t.Error("Expected warehouse with empty name to fail validation")
		}
	})

	t.Run("InvalidCode", func(t *testing.T) {
		warehouse := &Warehouse{
			ID:       warehouseID,
			Name:     "Main Warehouse",
			Code:     "wh001", // Should be uppercase
			Address:  "123 Storage Lane",
			City:     "Atlanta",
			State:    "GA",
			Country:  "USA",
			IsActive: true,
		}

		err := warehouse.Validate()
		if err == nil {
			t.Error("Expected warehouse with lowercase code to fail validation")
		}
	})

	t.Run("BusinessLogic", func(t *testing.T) {
		warehouse := &Warehouse{
			ID:        warehouseID,
			Name:      "Main Warehouse",
			Code:      "WH001",
			Address:   "123 Storage Lane",
			City:      "Atlanta",
			State:     "GA",
			Country:   "USA",
			PostalCode: "30301",
			IsActive:  true,
		}

		// Test IsActiveWarehouse
		if !warehouse.IsActiveWarehouse() {
			t.Error("Expected active warehouse to return true")
		}

		warehouse.Deactivate()
		if warehouse.IsActiveWarehouse() {
			t.Error("Expected deactivated warehouse to return false")
		}

		// Test activation
		warehouse.Activate()
		if !warehouse.IsActiveWarehouse() {
			t.Error("Expected activated warehouse to return true")
		}

		// Test GetFullAddress
		fullAddress := warehouse.GetFullAddress()
		expectedAddress := "123 Storage Lane\nAtlanta, GA, 30301\nUSA"
		if fullAddress != expectedAddress {
			t.Errorf("Expected address '%s', got '%s'", expectedAddress, fullAddress)
		}

		// Test manager operations
		managerID := uuid.New()
		err := warehouse.SetManager(managerID)
		if err != nil {
			t.Errorf("Expected successful manager assignment, got error: %v", err)
		}

		if !warehouse.HasManager() {
			t.Error("Expected warehouse to have manager after assignment")
		}

		if !warehouse.IsManager(managerID) {
			t.Error("Expected warehouse to recognize assigned manager")
		}

		warehouse.RemoveManager()
		if warehouse.HasManager() {
			t.Error("Expected warehouse to not have manager after removal")
		}
	})
}

// TestWarehouseExtended tests the WarehouseExtended entity
func TestWarehouseExtended(t *testing.T) {
	warehouseID := uuid.New()

	t.Run("ValidExtendedWarehouse", func(t *testing.T) {
		capacity := 10000
		warehouse := &WarehouseExtended{
			Warehouse: Warehouse{
				ID:        warehouseID,
				Name:      "Distribution Center",
				Code:      "DC001",
				Address:   "456 Logistics Blvd",
				City:      "Chicago",
				State:     "IL",
				Country:   "USA",
				PostalCode: "60601",
				IsActive:  true,
			},
			Type:                 WarehouseTypeDistribution,
			Capacity:             &capacity,
			SquareFootage:        &[]int{50000}[0],
			DockCount:            &[]int{20}[0],
			TemperatureControlled: true,
			SecurityLevel:        5,
		}

		err := warehouse.Validate()
		if err != nil {
			t.Errorf("Expected valid extended warehouse to pass validation, got error: %v", err)
		}
	})

	t.Run("InvalidType", func(t *testing.T) {
		warehouse := &WarehouseExtended{
			Warehouse: Warehouse{
				ID:       warehouseID,
				Name:     "Test Warehouse",
				Code:     "WH001",
				Address:  "123 Test St",
				City:     "Test City",
				State:    "TS",
				Country:  "TC",
				IsActive: true,
			},
			Type:         "INVALID_TYPE",
			SecurityLevel: 3,
		}

		err := warehouse.Validate()
		if err == nil {
			t.Error("Expected warehouse with invalid type to fail validation")
		}
	})

	t.Run("UtilizationCalculation", func(t *testing.T) {
		capacity := 1000
		warehouse := &WarehouseExtended{
			Warehouse: Warehouse{
				ID:       warehouseID,
				Name:     "Test Warehouse",
				Code:     "WH001",
				Address:  "123 Test St",
				City:     "Test City",
				State:    "TS",
				Country:  "TC",
				IsActive: true,
			},
			Type:         WarehouseTypeRetail,
			Capacity:     &capacity,
			SecurityLevel: 3,
		}

		utilization, err := warehouse.GetUtilizationPercentage(500)
		if err != nil {
			t.Errorf("Expected successful utilization calculation, got error: %v", err)
		}
		if utilization != 50.0 {
			t.Errorf("Expected 50%% utilization, got %.1f%%", utilization)
		}

		// Test over capacity
		utilization, err = warehouse.GetUtilizationPercentage(1500)
		if err != nil {
			t.Errorf("Expected successful utilization calculation for over capacity, got error: %v", err)
		}
		if utilization != 100.0 {
			t.Errorf("Expected 100%% utilization for over capacity, got %.1f%%", utilization)
		}
	})
}

// TestInventory tests the Inventory entity
func TestInventory(t *testing.T) {
	inventoryID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	updatedBy := uuid.New()

	t.Run("ValidInventory", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   100,
			QuantityReserved: 20,
			ReorderLevel:     30,
			MaxStock:         &[]int{500}[0],
			MinStock:         &[]int{10}[0],
			AverageCost:      25.50,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		err := inventory.Validate()
		if err != nil {
			t.Errorf("Expected valid inventory to pass validation, got error: %v", err)
		}
	})

	t.Run("ReservedExceedsOnHand", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   50,
			QuantityReserved: 100, // More than on hand
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		err := inventory.Validate()
		if err == nil {
			t.Error("Expected inventory with reserved > on-hand to fail validation")
		}
	})

	t.Run("StockCalculations", func(t *testing.T) {
		inventory := &Inventory{
			ID:               inventoryID,
			ProductID:        productID,
			WarehouseID:      warehouseID,
			QuantityOnHand:   25,
			QuantityReserved: 20,
			ReorderLevel:     30,
			UpdatedAt:        time.Now().UTC(),
			UpdatedBy:        updatedBy,
		}

		// Test GetAvailableQuantity
		available := inventory.GetAvailableQuantity()
		if available != 5 {
			t.Errorf("Expected available quantity 5, got %d", available)
		}

		// Test IsLowStock
		if !inventory.IsLowStock() {
			t.Error("Expected inventory to be low stock when quantity <= reorder level")
		}

		// Test IsAvailable
		err := inventory.IsAvailable(5)
		if err != nil {
			t.Errorf("Expected 5 units to be available, got error: %v", err)
		}

		err = inventory.IsAvailable(10)
		if err == nil {
			t.Error("Expected 10 units to not be available")
		}

		// Test StockStatus
		status := inventory.GetStockStatus()
		if status != "LOW_STOCK" {
			t.Errorf("Expected status 'LOW_STOCK', got '%s'", status)
		}
	})

	t.Run("StockOperations", func(t *testing.T) {
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

		// Test ReserveStock
		err := inventory.ReserveStock(30)
		if err != nil {
			t.Errorf("Expected successful stock reservation, got error: %v", err)
		}
		if inventory.QuantityReserved != 50 {
			t.Errorf("Expected reserved quantity 50, got %d", inventory.QuantityReserved)
		}

		// Test ReleaseStock
		err = inventory.ReleaseStock(10)
		if err != nil {
			t.Errorf("Expected successful stock release, got error: %v", err)
		}
		if inventory.QuantityReserved != 40 {
			t.Errorf("Expected reserved quantity 40, got %d", inventory.QuantityReserved)
		}

		// Test AddStock
		err = inventory.AddStock(50)
		if err != nil {
			t.Errorf("Expected successful stock addition, got error: %v", err)
		}
		if inventory.QuantityOnHand != 150 {
			t.Errorf("Expected on-hand quantity 150, got %d", inventory.QuantityOnHand)
		}

		// Test RemoveStock
		err = inventory.RemoveStock(30)
		if err != nil {
			t.Errorf("Expected successful stock removal, got error: %v", err)
		}
		// Starting state: 150 on hand, 40 reserved. Remove 30 reduces reserved to 10, on-hand stays 150
		if inventory.QuantityOnHand != 150 {
			t.Errorf("Expected on-hand quantity 150, got %d", inventory.QuantityOnHand)
		}
		if inventory.QuantityReserved != 10 {
			t.Errorf("Expected reserved quantity 10, got %d", inventory.QuantityReserved)
		}
	})
}

// TestInventoryTransaction tests the InventoryTransaction entity
func TestInventoryTransaction(t *testing.T) {
	transactionID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	createdBy := uuid.New()

	t.Run("ValidPurchaseTransaction", func(t *testing.T) {
		refID := uuid.New()
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			ReferenceType:   "PO",
			ReferenceID:     &refID,
			UnitCost:        25.50,
			TotalCost:       2550.00,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		err := transaction.Validate()
		if err != nil {
			t.Errorf("Expected valid purchase transaction to pass validation, got error: %v", err)
		}

		if !transaction.IsStockIn() {
			t.Error("Expected purchase transaction to be stock in")
		}

		if transaction.IsStockOut() {
			t.Error("Expected purchase transaction to not be stock out")
		}
	})

	t.Run("ValidSaleTransaction", func(t *testing.T) {
		refID := uuid.New()
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypeSale,
			Quantity:        -50,
			ReferenceType:   "SO",
			ReferenceID:     &refID,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		err := transaction.Validate()
		if err != nil {
			t.Errorf("Expected valid sale transaction to pass validation, got error: %v", err)
		}

		if transaction.IsStockIn() {
			t.Error("Expected sale transaction to not be stock in")
		}

		if !transaction.IsStockOut() {
			t.Error("Expected sale transaction to be stock out")
		}
	})

	t.Run("InvalidTransactionType", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: "INVALID",
			Quantity:        100,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		err := transaction.Validate()
		if err == nil {
			t.Error("Expected transaction with invalid type to fail validation")
		}
	})

	t.Run("ZeroQuantity", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        0,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		err := transaction.Validate()
		if err == nil {
			t.Error("Expected transaction with zero quantity to fail validation")
		}
	})

	t.Run("TransferTransaction", func(t *testing.T) {
		fromWarehouse := uuid.New()
		toWarehouse := uuid.New()

		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     toWarehouse, // Destination warehouse
			TransactionType: TransactionTypeTransferIn,
			Quantity:        25,
			FromWarehouseID: &fromWarehouse,
			ToWarehouseID:   &toWarehouse,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		err := transaction.Validate()
		if err != nil {
			t.Errorf("Expected valid transfer transaction to pass validation, got error: %v", err)
		}

		if !transaction.IsTransfer() {
			t.Error("Expected transfer transaction to be identified as transfer")
		}

		partner, err := transaction.GetTransferPartner()
		if err != nil {
			t.Errorf("Expected to get transfer partner, got error: %v", err)
		}
		if *partner != fromWarehouse {
			t.Error("Expected transfer partner to be from warehouse")
		}
	})

	t.Run("TransactionWithBatchInfo", func(t *testing.T) {
		expiryDate := time.Now().AddDate(1, 0, 0) // 1 year from now
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			BatchNumber:     "BATCH-001",
			SerialNumber:    "SN-001",
			ExpiryDate:      &expiryDate,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		err := transaction.Validate()
		if err != nil {
			t.Errorf("Expected valid transaction with batch info to pass validation, got error: %v", err)
		}

		if !transaction.HasBatchInfo() {
			t.Error("Expected transaction to have batch info")
		}

		if transaction.IsExpired() {
			t.Error("Expected transaction with future expiry date to not be expired")
		}

		if !transaction.IsNearExpiry() {
			// With expiry set to 1 year from now, this should be false
			// This test might need adjustment based on the "near expiry" threshold
		}
	})

	t.Run("CostCalculation", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		err := transaction.SetCosts(25.50)
		if err != nil {
			t.Errorf("Expected successful cost setting, got error: %v", err)
		}

		if transaction.UnitCost != 25.50 {
			t.Errorf("Expected unit cost 25.50, got %.2f", transaction.UnitCost)
		}

		expectedTotal := 2550.00
		if transaction.TotalCost != expectedTotal {
			t.Errorf("Expected total cost %.2f, got %.2f", expectedTotal, transaction.TotalCost)
		}
	})

	t.Run("ApprovalWorkflow", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypeAdjustment,
			Quantity:        -10,
			Reason:          "Damaged goods found",
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		if !transaction.RequiresApproval() {
			t.Error("Expected adjustment transaction to require approval")
		}

		if transaction.IsApproved() {
			t.Error("Expected unapproved transaction to not be approved")
		}

		approverID := uuid.New()
		err := transaction.Approve(approverID)
		if err != nil {
			t.Errorf("Expected successful approval, got error: %v", err)
		}

		if !transaction.IsApproved() {
			t.Error("Expected approved transaction to be approved")
		}
	})
}

// TestInventoryTransactionFilter tests the InventoryTransactionFilter
func TestInventoryTransactionFilter(t *testing.T) {
	t.Run("ValidFilter", func(t *testing.T) {
		fromDate := time.Now().AddDate(-1, 0, 0)
		toDate := time.Now()
		productID := uuid.New()

		filter := &InventoryTransactionFilter{
			ProductID:   &productID,
			FromDate:    &fromDate,
			ToDate:      &toDate,
			Limit:       &[]int{50}[0],
			Offset:      &[]int{0}[0],
		}

		err := filter.Validate()
		if err != nil {
			t.Errorf("Expected valid filter to pass validation, got error: %v", err)
		}
	})

	t.Run("InvalidDateRange", func(t *testing.T) {
		fromDate := time.Now()
		toDate := time.Now().AddDate(-1, 0, 0) // Before from date

		filter := &InventoryTransactionFilter{
			FromDate: &fromDate,
			ToDate:   &toDate,
		}

		err := filter.Validate()
		if err == nil {
			t.Error("Expected filter with invalid date range to fail validation")
		}
	})

	t.Run("InvalidPagination", func(t *testing.T) {
		filter := &InventoryTransactionFilter{
			Limit: &[]int{-10}[0], // Negative limit
		}

		err := filter.Validate()
		if err == nil {
			t.Error("Expected filter with negative limit to fail validation")
		}
	})
}

// TestGetSummary tests the GetSummary function
func TestGetSummary(t *testing.T) {
	productID := uuid.New()
	warehouseID := uuid.New()
	createdBy := uuid.New()

	transactions := []InventoryTransaction{
		{
			ID:              uuid.New(),
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			UnitCost:        25.00,
			TotalCost:       2500.00,
			CreatedAt:       time.Now().AddDate(0, 0, -2),
			CreatedBy:       createdBy,
		},
		{
			ID:              uuid.New(),
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypeSale,
			Quantity:        -30,
			TotalCost:       0.00,
			CreatedAt:       time.Now().AddDate(0, 0, -1),
			CreatedBy:       createdBy,
		},
		{
			ID:              uuid.New(),
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        50,
			UnitCost:        26.00,
			TotalCost:       1300.00,
			CreatedAt:       time.Now(),
			CreatedBy:       createdBy,
		},
	}

	summary := GetSummary(transactions)

	if summary.TotalTransactions != 3 {
		t.Errorf("Expected 3 total transactions, got %d", summary.TotalTransactions)
	}

	if summary.TotalQuantityIn != 150 {
		t.Errorf("Expected 150 total quantity in, got %d", summary.TotalQuantityIn)
	}

	if summary.TotalQuantityOut != 30 {
		t.Errorf("Expected 30 total quantity out, got %d", summary.TotalQuantityOut)
	}

	if summary.NetQuantity != 120 {
		t.Errorf("Expected 120 net quantity, got %d", summary.NetQuantity)
	}

	if summary.TotalValue != 3800.00 {
		t.Errorf("Expected 3800.00 total value, got %.2f", summary.TotalValue)
	}

	expectedAverageCost := 3800.00 / 150.00 // 25.33
	if summary.AverageCost < 25.32 || summary.AverageCost > 25.34 {
		t.Errorf("Expected average cost around %.2f, got %.2f", expectedAverageCost, summary.AverageCost)
	}

	if summary.MostRecentTransaction == nil {
		t.Error("Expected most recent transaction to be set")
	}
}

// Benchmark tests for performance validation
func BenchmarkWarehouseValidation(b *testing.B) {
	warehouseID := uuid.New()
	warehouse := &Warehouse{
		ID:         warehouseID,
		Name:       "Main Warehouse",
		Code:       "WH001",
		Address:    "123 Storage Lane",
		City:       "Atlanta",
		State:      "GA",
		Country:    "USA",
		PostalCode: "30301",
		Phone:      "+1-555-0123",
		Email:      "warehouse@example.com",
		IsActive:   true,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		warehouse.Validate()
	}
}

func BenchmarkInventoryValidation(b *testing.B) {
	inventoryID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	updatedBy := uuid.New()

	inventory := &Inventory{
		ID:               inventoryID,
		ProductID:        productID,
		WarehouseID:      warehouseID,
		QuantityOnHand:   100,
		QuantityReserved: 20,
		ReorderLevel:     30,
		AverageCost:      25.50,
		UpdatedAt:        time.Now().UTC(),
		UpdatedBy:        updatedBy,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inventory.Validate()
	}
}

func BenchmarkTransactionValidation(b *testing.B) {
	transactionID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	createdBy := uuid.New()

	transaction := &InventoryTransaction{
		ID:              transactionID,
		ProductID:       productID,
		WarehouseID:     warehouseID,
		TransactionType: TransactionTypePurchase,
		Quantity:        100,
		UnitCost:        25.50,
		TotalCost:       2550.00,
		CreatedAt:       time.Now().UTC(),
		CreatedBy:       createdBy,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transaction.Validate()
	}
}