package http

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"erpgo/internal/interfaces/http/routes"
)

func TestSetupUserRoutes(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Test that we can create a router group (basic smoke test)
	router := gin.New()
	routeGroup := router.Group("/api/v1")
	assert.NotNil(t, routeGroup)
}

func TestUserRoutes_Registration(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Test that user routes can be set up
	router := gin.New()
	routes.SetupUserRoutes(router.Group("/api/v1"))

	// Test that routes are registered (basic smoke test)
	routeList := router.Routes()

	// Find auth routes
	expectedRoutes := []string{
		"POST /api/v1/auth/login",
		"POST /api/v1/auth/register",
		"POST /api/v1/auth/refresh",
		"POST /api/v1/auth/forgot-password",
		"POST /api/v1/auth/reset-password",
		"POST /api/v1/auth/logout",
	}

	foundRoutes := make(map[string]bool)
	for _, route := range routeList {
		routePath := route.Method + " " + route.Path
		foundRoutes[routePath] = true
	}

	for _, expectedRoute := range expectedRoutes {
		assert.True(t, foundRoutes[expectedRoute], "Route not found: "+expectedRoute)
	}

	// Find user routes
	userRoutes := []string{
		"GET /api/v1/users",
		"GET /api/v1/users/:id",
		"PUT /api/v1/users/:id",
		"DELETE /api/v1/users/:id",
		"GET /api/v1/users/:id/roles",
		"GET /api/v1/profile",
		"PUT /api/v1/profile",
	}

	for _, expectedRoute := range userRoutes {
		assert.True(t, foundRoutes[expectedRoute], "Route not found: "+expectedRoute)
	}
}
