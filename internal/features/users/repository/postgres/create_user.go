package users_repository_postgres

import (
	"context"
	"errors"
	"fmt"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
)

func (r *UsersRepository) CreateUser(ctx context.Context, user domain_models.User) (domain_models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		INSERT INTO gochat.users (username, email, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, username, email, password, created_at, updated_at;
	`
	row := r.pool.QueryRow(ctx, query, user.Username, user.Email, user.Password, user.CreatedAt, user.UpdatedAt)
	var userModel UserModel
	err := row.Scan(
		&userModel.ID, &userModel.Username, &userModel.Email,
		&userModel.Password, &userModel.CreatedAt, &userModel.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrUniqueViolation) {
			return domain_models.User{}, fmt.Errorf("user already exists: %w", core_error.ErrConflict)
		}
		return domain_models.User{}, fmt.Errorf("scan created user: %w", err)
	}
	return domain_models.NewUser(
		userModel.ID, userModel.Username, userModel.Email,
		userModel.Password, userModel.CreatedAt, userModel.UpdatedAt,
	), nil
}
