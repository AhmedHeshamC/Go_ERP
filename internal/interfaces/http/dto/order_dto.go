package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Order-related DTOs

// OrderResponse represents order information returned in responses
type OrderResponse struct {
	ID                 uuid.UUID           `json:"id"`
	OrderNumber        string              `json:"order_number"`
	CustomerID         uuid.UUID           `json:"customer_id"`
	CustomerName       string              `json:"customer_name"`
	Type               string              `json:"type"`
	Status             string              `json:"status"`
	Priority           string              `json:"priority"`
	PaymentStatus      string              `json:"payment_status"`
	FulfillmentStatus  string              `json:"fulfillment_status"`
	Currency           string              `json:"currency"`
	Subtotal           decimal.Decimal     `json:"subtotal"`
	TaxAmount          decimal.Decimal     `json:"tax_amount"`
	ShippingAmount     decimal.Decimal     `json:"shipping_amount"`
	DiscountAmount     decimal.Decimal     `json:"discount_amount"`
	TotalAmount        decimal.Decimal     `json:"total_amount"`
	PaidAmount         decimal.Decimal     `json:"paid_amount"`
	RefundedAmount     decimal.Decimal     `json:"refunded_amount"`
	Weight             decimal.Decimal     `json:"weight"`
	ShippingMethod     string              `json:"shipping_method"`
	TrackingNumber     string              `json:"tracking_number,omitempty"`
	ShippingAddress    *AddressResponse    `json:"shipping_address"`
	BillingAddress     *AddressResponse    `json:"billing_address"`
	Items              []OrderItemResponse `json:"items"`
	Notes              *string             `json:"notes,omitempty"`
	CustomerNotes      *string             `json:"customer_notes,omitempty"`
	InternalNotes      *string             `json:"internal_notes,omitempty"`
	RequiredDate       *time.Time          `json:"required_date,omitempty"`
	ShippedDate        *time.Time          `json:"shipped_date,omitempty"`
	DeliveredDate      *time.Time          `json:"delivered_date,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
	ApprovedAt         *time.Time          `json:"approved_at,omitempty"`
	ApprovedBy         *string             `json:"approved_by,omitempty"`
	CancelledAt        *time.Time          `json:"cancelled_at,omitempty"`
	CancelledBy        *string             `json:"cancelled_by,omitempty"`
	CancellationReason *string             `json:"cancellation_reason,omitempty"`
}

// OrderItemResponse represents order item information returned in responses
type OrderItemResponse struct {
	ID               uuid.UUID       `json:"id"`
	ProductID        uuid.UUID       `json:"product_id"`
	ProductSKU       string          `json:"product_sku"`
	ProductName      string          `json:"product_name"`
	VariantID        *uuid.UUID      `json:"variant_id,omitempty"`
	VariantName      *string         `json:"variant_name,omitempty"`
	Quantity         int32           `json:"quantity"`
	UnitPrice        decimal.Decimal `json:"unit_price"`
	TotalPrice       decimal.Decimal `json:"total_price"`
	TaxAmount        decimal.Decimal `json:"tax_amount"`
	DiscountAmount   decimal.Decimal `json:"discount_amount"`
	FinalPrice       decimal.Decimal `json:"final_price"`
	Weight           decimal.Decimal `json:"weight"`
	Status           string          `json:"status"`
	ShippedQuantity  int32           `json:"shipped_quantity"`
	ReturnedQuantity int32           `json:"returned_quantity"`
	Notes            *string         `json:"notes,omitempty"`
}

// AddressResponse represents address information returned in responses
type AddressResponse struct {
	ID         uuid.UUID `json:"id"`
	Type       string    `json:"type"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Company    *string   `json:"company,omitempty"`
	Address1   string    `json:"address1"`
	Address2   *string   `json:"address2,omitempty"`
	City       string    `json:"city"`
	State      string    `json:"state"`
	PostalCode string    `json:"postal_code"`
	Country    string    `json:"country"`
	Phone      *string   `json:"phone,omitempty"`
	Email      *string   `json:"email,omitempty"`
	IsDefault  bool      `json:"is_default"`
}

