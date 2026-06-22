package domain_dtos

import domain_models "go-chat/internal/core/domain/models"

type AuthResponseDTO struct {
	User        domain_models.User `json:"user"`
	AccessToken string             `json:"access_token"`
}

func NewAuthResponseDTO(user domain_models.User, accessToken string) AuthResponseDTO {
	return AuthResponseDTO{
		User:        user,
		AccessToken: accessToken,
	}
}
