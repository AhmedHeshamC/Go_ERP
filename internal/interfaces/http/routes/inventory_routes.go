package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/internal/interfaces/http/handlers"
	"erpgo/internal/interfaces/http/middleware"
)

// SetupInventoryRoutes configures all inventory-related routes
func SetupInventoryRoutes(
	router *gin.RouterGroup,
	warehouseHandler *handlers.WarehouseHandler,
	inventoryHandler *handlers.InventoryHandler,
	transactionHandler *handlers.InventoryTransactionHandler,
	authMiddleware gin.HandlerFunc,
	adminAuthMiddleware gin.HandlerFunc,
	validationMiddleware gin.HandlerFunc,
	logger zerolog.Logger,
) {
	// Warehouse routes (require authentication)
	warehouseGroup := router.Group("/warehouses")
	warehouseGroup.Use(authMiddleware)
	warehouseGroup.Use(middleware.Logger(logger))
	{
		// Warehouse CRUD operations
		warehouseGroup.POST("", warehouseHandler.CreateWarehouse)
		warehouseGroup.GET("", warehouseHandler.ListWarehouses)
		warehouseGroup.GET("/search", warehouseHandler.SearchWarehouses)
		warehouseGroup.GET("/:id", warehouseHandler.GetWarehouse)
		warehouseGroup.GET("/code/:code", warehouseHandler.GetWarehouseByCode)
		warehouseGroup.PUT("/:id", warehouseHandler.UpdateWarehouse)
		warehouseGroup.DELETE("/:id", warehouseHandler.DeleteWarehouse)

		// Warehouse operations
		warehouseGroup.POST("/:id/activate", warehouseHandler.ActivateWarehouse)
		warehouseGroup.POST("/:id/deactivate", warehouseHandler.DeactivateWarehouse)
		warehouseGroup.PUT("/:id/manager", warehouseHandler.AssignManager)
		warehouseGroup.DELETE("/:id/manager", warehouseHandler.RemoveManager)

		// Warehouse statistics
		warehouseGroup.GET("/stats", warehouseHandler.GetWarehouseStats)
	}

	// Inventory routes (require authentication)
	inventoryGroup := router.Group("/inventory")
	inventoryGroup.Use(authMiddleware)
	inventoryGroup.Use(middleware.Logger(logger))
	{
		// Inventory stock management
		inventoryGroup.POST("/adjust", inventoryHandler.AdjustInventory)
		inventoryGroup.POST("/reserve", inventoryHandler.ReserveInventory)
		inventoryGroup.POST("/release", inventoryHandler.ReleaseInventory)
		inventoryGroup.POST("/transfer", inventoryHandler.TransferInventory)

		// Inventory query operations
		inventoryGroup.GET("", inventoryHandler.ListInventory)
		inventoryGroup.GET("/product/:product_id/warehouse/:warehouse_id", inventoryHandler.GetInventory)
		inventoryGroup.GET("/product/:product_id/warehouse/:warehouse_id/check-availability", inventoryHandler.CheckInventoryAvailability)
		inventoryGroup.GET("/product/:product_id/warehouse/:warehouse_id/history", inventoryHandler.GetInventoryHistory)

		// Inventory statistics and reports
		inventoryGroup.GET("/stats", inventoryHandler.GetInventoryStats)
		inventoryGroup.GET("/low-stock", inventoryHandler.GetLowStockItems)

		// Bulk operations
		inventoryGroup.POST("/bulk-adjust", inventoryHandler.BulkInventoryAdjustment)
	}

	// Inventory transaction routes (require authentication)
	transactionGroup := router.Group("/inventory/transactions")
	transactionGroup.Use(authMiddleware)
	transactionGroup.Use(middleware.Logger(logger))
	{
		// Transaction CRUD operations
		transactionGroup.GET("", transactionHandler.ListTransactions)
		transactionGroup.GET("/:id", transactionHandler.GetTransaction)
		transactionGroup.GET("/search", transactionHandler.SearchTransactions)

		// Transaction management operations
		transactionGroup.POST("/:id/approve", transactionHandler.ApproveTransaction)
		transactionGroup.POST("/:id/reject", transactionHandler.RejectTransaction)
		transactionGroup.GET("/pending", transactionHandler.GetPendingApprovals)

		// Transaction statistics
		transactionGroup.GET("/stats", transactionHandler.GetTransactionStats)
	}

	// Low stock alert routes (require authentication)
	alertGroup := router.Group("/inventory/alerts/low-stock")
	alertGroup.Use(authMiddleware)
	alertGroup.Use(middleware.Logger(logger))
	{
		// Alert CRUD operations
		alertGroup.POST("", transactionHandler.CreateLowStockAlert)
		alertGroup.GET("", transactionHandler.ListLowStockAlerts)
		alertGroup.GET("/warehouse/:warehouse_id", transactionHandler.GetLowStockAlertsByWarehouse)
		alertGroup.PUT("/:id", transactionHandler.UpdateLowStockAlert)
		alertGroup.DELETE("/:id", transactionHandler.DeleteLowStockAlert)
	}

	// Admin inventory routes (require admin role)
	adminGroup := router.Group("/admin/inventory")
	adminGroup.Use(authMiddleware)
	adminGroup.Use(middleware.RequireRole("admin"))
	adminGroup.Use(middleware.Logger(logger))
	{
		// Admin warehouse operations
		adminGroup.POST("/warehouses/bulk", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Bulk warehouse operations not yet implemented"})
		})
		adminGroup.POST("/warehouses/import", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Warehouse import not yet implemented"})
		})
		adminGroup.GET("/warehouses/export", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Warehouse export not yet implemented"})
		})

		// Admin inventory operations
		adminGroup.POST("/bulk-operations", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Bulk inventory operations not yet implemented"})
		})
		adminGroup.POST("/import", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Inventory import not yet implemented"})
		})
		adminGroup.GET("/export", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Inventory export not yet implemented"})
		})
		adminGroup.GET("/reports", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Inventory reports not yet implemented"})
		})

		// Admin transaction operations
		adminGroup.POST("/transactions/bulk-approve", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Bulk transaction approval not yet implemented"})
		})
		adminGroup.POST("/transactions/bulk-reject", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Bulk transaction rejection not yet implemented"})
		})
		adminGroup.POST("/transactions/rollback/:id", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Transaction rollback not yet implemented"})
		})

		// Admin alert operations
		adminGroup.POST("/alerts/bulk", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Bulk alert operations not yet implemented"})
		})
		adminGroup.POST("/alerts/test/:id", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Alert testing not yet implemented"})
		})
		adminGroup.POST("/alerts/send-digest", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Alert digest sending not yet implemented"})
		})
	}

	// Public inventory routes (no authentication required - read-only)
	publicGroup := router.Group("/public/inventory")
	publicGroup.Use(middleware.Logger(logger))
	{
		// Public warehouse information (limited)
		publicGroup.GET("/warehouses", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Public warehouse listing not yet implemented"})
		})
		publicGroup.GET("/warehouses/:id", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Public warehouse details not yet implemented"})
		})

		// Public inventory availability (limited)
		publicGroup.GET("/availability/product/:product_id", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Public product availability not yet implemented"})
		})
	}

	// Webhook routes for external integrations
	webhookGroup := router.Group("/webhooks/inventory")
	webhookGroup.Use(middleware.Auth(nil)) // TODO: Replace with proper webhook auth
	webhookGroup.Use(middleware.Logger(logger))
	{
		// Warehouse webhooks
		webhookGroup.POST("/warehouse/created", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Warehouse created webhook not yet implemented"})
		})
		webhookGroup.POST("/warehouse/updated", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Warehouse updated webhook not yet implemented"})
		})

		// Inventory webhooks
		webhookGroup.POST("/inventory/updated", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Inventory updated webhook not yet implemented"})
		})
		webhookGroup.POST("/inventory/low-stock", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Low stock webhook not yet implemented"})
		})
		webhookGroup.POST("/inventory/out-of-stock", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Out of stock webhook not yet implemented"})
		})

		// Transaction webhooks
		webhookGroup.POST("/transaction/created", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Transaction created webhook not yet implemented"})
		})
		webhookGroup.POST("/transaction/approved", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Transaction approved webhook not yet implemented"})
		})
		webhookGroup.POST("/transaction/rejected", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Transaction rejected webhook not yet implemented"})
		})
	}
}