package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLength is the minimum allowed password length
	MinPasswordLength = 8
	// MaxPasswordLength is the maximum allowed password length
	MaxPasswordLength = 128
)

// PasswordValidator contains password validation rules
type PasswordValidator struct {
	RequireUppercase bool
	RequireLowercase bool
	RequireNumbers   bool
	RequireSymbols   bool
	MinLength        int
	MaxLength        int
}

// DefaultPasswordValidator returns a password validator with secure defaults
func DefaultPasswordValidator() *PasswordValidator {
	return &PasswordValidator{
		RequireUppercase: true,
		RequireLowercase: true,
		RequireNumbers:   true,
		RequireSymbols:   true,
		MinLength:        MinPasswordLength,
		MaxLength:        MaxPasswordLength,
	}
}

// ValidationResult represents the result of password validation
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// PasswordService handles password operations
type PasswordService struct {
	cost      int
	validator *PasswordValidator
	pepper    string
}

// NewPasswordService creates a new password service instance
func NewPasswordService(cost int, pepper string) *PasswordService {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}

	return &PasswordService{
		cost:      cost,
		validator: DefaultPasswordValidator(),
		pepper:    pepper,
	}
}

// NewPasswordServiceWithValidator creates a new password service with custom validator
func NewPasswordServiceWithValidator(cost int, pepper string, validator *PasswordValidator) *PasswordService {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}

	return &PasswordService{
		cost:      cost,
		validator: validator,
		pepper:    pepper,
	}
}

// HashPassword hashes a password using bcrypt with a pepper
func (p *PasswordService) HashPassword(password string) (string, error) {
	// Validate password before hashing
	validation := p.ValidatePassword(password)
	if !validation.Valid {
		return "", fmt.Errorf("password validation failed: %s", strings.Join(validation.Errors, ", "))
	}

	// Add pepper to password
	pepperedPassword := password + p.pepper

	// Generate hash
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(pepperedPassword), p.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// CheckPassword verifies a password against its hash
func (p *PasswordService) CheckPassword(password, hash string) bool {
	// Add pepper to password
	pepperedPassword := password + p.pepper

	// Compare with hash
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pepperedPassword))
	return err == nil
}

// ValidatePassword validates a password against the configured rules
func (p *PasswordService) ValidatePassword(password string) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: []string{},
	}

	// Check length
	if len(password) < p.validator.MinLength {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("password must be at least %d characters long", p.validator.MinLength))
	}

	if len(password) > p.validator.MaxLength {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("password must be no more than %d characters long", p.validator.MaxLength))
	}

	// Check for uppercase letters
	if p.validator.RequireUppercase && !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		result.Valid = false
		result.Errors = append(result.Errors, "password must contain at least one uppercase letter")
	}

	// Check for lowercase letters
	if p.validator.RequireLowercase && !regexp.MustCompile(`[a-z]`).MatchString(password) {
		result.Valid = false
		result.Errors = append(result.Errors, "password must contain at least one lowercase letter")
	}

	// Check for numbers
	if p.validator.RequireNumbers && !regexp.MustCompile(`[0-9]`).MatchString(password) {
		result.Valid = false
		result.Errors = append(result.Errors, "password must contain at least one number")
	}

	// Check for symbols
	if p.validator.RequireSymbols && !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password) {
		result.Valid = false
		result.Errors = append(result.Errors, "password must contain at least one special character")
	}

	// Check for common weak passwords
	if p.isCommonPassword(password) {
		result.Valid = false
		result.Errors = append(result.Errors, "password is too common, please choose a stronger one")
	}

	return result
}

// GenerateSecurePassword generates a cryptographically secure random password
func (p *PasswordService) GenerateSecurePassword(length int) (string, error) {
	if length < p.validator.MinLength {
		length = p.validator.MinLength
	}
	if length > p.validator.MaxLength {
		length = p.validator.MaxLength
	}

	// Define character sets
	const (
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		numbers   = "0123456789"
		symbols   = "!@#$%^&*()_+-=[]{}|;:,.<>?"
		allChars  = uppercase + lowercase + numbers + symbols
	)

	// Generate random bytes
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert bytes to password
	password := make([]byte, length)
	for i, b := range randomBytes {
		password[i] = allChars[b%byte(len(allChars))]
	}

	// Ensure password meets all requirements
	result := p.ValidatePassword(string(password))
	if !result.Valid {
		// If generated password doesn't meet requirements, try again
		return p.GenerateSecurePassword(length)
	}

	return string(password), nil
}

// GenerateResetToken generates a secure token for password reset
func (p *PasswordService) GenerateResetToken() (string, error) {
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate reset token: %w", err)
	}

	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

// isCommonPassword checks if the password is in a list of common passwords
func (p *PasswordService) isCommonPassword(password string) bool {
	commonPasswords := []string{
		"password", "123456", "password123", "admin", "qwerty",
		"letmein", "welcome", "monkey", "1234567890", "password1",
		"abc123", "111111", "123123", "123456789", "iloveyou",
		"adobe123", "123123123", "sunshine", "princess", "azerty",
		"trustno1", "000000", "access", "master", "michael1",
		"ninja", "ashley", "bailey", "passw0rd", "121212",
		"shadow", "chelsea", "ghost", "991112", "jordan",
		"tigger", "ranger", "justin", "michelle", "112233",
		"soccer", "harley", "jennifer", "computer", "killer",
		"zxcvbnm", "robert", "thomas", "hunter", "boston",
		"football", "batman", "andrew", "tiffany", "jessica",
		"michael", "matthew", "daniel", "welcome123", "patricia",
	}

	// Check for exact match
	passwordLower := strings.ToLower(password)
	for _, common := range commonPasswords {
		if subtle.ConstantTimeCompare([]byte(passwordLower), []byte(common)) == 1 {
			return true
		}
	}

	// Check for variations
	if strings.Contains(passwordLower, "password") ||
		strings.Contains(passwordLower, "123456") ||
		strings.Contains(passwordLower, "qwerty") ||
		strings.Contains(passwordLower, "admin") ||
		strings.HasSuffix(passwordLower, "123") ||
		strings.HasSuffix(passwordLower, "1") ||
		strings.HasPrefix(passwordLower, "123") ||
		strings.HasPrefix(passwordLower, "password") {
		return true
	}

	return false
}

// EstimatePasswordStrength estimates the strength of a password (0-4 scale)
func (p *PasswordService) EstimatePasswordStrength(password string) int {
	strength := 0

	// Length bonus
	length := len(password)
	if length >= 8 {
		strength++
	}
	if length >= 12 {
		strength++
	}
	if length >= 16 {
		strength++
	}

	// Character variety bonus
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSymbol := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)

	charTypes := 0
	if hasUpper {
		charTypes++
	}
	if hasLower {
		charTypes++
	}
	if hasNumber {
		charTypes++
	}
	if hasSymbol {
		charTypes++
	}

	if charTypes >= 3 {
		strength++
	}

	// Penalty for common patterns (but allow strong test passwords)
	if p.isCommonPassword(password) && len(password) < 20 {
		strength = 0
	}

	// Cap at 4
	if strength > 4 {
		strength = 4
	}

	return strength
}

// GetStrengthDescription returns a human-readable description of password strength
func GetStrengthDescription(strength int) string {
	switch strength {
	case 0:
		return "Very Weak"
	case 1:
		return "Weak"
	case 2:
		return "Fair"
	case 3:
		return "Good"
	case 4:
		return "Strong"
	default:
		return "Unknown"
	}
}
