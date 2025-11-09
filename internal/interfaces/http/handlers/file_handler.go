package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"erpgo/internal/infrastructure/storage"
	"erpgo/internal/interfaces/http/dto"
)

// FileHandler handles file HTTP requests
type FileHandler struct {
	storageProvider storage.StorageProvider
	logger          zerolog.Logger
	config          *FileHandlerConfig
}

// FileHandlerConfig contains configuration for the file handler
type FileHandlerConfig struct {
	MaxFileSize        int64    `json:"max_file_size" yaml:"max_file_size"`
	AllowedExtensions  []string `json:"allowed_extensions" yaml:"allowed_extensions"`
	AllowedMimeTypes   []string `json:"allowed_mime_types" yaml:"allowed_mime_types"`
	EnablePresignedURL bool     `json:"enable_presigned_url" yaml:"enable_presigned_url"`
	PresignedExpiry    int      `json:"presigned_expiry" yaml:"presigned_expiry"` // in minutes
	EnableThumbnails   bool     `json:"enable_thumbnails" yaml:"enable_thumbnails"`
	ThumbnailSize      int      `json:"thumbnail_size" yaml:"thumbnail_size"`
}

// NewFileHandler creates a new file handler
func NewFileHandler(storageProvider storage.StorageProvider, logger zerolog.Logger, config *FileHandlerConfig) *FileHandler {
	// Set default configuration if not provided
	if config == nil {
		config = &FileHandlerConfig{
			MaxFileSize:        10 * 1024 * 1024, // 10MB
			AllowedExtensions:  []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".pdf", ".doc", ".docx", ".txt", ".csv", ".xlsx"},
			AllowedMimeTypes:   []string{"image/jpeg", "image/png", "image/gif", "image/webp", "application/pdf", "text/plain", "text/csv"},
			EnablePresignedURL: true,
			PresignedExpiry:    60, // 1 hour
			EnableThumbnails:   true,
			ThumbnailSize:      200,
		}
	}

	return &FileHandler{
		storageProvider: storageProvider,
		logger:          logger,
		config:          config,
	}
}

// UploadFile uploads a generic file
// @Summary Upload file
// @Description Upload a file to storage with validation and automatic organization
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Param directory formData string false "Directory for file organization"
// @Param public formData bool false "Make file publicly accessible"
// @Success 201 {object} dto.UploadResult
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 413 {object} dto.ErrorResponse
// @Failure 415 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/files [post]
func (h *FileHandler) UploadFile(c *gin.Context) {
	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get uploaded file")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "No file uploaded",
			Details: "Please select a file to upload",
		})
		return
	}

	// Validate file
	if err := h.validateFile(file); err != nil {
		h.logger.Error().Err(err).Str("filename", file.Filename).Msg("File validation failed")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "File validation failed",
			Details: err.Error(),
		})
		return
	}

	// Get directory from form or use default
	directory := c.PostForm("directory")
	if directory == "" {
		directory = "uploads"
	}

	// Clean directory path
	directory = filepath.Clean(directory)
	if strings.HasPrefix(directory, "..") || strings.HasPrefix(directory, "/") {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid directory path",
		})
		return
	}

	// Generate unique filename
	uniqueFilename := h.generateUniqueFilename(file.Filename)
	key := filepath.Join(directory, uniqueFilename)

	// Determine if file should be public
	isPublic := c.PostForm("public") == "true"

	// Set upload options
	options := &storage.UploadOptions{
		ContentType: file.Header.Get("Content-Type"),
		Metadata: map[string]string{
			"original_name": file.Filename,
			"upload_time":   time.Now().Format(time.RFC3339),
			"uploaded_by":   "api", // TODO: Get from authenticated user
		},
		Tags: map[string]string{
			"directory": directory,
		},
	}

	if isPublic {
		options.ACL = "public-read"
	}

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		h.logger.Error().Err(err).Str("filename", file.Filename).Msg("Failed to open uploaded file")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to process uploaded file",
			Details: err.Error(),
		})
		return
	}
	defer src.Close()

	// Upload file to storage
	result, err := h.storageProvider.Upload(c, key, src, options.ContentType, options)
	if err != nil {
		h.logger.Error().Err(err).Str("key", key).Msg("Failed to upload file to storage")
		handleFileError(c, err)
		return
	}

	// Set original name in result metadata
	if result.Metadata == nil {
		result.Metadata = make(map[string]string)
	}
	result.Metadata["original_name"] = file.Filename

	h.logger.Info().
		Str("key", key).
		Str("original_name", file.Filename).
		Int64("size", result.Size).
		Str("url", result.URL).
		Msg("File uploaded successfully")

	c.JSON(http.StatusCreated, result)
}

