package domain_models

import "time"

type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

type RoomMember struct {
	RoomID     string      `json:"room_id"`
	UserID     string      `json:"user_id"`
	Username   string      `json:"username"`
	Role       MemberRole  `json:"role"`
	JoinedAt   time.Time   `json:"joined_at"`
	MutedUntil *time.Time  `json:"muted_until,omitempty"`
}

func NewRoomMember(roomID, userID, username string, role MemberRole, joinedAt time.Time, mutedUntil *time.Time) RoomMember {
	return RoomMember{
		RoomID:     roomID,
		UserID:     userID,
		Username:   username,
		Role:       role,
		JoinedAt:   joinedAt,
		MutedUntil: mutedUntil,
	}
}

func (m RoomMember) IsMuted() bool {
	if m.MutedUntil == nil {
		return false
	}
	return m.MutedUntil.After(time.Now())
}

func (m RoomMember) IsOwner() bool { return m.Role == MemberRoleOwner }
func (m RoomMember) IsAdmin() bool { return m.Role == MemberRoleAdmin || m.Role == MemberRoleOwner }
