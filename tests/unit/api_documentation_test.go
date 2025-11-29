package unit

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// OpenAPISpec represents the structure of an OpenAPI 3.0 specification
type OpenAPISpec struct {
	OpenAPI    string                 `yaml:"openapi" json:"openapi"`
	Info       OpenAPIInfo            `yaml:"info" json:"info"`
	Servers    []OpenAPIServer        `yaml:"servers" json:"servers"`
	Paths      map[string]interface{} `yaml:"paths" json:"paths"`
	Components OpenAPIComponents      `yaml:"components" json:"components"`
}

type OpenAPIInfo struct {
	Title       string `yaml:"title" json:"title"`
	Description string `yaml:"description" json:"description"`
	Version     string `yaml:"version" json:"version"`
}

type OpenAPIServer struct {
	URL         string `yaml:"url" json:"url"`
	Description string `yaml:"description" json:"description"`
}

type OpenAPIComponents struct {
	Schemas   map[string]interface{} `yaml:"schemas" json:"schemas"`
	Responses map[string]interface{} `yaml:"responses" json:"responses"`
}

// SwaggerSpec represents the structure of a Swagger 2.0 specification
type SwaggerSpec struct {
	Swagger     string                 `yaml:"swagger" json:"swagger"`
	Info        SwaggerInfo            `yaml:"info" json:"info"`
	Host        string                 `yaml:"host" json:"host"`
	BasePath    string                 `yaml:"basePath" json:"basePath"`
	Schemes     []string               `yaml:"schemes" json:"schemes"`
	Paths       map[string]interface{} `yaml:"paths" json:"paths"`
	Definitions map[string]interface{} `yaml:"definitions" json:"definitions"`
}

type SwaggerInfo struct {
	Title       string `yaml:"title" json:"title"`
	Description string `yaml:"description" json:"description"`
	Version     string `yaml:"version" json:"version"`
}

// TestOpenAPISpecValidity tests that the OpenAPI spec file is valid
func TestOpenAPISpecValidity(t *testing.T) {
	// Read the OpenAPI spec file
	data, err := os.ReadFile("../../docs/openapi.yaml")
	require.NoError(t, err, "OpenAPI spec file should exist and be readable")

	// Parse the YAML
	var spec OpenAPISpec
	err = yaml.Unmarshal(data, &spec)
	require.NoError(t, err, "OpenAPI spec should be valid YAML")

	// Validate OpenAPI version
	assert.Equal(t, "3.0.3", spec.OpenAPI, "OpenAPI version should be 3.0.3")

	// Validate required info fields
	assert.NotEmpty(t, spec.Info.Title, "API title should not be empty")
	assert.NotEmpty(t, spec.Info.Description, "API description should not be empty")
	assert.NotEmpty(t, spec.Info.Version, "API version should not be empty")

	// Validate servers are defined
	assert.NotEmpty(t, spec.Servers, "At least one server should be defined")

	// Validate paths are defined
	assert.NotEmpty(t, spec.Paths, "API paths should be defined")

	// Validate components are defined
	assert.NotEmpty(t, spec.Components.Schemas, "Component schemas should be defined")
}

// TestSwaggerSpecValidity tests that the Swagger spec file is valid
func TestSwaggerSpecValidity(t *testing.T) {
	// Read the Swagger spec file
	data, err := os.ReadFile("../../docs/swagger.yaml")
	require.NoError(t, err, "Swagger spec file should exist and be readable")

	// Parse the YAML
	var spec SwaggerSpec
	err = yaml.Unmarshal(data, &spec)
	require.NoError(t, err, "Swagger spec should be valid YAML")

	// Validate Swagger version
	assert.Equal(t, "2.0", spec.Swagger, "Swagger version should be 2.0")

	// Validate required info fields
	assert.NotEmpty(t, spec.Info.Title, "API title should not be empty")
	assert.NotEmpty(t, spec.Info.Description, "API description should not be empty")
	assert.NotEmpty(t, spec.Info.Version, "API version should not be empty")

	// Validate base path
	assert.NotEmpty(t, spec.BasePath, "Base path should be defined")

	// Validate paths are defined
	assert.NotEmpty(t, spec.Paths, "API paths should be defined")

	// Validate definitions are defined
	assert.NotEmpty(t, spec.Definitions, "Definitions should be defined")
}

