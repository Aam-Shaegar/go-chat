package repository

import (
	"context"
	"fmt"
	"go-chat/internal/domain"

	"github.com/jmoiron/sqlx"
)

type MessageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, msg domain.Message) (domain.Message, error) {
	query := `
		INSERT INTO messages (room_id, user_id, content)
		VALUES (:room_id, :user_id, :content)
		RETURNING id, room_id, user_id, content, edited_at, created_at
	`
	rows, err := r.db.NamedQueryContext(ctx, query, msg)
	if err != nil {
		return domain.Message{}, fmt.Errorf("create message: %w", err)
	}
	defer rows.Close()

	var created domain.Message
	if rows.Next() {
		if err := rows.StructScan(&created); err != nil {
			return domain.Message{}, fmt.Errorf("scan message: %w", err)
		}
	}
	return created, nil
}

func (r *MessageRepository) ListByRoomID(ctx context.Context, roomID string, limit int) ([]domain.Message, error) {
	var messages []domain.Message
	query := `
		SELECT id, room_id, user_id, content, edited_at, created_at
		FROM messages
		WHERE room_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	if err := r.db.SelectContext(ctx, &messages, query, roomID, limit); err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	if messages == nil {
		messages = []domain.Message{}
	}
	return messages, nil
}
