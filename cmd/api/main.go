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
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	_ "erpgo/docs" // Import generated docs

	"github.com/google/uuid"

	"erpgo/internal/domain/users/entities"
	"erpgo/internal/domain/users/repositories"
	"erpgo/internal/application/services/user"
	"erpgo/internal/application/services/product"
	"erpgo/internal/application/services/inventory"
	emailsvc "erpgo/internal/application/services/email"
	infrarepos "erpgo/internal/infrastructure/repositories"
	"erpgo/internal/interfaces/http/handlers"
	httpmiddleware "erpgo/internal/interfaces/http/middleware"
	"erpgo/internal/interfaces/http/routes"
	"erpgo/pkg/auth"
	"erpgo/pkg/cache"
	"erpgo/pkg/config"
	"erpgo/pkg/database"
	"erpgo/pkg/email"
	"erpgo/pkg/logger"
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
	log := logger.New(cfg.LogLevel, cfg.IsDevelopment())
	log.Info().
		Str("version", version).
		Str("build_time", buildTime).
		Str("commit", commit).
		Msg("Starting ERPGo API server")

	// Initialize database
	dbConfig := cfg.GetDatabaseConfig()
	db, err := database.NewWithLogger(database.Config{
		URL:             dbConfig.URL,
		MaxConnections:  dbConfig.MaxConnections,
		MinConnections:  dbConfig.MinConnections,
		ConnMaxLifetime: dbConfig.ConnMaxLifetime,
		ConnMaxIdleTime: dbConfig.ConnMaxIdleTime,
		SSLMode:         dbConfig.SSLMode,
		SSLCert:         dbConfig.SSLCert,
		SSLKey:          dbConfig.SSLKey,
		SSLCA:           dbConfig.SSLCA,
		SSLHost:         dbConfig.SSLHost,
	}, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer db.Close()

	// Initialize cache
	redisConfig := cfg.GetRedisConfig()
	cache, err := cache.NewWithLogger(cache.Config{
		URL:      redisConfig.URL,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
		PoolSize: redisConfig.PoolSize,
	}, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize cache")
	}
	defer cache.Close()

	// Initialize repositories with proper implementations
	userRepo := infrarepos.NewPostgresUserRepository(db)
	// TODO: Create proper repository implementations (Phase 1.1.1)
	// roleRepo := infrarepos.NewPostgresRoleRepository(db) // Doesn't exist yet
	// userRoleRepo := infrarepos.NewPostgresUserRoleRepository(db) // Doesn't exist yet
	var roleRepo repositories.RoleRepository = &MockRoleRepository{} // Temporary mock for Phase 1.1.1
	var userRoleRepo repositories.UserRoleRepository = &MockUserRoleRepository{} // Temporary mock for Phase 1.1.1

	// Initialize product repositories
	productRepo := infrarepos.NewPostgresProductRepository(db)
	categoryRepo := infrarepos.NewPostgresCategoryRepository(db)
	variantRepo := infrarepos.NewPostgresProductVariantRepository(db)
	variantAttrRepo := infrarepos.NewPostgresVariantAttributeRepository(db)
	variantImageRepo := infrarepos.NewPostgresVariantImageRepository(db)

	// TODO: Uncomment when order services are implemented
	// Initialize order repositories
	// orderRepo := infrarepos.NewPostgresOrderRepository(db)
	// orderItemRepo := infrarepos.NewPostgresOrderItemRepository(db)
	// customerRepo := infrarepos.NewPostgresCustomerRepository(db)
	// addressRepo := infrarepos.NewPostgresOrderAddressRepository(db)
	// companyRepo := infrarepos.NewPostgresCompanyRepository(db)
	// orderAnalyticsRepo := infrarepos.NewPostgresOrderAnalyticsRepository(db)

	// Initialize inventory repositories
	inventoryRepo := infrarepos.NewPostgresInventoryRepository(db)
	warehouseRepo := infrarepos.NewPostgresWarehouseRepository(db)
	transactionRepo := infrarepos.NewPostgresInventoryTransactionRepository(db)

	// Create default roles if they don't exist
	// TODO: Implement proper role repository and uncomment
	// if err := roleRepo.CreateDefaultRoles(context.Background()); err != nil {
	// 	logger.Fatal().Err(err).Msg("Failed to create default roles")
	// }

	// Initialize authentication services
	jwtConfig := cfg.GetJWTConfig()
	jwtSvc := auth.NewJWTService(jwtConfig.Secret, "erpgo-api", jwtConfig.Expiry, jwtConfig.RefreshExpiry)
	// TODO: Set Redis client for token blacklist when implemented
	// jwtSvc.SetRedisClient(cache.GetClient())
	passwordSvc := auth.NewPasswordService(12, "default-pepper") // TODO: Get from config

	// Initialize email service
	// TODO: Get email configuration from config file
	emailConfig := &entities.EmailConfig{
		SMTPHost:     "localhost", // TODO: Get from config
		SMTPPort:     587,         // TODO: Get from config
		SMTPUsername: "",          // TODO: Get from config
		SMTPPassword: "",          // TODO: Get from config
		FromEmail:    "noreply@erpgo.example.com", // TODO: Get from config
		FromName:     "ERPGo",                     // TODO: Get from config
		UseTLS:       true,                        // TODO: Get from config
	}

	// Initialize email verification repository
	emailVerificationRepo := infrarepos.NewPostgresEmailVerificationRepository(db)

	// Initialize SMTP service
	smtpSvc := email.NewSMTPService(emailConfig)

	// Initialize email service
	emailSvc := emailsvc.NewService(userRepo, emailVerificationRepo, smtpSvc)

	// Initialize services
	// TODO: Implement SimpleAuthService
	// simpleAuthService := services.NewSimpleAuthService(userRepo, cfg, cache)
	userService := user.NewUserService(userRepo, roleRepo, userRoleRepo, passwordSvc, jwtSvc, emailSvc, cache)

	// Initialize product service
	productService := product.NewService(productRepo, categoryRepo, variantRepo, variantAttrRepo, variantImageRepo)

	// Initialize inventory service
	inventoryService := inventory.NewService(inventoryRepo, warehouseRepo, transactionRepo, log)

	// Initialize order service (with some dependencies still nil)
	// TODO: Implement notification, payment, tax, and shipping services
	// TODO: Fix order service compilation issues
	/*
	orderService := order.NewService(
		orderRepo,
		orderItemRepo,
		customerRepo,
		addressRepo,
		companyRepo,
		orderAnalyticsRepo,
		nil, // productService - will need adapter or direct repo access
		inventoryService, // inventoryService - now available
		nil, // userService - will need adapter or direct repo access
		nil, // notificationService
		nil, // paymentService
		nil, // taxCalculator
		nil, // shippingCalculator
	)
	*/

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userService, *log)
	productHandler := handlers.NewProductHandler(productService, *log)
	// orderHandler := handlers.NewOrderHandler(orderService, *log) // TODO: Fix order service
	inventoryHandler := handlers.NewInventoryHandler(inventoryService, *log)
	warehouseHandler := handlers.NewWarehouseHandler(nil, *log) // TODO: Create warehouseService
	transactionHandler := handlers.NewInventoryTransactionHandler(nil, *log) // TODO: Create transactionService

	// Setup Gin
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add comprehensive security middleware
	securityMiddleware, securityCoordinator, err := httpmiddleware.SecurityMiddleware(cfg.Environment, cache, *log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize security middleware")
	}
	defer securityCoordinator.Stop()

	// Add basic middleware
	router.Use(httpmiddleware.Logger(*log))
	router.Use(httpmiddleware.Recovery(*log))
	router.Use(httpmiddleware.RequestID())

	// Apply comprehensive security middleware
	router.Use(securityMiddleware)

	// Setup routes
	routes.SetupRoutes(router, authHandler, productHandler, inventoryHandler, warehouseHandler, transactionHandler, cfg, *log)

	// Setup Swagger documentation route
	router.GET("/api/v1/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Security dashboard endpoint (only in development/staging)
	if !cfg.IsProduction() {
		router.GET("/security/dashboard", func(c *gin.Context) {
			stats := securityCoordinator.GetSecurityStats()
			c.JSON(http.StatusOK, gin.H{
				"security_stats": stats,
				"timestamp":      time.Now().UTC(),
				"version":        version,
			})
		})
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		// Check database health
		healthChecker := database.NewHealthChecker(db, log)
		dbHealth := healthChecker.Check(c.Request.Context())

		// Check cache health
		cacheHealthy := true
		var cacheError error
		if err := cache.Ping(c.Request.Context()); err != nil {
			cacheHealthy = false
			cacheError = err
		}

		status := "healthy"
		if dbHealth.Status != "healthy" || !cacheHealthy {
			status = "unhealthy"
		}

		response := gin.H{
			"status":    status,
			"timestamp": time.Now().UTC(),
			"version":   version,
			"checks": gin.H{
				"database": dbHealth,
				"cache": gin.H{
					"status": func() string {
						if cacheHealthy {
							return "healthy"
						}
						return "unhealthy"
					}(),
					"message": func() string {
						if cacheError != nil {
							return cacheError.Error()
						}
						return "Cache connection successful"
					}(),
				},
			},
		}

		if status == "healthy" {
			c.JSON(http.StatusOK, response)
		} else {
			c.JSON(http.StatusServiceUnavailable, response)
		}
	})

	// Metrics endpoint
	if cfg.MetricsEnabled {
		router.GET(cfg.MetricsPath, gin.WrapH(promhttp.Handler()))
	}

	// TODO: API documentation - SetupSwagger function needs to be implemented in routes package
	// if cfg.APIDocsEnabled {
	// 	routes.SetupSwagger(router, cfg)
	// }

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
		log.Info().Int("port", cfg.ServerPort).Msg("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}

// Temporary mock implementations for Phase 1.1.1 - Repository Interface Mismatch
// TODO: Replace with proper PostgresRoleRepository implementation

type MockRoleRepository struct{}

func (m *MockRoleRepository) CreateRole(ctx context.Context, role *entities.Role) error {
	return fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (*entities.Role, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) GetRoleByName(ctx context.Context, name string) (*entities.Role, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) GetAllRoles(ctx context.Context) ([]*entities.Role, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) UpdateRole(ctx context.Context, role *entities.Role) error {
	return fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) RoleExists(ctx context.Context, name string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID, assignedBy uuid.UUID) error {
	return fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*entities.Role, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) GetUsersWithRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) HasUserRole(ctx context.Context, userID, roleID uuid.UUID) (bool, error) {
	return false, fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) RemoveAllUserRoles(ctx context.Context, userID uuid.UUID) error {
	return fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) UserHasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) UserHasAnyPermission(ctx context.Context, userID uuid.UUID, permissions ...string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) UserHasAllPermissions(ctx context.Context, userID uuid.UUID, permissions ...string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) AddPermissionToRole(ctx context.Context, roleID uuid.UUID, permission string) error {
	return fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) RemovePermissionFromRole(ctx context.Context, roleID uuid.UUID, permission string) error {
	return fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) CreateDefaultRoles(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}
func (m *MockRoleRepository) GetRoleAssignmentHistory(ctx context.Context, userID uuid.UUID) ([]*entities.UserRole, error) {
	return nil, fmt.Errorf("not implemented")
}

type MockUserRoleRepository struct{}

func (m *MockUserRoleRepository) AssignRole(ctx context.Context, userID, roleID, assignedBy string) error {
	return fmt.Errorf("not implemented")
}
func (m *MockUserRoleRepository) RemoveRole(ctx context.Context, userID, roleID string) error {
	return fmt.Errorf("not implemented")
}
func (m *MockUserRoleRepository) GetUserRoles(ctx context.Context, userID string) ([]*entities.Role, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockUserRoleRepository) GetUsersByRole(ctx context.Context, roleID string) ([]*entities.User, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockUserRoleRepository) HasRole(ctx context.Context, userID, roleID string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}