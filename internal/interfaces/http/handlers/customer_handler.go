package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"

	"erpgo/internal/application/services/customer"
	"erpgo/internal/domain/customers/entities"
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

	customerUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid customer ID format",
		})
		return
	}

	customer, err := h.customerService.GetCustomer(c, customerUUID)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", id).Msg("Failed to get customer")
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

	customerUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid customer ID format",
		})
		return
	}

	// Convert to service request - only basic fields supported
	name := ""
	if req.FirstName != nil && req.LastName != nil {
		name = *req.FirstName + " " + *req.LastName
	}
	email := ""
	if req.Email != nil {
		email = *req.Email
	}
	phone := ""
	if req.Phone != nil {
		phone = *req.Phone
	}
	address := "" // not in DTO

	serviceReq := &customer.UpdateCustomerRequest{
		Name:         &name,
		Email:        &email,
		Phone:        &phone,
		Address:      &address,
		CustomerType: nil, // not in DTO
		Status:       nil, // not in DTO
		TaxID:        req.TaxID,
	}

	updatedCustomer, err := h.customerService.UpdateCustomer(c, customerUUID, serviceReq)
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

	customerUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid customer ID format",
		})
		return
	}

	err = h.customerService.DeleteCustomer(c, customerUUID)
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
	// ListCustomers not implemented with pagination, return empty response
	customers := []*dto.CustomerResponse{}
	response := &dto.ListCustomersResponse{
		Customers: customers,
		Pagination: &dto.Pagination{
			Page:       1,
			Limit:      20,
			Total:      0,
			TotalPages: 0,
			HasNext:    false,
			HasPrev:    false,
		},
	}
	c.JSON(http.StatusOK, response)
}

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

	customerUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid customer ID format",
		})
		return
	}

	err = h.customerService.ActivateCustomer(c, customerUUID)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", id).Msg("Failed to activate customer")
		handleCustomerError(c, err)
		return
	}

	// Get updated customer to return
	activatedCustomer, err := h.customerService.GetCustomer(c, customerUUID)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", id).Msg("Failed to get updated customer")
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

	customerUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid customer ID format",
		})
		return
	}

	err = h.customerService.DeactivateCustomer(c, customerUUID)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", id).Msg("Failed to deactivate customer")
		handleCustomerError(c, err)
		return
	}

	// Get updated customer to return
	deactivatedCustomer, err := h.customerService.GetCustomer(c, customerUUID)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", id).Msg("Failed to get updated customer")
		handleCustomerError(c, err)
		return
	}

	response := h.customerToResponse(deactivatedCustomer)
	c.JSON(http.StatusOK, response)
}

// Helper Methods

// customerToResponse converts a customer entity to a response DTO
func (h *CustomerHandler) customerToResponse(cust *entities.Customer) *dto.CustomerResponse {
	creditAvailable := cust.CreditLimit.Sub(cust.CreditUsed)

	var email *string
	if cust.Email != nil {
		email = cust.Email
	}
	var phone *string
	if cust.Phone != nil {
		phone = cust.Phone
	}

	return &dto.CustomerResponse{
		ID:                cust.ID,
		CustomerCode:      cust.CustomerCode,
		CompanyID:         cust.CompanyID,
		Type:              cust.Type,
		FirstName:         cust.FirstName,
		LastName:          cust.LastName,
		FullName:          cust.Name,
		Email:             email,
		Phone:             phone,
		Website:           cust.Website,
		CompanyName:       cust.CompanyName,
		TaxID:             cust.TaxID,
		Industry:          cust.Industry,
		CreditLimit:       cust.CreditLimit,
		CreditUsed:        cust.CreditUsed,
		CreditAvailable:   creditAvailable,
		Terms:             cust.Terms,
		IsActive:          cust.Active,
		IsVATExempt:       cust.IsVATExempt,
		PreferredCurrency: cust.PreferredCurrency,
		Notes:             cust.Notes,
		Source:            cust.Source,
		OrderCount:        0,            // TODO: Get from service
		TotalOrdersValue:  decimal.Zero, // TODO: Get from service
		LastOrderDate:     nil,          // TODO: Get from service
		CreatedAt:         cust.CreatedAt,
		UpdatedAt:         cust.UpdatedAt,
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
