package service

import (
	"context"
	"fmt"
	"go-chat/internal/domain"
	"go-chat/internal/repository"
	"time"
)

type DMService struct {
	dmRepo   *repository.DMRepository
	userRepo *repository.UserRepository
}

func NewDMService(dmRepo *repository.DMRepository, userRepo *repository.UserRepository) *DMService {
	return &DMService{
		dmRepo:   dmRepo,
		userRepo: userRepo,
	}
}

func (s *DMService) Send(ctx context.Context, fromID, toID, content string) (domain.DirectMessage, error) {
	if content == "" {
		return domain.DirectMessage{}, fmt.Errorf("content cannot be empty")
	}
	if len(content) > 4000 {
		return domain.DirectMessage{}, fmt.Errorf("message too long")
	}
	if fromID == toID {
		return domain.DirectMessage{}, fmt.Errorf("cannot send message to self")
	}
	_, err := s.userRepo.GetByID(ctx, toID)
	if err != nil {
		return domain.DirectMessage{}, fmt.Errorf("user not found")
	}
	return s.dmRepo.Create(ctx, domain.DirectMessage{
		FromUserID: fromID,
		ToUserID:   toID,
		Content:    content,
	})
}

func (s *DMService) GetHistory(ctx context.Context, userA, userB string, limit, offset int) ([]domain.DirectMessage, error) {
	return s.dmRepo.GetHistory(ctx, userA, userB, limit, offset)
}

func (s *DMService) GetConversations(ctx context.Context, userID string) ([]domain.DirectMessage, error) {
	return s.dmRepo.GetConversations(ctx, userID)
}

func (s *DMService) MarkRead(ctx context.Context, userID, otherUserID string) error {
	return s.dmRepo.MarkRead(ctx, userID, otherUserID)
}

func (s *DMService) GetAllUnreadCounts(ctx context.Context, userID string) ([]repository.DMUnread, error) {
	return s.dmRepo.GetAllUnreadCounts(ctx, userID)
}

func (s *DMService) GetLastReadAt(ctx context.Context, userID, otherUserID string) (time.Time, error) {
	return s.dmRepo.GetLastReadAt(ctx, userID, otherUserID)
}
