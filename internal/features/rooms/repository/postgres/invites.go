package rooms_repository_postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
	core_logger "go-chat/internal/core/logger"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"

	"go.uber.org/zap"
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

func (r *RoomsRepository) AcceptInvite(ctx context.Context, token, userID string) (domain_models.Room, domain_models.RoomInvite, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain_models.Room{}, domain_models.RoomInvite{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	inviteQuery := `
		SELECT room_id, max_uses, uses
		FROM gochat.room_invites
		WHERE token=$1
		  AND is_active=true
		  AND (expires_at IS NULL OR expires_at > NOW())
		  AND (max_uses = 0 OR uses < max_uses)
		FOR UPDATE;
	`
	var roomID string
	var maxUses, uses int
	if err := tx.QueryRow(ctx, inviteQuery, token).Scan(&roomID, &maxUses, &uses); err != nil {
		return domain_models.Room{}, domain_models.RoomInvite{}, fmt.Errorf("invite not usable: %w", err)
	}

	memberQuery := `SELECT EXISTS(SELECT 1 FROM gochat.room_members WHERE room_id=$1 AND user_id=$2);`
	var alreadyMember bool
	if err := tx.QueryRow(ctx, memberQuery, roomID, userID).Scan(&alreadyMember); err != nil {
		return domain_models.Room{}, domain_models.RoomInvite{}, fmt.Errorf("check membership: %w", err)
	}
	if alreadyMember {
		return domain_models.Room{}, domain_models.RoomInvite{}, fmt.Errorf("already a member: %w", core_error.ErrConflict)
	}

	updateQuery := `UPDATE gochat.room_invites SET uses = uses + 1 WHERE token=$1;`
	if _, err := tx.Exec(ctx, updateQuery, token); err != nil {
		return domain_models.Room{}, domain_models.RoomInvite{}, fmt.Errorf("increment invite uses: %w", err)
	}
	uses++

	insertQuery := `
		INSERT INTO gochat.room_members (room_id, user_id, role)
		VALUES ($1, $2, 'member');
	`
	if _, err := tx.Exec(ctx, insertQuery, roomID, userID); err != nil {
		if errors.Is(err, core_postgres_pool.ErrUniqueViolation) {
			return domain_models.Room{}, domain_models.RoomInvite{}, fmt.Errorf("already a member: %w", core_error.ErrConflict)
		}
		return domain_models.Room{}, domain_models.RoomInvite{}, fmt.Errorf("insert member: %w", err)
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
		return domain_models.Room{}, domain_models.RoomInvite{}, fmt.Errorf("accept invite: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain_models.Room{}, domain_models.RoomInvite{}, fmt.Errorf("commit tx: %w", err)
	}

	invite := domain_models.RoomInvite{
		Token:   token,
		RoomID:  roomID,
		MaxUses: maxUses,
		Uses:    uses,
	}
	return room, invite, nil
}

func (r *RoomsRepository) DeactivateInvite(ctx context.Context, token, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		DELETE FROM gochat.room_invites ri
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
		return fmt.Errorf("delete invite: %w", err)
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
		SELECT ri.id, ri.room_id, ri.token, u.username as created_by, ri.max_uses, ri.uses, ri.is_active, ri.expires_at, ri.created_at
		FROM gochat.room_invites ri
		JOIN gochat.users u ON u.id = ri.created_by
		WHERE ri.room_id=$1
		ORDER BY ri.created_at DESC;
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

func (r *RoomsRepository) cleanupExpiredInvites(ctx context.Context, log *core_logger.Logger) {
	const batchSize = 1000
	totalDeleted := int64(0)

	for {
		// Проверяем отмену контекста перед каждым батчем
		if ctx.Err() != nil {
			return
		}

		query := `
			DELETE FROM gochat.room_invites
			WHERE ctid IN (
				SELECT ctid
				FROM gochat.room_invites
				WHERE is_active = false
				   OR (max_uses > 0 AND uses >= max_uses)
				   OR (expires_at IS NOT NULL AND expires_at < NOW())
				LIMIT $1
			)
		`
		tag, err := r.pool.Exec(ctx, query, batchSize)
		if err != nil {
			log.Error("invite cleanup: delete failed", zap.Error(err))
			return
		}

		deleted := tag.RowsAffected()
		totalDeleted += deleted

		if deleted == 0 {
			break // больше ничего удалять не нужно
		}

		log.Debug("invite cleanup: batch deleted", zap.Int64("count", deleted))
	}

	if totalDeleted > 0 {
		log.Info("invite cleanup: completed", zap.Int64("total_deleted", totalDeleted))
	}
}

// StartInviteCleanup запускает фоновую очистку просроченных/использованных инвайтов
func (r *RoomsRepository) StartInviteCleanup(ctx context.Context, interval time.Duration, log *core_logger.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debug("invite cleanup: stopping")
			return
		case <-ticker.C:
			r.cleanupExpiredInvites(ctx, log)
		}
	}
}

// scanInvite сканирует строку в RoomInvite
func scanInvite(row core_postgres_pool.Row) (domain_models.RoomInvite, error) {
	var m inviteModel
	err := row.Scan(&m.ID, &m.RoomID, &m.Token, &m.CreatedBy,
		&m.MaxUses, &m.Uses, &m.IsActive, &m.ExpiresAt, &m.CreatedAt)
	if err != nil {
		return domain_models.RoomInvite{}, fmt.Errorf("scan invite: %w", err)
	}
	return inviteToDomain(m), nil
}
