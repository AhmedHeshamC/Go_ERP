package order

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/domain/orders/repositories"
)

func TestServiceImpl_CreateOrder(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrderRepo := &MockOrderRepository{}
	mockOrderItemRepo := &MockOrderItemRepository{}
	mockCustomerRepo := &MockCustomerRepository{}
	mockAddressRepo := &MockOrderAddressRepository{}
	mockCompanyRepo := &MockCompanyRepository{}
	mockAnalyticsRepo := &MockOrderAnalyticsRepository{}
	mockProductService := &MockProductService{}
	mockInventoryService := &MockInventoryService{}
	mockUserService := &MockUserService{}
	mockNotificationService := &MockNotificationService{}
	mockPaymentService := &MockPaymentService{}
	mockTaxCalculator := &MockTaxCalculator{}
	mockShippingCalculator := &MockShippingCalculator{}

	service := &ServiceImpl{
		orderRepo:          mockOrderRepo,
		orderItemRepo:      mockOrderItemRepo,
		customerRepo:       mockCustomerRepo,
		addressRepo:        mockAddressRepo,
		companyRepo:        mockCompanyRepo,
		analyticsRepo:      mockAnalyticsRepo,
		productService:     mockProductService,
		inventoryService:   mockInventoryService,
		userService:        mockUserService,
		notificationService: mockNotificationService,
		paymentService:     mockPaymentService,
		taxCalculator:      mockTaxCalculator,
		shippingCalculator: mockShippingCalculator,
		defaultCurrency:    "USD",
	}

	// Test data
	customerID := uuid.New()
	shippingAddressID := uuid.New()
	billingAddressID := uuid.New()
	productID := uuid.New()
	createdBy := uuid.New()

	customer := CreateTestCustomer(customerID)
	shippingAddress := CreateTestOrderAddress(shippingAddressID, customerID, "SHIPPING")
	billingAddress := CreateTestOrderAddress(billingAddressID, customerID, "BILLING")
	product := CreateTestProduct(productID)

	req := &CreateOrderRequest{
		CustomerID:        customerID.String(),
		Type:              entities.OrderTypeSales,
		Priority:          entities.OrderPriorityNormal,
		ShippingMethod:    entities.ShippingMethodStandard,
		ShippingAddressID: shippingAddressID.String(),
		BillingAddressID:  billingAddressID.String(),
		Currency:          "USD",
		Items: []CreateOrderItemRequest{
			{
				ProductID: productID.String(),
				Quantity:  2,
			},
		},
		CreatedBy: createdBy.String(),
	}

	t.Run("successful order creation", func(t *testing.T) {
		// Setup mocks
		mockCustomerRepo.On("GetByID", ctx, customerID).Return(customer, nil)
		mockAddressRepo.On("GetByID", ctx, shippingAddressID).Return(shippingAddress, nil)
		mockAddressRepo.On("GetByID", ctx, billingAddressID).Return(billingAddress, nil)
		mockOrderRepo.On("GenerateUniqueOrderNumber", ctx).Return("2024-001234", nil)
		mockProductService.On("GetProduct", ctx, productID.String()).Return(product, nil)
		mockInventoryService.On("CheckAvailability", ctx, mock.AnythingOfType("*order.CheckInventoryRequest")).Return(&CheckInventoryResponse{
			Available: true,
			Items: []CheckInventoryItemResponse{
				{
					ProductID: productID.String(),
					RequestedQty: 2,
					AvailableQty: 100,
					CanFulfill: true,
					UnitPrice: decimal.NewFromFloat(50.00),
					TotalValue: decimal.NewFromFloat(100.00),
				},
			},
		}, nil)
		mockTaxCalculator.On("CalculateTax", ctx, mock.AnythingOfType("*order.TaxCalculationRequest")).Return(&TaxCalculationResponse{
			TaxAmount: decimal.NewFromFloat(8.00),
		}, nil)
		mockShippingCalculator.On("CalculateShipping", ctx, mock.AnythingOfType("*order.ShippingCalculationRequest")).Return(&ShippingCalculationResponse{
			Cost: decimal.NewFromFloat(10.00),
		}, nil)

		// Mock transaction
		mockTx := &struct{}{}
		mockOrderRepo.On("BeginTransaction", ctx).Return(mockTx, nil)
		mockOrderRepo.On("Create", ctx, mock.AnythingOfType("*entities.Order")).Return(nil)
		mockOrderItemRepo.On("Create", ctx, mock.AnythingOfType("*entities.OrderItem")).Return(nil)
		mockInventoryService.On("ReserveInventory", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]order.ReserveItemRequest")).Return(nil)
		mockOrderRepo.On("UpdateCreditUsed", ctx, customerID, decimal.NewFromFloat(118.00)).Return(nil)
		mockOrderRepo.On("Commit", ctx).Return(nil)

		// Execute
		order, err := service.CreateOrder(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, entities.OrderStatusDraft, order.Status)
		assert.Equal(t, decimal.NewFromFloat(118.00), order.TotalAmount)
		assert.Equal(t, customerID, order.CustomerID)
		assert.Equal(t, "2024-001234", order.OrderNumber)

		// Verify all mocks were called
		mockCustomerRepo.AssertExpectations(t)
		mockAddressRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
		mockOrderItemRepo.AssertExpectations(t)
		mockProductService.AssertExpectations(t)
		mockInventoryService.AssertExpectations(t)
		mockTaxCalculator.AssertExpectations(t)
		mockShippingCalculator.AssertExpectations(t)
	})

	t.Run("customer not found", func(t *testing.T) {
		// Setup mocks
		mockCustomerRepo.On("GetByID", ctx, customerID).Return(nil, repositories.ErrCustomerNotFound)

		// Execute
		order, err := service.CreateOrder(ctx, req)

		// Assert
		require.Error(t, err)
		assert.Nil(t, order)
		assert.Contains(t, err.Error(), "customer not found")

		mockCustomerRepo.AssertExpectations(t)
	})

	t.Run("invalid customer ID", func(t *testing.T) {
		// Setup
		invalidReq := *req
		invalidReq.CustomerID = "invalid-uuid"

		// Execute
		order, err := service.CreateOrder(ctx, &invalidReq)

		// Assert
		require.Error(t, err)
		assert.Nil(t, order)
		assert.Contains(t, err.Error(), "invalid customer ID")
	})

	t.Run("insufficient inventory", func(t *testing.T) {
		// Setup mocks
		mockCustomerRepo.On("GetByID", ctx, customerID).Return(customer, nil)
		mockAddressRepo.On("GetByID", ctx, shippingAddressID).Return(shippingAddress, nil)
		mockAddressRepo.On("GetByID", ctx, billingAddressID).Return(billingAddress, nil)
		mockOrderRepo.On("GenerateUniqueOrderNumber", ctx).Return("2024-001234", nil)
		mockProductService.On("GetProduct", ctx, productID.String()).Return(product, nil)
		mockInventoryService.On("CheckAvailability", ctx, mock.AnythingOfType("*order.CheckInventoryRequest")).Return(&CheckInventoryResponse{
			Available: false,
			Reason: "Insufficient stock",
		}, nil)

		// Execute
		order, err := service.CreateOrder(ctx, req)

		// Assert
		require.Error(t, err)
		assert.Nil(t, order)
		assert.Contains(t, err.Error(), "inventory check failed")

		mockCustomerRepo.AssertExpectations(t)
		mockAddressRepo.AssertExpectations(t)
		mockProductService.AssertExpectations(t)
		mockInventoryService.AssertExpectations(t)
	})

	t.Run("customer credit limit exceeded", func(t *testing.T) {
		// Setup customer with low credit limit
		lowCreditCustomer := CreateTestCustomer(customerID)
		lowCreditCustomer.CreditLimit = decimal.NewFromFloat(50.00)

		// Setup mocks
		mockCustomerRepo.On("GetByID", ctx, customerID).Return(lowCreditCustomer, nil)
		mockAddressRepo.On("GetByID", ctx, shippingAddressID).Return(shippingAddress, nil)
		mockAddressRepo.On("GetByID", ctx, billingAddressID).Return(billingAddress, nil)
		mockOrderRepo.On("GenerateUniqueOrderNumber", ctx).Return("2024-001234", nil)
		mockProductService.On("GetProduct", ctx, productID.String()).Return(product, nil)
		mockInventoryService.On("CheckAvailability", ctx, mock.AnythingOfType("*order.CheckInventoryRequest")).Return(&CheckInventoryResponse{
			Available: true,
		}, nil)
		mockTaxCalculator.On("CalculateTax", ctx, mock.AnythingOfType("*order.TaxCalculationRequest")).Return(&TaxCalculationResponse{
			TaxAmount: decimal.NewFromFloat(8.00),
		}, nil)
		mockShippingCalculator.On("CalculateShipping", ctx, mock.AnythingOfType("*order.ShippingCalculationRequest")).Return(&ShippingCalculationResponse{
			Cost: decimal.NewFromFloat(10.00),
		}, nil)

		// Execute
		order, err := service.CreateOrder(ctx, req)

		// Assert
		require.Error(t, err)
		assert.Nil(t, order)
		assert.Contains(t, err.Error(), "insufficient customer credit")

		mockCustomerRepo.AssertExpectations(t)
	})
}

