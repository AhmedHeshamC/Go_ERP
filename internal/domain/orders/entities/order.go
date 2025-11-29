package entities

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	apperrors "erpgo/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusDraft            OrderStatus = "DRAFT"
	OrderStatusPending          OrderStatus = "PENDING"
	OrderStatusConfirmed        OrderStatus = "CONFIRMED"
	OrderStatusProcessing       OrderStatus = "PROCESSING"
	OrderStatusShipped          OrderStatus = "SHIPPED"
	OrderStatusDelivered        OrderStatus = "DELIVERED"
	OrderStatusCancelled        OrderStatus = "CANCELLED"
	OrderStatusRefunded         OrderStatus = "REFUNDED"
	OrderStatusReturned         OrderStatus = "RETURNED"
	OrderStatusOnHold           OrderStatus = "ON_HOLD"
	OrderStatusPartiallyShipped OrderStatus = "PARTIALLY_SHIPPED"
)

// OrderStatusTransitions defines valid status transitions
var OrderStatusTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusDraft:            {OrderStatusPending, OrderStatusCancelled},
	OrderStatusPending:          {OrderStatusConfirmed, OrderStatusCancelled, OrderStatusOnHold},
	OrderStatusConfirmed:        {OrderStatusProcessing, OrderStatusCancelled, OrderStatusOnHold},
	OrderStatusProcessing:       {OrderStatusShipped, OrderStatusPartiallyShipped, OrderStatusCancelled, OrderStatusOnHold},
	OrderStatusPartiallyShipped: {OrderStatusShipped, OrderStatusProcessing, OrderStatusCancelled, OrderStatusOnHold},
	OrderStatusShipped:          {OrderStatusDelivered, OrderStatusReturned, OrderStatusOnHold},
	OrderStatusDelivered:        {OrderStatusReturned, OrderStatusRefunded},
	OrderStatusOnHold:           {OrderStatusPending, OrderStatusConfirmed, OrderStatusProcessing, OrderStatusCancelled},
	OrderStatusCancelled:        {OrderStatusRefunded},
	OrderStatusReturned:         {OrderStatusRefunded},
	OrderStatusRefunded:         {}, // Terminal state
}

// IsValidStatusTransition checks if a status transition is valid
func IsValidStatusTransition(from, to OrderStatus) bool {
	allowedTransitions, exists := OrderStatusTransitions[from]
	if !exists {
		return false
	}

	for _, allowedStatus := range allowedTransitions {
		if allowedStatus == to {
			return true
		}
	}
	return false
}

// GetTerminalStatuses returns all terminal statuses
func GetTerminalStatuses() []OrderStatus {
	return []OrderStatus{OrderStatusDelivered, OrderStatusCancelled, OrderStatusRefunded}
}

// IsTerminalStatus checks if a status is terminal
func IsTerminalStatus(status OrderStatus) bool {
	for _, terminal := range GetTerminalStatuses() {
		if terminal == status {
			return true
		}
	}
	return false
}

// OrderPriority represents the priority level of an order
type OrderPriority string

const (
	OrderPriorityLow      OrderPriority = "LOW"
	OrderPriorityNormal   OrderPriority = "NORMAL"
	OrderPriorityHigh     OrderPriority = "HIGH"
	OrderPriorityUrgent   OrderPriority = "URGENT"
	OrderPriorityCritical OrderPriority = "CRITICAL"
)

// OrderType represents the type of order
type OrderType string

const (
	OrderTypeSales      OrderType = "SALES"
	OrderTypePurchase   OrderType = "PURCHASE"
	OrderTypeReturn     OrderType = "RETURN"
	OrderTypeExchange   OrderType = "EXCHANGE"
	OrderTypeTransfer   OrderType = "TRANSFER"
	OrderTypeAdjustment OrderType = "ADJUSTMENT"
)

// PaymentStatus represents the payment status of an order
type PaymentStatus string

const (
	PaymentStatusPending       PaymentStatus = "PENDING"
	PaymentStatusPaid          PaymentStatus = "PAID"
	PaymentStatusPartiallyPaid PaymentStatus = "PARTIALLY_PAID"
	PaymentStatusOverdue       PaymentStatus = "OVERDUE"
	PaymentStatusRefunded      PaymentStatus = "REFUNDED"
	PaymentStatusFailed        PaymentStatus = "FAILED"
)

// ShippingMethod represents the shipping method
type ShippingMethod string

const (
	ShippingMethodStandard      ShippingMethod = "STANDARD"
	ShippingMethodExpress       ShippingMethod = "EXPRESS"
	ShippingMethodOvernight     ShippingMethod = "OVERNIGHT"
	ShippingMethodInternational ShippingMethod = "INTERNATIONAL"
	ShippingMethodPickup        ShippingMethod = "PICKUP"
	ShippingMethodDigital       ShippingMethod = "DIGITAL"
)

// Order represents an order in the system
type Order struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	OrderNumber    string         `json:"order_number" db:"order_number"`
	CustomerID     uuid.UUID      `json:"customer_id" db:"customer_id"`
	Customer       *Customer      `json:"customer,omitempty" db:"-"`
	Status         OrderStatus    `json:"status" db:"status"`
	PreviousStatus *OrderStatus   `json:"previous_status,omitempty" db:"previous_status"`
	Priority       OrderPriority  `json:"priority" db:"priority"`
	Type           OrderType      `json:"type" db:"type"`
	PaymentStatus  PaymentStatus  `json:"payment_status" db:"payment_status"`
	ShippingMethod ShippingMethod `json:"shipping_method" db:"shipping_method"`

	// Financial fields
	Subtotal       decimal.Decimal `json:"subtotal" db:"subtotal"`
	TaxAmount      decimal.Decimal `json:"tax_amount" db:"tax_amount"`
	ShippingAmount decimal.Decimal `json:"shipping_amount" db:"shipping_amount"`
	DiscountAmount decimal.Decimal `json:"discount_amount" db:"discount_amount"`
	TotalAmount    decimal.Decimal `json:"total_amount" db:"total_amount"`
	PaidAmount     decimal.Decimal `json:"paid_amount" db:"paid_amount"`
	RefundedAmount decimal.Decimal `json:"refunded_amount" db:"refunded_amount"`
	Currency       string          `json:"currency" db:"currency"`

	// Date fields
	OrderDate     time.Time  `json:"order_date" db:"order_date"`
	RequiredDate  *time.Time `json:"required_date,omitempty" db:"required_date"`
	ShippingDate  *time.Time `json:"shipping_date,omitempty" db:"shipping_date"`
	DeliveryDate  *time.Time `json:"delivery_date,omitempty" db:"delivery_date"`
	CancelledDate *time.Time `json:"cancelled_date,omitempty" db:"cancelled_date"`

	// Address references
	ShippingAddressID uuid.UUID     `json:"shipping_address_id" db:"shipping_address_id"`
	BillingAddressID  uuid.UUID     `json:"billing_address_id" db:"billing_address_id"`
	ShippingAddress   *OrderAddress `json:"shipping_address,omitempty" db:"-"`
	BillingAddress    *OrderAddress `json:"billing_address,omitempty" db:"-"`

	// Metadata
	Notes          *string `json:"notes,omitempty" db:"notes"`
	InternalNotes  *string `json:"internal_notes,omitempty" db:"internal_notes"`
	CustomerNotes  *string `json:"customer_notes,omitempty" db:"customer_notes"`
	TrackingNumber *string `json:"tracking_number,omitempty" db:"tracking_number"`
	Carrier        *string `json:"carrier,omitempty" db:"carrier"`

	// System fields
	CreatedBy  uuid.UUID  `json:"created_by" db:"created_by"`
	ApprovedBy *uuid.UUID `json:"approved_by,omitempty" db:"approved_by"`
	ShippedBy  *uuid.UUID `json:"shipped_by,omitempty" db:"shipped_by"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	ApprovedAt *time.Time `json:"approved_at,omitempty" db:"approved_at"`
	ShippedAt  *time.Time `json:"shipped_at,omitempty" db:"shipped_at"`

	// Relationships
	Items []OrderItem `json:"items,omitempty" db:"-"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	OrderID        uuid.UUID       `json:"order_id" db:"order_id"`
	ProductID      uuid.UUID       `json:"product_id" db:"product_id"`
	ProductSKU     string          `json:"product_sku" db:"product_sku"`
	ProductName    string          `json:"product_name" db:"product_name"`
	Quantity       int             `json:"quantity" db:"quantity"`
	UnitPrice      decimal.Decimal `json:"unit_price" db:"unit_price"`
	DiscountAmount decimal.Decimal `json:"discount_amount" db:"discount_amount"`
	TaxRate        decimal.Decimal `json:"tax_rate" db:"tax_rate"`
	TaxAmount      decimal.Decimal `json:"tax_amount" db:"tax_amount"`
	TotalPrice     decimal.Decimal `json:"total_price" db:"total_price"`

	// Additional fields
	Weight     float64 `json:"weight" db:"weight"`
	Dimensions string  `json:"dimensions,omitempty" db:"dimensions"`
	Barcode    *string `json:"barcode,omitempty" db:"barcode"`
	Notes      *string `json:"notes,omitempty" db:"notes"`

	// Status tracking
	Status           string `json:"status" db:"status"` // ORDERED, SHIPPED, DELIVERED, CANCELLED, RETURNED
	QuantityShipped  int    `json:"quantity_shipped" db:"quantity_shipped"`
	QuantityReturned int    `json:"quantity_returned" db:"quantity_returned"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Customer represents a customer in the system
type Customer struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	CustomerCode string     `json:"customer_code" db:"customer_code"`
	CompanyID    *uuid.UUID `json:"company_id,omitempty" db:"company_id"`
	Company      *Company   `json:"company,omitempty" db:"-"`
	Type         string     `json:"type" db:"type"` // INDIVIDUAL, BUSINESS, GOVERNMENT, NON_PROFIT

	// Basic information
	FirstName string  `json:"first_name" db:"first_name"`
	LastName  string  `json:"last_name" db:"last_name"`
	Email     string  `json:"email,omitempty" db:"email"`
	Phone     string  `json:"phone,omitempty" db:"phone"`
	Website   *string `json:"website,omitempty" db:"website"`

	// Business information (for business customers)
	CompanyName *string `json:"company_name,omitempty" db:"company_name"`
	TaxID       *string `json:"tax_id,omitempty" db:"tax_id"`
	Industry    *string `json:"industry,omitempty" db:"industry"`

	// Financial information
	CreditLimit decimal.Decimal `json:"credit_limit" db:"credit_limit"`
	CreditUsed  decimal.Decimal `json:"credit_used" db:"credit_used"`
	Terms       string          `json:"terms" db:"terms"` // NET30, NET60, etc.

	// Status and settings
	IsActive          bool   `json:"is_active" db:"is_active"`
	IsVATExempt       bool   `json:"is_vat_exempt" db:"is_vat_exempt"`
	PreferredCurrency string `json:"preferred_currency" db:"preferred_currency"`

	// Metadata
	Notes  *string `json:"notes,omitempty" db:"notes"`
	Source string  `json:"source" db:"source"` // WEB, PHONE, EMAIL, REFERRAL, etc.

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Relationships
	Addresses []OrderAddress `json:"addresses,omitempty" db:"-"`
	Orders    []Order        `json:"orders,omitempty" db:"-"`
}

// Company represents a company for business customers
type Company struct {
	ID          uuid.UUID `json:"id" db:"id"`
	CompanyName string    `json:"company_name" db:"company_name"`
	LegalName   string    `json:"legal_name" db:"legal_name"`
	TaxID       string    `json:"tax_id" db:"tax_id"`
	Industry    string    `json:"industry" db:"industry"`
	Website     *string   `json:"website,omitempty" db:"website"`
	Phone       string    `json:"phone" db:"phone"`
	Email       string    `json:"email" db:"email"`

	// Address information
	Address    string `json:"address" db:"address"`
	City       string `json:"city" db:"city"`
	State      string `json:"state" db:"state"`
	Country    string `json:"country" db:"country"`
	PostalCode string `json:"postal_code" db:"postal_code"`

	// Status
	IsActive bool `json:"is_active" db:"is_active"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// OrderAddress represents an address for orders
