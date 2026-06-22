package jwt_service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	core_config "go-chat/internal/core/config"
	domain_models "go-chat/internal/core/domain/models"
	jwt_service "go-chat/internal/features/jwt/service"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Моки ---

type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) SaveRefreshToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	args := m.Called(ctx, userID, token, expiresAt)
	return args.Error(0)
}

func (m *MockTokenRepository) GetRefreshToken(ctx context.Context, token string) (jwt_service.RefreshTokenModel, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(jwt_service.RefreshTokenModel), args.Error(1)
}

func (m *MockTokenRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockTokenRepository) ReplaceRefreshToken(ctx context.Context, oldTokenHash, userID, newTokenHash string, expiresAt time.Time) error {
	args := m.Called(ctx, oldTokenHash, userID, newTokenHash, expiresAt)
	return args.Error(0)
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetUser(ctx context.Context, userID string) (domain_models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(domain_models.User), args.Error(1)
}

// --- Хелперы ---

func newTestConfig() *core_config.Config {
	return &core_config.Config{
		JwtAccessSecret:  "test-access-secret",
		JwtRefreshSecret: "test-refresh-secret",
		JwtAccessTTL:     1 * time.Hour,
		JwtRefreshTTL:    24 * time.Hour,
	}
}

func newTestUser() domain_models.User {
	return domain_models.User{
		ID:       "user-123",
		Username: "testuser",
		Email:    "test@test.com",
		Password: "hashed",
	}
}

func newService() (*jwt_service.JwtService, *MockTokenRepository, *MockUserRepository) {
	tokenRepo := new(MockTokenRepository)
	userRepo := new(MockUserRepository)
	cfg := newTestConfig()
	svc := jwt_service.NewJwtService(tokenRepo, userRepo, cfg)
	return svc, tokenRepo, userRepo
}

// generateTestToken создаёт JWT с указанными claims и секретом (для тестов).
func generateTestToken(secret string, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

// hashToken - локальная копия хеш-функции для тестов (если не экспортирована).
// Если jwt_service.HashToken экспортирована, используем её.
func hashToken(token string) string {
	// Если HashToken экспортирована, замените на jwt_service.HashToken(token)
	// Пока используем свою реализацию.
	return jwt_service.HashToken(token)
}

// --- Тесты ---

func TestGenerateAccessToken_Success(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	userID := "user-123"
	username := "testuser"

	token, err := svc.GenerateAccessToken(ctx, userID, username)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(newTestConfig().JwtAccessSecret), nil
	})
	assert.NoError(t, err)
	assert.True(t, parsed.Valid)

	claims := parsed.Claims.(jwt.MapClaims)
	assert.Equal(t, userID, claims["sub"])
	assert.Equal(t, username, claims["username"])
}

func TestGenerateRefreshToken_Success(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	userID := "user-123"

	token, err := svc.GenerateRefreshToken(ctx, userID)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(newTestConfig().JwtRefreshSecret), nil
	})
	assert.NoError(t, err)
	assert.True(t, parsed.Valid)

	claims := parsed.Claims.(jwt.MapClaims)
	assert.Equal(t, userID, claims["sub"])
}

