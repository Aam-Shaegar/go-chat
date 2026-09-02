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

	var expiresAt *time.Time
	if ttl != nil {
		t := time.Now().Add(*ttl)
		expiresAt = &t
	}

	if maxUses < 0 {
		maxUses = 0
	}

	return s.repo.CreateInvite(ctx, domain_models.NewRoomInvite(
		"", roomID, token, userID, maxUses, 0, expiresAt, time.Now(),
	))
}

func (s *RoomsService) AcceptInvite(ctx context.Context, token, userID string) (domain_models.Room, domain_models.RoomInvite, error) {
	if token == "" {
		return domain_models.Room{}, domain_models.RoomInvite{}, fmt.Errorf("token is required: %w", core_error.ErrInvalidArgument)
	}

	room, invite, err := s.repo.AcceptInvite(ctx, token, userID)
	if err != nil {
		if errors.Is(err, core_error.ErrConflict) {
			return domain_models.Room{}, domain_models.RoomInvite{}, err
		}
		return domain_models.Room{}, domain_models.RoomInvite{}, fmt.Errorf("invite expired, exhausted, inactive or already used: %w", core_error.ErrInvalidArgument)
	}

	banned, err := s.repo.IsBanned(ctx, room.ID, userID)
	if err != nil {
		return domain_models.Room{}, domain_models.RoomInvite{}, fmt.Errorf("check ban: %w", err)
	}
	if banned {
		ban, _ := s.repo.GetBan(ctx, room.ID, userID)
		if ban.IsExpired() {
			_ = s.repo.RemoveBan(ctx, room.ID, userID)
		} else {
			return domain_models.Room{}, domain_models.RoomInvite{}, fmt.Errorf("you are banned from this room: %w", core_error.ErrUnauthorized)
		}
	}

	return room, invite, nil
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

func (s *RoomsService) GetInviteByToken(ctx context.Context, token string) (domain_models.RoomInvite, error) {
	return s.repo.GetInviteByToken(ctx, token)
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
