package dto

import (
	"github.com/shopspring/decimal"
)

// Product Request DTOs

// CreateProductRequest represents a product creation request
type CreateProductRequest struct {
	SKU              string          `json:"sku,omitempty" binding:"omitempty,max=100"`
	Name             string          `json:"name" binding:"required,max=300"`
	Description      string          `json:"description,omitempty" binding:"omitempty,max=2000"`
	ShortDescription string          `json:"short_description,omitempty" binding:"omitempty,max=500"`
	CategoryID       string          `json:"category_id" binding:"required,uuid"`
	Price            decimal.Decimal `json:"price" binding:"required,gt=0"`
	Cost             decimal.Decimal `json:"cost,omitempty" binding:"omitempty,gte=0"`
	Weight           float64         `json:"weight,omitempty" binding:"omitempty,gte=0"`
	Length           float64         `json:"length,omitempty" binding:"omitempty,gte=0"`
	Width            float64         `json:"width,omitempty" binding:"omitempty,gte=0"`
	Height           float64         `json:"height,omitempty" binding:"omitempty,gte=0"`
	Barcode          string          `json:"barcode,omitempty" binding:"omitempty,max=50"`
	TrackInventory   bool            `json:"track_inventory"`
	StockQuantity    int             `json:"stock_quantity,omitempty" binding:"omitempty,gte=0"`
	MinStockLevel    int             `json:"min_stock_level,omitempty" binding:"omitempty,gte=0"`
	MaxStockLevel    int             `json:"max_stock_level,omitempty" binding:"omitempty,gte=0"`
	AllowBackorder   bool            `json:"allow_backorder"`
	RequiresShipping bool            `json:"requires_shipping"`
	Taxable          bool            `json:"taxable"`
	TaxRate          decimal.Decimal `json:"tax_rate,omitempty" binding:"omitempty,gte=0,lte=100"`
	IsFeatured       bool            `json:"is_featured"`
	IsDigital        bool            `json:"is_digital"`
	DownloadURL      string          `json:"download_url,omitempty" binding:"omitempty,url,max=1000"`
	MaxDownloads     int             `json:"max_downloads,omitempty" binding:"omitempty,gte=0,max=9999"`
	ExpiryDays       int             `json:"expiry_days,omitempty" binding:"omitempty,gte=0,max=3650"`
}

// UpdateProductRequest represents a product update request
type UpdateProductRequest struct {
	Name             *string          `json:"name,omitempty" binding:"omitempty,max=300"`
	Description      *string          `json:"description,omitempty" binding:"omitempty,max=2000"`
	ShortDescription *string          `json:"short_description,omitempty" binding:"omitempty,max=500"`
	CategoryID       *string          `json:"category_id,omitempty" binding:"omitempty,uuid"`
	Price            *decimal.Decimal `json:"price,omitempty" binding:"omitempty,gt=0"`
	Cost             *decimal.Decimal `json:"cost,omitempty" binding:"omitempty,gte=0"`
	Weight           *float64         `json:"weight,omitempty" binding:"omitempty,gte=0"`
	Length           *float64         `json:"length,omitempty" binding:"omitempty,gte=0"`
	Width            *float64         `json:"width,omitempty" binding:"omitempty,gte=0"`
	Height           *float64         `json:"height,omitempty" binding:"omitempty,gte=0"`
	Barcode          *string          `json:"barcode,omitempty" binding:"omitempty,max=50"`
	TrackInventory   *bool            `json:"track_inventory,omitempty"`
	MinStockLevel    *int             `json:"min_stock_level,omitempty" binding:"omitempty,gte=0"`
	MaxStockLevel    *int             `json:"max_stock_level,omitempty" binding:"omitempty,gte=0"`
	AllowBackorder   *bool            `json:"allow_backorder,omitempty"`
	RequiresShipping *bool            `json:"requires_shipping,omitempty"`
	Taxable          *bool            `json:"taxable,omitempty"`
	TaxRate          *decimal.Decimal `json:"tax_rate,omitempty" binding:"omitempty,gte=0,lte=100"`
	IsFeatured       *bool            `json:"is_featured,omitempty"`
	IsDigital        *bool            `json:"is_digital,omitempty"`
	DownloadURL      *string          `json:"download_url,omitempty" binding:"omitempty,url,max=1000"`
	MaxDownloads     *int             `json:"max_downloads,omitempty" binding:"omitempty,gte=0,max=9999"`
	ExpiryDays       *int             `json:"expiry_days,omitempty" binding:"omitempty,gte=0,max=3650"`
}

