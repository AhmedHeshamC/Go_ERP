package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Product Response DTOs

// ProductResponse represents a product response
type ProductResponse struct {
	ID               uuid.UUID        `json:"id"`
	SKU              string           `json:"sku"`
	Name             string           `json:"name"`
	Description      string           `json:"description,omitempty"`
	ShortDescription string           `json:"short_description,omitempty"`
	CategoryID       uuid.UUID        `json:"category_id"`
	Category         *CategoryInfo    `json:"category,omitempty"`
	Price            decimal.Decimal  `json:"price"`
	Cost             *decimal.Decimal `json:"cost,omitempty"` // Cost is often hidden from public APIs
	Weight           float64          `json:"weight,omitempty"`
	Dimensions       string           `json:"dimensions,omitempty"`
	Length           float64          `json:"length,omitempty"`
	Width            float64          `json:"width,omitempty"`
	Height           float64          `json:"height,omitempty"`
	Volume           float64          `json:"volume,omitempty"`
	Barcode          string           `json:"barcode,omitempty"`
	TrackInventory   bool             `json:"track_inventory"`
	StockQuantity    int              `json:"stock_quantity"`
	MinStockLevel    int              `json:"min_stock_level"`
	MaxStockLevel    int              `json:"max_stock_level,omitempty"`
	AllowBackorder   bool             `json:"allow_backorder"`
	RequiresShipping bool             `json:"requires_shipping"`
	Taxable          bool             `json:"taxable"`
	TaxRate          decimal.Decimal  `json:"tax_rate,omitempty"`
	IsActive         bool             `json:"is_active"`
	IsFeatured       bool             `json:"is_featured"`
	IsDigital        bool             `json:"is_digital"`
	DownloadURL      string           `json:"download_url,omitempty"`
	MaxDownloads     int              `json:"max_downloads,omitempty"`
	ExpiryDays       int              `json:"expiry_days,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// CategoryInfo represents basic category information in product responses
type CategoryInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Path string    `json:"path"`
}

// ProductListResponse represents a paginated product list response
type ProductListResponse struct {
	Products   []*ProductResponse `json:"products"`
	Pagination *PaginationInfo    `json:"pagination"`
}

// SearchProductsResponse represents a product search response
type SearchProductsResponse struct {
	Products []*ProductResponse `json:"products"`
	Total    int                `json:"total"`
	Query    string             `json:"query"`
}

// StockLevelResponse represents stock level information
type StockLevelResponse struct {
	ProductID      uuid.UUID `json:"product_id"`
	CurrentStock   int       `json:"current_stock"`
	MinStockLevel  int       `json:"min_stock_level"`
	MaxStockLevel  int       `json:"max_stock_level,omitempty"`
	IsLowStock     bool      `json:"is_low_stock"`
	IsOutOfStock   bool      `json:"is_out_of_stock"`
	TrackInventory bool      `json:"track_inventory"`
	AllowBackorder bool      `json:"allow_backorder"`
	LastUpdated    time.Time `json:"last_updated"`
}

// AvailabilityResponse represents product availability information
type AvailabilityResponse struct {
	ProductID        uuid.UUID `json:"product_id"`
	RequestedQty     int       `json:"requested_qty"`
	Available        bool      `json:"available"`
	Reason           string    `json:"reason,omitempty"`
	CanFulfill       bool      `json:"can_fulfill"`
	BackorderAllowed bool      `json:"backorder_allowed"`
}

// Category Response DTOs

// CategoryResponse represents a category response
type CategoryResponse struct {
	ID             uuid.UUID     `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description,omitempty"`
	ParentID       *uuid.UUID    `json:"parent_id,omitempty"`
	Parent         *CategoryInfo `json:"parent,omitempty"`
	Path           string        `json:"path"`
	Level          int           `json:"level"`
	IsActive       bool          `json:"is_active"`
	SortOrder      int           `json:"sort_order"`
	SEOTitle       string        `json:"seo_title,omitempty"`
	SEODescription string        `json:"seo_description,omitempty"`
	SEOKeywords    string        `json:"seo_keywords,omitempty"`
	ProductCount   int           `json:"product_count,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// CategoryTreeNode represents a category tree node
type CategoryTreeNode struct {
	*CategoryResponse
	Children []*CategoryTreeNode `json:"children,omitempty"`
}

// CategoryTreeResponse represents a category tree response
type CategoryTreeResponse struct {
	Tree []*CategoryTreeNode `json:"tree"`
}

// CategoryListResponse represents a paginated category list response
type CategoryListResponse struct {
	Categories []*CategoryResponse `json:"categories"`
	Pagination *PaginationInfo     `json:"pagination"`
}

// Product Variant Response DTOs

// VariantAttribute represents a variant attribute
type VariantAttribute struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Value string    `json:"value"`
}

// VariantImage represents a variant image
type VariantImage struct {
	ID        uuid.UUID `json:"id"`
	URL       string    `json:"url"`
	Alt       string    `json:"alt,omitempty"`
	SortOrder int       `json:"sort_order"`
	IsMain    bool      `json:"is_main"`
}

// ProductVariantResponse represents a product variant response
type ProductVariantResponse struct {
	ID             uuid.UUID           `json:"id"`
	ProductID      uuid.UUID           `json:"product_id"`
	SKU            string              `json:"sku"`
	Name           string              `json:"name"`
	Price          decimal.Decimal     `json:"price"`
	Cost           *decimal.Decimal    `json:"cost,omitempty"`
	Weight         float64             `json:"weight,omitempty"`
	Barcode        string              `json:"barcode,omitempty"`
	TrackInventory bool                `json:"track_inventory"`
	StockQuantity  int                 `json:"stock_quantity"`
	MinStockLevel  int                 `json:"min_stock_level"`
	MaxStockLevel  int                 `json:"max_stock_level,omitempty"`
	AllowBackorder bool                `json:"allow_backorder"`
	IsActive       bool                `json:"is_active"`
	Attributes     []*VariantAttribute `json:"attributes,omitempty"`
	Images         []*VariantImage     `json:"images,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// ProductVariantListResponse represents a paginated product variant list response
type ProductVariantListResponse struct {
	Variants   []*ProductVariantResponse `json:"variants"`
	Pagination *PaginationInfo           `json:"pagination"`
}

// Pagination and Common DTOs

// PaginationInfo represents pagination information
type PaginationInfo struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// Product Stats Response DTOs

// ProductStatsResponse represents product statistics
type ProductStatsResponse struct {
	TotalProducts      int             `json:"total_products"`
	ActiveProducts     int             `json:"active_products"`
	InactiveProducts   int             `json:"inactive_products"`
	FeaturedProducts   int             `json:"featured_products"`
	LowStockProducts   int             `json:"low_stock_products"`
	OutOfStockProducts int             `json:"out_of_stock_products"`
	DigitalProducts    int             `json:"digital_products"`
	PhysicalProducts   int             `json:"physical_products"`
	AveragePrice       decimal.Decimal `json:"average_price"`
	MinPrice           decimal.Decimal `json:"min_price"`
	MaxPrice           decimal.Decimal `json:"max_price"`
	TotalStockValue    decimal.Decimal `json:"total_stock_value"`
}

// Bulk Operation Response DTOs

// BulkOperationResponse represents a bulk operation response
type BulkOperationResponse struct {
	SuccessCount int                   `json:"success_count"`
	FailedCount  int                   `json:"failed_count"`
	TotalCount   int                   `json:"total_count"`
	Errors       []BulkOperationError  `json:"errors,omitempty"`
	Results      []BulkOperationResult `json:"results,omitempty"`
}

// BulkOperationError represents an error in a bulk operation
type BulkOperationError struct {
	Index   int    `json:"index"`
	ID      string `json:"id,omitempty"`
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// BulkOperationResult represents a result in a bulk operation
type BulkOperationResult struct {
	Index   int         `json:"index"`
	ID      string      `json:"id"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