func TestServiceImpl_GetOrder(t *testing.T) {
	ctx := context.Background()
	mockOrderRepo := &MockOrderRepository{}
	mockOrderItemRepo := &MockOrderItemRepository{}
	mockCustomerRepo := &MockCustomerRepository{}
	mockAddressRepo := &MockOrderAddressRepository{}

	service := &ServiceImpl{
		orderRepo:     mockOrderRepo,
		orderItemRepo: mockOrderItemRepo,
		customerRepo:  mockCustomerRepo,
		addressRepo:   mockAddressRepo,
	}

	orderID := uuid.New()
	customerID := uuid.New()
	shippingAddressID := uuid.New()
	productID := uuid.New()

	order := CreateTestOrder(orderID)
	order.CustomerID = customerID
	order.ShippingAddressID = shippingAddressID

	customer := CreateTestCustomer(customerID)
	shippingAddress := CreateTestOrderAddress(shippingAddressID, customerID, "SHIPPING")
	orderItem := CreateTestOrderItem(uuid.New(), orderID, productID)

	t.Run("successful order retrieval", func(t *testing.T) {
		// Setup mocks
		mockOrderRepo.On("GetByID", ctx, orderID).Return(order, nil)
		mockOrderItemRepo.On("GetByOrderID", ctx, orderID).Return([]*entities.OrderItem{orderItem}, nil)
		mockCustomerRepo.On("GetByID", ctx, customerID).Return(customer, nil)
		mockAddressRepo.On("GetByID", ctx, shippingAddressID).Return(shippingAddress, nil)

		// Execute
		result, err := service.GetOrder(ctx, orderID.String())

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, orderID, result.ID)
		assert.Equal(t, customerID, result.Customer.ID)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, shippingAddressID, result.ShippingAddress.ID)

		mockOrderRepo.AssertExpectations(t)
		mockOrderItemRepo.AssertExpectations(t)
		mockCustomerRepo.AssertExpectations(t)
		mockAddressRepo.AssertExpectations(t)
	})

	t.Run("order not found", func(t *testing.T) {
		// Setup mocks
		mockOrderRepo.On("GetByID", ctx, orderID).Return(nil, repositories.ErrOrderNotFound)

		// Execute
		result, err := service.GetOrder(ctx, orderID.String())

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrOrderNotFound, err)

		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("invalid order ID", func(t *testing.T) {
		// Execute
		result, err := service.GetOrder(ctx, "invalid-uuid")

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid order ID")
	})
}

