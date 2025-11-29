package file

import (
	"context"
	"crypto/rand"
	"erpgo/pkg/security"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service defines the file management interface
type Service interface {
	// File upload operations
	UploadFile(ctx context.Context, file *multipart.FileHeader, directory string) (*UploadResult, error)
	UploadProductImage(ctx context.Context, file *multipart.FileHeader, productID uuid.UUID) (*UploadResult, error)
	UploadProductVariantImage(ctx context.Context, file *multipart.FileHeader, productID, variantID uuid.UUID) (*UploadResult, error)

	// File retrieval operations
	GetFile(ctx context.Context, filename string) (*File, error)
	GetFileURL(ctx context.Context, filename string) (string, error)
	DeleteFile(ctx context.Context, filename string) error

	// Image processing operations
	ProcessImage(ctx context.Context, srcPath, dstPath string, options *ImageProcessingOptions) error
	GenerateThumbnail(ctx context.Context, srcPath, dstPath string, width, height int) error
	OptimizeImage(ctx context.Context, srcPath, dstPath string) error

	// File validation operations
	ValidateFile(file *multipart.FileHeader, allowedTypes []string, maxSize int64) error
	ValidateImageFile(file *multipart.FileHeader) error
}

// UploadResult represents the result of a file upload
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

// File represents a file entity
type File struct {
	ID           uuid.UUID `json:"id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"original_name"`
	Path         string    `json:"path"`
	URL          string    `json:"url"`
	Size         int64     `json:"size"`
	MimeType     string    `json:"mime_type"`
	Directory    string    `json:"directory"`
	IsPublic     bool      `json:"is_public"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ImageProcessingOptions contains options for image processing
type ImageProcessingOptions struct {
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Quality     int    `json:"quality,omitempty"`     // JPEG quality (1-100)
	Format      string `json:"format,omitempty"`      // jpeg, png, webp
	ResizeMode  string `json:"resize_mode,omitempty"` // fit, fill, crop
	MaxFileSize int64  `json:"max_file_size,omitempty"`
}

// FileUploadConfig contains configuration for file uploads
type FileUploadConfig struct {
	MaxFileSize      int64    `json:"max_file_size"`      // Maximum file size in bytes
	AllowedMimeTypes []string `json:"allowed_mime_types"` // Allowed MIME types
	UploadPath       string   `json:"upload_path"`        // Base upload directory
	PublicURL        string   `json:"public_url"`         // Base URL for public access
	MaxImageWidth    int      `json:"max_image_width"`    // Maximum image width
	MaxImageHeight   int      `json:"max_image_height"`   // Maximum image height
	ThumbnailWidth   int      `json:"thumbnail_width"`    // Thumbnail width
	ThumbnailHeight  int      `json:"thumbnail_height"`   // Thumbnail height
	EnableWebP       bool     `json:"enable_webp"`        // Enable WebP conversion
}

// Default configuration
var DefaultConfig = FileUploadConfig{
	MaxFileSize: 10 * 1024 * 1024, // 10MB
	AllowedMimeTypes: []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/webp",
		"image/gif",
	},
	UploadPath:      "./uploads",
	PublicURL:       "/uploads",
	MaxImageWidth:   2048,
	MaxImageHeight:  2048,
	ThumbnailWidth:  300,
	ThumbnailHeight: 300,
	EnableWebP:      true,
}

// ServiceImpl implements the file service interface
type ServiceImpl struct {
	config FileUploadConfig
	logger zerolog.Logger
}

// NewService creates a new file service instance
func NewService(config FileUploadConfig, logger zerolog.Logger) Service {
	return &ServiceImpl{
		config: config,
		logger: logger,
	}
}

// NewServiceWithDefaults creates a new file service with default configuration
func NewServiceWithDefaults(logger zerolog.Logger) Service {
	return NewService(DefaultConfig, logger)
}

// UploadFile uploads a file to the specified directory
func (s *ServiceImpl) UploadFile(ctx context.Context, file *multipart.FileHeader, directory string) (*UploadResult, error) {
	// Validate file
	if err := s.ValidateFile(file, s.config.AllowedMimeTypes, s.config.MaxFileSize); err != nil {
		return nil, fmt.Errorf("file validation failed: %w", err)
	}

	// Create upload directory if it doesn't exist
	uploadDir := filepath.Join(s.config.UploadPath, directory)

	// Validate upload directory path
	if err := security.ValidatePath(uploadDir, s.config.UploadPath); err != nil {
		return nil, fmt.Errorf("invalid upload directory path: %w", err)
	}

	if err := os.MkdirAll(uploadDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	uniqueFilename := s.generateUniqueFilename(ext)
	filePath := filepath.Join(uploadDir, uniqueFilename)

	// Validate destination file path
	if err := security.ValidatePath(filePath, s.config.UploadPath); err != nil {
		return nil, fmt.Errorf("invalid file path for upload: %w", err)
	}

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Process image if it's an image file
	if s.isImageFile(file.Header.Get("Content-Type")) {
		processedPath := filepath.Join(uploadDir, "processed", uniqueFilename)

		// Validate processed directory path
		if err := security.ValidatePath(filepath.Dir(processedPath), s.config.UploadPath); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to create processed directory, using original file")
		} else {
			if err := os.MkdirAll(filepath.Dir(processedPath), 0750); err != nil {
				s.logger.Warn().Err(err).Msg("Failed to create processed directory, using original file")
			} else {
				if err := s.ProcessImage(ctx, filePath, processedPath, &ImageProcessingOptions{
					Width:      s.config.MaxImageWidth,
					Height:     s.config.MaxImageHeight,
					Quality:    85,
					Format:     "jpeg",
					ResizeMode: "fit",
				}); err != nil {
					s.logger.Warn().Err(err).Msg("Failed to process image, using original file")
				} else {
					filePath = processedPath
				}

				// Generate thumbnail
				thumbnailPath := filepath.Join(uploadDir, "thumbnails", uniqueFilename)

				// Validate thumbnail directory path
				if err := security.ValidatePath(filepath.Dir(thumbnailPath), s.config.UploadPath); err != nil {
					s.logger.Warn().Err(err).Msg("Failed to create thumbnail directory")
				} else {
					if err := os.MkdirAll(filepath.Dir(thumbnailPath), 0750); err != nil {
						s.logger.Warn().Err(err).Msg("Failed to create thumbnail directory")
					} else {
						if err := s.GenerateThumbnail(ctx, filePath, thumbnailPath, s.config.ThumbnailWidth, s.config.ThumbnailHeight); err != nil {
							s.logger.Warn().Err(err).Msg("Failed to generate thumbnail")
						}
					}
				}
			}
		}
	}

	// Generate URL
	url := fmt.Sprintf("%s/%s/%s", s.config.PublicURL, directory, uniqueFilename)

	result := &UploadResult{
		Filename:     uniqueFilename,
		OriginalName: file.Filename,
		Size:         file.Size,
		MimeType:     file.Header.Get("Content-Type"),
		URL:          url,
		Path:         filePath,
		Directory:    directory,
		UploadedAt:   time.Now().UTC(),
	}

	s.logger.Info().
		Str("filename", uniqueFilename).
		Str("original_name", file.Filename).
		Int64("size", file.Size).
		Str("directory", directory).
		Msg("File uploaded successfully")

	return result, nil
}

// UploadProductImage uploads an image for a product
func (s *ServiceImpl) UploadProductImage(ctx context.Context, file *multipart.FileHeader, productID uuid.UUID) (*UploadResult, error) {
	directory := fmt.Sprintf("products/%s", productID.String())
	return s.UploadFile(ctx, file, directory)
}

// UploadProductVariantImage uploads an image for a product variant
func (s *ServiceImpl) UploadProductVariantImage(ctx context.Context, file *multipart.FileHeader, productID, variantID uuid.UUID) (*UploadResult, error) {
	directory := fmt.Sprintf("products/%s/variants/%s", productID.String(), variantID.String())
	return s.UploadFile(ctx, file, directory)
}

// GetFile retrieves a file by filename
func (s *ServiceImpl) GetFile(ctx context.Context, filename string) (*File, error) {
	// Validate filename path
	if err := security.ValidatePath(filename, s.config.UploadPath); err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	// This would typically fetch from a database
	// For now, we'll create a basic file object
	filePath := filepath.Join(s.config.UploadPath, filename)

	// Validate full file path
	if err := security.ValidatePath(filePath, s.config.UploadPath); err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// Determine directory from path
	relPath, err := filepath.Rel(s.config.UploadPath, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative path: %w", err)
	}

	directory := filepath.Dir(relPath)
	url := fmt.Sprintf("%s/%s", s.config.PublicURL, relPath)

	return &File{
		ID:           uuid.New(),
		Filename:     filename,
		OriginalName: filename,
		Path:         filePath,
		URL:          url,
		Size:         fileInfo.Size(),
		MimeType:     s.getMimeType(filename),
		Directory:    directory,
		IsPublic:     true,
		CreatedAt:    fileInfo.ModTime().UTC(),
		UpdatedAt:    fileInfo.ModTime().UTC(),
	}, nil
}

// GetFileURL returns the public URL for a file
func (s *ServiceImpl) GetFileURL(ctx context.Context, filename string) (string, error) {
	return fmt.Sprintf("%s/%s", s.config.PublicURL, filename), nil
}

// DeleteFile deletes a file
func (s *ServiceImpl) DeleteFile(ctx context.Context, filename string) error {
	// Validate filename path
	if err := security.ValidatePath(filename, s.config.UploadPath); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	filePath := filepath.Join(s.config.UploadPath, filename)

	// Validate full file path
	if err := security.ValidatePath(filePath, s.config.UploadPath); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Also delete processed versions
	processedPath := filepath.Join(s.config.UploadPath, "processed", filename)
	if err := security.ValidatePath(processedPath, s.config.UploadPath); err == nil {
		if err := os.Remove(processedPath); err != nil && !os.IsNotExist(err) {
			s.logger.Warn().Err(err).Str("path", processedPath).Msg("Failed to delete processed file")
		}
	}

	thumbnailPath := filepath.Join(s.config.UploadPath, "thumbnails", filename)
	if err := security.ValidatePath(thumbnailPath, s.config.UploadPath); err == nil {
		if err := os.Remove(thumbnailPath); err != nil && !os.IsNotExist(err) {
			s.logger.Warn().Err(err).Str("path", thumbnailPath).Msg("Failed to delete thumbnail file")
		}
	}

	s.logger.Info().Str("filename", filename).Msg("File deleted successfully")
	return nil
}

// ProcessImage processes an image with the given options
func (s *ServiceImpl) ProcessImage(ctx context.Context, srcPath, dstPath string, options *ImageProcessingOptions) error {
	// Validate source path
	if err := security.ValidatePath(srcPath, s.config.UploadPath); err != nil {
		return fmt.Errorf("invalid source image path: %w", err)
	}

	// Validate destination path
	if err := security.ValidatePath(dstPath, s.config.UploadPath); err != nil {
		return fmt.Errorf("invalid destination image path: %w", err)
	}

	// Open source image file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source image: %w", err)
	}
	defer srcFile.Close()

	// Decode image
	img, _, err := image.Decode(srcFile)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize image if needed
	if options.Width > 0 || options.Height > 0 {
		img = s.resizeImage(img, options)
	}

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dstPath), 0750); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Encode and save image
	format := options.Format
	if format == "" {
		format = "jpeg"
	}

	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		quality := options.Quality
		if quality == 0 {
			quality = 85
		}
		err = jpeg.Encode(dstFile, img, &jpeg.Options{Quality: quality})
	case "png":
		err = png.Encode(dstFile, img)
	case "webp":
		// WebP support temporarily disabled due to API compatibility issues
		// TODO: Implement WebP encoding with correct API
		return fmt.Errorf("WebP format not currently supported - API compatibility issue")
	default:
		return fmt.Errorf("unsupported image format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	return nil
}

// GenerateThumbnail generates a thumbnail for an image
func (s *ServiceImpl) GenerateThumbnail(ctx context.Context, srcPath, dstPath string, width, height int) error {
	options := &ImageProcessingOptions{
		Width:      width,
		Height:     height,
		Quality:    80,
		Format:     "jpeg",
		ResizeMode: "fit",
	}
	return s.ProcessImage(ctx, srcPath, dstPath, options)
}

// OptimizeImage optimizes an image for web use
func (s *ServiceImpl) OptimizeImage(ctx context.Context, srcPath, dstPath string) error {
	options := &ImageProcessingOptions{
		Width:      s.config.MaxImageWidth,
		Height:     s.config.MaxImageHeight,
		Quality:    80,
		Format:     "jpeg",
		ResizeMode: "fit",
	}
	return s.ProcessImage(ctx, srcPath, dstPath, options)
}

// ValidateFile validates a file against allowed types and size
func (s *ServiceImpl) ValidateFile(file *multipart.FileHeader, allowedTypes []string, maxSize int64) error {
	// Check file size
	if file.Size > maxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", file.Size, maxSize)
	}

	// Check MIME type
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		// Try to detect MIME type from extension
		ext := strings.ToLower(filepath.Ext(file.Filename))
		mimeType = s.getMimeTypeFromExtension(ext)
	}

	isAllowed := false
	for _, allowedType := range allowedTypes {
		if mimeType == allowedType {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return fmt.Errorf("file type %s is not allowed", mimeType)
	}

	return nil
}

// ValidateImageFile validates that a file is a valid image
func (s *ServiceImpl) ValidateImageFile(file *multipart.FileHeader) error {
	// Check if it's an image file
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		mimeType = s.getMimeTypeFromExtension(ext)
	}

	imageTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/webp",
		"image/gif",
	}

	for _, imageType := range imageTypes {
		if mimeType == imageType {
			return nil
		}
	}

	return fmt.Errorf("file is not a valid image format")
}

// Helper methods

// generateUniqueFilename generates a unique filename
func (s *ServiceImpl) generateUniqueFilename(ext string) string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)

	return fmt.Sprintf("%d_%x%s", timestamp, randomBytes, ext)
}

// isImageFile checks if a file is an image based on MIME type
func (s *ServiceImpl) isImageFile(mimeType string) bool {
	imageTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/webp",
		"image/gif",
	}

	for _, imageType := range imageTypes {
		if mimeType == imageType {
			return true
		}
	}
	return false
}

// getMimeType returns MIME type based on file extension
func (s *ServiceImpl) getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	return s.getMimeTypeFromExtension(ext)
}

// getMimeTypeFromExtension returns MIME type based on file extension
func (s *ServiceImpl) getMimeTypeFromExtension(ext string) string {
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".webp": "image/webp",
		".gif":  "image/gif",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".txt":  "text/plain",
		".csv":  "text/csv",
	}

	if mimeType, exists := mimeTypes[ext]; exists {
		return mimeType
	}
	return "application/octet-stream"
}

// resizeImage resizes an image based on options
func (s *ServiceImpl) resizeImage(img image.Image, options *ImageProcessingOptions) image.Image {
	bounds := img.Bounds()
	originalWidth, originalHeight := bounds.Dx(), bounds.Dy()

	// Calculate new dimensions
	newWidth, newHeight := originalWidth, originalHeight

	if options.Width > 0 && options.Height > 0 {
		// Both width and height specified
		switch options.ResizeMode {
		case "fit":
			// Fit within specified dimensions (maintain aspect ratio)
			ratio := min(float64(options.Width)/float64(originalWidth), float64(options.Height)/float64(originalHeight))
			newWidth = int(float64(originalWidth) * ratio)
			newHeight = int(float64(originalHeight) * ratio)
		case "fill":
			// Fill specified dimensions (maintain aspect ratio, crop if needed)
			ratio := max(float64(options.Width)/float64(originalWidth), float64(options.Height)/float64(originalHeight))
			newWidth = int(float64(originalWidth) * ratio)
			newHeight = int(float64(originalHeight) * ratio)
		case "crop":
			// Crop to exact dimensions
			newWidth = options.Width
			newHeight = options.Height
		}
	} else if options.Width > 0 {
		// Only width specified
		ratio := float64(options.Width) / float64(originalWidth)
		newWidth = options.Width
		newHeight = int(float64(originalHeight) * ratio)
	} else if options.Height > 0 {
		// Only height specified
		ratio := float64(options.Height) / float64(originalHeight)
		newHeight = options.Height
		newWidth = int(float64(originalWidth) * ratio)
	}

	// Create new image with calculated dimensions
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Simple resize (would use better algorithms in production)
	// This is a placeholder implementation
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			// Map new coordinates to original coordinates
			srcX := x * originalWidth / newWidth
			srcY := y * originalHeight / newHeight
			newImg.Set(x, y, img.At(srcX, srcY))
		}
	}

	return newImg
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
