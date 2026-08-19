package messages_service

import (
	"context"
	"fmt"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
)

func (s *MessagesService) EditMessage(ctx context.Context, roomID, userID, messageID, content string, updatedAt time.Time) (domain_models.Message, error) {
	if content == "" {
		return domain_models.Message{}, fmt.Errorf("content is required: %w", core_error.ErrInvalidArgument)
	}

	room, err := s.roomRepo.GetRoom(ctx, roomID)
	if err != nil {
		return domain_models.Message{}, fmt.Errorf("get room: %w", err)
	}

	if room.IsPrivate || room.IsDM {
		isMember, err := s.roomRepo.IsMember(ctx, roomID, userID)
		if err != nil {
			return domain_models.Message{}, fmt.Errorf("check membership: %w", err)
		}
		if !isMember {
			return domain_models.Message{}, fmt.Errorf("access denied: %w", core_error.ErrUnauthorized)
		}
	}

	msg, err := s.repo.EditMessage(ctx, roomID, userID, messageID, content, updatedAt)
	if err != nil {
		return domain_models.Message{}, fmt.Errorf("edit message: %w", err)
	}

	return msg, nil
}

func (s *MessagesService) DeleteMessage(ctx context.Context, roomID, userID, messageID string, deletedAt time.Time) (domain_models.Message, error) {
	room, err := s.roomRepo.GetRoom(ctx, roomID)
	if err != nil {
		return domain_models.Message{}, fmt.Errorf("get room: %w", err)
	}

	if room.IsPrivate || room.IsDM {
		isMember, err := s.roomRepo.IsMember(ctx, roomID, userID)
		if err != nil {
			return domain_models.Message{}, fmt.Errorf("check membership: %w", err)
		}
		if !isMember {
			return domain_models.Message{}, fmt.Errorf("access denied: %w", core_error.ErrUnauthorized)
		}
	}

	msg, err := s.repo.DeleteMessage(ctx, roomID, userID, messageID, deletedAt)
	if err != nil {
		return domain_models.Message{}, fmt.Errorf("delete message: %w", err)
	}

	return msg, nil
}