package users_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"erpgo/internal/domain/users/entities"
	apperrors "erpgo/pkg/errors"
)

func TestUser_Validate(t *testing.T) {
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
			name: "invalid email - no @ symbol",
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
			name: "invalid email - no domain",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@",
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
			name: "empty email",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "",
				Username:     "testuser",
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "email cannot be empty",
		},
		{
			name: "email too long",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        strings.Repeat("a", 250) + "@example.com",
				Username:     "testuser",
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "email cannot exceed 255 characters",
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
		{
			name: "username too short",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "ab",
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "username must be between 3 and 50 characters",
		},
		{
			name: "username too long",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     strings.Repeat("a", 51),
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "username must be between 3 and 50 characters",
		},
		{
			name: "username with invalid characters",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "test@user",
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "username can only contain letters, numbers, underscores, and hyphens",
		},
		{
			name: "empty password hash",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "",
				FirstName:    "Test",
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "password hash cannot be empty",
		},
		{
			name: "whitespace-only password hash",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "   ",
				FirstName:    "Test",
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "password hash cannot be empty",
		},
		{
			name: "empty first name",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "hashed_password_here",
				FirstName:    "",
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "first name cannot be empty",
		},
		{
			name: "whitespace-only first name",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "hashed_password_here",
				FirstName:    "   ",
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "first name cannot be empty",
		},
		{
			name: "first name too long",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "hashed_password_here",
				FirstName:    strings.Repeat("a", 101),
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "first name cannot exceed 100 characters",
		},
		{
			name: "empty last name",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     "",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "last name cannot be empty",
		},
		{
			name: "last name too long",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     strings.Repeat("a", 101),
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "last name cannot exceed 100 characters",
		},
		{
			name: "invalid phone format - too short",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     "User",
				Phone:        "123",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "invalid phone number format",
		},
		{
			name: "invalid phone format - invalid characters",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     "User",
				Phone:        "abc-def-ghij",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "invalid phone number format",
		},
		{
			name: "valid phone with international format",
			user: &entities.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     "User",
				Phone:        "+1 (555) 123-4567",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: false,
		},
		{
			name: "nil UUID",
			user: &entities.User{
				ID:           uuid.Nil,
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "hashed_password_here",
				FirstName:    "Test",
				LastName:     "User",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "user ID cannot be empty",
		},
		{
			name: "multiple validation errors",
			user: &entities.User{
				ID:           uuid.Nil,
				Email:        "invalid-email",
				Username:     "ab",
				PasswordHash: "",
				FirstName:    "",
				LastName:     "",
				Phone:        "123",
				IsActive:     true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()

			if tt.wantErr {
				require.Error(t, err)

				// Check if it's a ValidationError
				if validationErr, ok := err.(*apperrors.ValidationError); ok && tt.errMsg != "" {
					// Look for the error message in field errors
					found := false
					for _, messages := range validationErr.Fields {
						for _, msg := range messages {
							if strings.Contains(msg, tt.errMsg) {
								found = true
								break
							}
						}
						if found {
							break
						}
					}
					assert.True(t, found, "Expected error message '%s' not found in validation errors: %v", tt.errMsg, validationErr.Fields)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}


func TestUser_GetFullName(t *testing.T) {
	tests := []struct {
		name     string
		user     *entities.User
		expected string
	}{
		{
			name: "normal names",
			user: &entities.User{
				FirstName: "John",
				LastName:  "Doe",
			},
			expected: "John Doe",
		},
		{
			name: "names with leading/trailing spaces",
			user: &entities.User{
				FirstName: "  John  ",
				LastName:  "  Doe  ",
			},
			expected: "John Doe",
		},
		{
			name: "single character names",
			user: &entities.User{
				FirstName: "A",
				LastName:  "B",
			},
			expected: "A B",
		},
		{
			name: "names with special characters",
			user: &entities.User{
				FirstName: "Jean-Claude",
				LastName:  "O'Connor",
			},
			expected: "Jean-Claude O'Connor",
		},
		{
			name: "empty first name",
			user: &entities.User{
				FirstName: "",
				LastName:  "Doe",
			},
			expected: " Doe",
		},
		{
			name: "empty last name",
			user: &entities.User{
				FirstName: "John",
				LastName:  "",
			},
			expected: "John ",
		},
		{
			name: "both names empty",
			user: &entities.User{
				FirstName: "",
				LastName:  "",
			},
			expected: " ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.user.GetFullName())
		})
	}
}

func TestUser_HasPermission(t *testing.T) {
	user := &entities.User{
		ID:       uuid.New(),
		Username: "testuser",
	}

	// Test the placeholder implementation - should always return false
	tests := []struct {
		name       string
		permission string
		expected   bool
	}{
		{
			name:       "user read permission",
			permission: "users.read",
			expected:   false,
		},
		{
			name:       "user write permission",
			permission: "users.write",
			expected:   false,
		},
		{
			name:       "admin permission",
			permission: "system.admin",
			expected:   false,
		},
		{
			name:       "empty permission",
			permission: "",
			expected:   false,
		},
		{
			name:       "non-existent permission",
			permission: "non.existent.permission",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, user.HasPermission(tt.permission))
		})
	}
}

func TestUser_UpdateLastLogin(t *testing.T) {
	user := &entities.User{
		ID:       uuid.New(),
		Username: "testuser",
	}

	// Initially no last login
	assert.Nil(t, user.LastLoginAt)

	// Update last login
	user.UpdateLastLogin()

	// Should have a last login time now
	require.NotNil(t, user.LastLoginAt)

	// Should be within the last few seconds (allowing for test execution time)
	now := time.Now()
	assert.True(t, user.LastLoginAt.Before(now) || user.LastLoginAt.Equal(now))
	assert.True(t, user.LastLoginAt.After(now.Add(-10*time.Second)))

	// Test updating again
	oldLastLogin := *user.LastLoginAt
	time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamp
	user.UpdateLastLogin()

	// Should be updated to a newer time
	assert.True(t, user.LastLoginAt.After(oldLastLogin))
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
		{
			name: "default active user",
			user: &entities.User{},
			expected: false, // Default bool is false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.user.IsActiveUser())
		})
	}
}

