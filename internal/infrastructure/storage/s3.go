package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rs/zerolog"
)

// S3Storage implements StorageProvider for AWS S3 storage
type S3Storage struct {
	client    *s3.Client
	uploader  *manager.Uploader
	downloader *manager.Downloader
	bucket    string
	publicURL string
	region    string
	logger    zerolog.Logger
	config    *StorageConfig
}

// NewS3Storage creates a new S3 storage provider
func NewS3Storage(config *StorageConfig, logger zerolog.Logger) (*S3Storage, error) {
	if config.Bucket == "" {
		return nil, fmt.Errorf("bucket is required for S3 storage")
	}

	// Load AWS configuration
	cfg, err := loadAWSConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		// Set custom endpoint if provided (for MinIO, etc.)
		if config.Endpoint != "" {
			o.BaseEndpoint = aws.String(config.Endpoint)
			o.UsePathStyle = true // Required for MinIO
		}
	})

	// Create uploader and downloader with default configurations
	uploader := manager.NewUploader(client)
	downloader := manager.NewDownloader(client)

	return &S3Storage{
		client:    client,
		uploader:  uploader,
		downloader: downloader,
		bucket:    config.Bucket,
		publicURL: config.PublicURL,
		region:    config.Region,
		logger:    logger,
		config:    config,
	}, nil
}

// loadAWSConfig loads AWS configuration based on provided config
func loadAWSConfig(config *StorageConfig) (aws.Config, error) {
	// Use the awsconfig package to load configuration
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(config.Region),
	)
	if err != nil {
		return aws.Config{}, err
	}

	// If access key and secret key are provided
	if config.AccessKey != "" && config.SecretKey != "" {
		cfg.Credentials = credentials.NewStaticCredentialsProvider(config.AccessKey, config.SecretKey, "")
	}

	// If custom endpoint is provided (for MinIO, etc.), we need to set it after loading
	if config.Endpoint != "" {
		return cfg, nil // We'll handle the custom endpoint in the S3 client
	}

	return cfg, nil
}

// Upload uploads a file to S3
func (s *S3Storage) Upload(ctx context.Context, key string, data io.Reader, contentType string, options *UploadOptions) (*UploadResult, error) {
	// Validate key
	if err := s.validateKey(key); err != nil {
		return nil, err
	}

	// Set default content type
	if contentType == "" {
		contentType = mime.TypeByExtension(key[strings.LastIndex(key, "."):])
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Build upload input
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        data,
		ContentType: aws.String(contentType),
	}

	// Apply options
	if options != nil {
		if options.CacheControl != "" {
			input.CacheControl = aws.String(options.CacheControl)
		}
		if options.ContentEncoding != "" {
			input.ContentEncoding = aws.String(options.ContentEncoding)
		}
		if options.Metadata != nil {
			input.Metadata = options.Metadata
		}
		if options.Tags != nil {
			var tagSet []types.Tag
			for k, v := range options.Tags {
				tagSet = append(tagSet, types.Tag{
					Key:   aws.String(k),
					Value: aws.String(v),
				})
			}
			input.Tagging = aws.String(buildTagString(options.Tags))
		}
		if options.ACL != "" {
			input.ACL = types.ObjectCannedACL(options.ACL)
		}

		// Enable server-side encryption if configured
		if s.config.Encryption {
			input.ServerSideEncryption = types.ServerSideEncryptionAes256
		}
	}

	// Upload file
	_, uploadErr := s.uploader.Upload(ctx, input)
	if uploadErr != nil {
		s.logger.Error().Err(uploadErr).
			Str("bucket", s.bucket).
			Str("key", key).
			Msg("Failed to upload file to S3")
		return nil, fmt.Errorf("failed to upload file to S3: %w", uploadErr)
	}

	// Get object metadata
	headInput := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	headOutput, err := s.client.HeadObject(ctx, headInput)
	if err != nil {
		s.logger.Warn().Err(err).
			Str("bucket", s.bucket).
			Str("key", key).
			Msg("Failed to get object metadata after upload")
	}

	url, err := s.GetURL(ctx, key)
	if err != nil {
		return nil, err
	}

	size := int64(0)
	if headOutput.ContentLength != nil {
		size = *headOutput.ContentLength
	}

	uploadResult := &UploadResult{
		Key:          key,
		URL:          url,
		Size:         size,
		ContentType:  contentType,
		ETag:         aws.ToString(headOutput.ETag),
		LastModified: aws.ToTime(headOutput.LastModified),
		Metadata:     headOutput.Metadata,
	}

	s.logger.Info().
		Str("bucket", s.bucket).
		Str("key", key).
		Int64("size", uploadResult.Size).
		Str("etag", uploadResult.ETag).
		Msg("File uploaded to S3 successfully")

	return uploadResult, nil
}