// OrderRequest represents a request to create/update an order
type OrderRequest struct {
	CustomerID        uuid.UUID          `json:"customer_id" binding:"required,uuid"`
	Type              string             `json:"type" binding:"required,oneof=SALES PURCHASE RETURN EXCHANGE TRANSFER ADJUSTMENT"`
	Priority          string             `json:"priority" binding:"omitempty,oneof=LOW NORMAL HIGH URGENT CRITICAL"`
	ShippingMethod    string             `json:"shipping_method" binding:"required,oneof=STANDARD EXPRESS OVERNIGHT PICKUP DIGITAL"`
	ShippingAddressID uuid.UUID          `json:"shipping_address_id" binding:"required,uuid"`
	BillingAddressID  uuid.UUID          `json:"billing_address_id" binding:"required,uuid"`
	Currency          string             `json:"currency" binding:"required,len=3"`
	RequiredDate      *time.Time         `json:"required_date,omitempty"`
	Notes             *string            `json:"notes,omitempty"`
	CustomerNotes     *string            `json:"customer_notes,omitempty"`
	Items             []OrderItemRequest `json:"items" binding:"required,min=1"`
	DiscountCode      *string            `json:"discount_code,omitempty"`
	PaymentMethod     *string            `json:"payment_method,omitempty"`
}

// OrderItemRequest represents a request to create/update an order item
type OrderItemRequest struct {
	ProductID uuid.UUID       `json:"product_id" binding:"required,uuid"`
	VariantID *uuid.UUID      `json:"variant_id,omitempty"`
	Quantity  int32           `json:"quantity" binding:"required,min=1"`
	UnitPrice decimal.Decimal `json:"unit_price" binding:"required,gt=0"`
	Notes     *string         `json:"notes,omitempty"`
}

// UpdateOrderRequest represents a request to update an order
type UpdateOrderRequest struct {
	Type              *string    `json:"type,omitempty"`
	Priority          *string    `json:"priority,omitempty"`
	ShippingMethod    *string    `json:"shipping_method,omitempty"`
	ShippingAddressID *uuid.UUID `json:"shipping_address_id,omitempty"`
	BillingAddressID  *uuid.UUID `json:"billing_address_id,omitempty"`
	RequiredDate      *time.Time `json:"required_date,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	CustomerNotes     *string    `json:"customer_notes,omitempty"`
	InternalNotes     *string    `json:"internal_notes,omitempty"`
	DiscountCode      *string    `json:"discount_code,omitempty"`
}

// UpdateOrderStatusRequest represents a request to update order status
type UpdateOrderStatusRequest struct {
	Status         string  `json:"status" binding:"required"`
	Notes          *string `json:"notes,omitempty"`
	NotifyCustomer bool    `json:"notify_customer"`
}

// CancelOrderRequest represents a request to cancel an order
type CancelOrderRequest struct {
	Reason         string `json:"reason" binding:"required"`
	NotifyCustomer bool   `json:"notify_customer"`
	RefundPayment  bool   `json:"refund_payment"`
	RestockItems   bool   `json:"restock_items"`
}

// ProcessOrderRequest represents a request to process an order
type ProcessOrderRequest struct {
	NotifyCustomer bool `json:"notify_customer"`
}

// ShipOrderRequest represents a request to ship an order
type ShipOrderRequest struct {
	TrackingNumber string     `json:"tracking_number" binding:"required"`
	Carrier        string     `json:"carrier" binding:"required"`
	ShippingDate   *time.Time `json:"shipping_date,omitempty"`
	NotifyCustomer bool       `json:"notify_customer"`
	TrackingURL    *string    `json:"tracking_url,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
}

// DeliverOrderRequest represents a request to mark an order as delivered
type DeliverOrderRequest struct {
	DeliveryDate   *time.Time `json:"delivery_date,omitempty"`
	Signature      *string    `json:"signature,omitempty"`
	PhotoProofURL  *string    `json:"photo_proof_url,omitempty"`
	NotifyCustomer bool       `json:"notify_customer"`
	Notes          *string    `json:"notes,omitempty"`
}

// PartialShipOrderRequest represents a request to partially ship an order
type PartialShipOrderRequest struct {
	Items          []PartialShipItemRequest `json:"items" binding:"required,min=1"`
	TrackingNumber string                   `json:"tracking_number" binding:"required"`
	Carrier        string                   `json:"carrier" binding:"required"`
	ShippingDate   *time.Time               `json:"shipping_date,omitempty"`
	NotifyCustomer bool                     `json:"notify_customer"`
	TrackingURL    *string                  `json:"tracking_url,omitempty"`
	Notes          *string                  `json:"notes,omitempty"`
}

