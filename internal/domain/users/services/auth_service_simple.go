package services

import (
	"context"
	"fmt"
	"time"

	"erpgo/internal/domain/users/entities"
	"erpgo/internal/domain/users/repositories"
	"erpgo/pkg/auth"
	"erpgo/pkg/cache"
	"erpgo/pkg/config"
	"github.com/google/uuid"
)

// SimpleAuthService handles authentication operations with minimal logging
type SimpleAuthService struct {
	userRepo        repositories.UserRepository
	jwtService      *auth.JWTService
	passwordService *auth.PasswordService
	cacheService    cache.Cache
	config          *config.Config
}

// NewSimpleAuthService creates a new simple authentication service
func NewSimpleAuthService(
	userRepo repositories.UserRepository,
	cfg *config.Config,
	cacheService cache.Cache,
) *SimpleAuthService {
	jwtConfig := cfg.GetJWTConfig()

	return &SimpleAuthService{
		userRepo:        userRepo,
		jwtService:      auth.NewJWTService(jwtConfig.Secret, "erpgo-api", jwtConfig.Expiry, jwtConfig.RefreshExpiry),
		passwordService: auth.NewPasswordService(cfg.BcryptCost, cfg.PasswordPepper),
		cacheService:    cacheService,
		config:          cfg,
	}
}

// LoginRequest represents a login request
type SimpleLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Remember bool   `json:"remember"`
}

// SimpleLoginResponse represents a login response
type SimpleLoginResponse struct {
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	ExpiresAt    time.Time       `json:"expires_at"`
	User         *SimpleUserInfo `json:"user"`
}

// SimpleUserInfo represents user information returned in responses
type SimpleUserInfo struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Roles     []string  `json:"roles"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// SimpleRegisterRequest represents a registration request
type SimpleRegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required,min=1,max=50"`
	LastName  string `json:"last_name" binding:"required,min=1,max=50"`
}

// SimpleRefreshTokenRequest represents a refresh token request
type SimpleRefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Login authenticates a user and returns tokens
func (s *SimpleAuthService) Login(ctx context.Context, req *SimpleLoginRequest) (*SimpleLoginResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	// Check password
	if !s.passwordService.CheckPassword(req.Password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Get user roles
	roles, err := s.userRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		roles = []string{"user"} // Default role
	}

	// Generate tokens
	accessToken, refreshToken, err := s.jwtService.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Username,
		roles,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens")
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	user.UpdatedAt = now
	s.userRepo.Update(ctx, user)

	// Store refresh token in cache
	refreshKey := fmt.Sprintf("refresh_token:%s", user.ID.String())
	s.cacheService.Set(ctx, refreshKey, refreshToken, s.config.RefreshExpiry)

	return &SimpleLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(s.config.JWTExpiry),
		User:         s.userToSimpleUserInfo(user, roles),
	}, nil
}

// RefreshToken generates a new access token using a refresh token
func (s *SimpleAuthService) RefreshToken(ctx context.Context, req *SimpleRefreshTokenRequest) (*SimpleLoginResponse, error) {
	// Validate refresh token and get user ID
	userID, err := s.jwtService.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Check if refresh token exists in cache
	refreshKey := fmt.Sprintf("refresh_token:%s", userID.String())
	cachedToken, err := s.cacheService.Get(ctx, refreshKey)
	if err != nil || cachedToken != req.RefreshToken {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check if user is still active
	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	// Get user roles
	roles, err := s.userRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		roles = []string{"user"} // Default role
	}

	// Generate new access token
	accessToken, err := s.jwtService.RefreshAccessToken(req.RefreshToken, user.ID, user.Email, user.Username, roles)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token")
	}

	// Generate new refresh token
	newRefreshToken, err := s.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token")
	}

	// Update refresh token in cache
	s.cacheService.Set(ctx, refreshKey, newRefreshToken, s.config.RefreshExpiry)

	return &SimpleLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(s.config.JWTExpiry),
		User:         s.userToSimpleUserInfo(user, roles),
	}, nil
}

// Logout invalidates the user's refresh token
func (s *SimpleAuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	// Remove refresh token from cache
	refreshKey := fmt.Sprintf("refresh_token:%s", userID.String())
	s.cacheService.Delete(ctx, refreshKey)
	return nil
}

// Register creates a new user account
func (s *SimpleAuthService) Register(ctx context.Context, req *SimpleRegisterRequest) (*SimpleUserInfo, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	existingUser, err = s.userRepo.GetByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with this username already exists")
	}

	// Hash password
	passwordHash, err := s.passwordService.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to process password")
	}

	// Create user
	user := &entities.User{
		ID:           uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: passwordHash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		IsActive:     true,
		IsVerified:   false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user")
	}

	// Assign default role
	defaultRole := "customer"
	s.userRepo.AssignRole(ctx, user.ID, defaultRole, uuid.Nil)

	return s.userToSimpleUserInfo(user, []string{defaultRole}), nil
}

// ValidateToken validates an access token and returns user information
func (s *SimpleAuthService) ValidateToken(ctx context.Context, tokenString string) (*SimpleUserInfo, error) {
	// Validate token
	claims, err := s.jwtService.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check if user is still active
	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	return s.userToSimpleUserInfo(user, claims.Roles), nil
}

// userToSimpleUserInfo converts a user entity to simple user info
func (s *SimpleAuthService) userToSimpleUserInfo(user *entities.User, roles []string) *SimpleUserInfo {
	return &SimpleUserInfo{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Roles:     roles,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	}
}
