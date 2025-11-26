package repositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"erpgo/internal/domain/users/entities"
	"erpgo/internal/domain/users/repositories"
	"erpgo/pkg/database"
	"erpgo/pkg/validation"
)

// PostgresUserRepository implements UserRepository for PostgreSQL
type PostgresUserRepository struct {
	db              *database.Database
	logger          interface{} // Can be zerolog.Logger or any logger
	columnWhitelist *validation.SQLColumnWhitelist
}

// NewPostgresUserRepository creates a new PostgreSQL user repository
func NewPostgresUserRepository(db *database.Database) *PostgresUserRepository {
	return &PostgresUserRepository{
		db:              db,
		columnWhitelist: validation.NewUserColumnWhitelist(),
	}
}

// validateAndBuildOrderBy validates sort parameters and builds ORDER BY clause
func (r *PostgresUserRepository) validateAndBuildOrderBy(sortBy, sortOrder string, defaultSort string) (string, error) {
	// Set default sort column
	if sortBy == "" {
		sortBy = defaultSort
	} else {
		// Validate column name against whitelist
		if err := r.columnWhitelist.ValidateColumn(sortBy); err != nil {
			return "", fmt.Errorf("invalid sort column: %w", err)
		}
	}

	// Set default sort order
	if sortOrder == "" {
		sortOrder = "DESC"
	} else {
		sortOrder = strings.ToUpper(sortOrder)
		// Validate sort order
		if sortOrder != "ASC" && sortOrder != "DESC" {
			return "", fmt.Errorf("invalid sort order: must be ASC or DESC")
		}
	}

	return fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder), nil
}

// PostgresRoleRepository implements RoleRepository for PostgreSQL
type PostgresRoleRepository struct {
	db              *database.Database
	columnWhitelist *validation.SQLColumnWhitelist
}

// NewPostgresRoleRepository creates a new PostgreSQL role repository
func NewPostgresRoleRepository(db *database.Database) *PostgresRoleRepository {
	return &PostgresRoleRepository{
		db:              db,
		columnWhitelist: validation.NewRoleColumnWhitelist(),
	}
}

// validateAndBuildOrderByForRole validates sort parameters and builds ORDER BY clause for roles
func (r *PostgresRoleRepository) validateAndBuildOrderByForRole(sortBy, sortOrder string, defaultSort string) (string, error) {
	// Set default sort column
	if sortBy == "" {
		sortBy = defaultSort
	} else {
		// Validate column name against whitelist
		if err := r.columnWhitelist.ValidateColumn(sortBy); err != nil {
			return "", fmt.Errorf("invalid sort column: %w", err)
		}
	}

	// Set default sort order
	if sortOrder == "" {
		sortOrder = "DESC"
	} else {
		sortOrder = strings.ToUpper(sortOrder)
		// Validate sort order
		if sortOrder != "ASC" && sortOrder != "DESC" {
			return "", fmt.Errorf("invalid sort order: must be ASC or DESC")
		}
	}

	return fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder), nil
}

// PostgresUserRoleRepository implements UserRoleRepository for PostgreSQL
type PostgresUserRoleRepository struct {
	db *database.Database
}

// NewPostgresUserRoleRepository creates a new PostgreSQL user-role repository
func NewPostgresUserRoleRepository(db *database.Database) *PostgresUserRoleRepository {
	return &PostgresUserRoleRepository{
		db: db,
	}
}

