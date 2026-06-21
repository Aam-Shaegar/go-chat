package messages_repository_postgres

import (
	"context"
	"fmt"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
)

const defaultLimit = 50

// GetMessages возвращает сообщения комнаты с пагинацией по курсору.
// before — курсор (created_at последнего сообщения), nil — самые свежие.
// Возвращает сообщения от старых к новым.
func (r *MessagesRepository) GetMessages(ctx context.Context, roomID string, before *time.Time, limit int) ([]domain_models.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = defaultLimit
	}

	var rows core_postgres_pool.Rows
	var err error

	if before == nil {
		query := `
			SELECT id, room_id, user_id, reply_to_id, content, is_encrypted, created_at, updated_at, deleted_at
			FROM gochat.messages
			WHERE room_id=$1 AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT $2;
		`
		rows, err = r.pool.Query(ctx, query, roomID, limit)
	} else {
		query := `
			SELECT id, room_id, user_id, reply_to_id, content, is_encrypted, created_at, updated_at, deleted_at
			FROM gochat.messages
			WHERE room_id=$1 AND deleted_at IS NULL AND created_at < $2
			ORDER BY created_at DESC
			LIMIT $3;
		`
		rows, err = r.pool.Query(ctx, query, roomID, before, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var messages []domain_models.Message
	for rows.Next() {
		var m domain_models.Message
		if err := rows.Scan(
			&m.ID, &m.RoomID, &m.UserID, &m.ReplyToID,
			&m.Content, &m.IsEncrypted,
			&m.CreatedAt, &m.UpdatedAt, &m.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	// БД вернула DESC (новые первыми), разворачиваем для клиента — от старых к новым
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