func TestUser_IsVerifiedUser(t *testing.T) {
	tests := []struct {
		name     string
		user     *entities.User
		expected bool
	}{
		{
			name: "verified user",
			user: &entities.User{
				IsVerified: true,
			},
			expected: true,
		},
		{
			name: "unverified user",
			user: &entities.User{
				IsVerified: false,
			},
			expected: false,
		},
		{
			name: "default unverified user",
			user: &entities.User{},
			expected: false, // Default bool is false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.user.IsVerifiedUser())
		})
	}
}

func TestUser_ToSafeUser(t *testing.T) {
	now := time.Now()
	user := &entities.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "secret_password_hash",
		FirstName:    "Test",
		LastName:     "User",
		Phone:        "+1234567890",
		IsActive:     true,
		IsVerified:   true,
		LastLoginAt:  &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	safeUser := user.ToSafeUser()

	// Safe user should have all fields except password hash
	assert.Equal(t, user.ID, safeUser.ID)
	assert.Equal(t, user.Email, safeUser.Email)
	assert.Equal(t, user.Username, safeUser.Username)
	assert.Equal(t, user.FirstName, safeUser.FirstName)
	assert.Equal(t, user.LastName, safeUser.LastName)
	assert.Equal(t, user.Phone, safeUser.Phone)
	assert.Equal(t, user.IsActive, safeUser.IsActive)
	assert.Equal(t, user.IsVerified, safeUser.IsVerified)
	assert.Equal(t, user.LastLoginAt, safeUser.LastLoginAt)
	assert.Equal(t, user.CreatedAt, safeUser.CreatedAt)
	assert.Equal(t, user.UpdatedAt, safeUser.UpdatedAt)

	// Password hash should be empty in safe user
	assert.Equal(t, "", safeUser.PasswordHash)

	// Ensure it's actually a different object
	assert.NotSame(t, user, safeUser)

	// Modifying safe user shouldn't affect original user
	safeUser.Email = "modified@example.com"
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "modified@example.com", safeUser.Email)
}

