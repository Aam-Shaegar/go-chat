package jwt_service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func (s *JwtService) GenerateAccessToken(ctx context.Context, userID, username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      userID,
		"username": username,
		"exp":      time.Now().Add(s.cfg.JwtAccessTTL).Unix(),
	})
	signed, err := token.SignedString([]byte(s.cfg.JwtAccessSecret))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

func (s *JwtService) GenerateRefreshToken(ctx context.Context, userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(s.cfg.JwtRefreshTTL).Unix(),
	})
	signed, err := token.SignedString([]byte(s.cfg.JwtRefreshSecret))
	if err != nil {
		return "", fmt.Errorf("sign refresh token: %w", err)
	}
	return signed, nil
}
