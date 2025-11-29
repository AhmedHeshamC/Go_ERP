package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/internal/interfaces/http/handlers"
	"erpgo/internal/interfaces/http/middleware"
	"erpgo/pkg/config"
)

// SetupRoutes configures all application routes
func SetupRoutes(
	router *gin.Engine,
	authHandler *handlers.AuthHandler,
	productHandler *handlers.ProductHandler,
	inventoryHandler *handlers.InventoryHandler,
	warehouseHandler *handlers.WarehouseHandler,
	transactionHandler *handlers.InventoryTransactionHandler,
	cfg *config.Config,
	logger zerolog.Logger,
) {
	// Setup middlewares
	authMiddleware := middleware.Auth(nil)               // TODO: Pass actual JWT service
	validationMiddleware := middleware.SecurityHeaders() // Use available middleware
	// API v1 group
	v1 := router.Group("/api/v1")

	// Setup individual route groups
	SetupUserRoutes(v1, authHandler)
	SetupProductRoutes(v1, productHandler)
	// TODO: Implement OrderHandler
	// SetupOrderRoutes(v1, orderHandler)
	SetupInventoryRoutes(v1, warehouseHandler, inventoryHandler, transactionHandler, authMiddleware, authMiddleware, validationMiddleware, logger)

	// Root endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "ERPGo API Server",
			"version": "1.0.0",
			"docs":    "/api/v1/docs",
		})
	})
}
