package testutil

import (
	"context"
	"testing"
	"time"

	"erpgo/internal/domain/orders/entities"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// CreateTestOrder creates a test order for testing purposes
func CreateTestOrder(t *testing.T) *entities.Order {
	now := time.Now()
	customerID := uuid.New()
	shippingAddressID := uuid.New()
	billingAddressID := uuid.New()
	createdBy := uuid.New()

	return &entities.Order{
		ID:                uuid.New(),
		OrderNumber:       GenerateTestOrderNumber(),
		CustomerID:        customerID,
		Status:            entities.OrderStatusPending,
		Priority:          entities.OrderPriorityNormal,
		Type:              entities.OrderTypeSales,
		PaymentStatus:     entities.PaymentStatusPending,
		ShippingMethod:    entities.ShippingMethodStandard,
		Subtotal:          decimal.NewFromFloat(100.00),
		TaxAmount:         decimal.NewFromFloat(8.00),
		ShippingAmount:    decimal.NewFromFloat(10.00),
		DiscountAmount:    decimal.NewFromFloat(0.00),
		TotalAmount:       decimal.NewFromFloat(118.00),
		PaidAmount:        decimal.NewFromFloat(0.00),
		RefundedAmount:    decimal.NewFromFloat(0.00),
		Currency:          "USD",
		OrderDate:         now,
		RequiredDate:      func() *time.Time { t := now.AddDate(0, 0, 7); return &t }(),
		ShippingAddressID: shippingAddressID,
		BillingAddressID:  billingAddressID,
		Notes:             stringPtr("Test order notes"),
		CreatedBy:         createdBy,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// CreateTestOrderItem creates a test order item for testing purposes
func CreateTestOrderItem(t *testing.T, orderID uuid.UUID) *entities.OrderItem {
	productID := uuid.New()
	now := time.Now()

	return &entities.OrderItem{
		ID:               uuid.New(),
		OrderID:          orderID,
		ProductID:        productID,
		ProductSKU:       "TEST-SKU-001",
		ProductName:      "Test Product",
		Quantity:         2,
		UnitPrice:        decimal.NewFromFloat(50.00),
		DiscountAmount:   decimal.NewFromFloat(0.00),
		TaxRate:          decimal.NewFromFloat(0.08),
		TaxAmount:        decimal.NewFromFloat(8.00),
		TotalPrice:       decimal.NewFromFloat(108.00),
		Weight:           1.5,
		Dimensions:       "10x5x3",
		Status:           "ORDERED",
		QuantityShipped:  0,
		QuantityReturned: 0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// CreateTestCustomer creates a test customer for testing purposes
func CreateTestCustomer(t *testing.T) *entities.Customer {
	now := time.Now()

	return &entities.Customer{
		ID:                uuid.New(),
		CustomerCode:      GenerateTestCustomerCode(),
		Type:              "INDIVIDUAL",
		FirstName:         "John",
		LastName:          "Doe",
		Email:             "john.doe@example.com",
		Phone:             "+1-555-123-4567",
		CompanyName:       stringPtr("Test Company"),
		CreditLimit:       decimal.NewFromFloat(1000.00),
		CreditUsed:        decimal.NewFromFloat(0.00),
		Terms:             "NET30",
		IsActive:          true,
		IsVATExempt:       false,
		PreferredCurrency: "USD",
		Notes:             stringPtr("Test customer"),
		Source:            "WEB",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// CreateTestOrderAddress creates a test order address for testing purposes
func CreateTestOrderAddress(t *testing.T, customerID *uuid.UUID, orderID *uuid.UUID, addressType string) *entities.OrderAddress {
	now := time.Now()

	return &entities.OrderAddress{
		ID:           uuid.New(),
		CustomerID:   customerID,
		OrderID:      orderID,
		Type:         addressType,
		FirstName:    "John",
		LastName:     "Doe",
		Company:      stringPtr("Test Company"),
		AddressLine1: "123 Test St",
		AddressLine2: stringPtr("Apt 4B"),
		City:         "Test City",
		State:        "CA",
		PostalCode:   "12345",
		Country:      "USA",
		Phone:        stringPtr("+1-555-123-4567"),
		Email:        stringPtr("john.doe@example.com"),
		Instructions: stringPtr("Leave at door"),
		IsDefault:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// CreateTestCompany creates a test company for testing purposes
func CreateTestCompany(t *testing.T) *entities.Company {
	now := time.Now()

	return &entities.Company{
		ID:          uuid.New(),
		CompanyName: "Test Company LLC",
		LegalName:   "Test Company LLC",
		TaxID:       "12-3456789",
		Industry:    "Technology",
		Website:     stringPtr("https://testcompany.com"),
		Phone:       "+1-555-987-6543",
		Email:       "info@testcompany.com",
		Address:     "456 Business Ave",
		City:        "Business City",
		State:       "NY",
		Country:     "USA",
		PostalCode:  "54321",
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// GenerateTestOrderNumber generates a unique test order number
func GenerateTestOrderNumber() string {
	return "TEST-" + time.Now().Format("20060102") + "-" + uuid.New().String()[:8]
}

// GenerateTestCustomerCode generates a unique test customer code
func GenerateTestCustomerCode() string {
	return "CUST-" + time.Now().Format("20060102") + "-" + uuid.New().String()[:8]
}

// AssertOrdersEqual asserts that two orders are equal (excluding timestamps)
func AssertOrdersEqual(t *testing.T, expected, actual *entities.Order) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.OrderNumber, actual.OrderNumber)
	require.Equal(t, expected.CustomerID, actual.CustomerID)
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.Priority, actual.Priority)
	require.Equal(t, expected.Type, actual.Type)
	require.Equal(t, expected.PaymentStatus, actual.PaymentStatus)
	require.Equal(t, expected.ShippingMethod, actual.ShippingMethod)
	require.True(t, expected.Subtotal.Equal(actual.Subtotal))
	require.True(t, expected.TaxAmount.Equal(actual.TaxAmount))
	require.True(t, expected.ShippingAmount.Equal(actual.ShippingAmount))
	require.True(t, expected.DiscountAmount.Equal(actual.DiscountAmount))
	require.True(t, expected.TotalAmount.Equal(actual.TotalAmount))
	require.True(t, expected.PaidAmount.Equal(actual.PaidAmount))
	require.True(t, expected.RefundedAmount.Equal(actual.RefundedAmount))
	require.Equal(t, expected.Currency, actual.Currency)
	require.Equal(t, expected.RequiredDate, actual.RequiredDate)
	require.Equal(t, expected.ShippingAddressID, actual.ShippingAddressID)
	require.Equal(t, expected.BillingAddressID, actual.BillingAddressID)
	require.Equal(t, expected.CreatedBy, actual.CreatedBy)
}

// AssertOrderItemsEqual asserts that two order items are equal (excluding timestamps)
func AssertOrderItemsEqual(t *testing.T, expected, actual *entities.OrderItem) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.OrderID, actual.OrderID)
	require.Equal(t, expected.ProductID, actual.ProductID)
	require.Equal(t, expected.ProductSKU, actual.ProductSKU)
	require.Equal(t, expected.ProductName, actual.ProductName)
	require.Equal(t, expected.Quantity, actual.Quantity)
	require.True(t, expected.UnitPrice.Equal(actual.UnitPrice))
	require.True(t, expected.DiscountAmount.Equal(actual.DiscountAmount))
	require.True(t, expected.TaxRate.Equal(actual.TaxRate))
	require.True(t, expected.TaxAmount.Equal(actual.TaxAmount))
	require.True(t, expected.TotalPrice.Equal(actual.TotalPrice))
	require.Equal(t, expected.Weight, actual.Weight)
	require.Equal(t, expected.Dimensions, actual.Dimensions)
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.QuantityShipped, actual.QuantityShipped)
	require.Equal(t, expected.QuantityReturned, actual.QuantityReturned)
}

// AssertCustomersEqual asserts that two customers are equal (excluding timestamps)
func AssertCustomersEqual(t *testing.T, expected, actual *entities.Customer) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.CustomerCode, actual.CustomerCode)
	require.Equal(t, expected.CompanyID, actual.CompanyID)
	require.Equal(t, expected.Type, actual.Type)
	require.Equal(t, expected.FirstName, actual.FirstName)
	require.Equal(t, expected.LastName, actual.LastName)
	require.Equal(t, expected.Email, actual.Email)
	require.Equal(t, expected.Phone, actual.Phone)
	require.Equal(t, expected.CompanyName, actual.CompanyName)
	require.Equal(t, expected.TaxID, actual.TaxID)
	require.Equal(t, expected.Industry, actual.Industry)
	require.True(t, expected.CreditLimit.Equal(actual.CreditLimit))
	require.True(t, expected.CreditUsed.Equal(actual.CreditUsed))
	require.Equal(t, expected.Terms, actual.Terms)
	require.Equal(t, expected.IsActive, actual.IsActive)
	require.Equal(t, expected.IsVATExempt, actual.IsVATExempt)
	require.Equal(t, expected.PreferredCurrency, actual.PreferredCurrency)
	require.Equal(t, expected.Source, actual.Source)
}

// CreateTestOrderFilter creates a test order filter
func CreateTestOrderFilter() map[string]interface{} {
	return map[string]interface{}{
		"status":           []string{"PENDING", "CONFIRMED"},
		"payment_status":   []string{"PENDING"},
		"start_date":       time.Now().AddDate(0, 0, -30),
		"end_date":         time.Now(),
		"min_total_amount": decimal.NewFromFloat(50.00),
		"max_total_amount": decimal.NewFromFloat(500.00),
		"page":             1,
		"limit":            10,
		"sort_by":          "order_date",
		"sort_order":       "DESC",
	}
}

// CreateTestCustomerFilter creates a test customer filter
func CreateTestCustomerFilter() map[string]interface{} {
	return map[string]interface{}{
		"type":             "INDIVIDUAL",
		"is_active":        true,
		"start_date":       time.Now().AddDate(0, 0, -30),
		"end_date":         time.Now(),
		"min_credit_limit": decimal.NewFromFloat(100.00),
		"page":             1,
		"limit":            10,
		"sort_by":          "created_at",
		"sort_order":       "DESC",
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// CreateTestContext creates a test context with timeout
func CreateTestContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	return ctx
}

// CleanupTestData cleans up test data from the database
func CleanupTestData(ctx context.Context, db interface{}, tableNames []string) error {
	// This would be implemented based on the specific database interface
	// For now, it's a placeholder
	return nil
}
