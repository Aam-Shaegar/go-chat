package ws_repository_postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
)

type WSRepository struct {
	pool core_postgres_pool.Pool
}

func NewWSRepository(pool core_postgres_pool.Pool) *WSRepository {
	return &WSRepository{pool: pool}
}

func (r *WSRepository) SaveMessage(ctx context.Context, msg domain_models.Message) (domain_models.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		INSERT INTO gochat.messages (room_id, user_id, reply_to_id, content, is_encrypted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, room_id, user_id, reply_to_id, content, is_encrypted, created_at, updated_at, deleted_at;
	`
	row := r.pool.QueryRow(ctx, query,
		msg.RoomID, msg.UserID, msg.ReplyToID,
		msg.Content, msg.IsEncrypted,
		msg.CreatedAt, msg.UpdatedAt,
	)
	return scanMessage(row)
}

func (r *WSRepository) EditMessage(ctx context.Context, messageID, userID, content string, updatedAt time.Time) (domain_models.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		UPDATE gochat.messages
		SET content=$1, updated_at=$2
		WHERE id=$3 AND user_id=$4 AND deleted_at IS NULL
		RETURNING id, room_id, user_id, reply_to_id, content, is_encrypted, created_at, updated_at, deleted_at;
	`
	row := r.pool.QueryRow(ctx, query, content, updatedAt, messageID, userID)
	msg, err := scanMessage(row)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain_models.Message{}, fmt.Errorf("message not found or not owned by user: %w", core_postgres_pool.ErrNoRows)
		}
		return domain_models.Message{}, err
	}
	return msg, nil
}

func (r *WSRepository) DeleteMessage(ctx context.Context, messageID, userID string, deletedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		UPDATE gochat.messages
		SET deleted_at=$1
		WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL;
	`
	tag, err := r.pool.Exec(ctx, query, deletedAt, messageID, userID)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("message not found or not owned by user: %w", core_postgres_pool.ErrNoRows)
	}
	return nil
}

func (r *WSRepository) AddReaction(ctx context.Context, reaction domain_models.MessageReaction) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		INSERT INTO gochat.message_reactions (message_id, user_id, emoji, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING;
	`
	_, err := r.pool.Exec(ctx, query, reaction.MessageID, reaction.UserID, reaction.Emoji, reaction.CreatedAt)
	if err != nil {
		return fmt.Errorf("add reaction: %w", err)
	}
	return nil
}

func (r *WSRepository) RemoveReaction(ctx context.Context, messageID, userID, emoji string) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		DELETE FROM gochat.message_reactions
		WHERE message_id=$1 AND user_id=$2 AND emoji=$3;
	`
	_, err := r.pool.Exec(ctx, query, messageID, userID, emoji)
	if err != nil {
		return fmt.Errorf("remove reaction: %w", err)
	}
	return nil
}

func scanMessage(row core_postgres_pool.Row) (domain_models.Message, error) {
	var m domain_models.Message
	err := row.Scan(
		&m.ID, &m.RoomID, &m.UserID, &m.ReplyToID,
		&m.Content, &m.IsEncrypted,
		&m.CreatedAt, &m.UpdatedAt, &m.DeletedAt,
	)
	if err != nil {
		return domain_models.Message{}, fmt.Errorf("scan message: %w", err)
	}
	return m, nil
}
