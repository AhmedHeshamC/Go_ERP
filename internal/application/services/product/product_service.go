package product

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"erpgo/internal/domain/products/entities"
	"erpgo/internal/domain/products/repositories"
)

// Service defines the business logic interface for product management
type Service interface {
	// Product management
	CreateProduct(ctx context.Context, req *CreateProductRequest) (*entities.Product, error)
	GetProduct(ctx context.Context, id string) (*entities.Product, error)
	GetProductBySKU(ctx context.Context, sku string) (*entities.Product, error)
	UpdateProduct(ctx context.Context, id string, req *UpdateProductRequest) (*entities.Product, error)
	DeleteProduct(ctx context.Context, id string) error
	ListProducts(ctx context.Context, req *ListProductsRequest) (*ListProductsResponse, error)
	SearchProducts(ctx context.Context, req *SearchProductsRequest) (*SearchProductsResponse, error)

	// Product operations
	ActivateProduct(ctx context.Context, id string) error
	DeactivateProduct(ctx context.Context, id string) error
	SetFeaturedProduct(ctx context.Context, id string, featured bool) error
	UpdateProductPrice(ctx context.Context, id string, req *UpdatePriceRequest) error
	UpdateProductStock(ctx context.Context, id string, req *UpdateStockRequest) error
	AdjustProductStock(ctx context.Context, id string, req *AdjustStockRequest) error

	// Category management
	CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*entities.ProductCategory, error)
	GetCategory(ctx context.Context, id string) (*entities.ProductCategory, error)
	GetCategoryByPath(ctx context.Context, path string) (*entities.ProductCategory, error)
	UpdateCategory(ctx context.Context, id string, req *UpdateCategoryRequest) (*entities.ProductCategory, error)
	DeleteCategory(ctx context.Context, id string) error
	ListCategories(ctx context.Context, req *ListCategoriesRequest) (*ListCategoriesResponse, error)
	GetCategoryTree(ctx context.Context) (*CategoryTreeResponse, error)
	GetCategoryDescendants(ctx context.Context, id string) ([]*entities.ProductCategory, error)
	GetCategoryAncestors(ctx context.Context, id string) ([]*entities.ProductCategory, error)
	MoveCategory(ctx context.Context, id string, newParentID *string) error

	// Product variant management
	CreateProductVariant(ctx context.Context, req *CreateProductVariantRequest) (*entities.ProductVariant, error)
	GetProductVariant(ctx context.Context, id string) (*entities.ProductVariant, error)
	GetProductVariantBySKU(ctx context.Context, sku string) (*entities.ProductVariant, error)
	UpdateProductVariant(ctx context.Context, id string, req *UpdateProductVariantRequest) (*entities.ProductVariant, error)
	DeleteProductVariant(ctx context.Context, id string) error
	ListProductVariants(ctx context.Context, productID string, req *ListProductVariantsRequest) (*ListProductVariantsResponse, error)

	// Inventory and stock management
	GetLowStockProducts(ctx context.Context, threshold int) ([]*entities.Product, error)
	GetProductStockLevel(ctx context.Context, id string) (*StockLevelResponse, error)
	CheckProductAvailability(ctx context.Context, id string, quantity int) (*AvailabilityResponse, error)
	GetProductStats(ctx context.Context, req *GetProductStatsRequest) (*repositories.ProductStats, error)
}

// Request/Response DTOs

type CreateProductRequest struct {
	SKU              string          `json:"sku,omitempty" validate:"max=100"`
	Name             string          `json:"name" validate:"required,max=300"`
	Description      string          `json:"description,omitempty" validate:"max=2000"`
	ShortDescription string          `json:"short_description,omitempty" validate:"max=500"`
	CategoryID       string          `json:"category_id" validate:"required,uuid"`
	Price            decimal.Decimal `json:"price" validate:"required,gt=0"`
	Cost             decimal.Decimal `json:"cost,omitempty" validate:"gte=0"`
	Weight           float64         `json:"weight,omitempty" validate:"gte=0"`
	Length           float64         `json:"length,omitempty" validate:"gte=0"`
	Width            float64         `json:"width,omitempty" validate:"gte=0"`
	Height           float64         `json:"height,omitempty" validate:"gte=0"`
	Barcode          string          `json:"barcode,omitempty" validate:"max=50"`
	TrackInventory   bool            `json:"track_inventory"`
	StockQuantity    int             `json:"stock_quantity,omitempty" validate:"gte=0"`
	MinStockLevel    int             `json:"min_stock_level,omitempty" validate:"gte=0"`
	MaxStockLevel    int             `json:"max_stock_level,omitempty" validate:"gte=0"`
	AllowBackorder   bool            `json:"allow_backorder"`
	RequiresShipping bool            `json:"requires_shipping"`
	Taxable          bool            `json:"taxable"`
	TaxRate          decimal.Decimal `json:"tax_rate,omitempty" validate:"gte=0,lte=100"`
	IsFeatured       bool            `json:"is_featured"`
	IsDigital        bool            `json:"is_digital"`
	DownloadURL      string          `json:"download_url,omitempty" validate:"omitempty,url,max=1000"`
	MaxDownloads     int             `json:"max_downloads,omitempty" validate:"gte=0,max=9999"`
	ExpiryDays       int             `json:"expiry_days,omitempty" validate:"gte=0,max=3650"`
}

type UpdateProductRequest struct {
	Name             *string          `json:"name,omitempty" validate:"omitempty,max=300"`
	Description      *string          `json:"description,omitempty" validate:"omitempty,max=2000"`
	ShortDescription *string          `json:"short_description,omitempty" validate:"omitempty,max=500"`
	CategoryID       *string          `json:"category_id,omitempty" validate:"omitempty,uuid"`
	Price            *decimal.Decimal `json:"price,omitempty" validate:"omitempty,gt=0"`
	Cost             *decimal.Decimal `json:"cost,omitempty" validate:"omitempty,gte=0"`
	Weight           *float64         `json:"weight,omitempty" validate:"omitempty,gte=0"`
	Length           *float64         `json:"length,omitempty" validate:"omitempty,gte=0"`
	Width            *float64         `json:"width,omitempty" validate:"omitempty,gte=0"`
	Height           *float64         `json:"height,omitempty" validate:"omitempty,gte=0"`
	Barcode          *string          `json:"barcode,omitempty" validate:"omitempty,max=50"`
	TrackInventory   *bool            `json:"track_inventory,omitempty"`
	MinStockLevel    *int             `json:"min_stock_level,omitempty" validate:"omitempty,gte=0"`
	MaxStockLevel    *int             `json:"max_stock_level,omitempty" validate:"omitempty,gte=0"`
	AllowBackorder   *bool            `json:"allow_backorder,omitempty"`
	RequiresShipping *bool            `json:"requires_shipping,omitempty"`
	Taxable          *bool            `json:"taxable,omitempty"`
	TaxRate          *decimal.Decimal `json:"tax_rate,omitempty" validate:"omitempty,gte=0,lte=100"`
	IsFeatured       *bool            `json:"is_featured,omitempty"`
	IsDigital        *bool            `json:"is_digital,omitempty"`
	DownloadURL      *string          `json:"download_url,omitempty" validate:"omitempty,url,max=1000"`
	MaxDownloads     *int             `json:"max_downloads,omitempty" validate:"omitempty,gte=0,max=9999"`
	ExpiryDays       *int             `json:"expiry_days,omitempty" validate:"omitempty,gte=0,max=3650"`
}

