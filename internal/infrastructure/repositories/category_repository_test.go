package repositories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"erpgo/internal/domain/products/entities"
	"erpgo/internal/domain/products/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresCategoryRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create test category
	category := createTestCategory(t)

	err := repo.Create(ctx, category)
	assert.NoError(t, err)

	// Verify category was created
	retrieved, err := repo.GetByID(ctx, category.ID)
	require.NoError(t, err)
	assert.Equal(t, category.ID, retrieved.ID)
	assert.Equal(t, category.Name, retrieved.Name)
	assert.Equal(t, category.Path, retrieved.Path)
}

func TestPostgresCategoryRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create test category
	category := createTestCategory(t)
	err := repo.Create(ctx, category)
	require.NoError(t, err)

	// Get category by ID
	retrieved, err := repo.GetByID(ctx, category.ID)
	require.NoError(t, err)
	assert.Equal(t, category.ID, retrieved.ID)
	assert.Equal(t, category.Name, retrieved.Name)
	assert.Equal(t, category.Path, retrieved.Path)
}

func TestPostgresCategoryRepository_GetByPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create test category
	category := createTestCategory(t)
	err := repo.Create(ctx, category)
	require.NoError(t, err)

	// Get category by path
	retrieved, err := repo.GetByPath(ctx, category.Path)
	require.NoError(t, err)
	assert.Equal(t, category.ID, retrieved.ID)
	assert.Equal(t, category.Name, retrieved.Name)
	assert.Equal(t, category.Path, retrieved.Path)
}

func TestPostgresCategoryRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create test category
	category := createTestCategory(t)
	err := repo.Create(ctx, category)
	require.NoError(t, err)

	// Update category
	category.Name = "Updated Category Name"
	category.Description = "Updated description"
	category.ImageURL = "https://example.com/updated-image.jpg"
	category.UpdatedAt = time.Now().UTC()

	err = repo.Update(ctx, category)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetByID(ctx, category.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Category Name", retrieved.Name)
	assert.Equal(t, "Updated description", retrieved.Description)
	assert.Equal(t, "https://example.com/updated-image.jpg", retrieved.ImageURL)
}

func TestPostgresCategoryRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create test category
	category := createTestCategory(t)
	err := repo.Create(ctx, category)
	require.NoError(t, err)

	// Delete category
	err = repo.Delete(ctx, category.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = repo.GetByID(ctx, category.ID)
	assert.Error(t, err)
}

func TestPostgresCategoryRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create test categories
	categories := make([]*entities.ProductCategory, 5)
	for i := 0; i < 5; i++ {
		category := createTestCategory(t)
		category.Name = fmt.Sprintf("Test Category %d", i+1)
		category.Path = fmt.Sprintf("/test-category-%d", i+1)
		category.SortOrder = i + 1
		category.IsActive = i < 3 // First 3 are active
		categories[i] = category
		err := repo.Create(ctx, category)
		require.NoError(t, err)
	}

	// Test list without filter
	filter := repositories.CategoryFilter{
		Limit: 10,
	}
	result, err := repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, result, 5)

	// Test list with search filter
	filter.Search = "Category 1"
	result, err = repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, result, 1)

	// Test list with active filter
	filter = repositories.CategoryFilter{
		IsActive: boolPtr(true),
	}
	result, err = repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestPostgresCategoryRepository_ListRoot(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create root categories
	rootCategories := make([]*entities.ProductCategory, 3)
	for i := 0; i < 3; i++ {
		category := createTestCategory(t)
		category.Name = fmt.Sprintf("Root Category %d", i+1)
		category.Path = fmt.Sprintf("/root-category-%d", i+1)
		category.ParentID = nil
		category.Level = 0
		rootCategories[i] = category
		err := repo.Create(ctx, category)
		require.NoError(t, err)
	}

	// Create child categories
	for i := 0; i < 2; i++ {
		category := createTestCategory(t)
		category.Name = fmt.Sprintf("Child Category %d", i+1)
		category.Path = fmt.Sprintf("/root-category-1/child-category-%d", i+1)
		category.ParentID = &rootCategories[0].ID
		category.Level = 1
		err := repo.Create(ctx, category)
		require.NoError(t, err)
	}

	// Get root categories
	rootCats, err := repo.ListRoot(ctx)
	require.NoError(t, err)
	assert.Len(t, rootCats, 3)

	// Verify all are root categories
	for _, cat := range rootCats {
		assert.Nil(t, cat.ParentID)
		assert.Equal(t, 0, cat.Level)
	}
}