// ListProductsRequest represents a product list request
type ListProductsRequest struct {
	Search         string           `form:"search,omitempty"`
	CategoryID     string           `form:"category_id,omitempty" binding:"omitempty,uuid"`
	CategoryIDs    []string         `form:"category_ids,omitempty" binding:"omitempty,dive,uuid"`
	SKU            string           `form:"sku,omitempty"`
	MinPrice       *decimal.Decimal `form:"min_price,omitempty" binding:"omitempty,gte=0"`
	MaxPrice       *decimal.Decimal `form:"max_price,omitempty" binding:"omitempty,gte=0"`
	IsActive       *bool            `form:"is_active,omitempty"`
	IsFeatured     *bool            `form:"is_featured,omitempty"`
	IsDigital      *bool            `form:"is_digital,omitempty"`
	TrackInventory *bool            `form:"track_inventory,omitempty"`
	InStock        *bool            `form:"in_stock,omitempty"`
	LowStock       *bool            `form:"low_stock,omitempty"`
	CreatedAfter   string           `form:"created_after,omitempty" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	CreatedBefore  string           `form:"created_before,omitempty" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	Page           int              `form:"page,omitempty" binding:"omitempty,min=1"`
	Limit          int              `form:"limit,omitempty" binding:"omitempty,min=1,max=100"`
	SortBy         string           `form:"sort_by,omitempty" binding:"omitempty,oneof=name price created_at updated_at sku"`
	SortOrder      string           `form:"sort_order,omitempty" binding:"omitempty,oneof=asc desc"`
}

// SearchProductsRequest represents a product search request
type SearchProductsRequest struct {
	Query string `form:"q" binding:"required,min=1"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=50"`
}

// UpdatePriceRequest represents a price update request
type UpdatePriceRequest struct {
	Price decimal.Decimal  `json:"price" binding:"required,gt=0"`
	Cost  *decimal.Decimal `json:"cost,omitempty" binding:"omitempty,gte=0"`
}

// UpdateStockRequest represents a stock update request
type UpdateStockRequest struct {
	Quantity int `json:"quantity" binding:"required,gte=0"`
}

// AdjustStockRequest represents a stock adjustment request
type AdjustStockRequest struct {
	Adjustment int    `json:"adjustment" binding:"required"`
	Reason     string `json:"reason,omitempty" binding:"omitempty,max=500"`
}

// Product Operations Request DTOs

