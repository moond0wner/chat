package history_postgres_repository

import core_postgres_pool "tcp_srv/internal/core/repository/postgres/pool"

type HistoryRepository struct {
	pool core_postgres_pool.Pool
}

func NewHistoryRepository(pool core_postgres_pool.Pool) *HistoryRepository {
	return &HistoryRepository{
		pool: pool,
	}
}
