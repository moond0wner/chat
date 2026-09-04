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
	"time"

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

	client := s.preRegistration(ctx, conn, roomName, welcomeMessage)

	reader := bufio.NewReader(client.Conn)
	defer func() {

		msg := core_domain.Message{
			RoomID:   client.RoomID,
			SenderID: client.ID,
			Text:     fmt.Sprintf("%s покинул комнату\n", client.Name),
			IsSystem: true,
		}
		if err := s.roomService.Broadcast(ctx, msg); err != nil {
			s.log.Error("Ошибка отправки уведомления о выходе",
				zap.Int("room_id", client.RoomID),
				zap.Error(err))
		}

		s.clientService.UnregisterClient(client.ID)

		conn.Close()
		s.log.Debug("Клиент отключился", zap.Int("client_id", client.ID), zap.String("client_name", client.Name))
	}()
	for {
		select {
		case <-ctx.Done():
			s.log.Debug("Сигнал остановки сервера, завершение обработки клиента",
				zap.Int("client_id", client.ID))
			conn.Close()
			return
		default:
		}
		if !s.handleMessage(ctx, client, reader) {
			return
		}
	}
}

func (s *Server) preRegistration(ctx context.Context, conn net.Conn, roomName, welcomeMessage string) *core_domain.Client {
	client := s.clientService.RegisterClient(ctx, conn)
	if client == nil {
		conn.Close()
		return nil
	}

	if err := s.roomService.JoinRoom(ctx, client, roomName); err != nil {
		s.log.Error("Ошибка подключения к комнате",
			zap.String("room", roomName),
			zap.Error(err))
		conn.Close()
		return nil
	}

	if _, err := fmt.Fprintf(client.Conn, "%s\n", welcomeMessage); err != nil {
		s.log.Warn("Ошибка отправки приветствия",
			zap.Int("client_id", client.ID),
			zap.Error(err))
	}

	s.log.Debug("Новое подключение",
		zap.Int("client_id", client.ID),
		zap.Int("room_id", client.RoomID))

	return client
}

func (s *Server) handleMessage(ctx context.Context, client *core_domain.Client, reader *bufio.Reader) bool {
	msg, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			s.log.Debug("Клиент закрыл соединение", zap.Int("client_id", client.ID))
		} else if errors.Is(err, net.ErrClosed) {
			s.log.Debug("Соединение закрыто сервером", zap.Int("client_id", client.ID))
		} else {
			s.log.Warn("Ошибка чтения сообщения",
				zap.Int("client_id", client.ID),
				zap.Error(err))
		}
		return false
	}

	msg = strings.TrimSpace(msg)
	if msg == "" {
		return true
	}

	if client.RoomID == s.roomService.RegisterRoomID {
		if !strings.HasPrefix(msg, "/") {
			fmt.Fprintln(client.Conn, "Сначала зарегистрируйтесь: /reg <ник>")
			return true
		}
	}

	if strings.HasPrefix(msg, "/") {
		if err = s.Manager.Execute(msg, client); err != nil {
			fmt.Fprintf(client.Conn, "Error: %s\n", err.Error())
			s.log.Warn("Command error",
				zap.Int("client_id", client.ID),
				zap.String("command", msg),
				zap.Error(err))
			return true
		}
	} else {
		formattedMsg := fmt.Sprintf("[%d] %s: %s\n", client.RoomID, client.Name, msg)
		s.log.Info("Сообщение",
			zap.Int("room_id", client.RoomID),
			zap.String("sender", client.Name),
			zap.String("message", msg))

		broadcastMsg := core_domain.Message{
			RoomID:     client.RoomID,
			SenderName: client.Name,
			SenderID:   client.ID,
			Text:       formattedMsg,
			SendAt:     time.Now(),
			IsSystem:   false,
		}
		if err = s.roomService.Broadcast(ctx, broadcastMsg); err != nil {
			s.log.Error("Ошибка отправки сообщения в комнату",
				zap.Int("room_id", client.RoomID),
				zap.Error(err))
			return true
		}
	}

	return true
}
