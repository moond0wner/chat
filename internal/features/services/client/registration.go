package client_service

import (
	"context"
	"fmt"
	"math/rand"
	core_domain "github.com/moond0wner/chat/internal/core/domain"
	core_transport "github.com/moond0wner/chat/internal/core/transport"

	"go.uber.org/zap"
)

func (cs *ClientService) RegisterClient(ctx context.Context, conn core_transport.Conn) *core_domain.Client {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	client := &core_domain.Client{
		Name: fmt.Sprintf("Unknown%d", rand.Intn(10000)),
		Conn: conn,
	}

	if err := cs.historyRepository.SaveUser(ctx, client); err != nil {
		cs.log.Warn("Error registration user", zap.String("client_ip", conn.LocalAddr().String()), zap.Error(err))
		return nil
	}

	cs.clients[client.ID] = client
	cs.nameMap[client.Name] = client
	cs.log.Debug("Client is registered", zap.Int("client_id", client.ID), zap.String("client_name", client.Name))
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
	cs.log.Debug("Client deleted", zap.Int("client_id", id))
}
