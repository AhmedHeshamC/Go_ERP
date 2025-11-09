package users_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"erpgo/internal/domain/users/entities"
)

func TestUser_NewUser(t *testing.T) {
	tests := []struct {
		name    string
		user    *entities.User
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid user",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid email",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "invalid-email",
				Username:     "testuser",
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name: "empty username",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "",
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "username cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUser_IsActiveUser(t *testing.T) {
	tests := []struct {
		name     string
		user     *entities.User
		expected bool
	}{
		{
			name: "active user",
			user: &entities.User{
				IsActive: true,
			},
			expected: true,
		},
		{
			name: "inactive user",
			user: &entities.User{
				IsActive: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.user.IsActive)
		})
	}
}

func TestUser_GetFullName(t *testing.T) {
	user := &entities.User{
		FirstName: "John",
		LastName:  "Doe",
	}

	expected := "John Doe"
	assert.Equal(t, expected, user.GetFullName())
}

func TestUser_HasPermission(t *testing.T) {
	// This would be implemented when we add permission logic to the User entity
	// For now, we'll test the basic structure
	user := &entities.User{
		ID:       uuid.New(),
		Username: "testuser",
	}

	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "testuser", user.Username)
}

func TestRole_NewRole(t *testing.T) {
	tests := []struct {
		name    string
		role    *entities.Role
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid role",
			role: &entities.Role{
				ID:          uuid.New(),
				Name:        "admin",
				Description: "Administrator role",
				Permissions: []string{"users:read", "users:write", "products:read"},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "empty role name",
			role: &entities.Role{
				ID:          uuid.New(),
				Name:        "",
				Description: "Empty name role",
				Permissions: []string{"users:read"},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			wantErr: true,
			errMsg:  "role name cannot be empty",
		},
		{
			name: "no permissions",
			role: &entities.Role{
				ID:          uuid.New(),
				Name:        "user",
				Description: "User role without permissions",
				Permissions: []string{},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			wantErr: true,
			errMsg:  "role must have at least one permission",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.role.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRole_HasPermission(t *testing.T) {
	role := &entities.Role{
		Name:        "admin",
		Permissions: []string{"users:read", "users:write", "products:read"},
	}

	tests := []struct {
		name     string
		permission string
		expected bool
	}{
		{
			name:      "has permission",
			permission: "users:read",
			expected:  true,
		},
		{
			name:      "does not have permission",
			permission: "products:write",
			expected:  false,
		},
		{
			name:      "empty permission",
			permission: "",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, role.HasPermission(tt.permission))
		})
	}
}

func TestUserRole_Assignment(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	assignedBy := uuid.New()

	userRole := &entities.UserRole{
		UserID:     userID,
		RoleID:     roleID,
		AssignedAt: time.Now(),
		AssignedBy: assignedBy,
	}

	assert.Equal(t, userID, userRole.UserID)
	assert.Equal(t, roleID, userRole.RoleID)
	assert.Equal(t, assignedBy, userRole.AssignedBy)
	assert.WithinDuration(t, time.Now(), userRole.AssignedAt, time.Second)
}

// Test factories for creating test data
func createTestUser(t *testing.T, overrides ...func(*entities.User)) *entities.User {
	user := &entities.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Username:  "testuser",
		PasswordHash: "hashed_password",
		FirstName: "Test",
		LastName:  "User",
		Phone:     "+1234567890",
		IsActive:  true,
		IsVerified: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	for _, override := range overrides {
		override(user)
	}

	return user
}

func createTestRole(t *testing.T, overrides ...func(*entities.Role)) *entities.Role {
	role := &entities.Role{
		ID:          uuid.New(),
		Name:        "user",
		Description: "Basic user role",
		Permissions: []string{"profile:read", "profile:write"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	for _, override := range overrides {
		override(role)
	}

	return role
}

// Example test using factories
func TestUser_Factory(t *testing.T) {
	// Test default user creation
	user := createTestUser(t)

	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "testuser", user.Username)
	assert.True(t, user.IsActive)

	// Test user with overrides
	customUser := createTestUser(t, func(u *entities.User) {
		u.Email = "custom@example.com"
		u.IsActive = false
	})

	assert.Equal(t, "custom@example.com", customUser.Email)
	assert.False(t, customUser.IsActive)
}

func TestRole_Factory(t *testing.T) {
	// Test default role creation
	role := createTestRole(t)

	assert.NotEmpty(t, role.ID)
	assert.Equal(t, "user", role.Name)
	assert.Equal(t, "Basic user role", role.Description)
	assert.NotEmpty(t, role.Permissions)

	// Test role with overrides
	adminRole := createTestRole(t, func(r *entities.Role) {
		r.Name = "admin"
		r.Description = "Administrator role"
		r.Permissions = []string{"*"}
	})

	assert.Equal(t, "admin", adminRole.Name)
	assert.Equal(t, "Administrator role", adminRole.Description)
	assert.Contains(t, adminRole.Permissions, "*")
}