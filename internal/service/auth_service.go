package service

import (
	"context"
	"fmt"

	"go-chat/internal/domain"
	"go-chat/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo     *repository.UserRepository
	tokenService *TokenService
}

func NewAuthService(userRepo *repository.UserRepository, tokenService *TokenService) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		tokenService: tokenService,
	}
}

func (s *AuthService) Register(ctx context.Context, input domain.RegisterInput) (domain.AuthResponse, string, error) {
	emailExists, err := s.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return domain.AuthResponse{}, "", fmt.Errorf("check email: %w", err)
	}
	if emailExists {
		return domain.AuthResponse{}, "", fmt.Errorf("Email alredy taken")
	}

	usernameExists, err := s.userRepo.ExistsByUsername(ctx, input.Username)
	if err != nil {
		return domain.AuthResponse{}, "", fmt.Errorf("check username: %w", err)
	}
	if usernameExists {
		return domain.AuthResponse{}, "", fmt.Errorf("username already taken")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.AuthResponse{}, "", fmt.Errorf("hash password: %w", err)
	}

	user, err := s.userRepo.Create(ctx, domain.User{
		Username: input.Username,
		Email:    input.Email,
		Password: string(hashed),
	})

	if err != nil {
		return domain.AuthResponse{}, "", fmt.Errorf("create user: %w", err)
	}
	return s.issueTokens(user)
}

func (s *AuthService) Login(ctx context.Context, input domain.LoginInput) (domain.AuthResponse, string, error) {
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return domain.AuthResponse{}, "", fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return domain.AuthResponse{}, "", fmt.Errorf("invalid credentials")
	}

	return s.issueTokens(user)
}

func (s *AuthService) issueTokens(user domain.User) (domain.AuthResponse, string, error) {
	accessToken, err := s.tokenService.GenerateAccess(user.ID)
	if err != nil {
		return domain.AuthResponse{}, "", fmt.Errorf("generate access token: %w", err)
	}
	refreshToken, err := s.tokenService.GenerateRefresh(user.ID)
	if err != nil {
		return domain.AuthResponse{}, "", fmt.Errorf("generate refresh token: %w", err)
	}
	return domain.AuthResponse{
		User:        user,
		AccessToken: accessToken,
	}, refreshToken, nil
}