func TestValidateRefreshToken_Success(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	cfg := newTestConfig()
	userID := "user-123"
	tokenString := generateTestToken(cfg.JwtRefreshSecret, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	result, err := svc.ValidateRefreshToken(ctx, tokenString)

	assert.NoError(t, err)
	assert.Equal(t, userID, result)
}

func TestValidateRefreshToken_InvalidSignature(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	tokenString := generateTestToken("wrong-secret", jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	result, err := svc.ValidateRefreshToken(ctx, tokenString)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestValidateRefreshToken_Expired(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	cfg := newTestConfig()
	tokenString := generateTestToken(cfg.JwtRefreshSecret, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	})

	result, err := svc.ValidateRefreshToken(ctx, tokenString)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestValidateRefreshToken_MissingSubject(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	cfg := newTestConfig()
	tokenString := generateTestToken(cfg.JwtRefreshSecret, jwt.MapClaims{
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	result, err := svc.ValidateRefreshToken(ctx, tokenString)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "invalid subject")
}

func TestValidateAccessToken_Success(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	cfg := newTestConfig()
	userID := "user-123"
	username := "testuser"
	tokenString := generateTestToken(cfg.JwtAccessSecret, jwt.MapClaims{
		"sub":      userID,
		"username": username,
		"exp":      time.Now().Add(1 * time.Hour).Unix(),
	})

	id, name, err := svc.ValidateAccessToken(ctx, tokenString)

	assert.NoError(t, err)
	assert.Equal(t, userID, id)
	assert.Equal(t, username, name)
}

func TestValidateAccessToken_WithoutUsername(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	cfg := newTestConfig()
	userID := "user-123"
	tokenString := generateTestToken(cfg.JwtAccessSecret, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	id, name, err := svc.ValidateAccessToken(ctx, tokenString)

	assert.NoError(t, err)
	assert.Equal(t, userID, id)
	assert.Empty(t, name)
}

func TestValidateAccessToken_InvalidSignature(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	tokenString := generateTestToken("wrong-secret", jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	id, name, err := svc.ValidateAccessToken(ctx, tokenString)

	assert.Error(t, err)
	assert.Empty(t, id)
	assert.Empty(t, name)
	assert.Contains(t, err.Error(), "invalid access token")
}

func TestValidateAccessToken_Expired(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	cfg := newTestConfig()
	tokenString := generateTestToken(cfg.JwtAccessSecret, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	})

	id, name, err := svc.ValidateAccessToken(ctx, tokenString)

	assert.Error(t, err)
	assert.Empty(t, id)
	assert.Empty(t, name)
	assert.Contains(t, err.Error(), "invalid access token")
}

func TestIssueTokens_Success(t *testing.T) {
	svc, tokenRepo, _ := newService()
	ctx := context.Background()
	user := newTestUser()

	tokenRepo.On("SaveRefreshToken", ctx, user.ID, mock.Anything, mock.Anything).Return(nil)

	dto, refreshToken, err := svc.IssueTokens(ctx, user)

	assert.NoError(t, err)
	assert.NotEmpty(t, dto.AccessToken)
	assert.NotEmpty(t, refreshToken)
	// Если у AuthResponseDTO есть поле Username или User, можно проверить:
	// assert.Equal(t, user.Username, dto.User.Username) или dto.Username
	tokenRepo.AssertExpectations(t)
}

func TestIssueTokens_SaveRefreshTokenError(t *testing.T) {
	svc, tokenRepo, _ := newService()
	ctx := context.Background()
	user := newTestUser()
	dbErr := errors.New("db error")

	tokenRepo.On("SaveRefreshToken", ctx, user.ID, mock.Anything, mock.Anything).Return(dbErr)

	dto, refreshToken, err := svc.IssueTokens(ctx, user)

	assert.Error(t, err)
	assert.Empty(t, dto.AccessToken)
	assert.Empty(t, refreshToken)
	assert.Contains(t, err.Error(), "save refresh token")
	tokenRepo.AssertExpectations(t)
}

func TestRefresh_Success(t *testing.T) {
	svc, tokenRepo, userRepo := newService()
	ctx := context.Background()
	cfg := newTestConfig()

	userID := "user-123"
	user := newTestUser()
	user.ID = userID

	validToken := generateTestToken(cfg.JwtRefreshSecret, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	tokenHash := hashToken(validToken)
	stored := domain_models.RefreshToken{
		ID:        "token-id",
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	tokenRepo.On("GetRefreshToken", ctx, tokenHash).Return(stored, nil)
	userRepo.On("GetUser", ctx, userID).Return(user, nil)
	tokenRepo.On("ReplaceRefreshToken", ctx, tokenHash, userID, mock.Anything, mock.Anything).Return(nil)

	accessToken, newRefreshToken, err := svc.Refresh(ctx, validToken)

	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, newRefreshToken)
	tokenRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()

	accessToken, refreshToken, err := svc.Refresh(ctx, "invalid-token")

	assert.Error(t, err)
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)
	assert.Contains(t, err.Error(), "validate refresh token")
}

func TestRefresh_TokenNotFound(t *testing.T) {
	svc, tokenRepo, _ := newService()
	ctx := context.Background()
	cfg := newTestConfig()
	validToken := generateTestToken(cfg.JwtRefreshSecret, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	tokenRepo.On("GetRefreshToken", ctx, hashToken(validToken)).Return(domain_models.RefreshToken{}, errors.New("not found"))

	accessToken, refreshToken, err := svc.Refresh(ctx, validToken)

	assert.Error(t, err)
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)
	assert.Contains(t, err.Error(), "token not found or already revoked")
}

func TestRefresh_TokenUserMismatch(t *testing.T) {
	svc, tokenRepo, _ := newService()
	ctx := context.Background()
	cfg := newTestConfig()
	validToken := generateTestToken(cfg.JwtRefreshSecret, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	tokenHash := hashToken(validToken)
	stored := domain_models.RefreshToken{
		ID:        "token-id",
		UserID:    "different-user",
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
	tokenRepo.On("GetRefreshToken", ctx, tokenHash).Return(stored, nil)

	accessToken, refreshToken, err := svc.Refresh(ctx, validToken)

	assert.Error(t, err)
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)
	assert.Contains(t, err.Error(), "token user mismatch")
}

func TestRefresh_TokenExpiredInDB(t *testing.T) {
	svc, tokenRepo, _ := newService()
	ctx := context.Background()
	cfg := newTestConfig()
	userID := "user-123"
	validToken := generateTestToken(cfg.JwtRefreshSecret, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	tokenHash := hashToken(validToken)
	stored := domain_models.RefreshToken{
		ID:        "token-id",
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now(),
	}
	tokenRepo.On("GetRefreshToken", ctx, tokenHash).Return(stored, nil)

	accessToken, refreshToken, err := svc.Refresh(ctx, validToken)

	assert.Error(t, err)
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)
	assert.Contains(t, err.Error(), "refresh token expired")
}

func TestRefresh_GetUserError(t *testing.T) {
	svc, tokenRepo, userRepo := newService()
	ctx := context.Background()
	cfg := newTestConfig()
	userID := "user-123"
	validToken := generateTestToken(cfg.JwtRefreshSecret, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	tokenHash := hashToken(validToken)
	stored := domain_models.RefreshToken{
		ID:        "token-id",
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
	tokenRepo.On("GetRefreshToken", ctx, tokenHash).Return(stored, nil)
	userRepo.On("GetUser", ctx, userID).Return(domain_models.User{}, errors.New("db error"))

	accessToken, refreshToken, err := svc.Refresh(ctx, validToken)

	assert.Error(t, err)
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)
	assert.Contains(t, err.Error(), "get user")
}

func TestRefresh_ReplaceRefreshTokenError(t *testing.T) {
	svc, tokenRepo, userRepo := newService()
	ctx := context.Background()
	cfg := newTestConfig()
	userID := "user-123"
	user := newTestUser()
	user.ID = userID

	validToken := generateTestToken(cfg.JwtRefreshSecret, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	tokenHash := hashToken(validToken)
	stored := domain_models.RefreshToken{
		ID:        "token-id",
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
	tokenRepo.On("GetRefreshToken", ctx, tokenHash).Return(stored, nil)
	userRepo.On("GetUser", ctx, userID).Return(user, nil)
	tokenRepo.On("ReplaceRefreshToken", ctx, tokenHash, userID, mock.Anything, mock.Anything).Return(errors.New("replace error"))

	accessToken, refreshToken, err := svc.Refresh(ctx, validToken)

	assert.Error(t, err)
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)
	assert.Contains(t, err.Error(), "replace refresh token")
}
