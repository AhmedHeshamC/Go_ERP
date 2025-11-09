package entities

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Product represents a product in the system
type Product struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	SKU             string          `json:"sku" db:"sku"`
	Name            string          `json:"name" db:"name"`
	Description     string          `json:"description" db:"description"`
	ShortDescription string         `json:"short_description" db:"short_description"`
	CategoryID      uuid.UUID       `json:"category_id" db:"category_id"`
	Price           decimal.Decimal `json:"price" db:"price"`
	Cost            decimal.Decimal `json:"cost" db:"cost"`
	Weight          float64         `json:"weight" db:"weight"`
	Dimensions      string          `json:"dimensions" db:"dimensions"`
	Length          float64         `json:"length" db:"length"`
	Width           float64         `json:"width" db:"width"`
	Height          float64         `json:"height" db:"height"`
	Volume          float64         `json:"volume" db:"volume"`
	Barcode         string          `json:"barcode" db:"barcode"`
	TrackInventory  bool            `json:"track_inventory" db:"track_inventory"`
	StockQuantity   int             `json:"stock_quantity" db:"stock_quantity"`
	MinStockLevel   int             `json:"min_stock_level" db:"min_stock_level"`
	MaxStockLevel   int             `json:"max_stock_level" db:"max_stock_level"`
	AllowBackorder  bool            `json:"allow_backorder" db:"allow_backorder"`
	RequiresShipping bool           `json:"requires_shipping" db:"requires_shipping"`
	Taxable         bool            `json:"taxable" db:"taxable"`
	TaxRate         decimal.Decimal `json:"tax_rate" db:"tax_rate"`
	IsActive        bool            `json:"is_active" db:"is_active"`
	IsFeatured      bool            `json:"is_featured" db:"is_featured"`
	IsDigital       bool            `json:"is_digital" db:"is_digital"`
	DownloadURL     string          `json:"download_url" db:"download_url"`
	MaxDownloads    int             `json:"max_downloads" db:"max_downloads"`
	ExpiryDays      int             `json:"expiry_days" db:"expiry_days"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// Validate validates the product entity
func (p *Product) Validate() error {
	var errs []error

	// Validate UUID
	if p.ID == uuid.Nil {
		errs = append(errs, errors.New("product ID cannot be empty"))
	}

	// Validate SKU
	if err := p.validateSKU(); err != nil {
		errs = append(errs, fmt.Errorf("invalid SKU: %w", err))
	}

	// Validate name
	if err := p.validateName(); err != nil {
		errs = append(errs, fmt.Errorf("invalid name: %w", err))
	}

	// Validate descriptions
	if err := p.validateDescriptions(); err != nil {
		errs = append(errs, fmt.Errorf("invalid description: %w", err))
	}

	// Validate category ID
	if p.CategoryID == uuid.Nil {
		errs = append(errs, errors.New("category ID cannot be empty"))
	}

	// Validate price and cost
	if err := p.validatePricing(); err != nil {
		errs = append(errs, fmt.Errorf("invalid pricing: %w", err))
	}

	// Validate dimensions and weight
	if err := p.validatePhysicalProperties(); err != nil {
		errs = append(errs, fmt.Errorf("invalid physical properties: %w", err))
	}

	// Validate barcode
	if err := p.validateBarcode(); err != nil {
		errs = append(errs, fmt.Errorf("invalid barcode: %w", err))
	}

	// Validate inventory
	if err := p.validateInventory(); err != nil {
		errs = append(errs, fmt.Errorf("invalid inventory: %w", err))
	}

	// Validate tax settings
	if err := p.validateTaxSettings(); err != nil {
		errs = append(errs, fmt.Errorf("invalid tax settings: %w", err))
	}

	// Validate digital product settings
	if err := p.validateDigitalSettings(); err != nil {
		errs = append(errs, fmt.Errorf("invalid digital settings: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// validateSKU validates the product SKU
func (p *Product) validateSKU() error {
	sku := strings.TrimSpace(p.SKU)
	if sku == "" {
		return errors.New("SKU cannot be empty")
	}

	if len(sku) > 100 {
		return errors.New("SKU cannot exceed 100 characters")
	}

	// SKU should be alphanumeric with hyphens and underscores
	skuRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !skuRegex.MatchString(sku) {
		return errors.New("SKU can only contain letters, numbers, hyphens, and underscores")
	}

	return nil
}

// validateName validates the product name
func (p *Product) validateName() error {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return errors.New("product name cannot be empty")
	}

	if len(name) > 300 {
		return errors.New("product name cannot exceed 300 characters")
	}

	return nil
}

// validateDescriptions validates product descriptions
func (p *Product) validateDescriptions() error {
	// Short description validation
	if p.ShortDescription != "" && len(p.ShortDescription) > 500 {
		return errors.New("short description cannot exceed 500 characters")
	}

	// Long description validation
	if p.Description != "" && len(p.Description) > 2000 {
		return errors.New("description cannot exceed 2000 characters")
	}

	return nil
}

// validatePricing validates price and cost
func (p *Product) validatePricing() error {
	// Price validation
	if p.Price.LessThanOrEqual(decimal.Zero) {
		return errors.New("price must be greater than 0")
	}

	if p.Price.GreaterThan(decimal.NewFromFloat(999999.99)) {
		return errors.New("price cannot exceed 999999.99")
	}

	// Cost validation
	if p.Cost.LessThan(decimal.Zero) {
		return errors.New("cost cannot be negative")
	}

	if p.Cost.GreaterThan(decimal.NewFromFloat(999999.99)) {
		return errors.New("cost cannot exceed 999999.99")
	}

	// Cost should not be higher than price (business rule)
	if p.Cost.GreaterThan(p.Price) {
		return errors.New("cost cannot be higher than price")
	}

	return nil
}

// validatePhysicalProperties validates dimensions and weight
func (p *Product) validatePhysicalProperties() error {
	// Weight validation
	if p.Weight < 0 {
		return errors.New("weight cannot be negative")
	}

	if p.Weight > 999999.99 {
		return errors.New("weight cannot exceed 999999.99")
	}

	// Dimensions validation
	if p.Length < 0 || p.Width < 0 || p.Height < 0 {
		return errors.New("dimensions cannot be negative")
	}

	if p.Length > 99999 || p.Width > 99999 || p.Height > 99999 {
		return errors.New("individual dimensions cannot exceed 99999")
	}

	// Volume validation (if set)
	if p.Volume < 0 {
		return errors.New("volume cannot be negative")
	}

	// Custom dimensions string validation
	if p.Dimensions != "" {
		// Format should be like "L x W x H" or "LxWxH"
		dimRegex := regexp.MustCompile(`^\d+(\.\d+)?\s*[xX]\s*\d+(\.\d+)?\s*[xX]\s*\d+(\.\d+)?\s*$`)
		if !dimRegex.MatchString(p.Dimensions) {
			return errors.New("dimensions must be in format 'L x W x H' or 'LxWxH'")
		}
	}

	return nil
}

// validateBarcode validates the product barcode
func (p *Product) validateBarcode() error {
	if p.Barcode != "" {
		barcode := strings.TrimSpace(p.Barcode)

		if len(barcode) > 50 {
			return errors.New("barcode cannot exceed 50 characters")
		}

		// Allow common barcode formats (numeric, alphanumeric)
		barcodeRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
		if !barcodeRegex.MatchString(barcode) {
			return errors.New("barcode can only contain letters, numbers, hyphens, and underscores")
		}
	}

	return nil
}

// validateInventory validates inventory settings
func (p *Product) validateInventory() error {
	// Stock quantity validation
	if p.StockQuantity < 0 {
		return errors.New("stock quantity cannot be negative")
	}

	if p.StockQuantity > 999999 {
		return errors.New("stock quantity cannot exceed 999999")
	}

	// Min stock level validation
	if p.MinStockLevel < 0 {
		return errors.New("minimum stock level cannot be negative")
	}

	if p.MinStockLevel > 999999 {
		return errors.New("minimum stock level cannot exceed 999999")
	}

	// Max stock level validation
	if p.MaxStockLevel < 0 {
		return errors.New("maximum stock level cannot be negative")
	}

	if p.MaxStockLevel > 999999 {
		return errors.New("maximum stock level cannot exceed 999999")
	}

	// Max stock should be >= min stock (if both are set)
	if p.MaxStockLevel > 0 && p.MinStockLevel > 0 && p.MaxStockLevel < p.MinStockLevel {
		return errors.New("maximum stock level cannot be less than minimum stock level")
	}

	return nil
}

// validateTaxSettings validates tax-related settings
func (p *Product) validateTaxSettings() error {
	if p.Taxable {
		if p.TaxRate.LessThan(decimal.Zero) {
			return errors.New("tax rate cannot be negative")
		}

		if p.TaxRate.GreaterThan(decimal.NewFromFloat(100)) {
			return errors.New("tax rate cannot exceed 100%")
		}
	} else {
		// If not taxable, tax rate should be 0
		if !p.TaxRate.Equal(decimal.Zero) {
			return errors.New("tax rate must be 0 for non-taxable products")
		}
	}

	return nil
}

// validateDigitalSettings validates digital product settings
func (p *Product) validateDigitalSettings() error {
	if p.IsDigital {
		// Digital products shouldn't require shipping
		if p.RequiresShipping {
			return errors.New("digital products cannot require shipping")
		}

		// Digital products should have download URL
		if p.DownloadURL == "" {
			return errors.New("digital products must have a download URL")
		}

		if err := p.validateDownloadURL(); err != nil {
			return fmt.Errorf("invalid download URL: %w", err)
		}

		// Max downloads validation
		if p.MaxDownloads < 0 {
			return errors.New("maximum downloads cannot be negative")
		}

		if p.MaxDownloads > 9999 {
			return errors.New("maximum downloads cannot exceed 9999")
		}

		// Expiry days validation
		if p.ExpiryDays < 0 {
			return errors.New("expiry days cannot be negative")
		}

		if p.ExpiryDays > 3650 {
			return errors.New("expiry days cannot exceed 3650 (10 years)")
		}
	} else {
		// Physical products shouldn't have digital settings
		if p.DownloadURL != "" {
			return errors.New("physical products cannot have download URLs")
		}

		if p.MaxDownloads > 0 {
			return errors.New("physical products cannot have maximum downloads limit")
		}

		if p.ExpiryDays > 0 {
			return errors.New("physical products cannot have expiry days")
		}
	}

	return nil
}

// validateDownloadURL validates the download URL
func (p *Product) validateDownloadURL() error {
	url := strings.TrimSpace(p.DownloadURL)
	if url == "" {
		return nil // URL is optional for physical products
	}

	if len(url) > 1000 {
		return errors.New("download URL cannot exceed 1000 characters")
	}

	// Basic URL regex pattern
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(url) {
		return errors.New("invalid download URL format")
	}

	return nil
}

// Business Logic Methods

// IsActiveProduct returns true if the product is active
func (p *Product) IsActiveProduct() bool {
	return p.IsActive
}

// IsInStock returns true if the product is in stock
func (p *Product) IsInStock() bool {
	if !p.TrackInventory {
		return true // Always in stock if not tracking inventory
	}
	return p.StockQuantity > 0 || p.AllowBackorder
}

// IsLowStock returns true if the product stock is below minimum level
func (p *Product) IsLowStock() bool {
	if !p.TrackInventory || p.MinStockLevel <= 0 {
		return false
	}
	return p.StockQuantity <= p.MinStockLevel
}

// CanFulfillOrder checks if the product can fulfill a given quantity
func (p *Product) CanFulfillOrder(quantity int) error {
	if quantity <= 0 {
		return errors.New("order quantity must be positive")
	}

	if !p.IsActive {
		return errors.New("product is not active")
	}

	if p.TrackInventory && !p.AllowBackorder && p.StockQuantity < quantity {
		return fmt.Errorf("insufficient stock: %d available, %d requested", p.StockQuantity, quantity)
	}

	return nil
}

// CalculateProfit calculates the profit margin for the product
func (p *Product) CalculateProfit() decimal.Decimal {
	return p.Price.Sub(p.Cost)
}

// CalculateProfitMargin calculates the profit margin as a percentage
func (p *Product) CalculateProfitMargin() decimal.Decimal {
	if p.Price.Equal(decimal.Zero) {
		return decimal.Zero
	}
	profit := p.CalculateProfit()
	return profit.Div(p.Price).Mul(decimal.NewFromInt(100))
}

// CalculateTax calculates the tax amount for the product
func (p *Product) CalculateTax() decimal.Decimal {
	if !p.Taxable {
		return decimal.Zero
	}
	return p.Price.Mul(p.TaxRate).Div(decimal.NewFromInt(100))
}

// CalculateTotalPrice calculates the total price including tax
func (p *Product) CalculateTotalPrice() decimal.Decimal {
	tax := p.CalculateTax()
	return p.Price.Add(tax)
}

// UpdateStock updates the stock quantity
func (p *Product) UpdateStock(newQuantity int) error {
	if newQuantity < 0 {
		return errors.New("stock quantity cannot be negative")
	}

	if newQuantity > 999999 {
		return errors.New("stock quantity cannot exceed 999999")
	}

	p.StockQuantity = newQuantity
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// AdjustStock adjusts the stock quantity by a given amount
func (p *Product) AdjustStock(adjustment int) error {
	newQuantity := p.StockQuantity + adjustment
	return p.UpdateStock(newQuantity)
}

// UpdatePrice updates the product price
func (p *Product) UpdatePrice(newPrice decimal.Decimal) error {
	if newPrice.LessThanOrEqual(decimal.Zero) {
		return errors.New("price must be greater than 0")
	}

	if newPrice.GreaterThan(decimal.NewFromFloat(999999.99)) {
		return errors.New("price cannot exceed 999999.99")
	}

	p.Price = newPrice
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateCost updates the product cost
func (p *Product) UpdateCost(newCost decimal.Decimal) error {
	if newCost.LessThan(decimal.Zero) {
		return errors.New("cost cannot be negative")
	}

	if newCost.GreaterThan(decimal.NewFromFloat(999999.99)) {
		return errors.New("cost cannot exceed 999999.99")
	}

	// Ensure cost is not higher than price
	if newCost.GreaterThan(p.Price) {
		return errors.New("cost cannot be higher than price")
	}

	p.Cost = newCost
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Activate activates the product
func (p *Product) Activate() {
	p.IsActive = true
	p.UpdatedAt = time.Now().UTC()
}

// Deactivate deactivates the product
func (p *Product) Deactivate() {
	p.IsActive = false
	p.UpdatedAt = time.Now().UTC()
}

// SetFeatured sets or unsets the product as featured
func (p *Product) SetFeatured(featured bool) {
	p.IsFeatured = featured
	p.UpdatedAt = time.Now().UTC()
}

// UpdateCategory moves the product to a different category
func (p *Product) UpdateCategory(categoryID uuid.UUID) error {
	if categoryID == uuid.Nil {
		return errors.New("category ID cannot be empty")
	}

	p.CategoryID = categoryID
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateDetails updates product basic information
func (p *Product) UpdateDetails(name, description, shortDescription string) error {
	// Create temporary product for validation
	tempP := &Product{
		Name:            name,
		Description:     description,
		ShortDescription: shortDescription,
	}

	if err := tempP.validateName(); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	if err := tempP.validateDescriptions(); err != nil {
		return fmt.Errorf("invalid descriptions: %w", err)
	}

	p.Name = name
	p.Description = description
	p.ShortDescription = shortDescription
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// ToSafeProduct returns a product object without sensitive information
func (p *Product) ToSafeProduct() *Product {
	return &Product{
		ID:                p.ID,
		SKU:               p.SKU,
		Name:              p.Name,
		Description:       p.Description,
		ShortDescription:  p.ShortDescription,
		CategoryID:        p.CategoryID,
		Price:             p.Price,
		Weight:            p.Weight,
		Dimensions:        p.Dimensions,
		TrackInventory:    p.TrackInventory,
		StockQuantity:     p.StockQuantity,
		AllowBackorder:    p.AllowBackorder,
		RequiresShipping:  p.RequiresShipping,
		Taxable:           p.Taxable,
		TaxRate:           p.TaxRate,
		IsActive:          p.IsActive,
		IsFeatured:        p.IsFeatured,
		IsDigital:         p.IsDigital,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}

// Additional supporting entities for inventory management

// Warehouse represents a storage location
type Warehouse struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Code        string    `json:"code" db:"code"`
	Address     string    `json:"address" db:"address"`
	City        string    `json:"city" db:"city"`
	State       string    `json:"state" db:"state"`
	Country     string    `json:"country" db:"country"`
	PostalCode  string    `json:"postal_code" db:"postal_code"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Inventory represents inventory levels for a product in a warehouse
