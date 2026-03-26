package domain

import "time"

type Message struct {
	ID        string     `db:"id" json:"id"`
	RoomID    string     `db:"room_id" json:"room_id"`
	UserID    string     `db:"user_id" json:"user_id"`
	Content   string     `db:"content" json:"content"`
	EditedAt  *time.Time `db:"edited_at" json:"edited_at,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}
