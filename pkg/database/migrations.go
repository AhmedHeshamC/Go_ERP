package database

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Name        string
	UpSQL       string
	DownSQL     string
	AppliedAt   *time.Time
	Description string
}

// MigrationRunner handles database migrations
type MigrationRunner struct {
	db        *Database
	logger    *zerolog.Logger
	migrations []Migration
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *Database, logger *zerolog.Logger) *MigrationRunner {
	return &MigrationRunner{
		db:     db,
		logger: logger,
	}
}

// AddMigration adds a migration to be run
func (mr *MigrationRunner) AddMigration(migration Migration) {
	mr.migrations = append(mr.migrations, migration)
}

// AddMigrations adds multiple migrations
func (mr *MigrationRunner) AddMigrations(migrations []Migration) {
	mr.migrations = append(mr.migrations, migrations...)
}

// LoadMigrationsFromFS loads migrations from an embedded filesystem
func (mr *MigrationRunner) LoadMigrationsFromFS(migrationFS fs.FS, dir string) error {
	return fs.WalkDir(migrationFS, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		filename := filepath.Base(path)

		// Only process .sql files
		if !strings.HasSuffix(strings.ToLower(filename), ".sql") {
			return nil
		}

		// Parse migration filename format: 001_create_users_table.up.sql
		parts := strings.Split(filename, "_")
		if len(parts) < 3 {
			mr.logger.Warn().Str("file", filename).Msg("Ignoring migration file with invalid format")
			return nil
		}

		versionStr := parts[0]
		version := 0
		if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
			mr.logger.Warn().Str("file", filename).Msg("Ignoring migration file with invalid version")
			return nil
		}

		// Read migration content
		content, err := fs.ReadFile(migrationFS, path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", path, err)
		}

		// Extract migration name from filename
		nameParts := strings.Split(filename, ".")
		name := strings.Join(nameParts[:len(nameParts)-2], ".")

		// Determine if it's an up or down migration
		var isUpMigration bool
		if strings.HasSuffix(strings.ToLower(filename), ".up.sql") {
			isUpMigration = true
		} else if strings.HasSuffix(strings.ToLower(filename), ".down.sql") {
			isUpMigration = false
		} else {
			mr.logger.Warn().Str("file", filename).Msg("Ignoring migration file without .up.sql or .down.sql suffix")
			return nil
		}

		// Create migration
		migration := Migration{
			Version:     version,
			Name:        name,
			Description: name,
		}

		if isUpMigration {
			migration.UpSQL = string(content)
		} else {
			migration.DownSQL = string(content)
		}

		// Check if migration already exists
		for i, existing := range mr.migrations {
			if existing.Version == migration.Version {
				if isUpMigration {
					mr.migrations[i].UpSQL = migration.UpSQL
				} else {
					mr.migrations[i].DownSQL = migration.DownSQL
				}
				mr.logger.Debug().Int("version", version).Msg("Updated existing migration")
				return nil
			}
		}

		mr.migrations = append(mr.migrations, migration)
		mr.logger.Debug().Int("version", version).Str("name", name).Msg("Loaded migration")

		return nil
	})
}

// LoadMigrationsFromDir loads migrations from a directory on the filesystem
func (mr *MigrationRunner) LoadMigrationsFromDir(dir string) error {
	return mr.LoadMigrationsFromFS(os.DirFS(dir), ".")
}

