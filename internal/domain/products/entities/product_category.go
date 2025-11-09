package entities

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ProductCategory represents a product category with hierarchical structure
type ProductCategory struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Name          string     `json:"name" db:"name"`
	Description   string     `json:"description" db:"description"`
	ParentID      *uuid.UUID `json:"parent_id" db:"parent_id"`
	Level         int        `json:"level" db:"level"`
	Path          string     `json:"path" db:"path"`
	ImageURL      string     `json:"image_url" db:"image_url"`
	SortOrder     int        `json:"sort_order" db:"sort_order"`
	IsActive      bool       `json:"is_active" db:"is_active"`
	SEOTitle       string     `json:"seo_title" db:"seo_title"`
	SEODescription string     `json:"seo_description" db:"seo_description"`
	SEOKeywords   string     `json:"seo_keywords" db:"seo_keywords"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// Validate validates the product category entity
func (pc *ProductCategory) Validate() error {
	var errs []error

	// Validate UUID
	if pc.ID == uuid.Nil {
		errs = append(errs, errors.New("category ID cannot be empty"))
	}

	// Validate name
	if err := pc.validateName(); err != nil {
		errs = append(errs, fmt.Errorf("invalid name: %w", err))
	}

	// Validate description
	if err := pc.validateDescription(); err != nil {
		errs = append(errs, fmt.Errorf("invalid description: %w", err))
	}

	// Validate level
	if pc.Level < 0 {
		errs = append(errs, errors.New("category level cannot be negative"))
	}

	// Validate path
	if err := pc.validatePath(); err != nil {
		errs = append(errs, fmt.Errorf("invalid path: %w", err))
	}

	// Validate image URL (optional)
	if pc.ImageURL != "" {
		if err := pc.validateImageURL(); err != nil {
			errs = append(errs, fmt.Errorf("invalid image URL: %w", err))
		}
	}

	// Validate sort order
	if pc.SortOrder < 0 {
		errs = append(errs, errors.New("sort order cannot be negative"))
	}

	// Validate SEO fields (optional)
	if err := pc.validateSEOFields(); err != nil {
		errs = append(errs, fmt.Errorf("invalid SEO fields: %w", err))
	}

	// Validate parent-child relationship
	if pc.ParentID != nil {
		if *pc.ParentID == uuid.Nil {
			errs = append(errs, errors.New("parent ID cannot be empty when provided"))
		}

		// Parent cannot be the same as the category itself
		if pc.ParentID != nil && *pc.ParentID == pc.ID {
			errs = append(errs, errors.New("category cannot be its own parent"))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errors.Join(errs...))
	}

	return nil
}

// validateName validates the category name
func (pc *ProductCategory) validateName() error {
	name := strings.TrimSpace(pc.Name)
	if name == "" {
		return errors.New("category name cannot be empty")
	}

	if len(name) > 200 {
		return errors.New("category name cannot exceed 200 characters")
	}

	// Allow letters, numbers, spaces, hyphens, underscores, and forward slashes
	nameRegex := regexp.MustCompile(`^[a-zA-Z0-9\s\-_/]+$`)
	if !nameRegex.MatchString(name) {
		return errors.New("category name can only contain letters, numbers, spaces, hyphens, underscores, and forward slashes")
	}

	return nil
}

// validateDescription validates the category description
func (pc *ProductCategory) validateDescription() error {
	desc := strings.TrimSpace(pc.Description)
	if desc != "" && len(desc) > 1000 {
		return errors.New("category description cannot exceed 1000 characters")
	}
	return nil
}

// validatePath validates the category path
func (pc *ProductCategory) validatePath() error {
	path := strings.TrimSpace(pc.Path)
	if path == "" && pc.ParentID == nil {
		// Root category should have a path
		return errors.New("root category path cannot be empty")
	}

	if path != "" {
		// Path should be in format like "/electronics/computers/laptops"
		if !strings.HasPrefix(path, "/") {
			return errors.New("path must start with forward slash")
		}

		if strings.HasSuffix(path, "/") {
			return errors.New("path cannot end with forward slash")
		}

		// Validate path segments
		segments := strings.Split(strings.Trim(path, "/"), "/")
		for _, segment := range segments {
			if strings.TrimSpace(segment) == "" {
				return errors.New("path cannot contain empty segments")
			}

			segmentRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
			if !segmentRegex.MatchString(segment) {
				return errors.New("path segments can only contain letters, numbers, hyphens, and underscores")
			}
		}

		// Path length validation
		if len(path) > 500 {
			return errors.New("path cannot exceed 500 characters")
		}
	}

	return nil
}

// validateImageURL validates the image URL
func (pc *ProductCategory) validateImageURL() error {
	url := strings.TrimSpace(pc.ImageURL)
	if url == "" {
		return nil // URL is optional
	}

	if len(url) > 500 {
		return errors.New("image URL cannot exceed 500 characters")
	}

	// Basic URL regex pattern - simplified for practical use
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(url) {
		return errors.New("invalid image URL format")
	}

	// Check for common image file extensions
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg"}
	hasValidExtension := false
	for _, ext := range imageExtensions {
		if strings.HasSuffix(strings.ToLower(url), ext) {
			hasValidExtension = true
			break
		}
	}

	if !hasValidExtension {
		return errors.New("image URL must end with a valid image extension (.jpg, .jpeg, .png, .gif, .webp, .svg)")
	}

	return nil
}

// IsRootCategory returns true if this is a root category (no parent)
func (pc *ProductCategory) IsRootCategory() bool {
	return pc.ParentID == nil
}

// IsChildOf returns true if this category is a child of the given category
func (pc *ProductCategory) IsChildOf(parentID uuid.UUID) bool {
	return pc.ParentID != nil && *pc.ParentID == parentID
}

// GetDepth returns the depth level in the category tree
func (pc *ProductCategory) GetDepth() int {
	return pc.Level
}

// CanHaveChildren returns true if this category can have child categories
func (pc *ProductCategory) CanHaveChildren() bool {
	// Most categories can have children, but you could add business logic here
	// For example, limiting maximum depth
	return pc.Level < 5 // Maximum depth of 5 levels
}

// BuildPath builds the category path based on name and parent path
func (pc *ProductCategory) BuildPath(parentPath string) {
	if pc.ParentID == nil {
		// Root category
		pc.Path = "/" + strings.ToLower(strings.ReplaceAll(strings.TrimSpace(pc.Name), " ", "-"))
		pc.Level = 0
	} else {
		// Child category
		nameSlug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(pc.Name), " ", "-"))
		pc.Path = parentPath + "/" + nameSlug
		pc.Level = strings.Count(parentPath, "/")
	}
}

// GetPathSegments returns the path as an array of segments
func (pc *ProductCategory) GetPathSegments() []string {
	if pc.Path == "" {
		return []string{}
	}
	return strings.Split(strings.Trim(pc.Path, "/"), "/")
}

// IsActiveCategory returns true if the category is active
func (pc *ProductCategory) IsActiveCategory() bool {
	return pc.IsActive
}

// Activate activates the category
func (pc *ProductCategory) Activate() {
	pc.IsActive = true
	pc.UpdatedAt = time.Now().UTC()
}

// Deactivate deactivates the category
func (pc *ProductCategory) Deactivate() {
	pc.IsActive = false
	pc.UpdatedAt = time.Now().UTC()
}

// UpdateSortOrder updates the sort order
func (pc *ProductCategory) UpdateSortOrder(newOrder int) error {
	if newOrder < 0 {
		return errors.New("sort order cannot be negative")
	}
	pc.SortOrder = newOrder
	pc.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateImage updates the category image URL
func (pc *ProductCategory) UpdateImage(imageURL string) error {
	if imageURL != "" {
		// Validate the new image URL
		tempPC := &ProductCategory{ImageURL: imageURL}
		if err := tempPC.validateImageURL(); err != nil {
			return fmt.Errorf("invalid image URL: %w", err)
		}
	}
	pc.ImageURL = imageURL
	pc.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateDetails updates the category name and description
func (pc *ProductCategory) UpdateDetails(name, description string) error {
	// Create temporary category for validation
	tempPC := &ProductCategory{
		Name:        name,
		Description: description,
	}

	if err := tempPC.validateName(); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	if err := tempPC.validateDescription(); err != nil {
		return fmt.Errorf("invalid description: %w", err)
	}

	pc.Name = name
	pc.Description = description
	pc.UpdatedAt = time.Now().UTC()
	return nil
}

// MoveToParent moves this category to a new parent
func (pc *ProductCategory) MoveToParent(newParentID *uuid.UUID, newParentPath string) error {
	// Cannot move to self
	if newParentID != nil && *newParentID == pc.ID {
		return errors.New("category cannot be moved to be its own parent")
	}

	oldParentID := pc.ParentID
	pc.ParentID = newParentID

	// Rebuild path based on new parent
	if newParentID == nil {
		// Moving to root
		pc.BuildPath("")
		pc.Level = 0
	} else {
		// Moving under a parent
		pc.BuildPath(newParentPath)
	}

	pc.UpdatedAt = time.Now().UTC()

	// If we moved to a different parent, we might need to recalculate the path
	// This would be handled at the repository/service level for all children
	if (oldParentID == nil && newParentID != nil) ||
	   (oldParentID != nil && newParentID == nil) ||
	   (oldParentID != nil && newParentID != nil && *oldParentID != *newParentID) {
		return fmt.Errorf("category moved to different parent - child paths need recalculation")
	}

	return nil
}

// validateSEOFields validates the SEO fields
func (pc *ProductCategory) validateSEOFields() error {
	// SEO Title validation (optional)
	if pc.SEOTitle != "" && len(pc.SEOTitle) > 200 {
		return errors.New("SEO title cannot exceed 200 characters")
	}

	// SEO Description validation (optional)
	if pc.SEODescription != "" && len(pc.SEODescription) > 300 {
		return errors.New("SEO description cannot exceed 300 characters")
	}

	// SEO Keywords validation (optional)
	if pc.SEOKeywords != "" && len(pc.SEOKeywords) > 500 {
		return errors.New("SEO keywords cannot exceed 500 characters")
	}

	return nil
}

// UpdateSEOFields updates the SEO fields
func (pc *ProductCategory) UpdateSEOFields(title, description, keywords string) error {
	// Create temporary category for validation
	tempPC := &ProductCategory{
		SEOTitle:       title,
		SEODescription: description,
		SEOKeywords:   keywords,
	}

	if err := tempPC.validateSEOFields(); err != nil {
		return fmt.Errorf("invalid SEO fields: %w", err)
	}

	pc.SEOTitle = title
	pc.SEODescription = description
	pc.SEOKeywords = keywords
	pc.UpdatedAt = time.Now().UTC()
	return nil
}

// ToSafeCategory returns a category object without sensitive information
// (included for consistency, though categories don't typically contain sensitive data)
func (pc *ProductCategory) ToSafeCategory() *ProductCategory {
	return &ProductCategory{
		ID:            pc.ID,
		Name:          pc.Name,
		Description:   pc.Description,
		ParentID:      pc.ParentID,
		Level:         pc.Level,
		Path:          pc.Path,
		ImageURL:      pc.ImageURL,
		SortOrder:     pc.SortOrder,
		IsActive:      pc.IsActive,
		SEOTitle:       pc.SEOTitle,
		SEODescription: pc.SEODescription,
		SEOKeywords:   pc.SEOKeywords,
		CreatedAt:     pc.CreatedAt,
		UpdatedAt:     pc.UpdatedAt,
	}
}

// CategoryMetadata represents additional metadata for a category
type CategoryMetadata struct {
	CategoryID    uuid.UUID `json:"category_id" db:"category_id"`
	SeoTitle      string    `json:"seo_title" db:"seo_title"`
	SeoDescription string   `json:"seo_description" db:"seo_description"`
	SeoKeywords   string    `json:"seo_keywords" db:"seo_keywords"`
	MetaData      string    `json:"meta_data" db:"meta_data"` // JSON string for additional metadata
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// Validate validates the category metadata
func (cm *CategoryMetadata) Validate() error {
	var errs []error

	if cm.CategoryID == uuid.Nil {
		errs = append(errs, errors.New("category ID cannot be empty"))
	}

	if cm.SeoTitle != "" && len(cm.SeoTitle) > 200 {
		errs = append(errs, errors.New("SEO title cannot exceed 200 characters"))
	}

	if cm.SeoDescription != "" && len(cm.SeoDescription) > 300 {
		errs = append(errs, errors.New("SEO description cannot exceed 300 characters"))
	}

	if cm.SeoKeywords != "" && len(cm.SeoKeywords) > 500 {
		errs = append(errs, errors.New("SEO keywords cannot exceed 500 characters"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errors.Join(errs...))
	}

	return nil
}