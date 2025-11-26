package validation

import (
	"fmt"
	"strings"
)

// SQLColumnWhitelist validates SQL column names against a whitelist
type SQLColumnWhitelist struct {
	AllowedColumns map[string]bool
}

// NewSQLColumnWhitelist creates a new SQL column whitelist validator
func NewSQLColumnWhitelist(columns []string) *SQLColumnWhitelist {
	allowed := make(map[string]bool)
	for _, col := range columns {
		// Store both original and lowercase versions for case-insensitive matching
		allowed[col] = true
		allowed[strings.ToLower(col)] = true
	}

	return &SQLColumnWhitelist{
		AllowedColumns: allowed,
	}
}

// ValidateColumn validates a single column name
func (w *SQLColumnWhitelist) ValidateColumn(column string) error {
	if column == "" {
		return fmt.Errorf("column name cannot be empty")
	}

	// Normalize column name (remove quotes, trim spaces)
	normalized := normalizeColumnName(column)

	// Check if column is in whitelist
	if !w.AllowedColumns[normalized] && !w.AllowedColumns[strings.ToLower(normalized)] {
		return fmt.Errorf("column '%s' is not in the allowed list", column)
	}

	return nil
}

// ValidateColumns validates multiple column names
func (w *SQLColumnWhitelist) ValidateColumns(columns []string) error {
	for _, col := range columns {
		if err := w.ValidateColumn(col); err != nil {
			return err
		}
	}
	return nil
}

// ValidateOrderByClause validates an ORDER BY clause
func (w *SQLColumnWhitelist) ValidateOrderByClause(orderBy string) error {
	if orderBy == "" {
		return nil // Empty ORDER BY is valid
	}

	// Split by comma for multiple columns
	parts := strings.Split(orderBy, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Extract column name (remove ASC/DESC)
		column := extractColumnName(part)

		if err := w.ValidateColumn(column); err != nil {
			return fmt.Errorf("invalid ORDER BY clause: %w", err)
		}
	}

	return nil
}

// normalizeColumnName normalizes a column name by removing quotes and trimming
func normalizeColumnName(column string) string {
	// Remove quotes
	column = strings.Trim(column, `"'` + "`")

	// Trim whitespace
	column = strings.TrimSpace(column)

	return column
}

// extractColumnName extracts the column name from an ORDER BY part
func extractColumnName(part string) string {
	// Remove ASC/DESC
	part = strings.TrimSpace(part)
	part = strings.TrimSuffix(strings.ToUpper(part), " ASC")
	part = strings.TrimSuffix(strings.ToUpper(part), " DESC")
	part = strings.TrimSuffix(part, " ASC")
	part = strings.TrimSuffix(part, " DESC")
	part = strings.TrimSuffix(part, " asc")
	part = strings.TrimSuffix(part, " desc")

	// Handle "column ASC" or "column DESC" patterns
	words := strings.Fields(part)
	if len(words) > 0 {
		return normalizeColumnName(words[0])
	}

	return normalizeColumnName(part)
}

// Common table column whitelists

// UserTableColumns returns allowed columns for users table
func UserTableColumns() []string {
	return []string{
		"id", "email", "username", "first_name", "last_name",
		"is_active", "is_verified", "created_at", "updated_at",
		"last_login_at", "password_changed_at",
	}
}

// ProductTableColumns returns allowed columns for products table
func ProductTableColumns() []string {
	return []string{
		"id", "name", "description", "sku", "price", "cost",
		"category_id", "is_active", "stock_quantity", "reorder_level",
		"created_at", "updated_at", "deleted_at",
	}
}

// OrderTableColumns returns allowed columns for orders table
func OrderTableColumns() []string {
	return []string{
		"id", "order_number", "customer_id", "status", "total_amount",
		"subtotal", "tax_amount", "shipping_amount", "discount_amount",
		"payment_status", "payment_method", "shipping_method",
		"created_at", "updated_at", "completed_at", "cancelled_at",
	}
}

// CustomerTableColumns returns allowed columns for customers table
func CustomerTableColumns() []string {
	return []string{
		"id", "email", "first_name", "last_name", "company",
		"phone", "is_active", "created_at", "updated_at",
	}
}

// InventoryTableColumns returns allowed columns for inventory table
func InventoryTableColumns() []string {
	return []string{
		"id", "product_id", "warehouse_id", "quantity",
		"reserved_quantity", "available_quantity",
		"created_at", "updated_at",
	}
}

// RoleTableColumns returns allowed columns for roles table
func RoleTableColumns() []string {
	return []string{
		"id", "name", "description", "is_system",
		"created_at", "updated_at",
	}
}

// CategoryTableColumns returns allowed columns for product_categories table
func CategoryTableColumns() []string {
	return []string{
		"id", "name", "description", "parent_id", "slug",
		"is_active", "display_order", "created_at", "updated_at",
	}
}

// WarehouseTableColumns returns allowed columns for warehouses table
func WarehouseTableColumns() []string {
	return []string{
		"id", "name", "code", "address", "city", "state",
		"country", "postal_code", "is_active",
		"created_at", "updated_at",
	}
}

// NewUserColumnWhitelist creates a whitelist for user table columns
func NewUserColumnWhitelist() *SQLColumnWhitelist {
	return NewSQLColumnWhitelist(UserTableColumns())
}

// NewProductColumnWhitelist creates a whitelist for product table columns
func NewProductColumnWhitelist() *SQLColumnWhitelist {
	return NewSQLColumnWhitelist(ProductTableColumns())
}

// NewOrderColumnWhitelist creates a whitelist for order table columns
func NewOrderColumnWhitelist() *SQLColumnWhitelist {
	return NewSQLColumnWhitelist(OrderTableColumns())
}

// NewCustomerColumnWhitelist creates a whitelist for customer table columns
func NewCustomerColumnWhitelist() *SQLColumnWhitelist {
	return NewSQLColumnWhitelist(CustomerTableColumns())
}

// NewInventoryColumnWhitelist creates a whitelist for inventory table columns
func NewInventoryColumnWhitelist() *SQLColumnWhitelist {
	return NewSQLColumnWhitelist(InventoryTableColumns())
}

// NewRoleColumnWhitelist creates a whitelist for role table columns
func NewRoleColumnWhitelist() *SQLColumnWhitelist {
	return NewSQLColumnWhitelist(RoleTableColumns())
}

// NewCategoryColumnWhitelist creates a whitelist for category table columns
func NewCategoryColumnWhitelist() *SQLColumnWhitelist {
	return NewSQLColumnWhitelist(CategoryTableColumns())
}

// NewWarehouseColumnWhitelist creates a whitelist for warehouse table columns
func NewWarehouseColumnWhitelist() *SQLColumnWhitelist {
	return NewSQLColumnWhitelist(WarehouseTableColumns())
}
