package security

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidatePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "security_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get absolute path of temp dir
	_, err = filepath.Abs(tempDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		allowedRoot string
		expectErr   bool
	}{
		{
			name:        "Valid path within root",
			path:        filepath.Join(tempDir, "file.txt"),
			allowedRoot: tempDir,
			expectErr:   false,
		},
		{
			name:        "Valid nested path within root",
			path:        filepath.Join(tempDir, "subdir", "file.txt"),
			allowedRoot: tempDir,
			expectErr:   false,
		},
		{
			name:        "Root path itself is valid",
			path:        tempDir,
			allowedRoot: tempDir,
			expectErr:   false,
		},
		{
			name:        "Relative path within root",
			path:        "subdir/file.txt",
			allowedRoot: tempDir,
			expectErr:   false,
		},
		{
			name:        "Path traversal attempt",
			path:        filepath.Join(tempDir, "..", "outside.txt"),
			allowedRoot: tempDir,
			expectErr:   true,
		},
		{
			name:        "Path outside root directory",
			path:        "/etc/passwd",
			allowedRoot: tempDir,
			expectErr:   true,
		},
		{
			name:        "Empty path",
			path:        "",
			allowedRoot: tempDir,
			expectErr:   true,
		},
		{
			name:        "Empty allowed root",
			path:        filepath.Join(tempDir, "file.txt"),
			allowedRoot: "",
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path, tt.allowedRoot)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidatePath() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePath(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePath() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsPathInAllowedDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "security_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file in the temp directory
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		allowedRoot string
		expected    bool
	}{
		{
			name:        "File within allowed directory",
			path:        testFile,
			allowedRoot: tempDir,
			expected:    true,
		},
		{
			name:        "Directory within allowed directory",
			path:        filepath.Join(tempDir, "subdir"),
			allowedRoot: tempDir,
			expected:    true,
		},
		{
			name:        "File outside allowed directory",
			path:        "/etc/passwd",
			allowedRoot: tempDir,
			expected:    false,
		},
		{
			name:        "Path traversal attempt",
			path:        filepath.Join(tempDir, "..", "outside.txt"),
			allowedRoot: tempDir,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPathInAllowedDirectory(tt.path, tt.allowedRoot)
			if result != tt.expected {
				t.Errorf("IsPathInAllowedDirectory() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestValidatePathWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific tests on non-Windows OS")
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "security_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get absolute path of temp dir
	_, err = filepath.Abs(tempDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		allowedRoot string
		expectErr   bool
	}{
		{
			name:        "Valid Windows path",
			path:        filepath.Join(tempDir, "file.txt"),
			allowedRoot: tempDir,
			expectErr:   false,
		},
		{
			name:        "Path with drive letter",
			path:        "C:\\Windows\\System32",
			allowedRoot: tempDir,
			expectErr:   true,
		},
		{
			name:        "Windows path traversal attempt",
			path:        filepath.Join(tempDir, "..", "outside.txt"),
			allowedRoot: tempDir,
			expectErr:   true,
		},
		{
			name:        "UNC path attempt",
			path:        `\\server\share\file.txt`,
			allowedRoot: tempDir,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path, tt.allowedRoot)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidatePath() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}
