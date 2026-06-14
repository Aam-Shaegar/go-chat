package jwt_service

import (
	"context"
	"fmt"
	core_error "go-chat/internal/core/errors"

	"github.com/golang-jwt/jwt/v5"
)

func (s *JwtService) ValidateRefreshToken(ctx context.Context, tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %w", core_error.ErrUnauthorized)
		}
		return []byte(s.cfg.JwtRefreshSecret), nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid refresh token: %w", core_error.ErrUnauthorized)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid claims: %w", core_error.ErrUnauthorized)
	}
	userID, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("invalid subject: %w", core_error.ErrUnauthorized)
	}
	return userID, nil
}

func (s *JwtService) ValidateAccessToken(ctx context.Context, tokenString string) (string, string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %w", core_error.ErrUnauthorized)
		}
		return []byte(s.cfg.JwtAccessSecret), nil
	})
	if err != nil || !token.Valid {
		return "", "", fmt.Errorf("invalid access token: %w", core_error.ErrUnauthorized)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", fmt.Errorf("invalid claims: %w", core_error.ErrUnauthorized)
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return "", "", fmt.Errorf("invalid subject: %w", core_error.ErrUnauthorized)
	}

	// username опционален — если нет в токене, вернём пустую строку
	username, _ := claims["username"].(string)

	return userID, username, nil
}
