package room_service

import (
	"context"
	"errors"
	"fmt"
	core_domain "github.com/moond0wner/chat/internal/core/domain"

	"go.uber.org/zap"
)

func (rs *RoomService) GetRoomByName(ctx context.Context, name string) (*core_domain.Room, error) {
	rs.mtx.RLock()
	id, ok := rs.roomsByName[name]
	rs.mtx.RUnlock()

	if ok {
		rs.mtx.RLock()
		room := rs.rooms[id]
		rs.mtx.RUnlock()
		return room, nil
	}

	id, err := rs.historyRepository.GetIDRoomByName(ctx, name)
	if err != nil {
		rs.log.Warn("Error get room from DB", zap.String("room_name", name), zap.Error(err))
		return nil, fmt.Errorf("Error get room from DB: %v", err)
	}

	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	if _, ok = rs.roomsByName[name]; ok {
		return rs.rooms[id], nil
	}

	rs.roomsByName[name] = id
	room := rs.rooms[id]
	return room, nil
}

func (rs *RoomService) GetRoomByID(ctx context.Context, id int) (*core_domain.Room, error) {
	rs.mtx.RLock()
	room, ok := rs.rooms[id]
	rs.mtx.RUnlock()

	if ok {
		return room, nil
	} else {
		return nil, errors.New("Комната не найдена")
	}
}
