//go:build integration
// +build integration

package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	userService "erpgo/internal/application/services/user"
	"erpgo/internal/domain/users/entities"
	"erpgo/internal/interfaces/http/handlers"
	"erpgo/internal/interfaces/http/routes"
	"erpgo/pkg/auth"
	erpgoDatabase "erpgo/pkg/database"
	"erpgo/tests/integration/testutil"
)

type UserAPITestSuite struct {
	suite.Suite
	server         *httptest.Server
	db             *erpgoDatabase.Database
	userService    userService.Service
	passwordService *auth.PasswordService
	jwtService     *auth.JWTService
	testDB         *testutil.TestDatabase
}

func (suite *UserAPITestSuite) SetupSuite() {
	// Setup test database
	suite.testDB = testutil.SetupTestDatabase(suite.T())
	suite.db = suite.testDB.DB
	suite.passwordService = suite.testDB.PasswordService

	// Create JWT service
	suite.jwtService = auth.NewJWTService("test-secret-key", 24*60*60, 7*24*60*60) // 24 hours access, 7 days refresh

	// Create repositories - Note: Using mock implementations for now
	// In a real implementation, you would create proper repository implementations
	userRepo := &MockUserRepository{db: suite.db}
	roleRepo := &MockRoleRepository{db: suite.db}
	userRoleRepo := &MockUserRoleRepository{db: suite.db}

	// Create user service
	suite.userService = userService.NewUserService(userRepo, roleRepo, userRoleRepo, suite.passwordService, suite.jwtService)

	// Setup test server
	suite.server = httptest.NewServer(setupTestRouter(suite.userService))
}

func (suite *UserAPITestSuite) TearDownSuite() {
	suite.server.Close()
	suite.testDB.Cleanup()
}

func (suite *UserAPITestSuite) SetupTest() {
	// Clean up database before each test
	suite.testDB.Cleanup()
}

func (suite *UserAPITestSuite) TestCreateUser() {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid user creation",
			requestBody: map[string]interface{}{
				"email":     "test@example.com",
				"username":  "testuser",
				"password":  "password123!",
				"first_name": "Test",
				"last_name":  "User",
				"phone":     "+1234567890",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid email",
			requestBody: map[string]interface{}{
				"email":     "invalid-email",
				"username":  "testuser",
				"password":  "password123!",
				"first_name": "Test",
				"last_name":  "User",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid email format",
		},
		{
			name: "missing required fields",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Prepare request
			body, err := json.Marshal(tt.requestBody)
			require.NoError(suite.T(), err)

			req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/auth/register", bytes.NewBuffer(body))
			require.NoError(suite.T(), err)
			req.Header.Set("Content-Type", "application/json")

			// Make request
			resp, err := http.DefaultClient.Do(req)
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			// Check status
			assert.Equal(suite.T(), tt.expectedStatus, resp.StatusCode)

			// Parse response
			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(suite.T(), err)

			if tt.expectedError != "" {
				assert.Contains(suite.T(), response["error"], tt.expectedError)
			} else {
				user := response.(map[string]interface{})
				assert.NotEmpty(suite.T(), user["id"])
				assert.Equal(suite.T(), tt.requestBody.(map[string]interface{})["email"], user["email"])
				assert.Equal(suite.T(), tt.requestBody.(map[string]interface{})["username"], user["username"])
				// Password should not be returned
				assert.NotContains(suite.T(), user, "password")
			}
		})
	}
}

func (suite *UserAPITestSuite) TestGetUser() {
	// Create a test user first
	user := suite.createTestUser()
	authToken := suite.createAuthToken(user.ID, user.Email, user.Username)

	tests := []struct {
		name           string
		userID         string
		authToken      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "get existing user",
			userID:         user.ID.String(),
			authToken:      authToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get non-existent user",
			userID:         "00000000-0000-0000-0000-000000000000",
			authToken:      authToken,
			expectedStatus: http.StatusNotFound,
			expectedError:  "User not found",
		},
		{
			name:           "invalid user ID",
			userID:         "invalid-uuid",
			authToken:      authToken,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "User ID is required",
		},
		{
			name:           "unauthorized access",
			userID:         user.ID.String(),
			authToken:      "invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Prepare request
			req, err := http.NewRequest(http.MethodGet, suite.server.URL+"/api/v1/users/"+tt.userID, nil)
			require.NoError(suite.T(), err)
			req.Header.Set("Authorization", "Bearer "+tt.authToken)

			// Make request
			resp, err := http.DefaultClient.Do(req)
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			// Check status
			assert.Equal(suite.T(), tt.expectedStatus, resp.StatusCode)

			// Parse response
			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(suite.T(), err)

			if tt.expectedError != "" {
				assert.Contains(suite.T(), response["error"], tt.expectedError)
			} else {
				userResp := response.(map[string]interface{})
				assert.Equal(suite.T(), tt.userID, userResp["id"])
				assert.Equal(suite.T(), user.Email, userResp["email"])
				assert.Equal(suite.T(), user.Username, userResp["username"])
			}
		})
	}
}

