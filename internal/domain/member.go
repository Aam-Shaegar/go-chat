package domain

import "time"

type Member struct {
	UserID   string    `db:"user_id" json:"user_id"`
	Username string    `db:"username" json:"username"`
	JoinedAt time.Time `db:"joined_at" json:"joined_at"`
}
