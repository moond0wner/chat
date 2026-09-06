package room_service

import (
	"context"
	"fmt"
	core_domain "github.com/moond0wner/chat/internal/core/domain"

	"go.uber.org/zap"
)

func (rs *RoomService) InitSystemRooms(ctx context.Context) error {
	generalID, err := rs.historyRepository.GetIDRoomByName(ctx, "general")
	if err != nil {
		return fmt.Errorf("failed to get general room id: %w", err)
	}
	rs.GeneralRoomID = generalID

	registerID, err := rs.historyRepository.GetIDRoomByName(ctx, "register")
	if err != nil {
		return fmt.Errorf("failed to get register room id: %w", err)
	}
	rs.RegisterRoomID = registerID

	return nil
}

func (rs *RoomService) LoadAllRooms(ctx context.Context) error {
	rooms, err := rs.historyRepository.GetAllRooms(ctx)
	if err != nil {
		return fmt.Errorf("failed to load rooms from db: %w", err)
	}

	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	for _, room := range *rooms {
		rs.rooms[room.ID] = &room
		rs.roomsByName[room.Name] = room.ID
		room.Clients = make(map[int]*core_domain.Client)
	}

	rs.log.Info("Rooms loaded from DB", zap.Int("count", len(*rooms)))
	return nil
}
