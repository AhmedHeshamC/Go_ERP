# Migration Property-Based Test

## Overview

This document describes the property-based test for database migration transaction rollback (Property 17).

## Property Being Tested

**Property 17: Migration Transaction Rollback**

For any database migration that fails, all changes made by that migration must be rolled back.

**Validates: Requirements 19.2**

## Test Description

The property test (`TestProperty_MigrationTransactionRollback`) verifies that when a migration fails:

1. Any database changes made before the failure are rolled back
2. The migration is not recorded in the `schema_migrations` table
3. The database remains in a consistent state

## Test Strategy

The test uses property-based testing with `gopter` to generate random table names and create migrations that:

1. Successfully create a table (CREATE TABLE statement)
2. Then fail with invalid SQL syntax

The property being tested is: **If the migration fails, the table should NOT exist** (because the transaction should have been rolled back).

## Current Status

**✅ IMPLEMENTED** - Transaction support has been added to the migration system:

1. ✅ Write failing test (property-based test)
2. ✅ Implement transaction support in migrations (task 21.1)
3. ⏳ Verify test passes (requires test database)

## Running the Test

### Prerequisites

You need a running PostgreSQL test database. Set up using one of these methods:

#### Option 1: Using Docker

```bash
# Start a test database
docker run --name postgres-test -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=erpgo_test -p 5433:5432 -d postgres:15-alpine

# Set the test database URL
export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/erpgo_test?sslmode=disable"
```

#### Option 2: Using Existing PostgreSQL

```bash
# Create test database
createdb erpgo_test

# Set the test database URL
export TEST_DATABASE_URL="postgres://your_user:your_password@localhost:5432/erpgo_test?sslmode=disable"
```

### Run the Test

```bash
# Run the property test
go test -v -run TestProperty_MigrationTransactionRollback ./pkg/database/

# Run with more iterations (default is 100)
go test -v -run TestProperty_MigrationTransactionRollback ./pkg/database/ -gopter.minSuccessfulTests=1000
```

### Expected Output

With the transaction support implemented, the test should PASS:

```
+ failed migrations rollback all changes: OK, passed 100 tests.
```

If the test fails, it indicates a regression in the transaction rollback functionality.

## Implementation Notes

The migration system now implements transaction support:

1. ✅ Each migration is wrapped in a database transaction (`runMigrationInTransaction`)
2. ✅ If the migration fails, the transaction is rolled back automatically
3. ✅ The migration is only recorded in `schema_migrations` within the same transaction
4. ✅ Both the migration SQL and the recording step are atomic

### Key Implementation Details

- **`runMigrationInTransaction`**: Wraps migration execution in a transaction
- **`runRollbackInTransaction`**: Wraps migration rollback in a transaction
- **Deferred rollback**: Uses `defer` to ensure rollback on error
- **Atomic recording**: Migration record is inserted in the same transaction as the migration SQL

### Additional Features Implemented

- **`RequireNoPendingMigrations`**: Prevents application startup with pending migrations
- **`GetPendingMigrationsList`**: Returns list of pending migrations for reporting

## Related Files

- `pkg/database/migrations.go` - Migration runner implementation (needs transaction support)
- `pkg/database/migrations_test.go` - Unit tests for migrations
- `pkg/database/migrations_property_test.go` - This property-based test
- `.kiro/specs/production-readiness/tasks.md` - Task 21 and 21.1
- `.kiro/specs/production-readiness/design.md` - Property 17 specification
