package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"erpgo/pkg/config"
)

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.Nop()

	tests := []struct {
		name           string
		corsConfig     config.CORSConfig
		origin         string
		method         string
		expectedStatus int
		expectCORS     bool
	}{
		{
			name: "Development - allowed origin",
			corsConfig: config.CORSConfig{
				Origins:      []string{"http://localhost:3000"},
				Methods:      []string{"GET", "POST"},
				Headers:      []string{"Content-Type"},
				MaxAge:       86400,
				Credentials:  true,
				IsProduction: false,
			},
			origin:         "http://localhost:3000",
			method:         "GET",
			expectedStatus: 200,
			expectCORS:     true,
		},
		{
			name: "Development - wildcard allowed",
			corsConfig: config.CORSConfig{
				Origins:      []string{"*"},
				Methods:      []string{"GET", "POST"},
				Headers:      []string{"Content-Type"},
				MaxAge:       86400,
				Credentials:  true,
				IsProduction: false,
			},
			origin:         "http://example.com",
			method:         "GET",
			expectedStatus: 200,
			expectCORS:     true,
		},
		{
			name: "Production - specific origin allowed",
			corsConfig: config.CORSConfig{
				Origins:      []string{"https://app.example.com"},
				Methods:      []string{"GET", "POST"},
				Headers:      []string{"Content-Type"},
				MaxAge:       86400,
				Credentials:  true,
				IsProduction: true,
			},
			origin:         "https://app.example.com",
			method:         "GET",
			expectedStatus: 200,
			expectCORS:     true,
		},
		{
			name: "Production - wildcard not allowed",
			corsConfig: config.CORSConfig{
				Origins:      []string{"*"},
				Methods:      []string{"GET", "POST"},
				Headers:      []string{"Content-Type"},
				MaxAge:       86400,
				Credentials:  true,
				IsProduction: true,
			},
			origin:         "https://malicious.com",
			method:         "GET",
			expectedStatus: 403,
			expectCORS:     false,
		},
		{
			name: "Production - HTTPS required",
			corsConfig: config.CORSConfig{
				Origins:      []string{"https://app.example.com"},
				Methods:      []string{"GET", "POST"},
				Headers:      []string{"Content-Type"},
				MaxAge:       86400,
				Credentials:  true,
				IsProduction: true,
			},
			origin:         "http://app.example.com",
			method:         "GET",
			expectedStatus: 403,
			expectCORS:     false,
		},
		{
			name: "Preflight request - allowed",
			corsConfig: config.CORSConfig{
				Origins:      []string{"http://localhost:3000"},
				Methods:      []string{"GET", "POST"},
				Headers:      []string{"Content-Type"},
				MaxAge:       86400,
				Credentials:  true,
				IsProduction: false,
			},
			origin:         "http://localhost:3000",
			method:         "OPTIONS",
			expectedStatus: 204,
			expectCORS:     true,
		},
		{
			name: "No origin header",
			corsConfig: config.CORSConfig{
				Origins:      []string{"http://localhost:3000"},
				Methods:      []string{"GET", "POST"},
				Headers:      []string{"Content-Type"},
				MaxAge:       86400,
				Credentials:  true,
				IsProduction: false,
			},
			origin:         "",
			method:         "GET",
			expectedStatus: 200,
			expectCORS:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create router with CORS middleware
			router := gin.New()
			router.Use(CORSMiddleware(tt.corsConfig, logger))

			// Add a test endpoint
			router.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})

			// Perform request
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check response status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check CORS headers
			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectCORS {
				if tt.origin != "" {
					assert.Equal(t, tt.origin, allowOrigin)
				}
				assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))
				assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
			} else {
				if tt.expectedStatus == 403 {
					assert.Empty(t, allowOrigin)
				}
			}
		})
	}
}