// PartialShipItemRequest represents an item to be partially shipped
type PartialShipItemRequest struct {
	OrderItemID uuid.UUID `json:"order_item_id" binding:"required,uuid"`
	Quantity    int32     `json:"quantity" binding:"required,min=1"`
}

// ReturnItemsRequest represents a request to return order items
type ReturnItemsRequest struct {
	Items          []ReturnItemRequest `json:"items" binding:"required,min=1"`
	Reason         string              `json:"reason" binding:"required"`
	RestockItems   bool                `json:"restock_items"`
	RefundAmount   bool                `json:"refund_amount"`
	NotifyCustomer bool                `json:"notify_customer"`
	Notes          *string             `json:"notes,omitempty"`
}

// ReturnItemRequest represents an item to be returned
type ReturnItemRequest struct {
	OrderItemID uuid.UUID `json:"order_item_id" binding:"required,uuid"`
	Quantity    int32     `json:"quantity" binding:"required,min=1"`
	Reason      *string   `json:"reason,omitempty"`
}

// ProcessPaymentRequest represents a request to process payment
type ProcessPaymentRequest struct {
	PaymentMethod string          `json:"payment_method" binding:"required"`
	Amount        decimal.Decimal `json:"amount" binding:"required,gt=0"`
	TransactionID *string         `json:"transaction_id,omitempty"`
	PaymentDate   *time.Time      `json:"payment_date,omitempty"`
	Notes         *string         `json:"notes,omitempty"`
}

// RefundOrderRequest represents a request to refund an order
type RefundOrderRequest struct {
	Amount         decimal.Decimal `json:"amount" binding:"required,gt=0"`
	Reason         string          `json:"reason" binding:"required"`
	RestockItems   bool            `json:"restock_items"`
	NotifyCustomer bool            `json:"notify_customer"`
	Notes          *string         `json:"notes,omitempty"`
}

// PartialRefundOrderRequest represents a request to partially refund an order
type PartialRefundOrderRequest struct {
	Items          []PartialRefundItemRequest `json:"items" binding:"required,min=1"`
	Reason         string                     `json:"reason" binding:"required"`
	RestockItems   bool                       `json:"restock_items"`
	NotifyCustomer bool                       `json:"notify_customer"`
	Notes          *string                    `json:"notes,omitempty"`
}

// PartialRefundItemRequest represents an item to be partially refunded
type PartialRefundItemRequest struct {
	OrderItemID uuid.UUID       `json:"order_item_id" binding:"required,uuid"`
	Quantity    int32           `json:"quantity" binding:"required,min=1"`
	Amount      decimal.Decimal `json:"amount" binding:"required,gt=0"`
	Reason      *string         `json:"reason,omitempty"`
}

// AddOrderItemRequest represents a request to add an item to an order
type AddOrderItemRequest struct {
	ProductID uuid.UUID       `json:"product_id" binding:"required,uuid"`
	VariantID *uuid.UUID      `json:"variant_id,omitempty"`
	Quantity  int32           `json:"quantity" binding:"required,min=1"`
	UnitPrice decimal.Decimal `json:"unit_price" binding:"required,gt=0"`
	Notes     *string         `json:"notes,omitempty"`
}

// UpdateOrderItemRequest represents a request to update an order item
type UpdateOrderItemRequest struct {
	Quantity  *int32           `json:"quantity,omitempty"`
	UnitPrice *decimal.Decimal `json:"unit_price,omitempty"`
	Notes     *string          `json:"notes,omitempty"`
}

// ListOrdersRequest represents a request to list orders
type ListOrdersRequest struct {
	CustomerID        *uuid.UUID `json:"customer_id,omitempty"`
	Status            *string    `json:"status,omitempty"`
	Type              *string    `json:"type,omitempty"`
	Priority          *string    `json:"priority,omitempty"`
	PaymentStatus     *string    `json:"payment_status,omitempty"`
	FulfillmentStatus *string    `json:"fulfillment_status,omitempty"`
	Currency          *string    `json:"currency,omitempty"`
	CreatedAfter      *time.Time `json:"created_after,omitempty"`
	CreatedBefore     *time.Time `json:"created_before,omitempty"`
	RequiredAfter     *time.Time `json:"required_after,omitempty"`
	RequiredBefore    *time.Time `json:"required_before,omitempty"`
	Search            *string    `json:"search,omitempty"`
	SortBy            *string    `json:"sort_by,omitempty"`
	SortOrder         *string    `json:"sort_order,omitempty"`
	Page              int        `json:"page" binding:"omitempty,min=1"`
	Limit             int        `json:"limit" binding:"omitempty,min=1,max=100"`
}

