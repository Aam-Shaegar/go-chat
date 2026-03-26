package domain

import "time"

type Room struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	IsPrivate   bool      `db:"is_private" json:"is_private"`
	OwnerID     string    `db:"owner_id" json:"owner_id"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type RoomMember struct {
	RoomID   string    `db:"room_id" json:"room_id"`
	UserD    string    `db:"user_id" json:"user_id"`
	JoinedAt time.Time `db:"joined_at" json:"joined_at"`
}
