package domain

import "time"

type RoomInvite struct {
	ID        string    `db:"id" json:"id"`
	RoomID    string    `db:"room_id" json:"room_id"`
	CreatedBy string    `db:"created_by" json:"created_by"`
	UsedBy    *string   `db:"used_by" json:"used_by,omitempty"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