// SearchOrdersRequest represents a request to search orders
type SearchOrdersRequest struct {
	Query         string     `json:"query" binding:"required"`
	CustomerID    *uuid.UUID `json:"customer_id,omitempty"`
	Status        *string    `json:"status,omitempty"`
	Type          *string    `json:"type,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	SearchFields  []string   `json:"search_fields,omitempty"`
	SortBy        *string    `json:"sort_by,omitempty"`
	SortOrder     *string    `json:"sort_order,omitempty"`
	Page          int        `json:"page" binding:"omitempty,min=1"`
	Limit         int        `json:"limit" binding:"omitempty,min=1,max=100"`
}

// GetCustomerOrdersRequest represents a request to get customer orders
type GetCustomerOrdersRequest struct {
	CustomerID    uuid.UUID  `json:"customer_id" binding:"required,uuid"`
	Status        *string    `json:"status,omitempty"`
	Type          *string    `json:"type,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	SortBy        *string    `json:"sort_by,omitempty"`
	SortOrder     *string    `json:"sort_order,omitempty"`
	Page          int        `json:"page" binding:"omitempty,min=1"`
	Limit         int        `json:"limit" binding:"omitempty,min=1,max=100"`
}

// OrderValidationRequest represents a request to validate an order
type OrderValidationRequest struct {
	ValidateInventory bool `json:"validate_inventory"`
	ValidatePricing   bool `json:"validate_pricing"`
	ValidateCustomer  bool `json:"validate_customer"`
	ValidateAddresses bool `json:"validate_addresses"`
}

// OrderValidationResponse represents order validation results
type OrderValidationResponse struct {
	IsValid        bool                       `json:"is_valid"`
	Errors         []string                   `json:"errors,omitempty"`
	Warnings       []string                   `json:"warnings,omitempty"`
	InventoryCheck *InventoryValidationResult `json:"inventory_check,omitempty"`
	PricingCheck   *PricingValidationResult   `json:"pricing_check,omitempty"`
	CustomerCheck  *CustomerValidationResult  `json:"customer_check,omitempty"`
	AddressCheck   *AddressValidationResult   `json:"address_check,omitempty"`
}

// InventoryValidationResult represents inventory validation results
type InventoryValidationResult struct {
	IsValid   bool                      `json:"is_valid"`
	Available bool                      `json:"available"`
	Items     []InventoryItemValidation `json:"items"`
}

// InventoryItemValidation represents inventory validation for a single item
type InventoryItemValidation struct {
	ProductID   uuid.UUID  `json:"product_id"`
	VariantID   *uuid.UUID `json:"variant_id,omitempty"`
	Requested   int32      `json:"requested"`
	Available   int32      `json:"available"`
	IsAvailable bool       `json:"is_available"`
}

// PricingValidationResult represents pricing validation results
type PricingValidationResult struct {
	IsValid        bool                    `json:"is_valid"`
	Subtotal       decimal.Decimal         `json:"subtotal"`
	TaxAmount      decimal.Decimal         `json:"tax_amount"`
	ShippingAmount decimal.Decimal         `json:"shipping_amount"`
	DiscountAmount decimal.Decimal         `json:"discount_amount"`
	TotalAmount    decimal.Decimal         `json:"total_amount"`
	Items          []PricingItemValidation `json:"items"`
}

// PricingItemValidation represents pricing validation for a single item
type PricingItemValidation struct {
	ProductID      uuid.UUID       `json:"product_id"`
	UnitPrice      decimal.Decimal `json:"unit_price"`
	TotalPrice     decimal.Decimal `json:"total_price"`
	TaxAmount      decimal.Decimal `json:"tax_amount"`
	DiscountAmount decimal.Decimal `json:"discount_amount"`
	FinalPrice     decimal.Decimal `json:"final_price"`
}

// CustomerValidationResult represents customer validation results
type CustomerValidationResult struct {
	IsValid     bool            `json:"is_valid"`
	CustomerID  uuid.UUID       `json:"customer_id"`
	Name        string          `json:"name"`
	Email       string          `json:"email"`
	Phone       string          `json:"phone"`
	IsActive    bool            `json:"is_active"`
	CreditLimit decimal.Decimal `json:"credit_limit"`
	Balance     decimal.Decimal `json:"balance"`
}

