package messages_service

import (
	"context"

	domain_models "go-chat/internal/core/domain/models"
)

type MessagesService struct {
	repo     Repository
	roomRepo RoomRepository
}

func NewMessagesService(repo Repository, roomRepo RoomRepository) *MessagesService {
	return &MessagesService{repo: repo, roomRepo: roomRepo}
}

type Repository interface {
	GetMessages(ctx context.Context, roomID string, before *domain_models.MessageCursor, limit int) ([]domain_models.Message, error)
}

type RoomRepository interface {
	IsMember(ctx context.Context, roomID, userID string) (bool, error)
	GetRoom(ctx context.Context, roomID string) (domain_models.Room, error)
}
