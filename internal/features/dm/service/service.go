package dm_service

import (
	"context"
	"errors"
	"fmt"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
)

type DMService struct {
	repo      Repository
	roomRepo  RoomRepository
	userRepo  UserRepository
}

func NewDMService(repo Repository, roomRepo RoomRepository, userRepo UserRepository) *DMService {
	return &DMService{repo: repo, roomRepo: roomRepo, userRepo: userRepo}
}

type RoomRepository interface {
	GetMembers(ctx context.Context, roomID string) ([]domain_models.RoomMember, error)
}

type Repository interface {
	FindDM(ctx context.Context, userID1, userID2 string) (domain_models.Room, error)
	CreateDM(ctx context.Context, userID1, userID2 string) (domain_models.Room, error)
	GetUserDMs(ctx context.Context, userID string) ([]domain_models.Room, error)
}

type UserRepository interface {
	GetUser(ctx context.Context, userID string) (domain_models.User, error)
}

// OpenDM возвращает существующий DM или создаёт новый.
func (s *DMService) OpenDM(ctx context.Context, requesterID, targetUserID string) (domain_models.Room, error) {
	if requesterID == targetUserID {
		return domain_models.Room{}, fmt.Errorf("cannot open DM with yourself: %w", core_error.ErrInvalidArgument)
	}

	// Проверяем, что целевой пользователь существует
	if _, err := s.userRepo.GetUser(ctx, targetUserID); err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) || errors.Is(err, core_error.ErrNotFound) {
			return domain_models.Room{}, fmt.Errorf("target user not found: %w", core_error.ErrNotFound)
		}
		return domain_models.Room{}, fmt.Errorf("target user not found: %w", err)
	}

	room, err := s.repo.FindDM(ctx, requesterID, targetUserID)
	if err == nil {
		return room, nil
	}
	if !errors.Is(err, core_postgres_pool.ErrNoRows) {
		return domain_models.Room{}, fmt.Errorf("find dm: %w", err)
	}

	room, err = s.repo.CreateDM(ctx, requesterID, targetUserID)
	if err == nil {
		return room, nil
	}
	if errors.Is(err, core_postgres_pool.ErrUniqueViolation) {
		return s.repo.FindDM(ctx, requesterID, targetUserID)
	}
	return domain_models.Room{}, err
}

func (s *DMService) GetUserDMs(ctx context.Context, userID string) ([]domain_models.Room, error) {
	rooms, err := s.repo.GetUserDMs(ctx, userID)
	if err != nil {
		return nil, err
	}
	// Populate OtherUserID for each DM
	for i := range rooms {
		if rooms[i].IsDM {
			members, err := s.roomRepo.GetMembers(ctx, rooms[i].ID)
			if err == nil {
				for _, m := range members {
					if m.UserID != userID {
						rooms[i].OtherUserID = m.UserID
						break
					}
				}
			}
		}
	}
	return rooms, nil
}
