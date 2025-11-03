//go:build integration
// +build integration

package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/erpgo/erpgo/internal/interfaces/http/handlers"
	"github.com/erpgo/erpgo/pkg/config"
	"github.com/erpgo/erpgo/pkg/database"
	"github.com/erpgo/erpgo/tests/integration/testutil"
)

type UserAPITestSuite struct {
	suite.Suite
	server    *httptest.Server
	db        *database.DB
	userHandler *handlers.UserHandler
	testUtil  *testutil.TestUtil
}

func (suite *UserAPITestSuite) SetupSuite() {
	// Setup test database
	suite.testUtil = testutil.NewTestUtil(suite.T())
	suite.db = suite.testUtil.SetupDatabase()

	// Setup configuration
	cfg := &config.Config{
		DatabaseURL: suite.testUtil.GetDatabaseURL(),
		JWTSecret:   "test-secret",
	}

	// Setup handlers
	suite.userHandler = handlers.NewUserHandler(cfg)

	// Setup test server
	suite.server = httptest.NewServer(setupTestRouter(suite.userHandler))
}

func (suite *UserAPITestSuite) TearDownSuite() {
	suite.server.Close()
	suite.testUtil.Cleanup()
}

func (suite *UserAPITestSuite) SetupTest() {
	// Clean up database before each test
	suite.testUtil.CleanupDatabase()
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
				"password":  "password123",
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
				"password":  "password123",
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
			expectedError:  "required field",
		},
		{
			name: "weak password",
			requestBody: map[string]interface{}{
				"email":     "test@example.com",
				"username":  "testuser",
				"password":  "123",
				"first_name": "Test",
				"last_name":  "User",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Prepare request
			body, err := json.Marshal(tt.requestBody)
			require.NoError(suite.T(), err)

			req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/users", bytes.NewBuffer(body))
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
				assert.True(suite.T(), response["success"].(bool))
				user := response["data"].(map[string]interface{})
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
	user := suite.testUtil.CreateTestUser()

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
			authToken:      suite.testUtil.CreateAuthToken(user.ID),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get non-existent user",
			userID:         "00000000-0000-0000-0000-000000000000",
			authToken:      suite.testUtil.CreateAuthToken(user.ID),
			expectedStatus: http.StatusNotFound,
			expectedError:  "user not found",
		},
		{
			name:           "invalid user ID",
			userID:         "invalid-uuid",
			authToken:      suite.testUtil.CreateAuthToken(user.ID),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid user ID",
		},
		{
			name:           "unauthorized access",
			userID:         user.ID.String(),
			authToken:      "invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "unauthorized",
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
				assert.True(suite.T(), response["success"].(bool))
				user := response["data"].(map[string]interface{})
				assert.Equal(suite.T(), tt.userID, user["id"])
				assert.Equal(suite.T(), user.Email, user["email"])
				assert.Equal(suite.T(), user.Username, user["username"])
			}
		})
	}
}

func (suite *UserAPITestSuite) TestUpdateUser() {
	// Create a test user
	user := suite.testUtil.CreateTestUser()
	authToken := suite.testUtil.CreateAuthToken(user.ID)

	tests := []struct {
		name           string
		userID         string
		requestBody    interface{}
		authToken      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "update user successfully",
			userID: user.ID.String(),
			requestBody: map[string]interface{}{
				"first_name": "Updated",
				"last_name":  "Name",
				"phone":      "+9876543210",
			},
			authToken:      authToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:   "update user email (should fail)",
			userID: user.ID.String(),
			requestBody: map[string]interface{}{
				"email": "newemail@example.com",
			},
			authToken:      authToken,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email cannot be updated",
		},
		{
			name:           "update non-existent user",
			userID:         "00000000-0000-0000-0000-000000000000",
			requestBody:    map[string]interface{}{"first_name": "Updated"},
			authToken:      authToken,
			expectedStatus: http.StatusNotFound,
			expectedError:  "user not found",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Prepare request
			body, err := json.Marshal(tt.requestBody)
			require.NoError(suite.T(), err)

			req, err := http.NewRequest(http.MethodPut, suite.server.URL+"/api/v1/users/"+tt.userID, bytes.NewBuffer(body))
			require.NoError(suite.T(), err)
			req.Header.Set("Content-Type", "application/json")
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
				assert.True(suite.T(), response["success"].(bool))
				user := response["data"].(map[string]interface{})
				assert.Equal(suite.T(), tt.userID, user["id"])
			}
		})
	}
}

