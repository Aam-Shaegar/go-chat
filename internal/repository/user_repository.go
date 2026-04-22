package repository

import (
	"context"
	"fmt"
	"go-chat/internal/domain"

	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	query := `
		INSERT INTO users (username, email, password)
		VALUES (:username, :email, :password)
		RETURNING id, username, email, password, created_at, updated_at
	`

	rows, err := r.db.NamedQueryContext(ctx, query, user)
	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}
	defer rows.Close()

	var created domain.User
	if rows.Next() {
		if err := rows.StructScan(&created); err != nil {
			return domain.User{}, fmt.Errorf("scan user: %w", err)
		}
	}

	return created, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	var user domain.User
	query := `SELECT id, username, email, password, created_at, updated_at FROM users WHERE email = $1`
	if err := r.db.GetContext(ctx, &user, query, email); err != nil {
		return domain.User{}, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	if err := r.db.GetContext(ctx, &exists, query, email); err != nil {
		return false, fmt.Errorf("check email exists: %w", err)
	}

	return exists, nil
}

func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	if err := r.db.GetContext(ctx, &exists, query, username); err != nil {
		return false, fmt.Errorf("check username exists: %w", err)
	}

	return exists, nil
}

func (r *UserRepository) ListAll(ctx context.Context, excludeID string) ([]domain.User, error) {
	var users []domain.User
	query := `SELECT id, username, email, created_at, updated_at
		FROM users
		WHERE id != $1
		ORDER BY username ASC
	`
	if err := r.db.SelectContext(ctx, &users, query, excludeID); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	if users == nil {
		users = []domain.User{}
	}
	return users, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	var user domain.User
	query := `SELECT id, username, email, created_at, updated_at FROM users WHERE id  = $1`
	if err := r.db.GetContext(ctx, &user, query, id); err != nil {
		return domain.User{}, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}
