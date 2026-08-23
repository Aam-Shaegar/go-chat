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
	EditMessage(ctx context.Context, messageID, roomID, userID, content string, updatedAt time.Time) (domain_models.Message, error)
	DeleteMessage(ctx context.Context, messageID, roomID, userID string, deletedAt time.Time) (domain_models.Message, error)
	AddReaction(ctx context.Context, roomID string, reaction domain_models.RawMessageReaction) (domain_models.Message, error)
	RemoveReaction(ctx context.Context, messageID, roomID, userID string) (domain_models.Message, error)
	GetRoomMemberIDs(ctx context.Context, roomID string) ([]string, error)
	GetMember(ctx context.Context, roomID, userID string) (domain_models.RoomMember, error)
}

type Hub interface {
	Publish(ctx context.Context, roomID string, event ws_domain.OutgoingEvent) error
	PublishToUser(ctx context.Context, userID string, event ws_domain.OutgoingEvent) error
	Unregister(client *ws_client.Client)
}

type ServiceInterface interface {
	Handle(client *ws_client.Client, event ws_domain.IncomingEvent)
	OnClose(client *ws_client.Client)
	OnConnect(client *ws_client.Client)
	PublishUserMuted(ctx context.Context, roomID, targetUserID, targetUsername, mutedBy, mutedByName string, mutedUntil time.Time) error
	PublishUserUnmuted(ctx context.Context, roomID, targetUserID, targetUsername, unmutedBy, unmutedByName string) error
	PublishUserKicked(ctx context.Context, roomID, targetUserID, targetUsername, kickedBy, kickedByName string) error
	PublishInviteDeactivated(ctx context.Context, roomID, token string) error
	PublishInviteUsed(ctx context.Context, roomID, token string, uses, maxUses int) error
}

func (s *WSService) OnConnect(client *ws_client.Client) {
	if client.RoomID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.publishRoomEvent(ctx, client.RoomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeUserJoined,
		Payload: ws_domain.UserJoinedPayload{
			RoomID:   client.RoomID,
			UserID:   client.ID,
			Username: client.Username,
		},
	}, map[string]struct{}{client.ID: {}})
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
		err = s.handleTyping(ctx, client, event.Payload)
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
	roomID, err := resolveRoomID(client, p.RoomID)
	if err != nil {
		return err
	}

	// Check if user is muted in this room
	member, err := s.repo.GetMember(ctx, roomID, client.ID)
	if err != nil {
		return fmt.Errorf("get member: %w", err)
	}
	if member.IsMuted() {
		return fmt.Errorf("you are muted in this room: %w", core_error.ErrUnauthorized)
	}

	now := time.Now()
	msg, err := s.repo.SaveMessage(ctx, domain_models.NewMessage(
		"", roomID, client.ID,
		p.ReplyToID, p.Content, false,
		now, now,
	))
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	event := ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeNewMessage,
		Payload: ws_domain.NewMessagePayload{
			ID:        msg.ID,
			RoomID:    msg.RoomID,
			UserID:    msg.UserID,
			Username:  displayUsername(msg.Username, client.Username),
			ReplyToID: msg.ReplyToID,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		},
	}
	return s.publishRoomEvent(ctx, msg.RoomID, event, nil)
}

func (s *WSService) handleEditMessage(ctx context.Context, client *ws_client.Client, raw []byte) error {
	var p ws_domain.EditMessagePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("invalid payload: %w", core_error.ErrInvalidArgument)
	}
	if p.MessageID == "" || p.Content == "" {
		return fmt.Errorf("message_id and content are required: %w", core_error.ErrInvalidArgument)
	}
	roomID, err := resolveRoomID(client, p.RoomID)
	if err != nil {
		return err
	}

	if err := s.checkMuted(ctx, roomID, client.ID); err != nil {
		return err
	}

	msg, err := s.repo.EditMessage(ctx, p.MessageID, roomID, client.ID, p.Content, time.Now())
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}

	return s.publishRoomEvent(ctx, msg.RoomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeMessageEdited,
		Payload: ws_domain.MessageEditedPayload{
			MessageID: msg.ID,
			RoomID:    msg.RoomID,
			Content:   msg.Content,
			UpdatedAt: msg.UpdatedAt,
		},
	}, nil)
}