type ListProductsRequest struct {
	Search         string           `json:"search,omitempty"`
	CategoryID     string           `json:"category_id,omitempty"`
	CategoryIDs    []string         `json:"category_ids,omitempty"`
	SKU            string           `json:"sku,omitempty"`
	MinPrice       *decimal.Decimal `json:"min_price,omitempty"`
	MaxPrice       *decimal.Decimal `json:"max_price,omitempty"`
	IsActive       *bool            `json:"is_active,omitempty"`
	IsFeatured     *bool            `json:"is_featured,omitempty"`
	IsDigital      *bool            `json:"is_digital,omitempty"`
	TrackInventory *bool            `json:"track_inventory,omitempty"`
	InStock        *bool            `json:"in_stock,omitempty"`
	LowStock       *bool            `json:"low_stock,omitempty"`
	CreatedAfter   *time.Time       `json:"created_after,omitempty"`
	CreatedBefore  *time.Time       `json:"created_before,omitempty"`
	Page           int              `json:"page,omitempty" validate:"min=1"`
	Limit          int              `json:"limit,omitempty" validate:"min=1,max=100"`
	SortBy         string           `json:"sort_by,omitempty" validate:"omitempty,oneof=name price created_at updated_at sku"`
	SortOrder      string           `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

type ListProductsResponse struct {
	Products   []*entities.Product `json:"products"`
	Pagination *Pagination         `json:"pagination"`
}

type SearchProductsRequest struct {
	Query string `json:"query" validate:"required,min=1"`
	Limit int    `json:"limit,omitempty" validate:"min=1,max=50"`
}

type SearchProductsResponse struct {
	Products []*entities.Product `json:"products"`
	Total    int                 `json:"total"`
}

type UpdatePriceRequest struct {
	Price decimal.Decimal  `json:"price" validate:"required,gt=0"`
	Cost  *decimal.Decimal `json:"cost,omitempty" validate:"omitempty,gte=0"`
}

type UpdateStockRequest struct {
	Quantity int `json:"quantity" validate:"required,gte=0"`
}

type AdjustStockRequest struct {
	Adjustment int    `json:"adjustment" validate:"required"`
	Reason     string `json:"reason,omitempty" validate:"max=500"`
}

type Pagination struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

type CreateCategoryRequest struct {
	Name           string  `json:"name" validate:"required,min=2,max=100"`
	Description    string  `json:"description,omitempty" validate:"max=500"`
	ParentID       *string `json:"parent_id,omitempty" validate:"omitempty,uuid"`
	IsActive       bool    `json:"is_active"`
	SortOrder      int     `json:"sort_order,omitempty" validate:"gte=0"`
	SEOTitle       string  `json:"seo_title,omitempty" validate:"max=60"`
	SEODescription string  `json:"seo_description,omitempty" validate:"max=160"`
	SEOKeywords    string  `json:"seo_keywords,omitempty" validate:"max=255"`
}

type UpdateCategoryRequest struct {
	Name           *string `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Description    *string `json:"description,omitempty" validate:"omitempty,max=500"`
	IsActive       *bool   `json:"is_active,omitempty"`
	SortOrder      *int    `json:"sort_order,omitempty" validate:"omitempty,gte=0"`
	SEOTitle       *string `json:"seo_title,omitempty" validate:"omitempty,max=60"`
	SEODescription *string `json:"seo_description,omitempty" validate:"omitempty,max=160"`
	SEOKeywords    *string `json:"seo_keywords,omitempty" validate:"omitempty,max=255"`
}

