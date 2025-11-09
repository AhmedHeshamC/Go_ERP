package order

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"

	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/domain/orders/repositories"
)

// Service defines the business logic interface for order management
type Service interface {
	// Order management
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*entities.Order, error)
	GetOrder(ctx context.Context, id string) (*entities.Order, error)
	GetOrderByNumber(ctx context.Context, orderNumber string) (*entities.Order, error)
	UpdateOrder(ctx context.Context, id string, req *UpdateOrderRequest) (*entities.Order, error)
	DeleteOrder(ctx context.Context, id string) error
	ListOrders(ctx context.Context, req *ListOrdersRequest) (*ListOrdersResponse, error)
	SearchOrders(ctx context.Context, req *SearchOrdersRequest) (*SearchOrdersResponse, error)

	// Order status management
	UpdateOrderStatus(ctx context.Context, id string, req *UpdateOrderStatusRequest) (*entities.Order, error)
	CancelOrder(ctx context.Context, id string, req *CancelOrderRequest) (*entities.Order, error)
	ApproveOrder(ctx context.Context, id string, approvedBy string) (*entities.Order, error)
	HoldOrder(ctx context.Context, id string, reason string) (*entities.Order, error)
	UnholdOrder(ctx context.Context, id string) (*entities.Order, error)

	// Order fulfillment
	ProcessOrder(ctx context.Context, id string) (*entities.Order, error)
	ShipOrder(ctx context.Context, id string, req *ShipOrderRequest) (*entities.Order, error)
	DeliverOrder(ctx context.Context, id string, req *DeliverOrderRequest) (*entities.Order, error)
	PartialShipOrder(ctx context.Context, id string, req *PartialShipOrderRequest) (*entities.Order, error)
	ReturnOrderItems(ctx context.Context, id string, req *ReturnItemsRequest) (*entities.Order, error)

	// Payment processing
	ProcessPayment(ctx context.Context, id string, req *ProcessPaymentRequest) (*entities.Order, error)
	RefundOrder(ctx context.Context, id string, req *RefundOrderRequest) (*entities.Order, error)
	PartialRefundOrder(ctx context.Context, id string, req *PartialRefundOrderRequest) (*entities.Order, error)

	// Order item management
	AddOrderItem(ctx context.Context, orderID string, req *AddOrderItemRequest) (*entities.Order, error)
	UpdateOrderItem(ctx context.Context, orderID, itemID string, req *UpdateOrderItemRequest) (*entities.Order, error)
	RemoveOrderItem(ctx context.Context, orderID, itemID string) (*entities.Order, error)

	// Order validation and calculation
	ValidateOrder(ctx context.Context, id string) (*entities.OrderValidation, error)
	CalculateOrderTotals(ctx context.Context, id string) (*entities.OrderCalculation, error)
	RecalculateOrder(ctx context.Context, id string) (*entities.Order, error)

	// Customer order management
	GetCustomerOrders(ctx context.Context, customerID string, req *GetCustomerOrdersRequest) (*GetCustomerOrdersResponse, error)
	GetCustomerOrderHistory(ctx context.Context, customerID string, limit int) ([]*entities.Order, error)

	// Order analytics and reporting
	GetOrderStats(ctx context.Context, req *GetOrderStatsRequest) (*repositories.OrderStats, error)
	GetRevenueByPeriod(ctx context.Context, req *GetRevenueByPeriodRequest) ([]*repositories.RevenueByPeriod, error)
	GetTopCustomers(ctx context.Context, req *GetTopCustomersRequest) ([]*repositories.CustomerOrderStats, error)
	GetSalesByProduct(ctx context.Context, req *GetSalesByProductRequest) ([]*repositories.ProductSalesStats, error)
	GetOrderAnalytics(ctx context.Context, req *GetOrderAnalyticsRequest) (*OrderAnalyticsResponse, error)

	// Inventory integration
	CheckInventoryAvailability(ctx context.Context, req *CheckInventoryRequest) (*CheckInventoryResponse, error)
	ReserveInventory(ctx context.Context, orderID string) error
	ReleaseInventoryReservation(ctx context.Context, orderID string) error
	ConsumeInventory(ctx context.Context, orderID string) error

	// Bulk operations
	BulkUpdateStatus(ctx context.Context, req *BulkUpdateStatusRequest) (*BulkUpdateStatusResponse, error)
	BulkCancelOrders(ctx context.Context, req *BulkCancelOrdersRequest) (*BulkCancelOrdersResponse, error)

	// Order management utilities
	GenerateOrderNumber(ctx context.Context) (string, error)
	CloneOrder(ctx context.Context, id string, req *CloneOrderRequest) (*entities.Order, error)
	ArchiveOrder(ctx context.Context, id string) error
	RestoreOrder(ctx context.Context, id string) error
}

