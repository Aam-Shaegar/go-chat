package ws_repository_postgres

import (
	"context"
	"encoding/json"
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
		WITH inserted AS (
			INSERT INTO gochat.messages (room_id, user_id, reply_to_id, content, is_encrypted, created_at, updated_at)
			SELECT $1, $2, $3, $4, $5, $6, $7
			WHERE EXISTS (
				SELECT 1 FROM gochat.room_members
				WHERE room_id=$1 AND user_id=$2
			)
			RETURNING id, room_id, user_id, reply_to_id, content, is_encrypted, created_at, updated_at, deleted_at
		)
		SELECT i.id, i.room_id, i.user_id, u.username, i.reply_to_id, i.content, i.is_encrypted,
		       i.created_at, i.updated_at, i.deleted_at
		FROM inserted i
		JOIN gochat.users u ON u.id = i.user_id;
	`
	row := r.pool.QueryRow(ctx, query,
		msg.RoomID, msg.UserID, msg.ReplyToID,
		msg.Content, msg.IsEncrypted,
		msg.CreatedAt, msg.UpdatedAt,
	)
	return scanMessage(row)
}

func (r *WSRepository) EditMessage(ctx context.Context, messageID, roomID, userID, content string, updatedAt time.Time) (domain_models.Message, error) {
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
		return domain_models.Message{}, err
	}
	return msg, nil
}

func (r *WSRepository) DeleteMessage(ctx context.Context, messageID, roomID, userID string, deletedAt time.Time) (domain_models.Message, error) {
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

func (r *WSRepository) AddReaction(ctx context.Context, roomID string, reaction domain_models.RawMessageReaction) (domain_models.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		WITH target AS (
			SELECT m.id, m.room_id, m.user_id, m.reply_to_id, m.content, m.is_encrypted,
			       m.created_at, m.updated_at, m.deleted_at
			FROM gochat.messages m
			WHERE m.id=$1 AND m.room_id=$2 AND m.deleted_at IS NULL
		),
		authorized AS (
			SELECT t.*
			FROM target t
			WHERE EXISTS (
				SELECT 1 FROM gochat.room_members
				WHERE room_id=$2 AND user_id=$3
			)
		),
		upserted AS (
			INSERT INTO gochat.message_reactions (message_id, user_id, emoji, created_at)
			SELECT id, $3, $4, $5 FROM authorized
			ON CONFLICT (message_id, user_id) DO UPDATE SET
				emoji = EXCLUDED.emoji,
				created_at = EXCLUDED.created_at
			RETURNING message_id
		),
		reactions_agg AS (
			SELECT 
				json_agg(
					json_build_object(
						'emoji', mr.emoji,
						'count', mr.cnt,
						'users', mr.user_ids,
						'is_reacted_by_me', mr.user_ids @> ARRAY[$3::uuid]
					)
				) as reactions_json
			FROM (
				SELECT emoji, COUNT(*) as cnt, ARRAY_AGG(user_id) as user_ids
				FROM gochat.message_reactions
				WHERE message_id = $1
				GROUP BY emoji
			) mr
		)
		SELECT a.id, a.room_id, a.user_id, u.username, a.reply_to_id, a.content, a.is_encrypted,
		       a.created_at, a.updated_at, a.deleted_at,
		       COALESCE(ra.reactions_json, '[]'::json)
		FROM authorized a
		JOIN gochat.users u ON u.id = a.user_id
		LEFT JOIN upserted i ON i.message_id = a.id
		CROSS JOIN reactions_agg ra;
	`
	row := r.pool.QueryRow(ctx, query, reaction.MessageID, roomID, reaction.UserID, reaction.Emoji, reaction.CreatedAt)
	msg, err := scanMessageWithReactions(row)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain_models.Message{}, fmt.Errorf("message not found or access denied: %w", core_postgres_pool.ErrNoRows)
		}
		return domain_models.Message{}, fmt.Errorf("add reaction: %w", err)
	}
	return msg, nil
}

