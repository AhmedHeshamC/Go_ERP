package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"erpgo/internal/application/services/file"
	"erpgo/internal/interfaces/http/dto"
)

// ImageHandler handles image upload and management HTTP requests
type ImageHandler struct {
	fileService file.Service
	logger      zerolog.Logger
}

// NewImageHandler creates a new image handler
func NewImageHandler(fileService file.Service, logger zerolog.Logger) *ImageHandler {
	return &ImageHandler{
		fileService: fileService,
		logger:      logger,
	}
}

// UploadProductImage uploads an image for a product
// @Summary Upload product image
// @Description Upload an image for a specific product with automatic processing
// @Tags images
// @Accept multipart/form-data
// @Produce json
// @Param product_id path string true "Product ID"
// @Param image formData file true "Image file"
// @Success 201 {object} dto.UploadResult
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 413 {object} dto.ErrorResponse
// @Failure 415 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{product_id}/images [post]
func (h *ImageHandler) UploadProductImage(c *gin.Context) {
	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	// Validate product ID format
	if _, err := uuid.Parse(productID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid product ID format",
			Details: err.Error(),
		})
		return
	}

	// Get uploaded file
	file, err := c.FormFile("image")
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get uploaded file")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "No file uploaded",
			Details: "Please select an image file to upload",
		})
		return
	}

	// Validate file size
	if file.Size > 10*1024*1024 { // 10MB limit
		c.JSON(http.StatusRequestEntityTooLarge, dto.ErrorResponse{
			Error:   "File too large",
			Details: "Maximum file size is 10MB",
		})
		return
	}

	// Upload file
	productUUID := uuid.MustParse(productID)
	result, err := h.fileService.UploadProductImage(c, file, productUUID)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", productID).Str("filename", file.Filename).Msg("Failed to upload product image")
		handleImageError(c, err)
		return
	}

	h.logger.Info().
		Str("product_id", productID).
		Str("filename", result.Filename).
		Str("url", result.URL).
		Msg("Product image uploaded successfully")

	c.JSON(http.StatusCreated, result)
}

// UploadProductVariantImage uploads an image for a product variant
// @Summary Upload product variant image
// @Description Upload an image for a specific product variant with automatic processing
// @Tags images
// @Accept multipart/form-data
// @Produce json
// @Param product_id path string true "Product ID"
// @Param variant_id path string true "Variant ID"
// @Param image formData file true "Image file"
// @Param is_main formData bool false "Set as main image"
// @Success 201 {object} dto.UploadResult
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 413 {object} dto.ErrorResponse
// @Failure 415 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{product_id}/variants/{variant_id}/images [post]
func (h *ImageHandler) UploadProductVariantImage(c *gin.Context) {
	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	variantID := c.Param("variant_id")
	if variantID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Variant ID is required",
		})
		return
	}

	// Validate IDs format
	if _, err := uuid.Parse(productID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid product ID format",
			Details: err.Error(),
		})
		return
	}

	if _, err := uuid.Parse(variantID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid variant ID format",
			Details: err.Error(),
		})
		return
	}

	// Get uploaded file
	file, err := c.FormFile("image")
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get uploaded file")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "No file uploaded",
			Details: "Please select an image file to upload",
		})
		return
	}

	// Check if this should be set as main image
	isMainStr := c.PostForm("is_main")
	isMain := isMainStr == "true"

	// Validate file size
	if file.Size > 10*1024*1024 { // 10MB limit
		c.JSON(http.StatusRequestEntityTooLarge, dto.ErrorResponse{
			Error:   "File too large",
			Details: "Maximum file size is 10MB",
		})
		return
	}

	// Upload file
	productUUID := uuid.MustParse(productID)
	variantUUID := uuid.MustParse(variantID)
	result, err := h.fileService.UploadProductVariantImage(c, file, productUUID, variantUUID)
	if err != nil {
		h.logger.Error().Err(err).
			Str("product_id", productID).
			Str("variant_id", variantID).
			Str("filename", file.Filename).
			Msg("Failed to upload product variant image")
		handleImageError(c, err)
		return
	}

	// TODO: Update variant to set as main image if requested
	if isMain {
		h.logger.Info().
			Str("variant_id", variantID).
			Str("image_url", result.URL).
			Msg("Image set as main (implementation pending)")
	}

	h.logger.Info().
		Str("product_id", productID).
		Str("variant_id", variantID).
		Str("filename", result.Filename).
		Str("url", result.URL).
		Bool("is_main", isMain).
		Msg("Product variant image uploaded successfully")

	c.JSON(http.StatusCreated, result)
}