// Create creates a new user
func (r *PostgresUserRepository) Create(ctx context.Context, user *entities.User) error {
	query := `
		INSERT INTO users (id, email, username, password_hash, first_name, last_name, phone, is_active, is_verified, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Username,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.Phone,
		user.IsActive,
		user.IsVerified,
		user.LastLoginAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, phone,
		       is_active, is_verified, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &entities.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.IsActive,
		&user.IsVerified,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, phone,
		       is_active, is_verified, last_login_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &entities.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.IsActive,
		&user.IsVerified,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user with email %s not found", email)
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetByUsername retrieves a user by username
func (r *PostgresUserRepository) GetByUsername(ctx context.Context, username string) (*entities.User, error) {
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, phone,
		       is_active, is_verified, last_login_at, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	user := &entities.User{}
	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.IsActive,
		&user.IsVerified,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user with username %s not found", username)
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return user, nil
}

// Update updates a user
func (r *PostgresUserRepository) Update(ctx context.Context, user *entities.User) error {
	query := `
		UPDATE users
		SET email = $2, username = $3, password_hash = $4, first_name = $5,
		    last_name = $6, phone = $7, is_active = $8, is_verified = $9,
		    last_login_at = $10
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Username,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.Phone,
		user.IsActive,
		user.IsVerified,
		user.LastLoginAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// Delete deletes a user
func (r *PostgresUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// List retrieves a list of users
func (r *PostgresUserRepository) List(ctx context.Context, filter repositories.UserFilter) ([]*entities.User, error) {
	// Build the base query
	baseQuery := `
		SELECT id, email, username, password_hash, first_name, last_name, phone,
		       is_active, is_verified, last_login_at, created_at, updated_at
		FROM users
		WHERE 1=1
	`

	// Build WHERE conditions
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(email ILIKE $%d OR username ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)", argIndex, argIndex+1, argIndex+2, argIndex+3))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
		argIndex += 4
	}

	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.IsVerified != nil {
		conditions = append(conditions, fmt.Sprintf("is_verified = $%d", argIndex))
		args = append(args, *filter.IsVerified)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY with validation
	orderByClause, err := r.validateAndBuildOrderBy(filter.SortBy, filter.SortOrder, "created_at")
	if err != nil {
		return nil, err
	}
	baseQuery += orderByClause

	// Add LIMIT and OFFSET for pagination
	if filter.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++

		if filter.Page > 1 {
			offset := (filter.Page - 1) * filter.Limit
			baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, offset)
		}
	}

	// Execute query
	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*entities.User
	for rows.Next() {
		user := &entities.User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Username,
			&user.PasswordHash,
			&user.FirstName,
			&user.LastName,
			&user.Phone,
			&user.IsActive,
			&user.IsVerified,
			&user.LastLoginAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}

// Count returns the count of users matching the filter
func (r *PostgresUserRepository) Count(ctx context.Context, filter repositories.UserFilter) (int, error) {
	baseQuery := `SELECT COUNT(*) FROM users WHERE 1=1`

	// Build WHERE conditions (same as in List)
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(email ILIKE $%d OR username ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)", argIndex, argIndex+1, argIndex+2, argIndex+3))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
		argIndex += 4
	}

	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.IsVerified != nil {
		conditions = append(conditions, fmt.Sprintf("is_verified = $%d", argIndex))
		args = append(args, *filter.IsVerified)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	var count int
	err := r.db.QueryRow(ctx, baseQuery, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// ExistsByEmail checks if a user exists by email
func (r *PostgresUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists by email: %w", err)
	}

	return exists, nil
}

// ExistsByUsername checks if a user exists by username
func (r *PostgresUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists by username: %w", err)
	}

	return exists, nil
}

// UpdateLastLogin updates the user's last login time
func (r *PostgresUserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// GetUserRoles retrieves roles for a user
func (r *PostgresUserRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `
		SELECT r.name
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		err := rows.Scan(&role)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role row: %w", err)
		}
		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating role rows: %w", err)
	}

	return roles, nil
}

// AssignRole assigns a role to a user
func (r *PostgresUserRepository) AssignRole(ctx context.Context, userID uuid.UUID, roleName string, assignedBy uuid.UUID) error {
	// First get the role ID
	var roleID uuid.UUID
	roleQuery := `SELECT id FROM roles WHERE name = $1`
	err := r.db.QueryRow(ctx, roleQuery, roleName).Scan(&roleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("role %s not found", roleName)
		}
		return fmt.Errorf("failed to get role id: %w", err)
	}

	// Assign the role
	query := `
		INSERT INTO user_roles (user_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`

	_, err = r.db.Exec(ctx, query, userID, roleID, assignedBy)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	return nil
}

// Role Repository Methods

// Create creates a new role
func (r *PostgresRoleRepository) Create(ctx context.Context, role *entities.Role) error {
	query := `
		INSERT INTO roles (id, name, description, permissions)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.Exec(ctx, query,
		role.ID,
		role.Name,
		role.Description,
		role.Permissions,
	)

	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return nil
}

// GetByID retrieves a role by ID
func (r *PostgresRoleRepository) GetByID(ctx context.Context, id string) (*entities.Role, error) {
	query := `
		SELECT id, name, description, permissions, created_at, updated_at
		FROM roles
		WHERE id = $1
	`

	role := &entities.Role{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.Permissions,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("role with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get role by id: %w", err)
	}

	return role, nil
}

// GetByName retrieves a role by name
func (r *PostgresRoleRepository) GetByName(ctx context.Context, name string) (*entities.Role, error) {
	query := `
		SELECT id, name, description, permissions, created_at, updated_at
		FROM roles
		WHERE name = $1
	`

	role := &entities.Role{}
	err := r.db.QueryRow(ctx, query, name).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.Permissions,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("role with name %s not found", name)
		}
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}

	return role, nil
}

// Update updates a role
func (r *PostgresRoleRepository) Update(ctx context.Context, role *entities.Role) error {
	query := `
		UPDATE roles
		SET name = $2, description = $3, permissions = $4
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		role.ID,
		role.Name,
		role.Description,
		role.Permissions,
	)

	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}

// Delete deletes a role
func (r *PostgresRoleRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM roles WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}

// List retrieves a list of roles
func (r *PostgresRoleRepository) List(ctx context.Context, filter repositories.RoleFilter) ([]*entities.Role, error) {
	// Build the base query
	baseQuery := `
		SELECT id, name, description, permissions, created_at, updated_at
		FROM roles
		WHERE 1=1
	`

	// Build WHERE conditions
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex+1))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY with validation
	orderByClause, err := r.validateAndBuildOrderByForRole(filter.SortBy, filter.SortOrder, "created_at")
	if err != nil {
		return nil, err
	}
	baseQuery += orderByClause

	// Add LIMIT and OFFSET for pagination
	if filter.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++

		if filter.Page > 1 {
			offset := (filter.Page - 1) * filter.Limit
			baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, offset)
		}
	}

	// Execute query
	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []*entities.Role
	for rows.Next() {
		role := &entities.Role{}
		err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Description,
			&role.Permissions,
			&role.CreatedAt,
			&role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role row: %w", err)
		}
		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating role rows: %w", err)
	}

	return roles, nil
}

// Count returns the count of roles matching the filter
func (r *PostgresRoleRepository) Count(ctx context.Context, filter repositories.RoleFilter) (int, error) {
	baseQuery := `SELECT COUNT(*) FROM roles WHERE 1=1`

	// Build WHERE conditions (same as in List)
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex+1))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	var count int
	err := r.db.QueryRow(ctx, baseQuery, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count roles: %w", err)
	}

	return count, nil
}

// User Role Repository Methods

// AssignRole assigns a role to a user
func (r *PostgresUserRoleRepository) AssignRole(ctx context.Context, userID, roleID, assignedBy string) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, userID, roleID, assignedBy)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	return nil
}

// RemoveRole removes a role from a user
func (r *PostgresUserRoleRepository) RemoveRole(ctx context.Context, userID, roleID string) error {
	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`

	_, err := r.db.Exec(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}

	return nil
}

// GetUserRoles retrieves roles for a user
func (r *PostgresUserRoleRepository) GetUserRoles(ctx context.Context, userID string) ([]*entities.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.permissions, r.created_at, r.updated_at
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY ur.assigned_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var roles []*entities.Role
	for rows.Next() {
		role := &entities.Role{}
		err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Description,
			&role.Permissions,
			&role.CreatedAt,
			&role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role row: %w", err)
		}
		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating role rows: %w", err)
	}

	return roles, nil
}

