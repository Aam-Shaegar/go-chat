package users_service

import (
	"context"
	"fmt"
	"strings"
	"time"

	domain_dtos "go-chat/internal/core/domain/dtos"
	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"

	"golang.org/x/crypto/bcrypt"
)

func (s *UsersService) Register(ctx context.Context, username, email, password string) (domain_dtos.AuthResponseDTO, string, error) {
	username = strings.TrimSpace(username)
	email = strings.ToLower(strings.TrimSpace(email))

	if username == "" || email == "" || password == "" {
		return domain_dtos.AuthResponseDTO{}, "", fmt.Errorf(
			"username, email, password are required: %w",
			core_error.ErrInvalidArgument,
		)
	}
	exists, err := s.usersRepository.UserExistsByEmail(ctx, email)
	if err != nil {
		return domain_dtos.AuthResponseDTO{}, "", fmt.Errorf("check email exists: %w", err)
	}
	if exists {
		return domain_dtos.AuthResponseDTO{}, "", fmt.Errorf(
			"email already taken: %w",
			core_error.ErrConflict,
		)
	}
	exists, err = s.usersRepository.UserExistsByUsername(ctx, username)
	if err != nil {
		return domain_dtos.AuthResponseDTO{}, "", fmt.Errorf("check username exists: %w", err)
	}
	if exists {
		return domain_dtos.AuthResponseDTO{}, "", fmt.Errorf(
			"username already taken: %w",
			core_error.ErrConflict,
		)
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain_dtos.AuthResponseDTO{}, "", fmt.Errorf("hash password: %w", err)
	}
	user := domain_models.NewUser("", username, email, string(hashed), time.Now(), time.Now())
	createdUser, err := s.usersRepository.CreateUser(ctx, user)
	if err != nil {
		return domain_dtos.AuthResponseDTO{}, "", fmt.Errorf("create user: %w", err)
	}
	return s.authService.IssueTokens(ctx, createdUser)
}
