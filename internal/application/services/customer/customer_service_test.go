package customer

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"erpgo/internal/domain/customers/entities"
	"erpgo/internal/domain/customers/repositories"
)

// MockCustomerRepository is a mock implementation of the CustomerRepository interface
type MockCustomerRepository struct {
	customers map[uuid.UUID]*entities.Customer
	byEmail   map[string]*entities.Customer
	byCode    map[string]*entities.Customer
}

// NewMockCustomerRepository creates a new mock customer repository
func NewMockCustomerRepository() *MockCustomerRepository {
	return &MockCustomerRepository{
		customers: make(map[uuid.UUID]*entities.Customer),
		byEmail:   make(map[string]*entities.Customer),
		byCode:    make(map[string]*entities.Customer),
	}
}

// Create adds a new customer to the repository
func (m *MockCustomerRepository) Create(ctx context.Context, customer *entities.Customer) error {
	if _, exists := m.customers[customer.ID]; exists {
		return errors.New("customer already exists")
	}

	if customer.Email != nil {
		if _, exists := m.byEmail[*customer.Email]; exists {
			return errors.New("customer with this email already exists")
		}
	}

	m.customers[customer.ID] = customer
	if customer.Email != nil {
		m.byEmail[*customer.Email] = customer
	}
	m.byCode[customer.CustomerCode] = customer

	return nil
}

// GetByID retrieves a customer by ID
func (m *MockCustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Customer, error) {
	customer, exists := m.customers[id]
	if !exists {
		return nil, errors.New("customer not found")
	}
	return customer, nil
}

// GetByEmail retrieves a customer by email
func (m *MockCustomerRepository) GetByEmail(ctx context.Context, email string) (*entities.Customer, error) {
	customer, exists := m.byEmail[email]
	if !exists {
		return nil, errors.New("customer not found")
	}
	return customer, nil
}

// GetByCustomerCode retrieves a customer by customer code
func (m *MockCustomerRepository) GetByCustomerCode(ctx context.Context, code string) (*entities.Customer, error) {
	customer, exists := m.byCode[code]
	if !exists {
		return nil, errors.New("customer not found")
	}
	return customer, nil
}

// Update updates an existing customer in the repository
func (m *MockCustomerRepository) Update(ctx context.Context, customer *entities.Customer) error {
	if _, exists := m.customers[customer.ID]; !exists {
		return errors.New("customer not found")
	}

	// Update email index if email changed
	if customer.Email != nil {
		if oldCustomer := m.customers[customer.ID]; oldCustomer.Email == nil || *oldCustomer.Email != *customer.Email {
			// Remove old email index if it exists
			if oldCustomer.Email != nil {
				delete(m.byEmail, *oldCustomer.Email)
			}
			// Add new email index
			m.byEmail[*customer.Email] = customer
		}
	}

	m.customers[customer.ID] = customer
	m.byCode[customer.CustomerCode] = customer

	return nil
}

// Delete soft deletes a customer from the repository
func (m *MockCustomerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if _, exists := m.customers[id]; !exists {
		return errors.New("customer not found")
	}

	customer := m.customers[id]
	delete(m.customers, id)
	if customer.Email != nil {
		delete(m.byEmail, *customer.Email)
	}
	delete(m.byCode, customer.CustomerCode)

	return nil
}

// List retrieves customers based on filter criteria
func (m *MockCustomerRepository) List(ctx context.Context, filter *repositories.CustomerFilter) ([]*entities.Customer, error) {
	var result []*entities.Customer

	for _, customer := range m.customers {
		if m.matchesFilter(customer, filter) {
			result = append(result, customer)
		}
	}

	return result, nil
}

// Count returns the total number of customers based on filter criteria
func (m *MockCustomerRepository) Count(ctx context.Context, filter *repositories.CustomerFilter) (int, error) {
	count := 0
	for _, customer := range m.customers {
		if m.matchesFilter(customer, filter) {
			count++
		}
	}
	return count, nil
}

// Exists checks if a customer with the given email exists
func (m *MockCustomerRepository) Exists(ctx context.Context, email string) (bool, error) {
	_, exists := m.byEmail[email]
	return exists, nil
}

