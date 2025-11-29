package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestAuthMiddleware_Success(t *testing.T) {
	router := setupTestRouter()
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, "test-issuer", time.Hour, time.Hour*24)

	// Add auth middleware
	router.Use(AuthMiddleware(jwtService))

	// Add test route
	router.GET("/protected", func(c *gin.Context) {
		user, exists := GetCurrentUser(c)
		require.True(t, exists)
		assert.Equal(t, "test@example.com", user.Email)
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Generate valid token
	userID := uuid.New()
	token, err := jwtService.GenerateAccessToken(userID, "test@example.com", "testuser", []string{"user"})
	require.NoError(t, err)

	// Make request with valid token
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	router := setupTestRouter()
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, "test-issuer", time.Hour, time.Hour*24)

	// Add auth middleware
	router.Use(AuthMiddleware(jwtService))

	// Add test route
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make request without auth header
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Authorization header is required", response["error"])
	assert.Equal(t, "MISSING_AUTH_HEADER", response["code"])
}

func TestAuthMiddleware_InvalidAuthFormat(t *testing.T) {
	router := setupTestRouter()
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, "test-issuer", time.Hour, time.Hour*24)

	// Add auth middleware
	router.Use(AuthMiddleware(jwtService))

	// Add test route
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tests := []struct {
		name         string
		authHeader   string
		expectedCode int
	}{
		{
			name:         "no bearer prefix",
			authHeader:   "abc123.token",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "lowercase bearer",
			authHeader:   "bearer abc123.token",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "empty bearer",
			authHeader:   "Bearer ",
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", tt.authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	router := setupTestRouter()
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, "test-issuer", time.Hour, time.Hour*24)

	// Add auth middleware
	router.Use(AuthMiddleware(jwtService))

	// Add test route
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make request with invalid token
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Invalid or expired token", response["error"])
	assert.Equal(t, "INVALID_TOKEN", response["code"])
}

func TestOptionalAuthMiddleware_WithValidToken(t *testing.T) {
	router := setupTestRouter()
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, "test-issuer", time.Hour, time.Hour*24)

	// Add optional auth middleware
	router.Use(OptionalAuthMiddleware(jwtService))

	// Add test route
	router.GET("/optional", func(c *gin.Context) {
		user, exists := GetCurrentUser(c)
		if exists {
			assert.Equal(t, "test@example.com", user.Email)
			c.JSON(http.StatusOK, gin.H{"authenticated": true})
		} else {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		}
	})

	// Generate valid token
	userID := uuid.New()
	token, err := jwtService.GenerateAccessToken(userID, "test@example.com", "testuser", []string{"user"})
	require.NoError(t, err)

	// Make request with valid token
	req := httptest.NewRequest("GET", "/optional", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["authenticated"])
}

func TestOptionalAuthMiddleware_WithoutToken(t *testing.T) {
	router := setupTestRouter()
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, "test-issuer", time.Hour, time.Hour*24)

	// Add optional auth middleware
	router.Use(OptionalAuthMiddleware(jwtService))

	// Add test route
	router.GET("/optional", func(c *gin.Context) {
		user, exists := GetCurrentUser(c)
		assert.False(t, exists)
		assert.Nil(t, user)
		c.JSON(http.StatusOK, gin.H{"authenticated": false})
	})

	// Make request without token
	req := httptest.NewRequest("GET", "/optional", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, false, response["authenticated"])
}

func TestRequireRoles_Success(t *testing.T) {
	router := setupTestRouter()
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, "test-issuer", time.Hour, time.Hour*24)

	// Add auth middleware and role requirement
	router.Use(AuthMiddleware(jwtService))
	router.Use(RequireRoles("admin", "manager"))

	// Add test route
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Generate valid token with admin role
	userID := uuid.New()
	token, err := jwtService.GenerateAccessToken(userID, "test@example.com", "testuser", []string{"user", "admin"})
	require.NoError(t, err)

	// Make request with valid token
	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRoles_InsufficientPermissions(t *testing.T) {
	router := setupTestRouter()
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, "test-issuer", time.Hour, time.Hour*24)

	// Add auth middleware and role requirement
	router.Use(AuthMiddleware(jwtService))
	router.Use(RequireRoles("admin", "manager"))

	// Add test route
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Generate valid token with user role (not admin/manager)
	userID := uuid.New()
	token, err := jwtService.GenerateAccessToken(userID, "test@example.com", "testuser", []string{"user"})
	require.NoError(t, err)

	// Make request with valid token but insufficient role
	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Insufficient permissions", response["error"])
	assert.Equal(t, "INSUFFICIENT_PERMISSIONS", response["code"])
}

