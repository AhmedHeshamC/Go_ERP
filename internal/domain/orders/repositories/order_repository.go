package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"erpgo/internal/domain/orders/entities"
)

// OrderRepository defines the interface for order data operations
type OrderRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, order *entities.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Order, error)
	GetByOrderNumber(ctx context.Context, orderNumber string) (*entities.Order, error)
	Update(ctx context.Context, order *entities.Order) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Query operations
	List(ctx context.Context, filter OrderFilter) ([]*entities.Order, error)
	Count(ctx context.Context, filter OrderFilter) (int, error)

	// Status operations
	GetByStatus(ctx context.Context, status entities.OrderStatus) ([]*entities.Order, error)
	GetByStatusAndDateRange(ctx context.Context, status entities.OrderStatus, startDate, endDate time.Time) ([]*entities.Order, error)
	UpdateStatus(ctx context.Context, orderID uuid.UUID, newStatus entities.OrderStatus, updatedBy uuid.UUID) error

	// Customer operations
	GetByCustomerID(ctx context.Context, customerID uuid.UUID, filter OrderFilter) ([]*entities.Order, error)
	GetCustomerOrderHistory(ctx context.Context, customerID uuid.UUID, limit int) ([]*entities.Order, error)

	// Date range operations
	GetByDateRange(ctx context.Context, startDate, endDate time.Time, filter OrderFilter) ([]*entities.Order, error)
	GetOverdueOrders(ctx context.Context) ([]*entities.Order, error)
	GetPendingOrders(ctx context.Context) ([]*entities.Order, error)

	// Financial operations
	GetUnpaidOrders(ctx context.Context) ([]*entities.Order, error)
	GetOrdersByPaymentStatus(ctx context.Context, paymentStatus entities.PaymentStatus) ([]*entities.Order, error)
	UpdatePaymentStatus(ctx context.Context, orderID uuid.UUID, paymentStatus entities.PaymentStatus, paidAmount decimal.Decimal) error

	// Bulk operations
	BulkUpdateStatus(ctx context.Context, orderIDs []uuid.UUID, newStatus entities.OrderStatus, updatedBy uuid.UUID) error
	BulkCreate(ctx context.Context, orders []*entities.Order) error

	// Analytics and reporting
	GetOrderStats(ctx context.Context, filter OrderStatsFilter) (*OrderStats, error)
	GetRevenueByPeriod(ctx context.Context, startDate, endDate time.Time, groupBy string) ([]*RevenueByPeriod, error)
	GetTopCustomers(ctx context.Context, startDate, endDate time.Time, limit int) ([]*CustomerOrderStats, error)
	GetSalesByProduct(ctx context.Context, startDate, endDate time.Time, limit int) ([]*ProductSalesStats, error)

	// Advanced search
	SearchOrders(ctx context.Context, query string, filter OrderFilter) ([]*entities.Order, error)
	GetOrdersWithItems(ctx context.Context, orderIDs []uuid.UUID) ([]*entities.Order, error)

	// System operations
	ExistsByOrderNumber(ctx context.Context, orderNumber string) (bool, error)
	GenerateUniqueOrderNumber(ctx context.Context) (string, error)
}

// OrderItemRepository defines the interface for order item data operations
type OrderItemRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, item *entities.OrderItem) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.OrderItem, error)
	Update(ctx context.Context, item *entities.OrderItem) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Query operations
	GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entities.OrderItem, error)
	GetByProductID(ctx context.Context, productID uuid.UUID) ([]*entities.OrderItem, error)
	GetByOrderAndProduct(ctx context.Context, orderID, productID uuid.UUID) (*entities.OrderItem, error)

	// Status tracking
	UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status string) error
	UpdateShippedQuantity(ctx context.Context, itemID uuid.UUID, quantity int) error
	UpdateReturnedQuantity(ctx context.Context, itemID uuid.UUID, quantity int) error

	// Bulk operations
	BulkCreate(ctx context.Context, items []*entities.OrderItem) error
	BulkUpdate(ctx context.Context, items []*entities.OrderItem) error
	DeleteByOrderID(ctx context.Context, orderID uuid.UUID) error

	// Analytics
	GetProductOrderHistory(ctx context.Context, productID uuid.UUID, limit int) ([]*entities.OrderItem, error)
	GetItemsByStatus(ctx context.Context, status string) ([]*entities.OrderItem, error)
	GetLowStockItems(ctx context.Context, threshold int) ([]*ProductLowStock, error)
}

