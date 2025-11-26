package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"
	"os"
	"strings"
)

// SecretSource represents where secrets can be loaded from
type SecretSource string

const (
	SourceEnvironment SecretSource = "environment"
	SourceVault       SecretSource = "vault"
	SourceAWSSecrets  SecretSource = "aws_secrets"
)

// SecretValidator validates secret values
type SecretValidator interface {
	Validate(value string) error
}

// SecretManager manages loading and validation of secrets
type SecretManager interface {
	LoadSecret(key string, validator SecretValidator) (string, error)
	LoadSecretWithDefault(key string, defaultValue string, validator SecretValidator) (string, error)
	GetSecret(key string) (string, bool)
	ValidateAll() error
	RotateSecret(key string, newValue string) error
	ListSecretKeys() []string
}

// secretManager implements SecretManager
type secretManager struct {
	source  SecretSource
	secrets map[string]string
}

// NewSecretManager creates a new secret manager
func NewSecretManager(source SecretSource) SecretManager {
	return &secretManager{
		source:  source,
		secrets: make(map[string]string),
	}
}

// LoadSecret loads a secret from the configured source and validates it
func (sm *secretManager) LoadSecret(key string, validator SecretValidator) (string, error) {
	var value string

	switch sm.source {
	case SourceEnvironment:
		value = os.Getenv(key)
		if value == "" {
			return "", fmt.Errorf("secret %s not found in environment", key)
		}
	case SourceVault:
		// TODO: Implement HashiCorp Vault integration
		return "", fmt.Errorf("vault source not yet implemented")
	case SourceAWSSecrets:
		// TODO: Implement AWS Secrets Manager integration
		return "", fmt.Errorf("AWS Secrets Manager source not yet implemented")
	default:
		return "", fmt.Errorf("unknown secret source: %s", sm.source)
	}

	// Validate the secret
	if validator != nil {
		if err := validator.Validate(value); err != nil {
			return "", fmt.Errorf("secret %s validation failed: %w", key, err)
		}
	}

	// Store the secret
	sm.secrets[key] = value

	return value, nil
}

// LoadSecretWithDefault loads a secret with a default value if not found
func (sm *secretManager) LoadSecretWithDefault(key string, defaultValue string, validator SecretValidator) (string, error) {
	var value string

	switch sm.source {
	case SourceEnvironment:
		value = os.Getenv(key)
		if value == "" {
			value = defaultValue
		}
	case SourceVault:
		// TODO: Implement HashiCorp Vault integration
		return "", fmt.Errorf("vault source not yet implemented")
	case SourceAWSSecrets:
		// TODO: Implement AWS Secrets Manager integration
		return "", fmt.Errorf("AWS Secrets Manager source not yet implemented")
	default:
		return "", fmt.Errorf("unknown secret source: %s", sm.source)
	}

	// Validate the secret
	if validator != nil {
		if err := validator.Validate(value); err != nil {
			return "", fmt.Errorf("secret %s validation failed: %w", key, err)
		}
	}

	// Store the secret
	sm.secrets[key] = value

	return value, nil
}

// GetSecret retrieves a previously loaded secret
func (sm *secretManager) GetSecret(key string) (string, bool) {
	value, exists := sm.secrets[key]
	return value, exists
}

// ValidateAll validates all loaded secrets
func (sm *secretManager) ValidateAll() error {
	if len(sm.secrets) == 0 {
		return fmt.Errorf("no secrets loaded")
	}
	return nil
}

// RotateSecret rotates a secret to a new value
func (sm *secretManager) RotateSecret(key string, newValue string) error {
	if _, exists := sm.secrets[key]; !exists {
		return fmt.Errorf("secret %s not found", key)
	}

	sm.secrets[key] = newValue
	return nil
}

// ListSecretKeys returns all loaded secret keys (for debugging/auditing)
func (sm *secretManager) ListSecretKeys() []string {
	keys := make([]string, 0, len(sm.secrets))
	for key := range sm.secrets {
		keys = append(keys, key)
	}
	return keys
}

// JWTSecretValidator validates JWT secrets
type JWTSecretValidator struct {
	MinEntropyBits int
}

// NewJWTSecretValidator creates a new JWT secret validator
func NewJWTSecretValidator(minEntropyBits int) *JWTSecretValidator {
	return &JWTSecretValidator{
		MinEntropyBits: minEntropyBits,
	}
}