// GetUsersByRole retrieves users that have a specific role
func (r *PostgresUserRoleRepository) GetUsersByRole(ctx context.Context, roleID string) ([]*entities.User, error) {
	query := `
		SELECT u.id, u.email, u.username, u.password_hash, u.first_name, u.last_name,
		       u.phone, u.is_active, u.is_verified, u.last_login_at, u.created_at, u.updated_at
		FROM users u
		INNER JOIN user_roles ur ON u.id = ur.user_id
		WHERE ur.role_id = $1
		ORDER BY ur.assigned_at DESC
	`

	rows, err := r.db.Query(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by role: %w", err)
	}
	defer rows.Close()

	var users []*entities.User
	for rows.Next() {
		user := &entities.User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Username,
			&user.PasswordHash,
			&user.FirstName,
			&user.LastName,
			&user.Phone,
			&user.IsActive,
			&user.IsVerified,
			&user.LastLoginAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}

// HasRole checks if a user has a specific role
func (r *PostgresUserRoleRepository) HasRole(ctx context.Context, userID, roleID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id = $1 AND role_id = $2)`

	var exists bool
	err := r.db.QueryRow(ctx, query, userID, roleID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if user has role: %w", err)
	}

	return exists, nil
}

// Additional Role Repository Methods for Complete RBAC Interface

// RoleExists checks if a role exists by name
func (r *PostgresRoleRepository) RoleExists(ctx context.Context, name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM roles WHERE name = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check role existence: %w", err)
	}

	return exists, nil
}

// GetUserRoles retrieves all roles for a user
func (r *PostgresRoleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*entities.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.permissions, r.created_at, r.updated_at
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY r.name`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var roles []*entities.Role
	for rows.Next() {
		role := &entities.Role{}
		err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Description,
			&role.Permissions,
			&role.CreatedAt,
			&role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role row: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetUsersWithRole retrieves all users that have a specific role
func (r *PostgresRoleRepository) GetUsersWithRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT user_id FROM user_roles WHERE role_id = $1`

	rows, err := r.db.Query(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users with role: %w", err)
	}
	defer rows.Close()

	var userIDs []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		err := rows.Scan(&userID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user ID: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, nil
}

// GetUserPermissions retrieves all permissions for a user
func (r *PostgresRoleRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `
		SELECT DISTINCT unnest(r.permissions)
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	defer rows.Close()

	var permissions []string
	permissionMap := make(map[string]bool)

	for rows.Next() {
		var permission string
		err := rows.Scan(&permission)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		if !permissionMap[permission] {
			permissionMap[permission] = true
			permissions = append(permissions, permission)
		}
	}

	return permissions, nil
}

// HasUserRole checks if a user has a specific role (UUID version)
func (r *PostgresRoleRepository) HasUserRole(ctx context.Context, userID, roleID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id = $1 AND role_id = $2)`

	var exists bool
	err := r.db.QueryRow(ctx, query, userID, roleID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user role: %w", err)
	}

	return exists, nil
}

// RemoveAllUserRoles removes all roles from a user
func (r *PostgresRoleRepository) RemoveAllUserRoles(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM user_roles WHERE user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to remove all user roles: %w", err)
	}

	return nil
}

// UserHasPermission checks if a user has a specific permission
func (r *PostgresRoleRepository) UserHasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM roles r
			INNER JOIN user_roles ur ON r.id = ur.role_id
			WHERE ur.user_id = $1 AND $2 = ANY(r.permissions)
		)`

	var exists bool
	err := r.db.QueryRow(ctx, query, userID, permission).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user permission: %w", err)
	}

	return exists, nil
}

