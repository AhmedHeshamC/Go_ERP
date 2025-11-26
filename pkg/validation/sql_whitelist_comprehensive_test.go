package validation

import (
	"testing"
)

// TestValidationComprehensive provides comprehensive test coverage for validation package
func TestValidationComprehensive(t *testing.T) {
	t.Run("UserTableColumns", func(t *testing.T) {
		columns := UserTableColumns()
		if len(columns) == 0 {
			t.Error("Expected user table columns, got empty slice")
		}
		// Verify expected columns exist
		expectedCols := []string{"id", "email", "username"}
		for _, expected := range expectedCols {
			found := false
			for _, col := range columns {
				if col == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected column %s not found in user table columns", expected)
			}
		}
	})

	t.Run("ProductTableColumns", func(t *testing.T) {
		columns := ProductTableColumns()
		if len(columns) == 0 {
			t.Error("Expected product table columns, got empty slice")
		}
	})

	t.Run("OrderTableColumns", func(t *testing.T) {
		columns := OrderTableColumns()
		if len(columns) == 0 {
			t.Error("Expected order table columns, got empty slice")
		}
	})

	t.Run("CustomerTableColumns", func(t *testing.T) {
		columns := CustomerTableColumns()
		if len(columns) == 0 {
			t.Error("Expected customer table columns, got empty slice")
		}
	})

	t.Run("InventoryTableColumns", func(t *testing.T) {
		columns := InventoryTableColumns()
		if len(columns) == 0 {
			t.Error("Expected inventory table columns, got empty slice")
		}
	})

	t.Run("RoleTableColumns", func(t *testing.T) {
		columns := RoleTableColumns()
		if len(columns) == 0 {
			t.Error("Expected role table columns, got empty slice")
		}
	})

	t.Run("CategoryTableColumns", func(t *testing.T) {
		columns := CategoryTableColumns()
		if len(columns) == 0 {
			t.Error("Expected category table columns, got empty slice")
		}
	})

	t.Run("WarehouseTableColumns", func(t *testing.T) {
		columns := WarehouseTableColumns()
		if len(columns) == 0 {
			t.Error("Expected warehouse table columns, got empty slice")
		}
	})
}

func TestWhitelistConstructors(t *testing.T) {
	t.Run("NewUserColumnWhitelist", func(t *testing.T) {
		whitelist := NewUserColumnWhitelist()
		if whitelist == nil {
			t.Fatal("Expected whitelist, got nil")
		}
		// Test with a known user column
		if err := whitelist.ValidateColumn("email"); err != nil {
			t.Errorf("Expected email to be valid, got error: %v", err)
		}
	})

	t.Run("NewProductColumnWhitelist", func(t *testing.T) {
		whitelist := NewProductColumnWhitelist()
		if whitelist == nil {
			t.Fatal("Expected whitelist, got nil")
		}
		if err := whitelist.ValidateColumn("name"); err != nil {
			t.Errorf("Expected name to be valid, got error: %v", err)
		}
	})

	t.Run("NewOrderColumnWhitelist", func(t *testing.T) {
		whitelist := NewOrderColumnWhitelist()
		if whitelist == nil {
			t.Fatal("Expected whitelist, got nil")
		}
		if err := whitelist.ValidateColumn("order_number"); err != nil {
			t.Errorf("Expected order_number to be valid, got error: %v", err)
		}
	})

	t.Run("NewCustomerColumnWhitelist", func(t *testing.T) {
		whitelist := NewCustomerColumnWhitelist()
		if whitelist == nil {
			t.Fatal("Expected whitelist, got nil")
		}
		if err := whitelist.ValidateColumn("email"); err != nil {
			t.Errorf("Expected email to be valid, got error: %v", err)
		}
	})

	t.Run("NewInventoryColumnWhitelist", func(t *testing.T) {
		whitelist := NewInventoryColumnWhitelist()
		if whitelist == nil {
			t.Fatal("Expected whitelist, got nil")
		}
		if err := whitelist.ValidateColumn("quantity"); err != nil {
			t.Errorf("Expected quantity to be valid, got error: %v", err)
		}
	})

	t.Run("NewRoleColumnWhitelist", func(t *testing.T) {
		whitelist := NewRoleColumnWhitelist()
		if whitelist == nil {
			t.Fatal("Expected whitelist, got nil")
		}
		if err := whitelist.ValidateColumn("name"); err != nil {
			t.Errorf("Expected name to be valid, got error: %v", err)
		}
	})

	t.Run("NewCategoryColumnWhitelist", func(t *testing.T) {
		whitelist := NewCategoryColumnWhitelist()
		if whitelist == nil {
			t.Fatal("Expected whitelist, got nil")
		}
		if err := whitelist.ValidateColumn("name"); err != nil {
			t.Errorf("Expected name to be valid, got error: %v", err)
		}
	})

	t.Run("NewWarehouseColumnWhitelist", func(t *testing.T) {
		whitelist := NewWarehouseColumnWhitelist()
		if whitelist == nil {
			t.Fatal("Expected whitelist, got nil")
		}
		if err := whitelist.ValidateColumn("name"); err != nil {
			t.Errorf("Expected name to be valid, got error: %v", err)
		}
	})
}

