package entities

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestNewCustomer(t *testing.T) {
	customer := NewCustomer()

	if customer.ID == uuid.Nil {
		t.Error("Expected customer ID to be set")
	}

	if customer.CustomerType != CustomerTypeIndividual {
		t.Errorf("Expected customer type to be %s, got %s", CustomerTypeIndividual, customer.CustomerType)
	}

	if customer.Status != CustomerStatusActive {
		t.Errorf("Expected customer status to be %s, got %s", CustomerStatusActive, customer.Status)
	}

	if !customer.CreditLimit.IsZero() {
		t.Error("Expected credit limit to be zero")
	}

	if !customer.CreditUsed.IsZero() {
		t.Error("Expected credit used to be zero")
	}

	if !customer.Active {
		t.Error("Expected customer to be active")
	}

	if customer.IsVATExempt {
		t.Error("Expected VAT exempt to be false")
	}

	if customer.PreferredCurrency != "USD" {
		t.Errorf("Expected preferred currency to be USD, got %s", customer.PreferredCurrency)
	}

	if customer.CreatedAt.IsZero() {
		t.Error("Expected created at to be set")
	}

	if customer.UpdatedAt.IsZero() {
		t.Error("Expected updated at to be set")
	}
}

func TestCustomer_Validate(t *testing.T) {
	tests := []struct {
		name      string
		customer  *Customer
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid customer",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "CUST-12345",
				Name:              "John Doe",
				Email:             stringPtr("john.doe@example.com"),
				CustomerType:      CustomerTypeIndividual,
				Status:            CustomerStatusActive,
				CreditLimit:       decimal.NewFromFloat(1000),
				CreditUsed:        decimal.Zero,
				Terms:             "NET30",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "web",
			},
			expectErr: false,
		},
		{
			name: "Valid business customer",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "CUST-54321",
				Name:              "Acme Corporation",
				Email:             stringPtr("info@acme.com"),
				CustomerType:      CustomerTypeBusiness,
				Status:            CustomerStatusActive,
				CompanyName:       stringPtr("Acme Corporation"),
				CreditLimit:       decimal.NewFromFloat(5000),
				CreditUsed:        decimal.NewFromFloat(1000),
				Terms:             "NET60",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "sales",
			},
			expectErr: false,
		},
		{
			name: "Invalid ID",
			customer: &Customer{
				ID:                uuid.Nil,
				CustomerCode:      "CUST-12345",
				Name:              "John Doe",
				CustomerType:      CustomerTypeIndividual,
				Status:            CustomerStatusActive,
				CreditLimit:       decimal.NewFromFloat(1000),
				CreditUsed:        decimal.Zero,
				Terms:             "NET30",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "web",
			},
			expectErr: true,
			errMsg:    "customer ID is required",
		},
		{
			name: "Empty customer code",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "",
				Name:              "John Doe",
				CustomerType:      CustomerTypeIndividual,
				Status:            CustomerStatusActive,
				CreditLimit:       decimal.NewFromFloat(1000),
				CreditUsed:        decimal.Zero,
				Terms:             "NET30",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "web",
			},
			expectErr: true,
			errMsg:    "customer code is required",
		},
		{
			name: "Empty name",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "CUST-12345",
				Name:              "",
				CustomerType:      CustomerTypeIndividual,
				Status:            CustomerStatusActive,
				CreditLimit:       decimal.NewFromFloat(1000),
				CreditUsed:        decimal.Zero,
				Terms:             "NET30",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "web",
			},
			expectErr: true,
			errMsg:    "customer name is required",
		},
		{
			name: "Invalid email",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "CUST-12345",
				Name:              "John Doe",
				Email:             stringPtr("invalid-email"),
				CustomerType:      CustomerTypeIndividual,
				Status:            CustomerStatusActive,
				CreditLimit:       decimal.NewFromFloat(1000),
				CreditUsed:        decimal.Zero,
				Terms:             "NET30",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "web",
			},
			expectErr: true,
			errMsg:    "invalid email format",
		},
		{
			name: "Invalid phone",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "CUST-12345",
				Name:              "John Doe",
				Phone:             stringPtr("123"),
				CustomerType:      CustomerTypeIndividual,
				Status:            CustomerStatusActive,
				CreditLimit:       decimal.NewFromFloat(1000),
				CreditUsed:        decimal.Zero,
				Terms:             "NET30",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "web",
			},
			expectErr: true,
			errMsg:    "invalid phone number format",
		},
		{
			name: "Invalid customer type",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "CUST-12345",
				Name:              "John Doe",
				CustomerType:      CustomerType("invalid"),
				Status:            CustomerStatusActive,
				CreditLimit:       decimal.NewFromFloat(1000),
				CreditUsed:        decimal.Zero,
				Terms:             "NET30",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "web",
			},
			expectErr: true,
			errMsg:    "invalid customer type",
		},
		{
			name: "Invalid status",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "CUST-12345",
				Name:              "John Doe",
				CustomerType:      CustomerTypeIndividual,
				Status:            CustomerStatus("invalid"),
				CreditLimit:       decimal.NewFromFloat(1000),
				CreditUsed:        decimal.Zero,
				Terms:             "NET30",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "web",
			},
			expectErr: true,
			errMsg:    "invalid customer status",
		},
		{
			name: "Negative credit limit",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "CUST-12345",
				Name:              "John Doe",
				CustomerType:      CustomerTypeIndividual,
				Status:            CustomerStatusActive,
				CreditLimit:       decimal.NewFromFloat(-100),
				CreditUsed:        decimal.Zero,
				Terms:             "NET30",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "web",
			},
			expectErr: true,
			errMsg:    "credit limit cannot be negative",
		},
		{
			name: "Negative credit used",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "CUST-12345",
				Name:              "John Doe",
				CustomerType:      CustomerTypeIndividual,
				Status:            CustomerStatusActive,
				CreditLimit:       decimal.NewFromFloat(1000),
				CreditUsed:        decimal.NewFromFloat(-50),
				Terms:             "NET30",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "web",
			},
			expectErr: true,
			errMsg:    "credit used cannot be negative",
		},
		{
			name: "Empty terms",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "CUST-12345",
				Name:              "John Doe",
				CustomerType:      CustomerTypeIndividual,
				Status:            CustomerStatusActive,
				CreditLimit:       decimal.NewFromFloat(1000),
				CreditUsed:        decimal.Zero,
				Terms:             "",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "web",
			},
			expectErr: true,
			errMsg:    "payment terms are required",
		},
		{
			name: "Empty preferred currency",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "CUST-12345",
				Name:              "John Doe",
				CustomerType:      CustomerTypeIndividual,
				Status:            CustomerStatusActive,
				CreditLimit:       decimal.NewFromFloat(1000),
				CreditUsed:        decimal.Zero,
				Terms:             "NET30",
				Active:            true,
				PreferredCurrency: "",
				Source:            "web",
			},
			expectErr: true,
			errMsg:    "preferred currency is required",
		},
		{
			name: "Empty source",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "CUST-12345",
				Name:              "John Doe",
				CustomerType:      CustomerTypeIndividual,
				Status:            CustomerStatusActive,
				CreditLimit:       decimal.NewFromFloat(1000),
				CreditUsed:        decimal.Zero,
				Terms:             "NET30",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "",
			},
			expectErr: true,
			errMsg:    "customer source is required",
		},
		{
			name: "Credit used exceeds credit limit",
			customer: &Customer{
				ID:                uuid.New(),
				CustomerCode:      "CUST-12345",
				Name:              "John Doe",
				CustomerType:      CustomerTypeIndividual,
				Status:            CustomerStatusActive,
				CreditLimit:       decimal.NewFromFloat(1000),
				CreditUsed:        decimal.NewFromFloat(1500),
				Terms:             "NET30",
				Active:            true,
				PreferredCurrency: "USD",
				Source:            "web",
			},
			expectErr: true,
			errMsg:    "credit used cannot exceed credit limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.customer.Validate()
			if (err != nil) != tt.expectErr {
				t.Errorf("Validate() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, expected message %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestCustomer_GetAvailableCredit(t *testing.T) {
	customer := &Customer{
		CreditLimit: decimal.NewFromFloat(1000),
		CreditUsed:  decimal.NewFromFloat(300),
	}

	available := customer.GetAvailableCredit()
	expected := decimal.NewFromFloat(700)
	if !available.Equal(expected) {
		t.Errorf("Expected available credit to be %s, got %s", expected, available)
	}
}

func TestCustomer_HasAvailableCredit(t *testing.T) {
	tests := []struct {
		name         string
		creditLimit  float64
		creditUsed   float64
		amount       float64
		expectedBool bool
	}{
		{
			name:         "Sufficient credit",
			creditLimit:  1000,
			creditUsed:   300,
			amount:       500,
			expectedBool: true,
		},
		{
			name:         "Exact credit limit",
			creditLimit:  1000,
			creditUsed:   700,
			amount:       300,
			expectedBool: true,
		},
		{
			name:         "Insufficient credit",
			creditLimit:  1000,
			creditUsed:   700,
			amount:       400,
			expectedBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customer := &Customer{
				CreditLimit: decimal.NewFromFloat(tt.creditLimit),
				CreditUsed:  decimal.NewFromFloat(tt.creditUsed),
			}

			hasCredit := customer.HasAvailableCredit(decimal.NewFromFloat(tt.amount))
			if hasCredit != tt.expectedBool {
				t.Errorf("HasAvailableCredit() = %v, expected %v", hasCredit, tt.expectedBool)
			}
		})
	}
}

func TestCustomer_UseCredit(t *testing.T) {
	tests := []struct {
		name        string
		creditLimit float64
		creditUsed  float64
		amount      float64
		expectErr   bool
		errMsg      string
		finalUsed   float64
	}{
		{
			name:        "Use credit successfully",
			creditLimit: 1000,
			creditUsed:  300,
			amount:      200,
			expectErr:   false,
			finalUsed:   500,
		},
		{
			name:        "Negative amount",
			creditLimit: 1000,
			creditUsed:  300,
			amount:      -100,
			expectErr:   true,
			errMsg:      "credit amount cannot be negative",
			finalUsed:   300, // Unchanged
		},
		{
			name:        "Insufficient credit",
			creditLimit: 1000,
			creditUsed:  800,
			amount:      300,
			expectErr:   true,
			errMsg:      "insufficient credit available",
			finalUsed:   800, // Unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customer := &Customer{
				CreditLimit: decimal.NewFromFloat(tt.creditLimit),
				CreditUsed:  decimal.NewFromFloat(tt.creditUsed),
			}

			err := customer.UseCredit(decimal.NewFromFloat(tt.amount))
			if (err != nil) != tt.expectErr {
				t.Errorf("UseCredit() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("UseCredit() error = %v, expected message %v", err.Error(), tt.errMsg)
			}

			if !customer.CreditUsed.Equal(decimal.NewFromFloat(tt.finalUsed)) {
				t.Errorf("Expected credit used to be %v, got %v", decimal.NewFromFloat(tt.finalUsed), customer.CreditUsed)
			}
		})
	}
}

