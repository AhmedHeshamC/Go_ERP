package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"

	"erpgo/internal/application/services/customer"
	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/interfaces/http/dto"
)

// CustomerHandler handles customer HTTP requests
type CustomerHandler struct {
	customerService customer.Service
	logger          zerolog.Logger
}

// NewCustomerHandler creates a new customer handler
func NewCustomerHandler(customerService customer.Service, logger zerolog.Logger) *CustomerHandler {
	return &CustomerHandler{
		customerService: customerService,
		logger:          logger,
	}
}

// Customer CRUD Operations

// CreateCustomer creates a new customer
// @Summary Create customer
// @Description Create a new customer with automatic customer code generation
// @Tags customers
// @Accept json
// @Produce json
// @Param customer body dto.CreateCustomerRequest true "Customer data"
// @Success 201 {object} dto.CustomerResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/customers [post]
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req dto.CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid customer creation request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &customer.CreateCustomerRequest{
		CompanyID:         req.CompanyID,
		Type:              req.Type,
		FirstName:         req.FirstName,
		LastName:          req.LastName,
		Email:             req.Email,
		Phone:             req.Phone,
		Website:           req.Website,
		CompanyName:       req.CompanyName,
		TaxID:             req.TaxID,
		Industry:          req.Industry,
		CreditLimit:       req.CreditLimit,
		Terms:             req.Terms,
		IsVATExempt:       req.IsVATExempt,
		PreferredCurrency: req.PreferredCurrency,
		Notes:             req.Notes,
		Source:            req.Source,
	}

	createdCustomer, err := h.customerService.CreateCustomer(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create customer")
		handleCustomerError(c, err)
		return
	}

	response := h.customerToResponse(createdCustomer)
	c.JSON(http.StatusCreated, response)
}

// GetCustomer retrieves a customer by ID
// @Summary Get customer
// @Description Get a customer by its ID
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {object} dto.CustomerResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/customers/{id} [get]
func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Customer ID is required",
		})
		return
	}

	customer, err := h.customerService.GetCustomer(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", id).Msg("Failed to get customer")
		handleCustomerError(c, err)
		return
	}

	response := h.customerToResponse(customer)
	c.JSON(http.StatusOK, response)
}

// GetCustomerByCode retrieves a customer by customer code
// @Summary Get customer by code
// @Description Get a customer by its customer code
// @Tags customers
// @Accept json
// @Produce json
// @Param code path string true "Customer code"
// @Success 200 {object} dto.CustomerResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/customers/code/{code} [get]
func (h *CustomerHandler) GetCustomerByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Customer code is required",
		})
		return
	}

	customer, err := h.customerService.GetCustomerByCode(c, code)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_code", code).Msg("Failed to get customer by code")
		handleCustomerError(c, err)
		return
	}

	response := h.customerToResponse(customer)
	c.JSON(http.StatusOK, response)
}

// UpdateCustomer updates a customer
// @Summary Update customer
// @Description Update an existing customer
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Param customer body dto.UpdateCustomerRequest true "Customer data"
// @Success 200 {object} dto.CustomerResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/customers/{id} [put]
func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Customer ID is required",
		})
		return
	}

	var req dto.UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid customer update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &customer.UpdateCustomerRequest{
		CompanyID:         req.CompanyID,
		Type:              req.Type,
		FirstName:         req.FirstName,
		LastName:          req.LastName,
		Email:             req.Email,
		Phone:             req.Phone,
		Website:           req.Website,
		CompanyName:       req.CompanyName,
		TaxID:             req.TaxID,
		Industry:          req.Industry,
		CreditLimit:       req.CreditLimit,
		Terms:             req.Terms,
		IsVATExempt:       req.IsVATExempt,
		PreferredCurrency: req.PreferredCurrency,
		Notes:             req.Notes,
		IsActive:          req.IsActive,
	}

	updatedCustomer, err := h.customerService.UpdateCustomer(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", id).Msg("Failed to update customer")
		handleCustomerError(c, err)
		return
	}

	response := h.customerToResponse(updatedCustomer)
	c.JSON(http.StatusOK, response)
}