// Download downloads a file from S3
func (s *S3Storage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	// Validate key
	if err := s.validateKey(key); err != nil {
		return nil, err
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		if isNotFoundError(err) {
			return nil, ErrObjectNotFound
		}
		s.logger.Error().Err(err).
			Str("bucket", s.bucket).
			Str("key", key).
			Msg("Failed to download file from S3")
		return nil, fmt.Errorf("failed to download file from S3: %w", err)
	}

	s.logger.Debug().
		Str("bucket", s.bucket).
		Str("key", key).
		Msg("File downloaded from S3 successfully")

	return result.Body, nil
}

// Delete deletes a file from S3
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	// Validate key
	if err := s.validateKey(key); err != nil {
		return err
	}

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		s.logger.Error().Err(err).
			Str("bucket", s.bucket).
			Str("key", key).
			Msg("Failed to delete file from S3")
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	s.logger.Info().
		Str("bucket", s.bucket).
		Str("key", key).
		Msg("File deleted from S3 successfully")

	return nil
}

// Exists checks if a file exists in S3
func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	// Validate key
	if err := s.validateKey(key); err != nil {
		return false, err
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.HeadObject(ctx, input)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		s.logger.Error().Err(err).
			Str("bucket", s.bucket).
			Str("key", key).
			Msg("Failed to check file existence in S3")
		return false, fmt.Errorf("failed to check file existence in S3: %w", err)
	}

	return true, nil
}

// GetURL returns the URL for a file
func (s *S3Storage) GetURL(ctx context.Context, key string) (string, error) {
	if s.publicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(s.publicURL, "/"), key), nil
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key), nil
}

// GetPresignedURL returns a presigned URL for a file
func (s *S3Storage) GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	// Validate key
	if err := s.validateKey(key); err != nil {
		return "", err
	}

	presignClient := s3.NewPresignClient(s.client)

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	result, err := presignClient.PresignGetObject(ctx, input, func(po *s3.PresignOptions) {
		po.Expires = expiration
	})
	if err != nil {
		s.logger.Error().Err(err).
			Str("bucket", s.bucket).
			Str("key", key).
			Dur("expiration", expiration).
			Msg("Failed to generate presigned URL")
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return result.URL, nil
}

// List lists objects in S3
func (s *S3Storage) List(ctx context.Context, prefix string, options *ListOptions) ([]*StorageObject, error) {
	result, err := s.ListWithPagination(ctx, prefix, options)
	if err != nil {
		return nil, err
	}
	return result.Objects, nil
}

