package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erpgo/internal/application/services/order"
	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/domain/orders/repositories"
	"erpgo/internal/interfaces/http/dto"
	"erpgo/internal/interfaces/http/handlers"
	"erpgo/internal/interfaces/http/routes"
	"erpgo/tests/integration/testutil"
)

func TestOrderRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Mock order service
	mockOrderService := &testutil.MockOrderService{}

	// Setup test logger
	logger := testutil.NewTestLogger(t)

	// Setup Gin router
	router := gin.New()
	api := router.Group("/api/v1")
	routes.SetupOrderRoutes(api, mockOrderService, logger)

	// Test data
	testCustomerID := uuid.New()
	testShippingAddressID := uuid.New()
	testBillingAddressID := uuid.New()
	testProductID := uuid.New()
	testOrderID := uuid.New()
	testOrderNumber := "ORD-12345"

	// Create test order entity
	testOrder := &entities.Order{
		ID:                 testOrderID,
		OrderNumber:        testOrderNumber,
		CustomerID:         testCustomerID,
		CustomerName:       "John Doe",
		Type:               entities.OrderTypeSales,
		Status:             entities.OrderStatusDraft,
		Priority:           entities.OrderPriorityNormal,
		PaymentStatus:      entities.PaymentStatusPending,
		FulfillmentStatus:  entities.FulfillmentStatusUnfulfilled,
		Currency:           "USD",
		Subtotal:           decimal.NewFromFloat(100.00),
		TaxAmount:          decimal.NewFromFloat(8.00),
		ShippingAmount:     decimal.NewFromFloat(10.00),
		DiscountAmount:     decimal.NewFromFloat(0.00),
		TotalAmount:        decimal.NewFromFloat(118.00),
		PaidAmount:         decimal.Zero,
		RefundedAmount:     decimal.Zero,
		Weight:             decimal.NewFromFloat(1.5),
		ShippingMethod:     entities.ShippingMethodStandard,
		TrackingNumber:     "",
		Notes:              stringPtr("Test order notes"),
		CustomerNotes:      stringPtr("Customer notes"),
		InternalNotes:      stringPtr("Internal notes"),
		RequiredDate:       &time.Time{},
		ShippedDate:        nil,
		DeliveredDate:      nil,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		ApprovedAt:         nil,
		ApprovedBy:         nil,
		CancelledAt:        nil,
		CancelledBy:        nil,
		CancellationReason: nil,
		Items: []entities.OrderItem{
			{
				ID:              uuid.New(),
				ProductID:       testProductID,
				ProductSKU:      "TEST-001",
				ProductName:     "Test Product",
				VariantID:       nil,
				VariantName:     nil,
				Quantity:        2,
				UnitPrice:       decimal.NewFromFloat(50.00),
				TotalPrice:      decimal.NewFromFloat(100.00),
				TaxAmount:       decimal.NewFromFloat(8.00),
				DiscountAmount:  decimal.Zero,
				FinalPrice:      decimal.NewFromFloat(108.00),
				Weight:          decimal.NewFromFloat(1.5),
				Status:          entities.OrderItemStatusPending,
				ShippedQuantity: 0,
				ReturnedQuantity: 0,
				Notes:           nil,
			},
		},
	}

	// Create test order pagination response
	testOrderListResponse := &order.ListOrdersResponse{
		Orders: []*entities.Order{testOrder},
		Pagination: &order.Pagination{
			Page:       1,
			Limit:      20,
			Total:      1,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		},
	}

	// Create test order stats
	testOrderStats := &repositories.OrderStats{
		TotalOrders:         1,
		TotalRevenue:        decimal.NewFromFloat(118.00),
		OrdersByStatus:      map[string]int64{"DRAFT": 1},
		OrdersByType:        map[string]int64{"SALES": 1},
		TopProducts:         []repositories.ProductSalesStats{},
		TopCustomers:        []repositories.CustomerOrderStats{},
		AverageOrderValue:   decimal.NewFromFloat(118.00),
	}

	t.Run("GetOrders - Success", func(t *testing.T) {
		// Setup mock
		mockOrderService.On("ListOrders", mock.Anything, mock.AnythingOfType("*order.ListOrdersRequest")).
			Return(testOrderListResponse, nil).Once()

		// Make request
		req, _ := http.NewRequest("GET", "/api/v1/orders", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ListOrdersResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Orders, 1)
		assert.Equal(t, testOrderNumber, response.Orders[0].OrderNumber)
		assert.Equal(t, "John Doe", response.Orders[0].CustomerName)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("GetOrders - With Filters", func(t *testing.T) {
		// Setup mock
		mockOrderService.On("ListOrders", mock.Anything, mock.MatchedBy(func(req *order.ListOrdersRequest) bool {
			return req.CustomerID != nil && *req.CustomerID == testCustomerID &&
				req.Status != nil && *req.Status == "DRAFT" &&
				req.Page == 2 && req.Limit == 10
		})).Return(testOrderListResponse, nil).Once()

		// Make request with filters
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/orders?customer_id=%s&status=DRAFT&page=2&limit=10", testCustomerID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ListOrdersResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Orders, 1)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("GetOrder - Success", func(t *testing.T) {
		// Setup mock
		mockOrderService.On("GetOrder", mock.Anything, testOrderID.String()).
			Return(testOrder, nil).Once()

		// Make request
		req, _ := http.NewRequest("GET", "/api/v1/orders/"+testOrderID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.OrderResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testOrderNumber, response.OrderNumber)
		assert.Equal(t, "John Doe", response.CustomerName)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("GetOrder - Not Found", func(t *testing.T) {
		// Setup mock
		mockOrderService.On("GetOrder", mock.Anything, mock.AnythingOfType("string")).
			Return(nil, fmt.Errorf("order not found")).Once()

		// Make request
		req, _ := http.NewRequest("GET", "/api/v1/orders/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response dto.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Order not found", response.Error)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("CreateOrder - Success", func(t *testing.T) {
		// Setup mock
		mockOrderService.On("CreateOrder", mock.Anything, mock.MatchedBy(func(req *order.CreateOrderRequest) bool {
			return req.CustomerID == testCustomerID.String() &&
				req.Type == entities.OrderTypeSales &&
				len(req.Items) == 1 &&
				req.Items[0].ProductID == testProductID.String()
		})).Return(testOrder, nil).Once()

		// Create request body
		createReq := dto.OrderRequest{
			CustomerID:        testCustomerID,
			Type:              "SALES",
			Priority:          "NORMAL",
			ShippingMethod:    "STANDARD",
			ShippingAddressID: testShippingAddressID,
			BillingAddressID:  testBillingAddressID,
			Currency:          "USD",
			Notes:             stringPtr("Test order notes"),
			CustomerNotes:     stringPtr("Customer notes"),
			Items: []dto.OrderItemRequest{
				{
					ProductID: testProductID,
					Quantity:  2,
					UnitPrice: decimal.NewFromFloat(50.00),
				},
			},
		}

		reqBody, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.OrderResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testOrderNumber, response.OrderNumber)
		assert.Equal(t, "John Doe", response.CustomerName)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("CreateOrder - Invalid Request", func(t *testing.T) {
		// Create invalid request body (missing required fields)
		createReq := dto.OrderRequest{
			Type: "SALES", // Missing CustomerID, ShippingMethod, etc.
		}

		reqBody, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response dto.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Invalid request body", response.Error)
	})

	t.Run("UpdateOrder - Success", func(t *testing.T) {
		// Setup mock
		updatedOrder := *testOrder
		updatedOrder.Notes = stringPtr("Updated notes")

		mockOrderService.On("UpdateOrder", mock.Anything, testOrderID.String(), mock.MatchedBy(func(req *order.UpdateOrderRequest) bool {
			return req.Notes != nil && *req.Notes == "Updated notes"
		})).Return(&updatedOrder, nil).Once()

		// Create request body
		updateReq := dto.UpdateOrderRequest{
			Notes: stringPtr("Updated notes"),
		}

		reqBody, _ := json.Marshal(updateReq)
		req, _ := http.NewRequest("PUT", "/api/v1/orders/"+testOrderID.String(), bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.OrderResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Updated notes", *response.Notes)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("DeleteOrder - Success", func(t *testing.T) {
		// Setup mock
		mockOrderService.On("DeleteOrder", mock.Anything, testOrderID.String()).
			Return(nil).Once()

		// Make request
		req, _ := http.NewRequest("DELETE", "/api/v1/orders/"+testOrderID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Order deleted successfully", response.Message)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("UpdateOrderStatus - Success", func(t *testing.T) {
		// Setup mock
		updatedOrder := *testOrder
		updatedOrder.Status = entities.OrderStatusConfirmed

		mockOrderService.On("UpdateOrderStatus", mock.Anything, testOrderID.String(), mock.MatchedBy(func(req *order.UpdateOrderStatusRequest) bool {
			return req.Status == entities.OrderStatusConfirmed && req.NotifyCustomer == true
		})).Return(&updatedOrder, nil).Once()

		// Create request body
		statusReq := dto.UpdateOrderStatusRequest{
			Status:         "CONFIRMED",
			NotifyCustomer: true,
		}

		reqBody, _ := json.Marshal(statusReq)
		req, _ := http.NewRequest("PUT", "/api/v1/orders/"+testOrderID.String()+"/status", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.OrderResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "CONFIRMED", response.Status)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("CancelOrder - Success", func(t *testing.T) {
		// Setup mock
		cancelledOrder := *testOrder
		cancelledOrder.Status = entities.OrderStatusCancelled
		cancelledOrder.CancellationReason = stringPtr("Customer requested cancellation")

		mockOrderService.On("CancelOrder", mock.Anything, testOrderID.String(), mock.MatchedBy(func(req *order.CancelOrderRequest) bool {
			return req.Reason == "Customer requested cancellation" &&
				req.NotifyCustomer == true &&
				req.RefundPayment == true &&
				req.RestockItems == true
		})).Return(&cancelledOrder, nil).Once()

		// Create request body
		cancelReq := dto.CancelOrderRequest{
			Reason:          "Customer requested cancellation",
			NotifyCustomer:  true,
			RefundPayment:   true,
			RestockItems:    true,
		}

		reqBody, _ := json.Marshal(cancelReq)
		req, _ := http.NewRequest("POST", "/api/v1/orders/"+testOrderID.String()+"/cancel", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.OrderResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "CANCELLED", response.Status)
		assert.Equal(t, "Customer requested cancellation", *response.CancellationReason)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("ProcessOrder - Success", func(t *testing.T) {
		// Setup mock
		processedOrder := *testOrder
		processedOrder.Status = entities.OrderStatusProcessing

		mockOrderService.On("ProcessOrder", mock.Anything, testOrderID.String()).
			Return(&processedOrder, nil).Once()

		// Create request body
		processReq := dto.ProcessOrderRequest{
			NotifyCustomer: true,
		}

		reqBody, _ := json.Marshal(processReq)
		req, _ := http.NewRequest("POST", "/api/v1/orders/"+testOrderID.String()+"/process", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.OrderResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "PROCESSING", response.Status)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("ShipOrder - Success", func(t *testing.T) {
		// Setup mock
		shippedOrder := *testOrder
		shippedOrder.Status = entities.OrderStatusShipped
		shippedOrder.TrackingNumber = "TRACK123456"
		now := time.Now()
		shippedOrder.ShippedDate = &now

		mockOrderService.On("ShipOrder", mock.Anything, testOrderID.String(), mock.MatchedBy(func(req *order.ShipOrderRequest) bool {
			return req.TrackingNumber == "TRACK123456" &&
				req.Carrier == "FedEx" &&
				req.NotifyCustomer == true
		})).Return(&shippedOrder, nil).Once()

		// Create request body
		shipReq := dto.ShipOrderRequest{
			TrackingNumber: "TRACK123456",
			Carrier:        "FedEx",
			NotifyCustomer: true,
		}

		reqBody, _ := json.Marshal(shipReq)
		req, _ := http.NewRequest("POST", "/api/v1/orders/"+testOrderID.String()+"/ship", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.OrderResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "SHIPPED", response.Status)
		assert.Equal(t, "TRACK123456", response.TrackingNumber)
		assert.NotNil(t, response.ShippedDate)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("DeliverOrder - Success", func(t *testing.T) {
		// Setup mock
		deliveredOrder := *testOrder
		deliveredOrder.Status = entities.OrderStatusDelivered
		now := time.Now()
		deliveredOrder.DeliveredDate = &now

		mockOrderService.On("DeliverOrder", mock.Anything, testOrderID.String(), mock.MatchedBy(func(req *order.DeliverOrderRequest) bool {
			return req.Signature != nil && *req.Signature == "John Doe" &&
				req.NotifyCustomer == true
		})).Return(&deliveredOrder, nil).Once()

		// Create request body
		deliverReq := dto.DeliverOrderRequest{
			Signature:     stringPtr("John Doe"),
			NotifyCustomer: true,
		}

		reqBody, _ := json.Marshal(deliverReq)
		req, _ := http.NewRequest("POST", "/api/v1/orders/"+testOrderID.String()+"/deliver", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.OrderResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "DELIVERED", response.Status)
		assert.NotNil(t, response.DeliveredDate)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("SearchOrders - Success", func(t *testing.T) {
		// Setup mock
		searchResponse := &order.SearchOrdersResponse{
			Orders:     []*entities.Order{testOrder},
			Pagination: testOrderListResponse.Pagination,
			TotalCount: 1,
		}

		mockOrderService.On("SearchOrders", mock.Anything, mock.MatchedBy(func(req *order.SearchOrdersRequest) bool {
			return req.Query == "John Doe" && req.Page == 1 && req.Limit == 20
		})).Return(searchResponse, nil).Once()

		// Create request body
		searchReq := dto.SearchOrdersRequest{
			Query: "John Doe",
		}

		reqBody, _ := json.Marshal(searchReq)
		req, _ := http.NewRequest("POST", "/api/v1/orders/search", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.SearchOrdersResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Orders, 1)
		assert.Equal(t, 1, response.TotalCount)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("GetOrderByNumber - Success", func(t *testing.T) {
		// Setup mock
		mockOrderService.On("GetOrderByNumber", mock.Anything, testOrderNumber).
			Return(testOrder, nil).Once()

		// Make request
		req, _ := http.NewRequest("GET", "/api/v1/orders/by-number/"+testOrderNumber, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.OrderResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testOrderNumber, response.OrderNumber)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("GetOrderStats - Success", func(t *testing.T) {
		// Setup mock
		mockOrderService.On("GetOrderStats", mock.Anything, mock.AnythingOfType("*order.GetOrderStatsRequest")).
			Return(testOrderStats, nil).Once()

		// Make request
		req, _ := http.NewRequest("GET", "/api/v1/orders/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.OrderStatsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, int64(1), response.TotalOrders)
		assert.Equal(t, "118.00", response.TotalRevenue.String())

		mockOrderService.AssertExpectations(t)
	})

	t.Run("GetOrderStats - With Date Filters", func(t *testing.T) {
		// Setup mock
		mockOrderService.On("GetOrderStats", mock.Anything, mock.MatchedBy(func(req *order.GetOrderStatsRequest) bool {
			return req.CustomerID != nil && *req.CustomerID == testCustomerID &&
				req.StartDate != nil && req.EndDate != nil
		})).Return(testOrderStats, nil).Once()

		// Make request with date filters
		startDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
		endDate := time.Now().Format(time.RFC3339)
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/orders/stats?start_date=%s&end_date=%s&customer_id=%s", startDate, endDate, testCustomerID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.OrderStatsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, int64(1), response.TotalOrders)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("GetOrder - Invalid Order ID", func(t *testing.T) {
		// Make request with invalid UUID
		req, _ := http.NewRequest("GET", "/api/v1/orders/invalid-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response dto.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response.Error, "invalid")
	})

	t.Run("GetOrders - Invalid Date Format", func(t *testing.T) {
		// Make request with invalid date format
		req, _ := http.NewRequest("GET", "/api/v1/orders?created_after=invalid-date", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still return 200 since date parsing errors are ignored
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

func TestOrderValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		request    dto.OrderRequest
		expectCode int
		expectErr  string
	}{
		{
			name: "Valid Order Request",
			request: dto.OrderRequest{
				CustomerID:        uuid.New(),
				Type:              "SALES",
				Priority:          "NORMAL",
				ShippingMethod:    "STANDARD",
				ShippingAddressID: uuid.New(),
				BillingAddressID:  uuid.New(),
				Currency:          "USD",
				Items: []dto.OrderItemRequest{
					{
						ProductID: uuid.New(),
						Quantity:  1,
						UnitPrice: decimal.NewFromFloat(10.00),
					},
				},
			},
			expectCode: http.StatusCreated,
		},
		{
			name: "Missing Customer ID",
			request: dto.OrderRequest{
				Type:              "SALES",
				ShippingMethod:    "STANDARD",
				ShippingAddressID: uuid.New(),
				BillingAddressID:  uuid.New(),
				Currency:          "USD",
				Items: []dto.OrderItemRequest{
					{
						ProductID: uuid.New(),
						Quantity:  1,
						UnitPrice: decimal.NewFromFloat(10.00),
					},
				},
			},
			expectCode: http.StatusBadRequest,
			expectErr:  "Invalid request body",
		},
		{
			name: "Invalid Order Type",
			request: dto.OrderRequest{
				CustomerID:        uuid.New(),
				Type:              "INVALID_TYPE",
				ShippingMethod:    "STANDARD",
				ShippingAddressID: uuid.New(),
				BillingAddressID:  uuid.New(),
				Currency:          "USD",
				Items: []dto.OrderItemRequest{
					{
						ProductID: uuid.New(),
						Quantity:  1,
						UnitPrice: decimal.NewFromFloat(10.00),
					},
				},
			},
			expectCode: http.StatusBadRequest,
			expectErr:  "Invalid request body",
		},
		{
			name: "Missing Items",
			request: dto.OrderRequest{
				CustomerID:        uuid.New(),
				Type:              "SALES",
				ShippingMethod:    "STANDARD",
				ShippingAddressID: uuid.New(),
				BillingAddressID:  uuid.New(),
				Currency:          "USD",
				Items:             []dto.OrderItemRequest{},
			},
			expectCode: http.StatusBadRequest,
			expectErr:  "Invalid request body",
		},
		{
			name: "Invalid Currency Length",
			request: dto.OrderRequest{
				CustomerID:        uuid.New(),
				Type:              "SALES",
				ShippingMethod:    "STANDARD",
				ShippingAddressID: uuid.New(),
				BillingAddressID:  uuid.New(),
				Currency:          "US", // Too short
				Items: []dto.OrderItemRequest{
					{
						ProductID: uuid.New(),
						Quantity:  1,
						UnitPrice: decimal.NewFromFloat(10.00),
					},
				},
			},
			expectCode: http.StatusBadRequest,
			expectErr:  "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock order service (only for successful cases)
			mockOrderService := &testutil.MockOrderService{}
			logger := testutil.NewTestLogger(t)

			if tt.expectCode == http.StatusCreated {
				testOrder := &entities.Order{
					ID:          uuid.New(),
					OrderNumber: "TEST-ORDER",
					CustomerID:  tt.request.CustomerID,
					Type:        entities.OrderType(tt.request.Type),
					Status:      entities.OrderStatusDraft,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}

				mockOrderService.On("CreateOrder", mock.Anything, mock.AnythingOfType("*order.CreateOrderRequest")).
					Return(testOrder, nil).Once()
			}

			// Setup router
			router := gin.New()
			api := router.Group("/api/v1")
			routes.SetupOrderRoutes(api, mockOrderService, logger)

			// Make request
			reqBody, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectCode, w.Code)

			if tt.expectErr != "" {
				var response dto.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, tt.expectErr)
			}

			mockOrderService.AssertExpectations(t)
		})
	}
}