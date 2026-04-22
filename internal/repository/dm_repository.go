package repository

import (
	"context"
	"fmt"
	"go-chat/internal/domain"
	"time"

	"github.com/jmoiron/sqlx"
)

type DMRepository struct {
	db *sqlx.DB
}

func NewDMRepository(db *sqlx.DB) *DMRepository {
	return &DMRepository{db: db}
}

func (r *DMRepository) Create(ctx context.Context, msg domain.DirectMessage) (domain.DirectMessage, error) {
	query := `
		INSERT INTO direct_messages (from_user_id, to_user_id, content)
		VALUES (:from_user_id, :to_user_id, :content)
		RETURNING id, from_user_id, to_user_id, content, edited_at, created_at
	`
	rows, err := r.db.NamedQueryContext(ctx, query, msg)
	if err != nil {
		return domain.DirectMessage{}, fmt.Errorf("create dm: %w", err)
	}
	defer rows.Close()

	var created domain.DirectMessage
	if rows.Next() {
		if err := rows.StructScan(&created); err != nil {
			return domain.DirectMessage{}, fmt.Errorf("scan dm: %w", err)
		}
	}
	return created, nil
}

func (r *DMRepository) GetHistory(ctx context.Context, userA, userB string, limit, offset int) ([]domain.DirectMessage, error) {
	var msgs []domain.DirectMessage
	query := `
		SELECT id, from_user_id, to_user_id, content, edited_at, created_at
		FROM direct_messages
		WHERE (from_user_id = $1 and to_user_id = $2)
			OR (from_user_id = $2 and to_user_id = $1)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	if err := r.db.SelectContext(ctx, &msgs, query, userA, userB, limit, offset); err != nil {
		return nil, fmt.Errorf("get dm history: %w", err)
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	if msgs == nil {
		msgs = []domain.DirectMessage{}
	}
	return msgs, nil
}

func (r *DMRepository) GetConversations(ctx context.Context, userID string) ([]domain.DirectMessage, error) {
	var msgs []domain.DirectMessage
	query := `
        SELECT DISTINCT ON (
            LEAST(from_user_id::text, to_user_id::text),
            GREATEST(from_user_id::text, to_user_id::text)
        )
        id, from_user_id, to_user_id, content, edited_at, created_at
        FROM direct_messages
        WHERE from_user_id = $1 OR to_user_id = $1
        ORDER BY
            LEAST(from_user_id::text, to_user_id::text),
            GREATEST(from_user_id::text, to_user_id::text),
            created_at DESC
    `
	if err := r.db.SelectContext(ctx, &msgs, query, userID); err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	if msgs == nil {
		msgs = []domain.DirectMessage{}
	}
	return msgs, nil
}

type DMUnread struct {
	FromUserID string `db:"from_user_id" json:"from_user_id"`
	Unread     int    `db:"unread"       json:"unread"`
}

func (r *DMRepository) MarkRead(ctx context.Context, userID, otherUserID string) error {
	query := `
        INSERT INTO dm_reads (user_id, other_user_id, last_read_at)
        VALUES ($1, $2, NOW())
        ON CONFLICT (user_id, other_user_id) DO UPDATE
        SET last_read_at = NOW()
    `
	if _, err := r.db.ExecContext(ctx, query, userID, otherUserID); err != nil {
		return fmt.Errorf("mark dm read: %w", err)
	}
	return nil
}

func (r *DMRepository) GetLastReadAt(ctx context.Context, userID, otherUserID string) (time.Time, error) {
	var t time.Time
	query := `SELECT last_read_at FROM dm_reads WHERE user_id = $1 AND other_user_id = $2`
	if err := r.db.GetContext(ctx, &t, query, userID, otherUserID); err != nil {
		return time.Time{}, nil // не читал вообще — нормально
	}
	return t, nil
}

func (r *DMRepository) GetAllUnreadCounts(ctx context.Context, userID string) ([]DMUnread, error) {
	var result []DMUnread
	query := `
        SELECT
            dm.from_user_id,
            COUNT(*) AS unread
        FROM direct_messages dm
        LEFT JOIN dm_reads dr
            ON dr.user_id = $1 AND dr.other_user_id = dm.from_user_id
        WHERE dm.to_user_id = $1
          AND (dr.last_read_at IS NULL OR dm.created_at > dr.last_read_at)
        GROUP BY dm.from_user_id
    `
	if err := r.db.SelectContext(ctx, &result, query, userID); err != nil {
		return nil, fmt.Errorf("get dm unread counts: %w", err)
	}
	return result, nil
}
