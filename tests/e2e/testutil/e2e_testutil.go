package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	userService "erpgo/internal/application/services/user"
	"erpgo/internal/domain/users/entities"
	"erpgo/internal/domain/users/repositories"
	"erpgo/internal/interfaces/http/handlers"
	"erpgo/internal/interfaces/http/routes"
	"erpgo/pkg/auth"
	"erpgo/pkg/cache"
	erpgoDatabase "erpgo/pkg/database"
)

// E2ETestSetup represents the complete E2E test setup
type E2ETestSetup struct {
	TestServer      *httptest.Server
	DB              *erpgoDatabase.Database
	Pool            *pgxpool.Pool
	UserRepo        repositories.UserRepository
	RoleRepo        repositories.RoleRepository
	UserRoleRepo    repositories.UserRoleRepository
	PasswordService *auth.PasswordService
	JWTService      *auth.JWTService
	UserService     userService.Service
	Cleanup         func()
}

// SetupE2ETest creates a complete E2E test environment
func SetupE2ETest(t *testing.T) *E2ETestSetup {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Get test database URL
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://erpgo_user:erpgo_password@localhost:5432/erpgo_test?sslmode=disable"
	}

	// Create database pool
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "Failed to create database pool")

	// Create database wrapper
	dbConfig := erpgoDatabase.Config{
		URL:             dbURL,
		MaxConnections:  10,
		MinConnections:  1,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
		SSLMode:         "disable",
	}
	db, err := erpgoDatabase.New(dbConfig)
	require.NoError(t, err, "Failed to create database wrapper")

	// Setup database schema
	setupE2EDatabaseSchema(t, pool)

	// Create repositories - Using mock implementations for now
	// In a real implementation, you would create proper repository implementations
	userRepo := &E2EMockUserRepository{pool: pool}
	// roleRepo := &E2EMockRoleRepository{pool: pool} // TODO: Implement full RoleRepository interface
	roleRepo := &MockRoleRepository{} // Simple mock for now
	userRoleRepo := &E2EMockUserRoleRepository{pool: pool}

	// Create services
	passwordService := auth.NewPasswordService(12, "test-pepper")
	jwtService := auth.NewJWTService("test-secret-key", "test-refresh-secret", 24*time.Hour, 7*24*time.Hour)

	// Create a mock cache for testing
	mockCache := cache.NewMockCache()

	userSvc := userService.NewService(userRepo, roleRepo, userRoleRepo, passwordService, jwtService, nil, mockCache, nil)

	// Create logger
	logger := zerolog.Nop()

	// Create Gin router
	router := gin.New()

	// Setup middleware
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Create auth handler
	authHandler := handlers.NewAuthHandler(userSvc, logger)

	// Setup routes
	routes.SetupUserRoutes(router.Group("/api/v1"), authHandler)

	// Create test server
	testServer := httptest.NewServer(router)

	// Create cleanup function
	cleanup := func() {
		testServer.Close()
		cleanupE2EDatabase(t, pool)
		pool.Close()
		db.Close()
	}

	return &E2ETestSetup{
		TestServer:      testServer,
		DB:              db,
		Pool:            pool,
		UserRepo:        userRepo,
		RoleRepo:        roleRepo,
		UserRoleRepo:    userRoleRepo,
		PasswordService: passwordService,
		JWTService:      jwtService,
		UserService:     userSvc,
		Cleanup:         cleanup,
	}
}