// UpdateStatus updates the status of a customer
func (m *MockCustomerRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.CustomerStatus) error {
	customer, exists := m.customers[id]
	if !exists {
		return errors.New("customer not found")
	}

	customer.Status = status
	return nil
}

// UpdateCredit updates the credit information of a customer
func (m *MockCustomerRepository) UpdateCredit(ctx context.Context, id uuid.UUID, creditUsed decimal.Decimal) error {
	customer, exists := m.customers[id]
	if !exists {
		return errors.New("customer not found")
	}

	customer.CreditUsed = creditUsed
	return nil
}

// WithTx returns a new repository instance bound to the given transaction (mock implementation)
func (m *MockCustomerRepository) WithTx(ctx context.Context) repositories.CustomerRepository {
	return m
}

// matchesFilter checks if a customer matches the filter criteria (simplified)
func (m *MockCustomerRepository) matchesFilter(customer *entities.Customer, filter *repositories.CustomerFilter) bool {
	if filter == nil {
		return true
	}

	// Customer type filter
	if filter.CustomerType != nil && customer.CustomerType != *filter.CustomerType {
		return false
	}

	// Status filter
	if filter.Status != nil && customer.Status != *filter.Status {
		return false
	}

	// Company filter
	if filter.CompanyID != nil {
		if customer.CompanyID == nil || *customer.CompanyID != *filter.CompanyID {
			return false
		}
	}

	// Active status filter
	if filter.Active != nil && customer.Active != *filter.Active {
		return false
	}

	return true
}

