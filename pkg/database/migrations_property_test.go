package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/rs/zerolog"
)

// TestProperty_MigrationTransactionRollback tests Property 17:
// **Feature: production-readiness, Property 17: Migration Transaction Rollback**
// For any database migration that fails, all changes made by that migration must be rolled back
// **Validates: Requirements 19.2**
//
// NOTE: This test requires a running PostgreSQL test database.
// Set TEST_DATABASE_URL environment variable or ensure the default test database is available.
// To run: TEST_DATABASE_URL="postgres://user:pass@localhost:5432/testdb?sslmode=disable" go test -run TestProperty_MigrationTransactionRollback
func TestProperty_MigrationTransactionRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup test database
	ctx := context.Background()
	db, cleanup, err := setupTestDB(t)
	if err != nil {
		t.Skipf("Skipping test: test database not available: %v", err)
	}
	defer cleanup()

	logger := zerolog.Nop()

	properties := gopter.NewProperties(nil)

	// Counter for generating unique version numbers
	versionCounter := 1000

	properties.Property("failed migrations rollback all changes", prop.ForAll(
		func(tableName string) bool {
			// Create a new migration runner for each test
			mr := NewMigrationRunner(db, &logger)

			// Generate a unique version number to avoid conflicts
			version := versionCounter
			versionCounter++

			// Create a migration that creates a table, then fails
			// This simulates a migration that partially succeeds before failing
			createTableSQL := fmt.Sprintf("CREATE TABLE %s (id INT PRIMARY KEY, name VARCHAR(255));", tableName)
			invalidSQL := "INVALID SQL SYNTAX HERE;"
			migrationSQL := createTableSQL + "\n" + invalidSQL

			migration := Migration{
				Version:     version,
				Name:        fmt.Sprintf("test_migration_%d", version),
				Description: "Test migration for rollback",
				UpSQL:       migrationSQL,
				DownSQL:     fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName),
			}

			mr.AddMigration(migration)

			// Run the migration - it should fail
			err := mr.Up(ctx)

			// Migration should fail due to invalid SQL
			if err == nil {
				t.Logf("Expected migration to fail but it succeeded")
				return false
			}

			// PROPERTY: Verify the table was NOT created (rollback occurred)
			// This is the key property we're testing - if the migration fails,
			// ALL changes should be rolled back, including the CREATE TABLE
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
				t.Logf("Error checking if table exists: %v", queryErr)
				return false
			}

			if exists {
				// PROPERTY VIOLATION: Table exists after failed migration
				// This means rollback did NOT occur
				t.Logf("PROPERTY VIOLATION: Table %s exists after failed migration - rollback did not occur", tableName)

				// Clean up the orphaned table
				_, _ = db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))

				return false
			}

			// PROPERTY: Verify the migration was NOT recorded in schema_migrations
			var recorded bool
			recordCheckSQL := `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1);`
			recordErr := db.QueryRow(ctx, recordCheckSQL, version).Scan(&recorded)
			if recordErr != nil {
				t.Logf("Error checking if migration was recorded: %v", recordErr)
				return false
			}

			if recorded {
				// PROPERTY VIOLATION: Migration was recorded despite failure
				t.Logf("PROPERTY VIOLATION: Migration %d was recorded despite failure - rollback did not occur", version)

				// Clean up the orphaned record
				_, _ = db.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1;", version)

				return false
			}

			// Property holds: failed migration was fully rolled back
			return true
		},
		genValidTableName(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(true))
}

// genValidTableName generates valid PostgreSQL table names
func genValidTableName() gopter.Gen {
	return gen.Identifier().
		SuchThat(func(v interface{}) bool {
			name := v.(string)
			// Ensure the name is valid for PostgreSQL
			// - starts with a letter or underscore
			// - contains only letters, numbers, and underscores
			// - is not a reserved keyword
			// - is not too long
			if len(name) == 0 || len(name) > 63 {
				return false
			}

			// Check first character
			first := name[0]
			if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
				return false
			}

			// Avoid common reserved keywords
			reserved := map[string]bool{
				"user": true, "table": true, "select": true, "insert": true,
				"update": true, "delete": true, "from": true, "where": true,
				"order": true, "group": true, "having": true, "limit": true,
			}

			if reserved[name] {
				return false
			}

			return true
		}).
		Map(func(v interface{}) interface{} {
			// Prefix with "test_" to make it clear these are test tables
			return "test_" + v.(string)
		})
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*Database, func(), error) {
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
