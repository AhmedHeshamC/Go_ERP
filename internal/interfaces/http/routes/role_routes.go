package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/internal/domain/users/repositories"
	"erpgo/internal/interfaces/http/handlers"
	"erpgo/pkg/auth"
)

// RoleRoutes configures role management routes
type RoleRoutes struct {
	roleHandler *handlers.RoleHandler
	logger      zerolog.Logger
}

// NewRoleRoutes creates new role routes
func NewRoleRoutes(roleRepo repositories.RoleRepository, logger zerolog.Logger) *RoleRoutes {
	return &RoleRoutes{
		roleHandler: handlers.NewRoleHandler(roleRepo, logger),
		logger:      logger,
	}
}

// SetupRoleRoutes configures all role-related routes
func (r *RoleRoutes) SetupRoleRoutes(router *gin.RouterGroup, roleRepo repositories.RoleRepository, jwtService *auth.JWTService) {
	// Role management routes
	roles := router.Group("/roles")
	{
		// Public/Protected routes - require authentication
		roles.Use(auth.AuthMiddleware(jwtService))

		// Role CRUD operations
		roles.POST("",
			auth.RequirePermission(roleRepo, "role.create"),
			r.roleHandler.CreateRole,
		)
		roles.GET("",
			auth.RequirePermission(roleRepo, "role.read"),
			r.roleHandler.GetRoles,
		)
		roles.GET("/:id",
			auth.RequirePermission(roleRepo, "role.read"),
			r.roleHandler.GetRole,
		)
		roles.PUT("/:id",
			auth.RequirePermission(roleRepo, "role.update"),
			r.roleHandler.UpdateRole,
		)
		roles.DELETE("/:id",
			auth.RequirePermission(roleRepo, "role.delete"),
			r.roleHandler.DeleteRole,
		)

		// Role permission management
		roles.GET("/:id/permissions",
			auth.RequirePermission(roleRepo, "role.read"),
			r.roleHandler.GetRolePermissions,
		)

		// User role assignment
		roles.POST("/assign",
			auth.RequirePermission(roleRepo, "role.assign"),
			r.roleHandler.AssignRoleToUser,
		)
		roles.POST("/remove",
			auth.RequirePermission(roleRepo, "role.assign"),
			r.roleHandler.RemoveRoleFromUser,
		)

		// User role and permission queries
		roles.GET("/user/:userId",
			auth.RequirePermission(roleRepo, "role.read"),
			r.roleHandler.GetUserRoles,
		)
		roles.GET("/user/:userId/permissions",
			auth.RequirePermission(roleRepo, "role.read"),
			r.roleHandler.GetUserPermissions,
		)
	}
}