package rooms_repository_postgres

import (
	"context"
	"fmt"
	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
	"time"
)

func (r RoomsRepository) GetUser(ctx context.Context, userID string) (domain_models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `SELECT id, username, email, created_at, updated_at FROM gochat.users WHERE id=$1;`
	row := r.pool.QueryRow(ctx, query, userID)
	var u userModel
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == core_postgres_pool.ErrNoRows {
			return domain_models.User{}, fmt.Errorf("user not found: %w", core_postgres_pool.ErrNoRows)
		}
		return domain_models.User{}, fmt.Errorf("scan user: %w", err)
	}
	return userToDomain(u), nil
}

type userModel struct {
	ID        string
	Username  string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func userToDomain(u userModel) domain_models.User {
	return domain_models.User{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
