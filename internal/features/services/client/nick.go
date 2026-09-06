package client_service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	core_domain "github.com/moond0wner/chat/internal/core/domain"

	"go.uber.org/zap"
)

func (cs *ClientService) ChangeNick(ctx context.Context, client *core_domain.Client, newName string) error {
	if err := cs.validationNick(ctx, newName); err != nil {
		return fmt.Errorf("Ошибка проверки ника: %v", err)
	}

	oldName := client.Name
	client.Name = newName
	delete(cs.nameMap, oldName)
	cs.nameMap[newName] = client

	if err := cs.historyRepository.UpdateUser(ctx, client); err != nil {
		cs.log.Warn("Error update user in DB", zap.Int("client_id", client.ID), zap.Error(err))
		return errors.New("Не удалось обновить данные пользователя")
	}

	cs.log.Info("Changing user",
		zap.String("old_name", oldName),
		zap.String("new_name", newName),
		zap.Int("client_id", client.ID),
	)

	return nil
}

func (cs *ClientService) validationNick(ctx context.Context, newName string) error {
	const (
		minNickLength = 2
		maxNickLength = 20
	)

	cs.mtx.Lock()
	defer cs.mtx.Unlock()
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
		cs.log.Warn("Error check nick in DB", zap.String("checking_name", newName), zap.Error(err))
		return fmt.Errorf("не удалось проверить имя")
	}
	if taken {
		return fmt.Errorf("имя '%s' уже занято в системе", newName)
	}
	return nil
}