// Request/Response DTOs

// CreateOrderRequest represents a request to create a new order
type CreateOrderRequest struct {
	CustomerID        string                    `json:"customer_id" validate:"required,uuid"`
	Type              entities.OrderType       `json:"type" validate:"required"`
	Priority          entities.OrderPriority   `json:"priority"`
	ShippingMethod    entities.ShippingMethod  `json:"shipping_method" validate:"required"`
	ShippingAddressID string                    `json:"shipping_address_id" validate:"required,uuid"`
	BillingAddressID  string                    `json:"billing_address_id" validate:"required,uuid"`
	Currency          string                    `json:"currency" validate:"required,len=3"`
	RequiredDate      *time.Time                `json:"required_date,omitempty"`
	Notes             *string                   `json:"notes,omitempty"`
	CustomerNotes     *string                   `json:"customer_notes,omitempty"`
	Items             []CreateOrderItemRequest  `json:"items" validate:"required,min=1"`
	DiscountCode      *string                   `json:"discount_code,omitempty"`
	PaymentMethod     *string                   `json:"payment_method,omitempty"`
}

// CreateOrderItemRequest represents a request to add an item to an order
type CreateOrderItemRequest struct {
	ProductID      string          `json:"product_id" validate:"required,uuid"`
	Quantity       int             `json:"quantity" validate:"required,min=1"`
	UnitPrice      decimal.Decimal `json:"unit_price,omitempty"`
	DiscountAmount decimal.Decimal `json:"discount_amount,omitempty"`
	TaxRate        decimal.Decimal `json:"tax_rate,omitempty"`
	Notes          *string         `json:"notes,omitempty"`
}

// UpdateOrderRequest represents a request to update an order
type UpdateOrderRequest struct {
	ShippingMethod    *entities.ShippingMethod `json:"shipping_method,omitempty"`
	RequiredDate      *time.Time               `json:"required_date,omitempty"`
	Notes             *string                  `json:"notes,omitempty"`
	InternalNotes     *string                  `json:"internal_notes,omitempty"`
	CustomerNotes     *string                  `json:"customer_notes,omitempty"`
	DiscountAmount    *decimal.Decimal         `json:"discount_amount,omitempty"`
	TaxAmount         *decimal.Decimal         `json:"tax_amount,omitempty"`
	ShippingAmount    *decimal.Decimal         `json:"shipping_amount,omitempty"`
	Priority          *entities.OrderPriority  `json:"priority,omitempty"`
	ShippingAddressID *string                  `json:"shipping_address_id,omitempty"`
	BillingAddressID  *string                  `json:"billing_address_id,omitempty"`
}

