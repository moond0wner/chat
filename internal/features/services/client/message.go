package client_service

import (
	"errors"
	"fmt"
	core_domain "github.com/moond0wner/chat/internal/core/domain"

	"go.uber.org/zap"
)

func (cs *ClientService) SendMessageToUser(client *core_domain.Client, text string) error {
	if _, err := fmt.Fprintln(client.Conn, text); err != nil {
		cs.log.Warn("Error sending message",
			zap.Int("client_id", client.ID),
			zap.Error(err))
		return errors.New("Не удалось отправить пользователю")
	}
	return nil
}

func (cs *ClientService) SendPrivateMessage(message core_domain.PrivateMessage, client *core_domain.Client) error {
	recipent := cs.FindByName(message.RecipientName)
	if recipent == nil {
		return fmt.Errorf("Получатель с ником '%s' не найден", message.RecipientName)
	}
	sender := cs.FindByName(message.SenderName)
	if sender == nil {
		return fmt.Errorf("Отправитель с ником '%s' не найден", message.SenderName)
	}

	cs.mtx.Lock()
	defer cs.mtx.Unlock()
	if recipent.ID == sender.ID {
		return errors.New("Нельзя отправить сообщение самому себе")
	}

	text := fmt.Sprintf("Личное сообщение от '%s': %s\n", sender.Name, message.Text)
	if _, err := fmt.Fprintln(recipent.Conn, text); err != nil {
		cs.log.Warn(
			"Error send private message",
			zap.Int("recipent_id", recipent.ID),
			zap.Int("sender_id", sender.ID),
			zap.Error(err),
		)
		return errors.New("Не удалось отправить приватное сообщение")
	}
	if _, err := fmt.Fprintln(sender.Conn, "Личное сообщение отправлено"); err != nil {
		cs.log.Warn(
			"Error send confirmation",
			zap.Int("recipent_id", recipent.ID),
			zap.Int("sender_id", sender.ID),
			zap.Error(err),
		)
		return errors.New("Возникла непредвиденная ошибка")
	}

	return nil
}
