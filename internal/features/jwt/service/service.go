package jwt_service

import (
	"context"
	"time"

	core_config "go-chat/internal/core/config"
	domain_models "go-chat/internal/core/domain/models"
)

type JwtService struct {
	tokenRepository TokenRepository
	userRepository  UserRepository
	cfg             *core_config.Config
}

func NewJwtService(tokenRepository TokenRepository, userRepository UserRepository, cfg *core_config.Config) *JwtService {
	return &JwtService{
		tokenRepository: tokenRepository,
		userRepository:  userRepository,
		cfg:             cfg,
	}
}

type TokenRepository interface {
	SaveRefreshToken(ctx context.Context, userID, token string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, token string) (RefreshTokenModel, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	// ReplaceRefreshToken атомарно отзывает старый и сохраняет новый токен
	ReplaceRefreshToken(ctx context.Context, oldTokenHash, userID, newTokenHash string, expiresAt time.Time) error
}

type UserRepository interface {
	GetUser(ctx context.Context, userID string) (domain_models.User, error)
}
