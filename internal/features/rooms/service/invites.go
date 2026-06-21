package rooms_service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	invite, err := s.repo.GetInviteByToken(ctx, token)
	if err != nil {
		return domain_models.Room{}, fmt.Errorf("invite not found: %w", core_error.ErrNotFound)
	}

	if !invite.CanBeUsed() {
		return domain_models.Room{}, fmt.Errorf("invite expired, exhausted or inactive: %w", core_error.ErrInvalidArgument)
	}

	// Проверяем членство до инкремента, чтобы не списать использование зря
	alreadyMember, err := s.repo.IsMember(ctx, invite.RoomID, userID)
	if err != nil {
		return domain_models.Room{}, fmt.Errorf("check membership: %w", err)
	}
	if alreadyMember {
		return domain_models.Room{}, fmt.Errorf("already a member: %w", core_error.ErrConflict)
	}

	if err := s.repo.TryIncrementInviteUses(ctx, token); err != nil {
		return domain_models.Room{}, fmt.Errorf("invite exhausted or inactive: %w", core_error.ErrInvalidArgument)
	}

	if err := s.repo.AddMember(ctx, invite.RoomID, userID, domain_models.MemberRoleMember); err != nil {
		return domain_models.Room{}, fmt.Errorf("add member: %w", err)
	}

	return s.repo.GetRoom(ctx, invite.RoomID)
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
