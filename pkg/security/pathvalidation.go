package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath validates that a path is safe to access within an allowed root directory.
// It prevents path traversal attacks by checking that the resolved path is within the allowed root.
func ValidatePath(path, allowedRoot string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if allowedRoot == "" {
		return fmt.Errorf("allowed root cannot be empty")
	}

	// Clean both paths to remove any unnecessary elements
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(allowedRoot)

	// For relative paths, join them with the root directory first
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(cleanRoot, cleanPath)
	}

	// Convert to absolute paths
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", cleanPath, err)
	}

	absRoot, err := filepath.Abs(cleanRoot)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for root %s: %w", cleanRoot, err)
	}

	// Check if absolute path is within the allowed root
	if !strings.HasPrefix(absPath, absRoot+string(os.PathSeparator)) && absPath != absRoot {
		return fmt.Errorf("path %s is outside allowed root %s", absPath, absRoot)
	}

	return nil
}

// SanitizePath sanitizes a path by removing any potentially dangerous elements
// and normalizing it for safe use.
func SanitizePath(path string) string {
	// Remove any null bytes
	path = strings.ReplaceAll(path, "\x00", "")

	// Clean the path to remove any directory traversal attempts
	path = filepath.Clean(path)

	// Convert forward slashes to the correct separator for the current OS
	path = filepath.FromSlash(path)

	return path
}

// IsPathInAllowedDirectory checks if a path is within an allowed directory.
// This is a simpler version of ValidatePath that doesn't return an error.
func IsPathInAllowedDirectory(path, allowedRoot string) bool {
	err := ValidatePath(path, allowedRoot)
	return err == nil
}
