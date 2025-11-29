package dto

import (
	"time"

	"github.com/google/uuid"
)

// Image Response DTOs

// ImageInfo represents basic image information
type ImageInfo struct {
	Filename  string    `json:"filename"`
	URL       string    `json:"url"`
	Size      int64     `json:"size"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ImageDetail represents detailed image information
type ImageDetail struct {
	ID           uuid.UUID `json:"id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"original_name"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	Size         int64     `json:"size"`
	MimeType     string    `json:"mime_type"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	Directory    string    `json:"directory"`
	IsMain       bool      `json:"is_main"`
	SortOrder    int       `json:"sort_order"`
	Alt          string    `json:"alt,omitempty"`
	Title        string    `json:"title,omitempty"`
	Description  string    `json:"description,omitempty"`
	UploadedAt   time.Time `json:"uploaded_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProductImage represents a product image with variant association
type ProductImage struct {
	ID        uuid.UUID   `json:"id"`
	ProductID uuid.UUID   `json:"product_id"`
	VariantID *uuid.UUID  `json:"variant_id,omitempty"`
	ImageInfo ImageDetail `json:"image_info"`
	IsMain    bool        `json:"is_main"`
	SortOrder int         `json:"sort_order"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// ImageListResponse represents a paginated list of images
type ImageListResponse struct {
	Images     []*ImageDetail  `json:"images"`
	Pagination *PaginationInfo `json:"pagination"`
}

// Upload Result DTOs

// UploadResult represents a single file upload result
type UploadResult struct {
	Filename     string    `json:"filename"`
	OriginalName string    `json:"original_name"`
	Size         int64     `json:"size"`
	MimeType     string    `json:"mime_type"`
	URL          string    `json:"url"`
	Path         string    `json:"path"`
	Directory    string    `json:"directory"`
	UploadedAt   time.Time `json:"uploaded_at"`
}

// Batch Upload Response DTOs

// BatchImageUploadResponse represents a batch image upload response
type BatchImageUploadResponse struct {
	Results    []*UploadResult    `json:"results"`
	Errors     []BatchUploadError `json:"errors,omitempty"`
	TotalFiles int                `json:"total_files"`
	Successful int                `json:"successful"`
	Failed     int                `json:"failed"`
}

// BatchUploadError represents an error in batch upload
type BatchUploadError struct {
	Index    int    `json:"index"`
	Filename string `json:"filename"`
	Error    string `json:"error"`
}

// Image Management Request DTOs

// UpdateImageRequest represents an image update request
type UpdateImageRequest struct {
	Alt         string `json:"alt,omitempty" binding:"omitempty,max=255"`
	Title       string `json:"title,omitempty" binding:"omitempty,max=255"`
	Description string `json:"description,omitempty" binding:"omitempty,max=1000"`
	SortOrder   int    `json:"sort_order,omitempty" binding:"omitempty,gte=0"`
	IsMain      bool   `json:"is_main"`
}

// ReorderImagesRequest represents a request to reorder images
type ReorderImagesRequest struct {
	ImageOrders []ImageOrder `json:"image_orders" binding:"required,min=1"`
}

// ImageOrder represents a single image order
type ImageOrder struct {
	ID        string `json:"id" binding:"required,uuid"`
	SortOrder int    `json:"sort_order" binding:"required,gte=0"`
}

// SetMainImageRequest represents a request to set main image
type SetMainImageRequest struct {
	ImageID string `json:"image_id" binding:"required,uuid"`
}

// Image Processing DTOs

// ProcessImageRequest represents an image processing request
type ProcessImageRequest struct {
	Width      int    `json:"width,omitempty" binding:"omitempty,gte=1,max=4096"`
	Height     int    `json:"height,omitempty" binding:"omitempty,gte=1,max=4096"`
	Quality    int    `json:"quality,omitempty" binding:"omitempty,min=1,max=100"`
	Format     string `json:"format,omitempty" binding:"omitempty,oneof=jpeg png webp"`
	ResizeMode string `json:"resize_mode,omitempty" binding:"omitempty,oneof=fit fill crop"`
}

// ProcessImageResponse represents the result of image processing
type ProcessImageResponse struct {
	OriginalURL    string  `json:"original_url"`
	ProcessedURL   string  `json:"processed_url"`
	ThumbnailURL   string  `json:"thumbnail_url,omitempty"`
	OriginalSize   int64   `json:"original_size"`
	ProcessedSize  int64   `json:"processed_size,omitempty"`
	ThumbnailSize  int64   `json:"thumbnail_size,omitempty"`
	ProcessingTime float64 `json:"processing_time"`
}

// Image Analytics DTOs

// ImageStatsResponse represents image statistics
type ImageStatsResponse struct {
	TotalImages      int            `json:"total_images"`
	TotalSize        int64          `json:"total_size"`
	ImagesByType     map[string]int `json:"images_by_type"`
	UploadsToday     int            `json:"uploads_today"`
	UploadsThisWeek  int            `json:"uploads_this_week"`
	UploadsThisMonth int            `json:"uploads_this_month"`
	AverageSize      int64          `json:"average_size"`
	LargestSize      int64          `json:"largest_size"`
}

// ImageStorageInfo represents storage information
type ImageStorageInfo struct {
	UsedSpace       int64   `json:"used_space"`
	TotalSpace      int64   `json:"total_space"`
	AvailableSpace  int64   `json:"available_space"`
	UsagePercentage float64 `json:"usage_percentage"`
}

// Image Search and Filter DTOs

// ListImagesRequest represents an image list request
type ListImagesRequest struct {
	Search         string `form:"search,omitempty"`
	Directory      string `form:"directory,omitempty"`
	MimeType       string `form:"mime_type,omitempty"`
	MinSize        int64  `form:"min_size,omitempty"`
	MaxSize        int64  `form:"max_size,omitempty"`
	UploadedAfter  string `form:"uploaded_after,omitempty" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	UploadedBefore string `form:"uploaded_before,omitempty" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	Page           int    `form:"page,omitempty" binding:"omitempty,min=1"`
	Limit          int    `form:"limit,omitempty" binding:"omitempty,min=1,max=100"`
	SortBy         string `form:"sort_by,omitempty" binding:"omitempty,oneof=filename size uploaded_at"`
	SortOrder      string `form:"sort_order,omitempty" binding:"omitempty,oneof=asc desc"`
}

// SearchImagesRequest represents an image search request
type SearchImagesRequest struct {
	Query    string `form:"q" binding:"required,min=1"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=50"`
	SearchIn string `form:"search_in,omitempty" binding:"omitempty,oneof=filename alt title description"`
}

// SearchImagesResponse represents image search results
type SearchImagesResponse struct {
	Images []*ImageDetail `json:"images"`
	Total  int            `json:"total"`
	Query  string         `json:"query"`
	Limit  int            `json:"limit"`
}

// Image Validation DTOs

// ValidateImageRequest represents an image validation request
type ValidateImageRequest struct {
	ImageURL string `json:"image_url" binding:"required,url"`
}

// ImageValidationResponse represents image validation results
type ImageValidationResponse struct {
	IsValid   bool                   `json:"is_valid"`
	ImageInfo *ImageDetail           `json:"image_info,omitempty"`
	Warnings  []string               `json:"warnings,omitempty"`
	Errors    []ImageValidationError `json:"errors,omitempty"`
}

// ImageValidationError represents an image validation error
type ImageValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Common Response DTOs

// MessageResponse represents a simple message response
type MessageResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// UploadProgress represents upload progress information
type UploadProgress struct {
	ID            string  `json:"id"`
	Filename      string  `json:"filename"`
	Progress      float64 `json:"progress"` // 0-100
	Status        string  `json:"status"`   // uploading, processing, completed, failed
	Speed         int64   `json:"speed"`    // bytes per second
	BytesTotal    int64   `json:"bytes_total"`
	BytesUploaded int64   `json:"bytes_uploaded"`
	ETA           int     `json:"eta,omitempty"` // estimated time remaining in seconds
	Error         string  `json:"error,omitempty"`
}

// ImageImportResponse represents an image import response
type ImageImportResponse struct {
	ImportID    string             `json:"import_id"`
	TotalFiles  int                `json:"total_files"`
	Processed   int                `json:"processed"`
	Successful  int                `json:"successful"`
	Failed      int                `json:"failed"`
	Results     []*ImageDetail     `json:"results,omitempty"`
	Errors      []ImageImportError `json:"errors,omitempty"`
	StartedAt   time.Time          `json:"started_at"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
}

// ImageImportError represents an import error
type ImageImportError struct {
	Index     int    `json:"index"`
	Filename  string `json:"filename"`
	Error     string `json:"error"`
	Retryable bool   `json:"retryable"`
}

// Image Export DTOs

// ImageExportRequest represents an image export request
type ImageExportRequest struct {
	Format      string `form:"format" binding:"omitempty,oneof=json csv zip"`
	Directory   string `form:"directory,omitempty"`
	Compress    bool   `form:"compress"`
	IncludeMeta bool   `form:"include_metadata"`
}

// ImageExportResponse represents an image export response
type ImageExportResponse struct {
	ExportID    string    `json:"export_id"`
	Format      string    `json:"format"`
	FileName    string    `json:"file_name"`
	DownloadURL string    `json:"download_url"`
	FileSize    int64     `json:"file_size"`
	ImagesCount int       `json:"images_count"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Image Cleanup DTOs

// CleanupRequest represents a cleanup request
type CleanupRequest struct {
	OlderThan     string `form:"older_than" binding:"omitempty,datetime=2006-01-02"`
	DryRun        bool   `form:"dry_run"`
	DeleteOrphans bool   `form:"delete_orphans"`
}

// CleanupResponse represents a cleanup response
type CleanupResponse struct {
	FilesDeleted   int       `json:"files_deleted"`
	SpaceFreed     int64     `json:"space_freed"`
	OrphansDeleted int       `json:"orphans_deleted"`
	DryRun         bool      `json:"dry_run"`
	ProcessedAt    time.Time `json:"processed_at"`
}

// CDN/CDN Integration DTOs

// CDNConfig represents CDN configuration
type CDNConfig struct {
	Enabled   bool   `json:"enabled"`
	Provider  string `json:"provider"`
	BaseURL   string `json:"base_url"`
	SecretKey string `json:"secret_key,omitempty"`
	Region    string `json:"region,omitempty"`
	Bucket    string `json:"bucket,omitempty"`
}

// CDNUploadResponse represents a CDN upload response
type CDNUploadResponse struct {
	URL          string    `json:"url"`
	CDNURL       string    `json:"cdn_url"`
	PublicID     string    `json:"public_id"`
	ResourceType string    `json:"resource_type"`
	CreatedAt    time.Time `json:"created_at"`
	ETag         string    `json:"etag,omitempty"`
}