// DeleteCustomer deletes a customer
// @Summary Delete customer
// @Description Delete a customer
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 204
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/customers/{id} [delete]
func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Customer ID is required",
		})
		return
	}

	err := h.customerService.DeleteCustomer(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", id).Msg("Failed to delete customer")
		handleCustomerError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListCustomers retrieves a paginated list of customers
// @Summary List customers
// @Description Get a paginated list of customers with optional filtering
// @Tags customers
// @Accept json
// @Produce json
// @Param company_id query string false "Company ID"
// @Param type query string false "Customer type"
// @Param is_active query bool false "Filter by active status"
// @Param is_vat_exempt query bool false "Filter by VAT exempt status"
// @Param preferred_currency query string false "Preferred currency"
// @Param source query string false "Customer source"
// @Param industry query string false "Industry"
// @Param created_after query string false "Created after (ISO 8601)"
// @Param created_before query string false "Created before (ISO 8601)"
// @Param search query string false "Search term"
// @Param sort_by query string false "Sort field" default("created_at")
// @Param sort_order query string false "Sort order" Enums(asc,desc) default("desc")
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} dto.ListCustomersResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/customers [get]
func (h *CustomerHandler) ListCustomers(c *gin.Context) {
	var req dto.ListCustomersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid customer list request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Convert to service request
	serviceReq := &customer.ListCustomersRequest{
		CompanyID:         req.CompanyID,
		Type:              req.Type,
		IsActive:          req.IsActive,
		IsVATExempt:       req.IsVATExempt,
		PreferredCurrency: req.PreferredCurrency,
		Source:            req.Source,
		Industry:          req.Industry,
		CreatedAfter:      req.CreatedAfter,
		CreatedBefore:     req.CreatedBefore,
		Search:            req.Search,
		SortBy:            req.SortBy,
		SortOrder:         req.SortOrder,
		Page:              req.Page,
		Limit:             req.Limit,
	}

	result, err := h.customerService.ListCustomers(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list customers")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve customers",
		})
		return
	}

	// Convert to response DTO
	customers := make([]*dto.CustomerResponse, len(result.Customers))
	for i, cust := range result.Customers {
		customers[i] = h.customerToResponse(cust)
	}

	response := &dto.ListCustomersResponse{
		Customers: customers,
		Pagination: &dto.Pagination{
			Page:       result.Pagination.Page,
			Limit:      result.Pagination.Limit,
			Total:      int(result.Pagination.Total),
			TotalPages: result.Pagination.TotalPages,
			HasNext:    result.Pagination.HasNext,
			HasPrev:    result.Pagination.HasPrev,
		},
	}

	c.JSON(http.StatusOK, response)
}

// SearchCustomers searches customers by query
// @Summary Search customers
// @Description Search customers by query string
// @Tags customers
// @Accept json
// @Produce json
// @Param query query string true "Search query"
// @Param company_id query string false "Company ID"
// @Param type query string false "Customer type"
// @Param is_active query bool false "Filter by active status"
// @Param preferred_currency query string false "Preferred currency"
// @Param source query string false "Customer source"
// @Param created_after query string false "Created after (ISO 8601)"
// @Param created_before query string false "Created before (ISO 8601)"
// @Param search_fields query []string false "Search fields"
// @Param sort_by query string false "Sort field" default("created_at")
// @Param sort_order query string false "Sort order" Enums(asc,desc) default("desc")
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} dto.SearchCustomersResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/customers/search [get]
func (h *CustomerHandler) SearchCustomers(c *gin.Context) {
	var req dto.SearchCustomersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid customer search request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Convert to service request
	serviceReq := &customer.SearchCustomersRequest{
		Query:             req.Query,
		CompanyID:         req.CompanyID,
		Type:              req.Type,
		IsActive:          req.IsActive,
		PreferredCurrency: req.PreferredCurrency,
		Source:            req.Source,
		CreatedAfter:      req.CreatedAfter,
		CreatedBefore:     req.CreatedBefore,
		SearchFields:      req.SearchFields,
		SortBy:            req.SortBy,
		SortOrder:         req.SortOrder,
		Page:              req.Page,
		Limit:             req.Limit,
	}

	result, err := h.customerService.SearchCustomers(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to search customers")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to search customers",
		})
		return
	}

	// Convert to response DTO
	customers := make([]*dto.CustomerResponse, len(result.Customers))
	for i, cust := range result.Customers {
		customers[i] = h.customerToResponse(cust)
	}

	response := &dto.SearchCustomersResponse{
		Customers: customers,
		Pagination: &dto.Pagination{
			Page:       result.Pagination.Page,
			Limit:      result.Pagination.Limit,
			Total:      int(result.Pagination.Total),
			TotalPages: result.Pagination.TotalPages,
			HasNext:    result.Pagination.HasNext,
			HasPrev:    result.Pagination.HasPrev,
		},
		TotalCount: result.TotalCount,
	}

	c.JSON(http.StatusOK, response)
}

