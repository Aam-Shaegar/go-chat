package domain_models

import "time"

type Message struct {
	ID          string     `json:"id"`
	RoomID      string     `json:"room_id"`
	UserID      string     `json:"user_id"`
	ReplyToID   *string    `json:"reply_to_id,omitempty"`
	Content     string     `json:"content"`
	IsEncrypted bool       `json:"is_encrypted"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

func NewMessage(id, roomID, userID string, replyToID *string, content string, isEncrypted bool, createdAt, updatedAt time.Time) Message {
	return Message{
		ID:          id,
		RoomID:      roomID,
		UserID:      userID,
		ReplyToID:   replyToID,
		Content:     content,
		IsEncrypted: isEncrypted,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func (m *Message) Edit(content string, updatedAt time.Time) {
	m.Content = content
	m.UpdatedAt = updatedAt
}

func (m *Message) Delete(deletedAt time.Time) {
	m.DeletedAt = &deletedAt
}

func (m Message) IsDeleted() bool { return m.DeletedAt != nil }

type MessageReaction struct {
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}
