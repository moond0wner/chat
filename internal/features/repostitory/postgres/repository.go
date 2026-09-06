package history_postgres_repository

import core_postgres_pool "github.com/moond0wner/chat/internal/core/repository/postgres/pool"

type HistoryRepository struct {
	pool core_postgres_pool.Pool
}

func NewHistoryRepository(pool core_postgres_pool.Pool) *HistoryRepository {
	return &HistoryRepository{
		pool: pool,
	}
}
