package rooms_service

import (
	"context"
	"fmt"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
)

func (s *RoomsService) CreateRoom(ctx context.Context, name, description, ownerID string, isPrivate bool) (domain_models.Room, error) {
	if name == "" {
		return domain_models.Room{}, fmt.Errorf("name is required: %w", core_error.ErrInvalidArgument)
	}
	return s.repo.CreateRoom(ctx, domain_models.NewRoom(
		"", name, description, isPrivate, false, ownerID, time.Now(),
	))
}

func (s *RoomsService) GetRoom(ctx context.Context, roomID, userID string) (domain_models.Room, error) {
	room, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return domain_models.Room{}, fmt.Errorf("get room: %w", err)
	}
	if room.IsPrivate {
		isMember, err := s.repo.IsMember(ctx, roomID, userID)
		if err != nil {
			return domain_models.Room{}, fmt.Errorf("check membership: %w", err)
		}
		if !isMember {
			return domain_models.Room{}, fmt.Errorf("access denied: %w", core_error.ErrUnauthorized)
		}
	}
	return room, nil
}

func (s *RoomsService) GetPublicRooms(ctx context.Context, limit, offset int) ([]domain_models.Room, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetPublicRooms(ctx, limit, offset)
}

func (s *RoomsService) GetUserRooms(ctx context.Context, userID string) ([]domain_models.Room, error) {
	return s.repo.GetUserRooms(ctx, userID)
}

func (s *RoomsService) DeleteRoom(ctx context.Context, roomID, userID string) error {
	room, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if room.OwnerID != userID {
		return fmt.Errorf("only owner can delete room: %w", core_error.ErrUnauthorized)
	}
	return s.repo.DeleteRoom(ctx, roomID, userID)
}
