package users_repository_postgres

import (
	domain_models "go-chat/internal/core/domain/models"
	"time"
)

type UserModel struct {
	ID        string
	Username  string
	Email     string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func userDomainsFromModels(users []UserModel) []domain_models.User {
	userDomains := make([]domain_models.User, len(users))
	for i, user := range users {
		userDomains[i] = domain_models.NewUser(
			user.ID,
			user.Username,
			user.Email,
			user.Password,
			user.CreatedAt,
			user.UpdatedAt,
		)
	}
	return userDomains
}
