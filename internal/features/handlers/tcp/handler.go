package tcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	core_domain "github.com/moond0wner/chat/internal/core/domain"
	core_transport "github.com/moond0wner/chat/internal/core/transport"
	"github.com/moond0wner/chat/internal/features/handlers/common"
	"go.uber.org/zap"
)

func (s *Server) RegisterInServer(ctx context.Context, conn core_transport.Conn) {
	s.handleConnection(ctx, conn, "register", "Подключение к регистрации. Введите /reg <ник> для регистрации")
}

func (s *Server) HandleClient(ctx context.Context, conn core_transport.Conn) {
	s.handleConnection(ctx, conn, "general", "Подключение к комнате: general")
}

func (s *Server) handleConnection(ctx context.Context, conn core_transport.Conn, roomName, welcomeMessage string) {
	s.wg.Add(1)
	defer s.wg.Done()

	client := s.preRegistration(ctx, conn, roomName, welcomeMessage)
	if client == nil {
		return
	}

	reader := bufio.NewReader(client.Conn)
	defer func() {

		msg := core_domain.Message{
			RoomID:   client.RoomID,
			SenderID: client.ID,
			Text:     fmt.Sprintf("%s покинул комнату\n", client.Name),
			IsSystem: true,
		}
		if err := s.roomService.Broadcast(ctx, msg); err != nil {
			s.log.Error("Error sending exit notification",
				zap.Int("room_id", client.RoomID),
				zap.Error(err))
		}

		s.clientService.UnregisterClient(client.ID)

		conn.Close()
		s.log.Debug("Client disconnected", zap.Int("client_id", client.ID), zap.String("client_name", client.Name))
	}()
	for {
		select {
		case <-ctx.Done():
			s.log.Debug("Server stop signal, client processing termination",
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

func (s *Server) preRegistration(ctx context.Context, conn core_transport.Conn, roomName, welcomeMessage string) *core_domain.Client {
	return common.RegisterAndJoin(ctx, conn, roomName, welcomeMessage, s.clientService, s.roomService, s.log)
}

func (s *Server) handleMessage(ctx context.Context, client *core_domain.Client, reader *bufio.Reader) bool {
	msg, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			s.log.Debug("Client closed connection", zap.Int("client_id", client.ID))
		} else if errors.Is(err, net.ErrClosed) {
			s.log.Debug("Connection closed by server", zap.Int("client_id", client.ID))
		} else {
			s.log.Warn("Error read message",
				zap.Int("client_id", client.ID),
				zap.Error(err))
		}
		return false
	}

	common.ProcessMessage(ctx, client, strings.TrimSpace(msg), s.Manager, s.roomService, s.log)
	return true
}
