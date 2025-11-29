package order

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"erpgo/internal/domain/orders/entities"
)

// MockOrderRepository is a simplified mock for testing
type MockOrderRepository struct {
	mock.Mock
}

// NewMockOrderRepository creates a new mock order repository
func NewMockOrderRepository() *MockOrderRepository {
	return &MockOrderRepository{mock.Mock{}}
}

// Create mocks the Create method
func (m *MockOrderRepository) Create(ctx context.Context, order *entities.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

// GetByID mocks the GetByID method
func (m *MockOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Order, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*entities.Order), args.Error(1)
}

// Update mocks the Update method
func (m *MockOrderRepository) Update(ctx context.Context, order *entities.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

// List mocks the List method
func (m *MockOrderRepository) List(ctx context.Context, filter interface{}) ([]*entities.Order, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*entities.Order), args.Error(1)
}

// TestCreateOrder tests the CreateOrder method
func TestCreateOrder(t *testing.T) {
	// Setup
	mockRepo := NewMockOrderRepository()

	ctx := context.Background()

	// Test data
	order := &entities.Order{
		ID:          uuid.New(),
		CustomerID:  uuid.New(),
		OrderNumber: "TEST-001",
		Status:      entities.OrderStatusDraft,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Mock expectations
	mockRepo.On("Create", ctx, mock.AnythingOfType("*entities.Order")).Return(nil)

	// Execute
	err := mockRepo.Create(ctx, order)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestGetOrderByID tests the GetByID method
func TestGetOrderByID(t *testing.T) {
	// Setup
	mockRepo := NewMockOrderRepository()

	ctx := context.Background()
	orderID := uuid.New()

	expectedOrder := &entities.Order{
		ID:          orderID,
		CustomerID:  uuid.New(),
		OrderNumber: "TEST-001",
		Status:      entities.OrderStatusDraft,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Mock expectations
	mockRepo.On("GetByID", ctx, orderID).Return(expectedOrder, nil)

	// Execute
	order, err := mockRepo.GetByID(ctx, orderID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedOrder.ID, order.ID)
	assert.Equal(t, expectedOrder.OrderNumber, order.OrderNumber)
	mockRepo.AssertExpectations(t)
}

// TestUpdateOrder tests the UpdateOrder method
func TestUpdateOrder(t *testing.T) {
	// Setup
	mockRepo := NewMockOrderRepository()

	ctx := context.Background()
	orderID := uuid.New()
	updatedOrder := &entities.Order{
		ID:          orderID,
		CustomerID:  uuid.New(),
		OrderNumber: "TEST-001-UPDATED",
		Status:      entities.OrderStatusConfirmed,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Mock expectations
	mockRepo.On("Update", ctx, mock.AnythingOfType("*entities.Order")).Return(nil)

	// Execute
	err := mockRepo.Update(ctx, updatedOrder)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestListOrders tests the List method
func TestListOrders(t *testing.T) {
	// Setup
	mockRepo := NewMockOrderRepository()

	ctx := context.Background()

	// Test data
	expectedOrders := []*entities.Order{
		{
			ID:          uuid.New(),
			CustomerID:  uuid.New(),
			OrderNumber: "TEST-001",
			Status:      entities.OrderStatusDraft,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
		{
			ID:          uuid.New(),
			CustomerID:  uuid.New(),
			OrderNumber: "TEST-002",
			Status:      entities.OrderStatusConfirmed,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
	}

	// Mock expectations
	mockRepo.On("List", ctx, mock.AnythingOfType("interface{}")).Return(expectedOrders, nil)

	// Execute
	orders, err := mockRepo.List(ctx, nil)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, orders, 2)
	assert.Equal(t, expectedOrders[0].OrderNumber, "TEST-001")
	assert.Equal(t, expectedOrders[1].OrderNumber, "TEST-002")
	mockRepo.AssertExpectations(t)
}
