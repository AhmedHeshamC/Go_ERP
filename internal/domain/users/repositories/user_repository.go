package repositories

import (
	"context"

	"erpgo/internal/domain/users/entities"
	"github.com/google/uuid"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *entities.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	GetByUsername(ctx context.Context, username string) (*entities.User, error)
	Update(ctx context.Context, user *entities.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter UserFilter) ([]*entities.User, error)
	Count(ctx context.Context, filter UserFilter) (int, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
	AssignRole(ctx context.Context, userID uuid.UUID, roleName string, assignedBy uuid.UUID) error
}

// UserRoleRepository defines the interface for user-role relationship operations
type UserRoleRepository interface {
	AssignRole(ctx context.Context, userID, roleID, assignedBy string) error
	RemoveRole(ctx context.Context, userID, roleID string) error
	GetUserRoles(ctx context.Context, userID string) ([]*entities.Role, error)
	GetUsersByRole(ctx context.Context, roleID string) ([]*entities.User, error)
	HasRole(ctx context.Context, userID, roleID string) (bool, error)
}

// UserFilter defines filtering options for user queries
type UserFilter struct {
	Search     string
	IsActive   *bool
	IsVerified *bool
	RoleID     string
	Page       int
	Limit      int
	SortBy     string
	SortOrder  string
}

// RoleFilter defines filtering options for role queries
type RoleFilter struct {
	Search    string
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}
