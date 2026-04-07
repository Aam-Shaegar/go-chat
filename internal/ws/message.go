package ws

import "go-chat/internal/domain"

const (
	TypeSendMessage = "send_message"
	TypeTyping      = "typing"
	TypeRoomStats   = "room_stats"

	TypeNewMessage     = "new_message"
	TypeUserTyping     = "user_typing"
	TypeUserJoined     = "user_joined"
	TypeUserLeft       = "user_left"
	TypeError          = "error"
	TypeMessageDeleted = "message_deleted"
)

type RoomStatsPayload struct {
	OnlineCount int `json:"online_count"`
}

type MessageDeletedPayload struct {
	MessageID string `json:"message_id"`
	RoomID    string `json:"room_id"`
}

type IncomingMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type OutgoingMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type TypingPayload struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type UserEventPayload struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type MessagePayload struct {
	Message  domain.Message `json:"message"`
	Username string         `json:"username"`
}
