package users_service

import (
	"context"
	"fmt"

	domain_dtos "go-chat/internal/core/domain/dtos"
	core_error "go-chat/internal/core/errors"

	"golang.org/x/crypto/bcrypt"
)

func (s *UsersService) Login(ctx context.Context, email, password string) (domain_dtos.AuthResponseDTO, string, error) {
	if email == "" || password == "" {
		return domain_dtos.AuthResponseDTO{}, "", fmt.Errorf(
			"email and password are required: %w",
			core_error.ErrInvalidArgument,
		)
	}
	user, err := s.usersRepository.GetUserByEmail(ctx, email)
	if err != nil {
		return domain_dtos.AuthResponseDTO{}, "", fmt.Errorf(
			"invalid credentials: %w",
			core_error.ErrUnauthorized,
		)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return domain_dtos.AuthResponseDTO{}, "", fmt.Errorf(
			"invalid credentials: %w",
			core_error.ErrUnauthorized,
		)
	}
	return s.authService.IssueTokens(ctx, user)
}
