package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"erpgo/pkg/cache"
)

// APIKey represents an API key with associated metadata
type APIKey struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	KeyHash     string                 `json:"-" db:"key_hash"` // Never return the actual key
	KeyPrefix   string                 `json:"key_prefix" db:"key_prefix"`
	UserID      uuid.UUID              `json:"user_id" db:"user_id"`
	Roles       []string               `json:"roles" db:"roles"`
	Permissions []string               `json:"permissions" db:"permissions"`
	IsActive    bool                   `json:"is_active" db:"is_active"`
	ExpiresAt   *time.Time             `json:"expires_at" db:"expires_at"`
	LastUsedAt  *time.Time             `json:"last_used_at" db:"last_used_at"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	CreatedBy   uuid.UUID              `json:"created_by" db:"created_by"`
	Description string                 `json:"description" db:"description"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
}

// APIKeyRepository defines the interface for API key storage
type APIKeyRepository interface {
	Create(ctx context.Context, key *APIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*APIKey, error)
	GetByHash(ctx context.Context, keyHash string) (*APIKey, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*APIKey, error)
	Update(ctx context.Context, key *APIKey) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter APIKeyFilter) ([]*APIKey, error)
	Deactivate(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID, lastUsed time.Time) error
}

// APIKeyFilter defines filtering options for API keys
type APIKeyFilter struct {
	UserID        *uuid.UUID
	IsActive      *bool
	ExpiresBefore *time.Time
	ExpiresAfter  *time.Time
	CreatedBy     *uuid.UUID
	Roles         []string
	Limit         int
	Offset        int
}

// APIKeyService provides business logic for API key management
type APIKeyService struct {
	repository APIKeyRepository
	cache      cache.Cache
	logger     zerolog.Logger
	config     APIKeyConfig
}

// APIKeyConfig holds configuration for API key management
type APIKeyConfig struct {
	// Key generation settings
	KeyLength       int           `json:"key_length"`
	KeyPrefixLength int           `json:"key_prefix_length"`
	DefaultExpiry   time.Duration `json:"default_expiry"`
	MaxExpiry       time.Duration `json:"max_expiry"`

	// Security settings
	MinNameLength  int `json:"min_name_length"`
	MaxNameLength  int `json:"max_name_length"`
	MaxKeysPerUser int `json:"max_keys_per_user"`

	// Cache settings
	CacheEnabled bool          `json:"cache_enabled"`
	CacheTTL     time.Duration `json:"cache_ttl"`

	// Audit settings
	LogKeyUsage    bool `json:"log_key_usage"`
	LogKeyCreation bool `json:"log_key_creation"`
	LogKeyDeletion bool `json:"log_key_deletion"`
}

// DefaultAPIKeyConfig returns a secure default configuration
func DefaultAPIKeyConfig() APIKeyConfig {
	return APIKeyConfig{
		KeyLength:       64,
		KeyPrefixLength: 8,
		DefaultExpiry:   365 * 24 * time.Hour,     // 1 year
		MaxExpiry:       5 * 365 * 24 * time.Hour, // 5 years
		MinNameLength:   3,
		MaxNameLength:   100,
		MaxKeysPerUser:  10,
		CacheEnabled:    true,
		CacheTTL:        15 * time.Minute,
		LogKeyUsage:     true,
		LogKeyCreation:  true,
		LogKeyDeletion:  true,
	}
}

