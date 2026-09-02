package rooms_service

import (
	"context"
	domain_models "go-chat/internal/core/domain/models"
)

func (s *RoomsService) GetUser(ctx context.Context, userID string) (domain_models.User, error) {
	return s.repo.GetUser(ctx, userID)
}
