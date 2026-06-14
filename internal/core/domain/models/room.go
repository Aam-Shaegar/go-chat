package domain_models

import "time"

type Room struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsPrivate   bool      `json:"is_private"`
	IsDM        bool      `json:"is_dm"`
	OwnerID     string    `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewRoom(id, name, description string, isPrivate, isDM bool, ownerID string, createdAt time.Time) Room {
	return Room{
		ID:          id,
		Name:        name,
		Description: description,
		IsPrivate:   isPrivate,
		IsDM:        isDM,
		OwnerID:     ownerID,
		CreatedAt:   createdAt,
	}
}