type ListCategoriesRequest struct {
	Search    string  `json:"search,omitempty"`
	ParentID  *string `json:"parent_id,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
	Level     *int    `json:"level,omitempty"`
	Page      int     `json:"page,omitempty" validate:"min=1"`
	Limit     int     `json:"limit,omitempty" validate:"min=1,max=100"`
	SortBy    string  `json:"sort_by,omitempty" validate:"omitempty,oneof=name,sort_order,created_at"`
	SortOrder string  `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

type ListCategoriesResponse struct {
	Categories []*entities.ProductCategory `json:"categories"`
	Pagination *Pagination                 `json:"pagination"`
}

type CategoryTreeResponse struct {
	Tree []*CategoryNode `json:"tree"`
}

type CategoryNode struct {
	*entities.ProductCategory
	Children []*CategoryNode `json:"children"`
}

type CreateProductVariantRequest struct {
	ProductID      string                    `json:"product_id" validate:"required,uuid"`
	SKU            string                    `json:"sku,omitempty" validate:"max=100"`
	Name           string                    `json:"name" validate:"required,max=300"`
	Price          decimal.Decimal           `json:"price" validate:"required,gt=0"`
	Cost           decimal.Decimal           `json:"cost,omitempty" validate:"gte=0"`
	Weight         float64                   `json:"weight,omitempty" validate:"gte=0"`
	Barcode        string                    `json:"barcode,omitempty" validate:"max=50"`
	TrackInventory bool                      `json:"track_inventory"`
	StockQuantity  int                       `json:"stock_quantity,omitempty" validate:"gte=0"`
	MinStockLevel  int                       `json:"min_stock_level,omitempty" validate:"gte=0"`
	MaxStockLevel  int                       `json:"max_stock_level,omitempty" validate:"gte=0"`
	AllowBackorder bool                      `json:"allow_backorder"`
	IsActive       bool                      `json:"is_active"`
	Attributes     []VariantAttributeRequest `json:"attributes,omitempty"`
	Images         []VariantImageRequest     `json:"images,omitempty"`
}

type VariantAttributeRequest struct {
	Name  string `json:"name" validate:"required,max=100"`
	Value string `json:"value" validate:"required,max=255"`
}

type VariantImageRequest struct {
	URL       string `json:"url" validate:"required,url,max=1000"`
	Alt       string `json:"alt,omitempty" validate:"max=255"`
	SortOrder int    `json:"sort_order,omitempty" validate:"gte=0"`
	IsMain    bool   `json:"is_main"`
}

type UpdateProductVariantRequest struct {
	Name           *string                   `json:"name,omitempty" validate:"omitempty,max=300"`
	Price          *decimal.Decimal          `json:"price,omitempty" validate:"omitempty,gt=0"`
	Cost           *decimal.Decimal          `json:"cost,omitempty" validate:"omitempty,gte=0"`
	Weight         *float64                  `json:"weight,omitempty" validate:"omitempty,gte=0"`
	Barcode        *string                   `json:"barcode,omitempty" validate:"omitempty,max=50"`
	TrackInventory *bool                     `json:"track_inventory,omitempty"`
	MinStockLevel  *int                      `json:"min_stock_level,omitempty" validate:"omitempty,gte=0"`
	MaxStockLevel  *int                      `json:"max_stock_level,omitempty" validate:"omitempty,gte=0"`
	AllowBackorder *bool                     `json:"allow_backorder,omitempty"`
	IsActive       *bool                     `json:"is_active,omitempty"`
	Attributes     []VariantAttributeRequest `json:"attributes,omitempty"`
}

type ListProductVariantsRequest struct {
	Search    string           `json:"search,omitempty"`
	MinPrice  *decimal.Decimal `json:"min_price,omitempty"`
	MaxPrice  *decimal.Decimal `json:"max_price,omitempty"`
	IsActive  *bool            `json:"is_active,omitempty"`
	InStock   *bool            `json:"in_stock,omitempty"`
	Page      int              `json:"page,omitempty" validate:"min=1"`
	Limit     int              `json:"limit,omitempty" validate:"min=1,max=100"`
	SortBy    string           `json:"sort_by,omitempty" validate:"omitempty,oneof=name,price,created_at"`
	SortOrder string           `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

type ListProductVariantsResponse struct {
	Variants   []*entities.ProductVariant `json:"variants"`
	Pagination *Pagination                `json:"pagination"`
}

type StockLevelResponse struct {
	ProductID      uuid.UUID `json:"product_id"`
	CurrentStock   int       `json:"current_stock"`
	MinStockLevel  int       `json:"min_stock_level"`
	MaxStockLevel  int       `json:"max_stock_level"`
	IsLowStock     bool      `json:"is_low_stock"`
	IsOutOfStock   bool      `json:"is_out_of_stock"`
	TrackInventory bool      `json:"track_inventory"`
	AllowBackorder bool      `json:"allow_backorder"`
	LastUpdated    time.Time `json:"last_updated"`
}

type AvailabilityResponse struct {
	ProductID        uuid.UUID `json:"product_id"`
	RequestedQty     int       `json:"requested_qty"`
	Available        bool      `json:"available"`
	Reason           string    `json:"reason,omitempty"`
	CanFulfill       bool      `json:"can_fulfill"`
	BackorderAllowed bool      `json:"backorder_allowed"`
}

type GetProductStatsRequest struct {
	CategoryID    *string    `json:"category_id,omitempty"`
	IsActive      *bool      `json:"is_active,omitempty"`
	IsFeatured    *bool      `json:"is_featured,omitempty"`
	IsDigital     *bool      `json:"is_digital,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
}

// Errors
var (
	ErrProductNotFound          = errors.New("product not found")
	ErrProductAlreadyExists     = errors.New("product already exists")
	ErrInvalidSKU               = errors.New("invalid SKU")
	ErrInvalidPrice             = errors.New("invalid price")
	ErrInvalidStockLevel        = errors.New("invalid stock level")
	ErrInsufficientStock        = errors.New("insufficient stock")
	ErrCategoryNotFound         = errors.New("category not found")
	ErrCategoryAlreadyExists    = errors.New("category already exists")
	ErrInvalidCategoryHierarchy = errors.New("invalid category hierarchy")
	ErrCategoryHasProducts      = errors.New("category has products")
	ErrCategoryHasChildren      = errors.New("category has children")
	ErrVariantNotFound          = errors.New("product variant not found")
	ErrVariantAlreadyExists     = errors.New("product variant already exists")
	ErrInvalidQuantity          = errors.New("invalid quantity")
)

// ServiceImpl implements the product service interface
type ServiceImpl struct {
	productRepo      repositories.ProductRepository
	categoryRepo     repositories.CategoryRepository
	variantRepo      repositories.ProductVariantRepository
	variantAttrRepo  repositories.VariantAttributeRepository
	variantImageRepo repositories.VariantImageRepository
}

// NewService creates a new product service instance
func NewService(
	productRepo repositories.ProductRepository,
	categoryRepo repositories.CategoryRepository,
	variantRepo repositories.ProductVariantRepository,
	variantAttrRepo repositories.VariantAttributeRepository,
	variantImageRepo repositories.VariantImageRepository,
) Service {
	return &ServiceImpl{
		productRepo:      productRepo,
		categoryRepo:     categoryRepo,
		variantRepo:      variantRepo,
		variantAttrRepo:  variantAttrRepo,
		variantImageRepo: variantImageRepo,
	}
}

// Product Management Methods

// CreateProduct creates a new product with validation and automatic SKU generation
func (s *ServiceImpl) CreateProduct(ctx context.Context, req *CreateProductRequest) (*entities.Product, error) {
	// Validate business rules
	if err := s.validateCreateProductRequest(ctx, req); err != nil {
		return nil, err
	}

	// Generate SKU if not provided
	sku := req.SKU
	if sku == "" {
		generatedSKU, err := s.generateSKU(ctx, req.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to generate SKU: %w", err)
		}
		sku = generatedSKU
	} else {
		// Check if SKU already exists
		exists, err := s.productRepo.ExistsBySKU(ctx, sku)
		if err != nil {
			return nil, fmt.Errorf("failed to check SKU existence: %w", err)
		}
		if exists {
			return nil, ErrProductAlreadyExists
		}
	}

	// Parse category ID
	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("invalid category ID: %w", err)
	}

	// Check if category exists
	_, err = s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, ErrCategoryNotFound
	}

	// Create product entity
	product := &entities.Product{
		ID:               uuid.New(),
		SKU:              strings.ToUpper(strings.TrimSpace(sku)),
		Name:             strings.TrimSpace(req.Name),
		Description:      strings.TrimSpace(req.Description),
		ShortDescription: strings.TrimSpace(req.ShortDescription),
		CategoryID:       categoryID,
		Price:            req.Price,
		Cost:             req.Cost,
		Weight:           req.Weight,
		Length:           req.Length,
		Width:            req.Width,
		Height:           req.Height,
		Volume:           req.Length * req.Width * req.Height, // Calculate volume
		Barcode:          strings.TrimSpace(req.Barcode),
		TrackInventory:   req.TrackInventory,
		StockQuantity:    req.StockQuantity,
		MinStockLevel:    req.MinStockLevel,
		MaxStockLevel:    req.MaxStockLevel,
		AllowBackorder:   req.AllowBackorder,
		RequiresShipping: req.RequiresShipping,
		Taxable:          req.Taxable,
		TaxRate:          req.TaxRate,
		IsActive:         true, // Always create active products
		IsFeatured:       req.IsFeatured,
		IsDigital:        req.IsDigital,
		DownloadURL:      strings.TrimSpace(req.DownloadURL),
		MaxDownloads:     req.MaxDownloads,
		ExpiryDays:       req.ExpiryDays,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	// Validate product entity
	if err := product.Validate(); err != nil {
		return nil, fmt.Errorf("invalid product data: %w", err)
	}

	// Save product to database
	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	return product, nil
}

// GetProduct retrieves a product by ID
func (s *ServiceImpl) GetProduct(ctx context.Context, id string) (*entities.Product, error) {
	productID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid product ID: %w", err)
	}

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return product, nil
}

