package rooms_service

import (
	"context"
	"time"

	domain_models "go-chat/internal/core/domain/models"
)

type RoomsService struct {
	repo Repository
}

func NewRoomsService(repo Repository) *RoomsService {
	return &RoomsService{repo: repo}
}

type Repository interface {
	// Rooms
	CreateRoom(ctx context.Context, room domain_models.Room) (domain_models.Room, error)
	GetRoom(ctx context.Context, roomID string) (domain_models.Room, error)
	GetPublicRooms(ctx context.Context, limit, offset int) ([]domain_models.Room, error)
	GetUserRooms(ctx context.Context, userID string) ([]domain_models.Room, error)
	DeleteRoom(ctx context.Context, roomID, ownerID string) error

	// Members
	IsMember(ctx context.Context, roomID, userID string) (bool, error)
	GetMember(ctx context.Context, roomID, userID string) (domain_models.RoomMember, error)
	GetMembers(ctx context.Context, roomID string) ([]domain_models.RoomMember, error)
	AddMember(ctx context.Context, roomID, userID string, role domain_models.MemberRole) error
	RemoveMember(ctx context.Context, roomID, userID string) error
	UpdateMemberRole(ctx context.Context, roomID, userID string, role domain_models.MemberRole) error

	// Invites
	CreateInvite(ctx context.Context, invite domain_models.RoomInvite) (domain_models.RoomInvite, error)
	GetInviteByToken(ctx context.Context, token string) (domain_models.RoomInvite, error)
	TryIncrementInviteUses(ctx context.Context, token string) error
	DeactivateInvite(ctx context.Context, token, userID string) error
	GetRoomInvites(ctx context.Context, roomID string) ([]domain_models.RoomInvite, error)
}

const defaultInviteTTL = 7 * 24 * time.Hour
