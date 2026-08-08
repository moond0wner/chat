package services

import (
	"errors"
	"fmt"
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
		Name:   id,
		Conn:   conn,
		RoomID: "general",
	}

	cs.clients[id] = client
	cs.nameMap[id] = id
	cs.log.Debug("Клиент зарегистрирован", zap.String("client_id", id))
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
