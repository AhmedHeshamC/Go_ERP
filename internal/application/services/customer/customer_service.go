package customer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/domain/orders/repositories"
)

// Service defines the business logic interface for customer management
type Service interface {
	// Customer management
	CreateCustomer(ctx context.Context, req *CreateCustomerRequest) (*entities.Customer, error)
	GetCustomer(ctx context.Context, id string) (*entities.Customer, error)
	GetCustomerByCode(ctx context.Context, code string) (*entities.Customer, error)
	UpdateCustomer(ctx context.Context, id string, req *UpdateCustomerRequest) (*entities.Customer, error)
	DeleteCustomer(ctx context.Context, id string) error
	ListCustomers(ctx context.Context, req *ListCustomersRequest) (*ListCustomersResponse, error)
	SearchCustomers(ctx context.Context, req *SearchCustomersRequest) (*SearchCustomersResponse, error)

	// Company management
	CreateCompany(ctx context.Context, req *CreateCompanyRequest) (*entities.Company, error)
	GetCompany(ctx context.Context, id string) (*entities.Company, error)
	UpdateCompany(ctx context.Context, id string, req *UpdateCompanyRequest) (*entities.Company, error)
	DeleteCompany(ctx context.Context, id string) error
	ListCompanies(ctx context.Context, req *ListCompaniesRequest) (*ListCompaniesResponse, error)

	// Customer credit management
	UpdateCustomerCredit(ctx context.Context, id string, req *UpdateCustomerCreditRequest) (*entities.Customer, error)
	GetCustomerCredit(ctx context.Context, id string) (*CustomerCreditResponse, error)
	AdjustCustomerCredit(ctx context.Context, id string, adjustment decimal.Decimal, reason *string) (*entities.Customer, error)

	// Customer analytics and reporting
	GetCustomerStats(ctx context.Context, req *GetCustomerStatsRequest) (*CustomerStatsResponse, error)
	GetCompanyStats(ctx context.Context, req *GetCompanyStatsRequest) (*CompanyStatsResponse, error)
	GetTopCustomersByRevenue(ctx context.Context, req *GetTopCustomersRequest) ([]*entities.CustomerSummary, error)
	GetCustomerOrderHistory(ctx context.Context, customerID string, limit int) ([]*entities.Order, error)

	// Bulk operations
	BulkUpdateCustomers(ctx context.Context, req *BulkUpdateCustomersRequest) (*BulkUpdateCustomersResponse, error)
	BulkCreateCustomers(ctx context.Context, req *BulkCreateCustomersRequest) (*BulkCreateCustomersResponse, error)

	// Customer management utilities
	GenerateCustomerCode(ctx context.Context) (string, error)
	ValidateCustomerData(ctx context.Context, req *CreateCustomerRequest) error
	MergeCustomers(ctx context.Context, primaryID, secondaryID string) (*entities.Customer, error)
	ActivateCustomer(ctx context.Context, id string) (*entities.Customer, error)
	DeactivateCustomer(ctx context.Context, id string) (*entities.Customer, error)
}

// customerService implements the Service interface
type customerService struct {
	customerRepo repositories.CustomerRepository
	companyRepo   repositories.CompanyRepository
}

// NewCustomerService creates a new customer service
func NewCustomerService(
	customerRepo repositories.CustomerRepository,
	companyRepo repositories.CompanyRepository,
) Service {
	return &customerService{
		customerRepo: customerRepo,
		companyRepo:   companyRepo,
	}
}

// Request/Response DTOs

// CreateCustomerRequest represents a request to create a new customer
type CreateCustomerRequest struct {
	CompanyID         *uuid.UUID      `json:"company_id,omitempty"`
	Type              string          `json:"type"`
	FirstName         string          `json:"first_name"`
	LastName          string          `json:"last_name"`
	Email             *string         `json:"email,omitempty"`
	Phone             *string         `json:"phone,omitempty"`
	Website           *string         `json:"website,omitempty"`
	CompanyName       *string         `json:"company_name,omitempty"`
	TaxID             *string         `json:"tax_id,omitempty"`
	Industry          *string         `json:"industry,omitempty"`
	CreditLimit       decimal.Decimal `json:"credit_limit"`
	Terms             string          `json:"terms"`
	IsVATExempt       bool            `json:"is_vat_exempt"`
	PreferredCurrency string          `json:"preferred_currency"`
	Notes             *string         `json:"notes,omitempty"`
	Source            string          `json:"source"`
}