func TestPostgresCategoryRepository_GetChildren(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create parent category
	parentCategory := createTestCategory(t)
	parentCategory.Name = "Parent Category"
	parentCategory.Path = "/parent-category"
	err := repo.Create(ctx, parentCategory)
	require.NoError(t, err)

	// Create child categories
	childCategories := make([]*entities.ProductCategory, 3)
	for i := 0; i < 3; i++ {
		category := createTestCategory(t)
		category.Name = fmt.Sprintf("Child Category %d", i+1)
		category.Path = fmt.Sprintf("/parent-category/child-category-%d", i+1)
		category.ParentID = &parentCategory.ID
		category.Level = 1
		category.SortOrder = i + 1
		childCategories[i] = category
		err := repo.Create(ctx, category)
		require.NoError(t, err)
	}

	// Get children
	children, err := repo.GetChildren(ctx, parentCategory.ID)
	require.NoError(t, err)
	assert.Len(t, children, 3)

	// Verify children
	for i, child := range children {
		assert.Equal(t, childCategories[i].ID, child.ID)
		assert.Equal(t, childCategories[i].Name, child.Name)
		assert.Equal(t, parentCategory.ID, *child.ParentID)
		assert.Equal(t, 1, child.Level)
	}
}

func TestPostgresCategoryRepository_GetDescendants(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create grandparent category
	grandparent := createTestCategory(t)
	grandparent.Name = "Grandparent"
	grandparent.Path = "/grandparent"
	err := repo.Create(ctx, grandparent)
	require.NoError(t, err)

	// Create parent category
	parent := createTestCategory(t)
	parent.Name = "Parent"
	parent.Path = "/grandparent/parent"
	parent.ParentID = &grandparent.ID
	parent.Level = 1
	err = repo.Create(ctx, parent)
	require.NoError(t, err)

	// Create child categories
	child1 := createTestCategory(t)
	child1.Name = "Child 1"
	child1.Path = "/grandparent/parent/child-1"
	child1.ParentID = &parent.ID
	child1.Level = 2
	err = repo.Create(ctx, child1)
	require.NoError(t, err)

	child2 := createTestCategory(t)
	child2.Name = "Child 2"
	child2.Path = "/grandparent/parent/child-2"
	child2.ParentID = &parent.ID
	child2.Level = 2
	err = repo.Create(ctx, child2)
	require.NoError(t, err)

	// Create grandchild category
	grandchild := createTestCategory(t)
	grandchild.Name = "Grandchild"
	grandchild.Path = "/grandparent/parent/child-1/grandchild"
	grandchild.ParentID = &child1.ID
	grandchild.Level = 3
	err = repo.Create(ctx, grandchild)
	require.NoError(t, err)

	// Get descendants from parent
	descendants, err := repo.GetDescendants(ctx, parent.ID)
	require.NoError(t, err)
	assert.Len(t, descendants, 3) // child1, child2, grandchild

	// Verify descendants are ordered by level
	assert.True(t, descendants[0].Level <= descendants[1].Level)
	assert.True(t, descendants[1].Level <= descendants[2].Level)
}

func TestPostgresCategoryRepository_GetAncestors(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create ancestor categories
	grandparent := createTestCategory(t)
	grandparent.Name = "Grandparent"
	grandparent.Path = "/grandparent"
	err := repo.Create(ctx, grandparent)
	require.NoError(t, err)

	parent := createTestCategory(t)
	parent.Name = "Parent"
	parent.Path = "/grandparent/parent"
	parent.ParentID = &grandparent.ID
	parent.Level = 1
	err = repo.Create(ctx, parent)
	require.NoError(t, err)

	// Create child category
	child := createTestCategory(t)
	child.Name = "Child"
	child.Path = "/grandparent/parent/child"
	child.ParentID = &parent.ID
	child.Level = 2
	err = repo.Create(ctx, child)
	require.NoError(t, err)

	// Get ancestors from child
	ancestors, err := repo.GetAncestors(ctx, child.ID)
	require.NoError(t, err)
	assert.Len(t, ancestors, 2) // parent, grandparent

	// Verify ancestors are ordered from immediate to root (level DESC)
	assert.Equal(t, parent.ID, ancestors[0].ID)
	assert.Equal(t, grandparent.ID, ancestors[1].ID)
}

