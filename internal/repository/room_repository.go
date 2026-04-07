package repository

import (
	"context"
	"fmt"
	"go-chat/internal/domain"

	"github.com/jmoiron/sqlx"
)

type RoomRepository struct {
	db *sqlx.DB
}

func NewRoomRepository(db *sqlx.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) Create(ctx context.Context, room domain.Room) (domain.Room, error) {
	query := `
		INSERT INTO rooms (name, description, is_private, owner_id)
		VALUES (:name, :description, :is_private, :owner_id)
		RETURNING id, name, description, is_private, owner_id, created_at
	`

	rows, err := r.db.NamedQueryContext(ctx, query, room)
	if err != nil {
		return domain.Room{}, fmt.Errorf("create room: %w", err)
	}
	defer rows.Close()

	var created domain.Room
	if rows.Next() {
		if err := rows.StructScan(&created); err != nil {
			return domain.Room{}, fmt.Errorf("scan room: %w", err)
		}
	}

	return created, nil
}

func (r *RoomRepository) GetRoomByID(ctx context.Context, id string) (domain.Room, error) {
	var room domain.Room
	query := `
		SELECT id, name, description, is_private, owner_id, created_at
		FROM rooms WHERE id = $1
	`
	if err := r.db.GetContext(ctx, &room, query, id); err != nil {
		return domain.Room{}, fmt.Errorf("get room: %w", err)
	}

	return room, nil
}

func (r *RoomRepository) ListPublic(ctx context.Context) ([]domain.Room, error) {
	var rooms []domain.Room
	query := `
		SELECT id, name, description, is_private, owner_id, created_at
		FROM rooms 
		WHERE is_private = false
		ORDER BY created_at DESC
	`
	if err := r.db.SelectContext(ctx, &rooms, query); err != nil {
		return nil, fmt.Errorf("list public rooms: %w", err)
	}
	return rooms, nil
}

func (r *RoomRepository) ListByUserID(ctx context.Context, userID string) ([]domain.Room, error) {
	var rooms []domain.Room
	query := `
		SELECT rooms.id, rooms.name, rooms.description, rooms.is_private, rooms.owner_id, rooms.created_at
		FROM rooms 
		INNER JOIN room_members ON room_members.room_id = rooms.id
		WHERE room_members.user_id = $1
		ORDER BY rooms.created_at DESC
	`
	if err := r.db.SelectContext(ctx, &rooms, query, userID); err != nil {
		return nil, fmt.Errorf("list user rooms: %w", err)
	}
	return rooms, nil
}

func (r *RoomRepository) AddMember(ctx context.Context, roomID, userID string) error {
	query := `
		INSERT INTO room_members (room_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (room_id, user_id) DO NOTHING
	`
	if _, err := r.db.ExecContext(ctx, query, roomID, userID); err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (r *RoomRepository) Delete(ctx context.Context, roomID string) error {
	query := `DELETE FROM rooms WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, query, roomID); err != nil {
		return fmt.Errorf("delete room: %w", err)
	}
	return nil
}

func (r *RoomRepository) GetOwnerID(ctx context.Context, roomID string) (string, error) {
	var ownerID string
	query := `SELECT owner_id FROM rooms WHERE id = $1`
	if err := r.db.GetContext(ctx, &ownerID, query, roomID); err != nil {
		return "", fmt.Errorf("get owner: %w", err)
	}
	return ownerID, nil
}

func (r *RoomRepository) IsMember(ctx context.Context, roomID, userID string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(SELECT 1 FROM room_members WHERE room_id = $1 AND user_id = $2)
	`
	if err := r.db.GetContext(ctx, &exists, query, roomID, userID); err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	return exists, nil
}