// Customer Status Management

// ActivateCustomer activates a customer
// @Summary Activate customer
// @Description Activate a customer
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {object} dto.CustomerResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/customers/{id}/activate [post]
func (h *CustomerHandler) ActivateCustomer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Customer ID is required",
		})
		return
	}

	activatedCustomer, err := h.customerService.ActivateCustomer(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", id).Msg("Failed to activate customer")
		handleCustomerError(c, err)
		return
	}

	response := h.customerToResponse(activatedCustomer)
	c.JSON(http.StatusOK, response)
}

// DeactivateCustomer deactivates a customer
// @Summary Deactivate customer
// @Description Deactivate a customer
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {object} dto.CustomerResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/customers/{id}/deactivate [post]
func (h *CustomerHandler) DeactivateCustomer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Customer ID is required",
		})
		return
	}

	deactivatedCustomer, err := h.customerService.DeactivateCustomer(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", id).Msg("Failed to deactivate customer")
		handleCustomerError(c, err)
		return
	}

	response := h.customerToResponse(deactivatedCustomer)
	c.JSON(http.StatusOK, response)
}

// Customer Credit Management

// GetCustomerCredit retrieves customer credit information
// @Summary Get customer credit
// @Description Get detailed credit information for a customer
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {object} dto.CustomerCreditResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/customers/{id}/credit [get]
func (h *CustomerHandler) GetCustomerCredit(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Customer ID is required",
		})
		return
	}

	creditInfo, err := h.customerService.GetCustomerCredit(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", id).Msg("Failed to get customer credit")
		handleCustomerError(c, err)
		return
	}

	c.JSON(http.StatusOK, creditInfo)
}

// UpdateCustomerCredit updates customer credit information
// @Summary Update customer credit
// @Description Update credit limit for a customer
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Param credit body dto.UpdateCustomerCreditRequest true "Credit update data"
// @Success 200 {object} dto.CustomerResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/customers/{id}/credit [put]
func (h *CustomerHandler) UpdateCustomerCredit(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Customer ID is required",
		})
		return
	}

	var req dto.UpdateCustomerCreditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid credit update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	serviceReq := &customer.UpdateCustomerCreditRequest{
		CreditLimit: req.CreditLimit,
		Adjustment:  req.Adjustment,
		Reason:      req.Reason,
	}

	updatedCustomer, err := h.customerService.UpdateCustomerCredit(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", id).Msg("Failed to update customer credit")
		handleCustomerError(c, err)
		return
	}

	response := h.customerToResponse(updatedCustomer)
	c.JSON(http.StatusOK, response)
}

// Company Management

