package users_service

import (
	"context"
	"fmt"
	domain_models "go-chat/internal/core/domain/models"
)

func (s *UsersService) GetUser(ctx context.Context, userID string) (domain_models.User, error) {
	user, err := s.usersRepository.GetUser(ctx, userID)
	if err != nil {
		return domain_models.User{}, fmt.Errorf("get user from repository: %w", err)
	}
	return user, nil
}
