package rooms_repository_postgres

import (
	"context"
	"errors"
	"fmt"

	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
)

func (r *RoomsRepository) CreateInvite(ctx context.Context, invite domain_models.RoomInvite) (domain_models.RoomInvite, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		INSERT INTO gochat.room_invites (room_id, token, created_by, max_uses, uses, is_active, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, room_id, token, created_by, max_uses, uses, is_active, expires_at, created_at;
	`
	row := r.pool.QueryRow(ctx, query,
		invite.RoomID, invite.Token, invite.CreatedBy,
		invite.MaxUses, invite.Uses, invite.IsActive,
		invite.ExpiresAt, invite.CreatedAt,
	)
	return scanInvite(row)
}

func (r *RoomsRepository) GetInviteByToken(ctx context.Context, token string) (domain_models.RoomInvite, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT id, room_id, token, created_by, max_uses, uses, is_active, expires_at, created_at
		FROM gochat.room_invites WHERE token=$1;
	`
	row := r.pool.QueryRow(ctx, query, token)
	invite, err := scanInvite(row)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain_models.RoomInvite{}, fmt.Errorf("invite not found: %w", core_postgres_pool.ErrNoRows)
		}
		return domain_models.RoomInvite{}, err
	}
	return invite, nil
}

// TryIncrementInviteUses атомарно инкрементирует uses с проверкой лимита.
// Возвращает ErrNoRows, если инвайт исчерпан, истёк или неактивен.
func (r *RoomsRepository) TryIncrementInviteUses(ctx context.Context, token string) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		UPDATE gochat.room_invites
		SET uses = uses + 1
		WHERE token=$1
		  AND is_active=true
		  AND (expires_at IS NULL OR expires_at > NOW())
		  AND (max_uses = 0 OR uses < max_uses);
	`
	tag, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("increment invite uses: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("invite exhausted, expired or inactive: %w", core_postgres_pool.ErrNoRows)
	}
	return nil
}

// DeactivateInvite отключает инвайт. Может выполнить создатель инвайта или владелец комнаты.
func (r *RoomsRepository) DeactivateInvite(ctx context.Context, token, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		UPDATE gochat.room_invites ri
		SET is_active=false
		WHERE ri.token=$1
		  AND (
		    ri.created_by=$2
		    OR EXISTS (
		      SELECT 1 FROM gochat.rooms r
		      WHERE r.id=ri.room_id AND r.owner_id=$2
		    )
		  );
	`
	tag, err := r.pool.Exec(ctx, query, token, userID)
	if err != nil {
		return fmt.Errorf("deactivate invite: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("invite not found or access denied: %w", core_postgres_pool.ErrNoRows)
	}
	return nil
}

func (r *RoomsRepository) GetRoomInvites(ctx context.Context, roomID string) ([]domain_models.RoomInvite, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT id, room_id, token, created_by, max_uses, uses, is_active, expires_at, created_at
		FROM gochat.room_invites
		WHERE room_id=$1
		ORDER BY created_at DESC;
	`
	rows, err := r.pool.Query(ctx, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("query invites: %w", err)
	}
	defer rows.Close()

	var invites []domain_models.RoomInvite
	for rows.Next() {
		var m inviteModel
		if err := rows.Scan(&m.ID, &m.RoomID, &m.Token, &m.CreatedBy,
			&m.MaxUses, &m.Uses, &m.IsActive, &m.ExpiresAt, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan invite row: %w", err)
		}
		invites = append(invites, inviteToDomain(m))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return invites, nil
}

func scanInvite(row core_postgres_pool.Row) (domain_models.RoomInvite, error) {
	var m inviteModel
	err := row.Scan(&m.ID, &m.RoomID, &m.Token, &m.CreatedBy,
		&m.MaxUses, &m.Uses, &m.IsActive, &m.ExpiresAt, &m.CreatedAt)
	if err != nil {
		return domain_models.RoomInvite{}, fmt.Errorf("scan invite: %w", err)
	}
	return inviteToDomain(m), nil
}
