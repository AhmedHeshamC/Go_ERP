package entities

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrder_ComprehensiveCoverage tests all order methods for complete coverage
func TestOrder_ComprehensiveCoverage(t *testing.T) {
	t.Run("GetTotalWeight", func(t *testing.T) {
		order := generateTestOrder(t)
		order.Items = []OrderItem{
			{
				Quantity: 2,
				Weight:   1.5,
			},
			{
				Quantity: 3,
				Weight:   2.0,
			},
		}

		totalWeight := order.GetTotalWeight()
		expected := (2 * 1.5) + (3 * 2.0) // 3.0 + 6.0 = 9.0
		assert.Equal(t, expected, totalWeight)
	})

	t.Run("GetItemCount", func(t *testing.T) {
		order := generateTestOrder(t)
		order.Items = []OrderItem{
			{Quantity: 2},
			{Quantity: 3},
			{Quantity: 5},
		}

		count := order.GetItemCount()
		assert.Equal(t, 10, count)
	})

	t.Run("IsDigitalOrder", func(t *testing.T) {
		order := generateTestOrder(t)
		order.ShippingMethod = ShippingMethodDigital
		order.Items = []OrderItem{{}}

		assert.True(t, order.IsDigitalOrder())

		order.ShippingMethod = ShippingMethodStandard
		assert.False(t, order.IsDigitalOrder())

		order.Items = []OrderItem{}
		assert.False(t, order.IsDigitalOrder())
	})

	t.Run("SetPriority", func(t *testing.T) {
		order := generateTestOrder(t)

		// Valid priority
		err := order.SetPriority(OrderPriorityHigh)
		assert.NoError(t, err)
		assert.Equal(t, OrderPriorityHigh, order.Priority)

		// Invalid priority
		err = order.SetPriority(OrderPriority("INVALID"))
		assert.Error(t, err)
	})

	t.Run("UpdateTracking", func(t *testing.T) {
		order := generateTestOrder(t)

		// Valid tracking
		err := order.UpdateTracking("1Z999AA10123456784", "FedEx")
		assert.NoError(t, err)
		assert.Equal(t, "1Z999AA10123456784", *order.TrackingNumber)
		assert.Equal(t, "FedEx", *order.Carrier)

		// Tracking number too long
		longTracking := string(make([]byte, 101))
		err = order.UpdateTracking(longTracking, "FedEx")
		assert.Error(t, err)

		// Invalid tracking characters
		err = order.UpdateTracking("TRACK@#$", "FedEx")
		assert.Error(t, err)

		// Carrier too long
		longCarrier := string(make([]byte, 51))
		err = order.UpdateTracking("TRACK123", longCarrier)
		assert.Error(t, err)
	})
}

// TestOrderItem_ComprehensiveCoverage tests all order item methods
func TestOrderItem_ComprehensiveCoverage(t *testing.T) {
	t.Run("GetItemWeight", func(t *testing.T) {
		item := &OrderItem{
			Quantity: 5,
			Weight:   2.5,
		}

		totalWeight := item.GetItemWeight()
		assert.Equal(t, 12.5, totalWeight)
	})

	t.Run("CanBeShipped", func(t *testing.T) {
		item := &OrderItem{
			Status:          "ORDERED",
			Quantity:        10,
			QuantityShipped: 0,
		}

		assert.True(t, item.CanBeShipped())

		item.Status = "SHIPPED"
		assert.False(t, item.CanBeShipped())

		item.Status = "ORDERED"
		item.QuantityShipped = 10
		assert.False(t, item.CanBeShipped())
	})
}

