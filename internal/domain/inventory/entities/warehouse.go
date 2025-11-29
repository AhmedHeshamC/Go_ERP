package entities

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Warehouse represents a storage location in the system
type Warehouse struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	Name       string     `json:"name" db:"name"`
	Code       string     `json:"code" db:"code"`
	Address    string     `json:"address" db:"address"`
	City       string     `json:"city" db:"city"`
	State      string     `json:"state" db:"state"`
	Country    string     `json:"country" db:"country"`
	PostalCode string     `json:"postal_code" db:"postal_code"`
	Phone      string     `json:"phone,omitempty" db:"phone"`
	Email      string     `json:"email,omitempty" db:"email"`
	ManagerID  *uuid.UUID `json:"manager_id,omitempty" db:"manager_id"`
	IsActive   bool       `json:"is_active" db:"is_active"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// Validate validates the warehouse entity
func (w *Warehouse) Validate() error {
	var errs []error

	// Validate UUID
	if w.ID == uuid.Nil {
		errs = append(errs, errors.New("warehouse ID cannot be empty"))
	}

	// Validate name
	if err := w.validateName(); err != nil {
		errs = append(errs, fmt.Errorf("invalid name: %w", err))
	}

	// Validate code
	if err := w.validateCode(); err != nil {
		errs = append(errs, fmt.Errorf("invalid code: %w", err))
	}

	// Validate address components
	if err := w.validateAddress(); err != nil {
		errs = append(errs, fmt.Errorf("invalid address: %w", err))
	}

	// Validate contact information
	if err := w.validateContactInfo(); err != nil {
		errs = append(errs, fmt.Errorf("invalid contact information: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// validateName validates the warehouse name
func (w *Warehouse) validateName() error {
	name := strings.TrimSpace(w.Name)
	if name == "" {
		return errors.New("warehouse name cannot be empty")
	}

	if len(name) > 200 {
		return errors.New("warehouse name cannot exceed 200 characters")
	}

	// Name should contain valid characters
	nameRegex := regexp.MustCompile(`^[a-zA-Z0-9\s\-_&,.'()]+$`)
	if !nameRegex.MatchString(name) {
		return errors.New("warehouse name contains invalid characters")
	}

	return nil
}

// validateCode validates the warehouse code
func (w *Warehouse) validateCode() error {
	code := strings.TrimSpace(w.Code)
	if code == "" {
		return errors.New("warehouse code cannot be empty")
	}

	if len(code) < 2 || len(code) > 20 {
		return errors.New("warehouse code must be between 2 and 20 characters")
	}

	// Code should be uppercase alphanumeric with underscores and hyphens
	codeRegex := regexp.MustCompile(`^[A-Z0-9\-_]+$`)
	if !codeRegex.MatchString(code) {
		return errors.New("warehouse code must be uppercase alphanumeric with hyphens and underscores")
	}

	return nil
}

// validateAddress validates the address components
func (w *Warehouse) validateAddress() error {
	// Address validation
	if strings.TrimSpace(w.Address) == "" {
		return errors.New("address cannot be empty")
	}

	if len(w.Address) > 500 {
		return errors.New("address cannot exceed 500 characters")
	}

	// City validation
	if strings.TrimSpace(w.City) == "" {
		return errors.New("city cannot be empty")
	}

	if len(w.City) > 100 {
		return errors.New("city cannot exceed 100 characters")
	}

	// City name validation
	cityRegex := regexp.MustCompile(`^[a-zA-Z\s\-_.'&]+$`)
	if !cityRegex.MatchString(strings.TrimSpace(w.City)) {
		return errors.New("city contains invalid characters")
	}

	// State validation (optional for countries without states)
	if w.State != "" {
		if len(w.State) > 100 {
			return errors.New("state cannot exceed 100 characters")
		}

		stateRegex := regexp.MustCompile(`^[a-zA-Z\s\-_.'&]+$`)
		if !stateRegex.MatchString(strings.TrimSpace(w.State)) {
			return errors.New("state contains invalid characters")
		}
	}

	// Country validation
	if strings.TrimSpace(w.Country) == "" {
		return errors.New("country cannot be empty")
	}

	if len(w.Country) > 100 {
		return errors.New("country cannot exceed 100 characters")
	}

	countryRegex := regexp.MustCompile(`^[a-zA-Z\s\-_.'&]+$`)
	if !countryRegex.MatchString(strings.TrimSpace(w.Country)) {
		return errors.New("country contains invalid characters")
	}

	// Postal code validation
	if strings.TrimSpace(w.PostalCode) == "" {
		return errors.New("postal code cannot be empty")
	}

	if len(w.PostalCode) > 20 {
		return errors.New("postal code cannot exceed 20 characters")
	}

	// Basic postal code regex (allows for international formats)
	postalRegex := regexp.MustCompile(`^[a-zA-Z0-9\s\-_]+$`)
	if !postalRegex.MatchString(strings.TrimSpace(w.PostalCode)) {
		return errors.New("postal code contains invalid characters")
	}

	return nil
}

