package entities

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TransactionType represents the type of inventory transaction
type TransactionType string

const (
	TransactionTypePurchase    TransactionType = "PURCHASE"     // Stock in from purchase
	TransactionTypeSale        TransactionType = "SALE"         // Stock out from sale
	TransactionTypeAdjustment  TransactionType = "ADJUSTMENT"   // Manual adjustment
	TransactionTypeTransferIn  TransactionType = "TRANSFER_IN"  // Transfer in from another warehouse
	TransactionTypeTransferOut TransactionType = "TRANSFER_OUT" // Transfer out to another warehouse
	TransactionTypeReturn      TransactionType = "RETURN"       // Customer return
	TransactionTypeDamage      TransactionType = "DAMAGE"       // Damaged goods
	TransactionTypeTheft       TransactionType = "THEFT"        // Stolen goods
	TransactionTypeExpiry      TransactionType = "EXPIRY"       // Expired goods
	TransactionTypeProduction  TransactionType = "PRODUCTION"   // Production output
	TransactionTypeConsumption TransactionType = "CONSUMPTION"  // Used in production
	TransactionTypeCount       TransactionType = "COUNT"        // Cycle count adjustment
)

// InventoryTransaction represents a movement of inventory
type InventoryTransaction struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	ProductID       uuid.UUID       `json:"product_id" db:"product_id"`
	WarehouseID     uuid.UUID       `json:"warehouse_id" db:"warehouse_id"`
	TransactionType TransactionType `json:"transaction_type" db:"transaction_type"`
	Quantity        int             `json:"quantity" db:"quantity"`
	ReferenceType   string          `json:"reference_type,omitempty" db:"reference_type"`
	ReferenceID     *uuid.UUID      `json:"reference_id,omitempty" db:"reference_id"`
	Reason          string          `json:"reason,omitempty" db:"reason"`
	UnitCost        float64         `json:"unit_cost,omitempty" db:"unit_cost"`
	TotalCost       float64         `json:"total_cost,omitempty" db:"total_cost"`
	BatchNumber     string          `json:"batch_number,omitempty" db:"batch_number"`
	ExpiryDate      *time.Time      `json:"expiry_date,omitempty" db:"expiry_date"`
	SerialNumber    string          `json:"serial_number,omitempty" db:"serial_number"`
	FromWarehouseID *uuid.UUID      `json:"from_warehouse_id,omitempty" db:"from_warehouse_id"`
	ToWarehouseID   *uuid.UUID      `json:"to_warehouse_id,omitempty" db:"to_warehouse_id"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	CreatedBy       uuid.UUID       `json:"created_by" db:"created_by"`
	ApprovedAt      *time.Time      `json:"approved_at,omitempty" db:"approved_at"`
	ApprovedBy      *uuid.UUID      `json:"approved_by,omitempty" db:"approved_by"`
}

// Validate validates the inventory transaction entity
func (t *InventoryTransaction) Validate() error {
	var errs []error

	// Validate UUIDs
	if t.ID == uuid.Nil {
		errs = append(errs, errors.New("transaction ID cannot be empty"))
	}

	if t.ProductID == uuid.Nil {
		errs = append(errs, errors.New("product ID cannot be empty"))
	}

	if t.WarehouseID == uuid.Nil {
		errs = append(errs, errors.New("warehouse ID cannot be empty"))
	}

	if t.CreatedBy == uuid.Nil {
		errs = append(errs, errors.New("created by user ID cannot be empty"))
	}

	// Validate transaction type
	if err := t.validateTransactionType(); err != nil {
		errs = append(errs, fmt.Errorf("invalid transaction type: %w", err))
	}

	// Validate quantity
	if err := t.validateQuantity(); err != nil {
		errs = append(errs, fmt.Errorf("invalid quantity: %w", err))
	}

	// Validate reference information
	if err := t.validateReference(); err != nil {
		errs = append(errs, fmt.Errorf("invalid reference: %w", err))
	}

	// Validate costs
	if err := t.validateCosts(); err != nil {
		errs = append(errs, fmt.Errorf("invalid costs: %w", err))
	}

	// Validate batch and serial information
	if err := t.validateBatchInfo(); err != nil {
		errs = append(errs, fmt.Errorf("invalid batch information: %w", err))
	}

	// Validate transfer information
	if err := t.validateTransferInfo(); err != nil {
		errs = append(errs, fmt.Errorf("invalid transfer information: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// validateTransactionType validates the transaction type
func (t *InventoryTransaction) validateTransactionType() error {
	validTypes := map[TransactionType]bool{
		TransactionTypePurchase:    true,
		TransactionTypeSale:        true,
		TransactionTypeAdjustment:  true,
		TransactionTypeTransferIn:  true,
		TransactionTypeTransferOut: true,
		TransactionTypeReturn:      true,
		TransactionTypeDamage:      true,
		TransactionTypeTheft:       true,
		TransactionTypeExpiry:      true,
		TransactionTypeProduction:  true,
		TransactionTypeConsumption: true,
		TransactionTypeCount:       true,
	}

	if !validTypes[t.TransactionType] {
		return fmt.Errorf("invalid transaction type: %s", t.TransactionType)
	}

	return nil
}

// validateQuantity validates the transaction quantity
func (t *InventoryTransaction) validateQuantity() error {
	if t.Quantity == 0 {
		return errors.New("transaction quantity cannot be zero")
	}

	absQuantity := t.Quantity
	if absQuantity < 0 {
		absQuantity = -absQuantity
	}

	if absQuantity > 999999999 {
		return errors.New("transaction quantity cannot exceed 999,999,999")
	}

	// Validate quantity sign based on transaction type
	switch t.TransactionType {
	case TransactionTypePurchase, TransactionTypeTransferIn, TransactionTypeReturn,
		TransactionTypeProduction, TransactionTypeCount:
		if t.Quantity <= 0 {
			return fmt.Errorf("transaction type %s requires positive quantity", t.TransactionType)
		}
	case TransactionTypeSale, TransactionTypeTransferOut, TransactionTypeDamage,
		TransactionTypeTheft, TransactionTypeExpiry, TransactionTypeConsumption:
		if t.Quantity >= 0 {
			return fmt.Errorf("transaction type %s requires negative quantity", t.TransactionType)
		}
	case TransactionTypeAdjustment:
		// Adjustments can be positive or negative
	}

	return nil
}

// validateReference validates reference information
func (t *InventoryTransaction) validateReference() error {
	// Reference type validation (optional)
	if t.ReferenceType != "" {
		if len(t.ReferenceType) > 50 {
			return errors.New("reference type cannot exceed 50 characters")
		}

		// Reference type should be alphanumeric with underscores and hyphens
		refTypeRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
		if !refTypeRegex.MatchString(t.ReferenceType) {
			return errors.New("reference type can only contain letters, numbers, hyphens, and underscores")
		}
	}

	// Reference ID validation (optional)
	if t.ReferenceID != nil {
		if *t.ReferenceID == uuid.Nil {
			return errors.New("reference ID cannot be empty when provided")
		}

		// Reference type should be provided when reference ID is provided
		if t.ReferenceType == "" {
			return errors.New("reference type must be provided when reference ID is specified")
		}
	}

	// If reference type is provided, reference ID should also be provided
	if t.ReferenceType != "" && t.ReferenceID == nil {
		return errors.New("reference ID must be provided when reference type is specified")
	}

	return nil
}

// validateCosts validates cost information
func (t *InventoryTransaction) validateCosts() error {
	// Unit cost validation (optional)
	if t.UnitCost < 0 {
		return errors.New("unit cost cannot be negative")
	}

	if t.UnitCost > 999999999.99 {
		return errors.New("unit cost cannot exceed 999,999,999.99")
	}

	// Total cost validation (optional)
	if t.TotalCost < 0 {
		return errors.New("total cost cannot be negative")
	}

	if t.TotalCost > 999999999.99 {
		return errors.New("total cost cannot exceed 999,999,999.99")
	}

	// If both unit cost and total cost are provided, they should be consistent
	if t.UnitCost > 0 && t.TotalCost > 0 {
		absQuantity := t.Quantity
		if absQuantity < 0 {
			absQuantity = -absQuantity
		}
		expectedTotal := t.UnitCost * float64(absQuantity)

		// Allow for small floating point differences
		diff := expectedTotal - t.TotalCost
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.01 { // Allow 1 cent difference
			return fmt.Errorf("total cost (%.2f) does not match unit cost (%.2f) * quantity (%d)",
				t.TotalCost, t.UnitCost, absQuantity)
		}
	}

	return nil
}

// validateBatchInfo validates batch and serial information
func (t *InventoryTransaction) validateBatchInfo() error {
	// Batch number validation (optional)
	if t.BatchNumber != "" {
		if len(t.BatchNumber) > 100 {
			return errors.New("batch number cannot exceed 100 characters")
		}

		// Batch number should be alphanumeric with common special characters
		batchRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_#]+$`)
		if !batchRegex.MatchString(t.BatchNumber) {
			return errors.New("batch number can only contain letters, numbers, hyphens, underscores, and #")
		}
	}

	// Serial number validation (optional)
	if t.SerialNumber != "" {
		if len(t.SerialNumber) > 100 {
			return errors.New("serial number cannot exceed 100 characters")
		}

		// Serial number should be alphanumeric with common special characters
		serialRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_#]+$`)
		if !serialRegex.MatchString(t.SerialNumber) {
			return errors.New("serial number can only contain letters, numbers, hyphens, underscores, and #")
		}
	}

	// Expiry date validation (optional)
	if t.ExpiryDate != nil {
		if t.ExpiryDate.Before(time.Now().UTC()) {
			return errors.New("expiry date cannot be in the past")
		}

		// Reasonable limit for expiry date (10 years from now)
		maxExpiry := time.Now().AddDate(10, 0, 0)
		if t.ExpiryDate.After(maxExpiry) {
			return errors.New("expiry date cannot be more than 10 years in the future")
		}
	}

	return nil
}