func (s *WSService) handleDeleteMessage(ctx context.Context, client *ws_client.Client, raw []byte) error {
	var p ws_domain.DeleteMessagePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("invalid payload: %w", core_error.ErrInvalidArgument)
	}
	if p.MessageID == "" {
		return fmt.Errorf("message_id is required: %w", core_error.ErrInvalidArgument)
	}
	roomID, err := resolveRoomID(client, p.RoomID)
	if err != nil {
		return err
	}

	if err := s.checkMuted(ctx, roomID, client.ID); err != nil {
		return err
	}

	now := time.Now()
	msg, err := s.repo.DeleteMessage(ctx, p.MessageID, roomID, client.ID, now)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}

	return s.publishRoomEvent(ctx, msg.RoomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeMessageDeleted,
		Payload: ws_domain.MessageDeletedPayload{
			MessageID: p.MessageID,
			RoomID:    msg.RoomID,
			DeletedAt: now,
		},
	}, nil)
}

func (s *WSService) handleAddReaction(ctx context.Context, client *ws_client.Client, raw []byte) error {
	var p ws_domain.AddReactionPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("invalid payload: %w", core_error.ErrInvalidArgument)
	}
	if p.MessageID == "" || p.Emoji == "" {
		return fmt.Errorf("message_id and emoji are required: %w", core_error.ErrInvalidArgument)
	}
	roomID, err := resolveRoomID(client, p.RoomID)
	if err != nil {
		return err
	}

	if err := s.checkMuted(ctx, roomID, client.ID); err != nil {
		return err
	}

	msg, err := s.repo.AddReaction(ctx, roomID, domain_models.RawMessageReaction{
		MessageID: p.MessageID,
		UserID:    client.ID,
		Emoji:     p.Emoji,
		CreatedAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("add reaction: %w", err)
	}

	return s.publishRoomEvent(ctx, msg.RoomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeReactionAdded,
		Payload: ws_domain.ReactionPayload{
			MessageID:       p.MessageID,
			RoomID:          msg.RoomID,
			UserID:          client.ID,
			Emoji:           p.Emoji,
			IsReactedByMe:   true,
		},
	}, nil)
}

func (s *WSService) handleRemoveReaction(ctx context.Context, client *ws_client.Client, raw []byte) error {
	var p ws_domain.RemoveReactionPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("invalid payload: %w", core_error.ErrInvalidArgument)
	}
	if p.MessageID == "" || p.Emoji == "" {
		return fmt.Errorf("message_id and emoji are required: %w", core_error.ErrInvalidArgument)
	}
	roomID, err := resolveRoomID(client, p.RoomID)
	if err != nil {
		return err
	}

	if err := s.checkMuted(ctx, roomID, client.ID); err != nil {
		return err
	}

	msg, err := s.repo.RemoveReaction(ctx, p.MessageID, roomID, client.ID)
	if err != nil {
		return fmt.Errorf("remove reaction: %w", err)
	}

	return s.publishRoomEvent(ctx, msg.RoomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeReactionRemoved,
		Payload: ws_domain.ReactionPayload{
			MessageID:       p.MessageID,
			RoomID:          msg.RoomID,
			UserID:          client.ID,
			Emoji:           p.Emoji,
			IsReactedByMe:   false,
		},
	}, nil)
}

func (s *WSService) handleTyping(ctx context.Context, client *ws_client.Client, raw []byte) error {
	var p ws_domain.TypingPayload
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("invalid payload: %w", core_error.ErrInvalidArgument)
		}
	}
	roomID, err := resolveRoomID(client, p.RoomID)
	if err != nil {
		return err
	}

	return s.publishRoomEvent(ctx, roomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeUserTyping,
		Payload: ws_domain.UserTypingPayload{
			RoomID:   roomID,
			UserID:   client.ID,
			Username: client.Username,
		},
	}, map[string]struct{}{client.ID: {}})
}

// OnClose вызывается когда клиент отключается
func (s *WSService) OnClose(client *ws_client.Client) {
	s.hub.Unregister(client)
	if client.RoomID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.publishRoomEvent(ctx, client.RoomID, ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeUserLeft,
		Payload: ws_domain.UserLeftPayload{
			RoomID:   client.RoomID,
			UserID:   client.ID,
			Username: client.Username,
		},
	}, map[string]struct{}{client.ID: {}})
}

func resolveRoomID(client *ws_client.Client, payloadRoomID string) (string, error) {
	if client.RoomID != "" {
		if payloadRoomID != "" && payloadRoomID != client.RoomID {
			return "", fmt.Errorf("room_id does not match connection room: %w", core_error.ErrInvalidArgument)
		}
		return client.RoomID, nil
	}
	if payloadRoomID == "" {
		return "", fmt.Errorf("room_id is required: %w", core_error.ErrInvalidArgument)
	}
	return payloadRoomID, nil
}

