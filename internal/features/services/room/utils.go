package room_service

import (
	"context"
	"fmt"
	core_domain "tcp_srv/internal/core/domain"
)

func (rs *RoomService) removeClientFromRoom(client *core_domain.Client) (*core_domain.Room, []*core_domain.Client, bool) {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	room := rs.rooms[client.RoomID]
	if room == nil {
		return nil, nil, false
	}

	var otherClients []*core_domain.Client
	for id, c := range room.Clients {
		if id != client.ID {
			otherClients = append(otherClients, c)
		}
	}

	delete(room.Clients, client.ID)

	shouldDelete := len(room.Clients) == 0 && room.Name != "general" && room.Name != "register"

	return room, otherClients, shouldDelete
}

func (rs *RoomService) addClientToRoom(ctx context.Context, client *core_domain.Client, roomName string) (*core_domain.Room, []*core_domain.Client, error) {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	roomID, ok := rs.roomsByName[roomName]
	var room *core_domain.Room
	if !ok {
		room = core_domain.NewRoom(roomName)
		if err := rs.historyRepository.SaveRoom(ctx, room); err != nil {
			return nil, nil, fmt.Errorf("save room: %w", err)
		}
		rs.rooms[room.ID] = room
		rs.roomsByName[room.Name] = room.ID
	} else {
		room = rs.rooms[roomID]
	}

	room.Clients[client.ID] = client
	client.RoomID = room.ID

	var otherClients []*core_domain.Client
	for id, c := range room.Clients {
		if id != client.ID {
			otherClients = append(otherClients, c)
		}
	}

	return room, otherClients, nil
}

func (rs *RoomService) deleteRoom(roomID int) {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	delete(rs.rooms, roomID)
}
