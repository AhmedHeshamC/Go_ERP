package entities

import (
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

// Role represents a role in the system
type Role struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description string      `json:"description,omitempty" db:"description"`
	Permissions []string    `json:"permissions" db:"permissions"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

// UserRole represents the relationship between users and roles
type UserRole struct {
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	RoleID     uuid.UUID  `json:"role_id" db:"role_id"`
	AssignedAt time.Time  `json:"assigned_at" db:"assigned_at"`
	AssignedBy uuid.UUID  `json:"assigned_by" db:"assigned_by"`
}