func (r *WSRepository) RemoveReaction(ctx context.Context, messageID, roomID, userID string) (domain_models.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		WITH authorized AS (
			SELECT m.id, m.room_id, m.user_id, m.reply_to_id, m.content, m.is_encrypted,
			       m.created_at, m.updated_at, m.deleted_at
			FROM gochat.messages m
			WHERE m.id=$1 AND m.room_id=$2 AND m.deleted_at IS NULL
			  AND EXISTS (
				SELECT 1 FROM gochat.room_members
				WHERE room_id=$2 AND user_id=$3
			  )
		),
		deleted AS (
			DELETE FROM gochat.message_reactions
			WHERE message_id=$1 AND user_id=$3
			RETURNING message_id
		),
		reactions_agg AS (
			SELECT 
				json_agg(
					json_build_object(
						'emoji', mr.emoji,
						'count', mr.cnt,
						'users', mr.user_ids,
						'is_reacted_by_me', mr.user_ids @> ARRAY[$3::uuid]
					)
				) as reactions_json
			FROM (
				SELECT emoji, COUNT(*) as cnt, ARRAY_AGG(user_id) as user_ids
				FROM gochat.message_reactions
				WHERE message_id = $1
				GROUP BY emoji
			) mr
		)
		SELECT a.id, a.room_id, a.user_id, u.username, a.reply_to_id, a.content, a.is_encrypted,
		       a.created_at, a.updated_at, a.deleted_at,
		       COALESCE(ra.reactions_json, '[]'::json)
		FROM authorized a
		JOIN gochat.users u ON u.id = a.user_id
		LEFT JOIN deleted d ON d.message_id = a.id
		CROSS JOIN reactions_agg ra;
	`
	row := r.pool.QueryRow(ctx, query, messageID, roomID, userID)
	msg, err := scanMessageWithReactions(row)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain_models.Message{}, fmt.Errorf("message not found or access denied: %w", core_postgres_pool.ErrNoRows)
		}
		return domain_models.Message{}, fmt.Errorf("remove reaction: %w", err)
	}
	return msg, nil
}

func (r *WSRepository) GetRoomMemberIDs(ctx context.Context, roomID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `SELECT user_id FROM gochat.room_members WHERE room_id=$1;`
	rows, err := r.pool.Query(ctx, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("query room members: %w", err)
	}
	defer rows.Close()

	userIDs := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan room member: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return userIDs, nil
}

func (r *WSRepository) GetMember(ctx context.Context, roomID, userID string) (domain_models.RoomMember, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT rm.room_id, rm.user_id, u.username, rm.role, rm.joined_at, rm.muted_until
		FROM gochat.room_members rm
		JOIN gochat.users u ON u.id=rm.user_id
		WHERE rm.room_id=$1 AND rm.user_id=$2;
	`
	row := r.pool.QueryRow(ctx, query, roomID, userID)
	var m struct {
		RoomID     string
		UserID     string
		Username   string
		Role       string
		JoinedAt   time.Time
		MutedUntil *time.Time
	}
	err := row.Scan(&m.RoomID, &m.UserID, &m.Username, &m.Role, &m.JoinedAt, &m.MutedUntil)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain_models.RoomMember{}, fmt.Errorf("member not found: %w", core_postgres_pool.ErrNoRows)
		}
		return domain_models.RoomMember{}, fmt.Errorf("scan member: %w", err)
	}
	return domain_models.NewRoomMember(
		m.RoomID, m.UserID, m.Username,
		domain_models.MemberRole(m.Role),
		m.JoinedAt, m.MutedUntil,
	), nil
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

func scanMessageWithReactions(row core_postgres_pool.Row) (domain_models.Message, error) {
	var m domain_models.Message
	var reactionsJSON string
	err := row.Scan(
		&m.ID, &m.RoomID, &m.UserID, &m.Username, &m.ReplyToID,
		&m.Content, &m.IsEncrypted,
		&m.CreatedAt, &m.UpdatedAt, &m.DeletedAt,
		&reactionsJSON,
	)
	if err != nil {
		return domain_models.Message{}, fmt.Errorf("scan message: %w", err)
	}
	if reactionsJSON != "" && reactionsJSON != "[]" {
		var rawReactions []struct {
			Emoji          string   `json:"emoji"`
			Count          int      `json:"count"`
			Users          []string `json:"users"`
			IsReactedByMe  bool     `json:"is_reacted_by_me"`
		}
		if err := json.Unmarshal([]byte(reactionsJSON), &rawReactions); err == nil {
			m.Reactions = make([]domain_models.MessageReaction, len(rawReactions))
			for i, r := range rawReactions {
				m.Reactions[i] = domain_models.MessageReaction{
					Emoji:          r.Emoji,
					Count:          r.Count,
					Users:          r.Users,
					IsReactedByMe:  r.IsReactedByMe,
				}
			}
		}
	}
	return m, nil
}
