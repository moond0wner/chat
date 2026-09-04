package client_service

import (
	"sync"
	core_domain "tcp_srv/internal/core/domain"
	core_logger "tcp_srv/internal/core/logger"
	history_postgres_repository "tcp_srv/internal/features/repostitory/postgres"
)

type ClientService struct {
	clients           map[int]*core_domain.Client
	nameMap           map[string]*core_domain.Client
	historyRepository *history_postgres_repository.HistoryRepository
	log               *core_logger.Logger
	mtx               sync.RWMutex
}

func NewClientService(logger *core_logger.Logger, hs *history_postgres_repository.HistoryRepository) *ClientService {
	return &ClientService{
		clients:           make(map[int]*core_domain.Client),
		nameMap:           make(map[string]*core_domain.Client),
		historyRepository: hs,
		log:               logger,
	}
}