// TestAllEndpointsDocumented verifies that all critical endpoints are documented
func TestAllEndpointsDocumented(t *testing.T) {
	// Read the OpenAPI spec
	data, err := os.ReadFile("../../docs/openapi.yaml")
	require.NoError(t, err, "OpenAPI spec file should exist")

	var spec OpenAPISpec
	err = yaml.Unmarshal(data, &spec)
	require.NoError(t, err, "OpenAPI spec should be valid")

	// Define critical endpoints that must be documented
	criticalEndpoints := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/logout",
		"/api/v1/auth/refresh",
		"/api/v1/users",
		"/api/v1/users/{id}",
		"/api/v1/products",
		"/api/v1/products/{id}",
		"/api/v1/categories",
		"/api/v1/orders",
		"/api/v1/orders/{id}",
	}

	// Check each critical endpoint is documented
	for _, endpoint := range criticalEndpoints {
		t.Run(fmt.Sprintf("Endpoint_%s", strings.ReplaceAll(endpoint, "/", "_")), func(t *testing.T) {
			_, exists := spec.Paths[endpoint]
			assert.True(t, exists, "Endpoint %s should be documented in OpenAPI spec", endpoint)
		})
	}
}

// TestErrorCodesDocumented verifies that all error codes are documented
func TestErrorCodesDocumented(t *testing.T) {
	// Read the error codes documentation
	data, err := os.ReadFile("../../docs/ERROR_CODES.md")
	require.NoError(t, err, "ERROR_CODES.md file should exist")

	errorCodesDoc := string(data)

	// Define all error codes that should be documented (from requirements)
	requiredErrorCodes := []string{
		"VALIDATION_ERROR",
		"INVALID_REQUEST",
		"UNAUTHORIZED",
		"TOKEN_EXPIRED",
		"INVALID_CREDENTIALS",
		"FORBIDDEN",
		"ACCOUNT_LOCKED",
		"NOT_FOUND",
		"CONFLICT",
		"DUPLICATE_EMAIL",
		"DUPLICATE_SKU",
		"RATE_LIMIT_EXCEEDED",
		"INTERNAL_ERROR",
		"SERVICE_UNAVAILABLE",
	}

	// Check each error code is documented
	for _, errorCode := range requiredErrorCodes {
		t.Run(fmt.Sprintf("ErrorCode_%s", errorCode), func(t *testing.T) {
			assert.Contains(t, errorCodesDoc, errorCode,
				"Error code %s should be documented in ERROR_CODES.md", errorCode)
		})
	}
}

// TestHTTPStatusCodesDocumented verifies that all HTTP status codes are documented
func TestHTTPStatusCodesDocumented(t *testing.T) {
	// Read the error codes documentation
	data, err := os.ReadFile("../../docs/ERROR_CODES.md")
	require.NoError(t, err, "ERROR_CODES.md file should exist")

	errorCodesDoc := string(data)

	// Define all HTTP status codes that should be documented
	requiredStatusCodes := []struct {
		code        string
		description string
	}{
		{"200", "OK"},
		{"201", "Created"},
		{"204", "No Content"},
		{"400", "Bad Request"},
		{"401", "Unauthorized"},
		{"403", "Forbidden"},
		{"404", "Not Found"},
		{"409", "Conflict"},
		{"422", "Unprocessable Entity"},
		{"429", "Too Many Requests"},
		{"500", "Internal Server Error"},
		{"503", "Service Unavailable"},
	}

	// Check each status code is documented
	for _, statusCode := range requiredStatusCodes {
		t.Run(fmt.Sprintf("StatusCode_%s", statusCode.code), func(t *testing.T) {
			assert.Contains(t, errorCodesDoc, statusCode.code,
				"HTTP status code %s should be documented", statusCode.code)
		})
	}
}

