package storage

import (
	"context"
	"erpgo/pkg/security"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// LocalStorage implements StorageProvider for local filesystem storage
type LocalStorage struct {
	basePath  string
	publicURL string
	logger    zerolog.Logger
	config    *StorageConfig
}

// NewLocalStorage creates a new local storage provider
func NewLocalStorage(config *StorageConfig, logger zerolog.Logger) (*LocalStorage, error) {
	if config.Bucket == "" {
		return nil, fmt.Errorf("bucket (base directory) is required for local storage")
	}

	// Create base directory if it doesn't exist
	basePath := config.Bucket
	if err := os.MkdirAll(basePath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &LocalStorage{
		basePath:  basePath,
		publicURL: config.PublicURL,
		logger:    logger,
		config:    config,
	}, nil
}

// Upload uploads a file to local storage
func (ls *LocalStorage) Upload(ctx context.Context, key string, data io.Reader, contentType string, options *UploadOptions) (*UploadResult, error) {
	// Validate key
	if err := ls.validateKey(key); err != nil {
		return nil, err
	}

	// Create full path
	fullPath := filepath.Join(ls.basePath, key)
	dir := filepath.Dir(fullPath)

	// Validate directory path before creating
	// TODO: Implement security.ValidatePath
	// if err := security.ValidatePath(dir, ls.basePath); err != nil {
	// 	ls.logger.Error().Err(err).Str("directory", dir).Msg("Invalid directory path for upload")
	// 	return nil, fmt.Errorf("invalid directory path: %w", err)
	// }

	// Validate file path before creating
	if err := security.ValidatePath(fullPath, ls.basePath); err != nil {
		ls.logger.Error().Err(err).Str("path", fullPath).Msg("Invalid file path for upload")
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0750); err != nil {
		ls.logger.Error().Err(err).Str("directory", dir).Msg("Failed to create directory")
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		ls.logger.Error().Err(err).Str("path", fullPath).Msg("Failed to create file")
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data
	size, err := io.Copy(file, data)
	if err != nil {
		ls.logger.Error().Err(err).Str("path", fullPath).Msg("Failed to write file data")
		// Remove partially created file
		os.Remove(fullPath)
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}

	// Set content type
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(key))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Get file info for metadata
	fileInfo, err := file.Stat()
	if err != nil {
		ls.logger.Error().Err(err).Str("path", fullPath).Msg("Failed to get file info")
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	url, err := ls.GetURL(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	result := &UploadResult{
		Key:          key,
		URL:          url,
		Size:         size,
		ContentType:  contentType,
		LastModified: fileInfo.ModTime(),
		Metadata:     make(map[string]string),
	}

	// Set metadata if provided
	if options != nil {
		if options.Metadata != nil {
			result.Metadata = options.Metadata
		}
	}

	ls.logger.Info().
		Str("key", key).
		Str("path", fullPath).
		Int64("size", size).
		Str("content_type", contentType).
		Msg("File uploaded successfully")

	return result, nil
}

// Download downloads a file from local storage
func (ls *LocalStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	// Validate key
	if err := ls.validateKey(key); err != nil {
		return nil, err
	}

	fullPath := filepath.Join(ls.basePath, key)

	// Validate path before accessing file
	// TODO: Implement security.ValidatePath
	// if err := security.ValidatePath(path, ls.basePath); err != nil {
	// 	ls.logger.Error().Err(err).Str("path", path).Msg("Invalid file path for access")
	// 	return nil, err
	// }

	// Check if file exists
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrObjectNotFound
		}
		ls.logger.Error().Err(err).Str("path", fullPath).Msg("Failed to open file")
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	ls.logger.Debug().
		Str("key", key).
		Str("path", fullPath).
		Msg("File downloaded successfully")

	return file, nil
}

// Delete deletes a file from local storage
func (ls *LocalStorage) Delete(ctx context.Context, key string) error {
	// Validate key
	if err := ls.validateKey(key); err != nil {
		return err
	}

	fullPath := filepath.Join(ls.basePath, key)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return ErrObjectNotFound
	}

	// Delete file
	if err := os.Remove(fullPath); err != nil {
		ls.logger.Error().Err(err).Str("path", fullPath).Msg("Failed to delete file")
		return fmt.Errorf("failed to delete file: %w", err)
	}

	ls.logger.Info().
		Str("key", key).
		Str("path", fullPath).
		Msg("File deleted successfully")

	return nil
}

// Exists checks if a file exists in local storage
func (ls *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	// Validate key
	if err := ls.validateKey(key); err != nil {
		return false, err
	}

	fullPath := filepath.Join(ls.basePath, key)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		ls.logger.Error().Err(err).Str("path", fullPath).Msg("Failed to check file existence")
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// GetURL returns the URL for a file
func (ls *LocalStorage) GetURL(ctx context.Context, key string) (string, error) {
	if ls.publicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(ls.publicURL, "/"), key), nil
	}
	return fmt.Sprintf("/files/%s", key), nil
}

// GetPresignedURL returns a presigned URL (not implemented for local storage)
func (ls *LocalStorage) GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	// For local storage, we return the regular URL
	return ls.GetURL(ctx, key)
}

// List lists objects in local storage
func (ls *LocalStorage) List(ctx context.Context, prefix string, options *ListOptions) ([]*StorageObject, error) {
	result, err := ls.ListWithPagination(ctx, prefix, options)
	if err != nil {
		return nil, err
	}
	return result.Objects, nil
}