// GetProductBySKU retrieves a product by SKU
func (s *ServiceImpl) GetProductBySKU(ctx context.Context, sku string) (*entities.Product, error) {
	sku = strings.ToUpper(strings.TrimSpace(sku))
	if sku == "" {
		return nil, ErrInvalidSKU
	}

	product, err := s.productRepo.GetBySKU(ctx, sku)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product by SKU: %w", err)
	}

	return product, nil
}

// UpdateProduct updates product information with business rule validation
func (s *ServiceImpl) UpdateProduct(ctx context.Context, id string, req *UpdateProductRequest) (*entities.Product, error) {
	productID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid product ID: %w", err)
	}

	// Get existing product
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	// Update fields if provided
	if req.Name != nil {
		product.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		product.Description = strings.TrimSpace(*req.Description)
	}
	if req.ShortDescription != nil {
		product.ShortDescription = strings.TrimSpace(*req.ShortDescription)
	}
	if req.CategoryID != nil {
		categoryID, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("invalid category ID: %w", err)
		}
		// Check if category exists
		_, err = s.categoryRepo.GetByID(ctx, categoryID)
		if err != nil {
			return nil, ErrCategoryNotFound
		}
		product.CategoryID = categoryID
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.Cost != nil {
		product.Cost = *req.Cost
	}
	if req.Weight != nil {
		product.Weight = *req.Weight
	}
	if req.Length != nil {
		product.Length = *req.Length
	}
	if req.Width != nil {
		product.Width = *req.Width
	}
	if req.Height != nil {
		product.Height = *req.Height
	}
	if req.Barcode != nil {
		product.Barcode = strings.TrimSpace(*req.Barcode)
	}
	if req.TrackInventory != nil {
		product.TrackInventory = *req.TrackInventory
	}
	if req.MinStockLevel != nil {
		product.MinStockLevel = *req.MinStockLevel
	}
	if req.MaxStockLevel != nil {
		product.MaxStockLevel = *req.MaxStockLevel
	}
	if req.AllowBackorder != nil {
		product.AllowBackorder = *req.AllowBackorder
	}
	if req.RequiresShipping != nil {
		product.RequiresShipping = *req.RequiresShipping
	}
	if req.Taxable != nil {
		product.Taxable = *req.Taxable
	}
	if req.TaxRate != nil {
		product.TaxRate = *req.TaxRate
	}
	if req.IsFeatured != nil {
		product.IsFeatured = *req.IsFeatured
	}
	if req.IsDigital != nil {
		product.IsDigital = *req.IsDigital
	}
	if req.DownloadURL != nil {
		product.DownloadURL = strings.TrimSpace(*req.DownloadURL)
	}
	if req.MaxDownloads != nil {
		product.MaxDownloads = *req.MaxDownloads
	}
	if req.ExpiryDays != nil {
		product.ExpiryDays = *req.ExpiryDays
	}

	// Recalculate volume if dimensions changed
	product.Volume = product.Length * product.Width * product.Height

	// Validate updated product
	if err := product.Validate(); err != nil {
		return nil, fmt.Errorf("invalid product data: %w", err)
	}

	// Save updates
	if err := s.productRepo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	return product, nil
}

// DeleteProduct deletes a product (soft delete by setting inactive)
func (s *ServiceImpl) DeleteProduct(ctx context.Context, id string) error {
	productID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid product ID: %w", err)
	}

	// Check if product exists
	_, err = s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrProductNotFound
		}
		return fmt.Errorf("failed to get product: %w", err)
	}

	// Perform soft delete
	return s.productRepo.Delete(ctx, productID)
}

// ListProducts retrieves a paginated list of products with filtering
func (s *ServiceImpl) ListProducts(ctx context.Context, req *ListProductsRequest) (*ListProductsResponse, error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Build filter
	filter := repositories.ProductFilter{
		Search:         strings.TrimSpace(req.Search),
		MinPrice:       req.MinPrice,
		MaxPrice:       req.MaxPrice,
		IsActive:       req.IsActive,
		IsFeatured:     req.IsFeatured,
		IsDigital:      req.IsDigital,
		TrackInventory: req.TrackInventory,
		CreatedAfter:   req.CreatedAfter,
		CreatedBefore:  req.CreatedBefore,
		Page:           req.Page,
		Limit:          req.Limit,
		SortBy:         req.SortBy,
		SortOrder:      req.SortOrder,
	}

	// Handle category filters
	if req.CategoryID != "" {
		categoryID, err := uuid.Parse(req.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("invalid category ID: %w", err)
		}
		filter.CategoryID = &categoryID
	}

	if len(req.CategoryIDs) > 0 {
		categoryIDs := make([]uuid.UUID, len(req.CategoryIDs))
		for i, id := range req.CategoryIDs {
			categoryID, err := uuid.Parse(id)
			if err != nil {
				return nil, fmt.Errorf("invalid category ID at index %d: %w", i, err)
			}
			categoryIDs[i] = categoryID
		}
		filter.CategoryIDs = categoryIDs
	}

	// Handle stock filters
	if req.InStock != nil || req.LowStock != nil {
		if req.InStock != nil && *req.InStock {
			// Only products in stock
			products, err := s.productRepo.GetActive(ctx, req.Limit)
			if err != nil {
				return nil, fmt.Errorf("failed to get active products: %w", err)
			}
			return s.buildListProductsResponse(products, req.Page, req.Limit)
		}

		if req.LowStock != nil && *req.LowStock {
			// Only low stock products
			threshold := 10 // Default threshold
			products, err := s.productRepo.GetLowStock(ctx, threshold)
			if err != nil {
				return nil, fmt.Errorf("failed to get low stock products: %w", err)
			}
			return s.buildListProductsResponse(products, req.Page, req.Limit)
		}
	}

	// Get products and total count
	products, err := s.productRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	total, err := s.productRepo.Count(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count products: %w", err)
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))
	hasNext := req.Page < totalPages
	hasPrev := req.Page > 1

	return &ListProductsResponse{
		Products: products,
		Pagination: &Pagination{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    hasNext,
			HasPrev:    hasPrev,
		},
	}, nil
}

// SearchProducts searches products by query string
func (s *ServiceImpl) SearchProducts(ctx context.Context, req *SearchProductsRequest) (*SearchProductsResponse, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, errors.New("search query cannot be empty")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	products, err := s.productRepo.Search(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search products: %w", err)
	}

	return &SearchProductsResponse{
		Products: products,
		Total:    len(products),
	}, nil
}

// Product Operations Methods

// ActivateProduct activates a product
func (s *ServiceImpl) ActivateProduct(ctx context.Context, id string) error {
	product, err := s.GetProduct(ctx, id)
	if err != nil {
		return err
	}

	product.Activate()
	return s.productRepo.Update(ctx, product)
}

// DeactivateProduct deactivates a product
func (s *ServiceImpl) DeactivateProduct(ctx context.Context, id string) error {
	product, err := s.GetProduct(ctx, id)
	if err != nil {
		return err
	}

	product.Deactivate()
	return s.productRepo.Update(ctx, product)
}

