package storage

import (
	"fmt"

	"github.com/rs/zerolog"
)

// Factory creates storage providers based on configuration
type Factory struct {
	logger zerolog.Logger
}

// NewFactory creates a new storage factory
func NewFactory(logger zerolog.Logger) *Factory {
	return &Factory{
		logger: logger,
	}
}

// CreateProvider creates a storage provider based on the configuration
func (f *Factory) CreateProvider(config *StorageConfig) (StorageProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("storage configuration is required")
	}

	switch config.Provider {
	case "local":
		return NewLocalStorage(config, f.logger)
	case "s3":
		return NewS3Storage(config, f.logger)
	case "minio":
		// MinIO uses the S3 interface but with different default configuration
		if config.Endpoint == "" {
			return nil, fmt.Errorf("endpoint is required for MinIO storage")
		}
		return NewS3Storage(config, f.logger)
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", config.Provider)
	}
}

// ProviderInfo contains information about a storage provider
type ProviderInfo struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Capabilities []string `json:"capabilities"`
	Features    []string `json:"features"`
}

// GetProviderInfo returns information about available storage providers
func (f *Factory) GetProviderInfo() map[string]ProviderInfo {
	return map[string]ProviderInfo{
		"local": {
			Name:        "Local Filesystem",
			Type:        "local",
			Capabilities: []string{"upload", "download", "delete", "list", "metadata", "bulk"},
			Features:    []string{"fast", "simple", "no external dependencies"},
		},
		"s3": {
			Name:        "Amazon S3",
			Type:        "s3",
			Capabilities: []string{"upload", "download", "delete", "list", "metadata", "bulk", "presigned_url"},
			Features:    []string{"scalable", "durable", "cdn", "encryption"},
		},
		"minio": {
			Name:        "MinIO S3-Compatible",
			Type:        "minio",
			Capabilities: []string{"upload", "download", "delete", "list", "metadata", "bulk", "presigned_url"},
			Features:    []string{"self-hosted", "s3-compatible", "scalable"},
		},
	}
}

// ValidateConfig validates storage configuration
func (f *Factory) ValidateConfig(config *StorageConfig) error {
	if config == nil {
		return fmt.Errorf("storage configuration is required")
	}

	if config.Provider == "" {
		return fmt.Errorf("storage provider is required")
	}

	switch config.Provider {
	case "local":
		return f.validateLocalConfig(config)
	case "s3", "minio":
		return f.validateS3Config(config)
	default:
		return fmt.Errorf("unsupported storage provider: %s", config.Provider)
	}
}

// validateLocalConfig validates local storage configuration
func (f *Factory) validateLocalConfig(config *StorageConfig) error {
	if config.Bucket == "" {
		return fmt.Errorf("bucket (base directory) is required for local storage")
	}

	return nil
}

// validateS3Config validates S3/MinIO storage configuration
func (f *Factory) validateS3Config(config *StorageConfig) error {
	if config.Bucket == "" {
		return fmt.Errorf("bucket name is required for S3 storage")
	}

	if config.Provider == "minio" && config.Endpoint == "" {
		return fmt.Errorf("endpoint is required for MinIO storage")
	}

	if config.Region == "" && config.Provider == "s3" {
		return fmt.Errorf("region is required for S3 storage")
	}

	return nil
}

// StorageMetrics contains metrics about storage operations
type StorageMetrics struct {
	Provider       string `json:"provider"`
	TotalUploads   int64  `json:"total_uploads"`
	TotalDownloads int64  `json:"total_downloads"`
	TotalDeletes   int64  `json:"total_deletes"`
	TotalBytesIn   int64  `json:"total_bytes_in"`
	TotalBytesOut  int64  `json:"total_bytes_out"`
	ErrorCount     int64  `json:"error_count"`
	LastOperation  string `json:"last_operation"`
	Healthy        bool   `json:"healthy"`
}

// GetMetrics returns storage metrics (placeholder implementation)
func (f *Factory) GetMetrics(provider StorageProvider) (*StorageMetrics, error) {
	// This is a placeholder implementation
	// In a real application, you would track metrics across operations
	// You could use Prometheus, DataDog, or other monitoring systems

	return &StorageMetrics{
		Provider:       provider.GetProviderType(),
		TotalUploads:   0,
		TotalDownloads: 0,
		TotalDeletes:   0,
		TotalBytesIn:   0,
		TotalBytesOut:  0,
		ErrorCount:     0,
		LastOperation:  "unknown",
		Healthy:        true,
	}, nil
}