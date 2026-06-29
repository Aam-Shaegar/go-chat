package rooms_service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
)

func (s *RoomsService) CreateInvite(ctx context.Context, roomID, userID string, maxUses int, ttl *time.Duration) (domain_models.RoomInvite, error) {
	member, err := s.repo.GetMember(ctx, roomID, userID)
	if err != nil {
		return domain_models.RoomInvite{}, fmt.Errorf("get member: %w", err)
	}
	if !member.IsAdmin() {
		return domain_models.RoomInvite{}, fmt.Errorf("only admin or owner can create invites: %w", core_error.ErrUnauthorized)
	}

	token, err := generateToken()
	if err != nil {
		return domain_models.RoomInvite{}, fmt.Errorf("generate token: %w", err)
	}

	d := defaultInviteTTL
	if ttl != nil {
		d = *ttl
	}
	t := time.Now().Add(d)

	if maxUses < 0 {
		maxUses = 1
	}

	return s.repo.CreateInvite(ctx, domain_models.NewRoomInvite(
		"", roomID, token, userID, maxUses, &t, time.Now(),
	))
}

func (s *RoomsService) AcceptInvite(ctx context.Context, token, userID string) (domain_models.Room, error) {
	if token == "" {
		return domain_models.Room{}, fmt.Errorf("token is required: %w", core_error.ErrInvalidArgument)
	}

	room, err := s.repo.AcceptInvite(ctx, token, userID)
	if err != nil {
		if errors.Is(err, core_error.ErrConflict) {
			return domain_models.Room{}, err
		}
		return domain_models.Room{}, fmt.Errorf("invite expired, exhausted, inactive or already used: %w", core_error.ErrInvalidArgument)
	}
	return room, nil
}

func (s *RoomsService) GetRoomInvites(ctx context.Context, roomID, userID string) ([]domain_models.RoomInvite, error) {
	member, err := s.repo.GetMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("get member: %w", err)
	}
	if !member.IsAdmin() {
		return nil, fmt.Errorf("only admin or owner can view invites: %w", core_error.ErrUnauthorized)
	}
	return s.repo.GetRoomInvites(ctx, roomID)
}

func (s *RoomsService) DeactivateInvite(ctx context.Context, token, userID string) error {
	return s.repo.DeactivateInvite(ctx, token, userID)
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand read: %w", err)
	}
	return hex.EncodeToString(b), nil
}
