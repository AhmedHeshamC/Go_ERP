package dto

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Product Variant Request DTOs

// CreateProductVariantRequest represents a product variant creation request
type CreateProductVariantRequest struct {
	ProductID      string                `json:"product_id" binding:"required,uuid"`
	SKU            string                `json:"sku,omitempty" binding:"omitempty,max=100"`
	Name           string                `json:"name" binding:"required,max=300"`
	Price          decimal.Decimal       `json:"price" binding:"required,gt=0"`
	Cost           decimal.Decimal       `json:"cost,omitempty" binding:"omitempty,gte=0"`
	Weight         float64               `json:"weight,omitempty" binding:"omitempty,gte=0"`
	Barcode        string                `json:"barcode,omitempty" binding:"omitempty,max=50"`
	TrackInventory bool                  `json:"track_inventory"`
	StockQuantity  int                   `json:"stock_quantity,omitempty" binding:"omitempty,gte=0"`
	MinStockLevel  int                   `json:"min_stock_level,omitempty" binding:"omitempty,gte=0"`
	MaxStockLevel  int                   `json:"max_stock_level,omitempty" binding:"omitempty,gte=0"`
	AllowBackorder bool                  `json:"allow_backorder"`
	IsActive       bool                  `json:"is_active"`
	Attributes     []VariantAttributeReq `json:"attributes,omitempty"`
	Images         []VariantImageReq     `json:"images,omitempty"`
}

// UpdateProductVariantRequest represents a product variant update request
type UpdateProductVariantRequest struct {
	Name           *string               `json:"name,omitempty" binding:"omitempty,max=300"`
	Price          *decimal.Decimal      `json:"price,omitempty" binding:"omitempty,gt=0"`
	Cost           *decimal.Decimal      `json:"cost,omitempty" binding:"omitempty,gte=0"`
	Weight         *float64              `json:"weight,omitempty" binding:"omitempty,gte=0"`
	Barcode        *string               `json:"barcode,omitempty" binding:"omitempty,max=50"`
	TrackInventory *bool                 `json:"track_inventory,omitempty"`
	MinStockLevel  *int                  `json:"min_stock_level,omitempty" binding:"omitempty,gte=0"`
	MaxStockLevel  *int                  `json:"max_stock_level,omitempty" binding:"omitempty,gte=0"`
	AllowBackorder *bool                 `json:"allow_backorder,omitempty"`
	IsActive       *bool                 `json:"is_active,omitempty"`
	Attributes     []VariantAttributeReq `json:"attributes,omitempty"`
}

// ListProductVariantsRequest represents a product variant list request
type ListProductVariantsRequest struct {
	Search    string           `form:"search,omitempty"`
	MinPrice  *decimal.Decimal `form:"min_price,omitempty" binding:"omitempty,gte=0"`
	MaxPrice  *decimal.Decimal `form:"max_price,omitempty" binding:"omitempty,gte=0"`
	IsActive  *bool            `form:"is_active,omitempty"`
	InStock   *bool            `form:"in_stock,omitempty"`
	Page      int              `form:"page,omitempty" binding:"omitempty,min=1"`
	Limit     int              `form:"limit,omitempty" binding:"omitempty,min=1,max=100"`
	SortBy    string           `form:"sort_by,omitempty" binding:"omitempty,oneof=name price created_at"`
	SortOrder string           `form:"sort_order,omitempty" binding:"omitempty,oneof=asc desc"`
}

// Variant Attribute Request DTOs

// VariantAttributeReq represents a variant attribute request
type VariantAttributeReq struct {
	Name  string `json:"name" binding:"required,max=100"`
	Value string `json:"value" binding:"required,max=255"`
}

// VariantImageReq represents a variant image request
type VariantImageReq struct {
	URL       string `json:"url" binding:"required,url,max=1000"`
	Alt       string `json:"alt,omitempty" binding:"omitempty,max=255"`
	SortOrder int    `json:"sort_order" binding:"omitempty,gte=0"`
	IsMain    bool   `json:"is_main"`
}

