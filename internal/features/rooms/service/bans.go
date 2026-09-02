package rooms_service

import (
	"context"
	"fmt"
	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
	"time"
)

func (s *RoomsService) BanMember(ctx context.Context, roomID, requesterID, targetUserID, reason string, expiresAt *time.Time) error {
	if requesterID == targetUserID {
		return fmt.Errorf("cannot ban yourself: %w", core_error.ErrInvalidArgument)
	}

	room, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if room.IsDM {
		return fmt.Errorf("cannot ban in DM: %w", core_error.ErrInvalidArgument)
	}

	requester, err := s.repo.GetMember(ctx, roomID, requesterID)
	if err != nil {
		return fmt.Errorf("get requester: %w", err)
	}
	if !requester.IsAdmin() {
		return fmt.Errorf("only admin or owner can ban: %w", core_error.ErrUnauthorized)
	}

	target, err := s.repo.GetMember(ctx, roomID, targetUserID)
	if err != nil {
		return fmt.Errorf("get target: %w", err)
	}
	if target.IsOwner() {
		return fmt.Errorf("cannot ban room owner: %w", core_error.ErrUnauthorized)
	}
	if target.Role == domain_models.MemberRoleAdmin && !requester.IsOwner() {
		return fmt.Errorf("only owner can ban admins: %w", core_error.ErrUnauthorized)
	}

	banned, _ := s.repo.IsBanned(ctx, roomID, targetUserID)
	if banned {
		return fmt.Errorf("user already banned: %w", core_error.ErrConflict)
	}

	ban := domain_models.NewRoomBan(roomID, targetUserID, requesterID, reason, expiresAt)
	if err := s.repo.CreateBan(ctx, ban); err != nil {
		return fmt.Errorf("create ban: %w", err)
	}

	if err := s.repo.RemoveMember(ctx, roomID, targetUserID); err != nil {
		_ = s.repo.RemoveBan(ctx, roomID, targetUserID)
		return fmt.Errorf("remove member: %w", err)
	}

	return nil
}

func (s *RoomsService) UnbanMember(ctx context.Context, roomID, requesterID, targetUserID string) error {
	room, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return err
	}
	if room.IsDM {
		return fmt.Errorf("cannot unban in DM: %w", core_error.ErrInvalidArgument)
	}

	requester, err := s.repo.GetMember(ctx, roomID, requesterID)
	if err != nil {
		return err
	}
	if !requester.IsAdmin() {
		return fmt.Errorf("only admin or owner: %w", core_error.ErrUnauthorized)
	}

	target, err := s.repo.GetMember(ctx, roomID, targetUserID)
	if err != nil {
		return err
	}
	if target.IsOwner() {
		return fmt.Errorf("cannot unban owner: %w", core_error.ErrUnauthorized)
	}
	if target.Role == domain_models.MemberRoleAdmin && !requester.IsOwner() {
		return fmt.Errorf("only owner can unban admins: %w", core_error.ErrUnauthorized)
	}

	return s.repo.RemoveBan(ctx, roomID, targetUserID)
}

func (s *RoomsService) GetRoomBans(ctx context.Context, roomID, userID string) ([]domain_models.RoomBan, error) {
	isMember, err := s.repo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, fmt.Errorf("access denied: %w", core_error.ErrUnauthorized)
	}

	return s.repo.GetRoomBans(ctx, roomID)
}