// UploadMultipleFiles uploads multiple files
// @Summary Upload multiple files
// @Description Upload multiple files in a single request
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param files formData file true "Files to upload"
// @Param directory formData string false "Directory for file organization"
// @Param public formData bool false "Make files publicly accessible"
// @Success 201 {object} dto.BatchFileUploadResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 413 {object} dto.ErrorResponse
// @Failure 415 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/files/batch [post]
func (h *FileHandler) UploadMultipleFiles(c *gin.Context) {
	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse multipart form")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Failed to parse uploaded files",
			Details: err.Error(),
		})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "No files uploaded",
		})
		return
	}

	// Limit number of files
	if len(files) > 20 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Too many files uploaded",
			Details: "Maximum 20 files can be uploaded at once",
		})
		return
	}

	// Get directory from form or use default
	directory := c.PostForm("directory")
	if directory == "" {
		directory = "uploads"
	}

	// Clean directory path
	directory = filepath.Clean(directory)
	if strings.HasPrefix(directory, "..") || strings.HasPrefix(directory, "/") {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid directory path",
		})
		return
	}

	// Determine if files should be public
	isPublic := c.PostForm("public") == "true"

	results := make([]*dto.UploadResult, 0, len(files))
	errors := make([]dto.BatchFileUploadError, 0)

	// Process each file
	for i, file := range files {
		// Validate file
		if err := h.validateFile(file); err != nil {
			errors = append(errors, dto.BatchFileUploadError{
				Index:    i,
				Filename: file.Filename,
				Error:    err.Error(),
			})
			continue
		}

		// Generate unique filename
		uniqueFilename := h.generateUniqueFilename(file.Filename)
		key := filepath.Join(directory, uniqueFilename)

		// Set upload options
		options := &storage.UploadOptions{
			ContentType: file.Header.Get("Content-Type"),
			Metadata: map[string]string{
				"original_name": file.Filename,
				"upload_time":   time.Now().Format(time.RFC3339),
				"uploaded_by":   "api",
			},
		}

		if isPublic {
			options.ACL = "public-read"
		}

		// Open uploaded file
		src, err := file.Open()
		if err != nil {
			errors = append(errors, dto.BatchFileUploadError{
				Index:    i,
				Filename: file.Filename,
				Error:    fmt.Sprintf("Failed to open file: %v", err),
			})
			continue
		}

		// Upload file to storage
		result, err := h.storageProvider.Upload(c, key, src, options.ContentType, options)
		src.Close()

		if err != nil {
			errors = append(errors, dto.BatchFileUploadError{
				Index:    i,
				Filename: file.Filename,
				Error:    err.Error(),
			})
			continue
		}

		// Get original name from metadata
		originalName := result.Metadata["original_name"]
		if originalName == "" {
			originalName = file.Filename
		}
		results = append(results, &dto.UploadResult{
			Filename:     file.Filename,
			OriginalName: originalName,
			Size:         result.Size,
			MimeType:     result.ContentType,
			URL:          result.URL,
			Path:         result.URL, // For S3, URL is the path
			Directory:    directory,
			UploadedAt:   result.LastModified,
		})
	}

	// Log batch upload results
	h.logger.Info().
		Int("total_files", len(files)).
		Int("successful", len(results)).
		Int("failed", len(errors)).
		Str("directory", directory).
		Msg("Batch file upload completed")

	response := dto.BatchFileUploadResponse{
		Results:      results,
		Errors:       errors,
		TotalFiles:   len(files),
		Successful:   len(results),
		Failed:       len(errors),
	}

	c.JSON(http.StatusCreated, response)
}

// GetFile downloads a file
// @Summary Download file
// @Description Download a file from storage
// @Tags files
// @Accept json
// @Produce application/octet-stream
// @Param key path string true "File key"
// @Success 200 {file} binary
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/files/{key} [get]
func (h *FileHandler) GetFile(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "File key is required",
		})
		return
	}

	// Validate key
	key = filepath.Clean(key)
	if strings.HasPrefix(key, "..") {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid file key",
		})
		return
	}

	// Download file from storage
	reader, err := h.storageProvider.Download(c, key)
	if err != nil {
		h.logger.Error().Err(err).Str("key", key).Msg("Failed to download file from storage")
		handleFileError(c, err)
		return
	}
	defer reader.Close()

	// Get file metadata to set headers
	metadata, err := h.storageProvider.GetMetadata(c, key)
	if err == nil {
		// Set content type header
		if metadata.ContentType != "" {
			c.Header("Content-Type", metadata.ContentType)
		}
		// Set content length header
		if metadata.Size > 0 {
			c.Header("Content-Length", strconv.FormatInt(metadata.Size, 10))
		}
		// Set cache control
		c.Header("Cache-Control", "public, max-age=31536000") // 1 year
	}

	// Get filename from metadata or key
	filename := filepath.Base(key)
	if metadata != nil && metadata.Metadata["original_name"] != "" {
		filename = metadata.Metadata["original_name"]
	}

	// Set content disposition header for download
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))

	// Stream file to response
	c.DataFromReader(http.StatusOK, -1, metadata.ContentType, reader, nil)
}

