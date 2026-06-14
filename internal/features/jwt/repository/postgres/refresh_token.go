package jwt_repository_postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"

	"github.com/google/uuid"
)

func (r *JwtRepository) SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		INSERT INTO gochat.refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4);
	`
	_, err := r.pool.Exec(ctx, query, uuid.New(), userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (r *JwtRepository) GetRefreshToken(ctx context.Context, tokenHash string) (domain_models.RefreshToken, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT id, user_id, token_hash, expires_at, created_at
		FROM gochat.refresh_tokens
		WHERE token_hash=$1;
	`
	row := r.pool.QueryRow(ctx, query, tokenHash)
	var m domain_models.RefreshToken
	err := row.Scan(&m.ID, &m.UserID, &m.TokenHash, &m.ExpiresAt, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain_models.RefreshToken{}, fmt.Errorf("refresh token not found: %w", core_postgres_pool.ErrNoRows)
		}
		return domain_models.RefreshToken{}, fmt.Errorf("scan refresh token: %w", err)
	}
	return m, nil
}

func (r *JwtRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `DELETE FROM gochat.refresh_tokens WHERE token_hash=$1;`
	tag, err := r.pool.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("refresh token not found: %w", core_postgres_pool.ErrNoRows)
	}
	return nil
}

// ReplaceRefreshToken атомарно отзывает старый токен и сохраняет новый.
// Если любая из операций падает — транзакция откатывается,
// клиент остаётся со старым токеном и может повторить запрос.
func (r *JwtRepository) ReplaceRefreshToken(ctx context.Context, oldTokenHash, userID, newTokenHash string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Сначала сохраняем новый
	insertQuery := `
		INSERT INTO gochat.refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4);
	`
	if _, err := tx.Exec(ctx, insertQuery, uuid.New(), userID, newTokenHash, expiresAt); err != nil {
		return fmt.Errorf("insert new refresh token: %w", err)
	}

	// Потом отзываем старый
	deleteQuery := `DELETE FROM gochat.refresh_tokens WHERE token_hash=$1;`
	tag, err := tx.Exec(ctx, deleteQuery, oldTokenHash)
	if err != nil {
		return fmt.Errorf("revoke old refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("old refresh token not found: %w", core_postgres_pool.ErrNoRows)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