// setupE2EDatabaseSchema creates the database schema for E2E tests
func setupE2EDatabaseSchema(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()

	queries := []string{
		// Enable UUID extension
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,

		// Drop tables if they exist (clean start)
		`DROP TABLE IF EXISTS user_roles CASCADE`,
		`DROP TABLE IF EXISTS users CASCADE`,
		`DROP TABLE IF EXISTS roles CASCADE`,

		// Create users table
		`CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			phone VARCHAR(20),
			is_active BOOLEAN DEFAULT true,
			is_verified BOOLEAN DEFAULT false,
			last_login_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create roles table
		`CREATE TABLE roles (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name VARCHAR(100) UNIQUE NOT NULL,
			description TEXT,
			permissions TEXT[] DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create user_roles table
		`CREATE TABLE user_roles (
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
			assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
			PRIMARY KEY (user_id, role_id)
		)`,

		// Create trigger function for updated_at
		`CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ language 'plpgsql'`,

		// Create triggers
		`CREATE TRIGGER update_users_updated_at
			BEFORE UPDATE ON users
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`CREATE TRIGGER update_roles_updated_at
			BEFORE UPDATE ON roles
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		// Create indexes
		`CREATE INDEX idx_users_email ON users(email)`,
		`CREATE INDEX idx_users_username ON users(username)`,
		`CREATE INDEX idx_roles_name ON roles(name)`,
		`CREATE INDEX idx_user_roles_user_id ON user_roles(user_id)`,
		`CREATE INDEX idx_user_roles_role_id ON user_roles(role_id)`,
	}

	for _, query := range queries {
		_, err := pool.Exec(ctx, query)
		require.NoError(t, err, fmt.Sprintf("Failed to execute schema query: %s", query))
	}
}

// cleanupE2EDatabase cleans up the database after E2E tests
func cleanupE2EDatabase(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()

	queries := []string{
		`DELETE FROM user_roles`,
		`DELETE FROM users`,
		`DELETE FROM roles`,
	}

	for _, query := range queries {
		_, err := pool.Exec(ctx, query)
		if err != nil {
			t.Logf("Warning: failed to cleanup database: %v", err)
		}
	}
}

// CreateE2ETestUser creates a test user in the database
func CreateE2ETestUser(t *testing.T, setup *E2ETestSetup, email, username, password string) *entities.User {
	ctx := context.Background()

	// Hash password
	hash, err := setup.PasswordService.HashPassword(password)
	require.NoError(t, err)

	// Create user
	user := &entities.User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		PasswordHash: hash,
		FirstName:    "Test",
		LastName:     "User",
		Phone:        "+1234567890",
		IsActive:     true,
		IsVerified:   true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Insert into database
	err = setup.UserRepo.Create(ctx, user)
	require.NoError(t, err)

	return user
}

// CreateE2ETestRole creates a test role in the database
func CreateE2ETestRole(t *testing.T, setup *E2ETestSetup, name, description string, permissions []string) *entities.Role {
	ctx := context.Background()

	role := &entities.Role{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Permissions: permissions,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err := setup.RoleRepo.CreateRole(ctx, role)
	require.NoError(t, err)

	return role
}

// AuthenticateE2EUser authenticates a user and returns the token
func AuthenticateE2EUser(t *testing.T, setup *E2ETestSetup, email, password string) string {
	loginReq := map[string]string{
		"email":    email,
		"password": password,
	}

	body, _ := json.Marshal(loginReq)
	resp, err := http.Post(
		setup.TestServer.URL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	require.NoError(t, err)

	token, ok := loginResp["access_token"].(string)
	require.True(t, ok, "Access token not found in response")

	return token
}

// MakeAPIRequest makes an HTTP request to the test server
func MakeAPIRequest(t *testing.T, setup *E2ETestSetup, method, endpoint string, body interface{}, token string) *http.Response {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, setup.TestServer.URL+endpoint, reqBody)
	require.NoError(t, err)

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

// AssertAPIResponse asserts that an API response has the expected status and body
func AssertAPIResponse(t *testing.T, resp *http.Response, expectedStatus int, expectedBody interface{}) {
	defer resp.Body.Close()

	require.Equal(t, expectedStatus, resp.StatusCode,
		fmt.Sprintf("Expected status %d, got %d", expectedStatus, resp.StatusCode))

	if expectedBody != nil {
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var actualBody interface{}
		err = json.Unmarshal(body, &actualBody)
		require.NoError(t, err, "Failed to unmarshal response body")

		require.Equal(t, expectedBody, actualBody,
			fmt.Sprintf("Expected body %+v, got %+v", expectedBody, actualBody))
	}
}

// WaitForE2EDatabase waits for the E2E database to be ready
func WaitForE2EDatabase(t *testing.T, dbURL string, maxAttempts int) {
	ctx := context.Background()

	for i := 0; i < maxAttempts; i++ {
		pool, err := pgxpool.New(ctx, dbURL)
		if err == nil {
			if err := pool.Ping(ctx); err == nil {
				pool.Close()
				return
			}
			pool.Close()
		}

		if i < maxAttempts-1 {
			t.Logf("E2E Database not ready, waiting... (attempt %d/%d)", i+1, maxAttempts)
			time.Sleep(time.Second * 2)
		}
	}

	require.Fail(t, "E2E Database not ready after %d attempts", maxAttempts)
}

// Mock repository implementations for E2E testing

type E2EMockUserRepository struct {
	pool  *pgxpool.Pool
	users map[uuid.UUID]*entities.User
}

func (m *E2EMockUserRepository) Create(ctx context.Context, user *entities.User) error {
	if m.users == nil {
		m.users = make(map[uuid.UUID]*entities.User)
	}
	m.users[user.ID] = user
	return nil
}

func (m *E2EMockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	if user, exists := m.users[id]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (m *E2EMockUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *E2EMockUserRepository) GetByUsername(ctx context.Context, username string) (*entities.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *E2EMockUserRepository) Update(ctx context.Context, user *entities.User) error {
	if _, exists := m.users[user.ID]; exists {
		m.users[user.ID] = user
		return nil
	}
	return fmt.Errorf("user not found")
}

func (m *E2EMockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if _, exists := m.users[id]; exists {
		delete(m.users, id)
		return nil
	}
	return fmt.Errorf("user not found")
}

func (m *E2EMockUserRepository) List(ctx context.Context, filter repositories.UserFilter) ([]*entities.User, error) {
	var users []*entities.User
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func (m *E2EMockUserRepository) Count(ctx context.Context, filter repositories.UserFilter) (int, error) {
	return len(m.users), nil
}

func (m *E2EMockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, err := m.GetByEmail(ctx, email)
	return err == nil, nil
}

func (m *E2EMockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	_, err := m.GetByUsername(ctx, username)
	return err == nil, nil
}

func (m *E2EMockUserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	if user, exists := m.users[userID]; exists {
		user.UpdateLastLogin()
		return nil
	}
	return fmt.Errorf("user not found")
}

func (m *E2EMockUserRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return []string{"user"}, nil
}

func (m *E2EMockUserRepository) AssignRole(ctx context.Context, userID uuid.UUID, roleName string, assignedBy uuid.UUID) error {
	return nil
}

type E2EMockRoleRepository struct {
	pool  *pgxpool.Pool
	roles map[string]*entities.Role
}

func (m *E2EMockRoleRepository) Create(ctx context.Context, role *entities.Role) error {
	if m.roles == nil {
		m.roles = make(map[string]*entities.Role)
	}
	m.roles[role.Name] = role
	return nil
}

func (m *E2EMockRoleRepository) GetByID(ctx context.Context, id string) (*entities.Role, error) {
	for _, role := range m.roles {
		if role.ID.String() == id {
			return role, nil
		}
	}
	return nil, fmt.Errorf("role not found")
}

func (m *E2EMockRoleRepository) GetByName(ctx context.Context, name string) (*entities.Role, error) {
	if role, exists := m.roles[name]; exists {
		return role, nil
	}
	return nil, fmt.Errorf("role not found")
}

func (m *E2EMockRoleRepository) Update(ctx context.Context, role *entities.Role) error {
	if _, exists := m.roles[role.Name]; exists {
		m.roles[role.Name] = role
		return nil
	}
	return fmt.Errorf("role not found")
}

func (m *E2EMockRoleRepository) Delete(ctx context.Context, id string) error {
	for name, role := range m.roles {
		if role.ID.String() == id {
			delete(m.roles, name)
			return nil
		}
	}
	return fmt.Errorf("role not found")
}

func (m *E2EMockRoleRepository) List(ctx context.Context, filter repositories.RoleFilter) ([]*entities.Role, error) {
	var roles []*entities.Role
	for _, role := range m.roles {
		roles = append(roles, role)
	}
	return roles, nil
}

func (m *E2EMockRoleRepository) Count(ctx context.Context, filter repositories.RoleFilter) (int, error) {
	return len(m.roles), nil
}

type E2EMockUserRoleRepository struct {
	pool *pgxpool.Pool
}

func (m *E2EMockUserRoleRepository) AssignRole(ctx context.Context, userID, roleID, assignedBy string) error {
	return nil
}

func (m *E2EMockUserRoleRepository) RemoveRole(ctx context.Context, userID, roleID string) error {
	return nil
}

func (m *E2EMockUserRoleRepository) GetUserRoles(ctx context.Context, userID string) ([]*entities.Role, error) {
	return []*entities.Role{}, nil
}

func (m *E2EMockUserRoleRepository) GetUsersByRole(ctx context.Context, roleID string) ([]*entities.User, error) {
	return []*entities.User{}, nil
}

func (m *E2EMockUserRoleRepository) HasRole(ctx context.Context, userID, roleID string) (bool, error) {
	return false, nil
}

// MockRoleRepository implements a minimal RoleRepository for testing
type MockRoleRepository struct{}

func (m *MockRoleRepository) CreateRole(ctx context.Context, role *entities.Role) error { return nil }
func (m *MockRoleRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (*entities.Role, error) {
	return nil, nil
}
func (m *MockRoleRepository) GetRoleByName(ctx context.Context, name string) (*entities.Role, error) {
	return nil, nil
}
func (m *MockRoleRepository) GetAllRoles(ctx context.Context) ([]*entities.Role, error) {
	return []*entities.Role{}, nil
}
func (m *MockRoleRepository) UpdateRole(ctx context.Context, role *entities.Role) error { return nil }
func (m *MockRoleRepository) DeleteRole(ctx context.Context, id uuid.UUID) error        { return nil }
func (m *MockRoleRepository) RoleExists(ctx context.Context, name string) (bool, error) {
	return false, nil
}
func (m *MockRoleRepository) AssignRoleToUser(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error {
	return nil
}
func (m *MockRoleRepository) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return nil
}
func (m *MockRoleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*entities.Role, error) {
	return []*entities.Role{}, nil
}
func (m *MockRoleRepository) GetUsersWithRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}
func (m *MockRoleRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return []string{}, nil
}
func (m *MockRoleRepository) HasUserRole(ctx context.Context, userID, roleID uuid.UUID) (bool, error) {
	return false, nil
}
func (m *MockRoleRepository) RemoveAllUserRoles(ctx context.Context, userID uuid.UUID) error {
	return nil
}
func (m *MockRoleRepository) UserHasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	return false, nil
}
func (m *MockRoleRepository) UserHasAnyPermission(ctx context.Context, userID uuid.UUID, permissions ...string) (bool, error) {
	return false, nil
}
func (m *MockRoleRepository) UserHasAllPermissions(ctx context.Context, userID uuid.UUID, permissions ...string) (bool, error) {
	return false, nil
}
func (m *MockRoleRepository) AddPermissionToRole(ctx context.Context, roleID uuid.UUID, permission string) error {
	return nil
}
func (m *MockRoleRepository) RemovePermissionFromRole(ctx context.Context, roleID uuid.UUID, permission string) error {
	return nil
}
func (m *MockRoleRepository) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	return []string{}, nil
}
func (m *MockRoleRepository) CreateDefaultRoles(ctx context.Context) error { return nil }
func (m *MockRoleRepository) GetRoleAssignmentHistory(ctx context.Context, userID uuid.UUID) ([]*entities.UserRole, error) {
	return []*entities.UserRole{}, nil
}
