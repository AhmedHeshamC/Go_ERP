package repositories

import (
	"context"

	"erpgo/internal/domain/users/entities"
	"github.com/google/uuid"
)

// RoleRepository defines the interface for role data operations
type RoleRepository interface {
	// Role operations
	CreateRole(ctx context.Context, role *entities.Role) error
	GetRoleByID(ctx context.Context, id uuid.UUID) (*entities.Role, error)
	GetRoleByName(ctx context.Context, name string) (*entities.Role, error)
	GetAllRoles(ctx context.Context) ([]*entities.Role, error)
	UpdateRole(ctx context.Context, role *entities.Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
	RoleExists(ctx context.Context, name string) (bool, error)

	// User role assignments
	AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID, assignedBy uuid.UUID) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*entities.Role, error)
	GetUsersWithRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error)
	HasUserRole(ctx context.Context, userID, roleID uuid.UUID) (bool, error)
	RemoveAllUserRoles(ctx context.Context, userID uuid.UUID) error

	// Permission checking
	UserHasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
	UserHasAnyPermission(ctx context.Context, userID uuid.UUID, permissions ...string) (bool, error)
	UserHasAllPermissions(ctx context.Context, userID uuid.UUID, permissions ...string) (bool, error)

	// Role permission management
	AddPermissionToRole(ctx context.Context, roleID uuid.UUID, permission string) error
	RemovePermissionFromRole(ctx context.Context, roleID uuid.UUID, permission string) error
	GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]string, error)

	// Batch operations
	CreateDefaultRoles(ctx context.Context) error
	GetRoleAssignmentHistory(ctx context.Context, userID uuid.UUID) ([]*entities.UserRole, error)
}

// PermissionChecker provides a convenient interface for permission checking
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
	HasAnyPermission(ctx context.Context, userID uuid.UUID, permissions ...string) (bool, error)
	HasAllPermissions(ctx context.Context, userID uuid.UUID, permissions ...string) (bool, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error)
}
