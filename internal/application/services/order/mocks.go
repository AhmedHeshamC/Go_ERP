package order

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	invEntities "erpgo/internal/domain/inventory/entities"
	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/domain/orders/repositories"
)

// MockOrderRepository implements a mock for OrderRepository
type MockOrderRepository struct {
	mock.Mock
}

// NewMockOrderRepository creates a new mock order repository
func NewMockOrderRepository() *MockOrderRepository {
	return &MockOrderRepository{mock.Mock{}}
}

// Create mocks the Create method
func (m *MockOrderRepository) Create(ctx context.Context, order *entities.Order) (uuid.UUID, error) {
	args := m.Called(ctx, order)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

// GetByID mocks the GetByID method
func (m *MockOrderRepository) Get(ctx context.Context, id uuid.UUID) (*entities.Order, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*entities.Order), args.Error(1)
}

// Update mocks the Update method
func (m *MockOrderRepository) Update(ctx context.Context, order *entities.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// List mocks the List method
func (m *MockOrderRepository) List(ctx context.Context, filter *repositories.OrderFilter) ([]*entities.Order, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*entities.Order), args.Error(1)
}

// GetByCustomerID mocks the GetByCustomerID method
func (m *MockOrderRepository) GetByCustomer(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*entities.Order, error) {
	args := m.Called(ctx, customerID)
	return args.Get(0).([]*entities.Order), args.Error(1)
}

// GetByNumber mocks the GetByNumber method
func (m *MockOrderRepository) GetByNumber(ctx context.Context, number string) (*entities.Order, error) {
	args := m.Called(ctx, number)
	return args.Get(0).(*entities.Order), args.Error(1)
}

// GetByStatus mocks the GetByStatus method
func (m *MockOrderRepository) GetByStatus(ctx context.Context, status entities.OrderStatus, limit, offset int) ([]*entities.Order, error) {
	args := m.Called(ctx, status)
	return args.Get(0).([]*entities.Order), args.Error(1)
}

// Count mocks the Count method
func (m *MockOrderRepository) Count(ctx context.Context, filter *repositories.OrderFilter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

// GetOrderTotal mocks the GetOrderTotal method
func (m *MockOrderRepository) GetOrderTotal(ctx context.Context, customerID uuid.UUID, startDate, endDate time.Time) (float64, error) {
	args := m.Called(ctx, customerID, startDate, endDate)
	return args.Get(0).(float64), args.Error(1)
}

// GetOrderStats mocks the GetOrderStats method
// func (m *MockOrderRepository) GetOrderStats(ctx context.Context, filter *repositories.OrderFilter) (*entities.OrderStats, error) {
// 	args := m.Called(ctx, filter)
// 	return args.Get(0).(*entities.OrderStats), args.Error(1)
// }

// MockOrderItemRepository implements a mock for OrderItemRepository
type MockOrderItemRepository struct {
	mock.Mock
}

// NewMockOrderItemRepository creates a new mock order item repository
func NewMockOrderItemRepository() *MockOrderItemRepository {
	return &MockOrderItemRepository{mock.Mock{}}
}

// Create mocks the Create method
func (m *MockOrderItemRepository) Create(ctx context.Context, item *entities.OrderItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

// GetByID mocks the GetByID method
func (m *MockOrderItemRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.OrderItem, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*entities.OrderItem), args.Error(1)
}

// Update mocks the Update method
func (m *MockOrderItemRepository) Update(ctx context.Context, item *entities.OrderItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockOrderItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// List mocks the List method
func (m *MockOrderItemRepository) List(ctx context.Context, filter *repositories.OrderItemFilter) ([]*entities.OrderItem, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*entities.OrderItem), args.Error(1)
}

// GetByOrderID mocks the GetByOrderID method
// GetByProductID mocks the GetByProductID method
// GetProductDetails mocks the GetProductDetails method
func (m *MockOrderItemRepository) GetProductDetails(ctx context.Context, productID uuid.UUID) (*entities.OrderItem, error) {
	args := m.Called(ctx, productID)
	return args.Get(0).(*entities.OrderItem), args.Error(1)
}

// DeleteByOrderID mocks the DeleteByOrderID method
func (m *MockOrderItemRepository) DeleteByOrderID(ctx context.Context, orderID uuid.UUID) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
}

// MockOrderAddressRepository implements a mock for OrderAddressRepository
type MockOrderAddressRepository struct {
	mock.Mock
}

// NewMockOrderAddressRepository creates a new mock order address repository
func NewMockOrderAddressRepository() *MockOrderAddressRepository {
	return &MockOrderAddressRepository{mock.Mock{}}
}

// Create mocks the Create method
func (m *MockOrderAddressRepository) Create(ctx context.Context, address *entities.OrderAddress) error {
	args := m.Called(ctx, address)
	return args.Error(0)
}

// GetByID mocks the GetByID method
func (m *MockOrderAddressRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.OrderAddress, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*entities.OrderAddress), args.Error(1)
}

// Update mocks the Update method
func (m *MockOrderAddressRepository) Update(ctx context.Context, address *entities.OrderAddress) error {
	args := m.Called(ctx, address)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockOrderAddressRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// GetByOrderID mocks the GetByOrderID method
func (m *MockOrderAddressRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entities.OrderAddress, error) {
	args := m.Called(ctx, orderID)
	return args.Get(0).([]*entities.OrderAddress), args.Error(1)
}

// DeleteByOrderID mocks the DeleteByOrderID method
func (m *MockOrderAddressRepository) DeleteByOrderID(ctx context.Context, orderID uuid.UUID) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
}

// MockCompanyRepository implements a mock for CompanyRepository
type MockCompanyRepository struct {
	mock.Mock
}