// CreateCompany creates a new company
// @Summary Create company
// @Description Create a new company
// @Tags companies
// @Accept json
// @Produce json
// @Param company body dto.CreateCompanyRequest true "Company data"
// @Success 201 {object} dto.CompanyResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/companies [post]
func (h *CustomerHandler) CreateCompany(c *gin.Context) {
	var req dto.CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid company creation request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &customer.CreateCompanyRequest{
		CompanyName: req.CompanyName,
		LegalName:   req.LegalName,
		TaxID:       req.TaxID,
		Industry:    req.Industry,
		Website:     req.Website,
		Phone:       req.Phone,
		Email:       req.Email,
		Address:     req.Address,
		City:        req.City,
		State:       req.State,
		Country:     req.Country,
		PostalCode:  req.PostalCode,
	}

	createdCompany, err := h.customerService.CreateCompany(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create company")
		handleCustomerError(c, err)
		return
	}

	response := h.companyToResponse(createdCompany)
	c.JSON(http.StatusCreated, response)
}

// GetCompany retrieves a company by ID
// @Summary Get company
// @Description Get a company by its ID
// @Tags companies
// @Accept json
// @Produce json
// @Param id path string true "Company ID"
// @Success 200 {object} dto.CompanyResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/companies/{id} [get]
func (h *CustomerHandler) GetCompany(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Company ID is required",
		})
		return
	}

	company, err := h.customerService.GetCompany(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("company_id", id).Msg("Failed to get company")
		handleCustomerError(c, err)
		return
	}

	response := h.companyToResponse(company)
	c.JSON(http.StatusOK, response)
}

// UpdateCompany updates a company
// @Summary Update company
// @Description Update an existing company
// @Tags companies
// @Accept json
// @Produce json
// @Param id path string true "Company ID"
// @Param company body dto.UpdateCompanyRequest true "Company data"
// @Success 200 {object} dto.CompanyResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/companies/{id} [put]
func (h *CustomerHandler) UpdateCompany(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Company ID is required",
		})
		return
	}

	var req dto.UpdateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid company update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &customer.UpdateCompanyRequest{
		CompanyName: req.CompanyName,
		LegalName:   req.LegalName,
		TaxID:       req.TaxID,
		Industry:    req.Industry,
		Website:     req.Website,
		Phone:       req.Phone,
		Email:       req.Email,
		Address:     req.Address,
		City:        req.City,
		State:       req.State,
		Country:     req.Country,
		PostalCode:  req.PostalCode,
		IsActive:    req.IsActive,
	}

	updatedCompany, err := h.customerService.UpdateCompany(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("company_id", id).Msg("Failed to update company")
		handleCustomerError(c, err)
		return
	}

	response := h.companyToResponse(updatedCompany)
	c.JSON(http.StatusOK, response)
}

// DeleteCompany deletes a company
// @Summary Delete company
// @Description Delete a company
// @Tags companies
// @Accept json
// @Produce json
// @Param id path string true "Company ID"
// @Success 204
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/companies/{id} [delete]
func (h *CustomerHandler) DeleteCompany(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Company ID is required",
		})
		return
	}

	err := h.customerService.DeleteCompany(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("company_id", id).Msg("Failed to delete company")
		handleCustomerError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListCompanies retrieves a paginated list of companies
// @Summary List companies
// @Description Get a paginated list of companies with optional filtering
// @Tags companies
// @Accept json
// @Produce json
// @Param industry query string false "Industry"
// @Param is_active query bool false "Filter by active status"
// @Param created_after query string false "Created after (ISO 8601)"
// @Param created_before query string false "Created before (ISO 8601)"
// @Param search query string false "Search term"
// @Param sort_by query string false "Sort field" default("created_at")
// @Param sort_order query string false "Sort order" Enums(asc,desc) default("desc")
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} dto.ListCompaniesResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/companies [get]
func (h *CustomerHandler) ListCompanies(c *gin.Context) {
	var req dto.ListCompaniesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid company list request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Convert to service request
	serviceReq := &customer.ListCompaniesRequest{
		Industry:      req.Industry,
		IsActive:      req.IsActive,
		CreatedAfter:  req.CreatedAfter,
		CreatedBefore: req.CreatedBefore,
		Search:        req.Search,
		SortBy:        req.SortBy,
		SortOrder:     req.SortOrder,
		Page:          req.Page,
		Limit:         req.Limit,
	}

	result, err := h.customerService.ListCompanies(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list companies")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve companies",
		})
		return
	}

	// Convert to response DTO
	companies := make([]*dto.CompanyResponse, len(result.Companies))
	for i, comp := range result.Companies {
		companies[i] = h.companyToResponse(comp)
	}

	response := &dto.ListCompaniesResponse{
		Companies: companies,
		Pagination: &dto.Pagination{
			Page:       result.Pagination.Page,
			Limit:      result.Pagination.Limit,
			Total:      int(result.Pagination.Total),
			TotalPages: result.Pagination.TotalPages,
			HasNext:    result.Pagination.HasNext,
			HasPrev:    result.Pagination.HasPrev,
		},
	}

	c.JSON(http.StatusOK, response)
}