// UploadMultipleImages uploads multiple images for a product
// @Summary Upload multiple product images
// @Description Upload multiple images for a specific product with batch processing
// @Tags images
// @Accept multipart/form-data
// @Produce json
// @Param product_id path string true "Product ID"
// @Param images formData file true "Image files"
// @Success 201 {array} dto.UploadResult
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 413 {object} dto.ErrorResponse
// @Failure 415 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{product_id}/images/batch [post]
func (h *ImageHandler) UploadMultipleImages(c *gin.Context) {
	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	// Validate product ID format
	if _, err := uuid.Parse(productID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid product ID format",
			Details: err.Error(),
		})
		return
	}

	// Get all uploaded files
	form, err := c.MultipartForm()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse multipart form")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Failed to parse uploaded files",
			Details: err.Error(),
		})
		return
	}

	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "No files uploaded",
			Details: "Please select at least one image file to upload",
		})
		return
	}

	// Limit number of files
	if len(files) > 10 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Too many files",
			Details: "Maximum 10 files can be uploaded at once",
		})
		return
	}

	productUUID := uuid.MustParse(productID)
	results := make([]*dto.UploadResult, 0, len(files))
	errors := make([]dto.BatchUploadError, 0)

	// Process each file
	for i, file := range files {
		// Validate file size
		if file.Size > 10*1024*1024 { // 10MB limit
			errors = append(errors, dto.BatchUploadError{
				Index:    i,
				Filename: file.Filename,
				Error:    "File too large (maximum 10MB)",
			})
			continue
		}

		// Upload file
		result, err := h.fileService.UploadProductImage(c, file, productUUID)
		if err != nil {
			h.logger.Error().Err(err).
				Str("product_id", productID).
				Str("filename", file.Filename).
				Int("index", i).
				Msg("Failed to upload product image in batch")
			errors = append(errors, dto.BatchUploadError{
				Index:    i,
				Filename: file.Filename,
				Error:    err.Error(),
			})
			continue
		}

		results = append(results, &dto.UploadResult{
			Filename:     result.Filename,
			OriginalName: result.OriginalName,
			Size:         result.Size,
			MimeType:     result.MimeType,
			URL:          result.URL,
			Path:         result.Path,
			Directory:    result.Directory,
			UploadedAt:   result.UploadedAt,
		})
	}

	// Log batch upload results
	h.logger.Info().
		Str("product_id", productID).
		Int("total_files", len(files)).
		Int("successful", len(results)).
		Int("failed", len(errors)).
		Msg("Batch image upload completed")

	response := dto.BatchImageUploadResponse{
		Results:    results,
		Errors:     errors,
		TotalFiles: len(files),
		Successful: len(results),
		Failed:     len(errors),
	}

	c.JSON(http.StatusCreated, response)
}

// GetImage retrieves an image by filename
// @Summary Get image
// @Description Retrieve an image file by filename
// @Tags images
// @Accept json
// @Produce image/*
// @Param filename path string true "Image filename"
// @Success 200 {file} binary
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/images/{filename} [get]
func (h *ImageHandler) GetImage(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Filename is required",
		})
		return
	}

	// Get file info
	file, err := h.fileService.GetFile(c, filename)
	if err != nil {
		h.logger.Error().Err(err).Str("filename", filename).Msg("Failed to get image file")
		handleImageError(c, err)
		return
	}

	// Check if file exists and is accessible
	if _, err := os.Stat(file.Path); err != nil {
		h.logger.Error().Err(err).Str("path", file.Path).Msg("File not found on disk")
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error: "Image file not found",
		})
		return
	}

	// Serve the file
	c.File(file.Path)
}

// DeleteImage deletes an image file
// @Summary Delete image
// @Description Delete an image file by filename
// @Tags images
// @Accept json
// @Produce json
// @Param filename path string true "Image filename"
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/images/{filename} [delete]
func (h *ImageHandler) DeleteImage(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Filename is required",
		})
		return
	}

	// Check if file exists
	_, err := h.fileService.GetFile(c, filename)
	if err != nil {
		h.logger.Error().Err(err).Str("filename", filename).Msg("Failed to get image file")
		handleImageError(c, err)
		return
	}

	// Delete file
	if err := h.fileService.DeleteFile(c, filename); err != nil {
		h.logger.Error().Err(err).Str("filename", filename).Msg("Failed to delete image file")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to delete image",
			Details: err.Error(),
		})
		return
	}

	h.logger.Info().Str("filename", filename).Msg("Image deleted successfully")

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Image deleted successfully",
	})
}

