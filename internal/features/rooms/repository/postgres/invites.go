package rooms_repository_postgres

import (
	"context"
	"errors"
	"fmt"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
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

func (r *RoomsRepository) AcceptInvite(ctx context.Context, token, userID string) (domain_models.Room, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain_models.Room{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	inviteQuery := `
		SELECT room_id
		FROM gochat.room_invites
		WHERE token=$1
		  AND is_active=true
		  AND (expires_at IS NULL OR expires_at > NOW())
		  AND (max_uses = 0 OR uses < max_uses)
		FOR UPDATE;
	`
	var roomID string
	if err := tx.QueryRow(ctx, inviteQuery, token).Scan(&roomID); err != nil {
		return domain_models.Room{}, fmt.Errorf("invite not usable: %w", err)
	}

	memberQuery := `SELECT EXISTS(SELECT 1 FROM gochat.room_members WHERE room_id=$1 AND user_id=$2);`
	var alreadyMember bool
	if err := tx.QueryRow(ctx, memberQuery, roomID, userID).Scan(&alreadyMember); err != nil {
		return domain_models.Room{}, fmt.Errorf("check membership: %w", err)
	}
	if alreadyMember {
		return domain_models.Room{}, fmt.Errorf("already a member: %w", core_error.ErrConflict)
	}

	updateQuery := `UPDATE gochat.room_invites SET uses = uses + 1 WHERE token=$1;`
	if _, err := tx.Exec(ctx, updateQuery, token); err != nil {
		return domain_models.Room{}, fmt.Errorf("increment invite uses: %w", err)
	}

	insertQuery := `
		INSERT INTO gochat.room_members (room_id, user_id, role)
		VALUES ($1, $2, 'member');
	`
	if _, err := tx.Exec(ctx, insertQuery, roomID, userID); err != nil {
		if errors.Is(err, core_postgres_pool.ErrUniqueViolation) {
			return domain_models.Room{}, fmt.Errorf("already a member: %w", core_error.ErrConflict)
		}
		return domain_models.Room{}, fmt.Errorf("insert member: %w", err)
	}

	roomQuery := `
		SELECT r.id, r.name, r.description, r.is_private, r.is_dm, r.owner_id, r.created_at,
		       COALESCE(MAX(m.created_at), r.created_at) AS last_message_at
		FROM gochat.rooms r
		LEFT JOIN gochat.messages m ON m.room_id = r.id AND m.deleted_at IS NULL
		WHERE r.id=$1
		GROUP BY r.id;
	`
	room, err := scanRoom(tx.QueryRow(ctx, roomQuery, roomID))
	if err != nil {
		return domain_models.Room{}, fmt.Errorf("accept invite: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain_models.Room{}, fmt.Errorf("commit tx: %w", err)
	}
	return room, nil
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
