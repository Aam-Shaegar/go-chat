package messages_service

import (
	"context"
	"fmt"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
)

type GetMessagesResult struct {
	Messages   []domain_models.Message `json:"messages"`
	NextCursor *time.Time              `json:"next_cursor,omitempty"`
	HasMore    bool                    `json:"has_more"`
}

func (s *MessagesService) GetMessages(ctx context.Context, roomID, userID string, before *time.Time, limit int) (GetMessagesResult, error) {
	room, err := s.roomRepo.GetRoom(ctx, roomID)
	if err != nil {
		return GetMessagesResult{}, fmt.Errorf("get room: %w", err)
	}

	if room.IsPrivate || room.IsDM {
		isMember, err := s.roomRepo.IsMember(ctx, roomID, userID)
		if err != nil {
			return GetMessagesResult{}, fmt.Errorf("check membership: %w", err)
		}
		if !isMember {
			return GetMessagesResult{}, fmt.Errorf("access denied: %w", core_error.ErrUnauthorized)
		}
	}

	// Запрашиваем limit+1, чтобы понять, есть ли ещё страница
	messages, err := s.repo.GetMessages(ctx, roomID, before, limit+1)
	if err != nil {
		return GetMessagesResult{}, fmt.Errorf("get messages: %w", err)
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	var nextCursor *time.Time
	if hasMore && len(messages) > 0 {
		nextCursor = &messages[0].CreatedAt
	}

	return GetMessagesResult{
		Messages:   messages,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