func TestCustomer_ReleaseCredit(t *testing.T) {
	tests := []struct {
		name        string
		creditLimit float64
		creditUsed  float64
		amount      float64
		expectErr   bool
		errMsg      string
		finalUsed   float64
	}{
		{
			name:        "Release credit successfully",
			creditLimit: 1000,
			creditUsed:  500,
			amount:      200,
			expectErr:   false,
			finalUsed:   300,
		},
		{
			name:        "Negative amount",
			creditLimit: 1000,
			creditUsed:  500,
			amount:      -100,
			expectErr:   true,
			errMsg:      "credit amount cannot be negative",
			finalUsed:   500, // Unchanged
		},
		{
			name:        "Release exceeds credit used",
			creditLimit: 1000,
			creditUsed:  500,
			amount:      600,
			expectErr:   true,
			errMsg:      "release amount cannot exceed credit used",
			finalUsed:   500, // Unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customer := &Customer{
				CreditLimit: decimal.NewFromFloat(tt.creditLimit),
				CreditUsed:  decimal.NewFromFloat(tt.creditUsed),
			}

			err := customer.ReleaseCredit(decimal.NewFromFloat(tt.amount))
			if (err != nil) != tt.expectErr {
				t.Errorf("ReleaseCredit() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("ReleaseCredit() error = %v, expected message %v", err.Error(), tt.errMsg)
			}

			if !customer.CreditUsed.Equal(decimal.NewFromFloat(tt.finalUsed)) {
				t.Errorf("Expected credit used to be %v, got %v", decimal.NewFromFloat(tt.finalUsed), customer.CreditUsed)
			}
		})
	}
}