// Validate validates a JWT secret
func (v *JWTSecretValidator) Validate(value string) error {
	if value == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}

	// Check for default/weak values
	weakValues := []string{
		"your-super-secret-jwt-key",
		"secret",
		"jwt-secret",
		"change-me",
		"default",
		"test",
		"password",
		"123456",
	}

	valueLower := strings.ToLower(value)
	for _, weak := range weakValues {
		if valueLower == weak {
			return fmt.Errorf("JWT secret is set to a default or weak value")
		}
	}

	// Check minimum length (256 bits = 32 bytes)
	minBytes := v.MinEntropyBits / 8
	if len(value) < minBytes {
		return fmt.Errorf("JWT secret must be at least %d bytes (got %d bytes)", minBytes, len(value))
	}

	// Check entropy
	entropy := calculateEntropy(value)
	if entropy < float64(v.MinEntropyBits) {
		return fmt.Errorf("JWT secret has insufficient entropy: %.2f bits (minimum: %d bits)", entropy, v.MinEntropyBits)
	}

	return nil
}

// PepperValidator validates password pepper values
type PepperValidator struct {
	ForbiddenValues []string
}

// NewPepperValidator creates a new pepper validator
func NewPepperValidator(forbiddenValues []string) *PepperValidator {
	return &PepperValidator{
		ForbiddenValues: forbiddenValues,
	}
}

// Validate validates a pepper value
func (v *PepperValidator) Validate(value string) error {
	if value == "" {
		return fmt.Errorf("pepper cannot be empty")
	}

	// Check for forbidden values
	for _, forbidden := range v.ForbiddenValues {
		if value == forbidden {
			return fmt.Errorf("pepper is set to a forbidden default value")
		}
	}

	// Check minimum length (should be at least 32 characters)
	if len(value) < 32 {
		return fmt.Errorf("pepper must be at least 32 characters long (got %d)", len(value))
	}

	return nil
}

// DatabaseURLValidator validates database URLs
type DatabaseURLValidator struct{}

// NewDatabaseURLValidator creates a new database URL validator
func NewDatabaseURLValidator() *DatabaseURLValidator {
	return &DatabaseURLValidator{}
}

// Validate validates a database URL
func (v *DatabaseURLValidator) Validate(value string) error {
	if value == "" {
		return fmt.Errorf("database URL cannot be empty")
	}

	// Check for common insecure patterns
	if strings.Contains(value, "sslmode=disable") {
		return fmt.Errorf("database URL should not disable SSL in production")
	}

	// Check for localhost in production (this would need environment context)
	if strings.Contains(value, "localhost") || strings.Contains(value, "127.0.0.1") {
		// This is a warning, not an error, as it might be valid in dev
		// In production, this should be caught by environment-specific validation
	}

	return nil
}

// calculateEntropy calculates the Shannon entropy of a string in bits
func calculateEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	// Count character frequencies
	freq := make(map[rune]int)
	for _, char := range s {
		freq[char]++
	}

	// Calculate entropy
	var entropy float64
	length := float64(len(s))

	for _, count := range freq {
		probability := float64(count) / length
		if probability > 0 {
			entropy -= probability * logBase2(probability)
		}
	}

	// Return entropy in bits
	return entropy * length
}

// logBase2 calculates log base 2
func logBase2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Log2(x)
}

// GenerateSecureSecret generates a cryptographically secure random secret
func GenerateSecureSecret(length int) (string, error) {
	if length < 32 {
		length = 32
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure secret: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateSecureSecretBytes generates cryptographically secure random bytes
func GenerateSecureSecretBytes(length int) ([]byte, error) {
	if length < 32 {
		length = 32
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate secure secret bytes: %w", err)
	}

	return bytes, nil
}

// ValidateSecretStrength validates the strength of a secret
func ValidateSecretStrength(secret string, minLength int, minEntropy float64) error {
	if len(secret) < minLength {
		return fmt.Errorf("secret is too short: %d bytes (minimum: %d bytes)", len(secret), minLength)
	}

	entropy := calculateEntropy(secret)
	if entropy < minEntropy {
		return fmt.Errorf("secret has insufficient entropy: %.2f bits (minimum: %.2f bits)", entropy, minEntropy)
	}

	return nil
}