// SetFeaturedProduct sets or unsets a product as featured
func (s *ServiceImpl) SetFeaturedProduct(ctx context.Context, id string, featured bool) error {
	product, err := s.GetProduct(ctx, id)
	if err != nil {
		return err
	}

	product.SetFeatured(featured)
	return s.productRepo.Update(ctx, product)
}

// UpdateProductPrice updates product price with validation
func (s *ServiceImpl) UpdateProductPrice(ctx context.Context, id string, req *UpdatePriceRequest) error {
	product, err := s.GetProduct(ctx, id)
	if err != nil {
		return err
	}

	if err := product.UpdatePrice(req.Price); err != nil {
		return fmt.Errorf("invalid price: %w", err)
	}

	if req.Cost != nil {
		if err := product.UpdateCost(*req.Cost); err != nil {
			return fmt.Errorf("invalid cost: %w", err)
		}
	}

	return s.productRepo.Update(ctx, product)
}

// UpdateProductStock updates product stock quantity
func (s *ServiceImpl) UpdateProductStock(ctx context.Context, id string, req *UpdateStockRequest) error {
	if req.Quantity < 0 {
		return ErrInvalidStockLevel
	}

	return s.productRepo.UpdateStock(ctx, uuid.MustParse(id), req.Quantity)
}

// AdjustProductStock adjusts product stock by a given amount
func (s *ServiceImpl) AdjustProductStock(ctx context.Context, id string, req *AdjustStockRequest) error {
	productID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid product ID: %w", err)
	}

	// Get product to validate
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return ErrProductNotFound
	}

	newQuantity := product.StockQuantity + req.Adjustment
	if newQuantity < 0 {
		return ErrInvalidStockLevel
	}

	return s.productRepo.AdjustStock(ctx, productID, req.Adjustment)
}

// Category Management Methods

// CreateCategory creates a new product category with hierarchy validation
func (s *ServiceImpl) CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*entities.ProductCategory, error) {
	// Validate business rules
	if err := s.validateCreateCategoryRequest(ctx, req); err != nil {
		return nil, err
	}

	// Parse parent ID if provided
	var parentID *uuid.UUID
	if req.ParentID != nil {
		parsedUUID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent ID: %w", err)
		}

		// Check if parent category exists
		parent, err := s.categoryRepo.GetByID(ctx, parsedUUID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return nil, ErrCategoryNotFound
			}
			return nil, fmt.Errorf("failed to get parent category: %w", err)
		}

		// Prevent creating category under itself (circular reference)
		if parent.ID == parsedUUID {
			return nil, ErrInvalidCategoryHierarchy
		}

		parentID = &parsedUUID
	}

	// Generate category path
	path, err := s.generateCategoryPath(ctx, req.Name, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate category path: %w", err)
	}

	// Create category entity
	category := &entities.ProductCategory{
		ID:             uuid.New(),
		Name:           strings.TrimSpace(req.Name),
		Description:    strings.TrimSpace(req.Description),
		ParentID:       parentID,
		Path:           path,
		Level:          s.calculateCategoryLevel(parentID),
		IsActive:       req.IsActive,
		SortOrder:      req.SortOrder,
		SEOTitle:       strings.TrimSpace(req.SEOTitle),
		SEODescription: strings.TrimSpace(req.SEODescription),
		SEOKeywords:    strings.TrimSpace(req.SEOKeywords),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// Validate category entity
	if err := category.Validate(); err != nil {
		return nil, fmt.Errorf("invalid category data: %w", err)
	}

	// Save category to database
	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return category, nil
}

// GetCategory retrieves a category by ID
func (s *ServiceImpl) GetCategory(ctx context.Context, id string) (*entities.ProductCategory, error) {
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid category ID: %w", err)
	}

	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	return category, nil
}

// GetCategoryByPath retrieves a category by path
func (s *ServiceImpl) GetCategoryByPath(ctx context.Context, path string) (*entities.ProductCategory, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("category path cannot be empty")
	}

	category, err := s.categoryRepo.GetByPath(ctx, path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to get category by path: %w", err)
	}

	return category, nil
}

// UpdateCategory updates category information
func (s *ServiceImpl) UpdateCategory(ctx context.Context, id string, req *UpdateCategoryRequest) (*entities.ProductCategory, error) {
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid category ID: %w", err)
	}

	// Get existing category
	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	// Update fields if provided
	if req.Name != nil {
		category.Name = strings.TrimSpace(*req.Name)
		// Regenerate path if name changed
		path, err := s.generateCategoryPath(ctx, category.Name, category.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to regenerate category path: %w", err)
		}
		category.Path = path
	}
	if req.Description != nil {
		category.Description = strings.TrimSpace(*req.Description)
	}
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}
	if req.SortOrder != nil {
		category.SortOrder = *req.SortOrder
	}
	if req.SEOTitle != nil {
		category.SEOTitle = strings.TrimSpace(*req.SEOTitle)
	}
	if req.SEODescription != nil {
		category.SEODescription = strings.TrimSpace(*req.SEODescription)
	}
	if req.SEOKeywords != nil {
		category.SEOKeywords = strings.TrimSpace(*req.SEOKeywords)
	}

	// Validate updated category
	if err := category.Validate(); err != nil {
		return nil, fmt.Errorf("invalid category data: %w", err)
	}

	// Save updates
	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	return category, nil
}

// DeleteCategory deletes a category with validation
func (s *ServiceImpl) DeleteCategory(ctx context.Context, id string) error {
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid category ID: %w", err)
	}

	// Check if category exists
	_, err = s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrCategoryNotFound
		}
		return fmt.Errorf("failed to get category: %w", err)
	}

	// Check if category has products
	productCount, err := s.categoryRepo.CountProducts(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("failed to check category products: %w", err)
	}
	if productCount > 0 {
		return ErrCategoryHasProducts
	}

	// Check if category has children
	children, err := s.categoryRepo.GetChildren(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("failed to check category children: %w", err)
	}
	if len(children) > 0 {
		return ErrCategoryHasChildren
	}

	// Delete category
	return s.categoryRepo.Delete(ctx, categoryID)
}

// ListCategories retrieves a paginated list of categories
func (s *ServiceImpl) ListCategories(ctx context.Context, req *ListCategoriesRequest) (*ListCategoriesResponse, error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Build filter
	filter := repositories.CategoryFilter{
		Search:    strings.TrimSpace(req.Search),
		IsActive:  req.IsActive,
		Level:     req.Level,
		Page:      req.Page,
		Limit:     req.Limit,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	// Handle parent filter
	if req.ParentID != nil {
		parentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent ID: %w", err)
		}
		filter.ParentID = &parentID
	}

	// Get categories and total count
	categories, err := s.categoryRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}

	total, err := s.categoryRepo.Count(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count categories: %w", err)
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))
	hasNext := req.Page < totalPages
	hasPrev := req.Page > 1

	return &ListCategoriesResponse{
		Categories: categories,
		Pagination: &Pagination{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    hasNext,
			HasPrev:    hasPrev,
		},
	}, nil
}

