package room_service

import (
	"context"
	"fmt"
	core_domain "tcp_srv/internal/core/domain"

	"go.uber.org/zap"
)

func (rs *RoomService) CreateRoom(ctx context.Context, name string) error {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	if _, ok := rs.roomsByName[name]; ok {
		return nil
	}

	id, err := rs.historyRepository.GetIDRoomByName(ctx, name)
	if err == nil {
		room, err := rs.GetRoomByID(ctx, id)
		if err != nil {
			return fmt.Errorf("Error get room from db by id: %v", err)
		}
		rs.rooms[id] = room
		rs.roomsByName[name] = id
		return nil
	}
	room := core_domain.NewRoom(name)
	if err = rs.historyRepository.SaveRoom(ctx, room); err != nil {
		return fmt.Errorf("Error save room: %v", err)
	}
	rs.rooms[room.ID] = room
	rs.roomsByName[room.Name] = room.ID
	return nil
}

func (rs *RoomService) JoinRoom(ctx context.Context, client *core_domain.Client, newRoomName string) error {
	rs.mtx.RLock()
	currentRoom := rs.rooms[client.RoomID]
	rs.mtx.RUnlock()

	if currentRoom != nil && currentRoom.Name == newRoomName && client.RoomID != rs.RegisterRoomID {
		fmt.Fprintln(client.Conn, "Вы уже находитесь в этом канале")
		return nil
	}

	oldRoom, oldRoomClients, needDelete := rs.removeClientFromRoom(client)
	newRoom, newRoomClients, err := rs.addClientToRoom(ctx, client, newRoomName)
	if err != nil {
		if oldRoom != nil {
			if _, _, err = rs.addClientToRoom(ctx, client, oldRoom.Name); err != nil {
				rs.log.Error("Rollback failed: client lost in both rooms",
					zap.Int("client_id", client.ID),
					zap.String("old_room", oldRoom.Name),
					zap.String("new_room", newRoomName),
					zap.Error(err),
				)
				client.Conn.Close()
				return fmt.Errorf("Error rollback failed: %v", err)
			}
		}
		return fmt.Errorf("join room failed: %v", err)
	}

	if needDelete {
		rs.deleteRoom(oldRoom.ID)
		rs.log.Warn("Удалена пустая комната", zap.String("room", oldRoom.Name), zap.Int("room_id", oldRoom.ID))
	}

	if len(oldRoomClients) > 0 {
		msg := fmt.Sprintf("%s покинул комнату\n", client.Name)
		for _, c := range oldRoomClients {
			fmt.Fprintln(c.Conn, msg)
		}
		fmt.Fprintf(client.Conn, "Вы покинули комнату: %s\n", oldRoom.Name)
		rs.log.Info("Клиент покинул комнату",
			zap.String("client", client.Name),
			zap.String("room", oldRoom.Name))
	}

	if len(newRoomClients) > 0 {
		msg := fmt.Sprintf("%s подключился к комнате\n", client.Name)
		for _, c := range newRoomClients {
			fmt.Fprintln(c.Conn, msg)
		}
	}

	fmt.Fprintf(client.Conn, "Подключен к комнате: %s\n", newRoom.Name)
	rs.log.Info("Клиент подключился к комнате",
		zap.String("client", client.Name),
		zap.String("room", newRoom.Name))

	return nil
}

func (rs *RoomService) LeaveRoom(ctx context.Context, client *core_domain.Client) error {
	if client.RoomID == rs.GeneralRoomID {
		fmt.Fprintln(client.Conn, "Вы уже в general")
		return nil
	}
	if client.RoomID == rs.RegisterRoomID {
		client.Conn.Close()
		return nil
	}

	if err := rs.JoinRoom(ctx, client, "general"); err != nil {
		return fmt.Errorf("Ошибка подключения к комнате: %v", err)
	}
	return nil
}
