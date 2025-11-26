package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"erpgo/internal/application/services/email"
	"erpgo/internal/domain/users/entities"
	"erpgo/internal/domain/users/repositories"
	"erpgo/pkg/auth"
	"erpgo/pkg/cache"
	"erpgo/pkg/database"
	apperrors "erpgo/pkg/errors"
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
	ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest) error
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

	// Permission management
	GetUserPermissions(ctx context.Context, userID string) ([]string, error)
	UserHasPermission(ctx context.Context, userID, permission string) (bool, error)
	UserHasAnyPermission(ctx context.Context, userID string, permissions ...string) (bool, error)
	UserHasAllPermissions(ctx context.Context, userID string, permissions ...string) (bool, error)
	CheckUserPermission(ctx context.Context, userID, permission string) error

	// Email verification
	SendVerificationEmail(ctx context.Context, email string) error
	VerifyEmail(ctx context.Context, token string) error
	ResendVerificationEmail(ctx context.Context, email string) error
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

// Errors
var (
	ErrUserNotFound          = errors.New("user not found")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrUserAlreadyExists     = errors.New("user already exists")
	ErrInvalidToken          = errors.New("invalid token")
	ErrTokenExpired          = errors.New("token expired")
	ErrInsufficientPermission = errors.New("insufficient permission")
	ErrRoleNotFound          = errors.New("role not found")
	ErrInvalidPassword       = errors.New("invalid password")
	ErrEmailAlreadyVerified  = errors.New("email already verified")
)