// GetCategoryTree retrieves the complete category tree
func (s *ServiceImpl) GetCategoryTree(ctx context.Context) (*CategoryTreeResponse, error) {
	// Get root categories
	rootCategories, err := s.categoryRepo.ListRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get root categories: %w", err)
	}

	// Build tree recursively
	tree := make([]*CategoryNode, len(rootCategories))
	for i, category := range rootCategories {
		tree[i] = &CategoryNode{
			ProductCategory: category,
			Children:        s.buildCategoryTree(ctx, category.ID),
		}
	}

	return &CategoryTreeResponse{
		Tree: tree,
	}, nil
}

// GetCategoryDescendants retrieves all descendant categories
func (s *ServiceImpl) GetCategoryDescendants(ctx context.Context, id string) ([]*entities.ProductCategory, error) {
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid category ID: %w", err)
	}

	// Check if category exists
	_, err = s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	descendants, err := s.categoryRepo.GetDescendants(ctx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category descendants: %w", err)
	}

	return descendants, nil
}

// GetCategoryAncestors retrieves all ancestor categories
func (s *ServiceImpl) GetCategoryAncestors(ctx context.Context, id string) ([]*entities.ProductCategory, error) {
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid category ID: %w", err)
	}

	// Check if category exists
	_, err = s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	ancestors, err := s.categoryRepo.GetAncestors(ctx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category ancestors: %w", err)
	}

	return ancestors, nil
}

// MoveCategory moves a category to a new parent
func (s *ServiceImpl) MoveCategory(ctx context.Context, id string, newParentID *string) error {
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid category ID: %w", err)
	}

	// Get category to move
	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrCategoryNotFound
		}
		return fmt.Errorf("failed to get category: %w", err)
	}

	// Parse new parent ID if provided
	var newParentUUID *uuid.UUID
	if newParentID != nil {
		parsedUUID, err := uuid.Parse(*newParentID)
		if err != nil {
			return fmt.Errorf("invalid new parent ID: %w", err)
		}

		// Check if new parent exists
		_, err = s.categoryRepo.GetByID(ctx, parsedUUID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return ErrCategoryNotFound
			}
			return fmt.Errorf("failed to get new parent category: %w", err)
		}

		// Prevent moving category to be its own descendant
		descendants, err := s.categoryRepo.GetDescendants(ctx, categoryID)
		if err != nil {
			return fmt.Errorf("failed to check category descendants: %w", err)
		}

		for _, descendant := range descendants {
			if descendant.ID == parsedUUID {
				return ErrInvalidCategoryHierarchy
			}
		}

		newParentUUID = &parsedUUID
	}

	// Update category parent and regenerate path
	category.ParentID = newParentUUID
	category.Level = s.calculateCategoryLevel(newParentUUID)

	path, err := s.generateCategoryPath(ctx, category.Name, newParentUUID)
	if err != nil {
		return fmt.Errorf("failed to generate new category path: %w", err)
	}
	category.Path = path

	// Save updates
	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return fmt.Errorf("failed to move category: %w", err)
	}

	// Rebuild paths for all descendants
	return s.categoryRepo.RebuildCategoryPaths(ctx, categoryID)
}

// Product Variant Management Methods

// CreateProductVariant creates a new product variant with validation
func (s *ServiceImpl) CreateProductVariant(ctx context.Context, req *CreateProductVariantRequest) (*entities.ProductVariant, error) {
	// Validate business rules
	if err := s.validateCreateProductVariantRequest(ctx, req); err != nil {
		return nil, err
	}

	// Parse product ID
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("invalid product ID: %w", err)
	}

	// Check if product exists
	_, err = s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, ErrProductNotFound
	}

	// Generate SKU if not provided
	sku := req.SKU
	if sku == "" {
		generatedSKU, err := s.generateVariantSKU(ctx, req.Name, req.ProductID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate variant SKU: %w", err)
		}
		sku = generatedSKU
	} else {
		// Check if SKU already exists
		exists, err := s.variantRepo.ExistsBySKU(ctx, sku)
		if err != nil {
			return nil, fmt.Errorf("failed to check SKU existence: %w", err)
		}
		if exists {
			return nil, ErrVariantAlreadyExists
		}
	}

	// Create product variant entity
	variant := &entities.ProductVariant{
		ID:             uuid.New(),
		ProductID:      productID,
		SKU:            strings.ToUpper(strings.TrimSpace(sku)),
		Name:           strings.TrimSpace(req.Name),
		Price:          req.Price,
		Cost:           req.Cost,
		Weight:         req.Weight,
		Barcode:        strings.TrimSpace(req.Barcode),
		TrackInventory: req.TrackInventory,
		StockQuantity:  req.StockQuantity,
		MinStockLevel:  req.MinStockLevel,
		MaxStockLevel:  req.MaxStockLevel,
		AllowBackorder: req.AllowBackorder,
		IsActive:       req.IsActive,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// Validate variant entity
	if err := variant.Validate(); err != nil {
		return nil, fmt.Errorf("invalid product variant data: %w", err)
	}

	// Save variant to database
	if err := s.variantRepo.Create(ctx, variant); err != nil {
		return nil, fmt.Errorf("failed to create product variant: %w", err)
	}

	// Create variant attributes if provided
	if len(req.Attributes) > 0 {
		attributes := make([]*entities.VariantAttribute, len(req.Attributes))
		for i, attrReq := range req.Attributes {
			attributes[i] = &entities.VariantAttribute{
				ID:        uuid.New(),
				VariantID: variant.ID,
				Name:      strings.TrimSpace(attrReq.Name),
				Value:     strings.TrimSpace(attrReq.Value),
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
		}

		if err := s.variantAttrRepo.BulkCreate(ctx, attributes); err != nil {
			// Log error but don't fail variant creation
			// In production, this should be monitored
		}
	}

	// Create variant images if provided
	if len(req.Images) > 0 {
		images := make([]*entities.VariantImage, len(req.Images))
		for i, imgReq := range req.Images {
			images[i] = &entities.VariantImage{
				ID:        uuid.New(),
				VariantID: variant.ID,
				URL:       strings.TrimSpace(imgReq.URL),
				Alt:       strings.TrimSpace(imgReq.Alt),
				SortOrder: imgReq.SortOrder,
				IsMain:    imgReq.IsMain,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
		}

		if err := s.variantImageRepo.BulkCreate(ctx, images); err != nil {
			// Log error but don't fail variant creation
			// In production, this should be monitored
		}
	}

	return variant, nil
}

// GetProductVariant retrieves a product variant by ID
func (s *ServiceImpl) GetProductVariant(ctx context.Context, id string) (*entities.ProductVariant, error) {
	variantID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid variant ID: %w", err)
	}

	variant, err := s.variantRepo.GetByID(ctx, variantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrVariantNotFound
		}
		return nil, fmt.Errorf("failed to get product variant: %w", err)
	}

	return variant, nil
}

