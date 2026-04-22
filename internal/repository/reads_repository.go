package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type ReadsRepository struct {
	db *sqlx.DB
}

func NewReadsRepository(db *sqlx.DB) *ReadsRepository {
	return &ReadsRepository{db: db}
}

func (r *ReadsRepository) Upsert(ctx context.Context, roomID, userID, messageID string) error {
	query := `INSERT INTO room_reads (room_id, user_id, last_read_message_id, updated_at)
			  VALUES ($1, $2, $3, NOW())
			  ON CONFLICT (room_id, user_id) DO UPDATE
			  SET last_read_message_id =  $3, updated_at = NOW()`
	if _, err := r.db.ExecContext(ctx, query, roomID, userID, messageID); err != nil {
		return fmt.Errorf("upsert read: %w", err)
	}
	return nil
}

func (r *ReadsRepository) Get(ctx context.Context, roomID, userID string) (string, error) {
	var msgID string
	query := `SELECT COALESCE (last_read_message_id::text, '')
		FROM room_reads
		WHERE room_id = $1 AND user_id = $2`
	if err := r.db.GetContext(ctx, &msgID, query, roomID, userID); err != nil {
		return "", fmt.Errorf("get read: %w", err)
	}
	return msgID, nil
}

type RoomUnread struct {
	RoomID string `db:"room_id" json:"room_id"`
	Uread  int    `db:"unread" json:"unread"`
}

func (r *ReadsRepository) GetUnreadCounts(ctx context.Context, userID string) ([]RoomUnread, error) {
	var result []RoomUnread
	query := `
        SELECT
            rm.room_id,
            COUNT(m.id) AS unread
        FROM room_members rm
        LEFT JOIN room_reads rr
            ON rr.room_id = rm.room_id AND rr.user_id = rm.user_id
        LEFT JOIN messages m
            ON m.room_id = rm.room_id
            AND m.user_id != $1
            AND (rr.last_read_message_id IS NULL OR m.created_at > (
                SELECT created_at FROM messages WHERE id = rr.last_read_message_id
            ))
        WHERE rm.user_id = $1
        GROUP BY rm.room_id
    `
	if err := r.db.SelectContext(ctx, &result, query, userID); err != nil {
		return nil, fmt.Errorf("get unread counts: %w", err)
	}
	return result, nil
}
