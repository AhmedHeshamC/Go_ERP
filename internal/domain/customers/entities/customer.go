package entities

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CustomerType represents the type of customer
type CustomerType string

const (
	CustomerTypeIndividual CustomerType = "individual"
	CustomerTypeBusiness   CustomerType = "business"
)

// CustomerStatus represents the status of a customer
type CustomerStatus string

const (
	CustomerStatusActive    CustomerStatus = "active"
	CustomerStatusInactive  CustomerStatus = "inactive"
	CustomerStatusSuspended CustomerStatus = "suspended"
)

// Customer represents a customer entity
type Customer struct {
	ID                uuid.UUID       `json:"id"`
	CustomerCode      string          `json:"customer_code"`
	CompanyID         *uuid.UUID      `json:"company_id,omitempty"`
	Type              string          `json:"type"`
	FirstName         string          `json:"first_name"`
	LastName          string          `json:"last_name"`
	Name              string          `json:"name"`
	Email             *string         `json:"email,omitempty"`
	Phone             *string         `json:"phone,omitempty"`
	Website           *string         `json:"website,omitempty"`
	Address           string          `json:"address"`
	CustomerType      CustomerType    `json:"customer_type"`
	Status            CustomerStatus  `json:"status"`
	CompanyName       *string         `json:"company_name,omitempty"`
	TaxID             *string         `json:"tax_id,omitempty"`
	Industry          *string         `json:"industry,omitempty"`
	CreditLimit       decimal.Decimal `json:"credit_limit"`
	CreditUsed        decimal.Decimal `json:"credit_used"`
	Terms             string          `json:"terms"`
	Active            bool            `json:"active"`
	IsVATExempt       bool            `json:"is_vat_exempt"`
	PreferredCurrency string          `json:"preferred_currency"`
	Notes             *string         `json:"notes,omitempty"`
	Source            string          `json:"source"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// NewCustomer creates a new customer entity with default values
func NewCustomer() *Customer {
	now := time.Now().UTC()
	return &Customer{
		ID:                uuid.New(),
		CustomerType:      CustomerTypeIndividual,
		Status:            CustomerStatusActive,
		CreditLimit:       decimal.Zero,
		CreditUsed:        decimal.Zero,
		Active:            true,
		IsVATExempt:       false,
		PreferredCurrency: "USD",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// Validate validates the customer entity
func (c *Customer) Validate() error {
	// Validate ID
	if c.ID == uuid.Nil {
		return errors.New("customer ID is required")
	}

	// Validate CustomerCode
	if c.CustomerCode == "" {
		return errors.New("customer code is required")
	}

	// Validate Name
	if c.Name == "" {
		return errors.New("customer name is required")
	}

	// Validate Email if provided
	if c.Email != nil && *c.Email != "" {
		if _, err := mail.ParseAddress(*c.Email); err != nil {
			return errors.New("invalid email format")
		}
	}

	// Validate Phone if provided
	if c.Phone != nil && *c.Phone != "" {
		if !isValidPhone(*c.Phone) {
			return errors.New("invalid phone number format")
		}
	}

	// Validate Website if provided
	if c.Website != nil && *c.Website != "" {
		if !isValidURL(*c.Website) {
			return errors.New("invalid website URL format")
		}
	}

	// Validate CustomerType
	if c.CustomerType != CustomerTypeIndividual && c.CustomerType != CustomerTypeBusiness {
		return errors.New("invalid customer type")
	}

	// Validate Status
	if c.Status != CustomerStatusActive &&
		c.Status != CustomerStatusInactive &&
		c.Status != CustomerStatusSuspended {
		return errors.New("invalid customer status")
	}

	// Validate CreditLimit
	if c.CreditLimit.IsNegative() {
		return errors.New("credit limit cannot be negative")
	}

	// Validate CreditUsed
	if c.CreditUsed.IsNegative() {
		return errors.New("credit used cannot be negative")
	}

	// Validate Terms
	if c.Terms == "" {
		return errors.New("payment terms are required")
	}

	// Validate PreferredCurrency
	if c.PreferredCurrency == "" {
		return errors.New("preferred currency is required")
	}

	// Validate Source
	if c.Source == "" {
		return errors.New("customer source is required")
	}

	// Validate TaxID if provided
	if c.TaxID != nil && *c.TaxID != "" {
		if !isValidTaxID(*c.TaxID) {
			return errors.New("invalid tax ID format")
		}
	}

	// Check if credit used exceeds credit limit
	if c.CreditUsed.GreaterThan(c.CreditLimit) {
		return errors.New("credit used cannot exceed credit limit")
	}

	return nil
}

// GetAvailableCredit returns the available credit for the customer
func (c *Customer) GetAvailableCredit() decimal.Decimal {
	return c.CreditLimit.Sub(c.CreditUsed)
}

// HasAvailableCredit checks if the customer has available credit for a given amount
func (c *Customer) HasAvailableCredit(amount decimal.Decimal) bool {
	return c.CreditUsed.Add(amount).LessThanOrEqual(c.CreditLimit)
}

// UseCredit adds to the credit used amount
func (c *Customer) UseCredit(amount decimal.Decimal) error {
	if amount.IsNegative() {
		return errors.New("credit amount cannot be negative")
	}

	if !c.HasAvailableCredit(amount) {
		return errors.New("insufficient credit available")
	}

	c.CreditUsed = c.CreditUsed.Add(amount)
	c.UpdatedAt = time.Now().UTC()
	return nil
}

// ReleaseCredit reduces the credit used amount
func (c *Customer) ReleaseCredit(amount decimal.Decimal) error {
	if amount.IsNegative() {
		return errors.New("credit amount cannot be negative")
	}

	if amount.GreaterThan(c.CreditUsed) {
		return errors.New("release amount cannot exceed credit used")
	}

	c.CreditUsed = c.CreditUsed.Sub(amount)
	c.UpdatedAt = time.Now().UTC()
	return nil
}

// IsIndividual returns true if the customer is an individual
func (c *Customer) IsIndividual() bool {
	return c.CustomerType == CustomerTypeIndividual
}

// IsBusiness returns true if the customer is a business
func (c *Customer) IsBusiness() bool {
	return c.CustomerType == CustomerTypeBusiness
}

// IsActive returns true if the customer status is active
func (c *Customer) IsActive() bool {
	return c.Status == CustomerStatusActive
}

// IsInactive returns true if the customer status is inactive
func (c *Customer) IsInactive() bool {
	return c.Status == CustomerStatusInactive
}

// IsSuspended returns true if the customer status is suspended
func (c *Customer) IsSuspended() bool {
	return c.Status == CustomerStatusSuspended
}

// Activate sets the customer status to active
func (c *Customer) Activate() {
	c.Status = CustomerStatusActive
	c.Active = true
	c.UpdatedAt = time.Now().UTC()
}

// Deactivate sets the customer status to inactive
func (c *Customer) Deactivate() {
	c.Status = CustomerStatusInactive
	c.Active = false
	c.UpdatedAt = time.Now().UTC()
}

// Suspend sets the customer status to suspended
func (c *Customer) Suspend() {
	c.Status = CustomerStatusSuspended
	c.Active = false
	c.UpdatedAt = time.Now().UTC()
}

// UpdateContactInfo updates the customer contact information
func (c *Customer) UpdateContactInfo(email, phone, address string) error {
	if email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			return errors.New("invalid email format")
		}
		emailStr := strings.ToLower(strings.TrimSpace(email))
		c.Email = &emailStr
	}

	if phone != "" {
		if !isValidPhone(phone) {
			return errors.New("invalid phone number format")
		}
		phoneStr := strings.TrimSpace(phone)
		c.Phone = &phoneStr
	}

	if address != "" {
		c.Address = strings.TrimSpace(address)
	}

	c.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateName updates the customer name
func (c *Customer) UpdateName(firstName, lastName string) {
	c.FirstName = strings.TrimSpace(firstName)
	c.LastName = strings.TrimSpace(lastName)
	c.Name = strings.TrimSpace(c.FirstName + " " + c.LastName)
	c.UpdatedAt = time.Now().UTC()
}

// UpdateCreditLimit updates the credit limit
func (c *Customer) UpdateCreditLimit(limit decimal.Decimal) error {
	if limit.IsNegative() {
		return errors.New("credit limit cannot be negative")
	}

	// Check if reducing limit would cause credit used to exceed limit
	if c.CreditUsed.GreaterThan(limit) {
		return errors.New("cannot set credit limit lower than current credit used")
	}

	c.CreditLimit = limit
	c.UpdatedAt = time.Now().UTC()
	return nil
}

// isValidPhone validates a phone number format
func isValidPhone(phone string) bool {
	// Remove all non-digit characters
	digits := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")

	// Check if the phone number has a reasonable number of digits
	return len(digits) >= 10 && len(digits) <= 15
}

// isValidURL validates a URL format
func isValidURL(url string) bool {
	// Simple URL validation - could be improved with more sophisticated parsing
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "www.")
}

// isValidTaxID validates a tax ID format (basic implementation)
func isValidTaxID(taxID string) bool {
	// Remove all non-alphanumeric characters
	cleaned := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(taxID, "")

	// Check if the tax ID has a reasonable length
	return len(cleaned) >= 5 && len(cleaned) <= 20
}