// validateTransferInfo validates transfer-specific information
func (t *InventoryTransaction) validateTransferInfo() error {
	// Validate transfer information for transfer transactions
	if t.TransactionType == TransactionTypeTransferIn || t.TransactionType == TransactionTypeTransferOut {
		if t.FromWarehouseID == nil || t.ToWarehouseID == nil {
			return errors.New("both from and to warehouse IDs must be specified for transfer transactions")
		}

		if *t.FromWarehouseID == uuid.Nil || *t.ToWarehouseID == uuid.Nil {
			return errors.New("warehouse IDs cannot be empty for transfer transactions")
		}

		if *t.FromWarehouseID == *t.ToWarehouseID {
			return errors.New("from and to warehouses cannot be the same for transfer transactions")
		}

		// Current warehouse should match either from or to warehouse
		if t.WarehouseID != *t.FromWarehouseID && t.WarehouseID != *t.ToWarehouseID {
			return errors.New("current warehouse must be either from or to warehouse for transfer transactions")
		}
	} else {
		// Non-transfer transactions should not have transfer fields
		if t.FromWarehouseID != nil || t.ToWarehouseID != nil {
			return errors.New("transfer fields should only be specified for transfer transactions")
		}
	}

	return nil
}

// Business Logic Methods

// IsStockIn returns true if the transaction adds stock
func (t *InventoryTransaction) IsStockIn() bool {
	return t.Quantity > 0
}