func TestServiceImpl_UpdateOrderStatus(t *testing.T) {
	ctx := context.Background()
	mockOrderRepo := &MockOrderRepository{}
	mockOrderItemRepo := &MockOrderItemRepository{}
	mockCustomerRepo := &MockCustomerRepository{}
	mockAddressRepo := &MockOrderAddressRepository{}
	mockInventoryService := &MockInventoryService{}
	mockNotificationService := &MockNotificationService{}

	service := &ServiceImpl{
		orderRepo:          mockOrderRepo,
		orderItemRepo:      mockOrderItemRepo,
		customerRepo:       mockCustomerRepo,
		addressRepo:        mockAddressRepo,
		inventoryService:   mockInventoryService,
		notificationService: mockNotificationService,
	}

	orderID := uuid.New()
	customerID := uuid.New()
	updatedByID := uuid.New()

	order := CreateTestOrder(orderID)
	order.Status = entities.OrderStatusPending
	order.CustomerID = customerID

	customer := CreateTestCustomer(customerID)

	req := &UpdateOrderStatusRequest{
		Status:    entities.OrderStatusConfirmed,
		Reason:    "Order approved",
		Notify:    true,
		UpdatedBy: updatedByID.String(),
	}

	t.Run("successful status update", func(t *testing.T) {
		// Setup mocks
		mockOrderRepo.On("GetByID", ctx, orderID).Return(order, nil)
		mockOrderItemRepo.On("GetByOrderID", ctx, orderID).Return([]*entities.OrderItem{}, nil)
		mockCustomerRepo.On("GetByID", ctx, customerID).Return(customer, nil)
		mockAddressRepo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(&entities.OrderAddress{}, nil).Maybe()
		mockOrderRepo.On("Update", ctx, mock.AnythingOfType("*entities.Order")).Return(nil)
		mockNotificationService.On("SendOrderNotification", ctx, mock.AnythingOfType("*order.OrderNotificationRequest")).Return(nil)

		// Execute
		result, err := service.UpdateOrderStatus(ctx, orderID.String(), req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, entities.OrderStatusConfirmed, result.Status)

		mockOrderRepo.AssertExpectations(t)
		mockCustomerRepo.AssertExpectations(t)
		mockNotificationService.AssertExpectations(t)
	})

	t.Run("invalid status transition", func(t *testing.T) {
		// Setup
		invalidReq := *req
		invalidReq.Status = entities.OrderStatusDelivered // Can't go from Pending to Delivered

		// Setup mocks
		mockOrderRepo.On("GetByID", ctx, orderID).Return(order, nil)
		mockOrderItemRepo.On("GetByOrderID", ctx, orderID).Return([]*entities.OrderItem{}, nil)
		mockCustomerRepo.On("GetByID", ctx, customerID).Return(customer, nil)
		mockAddressRepo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(&entities.OrderAddress{}, nil).Maybe()

		// Execute
		result, err := service.UpdateOrderStatus(ctx, orderID.String(), &invalidReq)

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid status transition")

		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("order shipped - release inventory", func(t *testing.T) {
		// Setup
		shipReq := *req
		shipReq.Status = entities.OrderStatusShipped

		// Setup mocks
		mockOrderRepo.On("GetByID", ctx, orderID).Return(order, nil)
		mockOrderItemRepo.On("GetByOrderID", ctx, orderID).Return([]*entities.OrderItem{}, nil)
		mockCustomerRepo.On("GetByID", ctx, customerID).Return(customer, nil)
		mockAddressRepo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(&entities.OrderAddress{}, nil).Maybe()
		mockOrderRepo.On("Update", ctx, mock.AnythingOfType("*entities.Order")).Return(nil)
		mockInventoryService.On("ReleaseReservation", ctx, orderID.String()).Return(nil)
		mockNotificationService.On("SendOrderNotification", ctx, mock.AnythingOfType("*order.OrderNotificationRequest")).Return(nil)

		// Execute
		result, err := service.UpdateOrderStatus(ctx, orderID.String(), &shipReq)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, entities.OrderStatusShipped, result.Status)

		mockOrderRepo.AssertExpectations(t)
		mockInventoryService.AssertExpectations(t)
		mockNotificationService.AssertExpectations(t)
	})
}

func TestServiceImpl_CancelOrder(t *testing.T) {
	ctx := context.Background()
	mockOrderRepo := &MockOrderRepository{}
	mockOrderItemRepo := &MockOrderItemRepository{}
	mockCustomerRepo := &MockCustomerRepository{}
	mockAddressRepo := &MockOrderAddressRepository{}
	mockInventoryService := &MockInventoryService{}
	mockPaymentService := &MockPaymentService{}
	mockNotificationService := &MockNotificationService{}

	service := &ServiceImpl{
		orderRepo:          mockOrderRepo,
		orderItemRepo:      mockOrderItemRepo,
		customerRepo:       mockCustomerRepo,
		addressRepo:        mockAddressRepo,
		inventoryService:   mockInventoryService,
		paymentService:     mockPaymentService,
		notificationService: mockNotificationService,
	}

	orderID := uuid.New()
	customerID := uuid.New()
	cancelledByID := uuid.New()

	order := CreateTestOrder(orderID)
	order.Status = entities.OrderStatusPending
	order.CustomerID = customerID

	customer := CreateTestCustomer(customerID)

	req := &CancelOrderRequest{
		Reason:      "Customer requested cancellation",
		Refund:      true,
		Notify:      true,
		CancelledBy: cancelledByID.String(),
	}

	t.Run("successful order cancellation", func(t *testing.T) {
		// Setup mocks
		mockOrderRepo.On("GetByID", ctx, orderID).Return(order, nil)
		mockOrderItemRepo.On("GetByOrderID", ctx, orderID).Return([]*entities.OrderItem{}, nil)
		mockCustomerRepo.On("GetByID", ctx, customerID).Return(customer, nil)
		mockAddressRepo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(&entities.OrderAddress{}, nil).Maybe()
		mockPaymentService.On("ProcessRefund", ctx, mock.AnythingOfType("*order.RefundProcessRequest")).Return(&RefundResponse{
			Success:       true,
			TransactionID: "refund-123",
			Amount:        decimal.Zero,
			Status:        "completed",
		}, nil)
		mockInventoryService.On("ReleaseReservation", ctx, orderID.String()).Return(nil)
		mockCustomerRepo.On("UpdateCreditUsed", ctx, customerID, decimal.NewFromFloat(-118.00)).Return(nil)
		mockOrderRepo.On("Update", ctx, mock.AnythingOfType("*entities.Order")).Return(nil)
		mockNotificationService.On("SendOrderNotification", ctx, mock.AnythingOfType("*order.OrderNotificationRequest")).Return(nil)

		// Execute
		result, err := service.CancelOrder(ctx, orderID.String(), req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, entities.OrderStatusCancelled, result.Status)

		mockOrderRepo.AssertExpectations(t)
		mockPaymentService.AssertExpectations(t)
		mockInventoryService.AssertExpectations(t)
		mockCustomerRepo.AssertExpectations(t)
		mockNotificationService.AssertExpectations(t)
	})

	t.Run("order cannot be cancelled", func(t *testing.T) {
		// Setup order in terminal status
		completedOrder := CreateTestOrder(orderID)
		completedOrder.Status = entities.OrderStatusDelivered

		// Setup mocks
		mockOrderRepo.On("GetByID", ctx, orderID).Return(completedOrder, nil)

		// Execute
		result, err := service.CancelOrder(ctx, orderID.String(), req)

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrOrderCannotBeCancelled, err)

		mockOrderRepo.AssertExpectations(t)
	})
}

