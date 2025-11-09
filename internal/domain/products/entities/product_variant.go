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

// ProductVariant represents a variant of a product (e.g., different sizes, colors)
type ProductVariant struct {
	ID                uuid.UUID       `json:"id" db:"id"`
	ProductID         uuid.UUID       `json:"product_id" db:"product_id"`
	SKU               string          `json:"sku" db:"sku"`
	Name              string          `json:"name" db:"name"`
	Price             decimal.Decimal `json:"price" db:"price"`
	Cost              decimal.Decimal `json:"cost" db:"cost"`
	Weight            float64         `json:"weight" db:"weight"`
	Dimensions        string          `json:"dimensions" db:"dimensions"`
	Length            float64         `json:"length" db:"length"`
	Width             float64         `json:"width" db:"width"`
	Height            float64         `json:"height" db:"height"`
	Volume            float64         `json:"volume" db:"volume"`
	Barcode           string          `json:"barcode" db:"barcode"`
	ImageURL          string          `json:"image_url" db:"image_url"`
	TrackInventory    bool            `json:"track_inventory" db:"track_inventory"`
	StockQuantity     int             `json:"stock_quantity" db:"stock_quantity"`
	MinStockLevel     int             `json:"min_stock_level" db:"min_stock_level"`
	MaxStockLevel     int             `json:"max_stock_level" db:"max_stock_level"`
	AllowBackorder    bool            `json:"allow_backorder" db:"allow_backorder"`
	RequiresShipping  bool            `json:"requires_shipping" db:"requires_shipping"`
	Taxable           bool            `json:"taxable" db:"taxable"`
	TaxRate           decimal.Decimal `json:"tax_rate" db:"tax_rate"`
	IsActive          bool            `json:"is_active" db:"is_active"`
	IsDigital         bool            `json:"is_digital" db:"is_digital"`
	DownloadURL       string          `json:"download_url" db:"download_url"`
	MaxDownloads      int             `json:"max_downloads" db:"max_downloads"`
	ExpiryDays        int             `json:"expiry_days" db:"expiry_days"`
	SortOrder         int             `json:"sort_order" db:"sort_order"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`
}