func displayUsername(fromMessage, fallback string) string {
	if fromMessage != "" {
		return fromMessage
	}
	return fallback
}

func (s *WSService) publishRoomEvent(ctx context.Context, roomID string, event ws_domain.OutgoingEvent, exclude map[string]struct{}) error {
	if err := s.hub.Publish(ctx, roomID, event); err != nil {
		return err
	}

	userIDs, err := s.repo.GetRoomMemberIDs(ctx, roomID)
	if err != nil {
		return fmt.Errorf("get room members: %w", err)
	}
	for _, userID := range userIDs {
		if _, skip := exclude[userID]; skip {
			continue
		}
		if err := s.hub.PublishToUser(ctx, userID, event); err != nil {
			return fmt.Errorf("publish to user: %w", err)
		}
	}
	return nil
}

func (s *WSService) PublishUserMuted(ctx context.Context, roomID, targetUserID, targetUsername, mutedBy, mutedByName string, mutedUntil time.Time) error {
	event := ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeUserMuted,
		Payload: ws_domain.UserMutedPayload{
			RoomID:      roomID,
			UserID:      targetUserID,
			Username:    targetUsername,
			MutedUntil:  mutedUntil,
			MutedBy:     mutedBy,
			MutedByName: mutedByName,
		},
	}
	return s.publishRoomEvent(ctx, roomID, event, nil)
}

func (s *WSService) PublishUserUnmuted(ctx context.Context, roomID, targetUserID, targetUsername, unmutedBy, unmutedByName string) error {
	event := ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeUserUnmuted,
		Payload: ws_domain.UserUnmutedPayload{
			RoomID:        roomID,
			UserID:        targetUserID,
			Username:      targetUsername,
			UnmutedBy:     unmutedBy,
			UnmutedByName: unmutedByName,
		},
	}
	return s.publishRoomEvent(ctx, roomID, event, nil)
}

func (s *WSService) PublishUserKicked(ctx context.Context, roomID, targetUserID, targetUsername, kickedBy, kickedByName string) error {
	event := ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeUserLeft,
		Payload: ws_domain.UserLeftPayload{
			RoomID:        roomID,
			UserID:        targetUserID,
			Username:      targetUsername,
			KickedBy:      kickedBy,
			KickedByName:  kickedByName,
		},
	}
	// Publish to room channel (for other members)
	if err := s.hub.Publish(ctx, roomID, event); err != nil {
		return err
	}
	// Publish directly to kicked user's channel (they're no longer in room members)
	if err := s.hub.PublishToUser(ctx, targetUserID, event); err != nil {
		return fmt.Errorf("publish to kicked user: %w", err)
	}
	// Publish to other members via user channels
	userIDs, err := s.repo.GetRoomMemberIDs(ctx, roomID)
	if err != nil {
		return fmt.Errorf("get room members: %w", err)
	}
	for _, userID := range userIDs {
		if userID == targetUserID {
			continue // already sent above
		}
		if err := s.hub.PublishToUser(ctx, userID, event); err != nil {
			return fmt.Errorf("publish to user: %w", err)
		}
	}
	return nil
}

func (s *WSService) checkMuted(ctx context.Context, roomID, userID string) error {
	member, err := s.repo.GetMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("get member: %w", err)
	}
	if member.IsMuted() {
		return fmt.Errorf("you are muted in this room: %w", core_error.ErrUnauthorized)
	}
	return nil
}

func (s *WSService) PublishInviteDeactivated(ctx context.Context, roomID, token string) error {
	event := ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeInviteDeactivated,
		Payload: ws_domain.InviteDeactivatedPayload{
			RoomID: roomID,
			Token:  token,
		},
	}
	return s.publishRoomEvent(ctx, roomID, event, nil)
}

func (s *WSService) PublishInviteUsed(ctx context.Context, roomID, token string, uses, maxUses int) error {
	event := ws_domain.OutgoingEvent{
		Type: ws_domain.EventTypeInviteUsed,
		Payload: ws_domain.InviteUsedPayload{
			RoomID:  roomID,
			Token:   token,
			Uses:    uses,
			MaxUses: maxUses,
		},
	}
	return s.publishRoomEvent(ctx, roomID, event, nil)
}