// ListOrdersRequest represents a request to list orders
type ListOrdersRequest struct {
	Search            string                     `json:"search,omitempty"`
	Status            []entities.OrderStatus    `json:"status,omitempty"`
	PaymentStatus     []entities.PaymentStatus  `json:"payment_status,omitempty"`
	Priority          []entities.OrderPriority  `json:"priority,omitempty"`
	Type              []entities.OrderType      `json:"type,omitempty"`
	ShippingMethod    []entities.ShippingMethod `json:"shipping_method,omitempty"`
	CustomerID        *string                    `json:"customer_id,omitempty"`
	CompanyID         *string                    `json:"company_id,omitempty"`
	CustomerType      string                     `json:"customer_type,omitempty"`
	StartDate         *time.Time                 `json:"start_date,omitempty"`
	EndDate           *time.Time                 `json:"end_date,omitempty"`
	MinTotalAmount    *decimal.Decimal           `json:"min_total_amount,omitempty"`
	MaxTotalAmount    *decimal.Decimal           `json:"max_total_amount,omitempty"`
	Currency          string                     `json:"currency,omitempty"`
	CreatedBy         *string                    `json:"created_by,omitempty"`
	Page              int                        `json:"page,omitempty" validate:"min=1"`
	Limit             int                        `json:"limit,omitempty" validate:"min=1,max=100"`
	SortBy            string                     `json:"sort_by,omitempty"`
	SortOrder         string                     `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

// ListOrdersResponse represents the response for listing orders
type ListOrdersResponse struct {
	Orders     []*entities.Order `json:"orders"`
	Pagination *Pagination       `json:"pagination"`
}

// SearchOrdersRequest represents a request to search orders
type SearchOrdersRequest struct {
	Query string `json:"query" validate:"required,min=1"`
	Limit int    `json:"limit,omitempty" validate:"min=1,max=50"`
}

// SearchOrdersResponse represents the response for searching orders
type SearchOrdersResponse struct {
	Orders []*entities.Order `json:"orders"`
	Total  int               `json:"total"`
}

// UpdateOrderStatusRequest represents a request to update order status
type UpdateOrderStatusRequest struct {
	Status      entities.OrderStatus `json:"status" validate:"required"`
	Reason      string               `json:"reason,omitempty"`
	Notify      bool                 `json:"notify"`
	UpdatedBy   string               `json:"updated_by" validate:"required,uuid"`
}

// CancelOrderRequest represents a request to cancel an order
type CancelOrderRequest struct {
	Reason    string `json:"reason" validate:"required"`
	Refund    bool   `json:"refund"`
	Notify    bool   `json:"notify"`
	CancelledBy string `json:"cancelled_by" validate:"required,uuid"`
}

// ShipOrderRequest represents a request to ship an order
type ShipOrderRequest struct {
	TrackingNumber string    `json:"tracking_number,omitempty"`
	Carrier        string    `json:"carrier,omitempty"`
	ShippingDate   *time.Time `json:"shipping_date,omitempty"`
	Notify         bool       `json:"notify"`
	ShippedBy      string     `json:"shipped_by" validate:"required,uuid"`
	Items          []ShipItemRequest `json:"items,omitempty"`
}

// ShipItemRequest represents shipping information for an order item
type ShipItemRequest struct {
	ItemID       string `json:"item_id" validate:"required,uuid"`
	Quantity     int    `json:"quantity" validate:"required,min=1"`
	TrackingNumber string `json:"tracking_number,omitempty"`
}

// DeliverOrderRequest represents a request to mark an order as delivered
type DeliverOrderRequest struct {
	DeliveryDate *time.Time `json:"delivery_date,omitempty"`
	Proof        *string    `json:"proof,omitempty"` // Delivery proof URL or reference
	Notes        *string    `json:"notes,omitempty"`
	Notify       bool       `json:"notify"`
	DeliveredBy  string     `json:"delivered_by" validate:"required,uuid"`
}

// PartialShipOrderRequest represents a request to partially ship an order
type PartialShipOrderRequest struct {
	Items        []ShipItemRequest `json:"items" validate:"required,min=1"`
	TrackingNumber string          `json:"tracking_number,omitempty"`
	Carrier      string            `json:"carrier,omitempty"`
	ShippingDate *time.Time        `json:"shipping_date,omitempty"`
	Notify       bool              `json:"notify"`
	ShippedBy    string            `json:"shipped_by" validate:"required,uuid"`
}

// ReturnItemsRequest represents a request to return order items
type ReturnItemsRequest struct {
	Items      []ReturnItemRequest `json:"items" validate:"required,min=1"`
	Reason     string              `json:"reason" validate:"required"`
	Refund     bool                `json:"refund"`
	Notes      *string             `json:"notes,omitempty"`
	ReturnedBy string              `json:"returned_by" validate:"required,uuid"`
}

// ReturnItemRequest represents a return request for an order item
type ReturnItemRequest struct {
	ItemID       string          `json:"item_id" validate:"required,uuid"`
	Quantity     int             `json:"quantity" validate:"required,min=1"`
	RefundAmount decimal.Decimal `json:"refund_amount,omitempty"`
	Reason       string          `json:"reason,omitempty"`
}

// ProcessPaymentRequest represents a request to process a payment
type ProcessPaymentRequest struct {
	Amount        decimal.Decimal `json:"amount" validate:"required,gt=0"`
	PaymentMethod string          `json:"payment_method" validate:"required"`
	TransactionID string          `json:"transaction_id,omitempty"`
	Notes         *string         `json:"notes,omitempty"`
	PaymentBy     string          `json:"payment_by" validate:"required,uuid"`
}

// RefundOrderRequest represents a request to refund an order
type RefundOrderRequest struct {
	Amount       decimal.Decimal `json:"amount" validate:"required,gt=0"`
	Reason       string          `json:"reason" validate:"required"`
	RefundMethod string          `json:"refund_method,omitempty"`
	TransactionID string         `json:"transaction_id,omitempty"`
	Notes        *string         `json:"notes,omitempty"`
	RefundedBy   string          `json:"refunded_by" validate:"required,uuid"`
}

// PartialRefundOrderRequest represents a request for partial refund
type PartialRefundOrderRequest struct {
	Items        []RefundItemRequest `json:"items" validate:"required,min=1"`
	Reason       string              `json:"reason" validate:"required"`
	RefundMethod string              `json:"refund_method,omitempty"`
	TransactionID string             `json:"transaction_id,omitempty"`
	Notes        *string             `json:"notes,omitempty"`
	RefundedBy   string              `json:"refunded_by" validate:"required,uuid"`
}

// RefundItemRequest represents a refund request for an order item
type RefundItemRequest struct {
	ItemID       string          `json:"item_id" validate:"required,uuid"`
	Quantity     int             `json:"quantity" validate:"required,min=1"`
	RefundAmount decimal.Decimal `json:"refund_amount,omitempty"`
	Reason       string          `json:"reason,omitempty"`
}

// AddOrderItemRequest represents a request to add an item to an existing order
type AddOrderItemRequest struct {
	ProductID      string          `json:"product_id" validate:"required,uuid"`
	Quantity       int             `json:"quantity" validate:"required,min=1"`
	UnitPrice      decimal.Decimal `json:"unit_price,omitempty"`
	DiscountAmount decimal.Decimal `json:"discount_amount,omitempty"`
	TaxRate        decimal.Decimal `json:"tax_rate,omitempty"`
	Notes          *string         `json:"notes,omitempty"`
}

// UpdateOrderItemRequest represents a request to update an order item
type UpdateOrderItemRequest struct {
	Quantity       *int            `json:"quantity,omitempty" validate:"omitempty,min=1"`
	UnitPrice      *decimal.Decimal `json:"unit_price,omitempty" validate:"omitempty,gt=0"`
	DiscountAmount *decimal.Decimal `json:"discount_amount,omitempty"`
	TaxRate        *decimal.Decimal `json:"tax_rate,omitempty"`
	Notes          *string         `json:"notes,omitempty"`
}

// GetCustomerOrdersRequest represents a request to get customer orders
type GetCustomerOrdersRequest struct {
	Status         []entities.OrderStatus   `json:"status,omitempty"`
	StartDate      *time.Time              `json:"start_date,omitempty"`
	EndDate        *time.Time              `json:"end_date,omitempty"`
	Page           int                     `json:"page,omitempty" validate:"min=1"`
	Limit          int                     `json:"limit,omitempty" validate:"min=1,max=100"`
	SortBy         string                  `json:"sort_by,omitempty"`
	SortOrder      string                  `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

// GetCustomerOrdersResponse represents the response for customer orders
type GetCustomerOrdersResponse struct {
	Orders     []*entities.Order `json:"orders"`
	Pagination *Pagination       `json:"pagination"`
	Summary    *CustomerSummary  `json:"summary,omitempty"`
}

// CustomerSummary represents a customer's order summary
type CustomerSummary struct {
	TotalOrders        int             `json:"total_orders"`
	TotalAmount        decimal.Decimal `json:"total_amount"`
	AverageOrderValue  decimal.Decimal `json:"average_order_value"`
	FirstOrderDate     *time.Time      `json:"first_order_date,omitempty"`
	LastOrderDate      *time.Time      `json:"last_order_date,omitempty"`
	StatusBreakdown    map[string]int  `json:"status_breakdown"`
}

// GetOrderStatsRequest represents a request to get order statistics
type GetOrderStatsRequest struct {
	StartDate  *time.Time              `json:"start_date,omitempty"`
	EndDate    *time.Time              `json:"end_date,omitempty"`
	Status     []entities.OrderStatus  `json:"status,omitempty"`
	CustomerID *string                 `json:"customer_id,omitempty"`
	CompanyID  *string                 `json:"company_id,omitempty"`
}

// GetRevenueByPeriodRequest represents a request to get revenue by period
type GetRevenueByPeriodRequest struct {
	StartDate time.Time `json:"start_date" validate:"required"`
	EndDate   time.Time `json:"end_date" validate:"required"`
	GroupBy   string    `json:"group_by" validate:"required,oneof=day week month quarter year"`
}

// GetTopCustomersRequest represents a request to get top customers
type GetTopCustomersRequest struct {
	StartDate time.Time `json:"start_date" validate:"required"`
	EndDate   time.Time `json:"end_date" validate:"required"`
	Limit     int       `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}

// GetSalesByProductRequest represents a request to get sales by product
type GetSalesByProductRequest struct {
	StartDate time.Time `json:"start_date" validate:"required"`
	EndDate   time.Time `json:"end_date" validate:"required"`
	Limit     int       `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
	CategoryID *string  `json:"category_id,omitempty"`
}

// GetOrderAnalyticsRequest represents a request to get comprehensive order analytics
type GetOrderAnalyticsRequest struct {
	StartDate time.Time `json:"start_date" validate:"required"`
	EndDate   time.Time `json:"end_date" validate:"required"`
	GroupBy   string    `json:"group_by,omitempty" validate:"omitempty,oneof=day week month quarter year"`
	CustomerID *string  `json:"customer_id,omitempty"`
	CompanyID  *string  `json:"company_id,omitempty"`
}

// OrderAnalyticsResponse represents comprehensive order analytics
type OrderAnalyticsResponse struct {
	OrderStats      *repositories.OrderStats          `json:"order_stats"`
	RevenueByPeriod []*repositories.RevenueByPeriod   `json:"revenue_by_period"`
	TopCustomers    []*repositories.CustomerOrderStats `json:"top_customers"`
	TopProducts     []*repositories.ProductSalesStats  `json:"top_products"`
	Trends          *OrderTrends                      `json:"trends,omitempty"`
}

// OrderTrends represents order trends and patterns
type OrderTrends struct {
	GrowthRate          decimal.Decimal `json:"growth_rate"`
	AverageOrderValue   decimal.Decimal `json:"average_order_value"`
	CustomerRetention   decimal.Decimal `json:"customer_retention"`
	PopularProducts     []string        `json:"popular_products"`
	PeakOrderTimes      []string        `json:"peak_order_times"`
	CommonCancelReasons []string        `json:"common_cancel_reasons"`
}

// CheckInventoryRequest represents a request to check inventory availability
type CheckInventoryRequest struct {
	Items []CheckInventoryItemRequest `json:"items" validate:"required,min=1"`
}

// CheckInventoryItemRequest represents an item to check inventory for
type CheckInventoryItemRequest struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"required,min=1"`
	WarehouseID *string `json:"warehouse_id,omitempty"`
}

// CheckInventoryResponse represents the response for inventory check
type CheckInventoryResponse struct {
	Available    bool                         `json:"available"`
	Items        []CheckInventoryItemResponse  `json:"items"`
	TotalValue   decimal.Decimal               `json:"total_value"`
	Suggestions  []InventorySuggestion         `json:"suggestions,omitempty"`
}

// CheckInventoryItemResponse represents inventory availability for an item
type CheckInventoryItemResponse struct {
	ProductID       string          `json:"product_id"`
	ProductName     string          `json:"product_name"`
	RequestedQty    int             `json:"requested_qty"`
	AvailableQty    int             `json:"available_qty"`
	CanFulfill      bool            `json:"can_fulfill"`
	BackorderAllowed bool           `json:"backorder_allowed"`
	UnitPrice       decimal.Decimal `json:"unit_price"`
	TotalValue      decimal.Decimal `json:"total_value"`
	Reason          string          `json:"reason,omitempty"`
	Alternatives    []ProductAlternative `json:"alternatives,omitempty"`
}

// ProductAlternative represents an alternative product suggestion
type ProductAlternative struct {
	ProductID   string          `json:"product_id"`
	ProductName string          `json:"product_name"`
	ProductSKU  string          `json:"product_sku"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	AvailableQty int            `json:"available_qty"`
	MatchScore   decimal.Decimal `json:"match_score"`
}

