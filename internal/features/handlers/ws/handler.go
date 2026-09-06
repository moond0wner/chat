package ws

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	core_domain "github.com/moond0wner/chat/internal/core/domain"
	"github.com/moond0wner/chat/internal/features/handlers/common"
	"go.uber.org/zap"
)

func (s *Server) handleConnection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	wsConn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}
	defer wsConn.Close()

	adapter := NewWebSocketConnAdapter(wsConn)

	client := common.RegisterAndJoin(
		ctx,
		adapter,
		"register",
		"Подключение к регистрации. Введите /reg <ник> для регистрации",
		s.clientService,
		s.roomService,
		s.log,
	)

	s.handleMessages(ctx, client, wsConn)
}

func (s *Server) handleMessages(ctx context.Context, client *core_domain.Client, conn *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				s.log.Debug("Client disconnected", zap.Int("client_id", client.ID))
			} else {
				s.log.Error("WebSocket read error", zap.Int("client_id", client.ID), zap.Error(err))
			}
			return
		}
		go common.ProcessMessage(ctx, client, string(msg), s.manager, s.roomService, s.log)
		s.log.Debug("Received message", zap.Int("client_id", client.ID), zap.String("msg", string(msg)))
	}
}
