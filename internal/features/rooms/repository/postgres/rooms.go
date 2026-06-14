package rooms_repository_postgres

import (
	"context"
	"errors"
	"fmt"

	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
)

// CreateRoom создаёт комнату и добавляет владельца атомарно в одной транзакции
func (r *RoomsRepository) CreateRoom(ctx context.Context, room domain_models.Room) (domain_models.Room, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain_models.Room{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	roomQuery := `
		INSERT INTO gochat.rooms (name, description, is_private, is_dm, owner_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, description, is_private, is_dm, owner_id, created_at;
	`
	row := tx.QueryRow(ctx, roomQuery,
		room.Name, room.Description, room.IsPrivate, room.IsDM, room.OwnerID, room.CreatedAt,
	)
	created, err := scanRoom(row)
	if err != nil {
		return domain_models.Room{}, fmt.Errorf("insert room: %w", err)
	}

	memberQuery := `
		INSERT INTO gochat.room_members (room_id, user_id, role)
		VALUES ($1, $2, $3);
	`
	if _, err := tx.Exec(ctx, memberQuery, created.ID, room.OwnerID, string(domain_models.MemberRoleOwner)); err != nil {
		return domain_models.Room{}, fmt.Errorf("insert owner member: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain_models.Room{}, fmt.Errorf("commit tx: %w", err)
	}

	return created, nil
}

func (r *RoomsRepository) GetRoom(ctx context.Context, roomID string) (domain_models.Room, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT id, name, description, is_private, is_dm, owner_id, created_at
		FROM gochat.rooms WHERE id=$1;
	`
	row := r.pool.QueryRow(ctx, query, roomID)
	room, err := scanRoom(row)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain_models.Room{}, fmt.Errorf("room not found: %w", core_postgres_pool.ErrNoRows)
		}
		return domain_models.Room{}, err
	}
	return room, nil
}

func (r *RoomsRepository) GetPublicRooms(ctx context.Context, limit, offset int) ([]domain_models.Room, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT id, name, description, is_private, is_dm, owner_id, created_at
		FROM gochat.rooms
		WHERE is_private=false AND is_dm=false
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2;
	`
	return scanRooms(r.pool, ctx, query, limit, offset)
}

func (r *RoomsRepository) GetUserRooms(ctx context.Context, userID string) ([]domain_models.Room, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT r.id, r.name, r.description, r.is_private, r.is_dm, r.owner_id, r.created_at
		FROM gochat.rooms r
		JOIN gochat.room_members rm ON rm.room_id=r.id
		WHERE rm.user_id=$1 AND r.is_dm=false
		ORDER BY r.created_at DESC;
	`
	return scanRooms(r.pool, ctx, query, userID)
}

func (r *RoomsRepository) DeleteRoom(ctx context.Context, roomID, ownerID string) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `DELETE FROM gochat.rooms WHERE id=$1 AND owner_id=$2;`
	tag, err := r.pool.Exec(ctx, query, roomID, ownerID)
	if err != nil {
		return fmt.Errorf("delete room: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("room not found or not owned by user: %w", core_postgres_pool.ErrNoRows)
	}
	return nil
}

func scanRoom(row core_postgres_pool.Row) (domain_models.Room, error) {
	var m roomModel
	err := row.Scan(&m.ID, &m.Name, &m.Description, &m.IsPrivate, &m.IsDM, &m.OwnerID, &m.CreatedAt)
	if err != nil {
		return domain_models.Room{}, fmt.Errorf("scan room: %w", err)
	}
	return roomToDomain(m), nil
}

func scanRooms(pool core_postgres_pool.Pool, ctx context.Context, query string, args ...any) ([]domain_models.Room, error) {
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query rooms: %w", err)
	}
	defer rows.Close()

	var rooms []domain_models.Room
	for rows.Next() {
		var m roomModel
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.IsPrivate, &m.IsDM, &m.OwnerID, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan room row: %w", err)
		}
		rooms = append(rooms, roomToDomain(m))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return rooms, nil
}
