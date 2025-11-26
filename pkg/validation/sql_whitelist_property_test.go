package validation

import (
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: production-readiness, Property 4: SQL Column Whitelisting**
// For any dynamic SQL query with ORDER BY clause, if the column name is not in the whitelist,
// the system must reject the query
// **Validates: Requirements 2.2**
func TestProperty_SQLColumnWhitelisting(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Define a whitelist for testing
	allowedColumns := []string{"id", "name", "email", "created_at", "updated_at"}
	whitelist := NewSQLColumnWhitelist(allowedColumns)

	// Property: Columns in the whitelist must be accepted
	properties.Property("whitelisted columns are accepted", prop.ForAll(
		func(column string) bool {
			err := whitelist.ValidateColumn(column)
			// Column should be accepted (no error)
			return err == nil
		},
		genWhitelistedColumn(allowedColumns),
	))

	// Property: Columns NOT in the whitelist must be rejected
	properties.Property("non-whitelisted columns are rejected", prop.ForAll(
		func(column string) bool {
			// Skip empty strings as they have special handling
			if column == "" {
				return true
			}

			// Skip if column happens to be in whitelist
			normalized := strings.ToLower(strings.TrimSpace(column))
			for _, allowed := range allowedColumns {
				if normalized == strings.ToLower(allowed) {
					return true
				}
			}

			err := whitelist.ValidateColumn(column)
			// Column should be rejected (error expected)
			return err != nil
		},
		genNonWhitelistedColumn(allowedColumns),
	))

	// Property: Empty column names must be rejected
	properties.Property("empty columns are rejected", prop.ForAll(
		func() bool {
			err := whitelist.ValidateColumn("")
			// Empty column should be rejected
			return err != nil
		},
	))

	// Property: ORDER BY clauses with whitelisted columns must be accepted
	properties.Property("ORDER BY with whitelisted columns accepted", prop.ForAll(
		func(columns []string) bool {
			// Build ORDER BY clause from whitelisted columns
			orderBy := buildOrderByClause(columns)
			err := whitelist.ValidateOrderByClause(orderBy)
			// Should be accepted
			return err == nil
		},
		genWhitelistedOrderBy(allowedColumns),
	))

	// Property: ORDER BY clauses with non-whitelisted columns must be rejected
	properties.Property("ORDER BY with non-whitelisted columns rejected", prop.ForAll(
		func(badColumn string) bool {
			// Skip empty strings
			if badColumn == "" {
				return true
			}

			// Skip if column happens to be in whitelist
			normalized := strings.ToLower(strings.TrimSpace(badColumn))
			for _, allowed := range allowedColumns {
				if normalized == strings.ToLower(allowed) {
					return true
				}
			}

			// Create ORDER BY with a non-whitelisted column
			orderBy := badColumn + " DESC"
			err := whitelist.ValidateOrderByClause(orderBy)
			// Should be rejected
			return err != nil
		},
		genNonWhitelistedColumn(allowedColumns),
	))

	// Property: Case-insensitive matching - uppercase/lowercase variants should work
	properties.Property("case-insensitive column matching", prop.ForAll(
		func(column string, useUpper bool) bool {
			testColumn := column
			if useUpper {
				testColumn = strings.ToUpper(column)
			} else {
				testColumn = strings.ToLower(column)
			}

			err := whitelist.ValidateColumn(testColumn)
			// Should be accepted regardless of case
			return err == nil
		},
		genWhitelistedColumn(allowedColumns),
		gen.Bool(),
	))

	// Property: ORDER BY with ASC/DESC should work correctly
	properties.Property("ORDER BY with ASC/DESC modifiers", prop.ForAll(
		func(column string, direction string) bool {
			orderBy := column + " " + direction
			err := whitelist.ValidateOrderByClause(orderBy)
			// Should be accepted
			return err == nil
		},
		genWhitelistedColumn(allowedColumns),
		gen.OneConstOf("ASC", "DESC", "asc", "desc"),
	))

	// Property: Multiple columns in ORDER BY should all be validated
	properties.Property("multiple columns in ORDER BY validated", prop.ForAll(
		func(columns []string, hasBadColumn bool) bool {
			var orderByParts []string

			if hasBadColumn && len(columns) > 0 {
				// Add whitelisted columns
				for i := 0; i < len(columns)-1; i++ {
					orderByParts = append(orderByParts, columns[i])
				}
				// Add a non-whitelisted column
				orderByParts = append(orderByParts, "malicious_column")
			} else {
				// All whitelisted columns
				orderByParts = columns
			}

			orderBy := strings.Join(orderByParts, ", ")
			err := whitelist.ValidateOrderByClause(orderBy)

			if hasBadColumn && len(columns) > 0 {
				// Should be rejected due to bad column
				return err != nil
			}
			// Should be accepted
			return err == nil
		},
		genWhitelistedOrderBy(allowedColumns),
		gen.Bool(),
	))

	// Property: SQL injection attempts should be rejected
	properties.Property("SQL injection attempts rejected", prop.ForAll(
		func(injection string) bool {
			err := whitelist.ValidateColumn(injection)
			// Should be rejected
			return err != nil
		},
		genSQLInjectionAttempt(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Generator for whitelisted columns
func genWhitelistedColumn(allowedColumns []string) gopter.Gen {
	if len(allowedColumns) == 0 {
		return gen.Const("id")
	}
	// Convert to interface slice for OneConstOf
	values := make([]interface{}, len(allowedColumns))
	for i, col := range allowedColumns {
		values[i] = col
	}
	return gen.OneConstOf(values...)
}

// Generator for non-whitelisted columns
func genNonWhitelistedColumn(allowedColumns []string) gopter.Gen {
	// Create a map for quick lookup
	allowed := make(map[string]bool)
	for _, col := range allowedColumns {
		allowed[strings.ToLower(col)] = true
	}

	return gen.OneGenOf(
		// Random strings that are likely not in whitelist
		gen.AlphaString().SuchThat(func(s string) bool {
			if s == "" {
				return false
			}
			return !allowed[strings.ToLower(s)]
		}),
		// Common SQL column names not in our whitelist
		gen.OneConstOf(
			"password",
			"secret_key",
			"admin",
			"user_id",
			"deleted",
			"status",
			"role",
			"permissions",
			"token",
		),
	)
}

// Generator for ORDER BY clauses with whitelisted columns
func genWhitelistedOrderBy(allowedColumns []string) gopter.Gen {
	return gen.SliceOfN(
		3,
		genWhitelistedColumn(allowedColumns),
	).SuchThat(func(cols []string) bool {
		return len(cols) > 0 && len(cols) <= 3
	})
}

// Generator for SQL injection attempts
func genSQLInjectionAttempt() gopter.Gen {
	return gen.OneConstOf(
		"id; DROP TABLE users--",
		"id' OR '1'='1",
		"id UNION SELECT * FROM passwords",
		"id; DELETE FROM users WHERE 1=1--",
		"id' AND 1=1--",
		"id/**/OR/**/1=1",
		"id%20OR%201=1",
		"id' OR 'x'='x",
		"id); DROP TABLE users;--",
		"id' UNION ALL SELECT NULL--",
	)
}

// Helper function to build ORDER BY clause from columns
func buildOrderByClause(columns []string) string {
	if len(columns) == 0 {
		return ""
	}

	var parts []string
	for _, col := range columns {
		parts = append(parts, col)
	}
	return strings.Join(parts, ", ")
}
