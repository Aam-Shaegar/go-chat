package domain_models

import "time"

type RoomBan struct {
	RoomID        string     `json:"room_id"`
	UserID        string     `json:"user_id"`
	Username      string     `json:"username"`
	BannedBy      string     `json:"banned_by"`
	BannedByName  string     `json:"banned_by_name"`
	Reason        string     `json:"reason,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func NewRoomBan(roomID, userID, bannedBy, reason string, expiresAt *time.Time) RoomBan {
	return RoomBan{
		RoomID:    roomID,
		UserID:    userID,
		BannedBy:  bannedBy,
		Reason:    reason,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
}

func (b RoomBan) IsExpired() bool {
	if b.ExpiresAt == nil {
		return false
	}
	return b.ExpiresAt.Before(time.Now())
}
