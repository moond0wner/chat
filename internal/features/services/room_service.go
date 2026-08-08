package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	core_domain "tcp_srv/internal/core/domain"
	core_logger "tcp_srv/internal/core/logger"
	"time"

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

func (rs *RoomService) Broadcast(ctx context.Context, message core_domain.Message) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	room, ok := rs.rooms[message.RoomID]
	if !ok {
		return fmt.Errorf("room '%s' not found", message.RoomID)
	}

	var errs []error
	for id, client := range room.Clients {
		if message.SenderID != "" && id == message.SenderID {
			continue
		}

		client.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		_, err := fmt.Fprintln(client.Conn, message.Text)
		client.Conn.SetWriteDeadline(time.Time{})

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
	if currentRoom != nil {
		if _, ok := currentRoom.Clients[client.ID]; !ok {
			currentRoom = nil
		}
	}
	if currentRoom != nil && currentRoom.Name == roomName && currentRoom.Name != "general" && currentRoom.Name != "register" {
		rs.mtx.Unlock()
		fmt.Fprintln(client.Conn, "Вы уже находитесь в этом канале")
		return nil
	}

	var (
		oldRoomName       string
		needDeleteOldRoom bool
		oldRoomClients    []*core_domain.Client
	)

	if currentRoom != nil {
		oldRoomName = currentRoom.Name
		for id, c := range currentRoom.Clients {
			if id != client.ID {
				oldRoomClients = append(oldRoomClients, c)
			}
		}

		delete(currentRoom.Clients, client.ID)

		if len(currentRoom.Clients) == 0 && currentRoom.Name != "general" && currentRoom.Name != "register" {
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

	var newRoomClients []*core_domain.Client
	for id, c := range room.Clients {
		if id != client.ID {
			newRoomClients = append(newRoomClients, c)
		}
	}

	if needDeleteOldRoom {
		delete(rs.rooms, oldRoomName)
		rs.log.Warn("Удалена пустая комната", zap.String("room", oldRoomName))
	}

	rs.mtx.Unlock()

	if oldRoomName != "" && len(oldRoomClients) > 0 {
		msg := core_domain.Message{
			RoomID:     oldRoomName,
			SenderID:   client.ID,
			SenderName: client.Name,
			Text:       fmt.Sprintf("%s покинул комнату\n", client.Name),
		}
		for _, c := range oldRoomClients {
			fmt.Fprintln(c.Conn, msg.Text)
		}
		fmt.Fprintf(client.Conn, "Вы покинули комнату: %s\n", oldRoomName)

		rs.log.Info("Клиент покинул комнату",
			zap.String("client", client.Name),
			zap.String("room", oldRoomName))
	}

	if len(newRoomClients) > 0 {
		msg := core_domain.Message{
			RoomID:   roomName,
			SenderID: client.ID,
			Text:     fmt.Sprintf("%s подключился к комнате\n", client.Name),
		}
		for _, c := range newRoomClients {
			fmt.Fprintln(c.Conn, msg.Text)
		}
	}

	fmt.Fprintf(client.Conn, "Подключен к комнате: %s\n", roomName)

	rs.log.Info("Клиент подключился к комнате",
		zap.String("client", client.Name),
		zap.String("room", roomName))

	return nil
}

func (rs *RoomService) GetClients(roomName string) ([]*core_domain.Client, error) {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	room, ok := rs.rooms[roomName]
	if !ok {
		return nil, fmt.Errorf("'%s' channel not found", roomName)
	}

	clients := make([]*core_domain.Client, 0, len(room.Clients))
	for _, c := range room.Clients {
		clients = append(clients, c)
	}
	return clients, nil
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

func (rs *RoomService) GetAllInfo(clientID string) (string, error) {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()

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

func (rs *RoomService) GetInfoAboutRoom(clientID string, roomName string) (string, error) {
	clients, err := rs.GetClients(roomName)
	if err != nil {
		return "", fmt.Errorf("Error get info about room: %s: %v", roomName, err)
	}

	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	text := fmt.Sprintf("Информация о комнате '%s':\n", roomName)
	for i := 0; i < len(clients); i++ {
		name := clients[i].Name
		if clients[i].ID == clientID {
			name = fmt.Sprintf("%s (вы)", name)
		}
		text += fmt.Sprintf("%d. %s\n", i+1, name)
	}
	return text, nil
}