func (suite *UserAPITestSuite) TestListUsers() {
	// Create test users
	users := []*domain.User{
		suite.testUtil.CreateTestUser(),
		suite.testUtil.CreateTestUser(),
		suite.testUtil.CreateTestUser(),
	}

	authToken := suite.testUtil.CreateAuthToken(users[0].ID)

	tests := []struct {
		name           string
		queryParams    string
		authToken      string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "list all users",
			queryParams:    "",
			authToken:      authToken,
			expectedStatus: http.StatusOK,
			expectedCount:  len(users),
		},
		{
			name:           "list users with pagination",
			queryParams:    "?page=1&limit=2",
			authToken:      authToken,
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "list users with search",
			queryParams:    "?search=" + users[0].Username,
			authToken:      authToken,
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "list users without auth",
			queryParams:    "",
			authToken:      "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Prepare request
			url := suite.server.URL + "/api/v1/users" + tt.queryParams
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(suite.T(), err)

			if tt.authToken != "" {
				req.Header.Set("Authorization", "Bearer "+tt.authToken)
			}

			// Make request
			resp, err := http.DefaultClient.Do(req)
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			// Check status
			assert.Equal(suite.T(), tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				// Parse response
				var response map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(suite.T(), err)

				assert.True(suite.T(), response["success"].(bool))
				data := response["data"].(map[string]interface{})
				items := data["items"].([]interface{})
				assert.Equal(suite.T(), tt.expectedCount, len(items))

				// Check pagination structure
				pagination := data["pagination"].(map[string]interface{})
				assert.Contains(suite.T(), pagination, "page")
				assert.Contains(suite.T(), pagination, "limit")
				assert.Contains(suite.T(), pagination, "total")
			}
		})
	}
}

func (suite *UserAPITestSuite) TestDeleteUser() {
	// Create a test user
	user := suite.testUtil.CreateTestUser()
	authToken := suite.testUtil.CreateAuthToken(user.ID)

	tests := []struct {
		name           string
		userID         string
		authToken      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "delete user successfully",
			userID:         user.ID.String(),
			authToken:      authToken,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "delete non-existent user",
			userID:         "00000000-0000-0000-0000-000000000000",
			authToken:      authToken,
			expectedStatus: http.StatusNotFound,
			expectedError:  "user not found",
		},
		{
			name:           "delete user without auth",
			userID:         user.ID.String(),
			authToken:      "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Prepare request
			req, err := http.NewRequest(http.MethodDelete, suite.server.URL+"/api/v1/users/"+tt.userID, nil)
			require.NoError(suite.T(), err)
			req.Header.Set("Authorization", "Bearer "+tt.authToken)

			// Make request
			resp, err := http.DefaultClient.Do(req)
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			// Check status
			assert.Equal(suite.T(), tt.expectedStatus, resp.StatusCode)

			if tt.expectedError != "" {
				// Parse error response
				var response map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(suite.T(), err)
				assert.Contains(suite.T(), response["error"], tt.expectedError)
			}
		})
	}
}

// Test runner
func TestUserAPITestSuite(t *testing.T) {
	suite.Run(t, new(UserAPITestSuite))
}

// Helper function to setup test router
func setupTestRouter(userHandler *handlers.UserHandler) http.Handler {
	mux := http.NewServeMux()

	// User routes
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userHandler.ListUsers(w, r)
		case http.MethodPost:
			userHandler.CreateUser(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/users/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userHandler.GetUser(w, r)
		case http.MethodPut:
			userHandler.UpdateUser(w, r)
		case http.MethodDelete:
			userHandler.DeleteUser(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}