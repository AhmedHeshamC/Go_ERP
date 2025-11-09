package entities

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	Username     string     `json:"username" db:"username"`
	PasswordHash string     `json:"-" db:"password_hash"`
	FirstName    string     `json:"first_name" db:"first_name"`
	LastName     string     `json:"last_name" db:"last_name"`
	Phone        string     `json:"phone,omitempty" db:"phone"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	IsVerified   bool       `json:"is_verified" db:"is_verified"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// Validate validates the user entity
func (u *User) Validate() error {
	var errs []error

	// Validate UUID
	if u.ID == uuid.Nil {
		errs = append(errs, errors.New("user ID cannot be empty"))
	}

	// Validate email
	if err := u.validateEmail(); err != nil {
		errs = append(errs, fmt.Errorf("invalid email: %w", err))
	}

	// Validate username
	if err := u.validateUsername(); err != nil {
		errs = append(errs, fmt.Errorf("invalid username: %w", err))
	}

	// Validate password hash
	if strings.TrimSpace(u.PasswordHash) == "" {
		errs = append(errs, errors.New("password hash cannot be empty"))
	}

	// Validate first name
	if strings.TrimSpace(u.FirstName) == "" {
		errs = append(errs, errors.New("first name cannot be empty"))
	} else if len(u.FirstName) > 100 {
		errs = append(errs, errors.New("first name cannot exceed 100 characters"))
	}

	// Validate last name
	if strings.TrimSpace(u.LastName) == "" {
		errs = append(errs, errors.New("last name cannot be empty"))
	} else if len(u.LastName) > 100 {
		errs = append(errs, errors.New("last name cannot exceed 100 characters"))
	}

	// Validate phone (optional)
	if u.Phone != "" {
		if err := u.validatePhone(); err != nil {
			errs = append(errs, fmt.Errorf("invalid phone: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	return fmt.Sprintf("%s %s", strings.TrimSpace(u.FirstName), strings.TrimSpace(u.LastName))
}

// validateEmail validates the email format
func (u *User) validateEmail() error {
	email := strings.TrimSpace(u.Email)
	if email == "" {
		return errors.New("email cannot be empty")
	}

	// Basic email regex pattern
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}

	if len(email) > 255 {
		return errors.New("email cannot exceed 255 characters")
	}

	return nil
}

// validateUsername validates the username format
func (u *User) validateUsername() error {
	username := strings.TrimSpace(u.Username)
	if username == "" {
		return errors.New("username cannot be empty")
	}

	// Username should be 3-50 characters, alphanumeric with underscores and hyphens
	if len(username) < 3 || len(username) > 50 {
		return errors.New("username must be between 3 and 50 characters")
	}

	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !usernameRegex.MatchString(username) {
		return errors.New("username can only contain letters, numbers, underscores, and hyphens")
	}

	return nil
}

// validatePhone validates the phone number format
func (u *User) validatePhone() error {
	phone := strings.TrimSpace(u.Phone)
	if phone == "" {
		return nil // Phone is optional
	}

	// Basic phone regex - allows international format with +, spaces, hyphens, and parentheses
	phoneRegex := regexp.MustCompile(`^\+?[\d\s\-\(\)]{7,20}$`)
	if !phoneRegex.MatchString(phone) {
		return errors.New("invalid phone number format")
	}

	return nil
}

// IsActiveUser returns true if the user is active
func (u *User) IsActiveUser() bool {
	return u.IsActive
}

// IsVerifiedUser returns true if the user is verified
func (u *User) IsVerifiedUser() bool {
	return u.IsVerified
}

// UpdateLastLogin updates the user's last login time
func (u *User) UpdateLastLogin() {
	now := time.Now().UTC()
	u.LastLoginAt = &now
}

// ToSafeUser returns a user object without sensitive information
func (u *User) ToSafeUser() *User {
	return &User{
		ID:          u.ID,
		Email:       u.Email,
		Username:    u.Username,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		Phone:       u.Phone,
		IsActive:    u.IsActive,
		IsVerified:  u.IsVerified,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// HasPermission checks if the user has a specific permission
// Note: This is a placeholder implementation. In a real application,
// this would check the user's roles and their associated permissions
func (u *User) HasPermission(permission string) bool {
	// Placeholder: return false for all permissions for now
	// In a real implementation, this would:
	// 1. Fetch user's roles from the database
	// 2. Check each role's permissions
	// 3. Return true if any role has the requested permission
	return false
}

