package database

import (
	"context"
	"fmt"
	"testing"
	"testing/fstest"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationRunner_AddMigration(t *testing.T) {
	logger := zerolog.Nop()
	db := &Database{}
	mr := NewMigrationRunner(db, &logger)

	migration := Migration{
		Version:     1,
		Name:        "test_migration",
		Description: "Test migration",
		UpSQL:       "CREATE TABLE test (id INT);",
		DownSQL:     "DROP TABLE test;",
	}

	mr.AddMigration(migration)

	assert.Len(t, mr.migrations, 1)
	assert.Equal(t, migration, mr.migrations[0])
}

func TestMigrationRunner_AddMigrations(t *testing.T) {
	logger := zerolog.Nop()
	db := &Database{}
	mr := NewMigrationRunner(db, &logger)

	migrations := []Migration{
		{
			Version: 1,
			Name:    "first_migration",
			UpSQL:   "CREATE TABLE first (id INT);",
		},
		{
			Version: 2,
			Name:    "second_migration",
			UpSQL:   "CREATE TABLE second (id INT);",
		},
	}

	mr.AddMigrations(migrations)

	assert.Len(t, mr.migrations, 2)
	assert.Equal(t, migrations[0], mr.migrations[0])
	assert.Equal(t, migrations[1], mr.migrations[1])
}

func TestMigrationRunner_LoadMigrationsFromFS(t *testing.T) {
	logger := zerolog.Nop()
	db := &Database{}
	mr := NewMigrationRunner(db, &logger)

	// Create a mock filesystem
	fsys := fstest.MapFS{
		"001_create_users_table.up.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255));"),
		},
		"001_create_users_table.down.sql": &fstest.MapFile{
			Data: []byte("DROP TABLE users;"),
		},
		"002_create_products_table.up.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE products (id INT PRIMARY KEY, name VARCHAR(255));"),
		},
		"002_create_products_table.down.sql": &fstest.MapFile{
			Data: []byte("DROP TABLE products;"),
		},
		"invalid_file.txt": &fstest.MapFile{
			Data: []byte("invalid file"),
		},
		"003_invalid_format.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE invalid (id INT);"),
		},
		"not_a_migration.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE not_a_migration (id INT);"),
		},
	}

	err := mr.LoadMigrationsFromFS(fsys, ".")
	require.NoError(t, err)

	// Should have loaded 2 migrations (version 1 and 2)
	assert.Len(t, mr.migrations, 2)

	// Check first migration
	migration1 := mr.migrations[0]
	assert.Equal(t, 1, migration1.Version)
	assert.Equal(t, "001_create_users_table", migration1.Name)
	assert.Equal(t, "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255));", migration1.UpSQL)
	assert.Equal(t, "DROP TABLE users;", migration1.DownSQL)

	// Check second migration
	migration2 := mr.migrations[1]
	assert.Equal(t, 2, migration2.Version)
	assert.Equal(t, "002_create_products_table", migration2.Name)
	assert.Equal(t, "CREATE TABLE products (id INT PRIMARY KEY, name VARCHAR(255));", migration2.UpSQL)
	assert.Equal(t, "DROP TABLE products;", migration2.DownSQL)
}

func TestMigrationRunner_LoadMigrationsFromFS_WithDuplicates(t *testing.T) {
	logger := zerolog.Nop()
	db := &Database{}
	mr := NewMigrationRunner(db, &logger)

	// Create a mock filesystem with duplicate versions
	fsys := fstest.MapFS{
		"001_first.up.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE first (id INT PRIMARY KEY);"),
		},
		"001_second.up.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE second (id INT PRIMARY KEY);"),
		},
	}

	err := mr.LoadMigrationsFromFS(fsys, ".")
	require.NoError(t, err)

	// Should have only one migration with the version 1
	assert.Len(t, mr.migrations, 1)

	// The migration should have the UpSQL from the last file processed
	migration := mr.migrations[0]
	assert.Equal(t, 1, migration.Version)
	assert.Equal(t, "CREATE TABLE second (id INT PRIMARY KEY);", migration.UpSQL)
}

func TestMigrationRunner_LoadMigrationsFromFS_WithUpAndDown(t *testing.T) {
	logger := zerolog.Nop()
	db := &Database{}
	mr := NewMigrationRunner(db, &logger)

	// Create a mock filesystem with separate up and down files
	fsys := fstest.MapFS{
		"001_create_table.up.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE test (id INT PRIMARY KEY);"),
		},
		"001_create_table.down.sql": &fstest.MapFile{
			Data: []byte("DROP TABLE test;"),
		},
	}

	err := mr.LoadMigrationsFromFS(fsys, ".")
	require.NoError(t, err)

	// Should have one migration with both up and down SQL
	assert.Len(t, mr.migrations, 1)

	migration := mr.migrations[0]
	assert.Equal(t, 1, migration.Version)
	assert.Equal(t, "CREATE TABLE test (id INT PRIMARY KEY);", migration.UpSQL)
	assert.Equal(t, "DROP TABLE test;", migration.DownSQL)
}

func TestMigrationStatus(t *testing.T) {
	now := time.Now()
	status := MigrationStatus{
		Version:     1,
		Name:        "test_migration",
		Description: "Test migration",
		Applied:     true,
		AppliedAt:   &now,
	}

	assert.Equal(t, 1, status.Version)
	assert.Equal(t, "test_migration", status.Name)
	assert.Equal(t, "Test migration", status.Description)
	assert.True(t, status.Applied)
	assert.Equal(t, &now, status.AppliedAt)
}