// CustomerRepository defines the interface for customer data operations
type CustomerRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, customer *entities.Customer) (*entities.Customer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Customer, error)
	GetByCustomerCode(ctx context.Context, customerCode string) (*entities.Customer, error)
	GetByCode(ctx context.Context, code string) (*entities.Customer, error)
	Update(ctx context.Context, customer *entities.Customer) (*entities.Customer, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// Query operations
	List(ctx context.Context, filter CustomerFilter) ([]*entities.Customer, error)
	Count(ctx context.Context, filter CustomerFilter) (int, error)
	Search(ctx context.Context, query string, filter CustomerFilter) ([]*entities.Customer, error)

	// Email and uniqueness checks
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByCustomerCode(ctx context.Context, customerCode string) (bool, error)
	GetByEmail(ctx context.Context, email string) (*entities.Customer, error)

	// Status operations
	GetActiveCustomers(ctx context.Context) ([]*entities.Customer, error)
	GetInactiveCustomers(ctx context.Context) ([]*entities.Customer, error)
	GetCustomersByType(ctx context.Context, customerType string) ([]*entities.Customer, error)

	// Financial operations
	GetCustomersWithCreditLimit(ctx context.Context) ([]*entities.Customer, error)
	UpdateCreditUsed(ctx context.Context, customerID uuid.UUID, amount decimal.Decimal) error
	GetCustomersWithOverdueCredit(ctx context.Context) ([]*entities.Customer, error)

	// Company operations
	GetByCompanyID(ctx context.Context, companyID uuid.UUID) ([]*entities.Customer, error)
	GetBusinessCustomers(ctx context.Context) ([]*entities.Customer, error)

	// Analytics and reporting
	GetCustomerStats(ctx context.Context, filter CustomerStatsFilter) (*CustomerStats, error)
	GetCustomerOrdersSummary(ctx context.Context, customerID uuid.UUID) (*CustomerOrdersSummary, error)
	GetTopCustomersByRevenue(ctx context.Context, startDate, endDate time.Time, limit int) ([]*CustomerRevenueStats, error)
	GetNewCustomersByPeriod(ctx context.Context, startDate, endDate time.Time, groupBy string) ([]*NewCustomersByPeriod, error)
	GetStats(ctx context.Context, startDate, endDate *time.Time) (*CustomerStats, error)

	// Order operations
	GetCustomerOrders(ctx context.Context, customerID string, limit int) ([]*entities.Order, error)
	TransferOrders(ctx context.Context, fromCustomerID, toCustomerID string) error

	// Bulk operations
	BulkUpdate(ctx context.Context, customers []*entities.Customer) error
	BulkCreate(ctx context.Context, customers []*entities.Customer) error
}

// ListCustomersRequest represents a request to list customers
type ListCustomersRequest struct {
	CompanyID         *uuid.UUID
	Type              *string
	IsActive          *bool
	IsVATExempt       *bool
	PreferredCurrency *string
	Source            *string
	Industry          *string
	CreatedAfter      *time.Time
	CreatedBefore     *time.Time
	Search            *string
	SortBy            *string
	SortOrder         *string
	Page              int
	Limit             int
}

// SearchCustomersRequest represents a request to search customers
type SearchCustomersRequest struct {
	Query             string
	CompanyID         *uuid.UUID
	Type              *string
	IsActive          *bool
	PreferredCurrency *string
	Source            *string
	CreatedAfter      *time.Time
	CreatedBefore     *time.Time
	SearchFields      []string
	SortBy            *string
	SortOrder         *string
	Page              int
	Limit             int
}

// ListCompaniesRequest represents a request to list companies
type ListCompaniesRequest struct {
	Industry      *string
	IsActive      *bool
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	Search        *string
	SortBy        *string
	SortOrder     *string
	Page          int
	Limit         int
}

// OrderAddressRepository defines the interface for order address data operations
type OrderAddressRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, address *entities.OrderAddress) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.OrderAddress, error)
	Update(ctx context.Context, address *entities.OrderAddress) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Query operations
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*entities.OrderAddress, error)
	GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entities.OrderAddress, error)
	GetByCustomerAndType(ctx context.Context, customerID uuid.UUID, addressType string) ([]*entities.OrderAddress, error)
	GetDefaultAddress(ctx context.Context, customerID uuid.UUID, addressType string) (*entities.OrderAddress, error)

	// Type-specific operations
	GetShippingAddresses(ctx context.Context, customerID uuid.UUID) ([]*entities.OrderAddress, error)
	GetBillingAddresses(ctx context.Context, customerID uuid.UUID) ([]*entities.OrderAddress, error)

	// Bulk operations
	DeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error
	DeleteByOrderID(ctx context.Context, orderID uuid.UUID) error
}

