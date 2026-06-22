package users_service_test

import (
	"context"
	"testing"
	"time"

	domain_dtos "go-chat/internal/core/domain/dtos"
	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
	users_service "go-chat/internal/features/users/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// --- Моки ---

type MockUsersRepository struct {
	mock.Mock
}

func (m *MockUsersRepository) GetUsers(ctx context.Context, limit, offset *int) ([]domain_models.User, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]domain_models.User), args.Error(1)
}

func (m *MockUsersRepository) GetUser(ctx context.Context, userID string) (domain_models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(domain_models.User), args.Error(1)
}

func (m *MockUsersRepository) GetUserByEmail(ctx context.Context, email string) (domain_models.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(domain_models.User), args.Error(1)
}

func (m *MockUsersRepository) UserExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUsersRepository) UserExistsByUsername(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}

func (m *MockUsersRepository) CreateUser(ctx context.Context, user domain_models.User) (domain_models.User, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(domain_models.User), args.Error(1)
}

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) IssueTokens(ctx context.Context, user domain_models.User) (domain_dtos.AuthResponseDTO, string, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(domain_dtos.AuthResponseDTO), args.String(1), args.Error(2)
}

// --- Хелперы ---

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(h)
}

func newTestUser(t *testing.T) domain_models.User {
	return domain_models.NewUser(
		"f8a04d50-d8ac-43ff-b5d5-c3f789da04d3",
		"test", "test@test.com", hashPassword(t, "pass1234"),
		time.Now(), time.Now(),
	)
}

func newService() (*users_service.UsersService, *MockUsersRepository, *MockAuthService) {
	repo := new(MockUsersRepository)
	auth := new(MockAuthService)
	svc := users_service.NewUsersService(repo, auth)
	return svc, repo, auth
}

// --- Register тесты ---

func TestRegister_Success(t *testing.T) {
	svc, repo, auth := newService()
	ctx := context.Background()
	user := newTestUser(t)

	repo.On("UserExistsByEmail", ctx, "test@test.com").Return(false, nil)
	repo.On("UserExistsByUsername", ctx, "test").Return(false, nil)
	repo.On("CreateUser", ctx, mock.Anything).Return(user, nil)
	auth.On("IssueTokens", ctx, user).Return(domain_dtos.AuthResponseDTO{}, "refresh_token", nil)

	_, _, err := svc.Register(ctx, "test", "test@test.com", "pass1234")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	auth.AssertExpectations(t)
}

func TestRegister_EmailAlreadyTaken(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()

	repo.On("UserExistsByEmail", ctx, "test@test.com").Return(true, nil)

	_, _, err := svc.Register(ctx, "test", "test@test.com", "pass1234")

	assert.ErrorIs(t, err, core_error.ErrConflict)
	repo.AssertNotCalled(t, "CreateUser")
}

func TestRegister_UsernameAlreadyTaken(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()

	repo.On("UserExistsByEmail", ctx, "test@test.com").Return(false, nil)
	repo.On("UserExistsByUsername", ctx, "test").Return(true, nil)

	_, _, err := svc.Register(ctx, "test", "test@test.com", "pass1234")

	assert.ErrorIs(t, err, core_error.ErrConflict)
	repo.AssertNotCalled(t, "CreateUser")
}

func TestRegister_EmptyFields(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()

	_, _, err := svc.Register(ctx, "", "test@test.com", "pass1234")
	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)

	_, _, err = svc.Register(ctx, "test", "", "pass1234")
	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)

	_, _, err = svc.Register(ctx, "test", "test@test.com", "")
	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)
}

// --- Login тесты ---

func TestLogin_Success(t *testing.T) {
	svc, repo, auth := newService()
	ctx := context.Background()
	user := newTestUser(t) // хеш генерируется здесь, pass1234 → bcrypt

	repo.On("GetUserByEmail", ctx, "test@test.com").Return(user, nil)
	auth.On("IssueTokens", ctx, user).Return(domain_dtos.AuthResponseDTO{}, "refresh_token", nil)

	_, _, err := svc.Login(ctx, "test@test.com", "pass1234")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	auth.AssertExpectations(t)
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()

	repo.On("GetUserByEmail", ctx, "notexist@test.com").Return(domain_models.User{}, core_error.ErrNotFound)

	_, _, err := svc.Login(ctx, "notexist@test.com", "pass1234")

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()
	user := newTestUser(t)

	repo.On("GetUserByEmail", ctx, "test@test.com").Return(user, nil)

	_, _, err := svc.Login(ctx, "test@test.com", "wrongpassword")

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
}

func TestLogin_EmptyFields(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()

	_, _, err := svc.Login(ctx, "", "pass1234")
	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)

	_, _, err = svc.Login(ctx, "test@test.com", "")
	assert.ErrorIs(t, err, core_error.ErrInvalidArgument)
}

// --- GetUser тесты ---

func TestGetUser_Success(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()
	user := newTestUser(t)

	repo.On("GetUser", ctx, user.ID).Return(user, nil)

	result, err := svc.GetUser(ctx, user.ID)

	assert.NoError(t, err)
	assert.Equal(t, user.ID, result.ID)
	assert.Equal(t, user.Username, result.Username)
}

func TestGetUser_NotFound(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()

	repo.On("GetUser", ctx, "nonexistent").Return(domain_models.User{}, core_error.ErrNotFound)

	_, err := svc.GetUser(ctx, "nonexistent")

	assert.Error(t, err)
}