// GetProductVariantBySKU retrieves a product variant by SKU
func (s *ServiceImpl) GetProductVariantBySKU(ctx context.Context, sku string) (*entities.ProductVariant, error) {
	sku = strings.ToUpper(strings.TrimSpace(sku))
	if sku == "" {
		return nil, ErrInvalidSKU
	}

	variant, err := s.variantRepo.GetBySKU(ctx, sku)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrVariantNotFound
		}
		return nil, fmt.Errorf("failed to get product variant by SKU: %w", err)
	}

	return variant, nil
}

// UpdateProductVariant updates product variant information
func (s *ServiceImpl) UpdateProductVariant(ctx context.Context, id string, req *UpdateProductVariantRequest) (*entities.ProductVariant, error) {
	variantID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid variant ID: %w", err)
	}

	// Get existing variant
	variant, err := s.variantRepo.GetByID(ctx, variantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrVariantNotFound
		}
		return nil, fmt.Errorf("failed to get product variant: %w", err)
	}

	// Update fields if provided
	if req.Name != nil {
		variant.Name = strings.TrimSpace(*req.Name)
	}
	if req.Price != nil {
		variant.Price = *req.Price
	}
	if req.Cost != nil {
		variant.Cost = *req.Cost
	}
	if req.Weight != nil {
		variant.Weight = *req.Weight
	}
	if req.Barcode != nil {
		variant.Barcode = strings.TrimSpace(*req.Barcode)
	}
	if req.TrackInventory != nil {
		variant.TrackInventory = *req.TrackInventory
	}
	if req.MinStockLevel != nil {
		variant.MinStockLevel = *req.MinStockLevel
	}
	if req.MaxStockLevel != nil {
		variant.MaxStockLevel = *req.MaxStockLevel
	}
	if req.AllowBackorder != nil {
		variant.AllowBackorder = *req.AllowBackorder
	}
	if req.IsActive != nil {
		variant.IsActive = *req.IsActive
	}

	// Validate updated variant
	if err := variant.Validate(); err != nil {
		return nil, fmt.Errorf("invalid product variant data: %w", err)
	}

	// Save updates
	if err := s.variantRepo.Update(ctx, variant); err != nil {
		return nil, fmt.Errorf("failed to update product variant: %w", err)
	}

	// Update variant attributes if provided
	if len(req.Attributes) > 0 {
		// Delete existing attributes
		if err := s.variantAttrRepo.DeleteByVariantID(ctx, variantID); err != nil {
			// Log error but don't fail update
		}

		// Create new attributes
		attributes := make([]*entities.VariantAttribute, len(req.Attributes))
		for i, attrReq := range req.Attributes {
			attributes[i] = &entities.VariantAttribute{
				ID:        uuid.New(),
				VariantID: variantID,
				Name:      strings.TrimSpace(attrReq.Name),
				Value:     strings.TrimSpace(attrReq.Value),
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
		}

		if err := s.variantAttrRepo.BulkCreate(ctx, attributes); err != nil {
			// Log error but don't fail update
		}
	}

	return variant, nil
}

// DeleteProductVariant deletes a product variant
func (s *ServiceImpl) DeleteProductVariant(ctx context.Context, id string) error {
	variantID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid variant ID: %w", err)
	}

	// Check if variant exists
	_, err = s.variantRepo.GetByID(ctx, variantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrVariantNotFound
		}
		return fmt.Errorf("failed to get product variant: %w", err)
	}

	// Delete variant
	return s.variantRepo.Delete(ctx, variantID)
}

// ListProductVariants retrieves a paginated list of product variants
func (s *ServiceImpl) ListProductVariants(ctx context.Context, productID string, req *ListProductVariantsRequest) (*ListProductVariantsResponse, error) {
	// Parse product ID
	parsedProductID, err := uuid.Parse(productID)
	if err != nil {
		return nil, fmt.Errorf("invalid product ID: %w", err)
	}

	// Check if product exists
	_, err = s.productRepo.GetByID(ctx, parsedProductID)
	if err != nil {
		return nil, ErrProductNotFound
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Build filter
	filter := repositories.ProductVariantFilter{
		ProductID: &parsedProductID,
		Search:    strings.TrimSpace(req.Search),
		MinPrice:  req.MinPrice,
		MaxPrice:  req.MaxPrice,
		IsActive:  req.IsActive,
		Page:      req.Page,
		Limit:     req.Limit,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	// Get variants and total count
	variants, err := s.variantRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list product variants: %w", err)
	}

	total, err := s.variantRepo.Count(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count product variants: %w", err)
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))
	hasNext := req.Page < totalPages
	hasPrev := req.Page > 1

	return &ListProductVariantsResponse{
		Variants: variants,
		Pagination: &Pagination{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    hasNext,
			HasPrev:    hasPrev,
		},
	}, nil
}

// Inventory and Stock Management Methods

// GetLowStockProducts retrieves products with low stock levels
func (s *ServiceImpl) GetLowStockProducts(ctx context.Context, threshold int) ([]*entities.Product, error) {
	if threshold <= 0 {
		threshold = 10 // Default threshold
	}

	return s.productRepo.GetLowStock(ctx, threshold)
}

// GetProductStockLevel retrieves stock level information for a product
func (s *ServiceImpl) GetProductStockLevel(ctx context.Context, id string) (*StockLevelResponse, error) {
	product, err := s.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	return &StockLevelResponse{
		ProductID:      product.ID,
		CurrentStock:   product.StockQuantity,
		MinStockLevel:  product.MinStockLevel,
		MaxStockLevel:  product.MaxStockLevel,
		IsLowStock:     product.IsLowStock(),
		IsOutOfStock:   !product.IsInStock(),
		TrackInventory: product.TrackInventory,
		AllowBackorder: product.AllowBackorder,
		LastUpdated:    product.UpdatedAt,
	}, nil
}

// CheckProductAvailability checks if a product can fulfill a requested quantity
func (s *ServiceImpl) CheckProductAvailability(ctx context.Context, id string, quantity int) (*AvailabilityResponse, error) {
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}

	product, err := s.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	response := &AvailabilityResponse{
		ProductID:        product.ID,
		RequestedQty:     quantity,
		Available:        product.IsInStock(),
		CanFulfill:       true,
		BackorderAllowed: product.AllowBackorder,
	}

	// Check if product can fulfill the order
	if err := product.CanFulfillOrder(quantity); err != nil {
		response.CanFulfill = false
		response.Reason = err.Error()
	}

	return response, nil
}

// GetProductStats retrieves product statistics
func (s *ServiceImpl) GetProductStats(ctx context.Context, req *GetProductStatsRequest) (*repositories.ProductStats, error) {
	// Build filter
	filter := repositories.ProductFilter{
		IsActive:      req.IsActive,
		IsFeatured:    req.IsFeatured,
		IsDigital:     req.IsDigital,
		CreatedAfter:  req.CreatedAfter,
		CreatedBefore: req.CreatedBefore,
	}

	// Handle category filter
	if req.CategoryID != nil {
		categoryID, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("invalid category ID: %w", err)
		}
		filter.CategoryID = &categoryID
	}

	return s.productRepo.GetProductStats(ctx, filter)
}

// Helper Methods