// UserHasAnyPermission checks if a user has any of the specified permissions
func (r *PostgresRoleRepository) UserHasAnyPermission(ctx context.Context, userID uuid.UUID, permissions ...string) (bool, error) {
	if len(permissions) == 0 {
		return false, nil
	}

	// Create placeholders for the permissions
	placeholders := make([]string, len(permissions))
	args := make([]interface{}, len(permissions)+1)
	args[0] = userID

	for i, permission := range permissions {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = permission
	}

	query := fmt.Sprintf(`
		SELECT EXISTS(
			SELECT 1
			FROM roles r
			INNER JOIN user_roles ur ON r.id = ur.role_id
			WHERE ur.user_id = $1 AND r.permissions && ARRAY[%s]
		)`, strings.Join(placeholders, ","))

	var exists bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user permissions: %w", err)
	}

	return exists, nil
}

// UserHasAllPermissions checks if a user has all of the specified permissions
func (r *PostgresRoleRepository) UserHasAllPermissions(ctx context.Context, userID uuid.UUID, permissions ...string) (bool, error) {
	if len(permissions) == 0 {
		return true, nil
	}

	// Get all user permissions first
	userPermissions, err := r.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	// Create permission map for efficient lookup
	permissionMap := make(map[string]bool)
	for _, p := range userPermissions {
		permissionMap[p] = true
	}

	// Check if user has all required permissions
	for _, permission := range permissions {
		if !permissionMap[permission] {
			return false, nil
		}
	}

	return true, nil
}

