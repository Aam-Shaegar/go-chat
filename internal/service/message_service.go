package service

import (
	"context"
	"fmt"
	"go-chat/internal/domain"
	"go-chat/internal/repository"
)

type MessageService struct {
	messageRepo *repository.MessageRepository
	roomRepo    *repository.RoomRepository
}

func NewMessageService(messageRepo *repository.MessageRepository, roomRepo *repository.RoomRepository) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
		roomRepo:    roomRepo,
	}
}

func (s *MessageService) Delete(ctx context.Context, messageID, roomID, userID string) error {
	msg, err := s.messageRepo.GetMsgByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("message not found")
	}
	if msg.RoomID != roomID {
		return fmt.Errorf("message does not belong to this room")
	}
	if msg.UserID != userID {
		ownerID, err := s.roomRepo.GetOwnerID(ctx, roomID)
		if err != nil || ownerID != userID {
			return fmt.Errorf("permission denied")
		}
	}
	if err := s.messageRepo.Delete(ctx, messageID); err != nil {
		return fmt.Errorf("delete message: %w", err)
	}

	return nil
}

func (s *MessageService) ListByRoom(ctx context.Context, roomID string, limit, offset int) ([]domain.Message, error) {
	return s.messageRepo.CountByRoom(ctx, roomID, limit, offset)
}
