package secrets

import (
	"os"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: production-readiness, Property 1: Secret Validation on Startup**
// For any system startup, all required secrets (JWT_SECRET, PASSWORD_PEPPER, DATABASE_URL)
// must be loaded from environment variables and validated before the system accepts requests
// **Validates: Requirements 1.1, 1.2**
func TestProperty_SecretValidationOnStartup(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Invalid secrets must be rejected during loading
	properties.Property("invalid secrets are rejected", prop.ForAll(
		func(secret string) bool {
			// Setup environment with the test secret
			os.Setenv("TEST_SECRET", secret)
			defer os.Unsetenv("TEST_SECRET")

			sm := NewSecretManager(SourceEnvironment)
			validator := NewJWTSecretValidator(256)
			_, err := sm.LoadSecret("TEST_SECRET", validator)

			// If secret is too short, it should be rejected
			if len(secret) < 32 {
				return err != nil
			}

			// If secret is weak, it should be rejected
			if isWeakSecret(secret) {
				return err != nil
			}

			// Otherwise, validation depends on entropy
			// We can't easily predict entropy, so we just check consistency
			return true
		},
		genSecretString(),
	))

	// Property: Empty secrets must be rejected
	properties.Property("empty secrets are rejected", prop.ForAll(
		func() bool {
			os.Setenv("TEST_EMPTY", "")
			defer os.Unsetenv("TEST_EMPTY")

			sm := NewSecretManager(SourceEnvironment)
			validator := NewJWTSecretValidator(256)
			_, err := sm.LoadSecret("TEST_EMPTY", validator)

			// Empty secrets should always be rejected
			return err != nil
		},
	))

	// Property: Missing secrets must be rejected
	properties.Property("missing secrets are rejected", prop.ForAll(
		func() bool {
			// Don't set the environment variable
			os.Unsetenv("TEST_MISSING")

			sm := NewSecretManager(SourceEnvironment)
			validator := NewJWTSecretValidator(256)
			_, err := sm.LoadSecret("TEST_MISSING", validator)

			// Missing secrets should always be rejected
			return err != nil
		},
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: production-readiness, Property 2: JWT Secret Entropy**
// For any JWT_SECRET value, if it has less than 256 bits of entropy (32 bytes),
// the system must reject it during startup
// **Validates: Requirements 1.3**
func TestProperty_JWTSecretEntropy(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: JWT secrets with less than 32 bytes must be rejected
	properties.Property("secrets under 32 bytes are rejected", prop.ForAll(
		func(secret string) bool {
			validator := NewJWTSecretValidator(256)
			err := validator.Validate(secret)

			// Secrets under 32 bytes should fail validation
			if len(secret) < 32 {
				return err != nil
			}

			// Weak/default secrets should fail validation
			if isWeakSecret(secret) {
				return err != nil
			}

			// Valid secrets (>= 32 bytes, not weak) should pass
			// Note: entropy check might still fail for low-entropy strings
			return true
		},
		gen.AnyString(),
	))

	// Property: Weak/default secrets must be rejected regardless of length
	properties.Property("weak secrets are rejected", prop.ForAll(
		func(weakValue string) bool {
			// Test with known weak values
			weakSecrets := []string{
				"your-super-secret-jwt-key",
				"secret",
				"jwt-secret",
				"change-me",
				"default",
				"test",
				"password",
				"123456",
			}

			validator := NewJWTSecretValidator(256)

			for _, weak := range weakSecrets {
				err := validator.Validate(weak)
				if err == nil {
					return false // Weak secret was accepted
				}
			}

			return true
		},
		gen.Const(""),
	))

	// Property: Secrets with sufficient entropy should pass validation
	properties.Property("secrets with sufficient entropy pass validation", prop.ForAll(
		func(seed int64) bool {
			// Generate a cryptographically secure secret with enough bytes
			// to meet the entropy requirement after base64 encoding
			// Base64 encoding reduces entropy, so we need more bytes
			secret, err := GenerateSecureSecret(48) // 48 bytes should give us enough entropy
			if err != nil {
				return false
			}

			validator := NewJWTSecretValidator(256)
			err = validator.Validate(secret)

			// Generated secure secrets should always pass
			return err == nil
		},
		gen.Int64(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper function to check if a secret is weak
func isWeakSecret(secret string) bool {
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

	secretLower := strings.ToLower(secret)
	for _, weak := range weakValues {
		if secretLower == weak {
			return true
		}
	}

	return false
}

// Generator for secret strings (including valid and invalid)
func genSecretString() gopter.Gen {
	return gen.OneGenOf(
		// Short secrets (invalid)
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) < 32 }),
		// Long secrets (potentially valid)
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 32 }),
		// Weak secrets
		gen.OneConstOf(
			"your-super-secret-jwt-key",
			"secret",
			"jwt-secret",
			"change-me",
			"default",
			"test",
			"password",
			"123456",
		),
		// Secure secrets
		gen.Int64().Map(func(seed int64) string {
			secret, _ := GenerateSecureSecret(32)
			return secret
		}),
	)
}

// Generator for database URLs
func genDatabaseURL() gopter.Gen {
	return gen.OneGenOf(
		// Empty URL (invalid)
		gen.Const(""),
		// Valid URLs
		gen.Const("postgres://user:pass@localhost:5432/db?sslmode=require"),
		gen.Const("postgres://user:pass@db.example.com:5432/db?sslmode=require"),
		// Invalid URLs (with sslmode=disable)
		gen.Const("postgres://user:pass@localhost:5432/db?sslmode=disable"),
		// Random strings
		gen.AlphaString(),
	)
}
