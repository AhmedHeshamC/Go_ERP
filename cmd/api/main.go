package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/erpgo/erpgo/internal/domain/users/repositories"
	"github.com/erpgo/erpgo/internal/domain/users/services"
	"github.com/erpgo/erpgo/internal/interfaces/http/handlers"
	"github.com/erpgo/erpgo/internal/interfaces/http/middleware"
	"github.com/erpgo/erpgo/internal/interfaces/http/routes"
	"github.com/erpgo/erpgo/pkg/cache"
	"github.com/erpgo/erpgo/pkg/config"
	"github.com/erpgo/erpgo/pkg/database"
	"github.com/erpgo/erpgo/pkg/logger"
)

// Build information injected at build time
var (
	version   = "dev"
	buildTime = "unknown"
	commit    = "unknown"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := logger.New(cfg.LogLevel, cfg.IsDevelopment())
	logger.Info().
		Str("version", version).
		Str("build_time", buildTime).
		Str("commit", commit).
		Msg("Starting ERPGo API server")

	// Initialize database
	db, err := database.New(cfg.GetDatabaseConfig())
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer db.Close()

	// Initialize cache
	cache, err := cache.New(cfg.GetRedisConfig())
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize cache")
	}
	defer cache.Close()

	// Initialize repositories
	userRepo := repositories.NewPostgresUserRepository(db)
	roleRepo := repositories.NewPostgresRoleRepository(db)
	userRoleRepo := repositories.NewPostgresUserRoleRepository(db)

	// Initialize services
	userService := services.NewUserService(userRepo, roleRepo, userRoleRepo, cache, logger)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(userService, cfg.GetJWTConfig())

	// Setup Gin
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.CORS(cfg.CORSOrigins, cfg.CORSMethods, cfg.CORSHeaders))
	router.Use(middleware.RequestID())

	if cfg.RateLimitEnabled {
		router.Use(middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst, cache))
	}

	// Setup routes
	routes.SetupRoutes(router, userHandler, cfg)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"version":   version,
		})
	})

	// Metrics endpoint
	if cfg.MetricsEnabled {
		router.GET(cfg.MetricsPath, gin.WrapH(promhttp.Handler()))
	}

	// API documentation
	if cfg.APIDocsEnabled {
		routes.SetupSwagger(router, cfg)
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info().Int("port", cfg.ServerPort).Msg("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
	}

	logger.Info().Msg("Server exited")
}