// buildListProductsResponse builds a paginated response for product lists
func (s *ServiceImpl) buildListProductsResponse(products []*entities.Product, page, limit int) (*ListProductsResponse, error) {
	total := len(products)
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	hasNext := page < totalPages
	hasPrev := page > 1

	// Apply pagination to slice
	start := (page - 1) * limit
	end := start + limit
	if end > total {
		end = total
	}
	if start >= total {
		return &ListProductsResponse{
			Products: []*entities.Product{},
			Pagination: &Pagination{
				Page:       page,
				Limit:      limit,
				Total:      total,
				TotalPages: totalPages,
				HasNext:    hasNext,
				HasPrev:    hasPrev,
			},
		}, nil
	}

	paginatedProducts := products[start:end]

	return &ListProductsResponse{
		Products: paginatedProducts,
		Pagination: &Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    hasNext,
			HasPrev:    hasPrev,
		},
	}, nil
}

// buildCategoryTree recursively builds a category tree structure
func (s *ServiceImpl) buildCategoryTree(ctx context.Context, parentID uuid.UUID) []*CategoryNode {
	children, err := s.categoryRepo.GetChildren(ctx, parentID)
	if err != nil {
		return []*CategoryNode{}
	}

	tree := make([]*CategoryNode, len(children))
	for i, category := range children {
		tree[i] = &CategoryNode{
			ProductCategory: category,
			Children:        s.buildCategoryTree(ctx, category.ID),
		}
	}

	return tree
}

// generateSKU generates a unique SKU based on product name
func (s *ServiceImpl) generateSKU(ctx context.Context, name string) (string, error) {
	// Create base SKU from product name (first 3 letters, uppercase, no spaces)
	base := strings.ToUpper(strings.ReplaceAll(name, " ", ""))
	if len(base) > 3 {
		base = base[:3]
	}

	// Add timestamp suffix for uniqueness
	timestamp := time.Now().Unix()
	sku := fmt.Sprintf("%s-%d", base, timestamp)

	// Ensure SKU doesn't already exist
	exists, err := s.productRepo.ExistsBySKU(ctx, sku)
	if err != nil {
		return "", fmt.Errorf("failed to check SKU uniqueness: %w", err)
	}
	if exists {
		// Try again with different suffix
		timestamp++
		sku = fmt.Sprintf("%s-%d", base, timestamp)
	}

	return sku, nil
}

// generateVariantSKU generates a unique SKU for a product variant
func (s *ServiceImpl) generateVariantSKU(ctx context.Context, name, productID string) (string, error) {
	// Create base SKU from variant name (first 3 letters, uppercase, no spaces)
	base := strings.ToUpper(strings.ReplaceAll(name, " ", ""))
	if len(base) > 3 {
		base = base[:3]
	}

	// Add product ID suffix and timestamp
	timestamp := time.Now().Unix()
	productSuffix := strings.ToUpper(productID[:8]) // Use first 8 characters of product ID
	sku := fmt.Sprintf("%s-%s-%d", base, productSuffix, timestamp)

	// Ensure SKU doesn't already exist
	exists, err := s.variantRepo.ExistsBySKU(ctx, sku)
	if err != nil {
		return "", fmt.Errorf("failed to check variant SKU uniqueness: %w", err)
	}
	if exists {
		// Try again with different suffix
		timestamp++
		sku = fmt.Sprintf("%s-%s-%d", base, productSuffix, timestamp)
	}

	return sku, nil
}

// generateCategoryPath generates a hierarchical path for a category
func (s *ServiceImpl) generateCategoryPath(ctx context.Context, name string, parentID *uuid.UUID) (string, error) {
	// Convert name to URL-friendly format
	pathSegment := strings.ToLower(name)
	pathSegment = strings.ReplaceAll(pathSegment, " ", "-")
	pathSegment = strings.ReplaceAll(pathSegment, "&", "and")
	pathSegment = strings.ReplaceAll(pathSegment, "/", "-")

	// Remove multiple consecutive hyphens
	re := strings.NewReplacer("--", "-")
	pathSegment = re.Replace(pathSegment)

	if parentID == nil {
		return pathSegment, nil
	}

	// Get parent category
	parent, err := s.categoryRepo.GetByID(ctx, *parentID)
	if err != nil {
		return "", fmt.Errorf("failed to get parent category: %w", err)
	}

	// Combine parent path with current segment
	return fmt.Sprintf("%s/%s", parent.Path, pathSegment), nil
}

// calculateCategoryLevel calculates the level of a category in the hierarchy
func (s *ServiceImpl) calculateCategoryLevel(parentID *uuid.UUID) int {
	if parentID == nil {
		return 0 // Root level
	}

	// In a real implementation, you might fetch the parent to get its level
	// For now, we'll use a simple approach
	return 1 // Child level
}

// Validation Methods

// validateCreateProductRequest validates product creation request
func (s *ServiceImpl) validateCreateProductRequest(ctx context.Context, req *CreateProductRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("product name is required")
	}

	if req.Price.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidPrice
	}

	if req.Cost.LessThan(decimal.Zero) {
		return errors.New("cost cannot be negative")
	}

	// Validate cost is not higher than price
	if req.Cost.GreaterThan(req.Price) {
		return errors.New("cost cannot be higher than price")
	}

	// Validate digital product settings
	if req.IsDigital {
		if req.RequiresShipping {
			return errors.New("digital products cannot require shipping")
		}
		if req.DownloadURL == "" {
			return errors.New("digital products must have a download URL")
		}
	}

	return nil
}

// validateCreateCategoryRequest validates category creation request
func (s *ServiceImpl) validateCreateCategoryRequest(ctx context.Context, req *CreateCategoryRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("category name is required")
	}

	if len(req.Name) < 2 || len(req.Name) > 100 {
		return errors.New("category name must be between 2 and 100 characters")
	}

	// Check if category name already exists under the same parent
	if req.ParentID != nil {
		parentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return fmt.Errorf("invalid parent ID: %w", err)
		}

		exists, err := s.categoryRepo.ExistsByName(ctx, req.Name, &parentID)
		if err != nil {
			return fmt.Errorf("failed to check category name uniqueness: %w", err)
		}
		if exists {
			return ErrCategoryAlreadyExists
		}
	} else {
		// Check root level categories
		exists, err := s.categoryRepo.ExistsByName(ctx, req.Name, nil)
		if err != nil {
			return fmt.Errorf("failed to check category name uniqueness: %w", err)
		}
		if exists {
			return ErrCategoryAlreadyExists
		}
	}

	return nil
}

// validateCreateProductVariantRequest validates product variant creation request
func (s *ServiceImpl) validateCreateProductVariantRequest(ctx context.Context, req *CreateProductVariantRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("variant name is required")
	}

	if req.Price.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidPrice
	}

	if req.Cost.LessThan(decimal.Zero) {
		return errors.New("cost cannot be negative")
	}

	// Validate cost is not higher than price
	if req.Cost.GreaterThan(req.Price) {
		return errors.New("cost cannot be higher than price")
	}

	return nil
}
