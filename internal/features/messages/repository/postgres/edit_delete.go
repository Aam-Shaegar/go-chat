package messages_repository_postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
)

func (r *MessagesRepository) EditMessage(ctx context.Context, roomID, userID, messageID, content string, updatedAt time.Time) (domain_models.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		WITH updated AS (
			UPDATE gochat.messages
			SET content=$1, updated_at=$2
			WHERE id=$3 AND room_id=$4 AND user_id=$5 AND deleted_at IS NULL
			RETURNING id, room_id, user_id, reply_to_id, content, is_encrypted, created_at, updated_at, deleted_at
		)
		SELECT m.id, m.room_id, m.user_id, u.username, m.reply_to_id, m.content, m.is_encrypted,
		       m.created_at, m.updated_at, m.deleted_at
		FROM updated m
		JOIN gochat.users u ON u.id = m.user_id;
	`
	row := r.pool.QueryRow(ctx, query, content, updatedAt, messageID, roomID, userID)
	msg, err := scanMessage(row)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain_models.Message{}, fmt.Errorf("message not found or not owned by user: %w", core_postgres_pool.ErrNoRows)
		}
		return domain_models.Message{}, fmt.Errorf("edit message: %w", err)
	}
	return msg, nil
}

func (r *MessagesRepository) DeleteMessage(ctx context.Context, roomID, userID, messageID string, deletedAt time.Time) (domain_models.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		WITH updated AS (
			UPDATE gochat.messages
			SET deleted_at=$1
			WHERE id=$2 AND room_id=$3 AND user_id=$4 AND deleted_at IS NULL
			RETURNING id, room_id, user_id, reply_to_id, content, is_encrypted, created_at, updated_at, deleted_at
		)
		SELECT m.id, m.room_id, m.user_id, u.username, m.reply_to_id, m.content, m.is_encrypted,
		       m.created_at, m.updated_at, m.deleted_at
		FROM updated m
		JOIN gochat.users u ON u.id = m.user_id;
	`
	row := r.pool.QueryRow(ctx, query, deletedAt, messageID, roomID, userID)
	msg, err := scanMessage(row)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain_models.Message{}, fmt.Errorf("message not found or not owned by user: %w", core_postgres_pool.ErrNoRows)
		}
		return domain_models.Message{}, fmt.Errorf("delete message: %w", err)
	}
	return msg, nil
}

func scanMessage(row core_postgres_pool.Row) (domain_models.Message, error) {
	var m domain_models.Message
	err := row.Scan(
		&m.ID, &m.RoomID, &m.UserID, &m.Username, &m.ReplyToID,
		&m.Content, &m.IsEncrypted,
		&m.CreatedAt, &m.UpdatedAt, &m.DeletedAt,
	)
	if err != nil {
		return domain_models.Message{}, fmt.Errorf("scan message: %w", err)
	}
	return m, nil
}