package ws_service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	ws_client "go-chat/internal/features/ws/client"
	ws_domain "go-chat/internal/features/ws/domain"
	ws_service "go-chat/internal/features/ws/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// --- Моки ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SaveMessage(ctx context.Context, msg domain_models.Message) (domain_models.Message, error) {
	args := m.Called(ctx, msg)
	return args.Get(0).(domain_models.Message), args.Error(1)
}

func (m *MockRepository) EditMessage(ctx context.Context, messageID, userID, content string, updatedAt time.Time) (domain_models.Message, error) {
	args := m.Called(ctx, messageID, userID, content, updatedAt)
	return args.Get(0).(domain_models.Message), args.Error(1)
}

func (m *MockRepository) DeleteMessage(ctx context.Context, messageID, userID string, deletedAt time.Time) error {
	args := m.Called(ctx, messageID, userID, deletedAt)
	return args.Error(0)
}

func (m *MockRepository) AddReaction(ctx context.Context, reaction domain_models.MessageReaction) error {
	args := m.Called(ctx, reaction)
	return args.Error(0)
}

func (m *MockRepository) RemoveReaction(ctx context.Context, messageID, userID, emoji string) error {
	args := m.Called(ctx, messageID, userID, emoji)
	return args.Error(0)
}

type MockHub struct {
	mock.Mock
}

func (m *MockHub) Publish(ctx context.Context, roomID string, event ws_domain.OutgoingEvent) error {
	args := m.Called(ctx, roomID, event)
	return args.Error(0)
}

func (m *MockHub) Unregister(client *ws_client.Client) {
	m.Called(client)
}

// --- Хелперы ---

type testLogger struct{}

func (l testLogger) Error(msg string, fields ...zap.Field) {}
func (l testLogger) Warn(msg string, fields ...zap.Field)  {}

func newTestClient(userID, username, roomID string) *ws_client.Client {
	return ws_client.NewClient(userID, username, roomID, nil, testLogger{})
}

func newTestMessage(id, roomID, userID string, content string, replyToID *string) domain_models.Message {
	return domain_models.NewMessage(id, roomID, userID, replyToID, content, false, time.Now(), time.Now())
}

func newService() (*ws_service.WSService, *MockRepository, *MockHub) {
	repo := new(MockRepository)
	hub := new(MockHub)
	svc := ws_service.NewWSService(repo, hub)
	return svc, repo, hub
}

func readFromSend(t *testing.T, client *ws_client.Client) ws_domain.OutgoingEvent {
	select {
	case event := <-client.SendChan():
		return event
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for event")
		return ws_domain.OutgoingEvent{}
	}
}

// --- Тесты (успешные сценарии проверяют только вызовы моков, без чтения из канала) ---

func TestHandle_SendMessage_Success(t *testing.T) {
	svc, repo, hub := newService()
	client := newTestClient("user-1", "alice", "room-1")

	payload := ws_domain.SendMessagePayload{
		Content:   "hello",
		ReplyToID: nil,
	}
	raw, _ := json.Marshal(payload)
	event := ws_domain.IncomingEvent{
		Type:    ws_domain.EventTypeSendMessage,
		Payload: raw,
	}

	savedMsg := newTestMessage("msg-1", "room-1", "user-1", "hello", nil)
	repo.On("SaveMessage", mock.Anything, mock.Anything).Return(savedMsg, nil)
	hub.On("Publish", mock.Anything, "room-1", mock.MatchedBy(func(e ws_domain.OutgoingEvent) bool {
		return e.Type == ws_domain.EventTypeNewMessage &&
			e.Payload.(ws_domain.NewMessagePayload).ID == "msg-1" &&
			e.Payload.(ws_domain.NewMessagePayload).RoomID == "room-1" &&
			e.Payload.(ws_domain.NewMessagePayload).UserID == "user-1" &&
			e.Payload.(ws_domain.NewMessagePayload).Username == "alice" &&
			e.Payload.(ws_domain.NewMessagePayload).Content == "hello"
	})).Return(nil)

	svc.Handle(client, event)

	repo.AssertExpectations(t)
	hub.AssertExpectations(t)
}

func TestHandle_SendMessage_InvalidPayload(t *testing.T) {
	svc, _, _ := newService()
	client := newTestClient("user-1", "alice", "room-1")

	event := ws_domain.IncomingEvent{
		Type:    ws_domain.EventTypeSendMessage,
		Payload: []byte(`{invalid json}`),
	}

	svc.Handle(client, event)

	outEvent := readFromSend(t, client)
	assert.Equal(t, ws_domain.EventTypeError, outEvent.Type)
	errPayload, ok := outEvent.Payload.(ws_domain.ErrorPayload)
	assert.True(t, ok)
	assert.Contains(t, errPayload.Message, "invalid payload")
}

