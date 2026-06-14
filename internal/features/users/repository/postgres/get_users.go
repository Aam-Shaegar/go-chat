package users_repository_postgres

import (
	"context"
	"fmt"

	domain_models "go-chat/internal/core/domain/models"
)

const (
	maxLimit     = 100
	defaultLimit = 20
)

func (r *UsersRepository) GetUsers(ctx context.Context, limit, offset *int) ([]domain_models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	l := defaultLimit
	o := 0
	if limit != nil && *limit > 0 {
		l = *limit
	}
	if l > maxLimit {
		l = maxLimit
	}
	if offset != nil && *offset > 0 {
		o = *offset
	}

	query := `
		SELECT id, username, email, password, created_at, updated_at
		FROM gochat.users
		ORDER BY id ASC
		LIMIT $1 OFFSET $2;
	`
	rows, err := r.pool.Query(ctx, query, l, o)
	if err != nil {
		return nil, fmt.Errorf("select users: %w", err)
	}
	defer rows.Close()

	var userModels []UserModel
	for rows.Next() {
		var userModel UserModel
		if err := rows.Scan(
			&userModel.ID, &userModel.Username, &userModel.Email,
			&userModel.Password, &userModel.CreatedAt, &userModel.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan users: %w", err)
		}
		userModels = append(userModels, userModel)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}
	return userDomainsFromModels(userModels), nil
}
