package rooms_repository_postgres

import (
	domain_models "go-chat/internal/core/domain/models"
	"time"
)

type roomModel struct {
	ID          string
	Name        string
	Description string
	IsPrivate   bool
	IsDM        bool
	OwnerID     string
	CreatedAt   time.Time
}

type memberModel struct {
	RoomID   string
	UserID   string
	Username string
	Role     string
	JoinedAt time.Time
}

type inviteModel struct {
	ID        string
	RoomID    string
	Token     string
	CreatedBy string
	MaxUses   int
	Uses      int
	IsActive  bool
	ExpiresAt *time.Time
	CreatedAt time.Time
}

func roomToDomain(m roomModel) domain_models.Room {
	return domain_models.NewRoom(
		m.ID, m.Name, m.Description,
		m.IsPrivate, m.IsDM, m.OwnerID, m.CreatedAt,
	)
}

func memberToDomain(m memberModel) domain_models.RoomMember {
	return domain_models.NewRoomMember(
		m.RoomID, m.UserID, m.Username,
		domain_models.MemberRole(m.Role),
		m.JoinedAt,
	)
}

func inviteToDomain(m inviteModel) domain_models.RoomInvite {
	return domain_models.NewRoomInvite(
		m.ID, m.RoomID, m.Token, m.CreatedBy,
		m.MaxUses, m.ExpiresAt, m.CreatedAt,
	)
}
