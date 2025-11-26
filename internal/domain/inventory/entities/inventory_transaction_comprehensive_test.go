package entities

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestInventoryTransaction_ComprehensiveCoverage tests all transaction methods for complete coverage
func TestInventoryTransaction_ComprehensiveCoverage(t *testing.T) {
	transactionID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	createdBy := uuid.New()

	t.Run("GetTransactionTypeName", func(t *testing.T) {
		tests := []struct {
			txType   TransactionType
			expected string
		}{
			{TransactionTypePurchase, "Purchase"},
			{TransactionTypeSale, "Sale"},
			{TransactionTypeAdjustment, "Adjustment"},
			{TransactionTypeTransferIn, "Transfer In"},
			{TransactionTypeTransferOut, "Transfer Out"},
			{TransactionTypeReturn, "Customer Return"},
			{TransactionTypeDamage, "Damage"},
			{TransactionTypeTheft, "Theft"},
			{TransactionTypeExpiry, "Expiry"},
			{TransactionTypeProduction, "Production"},
			{TransactionTypeConsumption, "Consumption"},
			{TransactionTypeCount, "Cycle Count"},
			{TransactionType("UNKNOWN"), "Unknown"},
		}

		for _, tt := range tests {
			transaction := &InventoryTransaction{
				ID:              transactionID,
				ProductID:       productID,
				WarehouseID:     warehouseID,
				TransactionType: tt.txType,
				Quantity:        10,
				CreatedAt:       time.Now().UTC(),
				CreatedBy:       createdBy,
			}

			name := transaction.GetTransactionTypeName()
			if name != tt.expected {
				t.Errorf("For type %s, expected name '%s', got '%s'", tt.txType, tt.expected, name)
			}
		}
	})

	t.Run("GetDaysToExpiry", func(t *testing.T) {
		// Future expiry date
		futureDate := time.Now().AddDate(0, 0, 30)
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			ExpiryDate:      &futureDate,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		days, err := transaction.GetDaysToExpiry()
		if err != nil {
			t.Errorf("Expected successful calculation, got error: %v", err)
		}
		if days < 29 || days > 31 { // Allow some tolerance
			t.Errorf("Expected approximately 30 days, got %d", days)
		}

		// Past expiry date
		pastDate := time.Now().AddDate(0, 0, -10)
		transaction.ExpiryDate = &pastDate
		days, err = transaction.GetDaysToExpiry()
		if err != nil {
			t.Errorf("Expected successful calculation, got error: %v", err)
		}
		if days != 0 {
			t.Errorf("Expected 0 days for expired item, got %d", days)
		}

		// No expiry date
		transaction.ExpiryDate = nil
		_, err = transaction.GetDaysToExpiry()
		if err == nil {
			t.Error("Expected error for no expiry date")
		}
	})

	t.Run("HasReference", func(t *testing.T) {
		refID := uuid.New()
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			ReferenceType:   "PO",
			ReferenceID:     &refID,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		if !transaction.HasReference() {
			t.Error("Expected transaction to have reference")
		}

		transaction.ReferenceType = ""
		if transaction.HasReference() {
			t.Error("Expected transaction to not have reference")
		}
	})

	t.Run("GetTransferPartner", func(t *testing.T) {
		fromWarehouse := uuid.New()
		toWarehouse := uuid.New()

		// Transfer in
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     toWarehouse,
			TransactionType: TransactionTypeTransferIn,
			Quantity:        25,
			FromWarehouseID: &fromWarehouse,
			ToWarehouseID:   &toWarehouse,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		partner, err := transaction.GetTransferPartner()
		if err != nil {
			t.Errorf("Expected successful partner retrieval, got error: %v", err)
		}
		if *partner != fromWarehouse {
			t.Error("Expected from warehouse as partner for transfer in")
		}

		// Transfer out
		transaction.TransactionType = TransactionTypeTransferOut
		transaction.WarehouseID = fromWarehouse
		partner, err = transaction.GetTransferPartner()
		if err != nil {
			t.Errorf("Expected successful partner retrieval, got error: %v", err)
		}
		if *partner != toWarehouse {
			t.Error("Expected to warehouse as partner for transfer out")
		}

		// Non-transfer transaction
		transaction.TransactionType = TransactionTypePurchase
		_, err = transaction.GetTransferPartner()
		if err == nil {
			t.Error("Expected error for non-transfer transaction")
		}
	})

	t.Run("SetReference", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		refID := uuid.New()

		// Valid reference
		err := transaction.SetReference("PO", refID)
		if err != nil {
			t.Errorf("Expected successful reference set, got error: %v", err)
		}
		if transaction.ReferenceType != "PO" || *transaction.ReferenceID != refID {
			t.Error("Reference not set correctly")
		}

		// Empty reference type
		err = transaction.SetReference("", refID)
		if err == nil {
			t.Error("Expected error for empty reference type")
		}

		// Empty reference ID
		err = transaction.SetReference("PO", uuid.Nil)
		if err == nil {
			t.Error("Expected error for empty reference ID")
		}

		// Reference type too long
		longType := string(make([]byte, 51))
		err = transaction.SetReference(longType, refID)
		if err == nil {
			t.Error("Expected error for reference type too long")
		}

		// Invalid characters in reference type
		err = transaction.SetReference("PO@#$", refID)
		if err == nil {
			t.Error("Expected error for invalid characters in reference type")
		}
	})

	t.Run("SetBatchInfo", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		futureDate := time.Now().AddDate(1, 0, 0)

		// Valid batch info
		err := transaction.SetBatchInfo("BATCH-001", "SN-001", &futureDate)
		if err != nil {
			t.Errorf("Expected successful batch info set, got error: %v", err)
		}
		if transaction.BatchNumber != "BATCH-001" || transaction.SerialNumber != "SN-001" {
			t.Error("Batch info not set correctly")
		}

		// Batch number too long
		longBatch := string(make([]byte, 101))
		err = transaction.SetBatchInfo(longBatch, "", nil)
		if err == nil {
			t.Error("Expected error for batch number too long")
		}

		// Invalid batch number characters
		err = transaction.SetBatchInfo("BATCH@#$", "", nil)
		if err == nil {
			t.Error("Expected error for invalid batch number characters")
		}

		// Serial number too long
		longSerial := string(make([]byte, 101))
		err = transaction.SetBatchInfo("", longSerial, nil)
		if err == nil {
			t.Error("Expected error for serial number too long")
		}

		// Invalid serial number characters
		err = transaction.SetBatchInfo("", "SN@#$", nil)
		if err == nil {
			t.Error("Expected error for invalid serial number characters")
		}

		// Past expiry date
		pastDate := time.Now().AddDate(-1, 0, 0)
		err = transaction.SetBatchInfo("", "", &pastDate)
		if err == nil {
			t.Error("Expected error for past expiry date")
		}

		// Expiry date too far in future
		farFutureDate := time.Now().AddDate(11, 0, 0)
		err = transaction.SetBatchInfo("", "", &farFutureDate)
		if err == nil {
			t.Error("Expected error for expiry date too far in future")
		}
	})
}

