package service

import (
	"context"
	"fmt"
	"go-chat/internal/domain"
	"go-chat/internal/repository"
	"time"
)

type InviteService struct {
	inviteRepo *repository.InviteRepository
	roomRepo   *repository.RoomRepository
}

func NewInviteService(inviteRepo *repository.InviteRepository, roomRepo *repository.RoomRepository) *InviteService {
	return &InviteService{
		inviteRepo: inviteRepo,
		roomRepo:   roomRepo,
	}
}

func (s *InviteService) Create(ctx context.Context, roomID, userID string) (domain.RoomInvite, error) {
	ownerID, err := s.roomRepo.GetOwnerID(ctx, roomID)
	if err != nil {
		return domain.RoomInvite{}, fmt.Errorf("room not found")
	}
	if ownerID != userID {
		return domain.RoomInvite{}, fmt.Errorf("only room owner can create invites")
	}
	return s.inviteRepo.Create(ctx, roomID, userID)
}

func (s *InviteService) Accept(ctx context.Context, inviteID, userID string) (domain.Room, error) {
	invite, err := s.inviteRepo.GetByID(ctx, inviteID)
	if err != nil {
		return domain.Room{}, fmt.Errorf("invite not found")
	}
	if invite.UsedBy != nil {
		return domain.Room{}, fmt.Errorf("invite already used")
	}
	if time.Now().After(invite.ExpiresAt) {
		return domain.Room{}, fmt.Errorf("invite expired")
	}
	isMember, _ := s.roomRepo.IsMember(ctx, invite.RoomID, userID)
	if !isMember {
		if err := s.roomRepo.AddMember(ctx, invite.RoomID, userID); err != nil {
			return domain.Room{}, fmt.Errorf("join room:%w", err)
		}
	}
	if err := s.inviteRepo.MarkUsed(ctx, inviteID, userID); err != nil {
		return domain.Room{}, fmt.Errorf("mark invite used: %w", err)
	}
	room, err := s.roomRepo.GetRoomByID(ctx, invite.RoomID)
	if err != nil {
		return domain.Room{}, fmt.Errorf("get room: %w", err)
	}
	return room, nil
}

func (s *InviteService) ListByRoom(ctx context.Context, roomID, userID string) ([]domain.RoomInvite, error) {
	ownerID, err := s.roomRepo.GetOwnerID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("room not found")
	}
	if ownerID != userID {
		return nil, fmt.Errorf("only room owner can view invites")
	}
	return s.inviteRepo.ListByRoom(ctx, roomID)
}

func (s *InviteService) Delete(ctx context.Context, inviteID, userID string) error {
	invite, err := s.inviteRepo.GetByID(ctx, inviteID)
	if err != nil {
		return fmt.Errorf("invite not found")
	}
	ownerID, err := s.roomRepo.GetOwnerID(ctx, invite.RoomID)
	if err != nil || ownerID != userID {
		return fmt.Errorf("permission denied")
	}
	return s.inviteRepo.Delete(ctx, inviteID)
}

func (s *InviteService) GetByID(ctx context.Context, inviteID string) (domain.RoomInvite, error) {
	return s.inviteRepo.GetByID(ctx, inviteID)
}