// validateContactInfo validates phone and email
func (w *Warehouse) validateContactInfo() error {
	// Phone validation (optional)
	if w.Phone != "" {
		if err := w.validatePhone(); err != nil {
			return fmt.Errorf("invalid phone: %w", err)
		}
	}

	// Email validation (optional)
	if w.Email != "" {
		if err := w.validateEmail(); err != nil {
			return fmt.Errorf("invalid email: %w", err)
		}
	}

	return nil
}

// validatePhone validates the phone number format
func (w *Warehouse) validatePhone() error {
	phone := strings.TrimSpace(w.Phone)
	if phone == "" {
		return nil // Phone is optional
	}

	// Basic phone regex - allows international format with +, spaces, hyphens, and parentheses
	phoneRegex := regexp.MustCompile(`^\+?[\d\s\-\(\)]{7,20}$`)
	if !phoneRegex.MatchString(phone) {
		return errors.New("invalid phone number format")
	}

	return nil
}

// validateEmail validates the email format
func (w *Warehouse) validateEmail() error {
	email := strings.TrimSpace(w.Email)
	if email == "" {
		return nil // Email is optional
	}

	// Basic email regex pattern
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}

	if len(email) > 255 {
		return errors.New("email cannot exceed 255 characters")
	}

	return nil
}

// Business Logic Methods

// IsActiveWarehouse returns true if the warehouse is active
func (w *Warehouse) IsActiveWarehouse() bool {
	return w.IsActive
}

// GetFullAddress returns the complete formatted address
func (w *Warehouse) GetFullAddress() string {
	var parts []string

	if w.Address != "" {
		parts = append(parts, w.Address)
	}

	var cityStatePostal []string
	if w.City != "" {
		cityStatePostal = append(cityStatePostal, w.City)
	}
	if w.State != "" {
		cityStatePostal = append(cityStatePostal, w.State)
	}
	if w.PostalCode != "" {
		cityStatePostal = append(cityStatePostal, w.PostalCode)
	}

	if len(cityStatePostal) > 0 {
		parts = append(parts, strings.Join(cityStatePostal, ", "))
	}

	if w.Country != "" {
		parts = append(parts, w.Country)
	}

	return strings.Join(parts, "\n")
}

// Activate activates the warehouse
func (w *Warehouse) Activate() {
	w.IsActive = true
	w.UpdatedAt = time.Now().UTC()
}

// Deactivate deactivates the warehouse
func (w *Warehouse) Deactivate() {
	w.IsActive = false
	w.UpdatedAt = time.Now().UTC()
}

// UpdateManager updates the warehouse manager
func (w *Warehouse) UpdateManager(managerID *uuid.UUID) {
	w.ManagerID = managerID
	w.UpdatedAt = time.Now().UTC()
}