func TestPostgresCategoryRepository_GetPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create category path
	root := createTestCategory(t)
	root.Name = "Root"
	root.Path = "/root"
	root.ParentID = nil
	root.Level = 0
	err := repo.Create(ctx, root)
	require.NoError(t, err)

	level1 := createTestCategory(t)
	level1.Name = "Level 1"
	level1.Path = "/root/level-1"
	level1.ParentID = &root.ID
	level1.Level = 1
	err = repo.Create(ctx, level1)
	require.NoError(t, err)

	level2 := createTestCategory(t)
	level2.Name = "Level 2"
	level2.Path = "/root/level-1/level-2"
	level2.ParentID = &level1.ID
	level2.Level = 2
	err = repo.Create(ctx, level2)
	require.NoError(t, err)

	// Get full path
	path, err := repo.GetPath(ctx, level2.ID)
	require.NoError(t, err)
	assert.Len(t, path, 3)

	// Verify path order (root to current)
	assert.Equal(t, root.ID, path[0].ID)
	assert.Equal(t, level1.ID, path[1].ID)
	assert.Equal(t, level2.ID, path[2].ID)

	// Verify levels are sequential
	assert.Equal(t, 0, path[0].Level)
	assert.Equal(t, 1, path[1].Level)
	assert.Equal(t, 2, path[2].Level)
}

func TestPostgresCategoryRepository_Count(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create test categories
	for i := 0; i < 5; i++ {
		category := createTestCategory(t)
		category.Name = fmt.Sprintf("Test Category %d", i+1)
		category.Path = fmt.Sprintf("/test-category-%d", i+1)
		category.IsActive = i < 3 // First 3 are active
		err := repo.Create(ctx, category)
		require.NoError(t, err)
	}

	// Test count without filter
	filter := repositories.CategoryFilter{}
	count, err := repo.Count(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	// Test count with active filter
	filter.IsActive = boolPtr(true)
	count, err = repo.Count(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestPostgresCategoryRepository_ExistsByPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create test category
	category := createTestCategory(t)
	err := repo.Create(ctx, category)
	require.NoError(t, err)

	// Test existing path
	exists, err := repo.ExistsByPath(ctx, category.Path)
	require.NoError(t, err)
	assert.True(t, exists)

	// Test non-existing path
	exists, err = repo.ExistsByPath(ctx, "/non-existent-path")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPostgresCategoryRepository_ExistsByName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create test category
	category := createTestCategory(t)
	err := repo.Create(ctx, category)
	require.NoError(t, err)

	// Test existing name under same parent (nil)
	exists, err := repo.ExistsByName(ctx, category.Name, nil)
	require.NoError(t, err)
	assert.True(t, exists)

	// Test non-existing name
	exists, err = repo.ExistsByName(ctx, "Non-Existent", nil)
	require.NoError(t, err)
	assert.False(t, exists)

	// Test existing name under different parent
	parentID := uuid.New()
	exists, err = repo.ExistsByName(ctx, category.Name, &parentID)
	require.NoError(t, err)
	assert.False(t, exists) // Should not exist under different parent
}

func TestPostgresCategoryRepository_BulkUpdateSortOrder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresCategoryRepository(db)
	ctx := context.Background()

	// Create test categories
	categories := make([]*entities.ProductCategory, 3)
	for i := 0; i < 3; i++ {
		category := createTestCategory(t)
		category.Name = fmt.Sprintf("Category %d", i+1)
		category.Path = fmt.Sprintf("/category-%d", i+1)
		category.SortOrder = i + 1
		categories[i] = category
		err := repo.Create(ctx, category)
		require.NoError(t, err)
	}

	// Prepare sort order updates
	updates := []repositories.CategorySortUpdate{
		{CategoryID: categories[0].ID, SortOrder: 3},
		{CategoryID: categories[1].ID, SortOrder: 1},
		{CategoryID: categories[2].ID, SortOrder: 2},
	}

	// Update sort orders
	err := repo.BulkUpdateSortOrder(ctx, updates)
	assert.NoError(t, err)

	// Verify updates
	for _, update := range updates {
		category, err := repo.GetByID(ctx, update.CategoryID)
		require.NoError(t, err)
		assert.Equal(t, update.SortOrder, category.SortOrder)
	}
}
