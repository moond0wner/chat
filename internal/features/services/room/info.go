package room_service

import (
	"fmt"
	"strings"
	core_domain "github.com/moond0wner/chat/internal/core/domain"

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
		rs.log.Warn("Error get info about room",
			zap.String("room_name", roomName),
			zap.Error(err),
		)
		return "", fmt.Errorf("error get info about room: %w", err)
	}
	if len(rs.rooms) == 0 {
		rs.log.Warn("GetAllInfo called with empty rooms list", zap.Int("client_id", clientID))
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
		rs.log.Warn("Room exists in the index but not in memory", zap.Int("room_id", roomID), zap.String("room_name", roomName))
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
		rs.log.Warn("Room exists in the index but not in memory", zap.Int("room_id", roomID), zap.String("room_name", roomName))
		return false, fmt.Errorf("комната '%s' существует в индексе, но не в памяти", roomName)
	}

	return len(room.Clients) == 0, nil
}