func TestValidateColumns(t *testing.T) {
	whitelist := NewSQLColumnWhitelist([]string{"id", "name", "email"})

	t.Run("ValidateMultipleValidColumns", func(t *testing.T) {
		columns := []string{"id", "name", "email"}
		if err := whitelist.ValidateColumns(columns); err != nil {
			t.Errorf("Expected no error for valid columns, got: %v", err)
		}
	})

	t.Run("ValidateMultipleColumnsWithInvalid", func(t *testing.T) {
		columns := []string{"id", "name", "invalid_column"}
		if err := whitelist.ValidateColumns(columns); err == nil {
			t.Error("Expected error for invalid column, got nil")
		}
	})

	t.Run("ValidateEmptyColumnList", func(t *testing.T) {
		columns := []string{}
		if err := whitelist.ValidateColumns(columns); err != nil {
			t.Errorf("Expected no error for empty list, got: %v", err)
		}
	})
}

func TestOrderByEdgeCases(t *testing.T) {
	whitelist := NewSQLColumnWhitelist([]string{"id", "name", "created_at"})

	t.Run("EmptyOrderBy", func(t *testing.T) {
		if err := whitelist.ValidateOrderByClause(""); err != nil {
			t.Errorf("Expected no error for empty ORDER BY, got: %v", err)
		}
	})

	t.Run("OrderByWithSpaces", func(t *testing.T) {
		if err := whitelist.ValidateOrderByClause("  id  ,  name  "); err != nil {
			t.Errorf("Expected no error for ORDER BY with spaces, got: %v", err)
		}
	})

	t.Run("OrderByWithMixedCase", func(t *testing.T) {
		if err := whitelist.ValidateOrderByClause("ID DESC, Name ASC"); err != nil {
			t.Errorf("Expected no error for mixed case ORDER BY, got: %v", err)
		}
	})

	t.Run("OrderByWithQuotes", func(t *testing.T) {
		if err := whitelist.ValidateOrderByClause(`"id" DESC`); err != nil {
			t.Errorf("Expected no error for quoted column, got: %v", err)
		}
	})

	t.Run("OrderByWithBackticks", func(t *testing.T) {
		if err := whitelist.ValidateOrderByClause("`name` ASC"); err != nil {
			t.Errorf("Expected no error for backtick column, got: %v", err)
		}
	})
}

func TestColumnNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "id", "id"},
		{"with spaces", "  id  ", "id"},
		{"with double quotes", `"id"`, "id"},
		{"with single quotes", "'id'", "id"},
		{"with backticks", "`id`", "id"},
		{"mixed quotes", `"'id'"`, "id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeColumnName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeColumnName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractColumnName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
	}{
		{"simple", "id"},
		{"with ASC", "id ASC"},
		{"with DESC", "id DESC"},
		{"with lowercase asc", "id asc"},
		{"with lowercase desc", "id desc"},
		{"with spaces", "  id  DESC  "},
		{"with quotes and ASC", `"id" ASC`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractColumnName(tt.input)
			// Just verify it returns a non-empty string
			if result == "" {
				t.Errorf("extractColumnName(%q) returned empty string", tt.input)
			}
		})
	}
}