// AddressValidationResult represents address validation results
type AddressValidationResult struct {
	IsValid         bool             `json:"is_valid"`
	ShippingAddress *AddressResponse `json:"shipping_address,omitempty"`
	BillingAddress  *AddressResponse `json:"billing_address,omitempty"`
}

// ListOrdersResponse represents a paginated orders list response
type ListOrdersResponse struct {
	Orders     []*OrderResponse `json:"orders"`
	Pagination *Pagination      `json:"pagination"`
}

// SearchOrdersResponse represents a search orders response
type SearchOrdersResponse struct {
	Orders     []*OrderResponse `json:"orders"`
	Pagination *Pagination      `json:"pagination"`
	TotalCount int              `json:"total_count"`
}

// GetCustomerOrdersResponse represents customer orders response
type GetCustomerOrdersResponse struct {
	Customer   *CustomerInfo    `json:"customer"`
	Orders     []*OrderResponse `json:"orders"`
	Pagination *Pagination      `json:"pagination"`
}

// CustomerInfo represents customer information
type CustomerInfo struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Email       string          `json:"email"`
	Phone       string          `json:"phone"`
	CreditLimit decimal.Decimal `json:"credit_limit"`
	Balance     decimal.Decimal `json:"balance"`
	IsActive    bool            `json:"is_active"`
}

// OrderStatsResponse represents order statistics response
type OrderStatsResponse struct {
	TotalOrders       int64            `json:"total_orders"`
	TotalRevenue      decimal.Decimal  `json:"total_revenue"`
	OrdersByStatus    map[string]int64 `json:"orders_by_status"`
	OrdersByType      map[string]int64 `json:"orders_by_type"`
	TopProducts       []ProductStats   `json:"top_products"`
	TopCustomers      []CustomerStats  `json:"top_customers"`
	AverageOrderValue decimal.Decimal  `json:"average_order_value"`
}

// ProductStats represents product statistics
type ProductStats struct {
	ProductID    uuid.UUID       `json:"product_id"`
	ProductName  string          `json:"product_name"`
	SKU          string          `json:"sku"`
	TotalSales   int64           `json:"total_sales"`
	TotalRevenue decimal.Decimal `json:"total_revenue"`
}

// CustomerStats represents customer statistics
type CustomerStats struct {
	CustomerID   uuid.UUID       `json:"customer_id"`
	CustomerName string          `json:"customer_name"`
	Email        string          `json:"email"`
	TotalOrders  int64           `json:"total_orders"`
	TotalRevenue decimal.Decimal `json:"total_revenue"`
}

// RevenueByPeriodResponse represents revenue by period response
type RevenueByPeriodResponse struct {
	Period      string             `json:"period"`
	RevenueData []RevenueDataPoint `json:"revenue_data"`
}

// RevenueDataPoint represents a single revenue data point
type RevenueDataPoint struct {
	Period            time.Time       `json:"period"`
	Revenue           decimal.Decimal `json:"revenue"`
	Orders            int64           `json:"orders"`
	AverageOrderValue decimal.Decimal `json:"average_order_value"`
}

// OrderAnalyticsResponse represents comprehensive order analytics
type OrderAnalyticsResponse struct {
	Period             string             `json:"period"`
	Summary            OrderStatsResponse `json:"summary"`
	RevenueByPeriod    []RevenueDataPoint `json:"revenue_by_period"`
	OrdersByStatus     map[string]int64   `json:"orders_by_status"`
	OrdersByType       map[string]int64   `json:"orders_by_type"`
	TopProducts        []ProductStats     `json:"top_products"`
	TopCustomers       []CustomerStats    `json:"top_customers"`
	FulfillmentMetrics FulfillmentMetrics `json:"fulfillment_metrics"`
	PaymentMetrics     PaymentMetrics     `json:"payment_metrics"`
}

// FulfillmentMetrics represents fulfillment statistics
type FulfillmentMetrics struct {
	AverageFulfillmentTime float64 `json:"average_fulfillment_time"`
	OrdersShipped          int64   `json:"orders_shipped"`
	OrdersDelivered        int64   `json:"orders_delivered"`
	PartialShipmentRate    float64 `json:"partial_shipment_rate"`
	OnTimeDeliveryRate     float64 `json:"on_time_delivery_rate"`
}

