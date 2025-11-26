package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"erpgo/internal/domain/users/entities"
	"erpgo/internal/domain/users/repositories"
	"erpgo/internal/interfaces/http/dto"
	"erpgo/pkg/audit"
	"erpgo/pkg/auth"
)

// RoleHandler handles role management HTTP requests
type RoleHandler struct {
	roleRepo    repositories.RoleRepository
	logger      zerolog.Logger
	auditLogger audit.AuditLogger
}

// NewRoleHandler creates a new role handler
func NewRoleHandler(roleRepo repositories.RoleRepository, logger zerolog.Logger) *RoleHandler {
	return &RoleHandler{
		roleRepo:    roleRepo,
		logger:      logger,
		auditLogger: nil, // Will be set via SetAuditLogger if needed
	}
}

// SetAuditLogger sets the audit logger for the role handler
func (h *RoleHandler) SetAuditLogger(logger audit.AuditLogger) {
	h.auditLogger = logger
}

// CreateRole handles role creation
// @Summary Create a new role
// @Description Create a new role with optional permissions
// @Tags roles
// @Accept json
// @Produce json
// @Param role body dto.CreateRoleRequest true "Role data"
// @Success 201 {object} dto.RoleResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/roles [post]
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Check if role already exists
	exists, err := h.roleRepo.RoleExists(c, req.Name)
	if err != nil {
		h.logger.Error().Err(err).Str("role_name", req.Name).Msg("Failed to check if role exists")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to create role",
			Details: "Internal server error",
		})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Role already exists",
			Details: "A role with this name already exists",
		})
		return
	}

	// Create role entity
	role := &entities.Role{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create role in database
	if err := h.roleRepo.CreateRole(c, role); err != nil {
		h.logger.Error().Err(err).Str("role_name", req.Name).Msg("Failed to create role")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to create role",
			Details: "Internal server error",
		})
		return
	}

	// Add permissions if provided
	if len(req.Permissions) > 0 {
		for _, permission := range req.Permissions {
			if err := h.roleRepo.AddPermissionToRole(c, role.ID, permission); err != nil {
				h.logger.Error().Err(err).
					Str("role_id", role.ID.String()).
					Str("permission", permission).
					Msg("Failed to add permission to role")
				// Continue with other permissions even if one fails
			}
		}
	}

	// Get permissions for response
	permissions, _ := h.roleRepo.GetRolePermissions(c, role.ID)

	c.JSON(http.StatusCreated, dto.RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		Permissions: permissions,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	})
}

// GetRoles handles retrieving all roles
// @Summary Get all roles
// @Description Retrieve a list of all roles with optional pagination
// @Tags roles
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} dto.PaginatedResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/roles [get]
func (h *RoleHandler) GetRoles(c *gin.Context) {
	roles, err := h.roleRepo.GetAllRoles(c)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get roles")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to retrieve roles",
			Details: "Internal server error",
		})
		return
	}

	// Convert to response format
	var roleResponses []dto.RoleResponse
	for _, role := range roles {
		permissions, _ := h.roleRepo.GetRolePermissions(c, role.ID)
		roleResponses = append(roleResponses, dto.RoleResponse{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
			Permissions: permissions,
			CreatedAt:   role.CreatedAt,
			UpdatedAt:   role.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "Roles retrieved successfully",
		Data:    roleResponses,
	})
}

// GetRole handles retrieving a specific role
// @Summary Get a role by ID
// @Description Retrieve a specific role by its ID
// @Tags roles
// @Produce json
// @Param id path string true "Role ID"
// @Success 200 {object} dto.RoleResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/roles/{id} [get]
func (h *RoleHandler) GetRole(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid role ID",
			Details: "Role ID must be a valid UUID",
		})
		return
	}

	role, err := h.roleRepo.GetRoleByID(c, roleID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Role not found",
			Details: "The requested role does not exist",
		})
		return
	}

	permissions, _ := h.roleRepo.GetRolePermissions(c, role.ID)

	c.JSON(http.StatusOK, dto.RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		Permissions: permissions,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	})
}

