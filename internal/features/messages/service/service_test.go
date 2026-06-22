package messages_service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
	messages_service "go-chat/internal/features/messages/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Моки ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetMessages(ctx context.Context, roomID string, before *time.Time, limit int) ([]domain_models.Message, error) {
	args := m.Called(ctx, roomID, before, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain_models.Message), args.Error(1)
}

type MockRoomRepository struct {
	mock.Mock
}

func (m *MockRoomRepository) IsMember(ctx context.Context, roomID, userID string) (bool, error) {
	args := m.Called(ctx, roomID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRoomRepository) GetRoom(ctx context.Context, roomID string) (domain_models.Room, error) {
	args := m.Called(ctx, roomID)
	return args.Get(0).(domain_models.Room), args.Error(1)
}

// --- Хелперы ---

func newService() (*messages_service.MessagesService, *MockRepository, *MockRoomRepository) {
	repo := new(MockRepository)
	roomRepo := new(MockRoomRepository)
	svc := messages_service.NewMessagesService(repo, roomRepo)
	return svc, repo, roomRepo
}

func newRoom(isPrivate, isDM bool) domain_models.Room {
	return domain_models.NewRoom("room-123", "test", "desc", isPrivate, isDM, "owner", time.Now())
}

func newMessage(id string, createdAt time.Time) domain_models.Message {
	return domain_models.NewMessage(id, "room-123", "user", nil, "content", false, createdAt, createdAt)
}

// --- Тесты ---

func TestGetMessages_PublicRoom_Success(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"
	limit := 10
	now := time.Now()
	messages := []domain_models.Message{
		newMessage("1", now.Add(-2*time.Hour)),
		newMessage("2", now.Add(-1*time.Hour)),
	}

	roomRepo.On("GetRoom", ctx, roomID).Return(newRoom(false, false), nil)
	// Для публичной комнаты не вызывается IsMember
	repo.On("GetMessages", ctx, roomID, mock.Anything, limit+1).Return(messages, nil)

	result, err := svc.GetMessages(ctx, roomID, userID, nil, limit)

	assert.NoError(t, err)
	assert.Equal(t, messages, result.Messages)
	assert.False(t, result.HasMore)
	assert.Nil(t, result.NextCursor)
	roomRepo.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestGetMessages_PrivateRoom_Success(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"
	limit := 10
	messages := []domain_models.Message{
		newMessage("1", time.Now()),
	}

	roomRepo.On("GetRoom", ctx, roomID).Return(newRoom(true, false), nil)
	roomRepo.On("IsMember", ctx, roomID, userID).Return(true, nil)
	repo.On("GetMessages", ctx, roomID, mock.Anything, limit+1).Return(messages, nil)

	result, err := svc.GetMessages(ctx, roomID, userID, nil, limit)

	assert.NoError(t, err)
	assert.Equal(t, messages, result.Messages)
	assert.False(t, result.HasMore)
}

func TestGetMessages_DMRoom_Success(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"
	limit := 10
	messages := []domain_models.Message{
		newMessage("1", time.Now()),
	}

	roomRepo.On("GetRoom", ctx, roomID).Return(newRoom(true, true), nil)
	roomRepo.On("IsMember", ctx, roomID, userID).Return(true, nil)
	repo.On("GetMessages", ctx, roomID, mock.Anything, limit+1).Return(messages, nil)

	result, err := svc.GetMessages(ctx, roomID, userID, nil, limit)

	assert.NoError(t, err)
	assert.Equal(t, messages, result.Messages)
}

func TestGetMessages_PrivateRoom_NotMember(t *testing.T) {
	svc, _, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"

	roomRepo.On("GetRoom", ctx, roomID).Return(newRoom(true, false), nil)
	roomRepo.On("IsMember", ctx, roomID, userID).Return(false, nil)

	result, err := svc.GetMessages(ctx, roomID, userID, nil, 10)

	assert.ErrorIs(t, err, core_error.ErrUnauthorized)
	assert.Contains(t, err.Error(), "access denied")
	assert.Empty(t, result.Messages)
	roomRepo.AssertExpectations(t)
}

func TestGetMessages_GetRoomError(t *testing.T) {
	svc, _, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"
	dbErr := errors.New("db error")

	roomRepo.On("GetRoom", ctx, roomID).Return(domain_models.Room{}, dbErr)

	result, err := svc.GetMessages(ctx, roomID, userID, nil, 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get room")
	assert.ErrorIs(t, err, dbErr)
	assert.Empty(t, result.Messages)
	roomRepo.AssertExpectations(t)
}

func TestGetMessages_IsMemberError(t *testing.T) {
	svc, _, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"
	dbErr := errors.New("db error")

	roomRepo.On("GetRoom", ctx, roomID).Return(newRoom(true, false), nil)
	roomRepo.On("IsMember", ctx, roomID, userID).Return(false, dbErr)

	result, err := svc.GetMessages(ctx, roomID, userID, nil, 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check membership")
	assert.ErrorIs(t, err, dbErr)
	assert.Empty(t, result.Messages)
}

func TestGetMessages_RepoError(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"
	dbErr := errors.New("query error")

	roomRepo.On("GetRoom", ctx, roomID).Return(newRoom(false, false), nil)
	repo.On("GetMessages", ctx, roomID, mock.Anything, 11).Return(nil, dbErr)

	result, err := svc.GetMessages(ctx, roomID, userID, nil, 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get messages")
	assert.ErrorIs(t, err, dbErr)
	assert.Empty(t, result.Messages)
}

func TestGetMessages_HasMore_True(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"
	limit := 2
	now := time.Now()
	// Возвращаем 3 сообщения (limit+1) → hasMore true
	messages := []domain_models.Message{
		newMessage("1", now.Add(-3*time.Hour)),
		newMessage("2", now.Add(-2*time.Hour)),
		newMessage("3", now.Add(-1*time.Hour)),
	}

	roomRepo.On("GetRoom", ctx, roomID).Return(newRoom(false, false), nil)
	repo.On("GetMessages", ctx, roomID, mock.Anything, limit+1).Return(messages, nil)

	result, err := svc.GetMessages(ctx, roomID, userID, nil, limit)

	assert.NoError(t, err)
	assert.True(t, result.HasMore)
	assert.Len(t, result.Messages, limit)
	assert.NotNil(t, result.NextCursor)
	// Курсор = created_at самого старого из оставшихся (первый в массиве после обрезки)
	expectedCursor := messages[0].CreatedAt
	assert.Equal(t, &expectedCursor, result.NextCursor)
}

func TestGetMessages_HasMore_False(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"
	limit := 2
	messages := []domain_models.Message{
		newMessage("1", time.Now()),
		newMessage("2", time.Now()),
	}

	roomRepo.On("GetRoom", ctx, roomID).Return(newRoom(false, false), nil)
	repo.On("GetMessages", ctx, roomID, mock.Anything, limit+1).Return(messages, nil)

	result, err := svc.GetMessages(ctx, roomID, userID, nil, limit)

	assert.NoError(t, err)
	assert.False(t, result.HasMore)
	assert.Len(t, result.Messages, limit)
	assert.Nil(t, result.NextCursor)
}

func TestGetMessages_WithBefore(t *testing.T) {
	svc, repo, roomRepo := newService()
	ctx := context.Background()
	roomID := "room-123"
	userID := "user-456"
	limit := 5
	before := time.Now().Add(-24 * time.Hour)
	expectedMessages := []domain_models.Message{
		newMessage("1", before.Add(-2*time.Hour)),
	}

	roomRepo.On("GetRoom", ctx, roomID).Return(newRoom(false, false), nil)
	repo.On("GetMessages", ctx, roomID, &before, limit+1).Return(expectedMessages, nil)

	result, err := svc.GetMessages(ctx, roomID, userID, &before, limit)

	assert.NoError(t, err)
	assert.Equal(t, expectedMessages, result.Messages)
	assert.False(t, result.HasMore)
	assert.Nil(t, result.NextCursor)
	repo.AssertCalled(t, "GetMessages", ctx, roomID, &before, limit+1)
}