// GetFileURL returns a URL for a file
// @Summary Get file URL
// @Description Get a URL for accessing a file
// @Tags files
// @Accept json
// @Produce json
// @Param key path string true "File key"
// @Param presigned query bool false "Generate presigned URL"
// @Param expiry query int false "Presigned URL expiry in minutes" default(60)
// @Success 200 {object} dto.FileURLResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/files/{key}/url [get]
func (h *FileHandler) GetFileURL(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "File key is required",
		})
		return
	}

	// Validate key
	key = filepath.Clean(key)
	if strings.HasPrefix(key, "..") {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid file key",
		})
		return
	}

	// Check if file exists
	exists, err := h.storageProvider.Exists(c, key)
	if err != nil {
		h.logger.Error().Err(err).Str("key", key).Msg("Failed to check file existence")
		handleFileError(c, err)
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error: "File not found",
		})
		return
	}

	// Generate presigned URL if requested
	presigned := c.Query("presigned") == "true"
	var url string
	var expiresAt *time.Time

	if presigned && h.config.EnablePresignedURL {
		expiryMinutes := h.config.PresignedExpiry
		if expiryStr := c.Query("expiry"); expiryStr != "" {
			if minutes, err := strconv.Atoi(expiryStr); err == nil {
				expiryMinutes = minutes
			}
		}

		expiryDuration := time.Duration(expiryMinutes) * time.Minute
		url, err = h.storageProvider.GetPresignedURL(c, key, expiryDuration)
		if err != nil {
			h.logger.Error().Err(err).Str("key", key).Msg("Failed to generate presigned URL")
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Error:   "Failed to generate presigned URL",
				Details: err.Error(),
			})
			return
		}

		expiresAtTime := time.Now().Add(expiryDuration)
		expiresAt = &expiresAtTime
	} else {
		var err error
		url, err = h.storageProvider.GetURL(c, key)
		if err != nil {
			handleFileError(c, err)
			return
		}
	}

	response := dto.FileURLResponse{
		Key:       key,
		URL:       url,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// DeleteFile deletes a file
// @Summary Delete file
// @Description Delete a file from storage
// @Tags files
// @Accept json
// @Produce json
// @Param key path string true "File key"
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/files/{key} [delete]
func (h *FileHandler) DeleteFile(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "File key is required",
		})
		return
	}

	// Validate key
	key = filepath.Clean(key)
	if strings.HasPrefix(key, "..") {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid file key",
		})
		return
	}

	// Delete file from storage
	err := h.storageProvider.Delete(c, key)
	if err != nil {
		h.logger.Error().Err(err).Str("key", key).Msg("Failed to delete file from storage")
		handleFileError(c, err)
		return
	}

	h.logger.Info().Str("key", key).Msg("File deleted successfully")

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "File deleted successfully",
		Success: true,
	})
}

// ListFiles lists files in a directory
// @Summary List files
// @Description List files in storage with optional filtering
// @Tags files
// @Accept json
// @Produce json
// @Param prefix query string false "Directory prefix"
// @Param recursive query bool false "List recursively" default(false)
// @Param limit query int false "Maximum number of files" default(100)
// @Param continuation_token query string false "Continuation token for pagination"
// @Success 200 {object} dto.FileListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/files [get]
func (h *FileHandler) ListFiles(c *gin.Context) {
	prefix := c.Query("prefix")
	recursive := c.Query("recursive") == "true"

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	options := &storage.ListOptions{
		Prefix:            prefix,
		Recursive:         recursive,
		MaxKeys:          limit,
		ContinuationToken: c.Query("continuation_token"),
	}

	result, err := h.storageProvider.ListWithPagination(c, prefix, options)
	if err != nil {
		h.logger.Error().Err(err).Str("prefix", prefix).Msg("Failed to list files from storage")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to list files",
			Details: err.Error(),
		})
		return
	}

	// Convert to response format
	files := make([]*dto.FileInfo, len(result.Objects))
	for i, obj := range result.Objects {
		url, err := h.storageProvider.GetURL(c, obj.Key)
		if err != nil {
			h.logger.Error().Err(err).Str("key", obj.Key).Msg("Failed to get file URL")
			url = "" // Set empty URL on error
		}

		files[i] = &dto.FileInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			ContentType:  obj.ContentType,
			ETag:         obj.ETag,
			URL:          url,
		}
	}

	response := dto.FileListResponse{
		Files:                 files,
		Prefixes:              result.Prefixes,
		IsTruncated:           result.IsTruncated,
		NextContinuationToken: result.NextContinuationToken,
		TotalFiles:           len(files),
	}

	c.JSON(http.StatusOK, response)
}