// ResetTokenInfo holds information about password reset tokens
type ResetTokenInfo struct {
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// ServiceImpl implements the user service interface
type ServiceImpl struct {
	userRepo              repositories.UserRepository
	roleRepo              repositories.RoleRepository
	userRoleRepo          repositories.UserRoleRepository
	passwordSvc           *auth.PasswordService
	jwtSvc                *auth.JWTService
	emailVerificationSvc  email.Service
	cache                 cache.Cache
	permissionCache       *cache.PermissionCache
	txManager             database.TransactionManagerInterface
	defaultRole           string
	resetTokens           map[string]*ResetTokenInfo  // Fallback for when cache is not available
	resetMutex            sync.RWMutex
}

// NewService creates a new user service instance
func NewService(
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
	userRoleRepo repositories.UserRoleRepository,
	passwordSvc *auth.PasswordService,
	jwtSvc *auth.JWTService,
	emailVerificationSvc email.Service,
	cacheInstance cache.Cache,
	txManager database.TransactionManagerInterface,
) Service {
	// Initialize permission cache with default config
	var permissionCache *cache.PermissionCache
	if cacheInstance != nil {
		logger := zerolog.Nop()
		permissionCache = cache.NewPermissionCache(cacheInstance, &logger, cache.DefaultPermissionCacheConfig())
	}
	
	return &ServiceImpl{
		userRepo:              userRepo,
		roleRepo:              roleRepo,
		userRoleRepo:          userRoleRepo,
		passwordSvc:           passwordSvc,
		jwtSvc:                jwtSvc,
		emailVerificationSvc:  emailVerificationSvc,
		cache:                 cacheInstance,
		permissionCache:       permissionCache,
		txManager:             txManager,
		defaultRole:           "student", // Default role for new users
		resetTokens:           make(map[string]*ResetTokenInfo),
	}
}

// NewUserService creates a new user service instance (alias for NewService for naming consistency)
func NewUserService(
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
	userRoleRepo repositories.UserRoleRepository,
	passwordSvc *auth.PasswordService,
	jwtSvc *auth.JWTService,
	emailVerificationSvc email.Service,
	cache cache.Cache,
	txManager database.TransactionManagerInterface,
) Service {
	return NewService(userRepo, roleRepo, userRoleRepo, passwordSvc, jwtSvc, emailVerificationSvc, cache, txManager)
}

// storeResetToken stores a reset token in Redis with expiration
func (s *ServiceImpl) storeResetToken(ctx context.Context, token string, info *ResetTokenInfo) error {
	if s.cache != nil {
		// Try to store in Redis first
		tokenData, err := json.Marshal(info)
		if err != nil {
			return fmt.Errorf("failed to marshal token info: %w", err)
		}

		// Store with 1 hour expiration
		key := fmt.Sprintf("reset_token:%s", token)
		if err := s.cache.Set(ctx, key, string(tokenData), time.Hour); err != nil {
			// Fallback to memory storage if Redis fails
			s.resetMutex.Lock()
			s.resetTokens[token] = info
			s.resetMutex.Unlock()
		}
	} else {
		// Fallback to memory storage
		s.resetMutex.Lock()
		s.resetTokens[token] = info
		s.resetMutex.Unlock()
	}

	return nil
}

// getResetToken retrieves a reset token from Redis or memory
func (s *ServiceImpl) getResetToken(ctx context.Context, token string) (*ResetTokenInfo, error) {
	if s.cache != nil {
		// Try to get from Redis first
		key := fmt.Sprintf("reset_token:%s", token)
		tokenData, err := s.cache.Get(ctx, key)
		if err == nil && tokenData != "" {
			var info ResetTokenInfo
			if err := json.Unmarshal([]byte(tokenData), &info); err != nil {
				return nil, fmt.Errorf("failed to unmarshal token info: %w", err)
			}
			return &info, nil
		}
	}

	// Fallback to memory storage
	s.resetMutex.RLock()
	info, exists := s.resetTokens[token]
	s.resetMutex.RUnlock()

	if !exists {
		return nil, ErrInvalidToken
	}

	// Check if token is expired
	if time.Now().UTC().After(info.ExpiresAt) {
		// Remove expired token
		s.deleteResetToken(ctx, token)
		return nil, ErrTokenExpired
	}

	return info, nil
}

// deleteResetToken removes a reset token from Redis and memory
func (s *ServiceImpl) deleteResetToken(ctx context.Context, token string) {
	if s.cache != nil {
		// Try to delete from Redis
		key := fmt.Sprintf("reset_token:%s", token)
		s.cache.Delete(ctx, key)
	}

	// Also remove from memory storage
	s.resetMutex.Lock()
	delete(s.resetTokens, token)
	s.resetMutex.Unlock()
}

// cleanupExpiredTokens removes expired tokens from memory storage
func (s *ServiceImpl) cleanupExpiredTokens() {
	s.resetMutex.Lock()
	defer s.resetMutex.Unlock()

	now := time.Now().UTC()
	for token, info := range s.resetTokens {
		if now.After(info.ExpiresAt) {
			delete(s.resetTokens, token)
		}
	}
}

// CreateUser creates a new user with email validation
func (s *ServiceImpl) CreateUser(ctx context.Context, req *CreateUserRequest) (*entities.User, error) {
	// Validate business rules
	if err := s.validateCreateUserRequest(ctx, req); err != nil {
		return nil, err
	}

	// Check if user already exists
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperrors.WrapInternalError(err, "failed to check if user exists").
			WithContext("operation", "CreateUser").
			WithContext("email", req.Email)
	}
	if exists {
		return nil, apperrors.NewConflictError("user with this email already exists").
			WithContext("email", req.Email)
	}

	exists, err = s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, apperrors.WrapInternalError(err, "failed to check if username exists").
			WithContext("operation", "CreateUser").
			WithContext("username", req.Username)
	}
	if exists {
		return nil, apperrors.NewConflictError("user with this username already exists").
			WithContext("username", req.Username)
	}

	// Hash password
	hashedPassword, err := s.passwordSvc.HashPassword(req.Password)
	if err != nil {
		return nil, apperrors.WrapInternalError(err, "failed to hash password").
			WithContext("operation", "CreateUser")
	}

	// Create user entity
	user := &entities.User{
		ID:           uuid.New(),
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		Username:     strings.TrimSpace(req.Username),
		PasswordHash: hashedPassword,
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
		Phone:        strings.TrimSpace(req.Phone),
		IsActive:     true,
		IsVerified:   false, // Will need email verification
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Validate user entity
	if err := user.Validate(); err != nil {
		// The error from Validate is already a ValidationError with field details
		return nil, err
	}

	// Execute user creation and role assignment within a transaction
	err = s.txManager.WithRetryTransaction(ctx, func(tx pgx.Tx) error {
		// Save user to database
		if err := s.userRepo.Create(ctx, user); err != nil {
			return apperrors.ClassifyDatabaseError(err, "CreateUser")
		}

		// Assign default role
		if err := s.userRepo.AssignRole(ctx, user.ID, s.defaultRole, user.ID); err != nil {
			return apperrors.WrapInternalError(err, "failed to assign default role").
				WithContext("operation", "CreateUser").
				WithContext("user_id", user.ID.String()).
				WithContext("role", s.defaultRole)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Return user without sensitive data
	return user.ToSafeUser(), nil
}

// GetUser retrieves a user by ID
func (s *ServiceImpl) GetUser(ctx context.Context, id string) (*entities.User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, apperrors.NewBadRequestError("invalid user ID format").
			WithContext("user_id", id)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		dbErr := apperrors.ClassifyDatabaseError(err, "GetUser")
		if appErr, ok := dbErr.(*apperrors.AppError); ok {
			appErr.WithContext("user_id", id)
		}
		return nil, dbErr
	}

	return user.ToSafeUser(), nil
}

// GetUserByEmail retrieves a user by email
func (s *ServiceImpl) GetUserByEmail(ctx context.Context, email string) (*entities.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, apperrors.NewBadRequestError("email cannot be empty")
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		dbErr := apperrors.ClassifyDatabaseError(err, "GetUserByEmail")
		if appErr, ok := dbErr.(*apperrors.AppError); ok {
			appErr.WithContext("email", email)
		}
		return nil, dbErr
	}

	return user.ToSafeUser(), nil
}

// UpdateUser updates user profile information
func (s *ServiceImpl) UpdateUser(ctx context.Context, id string, req *UpdateUserRequest) (*entities.User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get existing user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update fields if provided
	if req.FirstName != nil {
		user.FirstName = strings.TrimSpace(*req.FirstName)
	}
	if req.LastName != nil {
		user.LastName = strings.TrimSpace(*req.LastName)
	}
	if req.Phone != nil {
		user.Phone = strings.TrimSpace(*req.Phone)
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	// Validate updated user
	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("invalid user data: %w", err)
	}

	// Save updates
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user.ToSafeUser(), nil
}

// DeleteUser deletes a user (soft delete by setting inactive)
func (s *ServiceImpl) DeleteUser(ctx context.Context, id string) error {
	userID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if user exists
	_, err = s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Perform soft delete
	return s.userRepo.Delete(ctx, userID)
}

// ListUsers retrieves a paginated list of users
func (s *ServiceImpl) ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Build filter
	filter := repositories.UserFilter{
		Search:     strings.TrimSpace(req.Search),
		IsActive:   req.IsActive,
		IsVerified: req.IsVerified,
		RoleID:     req.RoleID,
		Page:       req.Page,
		Limit:      req.Limit,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
	}

	// Get users and total count
	users, err := s.userRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	total, err := s.userRepo.Count(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Convert to safe users
	safeUsers := make([]*entities.User, len(users))
	for i, user := range users {
		safeUsers[i] = user.ToSafeUser()
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))
	hasNext := req.Page < totalPages
	hasPrev := req.Page > 1

	return &ListUsersResponse{
		Users: safeUsers,
		Pagination: &Pagination{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    hasNext,
			HasPrev:    hasPrev,
		},
	}, nil
}

