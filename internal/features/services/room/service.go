package room_service

import (
	"sync"
	core_domain "github.com/moond0wner/chat/internal/core/domain"
	core_logger "github.com/moond0wner/chat/internal/core/logger"

	history_postgres_repository "github.com/moond0wner/chat/internal/features/repostitory/postgres"
)

type RoomService struct {
	rooms             map[int]*core_domain.Room
	roomsByName       map[string]int
	historyRepository *history_postgres_repository.HistoryRepository
	log               *core_logger.Logger
	mtx               sync.RWMutex

	RegisterRoomID int
	GeneralRoomID  int
}

func NewRoomService(log *core_logger.Logger, hs *history_postgres_repository.HistoryRepository) *RoomService {
	return &RoomService{
		rooms:             make(map[int]*core_domain.Room),
		roomsByName:       make(map[string]int),
		historyRepository: hs,
		log:               log,
	}
}
