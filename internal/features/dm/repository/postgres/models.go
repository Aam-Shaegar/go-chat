package dm_repository_postgres

import (
	domain_models "go-chat/internal/core/domain/models"
	"time"
)

type roomModel struct {
	ID          string
	Name        string
	Description string
	IsPrivate   bool
	IsDM        bool
	OwnerID     string
	CreatedAt   time.Time
}

func roomToDomain(m roomModel) domain_models.Room {
	return domain_models.NewRoom(
		m.ID, m.Name, m.Description,
		m.IsPrivate, m.IsDM, m.OwnerID, m.CreatedAt,
	)
}
