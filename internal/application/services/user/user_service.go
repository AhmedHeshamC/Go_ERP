package user

import (
	"context"

	"github.com/erpgo/erpgo/internal/domain/users/entities"
)

// Service defines the business logic interface for user management
type Service interface {
	// User management
	CreateUser(ctx context.Context, req *CreateUserRequest) (*entities.User, error)
	GetUser(ctx context.Context, id string) (*entities.User, error)
	GetUserByEmail(ctx context.Context, email string) (*entities.User, error)
	UpdateUser(ctx context.Context, id string, req *UpdateUserRequest) (*entities.User, error)
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, filter *ListUsersRequest) (*ListUsersResponse, error)

	// Authentication
	Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)
	Logout(ctx context.Context, token string) error
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
	ChangePassword(ctx context.Context, req *ChangePasswordRequest) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, req *ResetPasswordRequest) error

	// Role management
	AssignRole(ctx context.Context, userID, roleID string) error
	RemoveRole(ctx context.Context, userID, roleID string) error
	GetUserRoles(ctx context.Context, userID string) ([]*entities.Role, error)
	CreateRole(ctx context.Context, req *CreateRoleRequest) (*entities.Role, error)
	UpdateRole(ctx context.Context, id string, req *UpdateRoleRequest) (*entities.Role, error)
	DeleteRole(ctx context.Context, id string) error
	ListRoles(ctx context.Context, filter *ListRolesRequest) (*ListRolesResponse, error)
}

// Request/Response DTOs
type CreateUserRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Username  string `json:"username" validate:"required,min=3,max=50"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Phone     string `json:"phone,omitempty"`
}

type UpdateUserRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	User         *entities.User `json:"user"`
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	ExpiresIn    int            `json:"expires_in"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type CreateRoleRequest struct {
	Name        string   `json:"name" validate:"required,min=2,max=50"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions" validate:"required"`
}

type UpdateRoleRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type ListUsersRequest struct {
	Search     string `json:"search,omitempty"`
	IsActive   *bool  `json:"is_active,omitempty"`
	IsVerified *bool  `json:"is_verified,omitempty"`
	RoleID     string `json:"role_id,omitempty"`
	Page       int    `json:"page,omitempty" validate:"min=1"`
	Limit      int    `json:"limit,omitempty" validate:"min=1,max=100"`
	SortBy     string `json:"sort_by,omitempty"`
	SortOrder  string `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

type ListUsersResponse struct {
	Users      []*entities.User `json:"users"`
	Pagination *Pagination      `json:"pagination"`
}

type ListRolesRequest struct {
	Search    string `json:"search,omitempty"`
	Page      int    `json:"page,omitempty" validate:"min=1"`
	Limit     int    `json:"limit,omitempty" validate:"min=1,max=100"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

type ListRolesResponse struct {
	Roles      []*entities.Role `json:"roles"`
	Pagination *Pagination     `json:"pagination"`
}

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}