func TestCustomerService_CreateCustomer(t *testing.T) {
	repo := NewMockCustomerRepository()
	service := NewCustomerService(repo)
	ctx := context.Background()

	// Create a valid customer request
	req := &CreateCustomerRequest{
		Type:              string(entities.CustomerTypeIndividual),
		FirstName:         "John",
		LastName:          "Doe",
		Email:             stringPtr("john.doe@example.com"),
		Phone:             stringPtr("+1-555-123-4567"),
		CreditLimit:       decimal.NewFromFloat(1000),
		Terms:             "NET30",
		IsVATExempt:       false,
		PreferredCurrency: "USD",
		Source:            "web",
	}

	// Create the customer
	customer, err := service.CreateCustomer(ctx, req)
	if err != nil {
		t.Fatalf("CreateCustomer() error = %v", err)
	}

	// Verify the customer was created correctly
	if customer.Name != "John Doe" {
		t.Errorf("Expected name = %s, got %s", "John Doe", customer.Name)
	}

	if customer.Email == nil || *customer.Email != "john.doe@example.com" {
		t.Errorf("Expected email = %s, got %v", "john.doe@example.com", customer.Email)
	}

	if customer.CustomerType != entities.CustomerTypeIndividual {
		t.Errorf("Expected customer type = %s, got %s", entities.CustomerTypeIndividual, customer.CustomerType)
	}

	if customer.Status != entities.CustomerStatusActive {
		t.Errorf("Expected status = %s, got %s", entities.CustomerStatusActive, customer.Status)
	}

	if !customer.CreditLimit.Equal(decimal.NewFromFloat(1000)) {
		t.Errorf("Expected credit limit = %v, got %v", decimal.NewFromFloat(1000), customer.CreditLimit)
	}

	// Verify the customer was saved to the repository
	savedCustomer, err := repo.GetByEmail(ctx, "john.doe@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}

	if savedCustomer.ID != customer.ID {
		t.Error("Customer was not saved correctly")
	}
}

func TestCustomerService_CreateCustomer_DuplicateEmail(t *testing.T) {
	repo := NewMockCustomerRepository()
	service := NewCustomerService(repo)
	ctx := context.Background()

	// Create a customer first
	req := &CreateCustomerRequest{
		Type:              string(entities.CustomerTypeIndividual),
		FirstName:         "John",
		LastName:          "Doe",
		Email:             stringPtr("john.doe@example.com"),
		CreditLimit:       decimal.NewFromFloat(1000),
		Terms:             "NET30",
		PreferredCurrency: "USD",
		Source:            "web",
	}

	_, err := service.CreateCustomer(ctx, req)
	if err != nil {
		t.Fatalf("CreateCustomer() error = %v", err)
	}

	// Try to create another customer with the same email
	req2 := &CreateCustomerRequest{
		Type:              string(entities.CustomerTypeIndividual),
		FirstName:         "Jane",
		LastName:          "Smith",
		Email:             stringPtr("john.doe@example.com"), // Same email
		CreditLimit:       decimal.NewFromFloat(1000),
		Terms:             "NET30",
		PreferredCurrency: "USD",
		Source:            "web",
	}

	_, err = service.CreateCustomer(ctx, req2)
	if err == nil {
		t.Error("Expected error for duplicate email")
	}
}

func TestCustomerService_GetCustomer(t *testing.T) {
	repo := NewMockCustomerRepository()
	service := NewCustomerService(repo)
	ctx := context.Background()

	// Create a customer first
	req := &CreateCustomerRequest{
		Type:              string(entities.CustomerTypeIndividual),
		FirstName:         "John",
		LastName:          "Doe",
		Email:             stringPtr("john.doe@example.com"),
		CreditLimit:       decimal.NewFromFloat(1000),
		Terms:             "NET30",
		PreferredCurrency: "USD",
		Source:            "web",
	}

	customer, err := service.CreateCustomer(ctx, req)
	if err != nil {
		t.Fatalf("CreateCustomer() error = %v", err)
	}

	// Get the customer
	retrieved, err := service.GetCustomer(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetCustomer() error = %v", err)
	}

	if retrieved.ID != customer.ID {
		t.Error("Retrieved customer ID doesn't match")
	}

	if retrieved.Name != "John Doe" {
		t.Errorf("Expected name = %s, got %s", "John Doe", retrieved.Name)
	}
}

func TestCustomerService_GetCustomerByEmail(t *testing.T) {
	repo := NewMockCustomerRepository()
	service := NewCustomerService(repo)
	ctx := context.Background()

	// Create a customer first
	req := &CreateCustomerRequest{
		Type:              string(entities.CustomerTypeIndividual),
		FirstName:         "John",
		LastName:          "Doe",
		Email:             stringPtr("john.doe@example.com"),
		CreditLimit:       decimal.NewFromFloat(1000),
		Terms:             "NET30",
		PreferredCurrency: "USD",
		Source:            "web",
	}

	_, err := service.CreateCustomer(ctx, req)
	if err != nil {
		t.Fatalf("CreateCustomer() error = %v", err)
	}

	// Get the customer by email
	retrieved, err := service.GetCustomerByEmail(ctx, "john.doe@example.com")
	if err != nil {
		t.Fatalf("GetCustomerByEmail() error = %v", err)
	}

	if retrieved.Name != "John Doe" {
		t.Errorf("Expected name = %s, got %s", "John Doe", retrieved.Name)
	}
}

func TestCustomerService_UpdateCustomer(t *testing.T) {
	repo := NewMockCustomerRepository()
	service := NewCustomerService(repo)
	ctx := context.Background()

	// Create a customer first
	req := &CreateCustomerRequest{
		Type:              string(entities.CustomerTypeIndividual),
		FirstName:         "John",
		LastName:          "Doe",
		Email:             stringPtr("john.doe@example.com"),
		CreditLimit:       decimal.NewFromFloat(1000),
		Terms:             "NET30",
		PreferredCurrency: "USD",
		Source:            "web",
	}

	customer, err := service.CreateCustomer(ctx, req)
	if err != nil {
		t.Fatalf("CreateCustomer() error = %v", err)
	}

	// Update the customer
	businessType := entities.CustomerTypeBusiness
	updateReq := &UpdateCustomerRequest{
		Name:         stringPtr("John Smith"),
		Email:        stringPtr("john.smith@example.com"),
		CustomerType: &businessType,
	}

	updated, err := service.UpdateCustomer(ctx, customer.ID, updateReq)
	if err != nil {
		t.Fatalf("UpdateCustomer() error = %v", err)
	}

	if updated.Name != "John Smith" {
		t.Errorf("Expected name = %s, got %s", "John Smith", updated.Name)
	}

	if updated.Email == nil || *updated.Email != "john.smith@example.com" {
		t.Errorf("Expected email = %s, got %v", "john.smith@example.com", updated.Email)
	}

	if updated.CustomerType != entities.CustomerTypeBusiness {
		t.Errorf("Expected customer type = %s, got %s", entities.CustomerTypeBusiness, updated.CustomerType)
	}
}

func TestCustomerService_DeleteCustomer(t *testing.T) {
	repo := NewMockCustomerRepository()
	service := NewCustomerService(repo)
	ctx := context.Background()

	// Create a customer first
	req := &CreateCustomerRequest{
		Type:              string(entities.CustomerTypeIndividual),
		FirstName:         "John",
		LastName:          "Doe",
		Email:             stringPtr("john.doe@example.com"),
		CreditLimit:       decimal.NewFromFloat(1000),
		Terms:             "NET30",
		PreferredCurrency: "USD",
		Source:            "web",
	}

	customer, err := service.CreateCustomer(ctx, req)
	if err != nil {
		t.Fatalf("CreateCustomer() error = %v", err)
	}

	// Delete the customer
	err = service.DeleteCustomer(ctx, customer.ID)
	if err != nil {
		t.Fatalf("DeleteCustomer() error = %v", err)
	}

	// Verify the customer was deleted
	_, err = repo.GetByID(ctx, customer.ID)
	if err == nil {
		t.Error("Expected error when getting deleted customer")
	}
}

func TestCustomerService_ListCustomers(t *testing.T) {
	repo := NewMockCustomerRepository()
	service := NewCustomerService(repo)
	ctx := context.Background()

	// Create some customers
	for i := 0; i < 3; i++ {
		req := &CreateCustomerRequest{
			Type:              string(entities.CustomerTypeIndividual),
			FirstName:         "John",
			LastName:          "Doe",
			Email:             stringPtr(fmt.Sprintf("john%d@example.com", i)),
			CreditLimit:       decimal.NewFromFloat(1000),
			Terms:             "NET30",
			PreferredCurrency: "USD",
			Source:            "web",
		}

		_, err := service.CreateCustomer(ctx, req)
		if err != nil {
			t.Fatalf("CreateCustomer() error = %v", err)
		}
	}

	// List all customers
	customers, err := service.ListCustomers(ctx, nil)
	if err != nil {
		t.Fatalf("ListCustomers() error = %v", err)
	}

	if len(customers) != 3 {
		t.Errorf("Expected 3 customers, got %d", len(customers))
	}

	// List with filter
	active := true
	filter := &repositories.CustomerFilter{
		Active: &active,
	}

	customers, err = service.ListCustomers(ctx, filter)
	if err != nil {
		t.Fatalf("ListCustomers() error = %v", err)
	}

	if len(customers) != 3 {
		t.Errorf("Expected 3 active customers, got %d", len(customers))
	}
}

func TestCustomerService_ActivateCustomer(t *testing.T) {
	repo := NewMockCustomerRepository()
	service := NewCustomerService(repo)
	ctx := context.Background()

	// Create a customer
	req := &CreateCustomerRequest{
		Type:              string(entities.CustomerTypeIndividual),
		FirstName:         "John",
		LastName:          "Doe",
		Email:             stringPtr("john.doe@example.com"),
		CreditLimit:       decimal.NewFromFloat(1000),
		Terms:             "NET30",
		PreferredCurrency: "USD",
		Source:            "web",
	}

	customer, err := service.CreateCustomer(ctx, req)
	if err != nil {
		t.Fatalf("CreateCustomer() error = %v", err)
	}

	// Deactivate the customer first
	err = service.DeactivateCustomer(ctx, customer.ID)
	if err != nil {
		t.Fatalf("DeactivateCustomer() error = %v", err)
	}

	// Activate the customer
	err = service.ActivateCustomer(ctx, customer.ID)
	if err != nil {
		t.Fatalf("ActivateCustomer() error = %v", err)
	}

	// Verify the customer was activated
	updated, err := service.GetCustomer(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetCustomer() error = %v", err)
	}

	if !updated.IsActive() {
		t.Error("Customer was not activated")
	}
}

func TestCustomerService_DeactivateCustomer(t *testing.T) {
	repo := NewMockCustomerRepository()
	service := NewCustomerService(repo)
	ctx := context.Background()

	// Create a customer
	req := &CreateCustomerRequest{
		Type:              string(entities.CustomerTypeIndividual),
		FirstName:         "John",
		LastName:          "Doe",
		Email:             stringPtr("john.doe@example.com"),
		CreditLimit:       decimal.NewFromFloat(1000),
		Terms:             "NET30",
		PreferredCurrency: "USD",
		Source:            "web",
	}

	customer, err := service.CreateCustomer(ctx, req)
	if err != nil {
		t.Fatalf("CreateCustomer() error = %v", err)
	}

	// Deactivate the customer
	err = service.DeactivateCustomer(ctx, customer.ID)
	if err != nil {
		t.Fatalf("DeactivateCustomer() error = %v", err)
	}

	// Verify the customer was deactivated
	updated, err := service.GetCustomer(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetCustomer() error = %v", err)
	}

	if !updated.IsInactive() {
		t.Error("Customer was not deactivated")
	}
}

func TestCustomerService_SuspendCustomer(t *testing.T) {
	repo := NewMockCustomerRepository()
	service := NewCustomerService(repo)
	ctx := context.Background()

	// Create a customer
	req := &CreateCustomerRequest{
		Type:              string(entities.CustomerTypeIndividual),
		FirstName:         "John",
		LastName:          "Doe",
		Email:             stringPtr("john.doe@example.com"),
		CreditLimit:       decimal.NewFromFloat(1000),
		Terms:             "NET30",
		PreferredCurrency: "USD",
		Source:            "web",
	}

	customer, err := service.CreateCustomer(ctx, req)
	if err != nil {
		t.Fatalf("CreateCustomer() error = %v", err)
	}

	// Suspend the customer
	err = service.SuspendCustomer(ctx, customer.ID)
	if err != nil {
		t.Fatalf("SuspendCustomer() error = %v", err)
	}

	// Verify the customer was suspended
	updated, err := service.GetCustomer(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetCustomer() error = %v", err)
	}

	if !updated.IsSuspended() {
		t.Error("Customer was not suspended")
	}
}

func TestCustomerService_CreateBusinessCustomer(t *testing.T) {
	repo := NewMockCustomerRepository()
	service := NewCustomerService(repo)
	ctx := context.Background()

	companyID := uuid.New()
	// Create a business customer
	req := &CreateCustomerRequest{
		CompanyID:         &companyID,
		Type:              string(entities.CustomerTypeBusiness),
		CompanyName:       stringPtr("Acme Corporation"),
		Email:             stringPtr("info@acme.com"),
		Phone:             stringPtr("+1-555-123-4567"),
		Website:           stringPtr("https://acme.com"),
		TaxID:             stringPtr("12-3456789"),
		Industry:          stringPtr("Technology"),
		CreditLimit:       decimal.NewFromFloat(10000),
		Terms:             "NET60",
		IsVATExempt:       false,
		PreferredCurrency: "USD",
		Source:            "sales",
	}

	// Create the customer
	customer, err := service.CreateCustomer(ctx, req)
	if err != nil {
		t.Fatalf("CreateCustomer() error = %v", err)
	}

	// Verify the business customer was created correctly
	if customer.CustomerType != entities.CustomerTypeBusiness {
		t.Errorf("Expected customer type = %s, got %s", entities.CustomerTypeBusiness, customer.CustomerType)
	}

	if customer.CompanyID == nil || *customer.CompanyID != companyID {
		t.Error("Company ID was not set correctly")
	}

	if customer.CompanyName == nil || *customer.CompanyName != "Acme Corporation" {
		t.Errorf("Expected company name = %s, got %v", "Acme Corporation", customer.CompanyName)
	}

	if customer.Email == nil || *customer.Email != "info@acme.com" {
		t.Errorf("Expected email = %s, got %v", "info@acme.com", customer.Email)
	}

	if customer.TaxID == nil || *customer.TaxID != "12-3456789" {
		t.Errorf("Expected tax ID = %s, got %v", "12-3456789", customer.TaxID)
	}

	if !customer.CreditLimit.Equal(decimal.NewFromFloat(10000)) {
		t.Errorf("Expected credit limit = %v, got %v", decimal.NewFromFloat(10000), customer.CreditLimit)
	}

	if customer.Terms != "NET60" {
		t.Errorf("Expected terms = %s, got %s", "NET60", customer.Terms)
	}
}

// Helper function to get a pointer to a string
func stringPtr(s string) *string {
	return &s
}
