package entities

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a role in the system
type Role struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	Permissions []string   `json:"permissions" db:"permissions"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// UserRole represents the relationship between a user and a role
type UserRole struct {
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	RoleID     uuid.UUID `json:"role_id" db:"role_id"`
	AssignedAt time.Time `json:"assigned_at" db:"assigned_at"`
	AssignedBy uuid.UUID `json:"assigned_by" db:"assigned_by"`
}

// HasPermission checks if the role has a specific permission
func (r *Role) HasPermission(permission string) bool {
	for _, p := range r.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if the role has any of the specified permissions
func (r *Role) HasAnyPermission(permissions ...string) bool {
	permissionMap := make(map[string]bool)
	for _, p := range r.Permissions {
		permissionMap[p] = true
	}

	for _, permission := range permissions {
		if permissionMap[permission] {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if the role has all of the specified permissions
func (r *Role) HasAllPermissions(permissions ...string) bool {
	permissionMap := make(map[string]bool)
	for _, p := range r.Permissions {
		permissionMap[p] = true
	}

	for _, permission := range permissions {
		if !permissionMap[permission] {
			return false
		}
	}
	return true
}

// RolePermission represents a granular permission that can be assigned to roles
type RolePermission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Resource    string    `json:"resource" db:"resource"`
	Action      string    `json:"action" db:"action"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Permission constants for the ERP system
const (
	// User permissions
	PermissionUserCreate = "users.create"
	PermissionUserRead   = "users.read"
	PermissionUserUpdate = "users.update"
	PermissionUserDelete = "users.delete"

	// Role permissions
	PermissionRoleCreate = "roles.create"
	PermissionRoleRead   = "roles.read"
	PermissionRoleUpdate = "roles.update"
	PermissionRoleDelete = "roles.delete"

	// Product permissions
	PermissionProductCreate = "products.create"
	PermissionProductRead   = "products.read"
	PermissionProductUpdate = "products.update"
	PermissionProductDelete = "products.delete"

	// Order permissions
	PermissionOrderCreate = "orders.create"
	PermissionOrderRead   = "orders.read"
	PermissionOrderUpdate = "orders.update"
	PermissionOrderDelete = "orders.delete"

	// Inventory permissions
	PermissionInventoryCreate = "inventory.create"
	PermissionInventoryRead   = "inventory.read"
	PermissionInventoryUpdate = "inventory.update"
	PermissionInventoryDelete = "inventory.delete"

	// System permissions
	PermissionSystemAdmin = "system.admin"
	PermissionSystemRead  = "system.read"

	// Course permissions (for educational platform)
	PermissionCourseCreate = "courses.create"
	PermissionCourseRead   = "courses.read"
	PermissionCourseUpdate = "courses.update"
	PermissionCourseDelete = "courses.delete"

	// Assignment permissions
	PermissionAssignmentCreate = "assignments.create"
	PermissionAssignmentRead   = "assignments.read"
	PermissionAssignmentUpdate = "assignments.update"
	PermissionAssignmentDelete = "assignments.delete"

	// Grade permissions
	PermissionGradeCreate = "grades.create"
	PermissionGradeRead   = "grades.read"
	PermissionGradeUpdate = "grades.update"

	// Profile permissions
	PermissionProfileRead = "profile.read"
	PermissionProfileUpdate = "profile.update"
)

// DefaultRoles returns the default roles for the ERP system
func DefaultRoles() []Role {
	return []Role{
		{
			Name:        "admin",
			Description: "System Administrator",
			Permissions: []string{
				PermissionUserCreate, PermissionUserRead, PermissionUserUpdate, PermissionUserDelete,
				PermissionRoleCreate, PermissionRoleRead, PermissionRoleUpdate, PermissionRoleDelete,
				PermissionProductCreate, PermissionProductRead, PermissionProductUpdate, PermissionProductDelete,
				PermissionOrderCreate, PermissionOrderRead, PermissionOrderUpdate, PermissionOrderDelete,
				PermissionInventoryCreate, PermissionInventoryRead, PermissionInventoryUpdate, PermissionInventoryDelete,
				PermissionSystemAdmin, PermissionSystemRead,
				PermissionCourseCreate, PermissionCourseRead, PermissionCourseUpdate, PermissionCourseDelete,
				PermissionAssignmentCreate, PermissionAssignmentRead, PermissionAssignmentUpdate, PermissionAssignmentDelete,
				PermissionGradeCreate, PermissionGradeRead, PermissionGradeUpdate,
				PermissionProfileRead, PermissionProfileUpdate,
			},
		},
		{
			Name:        "teacher",
			Description: "Teacher with full access to educational resources",
			Permissions: []string{
				PermissionUserCreate, PermissionUserRead, PermissionUserUpdate,
				PermissionCourseCreate, PermissionCourseRead, PermissionCourseUpdate, PermissionCourseDelete,
				PermissionAssignmentCreate, PermissionAssignmentRead, PermissionAssignmentUpdate, PermissionAssignmentDelete,
				PermissionGradeCreate, PermissionGradeRead, PermissionGradeUpdate,
				PermissionProfileRead, PermissionProfileUpdate,
			},
		},
		{
			Name:        "teaching_assistant",
			Description: "Teaching Assistant with limited access",
			Permissions: []string{
				PermissionUserRead,
				PermissionCourseRead,
				PermissionAssignmentRead,
				PermissionGradeRead, PermissionGradeUpdate,
				PermissionProfileRead, PermissionProfileUpdate,
			},
		},
		{
			Name:        "student",
			Description: "Student with basic access",
			Permissions: []string{
				PermissionCourseRead,
				PermissionAssignmentRead,
				PermissionGradeRead,
				PermissionProfileRead, PermissionProfileUpdate,
			},
		},
		{
			Name:        "manager",
			Description: "Manager with business function access",
			Permissions: []string{
				PermissionProductRead, PermissionProductUpdate,
				PermissionOrderCreate, PermissionOrderRead, PermissionOrderUpdate,
				PermissionInventoryRead, PermissionInventoryUpdate,
				PermissionUserRead,
				PermissionProfileRead, PermissionProfileUpdate,
			},
		},
		{
			Name:        "employee",
			Description: "Employee with basic operational access",
			Permissions: []string{
				PermissionProductRead,
				PermissionOrderRead,
				PermissionInventoryRead,
				PermissionProfileRead, PermissionProfileUpdate,
			},
		},
		{
			Name:        "customer",
			Description: "Customer with self-service access",
			Permissions: []string{
				PermissionProductRead,
				PermissionOrderCreate, PermissionOrderRead,
				PermissionProfileRead, PermissionProfileUpdate,
			},
		},
	}
}