// VariantAttribute represents an attribute for a product variant
type VariantAttribute struct {
	ID        uuid.UUID `json:"id" db:"id"`
	VariantID uuid.UUID `json:"variant_id" db:"variant_id"`
	Name      string    `json:"name" db:"name"`
	Value     string    `json:"value" db:"value"`
	Type      string    `json:"type" db:"type"` // 'color', 'size', 'material', etc.
	SortOrder int       `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// VariantImage represents an image for a product variant
type VariantImage struct {
	ID        uuid.UUID `json:"id" db:"id"`
	VariantID uuid.UUID `json:"variant_id" db:"variant_id"`
	ImageURL  string    `json:"image_url" db:"image_url"`
	URL       string    `json:"url" db:"url"` // Alias for ImageURL for compatibility
	Alt       string    `json:"alt" db:"alt"` // Alias for AltText for compatibility
	AltText   string    `json:"alt_text" db:"alt_text"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	IsMain    bool      `json:"is_main" db:"is_main"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Validate validates the product variant entity
func (pv *ProductVariant) Validate() error {
	var errs []error

	// Validate UUID
	if pv.ID == uuid.Nil {
		errs = append(errs, errors.New("variant ID cannot be empty"))
	}

	// Validate Product ID
	if pv.ProductID == uuid.Nil {
		errs = append(errs, errors.New("product ID cannot be empty"))
	}

	// Validate SKU
	if err := pv.validateSKU(); err != nil {
		errs = append(errs, fmt.Errorf("invalid SKU: %w", err))
	}

	// Validate name
	if err := pv.validateName(); err != nil {
		errs = append(errs, fmt.Errorf("invalid name: %w", err))
	}

	// Validate price and cost
	if err := pv.validatePricing(); err != nil {
		errs = append(errs, fmt.Errorf("invalid pricing: %w", err))
	}

	// Validate dimensions and weight
	if err := pv.validatePhysicalProperties(); err != nil {
		errs = append(errs, fmt.Errorf("invalid physical properties: %w", err))
	}

	// Validate barcode
	if err := pv.validateBarcode(); err != nil {
		errs = append(errs, fmt.Errorf("invalid barcode: %w", err))
	}

	// Validate image URL
	if err := pv.validateImageURL(); err != nil {
		errs = append(errs, fmt.Errorf("invalid image URL: %w", err))
	}

	// Validate inventory
	if err := pv.validateInventory(); err != nil {
		errs = append(errs, fmt.Errorf("invalid inventory: %w", err))
	}

	// Validate tax settings
	if err := pv.validateTaxSettings(); err != nil {
		errs = append(errs, fmt.Errorf("invalid tax settings: %w", err))
	}

	// Validate digital product settings
	if err := pv.validateDigitalSettings(); err != nil {
		errs = append(errs, fmt.Errorf("invalid digital settings: %w", err))
	}

	// Validate sort order
	if pv.SortOrder < 0 {
		errs = append(errs, errors.New("sort order cannot be negative"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// validateSKU validates the variant SKU
func (pv *ProductVariant) validateSKU() error {
	sku := strings.TrimSpace(pv.SKU)
	if sku == "" {
		return errors.New("variant SKU cannot be empty")
	}

	if len(sku) > 100 {
		return errors.New("variant SKU cannot exceed 100 characters")
	}

	// SKU should be alphanumeric with hyphens and underscores
	skuRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !skuRegex.MatchString(sku) {
		return errors.New("variant SKU can only contain letters, numbers, hyphens, and underscores")
	}

	return nil
}

// validateName validates the variant name
func (pv *ProductVariant) validateName() error {
	name := strings.TrimSpace(pv.Name)
	if name == "" {
		return errors.New("variant name cannot be empty")
	}

	if len(name) > 300 {
		return errors.New("variant name cannot exceed 300 characters")
	}

	return nil
}

// validatePricing validates variant price and cost
func (pv *ProductVariant) validatePricing() error {
	// Price validation
	if pv.Price.LessThanOrEqual(decimal.Zero) {
		return errors.New("variant price must be greater than 0")
	}

	if pv.Price.GreaterThan(decimal.NewFromFloat(999999.99)) {
		return errors.New("variant price cannot exceed 999999.99")
	}

	// Cost validation
	if pv.Cost.LessThan(decimal.Zero) {
		return errors.New("variant cost cannot be negative")
	}

	if pv.Cost.GreaterThan(decimal.NewFromFloat(999999.99)) {
		return errors.New("variant cost cannot exceed 999999.99")
	}

	// Cost should not be higher than price (business rule)
	if pv.Cost.GreaterThan(pv.Price) {
		return errors.New("variant cost cannot be higher than price")
	}

	return nil
}

// validatePhysicalProperties validates variant dimensions and weight
func (pv *ProductVariant) validatePhysicalProperties() error {
	// Weight validation
	if pv.Weight < 0 {
		return errors.New("variant weight cannot be negative")
	}

	if pv.Weight > 999999.99 {
		return errors.New("variant weight cannot exceed 999999.99")
	}

	// Dimensions validation
	if pv.Length < 0 || pv.Width < 0 || pv.Height < 0 {
		return errors.New("variant dimensions cannot be negative")
	}

	if pv.Length > 99999 || pv.Width > 99999 || pv.Height > 99999 {
		return errors.New("individual variant dimensions cannot exceed 99999")
	}

	// Volume validation (if set)
	if pv.Volume < 0 {
		return errors.New("variant volume cannot be negative")
	}

	// Custom dimensions string validation
	if pv.Dimensions != "" {
		// Format should be like "L x W x H" or "LxWxH"
		dimRegex := regexp.MustCompile(`^\d+(\.\d+)?\s*[xX]\s*\d+(\.\d+)?\s*[xX]\s*\d+(\.\d+)?\s*$`)
		if !dimRegex.MatchString(pv.Dimensions) {
			return errors.New("variant dimensions must be in format 'L x W x H' or 'LxWxH'")
		}
	}

	return nil
}

// validateBarcode validates the variant barcode
func (pv *ProductVariant) validateBarcode() error {
	if pv.Barcode != "" {
		barcode := strings.TrimSpace(pv.Barcode)

		if len(barcode) > 50 {
			return errors.New("variant barcode cannot exceed 50 characters")
		}

		// Allow common barcode formats (numeric, alphanumeric)
		barcodeRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
		if !barcodeRegex.MatchString(barcode) {
			return errors.New("variant barcode can only contain letters, numbers, hyphens, and underscores")
		}
	}

	return nil
}

// validateImageURL validates the variant image URL
func (pv *ProductVariant) validateImageURL() error {
	if pv.ImageURL != "" {
		url := strings.TrimSpace(pv.ImageURL)

		if len(url) > 1000 {
			return errors.New("variant image URL cannot exceed 1000 characters")
		}

		// Basic URL regex pattern
		urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
		if !urlRegex.MatchString(url) {
			return errors.New("invalid variant image URL format")
		}

		// Check for common image file extensions
		imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg"}
		hasValidExtension := false
		for _, ext := range imageExtensions {
			if strings.HasSuffix(strings.ToLower(url), ext) {
				hasValidExtension = true
				break
			}
		}

		if !hasValidExtension {
			return errors.New("variant image URL must end with a valid image extension (.jpg, .jpeg, .png, .gif, .webp, .svg)")
		}
	}

	return nil
}

// validateInventory validates variant inventory settings
func (pv *ProductVariant) validateInventory() error {
	// Stock quantity validation
	if pv.StockQuantity < 0 {
		return errors.New("variant stock quantity cannot be negative")
	}

	if pv.StockQuantity > 999999 {
		return errors.New("variant stock quantity cannot exceed 999999")
	}

	// Min stock level validation
	if pv.MinStockLevel < 0 {
		return errors.New("variant minimum stock level cannot be negative")
	}

	if pv.MinStockLevel > 999999 {
		return errors.New("variant minimum stock level cannot exceed 999999")
	}

	// Max stock level validation
	if pv.MaxStockLevel < 0 {
		return errors.New("variant maximum stock level cannot be negative")
	}

	if pv.MaxStockLevel > 999999 {
		return errors.New("variant maximum stock level cannot exceed 999999")
	}

	// Max stock should be >= min stock (if both are set)
	if pv.MaxStockLevel > 0 && pv.MinStockLevel > 0 && pv.MaxStockLevel < pv.MinStockLevel {
		return errors.New("variant maximum stock level cannot be less than minimum stock level")
	}

	return nil
}

// validateTaxSettings validates variant tax-related settings
func (pv *ProductVariant) validateTaxSettings() error {
	if pv.Taxable {
		if pv.TaxRate.LessThan(decimal.Zero) {
			return errors.New("variant tax rate cannot be negative")
		}

		if pv.TaxRate.GreaterThan(decimal.NewFromFloat(100)) {
			return errors.New("variant tax rate cannot exceed 100%")
		}
	} else {
		// If not taxable, tax rate should be 0
		if !pv.TaxRate.Equal(decimal.Zero) {
			return errors.New("variant tax rate must be 0 for non-taxable variants")
		}
	}

	return nil
}

// validateDigitalSettings validates variant digital product settings
func (pv *ProductVariant) validateDigitalSettings() error {
	if pv.IsDigital {
		// Digital variants shouldn't require shipping
		if pv.RequiresShipping {
			return errors.New("digital variants cannot require shipping")
		}

		// Digital variants should have download URL
		if pv.DownloadURL == "" {
			return errors.New("digital variants must have a download URL")
		}

		if err := pv.validateDownloadURL(); err != nil {
			return fmt.Errorf("invalid variant download URL: %w", err)
		}

		// Max downloads validation
		if pv.MaxDownloads < 0 {
			return errors.New("variant maximum downloads cannot be negative")
		}

		if pv.MaxDownloads > 9999 {
			return errors.New("variant maximum downloads cannot exceed 9999")
		}

		// Expiry days validation
		if pv.ExpiryDays < 0 {
			return errors.New("variant expiry days cannot be negative")
		}

		if pv.ExpiryDays > 3650 {
			return errors.New("variant expiry days cannot exceed 3650 (10 years)")
		}
	} else {
		// Physical variants shouldn't have digital settings
		if pv.DownloadURL != "" {
			return errors.New("physical variants cannot have download URLs")
		}

		if pv.MaxDownloads > 0 {
			return errors.New("physical variants cannot have maximum downloads limit")
		}

		if pv.ExpiryDays > 0 {
			return errors.New("physical variants cannot have expiry days")
		}
	}

	return nil
}

// validateDownloadURL validates the variant download URL
func (pv *ProductVariant) validateDownloadURL() error {
	url := strings.TrimSpace(pv.DownloadURL)
	if url == "" {
		return nil // URL is optional for physical variants
	}

	if len(url) > 1000 {
		return errors.New("variant download URL cannot exceed 1000 characters")
	}

	// Basic URL regex pattern
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(url) {
		return errors.New("invalid variant download URL format")
	}

	return nil
}

// Business Logic Methods

// IsActiveVariant returns true if the variant is active
func (pv *ProductVariant) IsActiveVariant() bool {
	return pv.IsActive
}

// IsInStock returns true if the variant is in stock
func (pv *ProductVariant) IsInStock() bool {
	if !pv.TrackInventory {
		return true // Always in stock if not tracking inventory
	}
	return pv.StockQuantity > 0 || pv.AllowBackorder
}

// IsLowStock returns true if the variant stock is below minimum level
func (pv *ProductVariant) IsLowStock() bool {
	if !pv.TrackInventory || pv.MinStockLevel <= 0 {
		return false
	}
	return pv.StockQuantity <= pv.MinStockLevel
}

// CanFulfillOrder checks if the variant can fulfill a given quantity
func (pv *ProductVariant) CanFulfillOrder(quantity int) error {
	if quantity <= 0 {
		return errors.New("order quantity must be positive")
	}

	if !pv.IsActive {
		return errors.New("variant is not active")
	}

	if pv.TrackInventory && !pv.AllowBackorder && pv.StockQuantity < quantity {
		return fmt.Errorf("insufficient variant stock: %d available, %d requested", pv.StockQuantity, quantity)
	}

	return nil
}

// CalculateProfit calculates the profit margin for the variant
func (pv *ProductVariant) CalculateProfit() decimal.Decimal {
	return pv.Price.Sub(pv.Cost)
}

// CalculateProfitMargin calculates the profit margin as a percentage
func (pv *ProductVariant) CalculateProfitMargin() decimal.Decimal {
	if pv.Price.Equal(decimal.Zero) {
		return decimal.Zero
	}
	profit := pv.CalculateProfit()
	return profit.Div(pv.Price).Mul(decimal.NewFromInt(100))
}

// CalculateTax calculates the tax amount for the variant
func (pv *ProductVariant) CalculateTax() decimal.Decimal {
	if !pv.Taxable {
		return decimal.Zero
	}
	return pv.Price.Mul(pv.TaxRate).Div(decimal.NewFromInt(100))
}

// CalculateTotalPrice calculates the total price including tax
func (pv *ProductVariant) CalculateTotalPrice() decimal.Decimal {
	tax := pv.CalculateTax()
	return pv.Price.Add(tax)
}

// UpdateStock updates the variant stock quantity
func (pv *ProductVariant) UpdateStock(newQuantity int) error {
	if newQuantity < 0 {
		return errors.New("variant stock quantity cannot be negative")
	}

	if newQuantity > 999999 {
		return errors.New("variant stock quantity cannot exceed 999999")
	}

	pv.StockQuantity = newQuantity
	pv.UpdatedAt = time.Now().UTC()
	return nil
}

// AdjustStock adjusts the variant stock quantity by a given amount
func (pv *ProductVariant) AdjustStock(adjustment int) error {
	newQuantity := pv.StockQuantity + adjustment
	return pv.UpdateStock(newQuantity)
}

// UpdatePrice updates the variant price
func (pv *ProductVariant) UpdatePrice(newPrice decimal.Decimal) error {
	if newPrice.LessThanOrEqual(decimal.Zero) {
		return errors.New("variant price must be greater than 0")
	}

	if newPrice.GreaterThan(decimal.NewFromFloat(999999.99)) {
		return errors.New("variant price cannot exceed 999999.99")
	}

	pv.Price = newPrice
	pv.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateCost updates the variant cost
func (pv *ProductVariant) UpdateCost(newCost decimal.Decimal) error {
	if newCost.LessThan(decimal.Zero) {
		return errors.New("variant cost cannot be negative")
	}

	if newCost.GreaterThan(decimal.NewFromFloat(999999.99)) {
		return errors.New("variant cost cannot exceed 999999.99")
	}

	// Ensure cost is not higher than price
	if newCost.GreaterThan(pv.Price) {
		return errors.New("variant cost cannot be higher than price")
	}

	pv.Cost = newCost
	pv.UpdatedAt = time.Now().UTC()
	return nil
}

// Activate activates the variant
func (pv *ProductVariant) Activate() {
	pv.IsActive = true
	pv.UpdatedAt = time.Now().UTC()
}

// Deactivate deactivates the variant
func (pv *ProductVariant) Deactivate() {
	pv.IsActive = false
	pv.UpdatedAt = time.Now().UTC()
}

// UpdateSortOrder updates the variant sort order
func (pv *ProductVariant) UpdateSortOrder(newOrder int) error {
	if newOrder < 0 {
		return errors.New("variant sort order cannot be negative")
	}

	pv.SortOrder = newOrder
	pv.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateImage updates the variant main image URL
func (pv *ProductVariant) UpdateImage(imageURL string) error {
	if imageURL != "" {
		// Validate the new image URL
		tempPV := &ProductVariant{ImageURL: imageURL}
		if err := tempPV.validateImageURL(); err != nil {
			return fmt.Errorf("invalid image URL: %w", err)
		}
	}
	pv.ImageURL = imageURL
	pv.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateDetails updates variant basic information
func (pv *ProductVariant) UpdateDetails(name string) error {
	// Create temporary variant for validation
	tempPV := &ProductVariant{Name: name}

	if err := tempPV.validateName(); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	pv.Name = name
	pv.UpdatedAt = time.Now().UTC()
	return nil
}

// ToSafeVariant returns a variant object without sensitive information
func (pv *ProductVariant) ToSafeVariant() *ProductVariant {
	return &ProductVariant{
		ID:               pv.ID,
		ProductID:        pv.ProductID,
		SKU:              pv.SKU,
		Name:             pv.Name,
		Price:            pv.Price,
		Weight:           pv.Weight,
		Dimensions:       pv.Dimensions,
		ImageURL:         pv.ImageURL,
		TrackInventory:   pv.TrackInventory,
		StockQuantity:    pv.StockQuantity,
		AllowBackorder:   pv.AllowBackorder,
		RequiresShipping: pv.RequiresShipping,
		Taxable:          pv.Taxable,
		TaxRate:          pv.TaxRate,
		IsActive:         pv.IsActive,
		IsDigital:        pv.IsDigital,
		SortOrder:        pv.SortOrder,
		CreatedAt:        pv.CreatedAt,
		UpdatedAt:        pv.UpdatedAt,
	}
}

// VariantAttribute Methods

// Validate validates the variant attribute entity
func (va *VariantAttribute) Validate() error {
	var errs []error

	// Validate UUIDs
	if va.ID == uuid.Nil {
		errs = append(errs, errors.New("attribute ID cannot be empty"))
	}

	if va.VariantID == uuid.Nil {
		errs = append(errs, errors.New("variant ID cannot be empty"))
	}

	// Validate name
	if strings.TrimSpace(va.Name) == "" {
		errs = append(errs, errors.New("attribute name cannot be empty"))
	} else if len(va.Name) > 100 {
		errs = append(errs, errors.New("attribute name cannot exceed 100 characters"))
	}

	// Validate value
	if strings.TrimSpace(va.Value) == "" {
		errs = append(errs, errors.New("attribute value cannot be empty"))
	} else if len(va.Value) > 200 {
		errs = append(errs, errors.New("attribute value cannot exceed 200 characters"))
	}

	// Validate type
	if strings.TrimSpace(va.Type) == "" {
		errs = append(errs, errors.New("attribute type cannot be empty"))
	} else if len(va.Type) > 50 {
		errs = append(errs, errors.New("attribute type cannot exceed 50 characters"))
	}

	// Validate sort order
	if va.SortOrder < 0 {
		errs = append(errs, errors.New("attribute sort order cannot be negative"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// VariantImage Methods

// Validate validates the variant image entity
func (vi *VariantImage) Validate() error {
	var errs []error

	// Validate UUIDs
	if vi.ID == uuid.Nil {
		errs = append(errs, errors.New("image ID cannot be empty"))
	}

	if vi.VariantID == uuid.Nil {
		errs = append(errs, errors.New("variant ID cannot be empty"))
	}

	// Validate image URL
	if strings.TrimSpace(vi.ImageURL) == "" {
		errs = append(errs, errors.New("image URL cannot be empty"))
	} else if len(vi.ImageURL) > 1000 {
		errs = append(errs, errors.New("image URL cannot exceed 1000 characters"))
	} else {
		// Basic URL validation
		urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
		if !urlRegex.MatchString(vi.ImageURL) {
			errs = append(errs, errors.New("invalid image URL format"))
		}
	}

	// Validate alt text
	if len(vi.AltText) > 200 {
		errs = append(errs, errors.New("alt text cannot exceed 200 characters"))
	}

	// Validate sort order
	if vi.SortOrder < 0 {
		errs = append(errs, errors.New("image sort order cannot be negative"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// SetAsMain sets this image as the main image for the variant
func (vi *VariantImage) SetAsMain() {
	vi.IsMain = true
}