// NewMockCompanyRepository creates a new mock company repository
func NewMockCompanyRepository() *MockCompanyRepository {
	return &MockCompanyRepository{mock.Mock{}}
}

// GetByID mocks the GetByID method
func (m *MockCompanyRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Company, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*entities.Company), args.Error(1)
}

// Update mocks the Update method
func (m *MockCompanyRepository) Update(ctx context.Context, company *entities.Company) error {
	args := m.Called(ctx, company)
	return args.Error(0)
}

// MockOrderAnalyticsRepository implements a mock for OrderAnalyticsRepository
type MockOrderAnalyticsRepository struct {
	mock.Mock
}

// NewMockOrderAnalyticsRepository creates a new mock order analytics repository
func NewMockOrderAnalyticsRepository() *MockOrderAnalyticsRepository {
	return &MockOrderAnalyticsRepository{mock.Mock{}}
}

// GetOrderStats mocks the GetOrderStats method
// func (m *MockOrderAnalyticsRepository) GetOrderStats(ctx context.Context, filter *repositories.OrderFilter) (*entities.OrderStats, error) {
// 	args := m.Called(ctx, filter)
// 	return args.Get(0).(*entities.OrderStats), args.Error(1)
// }

// GetCustomerStats mocks the GetCustomerStats method
// func (m *MockOrderAnalyticsRepository) GetCustomerStats(ctx context.Context, customerID uuid.UUID, startDate, endDate time.Time) (*entities.CustomerStats, error) {
// 	args := m.Called(ctx, customerID, startDate, endDate)
// 	return args.Get(0).(*entities.CustomerStats), args.Error(1)
// }

// MockProductService implements a mock for ProductService
type MockProductService struct {
	mock.Mock
}

// NewMockProductService creates a new mock product service
func NewMockProductService() *MockProductService {
	return &MockProductService{mock.Mock{}}
}

// GetProduct mocks the GetProduct method
// func (m *MockProductService) GetProduct(ctx context.Context, id uuid.UUID) (*entities.Product, error) {
// 	args := m.Called(ctx, id)
// 	return args.Get(0).(*entities.Product), args.Error(1)
// }

// UpdateStock mocks the UpdateStock method
func (m *MockProductService) UpdateStock(ctx context.Context, id uuid.UUID, quantity int) error {
	args := m.Called(ctx, id, quantity)
	return args.Error(0)
}

// MockInventoryService implements a mock for InventoryService
type MockInventoryService struct {
	mock.Mock
}

// NewMockInventoryService creates a new mock inventory service
func NewMockInventoryService() *MockInventoryService {
	return &MockInventoryService{mock.Mock{}}
}

// CheckAvailability mocks the CheckAvailability method
func (m *MockInventoryService) CheckAvailability(ctx context.Context, productID uuid.UUID, warehouseID uuid.UUID, quantity int) (bool, error) {
	args := m.Called(ctx, productID, warehouseID, quantity)
	return args.Bool(0), args.Error(1)
}

// ReserveInventory mocks the ReserveInventory method
func (m *MockInventoryService) ReserveInventory(ctx context.Context, productID, warehouseID uuid.UUID, quantity int) error {
	args := m.Called(ctx, productID, warehouseID, quantity)
	return args.Error(0)
}

// }

// IsAvailable mocks the IsAvailable method
func (m *MockInventoryService) IsAvailable(ctx context.Context, productID uuid.UUID, warehouseID uuid.UUID, quantity int) (bool, error) {
	args := m.Called(ctx, productID, warehouseID, quantity)
	return args.Bool(0), args.Error(1)
}

// MockUserService implements a mock for UserService
type MockUserService struct {
	mock.Mock
}

// NewMockUserService creates a new mock user service
func NewMockUserService() *MockUserService {
	return &MockUserService{mock.Mock{}}
}

// GetUser mocks the GetUser method
// User represents a basic user entity for mocking
type User struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

func (m *MockUserService) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*User), args.Error(1)
}

// MockNotificationService implements a mock for NotificationService
type MockNotificationService struct {
	mock.Mock
}

// NewMockNotificationService creates a new mock notification service
func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{mock.Mock{}}
}

// SendOrderNotification mocks the SendOrderNotification method
func (m *MockNotificationService) SendOrderNotification(ctx context.Context, order *entities.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

// SendInventoryNotification mocks the SendInventoryNotification method
func (m *MockNotificationService) SendInventoryNotification(ctx context.Context, inventory *invEntities.Inventory) error {
	args := m.Called(ctx, inventory)
	return args.Error(0)
}

// MockPaymentService implements a mock for PaymentService
type MockPaymentService struct {
	mock.Mock
}

// NewMockPaymentService creates a new mock payment service
func NewMockPaymentService() *MockPaymentService {
	return &MockPaymentService{mock.Mock{}}
}

// ProcessPayment mocks the ProcessPayment method
func (m *MockPaymentService) ProcessPayment(ctx context.Context, orderID uuid.UUID, amount float64) error {
	args := m.Called(ctx, orderID, amount)
	return args.Error(0)
}

// MockTaxCalculator implements a mock for TaxCalculator
type MockTaxCalculator struct {
	mock.Mock
}

// NewMockTaxCalculator creates a new mock tax calculator
func NewMockTaxCalculator() *MockTaxCalculator {
	return &MockTaxCalculator{mock.Mock{}}
}

// CalculateTax mocks the CalculateTax method
func (m *MockTaxCalculator) CalculateTax(ctx context.Context, order *entities.Order) (float64, error) {
	args := m.Called(ctx, order)
	return args.Get(0).(float64), args.Error(1)
}