func TestServiceImpl_ValidateOrder(t *testing.T) {
	ctx := context.Background()
	mockOrderRepo := &MockOrderRepository{}
	mockOrderItemRepo := &MockOrderItemRepository{}
	mockCustomerRepo := &MockCustomerRepository{}
	mockAddressRepo := &MockOrderAddressRepository{}
	mockInventoryService := &MockInventoryService{}

	service := &ServiceImpl{
		orderRepo:        mockOrderRepo,
		orderItemRepo:    mockOrderItemRepo,
		customerRepo:     mockCustomerRepo,
		addressRepo:      mockAddressRepo,
		inventoryService: mockInventoryService,
	}

	orderID := uuid.New()
	customerID := uuid.New()
	productID := uuid.New()

	order := CreateTestOrder(orderID)
	order.CustomerID = customerID

	customer := CreateTestCustomer(customerID)
	customer.CreditLimit = decimal.NewFromFloat(200.00) // Higher than order total

	orderItem := CreateTestOrderItem(uuid.New(), orderID, productID)

	t.Run("valid order", func(t *testing.T) {
		// Setup mocks
		mockOrderRepo.On("GetByID", ctx, orderID).Return(order, nil)
		mockOrderItemRepo.On("GetByOrderID", ctx, orderID).Return([]*entities.OrderItem{orderItem}, nil)
		mockCustomerRepo.On("GetByID", ctx, customerID).Return(customer, nil)
		mockInventoryService.On("CheckAvailability", ctx, mock.AnythingOfType("*order.CheckInventoryRequest")).Return(&CheckInventoryResponse{
			Available: true,
		}, nil)

		// Execute
		result, err := service.ValidateOrder(ctx, orderID.String())

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsValid)
		assert.Empty(t, result.Errors)

		mockOrderRepo.AssertExpectations(t)
		mockOrderItemRepo.AssertExpectations(t)
		mockCustomerRepo.AssertExpectations(t)
		mockInventoryService.AssertExpectations(t)
	})

	t.Run("insufficient customer credit", func(t *testing.T) {
		// Setup customer with low credit limit
		lowCreditCustomer := CreateTestCustomer(customerID)
		lowCreditCustomer.CreditLimit = decimal.NewFromFloat(50.00)

		// Setup mocks
		mockOrderRepo.On("GetByID", ctx, orderID).Return(order, nil)
		mockOrderItemRepo.On("GetByOrderID", ctx, orderID).Return([]*entities.OrderItem{orderItem}, nil)
		mockCustomerRepo.On("GetByID", ctx, customerID).Return(lowCreditCustomer, nil)
		mockInventoryService.On("CheckAvailability", ctx, mock.AnythingOfType("*order.CheckInventoryRequest")).Return(&CheckInventoryResponse{
			Available: true,
		}, nil)

		// Execute
		result, err := service.ValidateOrder(ctx, orderID.String())

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsValid)
		assert.NotEmpty(t, result.Errors)
		assert.Contains(t, result.Errors[0], "insufficient customer credit")

		mockOrderRepo.AssertExpectations(t)
		mockCustomerRepo.AssertExpectations(t)
	})

	t.Run("inventory unavailable", func(t *testing.T) {
		// Setup mocks
		mockOrderRepo.On("GetByID", ctx, orderID).Return(order, nil)
		mockOrderItemRepo.On("GetByOrderID", ctx, orderID).Return([]*entities.OrderItem{orderItem}, nil)
		mockCustomerRepo.On("GetByID", ctx, customerID).Return(customer, nil)
		mockInventoryService.On("CheckAvailability", ctx, mock.AnythingOfType("*order.CheckInventoryRequest")).Return(&CheckInventoryResponse{
			Available: false,
			Reason:    "Out of stock",
		}, nil)

		// Execute
		result, err := service.ValidateOrder(ctx, orderID.String())

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsValid)
		assert.Contains(t, result.Errors, "Insufficient inventory for some items")

		mockOrderRepo.AssertExpectations(t)
		mockInventoryService.AssertExpectations(t)
	})
}