// CompanyRepository defines the interface for company data operations
type CompanyRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, company *entities.Company) (*entities.Company, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Company, error)
	GetByTaxID(ctx context.Context, taxID string) (*entities.Company, error)
	Update(ctx context.Context, company *entities.Company) (*entities.Company, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// Query operations
	List(ctx context.Context, filter CompanyFilter) ([]*entities.Company, error)
	Count(ctx context.Context, filter CompanyFilter) (int, error)
	Search(ctx context.Context, query string, filter CompanyFilter) ([]*entities.Company, error)

	// Status operations
	GetActiveCompanies(ctx context.Context) ([]*entities.Company, error)
	GetInactiveCompanies(ctx context.Context) ([]*entities.Company, error)
	GetCompaniesByIndustry(ctx context.Context, industry string) ([]*entities.Company, error)

	// Uniqueness checks
	ExistsByTaxID(ctx context.Context, taxID string) (bool, error)
	ExistsByName(ctx context.Context, name string) (bool, error)

	// Analytics and reporting
	GetStats(ctx context.Context, startDate, endDate *time.Time) (*CompanyStats, error)
}

// CompanyStats represents company statistics
type CompanyStats struct {
	TotalCompanies             int64            `json:"total_companies"`
	ActiveCompanies            int64            `json:"active_companies"`
	InactiveCompanies          int64            `json:"inactive_companies"`
	CompaniesByIndustry        map[string]int64 `json:"companies_by_industry"`
	AverageCustomersPerCompany float64          `json:"average_customers_per_company"`
	NewCompaniesThisMonth      int64            `json:"new_companies_this_month"`
	NewCompaniesThisYear       int64            `json:"new_companies_this_year"`
}

// Filter types

// OrderFilter defines filtering options for order queries
type OrderFilter struct {
	// Basic filters
	Search         string             `json:"search,omitempty"`
	Status         []entities.OrderStatus `json:"status,omitempty"`
	PaymentStatus  []entities.PaymentStatus `json:"payment_status,omitempty"`
	Priority       []entities.OrderPriority `json:"priority,omitempty"`
	Type           []entities.OrderType `json:"type,omitempty"`
	ShippingMethod []entities.ShippingMethod `json:"shipping_method,omitempty"`

	// Customer filters
	CustomerID     *uuid.UUID         `json:"customer_id,omitempty"`
	CompanyID      *uuid.UUID         `json:"company_id,omitempty"`
	CustomerType   string             `json:"customer_type,omitempty"`

	// Date filters
	StartDate      *time.Time         `json:"start_date,omitempty"`
	EndDate        *time.Time         `json:"end_date,omitempty"`
	ShippingDateFrom *time.Time       `json:"shipping_date_from,omitempty"`
	ShippingDateTo   *time.Time       `json:"shipping_date_to,omitempty"`

	// Financial filters
	MinTotalAmount *decimal.Decimal   `json:"min_total_amount,omitempty"`
	MaxTotalAmount *decimal.Decimal   `json:"max_total_amount,omitempty"`
	Currency       string             `json:"currency,omitempty"`

	// Created by filter
	CreatedBy      *uuid.UUID         `json:"created_by,omitempty"`

	// Pagination
	Page           int                `json:"page"`
	Limit          int                `json:"limit"`
	Offset         int                `json:"offset,omitempty"`

	// Sorting
	SortBy         string             `json:"sort_by,omitempty"`
	SortOrder      string             `json:"sort_order,omitempty"`
}

// CustomerFilter defines filtering options for customer queries
type CustomerFilter struct {
	Search         string             `json:"search,omitempty"`
	Type           string             `json:"type,omitempty"`
	CompanyID      *uuid.UUID         `json:"company_id,omitempty"`
	IsActive       *bool              `json:"is_active,omitempty"`
	Industry       *string            `json:"industry,omitempty"`
	Source         string             `json:"source,omitempty"`

	// Financial filters
	HasCreditLimit *bool              `json:"has_credit_limit,omitempty"`
	MinCreditLimit *decimal.Decimal   `json:"min_credit_limit,omitempty"`
	MaxCreditLimit *decimal.Decimal   `json:"max_credit_limit,omitempty"`

	// Registration date filters
	StartDate      *time.Time         `json:"start_date,omitempty"`
	EndDate        *time.Time         `json:"end_date,omitempty"`

	// Pagination
	Page           int                `json:"page"`
	Limit          int                `json:"limit"`
	Offset         int                `json:"offset,omitempty"`

	// Sorting
	SortBy         string             `json:"sort_by,omitempty"`
	SortOrder      string             `json:"sort_order,omitempty"`
}

// CompanyFilter defines filtering options for company queries
type CompanyFilter struct {
	Search         string             `json:"search,omitempty"`
	Industry       *string            `json:"industry,omitempty"`
	IsActive       *bool              `json:"is_active,omitempty"`

	// Registration date filters
	StartDate      *time.Time         `json:"start_date,omitempty"`
	EndDate        *time.Time         `json:"end_date,omitempty"`

	// Pagination
	Page           int                `json:"page"`
	Limit          int                `json:"limit"`
	Offset         int                `json:"offset,omitempty"`

	// Sorting
	SortBy         string             `json:"sort_by,omitempty"`
	SortOrder      string             `json:"sort_order,omitempty"`
}