func TestUser_StateTransitions(t *testing.T) {
	user := &entities.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashed_password",
		FirstName:    "Test",
		LastName:     "User",
		IsActive:     false,
		IsVerified:   false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Test initial state
	assert.False(t, user.IsActiveUser())
	assert.False(t, user.IsVerifiedUser())
	assert.Nil(t, user.LastLoginAt)

	// Test activation
	user.IsActive = true
	assert.True(t, user.IsActiveUser())
	assert.False(t, user.IsVerifiedUser())

	// Test verification
	user.IsVerified = true
	assert.True(t, user.IsActiveUser())
	assert.True(t, user.IsVerifiedUser())

	// Test deactivation
	user.IsActive = false
	assert.False(t, user.IsActiveUser())
	assert.True(t, user.IsVerifiedUser()) // Verification should remain

	// Test login update
	user.UpdateLastLogin()
	assert.NotNil(t, user.LastLoginAt)
}

func TestRole_NewRole(t *testing.T) {
	tests := []struct {
		name string
		role *entities.Role
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just test that we can create a role entity
			// Validation logic would be tested if a Validate method existed
			assert.NotEqual(t, uuid.Nil, tt.role.ID)
			assert.NotNil(t, tt.role.Permissions)
		})
	}
}

func TestRole_HasPermission(t *testing.T) {
	role := &entities.Role{
		Name:        "admin",
		Permissions: []string{"users:read", "users:write", "products:read"},
	}

	tests := []struct {
		name       string
		permission string
		expected   bool
	}{
		{
			name:       "has permission",
			permission: "users:read",
			expected:   true,
		},
		{
			name:       "does not have permission",
			permission: "products:write",
			expected:   false,
		},
		{
			name:       "empty permission",
			permission: "",
			expected:   false,
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
		ID:           uuid.New(),
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashed_password",
		FirstName:    "Test",
		LastName:     "User",
		Phone:        "+1234567890",
		IsActive:     true,
		IsVerified:   true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
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

func TestEmailVerification_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "not expired - future time",
			expiresAt: now.Add(1 * time.Hour),
			expected:  false,
		},
		{
			name:      "not expired - now",
			expiresAt: now.Add(10 * time.Second),
			expected:  false,
		},
		{
			name:      "expired - past time",
			expiresAt: now.Add(-1 * time.Hour),
			expected:  true,
		},
		{
			name:      "expired - just past",
			expiresAt: now.Add(-1 * time.Second),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := &entities.EmailVerification{
				ExpiresAt: tt.expiresAt,
			}
			assert.Equal(t, tt.expected, ev.IsExpired())
		})
	}
}

func TestEmailVerification_IsValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt time.Time
		isUsed    bool
		expected  bool
	}{
		{
			name:      "valid - not used and not expired",
			expiresAt: now.Add(1 * time.Hour),
			isUsed:    false,
			expected:  true,
		},
		{
			name:      "invalid - expired",
			expiresAt: now.Add(-1 * time.Hour),
			isUsed:    false,
			expected:  false,
		},
		{
			name:      "invalid - already used",
			expiresAt: now.Add(1 * time.Hour),
			isUsed:    true,
			expected:  false,
		},
		{
			name:      "invalid - expired and used",
			expiresAt: now.Add(-1 * time.Hour),
			isUsed:    true,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := &entities.EmailVerification{
				ExpiresAt: tt.expiresAt,
				IsUsed:    tt.isUsed,
			}
			assert.Equal(t, tt.expected, ev.IsValid())
		})
	}
}

