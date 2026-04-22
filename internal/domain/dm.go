package domain

import "time"

type DirectMessage struct {
	ID         string     `db:"id" json:"id"`
	FromUserID string     `db:"from_user_id" json:"from_user_id"`
	ToUserID   string     `db:"to_user_id" json:"to_user_id"`
	Content    string     `db:"content" json:"content"`
	EditedAt   *time.Time `db:"edited_at" json:"edited_at"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

type DMConversation struct {
	User        User          `json:"user"`
	LastMessage DirectMessage `json:"last_message"`
	Unread      int           `json:"unread"`
}