// TestCustomer_ComprehensiveCoverage tests all customer methods
func TestCustomer_ComprehensiveCoverage(t *testing.T) {
	t.Run("GetDisplayName_Individual", func(t *testing.T) {
		customer := generateTestCustomer(t)
		customer.Type = "INDIVIDUAL"
		customer.CompanyName = nil

		displayName := customer.GetDisplayName()
		assert.Equal(t, "John Doe", displayName)
	})

	t.Run("GetDisplayName_Business", func(t *testing.T) {
		customer := generateTestCustomer(t)
		customer.Type = "BUSINESS"
		companyName := "Acme Corporation"
		customer.CompanyName = &companyName

		displayName := customer.GetDisplayName()
		assert.Equal(t, "Acme Corporation", displayName)
	})

	t.Run("GetFullName", func(t *testing.T) {
		customer := generateTestCustomer(t)
		fullName := customer.GetFullName()
		assert.Equal(t, "John Doe", fullName)
	})
}

// TestOrderAddress_ComprehensiveCoverage tests all address methods
func TestOrderAddress_ComprehensiveCoverage(t *testing.T) {
	customerID := uuid.New()

	t.Run("GetFullName", func(t *testing.T) {
		address := generateTestOrderAddress(t, &customerID)
		fullName := address.GetFullName()
		assert.Equal(t, "John Doe", fullName)
	})

	t.Run("GetFullAddress", func(t *testing.T) {
		address := generateTestOrderAddress(t, &customerID)
		fullAddress := address.GetFullAddress()

		assert.Contains(t, fullAddress, "Acme Corp")
		assert.Contains(t, fullAddress, "John Doe")
		assert.Contains(t, fullAddress, "123 Main St")
		assert.Contains(t, fullAddress, "Apt 4B")
		assert.Contains(t, fullAddress, "New York, NY 10001")
		assert.Contains(t, fullAddress, "US")
	})

	t.Run("GetSingleLineAddress", func(t *testing.T) {
		address := generateTestOrderAddress(t, &customerID)
		singleLine := address.GetSingleLineAddress()

		assert.Contains(t, singleLine, "123 Main St")
		assert.Contains(t, singleLine, "Apt 4B")
		assert.Contains(t, singleLine, "New York")
		assert.Contains(t, singleLine, "NY")
		assert.Contains(t, singleLine, "10001")
		assert.Contains(t, singleLine, "US")
	})

	t.Run("IsShippingAddress", func(t *testing.T) {
		address := generateTestOrderAddress(t, &customerID)
		address.Type = "SHIPPING"
		assert.True(t, address.IsShippingAddress())
		assert.False(t, address.IsBillingAddress())

		address.Type = "BOTH"
		assert.True(t, address.IsShippingAddress())
		assert.True(t, address.IsBillingAddress())
	})

	t.Run("IsBillingAddress", func(t *testing.T) {
		address := generateTestOrderAddress(t, &customerID)
		address.Type = "BILLING"
		assert.False(t, address.IsShippingAddress())
		assert.True(t, address.IsBillingAddress())
	})

	t.Run("SetAsDefault", func(t *testing.T) {
		address := generateTestOrderAddress(t, &customerID)
		address.IsDefault = false

		address.SetAsDefault()
		assert.True(t, address.IsDefault)
	})

	t.Run("UnsetDefault", func(t *testing.T) {
		address := generateTestOrderAddress(t, &customerID)
		address.IsDefault = true

		address.UnsetDefault()
		assert.False(t, address.IsDefault)
	})

	t.Run("ValidateAddress_US", func(t *testing.T) {
		address := &OrderAddress{
			AddressLine1: "123 Main St",
			City:         "New York",
			State:        "NY",
			PostalCode:   "10001",
			Country:      "US",
		}

		assert.True(t, address.ValidateAddress())

		// Invalid ZIP
		address.PostalCode = "INVALID"
		assert.False(t, address.ValidateAddress())

		// ZIP+4 format
		address.PostalCode = "10001-1234"
		assert.True(t, address.ValidateAddress())
	})

	t.Run("ValidateAddress_Canada", func(t *testing.T) {
		address := &OrderAddress{
			AddressLine1: "123 Main St",
			City:         "Toronto",
			State:        "ON",
			PostalCode:   "M5V 3L9",
			Country:      "CA",
		}

		assert.True(t, address.ValidateAddress())

		// Without space
		address.PostalCode = "M5V3L9"
		assert.True(t, address.ValidateAddress())

		// Invalid format
		address.PostalCode = "INVALID"
		assert.False(t, address.ValidateAddress())
	})

	t.Run("ValidateAddress_UK", func(t *testing.T) {
		address := &OrderAddress{
			AddressLine1: "123 Baker Street",
			City:         "London",
			State:        "England",
			PostalCode:   "NW1 6XE",
			Country:      "GB",
		}

		assert.True(t, address.ValidateAddress())

		// Invalid format
		address.PostalCode = "INVALID"
		assert.False(t, address.ValidateAddress())
	})

	t.Run("ValidateAddress_Other", func(t *testing.T) {
		address := &OrderAddress{
			AddressLine1: "123 Main St",
			City:         "Tokyo",
			State:        "Tokyo",
			PostalCode:   "100-0001",
			Country:      "JP",
		}

		assert.True(t, address.ValidateAddress())

		// Too short
		address.PostalCode = "12"
		assert.False(t, address.ValidateAddress())

		// Too long
		address.PostalCode = string(make([]byte, 21))
		assert.False(t, address.ValidateAddress())
	})

	t.Run("ValidateAddress_MissingFields", func(t *testing.T) {
		address := &OrderAddress{
			City:       "New York",
			State:      "NY",
			PostalCode: "10001",
			Country:    "US",
		}

		assert.False(t, address.ValidateAddress())
	})

	t.Run("MarkAsValidated", func(t *testing.T) {
		address := generateTestOrderAddress(t, &customerID)
		address.IsValidated = false

		address.MarkAsValidated()
		assert.True(t, address.IsValidated)
	})

	t.Run("MarkAsUnvalidated", func(t *testing.T) {
		address := generateTestOrderAddress(t, &customerID)
		address.IsValidated = true

		address.MarkAsUnvalidated()
		assert.False(t, address.IsValidated)
	})
}

