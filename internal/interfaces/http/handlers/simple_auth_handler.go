package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/internal/application/services/user"
	"erpgo/internal/domain/users/entities"
	"erpgo/internal/interfaces/http/dto"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	userService user.Service
	logger       zerolog.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userService user.Service, logger zerolog.Logger) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		logger:       logger,
	}
}

// Login handles user login
// @Summary User login
// @Description Authenticate a user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param login body dto.LoginRequest true "Login credentials"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &user.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	// Call service
	response, err := h.userService.Login(c, serviceReq)
	if err != nil {
		if err == user.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Invalid credentials",
				Details: "The email or password you provided is incorrect",
			})
			return
		}

		h.logger.Error().Err(err).Str("email", req.Email).Msg("Failed to login user")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Login failed",
			Details: err.Error(),
		})
		return
	}

	// Convert to DTO response
	c.JSON(http.StatusOK, dto.LoginResponse{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(response.ExpiresIn) * time.Second),
		User:         h.userToDTO(response.User),
	})
}

// Register handles user registration
// @Summary User registration
// @Description Register a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param register body dto.RegisterRequest true "Registration data"
// @Success 201 {object} dto.UserInfo
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &user.CreateUserRequest{
		Email:     req.Email,
		Username:  req.Username,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
	}

	// Call service
	user, err := h.userService.CreateUser(c, serviceReq)
	if err != nil {
		if err.Error() == "user already exists" {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Error:   "User already exists",
				Details: "A user with this email or username already exists",
			})
			return
		}

		h.logger.Error().Err(err).Str("email", req.Email).Msg("Failed to register user")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Registration failed",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, h.userToDTO(user))
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Get a new access token using a refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh body dto.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Call service
	response, err := h.userService.RefreshToken(c, req.RefreshToken)
	if err != nil {
		if err.Error() == "invalid token" || err.Error() == "token expired" {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Invalid refresh token",
				Details: "The refresh token is invalid or has expired",
			})
			return
		}

		h.logger.Error().Err(err).Msg("Failed to refresh token")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Token refresh failed",
			Details: err.Error(),
		})
		return
	}

	// Convert to DTO response (TokenResponse doesn't include user)
	c.JSON(http.StatusOK, dto.LoginResponse{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(response.ExpiresIn) * time.Second),
		User:         nil, // User not included in token refresh response
	})
}

// Logout handles user logout
// @Summary User logout
// @Description Logout a user and invalidate their token
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} dto.SuccessResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// In a real implementation, we would extract the token from the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Authorization header required",
			Details: "Please provide a valid authorization header",
		})
		return
	}

	// Extract token from Bearer header
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Invalid authorization header format",
			Details: "Authorization header must be in format: Bearer {token}",
		})
		return
	}

	token := authHeader[7:]

	// Call service
	err := h.userService.Logout(c, token)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to logout user")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Logout failed",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "Logged out successfully",
	})
}

// ForgotPassword handles password reset request
// @Summary Forgot password
// @Description Request a password reset email
// @Tags auth
// @Accept json
// @Produce json
// @Param forgot body dto.ForgotPasswordRequest true "Email address"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Call service
	err := h.userService.ForgotPassword(c, req.Email)
	if err != nil {
		h.logger.Error().Err(err).Str("email", req.Email).Msg("Failed to process forgot password")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to process request",
			Details: err.Error(),
		})
		return
	}

	// Always return success to prevent user enumeration
	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "If an account with this email exists, a password reset link has been sent",
	})
}

// ResetPassword handles password reset
// @Summary Reset password
// @Description Reset password using a reset token
// @Tags auth
// @Accept json
// @Produce json
// @Param reset body dto.ResetPasswordRequest true "Reset token and new password"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &user.ResetPasswordRequest{
		Token:    req.Token,
		Password: req.Password,
	}

	// Call service
	err := h.userService.ResetPassword(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to reset password")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to reset password",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "Password reset successfully",
	})
}

// Helper method to convert user entity to DTO
func (h *AuthHandler) userToDTO(user *entities.User) *dto.UserInfo {
	return &dto.UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Phone:     user.Phone,
		Roles:     []string{"user"}, // Default role for now
		IsActive:  user.IsActive,
		IsVerified: user.IsVerified,
		LastLogin: user.LastLoginAt,
		CreatedAt: user.CreatedAt,
	}
}