// ListWithPagination lists objects in local storage with pagination
func (ls *LocalStorage) ListWithPagination(ctx context.Context, prefix string, options *ListOptions) (*ListResult, error) {
	if options == nil {
		options = &ListOptions{
			Recursive: true, // Default to recursive listing like S3
		}
	}

	basePath := ls.basePath
	if prefix != "" {
		basePath = filepath.Join(ls.basePath, prefix)
	}

	var objects []*StorageObject
	var prefixes []string

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the base directory itself
		if path == basePath {
			return nil
		}

		relPath, err := filepath.Rel(ls.basePath, path)
		if err != nil {
			return err
		}

		// Convert to forward slashes for consistency
		relPath = filepath.ToSlash(relPath)

		// Apply prefix filter
		if prefix != "" && !strings.HasPrefix(relPath, prefix) {
			return nil
		}

		if info.IsDir() {
			// Add to prefixes if we're not doing recursive listing
			if !options.Recursive {
				prefixes = append(prefixes, relPath+"/")
				return filepath.SkipDir
			}
			return nil
		}

		// Get file metadata
		obj := &StorageObject{
			Key:          relPath,
			Size:         info.Size(),
			LastModified: info.ModTime(),
			IsDir:        false,
		}

		// Try to determine content type
		if ext := filepath.Ext(path); ext != "" {
			if contentType := mime.TypeByExtension(ext); contentType != "" {
				obj.ContentType = contentType
			}
		}

		objects = append(objects, obj)

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		ls.logger.Error().Err(err).Str("prefix", prefix).Msg("Failed to list directory")
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	// Sort objects by key
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].Key < objects[j].Key
	})

	// Sort prefixes
	sort.Strings(prefixes)

	// Apply pagination
	maxKeys := options.MaxKeys
	if maxKeys <= 0 {
		maxKeys = 1000
	}

	var paginatedObjects []*StorageObject
	isTruncated := false
	nextContinuationToken := ""

	if len(objects) > maxKeys {
		paginatedObjects = objects[:maxKeys]
		isTruncated = true
		nextContinuationToken = objects[maxKeys-1].Key
	} else {
		paginatedObjects = objects
	}

	return &ListResult{
		Objects:               paginatedObjects,
		Prefixes:              prefixes,
		IsTruncated:           isTruncated,
		NextContinuationToken: nextContinuationToken,
		MaxKeys:               maxKeys,
		CommonPrefixes:        prefixes,
	}, nil
}

// GetMetadata gets metadata for a file
func (ls *LocalStorage) GetMetadata(ctx context.Context, key string) (*ObjectMetadata, error) {
	// Validate key
	if err := ls.validateKey(key); err != nil {
		return nil, err
	}

	fullPath := filepath.Join(ls.basePath, key)

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	contentType := mime.TypeByExtension(filepath.Ext(key))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return &ObjectMetadata{
		Key:          key,
		Size:         fileInfo.Size(),
		LastModified: fileInfo.ModTime(),
		ContentType:  contentType,
		StorageClass: "STANDARD",
		Metadata:     make(map[string]string),
		Tags:         make(map[string]string),
	}, nil
}

// SetMetadata sets metadata for a file (limited implementation for local storage)
func (ls *LocalStorage) SetMetadata(ctx context.Context, key string, metadata map[string]string) error {
	// Local storage doesn't support arbitrary metadata
	// We could store it in a separate file or use extended attributes
	// For now, just validate that the file exists
	if _, err := ls.GetMetadata(ctx, key); err != nil {
		return err
	}

	ls.logger.Warn().Str("key", key).Msg("Local storage metadata setting is not supported")
	return nil
}

// BulkUpload uploads multiple files
func (ls *LocalStorage) BulkUpload(ctx context.Context, files []BulkUploadFile) ([]*UploadResult, error) {
	results := make([]*UploadResult, len(files))

	for i, file := range files {
		result, err := ls.Upload(ctx, file.Key, file.Data, file.ContentType, file.Options)
		if err != nil {
			ls.logger.Error().Err(err).Str("key", file.Key).Msg("Failed to upload file in bulk operation")
			// Continue with other files even if one fails
			results[i] = nil
		} else {
			results[i] = result
		}
	}

	return results, nil
}

// BulkDelete deletes multiple files
func (ls *LocalStorage) BulkDelete(ctx context.Context, keys []string) (*BulkDeleteResult, error) {
	result := &BulkDeleteResult{
		Deleted: make([]string, 0),
		Errors:  make([]DeleteError, 0),
	}

	for _, key := range keys {
		err := ls.Delete(ctx, key)
		if err != nil {
			result.Errors = append(result.Errors, DeleteError{
				Key:   key,
				Error: err.Error(),
			})
		} else {
			result.Deleted = append(result.Deleted, key)
		}
	}

	result.Total = len(keys)

	return result, nil
}

// GetProviderType returns the provider type
func (ls *LocalStorage) GetProviderType() string {
	return "local"
}

// HealthCheck checks the health of the local storage
func (ls *LocalStorage) HealthCheck(ctx context.Context) error {
	// Try to create a test file
	testKey := ".health-check"
	testPath := filepath.Join(ls.basePath, testKey)

	file, err := os.Create(testPath)
	if err != nil {
		return fmt.Errorf("failed to create health check file: %w", err)
	}
	file.Close()

	// Delete the test file
	if err := os.Remove(testPath); err != nil {
		return fmt.Errorf("failed to remove health check file: %w", err)
	}

	return nil
}

// validateKey validates the storage key
func (ls *LocalStorage) validateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	// Validate path to prevent directory traversal attacks
	if err := security.ValidatePath(key, ls.basePath); err != nil {
		return ErrInvalidKey
	}

	// Check for invalid characters
	invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range invalidChars {
		if strings.Contains(key, char) {
			return ErrInvalidKey
		}
	}

	return nil
}
