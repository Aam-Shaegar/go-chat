package domain

import "time"

type Room struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	IsPrivate   bool      `db:"is_private"`
	OwnerID     string    `db:"owner_id"`
	CreatedAt   time.Time `db:"created_at"`
}

type RoomMember struct {
	RoomID   string    `db:"room_id"`
	UserD    string    `db:"user_id"`
	JoinedAt time.Time `db:"joined_at"`
}