// BulkProductOperationRequest represents a bulk product operation request
type BulkProductOperationRequest struct {
	ProductIDs []string               `json:"product_ids" binding:"required,min=1,dive,uuid"`
	Operation  string                 `json:"operation" binding:"required,oneof=activate deactivate delete feature"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// BulkProductOperationResponse represents a bulk product operation response
type BulkProductOperationResponse struct {
	SuccessCount int                       `json:"success_count"`
	FailedCount  int                       `json:"failed_count"`
	TotalCount   int                       `json:"total_count"`
	Results      []*ProductOperationResult `json:"results,omitempty"`
}

// ProductOperationResult represents a single product operation result
type ProductOperationResult struct {
	Index   int              `json:"index"`
	ID      string           `json:"id,omitempty"`
	Success bool             `json:"success"`
	Data    *ProductResponse `json:"data,omitempty"`
	Error   string           `json:"error,omitempty"`
}

// Product Analytics Request DTOs

// GetProductStatsRequest represents a product stats request
type GetProductStatsRequest struct {
	CategoryID    string `form:"category_id,omitempty" binding:"omitempty,uuid"`
	IsActive      *bool  `form:"is_active,omitempty"`
	IsFeatured    *bool  `form:"is_featured,omitempty"`
	IsDigital     *bool  `form:"is_digital,omitempty"`
	CreatedAfter  string `form:"created_after,omitempty" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	CreatedBefore string `form:"created_before,omitempty" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

// Product Import/Export DTOs

// ProductImportRequest represents a product import request
type ProductImportRequest struct {
	Products  []CreateProductRequest `json:"products" binding:"required,min=1"`
	Strategy  string                 `json:"strategy" binding:"omitempty,oneof=create update merge"` // create, update, or merge
	UpdateSKU bool                   `json:"update_sku,omitempty"`                                   // Whether to update existing SKUs
}

// ProductImportResponse represents a product import response
type ProductImportResponse struct {
	TotalRows     int                       `json:"total_rows"`
	ProcessedRows int                       `json:"processed_rows"`
	CreatedCount  int                       `json:"created_count"`
	UpdatedCount  int                       `json:"updated_count"`
	FailedCount   int                       `json:"failed_count"`
	Errors        []ProductImportError      `json:"errors,omitempty"`
	Results       []*ProductOperationResult `json:"results,omitempty"`
}

// ProductImportError represents a product import error
type ProductImportError struct {
	Row    int    `json:"row"`
	Field  string `json:"field,omitempty"`
	Value  string `json:"value,omitempty"`
	Error  string `json:"error"`
	Reason string `json:"reason,omitempty"`
}

// ProductExportRequest represents a product export request
type ProductExportRequest struct {
	Format           string               `form:"format" binding:"omitempty,oneof=json csv xlsx"` // json, csv, or xlsx
	CategoryID       string               `form:"category_id,omitempty" binding:"omitempty,uuid"`
	IncludeInactive  bool                 `form:"include_inactive"`
	IncludeVariants  bool                 `form:"include_variants"`
	IncludeImages    bool                 `form:"include_images"`
	IncludeInventory bool                 `form:"include_inventory"`
	Fields           []string             `form:"fields,omitempty"` // Specific fields to export
	Filter           *ListProductsRequest `form:"-"`
}

// ProductExportResponse represents a product export response
type ProductExportResponse struct {
	Format      string      `json:"format"`
	FileName    string      `json:"file_name"`
	ContentType string      `json:"content_type"`
	Data        interface{} `json:"data"` // Can be []byte[], or structured data
	RecordCount int         `json:"record_count"`
	ExportedAt  string      `json:"exported_at"`
}

// Product Validation DTOs

// ValidateProductRequest represents a product validation request
type ValidateProductRequest struct {
	Product *CreateProductRequest `json:"product" binding:"required"`
}

// ProductValidationResponse represents product validation results
type ProductValidationResponse struct {
	IsValid  bool                     `json:"is_valid"`
	Warnings []string                 `json:"warnings,omitempty"`
	Errors   []ProductValidationError `json:"errors,omitempty"`
}

// ProductValidationError represents a product validation error
type ProductValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Product Inventory Request DTOs

// CheckAvailabilityRequest represents an availability check request
type CheckAvailabilityRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1"`
}

// LowStockProductsRequest represents a low stock products request
type LowStockProductsRequest struct {
	Threshold int `form:"threshold" binding:"omitempty,min=0"`     // Default: 10
	Limit     int `form:"limit" binding:"omitempty,min=1,max=100"` // Default: 20
}

// InventoryAdjustmentRequest represents an inventory adjustment request
type InventoryAdjustmentRequest struct {
	ProductID  string `json:"product_id" binding:"required,uuid"`
	Adjustment int    `json:"adjustment" binding:"required"`
	Reason     string `json:"reason" binding:"required,max=500"`
	AdjustBy   string `json:"adjust_by,omitempty" binding:"omitempty,uuid"` // User who made the adjustment
}

// ProductBulkInventoryAdjustmentRequest represents a bulk inventory adjustment request for products
type ProductBulkInventoryAdjustmentRequest struct {
	Adjustments []InventoryAdjustmentRequest `json:"adjustments" binding:"required,min=1,dive"`
}

// ProductBulkInventoryAdjustmentResponse represents a bulk inventory adjustment response for products
type ProductBulkInventoryAdjustmentResponse struct {
	SuccessCount int                          `json:"success_count"`
	FailedCount  int                          `json:"failed_count"`
	TotalCount   int                          `json:"total_count"`
	Results      []*InventoryAdjustmentResult `json:"results,omitempty"`
}

// InventoryAdjustmentResult represents a single inventory adjustment result
type InventoryAdjustmentResult struct {
	Index       int    `json:"index"`
	ProductID   string `json:"product_id,omitempty"`
	Success     bool   `json:"success"`
	StockBefore int    `json:"stock_before,omitempty"`
	StockAfter  int    `json:"stock_after,omitempty"`
	Error       string `json:"error,omitempty"`
}