func TestCustomer_TypeMethods(t *testing.T) {
	individual := &Customer{
		CustomerType: CustomerTypeIndividual,
	}

	business := &Customer{
		CustomerType: CustomerTypeBusiness,
	}

	// Test IsIndividual
	if !individual.IsIndividual() {
		t.Error("Expected individual.IsIndividual() to be true")
	}

	if business.IsIndividual() {
		t.Error("Expected business.IsIndividual() to be false")
	}

	// Test IsBusiness
	if individual.IsBusiness() {
		t.Error("Expected individual.IsBusiness() to be false")
	}

	if !business.IsBusiness() {
		t.Error("Expected business.IsBusiness() to be true")
	}
}

func TestCustomer_StatusMethods(t *testing.T) {
	tests := []struct {
		name          string
		status        CustomerStatus
		isActive      bool
		isInactive    bool
		isSuspended   bool
		afterActivate CustomerStatus
	}{
		{
			name:          "Active status",
			status:        CustomerStatusActive,
			isActive:      true,
			isInactive:    false,
			isSuspended:   false,
			afterActivate: CustomerStatusActive,
		},
		{
			name:          "Inactive status",
			status:        CustomerStatusInactive,
			isActive:      false,
			isInactive:    true,
			isSuspended:   false,
			afterActivate: CustomerStatusActive,
		},
		{
			name:          "Suspended status",
			status:        CustomerStatusSuspended,
			isActive:      false,
			isInactive:    false,
			isSuspended:   true,
			afterActivate: CustomerStatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customer := &Customer{
				Status: tt.status,
				Active: tt.isActive,
			}

			// Test status check methods
			if customer.IsActive() != tt.isActive {
				t.Errorf("IsActive() = %v, expected %v", customer.IsActive(), tt.isActive)
			}

			if customer.IsInactive() != tt.isInactive {
				t.Errorf("IsInactive() = %v, expected %v", customer.IsInactive(), tt.isInactive)
			}

			if customer.IsSuspended() != tt.isSuspended {
				t.Errorf("IsSuspended() = %v, expected %v", customer.IsSuspended(), tt.isSuspended)
			}

			// Test Activate method
			customer.Activate()
			if customer.Status != CustomerStatusActive {
				t.Errorf("After Activate(), status = %v, expected %v", customer.Status, CustomerStatusActive)
			}
			if !customer.Active {
				t.Error("After Activate(), Active should be true")
			}
		})
	}

	// Test Deactivate method
	customer := &Customer{
		Status: CustomerStatusActive,
		Active: true,
	}

	customer.Deactivate()
	if customer.Status != CustomerStatusInactive {
		t.Errorf("After Deactivate(), status = %v, expected %v", customer.Status, CustomerStatusInactive)
	}
	if customer.Active {
		t.Error("After Deactivate(), Active should be false")
	}

	// Test Suspend method
	customer.Suspend()
	if customer.Status != CustomerStatusSuspended {
		t.Errorf("After Suspend(), status = %v, expected %v", customer.Status, CustomerStatusSuspended)
	}
	if customer.Active {
		t.Error("After Suspend(), Active should be false")
	}
}

