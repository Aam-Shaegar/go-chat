package reads_repository_postgres

import core_postgres_pool "go-chat/internal/core/repository/postgres/pool"

type ReadsRepository struct {
	pool core_postgres_pool.Pool
}

func NewReadsRepository(pool core_postgres_pool.Pool) *ReadsRepository {
	return &ReadsRepository{pool: pool}
}
