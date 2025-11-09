package unit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"erpgo/internal/application/services/user"
	"erpgo/internal/domain/users/entities"
	"erpgo/internal/domain/users/repositories"
	"erpgo/pkg/auth"
	"erpgo/pkg/cache"
)

// MockUserRepository for testing
type MockUserRepo struct {
	users map[uuid.UUID]*entities.User
}

func NewMockUserRepo() *MockUserRepo {
	return &MockUserRepo{
		users: make(map[uuid.UUID]*entities.User),
	}
}

func (m *MockUserRepo) Create(ctx context.Context, user *entities.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	user, exists := m.users[id]
	if !exists {
		return nil, repositories.ErrUserNotFound
	}
	return user, nil
}

func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, repositories.ErrUserNotFound
}

func (m *MockUserRepo) Update(ctx context.Context, user *entities.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.users, id)
	return nil
}

func (m *MockUserRepo) List(ctx context.Context, filter *repositories.ListUsersFilter) ([]*entities.User, error) {
	var users []*entities.User
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func (m *MockUserRepo) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	_, exists := m.users[id]
	return exists, nil
}

func (m *MockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	for _, user := range m.users {
		if user.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockUserRepo) Count(ctx context.Context, filter *repositories.ListUsersFilter) (int64, error) {
	return int64(len(m.users)), nil
}

func (m *MockUserRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*entities.Role, error) {
	return []*entities.Role{}, nil
}

// MockRoleRepository for testing
type MockRoleRepo struct{}

func (m *MockRoleRepo) CreateRole(ctx context.Context, role *entities.Role) error                         { return nil }
func (m *MockRoleRepo) GetRoleByID(ctx context.Context, id uuid.UUID) (*entities.Role, error)         { return nil, nil }
func (m *MockRoleRepo) GetRoleByName(ctx context.Context, name string) (*entities.Role, error)        { return nil, nil }
func (m *MockRoleRepo) GetAllRoles(ctx context.Context) ([]*entities.Role, error)                    { return []*entities.Role{}, nil }
func (m *MockRoleRepo) UpdateRole(ctx context.Context, role *entities.Role) error                     { return nil }
func (m *MockRoleRepo) DeleteRole(ctx context.Context, id uuid.UUID) error                            { return nil }
func (m *MockRoleRepo) RoleExists(ctx context.Context, name string) (bool, error)                     { return false, nil }
func (m *MockRoleRepo) AssignRoleToUser(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error { return nil }
func (m *MockRoleRepo) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error        { return nil }
func (m *MockRoleRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*entities.Role, error) { return []*entities.Role{}, nil }
func (m *MockRoleRepo) GetUsersWithRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)   { return []uuid.UUID{}, nil }
func (m *MockRoleRepo) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error)   { return []string{}, nil }
func (m *MockRoleRepo) HasUserRole(ctx context.Context, userID, roleID uuid.UUID) (bool, error)      { return false, nil }
func (m *MockRoleRepo) RemoveAllUserRoles(ctx context.Context, userID uuid.UUID) error                { return nil }
func (m *MockRoleRepo) UserHasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) { return false, nil }
func (m *MockRoleRepo) UserHasAnyPermission(ctx context.Context, userID uuid.UUID, permissions ...string) (bool, error) { return false, nil }
func (m *MockRoleRepo) UserHasAllPermissions(ctx context.Context, userID uuid.UUID, permissions ...string) (bool, error) { return false, nil }
func (m *MockRoleRepo) AddPermissionToRole(ctx context.Context, roleID uuid.UUID, permission string) error { return nil }
func (m *MockRoleRepo) RemovePermissionFromRole(ctx context.Context, roleID uuid.UUID, permission string) error { return nil }
func (m *MockRoleRepo) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]string, error)   { return []string{}, nil }
func (m *MockRoleRepo) CreateDefaultRoles(ctx context.Context) error                                   { return nil }
func (m *MockRoleRepo) GetRoleAssignmentHistory(ctx context.Context, userID uuid.UUID) ([]*entities.UserRole, error) { return []*entities.UserRole{}, nil }

// MockUserRoleRepository for testing
type MockUserRoleRepo struct{}

