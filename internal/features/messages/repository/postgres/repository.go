package messages_repository_postgres

import core_postgres_pool "go-chat/internal/core/repository/postgres/pool"

type MessagesRepository struct {
	pool core_postgres_pool.Pool
}

func NewMessagesRepository(pool core_postgres_pool.Pool) *MessagesRepository {
	return &MessagesRepository{pool: pool}
}
