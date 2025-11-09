package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"erpgo/internal/domain/users/entities"
	"erpgo/internal/domain/users/repositories"
	"erpgo/pkg/auth"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *entities.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*entities.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *entities.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, filter repositories.UserFilter) ([]*entities.User, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.User), args.Error(1)
}

func (m *MockUserRepository) Count(ctx context.Context, filter repositories.UserFilter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockUserRepository) AssignRole(ctx context.Context, userID uuid.UUID, roleName string, assignedBy uuid.UUID) error {
	args := m.Called(ctx, userID, roleName, assignedBy)
	return args.Error(0)
}

// MockRoleRepository is a mock implementation of RoleRepository
type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) Create(ctx context.Context, role *entities.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) GetByID(ctx context.Context, id string) (*entities.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Role), args.Error(1)
}

func (m *MockRoleRepository) GetByName(ctx context.Context, name string) (*entities.Role, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Role), args.Error(1)
}

func (m *MockRoleRepository) Update(ctx context.Context, role *entities.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoleRepository) List(ctx context.Context, filter repositories.RoleFilter) ([]*entities.Role, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Role), args.Error(1)
}

func (m *MockRoleRepository) Count(ctx context.Context, filter repositories.RoleFilter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

// MockUserRoleRepository is a mock implementation of UserRoleRepository
type MockUserRoleRepository struct {
	mock.Mock
}

func (m *MockUserRoleRepository) AssignRole(ctx context.Context, userID, roleID, assignedBy string) error {
	args := m.Called(ctx, userID, roleID, assignedBy)
	return args.Error(0)
}

func (m *MockUserRoleRepository) RemoveRole(ctx context.Context, userID, roleID string) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockUserRoleRepository) GetUserRoles(ctx context.Context, userID string) ([]*entities.Role, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Role), args.Error(1)
}

func (m *MockUserRoleRepository) GetUsersByRole(ctx context.Context, roleID string) ([]*entities.User, error) {
	args := m.Called(ctx, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.User), args.Error(1)
}

func (m *MockUserRoleRepository) HasRole(ctx context.Context, userID, roleID string) (bool, error) {
	args := m.Called(ctx, userID, roleID)
	return args.Bool(0), args.Error(1)
}

// MockPasswordService is a mock implementation of PasswordService
type MockPasswordService struct {
	mock.Mock
}

func (m *MockPasswordService) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockPasswordService) CheckPassword(password, hash string) bool {
	args := m.Called(password, hash)
	return args.Bool(0)
}

func (m *MockPasswordService) ValidatePassword(password string) *auth.ValidationResult {
	args := m.Called(password)
	return args.Get(0).(*auth.ValidationResult)
}

func (m *MockPasswordService) GenerateSecurePassword(length int) (string, error) {
	args := m.Called(length)
	return args.String(0), args.Error(1)
}

func (m *MockPasswordService) GenerateResetToken() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// MockJWTService is a mock implementation of JWTService
type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) GenerateTokenPair(userID uuid.UUID, email, username string, roles []string) (string, string, error) {
	args := m.Called(userID, email, username, roles)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockJWTService) GenerateAccessToken(userID uuid.UUID, email, username string, roles []string) (string, error) {
	args := m.Called(userID, email, username, roles)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) ValidateToken(tokenString string) (*auth.Claims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.Claims), args.Error(1)
}

func (m *MockJWTService) ValidateAccessToken(tokenString string) (*auth.Claims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.Claims), args.Error(1)
}

func (m *MockJWTService) ValidateRefreshToken(tokenString string) (uuid.UUID, error) {
	args := m.Called(tokenString)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockJWTService) GetAccessExpiry() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func (m *MockJWTService) GetTokenExpiration(tokenString string) (*time.Time, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

func (m *MockJWTService) IsTokenExpired(tokenString string) bool {
	args := m.Called(tokenString)
	return args.Bool(0)
}