// IsStockOut returns true if the transaction removes stock
func (t *InventoryTransaction) IsStockOut() bool {
	return t.Quantity < 0
}

// GetAbsoluteQuantity returns the absolute value of the quantity
func (t *InventoryTransaction) GetAbsoluteQuantity() int {
	if t.Quantity < 0 {
		return -t.Quantity
	}
	return t.Quantity
}

// GetTransactionTypeName returns the human-readable name of the transaction type
func (t *InventoryTransaction) GetTransactionTypeName() string {
	switch t.TransactionType {
	case TransactionTypePurchase:
		return "Purchase"
	case TransactionTypeSale:
		return "Sale"
	case TransactionTypeAdjustment:
		return "Adjustment"
	case TransactionTypeTransferIn:
		return "Transfer In"
	case TransactionTypeTransferOut:
		return "Transfer Out"
	case TransactionTypeReturn:
		return "Customer Return"
	case TransactionTypeDamage:
		return "Damage"
	case TransactionTypeTheft:
		return "Theft"
	case TransactionTypeExpiry:
		return "Expiry"
	case TransactionTypeProduction:
		return "Production"
	case TransactionTypeConsumption:
		return "Consumption"
	case TransactionTypeCount:
		return "Cycle Count"
	default:
		return "Unknown"
	}
}

// RequiresApproval returns true if the transaction type requires approval
func (t *InventoryTransaction) RequiresApproval() bool {
	switch t.TransactionType {
	case TransactionTypeAdjustment, TransactionTypeDamage, TransactionTypeTheft, TransactionTypeExpiry:
		return true
	default:
		return false
	}
}

// IsApproved returns true if the transaction has been approved
func (t *InventoryTransaction) IsApproved() bool {
	return t.ApprovedAt != nil && t.ApprovedBy != nil
}

// Approve approves the transaction
func (t *InventoryTransaction) Approve(approvedBy uuid.UUID) error {
	if approvedBy == uuid.Nil {
		return errors.New("approved by user ID cannot be empty")
	}

	if !t.RequiresApproval() {
		return errors.New("this transaction type does not require approval")
	}

	now := time.Now().UTC()
	t.ApprovedAt = &now
	t.ApprovedBy = &approvedBy
	return nil
}

