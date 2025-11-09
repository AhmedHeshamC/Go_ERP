package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"erpgo/internal/domain/users/entities"
	"erpgo/pkg/auth"
	"erpgo/pkg/database"
)

// TestDatabase wraps a database connection for testing
type TestDatabase struct {
	DB       *database.Database
	PasswordService *auth.PasswordService
	Cleanup  func()
}

// SetupTestDatabase creates a test database connection
func SetupTestDatabase(t *testing.T) *TestDatabase {
	// Get database connection string from environment or use default
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://erpgo_user:erpgo_password@localhost:5432/erpgo_test?sslmode=disable"
	}

	// Create test database configuration
	config := database.Config{
		URL:             dbURL,
		MaxConnections:  10,
		MinConnections:  1,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
		SSLMode:         "disable",
	}

	// Initialize database
	db, err := database.New(config)
	require.NoError(t, err, "Failed to connect to test database")

	// Initialize password service
	passwordService := auth.NewPasswordService(12, "test-pepper")

	// Setup cleanup function
	cleanup := func() {
		CleanupTestDatabase(t, db)
		db.Close()
	}

	// Create test tables
	CreateTestTables(t, db)

	return &TestDatabase{
		DB:       db,
		PasswordService: passwordService,
		Cleanup:  cleanup,
	}
}

// CreateTestTables creates the necessary tables for testing
func CreateTestTables(t *testing.T, db *database.Database) {
	ctx := context.Background()

	queries := []string{
		// Enable UUID extension
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,

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
		`DROP TRIGGER IF EXISTS update_users_updated_at ON users`,
		`CREATE TRIGGER update_users_updated_at
			BEFORE UPDATE ON users
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_roles_updated_at ON roles`,
		`CREATE TRIGGER update_roles_updated_at
			BEFORE UPDATE ON roles
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		// Create indexes
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_users_verified ON users(is_verified)`,
		`CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name)`,
		`CREATE INDEX IF NOT EXISTS idx_roles_created_at ON roles(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_roles_assigned_at ON user_roles(assigned_at)`,
	}

	for _, query := range queries {
		_, err := db.Exec(ctx, query)
		require.NoError(t, err, fmt.Sprintf("Failed to execute query: %s", query))
	}
}

// CleanupTestDatabase cleans up all test data
func CleanupTestDatabase(t *testing.T, db *database.Database) {
	ctx := context.Background()

	queries := []string{
		`DELETE FROM user_roles`,
		`DELETE FROM users`,
		`DELETE FROM roles`,
	}

	for _, query := range queries {
		_, err := db.Exec(ctx, query)
		if err != nil {
			t.Logf("Warning: failed to cleanup test data: %v", err)
		}
	}
}

// CreateTestUser creates a test user with hashed password
func CreateTestUser(t *testing.T, passwordService *auth.PasswordService, email, username string) *entities.User {
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
	hash, err := passwordService.HashPassword("testPassword123!")
	require.NoError(t, err)
	user.PasswordHash = hash

	return user
}

// CreateTestRole creates a test role
func CreateTestRole(t *testing.T, name string, permissions []string) *entities.Role {
	return &entities.Role{
		ID:          uuid.New(),
		Name:        name,
		Description: "Test role for " + name,
		Permissions: permissions,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

// CreateAdminUser creates a test admin user
func CreateAdminUser(t *testing.T, passwordService *auth.PasswordService) *entities.User {
	user := CreateTestUser(t, passwordService, "admin@test.com", "admin")
	user.FirstName = "Admin"
	user.LastName = "User"
	user.IsVerified = true
	return user
}

// CreateTeacherUser creates a test teacher user
func CreateTeacherUser(t *testing.T, passwordService *auth.PasswordService) *entities.User {
	user := CreateTestUser(t, passwordService, "teacher@test.com", "teacher")
	user.FirstName = "Teacher"
	user.LastName = "User"
	user.IsVerified = true
	return user
}

// CreateStudentUser creates a test student user
func CreateStudentUser(t *testing.T, passwordService *auth.PasswordService) *entities.User {
	user := CreateTestUser(t, passwordService, "student@test.com", "student")
	user.FirstName = "Student"
	user.LastName = "User"
	user.IsVerified = true
	return user
}

// AssertUserEquals asserts that two users are equal (ignoring timestamps)
func AssertUserEquals(t *testing.T, expected, actual *entities.User) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.Email, actual.Email)
	require.Equal(t, expected.Username, actual.Username)
	require.Equal(t, expected.FirstName, actual.FirstName)
	require.Equal(t, expected.LastName, actual.LastName)
	require.Equal(t, expected.Phone, actual.Phone)
	require.Equal(t, expected.IsActive, actual.IsActive)
	require.Equal(t, expected.IsVerified, actual.IsVerified)
	// Note: We don't compare timestamps as they may differ slightly
}

// AssertRoleEquals asserts that two roles are equal (ignoring timestamps)
func AssertRoleEquals(t *testing.T, expected, actual *entities.Role) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.Name, actual.Name)
	require.Equal(t, expected.Description, actual.Description)
	require.Equal(t, expected.Permissions, actual.Permissions)
	// Note: We don't compare timestamps as they may differ slightly
}

// WaitForDatabase waits for the database to be ready
func WaitForDatabase(t *testing.T, dbURL string, maxAttempts int) {
	ctx := context.Background()

	for i := 0; i < maxAttempts; i++ {
		config, err := pgxpool.ParseConfig(dbURL)
		require.NoError(t, err)

		pool, err := pgxpool.NewWithConfig(ctx, config)
		if err == nil {
			if err := pool.Ping(ctx); err == nil {
				pool.Close()
				return
			}
			pool.Close()
		}

		if i < maxAttempts-1 {
			t.Logf("Database not ready, waiting... (attempt %d/%d)", i+1, maxAttempts)
			time.Sleep(time.Second * 2)
		}
	}

	require.Fail(t, "Database not ready after %d attempts", maxAttempts)
}