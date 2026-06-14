package domain_models

import "time"

type RoomInvite struct {
	ID        string     `json:"id"`
	RoomID    string     `json:"room_id"`
	Token     string     `json:"token"`
	CreatedBy string     `json:"created_by"`
	MaxUses   int        `json:"max_uses"`
	Uses      int        `json:"uses"`
	IsActive  bool       `json:"is_active"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func NewRoomInvite(id, roomID, token, createdBy string, maxUses int, expiresAt *time.Time, createdAt time.Time) RoomInvite {
	return RoomInvite{
		ID:        id,
		RoomID:    roomID,
		Token:     token,
		CreatedBy: createdBy,
		MaxUses:   maxUses,
		Uses:      0,
		IsActive:  true,
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
	}
}

func (i RoomInvite) IsExpired() bool {
	if i.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*i.ExpiresAt)
}

func (i RoomInvite) IsExhausted() bool {
	return i.MaxUses > 0 && i.Uses >= i.MaxUses
}

func (i RoomInvite) CanBeUsed() bool {
	return i.IsActive && !i.IsExpired() && !i.IsExhausted()
}
