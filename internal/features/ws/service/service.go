package ws_service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
	ws_client "go-chat/internal/features/ws/client"
	ws_domain "go-chat/internal/features/ws/domain"
)

const dbTimeout = 5 * time.Second

type WSService struct {
	repo Repository
	hub  Hub
}

func NewWSService(repo Repository, hub Hub) *WSService {
	return &WSService{repo: repo, hub: hub}
}

type Repository interface {
	SaveMessage(ctx context.Context, msg domain_models.Message) (domain_models.Message, error)
	EditMessage(ctx context.Context, messageID, userID, content string, updatedAt time.Time) (domain_models.Message, error)
	DeleteMessage(ctx context.Context, messageID, userID string, deletedAt time.Time) error
	AddReaction(ctx context.Context, reaction domain_models.MessageReaction) error
	RemoveReaction(ctx context.Context, messageID, userID, emoji string) error
}

type Hub interface {
	Publish(ctx context.Context, roomID string, event ws_domain.OutgoingEvent) error
	Unregister(client *ws_client.Client)
}

// Handle — точка входа для всех входящих событий от клиента
func (s *WSService) Handle(client *ws_client.Client, event ws_domain.IncomingEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var err error
	switch event.Type {
	case ws_domain.EventTypeSendMessage:
		err = s.handleSendMessage(ctx, client, event.Payload)
	case ws_domain.EventTypeEditMessage:
		err = s.handleEditMessage(ctx, client, event.Payload)
	case ws_domain.EventTypeDeleteMessage:
		err = s.handleDeleteMessage(ctx, client, event.Payload)
	case ws_domain.EventTypeAddReaction:
		err = s.handleAddReaction(ctx, client, event.Payload)
	case ws_domain.EventTypeRemoveReaction:
		err = s.handleRemoveReaction(ctx, client, event.Payload)
	case ws_domain.EventTypeTyping:
		err = s.handleTyping(ctx, client)
	default:
		client.SendEvent(ws_domain.OutgoingEvent{
			Type:    ws_domain.EventTypeError,
			Payload: ws_domain.ErrorPayload{Message: "unknown event type"},
		})
		return
	}

	if err != nil {
		client.SendEvent(ws_domain.OutgoingEvent{
			Type:    ws_domain.EventTypeError,
			Payload: ws_domain.ErrorPayload{Message: err.Error()},
		})
	}
}

func (s *WSService) handleSendMessage(ctx context.Context, client *ws_client.Client, raw []byte) error {
	var p ws_domain.SendMessagePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("invalid payload: %w", core_error.ErrInvalidArgument)
	}
	if p.Content == "" {
		return fmt.Errorf("content is required: %w", core_error.ErrInvalidArgument)
	}

	now := time.Now()
	msg, err := s.repo.SaveMessage(ctx, domain_models.NewMessage(
		"", client.RoomID, client.ID,
		p.ReplyToID, p.Content, false,
		now, now,
	))
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	return s.hub.Publish(ctx, client.RoomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeNewMessage,
		Payload: ws_domain.NewMessagePayload{
			ID:        msg.ID,
			RoomID:    msg.RoomID,
			UserID:    msg.UserID,
			Username:  client.Username,
			ReplyToID: msg.ReplyToID,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		},
	})
}

func (s *WSService) handleEditMessage(ctx context.Context, client *ws_client.Client, raw []byte) error {
	var p ws_domain.EditMessagePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("invalid payload: %w", core_error.ErrInvalidArgument)
	}
	if p.MessageID == "" || p.Content == "" {
		return fmt.Errorf("message_id and content are required: %w", core_error.ErrInvalidArgument)
	}

	msg, err := s.repo.EditMessage(ctx, p.MessageID, client.ID, p.Content, time.Now())
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}

	return s.hub.Publish(ctx, client.RoomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeMessageEdited,
		Payload: ws_domain.MessageEditedPayload{
			MessageID: msg.ID,
			RoomID:    msg.RoomID,
			Content:   msg.Content,
			UpdatedAt: msg.UpdatedAt,
		},
	})
}

func (s *WSService) handleDeleteMessage(ctx context.Context, client *ws_client.Client, raw []byte) error {
	var p ws_domain.DeleteMessagePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("invalid payload: %w", core_error.ErrInvalidArgument)
	}
	if p.MessageID == "" {
		return fmt.Errorf("message_id is required: %w", core_error.ErrInvalidArgument)
	}

	now := time.Now()
	if err := s.repo.DeleteMessage(ctx, p.MessageID, client.ID, now); err != nil {
		return fmt.Errorf("delete message: %w", err)
	}

	return s.hub.Publish(ctx, client.RoomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeMessageDeleted,
		Payload: ws_domain.MessageDeletedPayload{
			MessageID: p.MessageID,
			RoomID:    client.RoomID,
			DeletedAt: now,
		},
	})
}

func (s *WSService) handleAddReaction(ctx context.Context, client *ws_client.Client, raw []byte) error {
	var p ws_domain.AddReactionPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("invalid payload: %w", core_error.ErrInvalidArgument)
	}
	if p.MessageID == "" || p.Emoji == "" {
		return fmt.Errorf("message_id and emoji are required: %w", core_error.ErrInvalidArgument)
	}

	if err := s.repo.AddReaction(ctx, domain_models.MessageReaction{
		MessageID: p.MessageID,
		UserID:    client.ID,
		Emoji:     p.Emoji,
		CreatedAt: time.Now(),
	}); err != nil {
		return fmt.Errorf("add reaction: %w", err)
	}

	return s.hub.Publish(ctx, client.RoomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeReactionAdded,
		Payload: ws_domain.ReactionPayload{
			MessageID: p.MessageID,
			RoomID:    client.RoomID,
			UserID:    client.ID,
			Emoji:     p.Emoji,
		},
	})
}

func (s *WSService) handleRemoveReaction(ctx context.Context, client *ws_client.Client, raw []byte) error {
	var p ws_domain.RemoveReactionPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("invalid payload: %w", core_error.ErrInvalidArgument)
	}
	if p.MessageID == "" || p.Emoji == "" {
		return fmt.Errorf("message_id and emoji are required: %w", core_error.ErrInvalidArgument)
	}

	if err := s.repo.RemoveReaction(ctx, p.MessageID, client.ID, p.Emoji); err != nil {
		return fmt.Errorf("remove reaction: %w", err)
	}

	return s.hub.Publish(ctx, client.RoomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeReactionRemoved,
		Payload: ws_domain.ReactionPayload{
			MessageID: p.MessageID,
			RoomID:    client.RoomID,
			UserID:    client.ID,
			Emoji:     p.Emoji,
		},
	})
}

func (s *WSService) handleTyping(ctx context.Context, client *ws_client.Client) error {
	return s.hub.Publish(ctx, client.RoomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeUserTyping,
		Payload: ws_domain.UserTypingPayload{
			RoomID:   client.RoomID,
			UserID:   client.ID,
			Username: client.Username,
		},
	})
}

// OnClose вызывается когда клиент отключается
func (s *WSService) OnClose(client *ws_client.Client) {
	s.hub.Unregister(client)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.hub.Publish(ctx, client.RoomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeUserLeft,
		Payload: ws_domain.UserJoinedPayload{
			RoomID:   client.RoomID,
			UserID:   client.ID,
			Username: client.Username,
		},
	})
}