// ListWithPagination lists objects in S3 with pagination
func (s *S3Storage) ListWithPagination(ctx context.Context, prefix string, options *ListOptions) (*ListResult, error) {
	if options == nil {
		options = &ListOptions{}
	}

	// Validate MaxKeys is within int32 range
	maxKeys := options.MaxKeys
	if maxKeys > 0x7FFFFFFF || maxKeys < 0 {
		maxKeys = 1000 // Default to reasonable limit
	}
	
	input := &s3.ListObjectsV2Input{
		Bucket:            aws.String(s.bucket),
		Prefix:            aws.String(prefix),
		Delimiter:         aws.String(options.Delimiter),
		MaxKeys:           aws.Int32(int32(maxKeys)), // #nosec G115 - Validated above
		ContinuationToken: aws.String(options.ContinuationToken),
	}

	if !options.Recursive {
		input.Delimiter = aws.String("/")
	}

	result, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		s.logger.Error().Err(err).
			Str("bucket", s.bucket).
			Str("prefix", prefix).
			Msg("Failed to list objects in S3")
		return nil, fmt.Errorf("failed to list objects in S3: %w", err)
	}

	objects := make([]*StorageObject, len(result.Contents))
	for i, obj := range result.Contents {
		size := int64(0)
		if obj.Size != nil {
			size = *obj.Size
		}

		objects[i] = &StorageObject{
			Key:          aws.ToString(obj.Key),
			Size:         size,
			LastModified: aws.ToTime(obj.LastModified),
			ETag:         aws.ToString(obj.ETag),
			StorageClass: string(obj.StorageClass),
			IsDir:        false,
		}
	}

	var prefixes []string
	for _, p := range result.CommonPrefixes {
		prefixes = append(prefixes, aws.ToString(p.Prefix))
	}

	isTruncated := false
	if result.IsTruncated != nil {
		isTruncated = *result.IsTruncated
	}

	resultMaxKeys := 1000 // default
	if result.MaxKeys != nil {
		resultMaxKeys = int(*result.MaxKeys)
	}

	return &ListResult{
		Objects:              objects,
		Prefixes:             prefixes,
		IsTruncated:          isTruncated,
		NextContinuationToken: aws.ToString(result.NextContinuationToken),
		MaxKeys:              resultMaxKeys,
		CommonPrefixes:       prefixes,
	}, nil
}

