package rooms_repository_postgres

import (
	"context"
	"fmt"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
	core_logger "go-chat/internal/core/logger"
	"go.uber.org/zap"
)

func (r *RoomsRepository) CreateBan(ctx context.Context, ban domain_models.RoomBan) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
        INSERT INTO gochat.room_bans (room_id, user_id, banned_by, reason, expires_at)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (room_id, user_id) DO UPDATE SET
            banned_by = EXCLUDED.banned_by,
            reason = EXCLUDED.reason,
            expires_at = EXCLUDED.expires_at,
            created_at = NOW();
    `
	_, err := r.pool.Exec(ctx, query, ban.RoomID, ban.UserID, ban.BannedBy, ban.Reason, ban.ExpiresAt)
	return err
}

func (r *RoomsRepository) GetBan(ctx context.Context, roomID, userID string) (domain_models.RoomBan, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
        SELECT room_id, user_id, banned_by, reason, expires_at, created_at
        FROM gochat.room_bans WHERE room_id=$1 AND user_id=$2;
    `
	row := r.pool.QueryRow(ctx, query, roomID, userID)
	var b banModel
	err := row.Scan(&b.RoomID, &b.UserID, &b.BannedBy, &b.Reason, &b.ExpiresAt, &b.CreatedAt)
	if err == core_postgres_pool.ErrNoRows {
		return domain_models.RoomBan{}, fmt.Errorf("ban not found: %w", core_postgres_pool.ErrNoRows)
	}
	return banToDomain(b), err
}

func (r *RoomsRepository) RemoveBan(ctx context.Context, roomID, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `DELETE FROM gochat.room_bans WHERE room_id=$1 AND user_id=$2;`
	tag, err := r.pool.Exec(ctx, query, roomID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("ban not found: %w", core_postgres_pool.ErrNoRows)
	}
	return nil
}

func (r *RoomsRepository) IsBanned(ctx context.Context, roomID, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `SELECT EXISTS(SELECT 1 FROM gochat.room_bans WHERE room_id=$1 AND user_id=$2 AND (expires_at IS NULL OR expires_at > NOW()));`
	var exists bool
	err := r.pool.QueryRow(ctx, query, roomID, userID).Scan(&exists)
	return exists, err
}

func (r *RoomsRepository) GetRoomBans(ctx context.Context, roomID string) ([]domain_models.RoomBan, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `SELECT room_id, user_id, banned_by, reason, expires_at, created_at FROM gochat.room_bans WHERE room_id=$1 ORDER BY created_at DESC;`
	rows, err := r.pool.Query(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bans []domain_models.RoomBan
	for rows.Next() {
		var b banModel
		if err := rows.Scan(&b.RoomID, &b.UserID, &b.BannedBy, &b.Reason, &b.ExpiresAt, &b.CreatedAt); err != nil {
			return nil, err
		}
		bans = append(bans, banToDomain(b))
	}
	return bans, rows.Err()
}

func (r *RoomsRepository) CleanExpiredBans(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `DELETE FROM gochat.room_bans WHERE expires_at IS NOT NULL AND expires_at <= NOW();`
	_, err := r.pool.Exec(ctx, query)
	return err
}

type banModel struct {
	RoomID    string
	UserID    string
	BannedBy  string
	Reason    string
	ExpiresAt *time.Time
	CreatedAt time.Time
}

func banToDomain(b banModel) domain_models.RoomBan {
	return domain_models.RoomBan{
		RoomID:    b.RoomID,
		UserID:    b.UserID,
		BannedBy:  b.BannedBy,
		Reason:    b.Reason,
		ExpiresAt: b.ExpiresAt,
		CreatedAt: b.CreatedAt,
	}
}

func (r *RoomsRepository) StartBanCleanup(ctx context.Context, interval time.Duration, log *core_logger.Logger) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Debug("ban cleanup: stopping")
				return
			case <-ticker.C:
				if err := r.CleanExpiredBans(ctx); err != nil {
					log.Error("ban cleanup failed", zap.Error(err))
				}
			}
		}
	}()
}
