package repositories

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"erpgo/internal/domain/customers/entities"
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
		return ErrCustomerAlreadyExists
	}

	if customer.Email != nil {
		if _, exists := m.byEmail[*customer.Email]; exists {
			return ErrCustomerEmailAlreadyExists
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
		return nil, ErrCustomerNotFound
	}
	return customer, nil
}

// GetByEmail retrieves a customer by email
func (m *MockCustomerRepository) GetByEmail(ctx context.Context, email string) (*entities.Customer, error) {
	customer, exists := m.byEmail[email]
	if !exists {
		return nil, ErrCustomerNotFound
	}
	return customer, nil
}

// GetByCustomerCode retrieves a customer by customer code
func (m *MockCustomerRepository) GetByCustomerCode(ctx context.Context, code string) (*entities.Customer, error) {
	customer, exists := m.byCode[code]
	if !exists {
		return nil, ErrCustomerNotFound
	}
	return customer, nil
}

// Update updates an existing customer in the repository
func (m *MockCustomerRepository) Update(ctx context.Context, customer *entities.Customer) error {
	if _, exists := m.customers[customer.ID]; !exists {
		return ErrCustomerNotFound
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
		return ErrCustomerNotFound
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
func (m *MockCustomerRepository) List(ctx context.Context, filter *CustomerFilter) ([]*entities.Customer, error) {
	var result []*entities.Customer

	for _, customer := range m.customers {
		if m.matchesFilter(customer, filter) {
			result = append(result, customer)
		}
	}

	// Apply sorting
	result = m.applySorting(result, filter)

	// Apply pagination
	return m.applyPagination(result, filter), nil
}

// Count returns the total number of customers based on filter criteria
func (m *MockCustomerRepository) Count(ctx context.Context, filter *CustomerFilter) (int, error) {
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
		return ErrCustomerNotFound
	}

	customer.Status = status
	return nil
}

// UpdateCredit updates the credit information of a customer
func (m *MockCustomerRepository) UpdateCredit(ctx context.Context, id uuid.UUID, creditUsed decimal.Decimal) error {
	customer, exists := m.customers[id]
	if !exists {
		return ErrCustomerNotFound
	}

	customer.CreditUsed = creditUsed
	return nil
}

// WithTx returns a new repository instance bound to the given transaction (mock implementation)
func (m *MockCustomerRepository) WithTx(ctx context.Context) CustomerRepository {
	return m
}

// matchesFilter checks if a customer matches the filter criteria
func (m *MockCustomerRepository) matchesFilter(customer *entities.Customer, filter *CustomerFilter) bool {
	if filter == nil {
		return true
	}

	// Search filter
	if filter.Search != nil {
		search := *filter.Search
		match := false
		if search != "" {
			if customer.Name != "" && containsIgnoreCase(customer.Name, search) {
				match = true
			}
			if !match && customer.Email != nil && containsIgnoreCase(*customer.Email, search) {
				match = true
			}
			if !match && customer.CompanyName != nil && containsIgnoreCase(*customer.CompanyName, search) {
				match = true
			}
		}
		if !match {
			return false
		}
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

	// Source filter
	if filter.Source != nil && customer.Source != *filter.Source {
		return false
	}

	// Industry filter
	if filter.Industry != nil {
		if customer.Industry == nil || *customer.Industry != *filter.Industry {
			return false
		}
	}

	// VAT exemption filter
	if filter.IsVATExempt != nil && customer.IsVATExempt != *filter.IsVATExempt {
		return false
	}

	// Credit limit range filter
	if filter.CreditMin != nil && customer.CreditLimit.LessThan(*filter.CreditMin) {
		return false
	}
	if filter.CreditMax != nil && customer.CreditLimit.GreaterThan(*filter.CreditMax) {
		return false
	}

	// Creation date range filter
	if filter.CreatedAfter != nil && customer.CreatedAt.Before(*filter.CreatedAfter) {
		return false
	}
	if filter.CreatedBefore != nil && customer.CreatedAt.After(*filter.CreatedBefore) {
		return false
	}

	// Last updated date range filter
	if filter.UpdatedAfter != nil && customer.UpdatedAt.Before(*filter.UpdatedAfter) {
		return false
	}
	if filter.UpdatedBefore != nil && customer.UpdatedAt.After(*filter.UpdatedBefore) {
		return false
	}

	// Active status filter
	if filter.Active != nil && customer.Active != *filter.Active {
		return false
	}

	return true
}

// applySorting applies sorting to the customer list
func (m *MockCustomerRepository) applySorting(customers []*entities.Customer, filter *CustomerFilter) []*entities.Customer {
	if filter == nil || filter.SortBy == nil {
		return customers
	}

	sortBy := *filter.SortBy
	sortOrder := "asc"
	if filter.SortOrder != nil {
		sortOrder = *filter.SortOrder
	}

	switch sortBy {
	case "name":
		if sortOrder == "desc" {
			return m.sortByDesc(customers, func(c *entities.Customer) string { return c.Name })
		}
		return m.sortByAsc(customers, func(c *entities.Customer) string { return c.Name })
	case "created_at":
		if sortOrder == "desc" {
			return m.sortByDescTime(customers, func(c *entities.Customer) time.Time { return c.CreatedAt })
		}
		return m.sortByAscTime(customers, func(c *entities.Customer) time.Time { return c.CreatedAt })
	case "credit_limit":
		if sortOrder == "desc" {
			return m.sortByDescDecimal(customers, func(c *entities.Customer) decimal.Decimal { return c.CreditLimit })
		}
		return m.sortByAscDecimal(customers, func(c *entities.Customer) decimal.Decimal { return c.CreditLimit })
	default:
		return customers
	}
}

// applyPagination applies pagination to the customer list
func (m *MockCustomerRepository) applyPagination(customers []*entities.Customer, filter *CustomerFilter) []*entities.Customer {
	if filter == nil || filter.Page == nil || filter.PageSize == nil {
		return customers
	}

	page := *filter.Page
	pageSize := *filter.PageSize

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= len(customers) {
		return []*entities.Customer{}
	}

	if end > len(customers) {
		end = len(customers)
	}

	return customers[start:end]
}

// sortByAsc sorts customers in ascending order by string key
func (m *MockCustomerRepository) sortByAsc(customers []*entities.Customer, keyFunc func(*entities.Customer) string) []*entities.Customer {
	result := make([]*entities.Customer, len(customers))
	copy(result, customers)

	// Simple bubble sort for demo purposes
	for i := 0; i < len(result)-1; i++ {
		for j := 0; j < len(result)-i-1; j++ {
			if keyFunc(result[j]) > keyFunc(result[j+1]) {
				result[j], result[j+1] = result[j+1], result[j]
			}
		}
	}

	return result
}

// sortByDesc sorts customers in descending order by string key
func (m *MockCustomerRepository) sortByDesc(customers []*entities.Customer, keyFunc func(*entities.Customer) string) []*entities.Customer {
	result := make([]*entities.Customer, len(customers))
	copy(result, customers)

	// Simple bubble sort for demo purposes
	for i := 0; i < len(result)-1; i++ {
		for j := 0; j < len(result)-i-1; j++ {
			if keyFunc(result[j]) < keyFunc(result[j+1]) {
				result[j], result[j+1] = result[j+1], result[j]
			}
		}
	}

	return result
}

// sortByAscTime sorts customers in ascending order by time key
func (m *MockCustomerRepository) sortByAscTime(customers []*entities.Customer, keyFunc func(*entities.Customer) time.Time) []*entities.Customer {
	result := make([]*entities.Customer, len(customers))
	copy(result, customers)

	// Simple bubble sort for demo purposes
	for i := 0; i < len(result)-1; i++ {
		for j := 0; j < len(result)-i-1; j++ {
			if keyFunc(result[j]).After(keyFunc(result[j+1])) {
				result[j], result[j+1] = result[j+1], result[j]
			}
		}
	}

	return result
}

// sortByDescTime sorts customers in descending order by time key
func (m *MockCustomerRepository) sortByDescTime(customers []*entities.Customer, keyFunc func(*entities.Customer) time.Time) []*entities.Customer {
	result := make([]*entities.Customer, len(customers))
	copy(result, customers)

	// Simple bubble sort for demo purposes
	for i := 0; i < len(result)-1; i++ {
		for j := 0; j < len(result)-i-1; j++ {
			if keyFunc(result[j]).Before(keyFunc(result[j+1])) {
				result[j], result[j+1] = result[j+1], result[j]
			}
		}
	}

	return result
}

// sortByAscDecimal sorts customers in ascending order by decimal key
func (m *MockCustomerRepository) sortByAscDecimal(customers []*entities.Customer, keyFunc func(*entities.Customer) decimal.Decimal) []*entities.Customer {
	result := make([]*entities.Customer, len(customers))
	copy(result, customers)

	// Simple bubble sort for demo purposes
	for i := 0; i < len(result)-1; i++ {
		for j := 0; j < len(result)-i-1; j++ {
			if keyFunc(result[j]).GreaterThan(keyFunc(result[j+1])) {
				result[j], result[j+1] = result[j+1], result[j]
			}
		}
	}

	return result
}

// sortByDescDecimal sorts customers in descending order by decimal key
func (m *MockCustomerRepository) sortByDescDecimal(customers []*entities.Customer, keyFunc func(*entities.Customer) decimal.Decimal) []*entities.Customer {
	result := make([]*entities.Customer, len(customers))
	copy(result, customers)

	// Simple bubble sort for demo purposes
	for i := 0; i < len(result)-1; i++ {
		for j := 0; j < len(result)-i-1; j++ {
			if keyFunc(result[j]).LessThan(keyFunc(result[j+1])) {
				result[j], result[j+1] = result[j+1], result[j]
			}
		}
	}

	return result
}

// containsIgnoreCase checks if a string contains a substring, case-insensitive
func containsIgnoreCase(str, substr string) bool {
	str, substr = strings.ToLower(str), strings.ToLower(substr)
	return strings.Contains(str, substr)
}

// Test constants are defined in errors.go

func TestMockCustomerRepository_Create(t *testing.T) {
	repo := NewMockCustomerRepository()
	ctx := context.Background()

	// Create a customer
	customer := entities.NewCustomer()
	customer.CustomerCode = "CUST-001"
	customer.Name = "John Doe"
	customer.Email = stringPtr("john.doe@example.com")
	customer.CustomerType = entities.CustomerTypeIndividual
	customer.Status = entities.CustomerStatusActive
	customer.CreditLimit = decimal.NewFromFloat(1000)
	customer.Terms = "NET30"
	customer.Active = true
	customer.PreferredCurrency = "USD"
	customer.Source = "web"

	// Create the customer
	err := repo.Create(ctx, customer)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify the customer was created
	retrieved, err := repo.GetByID(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Name != "John Doe" {
		t.Errorf("Expected name = %s, got %s", "John Doe", retrieved.Name)
	}

	// Try to create the same customer again
	err = repo.Create(ctx, customer)
	if err != ErrCustomerAlreadyExists {
		t.Errorf("Expected error = %v, got %v", ErrCustomerAlreadyExists, err)
	}
}

func TestMockCustomerRepository_GetByID(t *testing.T) {
	repo := NewMockCustomerRepository()
	ctx := context.Background()

	// Get a non-existent customer
	_, err := repo.GetByID(ctx, uuid.New())
	if err != ErrCustomerNotFound {
		t.Errorf("Expected error = %v, got %v", ErrCustomerNotFound, err)
	}

	// Create a customer
	customer := entities.NewCustomer()
	customer.CustomerCode = "CUST-002"
	customer.Name = "Jane Smith"
	customer.Email = stringPtr("jane.smith@example.com")
	customer.CustomerType = entities.CustomerTypeIndividual
	customer.Status = entities.CustomerStatusActive
	customer.CreditLimit = decimal.NewFromFloat(1000)
	customer.Terms = "NET30"
	customer.Active = true
	customer.PreferredCurrency = "USD"
	customer.Source = "web"

	repo.Create(ctx, customer)

	// Get the customer
	retrieved, err := repo.GetByID(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Name != "Jane Smith" {
		t.Errorf("Expected name = %s, got %s", "Jane Smith", retrieved.Name)
	}
}

func TestMockCustomerRepository_GetByEmail(t *testing.T) {
	repo := NewMockCustomerRepository()
	ctx := context.Background()

	// Get a non-existent customer by email
	_, err := repo.GetByEmail(ctx, "nonexistent@example.com")
	if err != ErrCustomerNotFound {
		t.Errorf("Expected error = %v, got %v", ErrCustomerNotFound, err)
	}

	// Create a customer
	customer := entities.NewCustomer()
	customer.CustomerCode = "CUST-003"
	customer.Name = "Bob Johnson"
	customer.Email = stringPtr("bob.johnson@example.com")
	customer.CustomerType = entities.CustomerTypeIndividual
	customer.Status = entities.CustomerStatusActive
	customer.CreditLimit = decimal.NewFromFloat(1000)
	customer.Terms = "NET30"
	customer.Active = true
	customer.PreferredCurrency = "USD"
	customer.Source = "web"

	repo.Create(ctx, customer)

	// Get the customer by email
	retrieved, err := repo.GetByEmail(ctx, "bob.johnson@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}

	if retrieved.Name != "Bob Johnson" {
		t.Errorf("Expected name = %s, got %s", "Bob Johnson", retrieved.Name)
	}
}

func TestMockCustomerRepository_GetByCustomerCode(t *testing.T) {
	repo := NewMockCustomerRepository()
	ctx := context.Background()

	// Get a non-existent customer by code
	_, err := repo.GetByCustomerCode(ctx, "CUST-NONEXISTENT")
	if err != ErrCustomerNotFound {
		t.Errorf("Expected error = %v, got %v", ErrCustomerNotFound, err)
	}

	// Create a customer
	customer := entities.NewCustomer()
	customer.CustomerCode = "CUST-004"
	customer.Name = "Alice Williams"
	customer.Email = stringPtr("alice.williams@example.com")
	customer.CustomerType = entities.CustomerTypeIndividual
	customer.Status = entities.CustomerStatusActive
	customer.CreditLimit = decimal.NewFromFloat(1000)
	customer.Terms = "NET30"
	customer.Active = true
	customer.PreferredCurrency = "USD"
	customer.Source = "web"

	repo.Create(ctx, customer)

	// Get the customer by code
	retrieved, err := repo.GetByCustomerCode(ctx, "CUST-004")
	if err != nil {
		t.Fatalf("GetByCustomerCode() error = %v", err)
	}

	if retrieved.Name != "Alice Williams" {
		t.Errorf("Expected name = %s, got %s", "Alice Williams", retrieved.Name)
	}
}

func TestMockCustomerRepository_Update(t *testing.T) {
	repo := NewMockCustomerRepository()
	ctx := context.Background()

	// Create a customer
	customer := entities.NewCustomer()
	customer.CustomerCode = "CUST-005"
	customer.Name = "Charlie Brown"
	customer.Email = stringPtr("charlie.brown@example.com")
	customer.CustomerType = entities.CustomerTypeIndividual
	customer.Status = entities.CustomerStatusActive
	customer.CreditLimit = decimal.NewFromFloat(1000)
	customer.Terms = "NET30"
	customer.Active = true
	customer.PreferredCurrency = "USD"
	customer.Source = "web"

	repo.Create(ctx, customer)

	// Update the customer
	customer.Name = "Charlie Brown Jr."
	customer.Status = entities.CustomerStatusInactive
	newEmail := "charlie.brown.jr@example.com"
	customer.Email = &newEmail

	err := repo.Update(ctx, customer)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify the update
	retrieved, err := repo.GetByID(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Name != "Charlie Brown Jr." {
		t.Errorf("Expected name = %s, got %s", "Charlie Brown Jr.", retrieved.Name)
	}

	if retrieved.Status != entities.CustomerStatusInactive {
		t.Errorf("Expected status = %v, got %v", entities.CustomerStatusInactive, retrieved.Status)
	}

	if retrieved.Email == nil || *retrieved.Email != "charlie.brown.jr@example.com" {
		t.Errorf("Expected email = %s, got %v", "charlie.brown.jr@example.com", retrieved.Email)
	}

	// Try to update a non-existent customer
	nonExistentCustomer := entities.NewCustomer()
	err = repo.Update(ctx, nonExistentCustomer)
	if err != ErrCustomerNotFound {
		t.Errorf("Expected error = %v, got %v", ErrCustomerNotFound, err)
	}
}

func TestMockCustomerRepository_Delete(t *testing.T) {
	repo := NewMockCustomerRepository()
	ctx := context.Background()

	// Delete a non-existent customer
	err := repo.Delete(ctx, uuid.New())
	if err != ErrCustomerNotFound {
		t.Errorf("Expected error = %v, got %v", ErrCustomerNotFound, err)
	}

	// Create a customer
	customer := entities.NewCustomer()
	customer.CustomerCode = "CUST-006"
	customer.Name = "Diana Prince"
	customer.Email = stringPtr("diana.prince@example.com")
	customer.CustomerType = entities.CustomerTypeIndividual
	customer.Status = entities.CustomerStatusActive
	customer.CreditLimit = decimal.NewFromFloat(1000)
	customer.Terms = "NET30"
	customer.Active = true
	customer.PreferredCurrency = "USD"
	customer.Source = "web"

	repo.Create(ctx, customer)

	// Delete the customer
	err = repo.Delete(ctx, customer.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify the customer was deleted
	_, err = repo.GetByID(ctx, customer.ID)
	if err != ErrCustomerNotFound {
		t.Errorf("Expected error = %v, got %v", ErrCustomerNotFound, err)
	}
}

func TestMockCustomerRepository_List(t *testing.T) {
	repo := NewMockCustomerRepository()
	ctx := context.Background()

	// Create some customers
	customers := make([]*entities.Customer, 5)
	for i := 0; i < 5; i++ {
		customers[i] = entities.NewCustomer()
		customers[i].CustomerCode = fmt.Sprintf("CUST-%03d", i+7)
		customers[i].Name = fmt.Sprintf("Customer %d", i+1)
		email := fmt.Sprintf("customer%d@example.com", i+1)
		customers[i].Email = &email
		customers[i].CustomerType = entities.CustomerTypeIndividual
		customers[i].Status = entities.CustomerStatusActive
		customers[i].CreditLimit = decimal.NewFromFloat(1000 * float64(i+1))
		customers[i].Terms = "NET30"
		customers[i].Active = true
		customers[i].PreferredCurrency = "USD"
		customers[i].Source = "web"

		repo.Create(ctx, customers[i])
	}

	// List all customers
	list, err := repo.List(ctx, nil)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != 5 {
		t.Errorf("Expected 5 customers, got %d", len(list))
	}

	// List with filter
	active := true
	filter := &CustomerFilter{
		Active: &active,
	}

	list, err = repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != 5 {
		t.Errorf("Expected 5 active customers, got %d", len(list))
	}

	// List with status filter
	status := entities.CustomerStatusActive
	filter = &CustomerFilter{
		Status: &status,
	}

	list, err = repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != 5 {
		t.Errorf("Expected 5 customers with Active status, got %d", len(list))
	}

	// List with pagination
	page := 1
	pageSize := 2
	filter = &CustomerFilter{
		Page:     &page,
		PageSize: &pageSize,
	}

	list, err = repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != 2 {
		t.Errorf("Expected 2 customers in page 1, got %d", len(list))
	}

	// List with sorting
	sortBy := "name"
	sortOrder := "asc"
	filter = &CustomerFilter{
		SortBy:    &sortBy,
		SortOrder: &sortOrder,
	}

	list, err = repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != 5 {
		t.Errorf("Expected 5 customers when sorting, got %d", len(list))
	}

	// Verify sorting
	if list[0].Name != "Customer 1" {
		t.Errorf("Expected first customer to be %s, got %s", "Customer 1", list[0].Name)
	}
}

func TestMockCustomerRepository_Count(t *testing.T) {
	repo := NewMockCustomerRepository()
	ctx := context.Background()

	// Count with no customers
	count, err := repo.Count(ctx, nil)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 customers, got %d", count)
	}

	// Create some customers
	for i := 0; i < 5; i++ {
		customer := entities.NewCustomer()
		customer.CustomerCode = fmt.Sprintf("CUST-%03d", i+12)
		customer.Name = fmt.Sprintf("Customer %d", i+1)
		email := fmt.Sprintf("customer%d@example.com", i+1)
		customer.Email = &email
		customer.CustomerType = entities.CustomerTypeIndividual
		customer.Status = entities.CustomerStatusActive
		customer.CreditLimit = decimal.NewFromFloat(1000 * float64(i+1))
		customer.Terms = "NET30"
		customer.Active = true
		customer.PreferredCurrency = "USD"
		customer.Source = "web"

		repo.Create(ctx, customer)
	}

	// Count all customers
	count, err = repo.Count(ctx, nil)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 customers, got %d", count)
	}

	// Count with filter
	active := true
	filter := &CustomerFilter{
		Active: &active,
	}

	count, err = repo.Count(ctx, filter)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 active customers, got %d", count)
	}
}

func TestMockCustomerRepository_Exists(t *testing.T) {
	repo := NewMockCustomerRepository()
	ctx := context.Background()

	// Check if non-existent email exists
	exists, err := repo.Exists(ctx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if exists {
		t.Error("Expected false for non-existent email")
	}

	// Create a customer
	customer := entities.NewCustomer()
	customer.CustomerCode = "CUST-017"
	customer.Name = "Eve Adams"
	email := "eve.adams@example.com"
	customer.Email = &email
	customer.CustomerType = entities.CustomerTypeIndividual
	customer.Status = entities.CustomerStatusActive
	customer.CreditLimit = decimal.NewFromFloat(1000)
	customer.Terms = "NET30"
	customer.Active = true
	customer.PreferredCurrency = "USD"
	customer.Source = "web"

	repo.Create(ctx, customer)

	// Check if email exists
	exists, err = repo.Exists(ctx, "eve.adams@example.com")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if !exists {
		t.Error("Expected true for existing email")
	}
}

func TestMockCustomerRepository_UpdateStatus(t *testing.T) {
	repo := NewMockCustomerRepository()
	ctx := context.Background()

	// Update status for non-existent customer
	err := repo.UpdateStatus(ctx, uuid.New(), entities.CustomerStatusInactive)
	if err != ErrCustomerNotFound {
		t.Errorf("Expected error = %v, got %v", ErrCustomerNotFound, err)
	}

	// Create a customer
	customer := entities.NewCustomer()
	customer.CustomerCode = "CUST-018"
	customer.Name = "Frank Miller"
	email := "frank.miller@example.com"
	customer.Email = &email
	customer.CustomerType = entities.CustomerTypeIndividual
	customer.Status = entities.CustomerStatusActive
	customer.CreditLimit = decimal.NewFromFloat(1000)
	customer.Terms = "NET30"
	customer.Active = true
	customer.PreferredCurrency = "USD"
	customer.Source = "web"

	repo.Create(ctx, customer)

	// Update the status
	err = repo.UpdateStatus(ctx, customer.ID, entities.CustomerStatusInactive)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	// Verify the status was updated
	retrieved, err := repo.GetByID(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Status != entities.CustomerStatusInactive {
		t.Errorf("Expected status = %v, got %v", entities.CustomerStatusInactive, retrieved.Status)
	}
}

func TestMockCustomerRepository_UpdateCredit(t *testing.T) {
	repo := NewMockCustomerRepository()
	ctx := context.Background()

	// Update credit for non-existent customer
	err := repo.UpdateCredit(ctx, uuid.New(), decimal.NewFromFloat(500))
	if err != ErrCustomerNotFound {
		t.Errorf("Expected error = %v, got %v", ErrCustomerNotFound, err)
	}

	// Create a customer
	customer := entities.NewCustomer()
	customer.CustomerCode = "CUST-019"
	customer.Name = "Grace Kelly"
	email := "grace.kelly@example.com"
	customer.Email = &email
	customer.CustomerType = entities.CustomerTypeIndividual
	customer.Status = entities.CustomerStatusActive
	customer.CreditLimit = decimal.NewFromFloat(1000)
	customer.CreditUsed = decimal.NewFromFloat(300)
	customer.Terms = "NET30"
	customer.Active = true
	customer.PreferredCurrency = "USD"
	customer.Source = "web"

	repo.Create(ctx, customer)

	// Update the credit used
	err = repo.UpdateCredit(ctx, customer.ID, decimal.NewFromFloat(500))
	if err != nil {
		t.Fatalf("UpdateCredit() error = %v", err)
	}

	// Verify the credit was updated
	retrieved, err := repo.GetByID(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if !retrieved.CreditUsed.Equal(decimal.NewFromFloat(500)) {
		t.Errorf("Expected credit used = %v, got %v", decimal.NewFromFloat(500), retrieved.CreditUsed)
	}
}

// Helper function to get a pointer to a string
func stringPtr(s string) *string {
	return &s
}