type Inventory struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	ProductID         uuid.UUID  `json:"product_id" db:"product_id"`
	WarehouseID       uuid.UUID  `json:"warehouse_id" db:"warehouse_id"`
	QuantityAvailable int        `json:"quantity_available" db:"quantity_available"`
	QuantityReserved  int        `json:"quantity_reserved" db:"quantity_reserved"`
	ReorderLevel      int        `json:"reorder_level" db:"reorder_level"`
	MaxStock          *int       `json:"max_stock,omitempty" db:"max_stock"`
	LastUpdatedAt     time.Time  `json:"last_updated_at" db:"last_updated_at"`
	UpdatedBy         uuid.UUID  `json:"updated_by" db:"updated_by"`
}

// InventoryTransaction represents a movement of inventory
type InventoryTransaction struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	ProductID     uuid.UUID  `json:"product_id" db:"product_id"`
	WarehouseID   uuid.UUID  `json:"warehouse_id" db:"warehouse_id"`
	TransactionType string   `json:"transaction_type" db:"transaction_type"` // 'IN', 'OUT', 'ADJUST', 'TRANSFER'
	Quantity      int        `json:"quantity" db:"quantity"`
	ReferenceID   *uuid.UUID `json:"reference_id,omitempty" db:"reference_id"`
	Reason        string     `json:"reason,omitempty" db:"reason"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	CreatedBy     uuid.UUID  `json:"created_by" db:"created_by"`
}