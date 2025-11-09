package dto

import (
	"time"

	"github.com/google/uuid"
)

// Category Request DTOs

// CreateCategoryRequest represents a category creation request
type CreateCategoryRequest struct {
	Name          string `json:"name" binding:"required,min=2,max=100"`
	Description   string `json:"description,omitempty" binding:"max=500"`
	ParentID      string `json:"parent_id,omitempty" binding:"omitempty,uuid"`
	IsActive      bool   `json:"is_active"`
	SortOrder     int    `json:"sort_order,omitempty" binding:"omitempty,gte=0"`
	SEOTitle      string `json:"seo_title,omitempty" binding:"omitempty,max=60"`
	SEODescription string `json:"seo_description,omitempty" binding:"omitempty,max=160"`
	SEOKeywords   string `json:"seo_keywords,omitempty" binding:"omitempty,max=255"`
}

// UpdateCategoryRequest represents a category update request
type UpdateCategoryRequest struct {
	Name          *string `json:"name,omitempty" binding:"omitempty,min=2,max=100"`
	Description   *string `json:"description,omitempty" binding:"omitempty,max=500"`
	IsActive      *bool   `json:"is_active,omitempty"`
	SortOrder     *int    `json:"sort_order,omitempty" binding:"omitempty,gte=0"`
	SEOTitle      *string `json:"seo_title,omitempty" binding:"omitempty,max=60"`
	SEODescription *string `json:"seo_description,omitempty" binding:"omitempty,max=160"`
	SEOKeywords   *string `json:"seo_keywords,omitempty" binding:"omitempty,max=255"`
}

// MoveCategoryRequest represents a category move request
type MoveCategoryRequest struct {
	ParentID *string `json:"parent_id,omitempty" binding:"omitempty,uuid"`
}

// ListCategoriesRequest represents a category list request
type ListCategoriesRequest struct {
	Search    string `form:"search,omitempty"`
	ParentID  string `form:"parent_id,omitempty" binding:"omitempty,uuid"`
	IsActive  *bool  `form:"is_active,omitempty"`
	Level     *int   `form:"level,omitempty"`
	Page      int    `form:"page,omitempty" binding:"omitempty,min=1"`
	Limit     int    `form:"limit,omitempty" binding:"omitempty,min=1,max=100"`
	SortBy    string `form:"sort_by,omitempty" binding:"omitempty,oneof=name sort_order created_at"`
	SortOrder string `form:"sort_order,omitempty" binding:"omitempty,oneof=asc desc"`
}

// Category Validation DTOs

// CategoryPathValidationRequest represents a category path validation request
type CategoryPathValidationRequest struct {
	Path     string `json:"path" binding:"required"`
	ParentID string `json:"parent_id,omitempty" binding:"omitempty,uuid"`
}

// Category Operations Response DTOs

// CategoryOperationResponse represents a category operation response
type CategoryOperationResponse struct {
	Success bool                `json:"success"`
	Message string              `json:"message"`
	Data    *CategoryResponse   `json:"data,omitempty"`
	Error   string              `json:"error,omitempty"`
}

// CategoryBulkOperationResponse represents a bulk category operation response
type CategoryBulkOperationResponse struct {
	SuccessCount int                         `json:"success_count"`
	FailedCount  int                         `json:"failed_count"`
	TotalCount   int                         `json:"total_count"`
	Results      []*CategoryOperationResult   `json:"results,omitempty"`
}