func (m *MockUserRoleRepo) Create(ctx context.Context, userRole *entities.UserRole) error   { return nil }
func (m *MockUserRoleRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.UserRole, error) { return nil, nil }
func (m *MockUserRoleRepo) GetByUserIDAndRoleID(ctx context.Context, userID, roleID uuid.UUID) (*entities.UserRole, error) { return nil, nil }
func (m *MockUserRoleRepo) Update(ctx context.Context, userRole *entities.UserRole) error { return nil }
func (m *MockUserRoleRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockUserRoleRepo) List(ctx context.Context, filter *repositories.ListUserRolesFilter) ([]*entities.UserRole, error) { return []*entities.UserRole{}, nil }
func (m *MockUserRoleRepo) DeleteByUserIDAndRoleID(ctx context.Context, userID, roleID uuid.UUID) error { return nil }
func (m *MockUserRoleRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error { return nil }
func (m *MockUserRoleRepo) Exists(ctx context.Context, id uuid.UUID) (bool, error) { return false, nil }

func TestPasswordChange(t *testing.T) {
	ctx := context.Background()

	// Setup
	userRepo := NewMockUserRepo()
	roleRepo := &MockRoleRepo{}
	userRoleRepo := &MockUserRoleRepo{}
	mockCache := cache.NewMockCache()

	passwordSvc := auth.NewPasswordService(12, "test-pepper")
	jwtSvc := auth.NewJWTService("test-secret", "test-issuer", time.Hour, 24*time.Hour)

	userService := user.NewUserService(userRepo, roleRepo, userRoleRepo, passwordSvc, jwtSvc, nil, mockCache)

	// Create a test user
	user := &entities.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "oldhash",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Hash the old password
	oldPasswordHash, err := passwordSvc.HashPassword("oldpassword123")
	require.NoError(t, err)
	user.PasswordHash = oldPasswordHash

	err = userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Test changing password with correct old password
	changeReq := &user.ChangePasswordRequest{
		OldPassword: "oldpassword123",
		NewPassword: "newpassword456",
	}

	err = userService.ChangePassword(ctx, user.ID.String(), changeReq)
	assert.NoError(t, err)

	// Verify password was changed
	updatedUser, err := userRepo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.NotEqual(t, oldPasswordHash, updatedUser.PasswordHash)

	// Verify new password works
	assert.True(t, passwordSvc.CheckPassword("newpassword456", updatedUser.PasswordHash))

	// Test changing password with wrong old password should fail
	changeReq = &user.ChangePasswordRequest{
		OldPassword: "wrongpassword",
		NewPassword: "anotherpassword",
	}

	err = userService.ChangePassword(ctx, user.ID.String(), changeReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestPasswordResetFlow(t *testing.T) {
	ctx := context.Background()

	// Setup
	userRepo := NewMockUserRepo()
	roleRepo := &MockRoleRepo{}
	userRoleRepo := &MockUserRoleRepo{}
	mockCache := cache.NewMockCache()

	passwordSvc := auth.NewPasswordService(12, "test-pepper")
	jwtSvc := auth.NewJWTService("test-secret", "test-issuer", time.Hour, 24*time.Hour)

	userService := user.NewUserService(userRepo, roleRepo, userRoleRepo, passwordSvc, jwtSvc, nil, mockCache)

	// Create a test user
	user := &entities.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "oldhash",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Test forgot password - should not reveal if user exists or not
	err = userService.ForgotPassword(ctx, "nonexistent@example.com")
	assert.NoError(t, err) // Should not error for security

	// Test forgot password for existing user
	err = userService.ForgotPassword(ctx, "test@example.com")
	assert.NoError(t, err)

	// Note: In a real test, we would capture the reset token from the email service
	// For now, we'll test the flow conceptually

	// Test reset password with invalid token should fail
	resetReq := &user.ResetPasswordRequest{
		Token:    "invalid-token",
		Password: "newpassword123",
	}

	err = userService.ResetPassword(ctx, resetReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestPasswordValidation(t *testing.T) {
	ctx := context.Background()

	// Setup
	userRepo := NewMockUserRepo()
	roleRepo := &MockRoleRepo{}
	userRoleRepo := &MockUserRoleRepo{}
	mockCache := cache.NewMockCache()

	passwordSvc := auth.NewPasswordService(12, "test-pepper")
	jwtSvc := auth.NewJWTService("test-secret", "test-issuer", time.Hour, 24*time.Hour)

	userService := user.NewUserService(userRepo, roleRepo, userRoleRepo, passwordSvc, jwtSvc, nil, mockCache)

	// Create a test user
	user := &entities.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "oldhash",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	oldPasswordHash, err := passwordSvc.HashPassword("oldpassword123")
	require.NoError(t, err)
	user.PasswordHash = oldPasswordHash

	err = userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Test changing password with weak password should fail
	changeReq := &user.ChangePasswordRequest{
		OldPassword: "oldpassword123",
		NewPassword: "123", // Too short
	}

	err = userService.ChangePassword(ctx, user.ID.String(), changeReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")

	// Test changing password with same as old password should fail
	changeReq = &user.ChangePasswordRequest{
		OldPassword: "oldpassword123",
		NewPassword: "oldpassword123", // Same as old
	}

	err = userService.ChangePassword(ctx, user.ID.String(), changeReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "different")
}