// Analytics filter types

// OrderStatsFilter defines filters for order statistics
type OrderStatsFilter struct {
	StartDate      time.Time          `json:"start_date"`
	EndDate        time.Time          `json:"end_date"`
	Status         []entities.OrderStatus `json:"status,omitempty"`
	CustomerID     *uuid.UUID         `json:"customer_id,omitempty"`
	CompanyID      *uuid.UUID         `json:"company_id,omitempty"`
}

// CustomerStatsFilter defines filters for customer statistics
type CustomerStatsFilter struct {
	StartDate      time.Time          `json:"start_date"`
	EndDate        time.Time          `json:"end_date"`
	Type           string             `json:"type,omitempty"`
	IsActive       *bool              `json:"is_active,omitempty"`
}

// Analytics result types

// OrderStats represents order statistics
type OrderStats struct {
	TotalOrders    int64              `json:"total_orders"`
	TotalRevenue   decimal.Decimal    `json:"total_revenue"`
	AverageOrderValue decimal.Decimal  `json:"average_order_value"`
	StatusCounts   map[string]int64   `json:"status_counts"`
	PaymentStatusCounts map[string]int64 `json:"payment_status_counts"`
}

// RevenueByPeriod represents revenue grouped by time period
type RevenueByPeriod struct {
	Period         string             `json:"period"`
	Revenue        decimal.Decimal    `json:"revenue"`
	OrderCount     int64              `json:"order_count"`
	AverageOrderValue decimal.Decimal  `json:"average_order_value"`
}

// CustomerOrderStats represents customer order statistics
type CustomerOrderStats struct {
	CustomerID     uuid.UUID          `json:"customer_id"`
	CustomerName   string             `json:"customer_name"`
	CustomerEmail  string             `json:"customer_email"`
	OrderCount     int64              `json:"order_count"`
	TotalRevenue   decimal.Decimal    `json:"total_revenue"`
	AverageOrderValue decimal.Decimal  `json:"average_order_value"`
	LastOrderDate  *time.Time         `json:"last_order_date,omitempty"`
}

// ProductSalesStats represents product sales statistics
type ProductSalesStats struct {
	ProductID      uuid.UUID          `json:"product_id"`
	ProductSKU     string             `json:"product_sku"`
	ProductName    string             `json:"product_name"`
	QuantitySold   int64              `json:"quantity_sold"`
	TotalRevenue   decimal.Decimal    `json:"total_revenue"`
	OrderCount     int64              `json:"order_count"`
}

// ProductLowStock represents products with low stock
type ProductLowStock struct {
	ProductID      uuid.UUID          `json:"product_id"`
	ProductSKU     string             `json:"product_sku"`
	ProductName    string             `json:"product_name"`
	CurrentStock   int                `json:"current_stock"`
	ReorderLevel   int                `json:"reorder_level"`
	PendingOrders  int                `json:"pending_orders"`
}

// CustomerStats represents customer statistics
type CustomerStats struct {
	TotalCustomers int64              `json:"total_customers"`
	ActiveCustomers int64             `json:"active_customers"`
	NewCustomers   int64              `json:"new_customers"`
	CustomersByType map[string]int64  `json:"customers_by_type"`
	CustomersBySource map[string]int64 `json:"customers_by_source"`
}

// CustomerOrdersSummary represents a customer's order summary
type CustomerOrdersSummary struct {
	CustomerID     uuid.UUID          `json:"customer_id"`
	TotalOrders    int64              `json:"total_orders"`
	TotalRevenue   decimal.Decimal    `json:"total_revenue"`
	AverageOrderValue decimal.Decimal  `json:"average_order_value"`
	LastOrderDate  *time.Time         `json:"last_order_date,omitempty"`
	FirstOrderDate  *time.Time        `json:"first_order_date,omitempty"`
	StatusCounts   map[string]int64   `json:"status_counts"`
}

// CustomerRevenueStats represents customer revenue statistics
type CustomerRevenueStats struct {
	CustomerID     uuid.UUID          `json:"customer_id"`
	CustomerName   string             `json:"customer_name"`
	CustomerEmail  string             `json:"customer_email"`
	CompanyName    *string            `json:"company_name,omitempty"`
	TotalRevenue   decimal.Decimal    `json:"total_revenue"`
	OrderCount     int64              `json:"order_count"`
	AverageOrderValue decimal.Decimal  `json:"average_order_value"`
}

// NewCustomersByPeriod represents new customers grouped by time period
type NewCustomersByPeriod struct {
	Period         string             `json:"period"`
	NewCustomers   int64              `json:"new_customers"`
	TotalCustomers int64              `json:"total_customers"`
}

// Pagination represents pagination information
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}