func TestIsOriginValidForProduction(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		expected bool
	}{
		{
			name:     "Valid HTTPS origin",
			origin:   "https://app.example.com",
			expected: true,
		},
		{
			name:     "Invalid HTTP origin in production",
			origin:   "http://app.example.com",
			expected: false,
		},
		{
			name:     "Allowed localhost",
			origin:   "https://localhost:3000",
			expected: true,
		},
		{
			name:     "Allowed 127.0.0.1",
			origin:   "http://127.0.0.1:3000",
			expected: true,
		},
		{
			name:     "Invalid domain format",
			origin:   "https://invalid",
			expected: false,
		},
		{
			name:     "Valid subdomain",
			origin:   "https://api.app.example.com",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOriginValidForProduction(tt.origin)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCORSForAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.Nop()

	corsConfig := config.CORSConfig{
		Origins:      []string{"https://api.example.com"},
		Methods:      []string{"GET", "POST"},
		Headers:      []string{"Content-Type"},
		IsProduction: false,
	}

	// Create router with API CORS middleware
	router := gin.New()
	router.Use(CORSForAPI(corsConfig, logger))

	// Add a test endpoint
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Perform request
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Origin", "https://api.example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "https://api.example.com", w.Header().Get("Access-Control-Allow-Origin"))

	// Check that API-specific headers are included
	allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
	assert.Contains(t, allowHeaders, "X-API-Key")
	assert.Contains(t, allowHeaders, "X-Request-ID")
}

func TestCORSForWebApp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.Nop()

	corsConfig := config.CORSConfig{
		Origins:      []string{"https://app.example.com"},
		Methods:      []string{"GET", "POST"},
		Headers:      []string{"Content-Type"},
		IsProduction: false,
	}

	// Create router with Web App CORS middleware
	router := gin.New()
	router.Use(CORSForWebApp(corsConfig, logger))

	// Add a test endpoint
	router.GET("/app/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Perform request
	req := httptest.NewRequest("GET", "/app/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))

	// Check that web app specific headers are included
	allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
	assert.Contains(t, allowHeaders, "X-Requested-With")
	assert.Contains(t, allowHeaders, "Accept-Language")

	// Check that preflight cache is set to maximum
	assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
}

func TestCORSWithDynamicOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.Nop()

	corsConfig := config.CORSConfig{
		Methods:      []string{"GET", "POST"},
		Headers:      []string{"Content-Type"},
		IsProduction: false,
	}

	// Dynamic validator that allows only example.com origins
	validator := func(origin string) bool {
		return strings.HasSuffix(origin, "example.com")
	}

	// Create router with dynamic CORS middleware
	router := gin.New()
	router.Use(CORSWithDynamicOrigins(validator, corsConfig, logger))

	// Add a test endpoint
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	tests := []struct {
		name           string
		origin         string
		expectedStatus int
		expectAllowed  bool
	}{
		{
			name:           "Allowed dynamic origin",
			origin:         "https://app.example.com",
			expectedStatus: 200,
			expectAllowed:  true,
		},
		{
			name:           "Denied dynamic origin",
			origin:         "https://malicious.com",
			expectedStatus: 200, // Not production, so still allows
			expectAllowed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectAllowed {
				assert.Equal(t, tt.origin, allowOrigin)
			} else {
				assert.Empty(t, allowOrigin)
			}
		})
	}
}

func TestPreflightCache(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create router with preflight cache middleware
	router := gin.New()
	router.Use(PreflightCache())

	// Add a test endpoint
	router.OPTIONS("/test", func(c *gin.Context) {
		c.Status(204)
	})

	// Perform preflight request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check cache headers
	assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
	assert.Equal(t, "public, max-age=86400", w.Header().Get("Cache-Control"))
}

func TestBlockedOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.Nop()

	corsConfig := config.CORSConfig{
		Origins:      []string{"*"}, // Allow all origins
		Methods:      []string{"GET", "POST"},
		Headers:      []string{"Content-Type"},
		IsProduction: false,
	}

	// Create router with CORS middleware
	router := gin.New()
	router.Use(CORSMiddleware(corsConfig, logger))

	// Add a test endpoint
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	blockedOrigins := []string{
		"https://malicious.com",
		"https://evil.com",
		"https://attack.com",
		"https://phishing.com",
	}

	for _, origin := range blockedOrigins {
		t.Run("Blocked origin: "+origin, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", origin)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should be blocked
			assert.Equal(t, 403, w.Code)
			assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}
