package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestEnhancedSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		config          SecurityConfig
		expectedHeaders map[string]string
	}{
		{
			name:   "Production security headers",
			config: DefaultSecurityConfig(true),
			expectedHeaders: map[string]string{
				"X-Frame-Options":           "DENY",
				"X-Content-Type-Options":    "nosniff",
				"X-XSS-Protection":          "1; mode=block",
				"Referrer-Policy":           "strict-origin-when-cross-origin",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
				"Content-Security-Policy":   "default-src 'self'",
				"Permissions-Policy":        "geolocation=(), microphone=(), camera=()",
				"X-DNS-Prefetch-Control":    "off",
			},
		},
		{
			name:   "Development security headers",
			config: DevelopmentSecurityConfig(),
			expectedHeaders: map[string]string{
				"X-Frame-Options":        "DENY",
				"X-Content-Type-Options": "nosniff",
				"X-XSS-Protection":       "1; mode=block",
				"Referrer-Policy":        "strict-origin-when-cross-origin",
			},
		},
		{
			name:   "API security headers",
			config: APISecurityConfig(true),
			expectedHeaders: map[string]string{
				"X-Frame-Options":         "DENY",
				"X-Content-Type-Options":  "nosniff",
				"X-XSS-Protection":        "1; mode=block",
				"Referrer-Policy":         "strict-origin-when-cross-origin",
				"Content-Security-Policy": "default-src 'self'; script-src 'none'; style-src 'none'",
				"X-API-Version":           "v1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create router with security headers middleware
			router := gin.New()
			router.Use(EnhancedSecurityHeaders(tt.config))

			// Add a test endpoint
			router.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})

			// Perform request
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check response
			assert.Equal(t, 200, w.Code)

			// Check security headers
			for header, expectedValue := range tt.expectedHeaders {
				value := w.Header().Get(header)
				if expectedValue != "" {
					assert.NotEmpty(t, value, "Header %s should be set", header)
					if header == "Content-Security-Policy" {
						// For CSP, check that it starts with expected value (it may have more directives)
						assert.Contains(t, value, expectedValue, "CSP should contain expected directive")
					} else if header == "Permissions-Policy" {
						// For Permissions Policy, check that it contains expected features
						assert.Contains(t, value, "geolocation=()", "Permissions policy should contain geolocation restriction")
					} else if header != "Strict-Transport-Security" || tt.config.HSTSEnabled {
						// Only check HSTS if it's enabled
						assert.Equal(t, expectedValue, value, "Header %s should match expected value", header)
					}
				}
			}
		})
	}
}

func TestSecurityHeadersWithEnvironment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		isProduction bool
		expectHSTS   bool
		expectCSP    bool
	}{
		{
			name:         "Production environment",
			isProduction: true,
			expectHSTS:   true,
			expectCSP:    true,
		},
		{
			name:         "Development environment",
			isProduction: false,
			expectHSTS:   false,
			expectCSP:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create router with environment-based security headers
			router := gin.New()
			router.Use(SecurityHeadersWithEnvironment(tt.isProduction))

			// Add a test endpoint
			router.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})

			// Perform request
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check HSTS header
			hsts := w.Header().Get("Strict-Transport-Security")
			if tt.expectHSTS {
				assert.NotEmpty(t, hsts, "HSTS header should be set in production")
				assert.Contains(t, hsts, "max-age=")
			} else {
				assert.Empty(t, hsts, "HSTS header should not be set in development")
			}

			// Check CSP header
			csp := w.Header().Get("Content-Security-Policy")
			if tt.expectCSP {
				assert.NotEmpty(t, csp, "CSP header should be set")
				assert.Contains(t, csp, "default-src")
			}

			// Check basic security headers are always present
			assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
			assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		})
	}
}

func TestCORSSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	allowedOrigins := []string{"https://example.com", "https://app.example.com"}
	allowedMethods := []string{"GET", "POST", "PUT", "DELETE"}
	allowedHeaders := []string{"Content-Type", "Authorization"}

	tests := []struct {
		name           string
		origin         string
		method         string
		isProduction   bool
		expectedStatus int
		expectCORS     bool
	}{
		{
			name:           "Allowed origin in production",
			origin:         "https://example.com",
			method:         "GET",
			isProduction:   true,
			expectedStatus: 200,
			expectCORS:     true,
		},
		{
			name:           "Disallowed origin in production",
			origin:         "https://malicious.com",
			method:         "GET",
			isProduction:   true,
			expectedStatus: 403,
			expectCORS:     false,
		},
		{
			name:           "Allowed origin in development",
			origin:         "https://example.com",
			method:         "GET",
			isProduction:   false,
			expectedStatus: 200,
			expectCORS:     true,
		},
		{
			name:           "Preflight request",
			origin:         "https://example.com",
			method:         "OPTIONS",
			isProduction:   false,
			expectedStatus: 204,
			expectCORS:     true,
		},
		{
			name:           "No origin header",
			origin:         "",
			method:         "GET",
			isProduction:   false,
			expectedStatus: 200,
			expectCORS:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create router with CORS security headers
			router := gin.New()
			router.Use(CORSSecurityHeaders(allowedOrigins, allowedMethods, allowedHeaders, tt.isProduction))

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

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check CORS headers
			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectCORS {
				if tt.origin != "" {
					assert.Equal(t, tt.origin, allowOrigin)
				}
				assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
				assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))
				assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
			} else {
				assert.Empty(t, allowOrigin)
			}
		})
	}
}

func TestSensitiveEndpointDetection(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{
			path:     "/api/v1/auth/login",
			expected: true,
		},
		{
			path:     "/api/v1/users/profile",
			expected: true,
		},
		{
			path:     "/api/v1/admin/dashboard",
			expected: true,
		},
		{
			path:     "/api/v1/products",
			expected: false,
		},
		{
			path:     "/api/v1/public/status",
			expected: false,
		},
		{
			path:     "/api/v1/payment/process",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isSensitiveEndpoint(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildCSP(t *testing.T) {
	tests := []struct {
		name     string
		config   SecurityConfig
		contains []string
	}{
		{
			name: "Basic CSP",
			config: SecurityConfig{
				CSPEnabled:    true,
				CSPDefaultSrc: "'self'",
				CSPScriptSrc:  "'self' 'unsafe-inline'",
			},
			contains: []string{"default-src 'self'", "script-src 'self' 'unsafe-inline'"},
		},
		{
			name: "CSP with upgrade insecure requests",
			config: SecurityConfig{
				CSPEnabled:                 true,
				CSPDefaultSrc:              "'self'",
				CSPUpgradeInsecureRequests: true,
			},
			contains: []string{"default-src 'self'", "upgrade-insecure-requests"},
		},
		{
			name: "Complete CSP",
			config: SecurityConfig{
				CSPEnabled:    true,
				CSPDefaultSrc: "'self'",
				CSPScriptSrc:  "'self'",
				CSPStyleSrc:   "'self'",
				CSPImgSrc:     "'self' data:",
			},
			contains: []string{
				"default-src 'self'",
				"script-src 'self'",
				"style-src 'self'",
				"img-src 'self' data:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.buildCSP()
			assert.NotEmpty(t, result)

			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestBuildHSTS(t *testing.T) {
	tests := []struct {
		name     string
		config   SecurityConfig
		expected string
	}{
		{
			name: "HSTS disabled",
			config: SecurityConfig{
				HSTSEnabled: false,
			},
			expected: "",
		},
		{
			name: "Basic HSTS",
			config: SecurityConfig{
				HSTSEnabled: true,
				HSTSMaxAge:  31536000,
			},
			expected: "max-age=31536000",
		},
		{
			name: "HSTS with subdomains",
			config: SecurityConfig{
				HSTSEnabled:           true,
				HSTSMaxAge:            31536000,
				HSTSIncludeSubDomains: true,
			},
			expected: "max-age=31536000; includeSubDomains",
		},
		{
			name: "Complete HSTS",
			config: SecurityConfig{
				HSTSEnabled:           true,
				HSTSMaxAge:            31536000,
				HSTSIncludeSubDomains: true,
				HSTSPreload:           true,
			},
			expected: "max-age=31536000; includeSubDomains; preload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.buildHSTS()
			assert.Equal(t, tt.expected, result)
		})
	}
}
