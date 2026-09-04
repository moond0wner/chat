package room_service

import (
	"fmt"
	"strings"
	core_domain "tcp_srv/internal/core/domain"

	"go.uber.org/zap"
)

func (rs *RoomService) GetAllInfo(clientID int) (string, error) {
	rs.mtx.RLock()
	defer rs.mtx.RUnlock()

	text := "Полная информация о сервере:\n"
	for _, room := range rs.rooms {
		names := make([]string, 0, len(room.Clients))
		for _, client := range room.Clients {
			name := client.Name
			if client.ID == clientID {
				name = fmt.Sprintf("%s (вы)", client.Name)
			}
			names = append(names, name)
		}
		users := strings.Join(names, ", ")
		if users == "" {
			users = "(пусто)"
		}
		text += fmt.Sprintf("%s (%d): %s\n", room.Name, len(names), users)
	}
	return text, nil
}

func (rs *RoomService) GetInfoAboutRoom(clientID int, roomName string) (string, error) {
	clients, err := rs.GetClients(roomName)
	if err != nil {
		return "", fmt.Errorf("Error get info about room: %s: %v", roomName, err)
	}

	rs.mtx.RLock()
	defer rs.mtx.RUnlock()
	text := fmt.Sprintf("Информация о комнате '%s':\n", roomName)
	for i, client := range clients {
		name := client.Name
		if client.ID == clientID {
			name = fmt.Sprintf("%s (вы)", name)
		}
		text += fmt.Sprintf("%d. %s\n", i+1, name)
	}
	return text, nil
}

func (rs *RoomService) GetClients(roomName string) ([]*core_domain.Client, error) {
	rs.mtx.RLock()
	defer rs.mtx.RUnlock()

	roomID, ok := rs.roomsByName[roomName]
	if !ok {
		return nil, fmt.Errorf("комната '%s' не найдена", roomName)
	}

	room, ok := rs.rooms[roomID]
	if !ok {
		rs.log.Warn("Комната существует в индексе, но не в памяти", zap.Int("room_id", roomID), zap.String("room_name", roomName))
		return nil, fmt.Errorf("комната '%s' существует в индексе, но не в памяти", roomName)
	}

	clients := make([]*core_domain.Client, 0, len(room.Clients))
	for _, client := range room.Clients {
		clients = append(clients, client)
	}

	return clients, nil
}

func (rs *RoomService) IsEmpty(roomName string) (bool, error) {
	rs.mtx.RLock()
	defer rs.mtx.RUnlock()

	roomID, ok := rs.roomsByName[roomName]
	if !ok {
		return false, fmt.Errorf("комната '%s' не найдена", roomName)
	}

	room, ok := rs.rooms[roomID]
	if !ok {
		rs.log.Warn("Комната существует в индексе, но не в памяти", zap.Int("room_id", roomID), zap.String("room_name", roomName))
		return false, fmt.Errorf("комната '%s' существует в индексе, но не в памяти", roomName)
	}

	return len(room.Clients) == 0, nil
}
