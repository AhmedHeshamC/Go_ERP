package repositories

import (
	"context"
	"erpgo/internal/domain/customers/entities"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CustomerRepository defines the interface for customer data access
type CustomerRepository interface {
	// Create adds a new customer to the repository
	Create(ctx context.Context, customer *entities.Customer) error

	// GetByID retrieves a customer by ID
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Customer, error)

	// GetByEmail retrieves a customer by email
	GetByEmail(ctx context.Context, email string) (*entities.Customer, error)

	// GetByCustomerCode retrieves a customer by customer code
	GetByCustomerCode(ctx context.Context, code string) (*entities.Customer, error)

	// Update updates an existing customer in the repository
	Update(ctx context.Context, customer *entities.Customer) error

	// Delete soft deletes a customer from the repository
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves customers based on filter criteria
	List(ctx context.Context, filter *CustomerFilter) ([]*entities.Customer, error)

	// Count returns the total number of customers based on filter criteria
	Count(ctx context.Context, filter *CustomerFilter) (int, error)

	// Exists checks if a customer with the given email exists
	Exists(ctx context.Context, email string) (bool, error)

	// UpdateStatus updates the status of a customer
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.CustomerStatus) error

	// UpdateCredit updates the credit information of a customer
	UpdateCredit(ctx context.Context, id uuid.UUID, creditUsed decimal.Decimal) error
}

// CustomerFilter defines filter criteria for customer queries
type CustomerFilter struct {
	// Search term to search across name, email, and company name
	Search *string

	// Customer type filter
	CustomerType *entities.CustomerType

	// Customer status filter
	Status *entities.CustomerStatus

	// Company filter
	CompanyID *uuid.UUID

	// Source filter
	Source *string

	// Industry filter
	Industry *string

	// VAT exemption filter
	IsVATExempt *bool

	// Credit limit range
	CreditMin *decimal.Decimal
	CreditMax *decimal.Decimal

	// Creation date range
	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	// Last updated date range
	UpdatedAfter  *time.Time
	UpdatedBefore *time.Time

	// Active status filter
	Active *bool

	// Pagination
	Page     *int
	PageSize *int

	// Sort options
	SortBy    *string
	SortOrder *string
}

// CustomerRepositoryWithTx extends the CustomerRepository interface with transaction support
type CustomerRepositoryWithTx interface {
	CustomerRepository

	// WithTx returns a new repository instance bound to the given transaction
	WithTx(ctx context.Context) CustomerRepository
}
