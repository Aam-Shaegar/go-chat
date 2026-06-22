package dm_repository_postgres

import core_postgres_pool "go-chat/internal/core/repository/postgres/pool"

type DMRepository struct {
	pool core_postgres_pool.Pool
}

func NewDMRepository(pool core_postgres_pool.Pool) *DMRepository {
	return &DMRepository{pool: pool}
}