// ensureMigrationsTable creates the migrations table if it doesn't exist
func (mr *MigrationRunner) ensureMigrationsTable(ctx context.Context) error {
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			description TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at ON schema_migrations(applied_at);
	`

	_, err := mr.db.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	return nil
}

// getAppliedMigrations returns a map of applied migration versions
func (mr *MigrationRunner) getAppliedMigrations(ctx context.Context) (map[int]Migration, error) {
	applied := make(map[int]Migration)

	rows, err := mr.db.Query(ctx, "SELECT version, name, applied_at, description FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var migration Migration
		if err := rows.Scan(&migration.Version, &migration.Name, &migration.AppliedAt, &migration.Description); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		applied[migration.Version] = migration
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration rows: %w", err)
	}

	return applied, nil
}

// recordMigration records that a migration has been applied
func (mr *MigrationRunner) recordMigration(ctx context.Context, migration Migration) error {
	sql := `
		INSERT INTO schema_migrations (version, name, description, applied_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (version) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			applied_at = NOW()
	`

	_, err := mr.db.Exec(ctx, sql, migration.Version, migration.Name, migration.Description)
	if err != nil {
		return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
	}

	mr.logger.Info().
		Int("version", migration.Version).
		Str("name", migration.Name).
		Msg("Migration recorded")

	return nil
}

// removeMigration removes a migration record
func (mr *MigrationRunner) removeMigration(ctx context.Context, version int) error {
	sql := `DELETE FROM schema_migrations WHERE version = $1`

	result, err := mr.db.Exec(ctx, sql, version)
	if err != nil {
		return fmt.Errorf("failed to remove migration record %d: %w", version, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("migration %d not found in migrations table", version)
	}

	mr.logger.Info().
		Int("version", version).
		Msg("Migration record removed")

	return nil
}

// Up runs all pending migrations
func (mr *MigrationRunner) Up(ctx context.Context) error {
	// Ensure migrations table exists
	if err := mr.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	// Get applied migrations
	applied, err := mr.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	// Sort migrations by version
	sort.Slice(mr.migrations, func(i, j int) bool {
		return mr.migrations[i].Version < mr.migrations[j].Version
	})

	// Run pending migrations
	for _, migration := range mr.migrations {
		if _, exists := applied[migration.Version]; exists {
			mr.logger.Debug().
				Int("version", migration.Version).
				Msg("Migration already applied, skipping")
			continue
		}

		if migration.UpSQL == "" {
			return fmt.Errorf("migration %d (%s) has no UP SQL", migration.Version, migration.Name)
		}

		mr.logger.Info().
			Int("version", migration.Version).
			Str("name", migration.Name).
			Msg("Running migration UP")

		start := time.Now()
		
		// Run migration in a transaction for atomicity
		// If the migration fails, all changes will be rolled back
		err := mr.runMigrationInTransaction(ctx, migration)
		if err != nil {
			mr.logger.Error().
				Err(err).
				Int("version", migration.Version).
				Str("name", migration.Name).
				Msg("Migration failed and was rolled back")
			return fmt.Errorf("failed to run migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		duration := time.Since(start)
		mr.logger.Info().
			Int("version", migration.Version).
			Str("name", migration.Name).
			Str("duration", duration.String()).
			Msg("Migration UP completed successfully")
	}

	mr.logger.Info().Msg("All migrations completed successfully")
	return nil
}

// runMigrationInTransaction runs a single migration within a transaction
// This ensures that if the migration fails, all changes are rolled back
func (mr *MigrationRunner) runMigrationInTransaction(ctx context.Context, migration Migration) error {
	// Begin transaction
	tx, err := mr.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is rolled back if we return an error
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				mr.logger.Error().
					Err(rbErr).
					Int("version", migration.Version).
					Msg("Failed to rollback transaction")
			}
		}
	}()

	// Execute the migration SQL
	_, err = tx.Exec(ctx, migration.UpSQL)
	if err != nil {
		return fmt.Errorf("migration SQL failed: %w", err)
	}

	// Record the migration in the same transaction
	recordSQL := `
		INSERT INTO schema_migrations (version, name, description, applied_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (version) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			applied_at = NOW()
	`
	_, err = tx.Exec(ctx, recordSQL, migration.Version, migration.Name, migration.Description)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit the transaction
	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	mr.logger.Info().
		Int("version", migration.Version).
		Str("name", migration.Name).
		Msg("Migration recorded")

	return nil
}

// Down rolls back the last N migrations
func (mr *MigrationRunner) Down(ctx context.Context, steps int) error {
	// Ensure migrations table exists
	if err := mr.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	// Get applied migrations
	applied, err := mr.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	// Sort migrations by version in descending order
	sortedMigrations := make([]Migration, 0, len(applied))
	for _, migration := range applied {
		sortedMigrations = append(sortedMigrations, migration)
	}
	sort.Slice(sortedMigrations, func(i, j int) bool {
		return sortedMigrations[i].Version > sortedMigrations[j].Version
	})

	// Roll back the last N migrations
	count := 0
	for _, appliedMigration := range sortedMigrations {
		if count >= steps {
			break
		}

		// Find the corresponding migration with DOWN SQL
		var migration Migration
		found := false
		for _, m := range mr.migrations {
			if m.Version == appliedMigration.Version {
				migration = m
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("migration %d not found in loaded migrations", appliedMigration.Version)
		}

		if migration.DownSQL == "" {
			return fmt.Errorf("migration %d (%s) has no DOWN SQL", migration.Version, migration.Name)
		}

		mr.logger.Info().
			Int("version", migration.Version).
			Str("name", migration.Name).
			Msg("Running migration DOWN")

		start := time.Now()
		
		// Run rollback in a transaction for atomicity
		err := mr.runRollbackInTransaction(ctx, migration)
		if err != nil {
			mr.logger.Error().
				Err(err).
				Int("version", migration.Version).
				Str("name", migration.Name).
				Msg("Migration rollback failed")
			return fmt.Errorf("failed to rollback migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		duration := time.Since(start)
		mr.logger.Info().
			Int("version", migration.Version).
			Str("name", migration.Name).
			Str("duration", duration.String()).
			Msg("Migration DOWN completed successfully")

		count++
	}

	mr.logger.Info().Int("count", count).Msg("Migrations rolled back successfully")
	return nil
}

// runRollbackInTransaction runs a migration rollback within a transaction
func (mr *MigrationRunner) runRollbackInTransaction(ctx context.Context, migration Migration) error {
	// Begin transaction
	tx, err := mr.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is rolled back if we return an error
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				mr.logger.Error().
					Err(rbErr).
					Int("version", migration.Version).
					Msg("Failed to rollback transaction")
			}
		}
	}()

	// Execute the rollback SQL
	_, err = tx.Exec(ctx, migration.DownSQL)
	if err != nil {
		return fmt.Errorf("rollback SQL failed: %w", err)
	}

	// Remove the migration record in the same transaction
	removeSQL := `DELETE FROM schema_migrations WHERE version = $1`
	result, err := tx.Exec(ctx, removeSQL, migration.Version)
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("migration %d not found in migrations table", migration.Version)
	}

	// Commit the transaction
	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	mr.logger.Info().
		Int("version", migration.Version).
		Msg("Migration record removed")

	return nil
}

// Status returns the status of all migrations
func (mr *MigrationRunner) Status(ctx context.Context) ([]MigrationStatus, error) {
	// Ensure migrations table exists
	if err := mr.ensureMigrationsTable(ctx); err != nil {
		return nil, err
	}

	// Get applied migrations
	applied, err := mr.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	// Create status for all migrations
	sort.Slice(mr.migrations, func(i, j int) bool {
		return mr.migrations[i].Version < mr.migrations[j].Version
	})

	statuses := make([]MigrationStatus, 0, len(mr.migrations))
	for _, migration := range mr.migrations {
		appliedMigration, exists := applied[migration.Version]
		status := MigrationStatus{
			Version:     migration.Version,
			Name:        migration.Name,
			Description: migration.Description,
			Applied:     exists,
		}

		if exists {
			status.AppliedAt = appliedMigration.AppliedAt
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version     int
	Name        string
	Description string
	Applied     bool
	AppliedAt   *time.Time
}

// GetPendingMigrations returns the number of pending migrations
func (mr *MigrationRunner) GetPendingMigrations(ctx context.Context) (int, error) {
	statuses, err := mr.Status(ctx)
	if err != nil {
		return 0, err
	}

	pending := 0
	for _, status := range statuses {
		if !status.Applied {
			pending++
		}
	}

	return pending, nil
}

// RequireNoP endingMigrations checks if there are pending migrations and returns an error if there are
// This should be called during application startup to prevent running with an outdated schema
func (mr *MigrationRunner) RequireNoPendingMigrations(ctx context.Context) error {
	pending, err := mr.GetPendingMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to check pending migrations: %w", err)
	}

	if pending > 0 {
		return fmt.Errorf("cannot start application: %d pending migration(s) must be applied first", pending)
	}

	return nil
}

// GetPendingMigrationsList returns a list of pending migrations
func (mr *MigrationRunner) GetPendingMigrationsList(ctx context.Context) ([]Migration, error) {
	statuses, err := mr.Status(ctx)
	if err != nil {
		return nil, err
	}

	var pending []Migration
	for _, status := range statuses {
		if !status.Applied {
			// Find the migration
			for _, m := range mr.migrations {
				if m.Version == status.Version {
					pending = append(pending, m)
					break
				}
			}
		}
	}

	return pending, nil
}