// Login authenticates a user and returns tokens
func (s *ServiceImpl) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// Validate input
	if err := s.validateLoginRequest(req); err != nil {
		return nil, err
	}

	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		// Don't reveal whether user exists - return generic error
		return nil, apperrors.NewUnauthorizedError("invalid credentials").
			WithContext("operation", "Login")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, apperrors.NewUnauthorizedError("invalid credentials").
			WithContext("operation", "Login")
	}

	// Verify password
	if !s.passwordSvc.CheckPassword(req.Password, user.PasswordHash) {
		return nil, apperrors.NewUnauthorizedError("invalid credentials").
			WithContext("operation", "Login").
			WithContext("user_id", user.ID.String())
	}

	// Get user roles
	roles, err := s.userRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, apperrors.WrapInternalError(err, "failed to get user roles").
			WithContext("operation", "Login").
			WithContext("user_id", user.ID.String())
	}

	// Update last login
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		// Log error but don't fail login
	}

	// Generate tokens
	accessToken, refreshToken, err := s.jwtSvc.GenerateTokenPair(user.ID, user.Email, user.Username, roles)
	if err != nil {
		return nil, apperrors.WrapInternalError(err, "failed to generate tokens").
			WithContext("operation", "Login").
			WithContext("user_id", user.ID.String())
	}

	return &LoginResponse{
		User:         user.ToSafeUser(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.jwtSvc.GetAccessExpiry().Seconds()),
	}, nil
}

