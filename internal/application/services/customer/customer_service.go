package customer

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"erpgo/internal/domain/customers/entities"
	"erpgo/internal/domain/customers/repositories"
	"erpgo/pkg/errors"
)

// Service defines the customer service interface
type Service interface {
	CreateCustomer(ctx context.Context, req *CreateCustomerRequest) (*entities.Customer, error)
	GetCustomer(ctx context.Context, id uuid.UUID) (*entities.Customer, error)
	GetCustomerByEmail(ctx context.Context, email string) (*entities.Customer, error)
	UpdateCustomer(ctx context.Context, id uuid.UUID, req *UpdateCustomerRequest) (*entities.Customer, error)
	DeleteCustomer(ctx context.Context, id uuid.UUID) error
	ListCustomers(ctx context.Context, filter *repositories.CustomerFilter) ([]*entities.Customer, error)
	ActivateCustomer(ctx context.Context, id uuid.UUID) error
	DeactivateCustomer(ctx context.Context, id uuid.UUID) error
	SuspendCustomer(ctx context.Context, id uuid.UUID) error
}

// CustomerService implements the customer service
type CustomerService struct {
	repo repositories.CustomerRepository
}

// NewCustomerService creates a new customer service
func NewCustomerService(repo repositories.CustomerRepository) Service {
	return &CustomerService{
		repo: repo,
	}
}

// CreateCustomerRequest represents a request to create a customer
type CreateCustomerRequest struct {
	CompanyID         *uuid.UUID      `json:"company_id,omitempty"`
	Type              string          `json:"type" binding:"required"`
	FirstName         string          `json:"first_name" binding:"required"`
	LastName          string          `json:"last_name" binding:"required"`
	Email             *string         `json:"email,omitempty"`
	Phone             *string         `json:"phone,omitempty"`
	Website           *string         `json:"website,omitempty"`
	CompanyName       *string         `json:"company_name,omitempty"`
	TaxID             *string         `json:"tax_id,omitempty"`
	Industry          *string         `json:"industry,omitempty"`
	CreditLimit       decimal.Decimal `json:"credit_limit" binding:"required"`
	Terms             string          `json:"terms" binding:"required"`
	IsVATExempt       bool            `json:"is_vat_exempt"`
	PreferredCurrency string          `json:"preferred_currency" binding:"required"`
	Notes             *string         `json:"notes,omitempty"`
	Source            string          `json:"source" binding:"required"`
}

// UpdateCustomerRequest represents a request to update a customer
type UpdateCustomerRequest struct {
	Name         *string                  `json:"name"`
	Email        *string                  `json:"email"`
	Phone        *string                  `json:"phone"`
	Address      *string                  `json:"address"`
	CustomerType *entities.CustomerType   `json:"customer_type"`
	Status       *entities.CustomerStatus `json:"status"`
	TaxID        *string                  `json:"tax_id"`
}

// CreateCustomer creates a new customer
func (s *CustomerService) CreateCustomer(ctx context.Context, req *CreateCustomerRequest) (*entities.Customer, error) {
	// Check if customer with email already exists
	if req.Email != nil {
		exists, err := s.repo.Exists(ctx, *req.Email)
		if err != nil {
			return nil, err
		}

		if exists {
			return nil, errors.NewConflictError("customer with this email already exists")
		}
	}

	// Create customer entity with all required fields
	customer := s.createCustomerEntity(req)

	// Validate the customer after all fields are set
	if err := customer.Validate(); err != nil {
		return nil, err
	}

	// Save to repository
	if err := s.repo.Create(ctx, customer); err != nil {
		return nil, err
	}

	return customer, nil
}

// GetCustomer retrieves a customer by ID
func (s *CustomerService) GetCustomer(ctx context.Context, id uuid.UUID) (*entities.Customer, error) {
	return s.repo.GetByID(ctx, id)
}

// GetCustomerByEmail retrieves a customer by email
func (s *CustomerService) GetCustomerByEmail(ctx context.Context, email string) (*entities.Customer, error) {
	return s.repo.GetByEmail(ctx, email)
}

