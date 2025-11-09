package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"erpgo/internal/domain/users/entities"
	"erpgo/internal/domain/users/repositories"
	"erpgo/pkg/auth"
	"erpgo/pkg/database"
)

// PostgresUserRepositoryTestSuite contains all tests for the PostgreSQL user repository
type PostgresUserRepositoryTestSuite struct {
	suite.Suite
	db                *database.Database
	userRepo          *PostgresUserRepository
	roleRepo          *PostgresRoleRepository
	userRoleRepo      *PostgresUserRoleRepository
	passwordService   *auth.PasswordService
	ctx               context.Context
	cleanup           func()
}

// SetupSuite runs once before all tests
func (suite *PostgresUserRepositoryTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Create test database configuration
	config := database.Config{
		URL:             "postgres://erpgo_user:erpgo_password@localhost:5432/erpgo_test?sslmode=disable",
		MaxConnections:  10,
		MinConnections:  1,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
		SSLMode:         "disable",
	}

	// Initialize database
	db, err := database.New(config)
	require.NoError(suite.T(), err, "Failed to connect to test database")

	suite.db = db
	suite.userRepo = NewPostgresUserRepository(db)
	suite.roleRepo = NewPostgresRoleRepository(db)
	suite.userRoleRepo = NewPostgresUserRoleRepository(db)
	suite.passwordService = auth.NewPasswordService(12, "test-pepper")

	// Setup cleanup function
	suite.cleanup = func() {
		suite.cleanupTestData()
		db.Close()
	}

	// Create test tables
	suite.createTestTables()
}