// UpdateVariantImagesRequest represents a variant images update request
type UpdateVariantImagesRequest struct {
	Images []VariantImageReq `json:"images" binding:"required,min=1,dive"`
}

// Variant Inventory Request DTOs

// UpdateVariantStockRequest represents a variant stock update request
type UpdateVariantStockRequest struct {
	Quantity int `json:"quantity" binding:"required,gte=0"`
}

// AdjustVariantStockRequest represents a variant stock adjustment request
type AdjustVariantStockRequest struct {
	Adjustment int    `json:"adjustment" binding:"required"`
	Reason     string `json:"reason,omitempty" binding:"omitempty,max=500"`
}

// CheckVariantAvailabilityRequest represents a variant availability check request
type CheckVariantAvailabilityRequest struct {
	VariantID string `json:"variant_id" binding:"required,uuid"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
}

// Bulk Variant Operations Request DTOs

// BulkVariantOperationRequest represents a bulk variant operation request
type BulkVariantOperationRequest struct {
	VariantIDs []string               `json:"variant_ids" binding:"required,min=1,dive,uuid"`
	Operation  string                 `json:"operation" binding:"required,oneof=activate deactivate delete"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// BulkVariantOperationResponse represents a bulk variant operation response
type BulkVariantOperationResponse struct {
	SuccessCount int                       `json:"success_count"`
	FailedCount  int                       `json:"failed_count"`
	TotalCount   int                       `json:"total_count"`
	Results      []*VariantOperationResult `json:"results,omitempty"`
}

// VariantOperationResult represents a single variant operation result
type VariantOperationResult struct {
	Index   int                     `json:"index"`
	ID      string                  `json:"id,omitempty"`
	Success bool                    `json:"success"`
	Data    *ProductVariantResponse `json:"data,omitempty"`
	Error   string                  `json:"error,omitempty"`
}

// Variant Import/Export DTOs

// VariantImportRequest represents a variant import request
type VariantImportRequest struct {
	ProductID string                        `json:"product_id" binding:"required,uuid"`
	Variants  []CreateProductVariantRequest `json:"variants" binding:"required,min=1"`
	Strategy  string                        `json:"strategy" binding:"omitempty,oneof=create update merge"` // create, update, or merge
	UpdateSKU bool                          `json:"update_sku,omitempty"`                                   // Whether to update existing SKUs
}

// VariantImportResponse represents a variant import response
type VariantImportResponse struct {
	TotalRows     int                       `json:"total_rows"`
	ProcessedRows int                       `json:"processed_rows"`
	CreatedCount  int                       `json:"created_count"`
	UpdatedCount  int                       `json:"updated_count"`
	FailedCount   int                       `json:"failed_count"`
	Errors        []VariantImportError      `json:"errors,omitempty"`
	Results       []*VariantOperationResult `json:"results,omitempty"`
}

// VariantImportError represents a variant import error
type VariantImportError struct {
	Row    int    `json:"row"`
	Field  string `json:"field,omitempty"`
	Value  string `json:"value,omitempty"`
	Error  string `json:"error"`
	Reason string `json:"reason,omitempty"`
}

// VariantExportRequest represents a variant export request
type VariantExportRequest struct {
	ProductID        string   `form:"product_id" binding:"required,uuid"`
	Format           string   `form:"format" binding:"omitempty,oneof=json csv xlsx"` // json, csv, or xlsx
	IncludeInactive  bool     `form:"include_inactive"`
	IncludeImages    bool     `form:"include_images"`
	IncludeInventory bool     `form:"include_inventory"`
	Fields           []string `form:"fields,omitempty"` // Specific fields to export
}

// VariantExportResponse represents a variant export response
type VariantExportResponse struct {
	Format      string      `json:"format"`
	FileName    string      `json:"file_name"`
	ContentType string      `json:"content_type"`
	Data        interface{} `json:"data"` // Can be []byte[], or structured data
	RecordCount int         `json:"record_count"`
	ExportedAt  string      `json:"exported_at"`
}

// Variant Validation DTOs

// ValidateVariantRequest represents a variant validation request
type ValidateVariantRequest struct {
	Variant *CreateProductVariantRequest `json:"variant" binding:"required"`
}

// VariantValidationResponse represents variant validation results
type VariantValidationResponse struct {
	IsValid  bool                     `json:"is_valid"`
	Warnings []string                 `json:"warnings,omitempty"`
	Errors   []VariantValidationError `json:"errors,omitempty"`
}

// VariantValidationError represents a variant validation error
type VariantValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Variant Comparison DTOs

// CompareVariantsRequest represents a variant comparison request
type CompareVariantsRequest struct {
	VariantIDs []string `json:"variant_ids" binding:"required,min=2,max=5,dive,uuid"`
	Fields     []string `json:"fields,omitempty"` // Specific fields to compare
}

// CompareVariantsResponse represents a variant comparison response
type CompareVariantsResponse struct {
	Variants      []*ProductVariantResponse `json:"variants"`
	Comparisons   []VariantComparison       `json:"comparisons"`
	TotalVariants int                       `json:"total_variants"`
}

// VariantComparison represents a comparison between variants
type VariantComparison struct {
	Field       string   `json:"field"`
	Values      []string `json:"values"`
	Differences []string `json:"differences,omitempty"`
	Same        bool     `json:"same"`
}

// Variant Search DTOs

// SearchVariantsRequest represents a variant search request
type SearchVariantsRequest struct {
	Query           string `form:"q" binding:"required,min=1"`
	ProductID       string `form:"product_id,omitempty" binding:"omitempty,uuid"`
	SearchIn        string `form:"search_in" binding:"omitempty,oneof=name sku"` // name or sku
	IncludeInactive bool   `form:"include_inactive"`
	Limit           int    `form:"limit" binding:"omitempty,min=1,max=50"`
}

// SearchVariantsResponse represents a variant search response
type SearchVariantsResponse struct {
	Variants  []*ProductVariantResponse `json:"variants"`
	Total     int                       `json:"total"`
	Query     string                    `json:"query"`
	ProductID string                    `json:"product_id,omitempty"`
	Limit     int                       `json:"limit"`
}

// Variant Analytics DTOs

// VariantAnalyticsRequest represents a variant analytics request
type VariantAnalyticsRequest struct {
	ProductID string `form:"product_id" binding:"required,uuid"`
	Analytics string `form:"analytics" binding:"omitempty,oneof=inventory sales performance"` // inventory, sales, or performance
	DateRange string `form:"date_range,omitempty" binding:"omitempty,oneof=7d 30d 90d 1y"`    // predefined date ranges
	StartDate string `form:"start_date,omitempty" binding:"omitempty,datetime=2006-01-02"`
	EndDate   string `form:"end_date,omitempty" binding:"omitempty,datetime=2006-01-02"`
}

// VariantAnalyticsResponse represents variant analytics
type VariantAnalyticsResponse struct {
	VariantID    uuid.UUID `json:"variant_id"`
	VariantName  string    `json:"variant_name"`
	VariantSKU   string    `json:"variant_sku"`
	ProductID    uuid.UUID `json:"product_id"`
	ProductName  string    `json:"product_name"`
	StockLevel   int       `json:"stock_level"`
	SalesCount   int       `json:"sales_count"`
	Revenue      string    `json:"revenue"`       // Using string for decimal.Decimal compatibility
	Profit       string    `json:"profit"`        // Using string for decimal.Decimal compatibility
	ProfitMargin string    `json:"profit_margin"` // Using string for decimal.Decimal compatibility
	LastSold     string    `json:"last_sold,omitempty"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
}