// Logout handles user logout (token invalidation)
func (s *ServiceImpl) Logout(ctx context.Context, token string) error {
	// Validate token first
	_, err := s.jwtSvc.ValidateToken(token)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Add token to blacklist for immediate invalidation
	err = s.jwtSvc.InvalidateToken(ctx, token)
	if err != nil {
		// Log error but don't fail logout - token will expire naturally
		// In production, you might want to handle this more strictly
		return fmt.Errorf("failed to invalidate token: %w", err)
	}

	// Optionally: Invalidate all user tokens for enhanced security
	// This would require the user to login again on all devices
	// Uncomment the following lines if you want this behavior:
	/*
	err = s.jwtSvc.InvalidateUserTokens(ctx, claims.UserID)
	if err != nil {
		return fmt.Errorf("failed to invalidate user tokens: %w", err)
	}
	*/

	return nil
}

// RefreshToken generates new tokens from a refresh token using token rotation
func (s *ServiceImpl) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	// Validate refresh token
	claims, err := s.jwtSvc.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user to ensure they still exist and are active
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrInvalidCredentials
	}

	// Get current user roles
	roles, err := s.userRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	// Use refresh token rotation - invalidate old refresh token and generate new pair
	accessToken, newRefreshToken, err := s.jwtSvc.RefreshTokenRotation(ctx, refreshToken, user.ID, user.Email, user.Username, roles)
	if err != nil {
		return nil, fmt.Errorf("failed to rotate refresh token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(s.jwtSvc.GetAccessExpiry().Seconds()),
	}, nil
}

// ChangePassword changes user password
func (s *ServiceImpl) ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest) error {
	// Validate input
	if strings.TrimSpace(req.OldPassword) == "" {
		return errors.New("old password is required")
	}
	if strings.TrimSpace(req.NewPassword) == "" {
		return errors.New("new password is required")
	}

	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Get user from database
	user, err := s.userRepo.GetByID(ctx, userUUID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return errors.New("user account is not active")
	}

	// Verify old password
	if !s.passwordSvc.CheckPassword(req.OldPassword, user.PasswordHash) {
		return ErrInvalidCredentials
	}

	// Check if new password is same as old password
	if s.passwordSvc.CheckPassword(req.NewPassword, user.PasswordHash) {
		return errors.New("new password must be different from old password")
	}

	// Validate new password
	validation := s.passwordSvc.ValidatePassword(req.NewPassword)
	if !validation.Valid {
		return fmt.Errorf("new password validation failed: %s", strings.Join(validation.Errors, ", "))
	}

	// Hash new password
	newPasswordHash, err := s.passwordSvc.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update user password
	user.PasswordHash = newPasswordHash
	user.UpdatedAt = time.Now().UTC()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	return nil
}

