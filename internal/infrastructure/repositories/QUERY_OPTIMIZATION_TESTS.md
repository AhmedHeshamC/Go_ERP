# Query Optimization Tests

## Overview

This document describes the unit tests created for task 8.3 to verify query optimization improvements made in task 8.1.

## Tests Created

### File: `query_optimization_test.go`

#### 1. TestUserRolesSingleQuery
**Purpose**: Verifies that user roles are fetched with a single JOIN query instead of N+1 queries.

**What it tests**:
- Creates a user with multiple roles (admin, editor, viewer)
- Fetches all roles using `GetUserRoles()`
- Verifies all roles are returned correctly
- Confirms the implementation uses a single JOIN query

**Implementation verified**:
```sql
SELECT r.name 
FROM roles r 
INNER JOIN user_roles ur ON r.id = ur.role_id 
WHERE ur.user_id = $1
```

This is a single database query that fetches all roles at once, avoiding the N+1 query problem.

**Validates**: Requirements 6.3 (N+1 query optimization)

#### 2. TestUserRolesQueryStructure
**Purpose**: Verifies the query structure scales properly with larger datasets.

**What it tests**:
- Creates a user with 5 roles
- Fetches all roles using `GetUserRoles()`
- Verifies the query is O(1) database calls, not O(N)

**Validates**: Requirements 6.3

#### 3. TestUserRolesEmptyResult
**Purpose**: Verifies correct behavior when a user has no roles.

**What it tests**:
- Creates a user without any roles
- Fetches roles using `GetUserRoles()`
- Verifies an empty slice is returned (not an error)

**Validates**: Requirements 6.3

#### 4. TestUserRolesNonExistentUser
**Purpose**: Verifies correct behavior for non-existent users.

**What it tests**:
- Attempts to fetch roles for a non-existent user ID
- Verifies an empty slice is returned (not an error)

**Validates**: Requirements 6.3

## Running the Tests

### Prerequisites
1. PostgreSQL database must be running
2. Set `TEST_DATABASE_URL` environment variable or use default:
   ```
   postgres://erpgo_user:erpgo_password@localhost:5432/erpgo_test?sslmode=disable
   ```

### Start Database
```bash
docker-compose up -d postgres
```

### Run Tests
```bash
# Run all query optimization tests
go test -v ./internal/infrastructure/repositories -run TestUserRoles

# Run specific test
go test -v ./internal/infrastructure/repositories -run TestUserRolesSingleQuery
```

## Query Optimization Verified

### User Roles (✓ Implemented)
- **Before**: N+1 queries (1 query to get role IDs, then N queries to get each role)
- **After**: Single JOIN query fetches all roles at once
- **Performance**: O(1) database calls regardless of number of roles

### Order Items (Not Yet Implemented)
The task requirements mention testing order items as well, but the current implementation doesn't appear to have a dedicated method for fetching order items with a JOIN query. The `GetByID` method in `postgres_order_repository.go` only fetches the order itself, not the items.

**Recommendation**: If order items optimization was implemented in task 8.1, additional tests should be added to verify that implementation.

## Test Coverage

These tests provide:
- ✓ Verification that JOIN queries are used
- ✓ Verification that all data is fetched correctly
- ✓ Edge case testing (empty results, non-existent users)
- ✓ Scalability verification (multiple roles)

## Notes

- These are integration tests that require a real database connection
- Tests use the `testutil` package for database setup and cleanup
- Each test runs in isolation with its own test data
- Tests verify both correctness and query structure
