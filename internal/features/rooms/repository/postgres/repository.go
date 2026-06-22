package rooms_repository_postgres

import core_postgres_pool "go-chat/internal/core/repository/postgres/pool"

type RoomsRepository struct {
	pool core_postgres_pool.Pool
}

func NewRoomsRepository(pool core_postgres_pool.Pool) *RoomsRepository {
	return &RoomsRepository{pool: pool}
}