// NewAPIKeyService creates a new API key service
func NewAPIKeyService(repository APIKeyRepository, cache cache.Cache, logger zerolog.Logger, config APIKeyConfig) *APIKeyService {
	return &APIKeyService{
		repository: repository,
		cache:      cache,
		logger:     logger,
		config:     config,
	}
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name        string                 `json:"name" validate:"required,min=3,max=100"`
	Description string                 `json:"description" validate:"max=500"`
	UserID      uuid.UUID              `json:"user_id" validate:"required"`
	Roles       []string               `json:"roles"`
	Permissions []string               `json:"permissions"`
	ExpiresAt   *time.Time             `json:"expires_at"`
	CreatedBy   uuid.UUID              `json:"created_by" validate:"required"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// CreateAPIKeyResponse represents the response when creating an API key
type CreateAPIKeyResponse struct {
	APIKey    *APIKey `json:"api_key"`
	PlainText string  `json:"plain_text_key"` // Only returned once during creation
}

// ValidateAPIKeyRequest represents a request to validate an API key
type ValidateAPIKeyRequest struct {
	APIKey string `json:"api_key" validate:"required"`
}

// ValidateAPIKeyResponse represents the response when validating an API key
type ValidateAPIKeyResponse struct {
	Valid     bool    `json:"valid"`
	APIKey    *APIKey `json:"api_key,omitempty"`
	ExpiresIn int64   `json:"expires_in,omitempty"`
	Message   string  `json:"message,omitempty"`
}

// CreateAPIKey creates a new API key
func (s *APIKeyService) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	// Validate request
	if err := s.validateCreateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Generate API key
	plainTextKey, keyHash, keyPrefix, err := s.generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Set default expiry if not provided
	expiresAt := req.ExpiresAt
	if expiresAt == nil {
		exp := time.Now().Add(s.config.DefaultExpiry)
		expiresAt = &exp
	}

	// Ensure expiry doesn't exceed maximum
	maxExpiry := time.Now().Add(s.config.MaxExpiry)
	if expiresAt.After(maxExpiry) {
		expiresAt = &maxExpiry
	}

	// Create API key object
	apiKey := &APIKey{
		ID:          uuid.New(),
		Name:        req.Name,
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		UserID:      req.UserID,
		Roles:       req.Roles,
		Permissions: req.Permissions,
		IsActive:    true,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now().UTC(),
		CreatedBy:   req.CreatedBy,
		Description: req.Description,
		Metadata:    req.Metadata,
	}

	// Store API key
	if err := s.repository.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	// Log creation if enabled
	if s.config.LogKeyCreation {
		s.logger.Info().
			Str("api_key_id", apiKey.ID.String()).
			Str("api_key_name", apiKey.Name).
			Str("user_id", apiKey.UserID.String()).
			Str("created_by", req.CreatedBy.String()).
			Time("expires_at", *apiKey.ExpiresAt).
			Msg("API key created")
	}

	// Invalidate cache for user
	if s.config.CacheEnabled {
		cacheKey := s.getUserCacheKey(apiKey.UserID)
		s.cache.Delete(ctx, cacheKey)
	}

	return &CreateAPIKeyResponse{
		APIKey:    apiKey,
		PlainText: plainTextKey,
	}, nil
}

// ValidateAPIKey validates an API key and returns the associated key information
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, req *ValidateAPIKeyRequest) (*ValidateAPIKeyResponse, error) {
	if req.APIKey == "" {
		return &ValidateAPIKeyResponse{
			Valid:   false,
			Message: "API key is required",
		}, nil
	}

	// Generate hash for lookup
	keyHash := s.hashAPIKey(req.APIKey)

	// Try cache first
	var apiKey *APIKey
	cacheKey := s.getHashCacheKey(keyHash)

	if s.config.CacheEnabled {
		var cachedKey APIKey
		if err := s.cache.GetJSON(ctx, cacheKey, &cachedKey); err == nil {
			apiKey = &cachedKey
		}
	}

	// If not in cache, query repository
	if apiKey == nil {
		var err error
		apiKey, err = s.repository.GetByHash(ctx, keyHash)
		if err != nil {
			return &ValidateAPIKeyResponse{
				Valid:   false,
				Message: "Invalid API key",
			}, nil
		}

		// Cache the result
		if s.config.CacheEnabled && apiKey != nil {
			s.cache.SetJSON(ctx, cacheKey, apiKey, s.config.CacheTTL)
		}
	}

	// Check if key is active
	if !apiKey.IsActive {
		return &ValidateAPIKeyResponse{
			Valid:   false,
			Message: "API key is inactive",
		}, nil
	}

	// Check if key has expired
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return &ValidateAPIKeyResponse{
			Valid:   false,
			Message: "API key has expired",
		}, nil
	}

	// Update last used timestamp asynchronously
	go func() {
		if err := s.repository.UpdateLastUsed(context.Background(), apiKey.ID, time.Now().UTC()); err != nil {
			s.logger.Error().Err(err).Str("api_key_id", apiKey.ID.String()).Msg("Failed to update last used timestamp")
		}
	}()

	// Log usage if enabled
	if s.config.LogKeyUsage {
		s.logger.Info().
			Str("api_key_id", apiKey.ID.String()).
			Str("api_key_name", apiKey.Name).
			Str("user_id", apiKey.UserID.String()).
			Msg("API key used")
	}

	// Calculate expiration
	var expiresIn int64
	if apiKey.ExpiresAt != nil {
		expiresIn = int64(apiKey.ExpiresAt.Sub(time.Now()).Seconds())
	}

	return &ValidateAPIKeyResponse{
		Valid:     true,
		APIKey:    apiKey,
		ExpiresIn: expiresIn,
	}, nil
}

// GetAPIKeys retrieves API keys for a user
func (s *APIKeyService) GetAPIKeys(ctx context.Context, userID uuid.UUID) ([]*APIKey, error) {
	// Try cache first
	if s.config.CacheEnabled {
		cacheKey := s.getUserCacheKey(userID)
		var cachedKeys []*APIKey
		if err := s.cache.GetJSON(ctx, cacheKey, &cachedKeys); err == nil {
			return cachedKeys, nil
		}
	}

	// Query repository
	keys, err := s.repository.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	// Cache the result
	if s.config.CacheEnabled {
		cacheKey := s.getUserCacheKey(userID)
		s.cache.SetJSON(ctx, cacheKey, keys, s.config.CacheTTL)
	}

	return keys, nil
}

// DeleteAPIKey deletes an API key
func (s *APIKeyService) DeleteAPIKey(ctx context.Context, keyID, userID uuid.UUID) error {
	// Get the key first to validate ownership
	apiKey, err := s.repository.GetByID(ctx, keyID)
	if err != nil {
		return fmt.Errorf("API key not found: %w", err)
	}

	// Verify ownership
	if apiKey.UserID != userID {
		return fmt.Errorf("access denied: API key does not belong to user")
	}

	// Delete the key
	if err := s.repository.Delete(ctx, keyID); err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	// Log deletion if enabled
	if s.config.LogKeyDeletion {
		s.logger.Info().
			Str("api_key_id", apiKey.ID.String()).
			Str("api_key_name", apiKey.Name).
			Str("user_id", apiKey.UserID.String()).
			Msg("API key deleted")
	}

	// Invalidate cache
	if s.config.CacheEnabled {
		// Invalidate user cache
		userCacheKey := s.getUserCacheKey(apiKey.UserID)
		s.cache.Delete(ctx, userCacheKey)

		// Invalidate hash cache
		hashCacheKey := s.getHashCacheKey(apiKey.KeyHash)
		s.cache.Delete(ctx, hashCacheKey)
	}

	return nil
}

// RevokeAPIKey revokes (deactivates) an API key
func (s *APIKeyService) RevokeAPIKey(ctx context.Context, keyID, userID uuid.UUID) error {
	// Get the key first to validate ownership
	apiKey, err := s.repository.GetByID(ctx, keyID)
	if err != nil {
		return fmt.Errorf("API key not found: %w", err)
	}

	// Verify ownership
	if apiKey.UserID != userID {
		return fmt.Errorf("access denied: API key does not belong to user")
	}

	// Deactivate the key
	apiKey.IsActive = false
	if err := s.repository.Update(ctx, apiKey); err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	// Log revocation
	s.logger.Info().
		Str("api_key_id", apiKey.ID.String()).
		Str("api_key_name", apiKey.Name).
		Str("user_id", apiKey.UserID.String()).
		Msg("API key revoked")

	// Invalidate cache
	if s.config.CacheEnabled {
		// Invalidate user cache
		userCacheKey := s.getUserCacheKey(apiKey.UserID)
		s.cache.Delete(ctx, userCacheKey)

		// Invalidate hash cache
		hashCacheKey := s.getHashCacheKey(apiKey.KeyHash)
		s.cache.Delete(ctx, hashCacheKey)
	}

	return nil
}

// generateAPIKey generates a new API key and its hash
func (s *APIKeyService) generateAPIKey() (string, string, string, error) {
	// Generate random bytes
	bytes := make([]byte, s.config.KeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to base64
	key := base64.URLEncoding.EncodeToString(bytes)

	// Create prefix
	prefixLength := s.config.KeyPrefixLength
	if prefixLength > len(key) {
		prefixLength = len(key)
	}
	prefix := key[:prefixLength]

	// Generate hash
	hash := s.hashAPIKey(key)

	return key, hash, prefix, nil
}

// hashAPIKey creates a secure hash of an API key
func (s *APIKeyService) hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// validateCreateRequest validates a create API key request
func (s *APIKeyService) validateCreateRequest(ctx context.Context, req *CreateAPIKeyRequest) error {
	// Validate name length
	if len(req.Name) < s.config.MinNameLength {
		return fmt.Errorf("name must be at least %d characters", s.config.MinNameLength)
	}
	if len(req.Name) > s.config.MaxNameLength {
		return fmt.Errorf("name must be at most %d characters", s.config.MaxNameLength)
	}

	// Check if user has too many keys
	existingKeys, err := s.repository.GetByUserID(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to check existing API keys: %w", err)
	}

	activeKeys := 0
	for _, key := range existingKeys {
		if key.IsActive && (key.ExpiresAt == nil || key.ExpiresAt.After(time.Now())) {
			activeKeys++
		}
	}

	if activeKeys >= s.config.MaxKeysPerUser {
		return fmt.Errorf("user has reached maximum number of API keys (%d)", s.config.MaxKeysPerUser)
	}

	// Validate expiry
	if req.ExpiresAt != nil {
		maxExpiry := time.Now().Add(s.config.MaxExpiry)
		if req.ExpiresAt.After(maxExpiry) {
			return fmt.Errorf("expiry date cannot be more than %s in the future", s.config.MaxExpiry)
		}
		if req.ExpiresAt.Before(time.Now()) {
			return fmt.Errorf("expiry date cannot be in the past")
		}
	}

	return nil
}

// getUserCacheKey returns the cache key for a user's API keys
func (s *APIKeyService) getUserCacheKey(userID uuid.UUID) string {
	return fmt.Sprintf("api_keys:user:%s", userID.String())
}

// getHashCacheKey returns the cache key for an API key hash
func (s *APIKeyService) getHashCacheKey(keyHash string) string {
	return fmt.Sprintf("api_keys:hash:%s", keyHash)
}

// InMemoryAPIKeyRepository provides an in-memory implementation for testing
type InMemoryAPIKeyRepository struct {
	keys map[uuid.UUID]*APIKey
}

// NewInMemoryAPIKeyRepository creates a new in-memory API key repository
func NewInMemoryAPIKeyRepository() *InMemoryAPIKeyRepository {
	return &InMemoryAPIKeyRepository{
		keys: make(map[uuid.UUID]*APIKey),
	}
}

// Create saves an API key
func (r *InMemoryAPIKeyRepository) Create(ctx context.Context, key *APIKey) error {
	r.keys[key.ID] = key
	return nil
}

// GetByID retrieves an API key by ID
func (r *InMemoryAPIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*APIKey, error) {
	key, exists := r.keys[id]
	if !exists {
		return nil, fmt.Errorf("API key not found")
	}
	return key, nil
}

// GetByHash retrieves an API key by hash
func (r *InMemoryAPIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	for _, key := range r.keys {
		if key.KeyHash == keyHash {
			return key, nil
		}
	}
	return nil, fmt.Errorf("API key not found")
}

// GetByUserID retrieves all API keys for a user
func (r *InMemoryAPIKeyRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*APIKey, error) {
	var keys []*APIKey
	for _, key := range r.keys {
		if key.UserID == userID {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// Update updates an API key
func (r *InMemoryAPIKeyRepository) Update(ctx context.Context, key *APIKey) error {
	r.keys[key.ID] = key
	return nil
}

// Delete deletes an API key
func (r *InMemoryAPIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(r.keys, id)
	return nil
}

// List retrieves API keys with filtering
func (r *InMemoryAPIKeyRepository) List(ctx context.Context, filter APIKeyFilter) ([]*APIKey, error) {
	var keys []*APIKey
	for _, key := range r.keys {
		if filter.UserID != nil && key.UserID != *filter.UserID {
			continue
		}
		if filter.IsActive != nil && key.IsActive != *filter.IsActive {
			continue
		}
		if filter.ExpiresBefore != nil && (key.ExpiresAt == nil || key.ExpiresAt.After(*filter.ExpiresBefore)) {
			continue
		}
		if filter.ExpiresAfter != nil && (key.ExpiresAt == nil || key.ExpiresAt.Before(*filter.ExpiresAfter)) {
			continue
		}
		if filter.CreatedBy != nil && key.CreatedBy != *filter.CreatedBy {
			continue
		}
		if len(filter.Roles) > 0 && !containsAll(filter.Roles, key.Roles) {
			continue
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// Deactivate deactivates an API key
func (r *InMemoryAPIKeyRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	key, exists := r.keys[id]
	if !exists {
		return fmt.Errorf("API key not found")
	}
	key.IsActive = false
	return nil
}

// UpdateLastUsed updates the last used timestamp
func (r *InMemoryAPIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID, lastUsed time.Time) error {
	key, exists := r.keys[id]
	if !exists {
		return fmt.Errorf("API key not found")
	}
	key.LastUsedAt = &lastUsed
	return nil
}

// Helper function to check if slice contains all elements from another slice
func containsAll(required, actual []string) bool {
	requiredMap := make(map[string]bool)
	for _, req := range required {
		requiredMap[req] = true
	}

	for _, act := range actual {
		delete(requiredMap, act)
	}

	return len(requiredMap) == 0
}