// GetMetadata gets metadata for a file in S3
func (s *S3Storage) GetMetadata(ctx context.Context, key string) (*ObjectMetadata, error) {
	// Validate key
	if err := s.validateKey(key); err != nil {
		return nil, err
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.HeadObject(ctx, input)
	if err != nil {
		if isNotFoundError(err) {
			return nil, ErrObjectNotFound
		}
		s.logger.Error().Err(err).
			Str("bucket", s.bucket).
			Str("key", key).
			Msg("Failed to get object metadata from S3")
		return nil, fmt.Errorf("failed to get object metadata from S3: %w", err)
	}

	size := int64(0)
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	return &ObjectMetadata{
		Key:            key,
		Size:           size,
		LastModified:   aws.ToTime(result.LastModified),
		ContentType:    aws.ToString(result.ContentType),
		ETag:           aws.ToString(result.ETag),
		StorageClass:   string(result.StorageClass),
		Metadata:       result.Metadata,
		CacheControl:   aws.ToString(result.CacheControl),
		ContentEncoding: aws.ToString(result.ContentEncoding),
		Tags:           make(map[string]string), // TODO: Parse actual tags from result
	}, nil
}

// SetMetadata sets metadata for a file in S3
func (s *S3Storage) SetMetadata(ctx context.Context, key string, metadata map[string]string) error {
	// Validate key
	if err := s.validateKey(key); err != nil {
		return err
	}

	// Copy object to itself with new metadata
	copySource := fmt.Sprintf("%s/%s", s.bucket, key)
	copyInput := &s3.CopyObjectInput{
		Bucket:            aws.String(s.bucket),
		Key:               aws.String(key),
		CopySource:        aws.String(copySource),
		Metadata:          metadata,
		MetadataDirective: types.MetadataDirectiveReplace,
	}

	_, err := s.client.CopyObject(ctx, copyInput)
	if err != nil {
		s.logger.Error().Err(err).
			Str("bucket", s.bucket).
			Str("key", key).
			Msg("Failed to set object metadata in S3")
		return fmt.Errorf("failed to set object metadata in S3: %w", err)
	}

	return nil
}

// BulkUpload uploads multiple files to S3
func (s *S3Storage) BulkUpload(ctx context.Context, files []BulkUploadFile) ([]*UploadResult, error) {
	results := make([]*UploadResult, len(files))

	for i, file := range files {
		result, err := s.Upload(ctx, file.Key, file.Data, file.ContentType, file.Options)
		if err != nil {
			s.logger.Error().Err(err).Str("key", file.Key).Msg("Failed to upload file in bulk operation to S3")
			results[i] = nil
		} else {
			results[i] = result
		}
	}

	return results, nil
}

// BulkDelete deletes multiple files from S3
func (s *S3Storage) BulkDelete(ctx context.Context, keys []string) (*BulkDeleteResult, error) {
	// S3 supports deleting up to 1000 objects in a single request
	maxBatchSize := 1000
	var allErrors []types.Error
	var deletedKeys []string

	for i := 0; i < len(keys); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		objectIds := make([]types.ObjectIdentifier, len(batch))
		for j, key := range batch {
			objectIds[j] = types.ObjectIdentifier{
				Key: aws.String(key),
			}
		}

		input := &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{
				Objects: objectIds,
				Quiet:   aws.Bool(true),
			},
		}

		result, err := s.client.DeleteObjects(ctx, input)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to delete objects in bulk from S3")
			return nil, fmt.Errorf("failed to delete objects in bulk from S3: %w", err)
		}

		if len(result.Errors) > 0 {
			allErrors = append(allErrors, result.Errors...)
		}

		if len(result.Deleted) > 0 {
			for _, deleted := range result.Deleted {
				deletedKeys = append(deletedKeys, aws.ToString(deleted.Key))
			}
		}
	}

	// Convert errors to our format
	errors := make([]DeleteError, len(allErrors))
	for i, err := range allErrors {
		errors[i] = DeleteError{
			Key:   aws.ToString(err.Key),
			Error: aws.ToString(err.Message),
		}
	}

	return &BulkDeleteResult{
		Deleted: deletedKeys,
		Errors:  errors,
		Total:   len(keys),
	}, nil
}

// GetProviderType returns the provider type
func (s *S3Storage) GetProviderType() string {
	return "s3"
}

// HealthCheck checks the health of the S3 storage
func (s *S3Storage) HealthCheck(ctx context.Context) error {
	input := &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	}

	_, err := s.client.HeadBucket(ctx, input)
	if err != nil {
		return fmt.Errorf("S3 bucket health check failed: %w", err)
	}

	return nil
}

// validateKey validates the storage key
func (s *S3Storage) validateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	// S3 doesn't allow keys starting or ending with slashes
	if strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") {
		return ErrInvalidKey
	}

	return nil
}

// isNotFoundError checks if error is a not found error
func isNotFoundError(err error) bool {
	var noSuchKey *types.NoSuchKey
	var noSuchBucket *types.NoSuchBucket

	return err.Error() == "NoSuchKey" ||
		   err.Error() == "NoSuchBucket" ||
		   noSuchKey != nil ||
		   noSuchBucket != nil
}

// buildTagString builds a tag string from a map
func buildTagString(tags map[string]string) string {
	var tagPairs []string
	for k, v := range tags {
		// URL encode key and value
		tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v)))
	}
	return strings.Join(tagPairs, "&")
}

// parseTagString parses a tag string into a map
func parseTagString(tagString string) map[string]string {
	tags := make(map[string]string)
	if tagString == "" {
		return tags
	}

	pairs := strings.Split(tagString, "&")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			key, _ := url.QueryUnescape(kv[0])
			value, _ := url.QueryUnescape(kv[1])
			tags[key] = value
		}
	}

	return tags
}