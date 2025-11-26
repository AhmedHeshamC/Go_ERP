package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"erpgo/internal/application/services/user"
	"erpgo/internal/domain/users/entities"
	"erpgo/internal/interfaces/http/dto"
	"erpgo/internal/interfaces/http/middleware"
	"erpgo/pkg/audit"
	"erpgo/pkg/ratelimit"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	userService user.Service
	logger       zerolog.Logger
	rateLimiter ratelimit.EnhancedRateLimiter
	auditLogger audit.AuditLogger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userService user.Service, logger zerolog.Logger) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		logger:       logger,
		rateLimiter: nil, // Will be set via SetRateLimiter if needed
		auditLogger: nil, // Will be set via SetAuditLogger if needed
	}
}

// SetAuditLogger sets the audit logger for the auth handler
func (h *AuthHandler) SetAuditLogger(logger audit.AuditLogger) {
	h.auditLogger = logger
}

// SetRateLimiter sets the rate limiter for the auth handler
func (h *AuthHandler) SetRateLimiter(limiter ratelimit.EnhancedRateLimiter) {
	h.rateLimiter = limiter
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
// @Failure 429 {object} dto.ErrorResponse
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

	// Check rate limit if rate limiter is configured
	if h.rateLimiter != nil {
		// Check rate limit by IP
		ipAddress := c.ClientIP()
		allowed, err := h.rateLimiter.AllowLogin(c.Request.Context(), ipAddress)
		if err != nil || !allowed {
			h.logger.Warn().
				Str("ip", ipAddress).
				Str("email", req.Email).
				Msg("Rate limit exceeded for login attempt")

			// Check if account is locked
			if err != nil {
				if rateLimitErr, ok := err.(*ratelimit.RateLimitError); ok {
					c.JSON(http.StatusTooManyRequests, dto.ErrorResponse{
						Error:   "Rate limit exceeded",
						Details: rateLimitErr.Message,
					})
					return
				}
			}

			c.JSON(http.StatusTooManyRequests, dto.ErrorResponse{
				Error:   "Too many login attempts",
				Details: "Please try again later",
			})
			return
		}

		// Check if account is locked
		isLocked, unlockTime, err := h.rateLimiter.IsAccountLocked(c.Request.Context(), req.Email)
		if err == nil && isLocked {
			h.logger.Warn().
				Str("email", req.Email).
				Time("unlock_time", unlockTime).
				Msg("Login attempt for locked account")

			c.JSON(http.StatusTooManyRequests, dto.ErrorResponse{
				Error:   "Account locked",
				Details: "Account is locked due to too many failed login attempts. Please try again at " + unlockTime.Format(time.RFC3339),
			})
			return
		}
	}

	// Convert to service request
	serviceReq := &user.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	// Call service
	response, err := h.userService.Login(c, serviceReq)
	if err != nil {
		// Record failed login attempt if rate limiter is configured
		if h.rateLimiter != nil {
			if recordErr := h.rateLimiter.RecordFailedLogin(c.Request.Context(), req.Email); recordErr != nil {
				h.logger.Error().Err(recordErr).Str("email", req.Email).Msg("Failed to record failed login attempt")
			}
		}

		// Log failed login attempt to audit log
		if h.auditLogger != nil {
			auditEvent := audit.NewLoginFailedEvent(req.Email, c.ClientIP(), c.Request.UserAgent(), err.Error())
			if auditErr := h.auditLogger.LogEvent(c.Request.Context(), auditEvent); auditErr != nil {
				h.logger.Error().Err(auditErr).Msg("Failed to log audit event for failed login")
			}
		}

		h.logger.Error().Err(err).Str("email", req.Email).Msg("Failed to login user")
		middleware.HandleError(c, err)
		return
	}

	// Log successful login to audit log
	if h.auditLogger != nil {
		auditEvent := audit.NewLoginEvent(response.User.ID, c.ClientIP(), c.Request.UserAgent())
		if auditErr := h.auditLogger.LogEvent(c.Request.Context(), auditEvent); auditErr != nil {
			h.logger.Error().Err(auditErr).Msg("Failed to log audit event for successful login")
		}
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

	// Log logout to audit log
	// Try to get user ID from context (if available from auth middleware)
	if h.auditLogger != nil {
		if userIDVal, exists := c.Get("user_id"); exists {
			if userID, ok := userIDVal.(string); ok {
				if parsedUserID, parseErr := uuid.Parse(userID); parseErr == nil {
					auditEvent := audit.NewLogoutEvent(parsedUserID, c.ClientIP(), c.Request.UserAgent())
					if auditErr := h.auditLogger.LogEvent(c.Request.Context(), auditEvent); auditErr != nil {
						h.logger.Error().Err(auditErr).Msg("Failed to log audit event for logout")
					}
				}
			}
		}
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
		
		// Log failed password change to audit log
		if h.auditLogger != nil {
			// We don't have user ID here, but we can log the attempt
			auditEvent := &audit.AuditEvent{
				EventType: audit.EventTypePasswordChange,
				Action:    "password_reset_failed",
				IPAddress: c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
				Success:   false,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			}
			if auditErr := h.auditLogger.LogEvent(c.Request.Context(), auditEvent); auditErr != nil {
				h.logger.Error().Err(auditErr).Msg("Failed to log audit event for failed password reset")
			}
		}
		
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to reset password",
			Details: err.Error(),
		})
		return
	}

	// Log successful password change to audit log
	if h.auditLogger != nil {
		// We don't have user ID here, but we can log the success
		auditEvent := &audit.AuditEvent{
			EventType: audit.EventTypePasswordChange,
			Action:    "password_reset_success",
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Success:   true,
		}
		if auditErr := h.auditLogger.LogEvent(c.Request.Context(), auditEvent); auditErr != nil {
			h.logger.Error().Err(auditErr).Msg("Failed to log audit event for successful password reset")
		}
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