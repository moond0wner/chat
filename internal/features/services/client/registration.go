package client_service

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	core_domain "tcp_srv/internal/core/domain"

	"go.uber.org/zap"
)

func (cs *ClientService) RegisterClient(ctx context.Context, conn net.Conn) *core_domain.Client {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	client := &core_domain.Client{
		Name: fmt.Sprintf("Unknown%d", rand.Intn(10000)),
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