type OrderAddress struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	CustomerID *uuid.UUID `json:"customer_id,omitempty" db:"customer_id"`
	OrderID    *uuid.UUID `json:"order_id,omitempty" db:"order_id"`
	Type       string     `json:"type" db:"type"` // SHIPPING, BILLING, BOTH

	// Address fields
	FirstName    string  `json:"first_name" db:"first_name"`
	LastName     string  `json:"last_name" db:"last_name"`
	Company      *string `json:"company,omitempty" db:"company"`
	AddressLine1 string  `json:"address_line_1" db:"address_line_1"`
	AddressLine2 *string `json:"address_line_2,omitempty" db:"address_line_2"`
	City         string  `json:"city" db:"city"`
	State        string  `json:"state" db:"state"`
	PostalCode   string  `json:"postal_code" db:"postal_code"`
	Country      string  `json:"country" db:"country"`

	// Additional fields
	Phone        *string `json:"phone,omitempty" db:"phone"`
	Email        *string `json:"email,omitempty" db:"email"`
	Instructions *string `json:"instructions,omitempty" db:"instructions"`

	// Metadata
	IsDefault   bool `json:"is_default" db:"is_default"`
	IsActive    bool `json:"is_active" db:"is_active"`
	IsValidated bool `json:"is_validated" db:"is_validated"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// OrderValidation represents order validation results
type OrderValidation struct {
	IsValid  bool     `json:"is_valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// OrderCalculation represents order calculation breakdown
type OrderCalculation struct {
	Subtotal       decimal.Decimal `json:"subtotal"`
	TaxAmount      decimal.Decimal `json:"tax_amount"`
	ShippingAmount decimal.Decimal `json:"shipping_amount"`
	DiscountAmount decimal.Decimal `json:"discount_amount"`
	TotalAmount    decimal.Decimal `json:"total_amount"`

	// Tax breakdown by rate
	TaxBreakdown []TaxBreakdown `json:"tax_breakdown"`

	// Discount breakdown
	DiscountBreakdown []DiscountBreakdown `json:"discount_breakdown"`
}

// TaxBreakdown represents tax calculation details
type TaxBreakdown struct {
	TaxRate       decimal.Decimal `json:"tax_rate"`
	TaxAmount     decimal.Decimal `json:"tax_amount"`
	TaxableAmount decimal.Decimal `json:"taxable_amount"`
	TaxName       string          `json:"tax_name"`
}

// DiscountBreakdown represents discount calculation details
type DiscountBreakdown struct {
	DiscountType string          `json:"discount_type"` // PERCENTAGE, FIXED, COUPON
	Amount       decimal.Decimal `json:"amount"`
	Description  string          `json:"description"`
}

// OrderStatistics represents order statistics
type OrderStatistics struct {
	TotalOrders       int                   `json:"total_orders"`
	TotalAmount       decimal.Decimal       `json:"total_amount"`
	AverageOrderValue decimal.Decimal       `json:"average_order_value"`
	StatusBreakdown   map[OrderStatus]int   `json:"status_breakdown"`
	PaymentBreakdown  map[PaymentStatus]int `json:"payment_breakdown"`
}

// ==================== ORDER ENTITY METHODS ====================