// GetFileMetadata retrieves metadata for a file
// @Summary Get file metadata
// @Description Retrieve metadata for a specific file
// @Tags files
// @Accept json
// @Produce json
// @Param key path string true "File key"
// @Success 200 {object} dto.FileMetadataResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/files/{key}/metadata [get]
func (h *FileHandler) GetFileMetadata(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "File key is required",
		})
		return
	}

	// Validate key
	key = filepath.Clean(key)
	if strings.HasPrefix(key, "..") {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid file key",
		})
		return
	}

	// Get metadata from storage
	metadata, err := h.storageProvider.GetMetadata(c, key)
	if err != nil {
		h.logger.Error().Err(err).Str("key", key).Msg("Failed to get file metadata from storage")
		handleFileError(c, err)
		return
	}

	response := dto.FileMetadataResponse{
		Key:           metadata.Key,
		Size:          metadata.Size,
		LastModified:  metadata.LastModified,
		ContentType:   metadata.ContentType,
		ETag:          metadata.ETag,
		StorageClass:  metadata.StorageClass,
		Metadata:      metadata.Metadata,
		Tags:          metadata.Tags,
		CacheControl:  metadata.CacheControl,
		URL: func() string {
			url, err := h.storageProvider.GetURL(c, key)
			if err != nil {
				h.logger.Error().Err(err).Str("key", key).Msg("Failed to get file URL")
				return ""
			}
			return url
		}(),
	}

	c.JSON(http.StatusOK, response)
}

// UpdateFileMetadata updates metadata for a file
// @Summary Update file metadata
// @Description Update metadata for a specific file
// @Tags files
// @Accept json
// @Produce json
// @Param key path string true "File key"
// @Param metadata body dto.UpdateMetadataRequest true "Metadata to update"
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/files/{key}/metadata [put]
func (h *FileHandler) UpdateFileMetadata(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "File key is required",
		})
		return
	}

	// Validate key
	key = filepath.Clean(key)
	if strings.HasPrefix(key, "..") {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid file key",
		})
		return
	}

	var req dto.UpdateMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid metadata update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Update metadata in storage
	err := h.storageProvider.SetMetadata(c, key, req.Metadata)
	if err != nil {
		h.logger.Error().Err(err).Str("key", key).Msg("Failed to update file metadata in storage")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to update file metadata",
			Details: err.Error(),
		})
		return
	}

	h.logger.Info().Str("key", key).Msg("File metadata updated successfully")

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "File metadata updated successfully",
		Success: true,
	})
}

// Helper Methods

// validateFile validates an uploaded file
func (h *FileHandler) validateFile(file *multipart.FileHeader) error {
	// Check file size
	if file.Size > h.config.MaxFileSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", file.Size, h.config.MaxFileSize)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if len(h.config.AllowedExtensions) > 0 {
		allowed := false
		for _, allowedExt := range h.config.AllowedExtensions {
			if ext == allowedExt {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file extension %s is not allowed", ext)
		}
	}

	// Check MIME type
	contentType := file.Header.Get("Content-Type")
	if len(h.config.AllowedMimeTypes) > 0 {
		allowed := false
		for _, allowedType := range h.config.AllowedMimeTypes {
			if contentType == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("MIME type %s is not allowed", contentType)
		}
	}

	return nil
}

// generateUniqueFilename generates a unique filename
func (h *FileHandler) generateUniqueFilename(originalName string) string {
	ext := filepath.Ext(originalName)
	name := strings.TrimSuffix(originalName, ext)

	// Remove any special characters and spaces
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")

	// Add UUID and timestamp for uniqueness
	uniqueID := uuid.New().String()[:8]
	timestamp := time.Now().Format("20060102_150405")

	return fmt.Sprintf("%s_%s_%s%s", name, timestamp, uniqueID, ext)
}

// handleFileError handles storage errors and maps them to HTTP status codes
func handleFileError(c *gin.Context, err error) {
	if storage.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Validation error",
			Details: err.Error(),
		})
		return
	}

	if storage.IsNotFoundError(err) {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "File not found",
			Details: err.Error(),
		})
		return
	}

	if storage.IsAccessDeniedError(err) {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Error:   "Access denied",
			Details: err.Error(),
		})
		return
	}

	if storage.IsConnectionError(err) {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{
			Error:   "Storage service unavailable",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
		Error:   "Internal server error",
		Details: err.Error(),
	})
}