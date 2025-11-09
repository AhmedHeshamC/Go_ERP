package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	userService "erpgo/internal/application/services/user"
	"erpgo/internal/domain/users/entities"
	"erpgo/internal/interfaces/http/dto"
	"erpgo/internal/interfaces/http/handlers"
	"erpgo/internal/interfaces/http/routes"
)

func TestSetupUserRoutes(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create a simple mock service that satisfies the interface
	mockService := &SimpleMockUserService{}
	logger := zerolog.Nop()

	router := gin.New()
	routes.SetupUserRoutes(router.Group("/api/v1"), mockService, logger)

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

func TestAuthHandler_Register(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	mockService := &SimpleMockUserService{}
	logger := zerolog.Nop()

	handler := handlers.NewAuthHandler(mockService, logger)
	router := gin.New()
	router.POST("/register", handler.Register)

	// Create request body
	reqBody := dto.RegisterRequest{
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "Password123!",
		FirstName: "Test",
		LastName:  "User",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response dto.UserInfo
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", response.Email)
	assert.Equal(t, "testuser", response.Username)
	assert.Equal(t, "Test", response.FirstName)
	assert.Equal(t, "User", response.LastName)
}

func TestAuthHandler_Login(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	mockService := &SimpleMockUserService{}
	logger := zerolog.Nop()

	handler := handlers.NewAuthHandler(mockService, logger)
	router := gin.New()
	router.POST("/login", handler.Login)

	// Create request body
	reqBody := dto.LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
	assert.NotNil(t, response.User)
	assert.Equal(t, "test@example.com", response.User.Email)
}

func TestUserHandler_GetUsers(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	mockService := &SimpleMockUserService{}
	logger := zerolog.Nop()

	handler := handlers.NewUserHandler(mockService, logger)
	router := gin.New()
	router.GET("/users", handler.GetUsers)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/users?search=test", nil)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.ListUsersResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response.Users, 2)
	assert.NotNil(t, response.Pagination)
	assert.Equal(t, 1, response.Pagination.Page)
	assert.Equal(t, 2, response.Pagination.Total)
}

// SimpleMockUserService provides a minimal mock implementation for testing
type SimpleMockUserService struct{}

func (m *SimpleMockUserService) CreateUser(ctx context.Context, req *userService.CreateUserRequest) (*entities.User, error) {
	return &entities.User{
		ID:        uuid.New(),
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		IsActive:  true,
		IsVerified: false,
	}, nil
}

func (m *SimpleMockUserService) GetUser(ctx context.Context, id string) (*entities.User, error) {
	return &entities.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
	}, nil
}

func (m *SimpleMockUserService) GetUserByEmail(ctx context.Context, email string) (*entities.User, error) {
	return &entities.User{
		ID:        uuid.New(),
		Email:     email,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
	}, nil
}

func (m *SimpleMockUserService) UpdateUser(ctx context.Context, id string, req *userService.UpdateUserRequest) (*entities.User, error) {
	return &entities.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Username:  "testuser",
		FirstName: "Updated",
		LastName:  "Name",
		IsActive:  true,
	}, nil
}

func (m *SimpleMockUserService) DeleteUser(ctx context.Context, id string) error {
	return nil
}

func (m *SimpleMockUserService) ListUsers(ctx context.Context, filter *userService.ListUsersRequest) (*userService.ListUsersResponse, error) {
	users := []*entities.User{
		{
			ID:        uuid.New(),
			Email:     "user1@example.com",
			Username:  "user1",
			FirstName: "User",
			LastName:  "One",
			IsActive:  true,
		},
		{
			ID:        uuid.New(),
			Email:     "user2@example.com",
			Username:  "user2",
			FirstName: "User",
			LastName:  "Two",
			IsActive:  true,
		},
	}

	return &userService.ListUsersResponse{
		Users: users,
		Pagination: &userService.Pagination{
			Page:       1,
			Limit:      20,
			Total:      2,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		},
	}, nil
}

func (m *SimpleMockUserService) Login(ctx context.Context, req *userService.LoginRequest) (*userService.LoginResponse, error) {
	user := &entities.User{
		ID:        uuid.New(),
		Email:     req.Email,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
		IsVerified: true,
	}

	return &userService.LoginResponse{
		User:         user,
		AccessToken:  "mock_access_token",
		RefreshToken: "mock_refresh_token",
		ExpiresIn:    3600,
	}, nil
}

func (m *SimpleMockUserService) Logout(ctx context.Context, token string) error {
	return nil
}

func (m *SimpleMockUserService) RefreshToken(ctx context.Context, refreshToken string) (*userService.TokenResponse, error) {
	return &userService.TokenResponse{
		AccessToken:  "new_access_token",
		RefreshToken: "new_refresh_token",
		ExpiresIn:    3600,
	}, nil
}

func (m *SimpleMockUserService) ChangePassword(ctx context.Context, req *userService.ChangePasswordRequest) error {
	return nil
}

func (m *SimpleMockUserService) ForgotPassword(ctx context.Context, email string) error {
	return nil
}

func (m *SimpleMockUserService) ResetPassword(ctx context.Context, req *userService.ResetPasswordRequest) error {
	return nil
}

func (m *SimpleMockUserService) AssignRole(ctx context.Context, userID, roleID string) error {
	return nil
}

func (m *SimpleMockUserService) RemoveRole(ctx context.Context, userID, roleID string) error {
	return nil
}

func (m *SimpleMockUserService) GetUserRoles(ctx context.Context, userID string) ([]*entities.Role, error) {
	return []*entities.Role{}, nil
}

func (m *SimpleMockUserService) CreateRole(ctx context.Context, req *userService.CreateRoleRequest) (*entities.Role, error) {
	return &entities.Role{}, nil
}

func (m *SimpleMockUserService) UpdateRole(ctx context.Context, id string, req *userService.UpdateRoleRequest) (*entities.Role, error) {
	return &entities.Role{}, nil
}

func (m *SimpleMockUserService) DeleteRole(ctx context.Context, id string) error {
	return nil
}

func (m *SimpleMockUserService) ListRoles(ctx context.Context, filter *userService.ListRolesRequest) (*userService.ListRolesResponse, error) {
	return &userService.ListRolesResponse{}, nil
}