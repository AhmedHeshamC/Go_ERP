package routes

import (
	"github.com/gin-gonic/gin"

	"erpgo/internal/interfaces/http/handlers"
)

// SetupUserRoutes is a convenience function to setup all user-related routes
func SetupUserRoutes(
	router *gin.RouterGroup,
	authHandler *handlers.AuthHandler,
) {

	// Public routes (no authentication required)
	public := router.Group("/auth")
	{
		public.POST("/login", authHandler.Login)
		public.POST("/register", authHandler.Register)
		public.POST("/refresh", authHandler.RefreshToken)
		public.POST("/logout", authHandler.Logout)
	}
}