// UpdateRole handles role updates
// @Summary Update a role
// @Description Update an existing role's information and permissions
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Param role body dto.UpdateRoleRequest true "Role update data"
// @Success 200 {object} dto.RoleResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/roles/{id} [put]
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid role ID",
			Details: "Role ID must be a valid UUID",
		})
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Get existing role
	role, err := h.roleRepo.GetRoleByID(c, roleID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Role not found",
			Details: "The requested role does not exist",
		})
		return
	}

	// Update fields if provided
	if req.Name != "" {
		// Check if name already exists (for different role)
		existingRole, err := h.roleRepo.GetRoleByName(c, req.Name)
		if err == nil && existingRole.ID != roleID {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Error:   "Role name already exists",
				Details: "A role with this name already exists",
			})
			return
		}
		role.Name = req.Name
	}

	if req.Description != "" {
		role.Description = req.Description
	}

	
	role.UpdatedAt = time.Now()

	// Update role in database
	if err := h.roleRepo.UpdateRole(c, role); err != nil {
		h.logger.Error().Err(err).Str("role_id", roleID.String()).Msg("Failed to update role")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to update role",
			Details: "Internal server error",
		})
		return
	}

	// Update permissions if provided
	if req.Permissions != nil {
		// Get current permissions
		currentPermissions, err := h.roleRepo.GetRolePermissions(c, roleID)
		if err != nil {
			h.logger.Error().Err(err).Str("role_id", roleID.String()).Msg("Failed to get current permissions")
		} else {
			// Get current user ID for audit logging
			currentUserID, _ := auth.GetCurrentUserID(c)
			
			// Remove permissions that are no longer needed
			for _, perm := range currentPermissions {
				if !contains(req.Permissions, perm) {
					if err := h.roleRepo.RemovePermissionFromRole(c, roleID, perm); err != nil {
						h.logger.Error().Err(err).
							Str("role_id", roleID.String()).
							Str("permission", perm).
							Msg("Failed to remove permission from role")
					} else {
						// Log permission change to audit log
						if h.auditLogger != nil {
							auditEvent := &audit.AuditEvent{
								EventType:  audit.EventTypePermissionChange,
								UserID:     &currentUserID,
								ResourceID: roleID.String(),
								Action:     "remove_permission",
								IPAddress:  c.ClientIP(),
								Success:    true,
								Details: map[string]interface{}{
									"role_name":  role.Name,
									"permission": perm,
								},
							}
							if auditErr := h.auditLogger.LogEvent(c.Request.Context(), auditEvent); auditErr != nil {
								h.logger.Error().Err(auditErr).Msg("Failed to log audit event for permission removal")
							}
						}
					}
				}
			}

			// Add new permissions
			for _, perm := range req.Permissions {
				if !contains(currentPermissions, perm) {
					if err := h.roleRepo.AddPermissionToRole(c, roleID, perm); err != nil {
						h.logger.Error().Err(err).
							Str("role_id", roleID.String()).
							Str("permission", perm).
							Msg("Failed to add permission to role")
					} else {
						// Log permission change to audit log
						if h.auditLogger != nil {
							auditEvent := &audit.AuditEvent{
								EventType:  audit.EventTypePermissionChange,
								UserID:     &currentUserID,
								ResourceID: roleID.String(),
								Action:     "add_permission",
								IPAddress:  c.ClientIP(),
								Success:    true,
								Details: map[string]interface{}{
									"role_name":  role.Name,
									"permission": perm,
								},
							}
							if auditErr := h.auditLogger.LogEvent(c.Request.Context(), auditEvent); auditErr != nil {
								h.logger.Error().Err(auditErr).Msg("Failed to log audit event for permission addition")
							}
						}
					}
				}
			}
		}
	}

	// Get updated permissions
	permissions, _ := h.roleRepo.GetRolePermissions(c, role.ID)

	c.JSON(http.StatusOK, dto.RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		Permissions: permissions,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	})
}

// DeleteRole handles role deletion
// @Summary Delete a role
// @Description Delete a role (soft delete by setting is_active to false)
// @Tags roles
// @Produce json
// @Param id path string true "Role ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/roles/{id} [delete]
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid role ID",
			Details: "Role ID must be a valid UUID",
		})
		return
	}

	// Check if role exists
	_, err = h.roleRepo.GetRoleByID(c, roleID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Role not found",
			Details: "The requested role does not exist",
		})
		return
	}

	// Delete role
	if err := h.roleRepo.DeleteRole(c, roleID); err != nil {
		h.logger.Error().Err(err).Str("role_id", roleID.String()).Msg("Failed to delete role")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to delete role",
			Details: "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "Role deleted successfully",
	})
}