func TestHandle_SendMessage_EmptyContent(t *testing.T) {
	svc, _, _ := newService()
	client := newTestClient("user-1", "alice", "room-1")

	payload := ws_domain.SendMessagePayload{Content: ""}
	raw, _ := json.Marshal(payload)
	event := ws_domain.IncomingEvent{
		Type:    ws_domain.EventTypeSendMessage,
		Payload: raw,
	}

	svc.Handle(client, event)

	outEvent := readFromSend(t, client)
	assert.Equal(t, ws_domain.EventTypeError, outEvent.Type)
	errPayload, ok := outEvent.Payload.(ws_domain.ErrorPayload)
	assert.True(t, ok)
	assert.Contains(t, errPayload.Message, "content is required")
}

func TestHandle_SendMessage_RepoError(t *testing.T) {
	svc, repo, _ := newService()
	client := newTestClient("user-1", "alice", "room-1")

	payload := ws_domain.SendMessagePayload{Content: "hello"}
	raw, _ := json.Marshal(payload)
	event := ws_domain.IncomingEvent{
		Type:    ws_domain.EventTypeSendMessage,
		Payload: raw,
	}

	repo.On("SaveMessage", mock.Anything, mock.Anything).Return(domain_models.Message{}, errors.New("db error"))

	svc.Handle(client, event)

	outEvent := readFromSend(t, client)
	assert.Equal(t, ws_domain.EventTypeError, outEvent.Type)
	errPayload, ok := outEvent.Payload.(ws_domain.ErrorPayload)
	assert.True(t, ok)
	assert.Contains(t, errPayload.Message, "save message")
}

func TestHandle_EditMessage_Success(t *testing.T) {
	svc, repo, hub := newService()
	client := newTestClient("user-1", "alice", "room-1")

	payload := ws_domain.EditMessagePayload{
		MessageID: "msg-1",
		Content:   "edited",
	}
	raw, _ := json.Marshal(payload)
	event := ws_domain.IncomingEvent{
		Type:    ws_domain.EventTypeEditMessage,
		Payload: raw,
	}

	updatedMsg := newTestMessage("msg-1", "room-1", "user-1", "edited", nil)
	repo.On("EditMessage", mock.Anything, "msg-1", "user-1", "edited", mock.Anything).Return(updatedMsg, nil)
	hub.On("Publish", mock.Anything, "room-1", mock.MatchedBy(func(e ws_domain.OutgoingEvent) bool {
		return e.Type == ws_domain.EventTypeMessageEdited &&
			e.Payload.(ws_domain.MessageEditedPayload).MessageID == "msg-1" &&
			e.Payload.(ws_domain.MessageEditedPayload).RoomID == "room-1" &&
			e.Payload.(ws_domain.MessageEditedPayload).Content == "edited"
	})).Return(nil)

	svc.Handle(client, event)

	repo.AssertExpectations(t)
	hub.AssertExpectations(t)
}

func TestHandle_EditMessage_InvalidPayload(t *testing.T) {
	svc, _, _ := newService()
	client := newTestClient("user-1", "alice", "room-1")

	event := ws_domain.IncomingEvent{
		Type:    ws_domain.EventTypeEditMessage,
		Payload: []byte(`{invalid}`),
	}

	svc.Handle(client, event)

	outEvent := readFromSend(t, client)
	assert.Equal(t, ws_domain.EventTypeError, outEvent.Type)
	errPayload, ok := outEvent.Payload.(ws_domain.ErrorPayload)
	assert.True(t, ok)
	assert.Contains(t, errPayload.Message, "invalid payload")
}

func TestHandle_DeleteMessage_Success(t *testing.T) {
	svc, repo, hub := newService()
	client := newTestClient("user-1", "alice", "room-1")

	payload := ws_domain.DeleteMessagePayload{MessageID: "msg-1"}
	raw, _ := json.Marshal(payload)
	event := ws_domain.IncomingEvent{
		Type:    ws_domain.EventTypeDeleteMessage,
		Payload: raw,
	}

	repo.On("DeleteMessage", mock.Anything, "msg-1", "user-1", mock.Anything).Return(nil)
	hub.On("Publish", mock.Anything, "room-1", mock.MatchedBy(func(e ws_domain.OutgoingEvent) bool {
		return e.Type == ws_domain.EventTypeMessageDeleted &&
			e.Payload.(ws_domain.MessageDeletedPayload).MessageID == "msg-1" &&
			e.Payload.(ws_domain.MessageDeletedPayload).RoomID == "room-1"
	})).Return(nil)

	svc.Handle(client, event)

	repo.AssertExpectations(t)
	hub.AssertExpectations(t)
}

