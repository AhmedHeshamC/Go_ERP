package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/internal/application/services/user"
	"erpgo/internal/domain/users/entities"
	"erpgo/internal/interfaces/http/dto"
)

// UserHandler handles user HTTP requests
type UserHandler struct {
	userService user.Service
	logger       zerolog.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService user.Service, logger zerolog.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:       logger,
	}
}

// GetUsers returns a paginated list of users
// @Summary List users
// @Description Get a paginated list of users with optional filtering
// @Tags users
// @Accept json
// @Produce json
// @Param search query string false "Search term"
// @Param is_active query bool false "Filter by active status"
// @Param is_verified query bool false "Filter by verification status"
// @Param role_id query string false "Filter by role ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param sort_by query string false "Sort field" default("created_at")
// @Param sort_order query string false "Sort order" Enums(asc,desc) default("desc")
// @Success 200 {object} dto.ListUsersResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	// Parse query parameters
	req := &user.ListUsersRequest{
		Search:     c.Query("search"),
		SortBy:     c.Query("sort_by"),
		SortOrder:  c.Query("sort_order"),
	}

	// Parse boolean parameters
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			req.IsActive = &isActive
		}
	}

	if isVerifiedStr := c.Query("is_verified"); isVerifiedStr != "" {
		if isVerified, err := strconv.ParseBool(isVerifiedStr); err == nil {
			req.IsVerified = &isVerified
		}
	}

	// Parse pagination parameters
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			req.Limit = limit
		}
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}

	// Call service
	response, err := h.userService.ListUsers(c, req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list users")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to list users",
			Details: err.Error(),
		})
		return
	}

	// Convert to DTO response
	users := make([]*dto.UserInfo, len(response.Users))
	for i, user := range response.Users {
		users[i] = h.userToDTO(user)
	}

	c.JSON(http.StatusOK, dto.ListUsersResponse{
		Users:      users,
		Pagination: h.paginationToDTO(response.Pagination),
	})
}

// GetUser returns a user by ID
// @Summary Get user by ID
// @Description Get user information by user ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} dto.UserInfo
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "User ID is required",
			Details: "User ID parameter is missing",
		})
		return
	}

	// Call service
	user, err := h.userService.GetUser(c, userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "User not found",
				Details: "No user found with the provided ID",
			})
			return
		}

		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to get user",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, h.userToDTO(user))
}

// UpdateUser updates a user by ID
// @Summary Update user
// @Description Update user information by user ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body dto.UpdateUserRequest true "User update data"
// @Success 200 {object} dto.UserInfo
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "User ID is required",
			Details: "User ID parameter is missing",
		})
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &user.UpdateUserRequest{}
	if req.FirstName != nil {
		serviceReq.FirstName = req.FirstName
	}
	if req.LastName != nil {
		serviceReq.LastName = req.LastName
	}
	if req.Phone != nil {
		serviceReq.Phone = req.Phone
	}
	if req.IsActive != nil {
		serviceReq.IsActive = req.IsActive
	}

	// Call service
	updatedUser, err := h.userService.UpdateUser(c, userID, serviceReq)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "User not found",
				Details: "No user found with the provided ID",
			})
			return
		}

		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to update user")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to update user",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, h.userToDTO(updatedUser))
}

// DeleteUser deletes a user by ID
// @Summary Delete user
// @Description Delete a user by user ID (soft delete)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "User ID is required",
			Details: "User ID parameter is missing",
		})
		return
	}

	// Call service
	err := h.userService.DeleteUser(c, userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "User not found",
				Details: "No user found with the provided ID",
			})
			return
		}

		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to delete user")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to delete user",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "User deleted successfully",
	})
}

// GetUserRoles returns roles for a user
// @Summary Get user roles
// @Description Get all roles assigned to a user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {array} dto.RoleInfo
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users/{id}/roles [get]
func (h *UserHandler) GetUserRoles(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "User ID is required",
			Details: "User ID parameter is missing",
		})
		return
	}

	// Call service
	roles, err := h.userService.GetUserRoles(c, userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "User not found",
				Details: "No user found with the provided ID",
			})
			return
		}

		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user roles")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to get user roles",
			Details: err.Error(),
		})
		return
	}

	// Convert to DTO
	roleDTOs := make([]*dto.RoleInfo, len(roles))
	for i, role := range roles {
		roleDTOs[i] = h.roleToDTO(role)
	}

	c.JSON(http.StatusOK, roleDTOs)
}

// GetUserProfile returns the current user's profile
// @Summary Get current user profile
// @Description Get the current authenticated user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} dto.UserInfo
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/profile [get]
func (h *UserHandler) GetUserProfile(c *gin.Context) {
	// Extract user ID from context (JWT middleware should set this)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "User not authenticated",
			Details: "No user ID found in request context",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Invalid user ID format",
			Details: "User ID in context is not a valid string",
		})
		return
	}

	// Call service
	user, err := h.userService.GetUser(c, userIDStr)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "User not found",
				Details: "No user found with the provided ID",
			})
			return
		}

		h.logger.Error().Err(err).Str("user_id", userIDStr).Msg("Failed to get user profile")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to get user profile",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, h.userToDTO(user))
}

// UpdateUserProfile updates the current user's profile
// @Summary Update current user profile
// @Description Update the current authenticated user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Param profile body dto.UpdateUserRequest true "Profile update data"
// @Success 200 {object} dto.UserInfo
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/profile [put]
func (h *UserHandler) UpdateUserProfile(c *gin.Context) {
	// Extract user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "User not authenticated",
			Details: "No user ID found in request context",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Invalid user ID format",
			Details: "User ID in context is not a valid string",
		})
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &user.UpdateUserRequest{}
	if req.FirstName != nil {
		serviceReq.FirstName = req.FirstName
	}
	if req.LastName != nil {
		serviceReq.LastName = req.LastName
	}
	if req.Phone != nil {
		serviceReq.Phone = req.Phone
	}
	// Note: Users cannot change their active status through profile update
	// Only admins can change user active status

	// Call service
	updatedUser, err := h.userService.UpdateUser(c, userIDStr, serviceReq)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "User not found",
				Details: "No user found with the provided ID",
			})
			return
		}

		h.logger.Error().Err(err).Str("user_id", userIDStr).Msg("Failed to update user profile")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to update user profile",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, h.userToDTO(updatedUser))
}

// Helper methods

func (h *UserHandler) userToDTO(user *entities.User) *dto.UserInfo {
	// For now, return basic user info without roles
	// In a real implementation, we would get roles from the user entity directly
	// or make an additional service call
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

func (h *UserHandler) roleToDTO(role *entities.Role) *dto.RoleInfo {
	return &dto.RoleInfo{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		Permissions: role.Permissions,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}

func (h *UserHandler) paginationToDTO(pagination *user.Pagination) *dto.Pagination {
	return &dto.Pagination{
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		Total:      pagination.Total,
		TotalPages: pagination.TotalPages,
		HasNext:    pagination.HasNext,
		HasPrev:    pagination.HasPrev,
	}
}