// TestInventoryTransactionFilter_Comprehensive tests filter validation
func TestInventoryTransactionFilter_Comprehensive(t *testing.T) {
	t.Run("ValidFilter_AllFields", func(t *testing.T) {
		productID := uuid.New()
		warehouseID := uuid.New()
		txType := TransactionTypePurchase
		refType := "PO"
		refID := uuid.New()
		batchNumber := "BATCH-001"
		serialNumber := "SN-001"
		fromDate := time.Now().AddDate(-1, 0, 0)
		toDate := time.Now()
		createdBy := uuid.New()
		approvedBy := uuid.New()
		limit := 50
		offset := 0

		filter := &InventoryTransactionFilter{
			ProductID:       &productID,
			WarehouseID:     &warehouseID,
			TransactionType: &txType,
			ReferenceType:   &refType,
			ReferenceID:     &refID,
			BatchNumber:     &batchNumber,
			SerialNumber:    &serialNumber,
			FromDate:        &fromDate,
			ToDate:          &toDate,
			CreatedBy:       &createdBy,
			ApprovedBy:      &approvedBy,
			Limit:           &limit,
			Offset:          &offset,
		}

		err := filter.Validate()
		if err != nil {
			t.Errorf("Expected valid filter, got error: %v", err)
		}
	})

	t.Run("InvalidOffset", func(t *testing.T) {
		offset := -10
		filter := &InventoryTransactionFilter{
			Offset: &offset,
		}

		err := filter.Validate()
		if err == nil {
			t.Error("Expected error for negative offset")
		}
	})
}