func TestServiceImpl_ListOrders(t *testing.T) {
	ctx := context.Background()
	mockOrderRepo := &MockOrderRepository{}
	mockOrderItemRepo := &MockOrderItemRepository{}
	mockCustomerRepo := &MockCustomerRepository{}

	service := &ServiceImpl{
		orderRepo:     mockOrderRepo,
		orderItemRepo: mockOrderItemRepo,
		customerRepo:  mockCustomerRepo,
	}

	req := &ListOrdersRequest{
		Page:  1,
		Limit: 20,
	}

	orders := []*entities.Order{
		CreateTestOrder(uuid.New()),
		CreateTestOrder(uuid.New()),
	}

	t.Run("successful order listing", func(t *testing.T) {
		// Setup mocks
		mockOrderRepo.On("List", ctx, mock.AnythingOfType("repositories.OrderFilter")).Return(orders, nil)
		mockOrderRepo.On("Count", ctx, mock.AnythingOfType("repositories.OrderFilter")).Return(2, nil)
		mockOrderItemRepo.On("GetByOrderID", ctx, mock.AnythingOfType("uuid.UUID")).Return([]*entities.OrderItem{}, nil)
		mockCustomerRepo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(&entities.Customer{}, nil)

		// Execute
		result, err := service.ListOrders(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Orders, 2)
		assert.NotNil(t, result.Pagination)
		assert.Equal(t, 1, result.Pagination.Page)
		assert.Equal(t, 20, result.Pagination.Limit)
		assert.Equal(t, 2, result.Pagination.Total)

		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("empty order list", func(t *testing.T) {
		// Setup mocks
		mockOrderRepo.On("List", ctx, mock.AnythingOfType("repositories.OrderFilter")).Return([]*entities.Order{}, nil)
		mockOrderRepo.On("Count", ctx, mock.AnythingOfType("repositories.OrderFilter")).Return(0, nil)

		// Execute
		result, err := service.ListOrders(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Orders)
		assert.Equal(t, 0, result.Pagination.Total)

		mockOrderRepo.AssertExpectations(t)
	})
}

// Benchmark tests
func BenchmarkServiceImpl_CreateOrder(b *testing.B) {
	ctx := context.Background()
	mockOrderRepo := &MockOrderRepository{}
	mockOrderItemRepo := &MockOrderItemRepository{}
	mockCustomerRepo := &MockCustomerRepository{}
	mockAddressRepo := &MockOrderAddressRepository{}
	mockCompanyRepo := &MockCompanyRepository{}
	mockAnalyticsRepo := &MockOrderAnalyticsRepository{}
	mockProductService := &MockProductService{}
	mockInventoryService := &MockInventoryService{}
	mockUserService := &MockUserService{}
	mockNotificationService := &MockNotificationService{}
	mockPaymentService := &MockPaymentService{}
	mockTaxCalculator := &MockTaxCalculator{}
	mockShippingCalculator := &MockShippingCalculator{}

	service := &ServiceImpl{
		orderRepo:          mockOrderRepo,
		orderItemRepo:      mockOrderItemRepo,
		customerRepo:       mockCustomerRepo,
		addressRepo:        mockAddressRepo,
		companyRepo:        mockCompanyRepo,
		analyticsRepo:      mockAnalyticsRepo,
		productService:     mockProductService,
		inventoryService:   mockInventoryService,
		userService:        mockUserService,
		notificationService: mockNotificationService,
		paymentService:     mockPaymentService,
		taxCalculator:      mockTaxCalculator,
		shippingCalculator: mockShippingCalculator,
		defaultCurrency:    "USD",
	}

	// Setup test data
	customerID := uuid.New()
	shippingAddressID := uuid.New()
	billingAddressID := uuid.New()
	productID := uuid.New()
	createdBy := uuid.New()

	customer := CreateTestCustomer(customerID)
	shippingAddress := CreateTestOrderAddress(shippingAddressID, customerID, "SHIPPING")
	billingAddress := CreateTestOrderAddress(billingAddressID, customerID, "BILLING")
	product := CreateTestProduct(productID)

	req := &CreateOrderRequest{
		CustomerID:        customerID.String(),
		Type:              entities.OrderTypeSales,
		Priority:          entities.OrderPriorityNormal,
		ShippingMethod:    entities.ShippingMethodStandard,
		ShippingAddressID: shippingAddressID.String(),
		BillingAddressID:  billingAddressID.String(),
		Currency:          "USD",
		Items: []CreateOrderItemRequest{
			{
				ProductID: productID.String(),
				Quantity:  2,
			},
		},
		CreatedBy: createdBy.String(),
	}

	// Setup mocks
	mockCustomerRepo.On("GetByID", ctx, customerID).Return(customer, nil)
	mockAddressRepo.On("GetByID", ctx, shippingAddressID).Return(shippingAddress, nil)
	mockAddressRepo.On("GetByID", ctx, billingAddressID).Return(billingAddress, nil)
	mockOrderRepo.On("GenerateUniqueOrderNumber", ctx).Return("2024-001234", nil)
	mockProductService.On("GetProduct", ctx, productID.String()).Return(product, nil)
	mockInventoryService.On("CheckAvailability", ctx, mock.AnythingOfType("*order.CheckInventoryRequest")).Return(&CheckInventoryResponse{
		Available: true,
	}, nil)
	mockTaxCalculator.On("CalculateTax", ctx, mock.AnythingOfType("*order.TaxCalculationRequest")).Return(&TaxCalculationResponse{
		TaxAmount: decimal.NewFromFloat(8.00),
	}, nil)
	mockShippingCalculator.On("CalculateShipping", ctx, mock.AnythingOfType("*order.ShippingCalculationRequest")).Return(&ShippingCalculationResponse{
		Cost: decimal.NewFromFloat(10.00),
	}, nil)
	mockOrderRepo.On("BeginTransaction", ctx).Return(&struct{}{}, nil)
	mockOrderRepo.On("Create", ctx, mock.AnythingOfType("*entities.Order")).Return(nil)
	mockOrderItemRepo.On("Create", ctx, mock.AnythingOfType("*entities.OrderItem")).Return(nil)
	mockInventoryService.On("ReserveInventory", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]order.ReserveItemRequest")).Return(nil)
	mockOrderRepo.On("UpdateCreditUsed", ctx, customerID, decimal.NewFromFloat(118.00)).Return(nil)
	mockOrderRepo.On("Commit", ctx).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.CreateOrder(ctx, req)
		if err != nil {
			b.Fatalf("CreateOrder failed: %v", err)
		}
	}
}