// AssignRoleToUser handles role assignment to users
// @Summary Assign role to user
// @Description Assign a role to a user
// @Tags roles
// @Accept json
// @Produce json
// @Param assignment body dto.AssignRoleRequest true "Role assignment data"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/roles/assign [post]
func (h *RoleHandler) AssignRoleToUser(c *gin.Context) {
	var req dto.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Get current user ID from context
	currentUserID, exists := auth.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "User not authenticated",
			Details: "Authentication required",
		})
		return
	}

	// Check if role exists
	_, err := h.roleRepo.GetRoleByID(c, req.RoleID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Role not found",
			Details: "The requested role does not exist",
		})
		return
	}

	// Get role name for audit logging
	role, _ := h.roleRepo.GetRoleByID(c, req.RoleID)
	roleName := ""
	if role != nil {
		roleName = role.Name
	}

	// Assign role to user
	if err := h.roleRepo.AssignRoleToUser(c, req.UserID, req.RoleID, currentUserID); err != nil {
		h.logger.Error().Err(err).
			Str("user_id", req.UserID.String()).
			Str("role_id", req.RoleID.String()).
			Msg("Failed to assign role to user")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to assign role",
			Details: "Internal server error",
		})
		return
	}

	// Log role assignment to audit log
	if h.auditLogger != nil {
		auditEvent := audit.NewRoleAssignmentEvent(currentUserID, req.UserID, roleName, c.ClientIP())
		if auditErr := h.auditLogger.LogEvent(c.Request.Context(), auditEvent); auditErr != nil {
			h.logger.Error().Err(auditErr).Msg("Failed to log audit event for role assignment")
		}
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "Role assigned successfully",
	})
}

// RemoveRoleFromUser handles role removal from users
// @Summary Remove role from user
// @Description Remove a role from a user
// @Tags roles
// @Accept json
// @Produce json
// @Param assignment body dto.RemoveRoleRequest true "Role removal data"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/roles/remove [post]
func (h *RoleHandler) RemoveRoleFromUser(c *gin.Context) {
	var req dto.RemoveRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Check if role exists and get role name for audit logging
	role, err := h.roleRepo.GetRoleByID(c, req.RoleID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Role not found",
			Details: "The requested role does not exist",
		})
		return
	}

	// Get current user ID from context for audit logging
	currentUserID, _ := auth.GetCurrentUserID(c)

	// Remove role from user
	if err := h.roleRepo.RemoveRoleFromUser(c, req.UserID, req.RoleID); err != nil {
		h.logger.Error().Err(err).
			Str("user_id", req.UserID.String()).
			Str("role_id", req.RoleID.String()).
			Msg("Failed to remove role from user")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to remove role",
			Details: "Internal server error",
		})
		return
	}

	// Log role revocation to audit log
	if h.auditLogger != nil {
		auditEvent := audit.NewRoleRevocationEvent(currentUserID, req.UserID, role.Name, c.ClientIP())
		if auditErr := h.auditLogger.LogEvent(c.Request.Context(), auditEvent); auditErr != nil {
			h.logger.Error().Err(auditErr).Msg("Failed to log audit event for role revocation")
		}
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "Role removed successfully",
	})
}

// GetUserRoles handles retrieving user roles
// @Summary Get user roles
// @Description Retrieve all roles assigned to a user
// @Tags roles
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/roles/user/{userId} [get]
func (h *RoleHandler) GetUserRoles(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid user ID",
			Details: "User ID must be a valid UUID",
		})
		return
	}

	roles, err := h.roleRepo.GetUserRoles(c, userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to get user roles")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to retrieve user roles",
			Details: "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "User roles retrieved successfully",
		Data:    roles,
	})
}

// GetUserPermissions handles retrieving user permissions
// @Summary Get user permissions
// @Description Retrieve all permissions for a user based on their roles
// @Tags roles
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} dto.UserPermissionsResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/roles/user/{userId}/permissions [get]
func (h *RoleHandler) GetUserPermissions(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid user ID",
			Details: "User ID must be a valid UUID",
		})
		return
	}

	permissions, err := h.roleRepo.GetUserPermissions(c, userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to get user permissions")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to retrieve user permissions",
			Details: "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, dto.UserPermissionsResponse{
		UserID:      userID,
		Permissions: permissions,
	})
}

// GetRolePermissions handles retrieving role permissions
// @Summary Get role permissions
// @Description Retrieve all permissions for a specific role
// @Tags roles
// @Produce json
// @Param id path string true "Role ID"
// @Success 200 {object} dto.RolePermissionResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/roles/{id}/permissions [get]
func (h *RoleHandler) GetRolePermissions(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid role ID",
			Details: "Role ID must be a valid UUID",
		})
		return
	}

	// Get role
	role, err := h.roleRepo.GetRoleByID(c, roleID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Role not found",
			Details: "The requested role does not exist",
		})
		return
	}

	// Get permissions
	permissions, err := h.roleRepo.GetRolePermissions(c, roleID)
	if err != nil {
		h.logger.Error().Err(err).Str("role_id", roleID.String()).Msg("Failed to get role permissions")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to retrieve role permissions",
			Details: "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, dto.RolePermissionResponse{
		RoleID:      roleID,
		RoleName:    role.Name,
		Permissions: permissions,
	})
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}