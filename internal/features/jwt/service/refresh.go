package jwt_service

import (
	"context"
	"fmt"
	"time"

	core_error "go-chat/internal/core/errors"
)

func (s *JwtService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	userID, err := s.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("validate refresh token: %w", core_error.ErrUnauthorized)
	}

	stored, err := s.tokenRepository.GetRefreshToken(ctx, HashToken(refreshToken))
	if err != nil {
		return "", "", fmt.Errorf("token not found or already revoked: %w", core_error.ErrUnauthorized)
	}
	if stored.UserID != userID {
		return "", "", fmt.Errorf("token user mismatch: %w", core_error.ErrUnauthorized)
	}
	if time.Now().After(stored.ExpiresAt) {
		return "", "", fmt.Errorf("refresh token expired: %w", core_error.ErrUnauthorized)
	}

	user, err := s.userRepository.GetUser(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("get user: %w", core_error.ErrUnauthorized)
	}

	newRefreshToken, err := s.GenerateRefreshToken(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	newHash := HashToken(newRefreshToken)
	oldHash := HashToken(refreshToken)
	if err := s.tokenRepository.ReplaceRefreshToken(
		ctx, oldHash, userID, newHash, time.Now().Add(s.cfg.JwtRefreshTTL),
	); err != nil {
		return "", "", fmt.Errorf("replace refresh token: %w", err)
	}

	accessToken, err := s.GenerateAccessToken(ctx, userID, user.Username)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	return accessToken, newRefreshToken, nil
}