// TestAuthenticationDocumented verifies authentication is properly documented
func TestAuthenticationDocumented(t *testing.T) {
	// Read the OpenAPI spec
	data, err := os.ReadFile("../../docs/openapi.yaml")
	require.NoError(t, err, "OpenAPI spec file should exist")

	var spec OpenAPISpec
	err = yaml.Unmarshal(data, &spec)
	require.NoError(t, err, "OpenAPI spec should be valid")

	// Check that authentication is mentioned in the description
	assert.Contains(t, spec.Info.Description, "Authentication",
		"API description should mention authentication")
	assert.Contains(t, spec.Info.Description, "JWT",
		"API description should mention JWT tokens")
}

// TestRateLimitingDocumented verifies rate limiting is documented
func TestRateLimitingDocumented(t *testing.T) {
	// Read the OpenAPI spec
	data, err := os.ReadFile("../../docs/openapi.yaml")
	require.NoError(t, err, "OpenAPI spec file should exist")

	specContent := string(data)

	// Check that rate limiting is documented
	assert.Contains(t, specContent, "Rate Limit",
		"API spec should document rate limiting")

	// Read the error codes documentation
	errorCodesData, err := os.ReadFile("../../docs/ERROR_CODES.md")
	require.NoError(t, err, "ERROR_CODES.md file should exist")

	errorCodesDoc := string(errorCodesData)

	// Check rate limiting details in error codes
	assert.Contains(t, errorCodesDoc, "RATE_LIMIT_EXCEEDED",
		"Error codes should document rate limit exceeded error")
	assert.Contains(t, errorCodesDoc, "X-RateLimit",
		"Error codes should document rate limit headers")
}

// TestPaginationDocumented verifies pagination is documented
func TestPaginationDocumented(t *testing.T) {
	// Read the OpenAPI spec
	data, err := os.ReadFile("../../docs/openapi.yaml")
	require.NoError(t, err, "OpenAPI spec file should exist")

	specContent := string(data)

	// Check that pagination parameters are documented
	paginationParams := []string{"page", "limit"}
	for _, param := range paginationParams {
		assert.Contains(t, specContent, param,
			"API spec should document pagination parameter: %s", param)
	}
}

// TestErrorResponseFormatDocumented verifies error response format is documented
func TestErrorResponseFormatDocumented(t *testing.T) {
	// Read the error codes documentation
	data, err := os.ReadFile("../../docs/ERROR_CODES.md")
	require.NoError(t, err, "ERROR_CODES.md file should exist")

	errorCodesDoc := string(data)

	// Check that error response format is documented
	assert.Contains(t, errorCodesDoc, "Error Response Format",
		"Error codes should document error response format")
	assert.Contains(t, errorCodesDoc, `"error"`,
		"Error response format should include 'error' field")
	assert.Contains(t, errorCodesDoc, `"code"`,
		"Error response format should include 'code' field")
	assert.Contains(t, errorCodesDoc, `"details"`,
		"Error response format should include 'details' field")
}

// TestValidationErrorsDocumented verifies validation errors are documented
func TestValidationErrorsDocumented(t *testing.T) {
	// Read the error codes documentation
	data, err := os.ReadFile("../../docs/ERROR_CODES.md")
	require.NoError(t, err, "ERROR_CODES.md file should exist")

	errorCodesDoc := string(data)

	// Check that validation errors are documented
	assert.Contains(t, errorCodesDoc, "Validation Error",
		"Error codes should document validation errors")
	assert.Contains(t, errorCodesDoc, "VALIDATION_ERROR",
		"Error codes should include VALIDATION_ERROR code")
}

// TestSecurityBestPracticesDocumented verifies security best practices are documented
func TestSecurityBestPracticesDocumented(t *testing.T) {
	// Read the OpenAPI spec
	data, err := os.ReadFile("../../docs/openapi.yaml")
	require.NoError(t, err, "OpenAPI spec file should exist")

	specContent := string(data)

	// Check that security best practices are mentioned
	securityTopics := []string{"HTTPS", "token"}
	for _, topic := range securityTopics {
		assert.Contains(t, strings.ToLower(specContent), strings.ToLower(topic),
			"API spec should mention security topic: %s", topic)
	}
}