// UpdateDetails updates warehouse basic information
func (w *Warehouse) UpdateDetails(name, address, city, state, country, postalCode string) error {
	// Create temporary warehouse for validation
	tempW := &Warehouse{
		Name:       name,
		Code:       w.Code, // Keep existing code for validation
		Address:    address,
		City:       city,
		State:      state,
		Country:    country,
		PostalCode: postalCode,
	}

	if err := tempW.validateName(); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	if err := tempW.validateAddress(); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	w.Name = name
	w.Address = address
	w.City = city
	w.State = state
	w.Country = country
	w.PostalCode = postalCode
	w.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateContactInfo updates warehouse contact information
func (w *Warehouse) UpdateContactInfo(phone, email string) error {
	// Create temporary warehouse for validation
	tempW := &Warehouse{
		Phone: phone,
		Email: email,
	}

	if err := tempW.validateContactInfo(); err != nil {
		return fmt.Errorf("invalid contact information: %w", err)
	}

	w.Phone = phone
	w.Email = email
	w.UpdatedAt = time.Now().UTC()
	return nil
}

// SetManager assigns a manager to the warehouse
func (w *Warehouse) SetManager(managerID uuid.UUID) error {
	if managerID == uuid.Nil {
		return errors.New("manager ID cannot be empty")
	}

	w.ManagerID = &managerID
	w.UpdatedAt = time.Now().UTC()
	return nil
}

// RemoveManager removes the assigned manager from the warehouse
func (w *Warehouse) RemoveManager() {
	w.ManagerID = nil
	w.UpdatedAt = time.Now().UTC()
}

// HasManager returns true if the warehouse has an assigned manager
func (w *Warehouse) HasManager() bool {
	return w.ManagerID != nil
}

// IsManager returns true if the given user ID is the warehouse manager
func (w *Warehouse) IsManager(userID uuid.UUID) bool {
	return w.ManagerID != nil && *w.ManagerID == userID
}

// ToSafeWarehouse returns a warehouse object without sensitive information
func (w *Warehouse) ToSafeWarehouse() *Warehouse {
	return &Warehouse{
		ID:         w.ID,
		Name:       w.Name,
		Code:       w.Code,
		Address:    w.Address,
		City:       w.City,
		State:      w.State,
		Country:    w.Country,
		PostalCode: w.PostalCode,
		Phone:      w.Phone,
		Email:      w.Email,
		ManagerID:  w.ManagerID,
		IsActive:   w.IsActive,
		CreatedAt:  w.CreatedAt,
		UpdatedAt:  w.UpdatedAt,
	}
}

// WarehouseType represents different types of warehouses
type WarehouseType string

const (
	WarehouseTypeRetail       WarehouseType = "RETAIL"
	WarehouseTypeWholesale    WarehouseType = "WHOLESALE"
	WarehouseTypeDistribution WarehouseType = "DISTRIBUTION"
	WarehouseTypeFulfillment  WarehouseType = "FULFILLMENT"
	WarehouseTypeReturn       WarehouseType = "RETURN"
)

// WarehouseExtended represents an extended warehouse with additional metadata
type WarehouseExtended struct {
	Warehouse
	Type                  WarehouseType `json:"type" db:"type"`
	Capacity              *int          `json:"capacity,omitempty" db:"capacity"`
	SquareFootage         *int          `json:"square_footage,omitempty" db:"square_footage"`
	DockCount             *int          `json:"dock_count,omitempty" db:"dock_count"`
	TemperatureControlled bool          `json:"temperature_controlled" db:"temperature_controlled"`
	SecurityLevel         int           `json:"security_level" db:"security_level"`
	Description           string        `json:"description,omitempty" db:"description"`
}

// Validate validates the extended warehouse entity
func (we *WarehouseExtended) Validate() error {
	// Validate base warehouse first
	if err := we.Warehouse.Validate(); err != nil {
		return err
	}

	var errs []error

	// Validate warehouse type
	if err := we.validateType(); err != nil {
		errs = append(errs, fmt.Errorf("invalid type: %w", err))
	}

	// Validate capacity (optional)
	if we.Capacity != nil {
		if *we.Capacity < 0 {
			errs = append(errs, errors.New("capacity cannot be negative"))
		}
		if *we.Capacity > 999999999 {
			errs = append(errs, errors.New("capacity cannot exceed 999,999,999"))
		}
	}

	// Validate square footage (optional)
	if we.SquareFootage != nil {
		if *we.SquareFootage < 0 {
			errs = append(errs, errors.New("square footage cannot be negative"))
		}
		if *we.SquareFootage > 999999999 {
			errs = append(errs, errors.New("square footage cannot exceed 999,999,999"))
		}
	}

	// Validate dock count (optional)
	if we.DockCount != nil {
		if *we.DockCount < 0 {
			errs = append(errs, errors.New("dock count cannot be negative"))
		}
		if *we.DockCount > 9999 {
			errs = append(errs, errors.New("dock count cannot exceed 9,999"))
		}
	}

	// Validate security level
	if we.SecurityLevel < 0 || we.SecurityLevel > 10 {
		errs = append(errs, errors.New("security level must be between 0 and 10"))
	}

	// Validate description (optional)
	if we.Description != "" && len(we.Description) > 2000 {
		errs = append(errs, errors.New("description cannot exceed 2000 characters"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// validateType validates the warehouse type
func (we *WarehouseExtended) validateType() error {
	validTypes := map[WarehouseType]bool{
		WarehouseTypeRetail:       true,
		WarehouseTypeWholesale:    true,
		WarehouseTypeDistribution: true,
		WarehouseTypeFulfillment:  true,
		WarehouseTypeReturn:       true,
	}

	if !validTypes[we.Type] {
		return fmt.Errorf("invalid warehouse type: %s", we.Type)
	}

	return nil
}

// GetTypeName returns the human-readable name of the warehouse type
func (we *WarehouseExtended) GetTypeName() string {
	switch we.Type {
	case WarehouseTypeRetail:
		return "Retail"
	case WarehouseTypeWholesale:
		return "Wholesale"
	case WarehouseTypeDistribution:
		return "Distribution"
	case WarehouseTypeFulfillment:
		return "Fulfillment"
	case WarehouseTypeReturn:
		return "Return"
	default:
		return "Unknown"
	}
}

// UpdateCapacity updates the warehouse capacity
func (we *WarehouseExtended) UpdateCapacity(capacity int) error {
	if capacity < 0 {
		return errors.New("capacity cannot be negative")
	}

	if capacity > 999999999 {
		return errors.New("capacity cannot exceed 999,999,999")
	}

	we.Capacity = &capacity
	we.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateType updates the warehouse type
func (we *WarehouseExtended) UpdateType(warehouseType WarehouseType) error {
	tempWe := &WarehouseExtended{Type: warehouseType}
	if err := tempWe.validateType(); err != nil {
		return fmt.Errorf("invalid warehouse type: %w", err)
	}

	we.Type = warehouseType
	we.UpdatedAt = time.Now().UTC()
	return nil
}

// GetUtilizationPercentage returns the warehouse utilization as a percentage
func (we *WarehouseExtended) GetUtilizationPercentage(currentStock int) (float64, error) {
	if we.Capacity == nil || *we.Capacity == 0 {
		return 0, errors.New("warehouse capacity not set")
	}

	if currentStock < 0 {
		return 0, errors.New("current stock cannot be negative")
	}

	if currentStock > *we.Capacity {
		return 100, nil // Cap at 100%
	}

	utilization := float64(currentStock) / float64(*we.Capacity) * 100
	return utilization, nil
}
