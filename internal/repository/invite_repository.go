package repository

import (
	"context"
	"fmt"

	"go-chat/internal/domain"

	"github.com/jmoiron/sqlx"
)

type InviteRepository struct {
	db *sqlx.DB
}

func NewInviteRepository(db *sqlx.DB) *InviteRepository {
	return &InviteRepository{db: db}
}

func (r *InviteRepository) Create(ctx context.Context, roomID, userID string) (domain.RoomInvite, error) {
	query := `
        INSERT INTO room_invites (room_id, created_by)
        VALUES ($1, $2)
        RETURNING id, room_id, created_by, used_by, expires_at, created_at
    `
	var invite domain.RoomInvite
	if err := r.db.GetContext(ctx, &invite, query, roomID, userID); err != nil {
		return domain.RoomInvite{}, fmt.Errorf("create invite: %w", err)
	}
	return invite, nil
}

func (r *InviteRepository) GetByID(ctx context.Context, inviteID string) (domain.RoomInvite, error) {
	var invite domain.RoomInvite
	query := `
        SELECT id, room_id, created_by, used_by, expires_at, created_at
        FROM room_invites WHERE id = $1
    `
	if err := r.db.GetContext(ctx, &invite, query, inviteID); err != nil {
		return domain.RoomInvite{}, fmt.Errorf("get invite: %w", err)
	}
	return invite, nil
}

func (r *InviteRepository) MarkUsed(ctx context.Context, inviteID, userID string) error {
	query := `UPDATE room_invites SET used_by = $1 WHERE id = $2`
	if _, err := r.db.ExecContext(ctx, query, userID, inviteID); err != nil {
		return fmt.Errorf("mark invite used: %w", err)
	}
	return nil
}

func (r *InviteRepository) ListByRoom(ctx context.Context, roomID string) ([]domain.RoomInvite, error) {
	var invites []domain.RoomInvite
	query := `
        SELECT id, room_id, created_by, used_by, expires_at, created_at
        FROM room_invites
        WHERE room_id = $1
        ORDER BY created_at DESC
    `
	if err := r.db.SelectContext(ctx, &invites, query, roomID); err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	return invites, nil
}

func (r *InviteRepository) Delete(ctx context.Context, inviteID string) error {
	query := `DELETE FROM room_invites WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, query, inviteID); err != nil {
		return fmt.Errorf("delete invite: %w", err)
	}
	return nil
}
