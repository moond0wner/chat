package client_service

import (
	"context"
	"fmt"
	core_domain "github.com/moond0wner/chat/internal/core/domain"

	"go.uber.org/zap"
)

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

	cs.log.Info("Users loaded from DB", zap.Int("count", len(*users)))
	return nil
}
