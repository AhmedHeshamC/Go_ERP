package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erpgo/internal/domain/users/entities"
	"erpgo/pkg/auth"
	"erpgo/pkg/cache"
)

func TestCreateUser(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}
	mockTxManager := &MockTransactionManager{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), mockTxManager)

	ctx := context.Background()
	req := &CreateUserRequest{
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "Password123!",
		FirstName: "Test",
		LastName:  "User",
		Phone:     "+1234567890",
	}

	// Mock expectations
	mockUserRepo.On("ExistsByEmail", ctx, "test@example.com").Return(false, nil)
	mockUserRepo.On("ExistsByUsername", ctx, "testuser").Return(false, nil)
	mockPasswordSvc.On("HashPassword", "Password123!").Return("hashedPassword", nil)
	mockUserRepo.On("Create", ctx, mock.AnythingOfType("*entities.User")).Return(nil)
	mockUserRepo.On("AssignRole", ctx, mock.AnythingOfType("uuid.UUID"), "student", mock.AnythingOfType("uuid.UUID")).Return(nil)

	// Execute
	result, err := service.CreateUser(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test@example.com", result.Email)
	assert.Equal(t, "testuser", result.Username)
	assert.Equal(t, "Test", result.FirstName)
	assert.Equal(t, "User", result.LastName)
	assert.Equal(t, "+1234567890", result.Phone)
	assert.True(t, result.IsActive)
	assert.False(t, result.IsVerified)
	assert.Empty(t, result.PasswordHash) // Should be removed in ToSafeUser()

	mockUserRepo.AssertExpectations(t)
	mockPasswordSvc.AssertExpectations(t)
}

func TestCreateUserUserAlreadyExists(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}
	mockTxManager := &MockTransactionManager{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), mockTxManager)

	ctx := context.Background()
	req := &CreateUserRequest{
		Email:     "existing@example.com",
		Username:  "testuser",
		Password:  "Password123!",
		FirstName: "Test",
		LastName:  "User",
	}

	// Mock expectations - user already exists by email
	mockUserRepo.On("ExistsByEmail", ctx, "existing@example.com").Return(true, nil)

	// Execute
	result, err := service.CreateUser(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrUserAlreadyExists, err)

	mockUserRepo.AssertExpectations(t)
}

func TestGetUser(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	userID := uuid.New()
	testUser := &entities.User{
		ID:        userID,
		Email:     "test@example.com",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		PasswordHash: "hashedPassword",
		IsActive:  true,
		IsVerified: true,
	}

	// Mock expectations
	mockUserRepo.On("GetByID", ctx, userID).Return(testUser, nil)

	// Execute
	result, err := service.GetUser(ctx, userID.String())

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, userID, result.ID)
	assert.Equal(t, "test@example.com", result.Email)
	assert.Equal(t, "testuser", result.Username)
	assert.Equal(t, "Test", result.FirstName)
	assert.Equal(t, "User", result.LastName)
	assert.Empty(t, result.PasswordHash) // Should be removed in ToSafeUser()

	mockUserRepo.AssertExpectations(t)
}

func TestGetUserNotFound(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	userID := uuid.New()

	// Mock expectations
	mockUserRepo.On("GetByID", ctx, userID).Return(nil, ErrUserNotFound)

	// Execute
	result, err := service.GetUser(ctx, userID.String())

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrUserNotFound, err)

	mockUserRepo.AssertExpectations(t)
}

func TestLogin(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	req := &LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
	}

	userID := uuid.New()
	testUser := &entities.User{
		ID:        userID,
		Email:     "test@example.com",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		PasswordHash: "hashedPassword",
		IsActive:  true,
		IsVerified: true,
	}

	userRoles := []string{"student", "user"}

	// Mock expectations
	mockUserRepo.On("GetByEmail", ctx, "test@example.com").Return(testUser, nil)
	mockPasswordSvc.On("CheckPassword", "Password123!", "hashedPassword").Return(true)
	mockUserRepo.On("GetUserRoles", ctx, userID).Return(userRoles, nil)
	mockUserRepo.On("UpdateLastLogin", ctx, userID).Return(nil)
	mockJWTSvc.On("GenerateTokenPair", userID, "test@example.com", "testuser", userRoles).
		Return("access_token", "refresh_token", nil)
	mockJWTSvc.On("GetAccessExpiry").Return(time.Hour)

	// Execute
	result, err := service.Login(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.User)
	assert.Equal(t, "test@example.com", result.User.Email)
	assert.Equal(t, "testuser", result.User.Username)
	assert.Equal(t, "access_token", result.AccessToken)
	assert.Equal(t, "refresh_token", result.RefreshToken)
	assert.Equal(t, int(time.Hour.Seconds()), result.ExpiresIn)

	mockUserRepo.AssertExpectations(t)
	mockPasswordSvc.AssertExpectations(t)
	mockJWTSvc.AssertExpectations(t)
}

