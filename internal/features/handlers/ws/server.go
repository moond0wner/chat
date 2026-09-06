package ws

import (
	"context"
	"errors"
	"net/http"
	"time"

	core_command "github.com/moond0wner/chat/internal/core/command"
	core_logger "github.com/moond0wner/chat/internal/core/logger"
	client_service "github.com/moond0wner/chat/internal/features/services/client"
	room_service "github.com/moond0wner/chat/internal/features/services/room"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Server struct {
	upgrader      websocket.Upgrader
	httpServer    *http.Server
	manager       *core_command.Manager
	roomService   *room_service.RoomService
	clientService *client_service.ClientService
	log           *core_logger.Logger
}

func NewServer(
	log *core_logger.Logger,
	rs *room_service.RoomService,
	cs *client_service.ClientService,
	mng *core_command.Manager,
) *Server {
	return &Server{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},

		roomService:   rs,
		clientService: cs,
		log:           log,
		manager:       mng,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleConnection)

	s.httpServer = &http.Server{
		Addr:         ":8081",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	s.log.Info("WebSocket server started", zap.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Debug("Server shutdown")
	if err := s.httpServer.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.log.Debug("Error closed server", zap.Error(err))
		return err
	}
	s.log.Debug("Server is stopped")
	return nil
}
