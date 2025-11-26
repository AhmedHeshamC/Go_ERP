package secrets

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSecretWithDefault(t *testing.T) {
	sm := NewSecretManager(SourceEnvironment)

	t.Run("uses default when env var not set", func(t *testing.T) {
		os.Unsetenv("TEST_SECRET_DEFAULT")
		validator := NewJWTSecretValidator(256)
		
		// Generate a valid default
		defaultValue, err := GenerateSecureSecret(48)
		require.NoError(t, err)
		
		value, err := sm.LoadSecretWithDefault("TEST_SECRET_DEFAULT", defaultValue, validator)
		require.NoError(t, err)
		assert.Equal(t, defaultValue, value)
	})

	t.Run("uses env var when set", func(t *testing.T) {
		envValue, err := GenerateSecureSecret(48)
		require.NoError(t, err)
		os.Setenv("TEST_SECRET_ENV", envValue)
		defer os.Unsetenv("TEST_SECRET_ENV")
		
		validator := NewJWTSecretValidator(256)
		value, err := sm.LoadSecretWithDefault("TEST_SECRET_ENV", "default", validator)
		require.NoError(t, err)
		assert.Equal(t, envValue, value)
	})
}

func TestGetSecret(t *testing.T) {
	sm := NewSecretManager(SourceEnvironment)

	t.Run("returns false for non-existent secret", func(t *testing.T) {
		_, exists := sm.GetSecret("NON_EXISTENT")
		assert.False(t, exists)
	})

	t.Run("returns true for loaded secret", func(t *testing.T) {
		secret, err := GenerateSecureSecret(48)
		require.NoError(t, err)
		os.Setenv("TEST_GET_SECRET", secret)
		defer os.Unsetenv("TEST_GET_SECRET")
		
		validator := NewJWTSecretValidator(256)
		_, err = sm.LoadSecret("TEST_GET_SECRET", validator)
		require.NoError(t, err)
		
		value, exists := sm.GetSecret("TEST_GET_SECRET")
		assert.True(t, exists)
		assert.Equal(t, secret, value)
	})
}

func TestValidateAll(t *testing.T) {
	t.Run("returns error when no secrets loaded", func(t *testing.T) {
		sm := NewSecretManager(SourceEnvironment)
		err := sm.ValidateAll()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no secrets loaded")
	})

	t.Run("returns nil when secrets loaded", func(t *testing.T) {
		sm := NewSecretManager(SourceEnvironment)
		secret, err := GenerateSecureSecret(48)
		require.NoError(t, err)
		os.Setenv("TEST_VALIDATE_ALL", secret)
		defer os.Unsetenv("TEST_VALIDATE_ALL")
		
		validator := NewJWTSecretValidator(256)
		_, err = sm.LoadSecret("TEST_VALIDATE_ALL", validator)
		require.NoError(t, err)
		
		err = sm.ValidateAll()
		assert.NoError(t, err)
	})
}

