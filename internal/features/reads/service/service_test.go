package reads_service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	core_error "go-chat/internal/core/errors"
	reads_service "go-chat/internal/features/reads/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Моки ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) MarkRead(ctx context.Context, roomID, userID string, lastReadAt time.Time) error {
	args := m.Called(ctx, roomID, userID, lastReadAt)
	return args.Error(0)
}

func (m *MockRepository) GetUnreadCounts(ctx context.Context, userID string) (map[string]int64, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int64), args.Error(1)
}

func (m *MockRepository) GetUnreadCount(ctx context.Context, roomID, userID string) (int64, error) {
	args := m.Called(ctx, roomID, userID)
	return args.Get(0).(int64), args.Error(1)
}

type MockRoomRepository struct {
	mock.Mock
}

func (m *MockRoomRepository) IsMember(ctx context.Context, roomID, userID string) (bool, error) {
	args := m.Called(ctx, roomID, userID)
	return args.Bool(0), args.Error(1)
}

// --- Хелперы ---

func newService() (*reads_service.ReadsService, *MockRepository, *MockRoomRepository) {
	repo := new(MockRepository)
	roomRepo := new(MockRoomRepository)
	svc := reads_service.NewReadsService(repo, roomRepo)
	return svc, repo, roomRepo
}

// --- MarkRead тесты ---

func TestMarkRead_Success(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"

	roomRepo.On("IsMember", ctx, roomID, userID).Return(true, nil)
	// Проверяем, что lastReadAt передаётся (время примерно сейчас)
	repo.On("MarkRead", ctx, roomID, userID, mock.MatchedBy(func(t time.Time) bool {
		return time.Since(t) < 5*time.Second // допуск 5 сек
	})).Return(nil)

	err := svc.MarkRead(ctx, roomID, userID)

	assert.NoError(t, err)
	roomRepo.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestMarkRead_NotMember(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"

	roomRepo.On("IsMember", ctx, roomID, userID).Return(false, nil)

	err := svc.MarkRead(ctx, roomID, userID)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
	assert.Contains(t, err.Error(), "access denied")
	roomRepo.AssertExpectations(t)
	repo.AssertNotCalled(t, "MarkRead")
}

func TestMarkRead_IsMemberError(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"

	dbErr := errors.New("database error")
	roomRepo.On("IsMember", ctx, roomID, userID).Return(false, dbErr)

	err := svc.MarkRead(ctx, roomID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check membership")
	assert.ErrorIs(t, err, dbErr)
	repo.AssertNotCalled(t, "MarkRead")
}

func TestMarkRead_RepoError(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"

	dbErr := errors.New("save error")
	roomRepo.On("IsMember", ctx, roomID, userID).Return(true, nil)
	repo.On("MarkRead", ctx, roomID, userID, mock.Anything).Return(dbErr)

	err := svc.MarkRead(ctx, roomID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save error")
	assert.ErrorIs(t, err, dbErr)
}

// --- GetUnreadCounts тесты ---

func TestGetUnreadCounts_Success(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()
	userID := "user-456"

	expected := map[string]int64{
		"room-1": 5,
		"room-2": 0,
	}
	repo.On("GetUnreadCounts", ctx, userID).Return(expected, nil)

	result, err := svc.GetUnreadCounts(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestGetUnreadCounts_RepoError(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()
	userID := "user-456"

	dbErr := errors.New("query failed")
	repo.On("GetUnreadCounts", ctx, userID).Return(nil, dbErr)

	result, err := svc.GetUnreadCounts(ctx, userID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Nil(t, result)
}

// --- GetUnreadCount тесты ---

func TestGetUnreadCount_Success(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"

	roomRepo.On("IsMember", ctx, roomID, userID).Return(true, nil)
	repo.On("GetUnreadCount", ctx, roomID, userID).Return(int64(3), nil)

	count, err := svc.GetUnreadCount(ctx, roomID, userID)

	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
	roomRepo.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestGetUnreadCount_NotMember(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"

	roomRepo.On("IsMember", ctx, roomID, userID).Return(false, nil)

	count, err := svc.GetUnreadCount(ctx, roomID, userID)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
	assert.Contains(t, err.Error(), "access denied")
	assert.Equal(t, int64(0), count)
	repo.AssertNotCalled(t, "GetUnreadCount")
}

func TestGetUnreadCount_IsMemberError(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"

	dbErr := errors.New("db error")
	roomRepo.On("IsMember", ctx, roomID, userID).Return(false, dbErr)

	count, err := svc.GetUnreadCount(ctx, roomID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check membership")
	assert.ErrorIs(t, err, dbErr)
	assert.Equal(t, int64(0), count)
	repo.AssertNotCalled(t, "GetUnreadCount")
}

func TestGetUnreadCount_RepoError(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"

	dbErr := errors.New("count query failed")
	roomRepo.On("IsMember", ctx, roomID, userID).Return(true, nil)
	repo.On("GetUnreadCount", ctx, roomID, userID).Return(int64(0), dbErr)

	count, err := svc.GetUnreadCount(ctx, roomID, userID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Equal(t, int64(0), count)
}