func TestHandle_DeleteMessage_InvalidPayload(t *testing.T) {
	svc, _, _ := newService()
	client := newTestClient("user-1", "alice", "room-1")

	event := ws_domain.IncomingEvent{
		Type:    ws_domain.EventTypeDeleteMessage,
		Payload: []byte(`{invalid}`),
	}

	svc.Handle(client, event)

	outEvent := readFromSend(t, client)
	assert.Equal(t, ws_domain.EventTypeError, outEvent.Type)
	errPayload, ok := outEvent.Payload.(ws_domain.ErrorPayload)
	assert.True(t, ok)
	assert.Contains(t, errPayload.Message, "invalid payload")
}

func TestHandle_AddReaction_Success(t *testing.T) {
	svc, repo, hub := newService()
	client := newTestClient("user-1", "alice", "room-1")

	payload := ws_domain.AddReactionPayload{
		MessageID: "msg-1",
		Emoji:     "👍",
	}
	raw, _ := json.Marshal(payload)
	event := ws_domain.IncomingEvent{
		Type:    ws_domain.EventTypeAddReaction,
		Payload: raw,
	}

	repo.On("AddReaction", mock.Anything, mock.Anything).Return(nil)
	hub.On("Publish", mock.Anything, "room-1", mock.MatchedBy(func(e ws_domain.OutgoingEvent) bool {
		return e.Type == ws_domain.EventTypeReactionAdded &&
			e.Payload.(ws_domain.ReactionPayload).MessageID == "msg-1" &&
			e.Payload.(ws_domain.ReactionPayload).RoomID == "room-1" &&
			e.Payload.(ws_domain.ReactionPayload).UserID == "user-1" &&
			e.Payload.(ws_domain.ReactionPayload).Emoji == "👍"
	})).Return(nil)

	svc.Handle(client, event)

	repo.AssertExpectations(t)
	hub.AssertExpectations(t)
}

func TestHandle_RemoveReaction_Success(t *testing.T) {
	svc, repo, hub := newService()
	client := newTestClient("user-1", "alice", "room-1")

	payload := ws_domain.RemoveReactionPayload{
		MessageID: "msg-1",
		Emoji:     "👍",
	}
	raw, _ := json.Marshal(payload)
	event := ws_domain.IncomingEvent{
		Type:    ws_domain.EventTypeRemoveReaction,
		Payload: raw,
	}

	repo.On("RemoveReaction", mock.Anything, "msg-1", "user-1", "👍").Return(nil)
	hub.On("Publish", mock.Anything, "room-1", mock.MatchedBy(func(e ws_domain.OutgoingEvent) bool {
		return e.Type == ws_domain.EventTypeReactionRemoved &&
			e.Payload.(ws_domain.ReactionPayload).MessageID == "msg-1" &&
			e.Payload.(ws_domain.ReactionPayload).RoomID == "room-1" &&
			e.Payload.(ws_domain.ReactionPayload).UserID == "user-1" &&
			e.Payload.(ws_domain.ReactionPayload).Emoji == "👍"
	})).Return(nil)

	svc.Handle(client, event)

	repo.AssertExpectations(t)
	hub.AssertExpectations(t)
}

func TestHandle_Typing_Success(t *testing.T) {
	svc, _, hub := newService()
	client := newTestClient("user-1", "alice", "room-1")

	payload := ws_domain.TypingPayload{RoomID: "room-1"}
	raw, _ := json.Marshal(payload)
	event := ws_domain.IncomingEvent{
		Type:    ws_domain.EventTypeTyping,
		Payload: raw,
	}

	hub.On("Publish", mock.Anything, "room-1", mock.MatchedBy(func(e ws_domain.OutgoingEvent) bool {
		return e.Type == ws_domain.EventTypeUserTyping &&
			e.Payload.(ws_domain.UserTypingPayload).RoomID == "room-1" &&
			e.Payload.(ws_domain.UserTypingPayload).UserID == "user-1" &&
			e.Payload.(ws_domain.UserTypingPayload).Username == "alice"
	})).Return(nil)

	svc.Handle(client, event)

	hub.AssertExpectations(t)
}

func TestHandle_UnknownEventType(t *testing.T) {
	svc, _, _ := newService()
	client := newTestClient("user-1", "alice", "room-1")

	event := ws_domain.IncomingEvent{
		Type:    "unknown",
		Payload: []byte(`{}`),
	}

	svc.Handle(client, event)

	outEvent := readFromSend(t, client)
	assert.Equal(t, ws_domain.EventTypeError, outEvent.Type)
	errPayload, ok := outEvent.Payload.(ws_domain.ErrorPayload)
	assert.True(t, ok)
	assert.Contains(t, errPayload.Message, "unknown event type")
}

func TestOnClose(t *testing.T) {
	svc, _, hub := newService()
	client := newTestClient("user-1", "alice", "room-1")

	hub.On("Unregister", client).Return()
	hub.On("Publish", mock.Anything, "room-1", mock.Anything).Return(nil)

	svc.OnClose(client)

	hub.AssertExpectations(t)
}
