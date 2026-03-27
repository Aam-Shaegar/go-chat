package ws

import "go-chat/internal/domain"

const (
	TypeSendMessage = "send_message"
	TypeTyping      = "typing"

	TypeNewMessage = "new_message"
	TypeUserTyping = "user_typing"
	TypeUserJoined = "user_joined"
	TypeUserLeft   = "user_left"
	TypeError      = "error"
)

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
