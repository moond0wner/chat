package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	core_domain "tcp_srv/internal/core/domain"
	core_logger "tcp_srv/internal/core/logger"
	history_postgres_repository "tcp_srv/internal/features/repostitory/postgres"
	"time"

	"go.uber.org/zap"
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

	rs.log.Info("Загружены комнаты из БД", zap.Int("count", len(*rooms)))
	return nil
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
		return fmt.Errorf("room '%d' not found", message.RoomID)
	}
	if !message.IsSystem {
		if err := rs.historyRepository.SaveMessage(ctx, &message); err != nil {
			return fmt.Errorf("db error: %w", err)
		}

	}
	var errs []error
	for id, client := range room.Clients {
		if message.SenderID != 0 && id == message.SenderID {
			continue
		}

		client.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		_, err := fmt.Fprintln(client.Conn, message.Text)
		client.Conn.SetWriteDeadline(time.Time{})

		if err != nil {
			errs = append(errs, fmt.Errorf("client %d: %w", id, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("broadcast errors: %v", errs)
	}
	return nil
}

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
	if err := rs.historyRepository.SaveRoom(ctx, room); err != nil {
		return fmt.Errorf("Error save room: %v", err)
	}
	rs.rooms[room.ID] = room
	rs.roomsByName[room.Name] = room.ID
	return nil
}

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
		return nil, fmt.Errorf("Error get room from db: %v", err)
	}

	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	if _, ok := rs.roomsByName[name]; ok {
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
		return nil, errors.New("Room not found")
	}
}

func (rs *RoomService) getIDRoomByName(ctx context.Context, roomName string) (int, error) {
	rs.mtx.RLock()
	id, ok := rs.roomsByName[roomName]
	rs.mtx.RUnlock()
	if ok {
		return id, nil
	}

	id, err := rs.historyRepository.GetIDRoomByName(ctx, roomName)
	if err != nil {
		return 0, fmt.Errorf("Error get id room by name: %v", err)
	}
	return id, nil
}

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

func (rs *RoomService) JoinRoom(ctx context.Context, client *core_domain.Client, newRoomName string) error {
	rs.mtx.RLock()
	currentRoom := rs.rooms[client.RoomID]
	rs.mtx.RUnlock()

	if currentRoom != nil && currentRoom.Name == newRoomName {
		fmt.Fprintln(client.Conn, "Вы уже находитесь в этом канале")
		return nil
	}

	oldRoom, oldRoomClients, needDelete := rs.removeClientFromRoom(client)
	newRoom, newRoomClients, err := rs.addClientToRoom(ctx, client, newRoomName)
	if err != nil {
		if oldRoom != nil {
			rs.addClientToRoom(ctx, client, oldRoom.Name)
		}
		return fmt.Errorf("join room failed: %w", err)
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

func (rs *RoomService) LeaveRoom(ctx context.Context, client *core_domain.Client) error {
	if client.RoomID == rs.GeneralRoomID {
		fmt.Fprintln(client.Conn, "Вы уже в general")
		return nil
	}

	if err := rs.JoinRoom(ctx, client, "general"); err != nil {
		return fmt.Errorf("Ошибка подключения к комнате: %v", err)
	}
	return nil
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
	for i := 0; i < len(clients); i++ {
		name := clients[i].Name
		if clients[i].ID == clientID {
			name = fmt.Sprintf("%s (вы)", name)
		}
		text += fmt.Sprintf("%d. %s\n", i+1, name)
	}
	return text, nil
}