// ForgotPassword initiates password reset flow
func (s *ServiceImpl) ForgotPassword(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return errors.New("email cannot be empty")
	}

	// Check if user exists
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			// Don't reveal if user exists or not for security
			return nil
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		// Don't reveal if user is inactive for security
		return nil
	}

	// Generate reset token
	resetToken, err := s.passwordSvc.GenerateResetToken()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// Store reset token with 1 hour expiration using the helper method
	expiresAt := time.Now().UTC().Add(1 * time.Hour)
	tokenInfo := &ResetTokenInfo{
		UserID:    user.ID,
		Token:     resetToken,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.storeResetToken(ctx, resetToken, tokenInfo); err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	// Clean up expired tokens
	go s.cleanupExpiredTokens()

	// Send password reset email if email service is available
	if s.emailVerificationSvc != nil {
		// Cast to email service interface to access SendPasswordResetEmail
		if emailSvc, ok := s.emailVerificationSvc.(interface{ SendPasswordResetEmail(email, token string) error }); ok {
			if err := emailSvc.SendPasswordResetEmail(email, resetToken); err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Warning: failed to send password reset email: %v\n", err)
			}
		}
	} else {
		// Fallback to logging for development
		fmt.Printf("Password reset token for %s (user ID: %s): %s (expires: %s)\n",
			email, user.ID.String(), resetToken, expiresAt.Format(time.RFC3339))
	}

	return nil
}

// ResetPassword resets password using reset token
func (s *ServiceImpl) ResetPassword(ctx context.Context, req *ResetPasswordRequest) error {
	// Validate input
	if strings.TrimSpace(req.Token) == "" {
		return errors.New("reset token is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return errors.New("password is required")
	}

	// Check if reset token exists and is valid using helper method
	tokenInfo, err := s.getResetToken(ctx, req.Token)
	if err != nil {
		if err == ErrTokenExpired {
			return ErrTokenExpired
		}
		return ErrInvalidToken
	}

	// Get user from token
	user, err := s.userRepo.GetByID(ctx, tokenInfo.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return errors.New("user account is not active")
	}

	// Validate new password
	validation := s.passwordSvc.ValidatePassword(req.Password)
	if !validation.Valid {
		return fmt.Errorf("password validation failed: %s", strings.Join(validation.Errors, ", "))
	}

	// Hash new password
	newPasswordHash, err := s.passwordSvc.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user password
	user.PasswordHash = newPasswordHash
	user.UpdatedAt = time.Now().UTC()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	// Remove used token using helper method
	s.deleteResetToken(ctx, req.Token)

	return nil
}

// AssignRole assigns a role to a user
func (s *ServiceImpl) AssignRole(ctx context.Context, userID, roleID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		return fmt.Errorf("invalid role ID: %w", err)
	}

	// Execute role assignment within a transaction
	return s.txManager.WithRetryTransaction(ctx, func(tx pgx.Tx) error {
		// Check if user exists
		_, err := s.userRepo.GetByID(ctx, userUUID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return ErrUserNotFound
			}
			return fmt.Errorf("failed to get user: %w", err)
		}

		// Check if role exists
		_, err = s.roleRepo.GetRoleByID(ctx, roleUUID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return ErrRoleNotFound
			}
			return fmt.Errorf("failed to get role: %w", err)
		}

		// Assign role
		if err := s.userRoleRepo.AssignRole(ctx, userID, roleID, userID); err != nil {
			return fmt.Errorf("failed to assign role: %w", err)
		}

		// Invalidate user permission and role cache (Requirement 7.3)
		if s.permissionCache != nil {
			// Invalidate both permissions and roles since role assignment affects both
			if err := s.permissionCache.InvalidateUser(ctx, userUUID); err != nil {
				// Log error but don't fail the transaction
				// Cache invalidation failure should not prevent role assignment
			}
		} else if s.cache != nil {
			// Fallback to old cache invalidation method
			cacheKey := fmt.Sprintf("user:permissions:%s", userID)
			s.cache.Delete(ctx, cacheKey)
		}

		return nil
	})
}