func TestEmailVerification_MarkAsUsed(t *testing.T) {
	ev := &entities.EmailVerification{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Email:     "test@example.com",
		Token:     "test-token",
		TokenType: entities.TokenTypeVerification,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		IsUsed:    false,
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	// Initially not used
	assert.False(t, ev.IsUsed)
	assert.Nil(t, ev.UsedAt)

	// Mark as used
	beforeMarkAsUsed := time.Now()
	ev.MarkAsUsed()

	// Should be marked as used
	assert.True(t, ev.IsUsed)
	require.NotNil(t, ev.UsedAt)

	// UsedAt should be set to a recent time
	assert.True(t, ev.UsedAt.After(beforeMarkAsUsed.Add(-1*time.Second)) ||
		ev.UsedAt.Equal(beforeMarkAsUsed))
	assert.True(t, ev.UsedAt.Before(time.Now().Add(1*time.Second)))

	// UpdatedAt should also be updated
	assert.True(t, ev.UpdatedAt.After(beforeMarkAsUsed.Add(-1*time.Second)) ||
		ev.UpdatedAt.Equal(beforeMarkAsUsed))
}

func TestEmailVerification_GenerateVerificationToken(t *testing.T) {
	userID := uuid.New()
	email := "test@example.com"
	tokenType := entities.TokenTypeVerification
	expiration := 24 * time.Hour

	token := entities.GenerateVerificationToken(userID, email, tokenType, expiration)

	// Verify all fields are set correctly
	assert.NotEqual(t, uuid.Nil, token.ID)
	assert.Equal(t, userID, token.UserID)
	assert.Equal(t, email, token.Email)
	assert.Equal(t, tokenType, token.TokenType)
	assert.NotEmpty(t, token.Token)

	// Should not be used by default
	assert.False(t, token.IsUsed)
	assert.Nil(t, token.UsedAt)

	// Expiration should be approximately correct (within a reasonable tolerance)
	expectedExpiresAt := time.Now().Add(expiration)
	timeDiff := token.ExpiresAt.Sub(expectedExpiresAt)
	assert.True(t, timeDiff < 1*time.Second && timeDiff > -1*time.Second,
		"Expiration time should be within 1 second of expected, got difference: %v", timeDiff)

	// CreatedAt and UpdatedAt should be set
	assert.False(t, token.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(t, token.UpdatedAt.IsZero(), "UpdatedAt should be set")
	// CreatedAt should be recent (within last 5 seconds)
	assert.True(t, time.Since(token.CreatedAt) < 5*time.Second, "CreatedAt should be recent")
}

func TestEmailVerification_TokenTypeConstants(t *testing.T) {
	// Test that token type constants are properly defined
	assert.Equal(t, "verification", entities.TokenTypeVerification)
	assert.Equal(t, "password_reset", entities.TokenTypePasswordReset)
	assert.Equal(t, "email_change", entities.TokenTypeEmailChange)
}

func TestEmailVerification_ExpirationConstants(t *testing.T) {
	// Test that expiration constants are properly defined
	assert.Equal(t, 24*time.Hour, entities.DefaultVerificationExpiration)
	assert.Equal(t, 1*time.Hour, entities.DefaultPasswordResetExpiration)
	assert.Equal(t, 30*time.Minute, entities.DefaultEmailChangeExpiration)
}

func TestEmailVerification_SecureTokenGeneration(t *testing.T) {
	// Test that multiple calls generate different tokens
	userID := uuid.New()
	email := "test@example.com"
	tokenType := entities.TokenTypeVerification
	expiration := 1 * time.Hour

	token1 := entities.GenerateVerificationToken(userID, email, tokenType, expiration)
	time.Sleep(1 * time.Millisecond) // Small delay to ensure different timestamps
	token2 := entities.GenerateVerificationToken(userID, email, tokenType, expiration)

	// Tokens should be different
	assert.NotEqual(t, token1.Token, token2.Token)
	assert.NotEqual(t, token1.ID, token2.ID)

	// But other fields should be the same (except timestamps)
	assert.Equal(t, token1.UserID, token2.UserID)
	assert.Equal(t, token1.Email, token2.Email)
	assert.Equal(t, token1.TokenType, token2.TokenType)
}