// InventorySuggestion represents inventory optimization suggestions
type InventorySuggestion struct {
	Type        string          `json:"type"` // "RESTOCK", "PROMOTE", "DISCONTINUE"
	ProductID   string          `json:"product_id"`
	ProductName string          `json:"product_name"`
	CurrentStock int            `json:"current_stock"`
	RecommendedAction string     `json:"recommended_action"`
	PotentialRevenue decimal.Decimal `json:"potential_revenue"`
	Priority    string          `json:"priority"` // "HIGH", "MEDIUM", "LOW"
}

// BulkUpdateStatusRequest represents a request to bulk update order status
type BulkUpdateStatusRequest struct {
	OrderIDs    []string               `json:"order_ids" validate:"required,min=1"`
	Status      entities.OrderStatus   `json:"status" validate:"required"`
	Reason      string                 `json:"reason,omitempty"`
	Notify      bool                   `json:"notify"`
	UpdatedBy   string                 `json:"updated_by" validate:"required,uuid"`
}

// BulkUpdateStatusResponse represents the response for bulk status update
type BulkUpdateStatusResponse struct {
	UpdatedCount int                  `json:"updated_count"`
	FailedCount  int                  `json:"failed_count"`
	Results      []BulkUpdateResult   `json:"results"`
}

// BulkCancelOrdersRequest represents a request to bulk cancel orders
type BulkCancelOrdersRequest struct {
	OrderIDs    []string          `json:"order_ids" validate:"required,min=1"`
	Reason      string            `json:"reason" validate:"required"`
	Refund      bool              `json:"refund"`
	Notify      bool              `json:"notify"`
	CancelledBy string            `json:"cancelled_by" validate:"required,uuid"`
}

