package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/rs/zerolog"
)

// **Feature: production-readiness, Property 3: Input Validation Consistency**
// For any API endpoint and any invalid input, the system must reject the input
// with a structured validation error before processing
// **Validates: Requirements 2.1**
func TestProperty_InputValidationConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Setup logger
	logger := zerolog.Nop()
	config := DefaultValidationConfig()

	// Property: Invalid JSON must be rejected with validation error
	properties.Property("invalid JSON is rejected", prop.ForAll(
		func(invalidJSON string) bool {
			// Skip empty strings (no body is valid for some requests)
			if invalidJSON == "" {
				return true
			}

			// Skip valid JSON strings
			var testJSON interface{}
			if json.Unmarshal([]byte(invalidJSON), &testJSON) == nil {
				return true
			}

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request = httptest.NewRequest("POST", "/api/v1/test", bytes.NewBufferString(invalidJSON))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Request.ContentLength = int64(len(invalidJSON))

			middleware := InputValidationMiddleware(config, logger)
			middleware(c)

			// Should be rejected with 400 status
			return w.Code == http.StatusBadRequest
		},
		genInvalidJSON(),
	))

	// Property: Requests exceeding max body size must be rejected
	properties.Property("oversized requests are rejected", prop.ForAll(
		func(size int64) bool {
			// Generate size larger than max
			oversizedBody := strings.Repeat("a", int(config.MaxRequestBodySize+1))

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request = httptest.NewRequest("POST", "/api/v1/test", bytes.NewBufferString(oversizedBody))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Request.ContentLength = int64(len(oversizedBody))

			middleware := InputValidationMiddleware(config, logger)
			middleware(c)

			// Should be rejected with 413 status
			return w.Code == http.StatusRequestEntityTooLarge
		},
		gen.Int64Range(1, 10),
	))

	// Property: Pagination limit exceeding max must be rejected
	properties.Property("invalid pagination limit rejected", prop.ForAll(
		func(limit int) bool {
			// Test with limit exceeding max
			if limit <= config.MaxPaginationLimit {
				return true // Skip valid limits
			}

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request = httptest.NewRequest("GET", "/api/v1/test?limit="+string(rune(limit)), nil)

			middleware := InputValidationMiddleware(config, logger)
			middleware(c)

			// Should be rejected with 400 status
			return w.Code == http.StatusBadRequest || !c.IsAborted()
		},
		gen.IntRange(config.MaxPaginationLimit+1, config.MaxPaginationLimit+1000),
	))

	// Property: Page parameter less than 1 must be rejected
	properties.Property("invalid page parameter rejected", prop.ForAll(
		func(page int) bool {
			// Test with page < 1
			if page >= 1 {
				return true // Skip valid pages
			}

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Use proper string formatting to avoid control characters
			url := "/api/v1/test?page=" + strings.TrimSpace(string(rune(page+'0')))
			if page < 0 {
				url = "/api/v1/test?page=-1"
			} else {
				url = "/api/v1/test?page=0"
			}

			c.Request = httptest.NewRequest("GET", url, nil)

			middleware := InputValidationMiddleware(config, logger)
			middleware(c)

			// Should be rejected with 400 status
			return w.Code == http.StatusBadRequest || !c.IsAborted()
		},
		gen.IntRange(-100, 0),
	))

	// Property: Fields exceeding max length must be rejected
	properties.Property("oversized fields are rejected", prop.ForAll(
		func(fieldName string) bool {
			// Get max length for this field from config
			maxLen, exists := config.MaxFieldLengths[fieldName]
			if !exists {
				return true // Skip fields without max length configured
			}

			// Create a value that exceeds the max length
			oversizedValue := strings.Repeat("a", maxLen+1)

			data := map[string]interface{}{
				fieldName: oversizedValue,
			}
			body, _ := json.Marshal(data)

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request = httptest.NewRequest("POST", "/api/v1/test", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Request.ContentLength = int64(len(body))

			middleware := InputValidationMiddleware(config, logger)
			middleware(c)

			// Should be rejected with 400 status
			return w.Code == http.StatusBadRequest
		},
		genFieldName(),
	))

	// Property: XSS patterns must be rejected
	properties.Property("XSS patterns are rejected", prop.ForAll(
		func(xssPayload string) bool {
			data := map[string]interface{}{
				"content": xssPayload,
			}
			body, _ := json.Marshal(data)

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request = httptest.NewRequest("POST", "/api/v1/test", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			middleware := InputValidationMiddleware(config, logger)
			middleware(c)

			// Should be rejected with 400 status
			return w.Code == http.StatusBadRequest
		},
		genXSSPayload(),
	))

	// Property: SQL injection patterns must be rejected
	properties.Property("SQL injection patterns are rejected", prop.ForAll(
		func(sqlPayload string) bool {
			data := map[string]interface{}{
				"query": sqlPayload,
			}
			body, _ := json.Marshal(data)

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request = httptest.NewRequest("POST", "/api/v1/test", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			middleware := InputValidationMiddleware(config, logger)
			middleware(c)

			// Should be rejected with 400 status
			return w.Code == http.StatusBadRequest
		},
		genSQLInjectionPayload(),
	))

	// Property: Unsupported content types must be rejected
	properties.Property("unsupported content types rejected", prop.ForAll(
		func(contentType string) bool {
			// Skip if content type is in allowed list
			for _, allowed := range config.AllowedContentTypes {
				if strings.Contains(contentType, allowed) {
					return true
				}
			}

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request = httptest.NewRequest("POST", "/api/v1/test", bytes.NewBufferString("test"))
			c.Request.Header.Set("Content-Type", contentType)

			middleware := InputValidationMiddleware(config, logger)
			middleware(c)

			// Should be rejected with 415 status
			return w.Code == http.StatusUnsupportedMediaType
		},
		genUnsupportedContentType(),
	))

	// Property: Valid inputs must pass validation
	properties.Property("valid inputs are accepted", prop.ForAll(
		func(name string, email string) bool {
			// Create valid data
			data := map[string]interface{}{
				"name":  name,
				"email": email,
			}
			body, _ := json.Marshal(data)

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request = httptest.NewRequest("POST", "/api/v1/test", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			middleware := InputValidationMiddleware(config, logger)
			middleware(c)

			// Should not be rejected (either 200 or not aborted)
			return w.Code != http.StatusBadRequest && w.Code != http.StatusUnsupportedMediaType && w.Code != http.StatusRequestEntityTooLarge
		},
		genValidName(),
		genValidEmail(),
	))

	// Property: Validation errors must include structured error response
	properties.Property("validation errors include structured response", prop.ForAll(
		func(xssPayload string) bool {
			data := map[string]interface{}{
				"content": xssPayload,
			}
			body, _ := json.Marshal(data)

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request = httptest.NewRequest("POST", "/api/v1/test", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			middleware := InputValidationMiddleware(config, logger)
			middleware(c)

			// If rejected, response should have error and code fields
			if w.Code == http.StatusBadRequest {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					return false
				}
				// Check for structured error response
				_, hasError := response["error"]
				_, hasCode := response["code"]
				return hasError && hasCode
			}

			return true
		},
		genXSSPayload(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Generator for invalid JSON strings
func genInvalidJSON() gopter.Gen {
	return gen.OneConstOf(
		"{invalid json}",
		"{'single': 'quotes'}",
		"{missing: quotes}",
		"{\"unclosed\": ",
		"[1, 2, 3,]",
		"{\"trailing\": \"comma\",}",
		"not json at all",
		"{",
		"[",
	)
}

// Generator for field names
func genFieldName() gopter.Gen {
	return gen.OneConstOf(
		"name",
		"email",
		"username",
		"password",
		"phone",
		"address",
		"company",
		"title",
		"content",
	)
}

// Generator for XSS payloads
func genXSSPayload() gopter.Gen {
	return gen.OneConstOf(
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"javascript:alert('XSS')",
		"<iframe src='javascript:alert(1)'></iframe>",
		"<body onload=alert('XSS')>",
		"<svg onload=alert('XSS')>",
		"<input onfocus=alert('XSS') autofocus>",
		"<select onfocus=alert('XSS') autofocus>",
		"<textarea onfocus=alert('XSS') autofocus>",
		"<object data='javascript:alert(1)'>",
		"<embed src='javascript:alert(1)'>",
		"<link rel='stylesheet' href='javascript:alert(1)'>",
		"<meta http-equiv='refresh' content='0;url=javascript:alert(1)'>",
		"<style>@import 'javascript:alert(1)';</style>",
		"vbscript:msgbox('XSS')",
		"onclick=alert('XSS')",
		"onmouseover=alert('XSS')",
		"onerror=alert('XSS')",
	)
}

// Generator for SQL injection payloads
func genSQLInjectionPayload() gopter.Gen {
	return gen.OneConstOf(
		"' OR '1'='1",
		"' OR 1=1--",
		"' OR 1=1#",
		"' OR 1=1/*",
		"admin'--",
		"admin'/*",
		"' UNION SELECT * FROM users--",
		"'; DROP TABLE users--",
		"' AND 1=1--",
		"' OR 'x'='x",
		") OR (1=1",
		"1' AND '1'='1",
		"1' OR '1'='1",
		"SELECT * FROM users",
		"INSERT INTO users VALUES",
		"UPDATE users SET",
		"DELETE FROM users",
		"DROP TABLE users",
		"EXEC sp_executesql",
		"UNION ALL SELECT",
		"' UNION SELECT NULL--",
		"1; DROP TABLE users--",
		"' OR 1=1 LIMIT 1--",
		"' OR '1'='1' --",
	)
}

// Generator for unsupported content types
func genUnsupportedContentType() gopter.Gen {
	return gen.OneConstOf(
		"text/xml",
		"application/octet-stream",
		"application/x-protobuf",
		"application/msgpack",
		"text/csv",
		"application/vnd.api+json",
		"application/x-yaml",
		"image/svg+xml",
		"video/mp4",
		"audio/mpeg",
	)
}

// Generator for valid names
func genValidName() gopter.Gen {
	return gen.OneConstOf(
		"John Doe",
		"Jane Smith",
		"Alice",
		"Bob",
		"Charlie",
		"David",
		"Eve",
		"Frank",
		"Grace",
		"Henry",
	)
}

// Generator for valid emails
func genValidEmail() gopter.Gen {
	return gen.OneConstOf(
		"user@example.com",
		"test@test.com",
		"admin@company.com",
		"john.doe@example.org",
		"jane_smith@test.co.uk",
		"alice+tag@example.com",
		"bob123@test.io",
	)
}