// UpdateCustomer updates an existing customer
func (s *CustomerService) UpdateCustomer(ctx context.Context, id uuid.UUID, req *UpdateCustomerRequest) (*entities.Customer, error) {
	// Get existing customer
	customer, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		customer.Name = strings.TrimSpace(*req.Name)
	}

	if req.Email != nil {
		newEmail := strings.TrimSpace(strings.ToLower(*req.Email))
		if customer.Email == nil || newEmail != *customer.Email {
			// Check if new email already exists
			exists, err := s.repo.Exists(ctx, newEmail)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, errors.NewConflictError("customer with this email already exists")
			}
			customer.Email = &newEmail
		}
	}

	if req.Phone != nil {
		phoneStr := strings.TrimSpace(*req.Phone)
		customer.Phone = &phoneStr
	}

	if req.Address != nil {
		customer.Address = strings.TrimSpace(*req.Address)
	}

	if req.CustomerType != nil {
		customer.CustomerType = *req.CustomerType
	}

	if req.Status != nil {
		customer.Status = *req.Status
	}

	if req.TaxID != nil {
		taxIDStr := strings.TrimSpace(*req.TaxID)
		customer.TaxID = &taxIDStr
	}

	// Validate updated customer
	if err := customer.Validate(); err != nil {
		return nil, err
	}

	// Save updates
	if err := s.repo.Update(ctx, customer); err != nil {
		return nil, err
	}

	return customer, nil
}

// DeleteCustomer soft deletes a customer
func (s *CustomerService) DeleteCustomer(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// ListCustomers retrieves customers based on filter criteria
func (s *CustomerService) ListCustomers(ctx context.Context, filter *repositories.CustomerFilter) ([]*entities.Customer, error) {
	return s.repo.List(ctx, filter)
}

// ActivateCustomer activates a customer
func (s *CustomerService) ActivateCustomer(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, id, entities.CustomerStatusActive)
}

// DeactivateCustomer deactivates a customer
func (s *CustomerService) DeactivateCustomer(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, id, entities.CustomerStatusInactive)
}

// SuspendCustomer suspends a customer
func (s *CustomerService) SuspendCustomer(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, id, entities.CustomerStatusSuspended)
}

// createCustomerEntity creates a customer entity from request without validation
func (s *CustomerService) createCustomerEntity(req *CreateCustomerRequest) *entities.Customer {
	var emailPtr, phonePtr *string
	if req.Email != nil {
		emailStr := strings.TrimSpace(strings.ToLower(*req.Email))
		emailPtr = &emailStr
	}
	if req.Phone != nil {
		phoneStr := strings.TrimSpace(*req.Phone)
		phonePtr = &phoneStr
	}

	// Handle name for different customer types
	var customerName string
	var firstName, lastName string
	if req.Type == string(entities.CustomerTypeBusiness) && req.CompanyName != nil {
		// For business customers, use company name if provided
		customerName = *req.CompanyName
		firstName = ""
		lastName = ""
	} else {
		// For individual customers, combine first and last name
		customerName = req.FirstName + " " + req.LastName
		firstName = req.FirstName
		lastName = req.LastName
	}

	customer := &entities.Customer{
		ID:                uuid.New(),
		CustomerCode:      generateCustomerCode(),
		Type:              req.Type,
		FirstName:         firstName,
		LastName:          lastName,
		Name:              strings.TrimSpace(customerName),
		Email:             emailPtr,
		Phone:             phonePtr,
		CustomerType:      entities.CustomerType(req.Type),
		Status:            entities.CustomerStatusActive,
		CreditLimit:       req.CreditLimit,
		CreditUsed:        decimal.Zero,
		Terms:             req.Terms,
		Active:            true,
		IsVATExempt:       req.IsVATExempt,
		PreferredCurrency: req.PreferredCurrency,
		Source:            req.Source,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}

	// Set optional fields
	if req.CompanyID != nil {
		customer.CompanyID = req.CompanyID
	}
	if req.Website != nil {
		customer.Website = req.Website
	}
	if req.CompanyName != nil {
		customer.CompanyName = req.CompanyName
	}
	if req.TaxID != nil {
		taxIDStr := strings.TrimSpace(*req.TaxID)
		customer.TaxID = &taxIDStr
	}
	if req.Industry != nil {
		customer.Industry = req.Industry
	}
	if req.Notes != nil {
		customer.Notes = req.Notes
	}

	return customer
}

// generateCustomerCode generates a unique customer code
func generateCustomerCode() string {
	return "CUST-" + uuid.New().String()[:8]
}
