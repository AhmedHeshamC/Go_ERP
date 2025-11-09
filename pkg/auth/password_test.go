package auth

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordService_HashPassword(t *testing.T) {
	passwordService := NewPasswordService(12, "test-pepper")

	tests := []struct {
		name        string
		password    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid password",
			password:    "SecurePass123!",
			expectError: false,
		},
		{
			name:        "too short password",
			password:    "short",
			expectError: true,
			errorMsg:    "password must be at least 8 characters long",
		},
		{
			name:        "no uppercase",
			password:    "securepass123!",
			expectError: true,
			errorMsg:    "password must contain at least one uppercase letter",
		},
		{
			name:        "no lowercase",
			password:    "SECUREPASS123!",
			expectError: true,
			errorMsg:    "password must contain at least one lowercase letter",
		},
		{
			name:        "no numbers",
			password:    "SecurePassword!",
			expectError: true,
			errorMsg:    "password must contain at least one number",
		},
		{
			name:        "no symbols",
			password:    "SecurePass123",
			expectError: true,
			errorMsg:    "password must contain at least one special character",
		},
		{
			name:        "common password",
			password:    "Password123!",
			expectError: true,
			errorMsg:    "password is too common",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := passwordService.HashPassword(tt.password)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Empty(t, hash)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
				assert.NotEqual(t, tt.password, hash) // Hash should not equal password
			}
		})
	}
}

func TestPasswordService_CheckPassword(t *testing.T) {
	passwordService := NewPasswordService(12, "test-pepper")
	password := "SecurePass123!"

	// Hash password
	hash, err := passwordService.HashPassword(password)
	require.NoError(t, err)

	// Test correct password
	assert.True(t, passwordService.CheckPassword(password, hash))

	// Test incorrect password
	assert.False(t, passwordService.CheckPassword("WrongPassword123!", hash))

	// Test with different pepper (should fail)
	otherService := NewPasswordService(12, "different-pepper")
	assert.False(t, otherService.CheckPassword(password, hash))
}

func TestPasswordService_ValidatePassword(t *testing.T) {
	validator := &PasswordValidator{
		RequireUppercase: true,
		RequireLowercase: true,
		RequireNumbers:   true,
		RequireSymbols:   true,
		MinLength:        8,
		MaxLength:        128,
	}
	passwordService := NewPasswordServiceWithValidator(12, "test-pepper", validator)

	tests := []struct {
		name         string
		password     string
		expectValid  bool
		errorCount   int
		errorMsgs    []string
	}{
		{
			name:        "valid password",
			password:    "SecurePass123!",
			expectValid: true,
			errorCount:  0,
		},
		{
			name:        "too short",
			password:    "short",
			expectValid: false,
			errorCount:  4, // length, uppercase, number, symbol
		},
		{
			name:        "no uppercase",
			password:    "securepass123!",
			expectValid: false,
			errorCount:  1,
			errorMsgs:   []string{"password must contain at least one uppercase letter"},
		},
		{
			name:        "no lowercase",
			password:    "SECUREPASS123!",
			expectValid: false,
			errorCount:  1,
			errorMsgs:   []string{"password must contain at least one lowercase letter"},
		},
		{
			name:        "no numbers",
			password:    "SecurePassword!",
			expectValid: false,
			errorCount:  2, // no numbers + common password detection
		},
		{
			name:        "no symbols",
			password:    "SecurePass123",
			expectValid: false,
			errorCount:  2, // no symbols + common password detection
		},
		{
			name:        "multiple issues",
			password:    "weak",
			expectValid: false,
			errorCount:  4, // length, uppercase, number, symbol
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := passwordService.ValidatePassword(tt.password)

			assert.Equal(t, tt.expectValid, result.Valid)
			assert.Equal(t, tt.errorCount, len(result.Errors))

			if len(tt.errorMsgs) > 0 {
				for _, expectedMsg := range tt.errorMsgs {
					found := false
					for _, actualMsg := range result.Errors {
						if strings.Contains(actualMsg, expectedMsg) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error message not found: %s", expectedMsg)
				}
			}
		})
	}
}

func TestPasswordService_GenerateSecurePassword(t *testing.T) {
	passwordService := NewPasswordService(12, "test-pepper")

	// Test generating passwords of different lengths
	lengths := []int{8, 12, 16, 32}
	for _, length := range lengths {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			password, err := passwordService.GenerateSecurePassword(length)
			require.NoError(t, err)
			assert.Equal(t, length, len(password))

			// Generated password should pass validation
			result := passwordService.ValidatePassword(password)
			assert.True(t, result.Valid, "Generated password failed validation: %v", result.Errors)
		})
	}

	// Test that generated passwords are unique
	passwords := make(map[string]bool)
	for i := 0; i < 100; i++ {
		password, err := passwordService.GenerateSecurePassword(16)
		require.NoError(t, err)
		assert.False(t, passwords[password], "Generated duplicate password")
		passwords[password] = true
	}
}