// CategoryOperationResult represents a single category operation result
type CategoryOperationResult struct {
	Index   int               `json:"index"`
	ID      string            `json:"id,omitempty"`
	Success bool              `json:"success"`
	Data    *CategoryResponse `json:"data,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// Category Reorder Request DTOs

// ReorderCategoriesRequest represents a category reorder request
type ReorderCategoriesRequest struct {
	CategoryReorders []CategoryReorderItem `json:"category_reorders" binding:"required,min=1"`
}

// CategoryReorderItem represents a single category reorder item
type CategoryReorderItem struct {
	ID        string `json:"id" binding:"required,uuid"`
	SortOrder int    `json:"sort_order" binding:"required,gte=0"`
}

// Category Analytics DTOs

// CategoryAnalyticsResponse represents category analytics
type CategoryAnalyticsResponse struct {
	CategoryID    uuid.UUID `json:"category_id"`
	CategoryName  string    `json:"category_name"`
	ProductCount  int       `json:"product_count"`
	ActiveProducts int      `json:"active_products"`
	TotalValue    string    `json:"total_value"` // Using string for decimal.Decimal compatibility
	AveragePrice  string    `json:"average_price"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CategoryHierarchyResponse represents category hierarchy information
type CategoryHierarchyResponse struct {
	Category      *CategoryResponse       `json:"category"`
	Parent        *CategoryResponse       `json:"parent,omitempty"`
	Children      []*CategoryResponse     `json:"children,omitempty"`
	Ancestors     []*CategoryResponse     `json:"ancestors,omitempty"`
	Descendants   []*CategoryResponse     `json:"descendants,omitempty"`
	ProductCount  int                     `json:"product_count"`
	Depth         int                     `json:"depth"`
}

// Category Import/Export DTOs

// CategoryImportRequest represents a category import request
type CategoryImportRequest struct {
	Categories []CreateCategoryRequest `json:"categories" binding:"required,min=1"`
	Strategy   string                  `json:"strategy" binding:"omitempty,oneof=create update merge"` // create, update, or merge
}

// CategoryImportResponse represents a category import response
type CategoryImportResponse struct {
	TotalRows     int                         `json:"total_rows"`
	ProcessedRows int                         `json:"processed_rows"`
	CreatedCount  int                         `json:"created_count"`
	UpdatedCount  int                         `json:"updated_count"`
	FailedCount   int                         `json:"failed_count"`
	Errors        []CategoryImportError       `json:"errors,omitempty"`
	Results       []*CategoryOperationResult  `json:"results,omitempty"`
}

// CategoryImportError represents an import error
type CategoryImportError struct {
	Row    int    `json:"row"`
	Field  string `json:"field,omitempty"`
	Value  string `json:"value,omitempty"`
	Error  string `json:"error"`
	Reason string `json:"reason,omitempty"`
}

// CategoryExportRequest represents a category export request
type CategoryExportRequest struct {
	Format      string `form:"format" binding:"omitempty,oneof=json csv xlsx"` // json, csv, or xlsx
	IncludeInactive bool `form:"include_inactive"`
	IncludeHierarchy bool `form:"include_hierarchy"`
	IncludeAnalytics bool `form:"include_analytics"`
	Filter     *ListCategoriesRequest `form:"-"`
}

// CategoryExportResponse represents a category export response
type CategoryExportResponse struct {
	Format      string              `json:"format"`
	FileName    string              `json:"file_name"`
	ContentType string              `json:"content_type"`
	Data        interface{}         `json:"data"` // Can be []byte[], or structured data
	RecordCount int                 `json:"record_count"`
	ExportedAt  time.Time           `json:"exported_at"`
}

// Category Search DTOs

// CategorySearchRequest represents a category search request
type CategorySearchRequest struct {
	Query        string `form:"q" binding:"required,min=1"`
	SearchIn     string `form:"search_in" binding:"omitempty,oneof=name description path"` // name, description, or path
	IncludeInactive bool `form:"include_inactive"`
	Limit        int    `form:"limit" binding:"omitempty,min=1,max=50"`
}

// CategorySearchResponse represents a category search response
type CategorySearchResponse struct {
	Categories []*CategorySearchResult `json:"categories"`
	Total      int                     `json:"total"`
	Query      string                  `json:"query"`
	Limit      int                     `json:"limit"`
}

// CategorySearchResult represents a single category search result
type CategorySearchResult struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Level       int       `json:"level"`
	IsActive    bool      `json:"is_active"`
	ProductCount int      `json:"product_count"`
	MatchType   string    `json:"match_type"` // name, description, or path
	Highlight   string    `json:"highlight,omitempty"` // Highlighted text snippet
}

// Category Validation DTOs

// CategoryValidationResponse represents category validation results
type CategoryValidationResponse struct {
	IsValid  bool                      `json:"is_valid"`
	Warnings []string                  `json:"warnings,omitempty"`
	Errors   []CategoryValidationError `json:"errors,omitempty"`
}

// CategoryValidationError represents a category validation error
type CategoryValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code"`
}