package dm_service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
	dm_service "go-chat/internal/features/dm/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Моки ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) FindDM(ctx context.Context, userID1, userID2 string) (domain_models.Room, error) {
	args := m.Called(ctx, userID1, userID2)
	return args.Get(0).(domain_models.Room), args.Error(1)
}

func (m *MockRepository) CreateDM(ctx context.Context, userID1, userID2 string) (domain_models.Room, error) {
	args := m.Called(ctx, userID1, userID2)
	return args.Get(0).(domain_models.Room), args.Error(1)
}

func (m *MockRepository) GetUserDMs(ctx context.Context, userID string) ([]domain_models.Room, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain_models.Room), args.Error(1)
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetUser(ctx context.Context, userID string) (domain_models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(domain_models.User), args.Error(1)
}

// --- Хелперы ---

func newTestRoom(id string) domain_models.Room {
	return domain_models.NewRoom(id, "", "", true, true, "owner-id", time.Now())
}

func newTestUser(id string) domain_models.User {
	return domain_models.User{
		ID:        id,
		Username:  "user_" + id,
		Email:     id + "@test.com",
		Password:  "hashed",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func newService() (*dm_service.DMService, *MockRepository, *MockUserRepository) {
	repo := new(MockRepository)
	userRepo := new(MockUserRepository)
	svc := dm_service.NewDMService(repo, userRepo)
	return svc, repo, userRepo
}

// --- OpenDM тесты ---

func TestOpenDM_Success_CreateNew(t *testing.T) {
	svc, repo, userRepo := newService()
	ctx := context.Background()
	requesterID := "user-1"
	targetID := "user-2"
	expectedRoom := newTestRoom("dm-123")

	userRepo.On("GetUser", ctx, targetID).Return(newTestUser(targetID), nil)
	repo.On("FindDM", ctx, requesterID, targetID).Return(domain_models.Room{}, core_postgres_pool.ErrNoRows)
	repo.On("CreateDM", ctx, requesterID, targetID).Return(expectedRoom, nil)

	room, err := svc.OpenDM(ctx, requesterID, targetID)

	assert.NoError(t, err)
	assert.Equal(t, expectedRoom, room)
	userRepo.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestOpenDM_Success_Existing(t *testing.T) {
	svc, repo, userRepo := newService()
	ctx := context.Background()
	requesterID := "user-1"
	targetID := "user-2"
	existingRoom := newTestRoom("dm-456")

	userRepo.On("GetUser", ctx, targetID).Return(newTestUser(targetID), nil)
	repo.On("FindDM", ctx, requesterID, targetID).Return(existingRoom, nil)

	room, err := svc.OpenDM(ctx, requesterID, targetID)

	assert.NoError(t, err)
	assert.Equal(t, existingRoom, room)
	repo.AssertNotCalled(t, "CreateDM")
}

func TestOpenDM_Self(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	userID := "user-1"

	room, err := svc.OpenDM(ctx, userID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot open DM with yourself")
	assert.Empty(t, room.ID)
}

func TestOpenDM_TargetUserNotFound(t *testing.T) {
	svc, _, userRepo := newService()
	ctx := context.Background()
	requesterID := "user-1"
	targetID := "user-2"

	userRepo.On("GetUser", ctx, targetID).Return(domain_models.User{}, errors.New("not found"))

	room, err := svc.OpenDM(ctx, requesterID, targetID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target user not found")
	assert.Empty(t, room.ID)
	userRepo.AssertExpectations(t)
}

func TestOpenDM_FindDMError(t *testing.T) {
	svc, repo, userRepo := newService()
	ctx := context.Background()
	requesterID := "user-1"
	targetID := "user-2"
	dbErr := errors.New("db error")

	userRepo.On("GetUser", ctx, targetID).Return(newTestUser(targetID), nil)
	repo.On("FindDM", ctx, requesterID, targetID).Return(domain_models.Room{}, dbErr)

	room, err := svc.OpenDM(ctx, requesterID, targetID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find dm")
	assert.ErrorIs(t, err, dbErr)
	assert.Empty(t, room.ID)
	repo.AssertNotCalled(t, "CreateDM")
}

func TestOpenDM_CreateDMError(t *testing.T) {
	svc, repo, userRepo := newService()
	ctx := context.Background()
	requesterID := "user-1"
	targetID := "user-2"
	dbErr := errors.New("create error")

	userRepo.On("GetUser", ctx, targetID).Return(newTestUser(targetID), nil)
	repo.On("FindDM", ctx, requesterID, targetID).Return(domain_models.Room{}, core_postgres_pool.ErrNoRows)
	repo.On("CreateDM", ctx, requesterID, targetID).Return(domain_models.Room{}, dbErr)

	room, err := svc.OpenDM(ctx, requesterID, targetID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create error")
	assert.ErrorIs(t, err, dbErr)
	assert.Empty(t, room.ID)
}

// --- GetUserDMs тесты ---

func TestGetUserDMs_Success(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()
	userID := "user-1"
	expectedRooms := []domain_models.Room{
		newTestRoom("dm-1"),
		newTestRoom("dm-2"),
	}

	repo.On("GetUserDMs", ctx, userID).Return(expectedRooms, nil)

	rooms, err := svc.GetUserDMs(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedRooms, rooms)
	repo.AssertExpectations(t)
}

func TestGetUserDMs_Empty(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()
	userID := "user-1"

	repo.On("GetUserDMs", ctx, userID).Return([]domain_models.Room{}, nil)

	rooms, err := svc.GetUserDMs(ctx, userID)

	assert.NoError(t, err)
	assert.Empty(t, rooms)
	repo.AssertExpectations(t)
}

func TestGetUserDMs_RepoError(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()
	userID := "user-1"
	dbErr := errors.New("query error")

	repo.On("GetUserDMs", ctx, userID).Return(nil, dbErr)

	rooms, err := svc.GetUserDMs(ctx, userID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Nil(t, rooms)
}