// RemoveRole removes a role from a user
func (s *ServiceImpl) RemoveRole(ctx context.Context, userID, roleID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Execute role removal within a transaction
	return s.txManager.WithRetryTransaction(ctx, func(tx pgx.Tx) error {
		// Check if user exists
		_, err := s.userRepo.GetByID(ctx, userUUID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return ErrUserNotFound
			}
			return fmt.Errorf("failed to get user: %w", err)
		}

		// Remove role
		if err := s.userRoleRepo.RemoveRole(ctx, userID, roleID); err != nil {
			return fmt.Errorf("failed to remove role: %w", err)
		}

		// Invalidate user permission and role cache (Requirement 7.3)
		if s.permissionCache != nil {
			// Invalidate both permissions and roles since role removal affects both
			if err := s.permissionCache.InvalidateUser(ctx, userUUID); err != nil {
				// Log error but don't fail the transaction
				// Cache invalidation failure should not prevent role removal
			}
		} else if s.cache != nil {
			// Fallback to old cache invalidation method
			cacheKey := fmt.Sprintf("user:permissions:%s", userID)
			s.cache.Delete(ctx, cacheKey)
		}

		return nil
	})
}

// GetUserRoles retrieves roles for a user
func (s *ServiceImpl) GetUserRoles(ctx context.Context, userID string) ([]*entities.Role, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if user exists
	_, err = s.userRepo.GetByID(ctx, userUUID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get roles
	return s.userRoleRepo.GetUserRoles(ctx, userID)
}

// CreateRole creates a new role
func (s *ServiceImpl) CreateRole(ctx context.Context, req *CreateRoleRequest) (*entities.Role, error) {
	// Validate input
	if err := s.validateCreateRoleRequest(req); err != nil {
		return nil, err
	}

	// Check if role already exists
	existingRole, err := s.roleRepo.GetRoleByName(ctx, req.Name)
	if err == nil && existingRole != nil {
		return nil, errors.New("role already exists")
	}

	// Create role entity
	role := &entities.Role{
		ID:          uuid.New(),
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Permissions: req.Permissions,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Save role
	if err := s.roleRepo.CreateRole(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return role, nil
}

// UpdateRole updates an existing role
func (s *ServiceImpl) UpdateRole(ctx context.Context, id string, req *UpdateRoleRequest) (*entities.Role, error) {
	roleID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid role ID: %w", err)
	}

	// Get existing role
	role, err := s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// Update fields if provided
	if req.Name != nil {
		role.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		role.Description = strings.TrimSpace(*req.Description)
	}
	if req.Permissions != nil {
		role.Permissions = req.Permissions
	}

	// Validate role data
	if role.Name == "" {
		return nil, fmt.Errorf("role name cannot be empty")
	}

	// Save updates
	if err := s.roleRepo.UpdateRole(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return role, nil
}

// DeleteRole deletes a role
func (s *ServiceImpl) DeleteRole(ctx context.Context, id string) error {
	roleID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid role ID: %w", err)
	}

	// Check if role exists
	_, err = s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrRoleNotFound
		}
		return fmt.Errorf("failed to get role: %w", err)
	}

	// Delete role
	return s.roleRepo.DeleteRole(ctx, roleID)
}

// ListRoles retrieves a paginated list of roles
func (s *ServiceImpl) ListRoles(ctx context.Context, req *ListRolesRequest) (*ListRolesResponse, error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Get roles
	roles, err := s.roleRepo.GetAllRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	total := len(roles)

	// Calculate pagination
	totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))
	hasNext := req.Page < totalPages
	hasPrev := req.Page > 1

	return &ListRolesResponse{
		Roles: roles,
		Pagination: &Pagination{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    hasNext,
			HasPrev:    hasPrev,
		},
	}, nil
}

// Permission management implementation