// UpdateCustomerRequest represents a request to update a customer
type UpdateCustomerRequest struct {
	CompanyID         *uuid.UUID      `json:"company_id,omitempty"`
	Type              *string         `json:"type,omitempty"`
	FirstName         *string         `json:"first_name,omitempty"`
	LastName          *string         `json:"last_name,omitempty"`
	Email             *string         `json:"email,omitempty"`
	Phone             *string         `json:"phone,omitempty"`
	Website           *string         `json:"website,omitempty"`
	CompanyName       *string         `json:"company_name,omitempty"`
	TaxID             *string         `json:"tax_id,omitempty"`
	Industry          *string         `json:"industry,omitempty"`
	CreditLimit       *decimal.Decimal `json:"credit_limit,omitempty"`
	Terms             *string         `json:"terms,omitempty"`
	IsVATExempt       *bool           `json:"is_vat_exempt,omitempty"`
	PreferredCurrency *string         `json:"preferred_currency,omitempty"`
	Notes             *string         `json:"notes,omitempty"`
	IsActive          *bool           `json:"is_active,omitempty"`
}

// CreateCompanyRequest represents a request to create a new company
type CreateCompanyRequest struct {
	CompanyName string  `json:"company_name"`
	LegalName   string  `json:"legal_name"`
	TaxID       string  `json:"tax_id"`
	Industry    *string `json:"industry,omitempty"`
	Website     *string `json:"website,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Email       string  `json:"email"`
	Address     string  `json:"address"`
	City        string  `json:"city"`
	State       string  `json:"state"`
	Country     string  `json:"country"`
	PostalCode  string  `json:"postal_code"`
}

// UpdateCompanyRequest represents a request to update a company
type UpdateCompanyRequest struct {
	CompanyName *string `json:"company_name,omitempty"`
	LegalName   *string `json:"legal_name,omitempty"`
	TaxID       *string `json:"tax_id,omitempty"`
	Industry    *string `json:"industry,omitempty"`
	Website     *string `json:"website,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Email       *string `json:"email,omitempty"`
	Address     *string `json:"address,omitempty"`
	City        *string `json:"city,omitempty"`
	State       *string `json:"state,omitempty"`
	Country     *string `json:"country,omitempty"`
	PostalCode  *string `json:"postal_code,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
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

// ListCustomersResponse represents a paginated customers list response
type ListCustomersResponse struct {
	Customers  []*entities.Customer
	Pagination *repositories.Pagination
}

// SearchCustomersResponse represents a search customers response
type SearchCustomersResponse struct {
	Customers  []*entities.Customer
	Pagination *repositories.Pagination
	TotalCount int
}

// ListCompaniesResponse represents a paginated companies list response
type ListCompaniesResponse struct {
	Companies  []*entities.Company
	Pagination *repositories.Pagination
}

// UpdateCustomerCreditRequest represents a request to update customer credit
type UpdateCustomerCreditRequest struct {
	CreditLimit decimal.Decimal `json:"credit_limit"`
	Adjustment  decimal.Decimal `json:"adjustment"`
	Reason      *string         `json:"reason,omitempty"`
}

// CustomerCreditResponse represents customer credit information
type CustomerCreditResponse struct {
	CustomerID         uuid.UUID       `json:"customer_id"`
	CustomerCode       string          `json:"customer_code"`
	CustomerName       string          `json:"customer_name"`
	CreditLimit        decimal.Decimal `json:"credit_limit"`
	CreditUsed         decimal.Decimal `json:"credit_used"`
	CreditAvailable    decimal.Decimal `json:"credit_available"`
	AvailablePercentage float64        `json:"available_percentage"`
	LastUpdated        time.Time       `json:"last_updated"`
}

// GetCustomerStatsRequest represents a request for customer statistics
type GetCustomerStatsRequest struct {
	StartDate *time.Time
	EndDate   *time.Time
}

// CustomerStatsResponse represents customer statistics response
type CustomerStatsResponse struct {
	TotalCustomers           int64            `json:"total_customers"`
	ActiveCustomers          int64            `json:"active_customers"`
	InactiveCustomers        int64            `json:"inactive_customers"`
	CustomersByType          map[string]int64 `json:"customers_by_type"`
	CustomersBySource        map[string]int64 `json:"customers_by_source"`
	CustomersByIndustry      map[string]int64 `json:"customers_by_industry"`
	AverageCreditLimit       decimal.Decimal  `json:"average_credit_limit"`
	TotalCreditLimit         decimal.Decimal  `json:"total_credit_limit"`
	TotalCreditUsed          decimal.Decimal  `json:"total_credit_used"`
	NewCustomersThisMonth    int64            `json:"new_customers_this_month"`
	NewCustomersThisYear     int64            `json:"new_customers_this_year"`
}

// GetCompanyStatsRequest represents a request for company statistics
type GetCompanyStatsRequest struct {
	StartDate *time.Time
	EndDate   *time.Time
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

// GetTopCustomersRequest represents a request to get top customers
type GetTopCustomersRequest struct {
	Limit     int
	StartDate *time.Time
	EndDate   *time.Time
}

// BulkUpdateCustomersRequest represents a request to bulk update customers
type BulkUpdateCustomersRequest struct {
	CustomerIDs []string
	Updates     map[string]interface{}
	Notify      bool
}

// BulkUpdateCustomersResponse represents bulk update customers response
type BulkUpdateCustomersResponse struct {
	UpdatedCount    int
	FailedCount     int
	UpdatedCustomers []string
	FailedCustomers []FailedOperation
}

// BulkCreateCustomersRequest represents a request to bulk create customers
type BulkCreateCustomersRequest struct {
	Customers []CreateCustomerRequest
}

// BulkCreateCustomersResponse represents bulk create customers response
type BulkCreateCustomersResponse struct {
	CreatedCount    int
	FailedCount     int
	CreatedCustomers []string
	FailedCustomers []FailedOperation
}

// FailedOperation represents a failed bulk operation
type FailedOperation struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

// Service Implementation

// CreateCustomer creates a new customer
func (s *customerService) CreateCustomer(ctx context.Context, req *CreateCustomerRequest) (*entities.Customer, error) {
	// Validate customer data
	if err := s.ValidateCustomerData(ctx, req); err != nil {
		return nil, err
	}

	// Generate customer code
	customerCode, err := s.GenerateCustomerCode(ctx)
	if err != nil {
		return nil, err
	}

	customer := &entities.Customer{
		ID:                uuid.New(),
		CustomerCode:      customerCode,
		CompanyID:         req.CompanyID,
		Type:              req.Type,
		FirstName:         req.FirstName,
		LastName:          req.LastName,
		Email:             stringValue(req.Email),
		Phone:             stringValue(req.Phone),
		Website:           req.Website,
		CompanyName:       req.CompanyName,
		TaxID:             req.TaxID,
		Industry:          req.Industry,
		CreditLimit:       req.CreditLimit,
		CreditUsed:        decimal.Zero,
		Terms:             req.Terms,
		IsActive:          true,
		IsVATExempt:       req.IsVATExempt,
		PreferredCurrency: req.PreferredCurrency,
		Notes:             req.Notes,
		Source:            req.Source,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	return s.customerRepo.Create(ctx, customer)
}

// GetCustomer retrieves a customer by ID
func (s *customerService) GetCustomer(ctx context.Context, id string) (*entities.Customer, error) {
	customerID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid customer ID: %w", err)
	}
	return s.customerRepo.GetByID(ctx, customerID)
}

// GetCustomerByCode retrieves a customer by code
func (s *customerService) GetCustomerByCode(ctx context.Context, code string) (*entities.Customer, error) {
	return s.customerRepo.GetByCode(ctx, code)
}

// UpdateCustomer updates an existing customer
func (s *customerService) UpdateCustomer(ctx context.Context, id string, req *UpdateCustomerRequest) (*entities.Customer, error) {
	customerID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid customer ID: %w", err)
	}
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.CompanyID != nil {
		customer.CompanyID = req.CompanyID
	}
	if req.Type != nil {
		customer.Type = *req.Type
	}
	if req.FirstName != nil {
		customer.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		customer.LastName = *req.LastName
	}
	if req.Email != nil {
		customer.Email = *req.Email
	}
	if req.Phone != nil {
		customer.Phone = *req.Phone
	}
	if req.Website != nil {
		customer.Website = req.Website
	}
	if req.CompanyName != nil {
		customer.CompanyName = req.CompanyName
	}
	if req.TaxID != nil {
		customer.TaxID = req.TaxID
	}
	if req.Industry != nil {
		customer.Industry = req.Industry
	}
	if req.CreditLimit != nil {
		customer.CreditLimit = *req.CreditLimit
	}
	if req.Terms != nil {
		customer.Terms = *req.Terms
	}
	if req.IsVATExempt != nil {
		customer.IsVATExempt = *req.IsVATExempt
	}
	if req.PreferredCurrency != nil {
		customer.PreferredCurrency = *req.PreferredCurrency
	}
	if req.Notes != nil {
		customer.Notes = req.Notes
	}
	if req.IsActive != nil {
		customer.IsActive = *req.IsActive
	}

	customer.UpdatedAt = time.Now()

	return s.customerRepo.Update(ctx, customer)
}

// DeleteCustomer deletes a customer
func (s *customerService) DeleteCustomer(ctx context.Context, id string) error {
	// Check if customer has orders
	orders, err := s.customerRepo.GetCustomerOrders(ctx, id, 1)
	if err != nil {
		return err
	}
	if len(orders) > 0 {
		return errors.New("cannot delete customer with existing orders")
	}

	customerID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid customer ID: %w", err)
	}
	return s.customerRepo.Delete(ctx, customerID)
}

// ListCustomers retrieves a paginated list of customers
func (s *customerService) ListCustomers(ctx context.Context, req *ListCustomersRequest) (*ListCustomersResponse, error) {
	filter := repositories.CustomerFilter{
		CompanyID:         req.CompanyID,
		Type:              stringValue(req.Type),
		IsActive:          req.IsActive,
		Industry:          req.Industry,
		Source:            stringValue(req.Source),
		StartDate:         req.CreatedAfter,
		EndDate:           req.CreatedBefore,
		Search:            stringValue(req.Search),
		SortBy:            stringValue(req.SortBy),
		SortOrder:         stringValue(req.SortOrder),
		Page:              req.Page,
		Limit:             req.Limit,
	}

	customers, err := s.customerRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	total, err := s.customerRepo.Count(ctx, filter)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	pagination := &repositories.Pagination{
		Page:       req.Page,
		Limit:      req.Limit,
		Total:      int64(total),
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
		HasPrev:    req.Page > 1,
	}

	return &ListCustomersResponse{
		Customers:  customers,
		Pagination: pagination,
	}, nil
}

// SearchCustomers searches customers by query
func (s *customerService) SearchCustomers(ctx context.Context, req *SearchCustomersRequest) (*SearchCustomersResponse, error) {
	filter := repositories.CustomerFilter{
		CompanyID:         req.CompanyID,
		Type:              stringValue(req.Type),
		IsActive:          req.IsActive,
		Source:            stringValue(req.Source),
		StartDate:         req.CreatedAfter,
		EndDate:           req.CreatedBefore,
		Search:            req.Query,
		SortBy:            stringValue(req.SortBy),
		SortOrder:         stringValue(req.SortOrder),
		Page:              req.Page,
		Limit:             req.Limit,
	}

	customers, err := s.customerRepo.Search(ctx, req.Query, filter)
	if err != nil {
		return nil, err
	}

	total, err := s.customerRepo.Count(ctx, filter)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	pagination := &repositories.Pagination{
		Page:       req.Page,
		Limit:      req.Limit,
		Total:      int64(total),
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
		HasPrev:    req.Page > 1,
	}

	return &SearchCustomersResponse{
		Customers:  customers,
		Pagination: pagination,
		TotalCount: int(total),
	}, nil
}

// CreateCompany creates a new company
func (s *customerService) CreateCompany(ctx context.Context, req *CreateCompanyRequest) (*entities.Company, error) {
	company := &entities.Company{
		ID:          uuid.New(),
		CompanyName: req.CompanyName,
		LegalName:   req.LegalName,
		TaxID:       req.TaxID,
		Industry:    stringValue(req.Industry),
		Website:     req.Website,
		Phone:       stringValue(req.Phone),
		Email:       req.Email,
		Address:     req.Address,
		City:        req.City,
		State:       req.State,
		Country:     req.Country,
		PostalCode:  req.PostalCode,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return s.companyRepo.Create(ctx, company)
}

// GetCompany retrieves a company by ID
func (s *customerService) GetCompany(ctx context.Context, id string) (*entities.Company, error) {
	companyID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid company ID: %w", err)
	}
	return s.companyRepo.GetByID(ctx, companyID)
}

// UpdateCompany updates an existing company
func (s *customerService) UpdateCompany(ctx context.Context, id string, req *UpdateCompanyRequest) (*entities.Company, error) {
	companyID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid company ID: %w", err)
	}
	company, err := s.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.CompanyName != nil {
		company.CompanyName = *req.CompanyName
	}
	if req.LegalName != nil {
		company.LegalName = *req.LegalName
	}
	if req.TaxID != nil {
		company.TaxID = *req.TaxID
	}
	if req.Industry != nil {
		company.Industry = *req.Industry
	}
	if req.Website != nil {
		company.Website = req.Website
	}
	if req.Phone != nil {
		company.Phone = *req.Phone
	}
	if req.Email != nil {
		company.Email = *req.Email
	}
	if req.Address != nil {
		company.Address = *req.Address
	}
	if req.City != nil {
		company.City = *req.City
	}
	if req.State != nil {
		company.State = *req.State
	}
	if req.Country != nil {
		company.Country = *req.Country
	}
	if req.PostalCode != nil {
		company.PostalCode = *req.PostalCode
	}
	if req.IsActive != nil {
		company.IsActive = *req.IsActive
	}

	company.UpdatedAt = time.Now()

	return s.companyRepo.Update(ctx, company)
}

// DeleteCompany deletes a company
func (s *customerService) DeleteCompany(ctx context.Context, id string) error {
	companyID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid company ID: %w", err)
	}

	// Check if company has customers
	customers, err := s.customerRepo.GetByCompanyID(ctx, companyID)
	if err != nil {
		return err
	}
	if len(customers) > 0 {
		return errors.New("cannot delete company with existing customers")
	}

	return s.companyRepo.Delete(ctx, companyID)
}

// ListCompanies retrieves a paginated list of companies
func (s *customerService) ListCompanies(ctx context.Context, req *ListCompaniesRequest) (*ListCompaniesResponse, error) {
	filter := repositories.CompanyFilter{
		Industry:      req.Industry,
		IsActive:      req.IsActive,
		StartDate:     req.CreatedAfter,
		EndDate:       req.CreatedBefore,
		Search:        stringValue(req.Search),
		SortBy:        stringValue(req.SortBy),
		SortOrder:     stringValue(req.SortOrder),
		Page:          req.Page,
		Limit:         req.Limit,
	}

	companies, err := s.companyRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	total, err := s.companyRepo.Count(ctx, filter)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	pagination := &repositories.Pagination{
		Page:       req.Page,
		Limit:      req.Limit,
		Total:      int64(total),
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
		HasPrev:    req.Page > 1,
	}

	return &ListCompaniesResponse{
		Companies:  companies,
		Pagination: pagination,
	}, nil
}

// UpdateCustomerCredit updates customer credit limit
func (s *customerService) UpdateCustomerCredit(ctx context.Context, id string, req *UpdateCustomerCreditRequest) (*entities.Customer, error) {
	customerID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid customer ID: %w", err)
	}
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	customer.CreditLimit = req.CreditLimit
	customer.UpdatedAt = time.Now()

	return s.customerRepo.Update(ctx, customer)
}

// GetCustomerCredit retrieves customer credit information
func (s *customerService) GetCustomerCredit(ctx context.Context, id string) (*CustomerCreditResponse, error) {
	customerID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid customer ID: %w", err)
	}
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	creditAvailable := customer.CreditLimit.Sub(customer.CreditUsed)
	var availablePercentage float64
	if !customer.CreditLimit.IsZero() {
		availablePercentage, _ = creditAvailable.Div(customer.CreditLimit).Float64()
	}

	return &CustomerCreditResponse{
		CustomerID:          customer.ID,
		CustomerCode:        customer.CustomerCode,
		CustomerName:        customer.FirstName + " " + customer.LastName,
		CreditLimit:         customer.CreditLimit,
		CreditUsed:          customer.CreditUsed,
		CreditAvailable:     creditAvailable,
		AvailablePercentage: availablePercentage,
		LastUpdated:         customer.UpdatedAt,
	}, nil
}

// AdjustCustomerCredit adjusts customer credit usage
func (s *customerService) AdjustCustomerCredit(ctx context.Context, id string, adjustment decimal.Decimal, reason *string) (*entities.Customer, error) {
	customerID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid customer ID: %w", err)
	}
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	newCreditUsed := customer.CreditUsed.Add(adjustment)
	if newCreditUsed.LessThan(decimal.Zero) {
		return nil, errors.New("credit used cannot be negative")
	}
	if newCreditUsed.GreaterThan(customer.CreditLimit) {
		return nil, errors.New("credit used cannot exceed credit limit")
	}

	customer.CreditUsed = newCreditUsed
	customer.UpdatedAt = time.Now()

	return s.customerRepo.Update(ctx, customer)
}

// GetCustomerStats retrieves customer statistics
func (s *customerService) GetCustomerStats(ctx context.Context, req *GetCustomerStatsRequest) (*CustomerStatsResponse, error) {
	startDate := time.Time{}
	endDate := time.Now()

	if req.StartDate != nil {
		startDate = *req.StartDate
	}
	if req.EndDate != nil {
		endDate = *req.EndDate
	}

	stats, err := s.customerRepo.GetStats(ctx, &startDate, &endDate)
	if err != nil {
		return nil, err
	}

	return &CustomerStatsResponse{
		TotalCustomers:           stats.TotalCustomers,
		ActiveCustomers:          stats.ActiveCustomers,
		InactiveCustomers:        stats.TotalCustomers - stats.ActiveCustomers,
		CustomersByType:          stats.CustomersByType,
		CustomersBySource:        stats.CustomersBySource,
		AverageCreditLimit:       decimal.Zero, // TODO: Calculate this
		TotalCreditLimit:         decimal.Zero, // TODO: Calculate this
		TotalCreditUsed:          decimal.Zero, // TODO: Calculate this
		NewCustomersThisMonth:    0, // TODO: Calculate this
		NewCustomersThisYear:     0, // TODO: Calculate this
	}, nil
}

// GetCompanyStats retrieves company statistics
func (s *customerService) GetCompanyStats(ctx context.Context, req *GetCompanyStatsRequest) (*CompanyStatsResponse, error) {
	startDate := time.Time{}
	endDate := time.Now()

	if req.StartDate != nil {
		startDate = *req.StartDate
	}
	if req.EndDate != nil {
		endDate = *req.EndDate
	}

	stats, err := s.companyRepo.GetStats(ctx, &startDate, &endDate)
	if err != nil {
		return nil, err
	}

	return &CompanyStatsResponse{
		TotalCompanies:             stats.TotalCompanies,
		ActiveCompanies:            stats.ActiveCompanies,
		InactiveCompanies:          stats.InactiveCompanies,
		CompaniesByIndustry:        stats.CompaniesByIndustry,
		AverageCustomersPerCompany: stats.AverageCustomersPerCompany,
		NewCompaniesThisMonth:      stats.NewCompaniesThisMonth,
		NewCompaniesThisYear:       stats.NewCompaniesThisYear,
	}, nil
}

// GetTopCustomersByRevenue retrieves top customers by revenue
func (s *customerService) GetTopCustomersByRevenue(ctx context.Context, req *GetTopCustomersRequest) ([]*entities.CustomerSummary, error) {
	startDate := time.Time{}
	endDate := time.Now()

	if req.StartDate != nil {
		startDate = *req.StartDate
	}
	if req.EndDate != nil {
		endDate = *req.EndDate
	}

	revenueStats, err := s.customerRepo.GetTopCustomersByRevenue(ctx, startDate, endDate, req.Limit)
	if err != nil {
		return nil, err
	}

	// Convert CustomerRevenueStats to CustomerSummary
	customers := make([]*entities.CustomerSummary, len(revenueStats))
	for i, stat := range revenueStats {
		customers[i] = &entities.CustomerSummary{
			ID:                stat.CustomerID,
			CustomerCode:      "", // We'll need to fetch this
			CustomerName:      stat.CustomerName,
			Email:             stat.CustomerEmail,
			CompanyName:       stat.CompanyName,
			Type:              "", // We'll need to fetch this
			TotalOrders:       stat.OrderCount,
			TotalRevenue:      stat.TotalRevenue,
			AverageOrderValue: stat.AverageOrderValue,
			LastOrderDate:     nil, // We'll need to fetch this
			IsActive:          true, // Default assumption
			CreatedAt:         time.Time{}, // We'll need to fetch this
		}
	}

	return customers, nil
}

// GetCustomerOrderHistory retrieves customer order history
func (s *customerService) GetCustomerOrderHistory(ctx context.Context, customerID string, limit int) ([]*entities.Order, error) {
	return s.customerRepo.GetCustomerOrders(ctx, customerID, limit)
}

// BulkUpdateCustomers performs bulk updates on customers
func (s *customerService) BulkUpdateCustomers(ctx context.Context, req *BulkUpdateCustomersRequest) (*BulkUpdateCustomersResponse, error) {
	var updatedCount, failedCount int
	var updatedCustomers []string
	var failedCustomers []FailedOperation

	for _, customerID := range req.CustomerIDs {
		customerUUID, err := uuid.Parse(customerID)
		if err != nil {
			failedCount++
			failedCustomers = append(failedCustomers, FailedOperation{
				ID:    customerID,
				Error: err.Error(),
			})
			continue
		}

		customer, err := s.customerRepo.GetByID(ctx, customerUUID)
		if err != nil {
			failedCount++
			failedCustomers = append(failedCustomers, FailedOperation{
				ID:    customerID,
				Error: err.Error(),
			})
			continue
		}

		// Apply updates
		for field, value := range req.Updates {
			switch field {
			case "is_active":
				if isActive, ok := value.(bool); ok {
					customer.IsActive = isActive
				}
			case "credit_limit":
				if creditLimit, ok := value.(decimal.Decimal); ok {
					customer.CreditLimit = creditLimit
				}
			case "is_vat_exempt":
				if isVATExempt, ok := value.(bool); ok {
					customer.IsVATExempt = isVATExempt
				}
			}
		}

		customer.UpdatedAt = time.Now()

		_, err = s.customerRepo.Update(ctx, customer)
		if err != nil {
			failedCount++
			failedCustomers = append(failedCustomers, FailedOperation{
				ID:    customerID,
				Error: err.Error(),
			})
		} else {
			updatedCount++
			updatedCustomers = append(updatedCustomers, customerID)
		}
	}

	return &BulkUpdateCustomersResponse{
		UpdatedCount:    updatedCount,
		FailedCount:     failedCount,
		UpdatedCustomers: updatedCustomers,
		FailedCustomers:  failedCustomers,
	}, nil
}

// BulkCreateCustomers performs bulk creation of customers
func (s *customerService) BulkCreateCustomers(ctx context.Context, req *BulkCreateCustomersRequest) (*BulkCreateCustomersResponse, error) {
	var createdCount, failedCount int
	var createdCustomers []string
	var failedCustomers []FailedOperation

	for _, customerReq := range req.Customers {
		customerCode, err := s.GenerateCustomerCode(ctx)
		if err != nil {
			failedCount++
			failedCustomers = append(failedCustomers, FailedOperation{
				ID:    "",
				Error: err.Error(),
			})
			continue
		}

		customer := &entities.Customer{
			ID:                uuid.New(),
			CustomerCode:      customerCode,
			CompanyID:         customerReq.CompanyID,
			Type:              customerReq.Type,
			FirstName:         customerReq.FirstName,
			LastName:          customerReq.LastName,
			Email:             stringValue(customerReq.Email),
			Phone:             stringValue(customerReq.Phone),
			Website:           customerReq.Website,
			CompanyName:       customerReq.CompanyName,
			TaxID:             customerReq.TaxID,
			Industry:          customerReq.Industry,
			CreditLimit:       customerReq.CreditLimit,
			CreditUsed:        decimal.Zero,
			Terms:             customerReq.Terms,
			IsActive:          true,
			IsVATExempt:       customerReq.IsVATExempt,
			PreferredCurrency: customerReq.PreferredCurrency,
			Notes:             customerReq.Notes,
			Source:            customerReq.Source,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		_, err = s.customerRepo.Create(ctx, customer)
		if err != nil {
			failedCount++
			failedCustomers = append(failedCustomers, FailedOperation{
				ID:    customer.ID.String(),
				Error: err.Error(),
			})
		} else {
			createdCount++
			createdCustomers = append(createdCustomers, customer.ID.String())
		}
	}

	return &BulkCreateCustomersResponse{
		CreatedCount:    createdCount,
		FailedCount:     failedCount,
		CreatedCustomers: createdCustomers,
		FailedCustomers:  failedCustomers,
	}, nil
}

// GenerateCustomerCode generates a unique customer code
func (s *customerService) GenerateCustomerCode(ctx context.Context) (string, error) {
	timestamp := time.Now().Format("20060102")
	random := uuid.New().String()[:8]
	return "CUST-" + timestamp + "-" + random, nil
}

// ValidateCustomerData validates customer data
func (s *customerService) ValidateCustomerData(ctx context.Context, req *CreateCustomerRequest) error {
	if req.Type == "BUSINESS" && req.CompanyName == nil {
		return errors.New("company name is required for business customers")
	}

	if req.Email != nil {
		// Check if email already exists
		existing, err := s.customerRepo.GetByEmail(ctx, *req.Email)
		if err == nil && existing != nil {
			return errors.New("email already exists")
		}
	}

	return nil
}

// MergeCustomers merges two customers
func (s *customerService) MergeCustomers(ctx context.Context, primaryID, secondaryID string) (*entities.Customer, error) {
	primaryUUID, err := uuid.Parse(primaryID)
	if err != nil {
		return nil, fmt.Errorf("invalid primary customer ID: %w", err)
	}
	primary, err := s.customerRepo.GetByID(ctx, primaryUUID)
	if err != nil {
		return nil, err
	}

	secondaryUUID, err := uuid.Parse(secondaryID)
	if err != nil {
		return nil, fmt.Errorf("invalid secondary customer ID: %w", err)
	}
	_, err = s.customerRepo.GetByID(ctx, secondaryUUID)
	if err != nil {
		return nil, err
	}

	// Transfer orders from secondary to primary
	err = s.customerRepo.TransferOrders(ctx, secondaryID, primaryID)
	if err != nil {
		return nil, err
	}

	// Delete secondary customer
	err = s.customerRepo.Delete(ctx, secondaryUUID)
	if err != nil {
		return nil, err
	}

	return primary, nil
}

// ActivateCustomer activates a customer
func (s *customerService) ActivateCustomer(ctx context.Context, id string) (*entities.Customer, error) {
	return s.UpdateCustomer(ctx, id, &UpdateCustomerRequest{
		IsActive: boolPtr(true),
	})
}

// DeactivateCustomer deactivates a customer
func (s *customerService) DeactivateCustomer(ctx context.Context, id string) (*entities.Customer, error) {
	return s.UpdateCustomer(ctx, id, &UpdateCustomerRequest{
		IsActive: boolPtr(false),
	})
}

// Helper functions

func stringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func boolPtr(b bool) *bool {
	return &b
}