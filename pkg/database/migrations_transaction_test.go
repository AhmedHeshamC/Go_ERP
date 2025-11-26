package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// TestMigrationTransactionRollback_Integration is an integration test that verifies
// migrations are rolled back on failure
func TestMigrationTransactionRollback_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	ctx := context.Background()
	db, cleanup, err := setupTestDBForIntegration(t)
	if err != nil {
		t.Skipf("Skipping test: test database not available: %v", err)
	}
	defer cleanup()

	logger := zerolog.Nop()
	mr := NewMigrationRunner(db, &logger)

	// Create a migration that will fail
	tableName := "test_rollback_table"
	version := 9999 // Use a high version to avoid conflicts
	
	// This migration creates a table, then fails
	migrationSQL := fmt.Sprintf(`
		CREATE TABLE %s (id INT PRIMARY KEY, name VARCHAR(255));
		INVALID SQL SYNTAX HERE;
	`, tableName)

	migration := Migration{
		Version:     version,
		Name:        "test_rollback_migration",
		Description: "Test migration for rollback",
		UpSQL:       migrationSQL,
		DownSQL:     fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName),
	}

	mr.AddMigration(migration)

	// Run the migration - it should fail
	err = mr.Up(ctx)
	if err == nil {
		t.Fatal("Expected migration to fail, but it succeeded")
	}

	// Verify the table was NOT created (rollback occurred)
	var exists bool
	checkSQL := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = '%s'
		);
	`, tableName)
	
	queryErr := db.QueryRow(ctx, checkSQL).Scan(&exists)
	if queryErr != nil {
		t.Fatalf("Error checking if table exists: %v", queryErr)
	}

	if exists {
		t.Errorf("Table %s exists after failed migration - rollback did not occur", tableName)
		// Clean up
		_, _ = db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
	}

	// Verify the migration was NOT recorded
	var recorded bool
	recordCheckSQL := `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1);`
	recordErr := db.QueryRow(ctx, recordCheckSQL, version).Scan(&recorded)
	if recordErr != nil {
		t.Fatalf("Error checking if migration was recorded: %v", recordErr)
	}

	if recorded {
		t.Errorf("Migration %d was recorded despite failure", version)
		// Clean up
		_, _ = db.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1;", version)
	}
}

// TestMigrationTransactionSuccess_Integration verifies successful migrations are committed
func TestMigrationTransactionSuccess_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	ctx := context.Background()
	db, cleanup, err := setupTestDBForIntegration(t)
	if err != nil {
		t.Skipf("Skipping test: test database not available: %v", err)
	}
	defer cleanup()

	logger := zerolog.Nop()
	mr := NewMigrationRunner(db, &logger)

	// Create a migration that will succeed
	tableName := "test_success_table"
	version := 9998 // Use a high version to avoid conflicts
	
	migrationSQL := fmt.Sprintf(`
		CREATE TABLE %s (id INT PRIMARY KEY, name VARCHAR(255));
	`, tableName)

	migration := Migration{
		Version:     version,
		Name:        "test_success_migration",
		Description: "Test migration for success",
		UpSQL:       migrationSQL,
		DownSQL:     fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName),
	}

	mr.AddMigration(migration)

	// Run the migration - it should succeed
	err = mr.Up(ctx)
	if err != nil {
		t.Fatalf("Expected migration to succeed, but it failed: %v", err)
	}

	// Verify the table WAS created
	var exists bool
	checkSQL := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = '%s'
		);
	`, tableName)
	
	queryErr := db.QueryRow(ctx, checkSQL).Scan(&exists)
	if queryErr != nil {
		t.Fatalf("Error checking if table exists: %v", queryErr)
	}

	if !exists {
		t.Errorf("Table %s does not exist after successful migration", tableName)
	}

	// Verify the migration WAS recorded
	var recorded bool
	recordCheckSQL := `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1);`
	recordErr := db.QueryRow(ctx, recordCheckSQL, version).Scan(&recorded)
	if recordErr != nil {
		t.Fatalf("Error checking if migration was recorded: %v", recordErr)
	}

	if !recorded {
		t.Errorf("Migration %d was not recorded after success", version)
	}

	// Clean up
	_, _ = db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
	_, _ = db.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1;", version)
}

// setupTestDBForIntegration creates a test database connection for integration tests
func setupTestDBForIntegration(t *testing.T) (*Database, func(), error) {
	t.Helper()

	// Get database URL from environment or use default test database
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://erpgo_user:erpgo_password@localhost:5432/erpgo_test?sslmode=disable"
	}

	// Create test database configuration
	config := Config{
		URL:             dbURL,
		MaxConnections:  10,
		MinConnections:  1,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
		SSLMode:         "disable",
	}

	// Initialize database
	db, err := New(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup, nil
}
