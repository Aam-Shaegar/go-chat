package domain_models

import "time"

type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

type RoomMember struct {
	RoomID   string     `json:"room_id"`
	UserID   string     `json:"user_id"`
	Username string     `json:"username"`
	Role     MemberRole `json:"role"`
	JoinedAt time.Time  `json:"joined_at"`
}

func NewRoomMember(roomID, userID, username string, role MemberRole, joinedAt time.Time) RoomMember {
	return RoomMember{
		RoomID:   roomID,
		UserID:   userID,
		Username: username,
		Role:     role,
		JoinedAt: joinedAt,
	}
}

func (m RoomMember) IsOwner() bool { return m.Role == MemberRoleOwner }
func (m RoomMember) IsAdmin() bool { return m.Role == MemberRoleAdmin || m.Role == MemberRoleOwner }
