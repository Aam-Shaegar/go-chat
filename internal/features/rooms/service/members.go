package rooms_service

import (
	"context"
	"fmt"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
)

func (s *RoomsService) JoinPublicRoom(ctx context.Context, roomID, userID string) error {
	room, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if room.IsPrivate {
		return fmt.Errorf("room is private, use invite: %w", core_error.ErrUnauthorized)
	}
	if room.IsDM {
		return fmt.Errorf("cannot join DM room directly: %w", core_error.ErrInvalidArgument)
	}
	isMember, err := s.repo.IsMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if isMember {
		return fmt.Errorf("already a member: %w", core_error.ErrConflict)
	}
	return s.repo.AddMember(ctx, roomID, userID, domain_models.MemberRoleMember)
}

func (s *RoomsService) LeaveRoom(ctx context.Context, roomID, userID string) error {
	room, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if room.OwnerID == userID {
		return fmt.Errorf("owner cannot leave room, delete it instead: %w", core_error.ErrInvalidArgument)
	}
	return s.repo.RemoveMember(ctx, roomID, userID)
}

func (s *RoomsService) KickMember(ctx context.Context, roomID, requesterID, targetUserID string) error {
	if requesterID == targetUserID {
		return fmt.Errorf("cannot kick yourself: %w", core_error.ErrInvalidArgument)
	}
	requester, err := s.repo.GetMember(ctx, roomID, requesterID)
	if err != nil {
		return fmt.Errorf("get requester: %w", err)
	}
	if !requester.IsAdmin() {
		return fmt.Errorf("only admin or owner can kick: %w", core_error.ErrUnauthorized)
	}
	target, err := s.repo.GetMember(ctx, roomID, targetUserID)
	if err != nil {
		return fmt.Errorf("get target member: %w", err)
	}
	if target.IsOwner() {
		return fmt.Errorf("cannot kick room owner: %w", core_error.ErrUnauthorized)
	}
	if target.Role == domain_models.MemberRoleAdmin && !requester.IsOwner() {
		return fmt.Errorf("only owner can kick admins: %w", core_error.ErrUnauthorized)
	}
	return s.repo.RemoveMember(ctx, roomID, targetUserID)
}

func (s *RoomsService) UpdateMemberRole(ctx context.Context, roomID, requesterID, targetUserID string, role domain_models.MemberRole) error {
	requester, err := s.repo.GetMember(ctx, roomID, requesterID)
	if err != nil {
		return fmt.Errorf("get requester: %w", err)
	}
	if !requester.IsOwner() {
		return fmt.Errorf("only owner can change roles: %w", core_error.ErrUnauthorized)
	}
	target, err := s.repo.GetMember(ctx, roomID, targetUserID)
	if err != nil {
		return fmt.Errorf("get target: %w", err)
	}
	if target.IsOwner() {
		return fmt.Errorf("cannot change owner role: %w", core_error.ErrInvalidArgument)
	}
	if role == domain_models.MemberRoleOwner {
		return fmt.Errorf("cannot assign owner role: %w", core_error.ErrInvalidArgument)
	}
	return s.repo.UpdateMemberRole(ctx, roomID, targetUserID, role)
}

func (s *RoomsService) GetMembers(ctx context.Context, roomID, userID string) ([]domain_models.RoomMember, error) {
	isMember, err := s.repo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, fmt.Errorf("access denied: %w", core_error.ErrUnauthorized)
	}
	return s.repo.GetMembers(ctx, roomID)
}
