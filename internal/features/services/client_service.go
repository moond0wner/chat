package services

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	core_domain "tcp_srv/internal/core/domain"
	core_logger "tcp_srv/internal/core/logger"
	history_postgres_repository "tcp_srv/internal/features/repostitory/postgres"

	"go.uber.org/zap"
)

type ClientService struct {
	clients           map[int]*core_domain.Client
	nameMap           map[string]*core_domain.Client
	historyRepository *history_postgres_repository.HistoryRepository
	log               *core_logger.Logger
	mtx               sync.RWMutex
}

func NewClientService(logger *core_logger.Logger, hs *history_postgres_repository.HistoryRepository) *ClientService {
	return &ClientService{
		clients:           make(map[int]*core_domain.Client),
		nameMap:           make(map[string]*core_domain.Client),
		historyRepository: hs,
		log:               logger,
	}
}

func (cs *ClientService) LoadAllUsers(ctx context.Context) error {
	users, err := cs.historyRepository.GetAllUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load users from db: %w", err)
	}

	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	for _, user := range *users {
		client := &core_domain.Client{
			ID:     user.ID,
			Name:   user.Name,
			Conn:   nil,
			RoomID: user.RoomID,
		}
		cs.clients[user.ID] = client
		cs.nameMap[user.Name] = client
	}

	cs.log.Info("Загружены пользователи из БД", zap.Int("count", len(*users)))
	return nil
}

func (cs *ClientService) RegisterClient(ctx context.Context, conn net.Conn) *core_domain.Client {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	client := &core_domain.Client{
		Name: "Unknown" + fmt.Sprintf("%d", rand.Intn(10000)),
		Conn: conn,
	}

	if err := cs.historyRepository.SaveUser(ctx, client); err != nil {
		cs.log.Warn("Ошибка регистрации пользователя", zap.String("client_ip", conn.LocalAddr().String()), zap.Error(err))
		return nil
	}

	cs.clients[client.ID] = client
	cs.nameMap[client.Name] = client
	cs.log.Debug("Клиент зарегистрирован", zap.Int("client_id", client.ID), zap.String("client_name", client.Name))
	return client
}

func (cs *ClientService) UnregisterClient(id int) {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	client, ok := cs.clients[id]
	if !ok {
		return
	}

	delete(cs.nameMap, client.Name)
	delete(cs.clients, client.ID)
	cs.log.Debug("Клиент удален", zap.Int("client_id", id))
}

func (cs *ClientService) GetClientByID(id int) *core_domain.Client {
	cs.mtx.RLock()
	defer cs.mtx.RUnlock()
	return cs.clients[id]
}

func (cs *ClientService) FindByName(name string) *core_domain.Client {
	cs.mtx.RLock()
	defer cs.mtx.RUnlock()

	name = strings.TrimPrefix(name, "@")

	client, ok := cs.nameMap[name]
	if !ok {
		return nil
	}
	return client
}

func (cs *ClientService) SendPrivateMessage(message core_domain.PrivateMessage) error {

	recipent := cs.FindByName(message.RecipientName)
	if recipent == nil {
		return fmt.Errorf("User with name '%s' not found", message.RecipientName)
	}
	sender := cs.FindByName(message.SenderName)
	if sender == nil {
		return fmt.Errorf("Sender with name '%s' not found", message.SenderName)
	}

	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	if recipent.ID == sender.ID {
		fmt.Fprintln(sender.Conn, "Нельзя отправить сообщение самому себе")
		return fmt.Errorf("Error: send to self")
	}

	text := fmt.Sprintf("Личное сообщение от '%s': %s\n", sender.Name, message.Text)
	if _, err := fmt.Fprintln(recipent.Conn, text); err != nil {
		return fmt.Errorf("Error send private message: %v", err)
	}
	if _, err := fmt.Fprintln(sender.Conn, "Личное сообщение отправлено"); err != nil {
		return fmt.Errorf("Error send confirmation: %v", err)
	}

	return nil
}

func (cs *ClientService) IsNameTaken(name string) bool {
	cs.mtx.RLock()
	defer cs.mtx.RUnlock()
	_, ok := cs.nameMap[name]
	return ok
}

func (cs *ClientService) ChangeNick(ctx context.Context, clientID int, newName string) error {
	const (
		minNickLength = 2
		maxNickLength = 20
	)

	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	client, ok := cs.clients[clientID]
	if !ok {
		return errors.New("клиент не найден")
	}

	newName = strings.TrimSpace(newName)
	if newName == "" {
		return errors.New("имя не может быть пустым")
	}
	if len(newName) < minNickLength || len(newName) > maxNickLength {
		return fmt.Errorf("имя должно быть от %d до %d символов", minNickLength, maxNickLength)
	}
	if strings.Contains(newName, " ") {
		return errors.New("имя не может содержать пробелы")
	}

	if _, ok := cs.nameMap[newName]; ok {
		return fmt.Errorf("имя '%s' уже занято", newName)
	}

	taken, err := cs.historyRepository.IsNicknameTaken(ctx, newName)
	if err != nil {
		cs.log.Warn("Ошибка проверки ника в БД", zap.Error(err))
		return fmt.Errorf("ошибка проверки ника: %w", err)
	}
	if taken {
		return fmt.Errorf("имя '%s' уже занято в системе", newName)
	}

	oldName := client.Name
	client.Name = newName
	delete(cs.nameMap, oldName)
	cs.nameMap[newName] = client

	if err := cs.historyRepository.UpdateUser(ctx, client); err != nil {
		cs.log.Warn("Ошибка сохранения пользователя в db")
		return fmt.Errorf("Error save db: %v", err)
	}

	cs.log.Info("Смена имени",
		zap.String("old_name", oldName),
		zap.String("new_name", newName),
		zap.Int("client_id", client.ID),
	)

	return nil
}
func (cs *ClientService) GetAllClients() []*core_domain.Client {
	cs.mtx.RLock()
	defer cs.mtx.RUnlock()

	clients := make([]*core_domain.Client, 0, len(cs.clients))
	for _, c := range cs.clients {
		clients = append(clients, c)
	}
	return clients
}

func (cs *ClientService) GetAllNames() []string {
	cs.mtx.RLock()
	defer cs.mtx.RUnlock()

	names := make([]string, 0, len(cs.nameMap))
	for name := range cs.nameMap {
		names = append(names, name)
	}
	return names
}

func (cs *ClientService) Count() int {
	cs.mtx.RLock()
	defer cs.mtx.RUnlock()
	return len(cs.clients)
}
