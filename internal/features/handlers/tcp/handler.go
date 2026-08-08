package tcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"go.uber.org/zap"
)

func (s *Server) HandleClient(conn net.Conn, ctx context.Context) {
	s.wg.Add(1)
	defer s.wg.Done()

	client := s.clientService.RegisterClient(conn)

	if err := s.roomService.JoinRoom(client, "general"); err != nil {
		s.log.Error("Ошибка подключения к general", zap.Error(err))
		conn.Close()
		return
	}

	if _, err := fmt.Fprintf(client.Conn, "Подключение к комнате: %s\n", client.RoomID); err != nil {
		s.log.Warn("Ошибка отправки приветствия",
			zap.String("client_id", client.ID),
			zap.Error(err))
	}

	s.log.Debug("Новое подключение",
		zap.String("client_id", client.ID),
		zap.String("room", client.RoomID))

	defer func() {
		s.clientService.UnregisterClient(client.ID)

		if err := s.roomService.Broadcast(
			client.RoomID,
			fmt.Sprintf("%s покинул комнату\n", client.Name),
			client.ID,
		); err != nil {
			s.log.Error("Ошибка отправки уведомления о выходе",
				zap.String("room", client.RoomID),
				zap.Error(err))
		}

		s.clientService.UnregisterClient(client.ID)

		conn.Close()
		s.log.Debug("Клиент отключился", zap.String("client_id", client.ID))
	}()

	reader := bufio.NewReader(conn)
	for {
		select {
		case <-ctx.Done():
			s.log.Debug("Сигнал остановки сервера, завершение обработки клиента",
				zap.String("client_id", client.ID))
			conn.Close()
			return
		default:
		}

		msg, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				s.log.Debug("Клиент закрыл соединение", zap.String("client_id", client.ID))
			} else if errors.Is(err, net.ErrClosed) {
				s.log.Debug("Соединение закрыто сервером", zap.String("client_id", client.ID))
			} else {
				s.log.Warn("Ошибка чтения сообщения",
					zap.String("client_id", client.ID),
					zap.Error(err))
			}
			break
		}

		msg = strings.TrimSpace(msg)
		if msg == "" {
			continue
		}

		if strings.HasPrefix(msg, "/") {
			if err := s.Manager.Execute(msg, client); err != nil {
				fmt.Fprintf(client.Conn, "Error: %s\n", err.Error())
				s.log.Warn("Command error",
					zap.String("client_id", client.ID),
					zap.String("command", msg),
					zap.Error(err))
			}
			continue
		}

		formattedMsg := fmt.Sprintf("[%s] %s: %s\n", client.RoomID, client.Name, msg)
		s.log.Info("Сообщение",
			zap.String("room", client.RoomID),
			zap.String("sender", client.Name),
			zap.String("message", msg))

		if err := s.roomService.Broadcast(client.RoomID, formattedMsg, client.ID); err != nil {
			s.log.Error("Ошибка отправки сообщения в комнату",
				zap.String("room", client.RoomID),
				zap.Error(err))
		}
	}
}