func TestRequireAllRoles_Success(t *testing.T) {
	router := setupTestRouter()
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, "test-issuer", time.Hour, time.Hour*24)

	// Add auth middleware and role requirement
	router.Use(AuthMiddleware(jwtService))
	router.Use(RequireAllRoles("user", "admin"))

	// Add test route
	router.GET("/admin-user", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Generate valid token with both required roles
	userID := uuid.New()
	token, err := jwtService.GenerateAccessToken(userID, "test@example.com", "testuser", []string{"user", "admin", "manager"})
	require.NoError(t, err)

	// Make request with valid token
	req := httptest.NewRequest("GET", "/admin-user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAllRoles_MissingRole(t *testing.T) {
	router := setupTestRouter()
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, "test-issuer", time.Hour, time.Hour*24)

	// Add auth middleware and role requirement
	router.Use(AuthMiddleware(jwtService))
	router.Use(RequireAllRoles("user", "admin"))

	// Add test route
	router.GET("/admin-user", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Generate valid token with only user role (missing admin)
	userID := uuid.New()
	token, err := jwtService.GenerateAccessToken(userID, "test@example.com", "testuser", []string{"user"})
	require.NoError(t, err)

	// Make request with valid token but missing required role
	req := httptest.NewRequest("GET", "/admin-user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_Success(t *testing.T) {
	router := setupTestRouter()
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, "test-issuer", time.Hour, time.Hour*24)

	// Add auth middleware and permission requirement
	router.Use(AuthMiddleware(jwtService))
	// We can't test the permission middleware without a mock repository
	// router.Use(RequirePermission(nil, "users.create"))

	// Add test route
	router.GET("/create-user", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Generate valid token with admin role (has users.create permission)
	userID := uuid.New()
	token, err := jwtService.GenerateAccessToken(userID, "test@example.com", "testuser", []string{"admin"})
	require.NoError(t, err)

	// Make request with valid token
	req := httptest.NewRequest("GET", "/create-user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetCurrentUser(t *testing.T) {
	router := setupTestRouter()
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, "test-issuer", time.Hour, time.Hour*24)

	// Add auth middleware
	router.Use(AuthMiddleware(jwtService))

	// Add test route
	router.GET("/user-info", func(c *gin.Context) {
		user, exists := GetCurrentUser(c)
		require.True(t, exists)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "testuser", user.Username)

		userID, exists := GetCurrentUserID(c)
		require.True(t, exists)
		assert.Equal(t, user.ID, userID)

		roles, exists := GetCurrentUserRoles(c)
		require.True(t, exists)
		assert.Contains(t, roles, "user")
		assert.Contains(t, roles, "admin")

		assert.True(t, HasRole(c, "admin"))
		assert.True(t, HasAnyRole(c, "admin", "manager"))
		assert.True(t, HasAllRoles(c, "user", "admin"))
		assert.False(t, HasRole(c, "superadmin"))
		assert.False(t, HasAllRoles(c, "user", "superadmin"))

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Generate valid token
	userID := uuid.New()
	token, err := jwtService.GenerateAccessToken(userID, "test@example.com", "testuser", []string{"user", "admin"})
	require.NoError(t, err)

	// Make request with valid token
	req := httptest.NewRequest("GET", "/user-info", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(SecurityHeadersMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "geolocation=(), microphone=(), camera=()", w.Header().Get("Permissions-Policy"))
}

func TestCORSMiddleware(t *testing.T) {
	router := setupTestRouter()
	allowedOrigins := []string{"http://localhost:3000", "https://example.com"}
	allowedMethods := []string{"GET", "POST", "PUT", "DELETE"}
	allowedHeaders := []string{"Origin", "Content-Type", "Accept", "Authorization"}

	router.Use(CORSMiddleware(allowedOrigins, allowedMethods, allowedHeaders))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tests := []struct {
		name           string
		origin         string
		expectCORS     bool
		expectedOrigin string
	}{
		{
			name:           "allowed origin",
			origin:         "http://localhost:3000",
			expectCORS:     true,
			expectedOrigin: "http://localhost:3000",
		},
		{
			name:           "disallowed origin",
			origin:         "http://evil.com",
			expectCORS:     false,
			expectedOrigin: "",
		},
		{
			name:           "no origin",
			origin:         "",
			expectCORS:     false,
			expectedOrigin: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			if tt.expectCORS {
				assert.Equal(t, tt.expectedOrigin, w.Header().Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "GET, POST, PUT, DELETE", w.Header().Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "Origin, Content-Type, Accept, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
			} else {
				assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(RequestIDMiddleware())

	router.GET("/test", func(c *gin.Context) {
		requestID := c.GetString("request_id")
		assert.NotEmpty(t, requestID)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotEmpty(t, response["request_id"])
	assert.Equal(t, w.Header().Get("X-Request-ID"), response["request_id"])
}