func TestPasswordService_GenerateResetToken(t *testing.T) {
	passwordService := NewPasswordService(12, "test-pepper")

	// Generate multiple tokens
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := passwordService.GenerateResetToken()
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.False(t, tokens[token], "Generated duplicate reset token")
		tokens[token] = true
	}
}

func TestPasswordService_EstimatePasswordStrength(t *testing.T) {
	passwordService := NewPasswordService(12, "test-pepper")

	tests := []struct {
		name         string
		password     string
		expectedStr  string
		expectedMin  int
		expectedMax  int
	}{
		{
			name:        "very weak",
			password:    "password",
			expectedStr: "Very Weak",
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			name:        "weak",
			password:    "password1",
			expectedStr: "Very Weak", // Common password overrides strength calculation
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			name:        "fair",
			password:    "Password1",
			expectedStr: "Very Weak", // Common password overrides strength calculation
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			name:        "good",
			password:    "Password123!",
			expectedStr: "Very Weak", // Common password overrides strength calculation
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			name:        "strong",
			password:    "XyZ$uperStr0ngP@ssw0rd!@#",
			expectedStr: "Strong",
			expectedMin: 4,
			expectedMax: 4,
		},
		{
			name:        "very long strong",
			password:    "XyZ$uperStr0ngP@ssw0rdWithL0tsOfChars987654!@#",
			expectedStr: "Strong",
			expectedMin: 4,
			expectedMax: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strength := passwordService.EstimatePasswordStrength(tt.password)
			strengthDesc := GetStrengthDescription(strength)

			assert.GreaterOrEqual(t, strength, tt.expectedMin)
			assert.LessOrEqual(t, strength, tt.expectedMax)
			assert.Equal(t, tt.expectedStr, strengthDesc)
		})
	}
}

func TestPasswordService_IsCommonPassword(t *testing.T) {
	passwordService := NewPasswordService(12, "test-pepper")

	commonPasswords := []string{
		"password", "123456", "password123", "admin", "qwerty",
		"letmein", "welcome", "monkey", "1234567890", "password1",
		"abc123", "111111", "123123", "123456789", "iloveyou",
	}

	for _, password := range commonPasswords {
		t.Run(fmt.Sprintf("common_%s", password), func(t *testing.T) {
			result := passwordService.ValidatePassword(password)
			assert.False(t, result.Valid)
			assert.Contains(t, strings.Join(result.Errors, ","), "too common")
		})
	}

	// Test variations
	variations := []string{
		"Password2023", "MyPassword123", "admin123", "qwerty123",
		"password12345", "123456password", "MyPassword",
	}

	for _, password := range variations {
		t.Run(fmt.Sprintf("variation_%s", password), func(t *testing.T) {
			result := passwordService.ValidatePassword(password)
			assert.False(t, result.Valid)
			assert.Contains(t, strings.Join(result.Errors, ","), "too common")
		})
	}
}

func TestDefaultPasswordValidator(t *testing.T) {
	validator := DefaultPasswordValidator()

	assert.True(t, validator.RequireUppercase)
	assert.True(t, validator.RequireLowercase)
	assert.True(t, validator.RequireNumbers)
	assert.True(t, validator.RequireSymbols)
	assert.Equal(t, MinPasswordLength, validator.MinLength)
	assert.Equal(t, MaxPasswordLength, validator.MaxLength)
}

func TestGetStrengthDescription(t *testing.T) {
	tests := []struct {
		strength   int
		expected   string
	}{
		{0, "Very Weak"},
		{1, "Weak"},
		{2, "Fair"},
		{3, "Good"},
		{4, "Strong"},
		{-1, "Unknown"},
		{5, "Unknown"},
		{10, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("strength_%d", tt.strength), func(t *testing.T) {
			desc := GetStrengthDescription(tt.strength)
			assert.Equal(t, tt.expected, desc)
		})
	}
}

func TestPasswordService_CostValidation(t *testing.T) {
	// Test invalid cost values
	service1 := NewPasswordService(-1, "test-pepper")
	assert.NotNil(t, service1)

	service2 := NewPasswordService(100, "test-pepper")
	assert.NotNil(t, service2)

	// Test that service uses default cost for invalid values
	assert.Equal(t, bcrypt.DefaultCost, service1.cost)
	assert.Equal(t, bcrypt.DefaultCost, service2.cost)

	// Test valid cost
	service3 := NewPasswordService(12, "test-pepper")
	assert.Equal(t, 12, service3.cost)
}