// Helper Methods

// customerToResponse converts a customer entity to a response DTO
func (h *CustomerHandler) customerToResponse(cust *entities.Customer) *dto.CustomerResponse {
	creditAvailable := cust.CreditLimit.Sub(cust.CreditUsed)

	return &dto.CustomerResponse{
		ID:                cust.ID,
		CustomerCode:      cust.CustomerCode,
		CompanyID:         cust.CompanyID,
		Type:              cust.Type,
		FirstName:         cust.FirstName,
		LastName:          cust.LastName,
		FullName:          cust.FirstName + " " + cust.LastName,
		Email:             stringPtr(cust.Email),
		Phone:             stringPtr(cust.Phone),
		Website:           cust.Website,
		CompanyName:       cust.CompanyName,
		TaxID:             cust.TaxID,
		Industry:          cust.Industry,
		CreditLimit:       cust.CreditLimit,
		CreditUsed:        cust.CreditUsed,
		CreditAvailable:   creditAvailable,
		Terms:             cust.Terms,
		IsActive:          cust.IsActive,
		IsVATExempt:       cust.IsVATExempt,
		PreferredCurrency: cust.PreferredCurrency,
		Notes:             cust.Notes,
		Source:            cust.Source,
		OrderCount:        0, // TODO: Get from service
		TotalOrdersValue:  decimal.Zero, // TODO: Get from service
		LastOrderDate:     nil, // TODO: Get from service
		CreatedAt:         cust.CreatedAt,
		UpdatedAt:         cust.UpdatedAt,
	}
}

// companyToResponse converts a company entity to a response DTO
func (h *CustomerHandler) companyToResponse(comp *entities.Company) *dto.CompanyResponse {
	return &dto.CompanyResponse{
		ID:            comp.ID,
		CompanyName:   comp.CompanyName,
		LegalName:     comp.LegalName,
		TaxID:         comp.TaxID,
		Industry:      stringPtr(comp.Industry),
		Website:       comp.Website,
		Phone:         stringPtr(comp.Phone),
		Email:         comp.Email,
		Address:       comp.Address,
		City:          comp.City,
		State:         comp.State,
		Country:       comp.Country,
		PostalCode:    comp.PostalCode,
		IsActive:      comp.IsActive,
		CustomerCount: 0, // TODO: Get from service
		CreatedAt:     comp.CreatedAt,
		UpdatedAt:     comp.UpdatedAt,
	}
}

// handleCustomerError handles customer service errors
func handleCustomerError(c *gin.Context, err error) {
	switch {
	case err.Error() == "customer not found":
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Customer not found",
			Details: err.Error(),
		})
	case err.Error() == "company not found":
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Company not found",
			Details: err.Error(),
		})
	case err.Error() == "customer already exists":
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Customer already exists",
			Details: err.Error(),
		})
	case err.Error() == "company already exists":
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Company already exists",
			Details: err.Error(),
		})
	case err.Error() == "email already exists":
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Email already exists",
			Details: err.Error(),
		})
	case err.Error() == "cannot delete customer with existing orders":
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Cannot delete customer with existing orders",
			Details: err.Error(),
		})
	case err.Error() == "cannot delete company with existing customers":
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Cannot delete company with existing customers",
			Details: err.Error(),
		})
	case err.Error() == "company name is required for business customers":
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Company name is required for business customers",
			Details: err.Error(),
		})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Details: err.Error(),
		})
	}
}

// stringPtr converts a string to a pointer
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}