package dm_service

import (
	"context"
	"errors"
	"fmt"

	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
)

type DMService struct {
	repo     Repository
	userRepo UserRepository
}

func NewDMService(repo Repository, userRepo UserRepository) *DMService {
	return &DMService{repo: repo, userRepo: userRepo}
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
		return domain_models.Room{}, fmt.Errorf("cannot open DM with yourself")
	}

	// Проверяем что target пользователь существует
	if _, err := s.userRepo.GetUser(ctx, targetUserID); err != nil {
		return domain_models.Room{}, fmt.Errorf("target user not found: %w", err)
	}

	// Ищем существующий DM
	room, err := s.repo.FindDM(ctx, requesterID, targetUserID)
	if err == nil {
		// Уже существует — возвращаем
		return room, nil
	}
	if !errors.Is(err, core_postgres_pool.ErrNoRows) {
		return domain_models.Room{}, fmt.Errorf("find dm: %w", err)
	}

	// Создаём новый DM
	return s.repo.CreateDM(ctx, requesterID, targetUserID)
}

func (s *DMService) GetUserDMs(ctx context.Context, userID string) ([]domain_models.Room, error) {
	return s.repo.GetUserDMs(ctx, userID)
}