// TestSwaggerJSONValidity tests that the swagger.json file is valid JSON
func TestSwaggerJSONValidity(t *testing.T) {
	// Read the swagger.json file
	data, err := os.ReadFile("../../docs/swagger.json")
	require.NoError(t, err, "swagger.json file should exist and be readable")

	// Parse the JSON
	var spec map[string]interface{}
	err = json.Unmarshal(data, &spec)
	require.NoError(t, err, "swagger.json should be valid JSON")

	// Validate required fields
	assert.NotNil(t, spec["swagger"], "swagger field should be present")
	assert.NotNil(t, spec["info"], "info field should be present")
	assert.NotNil(t, spec["paths"], "paths field should be present")
}

// TestAPIVersionConsistency verifies API version is consistent across documentation
func TestAPIVersionConsistency(t *testing.T) {
	// Read OpenAPI spec
	openAPIData, err := os.ReadFile("../../docs/openapi.yaml")
	require.NoError(t, err, "OpenAPI spec file should exist")

	var openAPISpec OpenAPISpec
	err = yaml.Unmarshal(openAPIData, &openAPISpec)
	require.NoError(t, err, "OpenAPI spec should be valid")

	// Read Swagger spec
	swaggerData, err := os.ReadFile("../../docs/swagger.yaml")
	require.NoError(t, err, "Swagger spec file should exist")

	var swaggerSpec SwaggerSpec
	err = yaml.Unmarshal(swaggerData, &swaggerSpec)
	require.NoError(t, err, "Swagger spec should be valid")

	// Verify versions match
	assert.Equal(t, openAPISpec.Info.Version, swaggerSpec.Info.Version,
		"API version should be consistent across OpenAPI and Swagger specs")
}

// TestEndpointResponseCodesDocumented verifies that endpoints document their response codes
func TestEndpointResponseCodesDocumented(t *testing.T) {
	// Read the OpenAPI spec
	data, err := os.ReadFile("../../docs/openapi.yaml")
	require.NoError(t, err, "OpenAPI spec file should exist")

	var spec map[string]interface{}
	err = yaml.Unmarshal(data, &spec)
	require.NoError(t, err, "OpenAPI spec should be valid")

	paths, ok := spec["paths"].(map[string]interface{})
	require.True(t, ok, "Paths should be a map")

	// Check that at least some endpoints have response codes documented
	endpointsWithResponses := 0
	for path, pathItem := range paths {
		pathMap, ok := pathItem.(map[string]interface{})
		if !ok {
			continue
		}

		for method, operation := range pathMap {
			if method == "parameters" {
				continue
			}

			operationMap, ok := operation.(map[string]interface{})
			if !ok {
				continue
			}

			responses, ok := operationMap["responses"].(map[string]interface{})
			if ok && len(responses) > 0 {
				endpointsWithResponses++

				// Verify common response codes are present
				t.Run(fmt.Sprintf("%s_%s_has_responses", path, method), func(t *testing.T) {
					assert.NotEmpty(t, responses, "Endpoint %s %s should have response codes documented", method, path)
				})
			}
		}
	}

	assert.Greater(t, endpointsWithResponses, 0, "At least some endpoints should have response codes documented")
}

// TestCommonErrorCodesInSpec verifies common error codes are referenced in the spec
func TestCommonErrorCodesInSpec(t *testing.T) {
	// Read the OpenAPI spec
	data, err := os.ReadFile("../../docs/openapi.yaml")
	require.NoError(t, err, "OpenAPI spec file should exist")

	specContent := string(data)

	// Common HTTP status codes that should appear in responses
	// Check for both quoted and unquoted formats
	commonStatusCodes := []string{"200", "201", "400", "401", "404"}

	for _, statusCode := range commonStatusCodes {
		t.Run(fmt.Sprintf("StatusCode_%s_in_spec", statusCode), func(t *testing.T) {
			// Check if status code appears in either format: '200' or "200"
			hasQuoted := strings.Contains(specContent, fmt.Sprintf("'%s'", statusCode))
			hasDoubleQuoted := strings.Contains(specContent, fmt.Sprintf("\"%s\"", statusCode))
			assert.True(t, hasQuoted || hasDoubleQuoted,
				"Status code %s should be referenced in OpenAPI spec", statusCode)
		})
	}
}
