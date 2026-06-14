package jwt_service

import (
	"context"
	"fmt"
	"time"

	domain_dtos "go-chat/internal/core/domain/dtos"
	domain_models "go-chat/internal/core/domain/models"
)

func (s *JwtService) IssueTokens(ctx context.Context, user domain_models.User) (domain_dtos.AuthResponseDTO, string, error) {
	accessToken, err := s.GenerateAccessToken(ctx, user.ID, user.Username)
	if err != nil {
		return domain_dtos.AuthResponseDTO{}, "", fmt.Errorf("generate access token: %w", err)
	}
	refreshToken, err := s.GenerateRefreshToken(ctx, user.ID)
	if err != nil {
		return domain_dtos.AuthResponseDTO{}, "", fmt.Errorf("generate refresh token: %w", err)
	}
	err = s.tokenRepository.SaveRefreshToken(ctx, user.ID, hashToken(refreshToken), time.Now().Add(s.cfg.JwtRefreshTTL))
	if err != nil {
		return domain_dtos.AuthResponseDTO{}, "", fmt.Errorf("save refresh token: %w", err)
	}
	dto := domain_dtos.NewAuthResponseDTO(user, accessToken)
	return dto, refreshToken, nil
}
