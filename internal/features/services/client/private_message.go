package client_service

import (
	"errors"
	"fmt"
	core_domain "tcp_srv/internal/core/domain"
)

func (cs *ClientService) SendPrivateMessage(message core_domain.PrivateMessage, client *core_domain.Client) error {
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
		return errors.New("Error: send to self")
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