// GetUserPermissions retrieves all permissions for a user
// Uses cache with 5 minute TTL per Requirement 7.1
func (s *ServiceImpl) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Try to get from cache first
	if s.permissionCache != nil {
		cachedPermissions, err := s.permissionCache.GetUserPermissions(ctx, userUUID)
		if err == nil && cachedPermissions != nil {
			// Cache hit
			return cachedPermissions, nil
		}
	}

	// Cache miss or cache unavailable - fetch from database
	permissions, err := s.roleRepo.GetUserPermissions(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	// Store in cache for future requests
	if s.permissionCache != nil {
		if cacheErr := s.permissionCache.SetUserPermissions(ctx, userUUID, permissions); cacheErr != nil {
			// Log error but don't fail the request
			// Cache failure should not prevent the operation
		}
	}

	return permissions, nil
}

// UserHasPermission checks if a user has a specific permission
func (s *ServiceImpl) UserHasPermission(ctx context.Context, userID, permission string) (bool, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	return s.roleRepo.UserHasPermission(ctx, userUUID, permission)
}

// UserHasAnyPermission checks if a user has any of the specified permissions
func (s *ServiceImpl) UserHasAnyPermission(ctx context.Context, userID string, permissions ...string) (bool, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	return s.roleRepo.UserHasAnyPermission(ctx, userUUID, permissions...)
}

// UserHasAllPermissions checks if a user has all of the specified permissions
func (s *ServiceImpl) UserHasAllPermissions(ctx context.Context, userID string, permissions ...string) (bool, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	return s.roleRepo.UserHasAllPermissions(ctx, userUUID, permissions...)
}

// CheckUserPermission checks if a user has a specific permission and returns an error if not
func (s *ServiceImpl) CheckUserPermission(ctx context.Context, userID, permission string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	hasPermission, err := s.roleRepo.UserHasPermission(ctx, userUUID, permission)
	if err != nil {
		return fmt.Errorf("failed to check user permission: %w", err)
	}

	if !hasPermission {
		return fmt.Errorf("user does not have required permission: %s", permission)
	}

	return nil
}

// Email verification implementation

// SendVerificationEmail sends a verification email to a user
func (s *ServiceImpl) SendVerificationEmail(ctx context.Context, email string) error {
	req := &entities.EmailVerificationRequest{
		Email:    email,
		TokenType: entities.TokenTypeVerification,
	}

	return s.emailVerificationSvc.SendVerificationEmail(ctx, req)
}

// VerifyEmail verifies an email verification token
func (s *ServiceImpl) VerifyEmail(ctx context.Context, token string) error {
	req := &entities.VerifyEmailRequest{
		Token: token,
	}

	response, err := s.emailVerificationSvc.VerifyEmail(ctx, req)
	if err != nil {
		return err
	}

	if !response.Success {
		return errors.New(response.Message)
	}

	return nil
}

// ResendVerificationEmail resends a verification email
func (s *ServiceImpl) ResendVerificationEmail(ctx context.Context, email string) error {
	req := &entities.ResendVerificationRequest{
		Email: email,
	}

	return s.emailVerificationSvc.ResendVerificationEmail(ctx, req)
}

// Helper methods for validation

func (s *ServiceImpl) validateCreateUserRequest(ctx context.Context, req *CreateUserRequest) error {
	if strings.TrimSpace(req.Email) == "" {
		return errors.New("email is required")
	}
	if strings.TrimSpace(req.Username) == "" {
		return errors.New("username is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return errors.New("password is required")
	}
	if strings.TrimSpace(req.FirstName) == "" {
		return errors.New("first name is required")
	}
	if strings.TrimSpace(req.LastName) == "" {
		return errors.New("last name is required")
	}
	return nil
}

func (s *ServiceImpl) validateLoginRequest(req *LoginRequest) error {
	if strings.TrimSpace(req.Email) == "" {
		return errors.New("email is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return errors.New("password is required")
	}
	return nil
}

func (s *ServiceImpl) validateCreateRoleRequest(req *CreateRoleRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("role name is required")
	}
	if len(req.Permissions) == 0 {
		return errors.New("at least one permission is required")
	}
	return nil
}