// TestInventoryTransaction_AdditionalCoverage adds more coverage for missing methods
func TestInventoryTransaction_AdditionalCoverage(t *testing.T) {
	transactionID := uuid.New()
	productID := uuid.New()
	warehouseID := uuid.New()
	createdBy := uuid.New()

	t.Run("ToSafeTransaction", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		safe := transaction.ToSafeTransaction()
		if safe.ID != transaction.ID {
			t.Error("Safe transaction ID mismatch")
		}
		if safe.ProductID != transaction.ProductID {
			t.Error("Safe transaction ProductID mismatch")
		}
	})

	t.Run("IsStockIn", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		if !transaction.IsStockIn() {
			t.Error("Expected transaction to be stock in")
		}

		transaction.Quantity = -50
		if transaction.IsStockIn() {
			t.Error("Expected transaction to not be stock in")
		}
	})

	t.Run("IsStockOut", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypeSale,
			Quantity:        -50,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		if !transaction.IsStockOut() {
			t.Error("Expected transaction to be stock out")
		}

		transaction.Quantity = 100
		if transaction.IsStockOut() {
			t.Error("Expected transaction to not be stock out")
		}
	})

	t.Run("GetAbsoluteQuantity", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypeSale,
			Quantity:        -50,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		abs := transaction.GetAbsoluteQuantity()
		if abs != 50 {
			t.Errorf("Expected absolute quantity 50, got %d", abs)
		}

		transaction.Quantity = 100
		abs = transaction.GetAbsoluteQuantity()
		if abs != 100 {
			t.Errorf("Expected absolute quantity 100, got %d", abs)
		}
	})

	t.Run("RequiresApproval", func(t *testing.T) {
		tests := []struct {
			txType   TransactionType
			requires bool
		}{
			{TransactionTypeAdjustment, true},
			{TransactionTypeDamage, true},
			{TransactionTypeTheft, true},
			{TransactionTypeExpiry, true},
			{TransactionTypePurchase, false},
			{TransactionTypeSale, false},
		}

		for _, tt := range tests {
			transaction := &InventoryTransaction{
				ID:              transactionID,
				ProductID:       productID,
				WarehouseID:     warehouseID,
				TransactionType: tt.txType,
				Quantity:        10,
				CreatedAt:       time.Now().UTC(),
				CreatedBy:       createdBy,
			}

			requires := transaction.RequiresApproval()
			if requires != tt.requires {
				t.Errorf("For type %s, expected RequiresApproval=%v, got %v", 
					tt.txType, tt.requires, requires)
			}
		}
	})

	t.Run("IsApproved", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypeAdjustment,
			Quantity:        10,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		if transaction.IsApproved() {
			t.Error("Expected transaction to not be approved")
		}

		approvedBy := uuid.New()
		now := time.Now().UTC()
		transaction.ApprovedAt = &now
		transaction.ApprovedBy = &approvedBy

		if !transaction.IsApproved() {
			t.Error("Expected transaction to be approved")
		}
	})

	t.Run("Approve", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypeAdjustment,
			Quantity:        10,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		approvedBy := uuid.New()

		// Valid approval
		err := transaction.Approve(approvedBy)
		if err != nil {
			t.Errorf("Expected successful approval, got error: %v", err)
		}
		if !transaction.IsApproved() {
			t.Error("Expected transaction to be approved")
		}

		// Empty approved by
		transaction2 := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypeAdjustment,
			Quantity:        10,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}
		err = transaction2.Approve(uuid.Nil)
		if err == nil {
			t.Error("Expected error for empty approved by")
		}

		// Transaction that doesn't require approval
		transaction3 := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        10,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}
		err = transaction3.Approve(approvedBy)
		if err == nil {
			t.Error("Expected error for transaction that doesn't require approval")
		}
	})

	t.Run("IsExpired", func(t *testing.T) {
		// Future expiry date
		futureDate := time.Now().AddDate(0, 0, 30)
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			ExpiryDate:      &futureDate,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		if transaction.IsExpired() {
			t.Error("Expected transaction to not be expired")
		}

		// Past expiry date
		pastDate := time.Now().AddDate(0, 0, -10)
		transaction.ExpiryDate = &pastDate
		if !transaction.IsExpired() {
			t.Error("Expected transaction to be expired")
		}

		// No expiry date
		transaction.ExpiryDate = nil
		if transaction.IsExpired() {
			t.Error("Expected transaction with no expiry to not be expired")
		}
	})

	t.Run("IsNearExpiry", func(t *testing.T) {
		// 20 days from now (near expiry)
		nearDate := time.Now().AddDate(0, 0, 20)
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			ExpiryDate:      &nearDate,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		if !transaction.IsNearExpiry() {
			t.Error("Expected transaction to be near expiry")
		}

		// 60 days from now (not near expiry)
		farDate := time.Now().AddDate(0, 0, 60)
		transaction.ExpiryDate = &farDate
		if transaction.IsNearExpiry() {
			t.Error("Expected transaction to not be near expiry")
		}

		// No expiry date
		transaction.ExpiryDate = nil
		if transaction.IsNearExpiry() {
			t.Error("Expected transaction with no expiry to not be near expiry")
		}
	})

	t.Run("HasBatchInfo", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			BatchNumber:     "BATCH-001",
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		if !transaction.HasBatchInfo() {
			t.Error("Expected transaction to have batch info")
		}

		transaction.BatchNumber = ""
		if transaction.HasBatchInfo() {
			t.Error("Expected transaction to not have batch info")
		}

		transaction.SerialNumber = "SN-001"
		if !transaction.HasBatchInfo() {
			t.Error("Expected transaction to have batch info")
		}
	})

	t.Run("IsTransfer", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypeTransferIn,
			Quantity:        25,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		if !transaction.IsTransfer() {
			t.Error("Expected transaction to be a transfer")
		}

		transaction.TransactionType = TransactionTypePurchase
		if transaction.IsTransfer() {
			t.Error("Expected transaction to not be a transfer")
		}
	})

	t.Run("SetCosts", func(t *testing.T) {
		transaction := &InventoryTransaction{
			ID:              transactionID,
			ProductID:       productID,
			WarehouseID:     warehouseID,
			TransactionType: TransactionTypePurchase,
			Quantity:        100,
			CreatedAt:       time.Now().UTC(),
			CreatedBy:       createdBy,
		}

		// Valid cost
		err := transaction.SetCosts(25.50)
		if err != nil {
			t.Errorf("Expected successful cost set, got error: %v", err)
		}
		if transaction.UnitCost != 25.50 {
			t.Errorf("Expected unit cost 25.50, got %.2f", transaction.UnitCost)
		}
		if transaction.TotalCost != 2550.00 {
			t.Errorf("Expected total cost 2550.00, got %.2f", transaction.TotalCost)
		}

		// Negative cost
		err = transaction.SetCosts(-10.00)
		if err == nil {
			t.Error("Expected error for negative cost")
		}

		// Cost exceeding max
		err = transaction.SetCosts(1000000000.00)
		if err == nil {
			t.Error("Expected error for cost exceeding max")
		}
	})
}
