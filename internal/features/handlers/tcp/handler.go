package tcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	core_domain "tcp_srv/internal/core/domain"

	"go.uber.org/zap"
)

func (s *Server) RegisterInServer(ctx context.Context, conn net.Conn) {
	s.handleConnection(ctx, conn, "register", "Подключение к регистрации. Введите /reg <ник> для регистрации")
}

func (s *Server) HandleClient(ctx context.Context, conn net.Conn) {
	s.handleConnection(ctx, conn, "general", "Подключение к комнате: general")
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn, roomName, welcomeMessage string) {
	s.wg.Add(1)
	defer s.wg.Done()

	client := s.clientService.RegisterClient(conn)

	if err := s.roomService.JoinRoom(client, roomName); err != nil {
		s.log.Error("Ошибка подключения к комнате",
			zap.String("room", roomName),
			zap.Error(err))
		conn.Close()
		return
	}

	if _, err := fmt.Fprintf(client.Conn, "%s\n", welcomeMessage); err != nil {
		s.log.Warn("Ошибка отправки приветствия",
			zap.String("client_id", client.ID),
			zap.Error(err))
	}

	s.log.Debug("Новое подключение",
		zap.String("client_id", client.ID),
		zap.String("room", client.RoomID))

	defer func() {

		msg := core_domain.Message{
			RoomID:   client.RoomID,
			SenderID: client.ID,
			Text:     fmt.Sprintf("%s покинул комнату\n", client.Name),
		}
		if err := s.roomService.Broadcast(ctx, msg); err != nil {
			s.log.Error("Ошибка отправки уведомления о выходе",
				zap.String("room", client.RoomID),
				zap.Error(err))
		}

		s.clientService.UnregisterClient(client.ID)

		conn.Close()
		s.log.Debug("Клиент отключился", zap.String("client_id", client.ID), zap.String("client_name", client.Name))
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

		if client.RoomID == "register" && strings.HasPrefix(msg, "/") {
		} else if client.RoomID == "register" {
			fmt.Fprintln(client.Conn, "Сначала зарегистрируйтесь: /reg <ник>")
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

		broadcastMsg := core_domain.Message{
			RoomID:   client.RoomID,
			SenderID: client.ID,
			Text:     formattedMsg,
		}
		if err := s.roomService.Broadcast(ctx, broadcastMsg); err != nil {
			s.log.Error("Ошибка отправки сообщения в комнату",
				zap.String("room", client.RoomID),
				zap.Error(err))
		}
	}
}
