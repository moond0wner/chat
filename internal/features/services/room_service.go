package services

import (
	"errors"
	"fmt"
	"sync"
	core_domain "tcp_srv/internal/core/domain"
	core_logger "tcp_srv/internal/core/logger"

	"go.uber.org/zap"
)

type RoomService struct {
	rooms map[string]*core_domain.Room
	log   *core_logger.Logger
	mtx   sync.Mutex
}

func NewRoomService(log *core_logger.Logger) *RoomService {
	return &RoomService{
		rooms: make(map[string]*core_domain.Room),
		log:   log,
	}
}

func (rs *RoomService) Broadcast(roomName string, message string, senderID string) error {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	room, ok := rs.rooms[roomName]
	if !ok {
		return fmt.Errorf("room '%s' not found", roomName)
	}

	var errs []error
	for id, client := range room.Clients {
		if senderID != "" && id == senderID {
			continue
		}
		_, err := fmt.Fprintln(client.Conn, message)
		if err != nil {
			errs = append(errs, fmt.Errorf("client %s: %w", id, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("broadcast errors: %v", errs)
	}
	return nil
}

func (s *RoomService) CreateRoom(name string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if _, ok := s.rooms[name]; ok {
		return errors.New("Room is already exists")
	}

	s.rooms[name] = core_domain.NewRoom(name)
	s.log.Debug("create room", zap.String("room", name))

	return nil
}

func (rs *RoomService) GetRoom(name string) *core_domain.Room {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	return rs.rooms[name]
}

func (rs *RoomService) JoinRoom(client *core_domain.Client, roomName string) error {
	rs.mtx.Lock()

	currentRoom := rs.rooms[client.RoomID]

	if currentRoom != nil && currentRoom.Name == roomName && currentRoom.Name != "general" {
		rs.mtx.Unlock()
		fmt.Fprintln(client.Conn, "Вы уже находитесь в этом канале")
		return nil
	}

	var oldRoomName string
	var needDeleteOldRoom bool

	if currentRoom != nil {
		oldRoomName = currentRoom.Name

		delete(currentRoom.Clients, client.ID)

		if len(currentRoom.Clients) == 0 && currentRoom.Name != "general" {
			needDeleteOldRoom = true
		}
	}

	room := rs.rooms[roomName]
	if room == nil {
		room = core_domain.NewRoom(roomName)
		rs.rooms[roomName] = room
	}
	room.Clients[client.ID] = client
	client.RoomID = roomName

	if needDeleteOldRoom {
		delete(rs.rooms, oldRoomName)
		rs.log.Warn("Удалена пустая комната", zap.String("room", oldRoomName))
	}

	rs.mtx.Unlock()

	if oldRoomName != "" {
		rs.Broadcast(oldRoomName, fmt.Sprintf("%s left the room\n", client.Name), client.ID)
		fmt.Fprintf(client.Conn, "Вы покинули комнату: %s\n", oldRoomName)
	}

	rs.Broadcast(roomName, fmt.Sprintf("%s подключился к комнате\n", client.Name), client.ID)
	fmt.Fprintf(client.Conn, "Подключен к комнате: %s\n", roomName)

	rs.log.Info("Клиент подключился к комнате",
		zap.String("client", client.Name),
		zap.String("room", roomName))

	return nil
}

func (rs *RoomService) GetClients(roomName string) ([]string, error) {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	room, ok := rs.rooms[roomName]
	if !ok {
		return nil, fmt.Errorf("'%s' channel not found", roomName)
	}

	names := make([]string, 0, len(room.Clients))
	for _, c := range room.Clients {
		names = append(names, c.Name)
	}
	return names, nil
}

func (s *RoomService) LeaveRoom(client *core_domain.Client) error {
	if client.RoomID == "general" {
		fmt.Fprintln(client.Conn, "Вы уже в general")
		return nil
	}

	if err := s.JoinRoom(client, "general"); err != nil {
		return fmt.Errorf("Ошибка подключения к комнате: %v", err)
	}
	return nil
}

func (rs *RoomService) IsEmpty(roomName string) (bool, error) {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	room, ok := rs.rooms[roomName]
	if !ok {
		return false, fmt.Errorf("'%s' channel not found", roomName)
	}

	return len(room.Clients) == 0, nil
}
