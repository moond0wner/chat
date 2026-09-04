package client_service

import (
	"strings"
	core_domain "tcp_srv/internal/core/domain"
)

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

func (cs *ClientService) IsNameTaken(name string) bool {
	cs.mtx.RLock()
	defer cs.mtx.RUnlock()
	_, ok := cs.nameMap[name]
	return ok
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
