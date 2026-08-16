package messages_repository_postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
)

const defaultLimit = 50

type messageRow struct {
	ID            string
	RoomID        string
	UserID        string
	Username      string
	ReplyToID     sql.NullString
	Content       string
	IsEncrypted   bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     sql.NullTime
	ReactionsJSON sql.NullString
}

// GetMessages возвращает сообщения комнаты с пагинацией по курсору.
// before — курсор (created_at + id последнего сообщения), nil — самые свежие.
// Возвращает сообщения от старых к новым.
func (r *MessagesRepository) GetMessages(ctx context.Context, roomID string, before *domain_models.MessageCursor, limit int) ([]domain_models.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = defaultLimit
	}

	var rows core_postgres_pool.Rows
	var err error

	baseQuery := `
		SELECT 
			m.id, m.room_id, m.user_id, u.username, m.reply_to_id, m.content, m.is_encrypted,
			m.created_at, m.updated_at, m.deleted_at,
			COALESCE(
				json_agg(
					json_build_object(
						'emoji', mr.emoji,
						'count', mr.cnt,
						'users', mr.user_ids,
						'is_reacted_by_me', false
					)
				) FILTER (WHERE mr.emoji IS NOT NULL),
				'[]'
			) as reactions_json
		FROM gochat.messages m
		JOIN gochat.users u ON u.id = m.user_id
		LEFT JOIN LATERAL (
			SELECT 
				emoji,
				COUNT(*) as cnt,
				ARRAY_AGG(user_id) as user_ids
			FROM gochat.message_reactions
			WHERE message_id = m.id
			GROUP BY emoji
		) mr ON true
		WHERE m.room_id=$1 AND m.deleted_at IS NULL
	`

	if before == nil {
		query := baseQuery + `
			GROUP BY m.id, u.username
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT $2;
		`
		rows, err = r.pool.Query(ctx, query, roomID, limit)
	} else if before.ID == "" {
		query := baseQuery + `
			AND m.created_at < $2
			GROUP BY m.id, u.username
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT $3;
		`
		rows, err = r.pool.Query(ctx, query, roomID, before.CreatedAt, limit)
	} else {
		query := baseQuery + `
			AND (m.created_at < $2 OR (m.created_at = $2 AND m.id < $3::uuid))
			GROUP BY m.id, u.username
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT $4;
		`
		rows, err = r.pool.Query(ctx, query, roomID, before.CreatedAt, before.ID, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var messages []domain_models.Message
	for rows.Next() {
		var row messageRow
		if err := rows.Scan(
			&row.ID, &row.RoomID, &row.UserID, &row.Username, &row.ReplyToID,
			&row.Content, &row.IsEncrypted,
			&row.CreatedAt, &row.UpdatedAt, &row.DeletedAt,
			&row.ReactionsJSON,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}

		var replyToID *string
		if row.ReplyToID.Valid {
			replyToID = &row.ReplyToID.String
		}
		var deletedAt *time.Time
		if row.DeletedAt.Valid {
			deletedAt = &row.DeletedAt.Time
		}

		var reactions []domain_models.MessageReaction
		if row.ReactionsJSON.Valid && row.ReactionsJSON.String != "" && row.ReactionsJSON.String != "[]" {
			var rawReactions []struct {
				Emoji       string   `json:"emoji"`
				Count       int      `json:"count"`
				Users       []string `json:"users"`
				IsReactedByMe bool   `json:"is_reacted_by_me"`
			}
			if err := json.Unmarshal([]byte(row.ReactionsJSON.String), &rawReactions); err == nil {
				reactions = make([]domain_models.MessageReaction, len(rawReactions))
				for i, r := range rawReactions {
					reactions[i] = domain_models.MessageReaction{
						Emoji:          r.Emoji,
						Count:          r.Count,
						Users:          r.Users,
						IsReactedByMe:  r.IsReactedByMe,
					}
				}
			}
		}

		messages = append(messages, domain_models.Message{
			ID:           row.ID,
			RoomID:       row.RoomID,
			UserID:       row.UserID,
			Username:     row.Username,
			ReplyToID:    replyToID,
			Content:      row.Content,
			IsEncrypted:  row.IsEncrypted,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
			DeletedAt:    deletedAt,
			Reactions:    reactions,
		})
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
