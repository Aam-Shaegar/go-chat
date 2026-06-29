package dm_repository_postgres

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"

	"github.com/google/uuid"
)

// FindDM ищет существующий DM между двумя пользователями.
// Возвращает ErrNoRows, если не найден.
func (r *DMRepository) FindDM(ctx context.Context, userID1, userID2 string) (domain_models.Room, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT r.id, r.name, r.description, r.is_private, r.is_dm, r.owner_id, r.created_at,
		       COALESCE(MAX(m.created_at), r.created_at) AS last_message_at
		FROM gochat.rooms r
		JOIN gochat.direct_messages dm ON dm.room_id = r.id
		LEFT JOIN gochat.messages m ON m.room_id = r.id AND m.deleted_at IS NULL
		WHERE r.is_dm = true
		  AND dm.user_id_low = $1
		  AND dm.user_id_high = $2
		GROUP BY r.id
		LIMIT 1;
	`
	low, high := orderedPair(userID1, userID2)
	row := r.pool.QueryRow(ctx, query, low, high)
	var m roomModel
	err := row.Scan(&m.ID, &m.Name, &m.Description, &m.IsPrivate, &m.IsDM, &m.OwnerID, &m.CreatedAt, &m.LastMessageAt)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain_models.Room{}, core_postgres_pool.ErrNoRows
		}
		return domain_models.Room{}, fmt.Errorf("scan dm room: %w", err)
	}
	return roomToDomain(m), nil
}

// CreateDM создаёт DM-комнату и добавляет двух участников в одной транзакции.
func (r *DMRepository) CreateDM(ctx context.Context, userID1, userID2 string) (domain_models.Room, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain_models.Room{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Для DM имя не используется, идентификация идёт по участникам
	roomQuery := `
		INSERT INTO gochat.rooms (id, name, description, is_private, is_dm, owner_id, created_at)
		VALUES ($1, '', '', true, true, $2, $3)
		RETURNING id, name, description, is_private, is_dm, owner_id, created_at, created_at;
	`
	roomID := uuid.New().String()
	now := time.Now()
	row := tx.QueryRow(ctx, roomQuery, roomID, userID1, now)
	var m roomModel
	if err := row.Scan(&m.ID, &m.Name, &m.Description, &m.IsPrivate, &m.IsDM, &m.OwnerID, &m.CreatedAt, &m.LastMessageAt); err != nil {
		return domain_models.Room{}, fmt.Errorf("insert dm room: %w", err)
	}

	memberQuery := `
		INSERT INTO gochat.room_members (room_id, user_id, role)
		VALUES ($1, $2, 'member'), ($1, $3, 'member');
	`
	if _, err := tx.Exec(ctx, memberQuery, m.ID, userID1, userID2); err != nil {
		return domain_models.Room{}, fmt.Errorf("insert dm members: %w", err)
	}

	low, high := orderedPair(userID1, userID2)
	pairQuery := `
		INSERT INTO gochat.direct_messages (room_id, user_id_low, user_id_high)
		VALUES ($1, $2, $3);
	`
	if _, err := tx.Exec(ctx, pairQuery, m.ID, low, high); err != nil {
		return domain_models.Room{}, fmt.Errorf("insert dm pair: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain_models.Room{}, fmt.Errorf("commit tx: %w", err)
	}

	return roomToDomain(m), nil
}

// GetUserDMs возвращает все DM-комнаты пользователя.
func (r *DMRepository) GetUserDMs(ctx context.Context, userID string) ([]domain_models.Room, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT r.id, r.name, r.description, r.is_private, r.is_dm, r.owner_id, r.created_at,
		       COALESCE(MAX(m.created_at), r.created_at) AS last_message_at
		FROM gochat.rooms r
		JOIN gochat.room_members rm ON rm.room_id = r.id AND rm.user_id = $1
		LEFT JOIN gochat.messages m ON m.room_id = r.id AND m.deleted_at IS NULL
		WHERE r.is_dm = true
		GROUP BY r.id
		ORDER BY COALESCE(MAX(m.created_at), r.created_at) DESC;
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user dms: %w", err)
	}
	defer rows.Close()

	var rooms []domain_models.Room
	for rows.Next() {
		var m roomModel
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.IsPrivate, &m.IsDM, &m.OwnerID, &m.CreatedAt, &m.LastMessageAt); err != nil {
			return nil, fmt.Errorf("scan dm room: %w", err)
		}
		rooms = append(rooms, roomToDomain(m))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return rooms, nil
}

func orderedPair(userID1, userID2 string) (string, string) {
	values := []string{userID1, userID2}
	sort.Strings(values)
	return values[0], values[1]
}