// PaymentMetrics represents payment statistics
type PaymentMetrics struct {
	PaidOrders          int64            `json:"paid_orders"`
	UnpaidOrders        int64            `json:"unpaid_orders"`
	TotalPaidAmount     decimal.Decimal  `json:"total_paid_amount"`
	AveragePaymentTime  float64          `json:"average_payment_time"`
	PaymentMethods      map[string]int64 `json:"payment_methods"`
	RefundRate          float64          `json:"refund_rate"`
	TotalRefundedAmount decimal.Decimal  `json:"total_refunded_amount"`
}

// BulkUpdateStatusRequest represents a request to bulk update order status
type BulkUpdateStatusRequest struct {
	OrderIDs       []string `json:"order_ids" binding:"required,min=1"`
	Status         string   `json:"status" binding:"required"`
	NotifyCustomer bool     `json:"notify_customer"`
	Notes          *string  `json:"notes,omitempty"`
}

// BulkUpdateStatusResponse represents bulk update status response
type BulkUpdateStatusResponse struct {
	UpdatedCount  int               `json:"updated_count"`
	FailedCount   int               `json:"failed_count"`
	UpdatedOrders []string          `json:"updated_orders"`
	FailedOrders  []FailedOperation `json:"failed_orders"`
}

// BulkCancelOrdersRequest represents a request to bulk cancel orders
type BulkCancelOrdersRequest struct {
	OrderIDs       []string `json:"order_ids" binding:"required,min=1"`
	Reason         string   `json:"reason" binding:"required"`
	NotifyCustomer bool     `json:"notify_customer"`
	RefundPayment  bool     `json:"refund_payment"`
	RestockItems   bool     `json:"restock_items"`
}

// BulkCancelOrdersResponse represents bulk cancel orders response
type BulkCancelOrdersResponse struct {
	CancelledCount  int               `json:"cancelled_count"`
	FailedCount     int               `json:"failed_count"`
	CancelledOrders []string          `json:"cancelled_orders"`
	FailedOrders    []FailedOperation `json:"failed_orders"`
}

// FailedOperation represents a failed bulk operation
type FailedOperation struct {
	OrderID string `json:"order_id"`
	Error   string `json:"error"`
}

// CloneOrderRequest represents a request to clone an order
type CloneOrderRequest struct {
	IncludeItems     bool     `json:"include_items"`
	IncludeAddresses bool     `json:"include_addresses"`
	ItemIDs          []string `json:"item_ids,omitempty"`
	NewNotes         *string  `json:"new_notes,omitempty"`
}

// OrderCalculationResponse represents order calculation results
type OrderCalculationResponse struct {
	Subtotal         decimal.Decimal        `json:"subtotal"`
	TaxAmount        decimal.Decimal        `json:"tax_amount"`
	ShippingAmount   decimal.Decimal        `json:"shipping_amount"`
	DiscountAmount   decimal.Decimal        `json:"discount_amount"`
	TotalAmount      decimal.Decimal        `json:"total_amount"`
	ItemCalculations []OrderItemCalculation `json:"item_calculations"`
	Discounts        []OrderDiscount        `json:"discounts"`
	Taxes            []OrderTax             `json:"taxes"`
}

// OrderItemCalculation represents calculation for a single order item
type OrderItemCalculation struct {
	OrderItemID    uuid.UUID       `json:"order_item_id"`
	ProductID      uuid.UUID       `json:"product_id"`
	Quantity       int32           `json:"quantity"`
	UnitPrice      decimal.Decimal `json:"unit_price"`
	TotalPrice     decimal.Decimal `json:"total_price"`
	TaxAmount      decimal.Decimal `json:"tax_amount"`
	DiscountAmount decimal.Decimal `json:"discount_amount"`
	FinalPrice     decimal.Decimal `json:"final_price"`
}

// OrderDiscount represents an order discount
type OrderDiscount struct {
	ID         uuid.UUID        `json:"id"`
	Type       string           `json:"type"`
	Name       string           `json:"name"`
	Code       *string          `json:"code,omitempty"`
	Amount     decimal.Decimal  `json:"amount"`
	Percentage *decimal.Decimal `json:"percentage,omitempty"`
}

// OrderTax represents an order tax
type OrderTax struct {
	ID     uuid.UUID       `json:"id"`
	Name   string          `json:"name"`
	Rate   decimal.Decimal `json:"rate"`
	Amount decimal.Decimal `json:"amount"`
	Type   string          `json:"type"`
}
