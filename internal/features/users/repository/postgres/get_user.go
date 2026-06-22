package users_repository_postgres

import (
	"context"
	"errors"
	"fmt"
	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
)

func (r *UsersRepository) GetUser(ctx context.Context, userID string) (domain_models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT id, username, email, password, created_at, updated_at
		FROM gochat.users
		WHERE id=$1;
	`
	row := r.pool.QueryRow(ctx, query, userID)
	var userModel UserModel
	err := row.Scan(
		&userModel.ID,
		&userModel.Username,
		&userModel.Email,
		&userModel.Password,
		&userModel.CreatedAt,
		&userModel.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain_models.User{}, fmt.Errorf("user with id='%s': %w", userID, core_postgres_pool.ErrNoRows)
		}
		return domain_models.User{}, fmt.Errorf("scan error: %w", err)
	}
	return domain_models.NewUser(
		userModel.ID,
		userModel.Username,
		userModel.Email,
		userModel.Password,
		userModel.CreatedAt,
		userModel.UpdatedAt,
	), nil
}
