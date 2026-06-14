package jwt_service

import (
	"context"
	"fmt"
	"time"

	core_error "go-chat/internal/core/errors"
)

func (s *JwtService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	// 1. Валидируем подпись и срок действия
	userID, err := s.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("validate refresh token: %w", core_error.ErrUnauthorized)
	}

	// 2. Проверяем наличие в БД
	stored, err := s.tokenRepository.GetRefreshToken(ctx, hashToken(refreshToken))
	if err != nil {
		return "", "", fmt.Errorf("token not found or already revoked: %w", core_error.ErrUnauthorized)
	}

	// 3. Сверяем userID из токена с userID из БД
	if stored.UserID != userID {
		return "", "", fmt.Errorf("token user mismatch: %w", core_error.ErrUnauthorized)
	}

	// 4. Проверяем срок по БД
	if time.Now().After(stored.ExpiresAt) {
		return "", "", fmt.Errorf("refresh token expired: %w", core_error.ErrUnauthorized)
	}

	// 5. Получаем username из БД — нужен для access token claims
	user, err := s.userRepository.GetUser(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("get user: %w", core_error.ErrUnauthorized)
	}

	// 6. Генерируем новый refresh token
	newRefreshToken, err := s.GenerateRefreshToken(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	// 7. Атомарно: сохраняем новый и отзываем старый в одной транзакции
	// Если упадёт — клиент остаётся со старым токеном, может повторить
	newHash := hashToken(newRefreshToken)
	oldHash := hashToken(refreshToken)
	if err := s.tokenRepository.ReplaceRefreshToken(
		ctx, oldHash, userID, newHash, time.Now().Add(s.cfg.JwtRefreshTTL),
	); err != nil {
		return "", "", fmt.Errorf("replace refresh token: %w", err)
	}

	// 8. Генерируем access token с username
	accessToken, err := s.GenerateAccessToken(ctx, userID, user.Username)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	return accessToken, newRefreshToken, nil
}