// AddPermissionToRole adds a permission to a role
func (r *PostgresRoleRepository) AddPermissionToRole(ctx context.Context, roleID uuid.UUID, permission string) error {
	query := `
		UPDATE roles
		SET permissions = array_append(permissions, $2),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND NOT ($2 = ANY(permissions))`

	result, err := r.db.Exec(ctx, query, roleID, permission)
	if err != nil {
		return fmt.Errorf("failed to add permission to role: %w", err)
	}

	rowsAffected := result.RowsAffected()

	if rowsAffected == 0 {
		return fmt.Errorf("role not found or permission already exists")
	}

	return nil
}

// RemovePermissionFromRole removes a permission from a role
func (r *PostgresRoleRepository) RemovePermissionFromRole(ctx context.Context, roleID uuid.UUID, permission string) error {
	query := `
		UPDATE roles
		SET permissions = array_remove(permissions, $2),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND $2 = ANY(permissions)`

	result, err := r.db.Exec(ctx, query, roleID, permission)
	if err != nil {
		return fmt.Errorf("failed to remove permission from role: %w", err)
	}

	rowsAffected := result.RowsAffected()

	if rowsAffected == 0 {
		return fmt.Errorf("role not found or permission does not exist")
	}

	return nil
}

// GetRolePermissions retrieves all permissions for a role
func (r *PostgresRoleRepository) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	query := `SELECT permissions FROM roles WHERE id = $1`

	var permissions []string
	err := r.db.QueryRow(ctx, query, roleID).Scan(&permissions)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	return permissions, nil
}

// CreateDefaultRoles creates the default roles in the system
func (r *PostgresRoleRepository) CreateDefaultRoles(ctx context.Context) error {
	defaultRoles := entities.DefaultRoles()

	for _, role := range defaultRoles {
		exists, err := r.RoleExists(ctx, role.Name)
		if err != nil {
			return fmt.Errorf("failed to check if role %s exists: %w", role.Name, err)
		}

		if !exists {
			err = r.Create(ctx, &role)
			if err != nil {
				return fmt.Errorf("failed to create default role %s: %w", role.Name, err)
			}
		}
	}

	return nil
}

// GetRoleAssignmentHistory retrieves the role assignment history for a user
func (r *PostgresRoleRepository) GetRoleAssignmentHistory(ctx context.Context, userID uuid.UUID) ([]*entities.UserRole, error) {
	query := `
		SELECT user_id, role_id, assigned_at, assigned_by
		FROM user_roles
		WHERE user_id = $1
		ORDER BY assigned_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role assignment history: %w", err)
	}
	defer rows.Close()

	var assignments []*entities.UserRole
	for rows.Next() {
		assignment := &entities.UserRole{}
		err := rows.Scan(
			&assignment.UserID,
			&assignment.RoleID,
			&assignment.AssignedAt,
			&assignment.AssignedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role assignment: %w", err)
		}
		assignments = append(assignments, assignment)
	}

	return assignments, nil
}

// Helper method to adapt role methods to use UUID parameters
func (r *PostgresRoleRepository) CreateRole(ctx context.Context, role *entities.Role) error {
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	return r.Create(ctx, role)
}

func (r *PostgresRoleRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (*entities.Role, error) {
	return r.GetByID(ctx, id.String())
}

func (r *PostgresRoleRepository) GetRoleByName(ctx context.Context, name string) (*entities.Role, error) {
	return r.GetByName(ctx, name)
}

func (r *PostgresRoleRepository) UpdateRole(ctx context.Context, role *entities.Role) error {
	return r.Update(ctx, role)
}

func (r *PostgresRoleRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return r.Delete(ctx, id.String())
}

func (r *PostgresRoleRepository) GetAllRoles(ctx context.Context) ([]*entities.Role, error) {
	return r.List(ctx, repositories.RoleFilter{})
}

// UserRole assignment methods
func (r *PostgresUserRoleRepository) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID, assignedBy uuid.UUID) error {
	return r.AssignRole(ctx, userID.String(), roleID.String(), assignedBy.String())
}

func (r *PostgresUserRoleRepository) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return r.RemoveRole(ctx, userID.String(), roleID.String())
}