// IsExpired returns true if the batch is expired
func (t *InventoryTransaction) IsExpired() bool {
	if t.ExpiryDate == nil {
		return false
	}
	return t.ExpiryDate.Before(time.Now().UTC())
}

// GetDaysToExpiry returns the number of days until expiry
func (t *InventoryTransaction) GetDaysToExpiry() (int, error) {
	if t.ExpiryDate == nil {
		return 0, errors.New("no expiry date set")
	}

	now := time.Now().UTC()
	if t.ExpiryDate.Before(now) {
		return 0, nil // Already expired
	}

	duration := t.ExpiryDate.Sub(now)
	return int(duration.Hours() / 24), nil
}

// IsNearExpiry returns true if the batch is near expiry (within 30 days)
func (t *InventoryTransaction) IsNearExpiry() bool {
	if t.ExpiryDate == nil {
		return false
	}

	thirtyDaysFromNow := time.Now().AddDate(0, 0, 30)
	return t.ExpiryDate.Before(thirtyDaysFromNow)
}

// HasReference returns true if the transaction has a reference
func (t *InventoryTransaction) HasReference() bool {
	return t.ReferenceType != "" && t.ReferenceID != nil
}

// HasBatchInfo returns true if the transaction has batch information
func (t *InventoryTransaction) HasBatchInfo() bool {
	return t.BatchNumber != "" || t.SerialNumber != "" || t.ExpiryDate != nil
}

// IsTransfer returns true if the transaction is a transfer
func (t *InventoryTransaction) IsTransfer() bool {
	return t.TransactionType == TransactionTypeTransferIn || t.TransactionType == TransactionTypeTransferOut
}

// GetTransferPartner returns the partner warehouse ID for transfers
func (t *InventoryTransaction) GetTransferPartner() (*uuid.UUID, error) {
	if !t.IsTransfer() {
		return nil, errors.New("not a transfer transaction")
	}

	if t.TransactionType == TransactionTypeTransferIn {
		return t.FromWarehouseID, nil
	}
	return t.ToWarehouseID, nil
}

// SetCosts sets the unit and total costs
func (t *InventoryTransaction) SetCosts(unitCost float64) error {
	if unitCost < 0 {
		return errors.New("unit cost cannot be negative")
	}

	if unitCost > 999999999.99 {
		return errors.New("unit cost cannot exceed 999,999,999.99")
	}

	absQuantity := t.GetAbsoluteQuantity()
	totalCost := unitCost * float64(absQuantity)

	t.UnitCost = unitCost
	t.TotalCost = totalCost
	return nil
}

// SetReference sets the reference information
func (t *InventoryTransaction) SetReference(referenceType string, referenceID uuid.UUID) error {
	if strings.TrimSpace(referenceType) == "" {
		return errors.New("reference type cannot be empty")
	}

	if referenceID == uuid.Nil {
		return errors.New("reference ID cannot be empty")
	}

	if len(referenceType) > 50 {
		return errors.New("reference type cannot exceed 50 characters")
	}

	// Reference type should be alphanumeric with underscores and hyphens
	refTypeRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !refTypeRegex.MatchString(referenceType) {
		return errors.New("reference type can only contain letters, numbers, hyphens, and underscores")
	}

	t.ReferenceType = referenceType
	t.ReferenceID = &referenceID
	return nil
}

// SetBatchInfo sets batch information
func (t *InventoryTransaction) SetBatchInfo(batchNumber, serialNumber string, expiryDate *time.Time) error {
	// Validate batch number
	if batchNumber != "" {
		if len(batchNumber) > 100 {
			return errors.New("batch number cannot exceed 100 characters")
		}

		batchRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_#]+$`)
		if !batchRegex.MatchString(batchNumber) {
			return errors.New("batch number can only contain letters, numbers, hyphens, underscores, and #")
		}
	}

	// Validate serial number
	if serialNumber != "" {
		if len(serialNumber) > 100 {
			return errors.New("serial number cannot exceed 100 characters")
		}

		serialRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_#]+$`)
		if !serialRegex.MatchString(serialNumber) {
			return errors.New("serial number can only contain letters, numbers, hyphens, underscores, and #")
		}
	}

	// Validate expiry date
	if expiryDate != nil {
		if expiryDate.Before(time.Now().UTC()) {
			return errors.New("expiry date cannot be in the past")
		}

		maxExpiry := time.Now().AddDate(10, 0, 0)
		if expiryDate.After(maxExpiry) {
			return errors.New("expiry date cannot be more than 10 years in the future")
		}
	}

	t.BatchNumber = batchNumber
	t.SerialNumber = serialNumber
	t.ExpiryDate = expiryDate
	return nil
}

