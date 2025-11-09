package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInputValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.Nop()
	config := DefaultValidationConfig()

	tests := []struct {
		name           string
		method         string
		path           string
		contentType    string
		body           interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid JSON request",
			method:         "POST",
			path:           "/api/v1/users/register",
			contentType:    "application/json",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
				"username": "testuser",
			},
			expectedStatus: 200,
		},
		{
			name:           "Missing required field",
			method:         "POST",
			path:           "/api/v1/users/register",
			contentType:    "application/json",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
				// missing username
			},
			expectedStatus: 400,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "Invalid email format",
			method:         "POST",
			path:           "/api/v1/users/register",
			contentType:    "application/json",
			body: map[string]interface{}{
				"email":    "invalid-email",
				"password": "password123",
				"username": "testuser",
			},
			expectedStatus: 400,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "XSS attempt",
			method:         "POST",
			path:           "/api/v1/users/register",
			contentType:    "application/json",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
				"username": "<script>alert('xss')</script>",
			},
			expectedStatus: 400,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "SQL injection attempt",
			method:         "POST",
			path:           "/api/v1/users/register",
			contentType:    "application/json",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
				"username": "admin' OR '1'='1",
			},
			expectedStatus: 400,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "Unsupported content type",
			method:         "POST",
			path:           "/api/v1/users/register",
			contentType:    "text/xml",
			body:           "<data>test</data>",
			expectedStatus: 415,
			expectedError:  "UNSUPPORTED_MEDIA_TYPE",
		},
		{
			name:           "Field too long",
			method:         "POST",
			path:           "/api/v1/users/register",
			contentType:    "application/json",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
				"username": string(make([]byte, 101)), // 101 characters, exceeds max of 100
			},
			expectedStatus: 400,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "Valid GET request with query params",
			method:         "GET",
			path:           "/api/v1/users?name=test&email=test@example.com",
			contentType:    "",
			body:           nil,
			expectedStatus: 200,
		},
		{
			name:           "Invalid query params with XSS",
			method:         "GET",
			path:           "/api/v1/users?name=<script>alert('xss')</script>",
			contentType:    "",
			body:           nil,
			expectedStatus: 400,
			expectedError:  "INVALID_QUERY_PARAMS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create router with validation middleware
			router := gin.New()
			router.Use(InputValidation(config, logger))

			// Add a test endpoint
			router.Any("/*path", func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})

			// Prepare request
			var req *http.Request
			if tt.body != nil {
				body, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", tt.contentType)
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			// Perform request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, response["code"])
			}
		})
	}
}

func TestValidateContentType(t *testing.T) {
	tests := []struct {
		name           string
		contentType    string
		allowed        []string
		expectedResult bool
	}{
		{
			name:           "Allowed content type",
			contentType:    "application/json",
			allowed:        []string{"application/json", "text/plain"},
			expectedResult: true,
		},
		{
			name:           "Allowed content type with charset",
			contentType:    "application/json; charset=utf-8",
			allowed:        []string{"application/json"},
			expectedResult: true,
		},
		{
			name:           "Disallowed content type",
			contentType:    "text/xml",
			allowed:        []string{"application/json"},
			expectedResult: false,
		},
		{
			name:           "Empty content type",
			contentType:    "",
			allowed:        []string{"application/json"},
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/", nil)
			c.Request.Header.Set("Content-Type", tt.contentType)

			result := validateContentType(c, tt.allowed)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestContainsSQLInjectionPatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Normal input",
			input:    "John Doe",
			expected: false,
		},
		{
			name:     "SQL injection - union select",
			input:    "' UNION SELECT * FROM users --",
			expected: true,
		},
		{
			name:     "SQL injection - or 1=1",
			input:    "' OR '1'='1",
			expected: true,
		},
		{
			name:     "SQL injection - admin bypass",
			input:    "admin'--",
			expected: true,
		},
		{
			name:     "SQL injection - execute",
			input:    "'; EXEC xp_cmdshell 'dir' --",
			expected: true,
		},
		{
			name:     "Safe string with numbers",
			input:    "Product 12345",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsSQLInjectionPatterns(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal string",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "String with null bytes",
			input:    "Hello\x00World",
			expected: "HelloWorld",
		},
		{
			name:     "String with control characters",
			input:    "Hello\x01World\x02",
			expected: "HelloWorld",
		},
		{
			name:     "String with excessive whitespace",
			input:    "Hello    \t   World",
			expected: "Hello World",
		},
		{
			name:     "String with newline and tab (preserved)",
			input:    "Hello\nWorld\tTest",
			expected: "Hello\nWorld\tTest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateAndSanitize(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultValidationConfig()

	tests := []struct {
		name        string
		input       map[string]interface{}
		expectError bool
		expected    map[string]interface{}
	}{
		{
			name: "Valid data",
			input: map[string]interface{}{
				"email":    "test@example.com",
				"username": "testuser",
				"age":      25,
			},
			expectError: false,
			expected: map[string]interface{}{
				"email":    "test@example.com",
				"username": "testuser",
				"age":      25,
			},
		},
		{
			name: "Data with null bytes",
			input: map[string]interface{}{
				"email":    "test\x00@example.com",
				"username": "testuser",
			},
			expectError: false,
			expected: map[string]interface{}{
				"email":    "test@example.com",
				"username": "testuser",
			},
		},
		{
			name: "Invalid email format",
			input: map[string]interface{}{
				"email":    "invalid-email",
				"username": "testuser",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAndSanitize(tt.input, config, logger)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}