func TestCustomer_UpdateContactInfo(t *testing.T) {
	customer := &Customer{
		Name:    "John Doe",
		Address: "123 Main St",
	}

	// Test valid update
	err := customer.UpdateContactInfo("newemail@example.com", "+1-555-123-4567", "456 Oak Ave")
	if err != nil {
		t.Errorf("UpdateContactInfo() error = %v", err)
	}

	if customer.Email == nil || *customer.Email != "newemail@example.com" {
		t.Error("Expected email to be updated")
	}

	if customer.Phone == nil || *customer.Phone != "+1-555-123-4567" {
		t.Error("Expected phone to be updated")
	}

	if customer.Address != "456 Oak Ave" {
		t.Error("Expected address to be updated")
	}

	// Test invalid email
	err = customer.UpdateContactInfo("invalid-email", "+1-555-123-4567", "456 Oak Ave")
	if err == nil {
		t.Error("Expected error for invalid email")
	}

	// Test invalid phone
	err = customer.UpdateContactInfo("valid@example.com", "123", "456 Oak Ave")
	if err == nil {
		t.Error("Expected error for invalid phone")
	}
}

func TestCustomer_UpdateName(t *testing.T) {
	customer := &Customer{
		FirstName: "John",
		LastName:  "Doe",
		Name:      "John Doe",
	}

	customer.UpdateName("Jane", "Smith")
	if customer.FirstName != "Jane" {
		t.Errorf("Expected FirstName to be Jane, got %s", customer.FirstName)
	}

	if customer.LastName != "Smith" {
		t.Errorf("Expected LastName to be Smith, got %s", customer.LastName)
	}

	if customer.Name != "Jane Smith" {
		t.Errorf("Expected Name to be Jane Smith, got %s", customer.Name)
	}
}

func TestCustomer_UpdateCreditLimit(t *testing.T) {
	customer := &Customer{
		CreditLimit: decimal.NewFromFloat(1000),
		CreditUsed:  decimal.NewFromFloat(300),
	}

	// Test valid update
	err := customer.UpdateCreditLimit(decimal.NewFromFloat(2000))
	if err != nil {
		t.Errorf("UpdateCreditLimit() error = %v", err)
	}

	if !customer.CreditLimit.Equal(decimal.NewFromFloat(2000)) {
		t.Error("Expected credit limit to be updated")
	}

	// Test negative limit
	err = customer.UpdateCreditLimit(decimal.NewFromFloat(-100))
	if err == nil {
		t.Error("Expected error for negative credit limit")
	}

	// Test limit lower than credit used
	err = customer.UpdateCreditLimit(decimal.NewFromFloat(200))
	if err == nil {
		t.Error("Expected error when setting limit lower than credit used")
	}
}

// Helper function to get a pointer to a string
func stringPtr(s string) *string {
	return &s
}
