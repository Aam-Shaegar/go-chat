package users_service

import (
	"context"
	domain_dtos "go-chat/internal/core/domain/dtos"
	domain_models "go-chat/internal/core/domain/models"
)

type UsersService struct {
	usersRepository UsersRepository
	authService     AuthService
}

func NewUsersService(usersRepository UsersRepository, authService AuthService) *UsersService {
	return &UsersService{
		usersRepository: usersRepository,
		authService:     authService,
	}
}

type UsersRepository interface {
	GetUsers(ctx context.Context, limit, offset *int) ([]domain_models.User, error)
	GetUser(ctx context.Context, userID string) (domain_models.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain_models.User, error)
	UserExistsByEmail(ctx context.Context, email string) (bool, error)
	UserExistsByUsername(ctx context.Context, username string) (bool, error)
	CreateUser(ctx context.Context, user domain_models.User) (domain_models.User, error)
}

type AuthService interface {
	IssueTokens(ctx context.Context, user domain_models.User) (domain_dtos.AuthResponseDTO, string, error)
}