// BulkCancelOrdersResponse represents the response for bulk cancel orders
type BulkCancelOrdersResponse struct {
	CancelledCount int                `json:"cancelled_count"`
	FailedCount    int                `json:"failed_count"`
	Results        []BulkUpdateResult `json:"results"`
}

// BulkUpdateResult represents the result of a bulk operation
type BulkUpdateResult struct {
	OrderID string `json:"order_id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// CloneOrderRequest represents a request to clone an order
type CloneOrderRequest struct {
	CopyItems      bool   `json:"copy_items"`
	CopyAddresses  bool   `json:"copy_addresses"`
	CopyNotes      bool   `json:"copy_notes"`
	CopyDiscounts  bool   `json:"copy_discounts"`
	NewCustomerID  *string `json:"new_customer_id,omitempty"`
	Notes          *string `json:"notes,omitempty"`
}

// Pagination represents pagination information
type Pagination struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// Errors
var (
	ErrOrderNotFound           = errors.New("order not found")
	ErrOrderAlreadyExists      = errors.New("order already exists")
	ErrInvalidOrderStatus      = errors.New("invalid order status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrInvalidOrderNumber      = errors.New("invalid order number")
	ErrInsufficientInventory   = errors.New("insufficient inventory")
	ErrCustomerNotFound        = errors.New("customer not found")
	ErrProductNotFound         = errors.New("product not found")
	ErrInvalidPaymentAmount    = errors.New("invalid payment amount")
	ErrOrderCannotBeCancelled  = errors.New("order cannot be cancelled")
	ErrOrderCannotBeShipped    = errors.New("order cannot be shipped")
	ErrOrderCannotBeReturned   = errors.New("order cannot be returned")
	ErrInvalidQuantity         = errors.New("invalid quantity")
	ErrInvalidAddress          = errors.New("invalid address")
	ErrInvalidCurrency         = errors.New("invalid currency")
	ErrPaymentFailed           = errors.New("payment failed")
	ErrRefundFailed            = errors.New("refund failed")
	ErrInvalidDiscount         = errors.New("invalid discount")
	ErrInvalidTaxRate          = errors.New("invalid tax rate")
	ErrInventoryReservationFailed = errors.New("inventory reservation failed")
	ErrOrderAlreadyPaid        = errors.New("order is already paid")
	ErrOrderNotPaid            = errors.New("order is not paid")
	ErrInvalidShippingMethod   = errors.New("invalid shipping method")
	ErrOrderAlreadyArchived    = errors.New("order is already archived")
	ErrOrderNotArchived        = errors.New("order is not archived")
)