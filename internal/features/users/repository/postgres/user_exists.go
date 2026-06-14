package users_repository_postgres

import (
	"context"
	"fmt"
)

func (r *UsersRepository) UserExistsByEmail(ctx context.Context, email string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `SELECT EXISTS(SELECT 1 FROM gochat.users WHERE email=$1);`
	row := r.pool.QueryRow(ctx, query, email)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("scan exists by email: %w", err)
	}
	return exists, nil
}

func (r *UsersRepository) UserExistsByUsername(ctx context.Context, username string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `SELECT EXISTS(SELECT 1 FROM gochat.users WHERE username=$1);`
	row := r.pool.QueryRow(ctx, query, username)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("scan exists by username: %w", err)
	}
	return exists, nil
}
