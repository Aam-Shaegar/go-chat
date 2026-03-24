package domain

import "time"

type Message struct {
	ID        string     `db:"id"`
	RoomID    string     `db:"room_id"`
	UserID    string     `db:"user_id"`
	Content   string     `db:"content"`
	EditedAt  *time.Time `db:"edited_at"`
	CreatedAt time.Time  `db:"created_at"`
}
