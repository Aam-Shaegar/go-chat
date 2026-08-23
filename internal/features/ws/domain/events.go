package ws_domain

import (
	"encoding/json"
	"time"
)

type EventType string

const (
	// Клиент -> Сервер
	EventTypeSendMessage    EventType = "send_message"
	EventTypeEditMessage    EventType = "edit_message"
	EventTypeDeleteMessage  EventType = "delete_message"
	EventTypeAddReaction    EventType = "add_reaction"
	EventTypeRemoveReaction EventType = "remove_reaction"
	EventTypeTyping         EventType = "typing"

// Сервер -> Клиент
	EventTypeNewMessage        EventType = "new_message"
	EventTypeMessageEdited     EventType = "message_edited"
	EventTypeMessageDeleted    EventType = "message_deleted"
	EventTypeReactionAdded     EventType = "reaction_added"
	EventTypeReactionRemoved   EventType = "reaction_removed"
	EventTypeUserTyping        EventType = "user_typing"
	EventTypeUserJoined        EventType = "user_joined"
	EventTypeUserLeft          EventType = "user_left"
	EventTypeUserMuted         EventType = "user_muted"
	EventTypeUserUnmuted       EventType = "user_unmuted"
	EventTypeInviteDeactivated EventType = "invite_deactivated"
	EventTypeInviteUsed        EventType = "invite_used"
	EventTypeError             EventType = "error"
)

// IncomingEvent - входящее событие от клиента
type IncomingEvent struct {
	Type    EventType       `json:"type"`
	Payload json.RawMessage `json:"payload"` // json.RawMessage а не []byte - не кодируется в base64
}

// OutgoingEvent - исходящее событие клиенту
type OutgoingEvent struct {
	Type    EventType `json:"type"`
	Payload any       `json:"payload"`
}

// Payloads клиент -> сервер

type SendMessagePayload struct {
	RoomID    string  `json:"room_id,omitempty"`
	Content   string  `json:"content"`
	ReplyToID *string `json:"reply_to_id,omitempty"`
}

type EditMessagePayload struct {
	RoomID    string `json:"room_id,omitempty"`
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
}

type DeleteMessagePayload struct {
	RoomID    string `json:"room_id,omitempty"`
	MessageID string `json:"message_id"`
}

type AddReactionPayload struct {
	RoomID    string `json:"room_id,omitempty"`
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

type RemoveReactionPayload struct {
	RoomID    string `json:"room_id,omitempty"`
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

type TypingPayload struct {
	RoomID string `json:"room_id"`
}

// Payloads сервер -> клиент

type NewMessagePayload struct {
	ID          string    `json:"id"`
	RoomID      string    `json:"room_id"`
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	ReplyToID   *string   `json:"reply_to_id,omitempty"`
	Content     string    `json:"content"`
	IsEncrypted bool      `json:"is_encrypted"`
	CreatedAt   time.Time `json:"created_at"`
}

type MessageEditedPayload struct {
	MessageID string    `json:"message_id"`
	RoomID    string    `json:"room_id"`
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MessageDeletedPayload struct {
	MessageID string    `json:"message_id"`
	RoomID    string    `json:"room_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

type ReactionPayload struct {
	MessageID       string `json:"message_id"`
	RoomID          string `json:"room_id"`
	UserID          string `json:"user_id"`
	Emoji           string `json:"emoji"`
	IsReactedByMe   bool   `json:"is_reacted_by_me"`
}

type UserTypingPayload struct {
	RoomID   string `json:"room_id"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type UserJoinedPayload struct {
	RoomID   string `json:"room_id"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

type UserMutedPayload struct {
	RoomID      string    `json:"room_id"`
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	MutedUntil  time.Time `json:"muted_until"`
	MutedBy     string    `json:"muted_by"`
	MutedByName string    `json:"muted_by_name"`
}

type UserUnmutedPayload struct {
	RoomID        string `json:"room_id"`
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	UnmutedBy     string `json:"unmuted_by"`
	UnmutedByName string `json:"unmuted_by_name"`
}

type UserLeftPayload struct {
	RoomID        string `json:"room_id"`
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	KickedBy      string `json:"kicked_by,omitempty"`
	KickedByName  string `json:"kicked_by_name,omitempty"`
}

type InviteDeactivatedPayload struct {
	RoomID string `json:"room_id"`
	Token  string `json:"token"`
}

type InviteUsedPayload struct {
	RoomID string `json:"room_id"`
	Token  string `json:"token"`
	Uses   int    `json:"uses"`
	MaxUses int   `json:"max_uses"`
}