// TestOrder_AdditionalCoverage adds more coverage for missing methods
func TestOrder_AdditionalCoverage(t *testing.T) {
	t.Run("validateTrackingInfo_Carrier", func(t *testing.T) {
		order := generateTestOrder(t)

		// Carrier too long
		longCarrier := string(make([]byte, 51))
		order.Carrier = &longCarrier

		err := order.Validate()
		assert.Error(t, err)
	})

	t.Run("validateWebsite", func(t *testing.T) {
		customer := generateTestCustomer(t)

		// Invalid website
		invalidWebsite := "not-a-url"
		customer.Website = &invalidWebsite

		err := customer.Validate()
		assert.Error(t, err)

		// Valid website
		validWebsite := "https://example.com"
		customer.Website = &validWebsite

		err = customer.Validate()
		assert.NoError(t, err)
	})
}

// TestOrderCalculation_Comprehensive tests order calculation
func TestOrderCalculation_Comprehensive(t *testing.T) {
	order := generateTestOrder(t)
	order.DiscountAmount = decimal.Zero
	order.Items = []OrderItem{
		{
			Quantity:       2,
			UnitPrice:      decimal.NewFromFloat(25.00),
			DiscountAmount: decimal.NewFromFloat(5.00),
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

	// Calculate item totals first
	for i := range order.Items {
		order.Items[i].CalculateTotals()
	}

	// Calculate order totals
	err := order.CalculateTotals()
	require.NoError(t, err)

	// Verify subtotal is sum of item totals
	expectedSubtotal := decimal.Zero
	for _, item := range order.Items {
		expectedSubtotal = expectedSubtotal.Add(item.TotalPrice)
	}
	assert.True(t, order.Subtotal.Equal(expectedSubtotal))
}