func TestMigrationRunner_GetPendingMigrations(t *testing.T) {
	logger := zerolog.Nop()
	db := &Database{}
	mr := NewMigrationRunner(db, &logger)

	// Mock migrations
	mr.migrations = []Migration{
		{Version: 1, Name: "first"},
		{Version: 2, Name: "second"},
		{Version: 3, Name: "third"},
	}

	// Mock a scenario where only the first migration is applied
	ctx := context.Background()

	// Since we don't have a real database, this will fail, but we can test
	// the method structure
	pending, err := mr.GetPendingMigrations(ctx)
	assert.Error(t, err)        // Expected to fail without a real database
	assert.Equal(t, 0, pending) // Default value when error occurs
}

func TestMigrationRunner_Status_Structure(t *testing.T) {
	logger := zerolog.Nop()
	db := &Database{}
	mr := NewMigrationRunner(db, &logger)

	// Mock migrations
	mr.migrations = []Migration{
		{Version: 1, Name: "first", Description: "First migration"},
		{Version: 2, Name: "second", Description: "Second migration"},
	}

	// Test the structure (will fail without real DB, but tests the interface)
	ctx := context.Background()
	statuses, err := mr.Status(ctx)
	assert.Error(t, err)    // Expected to fail without a real database
	assert.Nil(t, statuses) // Should be nil when error occurs
}

func TestMigration_ErrorHandling(t *testing.T) {
	logger := zerolog.Nop()
	db := &Database{}
	mr := NewMigrationRunner(db, &logger)

	// Test adding migration with no UpSQL
	migration := Migration{
		Version: 1,
		Name:    "test",
		UpSQL:   "", // Missing UpSQL
		DownSQL: "DROP TABLE test;",
	}

	mr.AddMigration(migration)

	// Test running Up migration (will fail due to missing DB)
	ctx := context.Background()
	err := mr.Up(ctx)
	assert.Error(t, err) // Expected to fail without a real database
}

func TestMigration_Down_Methods(t *testing.T) {
	logger := zerolog.Nop()
	db := &Database{}
	mr := NewMigrationRunner(db, &logger)

	// Add a migration
	migration := Migration{
		Version: 1,
		Name:    "test",
		UpSQL:   "CREATE TABLE test (id INT);",
		DownSQL: "DROP TABLE test;",
	}

	mr.AddMigration(migration)

	// Test running Down migration (will fail due to missing DB)
	ctx := context.Background()
	err := mr.Down(ctx, 1)
	assert.Error(t, err) // Expected to fail without a real database
}

// Integration test patterns (these would require a real database)
func TestMigrationRunner_IntegrationPatterns(t *testing.T) {
	logger := zerolog.Nop()
	db := &Database{}
	mr := NewMigrationRunner(db, &logger)

	// Test that the runner handles empty migration list
	assert.Len(t, mr.migrations, 0)

	// Test adding migrations in non-sequential order
	migrations := []Migration{
		{Version: 3, Name: "third", UpSQL: "CREATE TABLE third (id INT);"},
		{Version: 1, Name: "first", UpSQL: "CREATE TABLE first (id INT);"},
		{Version: 2, Name: "second", UpSQL: "CREATE TABLE second (id INT);"},
	}

	mr.AddMigrations(migrations)
	assert.Len(t, mr.migrations, 3)

	// The runner should maintain the order of additions
	// (sorting happens during execution)
	assert.Equal(t, 3, mr.migrations[0].Version)
	assert.Equal(t, 1, mr.migrations[1].Version)
	assert.Equal(t, 2, mr.migrations[2].Version)
}

func TestMigration_ConfigurationValidation(t *testing.T) {
	logger := zerolog.Nop()

	// Test creating migration runner with nil database
	assert.NotPanics(t, func() {
		mr := NewMigrationRunner(nil, &logger)
		assert.NotNil(t, mr)
		assert.Nil(t, mr.db)
		assert.NotNil(t, mr.logger)
		assert.Empty(t, mr.migrations)
	})
}

// Benchmark tests
func BenchmarkMigrationRunner_AddMigration(b *testing.B) {
	logger := zerolog.Nop()
	db := &Database{}
	mr := NewMigrationRunner(db, &logger)

	migration := Migration{
		Version:     1,
		Name:        "test_migration",
		Description: "Test migration",
		UpSQL:       "CREATE TABLE test (id INT);",
		DownSQL:     "DROP TABLE test;",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mr.AddMigration(migration)
	}
}

func BenchmarkMigrationRunner_LoadMigrationsFromFS(b *testing.B) {
	logger := zerolog.Nop()
	db := &Database{}

	// Create a mock filesystem with many migrations
	fsys := fstest.MapFS{}
	for i := 1; i <= 100; i++ {
		version := i
		if version < 10 {
			fsys[fmt.Sprintf("00%d_test_table.up.sql", version)] = &fstest.MapFile{
				Data: []byte(fmt.Sprintf("CREATE TABLE test%d (id INT);", version)),
			}
		} else if version < 100 {
			fsys[fmt.Sprintf("0%d_test_table.up.sql", version)] = &fstest.MapFile{
				Data: []byte(fmt.Sprintf("CREATE TABLE test%d (id INT);", version)),
			}
		} else {
			fsys[fmt.Sprintf("%d_test_table.up.sql", version)] = &fstest.MapFile{
				Data: []byte(fmt.Sprintf("CREATE TABLE test%d (id INT);", version)),
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mr := NewMigrationRunner(db, &logger)
		_ = mr.LoadMigrationsFromFS(fsys, ".")
	}
}
