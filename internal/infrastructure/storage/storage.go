package storage

import (
	"context"
	"io"
	"time"
)

// StorageProvider defines the interface for storage providers
type StorageProvider interface {
	// File operations
	Upload(ctx context.Context, key string, data io.Reader, contentType string, options *UploadOptions) (*UploadResult, error)
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// URL operations
	GetURL(ctx context.Context, key string) (string, error)
	GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)

	// List operations
	List(ctx context.Context, prefix string, options *ListOptions) ([]*StorageObject, error)
	ListWithPagination(ctx context.Context, prefix string, options *ListOptions) (*ListResult, error)

	// Metadata operations
	GetMetadata(ctx context.Context, key string) (*ObjectMetadata, error)
	SetMetadata(ctx context.Context, key string, metadata map[string]string) error

	// Bulk operations
	BulkUpload(ctx context.Context, files []BulkUploadFile) ([]*UploadResult, error)
	BulkDelete(ctx context.Context, keys []string) (*BulkDeleteResult, error)

	// Provider-specific operations
	GetProviderType() string
	HealthCheck(ctx context.Context) error
}

// UploadOptions contains options for uploading files
type UploadOptions struct {
	ContentType     string            `json:"content_type,omitempty"`
	CacheControl    string            `json:"cache_control,omitempty"`
	ContentEncoding string            `json:"content_encoding,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Tags            map[string]string `json:"tags,omitempty"`
	ACL             string            `json:"acl,omitempty"` // For S3: private, public-read, etc.
}

// UploadResult represents the result of an upload operation
type UploadResult struct {
	Key         string            `json:"key"`
	URL         string            `json:"url"`
	Size        int64             `json:"size"`
	ContentType string            `json:"content_type"`
	ETag        string            `json:"etag,omitempty"`
	LastModified time.Time        `json:"last_modified"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ListOptions contains options for listing objects
type ListOptions struct {
	Prefix         string `json:"prefix,omitempty"`
	Delimiter      string `json:"delimiter,omitempty"`
	MaxKeys        int    `json:"max_keys,omitempty"`
	ContinuationToken string `json:"continuation_token,omitempty"`
	Recursive      bool   `json:"recursive"`
}

// StorageObject represents a storage object
type StorageObject struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"last_modified"`
	ETag         string            `json:"etag,omitempty"`
	StorageClass  string            `json:"storage_class,omitempty"`
	ContentType  string            `json:"content_type,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	IsDir        bool              `json:"is_dir"`
}

// ListResult represents the result of a list operation
type ListResult struct {
	Objects           []*StorageObject `json:"objects"`
	Prefixes          []string         `json:"prefixes"`
	IsTruncated       bool             `json:"is_truncated"`
	NextContinuationToken string        `json:"next_continuation_token,omitempty"`
	MaxKeys           int              `json:"max_keys"`
	CommonPrefixes    []string         `json:"common_prefixes,omitempty"`
}

// ObjectMetadata contains metadata about a storage object
type ObjectMetadata struct {
	Key           string            `json:"key"`
	Size          int64             `json:"size"`
	LastModified  time.Time         `json:"last_modified"`
	ContentType   string            `json:"content_type"`
	ETag          string            `json:"etag"`
	StorageClass  string            `json:"storage_class"`
	Metadata      map[string]string `json:"metadata"`
	Tags          map[string]string `json:"tags"`
	CacheControl  string            `json:"cache_control"`
	ContentEncoding string          `json:"content_encoding"`
}

// BulkUploadFile represents a file for bulk upload
type BulkUploadFile struct {
	Key          string         `json:"key"`
	Data         io.Reader      `json:"-"`
	ContentType  string         `json:"content_type,omitempty"`
	Options      *UploadOptions `json:"options,omitempty"`
}

// BulkDeleteResult represents the result of a bulk delete operation
type BulkDeleteResult struct {
	Deleted []string         `json:"deleted"`
	Errors  []DeleteError    `json:"errors"`
	Total   int              `json:"total"`
}

// DeleteError represents an error in bulk delete operation
type DeleteError struct {
	Key   string `json:"key"`
	Error string `json:"error"`
}

// StorageConfig contains configuration for storage providers
type StorageConfig struct {
	Provider     string                 `json:"provider" yaml:"provider"`
	Bucket       string                 `json:"bucket" yaml:"bucket"`
	Region       string                 `json:"region" yaml:"region"`
	Endpoint     string                 `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	AccessKey    string                 `json:"access_key,omitempty" yaml:"access_key,omitempty"`
	SecretKey    string                 `json:"secret_key,omitempty" yaml:"secret_key,omitempty"`
	Encryption   bool                   `json:"encryption,omitempty" yaml:"encryption,omitempty"`
	PublicURL    string                 `json:"public_url,omitempty" yaml:"public_url,omitempty"`
	CustomConfig map[string]interface{} `json:"custom_config,omitempty" yaml:"custom_config,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return e.Message
}

// Error types
var (
	ErrObjectNotFound    = &ValidationError{Field: "key", Message: "object not found"}
	ErrBucketNotFound    = &ValidationError{Field: "bucket", Message: "bucket not found"}
	ErrAccessDenied      = &ValidationError{Field: "access", Message: "access denied"}
	ErrInvalidKey        = &ValidationError{Field: "key", Message: "invalid key format"}
	ErrFileTooLarge      = &ValidationError{Field: "size", Message: "file too large"}
	ErrInvalidContentType = &ValidationError{Field: "content_type", Message: "invalid content type"}
	ErrStorageQuotaExceeded = &ValidationError{Field: "quota", Message: "storage quota exceeded"}
	ErrConnectionFailed = &ValidationError{Field: "connection", Message: "connection failed"}
)

// IsValidationError checks if error is a ValidationError
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

// IsNotFoundError checks if error is a not found error
func IsNotFoundError(err error) bool {
	if validationErr, ok := err.(*ValidationError); ok {
		return validationErr.Message == "object not found" || validationErr.Message == "bucket not found"
	}
	return false
}

// IsAccessDeniedError checks if error is an access denied error
func IsAccessDeniedError(err error) bool {
	if validationErr, ok := err.(*ValidationError); ok {
		return validationErr.Message == "access denied"
	}
	return false
}

// IsConnectionError checks if error is a connection error
func IsConnectionError(err error) bool {
	if validationErr, ok := err.(*ValidationError); ok {
		return validationErr.Message == "connection failed"
	}
	return false
}