package service

import (
	"context"
	"fmt"
	"go-chat/internal/domain"
	"go-chat/internal/repository"
)

type RoomService struct {
	roomRepo *repository.RoomRepository
}

func NewRoomService(roomRepo *repository.RoomRepository) *RoomService {
	return &RoomService{roomRepo: roomRepo}
}

type CreateRoomInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
}

func (s *RoomService) Create(ctx context.Context, input CreateRoomInput, ownerID string) (domain.Room, error) {
	if input.Name == "" {
		return domain.Room{}, fmt.Errorf("room name is required")
	}
	if len(input.Name) > 64 {
		return domain.Room{}, fmt.Errorf("room name is too long")
	}

	room, err := s.roomRepo.Create(ctx, domain.Room{
		Name:        input.Name,
		Description: input.Description,
		IsPrivate:   input.IsPrivate,
		OwnerID:     ownerID,
	})
	if err != nil {
		return domain.Room{}, fmt.Errorf("create room: %w", err)
	}
	if err := s.roomRepo.AddMember(ctx, room.ID, ownerID); err != nil {
		return domain.Room{}, fmt.Errorf("add owner as member: %w", err)
	}
	return room, nil
}

func (s *RoomService) GetRoomByID(ctx context.Context, roomID string) (domain.Room, error) {
	room, err := s.roomRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		return domain.Room{}, fmt.Errorf("room not found")
	}
	return room, nil
}

func (s *RoomService) ListPublic(ctx context.Context) ([]domain.Room, error) {
	rooms, err := s.roomRepo.ListPublic(ctx)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	if rooms == nil {
		rooms = []domain.Room{}
	}
	return rooms, nil
}

func (s *RoomService) ListMy(ctx context.Context, userID string) ([]domain.Room, error) {
	rooms, err := s.roomRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list my rooms: %w", err)
	}
	if rooms == nil {
		rooms = []domain.Room{}
	}
	return rooms, nil
}

func (s *RoomService) Join(ctx context.Context, roomID, userID string) error {
	room, err := s.roomRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("room not found")
	}
	if room.IsPrivate {
		return fmt.Errorf("room is private")
	}
	if err := s.roomRepo.AddMember(ctx, roomID, userID); err != nil {
		return fmt.Errorf("join room: %w", err)
	}
	return nil
}

func (s *RoomService) IsMember(ctx context.Context, roomID, userID string) (bool, error) {
	return s.roomRepo.IsMember(ctx, roomID, userID)
}

func (s *RoomService) Delete(ctx context.Context, roomID, userID string) error {
	ownerID, err := s.roomRepo.GetOwnerID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("room not found")
	}
	if ownerID != userID {
		return fmt.Errorf("only room owner can delete the room")
	}
	if err := s.roomRepo.Delete(ctx, roomID); err != nil {
		return fmt.Errorf("delete room: %w", err)
	}

	return nil
}

func (s *RoomService) GetOwnerID(ctx context.Context, roomID string) (string, error) {
	return s.roomRepo.GetOwnerID(ctx, roomID)
}