// Validate validates the order entity
func (o *Order) Validate() error {
	var errs []error

	// Validate UUID
	if o.ID == uuid.Nil {
		errs = append(errs, errors.New("order ID cannot be empty"))
	}

	// Validate order number
	if err := o.validateOrderNumber(); err != nil {
		errs = append(errs, fmt.Errorf("invalid order number: %w", err))
	}

	// Validate customer ID
	if o.CustomerID == uuid.Nil {
		errs = append(errs, errors.New("customer ID cannot be empty"))
	}

	// Validate status
	if err := o.validateStatus(); err != nil {
		errs = append(errs, fmt.Errorf("invalid status: %w", err))
	}

	// Validate priority
	if err := o.validatePriority(); err != nil {
		errs = append(errs, fmt.Errorf("invalid priority: %w", err))
	}

	// Validate type
	if err := o.validateType(); err != nil {
		errs = append(errs, fmt.Errorf("invalid type: %w", err))
	}

	// Validate payment status
	if err := o.validatePaymentStatus(); err != nil {
		errs = append(errs, fmt.Errorf("invalid payment status: %w", err))
	}

	// Validate shipping method
	if err := o.validateShippingMethod(); err != nil {
		errs = append(errs, fmt.Errorf("invalid shipping method: %w", err))
	}

	// Validate financial amounts
	if err := o.validateFinancialAmounts(); err != nil {
		errs = append(errs, fmt.Errorf("invalid financial amounts: %w", err))
	}

	// Validate currency
	if err := o.validateCurrency(); err != nil {
		errs = append(errs, fmt.Errorf("invalid currency: %w", err))
	}

	// Validate dates
	if err := o.validateDates(); err != nil {
		errs = append(errs, fmt.Errorf("invalid dates: %w", err))
	}

	// Validate addresses
	if err := o.validateAddresses(); err != nil {
		errs = append(errs, fmt.Errorf("invalid addresses: %w", err))
	}

	// Validate notes
	if err := o.validateNotes(); err != nil {
		errs = append(errs, fmt.Errorf("invalid notes: %w", err))
	}

	// Validate tracking information
	if err := o.validateTrackingInfo(); err != nil {
		errs = append(errs, fmt.Errorf("invalid tracking information: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// validateOrderNumber validates the order number format
func (o *Order) validateOrderNumber() error {
	orderNumber := strings.TrimSpace(o.OrderNumber)
	if orderNumber == "" {
		return errors.New("order number cannot be empty")
	}

	if len(orderNumber) > 50 {
		return errors.New("order number cannot exceed 50 characters")
	}

	// Order number format: YEAR-SEQUENCE (e.g., 2024-001234)
	orderNumberRegex := regexp.MustCompile(`^\d{4}-\d{6}$`)
	if !orderNumberRegex.MatchString(orderNumber) {
		return errors.New("order number must be in format YYYY-NNNNNN")
	}

	return nil
}

// validateStatus validates the order status
func (o *Order) validateStatus() error {
	validStatuses := []OrderStatus{
		OrderStatusDraft, OrderStatusPending, OrderStatusConfirmed,
		OrderStatusProcessing, OrderStatusShipped, OrderStatusDelivered,
		OrderStatusCancelled, OrderStatusRefunded, OrderStatusReturned,
		OrderStatusOnHold, OrderStatusPartiallyShipped,
	}

	for _, validStatus := range validStatuses {
		if o.Status == validStatus {
			return nil
		}
	}

	return fmt.Errorf("invalid order status: %s", o.Status)
}

// validatePriority validates the order priority
func (o *Order) validatePriority() error {
	validPriorities := []OrderPriority{
		OrderPriorityLow, OrderPriorityNormal, OrderPriorityHigh,
		OrderPriorityUrgent, OrderPriorityCritical,
	}

	for _, validPriority := range validPriorities {
		if o.Priority == validPriority {
			return nil
		}
	}

	return fmt.Errorf("invalid order priority: %s", o.Priority)
}

// validateType validates the order type
func (o *Order) validateType() error {
	validTypes := []OrderType{
		OrderTypeSales, OrderTypePurchase, OrderTypeReturn,
		OrderTypeExchange, OrderTypeTransfer, OrderTypeAdjustment,
	}

	for _, validType := range validTypes {
		if o.Type == validType {
			return nil
		}
	}

	return fmt.Errorf("invalid order type: %s", o.Type)
}

// validatePaymentStatus validates the payment status
func (o *Order) validatePaymentStatus() error {
	validStatuses := []PaymentStatus{
		PaymentStatusPending, PaymentStatusPaid, PaymentStatusPartiallyPaid,
		PaymentStatusOverdue, PaymentStatusRefunded, PaymentStatusFailed,
	}

	for _, validStatus := range validStatuses {
		if o.PaymentStatus == validStatus {
			return nil
		}
	}

	return fmt.Errorf("invalid payment status: %s", o.PaymentStatus)
}

// validateShippingMethod validates the shipping method
func (o *Order) validateShippingMethod() error {
	validMethods := []ShippingMethod{
		ShippingMethodStandard, ShippingMethodExpress, ShippingMethodOvernight,
		ShippingMethodInternational, ShippingMethodPickup, ShippingMethodDigital,
	}

	for _, validMethod := range validMethods {
		if o.ShippingMethod == validMethod {
			return nil
		}
	}

	return fmt.Errorf("invalid shipping method: %s", o.ShippingMethod)
}

// validateFinancialAmounts validates financial amounts
func (o *Order) validateFinancialAmounts() error {
	// Subtotal validation
	if o.Subtotal.LessThan(decimal.Zero) {
		return errors.New("subtotal cannot be negative")
	}

	// Tax amount validation
	if o.TaxAmount.LessThan(decimal.Zero) {
		return errors.New("tax amount cannot be negative")
	}

	// Shipping amount validation
	if o.ShippingAmount.LessThan(decimal.Zero) {
		return errors.New("shipping amount cannot be negative")
	}

	// Discount amount validation
	if o.DiscountAmount.LessThan(decimal.Zero) {
		return errors.New("discount amount cannot be negative")
	}

	// Total amount validation
	if o.TotalAmount.LessThan(decimal.Zero) {
		return errors.New("total amount cannot be negative")
	}

	// Paid amount validation
	if o.PaidAmount.LessThan(decimal.Zero) {
		return errors.New("paid amount cannot be negative")
	}

	// Refunded amount validation
	if o.RefundedAmount.LessThan(decimal.Zero) {
		return errors.New("refunded amount cannot be negative")
	}

	// Business rule: Total = Subtotal + Tax + Shipping - Discount
	expectedTotal := o.Subtotal.Add(o.TaxAmount).Add(o.ShippingAmount).Sub(o.DiscountAmount)
	if !o.TotalAmount.Equal(expectedTotal) {
		return fmt.Errorf("total amount calculation mismatch: expected %s, got %s", expectedTotal, o.TotalAmount)
	}

	// Business rule: Paid amount cannot exceed total amount
	if o.PaidAmount.GreaterThan(o.TotalAmount) {
		return errors.New("paid amount cannot exceed total amount")
	}

	// Business rule: Refunded amount cannot exceed paid amount
	if o.RefundedAmount.GreaterThan(o.PaidAmount) {
		return errors.New("refunded amount cannot exceed paid amount")
	}

	return nil
}

// validateCurrency validates the currency code
func (o *Order) validateCurrency() error {
	currency := strings.TrimSpace(o.Currency)
	if currency == "" {
		return errors.New("currency cannot be empty")
	}

	// Basic currency code validation (3 letters)
	currencyRegex := regexp.MustCompile(`^[A-Z]{3}$`)
	if !currencyRegex.MatchString(currency) {
		return errors.New("currency must be a valid 3-letter ISO 4217 code")
	}

	return nil
}

// validateDates validates date fields
func (o *Order) validateDates() error {
	// Order date should not be in the future
	if o.OrderDate.After(time.Now().UTC()) {
		return errors.New("order date cannot be in the future")
	}

	// Required date should be after order date
	if o.RequiredDate != nil && o.RequiredDate.Before(o.OrderDate) {
		return errors.New("required date cannot be before order date")
	}

	// Shipping date should be after order date
	if o.ShippingDate != nil && o.ShippingDate.Before(o.OrderDate) {
		return errors.New("shipping date cannot be before order date")
	}

	// Delivery date should be after shipping date
	if o.DeliveryDate != nil && o.ShippingDate != nil && o.DeliveryDate.Before(*o.ShippingDate) {
		return errors.New("delivery date cannot be before shipping date")
	}

	// Cancelled date should be after order date
	if o.CancelledDate != nil && o.CancelledDate.Before(o.OrderDate) {
		return errors.New("cancelled date cannot be before order date")
	}

	return nil
}

// validateAddresses validates address references
func (o *Order) validateAddresses() error {
	if o.ShippingAddressID == uuid.Nil {
		return errors.New("shipping address ID cannot be empty")
	}

	if o.BillingAddressID == uuid.Nil {
		return errors.New("billing address ID cannot be empty")
	}

	return nil
}

// validateNotes validates note fields
func (o *Order) validateNotes() error {
	// Notes validation
	if o.Notes != nil {
		notes := strings.TrimSpace(*o.Notes)
		if len(notes) > 2000 {
			return errors.New("notes cannot exceed 2000 characters")
		}
	}

	// Internal notes validation
	if o.InternalNotes != nil {
		internalNotes := strings.TrimSpace(*o.InternalNotes)
		if len(internalNotes) > 2000 {
			return errors.New("internal notes cannot exceed 2000 characters")
		}
	}

	// Customer notes validation
	if o.CustomerNotes != nil {
		customerNotes := strings.TrimSpace(*o.CustomerNotes)
		if len(customerNotes) > 1000 {
			return errors.New("customer notes cannot exceed 1000 characters")
		}
	}

	return nil
}

// validateTrackingInfo validates tracking information
func (o *Order) validateTrackingInfo() error {
	// Tracking number validation
	if o.TrackingNumber != nil {
		trackingNumber := strings.TrimSpace(*o.TrackingNumber)
		if len(trackingNumber) > 100 {
			return errors.New("tracking number cannot exceed 100 characters")
		}

		// Basic tracking number format validation
		trackingRegex := regexp.MustCompile(`^[A-Za-z0-9\- ]+$`)
		if !trackingRegex.MatchString(trackingNumber) {
			return errors.New("tracking number contains invalid characters")
		}
	}

	// Carrier validation
	if o.Carrier != nil {
		carrier := strings.TrimSpace(*o.Carrier)
		if len(carrier) > 50 {
			return errors.New("carrier name cannot exceed 50 characters")
		}
	}

	return nil
}

// ==================== ORDER BUSINESS LOGIC METHODS ====================

// ChangeStatus changes the order status with validation
func (o *Order) ChangeStatus(newStatus OrderStatus, reason string) error {
	// Validate status transition
	if !IsValidStatusTransition(o.Status, newStatus) {
		return apperrors.NewInvalidTransitionError(string(o.Status), string(newStatus))
	}

	// Store previous status (make a copy to avoid pointer issues)
	prevStatus := o.Status
	o.PreviousStatus = &prevStatus

	// Update status
	o.Status = newStatus
	o.UpdatedAt = time.Now().UTC()

	// Update relevant date fields based on status
	switch newStatus {
	case OrderStatusCancelled:
		now := time.Now().UTC()
		o.CancelledDate = &now
	case OrderStatusShipped:
		now := time.Now().UTC()
		o.ShippingDate = &now
	case OrderStatusDelivered:
		now := time.Now().UTC()
		o.DeliveryDate = &now
	}

	return nil
}

// CanBeCancelled checks if the order can be cancelled
func (o *Order) CanBeCancelled() bool {
	// Orders in terminal statuses cannot be cancelled
	if IsTerminalStatus(o.Status) {
		return false
	}

	// Already cancelled orders cannot be cancelled again
	if o.Status == OrderStatusCancelled {
		return false
	}

	// Orders that are already shipped or delivered may have restrictions
	if o.Status == OrderStatusShipped || o.Status == OrderStatusDelivered {
		return false // May need special handling for returns
	}

	return true
}

// IsFullyPaid checks if the order is fully paid
func (o *Order) IsFullyPaid() bool {
	return o.PaidAmount.GreaterThanOrEqual(o.TotalAmount)
}

// IsPartiallyPaid checks if the order is partially paid
func (o *Order) IsPartiallyPaid() bool {
	return o.PaidAmount.GreaterThan(decimal.Zero) && !o.IsFullyPaid()
}

// GetOutstandingBalance returns the outstanding balance
func (o *Order) GetOutstandingBalance() decimal.Decimal {
	return o.TotalAmount.Sub(o.PaidAmount)
}

// AddPayment adds a payment amount to the order
func (o *Order) AddPayment(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("payment amount must be positive")
	}

	// Check if payment would exceed total amount
	newPaidAmount := o.PaidAmount.Add(amount)
	if newPaidAmount.GreaterThan(o.TotalAmount) {
		return fmt.Errorf("payment amount %s exceeds outstanding balance %s", amount, o.GetOutstandingBalance())
	}

	// Update paid amount
	o.PaidAmount = newPaidAmount
	o.UpdatedAt = time.Now().UTC()

	// Update payment status
	if o.IsFullyPaid() {
		o.PaymentStatus = PaymentStatusPaid
	} else if o.IsPartiallyPaid() {
		o.PaymentStatus = PaymentStatusPartiallyPaid
	}

	return nil
}

// AddRefund adds a refund amount to the order
func (o *Order) AddRefund(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("refund amount must be positive")
	}

	// Check if refund would exceed paid amount
	newRefundedAmount := o.RefundedAmount.Add(amount)
	if newRefundedAmount.GreaterThan(o.PaidAmount) {
		return fmt.Errorf("refund amount %s exceeds paid amount %s", amount, o.PaidAmount)
	}

	// Update refunded amount
	o.RefundedAmount = newRefundedAmount
	o.UpdatedAt = time.Now().UTC()

	// Update payment status if fully refunded
	if newRefundedAmount.Equal(o.PaidAmount) {
		o.PaymentStatus = PaymentStatusRefunded
		o.Status = OrderStatusRefunded
	}

	return nil
}

// CalculateTotals recalculates order totals based on items
func (o *Order) CalculateTotals() error {
	if len(o.Items) == 0 {
		return errors.New("order has no items to calculate totals")
	}

	// Calculate subtotal
	subtotal := decimal.Zero
	for _, item := range o.Items {
		subtotal = subtotal.Add(item.TotalPrice)
	}
	o.Subtotal = subtotal

	// Calculate total (tax and shipping should be calculated separately)
	o.TotalAmount = o.Subtotal.Add(o.TaxAmount).Add(o.ShippingAmount).Sub(o.DiscountAmount)

	o.UpdatedAt = time.Now().UTC()
	return nil
}

// GetTotalWeight calculates the total weight of all items
func (o *Order) GetTotalWeight() float64 {
	totalWeight := 0.0
	for _, item := range o.Items {
		totalWeight += item.Weight * float64(item.Quantity)
	}
	return totalWeight
}

// GetItemCount returns the total number of items
func (o *Order) GetItemCount() int {
	count := 0
	for _, item := range o.Items {
		count += item.Quantity
	}
	return count
}

// IsDigitalOrder checks if the order contains only digital products
func (o *Order) IsDigitalOrder() bool {
	if len(o.Items) == 0 {
		return false
	}

	// For now, we'll assume the shipping method determines this
	return o.ShippingMethod == ShippingMethodDigital
}

// SetPriority sets the order priority
func (o *Order) SetPriority(priority OrderPriority) error {
	if err := o.validatePriorityValue(priority); err != nil {
		return err
	}

	o.Priority = priority
	o.UpdatedAt = time.Now().UTC()
	return nil
}

// validatePriorityValue validates a priority value (helper method)
func (o *Order) validatePriorityValue(priority OrderPriority) error {
	validPriorities := []OrderPriority{
		OrderPriorityLow, OrderPriorityNormal, OrderPriorityHigh,
		OrderPriorityUrgent, OrderPriorityCritical,
	}

	for _, validPriority := range validPriorities {
		if priority == validPriority {
			return nil
		}
	}

	return fmt.Errorf("invalid priority: %s", priority)
}

// UpdateTracking updates tracking information
func (o *Order) UpdateTracking(trackingNumber, carrier string) error {
	if trackingNumber != "" {
		if err := o.validateTrackingNumber(trackingNumber); err != nil {
			return fmt.Errorf("invalid tracking number: %w", err)
		}
		o.TrackingNumber = &trackingNumber
	}

	if carrier != "" {
		if len(carrier) > 50 {
			return errors.New("carrier name cannot exceed 50 characters")
		}
		o.Carrier = &carrier
	}

	o.UpdatedAt = time.Now().UTC()
	return nil
}

// validateTrackingNumber validates a tracking number (helper method)
func (o *Order) validateTrackingNumber(trackingNumber string) error {
	if len(trackingNumber) > 100 {
		return errors.New("tracking number cannot exceed 100 characters")
	}

	trackingRegex := regexp.MustCompile(`^[A-Za-z0-9\- ]+$`)
	if !trackingRegex.MatchString(trackingNumber) {
		return errors.New("tracking number contains invalid characters")
	}

	return nil
}

// ==================== ORDER ITEM ENTITY METHODS ====================

// Validate validates the order item entity
func (oi *OrderItem) Validate() error {
	var errs []error

	// Validate UUID
	if oi.ID == uuid.Nil {
		errs = append(errs, errors.New("order item ID cannot be empty"))
	}

	// Validate order ID
	if oi.OrderID == uuid.Nil {
		errs = append(errs, errors.New("order ID cannot be empty"))
	}

	// Validate product ID
	if oi.ProductID == uuid.Nil {
		errs = append(errs, errors.New("product ID cannot be empty"))
	}

	// Validate product SKU
	if err := oi.validateProductSKU(); err != nil {
		errs = append(errs, fmt.Errorf("invalid product SKU: %w", err))
	}

	// Validate product name
	if err := oi.validateProductName(); err != nil {
		errs = append(errs, fmt.Errorf("invalid product name: %w", err))
	}

	// Validate quantity
	if err := oi.validateQuantity(); err != nil {
		errs = append(errs, fmt.Errorf("invalid quantity: %w", err))
	}

	// Validate unit price
	if err := oi.validateUnitPrice(); err != nil {
		errs = append(errs, fmt.Errorf("invalid unit price: %w", err))
	}

	// Validate discount amount
	if err := oi.validateDiscountAmount(); err != nil {
		errs = append(errs, fmt.Errorf("invalid discount amount: %w", err))
	}

	// Validate tax rate
	if err := oi.validateTaxRate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid tax rate: %w", err))
	}

	// Validate tax amount
	if err := oi.validateTaxAmount(); err != nil {
		errs = append(errs, fmt.Errorf("invalid tax amount: %w", err))
	}

	// Validate total price
	if err := oi.validateTotalPrice(); err != nil {
		errs = append(errs, fmt.Errorf("invalid total price: %w", err))
	}

	// Validate weight
	if err := oi.validateWeight(); err != nil {
		errs = append(errs, fmt.Errorf("invalid weight: %w", err))
	}

	// Validate dimensions
	if err := oi.validateDimensions(); err != nil {
		errs = append(errs, fmt.Errorf("invalid dimensions: %w", err))
	}

	// Validate barcode
	if err := oi.validateBarcode(); err != nil {
		errs = append(errs, fmt.Errorf("invalid barcode: %w", err))
	}

	// Validate notes
	if err := oi.validateItemNotes(); err != nil {
		errs = append(errs, fmt.Errorf("invalid notes: %w", err))
	}

	// Validate status
	if err := oi.validateItemStatus(); err != nil {
		errs = append(errs, fmt.Errorf("invalid status: %w", err))
	}

	// Validate shipped and returned quantities
	if err := oi.validateQuantities(); err != nil {
		errs = append(errs, fmt.Errorf("invalid quantities: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("order item validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// validateProductSKU validates the product SKU
func (oi *OrderItem) validateProductSKU() error {
	sku := strings.TrimSpace(oi.ProductSKU)
	if sku == "" {
		return errors.New("product SKU cannot be empty")
	}

	if len(sku) > 100 {
		return errors.New("product SKU cannot exceed 100 characters")
	}

	return nil
}

// validateProductName validates the product name
func (oi *OrderItem) validateProductName() error {
	name := strings.TrimSpace(oi.ProductName)
	if name == "" {
		return errors.New("product name cannot be empty")
	}

	if len(name) > 300 {
		return errors.New("product name cannot exceed 300 characters")
	}

	return nil
}

// validateQuantity validates the quantity
func (oi *OrderItem) validateQuantity() error {
	if oi.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	if oi.Quantity > 9999 {
		return errors.New("quantity cannot exceed 9999")
	}

	return nil
}

// validateUnitPrice validates the unit price
func (oi *OrderItem) validateUnitPrice() error {
	if oi.UnitPrice.LessThan(decimal.Zero) {
		return errors.New("unit price cannot be negative")
	}

	if oi.UnitPrice.GreaterThan(decimal.NewFromFloat(999999.99)) {
		return errors.New("unit price cannot exceed 999999.99")
	}

	return nil
}

// validateDiscountAmount validates the discount amount
func (oi *OrderItem) validateDiscountAmount() error {
	if oi.DiscountAmount.LessThan(decimal.Zero) {
		return errors.New("discount amount cannot be negative")
	}

	// Discount cannot exceed unit price
	if oi.DiscountAmount.GreaterThan(oi.UnitPrice) {
		return errors.New("discount amount cannot exceed unit price")
	}

	return nil
}

// validateTaxRate validates the tax rate
func (oi *OrderItem) validateTaxRate() error {
	if oi.TaxRate.LessThan(decimal.Zero) {
		return errors.New("tax rate cannot be negative")
	}

	if oi.TaxRate.GreaterThan(decimal.NewFromFloat(100)) {
		return errors.New("tax rate cannot exceed 100%")
	}

	return nil
}

// validateTaxAmount validates the tax amount
func (oi *OrderItem) validateTaxAmount() error {
	if oi.TaxAmount.LessThan(decimal.Zero) {
		return errors.New("tax amount cannot be negative")
	}

	return nil
}

// validateTotalPrice validates the total price
func (oi *OrderItem) validateTotalPrice() error {
	if oi.TotalPrice.LessThan(decimal.Zero) {
		return errors.New("total price cannot be negative")
	}

	// Calculate expected total price
	expectedTotal := oi.UnitPrice.Mul(decimal.NewFromInt(int64(oi.Quantity))).Sub(oi.DiscountAmount).Add(oi.TaxAmount)
	if !oi.TotalPrice.Equal(expectedTotal) {
		return fmt.Errorf("total price calculation mismatch: expected %s, got %s", expectedTotal, oi.TotalPrice)
	}

	return nil
}

// validateWeight validates the weight
func (oi *OrderItem) validateWeight() error {
	if oi.Weight < 0 {
		return errors.New("weight cannot be negative")
	}

	if oi.Weight > 999999.99 {
		return errors.New("weight cannot exceed 999999.99")
	}

	return nil
}

// validateDimensions validates the dimensions
func (oi *OrderItem) validateDimensions() error {
	if oi.Dimensions != "" {
		dimensions := strings.TrimSpace(oi.Dimensions)
		if len(dimensions) > 100 {
			return errors.New("dimensions cannot exceed 100 characters")
		}

		// Basic dimensions format validation
		dimRegex := regexp.MustCompile(`^\d+(\.\d+)?\s*[xX]\s*\d+(\.\d+)?\s*[xX]\s*\d+(\.\d+)?\s*$`)
		if !dimRegex.MatchString(dimensions) {
			return errors.New("dimensions must be in format 'L x W x H'")
		}
	}

	return nil
}

// validateBarcode validates the barcode
func (oi *OrderItem) validateBarcode() error {
	if oi.Barcode != nil {
		barcode := strings.TrimSpace(*oi.Barcode)
		if barcode == "" {
			return nil // Empty barcode is allowed
		}

		if len(barcode) > 50 {
			return errors.New("barcode cannot exceed 50 characters")
		}

		barcodeRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
		if !barcodeRegex.MatchString(barcode) {
			return errors.New("barcode can only contain letters, numbers, hyphens, and underscores")
		}
	}

	return nil
}

// validateItemNotes validates the item notes
func (oi *OrderItem) validateItemNotes() error {
	if oi.Notes != nil {
		notes := strings.TrimSpace(*oi.Notes)
		if len(notes) > 500 {
			return errors.New("notes cannot exceed 500 characters")
		}
	}

	return nil
}

// validateItemStatus validates the item status
func (oi *OrderItem) validateItemStatus() error {
	validStatuses := []string{"ORDERED", "SHIPPED", "DELIVERED", "CANCELLED", "RETURNED"}

	for _, validStatus := range validStatuses {
		if oi.Status == validStatus {
			return nil
		}
	}

	return fmt.Errorf("invalid item status: %s", oi.Status)
}

// validateQuantities validates shipped and returned quantities
func (oi *OrderItem) validateQuantities() error {
	if oi.QuantityShipped < 0 {
		return errors.New("shipped quantity cannot be negative")
	}

	if oi.QuantityShipped > oi.Quantity {
		return fmt.Errorf("shipped quantity (%d) cannot exceed ordered quantity (%d)", oi.QuantityShipped, oi.Quantity)
	}

	if oi.QuantityReturned < 0 {
		return errors.New("returned quantity cannot be negative")
	}

	if oi.QuantityReturned > oi.QuantityShipped {
		return fmt.Errorf("returned quantity (%d) cannot exceed shipped quantity (%d)", oi.QuantityReturned, oi.QuantityShipped)
	}

	return nil
}

// CalculateTotals calculates item totals based on unit price, quantity, discount, and tax
func (oi *OrderItem) CalculateTotals() {
	// Calculate subtotal before discount
	subtotal := oi.UnitPrice.Mul(decimal.NewFromInt(int64(oi.Quantity)))

	// Apply discount (DiscountAmount is per-item, so multiply by quantity)
	totalDiscount := oi.DiscountAmount.Mul(decimal.NewFromInt(int64(oi.Quantity)))
	afterDiscount := subtotal.Sub(totalDiscount)

	// Calculate tax
	oi.TaxAmount = afterDiscount.Mul(oi.TaxRate).Div(decimal.NewFromInt(100))

	// Calculate total
	oi.TotalPrice = afterDiscount.Add(oi.TaxAmount)

	oi.UpdatedAt = time.Now().UTC()
}

// GetItemWeight returns the total weight for this item
func (oi *OrderItem) GetItemWeight() float64 {
	return oi.Weight * float64(oi.Quantity)
}

// CanBeShipped checks if the item can be shipped
func (oi *OrderItem) CanBeShipped() bool {
	return oi.Status == "ORDERED" && oi.QuantityShipped < oi.Quantity
}

// ShipItem ships a quantity of the item
func (oi *OrderItem) ShipItem(quantity int) error {
	if quantity <= 0 {
		return errors.New("shipping quantity must be positive")
	}

	if oi.QuantityShipped+quantity > oi.Quantity {
		return fmt.Errorf("cannot ship %d items, only %d remaining", quantity, oi.Quantity-oi.QuantityShipped)
	}

	oi.QuantityShipped += quantity
	oi.UpdatedAt = time.Now().UTC()

	// Update status if fully shipped
	if oi.QuantityShipped == oi.Quantity {
		oi.Status = "SHIPPED"
	} else if oi.QuantityShipped > 0 {
		oi.Status = "PARTIALLY_SHIPPED"
	}

	return nil
}

// ReturnItem returns a quantity of the item
func (oi *OrderItem) ReturnItem(quantity int) error {
	if quantity <= 0 {
		return errors.New("return quantity must be positive")
	}

	if oi.QuantityReturned+quantity > oi.QuantityShipped {
		return fmt.Errorf("cannot return %d items, only %d shipped", quantity, oi.QuantityShipped-oi.QuantityReturned)
	}

	oi.QuantityReturned += quantity
	oi.UpdatedAt = time.Now().UTC()

	// Update status if all shipped items are returned
	if oi.QuantityReturned == oi.QuantityShipped {
		oi.Status = "RETURNED"
	}

	return nil
}

// ==================== CUSTOMER ENTITY METHODS ====================

// Validate validates the customer entity
func (c *Customer) Validate() error {
	var errs []error

	// Validate UUID
	if c.ID == uuid.Nil {
		errs = append(errs, errors.New("customer ID cannot be empty"))
	}

	// Validate customer code
	if err := c.validateCustomerCode(); err != nil {
		errs = append(errs, fmt.Errorf("invalid customer code: %w", err))
	}

	// Validate type
	if err := c.validateCustomerType(); err != nil {
		errs = append(errs, fmt.Errorf("invalid customer type: %w", err))
	}

	// Validate name fields
	if err := c.validateNames(); err != nil {
		errs = append(errs, fmt.Errorf("invalid names: %w", err))
	}

	// Validate contact information
	if err := c.validateContactInfo(); err != nil {
		errs = append(errs, fmt.Errorf("invalid contact information: %w", err))
	}

	// Validate business information
	if err := c.validateBusinessInfo(); err != nil {
		errs = append(errs, fmt.Errorf("invalid business information: %w", err))
	}

	// Validate financial information
	if err := c.validateFinancialInfo(); err != nil {
		errs = append(errs, fmt.Errorf("invalid financial information: %w", err))
	}

	// Validate settings
	if err := c.validateSettings(); err != nil {
		errs = append(errs, fmt.Errorf("invalid settings: %w", err))
	}

	// Validate metadata
	if err := c.validateMetadata(); err != nil {
		errs = append(errs, fmt.Errorf("invalid metadata: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("customer validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// validateCustomerCode validates the customer code
func (c *Customer) validateCustomerCode() error {
	code := strings.TrimSpace(c.CustomerCode)
	if code == "" {
		return errors.New("customer code cannot be empty")
	}

	if len(code) > 50 {
		return errors.New("customer code cannot exceed 50 characters")
	}

	// Customer code format validation
	codeRegex := regexp.MustCompile(`^[A-Z0-9\-_]+$`)
	if !codeRegex.MatchString(code) {
		return errors.New("customer code can only contain uppercase letters, numbers, hyphens, and underscores")
	}

	return nil
}

// validateCustomerType validates the customer type
func (c *Customer) validateCustomerType() error {
	validTypes := []string{"INDIVIDUAL", "BUSINESS", "GOVERNMENT", "NON_PROFIT"}

	for _, validType := range validTypes {
		if c.Type == validType {
			return nil
		}
	}

	return fmt.Errorf("invalid customer type: %s", c.Type)
}

// validateNames validates name fields
func (c *Customer) validateNames() error {
	// First name validation
	if strings.TrimSpace(c.FirstName) == "" {
		return errors.New("first name cannot be empty")
	}
	if len(c.FirstName) > 100 {
		return errors.New("first name cannot exceed 100 characters")
	}

	// Last name validation
	if strings.TrimSpace(c.LastName) == "" {
		return errors.New("last name cannot be empty")
	}
	if len(c.LastName) > 100 {
		return errors.New("last name cannot exceed 100 characters")
	}

	// Company name validation (for business customers)
	if c.CompanyName != nil {
		companyName := strings.TrimSpace(*c.CompanyName)
		if companyName != "" && len(companyName) > 200 {
			return errors.New("company name cannot exceed 200 characters")
		}
	}

	return nil
}

// validateContactInfo validates contact information
func (c *Customer) validateContactInfo() error {
	// Email validation
	if c.Email != "" {
		if err := c.validateEmail(); err != nil {
			return err
		}
	}

	// Phone validation
	if c.Phone != "" {
		if err := c.validatePhone(); err != nil {
			return err
		}
	}

	// Website validation
	if c.Website != nil {
		if err := c.validateWebsite(); err != nil {
			return err
		}
	}

	return nil
}

// validateEmail validates the email format
func (c *Customer) validateEmail() error {
	email := strings.TrimSpace(c.Email)
	if email == "" {
		return nil // Email is optional
	}

	if len(email) > 255 {
		return errors.New("email cannot exceed 255 characters")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}

	return nil
}

// validatePhone validates the phone number format
func (c *Customer) validatePhone() error {
	phone := strings.TrimSpace(c.Phone)
	if phone == "" {
		return nil // Phone is optional
	}

	if len(phone) > 50 {
		return errors.New("phone cannot exceed 50 characters")
	}

	phoneRegex := regexp.MustCompile(`^\+?[\d\s\-\(\)]{7,20}$`)
	if !phoneRegex.MatchString(phone) {
		return errors.New("invalid phone number format")
	}

	return nil
}

// validateWebsite validates the website URL
func (c *Customer) validateWebsite() error {
	website := strings.TrimSpace(*c.Website)
	if website == "" {
		return nil // Website is optional
	}

	if len(website) > 500 {
		return errors.New("website cannot exceed 500 characters")
	}

	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(website) {
		return errors.New("invalid website URL format")
	}

	return nil
}

// validateBusinessInfo validates business information
func (c *Customer) validateBusinessInfo() error {
	// Tax ID validation
	if c.TaxID != nil {
		taxID := strings.TrimSpace(*c.TaxID)
		if taxID != "" && len(taxID) > 50 {
			return errors.New("tax ID cannot exceed 50 characters")
		}
	}

	// Industry validation
	if c.Industry != nil {
		industry := strings.TrimSpace(*c.Industry)
		if industry != "" && len(industry) > 100 {
			return errors.New("industry cannot exceed 100 characters")
		}
	}

	return nil
}

// validateFinancialInfo validates financial information
func (c *Customer) validateFinancialInfo() error {
	// Credit limit validation
	if c.CreditLimit.LessThan(decimal.Zero) {
		return errors.New("credit limit cannot be negative")
	}

	if c.CreditLimit.GreaterThan(decimal.NewFromFloat(999999999.99)) {
		return errors.New("credit limit cannot exceed 999,999,999.99")
	}

	// Credit used validation
	if c.CreditUsed.LessThan(decimal.Zero) {
		return errors.New("credit used cannot be negative")
	}

	if c.CreditUsed.GreaterThan(c.CreditLimit) {
		return errors.New("credit used cannot exceed credit limit")
	}

	// Terms validation
	if strings.TrimSpace(c.Terms) == "" {
		return errors.New("payment terms cannot be empty")
	}

	if len(c.Terms) > 20 {
		return errors.New("payment terms cannot exceed 20 characters")
	}

	// Validate terms format
	termsRegex := regexp.MustCompile(`^NET\d+$`)
	if !termsRegex.MatchString(strings.ToUpper(c.Terms)) {
		return errors.New("payment terms must be in format NET30, NET60, etc.")
	}

	return nil
}

// validateSettings validates customer settings
func (c *Customer) validateSettings() error {
	// Preferred currency validation
	currency := strings.TrimSpace(c.PreferredCurrency)
	if currency == "" {
		return errors.New("preferred currency cannot be empty")
	}

	currencyRegex := regexp.MustCompile(`^[A-Z]{3}$`)
	if !currencyRegex.MatchString(currency) {
		return errors.New("currency must be a valid 3-letter ISO 4217 code")
	}

	return nil
}

// validateMetadata validates metadata fields
func (c *Customer) validateMetadata() error {
	// Notes validation
	if c.Notes != nil {
		notes := strings.TrimSpace(*c.Notes)
		if len(notes) > 2000 {
			return errors.New("notes cannot exceed 2000 characters")
		}
	}

	// Source validation
	if strings.TrimSpace(c.Source) == "" {
		return errors.New("source cannot be empty")
	}

	validSources := []string{"WEB", "PHONE", "EMAIL", "REFERRAL", "WALK_IN", "SOCIAL", "ADVERTISEMENT", "OTHER"}
	for _, validSource := range validSources {
		if c.Source == validSource {
			return nil
		}
	}

	return fmt.Errorf("invalid source: %s", c.Source)
}

// GetFullName returns the customer's full name
func (c *Customer) GetFullName() string {
	// Always return the person's name, regardless of customer type
	return fmt.Sprintf("%s %s", strings.TrimSpace(c.FirstName), strings.TrimSpace(c.LastName))
}

// GetDisplayName returns the display name based on customer type
func (c *Customer) GetDisplayName() string {
	if c.Type == "BUSINESS" && c.CompanyName != nil && strings.TrimSpace(*c.CompanyName) != "" {
		return *c.CompanyName
	}
	return c.GetFullName()
}

// GetAvailableCredit returns the available credit
func (c *Customer) GetAvailableCredit() decimal.Decimal {
	return c.CreditLimit.Sub(c.CreditUsed)
}

// HasAvailableCredit checks if customer has available credit for a given amount
func (c *Customer) HasAvailableCredit(amount decimal.Decimal) bool {
	return c.GetAvailableCredit().GreaterThanOrEqual(amount)
}

// UseCredit uses a specified amount of credit
func (c *Customer) UseCredit(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("credit amount must be positive")
	}

	if !c.HasAvailableCredit(amount) {
		return fmt.Errorf("insufficient credit: available %s, requested %s", c.GetAvailableCredit(), amount)
	}

	c.CreditUsed = c.CreditUsed.Add(amount)
	c.UpdatedAt = time.Now().UTC()
	return nil
}

// ReleaseCredit releases a specified amount of credit (e.g., after payment)
func (c *Customer) ReleaseCredit(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("credit amount must be positive")
	}

	newCreditUsed := c.CreditUsed.Sub(amount)
	if newCreditUsed.LessThan(decimal.Zero) {
		return errors.New("cannot release more credit than used")
	}

	c.CreditUsed = newCreditUsed
	c.UpdatedAt = time.Now().UTC()
	return nil
}

// ==================== ORDER ADDRESS ENTITY METHODS ====================

// Validate validates the order address entity
func (oa *OrderAddress) Validate() error {
	var errs []error

	// Validate UUID
	if oa.ID == uuid.Nil {
		errs = append(errs, errors.New("address ID cannot be empty"))
	}

	// Validate relationships (at least one must be set)
	if oa.CustomerID == nil && oa.OrderID == nil {
		errs = append(errs, errors.New("address must be associated with either a customer or an order"))
	}

	// Validate type
	if err := oa.validateAddressType(); err != nil {
		errs = append(errs, fmt.Errorf("invalid address type: %w", err))
	}

	// Validate name fields
	if err := oa.validateAddressNames(); err != nil {
		errs = append(errs, fmt.Errorf("invalid names: %w", err))
	}

	// Validate address fields
	if err := oa.validateAddressFields(); err != nil {
		errs = append(errs, fmt.Errorf("invalid address fields: %w", err))
	}

	// Validate company name
	if err := oa.validateAddressCompany(); err != nil {
		errs = append(errs, fmt.Errorf("invalid company: %w", err))
	}

	// Validate contact information
	if err := oa.validateAddressContactInfo(); err != nil {
		errs = append(errs, fmt.Errorf("invalid contact information: %w", err))
	}

	// Validate instructions
	if err := oa.validateAddressInstructions(); err != nil {
		errs = append(errs, fmt.Errorf("invalid instructions: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("address validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// validateAddressType validates the address type
func (oa *OrderAddress) validateAddressType() error {
	validTypes := []string{"SHIPPING", "BILLING", "BOTH"}

	for _, validType := range validTypes {
		if oa.Type == validType {
			return nil
		}
	}

	return fmt.Errorf("invalid address type: %s", oa.Type)
}

// validateAddressNames validates name fields
func (oa *OrderAddress) validateAddressNames() error {
	// First name validation
	if strings.TrimSpace(oa.FirstName) == "" {
		return errors.New("first name cannot be empty")
	}
	if len(oa.FirstName) > 100 {
		return errors.New("first name cannot exceed 100 characters")
	}

	// Last name validation
	if strings.TrimSpace(oa.LastName) == "" {
		return errors.New("last name cannot be empty")
	}
	if len(oa.LastName) > 100 {
		return errors.New("last name cannot exceed 100 characters")
	}

	return nil
}

// validateAddressFields validates address fields
func (oa *OrderAddress) validateAddressFields() error {
	// Address line 1 validation
	if strings.TrimSpace(oa.AddressLine1) == "" {
		return errors.New("address line 1 cannot be empty")
	}
	if len(oa.AddressLine1) > 255 {
		return errors.New("address line 1 cannot exceed 255 characters")
	}

	// Address line 2 validation (optional)
	if oa.AddressLine2 != nil {
		addressLine2 := strings.TrimSpace(*oa.AddressLine2)
		if addressLine2 != "" && len(addressLine2) > 255 {
			return errors.New("address line 2 cannot exceed 255 characters")
		}
	}

	// City validation
	if strings.TrimSpace(oa.City) == "" {
		return errors.New("city cannot be empty")
	}
	if len(oa.City) > 100 {
		return errors.New("city cannot exceed 100 characters")
	}

	// State validation
	if strings.TrimSpace(oa.State) == "" {
		return errors.New("state cannot be empty")
	}
	if len(oa.State) > 100 {
		return errors.New("state cannot exceed 100 characters")
	}

	// Postal code validation
	if strings.TrimSpace(oa.PostalCode) == "" {
		return errors.New("postal code cannot be empty")
	}
	if len(oa.PostalCode) > 20 {
		return errors.New("postal code cannot exceed 20 characters")
	}

	// Country validation
	if strings.TrimSpace(oa.Country) == "" {
		return errors.New("country cannot be empty")
	}
	if len(oa.Country) > 100 {
		return errors.New("country cannot exceed 100 characters")
	}

	return nil
}

// validateAddressCompany validates the company name
func (oa *OrderAddress) validateAddressCompany() error {
	if oa.Company != nil {
		company := strings.TrimSpace(*oa.Company)
		if company != "" && len(company) > 200 {
			return errors.New("company name cannot exceed 200 characters")
		}
	}

	return nil
}

// validateAddressContactInfo validates contact information
func (oa *OrderAddress) validateAddressContactInfo() error {
	// Phone validation
	if oa.Phone != nil {
		phone := strings.TrimSpace(*oa.Phone)
		if phone != "" {
			if len(phone) > 50 {
				return errors.New("phone cannot exceed 50 characters")
			}

			phoneRegex := regexp.MustCompile(`^\+?[\d\s\-\(\)]{7,20}$`)
			if !phoneRegex.MatchString(phone) {
				return errors.New("invalid phone number format")
			}
		}
	}

	// Email validation
	if oa.Email != nil {
		email := strings.TrimSpace(*oa.Email)
		if email != "" {
			if len(email) > 255 {
				return errors.New("email cannot exceed 255 characters")
			}

			emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
			if !emailRegex.MatchString(email) {
				return errors.New("invalid email format")
			}
		}
	}

	return nil
}

// validateAddressInstructions validates delivery instructions
func (oa *OrderAddress) validateAddressInstructions() error {
	if oa.Instructions != nil {
		instructions := strings.TrimSpace(*oa.Instructions)
		if instructions != "" && len(instructions) > 500 {
			return errors.New("instructions cannot exceed 500 characters")
		}
	}

	return nil
}

// GetFullName returns the full name for the address
func (oa *OrderAddress) GetFullName() string {
	return fmt.Sprintf("%s %s", strings.TrimSpace(oa.FirstName), strings.TrimSpace(oa.LastName))
}

// GetFullAddress returns the complete formatted address
func (oa *OrderAddress) GetFullAddress() string {
	var addressParts []string

	// Add company name if present
	if oa.Company != nil && strings.TrimSpace(*oa.Company) != "" {
		addressParts = append(addressParts, strings.TrimSpace(*oa.Company))
	}

	// Add name
	addressParts = append(addressParts, oa.GetFullName())

	// Add address lines
	addressParts = append(addressParts, oa.AddressLine1)
	if oa.AddressLine2 != nil && strings.TrimSpace(*oa.AddressLine2) != "" {
		addressParts = append(addressParts, strings.TrimSpace(*oa.AddressLine2))
	}

	// Add city, state, postal code
	cityStateZip := fmt.Sprintf("%s, %s %s", oa.City, oa.State, oa.PostalCode)
	addressParts = append(addressParts, cityStateZip)

	// Add country
	addressParts = append(addressParts, oa.Country)

	return strings.Join(addressParts, "\n")
}

// GetSingleLineAddress returns the address in a single line format
func (oa *OrderAddress) GetSingleLineAddress() string {
	parts := []string{oa.AddressLine1}

	if oa.AddressLine2 != nil && strings.TrimSpace(*oa.AddressLine2) != "" {
		parts = append(parts, strings.TrimSpace(*oa.AddressLine2))
	}

	parts = append(parts, oa.City, oa.State, oa.PostalCode, oa.Country)

	return strings.Join(parts, ", ")
}

// IsShippingAddress checks if this is a shipping address
func (oa *OrderAddress) IsShippingAddress() bool {
	return oa.Type == "SHIPPING" || oa.Type == "BOTH"
}

// IsBillingAddress checks if this is a billing address
func (oa *OrderAddress) IsBillingAddress() bool {
	return oa.Type == "BILLING" || oa.Type == "BOTH"
}

// SetAsDefault sets this address as the default
func (oa *OrderAddress) SetAsDefault() {
	oa.IsDefault = true
	oa.UpdatedAt = time.Now().UTC()
}

// UnsetDefault removes default status
func (oa *OrderAddress) UnsetDefault() {
	oa.IsDefault = false
	oa.UpdatedAt = time.Now().UTC()
}

// ValidateAddress performs address validation (could integrate with external services)
func (oa *OrderAddress) ValidateAddress() bool {
	// Basic validation - in a real implementation, this could call
	// external address validation services like USPS, Google Maps API, etc.

	// For now, just check if all required fields are present
	if strings.TrimSpace(oa.AddressLine1) == "" ||
		strings.TrimSpace(oa.City) == "" ||
		strings.TrimSpace(oa.State) == "" ||
		strings.TrimSpace(oa.PostalCode) == "" ||
		strings.TrimSpace(oa.Country) == "" {
		return false
	}

	// Basic postal code validation based on country
	switch strings.ToUpper(oa.Country) {
	case "US", "USA", "UNITED STATES":
		// US ZIP code validation (5 digits or 5+4 format)
		usZipRegex := regexp.MustCompile(`^\d{5}(-\d{4})?$`)
		return usZipRegex.MatchString(strings.TrimSpace(oa.PostalCode))
	case "CA", "CAN", "CANADA":
		// Canadian postal code validation (A1A 1A1 or A1A1A1 format)
		caPostalRegex := regexp.MustCompile(`^[A-Z]\d[A-Z]\s?\d[A-Z]\d$`)
		return caPostalRegex.MatchString(strings.ToUpper(oa.PostalCode))
	case "GB", "UK", "UNITED KINGDOM":
		// UK postcode validation (basic format)
		ukPostcodeRegex := regexp.MustCompile(`^[A-Z]{1,2}\d[A-Z\d]? \d[A-Z]{2}$`)
		return ukPostcodeRegex.MatchString(strings.ToUpper(oa.PostalCode))
	default:
		// For other countries, just check if postal code has reasonable format
		return len(strings.TrimSpace(oa.PostalCode)) >= 3 && len(strings.TrimSpace(oa.PostalCode)) <= 20
	}
}

// MarkAsValidated marks the address as validated
func (oa *OrderAddress) MarkAsValidated() {
	oa.IsValidated = true
	oa.UpdatedAt = time.Now().UTC()
}

// MarkAsUnvalidated marks the address as not validated
func (oa *OrderAddress) MarkAsUnvalidated() {
	oa.IsValidated = false
	oa.UpdatedAt = time.Now().UTC()
}

// ==================== ORDER CALCULATION METHODS ====================

// CalculateOrderTotals performs comprehensive order calculations
func CalculateOrderTotals(order *Order, taxRate decimal.Decimal, shippingCost decimal.Decimal) (*OrderCalculation, error) {
	if len(order.Items) == 0 {
		return nil, errors.New("order has no items to calculate")
	}

	calculation := &OrderCalculation{
		Subtotal:          decimal.Zero,
		TaxAmount:         decimal.Zero,
		ShippingAmount:    shippingCost,
		DiscountAmount:    decimal.Zero,
		TaxBreakdown:      []TaxBreakdown{},
		DiscountBreakdown: []DiscountBreakdown{},
	}

	// Calculate subtotal and tax breakdown
	for _, item := range order.Items {
		// Calculate item subtotal
		itemSubtotal := item.UnitPrice.Mul(decimal.NewFromInt(int64(item.Quantity)))
		calculation.Subtotal = calculation.Subtotal.Add(itemSubtotal)

		// Calculate item discount
		if item.DiscountAmount.GreaterThan(decimal.Zero) {
			calculation.DiscountAmount = calculation.DiscountAmount.Add(item.DiscountAmount)
			calculation.DiscountBreakdown = append(calculation.DiscountBreakdown, DiscountBreakdown{
				DiscountType: "ITEM_DISCOUNT",
				Amount:       item.DiscountAmount,
				Description:  fmt.Sprintf("Discount on %s", item.ProductName),
			})
		}

		// Calculate tax for taxable items
		if item.TaxRate.GreaterThan(decimal.Zero) {
			taxableAmount := itemSubtotal.Sub(item.DiscountAmount)
			itemTax := taxableAmount.Mul(item.TaxRate).Div(decimal.NewFromInt(100))
			calculation.TaxAmount = calculation.TaxAmount.Add(itemTax)

			calculation.TaxBreakdown = append(calculation.TaxBreakdown, TaxBreakdown{
				TaxRate:       item.TaxRate,
				TaxAmount:     itemTax,
				TaxableAmount: taxableAmount,
				TaxName:       fmt.Sprintf("Tax @ %s%%", item.TaxRate.String()),
			})
		}
	}

	// Apply order-level discount if any
	if order.DiscountAmount.GreaterThan(decimal.Zero) {
		calculation.DiscountAmount = calculation.DiscountAmount.Add(order.DiscountAmount)
		calculation.DiscountBreakdown = append(calculation.DiscountBreakdown, DiscountBreakdown{
			DiscountType: "ORDER_DISCOUNT",
			Amount:       order.DiscountAmount,
			Description:  "Order level discount",
		})
	}

	// Calculate total
	calculation.TotalAmount = calculation.Subtotal.Add(calculation.TaxAmount).Add(calculation.ShippingAmount).Sub(calculation.DiscountAmount)

	return calculation, nil
}

// ValidateOrder validates an entire order including all items and relationships
func ValidateOrder(order *Order) *OrderValidation {
	validation := &OrderValidation{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Validate the order itself
	if err := order.Validate(); err != nil {
		validation.IsValid = false
		validation.Errors = append(validation.Errors, fmt.Sprintf("Order validation failed: %s", err))
	}

	// Validate order items
	if len(order.Items) == 0 {
		validation.IsValid = false
		validation.Errors = append(validation.Errors, "Order must have at least one item")
	} else {
		for i, item := range order.Items {
			if err := item.Validate(); err != nil {
				validation.IsValid = false
				validation.Errors = append(validation.Errors, fmt.Sprintf("Item %d validation failed: %s", i+1, err))
			}
		}
	}

	// Check business rules
	if order.Subtotal.LessThanOrEqual(decimal.Zero) && len(order.Items) > 0 {
		validation.IsValid = false
		validation.Errors = append(validation.Errors, "Order subtotal must be greater than zero")
	}

	// Check if order has valid status transition
	if order.PreviousStatus != nil {
		if !IsValidStatusTransition(*order.PreviousStatus, order.Status) {
			validation.IsValid = false
			validation.Errors = append(validation.Errors, fmt.Sprintf("Invalid status transition from %s to %s", *order.PreviousStatus, order.Status))
		}
	}

	// Add warnings for potential issues
	if order.RequiredDate != nil && time.Now().UTC().After(*order.RequiredDate) {
		validation.Warnings = append(validation.Warnings, "Order required date has passed")
	}

	if order.TotalAmount.GreaterThan(decimal.NewFromInt(10000)) {
		validation.Warnings = append(validation.Warnings, "Order total exceeds $10,000 - additional approval may be required")
	}

	if order.IsDigitalOrder() && order.ShippingMethod != ShippingMethodDigital {
		validation.Warnings = append(validation.Warnings, "Digital order with non-digital shipping method")
	}

	return validation
}

// GenerateOrderNumber generates a unique order number
func GenerateOrderNumber() string {
	now := time.Now().UTC()
	year := now.Year()

	// In a real implementation, this would query the database to get the next sequence
	// For now, we'll use a timestamp-based approach
	sequence := now.Unix() % 1000000 // Last 6 digits of timestamp

	return fmt.Sprintf("%d-%06d", year, sequence)
}

// CalculateShippingWeight calculates total shipping weight for an order
func CalculateShippingWeight(order *Order) float64 {
	totalWeight := 0.0
	for _, item := range order.Items {
		totalWeight += item.Weight * float64(item.Quantity)
	}
	return totalWeight
}

// IsOrderComplete checks if an order is completely processed
func IsOrderComplete(order *Order) bool {
	return IsTerminalStatus(order.Status)
}

// CanOrderBeModified checks if an order can be modified
func CanOrderBeModified(order *Order) bool {
	// Orders in terminal statuses cannot be modified
	if IsTerminalStatus(order.Status) {
		return false
	}

	// Orders that are already shipped may have limited modification options
	if order.Status == OrderStatusShipped || order.Status == OrderStatusPartiallyShipped {
		return false
	}

	return true
}

// CustomerSummary represents a simplified customer view for analytics
type CustomerSummary struct {
	ID                uuid.UUID       `json:"id"`
	CustomerCode      string          `json:"customer_code"`
	CustomerName      string          `json:"customer_name"`
	Email             string          `json:"email"`
	CompanyName       *string         `json:"company_name,omitempty"`
	Type              string          `json:"type"`
	TotalOrders       int64           `json:"total_orders"`
	TotalRevenue      decimal.Decimal `json:"total_revenue"`
	LastOrderDate     *time.Time      `json:"last_order_date,omitempty"`
	AverageOrderValue decimal.Decimal `json:"average_order_value"`
	IsActive          bool            `json:"is_active"`
	CreatedAt         time.Time       `json:"created_at"`
}
