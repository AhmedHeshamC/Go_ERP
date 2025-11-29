package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"erpgo/pkg/security"
)

// TestPathTraversalProtection tests that the path validation utility prevents
// path traversal attacks in various scenarios
func TestPathTraversalProtection(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "path_traversal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get absolute path of temp dir
	absTempDir, err := filepath.Abs(tempDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Test cases for path traversal attempts
	testCases := []struct {
		name          string
		path          string
		shouldBeValid bool
	}{
		{
			name:          "Simple dot dot slash attack",
			path:          filepath.Join(tempDir, "..", "etc", "passwd"),
			shouldBeValid: false,
		},
		{
			name:          "Multiple dot dot slash attacks",
			path:          filepath.Join(tempDir, "..", "..", "..", "root", ".ssh", "authorized_keys"),
			shouldBeValid: false,
		},
		{
			name:          "Encoded path traversal",
			path:          filepath.Join(tempDir, "%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd"),
			shouldBeValid: false,
		},
		{
			name:          "Null byte injection",
			path:          filepath.Join(tempDir, "file\x00.txt"),
			shouldBeValid: false,
		},
		{
			name:          "Forward slash traversal",
			path:          filepath.Join(tempDir, "..", "..", "var", "log"),
			shouldBeValid: false,
		},
		{
			name:          "Valid subdirectory path",
			path:          filepath.Join(tempDir, "subdir", "file.txt"),
			shouldBeValid: true,
		},
		{
			name:          "Valid file path",
			path:          filepath.Join(tempDir, "file.txt"),
			shouldBeValid: true,
		},
		{
			name:          "Absolute path within allowed directory",
			path:          absTempDir + string(filepath.Separator) + "file.txt",
			shouldBeValid: true,
		},
		{
			name:          "Relative path within allowed directory",
			path:          "subdir/file.txt",
			shouldBeValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := security.ValidatePath(tc.path, tempDir)

			if tc.shouldBeValid && err != nil {
				t.Errorf("Expected path to be valid, but got error: %v", err)
			}

			if !tc.shouldBeValid && err == nil {
				t.Errorf("Expected path to be invalid, but validation passed")
			}

			// Check error message for invalid paths
			if !tc.shouldBeValid && err != nil {
				errMsg := err.Error()
				if !strings.Contains(errMsg, "outside allowed root") &&
					!strings.Contains(errMsg, "cannot be empty") &&
					!strings.Contains(errMsg, "failed to get absolute path") &&
					!strings.Contains(errMsg, "contains null bytes") &&
					!strings.Contains(errMsg, "contains encoded traversal sequence") {
					t.Errorf("Expected descriptive error message for invalid path, got: %s", errMsg)
				}
			}
		})
	}
}

// TestSanitizePath tests path sanitization functionality
func TestSanitizePath(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal path",
			input:    "/normal/path",
			expected: filepath.FromSlash("/normal/path"),
		},
		{
			name:     "Path with null bytes",
			input:    "/path\x00with\x00nulls",
			expected: filepath.FromSlash("/pathwithnulls"),
		},
		{
			name:     "Path with directory traversal",
			input:    "/path/../traversal",
			expected: "/traversal",
		},
		{
			name:     "Path with multiple separators",
			input:    "/path//with///multiple/separators",
			expected: filepath.FromSlash("/path/with/multiple/separators"),
		},
		{
			name:     "Relative path",
			input:    "relative/path",
			expected: "relative/path",
		},
		{
			name:     "Empty path",
			input:    "",
			expected: ".",
		},
		{
			name:     "Root path",
			input:    "/",
			expected: string(filepath.Separator),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := security.SanitizePath(tc.input)
			if result != tc.expected {
				t.Errorf("SanitizePath(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestIsPathInAllowedDirectory tests the convenience function
func TestIsPathInAllowedDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "allowed_dir_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file in the temp directory
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "File within allowed directory",
			path:     testFile,
			expected: true,
		},
		{
			name:     "Directory within allowed directory",
			path:     filepath.Join(tempDir, "subdir"),
			expected: true,
		},
		{
			name:     "File outside allowed directory",
			path:     "/etc/passwd",
			expected: false,
		},
		{
			name:     "Path traversal attempt",
			path:     filepath.Join(tempDir, "..", "outside.txt"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := security.IsPathInAllowedDirectory(tc.path, tempDir)
			if result != tc.expected {
				t.Errorf("IsPathInAllowedDirectory(%q, %q) = %v, expected %v", tc.path, tempDir, result, tc.expected)
			}
		})
	}
}
