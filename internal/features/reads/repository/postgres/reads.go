package reads_repository_postgres

import (
	"context"
	"fmt"
	"time"
)

// MarkRead сохраняет или обновляет last_read_at для пользователя в комнате.
// UPSERT — если записи нет, создаёт; если есть, обновляет только если новое время позже.
func (r *ReadsRepository) MarkRead(ctx context.Context, roomID, userID string, lastReadAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		INSERT INTO gochat.room_reads (room_id, user_id, last_read_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (room_id, user_id)
		DO UPDATE SET last_read_at = EXCLUDED.last_read_at
		WHERE gochat.room_reads.last_read_at < EXCLUDED.last_read_at;
	`
	_, err := r.pool.Exec(ctx, query, roomID, userID, lastReadAt)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	return nil
}

// GetUnreadCounts возвращает количество непрочитанных сообщений
// по всем комнатам где состоит пользователь.
func (r *ReadsRepository) GetUnreadCounts(ctx context.Context, userID string) (map[string]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	// LEFT JOIN с room_reads — если записи нет, считаем все сообщения непрочитанными
	query := `
		SELECT
			m.room_id,
			COUNT(*) AS unread_count
		FROM gochat.messages m
		JOIN gochat.room_members rm ON rm.room_id = m.room_id AND rm.user_id = $1
		LEFT JOIN gochat.room_reads rr ON rr.room_id = m.room_id AND rr.user_id = $1
		WHERE
			m.deleted_at IS NULL
			AND m.user_id != $1
			AND (rr.last_read_at IS NULL OR m.created_at > rr.last_read_at)
		GROUP BY m.room_id;
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query unread counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var roomID string
		var count int64
		if err := rows.Scan(&roomID, &count); err != nil {
			return nil, fmt.Errorf("scan unread count: %w", err)
		}
		counts[roomID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return counts, nil
}

// GetUnreadCount возвращает количество непрочитанных для одной комнаты.
func (r *ReadsRepository) GetUnreadCount(ctx context.Context, roomID, userID string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT COUNT(*)
		FROM gochat.messages m
		LEFT JOIN gochat.room_reads rr ON rr.room_id = m.room_id AND rr.user_id = $2
		WHERE
			m.room_id = $1
			AND m.deleted_at IS NULL
			AND m.user_id != $2
			AND (rr.last_read_at IS NULL OR m.created_at > rr.last_read_at);
	`
	row := r.pool.QueryRow(ctx, query, roomID, userID)
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("scan unread count: %w", err)
	}
	return count, nil
}