func (suite *UserAPITestSuite) TestLogin() {
	// Create a test user first
	user := suite.createTestUser()

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid login",
			requestBody: map[string]interface{}{
				"email":    user.Email,
				"password": "testPassword123!",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid email",
			requestBody: map[string]interface{}{
				"email":    "invalid@example.com",
				"password": "testPassword123!",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid credentials",
		},
		{
			name: "invalid password",
			requestBody: map[string]interface{}{
				"email":    user.Email,
				"password": "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid credentials",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Prepare request
			body, err := json.Marshal(tt.requestBody)
			require.NoError(suite.T(), err)

			req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/auth/login", bytes.NewBuffer(body))
			require.NoError(suite.T(), err)
			req.Header.Set("Content-Type", "application/json")

			// Make request
			resp, err := http.DefaultClient.Do(req)
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			// Check status
			assert.Equal(suite.T(), tt.expectedStatus, resp.StatusCode)

			// Parse response
			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(suite.T(), err)

			if tt.expectedError != "" {
				assert.Contains(suite.T(), response["error"], tt.expectedError)
			} else {
				assert.NotEmpty(suite.T(), response["access_token"])
				assert.NotEmpty(suite.T(), response["refresh_token"])
				userResp := response["user"].(map[string]interface{})
				assert.Equal(suite.T(), user.Email, userResp["email"])
			}
		})
	}
}

// Helper methods

func (suite *UserAPITestSuite) createTestUser() *entities.User {
	user := testutil.CreateTestUser(suite.T(), suite.passwordService, "test@example.com", "testuser")
	return user
}

func (suite *UserAPITestSuite) createAuthToken(userID uuid.UUID, email, username string) string {
	token, err := suite.jwtService.GenerateAccessToken(userID, email, username, []string{"user"})
	require.NoError(suite.T(), err)
	return token
}

// Mock implementations for testing

type MockUserRepository struct {
	db *erpgoDatabase.Database
	users map[uuid.UUID]*entities.User
}

func (m *MockUserRepository) Create(ctx context.Context, user *entities.User) error {
	if m.users == nil {
		m.users = make(map[uuid.UUID]*entities.User)
	}
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	if user, exists := m.users[id]; exists {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*entities.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *MockUserRepository) Update(ctx context.Context, user *entities.User) error {
	if _, exists := m.users[user.ID]; exists {
		m.users[user.ID] = user
		return nil
	}
	return errors.New("user not found")
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if _, exists := m.users[id]; exists {
		delete(m.users, id)
		return nil
	}
	return errors.New("user not found")
}

func (m *MockUserRepository) List(ctx context.Context, filter interface{}) ([]*entities.User, error) {
	var users []*entities.User
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func (m *MockUserRepository) Count(ctx context.Context, filter interface{}) (int, error) {
	return len(m.users), nil
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, err := m.GetByEmail(ctx, email)
	return err == nil, nil
}

func (m *MockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	_, err := m.GetByUsername(ctx, username)
	return err == nil, nil
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	if user, exists := m.users[userID]; exists {
		user.UpdateLastLogin()
		return nil
	}
	return errors.New("user not found")
}

func (m *MockUserRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return []string{"user"}, nil
}

func (m *MockUserRepository) AssignRole(ctx context.Context, userID uuid.UUID, roleName string, assignedBy uuid.UUID) error {
	return nil
}

type MockRoleRepository struct {
	db *erpgoDatabase.Database
	roles map[string]*entities.Role
}

func (m *MockRoleRepository) Create(ctx context.Context, role *entities.Role) error {
	if m.roles == nil {
		m.roles = make(map[string]*entities.Role)
	}
	m.roles[role.Name] = role
	return nil
}

func (m *MockRoleRepository) GetByID(ctx context.Context, id string) (*entities.Role, error) {
	for _, role := range m.roles {
		if role.ID.String() == id {
			return role, nil
		}
	}
	return nil, errors.New("role not found")
}

func (m *MockRoleRepository) GetByName(ctx context.Context, name string) (*entities.Role, error) {
	if role, exists := m.roles[name]; exists {
		return role, nil
	}
	return nil, errors.New("role not found")
}

func (m *MockRoleRepository) Update(ctx context.Context, role *entities.Role) error {
	if _, exists := m.roles[role.Name]; exists {
		m.roles[role.Name] = role
		return nil
	}
	return errors.New("role not found")
}

func (m *MockRoleRepository) Delete(ctx context.Context, id string) error {
	for name, role := range m.roles {
		if role.ID.String() == id {
			delete(m.roles, name)
			return nil
		}
	}
	return errors.New("role not found")
}

func (m *MockRoleRepository) List(ctx context.Context, filter interface{}) ([]*entities.Role, error) {
	var roles []*entities.Role
	for _, role := range m.roles {
		roles = append(roles, role)
	}
	return roles, nil
}

func (m *MockRoleRepository) Count(ctx context.Context, filter interface{}) (int, error) {
	return len(m.roles), nil
}

type MockUserRoleRepository struct {
	db *erpgoDatabase.Database
}

func (m *MockUserRoleRepository) AssignRole(ctx context.Context, userID, roleID, assignedBy string) error {
	return nil
}

func (m *MockUserRoleRepository) RemoveRole(ctx context.Context, userID, roleID string) error {
	return nil
}

func (m *MockUserRoleRepository) GetUserRoles(ctx context.Context, userID string) ([]*entities.Role, error) {
	return []*entities.Role{}, nil
}

func (m *MockUserRoleRepository) GetUsersByRole(ctx context.Context, roleID string) ([]*entities.User, error) {
	return []*entities.User{}, nil
}

func (m *MockUserRoleRepository) HasRole(ctx context.Context, userID, roleID string) (bool, error) {
	return false, nil
}

// Helper function to setup test router
func setupTestRouter(userService userService.Service) http.Handler {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create a simple logger for testing
	logger := gin.New()

	// Setup user routes
	routes.SetupUserRoutes(router.Group("/api/v1"), userService, logger)

	return router
}

// Test runner
func TestUserAPITestSuite(t *testing.T) {
	suite.Run(t, new(UserAPITestSuite))
}