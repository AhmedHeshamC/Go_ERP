package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"erpgo/internal/interfaces/http/handlers"
	"erpgo/internal/interfaces/http/middleware"
)

// SetupImageRoutes configures all image-related routes
func SetupImageRoutes(
	router *gin.RouterGroup,
	imageHandler *handlers.ImageHandler,
	authMiddleware gin.HandlerFunc,
	validationMiddleware gin.HandlerFunc,
	logger zerolog.Logger,
) {
	// Product image routes (require authentication)
	productImageGroup := router.Group("/products/:product_id/images")
	productImageGroup.Use(authMiddleware)
	productImageGroup.Use(middleware.Logger(logger))
	{
		productImageGroup.POST("", imageHandler.UploadProductImage)
		productImageGroup.POST("/batch", imageHandler.UploadMultipleImages)
		productImageGroup.GET("", imageHandler.GetProductImages)
	}

	// Product variant image routes (require authentication)
	variantImageGroup := router.Group("/products/:product_id/variants/:variant_id/images")
	variantImageGroup.Use(authMiddleware)
	variantImageGroup.Use(middleware.Logger(logger))
	{
		variantImageGroup.POST("", imageHandler.UploadProductVariantImage)
	}

	// General image management routes (require authentication)
	imageGroup := router.Group("/images")
	imageGroup.Use(authMiddleware)
	imageGroup.Use(middleware.Logger(logger))
	{
		imageGroup.GET("/:filename", imageHandler.GetImage)
		imageGroup.DELETE("/:filename", imageHandler.DeleteImage)
	}

	// Public image routes (no authentication required for viewing)
	publicImageGroup := router.Group("/public/images")
	publicImageGroup.Use(middleware.Logger(logger))
	{
		publicImageGroup.GET("/:filename", imageHandler.GetImage)
	}

	// Admin image management routes (require admin role)
	adminImageGroup := router.Group("/admin/images")
	adminImageGroup.Use(authMiddleware)
	adminImageGroup.Use(middleware.RequireRole("admin"))
	adminImageGroup.Use(middleware.Logger(logger))
	{
		// Bulk operations
		adminImageGroup.POST("/import", imageHandler.ImportImages)
		adminImageGroup.GET("/export", imageHandler.ExportImages)
		adminImageGroup.POST("/cleanup", imageHandler.CleanupImages)
		adminImageGroup.GET("/stats", imageHandler.GetImageStats)
		adminImageGroup.GET("/storage", imageHandler.GetStorageInfo)

		// CDN management
		adminImageGroup.POST("/cdn/sync", imageHandler.SyncCDN)
		adminImageGroup.GET("/cdn/config", imageHandler.GetCDNConfig)
		adminImageGroup.PUT("/cdn/config", imageHandler.UpdateCDNConfig)

		// Image processing
		adminImageGroup.POST("/process", imageHandler.ProcessImages)
		adminImageGroup.POST("/optimize", imageHandler.OptimizeImages)
	}
}