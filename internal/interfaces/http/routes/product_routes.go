package routes

import (
	"github.com/gin-gonic/gin"

	"erpgo/internal/interfaces/http/handlers"
)

// SetupProductRoutes configures all product-related routes
func SetupProductRoutes(
	router *gin.RouterGroup,
	productHandler *handlers.ProductHandler,
) {
	// Product routes (require authentication)
	productGroup := router.Group("/products")
	{
		// Product CRUD operations
		productGroup.POST("", productHandler.CreateProduct)
		productGroup.GET("", productHandler.ListProducts)
		productGroup.GET("/search", productHandler.SearchProducts)
		productGroup.GET("/:id", productHandler.GetProduct)
		productGroup.GET("/sku/:sku", productHandler.GetProductBySKU)
		productGroup.PUT("/:id", productHandler.UpdateProduct)
		productGroup.DELETE("/:id", productHandler.DeleteProduct)

		// Product operations
		productGroup.POST("/:id/activate", productHandler.ActivateProduct)
		productGroup.POST("/:id/deactivate", productHandler.DeactivateProduct)
		productGroup.PUT("/:id/featured", productHandler.SetFeaturedProduct)
		productGroup.PUT("/:id/price", productHandler.UpdateProductPrice)
		productGroup.PUT("/:id/stock", productHandler.UpdateProductStock)
		productGroup.POST("/:id/stock/adjust", productHandler.AdjustProductStock)
		productGroup.GET("/:id/stock", productHandler.GetProductStockLevel)
		productGroup.POST("/:id/check-availability", productHandler.CheckProductAvailability)

		// Bulk operations
		productGroup.POST("/bulk", productHandler.BulkProductOperation)
		productGroup.POST("/import", productHandler.ImportProducts)
		productGroup.GET("/export", productHandler.ExportProducts)

		// Analytics and reporting
		productGroup.GET("/stats", productHandler.GetProductStats)
		productGroup.GET("/low-stock", productHandler.GetLowStockProducts)
		productGroup.POST("/inventory/adjust", productHandler.BulkInventoryAdjustment)
	}

	// Public product routes (no authentication required)
	publicGroup := router.Group("/public/products")
	{
		publicGroup.GET("", productHandler.ListProducts) // Read-only access
		publicGroup.GET("/search", productHandler.SearchProducts)
		publicGroup.GET("/:id", productHandler.GetProduct)
		publicGroup.GET("/sku/:sku", productHandler.GetProductBySKU)
	}

	// Admin product routes (require admin role)
	adminGroup := router.Group("/admin/products")
	{
		// Bulk operations
		adminGroup.POST("/bulk", productHandler.BulkProductOperation)
		adminGroup.POST("/import", productHandler.ImportProducts)
		adminGroup.GET("/export", productHandler.ExportProducts)
		adminGroup.GET("/stats", productHandler.GetProductStats)
		adminGroup.GET("/low-stock", productHandler.GetLowStockProducts)
		adminGroup.POST("/stock/bulk-adjust", productHandler.BulkInventoryAdjustment)
	}
}
