package entities

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data generators
func generateTestCustomer(t *testing.T) *Customer {
	return &Customer{
		ID:              uuid.New(),
		CustomerCode:    "CUST001",
		Type:            "INDIVIDUAL",
		FirstName:       "John",
		LastName:        "Doe",
		Email:           "john.doe@example.com",
		Phone:           "+1234567890",
		CreditLimit:     decimal.NewFromFloat(1000.00),
		CreditUsed:      decimal.Zero,
		Terms:           "NET30",
		IsActive:        true,
		IsVATExempt:     false,
		PreferredCurrency: "USD",
		Source:          "WEB",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func generateTestOrderAddress(t *testing.T, customerID *uuid.UUID) *OrderAddress {
	return &OrderAddress{
		ID:           uuid.New(),
		CustomerID:   customerID,
		Type:         "SHIPPING",
		FirstName:    "John",
		LastName:     "Doe",
		Company:      stringPtr("Acme Corp"),
		AddressLine1: "123 Main St",
		AddressLine2: stringPtr("Apt 4B"),
		City:         "New York",
		State:        "NY",
		PostalCode:   "10001",
		Country:      "US",
		Phone:        stringPtr("+1234567890"),
		Email:        stringPtr("john@example.com"),
		IsDefault:    true,
		IsActive:     true,
		IsValidated:  true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func generateTestOrderItem(t *testing.T, orderID uuid.UUID) *OrderItem {
	return &OrderItem{
		ID:             uuid.New(),
		OrderID:        orderID,
		ProductID:      uuid.New(),
		ProductSKU:     "PROD001",
		ProductName:    "Test Product",
		Quantity:       2,
		UnitPrice:      decimal.NewFromFloat(29.99),
		DiscountAmount: decimal.NewFromFloat(5.00),
		TaxRate:        decimal.NewFromFloat(8.25),
		TaxAmount:      decimal.NewFromFloat(4.12),
		TotalPrice:     decimal.NewFromFloat(59.10),
		Weight:         1.5,
		Dimensions:     "10 x 8 x 2",
		Barcode:        stringPtr("1234567890123"),
		Notes:          stringPtr("Handle with care"),
		Status:         "ORDERED",
		QuantityShipped: 0,
		QuantityReturned: 0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func generateTestOrder(t *testing.T) *Order {
	customerID := uuid.New()
	shippingAddressID := uuid.New()
	billingAddressID := uuid.New()

	return &Order{
		ID:                uuid.New(),
		OrderNumber:       "2024-000001",
		CustomerID:        customerID,
		Status:            OrderStatusPending,
		Priority:          OrderPriorityNormal,
		Type:              OrderTypeSales,
		PaymentStatus:     PaymentStatusPending,
		ShippingMethod:    ShippingMethodStandard,
		Subtotal:          decimal.NewFromFloat(100.00),
		TaxAmount:         decimal.NewFromFloat(8.25),
		ShippingAmount:    decimal.NewFromFloat(10.00),
		DiscountAmount:    decimal.NewFromFloat(5.00),
		TotalAmount:       decimal.NewFromFloat(113.25),
		PaidAmount:        decimal.Zero,
		RefundedAmount:    decimal.Zero,
		Currency:          "USD",
		OrderDate:         time.Now(),
		ShippingAddressID: shippingAddressID,
		BillingAddressID:  billingAddressID,
		Notes:             stringPtr("Please deliver after 5 PM"),
		InternalNotes:     stringPtr("VIP customer"),
		CreatedBy:         uuid.New(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

// ==================== ORDER STATUS TESTS ====================

func TestIsValidStatusTransition(t *testing.T) {
	tests := []struct {
		name     string
		from     OrderStatus
		to       OrderStatus
		expected bool
	}{
		{
			name:     "valid transition: draft to pending",
			from:     OrderStatusDraft,
			to:       OrderStatusPending,
			expected: true,
		},
		{
			name:     "valid transition: pending to confirmed",
			from:     OrderStatusPending,
			to:       OrderStatusConfirmed,
			expected: true,
		},
		{
			name:     "valid transition: confirmed to processing",
			from:     OrderStatusConfirmed,
			to:       OrderStatusProcessing,
			expected: true,
		},
		{
			name:     "valid transition: processing to shipped",
			from:     OrderStatusProcessing,
			to:       OrderStatusShipped,
			expected: true,
		},
		{
			name:     "valid transition: shipped to delivered",
			from:     OrderStatusShipped,
			to:       OrderStatusDelivered,
			expected: true,
		},
		{
			name:     "valid transition: pending to cancelled",
			from:     OrderStatusPending,
			to:       OrderStatusCancelled,
			expected: true,
		},
		{
			name:     "invalid transition: draft to shipped",
			from:     OrderStatusDraft,
			to:       OrderStatusShipped,
			expected: false,
		},
		{
			name:     "invalid transition: delivered to pending",
			from:     OrderStatusDelivered,
			to:       OrderStatusPending,
			expected: false,
		},
		{
			name:     "invalid transition: cancelled to confirmed",
			from:     OrderStatusCancelled,
			to:       OrderStatusConfirmed,
			expected: false,
		},
		{
			name:     "terminal status: delivered cannot transition",
			from:     OrderStatusDelivered,
			to:       OrderStatusShipped,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidStatusTransition(tt.from, tt.to)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   OrderStatus
		expected bool
	}{
		{"terminal status: delivered", OrderStatusDelivered, true},
		{"terminal status: cancelled", OrderStatusCancelled, true},
		{"terminal status: refunded", OrderStatusRefunded, true},
		{"non-terminal status: draft", OrderStatusDraft, false},
		{"non-terminal status: pending", OrderStatusPending, false},
		{"non-terminal status: confirmed", OrderStatusConfirmed, false},
		{"non-terminal status: processing", OrderStatusProcessing, false},
		{"non-terminal status: shipped", OrderStatusShipped, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTerminalStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTerminalStatuses(t *testing.T) {
	terminalStatuses := GetTerminalStatuses()
	expected := []OrderStatus{OrderStatusDelivered, OrderStatusCancelled, OrderStatusRefunded}

	assert.Equal(t, len(expected), len(terminalStatuses))
	for _, expectedStatus := range expected {
		assert.Contains(t, terminalStatuses, expectedStatus)
	}
}

// ==================== ORDER ENTITY TESTS ====================

func TestOrder_Validate(t *testing.T) {
	tests := []struct {
		name        string
		order       *Order
		expectError bool
		errorMsg    string
	}{
		{
			name:  "valid order",
			order: generateTestOrder(t),
			expectError: false,
		},
		{
			name: "missing ID",
			order: &Order{
				OrderNumber: "2024-000001",
				CustomerID:  uuid.New(),
				Status:      OrderStatusPending,
				Currency:    "USD",
				OrderDate:   time.Now(),
				ShippingAddressID: uuid.New(),
				BillingAddressID:  uuid.New(),
				CreatedBy:   uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			expectError: true,
			errorMsg:    "order ID cannot be empty",
		},
		{
			name: "invalid order number format",
			order: &Order{
				ID:          uuid.New(),
				OrderNumber: "INVALID",
				CustomerID:  uuid.New(),
				Status:      OrderStatusPending,
				Currency:    "USD",
				OrderDate:   time.Now(),
				ShippingAddressID: uuid.New(),
				BillingAddressID:  uuid.New(),
				CreatedBy:   uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			expectError: true,
			errorMsg:    "order number must be in format YYYY-NNNNNN",
		},
		{
			name: "missing customer ID",
			order: &Order{
				ID:          uuid.New(),
				OrderNumber: "2024-000001",
				Status:      OrderStatusPending,
				Currency:    "USD",
				OrderDate:   time.Now(),
				ShippingAddressID: uuid.New(),
				BillingAddressID:  uuid.New(),
				CreatedBy:   uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			expectError: true,
			errorMsg:    "customer ID cannot be empty",
		},
		{
			name: "negative total amount",
			order: &Order{
				ID:                uuid.New(),
				OrderNumber:       "2024-000001",
				CustomerID:        uuid.New(),
				Status:            OrderStatusPending,
				TotalAmount:       decimal.NewFromFloat(-10.00),
				Currency:          "USD",
				OrderDate:         time.Now(),
				ShippingAddressID: uuid.New(),
				BillingAddressID:  uuid.New(),
				CreatedBy:         uuid.New(),
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			},
			expectError: true,
			errorMsg:    "total amount cannot be negative",
		},
		{
			name: "invalid currency format",
			order: &Order{
				ID:          uuid.New(),
				OrderNumber: "2024-000001",
				CustomerID:  uuid.New(),
				Status:      OrderStatusPending,
				Currency:    "INVALID",
				OrderDate:   time.Now(),
				ShippingAddressID: uuid.New(),
				BillingAddressID:  uuid.New(),
				CreatedBy:   uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			expectError: true,
			errorMsg:    "currency must be a valid 3-letter ISO 4217 code",
		},
		{
			name: "order date in future",
			order: &Order{
				ID:          uuid.New(),
				OrderNumber: "2024-000001",
				CustomerID:  uuid.New(),
				Status:      OrderStatusPending,
				Currency:    "USD",
				OrderDate:   time.Now().Add(24 * time.Hour),
				ShippingAddressID: uuid.New(),
				BillingAddressID:  uuid.New(),
				CreatedBy:   uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			expectError: true,
			errorMsg:    "order date cannot be in the future",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.order.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOrder_ChangeStatus(t *testing.T) {
	order := generateTestOrder(t)
	// Force the status to PENDING for testing
	order.Status = OrderStatusPending
	originalStatus := order.Status

	t.Run("valid status transition", func(t *testing.T) {
		err := order.ChangeStatus(OrderStatusConfirmed, "Customer confirmed order")
		require.NoError(t, err)
		assert.Equal(t, OrderStatusConfirmed, order.Status)
		assert.NotNil(t, order.PreviousStatus)
		assert.Equal(t, originalStatus, *order.PreviousStatus)
	})

	t.Run("invalid status transition", func(t *testing.T) {
		currentStatus := order.Status // Should be CONFIRMED after previous test
		err := order.ChangeStatus(OrderStatusDraft, "Cannot go back to draft")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
		assert.Equal(t, currentStatus, order.Status) // Status should not change
	})

	t.Run("transition to cancelled sets cancelled date", func(t *testing.T) {
		err := order.ChangeStatus(OrderStatusCancelled, "Customer requested cancellation")
		require.NoError(t, err)
		assert.Equal(t, OrderStatusCancelled, order.Status)
		assert.NotNil(t, order.CancelledDate)
	})
}

func TestOrder_PaymentMethods(t *testing.T) {
	order := generateTestOrder(t)
	order.TotalAmount = decimal.NewFromFloat(100.00)

	t.Run("add valid payment", func(t *testing.T) {
		err := order.AddPayment(decimal.NewFromFloat(50.00))
		require.NoError(t, err)
		assert.True(t, decimal.NewFromFloat(50.00).Equal(order.PaidAmount), "PaidAmount mismatch")
		assert.Equal(t, PaymentStatusPartiallyPaid, order.PaymentStatus)
	})

	t.Run("complete payment", func(t *testing.T) {
		err := order.AddPayment(decimal.NewFromFloat(50.00))
		require.NoError(t, err)
		assert.True(t, decimal.NewFromFloat(100.00).Equal(order.PaidAmount), "PaidAmount mismatch after full payment")
		assert.Equal(t, PaymentStatusPaid, order.PaymentStatus)
		assert.True(t, order.IsFullyPaid())
	})

	t.Run("payment exceeding total", func(t *testing.T) {
		err := order.AddPayment(decimal.NewFromFloat(1.00))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds outstanding balance")
	})

	t.Run("negative payment", func(t *testing.T) {
		err := order.AddPayment(decimal.NewFromFloat(-10.00))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be positive")
	})
}

func TestOrder_RefundMethods(t *testing.T) {
	order := generateTestOrder(t)
	order.TotalAmount = decimal.NewFromFloat(100.00)
	order.PaidAmount = decimal.NewFromFloat(100.00)

	t.Run("valid refund", func(t *testing.T) {
		err := order.AddRefund(decimal.NewFromFloat(25.00))
		require.NoError(t, err)
		assert.True(t, decimal.NewFromFloat(25.00).Equal(order.RefundedAmount), "RefundedAmount mismatch")
	})

	t.Run("full refund changes status", func(t *testing.T) {
		err := order.AddRefund(decimal.NewFromFloat(75.00))
		require.NoError(t, err)
		assert.True(t, decimal.NewFromFloat(100.00).Equal(order.RefundedAmount), "RefundedAmount mismatch after full refund")
		assert.Equal(t, PaymentStatusRefunded, order.PaymentStatus)
		assert.Equal(t, OrderStatusRefunded, order.Status)
	})

	t.Run("refund exceeding paid amount", func(t *testing.T) {
		err := order.AddRefund(decimal.NewFromFloat(1.00))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds paid amount")
	})
}

func TestOrder_BusinessLogic(t *testing.T) {
	t.Run("can be cancelled", func(t *testing.T) {
		order := generateTestOrder(t)
		order.Status = OrderStatusPending
		assert.True(t, order.CanBeCancelled())

		order.Status = OrderStatusShipped
		assert.False(t, order.CanBeCancelled())

		order.Status = OrderStatusDelivered
		assert.False(t, order.CanBeCancelled())
	})

	t.Run("calculate totals", func(t *testing.T) {
		order := generateTestOrder(t)
		order.Items = []OrderItem{
			*generateTestOrderItem(t, order.ID),
			*generateTestOrderItem(t, order.ID),
		}

		err := order.CalculateTotals()
		require.NoError(t, err)
		assert.True(t, order.Subtotal.GreaterThan(decimal.Zero), "Subtotal should be greater than zero")
	})

	t.Run("get outstanding balance", func(t *testing.T) {
		order := generateTestOrder(t)
		order.TotalAmount = decimal.NewFromFloat(100.00)
		order.PaidAmount = decimal.NewFromFloat(30.00)

		balance := order.GetOutstandingBalance()
		assert.True(t, decimal.NewFromFloat(70.00).Equal(balance), "Outstanding balance mismatch")
	})

	t.Run("update tracking", func(t *testing.T) {
		order := generateTestOrder(t)
		err := order.UpdateTracking("1Z999AA10123456784", "FedEx")
		require.NoError(t, err)
		assert.Equal(t, stringPtr("1Z999AA10123456784"), order.TrackingNumber)
		assert.Equal(t, stringPtr("FedEx"), order.Carrier)
	})
}

// ==================== ORDER ITEM ENTITY TESTS ====================

func TestOrderItem_Validate(t *testing.T) {
	tests := []struct {
		name        string
		item        *OrderItem
		expectError bool
		errorMsg    string
	}{
		{
			name:  "valid order item",
			item:  generateTestOrderItem(t, uuid.New()),
			expectError: false,
		},
		{
			name: "missing ID",
			item: &OrderItem{
				OrderID:     uuid.New(),
				ProductID:   uuid.New(),
				ProductSKU:  "PROD001",
				ProductName: "Test Product",
				Quantity:    1,
				UnitPrice:   decimal.NewFromFloat(10.00),
				Status:      "ORDERED",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			expectError: true,
			errorMsg:    "order item ID cannot be empty",
		},
		{
			name: "invalid quantity",
			item: &OrderItem{
				ID:          uuid.New(),
				OrderID:     uuid.New(),
				ProductID:   uuid.New(),
				ProductSKU:  "PROD001",
				ProductName: "Test Product",
				Quantity:    0,
				UnitPrice:   decimal.NewFromFloat(10.00),
				Status:      "ORDERED",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			expectError: true,
			errorMsg:    "quantity must be positive",
		},
		{
			name: "negative unit price",
			item: &OrderItem{
				ID:          uuid.New(),
				OrderID:     uuid.New(),
				ProductID:   uuid.New(),
				ProductSKU:  "PROD001",
				ProductName: "Test Product",
				Quantity:    1,
				UnitPrice:   decimal.NewFromFloat(-10.00),
				Status:      "ORDERED",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			expectError: true,
			errorMsg:    "unit price cannot be negative",
		},
		{
			name: "discount exceeding unit price",
			item: &OrderItem{
				ID:             uuid.New(),
				OrderID:        uuid.New(),
				ProductID:      uuid.New(),
				ProductSKU:     "PROD001",
				ProductName:    "Test Product",
				Quantity:       1,
				UnitPrice:      decimal.NewFromFloat(10.00),
				DiscountAmount: decimal.NewFromFloat(15.00),
				Status:         "ORDERED",
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
			expectError: true,
			errorMsg:    "discount amount cannot exceed unit price",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.item.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOrderItem_ShippingAndReturns(t *testing.T) {
	item := generateTestOrderItem(t, uuid.New())
	item.Quantity = 10

	t.Run("ship valid quantity", func(t *testing.T) {
		err := item.ShipItem(5)
		require.NoError(t, err)
		assert.Equal(t, 5, item.QuantityShipped)
		assert.Equal(t, "PARTIALLY_SHIPPED", item.Status)
	})

	t.Run("ship remaining quantity", func(t *testing.T) {
		err := item.ShipItem(5)
		require.NoError(t, err)
		assert.Equal(t, 10, item.QuantityShipped)
		assert.Equal(t, "SHIPPED", item.Status)
	})

	t.Run("ship more than ordered", func(t *testing.T) {
		err := item.ShipItem(1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot ship 1 items, only 0 remaining")
	})

	t.Run("return valid quantity", func(t *testing.T) {
		err := item.ReturnItem(3)
		require.NoError(t, err)
		assert.Equal(t, 3, item.QuantityReturned)
	})

	t.Run("return more than shipped", func(t *testing.T) {
		err := item.ReturnItem(8)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot return 8 items, only 7 shipped")
	})
}

func TestOrderItem_CalculateTotals(t *testing.T) {
	item := generateTestOrderItem(t, uuid.New())
	item.Quantity = 3
	item.UnitPrice = decimal.NewFromFloat(20.00)
	item.DiscountAmount = decimal.NewFromFloat(5.00)
	item.TaxRate = decimal.NewFromFloat(10.00)

	item.CalculateTotals()

	expectedSubtotal := decimal.NewFromFloat(60.00) // 3 * 20.00
	expectedAfterDiscount := expectedSubtotal.Sub(decimal.NewFromFloat(15.00)) // 60.00 - 5.00 per item
	expectedTax := expectedAfterDiscount.Mul(decimal.NewFromFloat(0.10))
	expectedTotal := expectedAfterDiscount.Add(expectedTax)

	assert.True(t, expectedTax.Equal(item.TaxAmount), "TaxAmount mismatch: expected %s, got %s", expectedTax, item.TaxAmount)
	assert.True(t, expectedTotal.Equal(item.TotalPrice), "TotalPrice mismatch: expected %s, got %s", expectedTotal, item.TotalPrice)
}

// ==================== CUSTOMER ENTITY TESTS ====================

func TestCustomer_Validate(t *testing.T) {
	tests := []struct {
		name        string
		customer    *Customer
		expectError bool
		errorMsg    string
	}{
		{
			name:     "valid individual customer",
			customer: generateTestCustomer(t),
			expectError: false,
		},
		{
			name: "valid business customer",
			customer: &Customer{
				ID:           uuid.New(),
				CustomerCode: "BUS001",
				Type:         "BUSINESS",
				FirstName:    "John",
				LastName:     "Doe",
				CompanyName:  stringPtr("Acme Corporation"),
				Email:        "john@acme.com",
				Phone:        "+1234567890",
				CreditLimit:  decimal.NewFromFloat(5000.00),
				CreditUsed:   decimal.Zero,
				Terms:        "NET30",
				IsActive:     true,
				PreferredCurrency: "USD",
				Source:       "REFERRAL",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			expectError: false,
		},
		{
			name: "missing customer code",
			customer: &Customer{
				ID:         uuid.New(),
				Type:       "INDIVIDUAL",
				FirstName:  "John",
				LastName:   "Doe",
				Terms:      "NET30",
				IsActive:   true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			expectError: true,
			errorMsg:    "customer code cannot be empty",
		},
		{
			name: "invalid customer type",
			customer: &Customer{
				ID:           uuid.New(),
				CustomerCode: "CUST001",
				Type:         "INVALID",
				FirstName:    "John",
				LastName:     "Doe",
				Terms:        "NET30",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			expectError: true,
			errorMsg:    "invalid customer type: INVALID",
		},
		{
			name: "credit used exceeds credit limit",
			customer: &Customer{
				ID:           uuid.New(),
				CustomerCode: "CUST001",
				Type:         "INDIVIDUAL",
				FirstName:    "John",
				LastName:     "Doe",
				CreditLimit:  decimal.NewFromFloat(1000.00),
				CreditUsed:   decimal.NewFromFloat(1500.00),
				Terms:        "NET30",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			expectError: true,
			errorMsg:    "credit used cannot exceed credit limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.customer.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCustomer_CreditMethods(t *testing.T) {
	customer := generateTestCustomer(t)
	customer.CreditLimit = decimal.NewFromFloat(1000.00)

	t.Run("has available credit", func(t *testing.T) {
		assert.True(t, customer.HasAvailableCredit(decimal.NewFromFloat(500.00)))
		assert.False(t, customer.HasAvailableCredit(decimal.NewFromFloat(1500.00)))
	})

	t.Run("use credit", func(t *testing.T) {
		err := customer.UseCredit(decimal.NewFromFloat(300.00))
		require.NoError(t, err)
		assert.True(t, decimal.NewFromFloat(300.00).Equal(customer.CreditUsed), "CreditUsed mismatch")
		assert.True(t, decimal.NewFromFloat(700.00).Equal(customer.GetAvailableCredit()), "AvailableCredit mismatch")
	})

	t.Run("use credit exceeding limit", func(t *testing.T) {
		err := customer.UseCredit(decimal.NewFromFloat(800.00))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient credit")
	})

	t.Run("release credit", func(t *testing.T) {
		err := customer.ReleaseCredit(decimal.NewFromFloat(200.00))
		require.NoError(t, err)
		assert.True(t, decimal.NewFromFloat(100.00).Equal(customer.CreditUsed), "CreditUsed mismatch after release")
	})

	t.Run("release more credit than used", func(t *testing.T) {
		err := customer.ReleaseCredit(decimal.NewFromFloat(200.00))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot release more credit than used")
	})
}

func TestCustomer_DisplayMethods(t *testing.T) {
	t.Run("individual customer display name", func(t *testing.T) {
		customer := generateTestCustomer(t)
		customer.Type = "INDIVIDUAL"
		assert.Equal(t, "John Doe", customer.GetFullName())
		assert.Equal(t, "John Doe", customer.GetDisplayName())
	})

	t.Run("business customer display name", func(t *testing.T) {
		customer := generateTestCustomer(t)
		customer.Type = "BUSINESS"
		customer.CompanyName = stringPtr("Acme Corporation")
		assert.Equal(t, "John Doe", customer.GetFullName())
		assert.Equal(t, "Acme Corporation", customer.GetDisplayName()) // Business customers show company name
	})
}

// ==================== ORDER ADDRESS ENTITY TESTS ====================

func TestOrderAddress_Validate(t *testing.T) {
	customerID := uuid.New()

	tests := []struct {
		name        string
		address     *OrderAddress
		expectError bool
		errorMsg    string
	}{
		{
			name:    "valid shipping address",
			address: generateTestOrderAddress(t, &customerID),
			expectError: false,
		},
		{
			name: "valid billing address",
			address: &OrderAddress{
				ID:           uuid.New(),
				CustomerID:   &customerID,
				Type:         "BILLING",
				FirstName:    "Jane",
				LastName:     "Smith",
				AddressLine1: "456 Oak Ave",
				City:         "Los Angeles",
				State:        "CA",
				PostalCode:   "90210",
				Country:      "US",
				IsDefault:    false,
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			expectError: false,
		},
		{
			name: "missing address association",
			address: &OrderAddress{
				ID:           uuid.New(),
				Type:         "SHIPPING",
				FirstName:    "John",
				LastName:     "Doe",
				AddressLine1: "123 Main St",
				City:         "New York",
				State:        "NY",
				PostalCode:   "10001",
				Country:      "US",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			expectError: true,
			errorMsg:    "address must be associated with either a customer or an order",
		},
		{
			name: "invalid address type",
			address: &OrderAddress{
				ID:           uuid.New(),
				CustomerID:   &customerID,
				Type:         "INVALID",
				FirstName:    "John",
				LastName:     "Doe",
				AddressLine1: "123 Main St",
				City:         "New York",
				State:        "NY",
				PostalCode:   "10001",
				Country:      "US",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			expectError: true,
			errorMsg:    "invalid address type: INVALID",
		},
		{
			name: "missing required address fields",
			address: &OrderAddress{
				ID:         uuid.New(),
				CustomerID: &customerID,
				Type:       "SHIPPING",
				FirstName:  "John",
				LastName:   "Doe",
				City:       "New York",
				State:      "NY",
				Country:    "US",
				IsActive:   true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			expectError: true,
			errorMsg:    "address line 1 cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.address.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOrderAddress_AddressMethods(t *testing.T) {
	address := generateTestOrderAddress(t, nil)

	t.Run("get full name", func(t *testing.T) {
		assert.Equal(t, "John Doe", address.GetFullName())
	})

	t.Run("get full address", func(t *testing.T) {
		fullAddress := address.GetFullAddress()
		assert.Contains(t, fullAddress, "Acme Corp")
		assert.Contains(t, fullAddress, "John Doe")
		assert.Contains(t, fullAddress, "123 Main St")
		assert.Contains(t, fullAddress, "Apt 4B")
		assert.Contains(t, fullAddress, "New York, NY 10001")
		assert.Contains(t, fullAddress, "US")
	})

	t.Run("get single line address", func(t *testing.T) {
		singleLine := address.GetSingleLineAddress()
		assert.Contains(t, singleLine, "123 Main St")
		assert.Contains(t, singleLine, "Apt 4B")
		assert.Contains(t, singleLine, "New York")
		assert.Contains(t, singleLine, "NY")
		assert.Contains(t, singleLine, "10001")
		assert.Contains(t, singleLine, "US")
	})

	t.Run("address type checks", func(t *testing.T) {
		address.Type = "SHIPPING"
		assert.True(t, address.IsShippingAddress())
		assert.False(t, address.IsBillingAddress())

		address.Type = "BILLING"
		assert.False(t, address.IsShippingAddress())
		assert.True(t, address.IsBillingAddress())

		address.Type = "BOTH"
		assert.True(t, address.IsShippingAddress())
		assert.True(t, address.IsBillingAddress())
	})

	t.Run("default address management", func(t *testing.T) {
		address.SetAsDefault()
		assert.True(t, address.IsDefault)

		address.UnsetDefault()
		assert.False(t, address.IsDefault)
	})
}

func TestOrderAddress_AddressValidation(t *testing.T) {
	tests := []struct {
		name     string
		address  *OrderAddress
		expected bool
	}{
		{
			name: "valid US address",
			address: &OrderAddress{
				AddressLine1: "123 Main St",
				City:         "New York",
				State:        "NY",
				PostalCode:   "10001",
				Country:      "US",
			},
			expected: true,
		},
		{
			name: "valid US address with ZIP+4",
			address: &OrderAddress{
				AddressLine1: "123 Main St",
				City:         "New York",
				State:        "NY",
				PostalCode:   "10001-1234",
				Country:      "US",
			},
			expected: true,
		},
		{
			name: "valid Canadian address",
			address: &OrderAddress{
				AddressLine1: "123 Main St",
				City:         "Toronto",
				State:        "ON",
				PostalCode:   "M5V 3L9",
				Country:      "CA",
			},
			expected: true,
		},
		{
			name: "valid UK address",
			address: &OrderAddress{
				AddressLine1: "123 Baker Street",
				City:         "London",
				State:        "England",
				PostalCode:   "NW1 6XE",
				Country:      "GB",
			},
			expected: true,
		},
		{
			name: "invalid US ZIP code",
			address: &OrderAddress{
				AddressLine1: "123 Main St",
				City:         "New York",
				State:        "NY",
				PostalCode:   "INVALID",
				Country:      "US",
			},
			expected: false,
		},
		{
			name: "missing required fields",
			address: &OrderAddress{
				AddressLine1: "123 Main St",
				City:         "New York",
				State:        "NY",
				Country:      "US",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.address.ValidateAddress()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== ORDER CALCULATION TESTS ====================

func TestCalculateOrderTotals(t *testing.T) {
	order := generateTestOrder(t)
	// Clear order-level discount to test item-level discounts only
	order.DiscountAmount = decimal.Zero
	order.Items = []OrderItem{
		{
			Quantity:       2,
			UnitPrice:      decimal.NewFromFloat(25.00),
			DiscountAmount: decimal.NewFromFloat(5.00), // Total discount for this line item
			TaxRate:        decimal.NewFromFloat(8.00),
			ProductName:    "Test Product 1",
		},
		{
			Quantity:       1,
			UnitPrice:      decimal.NewFromFloat(50.00),
			DiscountAmount: decimal.Zero,
			TaxRate:        decimal.NewFromFloat(8.00),
			ProductName:    "Test Product 2",
		},
	}

	calculation, err := CalculateOrderTotals(order, decimal.NewFromFloat(8.00), decimal.NewFromFloat(10.00))
	require.NoError(t, err)

	// Expected calculations:
	// Item 1: (2 * 25.00) = 50.00, discount = 5.00, taxable = 45.00, tax = 45.00 * 0.08 = 3.60
	// Item 2: (1 * 50.00) = 50.00, discount = 0.00, taxable = 50.00, tax = 50.00 * 0.08 = 4.00
	// Subtotal = 50.00 + 50.00 = 100.00
	// Tax = 3.60 + 4.00 = 7.60
	// Shipping = 10.00
	// Discount = 5.00 (from item 1)
	// Total = 100.00 + 7.60 + 10.00 - 5.00 = 112.60

	expectedSubtotal := decimal.NewFromFloat(100.00)
	expectedTax := decimal.NewFromFloat(7.60)
	expectedTotal := decimal.NewFromFloat(112.60)

	assert.True(t, expectedSubtotal.Equal(calculation.Subtotal), "Subtotal mismatch: expected %s, got %s", expectedSubtotal, calculation.Subtotal)
	assert.True(t, expectedTax.Equal(calculation.TaxAmount), "TaxAmount mismatch: expected %s, got %s", expectedTax, calculation.TaxAmount)
	assert.True(t, decimal.NewFromFloat(10.00).Equal(calculation.ShippingAmount), "ShippingAmount mismatch")
	assert.True(t, decimal.NewFromFloat(5.00).Equal(calculation.DiscountAmount), "DiscountAmount mismatch") // From item 1
	assert.True(t, expectedTotal.Equal(calculation.TotalAmount), "TotalAmount mismatch: expected %s, got %s", expectedTotal, calculation.TotalAmount)
	assert.Len(t, calculation.TaxBreakdown, 2)
	assert.Len(t, calculation.DiscountBreakdown, 1)
}

func TestValidateOrder(t *testing.T) {
	t.Run("valid order", func(t *testing.T) {
		order := generateTestOrder(t)
		order.Items = []OrderItem{*generateTestOrderItem(t, order.ID)}

		validation := ValidateOrder(order)
		assert.True(t, validation.IsValid)
		assert.Empty(t, validation.Errors)
	})

	t.Run("order with no items", func(t *testing.T) {
		order := generateTestOrder(t)
		order.Items = []OrderItem{}

		validation := ValidateOrder(order)
		assert.False(t, validation.IsValid)
		assert.Contains(t, validation.Errors, "Order must have at least one item")
	})

	t.Run("order with invalid status transition", func(t *testing.T) {
		order := generateTestOrder(t)
		order.Items = []OrderItem{*generateTestOrderItem(t, order.ID)}
		order.Status = OrderStatusDelivered
		previousStatus := OrderStatusPending
		order.PreviousStatus = &previousStatus

		validation := ValidateOrder(order)
		assert.False(t, validation.IsValid)
		assert.True(t, len(validation.Errors) > 0 && strings.Contains(validation.Errors[0], "Invalid status transition"), "Expected invalid status transition error")
	})

	t.Run("order with warnings", func(t *testing.T) {
		order := generateTestOrder(t)
		// Create an item with high value to trigger the $10k warning
		item := generateTestOrderItem(t, order.ID)
		item.Quantity = 100
		item.UnitPrice = decimal.NewFromFloat(150.00)
		item.DiscountAmount = decimal.Zero
		item.TaxRate = decimal.NewFromFloat(8.00)
		item.CalculateTotals()
		order.Items = []OrderItem{*item}
		
		// Calculate totals properly
		err := order.CalculateTotals()
		require.NoError(t, err)
		
		// Set required date to past to trigger warning (but after order date to pass validation)
		order.OrderDate = time.Now().Add(-48 * time.Hour) // Order date 2 days ago
		pastDate := time.Now().Add(-24 * time.Hour) // Required date yesterday (after order date but before now)
		order.RequiredDate = &pastDate

		validation := ValidateOrder(order)
		if !validation.IsValid {
			t.Logf("Validation errors: %v", validation.Errors)
		}
		assert.True(t, validation.IsValid) // Warnings don't make order invalid
		assert.GreaterOrEqual(t, len(validation.Warnings), 1) // At least one warning
		// Check for the warnings we expect
		hasRequiredDateWarning := false
		hasTotalWarning := false
		for _, w := range validation.Warnings {
			if strings.Contains(w, "required date has passed") {
				hasRequiredDateWarning = true
			}
			if strings.Contains(w, "exceeds $10,000") {
				hasTotalWarning = true
			}
		}
		assert.True(t, hasRequiredDateWarning || hasTotalWarning, "Expected at least one of the warnings")
	})
}

func TestGenerateOrderNumber(t *testing.T) {
	orderNumber := GenerateOrderNumber()

	// Should be in format YYYY-NNNNNN
	assert.Len(t, orderNumber, 11) // 4 + 1 + 6
	assert.Contains(t, orderNumber, "-")

	year := time.Now().Year()
	assert.Contains(t, orderNumber, fmt.Sprintf("%d-", year))
}

func TestCalculateShippingWeight(t *testing.T) {
	order := generateTestOrder(t)
	order.Items = []OrderItem{
		{Quantity: 2, Weight: 1.5},
		{Quantity: 1, Weight: 3.0},
	}

	totalWeight := CalculateShippingWeight(order)
	expectedWeight := (2 * 1.5) + (1 * 3.0) // 3.0 + 3.0 = 6.0
	assert.Equal(t, expectedWeight, totalWeight)
}

func TestIsOrderComplete(t *testing.T) {
	tests := []struct {
		name     string
		status   OrderStatus
		expected bool
	}{
		{"terminal status: delivered", OrderStatusDelivered, true},
		{"terminal status: cancelled", OrderStatusCancelled, true},
		{"terminal status: refunded", OrderStatusRefunded, true},
		{"non-terminal: draft", OrderStatusDraft, false},
		{"non-terminal: pending", OrderStatusPending, false},
		{"non-terminal: confirmed", OrderStatusConfirmed, false},
		{"non-terminal: processing", OrderStatusProcessing, false},
		{"non-terminal: shipped", OrderStatusShipped, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := generateTestOrder(t)
			order.Status = tt.status
			result := IsOrderComplete(order)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCanOrderBeModified(t *testing.T) {
	tests := []struct {
		name     string
		status   OrderStatus
		expected bool
	}{
		{"modifiable: draft", OrderStatusDraft, true},
		{"modifiable: pending", OrderStatusPending, true},
		{"modifiable: confirmed", OrderStatusConfirmed, true},
		{"modifiable: processing", OrderStatusProcessing, true},
		{"not modifiable: shipped", OrderStatusShipped, false},
		{"not modifiable: delivered", OrderStatusDelivered, false},
		{"not modifiable: cancelled", OrderStatusCancelled, false},
		{"not modifiable: refunded", OrderStatusRefunded, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := generateTestOrder(t)
			order.Status = tt.status
			result := CanOrderBeModified(order)
			assert.Equal(t, tt.expected, result)
		})
	}
}