func TestLoginInvalidCredentials(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	req := &LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	testUser := &entities.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashedPassword",
		IsActive:     true,
	}

	// Mock expectations
	mockUserRepo.On("GetByEmail", ctx, "test@example.com").Return(testUser, nil)
	mockPasswordSvc.On("CheckPassword", "wrongpassword", "hashedPassword").Return(false)

	// Execute
	result, err := service.Login(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrInvalidCredentials, err)

	mockUserRepo.AssertExpectations(t)
	mockPasswordSvc.AssertExpectations(t)
}

func TestListUsers(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	req := &ListUsersRequest{
		Page:   1,
		Limit:  10,
		Search: "test",
	}

	testUsers := []*entities.User{
		{
			ID:        uuid.New(),
			Email:     "test1@example.com",
			Username:  "testuser1",
			FirstName: "Test",
			LastName:  "User1",
		},
		{
			ID:        uuid.New(),
			Email:     "test2@example.com",
			Username:  "testuser2",
			FirstName: "Test",
			LastName:  "User2",
		},
	}

	// Mock expectations
	mockUserRepo.On("List", ctx, mock.AnythingOfType("repositories.UserFilter")).Return(testUsers, nil)
	mockUserRepo.On("Count", ctx, mock.AnythingOfType("repositories.UserFilter")).Return(2, nil)

	// Execute
	result, err := service.ListUsers(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Users, 2)
	assert.NotNil(t, result.Pagination)
	assert.Equal(t, 1, result.Pagination.Page)
	assert.Equal(t, 10, result.Pagination.Limit)
	assert.Equal(t, 2, result.Pagination.Total)
	assert.Equal(t, 1, result.Pagination.TotalPages)
	assert.False(t, result.Pagination.HasNext)
	assert.False(t, result.Pagination.HasPrev)

	mockUserRepo.AssertExpectations(t)
}

func TestUpdateUser(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	userID := uuid.New()

	firstName := "Updated"
	lastName := "Name"
	req := &UpdateUserRequest{
		FirstName: &firstName,
		LastName:  &lastName,
	}

	existingUser := &entities.User{
		ID:        userID,
		Email:     "test@example.com",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
	}

	// Mock expectations
	mockUserRepo.On("GetByID", ctx, userID).Return(existingUser, nil)
	mockUserRepo.On("Update", ctx, mock.AnythingOfType("*entities.User")).Return(nil)

	// Execute
	result, err := service.UpdateUser(ctx, userID.String(), req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Updated", result.FirstName)
	assert.Equal(t, "Name", result.LastName)

	mockUserRepo.AssertExpectations(t)
}

func TestAssignRole(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	userID := uuid.New()
	roleID := uuid.New()

	testUser := &entities.User{
		ID:    userID,
		Email: "test@example.com",
	}

	testRole := &entities.Role{
		ID:   roleID,
		Name: "admin",
	}

	// Mock expectations
	mockUserRepo.On("GetByID", ctx, userID).Return(testUser, nil)
	mockRoleRepo.On("GetByID", ctx, roleID.String()).Return(testRole, nil)
	mockUserRoleRepo.On("AssignRole", ctx, userID.String(), roleID.String(), userID.String()).Return(nil)

	// Execute
	err := service.AssignRole(ctx, userID.String(), roleID.String())

	// Assert
	require.NoError(t, err)

	mockUserRepo.AssertExpectations(t)
	mockRoleRepo.AssertExpectations(t)
	mockUserRoleRepo.AssertExpectations(t)
}

func TestCreateRole(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	req := &CreateRoleRequest{
		Name:        "manager",
		Description: "Manager role",
		Permissions: []string{"users.read", "users.write"},
	}

	// Mock expectations
	mockRoleRepo.On("GetByName", ctx, "manager").Return(nil, errors.New("not found"))
	mockRoleRepo.On("Create", ctx, mock.AnythingOfType("*entities.Role")).Return(nil)

	// Execute
	result, err := service.CreateRole(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "manager", result.Name)
	assert.Equal(t, "Manager role", result.Description)
	assert.Equal(t, []string{"users.read", "users.write"}, result.Permissions)

	mockRoleRepo.AssertExpectations(t)
}

func TestDeleteUser(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	userID := uuid.New()

	testUser := &entities.User{
		ID:    userID,
		Email: "test@example.com",
	}

	// Mock expectations
	mockUserRepo.On("GetByID", ctx, userID).Return(testUser, nil)
	mockUserRepo.On("Delete", ctx, userID).Return(nil)

	// Execute
	err := service.DeleteUser(ctx, userID.String())

	// Assert
	require.NoError(t, err)

	mockUserRepo.AssertExpectations(t)
}

func TestChangePassword(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	userID := uuid.New().String()
	req := &ChangePasswordRequest{
		OldPassword: "OldPassword123!",
		NewPassword: "NewPassword456!",
	}

	testUser := &entities.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		PasswordHash: "hashedOldPassword",
		IsActive:     true,
		UpdatedAt:    time.Now().UTC(),
	}

	// Mock expectations
	mockUserRepo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(testUser, nil)
	mockPasswordSvc.On("CheckPassword", "OldPassword123!", "hashedOldPassword").Return(true)
	mockPasswordSvc.On("CheckPassword", "NewPassword456!", "hashedOldPassword").Return(false)
	mockPasswordSvc.On("ValidatePassword", "NewPassword456!").Return(&auth.ValidationResult{
		Valid:  true,
		Errors: []string{},
	})
	mockPasswordSvc.On("HashPassword", "NewPassword456!").Return("hashedNewPassword", nil)
	mockUserRepo.On("Update", ctx, mock.AnythingOfType("*entities.User")).Return(nil)

	// Execute
	err := service.ChangePassword(ctx, userID, req)

	// Assert
	require.NoError(t, err)
	mockUserRepo.AssertExpectations(t)
	mockPasswordSvc.AssertExpectations(t)
}

