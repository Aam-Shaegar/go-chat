package domain_dtos

import (
	domain_models "go-chat/internal/core/domain/models"
	"time"
)

type UserResponseDTO struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

func NewUserResponseDTO(id string, username string, createdAt time.Time) UserResponseDTO {
	return UserResponseDTO{
		ID:        id,
		Username:  username,
		CreatedAt: createdAt,
	}
}

func UserDTOFromDomain(user domain_models.User) UserResponseDTO {
	return NewUserResponseDTO(user.ID, user.Username, user.CreatedAt)
}

func UsersDTOFromDomains(users []domain_models.User) []UserResponseDTO {
	DTOs := make([]UserResponseDTO, len(users))
	for i, user := range users {
		DTOs[i] = UserDTOFromDomain(user)
	}
	return DTOs
}
