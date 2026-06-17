package reads_service

import (
	"context"
	"fmt"
	"time"

	core_error "go-chat/internal/core/errors"
)

type ReadsService struct {
	repo     Repository
	roomRepo RoomRepository
}

func NewReadsService(repo Repository, roomRepo RoomRepository) *ReadsService {
	return &ReadsService{repo: repo, roomRepo: roomRepo}
}

type Repository interface {
	MarkRead(ctx context.Context, roomID, userID string, lastReadAt time.Time) error
	GetUnreadCounts(ctx context.Context, userID string) (map[string]int64, error)
	GetUnreadCount(ctx context.Context, roomID, userID string) (int64, error)
}

type RoomRepository interface {
	IsMember(ctx context.Context, roomID, userID string) (bool, error)
}

func (s *ReadsService) MarkRead(ctx context.Context, roomID, userID string) error {
	isMember, err := s.roomRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return fmt.Errorf("access denied: %w", core_error.ErrUnauthorized)
	}
	return s.repo.MarkRead(ctx, roomID, userID, time.Now())
}

func (s *ReadsService) GetUnreadCounts(ctx context.Context, userID string) (map[string]int64, error) {
	return s.repo.GetUnreadCounts(ctx, userID)
}

func (s *ReadsService) GetUnreadCount(ctx context.Context, roomID, userID string) (int64, error) {
	isMember, err := s.roomRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return 0, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return 0, fmt.Errorf("access denied: %w", core_error.ErrUnauthorized)
	}
	return s.repo.GetUnreadCount(ctx, roomID, userID)
}
