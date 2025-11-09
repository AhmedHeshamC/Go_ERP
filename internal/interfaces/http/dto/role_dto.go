package dto

import (
	"time"

	"github.com/google/uuid"
)

// RoleResponse represents a role response
type RoleResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Permissions []string   `json:"permissions,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateRoleRequest represents a role creation request
type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required,min=2,max=50"`
	Description string   `json:"description" binding:"max=255"`
	Permissions []string `json:"permissions,omitempty"`
}

// UpdateRoleRequest represents a role update request
type UpdateRoleRequest struct {
	Name        string   `json:"name,omitempty" binding:"omitempty,min=2,max=50"`
	Description string   `json:"description,omitempty" binding:"omitempty,max=255"`
	Permissions []string `json:"permissions,omitempty"`
}

// AssignRoleRequest represents a role assignment request
type AssignRoleRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required,uuid"`
	RoleID uuid.UUID `json:"role_id" binding:"required,uuid"`
}

// RemoveRoleRequest represents a role removal request
type RemoveRoleRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required,uuid"`
	RoleID uuid.UUID `json:"role_id" binding:"required,uuid"`
}

// UserRoleResponse represents a user role response
type UserRoleResponse struct {
	UserID     uuid.UUID  `json:"user_id"`
	RoleID     uuid.UUID  `json:"role_id"`
	RoleName   string     `json:"role_name"`
	AssignedBy uuid.UUID  `json:"assigned_by"`
	AssignedAt time.Time  `json:"assigned_at"`
}

// UserPermissionsResponse represents user permissions response
type UserPermissionsResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	Permissions []string  `json:"permissions"`
}

// RolePermissionRequest represents a role permission update request
type RolePermissionRequest struct {
	RoleID      uuid.UUID `json:"role_id" binding:"required,uuid"`
	Permissions []string  `json:"permissions" binding:"required,min=1"`
}

// RolePermissionResponse represents a role permission response
type RolePermissionResponse struct {
	RoleID      uuid.UUID `json:"role_id"`
	RoleName    string    `json:"role_name"`
	Permissions []string  `json:"permissions"`
}