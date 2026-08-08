package services

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	core_domain "tcp_srv/internal/core/domain"
	core_logger "tcp_srv/internal/core/logger"

	"go.uber.org/zap"
)

type ClientService struct {
	clients map[string]*core_domain.Client
	nameMap map[string]string
	log     *core_logger.Logger
	mtx     sync.RWMutex
}

func NewClientService(logger *core_logger.Logger) *ClientService {
	return &ClientService{
		clients: make(map[string]*core_domain.Client),
		nameMap: make(map[string]string),
		log:     logger,
	}
}
func (cs *ClientService) RegisterClient(conn net.Conn) *core_domain.Client {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	id := conn.RemoteAddr().String()
	client := &core_domain.Client{
		ID:     id,
		Name:   "Unknown" + fmt.Sprintf("%d", rand.Intn(10000)),
		Conn:   conn,
		RoomID: "general",
	}

	cs.clients[id] = client
	cs.nameMap[id] = client.Name
	cs.log.Debug("Клиент зарегистрирован", zap.String("client_id", id), zap.String("client_name", client.Name))
	return client
}

func (cs *ClientService) UnregisterClient(id string) {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	client, ok := cs.clients[id]
	if !ok {
		return
	}

	delete(cs.nameMap, client.Name)
	delete(cs.clients, id)
	cs.log.Debug("Клиент удален", zap.String("client_id", id))
}

func (cs *ClientService) GetClientByID(id string) *core_domain.Client {
	cs.mtx.RLock()
	defer cs.mtx.RUnlock()
	return cs.clients[id]
}

func (cs *ClientService) FindByName(name string) *core_domain.Client {
	cs.mtx.RLock()
	defer cs.mtx.RUnlock()

	name = strings.TrimPrefix(name, "@")

	id, ok := cs.nameMap[name]
	if !ok {
		return nil
	}
	return cs.clients[id]
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

func (cs *ClientService) ChangeNick(clientID, newName string) error {
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
	if len(newName) < 2 || len(newName) > 20 {
		return errors.New("имя должно быть от 2 до 20 символов")
	}
	if strings.Contains(newName, " ") {
		return errors.New("имя не может содержать пробелы")
	}

	if _, ok := cs.nameMap[newName]; ok {
		return fmt.Errorf("имя '%s' уже занято", newName)
	}

	delete(cs.nameMap, client.Name)

	oldName := client.Name
	client.Name = newName
	cs.nameMap[newName] = clientID

	cs.log.Info("Смена имени",
		zap.String("old_name", oldName),
		zap.String("new_name", newName),
		zap.String("client_id", clientID),
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