// ToSafeTransaction returns a transaction object without sensitive information
func (t *InventoryTransaction) ToSafeTransaction() *InventoryTransaction {
	return &InventoryTransaction{
		ID:              t.ID,
		ProductID:       t.ProductID,
		WarehouseID:     t.WarehouseID,
		TransactionType: t.TransactionType,
		Quantity:        t.Quantity,
		ReferenceType:   t.ReferenceType,
		ReferenceID:     t.ReferenceID,
		Reason:          t.Reason,
		UnitCost:        t.UnitCost,
		TotalCost:       t.TotalCost,
		BatchNumber:     t.BatchNumber,
		ExpiryDate:      t.ExpiryDate,
		SerialNumber:    t.SerialNumber,
		FromWarehouseID: t.FromWarehouseID,
		ToWarehouseID:   t.ToWarehouseID,
		CreatedAt:       t.CreatedAt,
		CreatedBy:       t.CreatedBy,
		ApprovedAt:      t.ApprovedAt,
		ApprovedBy:      t.ApprovedBy,
	}
}

// InventoryTransactionFilter represents filters for querying inventory transactions
type InventoryTransactionFilter struct {
	ProductID       *uuid.UUID       `json:"product_id,omitempty"`
	WarehouseID     *uuid.UUID       `json:"warehouse_id,omitempty"`
	TransactionType *TransactionType `json:"transaction_type,omitempty"`
	ReferenceType   *string          `json:"reference_type,omitempty"`
	ReferenceID     *uuid.UUID       `json:"reference_id,omitempty"`
	BatchNumber     *string          `json:"batch_number,omitempty"`
	SerialNumber    *string          `json:"serial_number,omitempty"`
	FromDate        *time.Time       `json:"from_date,omitempty"`
	ToDate          *time.Time       `json:"to_date,omitempty"`
	CreatedBy       *uuid.UUID       `json:"created_by,omitempty"`
	ApprovedBy      *uuid.UUID       `json:"approved_by,omitempty"`
	Limit           *int             `json:"limit,omitempty"`
	Offset          *int             `json:"offset,omitempty"`
}

// Validate validates the transaction filter
func (f *InventoryTransactionFilter) Validate() error {
	// Validate date range
	if f.FromDate != nil && f.ToDate != nil {
		if f.FromDate.After(*f.ToDate) {
			return errors.New("from date cannot be after to date")
		}
	}

	// Validate pagination
	if f.Limit != nil && *f.Limit <= 0 {
		return errors.New("limit must be positive")
	}

	if f.Offset != nil && *f.Offset < 0 {
		return errors.New("offset cannot be negative")
	}

	return nil
}

// InventoryTransactionSummary represents a summary of inventory transactions
type InventoryTransactionSummary struct {
	TotalTransactions     int        `json:"total_transactions"`
	TotalQuantityIn       int        `json:"total_quantity_in"`
	TotalQuantityOut      int        `json:"total_quantity_out"`
	NetQuantity           int        `json:"net_quantity"`
	TotalValue            float64    `json:"total_value"`
	AverageCost           float64    `json:"average_cost"`
	MostRecentTransaction *time.Time `json:"most_recent_transaction,omitempty"`
}

// GetSummary generates a summary from a list of transactions
func GetSummary(transactions []InventoryTransaction) InventoryTransactionSummary {
	summary := InventoryTransactionSummary{
		TotalTransactions: len(transactions),
	}

	var mostRecentTime time.Time
	hasTransactions := false

	for _, tx := range transactions {
		if tx.Quantity > 0 {
			summary.TotalQuantityIn += tx.Quantity
		} else {
			summary.TotalQuantityOut += -tx.Quantity
		}

		summary.TotalValue += tx.TotalCost

		if !hasTransactions || tx.CreatedAt.After(mostRecentTime) {
			mostRecentTime = tx.CreatedAt
			hasTransactions = true
		}
	}

	summary.NetQuantity = summary.TotalQuantityIn - summary.TotalQuantityOut

	if hasTransactions {
		summary.MostRecentTransaction = &mostRecentTime
	}

	// Calculate average cost
	if summary.TotalQuantityIn > 0 {
		summary.AverageCost = summary.TotalValue / float64(summary.TotalQuantityIn)
	}

	return summary
}
