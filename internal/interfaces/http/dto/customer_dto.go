package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Customer-related DTOs

// CustomerResponse represents customer information returned in responses
type CustomerResponse struct {
	ID                uuid.UUID       `json:"id"`
	CustomerCode      string          `json:"customer_code"`
	CompanyID         *uuid.UUID      `json:"company_id,omitempty"`
	Type              string          `json:"type"`
	FirstName         string          `json:"first_name"`
	LastName          string          `json:"last_name"`
	FullName          string          `json:"full_name"`
	Email             *string         `json:"email,omitempty"`
	Phone             *string         `json:"phone,omitempty"`
	Website           *string         `json:"website,omitempty"`
	CompanyName       *string         `json:"company_name,omitempty"`
	TaxID             *string         `json:"tax_id,omitempty"`
	Industry          *string         `json:"industry,omitempty"`
	CreditLimit       decimal.Decimal `json:"credit_limit"`
	CreditUsed        decimal.Decimal `json:"credit_used"`
	CreditAvailable   decimal.Decimal `json:"credit_available"`
	Terms             string          `json:"terms"`
	IsActive          bool            `json:"is_active"`
	IsVATExempt       bool            `json:"is_vat_exempt"`
	PreferredCurrency string          `json:"preferred_currency"`
	Notes             *string         `json:"notes,omitempty"`
	Source            string          `json:"source"`
	OrderCount        int64           `json:"order_count"`
	TotalOrdersValue  decimal.Decimal `json:"total_orders_value"`
	LastOrderDate     *time.Time      `json:"last_order_date,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// CompanyResponse represents company information returned in responses
type CompanyResponse struct {
	ID            uuid.UUID `json:"id"`
	CompanyName   string    `json:"company_name"`
	LegalName     string    `json:"legal_name"`
	TaxID         string    `json:"tax_id"`
	Industry      *string   `json:"industry,omitempty"`
	Website       *string   `json:"website,omitempty"`
	Phone         *string   `json:"phone,omitempty"`
	Email         string    `json:"email"`
	Address       string    `json:"address"`
	City          string    `json:"city"`
	State         string    `json:"state"`
	Country       string    `json:"country"`
	PostalCode    string    `json:"postal_code"`
	IsActive      bool      `json:"is_active"`
	CustomerCount int64     `json:"customer_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateCustomerRequest represents a request to create a new customer
type CreateCustomerRequest struct {
	CompanyID         *uuid.UUID      `json:"company_id,omitempty" binding:"omitempty,uuid"`
	Type              string          `json:"type" binding:"required,oneof=INDIVIDUAL BUSINESS GOVERNMENT NON_PROFIT"`
	FirstName         string          `json:"first_name" binding:"required,min=2,max=100"`
	LastName          string          `json:"last_name" binding:"required,min=2,max=100"`
	Email             *string         `json:"email,omitempty" binding:"omitempty,email,max=255"`
	Phone             *string         `json:"phone,omitempty" binding:"omitempty,max=50"`
	Website           *string         `json:"website,omitempty" binding:"omitempty,url,max=500"`
	CompanyName       *string         `json:"company_name,omitempty" binding:"omitempty,max=200"`
	TaxID             *string         `json:"tax_id,omitempty" binding:"omitempty,max=50"`
	Industry          *string         `json:"industry,omitempty" binding:"omitempty,max=100"`
	CreditLimit       decimal.Decimal `json:"credit_limit" binding:"required,gt=0"`
	Terms             string          `json:"terms" binding:"required,match_pattern=^NET\\d+$"`
	IsVATExempt       bool            `json:"is_vat_exempt"`
	PreferredCurrency string          `json:"preferred_currency" binding:"required,len=3,uppercase"`
	Notes             *string         `json:"notes,omitempty"`
	Source            string          `json:"source" binding:"required,oneof=WEB PHONE EMAIL REFERRAL WALK_IN SOCIAL ADVERTISEMENT OTHER"`
}

// UpdateCustomerRequest represents a request to update a customer
type UpdateCustomerRequest struct {
	CompanyID         *uuid.UUID       `json:"company_id,omitempty" binding:"omitempty,uuid"`
	Type              *string          `json:"type,omitempty" binding:"omitempty,oneof=INDIVIDUAL BUSINESS GOVERNMENT NON_PROFIT"`
	FirstName         *string          `json:"first_name,omitempty" binding:"omitempty,min=2,max=100"`
	LastName          *string          `json:"last_name,omitempty" binding:"omitempty,min=2,max=100"`
	Email             *string          `json:"email,omitempty" binding:"omitempty,email,max=255"`
	Phone             *string          `json:"phone,omitempty" binding:"omitempty,max=50"`
	Website           *string          `json:"website,omitempty" binding:"omitempty,url,max=500"`
	CompanyName       *string          `json:"company_name,omitempty" binding:"omitempty,max=200"`
	TaxID             *string          `json:"tax_id,omitempty" binding:"omitempty,max=50"`
	Industry          *string          `json:"industry,omitempty" binding:"omitempty,max=100"`
	CreditLimit       *decimal.Decimal `json:"credit_limit,omitempty" binding:"omitempty,gt=0"`
	Terms             *string          `json:"terms,omitempty" binding:"omitempty,match_pattern=^NET\\d+$"`
	IsVATExempt       *bool            `json:"is_vat_exempt,omitempty"`
	PreferredCurrency *string          `json:"preferred_currency,omitempty" binding:"omitempty,len=3,uppercase"`
	Notes             *string          `json:"notes,omitempty"`
	IsActive          *bool            `json:"is_active,omitempty"`
}

// CreateCompanyRequest represents a request to create a new company
type CreateCompanyRequest struct {
	CompanyName string  `json:"company_name" binding:"required,min=2,max=200"`
	LegalName   string  `json:"legal_name" binding:"required,min=2,max=200"`
	TaxID       string  `json:"tax_id" binding:"required,min=2,max=50"`
	Industry    *string `json:"industry,omitempty" binding:"omitempty,max=100"`
	Website     *string `json:"website,omitempty" binding:"omitempty,url,max=500"`
	Phone       *string `json:"phone,omitempty" binding:"omitempty,max=50"`
	Email       string  `json:"email" binding:"required,email,max=255"`
	Address     string  `json:"address" binding:"required,min=5,max=500"`
	City        string  `json:"city" binding:"required,min=2,max=100"`
	State       string  `json:"state" binding:"required,min=2,max=100"`
	Country     string  `json:"country" binding:"required,min=2,max=100"`
	PostalCode  string  `json:"postal_code" binding:"required,min=3,max=20"`
}

// UpdateCompanyRequest represents a request to update a company
type UpdateCompanyRequest struct {
	CompanyName *string `json:"company_name,omitempty" binding:"omitempty,min=2,max=200"`
	LegalName   *string `json:"legal_name,omitempty" binding:"omitempty,min=2,max=200"`
	TaxID       *string `json:"tax_id,omitempty" binding:"omitempty,min=2,max=50"`
	Industry    *string `json:"industry,omitempty" binding:"omitempty,max=100"`
	Website     *string `json:"website,omitempty" binding:"omitempty,url,max=500"`
	Phone       *string `json:"phone,omitempty" binding:"omitempty,max=50"`
	Email       *string `json:"email,omitempty" binding:"omitempty,email,max=255"`
	Address     *string `json:"address,omitempty" binding:"omitempty,min=5,max=500"`
	City        *string `json:"city,omitempty" binding:"omitempty,min=2,max=100"`
	State       *string `json:"state,omitempty" binding:"omitempty,min=2,max=100"`
	Country     *string `json:"country,omitempty" binding:"omitempty,min=2,max=100"`
	PostalCode  *string `json:"postal_code,omitempty" binding:"omitempty,min=3,max=20"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// ListCustomersRequest represents a request to list customers
type ListCustomersRequest struct {
	CompanyID         *uuid.UUID `json:"company_id,omitempty"`
	Type              *string    `json:"type,omitempty"`
	IsActive          *bool      `json:"is_active,omitempty"`
	IsVATExempt       *bool      `json:"is_vat_exempt,omitempty"`
	PreferredCurrency *string    `json:"preferred_currency,omitempty"`
	Source            *string    `json:"source,omitempty"`
	Industry          *string    `json:"industry,omitempty"`
	CreatedAfter      *time.Time `json:"created_after,omitempty"`
	CreatedBefore     *time.Time `json:"created_before,omitempty"`
	Search            *string    `json:"search,omitempty"`
	SortBy            *string    `json:"sort_by,omitempty"`
	SortOrder         *string    `json:"sort_order,omitempty"`
	Page              int        `json:"page" binding:"omitempty,min=1"`
	Limit             int        `json:"limit" binding:"omitempty,min=1,max=100"`
}

// SearchCustomersRequest represents a request to search customers
type SearchCustomersRequest struct {
	Query             string     `json:"query" binding:"required,min=2"`
	CompanyID         *uuid.UUID `json:"company_id,omitempty"`
	Type              *string    `json:"type,omitempty"`
	IsActive          *bool      `json:"is_active,omitempty"`
	PreferredCurrency *string    `json:"preferred_currency,omitempty"`
	Source            *string    `json:"source,omitempty"`
	CreatedAfter      *time.Time `json:"created_after,omitempty"`
	CreatedBefore     *time.Time `json:"created_before,omitempty"`
	SearchFields      []string   `json:"search_fields,omitempty"`
	SortBy            *string    `json:"sort_by,omitempty"`
	SortOrder         *string    `json:"sort_order,omitempty"`
	Page              int        `json:"page" binding:"omitempty,min=1"`
	Limit             int        `json:"limit" binding:"omitempty,min=1,max=100"`
}

// ListCompaniesRequest represents a request to list companies
type ListCompaniesRequest struct {
	Industry      *string    `json:"industry,omitempty"`
	IsActive      *bool      `json:"is_active,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	Search        *string    `json:"search,omitempty"`
	SortBy        *string    `json:"sort_by,omitempty"`
	SortOrder     *string    `json:"sort_order,omitempty"`
	Page          int        `json:"page" binding:"omitempty,min=1"`
	Limit         int        `json:"limit" binding:"omitempty,min=1,max=100"`
}

// ListCustomersResponse represents a paginated customers list response
type ListCustomersResponse struct {
	Customers  []*CustomerResponse `json:"customers"`
	Pagination *Pagination         `json:"pagination"`
}

// SearchCustomersResponse represents a search customers response
type SearchCustomersResponse struct {
	Customers  []*CustomerResponse `json:"customers"`
	Pagination *Pagination         `json:"pagination"`
	TotalCount int                 `json:"total_count"`
}

// ListCompaniesResponse represents a paginated companies list response
type ListCompaniesResponse struct {
	Companies  []*CompanyResponse `json:"companies"`
	Pagination *Pagination        `json:"pagination"`
}

// CustomerStatsResponse represents customer statistics response
type CustomerStatsResponse struct {
	TotalCustomers        int64            `json:"total_customers"`
	ActiveCustomers       int64            `json:"active_customers"`
	InactiveCustomers     int64            `json:"inactive_customers"`
	CustomersByType       map[string]int64 `json:"customers_by_type"`
	CustomersBySource     map[string]int64 `json:"customers_by_source"`
	CustomersByIndustry   map[string]int64 `json:"customers_by_industry"`
	AverageCreditLimit    decimal.Decimal  `json:"average_credit_limit"`
	TotalCreditLimit      decimal.Decimal  `json:"total_credit_limit"`
	TotalCreditUsed       decimal.Decimal  `json:"total_credit_used"`
	NewCustomersThisMonth int64            `json:"new_customers_this_month"`
	NewCustomersThisYear  int64            `json:"new_customers_this_year"`
}

// CompanyStatsResponse represents company statistics response
type CompanyStatsResponse struct {
	TotalCompanies             int64            `json:"total_companies"`
	ActiveCompanies            int64            `json:"active_companies"`
	InactiveCompanies          int64            `json:"inactive_companies"`
	CompaniesByIndustry        map[string]int64 `json:"companies_by_industry"`
	AverageCustomersPerCompany float64          `json:"average_customers_per_company"`
	NewCompaniesThisMonth      int64            `json:"new_companies_this_month"`
	NewCompaniesThisYear       int64            `json:"new_companies_this_year"`
}

// UpdateCustomerCreditRequest represents a request to update customer credit
type UpdateCustomerCreditRequest struct {
	CreditLimit decimal.Decimal `json:"credit_limit" binding:"required,gt=0"`
	Adjustment  decimal.Decimal `json:"adjustment" binding:"required"`
	Reason      *string         `json:"reason,omitempty"`
}

// CustomerCreditResponse represents customer credit information
type CustomerCreditResponse struct {
	CustomerID          uuid.UUID       `json:"customer_id"`
	CustomerCode        string          `json:"customer_code"`
	CustomerName        string          `json:"customer_name"`
	CreditLimit         decimal.Decimal `json:"credit_limit"`
	CreditUsed          decimal.Decimal `json:"credit_used"`
	CreditAvailable     decimal.Decimal `json:"credit_available"`
	AvailablePercentage float64         `json:"available_percentage"`
	LastUpdated         time.Time       `json:"last_updated"`
}

// BulkUpdateCustomersRequest represents a request to bulk update customers
type BulkUpdateCustomersRequest struct {
	CustomerIDs []string               `json:"customer_ids" binding:"required,min=1"`
	Updates     map[string]interface{} `json:"updates"`
	Notify      bool                   `json:"notify"`
}

// BulkUpdateCustomersResponse represents bulk update customers response
type BulkUpdateCustomersResponse struct {
	UpdatedCount     int               `json:"updated_count"`
	FailedCount      int               `json:"failed_count"`
	UpdatedCustomers []string          `json:"updated_customers"`
	FailedCustomers  []FailedOperation `json:"failed_customers"`
}