func TestRotateSecret(t *testing.T) {
	sm := NewSecretManager(SourceEnvironment)

	t.Run("returns error for non-existent secret", func(t *testing.T) {
		err := sm.RotateSecret("NON_EXISTENT", "new_value")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("rotates existing secret", func(t *testing.T) {
		secret, err := GenerateSecureSecret(48)
		require.NoError(t, err)
		os.Setenv("TEST_ROTATE", secret)
		defer os.Unsetenv("TEST_ROTATE")
		
		validator := NewJWTSecretValidator(256)
		_, err = sm.LoadSecret("TEST_ROTATE", validator)
		require.NoError(t, err)
		
		newSecret, err := GenerateSecureSecret(48)
		require.NoError(t, err)
		err = sm.RotateSecret("TEST_ROTATE", newSecret)
		require.NoError(t, err)
		
		value, exists := sm.GetSecret("TEST_ROTATE")
		assert.True(t, exists)
		assert.Equal(t, newSecret, value)
	})
}

func TestListSecretKeys(t *testing.T) {
	sm := NewSecretManager(SourceEnvironment)

	t.Run("returns empty list when no secrets", func(t *testing.T) {
		keys := sm.ListSecretKeys()
		assert.Empty(t, keys)
	})

	t.Run("returns all loaded secret keys", func(t *testing.T) {
		secret1, _ := GenerateSecureSecret(48)
		secret2, _ := GenerateSecureSecret(48)
		
		os.Setenv("TEST_KEY1", secret1)
		os.Setenv("TEST_KEY2", secret2)
		defer os.Unsetenv("TEST_KEY1")
		defer os.Unsetenv("TEST_KEY2")
		
		validator := NewJWTSecretValidator(256)
		sm.LoadSecret("TEST_KEY1", validator)
		sm.LoadSecret("TEST_KEY2", validator)
		
		keys := sm.ListSecretKeys()
		assert.Len(t, keys, 2)
		assert.Contains(t, keys, "TEST_KEY1")
		assert.Contains(t, keys, "TEST_KEY2")
	})
}

func TestPepperValidator(t *testing.T) {
	t.Run("rejects empty pepper", func(t *testing.T) {
		validator := NewPepperValidator([]string{})
		err := validator.Validate("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("rejects short pepper", func(t *testing.T) {
		validator := NewPepperValidator([]string{})
		err := validator.Validate("short")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 32 characters")
	})

	t.Run("rejects forbidden values", func(t *testing.T) {
		forbidden := []string{"default-pepper", "test-pepper"}
		validator := NewPepperValidator(forbidden)
		err := validator.Validate("default-pepper")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden default value")
	})

	t.Run("accepts valid pepper", func(t *testing.T) {
		validator := NewPepperValidator([]string{})
		pepper, _ := GenerateSecureSecret(32)
		err := validator.Validate(pepper)
		assert.NoError(t, err)
	})
}

func TestDatabaseURLValidator(t *testing.T) {
	validator := NewDatabaseURLValidator()

	t.Run("rejects empty URL", func(t *testing.T) {
		err := validator.Validate("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("rejects URL with sslmode=disable", func(t *testing.T) {
		err := validator.Validate("postgres://user:pass@localhost:5432/db?sslmode=disable")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "should not disable SSL")
	})

	t.Run("accepts valid URL with SSL", func(t *testing.T) {
		err := validator.Validate("postgres://user:pass@db.example.com:5432/db?sslmode=require")
		assert.NoError(t, err)
	})

	t.Run("accepts localhost URL (dev environment)", func(t *testing.T) {
		err := validator.Validate("postgres://user:pass@localhost:5432/db?sslmode=require")
		assert.NoError(t, err)
	})
}

func TestGenerateSecureSecretBytes(t *testing.T) {
	t.Run("generates bytes of requested length", func(t *testing.T) {
		bytes, err := GenerateSecureSecretBytes(64)
		require.NoError(t, err)
		assert.Len(t, bytes, 64)
	})

	t.Run("enforces minimum length of 32", func(t *testing.T) {
		bytes, err := GenerateSecureSecretBytes(16)
		require.NoError(t, err)
		assert.Len(t, bytes, 32)
	})

	t.Run("generates different values each time", func(t *testing.T) {
		bytes1, err := GenerateSecureSecretBytes(32)
		require.NoError(t, err)
		bytes2, err := GenerateSecureSecretBytes(32)
		require.NoError(t, err)
		assert.NotEqual(t, bytes1, bytes2)
	})
}

func TestValidateSecretStrength(t *testing.T) {
	t.Run("rejects short secrets", func(t *testing.T) {
		err := ValidateSecretStrength("short", 32, 100.0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too short")
	})

	t.Run("rejects low entropy secrets", func(t *testing.T) {
		// String with low entropy (repeated characters)
		lowEntropy := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		err := ValidateSecretStrength(lowEntropy, 32, 200.0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient entropy")
	})

	t.Run("accepts strong secrets", func(t *testing.T) {
		secret, _ := GenerateSecureSecret(48)
		err := ValidateSecretStrength(secret, 32, 200.0)
		assert.NoError(t, err)
	})
}

func TestSecretManagerUnsupportedSources(t *testing.T) {
	t.Run("vault source returns error", func(t *testing.T) {
		sm := NewSecretManager(SourceVault)
		_, err := sm.LoadSecret("TEST", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("AWS secrets source returns error", func(t *testing.T) {
		sm := NewSecretManager(SourceAWSSecrets)
		_, err := sm.LoadSecret("TEST", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
	})
}