// GetProductImages retrieves all images for a product
// @Summary Get product images
// @Description Get a list of all images uploaded for a specific product
// @Tags images
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {array} dto.ImageInfo
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/products/{product_id}/images [get]
func (h *ImageHandler) GetProductImages(c *gin.Context) {
	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Product ID is required",
		})
		return
	}

	// Validate product ID format
	if _, err := uuid.Parse(productID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid product ID format",
			Details: err.Error(),
		})
		return
	}

	// This would typically fetch from a database
	// For now, we'll scan the upload directory
	productUUID := uuid.MustParse(productID)
	productDir := filepath.Join("uploads", "products", productUUID.String())

	images, err := h.scanDirectoryForImages(productDir)
	if err != nil {
		h.logger.Error().Err(err).Str("product_id", productID).Str("directory", productDir).Msg("Failed to scan product images directory")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to retrieve product images",
			Details: err.Error(),
		})
		return
	}

	h.logger.Info().
		Str("product_id", productID).
		Int("image_count", len(images)).
		Msg("Retrieved product images")

	c.JSON(http.StatusOK, images)
}

// Admin Operations (placeholder implementations)

// ImportImages imports images from URL or file
func (h *ImageHandler) ImportImages(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Image import is not yet implemented",
	})
}

// ExportImages exports images to various formats
func (h *ImageHandler) ExportImages(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Image export is not yet implemented",
	})
}

// CleanupImages cleans up old/orphaned images
func (h *ImageHandler) CleanupImages(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Image cleanup is not yet implemented",
	})
}

// GetImageStats retrieves image statistics
func (h *ImageHandler) GetImageStats(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Image statistics are not yet implemented",
	})
}

// GetStorageInfo retrieves storage information
func (h *ImageHandler) GetStorageInfo(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Storage information is not yet implemented",
	})
}

// CDN Operations (placeholder implementations)

// SyncCDN syncs images to CDN
func (h *ImageHandler) SyncCDN(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "CDN synchronization is not yet implemented",
	})
}

// GetCDNConfig retrieves CDN configuration
func (h *ImageHandler) GetCDNConfig(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "CDN configuration retrieval is not yet implemented",
	})
}

// UpdateCDNConfig updates CDN configuration
func (h *ImageHandler) UpdateCDNConfig(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "CDN configuration update is not yet implemented",
	})
}

// Image Processing Operations (placeholder implementations)

// ProcessImages processes multiple images
func (h *ImageHandler) ProcessImages(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Batch image processing is not yet implemented",
	})
}

// OptimizeImages optimizes multiple images
func (h *ImageHandler) OptimizeImages(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse{
		Error: "Batch image optimization is not yet implemented",
	})
}

// Helper Methods

// scanDirectoryForImages scans a directory for image files
func (h *ImageHandler) scanDirectoryForImages(directory string) ([]*dto.ImageInfo, error) {
	var images []*dto.ImageInfo

	// Check if directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return images, nil // Return empty list if directory doesn't exist
	}

	// Read directory
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		// Check if it's an image file
		if h.isImageFile(entry.Name()) {
			info, err := entry.Info()
			if err != nil {
				h.logger.Warn().Err(err).Str("filename", entry.Name()).Msg("Failed to get file info")
				continue
			}

			// Generate URL
			filename := filepath.Join("products", filepath.Base(directory), entry.Name())
			url := fmt.Sprintf("/uploads/%s", filename)

			images = append(images, &dto.ImageInfo{
				Filename:  entry.Name(),
				URL:       url,
				Size:      info.Size(),
				UpdatedAt: info.ModTime(),
			})
		}
	}

	return images, nil
}

// isImageFile checks if a filename has an image extension
func (h *ImageHandler) isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	imageExts := []string{".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp", ".svg"}

	for _, imageExt := range imageExts {
		if ext == imageExt {
			return true
		}
	}
	return false
}

// handleImageError handles image service errors
func handleImageError(c *gin.Context, err error) {
	switch {
	case strings.Contains(err.Error(), "file not found"):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Image not found",
			Details: err.Error(),
		})
	case strings.Contains(err.Error(), "file validation failed"):
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid file",
			Details: err.Error(),
		})
	case strings.Contains(err.Error(), "file size"):
		c.JSON(http.StatusRequestEntityTooLarge, dto.ErrorResponse{
			Error:   "File too large",
			Details: err.Error(),
		})
	case strings.Contains(err.Error(), "file type") || strings.Contains(err.Error(), "not allowed"):
		c.JSON(http.StatusUnsupportedMediaType, dto.ErrorResponse{
			Error:   "Unsupported file type",
			Details: err.Error(),
		})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Details: err.Error(),
		})
	}
}
