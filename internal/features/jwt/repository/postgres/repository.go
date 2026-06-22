package jwt_repository_postgres

import core_postgres_pool "go-chat/internal/core/repository/postgres/pool"

type JwtRepository struct {
	pool core_postgres_pool.Pool
}

func NewJwtRepository(pool core_postgres_pool.Pool) *JwtRepository {
	return &JwtRepository{
		pool: pool,
	}
}