func TestChangePasswordInvalidOldPassword(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	userID := uuid.New().String()
	req := &ChangePasswordRequest{
		OldPassword: "WrongPassword123!",
		NewPassword: "NewPassword456!",
	}

	testUser := &entities.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		PasswordHash: "hashedOldPassword",
		IsActive:     true,
		UpdatedAt:    time.Now().UTC(),
	}

	// Mock expectations
	mockUserRepo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(testUser, nil)
	mockPasswordSvc.On("CheckPassword", "WrongPassword123!", "hashedOldPassword").Return(false)

	// Execute
	err := service.ChangePassword(ctx, userID, req)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidCredentials, err)
	mockUserRepo.AssertExpectations(t)
	mockPasswordSvc.AssertExpectations(t)
}

func TestForgotPassword(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	email := "test@example.com"

	testUser := &entities.User{
		ID:       uuid.New(),
		Email:    email,
		IsActive: true,
	}

	// Mock expectations
	mockUserRepo.On("GetByEmail", ctx, email).Return(testUser, nil)
	mockPasswordSvc.On("GenerateResetToken").Return("reset-token-123", nil)

	// Execute
	err := service.ForgotPassword(ctx, email)

	// Assert
	require.NoError(t, err)
	mockUserRepo.AssertExpectations(t)
	mockPasswordSvc.AssertExpectations(t)
}

func TestResetPassword(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	userID := uuid.New()
	req := &ResetPasswordRequest{
		Token:    "reset-token-123",
		Password: "NewPassword123!",
	}

	testUser := &entities.User{
		ID:           userID,
		Email:        "test@example.com",
		PasswordHash: "hashedOldPassword",
		IsActive:     true,
		UpdatedAt:    time.Now().UTC(),
	}

	// Store reset token manually for test
	service.(*ServiceImpl).resetTokens["reset-token-123"] = &ResetTokenInfo{
		UserID:    userID,
		Token:     "reset-token-123",
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}

	// Mock expectations
	mockUserRepo.On("GetByID", ctx, userID).Return(testUser, nil)
	mockPasswordSvc.On("ValidatePassword", "NewPassword123!").Return(&auth.ValidationResult{
		Valid:  true,
		Errors: []string{},
	})
	mockPasswordSvc.On("HashPassword", "NewPassword123!").Return("hashedNewPassword", nil)
	mockUserRepo.On("Update", ctx, mock.AnythingOfType("*entities.User")).Return(nil)

	// Execute
	err := service.ResetPassword(ctx, req)

	// Assert
	require.NoError(t, err)
	mockUserRepo.AssertExpectations(t)
	mockPasswordSvc.AssertExpectations(t)

	// Verify token was removed
	_, exists := service.(*ServiceImpl).resetTokens["reset-token-123"]
	assert.False(t, exists)
}

func TestResetPasswordInvalidToken(t *testing.T) {
	// Setup
	mockUserRepo := &MockUserRepository{}
	mockRoleRepo := &MockRoleRepository{}
	mockUserRoleRepo := &MockUserRoleRepository{}
	mockPasswordSvc := &MockPasswordService{}
	mockJWTSvc := &MockJWTService{}

	service := NewService(mockUserRepo, mockRoleRepo, mockUserRoleRepo, mockPasswordSvc, mockJWTSvc, nil, cache.NewMockCache(), &MockTransactionManager{})

	ctx := context.Background()
	req := &ResetPasswordRequest{
		Token:    "invalid-token",
		Password: "NewPassword123!",
	}

	// Execute
	err := service.ResetPassword(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}