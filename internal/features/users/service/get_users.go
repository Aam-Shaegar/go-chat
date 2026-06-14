package users_service

import (
	"context"
	"fmt"
	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
)

func (s *UsersService) GetUsers(ctx context.Context, limit, offset *int) ([]domain_models.User, error) {
	if limit != nil && *limit < 0 {
		return nil, fmt.Errorf("limit must be non-negative: %w", core_error.ErrInvalidArgument)
	}
	if offset != nil && *offset < 0 {
		return nil, fmt.Errorf("offset must be non-negative: %w", core_error.ErrInvalidArgument)
	}
	users, err := s.usersRepository.GetUsers(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get users from repository: %w", err)
	}
	return users, nil
}