// TearDownSuite runs once after all tests
func (suite *PostgresUserRepositoryTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

// SetupTest runs before each test
func (suite *PostgresUserRepositoryTestSuite) SetupTest() {
	suite.cleanupTestData()
}

// createTestTables creates the necessary tables for testing
func (suite *PostgresUserRepositoryTestSuite) createTestTables() {
	queries := []string{
		// Create users table
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			phone VARCHAR(20),
			is_active BOOLEAN DEFAULT true,
			is_verified BOOLEAN DEFAULT false,
			last_login_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		// Create roles table
		`CREATE TABLE IF NOT EXISTS roles (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name VARCHAR(100) UNIQUE NOT NULL,
			description TEXT,
			permissions TEXT[] DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		// Create user_roles table
		`CREATE TABLE IF NOT EXISTS user_roles (
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
			assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
			PRIMARY KEY (user_id, role_id)
		)`,
		// Create trigger function for updated_at
		`CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ language 'plpgsql'`,
		// Create triggers
		`CREATE TRIGGER update_users_updated_at
			BEFORE UPDATE ON users
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,
		`CREATE TRIGGER update_roles_updated_at
			BEFORE UPDATE ON roles
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,
	}

	for _, query := range queries {
		_, err := suite.db.Exec(suite.ctx, query)
		require.NoError(suite.T(), err, "Failed to create test table")
	}
}

// cleanupTestData cleans up all test data
func (suite *PostgresUserRepositoryTestSuite) cleanupTestData() {
	queries := []string{
		`DELETE FROM user_roles`,
		`DELETE FROM users`,
		`DELETE FROM roles`,
	}

	for _, query := range queries {
		_, err := suite.db.Exec(suite.ctx, query)
		if err != nil {
			// Log error but don't fail the test
			suite.T().Logf("Warning: failed to cleanup test data: %v", err)
		}
	}
}

// createTestUser creates a test user
func (suite *PostgresUserRepositoryTestSuite) createTestUser(email, username string) *entities.User {
	user := &entities.User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		FirstName:    "Test",
		LastName:     "User",
		Phone:        "+1234567890",
		IsActive:     true,
		IsVerified:   false,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Hash password
	hash, err := suite.passwordService.HashPassword("testPassword123!")
	require.NoError(suite.T(), err)
	user.PasswordHash = hash

	return user
}

// createTestRole creates a test role
func (suite *PostgresUserRepositoryTestSuite) createTestRole(name string) *entities.Role {
	return &entities.Role{
		ID:          uuid.New(),
		Name:        name,
		Description: "Test role",
		Permissions: []string{"test.read", "test.write"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

// TestCreateUser tests creating a new user
func (suite *PostgresUserRepositoryTestSuite) TestCreateUser() {
	user := suite.createTestUser("test@example.com", "testuser")

	err := suite.userRepo.Create(suite.ctx, user)
	assert.NoError(suite.T(), err)

	// Verify user was created
	retrieved, err := suite.userRepo.GetByID(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Email, retrieved.Email)
	assert.Equal(suite.T(), user.Username, retrieved.Username)
	assert.Equal(suite.T(), user.FirstName, retrieved.FirstName)
	assert.Equal(suite.T(), user.LastName, retrieved.LastName)
}

// TestGetByID tests retrieving a user by ID
func (suite *PostgresUserRepositoryTestSuite) TestGetByID() {
	user := suite.createTestUser("test@example.com", "testuser")
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Test existing user
	retrieved, err := suite.userRepo.GetByID(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Email, retrieved.Email)
	assert.Equal(suite.T(), user.Username, retrieved.Username)

	// Test non-existing user
	nonExistentID := uuid.New()
	_, err = suite.userRepo.GetByID(suite.ctx, nonExistentID)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not found")
}

// TestGetByEmail tests retrieving a user by email
func (suite *PostgresUserRepositoryTestSuite) TestGetByEmail() {
	user := suite.createTestUser("test@example.com", "testuser")
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Test existing user
	retrieved, err := suite.userRepo.GetByEmail(suite.ctx, user.Email)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, retrieved.ID)
	assert.Equal(suite.T(), user.Username, retrieved.Username)

	// Test non-existing user
	_, err = suite.userRepo.GetByEmail(suite.ctx, "nonexistent@example.com")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not found")
}

// TestGetByUsername tests retrieving a user by username
func (suite *PostgresUserRepositoryTestSuite) TestGetByUsername() {
	user := suite.createTestUser("test@example.com", "testuser")
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Test existing user
	retrieved, err := suite.userRepo.GetByUsername(suite.ctx, user.Username)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, retrieved.ID)
	assert.Equal(suite.T(), user.Email, retrieved.Email)

	// Test non-existing user
	_, err = suite.userRepo.GetByUsername(suite.ctx, "nonexistent")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not found")
}

// TestUpdateUser tests updating a user
func (suite *PostgresUserRepositoryTestSuite) TestUpdateUser() {
	user := suite.createTestUser("test@example.com", "testuser")
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Update user
	user.FirstName = "Updated"
	user.LastName = "Name"
	user.Phone = "+0987654321"
	user.IsActive = false
	user.IsVerified = true

	err = suite.userRepo.Update(suite.ctx, user)
	assert.NoError(suite.T(), err)

	// Verify update
	retrieved, err := suite.userRepo.GetByID(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated", retrieved.FirstName)
	assert.Equal(suite.T(), "Name", retrieved.LastName)
	assert.Equal(suite.T(), "+0987654321", retrieved.Phone)
	assert.Equal(suite.T(), false, retrieved.IsActive)
	assert.Equal(suite.T(), true, retrieved.IsVerified)
}

// TestDeleteUser tests deleting a user
func (suite *PostgresUserRepositoryTestSuite) TestDeleteUser() {
	user := suite.createTestUser("test@example.com", "testuser")
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Delete user
	err = suite.userRepo.Delete(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)

	// Verify deletion
	_, err = suite.userRepo.GetByID(suite.ctx, user.ID)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not found")
}

// TestListUsers tests listing users with filters
func (suite *PostgresUserRepositoryTestSuite) TestListUsers() {
	// Create test users
	users := []*entities.User{
		suite.createTestUser("user1@example.com", "user1"),
		suite.createTestUser("user2@example.com", "user2"),
		suite.createTestUser("admin@example.com", "admin"),
		suite.createTestUser("test.user@example.com", "testuser"),
	}

	for i, user := range users {
		user.FirstName = "User"
		user.LastName = string(rune('A' + i))
		if i == 2 {
			user.IsActive = false // Make admin user inactive
		}
		err := suite.userRepo.Create(suite.ctx, user)
		require.NoError(suite.T(), err)
	}

	// Test listing all users
	filter := repositories.UserFilter{}
	retrieved, err := suite.userRepo.List(suite.ctx, filter)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), retrieved, 4)

	// Test pagination
	filter = repositories.UserFilter{
		Page:  1,
		Limit: 2,
	}
	retrieved, err = suite.userRepo.List(suite.ctx, filter)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), retrieved, 2)

	// Test search by email
	filter = repositories.UserFilter{
		Search: "admin",
	}
	retrieved, err = suite.userRepo.List(suite.ctx, filter)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), retrieved, 1)
	assert.Equal(suite.T(), "admin@example.com", retrieved[0].Email)

	// Test search by name
	filter = repositories.UserFilter{
		Search: "User",
	}
	retrieved, err = suite.userRepo.List(suite.ctx, filter)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), retrieved, 4)

	// Test filter by active status
	active := true
	filter = repositories.UserFilter{
		IsActive: &active,
	}
	retrieved, err = suite.userRepo.List(suite.ctx, filter)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), retrieved, 3)

	// Test filter by inactive status
	inactive := false
	filter = repositories.UserFilter{
		IsActive: &inactive,
	}
	retrieved, err = suite.userRepo.List(suite.ctx, filter)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), retrieved, 1)

	// Test sorting
	filter = repositories.UserFilter{
		SortBy:    "email",
		SortOrder: "ASC",
	}
	retrieved, err = suite.userRepo.List(suite.ctx, filter)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), retrieved, 4)
	assert.Equal(suite.T(), "admin@example.com", retrieved[0].Email)
}

// TestCountUsers tests counting users with filters
func (suite *PostgresUserRepositoryTestSuite) TestCountUsers() {
	// Create test users
	users := []*entities.User{
		suite.createTestUser("user1@example.com", "user1"),
		suite.createTestUser("user2@example.com", "user2"),
		suite.createTestUser("admin@example.com", "admin"),
	}

	for i, user := range users {
		user.FirstName = "User"
		user.LastName = string(rune('A' + i))
		if i == 2 {
			user.IsActive = false
		}
		err := suite.userRepo.Create(suite.ctx, user)
		require.NoError(suite.T(), err)
	}

	// Test counting all users
	filter := repositories.UserFilter{}
	count, err := suite.userRepo.Count(suite.ctx, filter)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, count)

	// Test counting with search filter
	filter = repositories.UserFilter{
		Search: "admin",
	}
	count, err = suite.userRepo.Count(suite.ctx, filter)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, count)

	// Test counting with active filter
	active := true
	filter = repositories.UserFilter{
		IsActive: &active,
	}
	count, err = suite.userRepo.Count(suite.ctx, filter)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, count)
}

// TestExistsByEmail tests checking if user exists by email
func (suite *PostgresUserRepositoryTestSuite) TestExistsByEmail() {
	user := suite.createTestUser("test@example.com", "testuser")
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Test existing email
	exists, err := suite.userRepo.ExistsByEmail(suite.ctx, user.Email)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// Test non-existing email
	exists, err = suite.userRepo.ExistsByEmail(suite.ctx, "nonexistent@example.com")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

// TestExistsByUsername tests checking if user exists by username
func (suite *PostgresUserRepositoryTestSuite) TestExistsByUsername() {
	user := suite.createTestUser("test@example.com", "testuser")
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Test existing username
	exists, err := suite.userRepo.ExistsByUsername(suite.ctx, user.Username)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// Test non-existing username
	exists, err = suite.userRepo.ExistsByUsername(suite.ctx, "nonexistent")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

// TestUpdateLastLogin tests updating user's last login time
func (suite *PostgresUserRepositoryTestSuite) TestUpdateLastLogin() {
	user := suite.createTestUser("test@example.com", "testuser")
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Initially last login should be nil
	assert.Nil(suite.T(), user.LastLoginAt)

	// Update last login
	err = suite.userRepo.UpdateLastLogin(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)

	// Verify last login was updated
	retrieved, err := suite.userRepo.GetByID(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), retrieved.LastLoginAt)
}

// TestGetUserRoles tests retrieving user roles
func (suite *PostgresUserRepositoryTestSuite) TestGetUserRoles() {
	// Create test user
	user := suite.createTestUser("test@example.com", "testuser")
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Create test roles
	role1 := suite.createTestRole("user")
	role2 := suite.createTestRole("admin")
	err = suite.roleRepo.Create(suite.ctx, role1)
	require.NoError(suite.T(), err)
	err = suite.roleRepo.Create(suite.ctx, role2)
	require.NoError(suite.T(), err)

	// Assign roles to user
	err = suite.userRoleRepo.AssignRole(suite.ctx, user.ID.String(), role1.ID.String(), user.ID.String())
	require.NoError(suite.T(), err)
	err = suite.userRoleRepo.AssignRole(suite.ctx, user.ID.String(), role2.ID.String(), user.ID.String())
	require.NoError(suite.T(), err)

	// Get user roles
	roles, err := suite.userRepo.GetUserRoles(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), roles, 2)
	assert.Contains(suite.T(), roles, "user")
	assert.Contains(suite.T(), roles, "admin")
}

// TestAssignRole tests assigning a role to a user
func (suite *PostgresUserRepositoryTestSuite) TestAssignRole() {
	// Create test user
	user := suite.createTestUser("test@example.com", "testuser")
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Create test role
	role := suite.createTestRole("user")
	err = suite.roleRepo.Create(suite.ctx, role)
	require.NoError(suite.T(), err)

	// Assign role to user
	err = suite.userRepo.AssignRole(suite.ctx, user.ID, role.Name, user.ID)
	assert.NoError(suite.T(), err)

	// Verify role was assigned
	roles, err := suite.userRepo.GetUserRoles(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), roles, 1)
	assert.Equal(suite.T(), role.Name, roles[0])

	// Test assigning non-existing role
	err = suite.userRepo.AssignRole(suite.ctx, user.ID, "nonexistent", user.ID)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not found")
}

// TestRoleRepository tests the role repository
func (suite *PostgresUserRepositoryTestSuite) TestRoleRepository() {
	// Create test role
	role := suite.createTestRole("testrole")
	err := suite.roleRepo.Create(suite.ctx, role)
	assert.NoError(suite.T(), err)

	// Get role by ID
	retrieved, err := suite.roleRepo.GetByID(suite.ctx, role.ID.String())
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), role.Name, retrieved.Name)

	// Get role by name
	retrieved, err = suite.roleRepo.GetByName(suite.ctx, role.Name)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), role.ID, retrieved.ID)

	// Update role
	role.Description = "Updated description"
	role.Permissions = []string{"test.read", "test.write", "test.admin"}
	err = suite.roleRepo.Update(suite.ctx, role)
	assert.NoError(suite.T(), err)

	retrieved, err = suite.roleRepo.GetByID(suite.ctx, role.ID.String())
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated description", retrieved.Description)
	assert.Equal(suite.T(), 3, len(retrieved.Permissions))

	// List roles
	filter := repositories.RoleFilter{}
	roles, err := suite.roleRepo.List(suite.ctx, filter)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), roles, 1)

	// Count roles
	count, err := suite.roleRepo.Count(suite.ctx, filter)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, count)

	// Delete role
	err = suite.roleRepo.Delete(suite.ctx, role.ID.String())
	assert.NoError(suite.T(), err)

	_, err = suite.roleRepo.GetByID(suite.ctx, role.ID.String())
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not found")
}

// TestUserRoleRepository tests the user role repository
func (suite *PostgresUserRepositoryTestSuite) TestUserRoleRepository() {
	// Create test users
	user1 := suite.createTestUser("user1@example.com", "user1")
	user2 := suite.createTestUser("user2@example.com", "user2")
	err := suite.userRepo.Create(suite.ctx, user1)
	require.NoError(suite.T(), err)
	err = suite.userRepo.Create(suite.ctx, user2)
	require.NoError(suite.T(), err)

	// Create test role
	role := suite.createTestRole("testrole")
	err = suite.roleRepo.Create(suite.ctx, role)
	require.NoError(suite.T(), err)

	// Assign role to users
	err = suite.userRoleRepo.AssignRole(suite.ctx, user1.ID.String(), role.ID.String(), user1.ID.String())
	require.NoError(suite.T(), err)
	err = suite.userRoleRepo.AssignRole(suite.ctx, user2.ID.String(), role.ID.String(), user1.ID.String())
	require.NoError(suite.T(), err)

	// Get users by role
	users, err := suite.userRoleRepo.GetUsersByRole(suite.ctx, role.ID.String())
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), users, 2)

	// Check if user has role
	hasRole, err := suite.userRoleRepo.HasRole(suite.ctx, user1.ID.String(), role.ID.String())
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasRole)

	// Check if user doesn't have role
	nonExistentRoleID := uuid.New().String()
	hasRole, err = suite.userRoleRepo.HasRole(suite.ctx, user1.ID.String(), nonExistentRoleID)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), hasRole)

	// Remove role from user
	err = suite.userRoleRepo.RemoveRole(suite.ctx, user1.ID.String(), role.ID.String())
	assert.NoError(suite.T(), err)

	hasRole, err = suite.userRoleRepo.HasRole(suite.ctx, user1.ID.String(), role.ID.String())
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), hasRole)

	// Verify other user still has role
	hasRole, err = suite.userRoleRepo.HasRole(suite.ctx, user2.ID.String(), role.ID.String())
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasRole)
}

// TestUserRepository runs all user repository tests
func TestUserRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping repository tests in short mode")
	}

	suite.Run(t, new(PostgresUserRepositoryTestSuite))
}