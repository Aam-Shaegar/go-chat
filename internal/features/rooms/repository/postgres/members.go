package rooms_repository_postgres

import (
	"context"
	"fmt"

	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
)

func (r *RoomsRepository) IsMember(ctx context.Context, roomID, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `SELECT EXISTS(SELECT 1 FROM gochat.room_members WHERE room_id=$1 AND user_id=$2);`
	row := r.pool.QueryRow(ctx, query, roomID, userID)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("check is member: %w", err)
	}
	return exists, nil
}

func (r *RoomsRepository) GetMember(ctx context.Context, roomID, userID string) (domain_models.RoomMember, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT rm.room_id, rm.user_id, u.username, rm.role, rm.joined_at
		FROM gochat.room_members rm
		JOIN gochat.users u ON u.id=rm.user_id
		WHERE rm.room_id=$1 AND rm.user_id=$2;
	`
	row := r.pool.QueryRow(ctx, query, roomID, userID)
	var m memberModel
	err := row.Scan(&m.RoomID, &m.UserID, &m.Username, &m.Role, &m.JoinedAt)
	if err != nil {
		if err == core_postgres_pool.ErrNoRows {
			return domain_models.RoomMember{}, fmt.Errorf("member not found: %w", core_postgres_pool.ErrNoRows)
		}
		return domain_models.RoomMember{}, fmt.Errorf("scan member: %w", err)
	}
	return memberToDomain(m), nil
}

func (r *RoomsRepository) GetMembers(ctx context.Context, roomID string) ([]domain_models.RoomMember, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		SELECT rm.room_id, rm.user_id, u.username, rm.role, rm.joined_at
		FROM gochat.room_members rm
		JOIN gochat.users u ON u.id=rm.user_id
		WHERE rm.room_id=$1
		ORDER BY rm.joined_at ASC;
	`
	rows, err := r.pool.Query(ctx, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("query members: %w", err)
	}
	defer rows.Close()

	var members []domain_models.RoomMember
	for rows.Next() {
		var m memberModel
		if err := rows.Scan(&m.RoomID, &m.UserID, &m.Username, &m.Role, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan member row: %w", err)
		}
		members = append(members, memberToDomain(m))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return members, nil
}

func (r *RoomsRepository) AddMember(ctx context.Context, roomID, userID string, role domain_models.MemberRole) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		INSERT INTO gochat.room_members (room_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING;
	`
	_, err := r.pool.Exec(ctx, query, roomID, userID, string(role))
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (r *RoomsRepository) RemoveMember(ctx context.Context, roomID, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `DELETE FROM gochat.room_members WHERE room_id=$1 AND user_id=$2;`
	tag, err := r.pool.Exec(ctx, query, roomID, userID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("member not found: %w", core_postgres_pool.ErrNoRows)
	}
	return nil
}

func (r *RoomsRepository) UpdateMemberRole(ctx context.Context, roomID, userID string, role domain_models.MemberRole) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `UPDATE gochat.room_members SET role=$1 WHERE room_id=$2 AND user_id=$3;`
	tag, err := r.pool.Exec(ctx, query, string(role), roomID, userID)
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("member not found: %w", core_postgres_pool.ErrNoRows)
	}
	return nil
}
